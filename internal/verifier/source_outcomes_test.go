// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// The frame of a row read from several sources (SPEC.md section 10.4,
// "Several sources" and "What a row read"), over a stand-in rule: each
// source's outcome is given by the test, by the source's URL, and the
// rule records which sources it was made for. No read is made.

// standIn is a rule that gives each source the outcome outcomes holds for
// its URL and records each source it is made for.
type standIn struct {
	outcomes map[string]SourceOutcome
	asked    []string
}

func (s *standIn) rule(record SourceRecord) SourceOutcome {
	s.asked = append(s.asked, record.Source.URL)
	return s.outcomes[record.Source.URL]
}

// identityAt is when each stand-in source's identity read came back.
var identityAt = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

// standInSet is a Stellar testnet network under policy whose sources are
// https://<name>.example for each of names, each confirmed but those whose
// name begins "down", whose identity read got no usable answer.
func standInSet(policy SourcePolicy, names ...string) SourceSet {
	set := SourceSet{Chain: ChainStellar, Network: "testnet", Policy: policy}
	for _, name := range names {
		record := SourceRecord{
			Source:    Source{Chain: ChainStellar, Provider: source.Horizon, URL: "https://" + name + ".example", Network: "testnet"},
			From:      FromOperator,
			Status:    SourceConfirmed,
			FetchedAt: identityAt,
		}
		if strings.HasPrefix(name, "down") {
			record.Status, record.Reason = SourceUnreachable, "HTTP 503"
		}
		set.Sources = append(set.Sources, record)
	}
	return set
}

// at is a candidate of the Stellar ledger close read, at second s past
// the epoch.
func at(s int64) *Candidate {
	return &Candidate{Basis: "stellar_ledger_close_read", Class: ClassTIME, Instant: s, FetchedAt: identityAt.Add(time.Minute)}
}

// passes, fails and skips are the outcomes of a source.
func passes(c *Candidate) SourceOutcome { return SourceOutcome{Status: StatusPass, Candidate: c} }

var fails = SourceOutcome{Status: StatusFail, Code: specdata.ReasonChainMismatch}

func skips(code specdata.Reason) SourceOutcome { return SourceOutcome{Status: StatusSkip, Code: code} }

// combine runs the stand-in over a set of the sources named by the
// outcomes' order, outcomes[i] for the i-th; a name beginning "down" has
// no confirmed identity, and its outcome is not given to the rule.
func combine(policy SourcePolicy, names []string, outcomes ...SourceOutcome) (CombinedOutcome, []string) {
	s := &standIn{outcomes: map[string]SourceOutcome{}}
	for i, name := range names {
		s.outcomes["https://"+name+".example"] = outcomes[i]
	}
	combined := CombineSources(standInSet(policy, names...), s.rule)
	var asked []string
	for _, url := range s.asked {
		asked = append(asked, strings.TrimSuffix(strings.TrimPrefix(url, "https://"), ".example"))
	}
	return combined, asked
}

var (
	first   = SourcePolicy{Kind: SourcePolicyFirst}
	all     = SourcePolicy{Kind: SourcePolicyAll}
	atLeast = func(n int) SourcePolicy { return SourcePolicy{Kind: SourcePolicyAtLeast, N: n} }
)

// summary writes a combined outcome as "status code candidate counted
// asked", the candidate as its instant in milliseconds or "-".
func summary(c CombinedOutcome, asked []string) string {
	candidate := "-"
	if c.Candidate != nil {
		candidate = fmt.Sprint(c.Candidate.Milliseconds())
	}
	var counted []string
	for _, a := range c.Counted {
		counted = append(counted, strings.TrimSuffix(strings.TrimPrefix(a.Source.Source.URL, "https://"), ".example"))
	}
	code := string(c.Code)
	if code == "" {
		code = "-"
	}
	return fmt.Sprintf("%s %s %s [%s] [%s]", c.Status, code, candidate, strings.Join(counted, " "), strings.Join(asked, " "))
}

// Every outcome mix SPEC.md section 10.4 names, under each policy, as the
// rule's outcomes give it: under first, the first conclusive outcome, no
// source after it asked, and with none conclusive the same skip or
// sources_incomplete; under at_least n and all, rule 1 (a pass beside a
// failure, or two passes at different instants: sources_disagree) before
// rules 2 and 3 (n passes: pass, with their shared candidate, resting on
// every source that passed; n failures: fail), then rule 4 (every source
// skipped with one code: that skip) and rule 5 (sources_incomplete); two
// passes at one instant, one read to the second and one to the
// millisecond, agree; a source whose identity read got no usable answer is
// skip source_unreachable, with the rule not made for it. SPEC.md's own
// examples are among them: under all, a pass beside a source that does
// not answer is sources_incomplete, and of three sources, two that
// conflict beside one that times out disagree. Mutants killed: first
// asking every source (the later sources are asked); first taking the
// first outcome, conclusive or not (a skip is taken); rule 1 after rule 2
// (two passes and a failure under at_least 2 pass); rule 1 without the
// candidates' instants (passes at 100 s and 200 s pass); instants
// compared in their own units (100 s and 100,000 ms disagree); rule 4
// taking the first source's code unseen (two skip codes give the first);
// rule 4 comparing only the first two codes (three skips, the last
// differing, give the first's code); the rule made for a source with no
// confirmed identity (down is asked); Counted only the first pass (under
// all, two counted).
func TestCombineSources_EveryMixUnderEachPolicy(t *testing.T) {
	c100, c200 := at(100), at(200)
	nist := &Candidate{Basis: "nist_pulse_time_read", Class: ClassNISTTIME, Instant: 100000}
	unreachable, notHeld, notIndexed, notFinal := specdata.ReasonSourceUnreachable, specdata.ReasonNotHeldBySource,
		specdata.ReasonNotYetIndexed, specdata.ReasonChainNotFinal
	for _, c := range []struct {
		name     string
		policy   SourcePolicy
		names    []string
		outcomes []SourceOutcome
		want     string
	}{
		{"first: one pass", first, []string{"a"}, []SourceOutcome{passes(c100)}, "pass - 100000 [a] [a]"},
		{"first: a skip, a pass, a failure", first, []string{"a", "b", "c"}, []SourceOutcome{skips(notIndexed), passes(c100), fails}, "pass - 100000 [b] [a b]"},
		{"first: a failure first", first, []string{"a", "b"}, []SourceOutcome{fails, passes(c100)}, "fail chain_mismatch - [a] [a]"},
		{"first: a skip, then a failure", first, []string{"a", "b"}, []SourceOutcome{skips(notHeld), fails}, "fail chain_mismatch - [b] [a b]"},
		{"first: an unconfirmed identity, then a pass", first, []string{"down", "b"}, []SourceOutcome{{}, passes(c100)}, "pass - 100000 [b] [b]"},
		{"first: one skip code", first, []string{"a", "b"}, []SourceOutcome{skips(notHeld), skips(notHeld)}, "skip not_held_by_source - [] [a b]"},
		{"first: two skip codes", first, []string{"a", "b"}, []SourceOutcome{skips(unreachable), skips(notIndexed)}, "skip sources_incomplete - [] [a b]"},
		{"first: unconfirmed, then unreachable", first, []string{"down", "b"}, []SourceOutcome{{}, skips(unreachable)}, "skip source_unreachable - [] [b]"},
		{"at_least 2: two passes and a failure", atLeast(2), []string{"a", "b", "c"}, []SourceOutcome{passes(c100), passes(c100), fails}, "skip sources_disagree - [] [a b c]"},
		{"at_least 2: two passes and a skip", atLeast(2), []string{"a", "b", "c"}, []SourceOutcome{passes(c100), skips(unreachable), passes(c100)}, "pass - 100000 [a c] [a b c]"},
		{"at_least 2: passes at two instants", atLeast(2), []string{"a", "b", "c"}, []SourceOutcome{passes(c100), passes(c200), skips(unreachable)}, "skip sources_disagree - [] [a b c]"},
		{"at_least 2: one instant, two precisions", atLeast(2), []string{"a", "b"}, []SourceOutcome{passes(c100), passes(nist)}, "pass - 100000 [a b] [a b]"},
		{"at_least 2: two failures and a skip", atLeast(2), []string{"a", "b", "c"}, []SourceOutcome{fails, skips(notHeld), fails}, "fail chain_mismatch - [a c] [a b c]"},
		{"at_least 2: a failure and two skips of one code", atLeast(2), []string{"a", "b", "c"}, []SourceOutcome{fails, skips(notHeld), skips(notHeld)}, "skip sources_incomplete - [] [a b c]"},
		{"at_least 2: one pass and two skips", atLeast(2), []string{"a", "b", "c"}, []SourceOutcome{skips(notHeld), passes(c100), skips(notHeld)}, "skip sources_incomplete - [] [a b c]"},
		{"at_least 2: three skips of one code", atLeast(2), []string{"a", "b", "c"}, []SourceOutcome{skips(notFinal), skips(notFinal), skips(notFinal)}, "skip chain_not_final - [] [a b c]"},
		{"at_least 2: three skips, the last differing", atLeast(2), []string{"a", "b", "c"}, []SourceOutcome{skips(notFinal), skips(notFinal), skips(notIndexed)}, "skip sources_incomplete - [] [a b c]"},
		{"at_least 1: a pass and a skip", atLeast(1), []string{"a", "b"}, []SourceOutcome{skips(unreachable), passes(c100)}, "pass - 100000 [b] [a b]"},
		{"at_least 1: an unconfirmed identity and a failure", atLeast(1), []string{"down", "b"}, []SourceOutcome{{}, fails}, "fail chain_mismatch - [b] [b]"},
		{"all: a pass beside a source that does not answer", all, []string{"a", "b"}, []SourceOutcome{passes(c100), skips(unreachable)}, "skip sources_incomplete - [] [a b]"},
		{"all: two in conflict beside one that times out", all, []string{"a", "b", "c"}, []SourceOutcome{passes(c100), fails, skips(unreachable)}, "skip sources_disagree - [] [a b c]"},
		{"all: every source passes", all, []string{"a", "b"}, []SourceOutcome{passes(c100), passes(c100)}, "pass - 100000 [a b] [a b]"},
		{"all: every source fails", all, []string{"a", "b"}, []SourceOutcome{fails, fails}, "fail chain_mismatch - [a b] [a b]"},
		{"all: every source skips with one code", all, []string{"a", "down"}, []SourceOutcome{skips(unreachable), {}}, "skip source_unreachable - [] [a]"},
		{"all: two skip codes", all, []string{"a", "b"}, []SourceOutcome{skips(notIndexed), skips(notHeld)}, "skip sources_incomplete - [] [a b]"},
		{"all: passes without a candidate", all, []string{"a", "b"}, []SourceOutcome{passes(nil), passes(nil)}, "pass - - [a b] [a b]"},
	} {
		combined, asked := combine(c.policy, c.names, c.outcomes...)
		if got := summary(combined, asked); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
		if row := combined.Row("CHAIN", Pointer{Name("commitments"), Name("stellar")}, "read"); row.Code != combined.Code ||
			row.Status != combined.Status || *row.Part != "/commitments/stellar" {
			t.Errorf("%s: the row %+v", c.name, row)
		}
	}
}

// With one source, every policy gives that source's outcome, its status,
// code and candidate, whatever the outcome (SPEC.md section 10.4, and
// section 13.2: one named source behaves as no sources), and its row's
// values differ only by their policy. Mutant killed: at_least and all
// with one source giving sources_incomplete for a skip (rule 4 compared
// with no other source), or the candidate of no source.
func TestCombineSources_OneSourceUnderEachPolicyIsThatSource(t *testing.T) {
	outcomes := []SourceOutcome{passes(at(100)), passes(nil), fails, {Status: StatusFail, Code: specdata.ReasonEntropyMismatch}}
	for _, code := range inconclusiveCodes {
		outcomes = append(outcomes, skips(code))
	}
	for i, outcome := range outcomes {
		for _, names := range [][]string{{"a"}, {"down"}} {
			if names[0] == "down" && i > 0 {
				continue
			}
			var values []string
			for _, policy := range []SourcePolicy{first, atLeast(1), all} {
				combined, _ := combine(policy, names, outcome)
				want := outcome
				if names[0] == "down" {
					want = skips(specdata.ReasonSourceUnreachable)
				}
				if combined.Status != want.Status || combined.Code != want.Code || combined.Candidate != want.Candidate ||
					combined.Required != 1 || len(combined.Asked) != 1 {
					t.Errorf("%v under %s: %+v", names, policy, combined)
				}
				encoded, err := jcs.Encode(combined.Values())
				if err != nil {
					t.Fatal(err)
				}
				values = append(values, strings.Replace(string(encoded), `"policy":"`+string(policy.Kind)+`"`, `"policy":"?"`, 1))
			}
			if values[0] != values[1] || values[1] != values[2] {
				t.Errorf("%v %+v: values differ beyond the policy: %q", names, outcome, values)
			}
		}
	}
}

// Each source an edge rests on carries its place in its network's order,
// from 1, and the number of that network's sources, from the source the
// row counted (SPEC.md section 11.5): of three sources under at_least 2,
// a first that does not hold the read and two passes, the edge rests on
// the second and third, places 2 and 3 of 3; under first, of such a first
// and a pass, on the second, place 2 of 2. Mutants killed: CombineSources
// numbering places from 0; edgeOf taking the count as the number of
// sources counted (2 and 1); edgeOf leaving the place and count unset.
func TestEdgeOf_EachSourceItsPlaceInItsNetwork(t *testing.T) {
	cases := []struct {
		policy   SourcePolicy
		names    []string
		outcomes []SourceOutcome
		want     string
	}{
		{atLeast(2), []string{"one", "two", "three"}, []SourceOutcome{skips(specdata.ReasonNotHeldBySource), passes(at(100)), passes(at(100))}, "two 2/3 three 3/3"},
		{first, []string{"one", "two"}, []SourceOutcome{skips(specdata.ReasonNotHeldBySource), passes(at(100))}, "two 2/2"},
	}
	for _, c := range cases {
		combined, _ := combine(c.policy, c.names, c.outcomes...)
		if combined.Status != StatusPass {
			t.Fatalf("%s: %+v", c.policy, combined)
		}
		edge := edgeOf(RowCandidate{Step: "CHAIN", Part: "/commitments/stellar", Candidate: *combined.Candidate, Counted: combined.Counted}, edgeSubmittedBefore)
		var got []string
		for _, s := range edge.Sources {
			got = append(got, fmt.Sprintf("%s %d/%d", strings.TrimSuffix(strings.TrimPrefix(s.Source, "https://"), ".example"), s.Place, s.Count))
		}
		if strings.Join(got, " ") != c.want {
			t.Errorf("%s: the edge rests on %q, want %q", c.policy, strings.Join(got, " "), c.want)
		}
	}
}

// What a row read (SPEC.md section 10.4): policy, required, one object
// for each source the row asked, in the network's order, and the row's
// candidate when it passed with one; each object {cached, candidate, code,
// fetched_at, reads, source, status}, its code absent exactly when it
// passed, its candidate present exactly when it passed with one, its
// cached false, its fetched_at that of its candidate read when it has a
// candidate, of its identity read when it was asked nothing else, and
// otherwise of its last read; its reads each {answer, cached, fetched_at,
// read} in the form of a replay record; its source printed as scheme and
// host; and a rule's notes by source. Under first, a source after the one
// the row took has no object; under at_least 2 and all, every source has
// one, in the network's order (c, a, b), which is neither the order of
// their statuses nor of their URLs. A candidate's instant is written in
// its class, a NIST candidate's with its milliseconds, at the row and at
// its source. Mutants killed: fetched_at from the first read (the
// not-found source's is its first read's); from the identity read whatever
// was read (the candidate's is the identity's); a code on a pass (the pass
// gains "code":""); Required giving the number of sources under first
// (required is 4); a source's whole URL printed (its path appears); cached
// true; the notes left out; CombineSources under at_least and all listing
// Asked as the passes, then the failures, then the skips (a, b, c under
// at_least 2, c, b, a under all); Asked sorted by URL, descending (c, b,
// a); AskedSource.value without its candidate case, so a candidate's
// outcome takes its last read's fetched_at (09:00:03.000Z, not its
// candidate read's 09:00:02.000Z); Candidate.value writing every instant
// as a TIME (the NIST instant, read as seconds, is out of range and
// written "").
func TestCombinedOutcome_Values(t *testing.T) {
	read := func(kind string, outcome ReadOutcome, value map[string]any, offset time.Duration) SourceRead {
		r := SourceRead{Read: map[string]any{"kind": kind, "network": "testnet"}, Outcome: outcome, Value: value, FetchedAt: identityAt.Add(offset)}
		if outcome == ReadUnreachable {
			r.Reason = "HTTP 429"
		}
		return r
	}
	candidate := &Candidate{Basis: "stellar_ledger_close_read", Class: ClassTIME, Instant: 1790387535, FetchedAt: identityAt.Add(3 * time.Second)}
	notFound := SourceOutcome{Status: StatusSkip, Code: specdata.ReasonNotHeldBySource,
		Reads: []SourceRead{read("stellar_transaction", ReadNotFound, nil, time.Second), read("stellar_root", ReadFound, map[string]any{"elder": "x"}, 2*time.Second)},
		Notes: map[string]any{"elder": "x"}}
	found := SourceOutcome{Status: StatusPass, Candidate: candidate,
		Reads: []SourceRead{read("stellar_transaction", ReadUnreachable, nil, 2*time.Second), read("stellar_ledger", ReadFound, map[string]any{"closed_at": "2026-09-26T01:52:15Z"}, 3*time.Second)}}
	set := standInSet(first, "down", "held", "b", "c")
	set.Sources[1].Source.URL = "https://held.example/api/secret-key"
	s := &standIn{outcomes: map[string]SourceOutcome{"https://held.example/api/secret-key": notFound, "https://b.example": found, "https://c.example": fails}}
	encoded, err := jcs.Encode(CombineSources(set, s.rule).Values())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"candidate":{"basis":"stellar_ledger_close_read","instant":"2026-09-26T01:52:15Z"},"notes":[{"elder":"x","source":"https://held.example"}],` +
		`"outcomes":[` +
		`{"cached":false,"code":"source_unreachable","fetched_at":"2026-10-20T09:00:00.000Z","reads":[],"source":"https://down.example","status":"skip"},` +
		`{"cached":false,"code":"not_held_by_source","fetched_at":"2026-10-20T09:00:02.000Z","reads":[` +
		`{"answer":{"outcome":"not_found"},"cached":false,"fetched_at":"2026-10-20T09:00:01.000Z","read":{"kind":"stellar_transaction","network":"testnet"}},` +
		`{"answer":{"outcome":"found","value":{"elder":"x"}},"cached":false,"fetched_at":"2026-10-20T09:00:02.000Z","read":{"kind":"stellar_root","network":"testnet"}}],` +
		`"source":"https://held.example","status":"skip"},` +
		`{"cached":false,"candidate":{"basis":"stellar_ledger_close_read","instant":"2026-09-26T01:52:15Z"},"fetched_at":"2026-10-20T09:00:03.000Z","reads":[` +
		`{"answer":{"outcome":"unreachable","reason":"HTTP 429"},"cached":false,"fetched_at":"2026-10-20T09:00:02.000Z","read":{"kind":"stellar_transaction","network":"testnet"}},` +
		`{"answer":{"outcome":"found","value":{"closed_at":"2026-09-26T01:52:15Z"}},"cached":false,"fetched_at":"2026-10-20T09:00:03.000Z","read":{"kind":"stellar_ledger","network":"testnet"}}],` +
		`"source":"https://b.example","status":"pass"}],` +
		`"policy":"first","required":1}`
	if string(encoded) != want || slices.Contains(s.asked, "https://c.example") {
		t.Fatalf("values %s\nwant   %s\nasked %q", encoded, want, s.asked)
	}

	set = standInSet(atLeast(2), "a", "b", "c")
	s = &standIn{outcomes: map[string]SourceOutcome{"https://a.example": skips(specdata.ReasonNotYetIndexed), "https://b.example": skips(specdata.ReasonNotYetIndexed), "https://c.example": skips(specdata.ReasonNotYetIndexed)}}
	combined := CombineSources(set, s.rule)
	values := combined.Values()
	if _, ok := values["candidate"]; ok || values["policy"] != "at_least" || fmt.Sprint(values["required"]) != "2" ||
		combined.Row("SOURCES", Pointer{Name("subject"), Name("entropy")}, "read").Code != specdata.ReasonNotYetIndexed {
		t.Fatalf("at_least 2: %v", values)
	}
	set = standInSet(all, "a", "b", "c")
	s = &standIn{outcomes: map[string]SourceOutcome{"https://a.example": fails, "https://b.example": fails, "https://c.example": fails}}
	if values := CombineSources(set, s.rule).Values(); values["policy"] != "all" || fmt.Sprint(values["required"]) != "3" {
		t.Fatalf("all: %v", values)
	}

	unread := func(name, status, code string) string {
		if code != "" {
			code = `"code":"` + code + `",`
		}
		return `{"cached":false,` + code + `"fetched_at":"2026-10-20T09:00:00.000Z","reads":[],"source":"https://` + name + `.example","status":"` + status + `"}`
	}
	notIndexed := skips(specdata.ReasonNotYetIndexed)
	stellarCandidate := &Candidate{Basis: "stellar_ledger_close_read", Class: ClassTIME, Instant: 1790387535, FetchedAt: identityAt.Add(2 * time.Second)}
	nistCandidate := &Candidate{Basis: "nist_pulse_time_read", Class: ClassNISTTIME, Instant: 1790598896789, FetchedAt: identityAt.Add(time.Minute)}
	outputValue := strings.Repeat("AB", 64)
	for _, c := range []struct {
		name     string
		policy   SourcePolicy
		names    []string
		outcomes []SourceOutcome
		want     string
	}{
		{"at_least 2: the network's order", atLeast(2), []string{"c", "a", "b"}, []SourceOutcome{notIndexed, passes(nil), fails},
			`{"outcomes":[` + unread("c", "skip", "not_yet_indexed") + `,` + unread("a", "pass", "") + `,` + unread("b", "fail", "chain_mismatch") + `],"policy":"at_least","required":2}`},
		{"all: the network's order", all, []string{"c", "a", "b"}, []SourceOutcome{passes(nil), notIndexed, passes(nil)},
			`{"outcomes":[` + unread("c", "pass", "") + `,` + unread("a", "skip", "not_yet_indexed") + `,` + unread("b", "pass", "") + `],"policy":"all","required":3}`},
		{"a read after the candidate read", first, []string{"a"}, []SourceOutcome{{Status: StatusPass, Candidate: stellarCandidate, Reads: []SourceRead{
			read("stellar_ledger", ReadFound, map[string]any{"closed_at": "2026-09-26T01:52:15Z"}, 2*time.Second),
			read("stellar_root", ReadFound, map[string]any{"latest": strictjson.Number(9)}, 3*time.Second)}}},
			`{"candidate":{"basis":"stellar_ledger_close_read","instant":"2026-09-26T01:52:15Z"},"outcomes":[` +
				`{"cached":false,"candidate":{"basis":"stellar_ledger_close_read","instant":"2026-09-26T01:52:15Z"},"fetched_at":"2026-10-20T09:00:02.000Z","reads":[` +
				`{"answer":{"outcome":"found","value":{"closed_at":"2026-09-26T01:52:15Z"}},"cached":false,"fetched_at":"2026-10-20T09:00:02.000Z","read":{"kind":"stellar_ledger","network":"testnet"}},` +
				`{"answer":{"outcome":"found","value":{"latest":9}},"cached":false,"fetched_at":"2026-10-20T09:00:03.000Z","read":{"kind":"stellar_root","network":"testnet"}}],` +
				`"source":"https://a.example","status":"pass"}],"policy":"first","required":1}`},
		{"a NIST candidate", first, []string{"beacon"}, []SourceOutcome{{Status: StatusPass, Candidate: nistCandidate, Reads: []SourceRead{{
			Read:    map[string]any{"chainIndex": strictjson.Number(2), "kind": "nist_pulse", "pulseIndex": strictjson.Number(1234)},
			Outcome: ReadFound, Value: map[string]any{"outputValue": outputValue, "timeStamp": "2026-09-28T12:34:56.789Z"}, FetchedAt: identityAt.Add(time.Minute)}}}},
			`{"candidate":{"basis":"nist_pulse_time_read","instant":"2026-09-28T12:34:56.789Z"},"outcomes":[` +
				`{"cached":false,"candidate":{"basis":"nist_pulse_time_read","instant":"2026-09-28T12:34:56.789Z"},"fetched_at":"2026-10-20T09:01:00.000Z","reads":[` +
				`{"answer":{"outcome":"found","value":{"outputValue":"` + outputValue + `","timeStamp":"2026-09-28T12:34:56.789Z"}},"cached":false,"fetched_at":"2026-10-20T09:01:00.000Z","read":{"chainIndex":2,"kind":"nist_pulse","pulseIndex":1234}}],` +
				`"source":"https://beacon.example","status":"pass"}],"policy":"first","required":1}`},
	} {
		combined, _ := combine(c.policy, c.names, c.outcomes...)
		encoded, err := jcs.Encode(combined.Values())
		if err != nil {
			t.Fatal(err)
		}
		if string(encoded) != c.want {
			t.Errorf("%s: values %s\nwant   %s", c.name, encoded, c.want)
		}
	}
}

// A rule's outcome is held to SPEC.md section 10.4's: a pass with no code,
// a failure of chain_mismatch or entropy_mismatch, a skip of one of the
// four inconclusive codes, a candidate only on a pass; any other is a
// defect, and so is a set with no source, whose panic names the network.
// Mutants killed: check accepting a skip sources_disagree (a combined code
// a source cannot give), a pass with a code, a candidate on a failure, or
// a failure of not_held_by_source; CombineSources without its refusal of
// no source (it fails on an index instead).
func TestCombineSources_RefusesAnOutcomeNoRuleGives(t *testing.T) {
	panics := func(name, says string, f func()) {
		t.Helper()
		defer func() {
			if r, ok := recover().(string); !ok || !strings.Contains(r, says) {
				t.Errorf("%s: recovered %v", name, r)
			}
		}()
		f()
	}
	for name, outcome := range map[string]SourceOutcome{
		"a skip of sources_disagree":         skips(specdata.ReasonSourcesDisagree),
		"a pass with a code":                 {Status: StatusPass, Code: specdata.ReasonChainMismatch},
		"a failure with a candidate":         {Status: StatusFail, Code: specdata.ReasonChainMismatch, Candidate: at(1)},
		"a failure of not_held_by_source":    {Status: StatusFail, Code: specdata.ReasonNotHeldBySource},
		"an info outcome":                    {Status: StatusInfo, Code: specdata.ReasonNotAttributable},
		"a skip with no code":                {Status: StatusSkip},
		"a skip of prior_failure":            skips(specdata.ReasonPriorFailure),
		"a pass with a candidate and a code": {Status: StatusPass, Code: specdata.ReasonEntropyMismatch, Candidate: at(1)},
	} {
		panics(name, "is none SPEC.md section 10.4 gives", func() { combine(first, []string{"a"}, outcome) })
	}
	panics("no source", "Bitcoin's regtest network has no source", func() {
		CombineSources(SourceSet{Chain: ChainBitcoin, Network: "regtest", Policy: first}, (&standIn{}).rule)
	})
}

// One answer per read and source in a run (SPEC.md section 10.4,
// "Caches"): a second request of a read a source answered found or not
// found takes the first answer, whatever the order of the read's members;
// another source's read, or another read, is made; an unreachable answer
// is never kept, so the read is made again. Mutants killed: the key
// without the source's URL (the second source takes the first's answer);
// the key without the network (the other network takes it); an unreachable
// answer kept (the root's second request is not made); no answer kept
// (the reordered read is made again).
func TestRun_ReadSourceGivesOneAnswerPerReadAndSource(t *testing.T) {
	run := &Run{}
	set := standInSet(first, "a", "b")
	a, b := set.Sources[0], set.Sources[1]
	other := a
	other.Source.Network = "public"
	made := 0
	answer := func(outcome ReadOutcome) func() SourceRead {
		return func() SourceRead {
			made++
			return SourceRead{Outcome: outcome}
		}
	}
	tx := map[string]any{"hash": strings.Repeat("ab", 32), "kind": "stellar_transaction", "network": "testnet"}
	reordered := map[string]any{"network": "testnet", "kind": "stellar_transaction", "hash": strings.Repeat("ab", 32)}
	steps := []struct {
		source SourceRecord
		read   map[string]any
		answer ReadOutcome
		made   int
	}{
		{a, tx, ReadNotFound, 1},
		{a, reordered, ReadFound, 1},
		{b, tx, ReadFound, 2},
		{other, tx, ReadFound, 3},
		{a, map[string]any{"kind": "stellar_root", "network": "testnet"}, ReadUnreachable, 4},
		{a, map[string]any{"kind": "stellar_root", "network": "testnet"}, ReadFound, 5},
		{a, map[string]any{"kind": "stellar_root", "network": "testnet"}, ReadUnreachable, 5},
	}
	for i, step := range steps {
		got := run.ReadSource(step.source, step.read, answer(step.answer))
		if made != step.made {
			t.Fatalf("request %d: %d reads made, want %d", i, made, step.made)
		}
		if i == 1 && got.Outcome != ReadNotFound {
			t.Fatalf("request %d: %+v, want the first answer", i, got)
		}
	}
}

// ReadSource sets the read's name and its fetched_at itself, so a rule's
// do gives only the answer: the name is the read asked for and fetched_at
// the clock's reading when do returns, for a kept answer, an unreachable
// one and a read whose name JCS cannot write alike; a kept answer keeps
// its first fetched_at; and the row's values show both. Mutants killed:
// ReadSource not setting Read (the read is null); not setting FetchedAt
// (0001-01-01T00:00:00.000Z); setting them only on the kept path (the
// unwritable read is unnamed and unstamped); the clock read before do,
// so a do that reads the clock first takes the earlier reading; a kept
// answer stamped again on each request (the second request's time).
func TestRun_ReadSourceSetsTheReadAndItsFetchedAt(t *testing.T) {
	t1, t2, t3, t4 := identityAt.Add(time.Second), identityAt.Add(2*time.Second), identityAt.Add(3*time.Second), identityAt.Add(4*time.Second)
	useClock(t, t1, t2, t3, t4)
	run := &Run{}
	ledger := map[string]any{"kind": "stellar_ledger", "network": "testnet", "sequence": strictjson.Number(5)}
	root := map[string]any{"kind": "stellar_root", "network": "testnet"}
	unwritable := map[string]any{"kind": "stellar_ledger", "network": "testnet", "sequence": 5}
	answer := func(outcome ReadOutcome, value map[string]any) func() SourceRead {
		return func() SourceRead {
			readClock()
			r := SourceRead{Outcome: outcome, Value: value}
			if outcome == ReadUnreachable {
				r.Reason = "HTTP 429"
			}
			return r
		}
	}
	rule := func(record SourceRecord) SourceOutcome {
		found := run.ReadSource(record, ledger, answer(ReadFound, map[string]any{"closed_at": "2026-09-26T01:52:15Z"}))
		again := run.ReadSource(record, ledger, answer(ReadNotFound, nil))
		down := run.ReadSource(record, root, answer(ReadUnreachable, nil))
		unkept := run.ReadSource(record, unwritable, answer(ReadNotFound, nil))
		for i, c := range []struct {
			got  SourceRead
			read map[string]any
			at   time.Time
		}{{found, ledger, t2}, {again, ledger, t2}, {down, root, t4}, {unkept, unwritable, t4}} {
			if !reflect.DeepEqual(c.got.Read, c.read) || !c.got.FetchedAt.Equal(c.at) {
				t.Errorf("request %d: read %v at %v, want %v at %v", i, c.got.Read, c.got.FetchedAt, c.read, c.at)
			}
		}
		return SourceOutcome{Status: StatusSkip, Code: specdata.ReasonSourceUnreachable, Reads: []SourceRead{found, down}}
	}
	encoded, err := jcs.Encode(CombineSources(standInSet(first, "a"), rule).Values())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"outcomes":[{"cached":false,"code":"source_unreachable","fetched_at":"2026-10-20T09:00:04.000Z","reads":[` +
		`{"answer":{"outcome":"found","value":{"closed_at":"2026-09-26T01:52:15Z"}},"cached":false,"fetched_at":"2026-10-20T09:00:02.000Z","read":{"kind":"stellar_ledger","network":"testnet","sequence":5}},` +
		`{"answer":{"outcome":"unreachable","reason":"HTTP 429"},"cached":false,"fetched_at":"2026-10-20T09:00:04.000Z","read":{"kind":"stellar_root","network":"testnet"}}],` +
		`"source":"https://a.example","status":"skip"}],"policy":"first","required":1}`
	if string(encoded) != want {
		t.Fatalf("values %s\nwant   %s", encoded, want)
	}
}
