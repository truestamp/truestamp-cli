// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// --- [E.13] Step 5: inclusion proof walk ---

func stepInclusionProof(r *Report, bundle *proof.Bundle, subjectHash string) {
	if bundle.IsBlockLike() {
		r.skip(groupInclusion, CatCryptographic,
			"not applicable: a block-like subject IS the block, so there is no leaf to prove")
		return
	}
	if subjectHash == "" {
		r.fail(groupInclusion, CatCryptographic, "cannot verify: the subject hash could not be derived")
		return
	}
	steps, err := tscrypto.DecodeCompactMerkleProof(bundle.InclusionProof)
	if err != nil {
		r.fail(groupInclusion, CatCryptographic, fmt.Sprintf("could not decode inclusion_proof: %s", err))
		return
	}
	derived, err := tscrypto.WalkMerkleProof(subjectHash, steps)
	if err != nil {
		r.fail(groupInclusion, CatCryptographic, fmt.Sprintf("could not walk inclusion_proof: %s", err))
		return
	}
	r.check(groupInclusion, CatCryptographic,
		tscrypto.SecureEqual(derived, bundle.Block.MerkleRoot),
		fmt.Sprintf("%s; derived root matches block.merkle_root", stepsLabel(len(steps))),
		fmt.Sprintf("derived root does not match block.merkle_root (derived %s)", derived))
}

// --- [E.14] Step 6: block hash derivation ---
//
//	metadata_hash = SHA-256(0x33 || JCS(block.metadata))
//	block_hash    = SHA-256(0x32
//	                  || len32(id)                  || id       36-byte UUIDv7
//	                  || len32(previous_block_hash) || ...      32 bytes
//	                  || len32(merkle_root)         || ...      32 bytes
//	                  || len32(metadata_hash)       || ...      32 bytes, derived
//	                  || len32(signing_key_id)      || ...       4 bytes
//
// The preimage is 157 bytes, always. All five fields are required, and the
// metadata map is one of them, carried rather than digested.

func stepBlockHash(r *Report, bundle *proof.Bundle) string {
	hash, size, reason := blockHashFromMap(bundle.Block)
	if reason != "" {
		r.fail(groupBlockHash, CatCryptographic, reason)
		return ""
	}
	r.pass(groupBlockHash, CatCryptographic,
		"block metadata hash derived (0x33) from JCS(block.metadata), which the bundle carries")
	r.pass(groupBlockHash, CatCryptographic,
		fmt.Sprintf("block hash (0x32) derived from %d-byte preimage: %s", size, hash))
	return hash
}

// blockHashFromMap is the one place a block hash is computed, used for the
// containing block, every block_path entry, the block witness, and the
// signing key event's block. reason is "" on success.
func blockHashFromMap(b proof.BlockMap) (hash string, preimageSize int, reason string) {
	if !b.IsMap() {
		return "", 0, "block is not a map"
	}
	hexErr := firstHexError(
		hexField{name: "previous_block_hash", obj: b.Fields, key: "previous_block_hash"},
		hexField{name: "merkle_root", obj: b.Fields, key: "merkle_root"},
		hexField{name: "signing_key_id", obj: b.Fields, key: "signing_key_id"},
	)
	switch {
	case b.Metadata == nil:
		return "", 0, "block carries no metadata map, so its metadata hash cannot be derived"
	case !b.Fields.IsString("id") || !b.Fields.IsString("previous_block_hash") ||
		!b.Fields.IsString("merkle_root") || !b.Fields.IsString("signing_key_id"):
		return "", 0, "block is missing one of the five required fields (id, previous_block_hash, merkle_root, metadata, signing_key_id)"
	case hexErr != "":
		// Without this row an uppercase merkle_root would surface at E.13
		// as "derived root does not match", which reads as a forgery
		// accusation against a bundle whose only defect is that it shouted.
		return "", 0, "invalid_hex_encoding: cannot derive the block hash: " + hexErr
	}
	canonicalMeta, _, err := jcs.Canonicalize(b.Metadata)
	if err != nil {
		return "", 0, fmt.Sprintf("cannot canonicalize block metadata: %s", err)
	}
	metaHash := tscrypto.ComputeBlockMetadataHash(canonicalMeta)
	hash, err = tscrypto.ComputeBlockHash(b.ID, b.PreviousBlockHash, b.MerkleRoot, metaHash, b.SigningKeyID)
	if err != nil {
		return "", 0, fmt.Sprintf("cannot derive the block hash: %s", err)
	}
	preimageSize = 1 + (4 + len(b.ID)) + (4 + len(b.PreviousBlockHash)/2) + (4 + len(b.MerkleRoot)/2) + (4 + 32) + (4 + len(b.SigningKeyID)/2)
	return hash, preimageSize, ""
}

// --- [E.15] Step 7: epoch proof walks ---
//
// A block is not committed to a blockchain by itself. Blocks are gathered
// into an epoch, an epoch has its own Merkle tree over block hashes, and it
// is the epoch ROOT that goes on chain. Both chains name the committed
// value with the same key, `epoch_merkle_root`, so nothing here dispatches
// on chain to find the root.

func walkEpochProof(r *Report, entry *proof.Commitment, blockHash, group, category string) bool {
	chain := entry.Chain
	hexErr := firstHexError(hexField{name: "epoch_merkle_root", obj: entry.Fields, key: "epoch_merkle_root"})
	switch {
	case blockHash == "":
		r.fail(group, category, fmt.Sprintf("%s: no block hash to walk from", chain))
		return false
	case hexErr != "":
		r.fail(group, category, fmt.Sprintf("%s: invalid_hex_encoding: %s", chain, hexErr))
		return false
	}
	steps, err := tscrypto.DecodeCompactMerkleProof(entry.EpochProof)
	if err != nil {
		r.fail(group, category, fmt.Sprintf("%s: could not decode epoch_proof: %s", chain, err))
		return false
	}
	derived, err := tscrypto.WalkMerkleProof(blockHash, steps)
	if err != nil {
		r.fail(group, category, fmt.Sprintf("%s: could not walk epoch_proof: %s", chain, err))
		return false
	}
	return r.check(group, category,
		tscrypto.SecureEqual(derived, entry.EpochMerkleRoot),
		fmt.Sprintf("%s: %s; derived root matches epoch_merkle_root", chain, stepsLabel(len(steps))),
		fmt.Sprintf("%s: derived epoch root does not match (derived %s)", chain, derived))
}
