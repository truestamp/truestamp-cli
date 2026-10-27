// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package tscrypto

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

// --- DecodeCompactMerkleProof tests ---

func TestDecodeCompactMerkleProof_Empty(t *testing.T) {
	t.Parallel()
	// "AA" is base64url for a single byte 0x00 (depth=0)
	proof, err := DecodeCompactMerkleProof("AA")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(proof) != 0 {
		t.Errorf("expected empty proof slice, got %d elements", len(proof))
	}
}

func TestDecodeCompactMerkleProof_SingleStep(t *testing.T) {
	t.Parallel()
	// Build a 1-step compact proof manually:
	//   depth=1, bitfield=0x01 (right), 32 bytes of hash
	hashHex := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hashBytes, _ := hex.DecodeString(hashHex)

	var buf []byte
	buf = append(buf, 0x01)         // depth = 1
	buf = append(buf, 0x01)         // bitfield: bit 0 = 1 -> right
	buf = append(buf, hashBytes...) // 32 bytes of sibling hash

	encoded := base64.RawURLEncoding.EncodeToString(buf)

	proof, err := DecodeCompactMerkleProof(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(proof) != 1 {
		t.Fatalf("expected 1 proof step, got %d", len(proof))
	}
	expected := "r:" + hashHex
	if proof[0] != expected {
		t.Errorf("expected %s, got %s", expected, proof[0])
	}
}

func TestDecodeCompactMerkleProof_MultipleSteps(t *testing.T) {
	t.Parallel()
	// Build a 2-step compact proof:
	//   depth=2, bitfield=0x02 (bit0=0 -> left, bit1=1 -> right)
	hash1Hex := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hash2Hex := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	hash1Bytes, _ := hex.DecodeString(hash1Hex)
	hash2Bytes, _ := hex.DecodeString(hash2Hex)

	var buf []byte
	buf = append(buf, 0x02)          // depth = 2
	buf = append(buf, 0x02)          // bitfield: bit0=0 (left), bit1=1 (right) -> 0b00000010
	buf = append(buf, hash1Bytes...) // sibling hash for step 0
	buf = append(buf, hash2Bytes...) // sibling hash for step 1

	encoded := base64.RawURLEncoding.EncodeToString(buf)

	proof, err := DecodeCompactMerkleProof(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(proof) != 2 {
		t.Fatalf("expected 2 proof steps, got %d", len(proof))
	}
	if proof[0] != "l:"+hash1Hex {
		t.Errorf("step 0: expected l:%s, got %s", hash1Hex, proof[0])
	}
	if proof[1] != "r:"+hash2Hex {
		t.Errorf("step 1: expected r:%s, got %s", hash2Hex, proof[1])
	}
}

func TestDecodeCompactMerkleProof_InvalidBase64(t *testing.T) {
	t.Parallel()
	_, err := DecodeCompactMerkleProof("!!!not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64url input")
	}
}

func TestDecodeCompactMerkleProof_DepthTooLarge(t *testing.T) {
	t.Parallel()
	// Depth 65 exceeds the maximum of 64
	buf := []byte{65}
	encoded := base64.RawURLEncoding.EncodeToString(buf)

	_, err := DecodeCompactMerkleProof(encoded)
	if err == nil {
		t.Error("expected error for depth > 64")
	}
	if err != nil && !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("expected 'exceeds maximum' in error, got: %s", err)
	}
}

func TestDecodeCompactMerkleProof_WrongLength(t *testing.T) {
	t.Parallel()
	// depth=1 expects 1 + 1 + 32 = 34 bytes total, provide only 10
	var buf []byte
	buf = append(buf, 0x01)               // depth = 1
	buf = append(buf, 0x00)               // bitfield
	buf = append(buf, []byte("short")...) // only 5 bytes instead of 32

	encoded := base64.RawURLEncoding.EncodeToString(buf)

	_, err := DecodeCompactMerkleProof(encoded)
	if err == nil {
		t.Error("expected error for wrong data length")
	}
	if err != nil && !strings.Contains(err.Error(), "expected") {
		t.Errorf("expected length mismatch error, got: %s", err)
	}
}
