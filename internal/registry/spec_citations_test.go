// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// The source scan of the module's Go comments. SPEC.md and its vectors are
// the source of truth for the verifier, so a comment cites the SPEC.md
// section that governs the code beside it, never a verifier's source. The
// scan reads every comment of every Go file of the module, its tests
// included (go/parser; nothing is compiled or run), and refuses these
// shapes, each a rule of citationRules:
//
//   - "elixir path": a path into an Elixir source tree, the word elixir in
//     any case followed by a slash, as a citation of a verifier's source
//     file;
//   - "verifier path": a path into a verifier's Elixir source, the
//     directory verifier, verify or proof/verifier under a lib/truestamp
//     directory, or the Elixir file of that name;
//   - "module name": a module of elixirVerifierNamespace, which names a
//     verifier's Elixir module rather than the section it implements;
//   - "submodule": a module inside the library's top module or inside the
//     application's port, its top module's name followed by a dot and a
//     capital letter;
//   - "port module": the application's port by its own module name;
//   - "offline claim": the reference verifier, the reference verifiers in
//     this repository or the library by either of its names, called
//     offline, with or without an adverb or an article before the word, as
//     if it made no reads. SPEC.md section 13.1 names the reference
//     verifier, and section 8.3 says that, given a reader, it makes its
//     chain and source reads online.
//
// The bare names SPEC.md section 13.1 gives the library, its top module and
// its repository, are allowed, and so is a path into the application's
// other source, such as its source cache. A qualified statement that
// follows SPEC.md section 8.3 is allowed: its words "Offline, it makes no
// chain or source read", or a claim followed at once by the words "when it
// is given no reader", which restate them. Any other qualification, such as
// "by default" or "unless it is given a reader", is refused with the claim
// it qualifies.
//
// A comment's text is read whole, its lines joined and its whitespace
// collapsed, so a phrase broken across two lines is read as one. A string
// literal is not a comment and is not read.

// elixirVerifierNamespace is the namespace of the Elixir verifier modules
// a comment would name in citing such a module in place of SPEC.md.
const elixirVerifierNamespace = "Truestamp." + "Verifier"

// citationRule is one shape of comment the scan refuses. When allowedAfter
// is set, a match is allowed if allowedAfter matches the text right after
// it.
type citationRule struct {
	name         string
	pattern      *regexp.Regexp
	allowedAfter *regexp.Regexp
}

// find is the first match of the rule in text that is not allowed, or ""
// when there is none.
func (r citationRule) find(text string) string {
	for _, loc := range r.pattern.FindAllStringIndex(text, -1) {
		if r.allowedAfter != nil && r.allowedAfter.MatchString(text[loc[1]:]) {
			continue
		}
		return text[loc[0]:loc[1]]
	}
	return ""
}

// offlineClaim is the "offline claim" rule's pattern: a subject, an
// optional possessive, parenthetical and comma, any run of the listed
// verbs, adverbs and articles, and the word offline; or the adjective form
// before the word reference.
var offlineClaim = regexp.MustCompile(`(?i)` +
	`(?:\breference(?:\s+verifiers?)?(?:\s+in\s+this\s+repository)?|\bTruestamp\.Verify\b|\btruestamp-verify-ex\b)` +
	`(?:'s)?(?:\s*\([^()]*\))?(?:\s*,)?` +
	`(?:\s+(?:which|is|are|was|were|being|stays|remains|runs|works|always|only|purely|also|entirely|strictly|an?))*` +
	`\s+offline\b` +
	`|\boffline[\s-]+(?:only[\s-]+)?references?\b`)

// citationRules are the shapes the scan refuses.
var citationRules = []citationRule{
	{name: "elixir path", pattern: regexp.MustCompile(`(?i)\belixir/`)},
	{name: "verifier path", pattern: regexp.MustCompile(`\blib/truestamp/(?:verifier|verify|proof/verifier)(?:/|\.exs?\b)`)},
	{name: "module name", pattern: regexp.MustCompile(`\b` + regexp.QuoteMeta(elixirVerifierNamespace) + `\b`)},
	{name: "submodule", pattern: regexp.MustCompile(`\bTruestamp\.(?:Verify|Proof\.Verifier)\.[A-Z]`)},
	{name: "port module", pattern: regexp.MustCompile(`\bTruestamp\.Proof\.Verifier\b`)},
	{name: "offline claim", pattern: offlineClaim, allowedAfter: regexp.MustCompile(`^ when it is given no reader\b`)},
}

// commentFile is one parsed file and its comments, by its slash path from
// the module root.
type commentFile struct {
	rel  string
	file *ast.File
}

// parseModuleComments parses every Go file of the module at root, its tests
// included, with its comments. It skips what parseModule skips: the root's
// own skippedDirs, a directory whose name begins with "." at any depth, and
// a nested module.
func parseModuleComments(t *testing.T, root string) (*token.FileSet, []commentFile) {
	t.Helper()
	fset := token.NewFileSet()
	var files []commentFile
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if rel == "." {
				return nil
			}
			if strings.HasPrefix(entry.Name(), ".") || skippedDirs[rel] {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(name, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, name, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		files = append(files, commentFile{rel: rel, file: file})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fset, files
}

// commentText is the text of a comment group: each comment without its
// markers, the comments joined, and every run of whitespace written as one
// space.
func commentText(group *ast.CommentGroup) string {
	var parts []string
	for _, c := range group.List {
		text := c.Text
		if rest, ok := strings.CutPrefix(text, "//"); ok {
			text = rest
		} else {
			text = strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
		}
		parts = append(parts, text)
	}
	return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
}

// citationFindings reports, for each comment group of files, each rule of
// citationRules that finds a match in its text, at the group's first line,
// with the text it matched.
func citationFindings(fset *token.FileSet, files []commentFile) []finding {
	var out []finding
	for _, f := range files {
		for _, group := range f.file.Comments {
			text := commentText(group)
			for _, rule := range citationRules {
				if match := rule.find(text); match != "" {
					out = append(out, finding{
						rel:  f.rel,
						line: fset.Position(group.Pos()).Line,
						what: fmt.Sprintf("%s: %q", rule.name, match),
					})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].rel != out[j].rel {
			return out[i].rel < out[j].rel
		}
		if out[i].line != out[j].line {
			return out[i].line < out[j].line
		}
		return out[i].what < out[j].what
	})
	return out
}

// The planted citations, kept in strings so that no comment of this file
// holds one.
const (
	plantedElixirPath   = "elixir" + "/mix.lock"
	plantedModule       = elixirVerifierNamespace + ".Gate"
	plantedOfflineClaim = "the reference verifier is " + "offline"
)

// No comment of any Go file of the module, tests included, matches a rule
// of citationRules, and the scan is not empty: it reads this file, a test
// file of another package, the command's main and the verifier's package
// doc, and the comments it reads cite SPEC.md sections. Mutants killed
// (each run in a copy of the tree): plantedElixirPath written into a
// comment of internal/verifier/gate.go; plantedOfflineClaim written into a
// comment of internal/verifier/sources.go, on one line, and broken across
// two; plantedModule written into a comment of
// internal/verifier/derive_test.go; a path into the library's verifier
// source, and the sentence of SPEC.md section 10.2 about the reference
// verifiers in this repository, broken across two lines, each written into
// a comment of internal/verifier/gate.go.
func TestCitations_NoGoCommentCitesAVerifiersSourceOrCallsTheReferenceOffline(t *testing.T) {
	fset, files := parseModuleComments(t, repoRoot)
	if findings := citationFindings(fset, files); len(findings) > 0 {
		var lines []string
		for _, f := range findings {
			lines = append(lines, f.String())
		}
		t.Fatalf("Go comments that cite something in place of a SPEC.md section:\n  %s", strings.Join(lines, "\n  "))
	}
	seen := map[string]bool{}
	citing := 0
	for _, f := range files {
		seen[f.rel] = true
		for _, group := range f.file.Comments {
			if strings.Contains(commentText(group), "SPEC.md section") {
				citing++
			}
		}
	}
	for _, want := range []string{
		"internal/registry/spec_citations_test.go",
		"internal/verifier/derive_test.go",
		"internal/verifier/verifier.go",
		"cmd/truestamp/main.go",
	} {
		if !seen[want] {
			t.Errorf("the scan did not read %s", want)
		}
	}
	if len(files) < 300 || citing < 300 {
		t.Fatalf("the scan read %d files and %d comment groups citing SPEC.md", len(files), citing)
	}
}

// plantedComments is a file whose comments hold each shape the scan
// refuses, and, in its last two comments, shapes it allows.
var plantedComments = map[string]string{
	"internal/a/a.go": `package a

// This file ports gate.ex (` + plantedElixirPath + `).
func A() {}

// It follows ` + plantedModule + ` closely.
func B() {}

// A note, since ` + plantedOfflineClaim + `.
func C() {}

// Wrapped: the reference verifier is
// offline, which a scan of one line at a time misses.
func D() {}

// Its counterpart, which the always-offline reference never gives.
func E() {}

// The reference, which is offline and takes its members from a case.
func F() {}

// Every module ports (the reference is offline);
func G() {}

var _ = "` + plantedElixirPath + `" // a string is not read; this comment is: ` + plantedOfflineClaim + `

/* a block comment: ` + plantedElixirPath + ` */
func H() {}

// Its fixtures are the reference verifier's offline reports.
func J() {}

// It ports Elixir/lib/truestamp/hash.ex.
func K() {}

// The Reference verifier is Offline here.
func L() {}

// As the reference verifier runs offline, it draws no window.
func M() {}

// It needs no network, as offline-only references do.
func N() {}

// The library Truestamp.Verify is offline.
func O() {}

// As truestamp-verify-ex is offline, it needs no reader.
func P() {}

// A verifier that makes no reads, such as the reference
// verifiers in this repository (section 8.3), always runs offline.
func Q() {}

// The reference verifier is purely offline.
func R() {}

// The reference verifier is also offline.
func S() {}

// The reference verifier is entirely offline.
func T() {}

// The reference verifier is strictly offline.
func U() {}

// The reference verifier is an offline verifier.
func V() {}

// The reference verifier, being offline, draws no window.
func W() {}

// The reference verifier runs offline by default, and online when it is given a reader.
func X() {}

// The reference verifier is offline unless it is given a reader.
func Y() {}

// The reference verifier is offline when it is given a reader.
func Z() {}

// It ports lib/truestamp/verifier/gate.ex.
func AA() {}

// See lib/truestamp/verify/chains/bitcoin.ex.
func AB() {}

// It follows lib/truestamp/proof/verifier/gate.ex.
func AC() {}

// It follows lib/truestamp/verify.ex.
func AD() {}

// It follows Truestamp.Verify.Online.Frame.
func AE() {}

// It follows Truestamp.Proof.Verifier.Gate.
func AF() {}

// It follows Truestamp.Proof.Verifier as a whole.
func AG() {}

// Allowed, restating SPEC.md section 8.3: the reference verifier runs
// offline when it is given no reader. Offline, it makes no chain or source
// read.
func AH() {}

// Allowed: SPEC.md section 10.2, step 0 (CONFIGURATION), and section 8.3:
// the reference verifier of section 13.1 reads chains and sources online
// when it is given a reader. Offline, the reference verifier makes no chain
// or source read. The library Truestamp.Verify, in truestamp-verify-ex, and
// the application's lib/truestamp/source_cache/read_bounds.ex and
// lib/truestamp/hash.ex are named here as text. elixirish/x.ex.
func I() {}
`,
}

// wantPlantedCitations are the findings plantedComments give, by line.
var wantPlantedCitations = []string{
	"internal/a/a.go:3: elixir path",
	"internal/a/a.go:6: module name",
	"internal/a/a.go:9: offline claim",
	"internal/a/a.go:12: offline claim",
	"internal/a/a.go:16: offline claim",
	"internal/a/a.go:19: offline claim",
	"internal/a/a.go:22: offline claim",
	"internal/a/a.go:25: offline claim",
	"internal/a/a.go:27: elixir path",
	"internal/a/a.go:30: offline claim",
	"internal/a/a.go:33: elixir path",
	"internal/a/a.go:36: offline claim",
	"internal/a/a.go:39: offline claim",
	"internal/a/a.go:42: offline claim",
	"internal/a/a.go:45: offline claim",
	"internal/a/a.go:48: offline claim",
	"internal/a/a.go:51: offline claim",
	"internal/a/a.go:55: offline claim",
	"internal/a/a.go:58: offline claim",
	"internal/a/a.go:61: offline claim",
	"internal/a/a.go:64: offline claim",
	"internal/a/a.go:67: offline claim",
	"internal/a/a.go:70: offline claim",
	"internal/a/a.go:73: offline claim",
	"internal/a/a.go:76: offline claim",
	"internal/a/a.go:79: offline claim",
	"internal/a/a.go:82: verifier path",
	"internal/a/a.go:85: verifier path",
	"internal/a/a.go:88: verifier path",
	"internal/a/a.go:91: verifier path",
	"internal/a/a.go:94: submodule",
	"internal/a/a.go:97: port module",
	"internal/a/a.go:97: submodule",
	"internal/a/a.go:100: port module",
}

// Each planted shape gives exactly its findings, at its comment's first
// line, and the allowed shapes give none: a string literal; line 103's
// qualified claim in SPEC.md section 8.3's words; and line 108's comment,
// which cites SPEC.md, says what the reference verifier does online and
// that it makes no read offline, and names the library's bare names and two
// paths into the application's other source. Mutants killed:
//
//   - a rule of citationRules left out (its lines give no finding);
//   - the comments read one line at a time (line 12's wrapped claim gives
//     none);
//   - the scan reading only declarations' doc comments (the inline comment
//     at line 25 gives none);
//   - string literals read as comments (line 25 gives an elixir path);
//   - the elixir-path rule made case-sensitive (line 33 gives none);
//   - the offline rule made case-sensitive (line 36's capitalised claim
//     gives none);
//   - the possessive left out of the offline rule (line 30 gives none);
//   - the offline rule's verbs cut to the one word is (line 39 gives none);
//   - its adjective form left out (line 16 gives none), or cut to the two
//     words with no only between them (line 42 gives none);
//   - the offline rule widened to any sentence that holds both words (line
//     108 gives one);
//   - the library's module name or its repository's name left out of the
//     offline rule's subjects (line 45 or line 48 gives none);
//   - the words in this repository, the parenthetical or the comma left out
//     of the offline rule (line 51, the sentence of SPEC.md section 10.2,
//     gives none; with no comma, lines 19 and 70 give none too);
//   - purely, also, entirely or strictly left out of the offline rule
//     (line 55, 58, 61 or 64 gives none), or the article (line 67 gives
//     none);
//   - the offline rule's allowance dropped (line 103 gives one), made to
//     allow any text after the claim (lines 73, 76 and 79 give none), or cut
//     to the words when it is given (line 79 gives none);
//   - the verifier-path rule without verifier, verify, proof/verifier or
//     the file form (line 82, 85, 88 or 91 gives none), or widened to any
//     path under the application's lib/truestamp directory (lines 33 and
//     108 give one);
//   - the submodule rule without the library (line 94 gives none), or
//     widened to the library's bare name (line 108 gives one);
//   - the port-module rule left out (lines 97 and 100 lose theirs).
func TestCitations_FindsEachPlantedCitation(t *testing.T) {
	fset := token.NewFileSet()
	var files []commentFile
	for rel, source := range plantedComments {
		file, err := parser.ParseFile(fset, rel, source, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		files = append(files, commentFile{rel: rel, file: file})
	}
	var got []string
	for _, f := range citationFindings(fset, files) {
		name, _, _ := strings.Cut(f.what, ":")
		got = append(got, fmt.Sprintf("%s:%d: %s", f.rel, f.line, name))
	}
	if !slices.Equal(got, wantPlantedCitations) {
		t.Fatalf("findings\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(wantPlantedCitations, "\n  "))
	}
}

// The scan reads every Go file but those under the root's own skipped
// directories, a directory named with a leading dot and a nested module,
// tests included. Mutants killed: test files skipped (a_test.go is not
// read); skippedDirs matched at any depth (internal/v/testdata/a.go is
// not read).
func TestCitations_ReadsTestFilesAndSkipsTheRootsOwnDirectoriesOnly(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"go.mod",
		"main.go",
		"build/a.go", "dist/a.go", "tmp/a.go", "coverage/a.go", "testdata/a.go", "vendor/a.go",
		".hidden/a.go", "internal/v/.hidden/a.go",
		"internal/v/a.go", "internal/v/a_test.go", "internal/v/a.txt",
		"internal/v/testdata/a.go",
		"internal/nested/go.mod", "internal/nested/a.go",
	} {
		name := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		text := "package a\n"
		if strings.HasSuffix(rel, "go.mod") {
			text = "module example.com/a\n"
		}
		if err := os.WriteFile(name, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, files := parseModuleComments(t, root)
	var got []string
	for _, f := range files {
		got = append(got, f.rel)
	}
	sort.Strings(got)
	want := []string{"internal/v/a.go", "internal/v/a_test.go", "internal/v/testdata/a.go", "main.go"}
	if !slices.Equal(got, want) {
		t.Fatalf("the scan read %q, want %q", got, want)
	}
}
