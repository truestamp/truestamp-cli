// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// The human output's display rules (SPEC.md section 11.5), over reports
// made in this process: one written by hand, for a cached read, which the
// CLI never makes, and real runs of the vectors' online cases, answered
// from their replay files.

// rendered is the text renderReport writes for report.
func rendered(report *verifier.Report) string {
	var b strings.Builder
	_ = renderReport(&b, report, "")
	return b.String()
}

// Each source an edge rests on is named "read from <source> in this run"
// when its answer did not come from a cache, and "read from <source> at
// <fetched_at>" when it did, in the order of the edge's sources, after the
// edge's instant and basis, each with its place in its network's order,
// since its network has several sources. Mutants killed: the two wordings
// swapped; the place written as the source's index among the edge's
// sources.
func TestRenderReport_EachSourceAnEdgeRestsOn(t *testing.T) {
	fetched := time.Date(2026, 9, 26, 2, 0, 0, 123*int(time.Millisecond), time.UTC)
	edge := &verifier.Edge{
		Assumption: "Stellar validators did not set the ledger's close time behind real time; a close time set ahead of real time only makes this edge later",
		Candidate:  verifier.Candidate{Basis: "stellar_ledger_close_read", Class: verifier.ClassTIME, Instant: 1790387835, FetchedAt: fetched},
		Part:       "/commitments/stellar",
		Sources: []verifier.EdgeSource{
			{Source: "https://horizon-one.example", FetchedAt: fetched, Place: 1, Count: 3},
			{Source: "https://horizon-two.example", Cached: true, FetchedAt: fetched.Add(-time.Hour), Place: 3, Count: 3},
		},
	}
	report := &verifier.Report{
		Attribution: "keyring:" + strings.Repeat("ab", 32), Environment: "test", Label: verifier.LabelSuppliedKeyring,
		SubmittedBefore: edge, TrustRoot: verifier.TrustRoot{Kind: verifier.KindSuppliedKeyring, Location: "/k.json"},
		Type: "item", SubjectID: "01a0db6d-2b80-7928-9225-f8760ae1a795", Window: verifier.WindowEstablished,
	}
	want := "Submitted before: 2026-09-26T01:57:15Z (stellar_ledger_close_read, " +
		"read from https://horizon-one.example, source 1 of the 3 sources of its network, in this run; " +
		"read from https://horizon-two.example, source 3 of the 3 sources of its network, at 2026-09-26T01:00:00.123Z)\n"
	if text := rendered(report); !strings.Contains(text, want) {
		t.Errorf("the edge's line is not %q:\n%s", want, text)
	}
}

// Two sources of one network on one scheme, host and port, which a report
// shows alike, are told apart in the edge's text by their places in the
// network's order (SPEC.md section 11.5), and a source that is its
// network's only one is named with no place. Mutants killed: edgeText
// dropping the place (the two sources read alike); the place named for a
// network with one source.
func TestRenderReport_TwoSourcesOnOneHostToldApart(t *testing.T) {
	fetched := time.Date(2026, 9, 26, 2, 0, 0, 0, time.UTC)
	before := &verifier.Edge{
		Assumption: "at least 6 of blocks height + 1 to height + 10 carry honest times",
		Candidate:  verifier.Candidate{Basis: "bitcoin_median_time_read", Class: verifier.ClassTIME, Instant: 1790387835, FetchedAt: fetched},
		Part:       "/commitments/bitcoin",
		Sources: []verifier.EdgeSource{
			{Source: "https://btc.example", FetchedAt: fetched, Place: 1, Count: 2},
			{Source: "https://btc.example", FetchedAt: fetched.Add(time.Millisecond), Place: 2, Count: 2},
		},
	}
	after := &verifier.Edge{
		Assumption: "NIST does not release outputValue before timeStamp",
		Candidate:  verifier.Candidate{Basis: "nist_pulse_time_read", Class: verifier.ClassNISTTIME, Instant: 1790380800000, FetchedAt: fetched},
		Part:       "/entropy/nist",
		Sources:    []verifier.EdgeSource{{Source: "https://beacon.example", FetchedAt: fetched, Place: 1, Count: 1}},
	}
	report := &verifier.Report{
		Attribution: "keyring:" + strings.Repeat("ab", 32), Environment: "test", Label: verifier.LabelSuppliedKeyring,
		SubmittedAfter: after, SubmittedBefore: before, TrustRoot: verifier.TrustRoot{Kind: verifier.KindSuppliedKeyring, Location: "/k.json"},
		Type: "item", SubjectID: "01a0db6d-2b80-7928-9225-f8760ae1a795", Window: verifier.WindowEstablished,
	}
	text := rendered(report)
	for _, want := range []string{
		"Submitted before: 2026-09-26T01:57:15Z (bitcoin_median_time_read, " +
			"read from https://btc.example, source 1 of the 2 sources of its network, in this run; " +
			"read from https://btc.example, source 2 of the 2 sources of its network, in this run)\n",
		"(nist_pulse_time_read, read from https://beacon.example in this run)\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the text has no line ending %q:\n%s", want, text)
		}
	}
}

// An edge that rests on one source of a network that has several, as under
// the policy first, still names that source's place (SPEC.md section
// 11.5). Mutant killed: the place named only when the edge rests on more
// than one source (len(edge.Sources) > 1).
func TestRenderReport_OneOfSeveralSourcesNamesItsPlace(t *testing.T) {
	fetched := time.Date(2026, 9, 26, 2, 0, 0, 0, time.UTC)
	before := &verifier.Edge{
		Assumption: "at least 6 of blocks height + 1 to height + 10 carry honest times",
		Candidate:  verifier.Candidate{Basis: "bitcoin_median_time_read", Class: verifier.ClassTIME, Instant: 1790387835, FetchedAt: fetched},
		Part:       "/commitments/bitcoin",
		Sources:    []verifier.EdgeSource{{Source: "https://btc.example", FetchedAt: fetched, Place: 2, Count: 2}},
	}
	report := &verifier.Report{
		Attribution: "keyring:" + strings.Repeat("ab", 32), Environment: "test", Label: verifier.LabelSuppliedKeyring,
		SubmittedBefore: before, TrustRoot: verifier.TrustRoot{Kind: verifier.KindSuppliedKeyring, Location: "/k.json"},
		Type: "item", SubjectID: "01a0db6d-2b80-7928-9225-f8760ae1a795", Window: verifier.WindowEstablished,
	}
	want := "Submitted before: 2026-09-26T01:57:15Z (bitcoin_median_time_read, " +
		"read from https://btc.example, source 2 of the 2 sources of its network, in this run)\n"
	if text := rendered(report); !strings.Contains(text, want) {
		t.Errorf("the text has no line ending %q:\n%s", want, text)
	}
}

// replayOutcome runs the online case id in this process, its keyring the
// vectors' test keyring, each network its replay file holds a read of
// with one source the file answers for, as the conformance runner does
// for a case with one source per network (SPEC.md section 13.2), and
// expected as the caller's hash.
func replayOutcome(t *testing.T, id string, expected *string) verifier.Outcome {
	t.Helper()
	c := vectorCase(t, id)
	replay, err := vectorcases.LoadCaseReplay(c)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := verifier.NewConfig(c.Config.Environment)
	if err != nil {
		t.Fatal(err)
	}
	pins := c.Config.Networks
	cfg.Networks = specdata.NetworkPins{
		Commitment: specdata.ChainNetworks{Bitcoin: pins.Commitment["bitcoin"], Stellar: pins.Commitment["stellar"]},
		Entropy:    specdata.ChainNetworks{Bitcoin: pins.Entropy["bitcoin"], Stellar: pins.Entropy["stellar"]},
	}
	cfg.Online, cfg.ExpectedHash = true, expected
	cfg.Keyring = verifier.KeyringSource{Kind: verifier.SourceFile, Path: testKeyring()}
	cfg.ReadIdentity = func(s verifier.Source) (string, error) { return replay.Identity(s.Chain, s.Network, "") }
	cfg.ReadChain = func(_ verifier.Source, read map[string]any) verifier.SourceRead {
		answer, held := replay.Answer(read, "")
		if !held {
			t.Errorf("%s: a read the replay file does not hold: %v", id, read)
			return verifier.SourceRead{Outcome: verifier.ReadUnreachable, Reason: "no answer"}
		}
		return verifier.SourceRead{Outcome: verifier.ReadOutcome(answer.Outcome), Value: answer.Value, Reason: answer.Reason}
	}
	for _, s := range replay.Sources() {
		provider := source.Esplora
		if s.Chain == verifier.ChainStellar {
			provider = source.Horizon
		}
		cfg.NetworkSources = append(cfg.NetworkSources, verifier.NetworkSources{
			Chain: s.Chain, Network: s.Network, Policy: verifier.SourcePolicy{Kind: verifier.SourcePolicyFirst},
			Sources: []verifier.Source{{Chain: s.Chain, Provider: provider, URL: "https://replay.invalid/" + s.Chain + "/" + s.Network}},
		})
	}
	return verifier.Verify(cfg, func() ([]byte, error) { return strictjson.ReadFile(c.ProofPath()) })
}

// windowEdges are the instants of the edges the run's WINDOW row says it
// drew, which a failed report never shows.
func windowEdges(t *testing.T, report *verifier.Report) []string {
	t.Helper()
	var out []string
	for _, row := range report.Rows {
		if row.Step != "WINDOW" {
			continue
		}
		edges, _ := row.Values["edges"].(map[string]any)
		for _, edge := range edges {
			instant, _ := edge.(map[string]any)["instant"].(string)
			out = append(out, instant)
		}
	}
	return out
}

// A run failed by window_inverted drew both edges, as its WINDOW row
// says, and the text shows neither, nor any window: the report's edge
// objects are null when the label is failed, and the text draws edges
// from them alone, and lists the WINDOW row without its message, which
// names both (SPEC.md section 11.5). Mutants killed: windowText without
// its check of the label (the text shows "Submitted after" lines);
// rowsText writing a failed run's WINDOW message (the text shows the
// row's instants).
func TestRenderReport_AWindowInvertedRunShowsNoEdge(t *testing.T) {
	outcome := replayOutcome(t, "window-inverted", nil)
	report := outcome.Report
	if report == nil || report.Label != verifier.LabelFailed || report.Code != "window_inverted" || report.SubmittedAfter != nil || report.SubmittedBefore != nil {
		t.Fatalf("the run: %+v", outcome)
	}
	instants := windowEdges(t, report)
	if len(instants) != 2 {
		t.Fatalf("the WINDOW row drew %q, want two edges, so this test shows nothing", instants)
	}
	text := rendered(report)
	if !strings.Contains(text, "Window:       none is shown, since a check failed\n") || strings.Contains(text, "Submitted") {
		t.Errorf("the text shows a window:\n%s", text)
	}
	for _, instant := range instants {
		if strings.Contains(text, instant) {
			t.Errorf("the text shows the edge %s:\n%s", instant, text)
		}
	}
}

// A run that drew its submitted-before edge and then failed at
// EXPECTED_HASH, after WINDOW, shows no edge: the same case with no hash
// given is supplied-keyring and shows the edge, read from its source in
// this run, and with a hash that differs it is failed, its report's edges
// null, and the text shows neither the edge nor a window. Mutant killed:
// NewReport keeping the edges of a failed run, with windowText's check of
// the label dropped (the instant reaches the text).
func TestRenderReport_AFailureAfterWindowShowsNoEdge(t *testing.T) {
	const instant = "2026-09-26T01:57:15Z"
	sound := replayOutcome(t, "reads-key_event_commitment-not_attributable", nil)
	if sound.Report == nil || sound.Report.Label != verifier.LabelSuppliedKeyring ||
		!strings.Contains(rendered(sound.Report), "Submitted before: "+instant+" (stellar_ledger_close_read, read from https://replay.invalid in this run)\n") {
		t.Fatalf("with no hash given the edge is not shown, so this test shows nothing:\n%+v", sound.Report)
	}
	other := strings.Repeat("00", 32)
	failed := replayOutcome(t, "reads-key_event_commitment-not_attributable", &other)
	report := failed.Report
	if report == nil || report.Label != verifier.LabelFailed || report.Code != "expected_hash_mismatch" || report.SubmittedBefore != nil || report.Window != verifier.WindowNotEstablished {
		t.Fatalf("the failed run: %+v", failed)
	}
	if text := rendered(report); strings.Contains(text, instant) || strings.Contains(text, "Submitted") || !strings.Contains(text, "Window:       none is shown") {
		t.Errorf("the text of a run failed after WINDOW shows a window:\n%s", text)
	}
}

// In a run that is not failed, the two WINDOW rows whose message names an
// instant are listed without it: window_not_established, whose message
// names the submitted-after edge (reads-stellar-ledger_not_found), and
// recorded_time_outside_window, whose message names the time inside the
// subject's id (window-id_later_than_submitted_before). No line of the
// text but its "Submitted after" and "Submitted before" lines holds an
// instant (SPEC.md section 11.5). Mutant killed: rowsText writing these
// rows' messages (the text repeats the edge, and shows the id's time).
func TestRenderReport_NoWindowMessageRepeatsAnInstant(t *testing.T) {
	instant := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
	for _, c := range []struct {
		id   string
		code specdata.Reason
	}{
		{"reads-stellar-ledger_not_found", specdata.ReasonWindowNotEstablished},
		{"window-id_later_than_submitted_before", specdata.ReasonRecordedTimeOutsideWindow},
	} {
		report := replayOutcome(t, c.id, nil).Report
		if report == nil || report.Label == verifier.LabelFailed {
			t.Fatalf("%s: the run %+v", c.id, report)
		}
		at := slices.IndexFunc(report.Rows, func(row verifier.Row) bool { return row.Step == "WINDOW" && row.Code == c.code })
		if at < 0 || !instant.MatchString(report.Rows[at].Message) {
			t.Fatalf("%s: no WINDOW %s row whose message names an instant, so this test shows nothing", c.id, c.code)
		}
		text := rendered(report)
		if !strings.Contains(text, "  "+string(report.Rows[at].Status)+" WINDOW "+string(c.code)) {
			t.Errorf("%s: the text does not list the row:\n%s", c.id, text)
		}
		for _, line := range strings.Split(text, "\n") {
			if instant.MatchString(line) && !strings.HasPrefix(line, "Submitted after: ") && !strings.HasPrefix(line, "Submitted before:") {
				t.Errorf("%s: an instant outside the edge lines: %q", c.id, line)
			}
		}
	}
}
