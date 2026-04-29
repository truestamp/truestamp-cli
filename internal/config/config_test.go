// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

// --- Timeout -----------------------------------------------------------

func TestTimeout_ValidDuration(t *testing.T) {
	c := Config{HTTPTimeout: "25s"}
	if got := c.Timeout(); got != 25*time.Second {
		t.Errorf("Timeout: got %v, want 25s", got)
	}
}

func TestTimeout_FallbackOnInvalid(t *testing.T) {
	c := Config{HTTPTimeout: "not-a-duration"}
	if got := c.Timeout(); got != 10*time.Second {
		t.Errorf("Timeout fallback: got %v, want 10s", got)
	}
}

// --- Load: precedence chain (defaults → file → env → flags) ------------

func TestLoad_DefaultsOnly(t *testing.T) {
	// No config file, no env vars, no flags.
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.toml")

	// Isolate env. t.Setenv-then-Unsetenv preserves auto-restore.
	for _, v := range []string{
		"TRUESTAMP_BASE_URL", "TRUESTAMP_API_KEY", "TRUESTAMP_TEAM",
		"TRUESTAMP_HTTP_TIMEOUT",
		"TRUESTAMP_HASH_ALGORITHM", "TRUESTAMP_HASH_ENCODING",
		"TRUESTAMP_HASH_STYLE", "TRUESTAMP_CONVERT_TIME_ZONE",
		"TRUESTAMP_VERIFY_SILENT", "TRUESTAMP_VERIFY_JSON",
	} {
		t.Setenv(v, "sentinel") // mark as set so Setenv records it
		_ = os.Unsetenv(v)      // now actually remove for the test body
	}

	cfg, err := Load(path, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "https://www.truestamp.com" {
		t.Errorf("default base_url: got %q", cfg.BaseURL)
	}
	if cfg.APIURL != "https://www.truestamp.com/api/json" {
		t.Errorf("computed api_url: got %q", cfg.APIURL)
	}
	if cfg.KeyringURL != "https://www.truestamp.com/.well-known/keyring.json" {
		t.Errorf("computed keyring_url: got %q", cfg.KeyringURL)
	}
	if cfg.WebSocketURL != "wss://www.truestamp.com/console/websocket" {
		t.Errorf("computed websocket_url: got %q", cfg.WebSocketURL)
	}
	if cfg.HealthURL != "https://www.truestamp.com/health" {
		t.Errorf("computed health_url: got %q", cfg.HealthURL)
	}
	if cfg.HTTPTimeout != "10s" {
		t.Errorf("default http_timeout: got %q", cfg.HTTPTimeout)
	}
	if cfg.Hash.Algorithm != "sha256" {
		t.Errorf("default hash.algorithm: got %q", cfg.Hash.Algorithm)
	}
}

func TestLoad_FileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	tomlBody := `
base_url = "https://file.example.com"
http_timeout = "25s"
[hash]
algorithm = "sha512"
`
	if err := os.WriteFile(path, []byte(tomlBody), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "https://file.example.com" {
		t.Errorf("base_url: got %q", cfg.BaseURL)
	}
	if cfg.APIURL != "https://file.example.com/api/json" {
		t.Errorf("computed api_url: got %q", cfg.APIURL)
	}
	if cfg.HTTPTimeout != "25s" {
		t.Errorf("http_timeout: got %q", cfg.HTTPTimeout)
	}
	if cfg.Hash.Algorithm != "sha512" {
		t.Errorf("hash.algorithm: got %q", cfg.Hash.Algorithm)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(path, []byte(`base_url = "https://file.example.com"`), 0644)
	t.Setenv("TRUESTAMP_BASE_URL", "https://from-env.example.com")
	t.Setenv("TRUESTAMP_HASH_ALGORITHM", "sha3-256")

	cfg, err := Load(path, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "https://from-env.example.com" {
		t.Errorf("env should override file: got %q", cfg.BaseURL)
	}
	if cfg.Hash.Algorithm != "sha3-256" {
		t.Errorf("hash.algorithm env: got %q", cfg.Hash.Algorithm)
	}
}

func TestLoad_FlagOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(path, []byte(`base_url = "https://file.example.com"`), 0644)
	t.Setenv("TRUESTAMP_BASE_URL", "https://from-env.example.com")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("base-url", "", "")
	flags.String("algorithm", "", "")
	if err := flags.Parse([]string{"--base-url", "https://from-flag.example.com", "--algorithm", "blake2b-512"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path, flags)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "https://from-flag.example.com" {
		t.Errorf("flag should override env: got %q", cfg.BaseURL)
	}
	if cfg.Hash.Algorithm != "blake2b-512" {
		t.Errorf("flag hash.algorithm: got %q", cfg.Hash.Algorithm)
	}
}

// TestLoad_ComposesAllServiceURLs checks that BaseURL drives every
// derived URL — JSON API, keyring, console WebSocket (with scheme
// swap), and health — and that paths/queries on the configured base
// are normalised away so composition is deterministic.
func TestLoad_ComposesAllServiceURLs(t *testing.T) {
	cases := []struct {
		name           string
		base           string
		wantAPI        string
		wantKeyring    string
		wantWebSocket  string
		wantHealth     string
		wantNormalised string // BaseURL after Load (path stripped)
	}{
		{
			name:           "https origin",
			base:           "https://www.truestamp.com",
			wantAPI:        "https://www.truestamp.com/api/json",
			wantKeyring:    "https://www.truestamp.com/.well-known/keyring.json",
			wantWebSocket:  "wss://www.truestamp.com/console/websocket",
			wantHealth:     "https://www.truestamp.com/health",
			wantNormalised: "https://www.truestamp.com",
		},
		{
			name:           "http localhost preview",
			base:           "http://localhost:4010",
			wantAPI:        "http://localhost:4010/api/json",
			wantKeyring:    "http://localhost:4010/.well-known/keyring.json",
			wantWebSocket:  "ws://localhost:4010/console/websocket",
			wantHealth:     "http://localhost:4010/health",
			wantNormalised: "http://localhost:4010",
		},
		{
			name:           "trailing path stripped",
			base:           "https://staging.example.com/whatever?x=1",
			wantAPI:        "https://staging.example.com/api/json",
			wantKeyring:    "https://staging.example.com/.well-known/keyring.json",
			wantWebSocket:  "wss://staging.example.com/console/websocket",
			wantHealth:     "https://staging.example.com/health",
			wantNormalised: "https://staging.example.com",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.toml")
			_ = os.WriteFile(path, []byte(`base_url = `+`"`+tc.base+`"`), 0644)
			cfg, err := Load(path, nil)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.BaseURL != tc.wantNormalised {
				t.Errorf("BaseURL normalised: got %q, want %q", cfg.BaseURL, tc.wantNormalised)
			}
			if cfg.APIURL != tc.wantAPI {
				t.Errorf("APIURL: got %q, want %q", cfg.APIURL, tc.wantAPI)
			}
			if cfg.KeyringURL != tc.wantKeyring {
				t.Errorf("KeyringURL: got %q, want %q", cfg.KeyringURL, tc.wantKeyring)
			}
			if cfg.WebSocketURL != tc.wantWebSocket {
				t.Errorf("WebSocketURL: got %q, want %q", cfg.WebSocketURL, tc.wantWebSocket)
			}
			if cfg.HealthURL != tc.wantHealth {
				t.Errorf("HealthURL: got %q, want %q", cfg.HealthURL, tc.wantHealth)
			}
		})
	}
}

// TestLoad_BaseURLValidation rejects a base_url that is not a real
// origin (missing scheme or host). A typo should surface at Load
// rather than degrade to a silent fallback at request time.
func TestLoad_BaseURLValidation(t *testing.T) {
	cases := []string{
		`base_url = ""`,
		`base_url = "not-a-url"`,
		`base_url = "://broken"`,
	}
	for _, body := range cases {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		_ = os.WriteFile(path, []byte(body), 0644)
		if _, err := Load(path, nil); err == nil {
			t.Errorf("Load(%q): expected validation error, got nil", body)
		}
	}
}

// TestLoad_LegacyKeysWarn confirms the migration aid: a config still
// using api_url / keyring_url loads without erroring (the values are
// ignored — defaults take over) but a stderr warning fires so the
// user knows to migrate.
func TestLoad_LegacyKeysWarn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(path, []byte(`
api_url = "https://old.example/api/json"
keyring_url = "https://old.example/keyring.json"
`), 0644)

	// Capture stderr.
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	cfg, err := Load(path, nil)
	w.Close()
	os.Stderr = origStderr
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Defaults take effect; the legacy values are ignored.
	if cfg.BaseURL != "https://www.truestamp.com" {
		t.Errorf("legacy api_url should not change BaseURL: got %q", cfg.BaseURL)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	stderr := string(buf[:n])
	if !strings.Contains(stderr, "no longer recognized") || !strings.Contains(stderr, "base_url") {
		t.Errorf("expected legacy-keys warning on stderr, got:\n%s", stderr)
	}
}

func TestLoad_InvalidHTTPTimeout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(path, []byte(`http_timeout = "nope"`), 0644)
	_, err := Load(path, nil)
	if err == nil {
		t.Fatal("expected validation error for bad http_timeout")
	}
	if !strings.Contains(err.Error(), "http_timeout") {
		t.Errorf("error should mention http_timeout, got: %v", err)
	}
}

// TestLoad_NonPositiveHTTPTimeout rejects zero and negative durations
// since http.Client treats a zero timeout as "no timeout", which would
// silently disable the guard.
func TestLoad_NonPositiveHTTPTimeout(t *testing.T) {
	cases := []string{`http_timeout = "0s"`, `http_timeout = "-1s"`}
	for _, body := range cases {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		_ = os.WriteFile(path, []byte(body), 0644)
		_, err := Load(path, nil)
		if err == nil {
			t.Errorf("Load with %q: expected error, got nil", body)
			continue
		}
		if !strings.Contains(err.Error(), "positive") {
			t.Errorf("Load with %q: error should mention 'positive', got: %v", body, err)
		}
	}
}

// TestLoad_CosignPathAbsolute requires the resolved cosign_path to be
// absolute (or empty for $PATH lookup). Relative paths are rejected at
// load time so downstream callers can assume a validated value.
func TestLoad_CosignPathAbsolute(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"empty is ok (falls back to $PATH)", `cosign_path = ""`, false},
		{"absolute path is ok", `cosign_path = "/usr/local/bin/cosign"`, false},
		{"relative path is rejected", `cosign_path = "cosign"`, true},
		{"dot-relative path is rejected", `cosign_path = "./bin/cosign"`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.toml")
			_ = os.WriteFile(path, []byte(tc.body), 0644)
			_, err := Load(path, nil)
			if tc.wantErr && err == nil {
				t.Errorf("Load with %q: expected error, got nil", tc.body)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Load with %q: unexpected error %v", tc.body, err)
			}
			if tc.wantErr && err != nil && !strings.Contains(err.Error(), "cosign_path") {
				t.Errorf("Load with %q: error should mention 'cosign_path', got: %v", tc.body, err)
			}
		})
	}
}

// TestLoad_CosignPathFromEnv verifies TRUESTAMP_COSIGN_PATH flows
// through the koanf env provider into Config.CosignPath just like the
// other TRUESTAMP_* env vars. Env takes precedence over the (empty)
// config file value.
func TestLoad_CosignPathFromEnv(t *testing.T) {
	// t.Setenv precludes t.Parallel() on this specific test; other
	// cases in this file that don't mutate env can still parallelize.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(path, []byte(""), 0644)

	t.Setenv("TRUESTAMP_COSIGN_PATH", "/opt/cosign/bin/cosign")
	cfg, err := Load(path, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CosignPath != "/opt/cosign/bin/cosign" {
		t.Errorf("CosignPath from env: got %q, want %q", cfg.CosignPath, "/opt/cosign/bin/cosign")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(path, []byte("not valid toml ] ["), 0644)
	_, err := Load(path, nil)
	if err == nil {
		t.Fatal("expected parse error for malformed TOML")
	}
}

// --- ToTOML -----------------------------------------------------------

func TestToTOML_IncludesAllSections(t *testing.T) {
	c := &Config{
		BaseURL:     "https://x.example.com",
		APIKey:      "abcdef1234567890",
		Team:        "t",
		HTTPTimeout: "15s",
		Verify:      VerifyConfig{Silent: true, JSON: true, SkipExternal: true},
		Hash:        HashConfig{Algorithm: "sha256", Encoding: "hex", Style: "gnu"},
		Convert:     ConvertConfig{TimeZone: "UTC"},
	}
	out := c.ToTOML(false)
	for _, want := range []string{
		`base_url = "https://x.example.com"`,
		`api_key = "abcdef1234567890"`,
		`team = "t"`,
		`[verify]`,
		`silent = true`,
		`[hash]`,
		`algorithm = "sha256"`,
		`[convert]`,
		`time_zone = "UTC"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ToTOML missing %q, full output:\n%s", want, out)
		}
	}
	// Computed URL fields should NOT leak into rendered TOML — copying
	// the output back to a config file would otherwise hardcode stale
	// derived URLs that base_url no longer drives.
	for _, leak := range []string{`api_url`, `keyring_url`, `health_url`, `websocket_url`} {
		if strings.Contains(out, leak+" =") {
			t.Errorf("ToTOML leaked computed field %q:\n%s", leak, out)
		}
	}
}

func TestToTOML_MasksAPIKey(t *testing.T) {
	c := &Config{APIKey: "abcdef1234567890"}
	out := c.ToTOML(true)
	if strings.Contains(out, "abcdef1234567890") {
		t.Errorf("masked output must not contain full key, got:\n%s", out)
	}
	if !strings.Contains(out, "abcd...7890") {
		t.Errorf("masked output should show partial key, got:\n%s", out)
	}
}

func TestToTOML_MasksShortKey(t *testing.T) {
	c := &Config{APIKey: "short"}
	out := c.ToTOML(true)
	if !strings.Contains(out, `api_key = "****"`) {
		t.Errorf("short key should mask to ****, got:\n%s", out)
	}
}

func TestToTOML_EmptyAPIKey_NoMasking(t *testing.T) {
	c := &Config{APIKey: ""}
	out := c.ToTOML(true)
	if !strings.Contains(out, `api_key = ""`) {
		t.Errorf("empty key should render as empty string, got:\n%s", out)
	}
}

// --- ConfigDir --------------------------------------------------------

func TestConfigDir_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/explicit/xdg")
	t.Setenv("HOME", "/home/user")
	// ConfigDir branches on runtime.GOOS at call time; on non-Windows
	// the XDG_CONFIG_HOME override should win.
	if runtimeIsWindows() {
		t.Skip("XDG path not used on Windows")
	}
	got := ConfigDir()
	if got != "/explicit/xdg/truestamp" {
		t.Errorf("ConfigDir with XDG: got %q", got)
	}
}

func TestConfigDir_HomeFallback(t *testing.T) {
	if runtimeIsWindows() {
		t.Skip("POSIX-only fallback path")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/fallback")
	got := ConfigDir()
	if got != "/home/fallback/.config/truestamp" {
		t.Errorf("ConfigDir with HOME: got %q", got)
	}
}

func TestConfigDir_NoHome(t *testing.T) {
	if runtimeIsWindows() {
		t.Skip("POSIX-only path")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	// ConfigDir returns "" when no HOME is set.
	if got := ConfigDir(); got != "" {
		t.Errorf("ConfigDir with no HOME: got %q, want \"\"", got)
	}
}

// TestLoad_DefaultPathFallback exercises the `configPath == ""` branch
// that calls ConfigFilePath() to compute the path.
func TestLoad_DefaultPathFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	// No config file exists here; Load should fall back to defaults
	// when the resolved path is missing, not error.
	cfg, err := Load("", nil)
	if err != nil {
		t.Fatalf("Load(\"\"): %v", err)
	}
	if cfg.BaseURL == "" {
		t.Error("expected defaults to populate BaseURL")
	}
	if cfg.APIURL == "" {
		t.Error("expected defaults to populate computed APIURL")
	}
}

// TestLoad_UnknownFlagInFlagMapIsIgnored ensures flags that are not in
// flagKeyMap don't error — the closure returns ("", nil) and koanf
// skips them.
func TestLoad_UnknownFlagInFlagMapIsIgnored(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("not-in-map", "", "")
	if err := flags.Parse([]string{"--not-in-map", "value"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load("", flags)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg == nil {
		t.Fatal("nil config")
	}
}

// --- EnsureDefaultConfig error paths ----------------------------------

func TestEnsureDefaultConfig_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	// First call creates it.
	created, err := EnsureDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("first call should create the file")
	}
	// Second call should not create.
	created, err = EnsureDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("second call should not re-create an existing file")
	}
}

// --- helper -----------------------------------------------------------

func runtimeIsWindows() bool {
	// Avoid importing runtime in this test file's public surface; inline
	// the check via the goos file-separator heuristic.
	return os.PathSeparator == '\\'
}
