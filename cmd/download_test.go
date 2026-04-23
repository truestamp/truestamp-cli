// Copyright (c) 2021-2026 Truestamp, Inc.
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

// Minimal block-proof fixture (t=10, no s, no ip, one cx entry). Just
// enough fields to pass the CLI's post-download shape sniff and the
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
	testEntropyProofJSON = `{
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
// request body so tests can assert what the client actually sent,
// and responds with the provided proof envelope.
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

func TestCLI_Download_AutoBlock(t *testing.T) {
	url, lastBody, stop := startProofServer(t, testBlockProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, stderr, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%q", exit, stderr)
	}
	wantFile := filepath.Join(dir, "truestamp-block-019db702-b08c-73dc-a7cd-2c5e011f1dad.json")
	if _, err := os.Stat(wantFile); err != nil {
		t.Errorf("expected file %s: %v", wantFile, err)
	}
	// --type auto should send type="auto" on the wire.
	assertWireType(t, *lastBody, "auto")
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

func TestCLI_Download_TypeBeacon_AliasesToBlockOnWire(t *testing.T) {
	// --type beacon is a client-side alias: file is labelled 'beacon'
	// but the wire request carries type="block".
	url, lastBody, stop := startProofServer(t, testBlockProofJSON)
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
	assertWireType(t, *lastBody, "block")
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

func TestCLI_Download_TypeEntropy(t *testing.T) {
	url, lastBody, stop := startProofServer(t, testEntropyProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "--type", "entropy", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if _, err := os.Stat(filepath.Join(dir,
		"truestamp-entropy-019db702-b08c-73dc-a7cd-2c5e011f1dad.json")); err != nil {
		t.Errorf("missing entropy file: %v", err)
	}
	assertWireType(t, *lastBody, "entropy")
}

func TestCLI_Download_AutoClassifiesEntropy(t *testing.T) {
	// Server returns t=31 → stem should be "entropy" even under --type auto.
	url, _, stop := startProofServer(t, testEntropyProofJSON)
	defer stop()
	dir := withTempCWD(t)

	_, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"download", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if _, err := os.Stat(filepath.Join(dir,
		"truestamp-entropy-019db702-b08c-73dc-a7cd-2c5e011f1dad.json")); err != nil {
		t.Errorf("auto-stem failed to classify entropy: %v", err)
	}
}

func TestCLI_Download_OutputFlagWins(t *testing.T) {
	url, _, stop := startProofServer(t, testBlockProofJSON)
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
	// No auto-generated file should exist.
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
}

func TestCLI_Download_NotCommittedError(t *testing.T) {
	// Surface the server's "Subject has not yet been committed" 400 verbatim.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"Subject has not yet been committed to a public blockchain. Try again after the next epoch commit."}]}`))
	}))
	defer srv.Close()
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t,
		"--api-url", srv.URL, "--api-key", "test-key",
		"download", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "not yet been committed") {
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
