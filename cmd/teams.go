// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/teams"
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
the configured team: the same command line would mean different things
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
// and an appropriate non-zero exit. A 401 is the shared credential
// rendering every group uses. The 403 and 404 cases keep remediation
// specific to teams (list your teams, the id is wrong), which is why they
// are not left to the shared renderAPIError. A 403 for a team the caller
// is not a member of reads correctly as "no access to that team" whether
// the server refused the tenant header or the team record, so the text
// does not tell the two apart; the --json code does.
func teamRenderError(cmd *cobra.Command, err error, silent bool) error {
	if silent {
		return errSilentFail
	}
	if errors.Is(err, teams.ErrUnauthorized) {
		return renderAPIError(cmd, err, "team")
	}
	var apiErr *teams.APIError
	isAPI := errors.As(err, &apiErr)
	if errors.Is(err, teams.ErrForbidden) {
		e := &cliError{
			code:    codeForbidden,
			message: "You're authenticated, but you do not have access to that team. Run 'truestamp teams list' to see the teams you are a member of.",
			status:  403,
			cause:   err,
		}
		if isAPI && isTenantRefusal(apiErr) {
			e.code = codeTeamNotAccessible
		}
		return failWith(cmd, e, "Access denied",
			"You're authenticated, but you do not have access to that team.",
			"Run 'truestamp teams list' to see the teams you are a member of.")
	}
	if errors.Is(err, teams.ErrNotFound) {
		return failWith(cmd, &cliError{
			code:    codeNotFound,
			message: "No team exists with that id. Run 'truestamp teams list' to see valid options.",
			status:  404,
			cause:   err,
		}, "Team not found",
			"No team exists with that id. Run 'truestamp teams list' to see valid options.")
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
