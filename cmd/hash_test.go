// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestCLI_Hash_StdinABC cross-validates the CLI output for SHA-256("abc")
// against the canonical NIST FIPS 180-4 vector. If this ever drifts, every
// user's scripted output changes, catch it the moment it breaks.
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
	// JCS of {"b":2,"a":1} is {"a":1,"b":2} (13 bytes including braces).
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
// `jcs` + `hash --prefix 0x11` produce, i.e. that the two forms
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

// `truestamp hash --jcs` refuses an integer of magnitude above
// MAX_INTEGER and below 10^21 as rule J5 does, in each of J5's spellings,
// naming the input and malformed_json (number), and prints no digest; it
// hashes MAX_INTEGER itself, sha256(0x11 || {"n":9007199254740991}). Of
// two inputs, one refused, the other is still hashed. Mutant killed:
// hashOne canonicalizing with gowebpki/jcs's Transform (each refused
// spelling is hashed, exit 0).
func TestCLI_Hash_JCS_RefusesAnIntegerPastMaxInteger(t *testing.T) {
	for _, in := range []string{`{"n":9007199254740992}`, `{"n":9007199254740992.0}`, `{"n":1e16}`} {
		cmd := exec.Command(binaryPath, "hash", "--prefix", "0x11", "--jcs", "--style", "bare", "--no-filename")
		cmd.Stdin = strings.NewReader(in)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err == nil || stdout.Len() != 0 || !strings.Contains(stderr.String(), "malformed_json (number)") {
			t.Errorf("%s: %v, stdout %q, stderr %q", in, err, stdout.String(), stderr.String())
		}
	}
	cmd := exec.Command(binaryPath, "hash", "--prefix", "0x11", "--jcs", "--style", "bare", "--no-filename")
	cmd.Stdin = strings.NewReader(`{"n": 9007199254740991}`)
	out, err := cmd.Output()
	sum := sha256.Sum256([]byte("\x11" + `{"n":9007199254740991}`))
	if err != nil || strings.TrimSpace(string(out)) != hex.EncodeToString(sum[:]) {
		t.Errorf("MAX_INTEGER: %v, %q", err, out)
	}

	dir := t.TempDir()
	big, safe := filepath.Join(dir, "big.json"), filepath.Join(dir, "safe.json")
	if err := os.WriteFile(big, []byte(`{"n":9007199254740993}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(safe, []byte(`{"n":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command(binaryPath, "hash", "--jcs", big, safe)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	if err == nil || !strings.Contains(stderr.String(), "big.json: JCS canonicalization: malformed_json (number)") ||
		strings.Contains(stdout.String(), "big.json") || !strings.Contains(stdout.String(), "safe.json") {
		t.Errorf("two inputs: %v, stdout %q, stderr %q", err, stdout.String(), stderr.String())
	}
}

// --jcs reads a number as its nearest double, so 1 followed by 800 zeros
// and e-800 hashes as [1] does. Mutant killed: the strict reader placing a
// number's point by its first 800 digits, as strconv.ParseFloat's decimal
// fallback does (the spelling hashes as [0.1]).
func TestCLI_Hash_JCS_ANumberIsItsNearestDouble(t *testing.T) {
	digest := func(in string) string {
		t.Helper()
		cmd := exec.Command(binaryPath, "hash", "--jcs", "--style", "bare", "--no-filename")
		cmd.Stdin = strings.NewReader(in)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%.20s: %v", in, err)
		}
		return strings.TrimSpace(string(out))
	}
	sum := sha256.Sum256([]byte("[1]"))
	if got := digest("[1" + strings.Repeat("0", 800) + "e-800]"); got != hex.EncodeToString(sum[:]) {
		t.Errorf("digest %s, want sha256([1]) %x", got, sum)
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

// The claims_hash examples of `hash --help`, `jcs --help`, README.md and
// EXAMPLES.md give the claims_hash byte of SPEC.md section 4.3: each
// `--prefix 0xNN` they show, run with --jcs over the claims of the vector
// item proof, prints that proof's /proof/subject/record/claims_hash.
// Mutants killed: 0x11, item_hash's byte, restored in any one of the four
// (that example prints another digest, not the proof's claims_hash); an
// EXAMPLES.md claims_hash example dropped (the count changes).
func TestCLI_ClaimsHashExamplesGiveTheProofsClaimsHash(t *testing.T) {
	var doc struct {
		Proof struct {
			Subject struct {
				Claims jsontext.Value `json:"claims"`
				Record struct {
					ClaimsHash string `json:"claims_hash"`
				} `json:"record"`
			} `json:"subject"`
		} `json:"proof"`
	}
	if err := json.Unmarshal(readVector(t, "positive/item.proof.json"), &doc); err != nil {
		t.Fatal(err)
	}
	claims, want := doc.Proof.Subject.Claims, doc.Proof.Subject.Record.ClaimsHash
	if len(claims) == 0 || len(want) != 64 {
		t.Fatalf("vector item proof: claims %q, claims_hash %q", claims, want)
	}

	hashHelp, _, code := runCLI(t, "hash", "--help")
	if code != 0 {
		t.Fatalf("hash --help exited %d", code)
	}
	jcsHelp, _, code := runCLI(t, "jcs", "--help")
	if code != 0 {
		t.Fatalf("jcs --help exited %d", code)
	}
	readme, err := os.ReadFile(filepath.Join(repoRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	examples, err := os.ReadFile(filepath.Join(repoRoot(t), "EXAMPLES.md"))
	if err != nil {
		t.Fatal(err)
	}

	prefixArg := regexp.MustCompile(`--prefix (0x[0-9A-Fa-f]{2})\b`)
	for _, src := range []struct {
		name, text string
		examples   int
	}{
		{"hash --help", hashHelp, 1},
		{"jcs --help", jcsHelp, 2},
		{"README.md", string(readme), 2},
		{"EXAMPLES.md", string(examples), 6},
	} {
		found := prefixArg.FindAllStringSubmatch(src.text, -1)
		if len(found) != src.examples {
			t.Errorf("%s shows %d --prefix examples, want %d", src.name, len(found), src.examples)
		}
		for _, m := range found {
			cmd := exec.Command(binaryPath, "hash", "--prefix", m[1], "--jcs",
				"-a", "sha256", "--style", "bare", "--no-filename")
			cmd.Env = cleanEnv()
			cmd.Stdin = bytes.NewReader(claims)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("%s: hash --prefix %s --jcs: %v", src.name, m[1], err)
			}
			if got := strings.TrimSpace(string(out)); got != want {
				t.Errorf("%s: --prefix %s gives %s over the vector claims, want the proof's claims_hash %s",
					src.name, m[1], got, want)
			}
		}
	}
}
