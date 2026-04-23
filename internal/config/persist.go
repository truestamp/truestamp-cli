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

// apiKeyLineRe matches a top-level `api_key = "..."` TOML assignment.
var apiKeyLineRe = regexp.MustCompile(`^(\s*)api_key(\s*)=(\s*).*$`)

// SetAPIKey writes key as the top-level api_key value in the user's config
// file, preserving comments and other settings. Creates the file from the
// embedded default if it does not yet exist, and tightens permissions to
// 0600 because the file now contains a secret.
func SetAPIKey(key string) error {
	if _, err := EnsureDefaultConfig(); err != nil {
		return err
	}
	path := ConfigFilePath()

	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	replaced, err := replaceTopLevelAPIKey(contents, key)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, replaced, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("securing config: %w", err)
	}
	return nil
}

// replaceTopLevelAPIKey finds the `api_key` assignment in the top-level
// TOML section and rewrites it. If no such line exists, a new line is
// inserted just before the first `[section]` header, or appended if the
// file has no sections.
func replaceTopLevelAPIKey(contents []byte, key string) ([]byte, error) {
	newLine := fmt.Sprintf("api_key = %s", tomlQuote(key))

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

		if inTopLevel && !replaced && apiKeyLineRe.MatchString(line) {
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
