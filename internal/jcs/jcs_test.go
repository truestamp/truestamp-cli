// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	gojcs "github.com/gowebpki/jcs"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// digest0x11 is SHA-256(0x11 || JCS(text)), the digest each vector of
// TestTransform_Vectors gives.
func digest0x11(canonical []byte) string {
	sum := sha256.Sum256(append([]byte{0x11}, canonical...))
	return hex.EncodeToString(sum[:])
}

// Transform writes each of these texts as RFC 8785 does, and its 0x11
// digest is the one given: an explicit null, a control character and a
// tab, -0.0 written 0, exponential floats, non-ASCII written unescaped, an
// astral-plane name sorted on its UTF-16 code units, and MAX_INTEGER
// itself. Mutant killed: Transform writing the parsed value with
// encoding/json's Marshal (the astral-plane names sort by their UTF-8
// octets, and -0 is written -0).
func TestTransform_Vectors(t *testing.T) {
	vectors := []struct {
		name string
		in   string
		jcs  string
		want string
	}{
		{"explicit null", `{"a": null, "b": 1}`, `{"a":null,"b":1}`,
			"13ec6081eab8b87a582dc6c2d0f6ff561d79de8be10c652440b53e1b72a95099"},
		{"control character and tab", `{"c": "\u0001", "tab": "\t"}`, `{"c":"\u0001","tab":"\t"}`,
			"66d5ba7776a99a71bc63007419530297a231043bd74de64e9ef2b7b44f3deec2"},
		{"negative zero", `{"z": -0.0}`, `{"z":0}`,
			"901865b9e33220a2601150ae2f202ad45d172458c8eb5865ccfd695c353e2430"},
		{"exponential floats", `{"big": 1.0e21, "small": 1.0e-7}`, `{"big":1e+21,"small":1e-7}`,
			"2f01621ad310054550edf66468a6db72601e6e983464fb594cb884585beef1c7"},
		{"non-ASCII emitted unescaped", "{\"name\": \"\u00c1vila\", \"emoji\": \"\U0001f600\"}",
			"{\"emoji\":\"\U0001f600\",\"name\":\"\u00c1vila\"}",
			"65a18fbdd50f042c67de68433ede20023a529af601099cd735f4dd4805845eef"},
		{"astral-plane key ordering", "{\"\U0001f600\": 1, \"\uffff\": 2}", "{\"\U0001f600\":1,\"\uffff\":2}",
			"d0951cfce65f8af38e98d67bd15833bb89afa2163e1b4c707047c126d53a6710"},
	}
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			canonical, err := Transform([]byte(v.in))
			if err != nil {
				t.Fatalf("Transform: %v", err)
			}
			if string(canonical) != v.jcs {
				t.Errorf("canonical = %s, want %s", canonical, v.jcs)
			}
			if got := digest0x11(canonical); got != v.want {
				t.Errorf("0x11 digest = %s, want %s", got, v.want)
			}
		})
	}
	if got, err := Transform([]byte(`{"n":9007199254740991}`)); err != nil || string(got) != `{"n":9007199254740991}` {
		t.Errorf("MAX_INTEGER: %s, %v", got, err)
	}
}

// Transform refuses an integer of magnitude above MAX_INTEGER and below
// 10^21 as rule J5 does (SPEC.md section 10.1), however it is written:
// 9007199254740992, 9007199254740992.0 and 1e16, negative or not, at the
// top level or in an object, each malformed_json with the sub_reason
// number; 1e21, at the gap's top, 1.5, which is not integral, and
// -MAX_INTEGER are written. Mutants killed: Transform canonicalizing the bytes
// with gowebpki/jcs's Transform, which keeps 2^53 (each refusal is
// missing); J5's gap tested against 2^53 rather than MAX_INTEGER
// (9007199254740992 passes).
func TestTransform_RefusesAnIntegerPastMaxInteger(t *testing.T) {
	for _, in := range []string{
		`9007199254740992`, `9007199254740992.0`, `1e16`,
		`-9007199254740992`, `{"n":9007199254740992.0}`, `{"n":[1e16]}`, `{"n":9007199254740993}`,
	} {
		got, err := Transform([]byte(in))
		var rejection *strictjson.Rejection
		if !errors.As(err, &rejection) || rejection.Code != specdata.ReasonMalformedJSON || rejection.SubReason != specdata.SubReasonNumber || got != nil {
			t.Errorf("Transform(%s) = %s, %v; want malformed_json (number)", in, got, err)
		}
	}
	for in, want := range map[string]string{`1e21`: `1e+21`, `1.5`: `1.5`, `[-9007199254740991]`: `[-9007199254740991]`} {
		if got, err := Transform([]byte(in)); err != nil || string(got) != want {
			t.Errorf("Transform(%s) = %s, %v; want %s", in, got, err, want)
		}
	}
}

// A number is written as its nearest binary64 value is, whatever its
// spelling: 1 followed by 800 zeros and e-800, 806 octets, is 1, as is
// 1.000 and 100e-2. Mutant killed: the strict reader placing a number's
// point by its first 800 digits, as strconv.ParseFloat's decimal fallback
// does (the long spelling is written 0.1).
func TestTransform_ANumberIsItsNearestBinary64(t *testing.T) {
	long := "[1" + strings.Repeat("0", 800) + "e-800]"
	if len(long) != 808 {
		t.Fatalf("the spelling is %d octets", len(long))
	}
	for _, in := range []string{long, `[1.000]`, `[100e-2]`} {
		if got, err := Transform([]byte(in)); err != nil || string(got) != `[1]` {
			t.Errorf("Transform(%.20s...) = %s, %v; want [1]", in, got, err)
		}
	}
}

// gowebpki/jcs is not a validating parser and rewrites several malformed
// texts into different valid ones, which Transform refuses, each as the
// strict reader does; so two texts, one malformed, never give one digest.
// A repeated member name, which encoding/json keeps the last of, is refused
// too. Mutant killed: Transform canonicalizing the bytes with gowebpki/jcs's
// Transform (each rewrite is accepted).
func TestTransform_RefusesWhatTheLibraryRewrites(t *testing.T) {
	for _, tc := range []struct{ in, impostor string }{
		{`{"a":1 2}`, `{"a":12}`},
		{`{"a":+1}`, `{"a":1}`},
		{`{"a":0009}`, `{"a":9}`},
		{`{"a":01}`, `{"a":1}`},
		{`{"a":.5}`, `{"a":0.5}`},
		{`{"a":1.}`, `{"a":1}`},
		{`[1 2]`, `[12]`},
	} {
		got, libErr := gojcs.Transform([]byte(tc.in))
		if libErr != nil || string(got) != tc.impostor || json.Valid([]byte(tc.in)) {
			t.Fatalf("%s: the library gives %s, %v; this row no longer covers what it claims", tc.in, got, libErr)
		}
		if out, err := Transform([]byte(tc.in)); !isMalformed(err) || out != nil {
			t.Errorf("Transform(%s) = %s, %v; want malformed_json", tc.in, out, err)
		}
	}
	for _, in := range []string{`{"a":1,"a":2}`, ``, `   `, `{`, "\ufeff{}", `{"a":1}{"b":2}`, `{"huge":1e400}`} {
		if out, err := Transform([]byte(in)); !isMalformed(err) || out != nil {
			t.Errorf("Transform(%q) = %s, %v; want malformed_json", in, out, err)
		}
	}
}

// isMalformed reports whether err is the strict reader's malformed_json.
func isMalformed(err error) bool {
	var rejection *strictjson.Rejection
	return errors.As(err, &rejection) && rejection.Code == specdata.ReasonMalformedJSON
}

// Transform's output is canonical: transformed again it is unchanged, and
// {"b":2,"a":1} is written {"a":1,"b":2}. Mutant killed: Encode sorting
// object members in descending order (the first text is written
// {"b":2,"a":1}).
func TestTransform_IsIdempotent(t *testing.T) {
	for _, in := range []string{`{"b":2,"a":1}`, `{"z":-0.0,"big":1.0e21}`, `[9007199254740991,{"x":-9007199254740991}]`, `{"b":{"d":1,"c":2},"a":[3,1]}`} {
		once, err := Transform([]byte(in))
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		twice, err := Transform(once)
		if err != nil || string(once) != string(twice) {
			t.Errorf("%s: %s, then %s, %v", in, once, twice, err)
		}
		if in == `{"b":2,"a":1}` && string(once) != `{"a":1,"b":2}` {
			t.Errorf("%s: %s", in, once)
		}
	}
}

// decodeClaims parses a document the way cmd/create does: UseNumber, so every
// integer literal survives as text. A test that used json.Unmarshal here would
// be testing the bug rather than the fix.
func decodeClaims(t *testing.T, doc string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(doc))
	dec.UseNumber()
	var out map[string]any
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("decoding %s: %v", doc, err)
	}
	return out
}

// TestUnsafeIntegersBoundary walks the one bound, MAX_INTEGER, from both
// directions and in both signs: 2^53 is the first integer past it, and is
// refused as every integer past it is. Mutant killed: UnsafeIntegers
// passing a magnitude up to MAX_INTEGER + 1 (the 2^53 and -2^53 rows pass).
func TestUnsafeIntegersBoundary(t *testing.T) {
	for _, tc := range []struct {
		name   string
		lit    string
		reject bool
	}{
		{"2^53 - 1 is the largest producible integer", "9007199254740991", false},
		{"2^53 is the first integer past MAX_INTEGER", "9007199254740992", true},
		{"2^53 + 1 is rejected", "9007199254740993", true},
		{"-(2^53 - 1) is the smallest producible integer", "-9007199254740991", false},
		{"-2^53 is rejected", "-9007199254740992", true},
		{"unsigned 64-bit id is rejected", "18446744073709551615", true},
		{"small integer is fine", "42", false},
		{"zero is fine", "0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := UnsafeIntegers("claims", decodeClaims(t, `{"n":`+tc.lit+`}`))
			if tc.reject {
				if len(got) != 1 {
					t.Fatalf("UnsafeIntegers(%s) = %v, want exactly one violation", tc.lit, got)
				}
				if got[0].Path != "claims.n" {
					t.Errorf("path = %q, want %q", got[0].Path, "claims.n")
				}
				if got[0].Literal != tc.lit {
					t.Errorf("literal = %q, want %q, the reported value must be the user's own bytes", got[0].Literal, tc.lit)
				}
				return
			}
			if len(got) != 0 {
				t.Errorf("UnsafeIntegers(%s) = %v, want no violations", tc.lit, got)
			}
		})
	}
}

// TestUnsafeIntegersIgnoresFloats pins the classifier: UnsafeIntegers names
// integer literals only, so a value spelled as a float is never one of its
// violations, or every geolocation and every scientific measurement would be
// named as one. Under the one J5 bound an integral float above MAX_INTEGER,
// such as 9007199254740993.0, is still refused, by [Transform], which
// `truestamp items create` runs on the claims this walk passes. Mutant
// killed: the walk classifying through big.Float rather than a base-10
// big.Int parse (9007199254740993.0 and 9007199254740993e0 are named).
func TestUnsafeIntegersIgnoresFloats(t *testing.T) {
	doc := `{
	  "a": 1.5,
	  "b": 1e21,
	  "c": 1.0e-7,
	  "d": -0.0,
	  "e": 9007199254740993.0,
	  "f": 9007199254740993e0,
	  "g": 1.7976931348623157e308,
	  "lat": 37.7749,
	  "lon": -122.4194
	}`
	if got := UnsafeIntegers("claims", decodeClaims(t, doc)); len(got) != 0 {
		t.Errorf("UnsafeIntegers = %v, want none: floats are never a producer violation", got)
	}
}

// TestUnsafeIntegersIgnoresStrings confirms the documented escape hatch works:
// the fix a user is told to apply, send the value as a string, actually
// clears the guard.
func TestUnsafeIntegersIgnoresStrings(t *testing.T) {
	doc := `{"id":"18446744073709551615","nested":{"n":"9007199254740993"}}`
	if got := UnsafeIntegers("claims", decodeClaims(t, doc)); len(got) != 0 {
		t.Errorf("UnsafeIntegers = %v, want none: a stringified integer is the prescribed fix", got)
	}
}

// TestUnsafeIntegersNestedPaths exercises arbitrary-depth nesting through both
// maps and arrays, and pins the dotted-path syntax against the example the
// server's COMMS.md publishes: claims.metadata.rows[0].id.
func TestUnsafeIntegersNestedPaths(t *testing.T) {
	doc := `{
	  "metadata": {
	    "rows": [
	      {"id": 9007199254740993},
	      {"id": 42},
	      {"deep": {"deeper": [[18446744073709551615]]}}
	    ]
	  }
	}`
	got := UnsafeIntegers("claims", decodeClaims(t, doc))

	want := []UnsafeInteger{
		{Path: "claims.metadata.rows[0].id", Literal: "9007199254740993"},
		{Path: "claims.metadata.rows[2].deep.deeper[0][0]", Literal: "18446744073709551615"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("UnsafeIntegers = %#v,\nwant %#v", got, want)
	}
}

// TestUnsafeIntegersReportsEveryViolationDeterministically covers the two
// properties a user-facing list needs: completeness (fix them all in one pass,
// not one 422 at a time) and stable ordering despite Go's randomized map
// iteration.
func TestUnsafeIntegersReportsEveryViolationDeterministically(t *testing.T) {
	doc := `{
	  "zeta": 9007199254740993,
	  "alpha": 18446744073709551615,
	  "mid": {"beta": 9007199254740992, "ok": 7},
	  "arr": [1, 99999999999999999999, 3]
	}`
	claims := decodeClaims(t, doc)

	want := []UnsafeInteger{
		{Path: "claims.alpha", Literal: "18446744073709551615"},
		{Path: "claims.arr[1]", Literal: "99999999999999999999"},
		{Path: "claims.mid.beta", Literal: "9007199254740992"},
		{Path: "claims.zeta", Literal: "9007199254740993"},
	}

	// Repeat: one pass could agree with the expectation by luck of the map
	// iteration order. Many passes over a freshly decoded map cannot.
	for i := 0; i < 50; i++ {
		got := UnsafeIntegers("claims", decodeClaims(t, doc))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: UnsafeIntegers = %#v,\nwant %#v", i, got, want)
		}
	}

	if got := len(UnsafeIntegers("claims", claims)); got != 4 {
		t.Errorf("violation count = %d, want 4 (every offender, not just the first)", got)
	}
}

// TestUnsafeIntegersGoNativeIntegers covers the values Go code could put into
// a claims map directly. Nothing in the producer path does today, but they
// marshal to exact integer literals, so the walk stays total over anything
// encoding/json can emit as an integer.
func TestUnsafeIntegersGoNativeIntegers(t *testing.T) {
	claims := map[string]any{
		"i":    int(9007199254740993),
		"i64":  int64(-9007199254740992),
		"u64":  uint64(18446744073709551615),
		"okay": int64(MaxSafeInteger),
		"f":    float64(1e30),
	}
	got := UnsafeIntegers("claims", claims)
	want := []UnsafeInteger{
		{Path: "claims.i", Literal: "9007199254740993"},
		{Path: "claims.i64", Literal: "-9007199254740992"},
		{Path: "claims.u64", Literal: "18446744073709551615"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("UnsafeIntegers = %#v,\nwant %#v", got, want)
	}
}

// TestUnsafeIntegersEmptyAndPortable confirms the common case returns nil
// rather than an empty non-nil slice, so `if len(...) == 0` and `if ... == nil`
// agree at every call site.
func TestUnsafeIntegersEmptyAndPortable(t *testing.T) {
	for _, doc := range []string{`{}`, `{"name":"doc","n":42,"nested":{"a":[1,2,3]}}`} {
		if got := UnsafeIntegers("claims", decodeClaims(t, doc)); got != nil {
			t.Errorf("UnsafeIntegers(%s) = %v, want nil", doc, got)
		}
	}
}

// TestUnsafeIntegerMessage pins the four facts the message must carry, path,
// value, allowed range, and the stringify remedy, and the exact wording, which
// mirrors Truestamp.SafeIntegers.message/2 server-side so the local and remote
// rejections read as one system.
func TestUnsafeIntegerMessage(t *testing.T) {
	got := UnsafeIntegerMessage("claims.metadata.rows[0].id", "9007199254740993")

	want := "The integer 9007199254740993 at claims.metadata.rows[0].id is outside the range " +
		"-9007199254740991 to 9007199254740991 (+/- 2^53 - 1). " +
		"A value outside this range cannot be reproduced by a verifier that parses JSON " +
		"numbers as IEEE-754 doubles, which is most of them, so the resulting proof would " +
		"not be portably verifiable. Send the value as a string instead."

	if got != want {
		t.Errorf("UnsafeIntegerMessage =\n%q\nwant\n%q", got, want)
	}
}

// TestUnsafeIntegersPreservesArbitraryPrecision confirms the reported literal
// is the user's own bytes at any width, not a value that went through a
// fixed-width Go type on its way to the report.
func TestUnsafeIntegersPreservesArbitraryPrecision(t *testing.T) {
	huge := strings.Repeat("9", 400)
	got := UnsafeIntegers("claims", decodeClaims(t, `{"nonce":`+huge+`}`))
	if len(got) != 1 {
		t.Fatalf("UnsafeIntegers = %v, want one violation", got)
	}
	if got[0].Literal != huge {
		t.Errorf("literal has %d digits, want the original %d", len(got[0].Literal), len(huge))
	}
}
