// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package config resolves the CLI's runtime configuration from compiled
// defaults, an optional TOML config file, environment variables, and
// command-line flags, in that order, with later sources overriding
// earlier ones.
package config

import (
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

//go:embed defaults/config.toml
var DefaultTOML string

// Config holds the resolved configuration for the CLI.
//
// All Truestamp service URLs derive from a single BaseURL, the JSON
// API, keyring, console WebSocket, and health endpoint paths are
// fixed by the server and computed from BaseURL during Load. Callers
// that need a specific URL read the corresponding field directly
// (cfg.APIURL, cfg.KeyringURL, etc.); they should not be re-derived
// from BaseURL ad hoc, so the path layout stays in one place.
type Config struct {
	// BaseURL is the user-configurable origin (scheme + host + port)
	// for all Truestamp services. Default https://www.truestamp.com.
	// Settable via base_url in config.toml, TRUESTAMP_BASE_URL env,
	// or --base-url flag.
	BaseURL string `koanf:"base_url"`

	// Computed from BaseURL during Load, not user-settable.
	APIURL       string `koanf:"-"`
	KeyringURL   string `koanf:"-"`
	WebSocketURL string `koanf:"-"`
	HealthURL    string `koanf:"-"`

	APIKey string `koanf:"api_key"`
	// APIKeyExplicit is true when api_key came from an intentional
	// override, the --api-key flag or the TRUESTAMP_API_KEY env var,
	// rather than the config file. An explicit key wins over a stored
	// OAuth session so CI/headless behavior stays deterministic. Computed
	// during Load; not user-settable.
	APIKeyExplicit bool `koanf:"-"`

	Team        string `koanf:"team"`
	HTTPTimeout string `koanf:"http_timeout"`
	// CosignPath pins the cosign binary used during `truestamp upgrade`.
	// Must be an absolute path to an executable when set; empty means
	// fall back to $PATH lookup. Settable in config.toml as
	// `cosign_path = "..."` or via the TRUESTAMP_COSIGN_PATH env var.
	CosignPath string        `koanf:"cosign_path"`
	Verify     VerifyConfig  `koanf:"verify"`
	Hash       HashConfig    `koanf:"hash"`
	Convert    ConvertConfig `koanf:"convert"`
	Logging    LoggingConfig `koanf:"logging"`
}

// LoggingConfig holds CLI-wide logging settings. The logger is
// constructed once at the cobra root and shared by every subcommand
// (one-shot tools and the long-lived console TUI) so a single rotated
// JSON file captures every invocation.
type LoggingConfig struct {
	// File overrides the platform-default log file path. Empty means
	// fall back to logging.DefaultPath() (~/Library/Caches/truestamp/
	// truestamp.log on macOS, ~/.cache/truestamp/truestamp.log on
	// Linux, %LOCALAPPDATA%\truestamp\Cache\truestamp.log on Windows).
	File string `koanf:"file"`

	// Level filters output: "debug" | "info" | "warn" | "error".
	// Empty defaults to "info".
	Level string `koanf:"level"`

	// MaxSizeMB is the lumberjack rotation threshold. 0 = library
	// default (10 MB).
	MaxSizeMB int `koanf:"max_size_mb"`

	// MaxBackups is the count of rotated files retained. 0 = 5.
	MaxBackups int `koanf:"max_backups"`

	// MaxAgeDays is how long rotated files are kept. 0 = 14.
	MaxAgeDays int `koanf:"max_age_days"`
}

// Timeout parses the HTTPTimeout string as a Go duration. A zero or
// negative parsed value (which http.Client treats as "no timeout") is
// rejected in favour of the 10-second default, Load validates this up
// front, so reaching the fallback here means the config bypassed Load.
func (c Config) Timeout() time.Duration {
	d, err := time.ParseDuration(c.HTTPTimeout)
	if err != nil || d <= 0 {
		return 10 * time.Second
	}
	return d
}

// VerifyConfig holds verify-subcommand-specific configuration.
type VerifyConfig struct {
	Silent         bool `koanf:"silent"`
	JSON           bool `koanf:"json"`
	SkipExternal   bool `koanf:"skip_external"`
	SkipSignatures bool `koanf:"skip_signatures"`
	Remote         bool `koanf:"remote"`

	// Keyring is the path of a pinned copy of /.well-known/keyring.json
	// used for the Appendix E.17 key binding. Empty means an online run
	// fetches the live keyring from BaseURL and an offline run reports the
	// binding as not checked.
	Keyring string `koanf:"keyring"`
}

// HashConfig holds hash-subcommand-specific configuration.
type HashConfig struct {
	Algorithm string `koanf:"algorithm"`
	Encoding  string `koanf:"encoding"`
	Style     string `koanf:"style"`
}

// ConvertConfig holds convert-subcommand-specific configuration.
type ConvertConfig struct {
	TimeZone string `koanf:"time_zone"`
}

// sectionPrefixes lists known TOML section names for env var mapping.
var sectionPrefixes = []string{"verify", "hash", "convert", "logging"}

// flagKeyMap maps CLI flag names to their koanf key paths.
// Verify-subcommand flags are scoped under "verify."; hash under "hash.";
// convert under "convert."; logging under "logging.".
var flagKeyMap = map[string]string{
	// Root persistent flags
	"base-url":     "base_url",
	"api-key":      "api_key",
	"team":         "team",
	"http-timeout": "http_timeout",
	"log-file":     "logging.file",
	"log-level":    "logging.level",
	// Verify subcommand flags
	"silent":          "verify.silent",
	"json":            "verify.json",
	"skip-external":   "verify.skip_external",
	"offline":         "verify.skip_external",
	"skip-signatures": "verify.skip_signatures",
	"remote":          "verify.remote",
	"keyring":         "verify.keyring",
	// Hash subcommand flags
	"algorithm": "hash.algorithm",
	"encoding":  "hash.encoding",
	"style":     "hash.style",
	// Convert subcommand flags
	"to-zone": "convert.time_zone",
}

// Load reads configuration in priority order: defaults → config file → env → CLI flags.
// configPath overrides the default config file location if non-empty.
// flags should be the merged flag set from the executed command (rootFlags for persistent,
// cmdFlags for command-specific). Only flags that were explicitly set by the user override.
func Load(configPath string, flags *pflag.FlagSet) (*Config, error) {
	k := koanf.New(".")

	// 1. Compiled defaults
	defaults := map[string]interface{}{
		"base_url":               "https://www.truestamp.com",
		"api_key":                "",
		"team":                   "",
		"http_timeout":           "10s",
		"cosign_path":            "",
		"verify.silent":          false,
		"verify.json":            false,
		"verify.skip_external":   false,
		"verify.skip_signatures": false,
		"verify.remote":          false,
		"verify.keyring":         "",
		"hash.algorithm":         "sha256",
		"hash.encoding":          "hex",
		"hash.style":             "gnu",
		"convert.time_zone":      "UTC",
		"logging.file":           "",
		"logging.level":          "info",
		"logging.max_size_mb":    10,
		"logging.max_backups":    5,
		"logging.max_age_days":   14,
	}
	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// 2. Config file (skip if missing)
	//
	// Load deliberately does NOT record configPath as the process-wide
	// active path. Doing so here would make a pure read function mutate
	// global state as a side effect, including on the error paths below,
	// and would leak between callers: every test that loads a temp config
	// would silently retarget SetAPIKey / SetTeam / EnsureDefaultConfig at
	// a directory that no longer exists. The single CLI caller records it
	// explicitly via SetActivePath once Load has succeeded.
	if configPath == "" {
		configPath = ConfigFilePath()
	}
	if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading config file %s: %w", configPath, err)
	}

	// 3. Environment variables (TRUESTAMP_ prefix)
	envProvider := env.Provider(".", env.Opt{
		Prefix: "TRUESTAMP_",
		TransformFunc: func(k, v string) (string, any) {
			key := strings.ToLower(strings.TrimPrefix(k, "TRUESTAMP_"))
			for _, prefix := range sectionPrefixes {
				if strings.HasPrefix(key, prefix+"_") {
					key = prefix + "." + strings.TrimPrefix(key, prefix+"_")
					break
				}
			}
			return key, v
		},
	})
	if err := k.Load(envProvider, nil); err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}

	// 4. CLI flags (only explicitly set ones)
	if flags != nil {
		flagProvider := posflag.ProviderWithFlag(flags, ".", k, func(f *pflag.Flag) (string, interface{}) {
			if !f.Changed {
				return "", nil
			}
			key, ok := flagKeyMap[f.Name]
			if !ok {
				return "", nil
			}
			return key, posflag.FlagVal(flags, f)
		})
		if err := k.Load(flagProvider, nil); err != nil {
			return nil, fmt.Errorf("loading CLI flags: %w", err)
		}
	}

	// Detect legacy keys still lingering in the user's TOML / env / flags
	// (api_url, keyring_url, --api-url, --keyring-url) and warn once on
	// stderr. We don't error on them, the defaults are correct for the
	// production www.truestamp.com host, but the user should know they
	// are no longer load-bearing so they can clean up their config.
	warnLegacyURLKeys(k)

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Record whether the API key was supplied explicitly (env or flag) so
	// the auth resolver can let an explicit key override a stored OAuth
	// session. A key sitting in the config file is not "explicit".
	cfg.APIKeyExplicit = os.Getenv("TRUESTAMP_API_KEY") != "" ||
		(flags != nil && flags.Lookup("api-key") != nil && flags.Changed("api-key"))

	// Compose the canonical Truestamp service URLs from BaseURL. The path
	// layout is fixed by the server (lib/truestamp_web/router.ex) so we
	// own it here as a single source of truth. Failure to parse BaseURL
	// surfaces as an error rather than a silent fallback so the user sees
	// the typo in their config.
	if err := composeTruestampURLs(&cfg); err != nil {
		return nil, err
	}

	// Validate http_timeout is parseable and positive. A zero or negative
	// duration is treated by http.Client as "no timeout", which would
	// silently disable the guard, reject it up front so the user gets
	// a clear error instead of a hung request.
	d, err := time.ParseDuration(cfg.HTTPTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid http_timeout %q: %w", cfg.HTTPTimeout, err)
	}
	if d <= 0 {
		return nil, fmt.Errorf("invalid http_timeout %q: must be a positive duration", cfg.HTTPTimeout)
	}

	// Validate cosign_path: when set, it must be an absolute path.
	// Existence and the executable bit are re-checked lazily at use
	// time in selfupgrade.resolveCosignBinary, the upgrade command is
	// the only consumer, and the filesystem state there can differ from
	// startup (cosign installed/removed between a session's commands).
	if cfg.CosignPath != "" && !filepath.IsAbs(cfg.CosignPath) {
		return nil, fmt.Errorf("invalid cosign_path %q: must be an absolute path (or empty for $PATH lookup)", cfg.CosignPath)
	}

	return &cfg, nil
}

// truestampPaths centralises the well-known path components served by
// the Truestamp Phoenix backend. Callers compose <BaseURL> + path to
// build a full URL; nothing else in the CLI hardcodes these strings.
const (
	truestampAPIPath       = "/api/json"
	truestampKeyringPath   = "/.well-known/keyring.json"
	truestampWebSocketPath = "/console/websocket"
	truestampHealthPath    = "/health"
)

// composeTruestampURLs derives the per-service URLs from cfg.BaseURL
// and stores them on the struct. The BaseURL must include a scheme
// and host; trailing slashes / paths / queries / fragments are
// discarded so a user setting `https://www.truestamp.com/foo` still
// gets correct service URLs.
func composeTruestampURLs(cfg *Config) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("base_url is empty; set it in config.toml, TRUESTAMP_BASE_URL, or --base-url")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base_url %q: %w", cfg.BaseURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid base_url %q: must include scheme and host (e.g. https://www.truestamp.com)", cfg.BaseURL)
	}
	// Normalise: keep only scheme + host, drop everything else so
	// composition is deterministic.
	origin := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
	cfg.BaseURL = origin
	cfg.APIURL = origin + truestampAPIPath
	cfg.KeyringURL = origin + truestampKeyringPath
	cfg.HealthURL = origin + truestampHealthPath
	cfg.WebSocketURL = wsScheme(u.Scheme) + "://" + u.Host + truestampWebSocketPath
	return nil
}

// wsScheme maps an http(s) scheme to its WebSocket counterpart.
// Anything else passes through unchanged so an explicit ws:// /
// wss:// scheme in BaseURL is preserved.
func wsScheme(s string) string {
	switch strings.ToLower(s) {
	case "http":
		return "ws"
	case "https":
		return "wss"
	}
	return s
}

// warnLegacyURLKeys emits a one-line stderr warning if any of the
// retired URL knobs (api_url, keyring_url, --api-url, --keyring-url,
// or their TRUESTAMP_* env equivalents) are still set. This is a
// transition aid: the keys have no effect on the resolved config,
// callers should set base_url instead.
func warnLegacyURLKeys(k *koanf.Koanf) {
	hasOld := k.Exists("api_url") || k.Exists("keyring_url")
	if !hasOld {
		return
	}
	fmt.Fprintln(os.Stderr,
		"warning: 'api_url' and 'keyring_url' are no longer recognized; set 'base_url' instead.")
	fmt.Fprintln(os.Stderr,
		"         All Truestamp service URLs (API, keyring, console, health) derive from base_url.")
}

// ConfigDir returns the config directory path. The implementation is
// split across config_unix.go and config_windows.go via build tags so
// each platform's branch is exercised, and scored for coverage, only
// on the platform where it can actually run.
func ConfigDir() string { return configDir() }

// ConfigFilePath returns the full path to the PLATFORM DEFAULT config
// file, the location used when the user did not pass --config. It
// deliberately ignores any override so cmd/root.go can render it as the
// documented default in the --config flag's help text (built at init()
// time, long before any config is loaded).
//
// Callers asking "which file is actually in effect right now?" want
// [ActivePath] instead.
func ConfigFilePath() string {
	return filepath.Join(ConfigDir(), "config.toml")
}

// activeMu guards activeOverride. SetActivePath writes it, and every
// display and persistence site reads it; TestActivePath_Concurrent
// exercises concurrent SetActivePath / ActivePath calls under -race, so
// the state is mutex-guarded rather than a bare package var.
var (
	activeMu       sync.RWMutex
	activeOverride string
)

// SetActivePath records path as the --config override in effect for the
// rest of the process. The empty string clears the override, restoring
// the platform default. cmd/root.go calls this once [Load] has
// succeeded ([Load] itself deliberately does not; see the note in its
// body), so ordinary CLI flows are already covered; it is exported for
// tests and for callers that resolve the path themselves.
//
// This mirrors the package-global idiom already used by auth.SetDefault
// and httpclient.SetTransport.
func SetActivePath(path string) {
	activeMu.Lock()
	activeOverride = path
	activeMu.Unlock()
}

// activeOverridePath returns the recorded --config override, or "" when
// none is in effect.
func activeOverridePath() string {
	activeMu.RLock()
	defer activeMu.RUnlock()
	return activeOverride
}

// ActivePath returns the config file actually in effect: the --config
// override when one was supplied, otherwise the platform default.
//
// It is correct even when [Load] was never called, some commands and
// tests reach display/persistence sites before (or without) loading the
// config, and they must not silently target a different file than the
// one that was read.
func ActivePath() string {
	if p := activeOverridePath(); p != "" {
		return p
	}
	return ConfigFilePath()
}

// activeConfigDir returns the directory holding the file in effect, or
// "" when it cannot be determined. Without an override that is the
// platform config dir (empty when neither $HOME nor $XDG_CONFIG_HOME is
// resolvable); with an override it is that path's parent, which is
// always determinable.
func activeConfigDir() string {
	if p := activeOverridePath(); p != "" {
		return filepath.Dir(p)
	}
	return ConfigDir()
}

// EnsureDefaultConfig creates the directory of the config file in effect
// and writes the default config.toml there if it does not already exist.
// Returns true if the file was created.
//
// The path comes from [ActivePath], not [ConfigFilePath], so a --config
// override creates (and is subsequently written to) the file the user
// actually named.
func EnsureDefaultConfig() (bool, error) {
	dir := activeConfigDir()
	if dir == "" {
		return false, fmt.Errorf("cannot determine home directory; set $HOME or $XDG_CONFIG_HOME")
	}

	path := ActivePath()

	if _, err := os.Stat(path); err == nil {
		return false, nil // already exists
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(DefaultTOML), 0644); err != nil {
		return false, fmt.Errorf("writing default config: %w", err)
	}

	return true, nil
}

// ToTOML renders the config as a TOML string suitable for display.
// If the API key is set, it is masked.
func (c *Config) ToTOML(maskAPIKey bool) string {
	apiKey := c.APIKey
	if maskAPIKey && apiKey != "" {
		if len(apiKey) <= 8 {
			apiKey = "****"
		} else {
			apiKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
		}
	}

	return fmt.Sprintf(`base_url = %q
api_key = %q
team = %q
http_timeout = %q
cosign_path = %q

[verify]
silent = %v
json = %v
skip_external = %v
skip_signatures = %v
remote = %v
keyring = %q

[hash]
algorithm = %q
encoding = %q
style = %q

[convert]
time_zone = %q

[logging]
file = %q
level = %q
max_size_mb = %d
max_backups = %d
max_age_days = %d
`, c.BaseURL, apiKey, c.Team, c.HTTPTimeout, c.CosignPath,
		c.Verify.Silent, c.Verify.JSON,
		c.Verify.SkipExternal, c.Verify.SkipSignatures, c.Verify.Remote, c.Verify.Keyring,
		c.Hash.Algorithm, c.Hash.Encoding, c.Hash.Style,
		c.Convert.TimeZone,
		c.Logging.File, c.Logging.Level,
		c.Logging.MaxSizeMB, c.Logging.MaxBackups, c.Logging.MaxAgeDays)
}
