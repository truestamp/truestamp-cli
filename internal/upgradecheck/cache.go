// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package upgradecheck handles passive "new version available" notices
// emitted after unrelated commands succeed. Results are cached to disk
// with a 24-hour TTL so normal use never makes more than one GitHub API
// call per day per user.
package upgradecheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Cache holds the most recent upgrade check result. Time is always
// stored in UTC.
type Cache struct {
	LastChecked        time.Time `json:"last_checked"`
	LatestVersion      string    `json:"latest_version"`
	CheckedFromVersion string    `json:"checked_from_version"`
}

// DefaultTTL is how long a cache file is considered fresh.
const DefaultTTL = 24 * time.Hour

// CachePath returns the path to the cache file, creating the containing
// directory if needed. Follows XDG on Linux/macOS and %LOCALAPPDATA% on
// Windows.
func CachePath() (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "upgrade-check.json"), nil
}

// cacheDir is defined per-platform (cache_unix.go, cache_windows.go) so
// each branch is counted for coverage only on its own platform.

// ReadCache returns the cached result, or (nil, nil) if the cache is
// absent or unreadable. Never returns an error — cache misses are a
// normal part of operation.
func ReadCache() (*Cache, error) {
	path, err := CachePath()
	if err != nil {
		return nil, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var c Cache
	if err := json.Unmarshal(body, &c); err != nil {
		// Corrupt cache — pretend it's missing; next successful check
		// will overwrite it.
		return nil, nil
	}
	return &c, nil
}

// WriteCache atomically overwrites the cache file. Errors are returned
// to the caller but are typically non-fatal (the notification path
// ignores them).
func WriteCache(c Cache) error {
	path, err := CachePath()
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "upgrade-check-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// IsFresh returns true if c.LastChecked is within TTL of now.
func (c *Cache) IsFresh(ttl time.Duration) bool {
	if c == nil {
		return false
	}
	return time.Since(c.LastChecked) < ttl
}
