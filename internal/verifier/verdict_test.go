// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// Each label of SPEC.md section 10.5, the first that applies: failed when
// any row fails, with the run's code the first failing row's; then by the
// attribution: unattributed, supplied-keyring, and under truestamp
// internally-consistent unless WINDOW drew a submitted-before edge, which
// is verified; a skip, warn or info row is no failure, and a run that is
// not failed has no code. Mutants killed: Verdict taking the last failing
// row's code (the two failures give derivation_mismatch); failed only for
// a fail at a given step (the EXPECTED_HASH failure after a passing
// WINDOW is verified); verified whatever the window (the truestamp run
// with no edge is verified); verified on either edge (the truestamp run
// with only a submitted-after edge is verified); the attribution read
// before the failures (the failed unattributed run is unattributed); a
// warn row counted as a failure (the weak match is failed); Verdict
// leaving the code of an earlier call (the second call on the healed run
// keeps a code); the edge read from the WINDOW row whatever its status
// (window_not_established gives verified).
func TestVerdict_Labels(t *testing.T) {
	pass := func(step string) Row { return PassRow(step, nil, "held", nil) }
	gate := []Row{pass("CONFIGURATION"), pass("READ"), pass("PARSE"), pass("ENVELOPE"), pass("ENVIRONMENT"), pass("KEY_BINDING"), pass("SIGNATURE")}
	with := func(rows ...Row) []Row { return append(append([]Row{}, gate...), rows...) }
	policyFail := NewRow("POLICY", specdata.ReasonNetworkMismatch, Part("/commitments/bitcoin/network"), "no", nil)
	deriveFail := NewRow("DERIVE", specdata.ReasonDerivationMismatch, Part("/subject/item_hash"), "no", nil)
	notEstablished := NewRow("WINDOW", specdata.ReasonWindowNotEstablished, nil, "no edge", nil)
	weak := NewRow("EXPECTED_HASH", specdata.ReasonExpectedHashWeakAlgorithm, Part("/subject/claims/hash"), "weak", nil)
	mismatch := NewRow("EXPECTED_HASH", specdata.ReasonExpectedHashMismatch, Part("/subject/claims/hash"), "differs", nil)
	info := NewRow("DERIVE", specdata.ReasonNotAttributable, Part("/signing_key_event/block/record/metadata/key_event/public_key"), "info", nil)
	before, after := testEdge("/commitments/stellar"), testEdge("/witnesses/entropy_stellar")
	for _, c := range []struct {
		name          string
		attribution   string
		rows          []Row
		after, before *Edge
		label         string
		code          specdata.Reason
	}{
		{"two failures", deriveAttribution, with(policyFail, deriveFail), nil, nil, LabelFailed, specdata.ReasonNetworkMismatch},
		{"a failure at EXPECTED_HASH after an edge", AttributionTruestamp, with(pass("WINDOW"), mismatch), after, before, LabelFailed, specdata.ReasonExpectedHashMismatch},
		{"unattributed, failed", AttributionUnattributed, with(deriveFail), nil, nil, LabelFailed, specdata.ReasonDerivationMismatch},
		{"unattributed", AttributionUnattributed, with(info, notEstablished, weak), nil, nil, LabelUnattributed, ""},
		{"unattributed, an edge", AttributionUnattributed, with(pass("WINDOW")), nil, before, LabelUnattributed, ""},
		{"supplied keyring", deriveAttribution, with(info, notEstablished), nil, nil, LabelSuppliedKeyring, ""},
		{"supplied keyring, an edge", deriveAttribution, with(pass("WINDOW")), after, before, LabelSuppliedKeyring, ""},
		{"truestamp, no edge", AttributionTruestamp, with(notEstablished), nil, nil, LabelInternallyConsistent, ""},
		{"truestamp, a submitted-after edge only", AttributionTruestamp, with(notEstablished), after, nil, LabelInternallyConsistent, ""},
		{"truestamp, an edge", AttributionTruestamp, with(pass("WINDOW")), nil, before, LabelVerified, ""},
		{"truestamp, both edges", AttributionTruestamp, with(pass("WINDOW"), weak), after, before, LabelVerified, ""},
	} {
		run := &Run{Attribution: c.attribution, Rows: c.rows, SubmittedAfter: c.after, SubmittedBefore: c.before}
		Verdict(run)
		if run.Label != c.label || run.Code != c.code {
			t.Errorf("%s: %q %q, want %q %q", c.name, run.Label, run.Code, c.label, c.code)
		}
		run.Rows = with(notEstablished)
		Verdict(run)
		if run.Code != "" {
			t.Errorf("%s: the healed run keeps the code %q", c.name, run.Code)
		}
	}
}

// A run the gate failed has the attribution none and is failed by its gate
// row. Mutant killed: Verdict reading the attribution first (none gives
// internally-consistent).
func TestVerdict_AGateFailure(t *testing.T) {
	run := &Run{Attribution: AttributionNone, Rows: []Row{NewRow("KEY_BINDING", specdata.ReasonKeyringUnavailable, nil, "no keyring", nil)}}
	run.EndRun()
	Verdict(run)
	if run.Label != LabelFailed || run.Code != specdata.ReasonKeyringUnavailable {
		t.Fatalf("%q %q", run.Label, run.Code)
	}
}

// Each label is spelled as SPEC.md section 10.5 spells it: "label " and the
// label are the words its row of section 12.2's exit-code table begins
// with, the ending specdata holds to spec/exit-codes.json. The shared
// vectors reach only failed, supplied-keyring and unattributed (section
// 13.3), so this test is what holds the other two. Mutants killed: a
// label misspelled (internally_consistent for internally-consistent,
// verified_ for verified).
func TestVerdict_LabelSpellings(t *testing.T) {
	for _, c := range []struct {
		label  string
		ending specdata.Ending
	}{
		{LabelVerified, specdata.EndingVerified},
		{LabelInternallyConsistent, specdata.EndingInternallyConsistent},
		{LabelSuppliedKeyring, specdata.EndingSuppliedKeyring},
		{LabelUnattributed, specdata.EndingUnattributed},
		{LabelFailed, specdata.EndingFailed},
	} {
		if got := "label " + c.label; got != string(c.ending) {
			t.Errorf("%q, want %q", got, c.ending)
		}
	}
}
