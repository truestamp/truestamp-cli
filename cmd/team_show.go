// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var teamShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show the active team in detail (or a specific team by id)",
	Long: `Show the active team — the one currently configured under 'team' in
config.toml — with its name, role, personal flag, ownership model, and
public-web links. With no argument, defaults to the active team.

Pass an explicit team id to inspect a different team. The id must be
one the API key has membership in (the server enforces this; an
attempt to read a team you don't belong to surfaces a 403 banner).

Examples:
  truestamp team show
  truestamp team show 019dbd00-0000-7000-8000-000000000000`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runTeamShow,
}

func runTeamShow(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")
	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

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
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("No team configured"))
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp team set' to pick one interactively, or "+
					"'truestamp team list' to see all available teams."))
		}
		return errSilentFail
	}

	// Use the actor's CURRENT active tenant (cfg.Team) as the tenant
	// header — never the id we're looking up. The server's tenant
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
		// Soft-fail the role lookup — the team detail is the headline
		// info; missing role degrades to "(unknown)" rather than
		// erroring the whole subcommand.
		if !silent && !errors.Is(err, teams.ErrUnauthorized) {
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
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
// renderBeaconCard. Includes two public-web links so the user can
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

	fmt.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

func init() {
	f := teamShowCmd.Flags()
	f.Bool("json", false, "Print the raw JSON response, pretty-printed")
	f.BoolP("silent", "s", false, "No output, exit code only")

	teamCmd.AddCommand(teamShowCmd)
}
