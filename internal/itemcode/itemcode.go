// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package itemcode writes the item code of SPEC.md section 3.9 (The item
// code): a short name for an item, for people and links, computed from the
// 32 bytes of a recomputed item_hash. No proof carries it, no verification
// step reads it, and it is never a verdict input, a report member or
// evidence; a match shows only that two item_hash values begin with the
// same 75 bits.
package itemcode

import "github.com/truestamp/truestamp-cli/internal/registry"

// Alphabet is the 32 symbols of SPEC.md section 3.9, step 3: a value's
// symbol is the character at its position.
const Alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// Length is the number of symbols in an item code: 15 values of 5 bits
// and the check value.
const Length = 16

// FromItemHash returns the item code of the item whose item_hash is the 32
// bytes h, in its one spelling: 16 uppercase symbols with no separator.
//
// Step 1 takes the leftmost 75 bits of h, most significant bit of the
// first byte first, as 15 values of 5 bits. Step 2 runs r = m(r) XOR v over
// them, from r = 0, where m multiplies by x in GF(2^5) modulo x^5 + x^2 + 1
// (shift left one bit, and XOR with 37 when the result is 32 or more); the
// check value is m(r), the final m included. Step 3 writes the 15 values
// and the check value with [Alphabet].
func FromItemHash(h [32]byte) string {
	var values [Length]byte
	var r byte
	for i := 0; i < Length-1; i++ {
		values[i] = fiveBits(h, i*5)
		r = mulX(r) ^ values[i]
	}
	values[Length-1] = mulX(r)
	out := make([]byte, Length)
	for i, v := range values {
		out[i] = Alphabet[v]
	}
	return string(out)
}

// OfItemRecord returns the item code of the item whose record is record,
// the value an item proof carries at /proof/subject/record: the code of
// its item_hash recomputed as H(0x11, JCS(record)), the row item_hash of
// SPEC.md section 4.3 (The table), never a hash a document carries. It
// returns the registry's refusal of a record it cannot hash.
func OfItemRecord(record map[string]any) (string, error) {
	itemHash, err := registry.RecordHash(registry.ItemHash, record)
	if err != nil {
		return "", err
	}
	return FromItemHash(itemHash), nil
}

// fiveBits returns the 5 bits of h that start at bit offset, counted from
// the most significant bit of h[0].
func fiveBits(h [32]byte, offset int) byte {
	var v byte
	for bit := offset; bit < offset+5; bit++ {
		v = v<<1 | (h[bit/8]>>(7-bit%8))&1
	}
	return v
}

// mulX is m(v) of SPEC.md section 3.9, step 2.
func mulX(v byte) byte {
	v <<= 1
	if v >= 32 {
		v ^= 37
	}
	return v
}
