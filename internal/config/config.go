// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package config resolves the CLI's runtime configuration from compiled
// defaults, an optional TOML config file, environment variables, and
// command-line flags — in that order, with later sources overriding
// earlier ones.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
type Config struct {
	APIURL      string `koanf:"api_url"`
	APIKey      string `koanf:"api_key"`
	Team        string `koanf:"team"`
	KeyringURL  string `koanf:"keyring_url"`
	HTTPTimeout string `koanf:"http_timeout"`
	// CosignPath pins the cosign binary used during `truestamp upgrade`.
	// Must be an absolute path to an executable when set; empty means
	// fall back to $PATH lookup. Settable in config.toml as
	// `cosign_path = "..."` or via the TRUESTAMP_COSIGN_PATH env var.
	CosignPath string        `koanf:"cosign_path"`
	Verify     VerifyConfig  `koanf:"verify"`
	Hash       HashConfig    `koanf:"hash"`
	Convert    ConvertConfig `koanf:"convert"`
}

// Timeout parses the HTTPTimeout string as a Go duration. A zero or
// negative parsed value (which http.Client treats as "no timeout") is
// rejected in favour of the 10-second default — Load validates this up
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
var sectionPrefixes = []string{"verify", "hash", "convert"}

// flagKeyMap maps CLI flag names to their koanf key paths.
// Verify-subcommand flags are scoped under "verify."; hash under "hash.";
// convert under "convert.".
var flagKeyMap = map[string]string{
	// Root persistent flags
	"api-url":      "api_url",
	"api-key":      "api_key",
	"team":         "team",
	"keyring-url":  "keyring_url",
	"http-timeout": "http_timeout",
	// Verify subcommand flags
	"silent":          "verify.silent",
	"json":            "verify.json",
	"skip-external":   "verify.skip_external",
	"skip-signatures": "verify.skip_signatures",
	"remote":          "verify.remote",
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
		"api_url":                "https://www.truestamp.com/api/json",
		"api_key":                "",
		"team":                   "",
		"keyring_url":            "https://www.truestamp.com/.well-known/keyring.json",
		"http_timeout":           "10s",
		"cosign_path":            "",
		"verify.silent":          false,
		"verify.json":            false,
		"verify.skip_external":   false,
		"verify.skip_signatures": false,
		"verify.remote":          false,
		"hash.algorithm":         "sha256",
		"hash.encoding":          "hex",
		"hash.style":             "gnu",
		"convert.time_zone":      "UTC",
	}
	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// 2. Config file (skip if missing)
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

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Validate http_timeout is parseable and positive. A zero or negative
	// duration is treated by http.Client as "no timeout", which would
	// silently disable the guard — reject it up front so the user gets
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
	// time in selfupgrade.resolveCosignBinary — the upgrade command is
	// the only consumer, and the filesystem state there can differ from
	// startup (cosign installed/removed between a session's commands).
	if cfg.CosignPath != "" && !filepath.IsAbs(cfg.CosignPath) {
		return nil, fmt.Errorf("invalid cosign_path %q: must be an absolute path (or empty for $PATH lookup)", cfg.CosignPath)
	}

	return &cfg, nil
}

// ConfigDir returns the config directory path. The implementation is
// split across config_unix.go and config_windows.go via build tags so
// each platform's branch is exercised — and scored for coverage — only
// on the platform where it can actually run.
func ConfigDir() string { return configDir() }

// ConfigFilePath returns the full path to the config file.
func ConfigFilePath() string {
	return filepath.Join(ConfigDir(), "config.toml")
}

// EnsureDefaultConfig creates the config directory and writes the default
// config.toml if it does not already exist. Returns true if the file was created.
func EnsureDefaultConfig() (bool, error) {
	dir := ConfigDir()
	if dir == "" {
		return false, fmt.Errorf("cannot determine home directory; set $HOME or $XDG_CONFIG_HOME")
	}

	path := filepath.Join(dir, "config.toml")

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

	return fmt.Sprintf(`api_url = %q
api_key = %q
team = %q
keyring_url = %q
http_timeout = %q
cosign_path = %q

[verify]
silent = %v
json = %v
skip_external = %v
skip_signatures = %v
remote = %v

[hash]
algorithm = %q
encoding = %q
style = %q

[convert]
time_zone = %q
`, c.APIURL, apiKey, c.Team, c.KeyringURL, c.HTTPTimeout, c.CosignPath,
		c.Verify.Silent, c.Verify.JSON,
		c.Verify.SkipExternal, c.Verify.SkipSignatures, c.Verify.Remote,
		c.Hash.Algorithm, c.Hash.Encoding, c.Hash.Style,
		c.Convert.TimeZone)
}
