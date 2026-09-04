// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// registerHelpCommand replaces cobra's built-in `help` so that asking for
// help on something that does not exist is an error.
//
// Cobra's version prints "Unknown help topic", dumps the root help, and
// exits 0. That is the same failure mode the tree reorganization removed
// everywhere else: `truestamp items bogus` exits 1, and a reader following
// a stale document deserves the same signal from `truestamp help items
// bogus`. Exit 0 tells a script the topic was found.
//
// Wired from Execute rather than an init(), for the same reason
// registerEnumCompletions is: the whole tree has to exist before the
// replacement is installed, and cobra only creates its default help
// command lazily during Execute.
func registerHelpCommand(root *cobra.Command) {
	root.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Print the help for any command in the tree.

'truestamp help' with no arguments prints the root help. A topic that
names no command is an error and exits 1: there are no help topics
beyond the commands themselves.`,
		// Completion should offer real commands, which is what cobra's
		// own default does for this command.
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var out []string
			target, _, err := c.Root().Find(args)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if target == nil {
				target = c.Root()
			}
			for _, sub := range target.Commands() {
				if sub.IsAvailableCommand() && strings.HasPrefix(sub.Name(), toComplete) {
					out = append(out, fmt.Sprintf("%s\t%s", sub.Name(), sub.Short))
				}
			}
			return out, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(c *cobra.Command, args []string) error {
			target, rest, err := c.Root().Find(args)
			if target == nil || err != nil {
				return fmt.Errorf(
					"unknown help topic %q, run 'truestamp --help' for the command tree",
					strings.Join(args, " "))
			}
			// Find resolves the DEEPEST match and hands back what it could
			// not consume, so `help items bogus` returns the `items`
			// command with "bogus" left over rather than an error. After a
			// group, a leftover word names a sub-command that does not
			// exist; after a leaf it is an operand
			// (`help verify proof.json`), which is harmless.
			if target.HasSubCommands() && len(rest) > 0 {
				return fmt.Errorf(
					"unknown help topic %q: %q has no %q sub-command, run '%s --help'",
					strings.Join(args, " "), target.Name(), rest[0], target.CommandPath())
			}
			target.InitDefaultHelpFlag()
			target.InitDefaultVersionFlag()
			return target.Help()
		},
	})
}
