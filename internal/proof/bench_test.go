// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"os"
	"path/filepath"
	"testing"
)

// The benchmarks load real fixture proofs so results reflect the
// wall-clock cost of the hot path during `truestamp verify`. Skips
// when fixtures are missing so `go test ./...` still works outside
// the repo (e.g. via `go install`).

func loadFixture(b *testing.B, path string) []byte {
	b.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		b.Skipf("missing fixture %s: %v", path, err)
	}
	return data
}

func BenchmarkParseJSON(b *testing.B) {
	data := loadFixture(b, filepath.Join("..", "verify", "testdata", "proof_item.json"))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := ParseBytes(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseCBOR(b *testing.B) {
	data := loadFixture(b, filepath.Join("..", "verify", "testdata", "proof_item.cbor"))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := ParseCBOR(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	bundle, err := ParseBytes(loadFixture(b,
		filepath.Join("..", "verify", "testdata", "proof_item.json")))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := bundle.MarshalJSON(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalCBOR(b *testing.B) {
	bundle, err := ParseBytes(loadFixture(b,
		filepath.Join("..", "verify", "testdata", "proof_item.json")))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := bundle.MarshalCBOR(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRoundTripCBOR reflects the `convert proof --to cbor` path
// plus the round-trip stability check exercised by the MarshalCBOR
// unit tests.
func BenchmarkRoundTripCBOR(b *testing.B) {
	data := loadFixture(b, filepath.Join("..", "verify", "testdata", "proof_item.json"))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bundle, err := ParseBytes(data)
		if err != nil {
			b.Fatal(err)
		}
		out, err := bundle.MarshalCBOR()
		if err != nil {
			b.Fatal(err)
		}
		if _, err := ParseCBOR(out); err != nil {
			b.Fatal(err)
		}
	}
}
