// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// CHAIN under an open gate (SPEC.md sections 10.3, the key-history rows,
// and 10.4, "Several sources", "Rows" and "What a row read"), over the
// positive item proof, its Stellar commitment pin testnet, every chain
// read answered by a chainStandIn by each source's URL.

// The close time of the item proof's ledger L, as a sound source gives it,
// and one second later.
const (
	itemClosed = "2026-09-26T01:52:15Z"
	itemLater  = "2026-09-26T01:52:16Z"
)

var (
	topStellar = Pointer{Name("commitments"), Name("stellar")}
	keyStellar = Pointer{Name("signing_key_event"), Name("commitments"), Name("stellar")}
)

// stellarAnswers are the reads of the Stellar entry at part of proof on
// testnet and the answers a sound source gives them, its ledger L closed
// at closed.
type stellarAnswers struct {
	tx, ledger             map[string]any
	txAnswer, ledgerAnswer SourceRead
}

func soundStellar(t *testing.T, proof map[string]any, part Pointer, closed string) stellarAnswers {
	t.Helper()
	entry, _ := valueAt(proof, part).(map[string]any)
	hash, _ := entry["transaction_hash"].(string)
	ledger := entry["ledger"]
	memo := modelHash(t, "epoch_hash", valueAt(proof, under(part, "epoch", "record")))
	return stellarAnswers{
		tx:           map[string]any{"kind": source.KindStellarTransaction, "network": "testnet", "hash": hash},
		ledger:       map[string]any{"kind": source.KindStellarLedger, "network": "testnet", "sequence": ledger},
		txAnswer:     found(map[string]any{"successful": true, "memo_type": "hash", "memo": memo, "ledger": ledger}),
		ledgerAnswer: found(map[string]any{"closed_at": closed, "hash": strings.Repeat("1", 64)}),
	}
}

// answer sets the stand-in's answers of each source of names to e's, with
// tx and ledger in place of e's answers where they are not the zero value.
func (s *chainStandIn) answer(e stellarAnswers, tx, ledger SourceRead, names ...string) {
	if tx.Outcome == "" {
		tx = e.txAnswer
	}
	if ledger.Outcome == "" {
		ledger = e.ledgerAnswer
	}
	for _, name := range names {
		url := "https://" + name + ".example"
		s.answers[standInKey(url, e.tx)] = tx
		s.answers[standInKey(url, e.ledger)] = ledger
	}
}

// openItemRun is the item proof's run past DERIVE, online, its gate open,
// with attribution, its Stellar commitment pin testnet served by set, every
// chain read answered by stand.
func openItemRun(t *testing.T, proof map[string]any, attribution string, set SourceSet, stand *chainStandIn) *Run {
	t.Helper()
	run := deriveRun(proof, attribution)
	run.Config.Online = true
	run.Config.Networks.Commitment.Stellar = "testnet"
	run.Config.ReadChain = stand.reader
	run.SourceSets = []SourceSet{set}
	Derive(run)
	if run.ExternalGate != ExternalGateOpen {
		t.Fatalf("the gate is %q, rows %s", run.ExternalGate, rowSummary(run.Rows))
	}
	return run
}

// candidateText writes the run's candidates as "<part> <instant> <counted
// sources>".
func candidateText(run *Run) string {
	var parts []string
	for _, c := range run.Candidates {
		var counted []string
		for _, a := range c.Counted {
			counted = append(counted, strings.TrimSuffix(strings.TrimPrefix(a.Source.Source.URL, "https://"), ".example"))
		}
		instant, _ := FormatTime(c.Candidate.Class, c.Candidate.Instant)
		parts = append(parts, fmt.Sprintf("%s %s %s %s %v", c.Step, c.Part, c.Candidate.Basis, instant, counted))
	}
	return strings.Join(parts, "; ")
}

// askedFor returns the reads the stand-in was asked of the source name.
func askedFor(stand *chainStandIn, name string) []string {
	var out []string
	for _, a := range stand.asked {
		if strings.HasPrefix(a, name+" ") {
			out = append(out, strings.TrimPrefix(a, name+" "))
		}
	}
	return out
}

// A Stellar entry's row under each policy, from several sources, each
// source's rule made as if it were the only one: the row's status and
// code, the candidate the run keeps, the sources it rests on and the reads
// each source was asked (SPEC.md section 10.4, "Several sources"). The
// item proof's entry under /signing_key_event names the same transaction
// as its top-level one, so the run reads it once from each source and the
// signing-key row takes the same answer, and reads no ledger: it passes
// where only a ledger differs, and an unreachable answer, which the run
// keeps for no row, is read again. Under first the sources are
// asked in order and none after the first conclusive one, a source whose
// identity was unreachable is asked nothing; under all and at_least n
// every source is asked, a pass beside a failure or two close times
// disagree, a pass beside an unreachable source is incomplete, and the
// candidate rests on every source that passed. Mutants killed: chainOpen
// reading the entropy pin's sources (no set, a panic); Counted taken as
// the first source alone (the candidate under all rests on one);
// readChain reading every source's answers from the run's first source
// (b's later close time agrees); the candidate kept with the asked sources in
// place of the counted ones (first lists the unreachable a); ReadSource
// keeping no answer (a's found transaction is read again for the
// signing-key row).
func TestChain_AStellarRowUnderEachPolicy(t *testing.T) {
	item := positiveProofs(t)["item"]
	top := soundStellar(t, item, topStellar, itemClosed)
	other := soundStellar(t, item, topStellar, itemLater)
	differs := soundTx(map[string]any{"memo": strings.Repeat("ab", 32)})
	first, all := SourcePolicy{Kind: SourcePolicyFirst}, SourcePolicy{Kind: SourcePolicyAll}
	atLeast2 := SourcePolicy{Kind: SourcePolicyAtLeast, N: 2}
	type answers struct {
		name       string
		tx, ledger SourceRead
		closed     bool
	}
	for _, tc := range []struct {
		name       string
		policy     SourcePolicy
		sources    []string
		answers    []answers
		row, key   string
		candidates string
		asked      map[string][]string
	}{
		{"first, the first passes", first, []string{"a", "b"}, []answers{{name: "a"}, {name: "b"}},
			"[null /commitments/stellar pass CHAIN]", "[null /signing_key_event/commitments/stellar pass CHAIN]", "CHAIN /commitments/stellar stellar_ledger_close_read 2026-09-26T01:52:15Z [a]",
			map[string][]string{"a": {"stellar_transaction", "stellar_ledger"}, "b": nil}},
		{"first, past an unreachable transaction", first, []string{"a", "b"}, []answers{{name: "a", tx: unreachableRead}, {name: "b"}},
			"[null /commitments/stellar pass CHAIN]", "[null /signing_key_event/commitments/stellar pass CHAIN]", "CHAIN /commitments/stellar stellar_ledger_close_read 2026-09-26T01:52:15Z [b]",
			map[string][]string{"a": {"stellar_transaction", "stellar_transaction"}, "b": {"stellar_transaction", "stellar_ledger"}}},
		{"first, past an unreachable identity", first, []string{"down", "b"}, []answers{{name: "b"}},
			"[null /commitments/stellar pass CHAIN]", "[null /signing_key_event/commitments/stellar pass CHAIN]", "CHAIN /commitments/stellar stellar_ledger_close_read 2026-09-26T01:52:15Z [b]",
			map[string][]string{"down": nil, "b": {"stellar_transaction", "stellar_ledger"}}},
		{"first, stops at a failure", first, []string{"a", "b"}, []answers{{name: "a", tx: differs}, {name: "b"}},
			"[chain_mismatch /commitments/stellar fail CHAIN]", "[chain_mismatch /signing_key_event/commitments/stellar fail CHAIN]", "",
			map[string][]string{"a": {"stellar_transaction"}, "b": nil}},
		{"all agree", all, []string{"a", "b"}, []answers{{name: "a"}, {name: "b"}},
			"[null /commitments/stellar pass CHAIN]", "[null /signing_key_event/commitments/stellar pass CHAIN]", "CHAIN /commitments/stellar stellar_ledger_close_read 2026-09-26T01:52:15Z [a b]", nil},
		{"all, another close time", all, []string{"a", "b"}, []answers{{name: "a"}, {name: "b", closed: true}},
			"[sources_disagree /commitments/stellar skip CHAIN]", "[null /signing_key_event/commitments/stellar pass CHAIN]", "", nil},
		{"all, a pass and a failure", all, []string{"a", "b"}, []answers{{name: "a"}, {name: "b", tx: differs}},
			"[sources_disagree /commitments/stellar skip CHAIN]", "[sources_disagree /signing_key_event/commitments/stellar skip CHAIN]", "", nil},
		{"all, a pass and an unreachable source", all, []string{"a", "b"}, []answers{{name: "a"}, {name: "b", tx: unreachableRead}},
			"[sources_incomplete /commitments/stellar skip CHAIN]", "[sources_incomplete /signing_key_event/commitments/stellar skip CHAIN]", "", nil},
		{"all fail", all, []string{"a", "b"}, []answers{{name: "a", tx: differs}, {name: "b", tx: differs}},
			"[chain_mismatch /commitments/stellar fail CHAIN]", "[chain_mismatch /signing_key_event/commitments/stellar fail CHAIN]", "", nil},
		{"all, neither holds the ledger", all, []string{"a", "b"}, []answers{{name: "a", ledger: notFound}, {name: "b", ledger: notFound}},
			"[not_held_by_source /commitments/stellar skip CHAIN]", "[null /signing_key_event/commitments/stellar pass CHAIN]", "", nil},
		{"at_least 2, one unreachable", atLeast2, []string{"a", "b", "c"}, []answers{{name: "a"}, {name: "b"}, {name: "c", tx: unreachableRead}},
			"[null /commitments/stellar pass CHAIN]", "[null /signing_key_event/commitments/stellar pass CHAIN]", "CHAIN /commitments/stellar stellar_ledger_close_read 2026-09-26T01:52:15Z [a b]", nil},
		{"at_least 2, two fail", atLeast2, []string{"a", "b", "c"}, []answers{{name: "a", tx: differs}, {name: "b", tx: differs}, {name: "c", tx: unreachableRead}},
			"[chain_mismatch /commitments/stellar fail CHAIN]", "[chain_mismatch /signing_key_event/commitments/stellar fail CHAIN]", "", nil},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		for _, a := range tc.answers {
			e := top
			if a.closed {
				e = other
			}
			stand.answer(e, a.tx, a.ledger, a.name)
		}
		run := openItemRun(t, item, deriveAttribution, standInSet(tc.policy, tc.sources...), stand)
		before := len(run.Rows)
		Chain(run)
		rows := run.Rows[before:]
		if got := rowSummary(rows); got != tc.row+" "+tc.key {
			t.Errorf("%s: rows %s, want %s %s", tc.name, got, tc.row, tc.key)
		}
		if got := candidateText(run); got != tc.candidates {
			t.Errorf("%s: candidates %q, want %q", tc.name, got, tc.candidates)
		}
		for name, want := range tc.asked {
			if got := askedFor(stand, name); !slices.Equal(got, want) {
				t.Errorf("%s: %s was asked %v, want %v", tc.name, name, got, want)
			}
		}
		if run.ExternalGate != ExternalGateOpen {
			t.Errorf("%s: the gate is %q", tc.name, run.ExternalGate)
		}
	}
}

// A row's values are what it read (SPEC.md section 10.4, "What a row
// read"), exactly: policy, required, one outcome for each source asked,
// each with its reads in replay form, and the candidate; its message names
// the transaction, ledger L and the close time whole. Mutants killed: the
// row made without CombinedOutcome.Values (no outcomes); the candidate
// left out of a passing row's values; a ledger read's sequence written as
// a string.
func TestChain_AStellarRowsValues(t *testing.T) {
	t0 := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	useClock(t, t0, t0.Add(time.Second), t0.Add(2*time.Second))
	item := positiveProofs(t)["item"]
	top := soundStellar(t, item, topStellar, itemClosed)
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	stand.answer(top, SourceRead{}, SourceRead{}, "a")
	run := openItemRun(t, item, deriveAttribution, standInSet(SourcePolicy{Kind: SourcePolicyFirst}, "a"), stand)
	Chain(run)
	var row Row
	for _, r := range run.Rows {
		if r.Step == "CHAIN" && *r.Part == "/commitments/stellar" {
			row = r
		}
	}
	got, err := jcs.Encode(row.Values)
	if err != nil {
		t.Fatal(err)
	}
	memo := top.txAnswer.Value["memo"].(string)
	hash := top.tx["hash"].(string)
	want := `{"candidate":{"basis":"stellar_ledger_close_read","instant":"2026-09-26T01:52:15Z"},` +
		`"outcomes":[{"cached":false,"candidate":{"basis":"stellar_ledger_close_read","instant":"2026-09-26T01:52:15Z"},"fetched_at":"2026-10-20T09:00:01.000Z",` +
		`"reads":[{"answer":{"outcome":"found","value":{"ledger":1000255,"memo":"` + memo + `","memo_type":"hash","successful":true}},"cached":false,"fetched_at":"2026-10-20T09:00:00.000Z",` +
		`"read":{"hash":"` + hash + `","kind":"stellar_transaction","network":"testnet"}},` +
		`{"answer":{"outcome":"found","value":{"closed_at":"2026-09-26T01:52:15Z","hash":"` + strings.Repeat("1", 64) + `"}},"cached":false,"fetched_at":"2026-10-20T09:00:01.000Z",` +
		`"read":{"kind":"stellar_ledger","network":"testnet","sequence":1000255}}],"source":"https://a.example","status":"pass"}],"policy":"first","required":1}`
	if string(got) != want {
		t.Fatalf("values\n%s\nwant\n%s", got, want)
	}
	wantMessage := "Read from the pinned Stellar testnet network: the transaction " + hash + " succeeded in ledger 1000255 with this proof's epoch hash as its memo, and that ledger closed at 2026-09-26T01:52:15Z."
	if row.Message != wantMessage {
		t.Fatalf("message %q", row.Message)
	}
}

// rowAt is the run's CHAIN row at part.
func rowAt(t *testing.T, run *Run, part Pointer) Row {
	t.Helper()
	for _, r := range run.Rows {
		if r.Step == "CHAIN" && r.Part != nil && *r.Part == part.Encode() {
			return r
		}
	}
	t.Fatalf("no CHAIN row at %s in %s", part.Encode(), rowSummary(run.Rows))
	return Row{}
}

// An unreachable read keeps the reason its reader gave (SPEC.md section
// 13.2: the reason says why) and no value, in the row's values as in the
// answer readChain gives. Mutants killed: readChain giving every
// unreachable answer a text of its own (the reason is not "HTTP 429");
// readChain keeping an unreachable answer's value.
func TestChain_AnUnreachableReadKeepsItsReason(t *testing.T) {
	item := positiveProofs(t)["item"]
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	stand.answer(soundStellar(t, item, topStellar, itemClosed), unreachableRead, SourceRead{}, "a")
	run := openItemRun(t, item, deriveAttribution, standInSet(SourcePolicy{Kind: SourcePolicyFirst}, "a"), stand)
	Chain(run)
	outcomes, _ := rowAt(t, run, topStellar).Values["outcomes"].([]any)
	if len(outcomes) != 1 {
		t.Fatalf("outcomes %v", outcomes)
	}
	reads, _ := outcomes[0].(map[string]any)["reads"].([]any)
	if len(reads) != 1 {
		t.Fatalf("reads %v", reads)
	}
	got, err := jcs.Encode(reads[0].(map[string]any)["answer"])
	if err != nil || string(got) != `{"outcome":"unreachable","reason":"HTTP 429"}` {
		t.Fatalf("the answer %s, %v", got, err)
	}
	record := standInSet(SourcePolicy{Kind: SourcePolicyFirst}, "a").Sources[0]
	direct := &Run{Config: Config{ReadChain: func(Source, map[string]any) SourceRead {
		return SourceRead{Outcome: ReadUnreachable, Value: map[string]any{"x": 1.0}, Reason: "timeout"}
	}}}
	if r := direct.readChain(record, txRead("testnet")); r.Outcome != ReadUnreachable || r.Reason != "timeout" || r.Value != nil {
		t.Fatalf("an unreachable answer %+v", r)
	}
}

// The messages of a Stellar row that is not_held_by_source or
// source_unreachable say what the reads that settled it found (SPEC.md
// section 10.5, "Rows": a message is for display): a source that held the
// transaction, matching the proof, was settled by its read of ledger L, so
// the message says the transaction matches and names the ledger; a source
// settled at the transaction or the root says it of the transaction; and
// sources settled each way get a message true of both. Mutants killed:
// the not_held_by_source message chosen from the code alone (the ledger
// not held says the transaction is not held); the source_unreachable
// message chosen from the code alone (the ledger unreachable says no
// usable answer about the transaction); settledOf counting a found
// transaction with no later read (a transaction with no memo names the
// ledger); settledOf with no form for mixed sources (a source not holding
// the transaction is said to hold it).
func TestChain_AStellarRowsMessageFromItsReads(t *testing.T) {
	item := positiveProofs(t)["item"]
	top := soundStellar(t, item, topStellar, itemClosed)
	hash := top.tx["hash"].(string)
	network := "the pinned Stellar testnet network"
	first, all := SourcePolicy{Kind: SourcePolicyFirst}, SourcePolicy{Kind: SourcePolicyAll}
	type answers struct {
		name             string
		tx, ledger, root SourceRead
	}
	for _, tc := range []struct {
		name    string
		policy  SourcePolicy
		answers []answers
		want    string
	}{
		{"the ledger not held", first, []answers{{name: "a", ledger: notFound}},
			"Not confirmed: " + network + "'s source holds the transaction " + hash + ", which matches this proof, but not ledger 1000255. Absence is no contradiction."},
		{"the ledger unreachable", first, []answers{{name: "a", ledger: unreachableRead}},
			"Not confirmed: the transaction " + hash + " matches this proof, but no source of " + network + " gave a usable answer about ledger 1000255."},
		{"the candidate ledger without closed_at", first, []answers{{name: "a", ledger: found(map[string]any{"hash": strings.Repeat("1", 64)})}},
			"Not confirmed: the transaction " + hash + " matches this proof, but no source of " + network + " gave a usable answer about ledger 1000255."},
		{"the transaction and root not held", first, []answers{{name: "a", tx: notFound, root: notFound}},
			"Not confirmed: the history " + network + "'s source keeps does not hold the transaction " + hash + " or ledger 1000255. Absence is no contradiction."},
		{"the transaction unreachable", first, []answers{{name: "a", tx: unreachableRead}},
			"Not confirmed: no source of " + network + " gave a usable answer about the transaction " + hash + "."},
		{"a transaction with no memo", first, []answers{{name: "a", tx: soundTx(map[string]any{"memo": nil, "ledger": top.txAnswer.Value["ledger"]})}},
			"Not confirmed: no source of " + network + " gave a usable answer about the transaction " + hash + "."},
		{"all, one holds the transaction and not the ledger, one neither", all, []answers{{name: "a", ledger: notFound}, {name: "b", tx: notFound, root: notFound}},
			"Not confirmed: " + network + "'s sources do not hold both the transaction " + hash + " and ledger 1000255. Absence is no contradiction."},
		{"all, the ledger unreachable at one, the transaction at the other", all, []answers{{name: "a", ledger: unreachableRead}, {name: "b", tx: unreachableRead}},
			"Not confirmed: no source of " + network + " gave usable answers about both the transaction " + hash + " and ledger 1000255."},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		var names []string
		for _, a := range tc.answers {
			names = append(names, a.name)
			stand.answer(top, a.tx, a.ledger, a.name)
			if a.root.Outcome != "" {
				stand.answers[standInKey("https://"+a.name+".example", rootRead("testnet"))] = a.root
			}
		}
		run := openItemRun(t, item, deriveAttribution, standInSet(tc.policy, names...), stand)
		Chain(run)
		if got := rowAt(t, run, topStellar).Message; got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}

// KEY_EVENT_COMMITMENT under an open gate (SPEC.md section 10.3), from a
// key event whose one input is its Stellar entry, whose transaction is
// the top-level entry's, so both rows take one answer: pass under the
// attribution truestamp, info not_attributable under a supplied keyring's,
// fail with the input's code when it failed and skip with the input's
// code when it skipped; and no row while an input, the item proof's
// Bitcoin entry under /signing_key_event, has no row yet. Mutants killed:
// keyEventCommitment passing whatever the attribution (the keyring run
// passes); taking only failures (the skip passes); giving the row whatever
// its inputs (a row with the Bitcoin input missing); dropping the row
// (none is given).
func TestChain_TheKeyEventRowFromItsStellarInput(t *testing.T) {
	item := positiveProofs(t)["item"]
	stellarOnly := clone(item).(map[string]any)
	delete(stellarOnly["signing_key_event"].(map[string]any)["commitments"].(map[string]any), "bitcoin")
	for _, tc := range []struct {
		name        string
		proof       map[string]any
		attribution string
		keyTx       SourceRead
		want        string
	}{
		{"truestamp", stellarOnly, AttributionTruestamp, SourceRead{}, "[null /signing_key_event pass CHAIN]"},
		{"a supplied keyring", stellarOnly, deriveAttribution, SourceRead{}, "[not_attributable /signing_key_event info CHAIN]"},
		{"the input fails", stellarOnly, AttributionTruestamp, soundTx(map[string]any{"successful": false}), "[chain_mismatch /signing_key_event fail CHAIN]"},
		{"the input skips", stellarOnly, AttributionTruestamp, unreachableRead, "[source_unreachable /signing_key_event skip CHAIN]"},
		{"a Bitcoin input", item, AttributionTruestamp, SourceRead{}, ""},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		stand.answer(soundStellar(t, item, topStellar, itemClosed), tc.keyTx, SourceRead{}, "a")
		run := deriveRun(tc.proof, tc.attribution)
		run.Config = Config{Online: true, ReadChain: stand.reader}
		run.Config.Networks.Commitment.Stellar = "testnet"
		run.SourceSets = []SourceSet{standInSet(SourcePolicy{Kind: SourcePolicyFirst}, "a")}
		run.ExternalGate = ExternalGateOpen
		Chain(run)
		var got []Row
		for _, r := range run.Rows {
			if r.Step == "CHAIN" && *r.Part == keyEventCommitmentPart {
				got = append(got, r)
			}
		}
		if rowSummary(got) != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, rowSummary(got), tc.want)
		}
		if last := run.Rows[len(run.Rows)-1]; tc.want != "" && *last.Part != keyEventCommitmentPart {
			t.Errorf("%s: the key event's row is not CHAIN's last", tc.name)
		}
	}
}

// KEY_EVENT_COMMITMENT's status is the first that applies of a failing
// input, by the first failing input in document order, then a skipped
// one, by the first skipped input in document order (SPEC.md section
// 10.3), over two inputs whose rows are given: a skip of the Bitcoin
// entry beside a failure of the Stellar entry fails; two skips take the
// Bitcoin entry's code, its row first in document order; a genesis input
// comes before the commitments; a key event's input with no row gives no
// row. Mutants killed: the first non-passing input taken whatever its
// status (the skip wins over the failure); the last skipped input taken
// (not_held_by_source in place of chain_not_final); keyEventInputs
// without its sort (the genesis input after the commitments).
func TestChain_TheKeyEventRowsOrder(t *testing.T) {
	bitcoinAt := Pointer{Name("signing_key_event"), Name("commitments"), Name("bitcoin")}
	genesisAt := append(Pointer{Name("signing_key_event"), Name("block")}, genesisBitcoinAt...)
	proof := map[string]any{"signing_key_event": map[string]any{
		"commitments": map[string]any{"bitcoin": map[string]any{}, "stellar": map[string]any{}},
		"block":       map[string]any{"record": map[string]any{"metadata": map[string]any{"genesis": map[string]any{"bitcoin": map[string]any{}}}}},
	}}
	row := func(at Pointer, code specdata.Reason) chainRow {
		return chainRow{at: at, row: NewRow("CHAIN", code, Part(at.Encode()), "", nil)}
	}
	if got := keyEventInputs(proof); !reflect.DeepEqual(got, []Pointer{genesisAt, bitcoinAt, keyStellar}) {
		t.Fatalf("inputs %v", got)
	}
	for _, tc := range []struct {
		name string
		rows []chainRow
		want string
	}{
		{"a skip, then a failure", []chainRow{row(genesisAt, ""), row(bitcoinAt, specdata.ReasonChainNotFinal), row(keyStellar, specdata.ReasonChainMismatch)},
			"[chain_mismatch /signing_key_event fail CHAIN]"},
		{"two skips", []chainRow{row(genesisAt, ""), row(bitcoinAt, specdata.ReasonChainNotFinal), row(keyStellar, specdata.ReasonNotHeldBySource)},
			"[chain_not_final /signing_key_event skip CHAIN]"},
		{"the genesis input first", []chainRow{row(genesisAt, specdata.ReasonSourceUnreachable), row(bitcoinAt, specdata.ReasonChainNotFinal), row(keyStellar, "")},
			"[source_unreachable /signing_key_event skip CHAIN]"},
		{"every input passes", []chainRow{row(genesisAt, ""), row(bitcoinAt, ""), row(keyStellar, "")},
			"[null /signing_key_event pass CHAIN]"},
		{"an input with no row", []chainRow{row(genesisAt, ""), row(keyStellar, "")}, ""},
	} {
		got, ok := keyEventCommitment(&Run{Proof: proof, Attribution: AttributionTruestamp}, tc.rows)
		text := ""
		if ok {
			text = rowSummary([]Row{got})
		}
		if text != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, text, tc.want)
		}
	}
}

// A Stellar entry CHAIN cannot read, one SHAPE should have refused, is a
// fault: internal_error at CHAIN, part null, and the run ends, its later
// steps' rows skip prior_failure, with no read made. Mutants killed: the
// fault not ending the run (EXPECTED_HASH gives its own row); the fault
// ignored (the entries are read).
func TestChain_AnUnreadableEntryIsAFault(t *testing.T) {
	item := positiveProofs(t)["item"]
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	run := openItemRun(t, item, deriveAttribution, standInSet(SourcePolicy{Kind: SourcePolicyFirst}, "a"), stand)
	run.Proof = put(item, under(keyStellar, "ledger"), "1000255")
	before := len(run.Rows)
	Chain(run)
	Sources(run)
	Window(run)
	ExpectedHash(run)
	if got := rowSummary(run.Rows[before:]); got != "[internal_error null fail CHAIN] [prior_failure null skip SOURCES] [prior_failure null skip WINDOW] [prior_failure null skip EXPECTED_HASH]" || !run.Ended || len(stand.asked) != 0 {
		t.Fatalf("rows %s, ended %v, reads %v", got, run.Ended, stand.asked)
	}
}

// DefaultChainReader builds each read's SingleRead when the read is made,
// with the default transport and SourceReadDeadline, and gives the reader
// of internal/verifier/source's answer: a Horizon transaction in replay
// form against a loopback Horizon; a not-found answer read live every
// time, the same read made twice through one reader and once through
// another reaching a loopback Horizon that counts its requests three
// times (SPEC.md section 10.4, "Caches"); a read ending at its deadline
// when the server never answers; and, for each compiled-in Stellar
// default, a read refused in process once by this package's TestMain,
// unreachable (the control of the chain reads). Mutants killed:
// DefaultChainReader building its SingleRead once, when it is called; the
// answer's outcome dropped (found read as the zero outcome);
// DefaultChainReader keeping each not-found answer by URL and read (two
// requests); source.Read keeping every not-found answer for the process
// by base and read (one request).
func TestChain_TheDefaultReader(t *testing.T) {
	memo := strings.Repeat("ab", 32)
	horizon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transactions/"+ruleHash {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"successful":true,"ledger":7,"memo_type":"hash","memo":"q6urq6urq6urq6urq6urq6urq6urq6urq6urq6urq6s="}`))
	}))
	defer horizon.Close()
	var built []string
	saved := newSingleRead
	newSingleRead = func(transport http.RoundTripper, deadline time.Duration) *httpclient.SingleRead {
		built = append(built, fmt.Sprintf("%v %v", transport, deadline))
		return saved(transport, deadline)
	}
	defer func() { newSingleRead = saved }()
	reader := DefaultChainReader()
	if len(built) != 0 {
		t.Fatalf("a reader built %d reads before any read", len(built))
	}
	s := Source{Chain: ChainStellar, Provider: source.Horizon, URL: horizon.URL, Network: "testnet"}
	got := reader(s, map[string]any{"kind": source.KindStellarTransaction, "network": "testnet", "hash": ruleHash})
	want := SourceRead{Outcome: ReadFound, Value: map[string]any{"successful": true, "ledger": strictjson.Number(7), "memo_type": "hash", "memo": memo}}
	if !reflect.DeepEqual(got, want) || len(built) != 1 || built[0] != "<nil> 9s" {
		t.Fatalf("%+v, reads built %q", got, built)
	}
	if got := reader(s, map[string]any{"kind": source.KindStellarLedger, "network": "testnet", "sequence": strictjson.Number(7)}); got.Outcome != ReadNotFound || len(built) != 2 {
		t.Fatalf("a ledger Horizon does not hold: %+v", got)
	}

	var requests atomic.Int64
	absent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.NotFound(w, r)
	}))
	defer absent.Close()
	missing := Source{Chain: ChainStellar, Provider: source.Horizon, URL: absent.URL, Network: "testnet"}
	ledger := map[string]any{"kind": source.KindStellarLedger, "network": "testnet", "sequence": strictjson.Number(9)}
	for i, r := range []ChainReader{reader, reader, DefaultChainReader()} {
		if got := r(missing, ledger); got.Outcome != ReadNotFound {
			t.Fatalf("read %d of a ledger Horizon does not hold: %+v", i, got)
		}
	}
	if n := requests.Load(); n != 3 {
		t.Fatalf("three not-found reads reached Horizon %d times", n)
	}

	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer slow.Close()
	defer close(release)
	newSingleRead = func(transport http.RoundTripper, _ time.Duration) *httpclient.SingleRead {
		return saved(transport, 300*time.Millisecond)
	}
	start := time.Now()
	cut := DefaultChainReader()(Source{Chain: ChainStellar, Provider: source.Horizon, URL: slow.URL}, rootRead("testnet"))
	if cut.Outcome != ReadUnreachable || cut.Reason != "no answer within the read's deadline" || time.Since(start) > 3*time.Second {
		t.Fatalf("%+v after %v", cut, time.Since(start))
	}
	newSingleRead = saved

	for _, network := range []string{"public", "testnet"} {
		s, _ := DefaultSource(ChainStellar, network)
		before := refusals.Load()
		got := DefaultChainReader()(s, rootRead(network))
		if got.Outcome != ReadUnreachable || refusals.Load()-before != 1 {
			t.Fatalf("%s: %+v, %d refused", s.URL, got, refusals.Load()-before)
		}
	}
}
