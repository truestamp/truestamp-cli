// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// Download fetches a proof bundle from a URL using [context.Background].
// Prefer [DownloadCtx] when a cancellable context is available.
func Download(rawURL string) ([]byte, error) {
	return DownloadCtx(context.Background(), rawURL)
}

// DownloadCtx is the context-aware variant of [Download]. Honours ctx for
// cancellation (e.g. Ctrl-C while a proof is streaming).
func DownloadCtx(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("URL must include a host")
	}

	data, err := httpclient.GetJSONCtx(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("downloading proof: %w", err)
	}

	// Quick-validate: the response should look like a bundle. `version`
	// is deliberately not part of the test: E.6 exempts it from structural
	// gating, and a wrong or absent one belongs in the report as a failing
	// step, not in a download error. The draft layout is refused by the
	// parser with `unsupported_layout`, so it is not pre-empted here.
	var shape struct {
		PublicKey string          `json:"public_key"`
		Signature string          `json:"signature"`
		Block     json.RawMessage `json:"block"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return nil, fmt.Errorf("response is not valid JSON: %w", err)
	}
	if shape.PublicKey == "" || shape.Signature == "" || len(shape.Block) == 0 {
		return nil, fmt.Errorf("response does not appear to be a Truestamp proof bundle (missing public_key, signature, or block)")
	}

	return data, nil
}

// WitnessSelection is the `witnesses` argument of proof generation: which
// witness details the bundle should carry. The zero value selects every
// witness (the complete bundle); [NoWitnesses] selects none (the compact
// bundle); [SelectWitnesses] selects a subset (a partial bundle). All three
// are ordinary version 1 bundles.
type WitnessSelection struct {
	explicit bool
	names    []string
}

// AllWitnesses selects every witness; the argument is omitted from the
// request.
func AllWitnesses() WitnessSelection { return WitnessSelection{} }

// NoWitnesses selects no witness details; the request carries `[]`.
func NoWitnesses() WitnessSelection { return WitnessSelection{explicit: true} }

// SelectWitnesses selects the named witnesses. Every name must be a
// registered witness name (the server rejects an unknown one with
// `invalid_witness`, and so does this).
func SelectWitnesses(names []string) (WitnessSelection, error) {
	var out []string
	seen := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if !IsWitnessName(n) {
			return WitnessSelection{}, fmt.Errorf("unknown witness %q; valid names: %s", n, strings.Join(WitnessNames, ", "))
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return NoWitnesses(), nil
	}
	return WitnessSelection{explicit: true, names: out}, nil
}

// ParseWitnessSelection parses a `--witnesses` flag value: "" or "all"
// selects every witness, "none" selects none, and a comma-separated list
// selects a subset.
func ParseWitnessSelection(flag string) (WitnessSelection, error) {
	switch strings.ToLower(strings.TrimSpace(flag)) {
	case "", "all":
		return AllWitnesses(), nil
	case "none":
		return NoWitnesses(), nil
	}
	return SelectWitnesses(strings.Split(flag, ","))
}

// IsAll reports whether every witness is selected.
func (w WitnessSelection) IsAll() bool { return !w.explicit }

// IsNone reports whether no witness detail is selected.
func (w WitnessSelection) IsNone() bool { return w.explicit && len(w.names) == 0 }

// Names returns the selected names; nil for every witness.
func (w WitnessSelection) Names() []string { return w.names }

// FilenameSuffix returns the artifact filename suffix Appendix E.2 gives
// each variant: "" for complete, "-compact" for none, "-partial" for a
// subset.
func (w WitnessSelection) FilenameSuffix() string {
	switch {
	case w.IsAll():
		return ""
	case w.IsNone():
		return "-compact"
	default:
		return "-partial"
	}
}

// String renders the selection the way the flag spells it.
func (w WitnessSelection) String() string {
	switch {
	case w.IsAll():
		return "all"
	case w.IsNone():
		return "none"
	default:
		return strings.Join(w.names, ",")
	}
}

// Generate calls [GenerateCtx] with [context.Background].
func Generate(apiURL, team, id, subjectType, format string, witnesses WitnessSelection) ([]byte, error) {
	return GenerateCtx(context.Background(), apiURL, team, id, subjectType, format, witnesses)
}

// GenerateCtx requests a proof bundle from the Truestamp API for the given
// subject ID. subjectType MUST be one of the six registry names (the server
// does no auto-detection):
//
//	item | entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon
//
// format is "json" or "cbor". witnesses selects which witness details the
// bundle carries. Returns raw bytes ready to write to a file (pretty JSON,
// with every number literal preserved, or decoded CBOR binary). ctx cancels
// the in-flight request. The credential is applied by the process-wide
// [auth.Authorizer].
func GenerateCtx(ctx context.Context, apiURL, team, id, subjectType, format string, witnesses WitnessSelection) ([]byte, error) {
	dataFields := map[string]any{"id": id}
	if format != "" && format != "json" {
		dataFields["format"] = format
	}
	if subjectType != "" {
		dataFields["type"] = subjectType
	}
	if !witnesses.IsAll() {
		names := witnesses.Names()
		if names == nil {
			names = []string{}
		}
		dataFields["witnesses"] = names
	}
	requestBody := map[string]any{"data": dataFields}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encoding request body: %w", err)
	}

	status, respBody, err := postJSONAPI(ctx, apiURL+"/proof/generate", team, bodyBytes)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, parseGenerateError(status, respBody)
	}

	// Parse the response envelope {"result": ...}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("parsing API response: %w", err)
	}
	if len(envelope.Result) == 0 {
		return nil, fmt.Errorf("API response missing 'result' field")
	}

	if format == "cbor" {
		// Result is a base64-encoded CBOR binary string (JSON string with quotes)
		var b64 string
		if err := json.Unmarshal(envelope.Result, &b64); err != nil {
			return nil, fmt.Errorf("parsing CBOR base64 from response: %w", err)
		}
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("decoding CBOR base64: %w", err)
		}
		return decoded, nil
	}

	// JSON format: re-indent the result without re-encoding it, so every
	// number literal survives as the server wrote it. A round trip through
	// `any` parses numbers into float64 and silently rounds anything past
	// 2^53, which changes the bytes a hash is computed over.
	if !isObject(envelope.Result) {
		return nil, fmt.Errorf("API response 'result' is not a proof object")
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, envelope.Result, "", "  "); err != nil {
		return nil, fmt.Errorf("formatting proof JSON: %w", err)
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

// Codes the server sends in `meta.code` on POST /proof/generate.
//
// These are deliberately NOT in the Appendix E.23 block in errors.go: that
// vocabulary is the hard-rejection taxonomy every independent verifier must
// agree on when refusing a bundle, and these describe why the server could
// not build one. Mixing them would put a server-side condition into a
// normative verifier registry.
const (
	// GenerateCodeSubjectNotRecomputable means the subject's stored data no
	// longer reproduces the hash committed at submission, so no verifiable
	// proof can be generated for it. Terminal: retrying cannot help, and
	// the caller should be told that rather than left to poll.
	GenerateCodeSubjectNotRecomputable = "subject_not_recomputable"
	// GenerateCodeGenerationFailed means the server generated a bundle and
	// its own self-verification rejected it. Carries `meta.failed_steps`.
	GenerateCodeGenerationFailed = "generation_failed"
	// GenerateCodeNoExternalCommitments means the subject has no
	// public-chain commitment yet. Transient: it clears on its own.
	GenerateCodeNoExternalCommitments = "no_external_commitments"
	// GenerateCodeSubjectNotReady means the subject is not yet in a state a
	// proof can be built from. Transient, and the second half of the
	// server's closed retryable set; every other generate code is terminal
	// by omission.
	GenerateCodeSubjectNotReady = "subject_not_ready"
)

// GenerateAPIError is a structured error from /proof/generate carrying the
// server's `meta.code` when one was sent (for example
// `no_external_commitments` for a subject not yet committed to a public
// chain, or `invalid_witness` for an unknown witness name).
type GenerateAPIError struct {
	StatusCode int
	Code       string
	Detail     string
	// FailedSteps carries `meta.failed_steps`: the step messages from the
	// server's own self-verification of the bundle it just generated.
	// Without it a `generation_failed` says only that some check failed,
	// which is a day of bisection for whoever holds the subject.
	FailedSteps string
	// Drifted carries `meta.drifted`: which half of the composite subject
	// fingerprint no longer reproduces from stored data. One of "claims",
	// "metadata", or "claims and metadata".
	Drifted string
}

func (e *GenerateAPIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("API error (HTTP %d, %s): %s", e.StatusCode, e.Code, e.Detail)
	}
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Detail)
}

// parseGenerateError extracts a meaningful error message from the API response.
func parseGenerateError(statusCode int, body []byte) error {
	var envelope struct {
		Errors []struct {
			Status string `json:"status"`
			Detail string `json:"detail"`
			Title  string `json:"title"`
			Meta   struct {
				Code        string `json:"code"`
				FailedSteps string `json:"failed_steps"`
				Drifted     string `json:"drifted"`
			} `json:"meta"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		detail := first.Detail
		if detail == "" {
			detail = first.Title
		}
		if detail != "" {
			return &GenerateAPIError{
				StatusCode:  statusCode,
				Code:        first.Meta.Code,
				Detail:      detail,
				FailedSteps: first.Meta.FailedSteps,
				Drifted:     first.Meta.Drifted,
			}
		}
	}

	bodyStr := string(body)
	if len(bodyStr) > 0 && bodyStr[0] == '<' {
		return fmt.Errorf("API error (HTTP %d): server returned HTML error page", statusCode)
	}
	return fmt.Errorf("API error (HTTP %d): %s", statusCode, httpclient.Truncate(bodyStr, 200))
}

// postJSONAPI issues an authenticated JSON:API POST carrying the tenant
// header and returns the status and the size-capped body. GenerateCtx and
// ResolveSubjectType share it so the request shape is declared once.
func postJSONAPI(ctx context.Context, url, team string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")
	if err := auth.AuthorizeRequest(ctx, req); err != nil {
		return 0, nil, err
	}
	if team != "" {
		req.Header.Set("tenant", team)
	}

	resp, err := httpclient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return 0, nil, fmt.Errorf("reading API response: %w", err)
	}
	return resp.StatusCode, respBody, nil
}
