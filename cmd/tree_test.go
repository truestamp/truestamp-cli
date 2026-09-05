// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// These tests pin the structural rules from kb/command-tree.md against the
// live tree, so a future command cannot quietly violate them. They are
// deliberately about shape rather than behavior: the behavior of each
// command is covered by its own file.

// resourceGroups are the plural noun groups (R0). Everything about them is
// asserted uniformly, which is the point of having rules at all.
var resourceGroups = []string{"items", "proofs", "blocks", "beacons", "entropy", "keys", "teams"}

// TestTree_BareGroupPrintsHelp pins R0 for every group at once: a group is
// a namespace, never a command. Two groups used to run something —
// `beacon` ran `latest`, `team` ran `list` — which meant the group name
// meant two things, and there was no rule for the seventh noun.
//
// It also asserts help works with NO credential. `items list` needs one,
// so a group that ran its list would answer "what can I do here?" with an
// auth error on a fresh machine.
// allGroups is every namespace in the tree: the resource groups plus the
// four that own no server records. A fresh slice each call, so appending
// to it can never alias resourceGroups.
func allGroups() []string {
	return append(append([]string(nil), resourceGroups...), "auth", "config", "convert", "schema")
}

func TestTree_BareGroupPrintsHelp(t *testing.T) {
	for _, g := range allGroups() {
		t.Run(g, func(t *testing.T) {
			out, err := exec.Command(binaryPath, g).CombinedOutput()
			if err != nil {
				t.Fatalf("bare group %q must exit 0, got %v\n%s", g, err, out)
			}
			if !strings.Contains(string(out), "Available Commands:") {
				t.Errorf("bare group %q should print help, got:\n%s", g, out)
			}
		})
	}
}

// TestTree_GroupRejectsUnknownSubcommand is the other half of R0, and the
// reason groups have a RunE at all. Cobra's default for a command with
// subcommands and no Run is to print help and exit 0 EVEN WHEN handed an
// argument it does not recognize — so after `proof` moved to
// `proofs convert`, `truestamp convert proof` printed help and exited 0,
// giving a reader following an old document no signal whatsoever.
func TestTree_GroupRejectsUnknownSubcommand(t *testing.T) {
	for _, g := range allGroups() {
		t.Run(g, func(t *testing.T) {
			out, err := exec.Command(binaryPath, g, "definitely-not-a-subcommand").CombinedOutput()
			if err == nil {
				t.Fatalf("%s with an unknown subcommand must exit non-zero, got:\n%s", g, out)
			}
			if !strings.Contains(string(out), "unknown command") {
				t.Errorf("%s should say what went wrong, got:\n%s", g, out)
			}
		})
	}
}

// TestTree_GroupsDoNoResourceWork pins the R0 intent: a group's RunE may
// only print help or reject an unknown argument. It must never reach the
// network — a bare group is how a new user asks "what can I do here?", and
// that question must be answerable with no credential.
func TestTree_GroupsDoNoResourceWork(t *testing.T) {
	for _, g := range resourceGroups {
		t.Run(g, func(t *testing.T) {
			// No credential, and a base URL that cannot be reached: if the
			// group did any resource work it would fail or hang here.
			out, err := exec.Command(binaryPath, g,
				"--base-url", "http://127.0.0.1:9").CombinedOutput()
			if err != nil {
				t.Errorf("bare group %q did resource work: %v\n%s", g, err, out)
			}
		})
	}
}

// TestTree_NoAliasesAnywhere pins the clean break. Cobra Aliases are the
// easiest compatibility affordance to add back by accident, and adding one
// re-opens the reorganization it was the point of closing.
func TestTree_NoAliasesAnywhere(t *testing.T) {
	var walk func(c *cobra.Command, path string)
	walk = func(c *cobra.Command, path string) {
		if len(c.Aliases) > 0 {
			t.Errorf("%s declares aliases %v; the tree is a clean break with no aliases (kb/command-tree.md)",
				path, c.Aliases)
		}
		if c.Deprecated != "" {
			t.Errorf("%s is marked Deprecated; retired names are removed, not deprecated", path)
		}
		for _, sub := range c.Commands() {
			walk(sub, path+" "+sub.Name())
		}
	}
	walk(rootCmd, "truestamp")
}

// TestTree_NoBannedVerbs pins the closed verb vocabulary. `show`, `view`
// and the rest are banned as synonyms of `get`; `status` means credential
// liveness only, so `auth status` is the sole legitimate use. `edit` is a
// sanctioned write verb, not a synonym: see the note in `banned` below.
func TestTree_NoBannedVerbs(t *testing.T) {
	banned := map[string]string{
		"show": "use `get` (R9)", "view": "use `get` (R9)", "describe": "use `get` (R9)",
		"info": "use `get` (R9)", "fetch": "use `get` (R9)", "download": "use `get` (R9)",
		"retrieve": "use `get` (R9)", "new": "use `create` (R3)", "add": "use `create` (R3)",
		"set":     "use `create` or `update` (R3)",
		"by-hash": "fold into `get` by id shape",
		// `edit` was banned here as a synonym of `update`, and is now a
		// write verb in its own right. It is not a synonym: `update` sets
		// named fields non-interactively (`items update --tags q3`), while
		// `edit` hands a whole document to the user's editor and reads
		// back whatever they saved. Collapsing the two would have made
		// `update` mean two different things, which R13 forbids. It stays
		// restricted to a file the CLI owns -- see the R3 note in
		// kb/command-tree.md.
	}
	var walk func(c *cobra.Command, path string)
	walk = func(c *cobra.Command, path string) {
		// `config show` is the one documented exemption: `config` is not a
		// resource group, it owns no records, and it never gains a `get`.
		// A future scalar reader is `config show <key>`. See R9 and the
		// boundary note in kb/command-tree.md.
		if why, bad := banned[c.Name()]; bad && path != "truestamp config show" {
			t.Errorf("%s uses the banned verb %q: %s", path, c.Name(), why)
		}
		// `status` is reserved for credential liveness. auth status is the
		// one legitimate use; anything else collides with the Appendix E.22
		// per-step vocabulary and puts a server claim where evidence belongs.
		if c.Name() == "status" && path != "truestamp auth status" {
			t.Errorf("%s uses `status`, which means credential liveness only (R9)", path)
		}
		for _, sub := range c.Commands() {
			walk(sub, path+" "+sub.Name())
		}
	}
	walk(rootCmd, "truestamp")
}

// TestTree_LatestAndCurrentAreNotBothPresent pins R1a: a noun gets at most
// one selector, decided by whether the answer changes because something
// was appended (latest) or because a designation moved (current).
func TestTree_LatestAndCurrentAreNotBothPresent(t *testing.T) {
	for _, name := range resourceGroups {
		g := findCommand(t, name)
		var hasLatest, hasCurrent bool
		for _, sub := range g.Commands() {
			switch sub.Name() {
			case "latest":
				hasLatest = true
			case "current":
				hasCurrent = true
			}
		}
		if hasLatest && hasCurrent {
			t.Errorf("group %q has both `latest` and `current`; R1a allows at most one", name)
		}
	}
	// And the specific assignments, which are the load-bearing ones.
	for group, want := range map[string]string{
		"blocks": "latest", "beacons": "latest", "entropy": "latest", "keys": "current", "teams": "current",
	} {
		g := findCommand(t, group)
		found := false
		for _, sub := range g.Commands() {
			if sub.Name() == want {
				found = true
			}
		}
		if !found {
			t.Errorf("group %q should have %q per R1a", group, want)
		}
	}
}

// TestTree_CurrentImpliesUse pins R2: `current` plus `use` means the user
// points; `current` alone means the system points. keys has no `use`
// because Truestamp chooses the active key; teams does because you do.
func TestTree_CurrentImpliesUse(t *testing.T) {
	if hasSub(t, "teams", "use") == false {
		t.Error("teams has `current` but no `use`; R2 says the pair documents who points")
	}
	if hasSub(t, "keys", "use") {
		t.Error("keys has `use`, implying the user selects the signing key; Truestamp does")
	}
}

// TestTree_ReadOnlyGroupsHaveNoWriteVerbs pins R11: read-only is a stated
// property, not an accident. A `create` appearing on blocks or keys would
// mean someone believed the CLI could write the chain.
func TestTree_ReadOnlyGroupsHaveNoWriteVerbs(t *testing.T) {
	for _, name := range []string{"blocks", "beacons", "entropy", "keys"} {
		g := findCommand(t, name)
		if !strings.HasPrefix(g.Short, "Read-only:") {
			t.Errorf("group %q is read-only and its Short must say so (R11), got %q", name, g.Short)
		}
		for _, sub := range g.Commands() {
			switch sub.Name() {
			case "create", "update", "delete":
				t.Errorf("read-only group %q has write verb %q", name, sub.Name())
			}
		}
	}
}

// TestTree_NoProofVerbOnSourceNouns pins R7: a proof bundle is a derived
// artifact and lives in one place. `items proof` and `blocks proof` would
// each bring their own --type vocabulary and filename convention.
func TestTree_NoProofVerbOnSourceNouns(t *testing.T) {
	for _, name := range []string{"items", "blocks", "beacons", "entropy"} {
		if hasSub(t, name, "proof") || hasSub(t, name, "proofs") {
			t.Errorf("group %q has a proof verb; proofs are a derived artifact with their own group (R7)", name)
		}
	}
}

// TestTree_EveryRootCommandIsGrouped keeps root help readable: an
// ungrouped command falls into cobra's "Additional Commands" bucket, which
// is where `help` and `completion` belong and nothing else.
func TestTree_EveryRootCommandIsGrouped(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		// Help topics are not commands: cobra recognises a command with no
		// Run and no subcommands and lists it under its own "Additional
		// help topics:" heading, which is where they belong.
		if !c.Runnable() && !c.HasSubCommands() {
			continue
		}
		if c.GroupID == "" {
			t.Errorf("root command %q has no GroupID and would land under 'Additional Commands'", c.Name())
		}
	}
}

// TestTree_RetiredNamesAreGone is the direct assertion that the rename
// happened and left nothing behind.
func TestTree_RetiredNamesAreGone(t *testing.T) {
	// Deliberately no --help: cobra handles the help flag before argument
	// validation, so `truestamp convert proof --help` prints convert's help
	// and exits 0 on any group. What must fail is the invocation a reader
	// following an old document would actually type.
	for _, args := range [][]string{
		{"beacon"}, {"team"}, {"create"}, {"download"},
		{"beacon", "by-hash"}, {"team", "show"}, {"team", "set"}, {"team", "unset"},
		{"convert", "proof"},
	} {
		out, err := exec.Command(binaryPath, args...).CombinedOutput()
		if err == nil {
			t.Errorf("retired command %v still resolves:\n%s", args, out)
		}
	}
}

func findCommand(t *testing.T, name string) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("command %q not found in the tree", name)
	return nil
}

func hasSub(t *testing.T, group, sub string) bool {
	t.Helper()
	for _, c := range findCommand(t, group).Commands() {
		if c.Name() == sub {
			return true
		}
	}
	return false
}

// --- help topics -------------------------------------------------------

// TestTree_RootHelpLeadsWithCommands is the whole point of the
// reorganization, and it is easy to undo by accident.
//
// Cobra's help template prints Example BEFORE the command groups, not
// after. A pointer block in the root command's Example therefore pushes
// every actual command below it — so the first thing a reader saw was a
// list of help topics, and the commands were below the fold. Nothing
// about the groups themselves catches that; only the ordering does.
func TestTree_RootHelpLeadsWithCommands(t *testing.T) {
	out, err := exec.Command(binaryPath, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	help := string(out)

	firstGroup := strings.Index(help, "Verify a proof:")
	if firstGroup < 0 {
		t.Fatal("the first command group heading is missing from root help")
	}
	// Nothing may sit between the usage line and the first group but the
	// blank line separating them.
	usage := strings.Index(help, "Usage:")
	if usage < 0 || usage > firstGroup {
		t.Fatal("unexpected help layout")
	}
	between := help[usage:firstGroup]
	for _, intruder := range []string{"Examples:", "Aliases:"} {
		if strings.Contains(between, intruder) {
			t.Errorf("%q sits between the usage line and the commands, pushing them down:\n%s",
				intruder, between)
		}
	}

}

// --- runtime strings ---------------------------------------------------

// TestTree_NoRetiredNamesInRuntimeStrings scans every Short, Long and
// Example in the tree for a retired command path.
//
// The docs tests catch stale Markdown, but a stale hint printed by the
// binary itself is worse: the user is already in the tool, following its
// own instruction, and it names a command that no longer exists. This
// found four such strings when it was written, including `teams current`
// telling the reader to run `truestamp team set`.
func TestTree_NoRetiredNamesInRuntimeStrings(t *testing.T) {
	retired := []string{
		"truestamp team ", "truestamp beacon ", "truestamp create",
		"truestamp download", "truestamp convert proof",
		"truestamp team'", "truestamp beacon'",
	}
	var walk func(c *cobra.Command, path string)
	walk = func(c *cobra.Command, path string) {
		for field, text := range map[string]string{
			"Short": c.Short, "Long": c.Long, "Example": c.Example,
		} {
			for _, bad := range retired {
				if strings.Contains(text, bad) {
					t.Errorf("%s %s mentions the retired %q", path, field, strings.TrimSpace(bad))
				}
			}
		}
		c.LocalFlags().VisitAll(func(f *pflag.Flag) {
			for _, bad := range retired {
				if strings.Contains(f.Usage, bad) {
					t.Errorf("%s --%s usage mentions the retired %q", path, f.Name, strings.TrimSpace(bad))
				}
			}
		})
		for _, sub := range c.Commands() {
			walk(sub, path+" "+sub.Name())
		}
	}
	walk(rootCmd, "truestamp")
}

// TestTree_HelpNeverAdvertisesWhatIsNotBuilt extends R12 from commands to
// the text the binary prints.
//
// R12 keeps a server-blocked command out of the binary entirely, because
// predicting a command correctly and then hitting "not available" is
// worse than the command not existing. Help text can violate the same
// principle without registering anything: a `destructive` topic once
// opened with "Nothing in this CLI currently deletes a Truestamp record"
// and then spent two thirds of its length on item retirement that is not
// reachable from here. That is a placeholder with no command attached.
//
// Describing a REAL constraint is fine and expected — "claims are
// immutable", "a proof cannot be generated until the item is committed".
// What is not fine is promising a capability, or documenting one that
// lives somewhere else as though it were pending here.
func TestTree_HelpNeverAdvertisesWhatIsNotBuilt(t *testing.T) {
	// Phrases that only make sense when text is describing something this
	// binary does not do.
	banned := []string{
		"when item retirement does arrive",
		"does arrive",
		"is not yet",
		"not yet available",
		"coming soon",
		"will be added",
		"in a future release",
		"once the server",
		"until the commands",
	}
	var walk func(c *cobra.Command, path string)
	walk = func(c *cobra.Command, path string) {
		for field, text := range map[string]string{
			"Short": c.Short, "Long": c.Long, "Example": c.Example,
		} {
			lower := strings.ToLower(text)
			for _, bad := range banned {
				if strings.Contains(lower, bad) {
					t.Errorf("%s %s says %q; help describes what this binary does, "+
						"and unbuilt capability lives in kb/command-tree.md (R12)",
						path, field, bad)
				}
			}
		}
		for _, sub := range c.Commands() {
			walk(sub, path+" "+sub.Name())
		}
	}
	walk(rootCmd, "truestamp")
}

// TestTree_CaveatsLiveOnTheirCommands: removing the help topics must not
// lose the constraints they carried. Each one is a fact about a specific
// command, so it belongs in that command's help.
func TestTree_CaveatsLiveOnTheirCommands(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
		why  string
	}{
		{[]string{"items", "update", "--help"}, "immutable",
			"no flag here can reach a signed field"},
		{[]string{"teams", "use", "--help"}, "persists",
			"it writes config.toml"},
		{[]string{"proofs", "get", "--help"}, "stdout",
			"the payload goes to stdout unless a file is named"},
		{[]string{"keys", "--help"}, "no credential",
			"this is the only group that needs none"},
	} {
		out, err := exec.Command(binaryPath, tc.args...).CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if !strings.Contains(string(out), tc.want) {
			t.Errorf("%v should still explain that %s", tc.args, tc.why)
		}
	}
}

// TestTree_NoHelpTopics keeps the binary out of the documentation
// business.
//
// Three topics briefly existed: glossary, formatting and exit-codes. Each
// was a third copy of something already owned elsewhere — the vocabulary
// belongs to the whitepaper in truestamp-v2, the output contract to
// README and kb/command-tree.md R10, and the exit codes to README and to
// `schema get exit-codes`, which is generated. Prose in a Go string
// cannot be kept in sync with a document in another repository, and no
// test can check it, which is the same drift hazard CLAUDE.md already
// names as a hard rule for the two algorithm registries.
//
// Per-command help still explains the constraints that apply to that
// command, because those are facts about the command.
func TestTree_NoHelpTopics(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		// A cobra help topic is a command with no Run and no subcommands.
		if !c.Runnable() && !c.HasSubCommands() {
			t.Errorf("%q is a help topic; documentation belongs in README, kb/, "+
				"the whitepaper, or `schema`, each of which owns its subject",
				c.Name())
		}
	}
}

// TestTree_EveryCommandNamedInHelpTextExists.
//
// The strongest guard against this whole class: find every
// `truestamp <words>` invocation in every STRING LITERAL under cmd/ and
// internal/, and check each one resolves.
//
// Literals, via the AST, rather than raw file text: a comment explaining
// that `truestamp convert proof` used to exist is legitimate history, and
// grepping would flag it. What must not survive is an invocation the
// binary can actually print.
//
// TestTree_NoRetiredNamesInRuntimeStrings only catches a hard-coded list
// from one reorganization, and only in cobra fields. This catches any
// command named anywhere in printable text that does not exist, whatever
// the reason — which is what an audit found four of, all of them in
// runtime hint strings rather than help fields: `beacons list` and
// `teams list` each printed a hint naming a retired command, `teams
// create`'s success card closed by telling the user to run one, and the
// console's Teams pane named two at once.
func TestTree_EveryCommandNamedInHelpTextExists(t *testing.T) {
	// `help` is installed by Execute, not by an init, so the in-process
	// tree only carries it if an earlier test happened to call Execute.
	// Install it here so the walk sees the tree a user sees, whatever ran
	// before.
	registerHelpCommand(rootCmd)
	rootCmd.InitDefaultHelpCmd()

	// An INVOCATION, not prose. Two shapes count: quoted, as a hint does
	// ("'truestamp teams use <id>'"), or INDENTED at the start of a line,
	// as an example block does ("  truestamp convert merkle ...").
	//
	// The indentation matters. Unindented line-initial text is wrapped
	// prose or an output message — "truestamp is up to date (%s)", or a
	// Long paragraph that happened to wrap onto "truestamp subcommand
	// uses..." — and matching those produced four false positives.
	re := regexp.MustCompile("(?m)(?:^[ \\t]+|['`\"])truestamp((?: [a-z][a-z0-9-]*)+)")

	// Walk the live tree rather than shrinking a path until something
	// resolves. Shrinking accepts a prefix, which is exactly wrong here:
	// "truestamp teams set" would pass because `teams` exists, while the
	// reader who types it gets "unknown command \"set\"".
	//
	// Returns the deepest command reached and how many words it consumed.
	walkTo := func(words []string) (*cobra.Command, int) {
		cur, used := rootCmd, 0
		for _, w := range words {
			var next *cobra.Command
			for _, sub := range cur.Commands() {
				if sub.Name() == w {
					next = sub
					break
				}
			}
			if next == nil {
				break
			}
			cur, used = next, used+1
		}
		return cur, used
	}

	root := repoRoot(t)
	var checked int
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") {
				return err
			}
			fset := token.NewFileSet()
			file, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				t.Fatalf("parsing %s: %v", path, perr)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				text, uerr := strconv.Unquote(lit.Value)
				if uerr != nil {
					text = lit.Value // raw string with an escape strconv rejects
				}
				for _, m := range re.FindAllStringSubmatch(text, -1) {
					words := strings.Fields(m[1])
					checked++
					cmdAt, used := walkTo(words)
					// Unconsumed words are fine when the command reached is a
					// leaf — they are its arguments ("truestamp hash doc").
					// They are NOT fine when it still has subcommands: the
					// author wrote a subcommand that does not exist.
					if used < len(words) && cmdAt.HasSubCommands() {
						rel, _ := filepath.Rel(root, path)
						where := "truestamp"
						if used > 0 {
							where += " " + strings.Join(words[:used], " ")
						}
						t.Errorf("%s:%d names 'truestamp %s', but %q has no sub-command %q",
							rel, fset.Position(lit.Pos()).Line,
							strings.Join(words, " "), where, words[used])
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", dir, err)
		}
	}
	if checked < 20 {
		t.Fatalf("only found %d invocations in string literals; the matcher is probably broken", checked)
	}
	t.Logf("checked %d invocations in string literals", checked)
}
