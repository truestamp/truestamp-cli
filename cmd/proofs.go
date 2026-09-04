// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import "github.com/spf13/cobra"

// proofsCmd is the parent for the `truestamp proofs ...` subtree.
//
// A proof bundle is a derived artifact, generated on demand from an item,
// block, beacon, or entropy observation and never stored server-side.
// That is why `get` takes the *subject's* id and why there is no `list`:
// there is no collection to enumerate yet. See kb/command-tree.md R7.
var proofsCmd = &cobra.Command{
	Use:   "proofs",
	Short: "Fetch and convert proof bundles",
	Long: `Proof bundles, generated on demand. 'get' emits bundle bytes.

A proof is derived from a subject — an item, block, beacon, or entropy
observation — and is generated when you ask for it rather than stored,
so 'proofs get' takes the subject's id. Pass --type when the id shape is
ambiguous, or let the CLI resolve it in one extra round trip.

To check a bundle, use 'truestamp verify'; to see what one carries
without checking it, 'truestamp inspect'. Neither needs a credential or
this group.

  truestamp proofs get <id> | truestamp verify --offline`,
	Args: cobra.NoArgs,
}

func init() {
	proofsCmd.GroupID = groupResources
	rootCmd.AddCommand(asGroup(proofsCmd))
}
