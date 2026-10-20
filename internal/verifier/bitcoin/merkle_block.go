// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file reads a txoutproof (SPEC.md section 3.4), a BIP 37 merkleblock
// as bitcoind emits it, for the Bitcoin evidence of DERIVE (section 10.2,
// step 9 (d)). The bytes are well formed when these rules hold, checked in
// this order, the first that fails refusing them:
//
//	| Rule           | Holds when                                                                                       |
//	|----------------|--------------------------------------------------------------------------------------------------|
//	| parse          | they parse completely as a merkleblock: an 80-octet header; total_txs, 4 octets little-endian; a compact size and that many 32-octet hashes; a compact size and that many flag octets; and nothing after, with every compact size in its shortest form |
//	| total_txs      | total_txs is from 1 to BITCOIN_MAX_TOTAL_TXS (section 9)                                         |
//	| partial_tree   | the partial Merkle tree walks as Bitcoin Core's CPartialMerkleTree walks one: the walk uses every hash and every flag octet, and the bits after the last one used in the final flag octet are zero |
//	| duplicate_pair | no node of the walk has a right child equal to its left (CVE-2012-2459)                          |
//	| matches        | the walk matches exactly one transaction                                                         |
//	| merkle_root    | the root the walk gives equals the header's Merkle root field                                    |
//	| target         | the header meets the target its nBits field encodes (header.go)                                  |
//
// The walk (Bitcoin Core, src/merkleblock.cpp, CPartialMerkleTree's
// ExtractMatches and TraverseAndExtract): the tree over total_txs leaves
// has height h, the least with width(h) = 1, where width(g) = (total_txs +
// 2^g - 1) >> g. Before it, a tree with more hashes than total_txs, or
// with fewer flag bits than hashes, is refused, as Core refuses it. From
// the root, each node takes the next flag bit, the bits read from the
// lowest bit of each octet up. A node at height 0, or one whose bit is 0,
// takes the next hash as its value, and a leaf whose bit is 1 is a match.
// Any other node is SHA-256d of its left child and its right child; the
// right child is the left when the node has none (its position times 2,
// plus 1, is not below the width one level down), and a right child equal
// to the left is refused. A node that finds no flag bit or no hash left
// refuses the walk, which reports partial_tree or duplicate_pair in the
// order it meets them. After it, the hashes it used must be all of them,
// the flag octets it reached all of them, and the bits after the last one
// it used zero.
//
// The one hash a well-formed txoutproof matches is compared with SHA-256d
// of the raw_transaction (section 10.2, step 9 (d)).

package bitcoin

import (
	"encoding/binary"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// hashOctets is the length of a hash of the partial tree.
const hashOctets = 32

// MerkleBlock is what a well-formed txoutproof gives: its header, its
// total_txs and the one hash it matches, in the order it is hashed in.
type MerkleBlock struct {
	Header   Header
	TotalTxs uint32
	Matched  [32]byte
}

// ParseMerkleBlock judges b as a txoutproof (see the file's doc): what it
// gives when it is well formed, and otherwise the first rule it breaks, a
// [Refusal].
func ParseMerkleBlock(b []byte) (MerkleBlock, error) {
	header, rest, ok := take(b, HeaderOctets)
	if !ok {
		return MerkleBlock{}, RefusedParse
	}
	count, rest, ok := take(rest, 4)
	if !ok {
		return MerkleBlock{}, RefusedParse
	}
	total := binary.LittleEndian.Uint32(count)
	hashCount, rest, ok := compactSize(rest)
	if !ok || hashCount > uint64(len(rest))/hashOctets {
		return MerkleBlock{}, RefusedParse
	}
	hashBytes, rest, _ := take(rest, hashCount*hashOctets)
	flagCount, rest, ok := compactSize(rest)
	if !ok {
		return MerkleBlock{}, RefusedParse
	}
	flags, rest, ok := take(rest, flagCount)
	if !ok || len(rest) != 0 {
		return MerkleBlock{}, RefusedParse
	}
	if total < 1 || total > specdata.BitcoinMaxTotalTxs {
		return MerkleBlock{}, RefusedTotalTxs
	}
	hashes := make([][32]byte, hashCount)
	for i := range hashes {
		hashes[i] = [32]byte(hashBytes[i*hashOctets:])
	}
	root, matches, err := walkTree(total, hashes, flags)
	if err != nil {
		return MerkleBlock{}, err
	}
	if len(matches) != 1 {
		return MerkleBlock{}, RefusedMatches
	}
	h := Header(header)
	if root != MerkleRootField(h) {
		return MerkleBlock{}, RefusedMerkleRoot
	}
	if !MeetsTarget(h) {
		return MerkleBlock{}, RefusedTarget
	}
	return MerkleBlock{Header: h, TotalTxs: total, Matched: matches[0]}, nil
}

// partialTree is the state of one walk: the tree's total_txs, its hashes
// and flag octets, how many of each it has used, and the hashes matched.
type partialTree struct {
	total      uint32
	hashes     [][32]byte
	flags      []byte
	bitsUsed   int
	hashesUsed int
	matches    [][32]byte
}

// walkTree walks the partial tree of total leaves over hashes and flags
// (see the file's doc): its root and the hashes it matches, or
// [RefusedPartialTree] or [RefusedDuplicatePair].
func walkTree(total uint32, hashes [][32]byte, flags []byte) ([32]byte, [][32]byte, error) {
	bits := 8 * len(flags)
	if len(hashes) > int(total) || bits < len(hashes) {
		return [32]byte{}, nil, RefusedPartialTree
	}
	tree := &partialTree{total: total, hashes: hashes, flags: flags}
	height := 0
	for width(total, height) > 1 {
		height++
	}
	root, err := traverse(tree, height, 0)
	if err != nil {
		return [32]byte{}, nil, err
	}
	if tree.hashesUsed != len(hashes) || (tree.bitsUsed+7)/8 != len(flags) {
		return [32]byte{}, nil, RefusedPartialTree
	}
	for i := tree.bitsUsed; i < bits; i++ {
		if flag(flags, i) {
			return [32]byte{}, nil, RefusedPartialTree
		}
	}
	return root, tree.matches, nil
}

// width is the number of nodes at height of a tree over total leaves.
func width(total uint32, height int) uint64 {
	return (uint64(total) + 1<<height - 1) >> height
}

// flag is bit i of flags: bit i mod 8 of octet i div 8, from the lowest.
func flag(flags []byte, i int) bool {
	return flags[i/8]>>(i%8)&1 == 1
}

// traverse is the node at height and position of the tree's walk: its
// hash, or the refusal of the first node that finds no flag bit or hash
// left, or that has a right child equal to its left.
func traverse(tree *partialTree, height, position int) ([32]byte, error) {
	if tree.bitsUsed >= 8*len(tree.flags) {
		return [32]byte{}, RefusedPartialTree
	}
	parentOfMatch := flag(tree.flags, tree.bitsUsed)
	tree.bitsUsed++
	if height == 0 || !parentOfMatch {
		if tree.hashesUsed >= len(tree.hashes) {
			return [32]byte{}, RefusedPartialTree
		}
		hash := tree.hashes[tree.hashesUsed]
		tree.hashesUsed++
		if height == 0 && parentOfMatch {
			tree.matches = append(tree.matches, hash)
		}
		return hash, nil
	}
	left, err := traverse(tree, height-1, 2*position)
	if err != nil {
		return [32]byte{}, err
	}
	right := left
	if uint64(2*position+1) < width(tree.total, height-1) {
		if right, err = traverse(tree, height-1, 2*position+1); err != nil {
			return [32]byte{}, err
		}
		if right == left {
			return [32]byte{}, RefusedDuplicatePair
		}
	}
	return sha256d(append(left[:], right[:]...)), nil
}
