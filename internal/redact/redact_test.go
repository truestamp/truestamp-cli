// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package redact

import (
	"errors"
	"strings"
	"testing"
)

func TestStringAPIKey(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple query string",
			in:   `dial ws://x/y?api_key=truestamp_abc&vsn=2.0.0`,
			want: `dial ws://x/y?api_key=REDACTED&vsn=2.0.0`,
		},
		{
			name: "quoted url in error",
			in:   `Get "http://x/y?api_key=truestamp_secret123&vsn=2.0.0"`,
			want: `Get "http://x/y?api_key=REDACTED&vsn=2.0.0"`,
		},
		{
			name: "trailing key (end of string)",
			in:   `?api_key=truestamp_alone`,
			want: `?api_key=REDACTED`,
		},
		{
			name: "multi-line error chain",
			in:   "websocket.Dial: bad status\nupgrade URL: wss://x?api_key=hunter2&vsn=2.0.0",
			want: "websocket.Dial: bad status\nupgrade URL: wss://x?api_key=REDACTED&vsn=2.0.0",
		},
		{
			name: "no api_key present",
			in:   "connection refused",
			want: "connection refused",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := String(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStringBearer(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "header form",
			in:   `Authorization: Bearer truestamp_VERY_SECRET`,
			want: `Authorization: Bearer REDACTED`,
		},
		{
			name: "lowercase bearer (case-insensitive)",
			in:   `auth: bearer hunter2`,
			want: `auth: Bearer REDACTED`,
		},
		{
			name: "mixed case",
			in:   `Authorization: BEARER xyz123`,
			want: `Authorization: Bearer REDACTED`,
		},
		{
			name: "bearer in quoted error",
			in:   `request failed: "Authorization: Bearer truestamp_xyz" rejected`,
			want: `request failed: "Authorization: Bearer REDACTED" rejected`,
		},
		{
			name: "no bearer present",
			in:   "Authorization header missing",
			want: "Authorization header missing",
		},
		{
			name: "bearer-prefixed word should still match (defense in depth)",
			in:   "Bearer abc",
			want: "Bearer REDACTED",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := String(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStringOAuth(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "access_token query param",
			in:   `callback ?access_token=jwt.abc.def&state=x`,
			want: `callback ?access_token=REDACTED&state=x`,
		},
		{
			name: "refresh_token value redacted, grant_type label kept",
			in:   `grant_type=refresh_token&refresh_token=rt_secret&client_id=c`,
			want: `grant_type=refresh_token&refresh_token=REDACTED&client_id=c`,
		},
		{
			name: "code_verifier and code params",
			in:   `code_verifier=abc123&code=xyz`,
			want: `code_verifier=REDACTED&code=REDACTED`,
		},
		{
			name: "json token response body",
			in:   `{"access_token":"jwt","refresh_token":"rt","token_type":"Bearer"}`,
			want: `{"access_token":"REDACTED","refresh_token":"REDACTED","token_type":"Bearer"}`,
		},
		{
			name: "invalid_grant error body is not over-redacted",
			in:   `{"error":"invalid_grant"}`,
			want: `{"error":"invalid_grant"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := String(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStringBoth(t *testing.T) {
	in := `dial wss://x?api_key=truestamp_a&vsn=2.0.0; tried Authorization: Bearer truestamp_b`
	got := String(in)
	if strings.Contains(got, "truestamp_a") || strings.Contains(got, "truestamp_b") {
		t.Fatalf("secret leaked: %q", got)
	}
	if !strings.Contains(got, "api_key=REDACTED") {
		t.Errorf("api_key not redacted: %q", got)
	}
	if !strings.Contains(got, "Bearer REDACTED") {
		t.Errorf("Bearer not redacted: %q", got)
	}
}

func TestErrorHelper(t *testing.T) {
	if got := Error(nil); got != "" {
		t.Fatalf("Error(nil) = %q, want empty", got)
	}
	err := errors.New("dial wss://x?api_key=secret_token")
	got := Error(err)
	if strings.Contains(got, "secret_token") {
		t.Fatalf("Error() leaked secret: %q", got)
	}
	if !strings.Contains(got, "api_key=REDACTED") {
		t.Errorf("Error() did not redact: %q", got)
	}
}

// FuzzRedact asserts the invariants we care about under arbitrary input:
//
//  1. No panic.
//  2. Idempotent — String(String(s)) == String(s); a second pass must
//     find nothing left to redact.
//  3. Self-checking against the regexes — running them over the output
//     must find no matches other than the REDACTED sentinel.
//
// The seed corpus exercises the patterns that have hit production paths
// so the regression check is meaningful, not just random fuzzing noise.
func FuzzRedact(f *testing.F) {
	seeds := []string{
		"",
		"plain text",
		"api_key=a",
		"api_key=a&vsn=1",
		`?api_key=truestamp_abc"`,
		"Bearer abc",
		"Authorization: Bearer xyz",
		"api_key=a&vsn=1\nAuthorization: Bearer y",
		"api_key=\x00\x01\x02&vsn=1",
		"api_key=" + strings.Repeat("A", 1000),
		"access_token=jwt&refresh_token=rt&code=c&code_verifier=v",
		`{"access_token":"x","refresh_token":"y"}`,
		"grant_type=refresh_token&refresh_token=rt",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := String(s)
		if out2 := String(out); out2 != out {
			t.Fatalf("not idempotent: first=%q second=%q", out, out2)
		}
		// Every regex match remaining in the output must be exactly the
		// sentinel form. Anything else means redaction missed a token.
		for _, m := range apiKeyRe.FindAllString(out, -1) {
			if m != "api_key="+REDACTED {
				t.Fatalf("api_key match not redacted: %q (in %q)", m, out)
			}
		}
		for _, m := range bearerRe.FindAllString(out, -1) {
			// bearerRe is case-insensitive but ReplaceAllString
			// substitutes the literal "Bearer REDACTED" — so any
			// remaining match should equal that exactly.
			if m != "Bearer "+REDACTED {
				t.Fatalf("Bearer match not redacted: %q (in %q)", m, out)
			}
		}
		for _, m := range oauthQueryRe.FindAllString(out, -1) {
			if !strings.HasSuffix(m, "="+REDACTED) {
				t.Fatalf("oauth query match not redacted: %q (in %q)", m, out)
			}
		}
		for _, m := range oauthJSONRe.FindAllString(out, -1) {
			if !strings.Contains(m, REDACTED) {
				t.Fatalf("oauth json match not redacted: %q (in %q)", m, out)
			}
		}
	})
}
