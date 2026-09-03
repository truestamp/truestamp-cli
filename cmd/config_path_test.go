// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These subprocess tests pin the `--config <path>` contract: the file the
// CLI *reads* must also be the file it *reports* and the file it *writes*.
// Before the ActivePath fix, `--config` was honoured only on the read
// path, `config path` printed the platform default, and `team set` /
// `auth login --api-key` wrote the platform default while reporting
// success, silently discarding the user's change.
//
// Every case redirects both $XDG_CONFIG_HOME and $HOME at a t.TempDir()
// so the platform default itself is a throwaway path, a bug that writes
// there can be asserted on instead of touching the developer's real
// ~/.config/truestamp/config.toml.

// isolatedEnv returns an env slice with the platform config dir pointed
// at dir, plus the noise-suppressing flags every subprocess wants.
func isolatedEnv(dir string) []string {
	return append(os.Environ(),
		"XDG_CONFIG_HOME="+dir,
		"HOME="+dir,
		"APPDATA="+dir, // windows configDir()
		"NO_COLOR=1",
		"TRUESTAMP_NO_UPGRADE_CHECK=1",
	)
}

// platformDefault is where the CLI would write if it ignored --config.
func platformDefault(dir string) string {
	return filepath.Join(dir, "truestamp", "config.toml")
}

func TestCLI_ConfigPath_HonorsConfigFlag(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom", "config.toml")
	if err := os.MkdirAll(filepath.Dir(custom), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(custom, []byte("base_url = \"https://example.invalid\"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "config", "path", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config path failed: %s\n%s", err, out)
	}
	got := string(out)
	if !strings.Contains(got, custom) {
		t.Errorf("config path must print the --config file %q, got:\n%s", custom, got)
	}
	if strings.Contains(got, platformDefault(dir)) {
		t.Errorf("config path must not print the platform default when --config is given, got:\n%s", got)
	}
}

func TestCLI_ConfigPath_DefaultsToPlatformPath(t *testing.T) {
	dir := t.TempDir()

	cmd := exec.Command(binaryPath, "config", "path")
	cmd.Env = isolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config path failed: %s\n%s", err, out)
	}
	if got := string(out); !strings.Contains(got, platformDefault(dir)) {
		t.Errorf("config path without --config must print %q, got:\n%s", platformDefault(dir), got)
	}
}

func TestCLI_ConfigPath_IndicatesExistence(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.toml")

	// Absent.
	cmd := exec.Command(binaryPath, "config", "path", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config path failed: %s\n%s", err, out)
	}
	if got := string(out); !strings.Contains(got, "does not exist") {
		t.Errorf("expected a does-not-exist indication for a missing file, got:\n%s", got)
	}

	// Present.
	if err := os.WriteFile(custom, []byte("base_url = \"https://example.invalid\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command(binaryPath, "config", "path", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config path failed: %s\n%s", err, out)
	}
	got := string(out)
	if !strings.Contains(got, "exists") {
		t.Errorf("expected an exists indication for a present file, got:\n%s", got)
	}
	if strings.Contains(got, "does not exist") {
		t.Errorf("present file reported as missing, got:\n%s", got)
	}
}

func TestCLI_ConfigInit_HonorsConfigFlag(t *testing.T) {
	dir := t.TempDir()
	// A directory that does not exist yet: `config init` has to derive it
	// from the --config path, not from the platform config dir.
	custom := filepath.Join(dir, "nested", "custom.toml")

	cmd := exec.Command(binaryPath, "config", "init", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config init failed: %s\n%s", err, out)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Errorf("config init did not create %q: %v", custom, err)
	}
	if _, err := os.Stat(platformDefault(dir)); !os.IsNotExist(err) {
		t.Errorf("config init must not create the platform default %q (stat err = %v)",
			platformDefault(dir), err)
	}
	if got := string(out); !strings.Contains(got, custom) {
		t.Errorf("config init should report the created path %q, got:\n%s", custom, got)
	}
}

func TestCLI_ConfigShow_ReportsActiveFile(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(custom, []byte("base_url = \"https://example.invalid\"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "config", "show", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config show failed: %s\n%s", err, out)
	}
	got := string(out)
	// Proof the custom file was read...
	if !strings.Contains(got, "https://example.invalid") {
		t.Fatalf("custom config was not loaded, got:\n%s", got)
	}
	// ...and that the output names it.
	if !strings.Contains(got, custom) {
		t.Errorf("config show should report the active config file %q, got:\n%s", custom, got)
	}
}

// TestCLI_TeamUnset_WritesToConfigFlag is the CLI-level write-path
// regression. `team unset` is the one writer reachable without a live
// server: it short-circuits on appConfig.Team and calls config.SetTeam("")
// with no network access at all.
//
// Before the fix this test failed twice over, the custom file kept
// `team = "..."` and the platform default was created and written.
func TestCLI_TeamUnset_WritesToConfigFlag(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.toml")
	original := `base_url = "https://example.invalid"
team = "019dbd00-0000-7000-8000-000000000000"

[verify]
silent = true
`
	if err := os.WriteFile(custom, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "team", "unset", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("team unset failed: %s\n%s", err, out)
	}

	data, rerr := os.ReadFile(custom)
	if rerr != nil {
		t.Fatalf("read custom config: %v", rerr)
	}
	if !strings.Contains(string(data), `team = ""`) {
		t.Errorf("team unset --config did not clear the team in %q, got:\n%s", custom, data)
	}
	// Unrelated settings must survive the rewrite.
	if !strings.Contains(string(data), "silent = true") {
		t.Errorf("team unset clobbered unrelated settings in %q, got:\n%s", custom, data)
	}
	// The platform default must be untouched, not even created.
	if _, serr := os.Stat(platformDefault(dir)); !os.IsNotExist(serr) {
		t.Errorf("team unset --config must not write the platform default %q (stat err = %v)",
			platformDefault(dir), serr)
	}

	// The write is visible on the next read through the same --config.
	cmd = exec.Command(binaryPath, "team", "unset", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("second team unset failed: %s\n%s", err, out)
	}
	if !strings.Contains(string(out), "No active team") {
		t.Errorf("the cleared team should be visible on re-read, got:\n%s", out)
	}
}

func TestCLI_Version_ReportsActiveConfigPath(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(custom, []byte("base_url = \"https://example.invalid\"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "version", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version failed: %s\n%s", err, out)
	}
	if got := string(out); !strings.Contains(got, custom) {
		t.Errorf("version should report the active config path %q, got:\n%s", custom, got)
	}
}

// TestCLI_ConfigFlagHelp_ShowsPlatformDefault pins cmd/root.go's use of
// ConfigFilePath(): the --config help text describes the DEFAULT, and is
// built at init() time before any config is loaded, so it must never
// follow the override.
func TestCLI_ConfigFlagHelp_ShowsPlatformDefault(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.toml")

	// --config=<path> form: cobra's help fast-path stops flag parsing at
	// --help, so a space-separated value would be read as a subcommand.
	cmd := exec.Command(binaryPath, "--config="+custom, "--help")
	cmd.Env = isolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %s\n%s", err, out)
	}
	if got := string(out); !strings.Contains(got, platformDefault(dir)) {
		t.Errorf("--config help text should describe the platform default %q, got:\n%s",
			platformDefault(dir), got)
	}
}
