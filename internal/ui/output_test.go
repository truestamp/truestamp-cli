// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package ui

import (
	"bytes"
	"strings"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// esc is the byte every one of these tests is really about: if it
// survives into a redirected stream, `truestamp ... > file` writes
// terminal escape sequences into that file.
const esc = "\x1b"

// styled returns a string that definitely carries ANSI when rendered at
// a colour-capable profile, so a test asserting "no escapes" is asserting
// that something was stripped rather than that nothing was ever there.
// Without this the tests would pass against a no-op implementation.
func styled() string {
	return lipgloss.NewStyle().Bold(true).Foreground(Red).Render("sentinel")
}

func TestStyledFixtureActuallyCarriesANSI(t *testing.T) {
	// Guard against the whole file becoming vacuous: if Style.Render ever
	// stops emitting escapes on its own, every "stripped" assertion below
	// would pass for the wrong reason.
	var buf bytes.Buffer
	w := &colorprofile.Writer{Forward: &buf, Profile: colorprofile.TrueColor}
	if _, err := w.Write([]byte(styled())); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), esc) {
		t.Fatal("fixture carries no ANSI at TrueColor, so the strip assertions below prove nothing")
	}
}

func TestProfileWriter_NoColorForced_StripsANSI(t *testing.T) {
	prev := noColorForced
	t.Cleanup(func() { noColorForced = prev })
	noColorForced = true

	var buf bytes.Buffer
	if _, err := Fprintln(&buf, styled()); err != nil {
		t.Fatalf("Fprintln: %v", err)
	}
	got := buf.String()
	if strings.Contains(got, esc) {
		t.Errorf("--no-color did not strip ANSI: %q", got)
	}
	if !strings.Contains(got, "sentinel") {
		t.Errorf("stripping removed the text as well as the escapes: %q", got)
	}
}

func TestFprintf_NoColorForced_StripsANSI(t *testing.T) {
	prev := noColorForced
	t.Cleanup(func() { noColorForced = prev })
	noColorForced = true

	var buf bytes.Buffer
	if _, err := Fprintf(&buf, "  %s\n", styled()); err != nil {
		t.Fatalf("Fprintf: %v", err)
	}
	if got := buf.String(); strings.Contains(got, esc) {
		t.Errorf("--no-color did not strip ANSI from Fprintf: %q", got)
	}
}

func TestFprint_NoColorForced_StripsANSI(t *testing.T) {
	prev := noColorForced
	t.Cleanup(func() { noColorForced = prev })
	noColorForced = true

	var buf bytes.Buffer
	if _, err := Fprint(&buf, styled()); err != nil {
		t.Fatalf("Fprint: %v", err)
	}
	if got := buf.String(); strings.Contains(got, esc) {
		t.Errorf("--no-color did not strip ANSI from Fprint: %q", got)
	}
}

// TestProfileWriter_NonTerminal_StripsANSI is the case that matters even
// without --no-color: a bytes.Buffer is not a terminal, so the detected
// profile must strip. This is what makes `truestamp ... > file` clean.
func TestProfileWriter_NonTerminal_StripsANSI(t *testing.T) {
	prev := noColorForced
	t.Cleanup(func() { noColorForced = prev })
	noColorForced = false

	var buf bytes.Buffer
	if _, err := Fprintln(&buf, styled()); err != nil {
		t.Fatalf("Fprintln: %v", err)
	}
	if got := buf.String(); strings.Contains(got, esc) {
		t.Errorf("redirected output kept ANSI: %q", got)
	}
}

// TestProfileWriter_IsNotFmtFprintln pins the actual regression. A plain
// fmt.Fprintln of the same styled string keeps its escapes; routing the
// same value through the ui helper must not. If someone reverts the
// helpers to thin fmt wrappers, this fails.
func TestProfileWriter_IsNotFmtFprintln(t *testing.T) {
	prev := noColorForced
	t.Cleanup(func() { noColorForced = prev })
	noColorForced = true

	s := styled()

	var raw bytes.Buffer
	// Deliberately the unsafe call this package exists to replace.
	raw.WriteString(s + "\n")

	var viaUI bytes.Buffer
	if _, err := Fprintln(&viaUI, s); err != nil {
		t.Fatalf("Fprintln: %v", err)
	}

	if !strings.Contains(raw.String(), esc) {
		t.Skip("Style.Render produced no ANSI in this environment; nothing to compare")
	}
	if strings.Contains(viaUI.String(), esc) {
		t.Error("ui.Fprintln behaved like fmt.Fprintln: escapes survived")
	}
}
