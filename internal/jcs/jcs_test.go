// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
