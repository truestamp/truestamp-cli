// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"slices"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// specBytes is the table of SPEC.md section 4.3, byte by construction.
var specBytes = map[Construction]byte{
	MerkleLeaf:               0x00,
	MerkleNode:               0x01,
	ClaimsHash:               0x10,
	ItemHash:                 0x11,
	EntropyHash:              0x20,
	ObservationHash:          0x21,
	GenesisPreviousBlockHash: 0x30,
	BlockHash:                0x31,
	EpochHash:                0x40,
	ProofHash:                0x50,
	KeyID:                    0x60,
	PrerotationCommitment:    0x61,
	Handover:                 0x62,
}

// entryOf is c's row, which the test needs to exist.
func entryOf(t *testing.T, c Construction) entry {
	t.Helper()
	row, ok := c.entry()
	if !ok {
		t.Fatalf("%v names no row", c)
	}
	return row
}

// The registry equals spec/prefixes.json, read by the strict reader (which
// refuses a repeated name and trailing content), row by row and member by
// member, with no row or member left over. Mutants killed: a byte of the
// table changed (item_hash 0x11 to 0x13); a signer changed or dropped; a
// row's name changed.
func TestPrefixes_EqualSpecPrefixesJSON(t *testing.T) {
	file, ok := mustParseSpecFile(t, "prefixes.json").(map[string]any)
	if !ok {
		t.Fatal("spec/prefixes.json is not an object")
	}
	if len(file) != len(constructions()) {
		t.Fatalf("spec/prefixes.json has %d rows, the registry %d", len(file), len(constructions()))
	}
	for _, c := range constructions() {
		entry := entryOf(t, c)
		row, ok := file[entry.Name].(map[string]any)
		if !ok {
			t.Errorf("spec/prefixes.json has no row %q", entry.Name)
			continue
		}
		want := map[string]any{
			"construction": entry.Construction,
			"family":       entry.Family,
			"hashed":       entry.Hashed,
			"signed":       entry.Signed,
		}
		if entry.Signer != "" {
			want["signer"] = entry.Signer
		}
		text, ok := row["byte"].(string)
		if !ok {
			t.Errorf("%s: byte is not a string", entry.Name)
			continue
		}
		if got, err := specdata.PrefixByte(text); err != nil || got != entry.Byte {
			t.Errorf("%s: byte %q, the registry %#02x (%v)", entry.Name, text, entry.Byte, err)
		}
		if len(row) != len(want)+1 {
			t.Errorf("%s: the file's row has %d members, the registry's %d", entry.Name, len(row), len(want)+1)
		}
		for member, value := range want {
			if row[member] != value {
				t.Errorf("%s: %s is %v in the file, %v in the registry", entry.Name, member, row[member], value)
			}
		}
	}
}

// Each construction has the byte SPEC.md section 4.3 gives it,
// constructions lists them in the order of their bytes, and exactly the
// three kinds of SPEC.md section 4.1 are signed. Mutants killed: two rows
// of constructions swapped (ClaimsHash and ItemHash); a byte changed;
// a fourth kind marked signed.
func TestPrefixes_AreTheBytesOfSPECTable(t *testing.T) {
	all := constructions()
	if len(all) != 13 || len(specBytes) != 13 {
		t.Fatalf("%d constructions, want SPEC.md section 4.3's 13", len(all))
	}
	var signed []Construction
	for i, c := range all {
		entry := entryOf(t, c)
		if entry.Byte != specBytes[c] {
			t.Errorf("%v has byte %#02x, SPEC.md section 4.3 gives %#02x", c, entry.Byte, specBytes[c])
		}
		if i > 0 && entryOf(t, all[i-1]).Byte >= entry.Byte {
			t.Errorf("constructions lists %v after %v, out of byte order", c, all[i-1])
		}
		if c != MerkleLeaf+Construction(i) {
			t.Errorf("constructions()[%d] is %v, want the constant %d", i, c, int(MerkleLeaf)+i)
		}
		if entry.Signed {
			signed = append(signed, c)
		}
		if entry.Signed != (entry.Signer != "") {
			t.Errorf("%v: signed %v with signer %q", c, entry.Signed, entry.Signer)
		}
	}
	if want := []Construction{BlockHash, ProofHash, Handover}; !slices.Equal(signed, want) {
		t.Errorf("the signed kinds are %v, want %v", signed, want)
	}
}

// A value that names no row has no entry and no byte. Mutant killed:
// entry's upper bound off by one (c > Handover+1), which reads past the
// table.
func TestConstruction_ValuesThatNameNoRow(t *testing.T) {
	for _, c := range []Construction{0, -1, Handover + 1, 255} {
		if row, ok := c.entry(); ok {
			t.Errorf("%d names the row %v", int(c), row)
		}
	}
	if got, want := Construction(14).String(), "Construction(14)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got := GenesisPreviousBlockHash.String(); got != "genesis_previous_block_hash" {
		t.Errorf("String() = %q, want the row's name", got)
	}
}
