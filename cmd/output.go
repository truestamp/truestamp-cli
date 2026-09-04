// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// The R10 output contract, in one place so a new command cannot invent a
// variation of it. See kb/command-tree.md R10 for the rules and for the
// list of deliberate exemptions (the pipeline primitives that print one
// bare value or raw bytes, and `hash --style`, which is a byte-identical
// sha256sum drop-in).
//
// --json and --silent are NOT root persistent flags. A persistent flag
// appears on every command, including `auth login`, `console` and
// `completion`, which render no record and would then advertise an output
// mode they do not have. They are registered per command instead, from
// here, so the set of commands that honor the contract is exactly the set
// that says it does.

// addRecordOutputFlags registers --json and --silent on a command that
// renders a record. config.Load enforces their mutual exclusion once, for
// every command, so callers do not repeat that check.
func addRecordOutputFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.Bool("json", false, "Output as JSON")
	f.BoolP("silent", "s", false, "No output, exit code only")
}

// emitRecord writes v as indented JSON. It is the single JSON writer for
// record-rendering commands, so the shape of `--json` output is decided in
// one place rather than by whichever encoder each command reached for.
func emitRecord(w io.Writer, v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(out))
	return err
}

// outputMode reads the resolved output settings for the current
// invocation.
//
// It prefers appConfig, which config.Load has already merged from
// defaults, config file, environment and flags in precedence order — that
// is what makes `silent = true` in config.toml work at all. An explicitly
// changed flag is consulted directly as well, because it wins under that
// same precedence, and because unit tests that call a RunE without going
// through PersistentPreRunE have no resolved config to read.
func outputMode(cmd *cobra.Command) (jsonOut, silent bool) {
	if appConfig != nil {
		jsonOut, silent = appConfig.JSON, appConfig.Silent
	}
	if cmd == nil {
		return jsonOut, silent
	}
	if f := cmd.Flags().Lookup("json"); f != nil && f.Changed {
		jsonOut, _ = cmd.Flags().GetBool("json")
	}
	if f := cmd.Flags().Lookup("silent"); f != nil && f.Changed {
		silent, _ = cmd.Flags().GetBool("silent")
	}
	return jsonOut, silent
}
