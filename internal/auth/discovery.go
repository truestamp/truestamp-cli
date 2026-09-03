// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// discoveryPath is the RFC 8414 authorization-server metadata path.
const discoveryPath = "/.well-known/oauth-authorization-server"

// Discovery is the subset of RFC 8414 authorization-server metadata the
// CLI consumes. Endpoints are read from here (never hardcoded) so the
// same binary works against dev/staging/prod, which differ only by origin.
type Discovery struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// Fetch retrieves and parses the authorization-server metadata for the
// given base origin (scheme + host, e.g. "http://localhost:4000"). It
// reuses the shared HTTP client (timeout, User-Agent, response-size cap).
func Fetch(ctx context.Context, baseOrigin string) (*Discovery, error) {
	origin := canonicalOrigin(baseOrigin)
	if origin == "" {
		return nil, fmt.Errorf("invalid base origin %q", baseOrigin)
	}
	body, err := httpclient.GetJSONCtx(ctx, origin+discoveryPath)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", discoveryPath, err)
	}
	var d Discovery
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, fmt.Errorf("parsing authorization-server metadata: %w", err)
	}
	return &d, nil
}

// Validate checks that the discovery document is usable for the loopback
// PKCE flow against the expected origin: it must carry the authorization
// and token endpoints, advertise S256 PKCE, and its issuer must match the
// configured origin (a guard against a misconfigured base_url silently
// authenticating against the wrong server).
func (d *Discovery) Validate(baseOrigin string) error {
	if d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" {
		return fmt.Errorf("authorization server metadata is missing the authorization or token endpoint, OAuth may not be enabled on this server")
	}
	if !contains(d.CodeChallengeMethodsSupported, "S256") {
		return fmt.Errorf("authorization server does not advertise S256 PKCE (got %v); refusing to fall back to a weaker method", d.CodeChallengeMethodsSupported)
	}
	origin := canonicalOrigin(baseOrigin)
	if d.Issuer != "" && canonicalOrigin(d.Issuer) != origin {
		return fmt.Errorf("issuer %q does not match the configured base URL %q; check base_url", d.Issuer, baseOrigin)
	}
	// Pin every advertised endpoint to the configured origin. The token,
	// authorize, and revocation endpoints carry the PKCE code_verifier and
	// the rotating refresh token; a metadata document that scattered them
	// to a foreign host (while keeping a matching issuer) would exfiltrate
	// those secrets. RFC 8414 recommends this same-origin pinning.
	for _, ep := range []struct{ name, raw string }{
		{"authorization_endpoint", d.AuthorizationEndpoint},
		{"token_endpoint", d.TokenEndpoint},
		{"revocation_endpoint", d.RevocationEndpoint},
	} {
		if ep.raw == "" {
			continue
		}
		if canonicalOrigin(ep.raw) != origin {
			return fmt.Errorf("%s %q is not on the configured origin %q; refusing", ep.name, ep.raw, origin)
		}
	}
	return nil
}

// canonicalOrigin parses a URL and returns just its scheme+host, lowercased
// scheme, with no trailing slash. Returns "" if the input cannot be parsed
// into a scheme+host. Used both to derive the discovery URL and to key the
// per-environment token store.
func canonicalOrigin(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return (&url.URL{Scheme: strings.ToLower(u.Scheme), Host: u.Host}).String()
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
