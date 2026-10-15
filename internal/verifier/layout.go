// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.Layout
// (elixir/lib/truestamp/verifier/layout.ex in truestamp-verification):
// reading the value at a pointer ([Pointer]) into P, the parsed value of
// proof, where P holds one (SPEC.md sections 3.3 to 3.6). SHAPE reads the
// subject's id through it once check 3 has held P to the schema (SPEC.md
// section 10.2, step 7, and section 10.5).
//
// Adapted from the reference: of its functions, only get, the read at a
// pointer, is ported here, the one SHAPE's first three checks use. The
// positions of P's block nodes, commitment entries, entropy objects and
// genesis objects join with the checks that read them.

package verifier

// valueAt returns the value at pointer p in value, and nil where value
// holds none: past a member value lacks, an index past an array's end, or
// a segment that names a member of an array or an element of an object.
func valueAt(value any, p Pointer) any {
	for _, s := range p {
		switch v := value.(type) {
		case map[string]any:
			if s.isIndex {
				return nil
			}
			value = v[s.name]
		case []any:
			if !s.isIndex || s.index >= len(v) {
				return nil
			}
			value = v[s.index]
		default:
			return nil
		}
	}
	return value
}
