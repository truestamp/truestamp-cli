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
package jcs

import "github.com/truestamp/truestamp-cli/internal/strictjson"

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
