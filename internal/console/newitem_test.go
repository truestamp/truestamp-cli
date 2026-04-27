// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"strings"
	"testing"
)

// TestHashTypeOptionsCoverServerAcceptedTypes pins the canonical set
// the v2 backend accepts (lib/truestamp/hash.ex @hash_types). If the
// server adds an algorithm and we forget to mirror it, the new type
// is silently unselectable in the form. This test is the canary.
func TestHashTypeOptionsCoverServerAcceptedTypes(t *testing.T) {
	t.Parallel()

	// Mirror of the server's @hash_types map (just the wire keys).
	// Update both sides if the server's set changes.
	wantTypes := []string{
		"md5", "sha1", "sha224", "sha256", "sha384", "sha512",
		"sha3_224", "sha3_256", "sha3_384", "sha3_512",
		"blake2s", "blake2b",
	}

	got := make(map[string]bool, len(hashTypeOptions))
	for _, h := range hashTypeOptions {
		got[h.Value] = true
	}

	for _, want := range wantTypes {
		if !got[want] {
			t.Errorf("hashTypeOptions missing %q (server accepts it)", want)
		}
	}

	if len(hashTypeOptions) != len(wantTypes) {
		t.Errorf("hashTypeOptions has %d entries, server has %d — drift",
			len(hashTypeOptions), len(wantTypes))
	}
}

// TestHashTypeHexLengthsMatchByteSizes verifies the hex-length entry
// is exactly 2× the canonical byte size of each algorithm. Wrong
// lengths would let the user submit an unparseable hash that the
// server then rejects with a less helpful error.
func TestHashTypeHexLengthsMatchByteSizes(t *testing.T) {
	t.Parallel()

	wantBytes := map[string]int{
		"md5":      16,
		"sha1":     20,
		"sha224":   28,
		"sha256":   32,
		"sha384":   48,
		"sha512":   64,
		"sha3_224": 28,
		"sha3_256": 32,
		"sha3_384": 48,
		"sha3_512": 64,
		"blake2s":  32,
		"blake2b":  64,
	}

	for _, h := range hashTypeOptions {
		want, ok := wantBytes[h.Value]
		if !ok {
			t.Errorf("hashTypeOptions has unexpected %q", h.Value)
			continue
		}
		if h.HexLen != want*2 {
			t.Errorf("%s HexLen = %d, want %d (= 2 × %d bytes)",
				h.Value, h.HexLen, want*2, want)
		}
	}
}

// TestValidateHashAcrossAllTypes confirms validateHash accepts a
// correctly-sized hex digest for every algorithm and rejects
// off-by-one length errors.
func TestValidateHashAcrossAllTypes(t *testing.T) {
	t.Parallel()

	for _, h := range hashTypeOptions {
		t.Run(h.Value, func(t *testing.T) {
			ht := h.Value
			validate := validateHash(&ht)

			good := strings.Repeat("a", h.HexLen)
			if err := validate(good); err != nil {
				t.Errorf("validateHash rejected a %d-char hex digest for %s: %v",
					h.HexLen, h.Value, err)
			}

			tooShort := strings.Repeat("a", h.HexLen-1)
			if err := validate(tooShort); err == nil {
				t.Errorf("validateHash accepted a %d-char digest for %s (expected %d)",
					h.HexLen-1, h.Value, h.HexLen)
			}

			tooLong := strings.Repeat("a", h.HexLen+1)
			if err := validate(tooLong); err == nil {
				t.Errorf("validateHash accepted a %d-char digest for %s (expected %d)",
					h.HexLen+1, h.Value, h.HexLen)
			}
		})
	}
}

// TestValidateHashRejectsNonHex confirms the regex still bites for
// every hash type — algorithm choice doesn't bypass hex validation.
func TestValidateHashRejectsNonHex(t *testing.T) {
	t.Parallel()

	ht := "sha256"
	validate := validateHash(&ht)

	for _, bad := range []string{
		"not-hex-at-all-not-hex-at-all-not-hex-at-all-not-hex-at-all-1234",
		strings.Repeat("z", 64),
		strings.Repeat(" ", 64),
	} {
		if err := validate(bad); err == nil {
			t.Errorf("validateHash accepted non-hex input %q", bad)
		}
	}
}

// TestValidateHashRejectsOddLengthEvenWithoutHashType confirms the
// pair-regex catches odd-length input regardless of hashType. This
// is the defence-in-depth case: an unknown hashType skips the
// table-driven length check, so the regex must enforce
// even-byte-pairs by itself.
func TestValidateHashRejectsOddLengthEvenWithoutHashType(t *testing.T) {
	t.Parallel()

	// Empty hashType — table lookup is skipped, but even-length
	// must still be enforced.
	emptyHT := ""
	validate := validateHash(&emptyHT)

	cases := []string{
		"a",                               // 1 char
		"abc",                             // 3 chars
		strings.Repeat("a", 63),           // 63 = sha256 byte size minus 1
		strings.Repeat("a", 41),           // 41 = sha1 byte size plus 1
		"deadbee",                         // 7 chars
	}
	for _, in := range cases {
		if err := validate(in); err == nil {
			t.Errorf("validateHash accepted odd-length %q (len=%d) with empty hashType",
				in, len(in))
		}
	}

	// Unknown hashType (not in our canonical table) — same
	// defence-in-depth pathway must still reject odd-length.
	unknownHT := "ripemd160"
	validate = validateHash(&unknownHT)

	if err := validate(strings.Repeat("a", 39)); err == nil {
		t.Errorf("validateHash accepted odd-length input with unknown hashType %q", unknownHT)
	}
	// And: even-length input with unknown hashType is allowed
	// (we defer to the server's constraint regex for algorithm
	// validity).
	if err := validate(strings.Repeat("a", 40)); err != nil {
		t.Errorf("validateHash rejected even-length hex with unknown hashType: %v", err)
	}
}

// TestValidateHashEnforcesAllThreeChecks pins the three composed
// invariants (hex-only, even length, table-driven length) by walking
// the matrix of failure modes for a representative algorithm.
func TestValidateHashEnforcesAllThreeChecks(t *testing.T) {
	t.Parallel()

	ht := "sha256" // 64 hex chars
	validate := validateHash(&ht)

	cases := []struct {
		name string
		in   string
		ok   bool
	}{
		{"empty", "", false},
		{"odd length", strings.Repeat("a", 63), false},
		{"even length but wrong size", strings.Repeat("a", 32), false},
		{"non-hex same length", strings.Repeat("z", 64), false},
		{"correct", strings.Repeat("a", 64), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.in)
			if tc.ok && err != nil {
				t.Errorf("expected accept, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("expected reject, got nil")
			}
		})
	}
}