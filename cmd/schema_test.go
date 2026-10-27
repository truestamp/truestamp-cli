// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/introspect"
	"github.com/truestamp/truestamp-cli/internal/specdata"
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

// TestSchema_SubjectTypesAreFrozen guards the registry CLAUDE.md calls
// frozen. If a code ever changes, every proof signature in existence
// stops verifying, so a test that notices is cheap insurance.
func TestSchema_SubjectTypesAreFrozen(t *testing.T) {
	doc := runSchema(t, "subject-types")
	raw, _ := doc["subject_types"].([]any)
	got := map[string]float64{}
	for _, e := range raw {
		m := e.(map[string]any)
		code, ok := m["code"].(float64)
		if !ok {
			t.Errorf("subject type %v has no code", m["name"])
			continue
		}
		got[m["name"].(string)] = code
	}
	want := map[string]float64{
		"block": 10, "beacon": 11, "item": 20,
		"entropy_nist": 30, "entropy_stellar": 31, "entropy_bitcoin": 32,
	}
	for name, code := range want {
		if got[name] != code {
			t.Errorf("frozen subject type %q: got code %v, want %v", name, got[name], code)
		}
	}
	if len(got) != len(want) {
		t.Errorf("subject type registry changed size: got %d, want %d", len(got), len(want))
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
// showing a column after a field was added to the document.
func TestSchema_TextAndJSONCarryTheSameFacts(t *testing.T) {
	// The command tree is excluded: its JSON deliberately carries far more
	// than the text outline (every flag, type, default and enum), which is
	// what --json is for.
	for _, name := range []string{"algorithms", "subject-types", "exit-codes"} {
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

// TestSchema_ExitCodesComeFromTheConstants pins the numbers to the code
// that produces them. The meanings were once written out twice — in the
// JSON builder and again in the table — and had already drifted apart by
// a clause. Now there is one slice, and the codes are the constants.
func TestSchema_ExitCodesComeFromTheConstants(t *testing.T) {
	doc := runSchema(t, "exit-codes")

	read := func(key string) map[int]string {
		out := map[int]string{}
		raw, _ := doc[key].([]any)
		for _, e := range raw {
			m := e.(map[string]any)
			out[int(m["code"].(float64))] = m["meaning"].(string)
		}
		return out
	}

	general := read("exit_codes")
	if general[panicExitCode] == "" {
		t.Errorf("the panic exit code (%d) is missing from the document", panicExitCode)
	}
	if !strings.Contains(general[panicExitCode], "panic") {
		t.Errorf("code %d should be described as a panic, got %q",
			panicExitCode, general[panicExitCode])
	}

	upgrade := read("upgrade_check_exit_codes")
	for code, want := range map[int]string{
		checkExitUpToDate:     "up to date",
		checkExitUpgradeAvail: "upgrade is available",
		checkExitNetworkErr:   "network error",
		checkExitPreRelease:   "pre-release",
	} {
		got, present := upgrade[code]
		if !present {
			t.Errorf("upgrade --check code %d is missing", code)
			continue
		}
		if !strings.Contains(got, want) {
			t.Errorf("upgrade --check code %d: got %q, want it to mention %q", code, got, want)
		}
	}
}

// schema get exit-codes carries verify's own codes, SPEC.md section
// 12.2's table, equal to internal/specdata's in its order, and the general
// table no longer reads 1 as a failed verification, which verify exits 3
// for. Mutants killed: verifyExitCodes dropping or rewording a row of
// specdata.ExitCodes(); the general row 1 naming a failed verification.
func TestSchema_TheVerifyExitCodesAreTheSpecs(t *testing.T) {
	doc := runSchema(t, "exit-codes")
	raw, _ := doc["verify_exit_codes"].([]any)
	var got []specdata.ExitEntry
	for _, e := range raw {
		m := e.(map[string]any)
		got = append(got, specdata.ExitEntry{Code: int(m["code"].(float64)), Run: m["meaning"].(string)})
	}
	if want := specdata.ExitCodes(); !slices.Equal(got, want) {
		t.Errorf("verify_exit_codes are %v, want %v", got, want)
	}
	general, _ := doc["exit_codes"].([]any)
	for _, e := range general {
		m := e.(map[string]any)
		if m["code"].(float64) == 1 && strings.Contains(m["meaning"].(string), "verification") {
			t.Errorf("the general code 1 reads %q", m["meaning"])
		}
	}
}

// TestSchema_NotesDescribeTheirOwnDocument.
//
// A note is one line of context on a schema document. It must describe
// that document's subject and nothing else.
//
// The subject-types note originally ended "`truestamp inspect` prints the
// code beside the name". That was true when written and false a commit
// later, when inspect stopped reporting the code — a claim about another
// command's behaviour, sitting somewhere nobody would think to update.
// Behaviour belongs next to the behaviour, in that command's own help,
// where changing one without the other is obvious.
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
// table row reads as another row.
func TestSchema_TextSeparatesTableFromNote(t *testing.T) {
	for _, name := range []string{"subject-types", "commands"} {
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
// Three did not. --type is a proof subject type on `verify` and
// `proofs get` but an id shape on `convert id`; --format is a wire format
// on `proofs get` and a timestamp format on `convert time`; --to is a
// wire format on `proofs convert` and a byte encoding on `encode` and
// `decode`. Each bare entry made the shell offer values the other command
// rejects — `convert id --type <TAB>` offered the six proof types.
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
		{"proofs convert", "to"},
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
