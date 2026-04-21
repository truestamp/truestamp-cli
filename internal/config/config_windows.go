// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

//go:build windows

package config

import (
	"os"
	"path/filepath"
)

// configDir returns the Windows config directory. Honours %APPDATA%
// when set, otherwise falls back to %USERPROFILE%\AppData\Roaming.
func configDir() string {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "truestamp")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "AppData", "Roaming", "truestamp")
}
