// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package ids_test

import (
	"encoding/hex"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/ids"
	"github.com/truestamp/truestamp-cli/internal/proof"
)

// ValidateUUIDv7 and DetectIDType read a UUIDv7 by one rule: on every one
// of the 231 rows of the app's id_shapes.json (testdata/idshape, its
// SOURCE names the commit), ValidateUUIDv7 takes exactly the rows
// DetectIDType gives IDTypeUUIDv7, which are the rows whose item_ref is
// id. Mutants killed: ValidateUUIDv7 reading a UUID by gofrs FromString
// (rows 15, 16 and 19, the 32-hex, braced and urn: forms, are taken); a
// strings.TrimSpace before the check (the padded rows are taken); the
// version or variant check removed (rows 7 to 14 are taken).
func TestValidateUUIDv7_IsDetectIDTypesRule(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "idshape", "id_shapes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Rows []struct {
			About   string `json:"about"`
			Input   string `json:"input"`
			ItemRef string `json:"item_ref"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Rows) != 231 {
		t.Fatalf("id_shapes.json has %d rows, want 231", len(file.Rows))
	}
	taken := 0
	for i, row := range file.Rows {
		raw, err := hex.DecodeString(row.Input)
		if err != nil {
			t.Fatalf("row %d (%s): input is not hex", i, row.About)
		}
		input := string(raw)
		shape, _, _ := proof.DetectIDType(input)
		ok := ids.ValidateUUIDv7(input) == nil
		if ok != (shape == proof.IDTypeUUIDv7) || ok != (row.ItemRef == "id") || ok != ids.IsUUIDv7Text(input) {
			t.Errorf("row %d (%s): %q: ValidateUUIDv7 takes it %v, DetectIDType gives %q, item_ref %q", i, row.About, input, ok, shape, row.ItemRef)
		}
		if ok {
			taken++
		}
	}
	if taken != 7 {
		t.Errorf("%d rows taken, want 7", taken)
	}
}

// The refusal quotes the value as given and names the one form taken.
// Mutant killed: the message built from a trimmed value (the quoted value
// loses its space).
func TestValidateUUIDv7_RefusalQuotesTheValueAsGiven(t *testing.T) {
	const want = `invalid UUID " 019d6a32-13e6-72b0-97e5-3779231ea97b": an id is the 36-character text of a UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)`
	if err := ids.ValidateUUIDv7(" 019d6a32-13e6-72b0-97e5-3779231ea97b"); err == nil || err.Error() != want {
		t.Errorf("ValidateUUIDv7 = %v; want %q", err, want)
	}
}
