// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/ui"
)

const (
	testBeaconID        = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
	testBeaconHash      = "ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b"
	testBeaconPrev      = "1c4812bdfec2bf29333136d86bc996f866e38177acc90565a0554c7ec698029b"
	testBeaconTimestamp = "2026-04-22T21:05:00.000000Z"
	testBeaconJSON      = `{"id":"` + testBeaconID + `","hash":"` + testBeaconHash +
		`","timestamp":"` + testBeaconTimestamp + `","previous_hash":"` + testBeaconPrev + `"}`
)

// startBeaconServer spins up an httptest server that serves static beacon
// responses. Returns the base URL and a cleanup func. The stub is minimal
// by design — unit coverage of the client lives in internal/beacons.
func startBeaconServer(t *testing.T) (string, func()) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/beacons/latest", func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(testBeaconJSON))
	})
	mux.HandleFunc("/beacons/"+testBeaconID, func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(testBeaconJSON))
	})
	mux.HandleFunc("/beacons/by-hash/"+testBeaconHash, func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(testBeaconJSON))
	})
	mux.HandleFunc("/beacons", func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(`[` + testBeaconJSON + `,` + testBeaconJSON + `]`))
	})
	srv := httptest.NewServer(mux)
	return srv.URL, srv.Close
}

func requireBearer(t *testing.T, r *http.Request) {
	t.Helper()
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		t.Errorf("missing Bearer header, got %q", auth)
	}
}

// Unit coverage for the public-web link helpers. URLs render
// unconditionally — localhost, 127.0.0.1, and plain-http all produce
// a rendered link; suppression was removed by explicit user request
// so developers can see card URLs against their dev server.
func TestBeaconURLHelpers(t *testing.T) {
	const (
		hash = "ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b"
		id   = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
	)
	cases := []struct {
		name       string
		apiURL     string
		wantDetail string
		wantVerify string
	}{
		{
			name:       "prod https with /api/json suffix",
			apiURL:     "https://www.truestamp.com/api/json",
			wantDetail: "https://www.truestamp.com/beacons/" + hash,
			wantVerify: "https://www.truestamp.com/verify/beacon/" + id,
		},
		{
			name:       "prod https without suffix",
			apiURL:     "https://example.com",
			wantDetail: "https://example.com/beacons/" + hash,
			wantVerify: "https://example.com/verify/beacon/" + id,
		},
		{
			name:       "trailing slash is stripped",
			apiURL:     "https://example.com/api/json/",
			wantDetail: "https://example.com/beacons/" + hash,
			wantVerify: "https://example.com/verify/beacon/" + id,
		},
		{
			name:       "plain http still renders",
			apiURL:     "http://example.com/api/json",
			wantDetail: "http://example.com/beacons/" + hash,
			wantVerify: "http://example.com/verify/beacon/" + id,
		},
		{
			name:       "localhost still renders",
			apiURL:     "http://localhost:4000/api/json",
			wantDetail: "http://localhost:4000/beacons/" + hash,
			wantVerify: "http://localhost:4000/verify/beacon/" + id,
		},
		{
			name:       "127.0.0.1 still renders",
			apiURL:     "http://127.0.0.1:4000/api/json",
			wantDetail: "http://127.0.0.1:4000/beacons/" + hash,
			wantVerify: "http://127.0.0.1:4000/verify/beacon/" + id,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ui.BeaconDetailURL(c.apiURL, hash); got != c.wantDetail {
				t.Errorf("ui.BeaconDetailURL(%q) = %q, want %q", c.apiURL, got, c.wantDetail)
			}
			if got := ui.BeaconVerifyURL(c.apiURL, id); got != c.wantVerify {
				t.Errorf("ui.BeaconVerifyURL(%q) = %q, want %q", c.apiURL, got, c.wantVerify)
			}
		})
	}
}

func runCLI(t *testing.T, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	cmd.Env = cleanEnv()
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}
	return outBuf.String(), errBuf.String(), exit
}

// cleanEnv returns a minimal env for CLI subprocess tests. It strips any
// TRUESTAMP_* vars the developer's shell may have set and redirects
// HOME/XDG_CONFIG_HOME to non-existent paths so no config file leaks in.
// This mirrors the approach used in the golden test suite.
func cleanEnv() []string {
	out := []string{
		"NO_COLOR=1",
		"TRUESTAMP_NO_UPGRADE_CHECK=1",
		"HOME=/tmp/truestamp-test-home-does-not-exist",
		"XDG_CONFIG_HOME=/tmp/truestamp-test-config-does-not-exist",
	}
	for _, e := range os.Environ() {
		switch {
		case strings.HasPrefix(e, "TRUESTAMP_"):
			continue
		case strings.HasPrefix(e, "HOME="):
			continue
		case strings.HasPrefix(e, "XDG_CONFIG_HOME="):
			continue
		}
		out = append(out, e)
	}
	return out
}

// --- beacon ---------------------------------------------------------------

func TestCLI_Beacon_Latest_HashOnly(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"beacon", "latest", "--hash-only")
	if exit != 0 {
		t.Fatalf("exit=%d, stdout=%q", exit, stdout)
	}
	got := strings.TrimRight(stdout, "\n")
	if got != testBeaconHash {
		t.Errorf("want hash=%q, got %q", testBeaconHash, got)
	}
}

func TestCLI_Beacon_Default_IsLatest(t *testing.T) {
	// `truestamp beacon` (no subcommand) should behave as `beacon latest`.
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"beacon", "--hash-only")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if strings.TrimRight(stdout, "\n") != testBeaconHash {
		t.Errorf("want hash, got %q", stdout)
	}
}

func TestCLI_Beacon_Latest_JSON(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"beacon", "latest", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	var got struct {
		ID, Hash, Timestamp, PreviousHash string `json:",inline"`
	}
	// Use a local anonymous struct with explicit tags.
	var parsed struct {
		ID           string `json:"id"`
		Hash         string `json:"hash"`
		Timestamp    string `json:"timestamp"`
		PreviousHash string `json:"previous_hash"`
	}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if parsed.Hash != testBeaconHash {
		t.Errorf("wrong hash: %q", parsed.Hash)
	}
	_ = got
}

func TestCLI_Beacon_List_JSON(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"beacon", "list", "--limit", "2", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d, stdout=%q", exit, stdout)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 entries, got %d", len(items))
	}
}

func TestCLI_Beacon_List_HashOnly_Rejected(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	_, stderr, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"beacon", "list", "--hash-only")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "not valid") {
		t.Errorf("want 'not valid' message, stderr=%q", stderr)
	}
}

func TestCLI_Beacon_Get_HashOnly(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"beacon", "get", testBeaconID, "--hash-only")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if strings.TrimRight(stdout, "\n") != testBeaconHash {
		t.Errorf("want hash, got %q", stdout)
	}
}

func TestCLI_Beacon_ByHash_HashOnly(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--api-url", url, "--api-key", "test-key",
		"beacon", "by-hash", testBeaconHash, "--hash-only")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if strings.TrimRight(stdout, "\n") != testBeaconHash {
		t.Errorf("want hash, got %q", stdout)
	}
}

func TestCLI_Beacon_Get_BadUUIDClientSide(t *testing.T) {
	// Server should never be hit — client-side validation rejects first.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	_, stderr, exit := runCLI(t,
		"--api-url", srv.URL, "--api-key", "test-key",
		"beacon", "get", "not-a-uuid")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if called {
		t.Error("server should not have been called")
	}
	if !strings.Contains(stderr, "UUID") {
		t.Errorf("want UUID error, stderr=%q", stderr)
	}
}

func TestCLI_Beacon_ByHash_BadHashClientSide(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	_, stderr, exit := runCLI(t,
		"--api-url", srv.URL, "--api-key", "test-key",
		"beacon", "by-hash", "ABCDEF")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if called {
		t.Error("server should not have been called")
	}
	if !strings.Contains(stderr, "hash") {
		t.Errorf("want hash error, stderr=%q", stderr)
	}
}

func TestCLI_Beacon_MissingAPIKey_NotAuthenticated(t *testing.T) {
	// Even if we give a URL, without an API key the client should fail
	// fast with a non-silent "Not authenticated" banner on stderr.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	_, stderr, exit := runCLI(t,
		"--api-url", srv.URL,
		"beacon", "latest")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if called {
		t.Error("server should not have been called")
	}
	if !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("want 'Not authenticated', stderr=%q", stderr)
	}
}

func TestCLI_Beacon_MutualExclusion_SilentJSON(t *testing.T) {
	_, stderr, exit := runCLI(t,
		"--api-key", "test-key",
		"beacon", "latest", "--silent", "--json")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("want mutual-exclusion message, stderr=%q", stderr)
	}
}
