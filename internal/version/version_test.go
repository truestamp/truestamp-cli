// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package version

import (
	"runtime/debug"
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

// populateFromBuildInfo unit tests — exercise every branch of the
// BuildInfo parser without relying on runtime.debug.ReadBuildInfo
// (which is controlled by whatever the test harness built us with).
func TestPopulateFromBuildInfo_UsesMainPath(t *testing.T) {
	// Save and restore all package vars.
	origPath, origVersion, origCommit, origDate := Path, Version, GitCommit, BuildDate
	t.Cleanup(func() { Path, Version, GitCommit, BuildDate = origPath, origVersion, origCommit, origDate })

	Version, GitCommit, BuildDate = "dev", "unknown", "unknown"
	info := &debug.BuildInfo{
		Main: debug.Module{Path: "example.com/other", Version: "v1.2.3"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef1234567890"},
			{Key: "vcs.time", Value: "2024-01-15T10:00:00Z"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	populateFromBuildInfo(info)

	if Path != "example.com/other" {
		t.Errorf("Path: got %q, want example.com/other", Path)
	}
	if Version != "v1.2.3-dirty" {
		t.Errorf("Version should be bumped to -dirty, got %q", Version)
	}
	if GitCommit != "abcdef123" {
		t.Errorf("GitCommit should be truncated to 9 chars, got %q", GitCommit)
	}
	if BuildDate != "2024-01-15T10:00:00Z" {
		t.Errorf("BuildDate: got %q", BuildDate)
	}
}

func TestPopulateFromBuildInfo_CommandLineArgsPath(t *testing.T) {
	// When Main.Path is "command-line-arguments" (go run, go build .),
	// the existing Path value must be preserved.
	origPath, origVersion := Path, Version
	Path = "already-set"
	Version = "dev"
	t.Cleanup(func() { Path, Version = origPath, origVersion })

	populateFromBuildInfo(&debug.BuildInfo{Main: debug.Module{Path: "command-line-arguments"}})
	if Path != "already-set" {
		t.Errorf("Path should be preserved for command-line-arguments, got %q", Path)
	}
}

func TestPopulateFromBuildInfo_DevelVersion(t *testing.T) {
	// (devel) Main.Version should be treated as no version, leaving
	// the existing "dev" in place.
	orig := Version
	Version = "dev"
	t.Cleanup(func() { Version = orig })

	populateFromBuildInfo(&debug.BuildInfo{Main: debug.Module{Version: "(devel)"}})
	if Version != "dev" {
		t.Errorf("(devel) should not overwrite Version, got %q", Version)
	}
}

func TestPopulateFromBuildInfo_ExplicitLdflagsVersion(t *testing.T) {
	// When Version is already populated via ldflags, BuildInfo should
	// not overwrite it and the "-dirty" suffix should not be appended.
	origVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() { Version = origVersion })

	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v2.0.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.modified", Value: "true"},
		},
	}
	populateFromBuildInfo(info)
	if Version != "1.2.3" {
		t.Errorf("ldflags Version should be preserved, got %q", Version)
	}
}

func TestPopulateFromBuildInfo_ShortCommit(t *testing.T) {
	origCommit := GitCommit
	GitCommit = "unknown"
	t.Cleanup(func() { GitCommit = origCommit })

	populateFromBuildInfo(&debug.BuildInfo{
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}},
	})
	if GitCommit != "abc" {
		t.Errorf("short commit should not be truncated, got %q", GitCommit)
	}
}
