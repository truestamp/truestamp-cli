// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/beacons"
)

var beaconByHashCmd = &cobra.Command{
	Use:   "by-hash <hash>",
	Short: "Show a beacon by its 64-hex-char hash",
	Long: `Fetch a single beacon by its hash — useful when all you have is the
hash (e.g. printed on a receipt or read out of photo metadata).

The hash must be exactly 64 lowercase hexadecimal characters.

Examples:
  truestamp beacon by-hash ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b
  truestamp beacon by-hash <hash> --json`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runBeaconByHash,
}

func runBeaconByHash(cmd *cobra.Command, args []string) error {
	jsonOut, hashOnly, silent, err := beaconSharedFlags(cmd)
	if err != nil {
		return err
	}
	// Client-side validation saves a round trip on obvious typos.
	if err := beacons.ValidateHash(args[0]); err != nil {
		return err
	}
	cfg, err := beaconConfig(cmd)
	if err != nil {
		return err
	}
	b, err := beacons.ByHash(cmd.Context(), cfg, args[0])
	if err != nil {
		return beaconRenderError(cmd, err, silent)
	}
	return renderBeacon(cmd, b, jsonOut, hashOnly, silent)
}

func init() {
	f := beaconByHashCmd.Flags()
	f.Bool("json", false, "Print the raw JSON response, pretty-printed")
	f.Bool("hash-only", false, "Print only the beacon hash + newline")
	f.BoolP("silent", "s", false, "No output, exit code only")

	beaconCmd.AddCommand(beaconByHashCmd)
}
