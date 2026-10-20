// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the report of SPEC.md section 10.5, which every run that
// passes steps 0 to 3 gives, whatever its label, and which a bare
// rejection never gives (section 10.2, the gate; rejection.go). A report
// has exactly these members, and JCS writes them in this order (section
// 2.5):
//
//	| Member                              | Value                                                                       |
//	|-------------------------------------|-----------------------------------------------------------------------------|
//	| attribution                         | truestamp, keyring:<d>, unattributed or none                                |
//	| code                                | the run's code, present exactly when label is failed                        |
//	| environment                         | E_v, the verifier's environment (section 6.4); always present               |
//	| label                               | failed, unattributed, supplied-keyring, internally-consistent or verified   |
//	| rows                                | every row, {code, message, part, status, step, values}, in step order       |
//	| subject_id                          | the subject's id; null until SHAPE passes                                   |
//	| submitted_after, submitted_before   | the two edge objects of section 11.5, both null when label is failed        |
//	| trust_root                          | {digest, kind, location}, digest and location null where there is none     |
//	| type                                | the proof's type; null until SHAPE passes                                   |
//	| verified                            | true exactly when label is verified                                         |
//	| window                              | established when submitted_before is not null, and otherwise not_established |
//
// "Passed", meaning that no row is fail, is internal and never output
// (section 10.5). When the label is failed, the report's edge objects are
// both null, so the window is not established (section 10.2, step 14), even
// when WINDOW drew both and the run failed after it, at EXPECTED_HASH:
// [NewReport] writes them null, while the run keeps the edges WINDOW drew,
// so a caller reads the edges only from the report. A
// report holds no item code, which is never a member or row of a report
// (section 11.5), and no trace (section 10.6), which a run holds apart.

package verifier

import (
	"slices"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// The values of a report's window (SPEC.md section 10.5).
const (
	WindowEstablished    = "established"
	WindowNotEstablished = "not_established"
)

// Report is the report of a run (SPEC.md section 10.5; see the file's
// doc). A SubjectID or Type of "" is null, and so is a nil edge; Code is
// "" exactly when Label is not failed.
type Report struct {
	Attribution     string
	Code            specdata.Reason
	Environment     string
	Label           string
	Rows            []Row
	SubjectID       string
	SubmittedAfter  *Edge
	SubmittedBefore *Edge
	TrustRoot       TrustRoot
	Type            string
	Verified        bool
	Window          string
}

// NewReport returns the report of run, after VERDICT (step 14): its label,
// code and attribution, E_v, its rows, its type and subject id, its trust
// root, and the edges WINDOW drew, or none when the label is failed. It
// panics for a run VERDICT has not labelled, which has no report yet: a
// defect in the caller.
func NewReport(run *Run) *Report {
	if run.Label == "" {
		panic("verifier: a report of a run that VERDICT has not labelled")
	}
	report := &Report{
		Attribution: run.Attribution,
		Code:        run.Code,
		Environment: run.Config.Environment,
		Label:       run.Label,
		Rows:        slices.Clone(run.Rows),
		SubjectID:   run.SubjectID,
		TrustRoot:   run.TrustRoot,
		Type:        run.Type,
		Verified:    run.Label == LabelVerified,
		Window:      WindowNotEstablished,
	}
	if run.Label != LabelFailed {
		report.SubmittedAfter, report.SubmittedBefore = run.SubmittedAfter, run.SubmittedBefore
	}
	if report.SubmittedBefore != nil {
		report.Window = WindowEstablished
	}
	return report
}

// Value is the report as SPEC.md section 10.5 writes it, with exactly its
// members (see the file's doc).
func (r *Report) Value() map[string]any {
	rows := make([]any, 0, len(r.Rows))
	for _, row := range r.Rows {
		rows = append(rows, row.Value())
	}
	out := map[string]any{
		"attribution":      r.Attribution,
		"environment":      r.Environment,
		"label":            r.Label,
		"rows":             rows,
		"subject_id":       nullable(r.SubjectID),
		"submitted_after":  edgeValue(r.SubmittedAfter),
		"submitted_before": edgeValue(r.SubmittedBefore),
		"trust_root":       r.TrustRoot.Value(),
		"type":             nullable(r.Type),
		"verified":         r.Verified,
		"window":           r.Window,
	}
	if r.Label == LabelFailed {
		out["code"] = string(r.Code)
	}
	return out
}

// Encode returns the report written with JCS (SPEC.md section 2.5), its
// members in the order JCS gives them.
func (r *Report) Encode() ([]byte, error) { return jcs.Encode(r.Value()) }

// Value is trust_root as a report writes it, {digest, kind, location}
// (SPEC.md section 10.5), digest and location null where they are "".
func (t TrustRoot) Value() map[string]any {
	return map[string]any{"digest": nullable(t.Digest), "kind": string(t.Kind), "location": nullable(t.Location)}
}

// edgeValue is an edge object as a report writes it, or nil for none.
func edgeValue(edge *Edge) any {
	if edge == nil {
		return nil
	}
	return edge.Value()
}

// nullable is s, or nil for "".
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
