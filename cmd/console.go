// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/console"
	"github.com/truestamp/truestamp-cli/internal/logging"
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
frame decode errors) are written to a rotated log file rather than the
UI. The path is shown on the Connection pane.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runConsole,
}

func init() {
	f := consoleCmd.Flags()
	f.String("ws-url", "", "WebSocket URL (default derived from --api-url)")
	f.String("log-level", "info", "Log level: debug, info, warn, error")
	f.String("log-file", "", "Override log file path (default: "+logging.DefaultPath()+")")
	rootCmd.AddCommand(consoleCmd)
}

func runConsole(cmd *cobra.Command, _ []string) error {
	if appConfig.APIKey == "" {
		return fmt.Errorf("no API key configured — run `truestamp auth login` or set TRUESTAMP_API_KEY")
	}

	wsURL, _ := cmd.Flags().GetString("ws-url")
	if wsURL == "" {
		wsURL = deriveWSURL(appConfig.APIURL)
	}

	logLevel, _ := cmd.Flags().GetString("log-level")
	logFile, _ := cmd.Flags().GetString("log-file")

	logger, logPath, err := logging.New(logging.Options{
		Path:  logFile,
		Level: logLevel,
	})
	if err != nil {
		// Non-fatal: continue with a discard logger. The TUI still
		// runs; the user just won't have a postmortem trail.
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: log file disabled: %v\n", err)
	}

	logger.Info("console session start",
		"ws_url", wsURL,
		"log_path", logPath,
		"version", cmd.Root().Version)

	return console.Run(cmd.Context(), console.Options{
		WSURL:       wsURL,
		APIKey:      appConfig.APIKey,
		Logger:      logger,
		LogFilePath: logPath,
	})
}

// deriveWSURL maps an HTTP(S) API base URL to the corresponding WS(S)
// console endpoint. The transformations:
//
//	https://www.truestamp.com/api/json   →   wss://www.truestamp.com/console/websocket
//	http://localhost:4000/api/json       →   ws://localhost:4000/console/websocket
//	https://example.com                  →   wss://example.com/console/websocket
//
// On parse failure, returns the empty string — the caller will surface a
// clearer error from the WebSocket dial than from a synthetic guess.
func deriveWSURL(apiURL string) string {
	u, err := url.Parse(apiURL)
	if err != nil {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return ""
	}
	u.Path = "/console/websocket"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}
