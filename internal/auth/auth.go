// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package auth is the CLI's authentication core. It implements an
// OAuth 2.1 client (browser-based loopback Authorization Code + PKCE with
// rotating refresh tokens) and unifies it with the legacy long-lived API
// key behind a single [Authorizer] abstraction.
//
// Authentication is OAuth-first, API-key-second. The resolution order
// (see [Resolve]) is:
//
//  1. an explicitly-provided API key (--api-key flag or TRUESTAMP_API_KEY
//     env) — wins outright so CI/headless callers are deterministic;
//  2. a stored OAuth session (access token, auto-refreshed via the
//     rotating refresh token);
//  3. an API key from the config file;
//  4. otherwise unauthenticated.
//
// Both an OAuth access token and an API key are presented to the server as
// `Authorization: Bearer <value>` on the JSON API; the server accepts
// either. The console WebSocket differs: OAuth uses the Bearer request
// header on the upgrade, while an API key uses the `?api_key=` query
// param — callers branch on [Authorizer.Mode] for that.
package auth

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

// ClientID is the fixed, public OAuth client_id for the Truestamp CLI. It
// is a public client (PKCE is the per-flow secret), so embedding it in the
// binary is standard and safe — the same convention gh/gcloud/stripe use.
// The server seeds this exact id via an idempotent boot-upsert across
// dev/staging/prod, so a single constant is correct for every environment;
// only the issuer/endpoints vary, and those come from discovery.
const ClientID = "019ef661-6737-71ec-abd0-ac8f4684ce45"

// Scopes is the set requested at login. It deliberately excludes the
// mcp:* scopes (the CLI does not call /mcp); requesting an mcp scope is
// rejected at /oauth/authorize. api:* covers JSON:API + GraphQL,
// console:* covers the WebSocket console.
var Scopes = []string{"api:read", "api:write", "console:read", "console:write"}

// callbackPath is the loopback redirect path. The full redirect_uri
// (`http://127.0.0.1:<port>/callback`) must byte-for-byte match a
// redirect_uri registered on the server-side client, because the
// authorization server does exact matching (no RFC 8252 port flexibility).
const callbackPath = "/callback"

// loopbackPorts are the fixed ports the CLI binds for the callback, in
// preference order. Both are pre-registered on the server client. The CLI
// tries them in turn and errors clearly if both are busy rather than
// silently using an unregistered ephemeral port that the server would
// reject.
var loopbackPorts = []int{8976, 8765}

// ErrNoCredentials is returned by a no-credentials [Authorizer] when a
// bearer token is requested (e.g. by the WebSocket dialer, which cannot
// proceed unauthenticated).
var ErrNoCredentials = errors.New("not authenticated — run `truestamp auth login` (or set TRUESTAMP_API_KEY)")

// ErrSessionExpired is returned in OAuth mode when the refresh token is
// expired, revoked, or reused (the server's `invalid_grant`). The session
// is permanently dead; the user must re-authenticate.
var ErrSessionExpired = errors.New("OAuth session expired or revoked — run `truestamp auth login`")

// Mode identifies which credential an [Authorizer] carries.
type Mode int

const (
	// ModeNone means no credential is configured.
	ModeNone Mode = iota
	// ModeAPIKey means a long-lived API key is in use.
	ModeAPIKey
	// ModeOAuth means a short-lived, auto-refreshing OAuth access token
	// is in use.
	ModeOAuth
)

// String renders the mode for status output.
func (m Mode) String() string {
	switch m {
	case ModeAPIKey:
		return "api_key"
	case ModeOAuth:
		return "oauth"
	default:
		return "none"
	}
}

// Authorizer stamps outbound requests with the active credential and
// reports which credential is in use. Obtain one via [Resolve]. All
// methods are safe for concurrent use.
type Authorizer interface {
	// Authorize adds the Authorization header to req. In OAuth mode it
	// transparently refreshes an expired access token first (returning a
	// non-nil error if the session is dead). In no-credentials mode it is
	// a no-op returning nil so genuinely-public requests still go out.
	Authorize(ctx context.Context, req *http.Request) error

	// BearerToken returns the raw credential string — a fresh OAuth access
	// token (refreshed if needed) or the API key — for carriers that
	// cannot take an *http.Request (the WebSocket dialer). Returns
	// [ErrNoCredentials] in no-credentials mode.
	BearerToken(ctx context.Context) (string, error)

	// ForceRefresh proactively discards any cached credential and obtains a
	// fresh one. In OAuth mode it runs the refresh grant unconditionally
	// (used reactively when the server rejects a token the client still
	// believes is valid — a 401 on the HTTP API, or a token_expired push on
	// the WebSocket — to break the otherwise-possible "reuse the stale
	// token" loop). It is a no-op for the API-key and no-credentials modes.
	// Returns [ErrSessionExpired] when the refresh token is dead.
	ForceRefresh(ctx context.Context) error

	// AccessTokenExpiry reports when the current OAuth access token expires,
	// so a long-lived carrier (the console WebSocket) can proactively
	// refresh in-band before the server force-disconnects. Returns the zero
	// time for the API-key and no-credentials modes (no expiry).
	AccessTokenExpiry() time.Time

	// Mode reports the active credential type.
	Mode() Mode
}

// defaultAuthorizer is the process-wide Authorizer set once at startup by
// cmd after config load, mirroring httpclient's package-global pattern so
// the many existing call sites can authorize without threading an
// Authorizer through every signature.
var (
	defaultMu         sync.RWMutex
	defaultAuthorizer Authorizer = noneAuthorizer{}
)

// SetDefault installs the process-wide Authorizer. Pass nil to reset to
// the no-credentials authorizer.
func SetDefault(a Authorizer) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if a == nil {
		a = noneAuthorizer{}
	}
	defaultAuthorizer = a
}

// Default returns the process-wide Authorizer (never nil).
func Default() Authorizer {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultAuthorizer
}

// AuthorizeRequest stamps req using the process-wide [Default] Authorizer.
// This is the minimal-churn entry point for the existing HTTP call sites.
func AuthorizeRequest(ctx context.Context, req *http.Request) error {
	return Default().Authorize(ctx, req)
}
