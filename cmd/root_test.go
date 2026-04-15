// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import "testing"

func TestExecute_NoArgs(t *testing.T) {
	// Calling Execute with no args prints help and returns nil
	rootCmd.SetArgs([]string{})
	err := Execute()
	if err != nil {
		t.Errorf("Execute() with no args should succeed, got: %s", err)
	}
}

func TestExecute_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	err := Execute()
	if err != nil {
		t.Errorf("Execute() with --help should succeed, got: %s", err)
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"nonexistent"})
	err := Execute()
	if err == nil {
		t.Error("Execute() with unknown command should return error")
	}
}