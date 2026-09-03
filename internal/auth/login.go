// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cli/browser"
	"golang.org/x/oauth2"
)

// LoginOptions tunes the interactive login flow. The zero value is valid:
// it opens the system browser and writes prompts to os.Stderr-equivalent
// nil (suppressed). cmd populates Out and may override Open for tests.
type LoginOptions struct {
	// Open opens a URL in the user's browser. Defaults to browser.OpenURL.
	Open func(string) error
	// Out receives the human-facing "open this URL" guidance. Defaults to
	// io.Discard.
	Out io.Writer
}

// callbackResult carries the loopback handler's outcome to Login.
type callbackResult struct {
	code string
	err  error
}

// Login runs the browser-based loopback Authorization Code + PKCE flow
// against baseOrigin, persists the resulting session via store, and returns
// it. ctx bounds the whole flow (caller should pass a generous deadline,
// the user has to sign in and consent in a browser).
func Login(ctx context.Context, baseOrigin string, store Store, opts LoginOptions) (*Session, error) {
	if opts.Open == nil {
		opts.Open = browser.OpenURL
	}
	if opts.Out == nil {
		opts.Out = io.Discard
	}

	disc, err := Fetch(ctx, baseOrigin)
	if err != nil {
		return nil, err
	}
	if err := disc.Validate(baseOrigin); err != nil {
		return nil, err
	}

	ln, redirectURI, err := listenLoopback()
	if err != nil {
		return nil, err
	}
	defer ln.Close()

	verifier := oauth2.GenerateVerifier()
	state, err := randString(32)
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	conf := &oauth2.Config{
		ClientID:    ClientID,
		RedirectURL: redirectURI,
		Scopes:      Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   disc.AuthorizationEndpoint,
			TokenURL:  disc.TokenEndpoint,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
	authURL := conf.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	resCh := make(chan callbackResult, 1)
	srv := &http.Server{
		Handler:           callbackHandler(state, resCh),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	fmt.Fprintln(opts.Out, "Opening your browser to sign in:")
	fmt.Fprintln(opts.Out, "  "+authURL)
	if err := opts.Open(authURL); err != nil {
		fmt.Fprintln(opts.Out, "Could not open a browser automatically, visit the URL above to continue.")
	}

	var code string
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("login timed out or was cancelled: %w", ctx.Err())
	case r := <-resCh:
		if r.err != nil {
			return nil, r.err
		}
		code = r.code
	}

	tok, err := conf.Exchange(tokenContext(ctx), code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}

	sess := sessionFromToken(disc, tok)
	if err := store.Save(sess); err != nil {
		return nil, fmt.Errorf("saving session: %w", err)
	}
	return &sess, nil
}

// Logout best-effort revokes the stored refresh token (RFC 7009) and clears
// the local session. Revocation failures are non-fatal, the local session
// is cleared regardless, and access tokens are short-lived stateless JWTs.
func Logout(ctx context.Context, store Store) (revoked bool, err error) {
	sess, loadErr := store.Load()
	if loadErr == nil && sess.RevocationURL != "" && sess.RefreshToken != "" {
		if rerr := revokeToken(ctx, sess.RevocationURL, sess.RefreshToken); rerr == nil {
			revoked = true
		}
	}
	return revoked, store.Clear()
}

// revokeToken posts an RFC 7009 revocation for a refresh token.
func revokeToken(ctx context.Context, revocationURL, refreshToken string) error {
	form := url.Values{
		"token":           {refreshToken},
		"token_type_hint": {"refresh_token"},
		"client_id":       {ClientID},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revocationURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("revocation returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// listenLoopback binds the first available fixed loopback port and returns
// the listener plus the exact redirect_uri to advertise (which must match a
// server-registered redirect_uri byte-for-byte).
func listenLoopback() (net.Listener, string, error) {
	var lastErr error
	for _, port := range loopbackPorts {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			// Derive the redirect_uri from the actually-bound port. For the
			// fixed ports this equals the configured port; tests inject port
			// 0 and get the real ephemeral port here.
			actual := ln.Addr().(*net.TCPAddr).Port
			return ln, fmt.Sprintf("http://127.0.0.1:%d%s", actual, callbackPath), nil
		}
		lastErr = err
	}
	return nil, "", fmt.Errorf("could not bind a loopback callback port (tried %v): %w, close whatever is using them and retry", loopbackPorts, lastErr)
}

// callbackHandler serves the single OAuth redirect, validates state, and
// reports the code (or error) exactly once.
func callbackHandler(state string, resCh chan<- callbackResult) http.Handler {
	var once sync.Once
	send := func(r callbackResult) { once.Do(func() { resCh <- r }) }
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != callbackPath {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		// A request that doesn't carry THIS flow's crypto-random state is
		// not our redirect, it's a stray probe (a co-resident process, a
		// browser prefetch) hitting the fixed loopback port. Reject it with
		// a 400 but DO NOT resolve the flow, so it can't abort a legitimate
		// login. Only a matching-state request (success or an explicit
		// user denial) is terminal; the ctx deadline bounds the wait.
		if q.Get("state") != state {
			http.Error(w, "unrecognized or missing state parameter", http.StatusBadRequest)
			return
		}
		if e := q.Get("error"); e != "" {
			desc := q.Get("error_description")
			writeResultPage(w, false, "Authorization was denied.")
			send(callbackResult{err: fmt.Errorf("authorization denied: %s %s", e, desc)})
			return
		}
		code := q.Get("code")
		if code == "" {
			writeResultPage(w, false, "No authorization code was returned.")
			send(callbackResult{err: errors.New("no authorization code in callback")})
			return
		}
		writeResultPage(w, true, "You're signed in to the Truestamp CLI. You can close this tab and return to your terminal.")
		send(callbackResult{code: code})
	})
}

// writeResultPage renders a minimal browser landing page.
func writeResultPage(w http.ResponseWriter, ok bool, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	status := "Success"
	color := "#16a34a"
	if !ok {
		status = "Login failed"
		color = "#dc2626"
	}
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>Truestamp CLI</title>
<style>body{font-family:system-ui,sans-serif;background:#0b0b0c;color:#e5e5e5;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}
.card{max-width:30rem;padding:2rem;text-align:center}h1{color:%s;margin:0 0 .5rem}p{color:#a3a3a3;line-height:1.5}</style></head>
<body><div class="card"><h1>%s</h1><p>%s</p></div></body></html>`, color, status, msg)
}

// sessionFromToken assembles a Session from discovery + an exchanged token.
func sessionFromToken(disc *Discovery, tok *oauth2.Token) Session {
	scope, _ := tok.Extra("scope").(string)
	return Session{
		Issuer:        disc.Issuer,
		AuthURL:       disc.AuthorizationEndpoint,
		TokenURL:      disc.TokenEndpoint,
		RevocationURL: disc.RevocationEndpoint,
		AccessToken:   tok.AccessToken,
		RefreshToken:  tok.RefreshToken,
		TokenType:     tok.TokenType,
		Scope:         scope,
		Expiry:        tok.Expiry,
	}
}

// randString returns n cryptographically-random bytes, base64url-encoded.
func randString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
