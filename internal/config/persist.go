// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// keyLineRe builds a regex matching a top-level `<key> = ...` TOML line.
// Used by replaceTopLevelKey to find and rewrite a specific assignment
// outside any [section] header.
func keyLineRe(key string) *regexp.Regexp {
	return regexp.MustCompile(`^(\s*)` + regexp.QuoteMeta(key) + `(\s*)=(\s*).*$`)
}

// SetAPIKey writes key as the top-level api_key value in the config file
// in effect ([ActivePath], the --config override when one was given,
// otherwise the platform default), preserving comments and other
// settings. Creates the file from the embedded default if it does not
// yet exist, and tightens permissions to 0600 because the file now
// contains a secret.
func SetAPIKey(key string) error {
	return setTopLevelString("api_key", key, 0600)
}

// SetTeam writes id as the top-level team value in the config file in
// effect ([ActivePath], the --config override when one was given,
// otherwise the platform default), preserving comments and other
// settings. Creates the file from the embedded default if it does not
// yet exist. Permissions are kept at 0600 to match SetAPIKey, the file
// co-exists with the api_key secret, so a less-tight regime would only
// loosen security on the shared file.
func SetTeam(id string) error {
	return setTopLevelString("team", id, 0600)
}

// setTopLevelString is the shared write path used by SetAPIKey + SetTeam.
// It reads the existing file (creating from the embedded default if
// missing), rewrites a single top-level assignment, and writes the
// result with the given permissions.
//
// The target is [ActivePath], NOT [ConfigFilePath]: writing to the
// platform default while --config pointed somewhere else silently
// discarded the user's change and corrupted the untargeted file.
func setTopLevelString(key, value string, perm os.FileMode) error {
	if _, err := EnsureDefaultConfig(); err != nil {
		return err
	}
	path := ActivePath()

	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	replaced, err := replaceTopLevelKey(contents, key, value)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, replaced, perm); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("securing config: %w", err)
	}
	return nil
}

// replaceTopLevelAPIKey is kept as a thin pass-through for the existing
// test suite. New code should call replaceTopLevelKey directly.
func replaceTopLevelAPIKey(contents []byte, key string) ([]byte, error) {
	return replaceTopLevelKey(contents, "api_key", key)
}

// replaceTopLevelKey finds the `<key>` assignment in the top-level
// TOML section and rewrites it. If no such line exists, a new line is
// inserted just before the first `[section]` header, or appended if the
// file has no sections. Lines inside any `[section]` are never touched
// , `api_key = "..."` inside `[verify]` would be a different setting
// even though it shares a name.
func replaceTopLevelKey(contents []byte, key, value string) ([]byte, error) {
	newLine := fmt.Sprintf("%s = %s", key, tomlQuote(value))
	re := keyLineRe(key)

	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	inTopLevel := true
	replaced := false
	inserted := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if inTopLevel && strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if !replaced {
				out.WriteString(newLine)
				out.WriteByte('\n')
				inserted = true
			}
			inTopLevel = false
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}

		if inTopLevel && !replaced && re.MatchString(line) {
			out.WriteString(newLine)
			out.WriteByte('\n')
			replaced = true
			continue
		}

		out.WriteString(line)
		out.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning config: %w", err)
	}

	if !replaced && !inserted {
		out.WriteString(newLine)
		out.WriteByte('\n')
	}

	result := out.Bytes()
	if !bytes.HasSuffix(contents, []byte("\n")) && len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}

// tomlQuote returns a TOML basic-string literal of s. API keys are ASCII
// in practice; this covers backslash and quote escaping defensively.
func tomlQuote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
