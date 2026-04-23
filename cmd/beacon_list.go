// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"io"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/beacons"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

const beaconListDefaultLimit = 25

var beaconListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the most recent beacons (newest first)",
	Long: `List recent beacons, newest first. The server caps --limit at 100.

Examples:
  truestamp beacon list
  truestamp beacon list --limit 3
  truestamp beacon list --limit 10 --json | jq '.[].hash'`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runBeaconList,
}

func runBeaconList(cmd *cobra.Command, _ []string) error {
	jsonOut, hashOnly, silent, err := beaconSharedFlags(cmd)
	if err != nil {
		return err
	}
	if hashOnly {
		return fmt.Errorf("--hash-only is not valid on 'beacon list' (use --json and pipe to jq)")
	}

	limit, _ := cmd.Flags().GetInt("limit")
	if limit == 0 {
		limit = beaconListDefaultLimit
	}

	cfg, err := beaconConfig(cmd)
	if err != nil {
		return err
	}

	items, err := beacons.List(cmd.Context(), cfg, limit)
	if err != nil {
		return beaconRenderError(cmd, err, silent)
	}

	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), items)
	}
	renderBeaconList(cmd.OutOrStdout(), items)
	// One-line hint on interactive runs pointing at `download --type beacon`.
	// Suppressed when stdout is piped so shell pipelines stay clean.
	if stdoutIsTerminal() {
		fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
			"  Hint: 'truestamp download --type beacon <id>' fetches a verifiable proof bundle."))
	}
	return nil
}

// renderBeaconList prints a compact three-column table. Hashes are
// always shown full-width — truncation would silently drop the bytes a
// user came here to capture (the whole point of `beacon list` is to
// surface the hash for copy-paste or shell substitution).
func renderBeaconList(w io.Writer, items []beacons.Beacon) {
	heading := fmt.Sprintf("  Beacons (latest %d)", len(items))
	header := ui.AccentBoldStyle().Render(heading)

	rows := make([][]string, 0, len(items)+1)
	rows = append(rows, []string{"TIMESTAMP", "HASH", "ID"})
	for _, b := range items {
		// Drop fractional-second precision for readability. The full
		// precision is preserved in --json output; this only affects
		// the human-readable table.
		rows = append(rows, []string{ui.TruncateToSecond(b.Timestamp), b.Hash, b.ID})
	}

	tbl := ui.CompactTable().
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Foreground(ui.Label).PaddingLeft(2).PaddingRight(1).Bold(true)
			}
			return lipgloss.NewStyle().Foreground(ui.Value).PaddingLeft(2).PaddingRight(1)
		}).
		Rows(rows...)

	// Plain newline-join — see note in internal/verify/presenter.go
	// Present(). Avoids lipgloss.JoinVertical's pad-to-widest behaviour,
	// which would make long hash rows blow up vertical spacing on
	// narrow terminals.
	fmt.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

func init() {
	f := beaconListCmd.Flags()
	f.Int("limit", beaconListDefaultLimit, "How many beacons to fetch (1..100)")
	f.Bool("json", false, "Print the raw JSON response, pretty-printed")
	f.Bool("hash-only", false, "(invalid on 'list' — use --json + jq)")
	f.BoolP("silent", "s", false, "No output, exit code only")

	beaconCmd.AddCommand(beaconListCmd)
}
