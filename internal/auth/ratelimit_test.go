// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// The OAuth protocol endpoints share one request-rate limit and refuse
// with 429, a Retry-After header, and an RFC 6749 body whose `error`
// value differs by endpoint (`slow_down` on /oauth/token, `invalid_request`
// on /oauth/revoke). The client keys on the status and the header only,
// repeats the request once after the wait, and reports a refusal that
// survives that as a rate limit, never as a dead session.

// loginForTest runs the loopback flow against as on an ephemeral port and
// returns the store holding the session.
func loginForTest(t *testing.T, as *fakeAS) Store {
	t.Helper()
	useMockKeyring(t)
	store := NewStore(as.origin())
	orig := loopbackPorts
	loopbackPorts = []int{0}
	t.Cleanup(func() { loopbackPorts = orig })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := Login(ctx, as.origin(), store, LoginOptions{Open: browserOpener(t), Out: io.Discard}); err != nil {
		t.Fatalf("Login: %v", err)
	}
	return store
}

func TestRefresh_RetriesOnceAfterTokenRateLimit(t *testing.T) {
	as := newFakeAS(t)
	store := loginForTest(t, as)
	azr := Resolve(Credentials{}, store)

	// One refusal with the contract's minimum Retry-After of one whole
	// second; the retry goes out after it and is granted.
	as.rateLimit(1, "1")
	start := time.Now()
	if err := azr.ForceRefresh(context.Background()); err != nil {
		t.Fatalf("ForceRefresh after one 429: %v", err)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("the retry must wait the Retry-After, elapsed only %v", elapsed)
	}
	tok, err := azr.BearerToken(context.Background())
	if err != nil || tok != "access-2" {
		t.Errorf("BearerToken = %q, %v; want the refreshed access-2", tok, err)
	}
	if as.refusalsLeft() != 0 {
		t.Errorf("the refusal was not consumed")
	}
}

func TestRefresh_RateLimitedIsNotSessionExpired(t *testing.T) {
	as := newFakeAS(t)
	store := loginForTest(t, as)
	azr := Resolve(Credentials{}, store)

	// A wait over the retry cap is surfaced at once, and as what it is.
	as.rateLimit(2, "3600")
	err := azr.ForceRefresh(context.Background())
	var rl *TokenRateLimitedError
	if !errors.As(err, &rl) {
		t.Fatalf("err = %v, want *TokenRateLimitedError", err)
	}
	if rl.Wait != time.Hour {
		t.Errorf("Wait = %v, want the refusal's 3600 s", rl.Wait)
	}
	if errors.Is(err, ErrSessionExpired) {
		t.Error("a rate-limited refresh is not a dead session")
	}
	if as.refusalsLeft() != 1 {
		t.Errorf("a wait over the cap must not be retried, refusals left = %d", as.refusalsLeft())
	}

	// The session is intact: once the limiter clears, the refresh works
	// with the refresh token that was never consumed.
	as.rateLimit(0, "")
	if err := azr.ForceRefresh(context.Background()); err != nil {
		t.Fatalf("refresh after the limiter cleared: %v", err)
	}
}

func TestRetryTransport_SurfacesTokenRateLimitInsteadOf401(t *testing.T) {
	as := newFakeAS(t)
	store := loginForTest(t, as)
	SetDefault(Resolve(Credentials{}, store))
	t.Cleanup(func() { SetDefault(nil) })

	// The API rejects the token; the refresh the 401 path forces is rate
	// limited. Reporting the 401 would send the holder to re-login for
	// nothing, so the rate limit is the error that surfaces.
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(api.Close)
	as.rateLimit(2, "3600")

	ctx := context.Background()
	client := &http.Client{Transport: NewRetryTransport(nil)}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, api.URL, nil)
	if err := Default().Authorize(ctx, req); err != nil {
		t.Fatalf("authorize: %v", err)
	}
	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	var rl *TokenRateLimitedError
	if !errors.As(err, &rl) {
		t.Fatalf("err = %v (resp %v), want *TokenRateLimitedError instead of the 401", err, resp)
	}
}

func TestLogin_ExchangeRateLimited(t *testing.T) {
	useMockKeyring(t)
	as := newFakeAS(t)
	store := NewStore(as.origin())
	orig := loopbackPorts
	loopbackPorts = []int{0}
	t.Cleanup(func() { loopbackPorts = orig })

	as.rateLimit(2, "3600")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := Login(ctx, as.origin(), store, LoginOptions{Open: browserOpener(t), Out: io.Discard})
	var rl *TokenRateLimitedError
	if !errors.As(err, &rl) {
		t.Fatalf("Login err = %v, want *TokenRateLimitedError from the code exchange", err)
	}
	if _, lerr := store.Load(); lerr == nil {
		t.Error("no session may be saved when the exchange was refused")
	}
}

func TestLogout_RevokeRetriesAfterRateLimit(t *testing.T) {
	as := newFakeAS(t)
	store := loginForTest(t, as)

	// An HTTP-date Retry-After already in the past parses as a zero wait,
	// so the one retry is immediate and the test does not sleep.
	as.rateLimit(1, time.Now().Add(-time.Minute).UTC().Format(http.TimeFormat))
	revoked, err := Logout(context.Background(), store)
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if !revoked {
		t.Error("revocation must succeed on the retry")
	}
	if as.refusalsLeft() != 0 {
		t.Error("the refusal was not consumed")
	}
}
