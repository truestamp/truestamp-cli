// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

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
	// The list route answers a JSON:API document; the single routes keep
	// the bare object above.
	testBeaconResource = `{"type":"beacon","id":"` + testBeaconID + `","attributes":{"hash":"` + testBeaconHash +
		`","timestamp":"` + testBeaconTimestamp + `","previous_hash":"` + testBeaconPrev + `"}}`
)

// startBeaconServer spins up an httptest server that serves static beacon
// responses. Returns the base URL and a cleanup func. The stub is minimal
// by design, unit coverage of the client lives in internal/beacons.
func startBeaconServer(t *testing.T) (string, func()) {
	t.Helper()
	mux := http.NewServeMux()
	// Routes mirror the production layout under <base_url>/api/json/...
	// so tests pass --base-url <srv.URL> just like real users would.
	mux.HandleFunc("/api/json/beacons/latest", func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(testBeaconJSON))
	})
	mux.HandleFunc("/api/json/beacons/"+testBeaconID, func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(testBeaconJSON))
	})
	mux.HandleFunc("/api/json/beacons/by-hash/"+testBeaconHash, func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(testBeaconJSON))
	})
	mux.HandleFunc("/api/json/beacons", func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		if r.URL.Query().Get("page[limit]") == "" {
			t.Errorf("the list must page with page[limit], got %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"data":[` + testBeaconResource + `,` + testBeaconResource + `],` +
			`"links":{"first":"","self":"","next":null,"prev":null},"meta":{"page":{"limit":2}}}`))
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
// unconditionally, localhost, 127.0.0.1, and plain-http all produce
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

func TestCLI_Beacons_Latest_HashOnly(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--base-url", url, "--api-key", "test-key",
		"beacons", "latest", "--hash-only")
	if exit != 0 {
		t.Fatalf("exit=%d, stdout=%q", exit, stdout)
	}
	got := strings.TrimRight(stdout, "\n")
	if got != testBeaconHash {
		t.Errorf("want hash=%q, got %q", testBeaconHash, got)
	}
}

// TestCLI_Beacons_BareGroupPrintsHelp pins R0: a bare group is a
// namespace, never a command. `truestamp beacon` used to run `latest`,
// which made the group name mean two things and hid the word `latest`
// from the people who most needed to learn it. It also required a
// credential, so on a fresh machine the answer to "what can I do here?"
// was an auth error.
func TestCLI_Beacons_BareGroupPrintsHelp(t *testing.T) {
	// Deliberately no --api-key: help must work with no credential.
	stdout, _, exit := runCLI(t, "beacons")
	if exit != 0 {
		t.Fatalf("a bare group must exit 0, got %d", exit)
	}
	for _, want := range []string{"latest", "list", "get"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("bare group help should list %q, got:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, testBeaconHash) {
		t.Error("a bare group must not run latest")
	}
}

func TestCLI_Beacons_Latest_JSON(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--base-url", url, "--api-key", "test-key",
		"beacons", "latest", "--json")
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

func TestCLI_Beacons_List_JSON(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--base-url", url, "--api-key", "test-key",
		"beacons", "list", "--limit", "2", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d, stdout=%q", exit, stdout)
	}
	var got struct {
		Beacons    []map[string]any `json:"beacons"`
		NextCursor string           `json:"next_cursor"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(got.Beacons) != 2 || got.Beacons[0]["id"] != testBeaconID {
		t.Fatalf("want 2 beacons under the noun with identity from the resource object, got %s", stdout)
	}
	if got.NextCursor != "" {
		t.Errorf("an explicit null next link is no cursor, got %q", got.NextCursor)
	}
}

// TestCLI_Beacons_List_HasNoHashOnly: --hash-only was registered on
// `list` purely to reject it with a message. A flag that exists only to
// fail is a trap; it is now simply not registered there, so the shell
// completes it only where it works and cobra reports an unknown flag.
func TestCLI_Beacons_List_HasNoHashOnly(t *testing.T) {
	_, stderr, exit := runCLI(t, "beacons", "list", "--hash-only")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "unknown flag") {
		t.Errorf("want an unknown-flag error, stderr=%q", stderr)
	}
}

func TestCLI_Beacons_Get_HashOnly(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--base-url", url, "--api-key", "test-key",
		"beacons", "get", testBeaconID, "--hash-only")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if strings.TrimRight(stdout, "\n") != testBeaconHash {
		t.Errorf("want hash, got %q", stdout)
	}
}

func TestCLI_Beacons_GetByHash_HashOnly(t *testing.T) {
	url, stop := startBeaconServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--base-url", url, "--api-key", "test-key",
		"beacons", "get", testBeaconHash, "--hash-only")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if strings.TrimRight(stdout, "\n") != testBeaconHash {
		t.Errorf("want hash, got %q", stdout)
	}
}

func TestCLI_Beacons_Get_BadUUIDClientSide(t *testing.T) {
	// Server should never be hit, client-side validation rejects first.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	_, stderr, exit := runCLI(t,
		"--base-url", srv.URL, "--api-key", "test-key",
		"beacons", "get", "not-a-uuid")
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

func TestCLI_Beacons_GetByHash_BadHashClientSide(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	_, stderr, exit := runCLI(t,
		"--base-url", srv.URL, "--api-key", "test-key",
		"beacons", "get", "ABCDEF")
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

func TestCLI_Beacons_MissingAPIKey_NotAuthenticated(t *testing.T) {
	// Even if we give a URL, without an API key the client should fail
	// fast with a non-silent "Not authenticated" banner on stderr.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	_, stderr, exit := runCLI(t,
		"--base-url", srv.URL,
		"beacons", "latest")
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

func TestCLI_Beacons_MutualExclusion_SilentJSON(t *testing.T) {
	_, stderr, exit := runCLI(t,
		"--api-key", "test-key",
		"beacons", "latest", "--silent", "--json")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("want mutual-exclusion message, stderr=%q", stderr)
	}
}
