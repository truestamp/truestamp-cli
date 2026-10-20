// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// CHAIN's rule for a Stellar commitment entry against one source (SPEC.md
// section 10.4, "Stellar commitment entries", "Answers" and "Rows"), over
// a stand-in chain reader that answers each read from a table and records
// each read it was asked.

// The entry the rule is made for: its transaction, its ledger L and the
// memo the chain must hold.
const (
	ruleHash   = "860d4085c0af0b5f8cadff2bb0647cf75bbd16901817684dc4e180c200c70681"
	ruleMemo   = "488e6583a192071810724dd392b551c4ca119ca6688deb9034683eb42eba1364"
	ruleLedger = 1000255
	ruleClosed = "2026-09-26T01:52:15Z"
)

// chainStandIn is a chain reader that answers each read from answers, by
// the source's URL and the read's JCS, and records each read it was asked
// in order, as "<kind>" for the source's first name; a read it has no
// answer for is unreachable.
type chainStandIn struct {
	answers map[string]SourceRead
	asked   []string
}

// key is a read's place in answers.
func standInKey(url string, read map[string]any) string {
	encoded, err := jcs.Encode(read)
	if err != nil {
		panic(err)
	}
	return url + " " + string(encoded)
}

func (s *chainStandIn) reader(src Source, read map[string]any) SourceRead {
	host := strings.TrimSuffix(strings.TrimPrefix(src.URL, "https://"), ".example")
	s.asked = append(s.asked, host+" "+read["kind"].(string))
	if answer, ok := s.answers[standInKey(src.URL, read)]; ok {
		return answer
	}
	return SourceRead{Outcome: ReadUnreachable, Reason: "no answer in the stand-in"}
}

// The reads of the rule's entry on network.
func txRead(network string) map[string]any {
	return map[string]any{"kind": source.KindStellarTransaction, "network": network, "hash": ruleHash}
}

func ledgerRead(network string) map[string]any {
	return map[string]any{"kind": source.KindStellarLedger, "network": network, "sequence": strictjson.Number(ruleLedger)}
}

func rootRead(network string) map[string]any {
	return map[string]any{"kind": source.KindStellarRoot, "network": network}
}

// found and notFound are answers.
func found(value map[string]any) SourceRead { return SourceRead{Outcome: ReadFound, Value: value} }

var notFound = SourceRead{Outcome: ReadNotFound}

var unreachableRead = SourceRead{Outcome: ReadUnreachable, Reason: "HTTP 429"}

// soundTx is a transaction the rule passes, with members replaced by
// change (a nil value removes one).
func soundTx(change map[string]any) SourceRead {
	value := map[string]any{"successful": true, "memo_type": "hash", "memo": ruleMemo, "ledger": strictjson.Number(ruleLedger)}
	for name, v := range change {
		if v == nil {
			delete(value, name)
		} else {
			value[name] = v
		}
	}
	return found(value)
}

// root is a root answer with latest and elder.
func root(latest, elder int64) SourceRead {
	return found(map[string]any{"latest": strictjson.Number(latest), "elder": strictjson.Number(elder)})
}

// ruleOutcome makes the rule of the entry, top-level or under
// /signing_key_event, against one confirmed source of network,
// https://a.example, whose answers are reads, and gives its outcome and
// the reads the stand-in was asked.
func ruleOutcome(network string, topLevel bool, reads map[string]SourceRead) (SourceOutcome, []string) {
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	for name, answer := range reads {
		var read map[string]any
		switch name {
		case "tx":
			read = txRead(network)
		case "ledger":
			read = ledgerRead(network)
		case "root":
			read = rootRead(network)
		}
		stand.answers[standInKey("https://a.example", read)] = answer
	}
	at := Pointer{Name("commitments"), Name("stellar")}
	if !topLevel {
		at = Pointer{Name("signing_key_event"), Name("commitments"), Name("stellar")}
	}
	e := stellarEntry{at: at, topLevel: topLevel, hash: ruleHash, ledger: ruleLedger, memo: ruleMemo}
	run := &Run{Config: Config{ReadChain: stand.reader}}
	record := standInSet(SourcePolicy{Kind: SourcePolicyFirst}, "a").Sources[0]
	return stellarRule(run, network, e)(record), stand.asked
}

// outcomeText writes an outcome as "<status> <code> <notes JCS> <reads>",
// with "candidate <instant>" when it has one.
func outcomeText(o SourceOutcome, asked []string) string {
	notes, _ := jcs.Encode(o.Notes)
	if o.Notes == nil {
		notes = []byte("-")
	}
	text := fmt.Sprintf("%s %s %s %v", o.Status, o.Code, notes, asked)
	if o.Candidate != nil {
		text += fmt.Sprintf(" candidate %s %d", o.Candidate.Basis, o.Candidate.Instant)
	}
	return text
}

// closedAt is ruleClosed in Unix seconds.
var closedAt = func() int64 {
	s, err := ParseTime(ClassTIME, ruleClosed)
	if err != nil {
		panic(err)
	}
	return s
}()

// Every outcome of the rule, on public and on testnet alike, from the
// answers of one source (SPEC.md section 10.4): found and successful,
// with the candidate for a top-level entry from ledger L's closed_at and
// no ledger read for an entry under /signing_key_event; each check of the
// transaction that fails, chain_mismatch naming the fields that differ, a
// contradiction failing even beside a field the answer lacks; an
// incomplete answer judged on the fields it holds: a field the rule
// uses absent or in another form is source_unreachable naming it, a memo
// the answer lacks beside a memo_type other than hash leaving the
// contradiction alone, and a field it does not use ignored; not found, then the root's latest and elder
// against L at, within and beyond the margin and at, below and above
// elder, with ledger L read only between them; unreachable at each read.
// Mutants killed: successful not read (successful false passes); memo_type
// not compared (a return memo passes); hex64Field taking uppercase (an
// uppercase memo is compared, not unusable); the ledger check against
// L - 1; the transaction's unusable fields judged before its
// contradictions (successful false with no ledger skips); the candidate
// read for every entry (the signing-key entry reads its ledger); the margin compared with >= (L at latest - 60
// is not_yet_indexed); the margin taken as 59 (L at latest - 60 + 1 is
// read, L > latest - 59 false); elder compared with <= (L at elder is
// not_held_by_source); elder checked before the margin (within the margin
// and below elder gives not_held_by_source); the ledger after a missing
// transaction judged on its fields (a found ledger with no field is
// unreachable); a not-found ledger after a missing transaction read as a
// contradiction; a root that is not found read as unreachable; the root's
// elder not required (a root without elder reads the ledger); closed_at
// parsed as a NISTTIME (every candidate read is unusable); intField taking
// a negative number (the root's latest of -1 is not_yet_indexed); the rule
// reading the root before the transaction; a testnet-only branch, a
// missing transaction on testnet not_held_by_source with no root read
// (each such testnet row differs).
func TestStellarRule_EveryOutcome(t *testing.T) {
	ledgerFound := found(map[string]any{"closed_at": ruleClosed, "hash": strings.Repeat("0", 64)})
	L := int64(ruleLedger)
	for _, network := range []string{"public", "testnet"} {
		for _, tc := range []struct {
			name     string
			topLevel bool
			reads    map[string]SourceRead
			want     string
		}{
			{"found, top-level", true, map[string]SourceRead{"tx": soundTx(nil), "ledger": ledgerFound},
				fmt.Sprintf("pass  - [a stellar_transaction a stellar_ledger] candidate stellar_ledger_close_read %d", closedAt)},
			{"found, under the key event", false, map[string]SourceRead{"tx": soundTx(nil), "ledger": ledgerFound},
				"pass  - [a stellar_transaction]"},
			{"the candidate ledger without hash", true, map[string]SourceRead{"tx": soundTx(nil), "ledger": found(map[string]any{"closed_at": ruleClosed})},
				fmt.Sprintf("pass  - [a stellar_transaction a stellar_ledger] candidate stellar_ledger_close_read %d", closedAt)},
			{"a field the rule does not use", true, map[string]SourceRead{"tx": soundTx(map[string]any{"fee_charged": "100"}), "ledger": ledgerFound},
				fmt.Sprintf("pass  - [a stellar_transaction a stellar_ledger] candidate stellar_ledger_close_read %d", closedAt)},
			{"not successful", true, map[string]SourceRead{"tx": soundTx(map[string]any{"successful": false})},
				`fail chain_mismatch {"differs":["successful"]} [a stellar_transaction]`},
			{"a text memo", true, map[string]SourceRead{"tx": soundTx(map[string]any{"memo_type": "text", "memo": nil})},
				`fail chain_mismatch {"differs":["memo_type"]} [a stellar_transaction]`},
			{"a return memo", true, map[string]SourceRead{"tx": soundTx(map[string]any{"memo_type": "return"})},
				`fail chain_mismatch {"differs":["memo_type"]} [a stellar_transaction]`},
			{"another memo", false, map[string]SourceRead{"tx": soundTx(map[string]any{"memo": strings.Repeat("ab", 32)})},
				`fail chain_mismatch {"differs":["memo"]} [a stellar_transaction]`},
			{"another ledger", true, map[string]SourceRead{"tx": soundTx(map[string]any{"ledger": strictjson.Number(L - 1)})},
				`fail chain_mismatch {"differs":["ledger"]} [a stellar_transaction]`},
			{"not successful, no ledger", true, map[string]SourceRead{"tx": soundTx(map[string]any{"successful": false, "ledger": nil})},
				`fail chain_mismatch {"differs":["successful"]} [a stellar_transaction]`},
			{"no memo", true, map[string]SourceRead{"tx": soundTx(map[string]any{"memo": nil})},
				`skip source_unreachable {"unusable":["memo"]} [a stellar_transaction]`},
			{"an uppercase memo", true, map[string]SourceRead{"tx": soundTx(map[string]any{"memo": strings.ToUpper(ruleMemo)})},
				`skip source_unreachable {"unusable":["memo"]} [a stellar_transaction]`},
			{"no ledger", true, map[string]SourceRead{"tx": soundTx(map[string]any{"ledger": nil})},
				`skip source_unreachable {"unusable":["ledger"]} [a stellar_transaction]`},
			{"a ledger string", true, map[string]SourceRead{"tx": soundTx(map[string]any{"ledger": "1000255"})},
				`skip source_unreachable {"unusable":["ledger"]} [a stellar_transaction]`},
			{"successful a string", true, map[string]SourceRead{"tx": soundTx(map[string]any{"successful": "true"})},
				`skip source_unreachable {"unusable":["successful"]} [a stellar_transaction]`},
			{"an unknown memo_type, no memo", true, map[string]SourceRead{"tx": soundTx(map[string]any{"memo_type": "other", "memo": nil})},
				`skip source_unreachable {"unusable":["memo_type","memo"]} [a stellar_transaction]`},
			{"an empty answer", true, map[string]SourceRead{"tx": found(map[string]any{})},
				`skip source_unreachable {"unusable":["successful","memo_type","memo","ledger"]} [a stellar_transaction]`},
			{"the candidate ledger without closed_at", true, map[string]SourceRead{"tx": soundTx(nil), "ledger": found(map[string]any{"hash": strings.Repeat("0", 64)})},
				`skip source_unreachable {"unusable":["closed_at"]} [a stellar_transaction a stellar_ledger]`},
			{"the candidate ledger's closed_at in another form", true, map[string]SourceRead{"tx": soundTx(nil), "ledger": found(map[string]any{"closed_at": "2026-09-26 01:52:15"})},
				`skip source_unreachable {"unusable":["closed_at"]} [a stellar_transaction a stellar_ledger]`},
			{"the candidate ledger not found", true, map[string]SourceRead{"tx": soundTx(nil), "ledger": notFound},
				`skip not_held_by_source {"ledger":1000255} [a stellar_transaction a stellar_ledger]`},
			{"the candidate ledger unreachable", true, map[string]SourceRead{"tx": soundTx(nil), "ledger": unreachableRead},
				`skip source_unreachable - [a stellar_transaction a stellar_ledger]`},
			{"the transaction unreachable", true, map[string]SourceRead{"tx": unreachableRead},
				`skip source_unreachable - [a stellar_transaction]`},
			{"not found, within the margin", true, map[string]SourceRead{"tx": notFound, "root": root(L+59, L-100)},
				`skip not_yet_indexed {"latest":1000314,"ledger":1000255} [a stellar_transaction a stellar_root]`},
			{"not found, beyond latest", false, map[string]SourceRead{"tx": notFound, "root": root(L-5, L-100)},
				`skip not_yet_indexed {"latest":1000250,"ledger":1000255} [a stellar_transaction a stellar_root]`},
			{"not found, at the margin, ledger not held", true, map[string]SourceRead{"tx": notFound, "root": root(L+60, L-100), "ledger": notFound},
				`skip not_held_by_source {"ledger":1000255} [a stellar_transaction a stellar_root a stellar_ledger]`},
			{"not found, below elder", true, map[string]SourceRead{"tx": notFound, "root": root(L+1000, L+1)},
				`skip not_held_by_source {"elder":1000256} [a stellar_transaction a stellar_root]`},
			{"not found, within the margin and below elder", true, map[string]SourceRead{"tx": notFound, "root": root(L+10, L+5)},
				`skip not_yet_indexed {"latest":1000265,"ledger":1000255} [a stellar_transaction a stellar_root]`},
			{"not found, at elder, ledger held", true, map[string]SourceRead{"tx": notFound, "root": root(L+1000, L), "ledger": ledgerFound},
				`fail chain_mismatch {"ledger":1000255} [a stellar_transaction a stellar_root a stellar_ledger]`},
			{"not found, the ledger found with no field", false, map[string]SourceRead{"tx": notFound, "root": root(L+1000, L-1), "ledger": found(map[string]any{})},
				`fail chain_mismatch {"ledger":1000255} [a stellar_transaction a stellar_root a stellar_ledger]`},
			{"not found, the ledger unreachable", true, map[string]SourceRead{"tx": notFound, "root": root(L+1000, L-1), "ledger": unreachableRead},
				`skip source_unreachable - [a stellar_transaction a stellar_root a stellar_ledger]`},
			{"not found, the root unreachable", true, map[string]SourceRead{"tx": notFound, "root": unreachableRead},
				`skip source_unreachable - [a stellar_transaction a stellar_root]`},
			{"not found, the root not found", true, map[string]SourceRead{"tx": notFound, "root": notFound},
				`skip not_held_by_source - [a stellar_transaction a stellar_root]`},
			{"not found, the root without elder", true, map[string]SourceRead{"tx": notFound, "root": found(map[string]any{"latest": strictjson.Number(L + 1000)}), "ledger": ledgerFound},
				`skip source_unreachable {"unusable":["elder"]} [a stellar_transaction a stellar_root]`},
			{"not found, the root's latest negative", true, map[string]SourceRead{"tx": notFound, "root": found(map[string]any{"latest": strictjson.Number(-1), "elder": strictjson.Number(1)})},
				`skip source_unreachable {"unusable":["latest"]} [a stellar_transaction a stellar_root]`},
		} {
			got, asked := ruleOutcome(network, tc.topLevel, tc.reads)
			if text := outcomeText(got, asked); text != tc.want {
				t.Errorf("%s, %s:\n got %s\nwant %s", network, tc.name, text, tc.want)
			}
			if len(got.Reads) != len(asked) {
				t.Errorf("%s, %s: the outcome holds %d reads, the source was asked %d", network, tc.name, len(got.Reads), len(asked))
			}
			for _, r := range got.Reads {
				if r.Read["network"] != network {
					t.Errorf("%s, %s: a read of %v", network, tc.name, r.Read)
				}
			}
		}
	}
}

// Each read the rule makes is named as a replay record names it, {kind,
// network, hash} for the transaction, {kind, network, sequence} for a
// ledger and {kind, network} for the root, with the answer and the time it
// came back, and the candidate's fetched_at is its ledger read's. Mutants
// killed: the ledger's sequence written as a string (the read differs);
// the candidate's FetchedAt the transaction read's.
func TestStellarRule_TheReadsItRecords(t *testing.T) {
	t0 := time.Date(2026, 10, 20, 9, 0, 1, 0, time.UTC)
	useClock(t, t0, t0.Add(time.Second))
	ledger := found(map[string]any{"closed_at": ruleClosed})
	got, _ := ruleOutcome("testnet", true, map[string]SourceRead{"tx": soundTx(nil), "ledger": ledger})
	want := []SourceRead{
		{Read: txRead("testnet"), Outcome: ReadFound, Value: soundTx(nil).Value, FetchedAt: t0},
		{Read: ledgerRead("testnet"), Outcome: ReadFound, Value: ledger.Value, FetchedAt: t0.Add(time.Second)},
	}
	if !reflect.DeepEqual(got.Reads, want) || got.Candidate == nil || !got.Candidate.FetchedAt.Equal(t0.Add(time.Second)) {
		t.Fatalf("reads %+v, candidate %+v", got.Reads, got.Candidate)
	}
	missing, _ := ruleOutcome("public", true, map[string]SourceRead{"tx": notFound, "root": root(ruleLedger+100, ruleLedger-100), "ledger": notFound})
	var reads []map[string]any
	for _, r := range missing.Reads {
		reads = append(reads, r.Read)
	}
	if !reflect.DeepEqual(reads, []map[string]any{txRead("public"), rootRead("public"), ledgerRead("public")}) {
		t.Fatalf("reads %v", reads)
	}
}

// A run with no chain reader, and a reader whose answer is none of SPEC.md
// section 10.4's three outcomes, give no usable answer, so the source's
// outcome is source_unreachable; a found answer with no value holds no
// field, and a not-found one keeps neither value nor reason. Mutants
// killed: readChain without its nil-reader guard (a panic); an unknown
// outcome passed through (the read keeps the outcome "maybe"); a found
// answer's nil value kept nil; a not-found answer keeping the reader's
// value and reason.
func TestStellarRule_AReaderWithNoAnswer(t *testing.T) {
	e := stellarEntry{at: Pointer{Name("commitments"), Name("stellar")}, topLevel: true, hash: ruleHash, ledger: ruleLedger, memo: ruleMemo}
	record := standInSet(SourcePolicy{Kind: SourcePolicyFirst}, "a").Sources[0]
	for name, reader := range map[string]ChainReader{
		"no reader":          nil,
		"an unknown outcome": func(Source, map[string]any) SourceRead { return SourceRead{Outcome: "maybe", Value: map[string]any{}} },
		"a found nil value":  func(Source, map[string]any) SourceRead { return SourceRead{Outcome: ReadFound} },
	} {
		got := stellarRule(&Run{Config: Config{ReadChain: reader}}, "testnet", e)(record)
		want := SourceOutcome{Status: StatusSkip, Code: specdata.ReasonSourceUnreachable}
		if name == "a found nil value" {
			want.Notes = map[string]any{"unusable": []any{"successful", "memo_type", "memo", "ledger"}}
		}
		if got.Status != want.Status || got.Code != want.Code || !reflect.DeepEqual(got.Notes, want.Notes) || len(got.Reads) != 1 || got.Reads[0].Outcome == "maybe" ||
			(name == "a found nil value" && got.Reads[0].Value == nil) {
			t.Errorf("%s: %+v", name, got)
		}
	}
	run := &Run{Config: Config{ReadChain: func(Source, map[string]any) SourceRead {
		return SourceRead{Outcome: ReadNotFound, Value: map[string]any{"x": 1.0}, Reason: "said"}
	}}}
	if r := run.readChain(record, txRead("testnet")); r.Value != nil || r.Reason != "" || r.Outcome != ReadNotFound {
		t.Fatalf("a not-found answer %+v", r)
	}
}
