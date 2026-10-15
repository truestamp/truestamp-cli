// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.Check
// (elixir/lib/truestamp/verifier/check.ex in truestamp-verification),
// checked against the Truestamp application's port,
// Truestamp.Proof.Verifier.Check (lib/truestamp/proof/verifier/check.ex in
// truestamp-v2): one check of DERIVE (SPEC.md section 10.2, step 9), as the
// one engine of section 10.6 takes it. [Derive]'s engine turns each check
// into its report row and, when the check is traced, its trace row.
//
//	| Member       | Holds                                                                                              |
//	|--------------|----------------------------------------------------------------------------------------------------|
//	| part         | the check's part (section 10.2's one rule): the carried value the check finds wrong when it fails, which for a recomputation is the declared value |
//	| construction | the name of the row of the constructions text (section 7.3) that defines the value, one of deriveConstructions |
//	| inputs       | the pointers of the values the check reads, for reading only, in document order                    |
//	| declared     | the carried value the check compares: the value at part                                            |
//	| recomputed   | what the check compares declared with, as its table defines it (section 10.6), or nil where a walk gives no root |
//	| code         | "" for a pass, and the failure's reason code otherwise                                             |
//	| traced       | whether the check gives a trace row: its part is a CHECKED value (section 3.7), and it is not a row of LINK_HEAD, LINK_KEY_EVENT or LINK_CHAIN_ORDER (section 10.6) |
//	| message      | the row's message, for display only                                                                |
//
// A row's values (SPEC.md section 10.5). A pass row holds construction. A
// failing row holds construction, declared, the carried value the check
// found wrong, and recomputed, what it was compared with as section 10.6
// defines it, even when the proof carries that value, such as the record's
// leaf_count a tree_size is compared with (section 10.2, step 9).
//
// A trace row (SPEC.md section 10.6) is {construction, declared, inputs,
// part, recomputed, status}, with part and each input written as an RFC
// 6901 string, and status the status of its row, pass or fail.
//
// Adapted from the reference: a check is a struct with the code of its
// failure where the reference has an outcome; the comparison is
// [sameValue], which compares a string, a strictjson.Number and null as
// the reference's === compares them; and, as the application's port does,
// a check may name only one of the 14 rows the "Row of 7.3" columns of
// SPEC.md sections 10.2, step 9, and 10.3 give, so no check names a row
// the engine does not know. The info outcome and a failure's other values
// (a refusal's name, the rule a structure breaks) belong to the checks that
// give them: the key events, the other links and the Bitcoin evidence.

package verifier

import (
	"fmt"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// deriveStep is the name of step 9.
const deriveStep = "DERIVE"

// deriveConstructions are the names of the rows of the constructions text a
// check may name: the "Row of 7.3" columns of SPEC.md section 10.2, step 9
// ((a) claims_hash, item_hash, entropy_hash, observation_hash; (b)
// block_hash, genesis; (c) epoch_hash; (d) bitcoin), and of section 10.3
// (the links: inclusion, witnesses, height, key_event, chain_order; the
// key events: key_event, handover), sorted.
var deriveConstructions = []string{
	"bitcoin", "block_hash", "chain_order", "claims_hash", "entropy_hash", "epoch_hash", "genesis",
	"handover", "height", "inclusion", "item_hash", "key_event", "observation_hash", "witnesses",
}

// deriveCheck is one check of DERIVE (see the file's doc).
type deriveCheck struct {
	part         Pointer
	construction string
	inputs       []Pointer
	declared     any
	recomputed   any
	code         specdata.Reason
	traced       bool
	message      string
}

// equalCheck is the check that declared, the value at part, equals
// recomputed ([sameValue]): a pass when they are equal, and otherwise
// derivation_mismatch. It is traced, and noun names the value for its
// message. It panics for a construction that is not one of
// deriveConstructions: a defect in the caller.
func equalCheck(part Pointer, construction string, inputs []Pointer, declared, recomputed any, noun string) deriveCheck {
	if !slices.Contains(deriveConstructions, construction) {
		panic(fmt.Sprintf("verifier: %q is not a row DERIVE names", construction))
	}
	inputs = slices.Clone(inputs)
	SortBy(inputs, func(p Pointer) Pointer { return p })
	c := deriveCheck{
		part: part, construction: construction, inputs: inputs,
		declared: declared, recomputed: recomputed, traced: true,
		message: fmt.Sprintf("%s at %s equals the value this run recomputed for it.", noun, part.Encode()),
	}
	if !sameValue(declared, recomputed) {
		c.code = specdata.ReasonDerivationMismatch
		c.message = fmt.Sprintf("%s at %s is not the value this run recomputed for it, by the constructions row %s.",
			noun, part.Encode(), construction)
	}
	return c
}

// sameValue reports whether a and b are one value: two strings or two
// numbers that are equal, or both null. A value of any other kind is equal
// to nothing, so no comparison of DERIVE holds over one.
func sameValue(a, b any) bool {
	switch x := a.(type) {
	case nil:
		return b == nil
	case string:
		y, ok := b.(string)
		return ok && x == y
	case strictjson.Number:
		y, ok := b.(strictjson.Number)
		return ok && x == y
	}
	return false
}

// row is the check's report row (SPEC.md section 10.5).
func (c deriveCheck) row() Row {
	part := Part(c.part.Encode())
	values := map[string]any{"construction": c.construction}
	if c.code == "" {
		return PassRow(deriveStep, part, c.message, values)
	}
	values["declared"], values["recomputed"] = c.declared, c.recomputed
	return NewRow(deriveStep, c.code, part, c.message, values)
}

// TraceRow is one row of a run's trace (SPEC.md section 10.6): the
// constructions row that defines the value, the value the proof carries,
// the pointers of the values the check read, in document order, the part
// of its DERIVE row, what the check compared the declared value with (nil
// when a walk gives no root), and the status of its DERIVE row, pass or
// fail.
type TraceRow struct {
	Construction string
	Declared     any
	Inputs       []string
	Part         string
	Recomputed   any
	Status       Status
}

// Value returns the trace row as SPEC.md section 10.6 writes it, a value
// internal/jcs writes: {construction, declared, inputs, part, recomputed,
// status}.
func (r TraceRow) Value() map[string]any {
	inputs := make([]any, len(r.Inputs))
	for i, input := range r.Inputs {
		inputs[i] = input
	}
	return map[string]any{
		"construction": r.Construction, "declared": r.Declared, "inputs": inputs,
		"part": r.Part, "recomputed": r.Recomputed, "status": string(r.Status),
	}
}

// traceRow is the check's trace row. Only a traced check has one.
func (c deriveCheck) traceRow() TraceRow {
	inputs := make([]string, len(c.inputs))
	for i, input := range c.inputs {
		inputs[i] = input.Encode()
	}
	status := StatusPass
	if c.code != "" {
		status = StatusFail
	}
	return TraceRow{
		Construction: c.construction, Declared: c.declared, Inputs: inputs,
		Part: c.part.Encode(), Recomputed: c.recomputed, Status: status,
	}
}
