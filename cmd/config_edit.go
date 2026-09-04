// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
)

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the config file in your editor",
	Long: `Open the config file in effect in your editor.

The editor is $VISUAL, else $EDITOR, else a platform default (vi on Unix,
notepad on Windows). The value may carry arguments, so EDITOR="code -w"
works; it is split on whitespace and executed directly, not through a
shell, so quoting and redirection in it are not interpreted.

The file must already exist: run 'truestamp config init' first. This
command deliberately does not create it, so there is exactly one command
that writes the documented defaults and comments.

The CLI does not re-read the file after the editor exits; the next
invocation picks up whatever was saved. A file that no longer parses is
reported by the next command that loads it, so 'truestamp config show' is
a good thing to run afterwards.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.ActivePath()
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("no config file at %s: run 'truestamp config init' to create it", path)
		}

		// An editor needs a terminal to draw on. Without this a
		// `truestamp config edit` in CI or inside a captured subprocess
		// launches vi against a pipe and blocks forever -- the same trap
		// that made `verify --file </dev/null` hang before it moved to
		// term.IsTerminal.
		if !inputsrc.IsStdinTerminal() {
			return fmt.Errorf("config edit needs a terminal; edit %s directly instead", path)
		}

		editor, source := resolveEditor()
		if editor == "" {
			return fmt.Errorf("no editor found: set $VISUAL or $EDITOR, or edit %s directly", path)
		}
		parts := strings.Fields(editor)
		bin, err := exec.LookPath(parts[0])
		if err != nil {
			return fmt.Errorf("editor %q (from %s) not found on $PATH: %w", parts[0], source, err)
		}

		ed := exec.Command(bin, append(parts[1:], path)...)
		ed.Stdin, ed.Stdout, ed.Stderr = os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr()
		if err := ed.Run(); err != nil {
			return fmt.Errorf("editor %q exited with an error: %w", editor, err)
		}
		appLogger.Info("config_edited", "path", path, "editor", editor, "editor_source", source)
		return nil
	},
}

// resolveEditor returns the editor command and where it came from, so an
// error can name the variable the user actually set.
func resolveEditor() (editor, source string) {
	if v := strings.TrimSpace(os.Getenv("VISUAL")); v != "" {
		return v, "$VISUAL"
	}
	if v := strings.TrimSpace(os.Getenv("EDITOR")); v != "" {
		return v, "$EDITOR"
	}
	if runtime.GOOS == "windows" {
		return "notepad", "the platform default"
	}
	return "vi", "the platform default"
}

func init() {
	configCmd.AddCommand(configEditCmd)
}
