// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

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
	// cleanEnv already strips the developer's TRUESTAMP_* variables but the
	// dead TRUESTAMP_BASE_URL; the later entries win over its placeholder
	// HOME / XDG values.
	return append(cleanEnv(),
		"XDG_CONFIG_HOME="+dir,
		"HOME="+dir,
		"APPDATA="+dir, // windows configDir()
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
	// Without the dead TRUESTAMP_BASE_URL, which would win over the file's
	// base_url: the origin is then the file's .invalid host.
	for _, e := range isolatedEnv(dir) {
		if !strings.HasPrefix(e, "TRUESTAMP_BASE_URL=") {
			cmd.Env = append(cmd.Env, e)
		}
	}
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

// TestCLI_TeamsUseClear_WritesToConfigFlag is the CLI-level write-path
// regression. `teams use --clear` is the one writer reachable without a live
// server: it short-circuits on appConfig.Team and calls config.SetTeam("")
// with no network access at all.
//
// Before the fix this test failed twice over, the custom file kept
// `team = "..."` and the platform default was created and written.
func TestCLI_TeamsUseClear_WritesToConfigFlag(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.toml")
	// An unrelated setting that must survive the rewrite. Deliberately NOT
	// `silent`: since Phase 0 that key is CLI-wide, so setting it here
	// would silence the very command under test.
	original := `base_url = "https://example.invalid"
team = "019dbd00-0000-7000-8000-000000000000"

[hash]
algorithm = "sha512"
`
	if err := os.WriteFile(custom, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "teams", "use", "--clear", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("teams use --clear failed: %s\n%s", err, out)
	}

	data, rerr := os.ReadFile(custom)
	if rerr != nil {
		t.Fatalf("read custom config: %v", rerr)
	}
	if !strings.Contains(string(data), `team = ""`) {
		t.Errorf("teams use --clear --config did not clear the team in %q, got:\n%s", custom, data)
	}
	// Unrelated settings must survive the rewrite.
	if !strings.Contains(string(data), `algorithm = "sha512"`) {
		t.Errorf("teams use --clear clobbered unrelated settings in %q, got:\n%s", custom, data)
	}
	// The platform default must be untouched, not even created.
	if _, serr := os.Stat(platformDefault(dir)); !os.IsNotExist(serr) {
		t.Errorf("teams use --clear --config must not write the platform default %q (stat err = %v)",
			platformDefault(dir), serr)
	}

	// The write is visible on the next read through the same --config.
	cmd = exec.Command(binaryPath, "teams", "use", "--clear", "--config", custom)
	cmd.Env = isolatedEnv(dir)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("second teams use --clear failed: %s\n%s", err, out)
	}
	if !strings.Contains(string(out), "No team is configured") {
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

// maskUserinfo masks everything before a URL's last @, but its scheme,
// since any @ after the scheme's // could end user information: a URL
// that parses with a host and a path, such as http://u:12/ss@host, may be
// a password typed with a raw slash. A URL with no @ is shown whole.
// Mutants killed: showing whole a URL that parses with a host and no user
// information (http://u:12/ss@host shows 12/ss); masking only up to the
// first @ (a password holding an @ shows its tail); returning the URL
// whole; the scheme taken from a :// after the last @ (a URL whose query
// holds another URL shows its password).
func TestMaskUserinfo(t *testing.T) {
	for raw, want := range map[string]string{
		"http://u:12/ss@host":                        "http://[masked]@host",
		"http://u:12?ss@host":                        "http://[masked]@host",
		"http://user:pass@core.example:8332":         "http://[masked]@core.example:8332",
		"http://127.0.0.1:8332/wallet/x@y":           "http://[masked]@y",
		"https://esplora.example/api?who=a@b":        "https://[masked]@b",
		"http://user:pass@core.example:8332/w/x@y":   "http://[masked]@y",
		"http://us@er:p@ss@core.example:8332":        "http://[masked]@core.example:8332",
		"http://user:pa/ss@core.example:8332":        "http://[masked]@core.example:8332",
		"user:pass@core.example:8332":                "[masked]@core.example:8332",
		"user:pass@core.example:8332/?next=http://x": "[masked]@core.example:8332/?next=http://x",
		"https://horizon.example":                    "https://horizon.example",
		"http://127.0.0.1:8332/wallet/x":             "http://127.0.0.1:8332/wallet/x",
		"http://user@core.example":                   "http://[masked]@core.example",
		"http://user:secret@[::1]:8332/wallet/a@b#c": "http://[masked]@b#c",
	} {
		if got := maskUserinfo(raw); got != want {
			t.Errorf("maskUserinfo(%q) = %q, want %q", raw, got, want)
		}
	}
}
