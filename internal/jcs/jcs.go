// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package jcs writes JCS(x), the JSON Canonicalization Scheme of RFC 8785
// (SPEC.md sections 2.2 and 2.5), and the canonical form of SPEC.md
// section 9.
//
// [Encode] and [CanonicalForm] (encode.go and canonical_form.go) take a
// value that internal/strictjson read under the parsing rules of SPEC.md
// section 10.1, whose numbers rule J5 has already read as binary64 values.
// [Transform] takes bytes: it reads them with that same reader and writes
// the value with [Encode], so a text the rules refuse is refused, an
// integer of magnitude above MAX_INTEGER and below 10^21 among them, and
// every other number is written as its nearest binary64 value is. It is
// what `truestamp jcs` and `truestamp hash --jcs` canonicalize with.
//
// Everything else of RFC 8785, UTF-16 code-unit key ordering, string
// escaping and ES6 number formatting, is github.com/gowebpki/jcs's, which
// this package wraps and which no other package imports.
//
// [UnsafeIntegers] is the check `truestamp items create` makes of claims
// before it sends them: it names every integer above MAX_INTEGER in
// magnitude, the one bound of SPEC.md section 9.
package jcs

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// MaxSafeInteger is MAX_INTEGER, 2^53 - 1 (SPEC.md section 9), the bound
// [UnsafeIntegers] holds claims to: RFC 8785 Appendix B's SHOULD and
// JavaScript's Number.MAX_SAFE_INTEGER, which the Truestamp server enforces
// too (Truestamp.SafeIntegers in truestamp-v2). It is strictjson's
// MaxInteger, the bound rule J5 reads numbers under, so the CLI holds one
// bound in every path.
const MaxSafeInteger int64 = strictjson.MaxInteger

// maxSafeInteger is MaxSafeInteger as a big.Int: the literals being compared
// routinely overflow every fixed-width Go integer type (an unsigned 64-bit
// Stellar sequence, a 400-digit nonce), so the comparison cannot go through
// strconv.ParseInt.
var maxSafeInteger = big.NewInt(MaxSafeInteger)

// Transform returns JCS of the JSON text data: data read with
// [strictjson.Parse] (rules J2 to J6 of SPEC.md section 10.1) and written
// with [Encode]. A text the rules refuse gives the reader's
// [*strictjson.Rejection]: a repeated member name, invalid UTF-8, a lone
// surrogate, a depth above MAX_DEPTH, anything after the value, or a
// number J5 refuses, such as 9007199254740992, 9007199254740992.0 or 1e16.
// Every other number is written as its nearest binary64 value is, so
// 1 followed by 800 zeros and e-800 is written 1.
func Transform(data []byte) ([]byte, error) {
	value, err := strictjson.Parse(data)
	if err != nil {
		return nil, err
	}
	return Encode(value)
}

// UnsafeInteger names one integer in a decoded document that a conforming
// producer must not emit, together with the dotted key path that locates it.
type UnsafeInteger struct {
	// Path is the dotted key path to the value, rooted at the label the
	// caller passed to [UnsafeIntegers]: object keys joined with ".", array
	// indices as "[i]", e.g. "claims.metadata.rows[0].id". The syntax
	// matches Truestamp.SafeIntegers on the server so the local and remote
	// rejections name the same location for the same value.
	Path string

	// Literal is the integer exactly as the user wrote it, never
	// round-tripped through a float. Rendering it anywhere, an error
	// message, a JSON field, MUST use this string rather than a numeric
	// type, or the report reproduces the very rounding it is warning about.
	Literal string
}

// UnsafeIntegers reports every integer in v whose magnitude exceeds
// [MaxSafeInteger], one entry per occurrence, so a user fixes them all in one
// pass instead of one 422 at a time. It returns nil when v is portable.
//
// root labels the top of each returned path; the Truestamp producer passes
// "claims" to match the server's field name.
//
// v is a document decoded with [encoding/json.Decoder.UseNumber], that is
// load-bearing, not incidental. Without it every number arrives as a float64
// with the offending literal already destroyed, and this walk would inspect
// the rounded value and report nothing. json.Number is therefore the only
// decoded form that can be judged; Go's own integer types are accepted too
// (nothing in the producer path builds claims that way today, but they marshal
// to exact integer literals, so leaving a hole there would be a latent
// regression). Floats in any form are deliberately never reported: the
// producer rule is about integer literals, and a large-magnitude value
// SPELLED as a float ("1e21", "1.5") is not an integer literal. That classification is a base-10 big.Int parse, which
// accepts only an optional sign followed by digits.
//
// Ordering is deterministic, Go map iteration is randomized, so object keys
// are walked in sorted order, arrays in index order. Two runs over the same
// document produce the same slice in the same order, which is what lets a
// caller print the list and a test assert on it.
func UnsafeIntegers(root string, v any) []UnsafeInteger {
	var found []UnsafeInteger
	walkUnsafeIntegers(root, v, &found)
	return found
}

// walkUnsafeIntegers is the recursive half of [UnsafeIntegers], appending to
// found so the arbitrary-depth walk needs no per-level allocation.
func walkUnsafeIntegers(path string, v any, found *[]UnsafeInteger) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			walkUnsafeIntegers(path+"."+k, t[k], found)
		}

	case []any:
		for i, elem := range t {
			walkUnsafeIntegers(path+"["+strconv.Itoa(i)+"]", elem, found)
		}

	case json.Number:
		lit := t.String()
		// 15 digits reach at most 10^15 - 1, comfortably inside 2^53 - 1
		// (16 digits), so the overwhelmingly common case never allocates a
		// big.Int and a short float is skipped for free.
		if len(strings.TrimPrefix(lit, "-")) <= 15 {
			return
		}
		// Base 10 accepts only an optional sign followed by digits, so it
		// doubles as the integer/float classifier: "1.5", "1e30" and
		// "9007199254740993e0" all fail here and are correctly skipped.
		val, ok := new(big.Int).SetString(lit, 10)
		if !ok {
			return
		}
		if new(big.Int).Abs(val).Cmp(maxSafeInteger) <= 0 {
			return
		}
		*found = append(*found, UnsafeInteger{Path: path, Literal: lit})

	case int:
		appendIfUnsafe(path, big.NewInt(int64(t)), found)
	case int64:
		appendIfUnsafe(path, big.NewInt(t), found)
	case uint64:
		appendIfUnsafe(path, new(big.Int).SetUint64(t), found)
	}
}

// appendIfUnsafe records n when it falls outside the safe range. The literal
// is rendered from the big.Int, which is exact for every fixed-width integer.
func appendIfUnsafe(path string, n *big.Int, found *[]UnsafeInteger) {
	if new(big.Int).Abs(n).Cmp(maxSafeInteger) <= 0 {
		return
	}
	*found = append(*found, UnsafeInteger{Path: path, Literal: n.String()})
}

// UnsafeIntegerMessage renders the submitter-facing explanation of one
// offending value.
//
// The wording is a deliberate byte-for-byte mirror of
// Truestamp.SafeIntegers.message/2 in truestamp-v2, which the server surfaces
// as an Ash InvalidAttribute on the claims field. A user who hits the local
// guard and a user who hits the server guard must read the same sentence, or
// the two halves of one rule look like two unrelated rules.
func UnsafeIntegerMessage(path, literal string) string {
	return fmt.Sprintf(
		"The integer %s at %s is outside the range %d to %d (+/- 2^53 - 1). "+
			"A value outside this range cannot be reproduced by a verifier that parses JSON "+
			"numbers as IEEE-754 doubles, which is most of them, so the resulting proof would "+
			"not be portably verifiable. Send the value as a string instead.",
		literal, path, -MaxSafeInteger, MaxSafeInteger)
}
