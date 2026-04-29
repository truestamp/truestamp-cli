// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/console"
)

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Open the interactive Truestamp console (TUI)",
	Long: `Open an authenticated WebSocket console with multiple panes:

  • Monitor    — live waterfall of block, commitment, and entropy events
                 (subscribe/unsubscribe to streams on demand)
  • New Item   — create a timestamped item and watch its lifecycle live
  • Connection — diagnostics, scope, push counts, log file path

Authentication uses your existing API key (same key used by all other
truestamp commands). The wire protocol is Phoenix Channels' V2 JSON-array
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
	if appConfig.APIKey == "" {
		return fmt.Errorf("no API key configured — run `truestamp auth login` or set TRUESTAMP_API_KEY")
	}

	// Resolve the WebSocket URL. Default comes from base_url via Config;
	// --ws-url stays as an explicit per-invocation escape hatch for
	// debugging or pointing at a non-standard WS path.
	wsURL, _ := cmd.Flags().GetString("ws-url")
	if wsURL == "" {
		wsURL = appConfig.WebSocketURL
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
		APIKey:         appConfig.APIKey,
		Logger:         appLogger,
		LogFilePath:    appLogPath,
		ConfigFilePath: config.ConfigFilePath(),
		HealthTargets:  console.DefaultHealthTargets(appConfig.HealthURL, appConfig.KeyringURL),
	})
}
