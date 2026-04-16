// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package upgradecheck

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCacheRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	now := time.Now().UTC().Truncate(time.Second)
	want := Cache{
		LastChecked:        now,
		LatestVersion:      "v0.4.0",
		CheckedFromVersion: "v0.3.0",
	}
	if err := WriteCache(want); err != nil {
		t.Fatalf("WriteCache: %v", err)
	}

	got, err := ReadCache()
	if err != nil {
		t.Fatalf("ReadCache: %v", err)
	}
	if got == nil {
		t.Fatal("ReadCache returned nil")
	}
	if !got.LastChecked.Equal(want.LastChecked) ||
		got.LatestVersion != want.LatestVersion ||
		got.CheckedFromVersion != want.CheckedFromVersion {
		t.Errorf("round-trip mismatch: got %+v, want %+v", *got, want)
	}
}

func TestCache_MissingOrCorrupt(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	// Missing cache file: expect (nil, nil).
	got, err := ReadCache()
	if err != nil || got != nil {
		t.Errorf("missing cache: got %v, %v, want nil, nil", got, err)
	}

	// Corrupt file: expect (nil, nil) — caller should treat as absent.
	path, err := CachePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not valid json"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err = ReadCache()
	if err != nil || got != nil {
		t.Errorf("corrupt cache: got %v, %v, want nil, nil", got, err)
	}
}

func TestCache_IsFresh(t *testing.T) {
	fresh := &Cache{LastChecked: time.Now().Add(-1 * time.Hour)}
	stale := &Cache{LastChecked: time.Now().Add(-48 * time.Hour)}
	if !fresh.IsFresh(DefaultTTL) {
		t.Error("1h-old cache should be fresh at 24h TTL")
	}
	if stale.IsFresh(DefaultTTL) {
		t.Error("48h-old cache should not be fresh at 24h TTL")
	}

	var nilCache *Cache
	if nilCache.IsFresh(DefaultTTL) {
		t.Error("nil cache should not be fresh")
	}
}

func TestDisabled_flagOptOut(t *testing.T) {
	// Ensure env is clean.
	for _, v := range ciEnvVars {
		t.Setenv(v, "")
	}
	t.Setenv("TRUESTAMP_NO_UPGRADE_CHECK", "")

	// --no-upgrade-check flag short-circuits to disabled.
	if !Disabled(true, nil, "v0.3.0") {
		t.Error("Disabled should be true when flag is set")
	}
}

func TestDisabled_envOptOut(t *testing.T) {
	for _, v := range ciEnvVars {
		t.Setenv(v, "")
	}
	t.Setenv("TRUESTAMP_NO_UPGRADE_CHECK", "1")
	if !Disabled(false, nil, "v0.3.0") {
		t.Error("Disabled should be true when TRUESTAMP_NO_UPGRADE_CHECK set")
	}
}

func TestDisabled_devVersion(t *testing.T) {
	for _, v := range ciEnvVars {
		t.Setenv(v, "")
	}
	t.Setenv("TRUESTAMP_NO_UPGRADE_CHECK", "")
	if !Disabled(false, nil, "dev") {
		t.Error("Disabled should be true when current version is 'dev'")
	}
	if !Disabled(false, nil, "") {
		t.Error("Disabled should be true when current version is empty")
	}
}

func TestDisabled_ciEnv(t *testing.T) {
	// Clear all CI vars first.
	for _, v := range ciEnvVars {
		t.Setenv(v, "")
	}
	t.Setenv("TRUESTAMP_NO_UPGRADE_CHECK", "")

	for _, v := range ciEnvVars {
		t.Run(v, func(t *testing.T) {
			for _, clear := range ciEnvVars {
				t.Setenv(clear, "")
			}
			t.Setenv(v, "1")
			if !Disabled(false, nil, "v0.3.0") {
				t.Errorf("Disabled should be true when %s=1", v)
			}
		})
	}
}

func TestDisabled_nonTTY(t *testing.T) {
	for _, v := range ciEnvVars {
		t.Setenv(v, "")
	}
	t.Setenv("TRUESTAMP_NO_UPGRADE_CHECK", "")
	// A temp file is not a character device; isTerminal should return
	// false, which makes Disabled return true.
	f, err := os.CreateTemp(t.TempDir(), "stdout-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if !Disabled(false, f, "v0.3.0") {
		t.Error("Disabled should be true when stderr is a regular file (non-TTY)")
	}
}

func TestEmitIfNewer(t *testing.T) {
	var buf bytes.Buffer
	// Older current → newer latest: should emit. Display strips the
	// leading "v" so output format is consistent regardless of whether
	// the source (ldflags vs GitHub tag) carried the prefix.
	emitIfNewer(&buf, "v0.3.0", "v0.4.0")
	if !strings.Contains(buf.String(), "0.4.0 is available") {
		t.Errorf("emitIfNewer (upgrade case) didn't include version: %q", buf.String())
	}
	if strings.Contains(buf.String(), "v0.4.0 is available") {
		t.Errorf("emitIfNewer leaked the 'v' prefix into the notice: %q", buf.String())
	}
	if strings.Contains(buf.String(), "(current: v0.3.0)") {
		t.Errorf("emitIfNewer leaked the 'v' prefix into the current-version field: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "TRUESTAMP_NO_UPGRADE_CHECK") {
		t.Errorf("emitIfNewer didn't mention opt-out env var")
	}

	buf.Reset()
	// Same or older: should not emit.
	emitIfNewer(&buf, "v0.4.0", "v0.4.0")
	if buf.Len() != 0 {
		t.Errorf("emitIfNewer (same version): wrote %q, expected nothing", buf.String())
	}

	buf.Reset()
	emitIfNewer(&buf, "v0.5.0", "v0.4.0")
	if buf.Len() != 0 {
		t.Errorf("emitIfNewer (current > latest): wrote %q, expected nothing", buf.String())
	}

	buf.Reset()
	// Pre-release latest: never emit.
	emitIfNewer(&buf, "v0.3.0", "v1.0.0-rc.1")
	if buf.Len() != 0 {
		t.Errorf("emitIfNewer (pre-release latest): wrote %q, expected nothing", buf.String())
	}
}

func TestCachePath_usesXDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", xdg)

	path, err := CachePath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(xdg, "truestamp", "upgrade-check.json")
	if path != want {
		t.Errorf("CachePath = %q, want %q", path, want)
	}
}
