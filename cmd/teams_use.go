// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var teamsUseCmd = &cobra.Command{
	Use:   "use [id]",
	Short: "Point the CLI at a team (interactive picker if no id given)",
	Long: `Set the active team that the CLI sends with API requests, and persist
it under the top-level 'team' key in config.toml.

With an explicit id, validates by reading the team from the API
(rejects 401/403/404 before writing) and then persists. Without an id,
opens an interactive picker populated with the teams the API key has
membership in. Selecting "Personal" stores the personal team's UUID
explicitly so subsequent requests carry the tenant header instead of
relying on the server's personal-team auto-fallback.

Examples:
  truestamp teams use
  truestamp teams use 019dbd00-0000-7000-8000-000000000000`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runTeamsUse,
}

func runTeamsUse(cmd *cobra.Command, args []string) error {
	_, silent := outputMode(cmd)

	// --clear is the former `team unset`. It lives here rather than as its
	// own leaf because a `teams clear` would read as "delete all my teams",
	// a far worse failure in a group whose other verbs touch server
	// records. The wart is that `use --clear` reads as "use nothing".
	clear, _ := cmd.Flags().GetBool("clear")
	if clear {
		if len(args) > 0 {
			return fmt.Errorf("--clear takes no team id: it points the CLI at no team")
		}
		return clearActiveTeam(cmd, silent)
	}

	cfg, err := teamConfig(cmd)
	if err != nil {
		return err
	}

	var targetID string
	if len(args) > 0 {
		targetID = strings.TrimSpace(args[0])
	}

	if targetID == "" {
		// Interactive picker. Refuse early when stdin isn't a TTY so
		// scripted callers get a clear error instead of a hang.
		if !stdinIsTerminal() {
			return fmt.Errorf("interactive picker requires a TTY; pass an explicit id (truestamp teams use <id>)")
		}
		picked, err := pickTeamInteractive(cmd.Context(), cfg)
		if err != nil {
			return teamRenderError(cmd, err, silent)
		}
		if picked == "" {
			// User cancelled the picker (Esc). Treat as a no-op silent
			// success, exit 0 so chained commands don't see a failure.
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render("  Cancelled."))
			return nil
		}
		targetID = picked
	} else {
		// Explicit id: validate by hitting the API before persisting.
		// Tenant header stays at the actor's current active team
		// (cfg.Team), never the lookup target, so the server's
		// tenant-resolution check doesn't 403 us before the path
		// handler can answer "team not found / not a member".
		if _, err := teams.GetTeam(cmd.Context(), cfg, targetID); err != nil {
			return teamRenderError(cmd, err, silent)
		}
	}

	if err := config.SetTeam(targetID); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	if silent {
		return nil
	}

	// Re-fetch the team and role for the confirmation card. The
	// targetID is now the active team (config persisted), so it's
	// valid to use as the tenant header on these reads. Soft-fail
	// the role lookup so a degraded server-side response doesn't
	// suppress the success confirmation.
	postSwitchCfg := teams.Config{APIURL: cfg.APIURL, Team: targetID}
	team, err := teams.GetTeam(cmd.Context(), postSwitchCfg, targetID)
	if err != nil {
		ui.Fprintln(cmd.OutOrStdout(),
			ui.SuccessBanner("Active team set to "+targetID))
		return nil
	}
	role, _ := teams.GetMyRoleOnTeam(cmd.Context(), postSwitchCfg, targetID)

	ui.Fprintln(cmd.OutOrStdout(),
		ui.SuccessBanner("Active team updated"))
	renderTeamCard(cmd.OutOrStdout(), appConfig.APIURL, team, role, true)
	ui.Fprintln(cmd.OutOrStdout())
	ui.Fprintln(cmd.OutOrStdout(), ui.FaintStyle().Render(
		"  Stored under 'team' in "+config.ActivePath()+"."))
	return nil
}

// pickTeamInteractive runs a huh.Select over the user's memberships
// and returns the picked team id, or the empty string if the user
// cancelled. Errors only on transport / fetch failures.
func pickTeamInteractive(ctx context.Context, cfg teams.Config) (string, error) {
	memberships, err := fetchMyMembershipsCtx(ctx, cfg)
	if err != nil {
		return "", err
	}
	if len(memberships) == 0 {
		fmt.Println()
		fmt.Println(ui.FaintStyle().Render(
			"  No teams available. Visit " +
				ui.TeamCreateURL(cfg.APIURL) + " to create one."))
		return "", nil
	}

	// Same sort as `team list` so the picker matches the table the
	// user just saw: Personal first (one per user), then Owner →
	// Admin → Member → Viewer, alphabetical within each rank.
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

	options := make([]huh.Option[string], 0, len(memberships))
	for _, m := range memberships {
		label := teamDisplayName(m) + "  ·  " + teams.FormatRole(m.Role)
		if m.Team != nil && m.Team.Personal {
			label = "Personal  ·  " + teams.FormatRole(m.Role)
		}
		options = append(options, huh.NewOption(label, m.TeamID))
	}

	var picked string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Pick a team").
				Description("Selection persists under 'team' in config.toml.").
				Options(options...).
				Value(&picked),
		),
	).WithTheme(ui.HuhTheme())

	if err := form.Run(); err != nil {
		// huh returns a sentinel for ESC cancellation in some versions;
		// treat any error from Run() as cancel for UX simplicity (the
		// user can re-run the command). Transport errors above are
		// already surfaced before this point.
		return "", nil
	}
	return picked, nil
}

func init() {
	f := teamsUseCmd.Flags()
	f.Bool("clear", false, "Point the CLI at no team; the server falls back to your personal team")
	addRecordOutputFlags(teamsUseCmd)
	_ = f

	teamsCmd.AddCommand(teamsUseCmd)
}

// clearActiveTeam implements `teams use --clear`, formerly `team unset`.
func clearActiveTeam(cmd *cobra.Command, silent bool) error {
	if appConfig.Team == "" {
		if !silent {
			ui.Fprintln(cmd.OutOrStdout(), ui.FaintStyle().Render(
				"  No team is configured."))
		}
		return nil
	}

	if err := config.SetTeam(""); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	if silent {
		return nil
	}

	ui.Fprintln(cmd.OutOrStdout(),
		ui.SuccessBanner("Team cleared"))
	ui.Fprintln(cmd.OutOrStdout(), ui.FaintStyle().Render(
		"    Subsequent API requests will fall back to the personal team."))
	return nil
}
