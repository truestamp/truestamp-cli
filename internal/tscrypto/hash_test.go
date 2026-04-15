// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"encoding/base64"
	"testing"
)

func TestDomainHash_ClaimsVector(t *testing.T) {
	// Test vector from docs/CRYPTOGRAPHY.md: SHA256(0x11 || "truestamp")
	result := BytesToHex(DomainHash(0x11, []byte("truestamp")))
	expected := "9d9443e5133052d9fc837e150d32e3094fc979f097922034a984ddfbe5247aca"
	if result != expected {
		t.Errorf("claims hash: got %s, want %s", result, expected)
	}
}

func TestDomainHash_EntropyVector(t *testing.T) {
	// Test vector: SHA256(0x21 || "truestamp")
	result := BytesToHex(DomainHash(0x21, []byte("truestamp")))
	expected := "22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831"
	if result != expected {
		t.Errorf("entropy hash: got %s, want %s", result, expected)
	}
}

func TestDomainHash_GenesisVector(t *testing.T) {
	// Test vector: SHA256(0x31 || "truestamp")
	result := BytesToHex(DomainHash(0x31, []byte("truestamp")))
	expected := "96965205fa528419d226675f8ad4a11f978d5523e31c462f4af0f1405d2ac8da"
	if result != expected {
		t.Errorf("genesis hash: got %s, want %s", result, expected)
	}
}

func TestDomainHash_DifferentPrefixesDiffer(t *testing.T) {
	data := []byte("same input")
	h11 := BytesToHex(DomainHash(0x11, data))
	h12 := BytesToHex(DomainHash(0x12, data))
	h21 := BytesToHex(DomainHash(0x21, data))
	if h11 == h12 || h11 == h21 || h12 == h21 {
		t.Error("different domain prefixes should produce different hashes")
	}
}

func TestComputeKeyID(t *testing.T) {
	// Test vector: known public key -> key_id
	pubkeyB64 := "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="
	pubkey, _ := base64.StdEncoding.DecodeString(pubkeyB64)
	result := ComputeKeyID(pubkey)
	expected := "4ceefa4a"
	if result != expected {
		t.Errorf("key_id: got %s, want %s", result, expected)
	}
}

func TestLenPrefix(t *testing.T) {
	data := []byte("hello")
	result := LenPrefix(data)
	// 4-byte big-endian length (5) + "hello"
	if len(result) != 9 {
		t.Errorf("len_prefix length: got %d, want 9", len(result))
	}
	if result[0] != 0 || result[1] != 0 || result[2] != 0 || result[3] != 5 {
		t.Errorf("len_prefix header: got %v, want [0 0 0 5]", result[:4])
	}
	if string(result[4:]) != "hello" {
		t.Errorf("len_prefix data: got %s, want hello", string(result[4:]))
	}
}

func TestLenPrefix_Empty(t *testing.T) {
	result := LenPrefix([]byte{})
	if len(result) != 4 {
		t.Errorf("len_prefix empty length: got %d, want 4", len(result))
	}
	for _, b := range result {
		if b != 0 {
			t.Errorf("len_prefix empty: got non-zero byte")
		}
	}
}

func TestComputeItemHash(t *testing.T) {
	// ComputeItemHash now takes 4 args: id, claimsHashHex, metadataHashHex, signingKeyIDHex
	result, err := ComputeItemHash(
		"01KKW15X4C9J75W2C30286JY00",
		"1bb362c54384315006393c34d10c4bd4e46abf1270707c35780872926e4e995d",
		"041c99740c6426282271e9341129e75393ed864575ab403a85e8120eed361661",
		"4ceefa4a",
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 64 {
		t.Errorf("item_hash should be 64 hex chars, got %d", len(result))
	}
}

func TestComputeBlockHash(t *testing.T) {
	// ComputeBlockHash field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	result, err := ComputeBlockHash(
		"019cf813-99b8-730a-84f1-5a711a9c355e",
		"48b7f261f5a2e9dbc121d3541947e847665160d8d7baf4a74daca4ab3d17a09d",
		"076e537400835632e9de64edbf94e5825c8042c52d71187ba6d4af4ea0a18749",
		"14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1",
		"4ceefa4a",
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 64 {
		t.Errorf("block_hash should be 64 hex chars, got %d", len(result))
	}
}

func TestHexToBytes_Empty(t *testing.T) {
	result, err := HexToBytes("")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d bytes", len(result))
	}
}

func TestHexEqual(t *testing.T) {
	if !HexEqual("abcdef", "ABCDEF") {
		t.Error("HexEqual should be case-insensitive")
	}
	if !HexEqual("AABB", "aabb") {
		t.Error("HexEqual should be case-insensitive (reverse)")
	}
	if HexEqual("abcdef", "abcdee") {
		t.Error("HexEqual should detect differences")
	}
	if HexEqual("abcdef", "abcd") {
		t.Error("HexEqual should detect length differences")
	}
	if !HexEqual("", "") {
		t.Error("HexEqual should match empty strings")
	}
	if !HexEqual("aabb", "aabb") {
		t.Error("HexEqual should match identical lowercase")
	}
}

func TestComputeItemHash_InvalidClaimsHex(t *testing.T) {
	_, err := ComputeItemHash("id", "ZZZZ", "aabb", "ccdd")
	if err == nil {
		t.Error("expected error for invalid claims_hash hex")
	}
}

func TestComputeItemHash_InvalidMetadataHex(t *testing.T) {
	_, err := ComputeItemHash("id", "aabb", "ZZZZ", "ccdd")
	if err == nil {
		t.Error("expected error for invalid metadata_hash hex")
	}
}

func TestComputeItemHash_InvalidSigningKeyIDHex(t *testing.T) {
	_, err := ComputeItemHash("id", "aabb", "ccdd", "ZZZZ")
	if err == nil {
		t.Error("expected error for invalid signing_key_id hex")
	}
}

func TestComputeBlockHash_InvalidPrevHash(t *testing.T) {
	// Field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	_, err := ComputeBlockHash("id", "ZZZZ", "aabb", "ccdd", "eeff")
	if err == nil {
		t.Error("expected error for invalid previous_block_hash hex")
	}
}

func TestComputeBlockHash_InvalidMerkleRoot(t *testing.T) {
	// Field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	_, err := ComputeBlockHash("id", "aabb", "ZZZZ", "ccdd", "eeff")
	if err == nil {
		t.Error("expected error for invalid merkle_root hex")
	}
}

func TestComputeBlockHash_InvalidMetadataHash(t *testing.T) {
	// Field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	_, err := ComputeBlockHash("id", "aabb", "ccdd", "ZZZZ", "eeff")
	if err == nil {
		t.Error("expected error for invalid metadata_hash hex")
	}
}

func TestComputeBlockHash_InvalidSigningKeyID(t *testing.T) {
	// Field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	_, err := ComputeBlockHash("id", "aabb", "ccdd", "eeff", "ZZZZ")
	if err == nil {
		t.Error("expected error for invalid signing_key_id hex")
	}
}

func TestComputeCommitmentDataHash(t *testing.T) {
	// ComputeCommitmentDataHash takes JCS bytes and returns a hex hash
	jcsData := []byte(`{"blockchain":"stellar","network":"testnet"}`)
	result := ComputeCommitmentDataHash(jcsData)
	if len(result) != 64 {
		t.Errorf("commitment_data_hash should be 64 hex chars, got %d", len(result))
	}
	// Same input should produce the same hash (deterministic)
	result2 := ComputeCommitmentDataHash(jcsData)
	if result != result2 {
		t.Errorf("commitment_data_hash should be deterministic, got %s and %s", result, result2)
	}
	// Different input should produce a different hash
	result3 := ComputeCommitmentDataHash([]byte(`{"blockchain":"bitcoin"}`))
	if result == result3 {
		t.Error("different input should produce different commitment_data_hash")
	}
}

func TestComputeCommitmentHash(t *testing.T) {
	// ComputeCommitmentHash takes id, commitmentDataHashHex, ownerID, signingKeyIDHex
	cdHash := ComputeCommitmentDataHash([]byte(`{"test":"data"}`))
	result, err := ComputeCommitmentHash(
		"019cf813-99b8-730a-84f1-5a711a9c355e",
		cdHash,
		"019cf813-99b8-730a-84f1-aaaaaaaaaaaa",
		"4ceefa4a",
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 64 {
		t.Errorf("commitment_hash should be 64 hex chars, got %d", len(result))
	}
}

func TestComputeCommitmentHash_InvalidCommitmentDataHash(t *testing.T) {
	_, err := ComputeCommitmentHash("id", "ZZZZ", "owner", "4ceefa4a")
	if err == nil {
		t.Error("expected error for invalid commitment_data_hash hex")
	}
}

func TestComputeCommitmentHash_InvalidSigningKeyID(t *testing.T) {
	_, err := ComputeCommitmentHash("id", "aabb", "owner", "ZZZZ")
	if err == nil {
		t.Error("expected error for invalid signing_key_id hex")
	}
}

func TestComputeProofHash(t *testing.T) {
	// ComputeProofHash takes version, keyIDHex, itemHashHex, blockHashes, commitmentHashes
	blockHash := "48b7f261f5a2e9dbc121d3541947e847665160d8d7baf4a74daca4ab3d17a09d"
	itemHash := "6de54da2c7d3bddcd8a6c2bb22fc0c44af3bbad47d351cfd43ac8dbb5fc1cc3d"
	commitHash := "14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1"

	hexResult, hashBytes, err := ComputeProofHash(
		1,
		"4ceefa4a",
		itemHash,
		[]string{blockHash},
		[]string{commitHash},
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(hexResult) != 64 {
		t.Errorf("proof_hash hex should be 64 chars, got %d", len(hexResult))
	}
	if len(hashBytes) != 32 {
		t.Errorf("proof_hash bytes should be 32 bytes, got %d", len(hashBytes))
	}
	// Hex and bytes should be consistent
	if BytesToHex(hashBytes) != hexResult {
		t.Errorf("proof_hash hex/bytes mismatch: hex=%s, bytes_hex=%s", hexResult, BytesToHex(hashBytes))
	}
}

func TestComputeProofHash_EmptyLists(t *testing.T) {
	hexResult, hashBytes, err := ComputeProofHash(
		1,
		"4ceefa4a",
		"6de54da2c7d3bddcd8a6c2bb22fc0c44af3bbad47d351cfd43ac8dbb5fc1cc3d",
		[]string{},
		[]string{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(hexResult) != 64 {
		t.Errorf("proof_hash hex should be 64 chars, got %d", len(hexResult))
	}
	if len(hashBytes) != 32 {
		t.Errorf("proof_hash bytes should be 32 bytes, got %d", len(hashBytes))
	}
}

func TestComputeProofHash_InvalidKeyID(t *testing.T) {
	_, _, err := ComputeProofHash(1, "ZZZZ", "aabb", nil, nil)
	if err == nil {
		t.Error("expected error for invalid key_id hex")
	}
}

func TestComputeProofHash_InvalidSubjectHash(t *testing.T) {
	_, _, err := ComputeProofHash(1, "4ceefa4a", "ZZZZ", nil, nil)
	if err == nil {
		t.Error("expected error for invalid subject_hash hex")
	}
}

func TestComputeProofHash_InvalidBlockHash(t *testing.T) {
	_, _, err := ComputeProofHash(1, "4ceefa4a", "aabb", []string{"ZZZZ"}, nil)
	if err == nil {
		t.Error("expected error for invalid block_hash hex")
	}
}

func TestComputeProofHash_InvalidCommitmentHash(t *testing.T) {
	_, _, err := ComputeProofHash(1, "4ceefa4a", "aabb", nil, []string{"ZZZZ"})
	if err == nil {
		t.Error("expected error for invalid commitment_hash hex")
	}
}

func TestComputeEntropyHash(t *testing.T) {
	// Test vector: SHA256(0x21 || "truestamp") from hash_test.go's DomainHash test
	result := ComputeEntropyHash([]byte("truestamp"))
	expected := "22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831"
	if result != expected {
		t.Errorf("ComputeEntropyHash: got %s, want %s", result, expected)
	}
}

func TestComputeEntropyHash_Deterministic(t *testing.T) {
	data := []byte(`{"pulse":{"outputValue":"ABC123"}}`)
	r1 := ComputeEntropyHash(data)
	r2 := ComputeEntropyHash(data)
	if r1 != r2 {
		t.Error("ComputeEntropyHash should be deterministic")
	}
	if len(r1) != 64 {
		t.Errorf("ComputeEntropyHash should return 64 hex chars, got %d", len(r1))
	}
}

func TestComputeEntropyHash_DifferentFromClaims(t *testing.T) {
	data := []byte("same input")
	entropy := ComputeEntropyHash(data)
	claims := BytesToHex(DomainHash(PrefixItemClaims, data))
	if entropy == claims {
		t.Error("entropy hash (0x21) and claims hash (0x11) should differ for same input")
	}
}

func TestComputeObservationHash(t *testing.T) {
	// ComputeObservationHash takes id, entropyHashHex, metadataHashHex, signingKeyIDHex
	result, err := ComputeObservationHash(
		"019cf813-99b8-730a-84f1-5a711a9c355e",
		"22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831",
		"041c99740c6426282271e9341129e75393ed864575ab403a85e8120eed361661",
		"4ceefa4a",
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 64 {
		t.Errorf("observation_hash should be 64 hex chars, got %d", len(result))
	}
	// Same inputs should produce the same hash (deterministic)
	result2, _ := ComputeObservationHash(
		"019cf813-99b8-730a-84f1-5a711a9c355e",
		"22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831",
		"041c99740c6426282271e9341129e75393ed864575ab403a85e8120eed361661",
		"4ceefa4a",
	)
	if result != result2 {
		t.Error("observation_hash should be deterministic")
	}
	// Different ID should produce a different hash
	result3, _ := ComputeObservationHash(
		"019cf813-99b8-730a-84f1-000000000000",
		"22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831",
		"041c99740c6426282271e9341129e75393ed864575ab403a85e8120eed361661",
		"4ceefa4a",
	)
	if result == result3 {
		t.Error("different ID should produce different observation_hash")
	}
}

func TestComputeObservationHash_InvalidEntropyHex(t *testing.T) {
	_, err := ComputeObservationHash("id", "ZZZZ", "aabb", "ccdd")
	if err == nil {
		t.Error("expected error for invalid entropy_hash hex")
	}
}

func TestComputeObservationHash_InvalidMetadataHex(t *testing.T) {
	_, err := ComputeObservationHash("id", "aabb", "ZZZZ", "ccdd")
	if err == nil {
		t.Error("expected error for invalid metadata_hash hex")
	}
}

func TestComputeObservationHash_InvalidSigningKeyIDHex(t *testing.T) {
	_, err := ComputeObservationHash("id", "aabb", "ccdd", "ZZZZ")
	if err == nil {
		t.Error("expected error for invalid signing_key_id hex")
	}
}

func TestComputeObservationHash_DifferentFromItemHash(t *testing.T) {
	// Same field values but different domain prefixes (0x23 vs 0x13) should differ
	obsHash, _ := ComputeObservationHash("id", "aabb", "ccdd", "eeff")
	itemHash, _ := ComputeItemHash("id", "aabb", "ccdd", "eeff")
	if obsHash == itemHash {
		t.Error("observation_hash (0x23) and item_hash (0x13) should differ for same fields")
	}
}
