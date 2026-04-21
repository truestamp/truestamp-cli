// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"bytes"
	"testing"
)

// BenchmarkDomainHash measures the single call pattern the Truestamp
// wire format uses everywhere: SHA-256 of a one-byte prefix + payload.
// Called on every proof for claims_hash, item_hash, block_hash, etc.
func BenchmarkDomainHash(b *testing.B) {
	data := bytes.Repeat([]byte{0x5a}, 128)
	b.SetBytes(int64(len(data) + 1))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = DomainHash(PrefixItemClaims, data)
	}
}

// BenchmarkComputeItemHash profiles the length-prefixed item hash
// builder — the inner loop of proof verification. Uses realistic
// field sizes drawn from actual proofs.
func BenchmarkComputeItemHash(b *testing.B) {
	const (
		id       = "01HJHB01T8FYZ7YTR9P5N62K5B"
		claims   = "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd"
		metadata = "14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1"
		kid      = "4ceefa4a"
	)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := ComputeItemHash(id, claims, metadata, kid); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeCompactMerkleProof measures decoding of the binary
// compact proof format read from every proof bundle's `ip` field.
func BenchmarkDecodeCompactMerkleProof(b *testing.B) {
	// A real depth-4 compact proof captured from the sample bundle.
	const proof = "BAKbGnC2S9wB-uoc-ipZtm3XQi4yTfzoJ104AWSYH_qt6Dr1KQpjCQnFZZQ3Cl1T8frYIxF5l4vtsIQQs0hHXAF2PSRdVsGXyGOzNQoz-R9QS2Gq7X30GQ8jAK3EJz1qgWHLUIBht8G0Sdl2Z7NFP-7KPCtcMMUawPiQCmvjVLQn_g"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := DecodeCompactMerkleProof(proof); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVerifyMerkleProof measures the RFC 6962 walk — run once
// per proof against block.merkle_root.
func BenchmarkVerifyMerkleProof(b *testing.B) {
	leaf := "14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1"
	proof := []string{
		"l:9b1a70b64bdc01faea1cfa2a59b66dd7422e324dfce8275d380164981ffaade8",
		"r:3af5290a630909c56594370a5d53f1fad8231179978bedb08410b348475c0176",
		"l:3d245d56c197c863b3350a33f91f504b61aaed7df4190f2300adc4273d6a8161",
	}
	root := "abcdef01abcdef01abcdef01abcdef01abcdef01abcdef01abcdef01abcdef01"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = VerifyMerkleProof(leaf, proof, root)
	}
}

// BenchmarkComputeKeyID profiles the Ed25519 -> 4-byte kid derivation.
func BenchmarkComputeKeyID(b *testing.B) {
	key := bytes.Repeat([]byte{0xaa}, 32)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ComputeKeyID(key)
	}
}

// BenchmarkLenPrefix exercises the 4-byte length-prefix helper used
// when serializing hash inputs.
func BenchmarkLenPrefix(b *testing.B) {
	data := bytes.Repeat([]byte{0xaa}, 64)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = LenPrefix(data)
	}
}
