// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package items

import (
	"strings"
	"testing"
)

// TestClaimsJSONString pins the JSON string a claims value is written as:
// only a quotation mark, a reverse solidus and the control characters are
// escaped. Mutant killed: v1's encoding/json used to write it ("<" becomes
// \u003c, and U+2028 \u2028).
func TestClaimsJSONString(t *testing.T) {
	got := string(ClaimsJSONString("a\"b\\c\n\x01<&>\u2028é"))
	want := `"a\"b\\c\u000a\u0001<&>` + "\u2028é" + `"`
	if got != want {
		t.Errorf("ClaimsJSONString = %s, want %s", got, want)
	}
}

// TestLongNumber finds the first number literal over 1,024 octets as the
// service's scan does: a literal of exactly 1,024 octets is not one, a
// sign, point and exponent count toward the length, and digits inside a
// string, escaped quotes included, are not a literal. Mutants killed: the
// limit compared with >= (the 1,024-octet literal is found), and a string
// left unskipped (the digits in it are found).
func TestLongNumber(t *testing.T) {
	at1024 := "1." + strings.Repeat("0", 1022)
	over := "-1." + strings.Repeat("0", 1019) + "e+1"
	for _, tc := range []struct {
		name    string
		text    string
		offset  int
		length  int
		present bool
	}{
		{"exactly the limit", `{"n":` + at1024 + `}`, 0, 0, false},
		{"one over, sign and exponent counted", `{"a":[1,"x"],"n":` + over + `}`, 17, 1025, true},
		{"digits in a string", `{"s":"` + strings.Repeat("9", 2000) + `"}`, 0, 0, false},
		{"an escaped quote in a string", `{"s":"\"` + strings.Repeat("9", 2000) + `","n":1}`, 0, 0, false},
		{"after a string", `{"s":"\\","n":` + strings.Repeat("9", 1025) + `}`, 14, 1025, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			offset, length, present := LongNumber([]byte(tc.text))
			if offset != tc.offset || length != tc.length || present != tc.present {
				t.Errorf("LongNumber = %d, %d, %v, want %d, %d, %v", offset, length, present, tc.offset, tc.length, tc.present)
			}
		})
	}
}
