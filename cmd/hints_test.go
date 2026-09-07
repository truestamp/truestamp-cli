// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// withTerminalStdout forces the TTY gate open for the duration of a test.
// `go test` is never a terminal, so without this every hint assertion
// would run against an empty buffer and would keep passing after the text
// rotted. Not parallel-safe: it writes a package-level seam.
func withTerminalStdout(t *testing.T) {
	t.Helper()
	prev := stdoutIsTerminal
	stdoutIsTerminal = func() bool { return true }
	t.Cleanup(func() { stdoutIsTerminal = prev })
}

// Trailing guidance shares one indent, one style, one stream, and is
// separated from the content above it by a single blank line.
//
// Before hintBlock the cursor lines inset four spaces and the tips two,
// so a listing's footer stepped in and out under the table; and every
// line rendered through FaintStyle, which pairs a 2.30:1 foreground on a
// light terminal with the ANSI faint attribute.
func TestHintBlock_AlignmentAndSpacing(t *testing.T) {
	withTerminalStdout(t)

	cursor := "g2wAAAABbQAAACQwMWEwNzA1OC1jODUy"
	cases := []struct {
		name  string
		pg    listPage
		tip   string
		want  []string
		empty bool
	}{
		{
			name: "next only",
			pg:   listPage{Next: cursor},
			want: []string{hintIndent + "More: --after " + cursor},
		},
		{
			name: "next and prev",
			pg:   listPage{Next: cursor, Prev: "abc"},
			want: []string{hintIndent + "More: --after " + cursor, hintIndent + "Back: --before abc"},
		},
		{
			name: "clamp rides along only with a next page",
			pg:   listPage{Next: cursor, ClampedTo: 250},
			want: []string{hintIndent + "(the server caps a page at 250 rows"},
		},
		{
			name: "tip is appended after the cursors",
			pg:   listPage{Next: cursor},
			tip:  "Hint: do a thing.",
			want: []string{hintIndent + "More: --after " + cursor, hintIndent + "Hint: do a thing."},
		},
		{name: "clamp alone is not printed", pg: listPage{ClampedTo: 250}, empty: true},
		{name: "nothing to say leaves no stray gap", pg: listPage{}, empty: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var errw bytes.Buffer
			renderListHints(&errw, tc.pg, tc.tip)
			got := errw.String()

			if tc.empty {
				if got != "" {
					t.Fatalf("nothing to print must write nothing, got:\n%q", got)
				}
				return
			}
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("want %q in:\n%q", w, got)
				}
			}
			if !strings.HasPrefix(got, "\n") {
				t.Errorf("block must open with a blank separator, got:\n%q", got)
			}
			for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
				plain := stripANSI(line)
				if strings.TrimSpace(plain) == "" {
					continue
				}
				if n := len(plain) - len(strings.TrimLeft(plain, " ")); n != len(hintIndent) {
					t.Errorf("line %q indents %d, want %d", plain, n, len(hintIndent))
				}
			}
		})
	}
}

// Guidance is commentary about the output, not the output. A pipeline must
// never receive it, so every line is gated on stdout being a terminal --
// the cursor lines included, which used to go to stdout ungated.
func TestHintBlock_SuppressedWhenStdoutIsNotATerminal(t *testing.T) {
	prev := stdoutIsTerminal
	stdoutIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdoutIsTerminal = prev })

	var errw bytes.Buffer
	renderListHints(&errw, listPage{Next: "CUR", Prev: "P", ClampedTo: 250}, "Hint: do a thing.")
	if errw.Len() != 0 {
		t.Errorf("piped stdout must suppress the whole block, got:\n%q", errw.String())
	}
}

// The block never touches stdout, so redirecting a listing to a file
// yields only rows.
func TestHintBlock_NeverWritesToStdout(t *testing.T) {
	withTerminalStdout(t)
	var out, errw bytes.Buffer
	renderListHints(&errw, listPage{Next: "CUR"}, "Hint: do a thing.")
	if out.Len() != 0 {
		t.Errorf("stdout must stay clean, got:\n%q", out.String())
	}
	if !strings.Contains(errw.String(), "More: --after CUR") {
		t.Errorf("guidance belongs on stderr, got:\n%q", errw.String())
	}
}
