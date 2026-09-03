// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"errors"
	"fmt"
)

// Hard-rejection identifiers from the whitepaper's error taxonomy (Appendix
// E.23). A hard rejection aborts before any report exists (E.6), and
// independent verifiers are required to agree on this vocabulary so their
// outcomes are comparable, which is only possible if the identifier travels
// with the error rather than living in prose.
const (
	CodeNotAJSONObject                      = "not_a_json_object"
	CodeInvalidSubjectData                  = "invalid_subject_data"
	CodeUnsupportedLayout                   = "unsupported_layout"
	CodeInvalidSubjectType                  = "invalid_subject_type"
	CodeMissingBlock                        = "missing_block"
	CodeMissingMetadata                     = "missing_metadata"
	CodeNoExternalCommitments               = "no_external_commitments"
	CodeInvalidCommitmentEntry              = "invalid_commitment_entry"
	CodeUnexpectedSubjectFieldsForBlockLike = "unexpected_subject_fields_for_block_like"
	CodeMissingSubject                      = "missing_subject"
	CodeMissingInclusionProof               = "missing_inclusion_proof"

	// CodeSubjectTypeMismatch is the one rejection Appendix E does not
	// define: it is raised when the caller asserted a subject type
	// (`verify --type`) and the bundle's own signed `type` differs. The
	// server names the same condition the same way on /proof/verify. It
	// is a rejection rather than a graded row because E.25 forbids a
	// verifier adding a `fail` row for a step Appendix D.4 does not
	// report.
	CodeSubjectTypeMismatch = "subject_type_mismatch"
)

// RejectionError is a structural hard rejection: the bundle is malformed in
// a way E.6 says MUST abort before any step runs, so no Report is produced.
// Code is the E.23 identifier; Detail is the human-facing explanation.
type RejectionError struct {
	Code   string
	Detail string
}

func (e *RejectionError) Error() string { return e.Code + ": " + e.Detail }

// Advice returns the one-line operator instruction the reference verifier
// prints under a rejection.
func (e *RejectionError) Advice() string { return RejectionAdvice(e.Code) }

// RejectionAdvice returns the operator instruction for a rejection code,
// matching the reference verifier's wording.
func RejectionAdvice(code string) string {
	switch code {
	case CodeUnsupportedLayout:
		return "This file uses the pre-publication draft layout, which version 1 replaced.\n" +
			"Ask the holder to regenerate the proof; no verifier reads that layout."
	case CodeMissingMetadata:
		return "A bundle carries its metadata maps, never their hashes, so a verifier can\n" +
			"recompute every digest. This one omits a metadata map, so no steps were run."
	case CodeSubjectTypeMismatch:
		return "The bundle's signed type is authoritative. Drop --type, or pass the type the\n" +
			"bundle carries; the filename is never consulted."
	default:
		return "This input is not a well-formed proof bundle, so no steps were run."
	}
}

// RejectionCode returns the E.23 identifier carried by err, or "" when err
// is not a hard rejection. Callers use it to distinguish "this bundle was
// refused at the structural layer" from "verification produced a report".
func RejectionCode(err error) string {
	var re *RejectionError
	if errors.As(err, &re) {
		return re.Code
	}
	return ""
}

// Rejectf builds a RejectionError with a formatted detail.
func Rejectf(code, format string, args ...any) error {
	return &RejectionError{Code: code, Detail: fmt.Sprintf(format, args...)}
}

// rejectSubjectData builds the E.3 value-space rejection, matching the
// reference implementation's message shape so the two are diffable.
func rejectSubjectData(path, detail string) error {
	return Rejectf(CodeInvalidSubjectData, "%s: %s cannot round-trip to JSON (whitepaper E.3)", path, detail)
}
