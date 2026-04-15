// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

// JSONOutput is the structured output for --json mode.
// It mirrors the visual terminal output sections.
type JSONOutput struct {
	Result         string           `json:"result"`
	SubjectType    string           `json:"subject_type"`
	SubjectID      string           `json:"subject_id"`
	Subject        any              `json:"subject"`
	HashComparison *HashComparison  `json:"hash_comparison,omitempty"`
	Timeline       *JSONTimeline    `json:"timeline,omitempty"`
	Commitments    *JSONCommitments `json:"commitments,omitempty"`
	Issues         []JSONIssue      `json:"issues,omitempty"`
	Summary        JSONSummary      `json:"summary"`
}

// HashComparison shows the result of --hash verification.
type HashComparison struct {
	Provided string `json:"provided"`
	Found    string `json:"found"`
	Matched  bool   `json:"matched"`
}

// JSONTimeline holds the verified temporal bracket.
type JSONTimeline struct {
	ClaimedAt   string `json:"claimed_at,omitempty"`
	SubmittedAt string `json:"submitted_at,omitempty"`
	CapturedAt  string `json:"captured_at,omitempty"`
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
type BlockchainCommitment struct {
	Network             string `json:"network"`
	Ledger              int    `json:"ledger,omitempty"`
	BlockHeight         int    `json:"block_height,omitempty"`
	Timestamp           string `json:"timestamp,omitempty"`
	TxHash              string `json:"tx_hash"`
	CommittedHashHex    string `json:"committed_hash_hex"`
	CommittedHashBase64 string `json:"committed_hash_base64"`
	ExternallyVerified  bool   `json:"externally_verified"`
}

// JSONIssue represents a non-passing verification check.
type JSONIssue struct {
	Severity string `json:"severity"` // "error", "warning"
	Category string `json:"category"`
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
}

// JSONSummary holds check counts.
type JSONSummary struct {
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
	Skipped  int `json:"skipped"`
	Total    int `json:"total"`
}

// BuildJSONOutput creates a presentation DTO from the internal Report.
func BuildJSONOutput(r *Report) *JSONOutput {
	c := r.Counts()
	out := &JSONOutput{
		Result:      computeResult(r),
		SubjectType: r.SubjectType,
		SubjectID:   r.SubjectID,
		Subject:     buildSubject(r),
		Summary: JSONSummary{
			Passed:   c.Passed,
			Failed:   c.Failed,
			Warnings: c.Warned,
			Skipped:  c.Skipped,
			Total:    c.Total,
		},
	}

	// Hash comparison
	if r.HashProvided != "" {
		out.HashComparison = &HashComparison{
			Provided: r.HashProvided,
			Found:    r.Claims.Hash,
			Matched:  r.HashMatched(),
		}
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
				ExternallyVerified:  !ci.Skipped,
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

	// Issues (failures and warnings only)
	for _, s := range r.Steps {
		if s.Status != StatusFail && s.Status != StatusWarn {
			continue
		}
		severity := "error"
		if s.Status == StatusWarn {
			severity = "warning"
		}
		cat := s.Category
		if cat == "" {
			cat = CatStructural
		}
		out.Issues = append(out.Issues, JSONIssue{
			Severity: severity,
			Category: cat,
			Message:  s.Message,
			Detail:   lookupFailureDetail(s.Message),
		})
	}

	return out
}

func computeResult(r *Report) string {
	proofOK := r.ProofPassed()
	hashProvided := r.HashProvided != ""
	hashOK := r.HashMatched()

	switch {
	case !proofOK:
		return "failed"
	case hashProvided && !hashOK:
		return "hash_mismatch"
	case hashProvided && hashOK:
		return "fully_verified"
	default:
		return "verified"
	}
}

func buildSubject(r *Report) any {
	if r.SubjectType == "entropy" {
		m := map[string]any{}
		if r.EntropySubject.RawSource != "" {
			m["source"] = r.EntropySubject.RawSource
		}
		if r.EntropySubject.CapturedAt != "" {
			m["captured_at"] = r.EntropySubject.CapturedAt
		}
		// Source-specific fields
		switch r.EntropySubject.RawSource {
		case "nist_beacon":
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
		case "bitcoin_block":
			if r.EntropySubject.BlockHeight > 0 {
				m["block_height"] = r.EntropySubject.BlockHeight
			}
			if r.EntropySubject.BlockHash != "" {
				m["block_hash"] = r.EntropySubject.BlockHash
			}
			if r.EntropySubject.BlockTime > 0 {
				m["block_time"] = r.EntropySubject.BlockTime
			}
		case "stellar_ledger":
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
