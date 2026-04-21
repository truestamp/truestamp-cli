// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package encoding

import (
	"bytes"
	"testing"
)

// BenchmarkEncode covers all three text encodings against a typical
// 32-byte hash — the most common input size in this codebase.
func BenchmarkEncode(b *testing.B) {
	data := bytes.Repeat([]byte{0xaa}, 32)
	for _, enc := range []Encoding{Hex, Base64Std, Base64URL} {
		b.Run(enc.Name(), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = Encode(enc, data)
			}
		})
	}
}

// BenchmarkDecode covers the decoder paths including the
// cross-encoding-alphabet rejection branches.
func BenchmarkDecode(b *testing.B) {
	hex := []byte("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	b64 := []byte("qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq")
	b64url := []byte("qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq")

	b.Run("hex", func(b *testing.B) {
		b.SetBytes(int64(len(hex) / 2))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Decode(Hex, hex)
		}
	})
	b.Run("base64", func(b *testing.B) {
		b.SetBytes(int64(len(b64)))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Decode(Base64Std, b64)
		}
	})
	b.Run("base64url", func(b *testing.B) {
		b.SetBytes(int64(len(b64url)))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Decode(Base64URL, b64url)
		}
	})
}

// BenchmarkEncodeDecodeRoundTrip measures combined encode+decode cost.
// The realistic use case for the `encode | decode` pipeline.
func BenchmarkEncodeDecodeRoundTrip(b *testing.B) {
	data := bytes.Repeat([]byte{0x5a}, 256)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		encoded, _ := Encode(Base64Std, data)
		_, _ = Decode(Base64Std, encoded)
	}
}
