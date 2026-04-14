package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/version"
)

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
		return nil
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
}

// Execute runs the root command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
