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

// teamCmd is the parent for the `truestamp team ...` subtree. Invoking
// it without a subcommand is an alias for `team list`.
var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage the active Truestamp team context",
	Long: `Discover, switch between, and persist the active team that the CLI
sends with API requests as the multitenancy context. The team id is
stored under the top-level 'team' key in the user's config.toml so it
applies across CLI invocations.

Sub-commands:
  list    Show all teams the API key has membership in
  show    Show the currently-configured team in detail
  set     Set the active team (interactive picker if no id provided)
  unset   Clear the active team (server falls back to the personal team)

Invoking 'truestamp team' with no subcommand is an alias for 'team list'.

Team creation is currently web-only — visit '{public-web}/teams' in
your browser to create a new team. The CLI's empty-state hint surfaces
the right URL automatically.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runTeamList, // default = list
}

// teamConfig pulls the values the teams client needs from the resolved
// application config. Returns errSilentFail when no API key is set, after
// first printing a "not authenticated" banner to stderr (unless silent).
func teamConfig(cmd *cobra.Command) (teams.Config, error) {
	cfg := appConfig
	if cfg.APIKey == "" {
		silent, _ := cmd.Flags().GetBool("silent")
		if !silent {
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp auth login' to store an API key."))
		}
		return teams.Config{}, errSilentFail
	}
	return teams.Config{
		APIURL: cfg.APIURL,
		APIKey: cfg.APIKey,
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
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    The API key was rejected. Run 'truestamp auth login' to store a fresh key."))
		}
		return errSilentFail
	}
	if errors.Is(err, teams.ErrForbidden) {
		if !silent {
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Access denied"))
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Your API key is valid, but you do not have access to that team."))
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp team list' to see the teams you are a member of."))
		}
		return errSilentFail
	}
	if errors.Is(err, teams.ErrNotFound) {
		if !silent {
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Team not found"))
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    No team exists with that id. Run 'truestamp team list' to see valid options."))
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
	rootCmd.AddCommand(teamCmd)
}
