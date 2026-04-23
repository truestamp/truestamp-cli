// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package ui

import (
	"fmt"
	"os"
	"time"

	"charm.land/huh/v2"
	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// PickFileOptions configures an interactive file-picker form. Both fields
// are optional; Title defaults to "Select file" and AllowedTypes to "any
// file type".
type PickFileOptions struct {
	Title        string
	AllowedTypes []string
}

// PickFile launches a single-page interactive file picker themed with the
// CLI's palette and returns the selected path. Returns an error if the
// user aborts, the form fails, or no file is selected.
func PickFile(opts PickFileOptions) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	title := opts.Title
	if title == "" {
		title = "Select file"
	}

	picker := huh.NewFilePicker().
		Title(title).
		Description("up/down move  right/enter open  left/backspace back  enter select").
		CurrentDirectory(cwd).
		FileAllowed(true).
		DirAllowed(false).
		ShowSize(true).
		ShowPermissions(false).
		Picking(true).
		Height(20)
	if len(opts.AllowedTypes) > 0 {
		picker = picker.AllowedTypes(opts.AllowedTypes)
	}

	var path string
	err = huh.NewForm(
		huh.NewGroup(picker.Value(&path)),
	).WithTheme(HuhTheme()).Run()
	if err != nil {
		return "", fmt.Errorf("file selection: %w", err)
	}
	if path == "" {
		return "", fmt.Errorf("no file selected")
	}
	return path, nil
}

// CompactTable returns a lipgloss table with a hidden border AND no
// top/bottom/left/right border rows. `.Border(HiddenBorder())` on its
// own still emits invisible top and bottom rows of whitespace, which
// stacks with section separators in the card / verify-report output
// and wastes a blank line above and below every table. Using this
// helper ensures the table content is flush to whatever comes before
// and after it, letting callers use explicit `""` elements in a
// `strings.Join` (or a bare newline) for inter-section spacing when
// they want it. Every post-action card / verify-report table in the
// CLI routes through this.
func CompactTable() *table.Table {
	return table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false)
}

// LabelValueStyleFunc returns a lipgloss table StyleFunc that renders the
// first column as a right-aligned label and subsequent columns as the
// value. It matches the two-column "key: value" layout used by the create
// and download summary tables.
func LabelValueStyleFunc() func(row, col int) lipgloss.Style {
	return func(row, col int) lipgloss.Style {
		if col == 0 {
			return lipgloss.NewStyle().
				Foreground(Label).
				PaddingLeft(2).
				Align(lipgloss.Right).
				PaddingRight(1)
		}
		return lipgloss.NewStyle().Foreground(Value)
	}
}

// TruncateToSecond parses an RFC 3339 / ISO 8601 timestamp and re-emits
// it at second precision (drops fractional seconds). Returns the input
// string unchanged if it cannot be parsed, so it is safe to chain with
// already-truncated values. Used by every display site that shows a
// timestamp to a human — beacon list rows, beacon cards, the verify
// report's Timeline / Subject / Commitments sections. The `convert`
// subcommands deliberately bypass this helper because they exist
// precisely to extract high-precision timestamps from IDs.
func TruncateToSecond(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return ts
	}
	return t.Truncate(time.Second).Format(time.RFC3339)
}
