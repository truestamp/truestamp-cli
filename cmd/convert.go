// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

// convertCmd is the parent of the domain-specific conversion sub-commands:
//
//	convert time   , ISO 8601 ↔ Unix ↔ timezone
//	convert id     , UUIDv7 → the time recorded inside it
//	convert keyid  , Ed25519 public key → Truestamp 4-byte kid fingerprint
//
// Generic byte-encoding conversion (hex/base64/base64url/binary) lives
// at the top level as `truestamp encode` / `truestamp decode`, and JCS
// canonicalization as `truestamp jcs`, because those are pipeline
// primitives and benefit from shorter verbs.
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert Truestamp-domain values (time, IDs, keyid)",
	Long: `Convert between formats for Truestamp-domain values:

  convert time   , parse and re-format timestamps across zones and Unix formats
  convert id     , read the time recorded inside a UUIDv7 id
  convert keyid  , derive the 4-byte Truestamp kid fingerprint from an Ed25519 key

For generic byte-encoding conversion use 'truestamp encode' / 'truestamp decode'.
For JSON canonicalization (RFC 8785) use 'truestamp jcs'.`,
}

func init() {
	convertCmd.GroupID = groupTools
	rootCmd.AddCommand(asGroup(convertCmd))
}
