// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package ui

import (
	"fmt"
	"os"

	"charm.land/huh/v2"
	lipgloss "charm.land/lipgloss/v2"
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
