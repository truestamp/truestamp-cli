// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package proof

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// itemProofs are the derive answers of vectors/item-codes.json that name a
// proof file: the proof and the item code of its recomputed item_hash.
func itemProofs(t *testing.T) map[string]string {
	t.Helper()
	data, err := os.ReadFile(vectorcases.Path("item-codes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Derive []struct {
			ItemCode string `json:"item_code"`
			Proof    string `json:"proof"`
		} `json:"derive"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, d := range file.Derive {
		if d.Proof != "" {
			out[d.Proof] = d.ItemCode
		}
	}
	if len(out) != 3 {
		t.Fatalf("item-codes.json names %d proofs, want 3", len(out))
	}
	return out
}

// An item proof is named by the item code of the item_hash recomputed
// over its record, with nothing for its id or a constructions mode, and
// that code is the known answer of vectors/item-codes.json for each of its
// three proofs; changing the carried item_hash changes nothing. Mutants
// killed: the code computed from the carried /proof/subject/item_hash
// (the altered proof is named 0000000000000000.proof.json); the name built
// from the id (the name differs).
func TestFileName_AnItemIsNamedByItsRecomputedCode(t *testing.T) {
	for path, code := range itemProofs(t) {
		served, err := os.ReadFile(vectorcases.Path(path))
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(served, &value); err != nil {
			t.Fatal(err)
		}
		subject := value["proof"].(map[string]any)["subject"].(map[string]any)
		carried := subject["item_hash"].(string)
		altered := strings.ReplaceAll(string(served), carried, strings.Repeat("0", 64))
		if altered == string(served) {
			t.Fatalf("%s: the carried item_hash was not replaced", path)
		}
		for _, doc := range [][]byte{served, []byte(altered)} {
			got, err := FileName("item", "01a0db66-c2e0-78a4-b8cc-12b11bc97493", doc)
			if err != nil || got != code+".proof.json" {
				t.Errorf("%s: FileName = %q, %v; want %q", path, got, err, code+".proof.json")
			}
		}
	}
}

// Every other subject is named by its id as given. An id that could name a
// file outside the current directory, or none, gives no name, and so does
// an item request whose served bytes are no item proof with a record.
// Mutants killed: FileName without its path-separator check (../x is
// named ../x.proof.json); without its item check (the block that carries
// a record is named).
func TestFileName_OtherSubjectsAndRefusals(t *testing.T) {
	const id = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
	for _, typ := range []string{"block", "beacon", "entropy_nist", "entropy_stellar", "entropy_bitcoin"} {
		if got, err := FileName(typ, id, nil); err != nil || got != id+".proof.json" {
			t.Errorf("%s: FileName = %q, %v; want %q", typ, got, err, id+".proof.json")
		}
	}
	for _, bad := range []string{"../x", `a\b`, "a/b", "", ".", ".."} {
		if got, err := FileName("block", bad, nil); err == nil {
			t.Errorf("FileName(block, %q) = %q, want a refusal", bad, got)
		}
	}
	block, err := os.ReadFile(vectorcases.Path("positive/block.proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	for name, served := range map[string][]byte{
		"a block proof":                 block,
		"a block with an item's record": []byte(`{"proof":{"type":"block","subject":{"record":{"id":"x"}}},"signature":{}}`),
		"no item record":                []byte(`{"proof":{"type":"item","subject":{}},"signature":{}}`),
		"not JSON":                      []byte(`{"proof":`),
		"an array of proof":             []byte(`[{"proof":{"type":"item"}}]`),
	} {
		if got, err := FileName("item", "01a0db66-c2e0-78a4-b8cc-12b11bc97493", served); err == nil || !strings.HasPrefix(err.Error(), "no file name for the served proof: ") {
			t.Errorf("%s: FileName = %q, %v; want a refusal", name, got, err)
		}
	}
}

// CheckServedItem takes the proof served for an item code only when the
// code of its recomputed item_hash is that code, for an item_hash only
// when its recomputed item_hash is that hash, all 64 characters, and for a
// UUIDv7 only when its record's id is that UUIDv7: the sound item's own
// code, hash and id pass; another item's proof is ErrDifferentItem for
// all three; and a hash that differs from the recomputed one only in its
// last character, whose code is the same, is ErrDifferentItem too. The
// carried item_hash is never read: a proof whose carried hash is another
// still passes, by code, hash or id. A served document with no record to
// read is refused, by code and by id, and a subject other than an item is
// not checked. Mutants killed: the check left out (every case passes); an
// item_hash compared by its code (the hash one character off passes); the
// carried item_hash compared (the declared proof is refused); a code
// compared with the code of the carried hash; the UUIDv7 not checked
// (another item's proof by id passes); the carried item_hash compared in
// place of the record id (the sound item's proof by id is refused); the
// subject type not read (the block's id is refused); the record id compared
// case-insensitively (a record whose id is the uppercase form passes).
func TestCheckServedItem_TheServedItemIsTheOneNamed(t *testing.T) {
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(vectorcases.Path(path))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	const (
		code     = "WX495M26THGQBJ68"
		itemHash = "e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c"
		nearHash = "e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8d"
		soundID  = "01a0db66-c2e0-78a4-b8cc-12b11bc97493"
	)
	sound, declared, other := read("positive/item.proof.json"), read("negative/declared-item_hash.proof.json"), read("positive/item-genesis-witness.proof.json")
	for _, tc := range []struct {
		name   string
		shape  IDType
		ref    string
		served []byte
		want   string
	}{
		{"its code", IDTypeItemCode, code, sound, ""},
		{"its hash", IDTypeItemHash, itemHash, sound, ""},
		{"its code, another hash carried", IDTypeItemCode, code, declared, ""},
		{"its hash, another hash carried", IDTypeItemHash, itemHash, declared, ""},
		{"another item's proof by code", IDTypeItemCode, code, other,
			"the server returned a proof for a different item: its recomputed item_hash 8d4a6cc4b1608cefa713b45d3fc3e09e3390d64556e3ab4e0725e386b0dab8c7 has the item code HN56SH5HC26EZ9R4, not " + code + "; nothing was written"},
		{"another item's proof by hash", IDTypeItemHash, itemHash, other,
			"the server returned a proof for a different item: its recomputed item_hash is 8d4a6cc4b1608cefa713b45d3fc3e09e3390d64556e3ab4e0725e386b0dab8c7, not " + itemHash + "; nothing was written"},
		{"a hash of the same code", IDTypeItemHash, nearHash, sound,
			"the server returned a proof for a different item: its recomputed item_hash is " + itemHash + ", not " + nearHash + "; nothing was written"},
		{"a block proof by code", IDTypeItemCode, code, read("positive/block.proof.json"),
			"the served proof cannot be checked against " + code + ", so nothing was written: it is not an item proof"},
		{"its id", IDTypeUUIDv7, soundID, sound, ""},
		{"its id, another hash carried", IDTypeUUIDv7, soundID, declared, ""},
		{"another item's proof by id", IDTypeUUIDv7, soundID, other,
			`the server returned a proof for a different item: its record's id is "01a0db6d-2b80-7928-9225-f8760ae1a795", not ` + soundID + "; nothing was written"},
		{"a block proof by id", IDTypeUUIDv7, soundID, read("positive/block.proof.json"),
			"the served proof cannot be checked against " + soundID + ", so nothing was written: it is not an item proof"},
		{"a record whose id is its uppercase form", IDTypeUUIDv7, soundID, []byte(`{"proof":{"type":"item","subject":{"record":{"id":"` + strings.ToUpper(soundID) + `"}}},"signature":{}}`),
			`the server returned a proof for a different item: its record's id is "` + strings.ToUpper(soundID) + `", not ` + soundID + "; nothing was written"},
		{"a record with no id", IDTypeUUIDv7, soundID, []byte(`{"proof":{"type":"item","subject":{"record":{"id":1}}},"signature":{}}`),
			"the served proof cannot be checked against " + soundID + ", so nothing was written: its item record has no string id"},
	} {
		err := CheckServedItem("item", tc.shape, tc.ref, tc.served)
		switch {
		case tc.want == "" && err != nil:
			t.Errorf("%s: %v; want nil", tc.name, err)
		case tc.want != "" && (err == nil || err.Error() != tc.want):
			t.Errorf("%s: %v; want %q", tc.name, err, tc.want)
		}
		if tc.want != "" && strings.HasPrefix(tc.want, "the server returned") && !errors.Is(err, ErrDifferentItem) {
			t.Errorf("%s: %v is not ErrDifferentItem", tc.name, err)
		}
	}
	for _, typ := range []string{"block", "beacon", "entropy_nist", "entropy_stellar", "entropy_bitcoin"} {
		if err := CheckServedItem(typ, IDTypeUUIDv7, "019db702-b08c-73dc-a7cd-2c5e011f1dad", read("positive/block.proof.json")); err != nil {
			t.Errorf("%s by id: %v; want nil", typ, err)
		}
	}
}
