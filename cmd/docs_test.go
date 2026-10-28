// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/introspect"
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
	dir, err := findModuleRoot()
	if err != nil {
		t.Fatalf("could not locate repo root: %v", err)
	}
	return dir
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
	// The same thing written inline in prose, inside backticks. These are
	// read and copied as often as the fenced blocks are, and until this
	// existed they were checked by nothing: `truestamp help formatting`
	// survived in two files pointing at a help topic that had been deleted,
	// because it never appeared at the start of a line.
	inlineInvocationRe = regexp.MustCompile("`truestamp\\s+([^`]*)`")
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
			var candidates []string
			if m := invocationRe.FindStringSubmatch(line); m != nil {
				candidates = append(candidates, m[1])
			}
			for _, im := range inlineInvocationRe.FindAllStringSubmatch(line, -1) {
				candidates = append(candidates, im[1])
			}
			if len(candidates) == 0 {
				continue
			}
			for _, rest := range candidates {
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
		}
		_ = fh.Close()
		if err := sc.Err(); err != nil {
			t.Fatalf("scanning %s: %v", f, err)
		}
	}
	return found
}

// commandWords returns the leading words that could name a command: it
// stops at the first argument that looks like a flag, a path, a URL, or a
// placeholder. `|` is in the stop set because prose writes alternations
// like `truestamp convert time|id|keyid`, which name several commands
// rather than one.
func commandWords(args []string) []string {
	var path []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			break
		}
		// Placeholders and operands, not command names.
		if strings.ContainsAny(a, "/.<>[]{}$\"'=:|") || a == "-" {
			break
		}
		path = append(path, a)
	}
	return path
}

// commandPath resolves the longest leading run of words that names a real
// command, the way cobra's own Find does, and reports whether the words it
// could not consume leave an unresolved command behind. A bare operand
// (`truestamp schema get commands`, `truestamp keys get 96b1cd2f`) ends the
// path; a first word that names nothing (`truestamp team list`) is an
// unknown command, not an empty path.
func commandPath(args []string, known map[string]map[string]bool) (path []string, unknown string) {
	words := commandWords(args)
	for _, w := range words {
		next := strings.Join(append(append([]string{}, path...), w), " ")
		if _, ok := known[next]; !ok {
			if len(path) == 0 {
				return nil, w
			}
			break
		}
		path = append(path, w)
	}
	return path, ""
}

// docTree renders the live cobra tree as data once, mapping every command
// path to the long flags it accepts.
//
// Resolving a documented path against this is exact. The previous version
// of these tests shelled out to `<path> --help` and read the exit code,
// which cannot detect a retired SUBCOMMAND at all: cobra answers --help
// before it validates the argument, so `truestamp convert proof --help`
// and `truestamp proofs download --help` both exit 0 long after those
// paths stopped existing. Only an unknown *top-level* word was ever
// caught.
// docTreeAndGroups also reports which paths are groups. A group is a
// namespace, so any leftover plain word after one names a subcommand that does not
// exist: `truestamp convert proof` is not "convert with an operand", it
// names no subcommand, and cobra rejects it at runtime.
// After a runnable leaf the same word is a legitimate operand
// (`truestamp verify proof.json`), which is why this distinction is needed
// rather than a blanket "no leftover words" rule.
func docTreeAndGroups(t *testing.T) (map[string]map[string]bool, map[string]bool) {
	t.Helper()
	known := map[string]map[string]bool{}
	groups := map[string]bool{}
	var walk func(c introspect.Command)
	walk = func(c introspect.Command) {
		path := strings.TrimPrefix(c.Path, "truestamp")
		path = strings.TrimSpace(path)
		flags := map[string]bool{
			// Cobra creates these itself during Execute, so they are not in
			// the walked tree, but every command really does accept them.
			"help": true, "version": true,
		}
		for _, f := range c.Flags {
			flags[f.Name] = true
		}
		known[path] = flags
		if c.Group {
			groups[path] = true
		}
		for _, sub := range c.Subcommands {
			walk(sub)
		}
	}
	walk(introspect.Walk(rootCmd, cliEnums(), false))
	// introspect deliberately omits cobra's own help and completion
	// scaffolding: `schema get commands` describes the Truestamp interface,
	// not the framework's. Both are nonetheless real, invocable and
	// documented, so this test has to know them.
	known["help"] = map[string]bool{"help": true, "version": true}
	known["completion"] = map[string]bool{"help": true, "version": true}
	groups["completion"] = true
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		known["completion "+sh] = map[string]bool{"help": true, "version": true, "no-descriptions": true}
	}
	if len(known) < 20 {
		t.Fatalf("only %d command paths in the tree; introspection is broken", len(known))
	}
	return known, groups
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

// recordsRetiredNames reports whether a file is allowed to name paths and
// flags that no longer exist. kb/command-tree.md documents the
// reorganization itself -- including, as worked examples, the retired
// spellings and the shrink-a-prefix failure mode this very test exists to
// avoid -- so resolving its prose against the live tree is a category
// error. It was already exempt from the retired-name sweep below; the
// exemption belongs to every check that reads it as an invocation.
func recordsRetiredNames(file string) bool {
	return strings.HasSuffix(file, "command-tree.md")
}

// TestDocs_CommandsExist asserts every documented command path resolves
// in the live cobra tree. This is the test that catches a rename landing
// without a docs sweep.
func TestDocs_CommandsExist(t *testing.T) {
	known, groups := docTreeAndGroups(t)
	inv := collectInvocations(t)
	resolved := 0
	for _, in := range inv {
		if recordsRetiredNames(in.file) {
			continue
		}
		p, unknown := commandPath(in.args, known)
		if unknown != "" {
			t.Errorf("%s:%d documents `truestamp %s ...`, which is not a command\n  %s",
				in.file, in.line, unknown, in.raw)
			continue
		}
		if len(p) == 0 {
			continue
		}
		resolved++
		key := strings.Join(p, " ")
		if !groups[key] {
			continue
		}
		if rest := commandWords(in.args[len(p):]); len(rest) > 0 {
			t.Errorf("%s:%d documents `truestamp %s %s`, but %q is a group with no %q sub-command\n  %s",
				in.file, in.line, key, rest[0], key, rest[0], in.raw)
		}
	}
	// A resolver that silently matched nothing would make this pass while
	// checking nothing at all.
	if resolved < 50 {
		t.Fatalf("only resolved %d command paths out of %d invocations", resolved, len(inv))
	}
	t.Logf("resolved %d command paths across %d documented invocations", resolved, len(inv))
}

// TestDocs_FlagsExist asserts every long flag in a documented invocation
// is accepted by the command it is used with. Inherited persistent flags
// count, because `--help` lists them under Global Flags.
func TestDocs_FlagsExist(t *testing.T) {
	known, _ := docTreeAndGroups(t)
	checked := 0
	for _, in := range collectInvocations(t) {
		if recordsRetiredNames(in.file) {
			continue
		}
		p, unknown := commandPath(in.args, known)
		if unknown != "" || len(p) == 0 {
			// Unresolvable path is reported by TestDocs_CommandsExist.
			continue
		}
		key := strings.Join(p, " ")
		accepts := known[key]
		for _, a := range in.args[len(p):] {
			flag := longFlagRe.FindString(a)
			if flag == "" {
				continue
			}
			checked++
			// Exact membership, NOT strings.Contains against the help text.
			// A substring match passes any flag that is a prefix of a real
			// one: `--hash` rode into the docs on `--hash-type`'s back and
			// survived a full docs sweep, because the help output contains
			// those eight characters.
			if !accepts[strings.TrimPrefix(flag, "--")] {
				t.Errorf("%s:%d documents %s for %q, but the command does not accept it\n  %s",
					in.file, in.line, flag, key, in.raw)
			}
		}
	}
	if checked < 50 {
		t.Fatalf("only checked %d documented flags; the extractor is probably broken", checked)
	}
	t.Logf("checked %d documented flag uses", checked)
}

// retiredCommandPaths are command paths the tree does not have, written
// without the `truestamp` prefix, and why. Prose says "`beacon list`
// accepts 1..100" as often as it writes the whole invocation. The docs
// sweep below and the help-text sweep in tree_test.go both read it.
var retiredCommandPaths = map[string]string{
	"beacon list":    "renamed to `beacons list`",
	"beacon get":     "renamed to `beacons get`",
	"beacon by-hash": "folded into `beacons get`",
	"team show":      "split into `teams get` and `teams current`",
	"team set":       "renamed to `teams use`",
	"team unset":     "renamed to `teams use --clear`",
	"team list":      "renamed to `teams list`",
	"team create":    "renamed to `teams create`",
	"convert proof":  "removed; proofs are JSON only",
	"proofs convert": "removed; proofs are JSON only",
	"convert merkle": "removed with the earlier proof layout",
}

// TestDocs_NoRetiredNames is the belt to TestDocs_CommandsExist's braces.
// A retired name can survive in prose, a table, or a config snippet where
// no `truestamp ` prefix makes it an invocation, and those are exactly the
// places a reader copies from. CHANGELOG.md is exempt as history. A path
// written as one alternative of several (`convert time|id|keyid`) is
// checked as each path it names. Mutant killed: `convert
// time|id|keyid|merkle` put back in README.md.
func TestDocs_NoRetiredNames(t *testing.T) {
	// Each entry is a string that must not appear, and why.
	retired := map[string]string{
		"TRUESTAMP_VERIFY_SILENT":        "--silent is CLI-wide; the env var is TRUESTAMP_SILENT",
		"TRUESTAMP_VERIFY_JSON":          "--json is CLI-wide; the env var is TRUESTAMP_JSON",
		"TRUESTAMP_VERIFY_SKIP_EXTERNAL": "renamed to TRUESTAMP_VERIFY_OFFLINE",
		// Only the config key's spelling: the key is verify.offline.
		"verify.skip_external": "the config key is verify.offline",
		"--skip-external":      "removed; the flag is --offline",
		// The Ed25519 signature check has no switch that skips it.
		retiredSignatureFlag:   "removed; the signature is always checked",
		retiredSignatureEnv:    "removed; the signature is always checked",
		retiredSignatureConfig: "removed; the signature is always checked",
		// The binary carries no reference documentation: the three help
		// topics that briefly existed were each a third copy of something
		// owned elsewhere. `truestamp help <topic>` is an error for all of
		// them.
		"truestamp help formatting": "removed; there are no help topics",
		"truestamp help glossary":   "removed; there are no help topics",
		"truestamp help exit-codes": "removed; there are no help topics",
	}
	for p, why := range retiredCommandPaths {
		retired[p] = why
	}
	// Match on a trailing word boundary, not a bare substring. "beacon
	// list" is a retired command; "the beacon listing card" is ordinary
	// prose, and a Contains check cannot tell them apart.
	pat := map[string]*regexp.Regexp{}
	for needle := range retired {
		pat[needle] = regexp.MustCompile(regexp.QuoteMeta(needle) + `\b`)
	}
	for _, f := range docFiles(t) {
		if recordsRetiredNames(f) {
			continue
		}
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for i, line := range strings.Split(string(body), "\n") {
			variants := expandAlternations(line)
			for needle, why := range retired {
				for _, v := range variants {
					if pat[needle].MatchString(v) {
						t.Errorf("%s:%d still mentions %q (%s)\n  %s",
							f, i+1, needle, why, strings.TrimSpace(line))
						break
					}
				}
			}
		}
	}
}

// alternationRe finds a command path whose last word is an alternation,
// as prose writes `convert time|id|keyid`: one or more words, a space,
// then two or more words joined by `|`.
var alternationRe = regexp.MustCompile(`((?:[a-z][a-z0-9-]* )+)([a-z0-9-]+(?:\|[a-z0-9-]+)+)`)

// expandAlternations returns the line itself and, for each alternation in
// it, the command path spelled out once per alternative, so a retired
// name written as one alternative (`convert time|id|keyid|merkle` naming
// `convert merkle`) is seen as the path it names.
func expandAlternations(line string) []string {
	out := []string{line}
	for _, m := range alternationRe.FindAllStringSubmatch(line, -1) {
		for _, alt := range strings.Split(m[2], "|") {
			out = append(out, m[1]+alt)
		}
	}
	return out
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// TestDocs_ConformanceClaimIsReadFromTheFiles holds README.md's claim of
// conformance (SPEC.md section 13.1, which names the spec commit and the
// digest of MANIFEST.sha256) to the files that make it: README.md cites
// testdata/verification/SOURCE and testdata/verification/MANIFEST.sha256
// and writes neither the commit SOURCE holds nor the manifest's SHA-256,
// whole or as an 8-character prefix, so syncing the spec never leaves the
// README naming an older copy. Mutants killed: every mention of
// testdata/verification/SOURCE, or of testdata/verification/MANIFEST.sha256,
// dropped from README.md; the manifest's digest, or SOURCE's commit as its
// short form, written into README.md.
func TestDocs_ConformanceClaimIsReadFromTheFiles(t *testing.T) {
	root := repoRoot(t)
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	text := strings.ToLower(string(readme))
	for _, cited := range []string{"testdata/verification/SOURCE", "testdata/verification/MANIFEST.sha256"} {
		if !strings.Contains(string(readme), cited) {
			t.Errorf("README.md does not cite %s, which the claim of conformance is read from", cited)
		}
	}
	source, err := os.ReadFile(filepath.Join(root, "testdata", "verification", "SOURCE"))
	if err != nil {
		t.Fatalf("read SOURCE: %v", err)
	}
	commit := strings.TrimSuffix(string(source), "\n")
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(commit) {
		t.Fatalf("SOURCE holds %q, not a commit", source)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "testdata", "verification", "MANIFEST.sha256"))
	if err != nil {
		t.Fatalf("read MANIFEST.sha256: %v", err)
	}
	sum := sha256.Sum256(manifest)
	for what, value := range map[string]string{
		"the spec commit SOURCE names":   commit,
		"the SHA-256 of MANIFEST.sha256": hex.EncodeToString(sum[:]),
	} {
		if strings.Contains(text, value[:8]) {
			t.Errorf("README.md writes %s (%s): cite the file instead, so a sync cannot leave it stale", what, value[:8])
		}
	}
}
