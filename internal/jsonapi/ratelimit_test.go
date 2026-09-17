// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package jsonapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// TestMain installs an api-key Authorizer for the whole package so the
// transport tests can send authenticated requests.
func TestMain(m *testing.M) {
	auth.SetDefault(auth.APIKeyAuthorizer("test-key"))
	code := m.Run()
	auth.SetDefault(nil)
	os.Exit(code)
}

// The three refusal shapes of the rate-limit contract, verbatim.
const (
	// From the per-surface request-rate plug: a Retry-After header rides
	// alongside (the tests set it), and the meta names the same wait.
	plugRefusal = `{"errors":[{"id":"9c1f0a2e-5b6d-4e7f-8a9b-0c1d2e3f4a5b","status":"429","code":"rate_limited","title":"Too Many Requests","detail":"API rate limit exceeded. Slow down and retry shortly.","meta":{"retry_after_ms":36412,"limit":120}}]}`
	// Raised inside an action: no header, the wait only in the meta.
	actionRefusal = `{"errors":[{"status":"429","code":"rate_limited","title":"Too Many Requests","detail":"Rate limit exceeded. Retry after 12 seconds.","meta":{"retry_after_ms":11204,"limit":1000}}]}`
	// A request whose cost is above the limit: never admitted.
	neverRefusal = `{"errors":[{"status":"429","code":"rate_limited","title":"Too Many Requests","detail":"Rate limit exceeded.","meta":{"retry_after_ms":null,"limit":1000}}]}`
)

// stubWait replaces the sleeper for one test and records every wait.
func stubWait(t *testing.T) *[]time.Duration {
	t.Helper()
	var waits []time.Duration
	orig := wait
	wait = func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)
		return ctx.Err()
	}
	t.Cleanup(func() { wait = orig })
	return &waits
}

func TestParseError_RateLimitMeta(t *testing.T) {
	plug := ParseError(http.StatusTooManyRequests, []byte(plugRefusal))
	if plug.Code != CodeRateLimited || plug.RetryAfterMS != 36412 || plug.Limit != 120 || plug.NeverAdmitted {
		t.Errorf("plug refusal parsed as %+v", plug)
	}
	if !errors.Is(plug, ErrRateLimited) {
		t.Errorf("a 429 must classify as ErrRateLimited, got %v", plug.Sentinel)
	}
	if plug.Detail != "API rate limit exceeded. Slow down and retry shortly." {
		t.Errorf("detail = %q", plug.Detail)
	}

	action := ParseError(http.StatusTooManyRequests, []byte(actionRefusal))
	if action.RetryAfterMS != 11204 || action.Limit != 1000 || action.NeverAdmitted {
		t.Errorf("action refusal parsed as %+v", action)
	}

	never := ParseError(http.StatusTooManyRequests, []byte(neverRefusal))
	if !never.NeverAdmitted || never.RetryAfterMS != 0 || never.Limit != 1000 {
		t.Errorf("null retry_after_ms must read as never admitted, got %+v", never)
	}

	// A float is accepted, a negative or non-numeric value reads as absent.
	lenient := ParseError(429, []byte(`{"errors":[{"detail":"x","meta":{"retry_after_ms":36412.0,"limit":"many"}}]}`))
	if lenient.RetryAfterMS != 36412 || lenient.Limit != 0 {
		t.Errorf("lenient meta parsed as %+v", lenient)
	}
	negative := ParseError(429, []byte(`{"errors":[{"detail":"x","meta":{"retry_after_ms":-1}}]}`))
	if negative.RetryAfterMS != 0 || negative.NeverAdmitted {
		t.Errorf("negative retry_after_ms must read as absent, got %+v", negative)
	}

	// The meta is read on a 429 only: another status carrying the same
	// keys keeps its own behaviour.
	other := ParseError(http.StatusForbidden, []byte(actionRefusal))
	if other.RetryAfterMS != 0 || other.Limit != 0 || !errors.Is(other, ErrForbidden) {
		t.Errorf("rate-limit meta must be ignored on a 403, got %+v", other)
	}
}

func TestAPIError_RetryAfterDelay_HeaderWinsOverBody(t *testing.T) {
	both := &APIError{Status: 429, RetryAfter: "37", RetryAfterMS: 36412}
	if d, ok := both.RetryAfterDelay(); !ok || d != 37*time.Second {
		t.Errorf("header must win: got %v, %v", d, ok)
	}
	bodyOnly := &APIError{Status: 429, RetryAfterMS: 36412}
	if d, ok := bodyOnly.RetryAfterDelay(); !ok || d != 36412*time.Millisecond {
		t.Errorf("meta.retry_after_ms must be used when the header is absent: got %v, %v", d, ok)
	}
	neither := &APIError{Status: 429}
	if _, ok := neither.RetryAfterDelay(); ok {
		t.Error("a 429 naming no wait must report none")
	}
	not429 := &APIError{Status: 503, RetryAfter: "37"}
	if _, ok := not429.RetryAfterDelay(); ok {
		t.Error("only a 429 carries a rate-limit wait")
	}
}

func TestAPIError_RetryDelay(t *testing.T) {
	cases := []struct {
		name string
		err  *APIError
		want time.Duration
		ok   bool
	}{
		{"header", &APIError{Status: 429, RetryAfter: "12"}, 12 * time.Second, true},
		{"meta", &APIError{Status: 429, RetryAfterMS: 11204}, 11204 * time.Millisecond, true},
		{"none: default", &APIError{Status: 429}, httpclient.DefaultRetryAfter, true},
		{"null: never", &APIError{Status: 429, NeverAdmitted: true, RetryAfterMS: 0}, 0, false},
		{"over cap: shown, not slept", &APIError{Status: 429, RetryAfterMS: 3_600_000}, time.Hour, false},
		{"not a 429", &APIError{Status: 500, RetryAfter: "12"}, 0, false},
	}
	for _, tc := range cases {
		got, ok := tc.err.RetryDelay()
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: RetryDelay() = %v, %v; want %v, %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

// refusingServer answers the first n requests with a 429 carrying body
// (and Retry-After when non-empty), then 200 with `{"data":[]}`.
func refusingServer(t *testing.T, n int32, retryAfter, body string) (Config, *atomic.Int32, *[]*http.Request) {
	t.Helper()
	var calls atomic.Int32
	var seen []*http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, _ := io.ReadAll(r.Body)
		clone := r.Clone(context.Background())
		clone.Body = io.NopCloser(bytesReader(payload))
		seen = append(seen, clone)
		if calls.Add(1) <= n {
			if retryAfter != "" {
				w.Header().Set("Retry-After", retryAfter)
			}
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(body))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(srv.Close)
	return Config{APIURL: srv.URL}, &calls, &seen
}

func TestSend_RetriesOnceAfterPlugRefusal(t *testing.T) {
	waits := stubWait(t)
	var notified []string
	SetRateLimitNotifier(func(wait time.Duration, limit int64) {
		notified = append(notified, wait.String()+"/"+itoa(limit))
	})
	t.Cleanup(func() { SetRateLimitNotifier(nil) })

	cfg, calls, _ := refusingServer(t, 1, "37", plugRefusal)
	body, err := Get(context.Background(), cfg, "/items")
	if err != nil {
		t.Fatalf("Get after one 429: %v", err)
	}
	if string(body) != `{"data":[]}` {
		t.Errorf("body = %s", body)
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want 2", calls.Load())
	}
	if len(*waits) != 1 || (*waits)[0] != 37*time.Second {
		t.Errorf("waits = %v, want [37s]: the header wins over meta.retry_after_ms", *waits)
	}
	if len(notified) != 1 || notified[0] != "37s/120" {
		t.Errorf("notifier saw %v, want [37s/120]", notified)
	}
}

func TestSend_ActionRefusalWaitsRetryAfterMS(t *testing.T) {
	waits := stubWait(t)
	cfg, calls, _ := refusingServer(t, 1, "", actionRefusal)
	if _, err := Get(context.Background(), cfg, "/items"); err != nil {
		t.Fatalf("Get after one 429: %v", err)
	}
	if calls.Load() != 2 || len(*waits) != 1 || (*waits)[0] != 11204*time.Millisecond {
		t.Errorf("calls %d, waits %v; want 2 calls and [11.204s]", calls.Load(), *waits)
	}
}

func TestSend_DefaultWaitWhenNothingNamed(t *testing.T) {
	waits := stubWait(t)
	cfg, calls, _ := refusingServer(t, 1, "", `{"errors":[{"status":"429","title":"Too Many Requests"}]}`)
	if _, err := Get(context.Background(), cfg, "/items"); err != nil {
		t.Fatalf("Get after one 429: %v", err)
	}
	if calls.Load() != 2 || len(*waits) != 1 || (*waits)[0] != httpclient.DefaultRetryAfter {
		t.Errorf("calls %d, waits %v; want 2 calls and [%v]", calls.Load(), *waits, httpclient.DefaultRetryAfter)
	}
}

func TestSend_NeverAdmittedIsNotRetried(t *testing.T) {
	waits := stubWait(t)
	cfg, calls, _ := refusingServer(t, 1, "", neverRefusal)
	_, err := Get(context.Background(), cfg, "/items")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !apiErr.NeverAdmitted || apiErr.Limit != 1000 {
		t.Errorf("err = %+v, want NeverAdmitted with limit 1000", err)
	}
	if calls.Load() != 1 || len(*waits) != 0 {
		t.Errorf("calls %d, waits %v; a null retry_after_ms must never be retried", calls.Load(), *waits)
	}
}

func TestSend_OverCapIsNotRetried(t *testing.T) {
	waits := stubWait(t)
	cfg, calls, _ := refusingServer(t, 1, "3600", plugRefusal)
	_, err := Get(context.Background(), cfg, "/items")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want a rate-limited *APIError", err)
	}
	if apiErr.RetryAfter != "3600" {
		t.Errorf("the refusal's Retry-After must be carried for display, got %q", apiErr.RetryAfter)
	}
	if calls.Load() != 1 || len(*waits) != 0 {
		t.Errorf("calls %d, waits %v; a wait over the cap is surfaced, not slept", calls.Load(), *waits)
	}
}

func TestSend_SecondRefusalSurfaces(t *testing.T) {
	waits := stubWait(t)
	cfg, calls, _ := refusingServer(t, 2, "", actionRefusal)
	_, err := Get(context.Background(), cfg, "/items")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited after the retry", err)
	}
	if calls.Load() != 2 || len(*waits) != 1 {
		t.Errorf("calls %d, waits %v; want one retry, never more", calls.Load(), *waits)
	}
}

func TestSend_RetryResendsBodyAndHeaders(t *testing.T) {
	stubWait(t)
	cfg, calls, seen := refusingServer(t, 1, "", actionRefusal)
	cfg.Team = "team_42"
	if _, err := Post(context.Background(), cfg, "/items", map[string]string{"k": "v"}); err != nil {
		t.Fatalf("Post after one 429: %v", err)
	}
	if calls.Load() != 2 || len(*seen) != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
	first, second := (*seen)[0], (*seen)[1]
	b1, _ := io.ReadAll(first.Body)
	b2, _ := io.ReadAll(second.Body)
	if string(b1) != `{"k":"v"}` || string(b2) != `{"k":"v"}` {
		t.Errorf("bodies: first %q, retry %q; the retry must resend the body", b1, b2)
	}
	for _, h := range []string{"Authorization", "Content-Type", "Accept", "Tenant"} {
		if first.Header.Get(h) == "" || first.Header.Get(h) != second.Header.Get(h) {
			t.Errorf("%s: first %q, retry %q; the retry must carry the same headers", h, first.Header.Get(h), second.Header.Get(h))
		}
	}
}

func TestSend_CancelledDuringWait(t *testing.T) {
	orig := wait
	wait = func(ctx context.Context, d time.Duration) error { return context.Canceled }
	t.Cleanup(func() { wait = orig })
	cfg, calls, _ := refusingServer(t, 1, "", actionRefusal)
	_, err := Get(context.Background(), cfg, "/items")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1: the retry never went out", calls.Load())
	}
}

func TestSend_OtherStatusesAreNotRetried(t *testing.T) {
	waits := stubWait(t)
	for _, code := range []int{400, 401, 403, 404, 422, 500, 503} {
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"errors":[{"detail":"no"}]}`))
		}))
		_, err := Get(context.Background(), Config{APIURL: srv.URL}, "/x")
		srv.Close()
		if err == nil || calls.Load() != 1 {
			t.Errorf("HTTP %d: err %v after %d calls; want one call and an error", code, err, calls.Load())
		}
	}
	if len(*waits) != 0 {
		t.Errorf("no status but 429 may wait: %v", *waits)
	}
}

// rateLimitedAuthorizer plays an OAuth session whose refresh the
// authorization server is rate limiting.
type rateLimitedAuthorizer struct{ wait time.Duration }

func (a rateLimitedAuthorizer) Mode() auth.Mode { return auth.ModeOAuth }
func (a rateLimitedAuthorizer) Authorize(context.Context, *http.Request) error {
	return &auth.TokenRateLimitedError{Wait: a.wait}
}
func (a rateLimitedAuthorizer) BearerToken(context.Context) (string, error) {
	return "", &auth.TokenRateLimitedError{Wait: a.wait}
}
func (a rateLimitedAuthorizer) ForceRefresh(context.Context) error {
	return &auth.TokenRateLimitedError{Wait: a.wait}
}
func (a rateLimitedAuthorizer) AccessTokenExpiry() time.Time { return time.Time{} }

func TestDoRaw_TokenRateLimitedIs429NotUnauthorized(t *testing.T) {
	auth.SetDefault(rateLimitedAuthorizer{wait: 37 * time.Second})
	t.Cleanup(func() { auth.SetDefault(auth.APIKeyAuthorizer("test-key")) })

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
	}))
	t.Cleanup(srv.Close)

	_, err := Get(context.Background(), Config{APIURL: srv.URL}, "/items")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T %v, want *APIError", err, err)
	}
	if apiErr.Status != http.StatusTooManyRequests || !errors.Is(err, ErrRateLimited) || errors.Is(err, ErrUnauthorized) {
		t.Errorf("a rate-limited refresh must be a 429, not a 401: %+v", apiErr)
	}
	if apiErr.RetryAfter != "37" {
		t.Errorf("RetryAfter = %q, want the refusal's wait in whole seconds", apiErr.RetryAfter)
	}
	if calls.Load() != 0 {
		t.Errorf("no request may go out without a credential, saw %d", calls.Load())
	}
}

func TestErrorFromResponse_CarriesRetryAfterOn429Only(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "5")
	on429 := ErrorFromResponse(&http.Response{StatusCode: 429, Header: h}, []byte(plugRefusal))
	if on429.RetryAfter != "5" || on429.RetryAfterMS != 36412 {
		t.Errorf("429: %+v", on429)
	}
	on503 := ErrorFromResponse(&http.Response{StatusCode: 503, Header: h}, []byte(`{"errors":[{"detail":"down"}]}`))
	if on503.RetryAfter != "" || !errors.Is(on503, ErrServer) {
		t.Errorf("503: Retry-After is not a rate-limit signal, got %+v", on503)
	}
}
