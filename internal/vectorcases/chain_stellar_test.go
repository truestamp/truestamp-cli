// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The online cases of SPEC.md section 13.2 that read a Stellar commitment
// entry, through the runner (runner_test.go): CHAIN's rows of the Stellar
// entries, the reads the rule made of each source, and the candidate each
// passing top-level row keeps (SPEC.md sections 10.4, "Stellar commitment
// entries" and "Several sources", and 11.1).

package vectorcases_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// stellarCases are the cases whose Stellar rows this test holds: each
// reads-stellar-* case, each reads-unreachable-stellar_* case,
// reads-identity-stellar-unreachable, both leaf_count-record-*-stellar-online
// cases, gates-online-chain_failure and every sources-* case that names
// several Stellar sources.
func stellarCases(c vectorcases.Case) bool {
	id := c.ID
	if _, several := c.Config.Sources["stellar"]; several {
		return true
	}
	return strings.HasPrefix(id, "reads-stellar-") || strings.HasPrefix(id, "reads-unreachable-stellar_") ||
		id == "reads-identity-stellar-unreachable" || id == "gates-online-chain_failure" ||
		(strings.HasPrefix(id, "leaf_count-record-") && strings.HasSuffix(id, "-stellar-online"))
}

// chainRead is one chain read the adapter answered: its read and the
// source's name.
type chainRead struct {
	read   map[string]any
	source string
}

// chainReadsOf returns the reads of CHAIN the adapter answered, in the
// order made: those made before SOURCES began.
func chainReadsOf(replay *caseReplay) []chainRead {
	replay.mu.Lock()
	defer replay.mu.Unlock()
	return slices.Clone(replay.chainReads[:replay.chainEnd])
}

// sourcesReadsOf returns the reads of SOURCES the adapter answered, in the
// order made: those made after CHAIN ended.
func sourcesReadsOf(replay *caseReplay) []chainRead {
	replay.mu.Lock()
	defer replay.mu.Unlock()
	return slices.Clone(replay.chainReads[replay.chainEnd:])
}

// Each Stellar case's rows of its Stellar commitment entries are exactly
// its expected ones, through the replay adapter; every chain read the run
// makes is a Stellar read the case's replay file holds, or one of the two
// Bitcoin reads of CHAIN, of a source whose identity it confirmed, and the
// root of a source is read only after a
// transaction that source does not hold, and read at most once in the run
// when it answered, so two rows whose transactions a source does not hold
// take one root answer (sources-first-past_not_held's source a; an
// unreachable answer is kept for no row, SPEC.md section 10.4, "Answers");
// where only the read of ledger L settled the top-level row, its message
// says the transaction matches this proof (reads-stellar-ledger_not_found
// and reads-unreachable-stellar_ledger); and each passing
// /commitments/stellar row keeps a candidate on the run, basis
// stellar_ledger_close_read, at the closed_at of ledger L that each source
// it rests on gave, its sources the one whose outcome the row took under
// first and every source that passed under at_least n or all. Where the
// case's submitted-before edge is a Stellar ledger's close, the candidate
// is that edge, and no Stellar candidate is earlier than the edge; a
// failed case has no edge to hold it to. Mutants killed: the root read
// before the transaction (a root read with no not-found transaction
// before it); the candidate's instant taken from ledger L's sequence (the
// instant differs from the edge); Counted holding the first source alone
// (sources-all-agree's candidate rests on one source); the candidate read
// for every entry (a /signing_key_event entry reads a ledger the replay
// does not hold, and its row skips); the root read again for each row (a
// second root read of sources-first-past_not_held's source a); the
// message of a row not_held_by_source or source_unreachable chosen from
// its code alone (the ledger cases say nothing of the transaction
// matching); one failing source taken as the row's outcome under all
// (sources-all-fail_and_unreachable's row fails chain_mismatch, beside a
// source that is unreachable, where it skips sources_incomplete).
func TestCases_TheStellarCommitmentCases(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	held, messaged := 0, 0
	for _, c := range cases {
		if c.Config.Online == nil || !stellarCases(c) || c.Expected.Bare() {
			continue
		}
		held++
		t.Run(c.ID, func(t *testing.T) {
			got, replay := runCase(t, c)
			want := slices.DeleteFunc(slices.Clone(c.Expected.Rows), func(row vectorcases.Row) bool { return row.Step != "CHAIN" || !stellarEntry(row.Part) })
			gotRows := slices.DeleteFunc(slices.Clone(got.rows), func(row vectorcases.Row) bool { return row.Step != "CHAIN" || !stellarEntry(row.Part) })
			if len(want) == 0 || !rowsEqual(gotRows, want) {
				t.Fatalf("Stellar rows %v, want %v", gotRows, want)
			}
			file, err := vectorcases.LoadCaseReplay(c)
			if err != nil {
				t.Fatal(err)
			}
			missing := map[string]bool{}
			roots := map[string]int{}
			for _, r := range chainReadsOf(replay) {
				kind, _ := r.read["kind"].(string)
				if !strings.HasPrefix(kind, "stellar_") {
					if kind != "bitcoin_tip" && kind != "bitcoin_block_at_height" {
						t.Errorf("a read of kind %s", kind)
					}
					continue
				}
				if answer, _ := file.Answer(r.read, r.source); kind == "stellar_transaction" && answer.Outcome == vectorcases.OutcomeNotFound {
					missing[r.source] = true
				}
				if kind == "stellar_root" && !missing[r.source] {
					t.Errorf("the root of source %q read with no transaction not found before it", r.source)
				}
				if answer, _ := file.Answer(r.read, r.source); kind == "stellar_root" && answer.Outcome != vectorcases.OutcomeUnreachable {
					if roots[r.source]++; roots[r.source] > 1 {
						t.Errorf("the root of source %q read %d times", r.source, roots[r.source])
					}
				}
			}
			if c.ID == "reads-stellar-ledger_not_found" || c.ID == "reads-unreachable-stellar_ledger" {
				messaged++
				if message := topStellarMessage(t, got.run); !strings.Contains(message, "matches this proof") {
					t.Errorf("the message %q", message)
				}
			}
			checkStellarCandidate(t, c, got.run, file, replay)
		})
	}
	if held != 35 || messaged != 2 {
		t.Fatalf("%d Stellar cases, want 35, and %d ledger cases, want 2", held, messaged)
	}
}

// topStellarMessage is the message of the run's /commitments/stellar row.
func topStellarMessage(t *testing.T, r *verifier.Run) string {
	t.Helper()
	for _, row := range r.Rows {
		if row.Step == "CHAIN" && row.Part != nil && *row.Part == "/commitments/stellar" {
			return row.Message
		}
	}
	t.Fatal("no CHAIN row at /commitments/stellar")
	return ""
}

// checkStellarCandidate holds the candidates a Stellar case's run keeps
// (see TestCases_TheStellarCommitmentCases).
func checkStellarCandidate(t *testing.T, c vectorcases.Case, r *verifier.Run, file *vectorcases.Replay, replay *caseReplay) {
	t.Helper()
	var top *vectorcases.Row
	for _, row := range c.Expected.Rows {
		if row.Step == "CHAIN" && row.Part != nil && *row.Part == "/commitments/stellar" {
			top = &row
		}
	}
	var kept []verifier.RowCandidate
	for _, candidate := range r.Candidates {
		if candidate.Step == "SOURCES" {
			continue
		}
		if candidate.Step != "CHAIN" || (candidate.Part != "/commitments/stellar" && candidate.Part != "/commitments/bitcoin") {
			t.Errorf("a candidate at %s %s", candidate.Step, candidate.Part)
		}
		if candidate.Part == "/commitments/stellar" {
			kept = append(kept, candidate)
		}
	}
	if top == nil || top.Status != "pass" {
		if len(kept) != 0 {
			t.Errorf("candidates %+v of a row that did not pass", kept)
		}
		return
	}
	if len(kept) != 1 {
		t.Fatalf("candidates %+v, want one", kept)
	}
	candidate := kept[0]
	proof := proofOf(t, c)
	ledger := valueAtPart(proof, "/commitments/stellar/ledger")
	policy := c.Config.Sources["stellar"][c.Config.Networks.Commitment["stellar"]].Policy
	for i, counted := range candidate.Counted {
		name := replay.names[counted.Source.Source.URL]
		answer, ok := file.Answer(map[string]any{"kind": "stellar_ledger", "network": c.Config.Networks.Commitment["stellar"], "sequence": ledger}, name)
		instant, err := verifier.ParseTime(verifier.ClassTIME, answer.Value["closed_at"])
		if !ok || err != nil || instant != candidate.Candidate.Instant || counted.Outcome.Status != verifier.StatusPass {
			t.Errorf("counted source %d %q: %+v, ledger %v closed at %v", i, name, counted.Outcome, ledger, answer.Value["closed_at"])
		}
	}
	passed := 0
	for _, outcome := range rowOutcomes(t, r, "/commitments/stellar") {
		if outcome == "pass" {
			passed++
		}
	}
	if wantCounted := map[bool]int{true: 1, false: passed}[policy == "" || policy == vectorcases.PolicyFirst]; len(candidate.Counted) != wantCounted {
		t.Errorf("the candidate rests on %d sources, want %d", len(candidate.Counted), wantCounted)
	}
	if candidate.Candidate.Basis != "stellar_ledger_close_read" || candidate.Candidate.Class != verifier.ClassTIME {
		t.Errorf("candidate %+v", candidate.Candidate)
	}
	if string(c.Expected.SubmittedBefore) == "null" && c.Expected.Label == "failed" {
		return
	}
	var edge struct{ Basis, Instant string }
	if err := json.Unmarshal(c.Expected.SubmittedBefore, &edge); err != nil || edge.Instant == "" {
		t.Fatalf("the case's submitted-before edge %s: %v", c.Expected.SubmittedBefore, err)
	}
	at, err := verifier.ParseTime(verifier.ClassTIME, edge.Instant)
	if err != nil {
		t.Fatal(err)
	}
	if (edge.Basis == "stellar_ledger_close_read" && candidate.Candidate.Instant != at) || candidate.Candidate.Instant < at {
		t.Errorf("the candidate %d against the edge %+v", candidate.Candidate.Instant, edge)
	}
}

// rowOutcomes returns the status of each source's outcome on the row at
// part, from its values.
func rowOutcomes(t *testing.T, r *verifier.Run, part string) []string {
	t.Helper()
	for _, row := range r.Rows {
		if row.Part == nil || *row.Part != part || row.Step != "CHAIN" {
			continue
		}
		outcomes, _ := row.Values["outcomes"].([]any)
		var out []string
		for _, o := range outcomes {
			status, _ := o.(map[string]any)["status"].(string)
			out = append(out, status)
		}
		return out
	}
	t.Fatalf("no CHAIN row at %s", part)
	return nil
}

// valueAtPart is the value of proof at part, an RFC 6901 pointer of names.
func valueAtPart(proof map[string]any, part string) any {
	var value any = proof
	for _, name := range strings.Split(part, "/")[1:] {
		object, _ := value.(map[string]any)
		value = object[name]
	}
	return value
}
