// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"encoding/json"
	"testing"
)

// WINDOW under a closed gate (SPEC.md section 11.3): the run read nothing,
// so it draws no edge; its one row, part null, is skip prior_failure when
// a row failed and otherwise info window_not_established, and its values
// give no candidate, no edge, and for each absent edge the part and code
// of each row that could have given a candidate: under submitted_before
// the top-level entries' CHAIN rows only, never a /signing_key_event entry,
// a genesis.bitcoin or KEY_EVENT_COMMITMENT (section 11.1), and under
// submitted_after every SOURCES row, none for a block proof (section
// 11.2). Mutants killed: Window giving window_not_established after a
// failure (the failed run's row); absentReasons taking every CHAIN row
// (the signing key event's entries and the genesis row are listed);
// absentReasons leaving the submitted-after list null for a block proof
// (it is written null, not []).
func TestWindow_ClosedGate(t *testing.T) {
	proofs := positiveProofs(t)
	for _, c := range []struct {
		proof  string
		failed bool
		row    string
		values string
	}{
		{"item", false, "[window_not_established null info WINDOW]",
			`{"absent":{"submitted_after":[{"code":"offline","part":"/witnesses/entropy_bitcoin"},{"code":"offline","part":"/witnesses/entropy_nist"},{"code":"offline","part":"/witnesses/entropy_stellar"}],` +
				`"submitted_before":[{"code":"offline","part":"/commitments/bitcoin"},{"code":"offline","part":"/commitments/stellar"}]},"candidates":[],"edges":{}}`},
		{"item", true, "[prior_failure null skip WINDOW]",
			`{"absent":{"submitted_after":[{"code":"prior_failure","part":"/witnesses/entropy_bitcoin"},{"code":"prior_failure","part":"/witnesses/entropy_nist"},{"code":"prior_failure","part":"/witnesses/entropy_stellar"}],` +
				`"submitted_before":[{"code":"prior_failure","part":"/commitments/bitcoin"},{"code":"prior_failure","part":"/commitments/stellar"}]},"candidates":[],"edges":{}}`},
		{"block", false, "[window_not_established null info WINDOW]",
			`{"absent":{"submitted_after":[],"submitted_before":[{"code":"offline","part":"/commitments/bitcoin"},{"code":"offline","part":"/commitments/stellar"}]},"candidates":[],"edges":{}}`},
	} {
		var extra []Row
		if c.failed {
			extra = []Row{policyFailure}
		}
		run := throughDerive(proofs[c.proof], false, extra...)
		Chain(run)
		Sources(run)
		Window(run)
		row := run.Rows[len(run.Rows)-1]
		values, err := json.Marshal(row.Values)
		if got := rowSummary([]Row{row}); got != c.row || err != nil || string(values) != c.values {
			t.Errorf("%s, failed %v: %s, values %s, %v\nwant %s, %s", c.proof, c.failed, got, values, err, c.row, c.values)
		}
	}
}
