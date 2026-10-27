// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// escByte is what must never reach a redirected stream. `truestamp ... >
// file` writing terminal escape sequences into that file is the bug this
// file exists to prevent.
const escByte = "\x1b"

// offlineAPIKey is the only credential an offline invocation carries. It
// is explicit (TRUESTAMP_API_KEY), so the CLI never consults a stored OAuth
// session.
const offlineAPIKey = "ts_offline_test_key"

// offlineEnv is the environment of every subprocess in this file and of
// the auth and upgrade tests: the inherited environment without any
// TRUESTAMP_* variable, HOME, XDG directory, NO_COLOR (which would hide the
// escapes these tests look for) or GITHUB_TOKEN, with HOME and the XDG
// directories in a fresh temporary directory. apiKey, when not empty, is
// set as TRUESTAMP_API_KEY.
func offlineEnv(t *testing.T, apiKey string) []string {
	t.Helper()
	home := t.TempDir()
	var env []string
	for _, e := range os.Environ() {
		name, _, _ := strings.Cut(e, "=")
		switch {
		case strings.HasPrefix(name, "TRUESTAMP_"), strings.HasPrefix(name, "XDG_"),
			name == "HOME", name == "NO_COLOR", name == "GITHUB_TOKEN":
			continue
		}
		env = append(env, e)
	}
	env = append(env,
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, "config"),
		"XDG_CACHE_HOME="+filepath.Join(home, "cache"),
		"TRUESTAMP_NO_UPGRADE_CHECK=1",
	)
	if apiKey != "" {
		env = append(env, "TRUESTAMP_API_KEY="+apiKey)
	}
	return env
}

// startOfflineAPI is the loopback Truestamp API every offline invocation
// is pointed at. It answers the `auth status` user probe for offlineAPIKey
// and fails the test on any other request.
func startOfflineAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/json/users" || r.Header.Get("Authorization") != "Bearer "+offlineAPIKey {
			t.Errorf("unexpected request to the offline API: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[{"id":"u-offline","type":"user","attributes":{"email":"offline@example.test","full_name":"Offline Tester"}}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// offlineCommands returns a builder of subprocesses that run the binary
// with one offlineEnv (so one HOME, and outputs that can be compared),
// offlineAPIKey and --base-url pointing at the loopback API, never the
// inherited environment.
func offlineCommands(t *testing.T, baseURL string) func(args ...string) *exec.Cmd {
	t.Helper()
	env := offlineEnv(t, offlineAPIKey)
	return func(args ...string) *exec.Cmd {
		c := exec.Command(binaryPath, append(append([]string{}, args...), "--base-url", baseURL)...)
		c.Env = env
		return c
	}
}

// offlineInvocations are real invocations that exercise the styled render
// paths. offlineCommands runs each one, so none reaches beyond the loopback
// API and none carries a credential but offlineAPIKey; `auth status`
// reaches its success card there. Every one of them writes through
// a lipgloss style somewhere; a command that only prints plain text would
// make this test vacuous, so the set is deliberately drawn from the ones
// that render cards, tables, banners, and reports.
//
// Every file an invocation reads is a vector.
func offlineInvocations() [][]string {
	return [][]string{
		{"version"},
		{"config", "show"},
		{"config", "path"},
		{"auth", "status"},
		{"verify", soundItem(), "--offline", "--environment", "test", "--keyring", testKeyring()},
		{"inspect", soundItem()},
		{"convert", "time", "2026-09-03T12:00:00Z"},
		{"convert", "keyid", "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="},
		{"convert", "id", "01a0db66-c2e0-78a4-b8cc-12b11bc97493"},
		{"hash", soundItem()},
	}
}

// TestCLI_RedirectedOutput_HasNoANSI runs each invocation with its stdout
// and stderr connected to pipes, i.e. exactly the shape of `truestamp ...
// > file`, and asserts no escape byte survives. Mutants killed: verify's
// text written with an ANSI escape of its own; inspect's text written
// through fmt.Fprint, which keeps its styles' escapes.
//
// This is the regression that motivated internal/ui's Fprint helpers:
// commands built a styled string with Style.Render and then handed it to
// fmt.Fprintln, which writes straight to the destination and keeps the
// escapes. lipgloss.Println was fine because it goes through a colour
// profile writer; the fmt path was not.
func TestCLI_RedirectedOutput_HasNoANSI(t *testing.T) {
	command := offlineCommands(t, startOfflineAPI(t).URL)
	for _, args := range offlineInvocations() {
		name := strings.Join(args, "_")
		t.Run(name, func(t *testing.T) {
			// CombinedOutput gives pipes for both streams, so neither is
			// a terminal, which is the condition under test.
			out, _ := command(args...).CombinedOutput()
			if len(out) == 0 {
				t.Fatalf("%v produced no output, so this case asserts nothing", args)
			}
			if i := strings.Index(string(out), escByte); i >= 0 {
				t.Errorf("%v leaked an ANSI escape at byte %d:\n%q", args, i, snippet(string(out), i))
			}
		})
	}
	t.Run("auth_status_unauthenticated", func(t *testing.T) {
		out := unauthenticatedStatus(t)
		if i := strings.Index(string(out), escByte); i >= 0 {
			t.Errorf("unauthenticated auth status leaked an ANSI escape at byte %d:\n%q", i, snippet(string(out), i))
		}
	})
}

// TestCLI_NoColorFlag_HasNoANSI runs the same set with --no-color. The
// flag must be sufficient on its own, independent of terminal detection,
// because that is what users reach for when the detection is wrong.
func TestCLI_NoColorFlag_HasNoANSI(t *testing.T) {
	command := offlineCommands(t, startOfflineAPI(t).URL)
	for _, args := range offlineInvocations() {
		name := strings.Join(args, "_")
		t.Run(name, func(t *testing.T) {
			full := append(append([]string{}, args...), "--no-color")
			out, _ := command(full...).CombinedOutput()
			if len(out) == 0 {
				t.Fatalf("%v produced no output, so this case asserts nothing", full)
			}
			if i := strings.Index(string(out), escByte); i >= 0 {
				t.Errorf("%v leaked an ANSI escape at byte %d:\n%q", full, i, snippet(string(out), i))
			}
		})
	}
	t.Run("auth_status_unauthenticated", func(t *testing.T) {
		out := unauthenticatedStatus(t, "--no-color")
		if i := strings.Index(string(out), escByte); i >= 0 {
			t.Errorf("unauthenticated auth status --no-color leaked an ANSI escape at byte %d:\n%q", i, snippet(string(out), i))
		}
	})
}

// unauthenticatedStatus runs `auth status` and extra with no credential,
// in an offlineEnv of its own, against a countingAPI, so the text failure
// card is what it renders. It asserts the card says Not authenticated and
// that no request was sent, and returns the combined output.
func unauthenticatedStatus(t *testing.T, extra ...string) []byte {
	t.Helper()
	srv, hits := countingAPI(t)
	c := exec.Command(binaryPath, append([]string{"auth", "status", "--base-url", srv.URL}, extra...)...)
	c.Env = offlineEnv(t, "")
	out, err := c.CombinedOutput()
	if err == nil {
		t.Errorf("auth status with no credential exited 0:\n%s", out)
	}
	if !strings.Contains(string(out), "Not authenticated") {
		t.Fatalf("auth status with no credential did not render the failure card:\n%s", out)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("auth status with no credential sent %d requests, want 0", n)
	}
	return out
}

// TestCLI_NoColorFlag_IsAcceptedEverywhere guards the flag's reach: it is
// registered as a root persistent flag, so every command must accept it.
// A command that rejects it would fail the test above for the wrong
// reason (empty output), which this distinguishes.
func TestCLI_NoColorFlag_IsAcceptedEverywhere(t *testing.T) {
	command := offlineCommands(t, startOfflineAPI(t).URL)
	for _, args := range offlineInvocations() {
		full := append(append([]string{}, args...), "--no-color")
		out, err := command(full...).CombinedOutput()
		if err != nil && strings.Contains(string(out), "unknown flag") {
			t.Errorf("%v rejected --no-color: %s", args, out)
		}
	}
}

// snippet returns a short window around index i for a readable failure.
func snippet(s string, i int) string {
	start := max(i-40, 0)
	end := min(i+40, len(s))
	return s[start:end]
}

// TestCLI_JSONFlagIsNeverANoOp: a flag that is advertised, accepted, and
// changes nothing is a flag that lies about what it does.
//
// `schema get` shipped that way — its documents are JSON by nature, so
// --json had nothing to switch, yet it was offered on every invocation.
// The fix was to give `schema get` a text rendering so --json switches
// something, not to keep an inert flag as a courtesy.
//
// This checks the property rather than the instance: for every offline
// command that offers --json, passing it must change the output. Mutant
// killed: inspect writing its text under --json.
func TestCLI_JSONFlagIsNeverANoOp(t *testing.T) {
	command := offlineCommands(t, startOfflineAPI(t).URL)
	for _, args := range [][]string{
		{"version"},
		{"config", "show"},
		{"schema", "list"},
		{"auth", "status"},
		{"inspect", soundItem()},
		{"hash", soundItem()},
		{"schema", "get", "exit-codes"},
		{"schema", "get", "reason-codes"},
		{"schema", "get", "commands"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			help, err := command(append(append([]string{}, args...), "--help")...).CombinedOutput()
			if err != nil {
				t.Fatalf("--help: %v", err)
			}
			if !strings.Contains(string(help), "--json") {
				t.Skipf("%v does not offer --json", args)
			}
			plain, _ := command(args...).Output()
			asJSON, _ := command(append(append([]string{}, args...), "--json")...).Output()
			if len(plain) == 0 && len(asJSON) == 0 {
				t.Fatalf("%v produced no output either way", args)
			}
			if string(plain) == string(asJSON) {
				t.Errorf("%v advertises --json but it changes nothing; either give it "+
					"a text rendering or stop offering the flag", args)
			}
		})
	}
}

// TestCLI_SchemaGetRendersTextByDefault: --json is universal across the
// tree, so `schema get` is not an exception to it. It renders a table (or
// the command outline) by default and the document with --json.
//
// This briefly went the other way — the flag was dropped because the
// output was JSON either way and an inert flag is worse than none. The
// better fix was to give the command the text rendering every other
// command has, so the contract stays uniform. Mutant killed: the
// exit-codes text rendering writing the JSON document.
func TestCLI_SchemaGetRendersTextByDefault(t *testing.T) {
	help, err := exec.Command(binaryPath, "schema", "get", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("schema get --help: %v", err)
	}
	if !strings.Contains(string(help), "--json") {
		t.Error("--json is universal; schema get must offer it like everything else")
	}

	text, err := exec.Command(binaryPath, "schema", "get", "exit-codes").Output()
	if err != nil {
		t.Fatalf("schema get exit-codes: %v", err)
	}
	if json.Valid(text) {
		t.Errorf("the default rendering should be text, got JSON:\n%s", text)
	}
	if !strings.Contains(string(text), "MEANING") {
		t.Errorf("the text rendering should be a table, got:\n%s", text)
	}

	asJSON, err := exec.Command(binaryPath, "schema", "get", "exit-codes", "--json").Output()
	if err != nil {
		t.Fatalf("schema get exit-codes --json: %v", err)
	}
	if !json.Valid(asJSON) {
		t.Errorf("--json should emit JSON, got:\n%s", asJSON)
	}

	// --silent stays: "does this schema exist" has an exit code for an answer.
	if err := exec.Command(binaryPath, "schema", "get", "exit-codes", "--silent").Run(); err != nil {
		t.Errorf("schema get --silent should succeed quietly: %v", err)
	}
	if err := exec.Command(binaryPath, "schema", "get", "nope", "--silent").Run(); err == nil {
		t.Error("schema get --silent on an unknown name should exit non-zero")
	}
}
