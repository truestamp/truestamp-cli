// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
)

func TestDescribeRateLimit(t *testing.T) {
	cases := []struct {
		name string
		err  *jsonapi.APIError
		want string
	}{
		{
			"plug refusal with header and limit",
			&jsonapi.APIError{Status: 429, RetryAfter: "37", RetryAfterMS: 36412, Limit: 120,
				Detail: "API rate limit exceeded. Slow down and retry shortly."},
			"rate limited (retry after 37s, limit 120): API rate limit exceeded. Slow down and retry shortly.",
		},
		{
			"action refusal, wait from meta, rounded up",
			&jsonapi.APIError{Status: 429, RetryAfterMS: 11204, Limit: 1000,
				Detail: "Rate limit exceeded. Retry after 12 seconds."},
			"rate limited (retry after 12s, limit 1000): Rate limit exceeded. Retry after 12 seconds.",
		},
		{
			"never admitted",
			&jsonapi.APIError{Status: 429, NeverAdmitted: true, Limit: 1000, Detail: "Rate limit exceeded."},
			"rate limited (this request is over the limit on its own and no wait will admit it, limit 1000): Rate limit exceeded.",
		},
		{
			"bare 429 from an intermediary",
			&jsonapi.APIError{Status: 429, Detail: "Too Many Requests"},
			"rate limited: Too Many Requests",
		},
		{
			"sub-second wait never renders as 0s",
			&jsonapi.APIError{Status: 429, RetryAfterMS: 400, Detail: "slow down"},
			"rate limited (retry after 1s): slow down",
		},
	}
	for _, tc := range cases {
		if got := describeRateLimit(tc.err); got != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", tc.name, got, tc.want)
		}
	}
}

func TestRenderAPIError_RateLimited(t *testing.T) {
	c := &cobra.Command{}
	err := &jsonapi.APIError{Status: 429, RetryAfter: "37", Limit: 120,
		Detail: "API rate limit exceeded. Slow down and retry shortly.", Sentinel: jsonapi.ErrRateLimited}
	got := renderAPIError(c, err, "item")
	if got == nil || got.Error() != "rate limited (retry after 37s, limit 120): API rate limit exceeded. Slow down and retry shortly." {
		t.Errorf("renderAPIError = %v", got)
	}
	// Without a wait named, the message still names the class and the
	// server's detail rather than falling back to "HTTP 429".
	bare := &jsonapi.APIError{Status: 429, Detail: "Too Many Requests", Sentinel: jsonapi.ErrRateLimited}
	if got := renderAPIError(c, bare, "item"); got == nil || got.Error() != "rate limited: Too Many Requests" {
		t.Errorf("bare 429 rendered as %v", got)
	}
}

// rateLimitingAPI answers every request with the contract's plug refusal
// and a Retry-After over the retry cap, so the probe surfaces the 429
// after exactly one attempt and the test never sleeps.
func rateLimitingAPI(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errors":[{"status":"429","code":"rate_limited","title":"Too Many Requests","detail":"API rate limit exceeded. Slow down and retry shortly.","meta":{"retry_after_ms":3600000,"limit":120}}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestCheckAuth_RateLimited(t *testing.T) {
	srv, calls := rateLimitingAPI(t)
	res, err := checkAuth(context.Background(), auth.APIKeyAuthorizer("k"), srv.URL+"/api/json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.rateLimited || res.ok || res.unauthorized || res.httpStatus != http.StatusTooManyRequests {
		t.Errorf("result = %+v, want rateLimited only", res)
	}
	for _, want := range []string{"retry after 3600s", "limit 120", "Slow down"} {
		if !strings.Contains(res.message, want) {
			t.Errorf("message %q should mention %q", res.message, want)
		}
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1: a wait over the cap is not slept", calls.Load())
	}
}

func TestFetchTeam_RateLimited(t *testing.T) {
	srv, _ := rateLimitingAPI(t)
	res, err := fetchTeam(context.Background(), auth.APIKeyAuthorizer("k"), srv.URL+"/api/json", "team_42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.rateLimited || res.found || res.httpStatus != http.StatusTooManyRequests {
		t.Errorf("result = %+v, want rateLimited and not found", res)
	}
	if !strings.Contains(res.message, "retry after 3600s") {
		t.Errorf("message %q should carry the wait", res.message)
	}
}

func TestFormatWait(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                        "1s",
		400 * time.Millisecond:   "1s",
		time.Second:              "1s",
		11204 * time.Millisecond: "12s",
		37 * time.Second:         "37s",
		time.Hour:                "3600s",
	} {
		if got := formatWait(d); got != want {
			t.Errorf("formatWait(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestRateLimitNotice_GatedOnTerminalAndSilent(t *testing.T) {
	orig := stderrIsTerminal
	t.Cleanup(func() { stderrIsTerminal = orig })

	cases := []struct {
		name     string
		terminal bool
		silent   bool
		want     bool
	}{
		{"terminal", true, false, true},
		{"terminal under --silent", true, true, false},
		{"stderr redirected", false, false, false},
		{"redirected and silent", false, true, false},
	}
	for _, tc := range cases {
		stderrIsTerminal = func() bool { return tc.terminal }
		var buf bytes.Buffer
		c := &cobra.Command{}
		c.SetErr(&buf)
		rateLimitNotice(c, tc.silent)(37*time.Second, 120)
		got := strings.Contains(buf.String(), "Rate limited by the API, retrying in 37s.")
		if got != tc.want {
			t.Errorf("%s: notice printed = %v, want %v (stderr %q)", tc.name, got, tc.want, buf.String())
		}
	}
}
