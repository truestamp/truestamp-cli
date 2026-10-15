// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports checks 1 to 3 of the reference verifier's
// Truestamp.Verifier.Shape (elixir/lib/truestamp/verifier/shape.ex in
// truestamp-verification), checked against the Truestamp application's
// port, Truestamp.Proof.Verifier.Shape (lib/truestamp/proof/verifier/shape.ex
// in truestamp-v2): step 7 of SPEC.md section 10.2, SHAPE, on P, the parsed
// value of proof. Its checks run in this order, and the first that fails is
// the step's one failing row:
//
//	| Check | Holds when                                            | Else, at                                          |
//	|-------|-------------------------------------------------------|---------------------------------------------------|
//	| 1     | format is "truestamp-proof"                           | unsupported_format, /format                       |
//	| 2     | version is 1                                          | unsupported_version, /version                     |
//	| 3     | P satisfies spec/schema.json ([ProofFailures])        | shape_invalid, the first failure's part in document order |
//
// This file makes checks 1 to 3 of SHAPE's six; checks 4 to 6 (the
// witnesses of an item, the TIME and NISTTIME round trips and the
// constructions digest) are not in it.
//
// Checks 1 and 2 come first, so an absent format or version, or a format
// with a \n on its end, is that check's failure, not the schema's. The
// literals are compared exactly: format is a string equal to
// "truestamp-proof", and version a number equal to 1, which 1.0 and 1e0
// also read as (SPEC.md section 10.1, J5).
//
// Check 3's part is the pointer of the member that is missing, extra or
// forbidden for the type, or else of the value that breaks its rule (for a
// count, the object or array). Of several failures, the one first in
// document order is the row. A block record that breaks the empty-root
// rule of SPEC.md section 5.4, at /block, /witnesses/block or
// /signing_key_event/block, fails at its merkle_root, and one that breaks
// the genesis pairing of section 3.3 at its key event's type, or at the
// missing metadata.key_event when genesis stands alone: the schema places
// those failures there, in its BLOCK_RECORD and METADATA definitions, and
// the checker reports them where the schema places them. The key-event
// member sets, and the events allowed at /signing_key_event/block, come
// only from the schema; the checks of a key's bytes belong to DERIVE, so a
// key event whose public_key is a point of small order passes SHAPE.
//
// When the checks pass, SHAPE reports one row, pass, with part null, and
// the run takes the proof's type and subject id (SPEC.md section 10.5):
// /subject/record/id, or /block/record/id for block and beacon. A failure
// ends the run (SPEC.md section 10.2): each later step up to EXPECTED_HASH
// reports one row skip prior_failure, part null ([Run.EndRun]), and the
// attribution SIGNATURE gave the run stays. A run that has already ended
// passes through.
//
// Values (SPEC.md section 10.5). A failing row's values hold, for check 1
// or 2, declared, the format or version P carries, whatever JSON value it
// is, when it carries one; for check 3, keyword and schema, the rule broken
// and where it sits in the schema. A failing row's message never quotes a
// member name or a value P carries: the part already says where the
// failure is. The pass row's message names the proof's type, which check 3
// has held to the type registry.
//
// Adapted from the reference: check 3's message leaves out the pointer, as
// the application's port does, since a pointer can hold a member name P
// carries.

package verifier

import (
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// Format is the literal P's format must equal (SPEC.md section 3.2).
const Format = "truestamp-proof"

// Version is the one version this verifier reads (SPEC.md section 3.2).
const Version = 1

// shapeFailure is the one failing row of SHAPE: its code, part, message
// and values.
type shapeFailure struct {
	code    specdata.Reason
	part    Pointer
	message string
	values  map[string]any
}

// Shape is step 7, SHAPE: checks 1 to 3, in order, on the run's P (see the
// file's doc).
func Shape(run *Run) {
	if run.Ended {
		return
	}
	for _, check := range []func(map[string]any) *shapeFailure{shapeFormat, shapeVersion, shapeSchema} {
		if f := check(run.Proof); f != nil {
			run.AddRow(NewRow("SHAPE", f.code, Part(f.part.Encode()), f.message, f.values))
			run.EndRun()
			return
		}
	}
	run.Type = run.Proof["type"].(string)
	idAt := Pointer{Name("subject"), Name("record"), Name("id")}
	if run.Type == "block" || run.Type == "beacon" {
		idAt = Pointer{Name("block"), Name("record"), Name("id")}
	}
	run.SubjectID = valueAt(run.Proof, idAt).(string)
	run.AddRow(PassRow("SHAPE", nil, fmt.Sprintf("The proof has the shape its type, %s, gives it.", run.Type), nil))
}

// shapeFormat is check 1.
func shapeFormat(proof map[string]any) *shapeFailure {
	if format, ok := proof["format"].(string); ok && format == Format {
		return nil
	}
	return &shapeFailure{
		code:    specdata.ReasonUnsupportedFormat,
		part:    Pointer{Name("format")},
		message: fmt.Sprintf("This proof's format is not %q, the only format this verifier reads.", Format),
		values:  declared(proof, "format"),
	}
}

// shapeVersion is check 2.
func shapeVersion(proof map[string]any) *shapeFailure {
	if version, ok := proof["version"].(strictjson.Number); ok && version == Version {
		return nil
	}
	return &shapeFailure{
		code: specdata.ReasonUnsupportedVersion,
		part: Pointer{Name("version")},
		message: fmt.Sprintf("This proof's version is not %d, the only version this verifier reads; "+
			"a later format may need a later verifier.", Version),
		values: declared(proof, "version"),
	}
}

// declared returns the values of a failing check 1 or 2: declared, the
// member's value, when P carries the member, null included.
func declared(proof map[string]any, member string) map[string]any {
	if value, present := proof[member]; present {
		return map[string]any{"declared": value}
	}
	return map[string]any{}
}

// shapeSchema is check 3.
func shapeSchema(proof map[string]any) *shapeFailure {
	failures := ProofFailures(proof)
	if len(failures) == 0 {
		return nil
	}
	first := failures[0]
	return &shapeFailure{
		code: specdata.ReasonShapeInvalid,
		part: first.Part,
		message: fmt.Sprintf("This proof does not have the layout of a Truestamp proof: it breaks the rule %s "+
			"of the proof schema.", first.Schema),
		values: map[string]any{"keyword": first.Keyword, "schema": first.Schema},
	}
}
