// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package tscrypto

import "testing"

// FuzzDecodeCompactMerkleProof feeds arbitrary base64url strings to
// the compact-proof decoder. Must never panic, every malformed input
// returns a typed error.
func FuzzDecodeCompactMerkleProof(f *testing.F) {
	for _, s := range []string{
		"",
		"AA",
		"BAKbGnC2S9wB-uoc-ipZtm3XQi4yTfzoJ104AWSYH_qt6Dr1KQpjCQnFZZQ3Cl1T8frYIxF5l4vtsIQQs0hHXAF2PSRdVsGXyGOzNQoz-R9QS2Gq7X30GQ8jAK3EJz1qgWHLUIBht8G0Sdl2Z7NFP-7KPCtcMMUawPiQCmvjVLQn_g",
		"!!!",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = DecodeCompactMerkleProof(s)
	})
}

// FuzzExtractUUIDv7Timestamp: UUIDv7 parse surface.
func FuzzExtractUUIDv7Timestamp(f *testing.F) {
	f.Add("019cf813-99b8-730a-84f1-5a711a9c355e")
	f.Add("f47ac10b-58cc-4372-a567-0e02b2c3d479") // v4
	f.Add("")
	f.Add("garbage")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ExtractUUIDv7Timestamp(s)
	})
}

// FuzzHexEqual exercises the case-insensitive hex comparator, pure
// string code with explicit index arithmetic, worth fuzzing.
func FuzzHexEqual(f *testing.F) {
	f.Add("deadbeef", "DEADBEEF")
	f.Add("", "")
	f.Add("ab", "cd")
	// Non-hex and odd-length operands, and the bytes bracketing the 'A'-'F'
	// fold range, the cases the constant-time rewrite had to preserve.
	f.Add("zz", "zz")
	f.Add("abc", "abc")
	f.Add("@", "`")
	// A prefix pair in both orders: the shape a truncating comparison accepts.
	f.Add("b47c", "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380")
	f.Add("b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380", "b47c")

	f.Fuzz(func(t *testing.T, a, b string) {
		got := HexEqual(a, b)

		// Operands of different lengths can never match. Both live call sites
		// take one operand from the bundle or a remote API and one from the
		// caller, so a prefix acceptance here is a forged match.
		if len(a) != len(b) && got {
			t.Fatalf("HexEqual(%q, %q) matched operands of length %d and %d", a, b, len(a), len(b))
		}
		// Argument order is not part of the contract; the call sites use both.
		if flipped := HexEqual(b, a); flipped != got {
			t.Fatalf("HexEqual is not symmetric: (%q, %q) = %v, (%q, %q) = %v", a, b, got, b, a, flipped)
		}
		// Parity with the short-circuiting implementation the constant-time
		// rewrite replaced: hardening, never a semantic change.
		if want := hexEqualLegacy(a, b); got != want {
			t.Fatalf("HexEqual(%q, %q) = %v, legacy oracle gives %v", a, b, got, want)
		}
	})
}
