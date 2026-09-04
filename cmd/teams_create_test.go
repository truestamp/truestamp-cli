// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/teams"
)

func TestNormalizeOwnership(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "", false}, // empty -> server default
		{"creator_retains", teams.OwnershipCreatorRetains, false},
		{"creator", teams.OwnershipCreatorRetains, false},
		{"creator-retains", teams.OwnershipCreatorRetains, false},
		{"team_retains", teams.OwnershipTeamRetains, false},
		{"team", teams.OwnershipTeamRetains, false},
		{"TEAM", teams.OwnershipTeamRetains, false},
		{"  team_retains  ", teams.OwnershipTeamRetains, false},
		{"bogus", "", true},
	}
	for _, tc := range cases {
		got, err := normalizeOwnership(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("normalizeOwnership(%q): want error, got %q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeOwnership(%q): unexpected error %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("normalizeOwnership(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// --- runTeamCreate orchestration (via the createTeamCtx seam) ------------

// newCreateTestCmd builds a fresh command carrying the team-create flags,
// with stdout/stderr captured into the returned buffers.
func newCreateTestCmd() (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	c := &cobra.Command{}
	f := c.Flags()
	f.StringP("name", "n", "", "")
	f.String("ownership-model", "", "")
	f.Bool("set", false, "")
	f.Bool("json", false, "")
	f.BoolP("silent", "s", false, "")
	var out, errb bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&errb)
	c.SetContext(context.Background())
	return c, &out, &errb
}

// withTeamCreateEnv sets the process-wide auth + appConfig + createTeamCtx
// seam for the duration of a test and restores them after.
func withTeamCreateEnv(t *testing.T, stub func(ctx context.Context, cfg teams.Config, name, ownership string) (*teams.Team, error)) {
	t.Helper()
	origCfg := appConfig
	origCreate := createTeamCtx
	origAuth := auth.Default()
	appConfig = &config.Config{APIURL: "http://example.test/api/json"}
	auth.SetDefault(auth.APIKeyAuthorizer("test-key"))
	createTeamCtx = stub
	t.Cleanup(func() {
		appConfig = origCfg
		createTeamCtx = origCreate
		auth.SetDefault(origAuth)
	})
}

func TestRunTeamCreate_Success(t *testing.T) {
	var gotName, gotOwnership string
	withTeamCreateEnv(t, func(_ context.Context, _ teams.Config, name, ownership string) (*teams.Team, error) {
		gotName, gotOwnership = name, ownership
		return &teams.Team{ID: "019dbd00", Name: name, OwnershipModel: teams.OwnershipCreatorRetains}, nil
	})

	c, out, _ := newCreateTestCmd()
	if err := runTeamCreate(c, []string{"Acme"}); err != nil {
		t.Fatalf("runTeamCreate: %v", err)
	}
	if gotName != "Acme" {
		t.Errorf("create called with name %q, want Acme", gotName)
	}
	if gotOwnership != "" {
		t.Errorf("ownership = %q, want empty (server default)", gotOwnership)
	}
	if s := out.String(); !strings.Contains(s, "Team created") || !strings.Contains(s, "019dbd00") {
		t.Errorf("stdout = %q, want 'Team created' + id", s)
	}
}

func TestRunTeamCreate_JSONSuccess(t *testing.T) {
	withTeamCreateEnv(t, func(_ context.Context, _ teams.Config, name, ownership string) (*teams.Team, error) {
		return &teams.Team{ID: "019dbd00", Name: name, OwnershipModel: teams.OwnershipTeamRetains, CreatedAt: "2026-06-24T00:00:00Z"}, nil
	})

	c, out, _ := newCreateTestCmd()
	_ = c.Flags().Set("json", "true")
	_ = c.Flags().Set("ownership-model", "team_retains")
	if err := runTeamCreate(c, []string{"Acme"}); err != nil {
		t.Fatalf("runTeamCreate: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out.String())
	}
	if got["id"] != "019dbd00" || got["ownership_model"] != "team_retains" {
		t.Errorf("json = %v", got)
	}
}

func TestRunTeamCreate_ErrorRoutesToRenderer(t *testing.T) {
	withTeamCreateEnv(t, func(_ context.Context, _ teams.Config, _, _ string) (*teams.Team, error) {
		return nil, fmt.Errorf("create: %w", teams.ErrTeamLimitReached)
	})

	c, out, errb := newCreateTestCmd()
	err := runTeamCreate(c, []string{"Acme"})
	if err == nil {
		t.Fatal("want non-nil error (errSilentFail) on create failure")
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty on a non-json failure, got %q", out.String())
	}
	if !strings.Contains(errb.String(), "Team limit reached") {
		t.Errorf("stderr = %q, want 'Team limit reached'", errb.String())
	}
}

func TestRunTeamCreate_JSONErrorIsParseable(t *testing.T) {
	withTeamCreateEnv(t, func(_ context.Context, _ teams.Config, _, _ string) (*teams.Team, error) {
		return nil, fmt.Errorf("create: %w", teams.ErrOwnershipNotEntitled)
	})

	c, out, _ := newCreateTestCmd()
	_ = c.Flags().Set("json", "true")
	if err := runTeamCreate(c, []string{"Acme"}); err == nil {
		t.Fatal("want non-nil error on create failure")
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("--json failure stdout is not parseable JSON: %v\n%s", err, out.String())
	}
	if got["code"] != "ownership_not_entitled" {
		t.Errorf("json code = %v, want ownership_not_entitled", got["code"])
	}
}

func TestRunTeamCreate_MissingNameNonInteractive(t *testing.T) {
	called := false
	withTeamCreateEnv(t, func(_ context.Context, _ teams.Config, _, _ string) (*teams.Team, error) {
		called = true
		return &teams.Team{ID: "x"}, nil
	})
	c, _, _ := newCreateTestCmd()
	_ = c.Flags().Set("json", "true") // forces non-interactive
	if err := runTeamCreate(c, nil); err == nil {
		t.Error("missing name in --json mode should be a hard error")
	}
	if called {
		t.Error("create must not be called when the name is missing")
	}
}

func TestApiErrorDetailRedacts(t *testing.T) {
	err := &teams.APIError{Status: 502, Detail: "upstream api_key=SECRETKEY12345 failed"}
	got := apiErrorDetail(err)
	if strings.Contains(got, "SECRETKEY12345") {
		t.Errorf("apiErrorDetail leaked a secret: %q", got)
	}
	// A non-APIError yields the empty string.
	if d := apiErrorDetail(fmt.Errorf("plain")); d != "" {
		t.Errorf("apiErrorDetail(plain) = %q, want empty", d)
	}
}
