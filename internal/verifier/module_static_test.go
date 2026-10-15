// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
)

// The static tests of this package that read the whole module, the
// constructions test (constructions_static_test.go) and the pins seam's
// (policy_test.go), parse every non-test Go file of the module, skipping
// what internal/registry/rule2_test.go's parseModule skips, and type-check
// each package with go/types, from the standard library: nothing is
// compiled or run. Each package of the module is checked from its own
// files, and a package it imports from the module is the one checked here,
// so a constant's value and the object a selector names are known across
// the module. A package from outside the module is given as an empty
// package, since no member name or pin of the verifier comes from one: an
// expression that needs it is invalid, which the checker reports and these
// tests set aside, and every other expression is still checked.

// modulePath is the module's path, from go.mod.
const modulePath = "github.com/truestamp/truestamp-cli"

// moduleRoot is the module root, from this package's directory.
var moduleRoot = filepath.Join("..", "..")

// moduleSkippedDirs are the directories at the module root that hold no
// package of the module, as rule2_test.go's skippedDirs names them.
var moduleSkippedDirs = map[string]bool{"build": true, "dist": true, "tmp": true, "coverage": true, "testdata": true, "vendor": true}

// moduleFile is one parsed non-test Go file, by its slash path from the
// module root.
type moduleFile struct {
	rel  string
	file *ast.File
}

// typedModule is the module's files, parsed and type-checked.
type typedModule struct {
	fset  *token.FileSet
	files []moduleFile
	info  *types.Info
	// packages are the checked packages, by import path.
	packages map[string]*types.Package
}

// parseModuleFiles parses every non-test Go file of the module at root,
// sorted by path. A directory whose name begins with "." is skipped at any
// depth, as is a nested module, and moduleSkippedDirs at the root only.
func parseModuleFiles(root string) (*token.FileSet, []moduleFile, error) {
	fset := token.NewFileSet()
	var files []moduleFile
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
			if strings.HasPrefix(entry.Name(), ".") || moduleSkippedDirs[rel] {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(name, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		files = append(files, moduleFile{rel: rel, file: file})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	return fset, files, err
}

// importerFunc is a types.Importer.
type importerFunc func(importPath string) (*types.Package, error)

func (f importerFunc) Import(importPath string) (*types.Package, error) { return f(importPath) }

// typeCheck type-checks files, the module's, package by package: the files
// of one directory and one package name are a package, whose import path
// is the module's path and the directory. Type errors are set aside (see
// the doc above).
func typeCheck(fset *token.FileSet, files []moduleFile) *typedModule {
	m := &typedModule{fset: fset, files: files, packages: map[string]*types.Package{}, info: &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}}
	groups := map[string][]*ast.File{}
	names := map[string][]string{}
	for _, f := range files {
		dir, name := path.Dir(f.rel), f.file.Name.Name
		key := dir + " " + name
		if groups[key] == nil {
			names[dir] = append(names[dir], name)
		}
		groups[key] = append(groups[key], f.file)
	}
	checking := map[string]bool{}
	var check func(key, importPath string) *types.Package
	external := map[string]*types.Package{}
	importer := importerFunc(func(importPath string) (*types.Package, error) {
		if dir, ok := strings.CutPrefix(importPath, modulePath+"/"); ok && len(names[dir]) > 0 {
			candidates := slices.Clone(names[dir])
			slices.Sort(candidates)
			name := candidates[0]
			if i := slices.IndexFunc(candidates, func(n string) bool { return n != "main" }); i >= 0 {
				name = candidates[i]
			}
			if checking[importPath] {
				return nil, fmt.Errorf("import cycle through %s", importPath)
			}
			return check(dir+" "+name, importPath), nil
		}
		if p, ok := external[importPath]; ok {
			return p, nil
		}
		p := types.NewPackage(importPath, path.Base(importPath))
		p.MarkComplete()
		external[importPath] = p
		return p, nil
	})
	check = func(key, importPath string) *types.Package {
		if p, ok := m.packages[key]; ok {
			return p
		}
		checking[importPath] = true
		config := types.Config{Importer: importer, Error: func(error) {}}
		p, _ := config.Check(importPath, fset, groups[key], m.info)
		checking[importPath] = false
		m.packages[key] = p
		return p
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		dir, _, _ := strings.Cut(key, " ")
		check(key, modulePath+"/"+dir)
	}
	byPath := map[string]*types.Package{}
	for _, p := range m.packages {
		if p != nil && p.Name() != "main" {
			byPath[p.Path()] = p
		}
	}
	m.packages = byPath
	return m
}

// loadTypedModule parses and type-checks the module once for every test
// of the package.
var loadTypedModule = sync.OnceValues(func() (*typedModule, error) {
	fset, files, err := parseModuleFiles(moduleRoot)
	if err != nil {
		return nil, err
	}
	return typeCheck(fset, files), nil
})

// typedModuleOf returns the module, parsed and type-checked.
func typedModuleOf(t *testing.T) *typedModule {
	t.Helper()
	m, err := loadTypedModule()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// typedSources parses and type-checks sources, planted files by their
// slash path from a module root, as typeCheck does the module.
func typedSources(t *testing.T, sources map[string]string) *typedModule {
	t.Helper()
	fset := token.NewFileSet()
	var files []moduleFile
	for rel, source := range sources {
		file, err := parser.ParseFile(fset, rel, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		files = append(files, moduleFile{rel: rel, file: file})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	return typeCheck(fset, files)
}

// enclosingFunction returns the name of the top-level function or method
// pos lies in, "" outside every function.
func enclosingFunction(file *ast.File, pos token.Pos) string {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Pos() <= pos && pos < fn.End() {
			return fn.Name.Name
		}
	}
	return ""
}
