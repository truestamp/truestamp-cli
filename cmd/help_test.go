// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestCLI_Help_UnknownTopicExitsNonZero.
//
// Cobra's built-in help command prints "Unknown help topic", dumps the root
// help and exits 0, which tells a caller the topic was found. Everywhere
// else in this tree a name that does not resolve exits 1 -- `truestamp
// items bogus` does -- and `help` is exactly where a reader following a
// stale document lands, so it is the last place that should answer 0.
func TestCLI_Help_UnknownTopicExitsNonZero(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{"retired help topic", []string{"help", "formatting"}, 1},
		{"nonsense topic", []string{"help", "not-a-command"}, 1},
		{"unknown subcommand of a real group", []string{"help", "items", "bogus"}, 1},
		{"bare help", []string{"help"}, 0},
		{"real leaf", []string{"help", "verify"}, 0},
		{"real nested leaf", []string{"help", "items", "list"}, 0},
		{"real group", []string{"help", "proofs"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, got := runCLIText(t, tc.args...)
			if got != tc.want {
				t.Errorf("%v exited %d, want %d\n%s", tc.args, got, tc.want, firstLine(out))
			}
			if tc.want == 1 && !strings.Contains(string(out), "unknown help topic") {
				t.Errorf("%v should say what was not found, got %q", tc.args, firstLine(string(out)))
			}
		})
	}
}

// No help text joins its two clauses with a comma ("Requires
// authentication, run ..."); the colon form is the one EXAMPLES.md uses.
func TestHelp_NoRequiresAuthenticationCommaSplice(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if strings.Contains(c.Long, "Requires authentication, run") {
			t.Errorf("%s: comma splice in its help", c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
}

// keys is not the only group that needs no credential (hash, encode, jcs,
// convert, verify and inspect need none either), so its help must not say
// it is. Mutant killed: the claim restored.
func TestHelp_KeysDoesNotClaimToBeTheOnlyCredentialFreeGroup(t *testing.T) {
	if strings.Contains(keysCmd.Long, "only command group") {
		t.Error("keys --help claims to be the only command group that needs no credential")
	}
}
