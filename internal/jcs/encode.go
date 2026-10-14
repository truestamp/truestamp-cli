// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.JCS
// (elixir/lib/truestamp/verifier/jcs.ex in truestamp-verification): JCS(x),
// the JSON Canonicalization Scheme of RFC 8785 (SPEC.md sections 2.2 and
// 2.5), of a value the strict reader (internal/strictjson, SPEC.md section
// 10.1) gives. JCS is injective over the value profile of section 10.1, and
// for every value v in it JCS(v) parses back to v, so JCS of the parsed
// proof is the M its signature covers (section 6.1).
//
// Adapted from the reference: a number is written by gowebpki's
// NumberToJSON, ECMAScript's Number::toString for a binary64 value, in place
// of the reference's own formatting of OTP's shortest digits; and, since Go
// has one number type where Elixir has an integer and a float, the
// reference's refusal of an integer term above MAX_INTEGER becomes a
// refusal of any Number that is integral, of magnitude above MAX_INTEGER and
// below 10^21, which no parse gives.

package jcs

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"unicode/utf16"
	"unicode/utf8"

	gojcs "github.com/gowebpki/jcs"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// ErrOutsideProfile marks a value [Encode] or [CanonicalForm] refuses: one
// that is not a value the strict reader can give.
var ErrOutsideProfile = errors.New("jcs: the value is not within the value profile")

// Encode returns JCS(v), the RFC 8785 canonical form of v: no whitespace,
// object members sorted by the UTF-16 code units of their names, strings
// escaped as RFC 8785 section 3.2.2.2 says, and numbers written as
// ECMAScript's Number.prototype.toString writes them (section 3.2.2.3).
//
// v is a value as [strictjson.Parse] gives it: nil, a bool, a
// [strictjson.Number], a string, a []any or a map[string]any of such
// values. Encode refuses, with [ErrOutsideProfile], any other Go type, a
// string or member name that is not UTF-8, a Number that is not finite, and
// a Number that is integral, of magnitude above MAX_INTEGER and below 10^21.
// It checks nothing else of the profile: a Number of -0 is written "0", as
// RFC 8785 writes it, and the depth of a value is not checked. No parsed
// value is refused, so a value that is comes from a defect in the caller.
func Encode(v any) ([]byte, error) {
	return appendValue(nil, v)
}

func appendValue(b []byte, v any) ([]byte, error) {
	switch t := v.(type) {
	case nil:
		return append(b, "null"...), nil
	case bool:
		if t {
			return append(b, "true"...), nil
		}
		return append(b, "false"...), nil
	case string:
		return appendString(b, t)
	case strictjson.Number:
		text, err := numberText(t)
		if err != nil {
			return nil, err
		}
		return append(b, text...), nil
	case []any:
		b = append(b, '[')
		for i, element := range t {
			if i > 0 {
				b = append(b, ',')
			}
			var err error
			if b, err = appendValue(b, element); err != nil {
				return nil, err
			}
		}
		return append(b, ']'), nil
	case map[string]any:
		names, err := sortedNames(t)
		if err != nil {
			return nil, err
		}
		b = append(b, '{')
		for i, name := range names {
			if i > 0 {
				b = append(b, ',')
			}
			if b, err = appendString(b, name); err != nil {
				return nil, err
			}
			b = append(b, ':')
			if b, err = appendValue(b, t[name]); err != nil {
				return nil, err
			}
		}
		return append(b, '}'), nil
	default:
		return nil, fmt.Errorf("%w: a %T", ErrOutsideProfile, v)
	}
}

// numberText writes n as RFC 8785 section 3.2.2.3 does, through gowebpki's
// NumberToJSON, which writes both zeros as "0".
func numberText(n strictjson.Number) (string, error) {
	f := float64(n)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "", fmt.Errorf("%w: a number that is not finite", ErrOutsideProfile)
	}
	if strictjson.InIntegerGap(f) {
		return "", fmt.Errorf("%w: an integer of magnitude above MAX_INTEGER and below 10^21", ErrOutsideProfile)
	}
	text, err := gojcs.NumberToJSON(f)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrOutsideProfile, err)
	}
	return text, nil
}

// sortedNames returns the member names of object in RFC 8785 section 3.2.3's
// order: by their UTF-16 code units, compared as unsigned integers, with a
// name that is a prefix of another first.
func sortedNames(object map[string]any) ([]string, error) {
	type keyed struct {
		name string
		key  []uint16
	}
	entries := make([]keyed, 0, len(object))
	for name := range object {
		if !utf8.ValidString(name) {
			return nil, fmt.Errorf("%w: a member name that is not UTF-8", ErrOutsideProfile)
		}
		entries = append(entries, keyed{name: name, key: utf16.Encode([]rune(name))})
	}
	slices.SortFunc(entries, func(a, b keyed) int { return slices.Compare(a.key, b.key) })
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.name
	}
	return names, nil
}

// appendString writes s as RFC 8785 section 3.2.2.2 does: '"' and '\' are
// escaped, the controls U+0000 to U+001F are escaped with the short forms
// where JSON has one and as \u00xx in lowercase hex otherwise, and every
// other character is written as it is. An octet below 0x80 is never part of
// a multi-octet UTF-8 sequence, so s is escaped octet by octet.
func appendString(b []byte, s string) ([]byte, error) {
	if !utf8.ValidString(s) {
		return nil, fmt.Errorf("%w: a string that is not UTF-8", ErrOutsideProfile)
	}
	const hexDigits = "0123456789abcdef"
	b = append(b, '"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"':
			b = append(b, '\\', '"')
		case '\\':
			b = append(b, '\\', '\\')
		case 0x08:
			b = append(b, '\\', 'b')
		case '\t':
			b = append(b, '\\', 't')
		case '\n':
			b = append(b, '\\', 'n')
		case 0x0c:
			b = append(b, '\\', 'f')
		case '\r':
			b = append(b, '\\', 'r')
		default:
			if c < 0x20 {
				b = append(b, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0x0f])
			} else {
				b = append(b, c)
			}
		}
	}
	return append(b, '"'), nil
}
