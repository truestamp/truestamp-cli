// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.Pointer
// (elixir/lib/truestamp/verifier/pointer.ex in truestamp-verification),
// checked against the Truestamp application's port,
// Truestamp.Proof.Verifier.Pointer (lib/truestamp/proof/verifier/pointer.ex
// in truestamp-v2): JSON Pointers (RFC 6901) into P, the parsed value of
// proof, and the document order that orders a step's rows (SPEC.md section
// 10.2).
//
// A pointer is held as its segments, each a member's name or an array
// element's index, so Pointer{Name("block"), Name("record"),
// Name("merkle_root")} is /block/record/merkle_root. [Pointer.Encode]
// writes a row's part from it.
//
// Document order is the order in which a depth-first walk of P visits
// values, each value before its contents, an object's members in JCS order
// and an array's elements by index. Of two pointers, the one that is a
// prefix of the other comes first, and otherwise the first segment in
// which they differ decides: two names by JCS order, the order of their
// UTF-16 code units (RFC 8785 section 3.2.3), and two indices by value. A
// pointer can name a member that P lacks, and it takes the place that
// member would take. Adapted from the reference: a segment is a struct,
// since Go has no union of a string and an integer.

package verifier

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Segment is one segment of a pointer: a member's name or an element's
// index.
type Segment struct {
	name    string
	index   int
	isIndex bool
}

// Name returns the segment of the member named name.
func Name(name string) Segment { return Segment{name: name} }

// Index returns the segment of the array element at index, which must not
// be negative.
func Index(index int) Segment {
	if index < 0 {
		panic(fmt.Sprintf("verifier: a pointer's index %d is negative", index))
	}
	return Segment{index: index, isIndex: true}
}

// Pointer is a JSON Pointer as its segments, outermost first.
type Pointer []Segment

// Encode returns the RFC 6901 string of p: each segment after a /, with ~
// written ~0 and / written ~1 in a name.
func (p Pointer) Encode() string {
	var b strings.Builder
	for _, s := range p {
		b.WriteByte('/')
		if s.isIndex {
			b.WriteString(strconv.Itoa(s.index))
			continue
		}
		b.WriteString(strings.ReplaceAll(strings.ReplaceAll(s.name, "~", "~0"), "/", "~1"))
	}
	return b.String()
}

// Compare gives the document order of a and b: -1 when a comes first, 0
// when they are one pointer, and +1 when b comes first.
func Compare(a, b Pointer) int {
	for i := range min(len(a), len(b)) {
		if c := compareSegments(a[i], b[i]); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(a), len(b))
}

// compareSegments orders two segments of one position. They name members
// of one object, or elements of one array; a name and an index never meet
// there, and are ordered index first only so the order is total.
func compareSegments(x, y Segment) int {
	switch {
	case x.isIndex && y.isIndex:
		return cmp.Compare(x.index, y.index)
	case x.isIndex:
		return -1
	case y.isIndex:
		return 1
	}
	return slices.Compare(utf16.Encode([]rune(x.name)), utf16.Encode([]rune(y.name)))
}

// SortBy sorts list into the document order of the pointer part gives for
// each element. The sort is stable, so elements at one pointer keep their
// order.
func SortBy[T any](list []T, part func(T) Pointer) {
	slices.SortStableFunc(list, func(x, y T) int { return Compare(part(x), part(y)) })
}

// FirstBy returns the element of list whose pointer, as part gives it,
// comes first in document order, the earliest in list among equals, and
// false when list is empty.
func FirstBy[T any](list []T, part func(T) Pointer) (T, bool) {
	var first T
	if len(list) == 0 {
		return first, false
	}
	first = list[0]
	for _, element := range list[1:] {
		if Compare(part(element), part(first)) < 0 {
			first = element
		}
	}
	return first, true
}
