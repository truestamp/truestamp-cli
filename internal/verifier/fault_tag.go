// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build truestamp_fault

// This file is built only with the build tag truestamp_fault, which no
// shipped build sets: the cmd package's tests build a binary with it to
// see how `truestamp verify` ends on a fault (SPEC.md sections 10.5 and
// 12.2). In that binary, [Verify] raises a fault before the step of
// SPEC.md section 10.2 that faultStep names, which the test sets at link
// time (-ldflags -X).

package verifier

// faultStep names the step a fault is raised before; "" raises none.
var faultStep string

func init() {
	faultAt = func(step string) {
		if faultStep != "" && step == faultStep {
			panic("a fault raised at " + step + " by the build's faultStep")
		}
	}
}
