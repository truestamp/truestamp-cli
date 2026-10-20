// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file is the frame every CHAIN and SOURCES row read under an open
// external gate runs through (SPEC.md section 10.4, "Several sources",
// "What a row read", "Answers" and "Caches", and section 10.5, the values
// of a row): a row's rule, its whole rule of section 10.4 against one
// source as if that source were the only one, is made once for each source
// of the row's network, and the sources' outcomes are combined under the
// network's source policy into the row's outcome, its values and its
// candidate. A row is never made from the sources' reads, only from their
// outcomes, since sources legitimately differ on reads that move.
//
// Each source's outcome ([SourceOutcome]) is conclusive, pass, with its
// candidate when the row gives one, or fail, with chain_mismatch or
// entropy_mismatch; or inconclusive, a skip: source_unreachable,
// not_held_by_source, not_yet_indexed or chain_not_final. A source whose
// identity read at step 0 got no usable answer is skip source_unreachable
// with no read, and its rule is not made (SPEC.md section 10.2, step 0).
//
// [CombineSources] gives the row's outcome ([CombinedOutcome]) under the
// policy:
//
//   - first: the sources are asked in their order, and the row takes the
//     first conclusive outcome; no source after it is asked. When none is
//     conclusive, the row is what items 4 and 5 below give, with n = 1;
//   - at_least n, and all, which is at_least the number of sources: every
//     source is asked, and the row is the first of these that applies:
//     1. one source passes and another fails, or two pass with candidates
//     at different instants (instants compared as instants, SPEC.md
//     section 11.3): skip sources_disagree;
//     2. at least n pass: pass, with the candidate they share;
//     3. at least n fail: fail, with the row's code;
//     4. no source is conclusive, and every source skipped with one code:
//     skip with that code;
//     5. otherwise: skip sources_incomplete.
//
// With one source, every policy gives that source's outcome. A passing
// row's candidate rests on the sources it counted: the one source whose
// outcome it took under first, and every source that passed under
// at_least n or all (SPEC.md section 11.5).
//
// The row's values ([CombinedOutcome.Values]) are SPEC.md section 10.4's
// "what a row read": policy (first, at_least or all), required (the n of
// the policy: 1 under first, the number of sources under all), outcomes,
// one object for each source the row asked, in the order of the network's
// sources, each {cached, candidate, code, fetched_at, reads, source,
// status}, and candidate, {basis, instant}, exactly when the row passed
// with one; and notes, by source, for what a rule noted beside its outcome,
// such as the elder ledger a not_held_by_source names. An outcome's code is
// absent exactly when its status is pass, and its candidate present exactly
// when it passed with one. Its cached and fetched_at are those of the read
// whose answer settled its outcome: its candidate read when it has a
// candidate, its identity read when it was asked nothing else, and
// otherwise its last read, since a rule reads nothing after its outcome is
// settled. Its reads are every read the rule made of it, in the order
// made, each {answer, cached, fetched_at, read} with read and answer in
// the form of a replay record (SPEC.md section 13.2). Its source is the
// source as a message prints it, its scheme, host and port ([printedURL]),
// as CONFIGURATION's values print it, since a hosted node's path may hold
// an API key and a report is made to be shared.
//
// The CLI caches nothing: every read is made live, so every cached is
// false, and a read's fetched_at is when its answer, or its failure, came
// back, by the verifier's clock, the one thing that clock stamps (SPEC.md
// section 11). A source's answers are its own: [Run.ReadSource] gives a
// rule one answer for every row of the run that makes the same read of the
// same source, and never another source's (SPEC.md section 10.4,
// "Caches").

package verifier

import (
	"fmt"
	"slices"
	"time"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// readClock is the verifier's clock, which stamps each read's fetched_at
// and nothing else (SPEC.md section 11). The package's tests replace it.
var readClock = time.Now

// stamp is the clock's reading for a read's fetched_at, in UTC.
func stamp() time.Time { return readClock().UTC() }

// ReadOutcome is a read's outcome (SPEC.md section 10.4, "Answers").
type ReadOutcome string

// The outcomes of a read.
const (
	// ReadFound: the source returned the object asked for.
	ReadFound ReadOutcome = "found"
	// ReadNotFound: the source answered that it holds no such object.
	ReadNotFound ReadOutcome = "not_found"
	// ReadUnreachable: there is no usable answer.
	ReadUnreachable ReadOutcome = "unreachable"
)

// SourceRead is one read a rule made of one source, in the form of a
// replay record (SPEC.md section 13.2): Read names the read, its kind, its
// network for a Stellar or Bitcoin source and its parameters, each spelled
// as the proof spells the value it comes from; Outcome, with Value, the
// fields the read uses, for a found answer and Reason, which says why and
// never names the source's URL, for an unreachable one; and FetchedAt,
// when the answer came back. [Run.ReadSource] sets Read and FetchedAt.
// Each member of Read and Value is a value internal/jcs can write.
type SourceRead struct {
	Read      map[string]any
	Outcome   ReadOutcome
	Value     map[string]any
	Reason    string
	FetchedAt time.Time
}

// value is the read as a row's values hold it: {answer, cached,
// fetched_at, read}, the answer {outcome, value}, {outcome} or {outcome,
// reason}.
func (r SourceRead) value() map[string]any {
	answer := map[string]any{"outcome": string(r.Outcome)}
	switch r.Outcome {
	case ReadFound:
		answer["value"] = r.Value
	case ReadUnreachable:
		answer["reason"] = r.Reason
	}
	return map[string]any{"answer": answer, "cached": false, "fetched_at": fetchedAt(r.FetchedAt), "read": r.Read}
}

// fetchedAt writes a read's fetched_at as an RFC 3339 UTC time with
// milliseconds, as a NISTTIME is written (SPEC.md section 2.4).
func fetchedAt(t time.Time) string {
	text, _ := FormatTime(ClassNISTTIME, t.UnixMilli())
	return text
}

// Candidate is a candidate for an edge (SPEC.md sections 11.1, 11.2 and
// 11.5): Basis, stellar_ledger_close_read, bitcoin_median_time_read or
// nist_pulse_time_read; Instant, at its source's precision, in Class's
// unit: Unix seconds for TIME (Stellar and Bitcoin) and Unix milliseconds
// for NISTTIME (NIST); and FetchedAt, when the read that gave it came
// back.
type Candidate struct {
	Basis     string
	Class     TimeClass
	Instant   int64
	FetchedAt time.Time
}

// Milliseconds is the candidate's instant in Unix milliseconds, so two
// candidates compare as instants (SPEC.md section 11.3).
func (c Candidate) Milliseconds() int64 {
	if c.Class == ClassTIME {
		return c.Instant * 1000
	}
	return c.Instant
}

// value is the candidate as a row's values hold it, {basis, instant}, the
// instant written in its class.
func (c Candidate) value() map[string]any {
	instant, _ := FormatTime(c.Class, c.Instant)
	return map[string]any{"basis": c.Basis, "instant": instant}
}

// SourceOutcome is one source's outcome on a row (SPEC.md section 10.4,
// "Several sources"): Status pass, fail or skip; Code, "" exactly when
// Status is pass, chain_mismatch or entropy_mismatch for a failure, and
// source_unreachable, not_held_by_source, not_yet_indexed or
// chain_not_final for a skip; Candidate, present exactly when the source
// passed with one; Reads, every read the rule made of the source, in the
// order made; and Notes, anything else the rule noted, which a row's
// values hold under notes, each a value internal/jcs can write.
type SourceOutcome struct {
	Status    Status
	Code      specdata.Reason
	Candidate *Candidate
	Reads     []SourceRead
	Notes     map[string]any
}

// inconclusiveCodes are the codes of a source's skip (SPEC.md section
// 10.4, "Several sources").
var inconclusiveCodes = []specdata.Reason{
	specdata.ReasonSourceUnreachable, specdata.ReasonNotHeldBySource,
	specdata.ReasonNotYetIndexed, specdata.ReasonChainNotFinal,
}

// conclusive reports whether the outcome is pass or fail.
func (o SourceOutcome) conclusive() bool {
	return o.Status == StatusPass || o.Status == StatusFail
}

// check panics for an outcome no rule of section 10.4 gives: a defect in
// the rule.
func (o SourceOutcome) check() {
	ok := false
	switch o.Status {
	case StatusPass:
		ok = o.Code == ""
	case StatusFail:
		ok = o.Code == specdata.ReasonChainMismatch || o.Code == specdata.ReasonEntropyMismatch
	case StatusSkip:
		ok = slices.Contains(inconclusiveCodes, o.Code)
	}
	if !ok || (o.Candidate != nil && o.Status != StatusPass) {
		panic(fmt.Sprintf("verifier: a source's outcome %s %q with candidate %v is none SPEC.md section 10.4 gives", o.Status, o.Code, o.Candidate))
	}
}

// RowRule is a row's whole rule of SPEC.md section 10.4 made against one
// source, as if it were the only one: every read it makes goes to that
// source, through [Run.ReadSource], and that source's own answers decide
// each branch. It is made only for a source whose identity was confirmed.
type RowRule func(source SourceRecord) SourceOutcome

// AskedSource is one source a row asked, and its outcome.
type AskedSource struct {
	Source  SourceRecord
	Outcome SourceOutcome
}

// CombinedOutcome is a row's outcome from its network's sources (see the
// file's doc): its Status and Code, Code "" exactly when Status is pass;
// its Candidate, the one its counted sources share, when it passed with
// one; Counted, the sources that candidate, or the pass, rests on; Policy
// and Required, the policy's n; and Asked, each source the row asked, in
// the order of the network's sources.
type CombinedOutcome struct {
	Status    Status
	Code      specdata.Reason
	Candidate *Candidate
	Counted   []AskedSource
	Policy    SourcePolicy
	Required  int
	Asked     []AskedSource
}

// CombineSources makes rule once for each source of set the policy asks,
// in the set's order, and combines their outcomes under the set's policy
// (see the file's doc). A source whose identity read got no usable answer
// is skip source_unreachable, with no read, and rule is not made for it.
// It panics for a set with no source, which no network an online run pins
// has once CONFIGURATION passes, and for an outcome rule gives that no
// rule of section 10.4 gives: defects in the caller.
func CombineSources(set SourceSet, rule RowRule) CombinedOutcome {
	if len(set.Sources) == 0 {
		panic(fmt.Sprintf("verifier: %s's %s network has no source", chainName(set.Chain), set.Network))
	}
	combined := CombinedOutcome{Policy: set.Policy, Required: set.Required()}
	ask := func(record SourceRecord) SourceOutcome {
		outcome := SourceOutcome{Status: StatusSkip, Code: specdata.ReasonSourceUnreachable}
		if record.Status != SourceUnreachable {
			outcome = rule(record)
		}
		outcome.check()
		combined.Asked = append(combined.Asked, AskedSource{record, outcome})
		return outcome
	}
	if set.Policy.Kind == SourcePolicyFirst {
		for _, record := range set.Sources {
			if outcome := ask(record); outcome.conclusive() {
				return combined.taking(outcome, combined.Asked[len(combined.Asked)-1:])
			}
		}
		return combined.inconclusive()
	}
	var passes, fails []AskedSource
	for _, record := range set.Sources {
		switch outcome := ask(record); outcome.Status {
		case StatusPass:
			passes = append(passes, combined.Asked[len(combined.Asked)-1])
		case StatusFail:
			fails = append(fails, combined.Asked[len(combined.Asked)-1])
		}
	}
	switch {
	case disagree(passes, fails):
		return combined.skip(specdata.ReasonSourcesDisagree)
	case len(passes) >= combined.Required:
		return combined.taking(passes[0].Outcome, passes)
	case len(fails) >= combined.Required:
		return combined.taking(fails[0].Outcome, fails)
	}
	return combined.inconclusive()
}

// disagree is rule 1: a pass beside a failure, or two passes whose
// candidates are at different instants.
func disagree(passes, fails []AskedSource) bool {
	if len(passes) > 0 && len(fails) > 0 {
		return true
	}
	var instants []int64
	for _, p := range passes {
		if c := p.Outcome.Candidate; c != nil && !slices.Contains(instants, c.Milliseconds()) {
			instants = append(instants, c.Milliseconds())
		}
	}
	return len(instants) > 1
}

// taking is the row taking outcome's status, code and candidate, resting
// on counted.
func (c CombinedOutcome) taking(outcome SourceOutcome, counted []AskedSource) CombinedOutcome {
	c.Status, c.Code, c.Candidate, c.Counted = outcome.Status, outcome.Code, outcome.Candidate, counted
	return c
}

// skip is the row skipping with code, resting on no source.
func (c CombinedOutcome) skip(code specdata.Reason) CombinedOutcome {
	c.Status, c.Code = StatusSkip, code
	return c
}

// inconclusive is items 4 and 5: every source asked skipped, with one code
// or not, or too few were conclusive.
func (c CombinedOutcome) inconclusive() CombinedOutcome {
	code := c.Asked[0].Outcome.Code
	for _, asked := range c.Asked {
		if asked.Outcome.conclusive() || asked.Outcome.Code != code {
			return c.skip(specdata.ReasonSourcesIncomplete)
		}
	}
	return c.skip(code)
}

// Row is the combined outcome as the row of step at part, with message
// and [CombinedOutcome.Values].
func (c CombinedOutcome) Row(step string, part Pointer, message string) Row {
	return NewRow(step, c.Code, Part(part.Encode()), message, c.Values())
}

// Values are the row's values: what it read (see the file's doc).
func (c CombinedOutcome) Values() map[string]any {
	outcomes := make([]any, 0, len(c.Asked))
	var notes []any
	for _, asked := range c.Asked {
		outcomes = append(outcomes, asked.value())
		if len(asked.Outcome.Notes) > 0 {
			note := map[string]any{"source": printedURL(asked.Source.Source.URL)}
			for name, value := range asked.Outcome.Notes {
				note[name] = value
			}
			notes = append(notes, note)
		}
	}
	values := map[string]any{
		"outcomes": outcomes,
		"policy":   string(c.Policy.Kind),
		"required": strictjson.Number(float64(c.Required)),
	}
	if c.Candidate != nil {
		values["candidate"] = c.Candidate.value()
	}
	if notes != nil {
		values["notes"] = notes
	}
	return values
}

// value is one source's object in a row's outcomes.
func (a AskedSource) value() map[string]any {
	o := a.Outcome
	reads := make([]any, 0, len(o.Reads))
	for _, r := range o.Reads {
		reads = append(reads, r.value())
	}
	settled := a.Source.FetchedAt
	switch {
	case o.Candidate != nil:
		settled = o.Candidate.FetchedAt
	case len(o.Reads) > 0:
		settled = o.Reads[len(o.Reads)-1].FetchedAt
	}
	value := map[string]any{
		"cached":     false,
		"fetched_at": fetchedAt(settled),
		"reads":      reads,
		"source":     printedURL(a.Source.Source.URL),
		"status":     string(o.Status),
	}
	if o.Code != "" {
		value["code"] = string(o.Code)
	}
	if o.Candidate != nil {
		value["candidate"] = o.Candidate.value()
	}
	return value
}

// sourceReadKey finds one read of one source among a run's answers: the
// source's chain, provider, URL and network, and the read's JCS.
type sourceReadKey struct {
	chain, provider, url, network, read string
}

// ReadSource gives the answer of the read named read, of source, through
// do, which makes the read live and gives its Outcome, Value and Reason:
// the answer this run already has for the same read of the same source
// when it has one, so every row of a run that needs that read takes one
// answer (SPEC.md section 10.4, "Caches"), and otherwise do's answer, its
// Read set to read and its FetchedAt to the clock's reading when do
// returns, which the run keeps when it is found or not found. An
// unreachable answer is never kept, so a later row that needs the read
// makes it again ("no cache keeps the answer", section 10.4, "Answers"). No
// answer of one source ever stands as another's, since the source is part
// of the key. A read whose name JCS cannot write is not kept. The reads of
// the identities are not made here: step 0 makes each once.
func (r *Run) ReadSource(source SourceRecord, read map[string]any, do func() SourceRead) SourceRead {
	made := func() SourceRead {
		answer := do()
		answer.Read, answer.FetchedAt = read, stamp()
		return answer
	}
	encoded, err := jcs.Encode(read)
	if err != nil {
		return made()
	}
	key := sourceReadKey{source.Source.Chain, string(source.Source.Provider), source.Source.URL, source.Source.Network, string(encoded)}
	if answer, ok := r.answers[key]; ok {
		return answer
	}
	answer := made()
	if answer.Outcome == ReadUnreachable {
		return answer
	}
	if r.answers == nil {
		r.answers = map[sourceReadKey]SourceRead{}
	}
	r.answers[key] = answer
	return answer
}
