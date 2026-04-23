// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/bitcoin"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// RemoteOptions holds configuration for remote verification.
type RemoteOptions struct {
	APIURL       string
	APIKey       string
	Team         string // team ID, sent as tenant header
	ExpectedHash string // hex hash to compare against claims.hash
}

// apiEnvelope wraps the top-level API response.
type apiEnvelope struct {
	Result *apiResult `json:"result"`
	Errors []apiError `json:"errors"`
}

// apiResult is the verification result from the server.
type apiResult struct {
	ProofVersion    *int            `json:"proof_version"`
	SubjectID       *string         `json:"subject_id"`
	SubjectType     *string         `json:"subject_type"`
	GeneratedAt     *string         `json:"generated_at"`
	Source          *string         `json:"source"`
	Passed          bool            `json:"passed"`
	Temporal        TemporalSummary `json:"temporal"`
	HashProvided    *string         `json:"hash_provided"`
	HashMatched     bool            `json:"hash_matched"`
	SkippedExternal bool            `json:"skipped_external"`
	Steps           []Step          `json:"steps"`
	ItemID          *string         `json:"item_id"` // backward compat
}

// apiError represents an error from the JSON:API response.
type apiError struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// RunRemote sends the proof to the Truestamp API for server-side verification
// and returns a Report compatible with the local verification output.
func RunRemote(filename string, opts RemoteOptions) (*Report, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading proof file: %w", err)
	}

	// Single parse to extract type and preserve raw JSON for the API request
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parsing proof: %w", err)
	}

	// If the input is CBOR, convert to JSON for the API request
	jsonData := data
	if proof.IsCBORProof(data) {
		jsonData, err = bundle.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("converting CBOR proof to JSON: %w", err)
		}
	}

	// Route to the unified proof verification endpoint
	endpoint := "/proof/verify"

	// Build request body
	var raw json.RawMessage = jsonData
	dataFields := map[string]any{
		"proof": raw,
	}
	if opts.ExpectedHash != "" {
		dataFields["expected_hash"] = opts.ExpectedHash
	}
	requestBody := map[string]any{
		"data": dataFields,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encoding request body: %w", err)
	}

	// POST to the API
	url := opts.APIURL + endpoint
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	if opts.Team != "" {
		req.Header.Set("tenant", opts.Team)
	}

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading API response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("parsing API response: %w", err)
	}
	if envelope.Result == nil {
		return nil, fmt.Errorf("API response missing 'result' field")
	}

	report := mapToReport(filename, int64(len(data)), envelope.Result)

	// Extract display data from the already-parsed proof bundle
	populateFromBundle(report, bundle)

	return report, nil
}

// mapToReport converts the API result to a Report struct.
func mapToReport(filename string, fileSize int64, result *apiResult) *Report {
	r := &Report{
		Filename:        filename,
		FileSize:        fileSize,
		Temporal:        result.Temporal,
		Steps:           result.Steps,
		Remote:          true,
		SkippedExternal: result.SkippedExternal,
	}

	if result.ProofVersion != nil {
		r.ProofVersion = *result.ProofVersion
	}
	if result.SubjectID != nil {
		r.SubjectID = *result.SubjectID
	} else if result.ItemID != nil {
		r.SubjectID = *result.ItemID
	}
	if result.SubjectType != nil {
		r.SubjectType = *result.SubjectType
	} else {
		r.SubjectType = "item"
	}
	if result.GeneratedAt != nil {
		r.GeneratedAt = *result.GeneratedAt
	}
	if result.Source != nil {
		r.Source = *result.Source
	}
	if result.HashProvided != nil {
		r.HashProvided = *result.HashProvided
	}

	return r
}

// populateFromBundle extracts display data from a parsed proof bundle
// for presenter parity with local mode.
func populateFromBundle(r *Report, bundle *proof.ProofBundle) {
	subject := bundle.Subject
	t := bundle.T

	r.SubjectType = ptype.Name(t)

	if r.SubjectID == "" {
		// Block-like (plain block or beacon): SubjectID falls back to the
		// block id. Otherwise use the Subject.ID (item or entropy).
		if bundle.IsBlockLike() {
			r.SubjectID = bundle.Block.ID
		} else if subject != nil {
			r.SubjectID = subject.ID
		}
	}

	// Claims (item proofs)
	if bundle.IsItem() && subject != nil {
		r.Claims = parseClaims(bundle.RawData)

		// Derive TimestampStatus from steps (server already validated)
		for _, s := range r.Steps {
			if s.Status == StatusWarn {
				if strings.Contains(s.Message, "future-dated claim") {
					r.Claims.TimestampStatus = TimestampFuture
					r.Claims.TimestampNote = s.Message
				} else if strings.Contains(s.Message, "stale claim") {
					r.Claims.TimestampStatus = TimestampStale
					r.Claims.TimestampNote = s.Message
				}
			}
		}

		if r.Temporal.SubmittedAt == "" {
			r.Temporal.SubmittedAt = tscrypto.FormatItemTime(subject.ID)
		}

		if r.Temporal.ClaimedAt == "" {
			if ts := extractClaimsTimestamp(subject.Data); ts != "" {
				r.Temporal.ClaimedAt = truncateToSecond(ts)
			}
		}
	}

	if bundle.IsEntropy() && subject != nil {
		r.EntropySubject = parseEntropySubject(t, subject.Data)

		if r.Temporal.CapturedAt == "" {
			r.Temporal.CapturedAt = tscrypto.FormatBlockTime(subject.ID)
		}
	}

	r.ChainLength = 1
	r.SigningKeyID = bundle.Block.SigningKeyID

	if r.Temporal.CommittedAt == "" {
		r.Temporal.CommittedAt = tscrypto.FormatBlockTime(bundle.Block.ID)
	}

	for i := range bundle.Commitments {
		cx := &bundle.Commitments[i]

		switch cx.Type {
		case ptype.CommitmentStellar:
			ci := CommitmentInfo{
				Method:        "stellar",
				Network:       cx.Network,
				Ledger:        cx.Ledger,
				TxHash:        cx.TransactionHash,
				CommittedHash: cx.MemoHash,
				Timestamp:     cx.Timestamp,
			}
			r.CommitmentInfos = append(r.CommitmentInfos, ci)

		case ptype.CommitmentBitcoin:
			ci := CommitmentInfo{
				Method:        "bitcoin",
				Network:       cx.Network,
				Height:        cx.BlockHeight,
				TxHash:        cx.TransactionHash,
				CommittedHash: cx.OpReturn,
				Timestamp:     cx.Timestamp,
			}
			if cx.TxoutproofHex != "" {
				if blockHash, err := bitcoin.ParseBlockHash(cx.TxoutproofHex); err == nil {
					ci.BlockHash = blockHash
				}
			}
			r.CommitmentInfos = append(r.CommitmentInfos, ci)
		}
	}
}

// parseAPIError extracts a meaningful error message from the API response body.
func parseAPIError(statusCode int, body []byte) error {
	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		if first.Detail != "" {
			return fmt.Errorf("API error (HTTP %d): %s", statusCode, first.Detail)
		}
		if first.Title != "" {
			return fmt.Errorf("API error (HTTP %d): %s", statusCode, first.Title)
		}
	}

	bodyStr := string(body)
	if len(bodyStr) > 0 && bodyStr[0] == '<' {
		return fmt.Errorf("API error (HTTP %d): server returned HTML error page", statusCode)
	}

	return fmt.Errorf("API error (HTTP %d): %s", statusCode, httpclient.Truncate(bodyStr, 200))
}
