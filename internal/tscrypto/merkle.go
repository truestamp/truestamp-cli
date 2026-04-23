// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const maxProofDepth = 64

// VerifyMerkleProof walks an RFC 6962 Merkle proof from leaf to root.
// Proof elements are in "l:hex" or "r:hex" format.
// Returns true if the computed root matches expectedRootHex.
func VerifyMerkleProof(itemHashHex string, proof []string, expectedRootHex string) (bool, error) {
	leafBytes, err := HexToBytes(itemHashHex)
	if err != nil {
		return false, fmt.Errorf("decoding item_hash: %w", err)
	}

	// Leaf hash: SHA256(0x00 || item_hash_bytes)
	current := DomainHash(PrefixMerkleLeaf, leafBytes)

	// Walk proof path
	for _, element := range proof {
		idx := strings.IndexByte(element, ':')
		if idx < 0 {
			return false, fmt.Errorf("invalid proof element: %s", element)
		}
		direction := element[:idx]
		siblingBytes, err := HexToBytes(element[idx+1:])
		if err != nil {
			return false, fmt.Errorf("decoding sibling hash: %w", err)
		}

		var nodeData []byte
		switch direction {
		case "l":
			nodeData = append(siblingBytes, current...)
		case "r":
			nodeData = append(current, siblingBytes...)
		default:
			return false, fmt.Errorf("invalid direction: %s", direction)
		}
		current = DomainHash(PrefixMerkleInternal, nodeData)
	}

	return HexEqual(BytesToHex(current), expectedRootHex), nil
}

// DecodeCompactMerkleProof decodes a compact base64url-encoded Merkle proof
// into the standard ["l:hex", "r:hex", ...] format.
//
// Binary format:
//   - Byte 0: depth (number of proof steps, 0-64)
//   - Next ceil(depth/8) bytes: direction bitfield (little-endian)
//     bit=0 means left sibling ("l:"), bit=1 means right sibling ("r:")
//   - Remaining: depth * 32 bytes of raw sibling hashes
func DecodeCompactMerkleProof(base64urlProof string) ([]string, error) {
	data, err := base64.RawURLEncoding.DecodeString(base64urlProof)
	if err != nil {
		return nil, fmt.Errorf("base64url decode failed: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty proof data")
	}

	depth := int(data[0])

	// Empty proof (depth == 0)
	if depth == 0 {
		if len(data) != 1 {
			return nil, fmt.Errorf("depth 0 proof should be exactly 1 byte, got %d", len(data))
		}
		return []string{}, nil
	}

	if depth > maxProofDepth {
		return nil, fmt.Errorf("proof depth %d exceeds maximum %d", depth, maxProofDepth)
	}

	bitfieldBytes := (depth + 7) / 8
	expectedHashBytes := depth * 32
	expectedLen := 1 + bitfieldBytes + expectedHashBytes

	if len(data) != expectedLen {
		return nil, fmt.Errorf("expected %d bytes for depth %d proof, got %d", expectedLen, depth, len(data))
	}

	// Extract bitfield and hashes
	bitfield := data[1 : 1+bitfieldBytes]
	hashes := data[1+bitfieldBytes:]

	proof := make([]string, depth)
	for i := range depth {
		// Read direction bit (little-endian bitfield)
		byteIdx := i / 8
		bitIdx := uint(i % 8)
		bit := (bitfield[byteIdx] >> bitIdx) & 1

		direction := "l"
		if bit == 1 {
			direction = "r"
		}

		hashBytes := hashes[i*32 : (i+1)*32]
		proof[i] = direction + ":" + BytesToHex(hashBytes)
	}

	return proof, nil
}
