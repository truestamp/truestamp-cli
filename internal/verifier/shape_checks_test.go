// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// Step 7, SHAPE, checks 4 to 6 (SPEC.md section 10.2), on mutants of the
// 11 positive proofs, each signed again and run through the gate keyless
// under E_v test (SPEC.md section 8.3), then SHAPE.

// shapeRowOf returns SHAPE's row of proof.
func shapeRowOf(t *testing.T, proof map[string]any) Row {
	t.Helper()
	for _, row := range shapeRun(t, proof).Rows {
		if row.Step == "SHAPE" {
			return row
		}
	}
	t.Fatal("no SHAPE row")
	return Row{}
}

// timesOf returns every TIME and NISTTIME value of proof, by the test's
// own walk: each header_time, closed_at and timeStamp member.
func timesOf(proof map[string]any) []timePosition {
	var out []timePosition
	for _, n := range nodes(proof) {
		if len(n.at) == 0 || n.at[len(n.at)-1].isIndex {
			continue
		}
		if _, ok := n.value.(string); !ok {
			continue
		}
		switch n.at[len(n.at)-1].name {
		case "header_time", "closed_at":
			out = append(out, timePosition{n.at, ClassTIME})
		case "timeStamp":
			out = append(out, timePosition{n.at, ClassNISTTIME})
		}
	}
	return out
}

func timeSuffix(class TimeClass) string {
	if class == ClassNISTTIME {
		return ".000Z"
	}
	return "Z"
}

// Check 4: a witness missing, one more than witness_hashes names, or one
// other in its place with as many of each, fails at /witnesses, whose
// values name the rule and never the names, and whose message quotes no
// name; the same name left out of both passes; and no type but item is
// checked. Mutants killed: check 4 comparing the counts alone (the swapped
// mutant passes); check 4 asking only that each name of witness_hashes be
// a witness (the extra mutant passes); the values carrying the record's
// witness names.
func TestShape_CheckFourTheWitnessesOfAnItem(t *testing.T) {
	proofs := positiveProofs(t)
	item := proofs["item"]
	witnesses := Pointer{Name("witnesses")}
	names := Pointer{Name("subject"), Name("record"), Name("witness_hashes")}
	for _, name := range []string{"entropy_nist", "entropy_stellar"} {
		if valueAt(item, with(witnesses, Name(name))) == nil || valueAt(item, with(names, Name(name))) == nil {
			t.Fatalf("the item carries no %s", name)
		}
	}
	missing := remove(item, with(witnesses, Name("entropy_nist")))
	extra := remove(item, with(names, Name("entropy_nist")))
	swapped := remove(remove(item, with(witnesses, Name("entropy_nist"))), with(names, Name("entropy_stellar")))
	if len(valueAt(swapped, witnesses).(map[string]any)) != len(valueAt(swapped, names).(map[string]any)) {
		t.Fatal("the swapped mutant has more of one than of the other")
	}
	for kind, mutant := range map[string]map[string]any{"missing": missing, "extra": extra, "swapped": swapped} {
		if failures := ProofFailures(mutant); len(failures) != 0 {
			t.Fatalf("%s: the schema fails it: %v", kind, failures)
		}
		row := shapeRowOf(t, mutant)
		if got := rowSummary([]Row{row}); got != failsAt(specdata.ReasonShapeInvalid, witnesses) {
			t.Errorf("%s: %s", kind, got)
		}
		if got := string(encode(t, row.Values)); got != `{"rule":"witnesses_named_by_witness_hashes"}` || strings.Contains(row.Message, "entropy") {
			t.Errorf("%s: values %s, message %q", kind, got, row.Message)
		}
	}
	both := remove(remove(item, with(witnesses, Name("entropy_nist"))), with(names, Name("entropy_nist")))
	if got := shapeOutcome(t, both); got != shapePasses {
		t.Errorf("the same name left out of both: %s", got)
	}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		if _, carries := proof["witnesses"]; carries != (proof["type"] == "item") {
			t.Errorf("%s: witnesses carried %v", name, carries)
		}
		if got := shapeOutcome(t, proof); got != shapePasses {
			t.Errorf("%s: %s", name, got)
		}
	}
}

// roundTripFailures are TIME fields that match the pattern and do not
// write back to themselves.
var roundTripFailures = []string{
	"2026-09-28T24:00:00", "2026-02-30T00:00:00", "2026-09-28T23:59:60", "2016-12-31T23:59:60",
	"1900-02-29T00:00:00", "2100-02-29T00:00:00", "2026-02-29T00:00:00", "2026-13-01T00:00:00",
	"2026-00-10T00:00:00", "2026-04-31T00:00:00", "2026-09-28T12:60:00", "2026-01-00T00:00:00",
}

// Check 5: each value that does not write back to itself, at every TIME
// and NISTTIME position of every positive proof, fails shape_invalid at
// that value, with the class and the value as declared in its values, and
// a message that does not quote it. The positions are every header_time,
// closed_at and timeStamp, under /commitments, /signing_key_event,
// /subject, /witnesses and /observations. Mutants killed: timePositions
// leaving out /signing_key_event/commitments (its header_time mutants
// pass); check 5 skipping NIST (the timeStamp mutants pass); the values'
// class always TIME (the timeStamp rows name TIME).
func TestShape_CheckFiveEachTimeThatDoesNotWriteBack(t *testing.T) {
	proofs := positiveProofs(t)
	members, roots := map[string]bool{}, map[string]bool{}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		if got, want := len(timePositions(proof)), len(timesOf(proof)); got != want {
			t.Errorf("%s: the layout gives %d times, the walk %d", name, got, want)
		}
		for _, p := range timesOf(proof) {
			for _, bad := range roundTripFailures {
				text := bad + timeSuffix(p.class)
				mutant := put(proof, p.at, text)
				if failures := ProofFailures(mutant); len(failures) != 0 {
					t.Fatalf("%s %s: the schema fails %q", name, p.at.Encode(), text)
				}
				row := shapeRowOf(t, mutant)
				want := `{"class":"` + p.class.String() + `","declared":"` + text + `"}`
				if got := rowSummary([]Row{row}); got != failsAt(specdata.ReasonShapeInvalid, p.at) {
					t.Errorf("%s %s %q: %s", name, p.at.Encode(), text, got)
				} else if got := string(encode(t, row.Values)); got != want || strings.Contains(row.Message, text) {
					t.Errorf("%s %s: values %s, message %q", name, p.at.Encode(), got, row.Message)
				}
			}
			members[p.at[len(p.at)-1].name] = true
			roots[p.at[0].name] = true
		}
	}
	if len(members) != 3 || len(roots) != 5 {
		t.Fatalf("members %v, roots %v", members, roots)
	}
}

// Check 5: the boundaries that write back pass at every position: February
// 29 of 2024, 2000 and 0000, 23:59:59, the first and last seconds of the
// years 0000 and 9999 and the epoch, and the fractions .000, .001 and .999.
// Mutants killed: leapYear without the 400-year clause (2000-02-29 fails);
// the hour allowed only up to 22 (23:59:59 fails).
func TestShape_CheckFiveTheBoundariesThatWriteBackPass(t *testing.T) {
	proofs := positiveProofs(t)
	good := []string{"2024-02-29T00:00:00", "2000-02-29T00:00:00", "0000-02-29T00:00:00", "2026-09-28T23:59:59",
		"0000-01-01T00:00:00", "9999-12-31T23:59:59", "1970-01-01T00:00:00", "2026-12-31T23:59:59"}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		for _, p := range timesOf(proof) {
			texts := []string{}
			for _, g := range good {
				texts = append(texts, g+timeSuffix(p.class))
			}
			if p.class == ClassNISTTIME {
				texts = append(texts, "2026-09-28T23:59:59.001Z", "2026-09-28T23:59:59.999Z")
			}
			for _, text := range texts {
				if got := shapeOutcome(t, put(proof, p.at, text)); got != shapePasses {
					t.Errorf("%s %s %q: %s", name, p.at.Encode(), text, got)
				}
			}
		}
	}
}

// Check 5 comes after check 3: an offset, a lower case z or t, a missing
// leading zero, and a missing, short, long or comma fraction each fail the
// schema's pattern at that value, so the row's values are check 3's, with
// no class. Mutant killed: check 5 made before the schema (each row's
// values name its class).
func TestShape_CheckFiveAValueOffItsPatternFailsCheckThree(t *testing.T) {
	proofs := positiveProofs(t)
	others := map[TimeClass][]string{
		ClassTIME: {"2026-09-28T12:00:00+00:00", "2026-09-28T12:00:00-05:00", "2026-09-28T12:00:00z", "2026-09-28t12:00:00Z",
			"2026-09-28T12:00:00.000Z", "2026-09-28 12:00:00Z", "2026-9-28T12:00:00Z", "2026-09-28T2:00:00Z"},
		ClassNISTTIME: {"2026-09-28T12:00:00Z", "2026-09-28T12:00:00.00Z", "2026-09-28T12:00:00.0000Z",
			"2026-09-28T12:00:00.000+00:00", "2026-09-28T12:00:00,000Z", "2026-09-28T12:00:00.000z"},
	}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		for _, p := range timesOf(proof) {
			for _, text := range others[p.class] {
				row := shapeRowOf(t, put(proof, p.at, text))
				if got := rowSummary([]Row{row}); got != failsAt(specdata.ReasonShapeInvalid, p.at) || row.Values["class"] != nil || row.Values["keyword"] == nil {
					t.Errorf("%s %s %q: %s %v", name, p.at.Encode(), text, got, row.Values)
				}
			}
		}
	}
}

// Check 5: of any two times that fail, in any positive proof, the first in
// document order is the row, which in a beacon is not the layout's order:
// /observations sorts between /commitments and /signing_key_event.
// Mutant killed: check 5 taking the first failing time in the layout's
// order (the beacon's pairs of /observations and /signing_key_event fail
// at the later).
func TestShape_CheckFiveTheFirstInDocumentOrder(t *testing.T) {
	proofs := positiveProofs(t)
	crossed := false
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		times := timesOf(proof)
		for i, a := range times {
			for _, b := range times[i+1:] {
				mutant := put(put(proof, a.at, "2026-02-30T00:00:00"+timeSuffix(a.class)), b.at, "2026-02-30T00:00:00"+timeSuffix(b.class))
				first := a.at
				if Compare(b.at, a.at) < 0 {
					first = b.at
				}
				if got := shapeOutcome(t, mutant); got != failsAt(specdata.ReasonShapeInvalid, first) {
					t.Errorf("%s %s and %s: %s", name, a.at.Encode(), b.at.Encode(), got)
				}
				pair := a.at[0].name + " " + b.at[0].name
				crossed = crossed || pair == "observations signing_key_event" || pair == "signing_key_event observations"
			}
		}
	}
	if !crossed {
		t.Fatal("no pair of /observations and /signing_key_event times")
	}
}

// digestOfText is d of a full constructions text, SHA-256 of its JCS,
// computed here without specdata.
func digestOfText(t *testing.T, text any) string {
	t.Helper()
	encoded, err := jcs.Encode(text)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// Check 6: CONSTRUCTIONS_ACCEPTED holds the one text, which every positive
// proof carries in full; each mode that names an accepted text, or none,
// passes, and the row names the mode, with the d it computed in full mode
// only; a full text changed by a byte or a row fails construction_mismatch
// with the d it computed, member order being no byte of it; a reference to
// a text not accepted fails construction_mismatch with its d declared; and
// any other value fails shape_invalid at check 3. Mutants killed: the
// digest taken over encoding/json's Marshal of the text (every full text
// fails, since Marshal escapes its < and >); a reference's pass row giving
// its d (the reference row's values carry constructions_digest); a
// reference compared without its digest checked (the unaccepted reference
// passes).
func TestShape_CheckSixConstructions(t *testing.T) {
	proofs := positiveProofs(t)
	accepted := specdata.Accepted()
	if len(accepted) != 1 {
		t.Fatalf("CONSTRUCTIONS_ACCEPTED holds %d texts", len(accepted))
	}
	d := accepted[0]
	url := constructionsReferencePrefix + d + constructionsReferenceSuffix
	constructions := Pointer{Name("constructions")}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		if digestOfText(t, proof["constructions"]) != d {
			t.Fatalf("%s does not carry the accepted text in full", name)
		}
		for mode, c := range map[string]struct {
			proof  map[string]any
			values string
		}{
			"full":      {proof, `{"constructions":"full","constructions_digest":"` + d + `"}`},
			"reference": {put(proof, constructions, url), `{"constructions":"reference"}`},
			"none":      {remove(proof, constructions), `{"constructions":"none"}`},
		} {
			row := shapeRowOf(t, c.proof)
			if got := rowSummary([]Row{row}); got != shapePasses || string(encode(t, row.Values)) != c.values {
				t.Errorf("%s, %s: %s %s", name, mode, got, encode(t, row.Values))
			}
		}
	}
	item := proofs["item"]
	text := item["constructions"].(map[string]any)
	changed := clone(text).(map[string]any)
	changed["merkle_path"] = changed["merkle_path"].(string) + " "
	missing := clone(text).(map[string]any)
	delete(missing, "merkle_path")
	extra := clone(text).(map[string]any)
	extra["zz"] = strictjson.Number(1)
	for kind, mutant := range map[string]map[string]any{"a byte changed": changed, "a row missing": missing, "a row added": extra} {
		row := shapeRowOf(t, put(item, constructions, mutant))
		want := `{"constructions":"full","constructions_digest":"` + digestOfText(t, mutant) + `"}`
		if got := rowSummary([]Row{row}); got != failsAt(specdata.ReasonConstructionMismatch, constructions) ||
			string(encode(t, row.Values)) != want || !strings.Contains(row.Message, "carries a constructions text") {
			t.Errorf("%s: %s %s %q", kind, got, encode(t, row.Values), row.Message)
		}
	}
	other := strings.Repeat("0", 64)
	row := shapeRowOf(t, put(item, constructions, constructionsReferencePrefix+other+constructionsReferenceSuffix))
	if got := rowSummary([]Row{row}); got != failsAt(specdata.ReasonConstructionMismatch, constructions) ||
		string(encode(t, row.Values)) != `{"constructions":"reference","declared":"`+other+`"}` ||
		!strings.Contains(row.Message, "names by reference a constructions text") || strings.Contains(row.Message, "carries") {
		t.Errorf("an unaccepted reference: %s %s %q", got, encode(t, row.Values), row.Message)
	}
	for _, value := range []any{nil, strictjson.Number(1), []any{}, true, strings.Replace(url, "www.truestamp.com", "truestamp.com", 1),
		strings.ToUpper(url), url + "\n", "http" + strings.TrimPrefix(url, "https"), strings.Replace(url, d, strings.ToUpper(d), 1)} {
		row := shapeRowOf(t, put(item, constructions, value))
		if got := rowSummary([]Row{row}); got != failsAt(specdata.ReasonShapeInvalid, constructions) || row.Values["keyword"] == nil {
			t.Errorf("constructions %#v: %s %v", value, got, row.Values)
		}
	}
}

// The order of checks 3 to 6: each check's failure hides the later
// checks' failures, and a time off its pattern fails at check 3 before
// check 4's /witnesses. Mutants killed: check 5 made before check 4 (the
// witnesses and time mutant fails at the time); check 6 made before check
// 5 (the time and text mutant fails at /constructions).
func TestShape_TheOrderOfChecksThreeToSix(t *testing.T) {
	item := positiveProofs(t)["item"]
	height := Pointer{Name("block"), Name("record"), Name("height")}
	witness := Pointer{Name("witnesses"), Name("entropy_nist")}
	firstTime := timesOf(item)[0]
	badTime := func(p map[string]any) map[string]any { return put(p, firstTime.at, "2026-02-30T00:00:00Z") }
	badText := func(p map[string]any) map[string]any {
		return put(p, Pointer{Name("constructions")}, map[string]any{"x": "y"})
	}
	for _, c := range []struct {
		mutant map[string]any
		want   string
	}{
		{badText(badTime(remove(put(item, height, strictjson.Number(-1)), witness))), failsAt(specdata.ReasonShapeInvalid, height)},
		{badText(badTime(remove(item, witness))), failsAt(specdata.ReasonShapeInvalid, Pointer{Name("witnesses")})},
		{badText(badTime(item)), failsAt(specdata.ReasonShapeInvalid, firstTime.at)},
		{badText(item), failsAt(specdata.ReasonConstructionMismatch, Pointer{Name("constructions")})},
	} {
		if got := shapeOutcome(t, c.mutant); got != c.want {
			t.Errorf("%s, want %s", got, c.want)
		}
	}
	extra := remove(item, Pointer{Name("subject"), Name("record"), Name("witness_hashes"), Name("entropy_stellar")})
	for _, c := range []struct {
		at   Pointer
		text string
	}{
		{Pointer{Name("commitments"), Name("bitcoin"), Name("header_time")}, "2026-01-01 00:00:00Z"},
		{Pointer{Name("witnesses"), Name("entropy_nist"), Name("payload"), Name("pulse"), Name("timeStamp")}, "2026-01-01 00:00:00.000Z"},
	} {
		if _, ok := valueAt(item, c.at).(string); !ok {
			t.Fatalf("the item has no time at %s", c.at.Encode())
		}
		if got := shapeOutcome(t, put(extra, c.at, c.text)); got != failsAt(specdata.ReasonShapeInvalid, c.at) {
			t.Errorf("%s: %s", c.at.Encode(), got)
		}
	}
}

// A failure at checks 4 to 6 ends the run, leaves the type and subject id
// "" and the attribution SIGNATURE gave the run, and gives the later steps
// skip prior_failure. Mutant killed: shapeConstructions's failure added as
// a row without ending the run (POLICY's rows follow it).
func TestShape_AFailureAtChecksFourToSixEndsTheRun(t *testing.T) {
	item := positiveProofs(t)["item"]
	firstTime := timesOf(item)[0]
	for _, mutant := range []map[string]any{
		remove(item, Pointer{Name("witnesses"), Name("entropy_nist")}),
		put(item, firstTime.at, "2026-02-30T00:00:00Z"),
		put(item, Pointer{Name("constructions")}, map[string]any{"x": "y"}),
	} {
		run := shapeRun(t, mutant)
		Policy(run)
		if !run.Ended || run.Type != "" || run.SubjectID != "" || run.Attribution != AttributionUnattributed ||
			len(run.Rows) != 14 || rowSummary(run.Rows[8:9]) != "[prior_failure null skip POLICY]" {
			t.Errorf("ended %v, type %q, subject %q, attribution %q, rows %s", run.Ended, run.Type, run.SubjectID, run.Attribution, rowSummary(run.Rows[7:]))
		}
	}
}
