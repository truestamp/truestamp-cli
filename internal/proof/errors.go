// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"errors"
	"fmt"
)

// Hard-rejection identifiers from the whitepaper's error taxonomy (E.23).
// A hard rejection aborts before any report exists (E.6) — independent
// verifiers are required to agree on this vocabulary so their outcomes are
// comparable, which is only possible if the identifier travels with the
// error rather than living in prose.
const (
	CodeNotAJSONObject                      = "not_a_json_object"
	CodeMissingTypeCode                     = "missing_type_code"
	CodeInvalidSubjectTypeCode              = "invalid_subject_type_code"
	CodeMissingBlock                        = "missing_block"
	CodeNoExternalCommitments               = "no_external_commitments"
	CodeInvalidExternalCommitmentEntry      = "invalid_external_commitment_entry"
	CodeUnexpectedSubjectFieldsForBlockLike = "unexpected_subject_fields_for_block_like"
	CodeMissingSubject                      = "missing_subject"
	CodeMissingInclusionProof               = "missing_inclusion_proof"

	// CodeInvalidSubjectData covers a CBOR `s.d` value that has no JSON
	// counterpart (E.3). E.23 names no identifier for this class; this one
	// is taken verbatim from the Elixir reference verifier
	// (truestamp-v2 lib/truestamp/proof/binary.ex) so both implementations
	// report the same word for the same rejection.
	CodeInvalidSubjectData = "invalid_subject_data"
)

// RejectionError is a structural hard rejection: the bundle is malformed in
// a way E.6 says MUST abort before any step runs, so no Report is produced.
// Code is the E.23 identifier; Detail is the human-facing explanation.
type RejectionError struct {
	Code   string
	Detail string
}

func (e *RejectionError) Error() string { return e.Code + ": " + e.Detail }

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

// rejectf builds a RejectionError with a formatted detail.
func rejectf(code, format string, args ...any) error {
	return &RejectionError{Code: code, Detail: fmt.Sprintf(format, args...)}
}

// rejectSubjectData builds the E.3 subject-data rejection, matching the
// Elixir reference verifier's message shape so the two are diffable.
func rejectSubjectData(path, detail string) error {
	return rejectf(CodeInvalidSubjectData, "%s: %s cannot round-trip to JSON", path, detail)
}
