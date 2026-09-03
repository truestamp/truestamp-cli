// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package auth

import (
	"encoding/json"
	"testing"
)

// FuzzDiscoveryValidate hardens the OAuth authorization-server metadata
// path. The discovery document is network input, fetched from the
// configured base origin, so a malicious or MITM'd server could return
// arbitrary JSON. Validate and canonicalOrigin do URL/string work over the
// parsed issuer + endpoints; they must never panic, for any parsed contents
// and against any origin the CLI may be configured with.
//
// Seeds pin the production-shaped doc plus the deliberately-rejected
// shapes (cross-origin endpoint, plain-only PKCE, non-URL issuer) so the
// replay run keeps exercising the Validate branches.
func FuzzDiscoveryValidate(f *testing.F) {
	f.Add([]byte(`{"issuer":"http://localhost:4000","authorization_endpoint":"http://localhost:4000/oauth/authorize","token_endpoint":"http://localhost:4000/oauth/token","revocation_endpoint":"http://localhost:4000/oauth/revoke","code_challenge_methods_supported":["S256"]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"issuer":"http://localhost:4000","authorization_endpoint":"http://localhost:4000/a","token_endpoint":"http://evil.test/t","code_challenge_methods_supported":["S256"]}`))
	f.Add([]byte(`{"issuer":"::not a url::","token_endpoint":"","code_challenge_methods_supported":["plain"]}`))
	f.Add([]byte(`{"code_challenge_methods_supported":[""],"authorization_endpoint":"   ","token_endpoint":"\t"}`))

	origins := []string{"http://localhost:4000", "https://www.truestamp.com", "", "not a url", "http://"}

	f.Fuzz(func(t *testing.T, data []byte) {
		var d Discovery
		if json.Unmarshal(data, &d) != nil {
			return // Fetch surfaces the parse error; there's nothing to validate
		}
		// Must never panic for any parsed metadata document, for any origin.
		for _, origin := range origins {
			_ = d.Validate(origin)
		}
		_ = canonicalOrigin(d.Issuer)
		_ = canonicalOrigin(d.TokenEndpoint)
		_ = canonicalOrigin(d.AuthorizationEndpoint)
	})
}

// FuzzSessionDecode hardens the token store's decode path. The persisted
// session, a keychain value or the 0600 fallback file, is read back with
// encoding/json into both the per-origin map (the file store) and a bare
// Session. Garbage or a tampered file must yield a clean error, never a
// panic.
func FuzzSessionDecode(f *testing.F) {
	f.Add([]byte(`{"http://localhost:4000":{"access_token":"a","refresh_token":"r","token_type":"Bearer","expiry":"2026-01-01T00:00:00Z"}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`{"x":{"expiry":"garbage"}}`))
	f.Add([]byte(`[1,2,3]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var m map[string]Session
		_ = json.Unmarshal(data, &m)
		var s Session
		_ = json.Unmarshal(data, &s)
	})
}
