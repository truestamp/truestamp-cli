// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package specdata

// The copies of truestamp-verification that task verification-sync makes.
//
// testdata/verification/ holds spec/, vectors/ and MANIFEST.sha256 from one
// commit of the spec repository, and verification/ beside this file holds
// the same spec/, the runtime pins this package embeds, because go:embed
// reaches only its own package's directory. Each holds SOURCE, the full id of
// that commit.
//
// These tests hold the first copy to its manifest byte for byte, the second
// and the embedded files to the first, so a hand edit, a partial sync, or a
// tool that rewrites a file (a final newline added to a proof, a line-end
// conversion) fails here. A failure is fixed by running task
// verification-sync, never by editing a copied file. Two more checks keep the
// tools that could change a file away from both copies: no copy holds a name
// that gofmt or Go's module rules act on (gofmt -w ., which task fmt runs,
// rewrites every .go file in the tree, testdata/ included, and the module zip
// go install builds from refuses or drops some names), and the root
// .gitattributes stops git's line-end conversion, read here from the file,
// since no test runs git. The last tests make each defect in a scratch copy
// and check that it is reported.
//
// None of this ties the copies to the commit SOURCE names: an edit made
// together with its manifest line passes. task verification-check does that,
// from the spec repository.

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"golang.org/x/mod/module"
)

const (
	// repoRoot is the module root, from this package's directory.
	repoRoot = "../.."
	// fixturesDir is the copy with the vectors and the manifest.
	fixturesDir = "../../testdata/verification"
	// pinsDir is the copy this package embeds.
	pinsDir = "verification"
	// fixturesRepoPath and pinsRepoPath are where the two copies are in the
	// repository, and so in the module zip go install builds from.
	fixturesRepoPath = "testdata/verification"
	pinsRepoPath     = "internal/specdata/verification"
)

// manifestLine is SPEC.md section 9's manifest line: a lowercase SHA-256,
// two spaces and a path under spec/ or vectors/, in the format sha256sum
// reads. No path segment begins with a dot, so no line names ".", "..", or a
// dotfile such as a nested .gitattributes. task verification-sync refuses a
// manifest with any other line.
var manifestLine = regexp.MustCompile(`^([0-9a-f]{64})  ((?:spec|vectors)(?:/[A-Za-z0-9_-][A-Za-z0-9._-]*)+)$`)

// sourceForm is a SOURCE file: a 40-hex commit id and a newline.
var sourceForm = regexp.MustCompile(`^[0-9a-f]{40}\n$`)

// What each copy's top level holds, and the type of each entry, read with
// lstat so a link is reported rather than followed. A linked
// verification/spec would pass every other check and break the embed.
var (
	fixturesLayout = map[string]fs.FileMode{
		"MANIFEST.sha256": 0,
		"SOURCE":          0,
		"spec":            fs.ModeDir,
		"vectors":         fs.ModeDir,
	}
	pinsLayout = map[string]fs.FileMode{
		"SOURCE": 0,
		"spec":   fs.ModeDir,
	}
)

// requiredAttributes are the root .gitattributes lines that stop git
// converting line endings in either copy.
var requiredAttributes = []string{
	"/testdata/verification/** -text",
	"/internal/specdata/verification/** -text",
}

// otherCopyAttributes are root .gitattributes lines that stop git
// converting line endings in a copy another package holds to its source,
// internal/registry's Merkle known answers; each names a path outside both
// copies here, so neither copy's bytes depend on it.
var otherCopyAttributes = []string{"/testdata/merkle/** -text"}

// conversionAttributes are the attributes by which git can change a file's
// bytes on checkout or commit.
var conversionAttributes = []string{"text", "eol", "crlf", "filter", "ident", "working-tree-encoding"}

func TestCopies_FixturesAreExactlyWhatTheirManifestLists(t *testing.T) {
	if problems := manifestProblems(fixturesDir); len(problems) > 0 {
		t.Fatal(report(fixturesDir, problems))
	}
}

func TestCopies_PinsAreTheFixturesSpecByteForByte(t *testing.T) {
	if problems := copyProblems(pinsDir, fixturesDir); len(problems) > 0 {
		t.Fatal(report(pinsDir, problems))
	}
}

// What the binary carries is what File serves, so the embedded files are
// held to the fixtures' spec/ themselves, not only the directory they are
// embedded from.
func TestCopies_EmbeddedFilesAreTheFixturesSpecByteForByte(t *testing.T) {
	var want []string
	for _, e := range entries(fixturesDir, "spec") {
		want = append(want, strings.TrimPrefix(e.path, "spec/"))
	}
	slices.Sort(want)
	if got := Names(); !slices.Equal(got, want) {
		t.Fatalf("Names() = %q, want the files under %s/spec: %q", got, fixturesDir, want)
	}
	for _, name := range want {
		got, err := File(name)
		if err != nil {
			t.Fatalf("File(%q): %v", name, err)
		}
		if !bytes.Equal(got, mustRead(t, filepath.Join(fixturesDir, "spec", name))) {
			t.Errorf("File(%q) differs from %s/spec/%s", name, fixturesDir, name)
		}
	}
}

func TestCopies_BothSourceFilesNameTheSameFullCommitID(t *testing.T) {
	if problems := sourceProblems(fixturesDir, pinsDir); len(problems) > 0 {
		t.Fatal(report("SOURCE", problems))
	}
	if got, want := sourceFile, string(mustRead(t, filepath.Join(fixturesDir, "SOURCE"))); got != want {
		t.Fatalf("the embedded SOURCE is %q, want %s/SOURCE, %q", got, fixturesDir, want)
	}
	if got := Source() + "\n"; got != sourceFile {
		t.Fatalf("Source() = %q, want the embedded SOURCE without its newline", Source())
	}
}

// git is not run here, so its reading of the attributes is not asked for:
// the root file is read for the two lines, and for any other line that could
// change a copied file's bytes, and the directories above the copies for a
// .gitattributes of their own. The one other line allowed is
// otherCopyAttributes' exact line for the Merkle copy, which stops git
// converting that copy and names no path in either copy here. The copies
// themselves cannot hold one: no manifest line names a dotfile, and the pins
// hold only what the fixtures' spec/ holds.
func TestCopies_GitLeavesTheLineEndingsOfEveryCopiedFileAlone(t *testing.T) {
	if problems := gitattributesProblems(repoRoot); len(problems) > 0 {
		t.Fatal("git could convert line endings in a copied file, so a clone with " +
			"core.autocrlf=true would rewrite it. The root .gitattributes needs " +
			fmt.Sprintf("%q", requiredAttributes) + ", and nothing else there or above " +
			"the copies may set a conversion attribute but the exact line " +
			fmt.Sprintf("%q", otherCopyAttributes) + " for the Merkle copy:\n  " +
			strings.Join(problems, "\n  "))
	}
}

// Each check reports its defect, made in a scratch copy of both copies.

func TestManifestProblems_FlippedByteInAListedFile(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	listed := manifestEntries(fixtures)
	if len(listed) == 0 {
		t.Fatal("the synced copy has no well-formed manifest line for this test to break")
	}
	flipFirstByte(t, filepath.Join(fixtures, listed[0].path))
	assertReported(t, manifestProblems(fixtures), listed[0].path+" does not match its digest")
}

func TestManifestProblems_FinalNewlineAddedToAProof(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	var proof string
	for _, l := range manifestEntries(fixtures) {
		if strings.HasSuffix(l.path, ".proof.json") {
			proof = l.path
			break
		}
	}
	if proof == "" {
		t.Fatal("the synced copy lists no .proof.json for this test to break")
	}
	appendTo(t, filepath.Join(fixtures, proof), "\n")
	assertReported(t, manifestProblems(fixtures), proof+" does not match its digest")
}

func TestManifestProblems_FileTheManifestDoesNotList(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	writeFile(t, filepath.Join(fixtures, "vectors", "unlisted.json"), "{}\n")
	assertReported(t, manifestProblems(fixtures), "vectors/unlisted.json is present but not listed")
}

func TestManifestProblems_ListedFileDeleted(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	listed := manifestEntries(fixtures)
	if len(listed) == 0 {
		t.Fatal("the synced copy has no well-formed manifest line for this test to break")
	}
	if err := os.Remove(filepath.Join(fixtures, listed[0].path)); err != nil {
		t.Fatal(err)
	}
	assertReported(t, manifestProblems(fixtures), listed[0].path+" is listed but missing")
}

// This test and the next compare the problems before and after, so a defect
// already in the copy, which the tests above report, does not count as
// theirs.
func TestManifestProblems_LinesOutOfOrder(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	before := manifestProblems(fixtures)
	lines := manifestLines(t, fixtures)
	last := lines[len(lines)-1]
	writeManifest(t, fixtures, append([]string{last}, lines[:len(lines)-1]...))
	added := newProblems(before, manifestProblems(fixtures))
	if !slices.ContainsFunc(added, func(p string) bool { return strings.Contains(p, "not sorted by path") }) {
		t.Fatalf("no new problem says the manifest is not sorted by path: %q", added)
	}
}

func TestManifestProblems_DuplicatedLineReportedOnce(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	before := unnumbered(manifestProblems(fixtures))
	lines := manifestLines(t, fixtures)
	listed := manifestEntries(fixtures)
	if len(listed) == 0 {
		t.Fatal("the synced copy has no well-formed manifest line for this test to break")
	}
	writeManifest(t, fixtures, append([]string{listed[0].digest + "  " + listed[0].path}, lines...))
	added := newProblems(before, unnumbered(manifestProblems(fixtures)))
	if want := listed[0].path + " is listed 2 times"; !slices.Equal(added, []string{want}) {
		t.Fatalf("new problems = %q, want only %q", added, want)
	}
}

func TestManifestProblems_LineOutsideTheGrammar(t *testing.T) {
	for name, line := range map[string]string{
		"an uppercase digest":               strings.Repeat("A", 64) + "  spec/constants.json",
		"one space":                         strings.Repeat("a", 64) + " spec/constants.json",
		"a path outside spec/ and vectors/": strings.Repeat("a", 64) + "  elixir/mix.exs",
		"a segment beginning with a dot":    strings.Repeat("a", 64) + "  spec/.gitattributes",
	} {
		t.Run(name, func(t *testing.T) {
			fixtures, _ := scratchCopies(t)
			writeManifest(t, fixtures, append(manifestLines(t, fixtures), line))
			want := fmt.Sprintf("MANIFEST.sha256 line %d is malformed: %q", len(manifestLines(t, fixtures)), line)
			assertReported(t, manifestProblems(fixtures), want)
		})
	}
}

func TestManifestProblems_NoFinalNewline(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	manifest := filepath.Join(fixtures, "MANIFEST.sha256")
	data := mustRead(t, manifest)
	writeFile(t, manifest, strings.TrimSuffix(string(data), "\n"))
	assertReported(t, manifestProblems(fixtures), "MANIFEST.sha256 has no final newline")
}

// A .go file, planted at several depths of each top directory, is reported
// wherever it is.
func TestManifestProblems_FileGofmtRewrites(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	var planted []string
	for _, dir := range []string{"spec", "spec/constructions", "vectors", "vectors/corpus"} {
		p := dir + "/planted.go"
		writeFile(t, filepath.Join(fixtures, filepath.FromSlash(p)), "")
		planted = append(planted, p)
	}
	problems := manifestProblems(fixtures)
	for _, p := range planted {
		assertReported(t, problems, fixturesRepoPath+"/"+p+" is a .go file, which gofmt -w . rewrites")
	}
}

// Each name Go's module zip refuses or drops, planted in the fixtures with
// its manifest line, is reported by its path in the repository.
func TestManifestProblems_NameGosModuleRulesRefuse(t *testing.T) {
	for p, want := range map[string]string{
		"vectors/corpus/aux.json":          "is refused by Go's module rules",
		"vectors/corpus/x.":                "is refused by Go's module rules",
		"vectors/corpus/con.input.json":    "is refused by Go's module rules",
		"spec/LPT9.json":                   "is refused by Go's module rules",
		"vectors/corpus/vendor/pkg/x.json": "is under a vendor folder, whose files the module zip leaves out",
		"vectors/corpus/jcs/Go.mod":        "is a nested go.mod, which makes the module zip drop its folder and can make go:embed skip it silently",
		"spec/go.mod":                      "is a nested go.mod, which makes the module zip drop its folder and can make go:embed skip it silently",
	} {
		t.Run(p, func(t *testing.T) {
			fixtures, _ := scratchCopies(t)
			writeFile(t, filepath.Join(fixtures, filepath.FromSlash(p)), "{}\n")
			writeManifest(t, fixtures, withLines(t, fixtures, p))
			assertReportedPrefix(t, manifestProblems(fixtures), fixturesRepoPath+"/"+p+" "+want)
		})
	}
}

// A manifest that lists two paths differing only in case is reported, though
// a case-insensitive disk holds them as one file.
func TestManifestProblems_ManifestListsTwoPathsThatDifferOnlyInCase(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	writeFile(t, filepath.Join(fixtures, "vectors", "corpus", "A.json"), "{}\n")
	writeManifest(t, fixtures, withLines(t, fixtures, "vectors/corpus/A.json", "vectors/corpus/a.json"))
	assertReportedPrefix(t, manifestProblems(fixtures),
		fixturesRepoPath+"/vectors/corpus/a.json and "+fixturesRepoPath+"/vectors/corpus/A.json differ only in case")
}

func TestGoToolProblems_TwoPathsThatDifferOnlyInCase(t *testing.T) {
	for name, c := range map[string]struct {
		paths []string
		want  string
	}{
		"two files": {
			paths: []string{"vectors/corpus/A.json", "vectors/corpus/a.json"},
			want:  "x/vectors/corpus/a.json and x/vectors/corpus/A.json differ only in case",
		},
		"two folders": {
			paths: []string{"vectors/Corpus/x.json", "vectors/corpus/y.json"},
			want:  "x/vectors/corpus and x/vectors/Corpus differ only in case",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assertReportedPrefix(t, goToolProblems("x", c.paths), c.want)
		})
	}
}

// Names that only resemble the ones Go refuses are taken: a go.work is an
// ordinary file to gofmt, go:embed and the module zip.
func TestGoToolProblems_NamesGoTakes(t *testing.T) {
	paths := []string{
		"spec/go.work", "vectors/go.work", "vectors/auxiliary.json", "vectors/com10.json",
		"vectors/lpt.json", "vectors/vendors/x.json", "vectors/a.go.json", "vectors/x.y.json",
		"vectors/go.mod.json", "vectors/A.json", "vectors/b.json",
	}
	if problems := goToolProblems(fixturesRepoPath, paths); len(problems) > 0 {
		t.Fatalf("problems = %q, want none", problems)
	}
}

func TestCopyProblems_LinkAtTheTopLevelOfEitherCopy(t *testing.T) {
	fixtures, pins := scratchCopies(t)
	if err := os.RemoveAll(filepath.Join(pins, "spec")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(fixtures, "spec"), filepath.Join(pins, "spec")); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(fixtures, "MANIFEST.sha256")
	outside := filepath.Join(filepath.Dir(fixtures), "MANIFEST.sha256")
	if err := os.Rename(manifest, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, manifest); err != nil {
		t.Fatal(err)
	}
	assertReported(t, copyProblems(pins, fixtures), filepath.Join(pins, "spec")+" is a symlink, not a directory")
	assertReported(t, manifestProblems(fixtures), manifest+" is a symlink, not a regular file")
}

func TestCopyProblems_LinkInsideTheFixtures(t *testing.T) {
	fixtures, _ := scratchCopies(t)
	if err := os.Symlink("constants.json", filepath.Join(fixtures, "spec", "linked.json")); err != nil {
		t.Fatal(err)
	}
	assertReported(t, manifestProblems(fixtures), "spec/linked.json is a symlink, not a regular file")
}

func TestCopyProblems_FlippedByteInThePins(t *testing.T) {
	fixtures, pins := scratchCopies(t)
	flipFirstByte(t, filepath.Join(pins, "spec", "constants.json"))
	assertReported(t, copyProblems(pins, fixtures), "spec/constants.json differs from "+fixtures)
}

func TestCopyProblems_NameGosModuleRulesRefuseInThePins(t *testing.T) {
	fixtures, pins := scratchCopies(t)
	writeFile(t, filepath.Join(pins, "spec", "nul.json"), "{}\n")
	assertReportedPrefix(t, copyProblems(pins, fixtures), pinsRepoPath+"/spec/nul.json is refused by Go's module rules")
}

func TestCopyProblems_ExtraFileInThePins(t *testing.T) {
	fixtures, pins := scratchCopies(t)
	writeFile(t, filepath.Join(pins, "spec", "unlisted.json"), "{}\n")
	assertReported(t, copyProblems(pins, fixtures), "spec/unlisted.json is not in "+fixtures)
}

func TestCopyProblems_FileMissingFromThePins(t *testing.T) {
	fixtures, pins := scratchCopies(t)
	if err := os.Remove(filepath.Join(pins, "spec", "constants.json")); err != nil {
		t.Fatal(err)
	}
	assertReported(t, copyProblems(pins, fixtures), "spec/constants.json is missing")
}

func TestCopyProblems_ExtraEntryAtTheTopOfThePins(t *testing.T) {
	fixtures, pins := scratchCopies(t)
	writeFile(t, filepath.Join(pins, "vectors.json"), "{}\n")
	assertReported(t, copyProblems(pins, fixtures),
		fmt.Sprintf("%s holds %q, not %q", pins, []string{"SOURCE", "spec", "vectors.json"}, []string{"SOURCE", "spec"}))
}

func TestSourceProblems_FilesThatNameDifferentCommits(t *testing.T) {
	fixtures, pins := scratchCopies(t)
	writeFile(t, filepath.Join(pins, "SOURCE"), strings.Repeat("0", 40)+"\n")
	problems := sourceProblems(fixtures, pins)
	if len(problems) != 1 || !strings.HasPrefix(problems[0], "the SOURCE files differ") {
		t.Fatalf("problems = %q, want that the SOURCE files differ", problems)
	}
}

func TestSourceProblems_NotOneLineHoldingA40HexCommitID(t *testing.T) {
	for name, text := range map[string]string{
		"no newline":        strings.Repeat("a", 40),
		"a second newline":  strings.Repeat("a", 40) + "\n\n",
		"uppercase hex":     strings.Repeat("A", 40) + "\n",
		"39 digits":         strings.Repeat("a", 39) + "\n",
		"a ref, not an id":  "main\n",
		"a carriage return": strings.Repeat("a", 40) + "\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			fixtures, pins := scratchCopies(t)
			writeFile(t, filepath.Join(fixtures, "SOURCE"), text)
			writeFile(t, filepath.Join(pins, "SOURCE"), text)
			assertReported(t, sourceProblems(fixtures, pins),
				fmt.Sprintf("%s/SOURCE is not one line holding a 40-hex commit id: %q", fixtures, text))
		})
	}
}

// Each defect is reported, and the allowed lines are not. Mutants killed:
// the skip of otherCopyAttributes removed, which reports the Merkle copy's
// line; the skip widened to a prefix match on /testdata/merkle/, which lets
// a later line convert the Merkle copy again.
func TestGitattributesProblems_EachDefect(t *testing.T) {
	good := "# comment\n" + strings.Join(requiredAttributes, "\n") + "\n"
	cases := []struct {
		name  string
		root  string
		extra map[string]string
		want  string
	}{
		{name: "a required line missing", root: requiredAttributes[0] + "\n",
			want: ".gitattributes lacks the line " + requiredAttributes[1]},
		{name: "a later line that sets text again", root: good + "*.json text\n",
			want: ".gitattributes line 4 sets a conversion attribute: *.json text"},
		{name: "a line that sets eol", root: good + "/testdata/** eol=crlf\n",
			want: ".gitattributes line 4 sets a conversion attribute: /testdata/** eol=crlf"},
		{name: "a line that sets a filter", root: "* filter=lfs\n" + good,
			want: ".gitattributes line 1 sets a conversion attribute: * filter=lfs"},
		{name: "a .gitattributes above a copy", root: good,
			extra: map[string]string{"internal/specdata/.gitattributes": "* text\n"},
			want:  "internal/specdata/.gitattributes may set attributes on a copy"},
		{name: "no root .gitattributes", root: "",
			want: ".gitattributes cannot be read"},
		{name: "a line that sets text on the Merkle copy", root: good + otherCopyAttributes[0] + "\n/testdata/merkle/** text\n",
			want: ".gitattributes line 5 sets a conversion attribute: /testdata/merkle/** text"},
		{name: "a line that sets eol on the Merkle copy", root: good + otherCopyAttributes[0] + "\n/testdata/merkle/** eol=crlf\n",
			want: ".gitattributes line 5 sets a conversion attribute: /testdata/merkle/** eol=crlf"},
		{name: "a wider -text line than the Merkle copy's", root: good + "/testdata/** -text\n",
			want: ".gitattributes line 4 sets a conversion attribute: /testdata/** -text"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			if c.root != "" {
				writeFile(t, filepath.Join(root, ".gitattributes"), c.root)
			}
			for name, text := range c.extra {
				writeFile(t, filepath.Join(root, filepath.FromSlash(name)), text)
			}
			problems := gitattributesProblems(root)
			if !slices.ContainsFunc(problems, func(p string) bool { return strings.HasPrefix(p, c.want) }) {
				t.Fatalf("problems = %q, want one beginning %q", problems, c.want)
			}
		})
	}

	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitattributes"), good+"*.png binary\n*.md diff=markdown\n")
	if problems := gitattributesProblems(root); len(problems) > 0 {
		t.Fatalf("the required lines with lines that convert nothing gave %q, want none", problems)
	}
	// The Merkle copy's exact line is the one conversion attribute allowed
	// besides the required lines.
	merkle := t.TempDir()
	writeFile(t, filepath.Join(merkle, ".gitattributes"), good+"# the Merkle copy\n"+otherCopyAttributes[0]+"\n")
	if problems := gitattributesProblems(merkle); len(problems) > 0 {
		t.Fatalf("the required lines with the Merkle copy's line %q gave %q, want none", otherCopyAttributes[0], problems)
	}
}

// listing is one well-formed manifest line.
type listing struct {
	path   string
	digest string
}

// entry is a file, link or other entry under a copy, by its path relative to
// the copy's root, with '/' between segments, and its type from lstat.
type entry struct {
	path string
	mode fs.FileMode
}

// manifestProblems recomputes every listed digest and holds the files under
// spec/ and vectors/ to exactly the listed ones. The sync replaces the
// directory whole, so it holds nothing else.
func manifestProblems(root string) []string {
	manifest, _ := os.ReadFile(filepath.Join(root, "MANIFEST.sha256"))
	listed := manifestEntries(root)
	present := append(entries(root, "spec"), entries(root, "vectors")...)

	var paths []string
	for _, l := range listed {
		paths = append(paths, l.path)
	}
	var problems []string
	problems = append(problems, layoutProblems(root, fixturesLayout)...)
	problems = append(problems, lineProblems(string(manifest))...)
	problems = append(problems, orderProblems(paths)...)
	problems = append(problems, fileProblems(root, listed, present)...)
	names := slices.Clone(paths)
	for _, e := range present {
		names = append(names, e.path)
	}
	slices.Sort(names)
	problems = append(problems, goToolProblems(fixturesRepoPath, slices.Compact(names))...)
	return problems
}

func lineProblems(manifest string) []string {
	var problems []string
	number := 0
	for _, line := range strings.Split(manifest, "\n") {
		if line == "" {
			continue
		}
		number++
		if !manifestLine.MatchString(line) {
			problems = append(problems, fmt.Sprintf("MANIFEST.sha256 line %d is malformed: %q", number, line))
		}
	}
	if !strings.HasSuffix(manifest, "\n") {
		problems = append(problems, "MANIFEST.sha256 has no final newline")
	}
	return problems
}

// orderProblems: both top directories covered, the paths in byte order (as
// LC_ALL=C sort gives; only the first pair out of order is named), and each
// path once.
func orderProblems(paths []string) []string {
	var problems []string
	for _, top := range []string{"spec/", "vectors/"} {
		if !slices.ContainsFunc(paths, func(p string) bool { return strings.HasPrefix(p, top) }) {
			problems = append(problems, "MANIFEST.sha256 lists no file under "+top)
		}
	}
	for i := 1; i < len(paths); i++ {
		if paths[i-1] > paths[i] {
			problems = append(problems, fmt.Sprintf("MANIFEST.sha256 is not sorted by path: %s comes before %s", paths[i-1], paths[i]))
			break
		}
	}
	counts := map[string]int{}
	for _, p := range paths {
		counts[p]++
	}
	for _, p := range sortedKeys(counts) {
		if counts[p] > 1 {
			problems = append(problems, fmt.Sprintf("%s is listed %d times", p, counts[p]))
		}
	}
	return problems
}

// fileProblems holds the listed files to the present ones. A path listed
// twice is counted once, so it is reported only as repeated.
func fileProblems(root string, listed []listing, present []entry) []string {
	var problems []string
	digests := map[string]string{}
	for _, l := range listed {
		if _, seen := digests[l.path]; !seen {
			digests[l.path] = l.digest
		}
	}
	types := map[string]fs.FileMode{}
	for _, e := range present {
		types[e.path] = e.mode.Type()
		if !e.mode.IsRegular() {
			problems = append(problems, fmt.Sprintf("%s is a %s, not a regular file", e.path, describe(e.mode)))
		}
	}
	for _, p := range sortedKeys(digests) {
		if _, ok := types[p]; !ok {
			problems = append(problems, p+" is listed but missing")
		}
	}
	for _, p := range sortedKeys(types) {
		if _, ok := digests[p]; !ok {
			problems = append(problems, p+" is present but not listed")
		}
	}
	for _, p := range sortedKeys(digests) {
		if mode, ok := types[p]; !ok || !mode.IsRegular() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		sum := sha256.Sum256(data)
		if err != nil || hex.EncodeToString(sum[:]) != digests[p] {
			problems = append(problems, p+" does not match its digest")
		}
	}
	return problems
}

// goToolProblems reports each path, under prefix in the repository, that
// gofmt or Go's module rules act on. gofmt -w . rewrites a .go file even
// under testdata/. go install builds the CLI from Go's module zip of this
// repository, testdata/ included, and that zip is refused for a path
// module.CheckFilePath rejects (a segment ending in a dot, or one whose part
// before its first dot is a name Windows reserves, such as aux or com1) and
// for two paths that differ only in case. It leaves out the files of a
// vendor folder. A nested go.mod, its name in any case, makes the module zip
// drop its folder; go:embed skips that folder silently when the name is
// exactly go.mod, or on a disk that ignores case. The sync refuses each.
func goToolProblems(prefix string, paths []string) []string {
	var problems []string
	folded := map[string]string{}
	for _, p := range paths {
		full := prefix + "/" + p
		if strings.HasSuffix(full, ".go") {
			problems = append(problems, full+" is a .go file, which gofmt -w . rewrites")
		}
		if err := module.CheckFilePath(full); err != nil {
			problems = append(problems, full+" is refused by Go's module rules: "+err.Error())
		}
		segments := strings.Split(full, "/")
		if slices.ContainsFunc(segments, func(s string) bool { return strings.EqualFold(s, "go.mod") }) {
			problems = append(problems, full+" is a nested go.mod, which makes the module zip drop its folder and can make go:embed skip it silently")
		}
		if slices.Contains(segments, "vendor") {
			problems = append(problems, full+" is under a vendor folder, whose files the module zip leaves out")
		}
		for name := full; name != "."; name = path.Dir(name) {
			key := strings.ToLower(name)
			if other, ok := folded[key]; ok && other != name {
				problems = append(problems, name+" and "+other+" differ only in case, which the module zip refuses")
				break
			}
			folded[key] = name
		}
	}
	return problems
}

// copyProblems holds every entry under the pins' spec/ to the fixtures'
// spec/, by path, type and bytes.
func copyProblems(pins, fixtures string) []string {
	expected := contents(fixtures, "spec")
	actual := contents(pins, "spec")
	problems := layoutProblems(pins, pinsLayout)
	problems = append(problems, goToolProblems(pinsRepoPath, sortedKeys(actual))...)
	for _, p := range sortedKeys(expected) {
		if _, ok := actual[p]; !ok {
			problems = append(problems, p+" is missing")
		}
	}
	for _, p := range sortedKeys(actual) {
		want, ok := expected[p]
		switch {
		case !ok:
			problems = append(problems, p+" is not in "+fixtures)
		case actual[p] != want:
			problems = append(problems, p+" differs from "+fixtures)
		}
	}
	return problems
}

func sourceProblems(fixtures, pins string) []string {
	var problems []string
	texts := map[string]string{}
	for _, root := range []string{fixtures, pins} {
		data, err := os.ReadFile(filepath.Join(root, "SOURCE"))
		if err != nil || !sourceForm.Match(data) {
			problems = append(problems, fmt.Sprintf("%s/SOURCE is not one line holding a 40-hex commit id: %q", root, data))
		}
		texts[root] = string(data)
	}
	if len(problems) == 0 && texts[fixtures] != texts[pins] {
		problems = append(problems, fmt.Sprintf("the SOURCE files differ: %s holds %q and %s holds %q",
			fixtures, texts[fixtures], pins, texts[pins]))
	}
	return problems
}

// gitattributesProblems reads root/.gitattributes for the required lines
// and for any other line that sets a conversion attribute, and the
// directories above the copies for a .gitattributes of their own.
func gitattributesProblems(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, ".gitattributes"))
	if err != nil {
		return []string{".gitattributes cannot be read: " + err.Error()}
	}
	var problems []string
	seen := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for number := 1; scanner.Scan(); number++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if slices.Contains(requiredAttributes, line) {
			seen[line] = true
			continue
		}
		if slices.Contains(otherCopyAttributes, line) {
			continue
		}
		fields := strings.Fields(line)
		for _, attribute := range fields[1:] {
			name := strings.TrimLeft(attribute, "-!")
			name, _, _ = strings.Cut(name, "=")
			if slices.Contains(conversionAttributes, name) {
				problems = append(problems, fmt.Sprintf(".gitattributes line %d sets a conversion attribute: %s", number, line))
				break
			}
		}
	}
	for _, line := range requiredAttributes {
		if !seen[line] {
			problems = append(problems, ".gitattributes lacks the line "+line)
		}
	}
	for _, dir := range []string{"testdata", "internal", "internal/specdata"} {
		name := dir + "/.gitattributes"
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(name))); err == nil {
			problems = append(problems, name+" may set attributes on a copy")
		}
	}
	return problems
}

// layoutProblems holds root to a directory holding exactly the layout's
// names, each of its type.
func layoutProblems(root string, layout map[string]fs.FileMode) []string {
	info, err := os.Lstat(root)
	if err != nil {
		return []string{root + " cannot be read: " + err.Error()}
	}
	if !info.IsDir() {
		return []string{fmt.Sprintf("%s is a %s, not a directory", root, describe(info.Mode()))}
	}
	dirEntries, err := os.ReadDir(root)
	if err != nil {
		return []string{root + " cannot be read: " + err.Error()}
	}
	var names []string
	for _, e := range dirEntries {
		names = append(names, e.Name())
	}
	var problems []string
	if expected := sortedKeys(layout); !slices.Equal(names, expected) {
		problems = append(problems, fmt.Sprintf("%s holds %q, not %q", root, names, expected))
	}
	for _, name := range sortedKeys(layout) {
		info, err := os.Lstat(filepath.Join(root, name))
		if err != nil {
			continue
		}
		if info.Mode().Type() != layout[name] {
			problems = append(problems, fmt.Sprintf("%s is a %s, not a %s",
				filepath.Join(root, name), describe(info.Mode()), describe(layout[name])))
		}
	}
	return problems
}

// manifestEntries returns the well-formed lines of root's manifest.
func manifestEntries(root string) []listing {
	data, err := os.ReadFile(filepath.Join(root, "MANIFEST.sha256"))
	if err != nil {
		return nil
	}
	var listed []listing
	for _, line := range strings.Split(string(data), "\n") {
		if m := manifestLine.FindStringSubmatch(line); m != nil {
			listed = append(listed, listing{path: m[2], digest: m[1]})
		}
	}
	return listed
}

// entries returns each entry under root/dir, sorted, from lstat, so a link is
// reported as a link rather than followed, and a directory is walked.
// Nothing when root/dir is not itself a directory, which layoutProblems
// reports.
func entries(root, dir string) []entry {
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil || !info.IsDir() {
		return nil
	}
	dirEntries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		return nil
	}
	var out []entry
	for _, e := range dirEntries {
		p := dir + "/" + e.Name()
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			continue
		}
		if info.IsDir() {
			out = append(out, entries(root, p)...)
			continue
		}
		out = append(out, entry{path: p, mode: info.Mode()})
	}
	return out
}

// contents maps each entry under root/dir to its bytes, or, for an entry
// that is not a regular file, to its type.
func contents(root, dir string) map[string]string {
	out := map[string]string{}
	for _, e := range entries(root, dir) {
		if !e.mode.IsRegular() {
			out[e.path] = "a " + describe(e.mode)
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.path)))
		if err != nil {
			out[e.path] = "unreadable"
			continue
		}
		out[e.path] = "bytes " + string(data)
	}
	return out
}

func describe(mode fs.FileMode) string {
	switch {
	case mode.IsRegular():
		return "regular file"
	case mode.IsDir():
		return "directory"
	case mode&fs.ModeSymlink != 0:
		return "symlink"
	default:
		return "special file (" + mode.Type().String() + ")"
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func report(subject string, problems []string) string {
	return subject + " does not match the synced spec; run task verification-sync:\n  " +
		strings.Join(problems, "\n  ")
}

// scratchCopies copies both copies into a new scratch directory and returns
// their paths there, so a test breaks a copy without touching the real ones.
// Only directories and regular files are copied; anything else in a real
// copy fails the test.
func scratchCopies(t *testing.T) (fixtures, pins string) {
	t.Helper()
	dir := t.TempDir()
	fixtures, pins = filepath.Join(dir, "fixtures"), filepath.Join(dir, "pins")
	copyTree(t, fixturesDir, fixtures)
	copyTree(t, pinsDir, pins)
	return fixtures, pins
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type().IsRegular():
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, 0o644)
		default:
			return fmt.Errorf("%s is not a file or a directory", p)
		}
	})
	if err != nil {
		t.Fatalf("copying %s: %v", src, err)
	}
}

func manifestLines(t *testing.T, root string) []string {
	t.Helper()
	text := strings.TrimSuffix(string(mustRead(t, filepath.Join(root, "MANIFEST.sha256"))), "\n")
	if text == "" {
		t.Fatal("the synced copy has no manifest line for this test to break")
	}
	return strings.Split(text, "\n")
}

// withLines returns root's manifest lines with a line for each of paths, each
// a file holding "{}" and a newline, sorted by path as the sync requires.
func withLines(t *testing.T, root string, paths ...string) []string {
	t.Helper()
	sum := sha256.Sum256([]byte("{}\n"))
	lines := manifestLines(t, root)
	for _, p := range paths {
		lines = append(lines, hex.EncodeToString(sum[:])+"  "+p)
	}
	slices.SortFunc(lines, func(a, b string) int { return strings.Compare(a[66:], b[66:]) })
	return lines
}

func writeManifest(t *testing.T, root string, lines []string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "MANIFEST.sha256"), strings.Join(lines, "\n")+"\n")
}

// newProblems returns the problems of after that before does not hold.
func newProblems(before, after []string) []string {
	var added []string
	for _, p := range after {
		if !slices.Contains(before, p) {
			added = append(added, p)
		}
	}
	return added
}

// unnumberedLine finds a "MANIFEST.sha256 line N" number.
var unnumberedLine = regexp.MustCompile(`^MANIFEST\.sha256 line \d+ `)

// unnumbered leaves line numbers out of problems, since an added line
// renumbers every malformed line after it.
func unnumbered(problems []string) []string {
	out := make([]string, len(problems))
	for i, p := range problems {
		out[i] = unnumberedLine.ReplaceAllString(p, "MANIFEST.sha256 line ")
	}
	return out
}

func assertReportedPrefix(t *testing.T, problems []string, want string) {
	t.Helper()
	if !slices.ContainsFunc(problems, func(p string) bool { return strings.HasPrefix(p, want) }) {
		t.Fatalf("problems = %q, want one beginning %q", problems, want)
	}
}

func assertReported(t *testing.T, problems []string, want string) {
	t.Helper()
	if !slices.Contains(problems, want) {
		t.Fatalf("problems = %q, want them to include %q", problems, want)
	}
}

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeFile(t *testing.T, name, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func appendTo(t *testing.T, name, text string) {
	t.Helper()
	writeFile(t, name, string(mustRead(t, name))+text)
}

func flipFirstByte(t *testing.T, name string) {
	t.Helper()
	data := mustRead(t, name)
	if len(data) == 0 {
		writeFile(t, name, "\n")
		return
	}
	data[0] ^= 1
	writeFile(t, name, string(data))
}
