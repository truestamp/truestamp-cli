// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds step 7 of SPEC.md section 10.2, SHAPE, on P, the parsed
// value of proof. Its checks run in this order, and the first that fails
// is the step's one failing row:
//
//	| Check | Holds when                                                          | Else, at                                                  |
//	|-------|---------------------------------------------------------------------|-----------------------------------------------------------|
//	| 1     | format is "truestamp-proof"                                         | unsupported_format, /format                               |
//	| 2     | version is 1                                                        | unsupported_version, /version                             |
//	| 3     | P satisfies spec/schema.json ([ProofFailures])                      | shape_invalid, the first failure's part in document order |
//	| 4     | in an item, /witnesses has exactly the names of /subject/record/witness_hashes | shape_invalid, /witnesses                       |
//	| 5     | every TIME and NISTTIME value equals the formatting of its own parse | shape_invalid, the first such value in document order    |
//	| 6     | when constructions is present, its digest d is in CONSTRUCTIONS_ACCEPTED | construction_mismatch, /constructions                 |
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
// Check 4 asks whether /witnesses holds exactly the names of the item
// record's witness_hashes, which check 3 has held to the witness names,
// each with a HEX64 value. Check 5 reads the TIME values (header_time of
// each Bitcoin commitment entry, closed_at of each Stellar entropy object)
// and the NISTTIME values (timeStamp of each NIST entropy object) at the
// positions layout.go gives, after check 3 has held each to its pattern,
// and parses each through [ParseTime]. Of several that fail, the first in
// document order is the row ([FirstBy]).
//
// Constructions (SPEC.md section 7.2). Check 6 is the one place any step
// reads constructions (SPEC.md section 10.6, rule 1), and
// constructions_static_test.go (TestConstructions_ReadOnlyAtShape) holds
// the module to that one read. Its JSON type tells the mode: an object is
// the full text, whose d is SHA-256(JCS(value)), computed here; a string is
// a reference, whose d is the 64 hex characters of its URL, and check 3 has
// already failed a string off the reference pattern, and a value of any
// other type, with shape_invalid; absent is mode none, which checks
// nothing. d is compared with CONSTRUCTIONS_ACCEPTED
// (specdata.IsAccepted). The text is never interpreted and the URL never
// fetched.
//
// When the checks pass, SHAPE reports one row, pass, with part null, and
// the run takes the proof's type and subject id (SPEC.md section 10.5):
// /subject/record/id, or /block/record/id for block and beacon. A failure
// ends the run (SPEC.md section 10.2): each later step up to EXPECTED_HASH
// reports one row skip prior_failure, part null ([Run.EndRun]), the run's
// type and subject id stay "", and the attribution SIGNATURE gave the run
// stays. A run that has already ended passes through.
//
// Values (SPEC.md section 10.5). A row's values hold what this run
// computed or holds, and a value the proof carries only under declared, in
// a failing row:
//
//	| Row          | values                                                                   |
//	|--------------|--------------------------------------------------------------------------|
//	| pass         | constructions, the mode; in full mode also constructions_digest, the d it computed |
//	| check 1 or 2 | declared, the format or version P carries, whatever JSON value it is, when it carries one |
//	| check 3      | keyword and schema, the rule broken and where it sits in the schema       |
//	| check 4      | rule, which names the rule; never the names P carries                     |
//	| check 5      | class, TIME or NISTTIME, and declared, the value                          |
//	| check 6      | constructions, the mode; in full mode constructions_digest, the d it computed, and in reference mode declared, the d of the URL |
//
// A failing row's message never quotes a member name or a value P carries:
// the part already says where the failure is. The pass row's message names
// the proof's type, which check 3 has held to the type registry.
//
// Check 3's message leaves out the pointer, since a pointer can hold a
// member name P carries; check 5's keeps it, since its pointer is one of
// layout.go's positions and holds no name P chose.

package verifier

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// Format is the literal P's format must equal (SPEC.md section 3.2).
const Format = "truestamp-proof"

// Version is the one version this verifier reads (SPEC.md section 3.2).
const Version = 1

// The reference mode's URL, with d between these two (SPEC.md section 7.2).
const (
	constructionsReferencePrefix = "https://www.truestamp.com/.well-known/constructions-"
	constructionsReferenceSuffix = ".json"
)

// witnessRule is what a check-4 failure's values name: the rule, never the
// names the proof carries.
const witnessRule = "witnesses_named_by_witness_hashes"

// shapeFailure is the one failing row of SHAPE: its code, part, message
// and values.
type shapeFailure struct {
	code    specdata.Reason
	part    Pointer
	message string
	values  map[string]any
}

// Shape is step 7, SHAPE: checks 1 to 6, in order, on the run's P (see
// the file's doc).
func Shape(run *Run) {
	if run.Ended {
		return
	}
	for _, check := range []func(map[string]any) *shapeFailure{shapeFormat, shapeVersion, shapeSchema, shapeWitnesses, shapeTimes} {
		if f := check(run.Proof); f != nil {
			shapeFail(run, f)
			return
		}
	}
	constructions, f := shapeConstructions(run.Proof)
	if f != nil {
		shapeFail(run, f)
		return
	}
	run.Type = run.Proof["type"].(string)
	idAt := Pointer{Name("subject"), Name("record"), Name("id")}
	if run.Type == "block" || run.Type == "beacon" {
		idAt = Pointer{Name("block"), Name("record"), Name("id")}
	}
	run.SubjectID = valueAt(run.Proof, idAt).(string)
	run.AddRow(PassRow("SHAPE", nil, fmt.Sprintf("The proof has the shape its type, %s, gives it.", run.Type), constructions))
}

// shapeFail adds SHAPE's failing row f and ends the run.
func shapeFail(run *Run, f *shapeFailure) {
	run.AddRow(NewRow("SHAPE", f.code, Part(f.part.Encode()), f.message, f.values))
	run.EndRun()
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

// shapeWitnesses is check 4: in an item, /witnesses has exactly the names
// of the record's witness_hashes.
func shapeWitnesses(proof map[string]any) *shapeFailure {
	if proof["type"] != "item" {
		return nil
	}
	witnesses := proof["witnesses"].(map[string]any)
	names := valueAt(proof, Pointer{Name("subject"), Name("record"), Name("witness_hashes")}).(map[string]any)
	exact := len(witnesses) == len(names)
	for name := range names {
		if _, ok := witnesses[name]; !ok {
			exact = false
		}
	}
	if exact {
		return nil
	}
	return &shapeFailure{
		code:    specdata.ReasonShapeInvalid,
		part:    Pointer{Name("witnesses")},
		message: "The item's witnesses are not exactly the ones its record's witness hashes name.",
		values:  map[string]any{"rule": witnessRule},
	}
}

// timePosition is a TIME or NISTTIME value's position in P and its class.
type timePosition struct {
	at    Pointer
	class TimeClass
}

// timePositions returns where P holds a TIME or NISTTIME value: the
// header_time of each Bitcoin commitment entry, the closed_at of each
// Stellar entropy object and the timeStamp of each NIST entropy object.
func timePositions(proof map[string]any) []timePosition {
	var out []timePosition
	for _, entry := range commitmentEntries(proof) {
		if entry.name == ChainBitcoin {
			out = append(out, timePosition{with(entry.at, Name("header_time")), ClassTIME})
		}
	}
	for _, object := range entropyObjects(proof) {
		switch object.name {
		case ChainStellar:
			out = append(out, timePosition{append(append(Pointer{}, object.at...), Name("payload"), Name("closed_at")), ClassTIME})
		case "nist":
			out = append(out, timePosition{append(append(Pointer{}, object.at...), Name("payload"), Name("pulse"), Name("timeStamp")), ClassNISTTIME})
		}
	}
	return out
}

// shapeTimes is check 5: every TIME and NISTTIME value equals the
// formatting of its own parse.
func shapeTimes(proof map[string]any) *shapeFailure {
	var failing []timePosition
	for _, p := range timePositions(proof) {
		if _, err := ParseTime(p.class, valueAt(proof, p.at)); err != nil {
			failing = append(failing, p)
		}
	}
	first, ok := FirstBy(failing, func(p timePosition) Pointer { return p.at })
	if !ok {
		return nil
	}
	return &shapeFailure{
		code: specdata.ReasonShapeInvalid,
		part: first.at,
		message: fmt.Sprintf("The time at %s names no instant: it does not write back to itself as %s.",
			first.at.Encode(), first.class),
		values: map[string]any{"class": first.class.String(), "declared": valueAt(proof, first.at)},
	}
}

// shapeConstructions is check 6, and the only read of constructions in any
// step (SPEC.md sections 7.2 and 10.6, rule 1): the pass row's values, or
// the failure. d of a full text is a content digest, the SHA-256 of its JCS
// with no prefix byte (SPEC.md sections 4.1 and 7.2), so this is one of the
// two functions of the new verifier outside internal/registry that
// internal/registry/rule2_test.go lets use crypto/sha256.
func shapeConstructions(proof map[string]any) (map[string]any, *shapeFailure) {
	value, present := proof["constructions"]
	if !present {
		return map[string]any{"constructions": "none"}, nil
	}
	switch text := value.(type) {
	case map[string]any:
		// Full mode: this run computed d, so the row gives it, pass or fail.
		encoded, err := jcs.Encode(text)
		if err != nil {
			panic(fmt.Sprintf("verifier: SIGNATURE encoded all of P, and its constructions text does not encode: %v", err))
		}
		sum := sha256.Sum256(encoded)
		digest := hex.EncodeToString(sum[:])
		values := map[string]any{"constructions": "full", "constructions_digest": digest}
		if specdata.IsAccepted(digest) {
			return values, nil
		}
		return nil, constructionMismatch("This proof carries a constructions text this verifier has not accepted, "+
			"so it cannot vouch for the proof's explanation of itself.", values)
	case string:
		// Reference mode: d is the proof's, so only a failure gives it,
		// marked as declared (SPEC.md section 10.5). Check 3 has held the
		// string to the reference pattern, so d is the 64 characters
		// between the prefix and the suffix.
		digest, prefixed := strings.CutPrefix(text, constructionsReferencePrefix)
		digest, suffixed := strings.CutSuffix(digest, constructionsReferenceSuffix)
		if !prefixed || !suffixed || len(digest) != 64 {
			panic("verifier: check 3 passed a constructions reference off its pattern")
		}
		if specdata.IsAccepted(digest) {
			return map[string]any{"constructions": "reference"}, nil
		}
		return nil, constructionMismatch("This proof names by reference a constructions text this verifier has not accepted, "+
			"so it cannot vouch for the proof's explanation of itself.",
			map[string]any{"constructions": "reference", "declared": digest})
	}
	panic("verifier: check 3 passed constructions that are neither an object nor a string")
}

// constructionMismatch is check 6's failure, at /constructions.
func constructionMismatch(message string, values map[string]any) *shapeFailure {
	return &shapeFailure{code: specdata.ReasonConstructionMismatch, part: Pointer{Name("constructions")}, message: message, values: values}
}
