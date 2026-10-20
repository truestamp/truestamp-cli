// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// windowBase is the second the window tests' instants count from,
// 2026-09-21T14:13:20Z.
const windowBase int64 = 1790000000

// windowFetched is the fetched_at of the window tests' reads: the base
// second plus n milliseconds.
func windowFetched(n int64) time.Time { return time.UnixMilli(windowBase*1000 + n).UTC() }

// idAt is a UUIDv7 whose unix_ts_ms is ms.
func idAt(ms int64) string {
	return fmt.Sprintf("%08x-%04x-7000-8000-000000000000", ms>>16, ms&0xffff)
}

// counted is a source a candidate rests on, by its URL, whose read that
// gave the candidate came back at fetched.
func counted(url string, candidate Candidate, fetched time.Time) AskedSource {
	candidate.FetchedAt = fetched
	return AskedSource{
		Source:  SourceRecord{Source: Source{URL: url}},
		Outcome: SourceOutcome{Status: StatusPass, Candidate: &candidate},
	}
}

// rowCandidate is the candidate of the row of step at part with basis,
// at instant in its class's unit, resting on the sources named by their
// URLs, each read at its own fetched_at, the first at 1 ms past the base
// second and each later one a millisecond after.
func rowCandidate(step, part, basis string, class TimeClass, instant int64, urls ...string) RowCandidate {
	c := Candidate{Basis: basis, Class: class, Instant: instant, FetchedAt: windowFetched(1)}
	rc := RowCandidate{Step: step, Part: part, Candidate: c}
	for i, url := range urls {
		rc.Counted = append(rc.Counted, counted(url, c, windowFetched(int64(i+1))))
	}
	return rc
}

// stellarAt, bitcoinAt and nistAt are candidates at a Stellar ledger's,
// a Bitcoin block's or a NIST pulse's instant, read from one source each.
func stellarAt(step, part string, seconds int64) RowCandidate {
	return rowCandidate(step, part, stellarCloseBasis, ClassTIME, seconds, "https://horizon.example")
}

func bitcoinAt(step, part string, seconds int64) RowCandidate {
	return rowCandidate(step, part, bitcoinMedianBasis, ClassTIME, seconds, "https://esplora.example")
}

func nistAt(part string, ms int64) RowCandidate {
	return rowCandidate("SOURCES", part, nistTimeBasis, ClassNISTTIME, ms, "https://beacon.nist.gov")
}

// windowRun is a run of type typ, whose subject id's time is idMS, under an
// open gate, with rows and candidates: a passing row is added for each
// candidate whose part has none among rows, in the candidates' order,
// after rows.
func windowRun(typ string, idMS int64, rows []Row, candidates ...RowCandidate) *Run {
	run := &Run{ExternalGate: ExternalGateOpen, Type: typ, SubjectID: idAt(idMS), Attribution: "keyring:d", Candidates: candidates}
	run.Rows = append(run.Rows, rows...)
	for _, c := range candidates {
		has := false
		for _, row := range rows {
			has = has || (row.Part != nil && *row.Part == c.Part && row.Step == c.Step)
		}
		if !has {
			run.AddRow(PassRow(c.Step, Part(c.Part), "read", nil))
		}
	}
	return run
}

// windowRows are the run's WINDOW rows.
func windowRows(run *Run) []Row {
	var out []Row
	for _, row := range run.Rows {
		if row.Step == "WINDOW" {
			out = append(out, row)
		}
	}
	return out
}

// edgeParts names the run's two edges by part and instant in
// milliseconds, "-" for an absent one.
func edgeParts(run *Run) string {
	name := func(e *Edge) string {
		if e == nil {
			return "-"
		}
		return fmt.Sprintf("%s@%d", e.Part, e.Candidate.Milliseconds())
	}
	return name(run.SubmittedAfter) + " " + name(run.SubmittedBefore)
}

// WINDOW's row under an open gate is the first of SPEC.md section 11.3's
// that applies, part null: skip prior_failure after a failure, even with
// inverted edges; fail window_inverted when the submitted-after edge is
// later; info window_not_established when the submitted-before edge is
// absent, whatever the submitted-after edge; and pass, edges at one
// instant included and a run with no submitted-after edge. An absent edge
// never fails the run. Mutants killed: the inverted check made before the
// failure check (the failed inverted run gives window_inverted); the
// inverted check left out (the inverted run passes); window_inverted on
// >= (the edges at one instant fail); the absence check left out (the run
// with no submitted-before edge panics); the failure check left out (the
// failed run gives window_inverted).
func TestWindow_TheRowsOrder(t *testing.T) {
	failure := NewRow("CHAIN", specdata.ReasonChainMismatch, Part("/signing_key_event/commitments/stellar"), "differs", nil)
	early, late := windowBase, windowBase+100
	for _, c := range []struct {
		name  string
		rows  []Row
		cands []RowCandidate
		want  string
	}{
		{"a failure, edges inverted", []Row{failure},
			[]RowCandidate{stellarAt("CHAIN", "/commitments/stellar", early), stellarAt("SOURCES", "/witnesses/entropy_stellar", late)},
			"[prior_failure null skip WINDOW]"},
		{"inverted", nil,
			[]RowCandidate{stellarAt("CHAIN", "/commitments/stellar", early), stellarAt("SOURCES", "/witnesses/entropy_stellar", late)},
			"[window_inverted null fail WINDOW]"},
		{"edges at one instant", nil,
			[]RowCandidate{stellarAt("CHAIN", "/commitments/stellar", early), nistAt("/witnesses/entropy_nist", early*1000)},
			"[null null pass WINDOW]"},
		{"no submitted-before edge", nil,
			[]RowCandidate{stellarAt("SOURCES", "/witnesses/entropy_stellar", late)},
			"[window_not_established null info WINDOW]"},
		{"no submitted-after edge", nil,
			[]RowCandidate{bitcoinAt("CHAIN", "/commitments/bitcoin", late)},
			"[null null pass WINDOW]"},
		{"no edge", nil, nil, "[window_not_established null info WINDOW]"},
	} {
		run := windowRun("item", early*1000, c.rows, c.cands...)
		Window(run)
		if got := rowSummary(windowRows(run)); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
		if failed := c.rows != nil || strings.Contains(c.want, "fail "); run.Failed() != failed {
			t.Errorf("%s: failed %v, want %v", c.name, run.Failed(), failed)
		}
	}
}

// Each edge's choice among its candidates (SPEC.md sections 11.1 to
// 11.3): the submitted-before edge is the earliest candidate of a
// top-level entry's CHAIN row, never one at /signing_key_event/commitments
// however early; the submitted-after edge the latest SOURCES candidate;
// of two at one instant, the first in row order; and instants compare at
// each source's precision, a Stellar or Bitcoin second as its first
// millisecond against a NIST millisecond. Mutants killed: the earliest
// taken with <= (the tie gives /commitments/stellar); the latest taken
// with >= (the tie gives /witnesses/entropy_stellar); the first candidate
// always taken (the later, earlier Bitcoin candidate is passed over);
// every CHAIN candidate taken (the signing key event's entry is the
// edge); Milliseconds ignoring the class (the NIST pulse 999 ms into the
// second before the Stellar ledger's compares as later); the
// submitted-after edge taken as the earliest (the Bitcoin candidate is
// the edge).
func TestWindow_EachEdgeChoice(t *testing.T) {
	s := windowBase
	for _, c := range []struct {
		name  string
		cands []RowCandidate
		want  string
	}{
		{"a tie of submitted-before candidates", []RowCandidate{
			bitcoinAt("CHAIN", "/commitments/bitcoin", s+50), stellarAt("CHAIN", "/commitments/stellar", s+50),
		}, fmt.Sprintf("- /commitments/bitcoin@%d", (s+50)*1000)},
		{"the earliest, second in row order", []RowCandidate{
			bitcoinAt("CHAIN", "/commitments/bitcoin", s+60), stellarAt("CHAIN", "/commitments/stellar", s+50),
		}, fmt.Sprintf("- /commitments/stellar@%d", (s+50)*1000)},
		{"a signing key event's entry gives no candidate", []RowCandidate{
			stellarAt("CHAIN", "/commitments/stellar", s+50), stellarAt("CHAIN", "/signing_key_event/commitments/stellar", s+10),
		}, fmt.Sprintf("- /commitments/stellar@%d", (s+50)*1000)},
		{"a tie of submitted-after candidates", []RowCandidate{
			nistAt("/witnesses/entropy_nist", s*1000), stellarAt("SOURCES", "/witnesses/entropy_stellar", s),
			stellarAt("CHAIN", "/commitments/stellar", s+50),
		}, fmt.Sprintf("/witnesses/entropy_nist@%d /commitments/stellar@%d", s*1000, (s+50)*1000)},
		{"the latest, by precision", []RowCandidate{
			bitcoinAt("SOURCES", "/witnesses/entropy_bitcoin", s-1), nistAt("/witnesses/entropy_nist", s*1000+999),
			stellarAt("SOURCES", "/witnesses/entropy_stellar", s+1), stellarAt("CHAIN", "/commitments/stellar", s+50),
		}, fmt.Sprintf("/witnesses/entropy_stellar@%d /commitments/stellar@%d", (s+1)*1000, (s+50)*1000)},
	} {
		run := windowRun("item", s*1000, nil, c.cands...)
		Window(run)
		if got := edgeParts(run); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

// Precision per source (SPEC.md section 11.3): a NIST pulse 500 ms past a
// Stellar ledger's close second is later than it, so a submitted-after
// edge there inverts the window. Mutant killed: instants compared in
// seconds (the NIST millisecond truncated to its second gives a pass).
func TestWindow_PrecisionPerSource(t *testing.T) {
	run := windowRun("item", windowBase*1000, nil,
		stellarAt("CHAIN", "/commitments/stellar", windowBase), nistAt("/witnesses/entropy_nist", windowBase*1000+500))
	Window(run)
	if got := rowSummary(windowRows(run)); got != "[window_inverted null fail WINDOW]" {
		t.Fatalf("%s", got)
	}
}

// Absent edges (SPEC.md sections 11.2 and 11.3): sources_disagree and
// sources_incomplete give no candidate, so they leave the submitted-before
// edge absent, with no failure, the row info window_not_established, and
// its reasons in the row's values, the part and code of each top-level
// entry's row and never a signing key event's entry, a genesis.bitcoin or
// KEY_EVENT_COMMITMENT; a block proof and an empty beacon have no SOURCES
// row, so their submitted-after edge is absent with an empty list, and a
// passing submitted-before edge gives a pass, an empty beacon's included.
// Mutants killed: absentReasons taking every CHAIN row (the signing key
// event's entry, the genesis row and KEY_EVENT_COMMITMENT are listed); a
// pass's code written "" (the passing row is listed with the code ""); a
// submitted-after edge copied from the submitted-before edge for an empty
// beacon (the beacon has a submitted-after edge).
func TestWindow_AbsentEdges(t *testing.T) {
	disagree := NewRow("CHAIN", specdata.ReasonSourcesDisagree, Part("/commitments/bitcoin"), "disagree", nil)
	incomplete := NewRow("CHAIN", specdata.ReasonSourcesIncomplete, Part("/commitments/stellar"), "too few", nil)
	genesis := PassRow("CHAIN", Part("/block/record/metadata/genesis/bitcoin"), "held", nil)
	keyEntry := PassRow("CHAIN", Part("/signing_key_event/commitments/stellar"), "held", nil)
	keyEvent := NewRow("CHAIN", specdata.ReasonNotAttributable, Part("/signing_key_event"), "info", nil)
	entropy := PassRow("SOURCES", Part("/witnesses/entropy_stellar"), "read", nil)
	for _, c := range []struct {
		name   string
		typ    string
		rows   []Row
		cands  []RowCandidate
		row    string
		absent string
	}{
		{"sources_disagree and sources_incomplete", "item", []Row{disagree, incomplete, keyEntry, keyEvent, entropy},
			[]RowCandidate{stellarAt("SOURCES", "/witnesses/entropy_stellar", windowBase)},
			"[window_not_established null info WINDOW]",
			`{"submitted_before":[{"code":"sources_disagree","part":"/commitments/bitcoin"},{"code":"sources_incomplete","part":"/commitments/stellar"}]}`},
		{"a block proof", "block", []Row{genesis, disagree, keyEntry},
			[]RowCandidate{stellarAt("CHAIN", "/commitments/stellar", windowBase)},
			"[null null pass WINDOW]", `{"submitted_after":[]}`},
		{"an empty beacon", "beacon", nil, nil,
			"[window_not_established null info WINDOW]",
			`{"submitted_after":[],"submitted_before":[]}`},
		{"an empty beacon with a submitted-before edge", "beacon", nil,
			[]RowCandidate{stellarAt("CHAIN", "/commitments/stellar", windowBase)},
			"[null null pass WINDOW]", `{"submitted_after":[]}`},
		{"a block proof with a passing row and no candidate", "block", []Row{PassRow("CHAIN", Part("/commitments/bitcoin"), "held", nil), incomplete}, nil,
			"[window_not_established null info WINDOW]",
			`{"submitted_after":[],"submitted_before":[{"code":null,"part":"/commitments/bitcoin"},{"code":"sources_incomplete","part":"/commitments/stellar"}]}`},
	} {
		run := windowRun(c.typ, windowBase*1000, c.rows, c.cands...)
		Window(run)
		rows := windowRows(run)
		absent, err := jcs.Encode(rows[0].Values["absent"])
		if got := rowSummary(rows); got != c.row || err != nil || string(absent) != c.absent || run.Failed() {
			t.Errorf("%s: %s, absent %s, %v, failed %v\nwant %s, %s", c.name, got, absent, err, run.Failed(), c.row, c.absent)
		}
		if c.typ != "item" && run.SubmittedAfter != nil {
			t.Errorf("%s: a submitted-after edge %+v", c.name, run.SubmittedAfter)
		}
	}
}

// The time inside the subject's id (SPEC.md section 11.3): its first 12
// hex digits, an unsigned count of milliseconds, compared in milliseconds
// with the edges after a pass only; later than the submitted-before edge,
// or earlier than the submitted-after edge, adds info
// recorded_time_outside_window at the id's pointer, /subject/record/id or
// /block/record/id for a block or a beacon, with the id's time written as
// a NISTTIME, or as its count past the year 9999; on an edge, or inside,
// adds none, nor does any row but a pass; and an id that spells no time is
// internal_error. Mutants killed: the id time compared in seconds (1 ms
// past the submitted-before second gives no row); compared with >= (an id
// on the edge gives one); the submitted-after check left out (the early
// id gives no row); the pointer /subject/record/id for every type (the
// block's row is there); the id's hex read with a sign (an id with "+"
// passes); the hyphen not skipped (no id reads); the hyphen's place left
// unchecked (the id with no hyphen reads a time); the id time always
// written as a count (the NISTTIME is not written); a NIST instant
// truncated to its second (the id 1 ms before the edge, 250 ms into its
// second, reads as inside).
func TestWindow_RecordedTime(t *testing.T) {
	s := windowBase
	before := stellarAt("CHAIN", "/commitments/stellar", s+100)
	after := nistAt("/witnesses/entropy_nist", s*1000+250)
	for _, c := range []struct {
		name  string
		typ   string
		id    string
		cands []RowCandidate
		want  string
		value string
	}{
		{"on the submitted-before edge", "item", idAt((s + 100) * 1000), []RowCandidate{before, after}, "[null null pass WINDOW]", ""},
		{"1 ms later", "item", idAt((s+100)*1000 + 1), []RowCandidate{before, after},
			"[null null pass WINDOW] [recorded_time_outside_window /subject/record/id info WINDOW]",
			`{"edge":"submitted_before","id_time":"2026-09-21T14:15:00.001Z"}`},
		{"on the submitted-after edge", "item", idAt(s*1000 + 250), []RowCandidate{before, after}, "[null null pass WINDOW]", ""},
		{"1 ms earlier", "item", idAt(s*1000 + 249), []RowCandidate{before, after},
			"[null null pass WINDOW] [recorded_time_outside_window /subject/record/id info WINDOW]",
			`{"edge":"submitted_after","id_time":"2026-09-21T14:13:20.249Z"}`},
		{"a block, later", "block", idAt((s + 101) * 1000), []RowCandidate{before},
			"[null null pass WINDOW] [recorded_time_outside_window /block/record/id info WINDOW]",
			`{"edge":"submitted_before","id_time":"2026-09-21T14:15:01.000Z"}`},
		{"a beacon, past the year 9999", "beacon", "ffffffff-ffff-7000-8000-000000000000", []RowCandidate{before},
			"[null null pass WINDOW] [recorded_time_outside_window /block/record/id info WINDOW]",
			`{"edge":"submitted_before","id_time":281474976710655}`},
		{"not established", "item", idAt(0), []RowCandidate{after}, "[window_not_established null info WINDOW]", ""},
		{"inverted", "item", idAt(0), []RowCandidate{stellarAt("CHAIN", "/commitments/stellar", s), after}, "[window_inverted null fail WINDOW]", ""},
		{"no time", "item", "+0000000-0000-7000-8000-000000000000", []RowCandidate{before},
			"[internal_error null fail WINDOW] [prior_failure null skip EXPECTED_HASH]", ""},
		{"no hyphen", "item", "0000000000007000-8000-000000000000", []RowCandidate{before},
			"[internal_error null fail WINDOW] [prior_failure null skip EXPECTED_HASH]", ""},
	} {
		run := windowRun(c.typ, 0, nil, c.cands...)
		run.SubjectID = c.id
		Window(run)
		var tail []Row
		for _, row := range run.Rows {
			if row.Step == "WINDOW" || row.Step == "EXPECTED_HASH" {
				tail = append(tail, row)
			}
		}
		got := rowSummary(tail)
		value := ""
		if n := len(tail); n == 2 && tail[1].Code == specdata.ReasonRecordedTimeOutsideWindow {
			data, err := jcs.Encode(tail[1].Values)
			if err != nil {
				t.Fatal(err)
			}
			value = string(data)
		}
		if got != c.want || value != c.value {
			t.Errorf("%s: %s %s\nwant %s %s", c.name, got, value, c.want, c.value)
		}
	}
}

// The stated assumptions of SPEC.md sections 11.1 and 11.2, word for word,
// for each basis and edge, the Stellar one for the submitted-before edge
// among them; NIST states none for the submitted-before edge. Mutants
// killed: any assumption's text changed (it differs from SPEC.md's); the
// Stellar submitted-after assumption given to the submitted-before edge
// (the submitted-before text differs).
func TestWindow_TheStatedAssumptions(t *testing.T) {
	for _, c := range []struct{ basis, edge, want string }{
		{stellarCloseBasis, edgeSubmittedBefore, "Stellar validators did not set the ledger's close time behind real time; a close time set ahead of real time only makes this edge later"},
		{bitcoinMedianBasis, edgeSubmittedBefore, "at least 6 of blocks height + 1 to height + 10 carry honest times"},
		{nistTimeBasis, edgeSubmittedAfter, "NIST does not release outputValue before timeStamp"},
		{stellarCloseBasis, edgeSubmittedAfter, "the network did not set the close time ahead of real time"},
		{bitcoinMedianBasis, edgeSubmittedAfter, "at least 6 of the 11 header times ending at that block are honest"},
	} {
		if got := assumptionOf(c.basis, c.edge); got != c.want {
			t.Errorf("%s, %s: %q, want %q", c.basis, c.edge, got, c.want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Fatal("NIST's submitted-before assumption did not panic")
		}
	}()
	assumptionOf(nistTimeBasis, edgeSubmittedBefore)
}

// Each edge's eight members (SPEC.md section 11.5) and the WINDOW row's
// values under an open gate (section 11.3), exactly: an edge resting on
// two sources lists both in their order, each with the fetched_at of its
// own read, and takes cached, fetched_at and source from the first; the
// source is named scheme://host[:port], without the URL's path; the
// instant is a TIME for Stellar and a NISTTIME for NIST; the candidates
// are listed in row order, each with the edge it is a candidate for; and
// both edges are on the run. Mutants killed: an edge's sources the first
// counted source only (one source listed); fetched_at taken from the row's
// candidate for every source (both sources give the first's); source and
// fetched_at taken from the last source (they differ from the first's);
// the source printed with its path (/accounts appears); the "edge" member
// left off a listed candidate; the run's edges left nil; the candidates
// left out of the values (none listed); a NIST instant written as a TIME
// (its milliseconds are lost); cached written true.
func TestWindow_TheEdgeObjects(t *testing.T) {
	before := rowCandidate("CHAIN", "/commitments/stellar", stellarCloseBasis, ClassTIME, windowBase+100,
		"https://horizon-a.example:8443/accounts", "https://horizon-b.example")
	after := nistAt("/witnesses/entropy_nist", windowBase*1000+250)
	run := windowRun("item", windowBase*1000+500, nil, before, after)
	Window(run)
	rows := windowRows(run)
	values, err := jcs.Encode(rows[0].Values)
	if err != nil {
		t.Fatal(err)
	}
	beforeEdge := `{"assumption":"Stellar validators did not set the ledger's close time behind real time; a close time set ahead of real time only makes this edge later",` +
		`"basis":"stellar_ledger_close_read","cached":false,"fetched_at":"2026-09-21T14:13:20.001Z","instant":"2026-09-21T14:15:00Z","part":"/commitments/stellar",` +
		`"source":"https://horizon-a.example:8443","sources":[{"cached":false,"fetched_at":"2026-09-21T14:13:20.001Z","source":"https://horizon-a.example:8443"},` +
		`{"cached":false,"fetched_at":"2026-09-21T14:13:20.002Z","source":"https://horizon-b.example"}]}`
	afterEdge := `{"assumption":"NIST does not release outputValue before timeStamp","basis":"nist_pulse_time_read","cached":false,"fetched_at":"2026-09-21T14:13:20.001Z",` +
		`"instant":"2026-09-21T14:13:20.250Z","part":"/witnesses/entropy_nist","source":"https://beacon.nist.gov",` +
		`"sources":[{"cached":false,"fetched_at":"2026-09-21T14:13:20.001Z","source":"https://beacon.nist.gov"}]}`
	var decoded struct {
		Absent     map[string]any    `json:"absent"`
		Candidates []json.RawMessage `json:"candidates"`
		Edges      map[string]json.RawMessage
	}
	if err := json.Unmarshal(values, &decoded); err != nil {
		t.Fatal(err)
	}
	canon := func(raw []byte) string {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	withEdge := func(edge, name string) string {
		var v map[string]any
		if err := json.Unmarshal([]byte(edge), &v); err != nil {
			t.Fatal(err)
		}
		v["edge"] = name
		data, _ := json.Marshal(v)
		return string(data)
	}
	switch {
	case len(decoded.Absent) != 0 || len(decoded.Candidates) != 2 || len(decoded.Edges) != 2:
		t.Fatalf("values %s", values)
	case canon(decoded.Edges["submitted_before"]) != canon([]byte(beforeEdge)):
		t.Errorf("submitted_before %s\nwant %s", decoded.Edges["submitted_before"], beforeEdge)
	case canon(decoded.Edges["submitted_after"]) != canon([]byte(afterEdge)):
		t.Errorf("submitted_after %s\nwant %s", decoded.Edges["submitted_after"], afterEdge)
	case canon(decoded.Candidates[0]) != withEdge(beforeEdge, "submitted_before") || canon(decoded.Candidates[1]) != withEdge(afterEdge, "submitted_after"):
		t.Errorf("candidates %s", values)
	}
	if run.SubmittedBefore == nil || run.SubmittedAfter == nil {
		t.Fatal("the run holds no edge")
	}
	for name, e := range map[string]*Edge{"submitted_before": run.SubmittedBefore, "submitted_after": run.SubmittedAfter} {
		data, err := jcs.Encode(e.Value())
		if err != nil || canon(data) != canon(decoded.Edges[name]) {
			t.Errorf("%s: the run's edge %s, the row's %s, %v", name, data, decoded.Edges[name], err)
		}
	}
}
