// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds step 12 of SPEC.md section 10.2, WINDOW: the submission
// window of section 11, drawn as a fixed function of the reads this run
// made, its rows (section 11.3) and its two edge objects (section 11.5).
//
// The candidates are those CHAIN and SOURCES kept on the run, in row order
// ([Run.Candidates]): each row that passed with one gives one, the one the
// sources it counted share (SPEC.md section 10.4).
//
//	| Edge             | Its candidates                                                                   | The edge is            |
//	|------------------|----------------------------------------------------------------------------------|------------------------|
//	| submitted_before | the CHAIN rows of the top-level entries, /commitments/C (SPEC.md section 11.1); never an entry at /signing_key_event/commitments | the earliest candidate |
//	| submitted_after  | the SOURCES rows, one for each entropy object read and found equal (section 11.2) | the latest candidate   |
//
// Of two candidates at the same instant, the edge is the first in row
// order (sections 11.1 and 11.2). Instants compare exactly, at the
// precision each source gives: milliseconds for NIST and for the time
// inside an id, and seconds for Stellar and Bitcoin (section 11.3), so a
// Stellar or Bitcoin second compares as its first millisecond. A block
// proof carries no entropy object, and an empty beacon none in its
// observations, so neither has a submitted-after edge (section 11.2). No
// time the proof carries is an edge (section 11): an edge is always a
// value read in this run.
//
// Each candidate's basis and stated assumption are those of its row's table
// in SPEC.md sections 11.1 and 11.2, word for word ([assumptionOf]); the
// Stellar basis, stellar_ledger_close_read, states one assumption for the
// submitted-before edge and another for the submitted-after edge.
//
// WINDOW gives the first of these rows that applies, with a null part
// (SPEC.md section 11.3):
//
//  1. a row has failed: skip prior_failure;
//  2. both edges exist and the submitted-after edge is later than the
//     submitted-before edge: fail window_inverted;
//  3. the submitted-before edge does not exist: info
//     window_not_established;
//  4. otherwise: pass.
//
// Absence never fails: a row skipped for any reason, sources_disagree and
// sources_incomplete among them, gives no candidate, so it leaves an edge
// absent, with its reason, and adds no failure. When the row is pass and
// the time inside the subject's id is later than the submitted-before
// edge, or earlier than the submitted-after edge, WINDOW adds one row, info
// recorded_time_outside_window, at the pointer of that id. That time is the
// leftmost 48 bits of the 16 bytes the id's 32 hex digits spell, its first
// 12 hex digits with the hyphen skipped, an unsigned big-endian count of
// milliseconds since the Unix epoch (unix_ts_ms, RFC 9562 section 5.7); no
// other bit is a time. It is RECORDED, so the row changes no label. A
// subject id SHAPE has held to UUID7 always spells one; one that does not
// is a fault, internal_error (section 10.5).
//
// The row's values give every candidate the run read, each with the edge
// it is a candidate for, each edge that exists and, for each edge that does
// not, its reason: the part and code of each CHAIN or SOURCES row that
// could have given a candidate, a list that is empty when the proof carries
// none, as for a block proof's submitted-after edge (section 11.3). Every
// row WINDOW gives carries them, under an open gate or a closed one, skip
// prior_failure included; the WINDOW row of a run that has ended is
// EndRun's, skip prior_failure with no values (section 10.2).
//
// An edge object is {assumption, basis, cached, fetched_at, instant, part,
// source, sources} (section 11.5, [Edge.Value]). The run keeps the edges
// WINDOW drew on [Run.SubmittedAfter] and [Run.SubmittedBefore]; a report
// whose label is failed writes both null (section 10.5). A run whose
// external gate is closed read nothing (section 10.4, "Offline"), so it has
// no candidate and draws neither edge: its row is skip prior_failure when a
// row has failed, and otherwise info window_not_established. A run that has
// ended passes through, since its WINDOW row is already skip prior_failure.

package verifier

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// The names of the two edges (SPEC.md section 11.5).
const (
	edgeSubmittedAfter  = "submitted_after"
	edgeSubmittedBefore = "submitted_before"
)

// Edge is an edge of the submission window (SPEC.md section 11.5): the
// Candidate that gave it, its stated Assumption, the Part of the CHAIN or
// SOURCES row whose read gave it, and Sources, the sources it rests on, in
// the order of their network's sources, never empty.
type Edge struct {
	Assumption string
	Candidate  Candidate
	Part       string
	Sources    []EdgeSource
}

// EdgeSource is one source an edge rests on, with its read that gave the
// candidate: Source, the source as a report names it, scheme://host[:port];
// Cached, whether that read's answer came from a cache, always false, since
// the CLI caches nothing; and FetchedAt, when that read came back.
type EdgeSource struct {
	Source    string
	Cached    bool
	FetchedAt time.Time
}

// Value is the edge as a report writes it, {assumption, basis, cached,
// fetched_at, instant, part, source, sources} (SPEC.md section 11.5):
// cached, fetched_at and source are those of the first of sources, and
// instant is written at its source's precision, as a TIME for Stellar and
// Bitcoin and as a NISTTIME for NIST.
func (e Edge) Value() map[string]any {
	sources := make([]any, 0, len(e.Sources))
	for _, s := range e.Sources {
		sources = append(sources, s.value())
	}
	first := e.Sources[0]
	instant, _ := FormatTime(e.Candidate.Class, e.Candidate.Instant)
	return map[string]any{
		"assumption": e.Assumption,
		"basis":      e.Candidate.Basis,
		"cached":     first.Cached,
		"fetched_at": fetchedAt(first.FetchedAt),
		"instant":    instant,
		"part":       e.Part,
		"source":     first.Source,
		"sources":    sources,
	}
}

// value is the source as an edge's sources write it, {cached, fetched_at,
// source}.
func (s EdgeSource) value() map[string]any {
	return map[string]any{"cached": s.Cached, "fetched_at": fetchedAt(s.FetchedAt), "source": s.Source}
}

// assumptionOf is the stated assumption of basis for the edge named edge,
// word for word as the tables of SPEC.md sections 11.1 and 11.2 give it.
// It panics for a pair neither table holds, NIST for the submitted-before
// edge among them: a defect in the caller.
func assumptionOf(basis, edge string) string {
	switch {
	case edge == edgeSubmittedBefore && basis == stellarCloseBasis:
		return "Stellar validators did not set the ledger's close time behind real time; a close time set ahead of real time only makes this edge later"
	case edge == edgeSubmittedBefore && basis == bitcoinMedianBasis:
		return "at least 6 of blocks height + 1 to height + 10 carry honest times"
	case edge == edgeSubmittedAfter && basis == nistTimeBasis:
		return "NIST does not release outputValue before timeStamp"
	case edge == edgeSubmittedAfter && basis == stellarCloseBasis:
		return "the network did not set the close time ahead of real time"
	case edge == edgeSubmittedAfter && basis == bitcoinMedianBasis:
		return "at least 6 of the 11 header times ending at that block are honest"
	}
	panic(fmt.Sprintf("verifier: SPEC.md section 11 states no assumption of %s for the %s edge", basis, edge))
}

// edgeOf is the edge named edge that candidate gives: its sources are those
// its row counted, in their order, each with the fetched_at of its read
// that gave the candidate. It panics for a candidate that rests on no
// source, which no row of SPEC.md section 10.4 gives: a defect in the
// caller.
func edgeOf(candidate RowCandidate, edge string) Edge {
	if len(candidate.Counted) == 0 {
		panic(fmt.Sprintf("verifier: the candidate of %s %s rests on no source", candidate.Step, candidate.Part))
	}
	sources := make([]EdgeSource, 0, len(candidate.Counted))
	for _, counted := range candidate.Counted {
		at := candidate.Candidate.FetchedAt
		if c := counted.Outcome.Candidate; c != nil {
			at = c.FetchedAt
		}
		sources = append(sources, EdgeSource{Source: printedURL(counted.Source.Source.URL), FetchedAt: at})
	}
	return Edge{Assumption: assumptionOf(candidate.Candidate.Basis, edge), Candidate: candidate.Candidate, Part: candidate.Part, Sources: sources}
}

// windowCandidate is a candidate of the run, and the edge it is one for.
type windowCandidate struct {
	edge      string
	candidate RowCandidate
}

// windowCandidates are the run's candidates for the two edges, in row
// order: a CHAIN candidate of a top-level entry for the submitted-before
// edge, and a SOURCES candidate for the submitted-after edge.
func windowCandidates(run *Run) []windowCandidate {
	var out []windowCandidate
	for _, c := range run.Candidates {
		switch {
		case c.Step == "CHAIN" && isTopLevelEntry(c.Part):
			out = append(out, windowCandidate{edgeSubmittedBefore, c})
		case c.Step == "SOURCES":
			out = append(out, windowCandidate{edgeSubmittedAfter, c})
		}
	}
	return out
}

// drawEdges draws both edges from candidates, in row order: the earliest
// submitted-before candidate and the latest submitted-after candidate, the
// first of those at one instant, or nil where an edge has none.
func drawEdges(candidates []windowCandidate) (after, before *Edge) {
	for _, c := range candidates {
		at := c.candidate.Candidate.Milliseconds()
		switch c.edge {
		case edgeSubmittedBefore:
			if before == nil || at < before.Candidate.Milliseconds() {
				e := edgeOf(c.candidate, c.edge)
				before = &e
			}
		case edgeSubmittedAfter:
			if after == nil || at > after.Candidate.Milliseconds() {
				e := edgeOf(c.candidate, c.edge)
				after = &e
			}
		}
	}
	return after, before
}

// Window is step 12, WINDOW (see the file's doc).
func Window(run *Run) {
	if run.Ended {
		return
	}
	if _, closed := closedGate(run, "WINDOW"); closed {
		windowClosed(run)
		return
	}
	candidates := windowCandidates(run)
	after, before := drawEdges(candidates)
	run.SubmittedAfter, run.SubmittedBefore = after, before
	values := windowValues(run, candidates, after, before)
	switch {
	case run.Failed():
		run.AddRow(NewRow("WINDOW", specdata.ReasonPriorFailure, nil, "Not drawn: a check of this run failed, so it gives no submission window.", values))
	case after != nil && before != nil && after.Candidate.Milliseconds() > before.Candidate.Milliseconds():
		run.AddRow(NewRow("WINDOW", specdata.ReasonWindowInverted, nil, fmt.Sprintf(
			"The submitted-after edge, %s, is later than the submitted-before edge, %s, so the reads of this run contradict each other.",
			edgeText(*after), edgeText(*before)), values))
	case before == nil:
		run.AddRow(NewRow("WINDOW", specdata.ReasonWindowNotEstablished, nil, notEstablishedMessage(after), values))
	default:
		windowPass(run, after, *before, values)
	}
}

// windowClosed is WINDOW under a closed gate (see the file's doc).
func windowClosed(run *Run) {
	values := map[string]any{
		"absent":     map[string]any{edgeSubmittedAfter: absentReasons(run, "SOURCES"), edgeSubmittedBefore: absentReasons(run, "CHAIN")},
		"candidates": []any{},
		"edges":      map[string]any{},
	}
	if run.Failed() {
		run.AddRow(NewRow("WINDOW", specdata.ReasonPriorFailure, nil, "Not drawn: a check of this run failed, so it gives no submission window.", values))
		return
	}
	run.AddRow(NewRow("WINDOW", specdata.ReasonWindowNotEstablished, nil,
		"No submitted-before edge was drawn: this run is offline, so it read no chain or source, and draws no edge.", values))
}

// windowPass adds the pass row and, when the time inside the subject's id
// lies outside the window, the info row recorded_time_outside_window at
// that id (see the file's doc).
func windowPass(run *Run, after *Edge, before Edge, values map[string]any) {
	idTime, ok := idTimeOf(run.SubjectID)
	if !ok {
		run.AddRow(NewRow("WINDOW", specdata.ReasonInternalError, nil,
			"The verifier could not read the time inside the subject's id, which SHAPE held to be a UUIDv7.",
			map[string]any{"error": "the subject id does not begin with the 12 hex digits of a unix_ts_ms"}))
		run.EndRun()
		return
	}
	message := "The submission window: submitted before " + edgeText(before) + "."
	if after != nil {
		message = "The submission window: submitted after " + edgeText(*after) + ", and submitted before " + edgeText(before) + "."
	}
	run.AddRow(PassRow("WINDOW", nil, message, values))
	var outside, relation string
	switch {
	case idTime > before.Candidate.Milliseconds():
		outside, relation = edgeSubmittedBefore, "later than the submitted-before edge"
	case after != nil && idTime < after.Candidate.Milliseconds():
		outside, relation = edgeSubmittedAfter, "earlier than the submitted-after edge"
	default:
		return
	}
	written := idTimeValue(idTime)
	text := fmt.Sprintf("%d milliseconds after the Unix epoch", idTime)
	if s, isText := written.(string); isText {
		text = s
	}
	whose := "declared by the signer (not attributed to Truestamp)"
	if run.Attribution == AttributionTruestamp {
		whose = "recorded by Truestamp"
	}
	run.AddRow(NewRow("WINDOW", specdata.ReasonRecordedTimeOutsideWindow, Part(subjectIDPointer(run.Type).Encode()), fmt.Sprintf(
		"The time inside the subject's id, %s, is %s. That time is %s, not confirmed in this run, and changes no label.", text, relation, whose),
		map[string]any{"edge": outside, "id_time": written}))
}

// idTimeOf is the time inside id, unix_ts_ms: the unsigned big-endian
// value of its first 12 hex digits, the hyphen after the eighth skipped
// (SPEC.md section 11.3). It gives false for an id that does not begin
// with them.
func idTimeOf(id string) (int64, bool) {
	if len(id) < 13 || id[8] != '-' {
		return 0, false
	}
	digits := id[:8] + id[9:13]
	for _, d := range digits {
		if !strings.ContainsRune("0123456789abcdefABCDEF", d) {
			return 0, false
		}
	}
	ms, err := strconv.ParseInt(digits, 16, 64)
	return ms, err == nil
}

// idTimeValue writes the time inside an id for a row's values: an RFC 3339
// UTC time with milliseconds, as a NISTTIME is written, or, for a time past
// the year 9999 that its 48 bits can hold, its count of milliseconds.
func idTimeValue(ms int64) any {
	if text, ok := FormatTime(ClassNISTTIME, ms); ok {
		return text
	}
	return strictjson.Number(float64(ms))
}

// windowValues are the WINDOW row's values under an open gate (see the
// file's doc).
func windowValues(run *Run, candidates []windowCandidate, after, before *Edge) map[string]any {
	listed := make([]any, 0, len(candidates))
	for _, c := range candidates {
		value := edgeOf(c.candidate, c.edge).Value()
		value["edge"] = c.edge
		listed = append(listed, value)
	}
	edges, absent := map[string]any{}, map[string]any{}
	if after != nil {
		edges[edgeSubmittedAfter] = after.Value()
	} else {
		absent[edgeSubmittedAfter] = absentReasons(run, "SOURCES")
	}
	if before != nil {
		edges[edgeSubmittedBefore] = before.Value()
	} else {
		absent[edgeSubmittedBefore] = absentReasons(run, "CHAIN")
	}
	return map[string]any{"absent": absent, "candidates": listed, "edges": edges}
}

// edgeText names an edge in a message: its instant, its basis and its part,
// each whole.
func edgeText(e Edge) string {
	instant, _ := FormatTime(e.Candidate.Class, e.Candidate.Instant)
	return fmt.Sprintf("%s (%s, from %s)", instant, e.Candidate.Basis, e.Part)
}

// notEstablishedMessage is the message of info window_not_established
// under an open gate.
func notEstablishedMessage(after *Edge) string {
	message := "No submitted-before edge was drawn: no CHAIN row of a top-level commitment entry gave a candidate in this run."
	if after != nil {
		message += " The submitted-after edge is " + edgeText(*after) + "."
	}
	return message
}

// absentReasons are the part and code of each row of step that could have
// given a candidate: under CHAIN, the rows of the top-level entries. A row
// that passed has no code, which the values write null.
func absentReasons(run *Run, step string) []any {
	reasons := []any{}
	for _, row := range run.Rows {
		if row.Step != step || row.Part == nil {
			continue
		}
		if step == "CHAIN" && !isTopLevelEntry(*row.Part) {
			continue
		}
		var code any
		if row.Code != "" {
			code = string(row.Code)
		}
		reasons = append(reasons, map[string]any{"code": code, "part": *row.Part})
	}
	return reasons
}

// isTopLevelEntry reports whether part is a top-level commitment entry,
// /commitments/C.
func isTopLevelEntry(part string) bool {
	chain, ok := strings.CutPrefix(part, "/commitments/")
	return ok && (chain == ChainBitcoin || chain == ChainStellar)
}
