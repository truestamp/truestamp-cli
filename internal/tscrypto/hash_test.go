// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package tscrypto

import (
	"encoding/base64"
	"sort"
	"strings"
	"testing"
)

func TestDomainHash_ClaimsVector(t *testing.T) {
	t.Parallel()
	// Test vector from kb/cryptography/hashing-domain-separation.md:
	// SHA256(0x11 || "truestamp")
	result := BytesToHex(DomainHash(0x11, []byte("truestamp")))
	expected := "9d9443e5133052d9fc837e150d32e3094fc979f097922034a984ddfbe5247aca"
	if result != expected {
		t.Errorf("claims hash: got %s, want %s", result, expected)
	}
}

func TestDomainHash_EntropyVector(t *testing.T) {
	t.Parallel()
	// Test vector: SHA256(0x21 || "truestamp")
	result := BytesToHex(DomainHash(0x21, []byte("truestamp")))
	expected := "22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831"
	if result != expected {
		t.Errorf("entropy hash: got %s, want %s", result, expected)
	}
}

func TestDomainHash_GenesisVector(t *testing.T) {
	t.Parallel()
	// Test vector: SHA256(0x31 || "truestamp")
	result := BytesToHex(DomainHash(0x31, []byte("truestamp")))
	expected := "96965205fa528419d226675f8ad4a11f978d5523e31c462f4af0f1405d2ac8da"
	if result != expected {
		t.Errorf("genesis hash: got %s, want %s", result, expected)
	}
}

func TestDomainHash_DifferentPrefixesDiffer(t *testing.T) {
	t.Parallel()
	data := []byte("same input")
	h11 := BytesToHex(DomainHash(0x11, data))
	h12 := BytesToHex(DomainHash(0x12, data))
	h21 := BytesToHex(DomainHash(0x21, data))
	if h11 == h12 || h11 == h21 || h12 == h21 {
		t.Error("different domain prefixes should produce different hashes")
	}
}

func TestComputeKeyID(t *testing.T) {
	t.Parallel()
	// Test vector: known public key -> key_id
	pubkeyB64 := "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="
	pubkey, _ := base64.StdEncoding.DecodeString(pubkeyB64)
	result := ComputeKeyID(pubkey)
	expected := "4ceefa4a"
	if result != expected {
		t.Errorf("key_id: got %s, want %s", result, expected)
	}
}

// A second key and its key id, so that ComputeKeyID is held to more than
// one answer. Mutant killed: ComputeKeyID under the prefix 0x60 of SPEC.md
// section 3.3's key id (both answers change).
func TestComputeKeyID_OfASecondKey(t *testing.T) {
	t.Parallel()
	pubkey, err := base64.StdEncoding.DecodeString("IVL40Zt5HSRFMkLhXy6rbLfP+ntqXtMAl5YOBpiB2xI=")
	if err != nil {
		t.Fatalf("decoding pk: %s", err)
	}
	if got := ComputeKeyID(pubkey); got != "f2c39df9" {
		t.Errorf("key_id = %s, want f2c39df9", got)
	}
}

// HexEqual folds the case of 'A' to 'F', for a value whose case another
// party chooses, over an 8-character key id and a 128-character digest.
// Mutant killed: foldASCIIHex returning its byte unchanged (both compare
// unequal).
func TestHexEqual_FoldsCase(t *testing.T) {
	t.Parallel()
	if !HexEqual("B47CC0F1", "b47cc0f1") {
		t.Error("an uppercase key id must equal its lowercase spelling")
	}
	if !HexEqual(strings.ToUpper(strings.Repeat("ab", 64)), strings.Repeat("ab", 64)) {
		t.Error("an uppercase digest must equal its lowercase spelling")
	}
}

func TestHexEqual(t *testing.T) {
	t.Parallel()
	if !HexEqual("abcdef", "ABCDEF") {
		t.Error("HexEqual should be case-insensitive")
	}
	if !HexEqual("AABB", "aabb") {
		t.Error("HexEqual should be case-insensitive (reverse)")
	}
	if HexEqual("abcdef", "abcdee") {
		t.Error("HexEqual should detect differences")
	}
	// Both directions: a guard that only rejected "first longer than
	// second" would let a 4-character operand "match" a 64-character one
	// when it comes first.
	if HexEqual("abcdef", "abcd") {
		t.Error("HexEqual should detect length differences (longer first)")
	}
	if HexEqual("abcd", "abcdef") {
		t.Error("HexEqual should detect length differences (shorter first)")
	}
	if HexEqual("", "abcd") || HexEqual("abcd", "") {
		t.Error("HexEqual should never match an empty operand against a non-empty one")
	}
	if !HexEqual("", "") {
		t.Error("HexEqual should match empty strings")
	}
	if !HexEqual("aabb", "aabb") {
		t.Error("HexEqual should match identical lowercase")
	}
}

// hexEqualLegacy is the short-circuiting implementation HexEqual replaced.
// Kept here as a differential oracle: making HexEqual constant-time was a
// hardening, not a semantic change, and any divergence from this function is
// a regression.
func hexEqualLegacy(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'F' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'F' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func TestHexEqualSemanticParity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		want bool
	}{
		{"abcdef", "ABCDEF", true},
		{"AABB", "aabb", true},
		{"deadbeef", "DEADBEEF", true},
		{"abcdef", "abcdee", false},
		// Length asymmetry in BOTH orders. Truncating one operand to the
		// other's length turns HexEqual into a prefix comparison, and a caller
		// may pass a supplied, possibly short, operand first.
		{"abcdef", "abcd", false},
		{"abcd", "abcdef", false},
		{"", "a", false},
		{"a", "", false},
		{"abcd", "abcdefabcdefabcdefabcdef", false},
		{"abcdefabcdefabcdefabcdef", "abcd", false},
		{"", "", true},
		{"aabb", "aabb", true},
		// Non-hex and odd-length operands compare as raw folded text. These
		// two rows are the reason HexEqual does NOT hex-decode its operands
		// before comparing: decoding would flip both to false, and both
		// operands can be remote-controlled.
		{"zz", "zz", true},
		{"abc", "abc", true},
		// Only 'A'-'F' fold. 'g'/'G' and the bytes bracketing the fold range
		// ('@' = 'A'-1, '`' = 'a'-1) must stay distinct.
		{"zz", "ZZ", false},
		{"g", "G", false},
		{"@", "`", false},
	}
	for _, tc := range cases {
		if got := HexEqual(tc.a, tc.b); got != tc.want {
			t.Errorf("HexEqual(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
		if got := hexEqualLegacy(tc.a, tc.b); got != tc.want {
			t.Errorf("legacy oracle disagrees with the table for (%q, %q)", tc.a, tc.b)
		}
	}
}

func TestHexEqualParityOverByteRange(t *testing.T) {
	t.Parallel()
	// Every single-byte pair, so the fold boundaries are exhaustively pinned
	// rather than sampled. Both operands are one byte long, so this sweep says
	// nothing about length handling, TestHexEqualParityOverLengths is what
	// covers that axis.
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			sa, sb := string([]byte{byte(a)}), string([]byte{byte(b)})
			if HexEqual(sa, sb) != hexEqualLegacy(sa, sb) {
				t.Fatalf("HexEqual(%q, %q) diverges from the legacy oracle", sa, sb)
			}
		}
	}
}

// TestHexEqualParityOverLengths sweeps the axis the single-byte parity test
// structurally cannot reach: operands of differing lengths, in both orders,
// including the prefix-of relation that a truncating comparison would accept.
// Every one of the live call sites feeds a bundle- or remote-controlled string
// into one side, so "a shorter operand matches a longer one" is a forgery, not
// a cosmetic bug.
func TestHexEqualParityOverLengths(t *testing.T) {
	t.Parallel()
	digest := "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"
	operands := []string{""}
	for n := 1; n <= len(digest); n++ {
		operands = append(operands, digest[:n], strings.ToUpper(digest[:n]))
	}
	operands = append(operands, digest+"0", "zz", "z")

	for _, a := range operands {
		for _, b := range operands {
			got, want := HexEqual(a, b), hexEqualLegacy(a, b)
			if got != want {
				t.Fatalf("HexEqual(%q, %q) = %v, legacy oracle gives %v", a, b, got, want)
			}
			// The oracle shares the length guard, so assert the property
			// directly too rather than trusting the two to be wrong together.
			if len(a) != len(b) && got {
				t.Fatalf("HexEqual(%q, %q) matched operands of length %d and %d", a, b, len(a), len(b))
			}
		}
	}
}

func TestHexEqualNoEarlyExit(t *testing.T) {
	t.Parallel()
	// A behavioural net for the constant-time rewrite: a mismatch at the
	// first byte and at the last byte are both detected. The timing property
	// itself is documented by BenchmarkHexEqualEarlyDiff /
	// BenchmarkHexEqualLateDiff, timing is not reliably assertable in CI and
	// must not become a flaky gate.
	base := strings.Repeat("a", 64)
	early := "b" + base[1:]
	late := base[:63] + "b"
	if HexEqual(base, early) {
		t.Error("mismatch at offset 0 should not compare equal")
	}
	if HexEqual(base, late) {
		t.Error("mismatch at offset 63 should not compare equal")
	}
	if !HexEqual(base, base) {
		t.Error("identical 64-char digests should compare equal")
	}
}

// hexOfLen builds a lowercase-hex string of exactly n characters.
func hexOfLen(n int) string {
	return strings.Repeat("0123456789abcdef", n/16+1)[:n]
}

func TestValidateClaimsHashRegistryMatchesSpec(t *testing.T) {
	t.Parallel()
	// The twelve registered algorithms of Appendix E.11, verbatim and in the
	// whitepaper's order. A thirteenth entry, a rename, or a dropped entry is
	// a wire-contract change, not a refactor.
	want := []string{
		"md5", "sha1", "sha224", "sha256", "sha384", "sha512",
		"sha3_224", "sha3_256", "sha3_384", "sha3_512", "blake2s", "blake2b",
	}
	got := make([]string, 0, len(hashTypes))
	for name := range hashTypes {
		got = append(got, name)
	}
	sort.Strings(got)
	sorted := append([]string(nil), want...)
	sort.Strings(sorted)
	if strings.Join(got, ",") != strings.Join(sorted, ",") {
		t.Errorf("registered hash types: got [%s], want [%s]", strings.Join(got, ","), strings.Join(sorted, ","))
	}
}

func TestValidateClaimsHashAcceptsLowercase(t *testing.T) {
	t.Parallel()
	// One valid lowercase digest per registered type at its exact width.
	widths := map[string]int{
		"md5": 32, "sha1": 40, "sha224": 56, "sha256": 64,
		"sha384": 96, "sha512": 128, "sha3_224": 56, "sha3_256": 64,
		"sha3_384": 96, "sha3_512": 128, "blake2s": 64, "blake2b": 128,
	}
	if len(widths) != len(hashTypes) {
		t.Fatalf("width table covers %d types, registry has %d", len(widths), len(hashTypes))
	}
	for hashType, width := range widths {
		if err := ValidateClaimsHash(hexOfLen(width), hashType); err != nil {
			t.Errorf("%s: unexpected error for a valid %d-char lowercase digest: %s", hashType, width, err)
		}
	}
}

func TestValidateClaimsHashCharset(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		hash string
	}{
		{"all z", strings.Repeat("z", 64)},
		{"uppercase hex", strings.ToUpper(hexOfLen(64))},
		{"punctuation", hexOfLen(63) + "!"},
		{"leading uppercase", "A" + hexOfLen(63)},
	}
	for _, tc := range cases {
		err := ValidateClaimsHash(tc.hash, "sha256")
		if err == nil {
			t.Errorf("%s: expected a charset error, got nil", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), "lowercase hex") {
			t.Errorf("%s: error should name the lowercase-hex requirement, got %q", tc.name, err)
		}
	}
}

func TestValidateClaimsHashLengthStillChecked(t *testing.T) {
	t.Parallel()
	// Short AND long. A guard written as "at least the expected width" passes
	// the short row and lets a padded digest through, and vice versa.
	cases := []struct {
		name string
		hash string
	}{
		{"too short", "deadbeef"},
		{"one char short", hexOfLen(63)},
		{"one char long", hexOfLen(65)},
		{"sha512-width value under sha256", hexOfLen(128)},
	}
	for _, tc := range cases {
		err := ValidateClaimsHash(tc.hash, "sha256")
		if err == nil {
			t.Errorf("%s: expected a length error for a %d-char sha256 hash", tc.name, len(tc.hash))
			continue
		}
		if !strings.Contains(err.Error(), "64 hex characters") {
			t.Errorf("%s: error should name the expected width, got %q", tc.name, err)
		}
	}
}

func TestValidateClaimsHashEmptyInputs(t *testing.T) {
	t.Parallel()
	for _, tc := range [][2]string{{"", ""}, {"abc", ""}, {"", "sha256"}} {
		if err := ValidateClaimsHash(tc[0], tc[1]); err != nil {
			t.Errorf("ValidateClaimsHash(%q, %q) = %s, want nil", tc[0], tc[1], err)
		}
	}
}

func TestValidateClaimsHashUnknownType(t *testing.T) {
	t.Parallel()
	err := ValidateClaimsHash(hexOfLen(64), "sha999")
	if err == nil {
		t.Fatal("expected an error for an unregistered hash type")
	}
	if !strings.Contains(err.Error(), "unknown hash type") {
		t.Errorf("error should name the unknown type, got %q", err)
	}
}
