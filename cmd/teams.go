// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// teamsCmd is the parent for the `truestamp teams ...` subtree. Like every
// group it is a namespace (asGroup): a bare `truestamp teams` prints help.
var teamsCmd = &cobra.Command{
	Use:   "teams",
	Short: "List, create, and switch teams",
	Long: `Discover, switch between, and persist the active team that the CLI
sends with API requests as the multitenancy context. The team id is
stored under the top-level 'team' key in the user's config.toml so it
applies across CLI invocations. That key stays singular: it names
exactly one team, and only the command group is plural.

'current' and 'use' are a pair: 'current' reads the ambient team, 'use'
sets it. That is why there is no bare 'get' that silently falls back to
the configured team — the same command line would mean different things
on different machines.`,
}

// teamConfig pulls the values the teams client needs from the resolved
// application config, after the shared credential gate.
func teamConfig(cmd *cobra.Command) (teams.Config, error) {
	if err := requireAuth(cmd); err != nil {
		return teams.Config{}, err
	}
	return teams.Config{APIURL: appConfig.APIURL, Team: appConfig.Team}, nil
}

// teamRenderError converts a client error into a user-facing message
// and an appropriate non-zero exit. The 401/403/404 split is
// load-bearing: each gets remediation guidance specific to teams (the
// credential was rejected, list your teams, the id is wrong), which is
// why these three cases are not left to the shared renderAPIError.
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
	return renderAPIError(cmd, err, "team")
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
