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
// internally-consistent unless the WINDOW row passed, which is verified;
// a skip, warn or info row is no failure, and a run that is not failed has
// no code. Mutants killed: Verdict taking the last failing row's code (the
// two failures give derivation_mismatch); failed only for a fail at a
// given step (the EXPECTED_HASH failure is supplied-keyring); verified
// whatever the window (the truestamp run without a passing WINDOW row is
// verified); the attribution read before the failures (the failed
// unattributed run is unattributed); a warn row counted as a failure (the
// weak match is failed); Verdict leaving the code of an earlier call (the
// second call on the healed run keeps a code); the edge read from any
// WINDOW row (window_not_established gives verified).
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
	for _, c := range []struct {
		name        string
		attribution string
		rows        []Row
		label       string
		code        specdata.Reason
	}{
		{"two failures", deriveAttribution, with(policyFail, deriveFail), LabelFailed, specdata.ReasonNetworkMismatch},
		{"a failure at EXPECTED_HASH", deriveAttribution, with(notEstablished, mismatch), LabelFailed, specdata.ReasonExpectedHashMismatch},
		{"unattributed, failed", AttributionUnattributed, with(deriveFail), LabelFailed, specdata.ReasonDerivationMismatch},
		{"unattributed", AttributionUnattributed, with(info, notEstablished, weak), LabelUnattributed, ""},
		{"supplied keyring", deriveAttribution, with(info, notEstablished), LabelSuppliedKeyring, ""},
		{"truestamp, no edge", AttributionTruestamp, with(notEstablished), LabelInternallyConsistent, ""},
		{"truestamp, no WINDOW row", AttributionTruestamp, with(), LabelInternallyConsistent, ""},
		{"truestamp, an edge", AttributionTruestamp, with(pass("WINDOW")), LabelVerified, ""},
		{"supplied keyring, an edge", deriveAttribution, with(pass("WINDOW")), LabelSuppliedKeyring, ""},
	} {
		run := &Run{Attribution: c.attribution, Rows: c.rows}
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
