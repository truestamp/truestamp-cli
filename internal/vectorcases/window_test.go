// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The submission window of SPEC.md section 11 through the cases: each
// online case's two edges (section 11.5) and the window-* family's rows
// and values (section 11.3), with the edges read here from the rows of
// CHAIN and SOURCES rather than from the verifier's candidates.

package vectorcases_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// specAssumptions are the stated assumptions of SPEC.md sections 11.1 and
// 11.2, by edge and basis, copied here word for word from SPEC.md.
var specAssumptions = map[string]string{
	"submitted_before stellar_ledger_close_read": "Stellar validators did not set the ledger's close time behind real time; a close time set ahead of real time only makes this edge later",
	"submitted_before bitcoin_median_time_read":  "at least 6 of blocks height + 1 to height + 10 carry honest times",
	"submitted_after nist_pulse_time_read":       "NIST does not release outputValue before timeStamp",
	"submitted_after stellar_ledger_close_read":  "the network did not set the close time ahead of real time",
	"submitted_after bitcoin_median_time_read":   "at least 6 of the 11 header times ending at that block are honest",
}

// expectedEdge is a case's expected edge, {basis, instant}, or nil for
// null.
type expectedEdge struct{ Basis, Instant string }

func expectedEdgeOf(t *testing.T, raw json.RawMessage) *expectedEdge {
	t.Helper()
	if string(raw) == "null" || len(raw) == 0 {
		return nil
	}
	var e expectedEdge
	if err := json.Unmarshal(raw, &e); err != nil || e.Basis == "" || e.Instant == "" {
		t.Fatalf("the edge %s: %v", raw, err)
	}
	return &e
}

// checkEdges holds an online case's run to its two edges and returns the
// kind of its edges, for the runner's tally. A case labelled failed
// expects both null (SPEC.md section 10.5), and its run is not held to the
// edges WINDOW drew, since a report writes them null (VERDICT). Any other
// case's run has exactly the edges the case expects, each with the basis
// and instant it gives, each an eight-member object of SPEC.md section
// 11.5 (checkEdgeObject), and its WINDOW row's values hold those edges and
// name each absent one. A run whose gate does not open draws none.
func checkEdges(t *testing.T, c vectorcases.Case, r *verifier.Run) string {
	t.Helper()
	wantAfter, wantBefore := expectedEdgeOf(t, c.Expected.SubmittedAfter), expectedEdgeOf(t, c.Expected.SubmittedBefore)
	if r.ExternalGate != verifier.ExternalGateOpen && (r.SubmittedAfter != nil || r.SubmittedBefore != nil) {
		t.Fatalf("a closed gate drew the edges %+v, %+v", r.SubmittedAfter, r.SubmittedBefore)
	}
	if c.Expected.Label == "failed" {
		if wantAfter != nil || wantBefore != nil {
			t.Fatalf("a failed case expects the edges %+v, %+v", wantAfter, wantBefore)
		}
		return "edges: none, label failed"
	}
	window := windowRow(t, r)
	edges, _ := window.Values["edges"].(map[string]any)
	absent, _ := window.Values["absent"].(map[string]any)
	for _, e := range []struct {
		name string
		got  *verifier.Edge
		want *expectedEdge
	}{{"submitted_after", r.SubmittedAfter, wantAfter}, {"submitted_before", r.SubmittedBefore, wantBefore}} {
		_, isAbsent := absent[e.name]
		switch {
		case e.want == nil && (e.got != nil || edges[e.name] != nil || !isAbsent):
			t.Fatalf("%s: drawn %+v, in values %v, absent %v; the case expects null", e.name, e.got, edges[e.name], isAbsent)
		case e.want == nil:
			continue
		case e.got == nil || isAbsent:
			t.Fatalf("%s: drawn %+v, absent %v; the case expects %+v", e.name, e.got, isAbsent, e.want)
		}
		value := e.got.Value()
		if value["basis"] != e.want.Basis || value["instant"] != e.want.Instant {
			t.Fatalf("%s: %v %v, want %+v", e.name, value["basis"], value["instant"], e.want)
		}
		if !sameJCS(t, value, edges[e.name]) {
			t.Fatalf("%s: the WINDOW row's values hold %v, the run %v", e.name, edges[e.name], value)
		}
		checkEdgeObject(t, r, e.name, value)
	}
	switch {
	case wantAfter != nil && wantBefore != nil:
		return "edges: both"
	case wantBefore != nil:
		return "edges: submitted_before only"
	case wantAfter != nil:
		return "edges: submitted_after only"
	}
	return "edges: none"
}

// checkEdgeObject holds an edge object, value, of the edge name, to SPEC.md
// section 11.5: exactly the eight members; its assumption that of its
// basis for its edge, word for word; its part that of a row that passed,
// a top-level entry's CHAIN row for the submitted-before edge and a
// SOURCES row for the submitted-after edge, whose candidate has its basis
// and instant; its sources those of that row's outcomes that passed, in
// order, each {cached false, fetched_at, source} as the outcome gives
// them; and its cached, fetched_at and source those of the first of them.
func checkEdgeObject(t *testing.T, r *verifier.Run, name string, value map[string]any) {
	t.Helper()
	members := []string{}
	for member := range value {
		members = append(members, member)
	}
	slices.Sort(members)
	if want := []string{"assumption", "basis", "cached", "fetched_at", "instant", "part", "source", "sources"}; !slices.Equal(members, want) {
		t.Fatalf("%s: members %q, want %q", name, members, want)
	}
	if want := specAssumptions[name+" "+value["basis"].(string)]; want == "" || value["assumption"] != want {
		t.Fatalf("%s: assumption %q for %v, want %q", name, value["assumption"], value["basis"], want)
	}
	step := map[string]string{"submitted_after": "SOURCES", "submitted_before": "CHAIN"}[name]
	part, _ := value["part"].(string)
	if name == "submitted_before" && part != "/commitments/bitcoin" && part != "/commitments/stellar" {
		t.Fatalf("%s: part %q is no top-level entry", name, part)
	}
	var row *verifier.Row
	for i := range r.Rows {
		if r.Rows[i].Step == step && r.Rows[i].Part != nil && *r.Rows[i].Part == part {
			row = &r.Rows[i]
		}
	}
	if row == nil || row.Status != verifier.StatusPass {
		t.Fatalf("%s: part %q has no passing %s row: %+v", name, part, step, row)
	}
	candidate, _ := row.Values["candidate"].(map[string]any)
	if candidate["basis"] != value["basis"] || candidate["instant"] != value["instant"] {
		t.Fatalf("%s: the row's candidate %v, the edge %v %v", name, candidate, value["basis"], value["instant"])
	}
	var want []any
	outcomes, _ := row.Values["outcomes"].([]any)
	for _, o := range outcomes {
		outcome := o.(map[string]any)
		if outcome["status"] == "pass" {
			want = append(want, map[string]any{"cached": false, "fetched_at": outcome["fetched_at"], "source": outcome["source"]})
		}
	}
	sources, _ := value["sources"].([]any)
	if len(want) == 0 || !sameJCS(t, sources, want) {
		t.Fatalf("%s: sources %v, want %v", name, sources, want)
	}
	first := sources[0].(map[string]any)
	if value["cached"] != false || value["fetched_at"] != first["fetched_at"] || value["source"] != first["source"] || !strings.Contains(first["source"].(string), "://") {
		t.Fatalf("%s: cached %v, fetched_at %v, source %v; the first source %v", name, value["cached"], value["fetched_at"], value["source"], first)
	}
}

// sameJCS reports whether a and b write the same JCS.
func sameJCS(t *testing.T, a, b any) bool {
	t.Helper()
	x, errX := jcs.Encode(a)
	y, errY := jcs.Encode(b)
	if errX != nil || errY != nil {
		t.Fatalf("JCS: %v, %v", errX, errY)
	}
	return bytes.Equal(x, y)
}

// windowRow is the run's one WINDOW row with a null part.
func windowRow(t *testing.T, r *verifier.Run) verifier.Row {
	t.Helper()
	var rows []verifier.Row
	for _, row := range r.Rows {
		if row.Step == "WINDOW" && row.Part == nil {
			rows = append(rows, row)
		}
	}
	if len(rows) != 1 {
		t.Fatalf("WINDOW rows with a null part: %+v", rows)
	}
	return rows[0]
}

// rowInstant is the instant of a row's candidate in Unix milliseconds,
// read here from its written form: a TIME's second, or a NISTTIME's
// millisecond.
func rowInstant(t *testing.T, row verifier.Row) int64 {
	t.Helper()
	candidate, _ := row.Values["candidate"].(map[string]any)
	text, _ := candidate["instant"].(string)
	layout := "2006-01-02T15:04:05Z"
	if len(text) == len("2006-01-02T15:04:05.000Z") {
		layout = "2006-01-02T15:04:05.000Z"
	}
	at, err := time.Parse(layout, text)
	if err != nil {
		t.Fatalf("the candidate %v: %v", candidate, err)
	}
	return at.UnixMilli()
}

// modelEdges draws both edges from the run's rows, here without the
// verifier's candidates: the part and instant of the earliest candidate of
// a passing top-level CHAIN row, and of the latest of a passing SOURCES
// row, the first of those at one instant, "" where there is none.
func modelEdges(t *testing.T, r *verifier.Run) (afterPart string, after int64, beforePart string, before int64) {
	t.Helper()
	for _, row := range r.Rows {
		if row.Status != verifier.StatusPass || row.Part == nil {
			continue
		}
		switch {
		case row.Step == "CHAIN" && (*row.Part == "/commitments/bitcoin" || *row.Part == "/commitments/stellar"):
			if at := rowInstant(t, row); beforePart == "" || at < before {
				beforePart, before = *row.Part, at
			}
		case row.Step == "SOURCES":
			if at := rowInstant(t, row); afterPart == "" || at > after {
				afterPart, after = *row.Part, at
			}
		}
	}
	return afterPart, after, beforePart, before
}

// idTime is the time inside an id, in Unix milliseconds: its first 12 hex
// digits, the hyphen skipped (SPEC.md section 11.3), read here without the
// verifier.
func idTime(t *testing.T, id string) int64 {
	t.Helper()
	ms, err := strconv.ParseUint(strings.ReplaceAll(id, "-", "")[:12], 16, 48)
	if err != nil {
		t.Fatalf("the id %q: %v", id, err)
	}
	return int64(ms)
}

// The window-* family (10), and the cases whose window the entry names,
// through every step (SPEC.md sections 11.1 to 11.3 and 11.5), beyond
// TestCases_ThroughEveryStep's rows, label and edges:
//
//   - each window-* case draws the edges a model reading the CHAIN and
//     SOURCES rows draws, at the same parts; its WINDOW row's candidates
//     are one for each passing top-level CHAIN row and each passing
//     SOURCES row, each named for its edge; and it has the row info
//     recorded_time_outside_window, at /subject/record/id with the id's
//     time in milliseconds, exactly when its WINDOW row passed and that
//     time lies outside the window;
//   - window-submitted_before_tie's two submitted-before candidates are at
//     one instant, and the edge is /commitments/bitcoin's, the first in
//     row order;
//   - window-edges_at_one_instant's edges are at one millisecond, a NIST
//     pulse and a Stellar ledger, and its row passes;
//   - window-id_on_submitted_before's id time equals the submitted-before
//     edge, and window-id_milliseconds_after_submitted_before's lies 1 to
//     999 milliseconds after it, within the edge's second;
//   - window-inverted's run drew both edges, the submitted-after edge
//     later;
//   - window-block's submitted-after edge is absent with an empty list,
//     since a block proof carries no entropy object;
//   - sources-all-pass_and_fail, -different_candidates and
//     -pass_and_unreachable leave the submitted-before edge absent, its
//     one reason /commitments/stellar's sources_disagree or
//     sources_incomplete, with no failure;
//   - sources-all-agree's edges each rest on both sources, a and b;
//   - gates-online-policy's WINDOW row is skip prior_failure under a
//     closed gate, with no candidate and both edges absent for
//     prior_failure; gates-online-chain_failure's is skip prior_failure
//     under an open gate, with both edges drawn, since its
//     /commitments/bitcoin row passed.
//
// Mutants killed: the earliest submitted-before candidate taken with <=
// (window-submitted_before_tie's edge is /commitments/stellar's); the
// first submitted-before candidate always taken, or the earliest
// submitted-after one (an edge differs from the model's); window_inverted
// on >= (window-edges_at_one_instant fails); the inverted check left out
// (window-inverted passes); the id time compared in seconds
// (window-id_milliseconds_after_submitted_before gives no info row); the
// id time compared with >= (window-id_on_submitted_before gives one); the
// submitted-after check left out (window-id_earlier_than_submitted_after
// gives no info row); the id time written as a count (its id_time is no
// NISTTIME); absentReasons listing every CHAIN row
// (sources-all-pass_and_fail's reasons hold the signing key event's
// entries); the edge's sources taken as the first counted source only
// (sources-all-agree's edges rest on one source); the candidates left out
// of the row's values, or listed without their edge (each window-* case's
// list differs, and gates-online-chain_failure's holds none).
func TestCases_TheWindow(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, c := range cases {
		named := map[string]bool{
			"sources-all-pass_and_fail": true, "sources-all-different_candidates": true, "sources-all-pass_and_unreachable": true,
			"sources-all-agree": true, "gates-online-policy": true, "gates-online-chain_failure": true,
		}[c.ID]
		if c.Family() != "window" && !named {
			continue
		}
		seen++
		got := run(t, c)
		if !rowsEqual(got.rows, expectedThrough(c)) {
			t.Fatalf("%s: rows %v, want %v", c.ID, got.rows, expectedThrough(c))
		}
		r := got.run
		window := windowRow(t, r)
		absent, _ := window.Values["absent"].(map[string]any)
		listed, _ := window.Values["candidates"].([]any)
		if c.Family() == "window" {
			checkWindowCase(t, c, r, window, listed)
		}
		switch c.ID {
		case "window-submitted_before_tie":
			var instants []int64
			for _, row := range r.Rows {
				if row.Step == "CHAIN" && row.Part != nil && strings.HasPrefix(*row.Part, "/commitments/") {
					instants = append(instants, rowInstant(t, row))
				}
			}
			if len(instants) != 2 || instants[0] != instants[1] || r.SubmittedBefore.Part != "/commitments/bitcoin" {
				t.Errorf("%s: candidates at %v, the edge %s", c.ID, instants, r.SubmittedBefore.Part)
			}
		case "window-edges_at_one_instant":
			if r.SubmittedAfter.Candidate.Milliseconds() != r.SubmittedBefore.Candidate.Milliseconds() ||
				r.SubmittedAfter.Candidate.Basis != "nist_pulse_time_read" || r.SubmittedBefore.Candidate.Basis != "stellar_ledger_close_read" {
				t.Errorf("%s: %+v, %+v", c.ID, r.SubmittedAfter.Candidate, r.SubmittedBefore.Candidate)
			}
		case "window-id_on_submitted_before", "window-id_milliseconds_after_submitted_before":
			gap := idTime(t, r.SubjectID) - r.SubmittedBefore.Candidate.Milliseconds()
			if (c.ID == "window-id_on_submitted_before") != (gap == 0) || gap < 0 || gap > 999 {
				t.Errorf("%s: the id time is %d ms after the edge", c.ID, gap)
			}
		case "window-inverted":
			if r.SubmittedAfter == nil || r.SubmittedBefore == nil || r.SubmittedAfter.Candidate.Milliseconds() <= r.SubmittedBefore.Candidate.Milliseconds() {
				t.Errorf("%s: %+v, %+v", c.ID, r.SubmittedAfter, r.SubmittedBefore)
			}
		case "window-block":
			if reasons, ok := absent["submitted_after"].([]any); !ok || len(reasons) != 0 || r.SubmittedAfter != nil || len(absent) != 1 {
				t.Errorf("%s: absent %v", c.ID, absent)
			}
		case "sources-all-pass_and_fail", "sources-all-different_candidates", "sources-all-pass_and_unreachable":
			code := "sources_disagree"
			if c.ID == "sources-all-pass_and_unreachable" {
				code = "sources_incomplete"
			}
			want := []any{map[string]any{"code": code, "part": "/commitments/stellar"}}
			if !sameJCS(t, absent["submitted_before"], want) || r.SubmittedBefore != nil || r.Failed() || len(absent) != 1 {
				t.Errorf("%s: absent %v, want submitted_before %v; failed %v", c.ID, absent, want, r.Failed())
			}
		case "sources-all-agree":
			for _, e := range []*verifier.Edge{r.SubmittedAfter, r.SubmittedBefore} {
				if e == nil || len(e.Sources) != 2 || e.Sources[1].FetchedAt.Before(e.Sources[0].FetchedAt) {
					t.Errorf("%s: an edge %+v; want one resting on both sources, in their order", c.ID, e)
				}
			}
		case "gates-online-policy":
			want := map[string]any{
				"submitted_after":  []any{map[string]any{"code": "prior_failure", "part": "/witnesses/entropy_bitcoin"}, map[string]any{"code": "prior_failure", "part": "/witnesses/entropy_nist"}, map[string]any{"code": "prior_failure", "part": "/witnesses/entropy_stellar"}},
				"submitted_before": []any{map[string]any{"code": "prior_failure", "part": "/commitments/bitcoin"}, map[string]any{"code": "prior_failure", "part": "/commitments/stellar"}},
			}
			if window.Code != "prior_failure" || len(listed) != 0 || !sameJCS(t, absent, want) || r.SubmittedAfter != nil || r.SubmittedBefore != nil {
				t.Errorf("%s: %s, candidates %v, absent %v", c.ID, window.Code, listed, absent)
			}
		case "gates-online-chain_failure":
			edges, _ := window.Values["edges"].(map[string]any)
			if window.Code != "prior_failure" || len(listed) != 4 || len(edges) != 2 || len(absent) != 0 ||
				r.SubmittedBefore == nil || r.SubmittedBefore.Part != "/commitments/bitcoin" || r.SubmittedAfter == nil {
				t.Errorf("%s: %s, candidates %d, edges %v, absent %v", c.ID, window.Code, len(listed), edges, absent)
			}
		}
	}
	if seen != 16 {
		t.Fatalf("%d cases, want the 10 window-* cases and 6 named", seen)
	}
}

// checkWindowCase holds a window-* case's run to the model's edges, its
// listed candidates and its info row (see TestCases_TheWindow).
func checkWindowCase(t *testing.T, c vectorcases.Case, r *verifier.Run, window verifier.Row, listed []any) {
	t.Helper()
	afterPart, after, beforePart, before := modelEdges(t, r)
	got := func(e *verifier.Edge) (string, int64) {
		if e == nil {
			return "", 0
		}
		return e.Part, e.Candidate.Milliseconds()
	}
	gotAfterPart, gotAfter := got(r.SubmittedAfter)
	gotBeforePart, gotBefore := got(r.SubmittedBefore)
	if gotAfterPart != afterPart || gotAfter != after || gotBeforePart != beforePart || gotBefore != before {
		t.Errorf("%s: edges %s %d, %s %d; the model %s %d, %s %d", c.ID, gotAfterPart, gotAfter, gotBeforePart, gotBefore, afterPart, after, beforePart, before)
	}
	var wantEdges []string
	for _, row := range r.Rows {
		if row.Status != verifier.StatusPass || row.Part == nil {
			continue
		}
		if row.Step == "CHAIN" && strings.HasPrefix(*row.Part, "/commitments/") {
			wantEdges = append(wantEdges, "submitted_before "+*row.Part)
		}
		if row.Step == "SOURCES" {
			wantEdges = append(wantEdges, "submitted_after "+*row.Part)
		}
	}
	var gotEdges []string
	for _, l := range listed {
		value := l.(map[string]any)
		gotEdges = append(gotEdges, fmt.Sprint(value["edge"], " ", value["part"]))
	}
	if !slices.Equal(gotEdges, wantEdges) {
		t.Errorf("%s: candidates %q, want %q", c.ID, gotEdges, wantEdges)
	}
	ms := idTime(t, r.SubjectID)
	outside := window.Status == verifier.StatusPass && (ms > before || (afterPart != "" && ms < after))
	var info []verifier.Row
	for _, row := range r.Rows {
		if row.Code == "recorded_time_outside_window" {
			info = append(info, row)
		}
	}
	if !outside {
		if len(info) != 0 {
			t.Errorf("%s: info rows %+v, the id time %d inside the window", c.ID, info, ms)
		}
		return
	}
	wantTime := time.UnixMilli(ms).UTC().Format("2006-01-02T15:04:05.000Z")
	if len(info) != 1 || info[0].Part == nil || *info[0].Part != "/subject/record/id" || info[0].Status != verifier.StatusInfo || info[0].Values["id_time"] != wantTime {
		t.Errorf("%s: info rows %+v, want one at /subject/record/id with %s", c.ID, info, wantTime)
	}
}
