// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package verify's conformance test.
//
// testdata/fixtures/appendix-d-item.json is a byte-for-byte copy of the
// worked example published as the Truestamp whitepaper's Appendix D.1
// bundle (whitepaper/vectors/bundle.json in truestamp/truestamp-v2). It is
// vendored here rather than read from the sibling repository so this
// package can self-certify without it.
//
// Appendix E.25 closes with the requirement this file encodes: a verifier
// built from Appendix E alone, run against the Appendix D bundle, MUST
// produce a report whose statuses match the Appendix D.4 table. Editing the
// fixture, or letting a report change shape until TestAppendixD4_Conformance
// has to be relaxed, forfeits that claim.
//
// The bundle's Stellar and Bitcoin transactions are illustrative and do not
// exist on either chain, so the conformance run is by construction offline
// (--skip-external). D.4 is defined as that offline run.
//
// E.25's checklist has one item this file cannot express, because it is
// about bundles the D.1 vector is not: the E.4 hex-encoding sweep, which
// must name every offender under Structure with E.23's invalid_hex_encoding
// identifier, fail the consuming step as well, stay silent on a conforming
// bundle, and leave rtx/txp/the ids/everything in s.d ungraded. That item is
// pinned by TestAppendixD_UppercaseHexFieldsAreRejected,
// TestConformingBundleEmitsNoEncodingRow and
// TestHexEncodingExclusionsAreNotGraded in verify_test.go. Its bearing on
// D.4 is only that a conforming bundle emits no row, which is what keeps
// the containment below unaffected.

package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/proof"
)

// appendixDClaimsHash is the Appendix D bundle's `s.d.hash` — the value a
// caller supplies as `--hash` to exercise E.7's Hash Comparison step, which
// D.4 reports as a pass.
const appendixDClaimsHash = "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"

// d4Row is one row of the Appendix D.4 report table.
//
// Count is the number of rows the CLI may emit for that (group, category,
// status) triple. It is 1 except where the appendix itself licenses more:
//
//   - Epoch Proof: D.4 lists two rows, one per `cx` entry (E.15 requires a
//     result per entry).
//   - Subject Data: D.4's single row reads "Data hash (0x11) and composite
//     fingerprint (0x13) reproduced". The CLI reports the two derivations
//     separately. That is permitted granularity, not an additive step: both
//     rows derive the one thing D.4's row names, they share its status, and
//     neither can be present without the other. No other group gets this
//     latitude — a second row under any other group asserts a proposition
//     D.4 does not report, which E.25 forbids for pass/warn/fail.
type d4Row struct {
	Group    string
	Category string
	Status   Status
	Count    int
}

// appendixD4 is the Appendix D.4 table, in its published order.
var appendixD4 = []d4Row{
	{groupHashComparison, CatDataIntegrity, StatusPass, 1},
	{groupStructure, CatStructural, StatusPass, 1},
	{groupSigningKey, CatCryptographic, StatusPass, 1},
	{groupSubjectData, CatCryptographic, StatusPass, 2}, // 0x11 + 0x13 split
	{groupInclusion, CatCryptographic, StatusPass, 1},
	{groupBlockHash, CatCryptographic, StatusPass, 1},
	{groupEpoch, CatCryptographic, StatusPass, 2}, // Stellar + Bitcoin
	{groupProof, CatCryptographic, StatusPass, 1},
	{groupKeyBinding, CatCryptographic, StatusSkip, 1},
	{groupStellar, CatBlockchain, StatusSkip, 1},
	{groupBitcoin, CatBlockchain, StatusSkip, 1},
	{groupSubmissionWindow, CatTiming, StatusPass, 1},
	{groupTemporalInfo, CatTiming, StatusInfo, 1},
}

// TestAppendixD4_Conformance is E.25's self-certification: the Appendix D
// bundle verified offline against its own claims hash must reproduce the
// D.4 table under one-way containment.
//
// E.25 defines "match" precisely, and the assertions below are that
// definition transcribed:
//
//  1. no step D.4 reports may be absent;
//  2. no step D.4 reports may carry a different status;
//  3. additional `skip` and `info` rows ARE conformant — neither can change
//     a verdict, and forbidding them would prohibit the honest reporting
//     E.17 and E.20 require;
//  4. an additional `fail`, `pass`, or `warn` row for a step D.4 does not
//     report is NOT conformant, because those statuses move a verdict or
//     read as an assertion.
func TestAppendixD4_Conformance(t *testing.T) {
	report, err := Run(appendixDFixture(t), Options{
		SkipExternal: true,
		ExpectedHash: appendixDClaimsHash,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The report is printed on any failure below, so a mismatch is
	// diagnosable without re-running the binary by hand.
	defer func() {
		if t.Failed() {
			t.Logf("observed report:\n%s", formatSteps(report.Steps))
		}
	}()

	for _, v := range d4Violations(report.Steps) {
		t.Error(v)
	}

	// The verdict D.4 states, and the exit code the CLI derives from it.
	if !report.Passed() {
		t.Error("D.4: no step fails, so the verdict must be passed")
	}
	if got := report.Verdict(); got != VerdictFullyVerified {
		t.Errorf("verdict: got %v, want VerdictFullyVerified (an expected hash was supplied and matches)", got)
	}
}

// d4Violations returns one message per way steps fails E.25's one-way
// containment against [appendixD4]. An empty result means conformant.
//
// The counting is done over rows carrying D.4's OWN status, not over every
// row sharing the (group, category). Two earlier defects came from
// conflating the two:
//
//   - No minimum. Only total absence was flagged, so one of D.4's two Epoch
//     Proof rows (or one of the two Subject Data rows) could vanish and the
//     predicate still called the report conformant — the appendix publishes
//     both, and a report that derives one epoch root is not the report D.4
//     describes.
//   - Additive skip/info rejected. Any row under a D.4 (group, category)
//     whose status differed was flagged, including the skip and info rows
//     clause (3) explicitly permits. That made the predicate over-strict in
//     exactly the direction E.17 and E.20 need latitude, and it contradicted
//     this file's own transcription of E.25.
//
// What the predicate still cannot see: a report that DROPS a D.4 row and
// appends a fabricated one with the same (group, category, status). Nothing
// in E.25's containment definition distinguishes them — the appendix grades
// statuses, not messages. That residue is covered by
// TestAppendixD4_ReportIsStable, which pins the full row list including
// message text.
func d4Violations(steps []Step) []string {
	var out []string

	// (1) + (2): every D.4 row present, carrying D.4's status, at least
	// Count times. A row that exists under some other status is not this
	// row: it is either an additive skip/info (clause 3 permits it, and
	// atStatus below simply does not count it) or an additive verdict-
	// moving row that the budget loop reports.
	for _, want := range appendixD4 {
		atStatus := 0
		for _, s := range steps {
			if s.Group == want.Group && s.Category == want.Category && s.Status == want.Status {
				atStatus++
			}
		}
		if atStatus < want.Count {
			out = append(out, fmt.Sprintf("D.4 row absent or short: %s / %s / %s — %d row(s), want %d",
				want.Group, want.Category, statusName(want.Status), atStatus, want.Count))
		}
		// The maximum applies only to verdict-moving statuses: an extra
		// `pass` asserts something D.4 does not report, while an extra
		// skip/info cannot move a verdict and clause (3) permits it.
		if atStatus > want.Count && want.Status != StatusSkip && want.Status != StatusInfo {
			out = append(out, fmt.Sprintf("D.4 row %s / %s: %d %s rows, at most %d permitted",
				want.Group, want.Category, atStatus, statusName(want.Status), want.Count))
		}
	}

	// (3) + (4): no additive pass/warn/fail. A row is "accounted for" when
	// its (group, category) appears in D.4 and it fits that row's count
	// budget; anything else with a verdict-moving status is additive.
	budget := map[d4Key]int{}
	for _, want := range appendixD4 {
		if want.Status == StatusPass || want.Status == StatusWarn || want.Status == StatusFail {
			budget[d4Key{want.Group, want.Category}] = want.Count
		}
	}
	for _, s := range steps {
		if s.Status == StatusSkip || s.Status == StatusInfo {
			continue // E.25: additive skip/info rows are conformant.
		}
		key := d4Key{s.Group, s.Category}
		if budget[key] <= 0 {
			out = append(out, fmt.Sprintf("additive %s row not accounted for by D.4: %s / %s — %s",
				statusName(s.Status), s.Group, s.Category, s.Message))
			continue
		}
		budget[key]--
	}
	return out
}

// TestAppendixD4_ReportIsStable pins the D.4 run's full row list —
// status, category, group AND message — against a committed golden.
//
// This is deliberately NOT E.25's containment predicate, and it must not
// be confused for it: E.25 grades statuses, so d4Violations cannot tell a
// dropped D.4 row from a fabricated row with the same (group, category,
// status) put in its place. Substituting a real assertion for an invented
// one is precisely the failure a conformance suite exists to catch, and
// only a full-row comparison sees it.
//
// The golden is expected to change when the report legitimately changes.
// Regenerating it is a deliberate act: read the diff, confirm every moved
// row is one the appendix still licenses, and only then commit it. The
// conformance vector itself is frozen, so an unexplained diff here means
// the verifier's behaviour moved on the one bundle whose report Appendix
// D publishes.
func TestAppendixD4_ReportIsStable(t *testing.T) {
	report, err := Run(appendixDFixture(t), Options{
		SkipExternal: true,
		ExpectedHash: appendixDClaimsHash,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := string(readFileT(t, filepath.Join("testdata", "golden", "appendix-d4-report.txt")))
	if got := formatSteps(report.Steps); got != want {
		t.Errorf("the Appendix D.4 report changed.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestD4Violations_IsNotVacuous proves the containment predicate actually
// discriminates. A conformance test that cannot fail certifies nothing, and
// this one guards a table (appendixD4) that a future editor could quietly
// widen until any report satisfies it.
func TestD4Violations_IsNotVacuous(t *testing.T) {
	report, err := Run(appendixDFixture(t), Options{
		SkipExternal: true,
		ExpectedHash: appendixDClaimsHash,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if v := d4Violations(report.Steps); len(v) != 0 {
		t.Fatalf("baseline is not conformant: %v", v)
	}

	for _, tc := range []struct {
		name  string
		mutet func([]Step) []Step
	}{
		{"additive pass row", func(s []Step) []Step {
			return append(s, Step{Group: "Novel Check", Category: CatCryptographic, Status: StatusPass, Message: "x"})
		}},
		{"additive warn row", func(s []Step) []Step {
			return append(s, Step{Group: "Novel Check", Category: CatStructural, Status: StatusWarn, Message: "x"})
		}},
		{"second Structure pass row", func(s []Step) []Step {
			return append(s, Step{Group: groupStructure, Category: CatStructural, Status: StatusPass, Message: "x"})
		}},
		{"third Subject Data pass row", func(s []Step) []Step {
			return append(s, Step{Group: groupSubjectData, Category: CatCryptographic, Status: StatusPass, Message: "x"})
		}},
		{"D.4 row missing", func(s []Step) []Step {
			var out []Step
			for _, st := range s {
				if st.Group != groupKeyBinding {
					out = append(out, st)
				}
			}
			return out
		}},
		{"D.4 row with the wrong status", func(s []Step) []Step {
			out := append([]Step(nil), s...)
			for i := range out {
				if out[i].Group == groupSubmissionWindow {
					out[i].Status = StatusSkip
				}
			}
			return out
		}},
		// The minimum-count hole. D.4 publishes two Epoch Proof rows —
		// E.15 requires one per `cx` entry — so a report that derives
		// only one epoch root is not D.4's report. The predicate used to
		// flag only TOTAL absence, so dropping one of a multi-row group
		// was invisible.
		{"one of the two Epoch Proof rows dropped", func(s []Step) []Step {
			return dropFirst(s, func(st Step) bool { return st.Group == groupEpoch })
		}},
		{"one of the two Subject Data pass rows dropped", func(s []Step) []Step {
			return dropFirst(s, func(st Step) bool {
				return st.Group == groupSubjectData && st.Category == CatCryptographic && st.Status == StatusPass
			})
		}},
		// The same hole reached the other way: a D.4 row replaced by a
		// row of a status that cannot stand in for it. Count stays at
		// two, so only a per-status minimum catches it.
		{"an Epoch Proof pass downgraded to a skip", func(s []Step) []Step {
			out := append([]Step(nil), s...)
			for i := range out {
				if out[i].Group == groupEpoch {
					out[i].Status = StatusSkip
					break
				}
			}
			return out
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := tc.mutet(append([]Step(nil), report.Steps...))
			if v := d4Violations(mutated); len(v) == 0 {
				t.Error("containment predicate accepted a non-conformant report")
			}
		})
	}

	// The converse: an additive skip or info row must stay conformant, or
	// the predicate would forbid exactly the honest reporting E.17 and
	// E.20 require.
	//
	// The (group, category) of each additive row is the part that used to
	// go untested. The old cases all used a group name that CANNOT collide
	// with a D.4 row, so the predicate's real behaviour — flagging any row
	// under a D.4 (group, category) whose status differed — never showed
	// up. A disclosure info row under Bitcoin Commitment, or a second skip
	// under Stellar Commitment, is precisely the shape E.25 permits and
	// the shape the CLI already emits elsewhere.
	for _, tc := range []struct {
		name     string
		group    string
		category string
		status   Status
	}{
		{"non-colliding skip", "Novel Check", CatCryptographic, StatusSkip},
		{"non-colliding info", "Novel Check", CatCryptographic, StatusInfo},
		{"info under a D.4 skip row", groupBitcoin, CatBlockchain, StatusInfo},
		{"second skip under a D.4 skip row", groupStellar, CatBlockchain, StatusSkip},
		{"info under a D.4 pass row", groupHashComparison, CatDataIntegrity, StatusInfo},
		{"skip under a D.4 pass row", groupStructure, CatStructural, StatusSkip},
		{"skip under a D.4 info row", groupTemporalInfo, CatTiming, StatusSkip},
	} {
		t.Run("additive/"+tc.name, func(t *testing.T) {
			extra := append(append([]Step(nil), report.Steps...),
				Step{Group: tc.group, Category: tc.category, Status: tc.status, Message: "additive disclosure"})
			if v := d4Violations(extra); len(v) != 0 {
				t.Errorf("additive %s row under %s / %s rejected, but E.25 permits it: %v",
					statusName(tc.status), tc.group, tc.category, v)
			}
		})
	}
}

// dropFirst returns steps with the first element satisfying pred removed.
func dropFirst(steps []Step, pred func(Step) bool) []Step {
	out := make([]Step, 0, len(steps))
	dropped := false
	for _, s := range steps {
		if !dropped && pred(s) {
			dropped = true
			continue
		}
		out = append(out, s)
	}
	return out
}

// TestAppendixD4_CategoryOrder pins E.22's fixed category report order on
// the D.4 run. D.4's own preamble states the table is grouped in that
// order, so a report that carries every row but shuffles the categories
// still fails to reproduce the table.
func TestAppendixD4_CategoryOrder(t *testing.T) {
	report, err := Run(appendixDFixture(t), Options{
		SkipExternal: true,
		ExpectedHash: appendixDClaimsHash,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := BuildJSONOutput(report)
	last := -1
	for _, s := range out.Steps {
		rank := categoryRank(s.Category)
		if rank < last {
			t.Fatalf("category order violates E.22 at %s / %s:\n%s",
				s.Group, s.Category, formatSteps(report.Steps))
		}
		last = rank
	}
}

// TestAppendixD4_CBORConformance runs the same containment check over the
// bundle re-encoded as deterministic CBOR, in both forms E.3 requires: the
// self-describing tag 55799 wrapper and a bare map.
//
// The report is the observable that proves E.3's CBOR-to-JSON value mapping
// was applied faithfully. A lossy mapping — a byte string left unconverted,
// a CBOR integer widened to a float — changes a derived hash and turns a
// pass into a fail, so an identical report is a stronger assertion than any
// structural comparison of the decoded bundles.
func TestAppendixD4_CBORConformance(t *testing.T) {
	jsonReport, err := Run(appendixDFixture(t), Options{
		SkipExternal: true,
		ExpectedHash: appendixDClaimsHash,
	})
	if err != nil {
		t.Fatalf("Run (json): %v", err)
	}

	bundle, err := proof.ParseBytes(fixtureBytes(t, "appendix-d-item.json"))
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	tagged, err := bundle.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	// MarshalCBOR always emits the 3-byte tag 55799 prefix; stripping it
	// yields the bare-map form E.3 also requires a verifier to accept.
	const selfDescribingTag = "\xd9\xd9\xf7"
	if !strings.HasPrefix(string(tagged), selfDescribingTag) {
		t.Fatalf("MarshalCBOR did not emit the tag 55799 prefix: % x", tagged[:3])
	}
	bare := tagged[len(selfDescribingTag):]

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"tagged", tagged},
		{"bare map", bare},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cborReport, err := RunFromBytes(tc.data, "appendix-d.cbor", Options{
				SkipExternal: true,
				ExpectedHash: appendixDClaimsHash,
			})
			if err != nil {
				t.Fatalf("RunFromBytes: %v", err)
			}
			if !cborReport.Passed() {
				t.Fatalf("CBOR round-trip failed verification:\n%s", formatSteps(cborReport.Steps))
			}
			for _, v := range d4Violations(cborReport.Steps) {
				t.Error(v)
			}
			if got, want := formatSteps(cborReport.Steps), formatSteps(jsonReport.Steps); got != want {
				t.Errorf("CBOR report differs from JSON report:\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

type d4Key struct{ group, category string }

// --- tamper suite ---

// TestAppendixD_TamperDetection mutates one field of the conformance vector
// at a time and asserts the verifier refuses it. Every case starts from a
// bundle that verifies (TestAppendixD4_Conformance proves it), so a case
// that stops failing is a detection the verifier lost, not a fixture that
// drifted.
//
// This is the load-bearing regression guard for the whole Appendix E
// conformance pass: a report-shape change that quietly turns a fail into a
// pass or a skip is worse than any non-conformance it fixed.
//
// A tamper is "detected" either as an E.6 hard rejection (no report at all)
// or as at least one `fail` step. Both are refusals; which one applies is a
// property of where the appendix places the check, not of how severe the
// tamper is.
func TestAppendixD_TamperDetection(t *testing.T) {
	base := fixtureBytes(t, "appendix-d-item.json")

	sub := func(m map[string]any) map[string]any { return m["s"].(map[string]any) }
	blk := func(m map[string]any) map[string]any { return m["b"].(map[string]any) }
	cx := func(m map[string]any, i int) map[string]any {
		return m["cx"].([]any)[i].(map[string]any)
	}

	for _, tc := range []struct {
		name string
		// wantRejected marks the cases E.6 refuses before any step runs.
		wantRejected bool
		mutate       func(map[string]any)
	}{
		{name: "s.d edited", mutate: func(m map[string]any) {
			sub(m)["d"].(map[string]any)["name"] = "Not the timestamped item"
		}},
		{name: "s.mh replaced", mutate: func(m map[string]any) {
			sub(m)["mh"] = strings.Repeat("ab", 32)
		}},
		{name: "s.kid replaced", mutate: func(m map[string]any) {
			sub(m)["kid"] = "deadbeef"
		}},
		{name: "s.id replaced", mutate: func(m map[string]any) {
			sub(m)["id"] = "01KY9ZWEX0248J48HK6D248NAP"
		}},
		{name: "b.mr replaced", mutate: func(m map[string]any) {
			blk(m)["mr"] = strings.Repeat("cd", 32)
		}},
		{name: "b.ph replaced", mutate: func(m map[string]any) {
			blk(m)["ph"] = strings.Repeat("ef", 32)
		}},
		{name: "b.mh replaced", mutate: func(m map[string]any) {
			blk(m)["mh"] = strings.Repeat("01", 32)
		}},
		{name: "b.kid replaced", mutate: func(m map[string]any) {
			blk(m)["kid"] = "0badc0de"
		}},
		{name: "epoch root tampered", mutate: func(m map[string]any) {
			cx(m, 0)["memo"] = strings.Repeat("23", 32)
		}},
		{name: "cx entries reordered", mutate: func(m map[string]any) {
			list := m["cx"].([]any)
			m["cx"] = []any{list[1], list[0]}
		}},
		{name: "ts shifted", mutate: func(m map[string]any) {
			m["ts"] = "2026-07-24T12:00:01Z"
		}},
		// E.16 puts `t` in the signed payload, so retyping a bundle
		// without re-signing must break the signature. t=30 keeps the
		// item/entropy wire shape so the bundle still parses and the
		// tamper is graded rather than rejected.
		{name: "t retyped to entropy_nist", mutate: func(m map[string]any) {
			m["t"] = 30
		}},
		// t=10 is block-like, and E.6 forbids `s`/`ip` on a block-like
		// bundle — the shape rule catches this one before any step.
		{name: "t retyped to block", wantRejected: true, mutate: func(m map[string]any) {
			m["t"] = 10
		}},
		// A correctly sized signature, so this exercises Ed25519's own
		// rejection rather than the length guard in front of it.
		{name: "sig replaced", mutate: func(m map[string]any) {
			m["sig"] = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
		}},
		{name: "sig wrong length", mutate: func(m map[string]any) {
			m["sig"] = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		}},
		{name: "pk replaced", mutate: func(m map[string]any) {
			m["pk"] = "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="
		}},
		{name: "ip truncated", mutate: func(m map[string]any) {
			ip := m["ip"].(string)
			m["ip"] = ip[:len(ip)-8]
		}},
		{name: "v bumped", mutate: func(m map[string]any) {
			m["v"] = 2
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := mutateBundle(t, base, tc.mutate)
			// SkipSignatures is deliberately NOT set: the signature is
			// the only thing several of these tampers disturb.
			report, err := RunFromBytes(raw, "tamper.json", Options{
				SkipExternal: true,
				ExpectedHash: appendixDClaimsHash,
			})
			if tc.wantRejected {
				if err == nil {
					t.Fatalf("tamper accepted: want an E.6 hard rejection, got a report:\n%s",
						formatSteps(report.Steps))
				}
				return
			}
			if err != nil {
				// A hard rejection is still a refusal. Report it so the
				// expectation can be re-keyed rather than silently
				// treating a rejection as a graded fail.
				t.Logf("hard rejection (also a refusal): %v", err)
				return
			}
			if report.Passed() {
				t.Fatalf("TAMPER NOT DETECTED — the mutated bundle passed:\n%s",
					formatSteps(report.Steps))
			}
			t.Logf("detected:\n%s", failedSteps(report.Steps))
		})
	}
}

// failedSteps renders only the fail rows, for tamper-suite output.
func failedSteps(steps []Step) string {
	var out []Step
	for _, s := range steps {
		if s.Status == StatusFail {
			out = append(out, s)
		}
	}
	return formatSteps(out)
}

// --- shared helpers for the conformance and tamper suites ---

func appendixDFixture(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "fixtures", "appendix-d-item.json")
}

// fixtureBytes reads a testdata/fixtures file. Distinct from fuzz_test.go's
// readFixture, which takes a full path and swallows errors so a missing seed
// corpus file does not fail a fuzz target.
func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	return readFileT(t, filepath.Join("testdata", "fixtures", name))
}

func readFileT(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

// mutateBundle decodes a bundle into a generic map, applies mutate, and
// re-encodes it. Decoding through map[string]any rather than ProofBundle
// keeps unknown and malformed shapes representable, which is the whole
// point of a tamper fixture.
func mutateBundle(t *testing.T, raw []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var m map[string]any
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber() // never round a big integer while building a tamper case
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	mutate(m)
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("encode bundle: %v", err)
	}
	return out
}

// formatSteps renders a report as a stable, greppable table. Used as the
// failure output for every conformance and tamper assertion.
func formatSteps(steps []Step) string {
	var b strings.Builder
	for _, s := range steps {
		fmt.Fprintf(&b, "  %-6s %-14s %-20s %s\n", statusName(s.Status), s.Category, s.Group, s.Message)
	}
	return b.String()
}

// statusName renders a Status for test output. Deliberately a helper and
// not a String method on the production type: a fmt.Stringer declared in a
// _test.go file would make %v print differently under test than in a real
// build.
func statusName(s Status) string {
	if str, ok := statusStrings[s]; ok {
		return str
	}
	return fmt.Sprintf("Status(%d)", int(s))
}
