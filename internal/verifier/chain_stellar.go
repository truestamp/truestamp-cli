// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds CHAIN's rule for a Stellar commitment entry, at
// /commitments/stellar or /signing_key_event/commitments/stellar, made
// against one source as if it were the only one (SPEC.md section 10.4,
// "Stellar commitment entries", "Answers" and "Rows"; section 11.1 for
// the candidate). With L the entry's ledger:
//
//  1. read the transaction whose hash is transaction_hash;
//  2. if it is found, each of these must hold, or the source's outcome is
//     fail chain_mismatch: successful is true; memo_type is hash, and the
//     memo is the 32 bytes of H(0x40, epoch.record) as this run computes
//     it, never the epoch_hash the entry carries; the transaction is in
//     ledger L. Then, for a top-level entry only, read ledger L: its
//     closed_at, a TIME, is the entry's candidate for the submitted-before
//     edge, basis stellar_ledger_close_read, at seconds' precision
//     (section 11.1). An entry under /signing_key_event gives no
//     candidate and reads no ledger, and passes on the transaction alone;
//  3. if it is not found, read the source's root, for latest and elder:
//     L > latest - STELLAR_INDEX_MARGIN is skip not_yet_indexed, its notes
//     naming latest and L; else L < elder is skip not_held_by_source, its
//     notes naming elder; else read ledger L: not found is skip
//     not_held_by_source, its notes naming L, and found is fail
//     chain_mismatch, since the source holds the ledger the proof names
//     and that ledger lacks the transaction.
//
// Every network follows these rules alike, testnet included, and the root
// is read only when the transaction is not found.
//
// An answer is judged on the fields the rule reads at that point, whatever
// else it lacks or gives in another form (SPEC.md section 10.4, "Answers"):
// the transaction on successful, memo_type, memo and ledger, so a memo the
// answer lacks beside a memo_type other than hash changes nothing, that
// memo_type already contradicting the proof; the root on
// latest and elder, each an INT; the candidate ledger on closed_at alone;
// and a ledger read after a transaction not found on being found. A field
// that contradicts the proof fails the outcome even when another is
// unusable, and an answer with an unusable field and no contradiction is
// skip source_unreachable, its notes naming the fields. An unreachable read
// is skip source_unreachable. A candidate read that answers not found is
// skip not_held_by_source, naming L; so is a root that answers not found,
// which no Horizon gives (section 13.2), since section 10.4 gives no
// outcome for that answer. Each source's outcome is combined with the
// network's other sources' under its policy (source_outcomes.go).

package verifier

import (
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// stellarMemoTypes are the memo types a Stellar transaction can have
// (SPEC.md section 13.2's table).
var stellarMemoTypes = []string{"none", "text", "id", "hash", "return"}

// stellarCloseBasis is the basis of a Stellar ledger's close time read
// (SPEC.md sections 11.1 and 11.5).
const stellarCloseBasis = "stellar_ledger_close_read"

// stellarEntry is what the rule reads of one Stellar commitment entry:
// its pointer; whether it is top-level, /commitments/stellar; its
// transaction_hash and its ledger L; and the memo the chain must hold, the
// lowercase hex of H(0x40, epoch.record) as this run computes it.
type stellarEntry struct {
	at       Pointer
	topLevel bool
	hash     string
	ledger   int64
	memo     string
}

// stellarEntryOf reads entry, a Stellar commitment entry of proof, or
// gives an error for one SHAPE should have refused.
func stellarEntryOf(proof map[string]any, entry layoutNode) (stellarEntry, error) {
	hash, isHash := hex64Field(entry.node, "transaction_hash")
	ledger, isLedger := intField(entry.node, "ledger")
	if !isHash || !isLedger {
		return stellarEntry{}, fmt.Errorf("the Stellar entry at %s has no transaction_hash and ledger in their classes", entry.at.Encode())
	}
	memo, err := recordHashAt(proof, registry.EpochHash, under(entry.at, "epoch", "record"))
	if err != nil {
		return stellarEntry{}, err
	}
	return stellarEntry{at: entry.at, topLevel: entry.at[0] == Name("commitments"), hash: hash, ledger: ledger, memo: memo}, nil
}

// stellarRule is the rule of e against one source of network.
func stellarRule(run *Run, network string, e stellarEntry) RowRule {
	return func(record SourceRecord) SourceOutcome {
		s := &stellarSource{chainSource{run: run, record: record, network: network}}
		return s.commitment(e)
	}
}

// stellarSource is one Stellar source the rule reads.
type stellarSource struct{ chainSource }

// commitment is the rule (see the file's doc).
func (s *stellarSource) commitment(e stellarEntry) SourceOutcome {
	tx := s.read(source.KindStellarTransaction, map[string]any{"hash": e.hash})
	switch tx.Outcome {
	case ReadFound:
		return s.transaction(e, tx.Value)
	case ReadNotFound:
		return s.missingTransaction(e)
	}
	return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
}

// transaction judges a transaction that was found.
func (s *stellarSource) transaction(e stellarEntry, value map[string]any) SourceOutcome {
	successful, isBool := boolField(value, "successful")
	memoType, isType := stringIn(value, "memo_type", stellarMemoTypes...)
	memo, isMemo := hex64Field(value, "memo")
	ledger, isLedger := intField(value, "ledger")
	contradicts, unusable := decideFields([]fieldCheck{
		{"successful", compared(successful, isBool, true)},
		{"memo_type", compared(memoType, isType, "hash")},
		{"memo", compared(memo, isMemo, e.memo)},
		{"ledger", compared(ledger, isLedger, e.ledger)},
	})
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

// candidate reads ledger L for a top-level entry's candidate.
func (s *stellarSource) candidate(e stellarEntry) SourceOutcome {
	ledger := s.read(source.KindStellarLedger, map[string]any{"sequence": intValue(e.ledger)})
	switch ledger.Outcome {
	case ReadFound:
		closedAt, err := ParseTime(ClassTIME, ledger.Value["closed_at"])
		if err != nil {
			return s.outcome(specdata.ReasonSourceUnreachable, nil, map[string]any{"unusable": []any{"closed_at"}})
		}
		return s.outcome("", &Candidate{Basis: stellarCloseBasis, Class: ClassTIME, Instant: closedAt, FetchedAt: ledger.FetchedAt}, nil)
	case ReadNotFound:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, map[string]any{"ledger": intValue(e.ledger)})
	}
	return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
}

// missingTransaction judges a transaction the source does not hold, from
// its root and, between the root's elder and latest, ledger L.
func (s *stellarSource) missingTransaction(e stellarEntry) SourceOutcome {
	root := s.read(source.KindStellarRoot, nil)
	switch root.Outcome {
	case ReadNotFound:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, nil)
	case ReadUnreachable:
		return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
	}
	latest, isLatest := intField(root.Value, "latest")
	elder, isElder := intField(root.Value, "elder")
	var unusable []any
	if !isLatest {
		unusable = append(unusable, "latest")
	}
	if !isElder {
		unusable = append(unusable, "elder")
	}
	if unusable != nil {
		return s.outcome(specdata.ReasonSourceUnreachable, nil, map[string]any{"unusable": unusable})
	}
	switch {
	case e.ledger > latest-specdata.StellarIndexMargin:
		return s.outcome(specdata.ReasonNotYetIndexed, nil, map[string]any{"latest": intValue(latest), "ledger": intValue(e.ledger)})
	case e.ledger < elder:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, map[string]any{"elder": intValue(elder)})
	}
	switch s.read(source.KindStellarLedger, map[string]any{"sequence": intValue(e.ledger)}).Outcome {
	case ReadFound:
		return s.outcome(specdata.ReasonChainMismatch, nil, map[string]any{"ledger": intValue(e.ledger)})
	case ReadNotFound:
		return s.outcome(specdata.ReasonNotHeldBySource, nil, map[string]any{"ledger": intValue(e.ledger)})
	}
	return s.outcome(specdata.ReasonSourceUnreachable, nil, nil)
}

// stellarMessage is the message of e's row, from its combined outcome and,
// for not_held_by_source and source_unreachable, the reads that settled
// each source's outcome ([stellarSettled]).
func stellarMessage(e stellarEntry, set SourceSet, combined CombinedOutcome) string {
	network := fmt.Sprintf("the pinned Stellar %s network", set.Network)
	switch combined.Code {
	case "":
		held := fmt.Sprintf("Read from %s: the transaction %s succeeded in ledger %d with this proof's epoch hash as its memo", network, e.hash, e.ledger)
		if combined.Candidate == nil {
			return held + "."
		}
		closed, _ := FormatTime(ClassTIME, combined.Candidate.Instant)
		return fmt.Sprintf("%s, and that ledger closed at %s.", held, closed)
	case specdata.ReasonChainMismatch:
		return fmt.Sprintf("Contradicted by %s: the transaction %s is not a successful transaction of ledger %d whose memo is this proof's epoch hash, or the source holds ledger %d and not the transaction.", network, e.hash, e.ledger, e.ledger)
	case specdata.ReasonNotYetIndexed:
		return fmt.Sprintf("Not confirmed: %s does not hold the transaction %s yet, and ledger %d is beyond, or within %d ledgers of, the latest ledger its source holds.", network, e.hash, e.ledger, specdata.StellarIndexMargin)
	case specdata.ReasonNotHeldBySource:
		byLedger := func(how stellarSettle) stellarSettle {
			if how == settledByLedgerAfterFound {
				return how
			}
			return settledByTransaction
		}
		how, alike := stellarSettled(combined, byLedger)
		switch {
		case !alike:
			return fmt.Sprintf("Not confirmed: %s's sources do not hold both the transaction %s and ledger %d. Absence is no contradiction.", network, e.hash, e.ledger)
		case how == settledByLedgerAfterFound:
			return fmt.Sprintf("Not confirmed: %s holds the transaction %s, which matches this proof, but not ledger %d. Absence is no contradiction.", asWhole(network, combined), e.hash, e.ledger)
		}
		return fmt.Sprintf("Not confirmed: the history %s keeps does not hold the transaction %s or ledger %d. Absence is no contradiction.", asWhole(network, combined), e.hash, e.ledger)
	case specdata.ReasonSourcesDisagree:
		return fmt.Sprintf("Not confirmed: the sources of %s disagree on the transaction %s. A disagreement is no failure.", network, e.hash)
	case specdata.ReasonSourcesIncomplete:
		return fmt.Sprintf("Not confirmed: fewer sources of %s than its policy %s asks for reached an outcome on the transaction %s.", network, set.Policy, e.hash)
	}
	how, alike := stellarSettled(combined, func(how stellarSettle) stellarSettle { return how })
	switch {
	case !alike:
		return fmt.Sprintf("Not confirmed: no source of %s gave usable answers about both the transaction %s and ledger %d.", network, e.hash, e.ledger)
	case how == settledByNothing:
		return fmt.Sprintf("Not confirmed: no source of %s had its identity confirmed, so none was read.", network)
	case how == settledByLedgerAfterFound:
		return fmt.Sprintf("Not confirmed: the transaction %s matches this proof, but no source of %s gave a usable answer about ledger %d.", e.hash, network, e.ledger)
	case how == settledByRoot:
		return fmt.Sprintf("Not confirmed: no source of %s holds the transaction %s, and none gave a usable answer about the ledgers it holds.", network, e.hash)
	case how == settledByLedgerAfterMissing:
		return fmt.Sprintf("Not confirmed: no source of %s holds the transaction %s, and none gave a usable answer about ledger %d.", network, e.hash, e.ledger)
	}
	return fmt.Sprintf("Not confirmed: no source of %s gave a usable answer about the transaction %s.", network, e.hash)
}

// stellarSettle is the read that settled a source's outcome on a Stellar
// row: no read at all, for a source whose identity was unreachable and
// that was asked nothing; its transaction read; ledger L's read after a
// transaction the source held that matched the proof, which the rule
// reads only after a found transaction passes every check; the root's
// read after a transaction the source does not hold; or ledger L's read
// after the root's.
type stellarSettle int

const (
	settledByNothing stellarSettle = iota
	settledByTransaction
	settledByLedgerAfterFound
	settledByRoot
	settledByLedgerAfterMissing
)

// stellarSettleOf is the read that settled o.
func stellarSettleOf(o SourceOutcome) stellarSettle {
	switch n := len(o.Reads); {
	case n == 0:
		return settledByNothing
	case n == 1:
		return settledByTransaction
	case o.Reads[0].Outcome == ReadFound:
		return settledByLedgerAfterFound
	case n == 2:
		return settledByRoot
	}
	return settledByLedgerAfterMissing
}

// stellarSettled is how every source combined asked was settled, as group
// counts two ways alike, and whether they were all settled alike.
func stellarSettled(combined CombinedOutcome, group func(stellarSettle) stellarSettle) (stellarSettle, bool) {
	how := group(stellarSettleOf(combined.Asked[0].Outcome))
	for _, asked := range combined.Asked[1:] {
		if group(stellarSettleOf(asked.Outcome)) != how {
			return how, false
		}
	}
	return how, true
}

// intValue is n as a read's parameters and a row's values hold it.
func intValue(n int64) strictjson.Number { return strictjson.Number(float64(n)) }
