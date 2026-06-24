// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package teams

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// decodeCreateBody reads the JSON:API create envelope the CLI sends so a
// test can assert the exact attribute set (and absence of forbidden keys).
func decodeCreateBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	var env struct {
		Data struct {
			Type       string         `json:"type"`
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decoding body %q: %v", body, err)
	}
	if env.Data.Type != "team" {
		t.Errorf("data.type = %q, want \"team\"", env.Data.Type)
	}
	return env.Data.Attributes
}

func TestCreateTeam_HappyPath(t *testing.T) {
	var gotTenant string
	var gotContentType string
	var attrs map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/teams" {
			t.Errorf("got %s %s, want POST /teams", r.Method, r.URL.Path)
		}
		gotTenant = r.Header.Get("tenant")
		gotContentType = r.Header.Get("Content-Type")
		attrs = decodeCreateBody(t, r)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"type":"team","id":"019dbd00-0000-7000-8000-000000000000","attributes":{"name":"Acme","ownership_model":"team_retains","personal":false,"creator_id":"user-1","inserted_at":"2026-06-24T00:00:00Z","updated_at":"2026-06-24T00:00:00Z"}}}`)
	}))
	defer srv.Close()

	// cfg.Team is set, but CreateTeam must clear it (account-scoped action).
	cfg := Config{APIURL: srv.URL, Team: "some-active-team"}
	team, err := CreateTeam(context.Background(), cfg, "Acme", OwnershipTeamRetains)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	if team.ID != "019dbd00-0000-7000-8000-000000000000" {
		t.Errorf("id = %q", team.ID)
	}
	if team.Name != "Acme" {
		t.Errorf("name = %q", team.Name)
	}
	if team.OwnershipModel != OwnershipTeamRetains {
		t.Errorf("ownership_model = %q", team.OwnershipModel)
	}
	if team.Personal {
		t.Error("personal = true, want false")
	}
	if team.CreatedAt != "2026-06-24T00:00:00Z" {
		t.Errorf("created_at (inserted_at) = %q", team.CreatedAt)
	}

	if gotContentType != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotTenant != "" {
		t.Errorf("tenant header = %q, want empty (account-scoped)", gotTenant)
	}
	if attrs["name"] != "Acme" {
		t.Errorf("attributes.name = %v", attrs["name"])
	}
	if attrs["ownership_model"] != OwnershipTeamRetains {
		t.Errorf("attributes.ownership_model = %v", attrs["ownership_model"])
	}
	// The server rejects unknown attributes — the CLI must not send these.
	if _, ok := attrs["creator_id"]; ok {
		t.Error("attributes must not contain creator_id")
	}
	if _, ok := attrs["memberships"]; ok {
		t.Error("attributes must not contain memberships")
	}
}

func TestCreateTeam_OmitsOwnershipWhenEmpty(t *testing.T) {
	var attrs map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attrs = decodeCreateBody(t, r)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"type":"team","id":"x","attributes":{"name":"Acme","ownership_model":"creator_retains","personal":false}}}`)
	}))
	defer srv.Close()

	if _, err := CreateTeam(context.Background(), Config{APIURL: srv.URL}, "Acme", ""); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if _, ok := attrs["ownership_model"]; ok {
		t.Error("ownership_model must be omitted when empty so the server applies its default")
	}
}

func TestCreateTeam_EmptyNameRejectedClientSide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called for an empty name")
	}))
	defer srv.Close()

	if _, err := CreateTeam(context.Background(), Config{APIURL: srv.URL}, "   ", ""); err == nil {
		t.Fatal("want error for empty name, got nil")
	}
}

// Plan-limit: a 4xx validation error with NO source.pointer whose detail names
// the team limit maps to ErrTeamLimitReached.
func TestCreateTeam_PlanLimitMapsToSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusBadRequest) // server-verified: both plan failures are 400
		_, _ = io.WriteString(w, `{"errors":[{"detail":"You have reached your plan's team limit (1 of 1 teams). Upgrade your plan to create more teams."}]}`)
	}))
	defer srv.Close()

	_, err := CreateTeam(context.Background(), Config{APIURL: srv.URL}, "Acme", "")
	if !errors.Is(err, ErrTeamLimitReached) {
		t.Fatalf("err = %v, want ErrTeamLimitReached", err)
	}
	var ae *APIError
	if errors.As(err, &ae) && ae.Detail == "" {
		t.Error("APIError.Detail should be preserved for display")
	}
}

// Ownership-not-entitled: a 4xx with source.pointer == ownershipModelPointer
// maps to ErrOwnershipNotEntitled (the structural discriminator).
func TestCreateTeam_OwnershipNotEntitledMapsToSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusBadRequest) // server-verified: both plan failures are 400
		_, _ = io.WriteString(w, `{"errors":[{"detail":"Your plan does not include team-retained item ownership. Upgrade to Starter or higher.","source":{"pointer":"/data/attributes/ownership_model"}}]}`)
	}))
	defer srv.Close()

	_, err := CreateTeam(context.Background(), Config{APIURL: srv.URL}, "Acme", OwnershipTeamRetains)
	if !errors.Is(err, ErrOwnershipNotEntitled) {
		t.Fatalf("err = %v, want ErrOwnershipNotEntitled", err)
	}
}

// A free-plan user requesting team_retains trips BOTH the plan-limit and the
// ownership-entitlement rejection in one errors[] array. The pointer-bearing
// ownership error must win regardless of its position (here it is second).
func TestCreateTeam_BothErrors_OwnershipPointerWins(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"errors":[`+
			`{"detail":"You have reached your plan's team limit (1 of 1 teams)."},`+
			`{"detail":"Your plan does not include team-retained item ownership. Upgrade to Starter or higher.","source":{"pointer":"/data/attributes/ownership_model"}}`+
			`]}`)
	}))
	defer srv.Close()

	_, err := CreateTeam(context.Background(), Config{APIURL: srv.URL}, "Acme", OwnershipTeamRetains)
	if !errors.Is(err, ErrOwnershipNotEntitled) {
		t.Fatalf("err = %v, want ErrOwnershipNotEntitled (pointer wins over the limit error, any order)", err)
	}
	// The displayed detail must be the ownership one, not the limit one.
	var ae *APIError
	if errors.As(err, &ae) && !strings.Contains(ae.Detail, "team-retained") {
		t.Errorf("detail = %q, want the ownership detail", ae.Detail)
	}
}

// A generic 4xx validation (neither the ownership pointer nor a team-limit
// detail) must NOT be mis-mapped to a plan sentinel.
func TestCreateTeam_GenericValidationStaysGeneric(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // server-verified: both plan failures are 400
		_, _ = io.WriteString(w, `{"errors":[{"detail":"name is required","source":{"pointer":"/data/attributes/name"}}]}`)
	}))
	defer srv.Close()

	_, err := CreateTeam(context.Background(), Config{APIURL: srv.URL}, "Acme", "")
	if errors.Is(err, ErrTeamLimitReached) || errors.Is(err, ErrOwnershipNotEntitled) {
		t.Fatalf("generic validation error mis-mapped to a plan sentinel: %v", err)
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("err = %v, want the ErrBadRequest class", err)
	}
}

// A non-JSON:API error body falls back to the raw body, which must be
// redacted (defense in depth against a reflected credential).
func TestCreateTeam_RawBodyFallbackIsRedacted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `upstream proxy error: api_key=SECRETKEY12345 rejected`)
	}))
	defer srv.Close()

	_, err := CreateTeam(context.Background(), Config{APIURL: srv.URL}, "Acme", "")
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if strings.Contains(ae.Detail, "SECRETKEY12345") {
		t.Errorf("raw-body fallback leaked a secret: %q", ae.Detail)
	}
}
