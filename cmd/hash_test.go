// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_Hash_StdinABC cross-validates the CLI output for SHA-256("abc")
// against the canonical NIST FIPS 180-4 vector. If this ever drifts, every
// user's scripted output changes — catch it the moment it breaks.
func TestCLI_Hash_StdinABC(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "-a", "sha256")
	cmd.Stdin = strings.NewReader("abc")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hash failed: %s\n%s", err, out)
	}
	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad  -\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", string(out), want)
	}
}

// TestCLI_Hash_EmptyInput validates the canonical empty-input digest for
// every supported algorithm. Catches off-by-one init bugs that only
// surface on zero-length input.
func TestCLI_Hash_EmptyInput(t *testing.T) {
	cases := map[string]string{
		"md5":         "d41d8cd98f00b204e9800998ecf8427e",
		"sha1":        "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		"sha256":      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"sha512":      "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
		"sha3-256":    "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a",
		"blake2s-256": "69217a3079908094e11121d042354a7c1f55b6482ca1a51e1b250dfd1ed0eef9",
	}
	for algo, wantHex := range cases {
		t.Run(algo, func(t *testing.T) {
			cmd := exec.Command(binaryPath, "hash", "-a", algo, "--style", "bare", "--no-filename")
			cmd.Stdin = strings.NewReader("")
			// Use Output() so the legacy-warning stderr from MD5/SHA-1
			// doesn't pollute the stdout digest comparison.
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("hash -a %s failed: %s", algo, err)
			}
			got := strings.TrimSpace(string(out))
			if got != wantHex {
				t.Errorf("%s empty-digest: got %s, want %s", algo, got, wantHex)
			}
		})
	}
}

// TestCLI_Hash_MatchesSystemTool cross-checks our output against whichever
// of sha256sum / shasum / openssl dgst is installed. Skipped if none are.
// Byte-identical output is the defining feature of --style gnu default.
func TestCLI_Hash_MatchesSystemTool(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.bin")
	if err := os.WriteFile(path, []byte("The quick brown fox jumps over the lazy dog"), 0644); err != nil {
		t.Fatal(err)
	}

	ours := mustRun(t, binaryPath, "hash", path)

	// sha256sum prints the OS path as given to it. Our gnu style matches.
	if _, err := exec.LookPath("sha256sum"); err == nil {
		theirs := mustRun(t, "sha256sum", path)
		if ours != theirs {
			t.Errorf("sha256sum mismatch:\n  ours:   %q\n  system: %q", ours, theirs)
		}
		return
	}
	if _, err := exec.LookPath("shasum"); err == nil {
		theirs := mustRun(t, "shasum", "-a", "256", path)
		if ours != theirs {
			t.Errorf("shasum -a 256 mismatch:\n  ours:   %q\n  system: %q", ours, theirs)
		}
		return
	}
	t.Skip("neither sha256sum nor shasum found; skipping cross-check")
}

func mustRun(t *testing.T, name string, args ...string) string {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %s\n%s", name, args, err, out)
	}
	return string(out)
}

// TestCLI_Hash_MD5Warning verifies the stderr warning for legacy algorithms
// appears (and only appears) when output is for humans, not JSON or silent.
func TestCLI_Hash_MD5Warning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.bin")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	// Human output: warning present on stderr.
	cmd := exec.Command(binaryPath, "hash", "-a", "md5", path)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("hash -a md5 failed: %s", err)
	}
	if !strings.Contains(stderr.String(), "cryptographically broken") {
		t.Errorf("stderr should contain legacy warning, got: %q", stderr.String())
	}

	// Silent mode: no warning, no output.
	cmd = exec.Command(binaryPath, "hash", "-a", "md5", "--silent", path)
	var stderr2, stdout strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr2
	if err := cmd.Run(); err != nil {
		t.Fatalf("hash -a md5 --silent failed: %s", err)
	}
	if stderr2.Len() != 0 {
		t.Errorf("silent mode should emit no stderr, got: %q", stderr2.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("silent mode should emit no stdout, got: %q", stdout.String())
	}
}

// TestCLI_Hash_JSONShape checks the --json output has the documented keys.
func TestCLI_Hash_JSONShape(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "-a", "sha256", "--json")
	cmd.Stdin = strings.NewReader("abc")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("hash --json failed: %s", err)
	}
	var result struct {
		Algorithm string `json:"algorithm"`
		Digest    struct {
			Hex       string `json:"hex"`
			Base64    string `json:"base64"`
			Base64URL string `json:"base64url"`
		} `json:"digest"`
		SizeBytes int64 `json:"size_bytes"`
		Input     struct {
			Type string `json:"type"`
		} `json:"input"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("unmarshaling JSON: %v\nraw: %s", err, out)
	}
	if result.Algorithm != "sha256" {
		t.Errorf("algorithm: got %q, want sha256", result.Algorithm)
	}
	if result.Digest.Hex != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Errorf("digest.hex wrong: %s", result.Digest.Hex)
	}
	if result.SizeBytes != 3 {
		t.Errorf("size_bytes: got %d, want 3", result.SizeBytes)
	}
	if result.Input.Type != "stdin" {
		t.Errorf("input.type: got %q, want stdin", result.Input.Type)
	}
}

// TestCLI_Hash_PrefixAndJCS cross-validates the Truestamp-domain
// SHA-256(0x11 || JCS({"a":1,"b":2})) computation end-to-end. The
// expected digest was computed independently using `echo` + `jq -c` +
// `xxd` + `sha256sum` to confirm the pipeline composition.
func TestCLI_Hash_PrefixAndJCS(t *testing.T) {
	// JCS of {"b":2,"a":1} is {"a":1,"b":2} (26 bytes including braces).
	cmd := exec.Command(binaryPath, "hash", "--prefix", "0x11", "--jcs",
		"-a", "sha256", "--style", "bare", "--no-filename")
	cmd.Stdin = strings.NewReader(`{"b":2,"a":1}`)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("hash --prefix --jcs failed: %s", err)
	}
	digest := strings.TrimSpace(string(out))
	// Computed as sha256(0x11 || '{"a":1,"b":2}'). The hex string is
	// stable and any change here signals either a JCS regression or a
	// domain-prefix mistake.
	want := "06808c3a0e0f801b5a7ae8c5b27e01f282f8fc0ef4f10dc2c62257cf7d7bac13"
	if digest != want {
		t.Errorf("prefix+jcs digest: got %s, want %s", digest, want)
	}
}

// TestCLI_Hash_PrefixAlone_EqualsJCSSeparately verifies that hashing
// `{"a":1,"b":2}` (already canonical) with --prefix 0x11 matches what
// `jcs` + `hash --prefix 0x11` produce — i.e. that the two forms
// compose.
func TestCLI_Hash_PrefixMatchesJCS(t *testing.T) {
	input := `{"a":1,"b":2}`
	// Path A: hash --prefix 0x11 --jcs (buffered path).
	cmdA := exec.Command(binaryPath, "hash", "--prefix", "0x11", "--jcs",
		"-a", "sha256", "--style", "bare", "--no-filename")
	cmdA.Stdin = strings.NewReader(input)
	outA, err := cmdA.Output()
	if err != nil {
		t.Fatalf("A: %s", err)
	}
	// Path B: jcs | hash --prefix 0x11 (two-process pipeline emulated
	// here by running the inner shell).
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX sh available")
	}
	cmdB := exec.Command(sh, "-c", binaryPath+" jcs | "+binaryPath+
		" hash --prefix 0x11 -a sha256 --style bare")
	cmdB.Stdin = strings.NewReader(input)
	outB, err := cmdB.Output()
	if err != nil {
		t.Fatalf("B: %s", err)
	}
	if strings.TrimSpace(string(outA)) != strings.TrimSpace(string(outB)) {
		t.Errorf("the two JCS paths disagree:\n  buffered: %s  piped:    %s", outA, outB)
	}
}

// TestCLI_Hash_MultipleFiles confirms multi-input output order matches
// the argument order (sha256sum behaviour).
func TestCLI_Hash_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(a, []byte("A"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("B"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binaryPath, "hash", a, b)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("hash multi: %s", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), out)
	}
	if !strings.HasSuffix(lines[0], a) {
		t.Errorf("line 0 should end with %s, got %q", a, lines[0])
	}
	if !strings.HasSuffix(lines[1], b) {
		t.Errorf("line 1 should end with %s, got %q", b, lines[1])
	}
}

// TestCLI_Hash_CRLFDifferentFromLF confirms that hashing bytes-with-CRLF
// yields a different digest than bytes-with-LF. Guards against any
// accidental text-mode translation across platforms.
func TestCLI_Hash_CRLFDifferentFromLF(t *testing.T) {
	dir := t.TempDir()
	crlf := filepath.Join(dir, "crlf.txt")
	lf := filepath.Join(dir, "lf.txt")
	if err := os.WriteFile(crlf, []byte("hello\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lf, []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	a := mustRun(t, binaryPath, "hash", "--style", "bare", crlf)
	b := mustRun(t, binaryPath, "hash", "--style", "bare", lf)
	if a == b {
		t.Errorf("CRLF and LF files produced the same digest: %s", a)
	}
}

// TestCLI_Hash_UnknownAlgorithm_Error verifies a clear error including
// the list of supported algorithms.
func TestCLI_Hash_UnknownAlgorithm_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "-a", "crc32")
	cmd.Stdin = strings.NewReader("abc")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for unknown algorithm")
	}
	if !strings.Contains(string(out), "sha256") {
		t.Errorf("error should list supported algorithms, got: %s", out)
	}
}

// TestCLI_Hash_BSDStyle checks the BSD tagged output format.
func TestCLI_Hash_BSDStyle(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "-a", "sha256", "--style", "bsd")
	cmd.Stdin = strings.NewReader("abc")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := "SHA256 (-) = ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestCLI_Hash_List dumps the supported algorithm table.
func TestCLI_Hash_List(t *testing.T) {
	out := mustRun(t, binaryPath, "hash", "--list")
	for _, expect := range []string{"md5", "sha256", "sha3-256", "blake2b-512"} {
		if !strings.Contains(out, expect) {
			t.Errorf("--list missing %q", expect)
		}
	}
}
