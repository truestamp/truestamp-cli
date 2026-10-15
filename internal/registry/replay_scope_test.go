// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"
	"testing"
)

// The static test that the replay path stays in the tests. A keyring
// answered from a conformance case's replay file is a value with no bytes
// of its own (SPEC.md section 13.2 (The vectors)), and internal/verifier
// loads such an answer, a KeyringRead with Replayed set, without the
// digest of any bytes read. Only a test may give one. It parses every
// non-test Go file of the module, as rule2_test.go does (go/ast; nothing is
// compiled or run), and refuses:
//
//  1. a file outside internal/vectorcases importing it, the replay adapter,
//     which is test support;
//  2. a dot import of internal/verifier, under which KeyringRead is a bare
//     name the scan cannot tell from another package's;
//  3. a declared type, an alias or a named type, that is KeyringRead or an
//     array, slice, map or pointer built on it, since a literal of it
//     builds a KeyringRead under a name the scan does not know;
//  4. a KeyringRead composite literal that names Replayed or is written by
//     position, which sets every member, its type written or elided as an
//     element, key or value in a literal of an array, slice or map of
//     KeyringRead or of pointers to it, at any depth; any literal whose
//     elided type the scan cannot work out that names Replayed; and a
//     conversion to KeyringRead or
//     to an array, slice, map or pointer built on it;
//  5. an assignment to, or the address of, a selector .Replayed.
//
// What it knows: go/ast sees syntax, not types. KeyringRead is the bare
// name inside internal/verifier and X.KeyringRead for each name X a file
// imports internal/verifier by; Replayed is any member of that name, of
// any type. It reads every non-test file, whatever its build constraints.
// What it does not see: a KeyringRead named through a type parameter, or
// a literal of a generic type instantiated with it, such as
// list[KeyringRead]{{nil, true, v}} for a type list[T any] []T; an
// assignment to a parenthesized selector, such as (r.Replayed) = true, or a
// range clause that assigns to .Replayed; and a Replayed set by a decoder
// such as encoding/json, by reflect or by unsafe. It is a check against a mistake, not a proof that no non-test
// code can give a replayed answer. A planted file of each refused shape
// shows each check firing.

const (
	vectorcasesPath = "github.com/truestamp/truestamp-cli/internal/vectorcases"
	verifierPath    = "github.com/truestamp/truestamp-cli/internal/verifier"
	vectorcasesDir  = "internal/vectorcases"
	verifierDir     = "internal/verifier"
)

// replayFindings are the sites in files that would let a non-test file
// give a replayed keyring answer.
func replayFindings(fset *token.FileSet, files []goFile) []finding {
	var out []finding
	add := func(f goFile, pos token.Pos, format string, args ...any) {
		out = append(out, finding{rel: f.rel, line: fset.Position(pos).Line, what: fmt.Sprintf(format, args...)})
	}
	for _, f := range files {
		dir := path.Dir(f.rel)
		verifierNames := map[string]bool{}
		for _, spec := range f.file.Imports {
			p, _ := strconv.Unquote(spec.Path.Value)
			switch {
			case p == vectorcasesPath && dir != vectorcasesDir:
				add(f, spec.Pos(), "imports internal/vectorcases, the replay adapter, which only tests may use")
			case p == verifierPath && spec.Name != nil && spec.Name.Name == ".":
				add(f, spec.Pos(), "dot-imports internal/verifier, so KeyringRead is a bare name")
			case p == verifierPath && spec.Name != nil:
				verifierNames[spec.Name.Name] = true
			case p == verifierPath:
				verifierNames[path.Base(p)] = true
			}
		}
		isKeyringRead := func(expr ast.Expr) bool {
			switch e := ast.Unparen(expr).(type) {
			case *ast.Ident:
				return dir == verifierDir && e.Name == "KeyringRead"
			case *ast.SelectorExpr:
				x, ok := e.X.(*ast.Ident)
				return ok && verifierNames[x.Name] && e.Sel.Name == "KeyringRead"
			}
			return false
		}
		var builtOn func(ast.Expr) bool
		builtOn = func(expr ast.Expr) bool {
			switch e := ast.Unparen(expr).(type) {
			case *ast.ArrayType:
				return builtOn(e.Elt)
			case *ast.MapType:
				return builtOn(e.Key) || builtOn(e.Value)
			case *ast.StarExpr:
				return builtOn(e.X)
			}
			return isKeyringRead(expr)
		}
		// elided holds the type of each composite literal whose type is
		// elided as an element, key or value of a literal visited before it.
		elided := map[*ast.CompositeLit]ast.Expr{}
		inferElided := func(elt, typ ast.Expr) {
			lit, ok := elt.(*ast.CompositeLit)
			if !ok || lit.Type != nil || typ == nil {
				return
			}
			if star, ok := ast.Unparen(typ).(*ast.StarExpr); ok {
				typ = star.X
			}
			elided[lit] = typ
		}
		ast.Inspect(f.file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.TypeSpec:
				if builtOn(n.Type) {
					add(f, n.Pos(), "declares %s, a type built on KeyringRead", n.Name.Name)
				}
			case *ast.CallExpr:
				if len(n.Args) == 1 && builtOn(n.Fun) {
					add(f, n.Pos(), "converts a value to KeyringRead or a type built on it")
				}
			case *ast.CompositeLit:
				typ := n.Type
				if typ == nil {
					typ = elided[n]
				}
				var keyType, eltType ast.Expr
				switch t := ast.Unparen(typ).(type) {
				case *ast.ArrayType:
					eltType = t.Elt
				case *ast.MapType:
					keyType, eltType = t.Key, t.Value
				}
				for _, elt := range n.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						inferElided(kv.Key, keyType)
						inferElided(kv.Value, eltType)
					} else {
						inferElided(elt, eltType)
					}
				}
				if typ != nil && !isKeyringRead(typ) {
					return true
				}
				for i, elt := range n.Elts {
					kv, keyed := elt.(*ast.KeyValueExpr)
					if !keyed {
						if i == 0 && typ != nil {
							add(f, n.Pos(), "builds a KeyringRead by position, which sets Replayed")
						}
						continue
					}
					if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Replayed" {
						add(f, kv.Pos(), "sets Replayed in a composite literal")
					}
				}
			case *ast.AssignStmt:
				for _, lhs := range n.Lhs {
					if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "Replayed" {
						add(f, sel.Pos(), "assigns Replayed")
					}
				}
			case *ast.UnaryExpr:
				if sel, ok := ast.Unparen(n.X).(*ast.SelectorExpr); ok && n.Op == token.AND && sel.Sel.Name == "Replayed" {
					add(f, n.Pos(), "takes the address of Replayed")
				}
			}
			return true
		})
	}
	return out
}

// No non-test file of the module imports the replay adapter or gives a
// replayed keyring answer. The scan is not vacuous: it reads the adapter,
// the KeyringRead declaration and readOnline, which reads Replayed. Mutants
// killed, each a planted non-test file: one of cmd importing
// internal/vectorcases; a reader of internal/verifier returning
// KeyringRead{Replayed: true}; an alias of KeyringRead in internal/verifier
// whose literal sets Replayed; a []KeyringRead in internal/verifier whose
// element, its type elided, is written by position; a dot import of
// internal/verifier building KeyringRead{Replayed: true}; and, in cmd, a
// conversion to KeyringRead, the address of a KeyringRead's Replayed, and a
// named slice of KeyringRead whose element is written by position.
func TestReplayScope_NoNonTestFileGivesAReplayedKeyring(t *testing.T) {
	fset, files := parseModule(t, repoRoot)
	if findings := replayFindings(fset, files); len(findings) > 0 {
		var lines []string
		for _, f := range findings {
			lines = append(lines, f.String())
		}
		t.Fatalf("the replay path outside the tests:\n  %s", strings.Join(lines, "\n  "))
	}
	seen := map[string]bool{}
	for _, f := range files {
		seen[f.rel] = true
	}
	for _, want := range []string{"internal/vectorcases/replay.go", "internal/verifier/config.go", "internal/verifier/gate.go", "cmd/truestamp/main.go"} {
		if !seen[want] {
			t.Errorf("the scan did not read %s", want)
		}
	}
}

// replayPlanted are files of each shape the scan refuses, and of shapes it
// allows, with the findings each line gives.
var replayPlanted = map[string]string{
	"cmd/planted.go": `package cmd

import (
	vc "github.com/truestamp/truestamp-cli/internal/vectorcases" // want "imports internal/vectorcases"
	tv "github.com/truestamp/truestamp-cli/internal/verifier"
	again "github.com/truestamp/truestamp-cli/internal/verifier"
)

type plantedRead = tv.KeyringRead // want "declares plantedRead"

type plantedReads []*tv.KeyringRead // want "declares plantedReads"

func planted(v any) []tv.KeyringRead {
	read := tv.KeyringRead{Replayed: true, Value: v} // want "sets Replayed"
	pointer := &tv.KeyringRead{Value: v, Replayed: false} // want "sets Replayed"
	read.Replayed = true // want "assigns Replayed"
	flag := &read.Replayed // want "address of Replayed"
	byPosition := tv.KeyringRead{nil, true, v} // want "by position"
	underAnother := again.KeyringRead{nil, true, v} // want "by position"
	inSlice := []tv.KeyringRead{{nil, true, v}} // want "by position"
	inMap := map[string]*tv.KeyringRead{"read": {nil, true, v}} // want "by position"
	nested := [][]tv.KeyringRead{{{nil, true, v}}} // want "by position"
	converted := tv.KeyringRead(struct { // want "converts a value"
		Bytes    []byte
		Replayed bool
		Value    any
	}{nil, true, v})
	viaPointer := (*tv.KeyringRead)(&struct { // want "converts a value"
		Bytes    []byte
		Replayed bool
		Value    any
	}{nil, true, v})
	_, _, _, _, _, _, _, _, _, _ = pointer, flag, byPosition, underAnother, inSlice, inMap, nested, converted, viaPointer, vc.ErrNoAnswer
	return []tv.KeyringRead{{Replayed: true}} // want "sets Replayed"
}

type otherReads []struct{ Replayed bool }

func allowed(data []byte, reader tv.KeyringReader) tv.KeyringRead {
	read := tv.KeyringRead{Bytes: data}
	if read.Replayed {
		return tv.KeyringRead{}
	}
	other := struct{ Replayed bool }{true}
	others := otherReads{{true}}
	elided := []struct {
		Bytes    []byte
		Replayed bool
	}{{nil, true}}
	ints := [][]int{{1, 2}}
	_, _, _, _ = other, others, elided, ints
	_ = tv.KeyringReader(reader)
	return read
}
`,
	"internal/verifier/planted.go": `package verifier

type plantedRead = KeyringRead // want "declares plantedRead"

type plantedNamed KeyringRead // want "declares plantedNamed"

type plantedByName map[string]KeyringRead // want "declares plantedByName"

func planted(v any) (KeyringRead, KeyringRead, []KeyringRead) {
	var read KeyringRead
	read.Bytes, read.Replayed = nil, true // want "assigns Replayed"
	return KeyringRead{Replayed: true, Value: v}, read, []KeyringRead{{nil, true, v}} // want "sets Replayed" "by position"
}

func allowed(data []byte) KeyringRead {
	return KeyringRead{Bytes: data}
}
`,
	"internal/vectorcases/planted.go": `package vectorcases

import _ "github.com/truestamp/truestamp-cli/internal/vectorcases"
`,
	"internal/other/planted.go": `package other

type KeyringRead struct{ Replayed bool }

type reads []KeyringRead

func allowed() []KeyringRead {
	return []KeyringRead{KeyringRead{Replayed: true}, {true}}
}
`,
	"internal/dotted/planted.go": `package dotted

import . "github.com/truestamp/truestamp-cli/internal/verifier" // want "dot-imports internal/verifier"

func planted() KeyringRead {
	return KeyringRead{Replayed: true}
}
`,
}

// Each planted shape gives exactly the findings its line names, and the
// allowed shapes give none: a KeyringRead with Bytes alone, a read of
// Replayed, another struct's Replayed member set by position (its type
// written, or elided in a slice of it or in a declared slice of it), a
// slice of slices of int, a conversion to KeyringReader, an import of the
// adapter inside internal/vectorcases, and a KeyringRead of another
// package, with a slice of it declared and built. Mutants killed: check 1
// left out (the planted cmd import passes); check 1 made inside
// internal/vectorcases too; check 2 left out (the dot import passes);
// check 3 left out (the aliases and named types pass) or limited to
// KeyringRead itself (the slice and map types pass); check 4 limited to
// keyed literals (the positional ones pass), to written types (the elided
// one passes, keyed or by position), to the first name a file imports
// internal/verifier by (the second name passes), or matching KeyringRead
// in any package, its conversion left out (the conversions pass) or
// limited to KeyringRead itself (the pointer conversion passes); check 5's
// assignment or its address left out.
func TestReplayScope_FindsEveryPlantedShape(t *testing.T) {
	fset, files, want := parsePlanted(t, replayPlanted, "want")
	if len(want) != 21 {
		t.Fatalf("%d planted lines, want 21", len(want))
	}
	assertFindings(t, replayFindings(fset, files), want)
}
