// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("missing Bearer header")
		}
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

func TestCLI_Download_SmartDefaultULIDItem(t *testing.T) {
	url, lastBody, stop := startProofServer(t, testItemProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "download", "01HJHB01T8FYZ7YTR9P5N62K5B")
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

func TestCLI_Download_Witnesses(t *testing.T) {
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
			_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "download", "--witnesses", tc.flag, "01HJHB01T8FYZ7YTR9P5N62K5B")
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
	_, stderr, exit := runCLI(t, "--api-key", "test-key", "download", "--witnesses", "block,nope", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit == 0 || !strings.Contains(stderr, "nope") {
		t.Errorf("unknown witness accepted: exit=%d stderr=%q", exit, stderr)
	}
}

func TestCLI_Download_NoTypeUUIDv7Errors(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "download", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || called || !strings.Contains(stderr, "--type is required for UUIDv7") {
		t.Errorf("exit=%d called=%v stderr=%q", exit, called, stderr)
	}
}

func TestCLI_Download_Types(t *testing.T) {
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
			_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "download", "--type", tc.typeFlag, "-f", tc.format, tc.id)
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

func TestCLI_Download_CBOR(t *testing.T) {
	// The server answers CBOR as base64 of the bytes; the file must carry
	// the decoded bytes, tag and all.
	cborBytes, _ := os.ReadFile(testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdCBOR))
	b64, _ := json.Marshal(base64Std(cborBytes))
	url, lastBody, stop := startProofServer(t, string(b64))
	defer stop()
	dir := withTempCWD(t)
	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "download", "-f", "cbor", "01M1M0V3SE3C5P32TRAJSNX6QF")
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

// TestCLI_Download_PreservesNumbers pins that the pretty-printed JSON keeps
// every number literal as the server wrote it.
func TestCLI_Download_PreservesNumbers(t *testing.T) {
	body := strings.Replace(testItemProofJSON, `"claims":{"name":"x"}`, `"claims":{"name":"x","big":9007199254740993}`, 1)
	url, _, stop := startProofServer(t, body)
	defer stop()
	dir := withTempCWD(t)
	if _, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "download", "01HJHB01T8FYZ7YTR9P5N62K5B"); exit != 0 {
		t.Fatalf("exit=%d, stderr=%q", exit, stderr)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "truestamp-item-01HJHB01T8FYZ7YTR9P5N62K5B.json"))
	if !strings.Contains(string(got), "9007199254740993") {
		t.Errorf("large integer was rounded:\n%s", got)
	}
}

func TestCLI_Download_OutputFlagWins(t *testing.T) {
	url, _, stop := startProofServer(t, testBeaconProofJSON)
	defer stop()
	dir := withTempCWD(t)

	custom := filepath.Join(dir, "custom-name.json")
	_, _, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "download", "--type", "beacon", "-o", custom, "019db702-b08c-73dc-a7cd-2c5e011f1dad")
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

func TestCLI_Download_InvalidType(t *testing.T) {
	_, stderr, exit := runCLI(t, "--api-key", "test-key", "download", "--type", "bogus", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || !strings.Contains(stderr, "--type must be one of") {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	for _, want := range []string{"entropy_nist", "entropy_stellar", "entropy_bitcoin", "beacon"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("want %q listed in error message, got %q", want, stderr)
		}
	}
	_, stderr, exit = runCLI(t, "--api-key", "test-key", "download", "--type", "entropy", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || !strings.Contains(stderr, "--type must be one of") {
		t.Errorf("bare entropy: exit=%d stderr=%q", exit, stderr)
	}
}

func TestCLI_Download_ShapeVsType(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "download", "--type", "item", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || called || !strings.Contains(stderr, "requires a ULID") {
		t.Errorf("item+uuid: exit=%d called=%v stderr=%q", exit, called, stderr)
	}
	_, stderr, exit = runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "download", "--type", "block", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit == 0 || called || !strings.Contains(stderr, "requires a UUIDv7") {
		t.Errorf("block+ulid: exit=%d called=%v stderr=%q", exit, called, stderr)
	}
}

// TestCLI_Download_NotCommittedError surfaces the server's
// no_external_commitments answer with the wait-for-the-epoch advice.
func TestCLI_Download_NotCommittedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"Subject has not yet been committed to a public blockchain. Try again after the next epoch commit.","meta":{"code":"no_external_commitments"}}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "download", "--type", "block", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || !strings.Contains(stderr, "not yet been committed") || !strings.Contains(stderr, "first public-chain commitment") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
}

func TestCLI_Download_InvalidWitnessFromServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"invalid witness: valid names are block, entropy_stellar, entropy_nist, entropy_bitcoin, signing_key_event","meta":{"code":"invalid_witness"}}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)
	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "download", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit == 0 || !strings.Contains(stderr, "invalid_witness") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
}

func TestCLI_Download_SubjectTypeMismatchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"code":"subject_type_mismatch","detail":"Requested type entropy_nist but subject 019db702-b08c-73dc-a7cd-2c5e011f1dad has source entropy_stellar","meta":{"code":"subject_type_mismatch"}}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "download", "--type", "entropy_nist", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || !strings.Contains(stderr, "Requested type entropy_nist") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
}

// base64Std encodes bytes the way the API returns a CBOR bundle.
func base64Std(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
