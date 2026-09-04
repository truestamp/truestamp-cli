// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package introspect

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

// fixture builds a small tree with every shape the walker has to handle:
// a group with no Run, a runnable leaf, an inherited persistent flag, a
// hidden command, a flag with a shorthand and a default, and a flag with
// a NoOptDefVal sentinel.
func fixture() *cobra.Command {
	root := &cobra.Command{Use: "tool", Short: "root"}
	root.PersistentFlags().String("base-url", "https://example.test", "the origin")

	group := &cobra.Command{Use: "items", Short: "a group"}

	get := &cobra.Command{
		Use: "get", Short: "a leaf",
		Run: func(*cobra.Command, []string) {},
	}
	get.Flags().StringP("format", "f", "json", "output format")
	get.Flags().String("file", "", "a path")
	get.Flags().Lookup("file").NoOptDefVal = "(pick)"

	hidden := &cobra.Command{Use: "secret", Hidden: true, Run: func(*cobra.Command, []string) {}}

	group.AddCommand(get, hidden)
	root.AddCommand(group)
	return root
}

func TestWalk_RecordsPathsAndRunnability(t *testing.T) {
	tree := Walk(fixture(), nil, false)

	if tree.Path != "tool" {
		t.Errorf("root path: got %q", tree.Path)
	}
	if tree.Runnable {
		t.Error("a root with no Run should not be marked runnable")
	}
	if len(tree.Subcommands) != 1 {
		t.Fatalf("expected one visible subcommand, got %d", len(tree.Subcommands))
	}

	group := tree.Subcommands[0]
	if group.Path != "tool items" {
		t.Errorf("group path: got %q", group.Path)
	}
	if !group.Group {
		t.Error("a non-runnable command with subcommands is a group")
	}
	if group.Runnable {
		t.Error("the group has no Run and must not be marked runnable")
	}
}

// TestWalk_HidesHiddenByDefault: hidden commands are part of the tree but
// not part of the interface, and `schema get commands` describes the
// interface.
func TestWalk_HidesHiddenByDefault(t *testing.T) {
	visible := Paths(Walk(fixture(), nil, false))
	if slices.Contains(visible, "tool items secret") {
		t.Error("a hidden command must not appear by default")
	}
	withHidden := Paths(Walk(fixture(), nil, true))
	if !slices.Contains(withHidden, "tool items secret") {
		t.Error("includeHidden should surface it")
	}
}

// TestWalk_SkipsCobraScaffolding: `help` and `completion` are cobra's own
// machinery, not part of the interface being described.
func TestWalk_SkipsCobraScaffolding(t *testing.T) {
	root := fixture()
	root.AddCommand(&cobra.Command{Use: "help", Run: func(*cobra.Command, []string) {}})
	root.AddCommand(&cobra.Command{Use: "completion", Run: func(*cobra.Command, []string) {}})
	for _, p := range Paths(Walk(root, nil, true)) {
		if p == "tool help" || p == "tool completion" {
			t.Errorf("%q should not be described", p)
		}
	}
}

// TestCollectFlags_MarksInheritedAndCarriesMetadata is what makes the
// document usable: a caller must be able to build a valid invocation from
// it without reading help text.
func TestCollectFlags_MarksInheritedAndCarriesMetadata(t *testing.T) {
	tree := Walk(fixture(), nil, false)
	get, ok := Find(tree, "tool items get")
	if !ok {
		t.Fatal("leaf not found")
	}

	byName := map[string]Flag{}
	for _, f := range get.Flags {
		byName[f.Name] = f
	}

	format, ok := byName["format"]
	if !ok {
		t.Fatal("local flag missing")
	}
	if format.Shorthand != "f" {
		t.Errorf("shorthand: got %q", format.Shorthand)
	}
	if format.Default != "json" {
		t.Errorf("default: got %q", format.Default)
	}
	if format.Type != "string" {
		t.Errorf("type: got %q", format.Type)
	}
	if format.Inherited {
		t.Error("a local flag must not be marked inherited")
	}

	base, ok := byName["base-url"]
	if !ok {
		t.Fatal("inherited persistent flag missing; a caller would think it is unavailable here")
	}
	if !base.Inherited {
		t.Error("a persistent flag from the root must be marked inherited")
	}

	// The interactive sentinel is exactly what an agent must not trip over.
	file, ok := byName["file"]
	if !ok {
		t.Fatal("--file missing")
	}
	if file.NoOptDefVal != "(pick)" {
		t.Errorf("a flag usable with no value must disclose that: got %q", file.NoOptDefVal)
	}
}

func TestCollectFlags_LocalBeforeInherited(t *testing.T) {
	tree := Walk(fixture(), nil, false)
	get, _ := Find(tree, "tool items get")
	var sawInherited bool
	for _, f := range get.Flags {
		if f.Inherited {
			sawInherited = true
			continue
		}
		if sawInherited {
			t.Errorf("local flag %q sorted after an inherited one", f.Name)
		}
	}
}

// TestEnums_CommandScopedBeatsGlobal: cobra cannot model a closed value
// set, so it is supplied here. A flag whose valid values differ between
// two commands must be able to say so.
func TestEnums_CommandScopedBeatsGlobal(t *testing.T) {
	enums := EnumValues{
		"format":                {"json", "cbor"},
		"tool items get|format": {"only-this-one"},
	}
	tree := Walk(fixture(), enums, false)
	get, _ := Find(tree, "tool items get")
	for _, f := range get.Flags {
		if f.Name != "format" {
			continue
		}
		if !slices.Equal(f.Values, []string{"only-this-one"}) {
			t.Errorf("a command-scoped enum should win over the global one, got %v", f.Values)
		}
	}
}

func TestEnums_GlobalAppliesWhenUnscoped(t *testing.T) {
	tree := Walk(fixture(), EnumValues{"format": {"json", "cbor"}}, false)
	get, _ := Find(tree, "tool items get")
	for _, f := range get.Flags {
		if f.Name == "format" && !slices.Equal(f.Values, []string{"json", "cbor"}) {
			t.Errorf("global enum not applied: %v", f.Values)
		}
	}
}

func TestEnums_NilRegistryIsSafe(t *testing.T) {
	tree := Walk(fixture(), nil, false)
	get, _ := Find(tree, "tool items get")
	for _, f := range get.Flags {
		if f.Values != nil {
			t.Errorf("no registry means no values, got %v for %q", f.Values, f.Name)
		}
	}
}

func TestFind_MissAndNestedHit(t *testing.T) {
	tree := Walk(fixture(), nil, false)
	if _, ok := Find(tree, "tool nope"); ok {
		t.Error("Find should miss on an unknown path")
	}
	// A prefix that is not a full path must not match.
	if _, ok := Find(tree, "tool items ge"); ok {
		t.Error("Find must match whole paths, not prefixes")
	}
	if got, ok := Find(tree, "tool items get"); !ok || got.Path != "tool items get" {
		t.Errorf("Find missed a nested command: %v %q", ok, got.Path)
	}
}

// TestWalk_RunnableGroupIsStillAGroup pins the shape this CLI actually has.
// Every namespace here carries a RunE -- that is how a bare `truestamp
// items` prints help and how `truestamp items bogus` becomes an error
// instead of help with exit 0 -- so a group is emphatically not "the
// command that has no Run". Deriving Group from !Runnable() reported
// `"group": false` for all ten namespaces, and the fixture below is the
// case the original test did not have: subcommands AND a Run.
func TestWalk_RunnableGroupIsStillAGroup(t *testing.T) {
	root := &cobra.Command{Use: "tool"}
	group := &cobra.Command{
		Use: "items",
		Run: func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
	}
	group.AddCommand(&cobra.Command{Use: "list", Run: func(*cobra.Command, []string) {}})
	root.AddCommand(group)

	tree := Walk(root, nil, false)
	got := tree.Subcommands[0]
	if !got.Group {
		t.Error("a command with sub-commands is a group even when it is runnable")
	}
	if !got.Runnable {
		t.Error("the group has a Run and must be reported as runnable")
	}
	leaf := got.Subcommands[0]
	if leaf.Group {
		t.Error("a leaf with no sub-commands is not a group")
	}
}
