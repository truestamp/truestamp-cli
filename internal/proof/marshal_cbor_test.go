// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
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
	if !HasCBORTag(once) {
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
	// internal/jcs, not github.com/gowebpki/jcs: the wrapper deliberately
	// diverges from the library for integer literals beyond ±2^53 (it
	// splices them out so they are never rounded), and it is the wrapper
	// that produces every signed digest. Calling the library directly here
	// would leave this equivalence green while production canonicalization
	// moved underneath it.
	c1, _, err := jcs.Canonicalize(j1)
	if err != nil {
		t.Fatal(err)
	}
	c2, _, err := jcs.Canonicalize(j2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(c1, c2) {
		t.Errorf("json → cbor → json lost fidelity after JCS canonicalization:\n  before: %s\n  after:  %s", c1, c2)
	}
}

// TestMarshalCBOR_RefusesUppercaseHex pins the coherent half of the E.4
// lowercase-hex enforcement.
//
// hex.DecodeString is case-insensitive, so before this check `convert proof
// --to cbor` turned a bundle whose `b.kid` read "F2C39DF9" into a CBOR
// bundle carrying the identical four bytes as a byte string — a file that
// verifies, produced from a file that does not. That is a repair the CLI
// has no business performing silently: it hands the operator back a
// laundered artifact for a defect the verifier grades a failure. Refusing
// the conversion keeps the two commands agreeing about the same input.
func TestMarshalCBOR_RefusesUppercaseHex(t *testing.T) {
	t.Parallel()
	jsonPath := filepath.Join("..", "verify", "testdata", "fixtures", "appendix-d-item.json")
	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Skipf("no fixture: %v", err)
	}

	for _, tc := range []struct {
		name string
		// mutate uppercases one hex field on a freshly parsed bundle.
		mutate func(*ProofBundle)
		// want is the substring the error must carry so a caller learns
		// which field to fix rather than only that the encode failed.
		want string
	}{
		{"s.mh", func(b *ProofBundle) { b.Subject.MetadataHash = strings.ToUpper(b.Subject.MetadataHash) }, "mh"},
		{"s.kid", func(b *ProofBundle) { b.Subject.SigningKeyID = strings.ToUpper(b.Subject.SigningKeyID) }, "kid"},
		{"b.ph", func(b *ProofBundle) { b.Block.PreviousBlockHash = strings.ToUpper(b.Block.PreviousBlockHash) }, "ph"},
		{"b.mr", func(b *ProofBundle) { b.Block.MerkleRoot = strings.ToUpper(b.Block.MerkleRoot) }, "mr"},
		{"b.mh", func(b *ProofBundle) { b.Block.MetadataHash = strings.ToUpper(b.Block.MetadataHash) }, "mh"},
		{"b.kid", func(b *ProofBundle) { b.Block.SigningKeyID = strings.ToUpper(b.Block.SigningKeyID) }, "kid"},
		{"cx[0].tx", func(b *ProofBundle) {
			b.Commitments[0].TransactionHash = strings.ToUpper(b.Commitments[0].TransactionHash)
		}, "tx"},
		{"cx[0].memo", func(b *ProofBundle) { b.Commitments[0].MemoHash = strings.ToUpper(b.Commitments[0].MemoHash) }, "memo"},
		{"cx[1].op", func(b *ProofBundle) { b.Commitments[1].OpReturn = strings.ToUpper(b.Commitments[1].OpReturn) }, "op"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b, err := ParseBytes(jsonBytes)
			if err != nil {
				t.Fatalf("ParseBytes: %v", err)
			}
			tc.mutate(b)
			out, err := b.MarshalCBOR()
			if err == nil {
				t.Fatalf("MarshalCBOR accepted uppercase %s and emitted %d bytes; the defect was laundered away", tc.name, len(out))
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name %q", err, tc.want)
			}
			if !strings.Contains(err.Error(), "not lowercase hex") {
				t.Errorf("error %q does not say the value is not lowercase hex", err)
			}
		})
	}
}

// TestMarshalCBOR_AcceptsTheConformingVector is the control: the same
// fixture, unmutated, must still encode. A refusal rule that also rejects
// the Appendix D bundle would break every CBOR download, so this pins that
// the check discriminates rather than simply failing closed.
func TestMarshalCBOR_AcceptsTheConformingVector(t *testing.T) {
	t.Parallel()
	jsonPath := filepath.Join("..", "verify", "testdata", "fixtures", "appendix-d-item.json")
	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Skipf("no fixture: %v", err)
	}
	b, err := ParseBytes(jsonBytes)
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	if _, err := b.MarshalCBOR(); err != nil {
		t.Fatalf("the conforming Appendix D vector must still encode: %v", err)
	}
}
