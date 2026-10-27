// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package tscrypto holds the hash and encoding helpers of the earlier
// bundle layout that inspect, convert keyid and convert merkle print, the
// constant-time comparisons, the claims hash_type table of SPEC.md section
// 10.2, step 13 (EXPECTED_HASH), that items create checks a hash against,
// and the UUIDv7 and ULID time readers. This CLI's verifier, which takes
// its prefix bytes from SPEC.md section 4, is internal/verifier and
// internal/registry.
package tscrypto

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrNotLowercaseHex is the sentinel every refusal of [ValidateLowercaseHex]
// wraps, so a caller can tell "this field is not lowercase hex" from any
// other failure with errors.Is rather than by matching an error string.
var ErrNotLowercaseHex = errors.New("not lowercase hex")

// PrefixKeyID is the domain-separation prefix of [ComputeKeyID], the key
// id of the earlier bundle layout that `inspect` and `convert keyid` print.
// It is not the key id of SPEC.md section 3.3, whose prefix the registry
// of internal/registry holds.
const PrefixKeyID = 0x51

// ValidateLowercaseHex reports whether s is lowercase hex, the encoding
// a proof gives every hash (the HEX classes of SPEC.md section 2.4
// (Pattern classes)), which `proofs convert` holds each hash slot to
// before it writes the slot's bytes. It returns nil for a conforming value
// and, for a non-conforming one, an error naming the first offending byte
// and its offset so a report can say which character is wrong rather than
// only that the field "is invalid".
//
// Empty is accepted: an absent value is not an encoding defect, and
// reporting "" here would claim the document carries a malformed value
// where it carries none.
//
// Uppercase is called out separately from a non-hex byte because the two
// are different mistakes with different fixes, and because uppercase is
// the one Go would otherwise accept: hex.DecodeString is case-insensitive,
// so "F2C39DF9" decodes to the same four bytes as "f2c39df9", and a value
// the pattern classes refuse would be written as if it conformed.
func ValidateLowercaseHex(s string) error {
	if len(s)%2 != 0 {
		return fmt.Errorf("%w: odd length %d", ErrNotLowercaseHex, len(s))
	}
	for i := range len(s) {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		if c >= 'A' && c <= 'F' {
			return fmt.Errorf("%w: uppercase %q at offset %d", ErrNotLowercaseHex, string(c), i)
		}
		return fmt.Errorf("%w: %q at offset %d is not a hex digit", ErrNotLowercaseHex, string(c), i)
	}
	return nil
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

// HexEqual reports whether two hex strings are equal, ignoring ASCII case in
// the range 'A'-'F'.
//
// The comparison is constant-time in the CONTENTS of the operands: every byte
// is folded and accumulated, and only the length short-circuits (a length
// difference is not secret).
//
// Nothing is hex-decoded, so non-hex and odd-length operands compare on hex
// TEXT rather than decoded bytes. In verify a proof's hex is held to lowercase
// by SHAPE's pattern classes (SPEC.md section 2.4), and in proofs convert by
// [ValidateLowercaseHex]; this fold accepts neither.
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

// ValidateClaimsHash checks that a claimed hash has both the length AND the
// lowercase-hex character set required by its named hash type.
//
// Appendix E.11 of the whitepaper: "check that the hex length equals twice the
// algorithm's output size and that the character set is lowercase hex, for one
// of the twelve registered algorithms". Callers MUST render a non-nil error as
// a warn and MUST NOT fail on it, every E.11 soft check is advisory.
//
// The scan is a byte loop, not a rune loop: a multi-byte UTF-8 hash trips the
// length check first rather than the charset check, but either way the caller
// warns, which is all E.11 constrains.
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

	for i := range len(hash) {
		c := hash[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return fmt.Errorf("expected %d lowercase hex characters for %s, got %q at offset %d",
			expectedHex, hashType, string(c), i)
	}

	return nil
}

// SecureEqual reports whether two strings are byte-for-byte equal, in
// constant time for equal-length inputs. It is the comparison Appendix E.4
// requires for every hash and digest a verifier derives against one the
// bundle carries: exact, with no case folding, so a mis-cased wire value
// never compares equal to a derived one (the E.4 sweep names the encoding
// defect; this refuses to paper over it).
func SecureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
