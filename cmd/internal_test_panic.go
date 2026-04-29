// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// internalTestPanicCmd is a hidden command whose only purpose is to let
// the panic-recovery integration test in root_panic_test.go trigger a
// real panic from the cobra Run path. Registration is gated on the
// TRUESTAMP_INTERNAL_TEST_PANIC env var so the command is invisible in
// production builds — never appearing in `--help`, never affecting
// column padding, never callable. The test sets the env var when
// invoking the subprocess; nothing else does.
var internalTestPanicCmd = &cobra.Command{
	Use:    "__test-panic",
	Hidden: true,
	Args:   cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		panic("intentional test panic")
	},
}

func init() {
	if os.Getenv("TRUESTAMP_INTERNAL_TEST_PANIC") == "1" {
		rootCmd.AddCommand(internalTestPanicCmd)
	}
}
