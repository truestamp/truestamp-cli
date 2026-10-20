// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// testEdge is an edge read from one Stellar source, the submitted-before
// edge at 2026-09-21T14:13:20Z when part is a top-level commitment entry
// and otherwise the submitted-after edge at 2026-09-21T11:26:40Z, its read
// back at 2026-09-21T14:13:21.234Z.
func testEdge(part string) *Edge {
	name, instant := edgeSubmittedAfter, int64(1789990000)
	if strings.HasPrefix(part, "/commitments/") {
		name, instant = edgeSubmittedBefore, 1790000000
	}
	return &Edge{
		Assumption: assumptionOf(stellarCloseBasis, name),
		Candidate:  Candidate{Basis: stellarCloseBasis, Class: ClassTIME, Instant: instant},
		Part:       part,
		Sources:    []EdgeSource{{Source: "https://horizon.example", FetchedAt: time.UnixMilli(1790000001234)}},
	}
}

// reportRun is a run of an item past every step, offline under E_v test
// and a supplied keyring, with both edges drawn and its WINDOW row a pass,
// then extra, then EXPECTED_HASH's skip, labelled by VERDICT.
func reportRun(extra ...Row) *Run {
	digest := strings.Repeat("ab", 32)
	rows := []Row{
		PassRow("CONFIGURATION", nil, "held", nil),
		PassRow("WINDOW", nil, "drawn", map[string]any{"n": strictjson.Number(2)}),
	}
	rows = append(rows, extra...)
	rows = append(rows, NewRow("EXPECTED_HASH", specdata.ReasonExpectedHashNotChecked, nil, "none given", nil))
	run := &Run{
		Config:          Config{Environment: "test"},
		Attribution:     "keyring:" + digest,
		TrustRoot:       TrustRoot{Digest: digest, Kind: KindSuppliedKeyring, Location: "keyring.test.json"},
		Type:            "item",
		SubjectID:       "019a0000-0000-7000-8000-000000000000",
		Rows:            rows,
		SubmittedAfter:  testEdge("/witnesses/entropy_stellar"),
		SubmittedBefore: testEdge("/commitments/stellar"),
	}
	Verdict(run)
	return run
}

// encodedReport is run's report written with JCS.
func encodedReport(t *testing.T, run *Run) string {
	t.Helper()
	encoded, err := NewReport(run).Encode()
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

// The edge objects of reportRun, as JCS writes them (SPEC.md section
// 11.5), and its rows, written here by hand from SPEC.md section 10.5.
const (
	reportAfter = `{"assumption":"the network did not set the close time ahead of real time","basis":"stellar_ledger_close_read",` +
		`"cached":false,"fetched_at":"2026-09-21T14:13:21.234Z","instant":"2026-09-21T11:26:40Z","part":"/witnesses/entropy_stellar",` +
		`"source":"https://horizon.example","sources":[{"cached":false,"fetched_at":"2026-09-21T14:13:21.234Z","source":"https://horizon.example"}]}`
	reportBefore = `{"assumption":"Stellar validators did not set the ledger's close time behind real time; a close time set ahead of real time only makes this edge later",` +
		`"basis":"stellar_ledger_close_read","cached":false,"fetched_at":"2026-09-21T14:13:21.234Z","instant":"2026-09-21T14:13:20Z",` +
		`"part":"/commitments/stellar","source":"https://horizon.example","sources":[{"cached":false,"fetched_at":"2026-09-21T14:13:21.234Z","source":"https://horizon.example"}]}`
	reportRowsHead = `[{"message":"held","part":null,"status":"pass","step":"CONFIGURATION","values":{}},` +
		`{"message":"drawn","part":null,"status":"pass","step":"WINDOW","values":{"n":2}},`
	reportRowsTail  = `{"code":"expected_hash_not_checked","message":"none given","part":null,"status":"skip","step":"EXPECTED_HASH","values":{}}]`
	reportTrustRoot = `{"digest":"abababababababababababababababababababababababababababababababab","kind":"supplied_keyring","location":"keyring.test.json"}`
)

// The report is exactly SPEC.md section 10.5's: the members attribution,
// environment, label, rows, subject_id, submitted_after, submitted_before,
// trust_root, type, verified and window, in that order, which is JCS's,
// and code only when the label is failed; each row {code, message, part,
// status, step, values} with code absent on a pass; never "passed". A
// report that is not failed carries the edges WINDOW drew; a failed one,
// the same run with one more row that fails, carries both null and window
// not_established, its code that row's. Mutants killed: code written on
// every report (the supplied-keyring report has one); code never written
// (the failed report lacks it); a member renamed, added (a "passed"
// member) or left out (each literal differs); the edges written on a
// failed report (window_inverted keeps both); window established on a
// failed report whose run drew an edge; verified true for a
// supplied-keyring report with an edge; a pass row with a code member;
// a row with no values written null; the environment written as prod.
func TestReport_MembersAndOrder(t *testing.T) {
	head := `{"attribution":"keyring:abababababababababababababababababababababababababababababababab",`
	tail := `"subject_id":"019a0000-0000-7000-8000-000000000000",`
	want := head + `"environment":"test","label":"supplied-keyring","rows":` + reportRowsHead + reportRowsTail + `,` + tail +
		`"submitted_after":` + reportAfter + `,"submitted_before":` + reportBefore + `,"trust_root":` + reportTrustRoot +
		`,"type":"item","verified":false,"window":"established"}`
	if got := encodedReport(t, reportRun()); got != want {
		t.Errorf("the report\n%s\nwant\n%s", got, want)
	}

	inverted := NewRow("WINDOW", specdata.ReasonWindowInverted, nil, "inverted", nil)
	run := reportRun(inverted)
	want = head + `"code":"window_inverted","environment":"test","label":"failed","rows":` + reportRowsHead +
		`{"code":"window_inverted","message":"inverted","part":null,"status":"fail","step":"WINDOW","values":{}},` + reportRowsTail + `,` + tail +
		`"submitted_after":null,"submitted_before":null,"trust_root":` + reportTrustRoot + `,"type":"item","verified":false,"window":"not_established"}`
	if got := encodedReport(t, run); got != want {
		t.Errorf("the failed report\n%s\nwant\n%s", got, want)
	}
	if run.SubmittedAfter == nil || run.SubmittedBefore == nil {
		t.Errorf("VERDICT dropped the edges WINDOW drew from the run: %+v %+v", run.SubmittedAfter, run.SubmittedBefore)
	}
}

// A report labelled failed by EXPECTED_HASH, after WINDOW passed and drew
// both edges, has both edges null and window not_established, its code
// expected_hash_mismatch (SPEC.md section 10.2, steps 13 and 14): the
// step run on an item whose claims hash differs from the caller's value.
// Mutants killed: the report's edges taken from the run whatever the label
// (both edges written); the window read from the run's edge
// (established).
func TestReport_ExpectedHashFailsAfterWindow(t *testing.T) {
	run := deriveRun(positiveProofs(t)["item"], AttributionTruestamp)
	run.AddRow(PassRow("WINDOW", nil, "drawn", nil))
	run.SubmittedAfter, run.SubmittedBefore = testEdge("/witnesses/entropy_stellar"), testEdge("/commitments/stellar")
	differs := "09e3941192b086bfdc0e4e120c18ff12"
	run.Config.ExpectedHash = &differs
	ExpectedHash(run)
	Verdict(run)
	report := NewReport(run)
	last := report.Rows[len(report.Rows)-1]
	if report.Label != LabelFailed || report.Code != specdata.ReasonExpectedHashMismatch || report.SubmittedAfter != nil ||
		report.SubmittedBefore != nil || report.Window != WindowNotEstablished || report.Verified ||
		last.Step != "EXPECTED_HASH" || last.Status != StatusFail {
		t.Fatalf("%+v", report)
	}
	encoded := encodedReport(t, run)
	for _, member := range []string{`"submitted_after":null`, `"submitted_before":null`, `"window":"not_established"`, `"code":"expected_hash_mismatch","environment"`} {
		if !strings.Contains(encoded, member) {
			t.Errorf("the report lacks %s: %s", member, encoded)
		}
	}
}

// Each label's report (SPEC.md section 10.5): verified true only for
// verified; window established exactly when the report's submitted-before
// edge is not null, which a failed report never has; code present exactly
// on failed. Mutants killed: verified set from the window (the
// supplied-keyring and unattributed reports with an edge are verified);
// window set from either edge (the internally-consistent report with only
// a submitted-after edge is established); the edges dropped on every
// report (the verified report's window is not established).
func TestReport_EachLabel(t *testing.T) {
	pass := func(step string) Row { return PassRow(step, nil, "held", nil) }
	notEstablished := NewRow("WINDOW", specdata.ReasonWindowNotEstablished, nil, "no edge", nil)
	fail := NewRow("DERIVE", specdata.ReasonDerivationMismatch, Part("/subject/item_hash"), "no", nil)
	before, after := testEdge("/commitments/stellar"), testEdge("/witnesses/entropy_stellar")
	for _, c := range []struct {
		attribution   string
		rows          []Row
		after, before *Edge
		label         string
		verified      bool
		window        string
	}{
		{AttributionTruestamp, []Row{fail, pass("WINDOW")}, after, before, LabelFailed, false, WindowNotEstablished},
		{AttributionUnattributed, []Row{pass("WINDOW")}, after, before, LabelUnattributed, false, WindowEstablished},
		{deriveAttribution, []Row{pass("WINDOW")}, nil, before, LabelSuppliedKeyring, false, WindowEstablished},
		{deriveAttribution, []Row{notEstablished}, nil, nil, LabelSuppliedKeyring, false, WindowNotEstablished},
		{AttributionTruestamp, []Row{notEstablished}, after, nil, LabelInternallyConsistent, false, WindowNotEstablished},
		{AttributionTruestamp, []Row{pass("WINDOW")}, after, before, LabelVerified, true, WindowEstablished},
	} {
		run := &Run{Config: Config{Environment: "prod"}, Attribution: c.attribution, Rows: c.rows, SubmittedAfter: c.after, SubmittedBefore: c.before}
		Verdict(run)
		report := NewReport(run)
		var value map[string]any
		if err := json.Unmarshal([]byte(encodedReport(t, run)), &value); err != nil {
			t.Fatal(err)
		}
		code, hasCode := value["code"]
		if report.Label != c.label || value["label"] != c.label || value["verified"] != c.verified || value["window"] != c.window ||
			hasCode != (c.label == LabelFailed) || (hasCode && code != "derivation_mismatch") || (value["submitted_before"] == nil) != (c.window == WindowNotEstablished) {
			t.Errorf("%s: %v", c.label, value)
		}
	}
}

// A report of a run the gate ended, at ENVIRONMENT, in keyless mode: the
// attribution none, type and subject_id null, trust_root {null, none,
// null}, code environment_mismatch, every later step's skip prior_failure
// row, and both edges null. Mutants killed: type or subject_id written as
// "" (not null); trust_root's digest or location written as "".
func TestReport_BeforeShape(t *testing.T) {
	run := &Run{Config: Config{Environment: "prod"}, Attribution: AttributionNone, TrustRoot: TrustRoot{Kind: KindNone},
		Rows: []Row{NewRow("ENVIRONMENT", specdata.ReasonEnvironmentMismatch, Part("/environment"), "marker", nil)}}
	run.EndRun()
	Verdict(run)
	var value map[string]any
	if err := json.Unmarshal([]byte(encodedReport(t, run)), &value); err != nil {
		t.Fatal(err)
	}
	rows, _ := value["rows"].([]any)
	got := fmt.Sprintf("%v %v %v %v %v %v %v %v %d", value["attribution"], value["code"], value["type"], value["subject_id"],
		value["trust_root"], value["submitted_after"], value["submitted_before"], value["environment"], len(rows))
	if want := "none environment_mismatch <nil> <nil> map[digest:<nil> kind:none location:<nil>] <nil> <nil> prod 10"; got != want {
		t.Fatalf("%s, want %s", got, want)
	}
	for _, member := range []string{"type", "subject_id"} {
		if v, ok := value[member]; !ok || v != nil {
			t.Errorf("%s: %v %v", member, v, ok)
		}
	}
}

// A report exists only after VERDICT: NewReport refuses a run with no
// label. Mutant killed: the check left out (a report with label "").
func TestReport_OnlyAfterVerdict(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a report of an unlabelled run")
		}
	}()
	NewReport(&Run{Rows: []Row{PassRow("CONFIGURATION", nil, "held", nil)}})
}

// A bare rejection is only {code, message, sub_reason} (SPEC.md section
// 10.2, the gate), sub_reason present only for malformed_json and
// invalid_envelope (section 12.1), in JCS's order. Mutants killed:
// sub_reason written as "" or null for configuration_error and
// input_too_large; sub_reason left out of malformed_json and
// invalid_envelope; a fourth member.
func TestRejection_OnlyItsThreeMembers(t *testing.T) {
	for _, c := range []struct {
		rejection *Rejection
		want      string
	}{
		{ConfigurationError("no source"), `{"code":"configuration_error","message":"no source"}`},
		{InputTooLarge(), fmt.Sprintf(`{"code":"input_too_large","message":"The file is larger than %d octets, the most a verifier reads."}`, strictjson.MaxReadOctets)},
		{MalformedJSON(specdata.SubReasonDuplicateName), `{"code":"malformed_json","message":"The file is not JSON that every verifier reads alike.","sub_reason":"duplicate_name"}`},
		{InvalidEnvelope(specdata.SubReasonAlg), `{"code":"invalid_envelope","message":"This file is not a Truestamp proof.","sub_reason":"alg"}`},
	} {
		got, err := c.rejection.Encode()
		if err != nil || !bytes.Equal(got, []byte(c.want)) {
			t.Errorf("%s, %v; want %s", got, err, c.want)
		}
	}
}

// A row as a report writes it (SPEC.md section 10.5): code absent exactly
// on a pass, a null part null, no values {}. Mutants killed: code written
// on a pass as ""; code written only on a fail (the skip and info rows
// lack it); a null part written as ""; no values written null.
func TestRow_Value(t *testing.T) {
	for _, c := range []struct {
		row  Row
		want string
	}{
		{Row{Status: StatusPass, Step: "READ"}, `{"message":"","part":null,"status":"pass","step":"READ","values":{}}`},
		{NewRow("CHAIN", specdata.ReasonOffline, Part("/commitments/stellar"), "offline", map[string]any{"a": true}),
			`{"code":"offline","message":"offline","part":"/commitments/stellar","status":"skip","step":"CHAIN","values":{"a":true}}`},
		{NewRow("DERIVE", specdata.ReasonNotAttributable, nil, "info", nil),
			`{"code":"not_attributable","message":"info","part":null,"status":"info","step":"DERIVE","values":{}}`},
	} {
		got, err := json.Marshal(c.row.Value())
		if err != nil || string(got) != c.want {
			t.Errorf("%s, %v; want %s", got, err, c.want)
		}
	}
}
