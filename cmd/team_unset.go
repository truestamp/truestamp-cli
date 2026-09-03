// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var teamUnsetCmd = &cobra.Command{
	Use:   "unset",
	Short: "Clear the active team from config.toml",
	Long: `Clear the 'team' key in config.toml. The CLI then sends API requests
without an explicit tenant header; the server falls back to the
caller's personal team.

Most users should prefer 'truestamp team set' over 'unset' so the
team id is always sent explicitly with API requests, that avoids a
server-side personal-team lookup on every call.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runTeamUnset,
}

func runTeamUnset(cmd *cobra.Command, _ []string) error {
	silent, _ := cmd.Flags().GetBool("silent")

	if appConfig.Team == "" {
		if !silent {
			fmt.Fprintln(cmd.OutOrStdout(), ui.FaintStyle().Render(
				"  No active team is configured."))
		}
		return nil
	}

	if err := config.SetTeam(""); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	if silent {
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(),
		ui.SuccessBanner("Active team cleared"))
	fmt.Fprintln(cmd.OutOrStdout(), ui.FaintStyle().Render(
		"    Subsequent API requests will fall back to the personal team."))
	return nil
}

func init() {
	f := teamUnsetCmd.Flags()
	f.BoolP("silent", "s", false, "No output, exit code only")

	teamCmd.AddCommand(teamUnsetCmd)
}
