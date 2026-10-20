// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the base64url pattern classes B64U32 and B64U64 of
// SPEC.md section 2.4, unpadded base64url (RFC 4648 section 5) of 32 and of
// 64 octets, in the one canonical spelling each value has.
//
// The class of the last character forces the unused bits to zero, so a
// string that matches its class is exactly a canonical encoding. Stock
// decoders are not canonical: Go's base64.RawURLEncoding.Strict() skips CR
// and LF. So a string is matched against its class, as \A...\z matches,
// before it is decoded, as section 2.4 requires. The patterns are
// internal/specdata's transcription of spec/encodings.json, compiled once,
// and the decoded octets come back as a [32]byte or a [64]byte.

package registry

import (
	"encoding/base64"
	"errors"
	"regexp"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// B64UClass is a base64url pattern class of SPEC.md section 2.4.
type B64UClass int

// The two base64url classes.
const (
	// B64U32 is the class of a 32-octet value: a public key.
	B64U32 B64UClass = iota + 1
	// B64U64 is the class of a 64-octet value: a signature.
	B64U64
)

// ErrNotCanonicalB64U marks a value that is not a string of its base64url
// class in full.
var ErrNotCanonicalB64U = errors.New("registry: not the canonical base64url of its class")

// b64uClass is a class's pattern, written bare, its compiled whole-string
// form, and the octets it decodes to.
type b64uClass struct {
	pattern string
	whole   *regexp.Regexp
	octets  int
}

var b64uClasses = map[B64UClass]b64uClass{
	B64U32: compileB64U("B64U32"),
	B64U64: compileB64U("B64U64"),
}

func compileB64U(name string) b64uClass {
	encoding, ok := specdata.Encodings()[name]
	if !ok || encoding.Pattern == "" || encoding.DecodesToOctets == 0 {
		panic("registry: internal/specdata has no base64url class " + name)
	}
	return b64uClass{
		pattern: encoding.Pattern,
		// \A and \z hold the match to the whole string; $ in PCRE also
		// matches before a final newline (section 2.4).
		whole:  regexp.MustCompile(`\A(?:` + encoding.Pattern + `)\z`),
		octets: encoding.DecodesToOctets,
	}
}

// B64UPattern returns class's pattern, written bare as in SPEC.md section
// 2.4, and false for a value that is not a class.
func B64UPattern(class B64UClass) (string, bool) {
	c, ok := b64uClasses[class]
	return c.pattern, ok
}

// B64UOctets returns the number of octets a string of class decodes to,
// and false for a value that is not a class.
func B64UOctets(class B64UClass) (int, bool) {
	c, ok := b64uClasses[class]
	return c.octets, ok
}

// MatchesB64U reports whether text matches class in full. A value that is
// not a class matches nothing.
func MatchesB64U(class B64UClass, text string) bool {
	c, ok := b64uClasses[class]
	return ok && c.whole.MatchString(text)
}

// DecodeB64U32 returns the 32 octets text spells when it matches B64U32,
// and otherwise [ErrNotCanonicalB64U].
func DecodeB64U32(text string) ([32]byte, error) {
	decoded, err := decodeB64U(B64U32, text)
	if err != nil {
		return [32]byte{}, err
	}
	// A string of B64U32 has exactly 43 characters and spells exactly 32
	// octets, so the length is fixed once it has matched: the conversion
	// panics only on a defect in the pattern or the decoder, as digest does
	// on one in this package.
	return [32]byte(decoded), nil
}

// DecodeB64U64 returns the 64 octets text spells when it matches B64U64,
// and otherwise [ErrNotCanonicalB64U].
func DecodeB64U64(text string) ([64]byte, error) {
	decoded, err := decodeB64U(B64U64, text)
	if err != nil {
		return [64]byte{}, err
	}
	// A string of B64U64 has exactly 86 characters and spells exactly 64
	// octets; as in DecodeB64U32, the conversion panics only on a defect.
	return [64]byte(decoded), nil
}

// decodeB64U decodes text, once it has matched class, to the class's
// octets.
func decodeB64U(class B64UClass, text string) ([]byte, error) {
	if !MatchesB64U(class, text) {
		return nil, ErrNotCanonicalB64U
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(text)
	if err != nil {
		return nil, ErrNotCanonicalB64U
	}
	return decoded, nil
}

// EncodeB64U returns the canonical spelling of octets: unpadded base64url.
func EncodeB64U(octets []byte) string {
	return base64.RawURLEncoding.EncodeToString(octets)
}
