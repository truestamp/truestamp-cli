// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package tscrypto implements the Truestamp-specific cryptographic
// primitives used by proofs: SHA-256 with one-byte domain-separation
// prefixes (see docs/CRYPTOGRAPHY.md in truestamp-v2 for the prefix
// registry) and Ed25519 signature verification.
package tscrypto

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Domain separation prefix bytes per docs/CRYPTOGRAPHY.md.
const (
	PrefixMerkleLeaf      = 0x00
	PrefixMerkleInternal  = 0x01
	PrefixItemClaims      = 0x11
	PrefixItemMetadata    = 0x12
	PrefixItemHash        = 0x13
	PrefixEntropy         = 0x21
	PrefixEntropyMetadata = 0x22
	PrefixObservationHash = 0x23
	PrefixBlockMetadata   = 0x33
	PrefixBlockHash       = 0x32
	PrefixCommitmentData  = 0x34
	PrefixCommitmentHash  = 0x35
	PrefixKeyID           = 0x51
	PrefixProofHash       = 0x61
)

// HexToBytes decodes a hex string to bytes. Returns empty slice for empty input.
func HexToBytes(h string) ([]byte, error) {
	if h == "" {
		return []byte{}, nil
	}
	return hex.DecodeString(h)
}

// BytesToHex encodes bytes to lowercase hex string.
func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

// DomainHash computes SHA256(prefix_byte || data).
func DomainHash(prefix byte, data []byte) []byte {
	h := sha256.New()
	h.Write([]byte{prefix})
	h.Write(data)
	return h.Sum(nil)
}

// ComputeKeyID derives key_id from public key: truncate4(SHA256(0x51 || pubkey)).
func ComputeKeyID(pubkey []byte) string {
	hash := DomainHash(PrefixKeyID, pubkey)
	return BytesToHex(hash[:4])
}

// ComputeEntropyHash computes SHA256(0x21 || JCS(entropy_data)).
func ComputeEntropyHash(jcsBytes []byte) string {
	return BytesToHex(DomainHash(PrefixEntropy, jcsBytes))
}

// ComputeEntropyMetadataHash computes SHA256(0x22 || JCS(metadata)).
func ComputeEntropyMetadataHash(jcsBytes []byte) string {
	return BytesToHex(DomainHash(PrefixEntropyMetadata, jcsBytes))
}

// ComputeObservationHash computes the length-prefixed observation hash with domain prefix 0x23.
// Field order: id, entropy_hash, metadata_hash, signing_key_id
// This mirrors ComputeItemHash but uses prefix 0x23 for entropy observations.
func ComputeObservationHash(id, entropyHashHex, metadataHashHex, signingKeyIDHex string) (string, error) {
	entropyBytes, err := HexToBytes(entropyHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding entropy_hash: %w", err)
	}
	metaBytes, err := HexToBytes(metadataHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding metadata_hash: %w", err)
	}
	keyIDBytes, err := HexToBytes(signingKeyIDHex)
	if err != nil {
		return "", fmt.Errorf("decoding signing_key_id: %w", err)
	}

	totalSize := (4 + len(id)) + (4 + len(entropyBytes)) + (4 + len(metaBytes)) + (4 + len(keyIDBytes))
	serialized := make([]byte, 0, totalSize)
	serialized = appendLenPrefixed(serialized, []byte(id))
	serialized = appendLenPrefixed(serialized, entropyBytes)
	serialized = appendLenPrefixed(serialized, metaBytes)
	serialized = appendLenPrefixed(serialized, keyIDBytes)

	return BytesToHex(DomainHash(PrefixObservationHash, serialized)), nil
}

// ComputeItemHash computes the length-prefixed item hash with domain prefix 0x13.
// Field order: id, claims_hash, metadata_hash, signing_key_id
func ComputeItemHash(id, claimsHashHex, metadataHashHex, signingKeyIDHex string) (string, error) {
	claimsBytes, err := HexToBytes(claimsHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding claims_hash: %w", err)
	}
	metaBytes, err := HexToBytes(metadataHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding metadata_hash: %w", err)
	}
	keyIDBytes, err := HexToBytes(signingKeyIDHex)
	if err != nil {
		return "", fmt.Errorf("decoding signing_key_id: %w", err)
	}

	totalSize := (4 + len(id)) + (4 + len(claimsBytes)) + (4 + len(metaBytes)) + (4 + len(keyIDBytes))
	serialized := make([]byte, 0, totalSize)
	serialized = appendLenPrefixed(serialized, []byte(id))
	serialized = appendLenPrefixed(serialized, claimsBytes)
	serialized = appendLenPrefixed(serialized, metaBytes)
	serialized = appendLenPrefixed(serialized, keyIDBytes)

	return BytesToHex(DomainHash(PrefixItemHash, serialized)), nil
}

// ComputeBlockHash computes the length-prefixed block hash with domain prefix 0x32.
// Field order: id, previous_block_hash, merkle_root, metadata_hash, signing_key_id
func ComputeBlockHash(id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex string) (string, error) {
	prevBytes, err := HexToBytes(prevHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding previous_block_hash: %w", err)
	}
	merkleBytes, err := HexToBytes(merkleRootHex)
	if err != nil {
		return "", fmt.Errorf("decoding merkle_root: %w", err)
	}
	metaBytes, err := HexToBytes(metadataHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding metadata_hash: %w", err)
	}
	keyIDBytes, err := HexToBytes(signingKeyIDHex)
	if err != nil {
		return "", fmt.Errorf("decoding signing_key_id: %w", err)
	}

	idBytes := []byte(id)
	totalSize := (4 + len(idBytes)) + (4 + len(prevBytes)) + (4 + len(merkleBytes)) + (4 + len(metaBytes)) + (4 + len(keyIDBytes))
	serialized := make([]byte, 0, totalSize)
	serialized = appendLenPrefixed(serialized, idBytes)
	serialized = appendLenPrefixed(serialized, prevBytes)
	serialized = appendLenPrefixed(serialized, merkleBytes)
	serialized = appendLenPrefixed(serialized, metaBytes)
	serialized = appendLenPrefixed(serialized, keyIDBytes)

	return BytesToHex(DomainHash(PrefixBlockHash, serialized)), nil
}

// ComputeCommitmentDataHash computes SHA256(0x34 || JCS(commitment_data)).
func ComputeCommitmentDataHash(jcsBytes []byte) string {
	return BytesToHex(DomainHash(PrefixCommitmentData, jcsBytes))
}

// ComputeCommitmentHash computes the length-prefixed commitment hash with domain prefix 0x35.
// Field order: id, commitment_data_hash, owner_id, signing_key_id
func ComputeCommitmentHash(id, commitmentDataHashHex, ownerID, signingKeyIDHex string) (string, error) {
	cdHashBytes, err := HexToBytes(commitmentDataHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding commitment_data_hash: %w", err)
	}
	keyIDBytes, err := HexToBytes(signingKeyIDHex)
	if err != nil {
		return "", fmt.Errorf("decoding signing_key_id: %w", err)
	}

	idBytes := []byte(id)
	ownerBytes := []byte(ownerID)
	totalSize := (4 + len(idBytes)) + (4 + len(cdHashBytes)) + (4 + len(ownerBytes)) + (4 + len(keyIDBytes))
	serialized := make([]byte, 0, totalSize)
	serialized = appendLenPrefixed(serialized, idBytes)
	serialized = appendLenPrefixed(serialized, cdHashBytes)
	serialized = appendLenPrefixed(serialized, ownerBytes)
	serialized = appendLenPrefixed(serialized, keyIDBytes)

	return BytesToHex(DomainHash(PrefixCommitmentHash, serialized)), nil
}

// ComputeProofHash builds the binary proof_hash payload and computes SHA256(0x61 || payload).
// Returns both the hex hash and raw hash bytes (raw needed for Ed25519 verification).
func ComputeProofHash(version byte, keyIDHex, subjectHashHex string, blockHashes, commitmentHashes []string) (string, []byte, error) {
	keyIDBytes, err := HexToBytes(keyIDHex)
	if err != nil {
		return "", nil, fmt.Errorf("decoding key_id: %w", err)
	}
	subjectHashBytes, err := HexToBytes(subjectHashHex)
	if err != nil {
		return "", nil, fmt.Errorf("decoding subject_hash: %w", err)
	}

	if len(blockHashes) > 65535 {
		return "", nil, fmt.Errorf("block count %d exceeds maximum 65535", len(blockHashes))
	}
	if len(commitmentHashes) > 65535 {
		return "", nil, fmt.Errorf("commitment count %d exceeds maximum 65535", len(commitmentHashes))
	}

	// Build payload: version(1) || key_id(4) || subject_hash(32) || N(2) || block_hashes(32*N) || M(2) || commitment_hashes(32*M)
	payload := make([]byte, 0, 1+4+32+2+32*len(blockHashes)+2+32*len(commitmentHashes))
	payload = append(payload, version)
	payload = append(payload, keyIDBytes...)
	payload = append(payload, subjectHashBytes...)

	payload = append(payload, byte(len(blockHashes)>>8), byte(len(blockHashes)))
	for _, bh := range blockHashes {
		bhBytes, err := HexToBytes(bh)
		if err != nil {
			return "", nil, fmt.Errorf("decoding block_hash: %w", err)
		}
		payload = append(payload, bhBytes...)
	}

	payload = append(payload, byte(len(commitmentHashes)>>8), byte(len(commitmentHashes)))
	for _, ch := range commitmentHashes {
		chBytes, err := HexToBytes(ch)
		if err != nil {
			return "", nil, fmt.Errorf("decoding commitment_hash: %w", err)
		}
		payload = append(payload, chBytes...)
	}

	hashBytes := DomainHash(PrefixProofHash, payload)
	return BytesToHex(hashBytes), hashBytes, nil
}

// BuildCompactProofPayload builds the compact proof signature payload
// and computes SHA256(0x61 || payload). The returned 32-byte hash is what
// the Ed25519 signature covers.
//
// Byte layout (big-endian throughout):
//
//	offset  size  field
//	0       1     v  (version, uint8)
//	1       2     t  (type code, uint16 BE)
//	3       4     kid (raw 4 bytes from b.kid hex-decoded)
//	7       8     ts_ms (timestamp in ms since Unix epoch, uint64 BE)
//	15      32    subject_hash
//	47      32    block_hash
//	79      2     N (epoch root count, uint16 BE)
//	81      32*N  epoch_roots (concatenated, in cx order)
//
// For block-like subjects (t ∈ {10, 11} — plain block and beacon),
// subject_hash == block_hash — the same 32 bytes appear in both slots.
// The `t` byte in the payload domain-separates block (t=10) and beacon
// (t=11) signatures for the same underlying block.
func BuildCompactProofPayload(version byte, typeCode uint16, keyIDHex string, timestampMs uint64, subjectHashHex, blockHashHex string, epochRootHexes []string) ([]byte, error) {
	keyIDBytes, err := HexToBytes(keyIDHex)
	if err != nil {
		return nil, fmt.Errorf("decoding key_id: %w", err)
	}
	subjectHashBytes, err := HexToBytes(subjectHashHex)
	if err != nil {
		return nil, fmt.Errorf("decoding subject_hash: %w", err)
	}
	blockHashBytes, err := HexToBytes(blockHashHex)
	if err != nil {
		return nil, fmt.Errorf("decoding block_hash: %w", err)
	}

	if len(epochRootHexes) > 65535 {
		return nil, fmt.Errorf("epoch root count %d exceeds maximum 65535", len(epochRootHexes))
	}

	payload := make([]byte, 0, 1+2+4+8+32+32+2+32*len(epochRootHexes))
	payload = append(payload, version)
	payload = append(payload, byte(typeCode>>8), byte(typeCode))
	payload = append(payload, keyIDBytes...)

	var tsBuf [8]byte
	binary.BigEndian.PutUint64(tsBuf[:], timestampMs)
	payload = append(payload, tsBuf[:]...)

	payload = append(payload, subjectHashBytes...)
	payload = append(payload, blockHashBytes...)

	payload = append(payload, byte(len(epochRootHexes)>>8), byte(len(epochRootHexes)))
	for _, er := range epochRootHexes {
		erBytes, err := HexToBytes(er)
		if err != nil {
			return nil, fmt.Errorf("decoding epoch_root: %w", err)
		}
		payload = append(payload, erBytes...)
	}

	hashBytes := DomainHash(PrefixProofHash, payload)
	return hashBytes, nil
}

// HexEqual compares two hex strings case-insensitively.
func HexEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'F' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'F' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// LenPrefix prepends a 4-byte big-endian length prefix to data.
func LenPrefix(data []byte) []byte {
	return appendLenPrefixed(nil, data)
}

// appendLenPrefixed appends a 4-byte big-endian length prefix followed by data to dst.
func appendLenPrefixed(dst, data []byte) []byte {
	var prefix [4]byte
	binary.BigEndian.PutUint32(prefix[:], uint32(len(data)))
	dst = append(dst, prefix[:]...)
	dst = append(dst, data...)
	return dst
}

// hashTypeInfo holds metadata for a supported hash type.
type hashTypeInfo struct {
	Bytes int
	Name  string
}

// hashTypes maps hash type keys to their expected output sizes.
var hashTypes = map[string]hashTypeInfo{
	"md5":      {Bytes: 16, Name: "MD5"},
	"sha1":     {Bytes: 20, Name: "SHA-1"},
	"sha224":   {Bytes: 28, Name: "SHA-224"},
	"sha256":   {Bytes: 32, Name: "SHA-256"},
	"sha384":   {Bytes: 48, Name: "SHA-384"},
	"sha512":   {Bytes: 64, Name: "SHA-512"},
	"sha3_224": {Bytes: 28, Name: "SHA3-224"},
	"sha3_256": {Bytes: 32, Name: "SHA3-256"},
	"sha3_384": {Bytes: 48, Name: "SHA3-384"},
	"sha3_512": {Bytes: 64, Name: "SHA3-512"},
	"blake2s":  {Bytes: 32, Name: "BLAKE2s"},
	"blake2b":  {Bytes: 64, Name: "BLAKE2b"},
}

// ValidateClaimsHash checks that a hex hash string has the correct length
// for the given hash type.
func ValidateClaimsHash(hash, hashType string) error {
	if hash == "" || hashType == "" {
		return nil
	}

	info, ok := hashTypes[hashType]
	if !ok {
		return fmt.Errorf("unknown hash type: %s", hashType)
	}

	expectedHex := info.Bytes * 2
	if len(hash) != expectedHex {
		return fmt.Errorf("expected %d hex characters for %s, got %d", expectedHex, hashType, len(hash))
	}

	return nil
}
