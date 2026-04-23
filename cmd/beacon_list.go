// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"io"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
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

// renderBeaconList prints a compact three-column table. Hashes truncate on
// narrow terminals; full hashes are emitted when stdout is not a TTY so
// piped output stays byte-stable for downstream tools.
func renderBeaconList(w io.Writer, items []beacons.Beacon) {
	heading := fmt.Sprintf("  Beacons (latest %d)", len(items))
	header := ui.AccentBoldStyle().Render(heading)

	// Truncate hashes only when writing to a live terminal. When piped
	// (non-TTY), emit the full hash so downstream tools stay byte-stable.
	showFullHash := !stdoutIsTerminal()

	rows := make([][]string, 0, len(items)+1)
	rows = append(rows, []string{"TIMESTAMP", "HASH", "ID"})
	for _, b := range items {
		hashCell := b.Hash
		if !showFullHash {
			hashCell = truncateHashShort(b.Hash)
		}
		rows = append(rows, []string{b.Timestamp, hashCell, b.ID})
	}

	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Foreground(ui.Label).PaddingLeft(2).PaddingRight(1).Bold(true)
			}
			return lipgloss.NewStyle().Foreground(ui.Value).PaddingLeft(2).PaddingRight(1)
		}).
		Rows(rows...)

	fmt.Fprintln(w, lipgloss.JoinVertical(lipgloss.Left, header, "", tbl.String()))
}

// truncateHashShort returns the first 12 chars of a hash + ellipsis, for
// narrow TTYs. When stdout isn't a TTY, callers show the full hash.
func truncateHashShort(h string) string {
	if len(h) <= 13 {
		return h
	}
	return h[:12] + "…"
}

func init() {
	f := beaconListCmd.Flags()
	f.Int("limit", beaconListDefaultLimit, "How many beacons to fetch (1..100)")
	f.Bool("json", false, "Print the raw JSON response, pretty-printed")
	f.Bool("hash-only", false, "(invalid on 'list' — use --json + jq)")
	f.BoolP("silent", "s", false, "No output, exit code only")

	beaconCmd.AddCommand(beaconListCmd)
}
