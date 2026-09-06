// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/ui"
)

const (
	testTeamPersonalID  = "019dbd00-0000-7000-8000-000000000000"
	testTeamOtherID     = "019dbd00-0000-7000-8000-000000000001"
	testTeamPersonalMID = "019dbe00-0000-7000-8000-000000000000"
	testTeamOtherMID    = "019dbe00-0000-7000-8000-000000000001"

	testMembershipsBody = `{
  "data": [
    {"type":"membership","id":"` + testTeamPersonalMID + `","attributes":{"role":"team_owner"},
     "relationships":{"team":{"data":{"type":"team","id":"` + testTeamPersonalID + `"}}}},
    {"type":"membership","id":"` + testTeamOtherMID + `","attributes":{"role":"team_admin"},
     "relationships":{"team":{"data":{"type":"team","id":"` + testTeamOtherID + `"}}}}
  ],
  "included": [
    {"type":"team","id":"` + testTeamPersonalID + `","attributes":{"name":"Personal","personal":true,"ownership_model":"creator_retains","inserted_at":"2026-04-29T01:02:03Z"}},
    {"type":"team","id":"` + testTeamOtherID + `","attributes":{"name":"Engineering","personal":false,"ownership_model":"creator_retains","inserted_at":"2026-04-30T01:02:03Z"}}
  ]
}`

	testTeamPersonalSingleBody = `{"data":{"type":"team","id":"` + testTeamPersonalID + `","attributes":{"name":"Personal","personal":true,"ownership_model":"creator_retains","inserted_at":"2026-04-29T01:02:03Z"}}}`
	testTeamOtherSingleBody    = `{"data":{"type":"team","id":"` + testTeamOtherID + `","attributes":{"name":"Engineering","personal":false,"ownership_model":"creator_retains","inserted_at":"2026-04-30T01:02:03Z"}}}`
)

// startTeamServer mirrors startBeaconServer (cmd/beacons_test.go) for the
// team JSON:API surface. Routes match the production layout under
// <base_url>/api/json/... so tests pass --base-url <srv.URL>. The
// production CLI flow makes two parallel requests, `/teams` and
// `/memberships`, and joins client-side; the harness serves both.
func startTeamServer(t *testing.T) (string, func()) {
	t.Helper()
	const teamsListBody = `{
	  "data": [
	    {"type":"team","id":"` + testTeamPersonalID + `","attributes":{"name":"Personal","personal":true,"ownership_model":"creator_retains","inserted_at":"2026-04-29T01:02:03Z"}},
	    {"type":"team","id":"` + testTeamOtherID + `","attributes":{"name":"Engineering","personal":false,"ownership_model":"creator_retains","inserted_at":"2026-04-30T01:02:03Z"}}
	  ],
	  "links": {"next": null}
	}`
	mux := http.NewServeMux()
	mux.HandleFunc("/api/json/memberships", func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(testMembershipsBody))
	})
	// ServeMux prefers the more specific exact pattern `/api/json/teams`
	// (list) over the subtree pattern `/api/json/teams/` (single
	// resource), independent of registration order.
	mux.HandleFunc("/api/json/teams", func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(teamsListBody))
	})
	mux.HandleFunc("/api/json/teams/"+testTeamPersonalID, func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(testTeamPersonalSingleBody))
	})
	mux.HandleFunc("/api/json/teams/"+testTeamOtherID, func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		_, _ = w.Write([]byte(testTeamOtherSingleBody))
	})
	mux.HandleFunc("/api/json/teams/", func(w http.ResponseWriter, r *http.Request) {
		// Unknown id → 404 with JSON:API error envelope.
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"status":"404","code":"not_found","title":"Not Found"}]}`))
	})
	srv := httptest.NewServer(mux)
	return srv.URL, srv.Close
}

func TestCLI_Teams_List_JSON(t *testing.T) {
	url, stop := startTeamServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--base-url", url, "--api-key", "test-key",
		"teams", "list", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d, stdout=%q", exit, stdout)
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 memberships, got %d", len(got))
	}
}

// TestCLI_Teams_BareGroupPrintsHelp pins R0. `truestamp team` used to run
// `list` while `truestamp beacon` ran `latest`: two nouns, two defaults,
// two kinds of answer, and no rule for the seventh noun.
func TestCLI_Teams_BareGroupPrintsHelp(t *testing.T) {
	// Deliberately no credential: help must work without one.
	stdout, _, exit := runCLI(t, "teams")
	if exit != 0 {
		t.Fatalf("a bare group must exit 0, got %d", exit)
	}
	for _, want := range []string{"list", "get", "current", "create", "use"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("bare group help should list %q, got:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, testTeamPersonalID) {
		t.Error("a bare group must not run list")
	}
}

func TestCLI_Teams_Get_ByID_JSON(t *testing.T) {
	url, stop := startTeamServer(t)
	defer stop()

	stdout, _, exit := runCLI(t,
		"--base-url", url, "--api-key", "test-key",
		"teams", "get", testTeamOtherID, "--json")
	if exit != 0 {
		t.Fatalf("exit=%d, stdout=%q", exit, stdout)
	}
	var got struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Personal bool   `json:"personal"`
		Role     string `json:"role"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if got.ID != testTeamOtherID {
		t.Errorf("id = %q, want %q", got.ID, testTeamOtherID)
	}
	if got.Name != "Engineering" {
		t.Errorf("name = %q, want Engineering", got.Name)
	}
	if got.Role != "team_admin" {
		t.Errorf("role = %q, want team_admin", got.Role)
	}
	if got.Personal {
		t.Errorf("Engineering should not be personal")
	}
}

func TestCLI_Teams_Current_NoTeamConfigured(t *testing.T) {
	// Without --team and no config file, `teams current` should fail with
	// the no-team-configured banner. The cleanEnv() helper ensures no
	// TRUESTAMP_TEAM env var leaks in. `teams get` cannot stand in here:
	// it now requires an explicit id, which is the whole point of the
	// split.
	url, stop := startTeamServer(t)
	defer stop()

	_, stderr, exit := runCLI(t,
		"--base-url", url, "--api-key", "test-key",
		"teams", "current")
	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(stderr, "No team configured") {
		t.Errorf("want 'No team configured' banner, stderr=%q", stderr)
	}
}

func TestCLI_Teams_Get_NotFound(t *testing.T) {
	url, stop := startTeamServer(t)
	defer stop()

	bogus := "ffffffff-ffff-7fff-8fff-ffffffffffff"
	_, stderr, exit := runCLI(t,
		"--base-url", url, "--api-key", "test-key",
		"teams", "get", bogus)
	if exit == 0 {
		t.Fatal("expected non-zero exit for unknown team id")
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("want 'not found' in stderr, got %q", stderr)
	}
}

// TestTeamURLHelpers covers the TeamDetailURL / TeamCreateURL helpers
// in internal/ui/weburls.go. URLs render unconditionally for
// localhost / plain http / https.
func TestTeamURLHelpers(t *testing.T) {
	const id = "019dbd00-0000-7000-8000-000000000000"
	cases := []struct {
		apiURL     string
		wantDetail string
		wantCreate string
	}{
		{
			apiURL:     "https://www.truestamp.com/api/json",
			wantDetail: "https://www.truestamp.com/teams/" + id,
			wantCreate: "https://www.truestamp.com/teams",
		},
		{
			apiURL:     "http://localhost:4000/api/json",
			wantDetail: "http://localhost:4000/teams/" + id,
			wantCreate: "http://localhost:4000/teams",
		},
		{
			apiURL:     "https://example.com/api/json/",
			wantDetail: "https://example.com/teams/" + id,
			wantCreate: "https://example.com/teams",
		},
	}
	for _, c := range cases {
		t.Run(c.apiURL, func(t *testing.T) {
			if got := ui.TeamDetailURL(c.apiURL, id); got != c.wantDetail {
				t.Errorf("TeamDetailURL = %q, want %q", got, c.wantDetail)
			}
			if got := ui.TeamCreateURL(c.apiURL); got != c.wantCreate {
				t.Errorf("TeamCreateURL = %q, want %q", got, c.wantCreate)
			}
		})
	}
	// Empty inputs yield empty outputs (don't render a half-baked URL).
	if got := ui.TeamDetailURL("", id); got != "" {
		t.Errorf("TeamDetailURL with empty apiURL = %q, want empty", got)
	}
	if got := ui.TeamDetailURL("https://example.com", ""); got != "" {
		t.Errorf("TeamDetailURL with empty teamID = %q, want empty", got)
	}
}
