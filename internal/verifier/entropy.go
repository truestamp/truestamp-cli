// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds step 11 of SPEC.md section 10.2, SOURCES, under an open
// external gate (SPEC.md sections 10.2, step 11, and 10.4, "Sources",
// "Answers", "Entropy objects", "Several sources", "Rows" and "What a row
// read"; section 11.2 for the candidates).
//
// Each entropy object P carries, /subject/entropy in an entropy type, each
// /witnesses/entropy_X in an item and each /observations/entropy_X/entropy
// in a beacon, gets one row, at its pointer, in document order, whether or
// not a read is made. An object is read only when its entropy_hash was
// recomputed and matched at DERIVE: a DERIVE row at the part that compares
// it passed, /subject/record/entropy_hash in an entropy type,
// /observations/entropy_X/record/entropy_hash in a beacon and LINK_WITNESS's
// /subject/record/witness_hashes/entropy_X in an item. An object whose
// entropy_hash did not match depends on a row that failed, so its row is
// skip prior_failure and no read is made of it (SPEC.md section 12.1,
// prior_failure); under an open gate no row before CHAIN failed, so every
// object is read there.
//
// An object's rule is made once for each source of the run's entropy pin
// of its source ([Run.SourcesFor] with [PinEntropy]), never the commitment
// pin's, whose network can differ; NIST's Randomness Beacon is one source
// with no network and no identity read, at its compiled-in address
// ([NISTBeaconURL]), under the policy first (SPEC.md section 9). Every read
// goes through [Run.ReadSource] and the run's [ChainReader], and the
// sources' outcomes are combined under the network's source policy
// ([CombineSources]); the row's values are what it read
// ([CombinedOutcome.Values]). The proof chooses neither a source nor a
// network, and no read goes to an address the proof carries. Each rule,
// against one source as if it were the only one:
//
//	| Source  | Read                                   | If found, these must equal what was read, or fail entropy_mismatch | Candidate, for the submitted-after edge   | If not found |
//	|---------|----------------------------------------|---------------------------------------------------------------------|-------------------------------------------|--------------|
//	| NIST    | nist_pulse at chainIndex, pulseIndex   | outputValue (NISTHEX), timeStamp (NISTTIME, as an instant)           | the timeStamp read, nist_pulse_time_read, milliseconds | skip not_held_by_source |
//	| Stellar | stellar_ledger at sequence             | hash (HEX64), closed_at (TIME, as an instant)                        | the closed_at read, stellar_ledger_close_read, seconds | the root, read: skip not_yet_indexed if sequence > latest - STELLAR_INDEX_MARGIN, else skip not_held_by_source |
//	| Bitcoin | bitcoin_block by hash                  | height (INT), time (INT, the header's nTime)                         | its median_time_past, an INT a TIME can write, bitcoin_median_time_read, seconds | skip not_held_by_source |
//
// The Bitcoin block is read by its hash and need not be on the best chain:
// a witness from a block that was later orphaned is still a sound
// candidate, and a source that no longer holds it gives not_held_by_source,
// never a failure. The Stellar root is read only after the ledger is not
// found, and only its latest is read (its elder is not); a root that
// answers not found, which no Horizon gives (section 13.2), is
// not_held_by_source, since section 10.4 gives no outcome for that answer.
// Hashes compare as the bytes they spell, which for two values of one
// class is their text, and times as instants. Every network follows these
// rules alike.
//
// An answer is judged on the fields the rule reads at that point, whatever
// else it lacks or gives in another form (SPEC.md section 10.4,
// "Answers"): the pulse on outputValue and timeStamp, the ledger on hash
// and closed_at, the block on height, time and median_time_past, and the
// root on latest. A field that contradicts the object fails the outcome
// even when another is unusable, and an answer with an unusable field and
// no contradiction is skip source_unreachable, its notes naming the
// fields. An unreachable read is skip source_unreachable. A row that
// passed keeps its candidate on the run ([Run.Candidates]), after CHAIN's,
// in row order, with the sources it rests on, for WINDOW (SPEC.md sections
// 11.2 and 11.5).
//
// A fault while reading what a rule compares, an object SHAPE should have
// refused, is internal_error at SOURCES, part null, and ends the run
// before any read, as a fault at CHAIN does.

package verifier

import (
	"fmt"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// NISTBeaconURL is the base of NIST's Randomness Beacon 2.0 API, the one
// address the NIST read goes to (SPEC.md section 10.4, "Common sources").
const NISTBeaconURL = "https://beacon.nist.gov/beacon/2.0"

// nistBeaconBase is the base [nistSet] reads NIST's Randomness Beacon
// from: NISTBeaconURL in every build but the test binary of the cmd
// package, which sets it at link time (-ldflags -X) to a loopback server,
// so that no test of the built command reads NIST. It is a string
// variable only so that the linker can set it. nistSet is its one reader;
// no flag, configuration key, environment variable or document reaches it.
var nistBeaconBase = NISTBeaconURL

// SourceNIST is the name of the NIST entropy source (SPEC.md section 3.5).
const SourceNIST = "nist"

// The bases of an entropy candidate (SPEC.md sections 11.2 and 11.5).
const nistTimeBasis = "nist_pulse_time_read"

// nistSet is the NIST beacon as a set of sources: its one source,
// confirmed, since it has no identity read, under the policy first.
func nistSet() SourceSet {
	beacon := SourceRecord{
		Source: Source{Chain: SourceNIST, Provider: source.NISTBeacon, URL: nistBeaconBase},
		From:   FromDefault,
		Status: SourceConfirmed,
	}
	return SourceSet{Chain: SourceNIST, Policy: SourcePolicy{Kind: SourcePolicyFirst}, Sources: []SourceRecord{beacon}}
}

// entropyEntry is what a rule reads of one entropy object: its pointer,
// its source, the part of the DERIVE row that compares its entropy_hash,
// and the payload values the rule reads, each in its class: NIST's
// chainIndex, pulseIndex, outputValue and timeStamp (Unix milliseconds);
// Stellar's sequence, hash and closed_at (Unix seconds); and Bitcoin's
// hash, height and time.
type entropyEntry struct {
	at      Pointer
	source  string
	matched Pointer

	chainIndex, pulseIndex int64
	outputValue            string
	timeStamp              int64

	sequence int64
	hash     string
	closedAt int64

	height, time int64
}

// entropyEntryOf reads object, an entropy object of proof, or gives an
// error for one SHAPE should have refused.
func entropyEntryOf(proof map[string]any, object layoutNode) (entropyEntry, error) {
	e := entropyEntry{at: object.at, source: object.name, matched: entropyHashPart(proof, object)}
	payload, _ := object.node["payload"].(map[string]any)
	ok := false
	switch object.name {
	case SourceNIST:
		pulse, _ := payload["pulse"].(map[string]any)
		var isChain, isPulse, isOutput bool
		e.chainIndex, isChain = intField(pulse, "chainIndex")
		e.pulseIndex, isPulse = intField(pulse, "pulseIndex")
		e.outputValue, isOutput = nistHexField(pulse, "outputValue")
		var err error
		e.timeStamp, err = ParseTime(ClassNISTTIME, pulse["timeStamp"])
		ok = isChain && isPulse && isOutput && err == nil
	case ChainStellar:
		var isSequence, isHash bool
		e.sequence, isSequence = intField(payload, "sequence")
		e.hash, isHash = hex64Field(payload, "hash")
		var err error
		e.closedAt, err = ParseTime(ClassTIME, payload["closed_at"])
		ok = isSequence && isHash && err == nil
	case ChainBitcoin:
		var isHash, isHeight, isTime bool
		e.hash, isHash = hex64Field(payload, "hash")
		e.height, isHeight = intField(payload, "height")
		e.time, isTime = intField(payload, "time")
		ok = isHash && isHeight && isTime
	}
	if !ok {
		return entropyEntry{}, fmt.Errorf("the entropy object at %s has no payload in its classes", object.at.Encode())
	}
	return e, nil
}

// entropyHashPart is the part of the DERIVE row that compares the
// entropy_hash of object, an entropy object of proof (see the file's doc).
func entropyHashPart(proof map[string]any, object layoutNode) Pointer {
	switch proof["type"] {
	case "item":
		return Pointer{Name("subject"), Name("record"), Name("witness_hashes"), Name("entropy_" + object.name)}
	case "beacon":
		return under(object.at[:len(object.at)-1], "record", "entropy_hash")
	}
	return Pointer{Name("subject"), Name("record"), Name("entropy_hash")}
}

// matchedAtDerive reports whether a DERIVE row at e's matched part passed.
func matchedAtDerive(run *Run, e entropyEntry) bool {
	part := e.matched.Encode()
	for _, row := range run.Rows {
		if row.Step == "DERIVE" && row.Part != nil && *row.Part == part && row.Status == StatusPass {
			return true
		}
	}
	return false
}

// sourcesOpen is SOURCES under an open gate (see the file's doc).
func sourcesOpen(run *Run) {
	objects := entropyObjects(run.Proof)
	SortBy(objects, func(o layoutNode) Pointer { return o.at })
	entries := make([]entropyEntry, 0, len(objects))
	for _, object := range objects {
		e, err := entropyEntryOf(run.Proof, object)
		if err != nil {
			run.AddRow(NewRow("SOURCES", specdata.ReasonInternalError, nil,
				"The verifier could not read a value this proof's entropy object carries.", map[string]any{"error": err.Error()}))
			run.EndRun()
			return
		}
		entries = append(entries, e)
	}
	var rows []chainRow
	for _, e := range entries {
		if !matchedAtDerive(run, e) {
			rows = append(rows, chainRow{at: e.at, row: NewRow("SOURCES", specdata.ReasonPriorFailure, Part(e.at.Encode()),
				"Not read: this entropy object's entropy_hash did not match at DERIVE.", nil)})
			continue
		}
		set := nistSet()
		if e.source != SourceNIST {
			set = sourcesOf(run, PinEntropy, e.source)
		}
		combined := CombineSources(set, entropyRule(run, set.Network, e))
		made := chainRow{at: e.at, row: combined.Row("SOURCES", e.at, entropyMessage(e, set, combined))}
		if combined.Candidate != nil {
			made.candidate = &RowCandidate{Step: "SOURCES", Part: e.at.Encode(), Candidate: *combined.Candidate, Counted: combined.Counted}
		}
		rows = append(rows, made)
	}
	for _, r := range rows {
		run.AddRow(r.row)
		if r.candidate != nil {
			run.Candidates = append(run.Candidates, *r.candidate)
		}
	}
}

// entropyRule is the rule of e against one source of network, "" for
// NIST's.
func entropyRule(run *Run, network string, e entropyEntry) RowRule {
	return func(record SourceRecord) SourceOutcome {
		s := &entropySource{chainSource{run: run, record: record, network: network}}
		switch e.source {
		case SourceNIST:
			return s.pulse(e)
		case ChainStellar:
			return s.ledger(e)
		}
		return s.block(e)
	}
}

// entropySource is one source an entropy rule reads.
type entropySource struct{ chainSource }

// pulse is NIST's rule (see the file's doc).
func (s *entropySource) pulse(e entropyEntry) SourceOutcome {
	read := s.read(source.KindNISTPulse, map[string]any{"chainIndex": intValue(e.chainIndex), "pulseIndex": intValue(e.pulseIndex)})
	switch read.Outcome {
	case ReadNotFound:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, map[string]any{"chainIndex": intValue(e.chainIndex), "pulseIndex": intValue(e.pulseIndex)})
	case ReadUnreachable:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
	}
	output, isOutput := nistHexField(read.Value, "outputValue")
	stamp, err := ParseTime(ClassNISTTIME, read.Value["timeStamp"])
	return s.judged([]fieldCheck{
		{"outputValue", compared(output, isOutput, e.outputValue)},
		{"timeStamp", compared(stamp, err == nil, e.timeStamp)},
	}, &Candidate{Basis: nistTimeBasis, Class: ClassNISTTIME, Instant: stamp, FetchedAt: read.FetchedAt})
}

// ledger is Stellar's rule (see the file's doc).
func (s *entropySource) ledger(e entropyEntry) SourceOutcome {
	read := s.read(source.KindStellarLedger, map[string]any{"sequence": intValue(e.sequence)})
	switch read.Outcome {
	case ReadNotFound:
		return s.missingLedger(e)
	case ReadUnreachable:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
	}
	hash, isHash := hex64Field(read.Value, "hash")
	closed, err := ParseTime(ClassTIME, read.Value["closed_at"])
	return s.judged([]fieldCheck{
		{"hash", compared(hash, isHash, e.hash)},
		{"closed_at", compared(closed, err == nil, e.closedAt)},
	}, &Candidate{Basis: stellarCloseBasis, Class: ClassTIME, Instant: closed, FetchedAt: read.FetchedAt})
}

// missingLedger judges a ledger the source does not hold, from the latest
// ledger its root names.
func (s *entropySource) missingLedger(e entropyEntry) SourceOutcome {
	root := s.read(source.KindStellarRoot, nil)
	switch root.Outcome {
	case ReadNotFound:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, map[string]any{"sequence": intValue(e.sequence)})
	case ReadUnreachable:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
	}
	latest, isLatest := intField(root.Value, "latest")
	notes := map[string]any{"latest": intValue(latest), "sequence": intValue(e.sequence)}
	switch {
	case !isLatest:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, map[string]any{"unusable": []any{"latest"}})
	case e.sequence > latest-specdata.StellarIndexMargin:
		return s.outcome(specdata.ReasonNotYetIndexed, nil, notes)
	}
	return s.outcome(specdata.ReasonNotHeldBySource, nil, notes)
}

// block is Bitcoin's rule (see the file's doc).
func (s *entropySource) block(e entropyEntry) SourceOutcome {
	read := s.read(source.KindBitcoinBlock, map[string]any{"hash": e.hash})
	switch read.Outcome {
	case ReadNotFound:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, map[string]any{"hash": e.hash})
	case ReadUnreachable:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
	}
	height, isHeight := intField(read.Value, "height")
	blockTime, isTime := intField(read.Value, "time")
	median, isMedian := intField(read.Value, "median_time_past")
	usable := fieldHolds
	if _, isTimeClass := FormatTime(ClassTIME, median); !isMedian || !isTimeClass {
		usable = fieldUnusable
	}
	return s.judged([]fieldCheck{
		{"height", compared(height, isHeight, e.height)},
		{"time", compared(blockTime, isTime, e.time)},
		{"median_time_past", usable},
	}, &Candidate{Basis: bitcoinMedianBasis, Class: ClassTIME, Instant: median, FetchedAt: read.FetchedAt})
}

// judged is the outcome of a found answer from the checks of the fields
// the rule read: fail entropy_mismatch naming each field that contradicts
// the object, whatever else is unusable; else skip source_unreachable
// naming each unusable field; else pass with candidate.
func (s *entropySource) judged(checks []fieldCheck, candidate *Candidate) SourceOutcome {
	contradicts, unusable := decideFields(checks)
	switch {
	case contradicts != nil:
		return s.outcome(specdata.ReasonEntropyMismatch, nil, map[string]any{"differs": contradicts})
	case unusable != nil:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, map[string]any{"unusable": unusable})
	}
	return s.outcome("", candidate, nil)
}

// nistHexField is the field name of value as a NISTHEX (SPEC.md section
// 2.4), 128 uppercase hex digits, and whether it is one.
func nistHexField(value map[string]any, name string) (string, bool) {
	text, ok := value[name].(string)
	if !ok || len(text) != 128 || strings.Trim(text, "0123456789ABCDEF") != "" {
		return "", false
	}
	return text, true
}

// entropyMessage is the message of e's row, from its combined outcome and,
// for source_unreachable, the read that settled each source's outcome.
func entropyMessage(e entropyEntry, set SourceSet, combined CombinedOutcome) string {
	network, read := entropyNetwork(e, set), entropyWhat(e)
	whole := network
	if e.source != SourceNIST {
		whole = asWhole(network, combined)
	}
	switch combined.Code {
	case "":
		instant, _ := FormatTime(combined.Candidate.Class, combined.Candidate.Instant)
		switch e.source {
		case SourceNIST:
			return fmt.Sprintf("Read from %s: %s holds this entropy object's outputValue and its timeStamp, %s.", network, read, instant)
		case ChainStellar:
			return fmt.Sprintf("Read from %s: %s has this entropy object's hash and closed at %s, as it says.", network, read, instant)
		}
		return fmt.Sprintf("Read from %s: %s is at this entropy object's height %d with its time, and its median time past is %s.", network, read, e.height, instant)
	case specdata.ReasonEntropyMismatch:
		return fmt.Sprintf("Contradicted by %s: %s differs from this entropy object.", network, read)
	case specdata.ReasonNotYetIndexed:
		return fmt.Sprintf("Not confirmed: %s does not hold %s yet, which is beyond, or within %d ledgers of, the latest ledger it holds.", whole, read, specdata.StellarIndexMargin)
	case specdata.ReasonNotHeldBySource:
		switch e.source {
		case SourceNIST:
			return fmt.Sprintf("Not confirmed: %s does not hold %s. Absence is no contradiction.", whole, read)
		case ChainStellar:
			return fmt.Sprintf("Not confirmed: the history %s keeps does not hold %s. Absence is no contradiction.", whole, read)
		}
		return fmt.Sprintf("Not confirmed: %s does not hold %s, which need not be on the best chain. Absence is no contradiction.", whole, read)
	case specdata.ReasonSourcesDisagree:
		return fmt.Sprintf("Not confirmed: the sources of %s disagree on %s. A disagreement is no failure.", network, read)
	case specdata.ReasonSourcesIncomplete:
		return fmt.Sprintf("Not confirmed: fewer sources of %s than its policy %s asks for reached an outcome on %s.", network, set.Policy, read)
	}
	how, alike := entropySettled(combined)
	switch {
	case e.source == SourceNIST:
		return fmt.Sprintf("Not confirmed: %s gave no usable answer about %s.", network, read)
	case !alike:
		return fmt.Sprintf("Not confirmed: no source of %s gave a usable answer to every read this row makes.", network)
	case how == "":
		return fmt.Sprintf("Not confirmed: no source of %s had its identity confirmed, so none was read.", network)
	case how == source.KindStellarRoot:
		return fmt.Sprintf("Not confirmed: no source of %s holds %s, and none gave a usable answer about the ledgers it holds.", network, read)
	}
	return fmt.Sprintf("Not confirmed: no source of %s gave a usable answer about %s.", network, read)
}

// entropyNetwork names the network of e's sources as a message does.
func entropyNetwork(e entropyEntry, set SourceSet) string {
	switch e.source {
	case SourceNIST:
		return "NIST's Randomness Beacon"
	case ChainStellar:
		return fmt.Sprintf("the pinned Stellar %s network", set.Network)
	}
	return fmt.Sprintf("the pinned Bitcoin %s network", set.Network)
}

// entropyWhat names what e's rule reads, each value whole.
func entropyWhat(e entropyEntry) string {
	switch e.source {
	case SourceNIST:
		return fmt.Sprintf("the pulse at chainIndex %d and pulseIndex %d", e.chainIndex, e.pulseIndex)
	case ChainStellar:
		return fmt.Sprintf("ledger %d", e.sequence)
	}
	return "the block " + e.hash
}

// entropySettled is the kind of the last read of each source combined
// asked, "" for a source asked nothing, and whether every source's was the
// same.
func entropySettled(combined CombinedOutcome) (string, bool) {
	of := func(o SourceOutcome) string {
		if len(o.Reads) == 0 {
			return ""
		}
		kind, _ := o.Reads[len(o.Reads)-1].Read["kind"].(string)
		return kind
	}
	how := of(combined.Asked[0].Outcome)
	for _, asked := range combined.Asked[1:] {
		if of(asked.Outcome) != how {
			return how, false
		}
	}
	return how, true
}
