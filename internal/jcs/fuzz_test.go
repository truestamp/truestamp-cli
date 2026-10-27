// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

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
