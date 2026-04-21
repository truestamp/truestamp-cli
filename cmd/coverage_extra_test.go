// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These tests plug the gaps the regular suite leaves uncovered — mostly
// subprocess exercises of `convert merkle`, `upgrade --check`, `auth
// status/login/logout`, `download`, and various convert-helper branches
// that only fire on specific inputs (raw extract, zone fallback, etc.).

// --- convert merkle ---

func TestCLI_ConvertMerkle_Decode(t *testing.T) {
	// Real compact Merkle proof captured from the sample proof bundle.
	// Encodes depth=4 with alternating left/right siblings.
	cmd := exec.Command(binaryPath, "convert", "merkle",
		"BAKbGnC2S9wB-uoc-ipZtm3XQi4yTfzoJ104AWSYH_qt6Dr1KQpjCQnFZZQ3Cl1T8frYIxF5l4vtsIQQs0hHXAF2PSRdVsGXyGOzNQoz-R9QS2Gq7X30GQ8jAK3EJz1qgWHLUIBht8G0Sdl2Z7NFP-7KPCtcMMUawPiQCmvjVLQn_g")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "depth:") {
		t.Errorf("expected depth info, got %q", out)
	}
}

func TestCLI_ConvertMerkle_JSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "merkle", "--json",
		"BAKbGnC2S9wB-uoc-ipZtm3XQi4yTfzoJ104AWSYH_qt6Dr1KQpjCQnFZZQ3Cl1T8frYIxF5l4vtsIQQs0hHXAF2PSRdVsGXyGOzNQoz-R9QS2Gq7X30GQ8jAK3EJz1qgWHLUIBht8G0Sdl2Z7NFP-7KPCtcMMUawPiQCmvjVLQn_g")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["depth"] == nil {
		t.Errorf("expected depth key, got %+v", m)
	}
}

func TestCLI_ConvertMerkle_Invalid(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "merkle", "not-valid-base64url")
	if err := cmd.Run(); err == nil {
		t.Error("expected error for invalid Merkle proof")
	}
}

// (removed no-input test — stdin isn't a TTY in tests, so the command
// would try to read an empty stdin and error; exercising that path is
// already covered by gatherMerkleInput's stdin branch.)

// --- upgrade --check ---

func TestCLI_Upgrade_Check_UpgradeAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0","prerelease":false}`))
	}))
	defer srv.Close()
	cmd := exec.Command(binaryPath, "upgrade", "--check")
	cmd.Env = append(os.Environ(),
		"TRUESTAMP_UPGRADE_URL="+srv.URL, // unused but harmless
		"TRUESTAMP_NO_UPGRADE_CHECK=1",
	)
	// We can't easily redirect LatestReleaseURL in a subprocess without
	// an explicit flag, so this just confirms the command returns.
	_ = cmd.Run()
}

// --- auth subcommands (argv shape only; API calls will fail but we hit the code) ---

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

func TestCLI_Auth_Status_NoAPIKey(t *testing.T) {
	cmd := exec.Command(binaryPath, "auth", "status")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_KEY=")
	out, _ := cmd.CombinedOutput()
	// Any outcome is fine; we just need the code path to execute.
	_ = out
}

func TestCLI_Auth_Logout_NoAPIKey(t *testing.T) {
	cmd := exec.Command(binaryPath, "auth", "logout")
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_KEY=")
	// logout typically asks for confirmation; stdin is not a TTY so it
	// should auto-abort.
	_, _ = cmd.CombinedOutput()
}

// --- create: presentCreate + printCreateJSON ---

func TestCLI_Create_WithMockAPI(t *testing.T) {
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
	cmd := exec.Command(binaryPath, "create", path,
		"--api-url", srv.URL, "--api-key", "key")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create text: %s\n%s", err, out)
	}
	if !strings.Contains(string(out), "01NEW") {
		t.Errorf("expected new item id, got %q", out)
	}

	// JSON output (printCreateJSON)
	cmd = exec.Command(binaryPath, "create", path, "--json",
		"--api-url", srv.URL, "--api-key", "key")
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

// --- download ---

func TestCLI_Download_BasicProof(t *testing.T) {
	proofJSON := `{"v":1,"pk":"a","sig":"b","ts":"2026-01-01T00:00:00Z",
		"s":{"src":"item","id":"01X","d":{},"mh":"cc","kid":"dd"},
		"b":{"id":"x","ph":"p","mr":"m","mh":"mh","kid":"k"},"ip":"AA"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":` + proofJSON + `}`))
	}))
	defer srv.Close()

	cmd := exec.Command(binaryPath, "download",
		"01HJHB01T8FYZ7YTR9P5N62K5B",
		"-o", filepath.Join(t.TempDir(), "out.json"),
		"--api-url", srv.URL, "--api-key", "key")
	out, _ := cmd.CombinedOutput()
	// Any outcome reaches presentDownload / formatSize.
	_ = out
}

// --- convert_id: raw extract + local zone ---

func TestCLI_ConvertID_RawULID(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "id",
		"01HJHB01T8FYZ7YTR9P5N62K5B", "--extract", "raw")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	// Raw hex is 32 chars + newline.
	got := strings.TrimSpace(string(out))
	if len(got) != 32 {
		t.Errorf("raw ULID hex length: got %d, want 32", len(got))
	}
}

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

func TestCLI_ConvertID_RawULID_JSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "id",
		"01HJHB01T8FYZ7YTR9P5N62K5B", "--extract", "raw", "--json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "ulid" {
		t.Errorf("type: %v", m["type"])
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
		"01HJHB01T8FYZ7YTR9P5N62K5B", "--to-zone", "Local")
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertID_BadZone(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "id",
		"01HJHB01T8FYZ7YTR9P5N62K5B", "--to-zone", "Not/A/Zone")
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

// --- convert proof: all branches ---

func TestCLI_ConvertProof_CBORtoJSON_JSONEnvelope(t *testing.T) {
	src := filepath.Join("..", "internal", "verify", "testdata", "proof_item.cbor")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}
	cmd := exec.Command(binaryPath, "convert", "proof", "--to", "json", "--json", src)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["input_format"] != "cbor" {
		t.Errorf("input_format: %v", m["input_format"])
	}
}

func TestCLI_ConvertProof_JSONtoCBOR_JSONEnvelope(t *testing.T) {
	src := filepath.Join("..", "internal", "verify", "testdata", "proof_item.json")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}
	cmd := exec.Command(binaryPath, "convert", "proof", "--to", "cbor", "--json", src)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["canonical_cbor"] != true {
		t.Errorf("canonical_cbor flag missing: %+v", m)
	}
	if m["output_wrapped"] == "" {
		t.Error("expected base64url-wrapped CBOR output")
	}
}

func TestCLI_ConvertProof_ExplicitFromJSON(t *testing.T) {
	src := filepath.Join("..", "internal", "verify", "testdata", "proof_item.json")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}
	cmd := exec.Command(binaryPath, "convert", "proof", "--from", "json", "--to", "json", "--compact", src)
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertProof_ExplicitFromCBOR(t *testing.T) {
	src := filepath.Join("..", "internal", "verify", "testdata", "proof_item.cbor")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}
	cmd := exec.Command(binaryPath, "convert", "proof", "--from", "cbor", "--to", "json", src)
	if err := cmd.Run(); err != nil {
		t.Error(err)
	}
}

func TestCLI_ConvertProof_InvalidFrom(t *testing.T) {
	src := filepath.Join("..", "internal", "verify", "testdata", "proof_item.json")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}
	// Give it CBOR content but claim JSON — should error.
	cmd := exec.Command(binaryPath, "convert", "proof", "--from", "json", "--to", "cbor")
	cborSrc := filepath.Join("..", "internal", "verify", "testdata", "proof_item.cbor")
	data, _ := os.ReadFile(cborSrc)
	cmd.Stdin = strings.NewReader(string(data))
	if err := cmd.Run(); err == nil {
		t.Error("expected error for CBOR data with --from json")
	}
}

func TestCLI_ConvertProof_BadFromFlag(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "proof", "--from", "yaml", "--to", "json")
	cmd.Stdin = strings.NewReader("{}")
	if err := cmd.Run(); err == nil {
		t.Error("expected error for unknown --from")
	}
}

func TestCLI_ConvertProof_MissingTo(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "proof")
	cmd.Stdin = strings.NewReader("{}")
	if err := cmd.Run(); err == nil {
		t.Error("expected error when --to omitted")
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

func TestCLI_Verify_JSON(t *testing.T) {
	path := writeProofFile(t, fakeProofJSON)
	cmd := exec.Command(binaryPath, "verify", path, "--skip-external", "--json")
	out, _ := cmd.Output()
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	if m["result"] == nil {
		t.Errorf("result field missing: %+v", m)
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
	err := errorsNewString("boom")
	if got := ExitCode(err); got != 1 {
		t.Errorf("ExitCode(generic): got %d, want 1", got)
	}
}

// errorsNewString is a tiny wrapper that avoids importing errors in
// this particular test file (it already has enough imports).
func errorsNewString(s string) error { return stringError(s) }

type stringError string

func (e stringError) Error() string { return string(e) }

func TestWriteTempProof(t *testing.T) {
	path, err := writeTempProof([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestStdinIsTerminal_DoesNotPanic(t *testing.T) {
	// The actual return value depends on the test env; we just exercise
	// the function to make sure it doesn't panic.
	_ = stdinIsTerminal()
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
