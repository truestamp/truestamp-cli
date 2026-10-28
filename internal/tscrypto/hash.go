// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package tscrypto holds small hash and encoding helpers: a one-byte
// domain-separated SHA-256, the constant-time comparisons, and the UUIDv7
// time reader convert id uses.
// It verifies nothing: this CLI's verifier, which takes its prefix bytes
// from SPEC.md section 4, is internal/verifier and internal/registry.
package tscrypto

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

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

// HexEqual reports whether two hex strings are equal, ignoring ASCII case in
// the range 'A'-'F'.
//
// The comparison is constant-time in the CONTENTS of the operands: every byte
// is folded and accumulated, and only the length short-circuits (a length
// difference is not secret).
//
// Nothing is hex-decoded, so non-hex and odd-length operands compare on hex
// TEXT rather than decoded bytes. In verify a proof's hex is held to lowercase
// by SHAPE's pattern classes (SPEC.md section 2.4); this fold does not.
func HexEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range len(a) {
		diff |= foldASCIIHex(a[i]) ^ foldASCIIHex(b[i])
	}
	return subtle.ConstantTimeByteEq(diff, 0) == 1
}

// foldASCIIHex lowercases the ASCII range 'A'-'F' without branching on the
// byte's value, so the case fold does not reintroduce the side channel
// HexEqual exists to close.
func foldASCIIHex(c byte) byte {
	isUpperHex := subtle.ConstantTimeLessOrEq('A', int(c)) & subtle.ConstantTimeLessOrEq(int(c), 'F')
	return byte(subtle.ConstantTimeSelect(isUpperHex, int(c)+('a'-'A'), int(c)))
}

// SecureEqual reports whether two strings are byte-for-byte equal, in
// constant time for equal-length inputs. It is exact, with no case
// folding: a hash is lowercase hex (SPEC.md section 2.4 (Pattern
// classes)), so a mis-cased value never compares equal to a derived one.
func SecureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
