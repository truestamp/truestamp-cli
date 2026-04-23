// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/beacons"
)

var beaconGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a beacon by UUIDv7 id",
	Long: `Fetch a single beacon by its UUIDv7 id.

Examples:
  truestamp beacon get 019db702-b08c-73dc-a7cd-2c5e011f1dad
  truestamp beacon get 019db702-b08c-73dc-a7cd-2c5e011f1dad --hash-only
  truestamp beacon get 019db702-b08c-73dc-a7cd-2c5e011f1dad --json`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runBeaconGet,
}

func runBeaconGet(cmd *cobra.Command, args []string) error {
	jsonOut, hashOnly, silent, err := beaconSharedFlags(cmd)
	if err != nil {
		return err
	}
	// Client-side validation saves a round trip on obvious typos.
	if err := beacons.ValidateUUIDv7(args[0]); err != nil {
		return err
	}
	cfg, err := beaconConfig(cmd)
	if err != nil {
		return err
	}
	b, err := beacons.Get(cmd.Context(), cfg, args[0])
	if err != nil {
		return beaconRenderError(cmd, err, silent)
	}
	return renderBeacon(cmd, b, jsonOut, hashOnly, silent)
}

func init() {
	f := beaconGetCmd.Flags()
	f.Bool("json", false, "Print the raw JSON response, pretty-printed")
	f.Bool("hash-only", false, "Print only the beacon hash + newline")
	f.BoolP("silent", "s", false, "No output, exit code only")

	beaconCmd.AddCommand(beaconGetCmd)
}
