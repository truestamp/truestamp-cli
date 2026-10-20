// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the external gate of SPEC.md section 10.2 and the rows
// steps 10 and 11, CHAIN and SOURCES, give when it is closed (sections
// 10.2, the external gate, 10.3, the key-history rows, and 10.4, "Rows" and
// "Offline").
//
// The external gate is decided once, when DERIVE ends ([decideExternalGate],
// which [Derive] calls): it is open when the caller did not ask for an
// offline run ([Config.Online]) and no row so far is fail. Otherwise it is
// closed: prior_failure when a row so far is fail, and offline when none
// is. Only the caller opens it; nothing in the document can. A failure at
// CHAIN never closes it for SOURCES, since nothing decides it again.
//
// CHAIN gives one row for each commitment entry, at /commitments/C and
// /signing_key_event/commitments/C, and one for each genesis.bitcoin a
// block record carries, each at the pointer of its entry or object, in
// document order, and then, when P carries signing_key_event, the row
// KEY_EVENT_COMMITMENT at /signing_key_event, the last of CHAIN. SOURCES
// gives one row for each entropy object, /subject/entropy,
// /witnesses/entropy_X or /observations/entropy_X/entropy, at its pointer,
// in document order. The positions are layout.go's.
//
// When the gate is closed, every one of those rows is skip prior_failure
// or skip offline, as the gate is, KEY_EVENT_COMMITMENT included (section
// 10.3, item 1 of its status: skip prior_failure when a row of POLICY or
// DERIVE failed, which is when the gate closed on a failure, since a
// failure before POLICY ends the run). A skip offline row's message names
// the read that would confirm it, each value it names written whole; its
// values are empty, since the run read nothing. When the gate is open, the
// rows are those of each chain's and each source's rule of section 10.4,
// each made once for each source of its network and combined under the
// network's source policy (source_outcomes.go): CHAIN's are chain.go's,
// Stellar's rule (chain_stellar.go) and Bitcoin's, for a commitment entry
// and for genesis.bitcoin (chain_bitcoin.go), and SOURCES' are
// entropy.go's, NIST's, Stellar's and Bitcoin's rules for an entropy
// object. A run that has ended passes through, since its CHAIN and SOURCES
// rows are already skip prior_failure, part null.

package verifier

import (
	"fmt"
	"strconv"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// ExternalGate is the external gate of SPEC.md section 10.2, as DERIVE's
// end decides it.
type ExternalGate string

// The external gate's states.
const (
	// ExternalGateOpen: CHAIN and SOURCES read.
	ExternalGateOpen ExternalGate = "open"
	// ExternalGateOffline: closed, since the caller asked for an offline
	// run and no row failed; each row of CHAIN and SOURCES is skip offline.
	ExternalGateOffline ExternalGate = "offline"
	// ExternalGatePriorFailure: closed, since a row before CHAIN failed;
	// each row of CHAIN and SOURCES is skip prior_failure.
	ExternalGatePriorFailure ExternalGate = "prior_failure"
)

// keyEventCommitmentPart is the part of KEY_EVENT_COMMITMENT (SPEC.md
// section 10.3).
const keyEventCommitmentPart = "/signing_key_event"

// decideExternalGate decides the external gate of a run whose DERIVE has
// just ended (see the file's doc).
func decideExternalGate(run *Run) ExternalGate {
	switch {
	case run.Failed():
		return ExternalGatePriorFailure
	case run.Config.Online:
		return ExternalGateOpen
	}
	return ExternalGateOffline
}

// closedGate reports whether the run's external gate is closed, and the
// code each closed row takes. It panics for a run whose gate is
// undecided, since CHAIN and SOURCES follow DERIVE: a defect in the
// caller.
func closedGate(run *Run, step string) (specdata.Reason, bool) {
	switch run.ExternalGate {
	case ExternalGateOpen:
		return "", false
	case ExternalGateOffline:
		return specdata.ReasonOffline, true
	case ExternalGatePriorFailure:
		return specdata.ReasonPriorFailure, true
	}
	panic(fmt.Sprintf("verifier: %s before DERIVE decided the external gate", step))
}

// chainRead is one row of CHAIN before its outcome: its pointer, and the
// read that would confirm it, for a skip offline row's message.
type chainRead struct {
	at    Pointer
	would string
}

// Chain is step 10, CHAIN (see the file's doc).
func Chain(run *Run) {
	if run.Ended {
		return
	}
	code, closed := closedGate(run, "CHAIN")
	if !closed {
		chainOpen(run)
		return
	}
	var reads []chainRead
	for _, entry := range commitmentEntries(run.Proof) {
		reads = append(reads, chainRead{entry.at, commitmentRead(run.Config, entry)})
	}
	for _, genesis := range genesisBitcoin(run.Proof) {
		reads = append(reads, chainRead{genesis.at, fmt.Sprintf(
			"reading the hash of the best-chain block at height %s from a source of the pinned Bitcoin %s network",
			carried(genesis.node["height"]), run.Config.pinNetwork(PinCommitment, ChainBitcoin))})
	}
	SortBy(reads, func(r chainRead) Pointer { return r.at })
	for _, r := range reads {
		run.AddRow(closedRow("CHAIN", code, Part(r.at.Encode()), r.would))
	}
	if _, ok := run.Proof["signing_key_event"].(map[string]any); ok {
		run.AddRow(closedRow("CHAIN", code, Part(keyEventCommitmentPart),
			"reading each commitment entry under /signing_key_event, and its genesis.bitcoin when it carries one, from the chains"))
	}
}

// commitmentRead names the reads that would confirm a commitment entry
// (SPEC.md section 10.4): a top-level entry's also give its candidate.
func commitmentRead(cfg Config, entry layoutNode) string {
	network := cfg.pinNetwork(PinCommitment, entry.name)
	topLevel := entry.at[0] == Name("commitments")
	if entry.name == ChainStellar {
		read := fmt.Sprintf("reading the transaction %s from a source of the pinned Stellar %s network, which must be in ledger %s",
			carried(entry.node["transaction_hash"]), network, carried(entry.node["ledger"]))
		if topLevel {
			read += ", and then that ledger, for its close time"
		}
		return read
	}
	height := carried(entry.node["height"])
	read := fmt.Sprintf("reading the best-chain block at height %s from a source of the pinned Bitcoin %s network, with the tip at least %d blocks above it",
		height, network, specdata.BitcoinFinality)
	if topLevel {
		read += fmt.Sprintf(", and then the block %d above it, for its median time past", specdata.BitcoinFinality)
	}
	return read
}

// Sources is step 11, SOURCES (see the file's doc).
func Sources(run *Run) {
	if run.Ended {
		return
	}
	code, closed := closedGate(run, "SOURCES")
	if !closed {
		sourcesOpen(run)
		return
	}
	objects := entropyObjects(run.Proof)
	SortBy(objects, func(o layoutNode) Pointer { return o.at })
	for _, object := range objects {
		run.AddRow(closedRow("SOURCES", code, Part(object.at.Encode()), entropyRead(run.Config, object)))
	}
}

// entropyRead names the read that would confirm an entropy object (SPEC.md
// section 10.4, "Entropy objects").
func entropyRead(cfg Config, object layoutNode) string {
	payload, _ := object.node["payload"].(map[string]any)
	switch object.name {
	case ChainStellar:
		return fmt.Sprintf("reading ledger %s from a source of the pinned Stellar %s network",
			carried(payload["sequence"]), cfg.pinNetwork(PinEntropy, ChainStellar))
	case ChainBitcoin:
		return fmt.Sprintf("reading the block %s from a source of the pinned Bitcoin %s network",
			carried(payload["hash"]), cfg.pinNetwork(PinEntropy, ChainBitcoin))
	}
	pulse, _ := payload["pulse"].(map[string]any)
	return fmt.Sprintf("reading the pulse at chainIndex %s and pulseIndex %s from NIST's Randomness Beacon",
		carried(pulse["chainIndex"]), carried(pulse["pulseIndex"]))
}

// closedRow is a row of CHAIN or SOURCES under a closed gate: skip
// prior_failure, or skip offline with a message naming the read that would
// confirm it.
func closedRow(step string, code specdata.Reason, part *string, would string) Row {
	if code == specdata.ReasonPriorFailure {
		return NewRow(step, code, part, "Not read: a check before CHAIN failed, so this run reads no chain or source.", nil)
	}
	return NewRow(step, code, part, "Not read: this run is offline. Confirming it needs "+would+".", nil)
}

// carried writes a value the proof carries for a message, whole: a string
// as it is, and an integer in decimal.
func carried(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case strictjson.Number:
		if i, ok := v.Int64(); ok {
			return strconv.FormatInt(i, 10)
		}
	}
	return fmt.Sprint(value)
}
