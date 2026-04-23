// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package upgradecheck

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzReadCache exercises the JSON cache parser. A corrupted cache
// file must be treated as absent — never panic, never return an
// error the caller doesn't expect.
func FuzzReadCache(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("not json"))
	f.Add([]byte(`{"last_checked":"2026-04-21T00:00:00Z","latest_version":"v1.0.0"}`))
	f.Add([]byte(`{`))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", dir)

		cacheRoot := filepath.Join(dir, "truestamp")
		if err := os.MkdirAll(cacheRoot, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cacheRoot, "upgrade-check.json"), data, 0644); err != nil {
			t.Fatal(err)
		}

		// ReadCache by contract never returns an error on missing or
		// corrupt content; it returns (nil, nil) in both cases. Panics
		// would be an immediate fuzz failure.
		_, err := ReadCache()
		if err != nil {
			t.Errorf("ReadCache returned non-nil error on corrupt input: %v", err)
		}
	})
}

// FuzzIsPreReleaseTagName: tag-name classifier used when deciding
// whether to notify.
func FuzzIsPreReleaseTagName(f *testing.F) {
	for _, s := range []string{"", "v1.0.0", "v1.0.0-rc.1", "not-semver"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = isPreReleaseTagName(s)
	})
}
