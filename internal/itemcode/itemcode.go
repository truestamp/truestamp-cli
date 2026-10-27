// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package itemcode writes the item code of SPEC.md section 3.9 (The item
// code): a short name for an item, for people and links, computed from the
// 32 bytes of a recomputed item_hash, and read back from what a person
// types or pastes. No proof carries it, no verification step reads it, and
// it is never a verdict input, a report member or evidence; a match shows
// only that two item_hash values begin with the same 75 bits.
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

// ParseError is why [Parse] refused a value, one of the two constants
// below, which errors.Is compares.
type ParseError string

func (e ParseError) Error() string { return string(e) }

const (
	// ErrNotACode is a value that is not an item code at all.
	ErrNotACode ParseError = "not an item code"
	// ErrTypo is a value with the shape of an item code, four groups of
	// four ASCII letters or digits, that holds a letter outside
	// [Alphabet] (a U) or fails the check: a mistyped symbol or two
	// symbols exchanged.
	ErrTypo ParseError = "an item code with a mistyped or exchanged symbol"
)

// Parse reads a typed or pasted item code as SPEC.md section 3.9 (The item
// code) lets a reader, and nothing more, and returns it in its one
// spelling.
//
// The value is read exactly as given: nothing is trimmed. It must be ASCII,
// 16 letters or digits in four groups of four with at most one separator,
// a space, a hyphen or an underscore, at each of the three places between
// the groups and nowhere else, so it is at most 19 characters. A tab, a
// line feed, any other character and any character that is not ASCII make
// it not a code, [ErrNotACode]. Letters may be in any case, and I and L
// read as 1 and O as 0, in either case. A value of that shape whose 16
// symbols are not all of [Alphabet], or for which the loop of step 2 run
// over all 16 values does not end with r = 0, is [ErrTypo].
func Parse(s string) (string, error) {
	if len(s) < Length || len(s) > Length+3 {
		return "", ErrNotACode
	}
	symbols := make([]byte, 0, Length)
	i := 0
	for group := 0; group < 4; group++ {
		if i+4 > len(s) {
			return "", ErrNotACode
		}
		for _, c := range []byte(s[i : i+4]) {
			if !isASCIIAlnum(c) {
				return "", ErrNotACode
			}
			symbols = append(symbols, c)
		}
		i += 4
		if group < 3 && i < len(s) && isSeparator(s[i]) {
			i++
		}
	}
	if i != len(s) {
		return "", ErrNotACode
	}
	var r byte
	for j, c := range symbols {
		c = read(c)
		v, ok := symbolValue(c)
		if !ok {
			return "", ErrTypo
		}
		symbols[j] = c
		r = mulX(r) ^ v
	}
	if r != 0 {
		return "", ErrTypo
	}
	return string(symbols), nil
}

func isASCIIAlnum(c byte) bool {
	return '0' <= c && c <= '9' || 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

func isSeparator(c byte) bool { return c == ' ' || c == '-' || c == '_' }

// read is c in uppercase, then I and L as 1 and O as 0.
func read(c byte) byte {
	if 'a' <= c && c <= 'z' {
		c -= 'a' - 'A'
	}
	switch c {
	case 'I', 'L':
		return '1'
	case 'O':
		return '0'
	}
	return c
}

// symbolValue is the position of c in [Alphabet].
func symbolValue(c byte) (byte, bool) {
	for v := byte(0); v < byte(len(Alphabet)); v++ {
		if Alphabet[v] == c {
			return v, true
		}
	}
	return 0, false
}
