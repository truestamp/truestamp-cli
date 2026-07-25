// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// These tests build the actual binary and run it as a subprocess
// to verify exit codes and quiet mode behavior. This is the idiomatic
// Go approach for testing CLI tools that call os.Exit.

var (
	binaryPath         string
	subprocessCoverDir string // set during TestMain when coverage is requested
)

func TestMain(m *testing.M) {
	// Build the binary once for all tests in this package
	tmp, err := os.MkdirTemp("", "truestamp-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "truestamp")

	// Find the module root (where go.mod lives) relative to this test file
	modRoot, err := findModuleRoot()
	if err != nil {
		panic("cannot find module root: " + err.Error())
	}

	// When TRUESTAMP_COVERDIR is set (task test-coverage-full sets it to
	// coverage/bin), build the subprocess binary with -cover so its
	// runtime coverage is recorded there. We use our own env var rather
	// than GOCOVERDIR because `go test -cover` overwrites GOCOVERDIR in
	// the test process's env with its own temp dir — the subprocess
	// would then write to that temp dir and lose the data when go test
	// cleans it up.
	buildArgs := []string{"build"}
	subprocessCoverDir = os.Getenv("TRUESTAMP_COVERDIR")
	if subprocessCoverDir != "" {
		buildArgs = append(buildArgs, "-cover", "-coverpkg=./...")
		// Force the subprocess binary (and implicitly every child of
		// this test process) to write to our stable covdata dir.
		// go test still writes its own test-process covdata to a
		// different temp dir via -test.gocoverdir, which our task
		// merges separately.
		_ = os.Setenv("GOCOVERDIR", subprocessCoverDir)
	}
	buildArgs = append(buildArgs, "-o", binaryPath, "./cmd/truestamp")
	cmd := exec.Command("go", buildArgs...)
	cmd.Dir = modRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("failed to build binary: " + err.Error() + "\n" + string(out))
	}

	os.Exit(m.Run())
}

// findModuleRoot walks up from the current working directory to find go.mod.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func writeProofFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proof.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Minimal structurally valid proof with fake crypto values.
// Hex hashes are 64 chars (SHA-256), key IDs are 8 chars, base64 public_key is
// 32 bytes (44 chars), base64 signature is 64 bytes (88 chars).
const fakeProofJSON = `{
  "v": 1,
  "t": 20,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "ts": "2026-04-06T23:25:06Z",
  "s": {
    "id": "01HJHB01T8FYZ7YTR9P5N62K5B",
    "d": {"name": "test", "hash": "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd"},
    "mh": "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd",
    "kid": "4ceefa4a"
  },
  "b": {
    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
    "ph": "1111111111111111111111111111111111111111111111111111111111111111",
    "mr": "2222222222222222222222222222222222222222222222222222222222222222",
    "mh": "4444444444444444444444444444444444444444444444444444444444444444",
    "kid": "4ceefa4a"
  },
  "ip": "AA",
  "cx": [{"t": 40, "net": "testnet", "tx": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "l": 1, "ep": "AA"}]
}`

func TestCLI_MissingFile_ExitCode1(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", "/nonexistent/proof.json", "--skip-external")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for missing file")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %s", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code: got %d, want 1", exitErr.ExitCode())
	}
}

func TestCLI_FakeCrypto_ExitCode1(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for fake crypto")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %s", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code: got %d, want 1", exitErr.ExitCode())
	}
}

func TestCLI_Silent_NoOutput_MissingFile(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", "/nonexistent/proof.json", "--silent")
	out, _ := cmd.CombinedOutput()
	if len(out) != 0 {
		t.Errorf("silent mode should produce no output, got: %q", string(out))
	}
}

func TestCLI_Silent_NoOutput_FakeCrypto(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--silent")
	out, _ := cmd.CombinedOutput()
	if len(out) != 0 {
		t.Errorf("silent mode should produce no output, got: %q", string(out))
	}
}

func TestCLI_Silent_ExitCode1_MissingFile(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", "/nonexistent/proof.json", "--silent")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code: got %d, want 1", exitErr.ExitCode())
	}
}

func TestCLI_Normal_ProducesOutput(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external")
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Error("normal mode should produce output")
	}
}

func TestCLI_Default_ProducesOutput(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--skip-signatures")
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Error("default mode should produce output")
	}
}

func TestCLI_SkipSignatures_SkipsSignatureChecks(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--skip-signatures")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !containsString(output, "skipped") {
		t.Error("--skip-signatures should show skipped count in summary")
	}
}

func TestCLI_SkipSignatures_Silent_ExitCode(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--skip-signatures", "--silent")
	out, _ := cmd.CombinedOutput()
	if len(out) != 0 {
		t.Errorf("silent + skip-signatures should produce no output, got: %q", string(out))
	}
}

// Minimal entropy proof for testing --hash rejection.
const fakeEntropyProofJSON = `{
  "v": 1,
  "t": 30,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "ts": "2026-04-06T23:25:06Z",
  "s": {
    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
    "d": {"pulse": {"pulseIndex": 123}},
    "mh": "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd",
    "kid": "4ceefa4a"
  },
  "b": {
    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
    "ph": "1111111111111111111111111111111111111111111111111111111111111111",
    "mr": "2222222222222222222222222222222222222222222222222222222222222222",
    "mh": "4444444444444444444444444444444444444444444444444444444444444444",
    "kid": "4ceefa4a"
  },
  "ip": "AA",
  "cx": [{"t": 40, "net": "testnet", "tx": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "l": 1, "ep": "AA"}]
}`

// TestCLI_Hash_WithEntropyProof_Skips pins Appendix E.7 at the CLI
// boundary: an expected hash supplied for a non-item subject is reported
// as a visible skip and contributes no failure. The previous expectation
// (exit 1 with "not applicable to entropy") passed for two reasons at
// once — this fixture's Merkle values are fabricated, so it fails
// regardless — which is why the assertions below name the Hash
// Comparison row specifically rather than reading the exit code.
func TestCLI_Hash_WithEntropyProof_Skips(t *testing.T) {
	path := writeProofFile(t, fakeEntropyProofJSON)
	cmd := exec.Command(binaryPath, "verify", path,
		"--skip-external", "--skip-signatures", "--hash", "deadbeef", "--json")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	if !containsString(output, `"group": "Hash Comparison"`) ||
		!containsString(output, "--hash not applicable") {
		t.Errorf("expected a Hash Comparison row explaining that --hash does not apply, got: %s", output)
	}
	if containsString(output, "not applicable to entropy") {
		t.Errorf("the old fail wording should be gone, got: %s", output)
	}
	// E.22 requires "the caller supplied a hash" to stay readable
	// separately from "the hash matched": the caller DID supply one here,
	// and reporting supplied=false would erase the fact that a comparison
	// was requested and could not be performed.
	if !containsString(output, `"supplied": true`) {
		t.Errorf("a --hash was supplied and E.22 requires that to be readable, got: %s", output)
	}
	// The failures this fixture does carry are its fabricated Merkle
	// values; the Hash Comparison row must be a skip, not one of them.
	if !containsString(output, `"status": "skip",
      "message": "--hash not applicable`) {
		t.Errorf("the Hash Comparison row must be a skip, got: %s", output)
	}
}

func TestCLI_FailedProof_NormalMode_ExitCode1(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for failed proof in normal mode")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %s", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code: got %d, want 1", exitErr.ExitCode())
	}
	// Should still produce output in normal mode
	if len(out) == 0 {
		t.Error("normal mode should produce output even on failure")
	}
}

func TestCLI_ConfigPath(t *testing.T) {
	cmd := exec.Command(binaryPath, "config", "path")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config path failed: %s\n%s", err, out)
	}
	if len(out) == 0 {
		t.Error("config path should produce output")
	}
}

func TestCLI_ConfigShow(t *testing.T) {
	cmd := exec.Command(binaryPath, "config", "show")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config show failed: %s\n%s", err, out)
	}
	output := string(out)
	if !containsString(output, "API URL") {
		t.Error("config show should include API URL")
	}
	if !containsString(output, "Verification") {
		t.Error("config show should include Verification section")
	}
}

func TestCLI_EnvVarOverride(t *testing.T) {
	cmd := exec.Command(binaryPath, "config", "show")
	cmd.Env = append(os.Environ(), "TRUESTAMP_BASE_URL=https://custom.example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config show failed: %s\n%s", err, out)
	}
	output := string(out)
	if !containsString(output, "https://custom.example.com") {
		t.Errorf("env var override not reflected in config show output: %s", output)
	}
}

func TestCLI_FlagOverridesEnv(t *testing.T) {
	cmd := exec.Command(binaryPath, "config", "show", "--base-url=https://flag.example.com")
	cmd.Env = append(os.Environ(), "TRUESTAMP_BASE_URL=https://env.example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config show failed: %s\n%s", err, out)
	}
	output := string(out)
	if !containsString(output, "https://flag.example.com") {
		t.Errorf("CLI flag should override env var, got: %s", output)
	}
}

func TestCLI_NoArgs_ShowsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("verify with no args should exit 0, got error: %s", err)
	}
	output := string(out)
	if !containsString(output, "Usage") {
		t.Error("no-args output should contain Usage")
	}
}

func TestCLI_FileFlag_Works(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", "--file="+path, "--skip-external")
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Error("--file flag should produce output")
	}
}

func TestCLI_FileFlag_MissingFile(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", "--file=/nonexistent/proof.json")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for missing file via --file")
	}
}

func TestCLI_StdinPipe(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", "--skip-external")
	cmd.Stdin = strings.NewReader(fakeProofJSON)
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Error("stdin pipe should produce output")
	}
}

func TestCLI_StdinPipe_InvalidJSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", "--skip-external")
	cmd.Stdin = strings.NewReader("not json at all")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for invalid JSON on stdin")
	}
}

func TestCLI_HashFlag_Match(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	claimsHash := "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd"
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--hash="+claimsHash, "--skip-signatures")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	// Crypto will fail (fake values) but hash should show as verified in Proof section
	if !containsString(output, "verified") {
		t.Error("correct --hash should show hash as verified in Proof section")
	}
}

func TestCLI_HashFlag_InvalidHex(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--hash=xyz")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for invalid hex in --hash")
	}
}

func TestCLI_HashFlag_OddLength(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--hash=abc")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for odd-length hex in --hash")
	}
}

func TestCLI_NoHash_ShowsGuidance(t *testing.T) {
	// fakeProofJSON has bad crypto so it hits VERIFICATION FAILED, not the guidance.
	// The guidance only shows when proof passes without --hash.
	// Instead, verify that the no-hash banner doesn't say "FULLY VERIFIED"
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if containsString(output, "FULLY VERIFIED") {
		t.Error("output without --hash should never say FULLY VERIFIED")
	}
}

// --- Completion tests --------------------------------------------------------
// These tests guard against regressions that cause shell completion to hang.
// Root cause: lipgloss v2's compat package queries the terminal at package init
// time (HasDarkBackground). Inside source <(cmd), the process runs in a
// background group where reading from the terminal triggers SIGTTIN, stopping
// the process. The fix has two layers:
//   1. ui.go defers terminal detection to Init() (not package init)
//   2. root.go skips PersistentPreRunE for completion/help via os.Args check

func TestCLI_CompletionZsh_Succeeds(t *testing.T) {
	cmd := exec.Command(binaryPath, "completion", "zsh")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("completion zsh failed: %s\n%s", err, out)
	}
	output := string(out)
	if !containsString(output, "#compdef truestamp") {
		t.Error("completion zsh should contain #compdef header")
	}
	if !containsString(output, "_truestamp") {
		t.Error("completion zsh should define _truestamp function")
	}
}

func TestCLI_CompletionBash_Succeeds(t *testing.T) {
	cmd := exec.Command(binaryPath, "completion", "bash")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("completion bash failed: %s\n%s", err, out)
	}
	if len(out) == 0 {
		t.Error("completion bash should produce output")
	}
}

func TestCLI_DynamicCompletion_Succeeds(t *testing.T) {
	// Simulates what the sourced zsh script does: calls the binary with
	// __complete to get completions for a partial command. This must
	// complete without hanging (no terminal queries, no config loading).
	cmd := exec.Command(binaryPath, "__complete", "verify", "")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("__complete failed: %s\n%s", err, out)
	}
	output := string(out)
	// Cobra outputs completions followed by a directive line (:<int>)
	if !containsString(output, ":") {
		t.Error("__complete should output completions with a directive suffix")
	}
}

func TestCLI_CompletionZsh_NoStderr(t *testing.T) {
	// Completion must not write to stderr (would interfere with shell setup).
	cmd := exec.Command(binaryPath, "completion", "zsh")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("completion zsh failed: %s", err)
	}
	if stderr.Len() > 0 {
		t.Errorf("completion zsh should produce no stderr, got: %q", stderr.String())
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && indexOf(haystack, needle) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// --- Subject-type assertion (--type flag) -----------------------------------

// TestCLI_Verify_TypeMatches_NoAssertionFailure: --type matches the bundle's
// t (both item). The fake proof fails other checks (fake sig/hashes), but the
// Subject Type assertion should NOT contribute a failure of its own.
func TestCLI_Verify_TypeMatches_NoAssertionFailure(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", "--type", "item", "--skip-external", "--skip-signatures", path)
	cmd.Env = cleanEnv()
	out, _ := cmd.CombinedOutput()
	stripped := stripANSI(string(out))
	if strings.Contains(stripped, "--type item was requested") {
		t.Errorf("--type should not surface assertion failure when it matches, got: %s", stripped)
	}
	if strings.Contains(stripped, "Proof is item (t=20) but") {
		t.Errorf("unexpected Subject Type mismatch message when types match: %s", stripped)
	}
}

// TestCLI_Verify_TypeMismatch_Fails: --type beacon on an item proof should
// exit non-zero. The report still renders; the Subject Type group shows fail.
func TestCLI_Verify_TypeMismatch_Fails(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", "--type", "beacon", "--skip-external", "--skip-signatures", path)
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit on type mismatch")
	}
	// The mismatch renders as a failing step inside the report:
	//   x Proof is item (t=20) but --type beacon was requested
	stripped := stripANSI(string(out))
	if !strings.Contains(stripped, "Proof is item") {
		t.Errorf("expected 'Proof is item' in failure message, got: %s", stripped)
	}
	if !strings.Contains(stripped, "--type beacon was requested") {
		t.Errorf("expected '--type beacon was requested' in failure message, got: %s", stripped)
	}
}

// TestCLI_Verify_Type_InvalidEnum: --type with a value outside the six-value
// enum is rejected client-side before the file is even parsed.
func TestCLI_Verify_Type_InvalidEnum(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", "--type", "auto", path)
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit on --type=auto")
	}
	if !strings.Contains(string(out), "--type must be one of") {
		t.Errorf("expected enum validation message, got: %s", out)
	}
}

// TestCLI_Verify_Type_BareEntropy_Rejected: bare "entropy" is not in the
// six-value enum (must be entropy_nist / entropy_stellar / entropy_bitcoin).
func TestCLI_Verify_Type_BareEntropy_Rejected(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", "--type", "entropy", path)
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit on --type=entropy")
	}
	if !strings.Contains(string(out), "--type must be one of") {
		t.Errorf("expected enum validation message, got: %s", out)
	}
}

// --- Structured hard rejections (Appendix E.23) -----------------------------

// TestCLI_Verify_JSON_StructuredRejection: an Appendix E.6 hard rejection
// aborts before any step runs, so there is no report to render. Under --json
// the E.23 identifier has to reach the caller as data — the entire purpose of
// the taxonomy is that two independent verifiers can be compared on the
// identifier rather than on an English sentence that may differ between them.
func TestCLI_Verify_JSON_StructuredRejection(t *testing.T) {
	cases := []struct {
		name     string
		bundle   string
		wantCode string
	}{
		{"array", `[1,2]`, "not_a_json_object"},
		{"null", `null`, "not_a_json_object"},
		{"unknown type code", `{"v":1,"t":99}`, "invalid_subject_type_code"},
		{"missing type code", `{"v":1}`, "missing_type_code"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := writeProofFile(t, c.bundle)
			cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--json")
			cmd.Env = cleanEnv()
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err == nil {
				t.Fatalf("expected a non-zero exit for a hard rejection, got 0: %s", stdout.String())
			}
			var parsed struct {
				Result    string `json:"result"`
				Rejection struct {
					Code   string `json:"code"`
					Detail string `json:"detail"`
				} `json:"rejection"`
			}
			if jErr := json.Unmarshal(stdout.Bytes(), &parsed); jErr != nil {
				t.Fatalf("--json output is not JSON: %v\n%s", jErr, stdout.String())
			}
			if parsed.Result != "rejected" {
				t.Errorf("result: got %q, want \"rejected\"", parsed.Result)
			}
			if parsed.Rejection.Code != c.wantCode {
				t.Errorf("rejection.code: got %q, want %q", parsed.Rejection.Code, c.wantCode)
			}
			if parsed.Rejection.Detail == "" {
				t.Error("rejection.detail must explain the refusal")
			}
			// The code has its own field; repeating it inside the detail
			// would make the two disagree the moment either is reworded.
			if strings.HasPrefix(parsed.Rejection.Detail, c.wantCode+":") {
				t.Errorf("detail duplicates the code: %q", parsed.Rejection.Detail)
			}
			// A second, English copy on stderr would defeat the point.
			if stderr.Len() != 0 {
				t.Errorf("stderr should stay empty under --json, got %q", stderr.String())
			}
		})
	}
}

// TestCLI_Verify_Rejection_TextModeUnchanged: only the --json surface gained
// structure. Human output keeps the same prose it always emitted.
func TestCLI_Verify_Rejection_TextModeUnchanged(t *testing.T) {
	path := writeProofFile(t, `[1,2]`)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external")
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(string(out), "not a JSON object") {
		t.Errorf("expected the human-readable rejection, got: %s", out)
	}
}

// --- The filename never decides the subject type (Appendix E.24) ------------

// samplePath resolves a file under the repo's samples/ directory. Tests run
// with cmd/ as the working directory, so the module root has to be found
// rather than assumed.
func samplePath(t *testing.T, name string) string {
	t.Helper()
	root, err := findModuleRoot()
	if err != nil {
		t.Fatalf("locating module root: %v", err)
	}
	return filepath.Join(root, "samples", name)
}

// TestCLI_Verify_FilenameNeverAffectsVerdict is the regression test for the
// filename-derived --type inference this CLI used to apply. Appendix E.24
// requires the subject type to come from the bundle's signed `t` and never
// from the downloaded filename, and the inference broke that both ways: a
// conforming stem that disagreed with `t` failed a sound proof, and renaming
// away from a conforming stem silently dropped the assertion. The invariant
// is that byte-identical input produces a byte-identical verdict regardless
// of what the file is called.
func TestCLI_Verify_FilenameNeverAffectsVerdict(t *testing.T) {
	dir := t.TempDir()
	names := []string{
		"truestamp-beacon-019db702-b08c-73dc-a7cd-2c5e011f1dad.json",
		"truestamp-block-019db702-b08c-73dc-a7cd-2c5e011f1dad.json",
		"truestamp-entropy-nist-019db702-b08c-73dc-a7cd-2c5e011f1dad.json",
		"proof.json",
	}
	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(fakeProofJSON), 0644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--skip-signatures")
		cmd.Env = cleanEnv()
		out, _ := cmd.CombinedOutput()
		stripped := stripANSI(string(out))
		if strings.Contains(stripped, "Subject Type") || strings.Contains(stripped, "but --type") {
			t.Errorf("%s: filename produced a Subject Type assertion with no --type flag: %s", name, stripped)
		}
		if strings.Contains(stripped, "inferred --type") {
			t.Errorf("%s: filename inference hint still present: %s", name, stripped)
		}
	}
}

// TestCLI_Verify_ShippedBeaconSampleVerifies pins Appendix E.24's own named
// scenario against the artifact this repo ships: samples/truestamp-beacon-*
// carries t=10 (a plain block proof), exactly the case E.24 calls legitimate
// because the beacon show page names a block proof that way. It used to exit
// 1 on a "Subject Type" failure the user never asked for.
func TestCLI_Verify_ShippedBeaconSampleVerifies(t *testing.T) {
	for _, ext := range []string{".json", ".cbor"} {
		t.Run(ext, func(t *testing.T) {
			path := samplePath(t, "truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503"+ext)
			cmd := exec.Command(binaryPath, "verify", path, "--skip-external")
			cmd.Env = cleanEnv()
			out, err := cmd.CombinedOutput()
			stripped := stripANSI(string(out))
			if err != nil {
				t.Fatalf("expected exit 0 on the shipped beacon-named block proof, got %v: %s", err, stripped)
			}
			if !strings.Contains(stripped, "Block") {
				t.Errorf("expected the report to name the bundle's own type (Block), got: %s", stripped)
			}
		})
	}
}

// TestCLI_Verify_JSON_NoSubjectTypeIssueWithoutFlag is the machine-readable
// half of the same guarantee: no --type flag means no structural issue about
// one, and the run reports as passed.
func TestCLI_Verify_JSON_NoSubjectTypeIssueWithoutFlag(t *testing.T) {
	path := samplePath(t, "truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503.json")
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--json")
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got %v: %s", err, out)
	}
	var parsed struct {
		Result string `json:"result"`
		Issues []struct {
			Category string `json:"category"`
			Message  string `json:"message"`
		} `json:"issues"`
	}
	if jErr := json.Unmarshal(out, &parsed); jErr != nil {
		t.Fatalf("parsing --json output: %v\n%s", jErr, out)
	}
	if parsed.Result != "verified" && parsed.Result != "fully_verified" {
		t.Errorf("result: got %q, want a passing verdict", parsed.Result)
	}
	for _, issue := range parsed.Issues {
		if strings.Contains(issue.Message, "--type") {
			t.Errorf("unexpected --type issue with no --type flag: %+v", issue)
		}
	}
}

// TestCLI_Verify_ExplicitTypeStillAsserts guards against over-correction:
// only the inference was removed, not the flag. A user who asks for the
// assertion still gets it.
func TestCLI_Verify_ExplicitTypeStillAsserts(t *testing.T) {
	path := samplePath(t, "truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503.json")
	cmd := exec.Command(binaryPath, "verify", path, "--type", "beacon", "--skip-external")
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit when the explicitly asserted type disagrees with t")
	}
	stripped := stripANSI(string(out))
	if !strings.Contains(stripped, "Proof is block (t=10) but --type beacon was requested") {
		t.Errorf("expected the Subject Type assertion failure, got: %s", stripped)
	}
}

// TestCLI_Verify_Type_ExplicitMatchNoFailure: an explicit --type that agrees
// with the bundle surfaces no assertion failure, whatever the file is named.
func TestCLI_Verify_Type_ExplicitMatchNoFailure(t *testing.T) {
	dir := t.TempDir()
	beaconNamed := filepath.Join(dir, "truestamp-beacon-019db702.json")
	if err := os.WriteFile(beaconNamed, []byte(fakeProofJSON), 0644); err != nil {
		t.Fatal(err)
	}
	// fakeProofJSON is t=20 item; --type item agrees with it.
	cmd := exec.Command(binaryPath, "verify", "--type", "item", "--skip-external", "--skip-signatures", beaconNamed)
	cmd.Env = cleanEnv()
	out, _ := cmd.CombinedOutput()
	stripped := stripANSI(string(out))
	if strings.Contains(stripped, "Proof is item (t=20) but --type") {
		t.Errorf("matching --type should not surface assertion failure: %s", stripped)
	}
}

// TestCLI_Verify_Remote_NoTypePostedWithoutFlag pins the other half of the
// blast radius: the filename-derived type used to reach the server as
// `data.type` on --remote, where a mismatch came back as an opaque API error
// with no report at all. With no --type flag the key must be absent.
func TestCLI_Verify_Remote_NoTypePostedWithoutFlag(t *testing.T) {
	var mu sync.Mutex
	var bodies []map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Data map[string]any `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		mu.Lock()
		bodies = append(bodies, payload.Data)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		// A minimal verified report is enough — the assertion is on the
		// request, not the response.
		_, _ = w.Write([]byte(`{"data":{"verified":true,"subject_type":"block","steps":[]}}`))
	}))
	defer srv.Close()

	path := samplePath(t, "truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503.json")
	run := func(extra ...string) {
		args := append([]string{"verify", path, "--remote",
			"--base-url", srv.URL, "--api-key", "test-key"}, extra...)
		cmd := exec.Command(binaryPath, args...)
		cmd.Env = cleanEnv()
		_, _ = cmd.CombinedOutput()
	}
	run()
	// The fixture is a t=10 block despite its beacon filename, so "block"
	// is the type it actually carries and the one that must be forwarded.
	run("--type", "block")
	// A MISMATCHING type is asserted client-side and deliberately not
	// forwarded: posting it makes the server answer 4xx and the caller
	// gets no report at all, which is the condition --type exists to
	// detect.
	run("--type", "beacon")

	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 3 {
		t.Fatalf("expected 3 captured requests, got %d", len(bodies))
	}
	if _, ok := bodies[0]["type"]; ok {
		t.Errorf("no --type flag was passed, yet data.type was posted: %v", bodies[0])
	}
	if got := bodies[1]["type"]; got != "block" {
		t.Errorf("explicit --type block: data.type = %v, want \"block\"", got)
	}
	if got, ok := bodies[2]["type"]; ok {
		t.Errorf("mismatching --type beacon must not be forwarded, got data.type = %v", got)
	}
}

// stripANSI removes ANSI escape sequences from output so tests don't have to
// deal with lipgloss color codes when checking text content.
func stripANSI(s string) string {
	out := make([]byte, 0, len(s))
	skip := false
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == 0x1b {
			skip = true
			continue
		}
		if skip {
			if b == 'm' {
				skip = false
			}
			continue
		}
		out = append(out, b)
	}
	return string(out)
}

// conformanceVectorPath locates the vendored Appendix D.1 bundle. It lives
// under internal/verify/testdata so the conformance suite that owns it and
// this CLI-level check assert against the same bytes.
func conformanceVectorPath(t *testing.T) string {
	t.Helper()
	root, err := findModuleRoot()
	if err != nil {
		t.Fatalf("locating module root: %v", err)
	}
	return filepath.Join(root, "internal", "verify", "testdata", "fixtures", "appendix-d-item.json")
}

// TestCLI_AppendixD_Conformance is Appendix E.25's self-certification at the
// process boundary: the published worked example, verified offline against
// its own claims hash, must exit 0 and report every Appendix D.4 row with
// D.4's status.
//
// internal/verify's TestAppendixD4_Conformance asserts the same containment
// against the in-process report. This one exists because the exit code and
// the --json surface are what a caller actually consumes, and either can
// diverge from the report without the library test noticing.
func TestCLI_AppendixD_Conformance(t *testing.T) {
	out := runConformanceVector(t, cleanEnv())
	if out.Result != "fully_verified" {
		t.Errorf("result: got %q, want %q", out.Result, "fully_verified")
	}
	for _, v := range d4CLIViolations(out.Steps) {
		t.Error(v)
	}
}

// TestCLI_AppendixD_Conformance_ExitCodeIsNotTheAcceptanceCriterion pins
// the reason the test above sets cleanEnv() and the reason E.25's
// acceptance is a containment check on the ROWS rather than on the exit
// code.
//
// `verify`'s flags are also config keys: an ambient
// TRUESTAMP_VERIFY_SKIP_SIGNATURES, or a `skip_signatures = true` in a
// config.toml the caller forgot about, turns D.4's Proof Signature pass
// into a skip. The run still exits 0 and still reports fully_verified,
// because a skip cannot move a verdict — so a conformance procedure that
// reads the exit code accepts a run in which E.16 was never performed.
//
// Both halves are asserted: that the run really does still exit 0 with
// fully_verified (the hazard is real, and if that ever changes this test
// should be revisited rather than deleted), and that the containment
// check nonetheless refuses it (the acceptance criterion is sound).
func TestCLI_AppendixD_Conformance_ExitCodeIsNotTheAcceptanceCriterion(t *testing.T) {
	configDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(configDir, "truestamp"), 0o755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "truestamp", "config.toml"),
		[]byte("[verify]\nskip_signatures = true\n"), 0o600); err != nil {
		t.Fatalf("writing config.toml: %v", err)
	}

	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"ambient env var", append(cleanEnv(), "TRUESTAMP_VERIFY_SKIP_SIGNATURES=1")},
		{"ambient config.toml", replaceEnv(cleanEnv(), "XDG_CONFIG_HOME", configDir)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := runConformanceVector(t, tc.env)

			// The hazard: exit 0 (runConformanceVector fatals otherwise)
			// and a verdict that reads as fully verified.
			if out.Result != "fully_verified" {
				t.Logf("note: result is now %q, not fully_verified — the hazard this test documents may have been closed at the verdict level", out.Result)
			}
			// The defence: D.4's Proof Signature row is a pass, and this
			// run does not have one.
			violations := d4CLIViolations(out.Steps)
			if len(violations) == 0 {
				t.Fatalf("a run that skipped the signature satisfied D.4 containment; steps:\n%s", formatCLISteps(out.Steps))
			}
			var named bool
			for _, v := range violations {
				if strings.Contains(v, "Proof Signature") {
					named = true
				}
			}
			if !named {
				t.Errorf("containment failed, but not on Proof Signature: %v", violations)
			}
		})
	}
}

// TestCLI_SkipSignatures_HelpMatchesBehaviour ties the flag's one-line
// description to what the flag actually suppresses.
//
// Appendix E.9's Signing Key row is a local decode of `pk` plus the kid
// derivation; it runs and passes under --skip-signatures by design, and
// D.4's containment depends on that (the row is a pass there too). What
// the flag suppresses is E.16's Ed25519 verification and E.17's keyring
// cross-check. The help text used to say "Skip signing key and signature
// verification", which named a row the same run visibly passes — a
// caller reading it would conclude the reported Signing Key pass was
// stale output rather than a check that ran.
//
// Both halves are asserted together on purpose: pinning the string alone
// is a golden test (there is one), and pinning the behaviour alone
// leaves the string free to drift back. The contradiction is only
// visible when the two are read in one place.
func TestCLI_SkipSignatures_HelpMatchesBehaviour(t *testing.T) {
	// Half one: the rows the flag really does and does not suppress.
	cmd := exec.Command(binaryPath, "verify", conformanceVectorPath(t),
		"--skip-external", "--skip-signatures", "--json")
	cmd.Env = cleanEnv()
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0, got %v\noutput:\n%s", err, raw)
	}
	var out conformanceOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("parsing --json output: %v\n%s", err, raw)
	}

	// Proof Signature emits two rows under this flag — the skip and the
	// warn that says the run established nothing about who signed — so
	// the assertion is over the set of statuses, not over one row.
	signingKey := map[string]bool{}
	proofSig := map[string]bool{}
	for _, s := range out.Steps {
		switch s.Group {
		case "Signing Key":
			signingKey[s.Status] = true
		case "Proof Signature":
			proofSig[s.Status] = true
		}
	}
	if !signingKey["pass"] {
		t.Errorf("Signing Key has no pass row under --skip-signatures (E.9 is local and always runs)\n%s",
			formatCLISteps(out.Steps))
	}
	if !proofSig["skip"] || proofSig["pass"] {
		t.Errorf("Proof Signature under --skip-signatures: want a skip and no pass, got %v\n%s",
			proofSig, formatCLISteps(out.Steps))
	}

	// Half two: the description must not claim to skip the row that just
	// passed.
	helpCmd := exec.Command(binaryPath, "verify", "--help")
	helpCmd.Env = cleanEnv()
	help, err := helpCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("verify --help: %v\n%s", err, help)
	}
	var line string
	for _, l := range strings.Split(string(help), "\n") {
		if strings.Contains(l, "--skip-signatures") {
			line = l
			break
		}
	}
	if line == "" {
		t.Fatalf("no --skip-signatures line in help:\n%s", help)
	}
	if strings.Contains(strings.ToLower(line), "signing key") {
		t.Errorf("--skip-signatures help claims to skip the Signing Key check, which still runs and passes: %q", strings.TrimSpace(line))
	}
}

// replaceEnv returns env with key set to value, dropping any existing
// entry for key.
func replaceEnv(env []string, key, value string) []string {
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if !strings.HasPrefix(e, key+"=") {
			out = append(out, e)
		}
	}
	return append(out, key+"="+value)
}

// conformanceOutput is the subset of `verify --json` the D.4 containment
// check reads.
type conformanceOutput struct {
	Result string `json:"result"`
	Steps  []struct {
		Group    string `json:"group"`
		Category string `json:"category"`
		Status   string `json:"status"`
		Message  string `json:"message"`
	} `json:"steps"`
}

// runConformanceVector verifies the Appendix D vector offline against its
// own claims hash under the supplied environment, and fatals unless the
// process exits 0.
func runConformanceVector(t *testing.T, env []string) conformanceOutput {
	t.Helper()
	const claimsHash = "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"

	cmd := exec.Command(binaryPath, "verify", conformanceVectorPath(t),
		"--skip-external", "--hash", claimsHash, "--json")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0, got %v\noutput:\n%s", err, out)
	}
	var got conformanceOutput
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("parsing --json output: %v\n%s", err, out)
	}
	return got
}

// d4CLIViolations is E.25's one-way containment against Appendix D.4,
// evaluated on the --json step list. It mirrors internal/verify's
// d4Violations — same rules, different surface — because the exit code
// and the published JSON are what a caller consumes and either can
// diverge from the in-process report.
//
// Counting is over rows carrying D.4's OWN status: a minimum (a D.4 row
// may not vanish, and a multi-row group may not shed one of its rows), a
// maximum on verdict-moving statuses only, and no constraint at all on
// additive skip/info rows, which E.25 permits wherever they land.
func d4CLIViolations(steps []struct {
	Group    string `json:"group"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Message  string `json:"message"`
},
) []string {
	// D.4's fourteen rows. The two multiplicities are the ones the
	// appendix itself licenses: Epoch Proof is one row per `cx` entry,
	// and D.4's single Subject Data row names both the 0x11 data hash and
	// the 0x13 composite, which the CLI reports separately.
	type key struct{ group, category, status string }
	want := map[key]int{
		{"Hash Comparison", "data_integrity", "pass"}: 1,
		{"Structure", "structural", "pass"}:           1,
		{"Signing Key", "cryptographic", "pass"}:      1,
		{"Subject Data", "cryptographic", "pass"}:     2,
		{"Inclusion Proof", "cryptographic", "pass"}:  1,
		{"Block Hash", "cryptographic", "pass"}:       1,
		{"Epoch Proof", "cryptographic", "pass"}:      2,
		{"Proof Signature", "cryptographic", "pass"}:  1,
		{"Key Binding", "cryptographic", "skip"}:      1,
		{"Stellar Commitment", "blockchain", "skip"}:  1,
		{"Bitcoin Commitment", "blockchain", "skip"}:  1,
		{"Submission Window", "timing", "pass"}:       1,
		{"Temporal Info", "timing", "info"}:           1,
	}

	seen := map[key]int{}
	for _, s := range steps {
		seen[key{s.Group, s.Category, s.Status}]++
	}

	var out []string
	for k, n := range want {
		if seen[k] < n {
			out = append(out, fmt.Sprintf("D.4 row missing or short: %s / %s / %s — got %d, want %d",
				k.group, k.category, k.status, seen[k], n))
		}
	}
	// E.25: additive skip and info rows are conformant; an additive
	// pass, warn or fail row is not.
	budget := map[key]int{}
	for k, n := range want {
		budget[k] = n
	}
	for _, s := range steps {
		if s.Status == "skip" || s.Status == "info" {
			continue
		}
		k := key{s.Group, s.Category, s.Status}
		if budget[k] == 0 {
			out = append(out, fmt.Sprintf("additive %s row not in D.4: %s / %s — %s", s.Status, s.Group, s.Category, s.Message))
			continue
		}
		budget[k]--
	}
	sort.Strings(out)
	return out
}

// formatCLISteps renders the --json step list for failure output.
func formatCLISteps(steps []struct {
	Group    string `json:"group"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Message  string `json:"message"`
},
) string {
	var b strings.Builder
	for _, s := range steps {
		fmt.Fprintf(&b, "  %-6s %-14s %-20s %s\n", s.Status, s.Category, s.Group, s.Message)
	}
	return b.String()
}

// TestCLI_AppendixD_WrongHash_ExitsNonZero pins E.7's mismatch arm at the
// process boundary, on the one bundle whose report Appendix D publishes.
//
// The judging pass found this producer unguarded: downgrading the single
// `r.fail` behind it to a skip left the whole suite green while the CLI
// printed "VERIFIED - proof is valid" and exited 0 for a --hash that does
// not match the proof's claims hash. internal/verify pins the step and the
// verdict; this pins what a caller actually observes — a non-zero exit and
// a verdict that says the data, not the proof, is wrong.
func TestCLI_AppendixD_WrongHash_ExitsNonZero(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", conformanceVectorPath(t),
		"--skip-external", "--hash", strings.Repeat("ab", 32), "--json")
	cmd.Env = cleanEnv()
	out, err := cmd.Output()
	if err == nil {
		t.Fatalf("a --hash that does not match exited 0:\n%s", out)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code: got %d, want 1", exitErr.ExitCode())
	}

	var got conformanceOutput
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("parsing --json output: %v\n%s", err, out)
	}
	if got.Result != "hash_mismatch" {
		t.Errorf("result: got %q, want %q", got.Result, "hash_mismatch")
	}
	// E.7 keeps the two facts apart: the proof is sound, the caller's
	// hash is not the one it commits to. A Hash Comparison row that is
	// anything but a fail here is the defect.
	var seen bool
	for _, s := range got.Steps {
		if s.Group != "Hash Comparison" {
			continue
		}
		seen = true
		if s.Status != "fail" {
			t.Errorf("Hash Comparison: status %q, want fail — %s", s.Status, s.Message)
		}
	}
	if !seen {
		t.Errorf("no Hash Comparison row at all:\n%s", formatCLISteps(got.Steps))
	}
}
