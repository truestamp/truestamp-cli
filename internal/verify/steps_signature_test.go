// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// TestKeyringUnavailable pins that an online run whose keyring fetch fails
// ends failed with keyring_unavailable, whatever the reason the keyring
// could not be read, while an offline run, which never fetches, keeps its
// stated skip. Every chain and source endpoint is stubbed, so the keyring
// is the only thing that differs between the cases.
func TestKeyringUnavailable(t *testing.T) {
	data := prodBundle(t, testfixtures.ProdComplete)
	keyringDoc := testfixtures.Read(t, testfixtures.ProdDir, testfixtures.ProdKeyring)

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch r.URL.Path {
		case "/keyring.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(keyringDoc)
		case "/portal":
			// A 200 that is not a keyring: a captive portal or a CDN stub.
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html><body>Sign in to continue</body></html>"))
		case "/no-keys":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.0"}`))
		default:
			http.Error(w, `{"error":"unavailable"}`, http.StatusServiceUnavailable)
		}
	}))
	t.Cleanup(srv.Close)

	// A server started and immediately closed: its address refuses
	// connections, the stand-in for a keyring host that cannot be reached.
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachable := dead.URL + "/.well-known/keyring.json"
	dead.Close()

	withStubbedSources(t, &blockstreamStub{})

	keyBindingRow := func(t *testing.T, r *Report) Step {
		t.Helper()
		rows := rowsOf(r, groupKeyBinding)
		if len(rows) != 1 {
			t.Fatalf("Key Binding rows = %d, want 1\n%s", len(rows), render(r.Steps))
		}
		return rows[0]
	}

	for _, tc := range []struct{ name, url string }{
		{"unreachable host", unreachable},
		{"server error", srv.URL + "/unavailable"},
		{"answer is not JSON", srv.URL + "/portal"},
		{"answer has no keys list", srv.URL + "/no-keys"},
	} {
		t.Run("online, "+tc.name, func(t *testing.T) {
			r := mustRun(t, data, Options{KeyringURL: tc.url})
			row := keyBindingRow(t, r)
			if row.Status != StatusFail || !strings.HasPrefix(row.Message, "keyring_unavailable: ") ||
				!strings.Contains(row.Message, tc.url) {
				t.Errorf("Key Binding = %s %q, want a keyring_unavailable failure naming %s", row.Status, row.Message, tc.url)
			}
			if r.Passed() || BuildJSONReport(r).Passed {
				t.Errorf("verdict PASSED, want FAILED\n%s", render(r.Steps))
			}
			// The bundle is otherwise sound, so the keyring is the one
			// failing row.
			if n := r.FailedCount(); n != 1 {
				t.Errorf("failed rows = %d, want only Key Binding\n%s", n, render(r.Steps))
			}
			if r.KeyringSource != "" {
				t.Errorf("keyring source = %q, want none in hand", r.KeyringSource)
			}
		})
	}

	// The same stubbed sources with a keyring that can be read: the run
	// passes, so the failures above come from the keyring alone.
	t.Run("online, keyring read", func(t *testing.T) {
		r := mustRun(t, data, Options{KeyringURL: srv.URL + "/keyring.json"})
		if row := keyBindingRow(t, r); row.Status != StatusPass {
			t.Errorf("Key Binding = %s %q, want a pass", row.Status, row.Message)
		}
		if !r.Passed() {
			t.Errorf("verdict FAILED, want PASSED\n%s", render(r.Steps))
		}
	})

	for _, tc := range []struct{ name, url string }{
		{"unreachable host", unreachable},
		{"server error", srv.URL + "/unavailable"},
	} {
		t.Run("offline, "+tc.name, func(t *testing.T) {
			before := requests.Load()
			r := mustRun(t, data, Options{SkipExternal: true, KeyringURL: tc.url})
			if row := keyBindingRow(t, r); row.Status != StatusSkip || row.Message != keyringNotSupplied {
				t.Errorf("Key Binding = %s %q, want the skip %q", row.Status, row.Message, keyringNotSupplied)
			}
			if !r.Passed() {
				t.Errorf("verdict FAILED, want PASSED\n%s", render(r.Steps))
			}
			if got := requests.Load(); got != before {
				t.Errorf("an offline run made %d keyring requests", got-before)
			}
		})
	}
}
