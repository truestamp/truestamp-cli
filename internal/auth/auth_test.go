// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

// fakeAS is an in-memory OAuth 2.1 authorization server for tests: RFC 8414
// discovery, an authorize endpoint that 302s to the loopback redirect with
// a code, a token endpoint enforcing S256 PKCE and rotating refresh tokens,
// and a revoke endpoint.
type fakeAS struct {
	srv *httptest.Server

	mu           sync.Mutex
	challenges   map[string]string // code -> code_challenge
	refresh      map[string]bool   // valid (unrotated) refresh tokens
	revoked      map[string]bool
	tokenCounter int
	// refusals is how many upcoming token and revocation requests answer
	// 429 with retryAfter as the Retry-After header, in the RFC 6749
	// shape the rate-limit contract gives the OAuth endpoints. See
	// rateLimit.
	refusals   int
	retryAfter string
}

// rateLimit makes the next n token and revocation requests answer 429
// with the given Retry-After header (none when empty).
func (f *fakeAS) rateLimit(n int, retryAfter string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refusals, f.retryAfter = n, retryAfter
}

// refusalsLeft reports how many rate-limit refusals are still queued.
func (f *fakeAS) refusalsLeft() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refusals
}

// refuse answers a queued rate-limit refusal and reports whether it did.
// code is the RFC 6749 error value: `slow_down` on /oauth/token and
// `invalid_request` on /oauth/revoke, which the client must not key on.
func (f *fakeAS) refuse(w http.ResponseWriter, code string) bool {
	f.mu.Lock()
	if f.refusals == 0 {
		f.mu.Unlock()
		return false
	}
	f.refusals--
	retryAfter := f.retryAfter
	f.mu.Unlock()
	if retryAfter != "" {
		w.Header().Set("Retry-After", retryAfter)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.MarshalWrite(w, map[string]string{
		"error":             code,
		"error_description": "Rate limit exceeded; retry after 1 seconds",
	})
	return true
}

func newFakeAS(t *testing.T) *fakeAS {
	t.Helper()
	f := &fakeAS{
		challenges: map[string]string{},
		refresh:    map[string]bool{},
		revoked:    map[string]bool{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(discoveryPath, f.handleDiscovery)
	mux.HandleFunc("/oauth/authorize", f.handleAuthorize)
	mux.HandleFunc("/oauth/token", f.handleToken)
	mux.HandleFunc("/oauth/revoke", f.handleRevoke)
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeAS) origin() string { return f.srv.URL }

func (f *fakeAS) handleDiscovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"issuer":                                f.srv.URL,
		"authorization_endpoint":                f.srv.URL + "/oauth/authorize",
		"token_endpoint":                        f.srv.URL + "/oauth/token",
		"revocation_endpoint":                   f.srv.URL + "/oauth/revoke",
		"registration_endpoint":                 f.srv.URL + "/oauth/register",
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"response_types_supported":              []string{"code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"scopes_supported":                      []string{"api:read", "api:write", "console:read", "console:write"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	})
}

func (f *fakeAS) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	challenge := q.Get("code_challenge")
	if q.Get("code_challenge_method") != "S256" || challenge == "" {
		http.Error(w, "missing S256 challenge", http.StatusBadRequest)
		return
	}
	code := fmt.Sprintf("code-%d", time.Now().UnixNano())
	f.mu.Lock()
	f.challenges[code] = challenge
	f.mu.Unlock()
	http.Redirect(w, r, redirectURI+"?code="+url.QueryEscape(code)+"&state="+url.QueryEscape(state), http.StatusFound)
}

func (f *fakeAS) handleToken(w http.ResponseWriter, r *http.Request) {
	if f.refuse(w, "slow_down") {
		return
	}
	_ = r.ParseForm()
	if r.PostForm.Get("client_id") != ClientID {
		writeTokenError(w, "invalid_client")
		return
	}
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		code := r.PostForm.Get("code")
		verifier := r.PostForm.Get("code_verifier")
		f.mu.Lock()
		challenge, ok := f.challenges[code]
		delete(f.challenges, code)
		f.mu.Unlock()
		if !ok || s256(verifier) != challenge {
			writeTokenError(w, "invalid_grant")
			return
		}
		f.issueTokens(w, "api:read api:write console:read console:write")
	case "refresh_token":
		rt := r.PostForm.Get("refresh_token")
		f.mu.Lock()
		valid := f.refresh[rt] && !f.revoked[rt]
		delete(f.refresh, rt) // single-use rotation
		f.mu.Unlock()
		if !valid {
			writeTokenError(w, "invalid_grant")
			return
		}
		f.issueTokens(w, "")
	default:
		writeTokenError(w, "unsupported_grant_type")
	}
}

func (f *fakeAS) issueTokens(w http.ResponseWriter, scope string) {
	f.mu.Lock()
	f.tokenCounter++
	n := f.tokenCounter
	rt := fmt.Sprintf("refresh-%d", n)
	f.refresh[rt] = true
	f.mu.Unlock()
	body := map[string]any{
		"access_token":  fmt.Sprintf("access-%d", n),
		"refresh_token": rt,
		"token_type":    "Bearer",
		"expires_in":    3600,
	}
	if scope != "" {
		body["scope"] = scope
	}
	writeJSON(w, body)
}

func (f *fakeAS) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if f.refuse(w, "invalid_request") {
		return
	}
	_ = r.ParseForm()
	f.mu.Lock()
	f.revoked[r.PostForm.Get("token")] = true
	f.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func s256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.MarshalWrite(w, v)
}

func writeTokenError(w http.ResponseWriter, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.MarshalWrite(w, map[string]string{"error": code})
}

// useMockKeyring routes go-keyring through its in-memory mock for the
// test, and gives it a HOME and XDG cache directory of its own, so the
// session store's fallback file is never the developer's.
func useMockKeyring(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	keyring.MockInit()
}

// browserOpener returns a LoginOptions.Open that plays the browser: it
// follows the authorize URL through the 302 to the loopback callback.
func browserOpener(t *testing.T) func(string) error {
	t.Helper()
	return func(authURL string) error {
		go func() {
			resp, err := http.Get(authURL) //nolint:gosec // test loopback
			if err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
}

func TestLoginAndRefreshRotation(t *testing.T) {
	useMockKeyring(t)
	as := newFakeAS(t)
	store := NewStore(as.origin())

	// Bind an ephemeral loopback port for the test instead of 8976/8765.
	orig := loopbackPorts
	loopbackPorts = []int{0}
	defer func() { loopbackPorts = orig }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sess, err := Login(ctx, as.origin(), store, LoginOptions{Open: browserOpener(t), Out: io.Discard})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if sess.AccessToken != "access-1" || sess.RefreshToken != "refresh-1" {
		t.Fatalf("unexpected initial tokens: %+v", sess)
	}
	if !strings.Contains(sess.Scope, "console:read") {
		t.Fatalf("scope not captured: %q", sess.Scope)
	}

	// Force the access token to be expired so the next use refreshes.
	expired := *sess
	expired.Expiry = time.Now().Add(-time.Minute)
	if err := store.Save(expired); err != nil {
		t.Fatalf("Save: %v", err)
	}

	authz := Resolve(Credentials{}, store)
	if authz.Mode() != ModeOAuth {
		t.Fatalf("expected OAuth mode, got %v", authz.Mode())
	}
	tok, err := authz.BearerToken(ctx)
	if err != nil {
		t.Fatalf("BearerToken (refresh): %v", err)
	}
	if tok != "access-2" {
		t.Fatalf("expected refreshed access token access-2, got %q", tok)
	}

	// The rotated refresh token must have been persisted.
	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.RefreshToken != "refresh-2" {
		t.Fatalf("rotated refresh token not persisted: %q", reloaded.RefreshToken)
	}
	if reloaded.AccessToken != "access-2" {
		t.Fatalf("refreshed access token not persisted: %q", reloaded.AccessToken)
	}
}

func TestResolvePrecedence(t *testing.T) {
	useMockKeyring(t)
	store := NewStore("https://example.test")

	// Explicit API key wins even when an OAuth session exists.
	if err := store.Save(Session{Issuer: "https://example.test", TokenURL: "https://example.test/oauth/token", RefreshToken: "rt", AccessToken: "at", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := Resolve(Credentials{APIKey: "k", APIKeyExplicit: true}, store).Mode(); got != ModeAPIKey {
		t.Fatalf("explicit key should win, got %v", got)
	}
	// No explicit key → OAuth session used.
	if got := Resolve(Credentials{APIKey: "k"}, store).Mode(); got != ModeOAuth {
		t.Fatalf("oauth session should be used, got %v", got)
	}
	// No session → config-file key used.
	emptyStore := NewStore("https://empty.test")
	if got := Resolve(Credentials{APIKey: "k"}, emptyStore).Mode(); got != ModeAPIKey {
		t.Fatalf("file key should be used, got %v", got)
	}
	// Nothing → none.
	if got := Resolve(Credentials{}, emptyStore).Mode(); got != ModeNone {
		t.Fatalf("expected none, got %v", got)
	}
}

func TestAccessTokenExpiry(t *testing.T) {
	useMockKeyring(t)
	store := NewStore("https://exp.test")
	exp := time.Now().Add(42 * time.Minute).Round(time.Second)
	if err := store.Save(Session{
		Issuer: "https://exp.test", TokenURL: "https://exp.test/oauth/token",
		RefreshToken: "rt", AccessToken: "at", Expiry: exp,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := Resolve(Credentials{}, store).AccessTokenExpiry(); !got.Equal(exp) {
		t.Fatalf("oauth AccessTokenExpiry = %v, want %v", got, exp)
	}
	if got := APIKeyAuthorizer("k").AccessTokenExpiry(); !got.IsZero() {
		t.Fatalf("api-key AccessTokenExpiry should be zero, got %v", got)
	}
}

func TestApiKeyAuthorizeHeader(t *testing.T) {
	a := apiKeyAuthorizer{key: "secret-key"}
	req, _ := http.NewRequest(http.MethodGet, "https://example.test", nil)
	if err := a.Authorize(context.Background(), req); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer secret-key" {
		t.Fatalf("unexpected header %q", got)
	}
}

func TestStoreFileFallback(t *testing.T) {
	// Force keychain failure so Save/Load use the 0600 file fallback.
	keyring.MockInitWithError(fmt.Errorf("no keychain"))
	t.Cleanup(func() { keyring.MockInit() })
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CACHE_HOME", tmp)

	store := NewStore("http://localhost:4000")
	sess := Session{Issuer: "http://localhost:4000", TokenURL: "http://localhost:4000/oauth/token", RefreshToken: "rt-file", AccessToken: "at-file", Expiry: time.Now().Add(time.Hour)}
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save (file): %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load (file): %v", err)
	}
	if got.RefreshToken != "rt-file" {
		t.Fatalf("file round-trip failed: %+v", got)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := store.Load(); err != ErrNoSession {
		t.Fatalf("expected ErrNoSession after clear, got %v", err)
	}
}

func TestDiscoveryValidate(t *testing.T) {
	good := &Discovery{
		Issuer:                        "http://localhost:4000",
		AuthorizationEndpoint:         "http://localhost:4000/oauth/authorize",
		TokenEndpoint:                 "http://localhost:4000/oauth/token",
		CodeChallengeMethodsSupported: []string{"S256"},
	}
	if err := good.Validate("http://localhost:4000"); err != nil {
		t.Fatalf("valid discovery rejected: %v", err)
	}
	noS256 := *good
	noS256.CodeChallengeMethodsSupported = []string{"plain"}
	if err := noS256.Validate("http://localhost:4000"); err == nil {
		t.Fatal("expected rejection without S256")
	}
	wrongIssuer := *good
	wrongIssuer.Issuer = "http://evil.test"
	if err := wrongIssuer.Validate("http://localhost:4000"); err == nil {
		t.Fatal("expected issuer-mismatch rejection")
	}
	// A matching issuer but a token endpoint on a foreign origin (which would
	// exfiltrate the code_verifier / refresh token) must be rejected.
	foreignToken := *good
	foreignToken.TokenEndpoint = "http://evil.test/oauth/token"
	if err := foreignToken.Validate("http://localhost:4000"); err == nil {
		t.Fatal("expected cross-origin token_endpoint rejection")
	}
}

func TestForceRefreshRotatesValidToken(t *testing.T) {
	useMockKeyring(t)
	as := newFakeAS(t)
	store := NewStore(as.origin())
	orig := loopbackPorts
	loopbackPorts = []int{0}
	defer func() { loopbackPorts = orig }()

	ctx := context.Background()
	if _, err := Login(ctx, as.origin(), store, LoginOptions{Open: browserOpener(t), Out: io.Discard}); err != nil {
		t.Fatalf("Login: %v", err)
	}
	azr := Resolve(Credentials{}, store)

	// The freshly-minted token is still valid; ForceRefresh must rotate it
	// anyway (this is what breaks the WS token_expired clock-skew loop).
	tok1, err := azr.BearerToken(ctx)
	if err != nil || tok1 != "access-1" {
		t.Fatalf("BearerToken before = %q, %v", tok1, err)
	}
	if err := azr.ForceRefresh(ctx); err != nil {
		t.Fatalf("ForceRefresh: %v", err)
	}
	tok2, err := azr.BearerToken(ctx)
	if err != nil || tok2 != "access-2" {
		t.Fatalf("BearerToken after ForceRefresh = %q, %v (want access-2)", tok2, err)
	}
	reloaded, err := store.Load()
	if err != nil || reloaded.RefreshToken != "refresh-2" {
		t.Fatalf("rotated refresh token not persisted: %+v (%v)", reloaded, err)
	}
}

func TestRetryTransportRefreshesOn401(t *testing.T) {
	useMockKeyring(t)
	as := newFakeAS(t)
	store := NewStore(as.origin())
	orig := loopbackPorts
	loopbackPorts = []int{0}
	defer func() { loopbackPorts = orig }()

	ctx := context.Background()
	if _, err := Login(ctx, as.origin(), store, LoginOptions{Open: browserOpener(t), Out: io.Discard}); err != nil {
		t.Fatalf("Login: %v", err)
	}
	SetDefault(Resolve(Credentials{}, store))
	t.Cleanup(func() { SetDefault(nil) })

	var mu sync.Mutex
	var seen []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
		// The first (stale) token is rejected; the refreshed one is accepted.
		if r.Header.Get("Authorization") == "Bearer access-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(api.Close)

	client := &http.Client{Transport: NewRetryTransport(nil)}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, api.URL, nil)
	if err := Default().Authorize(ctx, req); err != nil {
		t.Fatalf("authorize: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after refresh-and-retry, got %d", resp.StatusCode)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 || seen[0] != "Bearer access-1" || seen[1] != "Bearer access-2" {
		t.Fatalf("expected one 401 with access-1 then a retry with access-2, got %v", seen)
	}
}

// recordingServer answers every request with status, unless the request
// carries accept as its Authorization value, and records each value seen.
func recordingServer(t *testing.T, status int, accept string) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		mu.Lock()
		seen = append(seen, h)
		mu.Unlock()
		if accept != "" && h == accept {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

// A 401 from another service, to a request carrying that service's own
// credential, is not the session's to retry: no refresh is spent and the
// Truestamp access token never goes to that service. This is the upgrade
// check sending GITHUB_TOKEN to GitHub while an OAuth session is signed in.
// Mutant killed: retrying any 401 that carries an Authorization header.
func TestRetryTransportLeavesAForeignCredentialAlone(t *testing.T) {
	as := newFakeAS(t)
	store := loginForTest(t, as)
	SetDefault(Resolve(Credentials{}, store))
	t.Cleanup(func() { SetDefault(nil) })
	ctx := context.Background()

	foreign, seen := recordingServer(t, http.StatusUnauthorized, "")
	client := &http.Client{Transport: NewRetryTransport(nil)}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, foreign.URL, nil)
	req.Header.Set("Authorization", "Bearer foreign-token")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want the service's own 401", resp.StatusCode)
	}
	if got := seen(); len(got) != 1 || got[0] != "Bearer foreign-token" {
		t.Errorf("the other service saw %q, want exactly one request carrying its own credential", got)
	}
	if sess, err := store.Load(); err != nil || sess.RefreshToken != "refresh-1" {
		t.Errorf("stored session = %+v (%v), want refresh-1: a refresh was spent on another service's 401", sess, err)
	}
}

// A request sent with an access token that a concurrent refresh has since
// replaced is still one the session authenticated, so its 401 is retried.
// Mutant killed: recognizing only the current access token.
func TestRetryTransportRetriesARequestSentBeforeAConcurrentRefresh(t *testing.T) {
	as := newFakeAS(t)
	store := loginForTest(t, as)
	SetDefault(Resolve(Credentials{}, store))
	t.Cleanup(func() { SetDefault(nil) })
	ctx := context.Background()

	api, seen := recordingServer(t, http.StatusUnauthorized, "Bearer access-3")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, api.URL, nil)
	if err := Default().Authorize(ctx, req); err != nil {
		t.Fatalf("authorize: %v", err)
	}
	// Another call site refreshes while req is in flight.
	if err := Default().ForceRefresh(ctx); err != nil {
		t.Fatalf("ForceRefresh: %v", err)
	}

	client := &http.Client{Transport: NewRetryTransport(nil)}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 after the retry", resp.StatusCode)
	}
	if got := seen(); len(got) != 2 || got[0] != "Bearer access-1" || got[1] != "Bearer access-3" {
		t.Errorf("seen %q, want access-1 rejected and then a retry with access-3", got)
	}
}

// A request authorized by a second authorizer over the same stored session,
// as the identity probe of `auth login` is, carries the access token the
// process default holds too, so its 401 is retried.
// Mutant killed: recognizing only the values this authorizer set itself.
func TestRetryTransportRetriesARequestOfTheSameSession(t *testing.T) {
	as := newFakeAS(t)
	store := loginForTest(t, as)
	SetDefault(Resolve(Credentials{}, store))
	t.Cleanup(func() { SetDefault(nil) })
	ctx := context.Background()

	api, seen := recordingServer(t, http.StatusUnauthorized, "Bearer access-2")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, api.URL, nil)
	if err := Resolve(Credentials{}, store).Authorize(ctx, req); err != nil {
		t.Fatalf("authorize: %v", err)
	}

	client := &http.Client{Transport: NewRetryTransport(nil)}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 after the retry", resp.StatusCode)
	}
	if got := seen(); len(got) != 2 || got[0] != "Bearer access-1" || got[1] != "Bearer access-2" {
		t.Errorf("seen %q, want access-1 rejected and then a retry with access-2", got)
	}
}
