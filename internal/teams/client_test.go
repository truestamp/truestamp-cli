// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package teams

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	personalTeamID = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
	otherTeamID    = "019db702-c08c-73dc-a7cd-2c5e011f1dae"
	personalMID    = "019db702-d000-73dc-a7cd-2c5e011f1d00"
	otherMID       = "019db702-d000-73dc-a7cd-2c5e011f1d01"

	memberRoleOwner = "team_owner"
	memberRoleAdmin = "team_admin"
)

// jsonAPIMembershipsBody mirrors the real `GET /memberships` response
// shape from a live Ash JSON:API server: `team_id` lives in
// `attributes`, the `relationships.team.data` block exists but is
// nullable. Note we deliberately do NOT request `?include=team` so
// `included` is absent; the CLI joins teams from a parallel /teams
// fetch.
const jsonAPIMembershipsBody = `{
  "data": [
    {
      "type": "membership",
      "id": "` + personalMID + `",
      "attributes": {"role": "team_owner", "team_id": "` + personalTeamID + `", "user_id": "u-1", "inserted_at": "2026-04-29T01:02:03Z", "updated_at": "2026-04-29T01:02:03Z"},
      "relationships": {"team": {"data": {"type": "team", "id": "` + personalTeamID + `"}}, "user": {"data": null}}
    },
    {
      "type": "membership",
      "id": "` + otherMID + `",
      "attributes": {"role": "team_admin", "team_id": "` + otherTeamID + `", "user_id": "u-1", "inserted_at": "2026-04-30T01:02:03Z", "updated_at": "2026-04-30T01:02:03Z"},
      "relationships": {"team": {"data": null}, "user": {"data": null}}
    }
  ],
  "links": {"next": null, "self": "..."}
}`

// jsonAPITeamsListBody mirrors the real `GET /teams` response — the
// flat list of teams the actor can read. The CLI joins this onto
// memberships client-side.
const jsonAPITeamsListBody = `{
  "data": [
    {
      "type": "team",
      "id": "` + personalTeamID + `",
      "attributes": {"name": "Personal", "personal": true, "ownership_model": "creator_retains", "inserted_at": "2026-04-29T01:02:03Z"}
    },
    {
      "type": "team",
      "id": "` + otherTeamID + `",
      "attributes": {"name": "Engineering", "personal": false, "ownership_model": "creator_retains", "inserted_at": "2026-04-30T01:02:03Z"}
    }
  ],
  "links": {"next": null}
}`

const jsonAPISingleTeamBody = `{
  "data": {
    "type": "team",
    "id": "` + otherTeamID + `",
    "attributes": {"name": "Engineering", "personal": false, "ownership_model": "creator_retains", "inserted_at": "2026-04-30T01:02:03Z"}
  }
}`

func newServer(t *testing.T, fn http.HandlerFunc) (Config, func()) {
	t.Helper()
	srv := httptest.NewServer(fn)
	return Config{APIURL: srv.URL, APIKey: "test-key"}, srv.Close
}

// twoEndpointMux serves both /memberships and /teams against the
// supplied bodies. Mirrors the real-server pattern where the CLI
// makes two parallel requests and joins client-side.
func twoEndpointMux(t *testing.T, membershipsBody, teamsBody string) (Config, func()) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/memberships", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing Bearer header on /memberships: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/vnd.api+json" {
			t.Errorf("wrong Accept header: %q", r.Header.Get("Accept"))
		}
		// Critical regression: the CLI must NOT send ?include=team —
		// it's unreliable on real servers because the included array
		// is sparse. Instead, joining is done client-side via the
		// parallel /teams call. Anchor that here so a future
		// "optimization" doesn't reintroduce the include.
		if strings.Contains(r.URL.RawQuery, "include=team") {
			t.Errorf("CLI must not send ?include=team on /memberships; got %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(membershipsBody))
	})
	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing Bearer header on /teams: %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(teamsBody))
	})
	srv := httptest.NewServer(mux)
	return Config{APIURL: srv.URL, APIKey: "test-key"}, srv.Close
}

func TestListMyMemberships_HappyPath(t *testing.T) {
	cfg, stop := twoEndpointMux(t, jsonAPIMembershipsBody, jsonAPITeamsListBody)
	defer stop()

	got, err := ListMyMemberships(context.Background(), cfg)
	if err != nil {
		t.Fatalf("ListMyMemberships: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 memberships, got %d", len(got))
	}

	personal := got[0]
	if personal.TeamID != personalTeamID {
		t.Errorf("personal team_id = %q, want %q", personal.TeamID, personalTeamID)
	}
	if personal.Role != memberRoleOwner {
		t.Errorf("personal role = %q, want %q", personal.Role, memberRoleOwner)
	}
	if personal.Team == nil {
		t.Fatal("personal team not joined from /teams")
	}
	if personal.Team.Name != "Personal" || !personal.Team.Personal {
		t.Errorf("personal team attrs wrong: %+v", personal.Team)
	}

	other := got[1]
	if other.Role != memberRoleAdmin {
		t.Errorf("other role = %q, want %q", other.Role, memberRoleAdmin)
	}
	if other.Team == nil || other.Team.Name != "Engineering" || other.Team.Personal {
		t.Errorf("other team attrs wrong: %+v", other.Team)
	}
}

// TestListMyMemberships_AnchorsOnTeamsList simulates the real-server
// admin-key case: /memberships returns rows for teams the actor
// can't actually read (orphans or admin-bypass leakage). The CLI
// must drop those rows — the source of truth for "teams I can see"
// is /teams, not /memberships.
func TestListMyMemberships_AnchorsOnTeamsList(t *testing.T) {
	// /teams returns only Personal — the actor isn't a member of
	// the "other" team so the team READ policy filters it out.
	teamsWithOnlyPersonal := `{
	  "data": [
	    {"type":"team","id":"` + personalTeamID + `","attributes":{"name":"Personal","personal":true,"ownership_model":"creator_retains","inserted_at":"2026-04-29T01:02:03Z"}}
	  ],
	  "links": {"next": null}
	}`
	cfg, stop := twoEndpointMux(t, jsonAPIMembershipsBody, teamsWithOnlyPersonal)
	defer stop()

	got, err := ListMyMemberships(context.Background(), cfg)
	if err != nil {
		t.Fatalf("ListMyMemberships: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 row anchored on readable teams, got %d", len(got))
	}
	if got[0].TeamID != personalTeamID {
		t.Errorf("returned team_id = %q, want %q", got[0].TeamID, personalTeamID)
	}
	if got[0].Team == nil || !got[0].Team.Personal {
		t.Errorf("Personal team metadata wrong: %+v", got[0].Team)
	}
}

// TestListMyMemberships_TeamsCallReturnsError confirms that a /teams
// failure (the source of truth) propagates as an error rather than
// silently masking the outage. The user needs to know the team list
// is unavailable, not be shown a stale or empty rendering.
func TestListMyMemberships_TeamsCallReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/memberships", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(jsonAPIMembershipsBody))
	})
	mux.HandleFunc("/teams", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"db down"}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cfg := Config{APIURL: srv.URL, APIKey: "test-key"}

	if _, err := ListMyMemberships(context.Background(), cfg); err == nil {
		t.Fatal("expected error when /teams 500s")
	}
}

// TestListMyMemberships_MembershipsCallSoftFails: /teams is the
// source of truth, so the team list still renders if /memberships is
// down. Roles come back empty in that scenario.
func TestListMyMemberships_MembershipsCallSoftFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/memberships", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"db down"}]}`))
	})
	mux.HandleFunc("/teams", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(jsonAPITeamsListBody))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cfg := Config{APIURL: srv.URL, APIKey: "test-key"}

	got, err := ListMyMemberships(context.Background(), cfg)
	if err != nil {
		t.Fatalf("ListMyMemberships should not error when /memberships 500s: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 teams even with /memberships down, got %d", len(got))
	}
	for _, m := range got {
		if m.Team == nil {
			t.Errorf("Team should be populated from /teams; got nil")
		}
		if m.Role != "" {
			t.Errorf("Role should be empty under /memberships outage; got %q", m.Role)
		}
	}
}

// TestListMyMemberships_PaginationWalksMultiplePages confirms the
// walker follows links.next on both endpoints. Each endpoint returns
// two pages; the CLI must combine them.
func TestListMyMemberships_PaginationWalksMultiplePages(t *testing.T) {
	mux := http.NewServeMux()
	var page atomicCounter
	mux.HandleFunc("/memberships", func(w http.ResponseWriter, r *http.Request) {
		// First call returns page 1 with a next link; subsequent
		// call follows the cursor. Keying by page[after] presence
		// keeps the harness simple.
		if !strings.Contains(r.URL.RawQuery, "page[after]") {
			page1 := `{
			  "data": [{"type":"membership","id":"m-1","attributes":{"role":"team_owner","team_id":"` + personalTeamID + `","inserted_at":"2026-04-29T01:02:03Z","updated_at":"2026-04-29T01:02:03Z"}}],
			  "links": {"next": "` + serverURL(r) + `/memberships?page[after]=cursor1"}
			}`
			_, _ = w.Write([]byte(page1))
			return
		}
		page2 := `{
		  "data": [{"type":"membership","id":"m-2","attributes":{"role":"team_admin","team_id":"` + otherTeamID + `","inserted_at":"2026-04-30T01:02:03Z","updated_at":"2026-04-30T01:02:03Z"}}],
		  "links": {"next": null}
		}`
		_, _ = w.Write([]byte(page2))
	})
	mux.HandleFunc("/teams", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(jsonAPITeamsListBody))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	_ = page
	cfg := Config{APIURL: srv.URL, APIKey: "test-key"}

	got, err := ListMyMemberships(context.Background(), cfg)
	if err != nil {
		t.Fatalf("ListMyMemberships: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 memberships across 2 pages, got %d", len(got))
	}
	// Production output is sorted by team id for stability — both
	// memberships must be present, but order is determined by
	// teamID, not by which page the membership came from.
	ids := map[string]bool{}
	for _, m := range got {
		ids[m.ID] = true
	}
	if !ids["m-1"] || !ids["m-2"] {
		t.Errorf("pagination missed a row, got %+v", got)
	}
}

// serverURL returns the http-scheme self URL for a request against
// httptest.Server. Useful when emitting next-page cursors.
func serverURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// atomicCounter is a tiny helper kept untyped to avoid pulling in
// sync/atomic for a single test fixture.
type atomicCounter int

func TestListMyMemberships_TenantHeaderForwarded(t *testing.T) {
	mux := http.NewServeMux()
	check := func(label string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("tenant") != personalTeamID {
				t.Errorf("%s: expected tenant header %q, got %q", label, personalTeamID, r.Header.Get("tenant"))
			}
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}
	mux.HandleFunc("/memberships", check("memberships"))
	mux.HandleFunc("/teams", check("teams"))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cfg := Config{APIURL: srv.URL, APIKey: "test-key", Team: personalTeamID}

	if _, err := ListMyMemberships(context.Background(), cfg); err != nil {
		t.Fatalf("ListMyMemberships: %v", err)
	}
}

func TestGetTeam_HappyPath(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/teams/"+otherTeamID {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(jsonAPISingleTeamBody))
	})
	defer stop()

	team, err := GetTeam(context.Background(), cfg, otherTeamID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if team.ID != otherTeamID || team.Name != "Engineering" {
		t.Errorf("team attrs wrong: %+v", team)
	}
	if team.Personal {
		t.Errorf("Engineering should not be personal")
	}
}

func TestGetTeam_EmptyIDRejectedClientSide(t *testing.T) {
	cfg, stop := newServer(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called for empty id")
	})
	defer stop()

	if _, err := GetTeam(context.Background(), cfg, ""); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestDoGet_NoAPIKeyShortCircuit(t *testing.T) {
	cfg, stop := newServer(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called when api key is empty")
	})
	defer stop()
	cfg.APIKey = ""

	_, err := ListMyMemberships(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for empty api key")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err should wrap ErrUnauthorized, got %v", err)
	}
}

// TestDoGet_403WrapsErrForbidden anchors the 401-vs-403 split that
// the CLI's user-facing error rendering depends on. 401 means the
// API key is bad; 403 means the key is valid but the actor isn't
// allowed to read this resource (typically: tenant header points to
// a non-member team). Conflating them was a real bug — see the
// note on ErrForbidden in client.go.
func TestDoGet_403WrapsErrForbidden(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":[{"status":"403","code":"forbidden","title":"Forbidden","detail":"Invalid or unauthorized tenant specified"}]}`))
	})
	defer stop()

	_, err := GetTeam(context.Background(), cfg, otherTeamID)
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("403 should wrap ErrForbidden, got %v", err)
	}
	if errors.Is(err, ErrUnauthorized) {
		t.Errorf("403 must NOT wrap ErrUnauthorized; that's reserved for 401")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type = %T, want *APIError", err)
	}
	if !strings.Contains(apiErr.Detail, "tenant") {
		t.Errorf("detail should contain tenant, got %q", apiErr.Detail)
	}
}

// TestDoGet_401WrapsErrUnauthorized confirms 401 still maps to the
// auth-failure sentinel.
func TestDoGet_401WrapsErrUnauthorized(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors":[{"status":"401","code":"unauthorized","detail":"bad api key"}]}`))
	})
	defer stop()

	_, err := GetTeam(context.Background(), cfg, otherTeamID)
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("401 should wrap ErrUnauthorized, got %v", err)
	}
	if errors.Is(err, ErrForbidden) {
		t.Errorf("401 must NOT wrap ErrForbidden")
	}
}

func TestDoGet_404NotFound(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"status":"404","code":"not_found","title":"Not Found"}]}`))
	})
	defer stop()

	_, err := GetTeam(context.Background(), cfg, otherTeamID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err should wrap ErrNotFound, got %v", err)
	}
}

func TestDoGet_429RetryAfterPropagated(t *testing.T) {
	// /teams is the first call ListMyMemberships makes, so the rate-
	// limit response on that endpoint propagates through.
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"rate limited"}]}`))
	})
	defer stop()

	_, err := ListMyMemberships(context.Background(), cfg)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type = %T, want *APIError", err)
	}
	if apiErr.RetryAfter != "30" {
		t.Errorf("Retry-After = %q, want 30", apiErr.RetryAfter)
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("err should wrap ErrRateLimited, got %v", err)
	}
}

func TestParseAPIError_HTMLFallback(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<html><body>500 Internal Server Error</body></html>`))
	})
	defer stop()

	// /teams fails first (source of truth), so the HTML fallback
	// surfaces here.
	_, err := ListMyMemberships(context.Background(), cfg)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err type = %T, want *APIError", err)
	}
	if !strings.Contains(apiErr.Detail, "HTML") {
		t.Errorf("detail should mention HTML, got %q", apiErr.Detail)
	}
}

func TestGetMyRoleOnTeam_FoundAndMissing(t *testing.T) {
	cfg, stop := twoEndpointMux(t, jsonAPIMembershipsBody, jsonAPITeamsListBody)
	defer stop()

	role, err := GetMyRoleOnTeam(context.Background(), cfg, otherTeamID)
	if err != nil {
		t.Fatalf("GetMyRoleOnTeam: %v", err)
	}
	if role != memberRoleAdmin {
		t.Errorf("role = %q, want %q", role, memberRoleAdmin)
	}

	// Unknown team id → empty role, no error.
	role, err = GetMyRoleOnTeam(context.Background(), cfg, "ffffffff-0000-7000-8000-ffffffffffff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != "" {
		t.Errorf("role for unknown team = %q, want empty", role)
	}
}

func TestFormatRole(t *testing.T) {
	cases := map[string]string{
		"team_owner":  "Owner",
		"team_admin":  "Admin",
		"team_member": "Member",
		"team_viewer": "Viewer",
		"unknown":     "unknown",
		"":            "",
	}
	for in, want := range cases {
		if got := FormatRole(in); got != want {
			t.Errorf("FormatRole(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseMemberships_AttributeFallbackForTeamID(t *testing.T) {
	// Some Ash configurations may emit team_id as an attribute rather
	// than a relationship. Confirm the parser accepts both shapes.
	body := `{
	  "data": [
	    {
	      "type": "membership",
	      "id": "` + personalMID + `",
	      "attributes": {"role": "team_member", "team_id": "` + personalTeamID + `"}
	    }
	  ],
	  "included": []
	}`
	got, err := parseMembershipsWithIncluded([]byte(body))
	if err != nil {
		t.Fatalf("parseMembershipsWithIncluded: %v", err)
	}
	if len(got) != 1 || got[0].TeamID != personalTeamID {
		t.Errorf("expected attribute-form team_id=%q, got %+v", personalTeamID, got)
	}
}
