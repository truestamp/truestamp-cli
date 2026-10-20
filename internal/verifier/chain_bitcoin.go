// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds CHAIN's rule for a Bitcoin commitment entry, at
// /commitments/bitcoin or /signing_key_event/commitments/bitcoin, and for
// genesis.bitcoin, at /block, /witnesses/block or /signing_key_event/block
// (B/record/metadata/genesis/bitcoin), each made against one source as if
// it were the only one (SPEC.md section 10.4, "Bitcoin commitment
// entries", "genesis.bitcoin", "Median time past", "Answers" and "Rows";
// section 11.1 for the candidate). With h the entry's or the object's
// height:
//
//  1. read the source's tip, t;
//  2. h > t is skip not_yet_indexed, and t < h + BITCOIN_FINALITY is skip
//     chain_not_final, each with notes naming h and t;
//  3. read the best-chain block at height h. For a commitment entry, its
//     hash must equal the header_hash this run computes from the
//     txoutproof's header, rev(SHA-256d(header)), and its
//     transaction_count the txoutproof's total_txs, or the source's
//     outcome is fail chain_mismatch, its notes naming the fields that
//     differ. For genesis.bitcoin, only its hash is compared, with
//     genesis.bitcoin's header_hash, and nothing else is read: the row
//     gives no candidate;
//  4. for a top-level entry only, then read the best-chain block at height
//     h + BITCOIN_FINALITY: its median time past, the source's report of
//     it, an INT of Unix seconds that a TIME can write, is the entry's
//     candidate for the submitted-before edge, basis
//     bitcoin_median_time_read, at seconds' precision (section 11.1). An
//     entry under /signing_key_event gives no candidate and reads no block
//     past h.
//
// The proof's header_hash member is never what item 3 compares: DERIVE
// has held it to the recomputed hash, and the rule compares the
// recomputation (section 10.6, rule 4). No block is read by a hash the
// proof carries, and every read goes to a source of the run's Bitcoin
// commitment pin, for genesis.bitcoin too. Every network follows these
// rules alike.
//
// An answer is judged on the fields the rule reads at that point, whatever
// else it lacks or gives in another form (section 10.4, "Answers"): the
// tip on height; the block at h on hash, and on transaction_count for a
// commitment entry; and the block at h + BITCOIN_FINALITY on
// median_time_past alone, its hash and transaction count unread. A field
// that contradicts the proof fails the outcome even when another is
// unusable, and an answer with an unusable field and no contradiction is
// skip source_unreachable, its notes naming the fields. The block at a
// height is one read of two calls, its hash and then the block with that
// hash (source/bitcoin.go), and when the second call answers not found or
// gives no usable answer the read is found with the hash alone (section
// 10.4, "Answers", a read made of several calls): genesis.bitcoin's row is
// judged on that hash, which fails it when it differs, and a commitment
// entry's row whose hash matches is skip source_unreachable, since the
// transaction count is not given. An unreachable read is skip
// source_unreachable. A read that answers not found where the
// rule gives no outcome for that answer, the block at h or at h +
// BITCOIN_FINALITY, which the tip covers, is skip not_held_by_source, its
// notes naming the height read; so is a tip that answers not found, which
// no source gives (section 13.2). Each source's outcome is combined with
// the network's other sources' under its policy (source_outcomes.go).

package verifier

import (
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier/bitcoin"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// bitcoinMedianBasis is the basis of a Bitcoin median time past read
// (SPEC.md sections 11.1 and 11.5).
const bitcoinMedianBasis = "bitcoin_median_time_read"

// bitcoinEntry is what the rule reads of one Bitcoin commitment entry or
// genesis.bitcoin: its pointer; whether it is a genesis.bitcoin, and
// whether it is the top-level entry, /commitments/bitcoin; its height h;
// the hash the best-chain block at h must have, the header_hash this run
// computes from an entry's txoutproof, or a genesis.bitcoin's header_hash,
// each in display order; and an entry's txoutproof's total_txs.
type bitcoinEntry struct {
	at       Pointer
	genesis  bool
	topLevel bool
	height   int64
	hash     string
	totalTxs int64
}

// bitcoinEntryOf reads entry, a Bitcoin commitment entry of proof, or
// gives an error for one SHAPE should have refused, or whose txoutproof
// DERIVE has held well formed and is not.
func bitcoinEntryOf(proof map[string]any, entry layoutNode) (bitcoinEntry, error) {
	height, isHeight := intField(entry.node, "height")
	if !isHeight {
		return bitcoinEntry{}, fmt.Errorf("the Bitcoin entry at %s has no height in its class", entry.at.Encode())
	}
	raw, err := hexBytesAt(proof, under(entry.at, "txoutproof"))
	if err != nil {
		return bitcoinEntry{}, err
	}
	block, err := bitcoin.ParseMerkleBlock(raw)
	if err != nil {
		return bitcoinEntry{}, fmt.Errorf("the txoutproof at %s is not well formed: it breaks the rule %w", entry.at.Encode(), err)
	}
	return bitcoinEntry{
		at:       entry.at,
		topLevel: entry.at[0] == Name("commitments"),
		height:   height,
		hash:     bitcoin.Display(bitcoin.HeaderHash(block.Header)),
		totalTxs: int64(block.TotalTxs),
	}, nil
}

// bitcoinGenesisOf reads node, a genesis.bitcoin of P, or gives an error
// for one SHAPE should have refused.
func bitcoinGenesisOf(node layoutNode) (bitcoinEntry, error) {
	hash, isHash := hex64Field(node.node, "header_hash")
	height, isHeight := intField(node.node, "height")
	if !isHash || !isHeight {
		return bitcoinEntry{}, fmt.Errorf("the genesis.bitcoin at %s has no header_hash and height in their classes", node.at.Encode())
	}
	return bitcoinEntry{at: node.at, genesis: true, height: height, hash: hash}, nil
}

// bitcoinChainEntry is e's row of CHAIN, before it is read.
func bitcoinChainEntry(e bitcoinEntry) chainEntry {
	return chainEntry{at: e.at, chain: ChainBitcoin,
		rule:    func(run *Run, network string) RowRule { return bitcoinRule(run, network, e) },
		message: func(set SourceSet, c CombinedOutcome) string { return bitcoinMessage(e, set, c) }}
}

// bitcoinRule is the rule of e against one source of network.
func bitcoinRule(run *Run, network string, e bitcoinEntry) RowRule {
	return func(record SourceRecord) SourceOutcome {
		s := &bitcoinSource{chainSource{run: run, record: record, network: network}}
		return s.commitment(e)
	}
}

// bitcoinSource is one Bitcoin source the rule reads.
type bitcoinSource struct{ chainSource }

// commitment is the rule (see the file's doc).
func (s *bitcoinSource) commitment(e bitcoinEntry) SourceOutcome {
	if outcome, final := s.final(e.height); !final {
		return outcome
	}
	block := s.read(source.KindBitcoinBlockAtHeight, map[string]any{"height": intValue(e.height)})
	switch block.Outcome {
	case ReadNotFound:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, map[string]any{"height": intValue(e.height)})
	case ReadUnreachable:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
	}
	hash, isHash := hex64Field(block.Value, "hash")
	checks := []fieldCheck{{"hash", compared(hash, isHash, e.hash)}}
	if !e.genesis {
		count, isCount := intField(block.Value, "transaction_count")
		checks = append(checks, fieldCheck{"transaction_count", compared(count, isCount, e.totalTxs)})
	}
	contradicts, unusable := decideFields(checks)
	switch {
	case contradicts != nil:
		return s.outcome(specdata.ReasonChainMismatch, nil, map[string]any{"differs": contradicts})
	case unusable != nil:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, map[string]any{"unusable": unusable})
	case !e.topLevel:
		return s.outcome("", nil, nil)
	}
	return s.candidate(e)
}

// final reads the source's tip and gives the outcome of items 1 and 2 when
// they settle it, or false when the tip is at least h + BITCOIN_FINALITY.
func (s *bitcoinSource) final(h int64) (SourceOutcome, bool) {
	tip := s.read(source.KindBitcoinTip, nil)
	switch tip.Outcome {
	case ReadNotFound:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, nil), false
	case ReadUnreachable:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, nil), false
	}
	t, isTip := intField(tip.Value, "height")
	switch {
	case !isTip:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, map[string]any{"unusable": []any{"height"}}), false
	case h > t:
		return s.outcome(specdata.ReasonNotYetIndexed, nil, map[string]any{"height": intValue(h), "tip": intValue(t)}), false
	case t < h+specdata.BitcoinFinality:
		return s.outcome(specdata.ReasonChainNotFinal, nil, map[string]any{"height": intValue(h), "tip": intValue(t)}), false
	}
	return SourceOutcome{}, true
}

// candidate reads the best-chain block at h + BITCOIN_FINALITY for a
// top-level entry's candidate.
func (s *bitcoinSource) candidate(e bitcoinEntry) SourceOutcome {
	at := e.height + specdata.BitcoinFinality
	block := s.read(source.KindBitcoinBlockAtHeight, map[string]any{"height": intValue(at)})
	switch block.Outcome {
	case ReadNotFound:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, map[string]any{"height": intValue(at)})
	case ReadUnreachable:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
	}
	median, isMedian := intField(block.Value, "median_time_past")
	if _, isTime := FormatTime(ClassTIME, median); !isMedian || !isTime {
		return s.outcome(specdata.ReasonSourceUnreachable, nil, map[string]any{"unusable": []any{"median_time_past"}})
	}
	return s.outcome("", &Candidate{Basis: bitcoinMedianBasis, Class: ClassTIME, Instant: median, FetchedAt: block.FetchedAt}, nil)
}

// bitcoinMessage is the message of e's row, from its combined outcome and,
// for not_held_by_source and source_unreachable, the read that settled
// each source's outcome ([bitcoinSettled]).
func bitcoinMessage(e bitcoinEntry, set SourceSet, combined CombinedOutcome) string {
	network := fmt.Sprintf("the pinned Bitcoin %s network", set.Network)
	final := e.height + specdata.BitcoinFinality
	switch combined.Code {
	case "":
		if e.genesis {
			return fmt.Sprintf("Read from %s: the best-chain block at height %d is the block %s this genesis names.", network, e.height, e.hash)
		}
		held := fmt.Sprintf("Read from %s: the best-chain block at height %d is the block %s, with %d transactions, that this proof's txoutproof gives", network, e.height, e.hash, e.totalTxs)
		if combined.Candidate == nil {
			return held + "."
		}
		median, _ := FormatTime(ClassTIME, combined.Candidate.Instant)
		return fmt.Sprintf("%s, and the median time past of the best-chain block at height %d is %s.", held, final, median)
	case specdata.ReasonChainMismatch:
		if e.genesis {
			return fmt.Sprintf("Contradicted by %s: the best-chain block at height %d is not the block %s this genesis names.", network, e.height, e.hash)
		}
		return fmt.Sprintf("Contradicted by %s: the best-chain block at height %d is not the block %s, with %d transactions, that this proof's txoutproof gives.", network, e.height, e.hash, e.totalTxs)
	case specdata.ReasonNotYetIndexed:
		return fmt.Sprintf("Not confirmed: the tip of %s is below height %d, which it does not hold yet.", asWhole(network, combined), e.height)
	case specdata.ReasonChainNotFinal:
		return fmt.Sprintf("Not confirmed: the tip of %s is below height %d, %d blocks above height %d, so the block at height %d is not final yet.",
			asWhole(network, combined), final, specdata.BitcoinFinality, e.height, e.height)
	case specdata.ReasonSourcesDisagree:
		return fmt.Sprintf("Not confirmed: the sources of %s disagree on the best-chain block at height %d. A disagreement is no failure.", network, e.height)
	case specdata.ReasonSourcesIncomplete:
		return fmt.Sprintf("Not confirmed: fewer sources of %s than its policy %s asks for reached an outcome on the best-chain block at height %d.", network, set.Policy, e.height)
	}
	how, alike := bitcoinSettled(combined)
	if combined.Code == specdata.ReasonNotHeldBySource {
		switch {
		case !alike:
			return fmt.Sprintf("Not confirmed: the sources of %s do not all hold what this row reads. Absence is no contradiction.", network)
		case how.kind == source.KindBitcoinTip:
			return fmt.Sprintf("Not confirmed: %s answered that it holds no tip. Absence is no contradiction.", asWhole(network, combined))
		case how.height == final:
			return fmt.Sprintf("Not confirmed: %s holds the best-chain block at height %d, which matches this proof, but no best-chain block at height %d. Absence is no contradiction.", asWhole(network, combined), e.height, final)
		}
		return fmt.Sprintf("Not confirmed: %s holds no best-chain block at height %d, though its tip is past it. Absence is no contradiction.", asWhole(network, combined), e.height)
	}
	switch {
	case !alike:
		return fmt.Sprintf("Not confirmed: no source of %s gave a usable answer to every read this row makes.", network)
	case how.kind == "":
		return fmt.Sprintf("Not confirmed: no source of %s had its identity confirmed, so none was read.", network)
	case how.kind == source.KindBitcoinTip:
		return fmt.Sprintf("Not confirmed: no source of %s gave a usable answer about its tip.", network)
	case how.height == final:
		return fmt.Sprintf("Not confirmed: the best-chain block at height %d matches this proof, but no source of %s gave a usable answer about the best-chain block at height %d.", e.height, network, final)
	}
	return fmt.Sprintf("Not confirmed: no source of %s gave a usable answer about the best-chain block at height %d.", network, e.height)
}

// bitcoinSettle is the read that settled a source's outcome on a Bitcoin
// row, its last: its kind, "" for a source whose identity was unreachable
// and that was asked nothing, and the height a block read named.
type bitcoinSettle struct {
	kind   string
	height int64
}

// bitcoinSettled is the read that settled every source combined asked,
// and whether they were all settled by the same read.
func bitcoinSettled(combined CombinedOutcome) (bitcoinSettle, bool) {
	of := func(o SourceOutcome) bitcoinSettle {
		if len(o.Reads) == 0 {
			return bitcoinSettle{}
		}
		last := o.Reads[len(o.Reads)-1]
		height, _ := intField(last.Read, "height")
		kind, _ := last.Read["kind"].(string)
		return bitcoinSettle{kind: kind, height: height}
	}
	how := of(combined.Asked[0].Outcome)
	for _, asked := range combined.Asked[1:] {
		if of(asked.Outcome) != how {
			return how, false
		}
	}
	return how, true
}
