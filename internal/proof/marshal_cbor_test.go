// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gowebpki/jcs"
)

// TestMarshalCBOR_RoundTripStabilizes confirms the canonicalization
// guarantee documented in the plan: a single round-trip through
// ParseCBOR → MarshalCBOR may rewrite a non-deterministic source, but
// a second round-trip must produce identical bytes.
func TestMarshalCBOR_RoundTripStabilizes(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "verify", "testdata", "proof_item.cbor")
	orig, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("no fixture at %s: %v", path, err)
	}

	bundle, err := ParseCBOR(orig)
	if err != nil {
		t.Fatalf("ParseCBOR(orig): %v", err)
	}
	once, err := bundle.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}

	bundle2, err := ParseCBOR(once)
	if err != nil {
		t.Fatalf("ParseCBOR(once): %v", err)
	}
	twice, err := bundle2.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR (second pass): %v", err)
	}

	if !bytes.Equal(once, twice) {
		t.Errorf("canonical form not stable: %d bytes vs %d bytes", len(once), len(twice))
	}
	// The output must still begin with the CBOR self-describing tag so
	// IsCBORProof keeps detecting it.
	if !IsCBORProof(once) {
		t.Error("MarshalCBOR output missing self-describing tag 0xd9d9f7")
	}
}

// TestMarshalCBOR_JSONToCBORToJSON checks that a JSON proof survives a
// trip through CBOR. Content must remain equivalent — specifically, the
// same bundle parsed from both forms must produce byte-identical JSON.
func TestMarshalCBOR_JSONToCBORToJSON(t *testing.T) {
	t.Parallel()
	jsonPath := filepath.Join("..", "verify", "testdata", "proof_item.json")
	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Skipf("no fixture: %v", err)
	}

	b1, err := ParseBytes(jsonBytes)
	if err != nil {
		t.Fatalf("ParseBytes(json): %v", err)
	}
	cborBytes, err := b1.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	b2, err := ParseCBOR(cborBytes)
	if err != nil {
		t.Fatalf("ParseCBOR: %v", err)
	}

	// Re-marshal both bundles to JSON. The JSON emitted via MarshalJSON
	// preserves each subject's raw JSON bytes, so the direct path keeps
	// the original key order inside `d`. The CBOR round-trip re-parses
	// `d` and emits canonical (sorted-key) order. Compare after JCS
	// canonicalization — the semantic equivalence check.
	j1, err := b1.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	j2, err := b2.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	c1, err := jcs.Transform(j1)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := jcs.Transform(j2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(c1, c2) {
		t.Errorf("json → cbor → json lost fidelity after JCS canonicalization:\n  before: %s\n  after:  %s", c1, c2)
	}
}
