// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import "sort"

// JSONOutput is the structured output for --json mode.
// It mirrors the visual terminal output sections.
type JSONOutput struct {
	Result      string `json:"result"`
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`

	// SignaturesChecked is the machine-readable form of the verdict
	// line's --skip-signatures disclosure. E.25 does not list E.16 among
	// the steps a verifier MAY skip and still call a run verified, so
	// `result` alone is not enough: it reads "verified" whether the
	// Ed25519 signature was confirmed or never looked at.
	SignaturesChecked bool `json:"signatures_checked"`

	Subject        any              `json:"subject"`
	HashComparison HashComparison   `json:"hash_comparison"`
	Timeline       *JSONTimeline    `json:"timeline,omitempty"`
	Commitments    *JSONCommitments `json:"commitments,omitempty"`

	// Steps is the complete step record, in Appendix E.22's category
	// order. It precedes the two filtered views below because those
	// drop rows: `issues` keeps only failures and warnings, and
	// `verification_notes` only one group, so every skip and info row —
	// including three of Appendix D.4's fourteen (Key Binding, Stellar
	// Commitment, Bitcoin Commitment) — used to survive --json as
	// nothing but a count. Appendix E.25 requires that no step D.4
	// reports be absent from a verifier's output.
	Steps []Step `json:"steps"`

	VerificationNotes []JSONNote  `json:"verification_notes,omitempty"`
	Issues            []JSONIssue `json:"issues,omitempty"`
	Summary           JSONSummary `json:"summary"`
}

// JSONNote is a workflow-level observation about the verify session
// that is NOT a proof defect. Examples: "you can pass --hash to
// confirm a local file" (severity warning), or "this is a claims-only
// item, no external file to compare" (severity info). Notes are
// emitted separately from issues so consumers can render them as
// optional follow-ups rather than failed checks.
type JSONNote struct {
	Severity string `json:"severity"` // "warning" | "info"
	Message  string `json:"message"`
}

// HashComparison reports Appendix E.7's expected-hash check. `supplied`
// and `matched` are distinct facts and both are always emitted: E.7
// requires a consumer to be able to tell "no expected hash was given"
// from "one was given and did not match". Inferring the first from the
// object's absence does not work, because the step can also be reported
// as a skip for a subject that commits to no file hash at all.
type HashComparison struct {
	Supplied bool   `json:"supplied"`
	Matched  bool   `json:"matched"`
	Provided string `json:"provided,omitempty"` // the caller's expected hash
	Found    string `json:"found,omitempty"`    // the hash carried in the proof
}

// JSONTimeline holds the verified temporal bracket. CapturedAt carries the
// observation's record-into-Truestamp time and is emitted as `inserted_at`
// to match the server's renamed wire key; the Go field keeps its CapturedAt
// name, mirroring TemporalSummary. (Distinct from the entropy subject's own
// source-capture time in buildSubject, which stays `captured_at`.)
type JSONTimeline struct {
	ClaimedAt   string `json:"claimed_at,omitempty"`
	SubmittedAt string `json:"submitted_at,omitempty"`
	CapturedAt  string `json:"inserted_at,omitempty"`
	CommittedAt string `json:"committed_at,omitempty"`
}

// JSONCommitments holds structured commitment data keyed by blockchain.
type JSONCommitments struct {
	Truestamp *TruestampCommitment  `json:"truestamp,omitempty"`
	Stellar   *BlockchainCommitment `json:"stellar,omitempty"`
	Bitcoin   *BlockchainCommitment `json:"bitcoin,omitempty"`
}

// TruestampCommitment holds the internal chain summary.
type TruestampCommitment struct {
	ChainLength  int    `json:"chain_length"`
	SigningKeyID string `json:"signing_key_id"`
}

// BlockchainCommitment holds data for a Stellar or Bitcoin commitment.
//
// ExternalCheck is the machine-readable form of [ExternalStatus]'s three
// states and ExternallyVerified is its `== "confirmed"` projection. The
// boolean alone collapsed "the chain answered and disagreed" and "no
// lookup was attempted" into one `false`, which is the collapse the
// tri-state was introduced to remove — but only the terminal presenter
// had been taught the difference, so a JSON consumer still got the
// boolean back.
type BlockchainCommitment struct {
	Network             string `json:"network"`
	Ledger              int    `json:"ledger,omitempty"`
	BlockHeight         int    `json:"block_height,omitempty"`
	Timestamp           string `json:"timestamp,omitempty"`
	TxHash              string `json:"tx_hash"`
	CommittedHashHex    string `json:"committed_hash_hex"`
	CommittedHashBase64 string `json:"committed_hash_base64"`
	ExternalCheck       string `json:"external_check"` // "confirmed" | "skipped" | "failed"
	ExternallyVerified  bool   `json:"externally_verified"`
}

// JSONIssue represents a non-passing verification check. It carries the
// same rows the terminal's "Issues" section renders — failures,
// warnings AND skips — because the two surfaces are two renderings of
// one Report and a consumer comparing them must not find the same row in
// different buckets. Three of Appendix D.4's fourteen rows are skips;
// dropping them here left `issues` absent from the --json document of a
// run whose terminal output printed an Issues heading with three
// entries.
type JSONIssue struct {
	Severity string `json:"severity"` // "error", "warning", "skipped"
	Category string `json:"category"`
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
}

// JSONSummary holds step counts, one field per Appendix E.22 status
// plus a total that includes all five. `info` used to be both absent
// and excluded from `total`, so summing the summary gave a different
// number than iterating `steps` with nothing explaining the gap.
type JSONSummary struct {
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
	Skipped  int `json:"skipped"`
	Info     int `json:"info"`
	Total    int `json:"total"`
}

// BuildJSONOutput creates a presentation DTO from the internal Report.
func BuildJSONOutput(r *Report) *JSONOutput {
	c := r.Counts()
	out := &JSONOutput{
		Result:            computeResult(r),
		SubjectType:       r.SubjectType,
		SubjectID:         r.SubjectID,
		SignaturesChecked: !r.SignaturesSkipped(),
		Subject:           buildSubject(r),
		Summary: JSONSummary{
			Passed:   c.Passed,
			Failed:   c.Failed,
			Warnings: c.Warned,
			Skipped:  c.Skipped,
			Info:     c.Info,
			Total:    c.Total,
		},
	}

	// Hash comparison. Emitted unconditionally so "supplied" carries
	// the answer rather than the object's presence (E.7).
	out.HashComparison = HashComparison{
		Supplied: r.HashProvided != "",
		Matched:  r.HashMatched(),
		Provided: r.HashProvided,
		Found:    r.Claims.Hash,
	}

	// Timeline
	t := r.Temporal
	if t.ClaimedAt != "" || t.SubmittedAt != "" || t.CapturedAt != "" || t.CommittedAt != "" {
		out.Timeline = &JSONTimeline{
			ClaimedAt:   t.ClaimedAt,
			SubmittedAt: t.SubmittedAt,
			CapturedAt:  t.CapturedAt,
			CommittedAt: t.CommittedAt,
		}
	}

	// Commitments
	if r.ChainLength > 0 {
		c := &JSONCommitments{
			Truestamp: &TruestampCommitment{
				ChainLength:  r.ChainLength,
				SigningKeyID: r.SigningKeyID,
			},
		}
		for _, ci := range r.CommitmentInfos {
			bc := &BlockchainCommitment{
				Network:             ci.Network,
				Timestamp:           ci.Timestamp,
				TxHash:              ci.TxHash,
				CommittedHashHex:    ci.CommittedHash,
				CommittedHashBase64: HexToBase64(ci.CommittedHash),
				ExternalCheck:       ci.ExternalCheck.String(),
				// Only a lookup that ran and agreed may claim this.
				ExternallyVerified: ci.ExternalCheck == ExternalConfirmed,
			}
			switch ci.Method {
			case "stellar":
				bc.Ledger = ci.Ledger
				c.Stellar = bc
			case "bitcoin":
				bc.BlockHeight = ci.Height
				c.Bitcoin = bc
			}
		}
		out.Commitments = c
	}

	out.Steps = buildSteps(r)

	// Issues and Verification Notes partition the non-passing rows
	// exactly the way renderIssues and renderVerificationNotes do, so
	// the two surfaces never file the same row under different headings:
	//
	//   notes  = the Verification Notes group (any status) ∪ every info row
	//   issues = every remaining fail / warn / skip
	//
	// Passing rows appear only in `steps`.
	for _, s := range r.Steps {
		if s.Group == groupVerificationNotes || s.Status == StatusInfo {
			severity := "info"
			if s.Status == StatusWarn {
				severity = "warning"
			}
			out.VerificationNotes = append(out.VerificationNotes, JSONNote{
				Severity: severity,
				Message:  s.Message,
			})
			continue
		}
		if s.Status == StatusPass {
			continue
		}
		severity := "error"
		switch s.Status {
		case StatusWarn:
			severity = "warning"
		case StatusSkip:
			severity = "skipped"
		}
		cat := s.Category
		if cat == "" {
			cat = CatStructural
		}
		issue := JSONIssue{
			Severity: severity,
			Category: cat,
			Message:  s.Message,
		}
		// Details explain a consequence, so they belong only where
		// something went wrong — the same rule renderIssues applies.
		if s.Status == StatusFail || s.Status == StatusWarn {
			issue.Detail = lookupFailureDetail(s.Message)
		}
		out.Issues = append(out.Issues, issue)
	}

	return out
}

// buildSteps returns every Report step in Appendix E.22's category
// order, ready to marshal.
func buildSteps(r *Report) []Step {
	// Never nil: an empty step list must marshal as [] so a consumer
	// can iterate it without a null check.
	steps := make([]Step, 0, len(r.Steps))
	for _, s := range r.Steps {
		// Mirror the coercion the issues array and the presenter apply,
		// so a consumer never sees an empty category string.
		if s.Category == "" {
			s.Category = CatStructural
		}
		steps = append(steps, s)
	}
	// Stable, and keyed on the category alone: emit order within a
	// category carries meaning (Epoch Proof 0 before Epoch Proof 1,
	// cx order for the commitment rows) and must survive the grouping.
	// Sorting by status as well — the way the terminal Issues section
	// does — would destroy it.
	sort.SliceStable(steps, func(i, j int) bool {
		return categoryRank(steps[i].Category) < categoryRank(steps[j].Category)
	})
	return steps
}

// computeResult maps the report's verdict onto the --json `result`
// vocabulary. The mapping lives here rather than on Verdict so the wire
// strings stay in the DTO that publishes them.
func computeResult(r *Report) string {
	switch r.Verdict() {
	case VerdictFullyVerified:
		return "fully_verified"
	case VerdictHashMismatch:
		return "hash_mismatch"
	case VerdictFailed:
		return "failed"
	default:
		return "verified"
	}
}

func buildSubject(r *Report) any {
	switch r.SubjectType {
	// Beacon (t=11) and plain block (t=10) share one wire shape — the
	// block IS the subject — so they share one subject projection.
	// "beacon" used to fall through to the item default, which found no
	// claims and published an empty object, silently dropping block_id,
	// signing_key and committed_at from the machine-readable surface for
	// exactly the subject type the t=11 cutover made first-class.
	case "block", "beacon":
		return map[string]any{
			"block_id": r.SubjectID,
			// The block's own b.kid, which is what a field of the block
			// means. The signer (E.9's pk-derived key id, which may
			// differ under rotation) is published under
			// commitments.truestamp.signing_key_id.
			"signing_key":  r.BlockKeyID(),
			"committed_at": r.Temporal.CommittedAt,
		}

	case "entropy_nist", "entropy_stellar", "entropy_bitcoin":
		m := map[string]any{}
		if r.EntropySubject.RawSource != "" {
			m["source"] = r.EntropySubject.RawSource
		}
		// Source capture time (NIST pulse / BTC block / Stellar close): when the
		// entropy was produced upstream. This is NOT the server's renamed
		// inserted_at (Truestamp record time, in timeline) — it maps to the
		// server's unchanged source-chain `timestamp`, so it stays captured_at.
		if r.EntropySubject.CapturedAt != "" {
			m["captured_at"] = r.EntropySubject.CapturedAt
		}
		switch r.EntropySubject.RawSource {
		case "entropy_nist":
			if r.EntropySubject.PulseIndex > 0 {
				m["pulse_index"] = r.EntropySubject.PulseIndex
			}
			if r.EntropySubject.ChainIndex > 0 {
				m["chain_index"] = r.EntropySubject.ChainIndex
			}
			if r.EntropySubject.Version != "" {
				m["version"] = r.EntropySubject.Version
			}
			if r.EntropySubject.OutputValue != "" {
				m["output_value"] = r.EntropySubject.OutputValue
			}
		case "entropy_bitcoin":
			if r.EntropySubject.BlockHeight > 0 {
				m["block_height"] = r.EntropySubject.BlockHeight
			}
			if r.EntropySubject.BlockHash != "" {
				m["block_hash"] = r.EntropySubject.BlockHash
			}
			if r.EntropySubject.BlockTime > 0 {
				m["block_time"] = r.EntropySubject.BlockTime
			}
		case "entropy_stellar":
			if r.EntropySubject.LedgerSequence > 0 {
				m["ledger_sequence"] = r.EntropySubject.LedgerSequence
			}
			if r.EntropySubject.LedgerHash != "" {
				m["ledger_hash"] = r.EntropySubject.LedgerHash
			}
			if r.EntropySubject.LedgerClosedAt != "" {
				m["ledger_closed_at"] = r.EntropySubject.LedgerClosedAt
			}
		}
		return m
	}

	// Default: item
	m := map[string]any{}
	if r.Claims.Name != "" {
		m["name"] = r.Claims.Name
	}
	if r.Claims.Hash != "" {
		m["hash"] = r.Claims.Hash
	}
	if r.Claims.HashType != "" {
		m["hash_type"] = r.Claims.HashType
	}
	if r.Claims.Description != "" {
		m["description"] = r.Claims.Description
	}
	if r.Claims.Timestamp != "" {
		m["timestamp"] = r.Claims.Timestamp
	}
	if r.Claims.URL != "" {
		m["url"] = r.Claims.URL
	}
	if r.Claims.Location != nil {
		m["location"] = r.Claims.Location
	}
	if r.Claims.HasMetadata {
		m["has_metadata"] = true
	}
	return m
}
