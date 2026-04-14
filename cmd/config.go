package cmd

import (
	"fmt"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  "View and manage the Truestamp CLI configuration file and resolved settings.",
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the config file path",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		label := lipgloss.NewStyle().Foreground(ui.Label).Render("Config Path")
		value := lipgloss.NewStyle().Foreground(ui.Value).Render(config.ConfigFilePath())
		lipgloss.Println(label + "  " + value)
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default config file if it doesn't exist",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		created, err := config.EnsureDefaultConfig()
		if err != nil {
			return err
		}
		path := config.ConfigFilePath()
		if created {
			lipgloss.Println(ui.SuccessBanner("Created default config"))
			label := lipgloss.NewStyle().Foreground(ui.Label).Render("Path")
			value := lipgloss.NewStyle().Foreground(ui.Value).Render(path)
			lipgloss.Println(label + "  " + value)
		} else {
			lipgloss.Println(ui.FaintStyle().Render("Config already exists at " + path))
		}
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the resolved configuration",
	Long:  "Print the fully resolved configuration after merging defaults, config file, env vars, and CLI flags.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if appConfig == nil {
			return fmt.Errorf("configuration not loaded")
		}
		presentConfig(appConfig)
		return nil
	},
}

func presentConfig(cfg *config.Config) {
	header := ui.HeaderBox("Truestamp CLI Configuration", "Resolved settings")

	general := table.New().
		Row("API URL", cfg.APIURL).
		Row("API Key", maskAPIKey(cfg.APIKey)).
		Row("Team", valueOrNotSet(cfg.Team)).
		Row("Keyring URL", cfg.KeyringURL).
		Row("HTTP Timeout", cfg.HTTPTimeout).
		Border(lipgloss.HiddenBorder()).
		StyleFunc(configStyleFunc)

	verify := table.New().
		Row("Silent", fmt.Sprintf("%v", cfg.Verify.Silent)).
		Row("JSON", fmt.Sprintf("%v", cfg.Verify.JSON)).
		Row("Skip External", fmt.Sprintf("%v", cfg.Verify.SkipExternal)).
		Row("Skip Signatures", fmt.Sprintf("%v", cfg.Verify.SkipSignatures)).
		Row("Remote", fmt.Sprintf("%v", cfg.Verify.Remote)).
		Border(lipgloss.HiddenBorder()).
		StyleFunc(configStyleFunc)

	output := lipgloss.JoinVertical(lipgloss.Left,
		header, "",
		ui.SectionHeader("General"),
		general.String(), "",
		ui.SectionHeader("Verification"),
		verify.String(),
	)
	lipgloss.Println(output)
}

func configStyleFunc(row, col int) lipgloss.Style {
	if col == 0 {
		return lipgloss.NewStyle().
			Foreground(ui.Label).
			Width(18).
			Align(lipgloss.Right).
			PaddingRight(1)
	}
	return lipgloss.NewStyle().Foreground(ui.Value)
}

func maskAPIKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func valueOrNotSet(v string) string {
	if v == "" {
		return "(not set)"
	}
	return v
}

func init() {
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
