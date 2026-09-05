// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/beacons"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var beaconsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the most recent beacons (newest first)",
	Long: `List beacons, newest first.

Paging is by keyset cursor, like every other list: --limit sets the page
size (the server clamps a page above 100 to 100), --after and --before
continue from a cursor a previous page printed, --oldest-first starts at
the genesis beacon, --max follows cursors until that many beacons have
been fetched, and --count adds the total.

Examples:
  truestamp beacons list
  truestamp beacons list --limit 3
  truestamp beacons list --oldest-first --limit 3
  truestamp beacons list --limit 10 --json | jq -r '.beacons[].hash'`,
	Args: cobra.NoArgs,
	RunE: runBeaconsList,
}

func runBeaconsList(cmd *cobra.Command, _ []string) error {
	jsonOut, _, silent, err := beaconSharedFlags(cmd)
	if err != nil {
		return err
	}
	paging, err := readPagingOptions(cmd)
	if err != nil {
		return err
	}
	cfg, err := beaconConfig(cmd)
	if err != nil {
		return err
	}
	rows, pg, err := walkPages(paging, func(after, before string, limit int) (*pageOf[beacons.Beacon], error) {
		p, err := beacons.List(cmd.Context(), cfg, beacons.ListOptions{
			Limit: limit, After: after, Before: before, OldestFirst: paging.OldestFirst, Count: paging.Count,
		})
		if err != nil {
			return nil, err
		}
		return &pageOf[beacons.Beacon]{Rows: p.Beacons, NextCursor: p.NextCursor, PrevCursor: p.PrevCursor, Total: p.Total}, nil
	})
	if err != nil {
		return renderAPIError(cmd, err, "beacon")
	}

	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), listEnvelope("beacons", rows, pg))
	}
	renderBeaconList(cmd.OutOrStdout(), rows, pg)
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
func renderBeaconList(w io.Writer, items []beacons.Beacon, pg listPage) {
	header := ui.AccentBoldStyle().Render(listHeading("Beacons", len(items), pg))

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
	renderMoreHint(w, pg)
}

func init() {
	addPagingFlags(beaconsListCmd, "beacons")
	addRecordOutputFlags(beaconsListCmd)
	beaconsCmd.AddCommand(beaconsListCmd)
}
