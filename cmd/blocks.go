// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/blocks"
	"github.com/truestamp/truestamp-cli/internal/ids"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var blocksCmd = &cobra.Command{
	Use:   "blocks",
	Short: "Read-only: Truestamp's block chain",
	Long: `Read Truestamp's internal block chain.

A block is the full signed record: Merkle root, state, signature, key id,
and the links to its predecessor. The same block projected to four public
fields is a beacon; see 'truestamp beacons'.

The two are not interchangeable. Only finalized or committed blocks
project as beacons, and the chain advances about once a minute, so the
head block is routinely not yet finalized: 'blocks latest' and
'beacons latest' return different rows most of the time. They are two
questions, not two spellings of one.

Requires authentication, run 'truestamp auth login', or set TRUESTAMP_API_KEY / --api-key for headless/CI use.`,
}

var blocksListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the most recent blocks (newest first)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := blocksConfig(cmd)
		if err != nil {
			return err
		}
		limit, err := pageLimit(cmd)
		if err != nil {
			return err
		}
		list, err := blocks.List(cmd.Context(), cfg, limit)
		if err != nil {
			return renderAPIError(cmd, err, "block")
		}
		return renderBlockList(cmd, list)
	},
}

var blocksGetCmd = &cobra.Command{
	Use:   "get <id-or-hash>",
	Short: "Show one block by UUIDv7 id or by 64-hex-char block hash",
	Long: `Fetch one block, addressed either way.

The two shapes are disjoint — a UUIDv7 has hyphens, a block hash is
exactly 64 lowercase hex characters — so which you passed is unambiguous.

There is no by-hash route on the server, so a hash lookup is a filter.
Unlike the beacons by-hash action it carries no server-side shape guard,
so the hex is validated here before the request is sent. The server does
not assume block-hash uniqueness either: more than one match is reported
as an error, never resolved by picking one.

Examples:
  truestamp blocks get 019db702-b08c-73dc-a7cd-2c5e011f1dad
  truestamp blocks get ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := blocksConfig(cmd)
		if err != nil {
			return err
		}
		arg := strings.TrimSpace(args[0])

		var b *blocks.Block
		switch {
		case ids.LooksLikeHash64(arg):
			b, err = blocks.ByHash(cmd.Context(), cfg, arg)
		case strings.Contains(arg, "-"):
			b, err = blocks.Get(cmd.Context(), cfg, arg)
		default:
			return fmt.Errorf("%q is neither a UUIDv7 id nor a 64-hex-char block hash", arg)
		}
		if err != nil {
			return renderAPIError(cmd, err, "block")
		}
		return renderBlock(cmd, b)
	},
}

var blocksLatestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Show the head block",
	Long: `Show the head block: the newest block, whatever its state.

This is not the same as 'truestamp beacons latest', which shows the most
recent *finalized* block. The head is routinely not yet finalized.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := blocksConfig(cmd)
		if err != nil {
			return err
		}
		b, err := blocks.Latest(cmd.Context(), cfg)
		if err != nil {
			return renderAPIError(cmd, err, "block")
		}
		return renderBlock(cmd, b)
	},
}

var blocksGenesisCmd = &cobra.Command{
	Use:   "genesis",
	Short: "Show the first block in the chain",
	Long: `Show the genesis block: the first block, identifiable by its id being
its own previous_block_id.

It is a distinguished object rather than merely the oldest row: it is the
root every chain walk terminates at.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := blocksConfig(cmd)
		if err != nil {
			return err
		}
		b, err := blocks.Genesis(cmd.Context(), cfg)
		if err != nil {
			return renderAPIError(cmd, err, "block")
		}
		return renderBlock(cmd, b)
	},
}

// blocksConfig pulls the values the blocks client needs from the resolved
// application config, after the shared credential gate.
func blocksConfig(cmd *cobra.Command) (blocks.Config, error) {
	if err := requireAuth(cmd); err != nil {
		return blocks.Config{}, err
	}
	return blocks.Config{APIURL: appConfig.APIURL, Team: appConfig.Team}, nil
}

func renderBlock(cmd *cobra.Command, b *blocks.Block) error {
	jsonOut, silent := outputMode(cmd)
	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), b)
	}
	renderBlockCard(cmd.OutOrStdout(), b)
	return nil
}

func renderBlockCard(w io.Writer, b *blocks.Block) {
	header := ui.AccentBoldStyle().Render("  Block")
	tbl := ui.CompactTable().
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("ID", b.ID).
		Row("State", b.State).
		Row("Block Hash", b.BlockHash).
		Row("Merkle Root", b.MerkleRoot).
		Row("Previous", b.PreviousBlockHash).
		Row("Signing Key", b.SigningKeyID)
	if b.InsertedAt != "" {
		tbl = tbl.Row("Created", timestampWithRelative(b.InsertedAt))
	}
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

func renderBlockList(cmd *cobra.Command, list []blocks.Block) error {
	jsonOut, silent := outputMode(cmd)
	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), list)
	}
	w := cmd.OutOrStdout()
	if len(list) == 0 {
		ui.Fprintln(w, ui.FaintStyle().Render("  No blocks."))
		return nil
	}
	header := ui.AccentBoldStyle().Render(fmt.Sprintf("  Blocks (%d)", len(list)))
	tbl := ui.CompactTable().StyleFunc(ui.LabelValueStyleFunc())
	for _, b := range list {
		tbl = tbl.Row(b.ID, fmt.Sprintf("%-9s %s", b.State, truncateHash(b.BlockHash)))
	}
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
	return nil
}

// truncateHash shortens a 64-hex hash for a list column. The full value
// is always available from `blocks get` and from --json; this is display
// only, and never something another command consumes.
func truncateHash(h string) string {
	if len(h) <= 16 {
		return h
	}
	return h[:8] + "…" + h[len(h)-8:]
}

func init() {
	addLimitFlag(blocksListCmd, "blocks")
	for _, c := range []*cobra.Command{blocksListCmd, blocksGetCmd, blocksLatestCmd, blocksGenesisCmd} {
		addRecordOutputFlags(c)
		blocksCmd.AddCommand(c)
	}
	blocksCmd.GroupID = groupResources
	rootCmd.AddCommand(asGroup(blocksCmd))
}
