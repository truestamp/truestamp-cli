// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// Credentials carries the resolved API-key state used by [Resolve].
type Credentials struct {
	// APIKey is the merged api_key value (config file, env, or flag).
	APIKey string
	// APIKeyExplicit is true when the key came from an intentional
	// override — the --api-key flag or TRUESTAMP_API_KEY env — as opposed
	// to the config file. An explicit key wins over an OAuth session so
	// CI/headless behavior is deterministic.
	APIKeyExplicit bool
}

// Resolve selects the active credential per the documented precedence:
// explicit API key → OAuth session → config-file API key → none.
func Resolve(creds Credentials, store Store) Authorizer {
	if creds.APIKeyExplicit && creds.APIKey != "" {
		return apiKeyAuthorizer{key: creds.APIKey}
	}
	if sess, err := store.Load(); err == nil && sess.RefreshToken != "" {
		return newOAuthAuthorizer(sess, store)
	}
	if creds.APIKey != "" {
		return apiKeyAuthorizer{key: creds.APIKey}
	}
	return noneAuthorizer{}
}

// APIKeyAuthorizer returns an Authorizer that presents key as a Bearer
// token. Exported for cmd wiring and for tests that exercise the API-key
// path directly.
func APIKeyAuthorizer(key string) Authorizer { return apiKeyAuthorizer{key: key} }

// --- API-key authorizer -----------------------------------------------------

type apiKeyAuthorizer struct{ key string }

func (a apiKeyAuthorizer) Mode() Mode { return ModeAPIKey }

func (a apiKeyAuthorizer) Authorize(_ context.Context, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+a.key)
	return nil
}

func (a apiKeyAuthorizer) BearerToken(context.Context) (string, error) {
	return a.key, nil
}

// ForceRefresh is a no-op: an API key is long-lived and not refreshable.
func (a apiKeyAuthorizer) ForceRefresh(context.Context) error { return nil }

// AccessTokenExpiry is the zero time: an API key has no expiry.
func (a apiKeyAuthorizer) AccessTokenExpiry() time.Time { return time.Time{} }

// --- no-credentials authorizer ----------------------------------------------

type noneAuthorizer struct{}

func (noneAuthorizer) Mode() Mode { return ModeNone }

// Authorize is a no-op: a genuinely public request still goes out, and the
// server returns 401 if the resource needs auth.
func (noneAuthorizer) Authorize(context.Context, *http.Request) error { return nil }

func (noneAuthorizer) BearerToken(context.Context) (string, error) {
	return "", ErrNoCredentials
}

// ForceRefresh is a no-op: there is no credential to refresh.
func (noneAuthorizer) ForceRefresh(context.Context) error { return nil }

// AccessTokenExpiry is the zero time: there is no credential.
func (noneAuthorizer) AccessTokenExpiry() time.Time { return time.Time{} }

// --- OAuth authorizer -------------------------------------------------------

// oauthAuthorizer holds a self-managed access/refresh token pair. It caches
// a valid access token and runs the refresh grant when the token is expired
// (or when ForceRefresh is called), persisting the rotated refresh token on
// every refresh. A mutex serializes refreshes so concurrent HTTP commands
// and the console WebSocket never trigger a refresh storm.
type oauthAuthorizer struct {
	conf  *oauth2.Config
	store Store

	mu   sync.Mutex
	tok  *oauth2.Token
	sess Session
}

func newOAuthAuthorizer(sess Session, store Store) Authorizer {
	return &oauthAuthorizer{
		conf:  refreshConfig(sess),
		store: store,
		tok: &oauth2.Token{
			AccessToken:  sess.AccessToken,
			RefreshToken: sess.RefreshToken,
			TokenType:    sess.TokenType,
			Expiry:       sess.Expiry,
		},
		sess: sess,
	}
}

func (a *oauthAuthorizer) Mode() Mode { return ModeOAuth }

func (a *oauthAuthorizer) Authorize(ctx context.Context, req *http.Request) error {
	tok, err := a.token(ctx, false)
	if err != nil {
		return err
	}
	tok.SetAuthHeader(req)
	return nil
}

func (a *oauthAuthorizer) BearerToken(ctx context.Context) (string, error) {
	tok, err := a.token(ctx, false)
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

// ForceRefresh runs the refresh grant unconditionally (skipping the cached
// access token's validity check) so the next request carries a brand-new
// token even when the local copy still looks valid but the server rejected
// it (clock skew, server-side expiry/revocation).
func (a *oauthAuthorizer) ForceRefresh(ctx context.Context) error {
	_, err := a.token(ctx, true)
	return err
}

// AccessTokenExpiry returns the cached access token's expiry so the console
// WebSocket can schedule a proactive in-band refresh before the server
// force-disconnects.
func (a *oauthAuthorizer) AccessTokenExpiry() time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tok == nil {
		return time.Time{}
	}
	return a.tok.Expiry
}

// token returns a valid access token. It refreshes (and persists the
// rotated refresh token) when the cached token is expired or force is set,
// translating a dead-session refresh failure into [ErrSessionExpired].
func (a *oauthAuthorizer) token(ctx context.Context, force bool) (*oauth2.Token, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !force && a.tok.Valid() {
		return a.tok, nil
	}
	// A source seeded with only the refresh token (no access token, zero
	// expiry) treats itself as expired, so the first Token() call runs the
	// refresh grant. The 30s-bounded token client honors the caller's ctx
	// for cancellation.
	src := a.conf.TokenSource(tokenContext(ctx), &oauth2.Token{RefreshToken: a.tok.RefreshToken})
	nt, err := src.Token()
	if err != nil {
		if IsInvalidGrant(err) {
			return nil, ErrSessionExpired
		}
		return nil, err
	}
	a.tok = nt
	a.sess.AccessToken = nt.AccessToken
	a.sess.RefreshToken = nt.RefreshToken
	a.sess.TokenType = nt.TokenType
	a.sess.Expiry = nt.Expiry
	if scope, ok := nt.Extra("scope").(string); ok && scope != "" {
		a.sess.Scope = scope
	}
	// Best-effort persist: a failed write must not break the request — the
	// in-memory token is still valid for this process.
	_ = a.store.Save(a.sess)
	return a.tok, nil
}

// refreshConfig builds the oauth2.Config used for refresh (and revoke).
// AuthStyleInParams is set because the CLI is a public client
// (token_endpoint_auth_method=none): client_id goes in the form body, not
// HTTP Basic — and we skip oauth2's auth-style autodetect probe.
func refreshConfig(sess Session) *oauth2.Config {
	return &oauth2.Config{
		ClientID: ClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:   sess.AuthURL,
			TokenURL:  sess.TokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

// --- reactive 401 retry transport -------------------------------------------

// retryTransport retries an OAuth-authenticated request once after forcing a
// token refresh when the server returns 401. The server's OAuth→API-key
// fallback means an expired OAuth token can surface as a bare 401 without a
// `WWW-Authenticate` challenge, so the contract is "treat any 401 in OAuth
// mode as refresh-and-retry-once" — implemented here, centrally, for every
// HTTP call site.
type retryTransport struct{ base http.RoundTripper }

// NewRetryTransport wraps base (or http.DefaultTransport) with the reactive
// 401 → refresh → retry-once behavior. Install it on the shared HTTP client.
func NewRetryTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return retryTransport{base: base}
}

func (t retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}
	// Only retry requests WE authenticated (they carry a Bearer header) in
	// OAuth mode — never an unrelated 401 from an external API or an
	// API-key request (which can't be refreshed).
	azr := Default()
	if azr.Mode() != ModeOAuth || req.Header.Get("Authorization") == "" {
		return resp, err
	}
	// Force a refresh; if the session is dead (or the body can't be
	// rewound), surface the original 401.
	if rerr := azr.ForceRefresh(req.Context()); rerr != nil {
		return resp, err
	}
	if req.Body != nil {
		if req.GetBody == nil {
			return resp, err
		}
		body, berr := req.GetBody()
		if berr != nil {
			return resp, err
		}
		req.Body = body
	}
	if aerr := azr.Authorize(req.Context(), req); aerr != nil {
		return resp, err
	}
	drainAndClose(resp)
	return t.base.RoundTrip(req)
}

func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
}

// IsInvalidGrant reports whether err is an OAuth `invalid_grant` token
// error — the signal that a refresh token is expired, reused, or revoked
// and the session is permanently dead (the caller should stop retrying and
// prompt re-login).
func IsInvalidGrant(err error) bool {
	var re *oauth2.RetrieveError
	if errors.As(err, &re) {
		return re.ErrorCode == "invalid_grant"
	}
	return false
}

// tokenContext returns ctx carrying a dedicated HTTP client for OAuth token
// endpoint calls (a bounded timeout, independent of the per-command client).
func tokenContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: 30 * time.Second})
}
