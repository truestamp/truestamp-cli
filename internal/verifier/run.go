// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the state of a run (SPEC.md section 10.2), the rows so
// far and what the later steps read. CONFIGURATION ([Configure]) creates
// the run from a [Config]; READ, PARSE and ENVELOPE each take it, add their
// row and give nil, or give a bare rejection. So a run that reaches the
// caller has passed steps 0 to 3. ENVIRONMENT, KEY_BINDING, SIGNATURE and
// SHAPE each take it and add their row; a failure at any of them ends it
// ([Run.EndRun]), and a SHAPE that passes gives the run its type and
// subject id. POLICY takes it and adds a row for each network of the
// proof; a failure there does not end it. DERIVE takes it, reads K and the
// attribution for LINK_KEY_EVENT, adds its rows and sets its trace; a
// failure there does not end it either, and its info row is no failure
// ([Run.Failed]).
//
// The CLI runs online, so the run says when the keyring that would govern
// could not be read (KEY_BINDING then fails keyring_unavailable whatever K
// is, SPEC.md section 8.3), trust_root takes the kind current_keyring under
// the keyring read from www.truestamp.com (section 10.5), and the run holds
// the outcome of each source's identity read and each pinned network's
// sources and source policy (SPEC.md sections 9 and 10.2, step 0). Its
// members are those steps 0 to 9 set; it holds no external gate, label or
// edges.

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

// TrustRoot is trust_root, {digest, kind, location} (SPEC.md section
// 10.5): Digest, d, the lowercase hex SHA-256 of the bytes of the keyring
// that governs key binding as read (section 8.3), or "", which a report
// writes as null, in keyless mode and when the keyring could not be read;
// Kind; and Location, the URL or file the keys came from, or "" when they
// came from neither, which a report writes as null.
type TrustRoot struct {
	Digest   string
	Kind     TrustRootKind
	Location string
}

// The attributions of SPEC.md section 10.5 that name no keyring; a
// supplied keyring's is "keyring:" and its digest.
const (
	AttributionTruestamp    = "truestamp"
	AttributionUnattributed = "unattributed"
	AttributionNone         = "none"
)

// Run is the state of a run.
//
//	| Member              | Holds                                                       | Set at        |
//	|---------------------|-------------------------------------------------------------|---------------|
//	| Config              | the run's configuration                                     | CONFIGURATION |
//	| Keyring             | the keyring that governs key binding; nil in keyless mode, and when it could not be read | CONFIGURATION |
//	| KeyringUnavailable  | an online read of the keyring that would govern failed, so no key binds | CONFIGURATION |
//	| TrustRoot           | trust_root, {digest, kind, location} (SPEC.md section 10.5) | CONFIGURATION |
//	| Sources             | in an online run, each Stellar and Bitcoin source, confirmed or unreachable, with the pins it serves (sources.go); empty in an offline run | CONFIGURATION |
//	| SourceSets          | in an online run, each pinned Stellar and Bitcoin network's sources, in their order, and its source policy (sources.go); empty in an offline run | CONFIGURATION |
//	| Document            | the parsed document                                         | PARSE         |
//	| Proof               | P, the parsed value of proof                                | ENVELOPE      |
//	| PublicKey, Signature| the 32 octets of K and the 64 of V                          | ENVELOPE      |
//	| BoundKey            | the key of the keyring whose 32 octets equal K's, or nil     | KEY_BINDING   |
//	| ProofHash           | proof_hash, computed from P as parsed                       | SIGNATURE     |
//	| Attribution         | attribution (SPEC.md section 10.5): truestamp, keyring:<d> or unattributed when SIGNATURE passes, none when the gate fails, "" before | SIGNATURE, or a failure at steps 4 to 6 |
//	| Type                | the proof's type, which SHAPE has held to the type registry; "" until SHAPE passes, which a report writes as null | SHAPE |
//	| SubjectID           | the subject's id: /subject/record/id, or /block/record/id for block and beacon (SPEC.md section 10.5); "" until SHAPE passes, which a report writes as null | SHAPE |
//	| Trace               | the trace (SPEC.md section 10.6): one row for each DERIVE row whose part is a CHECKED value, other than a row of LINK_HEAD, LINK_KEY_EVENT or LINK_CHAIN_ORDER, in DERIVE's order; nil until DERIVE, for a run that ends before it, and after a fault in it | DERIVE |
//	| Rows                | the rows so far, in step order                              | every step    |
//	| Ended               | whether a failure ended the run                             | a failure at steps 4 to 7, or a caught fault from ENVIRONMENT on |
type Run struct {
	Config             Config
	Keyring            *keyring.Keyring
	KeyringUnavailable bool
	TrustRoot          TrustRoot
	Sources            []SourceRecord
	SourceSets         []SourceSet
	Document           any
	Proof              map[string]any
	PublicKey          [32]byte
	Signature          [64]byte
	BoundKey           *keyring.Key
	ProofHash          [32]byte
	Attribution        string
	Type               string
	SubjectID          string
	Trace              []TraceRow
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
