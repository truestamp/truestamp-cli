// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package cmd wires up the cobra command tree for the Truestamp CLI. The
// main entrypoint (cmd/truestamp/main.go) calls [Execute]; everything else
// here registers subcommands, flags, and the shared PersistentPreRunE that
// loads the resolved configuration into [appConfig].
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/upgradecheck"
	"github.com/truestamp/truestamp-cli/internal/version"
)

// errSilentFail signals a failure that should exit non-zero without any
// output. Commands use it for modes like `verify --silent` where the user
// has explicitly asked for no output.
var errSilentFail = errors.New("silent failure")

// appConfig holds the resolved configuration, available to all subcommands.
var appConfig *config.Config

// configFile is the --config flag value (overrides default path).
var configFile string

var rootCmd = &cobra.Command{
	Use:     "truestamp",
	Short:   "Truestamp CLI — tools for cryptographic timestamping",
	Long:    "Truestamp CLI — tools for cryptographic timestamping\n\n" + version.Copyright(),
	Version: version.Short(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for completion/help (no config/HTTP needed).
		// Check os.Args directly: cobra's __complete command has
		// DisableFlagParsing=true, so the command hierarchy may not be
		// fully initialized when PersistentPreRunE fires.
		if len(os.Args) > 1 {
			switch os.Args[1] {
			case "completion", "__complete", "__completeNoDesc", "help":
				return nil
			}
		}

		noColor, _ := cmd.Flags().GetBool("no-color")
		ui.Init(noColor)

		cfg, err := config.Load(configFile, cmd.Flags())
		if err != nil {
			return err
		}
		appConfig = cfg
		httpclient.Init(cfg.Timeout())
		httpclient.SetUserAgent(version.Version)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		maybeEmitUpgradeNotice(cmd)
	},
}

func init() {
	rootCmd.SetVersionTemplate(version.Full() + "\n")

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to config file (default: "+config.ConfigFilePath()+")")
	rootCmd.PersistentFlags().String("api-url", "", "Base URL of the Truestamp API")
	rootCmd.PersistentFlags().String("api-key", "", "API key for authenticating with the Truestamp API")
	rootCmd.PersistentFlags().String("team", "", "Team ID for multi-tenant API operations")
	rootCmd.PersistentFlags().String("keyring-url", "", "URL of the Truestamp keyring endpoint")
	rootCmd.PersistentFlags().String("http-timeout", "", "HTTP timeout for external API calls (e.g. 10s, 30s, 1m)")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable color output")
	rootCmd.PersistentFlags().Bool("no-upgrade-check", false, "Disable the once-per-day 'new version available' notice")
}

// maybeEmitUpgradeNotice runs after any successful subcommand and may
// write a one-line "upgrade available" notice to stderr. It is a no-op
// when the upgrade-check is disabled (flag, env var, CI, non-TTY), when
// the invoked subcommand is itself about upgrades/version/help, or when
// no newer release is found. All failures are swallowed; this path is
// never load-bearing.
func maybeEmitUpgradeNotice(cmd *cobra.Command) {
	// Only run for subcommands where a nag wouldn't be annoying. Also
	// skip for the root command itself (help output) and for hidden
	// completion helpers.
	name := cmd.Name()
	switch name {
	case "truestamp", "upgrade", "version", "help", "completion",
		"__complete", "__completeNoDesc":
		return
	}

	// Don't emit on non-TTY or CI — the Disabled() check handles those.
	flagDisabled, _ := cmd.Flags().GetBool("no-upgrade-check")
	upgradecheck.MaybeNotify(cmd.ErrOrStderr(), flagDisabled, version.Version)
}

// Execute runs the root command. Commands set SilenceErrors so cobra does
// not print their errors; Execute is the single place errors reach stderr.
// A command that needs silent-on-error UX (e.g. `verify --silent`) returns
// errSilentFail instead of the real error to opt out of printing. The
// upgrade --check flow uses exitCodeErr to return a specific exit code
// without also printing an error line.
func Execute() error {
	err := rootCmd.Execute()
	if err == nil {
		return nil
	}
	var ec exitCodeErr
	if errors.As(err, &ec) {
		return err
	}
	if !errors.Is(err, errSilentFail) {
		fmt.Fprintln(os.Stderr, err)
	}
	return err
}
