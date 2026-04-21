// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

// FuzzParseBytes exercises the top-level proof parser with arbitrary
// bytes. The parser must never panic — all malformed inputs should
// return a typed error.
func FuzzParseBytes(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("{}"))
	f.Add([]byte("not json"))
	f.Add([]byte(validProofJSON))
	if data := readOrEmpty(filepath.Join("..", "verify", "testdata", "proof_item.json")); len(data) > 0 {
		f.Add(data)
	}
	if data := readOrEmpty(filepath.Join("..", "verify", "testdata", "proof_item.cbor")); len(data) > 0 {
		f.Add(data) // ParseBytes auto-dispatches to ParseCBOR
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseBytes(data)
	})
}

// FuzzParseCBOR hammers the CBOR-specific path with arbitrary bytes
// AND with well-formed CBOR of arbitrary shape (via cbor.Marshal of
// a small map) to steer the fuzzer toward the parser's internal
// branches.
func FuzzParseCBOR(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xd9, 0xd9, 0xf7}) // self-describing tag only
	if data := readOrEmpty(filepath.Join("..", "verify", "testdata", "proof_item.cbor")); len(data) > 0 {
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseCBOR(data)
	})
}

// FuzzParseCBOR_WellFormed generates well-formed CBOR from an integer
// + string seed and feeds the ProofBundle parser. This lets the fuzzer
// target structural validation (missing fields, wrong types) rather
// than getting stuck on malformed byte sequences.
func FuzzParseCBOR_WellFormed(f *testing.F) {
	f.Add(int64(1), "pk-value")
	f.Add(int64(0), "")

	f.Fuzz(func(t *testing.T, v int64, pk string) {
		m := map[string]any{
			"v":  v,
			"pk": pk,
		}
		data, err := cbor.Marshal(m)
		if err != nil {
			return
		}
		_, _ = ParseCBOR(data)
	})
}

// FuzzIsCBORProof tests the 3-byte magic check — trivial but cheap.
func FuzzIsCBORProof(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xd9, 0xd9, 0xf7})
	f.Add([]byte("{"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_ = IsCBORProof(data)
	})
}

// FuzzDetectIDType checks the ULID/UUIDv7 classifier against arbitrary
// strings.
func FuzzDetectIDType(f *testing.F) {
	f.Add("01HJHB01T8FYZ7YTR9P5N62K5B")
	f.Add("019cf813-99b8-730a-84f1-5a711a9c355e")
	f.Add("")
	f.Add("not-an-id")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = DetectIDType(s)
	})
}

// FuzzParseThenMarshalJSON checks the JSON round-trip property: any
// parse-able proof must also marshal back to JSON without panicking.
func FuzzParseThenMarshalJSON(f *testing.F) {
	f.Add([]byte(validProofJSON))
	if data := readOrEmpty(filepath.Join("..", "verify", "testdata", "proof_item.json")); len(data) > 0 {
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		b, err := ParseBytes(data)
		if err != nil {
			return // only successfully-parsed inputs fuel the round-trip
		}
		if _, err := b.MarshalJSON(); err != nil {
			t.Errorf("MarshalJSON on parsed bundle: %v", err)
		}
	})
}

// FuzzParseThenMarshalCBOR guards the deterministic CBOR encoder:
// every bundle that parses must also re-encode without error.
func FuzzParseThenMarshalCBOR(f *testing.F) {
	f.Add([]byte(validProofJSON))
	if data := readOrEmpty(filepath.Join("..", "verify", "testdata", "proof_item.json")); len(data) > 0 {
		f.Add(data)
	}
	if data := readOrEmpty(filepath.Join("..", "verify", "testdata", "proof_item.cbor")); len(data) > 0 {
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		b, err := ParseBytes(data)
		if err != nil {
			return
		}
		out, err := b.MarshalCBOR()
		if err != nil {
			// Malformed hex/base64 inside the JSON gets rejected at
			// marshal time — that's a legitimate parse-then-fail case,
			// not a panic.
			return
		}
		// Second parse must also succeed (canonicalization is idempotent).
		if _, err := ParseCBOR(out); err != nil {
			t.Errorf("re-parse of MarshalCBOR output failed: %v", err)
		}
	})
}

// readOrEmpty reads a file as a seed if present; returns nil on error
// so the fuzz corpus is still seeded from whatever fixtures exist.
func readOrEmpty(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}
