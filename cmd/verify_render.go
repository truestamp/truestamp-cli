// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// This file is the human output of `truestamp verify`: a report (SPEC.md
// section 10.5) or a bare rejection (section 10.2, the gate), written as
// text. It follows the display rules of SPEC.md:
//
//   - a non-production E_v is shown beside the verdict (section 6.4), and
//     keyless mode, the one run whose trust_root kind is none, is said in
//     words (section 8.3);
//   - the edges are drawn only from the report's two edge objects,
//     captioned "Submitted after" and "Submitted before", and no window is
//     shown for a run whose label is failed (section 11.5); a report's edge
//     objects are [verifier.NewReport]'s, never the run's, so a run that
//     failed after WINDOW shows no edge;
//   - each source an edge rests on is named "read from <source> in this
//     run", or, for an answer from a cache, "read from <source> at
//     <fetched_at>" (section 11.5);
//   - no time the proof carries is shown, and no marker the proof carries
//     is quoted (section 6.4): only the rows' messages, which the verifier
//     writes, say what a check found;
//   - every row that does not pass is listed, after a count of each
//     status; a WINDOW row without its message where the message names
//     an instant: a failed run's, which names the edges WINDOW drew, and
//     window_not_established and recorded_time_outside_window in any run,
//     which name the submitted-after edge and the time inside the
//     subject's id.
//
// --json writes the report itself instead, exactly as SPEC.md section 10.5
// gives it, and nothing here.

// labelSays is what each label says (SPEC.md section 10.5, "Labels").
var labelSays = map[string]string{
	verifier.LabelFailed:               "A check failed, and its row says which.",
	verifier.LabelUnattributed:         "The proof is intact under the key it carries, which was checked against no keyring, so anyone could have made it.",
	verifier.LabelSuppliedKeyring:      "The proof is intact and signed by a key in the keyring the operator supplied; this says nothing about Truestamp.",
	verifier.LabelInternallyConsistent: "The proof is intact and signed by Truestamp, but no read in this run established its submitted-before edge.",
	verifier.LabelVerified:             "The proof is intact, signed by Truestamp, and places its subject's submission before an instant read from a public chain in this run.",
}

// renderReport writes report as text (see the file's doc) in one write,
// and gives that write's error.
func renderReport(w io.Writer, report *verifier.Report) error {
	var b strings.Builder
	beside := ""
	if report.Environment != "prod" {
		beside = fmt.Sprintf(" (environment %s, not production)", report.Environment)
	}
	fmt.Fprintf(&b, "Verdict:      %s%s\n", report.Label, beside)
	fmt.Fprintf(&b, "              %s\n", labelSays[report.Label])
	if report.Label == verifier.LabelFailed {
		fmt.Fprintf(&b, "Code:         %s\n", report.Code)
	}
	fmt.Fprintf(&b, "Attribution:  %s\n", report.Attribution)
	fmt.Fprintf(&b, "Trust root:   %s\n", trustRootText(report.TrustRoot))
	if report.Environment == "prod" {
		b.WriteString("Environment:  prod\n")
	} else {
		fmt.Fprintf(&b, "Environment:  %s, not production: this run checks proofs of Truestamp's %s environment\n", report.Environment, report.Environment)
	}
	if report.TrustRoot.Kind == verifier.KindNone {
		b.WriteString("Keyless:      this run checked the signature only against the key the proof carries, against no keyring\n")
	}
	if report.Type != "" {
		fmt.Fprintf(&b, "Type:         %s\n", report.Type)
		fmt.Fprintf(&b, "Subject id:   %s\n", report.SubjectID)
	}
	b.WriteString(windowText(report))
	b.WriteString(rowsText(report.Rows, report.Label == verifier.LabelFailed))
	_, err := io.WriteString(w, b.String())
	return err
}

// trustRootText is trust_root in words: its kind, where its keys came
// from, and the SHA-256 of the keyring's bytes, each where there is one.
func trustRootText(root verifier.TrustRoot) string {
	text := string(root.Kind)
	if root.Location != "" {
		text += ", from " + root.Location
	}
	if root.Digest != "" {
		text += ", SHA-256 " + root.Digest
	}
	return text
}

// windowText is the submission window: both edges, each from its edge
// object or absent with its reasons, and none for a failed run.
func windowText(report *verifier.Report) string {
	if report.Label == verifier.LabelFailed {
		return "Window:       none is shown, since a check failed\n"
	}
	reasons := absentReasons(report.Rows)
	var b strings.Builder
	b.WriteString(edgeText("Submitted after: ", report.SubmittedAfter, reasons["submitted_after"]))
	b.WriteString(edgeText("Submitted before:", report.SubmittedBefore, reasons["submitted_before"]))
	if report.Type == "beacon" && report.SubmittedAfter != nil {
		instant, _ := report.SubmittedAfter.Value()["instant"].(string)
		fmt.Fprintf(&b, "              No one, Truestamp included, could compute this value before %s. Truestamp chooses the block contents and could bias it.\n", instant)
	}
	return b.String()
}

// edgeText is one edge: its instant and basis, and each source it rests
// on (SPEC.md section 11.5), all read from its edge object; or, for an
// edge that does not exist, the codes of the rows that could have given
// it, or that the proof carries nothing that could.
func edgeText(caption string, edge *verifier.Edge, reasons []string) string {
	if edge == nil {
		switch {
		case reasons == nil:
			return caption + " not established in this run\n"
		case len(reasons) == 0:
			return caption + " none: this proof carries nothing that could give this edge\n"
		}
		return caption + " not established in this run (" + strings.Join(reasons, ", ") + ")\n"
	}
	value := edge.Value()
	instant, _ := value["instant"].(string)
	basis, _ := value["basis"].(string)
	var reads []string
	sources, _ := value["sources"].([]any)
	for _, s := range sources {
		read, _ := s.(map[string]any)
		name, _ := read["source"].(string)
		if cached, _ := read["cached"].(bool); cached {
			at, _ := read["fetched_at"].(string)
			reads = append(reads, "read from "+name+" at "+at)
		} else {
			reads = append(reads, "read from "+name+" in this run")
		}
	}
	return fmt.Sprintf("%s %s (%s, %s)\n", caption, instant, basis, strings.Join(reads, "; "))
}

// absentReasons are, for each edge the WINDOW row says is absent, the code
// and part of each row that could have given it, as "<code> at <part>"
// ("pass at <part>" for a row that passed with no candidate); an edge the
// row names no reasons for is missing from the map.
func absentReasons(rows []verifier.Row) map[string][]string {
	out := map[string][]string{}
	for _, row := range rows {
		if row.Step != "WINDOW" || row.Part != nil {
			continue
		}
		absent, _ := row.Values["absent"].(map[string]any)
		for edge, listed := range absent {
			items, _ := listed.([]any)
			reasons := []string{}
			for _, item := range items {
				reason, _ := item.(map[string]any)
				code, _ := reason["code"].(string)
				if code == "" {
					code = "pass"
				}
				part, _ := reason["part"].(string)
				reasons = append(reasons, code+" at "+part)
			}
			out[edge] = reasons
		}
	}
	return out
}

// rowsText counts the rows by status and lists every row that does not
// pass, with its step, code, part and message; a WINDOW row whose message
// names an instant is listed without it, since the edges are drawn only
// from the report's edge objects (SPEC.md section 11.5): when failed (the
// label is failed, and no window is shown) every WINDOW row, and in any
// run the rows of [windowRowsWithoutMessage].
func rowsText(rows []verifier.Row, failed bool) string {
	counts := map[verifier.Status]int{}
	for _, row := range rows {
		counts[row.Status]++
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Rows:         %d (%d pass, %d fail, %d skip, %d warn, %d info)\n", len(rows),
		counts[verifier.StatusPass], counts[verifier.StatusFail], counts[verifier.StatusSkip],
		counts[verifier.StatusWarn], counts[verifier.StatusInfo])
	for _, row := range rows {
		if row.Status == verifier.StatusPass {
			continue
		}
		at := ""
		if row.Part != nil {
			at = " at " + *row.Part
		}
		if row.Step == "WINDOW" && (failed || windowRowsWithoutMessage[row.Code]) {
			fmt.Fprintf(&b, "  %s %s %s%s\n", row.Status, row.Step, row.Code, at)
			continue
		}
		fmt.Fprintf(&b, "  %s %s %s%s: %s\n", row.Status, row.Step, row.Code, at, row.Message)
	}
	return b.String()
}

// windowRowsWithoutMessage are the WINDOW codes whose message names an
// instant: window_not_established the submitted-after edge, which its
// edge line already shows from the edge object, and
// recorded_time_outside_window the time inside the subject's id, a time
// the proof carries, which the text never shows.
var windowRowsWithoutMessage = map[specdata.Reason]bool{
	specdata.ReasonWindowNotEstablished:      true,
	specdata.ReasonRecordedTimeOutsideWindow: true,
}

// renderRejection writes a bare rejection as text: its code, its
// sub_reason where it has one, and its message, and that there is no
// report and no verdict (SPEC.md section 10.2, the gate); it gives the
// write's error.
func renderRejection(w io.Writer, rejection *verifier.Rejection) error {
	reason := ""
	if rejection.SubReason != "" {
		reason = " (" + string(rejection.SubReason) + ")"
	}
	_, err := fmt.Fprintf(w, "Rejected:     %s%s\n              %s\n              No report and no verdict.\n", rejection.Code, reason, rejection.Message)
	return err
}
