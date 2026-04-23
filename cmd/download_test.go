// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Minimal proof fixtures (no s for block/beacon, with s for item/entropy).
// Just enough fields to pass the CLI's post-download shape sniff and any
// `t` parser. The byte shapes aren't crypto-valid — verify isn't called
// here. Unit coverage of verify lives in internal/verify.
const (
	testBlockProofJSON = `{
	  "v":1,"t":10,
	  "pk":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	  "sig":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
	  "ts":"2026-04-22T21:05:00Z",
	  "b":{"id":"019db702-b08c-73dc-a7cd-2c5e011f1dad","ph":"11","mr":"22","mh":"33","kid":"4ceefa4a"},
	  "cx":[{"t":40,"net":"testnet","ep":"AA","memo":"aa","tx":"bb","l":1}]
	}`
	testBeaconProofJSON = `{
	  "v":1,"t":11,
	  "pk":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	  "sig":"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB==",
	  "ts":"2026-04-22T21:05:00Z",
	  "b":{"id":"019db702-b08c-73dc-a7cd-2c5e011f1dad","ph":"11","mr":"22","mh":"33","kid":"4ceefa4a"},
	  "cx":[{"t":40,"net":"testnet","ep":"AA","memo":"aa","tx":"bb","l":1}]
	}`
	testItemProofJSON = `{
	  "v":1,"t":20,
	  "pk":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	  "sig":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
	  "ts":"2026-04-22T21:05:00Z",
	  "s":{"id":"01HJHB01T8FYZ7YTR9P5N62K5B","d":{"name":"x"},"mh":"33","kid":"4ceefa4a"},
	  "ip":"AA",
	  "b":{"id":"019db702-b08c-73dc-a7cd-2c5e011f1dad","ph":"11","mr":"22","mh":"33","kid":"4ceefa4a"},
	  "cx":[{"t":40,"net":"testnet","ep":"AA","memo":"aa","tx":"bb","l":1}]
	}`
	testEntropyStellarProofJSON = `{
	  "v":1,"t":31,
	  "pk":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	  "sig":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
	  "ts":"2026-04-22T21:05:00Z",
	  "s":{"id":"019db702-b08c-73dc-a7cd-2c5e011f1dad","d":{"hash":"x"},"mh":"33","kid":"4ceefa4a"},
	  "ip":"AA",
	  "b":{"id":"019db702-b08c-73dc-a7cd-2c5e011f1dad","ph":"11","mr":"22","mh":"33","kid":"4ceefa4a"},
	  "cx":[{"t":40,"net":"testnet","ep":"AA","memo":"aa","tx":"bb","l":1}]
	}`
)

// startProofServer spins up an httptest server that records the last
// request body so tests can assert what the client actually sent, and
// responds with the provided proof envelope.
func startProofServer(t *testing.T, responseBody string) (string, *string, func()) {
	t.Helper()
	var lastBody string
	mux := http.NewServeMux()
	mux.HandleFunc("/proof/generate", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
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

// TestCLI_Download_SmartDefaultULIDItem: a ULID positional with no --type
// flag should default to type=item and produce truestamp-item-*.
func TestCLI_Download_SmartDefaultULIDItem(t *testing.T) {
	url, lastBody, stop := startProofServer(t, testItemProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, stderr, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%q", exit, stderr)
	}
	wantFile := filepath.Join(dir, "truestamp-item-01HJHB01T8FYZ7YTR9P5N62K5B.json")
	if _, err := os.Stat(wantFile); err != nil {
		t.Errorf("expected file %s: %v", wantFile, err)
	}
	assertWireType(t, *lastBody, "item")
}

// TestCLI_Download_NoTypeUUIDv7Errors: a UUIDv7 positional with no --type
// flag must fail client-side without a network call.
func TestCLI_Download_NoTypeUUIDv7Errors(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t,
		"--api-url", srv.URL, "--api-key", "test-key",
		"download", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if called {
		t.Error("server should not have been called")
	}
	if !strings.Contains(stderr, "--type is required for UUIDv7") {
		t.Errorf("want UUIDv7-requires-type message, stderr=%q", stderr)
	}
}

func TestCLI_Download_TypeBlock(t *testing.T) {
	url, lastBody, stop := startProofServer(t, testBlockProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "--type", "block", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if _, err := os.Stat(filepath.Join(dir,
		"truestamp-block-019db702-b08c-73dc-a7cd-2c5e011f1dad.json")); err != nil {
		t.Errorf("missing block file: %v", err)
	}
	assertWireType(t, *lastBody, "block")
}

// TestCLI_Download_TypeBeacon_SendsBeacon: the beacon alias is gone.
// --type beacon must send type="beacon" on the wire and produce a
// truestamp-beacon-*.json file. Server is expected to return a t=11
// bundle (fixture does).
func TestCLI_Download_TypeBeacon_SendsBeacon(t *testing.T) {
	url, lastBody, stop := startProofServer(t, testBeaconProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "--type", "beacon", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if _, err := os.Stat(filepath.Join(dir,
		"truestamp-beacon-019db702-b08c-73dc-a7cd-2c5e011f1dad.json")); err != nil {
		t.Errorf("missing beacon file: %v", err)
	}
	assertWireType(t, *lastBody, "beacon")
}

func TestCLI_Download_TypeItem(t *testing.T) {
	url, lastBody, stop := startProofServer(t, testItemProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "--type", "item", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if _, err := os.Stat(filepath.Join(dir,
		"truestamp-item-01HJHB01T8FYZ7YTR9P5N62K5B.json")); err != nil {
		t.Errorf("missing item file: %v", err)
	}
	assertWireType(t, *lastBody, "item")
}

// TestCLI_Download_TypeEntropyStellar: subtype-specific wire value and a
// hyphenated filename stem (entropy-stellar, not entropy_stellar).
func TestCLI_Download_TypeEntropyStellar(t *testing.T) {
	url, lastBody, stop := startProofServer(t, testEntropyStellarProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "--type", "entropy_stellar", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if _, err := os.Stat(filepath.Join(dir,
		"truestamp-entropy-stellar-019db702-b08c-73dc-a7cd-2c5e011f1dad.json")); err != nil {
		t.Errorf("missing entropy-stellar file: %v", err)
	}
	assertWireType(t, *lastBody, "entropy_stellar")
}

// TestCLI_Download_TypeEntropyNIST_CBOR: ensures the hyphen translation
// works with CBOR too, and the NIST subtype routes correctly.
func TestCLI_Download_TypeEntropyNIST_CBOR(t *testing.T) {
	// Use the stellar JSON body — the CBOR output format is encoded on
	// the wire; the test only cares about filename + wire type field.
	// A real server would return base64-encoded CBOR here; the test
	// server returns whatever we hand it, which is fine for the
	// filename/wire-body assertions we make.
	cborEncoded := `"AAAA"` // base64-encoded fake CBOR bytes
	url, lastBody, stop := startProofServer(t, cborEncoded)
	defer stop()
	dir := withTempCWD(t)

	_, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "--type", "entropy_nist", "-f", "cbor",
		"019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if _, err := os.Stat(filepath.Join(dir,
		"truestamp-entropy-nist-019db702-b08c-73dc-a7cd-2c5e011f1dad.cbor")); err != nil {
		t.Errorf("missing entropy-nist cbor file: %v", err)
	}
	assertWireType(t, *lastBody, "entropy_nist")
}

func TestCLI_Download_OutputFlagWins(t *testing.T) {
	url, _, stop := startProofServer(t, testBeaconProofJSON)
	defer stop()
	dir := withTempCWD(t)

	custom := filepath.Join(dir, "custom-name.json")
	_, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "--type", "beacon", "-o", custom,
		"019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Errorf("custom output path not honoured: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir,
		"truestamp-beacon-019db702-b08c-73dc-a7cd-2c5e011f1dad.json")); err == nil {
		t.Error("auto-named file should not exist when -o is set")
	}
}

func TestCLI_Download_InvalidType(t *testing.T) {
	_, stderr, exit := runCLI(t,
		"--api-key", "test-key",
		"download", "--type", "bogus", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "--type must be one of") {
		t.Errorf("want --type validation message, got %q", stderr)
	}
	// The server's enum strings should be listed in the error.
	for _, want := range []string{"entropy_nist", "entropy_stellar", "entropy_bitcoin", "beacon"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("want %q listed in error message, got %q", want, stderr)
		}
	}
}

// TestCLI_Download_BareEntropyRejected: bare "entropy" is no longer a
// valid --type value (server dropped it; CLI must reject it client-side).
func TestCLI_Download_BareEntropyRejected(t *testing.T) {
	_, stderr, exit := runCLI(t,
		"--api-key", "test-key",
		"download", "--type", "entropy", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "--type must be one of") {
		t.Errorf("want --type validation message, got %q", stderr)
	}
}

// TestCLI_Download_TypeItemWithUUIDv7: --type item + UUIDv7 should fail
// client-side before the network call.
func TestCLI_Download_TypeItemWithUUIDv7(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t,
		"--api-url", srv.URL, "--api-key", "test-key",
		"download", "--type", "item", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if called {
		t.Error("server should not have been called")
	}
	if !strings.Contains(stderr, "requires a ULID") {
		t.Errorf("want ULID-required message, stderr=%q", stderr)
	}
}

// TestCLI_Download_TypeBlockWithULID: --type block + ULID should fail
// client-side before the network call.
func TestCLI_Download_TypeBlockWithULID(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t,
		"--api-url", srv.URL, "--api-key", "test-key",
		"download", "--type", "block", "01HJHB01T8FYZ7YTR9P5N62K5B")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if called {
		t.Error("server should not have been called")
	}
	if !strings.Contains(stderr, "requires a UUIDv7") {
		t.Errorf("want UUIDv7-required message, stderr=%q", stderr)
	}
}

// TestCLI_Download_NotCommittedError: surface the server's "not committed"
// 400 verbatim. Use --type block so the CLI pre-flight doesn't short-
// circuit (UUIDv7 + block is valid locally).
func TestCLI_Download_NotCommittedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"Subject has not yet been committed to a public blockchain. Try again after the next epoch commit."}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t,
		"--api-url", srv.URL, "--api-key", "test-key",
		"download", "--type", "block", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "not yet been committed") {
		t.Errorf("detail not preserved, stderr=%q", stderr)
	}
}

// TestCLI_Download_SubjectTypeMismatchError: surface the server's 422
// subject_type_mismatch verbatim when the caller asks for entropy_nist
// against a stellar observation.
func TestCLI_Download_SubjectTypeMismatchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"code":"subject_type_mismatch","detail":"Requested type entropy_nist but subject 019db702-b08c-73dc-a7cd-2c5e011f1dad has source entropy_stellar","meta":{"code":"subject_type_mismatch","requested_type":"entropy_nist","actual_source":"entropy_stellar"}}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t,
		"--api-url", srv.URL, "--api-key", "test-key",
		"download", "--type", "entropy_nist", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "Requested type entropy_nist") {
		t.Errorf("detail not preserved, stderr=%q", stderr)
	}
}

// assertWireType unmarshals the captured request body and checks that
// data.type equals want.
func assertWireType(t *testing.T, body, want string) {
	t.Helper()
	var parsed struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("captured body not valid JSON: %v\nbody=%q", err, body)
	}
	if got := parsed.Data["type"]; got != want {
		t.Errorf("wire type: want %q, got %q (full body: %s)", want, got, body)
	}
}
