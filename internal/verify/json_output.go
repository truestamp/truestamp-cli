// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/version"
)

// JSONReport is the machine-readable form `inspect --json` writes for a
// rejection of the earlier bundle parser: the field names of that
// parser's report, plus `verifier` naming this implementation and
// `rejection` for the rejection.
type JSONReport struct {
	Verifier Verifier `json:"verifier"`

	// Passed is false: BuildJSONRejection, which writes every JSONReport,
	// writes a rejection.
	Passed bool `json:"passed"`

	// Rejection is set, and every other result field is zero, when the
	// bundle was refused before any step ran.
	Rejection *Rejection `json:"rejection,omitempty"`

	ID     string `json:"id,omitempty"`
	Source string `json:"source,omitempty"`

	// Steps is the step record, empty for a rejection.
	Steps []Step `json:"steps"`

	Temporal *Temporal `json:"temporal,omitempty"`

	PassCount   int `json:"pass_count"`
	FailedCount int `json:"failed_count"`
	WarnCount   int `json:"warn_count"`
	SkipCount   int `json:"skip_count"`
	InfoCount   int `json:"info_count"`

	// HashProvided is the normalized expected hash when one reached a
	// comparison, else null. ExpectedHashProvided and HashMatched are
	// reported separately.
	HashProvided         *string `json:"hash_provided"`
	ExpectedHashProvided bool    `json:"expected_hash_provided"`
	HashMatched          bool    `json:"hash_matched"`

	ProofVersion    int    `json:"proof_version"`
	SkippedExternal bool   `json:"skipped_external"`
	GeneratedAt     string `json:"generated_at,omitempty"`
}

// Verifier names the implementation that produced a report.
type Verifier struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Rejection carries the earlier bundle parser's rejection identifier.
type Rejection struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Advice string `json:"advice"`
}

// ThisVerifier identifies this CLI build.
func ThisVerifier() Verifier {
	return Verifier{Name: "truestamp-cli", Version: version.Version}
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
