// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

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
	"fmt"
	"image/color"
	"io"
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
	Cyan   color.Color = catppuccin.Mocha.Sky()
	Accent color.Color = catppuccin.Mocha.Mauve()
	Dim    color.Color = catppuccin.Mocha.Overlay0()
	Hint   color.Color = catppuccin.Mocha.Subtext1()
	Label  color.Color = catppuccin.Mocha.Subtext0()
	Value  color.Color = catppuccin.Mocha.Text()
	Banner color.Color = catppuccin.Mocha.Text()
)

var initOnce sync.Once

// noColorForced records the --no-color flag so [Fprintln] and [Fprintf]
// can honor it. lipgloss offers no global "never emit ANSI" switch:
// Style.Render always embeds escape sequences and they are stripped at
// write time by a colorprofile.Writer. lipgloss.Writer is one such
// writer, which is why lipgloss.Println respects the flag, but a
// fmt.Fprintln(w, style.Render(...)) writes straight to w and keeps the
// escapes. lipgloss.Fprintln builds its writer from os.Environ(), so it
// honors NO_COLOR but cannot see our flag either. Hence this.
var noColorForced bool

// Fprintln writes to w through a colour profile writer, so styled text
// is downsampled or stripped to match what the destination can actually
// render, and --no-color / NO_COLOR are honored. Use it instead of
// fmt.Fprintln anywhere the arguments may contain Style.Render output.
func Fprintln(w io.Writer, a ...any) (int, error) {
	return fmt.Fprintln(ProfileWriter(w), a...)
}

// Fprint is Fprintln without the trailing newline. Same contract.
func Fprint(w io.Writer, a ...any) (int, error) {
	return fmt.Fprint(ProfileWriter(w), a...)
}

// Fprintf is Fprintln's formatting counterpart. Same contract.
func Fprintf(w io.Writer, format string, a ...any) (int, error) {
	return fmt.Fprintf(ProfileWriter(w), format, a...)
}

// ProfileWriter wraps w so ANSI is stripped or downsampled to suit the
// destination. Detection runs on every call, so a caller writing many
// lines to one destination (the hash listing) takes the writer once
// rather than paying for it per line.
func ProfileWriter(w io.Writer) io.Writer {
	if noColorForced {
		return &colorprofile.Writer{Forward: w, Profile: colorprofile.NoTTY}
	}
	return colorprofile.NewWriter(w, os.Environ())
}

// Init configures the global color profile and detects the terminal's
// background color. Call once from root command before any output.
// If noColor is true, all ANSI sequences are stripped.
// The NO_COLOR env var is handled automatically by lipgloss.
func Init(noColor bool) {
	initOnce.Do(func() {
		if noColor {
			noColorForced = true
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
			Cyan = catppuccin.Latte.Sky()
			Accent = catppuccin.Latte.Mauve()
			Dim = catppuccin.Latte.Overlay0()
			Hint = catppuccin.Latte.Subtext1()
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

// FaintStyle returns a faint/dim style. It is for decoration that the
// reader is not expected to act on. Anything meant to be READ — a hint, a
// cursor to paste, an empty-state line — uses [HintStyle] instead.
func FaintStyle() lipgloss.Style {
	return lipgloss.NewStyle().Faint(true).Foreground(Dim)
}

// HintStyle returns the style for trailing guidance: the "More:" / "Back:"
// cursor lines, the "Hint:" tips, and empty-state text.
//
// It is deliberately NOT [FaintStyle]. That style compounds two dimming
// effects — an already-low-contrast foreground (Overlay0) plus the ANSI
// faint attribute (SGR 2), which terminals render by reducing brightness
// again. Measured against each flavour's base, Overlay0 is 3.36:1 on
// Mocha and 2.30:1 on Latte: below the WCAG AA 4.5:1 floor on dark and
// failing outright on light, before the faint attribute is applied at
// all. Subtext1 is 9.26:1 and 5.53:1, so it clears AA in both themes
// while still reading as subordinate to Value.
//
// A hint the reader cannot see is not a subtle hint, it is a missing one,
// and the "More: --after <cursor>" line exists to be copied.
func HintStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Hint)
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
