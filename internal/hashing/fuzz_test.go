// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package hashing

import (
	"bytes"
	"context"
	"testing"
)

// FuzzLookup: algorithm-name resolver with arbitrary strings.
func FuzzLookup(f *testing.F) {
	for _, s := range []string{"", "sha256", "SHA256", "sha3_256", "blake2b", "unknown"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = Lookup(s)
	})
}

// FuzzCompute feeds arbitrary bytes through every registered hasher.
// The hashers themselves are stdlib (and internally fuzz-hardened),
// but this also exercises our Compute wrapper, io.Copy plumbing, and
// size accounting.
func FuzzCompute(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("abc"))
	f.Add(bytes.Repeat([]byte{0x5a}, 1024))

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, alg := range Algorithms() {
			digest, n, err := Compute(context.Background(), alg, bytes.NewReader(data))
			if err != nil {
				t.Errorf("Compute(%s): %v", alg.Name, err)
				continue
			}
			if n != int64(len(data)) {
				t.Errorf("Compute(%s) size: got %d, want %d", alg.Name, n, len(data))
			}
			if len(digest) != alg.Size {
				t.Errorf("Compute(%s) digest size: got %d, want %d",
					alg.Name, len(digest), alg.Size)
			}
		}
	})
}

// FuzzGnuEscape: filename-escaping helper for sha256sum-compatible
// output. Fuzz with arbitrary strings to catch any mishandled byte
// sequence.
func FuzzGnuEscape(f *testing.F) {
	for _, s := range []string{"", "normal", `a\b`, "a\nb", "\\\\\n\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = gnuEscape(s)
	})
}

// FuzzFormatGNU: full output-line formatter. Arbitrary digest + name.
func FuzzFormatGNU(f *testing.F) {
	f.Add("deadbeef", "foo.bin", false)
	f.Add("", "", true)
	f.Add("abc", "a\nb", false)

	f.Fuzz(func(t *testing.T, digest, name string, binary bool) {
		_ = FormatGNU(digest, name, binary)
	})
}
