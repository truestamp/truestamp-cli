// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// isolateConfig points the platform default at a fresh temp dir and
// guarantees the process-wide --config override is cleared both before
// and after the test, so neither a leaked override from an earlier test
// nor one this test sets can reach the developer's real config file.
func isolateConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir) // windows configDir()
	SetActivePath("")
	t.Cleanup(func() { SetActivePath("") })
	return dir
}

// --- ActivePath resolution -------------------------------------------

func TestActivePath_DefaultsToPlatformPath(t *testing.T) {
	isolateConfig(t)

	if got, want := ActivePath(), ConfigFilePath(); got != want {
		t.Errorf("ActivePath with no override: got %q, want %q", got, want)
	}
}

func TestActivePath_HonorsOverride(t *testing.T) {
	dir := isolateConfig(t)
	custom := filepath.Join(dir, "elsewhere", "custom.toml")

	SetActivePath(custom)

	if got := ActivePath(); got != custom {
		t.Errorf("ActivePath with override: got %q, want %q", got, custom)
	}
	// ConfigFilePath keeps meaning "platform default", cmd/root.go's
	// --config help text depends on it never following the override.
	if got := ConfigFilePath(); got == custom {
		t.Error("ConfigFilePath must not follow the --config override")
	}
}

func TestActivePath_EmptyOverrideClears(t *testing.T) {
	dir := isolateConfig(t)

	SetActivePath(filepath.Join(dir, "custom.toml"))
	SetActivePath("")

	if got, want := ActivePath(), ConfigFilePath(); got != want {
		t.Errorf("ActivePath after clearing override: got %q, want %q", got, want)
	}
}

// TestLoad_RecordsActivePath is the linchpin: a single assignment in
// Load is what makes every downstream display + write site agree with
// the file that was actually read.
// TestLoad_DoesNotMutateActivePath pins the contract that makes the
// active path safe: Load reads, it does not publish. Recording the
// override inside Load would leak between callers, because a Load of a
// throwaway file would silently retarget SetAPIKey / SetTeam /
// EnsureDefaultConfig at a directory that is about to be removed. The
// single CLI caller publishes the path explicitly once Load succeeds.
func TestLoad_DoesNotMutateActivePath(t *testing.T) {
	dir := isolateConfig(t)
	custom := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(custom, []byte(`base_url = "https://example.invalid"`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(custom, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Sanity: the custom file really was the one read.
	if cfg.BaseURL != "https://example.invalid" {
		t.Fatalf("custom file not loaded, BaseURL = %q", cfg.BaseURL)
	}
	if got, want := ActivePath(), ConfigFilePath(); got != want {
		t.Errorf("Load must not publish its argument: ActivePath = %q, want the untouched default %q", got, want)
	}
}

// TestLoad_FailureDoesNotMutateActivePath is the sharper half: a Load
// that errors must not leave a bad path installed for the writers.
func TestLoad_FailureDoesNotMutateActivePath(t *testing.T) {
	dir := isolateConfig(t)
	bad := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(bad, []byte("this is not = valid = toml\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(bad, nil); err == nil {
		t.Fatal("Load of malformed TOML should fail")
	}
	if got, want := ActivePath(), ConfigFilePath(); got != want {
		t.Errorf("a failed Load installed %q as the active path; want %q", got, want)
	}
}

// TestSetActivePath_EmptyRestoresDefault covers the no---config case the
// CLI hits on every ordinary invocation: root.go publishes the empty flag
// value, which must resolve back to the platform default rather than
// pinning whatever was set before.
func TestSetActivePath_EmptyRestoresDefault(t *testing.T) {
	dir := isolateConfig(t)

	SetActivePath(filepath.Join(dir, "stale.toml"))
	SetActivePath("")

	if got, want := ActivePath(), ConfigFilePath(); got != want {
		t.Errorf("SetActivePath(\"\") must restore the default: got %q, want %q", got, want)
	}
}

// TestActivePath_Concurrent guards the package-level state against the
// data race that `task test-race` would otherwise report.
func TestActivePath_Concurrent(t *testing.T) {
	isolateConfig(t)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				SetActivePath("")
			} else {
				SetActivePath(filepath.Join(t.TempDir(), "c.toml"))
			}
			_ = ActivePath()
		}(i)
	}
	wg.Wait()
}

// --- EnsureDefaultConfig honours the override -------------------------

func TestEnsureDefaultConfig_CreatesAtOverride(t *testing.T) {
	dir := isolateConfig(t)
	// A nested directory that does not exist yet, the directory must be
	// derived from the active path, not from ConfigDir().
	custom := filepath.Join(dir, "nested", "custom.toml")
	SetActivePath(custom)

	created, err := EnsureDefaultConfig()
	if err != nil {
		t.Fatalf("EnsureDefaultConfig: %v", err)
	}
	if !created {
		t.Error("expected the override file to be created")
	}
	if _, err := os.Stat(custom); err != nil {
		t.Errorf("override config not created: %v", err)
	}
	if _, err := os.Stat(ConfigFilePath()); !os.IsNotExist(err) {
		t.Errorf("platform default %q must not be created, stat err = %v", ConfigFilePath(), err)
	}

	// Idempotent.
	created, err = EnsureDefaultConfig()
	if err != nil {
		t.Fatalf("EnsureDefaultConfig (2nd): %v", err)
	}
	if created {
		t.Error("second call should not re-create an existing file")
	}
}

// --- the write-path regression ---------------------------------------

// TestSetTeam_WritesToOverride is the data-integrity regression: before
// the fix SetTeam ignored --config entirely, reporting success while
// writing the platform default and leaving the targeted file untouched.
func TestSetTeam_WritesToOverride(t *testing.T) {
	dir := isolateConfig(t)
	custom := filepath.Join(dir, "custom.toml")
	original := `base_url = "https://example.invalid"
team = "old-team"

[verify]
silent = true
`
	if err := os.WriteFile(custom, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	SetActivePath(custom)

	const newTeam = "019dbd00-0000-7000-8000-000000000000"
	if err := SetTeam(newTeam); err != nil {
		t.Fatalf("SetTeam: %v", err)
	}

	got, err := os.ReadFile(custom)
	if err != nil {
		t.Fatalf("read override: %v", err)
	}
	if !strings.Contains(string(got), `team = "`+newTeam+`"`) {
		t.Errorf("override file not updated, got:\n%s", got)
	}
	// Unrelated settings survive.
	if !strings.Contains(string(got), "silent = true") {
		t.Errorf("override file clobbered, got:\n%s", got)
	}
	// And the platform default was never touched.
	if _, err := os.Stat(ConfigFilePath()); !os.IsNotExist(err) {
		t.Errorf("platform default %q must not be written, stat err = %v", ConfigFilePath(), err)
	}
}

func TestSetAPIKey_WritesToOverride(t *testing.T) {
	dir := isolateConfig(t)
	custom := filepath.Join(dir, "custom.toml")
	SetActivePath(custom)

	if err := SetAPIKey("ts_override_key"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}

	got, err := os.ReadFile(custom)
	if err != nil {
		t.Fatalf("read override: %v", err)
	}
	if !strings.Contains(string(got), `api_key = "ts_override_key"`) {
		t.Errorf("override file missing api_key, got:\n%s", got)
	}
	info, err := os.Stat(custom)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("override file permissions: got %o, want 0600", mode)
	}
	if _, err := os.Stat(ConfigFilePath()); !os.IsNotExist(err) {
		t.Errorf("platform default %q must not be written, stat err = %v", ConfigFilePath(), err)
	}
}

// TestSetTeam_RoundTripsThroughLoad closes the loop end to end: what a
// write with --config puts on disk is what the next read with the same
// --config sees.
func TestSetTeam_RoundTripsThroughLoad(t *testing.T) {
	dir := isolateConfig(t)
	custom := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(custom, []byte("base_url = \"https://example.invalid\"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// Mirror what the CLI does: load the file, then publish it as the
	// active path. Load deliberately does not publish on its own.
	if _, err := Load(custom, nil); err != nil {
		t.Fatalf("Load: %v", err)
	}
	SetActivePath(custom)

	const teamID = "019dbd00-0000-7000-8000-000000000001"
	if err := SetTeam(teamID); err != nil {
		t.Fatalf("SetTeam: %v", err)
	}

	cfg, err := Load(custom, nil)
	if err != nil {
		t.Fatalf("Load (reread): %v", err)
	}
	if cfg.Team != teamID {
		t.Errorf("team did not round-trip through --config: got %q, want %q", cfg.Team, teamID)
	}
}
