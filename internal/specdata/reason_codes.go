// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.ReasonCodes
// (elixir/lib/truestamp/verifier/reason_codes.ex in truestamp-verification):
// the reason codes of SPEC.md section 12.1, transcribed from
// spec/reason-codes.json in its order, which is step order, with the
// sub_reason values the rows of malformed_json and invalid_envelope list
// (SPEC.md sections 10.1 and 10.2, step 3), and held equal to the file by
// TestAgreement_ReasonCodesWithTheirSubReasons. No other code is valid. Code
// that names a code writes its constant, such as ReasonMalformedJSON, so a
// name the table lacks does not compile, as the reference's code/1 macro
// stops its build.

package specdata

import (
	"fmt"
	"slices"
)

// Reason is a reason code of SPEC.md section 12.1.
type Reason string

// SubReason is a value the sub_reason of a malformed_json or
// invalid_envelope rejection takes.
type SubReason string

// Status is a reason code's status: StatusBare ends the run in a bare
// rejection, and the others are row statuses.
type Status string

// The statuses spec/reason-codes.json writes.
const (
	StatusBare Status = "bare"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
	StatusInfo Status = "info"
	StatusWarn Status = "warn"
)

// ReasonEntry is one row of the table of SPEC.md section 12.1.
type ReasonEntry struct {
	Code   Reason
	Status Status
	// Step is the step as the table writes it, such as "DERIVE, CHAIN" for
	// a code of two steps.
	Step    string
	Meaning string
	// SubReasons lists the values the code's sub_reason takes, in SPEC.md's
	// order. It is empty for every code but malformed_json and
	// invalid_envelope, since no other code carries a sub_reason.
	SubReasons []SubReason
}

// The reason codes of SPEC.md section 12.1, in the table's order.
const (
	ReasonConfigurationError        Reason = "configuration_error"
	ReasonInputTooLarge             Reason = "input_too_large"
	ReasonMalformedJSON             Reason = "malformed_json"
	ReasonInvalidEnvelope           Reason = "invalid_envelope"
	ReasonEnvironmentMismatch       Reason = "environment_mismatch"
	ReasonKeyringUnavailable        Reason = "keyring_unavailable"
	ReasonKeyBindingFailed          Reason = "key_binding_failed"
	ReasonKeyBindingNotPerformed    Reason = "key_binding_not_performed"
	ReasonSignatureInvalid          Reason = "signature_invalid"
	ReasonUnsupportedFormat         Reason = "unsupported_format"
	ReasonUnsupportedVersion        Reason = "unsupported_version"
	ReasonShapeInvalid              Reason = "shape_invalid"
	ReasonConstructionMismatch      Reason = "construction_mismatch"
	ReasonNetworkMismatch           Reason = "network_mismatch"
	ReasonDerivationMismatch        Reason = "derivation_mismatch"
	ReasonBitcoinEvidenceInvalid    Reason = "bitcoin_evidence_invalid"
	ReasonKeyEventMismatch          Reason = "key_event_mismatch"
	ReasonOrderingViolation         Reason = "ordering_violation"
	ReasonNotAttributable           Reason = "not_attributable"
	ReasonOffline                   Reason = "offline"
	ReasonChainMismatch             Reason = "chain_mismatch"
	ReasonSourceUnreachable         Reason = "source_unreachable"
	ReasonNotHeldBySource           Reason = "not_held_by_source"
	ReasonNotYetIndexed             Reason = "not_yet_indexed"
	ReasonChainNotFinal             Reason = "chain_not_final"
	ReasonEntropyMismatch           Reason = "entropy_mismatch"
	ReasonWindowNotEstablished      Reason = "window_not_established"
	ReasonWindowInverted            Reason = "window_inverted"
	ReasonRecordedTimeOutsideWindow Reason = "recorded_time_outside_window"
	ReasonExpectedHashMismatch      Reason = "expected_hash_mismatch"
	ReasonExpectedHashWeakAlgorithm Reason = "expected_hash_weak_algorithm"
	ReasonExpectedHashNotChecked    Reason = "expected_hash_not_checked"
	ReasonPriorFailure              Reason = "prior_failure"
	ReasonInternalError             Reason = "internal_error"
)

// The sub_reason values of malformed_json (SPEC.md section 10.1) and of
// invalid_envelope (SPEC.md section 10.2, step 3), each in its order.
const (
	SubReasonSyntax        SubReason = "syntax"
	SubReasonInvalidUTF8   SubReason = "invalid_utf8"
	SubReasonLoneSurrogate SubReason = "lone_surrogate"
	SubReasonDuplicateName SubReason = "duplicate_name"
	SubReasonNumber        SubReason = "number"
	SubReasonDepth         SubReason = "depth"
	SubReasonDocument      SubReason = "document"
	SubReasonProof         SubReason = "proof"
	SubReasonSignature     SubReason = "signature"
	SubReasonAlg           SubReason = "alg"
	SubReasonPublicKey     SubReason = "public_key"
	SubReasonValue         SubReason = "value"
)

// reasonTable is spec/reason-codes.json, in its order.
var reasonTable = []ReasonEntry{
	{
		Code:    ReasonConfigurationError,
		Status:  StatusBare,
		Step:    "CONFIGURATION",
		Meaning: "a check of step 0 fails (section 10.2): a keyring file or configured keyring fails its load checks, a run whose environment is not prod is not keyless and has no keyring source (section 6.4), an online run pins a network it has no source for, or a source's identity read gives a value step 0 forbids",
	},
	{
		Code:    ReasonInputTooLarge,
		Status:  StatusBare,
		Step:    "READ",
		Meaning: "the file is over MAX_READ_OCTETS",
	},
	{
		Code:       ReasonMalformedJSON,
		Status:     StatusBare,
		Step:       "PARSE",
		Meaning:    "the file breaks a parsing rule of section 10.1; sub_reason names the rule",
		SubReasons: []SubReason{SubReasonSyntax, SubReasonInvalidUTF8, SubReasonLoneSurrogate, SubReasonDuplicateName, SubReasonNumber, SubReasonDepth},
	},
	{
		Code:       ReasonInvalidEnvelope,
		Status:     StatusBare,
		Step:       "ENVELOPE",
		Meaning:    "the document is not exactly the envelope of section 3.1: a member missing or extra, alg other than the pin, K or V off its class, or K failing the checks of a key's bytes (section 6.2); sub_reason names the first check of section 10.2, step 3, that fails",
		SubReasons: []SubReason{SubReasonDocument, SubReasonProof, SubReasonSignature, SubReasonAlg, SubReasonPublicKey, SubReasonValue},
	},
	{
		Code:    ReasonEnvironmentMismatch,
		Status:  StatusFail,
		Step:    "ENVIRONMENT",
		Meaning: "environment fails the check of section 6.4",
	},
	{
		Code:    ReasonKeyringUnavailable,
		Status:  StatusFail,
		Step:    "KEY_BINDING",
		Meaning: "the keyring that would govern cannot be used: Truestamp's key chain is not verified, or an external verifier's online keyring read failed, so no key binds (sections 8.2 and 8.3)",
	},
	{
		Code:    ReasonKeyBindingFailed,
		Status:  StatusFail,
		Step:    "KEY_BINDING",
		Meaning: "K is not in the keyring that governs (section 6.1); a revoked key is in no current keyring (section 8.1)",
	},
	{
		Code:    ReasonKeyBindingNotPerformed,
		Status:  StatusSkip,
		Step:    "KEY_BINDING",
		Meaning: "keyless mode, which the verifier states in words (section 8.3)",
	},
	{
		Code:    ReasonSignatureInvalid,
		Status:  StatusFail,
		Step:    "SIGNATURE",
		Meaning: "V does not verify under K over proof_hash (section 6.1) by the strict rules of section 6.2, including an s that is not below L",
	},
	{
		Code:    ReasonUnsupportedFormat,
		Status:  StatusFail,
		Step:    "SHAPE",
		Meaning: "format is not \"truestamp-proof\"",
	},
	{
		Code:    ReasonUnsupportedVersion,
		Status:  StatusFail,
		Step:    "SHAPE",
		Meaning: "version is not 1; the only signal of a later format",
	},
	{
		Code:    ReasonShapeInvalid,
		Status:  StatusFail,
		Step:    "SHAPE",
		Meaning: "the proof fails a check of SHAPE other than those of format, version and the constructions digest (section 10.2, step 7)",
	},
	{
		Code:    ReasonConstructionMismatch,
		Status:  StatusFail,
		Step:    "SHAPE",
		Meaning: "the constructions digest is not in CONSTRUCTIONS_ACCEPTED",
	},
	{
		Code:    ReasonNetworkMismatch,
		Status:  StatusFail,
		Step:    "POLICY",
		Meaning: "a network differs from the verifier's pin for that chain or source",
	},
	{
		Code:    ReasonDerivationMismatch,
		Status:  StatusFail,
		Step:    "DERIVE",
		Meaning: "a CHECKED value that no key-event check of section 10.3 covers differs from its recomputation (section 10.2, step 9): a record hash; a walk that fails or reaches another root; a tree_size that differs from its record's leaf_count; an epoch_hash that differs from the copy in the carried Bitcoin transaction; the hash a txoutproof matches, or a Bitcoin transaction_hash, header_hash, header_merkle_root or header_time; a LINK_WITNESS link",
	},
	{
		Code:    ReasonBitcoinEvidenceInvalid,
		Status:  StatusFail,
		Step:    "DERIVE",
		Meaning: "a raw_transaction or txoutproof that is not well formed, including a header hash above the target its own nBits encodes (section 10.2, step 9)",
	},
	{
		Code:    ReasonKeyEventMismatch,
		Status:  StatusFail,
		Step:    "DERIVE",
		Meaning: "a key event fails a check of section 10.3 (its ids, its position, its handover signature, or the checks of its public_key's bytes), or the event at /signing_key_event/block does not introduce K (LINK_KEY_EVENT)",
	},
	{
		Code:    ReasonOrderingViolation,
		Status:  StatusFail,
		Step:    "DERIVE",
		Meaning: "LINK_HEAD or LINK_CHAIN_ORDER fails (section 10.3)",
	},
	{
		Code:    ReasonNotAttributable,
		Status:  StatusInfo,
		Step:    "DERIVE, CHAIN",
		Meaning: "on LINK_KEY_EVENT or KEY_EVENT_COMMITMENT, in place of pass when the attribution is not truestamp",
	},
	{
		Code:    ReasonOffline,
		Status:  StatusSkip,
		Step:    "CHAIN, SOURCES",
		Meaning: "no read was made in this run; the message names the read that would confirm the row",
	},
	{
		Code:    ReasonChainMismatch,
		Status:  StatusFail,
		Step:    "CHAIN",
		Meaning: "a chain read contradicts the proof (section 10.4)",
	},
	{
		Code:    ReasonSourceUnreachable,
		Status:  StatusSkip,
		Step:    "CHAIN, SOURCES",
		Meaning: "the source gave no usable answer (section 10.4)",
	},
	{
		Code:    ReasonNotHeldBySource,
		Status:  StatusSkip,
		Step:    "CHAIN, SOURCES",
		Meaning: "the source does not hold the value, and its absence is no contradiction",
	},
	{
		Code:    ReasonNotYetIndexed,
		Status:  StatusSkip,
		Step:    "CHAIN, SOURCES",
		Meaning: "the value lies beyond the source's indexed tip, or within STELLAR_INDEX_MARGIN of it",
	},
	{
		Code:    ReasonChainNotFinal,
		Status:  StatusSkip,
		Step:    "CHAIN",
		Meaning: "the Bitcoin tip is below height + BITCOIN_FINALITY",
	},
	{
		Code:    ReasonEntropyMismatch,
		Status:  StatusFail,
		Step:    "SOURCES",
		Meaning: "the pulse, ledger or block read differs from the entropy object",
	},
	{
		Code:    ReasonWindowNotEstablished,
		Status:  StatusInfo,
		Step:    "WINDOW",
		Meaning: "no submitted-before edge was drawn; values give the reason for each absent edge",
	},
	{
		Code:    ReasonWindowInverted,
		Status:  StatusFail,
		Step:    "WINDOW",
		Meaning: "the submitted-after edge is later than the submitted-before edge",
	},
	{
		Code:    ReasonRecordedTimeOutsideWindow,
		Status:  StatusInfo,
		Step:    "WINDOW",
		Meaning: "the time in a subject's id lies outside an established window",
	},
	{
		Code:    ReasonExpectedHashMismatch,
		Status:  StatusFail,
		Step:    "EXPECTED_HASH",
		Meaning: "the caller gave a hash, and the proof has no claims.hash of a listed type to compare it with, or the caller's hash is not hex of that type's length, or it differs from claims.hash (section 10.2, step 13)",
	},
	{
		Code:    ReasonExpectedHashWeakAlgorithm,
		Status:  StatusWarn,
		Step:    "EXPECTED_HASH",
		Meaning: "the caller's hash matched, and claims.hash_type is md5 or sha1, whose collisions are practical",
	},
	{
		Code:    ReasonExpectedHashNotChecked,
		Status:  StatusSkip,
		Step:    "EXPECTED_HASH",
		Meaning: "no expected hash was given",
	},
	{
		Code:    ReasonPriorFailure,
		Status:  StatusSkip,
		Step:    "any step after a failure",
		Meaning: "a row it depends on failed: every later step up to EXPECTED_HASH after a failure at steps 4 to 7 (section 10.2); a CHAIN or SOURCES row, KEY_EVENT_COMMITMENT included, after a failed row before CHAIN (the external gate of section 10.2); and the WINDOW row after any failed row (section 11.3)",
	},
	{
		Code:    ReasonInternalError,
		Status:  StatusFail,
		Step:    "any step from ENVIRONMENT on",
		Meaning: "a fault in the verifier (section 10.5)",
	},
}

// reasonIndex finds a code's row in reasonTable.
var reasonIndex = indexReasons(reasonTable)

// indexReasons maps each code to its row, and stops the program on a code
// listed twice, as the reference stops its build.
func indexReasons(table []ReasonEntry) map[Reason]int {
	index := make(map[Reason]int, len(table))
	for i, entry := range table {
		if _, seen := index[entry.Code]; seen {
			panic(fmt.Sprintf("specdata: reason code %s appears twice", entry.Code))
		}
		index[entry.Code] = i
	}
	return index
}

// ReasonCodes returns the table of SPEC.md section 12.1, in its order. Each
// call returns a new slice, sharing nothing with the table.
func ReasonCodes() []ReasonEntry {
	out := make([]ReasonEntry, len(reasonTable))
	for i, entry := range reasonTable {
		entry.SubReasons = slices.Clone(entry.SubReasons)
		out[i] = entry
	}
	return out
}

// LookupReason returns the row of code, and whether code is in the table.
func LookupReason(code Reason) (ReasonEntry, bool) {
	i, ok := reasonIndex[code]
	if !ok {
		return ReasonEntry{}, false
	}
	entry := reasonTable[i]
	entry.SubReasons = slices.Clone(entry.SubReasons)
	return entry, true
}

// IsReason reports whether name is a reason code of the table.
func IsReason(name string) bool {
	_, ok := reasonIndex[Reason(name)]
	return ok
}

// SubReasons returns the values the sub_reason of code takes, in SPEC.md's
// order: those of malformed_json and of invalid_envelope, and none for any
// other code.
func SubReasons(code Reason) []SubReason {
	entry, _ := LookupReason(code)
	return entry.SubReasons
}
