// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package strictjson reads a JSON text under the parsing rules J1 to J6 of
// SPEC.md section 10.1 (Parsing), steps READ (J1) and PARSE (J2 to J6) of
// section 10.2, each rule chosen so that no two verifiers read one file
// differently. It is the reader every later verification step reads the
// document through, and a keyring (section 10.2, step 0) is read through it
// too.
//
// It is named after the Truestamp application's Truestamp.StrictJSON.
//
// [Read] and [ReadFile] make J1, reading at most MAX_READ_OCTETS + 1 octets,
// which decides it; [CheckSize] makes J1 on bytes already in memory.
// [Parse] makes J2 to J6 with its own tokenizer, which builds the value
// directly: nothing is decoded by encoding/json, which keeps the last of two
// members with one name and replaces invalid UTF-8 and lone surrogates with
// U+FFFD. [Decode] makes J1 and then J2 to J6, in step order. A file the
// rules refuse is a [*Rejection], the bare rejection input_too_large or
// malformed_json with its sub_reason (SPEC.md sections 10.1 and 12.1).
//
// # Values
//
// A parsed value is within the value profile of SPEC.md section 10.1, and
// has one Go form per JSON value, so two files that differ only in
// formatting parse to equal values:
//
//   - an object is a map[string]any from its member names to their values,
//     and an array is a []any;
//   - a string is a Go string of Unicode scalar values, with every escape
//     resolved;
//   - true and false are bools, and null is nil;
//   - a number is a [Number], its nearest binary64 value (J5): 1, 1.0 and
//     1e0 are one Number, and -0, -0.0 and -1e-400 read as 0.
//
// # Constants
//
// MAX_READ_OCTETS, MAX_DEPTH and MAX_INTEGER come from internal/specdata,
// as do the reason codes and sub_reasons. The truestamp binary links
// internal/specdata through internal/jcs and this package, so the checks
// specdata makes when it is initialized run at the binary's start-up, as
// they do in every program that links this package: a file under
// specdata's embedded verification/spec/constructions/ that is not named by
// its digest, such as a stray Finder .DS_Store, stops such a program at
// start-up with a panic. specdata's tests and the sync's refusal of
// dotfiles catch it before a build does.
package strictjson

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"unicode/utf8"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// The limits of SPEC.md section 9 that the rules use.
const (
	// MaxReadOctets is MAX_READ_OCTETS, the J1 limit.
	MaxReadOctets = specdata.MaxReadOctets
	// MaxDepth is MAX_DEPTH, the J6 limit, counting the outermost value as 1.
	MaxDepth = specdata.MaxDepth
	// MaxInteger is MAX_INTEGER, 2^53 - 1: J5 lets no integral value through
	// whose magnitude is above it and below 10^21.
	MaxInteger = specdata.MaxInteger
)

// integerCeiling is 10^21: J5 refuses no integral value at or above it.
const integerCeiling = 1e21

// InIntegerGap reports whether f is in J5's integer gap: integral, of
// magnitude above MAX_INTEGER and below 10^21. J5 refuses every such value,
// and no other finite value.
func InIntegerGap(f float64) bool {
	magnitude := math.Abs(f)
	return f == math.Trunc(f) && magnitude > MaxInteger && magnitude < integerCeiling
}

// Number is a JSON number as J5 reads it: its nearest binary64 value, never
// -0, never infinite, and never an integer of magnitude above MAX_INTEGER
// and below 10^21. A Number the parser gives is always such a value; JCS
// (internal/jcs) refuses one that is not.
type Number float64

// Float64 returns n as a float64.
func (n Number) Float64() float64 { return float64(n) }

// Int64 returns n as an int64 and true when n is an integer of magnitude at
// most MAX_INTEGER (SPEC.md section 9), the bound of the pattern class INT
// (section 2.4); otherwise 0 and false.
func (n Number) Int64() (int64, bool) {
	f := float64(n)
	if f != math.Trunc(f) || math.Abs(f) > MaxInteger {
		return 0, false
	}
	return int64(f), true
}

// Rejection is a bare rejection of step READ or PARSE (SPEC.md section
// 10.2): input_too_large with no sub_reason, or malformed_json with the
// sub_reason naming a rule the text breaks. For a text that breaks more than
// one rule, the sub_reason is that of the first this reader meets, which
// SPEC.md section 10.1 leaves open.
type Rejection struct {
	// Code is specdata.ReasonInputTooLarge or specdata.ReasonMalformedJSON.
	Code specdata.Reason
	// SubReason is empty for input_too_large, and for malformed_json one of
	// syntax, invalid_utf8, lone_surrogate, duplicate_name, number and depth.
	SubReason specdata.SubReason
	// Offset is the octet at which the reader stopped. It is informational
	// and never compared (SPEC.md section 13.1).
	Offset int
}

// Error names the code, the sub_reason and the offset, and quotes nothing of
// the text.
func (r *Rejection) Error() string {
	if r.Code == specdata.ReasonInputTooLarge {
		return fmt.Sprintf("%s: the input is over %d octets", r.Code, MaxReadOctets)
	}
	return fmt.Sprintf("%s (%s) at octet %d", r.Code, r.SubReason, r.Offset)
}

// ErrUnreadable marks an input that could not be read at all, such as a
// missing file. It is not a verification result (SPEC.md section 12.2,
// exit code 1), so it is never a [*Rejection].
var ErrUnreadable = errors.New("strictjson: the input cannot be read")

// CheckSize is J1 on bytes in memory: nil when data is at most
// MAX_READ_OCTETS octets, and otherwise the rejection input_too_large.
func CheckSize(data []byte) error {
	if len(data) > MaxReadOctets {
		return &Rejection{Code: specdata.ReasonInputTooLarge, Offset: MaxReadOctets}
	}
	return nil
}

// Read is step READ over r: it reads at most MAX_READ_OCTETS + 1 octets,
// which decides J1, and returns them when there are at most MAX_READ_OCTETS.
// More is the rejection input_too_large; a failed read wraps ErrUnreadable.
func Read(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxReadOctets+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	if err := CheckSize(data); err != nil {
		return nil, err
	}
	return data, nil
}

// ReadFile is [Read] over the file at name, for the proof READ reads and
// for every keyring a run loads. A file that cannot be opened or read wraps
// ErrUnreadable.
func ReadFile(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	defer func() { _ = f.Close() }()
	return Read(f)
}

// Decode is steps READ and PARSE in order: J1, then J2 to J6.
func Decode(data []byte) (any, error) {
	if err := CheckSize(data); err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse is step PARSE, J2 to J6: the value data holds, or a [*Rejection]
// whose code is malformed_json. It does not make J1; [Decode] makes both.
func Parse(data []byte) (any, error) {
	// J3: the bytes are valid UTF-8. utf8.Valid also refuses overlong forms,
	// the encoded surrogates U+D800 to U+DFFF, which are not scalar values,
	// and anything above U+10FFFF.
	if !utf8.Valid(data) {
		return nil, malformed(specdata.SubReasonInvalidUTF8, firstInvalidUTF8(data))
	}
	p := &parser{data: data}
	p.skipWhitespace()
	value, err := p.parseValue(1)
	if err != nil {
		return nil, err
	}
	// J2: nothing after the value but whitespace. A byte order mark is not
	// whitespace, so a text that begins with one fails at its first octet.
	p.skipWhitespace()
	if p.pos != len(data) {
		return nil, p.fail(specdata.SubReasonSyntax)
	}
	return value, nil
}

// firstInvalidUTF8 returns the offset of the first octet of data that does
// not begin a valid UTF-8 sequence.
func firstInvalidUTF8(data []byte) int {
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size <= 1 {
			return i
		}
		i += size
	}
	return len(data)
}

func malformed(sub specdata.SubReason, offset int) *Rejection {
	return &Rejection{Code: specdata.ReasonMalformedJSON, SubReason: sub, Offset: offset}
}

// parser reads one text. The text is valid UTF-8 before it starts.
type parser struct {
	data []byte
	pos  int
}

func (p *parser) fail(sub specdata.SubReason) *Rejection {
	return malformed(sub, p.pos)
}

// skipWhitespace skips RFC 8259's whitespace: space, tab, line feed and
// carriage return, and nothing else.
func (p *parser) skipWhitespace() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

// hasPrefix reports whether the text at the reader's position begins with
// literal.
func (p *parser) hasPrefix(literal string) bool {
	return len(p.data)-p.pos >= len(literal) && string(p.data[p.pos:p.pos+len(literal)]) == literal
}

// parseValue reads the value at the reader's position. depth is the depth
// of that value (J6): the outermost value has depth 1, and only an object or
// an array can exceed MAX_DEPTH, since no other value adds a level.
func (p *parser) parseValue(depth int) (any, error) {
	if p.pos >= len(p.data) {
		return nil, p.fail(specdata.SubReasonSyntax)
	}
	switch c := p.data[p.pos]; {
	case (c == '{' || c == '[') && depth > MaxDepth:
		return nil, p.fail(specdata.SubReasonDepth)
	case c == '{':
		p.pos++
		return p.parseObject(depth)
	case c == '[':
		p.pos++
		return p.parseArray(depth)
	case c == '"':
		p.pos++
		return p.parseString()
	case c == '-' || ('0' <= c && c <= '9'):
		return p.parseNumber()
	case p.hasPrefix("true"):
		p.pos += len("true")
		return true, nil
	case p.hasPrefix("false"):
		p.pos += len("false")
		return false, nil
	case p.hasPrefix("null"):
		p.pos += len("null")
		return nil, nil
	default:
		return nil, p.fail(specdata.SubReasonSyntax)
	}
}

// parseObject reads an object after its '{'. J4: no two members with the
// same name, compared as code points after unescaping. Names are Go strings
// with every escape resolved, so string equality is that comparison, with
// no normalization.
func (p *parser) parseObject(depth int) (any, error) {
	members := map[string]any{}
	p.skipWhitespace()
	if p.pos < len(p.data) && p.data[p.pos] == '}' {
		p.pos++
		return members, nil
	}
	for {
		if p.pos >= len(p.data) || p.data[p.pos] != '"' {
			return nil, p.fail(specdata.SubReasonSyntax)
		}
		nameAt := p.pos
		p.pos++
		name, err := p.parseString()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return nil, p.fail(specdata.SubReasonSyntax)
		}
		p.pos++
		p.skipWhitespace()
		value, err := p.parseValue(depth + 1)
		if err != nil {
			return nil, err
		}
		if _, seen := members[name]; seen {
			return nil, malformed(specdata.SubReasonDuplicateName, nameAt)
		}
		members[name] = value

		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return nil, p.fail(specdata.SubReasonSyntax)
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
			p.skipWhitespace()
		case '}':
			p.pos++
			return members, nil
		default:
			return nil, p.fail(specdata.SubReasonSyntax)
		}
	}
}

// parseArray reads an array after its '['.
func (p *parser) parseArray(depth int) (any, error) {
	elements := []any{}
	p.skipWhitespace()
	if p.pos < len(p.data) && p.data[p.pos] == ']' {
		p.pos++
		return elements, nil
	}
	for {
		value, err := p.parseValue(depth + 1)
		if err != nil {
			return nil, err
		}
		elements = append(elements, value)

		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return nil, p.fail(specdata.SubReasonSyntax)
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
			p.skipWhitespace()
		case ']':
			p.pos++
			return elements, nil
		default:
			return nil, p.fail(specdata.SubReasonSyntax)
		}
	}
}

// shortEscapes are the two-character escapes of RFC 8259 section 7, by the
// character after the backslash.
var shortEscapes = [256]byte{
	'"': '"', '\\': '\\', '/': '/', 'b': 0x08, 'f': 0x0c, 'n': '\n', 'r': '\r', 't': '\t',
}

// parseString reads a string after its opening '"'. The text is valid UTF-8
// already, so every octet at or above 0x80 is part of a well-formed sequence
// and is copied as it is.
func (p *parser) parseString() (string, error) {
	var out []byte
	for {
		start := p.pos
		for p.pos < len(p.data) {
			c := p.data[p.pos]
			if c == '"' || c == '\\' || c < 0x20 {
				break
			}
			p.pos++
		}
		out = append(out, p.data[start:p.pos]...)
		if p.pos >= len(p.data) || p.data[p.pos] < 0x20 {
			return "", p.fail(specdata.SubReasonSyntax)
		}
		if p.data[p.pos] == '"' {
			p.pos++
			return string(out), nil
		}

		// A backslash.
		p.pos++
		if p.pos >= len(p.data) {
			return "", p.fail(specdata.SubReasonSyntax)
		}
		c := p.data[p.pos]
		if c == 'u' {
			p.pos++
			r, err := p.parseUnicodeEscape()
			if err != nil {
				return "", err
			}
			out = utf8.AppendRune(out, r)
			continue
		}
		if shortEscapes[c] == 0 {
			return "", p.fail(specdata.SubReasonSyntax)
		}
		out = append(out, shortEscapes[c])
		p.pos++
	}
}

// parseUnicodeEscape reads the four hex digits after "\u" and, for a high
// surrogate, the escaped low surrogate after it. J3: an escaped high
// surrogate is valid only as the first half of an escaped pair, and an
// escaped low surrogate only as the second.
func (p *parser) parseUnicodeEscape() (rune, error) {
	unit, err := p.hex4()
	if err != nil {
		return 0, err
	}
	switch {
	case 0xdc00 <= unit && unit <= 0xdfff:
		return 0, p.fail(specdata.SubReasonLoneSurrogate)
	case 0xd800 <= unit && unit <= 0xdbff:
		if !p.hasPrefix(`\u`) {
			return 0, p.fail(specdata.SubReasonLoneSurrogate)
		}
		p.pos += 2
		low, err := p.hex4()
		if err != nil {
			return 0, err
		}
		if low < 0xdc00 || low > 0xdfff {
			return 0, p.fail(specdata.SubReasonLoneSurrogate)
		}
		return 0x10000 + (unit-0xd800)<<10 + (low - 0xdc00), nil
	default:
		return unit, nil
	}
}

// hex4 reads four hex digits, in either case.
func (p *parser) hex4() (rune, error) {
	if len(p.data)-p.pos < 4 {
		return 0, p.fail(specdata.SubReasonSyntax)
	}
	var unit rune
	for _, c := range p.data[p.pos : p.pos+4] {
		var digit byte
		switch {
		case '0' <= c && c <= '9':
			digit = c - '0'
		case 'a' <= c && c <= 'f':
			digit = c - 'a' + 10
		case 'A' <= c && c <= 'F':
			digit = c - 'A' + 10
		default:
			return 0, p.fail(specdata.SubReasonSyntax)
		}
		unit = unit<<4 | rune(digit)
	}
	p.pos += 4
	return unit, nil
}

// parseNumber matches the text at the reader's position against RFC 8259's
// number grammar, then reads it as its nearest binary64 value (J5) by
// nearest, which rounds to nearest with ties to even, reads an underflow as
// zero and refuses an overflow, however long the spelling.
func (p *parser) parseNumber() (any, error) {
	start := p.pos
	if p.data[p.pos] == '-' {
		p.pos++
	}
	// `0`, or a digit from 1 to 9 followed by any digits.
	switch {
	case p.pos < len(p.data) && p.data[p.pos] == '0':
		p.pos++
	case p.pos < len(p.data) && '1' <= p.data[p.pos] && p.data[p.pos] <= '9':
		p.pos = runEnd(p.data, p.pos)
	default:
		return nil, p.fail(specdata.SubReasonSyntax)
	}
	// Nothing, or `.` followed by at least one digit.
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		if err := p.atLeastOneDigit(); err != nil {
			return nil, err
		}
	}
	// Nothing, or `e` or `E`, an optional sign and at least one digit.
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		if err := p.atLeastOneDigit(); err != nil {
			return nil, err
		}
	}

	value, ok := nearest(p.data[start:p.pos])
	if !ok {
		return nil, malformed(specdata.SubReasonNumber, start)
	}
	return profileNumber(value, start)
}

func (p *parser) atLeastOneDigit() error {
	end := runEnd(p.data, p.pos)
	if end == p.pos {
		return p.fail(specdata.SubReasonSyntax)
	}
	p.pos = end
	return nil
}

// profileNumber keeps a binary64 value within the value profile: -0 reads as
// 0, and an integral value of magnitude above MAX_INTEGER and below 10^21 is
// refused (J5). Every binary64 value of magnitude 2^53 or more is integral,
// so that refuses every magnitude from 2^53 up to, but not including, 10^21.
func profileNumber(value float64, at int) (any, error) {
	if value == 0 {
		// Both zeros compare equal; this keeps +0 only.
		return Number(0), nil
	}
	if InIntegerGap(value) {
		return nil, malformed(specdata.SubReasonNumber, at)
	}
	return Number(value), nil
}
