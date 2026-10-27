// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package proof

import (
	"encoding/json"
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
			got, err := FileName("item", "01HJHB01T8FYZ7YTR9P5N62K5B", doc)
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
		if got, err := FileName("item", "01HJHB01T8FYZ7YTR9P5N62K5B", served); err == nil || !strings.HasPrefix(err.Error(), "no file name for the served proof: ") {
			t.Errorf("%s: FileName = %q, %v; want a refusal", name, got, err)
		}
	}
}
