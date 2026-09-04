// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/blocks"
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

Sub-commands:
  list      Show the most recent blocks (newest first)
  get       Show one block by UUIDv7 id or by 64-hex block hash
  latest    Show the head block
  genesis   Show the first block, the root every chain walk terminates at

Requires authentication, run 'truestamp auth login', or set TRUESTAMP_API_KEY / --api-key for headless/CI use.`,
	Args: cobra.NoArgs,
}

var blocksListCmd = &cobra.Command{
	Use:           "list",
	Short:         "Show the most recent blocks (newest first)",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := blocksConfig(cmd)
		if err != nil {
			return err
		}
		limit, err := pageLimit(cmd, 25)
		if err != nil {
			return err
		}
		list, err := blocks.List(cmd.Context(), cfg, limit)
		if err != nil {
			return blocksRenderError(cmd, err)
		}
		return renderBlockList(cmd, list)
	},
}

var blocksGetCmd = &cobra.Command{
	Use:   "get <id-or-hash>",
	Short: "Show one block by UUIDv7 id or by 64-hex block hash",
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
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := blocksConfig(cmd)
		if err != nil {
			return err
		}
		arg := strings.TrimSpace(args[0])

		var b *blocks.Block
		switch {
		case !strings.Contains(arg, "-") && len(arg) == 64:
			b, err = blocks.ByHash(cmd.Context(), cfg, arg)
		case strings.Contains(arg, "-"):
			b, err = blocks.Get(cmd.Context(), cfg, arg)
		default:
			return fmt.Errorf("%q is neither a UUIDv7 id nor a 64-hex-char block hash", arg)
		}
		if err != nil {
			return blocksRenderError(cmd, err)
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
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := blocksConfig(cmd)
		if err != nil {
			return err
		}
		b, err := blocks.Latest(cmd.Context(), cfg)
		if err != nil {
			return blocksRenderError(cmd, err)
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
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := blocksConfig(cmd)
		if err != nil {
			return err
		}
		b, err := blocks.Genesis(cmd.Context(), cfg)
		if err != nil {
			return blocksRenderError(cmd, err)
		}
		return renderBlock(cmd, b)
	},
}

// blocksConfig mirrors beaconConfig: it refuses early, with a banner,
// when no credential is configured.
func blocksConfig(cmd *cobra.Command) (blocks.Config, error) {
	if !authConfigured() {
		_, silent := outputMode(cmd)
		if !silent {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp auth login' to sign in (or set TRUESTAMP_API_KEY)."))
		}
		return blocks.Config{}, errSilentFail
	}
	return blocks.Config{APIURL: appConfig.APIURL, Team: appConfig.Team}, nil
}

func blocksRenderError(cmd *cobra.Command, err error) error {
	_, silent := outputMode(cmd)
	if errors.Is(err, blocks.ErrUnauthorized) {
		if !silent {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp auth login' to sign in (or set TRUESTAMP_API_KEY)."))
		}
		return errSilentFail
	}
	if silent {
		return errSilentFail
	}
	var apiErr *blocks.APIError
	if errors.As(err, &apiErr) {
		if errors.Is(err, blocks.ErrNotFound) {
			return fmt.Errorf("block not found")
		}
		if errors.Is(err, blocks.ErrRateLimited) && apiErr.RetryAfter != "" {
			return fmt.Errorf("rate limited (Retry-After: %s): %s", apiErr.RetryAfter, apiErr.Detail)
		}
		return fmt.Errorf("%s", apiErr.Error())
	}
	return err
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
	blocksListCmd.Flags().Int("limit", 25, "How many blocks to show; the server caps it and says so if you ask for more")
	for _, c := range []*cobra.Command{blocksListCmd, blocksGetCmd, blocksLatestCmd, blocksGenesisCmd} {
		addRecordOutputFlags(c)
		blocksCmd.AddCommand(c)
	}
	blocksCmd.GroupID = groupResources
	rootCmd.AddCommand(asGroup(blocksCmd))
}
