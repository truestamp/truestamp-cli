// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoad_TOML feeds arbitrary bytes into the koanf TOML loader via
// a real file on disk — the same path the CLI uses at startup. The
// config file is user-editable, so it's an untrusted-input surface.
func FuzzLoad_TOML(f *testing.F) {
	f.Add([]byte(``))
	f.Add([]byte(`api_url = "https://example.com"`))
	f.Add([]byte("[verify]\nsilent = true\n"))
	f.Add([]byte(`malformed = ][`))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
		_, _ = Load(path, nil)
	})
}

// FuzzReplaceTopLevelAPIKey exercises the in-place TOML rewriter with
// arbitrary existing content + arbitrary keys. The rewriter runs
// under SetAPIKey when the user calls `truestamp auth login`, so any
// panic here corrupts the user's config file.
func FuzzReplaceTopLevelAPIKey(f *testing.F) {
	f.Add([]byte(""), "")
	f.Add([]byte(`api_url = "x"
api_key = ""
`), "new-key")
	f.Add([]byte(`[verify]
silent = true
`), "key-with-newline\nin-it")
	f.Add([]byte(`malformed`), "k")

	f.Fuzz(func(t *testing.T, contents []byte, key string) {
		_, _ = replaceTopLevelAPIKey(contents, key)
	})
}

// FuzzTomlQuote: the string-escape helper. Fuzz with arbitrary runes
// to ensure the quote/escape logic never panics on weird input.
func FuzzTomlQuote(f *testing.F) {
	for _, s := range []string{"", "abc", "a\"b", "a\\b", "line\nend", "tab\there"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = tomlQuote(s)
	})
}
