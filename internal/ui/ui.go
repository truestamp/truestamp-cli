// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package ui provides shared styling for the Truestamp CLI using lipgloss v2.
// Colors are sourced from the Catppuccin palette (Latte for light terminals,
// Mocha for dark). The same palette is used for both static lipgloss output
// and interactive huh components via HuhTheme().
//
// IMPORTANT: Color initialization is deferred to Init() to avoid terminal
// queries at package init time. Querying the terminal (for dark/light
// background detection) in a background process group causes SIGTTIN,
// which hangs shell completion via source <(truestamp completion zsh).
package ui

import (
	"image/color"
	"os"
	"strings"
	"sync"

	"charm.land/huh/v2"
	lipgloss "charm.land/lipgloss/v2"
	catppuccin "github.com/catppuccin/go"
	"github.com/charmbracelet/colorprofile"
)

// Color variables. Defaults to dark (Mocha) palette. Updated by Init()
// if the terminal has a light background.
var (
	Green  color.Color = catppuccin.Mocha.Green()
	Red    color.Color = catppuccin.Mocha.Red()
	Yellow color.Color = catppuccin.Mocha.Yellow()
	Blue   color.Color = catppuccin.Mocha.Blue()
	Accent color.Color = catppuccin.Mocha.Mauve()
	Dim    color.Color = catppuccin.Mocha.Overlay0()
	Label  color.Color = catppuccin.Mocha.Subtext0()
	Value  color.Color = catppuccin.Mocha.Text()
	Banner color.Color = catppuccin.Mocha.Text()
)

var initOnce sync.Once

// Init configures the global color profile and detects the terminal's
// background color. Call once from root command before any output.
// If noColor is true, all ANSI sequences are stripped.
// The NO_COLOR env var is handled automatically by lipgloss.
func Init(noColor bool) {
	initOnce.Do(func() {
		if noColor {
			lipgloss.Writer.Profile = colorprofile.NoTTY
			return
		}

		// Detect light/dark background now that we know we're in
		// a real command (not completion) with terminal access.
		if !hasDarkBackground() {
			Green = catppuccin.Latte.Green()
			Red = catppuccin.Latte.Red()
			Yellow = catppuccin.Latte.Yellow()
			Blue = catppuccin.Latte.Blue()
			Accent = catppuccin.Latte.Mauve()
			Dim = catppuccin.Latte.Overlay0()
			Label = catppuccin.Latte.Subtext0()
			Value = catppuccin.Latte.Text()
			Banner = catppuccin.Latte.Text()
		}
	})
}

// hasDarkBackground safely checks the terminal background, returning
// true (dark) as the default if detection fails.
func hasDarkBackground() bool {
	// Only query if stdout looks like a terminal
	stat, err := os.Stdout.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) != 0 {
		return lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	}
	return true // default to dark
}

// --- Shared Components ---

// sectionWidth is the shared width for section separator lines and the header box.
const sectionWidth = 44

// HeaderBox renders a bordered title box with an optional subtitle.
func HeaderBox(title, subtitle string) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(Banner)
	subStyle := lipgloss.NewStyle().Faint(true).Foreground(Dim)

	content := titleStyle.Render(title)
	if subtitle != "" {
		content += "\n" + subStyle.Render(subtitle)
	}

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Accent).
		Padding(0, 2).
		Width(sectionWidth).
		Render(content)
}

// SectionHeader renders a styled group header with a separator line.
func SectionHeader(name string) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(Accent).Render(name)
	line := lipgloss.NewStyle().Foreground(Dim).Render(strings.Repeat("─", sectionWidth-2))
	return "  " + title + "\n  " + line
}

// SuccessBanner renders a bold green success message.
func SuccessBanner(text string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(Green).Render("  " + text)
}

// FailureBanner renders a bold red failure message.
func FailureBanner(text string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(Red).Render("  " + text)
}

// LabelStyle returns the style for key-value labels.
func LabelStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Label)
}

// ValueStyle returns the style for key-value values.
func ValueStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Value)
}

// FaintStyle returns a faint/dim style.
func FaintStyle() lipgloss.Style {
	return lipgloss.NewStyle().Faint(true).Foreground(Dim)
}

// AccentBoldStyle returns a bold accent-colored style.
func AccentBoldStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Accent)
}

// HuhTheme returns the Catppuccin theme for huh interactive components.
// Uses the same Latte/Mocha palette as the static color definitions above.
func HuhTheme() huh.Theme {
	return huh.ThemeFunc(huh.ThemeCatppuccin)
}
