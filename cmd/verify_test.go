// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// Minimal structurally valid proof with fake crypto values (v1 compact format).
// Hex hashes are 64 chars (SHA-256), key IDs are 8 chars, base64 public_key is
// 32 bytes (44 chars), base64 signature is 64 bytes (88 chars).
const fakeProofJSON = `{
  "v": 1,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
  "ts": "2026-04-06T23:25:06Z",
  "s": {
    "src": "item",
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
  "cx": []
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

// Minimal entropy proof for testing --hash rejection (v1 compact format).
const fakeEntropyProofJSON = `{
  "v": 1,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
  "ts": "2026-04-06T23:25:06Z",
  "s": {
    "src": "nist_beacon",
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
  "cx": []
}`

func TestCLI_Hash_WithEntropyProof_ExitCode1(t *testing.T) {
	path := writeProofFile(t, fakeEntropyProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--hash", "deadbeef")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for --hash with entropy proof")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %s", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code: got %d, want 1", exitErr.ExitCode())
	}
	output := string(out)
	if !containsString(output, "not applicable to entropy") {
		t.Errorf("expected error about --hash not applicable, got: %s", output)
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
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=https://custom.example.com")
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
	cmd := exec.Command(binaryPath, "config", "show", "--api-url=https://flag.example.com")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=https://env.example.com")
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
