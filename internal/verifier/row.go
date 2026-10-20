// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds a report row, {code, message, part, status, step,
// values} (SPEC.md section 10.5 (Verdict and report)). The codes are
// internal/specdata's transcription of spec/reason-codes.json (section
// 12.1), a null part is a nil *string, and a number among the values is a
// strictjson.Number, so internal/jcs writes them.

package verifier

import (
	"fmt"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// Status is a row's status (SPEC.md section 10.5).
type Status string

// The row statuses.
const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
	StatusWarn Status = "warn"
	StatusInfo Status = "info"
)

// Row is a report row:
//
//   - Code: a reason code of SPEC.md section 12.1, "" exactly when the
//     status is pass;
//   - Message: for display only, and never compared;
//   - Part: an RFC 6901 pointer into the proof, or nil for a row about no
//     single member;
//   - Status and Step: the row's status and the name of its step (SPEC.md
//     section 10.2), such as "ENVELOPE";
//   - Values: what this run computed or read, and, for a failed comparison,
//     the declared value it contradicts, under "declared". Each is a value
//     internal/jcs can write: a string, a bool, a strictjson.Number, nil,
//     or a []any or map[string]any of them.
type Row struct {
	Code    specdata.Reason
	Message string
	Part    *string
	Status  Status
	Step    string
	Values  map[string]any
}

// Part returns pointer as a row's part: a JSON Pointer that is not null.
func Part(pointer string) *string { return &pointer }

// PassRow returns a pass row of step at part.
func PassRow(step string, part *string, message string, values map[string]any) Row {
	return NewRow(step, "", part, message, values)
}

// NewRow returns a row of step with code, at part: its status is the one
// SPEC.md section 12.1 gives the code, and pass for no code. It panics for
// a step SPEC.md section 10.2 does not name, a code the table lacks, or a
// code that ends a run in a bare rejection, so no row can carry a code that
// does not mean what its status says.
func NewRow(step string, code specdata.Reason, part *string, message string, values map[string]any) Row {
	if !slices.Contains(rowSteps, step) {
		panic(fmt.Sprintf("verifier: %q is not a step that reports rows", step))
	}
	if values == nil {
		values = map[string]any{}
	}
	row := Row{Code: code, Message: message, Part: part, Status: StatusPass, Step: step, Values: values}
	if code == "" {
		return row
	}
	entry, ok := specdata.LookupReason(code)
	switch {
	case !ok:
		panic(fmt.Sprintf("verifier: %q is not a reason code of SPEC.md section 12.1", code))
	case entry.Status == specdata.StatusBare:
		panic(fmt.Sprintf("verifier: %s ends a run in a bare rejection, not a row", code))
	}
	row.Status = Status(entry.Status)
	return row
}

// Reduce returns what conformance compares of a row (SPEC.md section
// 13.1): its code, part, status and step, with a nil code for a pass.
func (r Row) Reduce() (code *string, part *string, status, step string) {
	if r.Code != "" {
		text := string(r.Code)
		code = &text
	}
	return code, r.Part, string(r.Status), r.Step
}

// rowSteps are the steps that report rows: every step of SPEC.md section
// 10.2 but VERDICT (step 14), which adds none.
var rowSteps = steps[:len(steps)-1]
