// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.Derive
// (elixir/lib/truestamp/verifier/derive.ex in truestamp-verification),
// checked against the Truestamp application's port,
// Truestamp.Proof.Verifier.Derive (lib/truestamp/proof/verifier/derive.ex in
// truestamp-v2): step 9 of SPEC.md section 10.2, DERIVE, offline, in every
// run that reaches it, and the one engine of section 10.6 that every DERIVE
// check goes through.
//
// DERIVE recomputes each value that the constructions text (section 7.3)
// defines from the preimage P carries, compares it with the value P carries
// for it, and checks every link (section 10.3). It makes every check and
// does not stop at a failure. Its checks are, in this order:
//
//	| Part of step 9                                                              | File                 |
//	|-----------------------------------------------------------------------------|----------------------|
//	| (a) subject hashes, (b) block nodes, (c) epochs                             | derive_records.go    |
//	| (b) each key event a block node carries (section 10.3), through the records | derive_key_events.go |
//	| (e) LINK_INCLUSION, LINK_WITNESS, LINK_HEAD, LINK_KEY_EVENT, LINK_CHAIN_ORDER | derive_links.go    |
//
// The Bitcoin evidence of (d) joins between the records and the links.
// Each check (derive_check.go) names its part by section 10.2's one rule:
// the carried value it finds wrong when it fails, which for a
// recomputation is the declared value.
//
// The engine, [deriveEngine], takes the checks in the order above and
// gives:
//
//   - the rows: one for each check, in the document order of their parts
//     ([SortBy], a stable sort), so that rows sharing a part keep the order
//     of their checks: the order of parts (a) to (e) and of each table's
//     rows, with LINK_INCLUSION's root rows at one merkle_root in the
//     document order of their inclusion proofs;
//   - the trace: one trace row for each traced check, whose part is a
//     CHECKED value and which is not a row of LINK_HEAD, LINK_KEY_EVENT or
//     LINK_CHAIN_ORDER, in the same order, so a root that three beacon walks
//     reach has three trace rows (section 10.6).
//
// A mismatch is derivation_mismatch unless its check names another code
// (key_event_mismatch or ordering_violation). A failure here does not end
// the run, and LINK_KEY_EVENT's info not_attributable row is no failure.
// A run that has already ended passes through unchanged, since its DERIVE
// row is already skip prior_failure, and keeps an empty trace. A preimage
// the registry refuses, or a value that is not of the class SHAPE has held
// it to (a key that does not decode, a height that is not a number, a key
// event of a type the schema does not list), which SHAPE has made
// impossible, is a fault of the verifier: one row fail internal_error, part
// null, and the run ends (section 10.5), with no trace.
//
// DERIVE reads P, K (Run.PublicKey, for LINK_KEY_EVENT) and the run's
// attribution (Run.Attribution, for the key-history row), and nothing else
// of the run: not constructions, which only SHAPE reads (section 10.6,
// rule 1), and no network or pin. The run holds its trace (Run.Trace),
// which a report gives on request (section 10.6).
//
// Adapted from the reference: the external gate that the reference decides
// when DERIVE ends is decided with the online frame.

package verifier

import (
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// Derive is step 9, DERIVE: every recomputation and every link, the
// Bitcoin evidence of (d) aside, with the run's trace (see the file's doc).
func Derive(run *Run) {
	if run.Ended {
		return
	}
	checks, err := deriveChecks(run.Proof, run.PublicKey, run.Attribution)
	if err != nil {
		run.AddRow(NewRow(deriveStep, specdata.ReasonInternalError, nil,
			"The verifier could not recompute a value of this proof.", map[string]any{"error": err.Error()}))
		run.EndRun()
		return
	}
	rows, trace := deriveEngine(checks)
	for _, row := range rows {
		run.AddRow(row)
	}
	run.Trace = trace
}

// deriveChecks are every check of DERIVE for P, in the order of step 9's
// parts and of each table's rows, before the engine orders them: the
// records' checks, the key events' among them, then the links'. publicKey
// is the 32 octets of K, and attribution the run's.
func deriveChecks(proof map[string]any, publicKey [32]byte, attribution string) ([]deriveCheck, error) {
	records, err := recordChecks(proof)
	if err != nil {
		return nil, fmt.Errorf("DERIVE: %w", err)
	}
	links, err := linkChecks(proof, publicKey, attribution)
	if err != nil {
		return nil, fmt.Errorf("DERIVE: %w", err)
	}
	return append(records, links...), nil
}

// deriveEngine is the one engine: the rows and the trace of checks, each
// in DERIVE's order. checks is not changed.
func deriveEngine(checks []deriveCheck) ([]Row, []TraceRow) {
	ordered := append([]deriveCheck(nil), checks...)
	SortBy(ordered, func(c deriveCheck) Pointer { return c.part })
	rows := make([]Row, 0, len(ordered))
	trace := []TraceRow{}
	for _, c := range ordered {
		rows = append(rows, c.row())
		if c.traced {
			trace = append(trace, c.traceRow())
		}
	}
	return rows, trace
}

// under returns base with a segment for each of names after it, never
// sharing base's array.
func under(base Pointer, names ...string) Pointer {
	out := make(Pointer, 0, len(base)+len(names))
	out = append(out, base...)
	for _, name := range names {
		out = append(out, Name(name))
	}
	return out
}
