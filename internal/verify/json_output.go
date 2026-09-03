// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"sort"

	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/version"
)

// JSONReport is the machine-readable form of a report. Its field names are
// the server's (/proof/verify), so a CLI report and an API report are
// directly comparable, plus `verifier` naming this implementation and
// `rejection` for an Appendix E.6 hard rejection.
type JSONReport struct {
	Verifier Verifier `json:"verifier"`

	// Passed is the verdict: no step failed. False for a rejection.
	Passed bool `json:"passed"`

	// Rejection is set, and every other result field is zero, when the
	// bundle was refused before any step ran (E.6).
	Rejection *Rejection `json:"rejection,omitempty"`

	ID     string `json:"id,omitempty"`
	Source string `json:"source,omitempty"`

	// Steps is the complete step record in Appendix E.22's category order,
	// each category's rows in emission order.
	Steps []Step `json:"steps"`

	Temporal *Temporal `json:"temporal,omitempty"`

	PassCount   int `json:"pass_count"`
	FailedCount int `json:"failed_count"`
	WarnCount   int `json:"warn_count"`
	SkipCount   int `json:"skip_count"`
	InfoCount   int `json:"info_count"`

	// HashProvided is the normalized expected hash when one reached a
	// comparison, else null. ExpectedHashProvided and HashMatched are the
	// two facts Appendix E.7 requires to be reportable separately.
	HashProvided         *string `json:"hash_provided"`
	ExpectedHashProvided bool    `json:"expected_hash_provided"`
	HashMatched          bool    `json:"hash_matched"`

	ProofVersion    int    `json:"proof_version"`
	SkippedExternal bool   `json:"skipped_external"`
	GeneratedAt     string `json:"generated_at,omitempty"`

	// SignaturesChecked is false when --skip-signatures left E.16
	// unperformed; `passed` alone cannot say so.
	SignaturesChecked bool `json:"signatures_checked"`

	// Remote is true when the steps were reported by the Truestamp server.
	Remote bool `json:"remote,omitempty"`
}

// Verifier names the implementation that produced a report.
type Verifier struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Rejection carries an Appendix E.23 rejection identifier.
type Rejection struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Advice string `json:"advice"`
}

// ThisVerifier identifies this CLI build.
func ThisVerifier() Verifier {
	return Verifier{Name: "truestamp-cli", Version: version.Version}
}

// BuildJSONReport creates the machine-readable form of a report.
func BuildJSONReport(r *Report) *JSONReport {
	c := r.Counts()
	out := &JSONReport{
		Verifier:             ThisVerifier(),
		Passed:               r.Passed(),
		ID:                   r.SubjectID,
		Source:               r.SubjectType,
		Steps:                orderedSteps(r),
		PassCount:            c.Passed,
		FailedCount:          c.Failed,
		WarnCount:            c.Warned,
		SkipCount:            c.Skipped,
		InfoCount:            c.Info,
		ExpectedHashProvided: r.ExpectedHash != "",
		HashMatched:          r.HashMatched(),
		ProofVersion:         r.ProofVersion,
		SkippedExternal:      r.SkippedExternal,
		GeneratedAt:          r.GeneratedAt,
		SignaturesChecked:    !r.SignaturesSkipped(),
		Remote:               r.Remote,
	}
	if r.HashProvided() {
		h := r.ExpectedHash
		out.HashProvided = &h
	}
	if !r.Temporal.IsZero() {
		t := r.Temporal
		out.Temporal = &t
	}
	return out
}

// BuildJSONRejection creates the machine-readable form of a hard rejection.
func BuildJSONRejection(err error) *JSONReport {
	code := proof.RejectionCode(err)
	detail := err.Error()
	if re, ok := err.(*proof.RejectionError); ok {
		detail = re.Detail
	}
	return &JSONReport{
		Verifier:  ThisVerifier(),
		Passed:    false,
		Rejection: &Rejection{Code: code, Detail: detail, Advice: proof.RejectionAdvice(code)},
		Steps:     []Step{},
	}
}

// orderedSteps returns every step in Appendix E.22's category order. The
// sort is stable and keyed on the category alone, so emit order within a
// category survives.
func orderedSteps(r *Report) []Step {
	steps := make([]Step, 0, len(r.Steps))
	steps = append(steps, r.Steps...)
	sort.SliceStable(steps, func(i, j int) bool {
		return categoryRank(steps[i].Category) < categoryRank(steps[j].Category)
	})
	return steps
}
