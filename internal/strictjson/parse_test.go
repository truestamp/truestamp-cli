// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Tests of step PARSE, J2 to J6 of SPEC.md section 10.1, rule by rule, as the
// reference verifier's elixir/test/verifier/parser_test.exs tests them.

package strictjson

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// refuses fails the test unless Parse refuses text with sub.
func refuses(t *testing.T, text string, sub specdata.SubReason) {
	t.Helper()
	v, err := Parse([]byte(text))
	if err == nil {
		t.Fatalf("Parse(%q) = %#v, want malformed_json (%s)", abbreviate(text), v, sub)
	}
	requireRejection(t, err, specdata.ReasonMalformedJSON, sub)
}

// accepts fails the test unless Parse reads text as want.
func accepts(t *testing.T, text string, want any) {
	t.Helper()
	got, err := Parse([]byte(text))
	if err != nil {
		t.Fatalf("Parse(%q) error = %v, want %#v", abbreviate(text), err, want)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse(%q) = %#v, want %#v", abbreviate(text), got, want)
	}
}

// abbreviate keeps a long test input short in a failure message.
func abbreviate(text string) string {
	if len(text) > 80 {
		return text[:60] + "..." + text[len(text)-12:]
	}
	return text
}

// The values a text reads as: one Go form per JSON value. Mutant killed:
// parseArray returning a nil slice for [] (DeepEqual tells them apart), and
// the "\/" escape mapping to '\'.
func TestParse_Values(t *testing.T) {
	accepts(t, `null`, nil)
	accepts(t, `true`, true)
	accepts(t, `false`, false)
	accepts(t, `"a"`, "a")
	accepts(t, `[]`, []any{})
	accepts(t, `{}`, map[string]any{})
	accepts(t, " \t\r\n[ 1 , \"x\" , null ] \t\r\n", []any{Number(1), "x", nil})
	accepts(t, `{"a":{"b":[true,false]},"c":-2.5}`, map[string]any{
		"a": map[string]any{"b": []any{true, false}},
		"c": Number(-2.5),
	})
	accepts(t, `"\"\\\/\b\f\n\r\t"`, "\"\\/\b\f\n\r\t")
	accepts(t, `"\u0000\u001f\u0041\u00e9\u20AC\ud83d\ude00"`, "\x00\x1fA\u00e9\u20ac\U0001f600")
	accepts(t, "\"\u00e9\U0001f600\x7f\"", "\u00e9\U0001f600\x7f")
}

// Two spellings of one number read alike. Mutant killed: profileNumber
// keeping -0 (the -0 cases read as a value whose sign bit is set).
func TestParse_NumberSpellingsReadAlike(t *testing.T) {
	for _, text := range []string{"1", "1.0", "1e0", "1E0", "10e-1", "0.1e1", "1.00000000000000000000000000000000001e0"} {
		accepts(t, text, Number(1))
	}
	for _, text := range []string{"0", "-0", "-0.0", "-0e5", "-0E-5", "0e99999999999999999999", "1e-400", "-1e-400", "-2.4703282292062327e-324"} {
		v, err := Parse([]byte(text))
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", text, err)
		}
		n, ok := v.(Number)
		if !ok || n != 0 || math.Signbit(float64(n)) {
			t.Fatalf("Parse(%q) = %#v, want +0", text, v)
		}
	}
}

// J5 at MAX_INTEGER: 2^53 - 1 is accepted, and from 2^53 up to, not
// including, 10^21 every integral value is refused however it is spelled.
// Mutants killed: InIntegerGap comparing the magnitude with >= MaxInteger
// (refuses 2^53 - 1), and with <= integerCeiling (refuses 1e21).
func TestParse_J5_IntegerGap(t *testing.T) {
	accepts(t, "9007199254740991", Number(MaxInteger))
	accepts(t, "-9007199254740991", Number(-MaxInteger))
	accepts(t, "9007199254740991.0", Number(MaxInteger))
	accepts(t, "9.007199254740991e15", Number(MaxInteger))
	for _, text := range []string{
		"9007199254740992",
		"-9007199254740992",
		"9007199254740992.0",
		"9007199254740992e0",
		"9.007199254740992e15",
		"0.9007199254740992E16",
		"90071992547409920e-1",
		"9007199254740993",   // a tie, read as 2^53
		"9007199254740993e0", // SPEC.md section 10.1's example
		"9007199254740991.5", // rounds to 2^53
		"1e16",
		"1152921504606846976", // 2^60
		"123456789012345678901",
		"999999999999999868928", // the largest binary64 value below 10^21
		"-999999999999999868928",
		"9.99999999999999868928e20",
		"1" + strings.Repeat("0", 20) + ".0", // 10^20
	} {
		refuses(t, text, specdata.SubReasonNumber)
	}
	// 10^21 and beyond are accepted, and so is a spelling below 10^21 whose
	// nearest binary64 value is 10^21.
	accepts(t, "1e21", Number(1e21))
	accepts(t, "1000000000000000000000", Number(1e21))
	accepts(t, "999999999999999999999", Number(1e21))
	accepts(t, "-1e21", Number(-1e21))
	accepts(t, "1e22", Number(1e22))
	// A value with a fraction is not an integer, however large.
	accepts(t, "9007199254740990.5", Number(9007199254740990.5))
}

// J5's overflow: a magnitude at or above 2^1024 - 2^970 is refused, and
// one just below reads as the largest binary64 value. Mutant killed:
// roundBinary without its lsb > maxLSB refusal.
func TestParse_J5_Overflow(t *testing.T) {
	largest := Number(math.MaxFloat64)
	accepts(t, "1.7976931348623158e308", largest)
	accepts(t, "-1.7976931348623158e308", -largest)
	accepts(t, "1.7976931348623157e308", largest)
	accepts(t, dyadicMinusOne(overflowBoundary()), largest)
	for _, text := range []string{
		"1.7976931348623159e308",
		"-1.7976931348623159e308",
		"1e400",
		"-1e400",
		"1e309",
		"1e99999999999999999999",
		overflowBoundary(),
		overflowBoundary() + ".0",
	} {
		refuses(t, text, specdata.SubReasonNumber)
	}
}

// J2: one JSON text and nothing after it but whitespace; no byte order
// mark; a \u escape of four hex digits only. Mutants killed: Parse without
// its trailing-content check, and hex4's ranges widened at any edge (the
// lowercase range to 'g' or from '`', the uppercase from '@', the digits to
// ':' or from '/').
func TestParse_J2_Syntax(t *testing.T) {
	for _, text := range []string{
		"", " ", "\xef\xbb\xbf{}", "\xef\xbb\xbf 1", "{}\xef\xbb\xbf", "{} {}", "{}x", "[]]", "1 2", "{}\x00",
		"\f{}", "{}\f", "\u00a0{}", "{}\u00a0", "{}\v",
		"01", "-", "-a", "+1", ".5", "1.", "1.e5", "1e", "1e+", "1E-", "-01", "0x10", "1_000",
		"NaN", "Infinity", "-Infinity", "tru", "nul", "fals", "True", "NULL", "truex",
		"[1,]", "[,1]", "[1 2]", "{,}", `{"a"}`, `{"a":}`, `{"a" 1}`, `{"a":1,}`, `{a:1}`, `{'a':1}`,
		`{"a":1 "b":2}`, `[`, `{`, `{"a":1`, `["a"`, `"abc`, `"\`, `"\x"`, `"\u12G4"`, `"\u12g4"`, `"\u00g0"`, `"\ug000"`, "\"\\u00`0\"", `"\u00@0"`, `"\u00:0"`, `"\u00/0"`, `"\u12"`,
		`"\U0041"`, `"\'"`, "\"a\x01b\"", "\"tab\there\"", "\"line\nfeed\"", "\"\x1f\"",
		"// c\n{}", "{} // c", "/* c */{}",
	} {
		refuses(t, text, specdata.SubReasonSyntax)
	}
	accepts(t, " \t\n\r{} \t\n\r", map[string]any{})
}

// J3: every class of invalid UTF-8, inside a string, in a name and between
// tokens, is invalid_utf8, and the scalar values at the edges are read.
// Mutant killed: Parse without its utf8.Valid check (a stray octet in a
// string would be copied, and one between tokens would be syntax).
func TestParse_J3_InvalidUTF8(t *testing.T) {
	classes := map[string]string{
		"a lone continuation octet":       "\x80",
		"a lone last continuation octet":  "\xbf",
		"a two-octet lead with no tail":   "\xc3",
		"a three-octet lead with one":     "\xe2\x82",
		"a four-octet lead with two":      "\xf0\x9f\x98",
		"an overlong two-octet form C0":   "\xc0\x80",
		"an overlong two-octet form C1":   "\xc1\xbf",
		"an overlong three-octet form":    "\xe0\x80\x80",
		"an overlong three-octet maximum": "\xe0\x9f\xbf",
		"an overlong four-octet form":     "\xf0\x80\x80\x80",
		"an overlong four-octet maximum":  "\xf0\x8f\xbf\xbf",
		"an encoded high surrogate":       "\xed\xa0\x80",
		"an encoded low surrogate":        "\xed\xbf\xbf",
		"an encoded surrogate pair":       "\xed\xa0\xbd\xed\xb8\x80",
		"above U+10FFFF":                  "\xf4\x90\x80\x80",
		"a lead of F5":                    "\xf5\x80\x80\x80",
		"a lead of F8":                    "\xf8\x88\x80\x80\x80",
		"the octet FE":                    "\xfe",
		"the octet FF":                    "\xff",
		"a lead before an ASCII octet":    "\xc3A",
	}
	for about, octets := range classes {
		for _, text := range []string{
			`"` + octets + `"`,
			`{"` + octets + `":1}`,
			`[1,` + octets + `]`,
			`{"a":"b"}` + octets,
		} {
			v, err := Parse([]byte(text))
			if err == nil {
				t.Fatalf("%s: Parse(%q) = %#v, want invalid_utf8", about, text, v)
			}
			requireRejection(t, err, specdata.ReasonMalformedJSON, specdata.SubReasonInvalidUTF8)
		}
	}
	// The edges of the scalar values: U+D7FF, U+E000, U+FFFF and U+10FFFF.
	accepts(t, "\"\xed\x9f\xbf\xee\x80\x80\xef\xbf\xbf\xf4\x8f\xbf\xbf\"", "\ud7ff\ue000\uffff\U0010ffff")
	// The offset is the first invalid octet.
	_, err := Parse([]byte("[\"ok\",\"\xff\"]"))
	if r := requireRejection(t, err, specdata.ReasonMalformedJSON, specdata.SubReasonInvalidUTF8); r.Offset != 7 {
		t.Fatalf("offset = %d, want 7", r.Offset)
	}
}

// J3: an escaped surrogate is valid only as the matching half of an escaped
// pair, in both orders and in names. Mutant killed: parseUnicodeEscape
// without its check that the second unit is a low surrogate (a high then a
// high would be combined).
func TestParse_J3_LoneSurrogates(t *testing.T) {
	for _, text := range []string{
		`"\ud800"`, `"\uDBFF"`, `"\udc00"`, `"\uDFFF"`,
		`"\udc00\ud800"`, `"\ude00\ud83d"`,
		`"\ud800\ud800"`, `"\ud83d\uDBFF"`,
		`"\ud800x"`, `"\ud800\n"`, `"\ud800\u0041"`, `"\ud800\\"`,
		`"a\ud800"`, `"\ud83d\ude00\ude00"`,
		`{"\ud800":1}`, `{"a":"\udc00"}`, `["\ud800"]`,
	} {
		refuses(t, text, specdata.SubReasonLoneSurrogate)
	}
	accepts(t, `"\ud800\udc00"`, "\U00010000")
	accepts(t, `"\uDBFF\uDFFF"`, "\U0010ffff")
	accepts(t, `{"\ud83d\ude00":"\uD83D\uDE00"}`, map[string]any{"\U0001f600": "\U0001f600"})
	// A high surrogate whose partner's escape is cut short is a syntax error.
	refuses(t, `"\ud800\udc0"`, specdata.SubReasonSyntax)
}

// J4: no two members with one name, compared as code points after
// unescaping, with no normalization. Mutant killed: parseObject without its
// duplicate check (the last value would win).
func TestParse_J4_DuplicateNames(t *testing.T) {
	for _, text := range []string{
		`{"a":1,"a":2}`,
		`{"a":1,"a":1}`,
		`{"a":1,"b":2,"a":3}`,
		`{"a":1,"\u0061":2}`,
		`{"\u0061":1,"a":2}`,
		`{"\u00e9":1,"` + "\u00e9" + `":2}`,
		`{"\u00E9":1,"\u00e9":2}`,
		`{"\ud83d\ude00":1,"` + "\U0001f600" + `":2}`,
		`{"\/":1,"/":2}`,
		`{"\n":1,"\u000a":2}`,
		`{"":1,"":2}`,
		`{"proof":{},"signature":{},"proof":{}}`,
		`[{"x":[{"a":1,"a":2}]}]`,
	} {
		refuses(t, text, specdata.SubReasonDuplicateName)
	}
	// Names equal only under Unicode normalization are distinct.
	accepts(t, "{\"\u00e9\":1,\"e\u0301\":2}", map[string]any{"\u00e9": Number(1), "e\u0301": Number(2)})
	// One name in two objects is not a duplicate.
	accepts(t, `[{"a":1},{"a":2}]`, []any{map[string]any{"a": Number(1)}, map[string]any{"a": Number(2)}})
	accepts(t, `{"a":{"a":1}}`, map[string]any{"a": map[string]any{"a": Number(1)}})
}

// nest builds a text of levels nested values around inner: arrays, or,
// when mixed, arrays and objects in turn from the outside in.
func nest(levels int, mixed bool, inner string) string {
	var b strings.Builder
	for i := range levels {
		if mixed && i%2 == 1 {
			b.WriteString(`{"k":`)
		} else {
			b.WriteString("[")
		}
	}
	return b.String() + inner + reverseClose(levels, mixed)
}

// reverseClose closes what nest opens, innermost first.
func reverseClose(levels int, mixed bool) string {
	var b strings.Builder
	for i := levels - 1; i >= 0; i-- {
		if mixed && i%2 == 1 {
			b.WriteString("}")
		} else {
			b.WriteString("]")
		}
	}
	return b.String()
}

// J6: the outermost value has depth 1, each object or array is one deeper
// than the object or array that holds it, and no other value adds a level.
// Mutant killed: parseValue checking depth > MaxDepth+1, and parseArray
// passing depth (not depth + 1) to its elements (arrays would not count).
func TestParse_J6_Depth(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		for _, inner := range []string{"1", `"s"`, "null", "true"} {
			if _, err := Parse([]byte(nest(MaxDepth, mixed, inner))); err != nil {
				t.Fatalf("depth 32 (mixed %v, inner %q): %v", mixed, inner, err)
			}
		}
		// A 33rd level, empty or not, array or object, is refused.
		for _, inner := range []string{"[]", "{}", "[1]", `{"a":1}`} {
			refuses(t, nest(MaxDepth, mixed, inner), specdata.SubReasonDepth)
		}
		refuses(t, nest(MaxDepth+1, mixed, "1"), specdata.SubReasonDepth)
		refuses(t, nest(1000, mixed, "1"), specdata.SubReasonDepth)
	}
	// Objects alone count as arrays do.
	objects := strings.Repeat(`{"a":`, MaxDepth) + "1" + strings.Repeat("}", MaxDepth)
	if _, err := Parse([]byte(objects)); err != nil {
		t.Fatalf("32 objects: %v", err)
	}
	refuses(t, strings.Repeat(`{"a":`, MaxDepth+1)+"1"+strings.Repeat("}", MaxDepth+1), specdata.SubReasonDepth)
	// Depth counts per branch, not across siblings.
	branch := nest(MaxDepth-1, false, "1")
	if _, err := Parse([]byte("[" + branch + "," + branch + "," + branch + "]")); err != nil {
		t.Fatalf("three siblings of depth 32: %v", err)
	}
	// A text that is too deep and never closed is a depth error, read no
	// further than its 33rd level.
	refuses(t, strings.Repeat("[", MaxDepth+1), specdata.SubReasonDepth)
	refuses(t, strings.Repeat("[", MaxDepth), specdata.SubReasonSyntax)
}

// Number.Int64 is an integer exactly when the reference reads one. Mutant
// killed: Int64 accepting a fraction.
func TestNumber_Int64(t *testing.T) {
	for _, tc := range []struct {
		n    Number
		want int64
		ok   bool
	}{
		{0, 0, true}, {1, 1, true}, {-7, -7, true},
		{MaxInteger, MaxInteger, true}, {-MaxInteger, -MaxInteger, true},
		{0.5, 0, false}, {1e21, 0, false}, {-1e21, 0, false}, {5e-324, 0, false},
	} {
		got, ok := tc.n.Int64()
		if got != tc.want || ok != tc.ok {
			t.Fatalf("Number(%v).Int64() = %d, %v; want %d, %v", float64(tc.n), got, ok, tc.want, tc.ok)
		}
	}
	if Number(2.5).Float64() != 2.5 {
		t.Fatal("Float64")
	}
}

// A rejection's message names the code, the sub_reason and the offset,
// which for a repeated name is that of the second name, and quotes nothing
// of the text. Mutant killed: parseObject giving a repeated name's
// rejection the offset of the reader after its value.
func TestRejection_Message(t *testing.T) {
	_, err := Parse([]byte(`{"secret-name":1,"secret-name":2}`))
	r := requireRejection(t, err, specdata.ReasonMalformedJSON, specdata.SubReasonDuplicateName)
	if r.Error() != "malformed_json (duplicate_name) at octet 17" {
		t.Fatalf("message = %q", r.Error())
	}
}
