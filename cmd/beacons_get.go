// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/beacons"
)

var beaconsGetCmd = &cobra.Command{
	Use:   "get <id-or-hash>",
	Short: "Show a beacon by UUIDv7 id or by 64-hex-char hash",
	Long: `Fetch a single beacon, addressed either way.

The two id shapes are disjoint — a UUIDv7 has hyphens and is 36
characters, a beacon hash is exactly 64 lowercase hex characters — so
which one you passed is unambiguous and there is no separate command for
each. Users address records, not routes.

Examples:
  truestamp beacons get 019db702-b08c-73dc-a7cd-2c5e011f1dad
  truestamp beacons get ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b
  truestamp beacons get 019db702-b08c-73dc-a7cd-2c5e011f1dad --hash-only
  truestamp beacons get 019db702-b08c-73dc-a7cd-2c5e011f1dad --json`,
	Args: cobra.ExactArgs(1),
	RunE: runBeaconsGet,
}

// looksLikeHash reports whether arg has the shape of a 64-hex block or
// beacon hash rather than a UUIDv7. This is a shape test on a value the
// user typed, not an inference about a proof bundle's subject type: the
// filename-independence rule in CLAUDE.md is about the latter and does
// not apply. Shared by `beacons get` and `blocks get`.
func looksLikeHash(arg string) bool {
	return !strings.Contains(arg, "-") && len(arg) == 64
}

func runBeaconsGet(cmd *cobra.Command, args []string) error {
	jsonOut, hashOnly, silent, err := beaconSharedFlags(cmd)
	if err != nil {
		return err
	}
	arg := strings.TrimSpace(args[0])

	cfg, err := beaconConfig(cmd)
	if err != nil {
		return err
	}

	// The client validates the id or hash before any request is sent, so
	// an obvious typo costs no round trip and is reported the same way
	// whichever shape it has.
	var b *beacons.Beacon
	switch {
	case looksLikeHash(arg):
		b, err = beacons.ByHash(cmd.Context(), cfg, arg)
	case strings.Contains(arg, "-"):
		b, err = beacons.Get(cmd.Context(), cfg, arg)
	default:
		return fmt.Errorf(
			"%q is neither a UUIDv7 id nor a 64-hex-char beacon hash", arg)
	}
	if err != nil {
		return beaconRenderError(cmd, err, silent)
	}
	return renderBeacon(cmd, b, jsonOut, hashOnly, silent)
}

func init() {
	beaconsGetCmd.Flags().Bool("hash-only", false, "Print only the beacon hash + newline")
	addRecordOutputFlags(beaconsGetCmd)
	beaconsCmd.AddCommand(beaconsGetCmd)
}
