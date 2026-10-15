// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package httpclient provides a shared HTTP client for all external API calls.
// The client is safe for concurrent use and reuses connections.
// Call Init once during startup to set the timeout; the default is 10s.
//
// internal/verifier's reads outside the document, such as the keyring
// read of SPEC.md section 10.2, step 0 (CONFIGURATION), do not use the
// shared client: each goes through a [SingleRead], one attempt that
// follows no redirect and ends within one deadline (SPEC.md sections 8.3
// (External verifiers) and 10.4 (Chain and source reads)). The verify
// command's current path, internal/verify through internal/external, still
// makes its keyring, chain and NIST Beacon reads through the shared client
// and its redirect policy.
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
var httpClient = newClient(10 * time.Second)

// ErrRedirectDowngrade is returned (wrapped in a *url.Error by net/http)
// when a server tries to redirect an https request to a plaintext http
// URL. Following such a redirect would strip TLS mid-chain, which for the
// keyring fetch in particular is a total compromise: an attacker who can
// substitute the keyring can substitute signing keys, and every downstream
// signature then validates against their key. See VerifyKeyring's threat
// note in internal/external/keyring.go.
var ErrRedirectDowngrade = errors.New("refusing redirect that downgrades https to http")

// maxRedirects matches net/http's default cap. It is restated here because
// supplying CheckRedirect replaces that default entirely.
const maxRedirects = 10

// checkRedirect refuses any redirect that drops from https to http, and
// otherwise preserves net/http's default 10-hop cap.
//
// Cross-host redirects that STAY on https are deliberately allowed: GitHub
// release assets (fetched by internal/selfupgrade through this same client)
// redirect from github.com to objects.githubusercontent.com, so banning
// cross-host would break `truestamp upgrade`. A redirect that keeps TLS is
// still authenticated by the CA chain; the threat this guards is the
// downgrade to an unauthenticated channel.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	prev := via[len(via)-1]
	if prev.URL.Scheme == "https" && req.URL.Scheme != "https" {
		return fmt.Errorf("%w: %s -> %s", ErrRedirectDowngrade, prev.URL.Scheme, req.URL.Scheme)
	}
	return nil
}

// newClient builds a shared client carrying the redirect policy. Every
// construction site of the shared client must go through it so the policy
// cannot be lost by building an http.Client literal. A [SingleRead] builds
// its own client, with the stricter policy of following no redirect.
func newClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, CheckRedirect: checkRedirect}
}

// userAgent is stamped onto every outbound request by SetUserAgent and
// applied centrally in Do. Empty means "don't override any UA the caller
// already set". Populated once at startup via SetUserAgent from cmd/root.
var userAgent string

// MaxResponseSize limits HTTP response bodies to 1 MB to prevent OOM.
const MaxResponseSize = 1 << 20

// Init creates a new HTTP client with the given timeout.
// Must be called once during startup before any external calls.
// The redirect policy in [checkRedirect] is reinstalled here; building a
// bare http.Client instead would silently drop it.
func Init(timeout time.Duration) {
	httpClient = newClient(timeout)
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

// SingleRead makes the reads a verifier makes outside the document, such
// as the keyring read of SPEC.md section 10.2, step 0 (CONFIGURATION), by
// the rules SPEC.md section 8.3 (External verifiers) and section 10.4
// (Chain and source reads) need of a read whose answer decides a verdict.
// It is modeled on the Truestamp application's
// Truestamp.SourceCache.ReadBounds (lib/truestamp/source_cache/read_bounds.ex
// in truestamp-v2), which gives every verification read one attempt, no
// redirect followed and one deadline over the whole read:
//
//   - one GET, with no retry;
//   - no redirect followed, to any host: a 3xx is the answer, an HTTP error
//     like any status but 200 OK, so the origin a read names is the only
//     one that can answer it;
//   - one deadline over the whole read: the connect, a TLS handshake, the
//     wait for the headers and every wait for the body, so a source that
//     answers slowly, or trickles its answer, ends the read when the
//     deadline passes;
//   - a bound on the body, so an answer larger than its caller can use is
//     refused unread past the bound.
//
// It has its own client, apart from the shared one, so neither the shared
// client's redirect policy nor the transport [SetTransport] installs (the
// auth package's 401 retry) applies to it, and it sends no credential.
type SingleRead struct {
	client   *http.Client
	deadline time.Duration
}

// NewSingleRead returns a [SingleRead] whose reads go through transport,
// http.DefaultTransport when it is nil, and each end within deadline.
func NewSingleRead(transport http.RoundTripper, deadline time.Duration) *SingleRead {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &SingleRead{
		client:   &http.Client{Transport: transport, CheckRedirect: noRedirect},
		deadline: deadline,
	}
}

// noRedirect makes the client return a 3xx as the answer, unfollowed.
func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// Get reads url once, within the reader's deadline, and returns the body
// of a 200 OK answer of at most limit octets. Errors are typed as
// [GetJSONCtx]'s are: a *TransportError for a failure before the whole
// body arrived (the deadline passing included), a *StatusError for any
// status but 200 OK (a 3xx included, unfollowed), and a *TruncatedError
// for a body over limit, of which at most limit + 1 octets are read.
func (r *SingleRead) Get(ctx context.Context, url string, limit int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, r.deadline)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &TransportError{URL: url, Err: err}
	}
	stampUserAgent(req)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, &TransportError{URL: url, Err: err}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		head, _ := io.ReadAll(io.LimitReader(resp.Body, 81))
		bodyStr := Truncate(string(head), 80)
		return nil, &StatusError{
			StatusCode: resp.StatusCode,
			URL:        url,
			Body:       bodyStr,
			HTML:       len(bodyStr) > 0 && bodyStr[0] == '<',
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	if err != nil {
		return nil, &TransportError{URL: url, Err: err}
	}
	if len(body) > limit {
		return nil, &TruncatedError{URL: url, Limit: limit}
	}
	return body, nil
}
