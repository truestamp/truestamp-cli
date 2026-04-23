// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

//go:build windows

package upgradecheck

import (
	"os"
	"path/filepath"
)

// cacheDir returns the Windows cache directory. Prefers %LOCALAPPDATA%,
// falls back to XDG_CACHE_HOME, then ~/.cache for parity with POSIX.
func cacheDir() (string, error) {
	if d := os.Getenv("LOCALAPPDATA"); d != "" {
		return filepath.Join(d, "truestamp"), nil
	}
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return filepath.Join(d, "truestamp"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "truestamp"), nil
}
