// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/teams"
)

// useHTTPTimeout gives the shared HTTP client timeout d for one test and
// puts back the client's 10-second default when the test ends. The client
// is process-wide, so a short timeout a test leaves behind governs every
// later test in the package: TestCheckAPIKey_TransportErrorReturnsError's
// 100ms once failed TestCheckAuth_RateLimited on a slow CI runner, whose
// loopback server took longer than that to answer.
func useHTTPTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	httpclient.Init(d)
	t.Cleanup(func() { httpclient.Init(10 * time.Second) })
}

func TestAPIKeysURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
		err  bool
	}{
		{"https://www.truestamp.com/api/json", "https://www.truestamp.com/api-keys", false},
		{"http://localhost:4000/api/json", "http://localhost:4000/api-keys", false},
		{"https://alt.example.com", "https://alt.example.com/api-keys", false},
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
	useHTTPTimeout(t, 5*time.Second)

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

			res, err := checkAuth(context.Background(), auth.APIKeyAuthorizer("test-key"), srv.URL+"/api/json")
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
	useHTTPTimeout(t, 5*time.Second)

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

			res, err := fetchTeam(context.Background(), auth.APIKeyAuthorizer("k"), srv.URL+"/api/json", "team_42")
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

// formatTeamLines is the renderer `auth status` actually calls;
// TestFormatTeam used to cover a single-line variant no command reached.
func TestFormatTeamLines(t *testing.T) {
	cases := []struct {
		name   string
		teamID string
		r      *teamCheckResult
		want   []string
	}{
		{"no team", "", nil, []string{"Team: personal team (no tenant header sent)"}},
		{"id only, no lookup", "team_42", nil, []string{"Team Id:   team_42"}},
		{"id only, empty lookup", "team_42", &teamCheckResult{}, []string{"Team Id:   team_42"}},
		{"named", "team_42", &teamCheckResult{name: "Acme Corp"},
			[]string{"Team Id:   team_42", "Team Name: Acme Corp"}},
		{"personal is annotated", "team_me", &teamCheckResult{name: "Alice's Team", personal: true},
			[]string{"Team Id:   team_me", "Team Name: Alice's Team (personal)"}},
		{"role included when known", "team_42", &teamCheckResult{name: "Acme Corp", role: "team_owner"},
			[]string{"Team Id:   team_42", "Team Name: Acme Corp", "Team Role: " + teams.FormatRole("team_owner")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := formatTeamLines(c.teamID, c.r)
			if len(got) != len(c.want) {
				t.Fatalf("formatTeamLines = %q, want %q", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

// TestCheckAuth_NeverSendsTenantHeader: the /users probe tests the
// credential alone. The server refuses a tenant the caller cannot use with
// a 403 before any action runs, so a probe that carried the configured team
// read a valid credential as rejected (auth status) and lost the identity
// (auth login, which runs the same probe). The stub answers like the
// server: 403 whenever a tenant header is present.
func TestCheckAuth_NeverSendsTenantHeader(t *testing.T) {
	useHTTPTimeout(t, 5*time.Second)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("tenant"); got != "" {
			t.Errorf("tenant header = %q, want none", got)
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(tenantRefusalBody))
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[{"type":"user","id":"u1","attributes":{"email":"a@example.com"}}]}`))
	}))
	t.Cleanup(srv.Close)

	res, err := checkAuth(context.Background(), auth.APIKeyAuthorizer("k"), srv.URL+"/api/json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.ok || res.email != "a@example.com" {
		t.Errorf("result = %+v, want ok with the identity resolved", res)
	}
}

// tenantRefusalBody is the server's answer to a tenant header naming a
// team the caller cannot use (truestamp-v2's tenant plug), without the
// source member a newer server adds.
const tenantRefusalBody = `{"errors":[{"status":"403","code":"forbidden","title":"Forbidden","detail":"Invalid or unauthorized tenant specified"}]}`

// tenantStrictServer answers /api/json/users with a user unless a tenant
// header is present, and /api/json/teams/<team> with the tenant refusal,
// as the server does for a team the caller is not a member of.
func tenantStrictServer(t *testing.T, team string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch {
		case r.URL.Path == "/api/json/users" && r.Header.Get("tenant") == "":
			_, _ = w.Write([]byte(`{"data":[{"type":"user","id":"u1","attributes":{"email":"a@example.com"}}]}`))
		case r.URL.Path == "/api/json/users", r.URL.Path == "/api/json/teams/"+team:
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(tenantRefusalBody))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestAuthStatus_InaccessibleTeam_TeamNotAccessible: a valid credential
// with a --team the caller cannot use is reported as the team's failure,
// the reason the help documents, never as a rejected credential.
func TestAuthStatus_InaccessibleTeam_TeamNotAccessible(t *testing.T) {
	const team = "019dbd00-0000-7000-8000-00000000000a"
	srv := tenantStrictServer(t, team)

	stdout, stderr, exit := runCLI(t, "auth", "status", "--api-key", "k", "--base-url", srv.URL, "--team", team, "--json")
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\nstdout=%s\nstderr=%s", exit, stdout, stderr)
	}
	var rec struct {
		OK         bool   `json:"ok"`
		Reason     string `json:"reason"`
		HTTPStatus int    `json:"http_status"`
		TeamID     string `json:"team_id"`
		Email      string `json:"email"`
	}
	if err := json.Unmarshal([]byte(stdout), &rec); err != nil {
		t.Fatalf("stdout is not one JSON record: %v\n%s", err, stdout)
	}
	if rec.OK || rec.Reason != "team_not_accessible" || rec.HTTPStatus != 403 || rec.TeamID != team {
		t.Errorf("record = %+v, want ok=false reason=team_not_accessible http_status=403 team_id=%s", rec, team)
	}
	if rec.Email != "a@example.com" {
		t.Errorf("email = %q, want the identity the credential resolved", rec.Email)
	}

	text, _, exit := runCLI(t, "auth", "status", "--api-key", "k", "--base-url", srv.URL, "--team", team)
	if exit != 1 {
		t.Fatalf("text exit = %d, want 1\n%s", exit, text)
	}
	if !strings.Contains(text, "Team "+team+" is not accessible") {
		t.Errorf("text output should name the team as not accessible:\n%s", text)
	}
	if strings.Contains(text, "Credential rejected") {
		t.Errorf("text output must not call the credential rejected:\n%s", text)
	}
}

func TestCheckAPIKey_TransportErrorReturnsError(t *testing.T) {
	useHTTPTimeout(t, 100*time.Millisecond)
	// Point at a TCP port we're not listening on.
	res, err := checkAuth(context.Background(), auth.APIKeyAuthorizer("k"), "http://127.0.0.1:1/api/json")
	if err == nil {
		t.Fatalf("expected transport-level error, got result: %+v", res)
	}
}

// auth status --help lists every reason identifier the command can emit,
// read from the source so the list cannot fall behind it (rate_limited
// was missing).
func TestAuthStatus_HelpNamesEveryReason(t *testing.T) {
	src, err := os.ReadFile("auth.go")
	if err != nil {
		t.Fatal(err)
	}
	reasons := regexp.MustCompile(`rec\.Reason = "([a-z_]+)"`).FindAllStringSubmatch(string(src), -1)
	if len(reasons) < 5 {
		t.Fatalf("found only %d reasons in auth.go", len(reasons))
	}
	long := strings.Join(strings.Fields(authStatusCmd.Long), " ")
	for _, m := range reasons {
		if !regexp.MustCompile(`\b` + m[1] + `\b`).MatchString(long) {
			t.Errorf("auth status --help does not name reason %q", m[1])
		}
	}
}
