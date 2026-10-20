// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file reads the 80-octet block header at the front of a txoutproof
// (SPEC.md section 10.2, step 9 (d)). Its fields are, by octet:
//
//	| Octets   | Field                                          |
//	|----------|------------------------------------------------|
//	| 0 to 3   | the version                                    |
//	| 4 to 35  | the previous block hash                        |
//	| 36 to 67 | the Merkle root field                          |
//	| 68 to 71 | nTime, little-endian Unix seconds              |
//	| 72 to 75 | nBits, little-endian                           |
//	| 76 to 79 | the nonce                                      |
//
// header_hash is rev(SHA-256d(header)) and header_merkle_root rev(the
// Merkle root field) (section 3.4), each in display order ([Display]), and
// header_time is nTime formatted as TIME.
//
// The target nBits encodes ([Target]): with n the nBits value, e = n >> 24,
// and w = n & 0x007fffff, shifted right by 8 x (3 - e) bits when e is at
// most 3,
//
//   - the encoding is negative when w is not 0 and n & 0x00800000 is not 0;
//   - it overflows when w is not 0 and e > 34, or w > 0xff and e > 33, or
//     w > 0xffff and e > 32;
//   - the target is w when e is at most 3, and w x 256^(e - 3) otherwise.
//
// The header fails ([MeetsTarget]) when its encoding is negative or
// overflows, when the target is 0, or when rev(SHA-256d(header)), read as a
// big-endian integer, is above the target.

package bitcoin

import (
	"encoding/binary"
	"errors"
	"math/big"
)

// HeaderOctets is the length of a block header.
const HeaderOctets = 80

// Header is an 80-octet block header.
type Header [HeaderOctets]byte

// The offsets of the header's fields.
const (
	merkleRootAt = 36
	nTimeAt      = 68
	nBitsAt      = 72
)

// The two ways an nBits value fails to encode a target.
var (
	// ErrNegativeTarget: the encoding is negative.
	ErrNegativeTarget = errors.New("the nBits encoding is negative")
	// ErrTargetOverflow: the encoding overflows.
	ErrTargetOverflow = errors.New("the nBits encoding overflows")
)

// HeaderHash is SHA-256d(header), in the order it is hashed in; rev of it
// is header_hash.
func HeaderHash(header Header) [32]byte {
	return sha256d(header[:])
}

// MerkleRootField is the header's Merkle root field, octets 36 to 67, in
// the order it is hashed in; rev of it is header_merkle_root.
func MerkleRootField(header Header) [32]byte {
	return [32]byte(header[merkleRootAt : merkleRootAt+32])
}

// HeaderTime is the header's nTime, in Unix seconds.
func HeaderTime(header Header) uint32 {
	return binary.LittleEndian.Uint32(header[nTimeAt:])
}

// NBits is the header's nBits value.
func NBits(header Header) uint32 {
	return binary.LittleEndian.Uint32(header[nBitsAt:])
}

// Target is the target the nBits value n encodes (see the file's doc), or
// [ErrNegativeTarget] or [ErrTargetOverflow].
func Target(n uint32) (*big.Int, error) {
	e := n >> 24
	w := n & 0x007fffff
	if e <= 3 {
		w >>= 8 * (3 - e)
	}
	switch {
	case w != 0 && n&0x00800000 != 0:
		return nil, ErrNegativeTarget
	case w != 0 && (e > 34 || (w > 0xff && e > 33) || (w > 0xffff && e > 32)):
		return nil, ErrTargetOverflow
	}
	target := new(big.Int).SetUint64(uint64(w))
	if e > 3 {
		target.Lsh(target, uint(8*(e-3)))
	}
	return target, nil
}

// MeetsTarget reports whether header meets the target its nBits field
// encodes (see the file's doc).
func MeetsTarget(header Header) bool {
	return meetsTarget(HeaderHash(header), NBits(header))
}

// meetsTarget reports whether a header whose SHA-256d is hash, in the
// order it is hashed in, meets the target n encodes: the encoding is
// neither negative nor overflows, the target is not 0, and rev(hash), read
// as a big-endian integer, is not above it.
func meetsTarget(hash [32]byte, n uint32) bool {
	target, err := Target(n)
	if err != nil || target.Sign() == 0 {
		return false
	}
	var displayed [32]byte
	for i, b := range hash {
		displayed[31-i] = b
	}
	return new(big.Int).SetBytes(displayed[:]).Cmp(target) <= 0
}
