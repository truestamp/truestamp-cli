// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Documentation is part of the interface. A renamed command or a retired
// flag that still appears in README.md, EXAMPLES.md, CLAUDE.md or kb/ is
// a defect the same way a broken link is: the reader types what we told
// them to and it fails.
//
// These tests parse every `truestamp ...` invocation out of the docs and
// check it against the live cobra tree, so the command-tree reorganization
// cannot land with stale docs behind it.

// docFiles are every Markdown file whose examples are meant to be
// runnable. CHANGELOG.md is deliberately excluded: it is history, and its
// entries describe commands as they were at the time.
func docFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var out []string
	for _, p := range []string{"README.md", "EXAMPLES.md", "CLAUDE.md"} {
		out = append(out, filepath.Join(root, p))
	}
	err := filepath.Walk(filepath.Join(root, "kb"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking kb/: %v", err)
	}
	return out
}

// repoRoot walks up from the test's working directory (cmd/) to the
// module root, so the test does not depend on how it was invoked.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for range 5 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate repo root (no go.mod found walking up)")
	return ""
}

// docInvocation is one `truestamp ...` line lifted from the docs.
type docInvocation struct {
	file string
	line int
	raw  string
	args []string
}

var (
	// A documented invocation: a line whose first word is `truestamp`,
	// optionally indented, optionally preceded by a `$ ` prompt.
	invocationRe = regexp.MustCompile(`^\s*\$?\s*truestamp\s+(.*)$`)
	// Command substitutions. A nested `$(truestamp hash --style bare f)`
	// inside a `truestamp verify ...` line belongs to the inner command,
	// not the outer one; without this the outer command appears to accept
	// the inner one's flags.
	substitutionRe = regexp.MustCompile(`\$\([^)]*\)`)
	// Long flags only. Short flags are ambiguous with negative numbers
	// and with `-` used as the stdin sentinel.
	longFlagRe = regexp.MustCompile(`^--[a-z0-9][a-z0-9-]*`)
)

// collectInvocations extracts documented commands. It stops each line at
// the first shell metacharacter, because what is being validated is the
// truestamp command itself, not the pipeline it sits in.
func collectInvocations(t *testing.T) []docInvocation {
	t.Helper()
	var found []docInvocation
	for _, f := range docFiles(t) {
		fh, err := os.Open(f)
		if err != nil {
			t.Fatalf("open %s: %v", f, err)
		}
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNo := 0
		for sc.Scan() {
			lineNo++
			line := sc.Text()
			m := invocationRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			rest := m[1]
			// Lift nested command substitutions out first, and record any
			// truestamp invocation inside them as an invocation in its own
			// right, so those flags are checked against the right command.
			for _, sub := range substitutionRe.FindAllString(rest, -1) {
				inner := strings.TrimSuffix(strings.TrimPrefix(sub, "$("), ")")
				if inner2, ok := strings.CutPrefix(strings.TrimSpace(inner), "truestamp "); ok {
					if a := strings.Fields(inner2); len(a) > 0 {
						found = append(found, docInvocation{
							file: f, line: lineNo, raw: strings.TrimSpace(inner), args: a,
						})
					}
				}
			}
			rest = substitutionRe.ReplaceAllString(rest, "SUBST")
			// Cut at a shell metacharacter or a trailing comment.
			for _, cut := range []string{" | ", " > ", " >> ", " && ", " || ", ";", " #", "\t#"} {
				if i := strings.Index(rest, cut); i >= 0 {
					rest = rest[:i]
				}
			}
			args := strings.Fields(rest)
			if len(args) == 0 {
				continue
			}
			found = append(found, docInvocation{
				file: f, line: lineNo, raw: strings.TrimSpace(line), args: args,
			})
		}
		_ = fh.Close()
		if err := sc.Err(); err != nil {
			t.Fatalf("scanning %s: %v", f, err)
		}
	}
	return found
}

// commandPath returns the leading non-flag words, which name the command,
// and stops at the first argument that looks like a flag, a path, a URL,
// or a placeholder. Cobra resolves `truestamp convert time` to the `time`
// subcommand; anything after that is an argument, not a command.
func commandPath(args []string) []string {
	var path []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			break
		}
		// Placeholders and operands, not command names.
		if strings.ContainsAny(a, "/.<>[]{}$\"'=:") || a == "-" {
			break
		}
		path = append(path, a)
	}
	return path
}

func TestDocs_InvocationsAreExtractable(t *testing.T) {
	inv := collectInvocations(t)
	// A parser that silently matches nothing would make every test below
	// pass while checking nothing at all.
	if len(inv) < 50 {
		t.Fatalf("only extracted %d documented invocations; the parser is probably broken", len(inv))
	}
	t.Logf("checking %d documented invocations", len(inv))
}

// TestDocs_CommandsExist asserts every documented command path resolves
// in the live cobra tree. This is the test that catches a rename landing
// without a docs sweep.
func TestDocs_CommandsExist(t *testing.T) {
	seen := map[string][]docInvocation{}
	for _, in := range collectInvocations(t) {
		p := commandPath(in.args)
		if len(p) == 0 {
			continue
		}
		key := strings.Join(p, " ")
		seen[key] = append(seen[key], in)
	}
	if len(seen) == 0 {
		t.Fatal("no command paths extracted")
	}
	for key, uses := range seen {
		t.Run(key, func(t *testing.T) {
			args := append(strings.Fields(key), "--help")
			out, err := exec.Command(binaryPath, args...).CombinedOutput()
			if err != nil {
				t.Errorf("documented command %q does not exist (%v)\nfirst use: %s:%d\n  %s\nCLI said: %s",
					key, err, uses[0].file, uses[0].line, uses[0].raw, firstLine(string(out)))
			}
		})
	}
}

// TestDocs_FlagsExist asserts every long flag in a documented invocation
// is accepted by the command it is used with. Inherited persistent flags
// count, because `--help` lists them under Global Flags.
func TestDocs_FlagsExist(t *testing.T) {
	helpCache := map[string]string{}
	for _, in := range collectInvocations(t) {
		p := commandPath(in.args)
		if len(p) == 0 {
			continue
		}
		key := strings.Join(p, " ")
		help, ok := helpCache[key]
		if !ok {
			out, err := exec.Command(binaryPath, append(strings.Fields(key), "--help")...).CombinedOutput()
			if err != nil {
				// Reported by TestDocs_CommandsExist; don't double-report.
				helpCache[key] = ""
				continue
			}
			help = string(out)
			helpCache[key] = help
		}
		if help == "" {
			continue
		}
		for _, a := range in.args {
			flag := longFlagRe.FindString(a)
			if flag == "" {
				continue
			}
			if !strings.Contains(help, flag) {
				t.Errorf("%s:%d documents %s for %q, but the command does not accept it\n  %s",
					in.file, in.line, flag, key, in.raw)
			}
		}
	}
}

// TestDocs_NoRetiredNames is the belt to TestDocs_CommandsExist's braces.
// A retired name can survive in prose, a table, or a config snippet where
// no `truestamp ` prefix makes it an invocation, and those are exactly the
// places a reader copies from. CHANGELOG.md is exempt as history.
func TestDocs_NoRetiredNames(t *testing.T) {
	// Each entry is a string that must not appear, and why.
	retired := map[string]string{
		"TRUESTAMP_VERIFY_SILENT":        "--silent is CLI-wide; the env var is TRUESTAMP_SILENT",
		"TRUESTAMP_VERIFY_JSON":          "--json is CLI-wide; the env var is TRUESTAMP_JSON",
		"TRUESTAMP_VERIFY_SKIP_EXTERNAL": "renamed to TRUESTAMP_VERIFY_OFFLINE",
		// Only the config-key spelling. A bare `skip_external` is also the
		// server's own wire field on POST /proof/verify, which is correct
		// and must keep its name.
		"verify.skip_external": "the config key is verify.offline",
		"--skip-external":      "removed; the flag is --offline",
	}
	for _, f := range docFiles(t) {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		text := string(body)
		for needle, why := range retired {
			if !strings.Contains(text, needle) {
				continue
			}
			// kb/command-tree.md records the reorganization itself and is
			// allowed to name what was retired.
			if strings.HasSuffix(f, "command-tree.md") {
				continue
			}
			for i, line := range strings.Split(text, "\n") {
				if strings.Contains(line, needle) {
					t.Errorf("%s:%d still mentions %q (%s)\n  %s",
						f, i+1, needle, why, strings.TrimSpace(line))
				}
			}
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
