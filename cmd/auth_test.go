// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

func TestAPIKeysURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
		err  bool
	}{
		{"https://www.truestamp.com/api/json", "https://www.truestamp.com/api-keys", false},
		{"http://localhost:4000/api/json", "http://localhost:4000/api-keys", false},
		{"https://staging.truestamp.com", "https://staging.truestamp.com/api-keys", false},
		{"https://host:8443/custom/path", "https://host:8443/api-keys", false},
		{"", "", true},
		{"not a url", "", true},
		{"/relative/only", "", true},
	}
	for _, c := range cases {
		got, err := apiKeysURL(c.in)
		if c.err {
			if err == nil {
				t.Errorf("apiKeysURL(%q) = %q, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("apiKeysURL(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("apiKeysURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCheckAPIKey(t *testing.T) {
	httpclient.Init(5 * time.Second)

	cases := []struct {
		name            string
		status          int
		contentType     string
		body            string
		wantOK          bool
		wantUnauthd     bool
		wantUserID      string
		wantEmail       string
		wantFullName    string
		wantMsgContains string
	}{
		{
			name:         "200 with single user full_name",
			status:       200,
			contentType:  "application/vnd.api+json",
			body:         `{"data":{"id":"usr_01","type":"user","attributes":{"email":"alice@example.com","full_name":"Alice Smith"}}}`,
			wantOK:       true,
			wantUserID:   "usr_01",
			wantEmail:    "alice@example.com",
			wantFullName: "Alice Smith",
		},
		{
			name:         "200 with list of users and first/last name fallback",
			status:       200,
			contentType:  "application/vnd.api+json",
			body:         `{"data":[{"id":"usr_02","type":"user","attributes":{"email":"bob@example.com","first_name":"Bob","last_name":"Jones"}}]}`,
			wantOK:       true,
			wantUserID:   "usr_02",
			wantEmail:    "bob@example.com",
			wantFullName: "Bob Jones",
		},
		{
			name:        "200 with empty data still counts as authenticated",
			status:      200,
			contentType: "application/vnd.api+json",
			body:        `{"data":[]}`,
			wantOK:      true,
		},
		{
			name:            "401 unauthorized with error detail",
			status:          401,
			contentType:     "application/vnd.api+json",
			body:            `{"errors":[{"title":"Unauthorized","detail":"token expired"}]}`,
			wantUnauthd:     true,
			wantMsgContains: "token expired",
		},
		{
			name:        "403 forbidden with no body",
			status:      403,
			wantUnauthd: true,
		},
		{
			name:            "500 surfaces server message",
			status:          500,
			contentType:     "application/vnd.api+json",
			body:            `{"errors":[{"title":"Internal Server Error"}]}`,
			wantMsgContains: "Internal Server Error",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Path; got != "/api/json/users" {
					t.Errorf("path = %q, want /api/json/users", got)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
					t.Errorf("Authorization = %q, want 'Bearer test-key'", got)
				}
				if c.contentType != "" {
					w.Header().Set("Content-Type", c.contentType)
				}
				w.WriteHeader(c.status)
				if c.body != "" {
					_, _ = w.Write([]byte(c.body))
				}
			}))
			t.Cleanup(srv.Close)

			res, err := checkAPIKey(context.Background(), srv.URL+"/api/json", "test-key", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.ok != c.wantOK {
				t.Errorf("ok = %v, want %v", res.ok, c.wantOK)
			}
			if res.unauthorized != c.wantUnauthd {
				t.Errorf("unauthorized = %v, want %v", res.unauthorized, c.wantUnauthd)
			}
			if res.httpStatus != c.status {
				t.Errorf("httpStatus = %d, want %d", res.httpStatus, c.status)
			}
			if res.userID != c.wantUserID {
				t.Errorf("userID = %q, want %q", res.userID, c.wantUserID)
			}
			if res.email != c.wantEmail {
				t.Errorf("email = %q, want %q", res.email, c.wantEmail)
			}
			if res.fullName != c.wantFullName {
				t.Errorf("fullName = %q, want %q", res.fullName, c.wantFullName)
			}
			if c.wantMsgContains != "" && !strings.Contains(res.message, c.wantMsgContains) {
				t.Errorf("message = %q, want it to contain %q", res.message, c.wantMsgContains)
			}
		})
	}
}

func TestFetchTeam(t *testing.T) {
	httpclient.Init(5 * time.Second)

	cases := []struct {
		name          string
		status        int
		body          string
		wantFound     bool
		wantName      string
		wantPersonal  bool
		wantMsgNeedle string
	}{
		{
			name:         "200 returns team name",
			status:       200,
			body:         `{"data":{"id":"team_42","type":"team","attributes":{"name":"Acme Corp","personal":false}}}`,
			wantFound:    true,
			wantName:     "Acme Corp",
			wantPersonal: false,
		},
		{
			name:         "200 flags personal team",
			status:       200,
			body:         `{"data":{"id":"team_me","type":"team","attributes":{"name":"Alice's Team","personal":true}}}`,
			wantFound:    true,
			wantName:     "Alice's Team",
			wantPersonal: true,
		},
		{
			name:          "404 means not accessible",
			status:        404,
			body:          `{"errors":[{"status":"404","title":"Not Found","detail":"team not found"}]}`,
			wantFound:     false,
			wantMsgNeedle: "team not found",
		},
		{
			name:      "403 means not a member",
			status:    403,
			body:      `{"errors":[{"status":"403","title":"Forbidden"}]}`,
			wantFound: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got, want := r.URL.Path, "/api/json/teams/team_42"; got != want {
					t.Errorf("path = %q, want %q", got, want)
				}
				if got := r.Header.Get("tenant"); got != "team_42" {
					t.Errorf("tenant header = %q, want team_42", got)
				}
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(c.status)
				_, _ = w.Write([]byte(c.body))
			}))
			t.Cleanup(srv.Close)

			res, err := fetchTeam(context.Background(), srv.URL+"/api/json", "k", "team_42")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.found != c.wantFound {
				t.Errorf("found = %v, want %v", res.found, c.wantFound)
			}
			if res.name != c.wantName {
				t.Errorf("name = %q, want %q", res.name, c.wantName)
			}
			if res.personal != c.wantPersonal {
				t.Errorf("personal = %v, want %v", res.personal, c.wantPersonal)
			}
			if res.httpStatus != c.status {
				t.Errorf("httpStatus = %d, want %d", res.httpStatus, c.status)
			}
			if c.wantMsgNeedle != "" && !strings.Contains(res.message, c.wantMsgNeedle) {
				t.Errorf("message = %q, want it to contain %q", res.message, c.wantMsgNeedle)
			}
		})
	}
}

func TestFormatUserIdentity(t *testing.T) {
	cases := []struct {
		r    *authCheckResult
		want string
	}{
		{&authCheckResult{fullName: "Alice Smith", email: "a@b"}, "Alice Smith <a@b>"},
		{&authCheckResult{email: "a@b"}, "a@b"},
		{&authCheckResult{userID: "usr_01"}, "usr_01"},
		{&authCheckResult{}, "(identity not returned)"},
	}
	for _, c := range cases {
		if got := formatUserIdentity(c.r); got != c.want {
			t.Errorf("formatUserIdentity(%+v) = %q, want %q", c.r, got, c.want)
		}
	}
}

func TestFormatTeam(t *testing.T) {
	cases := []struct {
		teamID string
		r      *teamCheckResult
		want   string
	}{
		{"", nil, "personal team (no tenant header sent)"},
		{"team_42", nil, "team_42"},
		{"team_42", &teamCheckResult{}, "team_42"},
		{"team_42", &teamCheckResult{name: "Acme Corp"}, "Acme Corp  [team_42]"},
		{"team_me", &teamCheckResult{name: "Alice's Team", personal: true}, "Alice's Team (personal)  [team_me]"},
	}
	for _, c := range cases {
		if got := formatTeam(c.teamID, c.r); got != c.want {
			t.Errorf("formatTeam(%q, %+v) = %q, want %q", c.teamID, c.r, got, c.want)
		}
	}
}

func TestCheckAPIKey_SendsTenantHeaderWhenTeamSet(t *testing.T) {
	httpclient.Init(5 * time.Second)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("tenant"); got != "team_42" {
			t.Errorf("tenant header = %q, want team_42", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(srv.Close)

	if _, err := checkAPIKey(context.Background(), srv.URL+"/api/json", "k", "team_42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAPIKey_TransportErrorReturnsError(t *testing.T) {
	httpclient.Init(100 * time.Millisecond)
	// Point at a TCP port we're not listening on.
	res, err := checkAPIKey(context.Background(), "http://127.0.0.1:1/api/json", "k", "")
	if err == nil {
		t.Fatalf("expected transport-level error, got result: %+v", res)
	}
}
