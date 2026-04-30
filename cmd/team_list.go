// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all teams the API key has membership in",
	Long: `List the teams the current API key is a member of, with the user's role
in each. The active team (the one stored under 'team' in config.toml)
is marked with a star.

Examples:
  truestamp team list
  truestamp team list --json | jq '.[].id'`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runTeamList,
}

func runTeamList(cmd *cobra.Command, _ []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")
	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

	cfg, err := teamConfig(cmd)
	if err != nil {
		return err
	}

	memberships, err := fetchMyMembershipsCtx(cmd.Context(), cfg)
	if err != nil {
		return teamRenderError(cmd, err, silent)
	}

	// Personal team always goes first (there's exactly one per user
	// and it's the one most users want at a glance). Everything else
	// sorts by privilege rank — Owner → Admin → Member → Viewer —
	// then alphabetical by name within each rank. Stable order so
	// pipes and screenshots are reproducible across runs.
	sort.SliceStable(memberships, func(i, j int) bool {
		ip := memberships[i].Team != nil && memberships[i].Team.Personal
		jp := memberships[j].Team != nil && memberships[j].Team.Personal
		if ip != jp {
			return ip
		}
		ri, rj := teams.PrivilegeRank(memberships[i].Role), teams.PrivilegeRank(memberships[j].Role)
		if ri != rj {
			return ri < rj
		}
		return teamDisplayName(memberships[i]) < teamDisplayName(memberships[j])
	})

	if silent {
		if len(memberships) == 0 {
			return errSilentFail
		}
		return nil
	}

	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), memberships)
	}

	if len(memberships) == 0 {
		renderEmptyTeamList(cmd.OutOrStdout(), appConfig.APIURL)
		return nil
	}

	renderTeamList(cmd.OutOrStdout(), memberships, appConfig.Team)

	if stdoutIsTerminal() {
		hint := "  Hint: 'truestamp team set <id>' switches the active team."
		if appConfig.Team == "" {
			hint += "  No active team is currently set."
		} else {
			hint += "  ★ marks the current selection."
		}
		fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(hint))
	}
	return nil
}

// teamDisplayName returns the team's name (or the team id when name
// isn't available — happens when ?include=team didn't return the
// expected attributes).
func teamDisplayName(m teams.Membership) string {
	if m.Team != nil && m.Team.Name != "" {
		return m.Team.Name
	}
	return m.TeamID
}

func renderEmptyTeamList(w io.Writer, apiURL string) {
	header := ui.AccentBoldStyle().Render("  Teams")
	body := []string{
		header,
		"",
		"  " + ui.FaintStyle().Render("No teams found for this API key."),
	}
	if url := ui.TeamCreateURL(apiURL); url != "" {
		body = append(body,
			"  "+ui.FaintStyle().Render("Visit "+url+" in your browser to create one."),
		)
	}
	fmt.Fprintln(w, strings.Join(body, "\n"))
}

// renderTeamList prints a four-column table: active marker, Name,
// Role, Team ID. Name + Role lead because they're the fields users
// actually scan; the id is the precise selector you copy-paste into
// `truestamp team set <id>` and lives at the right where it doesn't
// crowd the readable columns.
func renderTeamList(w io.Writer, memberships []teams.Membership, activeTeamID string) {
	heading := fmt.Sprintf("  Teams (%d)", len(memberships))
	header := ui.AccentBoldStyle().Render(heading)

	rows := make([][]string, 0, len(memberships)+1)
	rows = append(rows, []string{"", "NAME", "ROLE", "TEAM ID"})
	for _, m := range memberships {
		marker := " "
		if m.TeamID == activeTeamID && activeTeamID != "" {
			marker = "★"
		}
		name := teamDisplayName(m)
		if m.Team != nil && m.Team.Personal {
			name += " (personal)"
		}
		rows = append(rows, []string{marker, name, teams.FormatRole(m.Role), m.TeamID})
	}

	tbl := ui.CompactTable().
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().PaddingLeft(2).PaddingRight(1)
			if row == 0 {
				return base.Foreground(ui.Label).Bold(true)
			}
			// Highlight the active row in the accent color so the eye
			// lands on it even on a long list.
			if col == 0 && row > 0 && rows[row][0] == "★" {
				return base.Foreground(ui.Accent).Bold(true)
			}
			return base.Foreground(ui.Value)
		}).
		Rows(rows...)

	fmt.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

func init() {
	for _, c := range []*cobra.Command{teamCmd, teamListCmd} {
		f := c.Flags()
		f.Bool("json", false, "Print the raw JSON response, pretty-printed")
		f.BoolP("silent", "s", false, "No output, exit code only")
	}
	teamCmd.AddCommand(teamListCmd)
}
