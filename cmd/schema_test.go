// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/truestamp/truestamp-cli/internal/introspect"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// runSchema returns the parsed document, failing the test if the command
// did not succeed or did not emit JSON.
func runSchema(t *testing.T, name string) map[string]any {
	t.Helper()
	// --json is required: schema get renders text by default, like every
	// other command in the tree.
	out, err := exec.Command(binaryPath, "schema", "get", name, "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("schema get %s --json: %v\n%s", name, err, out)
	}
	var doc map[string]any
	if jErr := json.Unmarshal(out, &doc); jErr != nil {
		t.Fatalf("schema get %s --json did not emit JSON: %v\n%s", name, jErr, out)
	}
	return doc
}

func schemaCommandTree(t *testing.T) introspect.Command {
	t.Helper()
	out, err := exec.Command(binaryPath, "schema", "get", "commands", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("schema get commands --json: %v\n%s", err, out)
	}
	var tree introspect.Command
	if jErr := json.Unmarshal(out, &tree); jErr != nil {
		t.Fatalf("parsing command tree: %v", jErr)
	}
	return tree
}

// TestSchema_ListMatchesGet is the consistency property: every document
// `schema list` advertises must actually be gettable, and `schema get` of
// something it does not advertise must fail. A list that drifted from the
// registry would send an agent to a command that errors.
func TestSchema_ListMatchesGet(t *testing.T) {
	out, err := exec.Command(binaryPath, "schema", "list", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("schema list --json: %v\n%s", err, out)
	}
	var listed struct {
		Schemas []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"schemas"`
	}
	if jErr := json.Unmarshal(out, &listed); jErr != nil {
		t.Fatalf("parsing schema list: %v\n%s", jErr, out)
	}
	if len(listed.Schemas) < 3 {
		t.Fatalf("only %d schemas listed; this test would prove little", len(listed.Schemas))
	}
	for _, s := range listed.Schemas {
		if s.Description == "" {
			t.Errorf("schema %q has no description", s.Name)
		}
		if _, err := exec.Command(binaryPath, "schema", "get", s.Name, "--json").CombinedOutput(); err != nil {
			t.Errorf("schema %q is listed but not gettable: %v", s.Name, err)
		}
	}
	if out, err := exec.Command(binaryPath, "schema", "get", "not-a-real-schema", "--json").CombinedOutput(); err == nil {
		t.Errorf("unknown schema name should fail, got: %s", out)
	}
}

// TestSchema_CommandsRoundTrip is the property that makes the document
// trustworthy: every path it lists is invocable, and every invocable path
// is listed. Either direction failing means an agent reading the schema
// gets a different tree from the one it can drive.
func TestSchema_CommandsRoundTrip(t *testing.T) {
	tree := schemaCommandTree(t)
	paths := introspect.Paths(tree)
	if len(paths) < 20 {
		t.Fatalf("tree has only %d paths; the walker is probably broken", len(paths))
	}

	// Forward: everything listed resolves.
	for _, p := range paths {
		args := append(strings.Fields(p)[1:], "--help") // drop the binary name
		if out, err := exec.Command(binaryPath, args...).CombinedOutput(); err != nil {
			t.Errorf("schema lists %q but it does not resolve: %v\n%s", p, err, firstLine(string(out)))
		}
	}

	// Reverse: every command cobra knows about is listed. Walking the
	// binary's own help output would just re-derive the same tree, so
	// this walks the live in-process tree instead, which is a genuinely
	// independent source.
	listed := map[string]bool{}
	for _, p := range paths {
		listed[p] = true
	}
	var missing []string
	for _, c := range rootCmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		p := "truestamp " + c.Name()
		if !listed[p] {
			missing = append(missing, p)
		}
		for _, sub := range c.Commands() {
			if sub.Hidden {
				continue
			}
			sp := p + " " + sub.Name()
			if !listed[sp] {
				missing = append(missing, sp)
			}
		}
	}
	if len(missing) > 0 {
		t.Errorf("commands exist but are absent from schema get commands: %v", missing)
	}
}

// TestSchema_CommandsCarryFlagMetadata checks the document is actually
// useful, not merely well-formed: a caller must be able to construct a
// valid invocation from it without reading help text. verify has no
// --type, and proofs get's carries its six values. Mutant killed: verify
// registering --type again.
func TestSchema_CommandsCarryFlagMetadata(t *testing.T) {
	tree := schemaCommandTree(t)

	verify, ok := introspect.Find(tree, "truestamp verify")
	if !ok {
		t.Fatal("verify missing from the tree")
	}
	byName := map[string]introspect.Flag{}
	for _, f := range verify.Flags {
		byName[f.Name] = f
	}

	if _, ok := byName["type"]; ok {
		t.Error("verify has a --type in the schema; the flag is retired")
	}
	proofsGet, ok := introspect.Find(tree, "truestamp proofs get")
	if !ok {
		t.Fatal("proofs get missing from the tree")
	}
	for _, f := range proofsGet.Flags {
		if f.Name == "type" && (len(f.Values) != 6 || !slices.Contains(f.Values, "entropy_nist")) {
			t.Errorf("proofs get --type should carry its six closed values, got %v", f.Values)
		}
	}

	// The interactive sentinel is exactly the kind of thing an agent must
	// not trip over, so the schema has to disclose it.
	fileFlag, ok := byName["file"]
	if !ok {
		t.Fatal("verify has no --file in the schema")
	}
	if fileFlag.NoOptDefVal == "" {
		t.Error("--file may be passed with no value (opening a picker); the schema must say so")
	}

	// Inherited flags must be present and marked, or a caller would think
	// --base-url is unavailable here.
	base, ok := byName["base-url"]
	if !ok {
		t.Fatal("inherited --base-url missing from verify's flags")
	}
	if !base.Inherited {
		t.Error("--base-url is a root persistent flag and should be marked inherited")
	}
}

// TestSchema_EnumsMatchCompletion pins the reason the enum registry is
// shared: what the schema advertises and what the shell offers must be
// the same set, here for proofs get --type. Mutant killed: the enum
// registry's proofs get|type entry removed (the schema has no values).
func TestSchema_EnumsMatchCompletion(t *testing.T) {
	tree := schemaCommandTree(t)
	proofsGet, _ := introspect.Find(tree, "truestamp proofs get")
	var schemaValues []string
	for _, f := range proofsGet.Flags {
		if f.Name == "type" {
			schemaValues = f.Values
		}
	}
	if len(schemaValues) == 0 {
		t.Fatal("no --type values in the schema; nothing to compare")
	}

	completed := completionValues(t, "proofs", "get", "--type", "")
	if !slices.Equal(schemaValues, completed) {
		t.Errorf("schema and completion disagree about --type\n schema:     %v\n completion: %v",
			schemaValues, completed)
	}
}

// completionValues runs cobra's hidden __complete for args and returns the
// offered values, with the directive line and the trailer stripped.
func completionValues(t *testing.T, args ...string) []string {
	t.Helper()
	out, err := exec.Command(binaryPath, append([]string{"__complete"}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("__complete %v: %v\n%s", args, err, out)
	}
	var values []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "Completion ended") {
			continue
		}
		values = append(values, line)
	}
	return values
}

// specFile is the bytes of the spec data file name, from the CLI's synced
// copy of spec/ (SPEC.md section 9).
func specFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(vectorcases.Dir(), "..", "spec", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// schema lists exactly its four documents, algorithms, commands,
// exit-codes and reason-codes: schema list names those, the shell offers
// those as a document name, schema get refuses subject-types and
// witnesses, and no flag of the command tree is --witnesses. Mutants
// killed: a subject-types document added to schemaDocs (listed, gettable
// and completed); a witnesses document added to schemaDocs (the same); a
// --witnesses flag added to proofs get (the tree carries it).
func TestSchema_ListsExactlyItsDocuments(t *testing.T) {
	out, err := exec.Command(binaryPath, "schema", "list", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("schema list --json: %v\n%s", err, out)
	}
	var listed struct {
		Schemas []struct {
			Name string `json:"name"`
		} `json:"schemas"`
	}
	if err := json.Unmarshal(out, &listed); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range listed.Schemas {
		names = append(names, s.Name)
	}
	if want := []string{"algorithms", "commands", "exit-codes", "reason-codes"}; !slices.Equal(names, want) {
		t.Errorf("schema list names %q, want %q", names, want)
	}
	completed := completionValues(t, "schema", "get", "")
	if want := []string{"algorithms", "commands", "exit-codes", "reason-codes"}; !slices.Equal(completed, want) {
		t.Errorf("schema get completes %q, want %q", completed, want)
	}
	for _, gone := range []string{"subject-types", "witnesses"} {
		if out, err := exec.Command(binaryPath, "schema", "get", gone, "--json").CombinedOutput(); err == nil {
			t.Errorf("schema get %s succeeded: %s", gone, out)
		}
	}
	var walk func(c introspect.Command)
	walk = func(c introspect.Command) {
		for _, f := range c.Flags {
			if f.Name == "witnesses" {
				t.Errorf("%s carries a --witnesses flag", c.Path)
			}
		}
		for _, sub := range c.Subcommands {
			walk(sub)
		}
	}
	walk(schemaCommandTree(t))
}

// schema get reason-codes publishes SPEC.md section 12.1's table, exactly
// spec/reason-codes.json: every code, in its order, with its step, status,
// meaning and sub_reasons. Mutants killed: the document built from a table
// that drops a code, or from one in another order; a sub_reasons list left
// out; a code's status written as its step.
func TestSchema_ReasonCodesAreTheSpecs(t *testing.T) {
	type entry struct {
		Code       string   `json:"code"`
		Meaning    string   `json:"meaning"`
		Status     string   `json:"status"`
		Step       string   `json:"step"`
		SubReasons []string `json:"sub_reasons"`
	}
	var want []entry
	if err := json.Unmarshal(specFile(t, "reason-codes.json"), &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != 36 {
		t.Fatalf("spec/reason-codes.json holds %d codes, want 36", len(want))
	}
	out, err := exec.Command(binaryPath, "schema", "get", "reason-codes", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("schema get reason-codes --json: %v\n%s", err, out)
	}
	var doc struct {
		ReasonCodes []entry `json:"reason_codes"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(doc.ReasonCodes, want, func(a, b entry) bool {
		return a.Code == b.Code && a.Meaning == b.Meaning && a.Status == b.Status && a.Step == b.Step && slices.Equal(a.SubReasons, b.SubReasons)
	}) {
		t.Errorf("reason_codes are\n%v\nwant\n%v", doc.ReasonCodes, want)
	}
}

// TestSchema_AlgorithmsMatchHashList keeps the two views of one registry
// honest: `hash --list` is the human form and `schema get algorithms` the
// machine form, and they read the same source.
func TestSchema_AlgorithmsMatchHashList(t *testing.T) {
	doc := runSchema(t, "algorithms")
	raw, _ := doc["algorithms"].([]any)
	if len(raw) < 10 {
		t.Fatalf("only %d algorithms; expected the full registry", len(raw))
	}
	listOut, err := exec.Command(binaryPath, "hash", "--list").CombinedOutput()
	if err != nil {
		t.Fatalf("hash --list: %v\n%s", err, listOut)
	}
	for _, e := range raw {
		name := e.(map[string]any)["name"].(string)
		if !strings.Contains(string(listOut), name) {
			t.Errorf("algorithm %q is in the schema but not in `hash --list`", name)
		}
	}
}

// TestSchema_TextAndJSONCarryTheSameFacts. Both renderings read one
// registry; the risk is that they drift anyway, because nothing stops a
// contributor updating one and not the other.
//
// This checks the property directly: every value the JSON reports must
// appear in the text. It is what catches a table that quietly stopped
// showing a column after a field was added to the document. Mutant
// killed: the reason-codes text leaving out each code's meaning.
func TestSchema_TextAndJSONCarryTheSameFacts(t *testing.T) {
	// The command tree is excluded: its JSON deliberately carries far more
	// than the text outline (every flag, type, default and enum), which is
	// what --json is for.
	for _, name := range []string{"algorithms", "reason-codes", "exit-codes"} {
		t.Run(name, func(t *testing.T) {
			text, err := exec.Command(binaryPath, "schema", "get", name).CombinedOutput()
			if err != nil {
				t.Fatalf("text: %v\n%s", err, text)
			}
			doc := runSchema(t, name)

			var missing []string
			var walk func(v any)
			walk = func(v any) {
				switch t := v.(type) {
				case map[string]any:
					for _, val := range t {
						walk(val)
					}
				case []any:
					for _, val := range t {
						walk(val)
					}
				case string:
					if t != "" && !strings.Contains(string(text), t) {
						missing = append(missing, t)
					}
				case float64:
					if !strings.Contains(string(text), strconv.FormatFloat(t, 'f', -1, 64)) {
						missing = append(missing, strconv.FormatFloat(t, 'f', -1, 64))
					}
				case bool:
					if !strings.Contains(string(text), strconv.FormatBool(t)) {
						missing = append(missing, strconv.FormatBool(t))
					}
				}
			}
			walk(doc)
			for _, m := range missing {
				t.Errorf("%q is in the JSON but not in the text rendering; the two "+
					"read one registry and must not drift", m)
			}
		})
	}
}

// schema get exit-codes keeps three tables apart: verify's own codes, 0 to
// 8, exactly spec/exit-codes.json (SPEC.md section 12.2) in its order; the
// codes of every other command, exactly 0, 1 and 2; and upgrade's under
// --check --exit-code, exactly its four constants. The text renders each
// under its own heading, its CODE and MEANING rows exactly that table's
// JSON rows in their order. Mutants killed: verifyExitCodes dropping or
// rewording a row of specdata.ExitCodes(); exitCodes given verify's rows 3
// to 8 (the general table is not 0, 1 and 2); verify_exit_codes built from
// the general table; the upgrade codes appended to verify's; the text
// rendering the verify table under the general heading; the text's
// general table rendered with verify's rows 3 to 8 after its own.
func TestSchema_ExitCodesAreKeptApart(t *testing.T) {
	type entry struct {
		Code    int    `json:"code"`
		Meaning string `json:"meaning"`
	}
	var spec []struct {
		Code int    `json:"code"`
		Run  string `json:"run"`
	}
	if err := json.Unmarshal(specFile(t, "exit-codes.json"), &spec); err != nil {
		t.Fatal(err)
	}
	var wantVerify []entry
	for _, e := range spec {
		wantVerify = append(wantVerify, entry{e.Code, e.Run})
	}
	if len(wantVerify) != 9 || wantVerify[0].Code != 0 || wantVerify[8].Code != 8 {
		t.Fatalf("spec/exit-codes.json holds %v, want the codes 0 to 8", wantVerify)
	}
	out, err := exec.Command(binaryPath, "schema", "get", "exit-codes", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("schema get exit-codes --json: %v\n%s", err, out)
	}
	var doc struct {
		General []entry `json:"exit_codes"`
		Verify  []entry `json:"verify_exit_codes"`
		Upgrade []entry `json:"upgrade_check_exit_codes"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(doc.Verify, wantVerify) {
		t.Errorf("verify_exit_codes are %v, want %v", doc.Verify, wantVerify)
	}
	codes := func(list []entry) []int {
		var out []int
		for _, e := range list {
			out = append(out, e.Code)
		}
		return out
	}
	if got := codes(doc.General); !slices.Equal(got, []int{0, 1, panicExitCode}) || panicExitCode != 2 {
		t.Errorf("exit_codes are %v, want 0, 1 and 2", got)
	}
	if !strings.Contains(doc.General[2].Meaning, "panic") || strings.Contains(doc.General[1].Meaning, "verification") {
		t.Errorf("exit_codes read %v: 2 is a panic, and 1 is not a failed verification", doc.General)
	}
	if got := codes(doc.Upgrade); !slices.Equal(got, []int{checkExitUpToDate, checkExitUpgradeAvail, checkExitNetworkErr, checkExitPreRelease}) {
		t.Errorf("upgrade_check_exit_codes are %v", got)
	}
	for code, want := range map[int]string{
		checkExitUpToDate:     "up to date",
		checkExitUpgradeAvail: "upgrade is available",
		checkExitNetworkErr:   "network error",
		checkExitPreRelease:   "pre-release",
	} {
		if i := slices.IndexFunc(doc.Upgrade, func(e entry) bool { return e.Code == code }); i < 0 || !strings.Contains(doc.Upgrade[i].Meaning, want) {
			t.Errorf("upgrade --check code %d does not mention %q: %v", code, want, doc.Upgrade)
		}
	}

	text, err := exec.Command(binaryPath, "schema", "get", "exit-codes").CombinedOutput()
	if err != nil {
		t.Fatalf("schema get exit-codes: %v\n%s", err, text)
	}
	headings := []string{"Exit codes of every other command", "verify (SPEC.md section 12.2)", "upgrade --check --exit-code"}
	sections := map[string]string{}
	rest := string(text)
	for i, heading := range headings {
		at := strings.Index(rest, heading)
		if at < 0 {
			t.Fatalf("the text has no heading %q after the one before it:\n%s", heading, text)
		}
		rest = rest[at+len(heading):]
		end := len(rest)
		if i+1 < len(headings) {
			if next := strings.Index(rest, headings[i+1]); next >= 0 {
				end = next
			}
		}
		sections[heading] = rest[:end]
	}
	row := regexp.MustCompile(`^\s*([0-9]+)\s+(.*\S)\s*$`)
	for heading, list := range map[string][]entry{headings[0]: doc.General, headings[1]: doc.Verify, headings[2]: doc.Upgrade} {
		var rows []entry
		for _, line := range strings.Split(stripANSI(sections[heading]), "\n") {
			if m := row.FindStringSubmatch(line); m != nil {
				code, _ := strconv.Atoi(m[1])
				rows = append(rows, entry{code, m[2]})
			}
		}
		if !slices.Equal(rows, list) {
			t.Errorf("under %q the text's rows are %v, want its JSON rows %v", heading, rows, list)
		}
	}
}

// TestSchema_NotesDescribeTheirOwnDocument: a note describes its own
// document's subject only; behaviour belongs in that command's help.
func TestSchema_NotesDescribeTheirOwnDocument(t *testing.T) {
	out, err := exec.Command(binaryPath, "schema", "list", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("schema list --json: %v", err)
	}
	var listed struct {
		Schemas []struct {
			Name string `json:"name"`
		} `json:"schemas"`
	}
	if jErr := json.Unmarshal(out, &listed); jErr != nil {
		t.Fatalf("parsing schema list: %v", jErr)
	}

	for _, s := range listed.Schemas {
		doc := runSchema(t, s.Name)
		note, _ := doc["note"].(string)
		if note == "" {
			continue
		}
		if strings.Contains(note, "truestamp ") {
			t.Errorf("the %s note names a command (%q); a note describes its own "+
				"document, and a claim about another command's output goes stale "+
				"where nobody looks", s.Name, note)
		}
	}
}

// TestSchema_TextSeparatesTableFromNote: a note butted against the last
// table row reads as another row. Mutant killed: the reason-codes text
// without its closing blank line.
func TestSchema_TextSeparatesTableFromNote(t *testing.T) {
	for _, name := range []string{"exit-codes", "reason-codes", "commands"} {
		t.Run(name, func(t *testing.T) {
			doc := runSchema(t, name)
			note, _ := doc["note"].(string)
			if note == "" {
				t.Skip("no note on this document")
			}
			out, err := exec.Command(binaryPath, "schema", "get", name).Output()
			if err != nil {
				t.Fatalf("schema get %s: %v", name, err)
			}
			lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")

			noteAt := -1
			for i, l := range lines {
				if strings.Contains(l, strings.Fields(note)[0]) && strings.Contains(l, strings.Fields(note)[1]) {
					noteAt = i
					break
				}
			}
			if noteAt <= 0 {
				t.Fatalf("could not locate the note in the text rendering:\n%s", out)
			}
			if strings.TrimSpace(lines[noteAt-1]) != "" {
				t.Errorf("the note is butted against %q; it reads as another row",
					strings.TrimSpace(lines[noteAt-1]))
			}
		})
	}
}

// TestSchema_NoUnscopedEnumForACollidingFlag.
//
// cliEnums may key an entry by bare flag name, which applies it to EVERY
// command carrying a flag of that name. That is safe only when exactly
// one command has it, or when they all mean the same thing.
//
// --type does not: it is a proof subject type on `proofs get` but an id
// shape on `convert id`, so a bare entry makes the shell offer values the
// other command rejects (`convert id --type <TAB>` would offer the six
// proof types).
//
// This is the same hazard internal/config's flagKeyMap had. There it was
// inert because nothing read the value; here it is not.
func TestSchema_NoUnscopedEnumForACollidingFlag(t *testing.T) {
	tree := schemaCommandTree(t)

	// Which commands carry each LOCAL flag name. Inherited flags are the
	// same flag seen from a child, not a second meaning.
	carriers := map[string][]string{}
	var walk func(c introspect.Command)
	walk = func(c introspect.Command) {
		for _, f := range c.Flags {
			if f.Inherited {
				continue
			}
			carriers[f.Name] = append(carriers[f.Name], c.Path)
		}
		for _, sub := range c.Subcommands {
			walk(sub)
		}
	}
	walk(tree)

	for key := range cliEnums() {
		if strings.Contains(key, "|") {
			continue // already scoped to one command
		}
		if paths := carriers[key]; len(paths) > 1 {
			t.Errorf("cliEnums keys %q by bare name, but %d commands carry that flag (%s). "+
				"A bare key applies to all of them, so the shell offers values the others "+
				"reject; scope it as \"<command path>|%s\".",
				key, len(paths), strings.Join(paths, ", "), key)
		}
	}
}

// TestSchema_EveryScopedEnumNamesARealFlag: each "<command path>|<flag>"
// key in cliEnums names a command in the tree that has that local flag.
// registerEnumCompletions looks a key up only for a flag that exists, so
// an entry left behind by a removed command or flag is never read and
// nothing else notices it. Mutant killed: the entry
// "truestamp proofs convert|to" put back in cliEnums.
func TestSchema_EveryScopedEnumNamesARealFlag(t *testing.T) {
	local := map[string]map[string]bool{}
	var walk func(c *cobra.Command, path string)
	walk = func(c *cobra.Command, path string) {
		flags := map[string]bool{}
		c.LocalFlags().VisitAll(func(f *pflag.Flag) { flags[f.Name] = true })
		local[path] = flags
		for _, sub := range c.Commands() {
			walk(sub, path+" "+sub.Name())
		}
	}
	walk(rootCmd, rootCmd.Name())

	scoped := 0
	for key := range cliEnums() {
		path, flag, ok := strings.Cut(key, "|")
		if !ok {
			continue
		}
		scoped++
		flags, isCommand := local[path]
		if !isCommand {
			t.Errorf("cliEnums keys %q, but %q is not a command", key, path)
			continue
		}
		if !flags[flag] {
			t.Errorf("cliEnums keys %q, but %q has no local flag --%s", key, path, flag)
		}
	}
	if scoped < 5 {
		t.Fatalf("only %d scoped keys in cliEnums; this test checks almost nothing", scoped)
	}
}

// TestSchema_CompletionOffersOnlyAcceptedValues is the behavioural half:
// for the closed enums that can be checked offline, every value the shell
// offers must be one the command's own help documents. Mutant killed: the
// completion of proofs get --type offering a value no help names.
func TestSchema_CompletionOffersOnlyAcceptedValues(t *testing.T) {
	for _, tc := range []struct{ path, flag string }{
		{"convert id", "type"},
		{"convert time", "format"},
		{"encode", "to"},
		{"decode", "to"},
		{"proofs get", "constructions"},
		{"proofs get", "type"},
	} {
		t.Run(tc.path+"_"+tc.flag, func(t *testing.T) {
			args := append(strings.Fields(tc.path), "--help")
			help, err := exec.Command(binaryPath, args...).CombinedOutput()
			if err != nil {
				t.Fatalf("--help: %v", err)
			}
			comp, err := exec.Command(binaryPath,
				append(append([]string{"__complete"}, strings.Fields(tc.path)...),
					"--"+tc.flag, "")...).CombinedOutput()
			if err != nil {
				t.Fatalf("__complete: %v", err)
			}
			var offered int
			for _, line := range strings.Split(string(comp), "\n") {
				v := strings.TrimSpace(line)
				if v == "" || strings.HasPrefix(v, ":") || strings.HasPrefix(v, "Completion ended") {
					continue
				}
				offered++
				if !strings.Contains(string(help), v) {
					t.Errorf("completion offers %q for %s --%s, but the command's own "+
						"help never mentions it", v, tc.path, tc.flag)
				}
			}
			if offered == 0 {
				t.Errorf("%s --%s offers no completions; this case asserts nothing", tc.path, tc.flag)
			}
		})
	}
}
