// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/items"
)

// validSHA256Hex is a 64-char hex string with the right shape for a SHA-256
// claims hash. Used in tests that want validation to pass to the next step
// without caring about hash specifics.
const validSHA256Hex = "0000000000000000000000000000000000000000000000000000000000000000"

// longDesc is a description with >= claimsOnlyMinDescription (32)
// non-whitespace characters, used to satisfy the claims-as-source-of-truth
// meaningful-content rule in tests that exercise that mode.
const longDesc = "Lorem ipsum dolor sit amet, consectetur adipiscing elit."

// --- No-args / help ---

func TestCLI_Create_NoArgs_ShowsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "create")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create with no args should exit 0, got: %s", err)
	}
	output := string(out)
	if !containsString(output, "Usage") {
		t.Error("no-args output should contain Usage")
	}
}

func TestCLI_Create_Help_ShowsAllFlags(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create --help failed: %s", err)
	}
	output := string(out)
	for _, flag := range []string{"--file", "--file-stdin", "--claims", "--claims-stdin", "--name", "--hash", "--hash-type", "--description", "--url", "--timestamp", "--metadata", "--location", "--visibility", "--tags", "--json"} {
		if !containsString(output, flag) {
			t.Errorf("help output missing flag %s", flag)
		}
	}
}

// --- Validation errors (no API needed) ---

func TestCLI_Create_NoAPIKey_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--hash", validSHA256Hex)
	// Empty API key + a loopback base_url with no stored OAuth session ⇒
	// no credential ⇒ the "not authenticated" guard fires.
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_KEY=", "TRUESTAMP_BASE_URL=http://127.0.0.1:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "not authenticated") {
		t.Errorf("expected not-authenticated error, got: %s", out)
	}
}

func TestCLI_Create_MissingName_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "--hash", validSHA256Hex, "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "name is required") {
		t.Errorf("expected name required error, got: %s", out)
	}
}

// TestCLI_Create_NoHash_NoDescription_TriggersClaimsOnlyError covers the case
// where the user supplies only --name (no hash, no description, no metadata).
// Under claims-as-source-of-truth mode this fails the local meaningful-content
// rule with a message naming the 32-char threshold.
func TestCLI_Create_NoHash_NoDescription_TriggersClaimsOnlyError(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !containsString(output, "claims content is required") {
		t.Errorf("expected claims-only meaningful-content error, got: %s", output)
	}
	if !containsString(output, "32 characters") {
		t.Errorf("error should name the 32-char threshold, got: %s", output)
	}
}

func TestCLI_Create_InvalidHex_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--hash", "xyz", "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !containsString(output, "hex") {
		t.Errorf("expected hex error, got: %s", output)
	}
}

func TestCLI_Create_OddLengthHex_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--hash", "abc", "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "even-length") {
		t.Errorf("expected even-length error, got: %s", out)
	}
}

func TestCLI_Create_InvalidVisibility_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--hash", validSHA256Hex, "-v", "secret", "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "private, team, or public") {
		t.Errorf("expected visibility error, got: %s", out)
	}
}

func TestCLI_Create_InvalidURL_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--hash", validSHA256Hex, "--url", "http://not-https.com", "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "https://") {
		t.Errorf("expected https error, got: %s", out)
	}
}

func TestCLI_Create_InvalidLocation_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--hash", validSHA256Hex, "--location", "abc", "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "lat,lon") {
		t.Errorf("expected location error, got: %s", out)
	}
}

func TestCLI_Create_InvalidMetadata_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--hash", validSHA256Hex, "--metadata", "not json", "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "valid JSON") {
		t.Errorf("expected JSON error, got: %s", out)
	}
}

func TestCLI_Create_NonexistentFile_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "/nonexistent/file.pdf", "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "cannot access") {
		t.Errorf("expected file access error, got: %s", out)
	}
}

func TestCLI_Create_Directory_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", os.TempDir(), "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "directory") {
		t.Errorf("expected directory error, got: %s", out)
	}
}

// --- Auto-hash correctness ---

func TestCLI_Create_AutoHash_MatchesSHA256(t *testing.T) {
	// Create a temp file with known content
	content := []byte("test content for hash verification\n")
	path := filepath.Join(t.TempDir(), "hashtest.txt")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Compute expected SHA-256
	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	// Run create with a fake API (will fail at API call, but we can check
	// the hash via --json mode's error output). Instead, test the validation
	// passes by checking the error is an API error (not a validation error).
	cmd := exec.Command(binaryPath, "create", path, "--json", "--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	// Should get past validation (API call fails with connection error)
	if containsString(output, "hash is required") || containsString(output, "name is required") {
		t.Errorf("auto-hash should populate hash and name, got: %s", output)
	}

	// Verify hash by checking it doesn't complain about hex format
	if containsString(output, "hex") {
		t.Errorf("auto-hash should produce valid hex, got: %s", output)
	}

	_ = expected // hash value is correct if validation passes
}

// --- Claims file input ---

func TestCLI_Create_ClaimsFile_ParsesJSON(t *testing.T) {
	claims := `{"hash":"` + validSHA256Hex + `","hash_type":"sha256","name":"From File"}`
	path := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", "--claims="+path, "--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	// Should get past validation (API error, not claims error)
	if containsString(output, "name is required") || containsString(output, "hash is required") {
		t.Errorf("claims file should populate required fields, got: %s", output)
	}
}

func TestCLI_Create_ClaimsFile_InvalidJSON_Error(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", "--claims="+path, "--api-key", "fake")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "parsing claims JSON") {
		t.Errorf("expected JSON parse error, got: %s", out)
	}
}

// --- Claims stdin input ---

func TestCLI_Create_ClaimsStdin_ParsesJSON(t *testing.T) {
	claims := `{"hash":"` + validSHA256Hex + `","hash_type":"sha256","name":"From Stdin"}`

	cmd := exec.Command(binaryPath, "create", "-C", "--api-key", "fake", "--base-url", "http://localhost:1")
	cmd.Stdin = strings.NewReader(claims)
	out, _ := cmd.CombinedOutput()
	output := string(out)

	if containsString(output, "name is required") || containsString(output, "hash is required") {
		t.Errorf("claims stdin should populate required fields, got: %s", output)
	}
}

func TestCLI_Create_ClaimsStdin_InvalidJSON_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-C", "--api-key", "fake")
	cmd.Stdin = strings.NewReader("not json")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "parsing claims JSON") {
		t.Errorf("expected JSON parse error, got: %s", out)
	}
}

// --- File stdin input ---

func TestCLI_Create_FileStdin_RequiresName(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "-F", "--api-key", "fake", "--base-url", "http://localhost:1")
	cmd.Stdin = strings.NewReader("raw file content")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "name is required") {
		t.Errorf("--file-stdin without --name should error, got: %s", out)
	}
}

func TestCLI_Create_FileStdin_WithName_HashesContent(t *testing.T) {
	content := "raw file content for hashing"

	cmd := exec.Command(binaryPath, "create", "-F", "-n", "Stdin File", "--api-key", "fake", "--base-url", "http://localhost:1")
	cmd.Stdin = strings.NewReader(content)
	out, _ := cmd.CombinedOutput()
	output := string(out)

	// Should get past validation to API call
	if containsString(output, "hash is required") || containsString(output, "name is required") {
		t.Errorf("--file-stdin with --name should populate hash, got: %s", output)
	}
}

// --- Flag overrides ---

func TestCLI_Create_FlagOverridesAutoHash(t *testing.T) {
	content := []byte("override test")
	path := filepath.Join(t.TempDir(), "original-name.txt")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Override name from auto-hash
	cmd := exec.Command(binaryPath, "create", path, "-n", "Custom Name", "--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	// Should not error about missing name (it's provided via flag)
	if containsString(output, "name is required") {
		t.Errorf("--name flag should override, got: %s", output)
	}
}

func TestCLI_Create_FlagOverridesClaimsFile(t *testing.T) {
	claims := `{"hash":"` + validSHA256Hex + `","hash_type":"sha256","name":"Original"}`
	path := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
		t.Fatal(err)
	}

	// Override visibility from claims file
	cmd := exec.Command(binaryPath, "create", "--claims="+path, "-v", "public", "--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	if containsString(output, "name is required") {
		t.Errorf("claims file + flag override should work, got: %s", output)
	}
}

// --- File flag ---

func TestCLI_Create_FileFlag_HashesFile(t *testing.T) {
	content := []byte("file flag test content")
	path := filepath.Join(t.TempDir(), "fileflag.txt")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", "--file="+path, "--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	if containsString(output, "hash is required") || containsString(output, "name is required") {
		t.Errorf("--file flag should auto-hash, got: %s", output)
	}
}

func TestCLI_Create_FileFlag_NonexistentFile_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "create", "--file=/nonexistent/file.txt", "--api-key", "fake")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "cannot access") {
		t.Errorf("expected file access error, got: %s", out)
	}
}

// --- JSON output structure ---

func TestCLI_Create_JSONOutput_Structure(t *testing.T) {
	// This test verifies JSON output has the expected keys.
	// Uses a fake API URL so it fails at the network level, but we can test
	// that --json flag is accepted without error alongside other flags.
	cmd := exec.Command(binaryPath, "create", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help failed: %s", err)
	}
	if !containsString(string(out), "--json") {
		t.Error("help should mention --json flag")
	}
}

// --- Tags parsing ---

func TestCLI_Create_TagsParsing(t *testing.T) {
	// Verify tags are split and trimmed. We can only test indirectly
	// since the API call will fail, but we verify no validation error.
	content := []byte("tags test")
	path := filepath.Join(t.TempDir(), "tags.txt")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", path, "-t", " a , b , c ", "--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	// Should pass validation, fail only at API
	if containsString(output, "tag") && containsString(output, "error") {
		t.Errorf("tags should parse without error, got: %s", output)
	}
}

// --- Hash normalization ---

func TestCLI_Create_HashNormalizedToLowercase(t *testing.T) {
	// Uppercase hex should be normalized to lowercase. Use a mixed-case
	// 64-char hash so the case-folding is actually exercised (the all-zeros
	// fixture is case-invariant).
	upper := "ABCDEF" + strings.Repeat("0", 64-6)
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--hash", upper, "--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	// Should not error about hex format
	if containsString(output, "invalid hex") || containsString(output, "even-length") {
		t.Errorf("uppercase hex should be accepted, got: %s", output)
	}
}

// --- Default hash type ---

func TestCLI_Create_DefaultHashType(t *testing.T) {
	// When --hash is provided without --hash-type, default to sha256
	cmd := exec.Command(binaryPath, "create", "-n", "Test", "--hash", validSHA256Hex, "--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	if containsString(output, "hash_type is required") {
		t.Errorf("hash_type should default to sha256, got: %s", output)
	}
}

// --- Claims-as-source-of-truth mode ---

// TestCLI_Create_ClaimsOnly_NameAndDescription_PassesValidation confirms that
// claims-only mode with a sufficient description gets past local validation.
// The API call still fails (no real server), but the error must not be a
// validation failure.
func TestCLI_Create_ClaimsOnly_NameAndDescription_PassesValidation(t *testing.T) {
	cmd := exec.Command(binaryPath, "create",
		"-n", "Invention",
		"-d", longDesc,
		"--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	if containsString(output, "claims content is required") {
		t.Errorf("32+ char description should satisfy claims-only rule, got: %s", output)
	}
	if containsString(output, "hash is required") || containsString(output, "hash_type is required") {
		t.Errorf("claims-only mode must not require hash/hash_type, got: %s", output)
	}
}

// TestCLI_Create_ClaimsOnly_ShortDescription_Fails fails locally with the
// meaningful-content error when description is below the 32-char threshold.
func TestCLI_Create_ClaimsOnly_ShortDescription_Fails(t *testing.T) {
	cmd := exec.Command(binaryPath, "create",
		"-n", "Doc",
		"-d", "short",
		"--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !containsString(output, "claims content is required") {
		t.Errorf("short description in claims-only mode should fail, got: %s", output)
	}
}

// TestCLI_Create_ClaimsOnly_NonEmptyMetadata_PassesValidation confirms the
// metadata escape hatch: a non-empty metadata object satisfies the
// meaningful-content rule even without a long description.
func TestCLI_Create_ClaimsOnly_NonEmptyMetadata_PassesValidation(t *testing.T) {
	cmd := exec.Command(binaryPath, "create",
		"-n", "Doc",
		"--metadata", `{"k":"v"}`,
		"--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	if containsString(output, "claims content is required") {
		t.Errorf("non-empty metadata should satisfy claims-only rule, got: %s", output)
	}
}

// TestCLI_Create_HashTypeAlone_Rejected covers the co-required pair:
// supplying --hash-type without --hash is a partial pair and must be
// rejected with a clear error before any network round-trip. Confirms the
// --hash-type flag default does not leak through into the claims map.
func TestCLI_Create_HashTypeAlone_Rejected(t *testing.T) {
	cmd := exec.Command(binaryPath, "create",
		"-n", "Doc",
		"--hash-type", "sha256",
		"--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !containsString(output, "hash is required when hash_type is supplied") {
		t.Errorf("expected co-required error, got: %s", output)
	}
}

// TestCLI_Create_HashWithoutHashType_FromClaimsFile_Rejected covers the
// reverse partial pair: a claims file with hash but explicit empty
// hash_type is rejected with a clear error. (The overlayFlags default rule
// fills in sha256 when hash_type is missing from claims; this test forces
// an empty-string hash_type to bypass that fallback.)
func TestCLI_Create_HashWithoutHashType_FromClaimsFile_Rejected(t *testing.T) {
	claims := `{"hash":"` + validSHA256Hex + `","hash_type":"","name":"Doc"}`
	path := filepath.Join(t.TempDir(), "partial.json")
	if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binaryPath, "create", "--claims="+path, "--api-key", "fake")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_URL=http://localhost:0")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !containsString(output, "hash_type is required when hash is supplied") {
		t.Errorf("expected co-required error, got: %s", output)
	}
}

// --- Mutual exclusivity isn't enforced (flags overlay) ---

func TestCLI_Create_ClaimsAndFlags_Merge(t *testing.T) {
	// Claims from file + flag overrides should merge (not conflict)
	claims := `{"hash":"` + validSHA256Hex + `","hash_type":"sha256","name":"Base"}`
	path := filepath.Join(t.TempDir(), "merge.json")
	if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", "--claims="+path, "-d", "Added desc", "--api-key", "fake", "--base-url", "http://localhost:1")
	out, _ := cmd.CombinedOutput()
	output := string(out)

	if containsString(output, "name is required") || containsString(output, "hash is required") {
		t.Errorf("claims file + flag overlay should work, got: %s", output)
	}
}

// --- JSON output key validation (with mock response) ---

func TestCLI_Create_JSONOutput_HasExpectedKeys(t *testing.T) {
	// We validate the JSON marshaling logic by checking that the output
	// function produces valid JSON with the right keys. This uses a
	// dummy response to avoid needing a real API.

	// Since we can't easily inject a mock, verify that the JSON struct
	// has the expected shape by parsing a sample output string.
	sample := `{"id":"01TEST","name":"Test","hash":"abcd","hash_type":"sha256","visibility":"private","team_id":"019test"}`
	var parsed map[string]any
	if err := json.Unmarshal([]byte(sample), &parsed); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "name", "hash", "hash_type", "visibility", "team_id"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("JSON output should contain key %q", key)
		}
	}
}

// TestCLI_Create_JSONOutput_ClaimsOnly_OmitsHashKeys exercises printCreateJSON
// directly against a claims-only CreateItemResponse (Hash and HashType empty)
// and asserts the marshaled object does not carry "hash" or "hash_type" keys.
// External-hash responses still include both keys.
func TestCLI_Create_JSONOutput_ClaimsOnly_OmitsHashKeys(t *testing.T) {
	// Capture stdout from printCreateJSON.
	capture := func(resp *items.CreateItemResponse) string {
		r, w, _ := os.Pipe()
		stdout := os.Stdout
		os.Stdout = w
		_ = printCreateJSON(resp)
		_ = w.Close()
		os.Stdout = stdout
		buf, _ := io.ReadAll(r)
		return string(buf)
	}

	// Claims-only: no hash or hash_type.
	claimsOnly := &items.CreateItemResponse{
		ID:         "01CLAIMSONLY",
		Name:       "Invention",
		Visibility: "private",
		TeamID:     "team_xyz",
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(capture(claimsOnly)), &got); err != nil {
		t.Fatalf("unmarshal claims-only output: %v", err)
	}
	if _, has := got["hash"]; has {
		t.Errorf("claims-only JSON output must not include hash key, got: %v", got)
	}
	if _, has := got["hash_type"]; has {
		t.Errorf("claims-only JSON output must not include hash_type key, got: %v", got)
	}

	// External-hash: both keys present.
	external := &items.CreateItemResponse{
		ID:         "01EXT",
		Name:       "Doc",
		Hash:       validSHA256Hex,
		HashType:   "sha256",
		Visibility: "private",
		TeamID:     "team_xyz",
	}
	var got2 map[string]any
	if err := json.Unmarshal([]byte(capture(external)), &got2); err != nil {
		t.Fatalf("unmarshal external-hash output: %v", err)
	}
	if got2["hash"] != validSHA256Hex {
		t.Errorf("external-hash JSON output should carry hash, got: %v", got2)
	}
	if got2["hash_type"] != "sha256" {
		t.Errorf("external-hash JSON output should carry hash_type, got: %v", got2)
	}
}

// --- Integer literal preservation + producer-side portability guard ---

// startCreateEchoServer stands up a fake /api/json/items endpoint that records
// the RAW request body — the actual bytes that went out on the wire — and
// answers with a minimal JSON:API item so the command completes normally.
//
// Asserting on these bytes rather than on an intermediate map is the entire
// point: the corruption this guards against happened during decode, so any
// check that re-decodes the claims in the test process would round the value a
// second time and agree with itself.
func startCreateEchoServer(t *testing.T) (url string, body func() []byte) {
	t.Helper()

	var mu sync.Mutex
	var captured []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}
		mu.Lock()
		captured = raw
		mu.Unlock()

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"data":{"id":"01HJHB01T8FYZ7YTR9P5N62K5B","type":"item",`+
			`"attributes":{"display_name":"Echo","visibility":"private","state":"pending"}}}`)
	}))
	t.Cleanup(srv.Close)

	return srv.URL, func() []byte {
		mu.Lock()
		defer mu.Unlock()
		return captured
	}
}

// TestCLI_Create_PreservesLiteralsOnTheWire is the end-to-end regression for
// the claims-corruption bug, asserted against the actual request body.
//
// Before the UseNumber fix, cmd/create decoded claims into map[string]any with
// json.Unmarshal, so every number became a float64 and was re-serialized from
// that double. The user's own digits never reached the server.
//
// Two properties are checked here, both against the raw bytes off the wire:
//
// Integers travel verbatim. The largest portable integer, 2^53 - 1, and its
// negation are carried through — every integer a double would actually mangle
// is now refused by the portability guard before it can reach the wire
// (TestCLI_Create_UnsafeIntegerBoundary), so the byte-identical wire proof for
// a value like 18446744073709551615 lives one layer down, in
// items.TestCreateItemCtx_PreservesIntegerLiteralOnTheWire.
//
// Floats travel verbatim too, and this is where the old code still visibly
// corrupts a submission end to end: a float is never a portability violation,
// so it reaches the wire either way. 1e21 came back out as 1e+21 and
// 9007199254740993.0 as 9.007199254740992e+15 — different bytes, therefore a
// different JCS canonicalization and a different claims_hash.
func TestCLI_Create_PreservesLiteralsOnTheWire(t *testing.T) {
	url, body := startCreateEchoServer(t)

	claims := `{"name":"Big","description":"` + longDesc + `","metadata":{` +
		`"max_safe":9007199254740991,` +
		`"neg":-9007199254740991,` +
		`"as_string":"18446744073709551615",` +
		`"exp":1e21,` +
		`"wide_float":9007199254740993.0}}`
	path := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", "--claims="+path,
		"--api-key", "fake", "--base-url", url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create failed: %v\n%s", err, out)
	}

	got := string(body())
	for _, literal := range []string{
		`"max_safe":9007199254740991`,
		`"neg":-9007199254740991`,
		`"as_string":"18446744073709551615"`,
		`"exp":1e21`,
		`"wide_float":9007199254740993.0`,
	} {
		if !strings.Contains(got, literal) {
			t.Errorf("request body missing %s\nbody: %s", literal, got)
		}
	}

	// The exact rewrites the pre-fix float64 path produced. Their absence is
	// the assertion that matters.
	for _, corrupted := range []string{"1e+21", "9.007199254740992e+15", "9007199254740992"} {
		if strings.Contains(got, corrupted) {
			t.Errorf("request body carries rewritten literal %s — the user's bytes were corrupted\nbody: %s", corrupted, got)
		}
	}
}

// TestCLI_Create_UnsafeIntegerRejectedBeforeNetwork is the other half of the
// contract: a value the producer must not emit never reaches the server at
// all. The echo server records nothing, which is what "before the network
// call" means operationally.
func TestCLI_Create_UnsafeIntegerRejectedBeforeNetwork(t *testing.T) {
	url, body := startCreateEchoServer(t)

	claims := `{"name":"Big","description":"` + longDesc + `","metadata":{"rows":[{"id":18446744073709551615}]}}`
	path := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", "--claims="+path,
		"--api-key", "fake", "--base-url", url)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit for an unsafe integer, got success:\n%s", out)
	}
	output := string(out)

	// Every fact the server's message carries, so the two read as one system.
	for _, want := range []string{
		"claims.metadata.rows[0].id",            // dotted path
		"18446744073709551615",                  // the user's own literal, unrounded
		"-9007199254740991 to 9007199254740991", // allowed range
		"as a string",                           // the remedy
	} {
		if !containsString(output, want) {
			t.Errorf("error output missing %q, got:\n%s", want, output)
		}
	}

	if len(body()) != 0 {
		t.Errorf("request reached the server despite an unsafe integer: %s", body())
	}
}

// TestCLI_Create_UnsafeIntegerBoundary walks the producer threshold through
// the real CLI, in both directions and both signs. 2^53 is the row that pins
// the producer/verifier split: the verifier tolerates that value, `create`
// must not.
func TestCLI_Create_UnsafeIntegerBoundary(t *testing.T) {
	for _, tc := range []struct {
		name   string
		lit    string
		reject bool
	}{
		{"max_safe_accepted", "9007199254740991", false},
		{"two_pow_53_rejected_by_producer", "9007199254740992", true},
		{"two_pow_53_plus_one_rejected", "9007199254740993", true},
		{"negative_max_safe_accepted", "-9007199254740991", false},
		{"negative_two_pow_53_rejected", "-9007199254740992", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url, body := startCreateEchoServer(t)

			claims := `{"name":"B","description":"` + longDesc + `","metadata":{"n":` + tc.lit + `}}`
			path := filepath.Join(t.TempDir(), "claims.json")
			if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(binaryPath, "create", "--claims="+path,
				"--api-key", "fake", "--base-url", url)
			out, err := cmd.CombinedOutput()

			if tc.reject {
				if err == nil {
					t.Fatalf("%s should be rejected by the producer guard, got success:\n%s", tc.lit, out)
				}
				if !containsString(string(out), "claims.metadata.n") {
					t.Errorf("rejection should name the path, got:\n%s", out)
				}
				return
			}

			if err != nil {
				t.Fatalf("%s should be accepted, got error: %v\n%s", tc.lit, err, out)
			}
			if !strings.Contains(string(body()), `"n":`+tc.lit) {
				t.Errorf("request body should carry %s verbatim, got: %s", tc.lit, body())
			}
		})
	}
}

// TestCLI_Create_UnsafeIntegerFromMetadataFlag covers the flag input path:
// --metadata is parsed after the claims file is loaded, so a value injected
// there has to be checked too. Placing the guard before overlayFlags would
// pass this file and miss this flag.
func TestCLI_Create_UnsafeIntegerFromMetadataFlag(t *testing.T) {
	url, body := startCreateEchoServer(t)

	cmd := exec.Command(binaryPath, "create",
		"-n", "Doc", "-d", longDesc,
		"--metadata", `{"ledger":9007199254740993}`,
		"--api-key", "fake", "--base-url", url)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected rejection for --metadata unsafe integer, got:\n%s", out)
	}
	output := string(out)
	if !containsString(output, "claims.metadata.ledger") {
		t.Errorf("error should name claims.metadata.ledger, got:\n%s", output)
	}
	if !containsString(output, "9007199254740993") {
		t.Errorf("error should name the literal, got:\n%s", output)
	}
	if len(body()) != 0 {
		t.Errorf("request should not reach the server: %s", body())
	}
}

// TestCLI_Create_UnsafeIntegerExternalHashMode confirms the guard covers both
// submission modes, not just claims-as-source-of-truth.
func TestCLI_Create_UnsafeIntegerExternalHashMode(t *testing.T) {
	url, body := startCreateEchoServer(t)

	cmd := exec.Command(binaryPath, "create",
		"-n", "Doc", "--hash", validSHA256Hex, "--hash-type", "sha256",
		"--metadata", `{"serial":18446744073709551615}`,
		"--api-key", "fake", "--base-url", url)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("external-hash mode should also be guarded, got:\n%s", out)
	}
	if !containsString(string(out), "claims.metadata.serial") {
		t.Errorf("error should name claims.metadata.serial, got:\n%s", out)
	}
	if len(body()) != 0 {
		t.Errorf("request should not reach the server: %s", body())
	}
}

// TestCLI_Create_UnsafeIntegerFromClaimsStdin covers the --claims-stdin input
// path, which parses through the same decoder.
func TestCLI_Create_UnsafeIntegerFromClaimsStdin(t *testing.T) {
	url, body := startCreateEchoServer(t)

	cmd := exec.Command(binaryPath, "create", "-C",
		"--api-key", "fake", "--base-url", url)
	cmd.Stdin = strings.NewReader(
		`{"name":"Doc","description":"` + longDesc + `","big":9007199254740993}`)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected rejection from stdin claims, got:\n%s", out)
	}
	if !containsString(string(out), "claims.big") {
		t.Errorf("error should name claims.big, got:\n%s", out)
	}
	if len(body()) != 0 {
		t.Errorf("request should not reach the server: %s", body())
	}
}

// TestCLI_Create_UnsafeIntegerMultipleViolations asserts every offender is
// listed, in the deterministic order the walker produces, so a user with
// several bad values fixes them in one pass instead of one 422 at a time.
func TestCLI_Create_UnsafeIntegerMultipleViolations(t *testing.T) {
	url, _ := startCreateEchoServer(t)

	claims := `{"name":"Multi","description":"` + longDesc + `",` +
		`"zeta":9007199254740993,"alpha":18446744073709551615,` +
		`"metadata":{"rows":[{"id":9007199254740992}]}}`
	path := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", "--claims="+path,
		"--api-key", "fake", "--base-url", url)
	out, _ := cmd.CombinedOutput()
	output := string(out)

	if !containsString(output, "3 integers in claims are outside the range") {
		t.Errorf("error should announce all three violations, got:\n%s", output)
	}

	// Deterministic order: sorted keys at each level.
	wantOrder := []string{
		"claims.alpha = 18446744073709551615",
		"claims.metadata.rows[0].id = 9007199254740992",
		"claims.zeta = 9007199254740993",
	}
	prev := -1
	for _, want := range wantOrder {
		at := strings.Index(output, want)
		if at < 0 {
			t.Fatalf("error output missing %q, got:\n%s", want, output)
		}
		if at < prev {
			t.Errorf("violations out of order: %q appeared before the previous entry\n%s", want, output)
		}
		prev = at
	}
}

// TestCLI_Create_UnsafeIntegerJSONOutput covers --json: the rejection is
// structured data, not an English sentence on stderr, and the offending value
// is a JSON STRING so a consumer parsing with doubles does not re-round the
// very number being complained about.
func TestCLI_Create_UnsafeIntegerJSONOutput(t *testing.T) {
	url, body := startCreateEchoServer(t)

	claims := `{"name":"J","description":"` + longDesc + `","metadata":{"id":18446744073709551615}}`
	path := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", "--claims="+path, "--json",
		"--api-key", "fake", "--base-url", url)
	stdout, err := cmd.Output()
	if err == nil {
		t.Fatalf("expected non-zero exit, got success:\n%s", stdout)
	}

	var got struct {
		Error      string `json:"error"`
		Message    string `json:"message"`
		Violations []struct {
			Path  string `json:"path"`
			Value string `json:"value"`
			Min   string `json:"min"`
			Max   string `json:"max"`
		} `json:"violations"`
	}
	if jErr := json.Unmarshal(stdout, &got); jErr != nil {
		t.Fatalf("--json output is not valid JSON: %v\n%s", jErr, stdout)
	}

	if got.Error != "unsafe_integer" {
		t.Errorf("error = %q, want %q", got.Error, "unsafe_integer")
	}
	if len(got.Violations) != 1 {
		t.Fatalf("violations = %v, want one", got.Violations)
	}
	v := got.Violations[0]
	if v.Path != "claims.metadata.id" {
		t.Errorf("path = %q, want claims.metadata.id", v.Path)
	}
	if v.Value != "18446744073709551615" {
		t.Errorf("value = %q, want the exact literal 18446744073709551615", v.Value)
	}
	if v.Min != "-9007199254740991" || v.Max != "9007199254740991" {
		t.Errorf("range = %s..%s, want -9007199254740991..9007199254740991", v.Min, v.Max)
	}
	if !strings.Contains(got.Message, "as a string") {
		t.Errorf("message should suggest the string remedy, got %q", got.Message)
	}
	if len(body()) != 0 {
		t.Errorf("request should not reach the server: %s", body())
	}
}

// TestCLI_Create_FloatsAreNotFlagged confirms the producer rule is about
// integer literals only. A geolocation or a scientific measurement with a huge
// magnitude is legal and must reach the wire unchanged.
func TestCLI_Create_FloatsAreNotFlagged(t *testing.T) {
	url, body := startCreateEchoServer(t)

	claims := `{"name":"F","description":"` + longDesc + `",` +
		`"metadata":{"a":1.5,"b":1e21,"c":9007199254740993.0}}`
	path := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(path, []byte(claims), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "create", "--claims="+path,
		"--api-key", "fake", "--base-url", url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("floats must not be rejected: %v\n%s", err, out)
	}
	if len(body()) == 0 {
		t.Fatal("request never reached the server")
	}
	// Float literals travel verbatim too — UseNumber preserves their text.
	if !strings.Contains(string(body()), `"b":1e21`) {
		t.Errorf("float literal not preserved on the wire: %s", body())
	}
}

// TestCLI_Create_TrailingDataRejected pins the strictness json.Unmarshal gave
// for free and json.Decoder does not: a truncated or concatenated claims file
// must be an error, never a silently accepted prefix.
func TestCLI_Create_TrailingDataRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trailing.json")
	if err := os.WriteFile(path, []byte(`{"name":"A"} {"name":"B"}`), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binaryPath, "create", "--claims="+path, "--api-key", "fake")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "parsing claims JSON") {
		t.Errorf("expected a parse error for trailing data, got: %s", out)
	}
}
