// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"strings"
	"testing"
)

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

// FuzzVerifyMerkleProof exercises the RFC 6962 walk with arbitrary
// leaf/root hex + a single proof step. The walker takes attacker-
// controlled strings so any panic is a security issue.
func FuzzVerifyMerkleProof(f *testing.F) {
	f.Add("deadbeef", "l:cafebabe", "abcdef01")
	f.Add("", "", "")

	f.Fuzz(func(t *testing.T, leafHex, step, rootHex string) {
		_, _ = VerifyMerkleProof(leafHex, []string{step}, rootHex)
	})
}

// FuzzExtractULIDTimestamp: ULID parse surface from untrusted proof
// bundles.
func FuzzExtractULIDTimestamp(f *testing.F) {
	f.Add("01HJHB01T8FYZ7YTR9P5N62K5B")
	f.Add("")
	f.Add("not-a-ulid")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ExtractULIDTimestamp(s)
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

// FuzzHexToBytes wraps the hex decoder used throughout the package.
func FuzzHexToBytes(f *testing.F) {
	f.Add("")
	f.Add("deadbeef")
	f.Add("DEADBEEF")
	f.Add("zz")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = HexToBytes(s)
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

// FuzzValidateClaimsHash: the hash_type+length+charset validator.
func FuzzValidateClaimsHash(f *testing.F) {
	f.Add("", "")
	f.Add("deadbeef", "sha256")
	f.Add("xxx", "sha3_256")
	// Correct width, wrong character set, the charset arm.
	f.Add(strings.Repeat("z", 64), "sha256")
	f.Add(strings.ToUpper(strings.Repeat("ab", 32)), "sha256")

	f.Fuzz(func(t *testing.T, hash, hashType string) {
		if err := ValidateClaimsHash(hash, hashType); err != nil {
			return
		}
		// A nil error is an assertion, so pin what it asserts: Appendix E.11's
		// "hex length equals twice the algorithm's output size and the
		// character set is lowercase hex", for a registered algorithm. The
		// either-empty short-circuit is the one documented exemption.
		if hash == "" || hashType == "" {
			return
		}
		info, ok := hashTypes[hashType]
		if !ok {
			t.Fatalf("ValidateClaimsHash accepted unregistered hash type %q", hashType)
		}
		if len(hash) != info.Bytes*2 {
			t.Fatalf("ValidateClaimsHash accepted a %d-char hash for %s, which is %d bytes wide", len(hash), hashType, info.Bytes)
		}
		for i := 0; i < len(hash); i++ {
			c := hash[i]
			if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
				continue
			}
			t.Fatalf("ValidateClaimsHash accepted %q at offset %d of %q", string(c), i, hash)
		}
	})
}

// FuzzComputeItemHash: domain-prefixed length-prefixed hash builder.
// Exercises the len-prefix arithmetic with arbitrary string lengths.
func FuzzComputeItemHash(f *testing.F) {
	f.Add("01HJHB01T8FYZ7YTR9P5N62K5B",
		"deadbeef",
		"deadbeef",
		"4ceefa4a")

	f.Fuzz(func(t *testing.T, id, claimsHex, metaHex, kidHex string) {
		_, _ = ComputeItemHash(id, claimsHex, metaHex, kidHex)
	})
}

// FuzzComputeObservationHash: same shape as ComputeItemHash but for
// entropy proofs.
func FuzzComputeObservationHash(f *testing.F) {
	f.Add("019cf813-99b8-730a-84f1-5a711a9c355e",
		"deadbeef",
		"deadbeef",
		"4ceefa4a")

	f.Fuzz(func(t *testing.T, id, entHex, metaHex, kidHex string) {
		_, _ = ComputeObservationHash(id, entHex, metaHex, kidHex)
	})
}

// FuzzComputeBlockHash: same shape, but with an extra "previous block"
// hash field.
func FuzzComputeBlockHash(f *testing.F) {
	f.Add("019cf813-99b8-730a-84f1-5a711a9c355e",
		"deadbeef", "deadbeef", "deadbeef", "4ceefa4a")

	f.Fuzz(func(t *testing.T, id, phHex, mrHex, mhHex, kidHex string) {
		_, _ = ComputeBlockHash(id, phHex, mrHex, mhHex, kidHex)
	})
}

// FuzzBuildCompactProofPayload: variadic-ish hex decoder that feeds
// the signature construction. Fuzz to ensure no slice-bounds panic.
func FuzzBuildCompactProofPayload(f *testing.F) {
	f.Add(uint16(20), "4ceefa4a", uint64(1700000000000),
		"deadbeef", "deadbeef", "cafebabe")

	f.Fuzz(func(t *testing.T, typeCode uint16, kid string, ts uint64, subj, block, epoch string) {
		_, _ = BuildCompactProofPayload(1, typeCode, kid, ts, subj, block, []string{epoch})
	})
}
