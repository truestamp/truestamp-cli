// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// RemoteOptions configures one request for the Truestamp service's
// verdict. The credential is applied by the process-wide auth.Authorizer;
// only the tenant scoping is carried here.
//
// The service's verifier is no part of this CLI's own verification, which
// never depends on it; --remote is an explicit "ask Truestamp too".
type RemoteOptions struct {
	APIURL string
	// Team is the team ID, sent as the tenant header.
	Team string
	// Offline asks for a run that reads nothing, `offline=true`.
	Offline bool
	// ExpectedHash, when not nil, is sent exactly as given as
	// `expected_hash`, the caller's hash for EXPECTED_HASH.
	ExpectedHash *string
	// HTTPTimeout is the configured http_timeout. The request waits for
	// the larger of it and remoteFloor.
	HTTPTimeout time.Duration
}

// remoteFloor is the least time the request waits for the service's
// answer: the 60 seconds the service gives an online run, and a margin
// for the transfer. A variable so this package's tests can shorten it.
var remoteFloor = 75 * time.Second

// proofMediaType is the media type SPEC.md section 3.1 (The envelope)
// gives a served proof.
const proofMediaType = "application/vnd.truestamp.proof+json"

// remoteLabels are the labels of SPEC.md section 10.5 (Verdict and report),
// the only verdicts a report of the service can give.
var remoteLabels = []string{
	verifier.LabelVerified, verifier.LabelInternallyConsistent, verifier.LabelSuppliedKeyring,
	verifier.LabelUnattributed, verifier.LabelFailed,
}

// AskRemoteCtx asks the Truestamp service to verify the proof file whose
// bytes are data, and returns the label of the report it answers with. It
// posts data exactly as given, never parsed or re-encoded, as the body of
//
//	POST {APIURL}/proof/raw/verify[?offline=true][&expected_hash=...]
//
// whose 200 is the service's report of SPEC.md section 10.5 (Verdict and
// report). Only a 200 whose report carries one of that section's labels is
// a verdict. Any other answer is an error: a 429 the [*jsonapi.APIError]
// of the rate limit, a refusal naming a `meta.code` a
// [*RemoteRejectionError], and every other status an error naming the
// status and the body. The request, its one retry of a rate limit
// included, waits for the larger of opts.HTTPTimeout and 75 seconds, and
// no answer within that is an error.
func AskRemoteCtx(ctx context.Context, data []byte, opts RemoteOptions) (string, error) {
	ctx, cancel := httpclient.WithBound(ctx, max(opts.HTTPTimeout, remoteFloor))
	defer cancel()
	query := url.Values{}
	if opts.Offline {
		query.Set("offline", "true")
	}
	if opts.ExpectedHash != nil {
		query.Set("expected_hash", *opts.ExpectedHash)
	}
	target := opts.APIURL + "/proof/raw/verify"
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", proofMediaType)
	req.Header.Set("Accept", "application/json, application/vnd.api+json")
	if err := auth.AuthorizeRequest(ctx, req); err != nil {
		return "", err
	}
	if opts.Team != "" {
		req.Header.Set("tenant", opts.Team)
	}

	// jsonapi.Send carries the one retry of a rate-limited request.
	resp, body, err := jsonapi.Send(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", jsonapi.ErrorFromResponse(resp, body)
	}
	if resp.StatusCode != http.StatusOK {
		return "", parseAPIError(resp.StatusCode, body)
	}
	var report struct {
		Label *string `json:"label"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return "", fmt.Errorf("the service's answer is not a report: %w", err)
	}
	if report.Label == nil {
		return "", fmt.Errorf("the service's report carries no label")
	}
	if !slices.Contains(remoteLabels, *report.Label) {
		return "", fmt.Errorf("the service's report carries the label %q, which SPEC.md section 10.5 does not define", truncateToken(*report.Label))
	}
	return *report.Label, nil
}

// RemoteRejectionError is the service's refusal of a request: an error
// whose `meta.code` names why, such as a proof file it refuses before any
// check. Code is that code and Detail the error's detail.
type RemoteRejectionError struct {
	StatusCode int
	Code       string
	Detail     string
}

func (e *RemoteRejectionError) Error() string {
	return fmt.Sprintf("the service refused the request (HTTP %d): %s", e.StatusCode, e.Detail)
}

// parseAPIError extracts a meaningful error from the API response body.
func parseAPIError(statusCode int, body []byte) error {
	var envelope struct {
		Errors []struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
			Meta   struct {
				Code string `json:"code"`
			} `json:"meta"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		detail := first.Detail
		if detail == "" {
			detail = first.Title
		}
		if first.Meta.Code != "" && detail != "" {
			return &RemoteRejectionError{StatusCode: statusCode, Code: first.Meta.Code, Detail: httpclient.Truncate(detail, 200)}
		}
		if detail != "" {
			return fmt.Errorf("API error (HTTP %d): %s", statusCode, httpclient.Truncate(detail, 200))
		}
	}
	bodyStr := string(body)
	if len(bodyStr) > 0 && bodyStr[0] == '<' {
		return fmt.Errorf("API error (HTTP %d): server returned HTML error page", statusCode)
	}
	return fmt.Errorf("API error (HTTP %d): %s", statusCode, httpclient.Truncate(bodyStr, 200))
}

// truncateToken bounds an echoed wire token so a hostile or broken server
// cannot inject an unbounded string into a message.
func truncateToken(s string) string {
	const max = 40
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
