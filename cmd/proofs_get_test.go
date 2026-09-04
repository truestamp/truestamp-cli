// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// Minimal bundles in the published layout: enough shape to pass the
// download path's sniff. Nothing here is crypto-valid; verify is not
// called. Unit coverage of verify lives in internal/verify.
const (
	testBlockMap = `{"id":"019db702-b08c-73dc-a7cd-2c5e011f1dad","previous_block_hash":"11","merkle_root":"22","metadata":{},"signing_key_id":"4ceefa4a"}`
	testCommits  = `[{"chain":"stellar","network":"testnet","epoch_proof":"AA","epoch_merkle_root":"aa","transaction_hash":"bb","ledger":1}]`

	testBlockProofJSON = `{"version":1,"type":"block",
	  "public_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	  "signature":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
	  "generated_at":"2026-04-22T21:05:00Z",
	  "block":` + testBlockMap + `,"commitments":` + testCommits + `}`
	testBeaconProofJSON = `{"version":1,"type":"beacon",
	  "public_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	  "signature":"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB==",
	  "generated_at":"2026-04-22T21:05:00Z",
	  "block":` + testBlockMap + `,"commitments":` + testCommits + `}`
	testItemProofJSON = `{"version":1,"type":"item",
	  "public_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	  "signature":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
	  "generated_at":"2026-04-22T21:05:00Z",
	  "subject":{"id":"01HJHB01T8FYZ7YTR9P5N62K5B","claims":{"name":"x"},"metadata":{"witnesses":{"block":"33"}},"signing_key_id":"4ceefa4a"},
	  "inclusion_proof":"AA",
	  "block":` + testBlockMap + `,"commitments":` + testCommits + `}`
	testEntropyStellarProofJSON = `{"version":1,"type":"entropy_stellar",
	  "public_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	  "signature":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
	  "generated_at":"2026-04-22T21:05:00Z",
	  "subject":{"id":"019db702-b08c-73dc-a7cd-2c5e011f1dad","entropy":{"hash":"x"},"metadata":{},"signing_key_id":"4ceefa4a"},
	  "inclusion_proof":"AA",
	  "block":` + testBlockMap + `,"commitments":` + testCommits + `}`
)

// startProofServer spins up an httptest server that records the last
// request body so tests can assert what the client actually sent, and
// responds with the provided proof envelope.
func startProofServer(t *testing.T, responseBody string) (string, *string, func()) {
	t.Helper()
	var lastBody string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/json/proof/generate", func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		b, _ := io.ReadAll(r.Body)
		lastBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"result":` + responseBody + `}`))
	})
	srv := httptest.NewServer(mux)
	return srv.URL, &lastBody, srv.Close
}

func withTempCWD(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}

// postedData decodes the captured request's `data` object.
func postedData(t *testing.T, body string) map[string]any {
	t.Helper()
	var parsed struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("captured body not valid JSON: %v\nbody=%q", err, body)
	}
	return parsed.Data
}

func assertWireType(t *testing.T, body, want string) {
	t.Helper()
	if got := postedData(t, body)["type"]; got != want {
		t.Errorf("wire type: want %q, got %v (full body: %s)", want, got, body)
	}
}

func TestCLI_ProofsGet_SmartDefaultULIDItem(t *testing.T) {
	url, lastBody, stop := startProofServer(t, testItemProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--to-file", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%q", exit, stderr)
	}
	wantFile := filepath.Join(dir, "truestamp-item-01HJHB01T8FYZ7YTR9P5N62K5B.json")
	if _, err := os.Stat(wantFile); err != nil {
		t.Errorf("expected file %s: %v", wantFile, err)
	}
	assertWireType(t, *lastBody, "item")
	if _, has := postedData(t, *lastBody)["witnesses"]; has {
		t.Errorf("the default (all witnesses) must omit the argument, got %s", *lastBody)
	}
}

func TestCLI_ProofsGet_Witnesses(t *testing.T) {
	cases := []struct {
		flag     string
		wantList any
		suffix   string
	}{
		{"none", []any{}, "-compact"},
		{"block,entropy_nist", []any{"block", "entropy_nist"}, "-partial"},
		{"all", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.flag, func(t *testing.T) {
			url, lastBody, stop := startProofServer(t, testItemProofJSON)
			defer stop()
			dir := withTempCWD(t)
			_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--witnesses", tc.flag, "--to-file", "01HJHB01T8FYZ7YTR9P5N62K5B")
			if exit != 0 {
				t.Fatalf("exit=%d, stderr=%q", exit, stderr)
			}
			data := postedData(t, *lastBody)
			got, has := data["witnesses"]
			if tc.wantList == nil {
				if has {
					t.Errorf("witnesses argument should be omitted, got %v", got)
				}
			} else if !has || len(got.([]any)) != len(tc.wantList.([]any)) {
				t.Errorf("witnesses = %v, want %v", got, tc.wantList)
			}
			want := filepath.Join(dir, "truestamp-item-01HJHB01T8FYZ7YTR9P5N62K5B"+tc.suffix+".json")
			if _, err := os.Stat(want); err != nil {
				t.Errorf("expected file %s: %v", want, err)
			}
		})
	}
	_, stderr, exit := runCLI(t, "--api-key", "test-key", "proofs", "get", "--witnesses", "block,nope", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit == 0 || !strings.Contains(stderr, "nope") {
		t.Errorf("unknown witness accepted: exit=%d stderr=%q", exit, stderr)
	}
}

// TestCLI_ProofsGet_NoTypeUUIDv7ResolvesThenFails: a bare UUIDv7 no longer
// fails outright — it is resolved against the server first. When
// resolution itself cannot answer, the failure must still be actionable
// and must not have generated a proof for a guessed type.
func TestCLI_ProofsGet_NoTypeUUIDv7ResolvesThenFails(t *testing.T) {
	var generateCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/utilities/resolve-id") {
			generateCalled = true
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"nope"}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key",
		"proofs", "get", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Error("an unresolvable bare UUIDv7 should fail")
	}
	if generateCalled {
		t.Error("no proof should be generated for a type that was never resolved")
	}
	if !strings.Contains(stderr, "--type") {
		t.Errorf("the error should name the escape hatch, got %q", stderr)
	}
}

func TestCLI_ProofsGet_Types(t *testing.T) {
	cases := []struct {
		typeFlag, id, body, format, wantFile string
	}{
		{"block", "019db702-b08c-73dc-a7cd-2c5e011f1dad", testBlockProofJSON, "json", "truestamp-block-019db702-b08c-73dc-a7cd-2c5e011f1dad.json"},
		{"beacon", "019db702-b08c-73dc-a7cd-2c5e011f1dad", testBeaconProofJSON, "json", "truestamp-beacon-019db702-b08c-73dc-a7cd-2c5e011f1dad.json"},
		{"item", "01HJHB01T8FYZ7YTR9P5N62K5B", testItemProofJSON, "json", "truestamp-item-01HJHB01T8FYZ7YTR9P5N62K5B.json"},
		{"entropy_stellar", "019db702-b08c-73dc-a7cd-2c5e011f1dad", testEntropyStellarProofJSON, "json", "truestamp-entropy-stellar-019db702-b08c-73dc-a7cd-2c5e011f1dad.json"},
	}
	for _, tc := range cases {
		t.Run(tc.typeFlag, func(t *testing.T) {
			url, lastBody, stop := startProofServer(t, tc.body)
			defer stop()
			dir := withTempCWD(t)
			_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", tc.typeFlag, "-f", tc.format, "--to-file", tc.id)
			if exit != 0 {
				t.Fatalf("exit=%d, stderr=%q", exit, stderr)
			}
			if _, err := os.Stat(filepath.Join(dir, tc.wantFile)); err != nil {
				t.Errorf("expected file %s: %v", tc.wantFile, err)
			}
			assertWireType(t, *lastBody, tc.typeFlag)
		})
	}
}

func TestCLI_ProofsGet_CBOR(t *testing.T) {
	// The server answers CBOR as base64 of the bytes; the file must carry
	// the decoded bytes, tag and all.
	cborBytes, _ := os.ReadFile(testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdCBOR))
	b64, _ := json.Marshal(base64Std(cborBytes))
	url, lastBody, stop := startProofServer(t, string(b64))
	defer stop()
	dir := withTempCWD(t)
	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "-f", "cbor", "--to-file", "01M1M0V3SE3C5P32TRAJSNX6QF")
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%q", exit, stderr)
	}
	got, err := os.ReadFile(filepath.Join(dir, "truestamp-item-01M1M0V3SE3C5P32TRAJSNX6QF.cbor"))
	if err != nil || string(got) != string(cborBytes) {
		t.Errorf("cbor file: %v (%d bytes, want %d)", err, len(got), len(cborBytes))
	}
	if postedData(t, *lastBody)["format"] != "cbor" {
		t.Errorf("format not posted: %s", *lastBody)
	}
}

// TestCLI_ProofsGet_PreservesNumbers pins that the pretty-printed JSON keeps
// every number literal as the server wrote it.
func TestCLI_ProofsGet_PreservesNumbers(t *testing.T) {
	body := strings.Replace(testItemProofJSON, `"claims":{"name":"x"}`, `"claims":{"name":"x","big":9007199254740993}`, 1)
	url, _, stop := startProofServer(t, body)
	defer stop()
	dir := withTempCWD(t)
	if _, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--to-file", "01HJHB01T8FYZ7YTR9P5N62K5B"); exit != 0 {
		t.Fatalf("exit=%d, stderr=%q", exit, stderr)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "truestamp-item-01HJHB01T8FYZ7YTR9P5N62K5B.json"))
	if !strings.Contains(string(got), "9007199254740993") {
		t.Errorf("large integer was rounded:\n%s", got)
	}
}

func TestCLI_ProofsGet_OutputFlagWins(t *testing.T) {
	url, _, stop := startProofServer(t, testBeaconProofJSON)
	defer stop()
	dir := withTempCWD(t)

	custom := filepath.Join(dir, "custom-name.json")
	_, _, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", "beacon", "-o", custom, "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Errorf("custom output path not honoured: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "truestamp-beacon-019db702-b08c-73dc-a7cd-2c5e011f1dad.json")); err == nil {
		t.Error("auto-named file should not exist when -o is set")
	}
}

func TestCLI_ProofsGet_InvalidType(t *testing.T) {
	_, stderr, exit := runCLI(t, "--api-key", "test-key", "proofs", "get", "--type", "bogus", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || !strings.Contains(stderr, "--type must be one of") {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	for _, want := range []string{"entropy_nist", "entropy_stellar", "entropy_bitcoin", "beacon"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("want %q listed in error message, got %q", want, stderr)
		}
	}
	_, stderr, exit = runCLI(t, "--api-key", "test-key", "proofs", "get", "--type", "entropy", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || !strings.Contains(stderr, "--type must be one of") {
		t.Errorf("bare entropy: exit=%d stderr=%q", exit, stderr)
	}
}

func TestCLI_ProofsGet_ShapeVsType(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "proofs", "get", "--type", "item", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || called || !strings.Contains(stderr, "requires a ULID") {
		t.Errorf("item+uuid: exit=%d called=%v stderr=%q", exit, called, stderr)
	}
	_, stderr, exit = runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "proofs", "get", "--type", "block", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit == 0 || called || !strings.Contains(stderr, "requires a UUIDv7") {
		t.Errorf("block+ulid: exit=%d called=%v stderr=%q", exit, called, stderr)
	}
}

// TestCLI_ProofsGet_NotCommittedError surfaces the server's
// no_external_commitments answer with the wait-for-the-epoch advice.
func TestCLI_ProofsGet_NotCommittedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"Subject has not yet been committed to a public blockchain. Try again after the next epoch commit.","meta":{"code":"no_external_commitments"}}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "proofs", "get", "--type", "block", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatalf("a refused generate must exit non-zero; stderr=%q", stderr)
	}
	// Assert on intent, not on a phrase that a line break can split: the
	// server's own detail, the machine-readable code, and the one thing
	// the server cannot say -- that waiting will fix this.
	for _, want := range []string{"not yet been committed", "no_external_commitments", "transient"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q\ngot: %s", want, stderr)
		}
	}
}

func TestCLI_ProofsGet_InvalidWitnessFromServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"invalid witness: valid names are block, entropy_stellar, entropy_nist, entropy_bitcoin, signing_key_event","meta":{"code":"invalid_witness"}}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)
	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "proofs", "get", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit == 0 || !strings.Contains(stderr, "invalid_witness") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
}

// TestCLI_ProofsGet_SubjectTypeMismatchError.
//
// 400, not 422. Every refusal on /proof/generate funnels through one
// constructor that builds an Ash.Error.Changes.InvalidChanges, whose class
// :invalid maps to 400 -- invalid_type, invalid_format, invalid_witness,
// id_format_mismatch, subject_type_mismatch, subject_not_ready,
// no_external_commitments, subject_not_recomputable and generation_failed
// alike. The only other outcomes on this route are 201 for a generated
// bundle and 404 for a subject that does not exist, both confirmed against
// the live API. No fixture here should model any other status.
func TestCLI_ProofsGet_SubjectTypeMismatchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"code":"subject_type_mismatch","detail":"Requested type entropy_nist but subject 019db702-b08c-73dc-a7cd-2c5e011f1dad has source entropy_stellar","meta":{"code":"subject_type_mismatch"}}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "proofs", "get", "--type", "entropy_nist", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || !strings.Contains(stderr, "Requested type entropy_nist") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
}

// base64Std encodes bytes the way the API returns a CBOR bundle.
func base64Std(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// --- R10 payload triad -------------------------------------------------
//
// The default changed from "always write a conventionally-named file into
// the cwd" to "write the bundle to stdout". That is what makes
// `truestamp proofs get <id> | truestamp verify --offline` possible without
// a temp file, in a CLI whose docs are built on pipelines. These tests pin
// all three destinations and the two refusals.

// TestCLI_ProofsGet_DefaultsToStdout is the behavior change itself: bytes on
// stdout, and nothing written into the working directory.
func TestCLI_ProofsGet_DefaultsToStdout(t *testing.T) {
	url, _, stop := startProofServer(t, testItemProofJSON)
	defer stop()
	dir := withTempCWD(t)

	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key",
		"proofs", "get", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%q", exit, stderr)
	}
	if !strings.Contains(stdout, `"signature"`) {
		t.Errorf("bundle did not reach stdout, got %d bytes: %q", len(stdout), truncate(stdout, 200))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading cwd: %v", err)
	}
	if len(entries) != 0 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("stdout mode must not write files, found: %v", names)
	}
}

// TestCLI_ProofsGet_StdoutIsPipeable proves the point of the change: the
// bytes on stdout are a bundle `verify` accepts. A test that only checked
// for a substring would not.
func TestCLI_ProofsGet_StdoutIsPipeable(t *testing.T) {
	// A JSON-format download returns the proof object itself, so the
	// fixture is the response body verbatim.
	bundle, err := os.ReadFile(testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	url, _, stop := startProofServer(t, string(bundle))
	defer stop()
	withTempCWD(t)

	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key",
		"proofs", "get", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit != 0 {
		t.Fatalf("download exit=%d, stderr=%q", exit, stderr)
	}

	// Feed exactly those bytes back to verify over stdin.
	vc := exec.Command(binaryPath, "verify", "--offline",
		"--keyring", testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdKeyring), "-")
	vc.Stdin = strings.NewReader(stdout)
	out, vErr := vc.CombinedOutput()
	if vErr != nil {
		t.Errorf("piping download into verify failed: %v\n%s", vErr, out)
	}
}

// TestCLI_ProofsGet_OutAndToFileConflict: the two file destinations are
// mutually exclusive, and the error must name both rather than silently
// letting one win.
func TestCLI_ProofsGet_OutAndToFileConflict(t *testing.T) {
	url, _, stop := startProofServer(t, testItemProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get",
		"-o", filepath.Join(dir, "x.json"), "--to-file", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit == 0 {
		t.Fatal("--out with --to-file should fail")
	}
	for _, want := range []string{"--out", "--to-file"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("error should name %s, got: %q", want, stderr)
		}
	}
}

// TestCLI_ProofsGet_ReceiptGoesToStderr keeps stdout usable when a file was
// written: the card is a receipt for a human, not part of any pipeline.
func TestCLI_ProofsGet_ReceiptGoesToStderr(t *testing.T) {
	url, _, stop := startProofServer(t, testItemProofJSON)
	defer stop()
	withTempCWD(t)

	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key",
		"proofs", "get", "--to-file", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%q", exit, stderr)
	}
	if !strings.Contains(stderr, "Proof Downloaded") {
		t.Errorf("receipt card should be on stderr, got: %q", stderr)
	}
	if strings.Contains(stdout, "Proof Downloaded") {
		t.Errorf("receipt card leaked into stdout: %q", stdout)
	}
}

// --- --type resolution -------------------------------------------------

// TestCLI_ProofsGet_ResolvesTypeForUUIDv7 pins the one place id-shape
// dispatch cannot work: a ULID is unambiguously an item, but blocks,
// beacons and entropy observations all use UUIDv7, so nothing client-side
// can tell them apart. Rather than requiring --type, the CLI asks the
// server, at the cost of one round trip.
func TestCLI_ProofsGet_ResolvesTypeForUUIDv7(t *testing.T) {
	var resolveCalls int
	var proofBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/utilities/resolve-id"):
			resolveCalls++
			// AshJsonApi generic routes take arguments under `data`; a flat
			// body is refused. Assert we send the shape the server wants.
			if !strings.Contains(string(body), `"data"`) {
				t.Errorf("resolve-id body must wrap arguments in `data`, got %s", body)
			}
			_, _ = w.Write([]byte(`{"result":{"id_format":"uuidv7","matches":[
			  {"kind":"block","proof_type":"block","verifiable":true}]}}`))
		default:
			proofBody = string(body)
			_, _ = w.Write([]byte(`{"result":` + testBlockProofJSON + `}`))
		}
	}))
	defer srv.Close()

	stdout, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "k",
		"proofs", "get", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	if resolveCalls != 1 {
		t.Errorf("expected exactly one resolve-id call, got %d", resolveCalls)
	}
	if !strings.Contains(proofBody, `"block"`) {
		t.Errorf("the resolved type should reach /proof/generate, got %s", proofBody)
	}
	if len(stdout) == 0 {
		t.Error("the bundle should still reach stdout")
	}
}

// TestCLI_ProofsGet_ExplicitTypeSkipsResolution: --type is one fewer round
// trip and keeps an id-classification service out of the loop entirely.
func TestCLI_ProofsGet_ExplicitTypeSkipsResolution(t *testing.T) {
	var resolveCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/utilities/resolve-id") {
			resolveCalls++
		}
		_, _ = w.Write([]byte(`{"result":` + testBlockProofJSON + `}`))
	}))
	defer srv.Close()

	_, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "k",
		"proofs", "get", "--type", "block", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if resolveCalls != 0 {
		t.Errorf("an explicit --type must not trigger resolution, got %d calls", resolveCalls)
	}
}

// TestCLI_ProofsGet_AmbiguousResolutionRefuses: a subject verifiable as
// more than one type must not be guessed at. Which one the caller meant
// changes what the proof commits to, and a block signature does not verify
// as a beacon signature.
func TestCLI_ProofsGet_AmbiguousResolutionRefuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"id_format":"uuidv7","matches":[
		  {"kind":"block","proof_type":"block"},
		  {"kind":"beacon","proof_type":"beacon"}]}}`))
	}))
	defer srv.Close()

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "k",
		"proofs", "get", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatal("an ambiguous id should be refused, not guessed")
	}
	if !strings.Contains(stderr, "--type") {
		t.Errorf("the error should tell the caller to disambiguate, got %q", stderr)
	}
}

// TestCLI_ProofsGet_UnresolvableIdSaysSo keeps the failure actionable: an
// id the caller cannot see returns no match, and the message must point at
// the escape hatch rather than just failing.
func TestCLI_ProofsGet_UnresolvableIdSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"id_format":"uuidv7","matches":[]}}`))
	}))
	defer srv.Close()

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "k",
		"proofs", "get", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatal("an unresolvable id should fail")
	}
	if !strings.Contains(stderr, "--type") {
		t.Errorf("the error should name the escape hatch, got %q", stderr)
	}
}

// TestCLI_ProofsGet_ExplainsGenerateCodes.
//
// The status here is 400, not 422. These refusals are
// Ash.Error.Changes.InvalidChanges, whose class :invalid maps to 400, and
// the JSON:API `code` member is the generic "invalid" for all of them --
// the Truestamp code is always in `meta.code`, which is what the parser
// reads. The renderer does not branch on the status at all, so these
// fixtures exist to model the wire shape truthfully rather than to drive
// behaviour.
//
// The server can say what went wrong; it cannot say whether waiting will
// help. That distinction is the whole difference between a caller polling
// for five minutes and a caller polling forever, and it is what
// `subject_not_recomputable` exists to make possible -- the condition it
// names is permanent, where `no_external_commitments` clears on its own.
func TestCLI_ProofsGet_ExplainsGenerateCodes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		meta     string
		detail   string
		wantAll  []string
		wantNone []string
	}{
		{
			name:   "subject_not_recomputable names what drifted and says it is permanent",
			status: 400,
			meta:   `"code":"subject_not_recomputable","drifted":"metadata"`,
			detail: "The item's stored metadata no longer reproduces the hash committed at submission.",
			wantAll: []string{
				"subject_not_recomputable", "What drifted: metadata.",
				"Retry: no, this condition is permanent.",
			},
		},
		{
			// The verdict comes from meta.code, so it is unchanged when the
			// server rewords `detail` -- which it has done once already.
			// This case carries no permanence wording of its own.
			name:   "the retry verdict survives a reworded detail",
			status: 400,
			meta:   `"code":"subject_not_recomputable","drifted":"claims and metadata"`,
			detail: "Stored data no longer reproduces its committed hash.",
			wantAll: []string{
				"What drifted: claims and metadata.",
				"Retry: no, this condition is permanent.",
			},
		},
		{
			name:   "generation_failed surfaces the failing steps",
			status: 400,
			meta:   `"code":"generation_failed","failed_steps":"Inclusion proof INVALID (derived root does not match block merkle_root)"`,
			detail: "Generated proof failed internal verification",
			wantAll: []string{
				"generation_failed",
				"Failed checks: Inclusion proof INVALID (derived root does not match block merkle_root)",
			},
		},
		{
			// The server embeds the steps in detail as well; printing both
			// reads as two separate failures.
			name:    "generation_failed does not repeat steps already in the detail",
			status:  400,
			meta:    `"code":"generation_failed","failed_steps":"Inclusion proof INVALID"`,
			detail:  "Generated proof failed internal verification: Inclusion proof INVALID",
			wantAll: []string{"Inclusion proof INVALID"},
			// "Failed checks:" would mean it was appended a second time.
			wantNone: []string{"Failed checks:"},
		},
		{
			// The server's retryable set is closed and has two members;
			// this is the other one. Everything else is terminal by
			// omission, so a code with no case must not claim retryability.
			name:    "subject_not_ready is reported as transient",
			status:  400,
			meta:    `"code":"subject_not_ready"`,
			detail:  "Subject is not yet ready for proof generation.",
			wantAll: []string{"subject_not_ready", "Retry: yes, this is transient.", "Try again shortly"},
		},
		{
			// A code the CLI has no case for must render plainly: no
			// invented retry advice, no invented permanence.
			name:     "an unknown code gets no invented advice",
			status:   400,
			meta:     `"code":"some_future_code"`,
			detail:   "Something the CLI has never seen.",
			wantAll:  []string{"some_future_code", "Something the CLI has never seen."},
			wantNone: []string{"Retry:", "transient", "permanent", "Try again"},
		},
		{
			name:    "no_external_commitments is reported as transient",
			status:  400,
			meta:    `"code":"no_external_commitments"`,
			detail:  "Subject has not yet been committed to a public blockchain.",
			wantAll: []string{"Retry: yes, this is transient.", "Try again shortly"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(tc.status)
				fmt.Fprintf(w, `{"errors":[{"code":"invalid","status":"%d","detail":%q,"meta":{%s}}]}`,
					tc.status, tc.detail, tc.meta)
			}))
			defer srv.Close()

			_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "k",
				"proofs", "get", "--type", "item", "01KNN33GX5E470CB9TRWAYF9DD")
			if exit == 0 {
				t.Fatal("a refused generate must exit non-zero")
			}
			for _, want := range tc.wantAll {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr missing %q\ngot: %s", want, stderr)
				}
			}
			for _, none := range tc.wantNone {
				if strings.Contains(stderr, none) {
					t.Errorf("stderr should not contain %q\ngot: %s", none, stderr)
				}
			}
		})
	}
}
