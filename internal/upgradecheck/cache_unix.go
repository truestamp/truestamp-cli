// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

//go:build !windows

package upgradecheck

import (
	"os"
	"path/filepath"
)

// cacheDir returns the XDG cache directory on POSIX systems. Honours
// XDG_CACHE_HOME when set, otherwise falls back to ~/.cache.
func cacheDir() (string, error) {
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return filepath.Join(d, "truestamp"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "truestamp"), nil
}
