// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// A group is a namespace, never a command (kb/command-tree.md R0): a bare
// `truestamp items` prints help and does nothing else.
//
// Cobra's default for a command with subcommands and no Run is to print
// help and exit 0 — including when it was handed an argument it does not
// recognize. That makes a retired path fail silently: after `proof` moved
// to `proofs convert`, `truestamp convert proof` printed convert's help
// and exited 0, so a reader following an old document got no signal at
// all. Setting Args to NoArgs does not fix it, because cobra never reaches
// the argument validator on that path.
//
// So every group gets this RunE. It is not a default action — the thing R0
// exists to forbid — it is the help behavior, plus an error for an
// argument that names nothing.
func groupRunE(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q, run '%s --help' for the available sub-commands",
			args[0], cmd.CommandPath(), cmd.CommandPath())
	}
	return cmd.Help()
}

// asGroup configures c as a namespace. Call it on every group so the
// behavior is identical across the tree rather than re-derived per file.
func asGroup(c *cobra.Command) *cobra.Command {
	c.Args = cobra.ArbitraryArgs
	c.RunE = groupRunE
	return c
}
