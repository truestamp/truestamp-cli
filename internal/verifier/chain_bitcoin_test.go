// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// CHAIN's rule for a Bitcoin commitment entry and for genesis.bitcoin
// against one source (SPEC.md section 10.4, "Bitcoin commitment entries",
// "genesis.bitcoin", "Answers" and "Rows"), over chain_stellar_test.go's
// stand-in chain reader.

// The entry the rule is made for: its height h, the header hash the block
// at h must have, its total_txs, and the median time past of the block at
// h + 10.
const (
	ruleHeight     = 222
	ruleHeaderHash = "066ee8ed37e3b69aaef26bca28f2818674e61412e1be5bc8f17b0e4cc5971386"
	ruleTotalTxs   = 5
	ruleMedian     = 1790387820
)

// The reads of the rule's entry on network.
func tipRead(network string) map[string]any {
	return map[string]any{"kind": source.KindBitcoinTip, "network": network}
}

func blockRead(network string, height int64) map[string]any {
	return map[string]any{"kind": source.KindBitcoinBlockAtHeight, "network": network, "height": strictjson.Number(float64(height))}
}

// tip is a tip answer at height.
func tip(height int64) SourceRead {
	return found(map[string]any{"height": strictjson.Number(float64(height))})
}

// soundBlock is the block at h the rule passes, with members replaced by
// change (a nil value removes one).
func soundBlock(change map[string]any) SourceRead {
	value := map[string]any{"hash": ruleHeaderHash, "transaction_count": strictjson.Number(ruleTotalTxs), "median_time_past": strictjson.Number(1790387220)}
	for name, v := range change {
		if v == nil {
			delete(value, name)
		} else {
			value[name] = v
		}
	}
	return found(value)
}

// finalBlock is a block at h + 10 whose median time past is median.
func finalBlock(median any) SourceRead {
	return found(map[string]any{"hash": strings.Repeat("2", 64), "transaction_count": strictjson.Number(1), "median_time_past": median})
}

// The kinds of entry the rule is made for: the top-level entry, the one
// under /signing_key_event, and a genesis.bitcoin.
var (
	topEntry     = bitcoinEntry{at: Pointer{Name("commitments"), Name("bitcoin")}, topLevel: true, height: ruleHeight, hash: ruleHeaderHash, totalTxs: ruleTotalTxs}
	keyEntry     = bitcoinEntry{at: Pointer{Name("signing_key_event"), Name("commitments"), Name("bitcoin")}, height: ruleHeight, hash: ruleHeaderHash, totalTxs: ruleTotalTxs}
	genesisEntry = bitcoinEntry{at: append(Pointer{Name("block")}, genesisBitcoinAt...), genesis: true, height: ruleHeight, hash: ruleHeaderHash}
)

// bitcoinOutcome makes the rule of e against one confirmed source of
// network, https://a.example, whose answers are reads, by "tip", "block"
// (the block at h) and "final" (the block at h + 10), and gives its outcome
// and the reads the stand-in was asked, each "a <kind> <height>".
func bitcoinOutcome(network string, e bitcoinEntry, reads map[string]SourceRead) (SourceOutcome, []string) {
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	for name, answer := range reads {
		read := map[string]map[string]any{"tip": tipRead(network), "block": blockRead(network, e.height), "final": blockRead(network, e.height+10)}[name]
		stand.answers[standInKey("https://a.example", read)] = answer
	}
	run := &Run{Config: Config{ReadChain: stand.reader}}
	record := bitcoinSet(SourcePolicy{Kind: SourcePolicyFirst}, network, "a").Sources[0]
	got := bitcoinRule(run, network, e)(record)
	var asked []string
	for _, r := range got.Reads {
		text := "a " + r.Read["kind"].(string)
		if h, ok := intField(r.Read, "height"); ok {
			text += fmt.Sprint(" ", h)
		}
		asked = append(asked, text)
	}
	return got, asked
}

// bitcoinSet is a set of the Bitcoin network network with the confirmed
// Esplora sources https://<name>.example under policy, a name starting
// with "down" one whose identity read was unreachable.
func bitcoinSet(policy SourcePolicy, network string, names ...string) SourceSet {
	set := standInSet(policy, names...)
	set.Chain, set.Network = ChainBitcoin, network
	for i := range set.Sources {
		set.Sources[i].Source.Chain, set.Sources[i].Source.Provider, set.Sources[i].Source.Network = ChainBitcoin, source.Esplora, network
	}
	return set
}

// Every outcome of the rule, on mainnet and on regtest alike, from the
// answers of one source (SPEC.md section 10.4): items 1 and 2, the tip
// below h not_yet_indexed and at h or up to h + 9 chain_not_final, each
// with notes naming h and the tip, and at h + 10 read on; item 3, the
// hash and transaction_count at h held to the recomputed header hash and
// total_txs, each that differs failing chain_mismatch, a contradiction
// failing even beside a field the answer lacks, an answer judged on the
// fields it holds (the block at h without median_time_past passes), a
// field absent or in another form source_unreachable naming it; item 4,
// for the top-level entry only, the block at h + 10's median time past
// the candidate, an INT a TIME can write, the block judged on it alone;
// genesis.bitcoin's hash alone compared and no block past h read; a block
// or a tip not found not_held_by_source, naming the height read; an
// unreachable read at each read. Mutants killed: the tip check h > t
// written h >= t (the tip at h is not_yet_indexed); BITCOIN_FINALITY taken
// as 9 (the tip at h + 9 reads on) or the finality check written t <= h +
// 10 (the tip at h + 10 is chain_not_final); the hash not compared (another
// hash passes); transaction_count not compared, or compared for
// genesis.bitcoin (another count passes, or the genesis with another count
// fails); unusable fields judged before contradictions (another hash with no
// count skips); the block at h required to hold median_time_past (it skips);
// the candidate read for every entry (the key entry reads h + 10); the
// candidate taken from the block at h (its median differs); the median read
// without its TIME bound (a median past 9999 passes); intField taking a
// negative tip; the block at h + 10 judged on its hash or count (a final
// block with neither skips); a not-found block read as a contradiction.
func TestBitcoinRule_EveryOutcome(t *testing.T) {
	h := int64(ruleHeight)
	sound := map[string]SourceRead{"tip": tip(h + 10), "block": soundBlock(nil), "final": finalBlock(strictjson.Number(ruleMedian))}
	with := func(change map[string]SourceRead) map[string]SourceRead {
		out := map[string]SourceRead{}
		for k, v := range sound {
			out[k] = v
		}
		for k, v := range change {
			out[k] = v
		}
		return out
	}
	passTop := fmt.Sprintf("pass  - [a bitcoin_tip a bitcoin_block_at_height 222 a bitcoin_block_at_height 232] candidate bitcoin_median_time_read %d", ruleMedian)
	for _, network := range []string{"mainnet", "regtest"} {
		for _, tc := range []struct {
			name  string
			e     bitcoinEntry
			reads map[string]SourceRead
			want  string
		}{
			{"the tip at h + 10, top-level", topEntry, sound, passTop},
			{"a later tip", topEntry, with(map[string]SourceRead{"tip": tip(h + 500)}), passTop},
			{"under the key event", keyEntry, sound, "pass  - [a bitcoin_tip a bitcoin_block_at_height 222]"},
			{"genesis.bitcoin", genesisEntry, sound, "pass  - [a bitcoin_tip a bitcoin_block_at_height 222]"},
			{"genesis.bitcoin, the block with its hash alone", genesisEntry, with(map[string]SourceRead{"block": found(map[string]any{"hash": ruleHeaderHash})}),
				"pass  - [a bitcoin_tip a bitcoin_block_at_height 222]"},
			{"genesis.bitcoin, another count", genesisEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"transaction_count": strictjson.Number(9)})}),
				"pass  - [a bitcoin_tip a bitcoin_block_at_height 222]"},
			{"the block at h without median_time_past", topEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"median_time_past": nil})}), passTop},
			{"the final block with its median alone", topEntry, with(map[string]SourceRead{"final": found(map[string]any{"median_time_past": strictjson.Number(ruleMedian)})}), passTop},
			{"the tip below h", topEntry, with(map[string]SourceRead{"tip": tip(h - 1)}),
				`skip not_yet_indexed {"height":222,"tip":221} [a bitcoin_tip]`},
			{"the tip at h", keyEntry, with(map[string]SourceRead{"tip": tip(h)}),
				`skip chain_not_final {"height":222,"tip":222} [a bitcoin_tip]`},
			{"the tip at h + 9", genesisEntry, with(map[string]SourceRead{"tip": tip(h + 9)}),
				`skip chain_not_final {"height":222,"tip":231} [a bitcoin_tip]`},
			{"another hash", topEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"hash": strings.Repeat("ab", 32)})}),
				`fail chain_mismatch {"differs":["hash"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"another count", keyEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"transaction_count": strictjson.Number(4)})}),
				`fail chain_mismatch {"differs":["transaction_count"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"both differ", topEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"hash": strings.Repeat("ab", 32), "transaction_count": strictjson.Number(4)})}),
				`fail chain_mismatch {"differs":["hash","transaction_count"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"another hash, no count", topEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"hash": strings.Repeat("ab", 32), "transaction_count": nil})}),
				`fail chain_mismatch {"differs":["hash"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"genesis.bitcoin, another hash", genesisEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"hash": strings.Repeat("ab", 32)})}),
				`fail chain_mismatch {"differs":["hash"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"no count", topEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"transaction_count": nil})}),
				`skip source_unreachable {"unusable":["transaction_count"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"a count string", topEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"transaction_count": "5"})}),
				`skip source_unreachable {"unusable":["transaction_count"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"an uppercase hash", keyEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"hash": strings.ToUpper(ruleHeaderHash)})}),
				`skip source_unreachable {"unusable":["hash"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"genesis.bitcoin, no hash", genesisEntry, with(map[string]SourceRead{"block": soundBlock(map[string]any{"hash": nil})}),
				`skip source_unreachable {"unusable":["hash"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"an empty block", topEntry, with(map[string]SourceRead{"block": found(map[string]any{})}),
				`skip source_unreachable {"unusable":["hash","transaction_count"]} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"the final block without its median", topEntry, with(map[string]SourceRead{"final": finalBlock(nil)}),
				`skip source_unreachable {"unusable":["median_time_past"]} [a bitcoin_tip a bitcoin_block_at_height 222 a bitcoin_block_at_height 232]`},
			{"the final block's median a string", topEntry, with(map[string]SourceRead{"final": finalBlock("1790387820")}),
				`skip source_unreachable {"unusable":["median_time_past"]} [a bitcoin_tip a bitcoin_block_at_height 222 a bitcoin_block_at_height 232]`},
			{"the final block's median past 9999", topEntry, with(map[string]SourceRead{"final": finalBlock(strictjson.Number(253402300800))}),
				`skip source_unreachable {"unusable":["median_time_past"]} [a bitcoin_tip a bitcoin_block_at_height 222 a bitcoin_block_at_height 232]`},
			{"the final block not found", topEntry, with(map[string]SourceRead{"final": notFound}),
				`skip not_held_by_source {"height":232} [a bitcoin_tip a bitcoin_block_at_height 222 a bitcoin_block_at_height 232]`},
			{"the final block unreachable", topEntry, with(map[string]SourceRead{"final": unreachableRead}),
				`skip source_unreachable - [a bitcoin_tip a bitcoin_block_at_height 222 a bitcoin_block_at_height 232]`},
			{"the block at h not found", genesisEntry, with(map[string]SourceRead{"block": notFound}),
				`skip not_held_by_source {"height":222} [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"the block at h unreachable", topEntry, with(map[string]SourceRead{"block": unreachableRead}),
				`skip source_unreachable - [a bitcoin_tip a bitcoin_block_at_height 222]`},
			{"the tip unreachable", topEntry, with(map[string]SourceRead{"tip": unreachableRead}),
				`skip source_unreachable - [a bitcoin_tip]`},
			{"the tip not found", keyEntry, with(map[string]SourceRead{"tip": notFound}),
				`skip not_held_by_source - [a bitcoin_tip]`},
			{"the tip without height", topEntry, with(map[string]SourceRead{"tip": found(map[string]any{})}),
				`skip source_unreachable {"unusable":["height"]} [a bitcoin_tip]`},
			{"a negative tip", topEntry, with(map[string]SourceRead{"tip": found(map[string]any{"height": strictjson.Number(-1)})}),
				`skip source_unreachable {"unusable":["height"]} [a bitcoin_tip]`},
		} {
			got, asked := bitcoinOutcome(network, tc.e, tc.reads)
			if text := outcomeText(got, asked); text != tc.want {
				t.Errorf("%s, %s:\n got %s\nwant %s", network, tc.name, text, tc.want)
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
// network} for the tip and {kind, network, height} for a block, the height
// a number, with the answer and the time it came back, and the
// candidate's fetched_at is the block at h + 10's read's. Mutants killed:
// the height written as a string (the read differs); the candidate's
// FetchedAt the tip's read's.
func TestBitcoinRule_TheReadsItRecords(t *testing.T) {
	t0 := time.Date(2026, 10, 20, 9, 0, 1, 0, time.UTC)
	useClock(t, t0, t0.Add(time.Second), t0.Add(2*time.Second))
	final := finalBlock(strictjson.Number(ruleMedian))
	got, _ := bitcoinOutcome("regtest", topEntry, map[string]SourceRead{"tip": tip(300), "block": soundBlock(nil), "final": final})
	want := []SourceRead{
		{Read: tipRead("regtest"), Outcome: ReadFound, Value: tip(300).Value, FetchedAt: t0},
		{Read: blockRead("regtest", 222), Outcome: ReadFound, Value: soundBlock(nil).Value, FetchedAt: t0.Add(time.Second)},
		{Read: blockRead("regtest", 232), Outcome: ReadFound, Value: final.Value, FetchedAt: t0.Add(2 * time.Second)},
	}
	if !reflect.DeepEqual(got.Reads, want) || got.Candidate == nil || !got.Candidate.FetchedAt.Equal(t0.Add(2*time.Second)) || got.Candidate.Class != ClassTIME {
		t.Fatalf("reads %+v, candidate %+v", got.Reads, got.Candidate)
	}
}

// itemBitcoin is a sound Bitcoin source's answers for the item proof's
// entries, both at height 222 with one header: its tip at tipAt, its
// block at 222 with the header hash and total_txs the proof's txoutproof
// carries, read here from its bytes, and its block at 232 with the median
// time past median.
type itemBitcoin struct {
	tip, block, final SourceRead
}

func soundItemBitcoin(t *testing.T, proof map[string]any, tipAt, median int64) itemBitcoin {
	t.Helper()
	entry, _ := valueAt(proof, Pointer{Name("commitments"), Name("bitcoin")}).(map[string]any)
	raw, err := hex.DecodeString(entry["txoutproof"].(string))
	if err != nil || len(raw) < 84 {
		t.Fatalf("the txoutproof: %v", err)
	}
	total := int64(raw[80]) | int64(raw[81])<<8 | int64(raw[82])<<16 | int64(raw[83])<<24
	return itemBitcoin{
		tip:   tip(tipAt),
		block: found(map[string]any{"hash": entry["header_hash"], "transaction_count": strictjson.Number(float64(total)), "median_time_past": strictjson.Number(1790387220)}),
		final: finalBlock(strictjson.Number(float64(median))),
	}
}

// answerBitcoin sets the stand-in's answers of each source of names, on
// regtest, to b's, with any of tip, block and final given in place of b's
// where it is not the zero value.
func (s *chainStandIn) answerBitcoin(b itemBitcoin, tip, block, final SourceRead, names ...string) {
	for _, a := range []struct {
		answer *SourceRead
		given  SourceRead
	}{{&b.tip, tip}, {&b.block, block}, {&b.final, final}} {
		if a.given.Outcome != "" {
			*a.answer = a.given
		}
	}
	for _, name := range names {
		url := "https://" + name + ".example"
		s.answers[standInKey(url, tipRead("regtest"))] = b.tip
		s.answers[standInKey(url, blockRead("regtest", ruleHeight))] = b.block
		s.answers[standInKey(url, blockRead("regtest", ruleHeight+10))] = b.final
	}
}

// keyBitcoin is the Bitcoin entry under /signing_key_event, beside
// derive_bitcoin_test.go's topBitcoin.
var keyBitcoin = Pointer{Name("signing_key_event"), Name("commitments"), Name("bitcoin")}

// bitcoinRows writes the run's CHAIN rows at the two Bitcoin entries and
// at /signing_key_event.
func bitcoinRows(run *Run) string {
	var rows []Row
	for _, r := range run.Rows {
		if r.Step == "CHAIN" && r.Part != nil && (*r.Part == topBitcoin.Encode() || *r.Part == keyBitcoin.Encode() || *r.Part == keyEventCommitmentPart) {
			rows = append(rows, r)
		}
	}
	return rowSummary(rows)
}

// The item proof's Bitcoin rows under each policy, from several sources,
// each source's rule made as if it were the only one (SPEC.md section
// 10.4, "Several sources"), its Stellar entries served by one sound
// source: the rows' status and code, KEY_EVENT_COMMITMENT after them, the
// candidate the run keeps and the sources it rests on, and the reads each
// source was asked. The entry under /signing_key_event names the
// top-level entry's height, so a source's tip and block at 222 are read
// once for both rows. Under all, tips a block apart, both at least h +
// 10, pass and agree; every tip below h is not_yet_indexed; a tip below h
// beside one below h + 10 is sources_incomplete, as under first; a median
// time past a second later disagrees; a block whose hash differs beside a
// sound one disagrees. Under first, a source past an unreachable tip
// passes alone. Under at_least 2 of three, two passes beside an
// unreachable source pass. Mutants killed: bitcoinChainEntry reading from
// the Stellar commitment pin's sources (the Bitcoin reads go to a Horizon
// source, and every Bitcoin row skips); CombineSources made with the first
// source alone (lagging tips rest on one source); the tip read live for
// each row (a source asked for its tip twice); every Bitcoin entry given
// the top-level entry's candidate (the key entry reads 232 again, and its
// row keeps a candidate).
func TestChain_ABitcoinRowUnderEachPolicy(t *testing.T) {
	item := positiveProofs(t)["item"]
	sound := soundItemBitcoin(t, item, 233, ruleMedian)
	first, all := SourcePolicy{Kind: SourcePolicyFirst}, SourcePolicy{Kind: SourcePolicyAll}
	atLeast2 := SourcePolicy{Kind: SourcePolicyAtLeast, N: 2}
	type answers struct {
		name              string
		tip, block, final SourceRead
	}
	differs := soundBlock(map[string]any{"hash": strings.Repeat("ab", 32)})
	for _, tc := range []struct {
		name       string
		policy     SourcePolicy
		sources    []string
		answers    []answers
		rows       string
		candidates string
		asked      map[string][]string
	}{
		{"all, tips a block apart", all, []string{"a", "b"}, []answers{{name: "a"}, {name: "b", tip: tip(232)}},
			"[null /commitments/bitcoin pass CHAIN] [null /signing_key_event/commitments/bitcoin pass CHAIN] [not_attributable /signing_key_event info CHAIN]",
			"CHAIN /commitments/bitcoin bitcoin_median_time_read 2026-09-26T01:57:00Z [a b]",
			map[string][]string{"a": {"bitcoin_tip", "bitcoin_block_at_height", "bitcoin_block_at_height"}, "b": {"bitcoin_tip", "bitcoin_block_at_height", "bitcoin_block_at_height"}}},
		{"all, every tip below h", all, []string{"a", "b"}, []answers{{name: "a", tip: tip(220)}, {name: "b", tip: tip(221)}},
			"[not_yet_indexed /commitments/bitcoin skip CHAIN] [not_yet_indexed /signing_key_event/commitments/bitcoin skip CHAIN] [not_yet_indexed /signing_key_event skip CHAIN]", "",
			map[string][]string{"a": {"bitcoin_tip"}, "b": {"bitcoin_tip"}}},
		{"all, one below h and one below h + 10", all, []string{"a", "b"}, []answers{{name: "a", tip: tip(221)}, {name: "b", tip: tip(231)}},
			"[sources_incomplete /commitments/bitcoin skip CHAIN] [sources_incomplete /signing_key_event/commitments/bitcoin skip CHAIN] [sources_incomplete /signing_key_event skip CHAIN]", "", nil},
		{"first, one below h and one below h + 10", first, []string{"a", "b"}, []answers{{name: "a", tip: tip(221)}, {name: "b", tip: tip(231)}},
			"[sources_incomplete /commitments/bitcoin skip CHAIN] [sources_incomplete /signing_key_event/commitments/bitcoin skip CHAIN] [sources_incomplete /signing_key_event skip CHAIN]", "", nil},
		{"all, another median", all, []string{"a", "b"}, []answers{{name: "a"}, {name: "b", final: finalBlock(strictjson.Number(ruleMedian + 1))}},
			"[sources_disagree /commitments/bitcoin skip CHAIN] [null /signing_key_event/commitments/bitcoin pass CHAIN] [not_attributable /signing_key_event info CHAIN]", "", nil},
		{"all, a pass and a failure", all, []string{"a", "b"}, []answers{{name: "a"}, {name: "b", block: differs}},
			"[sources_disagree /commitments/bitcoin skip CHAIN] [sources_disagree /signing_key_event/commitments/bitcoin skip CHAIN] [sources_disagree /signing_key_event skip CHAIN]", "", nil},
		{"first, past an unreachable tip", first, []string{"a", "b"}, []answers{{name: "a", tip: unreachableRead}, {name: "b"}},
			"[null /commitments/bitcoin pass CHAIN] [null /signing_key_event/commitments/bitcoin pass CHAIN] [not_attributable /signing_key_event info CHAIN]",
			"CHAIN /commitments/bitcoin bitcoin_median_time_read 2026-09-26T01:57:00Z [b]",
			map[string][]string{"a": {"bitcoin_tip", "bitcoin_tip"}, "b": {"bitcoin_tip", "bitcoin_block_at_height", "bitcoin_block_at_height"}}},
		{"first, past an unreachable identity", first, []string{"down", "b"}, []answers{{name: "b"}},
			"[null /commitments/bitcoin pass CHAIN] [null /signing_key_event/commitments/bitcoin pass CHAIN] [not_attributable /signing_key_event info CHAIN]",
			"CHAIN /commitments/bitcoin bitcoin_median_time_read 2026-09-26T01:57:00Z [b]", map[string][]string{"down": nil}},
		{"at_least 2, one unreachable", atLeast2, []string{"a", "b", "c"}, []answers{{name: "a"}, {name: "b"}, {name: "c", tip: unreachableRead}},
			"[null /commitments/bitcoin pass CHAIN] [null /signing_key_event/commitments/bitcoin pass CHAIN] [not_attributable /signing_key_event info CHAIN]",
			"CHAIN /commitments/bitcoin bitcoin_median_time_read 2026-09-26T01:57:00Z [a b]", nil},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		for _, a := range tc.answers {
			stand.answerBitcoin(sound, a.tip, a.block, a.final, a.name)
		}
		run := openItemRunWith(t, item, deriveAttribution, bitcoinSet(tc.policy, "regtest", tc.sources...), stand)
		Chain(run)
		if got := bitcoinRows(run); got != tc.rows {
			t.Errorf("%s: rows %s, want %s", tc.name, got, tc.rows)
		}
		var candidates []RowCandidate
		for _, c := range run.Candidates {
			if c.Part == topBitcoin.Encode() || c.Part == keyBitcoin.Encode() {
				candidates = append(candidates, c)
			}
		}
		if got := candidateText(&Run{Candidates: candidates}); got != tc.candidates {
			t.Errorf("%s: candidates %q, want %q", tc.name, got, tc.candidates)
		}
		for name, want := range tc.asked {
			if got := askedFor(stand, name); !reflect.DeepEqual(got, want) {
				t.Errorf("%s: %s was asked %v, want %v", tc.name, name, got, want)
			}
		}
	}
}

// A Bitcoin row's values are what it read (SPEC.md section 10.4, "What a
// row read"), exactly: policy, required, the one source's outcome with its
// reads in replay form, the heights as numbers, and the candidate; and its
// message names the block, its transactions and the median time past
// whole. A not_yet_indexed row's values note the height and the tip.
// Mutants killed: the candidate left out of the row's values; the
// not_yet_indexed notes without the tip.
func TestChain_ABitcoinRowsValues(t *testing.T) {
	t0 := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	useClock(t, t0, t0.Add(time.Second), t0.Add(2*time.Second))
	item := positiveProofs(t)["item"]
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	stand.answerBitcoin(soundItemBitcoin(t, item, 233, ruleMedian), SourceRead{}, SourceRead{}, SourceRead{}, "a")
	run := openItemRunWith(t, item, deriveAttribution, bitcoinSet(SourcePolicy{Kind: SourcePolicyFirst}, "regtest", "a"), stand)
	Chain(run)
	row := rowAt(t, run, topBitcoin)
	got, err := encodeValues(row.Values)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"candidate":{"basis":"bitcoin_median_time_read","instant":"2026-09-26T01:57:00Z"},` +
		`"outcomes":[{"cached":false,"candidate":{"basis":"bitcoin_median_time_read","instant":"2026-09-26T01:57:00Z"},"fetched_at":"2026-10-20T09:00:02.000Z",` +
		`"reads":[{"answer":{"outcome":"found","value":{"height":233}},"cached":false,"fetched_at":"2026-10-20T09:00:00.000Z","read":{"kind":"bitcoin_tip","network":"regtest"}},` +
		`{"answer":{"outcome":"found","value":{"hash":"` + ruleHeaderHash + `","median_time_past":1790387220,"transaction_count":5}},"cached":false,"fetched_at":"2026-10-20T09:00:01.000Z",` +
		`"read":{"height":222,"kind":"bitcoin_block_at_height","network":"regtest"}},` +
		`{"answer":{"outcome":"found","value":{"hash":"` + strings.Repeat("2", 64) + `","median_time_past":1790387820,"transaction_count":1}},"cached":false,"fetched_at":"2026-10-20T09:00:02.000Z",` +
		`"read":{"height":232,"kind":"bitcoin_block_at_height","network":"regtest"}}],"source":"https://a.example","status":"pass"}],"policy":"first","required":1}`
	if got != want {
		t.Fatalf("values\n%s\nwant\n%s", got, want)
	}
	wantMessage := "Read from the pinned Bitcoin regtest network: the best-chain block at height 222 is the block " + ruleHeaderHash +
		", with 5 transactions, that this proof's txoutproof gives, and the median time past of the best-chain block at height 232 is 2026-09-26T01:57:00Z."
	if row.Message != wantMessage {
		t.Fatalf("message %q", row.Message)
	}
	early := &chainStandIn{answers: map[string]SourceRead{}}
	early.answerBitcoin(soundItemBitcoin(t, item, 200, ruleMedian), SourceRead{}, SourceRead{}, SourceRead{}, "a")
	run = openItemRunWith(t, item, deriveAttribution, bitcoinSet(SourcePolicy{Kind: SourcePolicyFirst}, "regtest", "a"), early)
	Chain(run)
	notes, _ := encodeValues(rowAt(t, run, topBitcoin).Values["notes"])
	if notes != `[{"height":222,"source":"https://a.example","tip":200}]` {
		t.Fatalf("notes %s", notes)
	}
}

// The message of each Bitcoin row says what the reads that settled it
// found (SPEC.md section 10.5, "Rows"), for display only: the tip below h
// and below h + 10; the block at h not held, and the block at h + 10 not
// held after a block at h that matches; unreachable at the tip, at h and
// at h + 10, and at an identity; mixed sources in neutral words; and each
// source of several named as each. Mutants killed: the not_held_by_source
// message chosen from its code alone (the block at h + 10 not held says
// the block at h is not held); the source_unreachable message chosen from
// its code alone (each unreachable read says the same); asWhole naming
// one source for several (the sources all below h say "the source");
// bitcoinSettled with no mixed form (a tip unreachable beside a block
// unreachable names one read).
func TestChain_ABitcoinRowsMessageFromItsReads(t *testing.T) {
	item := positiveProofs(t)["item"]
	sound := soundItemBitcoin(t, item, 233, ruleMedian)
	network := "the pinned Bitcoin regtest network"
	first, all := SourcePolicy{Kind: SourcePolicyFirst}, SourcePolicy{Kind: SourcePolicyAll}
	type answers struct {
		name              string
		tip, block, final SourceRead
	}
	for _, tc := range []struct {
		name    string
		policy  SourcePolicy
		answers []answers
		want    string
	}{
		{"the tip below h", first, []answers{{name: "a", tip: tip(221)}},
			"Not confirmed: the tip of the source of " + network + " is below height 222, which it does not hold yet."},
		{"every tip below h", all, []answers{{name: "a", tip: tip(221)}, {name: "b", tip: tip(200)}},
			"Not confirmed: the tip of each source of " + network + " is below height 222, which it does not hold yet."},
		{"the tip below h + 10", first, []answers{{name: "a", tip: tip(231)}},
			"Not confirmed: the tip of the source of " + network + " is below height 232, 10 blocks above height 222, so the block at height 222 is not final yet."},
		{"the block at h not held", first, []answers{{name: "a", block: notFound}},
			"Not confirmed: the source of " + network + " holds no best-chain block at height 222, though its tip is past it. Absence is no contradiction."},
		{"the block at h + 10 not held", first, []answers{{name: "a", final: notFound}},
			"Not confirmed: the source of " + network + " holds the best-chain block at height 222, which matches this proof, but no best-chain block at height 232. Absence is no contradiction."},
		{"the tip not found", first, []answers{{name: "a", tip: notFound}},
			"Not confirmed: the source of " + network + " answered that it holds no tip. Absence is no contradiction."},
		{"not held, mixed", all, []answers{{name: "a", block: notFound}, {name: "b", final: notFound}},
			"Not confirmed: the sources of " + network + " do not all hold what this row reads. Absence is no contradiction."},
		{"the tip unreachable", first, []answers{{name: "a", tip: unreachableRead}},
			"Not confirmed: no source of " + network + " gave a usable answer about its tip."},
		{"the block at h unreachable", first, []answers{{name: "a", block: unreachableRead}},
			"Not confirmed: no source of " + network + " gave a usable answer about the best-chain block at height 222."},
		{"the block at h + 10 unreachable", first, []answers{{name: "a", final: unreachableRead}},
			"Not confirmed: the best-chain block at height 222 matches this proof, but no source of " + network + " gave a usable answer about the best-chain block at height 232."},
		{"the identity unreachable", first, []answers{{name: "down"}},
			"Not confirmed: no source of " + network + " had its identity confirmed, so none was read."},
		{"unreachable, mixed", all, []answers{{name: "a", tip: unreachableRead}, {name: "b", block: unreachableRead}},
			"Not confirmed: no source of " + network + " gave a usable answer to every read this row makes."},
		{"a mismatch", first, []answers{{name: "a", block: soundBlock(map[string]any{"transaction_count": strictjson.Number(4)})}},
			"Contradicted by " + network + ": the best-chain block at height 222 is not the block " + ruleHeaderHash + ", with 5 transactions, that this proof's txoutproof gives."},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		var names []string
		for _, a := range tc.answers {
			names = append(names, a.name)
			stand.answerBitcoin(sound, a.tip, a.block, a.final, a.name)
		}
		run := openItemRunWith(t, item, deriveAttribution, bitcoinSet(tc.policy, "regtest", names...), stand)
		Chain(run)
		if got := rowAt(t, run, topBitcoin).Message; got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}

// A genesis.bitcoin's row (SPEC.md section 10.4, "genesis.bitcoin"), in
// the block proof, read from the Bitcoin commitment pin's sources: one row
// at /block/record/metadata/genesis/bitcoin, CHAIN's first in document
// order, before /commitments, passing on the hash of the best-chain block
// at its height alone and giving no candidate, and failing chain_mismatch
// when that hash is another; its message names the block. Mutants killed:
// chainEntries without genesis.bitcoin (no row); the rows unsorted (the
// genesis row after the commitments'); the hash at the genesis height not
// compared (another hash passes).
func TestChain_AGenesisRow(t *testing.T) {
	block := positiveProofs(t)["block"]
	genesis, _ := valueAt(block, append(Pointer{Name("block")}, genesisBitcoinAt...)).(map[string]any)
	gHash, _ := genesis["header_hash"].(string)
	gHeight, _ := intField(genesis, "height")
	for _, tc := range []struct {
		name, hash, want string
	}{
		{"its hash", gHash, "pass"},
		{"another hash", strings.Repeat("cd", 32), "fail"},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		url := "https://a.example"
		stand.answers[standInKey(url, tipRead("regtest"))] = tip(300)
		stand.answers[standInKey(url, blockRead("regtest", gHeight))] = found(map[string]any{"hash": tc.hash})
		for _, part := range []Pointer{topBitcoin, keyBitcoin} {
			entry, _ := valueAt(block, part).(map[string]any)
			h, _ := intField(entry, "height")
			raw, _ := hex.DecodeString(entry["txoutproof"].(string))
			total := int64(raw[80]) | int64(raw[81])<<8 | int64(raw[82])<<16 | int64(raw[83])<<24
			stand.answers[standInKey(url, blockRead("regtest", h))] = found(map[string]any{"hash": entry["header_hash"], "transaction_count": strictjson.Number(float64(total))})
			stand.answers[standInKey(url, blockRead("regtest", h+10))] = finalBlock(strictjson.Number(ruleMedian))
		}
		run := openItemRunWith(t, block, deriveAttribution, bitcoinSet(SourcePolicy{Kind: SourcePolicyFirst}, "regtest", "a"), stand)
		before := len(run.Rows)
		Chain(run)
		first := run.Rows[before]
		gPart := append(Pointer{Name("block")}, genesisBitcoinAt...).Encode()
		if first.Part == nil || *first.Part != gPart || string(first.Status) != tc.want {
			t.Errorf("%s: CHAIN's first row %s", tc.name, rowSummary([]Row{first}))
		}
		for _, c := range run.Candidates {
			if c.Part == gPart {
				t.Errorf("%s: a candidate at %s", tc.name, c.Part)
			}
		}
		if tc.want == "pass" && first.Message != "Read from the pinned Bitcoin regtest network: the best-chain block at height 200 is the block "+gHash+" this genesis names." {
			t.Errorf("%s: message %q", tc.name, first.Message)
		}
		if got := rowAt(t, run, topBitcoin).Status; got != StatusPass {
			t.Errorf("%s: the top-level Bitcoin row %s", tc.name, got)
		}
	}
}

// answerBlockProof sets the stand-in's answers of the source at url, on
// regtest, for every Bitcoin row of the block proof: its tip at 300, the
// block at the genesis height with the genesis hash, and for each Bitcoin
// commitment entry the block at h with the header hash and total_txs its
// txoutproof gives and the block at h + 10 with the median time past
// ruleMedian.
func answerBlockProof(t *testing.T, block map[string]any, stand *chainStandIn, url string) {
	t.Helper()
	genesis, _ := valueAt(block, append(Pointer{Name("block")}, genesisBitcoinAt...)).(map[string]any)
	gHeight, _ := intField(genesis, "height")
	stand.answers[standInKey(url, tipRead("regtest"))] = tip(300)
	stand.answers[standInKey(url, blockRead("regtest", gHeight))] = found(map[string]any{"hash": genesis["header_hash"]})
	for _, part := range []Pointer{topBitcoin, keyBitcoin} {
		entry, _ := valueAt(block, part).(map[string]any)
		h, _ := intField(entry, "height")
		raw, err := hex.DecodeString(entry["txoutproof"].(string))
		if err != nil || len(raw) < 84 {
			t.Fatalf("the txoutproof at %s: %v", part.Encode(), err)
		}
		total := int64(raw[80]) | int64(raw[81])<<8 | int64(raw[82])<<16 | int64(raw[83])<<24
		stand.answers[standInKey(url, blockRead("regtest", h))] = found(map[string]any{"hash": entry["header_hash"], "transaction_count": strictjson.Number(float64(total))})
		stand.answers[standInKey(url, blockRead("regtest", h+10))] = finalBlock(strictjson.Number(ruleMedian))
	}
}

// bitcoinRowsAndCandidates writes the run's CHAIN rows of Bitcoin
// entries and genesis.bitcoin objects, then its Bitcoin candidates.
func bitcoinRowsAndCandidates(run *Run) string {
	var rows []Row
	for _, r := range run.Rows {
		if r.Step == "CHAIN" && r.Part != nil && strings.HasSuffix(*r.Part, "/bitcoin") {
			rows = append(rows, r)
		}
	}
	var candidates []RowCandidate
	for _, c := range run.Candidates {
		if strings.HasSuffix(c.Part, "/bitcoin") {
			candidates = append(candidates, c)
		}
	}
	return rowSummary(rows) + "; " + candidateText(&Run{Candidates: candidates})
}

// The block proof's Bitcoin rows, genesis.bitcoin's among them, read the
// run's Bitcoin commitment pin's sources and never its entropy pin's
// (SPEC.md section 10.4, "Sources" and "genesis.bitcoin"): with the
// Bitcoin entropy pin naming another network, testnet, whose one source,
// ent, has no answers, ent is asked nothing, and every Bitcoin row keeps
// the status and the candidate its commitment pin's source gives it, the
// genesis row and both entries' passing and the top-level entry's median
// time past the run's one Bitcoin candidate. Mutants killed: the genesis
// rows read from the entropy pin's sources (ent is asked its tip, and the
// genesis row skips); the genesis rows read from them whenever the run
// has a set for that pin (the same); every Bitcoin row read from them
// whenever the run has such a set (ent is asked its tip three times, and
// every Bitcoin row skips, with no candidate).
func TestChain_ABitcoinRowReadsTheCommitmentPin(t *testing.T) {
	block := positiveProofs(t)["block"]
	first := SourcePolicy{Kind: SourcePolicyFirst}
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	answerBlockProof(t, block, stand, "https://a.example")
	run := openItemRunWith(t, block, deriveAttribution, bitcoinSet(first, "regtest", "a"), stand)
	run.Config.Networks.Entropy.Bitcoin = "testnet"
	run.SourceSets = append(run.SourceSets, bitcoinSet(first, "testnet", "ent"))
	Chain(run)
	if got := askedFor(stand, "ent"); got != nil {
		t.Errorf("the entropy pin's source was asked %v", got)
	}
	want := "[null /block/record/metadata/genesis/bitcoin pass CHAIN] [null /commitments/bitcoin pass CHAIN] [null /signing_key_event/commitments/bitcoin pass CHAIN]; " +
		"CHAIN /commitments/bitcoin bitcoin_median_time_read 2026-09-26T01:57:00Z [a]"
	if got := bitcoinRowsAndCandidates(run); got != want {
		t.Errorf("the Bitcoin rows and candidates\n%s\nwant\n%s", got, want)
	}
}

// CHAIN holds the block at h to the header hash it recomputes from the
// txoutproof, never to the header_hash member P carries (SPEC.md sections
// 10.3, "Recomputed values only", and 10.6, rule 4): with another HEX64
// planted as /commitments/bitcoin/header_hash after DERIVE, which in a
// whole run DERIVE refuses first, the top-level row passes against a
// source answering the recomputed hash, and fails chain_mismatch against
// one answering the planted hash. Mutant killed: bitcoinEntryOf computing
// the recomputed hash and then taking the carried member in its place
// (the first row fails and the second passes).
func TestChain_ABitcoinRowComparesTheRecomputedHash(t *testing.T) {
	item := positiveProofs(t)["item"]
	planted := strings.Repeat("ab", 32)
	sound := soundItemBitcoin(t, item, 233, ruleMedian)
	for _, tc := range []struct {
		name  string
		block SourceRead
		want  string
	}{
		{"the recomputed hash", SourceRead{}, "[null /commitments/bitcoin pass CHAIN]"},
		{"the planted hash", found(map[string]any{"hash": planted, "transaction_count": sound.block.Value["transaction_count"]}), "[chain_mismatch /commitments/bitcoin fail CHAIN]"},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		stand.answerBitcoin(sound, SourceRead{}, tc.block, SourceRead{}, "a")
		run := openItemRunWith(t, item, deriveAttribution, bitcoinSet(SourcePolicy{Kind: SourcePolicyFirst}, "regtest", "a"), stand)
		run.Proof = put(item, under(topBitcoin, "header_hash"), planted)
		Chain(run)
		if got := rowSummary([]Row{rowAt(t, run, topBitcoin)}); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
}

// encodeValues is v's JCS as a string.
func encodeValues(v any) (string, error) {
	b, err := jcs.Encode(v)
	return string(b), err
}

// DefaultChainReader reads a Bitcoin source through
// internal/verifier/source's readers, the source's credentials sent to a
// Bitcoin Core node and to nothing else, and reaching no answer: an
// Esplora server's tip and block at a height, on loopback, with no
// Authorization; a Bitcoin Core node's, on loopback, each call with the
// source's credentials as HTTP Basic, and a node that echoes them into
// its answers, whose reasons and values hold none of them; and, for each
// compiled-in Bitcoin default, a tip read refused in process once by this
// package's TestMain, unreachable (the control of the Bitcoin reads).
// Mutants killed: DefaultChainReader passing no credentials (the node sees
// no Authorization); source.Read giving an Esplora source Bitcoin Core's
// reads (Esplora is sent the credentials).
func TestChain_TheDefaultReaderOfBitcoin(t *testing.T) {
	const user, secret = "rpcuser-planted-5d2a", "planted-secret-5d2a"
	creds := source.NewCredentials(user, secret)
	basic := "Basic " + base64Std(user+":"+secret)
	var esploraAuth, coreAuth []string
	esplora := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		esploraAuth = append(esploraAuth, r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/blocks/tip/height":
			_, _ = w.Write([]byte("233"))
		case "/block-height/222":
			_, _ = w.Write([]byte(ruleHeaderHash))
		case "/block/" + ruleHeaderHash:
			_, _ = w.Write([]byte(`{"tx_count":5,"mediantime":1790387220,"height":222}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer esplora.Close()
	echo := false
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		coreAuth = append(coreAuth, auth)
		body, _ := io.ReadAll(r.Body)
		switch {
		case echo:
			_, _ = fmt.Fprintf(w, `{"result":{"mediantime":%q,"nTx":%q},"error":{"code":-1,"message":%q},"id":"truestamp-verify"}`, auth, auth, auth)
		case strings.Contains(string(body), `"getblockcount"`):
			_, _ = w.Write([]byte(`{"result":233,"error":null,"id":"truestamp-verify"}`))
		case strings.Contains(string(body), `"getblockhash"`):
			_, _ = w.Write([]byte(`{"result":"` + ruleHeaderHash + `","error":null,"id":"truestamp-verify"}`))
		default:
			_, _ = w.Write([]byte(`{"result":{"nTx":5,"mediantime":1790387220},"error":null,"id":"truestamp-verify"}`))
		}
	}))
	defer node.Close()
	reader := DefaultChainReader()
	block := SourceRead{Outcome: ReadFound, Value: map[string]any{"hash": ruleHeaderHash, "transaction_count": strictjson.Number(5), "median_time_past": strictjson.Number(1790387220)}}
	for _, s := range []Source{
		{Chain: ChainBitcoin, Provider: source.Esplora, URL: esplora.URL, Credentials: creds, Network: "regtest"},
		{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: node.URL, Credentials: creds, Network: "regtest"},
	} {
		if got := reader(s, tipRead("regtest")); !reflect.DeepEqual(got, tip(233)) {
			t.Errorf("%s's tip: %+v", s.Provider, got)
		}
		if got := reader(s, blockRead("regtest", 222)); !reflect.DeepEqual(got, block) {
			t.Errorf("%s's block at 222: %+v", s.Provider, got)
		}
	}
	if !reflect.DeepEqual(esploraAuth, []string{"", "", ""}) || !reflect.DeepEqual(coreAuth, []string{basic, basic, basic}) {
		t.Fatalf("Esplora was sent %q, Bitcoin Core %q", esploraAuth, coreAuth)
	}
	echo = true
	core := Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: node.URL, Credentials: creds, Network: "regtest"}
	for _, read := range []map[string]any{tipRead("regtest"), blockRead("regtest", 222)} {
		got := reader(core, read)
		printed := fmt.Sprintf("%+v %v %s", got, got.Value, got.Reason)
		for _, planted := range []string{user, secret, basic} {
			if strings.Contains(printed, planted) {
				t.Errorf("%s: %q is in the answer %s", read["kind"], planted, printed)
			}
		}
	}
	for _, network := range []string{"mainnet", "testnet"} {
		s, _ := DefaultSource(ChainBitcoin, network)
		before := refusals.Load()
		got := DefaultChainReader()(s, tipRead(network))
		if got.Outcome != ReadUnreachable || refusals.Load()-before != 1 {
			t.Fatalf("%s: %+v, %d refused", s.URL, got, refusals.Load()-before)
		}
	}
}

// A Bitcoin block at a height the source does not hold is read live every
// time (SPEC.md section 10.4, "Caches"): the same bitcoin_block_at_height
// read, made twice through one DefaultChainReader and once through
// another, of an Esplora server that answers 404 to its block-height and
// of a Bitcoin Core node that answers getblockhash with its error -8 at
// status 500, is not found each time, and each server counts three
// requests. Mutants killed, each a package-level map: EsploraBlockAtHeight
// keeping each 404 by base and height (Esplora sees one request);
// CoreBlockAtHeight keeping each not-found by URL and height (Bitcoin Core
// sees one); DefaultChainReader keeping each not-found answer by URL and
// read (each sees one).
func TestChain_ABitcoinNotFoundIsReadLive(t *testing.T) {
	var esploraN, coreN atomic.Int64
	esplora := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/block-height/9000" {
			esploraN.Add(1)
		}
		http.NotFound(w, r)
	}))
	defer esplora.Close()
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreN.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"result":null,"error":{"code":-8,"message":"Block height out of range"},"id":"truestamp-verify"}`))
	}))
	defer node.Close()
	creds := source.NewCredentials("stand-in-user", "stand-in-pass")
	reader := DefaultChainReader()
	for _, s := range []Source{
		{Chain: ChainBitcoin, Provider: source.Esplora, URL: esplora.URL, Network: "regtest"},
		{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: node.URL, Credentials: creds, Network: "regtest"},
	} {
		for i, r := range []ChainReader{reader, reader, DefaultChainReader()} {
			if got := r(s, blockRead("regtest", 9000)); !reflect.DeepEqual(got, notFound) {
				t.Fatalf("%s, read %d: %+v", s.Provider, i, got)
			}
		}
	}
	if esploraN.Load() != 3 || coreN.Load() != 3 {
		t.Fatalf("Esplora saw %d requests, Bitcoin Core %d, want 3 each", esploraN.Load(), coreN.Load())
	}
}

// DefaultChainReader's deadline bounds the whole read, not each call
// alone: with each call allowed a minute and the read 300 ms, a best-chain
// block at a height whose first call answers and whose second never does
// ends in under two deadlines, for Esplora and for Bitcoin Core, found
// with the hash alone (the answer of a second call with no answer).
// Mutants killed: the reader giving source.Read a context with no deadline
// of its own (the read waits out the call's minute, cut by a 5 s guard);
// the whole read given twice its deadline (it ends after 600 ms).
func TestChain_ABitcoinReadEndsWithinItsDeadline(t *testing.T) {
	release := make(chan struct{})
	hang := func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}
	esplora := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/block-height/222" {
			_, _ = w.Write([]byte(ruleHeaderHash))
			return
		}
		hang(w, r)
	}))
	defer esplora.Close()
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), `"getblockhash"`) {
			_, _ = w.Write([]byte(`{"result":"` + ruleHeaderHash + `","error":null,"id":"truestamp-verify"}`))
			return
		}
		hang(w, r)
	}))
	defer node.Close()
	defer close(release)
	saved := newSingleRead
	newSingleRead = func(transport http.RoundTripper, _ time.Duration) *httpclient.SingleRead {
		return saved(transport, time.Minute)
	}
	defer func() { newSingleRead = saved }()
	want := found(map[string]any{"hash": ruleHeaderHash})
	for _, s := range []Source{
		{Chain: ChainBitcoin, Provider: source.Esplora, URL: esplora.URL, Network: "regtest"},
		{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: node.URL, Credentials: source.NewCredentials("stand-in-user", "stand-in-pass"), Network: "regtest"},
	} {
		done := make(chan SourceRead, 1)
		start := time.Now()
		go func() { done <- chainReaderWithin(300*time.Millisecond)(s, blockRead("regtest", 222)) }()
		select {
		case got := <-done:
			elapsed := time.Since(start)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: %+v after %v, want %+v", s.Provider, got, elapsed, want)
			}
			if elapsed >= 600*time.Millisecond {
				t.Errorf("%s: the read ended after %v, want under two deadlines (600ms)", s.Provider, elapsed)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: the read was not cut at its deadline", s.Provider)
		}
	}
}

// base64Std is the standard base64 of text.
func base64Std(text string) string { return base64.StdEncoding.EncodeToString([]byte(text)) }
