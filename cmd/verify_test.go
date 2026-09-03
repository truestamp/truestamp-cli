// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
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

	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// These tests build the actual binary and run it as a subprocess to verify
// exit codes and output. This is the idiomatic Go approach for testing CLI
// tools that call os.Exit.

var (
	binaryPath         string
	subprocessCoverDir string // set during TestMain when coverage is requested
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "truestamp-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "truestamp")

	modRoot, err := findModuleRoot()
	if err != nil {
		panic("cannot find module root: " + err.Error())
	}

	// When TRUESTAMP_COVERDIR is set (task test-coverage-full sets it to
	// coverage/bin), build the subprocess binary with -cover so its runtime
	// coverage is recorded there.
	buildArgs := []string{"build"}
	subprocessCoverDir = os.Getenv("TRUESTAMP_COVERDIR")
	if subprocessCoverDir != "" {
		buildArgs = append(buildArgs, "-cover", "-coverpkg=./...")
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

func containsString(s, substr string) bool { return strings.Contains(s, substr) }

// stripANSI removes ANSI escape sequences from output.
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

// Fixture paths under the repository's shared testdata/.
func prodPath(name string) string   { return testfixtures.Path(testfixtures.ProdDir, name) }
func tamperPath(name string) string { return testfixtures.Path(testfixtures.TamperDir, name) }
func prodKeyring() string           { return prodPath(testfixtures.ProdKeyring) }

// conformanceVectorPath locates the Appendix D.1 bundle.
func conformanceVectorPath(t *testing.T) string {
	t.Helper()
	return testfixtures.Path(testfixtures.WhitepaperDir, testfixtures.AppendixD)
}

const appendixDClaimsHash = "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"

// cliStep is one row of `verify --json`.
type cliStep struct {
	Group    string `json:"group"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// conformanceOutput is the subset of `verify --json` these tests read: the
// server's field names, plus the rejection object for a refused bundle.
type conformanceOutput struct {
	Passed    bool      `json:"passed"`
	Steps     []cliStep `json:"steps"`
	Rejection *struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
		Advice string `json:"advice"`
	} `json:"rejection"`
	PassCount            int     `json:"pass_count"`
	FailedCount          int     `json:"failed_count"`
	WarnCount            int     `json:"warn_count"`
	SkipCount            int     `json:"skip_count"`
	InfoCount            int     `json:"info_count"`
	HashProvided         *string `json:"hash_provided"`
	ExpectedHashProvided bool    `json:"expected_hash_provided"`
	HashMatched          bool    `json:"hash_matched"`
	ProofVersion         int     `json:"proof_version"`
	SkippedExternal      bool    `json:"skipped_external"`
	SignaturesChecked    bool    `json:"signatures_checked"`
	Verifier             struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"verifier"`
}

// formatCLISteps renders the --json step list for failure output.
func formatCLISteps(steps []cliStep) string {
	var b strings.Builder
	for _, s := range steps {
		fmt.Fprintf(&b, "  %-6s %-14s %-20s %s\n", s.Status, s.Category, s.Group, s.Message)
	}
	return b.String()
}

// runVerifyJSON runs `verify --json` and returns the parsed report plus the
// process exit code.
func runVerifyJSON(t *testing.T, bin, proofPath string, extra ...string) (conformanceOutput, int) {
	t.Helper()
	args := append([]string{"verify", proofPath, "--json"}, extra...)
	cmd := exec.Command(bin, args...)
	cmd.Env = cleanEnv()
	raw, err := cmd.Output()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("running %s: %v", bin, err)
		}
		code = exitErr.ExitCode()
	}
	var out conformanceOutput
	if jErr := json.Unmarshal(raw, &out); jErr != nil {
		t.Fatalf("parsing --json output (exit %d): %v\n%s", code, jErr, raw)
	}
	return out, code
}

// stepStatuses returns the set of statuses reported under a group.
func stepStatuses(out conformanceOutput, group string) map[string]bool {
	got := map[string]bool{}
	for _, s := range out.Steps {
		if s.Group == group {
			got[s.Status] = true
		}
	}
	return got
}

// rawIssueText concatenates every step message.
func rawIssueText(out conformanceOutput) string {
	var b strings.Builder
	for _, s := range out.Steps {
		b.WriteString(s.Message)
		b.WriteString("\n")
	}
	return b.String()
}

// runCLIText runs the binary and returns stdout+stderr (ANSI stripped) and
// the exit code.
func runCLIText(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("running: %v", err)
		}
		code = exitErr.ExitCode()
	}
	return stripANSI(string(out)), code
}

// --- basics ---

func TestCLI_MissingFile_ExitCode1(t *testing.T) {
	out, code := runCLIText(t, "verify", "/nonexistent/proof.json", "--offline")
	if code != 1 {
		t.Errorf("exit %d, want 1\n%s", code, out)
	}
}

func TestCLI_Silent_NoOutput(t *testing.T) {
	for _, args := range [][]string{
		{"verify", "/nonexistent/proof.json", "--silent"},
		{"verify", tamperPath("tamper-claims.json"), "--offline", "--silent"},
		{"verify", tamperPath("old-layout.json"), "--offline", "--silent"},
	} {
		out, code := runCLIText(t, args...)
		if code != 1 || out != "" {
			t.Errorf("%v: exit %d output %q", args, code, out)
		}
	}
	out, code := runCLIText(t, "verify", prodPath(testfixtures.ProdComplete), "--offline", "--silent")
	if code != 0 || out != "" {
		t.Errorf("silent pass: exit %d output %q", code, out)
	}
}

func TestCLI_NoArgs_ShowsHelp(t *testing.T) {
	out, code := runCLIText(t, "verify")
	if code != 0 || !strings.Contains(out, "Usage:") {
		t.Errorf("exit %d\n%s", code, out)
	}
}

func TestCLI_SilentAndJSON_Exclusive(t *testing.T) {
	out, code := runCLIText(t, "verify", prodPath(testfixtures.ProdComplete), "--silent", "--json")
	if code == 0 || !strings.Contains(out, "mutually exclusive") {
		t.Errorf("exit %d\n%s", code, out)
	}
}

// --- the production bundles, offline ---

func TestCLI_Verify_ProductionOffline_ReferenceShape(t *testing.T) {
	out, code := runCLIText(t, "verify", prodPath(testfixtures.ProdComplete), "--offline", "--keyring", prodKeyring())
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	for _, want := range []string{
		"VERIFICATION REPORT",
		"Data Integrity", "Cryptographic", "Structural", "Timing", "Blockchain",
		"[PASS]  Signing Key          public key valid, key_id 3c19f776",
		"[PASS]  Key Binding          key_id 3c19f776 found in the pinned keyring (sequence 0, active true)",
		"[SKIP]  Stellar Commitment   not checked offline",
		"21 passed   0 failed   0 warned   6 skipped   8 info",
		"file hash provided: no",
		"VERDICT: PASSED",
		"Any `skip` above is a check this",
		"keyring:      pinned ",
		"mode:         offline",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
}

func TestCLI_Verify_ProductionOffline_JSONShape(t *testing.T) {
	out, code := runVerifyJSON(t, binaryPath, prodPath(testfixtures.ProdComplete), "--offline", "--keyring", prodKeyring())
	if code != 0 || !out.Passed {
		t.Fatalf("exit %d passed %v\n%s", code, out.Passed, formatCLISteps(out.Steps))
	}
	if out.PassCount != 21 || out.FailedCount != 0 || out.WarnCount != 0 || out.SkipCount != 6 || out.InfoCount != 8 {
		t.Errorf("counts = %d/%d/%d/%d/%d", out.PassCount, out.FailedCount, out.WarnCount, out.SkipCount, out.InfoCount)
	}
	if out.HashProvided != nil || out.ExpectedHashProvided || out.HashMatched {
		t.Errorf("hash fields: %v %v %v", out.HashProvided, out.ExpectedHashProvided, out.HashMatched)
	}
	if out.ProofVersion != 1 || !out.SkippedExternal || !out.SignaturesChecked || out.Verifier.Name != "truestamp-cli" {
		t.Errorf("proof_version=%d skipped_external=%v signatures_checked=%v verifier=%+v", out.ProofVersion, out.SkippedExternal, out.SignaturesChecked, out.Verifier)
	}
	if len(out.Steps) != 35 {
		t.Errorf("%d steps, want 35\n%s", len(out.Steps), formatCLISteps(out.Steps))
	}
	// Category display order.
	rank := map[string]int{"data_integrity": 0, "cryptographic": 1, "structural": 2, "timing": 3, "blockchain": 4}
	last := -1
	for _, s := range out.Steps {
		if rank[s.Category] < last {
			t.Errorf("steps are not in category order at %s/%s", s.Category, s.Group)
		}
		last = rank[s.Category]
	}
}

func TestCLI_Verify_AllThreeVariantsAndCBOR(t *testing.T) {
	for _, name := range []string{testfixtures.ProdComplete, testfixtures.ProdCompact, testfixtures.ProdPartial, testfixtures.ProdCBOR} {
		out, code := runVerifyJSON(t, binaryPath, prodPath(name), "--offline", "--keyring", prodKeyring())
		if code != 0 || !out.Passed {
			t.Errorf("%s: exit %d passed %v\n%s", name, code, out.Passed, formatCLISteps(out.Steps))
		}
	}
}

func TestCLI_Verify_NoKeyring_KeyBindingSkips(t *testing.T) {
	out, code := runVerifyJSON(t, binaryPath, prodPath(testfixtures.ProdComplete), "--offline")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if st := stepStatuses(out, "Key Binding"); !st["skip"] || st["pass"] {
		t.Errorf("Key Binding = %v", st)
	}
	out, code = runVerifyJSON(t, binaryPath, prodPath(testfixtures.ProdComplete), "--offline", "--keyring", "/nonexistent/keyring.json")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if st := stepStatuses(out, "Key Binding"); !st["skip"] || !strings.Contains(rawIssueText(out), "could not read") {
		t.Errorf("unreadable keyring: %v", st)
	}
}

func TestCLI_Verify_Stdin(t *testing.T) {
	data, _ := os.ReadFile(prodPath(testfixtures.ProdComplete))
	cmd := exec.Command(binaryPath, "verify", "--offline", "--json")
	cmd.Env = cleanEnv()
	cmd.Stdin = bytes.NewReader(data)
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("stdin verify failed: %v\n%s", err, raw)
	}
	var out conformanceOutput
	if err := json.Unmarshal(raw, &out); err != nil || !out.Passed {
		t.Errorf("stdin: %v passed=%v", err, out.Passed)
	}
}

// --- rejections ---

func TestCLI_Verify_Rejection_Text(t *testing.T) {
	out, code := runCLIText(t, "verify", tamperPath("old-layout.json"), "--offline")
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	for _, want := range []string{"REJECTED: unsupported_layout", "pre-publication draft layout", "regenerate the proof"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "VERIFICATION REPORT") {
		t.Error("a rejection must not render a report")
	}
	out, code = runCLIText(t, "verify", tamperPath("tamper-type.json"), "--offline")
	if code != 1 || !strings.Contains(out, "REJECTED: unexpected_subject_fields_for_block_like") {
		t.Errorf("exit %d\n%s", code, out)
	}
}

func TestCLI_Verify_Rejection_JSON(t *testing.T) {
	out, code := runVerifyJSON(t, binaryPath, tamperPath("old-layout.json"), "--offline")
	if code != 1 || out.Passed || out.Rejection == nil || out.Rejection.Code != "unsupported_layout" {
		t.Errorf("exit %d passed %v rejection %+v", code, out.Passed, out.Rejection)
	}
	if len(out.Steps) != 0 {
		t.Errorf("a rejection carries no steps, got %d", len(out.Steps))
	}
	if !strings.Contains(out.Rejection.Advice, "regenerate") {
		t.Errorf("advice = %q", out.Rejection.Advice)
	}
}

func TestCLI_Verify_TypeAssertion(t *testing.T) {
	out, code := runVerifyJSON(t, binaryPath, prodPath(testfixtures.ProdComplete), "--offline", "--type", "block")
	if code != 1 || out.Rejection == nil || out.Rejection.Code != "subject_type_mismatch" {
		t.Errorf("mismatch: exit %d rejection %+v", code, out.Rejection)
	}
	out, code = runVerifyJSON(t, binaryPath, prodPath(testfixtures.ProdComplete), "--offline", "--type", "item")
	if code != 0 || !out.Passed {
		t.Errorf("match: exit %d passed %v", code, out.Passed)
	}
	text, code := runCLIText(t, "verify", prodPath(testfixtures.ProdComplete), "--offline", "--type", "entropy")
	if code == 0 || !strings.Contains(text, "--type must be one of") {
		t.Errorf("bare entropy accepted: exit %d\n%s", code, text)
	}
}

// TestCLI_Verify_FilenameNeverAffectsVerdict pins Appendix E.24: the type
// is read from the bundle's signed `type`, never from the filename.
func TestCLI_Verify_FilenameNeverAffectsVerdict(t *testing.T) {
	data, _ := os.ReadFile(prodPath(testfixtures.ProdComplete))
	path := filepath.Join(t.TempDir(), "truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	out, code := runVerifyJSON(t, binaryPath, path, "--offline")
	if code != 0 || !out.Passed {
		t.Errorf("renamed bundle: exit %d passed %v", code, out.Passed)
	}
}

// --- expected hash (E.7) ---

func TestCLI_Verify_ExpectedHash(t *testing.T) {
	path := conformanceVectorPath(t)
	out, code := runVerifyJSON(t, binaryPath, path, "--offline", "--expected-hash", strings.ToUpper(appendixDClaimsHash))
	if code != 0 || !out.Passed || !out.HashMatched || !out.ExpectedHashProvided || out.HashProvided == nil || *out.HashProvided != appendixDClaimsHash {
		t.Errorf("match: exit %d passed %v matched %v provided %v", code, out.Passed, out.HashMatched, out.HashProvided)
	}
	if out.PassCount != 22 || out.SkipCount != 7 || out.InfoCount != 9 || out.WarnCount != 0 {
		t.Errorf("D.4 counts = %d/%d/%d/%d", out.PassCount, out.WarnCount, out.SkipCount, out.InfoCount)
	}

	// The older spelling still works.
	out, code = runVerifyJSON(t, binaryPath, path, "--offline", "--hash", appendixDClaimsHash)
	if code != 0 || !out.HashMatched {
		t.Errorf("--hash alias: exit %d matched %v", code, out.HashMatched)
	}

	// A wrong hash fails only the Hash Comparison row and exits 1.
	out, code = runVerifyJSON(t, binaryPath, path, "--offline", "--expected-hash", strings.Repeat("ab", 32))
	if code != 1 || out.Passed || out.HashMatched || out.FailedCount != 1 {
		t.Errorf("mismatch: exit %d passed %v matched %v failed %d", code, out.Passed, out.HashMatched, out.FailedCount)
	}
	if st := stepStatuses(out, "Hash Comparison"); !st["fail"] {
		t.Errorf("Hash Comparison = %v", st)
	}

	// No hash on an item that commits to one: warn, never fail.
	out, code = runVerifyJSON(t, binaryPath, path, "--offline")
	if code != 0 || !out.Passed || out.WarnCount != 1 {
		t.Errorf("no hash: exit %d passed %v warns %d", code, out.Passed, out.WarnCount)
	}

	// A hash for a hashless item warns and never fails.
	out, code = runVerifyJSON(t, binaryPath, prodPath(testfixtures.ProdComplete), "--offline", "--expected-hash", appendixDClaimsHash)
	if code != 0 || !out.Passed || out.HashMatched || out.HashProvided != nil {
		t.Errorf("hashless item: exit %d passed %v matched %v provided %v", code, out.Passed, out.HashMatched, out.HashProvided)
	}
	if st := stepStatuses(out, "Hash Comparison"); !st["warn"] {
		t.Errorf("Hash Comparison for a hashless item = %v", st)
	}

	// Malformed arguments are refused before anything runs.
	for _, bad := range []string{"xyz", "abc"} {
		text, code := runCLIText(t, "verify", path, "--offline", "--expected-hash", bad)
		if code == 0 || !strings.Contains(text, "--expected-hash") {
			t.Errorf("%q accepted: exit %d\n%s", bad, code, text)
		}
	}
}

// --- tamper and flags ---

func TestCLI_Verify_Tamper_ExitCode1(t *testing.T) {
	out, code := runCLIText(t, "verify", tamperPath("tamper-claims.json"), "--offline", "--keyring", prodKeyring())
	if code != 1 || !strings.Contains(out, "VERDICT: FAILED") || !strings.Contains(out, "[FAIL]  Proof Signature") {
		t.Errorf("exit %d\n%s", code, out)
	}
}

func TestCLI_SkipSignatures_DisclosedNotHidden(t *testing.T) {
	out, code := runVerifyJSON(t, binaryPath, prodPath(testfixtures.ProdComplete), "--offline", "--skip-signatures")
	if code != 0 || !out.Passed || out.SignaturesChecked {
		t.Errorf("exit %d passed %v signatures_checked %v", code, out.Passed, out.SignaturesChecked)
	}
	if st := stepStatuses(out, "Proof Signature"); !st["skip"] || !st["warn"] || st["pass"] {
		t.Errorf("Proof Signature = %v", st)
	}
	if st := stepStatuses(out, "Signing Key"); !st["pass"] {
		t.Errorf("Signing Key still runs under --skip-signatures, got %v", st)
	}
	text, _ := runCLIText(t, "verify", prodPath(testfixtures.ProdComplete), "--offline", "--skip-signatures")
	if !strings.Contains(text, "NOT checked") {
		t.Errorf("the verdict does not disclose the skipped signature\n%s", text)
	}
	help, _ := runCLIText(t, "verify", "--help")
	for _, l := range strings.Split(help, "\n") {
		if strings.Contains(l, "--skip-signatures") && strings.Contains(strings.ToLower(l), "signing key") {
			t.Errorf("--skip-signatures help claims to skip the Signing Key check: %q", strings.TrimSpace(l))
		}
	}
}

func TestCLI_Verify_RemoteRequiresAuth(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", prodPath(testfixtures.ProdComplete), "--remote")
	cmd.Env = append(cleanEnv(), "HOME="+t.TempDir(), "XDG_CONFIG_HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "not authenticated") {
		t.Errorf("remote without a credential: %v\n%s", err, out)
	}
}

// TestCLI_Verify_Remote posts the bundle and reproduces the server's
// report, including a server-side rejection.
func TestCLI_Verify_Remote(t *testing.T) {
	serverReport, _ := os.ReadFile(prodPath("verify-complete.json"))
	// The handler runs on the server's goroutine and the assertions on the
	// test's, so the captured body is guarded.
	var mu sync.Mutex
	var posted map[string]any
	lastPosted := func() map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return posted
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/json/proof/verify", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("missing Bearer header")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		posted = body
		mu.Unlock()
		data := body["data"].(map[string]any)
		proofDoc := data["proof"].(map[string]any)
		if proofDoc["type"] == "block" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors":[{"code":"invalid","meta":{"code":"invalid_proof","reason":"unexpected_subject_fields_for_block_like"},"status":"400","title":"Invalid","detail":"Invalid proof: :unexpected_subject_fields_for_block_like"}]}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(serverReport)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	out, code := runVerifyJSON(t, binaryPath, prodPath(testfixtures.ProdComplete), "--remote", "--base-url", srv.URL, "--api-key", "test-key")
	if code != 0 || !out.Passed || out.PassCount != 29 {
		t.Errorf("remote: exit %d passed %v pass_count %d\n%s", code, out.Passed, out.PassCount, formatCLISteps(out.Steps))
	}
	if data := lastPosted()["data"].(map[string]any); data["skip_external"] != false {
		t.Errorf("skip_external not posted: %v", data)
	}

	// A CBOR input is posted as its JSON conversion.
	out, code = runVerifyJSON(t, binaryPath, prodPath(testfixtures.ProdCBOR), "--remote", "--base-url", srv.URL, "--api-key", "test-key")
	if code != 0 || !out.Passed {
		t.Errorf("remote cbor: exit %d passed %v", code, out.Passed)
	}
	if data := lastPosted()["data"].(map[string]any); data["proof"].(map[string]any)["type"] != "item" {
		t.Errorf("posted proof = %v", data["proof"])
	}

	// A local rejection never reaches the server; a server rejection is
	// rendered like a local one.
	text, code := runCLIText(t, "verify", tamperPath("old-layout.json"), "--remote", "--base-url", srv.URL, "--api-key", "test-key")
	if code != 1 || !strings.Contains(text, "REJECTED: unsupported_layout") {
		t.Errorf("remote local rejection: exit %d\n%s", code, text)
	}
	text, code = runCLIText(t, "verify", tamperPath("tamper-type.json"), "--remote", "--base-url", srv.URL, "--api-key", "test-key")
	if code != 1 || !strings.Contains(text, "REJECTED: unexpected_subject_fields_for_block_like") {
		t.Errorf("server rejection: exit %d\n%s", code, text)
	}
}

// --- inspect ---

func TestCLI_Inspect(t *testing.T) {
	out, code := runCLIText(t, "inspect", prodPath(testfixtures.ProdComplete))
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	for _, want := range []string{
		"item (code 20)", "01M1M0V3SE3C5P32TRAJSNX6QF", "Derived key id         3c19f776",
		"block, entropy_bitcoin, entropy_nist, entropy_stellar", "stellar public", "Carried                yes",
		"type genesis, sequence 0", "Inclusion proof        5 steps",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("inspect output lacks %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "VERDICT") {
		t.Error("inspect must not verify")
	}

	cmd := exec.Command(binaryPath, "inspect", prodPath(testfixtures.ProdCBOR), "--json")
	cmd.Env = cleanEnv()
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("inspect --json: %v\n%s", err, raw)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["format"] != "cbor" || m["type"] != "item" || m["key_id"] != "3c19f776" {
		t.Errorf("inspect json = %v", m)
	}
	subject := m["subject"].(map[string]any)
	if len(subject["carried_witnesses"].([]any)) != 4 || subject["signing_key_id"] != "3c19f776" {
		t.Errorf("subject = %v", subject)
	}

	out, code = runCLIText(t, "inspect", tamperPath("old-layout.json"))
	if code != 1 || !strings.Contains(out, "REJECTED: unsupported_layout") {
		t.Errorf("inspect rejection: exit %d\n%s", code, out)
	}
	out, code = runCLIText(t, "inspect", prodPath(testfixtures.ProdCompact))
	if code != 0 || !strings.Contains(out, "Carried witnesses      (none)") || !strings.Contains(out, "Carried                no") {
		t.Errorf("compact inspect: exit %d\n%s", code, out)
	}
}

// --- config and completion ---

func TestCLI_ConfigPath(t *testing.T) {
	out, code := runCLIText(t, "config", "path")
	if code != 0 || len(out) == 0 {
		t.Errorf("config path: exit %d %q", code, out)
	}
}

func TestCLI_ConfigShow(t *testing.T) {
	out, code := runCLIText(t, "config", "show")
	if code != 0 || !containsString(out, "API URL") || !containsString(out, "Verification") {
		t.Errorf("config show: exit %d\n%s", code, out)
	}
}

func TestCLI_EnvVarOverride(t *testing.T) {
	cmd := exec.Command(binaryPath, "config", "show")
	cmd.Env = append(os.Environ(), "TRUESTAMP_BASE_URL=https://custom.example.com")
	out, err := cmd.CombinedOutput()
	if err != nil || !containsString(string(out), "https://custom.example.com") {
		t.Errorf("env var override: %v\n%s", err, out)
	}
}

func TestCLI_FlagOverridesEnv(t *testing.T) {
	cmd := exec.Command(binaryPath, "config", "show", "--base-url=https://flag.example.com")
	cmd.Env = append(os.Environ(), "TRUESTAMP_BASE_URL=https://env.example.com")
	out, err := cmd.CombinedOutput()
	if err != nil || !containsString(string(out), "https://flag.example.com") {
		t.Errorf("flag override: %v\n%s", err, out)
	}
}

func TestCLI_Completion(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		cmd := exec.Command(binaryPath, "completion", shell)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil || len(out) == 0 {
			t.Errorf("completion %s: %v", shell, err)
		}
		if shell == "zsh" && stderr.Len() > 0 {
			t.Errorf("completion zsh wrote to stderr: %s", stderr.String())
		}
	}
}

// --- Appendix D.4 at the process boundary ---

// TestCLI_AppendixD_Conformance is Appendix E.25's self-certification at
// the process boundary: the published worked example, verified offline
// against its own claims hash, must exit 0 and report every D.4 row with
// D.4's status.
func TestCLI_AppendixD_Conformance(t *testing.T) {
	out, code := runVerifyJSON(t, binaryPath, conformanceVectorPath(t), "--offline", "--expected-hash", appendixDClaimsHash)
	if code != 0 || !out.Passed {
		t.Fatalf("exit %d passed %v\n%s", code, out.Passed, formatCLISteps(out.Steps))
	}
	if violations := d4Violations(out.Steps); len(violations) > 0 {
		t.Errorf("D.4 containment:\n  %s\n%s", strings.Join(violations, "\n  "), formatCLISteps(out.Steps))
	}
}

// d4Violations checks E.25's one-way containment against Appendix D.4.
func d4Violations(steps []cliStep) []string {
	type key struct{ group, category, status string }
	want := map[key]int{
		{"Hash Comparison", "data_integrity", "pass"}:  1,
		{"Signing Key", "cryptographic", "pass"}:       1,
		{"Subject Data", "cryptographic", "pass"}:      3,
		{"Inclusion Proof", "cryptographic", "pass"}:   1,
		{"Block Hash", "cryptographic", "pass"}:        2,
		{"Epoch Proof", "cryptographic", "pass"}:       2,
		{"Proof Signature", "cryptographic", "pass"}:   1,
		{"Key Binding", "cryptographic", "skip"}:       1,
		{"Signing Key Event", "cryptographic", "pass"}: 4,
		{"Signing Key Event", "cryptographic", "info"}: 1,
		{"Signing Key Event", "cryptographic", "skip"}: 1,
		{"Structure", "structural", "pass"}:            1,
		{"Witnesses", "timing", "pass"}:                5,
		{"Submission Window", "timing", "pass"}:        1,
		{"Temporal Info", "timing", "info"}:            1,
		{"Submitted After", "timing", "info"}:          5,
		{"Submitted Before", "timing", "info"}:         1,
		{"Stellar Commitment", "blockchain", "skip"}:   1,
		{"Bitcoin Commitment", "blockchain", "skip"}:   1,
		{"Entropy Source", "blockchain", "skip"}:       3,
	}
	seen := map[key]int{}
	for _, s := range steps {
		seen[key{s.Group, s.Category, s.Status}]++
	}
	var out []string
	for k, n := range want {
		if seen[k] < n {
			out = append(out, fmt.Sprintf("D.4 row missing or short: %s / %s / %s: got %d, want %d", k.group, k.category, k.status, seen[k], n))
		}
	}
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
			out = append(out, fmt.Sprintf("additive %s row not in D.4: %s / %s: %s", s.Status, s.Group, s.Category, s.Message))
			continue
		}
		budget[k]--
	}
	sort.Strings(out)
	return out
}
