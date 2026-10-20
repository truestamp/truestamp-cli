// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package bitcoin holds the strict readers of the Bitcoin bytes a commitment
// entry carries, for the Bitcoin evidence of DERIVE (SPEC.md section 10.2,
// step 9 (d)): a raw_transaction and a txoutproof, each judged well formed
// or not by the rules that step lists, and the values the step reads out
// of a well-formed one. It reads only bytes in memory and never the
// network.
//
//	| File            | What it reads                                                                 |
//	|-----------------|-------------------------------------------------------------------------------|
//	| bitcoin.go      | SHA-256d, display order and the compact size                                 |
//	| transaction.go  | a raw_transaction: its rules, its OP_RETURN, and rev(SHA-256d) of its bytes   |
//	| merkle_block.go | a txoutproof: its rules, the partial Merkle tree's walk and the matched hash  |
//	| header.go       | the 80-octet header: its fields, its hash and the target its nBits encodes    |
//
// SHA-256d(x) is SHA-256(SHA-256(x)), and rev(x) is x with its bytes in
// reverse order (SPEC.md section 2.2). SHA-256d is a hash an outside
// protocol fixes, so it is outside the byte-prefix registry (section 4.1):
// [sha256d] is this package's one use of crypto/sha256, and
// internal/registry/rule2_test.go names it, with each of its calls, as an
// exemption from its check that crypto/sha256 is used only in the
// registry's digest. A Bitcoin hash is written in display order, rev of
// the order it is hashed in, as a proof's members are (the encoding row of
// section 7.3); [Display] writes one.
//
// The compact size is Bitcoin's variable-length integer: one octet below
// 0xfd, or 0xfd, 0xfe or 0xff followed by 2, 4 or 8 octets little-endian.
// Step 9 (d) requires every compact size in a raw_transaction and a
// txoutproof in its shortest form, so [compactSize] reads only that form.
//
// A structure that is not well formed is refused with a [Refusal], the
// first rule it breaks. The rules of a raw_transaction are named as
// vectors/corpus/bitcoin/ names them (SPEC.md section 13.2): parse, length
// and op_return. SPEC.md names no rule of a txoutproof, so each clause of
// step 9 (d)'s bullets is named here: parse and total_txs (the first
// bullet), partial_tree and duplicate_pair (the second), matches and
// merkle_root (the third) and target (the fourth).
package bitcoin

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// Refusal is the first rule of a well-formed raw_transaction or txoutproof
// (SPEC.md section 10.2, step 9 (d)) that bytes break. Its text is the
// rule's name.
type Refusal string

// Error returns the rule's name.
func (r Refusal) Error() string { return string(r) }

// The rules a raw_transaction or a txoutproof can break (see the package's
// doc).
const (
	// RefusedParse: the bytes do not parse completely as the structure,
	// with every compact size in its shortest form and nothing after.
	RefusedParse Refusal = "parse"
	// RefusedLength: a raw_transaction is exactly 64 octets long
	// (CVE-2017-12842).
	RefusedLength Refusal = "length"
	// RefusedOpReturn: a raw_transaction has no output script that begins
	// with 0x6a, or more than one, or its one such script is not exactly
	// 0x6a 0x20 followed by 32 bytes.
	RefusedOpReturn Refusal = "op_return"
	// RefusedTotalTxs: a txoutproof's total_txs is not from 1 to
	// BITCOIN_MAX_TOTAL_TXS.
	RefusedTotalTxs Refusal = "total_txs"
	// RefusedPartialTree: a txoutproof's partial Merkle tree does not walk
	// as Bitcoin Core's CPartialMerkleTree walks one, using every hash and
	// every flag octet, with the bits after the last one used zero.
	RefusedPartialTree Refusal = "partial_tree"
	// RefusedDuplicatePair: a node of the walk has a right child equal to
	// its left (CVE-2012-2459).
	RefusedDuplicatePair Refusal = "duplicate_pair"
	// RefusedMatches: the walk does not match exactly one transaction.
	RefusedMatches Refusal = "matches"
	// RefusedMerkleRoot: the root the walk gives is not the header's
	// Merkle root field.
	RefusedMerkleRoot Refusal = "merkle_root"
	// RefusedTarget: the header does not meet the target its nBits field
	// encodes.
	RefusedTarget Refusal = "target"
)

// sha256d is SHA-256d(data), SHA-256 applied twice (SPEC.md section 2.2),
// in the order it is hashed in.
func sha256d(data []byte) [32]byte {
	sum := data
	var out [32]byte
	for range 2 {
		out = sha256.Sum256(sum)
		sum = out[:]
	}
	return out
}

// Display writes a Bitcoin hash in display order: rev(hash) as lowercase
// hex (the encoding row of SPEC.md section 7.3).
func Display(hash [32]byte) string {
	var reversed [32]byte
	for i, b := range hash {
		reversed[31-i] = b
	}
	return hex.EncodeToString(reversed[:])
}

// compactSize reads a compact size in its shortest form from the front of
// b: its value and the bytes after it, or ok false when b runs out or the
// size is not in its shortest form.
func compactSize(b []byte) (n uint64, rest []byte, ok bool) {
	if len(b) == 0 {
		return 0, nil, false
	}
	var width int
	var least uint64
	switch b[0] {
	case 0xfd:
		width, least = 2, 0xfd
	case 0xfe:
		width, least = 4, 0x10000
	case 0xff:
		width, least = 8, 0x100000000
	default:
		return uint64(b[0]), b[1:], true
	}
	if len(b) < 1+width {
		return 0, nil, false
	}
	var octets [8]byte
	copy(octets[:], b[1:1+width])
	n = binary.LittleEndian.Uint64(octets[:])
	if n < least {
		return 0, nil, false
	}
	return n, b[1+width:], true
}

// take splits n bytes off the front of b: those bytes and the rest, or ok
// false when b holds fewer than n.
func take(b []byte, n uint64) (head, rest []byte, ok bool) {
	if n > uint64(len(b)) {
		return nil, nil, false
	}
	return b[:n], b[n:], true
}
