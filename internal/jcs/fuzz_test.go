// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	gojcs "github.com/gowebpki/jcs"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// FuzzTransform drives Transform with arbitrary bytes, holding four
// properties: it accepts exactly the texts strictjson.Parse accepts; every
// text it accepts is valid JSON; its output is valid JSON that the strict
// reader accepts; and its output transformed again is unchanged.
func FuzzTransform(f *testing.F) {
	seeds := []string{
		`{"a": null, "b": 1}`,
		`{"c": "\u0001", "tab": "\t"}`,
		"{\"c\": \"\x01\", \"tab\": \"\\t\"}",
		`{"z": -0.0}`,
		`{"big": 1.0e21, "small": 1.0e-7}`,
		"{\"name\": \"\u00c1vila\", \"emoji\": \"\U0001f600\"}",
		"{\"\U0001f600\": 1, \"\uffff\": 2}",
		`{"n": 9007199254740991}`,
		`{"n": 9007199254740992}`,
		`{"n": 9007199254740992.0}`,
		`{"n": 1e16}`,
		`[1` + strings.Repeat("0", 800) + `e-800]`,
		`{"ledgerSequence":18446744073709551615}`,
		`{"nonce":` + strings.Repeat("9", 400) + `}`,
		`{"a":1 2,"b":9007199254740993}`,
		`{"a":+1}`,
		`{"a":0009}`,
		`{"a":.5}`,
		`{"a":1,"a":2}`,
		`{"huge":1e400}`,
		`{"a":1}{"b":2}`,
		`{`,
		``,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		canonical, err := Transform(data)
		_, parseErr := strictjson.Parse(data)
		if (err == nil) != (parseErr == nil) {
			t.Fatalf("Transform's error %v, the strict reader's %v", err, parseErr)
		}
		if err != nil {
			if canonical != nil {
				t.Fatalf("an error with a result %q", canonical)
			}
			return
		}
		if !json.Valid(data) || !json.Valid(canonical) {
			t.Fatalf("%q accepted as %q", data, canonical)
		}
		again, err := Transform(canonical)
		if err != nil || !bytes.Equal(canonical, again) {
			t.Fatalf("not idempotent: %q then %q, %v", canonical, again, err)
		}
	})
}

// FuzzUnsafeIntegers drives the producer-side walker over arbitrary claims
// documents, decoded exactly the way cmd/create decodes them. The walk is
// recursive over user-supplied structure, so this is the net for a panic on a
// shape nobody thought of, plus three invariants:
//
//  1. Every reported literal is a real base-10 integer outside the safe range.
//     A float reported here would fail a legitimate submission.
//  2. Every reported literal appears verbatim in the input bytes. The whole
//     point of the guard is naming the user's own value, so a literal that is
//     not a substring of what they wrote means a rounding crept back in.
//  3. The result is deterministic across repeated walks of freshly decoded
//     copies, despite Go's randomized map iteration.
func FuzzUnsafeIntegers(f *testing.F) {
	seeds := []string{
		`{"name":"doc","hash":"aa","hash_type":"sha256"}`,
		`{"n":9007199254740991}`,
		`{"n":9007199254740992}`,
		`{"n":-9007199254740992}`,
		`{"id":18446744073709551615}`,
		`{"metadata":{"rows":[{"id":9007199254740993},{"id":1}]}}`,
		`{"deep":[[[[{"x":123456789012345678901234567890}]]]]}`,
		`{"f":1.5,"e":1e21,"wide":9007199254740993.0,"exp":9007199254740993e0}`,
		`{"s":"18446744073709551615"}`,
		`{"nonce":` + strings.Repeat("9", 400) + `}`,
		`{"neg":-0}`,
		`{}`,
		`{"a":null,"b":true,"c":[],"d":{}}`,
		// Long keys, and an oversized value reachable only through one. Every
		// other seed uses short keys, so a walker that skipped long-keyed
		// members would satisfy the whole corpus; the completeness check below
		// can only bite on an input that actually has one.
		`{"a_considerably_longer_key_than_any_other_seed_uses":9007199254740993}`,
		`{"outer_container_key":{"inner_identifier_key":[{"deeply_nested_identifier":18446744073709551615}]}}`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	decode := func(data []byte) (map[string]any, bool) {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		var out map[string]any
		if err := dec.Decode(&out); err != nil {
			return nil, false
		}
		return out, true
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		claims, ok := decode(data)
		if !ok {
			return
		}

		got := UnsafeIntegers("claims", claims)
		for _, v := range got {
			n, isInt := new(big.Int).SetString(v.Literal, 10)
			if !isInt {
				t.Fatalf("reported %q at %s, which is not a base-10 integer", v.Literal, v.Path)
			}
			if new(big.Int).Abs(n).Cmp(maxSafeInteger) <= 0 {
				t.Fatalf("reported %q at %s, which is inside the safe range", v.Literal, v.Path)
			}
			if !bytes.Contains(data, []byte(v.Literal)) {
				t.Fatalf("reported %q at %s, which does not occur in the input, the literal was rewritten", v.Literal, v.Path)
			}
		}

		// Determinism across a freshly decoded copy: same map contents, new
		// map, therefore a new randomized iteration order.
		other, ok := decode(data)
		if !ok {
			t.Fatal("input decoded once but not twice")
		}
		if !reflect.DeepEqual(got, UnsafeIntegers("claims", other)) {
			t.Fatalf("non-deterministic result for %q", data)
		}

		// COMPLETENESS. The checks above are all soundness properties: they
		// constrain what the walker DOES report. A walker that silently skips
		// part of the document satisfies every one of them, and a false
		// negative is precisely the failure this guard exists to prevent, an
		// unsafe integer reaching the wire. So cross-check the value walk
		// against an independent scan of the raw bytes.
		//
		// gowebpki/jcs is the duplicate-name filter: a shadowed name
		// ({"a":<oversized>,"a":1}) leaves the literal in the bytes but not in
		// the decoded map, and the library refuses a repeated name, so a text
		// it accepts holds every literal as a value of the map.
		if _, err := gojcs.Transform(data); err != nil {
			return
		}
		reported := make(map[string]bool, len(got))
		for _, v := range got {
			reported[v.Literal] = true
		}
		for _, lit := range integerLiterals(data) {
			n, _ := new(big.Int).SetString(lit, 10)
			if new(big.Int).Abs(n).Cmp(maxSafeInteger) > 0 && !reported[lit] {
				t.Fatalf("raw scan found oversized literal %q that the value walk did not report, the walker is skipping part of the document", lit)
			}
		}
	})
}

// integerLiterals returns every number of the JSON text data that is a
// base-10 integer literal, as written, read by encoding/json's tokenizer,
// or none when data is not a single JSON text.
func integerLiterals(data []byte) []string {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var out []string
	for {
		tok, err := dec.Token()
		if err != nil {
			return out
		}
		if num, ok := tok.(json.Number); ok {
			if _, isInt := new(big.Int).SetString(num.String(), 10); isInt {
				out = append(out, num.String())
			}
		}
	}
}
