// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
	"github.com/truestamp/truestamp-cli/internal/proof"
)

// RemoteOptions holds configuration for server-side verification. The
// credential is applied by the process-wide [auth.Authorizer]; only the
// tenant scoping is carried here.
//
// The service's verifier is no part of this CLI's own verification, which
// never depends on it; --remote is an explicit "ask Truestamp too".
type RemoteOptions struct {
	APIURL       string
	Team         string // team ID, sent as the tenant header
	ExpectedHash string
	SkipExternal bool
}

// apiEnvelope wraps the top-level API response.
type apiEnvelope struct {
	Result *apiResult `json:"result"`
	Errors []apiError `json:"errors"`
}

// apiResult is the part of the server's verification result this CLI
// reads, in the server's field names: its verdict, its steps, and the
// expected hash it echoes with whether it matched.
type apiResult struct {
	Passed       bool    `json:"passed"`
	Steps        []Step  `json:"steps"`
	HashProvided *string `json:"hash_provided"`
	HashMatched  bool    `json:"hash_matched"`
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

// RunRemoteBytesCtx sends the proof in data to the Truestamp API for
// server-side verification and returns the service's verdict, with this
// CLI's own expected-hash comparison beside it.
func RunRemoteBytesCtx(ctx context.Context, data []byte, opts RemoteOptions) (*Report, error) {
	// One parse by the earlier bundle parser, so a bundle it refuses is
	// never posted.
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		return nil, err
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

	// jsonapi.Send carries the one retry of a rate-limited request; the
	// request itself is built here rather than by jsonapi.DoRaw so the
	// credential handling of this path stays exactly as it was.
	resp, respBody, err := jsonapi.Send(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, jsonapi.ErrorFromResponse(resp, respBody)
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

	report := &Report{
		Steps:        envelope.Result.Steps,
		ExpectedHash: strings.ToLower(strings.TrimSpace(opts.ExpectedHash)),
	}

	// The expected-hash comparison is made by this process, on the bundle
	// it just parsed.
	applyExpectedHash(report, bundle, envelope.Result)

	// Reconcile the server's own top-level verdict with the step list it
	// sent alongside it.
	applyServerVerdict(report, envelope.Result)
	return report, nil
}

// applyExpectedHash makes the expected-hash comparison locally, on the
// caller's expected hash and the subject.claims.hash of the bundle this
// process parsed and posted. The
// answer to "is my local file the timestamped one" is computed here rather
// than taken on trust, and a server row that disagrees with it fails the
// run rather than being averaged away.
func applyExpectedHash(r *Report, bundle *proof.Bundle, result *apiResult) {
	if r.ExpectedHash == "" {
		return
	}
	local := Report{ExpectedHash: r.ExpectedHash}
	stepExpectedHash(&local, bundle)
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
// the step list it sent. [Report.Passed] reads step statuses, so the
// server's own verdict is given a step of its own rather than bypassing
// it: a response reporting passed:false with no failing step, or no steps
// at all, must not read as passed.
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
// (HTTP 400 with meta.code invalid_proof), carrying the service's reason
// in Reason.
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
