// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

//go:build !windows

package config

import (
	"os"
	"path/filepath"
)

// configDir returns the XDG-style config directory on POSIX systems.
// Honours XDG_CONFIG_HOME when set, otherwise falls back to ~/.config.
func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "truestamp")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "truestamp")
}
