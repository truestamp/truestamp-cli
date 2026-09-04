// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/beacons"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var beaconsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the most recent beacons (newest first)",
	Long: `List recent beacons, newest first. The server caps --limit and says so if you ask for more.

Examples:
  truestamp beacons list
  truestamp beacons list --limit 3
  truestamp beacons list --limit 10 --json | jq '.[].hash'`,
	Args: cobra.NoArgs,
	RunE: runBeaconsList,
}

func runBeaconsList(cmd *cobra.Command, _ []string) error {
	jsonOut, _, silent, err := beaconSharedFlags(cmd)
	if err != nil {
		return err
	}

	limit, err := pageLimit(cmd)
	if err != nil {
		return err
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
	// One-line hint on interactive runs pointing at `proofs get --type beacon`.
	// Suppressed when stdout is piped so shell pipelines stay clean.
	if inputsrc.IsStdoutTerminal() {
		ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
			"  Hint: 'truestamp proofs get --type beacon <id>' fetches a verifiable proof bundle."))
	}
	return nil
}

// renderBeaconList prints a compact three-column table. Hashes are
// always shown full-width, truncation would silently drop the bytes a
// user came here to capture (the whole point of `beacons list` is to
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
		StyleFunc(ui.HeaderRowStyleFunc()).
		Rows(rows...)

	// Plain newline-join, see note in internal/verify/presenter.go
	// Present(). Avoids lipgloss.JoinVertical's pad-to-widest behaviour,
	// which would make long hash rows blow up vertical spacing on
	// narrow terminals.
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

func init() {
	addLimitFlag(beaconsListCmd, "beacons")
	addRecordOutputFlags(beaconsListCmd)

	beaconsCmd.AddCommand(beaconsListCmd)
}
