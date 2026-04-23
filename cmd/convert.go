// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/spf13/cobra"
)

// convertCmd is the parent of the domain-specific conversion sub-commands:
//
//	convert time    — ISO 8601 ↔ Unix ↔ timezone
//	convert proof   — JSON ↔ CBOR proof bundles
//	convert id      — ULID / UUIDv7 → embedded timestamp
//	convert keyid   — Ed25519 public key → Truestamp 4-byte kid fingerprint
//	convert merkle  — compact base64url Merkle proof → structured form
//
// Generic byte-encoding conversion (hex/base64/base64url/binary) lives
// at the top level as `truestamp encode` / `truestamp decode`, and JCS
// canonicalization as `truestamp jcs`, because those are pipeline
// primitives and benefit from shorter verbs.
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert Truestamp-domain values (time, proof format, IDs, keyid, Merkle proofs)",
	Long: `Convert between formats for Truestamp-domain values:

  convert time    — parse and re-format timestamps across zones and Unix formats
  convert proof   — switch a proof bundle between JSON and CBOR wire formats
  convert id      — extract the embedded timestamp from a ULID or UUIDv7
  convert keyid   — derive the 4-byte Truestamp kid fingerprint from an Ed25519 key
  convert merkle  — decode a compact base64url Merkle proof into its structure

For generic byte-encoding conversion use 'truestamp encode' / 'truestamp decode'.
For JSON canonicalization (RFC 8785) use 'truestamp jcs'.`,
}

func init() {
	rootCmd.AddCommand(convertCmd)
}

// addConvertCommonFlags registers the --json and --silent flags shared
// by every convert sub-command. Kept central so the flag name, default,
// help text, and -s short alias stay in sync across the family.
func addConvertCommonFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.Bool("json", false, "Output as JSON")
	f.BoolP("silent", "s", false, "No output, exit code only")
}
