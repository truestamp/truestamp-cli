// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The online cases of SPEC.md section 13.2 that read a Bitcoin commitment
// entry or a genesis.bitcoin, and those of KEY_EVENT_COMMITMENT, through
// the runner (runner_test.go): CHAIN's Bitcoin rows and its key-event
// row, the reads the rule made of each source, and the candidate each
// passing /commitments/bitcoin row keeps (SPEC.md sections 10.3, the
// key-history rows, 10.4, "Bitcoin commitment entries", "genesis.bitcoin"
// and "Several sources", and 11.1).

package vectorcases_test

import (
	"encoding/json/v2"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// bitcoinCases are the cases this test holds: each reads-bitcoin-* case,
// reads-unreachable-bitcoin_tip and -bitcoin_block_at_height,
// reads-identity-bitcoin-unreachable, each reads-key_event_commitment-*
// case and every sources-* case that names several Bitcoin sources.
func bitcoinCases(c vectorcases.Case) bool {
	id := c.ID
	if _, several := c.Config.Sources["bitcoin"]; several {
		return true
	}
	return strings.HasPrefix(id, "reads-bitcoin-") || id == "reads-unreachable-bitcoin_tip" || id == "reads-unreachable-bitcoin_block_at_height" ||
		id == "reads-identity-bitcoin-unreachable" || strings.HasPrefix(id, "reads-key_event_commitment-")
}

// bitcoinRow reports whether row is a CHAIN row of a Bitcoin commitment
// entry or a genesis.bitcoin, or KEY_EVENT_COMMITMENT.
func bitcoinRow(row vectorcases.Row) bool {
	return row.Step == "CHAIN" && row.Part != nil && (strings.HasSuffix(*row.Part, "/bitcoin") || *row.Part == "/signing_key_event")
}

// heightValue is a height a proof or a read holds, as the strict reader
// gives it.
func heightValue(v any) int64 {
	number, _ := v.(strictjson.Number)
	n, _ := number.Int64()
	return n
}

// Each such case's Bitcoin rows and KEY_EVENT_COMMITMENT are exactly its
// expected ones, through the replay adapter, and its reads are those of
// the rule: every Bitcoin read is the tip or the best-chain block at a
// height, never the block by a hash the proof carries; a source's blocks
// are read only after its tip, and its tip, when it answered, at most once
// in the run, so the rows of one source take one tip answer (an
// unreachable answer is kept for no row, SPEC.md section 10.4, "Answers",
// so reads-unreachable-bitcoin_tip reads it once for each row); every
// block read is at the height of a Bitcoin entry or a genesis.bitcoin, or
// 10 above the top-level entry's, for its candidate. A /commitments/bitcoin row that
// passes keeps one candidate on the run, basis bitcoin_median_time_read,
// at the median_time_past of the block 10 above the entry's height that
// each source it rests on gave, its sources the one whose outcome the row
// took under first and every source that passed under all; where the
// case's submitted-before edge is a Bitcoin median time past, the
// candidate is that edge, and no Bitcoin candidate is earlier than the
// edge; a row that does not pass keeps none. The messages of the two
// not-found cases name the read that settled them. Mutants killed: the
// block read by the header_hash the proof carries (a bitcoin_block read);
// the tip read again for each row (a source's tip read twice); the
// candidate read for every entry (the key event's entry of the block
// proof of reads-bitcoin-genesis_hash_differs, at height 222 beside the
// top-level entry's 218, reads a block at height 232); the candidate
// taken from the block at the entry's height (the instant is not the
// edge); Counted holding the first source alone (sources-all-agree's
// candidate rests on one source); the candidate's not-found message
// chosen from its code alone (reads-bitcoin-candidate_not_found says
// nothing of the block at the entry's height matching).
func TestCases_TheBitcoinCommitmentCases(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	held, keyEvents, messaged := 0, 0, 0
	for _, c := range cases {
		if c.Config.Online == nil || !bitcoinCases(c) || c.Expected.Bare() {
			continue
		}
		held++
		if strings.HasPrefix(c.ID, "reads-key_event_commitment-") {
			keyEvents++
		}
		t.Run(c.ID, func(t *testing.T) {
			got, replay := runCase(t, c)
			want := slices.DeleteFunc(slices.Clone(c.Expected.Rows), func(row vectorcases.Row) bool { return !bitcoinRow(row) })
			gotRows := slices.DeleteFunc(slices.Clone(got.rows), func(row vectorcases.Row) bool { return !bitcoinRow(row) })
			if len(want) == 0 || !rowsEqual(gotRows, want) {
				t.Fatalf("Bitcoin rows %v, want %v", gotRows, want)
			}
			proof := proofOf(t, c)
			heights := map[int64]bool{}
			for _, part := range []string{"/commitments/bitcoin", "/signing_key_event/commitments/bitcoin", "/block/record/metadata/genesis/bitcoin",
				"/signing_key_event/block/record/metadata/genesis/bitcoin", "/witnesses/block/record/metadata/genesis/bitcoin"} {
				if h, ok := valueAtPart(proof, part+"/height").(strictjson.Number); ok {
					heights[heightValue(h)] = true
				}
			}
			if h, ok := valueAtPart(proof, "/commitments/bitcoin/height").(strictjson.Number); ok {
				heights[heightValue(h)+10] = true
			}
			file, err := vectorcases.LoadCaseReplay(c)
			if err != nil {
				t.Fatal(err)
			}
			tips, kept := map[string]int{}, map[string]int{}
			for _, r := range chainReadsOf(replay) {
				switch kind, _ := r.read["kind"].(string); kind {
				case "bitcoin_tip":
					tips[r.source]++
					if answer, _ := file.Answer(r.read, r.source); answer.Outcome != vectorcases.OutcomeUnreachable {
						if kept[r.source]++; kept[r.source] > 1 {
							t.Errorf("the tip of source %q read %d times", r.source, kept[r.source])
						}
					}
				case "bitcoin_block_at_height":
					if tips[r.source] == 0 {
						t.Errorf("a block of source %q read before its tip", r.source)
					}
					if h := heightValue(r.read["height"]); !heights[h] {
						t.Errorf("a block read at height %d, none of %v", h, heights)
					}
				default:
					if strings.HasPrefix(kind, "bitcoin_") {
						t.Errorf("a read of kind %s", kind)
					}
				}
			}
			top := heightValue(valueAtPart(proof, "/commitments/bitcoin/height"))
			says := map[string]string{
				"reads-bitcoin-candidate_not_found": fmt.Sprintf("holds the best-chain block at height %d, which matches this proof, but no best-chain block at height %d.", top, top+10),
				"reads-bitcoin-block_not_found":     fmt.Sprintf("holds no best-chain block at height %d, though its tip is past it.", top),
			}
			if want, ok := says[c.ID]; ok {
				messaged++
				if m := rowAtPart(t, got.run, "/commitments/bitcoin").Message; !strings.Contains(m, want) {
					t.Errorf("the message %q", m)
				}
			}
			checkBitcoinCandidate(t, c, got.run, replay)
		})
	}
	if held != 21 || keyEvents != 5 || messaged != 2 {
		t.Fatalf("%d Bitcoin cases, want 21; %d key-event cases, want 5; %d message cases, want 2", held, keyEvents, messaged)
	}
}

// rowAtPart is the run's CHAIN row at part.
func rowAtPart(t *testing.T, r *verifier.Run, part string) verifier.Row {
	t.Helper()
	for _, row := range r.Rows {
		if row.Step == "CHAIN" && row.Part != nil && *row.Part == part {
			return row
		}
	}
	t.Fatalf("no CHAIN row at %s", part)
	return verifier.Row{}
}

// checkBitcoinCandidate holds the Bitcoin candidate a case's run keeps
// (see TestCases_TheBitcoinCommitmentCases).
func checkBitcoinCandidate(t *testing.T, c vectorcases.Case, r *verifier.Run, replay *caseReplay) {
	t.Helper()
	file, err := vectorcases.LoadCaseReplay(c)
	if err != nil {
		t.Fatal(err)
	}
	var kept []verifier.RowCandidate
	for _, candidate := range r.Candidates {
		if candidate.Part == "/commitments/bitcoin" {
			kept = append(kept, candidate)
		}
	}
	passed := false
	for _, row := range c.Expected.Rows {
		if row.Step == "CHAIN" && row.Part != nil && *row.Part == "/commitments/bitcoin" {
			passed = row.Status == "pass"
		}
	}
	if !passed {
		if len(kept) != 0 {
			t.Errorf("candidates %+v of a row that did not pass", kept)
		}
		return
	}
	if len(kept) != 1 {
		t.Fatalf("Bitcoin candidates %+v, want one", kept)
	}
	candidate := kept[0]
	network := c.Config.Networks.Commitment["bitcoin"]
	final := strictjson.Number(float64(heightValue(valueAtPart(proofOf(t, c), "/commitments/bitcoin/height")) + 10))
	for i, counted := range candidate.Counted {
		name := replay.names[counted.Source.Source.URL]
		answer, ok := file.Answer(map[string]any{"kind": "bitcoin_block_at_height", "network": network, "height": final}, name)
		if !ok || heightValue(answer.Value["median_time_past"]) != candidate.Candidate.Instant || counted.Outcome.Status != verifier.StatusPass {
			t.Errorf("counted source %d %q: %+v, the block at %v: %+v", i, name, counted.Outcome, final, answer)
		}
	}
	passes := 0
	for _, outcome := range rowOutcomes(t, r, "/commitments/bitcoin") {
		if outcome == "pass" {
			passes++
		}
	}
	policy := c.Config.Sources["bitcoin"][network].Policy
	if wantCounted := map[bool]int{true: 1, false: passes}[policy == "" || policy == vectorcases.PolicyFirst]; len(candidate.Counted) != wantCounted {
		t.Errorf("the candidate rests on %d sources, want %d", len(candidate.Counted), wantCounted)
	}
	if candidate.Step != "CHAIN" || candidate.Candidate.Basis != "bitcoin_median_time_read" || candidate.Candidate.Class != verifier.ClassTIME {
		t.Errorf("candidate %+v", candidate)
	}
	if string(c.Expected.SubmittedBefore) == "null" && c.Expected.Label == "failed" {
		return
	}
	var edge struct {
		Basis   string `json:"basis"`
		Instant string `json:"instant"`
	}
	if err := json.Unmarshal(c.Expected.SubmittedBefore, &edge); err != nil || edge.Instant == "" {
		t.Fatalf("the case's submitted-before edge %s: %v", c.Expected.SubmittedBefore, err)
	}
	at, err := verifier.ParseTime(verifier.ClassTIME, edge.Instant)
	if err != nil {
		t.Fatal(err)
	}
	if (edge.Basis == "bitcoin_median_time_read" && candidate.Candidate.Instant != at) || candidate.Candidate.Instant < at {
		t.Errorf("the candidate %d against the edge %+v", candidate.Candidate.Instant, edge)
	}
}
