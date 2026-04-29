// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"strings"
	"testing"
)

// TestHashTypeDisplayMatchesServerCanonical pins each algorithm's
// Display name to the v2 server's @hash_types[*].name field. The
// dropdown picker, the watching-screen summary, and any future
// surface that renders a hash type all flow through this Display
// value, so drift between client and server canonical names breaks
// the user's mental model.
func TestHashTypeDisplayMatchesServerCanonical(t *testing.T) {
	t.Parallel()

	// Mirror of lib/truestamp/hash.ex's @hash_types[*].name strings.
	serverNames := map[string]string{
		"md5":      "MD5",
		"sha1":     "SHA-1",
		"sha224":   "SHA-224",
		"sha256":   "SHA-256",
		"sha384":   "SHA-384",
		"sha512":   "SHA-512",
		"sha3_224": "SHA3-224",
		"sha3_256": "SHA3-256",
		"sha3_384": "SHA3-384",
		"sha3_512": "SHA3-512",
		"blake2s":  "BLAKE2s",
		"blake2b":  "BLAKE2b",
	}

	for _, h := range hashTypeOptions {
		want, ok := serverNames[h.Value]
		if !ok {
			t.Errorf("hashTypeOptions has %q which is not in the server map", h.Value)
			continue
		}
		if h.Display != want {
			t.Errorf("%s Display = %q, want %q (server canonical name)",
				h.Value, h.Display, want)
		}
	}
}

// TestSelectLabelStartsWithDisplayName verifies the dropdown label
// is derived from the canonical Display name — the algorithm name
// the user sees first in the picker is the same string that appears
// on the watching screen.
func TestSelectLabelStartsWithDisplayName(t *testing.T) {
	t.Parallel()

	for _, h := range hashTypeOptions {
		if !strings.HasPrefix(h.selectLabel(), h.Display) {
			t.Errorf("%s selectLabel = %q does not start with Display %q",
				h.Value, h.selectLabel(), h.Display)
		}
	}
}

// TestDisplayHashTypeFallsBackToWireValue confirms the lookup is
// best-effort: an unknown wire value renders as itself rather than
// blanking the watching-screen field.
func TestDisplayHashTypeFallsBackToWireValue(t *testing.T) {
	t.Parallel()

	if got := displayHashType("sha256"); got != "SHA-256" {
		t.Errorf("displayHashType(\"sha256\") = %q, want SHA-256", got)
	}
	if got := displayHashType("ripemd160"); got != "ripemd160" {
		t.Errorf("displayHashType of unknown should pass through, got %q", got)
	}
}

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

// TestValidateHashEmptyHashTypeIsRejected confirms an empty hashType
// is no longer a soft-fail — even with otherwise-valid hex input,
// the validator demands a chosen algorithm so the table-driven
// length check can run.
func TestValidateHashEmptyHashTypeIsRejected(t *testing.T) {
	t.Parallel()

	emptyHT := ""
	validate := validateHash(&emptyHT)

	// Even-length valid hex still rejected — algorithm must be
	// chosen so the length is checked against an expected size.
	if err := validate(strings.Repeat("a", 64)); err == nil {
		t.Error("validateHash accepted input with empty hashType")
	}

	// Odd-length still also rejected (regex bites first).
	if err := validate(strings.Repeat("a", 63)); err == nil {
		t.Error("validateHash accepted odd-length input with empty hashType")
	}
}

// TestValidateHashUnknownHashTypeIsRejected confirms a hashType not
// in our canonical table is rejected with a clear "unknown hash type"
// error, not silently skipped.
func TestValidateHashUnknownHashTypeIsRejected(t *testing.T) {
	t.Parallel()

	unknownHT := "ripemd160"
	validate := validateHash(&unknownHT)

	if err := validate(strings.Repeat("a", 40)); err == nil {
		t.Errorf("validateHash accepted input with unknown hashType %q", unknownHT)
	}
}

// TestValidateHashRejectsOddLengthForAllInputs confirms the pair-
// regex catches odd-length hex regardless of hashType. The regex
// runs before the table lookup so an empty / unknown hashType
// surfaces as the regex error first when the input is malformed.
func TestValidateHashRejectsOddLengthForAllInputs(t *testing.T) {
	t.Parallel()

	for _, ht := range []string{"sha256", "sha512", "blake2b", "sha1", "md5"} {
		validate := validateHash(&ht)
		cases := []string{
			"a",                     // 1 char
			"abc",                   // 3 chars
			strings.Repeat("a", 63), // 63 chars (sha256 minus 1)
			strings.Repeat("a", 41), // 41 chars (sha1 plus 1)
			"deadbee",               // 7 chars
		}
		for _, in := range cases {
			if err := validate(in); err == nil {
				t.Errorf("hashType=%s: validateHash accepted odd-length %q (len=%d)",
					ht, in, len(in))
			}
		}
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
