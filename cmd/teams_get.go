// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var teamsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one team by id",
	Long: `Show a team by id, with its name, role, personal flag, ownership
model, and public-web links.

The id must be one you have membership in; the server enforces this and
an attempt to read a team you do not belong to surfaces a 403 banner.

For the team the CLI is currently pointed at, use 'truestamp teams
current'. They are separate commands on purpose: a 'get' that silently
read config.toml when given no argument would mean different things on
different machines, which is a footgun in a script.

Examples:
  truestamp teams get 019dbd00-0000-7000-8000-000000000000
  truestamp teams get 019dbd00-0000-7000-8000-000000000000 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTeamsGet,
}

var teamsCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the team the CLI is currently pointed at",
	Long: `Show the active team, the one configured under 'team' in config.toml
or supplied by --team / TRUESTAMP_TEAM, with its name, role, personal
flag, ownership model, and public-web links.

Pairs with 'truestamp teams use', which sets it. Exits non-zero when no
team is configured.

Examples:
  truestamp teams current
  truestamp teams current --json`,
	Args: cobra.NoArgs,
	RunE: runTeamsGet,
}

func runTeamsGet(cmd *cobra.Command, args []string) error {
	jsonOut, silent := outputMode(cmd)

	cfg, err := teamConfig(cmd)
	if err != nil {
		return err
	}

	teamID := cfg.Team
	if len(args) > 0 {
		teamID = args[0]
	}
	if teamID == "" {
		if !silent {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("No team configured"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp teams use' to pick one interactively, or "+
					"'truestamp teams list' to see all available teams."))
		}
		return errSilentFail
	}

	// Use the actor's CURRENT active tenant (cfg.Team) as the tenant
	// header, never the id we're looking up. The server's tenant
	// resolution rejects the request with 403 if the tenant header
	// names a team the actor isn't a member of, which would mask the
	// real "not found" / "no membership" cause behind a misleading
	// "forbidden" surface.
	team, err := teams.GetTeam(cmd.Context(), cfg, teamID)
	if err != nil {
		return teamRenderError(cmd, err, silent)
	}

	role, err := teams.GetMyRoleOnTeam(cmd.Context(), cfg, teamID)
	if err != nil {
		// Soft-fail the role lookup, the team detail is the headline
		// info; missing role degrades to "(unknown)" rather than
		// erroring the whole subcommand.
		if !silent && !errors.Is(err, teams.ErrUnauthorized) {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"  warning: could not resolve role: "+err.Error()))
		}
	}

	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), struct {
			*teams.Team
			Role string `json:"role"`
		}{team, role})
	}

	renderTeamCard(cmd.OutOrStdout(), appConfig.APIURL, team, role, cfg.Team == teamID)
	return nil
}

// renderTeamCard prints a 5+-row team card matching the style of
// renderBeaconCard. Includes a public-web Details link so the user can
// click through to manage memberships in the web app.
func renderTeamCard(w io.Writer, apiURL string, team *teams.Team, role string, isActive bool) {
	heading := "  Team"
	if isActive {
		heading = "  Team (active)"
	}
	header := ui.AccentBoldStyle().Render(heading)

	personalLabel := "no"
	if team.Personal {
		personalLabel = "yes"
	}
	roleLabel := teams.FormatRole(role)
	if roleLabel == "" {
		roleLabel = "(unknown)"
	}

	tbl := ui.CompactTable().
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("ID", team.ID).
		Row("Name", team.Name).
		Row("Role", roleLabel).
		Row("Personal", personalLabel).
		Row("Ownership", team.OwnershipModel)

	if team.CreatedAt != "" {
		tbl = tbl.Row("Created", ui.TruncateToSecond(team.CreatedAt))
	}
	if detail := ui.TeamDetailURL(apiURL, team.ID); detail != "" {
		tbl = tbl.Row("Details", detail)
	}

	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

func init() {
	addRecordOutputFlags(teamsGetCmd)
	addRecordOutputFlags(teamsCurrentCmd)

	teamsCmd.AddCommand(teamsGetCmd)
	teamsCmd.AddCommand(teamsCurrentCmd)
}
