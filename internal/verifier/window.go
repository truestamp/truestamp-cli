// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds step 12 of SPEC.md section 10.2, WINDOW, the row of
// section 11.3, for a run whose external gate is closed.
//
// WINDOW gives the first of these rows that applies, with a null part
// (SPEC.md section 11.3): skip prior_failure when a row has failed; fail
// window_inverted when both edges exist and the submitted-after edge is
// later; info window_not_established when the submitted-before edge does
// not exist; and otherwise pass. A run whose gate is closed read nothing,
// so it has no candidate and draws neither edge (SPEC.md section 10.4,
// "Offline"): its row is skip prior_failure when a row has failed, and
// otherwise info window_not_established.
//
// The row's values give every candidate the run read, each edge that
// exists and, for each edge that does not, its reason: the part and code
// of each CHAIN or SOURCES row that could have given a candidate, a list
// that is empty when the proof carries none, as for a block proof's
// submitted-after edge (SPEC.md section 11.3). The submitted-before edge's
// rows are those of the top-level entries, /commitments/C (section 11.1),
// and the submitted-after edge's every SOURCES row (section 11.2). Under a
// closed gate, candidates and edges are empty.
//
// Under an open gate the window is drawn from the candidates of the CHAIN
// and SOURCES rows that passed, which the rules of each chain and source
// give (section 11); no such rule is in this package, so under an open
// gate WINDOW gives no row here. A run that has ended passes through, since
// its WINDOW row is already skip prior_failure.

package verifier

import (
	"strings"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// Window is step 12, WINDOW (see the file's doc).
func Window(run *Run) {
	if run.Ended {
		return
	}
	if _, closed := closedGate(run, "WINDOW"); !closed {
		return
	}
	values := map[string]any{
		"absent":     map[string]any{"submitted_after": absentReasons(run, "SOURCES"), "submitted_before": absentReasons(run, "CHAIN")},
		"candidates": []any{},
		"edges":      map[string]any{},
	}
	if run.Failed() {
		run.AddRow(NewRow("WINDOW", specdata.ReasonPriorFailure, nil, "Not drawn: a check of this run failed, so it gives no submission window.", values))
		return
	}
	run.AddRow(NewRow("WINDOW", specdata.ReasonWindowNotEstablished, nil,
		"No submitted-before edge was drawn: this run is offline, so it read no chain or source, and draws no edge.", values))
}

// absentReasons are the part and code of each row of step that could have
// given a candidate: under CHAIN, the rows of the top-level entries.
func absentReasons(run *Run, step string) []any {
	reasons := []any{}
	for _, row := range run.Rows {
		if row.Step != step || row.Part == nil {
			continue
		}
		if step == "CHAIN" && !isTopLevelEntry(*row.Part) {
			continue
		}
		reasons = append(reasons, map[string]any{"code": string(row.Code), "part": *row.Part})
	}
	return reasons
}

// isTopLevelEntry reports whether part is a top-level commitment entry,
// /commitments/C.
func isTopLevelEntry(part string) bool {
	chain, ok := strings.CutPrefix(part, "/commitments/")
	return ok && (chain == ChainBitcoin || chain == ChainStellar)
}
