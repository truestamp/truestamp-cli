// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"
)

// The table of SPEC.md section 10.2, step 13, as SPEC.md writes it: each
// row's hash_types and the number of hex characters of their hash.
var specHashTable = []struct {
	types      string
	characters int
}{
	{"md5", 32},
	{"sha1", 40},
	{"sha224, sha3_224", 56},
	{"sha256, sha3_256, blake2s", 64},
	{"sha384, sha3_384", 96},
	{"sha512, sha3_512, blake2b", 128},
}

// The step's table is SPEC.md's, name for name and length for length, and
// only md5 and sha1 are weak. Mutant killed: sha3_384 given 64 characters
// (or any one entry changed, added or removed).
func TestExpectedHash_TheTableIsSPECs(t *testing.T) {
	want := map[string]int{}
	for _, row := range specHashTable {
		for _, name := range strings.Split(row.types, ", ") {
			want[name] = row.characters
		}
	}
	if !maps.Equal(hashHexCharacters, want) || len(want) != 12 {
		t.Fatalf("the table %v, want %v", hashHexCharacters, want)
	}
	if !maps.Equal(weakHashTypes, map[string]bool{"md5": true, "sha1": true}) {
		t.Fatalf("weak %v", weakHashTypes)
	}
}

// expectedHashRow is EXPECTED_HASH's row for given (nil for none) over
// proof, in a run past DERIVE with attribution, its values as JSON.
func expectedHashRow(t *testing.T, proof map[string]any, attribution string, given *string) (string, string, string) {
	t.Helper()
	run := deriveRun(proof, attribution)
	run.Config.ExpectedHash = given
	before := len(run.Rows)
	ExpectedHash(run)
	if len(run.Rows) != before+1 {
		t.Fatalf("EXPECTED_HASH gave %d rows", len(run.Rows)-before)
	}
	row := run.Rows[before]
	values, err := json.Marshal(row.Values)
	if err != nil {
		t.Fatal(err)
	}
	return rowSummary([]Row{row}), string(values), row.Message
}

// Step 13 row by row (SPEC.md section 10.2): no hash given is skip
// expected_hash_not_checked, part null; a hash given is one row at
// /subject/claims/hash: pass on a match, lowercased (only ASCII letters),
// warn expected_hash_weak_algorithm on a match under md5 or sha1, and
// otherwise fail expected_hash_mismatch with its reason: no_listed_hash
// for a proof that is not an item, an item with no hash, a hash that is
// not a string and a hash_type the table does not list; not_hex_of_length,
// with the length, for a value too long, too short, with a newline on its
// end (never trimmed) or with a letter past f; and differs, with the
// proof's hash as declared. The values hold the caller's value lowercased.
// Mutants killed: the caller's value not lowercased (the capitals
// mismatch); trimmed (the newline matches); the length unchecked (the
// value one character long is a mismatch named differs); the hex check
// taking every lowercase letter (the "g" value is differs); md5 not weak
// (a pass); the type unchecked (a run of type block over an item's claims,
// which SHAPE never lets through, matches); a hash_type not in the table
// taken (crc32 fails at its length, not as no_listed_hash); a hash that is
// not a string compared (differs).
func TestExpectedHash_Rows(t *testing.T) {
	item := positiveProofs(t)["item"]
	md5 := "69e3941192b086bfdc0e4e120c18ff12"
	sha := put(put(item, Pointer{Name("subject"), Name("claims"), Name("hash_type")}, "sha256"),
		Pointer{Name("subject"), Name("claims"), Name("hash")}, strings.Repeat("0a", 32))
	text := func(s string) *string { return &s }
	at := "/subject/claims/hash"
	for _, c := range []struct {
		name   string
		proof  map[string]any
		given  *string
		row    string
		values string
	}{
		{"none given", item, nil, "[expected_hash_not_checked null skip EXPECTED_HASH]", `{}`},
		{"sha256, capitals", sha, text(strings.Repeat("0A", 32)), "[null " + at + " pass EXPECTED_HASH]",
			`{"expected_hash":"` + strings.Repeat("0a", 32) + `"}`},
		{"md5", item, text(md5), "[expected_hash_weak_algorithm " + at + " warn EXPECTED_HASH]", `{"expected_hash":"` + md5 + `"}`},
		{"sha1", put(put(item, Pointer{Name("subject"), Name("claims"), Name("hash_type")}, "sha1"),
			Pointer{Name("subject"), Name("claims"), Name("hash")}, strings.Repeat("1", 40)), text(strings.Repeat("1", 40)),
			"[expected_hash_weak_algorithm " + at + " warn EXPECTED_HASH]", `{"expected_hash":"` + strings.Repeat("1", 40) + `"}`},
		{"differs", item, text("09" + md5[2:]), "[expected_hash_mismatch " + at + " fail EXPECTED_HASH]",
			`{"declared":"` + md5 + `","expected_hash":"09` + md5[2:] + `","reason":"differs"}`},
		{"too long", item, text(md5 + "00"), "[expected_hash_mismatch " + at + " fail EXPECTED_HASH]",
			`{"expected_hash":"` + md5 + `00","hex_characters":32,"reason":"not_hex_of_length"}`},
		{"one character", item, text("6"), "[expected_hash_mismatch " + at + " fail EXPECTED_HASH]",
			`{"expected_hash":"6","hex_characters":32,"reason":"not_hex_of_length"}`},
		{"a newline on its end", item, text(md5 + "\n"), "[expected_hash_mismatch " + at + " fail EXPECTED_HASH]",
			`{"expected_hash":"` + md5 + `\n","hex_characters":32,"reason":"not_hex_of_length"}`},
		{"a letter past f", item, text("g" + md5[1:]), "[expected_hash_mismatch " + at + " fail EXPECTED_HASH]",
			`{"expected_hash":"g` + md5[1:] + `","hex_characters":32,"reason":"not_hex_of_length"}`},
		{"not an item", positiveProofs(t)["block"], text(md5), "[expected_hash_mismatch " + at + " fail EXPECTED_HASH]",
			`{"expected_hash":"` + md5 + `","reason":"no_listed_hash"}`},
		{"an item with no hash", put(item, Pointer{Name("subject"), Name("claims")}, map[string]any{"name": "x"}), text(md5),
			"[expected_hash_mismatch " + at + " fail EXPECTED_HASH]", `{"expected_hash":"` + md5 + `","reason":"no_listed_hash"}`},
		{"a hash that is not a string", put(item, Pointer{Name("subject"), Name("claims"), Name("hash")}, []any{md5}), text(md5),
			"[expected_hash_mismatch " + at + " fail EXPECTED_HASH]", `{"expected_hash":"` + md5 + `","reason":"no_listed_hash"}`},
		{"a block's type over an item's claims", put(item, Pointer{Name("type")}, "block"), text(md5),
			"[expected_hash_mismatch " + at + " fail EXPECTED_HASH]", `{"expected_hash":"` + md5 + `","reason":"no_listed_hash"}`},
		{"an unlisted hash_type", put(item, Pointer{Name("subject"), Name("claims"), Name("hash_type")}, "crc32"), text(md5),
			"[expected_hash_mismatch " + at + " fail EXPECTED_HASH]", `{"expected_hash":"` + md5 + `","reason":"no_listed_hash"}`},
	} {
		row, values, _ := expectedHashRow(t, c.proof, deriveAttribution, c.given)
		if row != c.row || values != c.values {
			t.Errorf("%s: %s %s\nwant %s %s", c.name, row, values, c.row, c.values)
		}
	}
}

// A match's message says, when the attribution is not truestamp, that it
// is against a document not attributed to Truestamp, and says nothing of
// the kind under truestamp (SPEC.md section 10.2, step 13). A value that
// is not UTF-8 is not held in the values, which JCS could not write.
// Mutants killed: the attribution's sentence on every match (the
// truestamp run's message has it); on none (the supplied keyring's lacks
// it); the caller's value held whatever its bytes (the values hold bytes
// that are not UTF-8).
func TestExpectedHash_TheMessageNamesTheAttribution(t *testing.T) {
	item := positiveProofs(t)["item"]
	md5 := "69e3941192b086bfdc0e4e120c18ff12"
	const sentence = "in a document not attributed to Truestamp"
	if _, _, message := expectedHashRow(t, item, AttributionTruestamp, &md5); strings.Contains(message, sentence) {
		t.Errorf("truestamp: %q", message)
	}
	for _, attribution := range []string{deriveAttribution, AttributionUnattributed} {
		if _, _, message := expectedHashRow(t, item, attribution, &md5); !strings.Contains(message, sentence) {
			t.Errorf("%s: %q", attribution, message)
		}
	}
	invalid := "\xff" + md5[1:]
	if row, values, _ := expectedHashRow(t, item, deriveAttribution, &invalid); values != `{"hex_characters":32,"reason":"not_hex_of_length"}` ||
		row != "[expected_hash_mismatch /subject/claims/hash fail EXPECTED_HASH]" {
		t.Errorf("a value that is not UTF-8: %s %s", row, values)
	}
}
