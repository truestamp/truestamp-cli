// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package introspect walks a live cobra command tree and renders it as
// data. It is what backs `truestamp schema get commands`.
//
// It reads the tree rather than a hand-maintained list on purpose: a
// description that is generated from the thing it describes cannot drift
// from it. That property is why the generated document can replace the
// golden help-text fixture as the reviewable artifact for a tree change —
// the diff of the schema *is* the design review.
package introspect

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Flag describes one flag as an agent needs to see it: enough to
// construct a valid invocation without reading prose.
type Flag struct {
	Name      string   `json:"name"`
	Shorthand string   `json:"shorthand,omitempty"`
	Usage     string   `json:"usage"`
	Type      string   `json:"type"`
	Default   string   `json:"default,omitempty"`
	Values    []string `json:"values,omitempty"`
	Inherited bool     `json:"inherited"`
	Hidden    bool     `json:"hidden,omitempty"`
	// NoOptDefVal records that the flag may be passed without a value,
	// and what it means when it is. The CLI uses this for the interactive
	// sentinels (`--file` with no path opens a picker), which is exactly
	// the kind of thing an agent must not trip over.
	NoOptDefVal string `json:"no_opt_default,omitempty"`
}

// Command describes one node of the tree.
type Command struct {
	Path        string    `json:"path"`
	Use         string    `json:"use"`
	Short       string    `json:"short"`
	Long        string    `json:"long,omitempty"`
	Example     string    `json:"example,omitempty"`
	Group       bool      `json:"group"`
	Runnable    bool      `json:"runnable"`
	Hidden      bool      `json:"hidden,omitempty"`
	Deprecated  string    `json:"deprecated,omitempty"`
	Aliases     []string  `json:"aliases,omitempty"`
	Flags       []Flag    `json:"flags,omitempty"`
	Subcommands []Command `json:"subcommands,omitempty"`
}

// EnumValues supplies the closed value set for a flag, keyed by
// "<command path>|<flag name>" or by "<flag name>" for a flag whose
// values are the same wherever it appears. Cobra does not model closed
// enums, so this is the one piece of information the tree cannot supply
// about itself; it is registered by the CLI and passed in.
type EnumValues map[string][]string

// Walk renders root and everything under it. Hidden commands are included
// only when includeHidden is set: they are part of the tree, but they are
// not part of the interface, and `schema get commands` describes the
// interface by default.
func Walk(root *cobra.Command, enums EnumValues, includeHidden bool) Command {
	return walk(root, root.Name(), enums, includeHidden)
}

func walk(c *cobra.Command, path string, enums EnumValues, includeHidden bool) Command {
	out := Command{
		Path:     path,
		Use:      c.Use,
		Short:    c.Short,
		Long:     c.Long,
		Example:  c.Example,
		Runnable: c.Runnable(),
		// A command with sub-commands is a namespace, full stop. This used
		// to also require !Runnable(), which silently stopped being true:
		// every group in this CLI carries a RunE so that a bare group can
		// print its help and an unknown sub-command can be an error rather
		// than help-with-exit-0. That made `group` false for `auth`,
		// `items`, `proofs` and every other namespace -- the exact opposite
		// of what the field exists to tell a reader.
		Group:      c.HasSubCommands(),
		Hidden:     c.Hidden,
		Deprecated: c.Deprecated,
		Aliases:    c.Aliases,
		Flags:      collectFlags(c, path, enums),
	}
	for _, sub := range c.Commands() {
		if sub.Hidden && !includeHidden {
			continue
		}
		// Cobra's own help/completion scaffolding is not part of the
		// interface this document describes.
		if sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		out.Subcommands = append(out.Subcommands, walk(sub, path+" "+sub.Name(), enums, includeHidden))
	}
	return out
}

func collectFlags(c *cobra.Command, path string, enums EnumValues) []Flag {
	seen := map[string]bool{}
	var out []Flag

	add := func(f *pflag.Flag, inherited bool) {
		if seen[f.Name] {
			return
		}
		seen[f.Name] = true
		out = append(out, Flag{
			Name:        f.Name,
			Shorthand:   f.Shorthand,
			Usage:       f.Usage,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Values:      LookupEnum(enums, path, f.Name),
			Inherited:   inherited,
			Hidden:      f.Hidden,
			NoOptDefVal: f.NoOptDefVal,
		})
	}

	c.LocalFlags().VisitAll(func(f *pflag.Flag) { add(f, false) })
	c.InheritedFlags().VisitAll(func(f *pflag.Flag) { add(f, true) })

	sort.Slice(out, func(i, j int) bool {
		if out[i].Inherited != out[j].Inherited {
			return !out[i].Inherited
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// lookupEnum prefers a command-scoped entry over a global one, so a flag
// whose valid values differ between two commands can say so.
// LookupEnum resolves a flag's closed value set: the path-scoped key first,
// then the bare flag name. Exported so shell completion and `schema get
// commands` resolve a flag the same way.
func LookupEnum(enums EnumValues, path, flag string) []string {
	if enums == nil {
		return nil
	}
	if v, ok := enums[path+"|"+flag]; ok {
		return v
	}
	return enums[flag]
}

// Paths returns every command path in the tree, depth first. Used to
// assert the round-trip property: everything listed is invocable, and
// everything invocable is listed.
func Paths(c Command) []string {
	out := []string{c.Path}
	for _, sub := range c.Subcommands {
		out = append(out, Paths(sub)...)
	}
	return out
}

// Find returns the node at the given space-separated path, or false.
func Find(c Command, path string) (Command, bool) {
	if c.Path == path {
		return c, true
	}
	for _, sub := range c.Subcommands {
		if strings.HasPrefix(path, sub.Path) {
			if found, ok := Find(sub, path); ok {
				return found, true
			}
		}
	}
	return Command{}, false
}
