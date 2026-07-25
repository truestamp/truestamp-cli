// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package jcs

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	gojcs "github.com/gowebpki/jcs"
)

// FuzzCanonicalize drives the canonicalizer with arbitrary bytes. Three
// invariants are asserted, the last of which is the load-bearing one: a
// document with no oversized integer must produce exactly what
// github.com/gowebpki/jcs produces, so this package can never be the reason a
// digest that verifies today stops verifying.
func FuzzCanonicalize(f *testing.F) {
	seeds := []string{
		// Appendix C.2a conformance vectors.
		`{"a": null, "b": 1}`,
		`{"c": "\u0001", "tab": "\t"}`,
		// The same document with a raw U+0001 byte in the value. RFC 8259 §7
		// forbids an unescaped control character, so this one is now rejected
		// by the strict gate; it is kept as a seed because that rejection is
		// itself an invariant worth fuzzing around.
		"{\"c\": \"\x01\", \"tab\": \"\\t\"}",
		`{"z": -0.0}`,
		`{"big": 1.0e21, "small": 1.0e-7}`,
		"{\"name\": \"Ávila\", \"emoji\": \"\U0001f600\"}",
		"{\"\U0001f600\": 1, \"￿\": 2}",
		`{"n": 9007199254740992}`,
		`{"n": 9007199254740993}`,

		// Subject-data shapes the verifier actually canonicalizes.
		`{"description":"Appendix D worked example","hash":"b47cc0","hash_type":"sha256","name":"x"}`,
		`{"chainIndex":2,"outputValue":"AB","pulseIndex":1234567,"timeStamp":"2026-04-06T23:00:00.000Z","version":"1.0"}`,

		// Integers at and beyond the exactly representable range.
		`{"ledgerSequence":18446744073709551615}`,
		`{"neg":-9007199254740993,"dup":[9007199254740993,9007199254740993]}`,
		`{"nested":{"deep":[{"x":123456789012345678901234567890}]}}`,
		`{"s":"9007199254740993","n":9007199254740993}`,
		`{"nonce":` + strings.Repeat("9", 400) + `}`,
		`9007199254740993`,
		`[9007199254740993]`,

		// Inputs the two parsers disagree about, plus plain malformed bytes.
		// The first four are the documents gowebpki/jcs silently rewrites into
		// DIFFERENT valid documents; the strict gate has to catch every one.
		`{"a":1 2,"b":9007199254740993}`,
		`{"a":+1}`,
		`{"a":0009}`,
		`{"a":.5}`,
		`{"huge":1e400}`,
		`{"a":1}{"b":2}`,
		`{`,
		``,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		canonical, oversized, err := Canonicalize(data)
		if err != nil {
			if canonical != nil || oversized != nil {
				t.Fatalf("error return carried a result: canonical=%q oversized=%v", canonical, oversized)
			}
			return
		}

		// The strict gate, as an invariant: nothing that is not a single
		// well-formed RFC 8259 document may reach a caller as a digestible
		// canonical form. gowebpki/jcs on its own rewrites several such inputs
		// into different valid documents, so an accept here would be a silently
		// wrong claims_hash rather than a crash.
		if !json.Valid(data) {
			t.Fatalf("accepted RFC-8259-invalid input %q as %q", data, canonical)
		}
		if !json.Valid(canonical) {
			t.Fatalf("canonical output is not valid JSON: %q", canonical)
		}

		again, _, err := Canonicalize(canonical)
		if err != nil {
			t.Fatalf("re-canonicalizing %q failed: %v", canonical, err)
		}
		if !bytes.Equal(canonical, again) {
			t.Fatalf("not idempotent: %q then %q", canonical, again)
		}

		if oversized == nil {
			want, wantErr := gojcs.Transform(data)
			if wantErr != nil {
				t.Fatalf("gowebpki/jcs rejected input we accepted: %v", wantErr)
			}
			if !bytes.Equal(canonical, want) {
				t.Fatalf("diverged from gowebpki/jcs with no oversized integer: got %q, want %q", canonical, want)
			}
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
				t.Fatalf("reported %q at %s, which does not occur in the input — the literal was rewritten", v.Literal, v.Path)
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
		// negative is precisely the failure this guard exists to prevent — an
		// unsafe integer reaching the wire. So cross-check the value walk
		// against an independent scan of the raw bytes.
		//
		// Canonicalize doubles as the duplicate-key filter. A shadowed key
		// ({"a":<oversized>,"a":1}) leaves the literal in the bytes but not in
		// the decoded map, which would make the comparison spuriously fail;
		// gowebpki/jcs rejects duplicate keys outright, so a successful
		// canonicalization means every literal in the input is also a value in
		// the map.
		if _, _, err := Canonicalize(data); err != nil {
			return
		}
		spans, err := scanIntegers(data)
		if err != nil {
			return
		}
		reported := make(map[string]bool, len(got))
		for _, v := range got {
			reported[v.Literal] = true
		}
		for _, sp := range spans {
			if new(big.Int).Abs(sp.val).Cmp(maxSafeInteger) <= 0 {
				continue
			}
			if !reported[string(sp.lit)] {
				t.Fatalf("raw scan found oversized literal %q that the value walk did not report — the walker is skipping part of the document", sp.lit)
			}
		}
	})
}
