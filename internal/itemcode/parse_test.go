// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package itemcode

import (
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"os"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// readerAnswers is the "accepted" and "refused" lists of
// vectors/item-codes.json (SPEC.md section 13.2), each input as the bytes
// its hex spells.
func readerAnswers(t *testing.T) (accepted []struct{ about, input, code string }, refused []struct{ about, input string }) {
	t.Helper()
	data, err := os.ReadFile(vectorcases.Path("item-codes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Accepted []struct {
			About    string `json:"about"`
			Input    string `json:"input"`
			ItemCode string `json:"item_code"`
		} `json:"accepted"`
		Refused []struct {
			About string `json:"about"`
			Input string `json:"input"`
		} `json:"refused"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Accepted) != 25 || len(file.Refused) != 99 {
		t.Fatalf("item-codes.json has %d accepted and %d refused answers, want 25 and 99", len(file.Accepted), len(file.Refused))
	}
	decode := func(about, h string) string {
		raw, err := hex.DecodeString(h)
		if err != nil {
			t.Fatalf("%s: input %q is not hex", about, h)
		}
		return string(raw)
	}
	for _, a := range file.Accepted {
		accepted = append(accepted, struct{ about, input, code string }{a.About, decode(a.About, a.Input), a.ItemCode})
	}
	for _, r := range file.Refused {
		refused = append(refused, struct{ about, input string }{r.About, decode(r.About, r.Input)})
	}
	return accepted, refused
}

// Parse gives every accepted answer of vectors/item-codes.json its code in
// its one spelling, and refuses every refused one. Mutants killed: a
// TrimSpace before reading (" 9WBPKNV7SCP87VZW" and its tab and line feed
// forms are accepted); two separators together allowed
// ("9WBPKNV7--SCP87VZW" is accepted); the I, L or O readings left out (the
// O, I and L answers are refused); the case fold left out (the lowercase
// answers are refused); the check left out (the wrong and exchanged
// symbols are accepted); the U read as a symbol (the U answers are
// accepted).
func TestParse_GivesTheReaderKnownAnswers(t *testing.T) {
	accepted, refused := readerAnswers(t)
	for _, a := range accepted {
		if got, err := Parse(a.input); err != nil || got != a.code {
			t.Errorf("%s: Parse(%q) = %q, %v; want %q", a.about, a.input, got, err, a.code)
		}
	}
	for _, r := range refused {
		if got, err := Parse(r.input); err == nil {
			t.Errorf("%s: Parse(%q) = %q; want a refusal", r.about, r.input, got)
		}
	}
}

// Parse says which refusal it is: a value with the shape of a code, four
// groups of four ASCII letters or digits, that holds a U or fails the check
// is ErrTypo, and anything else ErrNotACode. Mutants killed: a value too
// short or too long reported as ErrTypo (the 15-symbol, 20-character and
// empty values are typos); a failed check reported as ErrNotACode (the
// wrong check symbol is not a typo).
func TestParse_TellsATypoFromAValueThatIsNoCode(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  error
	}{
		{"9WBPKNV7SCP87VZX", ErrTypo},
		{"9WBPKNV7SCP8V7ZW", ErrTypo},
		{"9WBUKNV7SCP87VZW", ErrTypo},
		{"9wbp-knv7-scp8-7vzu", ErrTypo},
		{"1234567890123456", ErrTypo},
		{"9WBPKNV7SCP87VZ", ErrNotACode},
		{"9WBPKNV7SCP87VZW\n", ErrNotACode},
		{"9WBP.KNV7.SCP8.7VZW", ErrNotACode},
		{"9WBPKNV7SCP87VZW0000", ErrNotACode},
		{"", ErrNotACode},
	} {
		if got, err := Parse(tc.input); !errors.Is(err, tc.want) || got != "" {
			t.Errorf("Parse(%q) = %q, %v; want %v", tc.input, got, err, tc.want)
		}
	}
}
