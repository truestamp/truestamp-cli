// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// A fuzz target for the reader of SPEC.md section 10.1, with JCS (SPEC.md
// section 2.2) as its round trip.

package strictjson_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// FuzzParse drives the reader with arbitrary bytes and holds four
// invariants:
//
//   - a refusal is malformed_json with one of SPEC.md section 10.1's
//     sub_reasons, and a syntax refusal is a text RFC 8259 refuses too
//     (encoding/json's json.Valid), since the two grammars are one;
//   - an accepted text is one RFC 8259 accepts;
//   - JCS of the value succeeds, and parses back to a value with the same
//     JCS (SPEC.md section 10.1, "After the signature");
//   - the canonical form of the value parses back to that value too.
//
// Mutants killed (by the seeds alone, under go test): skipWhitespace taking
// a form feed as whitespace, which accepts the seed of a form feed and {},
// and hex4's lowercase range widened to 'g', which accepts the seed "\u00g0".
func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		`{}`, `[]`, `null`, `true`, `0`, `-0`, `1e21`, `1e-400`, `9007199254740991`,
		`9007199254740992`, `1.7976931348623158e308`, `1e400`, `"\ud83d\ude00"`, `"\ud800"`,
		`{"a":1,"a":2}`, `{"a":1,"\u0061":2}`, `{"a":1 "b":2}`, "\xef\xbb\xbf{}", `{} {}`, "\f{}", `"\u00g0"`,
		"{\"\xff\":1}", `[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[1]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]`,
		`[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[1]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]`,
		`{"\u00e9":"\u20ac","e\u0301":[1.5,-2e-7,true,null,"\u0000\u001f\"\\"]}`,
		`{"z":0.000001,"y":1e-7,"x":1.23e21,"w":5e-324,"\ud83d\ude00":1,"\uff5a":2}`,
	} {
		f.Add([]byte(seed))
	}
	subReasons := specdata.SubReasons(specdata.ReasonMalformedJSON)

	f.Fuzz(func(t *testing.T, data []byte) {
		value, err := strictjson.Parse(data)
		if err != nil {
			var rejection *strictjson.Rejection
			if !errors.As(err, &rejection) || rejection.Code != specdata.ReasonMalformedJSON ||
				!slices.Contains(subReasons, rejection.SubReason) {
				t.Fatalf("Parse(%q) error = %v, want malformed_json with a sub_reason", data, err)
			}
			if rejection.SubReason == specdata.SubReasonSyntax && json.Valid(data) {
				t.Fatalf("Parse(%q) is a syntax error, but RFC 8259 accepts the text", data)
			}
			return
		}
		if !json.Valid(data) {
			t.Fatalf("Parse(%q) accepted a text RFC 8259 refuses", data)
		}

		m, err := jcs.Encode(value)
		if err != nil {
			t.Fatalf("JCS of the value of %q: %v", data, err)
		}
		again, err := strictjson.Parse(m)
		if err != nil {
			t.Fatalf("JCS %q does not parse: %v", m, err)
		}
		if m2, err := jcs.Encode(again); err != nil || !bytes.Equal(m, m2) {
			t.Fatalf("JCS is not stable: %q then %q (%v)", m, m2, err)
		}

		form, err := jcs.CanonicalForm(value)
		if err != nil {
			t.Fatalf("canonical form of %q: %v", data, err)
		}
		fromForm, err := strictjson.Parse(form)
		if err != nil {
			t.Fatalf("canonical form %q does not parse: %v", form, err)
		}
		if m3, err := jcs.Encode(fromForm); err != nil || !bytes.Equal(m, m3) {
			t.Fatalf("the canonical form reads as another value: %q then %q (%v)", m, m3, err)
		}
	})
}
