// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The conformance test of SPEC.md section 13.1: every case of cases.json
// through the one runner (runner_test.go's runCase), each online case
// answered from its replay file by the replay adapter (section 13.2),
// compared as section 13.1 compares it and nothing more, from the bare
// rejection or the report as the verifier writes it, read back as any
// consumer reads it. The per-step tests of this package compare more: the
// values, the attribution, the trust root and every member of a report.

package vectorcases_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// conformance holds what one run gives against its case as SPEC.md
// section 13.1 compares them, and returns what it compared, for the
// tallies: for a bare rejection, its code, and its sub_reason, present
// exactly when the case gives one; otherwise the report's label, its code
// when the case's expected gives one, and its rows, each reduced to
// (code, part, status, step), all of them in order; for an online case,
// each edge's basis and instant, the instant compared as an instant
// (section 11.3), or the edge's absence; and, for a case that names a
// trace, the run's trace, its rows in order, each member but inputs.
// Messages, values and the exit code are never compared. The rejection
// and the report are read from the JSON the verifier writes, as a
// consumer of `truestamp verify --json` reads them.
func conformance(t *testing.T, c vectorcases.Case, got outcome) []string {
	t.Helper()
	if c.Expected.Bare() {
		if got.rejection == nil {
			t.Fatalf("a report, %v; the case expects the bare rejection %s", got.rows, c.Expected.Code)
		}
		encoded, err := got.rejection.Encode()
		if err != nil {
			t.Fatal(err)
		}
		var rejection map[string]any
		if err := json.Unmarshal(encoded, &rejection); err != nil {
			t.Fatal(err)
		}
		code, _ := rejection["code"].(string)
		subReason, hasSubReason := rejection["sub_reason"].(string)
		if code != c.Expected.Code || subReason != c.Expected.SubReason || hasSubReason != (c.Expected.SubReason != "") {
			t.Fatalf("the rejection %s; want code %q, sub_reason %q", encoded, c.Expected.Code, c.Expected.SubReason)
		}
		return []string{"bare rejection"}
	}
	if got.rejection != nil {
		t.Fatalf("the bare rejection %s (%s); the case expects a report", got.code, got.subReason)
	}
	encoded, err := verifier.NewReport(got.run).Encode()
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Code            *string           `json:"code"`
		Label           string            `json:"label"`
		Rows            []json.RawMessage `json:"rows"`
		SubmittedAfter  json.RawMessage   `json:"submitted_after"`
		SubmittedBefore json.RawMessage   `json:"submitted_before"`
	}
	if err := json.Unmarshal(encoded, &report); err != nil {
		t.Fatal(err)
	}
	compared := []string{"report"}
	if report.Label != c.Expected.Label {
		t.Fatalf("label %q, want %q", report.Label, c.Expected.Label)
	}
	if c.Expected.Code != "" {
		if report.Code == nil || *report.Code != c.Expected.Code {
			t.Fatalf("code %v, want %q", report.Code, c.Expected.Code)
		}
		compared = append(compared, "the code")
	}
	rows := make([]vectorcases.Row, 0, len(report.Rows))
	for _, raw := range report.Rows {
		var row struct {
			Code   *string `json:"code"`
			Part   *string `json:"part"`
			Status string  `json:"status"`
			Step   string  `json:"step"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, vectorcases.Row{Code: row.Code, Part: row.Part, Status: row.Status, Step: row.Step})
	}
	if !rowsEqual(rows, c.Expected.Rows) {
		t.Fatalf("rows\n  %v\nwant\n  %v", rows, c.Expected.Rows)
	}
	if c.Config.Online != nil {
		for _, edge := range []struct {
			name      string
			got, want json.RawMessage
		}{
			{"submitted_after", report.SubmittedAfter, c.Expected.SubmittedAfter},
			{"submitted_before", report.SubmittedBefore, c.Expected.SubmittedBefore},
		} {
			want, drawn := expectedEdgeOf(t, edge.want), expectedEdgeOf(t, edge.got)
			switch {
			case want == nil && drawn == nil:
			case want == nil || drawn == nil || drawn.Basis != want.Basis || !sameInstant(t, drawn.Instant, want.Instant):
				t.Fatalf("%s %s, want %s", edge.name, edge.got, edge.want)
			}
		}
		compared = append(compared, "the edges")
	}
	if c.Expected.Trace != "" {
		var trace []string
		for _, row := range got.run.Trace {
			trace = append(trace, withoutInputs(t, row.Value()))
		}
		var want []string
		for _, row := range traceFile(t, c.Expected.Trace) {
			value, err := strictjson.Parse([]byte(row))
			if err != nil {
				t.Fatal(err)
			}
			want = append(want, withoutInputs(t, value.(map[string]any)))
		}
		if len(want) == 0 || strings.Join(trace, "\n") != strings.Join(want, "\n") {
			t.Fatalf("trace\n  %s\nwant\n  %s", strings.Join(trace, "\n  "), strings.Join(want, "\n  "))
		}
		compared = append(compared, "the trace")
	}
	return compared
}

// fetchedAt is a read's fetched_at member as JCS writes it, the one
// member of a report that the clock, not the case, decides.
var fetchedAt = regexp.MustCompile(`"fetched_at":"[^"]*"`)

// shipped holds verifier.Verify, the function `truestamp verify` runs, to
// the one runner over case c: under the configuration runCase gives c,
// with its own replay adapter, reading c's file as the command reads a
// file, through strictjson.ReadFile, Verify gives a bare rejection when
// the runner gave one and a report when it gave one, and its encoding
// equals that of the runner's outcome got byte for byte, each fetched_at
// apart, which is when the read's answer came back.
func shipped(t *testing.T, c vectorcases.Case, got outcome) {
	t.Helper()
	o := verifier.Verify(caseConfig(t, c, replayOf(t, c)), func() ([]byte, error) {
		return strictjson.ReadFile(c.ProofPath())
	})
	var want, gave []byte
	var err error
	switch {
	case o.Err != nil:
		t.Fatalf("Verify: %v", o.Err)
	case got.rejection != nil && o.Rejection == nil, got.rejection == nil && o.Report == nil:
		t.Fatalf("Verify gives the report %v and the rejection %v; the runner gives the rejection %v", o.Report != nil, o.Rejection, got.rejection)
	case got.rejection != nil:
		if want, err = got.rejection.Encode(); err == nil {
			gave, err = o.Rejection.Encode()
		}
	default:
		if want, err = verifier.NewReport(got.run).Encode(); err == nil {
			gave, err = o.Report.Encode()
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	stamped := []byte(`"fetched_at":""`)
	if gave, want = fetchedAt.ReplaceAll(gave, stamped), fetchedAt.ReplaceAll(want, stamped); !bytes.Equal(gave, want) {
		t.Fatalf("Verify writes\n  %s\nthe runner\n  %s", gave, want)
	}
}

// withoutInputs is a trace row, value, as the strict reader gives it,
// written with JCS without its inputs member, as SPEC.md section 13.1
// compares it.
func withoutInputs(t *testing.T, value map[string]any) string {
	t.Helper()
	rest := map[string]any{}
	for name, member := range value {
		if name != "inputs" {
			rest[name] = member
		}
	}
	if len(rest) != len(value)-1 {
		t.Fatalf("a trace row without inputs: %v", value)
	}
	encoded, err := jcs.Encode(rest)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

// citation is the claim of conformance SPEC.md section 13.1 asks for: the
// commit of truestamp-verification the CLI's copy holds, from
// testdata/verification/SOURCE, and the SHA-256 of that copy's
// MANIFEST.sha256. It also holds cases.json, which the runner reads, to the
// digest the manifest gives it.
func citation(t *testing.T) (commit, manifest string) {
	t.Helper()
	root := filepath.Join(vectorcases.Dir(), "..")
	source, err := os.ReadFile(filepath.Join(root, "SOURCE"))
	if err != nil {
		t.Fatal(err)
	}
	commit = strings.TrimSuffix(string(source), "\n")
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(commit) {
		t.Fatalf("SOURCE holds %q, not a commit", source)
	}
	lines, err := os.ReadFile(filepath.Join(root, "MANIFEST.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	cases, err := os.ReadFile(vectorcases.Path("cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	casesDigest := sha256.Sum256(cases)
	listed := ""
	scanner := bufio.NewScanner(bytes.NewReader(lines))
	for scanner.Scan() {
		if digest, path, ok := strings.Cut(scanner.Text(), "  "); ok && path == "vectors/cases.json" {
			listed = digest
		}
	}
	if listed != hex.EncodeToString(casesDigest[:]) {
		t.Fatalf("MANIFEST.sha256 gives vectors/cases.json the digest %q; the file read has %x", listed, casesDigest)
	}
	sum := sha256.Sum256(lines)
	return commit, hex.EncodeToString(sum[:])
}

// The conformance test of SPEC.md section 13.1: this verifier gives the
// expected output of every case of cases.json, all 482, each of the 87
// online ones answered from its replay file by the replay adapter
// (SPEC.md section 13.2), as section 13.1 compares it (conformance): the
// 65 bare rejections by code and sub_reason, and the 417 reports by label
// and every row, the 81 online reports by their edges too, the one case
// whose expected gives a code (two_steps-policy-derive) by that code, and
// the 28 cases that name one of the 11 sound traces by that trace. It logs the claim of conformance
// section 13.1 asks for, the spec commit and the digest of MANIFEST.sha256
// (citation). Each case also goes through verifier.Verify, which `truestamp
// verify` runs, and its encoded output must equal the runner's (shipped).
// Mutants killed, each in internal/verifier: laterSteps with CHAIN and
// SOURCES swapped, or with WINDOW and EXPECTED_HASH swapped (Verify's rows
// come in another order than the runner's, which the cases hold); Verdict giving
// the code of the last failing row (two_steps-policy-derive's code is
// derivation_mismatch); Row.Value writing a null part for every skip row
// (every skip row's part differs); Edge.Value writing the instant one
// unit of its class late, a second or, for NIST, a millisecond (every
// drawn edge's instant differs); Rejection.Value
// writing no sub_reason (the 47 rejections with one differ); TraceRow.Value
// writing no part (every sound trace differs).
func TestConformance(t *testing.T) {
	commit, manifest := citation(t)
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 482 {
		t.Fatalf("cases.json holds %d cases, want 482", len(cases))
	}
	tally := map[string]int{}
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			got, _ := runCase(t, c)
			for _, compared := range conformance(t, c, got) {
				tally[compared]++
			}
			shipped(t, c, got)
			tally["through Verify"]++
			if c.Config.Online != nil {
				tally["online"]++
			}
			tally["matched"]++
		})
	}
	want := map[string]int{
		"matched": 482, "online": 87, "bare rejection": 65, "report": 417,
		"the code": 1, "the edges": 81, "the trace": 28, "through Verify": 482,
	}
	if len(tally) != len(want) {
		t.Fatalf("tally %v, want %v", tally, want)
	}
	for name, n := range want {
		if tally[name] != n {
			t.Fatalf("tally %v, want %v", tally, want)
		}
	}
	t.Logf("conforms to truestamp-verification %s, MANIFEST.sha256 %s: %d cases", commit, manifest, tally["matched"])
}

// One source configured for a network through a case's sources, under
// each policy of SPEC.md section 9, first, at_least 1 and all, gives the
// output of the case, which names no sources and so has one source under
// first (section 13.2): each of the 69 online cases that names none, run
// again with each network its replay file stands for given its one source
// under each policy, matches as the conformance test compares it, and the
// runner's checkSources holds each run to the policy given. Mutant killed:
// CombineSources counting a pass under at_least or all only beside
// another (len(passes) > Required: one source's pass becomes
// inconclusive, and every such case's CHAIN and SOURCES rows differ).
func TestConformance_OneSourceUnderEachPolicyIsNoSources(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	one := 1
	policies := []vectorcases.NetworkSources{
		{Policy: vectorcases.PolicyFirst},
		{Policy: vectorcases.PolicyAtLeast, Required: &one},
		{Policy: vectorcases.PolicyAll},
	}
	runs := 0
	for _, c := range cases {
		if c.Config.Online == nil || c.Config.Sources != nil {
			continue
		}
		file, err := vectorcases.LoadCaseReplay(c)
		if err != nil {
			t.Fatal(err)
		}
		for _, policy := range policies {
			t.Run(c.ID+"/"+policy.Policy, func(t *testing.T) {
				named := c
				named.Config.Sources = map[string]map[string]vectorcases.NetworkSources{}
				for _, s := range file.Sources() {
					if named.Config.Sources[s.Chain] == nil {
						named.Config.Sources[s.Chain] = map[string]vectorcases.NetworkSources{}
					}
					one := policy
					one.Sources = []string{""}
					named.Config.Sources[s.Chain][s.Network] = one
				}
				got, _ := runCase(t, named)
				conformance(t, named, got)
			})
			runs++
		}
	}
	if runs != 69*3 {
		t.Fatalf("%d runs, want %d: three for each of the 69 online cases that name no sources", runs, 69*3)
	}
}
