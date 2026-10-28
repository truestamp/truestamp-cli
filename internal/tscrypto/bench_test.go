// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package tscrypto

import (
	"bytes"
	"strings"
	"testing"
)

// BenchmarkDomainHash measures SHA-256 of a one-byte prefix + payload.
func BenchmarkDomainHash(b *testing.B) {
	data := bytes.Repeat([]byte{0x5a}, 128)
	b.SetBytes(int64(len(data) + 1))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = DomainHash(0x5a, data)
	}
}

// BenchmarkDecodeCompactMerkleProof measures decoding of the binary
// compact Merkle proof format.
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

// BenchmarkHexEqualEarlyDiff and BenchmarkHexEqualLateDiff document the
// constant-time property of HexEqual on realistic 64-char digest operands:
// the two should report the same ns/op regardless of WHERE the operands
// diverge. They are documentation, not a gate, timing is not reliably
// assertable in CI, so nothing asserts on the ratio.
func BenchmarkHexEqualEarlyDiff(b *testing.B) {
	base := strings.Repeat("a", 64)
	other := "b" + base[1:]
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = HexEqual(base, other)
	}
}

func BenchmarkHexEqualLateDiff(b *testing.B) {
	base := strings.Repeat("a", 64)
	other := base[:63] + "b"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = HexEqual(base, other)
	}
}

func BenchmarkHexEqualMatch(b *testing.B) {
	base := strings.Repeat("a", 64)
	other := strings.Repeat("a", 64)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = HexEqual(base, other)
	}
}
