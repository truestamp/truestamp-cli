// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.CanonicalForm
// (elixir/lib/truestamp/verifier/canonical_form.ex in
// truestamp-verification): the canonical form of a data file of SPEC.md
// section 9, the form of every file under spec/ but the constructions texts,
// of cases.json and of the trace files under vectors/positive/ (section
// 13.2), in which a run's trace is written (section 10.6).
//
// Adapted from the reference: members are ordered by the same UTF-16 sort
// as [Encode], and every name and scalar is written by Encode, so the form
// refuses exactly what Encode refuses.

package jcs

import "strings"

// CanonicalForm returns v in the canonical form of SPEC.md section 9, with
// its final newline: members in JCS order, two-space indentation and LF line
// ends. A member is its name, a colon, one space and its value; an object or
// array that is not empty puts each member or element on a line of its own,
// one level deeper and followed by a comma unless it is the last, and closes
// on a line of its own at the indentation of the line that opens it, while
// an empty one is {} or []; and every name and scalar is written as [Encode]
// writes it.
//
// v is a value as [Encode] takes it, and CanonicalForm refuses what Encode
// refuses, with [ErrOutsideProfile].
func CanonicalForm(v any) ([]byte, error) {
	b, err := appendForm(nil, v, 0)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func appendForm(b []byte, v any, level int) ([]byte, error) {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 0 {
			return append(b, "{}"...), nil
		}
		names, err := sortedNames(t)
		if err != nil {
			return nil, err
		}
		b = append(b, "{\n"...)
		for i, name := range names {
			if i > 0 {
				b = append(b, ",\n"...)
			}
			b = appendIndent(b, level+1)
			if b, err = appendString(b, name); err != nil {
				return nil, err
			}
			b = append(b, ": "...)
			if b, err = appendForm(b, t[name], level+1); err != nil {
				return nil, err
			}
		}
		b = append(b, '\n')
		b = appendIndent(b, level)
		return append(b, '}'), nil
	case []any:
		if len(t) == 0 {
			return append(b, "[]"...), nil
		}
		b = append(b, "[\n"...)
		for i, element := range t {
			if i > 0 {
				b = append(b, ",\n"...)
			}
			b = appendIndent(b, level+1)
			var err error
			if b, err = appendForm(b, element, level+1); err != nil {
				return nil, err
			}
		}
		b = append(b, '\n')
		b = appendIndent(b, level)
		return append(b, ']'), nil
	default:
		return appendValue(b, v)
	}
}

func appendIndent(b []byte, level int) []byte {
	return append(b, strings.Repeat("  ", level)...)
}
