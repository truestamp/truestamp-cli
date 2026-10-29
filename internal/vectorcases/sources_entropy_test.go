// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The online cases of SPEC.md section 13.2 whose external gate opens and
// whose proof carries an entropy object, through the runner
// (runner_test.go): SOURCES' rows, the reads each rule made of each
// source, and the candidate each passing row keeps (SPEC.md sections 10.2,
// step 11, 10.4, "Entropy objects" and "Several sources", and 11.2).

package vectorcases_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// entropyNamed are the cases whose SOURCES rows this test must hold: the
// reads-entropy-* cases, the two whose NIST pulse and Bitcoin block reads
// are unreachable, the Stellar case whose commitment pin is public and
// whose entropy pin is testnet, the two identity cases,
// gates-online-chain_failure, sources-all-entropy_disagree and
// sources-all-agree.
func entropyNamed(id string) bool {
	return strings.HasPrefix(id, "reads-entropy-") || strings.HasPrefix(id, "reads-identity-") || slices.Contains([]string{
		"reads-unreachable-nist_pulse", "reads-unreachable-bitcoin_block", "reads-stellar-commitment_public-entropy_testnet",
		"gates-online-chain_failure", "sources-all-entropy_disagree", "sources-all-agree",
	}, id)
}

// entropyBasis is the basis of each entropy source's candidate (SPEC.md
// section 11.2).
var entropyBasis = map[string]string{"nist": "nist_pulse_time_read", "stellar": "stellar_ledger_close_read", "bitcoin": "bitcoin_median_time_read"}

// entropyObjectsOf returns each entropy object the proof carries, by its
// part, with the source its position names.
func entropyObjectsOf(proof map[string]any) map[string]string {
	out := map[string]string{}
	for _, source := range []string{"bitcoin", "nist", "stellar"} {
		for _, part := range []string{"/witnesses/entropy_" + source, "/observations/entropy_" + source + "/entropy"} {
			if _, ok := valueAtPart(proof, part).(map[string]any); ok {
				out[part] = source
			}
		}
		if proof["type"] == "entropy_"+source {
			out["/subject/entropy"] = source
		}
	}
	return out
}

// Each such case's SOURCES rows are exactly its expected ones, through the
// replay adapter, and its reads are those of the entropy rules: each read
// SOURCES makes is a NIST pulse, naming no network and no source, at an
// object's chainIndex and pulseIndex; a Stellar ledger at an object's
// sequence, or the Stellar root after a ledger the source does not hold;
// or a Bitcoin block by an object's hash, never the tip or a best-chain
// block; each Stellar and Bitcoin read of the network of the case's
// entropy pin, which reads-stellar-commitment_public-entropy_testnet sets
// apart from its commitment pin. Each passing SOURCES row keeps one
// candidate, its basis its source's, at the instant each source it rests
// on read (the timeStamp, the closed_at or the median_time_past), and a
// row that does not pass keeps none; when the case's label is not failed,
// its submitted-after edge is the latest of those candidates, the first in
// row order of those at one instant, and there is none when it has no
// edge. Mutants killed: the Stellar entropy read made of the commitment
// pin's network (reads-stellar-commitment_public-entropy_testnet reads a
// ledger of public, which its file does not hold); the Bitcoin block read
// at the object's height on the best chain (no record holds that read, and
// each Bitcoin row skips); the NIST read naming a network (no record holds
// it, and each NIST row skips);
// the Bitcoin candidate taken from the block's time (the candidate is not
// its median_time_past, and the edge differs); SOURCES candidates not kept
// (no candidate is the edge); the root read before the ledger (a root
// read with no not-found ledger before it).
func TestCases_TheEntropyCases(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	held, named := 0, 0
	for _, c := range cases {
		if c.Config.Online == nil || c.Expected.Bare() || gateOf(c) != verifier.ExternalGateOpen || len(rowsBetween("SOURCES", "SOURCES", c.Expected.Rows)) == 0 {
			continue
		}
		held++
		if entropyNamed(c.ID) {
			named++
		}
		t.Run(c.ID, func(t *testing.T) {
			got, replay := runCase(t, c)
			want := rowsBetween("SOURCES", "SOURCES", c.Expected.Rows)
			if gotRows := rowsBetween("SOURCES", "SOURCES", got.rows); !rowsEqual(gotRows, want) {
				t.Fatalf("SOURCES rows %v, want %v", gotRows, want)
			}
			file, err := vectorcases.LoadCaseReplay(c)
			if err != nil {
				t.Fatal(err)
			}
			proof := proofOf(t, c)
			objects := entropyObjectsOf(proof)
			params := map[string]bool{}
			for part := range objects {
				for _, name := range []string{"payload/pulse/chainIndex", "payload/pulse/pulseIndex", "payload/sequence", "payload/hash"} {
					if v := valueAtPart(proof, part+"/"+name); v != nil {
						params[fmt.Sprint(strings.Split(name, "/")[len(strings.Split(name, "/"))-1], " ", v)] = true
					}
				}
			}
			missing := map[string]bool{}
			for _, r := range sourcesReadsOf(replay) {
				kind, _ := r.read["kind"].(string)
				network, hasNetwork := r.read["network"]
				switch kind {
				case "nist_pulse":
					if hasNetwork || r.source != "" || !params[fmt.Sprint("chainIndex ", r.read["chainIndex"])] || !params[fmt.Sprint("pulseIndex ", r.read["pulseIndex"])] {
						t.Errorf("a NIST read %v of %q", r.read, r.source)
					}
				case "stellar_ledger", "stellar_root":
					if network != c.Config.Networks.Entropy["stellar"] {
						t.Errorf("a Stellar read %v of the network %v", r.read, network)
					}
					if kind == "stellar_ledger" && !params[fmt.Sprint("sequence ", r.read["sequence"])] {
						t.Errorf("a ledger read %v at no object's sequence", r.read)
					}
					if answer, _ := file.Answer(r.read, r.source); kind == "stellar_ledger" && answer.Outcome == vectorcases.OutcomeNotFound {
						missing[r.source] = true
					}
					if kind == "stellar_root" && !missing[r.source] {
						t.Errorf("the root of source %q read with no ledger not found before it", r.source)
					}
				case "bitcoin_block":
					if network != c.Config.Networks.Entropy["bitcoin"] || !params[fmt.Sprint("hash ", r.read["hash"])] {
						t.Errorf("a Bitcoin read %v", r.read)
					}
				default:
					t.Errorf("SOURCES made a read of kind %s", kind)
				}
			}
			checkEntropyCandidates(t, c, got.run, objects, file, replay)
		})
	}
	if held != 66 || named != 19 {
		t.Fatalf("%d entropy cases, want 66; %d named, want 19", held, named)
	}
}

// checkEntropyCandidates holds the SOURCES candidates a case's run keeps
// (see TestCases_TheEntropyCases).
func checkEntropyCandidates(t *testing.T, c vectorcases.Case, r *verifier.Run, objects map[string]string, file *vectorcases.Replay, replay *caseReplay) {
	t.Helper()
	passed := map[string]bool{}
	for _, row := range rowsBetween("SOURCES", "SOURCES", c.Expected.Rows) {
		passed[*row.Part] = row.Status == "pass"
	}
	var kept []verifier.RowCandidate
	for _, candidate := range r.Candidates {
		if candidate.Step == "SOURCES" {
			kept = append(kept, candidate)
		}
	}
	at := map[string]verifier.RowCandidate{}
	for _, candidate := range kept {
		if _, twice := at[candidate.Part]; twice || !passed[candidate.Part] {
			t.Errorf("a candidate at %s, whose row passed %v", candidate.Part, passed[candidate.Part])
		}
		at[candidate.Part] = candidate
	}
	proof := proofOf(t, c)
	for part, source := range objects {
		candidate, ok := at[part]
		if ok != passed[part] {
			t.Errorf("%s: a candidate %v, the row passed %v", part, ok, passed[part])
			continue
		}
		if !ok {
			continue
		}
		class := verifier.ClassTIME
		if source == "nist" {
			class = verifier.ClassNISTTIME
		}
		if candidate.Candidate.Basis != entropyBasis[source] || candidate.Candidate.Class != class || len(candidate.Counted) == 0 {
			t.Errorf("%s: candidate %+v", part, candidate)
		}
		for _, counted := range candidate.Counted {
			name := replay.names[counted.Source.Source.URL]
			instant, ok := readInstant(t, c, proof, part, source, file, name)
			if !ok || instant != candidate.Candidate.Instant || counted.Outcome.Status != verifier.StatusPass {
				t.Errorf("%s: counted source %q gave %d, the candidate %d", part, name, instant, candidate.Candidate.Instant)
			}
		}
	}
	if c.Expected.Label == "failed" {
		return
	}
	var latest *verifier.RowCandidate
	for i := range kept {
		if latest == nil || kept[i].Candidate.Milliseconds() > latest.Candidate.Milliseconds() {
			latest = &kept[i]
		}
	}
	if string(c.Expected.SubmittedAfter) == "null" {
		if latest != nil {
			t.Errorf("no submitted-after edge, but the candidate %+v", latest)
		}
		return
	}
	var edge struct{ Basis, Instant string }
	if err := json.Unmarshal(c.Expected.SubmittedAfter, &edge); err != nil || latest == nil {
		t.Fatalf("the edge %s: %v, the latest candidate %v", c.Expected.SubmittedAfter, err, latest)
	}
	instant, _ := verifier.FormatTime(latest.Candidate.Class, latest.Candidate.Instant)
	if edge.Basis != latest.Candidate.Basis || edge.Instant != instant {
		t.Errorf("the latest candidate %s %s, the edge %+v", latest.Candidate.Basis, instant, edge)
	}
}

// readInstant is the instant the replay's source name gave in the read of
// the entropy object at part: the pulse's timeStamp in milliseconds, the
// ledger's closed_at and the block's median_time_past in seconds.
func readInstant(t *testing.T, c vectorcases.Case, proof map[string]any, part, source string, file *vectorcases.Replay, name string) (int64, bool) {
	t.Helper()
	switch source {
	case "nist":
		read := map[string]any{"kind": "nist_pulse", "chainIndex": valueAtPart(proof, part+"/payload/pulse/chainIndex"), "pulseIndex": valueAtPart(proof, part+"/payload/pulse/pulseIndex")}
		answer, ok := file.Answer(read, name)
		instant, err := verifier.ParseTime(verifier.ClassNISTTIME, answer.Value["timeStamp"])
		return instant, ok && err == nil
	case "stellar":
		read := map[string]any{"kind": "stellar_ledger", "network": c.Config.Networks.Entropy["stellar"], "sequence": valueAtPart(proof, part+"/payload/sequence")}
		answer, ok := file.Answer(read, name)
		instant, err := verifier.ParseTime(verifier.ClassTIME, answer.Value["closed_at"])
		return instant, ok && err == nil
	}
	read := map[string]any{"kind": "bitcoin_block", "network": c.Config.Networks.Entropy["bitcoin"], "hash": valueAtPart(proof, part+"/payload/hash")}
	answer, ok := file.Answer(read, name)
	median, isNumber := answer.Value["median_time_past"].(strictjson.Number)
	n, isInt := median.Int64()
	return n, ok && isNumber && isInt
}
