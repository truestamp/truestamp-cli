// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// Credentials carries the resolved API-key state used by [Resolve].
type Credentials struct {
	// APIKey is the merged api_key value (config file, env, or flag).
	APIKey string
	// APIKeyExplicit is true when the key came from an intentional
	// override, the --api-key flag or TRUESTAMP_API_KEY env, as opposed
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
	// sent holds the last few distinct Authorization values Authorize set,
	// newest last, so the 401 retry can tell a request this authorizer
	// authenticated from one that carries another service's credential.
	sent []string
}

// maxSent bounds the Authorization values an authorizer remembers. A value
// leaves the list only after this many newer ones were set, long after a
// request that carried it, sent before a concurrent refresh, has had its 401.
const maxSent = 4

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
	a.remember(req.Header.Get("Authorization"))
	return nil
}

// remember records h as an Authorization value this authorizer set.
func (a *oauthAuthorizer) remember(h string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if n := len(a.sent); n > 0 && a.sent[n-1] == h {
		return
	}
	a.sent = append(a.sent, h)
	if len(a.sent) > maxSent {
		a.sent = a.sent[len(a.sent)-maxSent:]
	}
}

// issued reports whether h is an Authorization value this authorizer set,
// or the one its current access token makes. The second covers a request
// authorized by another authorizer over the same stored session, such as
// the identity probe `auth login` runs beside the process default.
func (a *oauthAuthorizer) issued(h string) bool {
	if h == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tok != nil && a.tok.AccessToken != "" && sameValue(a.tok.Type()+" "+a.tok.AccessToken, h) {
		return true
	}
	for _, s := range a.sent {
		if sameValue(s, h) {
			return true
		}
	}
	return false
}

func sameValue(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
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
	// Serialize the refresh with every other truestamp process for this
	// origin (refreshlock.go), and read the session again once the lock is
	// held: another process may have rotated it since this one loaded it.
	// Refreshing with this process's token would then present one the
	// server has already rotated, which it reads as reuse and answers by
	// revoking the whole chain, the other process's new token included.
	release := a.store.lockRefresh(ctx)
	defer release()
	if a.adoptStored() && a.tok.Valid() {
		// The other process's access token serves this request too; a
		// forced refresh (a 401, or the console's token_expired) was about
		// this process's older token, which the adopted one replaces.
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
		return nil, tokenRateLimited(err)
	}
	a.tok = nt
	a.sess.AccessToken = nt.AccessToken
	a.sess.RefreshToken = nt.RefreshToken
	a.sess.TokenType = nt.TokenType
	a.sess.Expiry = nt.Expiry
	if scope, ok := nt.Extra("scope").(string); ok && scope != "" {
		a.sess.Scope = scope
	}
	// A failed write must not break the request: the in-memory token is
	// still valid for this process. But it cannot heal later. The server
	// rotates refresh tokens with reuse detection and no grace period, so
	// the next process presents the stored, already-rotated token, and
	// the server revokes the whole chain, this new token included. So the
	// failure is reported now (notifySaveFailure), not left to surface as
	// "session expired" on the next run.
	if err := a.store.Save(a.sess); err != nil {
		notifySaveFailure(err)
	}
	return a.tok, nil
}

// adoptStored replaces the authorizer's session with the stored one when
// the stored refresh token differs from the one this authorizer holds,
// which means another process has refreshed since this one loaded the
// session, and reports whether it did. a.mu must be held.
func (a *oauthAuthorizer) adoptStored() bool {
	stored, err := a.store.Load()
	if err != nil || stored.RefreshToken == "" || stored.RefreshToken == a.tok.RefreshToken {
		return false
	}
	a.sess = stored
	a.conf = refreshConfig(stored)
	a.tok = &oauth2.Token{
		AccessToken:  stored.AccessToken,
		RefreshToken: stored.RefreshToken,
		TokenType:    stored.TokenType,
		Expiry:       stored.Expiry,
	}
	return true
}

var (
	saveFailureMu     sync.Mutex
	saveFailureNotify = func(error) {}
)

// SetSaveFailureNotifier installs the function the OAuth authorizer calls
// when a refreshed session cannot be saved to the keychain or the 0600
// file. The command layer installs one that logs the failure and tells the
// user the next command will need a fresh sign-in; the console installs
// one that holds the message until its full-screen view has closed. nil
// restores the default, which does nothing.
func SetSaveFailureNotifier(fn func(error)) {
	saveFailureMu.Lock()
	defer saveFailureMu.Unlock()
	if fn == nil {
		fn = func(error) {}
	}
	saveFailureNotify = fn
}

func notifySaveFailure(err error) {
	saveFailureMu.Lock()
	fn := saveFailureNotify
	saveFailureMu.Unlock()
	fn(err)
}

// refreshConfig builds the oauth2.Config used for refresh (and revoke).
// AuthStyleInParams is set because the CLI is a public client
// (token_endpoint_auth_method=none): client_id goes in the form body, not
// HTTP Basic, and we skip oauth2's auth-style autodetect probe.
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
// mode as refresh-and-retry-once", implemented here, centrally, for every
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
	// Only retry requests WE authenticated in OAuth mode, never an API-key
	// request (which can't be refreshed) and never a request to another
	// service, even one that carries a credential of its own, such as the
	// GITHUB_TOKEN the upgrade check sends: refreshing on its 401 would
	// spend a refresh rotation, and the retry would send the Truestamp
	// access token to that service.
	azr := Default()
	if azr.Mode() != ModeOAuth || !authenticatedBy(azr, req) {
		return resp, err
	}
	// Force a refresh; if the session is dead (or the body can't be
	// rewound), surface the original 401. One refresh failure is not a
	// 401 in disguise: the authorization server rate limiting the refresh
	// says nothing about the credential, and reporting it as the 401
	// would send the holder to re-login for nothing, so that one is
	// surfaced as the error it is.
	if rerr := azr.ForceRefresh(req.Context()); rerr != nil {
		var rl *TokenRateLimitedError
		if errors.As(rerr, &rl) {
			drainAndClose(resp)
			return nil, rerr
		}
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

// authenticatedBy reports whether req carries an Authorization value azr
// set. Only the OAuth authorizer records what it set; any other reports
// false, so its requests are never retried.
func authenticatedBy(azr Authorizer, req *http.Request) bool {
	o, ok := azr.(*oauthAuthorizer)
	return ok && o.issued(req.Header.Get("Authorization"))
}

func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
}

// IsInvalidGrant reports whether err is an OAuth `invalid_grant` token
// error, the signal that a refresh token is expired, reused, or revoked
// and the session is permanently dead (the caller should stop retrying and
// prompt re-login).
func IsInvalidGrant(err error) bool {
	var re *oauth2.RetrieveError
	if errors.As(err, &re) {
		return re.ErrorCode == "invalid_grant"
	}
	return false
}

// TokenRateLimitedError is returned in OAuth mode when the authorization
// server refused a token request (the code exchange or a refresh) with
// 429 after the one retry the token client makes. The session is intact,
// which is why this is deliberately not ErrSessionExpired: the holder
// should wait, not re-login. Wait is the Retry-After the refusal named,
// zero when it named none.
type TokenRateLimitedError struct{ Wait time.Duration }

func (e *TokenRateLimitedError) Error() string {
	if e.Wait > 0 {
		return fmt.Sprintf("the authorization server is rate limiting token requests, retry after %ds",
			int64(math.Ceil(e.Wait.Seconds())))
	}
	return "the authorization server is rate limiting token requests, wait a moment and retry"
}

// tokenRateLimited converts a 429 from the token endpoint into a
// *TokenRateLimitedError and returns any other error as it is. It keys on
// the status and the Retry-After header, never on the RFC 6749 `error`
// value (`slow_down` there today), per the rate-limit contract.
func tokenRateLimited(err error) error {
	var re *oauth2.RetrieveError
	if !errors.As(err, &re) || re.Response == nil || re.Response.StatusCode != http.StatusTooManyRequests {
		return err
	}
	wait, _ := httpclient.ParseRetryAfter(re.Response.Header.Get("Retry-After"), time.Now())
	return &TokenRateLimitedError{Wait: wait}
}

// tokenAttemptTimeout bounds each attempt against a token or revocation
// endpoint. It is applied per attempt (on the transport) rather than as
// http.Client.Timeout, which spans the whole call and would cancel the
// Retry-After wait between a 429 and the one retry.
const tokenAttemptTimeout = 30 * time.Second

// tokenHTTPClient is the client for the OAuth protocol endpoints: each
// attempt bounded by tokenAttemptTimeout, a 429 repeated once after its
// Retry-After (httpclient.NewRetryAfterTransport), independent of the
// per-command shared client.
func tokenHTTPClient() *http.Client {
	return &http.Client{Transport: httpclient.NewRetryAfterTransport(httpclient.NewAttemptTransport(tokenAttemptTimeout))}
}

// tokenContext returns ctx carrying the dedicated HTTP client for OAuth
// token endpoint calls.
func tokenContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, tokenHTTPClient())
}
