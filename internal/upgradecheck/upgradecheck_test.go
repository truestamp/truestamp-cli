// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package upgradecheck

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/selfupgrade"
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

func TestCachePath_HomeFallback(t *testing.T) {
	// Clear XDG so the HOME branch executes.
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", t.TempDir())
	p, err := CachePath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, filepath.Join(".cache", "truestamp", "upgrade-check.json")) {
		t.Errorf("expected ~/.cache/truestamp/upgrade-check.json suffix, got %q", p)
	}
}

func TestCacheDir_NoHome(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")
	if _, err := cacheDir(); err == nil {
		t.Skip("platform provided a home dir fallback; no-home path not exercised")
	}
}

func TestWriteCache_BadDir(t *testing.T) {
	// Force CachePath to succeed into a directory we then make read-only
	// so the atomic rename fails.
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	// Create the cache dir then make it read-only so CreateTemp fails.
	cacheRoot := filepath.Join(dir, "truestamp")
	if err := os.MkdirAll(cacheRoot, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(cacheRoot, 0700) })

	// Now try to write; the CreateTemp call inside WriteCache should fail.
	err := WriteCache(Cache{LastChecked: time.Now()})
	if err == nil {
		t.Skip("OS allowed write to 0500 directory; skipping this branch")
	}
}

func TestMaybeNotify_FastCacheHit(t *testing.T) {
	// Pre-seed the cache with a fresh result so MaybeNotify takes the
	// cache-hit path (no network). Override the TTY check so Disabled()
	// returns false and emitIfNewer actually runs.
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)
	t.Setenv("TRUESTAMP_NO_UPGRADE_CHECK", "")
	for _, v := range ciEnvVars {
		t.Setenv(v, "")
	}
	orig := terminalCheck
	terminalCheck = func(*os.File) bool { return true }
	t.Cleanup(func() { terminalCheck = orig })

	c := Cache{
		LastChecked:   time.Now().Add(-1 * time.Hour),
		LatestVersion: "v99.0.0",
	}
	if err := WriteCache(c); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	MaybeNotify(&buf, false, "v0.3.0")
	if !strings.Contains(buf.String(), "99.0.0") {
		t.Errorf("expected version in notice, got %q", buf.String())
	}
}

func TestMaybeNotify_StaleCacheRefreshes(t *testing.T) {
	// Stale cache → MaybeNotify refreshes via selfupgrade.FetchLatest.
	// We redirect the GitHub URL to an httptest server returning a
	// newer version so the refresh path is exercised end-to-end.
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)
	t.Setenv("TRUESTAMP_NO_UPGRADE_CHECK", "")
	for _, v := range ciEnvVars {
		t.Setenv(v, "")
	}

	orig := terminalCheck
	terminalCheck = func(*os.File) bool { return true }
	t.Cleanup(func() { terminalCheck = orig })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0","prerelease":false}`))
	}))
	defer srv.Close()

	origURL := selfupgrade.LatestReleaseURL
	selfupgrade.LatestReleaseURL = srv.URL
	t.Cleanup(func() { selfupgrade.LatestReleaseURL = origURL })

	// Seed a stale cache so we force the refresh path (not the cache hit).
	stale := Cache{
		LastChecked:   time.Now().Add(-48 * time.Hour),
		LatestVersion: "v0.1.0",
	}
	_ = WriteCache(stale)

	var buf bytes.Buffer
	MaybeNotify(&buf, false, "v0.3.0")
	// The refresh is async with a 500ms wait budget; it may or may not
	// emit depending on timing. Either outcome exercises the branch.
	time.Sleep(200 * time.Millisecond)
	_ = buf.String()
}

func TestMaybeNotify_Disabled(t *testing.T) {
	// Explicit opt-out returns immediately without touching the cache or
	// network. Exercises the Disabled() early-return branch inside
	// MaybeNotify.
	var buf bytes.Buffer
	MaybeNotify(&buf, true, "v0.3.0")
	if buf.Len() != 0 {
		t.Errorf("MaybeNotify(disabled=true) should emit nothing, got %q", buf.String())
	}
}

func TestIsPreReleaseTagName(t *testing.T) {
	cases := map[string]bool{
		"v1.0.0":      false,
		"1.0.0":       false,
		"v1.0.0-rc.1": true,
		"v1.0.0-beta": true,
		"not-a-tag":   false, // bad semver → not treated as pre-release
	}
	for tag, want := range cases {
		if got := isPreReleaseTagName(tag); got != want {
			t.Errorf("isPreReleaseTagName(%q) = %v, want %v", tag, got, want)
		}
	}
}

func TestIsTerminal_Nil(t *testing.T) {
	if isTerminal(nil) {
		t.Error("nil file should not be a terminal")
	}
}

func TestEmitIfNewer_BadSemver(t *testing.T) {
	var buf bytes.Buffer
	// Either side being unparseable → early return, no emit.
	emitIfNewer(&buf, "not-a-version", "v1.0.0")
	if buf.Len() != 0 {
		t.Errorf("bad current semver should suppress: %q", buf.String())
	}
	buf.Reset()
	emitIfNewer(&buf, "v1.0.0", "not-a-version")
	if buf.Len() != 0 {
		t.Errorf("bad latest semver should suppress: %q", buf.String())
	}
	buf.Reset()
	// Empty latest → early return.
	emitIfNewer(&buf, "v1.0.0", "")
	if buf.Len() != 0 {
		t.Errorf("empty latest should suppress: %q", buf.String())
	}
}
