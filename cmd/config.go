// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/teams"
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

	general := ui.CompactTable().
		StyleFunc(configStyleFunc).
		Row("API URL", cfg.APIURL).
		Row("Auth Mode", authModeDisplay()).
		Row("API Key", maskAPIKey(cfg.APIKey)).
		Row("Team", valueOrNotSet(cfg.Team))

	// Resolve the team's name + role online when both an api key and a
	// team id are configured. Best-effort: a network or auth failure
	// suppresses the extra rows in favor of a faint hint, so `config
	// show` stays useful when offline. Capped to half the configured
	// HTTP timeout so the command stays snappy.
	if authConfigured() && cfg.Team != "" {
		general = appendTeamDetailRows(general, cfg)
	}

	general = general.
		Row("Keyring URL", cfg.KeyringURL).
		Row("HTTP Timeout", cfg.HTTPTimeout).
		Row("Cosign Path", cosignPathDisplay(cfg.CosignPath))

	verify := ui.CompactTable().
		StyleFunc(configStyleFunc).
		Row("Silent", fmt.Sprintf("%v", cfg.Verify.Silent)).
		Row("JSON", fmt.Sprintf("%v", cfg.Verify.JSON)).
		Row("Skip External", fmt.Sprintf("%v", cfg.Verify.SkipExternal)).
		Row("Skip Signatures", fmt.Sprintf("%v", cfg.Verify.SkipSignatures)).
		Row("Remote", fmt.Sprintf("%v", cfg.Verify.Remote))

	hash := ui.CompactTable().
		StyleFunc(configStyleFunc).
		Row("Algorithm", cfg.Hash.Algorithm).
		Row("Encoding", cfg.Hash.Encoding).
		Row("Style", cfg.Hash.Style)

	convert := ui.CompactTable().
		StyleFunc(configStyleFunc).
		Row("Time Zone", valueOrNotSet(cfg.Convert.TimeZone))

	// Plain newline-join — NOT lipgloss.JoinVertical. JoinVertical pads
	// every line to the widest line across all inputs; if any single
	// value (URL, long path) exceeds terminal width, the padding
	// inflates every row and the terminal hard-wraps, producing phantom
	// blank lines after every table row. See the matching note in
	// internal/verify/presenter.go Present().
	output := strings.Join([]string{
		header, "",
		ui.SectionHeader("General"),
		general.String(), "",
		ui.SectionHeader("Verification"),
		verify.String(), "",
		ui.SectionHeader("Hash"),
		hash.String(), "",
		ui.SectionHeader("Convert"),
		convert.String(),
	}, "\n")
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

// authModeDisplay renders the active credential mode for `config show`.
func authModeDisplay() string {
	switch auth.Default().Mode() {
	case auth.ModeOAuth:
		return "OAuth (browser sign-in)"
	case auth.ModeAPIKey:
		return "API key"
	default:
		return "none — run 'truestamp auth login'"
	}
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

// cosignPathDisplay renders the resolved cosign_path for `config show`.
// Empty means "use $PATH lookup" — distinct from "not set" for other
// string fields because zero-value here has a documented meaning.
func cosignPathDisplay(v string) string {
	if v == "" {
		return "(auto: $PATH lookup)"
	}
	return v
}

// configTeamLookupTimeout caps the wall time spent resolving team
// name + role in `config show`. Half the configured HTTP timeout
// (10s default) → 5s default, which is short enough that an offline
// or wedged server doesn't make `config show` feel broken.
func configTeamLookupTimeout(cfg *config.Config) time.Duration {
	d := cfg.Timeout() / 2
	if d <= 0 {
		return 5 * time.Second
	}
	return d
}

// appendTeamDetailRows tries to fetch the configured team's name +
// role and append them as extra rows. On any error (network, auth,
// not-found) it appends a single faint hint row instead of breaking
// the whole output. The function only runs when both api_key and
// team are set; the caller gates that.
func appendTeamDetailRows(tbl *table.Table, cfg *config.Config) *table.Table {
	ctx, cancel := context.WithTimeout(context.Background(), configTeamLookupTimeout(cfg))
	defer cancel()

	clientCfg := teams.Config{APIURL: cfg.APIURL, Team: cfg.Team}

	team, err := teams.GetTeam(ctx, clientCfg, cfg.Team)
	if err != nil {
		// Faint hint row — keep the column count consistent so the
		// table stylefunc doesn't have to special-case empty cells.
		return tbl.Row("Team Name", "(unavailable — try 'truestamp auth status')")
	}
	tbl = tbl.Row("Team Name", teamNameWithPersonal(team))

	role, err := teams.GetMyRoleOnTeam(ctx, clientCfg, cfg.Team)
	if err != nil || role == "" {
		return tbl.Row("Team Role", "(unavailable)")
	}
	return tbl.Row("Team Role", teams.FormatRole(role))
}

func teamNameWithPersonal(t *teams.Team) string {
	if t == nil {
		return "(unknown)"
	}
	if t.Personal {
		return t.Name + " (personal)"
	}
	return t.Name
}

func init() {
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
