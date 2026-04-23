// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package bitcoin provides BIP 37 partial merkle tree verification
// and thin wrappers around btcsuite/btcd for transaction and txoutproof parsing.
package bitcoin

import (
	"math"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// MerkleResult holds the result of a partial merkle tree verification.
type MerkleResult struct {
	Valid        bool
	ComputedRoot *chainhash.Hash
	MatchedTxIDs []*chainhash.Hash
}

// VerifyPartialMerkleTree verifies a Bitcoin BIP 37 partial merkle tree.
// Takes the hashes and raw flag bytes from wire.MsgMerkleBlock directly.
func VerifyPartialMerkleTree(
	hashes []*chainhash.Hash,
	flagBytes []byte,
	totalTxs uint32,
	expectedRoot *chainhash.Hash,
) MerkleResult {
	if totalTxs == 0 {
		return MerkleResult{Valid: false}
	}

	t := &treeTraversal{
		hashes:   hashes,
		flags:    expandFlags(flagBytes),
		totalTxs: totalTxs,
	}

	root := t.traverse(calcTreeHeight(totalTxs), 0)

	// BIP 37: all hashes must be consumed
	if t.hashIdx != len(hashes) {
		return MerkleResult{Valid: false}
	}

	// BIP 37: all flag bits must be consumed (except byte-alignment padding)
	if (t.flagIdx+7)/8 != len(flagBytes) {
		return MerkleResult{Valid: false}
	}

	// BIP 37: in two-child internal nodes, left and right must never be equal
	if t.dupFound {
		return MerkleResult{Valid: false}
	}

	valid := root != nil && expectedRoot != nil && root.IsEqual(expectedRoot)
	return MerkleResult{
		Valid:        valid,
		ComputedRoot: root,
		MatchedTxIDs: t.matched,
	}
}

// treeTraversal holds mutable state during BIP 37 partial merkle tree traversal.
type treeTraversal struct {
	hashes   []*chainhash.Hash
	flags    []bool
	totalTxs uint32
	hashIdx  int
	flagIdx  int
	matched  []*chainhash.Hash
	dupFound bool
}

func (t *treeTraversal) nextFlag() bool {
	flag := false
	if t.flagIdx < len(t.flags) {
		flag = t.flags[t.flagIdx]
	}
	t.flagIdx++
	return flag
}

func (t *treeTraversal) nextHash() *chainhash.Hash {
	var hash *chainhash.Hash
	if t.hashIdx < len(t.hashes) {
		hash = t.hashes[t.hashIdx]
	}
	t.hashIdx++
	return hash
}

func (t *treeTraversal) traverse(height, pos int) *chainhash.Hash {
	flag := t.nextFlag()

	if height == 0 {
		hash := t.nextHash()
		if flag && uint32(pos) < t.totalTxs {
			t.matched = append(t.matched, hash)
		}
		return hash
	}

	if !flag {
		return t.nextHash()
	}

	// Internal node with flag=1: descend to children
	left := t.traverse(height-1, pos*2)

	rightPos := pos*2 + 1
	if uint32(rightPos) < treeWidth(height-1, t.totalTxs) {
		right := t.traverse(height-1, rightPos)
		if left != nil && right != nil && left.IsEqual(right) {
			t.dupFound = true
		}
		return combineHashes(left, right)
	}
	// No right child: duplicate left
	return combineHashes(left, left)
}

func expandFlags(flagBytes []byte) []bool {
	flags := make([]bool, 0, len(flagBytes)*8)
	for _, b := range flagBytes {
		for bit := range 8 {
			flags = append(flags, (b>>uint(bit))&1 == 1)
		}
	}
	return flags
}

func calcTreeHeight(totalTxs uint32) int {
	if totalTxs <= 1 {
		return 0
	}
	return int(math.Ceil(math.Log2(float64(totalTxs))))
}

func treeWidth(height int, totalTxs uint32) uint32 {
	return (totalTxs + (1 << uint(height)) - 1) >> uint(height)
}

func combineHashes(left, right *chainhash.Hash) *chainhash.Hash {
	if left == nil || right == nil {
		return nil
	}
	var combined [64]byte
	copy(combined[:32], left[:])
	copy(combined[32:], right[:])
	result := chainhash.DoubleHashH(combined[:])
	return &result
}
