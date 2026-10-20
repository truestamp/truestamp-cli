// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the label and the run's code of step 14 of SPEC.md
// section 10.2, VERDICT, as section 10.5 gives them. VERDICT reports no
// row.
//
// The label is the first of these that applies (SPEC.md section 10.5,
// "Labels"):
//
//	| Label                 | When                                                                     |
//	|-----------------------|--------------------------------------------------------------------------|
//	| failed                | any row is fail                                                          |
//	| unattributed          | the attribution is unattributed                                          |
//	| supplied-keyring      | the attribution is keyring:<d>                                           |
//	| internally-consistent | the attribution is truestamp, and the submitted-before edge is not established |
//	| verified              | the attribution is truestamp, and the submitted-before edge is established |
//
// A run that is not failed has a WINDOW row that passes exactly when the
// submitted-before edge is established (SPEC.md section 11.3: fail
// window_inverted needs a failure, and info window_not_established says
// the edge does not exist), so that row is what VERDICT reads; a run with
// no WINDOW row has drawn no edge. The run's code, present exactly when
// the label is failed, is the code of its first failing step's first
// failing row, which, since rows are in step order, is its first failing
// row (section 10.2). A run that failed at the gate has the attribution
// none, and is failed by its gate row.

package verifier

import (
	"strings"
)

// The labels of SPEC.md section 10.5.
const (
	LabelFailed               = "failed"
	LabelUnattributed         = "unattributed"
	LabelSuppliedKeyring      = "supplied-keyring"
	LabelInternallyConsistent = "internally-consistent"
	LabelVerified             = "verified"
)

// Verdict is step 14, VERDICT: the run's label and, when it is failed, its
// code (see the file's doc).
func Verdict(run *Run) {
	run.Label, run.Code = LabelFailed, ""
	for _, row := range run.Rows {
		if row.Status == StatusFail {
			run.Code = row.Code
			return
		}
	}
	switch {
	case run.Attribution == AttributionUnattributed:
		run.Label = LabelUnattributed
	case strings.HasPrefix(run.Attribution, "keyring:"):
		run.Label = LabelSuppliedKeyring
	case submittedBeforeEstablished(run):
		run.Label = LabelVerified
	default:
		run.Label = LabelInternallyConsistent
	}
}

// submittedBeforeEstablished reports whether WINDOW drew a submitted-before
// edge in a run that did not fail: its WINDOW row passed.
func submittedBeforeEstablished(run *Run) bool {
	for _, row := range run.Rows {
		if row.Step == "WINDOW" && row.Part == nil && row.Status == StatusPass {
			return true
		}
	}
	return false
}
