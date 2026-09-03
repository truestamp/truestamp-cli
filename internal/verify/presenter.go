// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"fmt"
	"io"
	"os"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

const rule = "=============================================================================="
const thinRule = "------------------------------------------------------------------------------"

// Present renders a Report to stdout in the reference verifier's shape: a
// short bundle header, then the five categories in Appendix E.22's display
// order, each row as a badge, its group and its message, then the counts,
// whether a file hash was provided, and the verdict.
func Present(r *Report) {
	fmt.Print(Render(r, true))
}

// Render returns the report text. Badges are colored when color is true.
func Render(r *Report, color bool) string {
	var b strings.Builder
	writeHeader(&b, r)
	b.WriteString("\n" + rule + "\nVERIFICATION REPORT\n" + rule + "\n")

	for i, steps := range r.StepsByCategory() {
		if len(steps) == 0 {
			continue
		}
		title := "Other"
		if i < len(CategoryOrder) {
			title = CategoryLabel(CategoryOrder[i])
		}
		fmt.Fprintf(&b, "\n%s\n", title)
		for _, s := range steps {
			fmt.Fprintf(&b, "  %s  %-20s %s\n", badge(s.Status, color), s.Group, s.Message)
		}
	}

	c := r.Counts()
	fmt.Fprintf(&b, "\n%s\n", thinRule)
	fmt.Fprintf(&b, "  %d passed   %d failed   %d warned   %d skipped   %d info\n",
		c.Passed, c.Failed, c.Warned, c.Skipped, c.Info)
	fmt.Fprintf(&b, "  file hash provided: %s\n", yesNo(r.HashProvided()))
	if r.HashProvided() {
		fmt.Fprintf(&b, "  file hash matched: %s\n", yesNo(r.HashMatched()))
	}

	if r.Passed() {
		fmt.Fprintf(&b, "\n  VERDICT: %s\n", verdictStyle(true, color).Render("PASSED"))
		b.WriteString("\n  The proof is internally sound: every value in it reproduces, and the\n" +
			"  signature over those values is valid. Any `skip` above is a check this\n" +
			"  run did not perform, not a check that failed.\n")
		if r.SignaturesSkipped() {
			b.WriteString("\n  The Ed25519 signature was NOT checked (--skip-signatures); nothing here\n" +
				"  establishes who signed this proof.\n")
		}
	} else {
		fmt.Fprintf(&b, "\n  VERDICT: %s\n", verdictStyle(false, color).Render("FAILED"))
		b.WriteString("\n  At least one step failed. This proof should not be relied on.\n")
	}
	b.WriteString("\n" + rule + "\n")
	return b.String()
}

func writeHeader(b *strings.Builder, r *Report) {
	fmt.Fprintf(b, "\nBundle: %s (%s, %d bytes)\n", r.Filename, r.Format, r.FileSize)
	fmt.Fprintf(b, "  version=%s  type=%s\n", r.VersionLiteral, r.SubjectType)
	fmt.Fprintf(b, "  generated_at: %s\n", orAbsent(r.GeneratedAt))
	if r.SubjectType == proof.WitnessBlock || r.SubjectType == "beacon" {
		fmt.Fprintf(b, "  subject id:   (block-like: none)\n")
	} else {
		fmt.Fprintf(b, "  subject id:   %s\n", orAbsent(r.SubjectID))
	}
	fmt.Fprintf(b, "  block id:     %s\n", orAbsent(r.BlockID))
	witnesses := "(none carried)"
	if len(r.WitnessesCarried) > 0 {
		witnesses = strings.Join(r.WitnessesCarried, ", ")
	}
	fmt.Fprintf(b, "  witnesses:    %s\n", witnesses)
	var commits []string
	for _, c := range r.Commitments {
		if c.Network == "" {
			commits = append(commits, c.Chain)
		} else {
			commits = append(commits, c.Chain+" "+c.Network)
		}
	}
	fmt.Fprintf(b, "  commitments:  %s\n", strings.Join(commits, ", "))
	fmt.Fprintf(b, "  signing key event: %s\n", carriedOrAbsent(r.KeyEventCarried))
	keyring := "none in hand"
	if r.KeyringSource != "" {
		keyring = r.KeyringSource
	}
	fmt.Fprintf(b, "  keyring:      %s\n", keyring)
	if r.Remote {
		b.WriteString("  mode:         remote (steps reported by the Truestamp server)\n")
	} else if r.SkippedExternal {
		b.WriteString("  mode:         offline (no network lookups)\n")
	} else {
		b.WriteString("  mode:         online\n")
	}
}

func orAbsent(s string) string {
	if s == "" {
		return "(absent)"
	}
	return s
}

func carriedOrAbsent(ok bool) string {
	if ok {
		return "carried"
	}
	return "absent"
}

func yesNo(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}

func badge(s Status, color bool) string {
	text := s.Badge()
	if !color {
		return text
	}
	var style lipgloss.Style
	switch s {
	case StatusPass:
		style = lipgloss.NewStyle().Foreground(ui.Green)
	case StatusFail:
		style = lipgloss.NewStyle().Foreground(ui.Red).Bold(true)
	case StatusWarn:
		style = lipgloss.NewStyle().Foreground(ui.Yellow)
	default:
		style = lipgloss.NewStyle().Foreground(ui.Dim)
	}
	return style.Render(text)
}

func verdictStyle(passed, color bool) lipgloss.Style {
	if !color {
		return lipgloss.NewStyle()
	}
	if passed {
		return lipgloss.NewStyle().Foreground(ui.Green).Bold(true)
	}
	return lipgloss.NewStyle().Foreground(ui.Red).Bold(true)
}

// PresentRejection renders an Appendix E.6 hard rejection: no report
// exists, only the E.23 identifier and one line of advice.
func PresentRejection(w io.Writer, err error) {
	code := proof.RejectionCode(err)
	detail := err.Error()
	if re, ok := err.(*proof.RejectionError); ok {
		detail = re.Detail
	}
	fmt.Fprintf(w, "\n  REJECTED: %s\n  %s\n%s\n", code, detail, indent(proof.RejectionAdvice(code)))
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}

// PresentTo is Present writing to w without color.
func PresentTo(w io.Writer, r *Report) {
	fmt.Fprint(w, Render(r, false))
}

var _ = os.Stdout
