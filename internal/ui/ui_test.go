// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package ui

import (
	"strings"
	"testing"
)

// These tests exercise the styling functions' call paths. Actual visual
// correctness is out of scope — we just verify the functions produce
// non-empty, non-panicking output for a known input.

func TestInit_NoColor(t *testing.T) {
	// Calling Init(true) sets the no-color profile. The init is
	// protected by sync.Once so a second call is a no-op; either
	// outcome is acceptable for coverage.
	Init(true)
}

func TestHeaderBox(t *testing.T) {
	got := HeaderBox("Title", "Subtitle")
	if !strings.Contains(got, "Title") {
		t.Errorf("HeaderBox missing title, got %q", got)
	}

	// Empty subtitle is a valid branch.
	got = HeaderBox("Just Title", "")
	if !strings.Contains(got, "Just Title") {
		t.Error("HeaderBox missing title when subtitle empty")
	}
}

func TestSectionHeader(t *testing.T) {
	got := SectionHeader("Verification")
	if !strings.Contains(got, "Verification") {
		t.Errorf("SectionHeader missing name, got %q", got)
	}
}

func TestSuccessBanner(t *testing.T) {
	got := SuccessBanner("DONE")
	if !strings.Contains(got, "DONE") {
		t.Errorf("SuccessBanner missing text, got %q", got)
	}
}

func TestFailureBanner(t *testing.T) {
	got := FailureBanner("FAILED")
	if !strings.Contains(got, "FAILED") {
		t.Errorf("FailureBanner missing text, got %q", got)
	}
}

func TestStyleBuilders(t *testing.T) {
	// Just invoke the style builders to ensure they don't panic and
	// return usable lipgloss.Style values.
	if LabelStyle().Render("x") == "" {
		t.Error("LabelStyle().Render should produce output")
	}
	if ValueStyle().Render("x") == "" {
		t.Error("ValueStyle().Render should produce output")
	}
	if FaintStyle().Render("x") == "" {
		t.Error("FaintStyle().Render should produce output")
	}
	if AccentBoldStyle().Render("x") == "" {
		t.Error("AccentBoldStyle().Render should produce output")
	}
}

func TestHuhTheme(t *testing.T) {
	// HuhTheme returns a non-nil theme suitable for huh forms.
	_ = HuhTheme()
}

func TestLabelValueStyleFunc(t *testing.T) {
	f := LabelValueStyleFunc()
	// Column 0 is the label; column 1 is the value. Both should return
	// a usable style.
	if f(0, 0).Render("label") == "" {
		t.Error("label cell style should render")
	}
	if f(0, 1).Render("value") == "" {
		t.Error("value cell style should render")
	}
}

func TestHasDarkBackground_DoesNotPanic(t *testing.T) {
	// The result depends on the test environment; we just verify
	// there's no panic and a boolean is returned.
	_ = hasDarkBackground()
}
