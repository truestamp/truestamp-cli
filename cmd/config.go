// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
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
	Long: `Print the path of the config file in effect, the --config override
when one was supplied, otherwise the platform default, and whether
that file currently exists.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.ActivePath()
		_, statErr := os.Stat(path)
		exists := statErr == nil

		// `path` renders a record, so it carries the CLI-wide output
		// contract like every other record command. It is also the one
		// place where --json earns its keep twice over: the styled form
		// puts a label on stdout, so the `$(truestamp config path)` idiom
		// captures "Config Path  /path", not the path. `--json | jq -r
		// .path` captures the path and nothing else.
		jsonOut, silent := outputMode(cmd)
		switch {
		case silent:
			return nil
		case jsonOut:
			return emitJSON(cmd.OutOrStdout(), map[string]any{
				"path":   path,
				"exists": exists,
			})
		}

		label := lipgloss.NewStyle().Foreground(ui.Label).Render("Config Path")
		value := lipgloss.NewStyle().Foreground(ui.Value).Render(path)
		lipgloss.Println(label + "  " + value)
		// Existence goes to stderr so stdout stays a single line: this
		// command is routinely captured with `$(truestamp config path)`,
		// and a second stdout line would land inside the captured value.
		ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(configPathStatusIndent+configPathStatus(exists)))
		return nil
	},
}

// configPathStatusIndent aligns the status line under the path value:
// len("Config Path") + the two-space gap.
const configPathStatusIndent = "             "

// configPathStatus reports whether the config file in effect exists.
// A missing file is not an error, the CLI runs on compiled defaults,
// so the message points at the command that would create it.
func configPathStatus(exists bool) string {
	if exists {
		return "exists"
	}
	return "does not exist, run 'truestamp config init' to create it"
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default config file if it doesn't exist",
	Long: `Create the config file in effect from the embedded defaults if it does
not already exist. With --config the file is created at that path
instead of the platform default.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		created, err := config.EnsureDefaultConfig()
		if err != nil {
			return err
		}
		path := config.ActivePath()
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
		jsonOut, silent := outputMode(cmd)
		if silent {
			return nil
		}
		if jsonOut {
			return emitJSON(cmd.OutOrStdout(), newConfigRecord(appConfig))
		}
		presentConfig(appConfig)
		return nil
	},
}

// configRecord is the --json shape of `config show`: the resolved Config
// itself, serialised under its config-file key names, plus the two facts
// that are not config values (which file was in effect, which credential
// mode won). It embeds the struct rather than listing its fields, so a
// field added to Config appears here without anyone remembering to add
// it. The API key is masked exactly as it is in the text rendering:
// `config show --json` is the form a script or an agent reaches for,
// which is the last place a secret should become easy to exfiltrate.
type configRecord struct {
	ConfigFile string `json:"config_file"`
	AuthMode   string `json:"auth_mode"`
	*config.Config
}

func newConfigRecord(cfg *config.Config) configRecord {
	masked := *cfg
	masked.APIKey = maskAPIKey(cfg.APIKey)
	masked.Verify.Horizon = maskUserinfo(cfg.Verify.Horizon)
	masked.Verify.Esplora = maskUserinfo(cfg.Verify.Esplora)
	masked.Verify.BitcoinCore = maskUserinfo(cfg.Verify.BitcoinCore)
	masked.Verify.Sources = map[string]map[string]config.NetworkSourcesConfig{}
	for chain, networks := range cfg.Verify.Sources {
		masked.Verify.Sources[chain] = map[string]config.NetworkSourcesConfig{}
		for network, n := range networks {
			sources := make([]map[string]string, 0, len(n.Sources))
			for _, entry := range n.Sources {
				shown := map[string]string{}
				for provider, raw := range entry {
					shown[provider] = maskUserinfo(raw)
				}
				sources = append(sources, shown)
			}
			masked.Verify.Sources[chain][network] = config.NetworkSourcesConfig{Policy: n.Policy, Sources: sources}
		}
	}
	return configRecord{ConfigFile: config.ActivePath(), AuthMode: authModeDisplay(), Config: &masked}
}

// maskUserinfo is a source URL as config show shows it: whole, but for
// anything that could be user information, which the verifier refuses in a
// source URL and which would be a credential. Any @ after the scheme's //
// could end user information, even one a parser reads as part of a path,
// query or fragment (a password typed with a raw slash, as in
// http://u:12/ss@host), so everything before the last @, but the scheme,
// is shown as [masked]. An @ in a path masks more than it must, which
// costs only display. A Bitcoin Core node's RPC credentials are never
// shown at all.
func maskUserinfo(raw string) string {
	at := strings.LastIndex(raw, "@")
	if at < 0 {
		return raw
	}
	scheme := ""
	if i := strings.Index(raw, "://"); i >= 0 && i < at {
		scheme = raw[:i+3]
	}
	return scheme + "[masked]@" + raw[at+1:]
}

// sourceRows are the verifier's sources as config show's text shows them:
// each source URL, and each network's several sources with their policy.
func sourceRows(cfg *config.Config) [][2]string {
	rows := [][2]string{
		{"Horizon", valueOrNotSet(maskUserinfo(cfg.Verify.Horizon))},
		{"Esplora", valueOrNotSet(maskUserinfo(cfg.Verify.Esplora))},
		{"Bitcoin Core", valueOrNotSet(maskUserinfo(cfg.Verify.BitcoinCore))},
	}
	for _, chain := range slices.Sorted(maps.Keys(cfg.Verify.Sources)) {
		for _, network := range slices.Sorted(maps.Keys(cfg.Verify.Sources[chain])) {
			n := cfg.Verify.Sources[chain][network]
			var listed []string
			for _, entry := range n.Sources {
				for _, provider := range slices.Sorted(maps.Keys(entry)) {
					listed = append(listed, provider+" "+maskUserinfo(entry[provider]))
				}
			}
			rows = append(rows, [2]string{chain + " " + network, n.Policy + ": " + strings.Join(listed, ", ")})
		}
	}
	for _, k := range []struct{ name, path string }{
		{"Keyring (dev)", cfg.Verify.Keyrings.Dev},
		{"Keyring (staging)", cfg.Verify.Keyrings.Staging},
		{"Keyring (test)", cfg.Verify.Keyrings.Test},
	} {
		rows = append(rows, [2]string{k.name, valueOrNotSet(k.path)})
	}
	return rows
}

func presentConfig(cfg *config.Config) {
	header := ui.HeaderBox("Truestamp CLI Configuration", "Resolved settings")

	general := ui.CompactTable().
		StyleFunc(configStyleFunc).
		// The file these values were resolved from. Without it there is
		// no way to tell from the output whether a --config override
		// took effect.
		Row("Config File", config.ActivePath()).
		Row("API URL", cfg.APIURL).
		Row("Auth Mode", authModeDisplay()).
		Row("API Key", maskAPIKey(cfg.APIKey)).
		Row("Team", valueOrNotSet(cfg.Team))

	// Resolve the team's name + role online when a credential (an OAuth
	// session or an API key) and a team id are both configured.
	// Best-effort: a network or auth failure suppresses the extra rows
	// in favor of a faint hint, so `config show` stays useful when
	// offline. Capped to half the configured HTTP timeout so the
	// command stays snappy.
	if authConfigured() && cfg.Team != "" {
		general = appendTeamDetailRows(general, cfg)
	}

	general = general.
		Row("Keys keyring", cfg.KeyringURL).
		Row("HTTP Timeout", cfg.HTTPTimeout).
		Row("Cosign Path", cosignPathDisplay(cfg.CosignPath)).
		Row("Silent", fmt.Sprintf("%v", cfg.Silent)).
		Row("JSON", fmt.Sprintf("%v", cfg.JSON))

	verify := ui.CompactTable().
		StyleFunc(configStyleFunc).
		Row("Offline", fmt.Sprintf("%v", cfg.Verify.Offline)).
		Row("Remote", fmt.Sprintf("%v", cfg.Verify.Remote))
	for _, row := range sourceRows(cfg) {
		verify = verify.Row(row[0], row[1])
	}

	hash := ui.CompactTable().
		StyleFunc(configStyleFunc).
		Row("Algorithm", cfg.Hash.Algorithm).
		Row("Encoding", cfg.Hash.Encoding).
		Row("Style", cfg.Hash.Style)

	convert := ui.CompactTable().
		StyleFunc(configStyleFunc).
		Row("Time Zone", valueOrNotSet(cfg.Convert.TimeZone))

	// Plain newline-join, NOT lipgloss.JoinVertical. JoinVertical pads
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
		return "none, run 'truestamp auth login'"
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
// Empty means "use $PATH lookup", distinct from "not set" for other
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
// the whole output. The function only runs when a credential (an
// OAuth session or an API key) and a team id are configured; the
// caller gates that via authConfigured().
func appendTeamDetailRows(tbl *table.Table, cfg *config.Config) *table.Table {
	ctx, cancel := context.WithTimeout(context.Background(), configTeamLookupTimeout(cfg))
	defer cancel()

	clientCfg := teams.Config{APIURL: cfg.APIURL, Team: cfg.Team}

	team, err := teams.GetTeam(ctx, clientCfg, cfg.Team)
	if err != nil {
		// Faint hint row, keep the column count consistent so the
		// table stylefunc doesn't have to special-case empty cells.
		return tbl.Row("Team Name", "(unavailable, try 'truestamp auth status')")
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
	addRecordOutputFlags(configShowCmd)
	addRecordOutputFlags(configPathCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.GroupID = groupSetup
	rootCmd.AddCommand(asGroup(configCmd))
}
