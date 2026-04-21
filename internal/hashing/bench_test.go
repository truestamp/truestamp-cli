// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package hashing

import (
	"bytes"
	"context"
	"testing"
)

// BenchmarkCompute_ByAlgorithm measures hash throughput for every
// registered algorithm against a 1 MiB payload. Run with:
//
//	go test -bench=BenchmarkCompute_ByAlgorithm -benchmem ./internal/hashing/
//
// Use b.SetBytes so `go test -bench` reports MB/s in addition to ns/op.
func BenchmarkCompute_ByAlgorithm(b *testing.B) {
	data := bytes.Repeat([]byte{0x5a}, 1<<20) // 1 MiB
	ctx := context.Background()

	for _, alg := range Algorithms() {
		b.Run(alg.Name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, _, err := Compute(ctx, alg, bytes.NewReader(data)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCompute_SmallInput captures per-call overhead at small
// payload sizes — the regime where hasher setup cost dominates. Useful
// for spotting regressions in the algorithm registry lookup or
// io.Copy wiring.
func BenchmarkCompute_SmallInput(b *testing.B) {
	data := []byte("abc")
	ctx := context.Background()
	alg, _ := Lookup("sha256")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = Compute(ctx, alg, bytes.NewReader(data))
	}
}

// BenchmarkLookup measures the algorithm-name resolver. Called once
// per `truestamp hash` invocation.
func BenchmarkLookup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Lookup("sha256")
	}
}

// BenchmarkFormatGNU profiles the sha256sum-compatible output
// formatter — called per input line.
func BenchmarkFormatGNU(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = FormatGNU("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
			"testdata/fixture.bin", false)
	}
}
