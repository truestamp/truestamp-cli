// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceTopLevelAPIKey_Replace(t *testing.T) {
	in := []byte(`# comment
api_url = "https://example.com/api/json"
api_key = ""
team = ""

[verify]
silent = false
api_key = "should-not-touch-section-scoped"
`)
	out, err := replaceTopLevelAPIKey(in, "ts_live_abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `api_key = "ts_live_abc123"`) {
		t.Errorf("expected replaced api_key line, got:\n%s", s)
	}
	if !strings.Contains(s, `api_key = "should-not-touch-section-scoped"`) {
		t.Errorf("must not rewrite api_key inside [verify] section, got:\n%s", s)
	}
	if strings.Count(s, "ts_live_abc123") != 1 {
		t.Errorf("expected exactly one replacement, got:\n%s", s)
	}
	if !strings.Contains(s, "# comment") {
		t.Errorf("must preserve comments, got:\n%s", s)
	}
}

func TestReplaceTopLevelAPIKey_InsertBeforeSection(t *testing.T) {
	in := []byte(`api_url = "https://example.com/api/json"

[verify]
silent = false
`)
	out, err := replaceTopLevelAPIKey(in, "ts_new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `api_key = "ts_new"`) {
		t.Errorf("expected inserted api_key line, got:\n%s", s)
	}
	keyIdx := strings.Index(s, `api_key`)
	sectionIdx := strings.Index(s, `[verify]`)
	if keyIdx < 0 || sectionIdx < 0 || keyIdx > sectionIdx {
		t.Errorf("expected api_key to be inserted before [verify], got:\n%s", s)
	}
}

func TestReplaceTopLevelAPIKey_NoSectionsAppend(t *testing.T) {
	in := []byte(`api_url = "https://example.com/api/json"
team = ""
`)
	out, err := replaceTopLevelAPIKey(in, "k")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `api_key = "k"`) {
		t.Errorf("expected appended api_key line, got:\n%s", s)
	}
}

func TestReplaceTopLevelAPIKey_ClearEmpty(t *testing.T) {
	in := []byte(`api_key = "old"
`)
	out, err := replaceTopLevelAPIKey(in, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `api_key = ""`) {
		t.Errorf("expected cleared api_key line, got:\n%s", s)
	}
	if strings.Contains(s, `"old"`) {
		t.Errorf("old value must be removed, got:\n%s", s)
	}
}

func TestTOMLQuote(t *testing.T) {
	cases := map[string]string{
		``:          `""`,
		`abc`:       `"abc"`,
		`a"b`:       `"a\"b"`,
		`a\b`:       `"a\\b"`,
		"line\nend": `"line\nend"`,
		"tab\there": `"tab\there"`,
		"cr\rend":   `"cr\rend"`,
	}
	for in, want := range cases {
		if got := tomlQuote(in); got != want {
			t.Errorf("tomlQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSetAPIKey_NoHome covers the branch where EnsureDefaultConfig
// returns an error (no determinable config directory).
func TestSetAPIKey_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", "")
	if err := SetAPIKey("x"); err == nil {
		t.Skip("platform provided a fallback home; no-home path not exercised here")
	}
}

// TestSetAPIKey_ReadError exercises the ReadFile failure path. We
// create the config dir but make it unreadable.
func TestSetAPIKey_ReadError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	// Create the file, then make it unreadable.
	cfgDir := filepath.Join(dir, "truestamp")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(path, []byte(""), 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })

	if err := SetAPIKey("x"); err == nil {
		t.Skip("filesystem allowed unreadable file to be read (unusual)")
	}
}

func TestSetAPIKey_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	if err := SetAPIKey("ts_integration_key"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}

	path := ConfigFilePath()
	if !strings.HasPrefix(path, dir) {
		t.Fatalf("expected config path under tempdir %q, got %q", dir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), `api_key = "ts_integration_key"`) {
		t.Errorf("written config missing api_key, got:\n%s", data)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("expected 0600 permissions on config with secret, got %o", mode)
	}

	// Clearing leaves the file intact with an empty key.
	if err := SetAPIKey(""); err != nil {
		t.Fatalf("SetAPIKey empty: %v", err)
	}
	data2, _ := os.ReadFile(path)
	if !strings.Contains(string(data2), `api_key = ""`) {
		t.Errorf("cleared config missing empty api_key, got:\n%s", data2)
	}

	// Sanity check that the path we wrote to is reachable via filepath.
	if _, err := os.Stat(filepath.Join(dir, "truestamp", "config.toml")); err != nil {
		t.Errorf("unexpected missing config file: %v", err)
	}
}
