// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"bytes"
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// Step 7, SHAPE, checks 1 to 3 (SPEC.md section 10.2), on mutants of the
// 11 positive proofs, each signed again by a key of its own and run whole
// through the gate keyless under E_v test, then SHAPE. The vector cases
// are internal/vectorcases' runner's.

// shapeSigner signs every mutant.
var shapeSigner = signerOf("SHAPE's mutants")

// emptyRoot is the root of the empty tree, SHA-256 of zero bytes (SPEC.md
// section 5.4).
const emptyRoot = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// blockPositions are the positions of a block node (SPEC.md section 5.4).
var blockPositions = []Pointer{
	{Name("block")},
	{Name("witnesses"), Name("block")},
	{Name("signing_key_event"), Name("block")},
}

// positiveProofs returns P of each positive proof, by name.
func positiveProofs(t *testing.T) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for name, document := range positiveDocuments(t) {
		out[name] = document["proof"].(map[string]any)
	}
	return out
}

// sortedProofs returns positiveProofs' names in order.
func sortedProofs(proofs map[string]map[string]any) []string {
	names := make([]string, 0, len(proofs))
	for name := range proofs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// shapeRun signs proof, runs the gate over it keyless under E_v test and
// then SHAPE, and returns the run; the gate must pass.
func shapeRun(t *testing.T, proof map[string]any) *Run {
	t.Helper()
	cfg, err := NewConfig("test")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Keyring = KeyringSource{Kind: SourceKeyless}
	run := gate(t, cfg, signed(t, proof, shapeSigner))
	if run.Ended || run.Attribution != AttributionUnattributed {
		t.Fatalf("the gate: %s", rowSummary(run.Rows))
	}
	Shape(run)
	return run
}

// shapeOutcome returns SHAPE's row of proof as rowSummary writes it.
func shapeOutcome(t *testing.T, proof map[string]any) string {
	t.Helper()
	run := shapeRun(t, proof)
	for _, row := range run.Rows {
		if row.Step == "SHAPE" {
			return rowSummary([]Row{row})
		}
	}
	t.Fatalf("no SHAPE row: %s", rowSummary(run.Rows))
	return ""
}

func failsAt(code specdata.Reason, at Pointer) string {
	return fmt.Sprintf("[%s %s fail SHAPE]", code, at.Encode())
}

const shapePasses = "[null null pass SHAPE]"

// clone returns a deep copy of a parsed value.
func clone(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for name, x := range v {
			out[name] = clone(x)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = clone(x)
		}
		return out
	}
	return value
}

// put returns a copy of proof with the value at p replaced by value, or
// added where p's parent is an object that lacks it.
func put(proof map[string]any, p Pointer, value any) map[string]any {
	out := clone(proof).(map[string]any)
	var parent any = out
	for _, s := range p[:len(p)-1] {
		parent = valueAt(parent, Pointer{s})
	}
	last := p[len(p)-1]
	switch v := parent.(type) {
	case map[string]any:
		v[last.name] = value
	case []any:
		v[last.index] = value
	default:
		panic("put: no parent at " + p.Encode())
	}
	return out
}

// remove returns a copy of proof without the member at p.
func remove(proof map[string]any, p Pointer) map[string]any {
	out := clone(proof).(map[string]any)
	parent := valueAt(out, p[:len(p)-1]).(map[string]any)
	delete(parent, p[len(p)-1].name)
	return out
}

// proofNode is a value of P at its pointer.
type proofNode struct {
	at    Pointer
	value any
}

// nodes returns every value of proof with its pointer, in document order;
// of subject.claims, the open region, and of constructions, which no check
// here interprets, only the member itself.
func nodes(proof map[string]any) []proofNode {
	var out []proofNode
	var walk func(at Pointer, value any)
	walk = func(at Pointer, value any) {
		out = append(out, proofNode{at, value})
		if at.Encode() == "/subject/claims" || at.Encode() == "/constructions" {
			return
		}
		switch v := value.(type) {
		case map[string]any:
			for _, name := range sortedNames(v) {
				walk(with(at, Name(name)), v[name])
			}
		case []any:
			for i, x := range v {
				walk(with(at, Index(i)), x)
			}
		}
	}
	walk(Pointer{}, proof)
	return out
}

// classOf names the pattern class of SPEC.md section 2.4 text matches,
// by the spec schema's $defs, or "" for none.
func classOf(t *testing.T, text string) string {
	t.Helper()
	for _, name := range []string{"HEX64", "KID", "B64U32", "B64U64", "TIME", "NISTTIME", "NISTHEX", "UUID7", "HEXBYTES", "ASCII16", "ASCII128"} {
		if len(proofSchema.Failures(text, "$defs", name)) == 0 {
			return name
		}
	}
	return ""
}

// SPEC.md section 13.3, SHAPE: a \n on the end of every string of P but
// environment, of a pattern class or a literal, fails at that value,
// format with unsupported_format and every other with shape_invalid;
// every pattern class of SPEC.md section 2.4 is among them. The schema pairs each pattern
// with an exact length or the newline guard (SPEC.md section 9), so a
// pattern whose $ matched before a final newline would still fail each of
// these. Mutants killed: jsonEqual comparing strings after trimming space
// (a type, network or source with a newline passes its const or enum);
// shapeFormat trimming the format (truestamp-proof\n passes).
func TestShape_ANewlineOnAnyStringFailsAtIt(t *testing.T) {
	proofs := positiveProofs(t)
	classes, literals := map[string]int{}, map[string]int{}
	tested := 0
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		for _, n := range nodes(proof) {
			text, ok := n.value.(string)
			if !ok || n.at.Encode() == "/environment" {
				continue
			}
			tested++
			classes[classOf(t, text)]++
			literals[n.at[len(n.at)-1].name]++
			code := specdata.ReasonShapeInvalid
			if n.at.Encode() == "/format" {
				code = specdata.ReasonUnsupportedFormat
			}
			if got, want := shapeOutcome(t, put(proof, n.at, text+"\n")), failsAt(code, n.at); got != want {
				t.Errorf("%s %s: %s, want %s", name, n.at.Encode(), got, want)
			}
		}
	}
	for _, class := range []string{"HEX64", "KID", "B64U32", "B64U64", "TIME", "NISTTIME", "NISTHEX", "UUID7", "HEXBYTES", "ASCII16", "ASCII128"} {
		if classes[class] == 0 {
			t.Errorf("no string of the class %q: %v", class, classes)
		}
	}
	if classes["HEX64"] < 300 || tested < 600 || literals["format"] != 11 || literals["type"] < 11 ||
		literals["network"] == 0 || literals["source"] == 0 || literals["alg"] != 0 {
		t.Errorf("%d strings, by class %v, by member %v", tested, classes, literals)
	}
}

// SPEC.md section 13.3, SHAPE: every OPTIONAL member but environment, given
// as null, fails at that member. A member is OPTIONAL where P with it
// deleted still satisfies the schema. Mutants killed: type compared as a Go value
// that holds nil for null (a null passes every type); the schema's
// properties skipping a member whose value is null (each passes).
func TestShape_NullForAnyOptionalMemberFailsAtIt(t *testing.T) {
	proofs := positiveProofs(t)
	reached := map[string]bool{}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		for _, n := range nodes(proof) {
			if len(n.at) == 0 || n.at[len(n.at)-1].isIndex || n.at.Encode() == "/environment" {
				continue
			}
			if len(ProofFailures(remove(proof, n.at))) != 0 {
				continue
			}
			reached[n.at[len(n.at)-1].name] = true
			if got, want := shapeOutcome(t, put(proof, n.at, nil)), failsAt(specdata.ReasonShapeInvalid, n.at); got != want {
				t.Errorf("%s %s: %s, want %s", name, n.at.Encode(), got, want)
			}
		}
	}
	for _, member := range []string{"constructions", "signing_key_event", "entropy_bitcoin", "entropy_nist",
		"entropy_stellar", "bitcoin", "stellar", "key_event", "prerotation_commitment"} {
		if !reached[member] {
			t.Errorf("no OPTIONAL member %s among %v", member, reached)
		}
	}
}

// The empty-root rule (SPEC.md section 5.4): at each of /block,
// /witnesses/block and /signing_key_event/block, a block record with
// leaves under the empty root, and one with none under another root, fail
// shape_invalid at the record's merkle_root, by the schema's BLOCK_RECORD;
// and an empty tree, leaf_count 0 under the empty root, passes at every
// position. Mutants killed: the schema's if applying then whatever its
// schema gives (the leaves mutants pass); if applying else whatever (the
// empty trees fail); jsonEqual true for every two strings (the no-leaves
// mutants pass).
func TestShape_TheEmptyRootRuleAtEveryBlockPosition(t *testing.T) {
	proofs := positiveProofs(t)
	covered := map[string]bool{}
	empty := map[string]bool{}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		for _, position := range blockPositions {
			record := with(position, Name("record"))
			if _, ok := valueAt(proof, record).(map[string]any); !ok {
				continue
			}
			if count, _ := valueAt(proof, with(record, Name("leaf_count"))).(strictjson.Number); count == 0 {
				if valueAt(proof, with(record, Name("merkle_root"))) != emptyRoot {
					t.Fatalf("%s %s: an empty tree under another root", name, position.Encode())
				}
				if got := shapeOutcome(t, proof); got != shapePasses {
					t.Errorf("%s %s: the empty tree: %s", name, position.Encode(), got)
				}
				empty[position.Encode()] = true
			}
			leaves := put(put(proof, with(record, Name("leaf_count")), strictjson.Number(1)), with(record, Name("merkle_root")), emptyRoot)
			noLeaves := put(put(proof, with(record, Name("leaf_count")), strictjson.Number(0)), with(record, Name("merkle_root")), strings.Repeat("ab", 32))
			want := failsAt(specdata.ReasonShapeInvalid, with(record, Name("merkle_root")))
			for kind, mutant := range map[string]map[string]any{"leaves": leaves, "no leaves": noLeaves} {
				if got := shapeOutcome(t, mutant); got != want {
					t.Errorf("%s %s, %s: %s, want %s", name, position.Encode(), kind, got, want)
				}
				covered[position.Encode()+" "+kind] = true
			}
		}
	}
	if len(covered) != 6 || len(empty) != 3 {
		t.Fatalf("covered %v, empty trees at %v", covered, empty)
	}
}

// Of several failures of check 3, the first in document order wins: a
// prefix before what it holds, members in JCS order (an uppercase name
// before every lowercase one), elements by index and never as text, and a
// member P lacks in its place. Mutants killed: the failures left in the
// order the checker meets them; names compared after folding case (Zeta
// comes after height); indices compared as text (10 before 2).
func TestShape_TheFirstFailureInDocumentOrderWins(t *testing.T) {
	item := positiveProofs(t)["item"]
	path := Pointer{Name("subject"), Name("inclusion_proof"), Name("path")}
	nodes := func(first any, n int) []any {
		out := []any{first}
		for range n {
			out = append(out, emptyRoot)
		}
		return out
	}
	record := func(member string) Pointer { return Pointer{Name("block"), Name("record"), Name(member)} }
	stretched := slices.Clone(valueAt(item, path).([]any))
	for len(stretched) < 11 {
		stretched = append(stretched, emptyRoot)
	}
	stretched = stretched[:11]
	stretched[2], stretched[10] = "x", "x"
	cases := []struct {
		name   string
		mutant map[string]any
		want   Pointer
	}{
		{"a count at its array, before its first node", put(item, path, nodes("x", 32)), path},
		{"with 32 nodes, the first node", put(item, path, nodes("x", 31)), with(path, Index(0))},
		{"members in JCS order", put(put(put(item, Pointer{Name("witnesses"), Name("block"), Name("record"), Name("id")}, "x"),
			record("signing_key_id"), "x"), Pointer{Name("commitments"), Name("stellar"), Name("ledger")}, strictjson.Number(-1)),
			record("signing_key_id")},
		{"an uppercase name first", put(put(item, record("Zeta"), strictjson.Number(1)), record("height"), strictjson.Number(-1)), record("Zeta")},
		{"a lowercase name in its place", put(put(item, record("zeta"), strictjson.Number(1)), record("height"), strictjson.Number(-1)), record("height")},
		{"elements by index", put(item, path, stretched), with(path, Index(2))},
		{"a missing member after one that fails", put(remove(item, record("previous_block_hash")), record("merkle_root"), "x"), record("merkle_root")},
		{"a missing member before one that fails", put(remove(item, record("leaf_count")), record("merkle_root"), "x"), record("leaf_count")},
	}
	for _, c := range cases {
		if got, want := shapeOutcome(t, c.mutant), failsAt(specdata.ReasonShapeInvalid, c.want); got != want {
			t.Errorf("%s: %s, want %s", c.name, got, want)
		}
	}
}

// Checks 1 and 2 come before the schema, absent or wrong, format before
// version, and a version written 1.0 is 1. Mutants killed: checks 1 and 2
// made after the schema (an absent format fails shape_invalid); version
// compared with the string "1" (the sound proof fails, and "1" passes);
// shapeFormat comparing the format after trimming space (the format with
// a trailing newline passes); format compared folding case
// (Truestamp-proof passes).
func TestShape_ChecksOneAndTwoComeFirst(t *testing.T) {
	item := positiveProofs(t)["item"]
	bad := put(item, Pointer{Name("block"), Name("record"), Name("id")}, "x")
	format, version := Pointer{Name("format")}, Pointer{Name("version")}
	cases := []struct {
		mutant map[string]any
		want   string
	}{
		{remove(bad, format), failsAt(specdata.ReasonUnsupportedFormat, format)},
		{put(bad, format, Format+"\n"), failsAt(specdata.ReasonUnsupportedFormat, format)},
		{put(bad, format, "Truestamp-proof"), failsAt(specdata.ReasonUnsupportedFormat, format)},
		{put(bad, format, nil), failsAt(specdata.ReasonUnsupportedFormat, format)},
		{remove(bad, version), failsAt(specdata.ReasonUnsupportedVersion, version)},
		{put(bad, version, strictjson.Number(2)), failsAt(specdata.ReasonUnsupportedVersion, version)},
		{put(bad, version, "1"), failsAt(specdata.ReasonUnsupportedVersion, version)},
		{put(bad, version, strictjson.Number(1.5)), failsAt(specdata.ReasonUnsupportedVersion, version)},
		{put(put(bad, format, "x"), version, strictjson.Number(2)), failsAt(specdata.ReasonUnsupportedFormat, format)},
		{bad, failsAt(specdata.ReasonShapeInvalid, Pointer{Name("block"), Name("record"), Name("id")})},
		{item, shapePasses},
	}
	for i, c := range cases {
		if got := shapeOutcome(t, c.mutant); got != c.want {
			t.Errorf("case %d: %s, want %s", i, got, c.want)
		}
	}
	data := signed(t, item, shapeSigner)
	spelled := bytes.Replace(data, []byte(`"version":1`), []byte(`"version":1.0`), 1)
	if bytes.Equal(spelled, data) {
		t.Fatal("no version member in the signed bytes")
	}
	cfg, err := NewConfig("test")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Keyring = KeyringSource{Kind: SourceKeyless}
	run := gate(t, cfg, spelled)
	Shape(run)
	if got := rowSummary(run.Rows[7:]); got != shapePasses {
		t.Fatalf("version written 1.0: %s", got)
	}
}

// A pass gives one row, part null, whose values name the constructions
// mode (each positive proof is in full mode, so with the digest it
// computed) and nothing else, with a message naming the type, and the run
// takes the type and subject id: /subject/record/id,
// or /block/record/id for block and beacon. A failure ends the run, its
// later steps skip prior_failure, part null, up to EXPECTED_HASH, and
// leaves the attribution, trust_root, type and subject id as they were.
// An ended run passes through. Mutants killed: Shape not ending the run
// (no skip rows); the subject id read from /subject/record/id for every
// type (the block and beacon proofs panic); a failure setting the
// attribution none; Shape running over an ended run (a second SHAPE row);
// the pass row with no values (no mode named).
func TestShape_TheRunAndItsRows(t *testing.T) {
	proofs := positiveProofs(t)
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		run := shapeRun(t, proof)
		typ := proof["type"].(string)
		holder := "subject"
		if typ == "block" || typ == "beacon" {
			holder = "block"
		}
		id := valueAt(proof, Pointer{Name(holder), Name("record"), Name("id")})
		row := run.Rows[len(run.Rows)-1]
		values := string(encode(t, row.Values))
		wantValues := `{"constructions":"full","constructions_digest":"` + specdata.Accepted()[0] + `"}`
		if run.Ended || run.Type != typ || run.SubjectID != id || row.Step != "SHAPE" || row.Status != StatusPass ||
			row.Part != nil || row.Code != "" || values != wantValues || !strings.Contains(row.Message, ", "+typ+",") {
			t.Errorf("%s: type %q, subject id %q, ended %v, row %+v", name, run.Type, run.SubjectID, run.Ended, row)
		}
	}
	item := proofs["item"]
	run := shapeRun(t, put(item, Pointer{Name("block"), Name("record"), Name("id")}, "x"))
	var later []string
	for _, step := range []string{"POLICY", "DERIVE", "CHAIN", "SOURCES", "WINDOW", "EXPECTED_HASH"} {
		later = append(later, "[prior_failure null skip "+step+"]")
	}
	want := "[shape_invalid /block/record/id fail SHAPE] " + strings.Join(later, " ")
	if got := rowSummary(run.Rows[7:]); got != want || !run.Ended || run.Attribution != AttributionUnattributed ||
		run.TrustRoot.Kind != KindNone || run.Type != "" || run.SubjectID != "" {
		t.Fatalf("rows %s, ended %v, attribution %q, trust root %+v, type %q, subject %q",
			got, run.Ended, run.Attribution, run.TrustRoot, run.Type, run.SubjectID)
	}
	rows := slices.Clone(run.Rows)
	Shape(run)
	if len(run.Rows) != len(rows) {
		t.Fatalf("Shape over an ended run added %d rows", len(run.Rows)-len(rows))
	}
}

// A failing row's values: declared, the format or version P carries,
// whatever JSON value it is, when it carries one; for check 3 the keyword
// and where it sits. Its message never quotes a member name or a value P
// carries. Mutants killed: declared left out for a null version; the
// message of check 3 naming the part (Zeta shows); check 1's message
// quoting the declared format.
func TestShape_AFailingRowsValuesAndMessage(t *testing.T) {
	item := positiveProofs(t)["item"]
	rowOf := func(proof map[string]any) Row { return shapeRun(t, proof).Rows[7] }
	cases := []struct {
		proof  map[string]any
		values string
		quote  string
	}{
		{put(item, Pointer{Name("version")}, strictjson.Number(7)), `{"declared":7}`, "7"},
		{put(item, Pointer{Name("version")}, nil), `{"declared":null}`, "null"},
		{remove(item, Pointer{Name("format")}), `{}`, ""},
		{put(item, Pointer{Name("format")}, "forged-format"), `{"declared":"forged-format"}`, "forged-format"},
		{put(item, Pointer{Name("block"), Name("record"), Name("Zeta")}, strictjson.Number(1)),
			`{"keyword":"additionalProperties","schema":"#/$defs/BLOCK_RECORD/additionalProperties"}`, "Zeta"},
		{put(item, Pointer{Name("block"), Name("record"), Name("merkle_root")}, "forged-root"),
			`{"keyword":"minLength","schema":"#/$defs/HEX64/minLength"}`, "forged-root"},
	}
	for _, c := range cases {
		row := rowOf(c.proof)
		if got := string(encode(t, row.Values)); got != c.values {
			t.Errorf("values %s, want %s", got, c.values)
		}
		if c.quote != "" && strings.Contains(row.Message, c.quote) {
			t.Errorf("the message quotes %q: %s", c.quote, row.Message)
		}
	}
}

// A key event whose public_key or previous_public_key is a point of small
// order, a value of its class, passes SHAPE: its key's bytes are DERIVE's
// to check. Mutant killed: SHAPE checking a key event's key bytes (each
// mutant fails).
func TestShape_ASmallOrderKeyPassesShape(t *testing.T) {
	proofs := positiveProofs(t)
	keys := specdata.SmallOrderKeys()
	tested := map[string]bool{}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		for _, n := range nodes(proof) {
			if len(n.at) == 0 {
				continue
			}
			if last := n.at[len(n.at)-1]; last.isIndex || (last.name != "public_key" && last.name != "previous_public_key") {
				continue
			}
			for _, key := range keys {
				if got := shapeOutcome(t, put(proof, n.at, registry.EncodeB64U(key.Key[:]))); got != shapePasses {
					t.Errorf("%s %s: %s", name, n.at.Encode(), got)
				}
			}
			tested[n.at[0].name] = true
		}
	}
	if !tested["block"] || !tested["signing_key_event"] || len(keys) < 8 {
		t.Fatalf("positions %v, %d keys", tested, len(keys))
	}
}

// A member the proof's type forbids fails at that member, whether it holds
// a value or null; a string at ASCII128's cap of 128 code points passes
// and one over fails, as does a non-ASCII string at the cap; and an
// integer at MAX_INTEGER, the schema's inclusive maximum, passes, as -1
// and 1.5 do not. Mutants killed: the false schema passing every value
// (each forbidden member passes, the schema of its position for the type
// being false); the false schema passing null (a block or a beacon with a
// null subject passes, as nothing but the false schema holds subject to
// absent for those types); maxLength made strict
// (128 code points fail); maximum made strict (MAX_INTEGER fails); integer
// accepting a fraction (1.5 passes).
func TestShape_TypesCapsAndBounds(t *testing.T) {
	proofs := positiveProofs(t)
	item, beacon, block := proofs["item"], proofs["beacon"], proofs["block"]
	for _, c := range []struct {
		proof  map[string]any
		member string
		value  any
	}{
		{item, "observations", beacon["observations"]},
		{beacon, "subject", item["subject"]},
		{beacon, "witnesses", item["witnesses"]},
		{block, "witnesses", item["witnesses"]},
		{block, "subject", item["subject"]},
		{block, "observations", beacon["observations"]},
	} {
		if _, present := c.proof[c.member]; present || c.value == nil {
			t.Fatalf("%s with %s", c.proof["type"], c.member)
		}
		for held, value := range map[string]any{"a value": c.value, "null": nil} {
			if got, want := shapeOutcome(t, put(c.proof, Pointer{Name(c.member)}, value)), failsAt(specdata.ReasonShapeInvalid, Pointer{Name(c.member)}); got != want {
				t.Errorf("%s with %s holding %s: %s, want %s", c.proof["type"], c.member, held, got, want)
			}
		}
	}
	rotation := proofs["block-emergency-rotation"]
	reason := Pointer{Name("block"), Name("record"), Name("metadata"), Name("key_event"), Name("reason")}
	if _, ok := valueAt(rotation, reason).(string); !ok {
		t.Fatal("block-emergency-rotation carries no reason")
	}
	for value, want := range map[string]string{
		strings.Repeat("r", 128):       shapePasses,
		strings.Repeat("r", 129):       failsAt(specdata.ReasonShapeInvalid, reason),
		strings.Repeat("r", 127) + "é": failsAt(specdata.ReasonShapeInvalid, reason),
		strings.Repeat("é", 64):        failsAt(specdata.ReasonShapeInvalid, reason),
	} {
		if got := shapeOutcome(t, put(rotation, reason, value)); got != want {
			t.Errorf("reason of %d code points: %s, want %s", len([]rune(value)), got, want)
		}
	}
	height := Pointer{Name("block"), Name("record"), Name("height")}
	for value, want := range map[strictjson.Number]string{
		strictjson.MaxInteger: shapePasses,
		-1:                    failsAt(specdata.ReasonShapeInvalid, height),
		1.5:                   failsAt(specdata.ReasonShapeInvalid, height),
	} {
		if got := shapeOutcome(t, put(proofs["beacon-empty"], height, value)); got != want {
			t.Errorf("height %v: %s, want %s", value, got, want)
		}
	}
}
