// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package httpclient provides a shared HTTP client for all external API calls.
// The client is safe for concurrent use and reuses connections.
// Call Init once during startup to set the timeout; the default is 10s.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"
)

// httpClient is the shared HTTP client.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// userAgent is stamped onto every outbound request by SetUserAgent and
// applied centrally in Do. Empty means "don't override any UA the caller
// already set". Populated once at startup via SetUserAgent from cmd/root.
var userAgent string

// MaxResponseSize limits HTTP response bodies to 1 MB to prevent OOM.
const MaxResponseSize = 1 << 20

// Init creates a new HTTP client with the given timeout.
// Must be called once during startup before any external calls.
func Init(timeout time.Duration) {
	httpClient = &http.Client{Timeout: timeout}
}

// SetTransport installs rt as the round-tripper for the shared client. The
// CLI uses it to layer the auth package's reactive 401 → refresh →
// retry-once behavior on top of the default transport. Call after [Init],
// which replaces the client.
func SetTransport(rt http.RoundTripper) {
	if httpClient != nil {
		httpClient.Transport = rt
	}
}

// SetUserAgent configures the User-Agent header stamped onto every
// outbound request through [Do]. Typical value:
// "truestamp-cli/<version> (<os>/<arch>)". Pass an empty string to
// disable the override (requests keep whatever UA the caller set, or
// Go's default).
func SetUserAgent(version string) {
	if version == "" {
		userAgent = ""
		return
	}
	userAgent = fmt.Sprintf("truestamp-cli/%s (%s/%s)", version, runtime.GOOS, runtime.GOARCH)
}

// Do executes an HTTP request using the shared client. The request's
// existing [context.Context] (if any) is respected; callers that want
// cancellation should attach one via [http.Request.WithContext] before
// calling. If SetUserAgent has been called and the request has no
// User-Agent header, the configured value is applied.
func Do(req *http.Request) (*http.Response, error) {
	stampUserAgent(req)
	return httpClient.Do(req)
}

// stampUserAgent applies the shared User-Agent header when one is
// configured and the caller did not already set one.
func stampUserAgent(req *http.Request) {
	if userAgent == "" {
		return
	}
	if req.Header.Get("User-Agent") != "" {
		return
	}
	req.Header.Set("User-Agent", userAgent)
}

// GetJSON performs a GET request with [context.Background] and returns the
// response body. Prefer [GetJSONCtx] when a cancellable context is
// available (e.g. from Cobra's cmd.Context()).
func GetJSON(url string) ([]byte, error) {
	return GetJSONCtx(context.Background(), url)
}

// GetJSONCtx performs a context-aware GET request and returns the response
// body. Errors are typed: *TransportError before a response exists,
// *StatusError for a non-2xx status, *TruncatedError for a body over
// MaxResponseSize. Their Error() strings are unchanged from the untyped
// versions they replaced.
func GetJSONCtx(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &TransportError{URL: url, Err: err}
	}
	stampUserAgent(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, &TransportError{URL: url, Err: err}
	}
	defer resp.Body.Close()

	// Read one byte past the cap so an oversize body can be reported as
	// such instead of reaching the caller's decoder as truncated JSON.
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize+1))
	if err != nil {
		return nil, &TransportError{URL: url, Err: err}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyStr := Truncate(string(body), 80)
		return nil, &StatusError{
			StatusCode: resp.StatusCode,
			URL:        url,
			Body:       bodyStr,
			HTML:       len(bodyStr) > 0 && bodyStr[0] == '<',
		}
	}

	if len(body) > MaxResponseSize {
		return nil, &TruncatedError{URL: url, Limit: MaxResponseSize}
	}

	return body, nil
}

// Truncate shortens a string to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TransportError wraps a failure that happened before any HTTP response
// was received: request construction, DNS resolution, dial, TLS
// handshake, client timeout, context cancellation, or a body that could
// not be read. A verifier must report these as `skip` rather than
// `fail` (whitepaper Appendix E.17/E.18/E.21/E.22), so the distinction
// has to survive to the call site instead of collapsing into one opaque
// error.
type TransportError struct {
	URL string
	Err error
}

func (e *TransportError) Error() string { return e.Err.Error() }
func (e *TransportError) Unwrap() error { return e.Err }

// StatusError is returned for any non-2xx response. It preserves the
// status code so a caller can tell a definitive answer (404) from an
// availability problem (429, 5xx).
type StatusError struct {
	StatusCode int
	URL        string
	Body       string // truncated to 80 bytes
	HTML       bool   // the body looked like an HTML error page
}

func (e *StatusError) Error() string {
	if e.HTML {
		return fmt.Sprintf("HTTP %d (server returned HTML error page)", e.StatusCode)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// TruncatedError is returned when a 2xx body exceeded MaxResponseSize.
// Without it the cut-off bytes reach the caller's JSON decoder and
// surface as a syntax error, which reads as "the server sent junk" when
// the real cause is our own cap.
type TruncatedError struct {
	URL   string
	Limit int
}

func (e *TruncatedError) Error() string {
	return fmt.Sprintf("response body exceeds the %d byte limit", e.Limit)
}

// Status returns the HTTP status carried by err, or 0 when err is not
// (and does not wrap) a *StatusError.
func Status(err error) int {
	var se *StatusError
	if errors.As(err, &se) {
		return se.StatusCode
	}
	return 0
}

// IsTransport reports whether err is (or wraps) a *TransportError.
func IsTransport(err error) bool {
	var te *TransportError
	return errors.As(err, &te)
}

// IsTruncated reports whether err is (or wraps) a *TruncatedError.
func IsTruncated(err error) bool {
	var te *TruncatedError
	return errors.As(err, &te)
}
