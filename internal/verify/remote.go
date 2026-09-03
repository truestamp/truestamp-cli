// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/proof"
)

// RemoteOptions holds configuration for server-side verification. The
// credential is applied by the process-wide [auth.Authorizer]; only the
// tenant scoping is carried here.
//
// The server's verifier is NOT part of the independence argument (Appendix
// E.2) and this CLI's own verifier never depends on it; --remote is an
// explicit "ask Truestamp too".
type RemoteOptions struct {
	APIURL       string
	Team         string // team ID, sent as the tenant header
	ExpectedHash string
	SkipExternal bool

	// ExpectedSubjectType is asserted locally against the bundle's signed
	// `type` before anything is posted, as the hard rejection
	// `subject_type_mismatch`, and forwarded to the server when it holds.
	ExpectedSubjectType string
}

// apiEnvelope wraps the top-level API response.
type apiEnvelope struct {
	Result *apiResult `json:"result"`
	Errors []apiError `json:"errors"`
}

// apiResult is the verification result from the server, in its own field
// names.
type apiResult struct {
	ID                   *string  `json:"id"`
	Source               *string  `json:"source"`
	Passed               bool     `json:"passed"`
	Steps                []Step   `json:"steps"`
	Temporal             Temporal `json:"temporal"`
	HashProvided         *string  `json:"hash_provided"`
	ExpectedHashProvided bool     `json:"expected_hash_provided"`
	HashMatched          bool     `json:"hash_matched"`
	ProofVersion         *int     `json:"proof_version"`
	SkippedExternal      bool     `json:"skipped_external"`
	GeneratedAt          *string  `json:"generated_at"`
}

// apiError represents an error from the JSON:API response.
type apiError struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Meta   struct {
		Code   string `json:"code"`
		Reason string `json:"reason"`
	} `json:"meta"`
}

// RunRemote calls [RunRemoteCtx] with [context.Background].
func RunRemote(filename string, opts RemoteOptions) (*Report, error) {
	return RunRemoteCtx(context.Background(), filename, opts)
}

// RunRemoteCtx sends the proof to the Truestamp API for server-side
// verification and returns a Report compatible with the local output.
func RunRemoteCtx(ctx context.Context, filename string, opts RemoteOptions) (*Report, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading proof file: %w", err)
	}
	return RunRemoteBytesCtx(ctx, data, filename, opts)
}

// RunRemoteBytesCtx is [RunRemoteCtx] over bytes already in hand.
func RunRemoteBytesCtx(ctx context.Context, data []byte, displayName string, opts RemoteOptions) (*Report, error) {
	// One parse, applying the same E.6 gates the local path applies, so a
	// bundle this CLI would reject is never posted.
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		return nil, err
	}
	if opts.ExpectedSubjectType != "" && opts.ExpectedSubjectType != bundle.Type {
		return nil, proof.Rejectf(proof.CodeSubjectTypeMismatch,
			"the bundle's signed type is %s but --type %s was asserted", bundle.Type, opts.ExpectedSubjectType)
	}

	// The server takes JSON; a CBOR input is posted as its JSON conversion.
	jsonData, err := bundle.MarshalJSON()
	if err != nil {
		return nil, err
	}

	dataFields := map[string]any{
		"proof":         json.RawMessage(jsonData),
		"skip_external": opts.SkipExternal,
	}
	if opts.ExpectedHash != "" {
		dataFields["expected_hash"] = opts.ExpectedHash
	}
	if opts.ExpectedSubjectType != "" {
		dataFields["type"] = opts.ExpectedSubjectType
	}
	bodyBytes, err := json.Marshal(map[string]any{"data": dataFields})
	if err != nil {
		return nil, fmt.Errorf("encoding request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.APIURL+"/proof/verify", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")
	if err := auth.AuthorizeRequest(ctx, req); err != nil {
		return nil, err
	}
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

	localOpts := Options{ExpectedHash: opts.ExpectedHash, SkipExternal: opts.SkipExternal}
	report := newReport(bundle, displayName, int64(len(data)), localOpts)
	report.Remote = true
	report.Steps = envelope.Result.Steps
	report.Temporal = envelope.Result.Temporal
	report.SkippedExternal = envelope.Result.SkippedExternal
	if envelope.Result.ProofVersion != nil {
		report.ProofVersion = *envelope.Result.ProofVersion
	}

	// E.7 is enforced by this process, on the bundle it just parsed.
	applyExpectedHash(report, bundle, envelope.Result)

	// Reconcile the server's own top-level verdict with the step list it
	// sent alongside it.
	applyServerVerdict(report, envelope.Result)
	return report, nil
}

// applyExpectedHash performs Appendix E.7's comparison locally, on the same
// two operands the local pipeline uses: the caller's expected hash and the
// subject.claims.hash of the bundle this process parsed and posted. The
// answer to "is my local file the timestamped one" is computed here rather
// than taken on trust, and a server row that disagrees with it fails the
// run rather than being averaged away.
func applyExpectedHash(r *Report, bundle *proof.Bundle, result *apiResult) {
	if r.ExpectedHash == "" {
		return
	}
	var local Report
	local.ExpectedHash = r.ExpectedHash
	stepExpectedHash(&local, bundle, Options{ExpectedHash: r.ExpectedHash})
	if len(local.Steps) == 0 {
		return
	}
	mine := local.Steps[0]

	agrees, serverClaimsMatch := false, false
	for _, s := range r.Steps {
		if s.Group != groupHashComparison {
			continue
		}
		agrees = agrees || s.Status == mine.Status
		serverClaimsMatch = serverClaimsMatch || s.Status == StatusPass
	}
	if !agrees {
		r.Steps = append(r.Steps, mine)
	}

	echoed := result.HashProvided != nil && *result.HashProvided != ""
	serverClaimsMatch = serverClaimsMatch || (echoed && result.HashMatched)
	switch {
	case serverClaimsMatch && mine.Status != StatusPass:
		r.fail(groupHashComparison, CatDataIntegrity, fmt.Sprintf(
			"Expected-hash disagreement: the server reports the supplied hash as matching, this verifier's own comparison against the posted bundle reports %s", mine.Status))
	case echoed && mine.Status == StatusPass && !result.HashMatched:
		r.fail(groupHashComparison, CatDataIntegrity,
			"Expected-hash disagreement: the server reports the supplied hash as NOT matching, this verifier's own comparison against the posted bundle reports pass")
	}
}

// applyServerVerdict reconciles the server's top-level `passed` field with
// the step list it sent. E.22's verdict rule reads step statuses, so the
// server's own verdict is given a step of its own rather than bypassing the
// rule: a response reporting passed:false with no failing step, or no steps
// at all, must not render as verified.
func applyServerVerdict(r *Report, result *apiResult) {
	if len(result.Steps) == 0 {
		r.fail(groupServerVerdict, CatStructural,
			"Server returned no verification steps: this run establishes nothing about the proof")
		return
	}
	serverFailed := false
	for _, s := range result.Steps {
		if s.Status == StatusFail {
			serverFailed = true
			break
		}
	}
	if !result.Passed && !serverFailed {
		r.fail(groupServerVerdict, CatStructural,
			"Server reported the proof as NOT verified (passed: false) but sent no failing step: the reported outcome and the reported steps disagree")
	}
}

// RemoteRejectionError is a structured rejection returned by /proof/verify
// (HTTP 400 with meta.code invalid_proof), carrying the Appendix E.23
// identifier in Reason.
type RemoteRejectionError struct {
	StatusCode int
	Reason     string
	Detail     string
}

func (e *RemoteRejectionError) Error() string {
	return fmt.Sprintf("the server rejected the proof (HTTP %d): %s: %s", e.StatusCode, e.Reason, e.Detail)
}

// parseAPIError extracts a meaningful error from the API response body.
func parseAPIError(statusCode int, body []byte) error {
	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		if first.Meta.Code == "invalid_proof" && first.Meta.Reason != "" {
			return &RemoteRejectionError{StatusCode: statusCode, Reason: first.Meta.Reason, Detail: first.Detail}
		}
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
