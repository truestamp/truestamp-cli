// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// SOURCES under an open gate (SPEC.md sections 10.2, step 11, 10.4,
// "Entropy objects", "Several sources", "Rows" and "What a row read", and
// 11.2), over the positive proofs, every read answered by a chainStandIn by
// each source's URL, NIST's at its compiled-in address.

// The witnesses of the item proof, and the values a sound source gives
// for them: the witness block's median time past, 2026-09-26T01:40:00Z.
var (
	witnessNIST    = Pointer{Name("witnesses"), Name("entropy_nist")}
	witnessStellar = Pointer{Name("witnesses"), Name("entropy_stellar")}
	witnessBitcoin = Pointer{Name("witnesses"), Name("entropy_bitcoin")}
)

const (
	witnessMedian = 1790386800
	otherNISTHex  = "0F209B792804BD0BB0EC385917560B263A9E9D805F8EE0423CC1345E0E81156A0C9F3F14EBA50D8C39E21C4F247807A98CF4647C39BEA5A1B29064B8ECF90F4D"
)

// entropyEntryAt is the entry the rule reads of proof's entropy object at
// at.
func entropyEntryAt(t *testing.T, proof map[string]any, at Pointer) entropyEntry {
	t.Helper()
	for _, object := range entropyObjects(proof) {
		if Compare(object.at, at) == 0 {
			e, err := entropyEntryOf(proof, object)
			if err != nil {
				t.Fatal(err)
			}
			return e
		}
	}
	t.Fatalf("no entropy object at %s", at.Encode())
	return entropyEntry{}
}

// entropyObjectRead is the read the rule makes of e's object on network,
// in replay form.
func entropyObjectRead(e entropyEntry, network string) map[string]any {
	switch e.source {
	case SourceNIST:
		return map[string]any{"kind": source.KindNISTPulse, "chainIndex": strictjson.Number(float64(e.chainIndex)), "pulseIndex": strictjson.Number(float64(e.pulseIndex))}
	case ChainStellar:
		return map[string]any{"kind": source.KindStellarLedger, "network": network, "sequence": strictjson.Number(float64(e.sequence))}
	}
	return map[string]any{"kind": source.KindBitcoinBlock, "network": network, "hash": e.hash}
}

// soundEntropy is the answer a sound source gives the read of e's object.
func soundEntropy(t *testing.T, e entropyEntry) SourceRead {
	t.Helper()
	switch e.source {
	case SourceNIST:
		stamp, _ := FormatTime(ClassNISTTIME, e.timeStamp)
		return found(map[string]any{"outputValue": e.outputValue, "timeStamp": stamp})
	case ChainStellar:
		closed, _ := FormatTime(ClassTIME, e.closedAt)
		return found(map[string]any{"closed_at": closed, "hash": e.hash})
	}
	return found(map[string]any{"height": intValue(e.height), "median_time_past": intValue(witnessMedian), "time": intValue(e.time)})
}

// answerWith is answer with its value's members replaced by change (a nil value
// removes one).
func answerWith(answer SourceRead, change map[string]any) SourceRead {
	value := map[string]any{}
	for name, v := range answer.Value {
		value[name] = v
	}
	for name, v := range change {
		if v == nil {
			delete(value, name)
		} else {
			value[name] = v
		}
	}
	return found(value)
}

// answerEntropy sets the stand-in's answer to the read of e's object on
// network, from each source of names, or from the NIST beacon for a NIST
// object: answer, or a sound source's answer when answer is the zero value.
func (s *chainStandIn) answerEntropy(t *testing.T, e entropyEntry, network string, answer SourceRead, names ...string) {
	t.Helper()
	if answer.Outcome == "" {
		answer = soundEntropy(t, e)
	}
	if e.source == SourceNIST {
		s.answers[standInKey(NISTBeaconURL, entropyObjectRead(e, ""))] = answer
		return
	}
	for _, name := range names {
		s.answers[standInKey("https://"+name+".example", entropyObjectRead(e, network))] = answer
	}
}

// entropyRun is proof's run past DERIVE, online, its gate open, its
// commitment pins Stellar testnet and Bitcoin regtest, its entropy pins
// Stellar stellar and Bitcoin bitcoin, its sources sets, every read
// answered by stand.
func entropyRun(t *testing.T, proof map[string]any, stellar, bitcoin string, sets []SourceSet, stand *chainStandIn) *Run {
	t.Helper()
	run := deriveRun(proof, deriveAttribution)
	run.Config.Online = true
	run.Config.Networks = specdata.NetworkPins{
		Commitment: specdata.ChainNetworks{Bitcoin: "regtest", Stellar: "testnet"},
		Entropy:    specdata.ChainNetworks{Bitcoin: bitcoin, Stellar: stellar},
	}
	run.Config.ReadChain = stand.reader
	run.SourceSets = sets
	Derive(run)
	if run.ExternalGate != ExternalGateOpen {
		t.Fatalf("the gate is %q, rows %s", run.ExternalGate, rowSummary(run.Rows))
	}
	return run
}

// stellarSetOn is standInSet on the Stellar network network.
func stellarSetOn(policy SourcePolicy, network string, names ...string) SourceSet {
	set := standInSet(policy, names...)
	set.Network = network
	for i := range set.Sources {
		set.Sources[i].Source.Network = network
	}
	return set
}

// soundSources answers every entropy object of proof soundly, a Stellar
// object from the source stellar of testnet and a Bitcoin one from the
// source bitcoin of regtest, and gives their sets under first.
func soundSources(t *testing.T, proof map[string]any, stand *chainStandIn) []SourceSet {
	t.Helper()
	for _, object := range entropyObjects(proof) {
		e := entropyEntryAt(t, proof, object.at)
		network := map[string]string{ChainStellar: "testnet", ChainBitcoin: "regtest"}[e.source]
		stand.answerEntropy(t, e, network, SourceRead{}, map[string]string{ChainStellar: "stellar", ChainBitcoin: "bitcoin"}[e.source])
	}
	first := SourcePolicy{Kind: SourcePolicyFirst}
	return []SourceSet{stellarSetOn(first, "testnet", "stellar"), bitcoinSet(first, "regtest", "bitcoin")}
}

// entropyOutcomeText writes an outcome as "<status>[ <code>][ <notes as
// JCS>][ <basis> <instant>]".
func entropyOutcomeText(t *testing.T, o SourceOutcome) string {
	t.Helper()
	text := string(o.Status)
	if o.Code != "" {
		text += " " + string(o.Code)
	}
	if o.Notes != nil {
		notes, err := jcs.Encode(o.Notes)
		if err != nil {
			t.Fatal(err)
		}
		text += " " + string(notes)
	}
	if c := o.Candidate; c != nil {
		instant, _ := FormatTime(c.Class, c.Instant)
		text += " " + c.Basis + " " + instant
	}
	return text
}

// entropyOutcome makes e's rule against one confirmed source of network,
// https://a.example, or the NIST beacon for a NIST object, whose answers to
// the read of the object and to the root are object and root (none when
// the zero value), with the clock at at, and gives its outcome and the
// kinds of the reads it made.
func entropyOutcome(t *testing.T, e entropyEntry, network string, object, root SourceRead, at time.Time) (SourceOutcome, []string) {
	t.Helper()
	useClock(t, at)
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	record := nistSet().Sources[0]
	switch e.source {
	case ChainStellar:
		record = stellarSetOn(SourcePolicy{Kind: SourcePolicyFirst}, network, "a").Sources[0]
	case ChainBitcoin:
		record = bitcoinSet(SourcePolicy{Kind: SourcePolicyFirst}, network, "a").Sources[0]
	}
	if object.Outcome != "" {
		stand.answers[standInKey(record.Source.URL, entropyObjectRead(e, network))] = object
	}
	if root.Outcome != "" {
		stand.answers[standInKey(record.Source.URL, rootRead(network))] = root
	}
	run := &Run{Config: Config{ReadChain: stand.reader}}
	if e.source == SourceNIST {
		network = ""
	}
	got := entropyRule(run, network, e)(record)
	var kinds []string
	for _, r := range got.Reads {
		kinds = append(kinds, r.Read["kind"].(string))
	}
	return got, kinds
}

// Every outcome of each entropy rule from the answers of one source
// (SPEC.md section 10.4, "Entropy objects" and "Answers"), the Stellar and
// Bitcoin rules on two networks each alike, over the item proof's three
// witnesses: NIST's pulse found and matching passes with the timeStamp read
// as its candidate at a millisecond's precision, its outputValue or its
// timeStamp differing fails entropy_mismatch naming the field, not found
// is not_held_by_source naming the pulse, and unreachable,
// source_unreachable; Stellar's ledger found and matching passes with its
// closed_at, its hash or closed_at differing fails, and not found reads
// the root: a sequence above latest - STELLAR_INDEX_MARGIN not_yet_indexed,
// at it or below not_held_by_source, each naming latest and the sequence,
// whatever the root's elder, a root that lacks elder judged on latest, a
// root without latest or unreachable source_unreachable, and a root not
// found not_held_by_source; Bitcoin's block by its hash found passes with
// its median time past, with no read of the tip or of a best-chain block,
// so a block off the best chain is judged alike, its height or time
// differing fails, and not found is not_held_by_source naming the hash.
// An answer is judged on the fields the rule reads: one that lacks a field
// or gives it in another form (a lowercase outputValue, a timeStamp with no
// milliseconds, an uppercase hash, a median past 9999-12-31) is
// source_unreachable naming the field, one that gives more passes, and a
// contradiction fails even beside a field the answer lacks. A candidate's
// fetched_at is its read's. Mutants killed: the NIST rule giving
// source_unreachable for not found; outputValue left uncompared; the
// timeStamp compared at seconds' precision (a millisecond later passes);
// nistHexField taking lowercase; the unusable fields judged before the
// contradictions (each contradiction beside a missing field skips); the
// Stellar not-found test >= in place of > (the margin case is
// not_yet_indexed); the root required to hold elder (the elder-less root
// is source_unreachable); the root read before the ledger; the Bitcoin
// rule passing without a median time past; time left uncompared; the
// TIME bound on the median dropped; the candidate's fetched_at left unset.
func TestEntropy_EveryOutcomeOfEachRule(t *testing.T) {
	item := positiveProofs(t)["item"]
	nist, stellar, bitcoin := entropyEntryAt(t, item, witnessNIST), entropyEntryAt(t, item, witnessStellar), entropyEntryAt(t, item, witnessBitcoin)
	at := time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC)
	nistSound, stellarSound, bitcoinSound := soundEntropy(t, nist), soundEntropy(t, stellar), soundEntropy(t, bitcoin)
	seq := stellar.sequence
	none := SourceRead{}
	for _, tc := range []struct {
		name         string
		e            entropyEntry
		networks     []string
		object, root SourceRead
		want         string
		asked        []string
	}{
		{"a pulse that matches", nist, []string{""}, nistSound, none, "pass nist_pulse_time_read 2026-09-26T01:48:00.000Z", []string{"nist_pulse"}},
		{"a pulse with more fields", nist, []string{""}, answerWith(nistSound, map[string]any{"version": "2.0", "signatureValue": "AB"}), none,
			"pass nist_pulse_time_read 2026-09-26T01:48:00.000Z", []string{"nist_pulse"}},
		{"another outputValue", nist, []string{""}, answerWith(nistSound, map[string]any{"outputValue": otherNISTHex}), none,
			`fail entropy_mismatch {"differs":["outputValue"]}`, []string{"nist_pulse"}},
		{"a millisecond later", nist, []string{""}, answerWith(nistSound, map[string]any{"timeStamp": "2026-09-26T01:48:00.001Z"}), none,
			`fail entropy_mismatch {"differs":["timeStamp"]}`, []string{"nist_pulse"}},
		{"a pulse not held", nist, []string{""}, notFound, none, `skip not_held_by_source {"chainIndex":2,"pulseIndex":1200018}`, []string{"nist_pulse"}},
		{"the beacon unreachable", nist, []string{""}, unreachableRead, none, "skip source_unreachable", []string{"nist_pulse"}},
		{"no outputValue", nist, []string{""}, answerWith(nistSound, map[string]any{"outputValue": nil}), none,
			`skip source_unreachable {"unusable":["outputValue"]}`, []string{"nist_pulse"}},
		{"a lowercase outputValue", nist, []string{""}, answerWith(nistSound, map[string]any{"outputValue": strings.ToLower(nist.outputValue)}), none,
			`skip source_unreachable {"unusable":["outputValue"]}`, []string{"nist_pulse"}},
		{"a timeStamp with no milliseconds", nist, []string{""}, answerWith(nistSound, map[string]any{"timeStamp": "2026-09-26T01:48:00Z"}), none,
			`skip source_unreachable {"unusable":["timeStamp"]}`, []string{"nist_pulse"}},
		{"another outputValue and no timeStamp", nist, []string{""}, answerWith(nistSound, map[string]any{"outputValue": otherNISTHex, "timeStamp": nil}), none,
			`fail entropy_mismatch {"differs":["outputValue"]}`, []string{"nist_pulse"}},

		{"a ledger that matches", stellar, []string{"testnet", "public"}, stellarSound, none, "pass stellar_ledger_close_read 2026-09-26T01:48:30Z", []string{"stellar_ledger"}},
		{"a ledger with more fields", stellar, []string{"testnet", "public"}, answerWith(stellarSound, map[string]any{"sequence": intValue(seq)}), none,
			"pass stellar_ledger_close_read 2026-09-26T01:48:30Z", []string{"stellar_ledger"}},
		{"another ledger hash", stellar, []string{"testnet", "public"}, answerWith(stellarSound, map[string]any{"hash": strings.Repeat("ab", 32)}), none,
			`fail entropy_mismatch {"differs":["hash"]}`, []string{"stellar_ledger"}},
		{"a second later", stellar, []string{"testnet", "public"}, answerWith(stellarSound, map[string]any{"closed_at": "2026-09-26T01:48:31Z"}), none,
			`fail entropy_mismatch {"differs":["closed_at"]}`, []string{"stellar_ledger"}},
		{"no closed_at", stellar, []string{"testnet", "public"}, answerWith(stellarSound, map[string]any{"closed_at": nil}), none,
			`skip source_unreachable {"unusable":["closed_at"]}`, []string{"stellar_ledger"}},
		{"an uppercase hash", stellar, []string{"testnet", "public"}, answerWith(stellarSound, map[string]any{"hash": strings.ToUpper(stellar.hash)}), none,
			`skip source_unreachable {"unusable":["hash"]}`, []string{"stellar_ledger"}},
		{"another hash and no closed_at", stellar, []string{"testnet", "public"}, answerWith(stellarSound, map[string]any{"hash": strings.Repeat("ab", 32), "closed_at": nil}), none,
			`fail entropy_mismatch {"differs":["hash"]}`, []string{"stellar_ledger"}},
		{"not found, within the margin", stellar, []string{"testnet", "public"}, notFound, root(seq+59, 1),
			fmt.Sprintf(`skip not_yet_indexed {"latest":%d,"sequence":%d}`, seq+59, seq), []string{"stellar_ledger", "stellar_root"}},
		{"not found, beyond the latest", stellar, []string{"testnet", "public"}, notFound, root(seq-5, 1),
			fmt.Sprintf(`skip not_yet_indexed {"latest":%d,"sequence":%d}`, seq-5, seq), []string{"stellar_ledger", "stellar_root"}},
		{"not found, at the margin", stellar, []string{"testnet", "public"}, notFound, root(seq+60, 1),
			fmt.Sprintf(`skip not_held_by_source {"latest":%d,"sequence":%d}`, seq+60, seq), []string{"stellar_ledger", "stellar_root"}},
		{"not found, the elder above it", stellar, []string{"testnet", "public"}, notFound, root(seq+500, seq+1),
			fmt.Sprintf(`skip not_held_by_source {"latest":%d,"sequence":%d}`, seq+500, seq), []string{"stellar_ledger", "stellar_root"}},
		{"not found, a root with no elder", stellar, []string{"testnet", "public"}, notFound, found(map[string]any{"latest": intValue(seq + 500)}),
			fmt.Sprintf(`skip not_held_by_source {"latest":%d,"sequence":%d}`, seq+500, seq), []string{"stellar_ledger", "stellar_root"}},
		{"not found, a root with no latest", stellar, []string{"testnet", "public"}, notFound, found(map[string]any{"elder": intValue(1)}),
			`skip source_unreachable {"unusable":["latest"]}`, []string{"stellar_ledger", "stellar_root"}},
		{"not found, the root unreachable", stellar, []string{"testnet", "public"}, notFound, unreachableRead, "skip source_unreachable", []string{"stellar_ledger", "stellar_root"}},
		{"not found, the root not found", stellar, []string{"testnet", "public"}, notFound, notFound,
			fmt.Sprintf(`skip not_held_by_source {"sequence":%d}`, seq), []string{"stellar_ledger", "stellar_root"}},
		{"the ledger unreachable", stellar, []string{"testnet", "public"}, unreachableRead, none, "skip source_unreachable", []string{"stellar_ledger"}},

		{"a block that matches", bitcoin, []string{"regtest", "mainnet"}, bitcoinSound, none, "pass bitcoin_median_time_read 2026-09-26T01:40:00Z", []string{"bitcoin_block"}},
		{"a block off the best chain, with more fields", bitcoin, []string{"regtest", "mainnet"}, answerWith(bitcoinSound, map[string]any{"confirmations": strictjson.Number(-1)}), none,
			"pass bitcoin_median_time_read 2026-09-26T01:40:00Z", []string{"bitcoin_block"}},
		{"another height", bitcoin, []string{"regtest", "mainnet"}, answerWith(bitcoinSound, map[string]any{"height": intValue(216)}), none,
			`fail entropy_mismatch {"differs":["height"]}`, []string{"bitcoin_block"}},
		{"another time", bitcoin, []string{"regtest", "mainnet"}, answerWith(bitcoinSound, map[string]any{"time": intValue(bitcoin.time + 1)}), none,
			`fail entropy_mismatch {"differs":["time"]}`, []string{"bitcoin_block"}},
		{"another height and time", bitcoin, []string{"regtest", "mainnet"}, answerWith(bitcoinSound, map[string]any{"height": intValue(1), "time": intValue(1)}), none,
			`fail entropy_mismatch {"differs":["height","time"]}`, []string{"bitcoin_block"}},
		{"no median time past", bitcoin, []string{"regtest", "mainnet"}, answerWith(bitcoinSound, map[string]any{"median_time_past": nil}), none,
			`skip source_unreachable {"unusable":["median_time_past"]}`, []string{"bitcoin_block"}},
		{"a median past 9999", bitcoin, []string{"regtest", "mainnet"}, answerWith(bitcoinSound, map[string]any{"median_time_past": intValue(253402300800)}), none,
			`skip source_unreachable {"unusable":["median_time_past"]}`, []string{"bitcoin_block"}},
		{"no time", bitcoin, []string{"regtest", "mainnet"}, answerWith(bitcoinSound, map[string]any{"time": nil}), none,
			`skip source_unreachable {"unusable":["time"]}`, []string{"bitcoin_block"}},
		{"another height and no median", bitcoin, []string{"regtest", "mainnet"}, answerWith(bitcoinSound, map[string]any{"height": intValue(216), "median_time_past": nil}), none,
			`fail entropy_mismatch {"differs":["height"]}`, []string{"bitcoin_block"}},
		{"a block not held", bitcoin, []string{"regtest", "mainnet"}, notFound, none, `skip not_held_by_source {"hash":"` + bitcoin.hash + `"}`, []string{"bitcoin_block"}},
		{"the block unreachable", bitcoin, []string{"regtest", "mainnet"}, unreachableRead, none, "skip source_unreachable", []string{"bitcoin_block"}},
	} {
		for _, network := range tc.networks {
			got, asked := entropyOutcome(t, tc.e, network, tc.object, tc.root, at)
			if text := entropyOutcomeText(t, got); text != tc.want || !slices.Equal(asked, tc.asked) {
				t.Errorf("%s, %q: %s after %v\nwant %s after %v", tc.name, network, text, asked, tc.want, tc.asked)
			}
			if got.Candidate != nil && !got.Candidate.FetchedAt.Equal(at) {
				t.Errorf("%s, %q: the candidate's fetched_at %v", tc.name, network, got.Candidate.FetchedAt)
			}
		}
	}
}

// sourcesRows writes the run's SOURCES rows.
func sourcesRows(run *Run) string { return stepRows(run, "SOURCES") }

// Every proof type's SOURCES rows under an open gate: one for each entropy
// object, in document order, each read once from its source and passing,
// and each passing row's candidate kept on the run in row order, with its
// basis, instant and the one source it rests on (SPEC.md sections 10.4 and
// 11.2): the item's three witnesses, the beacon's three observations and
// each entropy type's subject; a block proof, which carries none, gives no
// row and reads nothing. Mutants killed: entropyHashPart naming the
// record's entropy_hash for an item (no DERIVE row is there, so no
// witness is read, and each row is skip prior_failure); the candidates not
// kept; a SOURCES candidate kept with the CHAIN step.
func TestEntropy_TheRowsOfEachType(t *testing.T) {
	proofs := positiveProofs(t)
	for _, c := range []struct {
		proof, rows, candidates string
	}{
		{"item",
			"[null /witnesses/entropy_bitcoin pass SOURCES] [null /witnesses/entropy_nist pass SOURCES] [null /witnesses/entropy_stellar pass SOURCES]",
			"SOURCES /witnesses/entropy_bitcoin bitcoin_median_time_read 2026-09-26T01:40:00Z [bitcoin]; " +
				"SOURCES /witnesses/entropy_nist nist_pulse_time_read 2026-09-26T01:48:00.000Z [beacon.nist.gov/beacon/2.0]; " +
				"SOURCES /witnesses/entropy_stellar stellar_ledger_close_read 2026-09-26T01:48:30Z [stellar]"},
		{"beacon",
			"[null /observations/entropy_bitcoin/entropy pass SOURCES] [null /observations/entropy_nist/entropy pass SOURCES] [null /observations/entropy_stellar/entropy pass SOURCES]",
			"SOURCES /observations/entropy_bitcoin/entropy bitcoin_median_time_read 2026-09-26T01:40:00Z [bitcoin]; " +
				"SOURCES /observations/entropy_nist/entropy nist_pulse_time_read 2026-09-26T01:45:00.000Z [beacon.nist.gov/beacon/2.0]; " +
				"SOURCES /observations/entropy_stellar/entropy stellar_ledger_close_read 2026-09-26T01:45:15Z [stellar]"},
		{"entropy_nist", "[null /subject/entropy pass SOURCES]", "SOURCES /subject/entropy nist_pulse_time_read 2026-09-26T01:48:00.000Z [beacon.nist.gov/beacon/2.0]"},
		{"entropy_stellar", "[null /subject/entropy pass SOURCES]", "SOURCES /subject/entropy stellar_ledger_close_read 2026-09-26T01:48:30Z [stellar]"},
		{"entropy_bitcoin", "[null /subject/entropy pass SOURCES]", "SOURCES /subject/entropy bitcoin_median_time_read 2026-09-26T01:40:00Z [bitcoin]"},
		{"block", "", ""},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		sets := soundSources(t, proofs[c.proof], stand)
		run := entropyRun(t, proofs[c.proof], "testnet", "regtest", sets, stand)
		Sources(run)
		if got := sourcesRows(run); got != c.rows {
			t.Errorf("%s: rows %s\nwant %s", c.proof, got, c.rows)
		}
		if got := candidateText(run); got != c.candidates {
			t.Errorf("%s: candidates %s\nwant %s", c.proof, got, c.candidates)
		}
		if len(stand.asked) != len(entropyObjects(proofs[c.proof])) {
			t.Errorf("%s: reads %v", c.proof, stand.asked)
		}
	}
}

// An entropy object whose entropy_hash did not match at DERIVE is never
// read, and its row is the one SPEC.md gives it, skip prior_failure (a row
// it depends on failed): in a whole run the failure closes the external
// gate, so every SOURCES row is skip prior_failure and nothing is read;
// and with the gate held open past that failure, the unmatched object
// alone is skip prior_failure, unread, and the others are read and pass,
// whichever row compared its entropy_hash: LINK_WITNESS for an item's NIST
// witness, and the entropy_hash check for a beacon's NIST observation and
// an entropy proof's subject, whose observation_hash rows pass beside it.
// Mutants killed: matchedAtDerive giving true for every object (the
// unmatched object is read, and fails entropy_mismatch); entropyHashPart
// naming the observation's observation_hash for a beacon, or the
// subject's for an entropy type (the unmatched object is read).
func TestEntropy_AnUnmatchedObjectIsNeverRead(t *testing.T) {
	proofs := positiveProofs(t)
	for _, c := range []struct {
		proof, failed string
		object        Pointer
		rows          string
		asked         []string
	}{
		{"item", "/subject/record/witness_hashes/entropy_nist", witnessNIST,
			"[null /witnesses/entropy_bitcoin pass SOURCES] [prior_failure /witnesses/entropy_nist skip SOURCES] [null /witnesses/entropy_stellar pass SOURCES]",
			[]string{"bitcoin bitcoin_block", "stellar stellar_ledger"}},
		{"beacon", "/observations/entropy_nist/record/entropy_hash", Pointer{Name("observations"), Name("entropy_nist"), Name("entropy")},
			"[null /observations/entropy_bitcoin/entropy pass SOURCES] [prior_failure /observations/entropy_nist/entropy skip SOURCES] [null /observations/entropy_stellar/entropy pass SOURCES]",
			[]string{"bitcoin bitcoin_block", "stellar stellar_ledger"}},
		{"entropy_nist", "/subject/record/entropy_hash", Pointer{Name("subject"), Name("entropy")},
			"[prior_failure /subject/entropy skip SOURCES]", nil},
	} {
		proof := proofs[c.proof]
		changed := put(proof, under(c.object, "payload", "pulse", "outputValue"), strings.Repeat("A", 128))
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		sets := soundSources(t, proof, stand)
		run := deriveRun(changed, deriveAttribution)
		run.Config.Online = true
		run.Config.Networks.Entropy = specdata.ChainNetworks{Bitcoin: "regtest", Stellar: "testnet"}
		run.Config.ReadChain = stand.reader
		run.SourceSets = sets
		Derive(run)
		if row := rowAtStep(run, "DERIVE", c.failed); row.Status != StatusFail || run.ExternalGate != ExternalGatePriorFailure {
			t.Fatalf("%s: the row at %s %v, the gate %q", c.proof, c.failed, row, run.ExternalGate)
		}
		Sources(run)
		closed := strings.ReplaceAll(strings.ReplaceAll(c.rows, "[null ", "[prior_failure "), " pass SOURCES]", " skip SOURCES]")
		if got := sourcesRows(run); got != closed || len(stand.asked) != 0 {
			t.Fatalf("%s: rows %s, reads %v", c.proof, got, stand.asked)
		}
		open := deriveRun(changed, deriveAttribution)
		open.Config = run.Config
		open.SourceSets = sets
		Derive(open)
		open.ExternalGate = ExternalGateOpen
		Sources(open)
		if got := sourcesRows(open); got != c.rows {
			t.Fatalf("%s: rows %s\nwant %s", c.proof, got, c.rows)
		}
		if got := stand.asked; !slices.Equal(got, c.asked) {
			t.Fatalf("%s: reads %v, want %v", c.proof, got, c.asked)
		}
	}
}

// rowAtStep is the run's row of step at part, or the zero Row.
func rowAtStep(run *Run, step, part string) Row {
	for _, r := range run.Rows {
		if r.Step == step && r.Part != nil && *r.Part == part {
			return r
		}
	}
	return Row{}
}

// SOURCES reads each Stellar and Bitcoin object from the sources of the
// run's entropy pin, never its commitment pin's, whose network can differ
// (SPEC.md section 10.4, "Sources"): with the commitment pins' sources cs
// and cb, which have no answers, and the entropy pins Stellar public and
// Bitcoin mainnet served by es and eb, every row passes from es and eb,
// and cs and cb are asked nothing. Mutant killed: sourcesOpen reading the
// commitment pin's sources (cs and cb are asked, and both rows skip
// source_unreachable).
func TestEntropy_ReadsTheEntropyPin(t *testing.T) {
	item := positiveProofs(t)["item"]
	first := SourcePolicy{Kind: SourcePolicyFirst}
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	stand.answerEntropy(t, entropyEntryAt(t, item, witnessNIST), "", SourceRead{})
	stand.answerEntropy(t, entropyEntryAt(t, item, witnessStellar), "public", SourceRead{}, "es")
	stand.answerEntropy(t, entropyEntryAt(t, item, witnessBitcoin), "mainnet", SourceRead{}, "eb")
	sets := []SourceSet{stellarSetOn(first, "testnet", "cs"), bitcoinSet(first, "regtest", "cb"), stellarSetOn(first, "public", "es"), bitcoinSet(first, "mainnet", "eb")}
	run := entropyRun(t, item, "public", "mainnet", sets, stand)
	Sources(run)
	want := "[null /witnesses/entropy_bitcoin pass SOURCES] [null /witnesses/entropy_nist pass SOURCES] [null /witnesses/entropy_stellar pass SOURCES]"
	if got := sourcesRows(run); got != want {
		t.Errorf("rows %s\nwant %s", got, want)
	}
	for name, want := range map[string][]string{"cs": nil, "cb": nil, "es": {"stellar_ledger"}, "eb": {"bitcoin_block"}} {
		if got := askedFor(stand, name); !slices.Equal(got, want) {
			t.Errorf("%s was asked %v, want %v", name, got, want)
		}
	}
}

// A Stellar and a Bitcoin witness's row under each policy, from several
// sources, each source's rule made as if it were the only one (SPEC.md
// section 10.4, "Several sources"), and NIST's always from its one source
// under first: under first the sources are asked in order and none after
// the first conclusive one, a source whose identity was unreachable is
// asked nothing; under all and at_least n every source is asked, a pass
// beside a failure, or two Bitcoin passes with median times at different
// instants, disagree, a pass beside an unreachable source is incomplete,
// two not found are not_held_by_source, a not found beside an unreachable
// source is incomplete, and the candidate rests on every source that
// passed. Mutants killed: CombineSources taking the first source alone
// for a SOURCES row (all's candidate rests on one); the policy ignored for
// SOURCES (first is asked every source); a candidate kept for a row that
// did not pass; the NIST beacon given the Stellar set (its row skips).
func TestEntropy_SeveralSourcesUnderEachPolicy(t *testing.T) {
	item := positiveProofs(t)["item"]
	stellar, bitcoin, nist := entropyEntryAt(t, item, witnessStellar), entropyEntryAt(t, item, witnessBitcoin), entropyEntryAt(t, item, witnessNIST)
	first, all, atLeast2 := SourcePolicy{Kind: SourcePolicyFirst}, SourcePolicy{Kind: SourcePolicyAll}, SourcePolicy{Kind: SourcePolicyAtLeast, N: 2}
	differs := map[string]SourceRead{
		ChainStellar: answerWith(soundEntropy(t, stellar), map[string]any{"hash": strings.Repeat("ab", 32)}),
		ChainBitcoin: answerWith(soundEntropy(t, bitcoin), map[string]any{"height": intValue(1)}),
	}
	later := answerWith(soundEntropy(t, bitcoin), map[string]any{"median_time_past": intValue(witnessMedian + 1)})
	type answer struct {
		name string
		read SourceRead
	}
	for _, chain := range []string{ChainStellar, ChainBitcoin} {
		e, network, part, basis, instant := stellar, "testnet", witnessStellar.Encode(), "stellar_ledger_close_read", "2026-09-26T01:48:30Z"
		kind := "stellar_ledger"
		if chain == ChainBitcoin {
			e, network, part, basis, instant, kind = bitcoin, "regtest", witnessBitcoin.Encode(), "bitcoin_median_time_read", "2026-09-26T01:40:00Z", "bitcoin_block"
		}
		for _, tc := range []struct {
			name    string
			policy  SourcePolicy
			sources []string
			answers []answer
			row     string
			counted string
			asked   map[string]int
		}{
			{"first, the first passes", first, []string{"a", "b"}, []answer{{"a", SourceRead{}}, {"b", SourceRead{}}}, "null pass", "[a]", map[string]int{"a": 1, "b": 0}},
			{"first, past an unreachable read", first, []string{"a", "b"}, []answer{{"a", unreachableRead}, {"b", SourceRead{}}}, "null pass", "[b]", map[string]int{"a": 1, "b": 1}},
			{"first, past an unreachable identity", first, []string{"down", "b"}, []answer{{"b", SourceRead{}}}, "null pass", "[b]", map[string]int{"down": 0, "b": 1}},
			{"first, stops at a failure", first, []string{"a", "b"}, []answer{{"a", differs[chain]}, {"b", SourceRead{}}}, "entropy_mismatch fail", "", map[string]int{"a": 1, "b": 0}},
			{"all agree", all, []string{"a", "b"}, []answer{{"a", SourceRead{}}, {"b", SourceRead{}}}, "null pass", "[a b]", map[string]int{"a": 1, "b": 1}},
			{"all, a pass and a failure", all, []string{"a", "b"}, []answer{{"a", SourceRead{}}, {"b", differs[chain]}}, "sources_disagree skip", "", nil},
			{"all, a pass and an unreachable source", all, []string{"a", "b"}, []answer{{"a", SourceRead{}}, {"b", unreachableRead}}, "sources_incomplete skip", "", nil},
			{"all, neither holds it", all, []string{"a", "b"}, []answer{{"a", notFound}, {"b", notFound}}, "not_held_by_source skip", "", nil},
			{"all, one holds nothing and one is unreachable", all, []string{"a", "b"}, []answer{{"a", notFound}, {"b", unreachableRead}}, "sources_incomplete skip", "", nil},
			{"all fail", all, []string{"a", "b"}, []answer{{"a", differs[chain]}, {"b", differs[chain]}}, "entropy_mismatch fail", "", nil},
			{"at_least 2, one unreachable", atLeast2, []string{"a", "b", "c"}, []answer{{"a", SourceRead{}}, {"b", SourceRead{}}, {"c", unreachableRead}}, "null pass", "[a b]", map[string]int{"c": 1}},
			{"at_least 2, two fail", atLeast2, []string{"a", "b", "c"}, []answer{{"a", differs[chain]}, {"b", differs[chain]}, {"c", unreachableRead}}, "entropy_mismatch fail", "", nil},
		} {
			stand := &chainStandIn{answers: map[string]SourceRead{}}
			for _, a := range tc.answers {
				stand.answerEntropy(t, e, network, a.read, a.name)
				// A Stellar ledger not found reads the root, whose latest is
				// far past the sequence.
				stand.answers[standInKey("https://"+a.name+".example", rootRead(network))] = root(stellar.sequence+500, 1)
			}
			stand.answerEntropy(t, nist, "", SourceRead{})
			set := stellarSetOn(tc.policy, network, tc.sources...)
			if chain == ChainBitcoin {
				set = bitcoinSet(tc.policy, network, tc.sources...)
			}
			sets := []SourceSet{set}
			other := map[string]SourceSet{ChainStellar: bitcoinSet(first, "regtest", "x"), ChainBitcoin: stellarSetOn(first, "testnet", "x")}[chain]
			run := entropyRun(t, item, "testnet", "regtest", append(sets, other), stand)
			Sources(run)
			row := rowAtStep(run, "SOURCES", part)
			code := "null"
			if row.Code != "" {
				code = string(row.Code)
			}
			if got := code + " " + string(row.Status); got != tc.row {
				t.Errorf("%s, %s: row %s, want %s", chain, tc.name, got, tc.row)
			}
			var kept []RowCandidate
			for _, c := range run.Candidates {
				if c.Part == part {
					kept = append(kept, c)
				}
			}
			wantCandidates := ""
			if tc.counted != "" {
				wantCandidates = "SOURCES " + part + " " + basis + " " + instant + " " + tc.counted
			}
			if got := candidateText(&Run{Candidates: kept}); got != wantCandidates {
				t.Errorf("%s, %s: candidates %q, want %q", chain, tc.name, got, wantCandidates)
			}
			for name, n := range tc.asked {
				if got := askedFor(stand, name); len(got) != n || (n > 0 && got[0] != kind) {
					t.Errorf("%s, %s: %s was asked %v, want %d reads of %s", chain, tc.name, name, got, n, kind)
				}
			}
			if got := rowAtStep(run, "SOURCES", witnessNIST.Encode()); got.Status != StatusPass || len(got.Values["outcomes"].([]any)) != 1 || got.Values["policy"] != "first" {
				t.Errorf("%s, %s: the NIST row %v", chain, tc.name, got)
			}
		}
	}
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	stand.answerEntropy(t, bitcoin, "regtest", SourceRead{}, "a")
	stand.answerEntropy(t, bitcoin, "regtest", later, "b")
	stand.answerEntropy(t, nist, "", SourceRead{})
	run := entropyRun(t, item, "testnet", "regtest", []SourceSet{bitcoinSet(all, "regtest", "a", "b"), stellarSetOn(first, "testnet", "x")}, stand)
	Sources(run)
	if row := rowAtStep(run, "SOURCES", witnessBitcoin.Encode()); row.Code != specdata.ReasonSourcesDisagree {
		t.Errorf("two median times at different instants: %v", row)
	}
}

// A SOURCES row's values are what it read (SPEC.md section 10.4, "What a
// row read"), exactly: policy, required, one outcome for each source
// asked, each with its reads in replay form, NIST's naming no network, and
// the candidate; and each row's message names what was read whole and is
// worded from the reads that settled it. Mutants killed: the NIST read
// naming an empty network (the stand-in holds no such read, and the row
// skips); the row made without CombinedOutcome.Values (no outcomes); the
// Stellar source_unreachable message chosen from the code alone (the
// root's case says nothing of the ledgers it holds); a source asked
// nothing worded as one whose read failed.
func TestEntropy_ARowsValuesAndMessage(t *testing.T) {
	t0 := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	useClock(t, t0, t0.Add(time.Second), t0.Add(2*time.Second))
	item := positiveProofs(t)["item"]
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	sets := soundSources(t, item, stand)
	run := entropyRun(t, item, "testnet", "regtest", sets, stand)
	Sources(run)
	values := map[string]string{}
	for _, at := range []Pointer{witnessBitcoin, witnessNIST, witnessStellar} {
		encoded, err := jcs.Encode(rowAtStep(run, "SOURCES", at.Encode()).Values)
		if err != nil {
			t.Fatal(err)
		}
		values[at.Encode()] = string(encoded)
	}
	bitcoin, stellar := entropyEntryAt(t, item, witnessBitcoin), entropyEntryAt(t, item, witnessStellar)
	want := map[string]string{
		witnessBitcoin.Encode(): `{"candidate":{"basis":"bitcoin_median_time_read","instant":"2026-09-26T01:40:00Z"},"outcomes":[{"cached":false,"candidate":{"basis":"bitcoin_median_time_read","instant":"2026-09-26T01:40:00Z"},"fetched_at":"2026-10-20T09:00:00.000Z",` +
			`"reads":[{"answer":{"outcome":"found","value":{"height":215,"median_time_past":1790386800,"time":1790387100}},"cached":false,"fetched_at":"2026-10-20T09:00:00.000Z","read":{"hash":"` + bitcoin.hash + `","kind":"bitcoin_block","network":"regtest"}}],` +
			`"source":"https://bitcoin.example","status":"pass"}],"policy":"first","required":1}`,
		witnessNIST.Encode(): `{"candidate":{"basis":"nist_pulse_time_read","instant":"2026-09-26T01:48:00.000Z"},"outcomes":[{"cached":false,"candidate":{"basis":"nist_pulse_time_read","instant":"2026-09-26T01:48:00.000Z"},"fetched_at":"2026-10-20T09:00:01.000Z",` +
			`"reads":[{"answer":{"outcome":"found","value":{"outputValue":"FDBA0E7DA81C11481C72F9B01CBAE7C364C53270C8EDFFB34537C8F7028D75163D70C142873DD560A6E985101A99E02BD7C7EA6ABA8BD36F3C581CB65C2F6216","timeStamp":"2026-09-26T01:48:00.000Z"}},"cached":false,"fetched_at":"2026-10-20T09:00:01.000Z","read":{"chainIndex":2,"kind":"nist_pulse","pulseIndex":1200018}}],` +
			`"source":"https://beacon.nist.gov","status":"pass"}],"policy":"first","required":1}`,
		witnessStellar.Encode(): `{"candidate":{"basis":"stellar_ledger_close_read","instant":"2026-09-26T01:48:30Z"},"outcomes":[{"cached":false,"candidate":{"basis":"stellar_ledger_close_read","instant":"2026-09-26T01:48:30Z"},"fetched_at":"2026-10-20T09:00:02.000Z",` +
			`"reads":[{"answer":{"outcome":"found","value":{"closed_at":"2026-09-26T01:48:30Z","hash":"` + stellar.hash + `"}},"cached":false,"fetched_at":"2026-10-20T09:00:02.000Z","read":{"kind":"stellar_ledger","network":"testnet","sequence":1000222}}],` +
			`"source":"https://stellar.example","status":"pass"}],"policy":"first","required":1}`,
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("values\n%v\nwant\n%v", values, want)
	}
	nist := entropyEntryAt(t, item, witnessNIST)
	for _, tc := range []struct {
		name    string
		e       entropyEntry
		policy  SourcePolicy
		sources []string
		answers map[string]SourceRead
		roots   map[string]SourceRead
		message string
	}{
		{"a NIST pass", nist, SourcePolicy{}, nil, nil, nil,
			"Read from NIST's Randomness Beacon: the pulse at chainIndex 2 and pulseIndex 1200018 holds this entropy object's outputValue and its timeStamp, 2026-09-26T01:48:00.000Z."},
		{"a Stellar pass", stellar, SourcePolicy{Kind: SourcePolicyFirst}, []string{"a"}, map[string]SourceRead{"a": {}}, nil,
			"Read from the pinned Stellar testnet network: ledger 1000222 has this entropy object's hash and closed at 2026-09-26T01:48:30Z, as it says."},
		{"a Bitcoin pass", bitcoin, SourcePolicy{Kind: SourcePolicyFirst}, []string{"a"}, map[string]SourceRead{"a": {}}, nil,
			"Read from the pinned Bitcoin regtest network: the block " + bitcoin.hash + " is at this entropy object's height 215 with its time, and its median time past is 2026-09-26T01:40:00Z."},
		{"a NIST mismatch", nist, SourcePolicy{}, nil, map[string]SourceRead{"": answerWith(soundEntropy(t, nist), map[string]any{"outputValue": otherNISTHex})}, nil,
			"Contradicted by NIST's Randomness Beacon: the pulse at chainIndex 2 and pulseIndex 1200018 differs from this entropy object."},
		{"a pulse not held", nist, SourcePolicy{}, nil, map[string]SourceRead{"": notFound}, nil,
			"Not confirmed: NIST's Randomness Beacon does not hold the pulse at chainIndex 2 and pulseIndex 1200018. Absence is no contradiction."},
		{"the beacon unreachable", nist, SourcePolicy{}, nil, map[string]SourceRead{"": unreachableRead}, nil,
			"Not confirmed: NIST's Randomness Beacon gave no usable answer about the pulse at chainIndex 2 and pulseIndex 1200018."},
		{"a ledger not yet indexed", stellar, SourcePolicy{Kind: SourcePolicyFirst}, []string{"a"}, map[string]SourceRead{"a": notFound}, map[string]SourceRead{"a": root(stellar.sequence, 1)},
			"Not confirmed: the source of the pinned Stellar testnet network does not hold ledger 1000222 yet, which is beyond, or within 60 ledgers of, the latest ledger it holds."},
		{"a ledger not held by two", stellar, SourcePolicy{Kind: SourcePolicyAll}, []string{"a", "b"}, map[string]SourceRead{"a": notFound, "b": notFound},
			map[string]SourceRead{"a": root(stellar.sequence+500, 1), "b": root(stellar.sequence+500, 1)},
			"Not confirmed: the history each source of the pinned Stellar testnet network keeps does not hold ledger 1000222. Absence is no contradiction."},
		{"a block not held", bitcoin, SourcePolicy{Kind: SourcePolicyFirst}, []string{"a"}, map[string]SourceRead{"a": notFound}, nil,
			"Not confirmed: the source of the pinned Bitcoin regtest network does not hold the block " + bitcoin.hash + ", which need not be on the best chain. Absence is no contradiction."},
		{"the root unreachable", stellar, SourcePolicy{Kind: SourcePolicyFirst}, []string{"a"}, map[string]SourceRead{"a": notFound}, map[string]SourceRead{"a": unreachableRead},
			"Not confirmed: no source of the pinned Stellar testnet network holds ledger 1000222, and none gave a usable answer about the ledgers it holds."},
		{"the ledger unreachable", stellar, SourcePolicy{Kind: SourcePolicyFirst}, []string{"a"}, map[string]SourceRead{"a": unreachableRead}, nil,
			"Not confirmed: no source of the pinned Stellar testnet network gave a usable answer about ledger 1000222."},
		{"the identity unreachable", bitcoin, SourcePolicy{Kind: SourcePolicyFirst}, []string{"down"}, nil, nil,
			"Not confirmed: no source of the pinned Bitcoin regtest network had its identity confirmed, so none was read."},
		{"a down source and an unreachable read", bitcoin, SourcePolicy{Kind: SourcePolicyAll}, []string{"a", "down"}, map[string]SourceRead{"a": unreachableRead}, nil,
			"Not confirmed: no source of the pinned Bitcoin regtest network gave a usable answer to every read this row makes."},
		{"two disagree", bitcoin, SourcePolicy{Kind: SourcePolicyAll}, []string{"a", "b"},
			map[string]SourceRead{"a": {}, "b": answerWith(soundEntropy(t, bitcoin), map[string]any{"time": intValue(1)})}, nil,
			"Not confirmed: the sources of the pinned Bitcoin regtest network disagree on the block " + bitcoin.hash + ". A disagreement is no failure."},
		{"too few", stellar, SourcePolicy{Kind: SourcePolicyAll}, []string{"a", "b"}, map[string]SourceRead{"a": {}, "b": unreachableRead}, nil,
			"Not confirmed: fewer sources of the pinned Stellar testnet network than its policy all asks for reached an outcome on ledger 1000222."},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		for _, object := range []entropyEntry{nist, stellar, bitcoin} {
			network := map[string]string{ChainStellar: "testnet", ChainBitcoin: "regtest"}[object.source]
			stand.answerEntropy(t, object, network, SourceRead{}, "x")
		}
		network := map[string]string{ChainStellar: "testnet", ChainBitcoin: "regtest"}[tc.e.source]
		for name, answer := range tc.answers {
			stand.answerEntropy(t, tc.e, network, answer, name)
		}
		for name, answer := range tc.roots {
			stand.answers[standInKey("https://"+name+".example", rootRead(network))] = answer
		}
		first := SourcePolicy{Kind: SourcePolicyFirst}
		sets := []SourceSet{stellarSetOn(first, "testnet", "x"), bitcoinSet(first, "regtest", "x")}
		switch tc.e.source {
		case ChainStellar:
			sets[0] = stellarSetOn(tc.policy, "testnet", tc.sources...)
		case ChainBitcoin:
			sets[1] = bitcoinSet(tc.policy, "regtest", tc.sources...)
		}
		run := entropyRun(t, item, "testnet", "regtest", sets, stand)
		Sources(run)
		if got := rowAtStep(run, "SOURCES", tc.e.at.Encode()).Message; got != tc.message {
			t.Errorf("%s: %q\nwant %q", tc.name, got, tc.message)
		}
	}
}

// An entropy object SOURCES cannot read, one SHAPE should have refused, is
// a fault: internal_error at SOURCES, part null, and the run ends, its
// later steps' rows skip prior_failure, with no read made: a Bitcoin
// height that is a string, a NIST timeStamp with no milliseconds and a
// Stellar hash in uppercase, each set after DERIVE. Mutants killed: the
// fault not ending the run (EXPECTED_HASH gives its own row); the fault
// ignored (the objects are read); entropyEntryOf without its class checks
// (the uppercase hash is read).
func TestEntropy_AnUnreadableObjectIsAFault(t *testing.T) {
	item := positiveProofs(t)["item"]
	for _, tc := range []struct {
		name  string
		proof map[string]any
	}{
		{"a Bitcoin height string", put(item, under(witnessBitcoin, "payload", "height"), "215")},
		{"a NIST timeStamp without milliseconds", put(item, under(witnessNIST, "payload", "pulse", "timeStamp"), "2026-09-26T01:48:00Z")},
		{"a Stellar hash in uppercase", put(item, under(witnessStellar, "payload", "hash"), strings.Repeat("AB", 32))},
	} {
		stand := &chainStandIn{answers: map[string]SourceRead{}}
		sets := soundSources(t, item, stand)
		run := entropyRun(t, item, "testnet", "regtest", sets, stand)
		run.Proof = tc.proof
		before := len(run.Rows)
		Sources(run)
		Window(run)
		ExpectedHash(run)
		if got := rowSummary(run.Rows[before:]); got != "[internal_error null fail SOURCES] [prior_failure null skip WINDOW] [prior_failure null skip EXPECTED_HASH]" || !run.Ended || len(stand.asked) != 0 {
			t.Errorf("%s: rows %s, ended %v, reads %v", tc.name, got, run.Ended, stand.asked)
		}
	}
}

// DefaultChainReader makes SOURCES' reads over HTTP through the source
// package: NIST's pulse against a loopback beacon, found in replay form;
// Esplora's and Bitcoin Core's block by its hash, found, with the
// credentials sent to Core alone; each not-found answer read live every
// time, the same read made twice through one reader and once through
// another reaching a counting server three times (SPEC.md section 10.4,
// "Caches"); and the compiled-in NIST beacon and the Bitcoin defaults'
// by-hash reads refused in process once each by this package's TestMain,
// unreachable. Mutants killed: source.Read giving the beacon no read
// (every pulse is unreachable with no request); DefaultChainReader keeping
// each not-found answer by URL and read (fewer requests); the beacon's
// address not the compiled-in one (no request reaches the refusal).
func TestEntropy_TheDefaultReader(t *testing.T) {
	const output = "FDBA0E7DA81C11481C72F9B01CBAE7C364C53270C8EDFFB34537C8F7028D75163D70C142873DD560A6E985101A99E02BD7C7EA6ABA8BD36F3C581CB65C2F6216"
	hash := strings.Repeat("ab", 32)
	var beaconHits, esploraHits, coreHits atomic.Int64
	var coreAuth atomic.Value
	coreAuth.Store("")
	beacon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		beaconHits.Add(1)
		if r.URL.Path != "/chain/2/pulse/1200018" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintf(w, `{"pulse":{"chainIndex":2,"pulseIndex":1200018,"outputValue":%q,"timeStamp":"2026-09-26T01:48:00.000Z"}}`, output)
	}))
	defer beacon.Close()
	esplora := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		esploraHits.Add(1)
		if r.Header.Get("Authorization") != "" || r.URL.Path != "/block/"+hash {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"height":215,"timestamp":1790387100,"mediantime":1790386800}`))
	}))
	defer esplora.Close()
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreHits.Add(1)
		coreAuth.Store(r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), hash) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"result":null,"error":{"code":-5,"message":"Block not found"},"id":"truestamp-verify"}`))
			return
		}
		_, _ = w.Write([]byte(`{"result":{"height":215,"time":1790387100,"mediantime":1790386800},"error":null,"id":"truestamp-verify"}`))
	}))
	defer node.Close()
	reader := DefaultChainReader()
	pulse := map[string]any{"kind": source.KindNISTPulse, "chainIndex": strictjson.Number(2), "pulseIndex": strictjson.Number(1200018)}
	beaconSource := Source{Chain: SourceNIST, Provider: source.NISTBeacon, URL: beacon.URL}
	want := SourceRead{Outcome: ReadFound, Value: map[string]any{"outputValue": output, "timeStamp": "2026-09-26T01:48:00.000Z"}}
	if got := reader(beaconSource, pulse); !reflect.DeepEqual(got, want) {
		t.Fatalf("the pulse: %+v", got)
	}
	block := map[string]any{"kind": source.KindBitcoinBlock, "network": "regtest", "hash": hash}
	creds := source.NewCredentials("rpcuser-planted-5d1", "planted-secret-5d1")
	esploraSource := Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: esplora.URL, Network: "regtest"}
	coreSource := Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: node.URL, Network: "regtest", Credentials: creds}
	wantBlock := SourceRead{Outcome: ReadFound, Value: map[string]any{"height": strictjson.Number(215), "median_time_past": strictjson.Number(1790386800), "time": strictjson.Number(1790387100)}}
	for _, s := range []Source{esploraSource, coreSource} {
		if got := reader(s, block); !reflect.DeepEqual(got, wantBlock) {
			t.Fatalf("%s's block: %+v", s.Provider, got)
		}
	}
	if auth, _ := coreAuth.Load().(string); !strings.HasPrefix(auth, "Basic ") {
		t.Fatalf("Bitcoin Core was sent %q", auth)
	}
	beaconHits.Store(0)
	esploraHits.Store(0)
	coreHits.Store(0)
	missingPulse := map[string]any{"kind": source.KindNISTPulse, "chainIndex": strictjson.Number(2), "pulseIndex": strictjson.Number(9)}
	missingBlock := map[string]any{"kind": source.KindBitcoinBlock, "network": "regtest", "hash": strings.Repeat("cd", 32)}
	for i, r := range []ChainReader{reader, reader, DefaultChainReader()} {
		for _, read := range []struct {
			s    Source
			read map[string]any
		}{{beaconSource, missingPulse}, {esploraSource, missingBlock}, {coreSource, missingBlock}} {
			if got := r(read.s, read.read); got.Outcome != ReadNotFound {
				t.Fatalf("read %d of %s: %+v", i, read.s.Provider, got)
			}
		}
	}
	if beaconHits.Load() != 3 || esploraHits.Load() != 3 || coreHits.Load() != 3 {
		t.Fatalf("three not-found reads reached the beacon %d, Esplora %d and Bitcoin Core %d times", beaconHits.Load(), esploraHits.Load(), coreHits.Load())
	}
	defaults := []Source{nistSet().Sources[0].Source}
	for _, network := range []string{"mainnet", "testnet"} {
		s, _ := DefaultSource(ChainBitcoin, network)
		defaults = append(defaults, s)
	}
	for _, s := range defaults {
		read := block
		if s.Provider == source.NISTBeacon {
			read = pulse
		}
		before := refusals.Load()
		if got := DefaultChainReader()(s, read); got.Outcome != ReadUnreachable || refusals.Load()-before != 1 {
			t.Fatalf("%s: %+v, %d refused", s.URL, got, refusals.Load()-before)
		}
	}
	if nistSet().Sources[0].Source.URL != NISTBeaconURL || NISTBeaconURL != "https://beacon.nist.gov/beacon/2.0" {
		t.Fatalf("the beacon's address %q", NISTBeaconURL)
	}
}
