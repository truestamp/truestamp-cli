// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package itemcode

import (
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"os"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// deriveAnswers is the "derive" list of vectors/item-codes.json (SPEC.md
// section 13.2): each recomputed item_hash with the item code it gives.
func deriveAnswers(t *testing.T) []struct {
	About    string `json:"about"`
	ItemHash string `json:"item_hash"`
	ItemCode string `json:"item_code"`
	Proof    string `json:"proof"`
} {
	t.Helper()
	data, err := os.ReadFile(vectorcases.Path("item-codes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Derive []struct {
			About    string `json:"about"`
			ItemHash string `json:"item_hash"`
			ItemCode string `json:"item_code"`
			Proof    string `json:"proof"`
		} `json:"derive"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Derive) != 10 {
		t.Fatalf("item-codes.json has %d derive answers, want 10", len(file.Derive))
	}
	return file.Derive
}

// FromItemHash gives every known answer of the derive list. Mutants
// killed: the check value written as r without the final m (the last
// symbol differs); the values taken least significant bit first; mulX
// without its reduction by 37 (the all-ones check value is past the
// alphabet, and indexing it panics).
func TestFromItemHash_GivesTheKnownAnswers(t *testing.T) {
	for _, answer := range deriveAnswers(t) {
		raw, err := hex.DecodeString(answer.ItemHash)
		if err != nil || len(raw) != 32 {
			t.Fatalf("%s: item_hash %q is not 32 bytes of hex", answer.About, answer.ItemHash)
		}
		var h [32]byte
		copy(h[:], raw)
		if got := FromItemHash(h); got != answer.ItemCode {
			t.Errorf("%s: FromItemHash = %q, want %q", answer.About, got, answer.ItemCode)
		}
	}
}

// OfItemRecord gives, for each derive answer that names a proof, the
// answer's code from the record that proof carries at
// /proof/subject/record, and refuses a nil record as the registry does.
// Mutant killed: the record hashed under another kind, such as
// registry.ClaimsHash (every code differs).
func TestOfItemRecord_GivesTheCodeOfTheRecomputedItemHash(t *testing.T) {
	proofs := 0
	for _, answer := range deriveAnswers(t) {
		if answer.Proof == "" {
			continue
		}
		proofs++
		data, err := os.ReadFile(vectorcases.Path(answer.Proof))
		if err != nil {
			t.Fatal(err)
		}
		value, err := strictjson.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		record := value.(map[string]any)["proof"].(map[string]any)["subject"].(map[string]any)["record"].(map[string]any)
		if got, err := OfItemRecord(record); err != nil || got != answer.ItemCode {
			t.Errorf("%s: OfItemRecord = %q, %v; want %q", answer.Proof, got, err, answer.ItemCode)
		}
	}
	if proofs != 3 {
		t.Fatalf("%d derive answers name a proof, want 3", proofs)
	}
	if got, err := OfItemRecord(nil); !errors.Is(err, registry.ErrNotObject) || got != "" {
		t.Errorf("OfItemRecord(nil) = %q, %v; want the registry's ErrNotObject", got, err)
	}
}
