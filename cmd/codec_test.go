// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_Encode_StdinToHex covers the default encode path (stdin → hex).
func TestCLI_Encode_StdinToHex(t *testing.T) {
	cmd := exec.Command(binaryPath, "encode")
	cmd.Stdin = strings.NewReader("hello")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "68656c6c6f" {
		t.Errorf("got %q, want 68656c6c6f", out)
	}
}

// TestCLI_Encode_ToBase64 covers explicit --to base64.
func TestCLI_Encode_ToBase64(t *testing.T) {
	cmd := exec.Command(binaryPath, "encode", "--to", "base64")
	cmd.Stdin = strings.NewReader("foobar")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "Zm9vYmFy" {
		t.Errorf("got %q", out)
	}
}

// TestCLI_Decode_Base64 covers the default decode path (base64 → binary).
// The fact that the stdout binary matches the plaintext confirms
// end-to-end behaviour.
func TestCLI_Decode_Base64(t *testing.T) {
	cmd := exec.Command(binaryPath, "decode", "--from", "base64")
	cmd.Stdin = strings.NewReader("Zm9vYmFy")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "foobar" {
		t.Errorf("got %q, want %q", out, "foobar")
	}
}

// TestCLI_EncodeDecode_RoundTrip pipes a binary fixture through encode and
// decode to confirm bytes-identical recovery. Fixture bytes include the
// full 0x00–0xff range to surface any sign-extension or text-mode bugs.
func TestCLI_EncodeDecode_RoundTrip(t *testing.T) {
	var fixture [256]byte
	for i := range fixture {
		fixture[i] = byte(i)
	}
	path := filepath.Join(t.TempDir(), "all-bytes.bin")
	if err := os.WriteFile(path, fixture[:], 0644); err != nil {
		t.Fatal(err)
	}

	// encode --to base64url.
	enc := exec.Command(binaryPath, "encode", "--to", "base64url", path)
	encoded, err := enc.Output()
	if err != nil {
		t.Fatal(err)
	}
	// decode --from base64url.
	dec := exec.Command(binaryPath, "decode", "--from", "base64url")
	dec.Stdin = bytes.NewReader(encoded)
	decoded, err := dec.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, fixture[:]) {
		t.Errorf("round-trip mismatch: first 8 bytes: %x vs %x", decoded[:8], fixture[:8])
	}
}

// TestCLI_Encode_TextToText covers the cross-encoding path
// (hex → base64) without an intermediate binary write.
func TestCLI_Encode_TextToText(t *testing.T) {
	cmd := exec.Command(binaryPath, "encode", "--from", "hex", "--to", "base64")
	cmd.Stdin = strings.NewReader("68656c6c6f")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "aGVsbG8=" {
		t.Errorf("got %q, want aGVsbG8=", out)
	}
}

// TestCLI_JCS_CanonicalizesJSON tests that RFC 8785 sorts keys.
func TestCLI_JCS_CanonicalizesJSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "jcs")
	cmd.Stdin = strings.NewReader(`{"b":2,"a":1,"c":[3,1,2]}`)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"a":1,"b":2,"c":[3,1,2]}` {
		t.Errorf("got %q", out)
	}
}

// TestCLI_JCS_HashEquivalence confirms that 'jcs | hash' produces the
// same digest as 'hash --jcs', the primary design invariant for the
// flagship "recompute a claims hash" use case.
func TestCLI_JCS_HashEquivalence(t *testing.T) {
	input := `{"b":2,"a":1}`
	// jcs | hash
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX sh available")
	}
	pipedCmd := exec.Command(sh, "-c", binaryPath+" jcs | "+binaryPath+
		" hash -a sha256 --style bare")
	pipedCmd.Stdin = strings.NewReader(input)
	piped, err := pipedCmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	// hash --jcs (built-in shortcut)
	direct := exec.Command(binaryPath, "hash", "--jcs",
		"-a", "sha256", "--style", "bare", "--no-filename")
	direct.Stdin = strings.NewReader(input)
	directOut, err := direct.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(piped)) != strings.TrimSpace(string(directOut)) {
		t.Errorf("JCS paths disagree:\n  piped:  %q\n  direct: %q", piped, directOut)
	}
}

// --- Oversized integers (Appendix C.2a / E.4) -------------------------------

// TestCLI_JCS_PreservesOversizedInteger pins Appendix C.2a's normative pair:
// 2^53 is exactly representable and 2^53 + 1 is not, and the two must
// canonicalize to distinct strings. `truestamp jcs` is documented as the way
// to recompute a claims_hash locally, so a strict RFC 8785 round-trip through
// a float64 here silently returned the wrong digest for the second row.
func TestCLI_JCS_PreservesOversizedInteger(t *testing.T) {
	cases := []struct {
		in         string
		want       string
		wantWarn   bool
		warnNumber string
	}{
		{`{"n":9007199254740992}`, `{"n":9007199254740992}`, false, ""},
		{`{"n":9007199254740993}`, `{"n":9007199254740993}`, true, "9007199254740993"},
		{`{"n":-9007199254740993}`, `{"n":-9007199254740993}`, true, "-9007199254740993"},
		{`{"n":18446744073709551615}`, `{"n":18446744073709551615}`, true, "18446744073709551615"},
		// Floats are lossy by construction and are never flagged.
		{`{"n":1.5}`, `{"n":1.5}`, false, ""},
	}
	for _, c := range cases {
		cmd := exec.Command(binaryPath, "jcs")
		cmd.Stdin = strings.NewReader(c.in)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s: %v (stderr: %s)", c.in, err, stderr.String())
		}
		if stdout.String() != c.want {
			t.Errorf("%s: got %q, want %q", c.in, stdout.String(), c.want)
		}
		gotWarn := strings.Contains(stderr.String(), "not portably verifiable")
		if gotWarn != c.wantWarn {
			t.Errorf("%s: portability warning = %v, want %v (stderr: %q)",
				c.in, gotWarn, c.wantWarn, stderr.String())
		}
		if c.wantWarn && !strings.Contains(stderr.String(), c.warnNumber) {
			t.Errorf("%s: warning should name %s, got: %q", c.in, c.warnNumber, stderr.String())
		}
	}
}

// TestCLI_JCS_OversizedWarningIsAdvisory: the digest above the warning is the
// correct one, so the warning must not move the exit code, and it must stay
// off stdout so `truestamp jcs | truestamp hash` is unaffected.
func TestCLI_JCS_OversizedWarningIsAdvisory(t *testing.T) {
	cmd := exec.Command(binaryPath, "jcs")
	cmd.Stdin = strings.NewReader(`{"n":9007199254740993}`)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("exit should stay 0 for an advisory warning, got %v", err)
	}
	if strings.Contains(stdout.String(), "warning") {
		t.Errorf("the warning leaked onto stdout: %q", stdout.String())
	}
}

// TestCLI_JCS_OversizedSuppressedUnderSilentAndJSON: --silent means no
// output at all, and --json routes the same fact through a structured field
// so machine consumers do not have to scrape stderr.
func TestCLI_JCS_OversizedSuppressedUnderSilentAndJSON(t *testing.T) {
	const input = `{"n":9007199254740993}`

	silent := exec.Command(binaryPath, "jcs", "--silent")
	silent.Stdin = strings.NewReader(input)
	var sOut, sErr bytes.Buffer
	silent.Stdout = &sOut
	silent.Stderr = &sErr
	if err := silent.Run(); err != nil {
		t.Fatalf("--silent: %v", err)
	}
	if sOut.Len() != 0 || sErr.Len() != 0 {
		t.Errorf("--silent must emit nothing, got stdout=%q stderr=%q", sOut.String(), sErr.String())
	}

	jsonCmd := exec.Command(binaryPath, "jcs", "--json")
	jsonCmd.Stdin = strings.NewReader(input)
	var jOut, jErr bytes.Buffer
	jsonCmd.Stdout = &jOut
	jsonCmd.Stderr = &jErr
	if err := jsonCmd.Run(); err != nil {
		t.Fatalf("--json: %v", err)
	}
	if jErr.Len() != 0 {
		t.Errorf("--json must not write the warning to stderr, got %q", jErr.String())
	}
	var parsed struct {
		Output            string   `json:"output"`
		OversizedIntegers []string `json:"oversized_integers"`
	}
	if err := json.Unmarshal(jOut.Bytes(), &parsed); err != nil {
		t.Fatalf("parsing --json output: %v\n%s", err, jOut.String())
	}
	if parsed.Output != `{"n":9007199254740993}` {
		t.Errorf("output: got %q", parsed.Output)
	}
	if len(parsed.OversizedIntegers) != 1 || parsed.OversizedIntegers[0] != "9007199254740993" {
		t.Errorf("oversized_integers: got %v, want [9007199254740993]", parsed.OversizedIntegers)
	}
}

// TestCLI_JCS_NoOversizedFieldWhenSafe: the field is omitempty so a document
// inside the safe range produces no key at all, not an empty array a consumer
// would have to distinguish from "not reported".
func TestCLI_JCS_NoOversizedFieldWhenSafe(t *testing.T) {
	cmd := exec.Command(binaryPath, "jcs", "--json")
	cmd.Stdin = strings.NewReader(`{"b":2,"a":1}`)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("oversized_integers")) {
		t.Errorf("safe document should carry no oversized_integers key, got: %s", out)
	}
}

// TestCLI_Encode_JSON_Shape confirms the --json envelope for encode.
func TestCLI_Encode_JSON_Shape(t *testing.T) {
	cmd := exec.Command(binaryPath, "encode", "--to", "hex", "--json")
	cmd.Stdin = strings.NewReader("abc")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, out)
	}
	if m["from"] != "binary" {
		t.Errorf("from: got %v, want binary", m["from"])
	}
	if m["to"] != "hex" {
		t.Errorf("to: got %v, want hex", m["to"])
	}
	if m["output"] != "616263" {
		t.Errorf("output: got %v, want 616263", m["output"])
	}
}

// TestCLI_Decode_RejectsCrossEncoding confirms that decode validates the
// input matches the declared --from (no silent mislabel acceptance).
func TestCLI_Decode_RejectsCrossEncoding(t *testing.T) {
	// "Zm9_" uses a base64url-specific '_' char; rejecting it on
	// --from base64 guarantees the two encodings stay distinct.
	cmd := exec.Command(binaryPath, "decode", "--from", "base64")
	cmd.Stdin = strings.NewReader("Zm9_YmFy")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected error for URL-safe chars under --from base64")
	}
}
