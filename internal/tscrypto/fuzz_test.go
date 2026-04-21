// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import "testing"

// FuzzDecodeCompactMerkleProof feeds arbitrary base64url strings to
// the compact-proof decoder. Must never panic — every malformed input
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

// FuzzHexEqual exercises the case-insensitive hex comparator — pure
// string code with explicit index arithmetic, worth fuzzing.
func FuzzHexEqual(f *testing.F) {
	f.Add("deadbeef", "DEADBEEF")
	f.Add("", "")
	f.Add("ab", "cd")

	f.Fuzz(func(t *testing.T, a, b string) {
		_ = HexEqual(a, b)
	})
}

// FuzzValidateClaimsHash: the hash_type+length validator.
func FuzzValidateClaimsHash(f *testing.F) {
	f.Add("", "")
	f.Add("deadbeef", "sha256")
	f.Add("xxx", "sha3_256")

	f.Fuzz(func(t *testing.T, hash, hashType string) {
		_ = ValidateClaimsHash(hash, hashType)
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
