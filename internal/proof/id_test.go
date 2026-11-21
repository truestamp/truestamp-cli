// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package proof

import (
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// idShapeRow is one row of testdata/idshape/id_shapes.json, the copy of
// the app's known answers for Truestamp.IdShape (its SOURCE names the
// commit): the input as the hex of its bytes, the class the app gives it,
// that class's canonical value, and what an item slot makes of it.
type idShapeRow struct {
	About   string `json:"about"`
	Input   string `json:"input"`
	Class   string `json:"class"`
	Value   any    `json:"value"`
	ItemRef string `json:"item_ref"`
}

func idShapeRows(t *testing.T) []idShapeRow {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "idshape", "id_shapes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Rows []idShapeRow `json:"rows"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Rows) != 231 {
		t.Fatalf("id_shapes.json has %d rows, want 231", len(file.Rows))
	}
	return file.Rows
}

// DetectIDType gives every one of the 231 rows of the app's id_shapes.json
// the answer an item slot gives it there, the row's item_ref: an id is
// IDTypeUUIDv7, an item code IDTypeItemCode and an item_hash
// IDTypeItemHash, each with the row's canonical value; a typo is
// ErrItemCodeTypo; and every other row, a UUID of another version, short
// hex, text, an integer, the empty value and bytes that are not UTF-8, is
// ErrNotAnID. Mutants killed: a UUIDv7 read in the forms gofrs FromString
// takes, 32 hex characters, braces and urn:uuid: (rows 15, 16 and 19 are
// UUIDv7s); a strings.TrimSpace before classifying (rows 65, 66, 69 to
// 71, 160 to 164 and every row padded with a VT, FF or Unicode space are
// ids or codes); a 26-character Crockford value taken as an id (rows 194
// to 196); a UUID of any version taken (rows 7 to 14); a typo reported as
// ErrNotAnID (rows 50 to 60 and 192); a UUIDv7 not lowercased (rows 1, 2
// and 6 give the typed value).
func TestDetectIDType_GivesEveryIdShapeRowItsItemRef(t *testing.T) {
	counts := map[string]int{}
	for i, row := range idShapeRows(t) {
		raw, err := hex.DecodeString(row.Input)
		if err != nil {
			t.Fatalf("row %d (%s): input is not hex", i, row.About)
		}
		input := string(raw)
		counts[row.ItemRef]++
		shape, value, err := DetectIDType(input)
		want, _ := row.Value.(string)
		switch row.ItemRef {
		case "id":
			if err != nil || shape != IDTypeUUIDv7 || value != want {
				t.Errorf("row %d (%s): %q gives %q %q %v; want %q %q", i, row.About, input, shape, value, err, IDTypeUUIDv7, want)
			}
		case "item_code":
			if err != nil || shape != IDTypeItemCode || value != want {
				t.Errorf("row %d (%s): %q gives %q %q %v; want %q %q", i, row.About, input, shape, value, err, IDTypeItemCode, want)
			}
		case "item_hash":
			if err != nil || shape != IDTypeItemHash || value != want {
				t.Errorf("row %d (%s): %q gives %q %q %v; want %q %q", i, row.About, input, shape, value, err, IDTypeItemHash, want)
			}
		case "typo":
			if !errors.Is(err, ErrItemCodeTypo) || errors.Is(err, ErrNotAnID) || shape != "" || value != "" {
				t.Errorf("row %d (%s): %q gives %q %q %v; want ErrItemCodeTypo", i, row.About, input, shape, value, err)
			}
		case "invalid":
			if !errors.Is(err, ErrNotAnID) || errors.Is(err, ErrItemCodeTypo) || shape != "" || value != "" {
				t.Errorf("row %d (%s): %q gives %q %q %v; want ErrNotAnID", i, row.About, input, shape, value, err)
			}
		default:
			t.Fatalf("row %d (%s): item_ref %q is not one this test knows", i, row.About, row.ItemRef)
		}
	}
	want := map[string]int{"id": 7, "item_code": 15, "item_hash": 6, "typo": 12, "invalid": 191}
	for ref, n := range want {
		if counts[ref] != n {
			t.Errorf("%d rows with item_ref %q, want %d", counts[ref], ref, n)
		}
	}
}

// The refusals name the value as given, quoted, so a stray space or line
// feed shows, and say only what the value is: each command names what it
// takes. Mutants killed: the message built from a trimmed value (the
// quoted value loses its space); the generic refusal naming an item code
// or an item_hash as expected (the message differs).
func TestDetectIDType_RefusalsQuoteTheValueAsGiven(t *testing.T) {
	for input, want := range map[string]string{
		" 018e90d8-06e8-7f9f-bfd7-6730ba98a51b":  `invalid ID format " 018e90d8-06e8-7f9f-bfd7-6730ba98a51b"`,
		"018e90d8-06e8-7f9f-bfd7-6730ba98a51b\n": `invalid ID format "018e90d8-06e8-7f9f-bfd7-6730ba98a51b\n"`,
		"550e8400-e29b-41d4-a716-446655440000":   `"550e8400-e29b-41d4-a716-446655440000" is a UUID of another version`,
		"018e90d806e87f9fbfd76730ba98a51b":       `"018e90d806e87f9fbfd76730ba98a51b" is 32 hex characters, which is no id`,
		"9WBPKNV7SCP87VZX":                       `"9WBPKNV7SCP87VZX" has the shape of an item code but is not one: a symbol is mistyped or two are exchanged`,
	} {
		if _, _, err := DetectIDType(input); err == nil || err.Error() != want {
			t.Errorf("DetectIDType(%q) = %v; want %q", input, err, want)
		}
	}
}
