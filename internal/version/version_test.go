// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package version

import (
	"strings"
	"testing"
)

func TestShort_Default(t *testing.T) {
	if Short() != Version {
		t.Errorf("Short() should return Version, got %q", Short())
	}
}

func TestFull_ContainsVersion(t *testing.T) {
	result := Full()
	if !strings.Contains(result, Version) {
		t.Errorf("Full() should contain version, got %q", result)
	}
	if !strings.HasPrefix(result, "truestamp ") {
		t.Errorf("Full() should start with 'truestamp ', got %q", result)
	}
}

func TestFull_ContainsCommit(t *testing.T) {
	result := Full()
	if !strings.Contains(result, GitCommit) {
		t.Errorf("Full() should contain commit, got %q", result)
	}
}

func TestFull_ContainsBuildDate(t *testing.T) {
	result := Full()
	if !strings.Contains(result, BuildDate) {
		t.Errorf("Full() should contain build date, got %q", result)
	}
}

func TestCopyright_ContainsCompanyName(t *testing.T) {
	result := Copyright()
	if !strings.Contains(result, "Truestamp, Inc.") {
		t.Errorf("Copyright() should contain company name, got %q", result)
	}
}

func TestCopyright_ContainsStartYear(t *testing.T) {
	result := Copyright()
	if !strings.Contains(result, "2021-") {
		t.Errorf("Copyright() should contain start year 2021, got %q", result)
	}
}

func TestCopyright_ContainsBuildYear(t *testing.T) {
	result := Copyright()
	if !strings.Contains(result, BuildYear) {
		t.Errorf("Copyright() should contain build year, got %q", result)
	}
}

func TestBuildYear_NotEmpty(t *testing.T) {
	// init() should set BuildYear to current year if empty
	if BuildYear == "" {
		t.Error("BuildYear should not be empty after init()")
	}
}

func TestBuildYear_FourDigits(t *testing.T) {
	if len(BuildYear) != 4 {
		t.Errorf("BuildYear should be 4 digits, got %q", BuildYear)
	}
}

func TestPath_MatchesGoMod(t *testing.T) {
	const want = "github.com/truestamp/truestamp-cli"
	if Path != want {
		t.Errorf("Path = %q, want %q (must match go.mod module directive)", Path, want)
	}
}

func TestPlatform_Format(t *testing.T) {
	p := Platform()
	parts := strings.Split(p, "/")
	if len(parts) != 2 {
		t.Fatalf("Platform() = %q, want GOOS/GOARCH", p)
	}
	if parts[0] == "" || parts[1] == "" {
		t.Errorf("Platform() = %q, halves must be non-empty", p)
	}
}

func TestGoVersion_HasGoPrefix(t *testing.T) {
	if !strings.HasPrefix(GoVersion(), "go") {
		t.Errorf("GoVersion() = %q, want %q prefix", GoVersion(), "go")
	}
}

func TestGoFor_Format(t *testing.T) {
	s := GoFor()
	if !strings.HasPrefix(s, "go") {
		t.Errorf("GoFor() = %q, want go prefix", s)
	}
	if !strings.Contains(s, " for ") {
		t.Errorf("GoFor() = %q, want ' for ' separator", s)
	}
	if !strings.Contains(s, Platform()) {
		t.Errorf("GoFor() = %q, want to contain Platform()=%q", s, Platform())
	}
}
