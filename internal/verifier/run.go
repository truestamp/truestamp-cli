// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.Run
// (elixir/lib/truestamp/verifier/run.ex in truestamp-verification),
// checked against the Truestamp application's port,
// Truestamp.Proof.Verifier.Run (lib/truestamp/proof/verifier/run.ex in
// truestamp-v2): the state of a run (SPEC.md section 10.2), the rows so
// far and what the later steps read. CONFIGURATION ([Configure]) creates
// the run from a [Config]; READ, PARSE and ENVELOPE each take it, add their
// row and give nil, or give a bare rejection. So a run that reaches the
// caller has passed steps 0 to 3.
//
// Adapted from the reference: the CLI runs online, so the run says when
// the keyring that would govern could not be read (KEY_BINDING then fails
// keyring_unavailable whatever K is, SPEC.md section 8.3), and trust_root
// takes the kind current_keyring, which the always-offline reference never
// gives. The members the later steps set (the bound key, proof_hash, the
// attribution, the type and subject id, the trace, the external gate, the
// label and the edges) join with the steps that set them, and so does
// trust_root's digest.

package verifier

import (
	"slices"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier/keyring"
)

// steps are the steps of SPEC.md section 10.2, 0 to 14, by name, in order.
var steps = []string{
	"CONFIGURATION", "READ", "PARSE", "ENVELOPE", "ENVIRONMENT", "KEY_BINDING",
	"SIGNATURE", "SHAPE", "POLICY", "DERIVE", "CHAIN", "SOURCES", "WINDOW",
	"EXPECTED_HASH", "VERDICT",
}

// Steps returns the steps of SPEC.md section 10.2, 0 to 14, by name, in
// order. Each call returns a new slice.
func Steps() []string { return slices.Clone(steps) }

// TrustRootKind is trust_root's kind (SPEC.md section 10.5), as section 8.3
// assigns it to an external verifier's keyring.
type TrustRootKind string

// The kinds of an external verifier's trust root.
const (
	KindCurrentKeyring  TrustRootKind = "current_keyring"
	KindCompiledPin     TrustRootKind = "compiled_pin"
	KindSuppliedKeyring TrustRootKind = "supplied_keyring"
	KindNone            TrustRootKind = "none"
)

// TrustRoot is where the keys that govern key binding came from: its kind,
// and Location, the URL or file they came from, or "" when they came from
// neither.
type TrustRoot struct {
	Kind     TrustRootKind
	Location string
}

// Run is the state of a run.
//
//	| Member              | Holds                                                       | Set at        |
//	|---------------------|-------------------------------------------------------------|---------------|
//	| Config              | the run's configuration                                     | CONFIGURATION |
//	| Keyring             | the keyring that governs key binding; nil in keyless mode, and when it could not be read | CONFIGURATION |
//	| KeyringUnavailable  | an online read of the keyring that would govern failed, so no key binds | CONFIGURATION |
//	| TrustRoot           | trust_root's kind and location (SPEC.md section 10.5)       | CONFIGURATION |
//	| Document            | the parsed document                                         | PARSE         |
//	| Proof               | P, the parsed value of proof                                | ENVELOPE      |
//	| PublicKey, Signature| the 32 octets of K and the 64 of V                          | ENVELOPE      |
//	| Rows                | the rows so far, in step order                              | every step    |
//	| Ended               | whether a failure ended the run                             | a failure at steps 4 to 7, or a caught fault from ENVIRONMENT on |
type Run struct {
	Config             Config
	Keyring            *keyring.Keyring
	KeyringUnavailable bool
	TrustRoot          TrustRoot
	Document           any
	Proof              map[string]any
	PublicKey          [32]byte
	Signature          [64]byte
	Rows               []Row
	Ended              bool
}

// AddRow adds row after the run's rows.
func (r *Run) AddRow(row Row) { r.Rows = append(r.Rows, row) }

// Failed reports whether any row so far is fail.
func (r *Run) Failed() bool {
	return slices.ContainsFunc(r.Rows, func(row Row) bool { return row.Status == StatusFail })
}

// EndRun ends the run after a failure at steps 4 to 7, or after a fault
// caught from ENVIRONMENT on: one row skip prior_failure, part null, for
// each step after the last row's up to EXPECTED_HASH (SPEC.md sections 10.2
// and 10.5). Every later step passes an ended run through unchanged.
func (r *Run) EndRun() {
	last := 0
	if n := len(r.Rows); n > 0 {
		last = slices.Index(rowSteps, r.Rows[n-1].Step)
	}
	for _, step := range rowSteps[last+1:] {
		r.AddRow(NewRow(step, specdata.ReasonPriorFailure, nil, "Not made: a check of an earlier step failed.", nil))
	}
	r.Ended = true
}
