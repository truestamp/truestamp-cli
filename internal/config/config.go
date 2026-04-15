// Copyright (c) 2021-2026 Truestamp, Inc.
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
	"runtime"
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
	APIURL      string       `koanf:"api_url"`
	APIKey      string       `koanf:"api_key"`
	Team        string       `koanf:"team"`
	KeyringURL  string       `koanf:"keyring_url"`
	HTTPTimeout string       `koanf:"http_timeout"`
	Verify      VerifyConfig `koanf:"verify"`
}

// Timeout parses the HTTPTimeout string as a Go duration.
func (c Config) Timeout() time.Duration {
	d, err := time.ParseDuration(c.HTTPTimeout)
	if err != nil {
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

// sectionPrefixes lists known TOML section names for env var mapping.
var sectionPrefixes = []string{"verify"}

// flagKeyMap maps CLI flag names to their koanf key paths.
// Verify-subcommand flags are scoped under "verify.".
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
		"verify.silent":          false,
		"verify.json":            false,
		"verify.skip_external":   false,
		"verify.skip_signatures": false,
		"verify.remote":          false,
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

	// Validate http_timeout is parseable
	if _, err := time.ParseDuration(cfg.HTTPTimeout); err != nil {
		return nil, fmt.Errorf("invalid http_timeout %q: %w", cfg.HTTPTimeout, err)
	}

	return &cfg, nil
}

// ConfigDir returns the config directory path, respecting XDG_CONFIG_HOME on Unix
// and APPDATA on Windows.
func ConfigDir() string {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "truestamp")
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, "AppData", "Roaming", "truestamp")
	}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "truestamp")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "truestamp")
}

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

[verify]
silent = %v
json = %v
skip_external = %v
skip_signatures = %v
remote = %v
`, c.APIURL, apiKey, c.Team, c.KeyringURL, c.HTTPTimeout,
		c.Verify.Silent, c.Verify.JSON,
		c.Verify.SkipExternal, c.Verify.SkipSignatures, c.Verify.Remote)
}
