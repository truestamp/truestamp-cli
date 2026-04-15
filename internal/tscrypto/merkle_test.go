// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gowebpki/jcs"
)

func TestVerifyMerkleProof_SingleLeaf(t *testing.T) {
	// Single-leaf tree: empty proof, root = SHA256(0x00 || leaf_bytes)
	leafHex := "6de54da2c7d3bddcd8a6c2bb22fc0c44af3bbad47d351cfd43ac8dbb5fc1cc3d"
	leafBytes, _ := HexToBytes(leafHex)

	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(leafBytes)
	expectedRoot := BytesToHex(h.Sum(nil))

	ok, err := VerifyMerkleProof(leafHex, []string{}, expectedRoot)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Error("single-leaf merkle proof should verify")
	}
}

func TestVerifyMerkleProof_WrongRoot(t *testing.T) {
	leafHex := "6de54da2c7d3bddcd8a6c2bb22fc0c44af3bbad47d351cfd43ac8dbb5fc1cc3d"
	wrongRoot := "0000000000000000000000000000000000000000000000000000000000000000"

	ok, err := VerifyMerkleProof(leafHex, []string{}, wrongRoot)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if ok {
		t.Error("merkle proof with wrong root should fail")
	}
}

func TestVerifyMerkleProof_InvalidDirection(t *testing.T) {
	leafHex := "6de54da2c7d3bddcd8a6c2bb22fc0c44af3bbad47d351cfd43ac8dbb5fc1cc3d"
	proof := []string{"x:0000000000000000000000000000000000000000000000000000000000000000"}

	_, err := VerifyMerkleProof(leafHex, proof, "anything")
	if err == nil {
		t.Error("expected error for invalid direction")
	}
}

func TestVerifyMerkleProof_InvalidHex(t *testing.T) {
	_, err := VerifyMerkleProof("not_hex", []string{}, "anything")
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

func TestVerifyMerkleProof_InvalidElementFormat(t *testing.T) {
	leafHex := "6de54da2c7d3bddcd8a6c2bb22fc0c44af3bbad47d351cfd43ac8dbb5fc1cc3d"
	proof := []string{"no-colon-separator"}

	_, err := VerifyMerkleProof(leafHex, proof, "anything")
	if err == nil {
		t.Error("expected error for invalid proof element (no colon)")
	}
}

func TestVerifyMerkleProof_InvalidSiblingHex(t *testing.T) {
	leafHex := "6de54da2c7d3bddcd8a6c2bb22fc0c44af3bbad47d351cfd43ac8dbb5fc1cc3d"
	proof := []string{"l:not_valid_hex_ZZZZ"}

	_, err := VerifyMerkleProof(leafHex, proof, "anything")
	if err == nil {
		t.Error("expected error for invalid sibling hex")
	}
}

func TestVerifyMerkleProof_WithProofSteps(t *testing.T) {
	// Build a simple 2-leaf tree manually and verify the proof
	leaf1, _ := HexToBytes("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	leaf2, _ := HexToBytes("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	// leaf hashes: SHA256(0x00 || leaf)
	h1 := sha256.New()
	h1.Write([]byte{0x00})
	h1.Write(leaf1)
	leafHash1 := h1.Sum(nil)

	h2 := sha256.New()
	h2.Write([]byte{0x00})
	h2.Write(leaf2)
	leafHash2 := h2.Sum(nil)

	// root: SHA256(0x01 || leafHash1 || leafHash2)
	hr := sha256.New()
	hr.Write([]byte{0x01})
	hr.Write(leafHash1)
	hr.Write(leafHash2)
	root := BytesToHex(hr.Sum(nil))

	// Proof for leaf1: sibling is leafHash2 on the right
	proof := []string{"r:" + BytesToHex(leafHash2)}

	ok, err := VerifyMerkleProof("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", proof, root)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Error("valid 2-leaf merkle proof should verify")
	}

	// Proof for leaf2: sibling is leafHash1 on the left
	proof2 := []string{"l:" + BytesToHex(leafHash1)}

	ok2, err := VerifyMerkleProof("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", proof2, root)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok2 {
		t.Error("valid 2-leaf merkle proof (left sibling) should verify")
	}
}

// --- DecodeCompactMerkleProof tests ---

func TestDecodeCompactMerkleProof_Empty(t *testing.T) {
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
	// Build a 1-step compact proof manually:
	//   depth=1, bitfield=0x01 (right), 32 bytes of hash
	hashHex := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hashBytes, _ := HexToBytes(hashHex)

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
	// Build a 2-step compact proof:
	//   depth=2, bitfield=0x02 (bit0=0 -> left, bit1=1 -> right)
	hash1Hex := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hash2Hex := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	hash1Bytes, _ := HexToBytes(hash1Hex)
	hash2Bytes, _ := HexToBytes(hash2Hex)

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
	_, err := DecodeCompactMerkleProof("!!!not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64url input")
	}
}

func TestDecodeCompactMerkleProof_DepthTooLarge(t *testing.T) {
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

func TestDecodeAndVerifyMerkleProof_RealProof(t *testing.T) {
	// End-to-end test using real proof data from the Truestamp system.
	// This validates that compact proof decoding + merkle verification
	// produce the correct root from real production data.

	// Step 1: Compute claims_hash = SHA256(0x11 || JCS(claims))
	claims := map[string]interface{}{
		"name":      "RODE Central macOS v2.0.110.zip",
		"hash":      "1abc53036d408fd969072b5f8ffaf4d7a5f3e1643c88f095c2991e58bea1e7d0",
		"hash_type": "sha256",
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshaling claims: %s", err)
	}
	jcsBytes, err := jcs.Transform(claimsJSON)
	if err != nil {
		t.Fatalf("JCS transform: %s", err)
	}
	claimsHash := BytesToHex(DomainHash(PrefixItemClaims, jcsBytes))

	expectedClaimsHash := "7d8912b9cecc1ab4ec379667337303ba8e252baee0f6eb702190346e0228c570"
	if claimsHash != expectedClaimsHash {
		t.Fatalf("claims_hash mismatch: got %s, want %s", claimsHash, expectedClaimsHash)
	}

	// Step 2: Compute item_hash using ComputeItemHash
	itemID := "01KMGXTC9CF141Y4MDYVJXR20N"
	metadataHash := "1eae1c546ac4c95c99ee45c1687bd36c0c8e05651d97ba60e69535fbef7de765"
	signingKeyID := "4ceefa4a"

	itemHash, err := ComputeItemHash(itemID, claimsHash, metadataHash, signingKeyID)
	if err != nil {
		t.Fatalf("computing item_hash: %s", err)
	}

	expectedItemHash := "c915b9542ec2f3bf0aa675cc4d573903ab36dbb6f2707ae3b17065b48748c0ad"
	if itemHash != expectedItemHash {
		t.Fatalf("item_hash mismatch: got %s, want %s", itemHash, expectedItemHash)
	}

	// Step 3: Decode compact merkle proof
	compactProof := "BAIXDapt1jqtKcGQbY2Naj6tPvfUJEcwFPe9MhYjDvdZNTr1KQpjCQnFZZQ3Cl1T8frYIxF5l4vtsIQQs0hHXAF25-_Gr0xFAC78T3EZNI42rbQcil-lRd7JgDsyg-1uZucdfHfCMxNGKuU_zplVOZlpG2pGvzKwyyIZTArfDiYyyg"

	proof, err := DecodeCompactMerkleProof(compactProof)
	if err != nil {
		t.Fatalf("decoding compact proof: %s", err)
	}
	if len(proof) != 4 {
		t.Fatalf("expected 4 proof steps, got %d", len(proof))
	}

	// Step 4: Verify the merkle proof against the expected root
	expectedMerkleRoot := "80a0901124f5af2436fcc35233f6956d7e12706d0e6e760810c27941671603de"

	ok, err := VerifyMerkleProof(itemHash, proof, expectedMerkleRoot)
	if err != nil {
		t.Fatalf("verifying merkle proof: %s", err)
	}
	if !ok {
		t.Error("real compact proof should verify against expected merkle root")
	}
}