// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// teamsCmd is the parent for the `truestamp teams ...` subtree. Like every
// group it has no RunE: a bare `truestamp teams` prints help.
var teamsCmd = &cobra.Command{
	Use:   "teams",
	Short: "List, create, and switch teams",
	Long: `Discover, switch between, and persist the active team that the CLI
sends with API requests as the multitenancy context. The team id is
stored under the top-level 'team' key in the user's config.toml so it
applies across CLI invocations. That key stays singular: it names
exactly one team, and only the command group is plural.

Sub-commands:
  list     Show all teams you are a member of
  get      Show one team by id
  current  Show the team the CLI is currently pointed at
  create   Create a new team (interactive prompt if no name given)
  use      Point the CLI at a team (interactive picker if no id given;
           --clear to point at none)

'current' and 'use' are a pair: 'current' reads the ambient team, 'use'
sets it. That is why there is no bare 'get' that silently falls back to
the configured team — the same command line would mean different things
on different machines.`,
	Args: cobra.NoArgs,
}

// teamConfig pulls the values the teams client needs from the resolved
// application config. Returns errSilentFail when no credential is
// configured (neither an OAuth session nor an API key), after first
// printing a "not authenticated" banner to stderr (unless silent).
func teamConfig(cmd *cobra.Command) (teams.Config, error) {
	cfg := appConfig
	if !authConfigured() {
		_, silent := outputMode(cmd)
		if !silent {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp auth login' to sign in (or set TRUESTAMP_API_KEY)."))
		}
		return teams.Config{}, errSilentFail
	}
	return teams.Config{
		APIURL: cfg.APIURL,
		Team:   cfg.Team,
	}, nil
}

// teamRenderError converts a client error into a user-facing message
// and an appropriate non-zero exit. The 401/403/404 split is
// load-bearing: each gets distinct remediation guidance because the
// fixes are different.
func teamRenderError(cmd *cobra.Command, err error, silent bool) error {
	if errors.Is(err, teams.ErrUnauthorized) {
		if !silent {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Your credential was rejected. Run 'truestamp auth login' to sign in again."))
		}
		return errSilentFail
	}
	if errors.Is(err, teams.ErrForbidden) {
		if !silent {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Access denied"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    You're authenticated, but you do not have access to that team."))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp teams list' to see the teams you are a member of."))
		}
		return errSilentFail
	}
	if errors.Is(err, teams.ErrNotFound) {
		if !silent {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Team not found"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    No team exists with that id. Run 'truestamp teams list' to see valid options."))
		}
		return errSilentFail
	}
	if silent {
		return errSilentFail
	}
	var apiErr *teams.APIError
	if errors.As(err, &apiErr) {
		switch {
		case errors.Is(err, teams.ErrRateLimited) && apiErr.RetryAfter != "":
			return fmt.Errorf("rate limited (Retry-After: %s): %s", apiErr.RetryAfter, apiErr.Detail)
		default:
			return fmt.Errorf("%s", apiErr.Error())
		}
	}
	return err
}

// fetchMyMembershipsCtx is a thin wrapper that lets the test suite
// stub the network without exporting the underlying client. Defaults
// to the real client at runtime.
var fetchMyMembershipsCtx = func(ctx context.Context, cfg teams.Config) ([]teams.Membership, error) {
	return teams.ListMyMemberships(ctx, cfg)
}

func init() {
	teamsCmd.GroupID = groupResources
	rootCmd.AddCommand(asGroup(teamsCmd))
}
