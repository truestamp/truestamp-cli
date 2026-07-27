// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/console"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Open the interactive Truestamp console (TUI)",
	Long: `Open an authenticated WebSocket console with multiple panes:

  • Monitor    — live waterfall of block, commitment, and entropy events
                 (subscribe/unsubscribe to streams on demand)
  • New Item   — create a timestamped item and watch its lifecycle live
  • Connection — diagnostics, scope, push counts, log file path

Authentication uses your active Truestamp session — an OAuth sign-in (run
'truestamp auth login') or, for headless/CI use, an API key via
TRUESTAMP_API_KEY. The wire protocol is Phoenix Channels' V2 JSON-array
format and is hand-writable from websocat for scripting.

Transport diagnostics (read EOFs, dial-attempt failures during reconnect,
frame decode errors) are written to the same JSON log file every other
truestamp subcommand uses. The path is shown on the Connection pane and
honors the --log-file / --log-level persistent flags.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runConsole,
}

func init() {
	f := consoleCmd.Flags()
	f.String("ws-url", "", "WebSocket URL override (default derived from --base-url)")
	rootCmd.AddCommand(consoleCmd)
}

func runConsole(cmd *cobra.Command, _ []string) error {
	if !authConfigured() {
		return fmt.Errorf("not authenticated — run `truestamp auth login` (or set TRUESTAMP_API_KEY)")
	}

	// Resolve the WebSocket URL. Default comes from base_url via Config;
	// --ws-url stays as an explicit per-invocation escape hatch for
	// debugging or pointing at a non-standard WS path.
	wsURL, _ := cmd.Flags().GetString("ws-url")
	if wsURL == "" {
		wsURL = appConfig.WebSocketURL
	}

	// First-run team picker — only when stdin is a TTY and no team
	// is configured. Non-TTY callers fall through to the existing
	// personal-team-fallback behaviour. The picker writes the chosen
	// team id to config.toml so subsequent invocations skip this
	// path. The picker is opt-out for users who explicitly want the
	// server's personal-team auto-fallback: hitting Esc dismisses
	// the picker without persisting.
	activeTeamID := appConfig.Team
	if activeTeamID == "" && stdinIsTerminal() {
		picked, err := promptForFirstRunTeam(cmd.Context())
		if err != nil {
			return err
		}
		if picked != "" {
			if err := config.SetTeam(picked); err != nil {
				return fmt.Errorf("writing config: %w", err)
			}
			activeTeamID = picked
		}
	}

	// The root PersistentPreRunE has already constructed appLogger,
	// resolved appLogPath, and tagged records with component=console
	// (logging.Options.Component = cmd.Name()). The console subcommand
	// just consumes the result.
	appLogger.Info("console session start",
		"ws_url", wsURL,
		"log_path", appLogPath,
		"version", cmd.Root().Version)

	return console.Run(cmd.Context(), console.Options{
		WSURL:          wsURL,
		Authorizer:     auth.Default(),
		APIURL:         appConfig.APIURL,
		ActiveTeamID:   activeTeamID,
		Logger:         appLogger,
		LogFilePath:    appLogPath,
		ConfigFilePath: config.ActivePath(),
		HealthTargets:  console.DefaultHealthTargets(appConfig.HealthURL, appConfig.KeyringURL),
	})
}

// promptForFirstRunTeam fetches the user's memberships and runs the
// huh picker over them. Returns the picked team id, or the empty
// string when the user cancelled (Esc) or has no memberships. Errors
// only on transport / fetch failures; an empty membership list
// degrades to the empty-string return + a faint stderr hint.
func promptForFirstRunTeam(ctx context.Context) (string, error) {
	pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cfg := teams.Config{
		APIURL: appConfig.APIURL,
	}
	memberships, err := teams.ListMyMemberships(pctx, cfg)
	if err != nil {
		// Soft-fail: the user can still launch the console without
		// configuring a team — the server falls back to the personal
		// team. Surface a faint warning so they know why the picker
		// didn't appear.
		fmt.Fprintln(os.Stderr,
			ui.FaintStyle().Render(
				"  Could not list teams: "+err.Error()+
					" — proceeding with personal-team fallback."))
		return "", nil
	}
	if len(memberships) == 0 {
		fmt.Fprintln(os.Stderr,
			ui.FaintStyle().Render(
				"  No teams found for this API key — proceeding with personal-team fallback."))
		return "", nil
	}

	// Use the same picker the team_set subcommand uses. Returning
	// "" from the picker is a graceful Esc-cancel.
	return pickTeamInteractive(pctx, cfg)
}
