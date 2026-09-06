// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import "github.com/spf13/cobra"

// itemsCmd is the parent for the `truestamp items ...` subtree. Like every
// group it is a namespace (asGroup): a bare `truestamp items` prints help. That matters
// more here than elsewhere — `items list` needs a credential and a network
// round trip, so a bare `truestamp items` that ran it would answer "what
// can I do here?" with an auth error on a fresh machine.
var itemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Create, list, and update timestamped items",
	Long: `An item is what Truestamp timestamps.

You submit claims — optionally including a hash of a file you keep
locally — Truestamp commits them to a Merkle tree, and the commitment is
written to Bitcoin and Stellar. Truestamp proves submission timing,
never creation timing.

The proof for an item is not a verb on this group: proof bundles are
derived artifacts and live under 'truestamp proofs'.

  truestamp proofs get <item-id> | truestamp verify --offline`,
}

func init() {
	itemsCmd.GroupID = groupResources
	rootCmd.AddCommand(asGroup(itemsCmd))
}
