// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package jsonapi is the one transport every Truestamp JSON:API client
// shares: the authenticated request carrying the tenant header, the
// response size cap, the one retry of a rate-limited request, and the
// classification of the error envelope into a small set of sentinels a
// command can errors.Is. The resource packages (internal/beacons,
// internal/blocks, internal/teams, internal/items, internal/proof) own
// only their routes and their decoding; each used to carry its own copy
// of this file, and the copies had started to differ in which statuses
// mapped to which class.
package jsonapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/redact"
)

// Config carries what a request needs beyond the credential, which the
// process-wide auth.Authorizer installed in cmd/root supplies out of band.
type Config struct {
	APIURL string // e.g. https://www.truestamp.com/api/json
	Team   string // optional tenant id; sent verbatim as the `tenant` header
}

// Class sentinels. An *APIError wraps exactly one of them, so a caller can
// errors.Is the class while still showing the server's detail text.
var (
	ErrUnauthorized = errors.New("not authenticated")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrBadRequest   = errors.New("bad request")
	ErrRateLimited  = errors.New("rate limited")
	ErrServer       = errors.New("server error")
)

// APIError carries the HTTP status and the preserved `errors[].detail`
// (falling back to `title`) from the JSON:API error envelope.
type APIError struct {
	Status  int
	Code    string // errors[].code, the server's machine-readable reason, when present
	Pointer string // errors[].source.pointer, when present
	Detail  string
	// RetryAfter is the verbatim Retry-After header on a 429. The
	// per-surface request-rate plug sends one (whole seconds, at least
	// 1); a refusal raised inside an action does not, and names its wait
	// in RetryAfterMS instead.
	RetryAfter string
	// RetryAfterMS is `meta.retry_after_ms` on a 429: the wait in
	// milliseconds before the request may be repeated. Zero when the
	// refusal did not carry one (see NeverAdmitted for the null case).
	RetryAfterMS int64
	// NeverAdmitted is set when `meta.retry_after_ms` was null: the
	// request costs more than the limit allows on its own, and no wait
	// admits it. Do not retry.
	NeverAdmitted bool
	// Limit is `meta.limit` on a 429, the limit that refused the request,
	// when carried. Its window is server configuration and is not carried,
	// so it is shown as sent and never assumed to be "per minute".
	Limit int64
	// Sentinel is the class this error belongs to; Unwrap returns it. A
	// resource package may narrow it to one of its own domain sentinels
	// once it has read the structural discriminators, as teams does for
	// the plan-limit and entitlement rejections on create.
	Sentinel error
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("HTTP %d: %s", e.Status, e.Detail)
	}
	return fmt.Sprintf("HTTP %d", e.Status)
}

func (e *APIError) Unwrap() error { return e.Sentinel }

// RetryAfterDelay is the wait the refusal named: the Retry-After header
// when it carried one, else `meta.retry_after_ms`. ok is false when it
// named none, or the error is not a 429.
func (e *APIError) RetryAfterDelay() (time.Duration, bool) {
	if e.Status != http.StatusTooManyRequests {
		return 0, false
	}
	if d, ok := httpclient.ParseRetryAfter(e.RetryAfter, time.Now()); ok {
		return d, true
	}
	if e.RetryAfterMS > 0 {
		return time.Duration(e.RetryAfterMS) * time.Millisecond, true
	}
	return 0, false
}

// RetryDelay reports how long to wait before repeating a rate-limited
// request, and whether repeating it can help at all. The wait is
// RetryAfterDelay when the refusal named one, else
// httpclient.DefaultRetryAfter. It is false when the error is not a 429,
// when `meta.retry_after_ms` was null (no wait admits the request), and
// when the wait is over httpclient.MaxRetryAfter; in the last case the
// delay is still returned so it can be shown.
func (e *APIError) RetryDelay() (time.Duration, bool) {
	if e.Status != http.StatusTooManyRequests || e.NeverAdmitted {
		return 0, false
	}
	return httpclient.BoundRetry(e.RetryAfterDelay())
}

// CodeInvalidKeyset is the server's code for a page[after] / page[before]
// value that is not a cursor it issued.
const CodeInvalidKeyset = "invalid_keyset"

// CodeRateLimited is the server's `errors[].code` on every rate-limit
// refusal, from the per-surface request-rate plug and from a limit raised
// inside an action alike. The status is always 429, and the class is
// decided on the status: a 429 that reaches the CLI without the code (an
// intermediary's, say) is still a rate limit. The code is what a caller
// that inspects the envelope itself should match.
const CodeRateLimited = "rate_limited"

// NotFound is the error a client returns when a filter-style lookup came
// back empty and there was no 404 to classify.
func NotFound(detail string) *APIError {
	return &APIError{Status: http.StatusNotFound, Detail: detail, Sentinel: ErrNotFound}
}

// Get issues an authenticated GET and returns the body on 2xx.
func Get(ctx context.Context, cfg Config, path string) ([]byte, error) {
	return Do(ctx, cfg, http.MethodGet, path, nil)
}

// Post marshals payload as the request body and POSTs it.
func Post(ctx context.Context, cfg Config, path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}
	return Do(ctx, cfg, http.MethodPost, path, body)
}

// Do issues an authenticated request and returns the body on 2xx. Any
// other status is an *APIError wrapping its class sentinel, with the
// Retry-After header preserved on 429.
func Do(ctx context.Context, cfg Config, method, path string, body []byte) ([]byte, error) {
	resp, respBody, err := DoRaw(ctx, cfg, method, path, body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}
	return nil, ErrorFromResponse(resp, respBody)
}

// DoRaw is Do without the status classification: the response (its body
// already read, capped and closed) comes back for a client with its own
// error envelope to parse, which proof generation's `meta.code` needs. A
// missing credential is still an *APIError, because no request is sent.
func DoRaw(ctx context.Context, cfg Config, method, path string, body []byte) (*http.Response, []byte, error) {
	if auth.Default().Mode() == auth.ModeNone {
		return nil, nil, &APIError{Status: http.StatusUnauthorized, Detail: "not authenticated", Sentinel: ErrUnauthorized}
	}
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, cfg.APIURL+path, rdr)
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/vnd.api+json")
	}
	if err := auth.AuthorizeRequest(ctx, req); err != nil {
		return nil, nil, authError(err)
	}
	if cfg.Team != "" {
		req.Header.Set("tenant", cfg.Team)
	}

	resp, respBody, err := Send(req)
	if err != nil {
		// The 401 retry transport refreshes the token mid-request; when
		// the authorization server rate limits that refresh, the failure
		// surfaces here as a transport error and is a 429, not a 401.
		if e := tokenRateLimited(err); e != nil {
			return nil, nil, e
		}
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}
	return resp, respBody, nil
}

// authError classifies a failure to attach the credential. A dead session
// is a 401 (re-login); the authorization server rate limiting the refresh
// is a 429 (wait), and must not be reported as the former, which would
// send the holder to re-login for nothing.
func authError(err error) *APIError {
	if e := tokenRateLimited(err); e != nil {
		return e
	}
	return &APIError{Status: http.StatusUnauthorized, Detail: err.Error(), Sentinel: ErrUnauthorized}
}

// tokenRateLimited returns the 429 an *auth.TokenRateLimitedError stands
// for, or nil when err is something else.
func tokenRateLimited(err error) *APIError {
	var rl *auth.TokenRateLimitedError
	if !errors.As(err, &rl) {
		return nil
	}
	e := &APIError{Status: http.StatusTooManyRequests, Detail: rl.Error(), Sentinel: ErrRateLimited}
	if rl.Wait > 0 {
		e.RetryAfter = strconv.FormatInt(int64(math.Ceil(rl.Wait.Seconds())), 10)
	}
	return e
}

// Send issues req through the shared client, reads its body (capped at
// httpclient.MaxResponseSize) and closes it. A 429 is repeated once, after
// the wait the refusal names: the Retry-After header, else
// `meta.retry_after_ms`, else httpclient.DefaultRetryAfter; never when
// `meta.retry_after_ms` is null, and not when the wait is over
// httpclient.MaxRetryAfter. The wait is jittered (httpclient.Jitter): the
// server's per-address guard is a window aligned to the clock minute, and
// a client that retried on the exact second it was told would arrive at
// the boundary with every other refused client. Whatever the second
// attempt answers is returned as is. Callers that build their own *http.Request (`auth
// status`'s probes, `verify --remote`) go through Send too, so the policy
// is one. A transport failure is returned unwrapped.
func Send(req *http.Request) (*http.Response, []byte, error) {
	resp, body, err := roundTrip(req)
	if err != nil || resp.StatusCode != http.StatusTooManyRequests {
		return resp, body, err
	}
	refusal := ErrorFromResponse(resp, body)
	delay, ok := refusal.RetryDelay()
	if !ok {
		return resp, body, nil
	}
	retry, ok := httpclient.Rewind(req)
	if !ok {
		return resp, body, nil
	}
	delay = httpclient.Jitter(delay)
	if fn := rateLimitNotifier(); fn != nil {
		fn(delay, refusal.Limit)
	}
	if werr := wait(req.Context(), delay); werr != nil {
		return nil, nil, werr
	}
	return roundTrip(retry)
}

// roundTrip is one attempt: the shared client, the capped read, the close.
func roundTrip(req *http.Request) (*http.Response, []byte, error) {
	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return nil, nil, fmt.Errorf("reading API response: %w", err)
	}
	return resp, body, nil
}

// wait is the sleeper Send uses between the two attempts, a variable so
// this package's tests can observe the wait without spending it.
var wait = httpclient.Wait

var (
	notifierMu sync.RWMutex
	notifier   func(wait time.Duration, limit int64)
)

// SetRateLimitNotifier installs the function Send calls before the one
// retry of a rate-limited request, with the wait about to be observed and
// the limit the refusal named (0 when it named none), so a command can
// tell the holder why it has gone quiet. Nil, the default, tells no one.
func SetRateLimitNotifier(fn func(wait time.Duration, limit int64)) {
	notifierMu.Lock()
	defer notifierMu.Unlock()
	notifier = fn
}

func rateLimitNotifier() func(time.Duration, int64) {
	notifierMu.RLock()
	defer notifierMu.RUnlock()
	return notifier
}

// ErrorFromResponse classifies a non-2xx response whose body has been
// read: ParseError, plus the Retry-After header a 429 may carry.
func ErrorFromResponse(resp *http.Response, body []byte) *APIError {
	e := ParseError(resp.StatusCode, body)
	if resp.StatusCode == http.StatusTooManyRequests {
		e.RetryAfter = resp.Header.Get("Retry-After")
	}
	return e
}

// ParseError classifies a non-2xx response. It keeps `errors[].detail`
// (or `title`) and the `source.pointer` of the first error that carries
// one: the server can return several errors at once (a free-plan user
// requesting team_retains trips both the plan-limit and the entitlement
// rejection), and the pointer is the structural discriminator, so it wins
// over array position. On a 429 it also reads the rate-limit meta:
// `retry_after_ms` (a number, or null for "never admitted") and `limit`.
func ParseError(status int, body []byte) *APIError {
	e := &APIError{Status: status, Sentinel: sentinelFor(status)}
	var envelope struct {
		Errors []struct {
			Code   string `json:"code"`
			Detail string `json:"detail"`
			Title  string `json:"title"`
			Source struct {
				Pointer string `json:"pointer"`
			} `json:"source"`
			Meta struct {
				RetryAfterMS json.RawMessage `json:"retry_after_ms"`
				Limit        json.RawMessage `json:"limit"`
			} `json:"meta"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		chosen := envelope.Errors[0]
		for i := range envelope.Errors {
			if envelope.Errors[i].Source.Pointer != "" {
				chosen = envelope.Errors[i]
				break
			}
		}
		e.Pointer = chosen.Source.Pointer
		e.Code = chosen.Code
		switch {
		case chosen.Detail != "":
			e.Detail = chosen.Detail
		case chosen.Title != "":
			e.Detail = chosen.Title
		}
		if status == http.StatusTooManyRequests {
			e.RetryAfterMS, e.NeverAdmitted = metaInt(chosen.Meta.RetryAfterMS)
			e.Limit, _ = metaInt(chosen.Meta.Limit)
		}
	}
	if e.Detail == "" {
		// Defense in depth: a server- or attacker-controlled raw body
		// (including a reflected request) is truncated AND run through the
		// secret redactor before it can reach a log or the terminal.
		trimmed := bytes.TrimSpace(body)
		if len(trimmed) > 0 && trimmed[0] == '<' {
			e.Detail = "server returned HTML error page"
		} else {
			e.Detail = redact.String(httpclient.Truncate(string(body), 200))
		}
	}
	return e
}

// metaInt reads a non-negative integer meta field. null is reported apart
// from absent because the contract gives null a meaning of its own on
// `retry_after_ms`: the request can never be admitted. Anything that is
// not a non-negative number reads as absent.
func metaInt(raw json.RawMessage) (n int64, null bool) {
	s := strings.TrimSpace(string(raw))
	switch {
	case s == "":
		return 0, false
	case s == "null":
		return 0, true
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil && v >= 0 {
		return v, false
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f >= 0 && f <= math.MaxInt64 {
		return int64(f), false
	}
	return 0, false
}

func sentinelFor(status int) error {
	switch {
	case status == http.StatusUnauthorized:
		return ErrUnauthorized
	case status == http.StatusForbidden:
		return ErrForbidden
	case status == http.StatusNotFound:
		return ErrNotFound
	case status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status >= 400 && status < 500:
		return ErrBadRequest
	case status >= 500:
		return ErrServer
	}
	return errors.New("unexpected status")
}
