// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds step 10 of SPEC.md section 10.2, CHAIN, under an open
// external gate (SPEC.md sections 10.2, step 10, 10.3, the key-history
// rows, and 10.4, "Sources", "Several sources", "Rows" and "What a row
// read"), and the seam its reads go through.
//
// Each commitment entry whose chain's rule is in this package gets one
// row, at the pointer of its entry: its rule of section 10.4, made once
// for each source of the run's commitment pin of that chain
// ([Run.SourcesFor]), every read through [Run.ReadSource] and the run's
// [ChainReader], and the sources' outcomes combined under the network's
// source policy ([CombineSources]); its values are what the row read
// ([CombinedOutcome.Values]). The rows are in document order, and a row
// that passed with a candidate keeps it on the run ([Run.Candidates]), in
// row order, with the sources it rests on, for WINDOW (SPEC.md sections
// 11.1 and 11.5). The proof chooses neither a source nor a network: every
// read goes to a source of the run's pin, which POLICY has already held
// every network member to (section 10.4, "Sources").
//
// The rules here are Stellar's (chain_stellar.go). The Bitcoin rule of
// section 10.4, for a Bitcoin commitment entry and for genesis.bitcoin, is
// not in this package yet, so under an open gate those rows are not
// given.
//
// KEY_EVENT_COMMITMENT is the last row of CHAIN, at /signing_key_event,
// when P carries signing_key_event (SPEC.md section 10.3). Its inputs are
// the CHAIN rows of the entries under /signing_key_event/commitments and,
// for a genesis event, the row of
// /signing_key_event/block/record/metadata/genesis/bitcoin. Under an open
// gate its status is the first of these that applies: an input failed,
// fail with the code of the first failing input in document order; an
// input was skipped, skip with the code of the first skipped input in
// document order; otherwise pass when the attribution is truestamp, else
// info not_attributable. It is given only when every input has its row, so
// a key event with a Bitcoin input gets it once the Bitcoin rule is here.
//
// A fault while reading what a rule compares, an entry SHAPE should have
// refused or an epoch record that cannot be hashed, is internal_error at
// CHAIN, part null, and ends the run, as a fault at DERIVE does.

package verifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// ChainReader makes one read of CHAIN or SOURCES of a source, live (SPEC.md
// section 10.4): read names it as a replay record does, {kind, network,
// ...} (section 13.2), and the reader gives its Outcome, with Value, the
// fields of a found answer, and Reason, for an unreachable one, which
// never names the source's URL. It is the seam every such read goes
// through: [DefaultChainReader] reads over HTTP, and a conformance test
// answers from a case's replay file. [Run.ReadSource] sets the answer's
// Read and FetchedAt.
type ChainReader func(source Source, read map[string]any) SourceRead

// DefaultChainReader returns the reader of each chain read over HTTP,
// through the default transport and internal/verifier/source's readers:
// each read builds its own SingleRead, within [SourceReadDeadline], when
// it is made.
func DefaultChainReader() ChainReader {
	return func(s Source, read map[string]any) SourceRead {
		reads := newSingleRead(nil, SourceReadDeadline)
		answer := source.Read(context.Background(), reads, s.Provider, s.URL, read)
		return SourceRead{Outcome: ReadOutcome(answer.Outcome), Value: answer.Value, Reason: answer.Reason}
	}
}

// readChain is the answer to read of record's source, through the run's
// chain reader, one answer per read and source within the run
// ([Run.ReadSource]). A run with no chain reader gets no usable answer,
// and so does an answer whose outcome is none of the three of SPEC.md
// section 10.4; a found answer with no value holds no field; an
// unreachable answer keeps the reader's reason, which says why (section
// 13.2), and no value.
func (r *Run) readChain(record SourceRecord, read map[string]any) SourceRead {
	return r.ReadSource(record, read, func() SourceRead {
		if r.Config.ReadChain == nil {
			return SourceRead{Outcome: ReadUnreachable, Reason: "this run has no reader of the chains"}
		}
		answer := r.Config.ReadChain(record.Source, read)
		switch answer.Outcome {
		case ReadFound:
			if answer.Value == nil {
				answer.Value = map[string]any{}
			}
		case ReadNotFound:
			answer.Value, answer.Reason = nil, ""
		case ReadUnreachable:
			answer.Value = nil
		default:
			return SourceRead{Outcome: ReadUnreachable, Reason: "the reader gave no answer of SPEC.md section 10.4"}
		}
		return answer
	})
}

// RowCandidate is the candidate of a CHAIN or SOURCES row that passed with
// one (SPEC.md sections 11.1, 11.2 and 11.5): the row's Step and Part, the
// Candidate its counted sources share, and Counted, those sources, the one
// whose outcome the row took under first and every source that passed
// under at_least n or all, in the order of the network's sources.
type RowCandidate struct {
	Step      string
	Part      string
	Candidate Candidate
	Counted   []AskedSource
}

// chainRow is one row CHAIN made under an open gate, and the candidate it
// keeps, if any.
type chainRow struct {
	at        Pointer
	row       Row
	candidate *RowCandidate
}

// chainOpen is CHAIN under an open gate (see the file's doc).
func chainOpen(run *Run) {
	var entries []stellarEntry
	for _, entry := range commitmentEntries(run.Proof) {
		if entry.name != ChainStellar {
			continue
		}
		e, err := stellarEntryOf(run.Proof, entry)
		if err != nil {
			run.AddRow(NewRow("CHAIN", specdata.ReasonInternalError, nil,
				"The verifier could not read a value this proof's commitment entry carries.", map[string]any{"error": err.Error()}))
			run.EndRun()
			return
		}
		entries = append(entries, e)
	}
	var rows []chainRow
	for _, e := range entries {
		set := sourcesOf(run, PinCommitment, ChainStellar)
		combined := CombineSources(set, stellarRule(run, set.Network, e))
		made := chainRow{at: e.at, row: combined.Row("CHAIN", e.at, stellarMessage(e, set, combined))}
		if combined.Candidate != nil {
			made.candidate = &RowCandidate{Step: "CHAIN", Part: e.at.Encode(), Candidate: *combined.Candidate, Counted: combined.Counted}
		}
		rows = append(rows, made)
	}
	SortBy(rows, func(r chainRow) Pointer { return r.at })
	for _, r := range rows {
		run.AddRow(r.row)
		if r.candidate != nil {
			run.Candidates = append(run.Candidates, *r.candidate)
		}
	}
	if row, ok := keyEventCommitment(run, rows); ok {
		run.AddRow(row)
	}
}

// sourcesOf is the run's sources of pin of chain. It panics for a run with
// none, which no online run has once CONFIGURATION passes: a defect in the
// caller.
func sourcesOf(run *Run, pin, chain string) SourceSet {
	set, ok := run.SourcesFor(pin, chain)
	if !ok {
		panic(fmt.Sprintf("verifier: CHAIN found no source of the %s pin of %s", pin, chainName(chain)))
	}
	return set
}

// keyEventInputs are the parts of KEY_EVENT_COMMITMENT's inputs P carries,
// in document order: each entry under /signing_key_event/commitments, and
// the genesis.bitcoin of /signing_key_event/block.
func keyEventInputs(proof map[string]any) []Pointer {
	var parts []Pointer
	for _, entry := range commitmentEntries(proof) {
		if entry.at[0] == Name("signing_key_event") {
			parts = append(parts, entry.at)
		}
	}
	for _, genesis := range genesisBitcoin(proof) {
		if genesis.at[0] == Name("signing_key_event") {
			parts = append(parts, genesis.at)
		}
	}
	SortBy(parts, func(p Pointer) Pointer { return p })
	return parts
}

// keyEventCommitment is KEY_EVENT_COMMITMENT under an open gate, from the
// rows CHAIN made (see the file's doc), and whether it is given: it is not
// when P carries no signing_key_event, or when an input has no row.
func keyEventCommitment(run *Run, rows []chainRow) (Row, bool) {
	if _, ok := run.Proof["signing_key_event"].(map[string]any); !ok {
		return Row{}, false
	}
	var inputs []Row
	for _, part := range keyEventInputs(run.Proof) {
		i := -1
		for j, r := range rows {
			if Compare(r.at, part) == 0 {
				i = j
			}
		}
		if i < 0 {
			return Row{}, false
		}
		inputs = append(inputs, rows[i].row)
	}
	values := map[string]any{"inputs": keyEventValues(inputs)}
	part := Part(keyEventCommitmentPart)
	for _, status := range []Status{StatusFail, StatusSkip} {
		for _, input := range inputs {
			if input.Status == status {
				return NewRow("CHAIN", input.Code, part, keyEventMessage(input), values), true
			}
		}
	}
	if run.Attribution == AttributionTruestamp {
		return PassRow("CHAIN", part, "Each commitment of the signing key's event is confirmed on its chain.", values), true
	}
	return NewRow("CHAIN", specdata.ReasonNotAttributable, part,
		"Each commitment of the signing key's event is confirmed on its chain. This run is not attributed to Truestamp, so the row states where the key came from and no more.", values), true
}

// keyEventValues are the part, status and code of each input of
// KEY_EVENT_COMMITMENT, in document order.
func keyEventValues(inputs []Row) []any {
	out := make([]any, 0, len(inputs))
	for _, input := range inputs {
		value := map[string]any{"part": *input.Part, "status": string(input.Status)}
		if input.Code != "" {
			value["code"] = string(input.Code)
		}
		out = append(out, value)
	}
	return out
}

// keyEventMessage is KEY_EVENT_COMMITMENT's message when it takes input's
// status and code.
func keyEventMessage(input Row) string {
	if input.Status == StatusFail {
		return fmt.Sprintf("A commitment of the signing key's event is contradicted by its chain: %s at %s.", input.Code, *input.Part)
	}
	return fmt.Sprintf("A commitment of the signing key's event is not confirmed: %s at %s.", input.Code, *input.Part)
}

// The judgement of one field a rule uses (SPEC.md section 10.4,
// "Answers"): it holds, it contradicts the proof, or it is unusable,
// absent from the answer or given in another form.
type fieldResult int

const (
	fieldHolds fieldResult = iota
	fieldContradicts
	fieldUnusable
)

// fieldCheck is one field's judgement, by the field's name.
type fieldCheck struct {
	name   string
	result fieldResult
}

// decideFields gives the names of the fields that contradict the proof
// and of those that are unusable. An answer that shows a contradiction
// fails its row even when it lacks a field another check uses, so a
// caller reads the contradictions first.
func decideFields(checks []fieldCheck) (contradicts, unusable []any) {
	for _, c := range checks {
		switch c.result {
		case fieldContradicts:
			contradicts = append(contradicts, c.name)
		case fieldUnusable:
			unusable = append(unusable, c.name)
		}
	}
	return contradicts, unusable
}

// compared is the judgement of a field read as got, usable when ok, and
// held to want.
func compared[T comparable](got T, ok bool, want T) fieldResult {
	switch {
	case !ok:
		return fieldUnusable
	case got != want:
		return fieldContradicts
	}
	return fieldHolds
}

// intField is the field name of value as an INT (SPEC.md section 2.4),
// and whether it is one.
func intField(value map[string]any, name string) (int64, bool) {
	num, isNumber := value[name].(strictjson.Number)
	n, isInt := num.Int64()
	return n, isNumber && isInt && n >= 0
}

// boolField is the field name of value as a boolean, and whether it is
// one.
func boolField(value map[string]any, name string) (bool, bool) {
	b, ok := value[name].(bool)
	return b, ok
}

// stringIn is the field name of value as one of the strings of allowed,
// and whether it is one.
func stringIn(value map[string]any, name string, allowed ...string) (string, bool) {
	text, ok := value[name].(string)
	for _, a := range allowed {
		if ok && text == a {
			return text, true
		}
	}
	return "", false
}

// hex64Field is the field name of value as a HEX64 (SPEC.md section 2.4),
// and whether it is one.
func hex64Field(value map[string]any, name string) (string, bool) {
	text, ok := value[name].(string)
	if !ok || len(text) != 64 || strings.Trim(text, "0123456789abcdef") != "" {
		return "", false
	}
	return text, true
}
