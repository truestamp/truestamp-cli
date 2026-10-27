// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// These tests plug the gaps the regular suite leaves uncovered, mostly
// subprocess exercises of `upgrade --check`, `auth status/login/logout`,
// and various convert-helper branches that only fire on specific inputs
// (raw extract, zone fallback, etc.).

// --- upgrade --check ---

// TestCLI_Upgrade_Check_UpgradeAvailable runs `upgrade --check --exit-code`
// against an httptest server standing in for the GitHub Releases API. The
// release endpoint is redirected at link time, as buildVerifyBinary does
// for the verifier's compiled-in addresses, so the shipped binary has no
// override.
func TestCLI_Upgrade_Check_UpgradeAvailable(t *testing.T) {
	const releasePath = "/repos/truestamp/truestamp-cli/releases/latest"
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != releasePath {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0","prerelease":false,"draft":false}`))
	}))
	defer srv.Close()

	bin := buildWithLdflags(t,
		"-X github.com/truestamp/truestamp-cli/internal/selfupgrade.LatestReleaseURL="+srv.URL+releasePath,
		"-X github.com/truestamp/truestamp-cli/internal/version.Version=1.0.0")
	cmd := exec.Command(bin, "upgrade", "--check", "--exit-code")
	cmd.Env = offlineEnv(t, offlineAPIKey)
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != checkExitUpgradeAvail {
		t.Fatalf("upgrade --check --exit-code: err = %v, want exit %d\n%s", err, checkExitUpgradeAvail, out)
	}
	if want := "truestamp 99.0.0 is available (current: 1.0.0)"; !strings.Contains(string(out), want) {
		t.Errorf("output %q does not contain %q", out, want)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("the release server saw %d requests, want 1", n)
	}
}

// buildWithLdflags compiles the CLI with the given -ldflags -X settings and
// returns the binary's path.
func buildWithLdflags(t *testing.T, settings ...string) string {
	t.Helper()
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatalf("locating module root: %v", err)
	}
	out := filepath.Join(t.TempDir(), "truestamp-ldflags")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-ldflags", strings.Join(settings, " "), "-o", out, "./cmd/truestamp")
	cmd.Dir = modRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the redirected binary: %v\n%s", err, b)
	}
	return out
}

// --- auth subcommands (an isolated HOME and a loopback --base-url) ---

func TestCLI_Auth_Help(t *testing.T) {
	for _, verb := range []string{"login", "logout", "status"} {
		out, err := exec.Command(binaryPath, "auth", verb, "--help").CombinedOutput()
		if err != nil {
			t.Errorf("auth %s --help failed: %s", verb, err)
		}
		if !strings.Contains(string(out), verb) {
			t.Errorf("auth %s --help should mention %q, got %q", verb, verb, out)
		}
	}
}

// countingAPI is a loopback API that answers 401 to every request and
// counts them, for the tests that must send none.
func countingAPI(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// TestCLI_Auth_Status_NoAPIKey: with no API key and no stored session, in
// an isolated HOME, `auth status` reports not_authenticated and sends no
// request.
func TestCLI_Auth_Status_NoAPIKey(t *testing.T) {
	srv, hits := countingAPI(t)
	cmd := exec.Command(binaryPath, "auth", "status", "--json", "--base-url", srv.URL)
	cmd.Env = offlineEnv(t, "")
	out, err := cmd.Output()
	if err == nil {
		t.Fatalf("auth status with no credential exited 0:\n%s", out)
	}
	var rec struct {
		OK     bool   `json:"ok"`
		Reason string `json:"reason"`
	}
	if jerr := json.Unmarshal(out, &rec); jerr != nil {
		t.Fatalf("auth status --json: %v\n%s", jerr, out)
	}
	if rec.OK || rec.Reason != "not_authenticated" {
		t.Errorf("auth status = %+v, want ok false and reason not_authenticated", rec)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("auth status with no credential sent %d requests, want 0", n)
	}
}

// TestCLI_Auth_Logout_NoAPIKey: with no API key and no stored session, in
// an isolated HOME, `auth logout` says so and sends no request.
func TestCLI_Auth_Logout_NoAPIKey(t *testing.T) {
	srv, hits := countingAPI(t)
	cmd := exec.Command(binaryPath, "auth", "logout", "--base-url", srv.URL)
	cmd.Env = offlineEnv(t, "")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("auth logout: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Not logged in.") {
		t.Errorf("auth logout output %q does not say Not logged in.", out)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("auth logout with no credential sent %d requests, want 0", n)
	}
}

// --- create: presentCreate + printCreateJSON ---

func TestCLI_ItemsCreate_WithMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"data":{"id":"01NEW","attributes":{
			"state":"new","claims":{"hash":"aa","hash_type":"sha256"},
			"display_name":"file.bin","visibility":"private",
			"tags":["a","b"],"team_id":"team-1"
		}}}`))
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "f.bin")
	_ = os.WriteFile(path, []byte("hello"), 0644)

	// Text output (presentCreate)
	cmd := exec.Command(binaryPath, "items", "create", path,
		"--base-url", srv.URL, "--api-key", "key")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create text: %s\n%s", err, out)
	}
	if !strings.Contains(string(out), "01NEW") {
		t.Errorf("expected new item id, got %q", out)
	}

	// JSON output (printCreateJSON)
	cmd = exec.Command(binaryPath, "items", "create", path, "--json",
		"--base-url", srv.URL, "--api-key", "key")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create json: %s\n%s", err, out)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out)
	}
	if m["id"] != "01NEW" {
		t.Errorf("id: %v", m["id"])
	}
	if m["tags"] == nil {
		t.Errorf("tags missing")
	}
}

// --- convert_id: raw extract + local zone ---

func TestCLI_ConvertID_RawUUID(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "id",
		"019cf813-99b8-730a-84f1-5a711a9c355e", "--extract", "raw")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(out))
	if len(got) != 32 {
		t.Errorf("raw UUID hex length: got %d, want 32", len(got))
	}
}

func TestCLI_ConvertID_RawUUID_JSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "id",
		"019cf813-99b8-730a-84f1-5a711a9c355e", "--extract", "raw", "--json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "uuid7" {
		t.Errorf("type: %v", m["type"])
	}
}

func TestCLI_ConvertID_Local(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "id",
		"019cf813-99b8-730a-84f1-5a711a9c355e", "--to-zone", "Local")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertID_BadZone(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "id",
		"019cf813-99b8-730a-84f1-5a711a9c355e", "--to-zone", "Not/A/Zone")
	if err := cmd.Run(); err == nil {
		t.Error("expected error for bad zone")
	}
}

func TestCLI_ConvertID_BadType(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "id", "anything", "--type", "foo")
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "--type") {
		t.Errorf("expected type error, got %q", out)
	}
}

// --- convert_time: exhaust the flag combinations ---

func TestCLI_ConvertTime_Now(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "time", "now")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertTime_UnixMs(t *testing.T) {
	// 13-digit input → unix-ms auto-detect.
	cmd := exec.Command(binaryPath, "convert", "time", "1700000000000")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(out)), "2023-11-") {
		t.Errorf("unexpected: %q", out)
	}
}

func TestCLI_ConvertTime_UnixUs(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "time", "1700000000000000")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertTime_UnixNs(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "time", "1700000000000000000")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertTime_FormatVariants(t *testing.T) {
	for _, f := range []string{"unix-s", "unix-ms", "unix-us", "unix-ns", "2006-01-02"} {
		cmd := exec.Command(binaryPath, "convert", "time", "1700000000", "--format", f)
		if err := cmd.Run(); err != nil {
			t.Errorf("--format %s: %v", f, err)
		}
	}
}

func TestCLI_ConvertTime_ExplicitFrom(t *testing.T) {
	for _, f := range []string{"rfc3339", "unix-s", "unix-ms", "unix-us", "unix-ns"} {
		val := "1700000000"
		if f == "rfc3339" {
			val = "2026-04-21T12:00:00Z"
		}
		cmd := exec.Command(binaryPath, "convert", "time", val, "--from", f)
		if err := cmd.Run(); err != nil {
			t.Errorf("--from %s: %v", f, err)
		}
	}
}

func TestCLI_ConvertTime_JSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "time", "1700000000", "--json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["unix_ms"] == nil {
		t.Errorf("expected unix_ms key")
	}
}

func TestCLI_ConvertTime_Stdin(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "time")
	cmd.Stdin = strings.NewReader("1700000000\n")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertTime_BadInput(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "time", "not-a-time")
	if err := cmd.Run(); err == nil {
		t.Error("expected error")
	}
}

// --- convert_keyid: auto-detect branches ---

func TestCLI_ConvertKeyID_Hex(t *testing.T) {
	// Ed25519 pubkey encoded as 64 hex chars.
	hex64 := strings.Repeat("ab", 32)
	cmd := exec.Command(binaryPath, "convert", "keyid", hex64)
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertKeyID_Base64URL(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	b64url := base64.RawURLEncoding.EncodeToString(raw) // 43 chars
	cmd := exec.Command(binaryPath, "convert", "keyid", b64url)
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertKeyID_Stdin(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "keyid")
	cmd.Stdin = strings.NewReader("CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertKeyID_WrongLength(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "keyid", "tooshort")
	if err := cmd.Run(); err == nil {
		t.Error("expected error for short key")
	}
}

func TestCLI_ConvertKeyID_ExplicitFrom(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "keyid",
		"CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=", "--from", "base64")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

// --- jcs --newline / --json ---

func TestCLI_JCS_Newline(t *testing.T) {
	cmd := exec.Command(binaryPath, "jcs", "--newline")
	cmd.Stdin = strings.NewReader(`{"b":2,"a":1}`)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(out), "\n") {
		t.Errorf("--newline should append \\n, got %q", out)
	}
}

func TestCLI_JCS_JSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "jcs", "--json")
	cmd.Stdin = strings.NewReader(`{"b":2,"a":1}`)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["output"] == nil {
		t.Error("expected output key in JSON envelope")
	}
}

func TestCLI_JCS_Silent(t *testing.T) {
	cmd := exec.Command(binaryPath, "jcs", "--silent")
	cmd.Stdin = strings.NewReader(`{"a":1}`)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("--silent should emit nothing, got %q", out)
	}
}

// --- encode/decode: JSON envelope branches ---

func TestCLI_Encode_BinaryOutput_JSON(t *testing.T) {
	// --to binary + --json should wrap the bytes in base64url under
	// output_wrapped (not output).
	cmd := exec.Command(binaryPath, "encode", "--to", "binary", "--json")
	cmd.Stdin = strings.NewReader("bytes")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["output_wrapped"] == nil {
		t.Errorf("expected output_wrapped for binary output: %+v", m)
	}
}

// --- hash --file picker sentinel: no-op TTY path ---

func TestCLI_Hash_BadPrefix(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "--prefix", "not-hex")
	cmd.Stdin = strings.NewReader("x")
	if err := cmd.Run(); err == nil {
		t.Error("expected error for invalid --prefix")
	}
}

func TestCLI_Hash_BadEncoding(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "--encoding", "binary")
	cmd.Stdin = strings.NewReader("x")
	if err := cmd.Run(); err == nil {
		t.Error("binary encoding on --encoding should be rejected")
	}
}

func TestCLI_Hash_BadStyle(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "--style", "ascii-art")
	cmd.Stdin = strings.NewReader("x")
	if err := cmd.Run(); err == nil {
		t.Error("unknown --style should be rejected")
	}
}

func TestCLI_Hash_NoFileOrStdin_ShowsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "Usage") {
		t.Errorf("expected help output, got %q", out)
	}
}

func TestCLI_Hash_MissingFile(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "/definitely/nonexistent/file")
	if err := cmd.Run(); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestCLI_Hash_FileAndPositional_Conflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x")
	_ = os.WriteFile(path, []byte("x"), 0644)
	cmd := exec.Command(binaryPath, "hash", "--file", path, "other-path")
	if err := cmd.Run(); err == nil {
		t.Error("expected conflict error")
	}
}

// --- verify JSON output path ---

// verify --json writes the report of SPEC.md section 10.5, with its
// label and rows and no member "passed", which is internal (section 10.5).
// Mutant killed: a "passed" member added to the report.
func TestCLI_Verify_JSON(t *testing.T) {
	out, _, code := verifyCLI(t, binaryPath, nil, nil, append([]string{soundItem(), "--json"}, underTest()...)...)
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	if _, passed := m["passed"]; code != 6 || m["label"] != "supplied-keyring" || m["rows"] == nil || passed {
		t.Errorf("exit %d, label/rows missing or passed present: %+v", code, m)
	}
}

// --- version subcommand ---

func TestCLI_Version_Subcommand(t *testing.T) {
	cmd := exec.Command(binaryPath, "version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "truestamp") {
		t.Errorf("expected 'truestamp' in version output, got %q", out)
	}
}

// --- config init + path + show ---

func TestCLI_Config_Init(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(binaryPath, "config", "init")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir, "HOME="+dir)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "truestamp", "config.toml")); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

// --- upgrade --help (exercise upgradeInstructionFor, etc.) ---

func TestCLI_Upgrade_Help(t *testing.T) {
	out, err := exec.Command(binaryPath, "upgrade", "--help").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "upgrade") {
		t.Errorf("expected help content, got %q", out)
	}
}

// --- In-process unit tests for small utility functions ---

func TestExitWith_ZeroReturnsNil(t *testing.T) {
	if err := exitWith(0); err != nil {
		t.Errorf("exitWith(0) should return nil, got %v", err)
	}
}

func TestExitWith_NonZeroReturnsExitCodeErr(t *testing.T) {
	err := exitWith(3)
	if err == nil {
		t.Fatal("exitWith(3) should return non-nil")
	}
	if got := ExitCode(err); got != 3 {
		t.Errorf("ExitCode: got %d, want 3", got)
	}
}

func TestExitCodeErr_ErrorMessage(t *testing.T) {
	err := exitCodeErr{code: 7}
	if err.Error() == "" {
		t.Error("Error() should produce a non-empty message")
	}
	if !strings.Contains(err.Error(), "7") {
		t.Errorf("Error() should mention the code, got %q", err.Error())
	}
}

func TestExitCode_NilError(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Errorf("ExitCode(nil): got %d, want 0", got)
	}
}

func TestExitCode_GenericError(t *testing.T) {
	err := errors.New("boom")
	if got := ExitCode(err); got != 1 {
		t.Errorf("ExitCode(generic): got %d, want 1", got)
	}
}

func TestStdinIsTerminal_DoesNotPanic(t *testing.T) {
	// The actual return value depends on the test env; we just exercise
	// the function to make sure it doesn't panic.
	_ = inputsrc.IsStdinTerminal()
}

func TestPrintAlgorithmList(t *testing.T) {
	var buf strings.Builder
	printAlgorithmList(&buf)
	out := buf.String()
	if !strings.Contains(out, "sha256") || !strings.Contains(out, "blake2b-512") {
		t.Errorf("missing algorithms: %q", out)
	}
}

func TestParsePrefixByte(t *testing.T) {
	cases := []struct {
		in   string
		want byte
		ok   bool
	}{
		{"0x11", 0x11, true},
		{"11", 0x11, true},
		{"ff", 0xff, true},
		{"0X1A", 0x1a, true},
		{"0", 0x00, true},
		{"0x0", 0x00, true},
		{"", 0, false},
		{"zz", 0, false},
		{"abc", 0, false}, // 3 hex digits too long
	}
	for _, c := range cases {
		got, err := parsePrefixByte(c.in)
		if c.ok {
			if err != nil || got != c.want {
				t.Errorf("parsePrefixByte(%q) = (0x%02x, %v), want (0x%02x, nil)",
					c.in, got, err, c.want)
			}
		} else if err == nil {
			t.Errorf("parsePrefixByte(%q) should error", c.in)
		}
	}
}
