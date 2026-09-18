// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"context"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Rate-limit refusals.
//
// Every Truestamp surface refuses an over-limit request the same way: HTTP
// 429, plus the wait before the request may be repeated. A refusal from a
// per-surface request-rate plug carries that wait in a Retry-After header
// (whole seconds, at least 1); a refusal raised inside an action carries
// it only in the body, in a shape that is the surface's own
// (internal/jsonapi reads the JSON:API envelope's meta.retry_after_ms).
// What every surface shares is the header, the bounds on how long the CLI
// will wait, the jitter added to that wait, and the single retry, so those
// live here.
//
// The shared client built by Init is deliberately NOT given the retry: the
// third-party sources the verifier consults through GetJSONCtx report a
// 429 as `skip` (whitepaper Appendix E.22) and must not be retried, and
// the upgrade check treats GitHub's 429 as a failed check. The Truestamp
// JSON:API gets the retry in internal/jsonapi.Send; the OAuth token and
// revocation endpoints get it from NewRetryAfterTransport.

const (
	// DefaultRetryAfter is the wait before the one retry of a 429 that
	// named no delay at all.
	DefaultRetryAfter = 2 * time.Second

	// MaxRetryAfter bounds the wait the CLI will sit through before its
	// one retry. The per-minute limiters name at most about a minute; a
	// refusal asking for longer (a daily quota, say) is surfaced at once,
	// with the wait it named, rather than parking the command.
	MaxRetryAfter = 60 * time.Second

	// RetryJitter is the most random delay added on top of the wait a
	// refusal named. The server's per-address guard (and the OAuth
	// endpoints' per-address limit) is a window aligned to the clock
	// minute, so every client it refuses in a given minute is told the
	// same second and, retrying on that exact second, would arrive at the
	// boundary together with all the others (truestamp-v2
	// kb/api/json-api.md: "add a little random jitter to its Retry-After
	// rather than retrying on the exact second"). The per-caller budget
	// is a token bucket whose Retry-After is the caller's own short wait,
	// where the draw is merely harmless. A uniform draw from
	// [0, RetryJitter) spreads the herd over a couple of seconds without
	// making a short wait meaningfully longer. It is applied after
	// BoundRetry has decided, so it never turns a retry into a refusal.
	RetryJitter = 2 * time.Second
)

// Jitter returns delay plus a random amount in [0, RetryJitter). Both
// retry sites, internal/jsonapi.Send and the transport below, apply it to
// the wait they are about to sleep.
func Jitter(delay time.Duration) time.Duration {
	return delay + jitter()
}

// jitter draws the random part, a variable so tests can pin it. The draw
// only spreads retries across a couple of seconds; nothing depends on it
// being unpredictable, so math/rand is the right source (gosec G404).
var jitter = func() time.Duration { return rand.N(RetryJitter) } // #nosec G404

// ParseRetryAfter parses a Retry-After header value (RFC 9110 §10.2.3):
// a non-negative integer number of seconds, or an HTTP-date, in which case
// the delay is measured from now. ok is false for an absent or malformed
// value. A date already in the past parses as a zero delay.
func ParseRetryAfter(value string, now time.Time) (delay time.Duration, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if secs, err := strconv.ParseInt(value, 10, 64); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if at, err := http.ParseTime(value); err == nil {
		if d := at.Sub(now); d > 0 {
			return d, true
		}
		return 0, true
	}
	return 0, false
}

// BoundRetry decides whether a 429 is retried, and after how long. named
// is false when the refusal named no delay, in which case DefaultRetryAfter
// applies. The result is false when the wait is over MaxRetryAfter; the
// delay is still returned so a caller can show it.
func BoundRetry(delay time.Duration, named bool) (time.Duration, bool) {
	if !named {
		delay = DefaultRetryAfter
	}
	if delay > MaxRetryAfter {
		return delay, false
	}
	return delay, true
}

// Wait blocks for d, or until ctx is done, whichever comes first, and
// returns ctx.Err() in the second case.
func Wait(ctx context.Context, d time.Duration) error {
	return sleep(ctx, d)
}

// sleep is Wait's implementation, a variable so this package's tests can
// observe a 30 s Retry-After without spending 30 s.
var sleep = func(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Rewind returns a copy of req that can be sent again, or false when the
// body was streamed and cannot be reproduced. Requests built by
// http.NewRequest from a bytes or strings reader carry GetBody and rewind;
// an arbitrary io.Reader does not.
func Rewind(req *http.Request) (*http.Request, bool) {
	retry := req.Clone(req.Context())
	if req.Body == nil || req.Body == http.NoBody {
		return retry, true
	}
	if req.GetBody == nil {
		return nil, false
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, false
	}
	retry.Body = body
	return retry, true
}

// Discard drains (up to 64 KiB) and closes a response body that is being
// thrown away, so the connection can be reused for the retry.
func Discard(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
}

// retryAfterTransport repeats a request once after a 429, waiting the
// Retry-After the refusal names (DefaultRetryAfter when it names none)
// plus Jitter, and gives up instead when that wait is over MaxRetryAfter
// or the body cannot be rewound. It keys on the status and the header only, never on
// the body: the OAuth endpoints answer with an RFC 6749 body whose error
// value is not the signal (`slow_down` on /oauth/token, `invalid_request`
// on /oauth/register and /oauth/revoke).
type retryAfterTransport struct{ base http.RoundTripper }

// NewRetryAfterTransport wraps base (http.DefaultTransport when nil) with
// the retry-once-on-429 behaviour above. internal/auth installs it on the
// clients that call the OAuth token and revocation endpoints. Give the
// client no http.Client.Timeout: that bound spans the whole call, wait
// included, and would cancel the very wait the refusal asked for; bound
// each attempt on the base transport instead (see NewAttemptTransport).
func NewRetryAfterTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return retryAfterTransport{base: base}
}

// NewAttemptTransport returns a copy of the default transport that gives
// up on a single attempt whose response headers have not arrived within
// timeout. It is the per-attempt bound to pair with NewRetryAfterTransport.
func NewAttemptTransport(timeout time.Duration) http.RoundTripper {
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		c := t.Clone()
		c.ResponseHeaderTimeout = timeout
		return c
	}
	return http.DefaultTransport
}

func (t retryAfterTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusTooManyRequests {
		return resp, err
	}
	delay, named := ParseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	delay, ok := BoundRetry(delay, named)
	if !ok {
		return resp, nil
	}
	retry, ok := Rewind(req)
	if !ok {
		return resp, nil
	}
	Discard(resp)
	if werr := Wait(req.Context(), Jitter(delay)); werr != nil {
		return nil, werr
	}
	return t.base.RoundTrip(retry)
}
