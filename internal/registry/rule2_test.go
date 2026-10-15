// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The static test of SPEC.md section 10.6, rule 2: prefix bytes and member
// sets come only from compiled constants, never from a document, and the
// leaf and node bytes 0x00 and 0x01 are used only inside the one walk of
// section 5.3. It parses every Go file of the module but its tests (go/ast;
// nothing is compiled or run), as the reference's derive_test.exs and the
// app's ast-grep rule proof-prefixes-from-spec do. Checks 1 to 5 and 9 read
// every file of the module:
//
//  1. a file other than internal/registry/walk.go importing
//     github.com/transparency-dev/merkle, whose RFC 6962 hasher applies
//     0x00 and 0x01; and, in walk.go, a use of it outside the function
//     walk;
//  2. a non-test file importing golang.org/x/mod/sumdb/tlog, which only
//     the tests use, to cross-check the walk;
//  3. the registry's MerkleLeaf or MerkleNode named outside
//     internal/registry/prefixes.go, the table;
//  4. a conversion to registry.Construction or registry.B64UClass, by
//     which a number a document carries could name a construction or a
//     class;
//  5. a reference to specdata.PrefixByte, which reads a byte as spec/
//     writes it, called or not, outside a test and its own declaration;
//  9. a package that imports internal/registry and is not one of
//     scannedPackages, so each package of the new verifier is scanned from
//     its first commit.
//
// Checks 6, 8 and 11 read only the packages of scannedPackages, the new
// verifier's, and checks 7 and 10 only those of them, or of their files,
// it marks: older code, such as internal/tscrypto's walk, still applies
// 0x00 and 0x01 itself. Check 12 reads the module, and holds the packages
// of the module a scanned package may import, directly or through another
// of them (importableDirs), to the sites it names.
//
//  6. crypto/sha256 used outside the registry's digest, the one function
//     that applies a prefix byte, and outside contentDigests, the two
//     functions of the new verifier that make a content digest, a SHA-256
//     with no prefix byte (SPEC.md section 4.1): keyring.Digest,
//     trust_root's digest of a keyring's bytes as read (section 8.3), and
//     SHAPE's shapeConstructions, check 6's d of a constructions text's JCS
//     (sections 7.2 and 10.2); in internal/registry, digest given a
//     construction that is not a registry constant (or RecordHash's
//     checked kind), or the leaf or node; a hex integer literal, or a byte
//     slice or array literal holding a literal, outside the table; and an
//     import by ".", which hides its uses from these checks;
//  7. in the registry, and in DERIVE and its layout, the files
//     internal/verifier/derive*.go and internal/verifier/layout.go, as the
//     reference's derive_test.exs scans them (never SHAPE, the schema's
//     checker, the gate, strictjson or jcs, which read a document's members
//     by design), a member set read from a value: the maps or reflect
//     packages, and a string literal named like a registry member ("byte",
//     "prefix", "prefixes", "members", "fields"), as part 5 of the app's
//     rule bans; in the registry, which reads its own compiled tables by a
//     key, a range over a map parameter, over a map a type assertion gives
//     or over a type switch's binding in a map case, and such a map indexed
//     by a name that is not a string literal; and in DERIVE and its layout,
//     typed by go/types from the standard library (memberSetTypes), a range
//     over a value of map type and a read of one by a key that is not a
//     constant, whatever yields the map (a parameter, a variable however it
//     is bound, a field, a call, a type assertion or a type switch's
//     binding), and a range or such a read over a value the checker cannot
//     type, which only a value from a package outside the module has until
//     a declared type is written for it. The one index allowed is the
//     layout's pointer read, valueAt's index of an object by its pointer
//     segment's name (pointerReads). The reference's lint,
//     pairs_from_document, flags a range or a read over a bound variable
//     other than a position or a node; reading the type of every
//     expression, this check also refuses a map a field or a call yields.
//     Its known limit is its file scope, the reference's: a helper DERIVE
//     calls that is declared in a file of internal/verifier outside
//     derive*.go and layout.go is not read;
//  8. an import that is not on the package's list in scannedPackages, so
//     that nothing in internal/registry hashes but digest and the walk, and
//     nothing in another package of the new verifier hashes but the two
//     content digests: no list holds crypto/sha256 but the registry's and
//     those of the content digests' two packages, internal/verifier and
//     internal/verifier/keyring, where check 6 confines it to those two
//     functions; and none holds another hash package (crypto, crypto/sha1,
//     crypto/sha512, crypto/md5, crypto/hmac, crypto/sha3, hash or a hash/
//     package), internal/tscrypto or internal/hashing, whose generic hashers
//     and prefix tables the other checks do not see. Another crypto/
//     package, such as crypto/ed25519, may be listed;
// 10. in a package marked to print nothing (internal/verifier/source, which
//     sends a Bitcoin Core node's RPC credentials and reads what the node
//     answers), a call of fmt's Print, Printf, Println, Fprint, Fprintf or
//     Fprintln, or of the print or println builtins; its import list,
//     which holds neither log, log/slog nor os, closes the other ways out;
// 11. in a scanned package other than the registry, the kind of a
//     registry.RecordHash or registry.RecordHashHex call that is not a
//     registry constant (other than the leaf and node) or a parameter of
//     the enclosing function declared as a registry.Construction; such a
//     parameter given, at a call of the package's own function, anything
//     but a registry constant or such a parameter; and the type
//     registry.Construction written anywhere but as the type of a
//     function's parameter, so no kind is held in a variable, a field or a
//     table, and each kind DERIVE hashes under is a constant written at a
//     call;
// 12. a named digest, one of contentDigests or importedDigests, referred
//     to anywhere but at its one call (digestCalls): keyring.Digest only in
//     keyring.go's Load, shapeConstructions only in shape.go's Shape,
//     specdata's acceptedDigests only in constructions.go's
//     mustAcceptedDigests and jcs's placeholderFor only in jcs.go's
//     Canonicalize; and, in a package of the module a scanned package may
//     import that is not itself scanned (check 6 holds those), crypto/sha256
//     used anywhere but in that package's importedDigests, at the top level
//     or in any other function, exported or not, and an import of a hash
//     package check 8 lists (hasher) other than crypto/sha256, of
//     crypto/sha256 by ".", or of a dependency outside the module that is
//     not on importableDependencies, so that no wrapper, variable or other
//     hash a verifier package could call hashes outside the registry. It
//     reads syntax: a named digest that hands out a hash function as a
//     value is beyond it.
//
// Checks other than 7 read syntax, not data flow, so they cannot see a byte
// or a name that reaches these sites through a variable or a call; a
// planted file of each shape shows each check firing.

const (
	registryPath   = "github.com/truestamp/truestamp-cli/internal/registry"
	specdataPath   = "github.com/truestamp/truestamp-cli/internal/specdata"
	jcsPath        = "github.com/truestamp/truestamp-cli/internal/jcs"
	strictjsonPath = "github.com/truestamp/truestamp-cli/internal/strictjson"
	merklePath     = "github.com/transparency-dev/merkle"
	tlogPath       = "golang.org/x/mod/sumdb/tlog"
	registryDir    = "internal/registry"
	specdataDir    = "internal/specdata"
	walkFile       = "internal/registry/walk.go"
	tableFile      = "internal/registry/prefixes.go"
)

// scannedPackage is how the scan reads one package of the new verifier.
type scannedPackage struct {
	// imports are the only packages its non-test files may import (check
	// 8).
	imports []string
	// memberSets says check 7 reads it.
	memberSets bool
	// memberSetFiles are the files check 7 reads in a package it does not
	// read whole, as path.Match patterns of their names.
	memberSetFiles []string
	// printsNothing says check 10 reads it.
	printsNothing bool
}

// scannedPackages are the packages of the new verifier that checks 6 to 8
// read, by their slash path from the module root. A package of the new
// verifier joins the list in the commit that adds it; check 9 fails one
// that imports internal/registry without being on it.
var scannedPackages = map[string]scannedPackage{
	registryDir: {imports: registryImports, memberSets: true},
	"internal/verifier": {memberSetFiles: []string{"derive*.go", "layout.go"}, imports: []string{
		"cmp", "context", "crypto/sha256", "encoding/hex", "errors", "fmt", "math", "net", "net/url", "regexp", "slices", "strconv", "strings", "sync", "time",
		"unicode/utf16", "unicode/utf8",
		"github.com/spf13/pflag",
		"github.com/truestamp/truestamp-cli/internal/config",
		"github.com/truestamp/truestamp-cli/internal/httpclient",
		jcsPath, registryPath, specdataPath, strictjsonPath,
		"github.com/truestamp/truestamp-cli/internal/verifier/ed25519",
		"github.com/truestamp/truestamp-cli/internal/verifier/keyring",
		"github.com/truestamp/truestamp-cli/internal/verifier/source",
	}},
	"internal/verifier/source": {imports: []string{
		"bytes", "context", "encoding/base64", "errors", "fmt", "io", "net/http", "net/url", "strings",
		"github.com/truestamp/truestamp-cli/internal/httpclient",
		strictjsonPath,
	}, printsNothing: true},
	"internal/verifier/keyring": {imports: []string{
		"crypto/sha256", "encoding/hex", "errors", "fmt", "regexp", "slices",
		registryPath, specdataPath, strictjsonPath,
		"github.com/truestamp/truestamp-cli/internal/verifier/ed25519",
	}},
	"internal/verifier/ed25519": {imports: []string{"crypto/ed25519", "errors", "math/big", specdataPath}},
}

// contentDigest is a function of the new verifier outside the registry
// that may use crypto/sha256, by its file and its name.
type contentDigest struct{ file, function string }

// contentDigests are the two content digests, SHA-256 with no prefix byte,
// which SPEC.md section 4.1 puts outside the registry: keyring.Digest,
// trust_root's digest d of a keyring's bytes as read (section 8.3), and
// shapeConstructions, SHAPE's check 6, whose d is the SHA-256 of a
// constructions text's JCS (sections 7.2 and 10.2). Check 6 allows the use
// in each of those functions of its file, and nowhere else in its
// package, and check 8 lets their two packages alone list crypto/sha256
// beside the registry.
var contentDigests = []contentDigest{
	{"internal/verifier/keyring/digest.go", "Digest"},
	{"internal/verifier/shape.go", "shapeConstructions"},
}

// inContentDigest reports whether pos, in f, lies in one of
// contentDigests.
func inContentDigest(f goFile, pos token.Pos) bool {
	function := enclosingFunc(f.file, pos)
	return slices.Contains(contentDigests, contentDigest{f.rel, function})
}

// importedDigests are the functions that may use crypto/sha256 in the
// packages of the module a scanned package may import that are not
// scanned (check 12), by their file and their name: specdata's
// acceptedDigests, which checks that each embedded constructions text is
// named by the SHA-256 of its bytes (SPEC.md section 7.2), and jcs's
// placeholderFor, the token Canonicalize splices in for an oversized
// integer and swaps back out, which no hash of the verifier reads.
var importedDigests = []contentDigest{
	{"internal/specdata/constructions.go", "acceptedDigests"},
	{"internal/jcs/jcs.go", "placeholderFor"},
}

// namedDigests are the functions outside the registry check 12 names:
// contentDigests and importedDigests.
var namedDigests = slices.Concat(contentDigests, importedDigests)

// digestCalls are the one call each of namedDigests has, by the file and
// the function it lies in (check 12).
var digestCalls = map[contentDigest]contentDigest{
	contentDigests[0]:  {"internal/verifier/keyring/keyring.go", "Load"},
	contentDigests[1]:  {"internal/verifier/shape.go", "Shape"},
	importedDigests[0]: {"internal/specdata/constructions.go", "mustAcceptedDigests"},
	importedDigests[1]: {"internal/jcs/jcs.go", "Canonicalize"},
}

// pointerRead is a function whose index of an object by a name that is not
// a literal check 7 allows, by its file and its name.
type pointerRead struct{ file, function string }

// pointerReads are the layout's pointer read: valueAt, which indexes an
// object by its pointer segment's name.
var pointerReads = []pointerRead{{"internal/verifier/layout.go", "valueAt"}}

// readsMemberSets reports whether check 7 reads the file rel of pkg: every
// file of a package marked memberSets, and the files of memberSetFiles.
func readsMemberSets(pkg scannedPackage, rel string) bool {
	if pkg.memberSets {
		return true
	}
	return slices.ContainsFunc(pkg.memberSetFiles, func(pattern string) bool {
		matched, err := path.Match(pattern, path.Base(rel))
		return err == nil && matched
	})
}

// contentDigestDir reports whether dir holds one of contentDigests.
func contentDigestDir(dir string) bool {
	return slices.ContainsFunc(contentDigests, func(d contentDigest) bool { return path.Dir(d.file) == dir })
}

// registryImports are the only packages internal/registry's non-test files
// may import (check 8). crypto/sha256 is used only in digest (check 6), and
// the Merkle library only in walk (check 1).
var registryImports = []string{
	"crypto/sha256",
	merklePath + "/proof",
	merklePath + "/rfc6962",
	"encoding/base64",
	"encoding/hex",
	"errors",
	"fmt",
	"regexp",
	"strconv",
	jcsPath,
	specdataPath,
	strictjsonPath,
}

// memberNames are the string literals check 7 bans in the scanned
// packages: the names of a registry row's members and of member sets.
var memberNames = []string{"byte", "prefix", "prefixes", "members", "fields"}

// goFile is one parsed file, by its slash path from the module root.
type goFile struct {
	rel  string
	file *ast.File
}

// finding is one site the rule refuses.
type finding struct {
	rel  string
	line int
	what string
}

func (f finding) String() string { return fmt.Sprintf("%s:%d: %s", f.rel, f.line, f.what) }

// skippedDirs are the directories at the module root that hold no package
// of the module: those the review copy leaves out, and test data. They are
// matched at the root only, so a package's own tmp or build directory is
// read.
var skippedDirs = map[string]bool{"build": true, "dist": true, "tmp": true, "coverage": true, "testdata": true, "vendor": true}

// parseModule parses every non-test Go file of the module at root. A
// directory whose name begins with "." is skipped at any depth, since the
// go tool builds no package there, and so is a nested module.
func parseModule(t *testing.T, root string) (*token.FileSet, []goFile) {
	t.Helper()
	fset := token.NewFileSet()
	var files []goFile
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
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		files = append(files, goFile{rel: rel, file: file})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fset, files
}

// importName is the name file refers to the package at importPath by, or
// "" when it does not import it.
func importName(file *ast.File, importPath string) string {
	for _, spec := range file.Imports {
		if p, _ := strconv.Unquote(spec.Path.Value); p == importPath {
			if spec.Name != nil {
				return spec.Name.Name
			}
			return path.Base(p)
		}
	}
	return ""
}

// importNames are the names file refers to the package at importPath by,
// one for each import of it.
func importNames(file *ast.File, importPath string) []string {
	var names []string
	for _, spec := range file.Imports {
		if p, _ := strconv.Unquote(spec.Path.Value); p == importPath {
			if spec.Name != nil {
				names = append(names, spec.Name.Name)
			} else {
				names = append(names, path.Base(p))
			}
		}
	}
	return names
}

// printFuncs are fmt's functions that write to an output (check 10).
var printFuncs = []string{"Print", "Printf", "Println", "Fprint", "Fprintf", "Fprintln"}

// constructionNames are the constants of the const block whose first
// constant has the type Construction, from the registry's files.
func constructionNames(files []goFile) map[string]bool {
	names := map[string]bool{}
	for _, f := range files {
		if path.Dir(f.rel) != registryDir {
			continue
		}
		for _, decl := range f.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST || len(gen.Specs) == 0 {
				continue
			}
			first := gen.Specs[0].(*ast.ValueSpec)
			if typ, ok := first.Type.(*ast.Ident); !ok || typ.Name != "Construction" {
				continue
			}
			for _, spec := range gen.Specs {
				for _, name := range spec.(*ast.ValueSpec).Names {
					names[name.Name] = true
				}
			}
		}
	}
	return names
}

// prefixByteDecl is the name in internal/specdata's declaration of the
// function PrefixByte, the one reference to it check 5 allows, or nil.
func prefixByteDecl(files []goFile) *ast.Ident {
	for _, f := range files {
		if path.Dir(f.rel) != specdataDir {
			continue
		}
		for _, decl := range f.file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "PrefixByte" {
				return fn.Name
			}
		}
	}
	return nil
}

// enclosingFunc returns the name of the top-level function each position
// of file lies in, "" outside every function.
func enclosingFunc(file *ast.File, pos token.Pos) string {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Pos() <= pos && pos < fn.End() {
			return fn.Name.Name
		}
	}
	return ""
}

// inTable reports whether pos lies in the declaration of the var table of
// the table's file.
func inTable(f goFile, pos token.Pos) bool {
	if f.rel != tableFile {
		return false
	}
	for _, decl := range f.file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value := spec.(*ast.ValueSpec)
			if len(value.Names) == 1 && value.Names[0].Name == "table" && value.Pos() <= pos && pos < value.End() {
				return true
			}
		}
	}
	return false
}

// rule2Findings reports every site of files the rule refuses, sorted, with
// scanned the packages checks 6 and 7 read.
func rule2Findings(fset *token.FileSet, files []goFile, scanned map[string]scannedPackage) []finding {
	var out []finding
	report := func(f goFile, pos token.Pos, format string, args ...any) {
		out = append(out, finding{rel: f.rel, line: fset.Position(pos).Line, what: fmt.Sprintf(format, args...)})
	}
	constants := constructionNames(files)
	declared := prefixByteDecl(files)
	kindParams := constructionParams(files)
	typed := memberSetTypes(fset, files, scanned)
	importable := importableDirs(files, scanned)
	for _, f := range files {
		dir := path.Dir(f.rel)
		inRegistry := dir == registryDir
		pkg, inScanned := scanned[dir]
		memberSets := inScanned && readsMemberSets(pkg, f.rel)
		printsNothing := inScanned && pkg.printsNothing
		fmtNames := importNames(f.file, "fmt")
		for _, spec := range f.file.Imports {
			p, _ := strconv.Unquote(spec.Path.Value)
			switch {
			case (p == merklePath || strings.HasPrefix(p, merklePath+"/")) && f.rel != walkFile:
				report(f, spec.Pos(), "imports %s, whose hasher applies 0x00 and 0x01, outside %s", p, walkFile)
			case p == tlogPath:
				report(f, spec.Pos(), "imports %s, the walk's cross-check, outside a test", p)
			case p == registryPath && !inScanned:
				report(f, spec.Pos(), "imports %s, but %s is not among the scanned packages", p, dir)
			case memberSets && (p == "maps" || p == "reflect"):
				report(f, spec.Pos(), "imports %s, which reads member names from a value", p)
			case inScanned && spec.Name != nil && spec.Name.Name == ".":
				report(f, spec.Pos(), "imports %s by \".\", which hides its uses from the scan", p)
			case inScanned && !slices.Contains(pkg.imports, p):
				report(f, spec.Pos(), "imports %s, which is not among the imports of %s", p, dir)
			}
		}
		specdataName := importName(f.file, specdataPath)
		registryName := importName(f.file, registryPath)
		sha256Name := importName(f.file, "crypto/sha256")
		var libraryNames []string
		for _, spec := range f.file.Imports {
			if p, _ := strconv.Unquote(spec.Path.Value); p == merklePath || strings.HasPrefix(p, merklePath+"/") {
				libraryNames = append(libraryNames, importName(f.file, p))
			}
		}
		ast.Inspect(f.file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.CallExpr:
				switch fun := n.Fun.(type) {
				case *ast.SelectorExpr:
					if x, ok := fun.X.(*ast.Ident); ok && x.Name == registryName && (fun.Sel.Name == "Construction" || fun.Sel.Name == "B64UClass") {
						report(f, n.Pos(), "converts a value to registry.%s", fun.Sel.Name)
					}
				case *ast.Ident:
					if printsNothing && (fun.Name == "print" || fun.Name == "println") {
						report(f, n.Pos(), "prints with the builtin %s, in a package that prints nothing", fun.Name)
					}
					if inRegistry && (fun.Name == "Construction" || fun.Name == "B64UClass") {
						report(f, n.Pos(), "converts a value to %s", fun.Name)
					}
					if inRegistry && fun.Name == "digest" && len(n.Args) > 0 {
						arg, ok := n.Args[0].(*ast.Ident)
						allowed := ok && ((constants[arg.Name] && arg.Name != "MerkleLeaf" && arg.Name != "MerkleNode") ||
							(arg.Name == "kind" && enclosingFunc(f.file, n.Pos()) == "RecordHash"))
						if !allowed {
							report(f, n.Pos(), "calls digest under a construction that is not a registry constant other than the Merkle leaf and node")
						}
					}
				}
			case *ast.SelectorExpr:
				x, ok := n.X.(*ast.Ident)
				if !ok {
					break
				}
				if printsNothing && slices.Contains(fmtNames, x.Name) && slices.Contains(printFuncs, n.Sel.Name) {
					report(f, n.Pos(), "prints with fmt.%s, in a package that prints nothing", n.Sel.Name)
				}
				if x.Name == specdataName && n.Sel.Name == "PrefixByte" {
					report(f, n.Pos(), "refers to specdata.PrefixByte, which reads a prefix byte, outside a test")
				}
				if !inRegistry && x.Name == registryName && (n.Sel.Name == "MerkleLeaf" || n.Sel.Name == "MerkleNode") {
					report(f, n.Pos(), "names the Merkle leaf or node byte, registry.%s, outside the walk", n.Sel.Name)
				}
				if inScanned && x.Name == sha256Name && (f.rel != tableFile || enclosingFunc(f.file, n.Pos()) != "digest") && !inContentDigest(f, n.Pos()) {
					report(f, n.Pos(), "uses crypto/sha256 outside the registry's digest")
				}
				if slices.Contains(libraryNames, x.Name) && enclosingFunc(f.file, n.Pos()) != "walk" {
					report(f, n.Pos(), "uses the Merkle library, %s.%s, outside the function walk", x.Name, n.Sel.Name)
				}
			case *ast.Ident:
				if dir == specdataDir && n.Name == "PrefixByte" && n != declared {
					report(f, n.Pos(), "refers to specdata.PrefixByte, which reads a prefix byte, outside a test")
				}
				if inRegistry && f.rel != tableFile && (n.Name == "MerkleLeaf" || n.Name == "MerkleNode") {
					report(f, n.Pos(), "names the Merkle leaf or node byte, %s, outside the table", n.Name)
				}
			case *ast.BasicLit:
				if !inScanned || inTable(f, n.Pos()) {
					break
				}
				if n.Kind == token.INT && strings.HasPrefix(strings.ToLower(n.Value), "0x") {
					report(f, n.Pos(), "writes the byte literal %s outside the table", n.Value)
				}
				if text, err := strconv.Unquote(n.Value); memberSets && n.Kind == token.STRING && err == nil && slices.Contains(memberNames, text) {
					report(f, n.Pos(), "names a registry member, %q, which only a compiled constant may name", text)
				}
			case *ast.CompositeLit:
				if !inScanned || inTable(f, n.Pos()) {
					break
				}
				array, ok := n.Type.(*ast.ArrayType)
				if !ok {
					break
				}
				if elt, ok := array.Elt.(*ast.Ident); !ok || elt.Name != "byte" {
					break
				}
				for _, element := range n.Elts {
					if _, literal := element.(*ast.BasicLit); literal {
						report(f, n.Pos(), "writes bytes as a literal outside the table")
						break
					}
				}
			}
			return true
		})
		switch {
		case memberSets && pkg.memberSets:
			out = append(out, syntacticMemberSetFindings(fset, f)...)
		case memberSets:
			out = append(out, memberSetFindings(fset, f, typed)...)
		}
		if inScanned && !inRegistry {
			out = append(out, kindFindings(fset, f, constants, kindParams[dir])...)
		}
		out = append(out, digestFindings(fset, f, !inScanned && importable[dir])...)
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

// syntacticMemberSetFindings reports check 7 in f, a file of a package
// check 7 reads whole, the registry, as its syntax shows a document's map:
// in each function, a range over a map parameter, over a map a type
// assertion gives or over a type switch's binding in a map case, and such
// a map indexed by a name that is not a string literal. The registry reads
// its own compiled tables, such as b64uClasses, by a key.
func syntacticMemberSetFindings(fset *token.FileSet, f goFile) []finding {
	var out []finding
	for _, decl := range f.file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		docMaps := map[string]bool{}
		for _, field := range fn.Type.Params.List {
			if _, isMap := field.Type.(*ast.MapType); isMap {
				for _, name := range field.Names {
					docMaps[name.Name] = true
				}
			}
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if assign, ok := node.(*ast.AssignStmt); ok && len(assign.Rhs) == 1 && isMapAssertion(assign.Rhs[0]) {
				if name, ok := assign.Lhs[0].(*ast.Ident); ok {
					docMaps[name.Name] = true
				}
			}
			return true
		})
		var inspect func(node ast.Node, docMaps map[string]bool)
		inspect = func(root ast.Node, docMaps map[string]bool) {
			isDocumentMap := func(expr ast.Expr) bool {
				if ident, ok := expr.(*ast.Ident); ok {
					return docMaps[ident.Name]
				}
				return isMapAssertion(expr)
			}
			ast.Inspect(root, func(node ast.Node) bool {
				switch n := node.(type) {
				case *ast.TypeSwitchStmt:
					binding := typeSwitchBinding(n)
					if binding == "" {
						return true
					}
					if n.Init != nil {
						inspect(n.Init, docMaps)
					}
					inspect(n.Assign, docMaps)
					for _, clause := range n.Body.List {
						clause := clause.(*ast.CaseClause)
						inner := docMaps
						if isMapCase(clause) != docMaps[binding] {
							inner = maps.Clone(docMaps)
							inner[binding] = isMapCase(clause)
						}
						for _, stmt := range clause.Body {
							inspect(stmt, inner)
						}
					}
					return false
				case *ast.RangeStmt:
					if isDocumentMap(n.X) {
						out = append(out, finding{rel: f.rel, line: fset.Position(n.Pos()).Line, what: "ranges over the members of an object"})
					}
				case *ast.IndexExpr:
					literal, isLiteral := n.Index.(*ast.BasicLit)
					if isDocumentMap(n.X) && (!isLiteral || literal.Kind != token.STRING) {
						out = append(out, finding{rel: f.rel, line: fset.Position(n.Pos()).Line, what: "reads a member of an object by a key that is not a literal"})
					}
				}
				return true
			})
		}
		inspect(fn.Body, docMaps)
	}
	return out
}

func isMapAssertion(expr ast.Expr) bool {
	assertion, ok := expr.(*ast.TypeAssertExpr)
	if !ok {
		return false
	}
	_, isMap := assertion.Type.(*ast.MapType)
	return isMap
}

// typeSwitchBinding is the name a type switch binds, as in switch v :=
// x.(type), or "" for one that binds none.
func typeSwitchBinding(n *ast.TypeSwitchStmt) string {
	if assign, ok := n.Assign.(*ast.AssignStmt); ok && len(assign.Lhs) == 1 {
		if name, ok := assign.Lhs[0].(*ast.Ident); ok {
			return name.Name
		}
	}
	return ""
}

// isMapCase reports whether a type switch's clause is a case of one map
// type, in which its binding is that map.
func isMapCase(clause *ast.CaseClause) bool {
	if len(clause.List) != 1 {
		return false
	}
	_, isMap := clause.List[0].(*ast.MapType)
	return isMap
}

// importerFunc is a types.Importer.
type importerFunc func(importPath string) (*types.Package, error)

func (f importerFunc) Import(importPath string) (*types.Package, error) { return f(importPath) }

// memberSetTypes type-checks, with go/types, each package of files that
// holds a file of memberSetFiles, and the packages of the module it imports,
// as internal/verifier/module_static_test.go's typeCheck does the module:
// the files of one directory and one package name are a package, whose
// import path is the module's path and the directory; a package from
// outside the module is given as an empty package, so an expression that
// needs one has no type; and type errors are set aside. Nothing is
// compiled or run.
func memberSetTypes(fset *token.FileSet, files []goFile, scanned map[string]scannedPackage) *types.Info {
	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Defs:  map[*ast.Ident]types.Object{},
		Uses:  map[*ast.Ident]types.Object{},
	}
	groups := map[string][]*ast.File{}
	names := map[string][]string{}
	var roots []string
	for _, f := range files {
		dir := path.Dir(f.rel)
		key := dir + " " + f.file.Name.Name
		if groups[key] == nil {
			names[dir] = append(names[dir], f.file.Name.Name)
		}
		groups[key] = append(groups[key], f.file)
		if pkg, ok := scanned[dir]; ok && !pkg.memberSets && readsMemberSets(pkg, f.rel) && !slices.Contains(roots, key) {
			roots = append(roots, key)
		}
	}
	checked := map[string]*types.Package{}
	checking := map[string]bool{}
	external := map[string]*types.Package{}
	var check func(key string) *types.Package
	importer := importerFunc(func(importPath string) (*types.Package, error) {
		if dir, ok := strings.CutPrefix(importPath, modulePrefix); ok && len(names[dir]) > 0 {
			candidates := slices.Sorted(slices.Values(names[dir]))
			name := candidates[0]
			if i := slices.IndexFunc(candidates, func(n string) bool { return n != "main" }); i >= 0 {
				name = candidates[i]
			}
			key := dir + " " + name
			if checking[key] {
				return nil, fmt.Errorf("import cycle through %s", importPath)
			}
			return check(key), nil
		}
		if p, ok := external[importPath]; ok {
			return p, nil
		}
		p := types.NewPackage(importPath, path.Base(importPath))
		p.MarkComplete()
		external[importPath] = p
		return p, nil
	})
	check = func(key string) *types.Package {
		if p, ok := checked[key]; ok {
			return p
		}
		dir, _, _ := strings.Cut(key, " ")
		checking[key] = true
		config := types.Config{Importer: importer, Error: func(error) {}}
		p, _ := config.Check(modulePrefix+dir, fset, groups[key], info)
		checking[key] = false
		checked[key] = p
		return p
	}
	slices.Sort(roots)
	for _, key := range roots {
		check(key)
	}
	return info
}

// The kinds of value memberSetFindings tells apart.
const (
	otherValue = iota
	mapValue
	untypedValue
)

// valueKind is what info says expr is: a map, a value it cannot type (an
// invalid type, which only a value from outside the module has, or a type
// parameter, whose type set the scan does not read), or another value.
func valueKind(info *types.Info, expr ast.Expr) int {
	t := info.TypeOf(expr)
	if t == nil {
		return untypedValue
	}
	switch u := t.Underlying().(type) {
	case *types.Map:
		return mapValue
	case *types.Interface:
		return untypedValue
	case *types.Basic:
		if u.Kind() == types.Invalid {
			return untypedValue
		}
	}
	return otherValue
}

// memberSetFindings reports, anywhere in f, a range over a value of map
// type and a read of one by a key that is not a constant, but the index of
// pointerReads, whatever yields the map, with info the types memberSetTypes
// gives; and a range, or a read by a key that is not a constant, over a
// value info cannot type.
func memberSetFindings(fset *token.FileSet, f goFile, info *types.Info) []finding {
	var out []finding
	report := func(pos token.Pos, what string) {
		out = append(out, finding{rel: f.rel, line: fset.Position(pos).Line, what: what})
	}
	ast.Inspect(f.file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.RangeStmt:
			switch valueKind(info, n.X) {
			case mapValue:
				report(n.Pos(), "ranges over the members of an object")
			case untypedValue:
				report(n.Pos(), "ranges over a value whose type the scan cannot tell")
			}
		case *ast.IndexExpr:
			if index, ok := info.Types[n.Index]; ok && (index.IsType() || index.Value != nil) {
				break
			}
			segment, isSegment := n.Index.(*ast.SelectorExpr)
			if isSegment && segment.Sel.Name == "name" && slices.Contains(pointerReads, pointerRead{f.rel, enclosingFunc(f.file, n.Pos())}) {
				break
			}
			switch valueKind(info, n.X) {
			case mapValue:
				report(n.Pos(), "reads a member of an object by a key that is not a constant")
			case untypedValue:
				report(n.Pos(), "reads a value whose type the scan cannot tell by a key that is not a constant")
			}
		}
		return true
	})
	return out
}

// modulePrefix is the import path prefix of the module's packages.
const modulePrefix = "github.com/truestamp/truestamp-cli/"

// isConstructionType reports whether expr is the type registry.Construction,
// with registryName the file's name for the registry.
func isConstructionType(expr ast.Expr, registryName string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || registryName == "" {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == registryName && sel.Sel.Name == "Construction"
}

// constructionParams gives, for each directory of files outside the
// registry and each top-level function declared there, the positions of
// its parameters declared as a registry.Construction (check 11).
func constructionParams(files []goFile) map[string]map[string][]int {
	out := map[string]map[string][]int{}
	for _, f := range files {
		dir := path.Dir(f.rel)
		registryName := importName(f.file, registryPath)
		if dir == registryDir || registryName == "" {
			continue
		}
		for _, decl := range f.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			position := 0
			for _, field := range fn.Type.Params.List {
				names := max(len(field.Names), 1)
				if isConstructionType(field.Type, registryName) {
					for i := range names {
						if out[dir] == nil {
							out[dir] = map[string][]int{}
						}
						out[dir][fn.Name.Name] = append(out[dir][fn.Name.Name], position+i)
					}
				}
				position += names
			}
		}
	}
	return out
}

// kindFindings reports check 11 in f, a file of a scanned package outside
// the registry: a RecordHash or RecordHashHex kind, or a kind given to a
// function of params (the package's, from constructionParams), that is
// neither a registry constant but the leaf and node nor a parameter of the
// enclosing function declared as a registry.Construction; and the type
// registry.Construction written anywhere but as the type of a top-level
// function's parameter.
func kindFindings(fset *token.FileSet, f goFile, constants map[string]bool, params map[string][]int) []finding {
	registryName := importName(f.file, registryPath)
	if registryName == "" {
		return nil
	}
	var out []finding
	report := func(pos token.Pos, what string) {
		out = append(out, finding{rel: f.rel, line: fset.Position(pos).Line, what: what})
	}
	allowedTypes := map[ast.Expr]bool{}
	for _, decl := range f.file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
			for _, field := range fn.Type.Params.List {
				allowedTypes[field.Type] = true
			}
		}
	}
	callees := map[ast.Expr]bool{}
	for _, decl := range f.file.Decls {
		fn, _ := decl.(*ast.FuncDecl)
		kindOK := func(arg ast.Expr) bool {
			if sel, ok := arg.(*ast.SelectorExpr); ok {
				x, ok := sel.X.(*ast.Ident)
				return ok && x.Name == registryName && constants[sel.Sel.Name] && sel.Sel.Name != "MerkleLeaf" && sel.Sel.Name != "MerkleNode"
			}
			ident, ok := arg.(*ast.Ident)
			if !ok || fn == nil || fn.Recv != nil {
				return false
			}
			for _, field := range fn.Type.Params.List {
				for _, name := range field.Names {
					if name.Name == ident.Name && isConstructionType(field.Type, registryName) {
						return true
					}
				}
			}
			return false
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			callees[call.Fun] = true
			var positions []int
			name := ""
			switch fun := call.Fun.(type) {
			case *ast.SelectorExpr:
				if x, ok := fun.X.(*ast.Ident); ok && x.Name == registryName && (fun.Sel.Name == "RecordHash" || fun.Sel.Name == "RecordHashHex") {
					positions, name = []int{0}, "registry."+fun.Sel.Name
				}
			case *ast.Ident:
				positions, name = params[fun.Name], fun.Name
			}
			for _, i := range positions {
				if i < len(call.Args) && !kindOK(call.Args[i]) {
					report(call.Args[i].Pos(), fmt.Sprintf("gives %s a kind that is not a registry constant or a registry.Construction parameter", name))
				}
			}
			return true
		})
	}
	ast.Inspect(f.file, func(node ast.Node) bool {
		if expr, ok := node.(ast.Expr); ok && isConstructionType(expr, registryName) && !allowedTypes[expr] && !callees[expr] {
			report(expr.Pos(), "holds a registry.Construction elsewhere than in a function's parameter")
		}
		return true
	})
	return out
}

// digestFindings reports check 12 in f: a reference to one of
// namedDigests but at its one call; and, with held true, f's package being
// one a scanned package may import and not itself scanned, a use of
// crypto/sha256 outside the package's importedDigests, and an import of a
// hash package other than crypto/sha256, or of crypto/sha256 by ".".
func digestFindings(fset *token.FileSet, f goFile, held bool) []finding {
	var out []finding
	report := func(pos token.Pos, what string) {
		out = append(out, finding{rel: f.rel, line: fset.Position(pos).Line, what: what})
	}
	dir := path.Dir(f.rel)
	// Names that are not references: a selector's member, a field's or a
	// method's name, and a key of a composite literal.
	notReferences := map[*ast.Ident]bool{}
	callees := map[*ast.Ident]bool{}
	ast.Inspect(f.file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.SelectorExpr:
			notReferences[n.Sel] = true
		case *ast.Field:
			for _, name := range n.Names {
				notReferences[name] = true
			}
		case *ast.KeyValueExpr:
			if key, ok := n.Key.(*ast.Ident); ok {
				notReferences[key] = true
			}
		case *ast.FuncDecl:
			notReferences[n.Name] = true
		case *ast.CallExpr:
			if ident, ok := n.Fun.(*ast.Ident); ok {
				callees[ident] = true
			}
		}
		return true
	})
	for _, digest := range namedDigests {
		call := digestCalls[digest]
		digestDir := path.Dir(digest.file)
		if dir == digestDir {
			ast.Inspect(f.file, func(node ast.Node) bool {
				ident, ok := node.(*ast.Ident)
				if !ok || ident.Name != digest.function || notReferences[ident] {
					return true
				}
				if !(f.rel == call.file && enclosingFunc(f.file, ident.Pos()) == call.function && callees[ident]) {
					report(ident.Pos(), fmt.Sprintf("refers to the digest %s outside its one call, in %s's %s", digest.function, call.file, call.function))
				}
				return true
			})
			continue
		}
		name := importName(f.file, modulePrefix+digestDir)
		if name == "" {
			continue
		}
		ast.Inspect(f.file, func(node ast.Node) bool {
			if sel, ok := node.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == name && sel.Sel.Name == digest.function {
					report(sel.Pos(), fmt.Sprintf("refers to the digest %s.%s outside its one call, in %s's %s", x.Name, digest.function, call.file, call.function))
				}
			}
			return true
		})
	}
	if !held {
		return out
	}
	for _, spec := range f.file.Imports {
		p, _ := strconv.Unquote(spec.Path.Value)
		switch {
		case hasher(p) && p != "crypto/sha256":
			report(spec.Pos(), fmt.Sprintf("imports %s, a hash package, into %s, which the new verifier may import", p, dir))
		case p == "crypto/sha256" && spec.Name != nil && spec.Name.Name == ".":
			report(spec.Pos(), fmt.Sprintf("imports %s by \".\", which hides its uses from the scan", p))
		case !standardLibrary(p) && !strings.HasPrefix(p, modulePrefix) && !importableDependency(p):
			report(spec.Pos(), fmt.Sprintf("imports %s, a dependency not on importableDependencies, into %s, which the new verifier may import", p, dir))
		}
	}
	sha256Name := importName(f.file, "crypto/sha256")
	if sha256Name == "" {
		return out
	}
	ast.Inspect(f.file, func(node ast.Node) bool {
		if sel, ok := node.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == sha256Name && !inImportedDigest(f, sel.Pos()) {
				report(sel.Pos(), fmt.Sprintf("uses crypto/sha256 outside the named digests of %s, which the new verifier may import", dir))
			}
		}
		return true
	})
	return out
}

// inImportedDigest reports whether pos, in f, lies in the body of a
// function, not a method, of importedDigests.
func inImportedDigest(f goFile, pos token.Pos) bool {
	for _, decl := range f.file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Pos() <= pos && pos < fn.End() {
			return slices.Contains(importedDigests, contentDigest{f.rel, fn.Name.Name})
		}
	}
	return false
}

// importableDirs are the directories of the packages a scanned package
// may import: the scanned packages, the module's packages on their lists,
// and, read from files, the module's packages any of those imports, and so
// on.
func importableDirs(files []goFile, scanned map[string]scannedPackage) map[string]bool {
	out := map[string]bool{}
	var queue []string
	add := func(dir string) {
		if !out[dir] {
			out[dir] = true
			queue = append(queue, dir)
		}
	}
	for dir, pkg := range scanned {
		add(dir)
		for _, p := range pkg.imports {
			if imported, ok := strings.CutPrefix(p, modulePrefix); ok {
				add(imported)
			}
		}
	}
	imports := map[string][]string{}
	for _, f := range files {
		for _, spec := range f.file.Imports {
			if p, _ := strconv.Unquote(spec.Path.Value); strings.HasPrefix(p, modulePrefix) {
				imports[path.Dir(f.rel)] = append(imports[path.Dir(f.rel)], strings.TrimPrefix(p, modulePrefix))
			}
		}
	}
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		for _, imported := range imports[dir] {
			add(imported)
		}
	}
	return out
}

// The module holds to checks 1 to 5 and 9, and the scanned packages to
// checks 6 to 8. The scan is not vacuous: it reads the commands and every
// internal package, each scanned package, the walk's own use of the
// library, and digest. Mutants killed: a 0x01 literal hashed in key_id.go;
// the library's hasher named in a second function of walk.go; a
// specdata.PrefixByte call added to cmd/hash.go; a scanned package's path
// misspelt in scannedPackages; layout.go left out of internal/verifier's
// memberSetFiles (check 7 does not read it); in DERIVE's files, a range
// over a block node's map field, over a call's map result, over a local
// bound from a call with a read of it, over a var, over an alias and over a
// closure's parameter, and a read of a node's field and of a call's result
// by a name (check 7); a keyring.Digest call added to Policy, an exported
// SHA-256 added to internal/specdata and called from Policy, an exported
// wrapper over an unexported SHA-256 helper added to internal/specdata and
// to internal/jcs and called from DERIVE with the prefix byte written
// outside a composite literal, and an exported var holding sha256.Sum256
// and a crypto/sha512 export added to internal/specdata and called from
// Policy (check 12).
func TestRule2_TheModuleAndTheScannedPackagesTakeNoByteOrMemberSetFromADocument(t *testing.T) {
	fset, files := parseModule(t, repoRoot)
	if findings := rule2Findings(fset, files, scannedPackages); len(findings) > 0 {
		var lines []string
		for _, f := range findings {
			lines = append(lines, f.String())
		}
		t.Fatalf("SPEC.md section 10.6, rule 2:\n  %s", strings.Join(lines, "\n  "))
	}
	seen := map[string]bool{}
	dirs := map[string]bool{}
	for _, f := range files {
		seen[f.rel] = true
		dirs[path.Dir(f.rel)] = true
		if f.rel == walkFile {
			uses := 0
			ast.Inspect(f.file, func(node ast.Node) bool {
				if sel, ok := node.(*ast.SelectorExpr); ok {
					if x, ok := sel.X.(*ast.Ident); ok && (x.Name == "proof" || x.Name == "rfc6962") && enclosingFunc(f.file, sel.Pos()) == "walk" {
						uses++
					}
				}
				return true
			})
			if uses < 2 {
				t.Errorf("walk uses the library %d times; the scan would show nothing", uses)
			}
		}
	}
	for _, want := range []string{walkFile, tableFile, "internal/registry/key_id.go", "cmd/hash.go", "internal/specdata/specfile.go", "cmd/truestamp/main.go"} {
		if !seen[want] {
			t.Errorf("the scan did not read %s", want)
		}
	}
	for dir := range scannedPackages {
		if !dirs[dir] {
			t.Errorf("the scanned package %s has no file the scan read", dir)
		}
	}
	if prefixByteDecl(files) == nil {
		t.Error("the scan found no declaration of specdata.PrefixByte")
	}
	var memberSetFiles []string
	for _, f := range files {
		if pkg, ok := scannedPackages[path.Dir(f.rel)]; ok && !pkg.memberSets && readsMemberSets(pkg, f.rel) {
			memberSetFiles = append(memberSetFiles, f.rel)
		}
	}
	sort.Strings(memberSetFiles)
	if want := []string{"internal/verifier/derive.go", "internal/verifier/derive_check.go", "internal/verifier/derive_links.go",
		"internal/verifier/derive_records.go", "internal/verifier/layout.go"}; !slices.Equal(memberSetFiles, want) {
		t.Errorf("check 7 reads %q of the verifier, want %q", memberSetFiles, want)
	}
	if len(files) < 150 || len(constructionNames(files)) != 13 {
		t.Fatalf("the scan read %d files and %d constructions", len(files), len(constructionNames(files)))
	}
}

// wantComment is a planted line's expected findings: `// want "a" "b"`, or,
// for a run in which the planted package is not scanned, `// unlisted "a"`.
var wantComment = regexp.MustCompile(`// (want|unlisted)((?: "(?:[^"\\]|\\.)+")+)\s*$`)

// quotedWant is one expected finding in a want comment.
var quotedWant = regexp.MustCompile(`"(?:[^"\\]|\\.)+"`)

// plantedFiles are files of each shape the rule refuses, with the findings
// each line should give, and files of the shapes it allows, which should
// give none.
var plantedFiles = map[string]string{
	"internal/registry/planted.go": `package registry

import (
	"crypto" // want "imports crypto, which is not among the imports of internal/registry"
	"crypto/sha256"
	"maps" // want "imports maps"
	"reflect" // want "imports reflect"
	"strconv"
	. "strings" // want "imports strings by"

	"github.com/transparency-dev/merkle/rfc6962" // want "imports github.com/transparency-dev/merkle/rfc6962"
	"github.com/truestamp/truestamp-cli/internal/tscrypto" // want "imports github.com/truestamp/truestamp-cli/internal/tscrypto, which is not among the imports of internal/registry"
)

func planted(doc map[string]any, name string, n int, v []byte) {
	_ = Construction(n) // want "converts a value to Construction"
	_ = B64UClass(n) // want "converts a value to B64UClass"
	_ = digest(MerkleNode, nil) // want "calls digest" "names the Merkle leaf or node byte, MerkleNode"
	_ = digest(Construction(n), nil) // want "calls digest" "converts a value to Construction"
	_ = digest(ClaimsHash-2, nil) // want "calls digest"
	_ = digest(kind, nil) // want "calls digest"
	_ = sha256.Sum256([]byte{1}) // want "uses crypto/sha256" "writes bytes as a literal"
	_ = 0x6a // want "writes the byte literal 0x6a"
	_ = [1]byte{0} // want "writes bytes as a literal"
	for member := range doc { // want "ranges over the members"
		_ = member
	}
	object := doc["x"].(map[string]any)
	for member := range object { // want "ranges over the members"
		_ = member
	}
	for member := range doc["y"].(map[string]any) { // want "ranges over the members"
		_ = member
	}
	_ = object[name] // want "by a key that is not a literal"
	_ = rfc6962.DefaultHasher // want "uses the Merkle library"
	_, _ = maps.Keys(doc), reflect.ValueOf(doc)
	_ = MerkleLeaf // want "names the Merkle leaf or node byte, MerkleLeaf"
	h := crypto.SHA256.New()
	h.Write([]byte{byte(0)})
	h.Write(make([]byte, 1))
	b, _ := strconv.ParseUint(doc["byte"].(string), 0, 8) // want "names a registry member, \"byte\""
	_ = []string{"prefix", "prefixes", "members", "fields"} // want "\"prefix\"" "\"prefixes\"" "\"members\"" "\"fields\""
	_, _ = b, ToLower(name)
	_ = tscrypto.DomainHash(tscrypto.PrefixMerkleLeaf, v)
}
`,
	"internal/registry/walk.go": `package registry

import (
	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
)

func walk() {
	_, _ = rfc6962.DefaultHasher, proof.RootFromInclusionProof
}

func another() {
	_ = rfc6962.DefaultHasher // want "uses the Merkle library, rfc6962.DefaultHasher"
}
`,
	"internal/registry/prefixes.go": `package registry

import "crypto/sha256"

type Construction int

const (
	MerkleLeaf Construction = iota + 1
	MerkleNode
	ClaimsHash
	KeyID
)

type entry struct{ Byte byte }

var table = [...]entry{{0x00}, {0x01}, {0x10}, {0x60}}

func (c Construction) entry() (entry, bool) { return table[c-MerkleLeaf], true }

func digest(c Construction, data []byte) [32]byte {
	_ = []byte{table[c-MerkleLeaf].Byte}
	return sha256.Sum256(data)
}

func leafOutsideDigest(v []byte) [32]byte {
	row, _ := MerkleLeaf.entry()
	return sha256.Sum256(append([]byte{row.Byte}, v...)) // want "uses crypto/sha256 outside the registry's digest"
}
`,
	"internal/registry/key_id.go": `package registry

import "crypto/sha256"

var table = []byte{0x01} // want "writes the byte literal 0x01" "writes bytes as a literal"

func digest(c Construction, data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the registry's digest"
}
`,
	"internal/registry/allowed.go": `package registry

func RecordHash(kind Construction, record map[string]any, path []any) {
	_ = digest(kind, nil)
	_ = digest(KeyID, nil)
	for i, node := range path {
		_, _ = i, node
	}
	_ = record["leaf_index"]
	object, _ := record["proof"].(map[string]any)
	_ = object["path"]
	_ = "a byte"
}
`,
	"cmd/planted.go": `package cmd

import (
	"github.com/transparency-dev/merkle/proof" // want "imports github.com/transparency-dev/merkle/proof"
	"golang.org/x/mod/sumdb/tlog" // want "imports golang.org/x/mod/sumdb/tlog"

	"github.com/truestamp/truestamp-cli/internal/registry" // want "cmd is not among the scanned packages"
	"github.com/truestamp/truestamp-cli/internal/specdata"
)

func planted(doc map[string]any, n int) {
	b, _ := specdata.PrefixByte(doc["byte"].(string)) // want "specdata.PrefixByte"
	read := specdata.PrefixByte // want "specdata.PrefixByte"
	_ = registry.Construction(n) // want "converts a value to registry.Construction"
	_ = registry.B64UClass(n) // want "converts a value to registry.B64UClass"
	_ = registry.MerkleLeaf // want "registry.MerkleLeaf"
	_ = registry.MerkleNode // want "registry.MerkleNode"
	_, _, _ = b, proof.VerifyInclusion, tlog.RecordHash // want "uses the Merkle library, proof.VerifyInclusion"
	_, _ = registry.RecordHash(registry.BlockHash, doc)
	_, _ = registry.KeyIDOf([32]byte{0x60}), read
}
`,
	"internal/specdata/specfile.go": `package specdata

func PrefixByte(text string) (byte, error) { return 0, nil }
`,
	"internal/specdata/planted.go": `package specdata

var read = PrefixByte // want "specdata.PrefixByte"

func planted(text string) {
	_, _ = PrefixByte(text) // want "specdata.PrefixByte"
}
`,
}

// derivePlanted is a new package of the verifier that hashes under the
// leaf and node bytes itself, takes a prefix byte from a document and
// reads a document's member names. While it is not among the scanned
// packages, check 9 reports its import of the registry and nothing else;
// once it is, checks 6 and 7 report each shape.
const derivePlanted = `package derive

import (
	"crypto/sha256"

	"github.com/truestamp/truestamp-cli/internal/registry" // unlisted "internal/derive is not among the scanned packages"
)

func Leaf(v []byte) [32]byte {
	return sha256.Sum256(append([]byte{0x00}, v...)) // want "uses crypto/sha256" "writes the byte literal 0x00" "writes bytes as a literal"
}

func Node(left, right []byte) []byte {
	h := sha256.New() // want "uses crypto/sha256"
	h.Write([]byte{0x01}) // want "writes the byte literal 0x01" "writes bytes as a literal"
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

func FromDocument(doc map[string]any, v []byte) [32]byte {
	p := byte(doc["prefix"].(float64)) // want "names a registry member, \"prefix\""
	return sha256.Sum256(append([]byte{p}, v...)) // want "uses crypto/sha256"
}

func Members(doc map[string]any) []string {
	var names []string
	for name := range doc { // want "ranges over the members"
		names = append(names, name)
	}
	return names
}

var _ = registry.KeyIDOf
`

// parsePlanted parses sources and returns them with the findings each line
// should give under the comments of kind, "want" or "unlisted".
func parsePlanted(t *testing.T, sources map[string]string, kind string) (*token.FileSet, []goFile, map[string][]string) {
	t.Helper()
	fset := token.NewFileSet()
	var files []goFile
	want := map[string][]string{}
	for rel, source := range sources {
		file, err := parser.ParseFile(fset, rel, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		files = append(files, goFile{rel: rel, file: file})
		for i, line := range strings.Split(source, "\n") {
			match := wantComment.FindStringSubmatch(line)
			if match == nil || match[1] != kind {
				continue
			}
			key := fmt.Sprintf("%s:%d", rel, i+1)
			for _, quoted := range quotedWant.FindAllString(match[2], -1) {
				text, err := strconv.Unquote(quoted)
				if err != nil {
					t.Fatalf("%s: %v", key, err)
				}
				want[key] = append(want[key], text)
			}
		}
	}
	return fset, files, want
}

// assertFindings holds the findings got to want, line by line: each line
// gives exactly as many findings as it names, each named, and no other
// line gives any.
func assertFindings(t *testing.T, got []finding, want map[string][]string) {
	t.Helper()
	byLine := map[string][]string{}
	for _, f := range got {
		key := fmt.Sprintf("%s:%d", f.rel, f.line)
		byLine[key] = append(byLine[key], f.what)
	}
	for key, substrings := range want {
		findings := byLine[key]
		if len(findings) != len(substrings) {
			t.Errorf("%s: %d findings %q, want %d: %q", key, len(findings), findings, len(substrings), substrings)
			continue
		}
		for _, substring := range substrings {
			if !slices.ContainsFunc(findings, func(what string) bool { return strings.Contains(what, substring) }) {
				t.Errorf("%s: no finding names %q among %q", key, substring, findings)
			}
		}
	}
	for key, findings := range byLine {
		if _, ok := want[key]; !ok {
			t.Errorf("%s: unexpected findings %q", key, findings)
		}
	}
}

// Each planted shape gives exactly the findings its line names, and the
// allowed shapes give none. Mutants killed: the range check left out; the
// digest argument check accepting MerkleNode; the conversion check limited
// to Construction; the registry's import list left out (crypto.SHA256 fed
// byte(0) or make([]byte, 1), and tscrypto.DomainHash under
// tscrypto.PrefixMerkleLeaf, then pass); the member-name check left out (a
// byte strconv.ParseUint reads from doc["byte"] passes); crypto/sha256
// allowed in a function named digest in any registry file, or in any
// function of prefixes.go (the Merkle leaf's byte hashed outside digest
// passes); check 5 limited to calls (specdata.PrefixByte taken as a value
// passes); the dot-import check left out; a var table in any registry file
// taken for the table (a byte literal in key_id.go's passes).
func TestRule2_FindsEveryPlantedViolation(t *testing.T) {
	fset, files, want := parsePlanted(t, plantedFiles, "want")
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)
	if len(want) < 35 {
		t.Fatalf("%d planted lines", len(want))
	}
}

// A new package that imports the registry is reported until it is among
// the scanned packages, and once it is, each shape it holds is reported.
// Mutants killed: check 9 left out (the unlisted package gives no
// finding); checks 6 and 7 held to internal/registry whatever the list
// says (the listed package's shapes give none).
func TestRule2_ScansEachNewPackageThatImportsTheRegistry(t *testing.T) {
	sources := map[string]string{"internal/derive/derive.go": derivePlanted}

	fset, files, unlisted := parsePlanted(t, sources, "unlisted")
	if len(unlisted) != 1 {
		t.Fatalf("%d unlisted lines, want the import alone", len(unlisted))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), unlisted)

	fset, files, want := parsePlanted(t, sources, "want")
	if len(want) != 6 {
		t.Fatalf("%d planted lines, want 6", len(want))
	}
	listed := maps.Clone(scannedPackages)
	listed["internal/derive"] = scannedPackage{imports: []string{"crypto/sha256", registryPath}, memberSets: true}
	assertFindings(t, rule2Findings(fset, files, listed), want)
}

// The module's own layout holds the one pointer read check 7 allows: with
// pointerReads empty, the scan of the module gives exactly one finding,
// valueAt's index in layout.go, so the allowance is in use and covers
// nothing else; and DERIVE's calls of the registry's record hash are
// among what check 11 reads. Mutants killed: check 7 not reading
// layout.go (no finding); the read of a map left out of the typed check
// (no finding).
func TestRule2_TheLayoutsPointerReadIsTheOneIndexAllowed(t *testing.T) {
	saved := pointerReads
	pointerReads = nil
	defer func() { pointerReads = saved }()
	fset, files := parseModule(t, repoRoot)
	findings := rule2Findings(fset, files, scannedPackages)
	if len(findings) != 1 || findings[0].rel != "internal/verifier/layout.go" || !strings.Contains(findings[0].what, "not a constant") {
		t.Fatalf("without the pointer read's allowance: %v, want the one index in layout.go's valueAt", findings)
	}
	calls := 0
	for _, f := range files {
		if path.Dir(f.rel) != "internal/verifier" {
			continue
		}
		name := importName(f.file, registryPath)
		ast.Inspect(f.file, func(node ast.Node) bool {
			if sel, ok := node.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == name && strings.HasPrefix(sel.Sel.Name, "RecordHash") {
					calls++
				}
			}
			return true
		})
	}
	if calls < 4 {
		t.Fatalf("internal/verifier calls the record hash %d times", calls)
	}
}

// The directories skipped at the module root are read below it, while a
// directory whose name begins with "." and a nested module are skipped at
// any depth. Mutant killed: skippedDirs matched against each directory's
// own name, at any depth, which skips a package's tmp and build.
func TestRule2_SkipsTheRootsOwnDirectoriesOnly(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"go.mod",
		"main.go",
		"build/a.go", "dist/a.go", "tmp/a.go", "coverage/a.go", "testdata/a.go", "vendor/a.go",
		".hidden/a.go", "internal/v/.hidden/a.go",
		"internal/v/a.go", "internal/v/a_test.go",
		"internal/v/build/a.go", "internal/v/dist/a.go", "internal/v/tmp/a.go",
		"internal/v/coverage/a.go", "internal/v/testdata/a.go", "internal/v/vendor/a.go",
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
	_, files := parseModule(t, root)
	var got []string
	for _, f := range files {
		got = append(got, f.rel)
	}
	sort.Strings(got)
	want := []string{
		"internal/v/a.go",
		"internal/v/build/a.go", "internal/v/coverage/a.go", "internal/v/dist/a.go",
		"internal/v/testdata/a.go", "internal/v/tmp/a.go", "internal/v/vendor/a.go",
		"main.go",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the scan read %q, want %q", got, want)
	}
}

// verifierPlanted is a file of a package of the new verifier other than the
// registry that hashes through what the scan's other checks do not see:
// the generic crypto package, another hash, the old prefix tables and the
// CLI's hash command's registry. Each import is off the package's list but
// crypto/sha256, which the package lists for its content digest, and
// which check 6 refuses here, outside that function.
const verifierPlanted = `package verifier

import (
	"crypto" // want "imports crypto, which is not among the imports of internal/verifier"
	"crypto/sha256"
	"crypto/sha512" // want "imports crypto/sha512, which is not among"
	"hash/fnv" // want "imports hash/fnv, which is not among"

	"github.com/truestamp/truestamp-cli/internal/hashing" // want "imports github.com/truestamp/truestamp-cli/internal/hashing, which is not among"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/tscrypto" // want "imports github.com/truestamp/truestamp-cli/internal/tscrypto, which is not among"
)

func planted(v []byte) {
	h := crypto.SHA256.New()
	h.Write(v)
	_ = sha256.New // want "uses crypto/sha256 outside the registry's digest"
	_, _ = sha512.New, fnv.New32
	_, _ = hashing.Names, tscrypto.DomainHash
	_ = specdata.MaxDepth
	_ = 0x50 // want "writes the byte literal 0x50"
}

// The gate reads a document's members by design, so check 7 does not
// read this package.
func members(document map[string]any, name string) {
	for member := range document {
		_ = document[member]
	}
	_, _ = document[name], "members"
}
`

// Each package of the new verifier is held to its own list of imports, not
// the registry alone, and a package's list allows exactly what it names;
// check 6 reads it, and check 7 does not, since the gate reads a
// document's members. Mutants killed: check 8 made in internal/registry
// alone (the planted verifier file's hashers give no finding); check 7
// made in every scanned package (the gate's range over the document's
// members gives a finding).
func TestRule2_HoldsEachScannedPackageToItsImports(t *testing.T) {
	sources := map[string]string{"internal/verifier/planted.go": verifierPlanted}
	fset, files, want := parsePlanted(t, sources, "want")
	if len(want) != 7 {
		t.Fatalf("%d planted lines, want 7", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)
}

// sourcePlanted is a file of internal/verifier/source that prints what a
// Bitcoin Core node is sent and answers, each way fmt and the builtins
// can, beside the formatting calls that print nothing.
const sourcePlanted = `package source

import (
	"fmt"
	out "fmt"
)

func planted(header string) error {
	fmt.Print(header) // want "prints with fmt.Print,"
	fmt.Printf("%s", header) // want "prints with fmt.Printf"
	fmt.Println(header) // want "prints with fmt.Println"
	fmt.Fprint(nil, header) // want "prints with fmt.Fprint,"
	fmt.Fprintf(nil, "%s", header) // want "prints with fmt.Fprintf"
	fmt.Fprintln(nil, header) // want "prints with fmt.Fprintln"
	out.Println(header) // want "prints with fmt.Println"
	print(header) // want "prints with the builtin print"
	println(header) // want "prints with the builtin println"
	p := fmt.Fprintln // want "prints with fmt.Fprintln"
	p(nil, header)
	_ = fmt.Sprintf("%s", header)
	return fmt.Errorf("%s", header)
}
`

// Check 10: internal/verifier/source prints nothing, by fmt, by fmt under
// another name or by a builtin, while it may format; a package not marked
// to print nothing, internal/verifier, may print. With the check, the scan
// of the module fails a fmt.Printf or a println of the Authorization
// header added to CoreIdentity. Mutants killed: check 10 left out, or the
// source package's printsNothing mark dropped (each planted print gives
// no finding); check 10 without the builtins (print and println pass);
// check 10 reading only the name fmt (out.Println passes); check 10
// reading only a call's function (fmt.Fprintln taken as a value and then
// called passes); check 10 read in every scanned package (the
// internal/verifier copy gives findings).
func TestRule2_TheSourcePackagePrintsNothing(t *testing.T) {
	fset, files, want := parsePlanted(t, map[string]string{"internal/verifier/source/planted.go": sourcePlanted}, "want")
	if len(want) != 10 {
		t.Fatalf("%d planted lines, want 10", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)
	fset, files, _ = parsePlanted(t, map[string]string{"internal/verifier/planted.go": strings.Replace(sourcePlanted, "package source", "package verifier", 1)}, "want")
	assertFindings(t, rule2Findings(fset, files, scannedPackages), nil)
}

// hasherPackages are the hash packages no package of the new verifier but
// the registry may import (check 8): the generic crypto package, each
// hash of the standard library, and the CLI's own hashers.
var hasherPackages = []string{
	"crypto", "crypto/sha256", "crypto/sha1", "crypto/sha512", "crypto/md5", "crypto/hmac", "crypto/sha3",
	"github.com/truestamp/truestamp-cli/internal/tscrypto", "github.com/truestamp/truestamp-cli/internal/hashing",
}

// importableDependencies are the dependencies outside the module that the
// packages the new verifier may import use (check 12), each a path or a
// path prefix ending in "/". A dependency off the list could hash, as
// btcd's chainhash and golang.org/x/crypto do, so a new one is reviewed
// before it joins.
var importableDependencies = []string{"github.com/gowebpki/jcs", "github.com/knadh/koanf/", "github.com/spf13/pflag"}

// importableDependency reports whether p is on importableDependencies.
func importableDependency(p string) bool {
	for _, d := range importableDependencies {
		if p == d || (strings.HasSuffix(d, "/") && strings.HasPrefix(p, d)) {
			return true
		}
	}
	return false
}

// standardLibrary reports whether p is a standard library package: its
// first element holds no dot.
func standardLibrary(p string) bool {
	first, _, _ := strings.Cut(p, "/")
	return !strings.Contains(first, ".")
}

// hasher reports whether p is a hash package: one of hasherPackages, hash
// or a hash/ package.
func hasher(p string) bool {
	return slices.Contains(hasherPackages, p) || p == "hash" || strings.HasPrefix(p, "hash/")
}

// hasherImports names each hash package a list of packages other than the
// registry's allows, but crypto/sha256 on the list of a package that holds
// one of contentDigests.
func hasherImports(packages map[string]scannedPackage) []string {
	var found []string
	for dir, pkg := range packages {
		if dir == registryDir {
			continue
		}
		for _, p := range pkg.imports {
			if hasher(p) && !(contentDigestDir(dir) && p == "crypto/sha256") {
				found = append(found, dir+" may import "+p)
			}
		}
	}
	sort.Strings(found)
	return found
}

// No list of a package of the new verifier but the registry's names a
// hash package, but crypto/sha256 on the lists of the two packages that
// hold contentDigests: crypto/sha256 is otherwise the registry's alone,
// and crypto, the other hashes of crypto/, hash and the hash/ packages,
// internal/tscrypto and internal/hashing are no one's; another crypto/
// package, such as crypto/ed25519, may be listed; and every package under
// internal/verifier is on the list. Mutants killed: crypto/sha256 added to
// internal/verifier/ed25519's list; contentDigests holding keyring.Digest
// alone (internal/verifier's crypto/sha256 is named); hasher banning every
// crypto/ package (the planted list's crypto/ed25519 is named); the
// content digests' allowance made for every package (the planted
// internal/verifier/x's crypto/sha256 is not named) or for every hash
// package of their own (the planted keyring's crypto/sha512 and
// internal/verifier's crypto/md5 are not named).
func TestRule2_NoOtherPackageOfTheNewVerifierMayImportAHasher(t *testing.T) {
	if found := hasherImports(scannedPackages); len(found) != 0 {
		t.Errorf("hash packages listed: %q", found)
	}
	planted := map[string]scannedPackage{
		registryDir:                 {imports: []string{"crypto/sha256"}},
		"internal/verifier":         {imports: []string{"crypto/sha256", "crypto/md5"}},
		"internal/verifier/ed25519": {imports: []string{"crypto/ed25519", "crypto/subtle", "errors", "math/big"}},
		"internal/verifier/keyring": {imports: []string{"crypto/sha256", "crypto/sha512"}},
		"internal/verifier/x": {imports: []string{
			"crypto", "crypto/sha256", "crypto/sha1", "crypto/sha512", "crypto/md5", "crypto/hmac", "crypto/sha3",
			"hash", "hash/fnv", "github.com/truestamp/truestamp-cli/internal/tscrypto",
			"github.com/truestamp/truestamp-cli/internal/hashing",
		}},
	}
	want := []string{"internal/verifier may import crypto/md5", "internal/verifier/keyring may import crypto/sha512"}
	for _, p := range planted["internal/verifier/x"].imports {
		want = append(want, "internal/verifier/x may import "+p)
	}
	sort.Strings(want)
	if found := hasherImports(planted); !slices.Equal(found, want) {
		t.Errorf("the planted lists give %q, want %q", found, want)
	}
	_, files := parseModule(t, repoRoot)
	for _, f := range files {
		if dir := path.Dir(f.rel); strings.HasPrefix(dir+"/", "internal/verifier/") {
			if _, ok := scannedPackages[dir]; !ok {
				t.Errorf("%s is under internal/verifier and not among the scanned packages", dir)
			}
		}
	}
}

// contentDigestPlanted are files of the content digests' packages and of
// another package of the new verifier that use crypto/sha256: the two
// functions check 6 allows; a second function in each one's file; a
// function of each one's name in another file of its package; a use in
// another file of internal/verifier; and a function named like one in
// another package, whose import check 8 also refuses.
var contentDigestPlanted = map[string]string{
	"internal/verifier/keyring/digest.go": `package keyring

import "crypto/sha256"

func Digest(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func another(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the registry's digest"
}
`,
	"internal/verifier/keyring/keyring.go": `package keyring

import "crypto/sha256"

func Digest(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the registry's digest"
}
`,
	"internal/verifier/shape.go": `package verifier

import "crypto/sha256"

func shapeConstructions(text []byte) [32]byte {
	return sha256.Sum256(text)
}

func shapeTimes(text []byte) [32]byte {
	return sha256.Sum256(text) // want "uses crypto/sha256 outside the registry's digest"
}
`,
	"internal/verifier/layout.go": `package verifier

import "crypto/sha256"

func shapeConstructions(text []byte) [32]byte {
	return sha256.Sum256(text) // want "uses crypto/sha256 outside the registry's digest"
}
`,
	"internal/verifier/policy.go": `package verifier

import "crypto/sha256"

func Policy(text []byte) {
	_ = sha256.Sum256(text) // want "uses crypto/sha256 outside the registry's digest"
}
`,
	"internal/verifier/digest.go": `package verifier

import "crypto/sha256"

func Digest(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the registry's digest"
}
`,
	"internal/verifier/ed25519/digest.go": `package ed25519

import "crypto/sha256" // want "imports crypto/sha256, which is not among the imports of internal/verifier/ed25519"

func shapeConstructions(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the registry's digest"
}
`,
}

// Check 6 allows crypto/sha256 in exactly two functions outside the
// registry, contentDigests, each by its file and its name, and the
// module's own keyring.Digest and shapeConstructions are those functions
// and each uses it once. Mutants killed: the allowance matched by the
// function's name alone (keyring.go's Digest and layout.go's
// shapeConstructions pass); by its file alone (another and shapeTimes
// pass); by its name in every scanned package (internal/verifier's Digest
// passes its check 6 finding); or for every function of a content
// digest's package (policy.go's Policy passes); and shapeConstructions
// left off contentDigests (the planted shape.go's shapeConstructions gives
// a finding).
func TestRule2_AllowsTheTwoContentDigests(t *testing.T) {
	fset, files, want := parsePlanted(t, contentDigestPlanted, "want")
	if len(want) != 8 {
		t.Fatalf("%d planted lines, want 8", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)

	_, module := parseModule(t, repoRoot)
	uses := map[contentDigest]int{}
	for _, f := range module {
		sha256Name := importName(f.file, "crypto/sha256")
		if sha256Name == "" {
			continue
		}
		ast.Inspect(f.file, func(node ast.Node) bool {
			if sel, ok := node.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == sha256Name && inContentDigest(f, sel.Pos()) {
					uses[contentDigest{f.rel, enclosingFunc(f.file, sel.Pos())}]++
				}
			}
			return true
		})
	}
	for _, d := range contentDigests {
		if uses[d] != 1 {
			t.Errorf("%s's %s uses crypto/sha256 %d times, want once", d.file, d.function, uses[d])
		}
	}
}

// memberSetPlanted are files of internal/verifier that read a document's
// member set each way check 7 refuses, in DERIVE's files and the layout,
// and the same shapes in files check 7 does not read.
var memberSetPlanted = map[string]string{
	"internal/verifier/derive_records.go": `package verifier

import "maps" // want "imports maps"

func planted(proof map[string]any, value any, name string) {
	for member := range proof { // want "ranges over the members"
		_ = member
	}
	switch v := value.(type) {
	case map[string]any:
		for member := range v { // want "ranges over the members"
			_ = member
		}
		_ = v[name] // want "by a key that is not a constant"
		_ = v["block_hash"]
	case []any:
		for i := range v {
			_ = i
		}
	}
	_ = "fields" // want "names a registry member"
	_ = maps.Keys(proof)
}
`,
	"internal/verifier/derive_later.go": `package verifier

func later(proof map[string]any) {
	for member := range proof { // want "ranges over the members"
		_ = member
	}
}
`,
	"internal/verifier/derive_links.go": `package verifier

type segment struct{ name string }

func valueAt(value any, s segment) any {
	switch v := value.(type) {
	case map[string]any:
		return v[s.name] // want "by a key that is not a constant"
	}
	return nil
}
`,
	"internal/verifier/layout.go": `package verifier

type segment struct{ name string }

func valueAt(value any, p []segment) any {
	for _, s := range p {
		switch v := value.(type) {
		case map[string]any:
			value = v[s.name]
		case []any:
			value = v[0]
		default:
			return nil
		}
	}
	return value
}

func another(value any, s segment) any {
	switch v := value.(type) {
	case map[string]any:
		return v[s.name] // want "by a key that is not a constant"
	}
	return nil
}
`,
	"internal/verifier/shape.go": `package verifier

func shapeWitnesses(witnesses map[string]any, names map[string]any) bool {
	for name := range names {
		_ = witnesses[name]
	}
	return true
}
`,
	"internal/verifier/schema.go": `package verifier

func members(value any) {
	switch v := value.(type) {
	case map[string]any:
		for name := range v {
			_, _ = v[name], "members"
		}
	}
}
`,
}

// Check 7 reads DERIVE's files, internal/verifier/derive*.go, and the
// layout, never SHAPE's or the schema's checker's, which read a document's
// members by design; in them it refuses a range over a map parameter or
// over a type switch's binding in a map case, such a binding indexed by a
// key that is not a constant, and a member's name as a literal; and it
// allows the layout's pointer read in valueAt alone. Mutants killed: check
// 7 read in every file of internal/verifier (shape.go's and schema.go's
// ranges give findings); memberSetFiles naming derive_records.go and
// layout.go alone (derive_later.go's range passes); the pointer read
// allowed in any function of layout.go (another's index passes) or in any
// function named valueAt (derive_links.go's index passes).
func TestRule2_ReadsTheMemberSetsOfDeriveAndTheLayoutOnly(t *testing.T) {
	fset, files, want := parsePlanted(t, memberSetPlanted, "want")
	if len(want) != 8 {
		t.Fatalf("%d planted lines, want 8", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)
}

// memberSetShapesPlanted is a file of DERIVE that ranges over, and reads
// by a key that is not a constant, a map each thing that can yield one
// gives: a field, a call, a variable declared with var, an alias, a value
// of a named map type and of an alias of one, a closure's parameter, a
// package-level closure's, and a local bound from a call; and a value from
// outside the module, which the checker cannot type; beside the reads of
// the same maps check 7 allows: by a literal or a named constant, and over
// a value whose type is written.
var memberSetShapesPlanted = map[string]string{
	"internal/verifier/derive_shapes.go": `package verifier

import "strings"

type shapeNode struct{ node map[string]any }

type object map[string]any

type entries = map[string]any

const blockHashName = "block_hash"

func shapeNodes(proof map[string]any) []shapeNode { return []shapeNode{{node: proof}} }

func shapeRecord(proof map[string]any) map[string]any {
	record, _ := proof["record"].(map[string]any)
	return record
}

var later = func(m map[string]any) {
	for member := range m { // want "ranges over the members"
		_ = member
	}
}

func shapes(proof map[string]any, name string, typed object, aliased entries) {
	for _, block := range shapeNodes(proof) {
		for member := range block.node { // want "ranges over the members"
			_ = member
		}
		_ = block.node[name] // want "by a key that is not a constant"
		_, _ = block.node["block_hash"], block.node[blockHashName]
	}
	for member := range shapeRecord(proof) { // want "ranges over the members"
		_ = member
	}
	_ = shapeRecord(proof)[name] // want "by a key that is not a constant"
	var subject map[string]any = proof
	for member := range subject { // want "ranges over the members"
		_ = member
	}
	alias := proof
	for member := range alias { // want "ranges over the members"
		_ = member
	}
	for member := range typed { // want "ranges over the members"
		_ = member
	}
	for member := range aliased { // want "ranges over the members"
		_ = member
	}
	each := func(m map[string]any) {
		for member := range m { // want "ranges over the members"
			_ = member
		}
	}
	each(proof)
	beside := shapeRecord(proof)
	for member := range beside { // want "ranges over the members"
		_ = beside[member] // want "by a key that is not a constant"
	}
	for _, field := range strings.Fields(name) { // want "ranges over a value whose type the scan cannot tell"
		_ = field
	}
	var fields []string = strings.Fields(name)
	for _, field := range fields {
		_ = field
	}
	_ = strings.Fields(name)[len(name)] // want "reads a value whose type the scan cannot tell"
	_ = later
}
`,
}

// Check 7 refuses a range over a map, and a read of one by a key that is
// not a constant, whatever yields the map, from the type go/types gives
// each expression; and a range or such a read over a value the checker
// cannot type; while a read by a literal or a named constant, and a range
// over a value whose declared type is not a map, pass. Mutants killed: the
// check as it read syntax alone, the maps of a parameter, of a type
// assertion and of a type switch's binding (the ranges over a field, a
// call, a var, an alias, the named type and its alias, a closure's
// parameter, the package-level closure's and the local bound from a call,
// and the reads of a field, a call and that local, pass); a named
// constant refused as a key (blockHashName's read gives a finding); the
// named map types read by their own type, not its underlying one (typed
// and aliased pass); a value the checker cannot type let through
// (strings.Fields's range and read pass); package-level declarations left
// unread (later's range passes).
func TestRule2_RefusesAMemberSetWhateverYieldsTheMap(t *testing.T) {
	fset, files, want := parsePlanted(t, memberSetShapesPlanted, "want")
	if len(want) != 14 {
		t.Fatalf("%d planted lines, want 14", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)
}

// kindPlanted is a file of DERIVE that hashes under kinds check 11 refuses
// beside the ones it allows: a registry constant at the call, and a
// parameter declared as a registry.Construction that the package's own
// callers give a constant.
const kindPlanted = `package verifier

import "github.com/truestamp/truestamp-cli/internal/registry"

var kinds = []registry.Construction{registry.ItemHash} // want "holds a registry.Construction"

type walkOf struct{ kind registry.Construction } // want "holds a registry.Construction"

func hashOf(kind registry.Construction, other registry.Construction, record map[string]any) {
	_, _ = registry.RecordHash(kind, record)
	_, _ = registry.RecordHashHex(other, record)
	_, _ = registry.RecordHash(registry.BlockHash, record)
}

func planted(record map[string]any, w walkOf) {
	kind := registry.EpochHash
	_, _ = registry.RecordHash(kind, record) // want "gives registry.RecordHash a kind"
	_, _ = registry.RecordHash(w.kind, record) // want "gives registry.RecordHash a kind"
	_, _ = registry.RecordHashHex(kinds[0], record) // want "gives registry.RecordHashHex a kind"
	_, _ = registry.RecordHash(registry.ClaimsHash+1, record) // want "gives registry.RecordHash a kind"
	_, _ = registry.RecordHash(registry.MerkleLeaf, record) // want "gives registry.RecordHash a kind" "registry.MerkleLeaf"
	hashOf(registry.ItemHash, registry.ObservationHash, record)
	hashOf(kind, registry.ItemHash, record) // want "gives hashOf a kind"
	hashOf(registry.ItemHash, w.kind, record) // want "gives hashOf a kind"
	_ = func(k registry.Construction) {} // want "holds a registry.Construction"
}

func (w walkOf) method(kind registry.Construction, record map[string]any) { // want "holds a registry.Construction"
	_, _ = registry.RecordHash(kind, record) // want "gives registry.RecordHash a kind"
}
`

// kindTable is the registry's constants, which check 11 reads.
const kindTable = `package registry

type Construction int

const (
	MerkleLeaf Construction = iota + 1
	MerkleNode
	ClaimsHash
	ItemHash
	EntropyHash
	ObservationHash
	GenesisPreviousBlockHash
	BlockHash
	EpochHash
)
`

// Check 11: in DERIVE, and in every scanned package but the registry, each
// kind a record hash is computed under is a registry constant written at
// the call, or a parameter of the enclosing function declared as a
// registry.Construction, which the package's own callers give a constant
// or such a parameter; and no kind is held anywhere else. Mutants killed:
// any identifier accepted as the kind (the local kind passes); any
// selector accepted (w.kind passes); the package's own callers left
// unchecked (hashOf given kind passes); the type allowed anywhere (kinds
// and walkOf pass); a method's parameter accepted (the method's
// RecordHash passes).
func TestRule2_EachRecordHashKindIsAConstantAtItsCall(t *testing.T) {
	fset, files, want := parsePlanted(t, map[string]string{"internal/verifier/derive_planted.go": kindPlanted, tableFile: kindTable}, "want")
	if len(want) != 12 {
		t.Fatalf("%d planted lines, want 12", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)
}

// callPlanted are files that call each content digest at its one call and
// elsewhere, and packages that export a SHA-256 of their own.
var callPlanted = map[string]string{
	"internal/verifier/keyring/digest.go": `package keyring

import "crypto/sha256"

func Digest(data []byte) [32]byte {
	return sha256.Sum256(data)
}
`,
	"internal/verifier/keyring/keyring.go": `package keyring

type Keyring struct{ Digest [32]byte }

func Load(data []byte) *Keyring {
	loaded := &Keyring{}
	loaded.Digest = Digest(data)
	_ = Digest // want "refers to the digest Digest outside its one call"
	return &Keyring{Digest: loaded.Digest}
}

func Reload(data []byte) [32]byte {
	return Digest(data) // want "refers to the digest Digest outside its one call"
}

var later = Digest // want "refers to the digest Digest outside its one call"
`,
	"internal/verifier/policy.go": `package verifier

import (
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier/keyring"
)

func Policy(data []byte) {
	_ = keyring.Digest(data) // want "refers to the digest keyring.Digest outside its one call"
	_ = specdata.ConstructionsDigest(data)
}
`,
	"internal/verifier/shape.go": `package verifier

import "crypto/sha256"

func Shape(text []byte) {
	_ = shapeConstructions(text)
}

func shapeConstructions(text []byte) [32]byte {
	return sha256.Sum256(text)
}

func shapeTimes(text []byte) {
	_ = shapeConstructions(text) // want "refers to the digest shapeConstructions outside its one call"
}
`,
	"internal/specdata/constructions.go": `package specdata

import "crypto/sha256"

func ConstructionsDigest(text []byte) [32]byte {
	return sha256.Sum256(text) // want "uses crypto/sha256 outside the named digests of internal/specdata"
}

func acceptedDigests(text []byte) [32]byte {
	return sha256.Sum256(text)
}
`,
	"internal/jcs/jcs.go": `package jcs

import "crypto/sha256"

type Encoder struct{}

func (Encoder) Sum(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the named digests of internal/jcs"
}
`,
	"cmd/hash.go": `package cmd

import "crypto/sha256"

func Hash(data []byte) [32]byte {
	return sha256.Sum256(data)
}
`,
}

// Check 12: each named digest is referred to only at its one call,
// keyring.Digest in keyring.go's Load and shapeConstructions in shape.go's
// Shape among them, and no package the new verifier may import uses a
// SHA-256 of its own outside its named digests; a command, which no
// verifier package imports, may. Mutants killed: the call allowed in any
// function of its file (Reload passes); a reference in the one caller
// that is not a call allowed (Load's Digest taken as a value passes); a
// reference from another package left out
// (Policy's keyring.Digest passes); the use rule left out (specdata's
// ConstructionsDigest, which Policy calls, passes); the use rule reading
// functions alone (jcs's method Sum passes); the use rule read in every
// package (cmd's Hash gives a finding).
func TestRule2_EachContentDigestHasItsOneCall(t *testing.T) {
	fset, files, want := parsePlanted(t, callPlanted, "want")
	if len(want) != 7 {
		t.Fatalf("%d planted lines, want 7", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)
	for digest := range digestCalls {
		if !slices.Contains(namedDigests, digest) {
			t.Errorf("%v has a call and is not a named digest", digest)
		}
	}
	if len(digestCalls) != len(namedDigests) {
		t.Errorf("%d named digests, %d calls", len(namedDigests), len(digestCalls))
	}
}

// importedDigestPlanted are files of the packages the new verifier may
// import that hash outside their named digests each way check 12 refuses:
// an exported wrapper over an unexported helper, in internal/specdata and
// in internal/jcs, which DERIVE calls with its prefix byte written outside
// a composite literal, so that no other check sees it; an exported var
// holding a hash function; a method named like a named digest; another
// hash package's export; crypto/sha256 imported by "."; a SHA-256 in a
// package of the module only an importable package imports; and each
// named digest referred to outside its one call. A command may hash as it
// likes.
var importedDigestPlanted = map[string]string{
	"internal/specdata/constructions.go": `package specdata

import "crypto/sha256"

var accepted = mustAcceptedDigests()

func acceptedDigests(text []byte) [32]byte {
	return sha256.Sum256(text)
}

func mustAcceptedDigests() [32]byte {
	return acceptedDigests(nil)
}

func Accepted(text []byte) [32]byte {
	return acceptedDigests(text) // want "refers to the digest acceptedDigests outside its one call"
}

type Digester struct{}

func (Digester) acceptedDigests(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the named digests of internal/specdata"
}
`,
	"internal/specdata/sumof.go": `package specdata

import "crypto/sha256"

func SumOf(data []byte) [32]byte {
	return sumOf(data)
}

func sumOf(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the named digests of internal/specdata"
}

var Hasher = sha256.Sum256 // want "uses crypto/sha256 outside the named digests of internal/specdata"
`,
	"internal/specdata/sum512.go": `package specdata

import "crypto/sha512" // want "imports crypto/sha512, a hash package, into internal/specdata"

func Sum512(data []byte) [64]byte {
	return sha512.Sum512(data)
}
`,
	"internal/jcs/jcs.go": `package jcs

import (
	"crypto/sha256"

	"github.com/truestamp/truestamp-cli/internal/textutil"
)

func Canonicalize(data []byte) string {
	return placeholderFor(data)
}

func placeholderFor(data []byte) string {
	sum := sha256.Sum256(data)
	return string(sum[:])
}

func Transform(data []byte) string {
	return placeholderFor(data) // want "refers to the digest placeholderFor outside its one call"
}

func SumOf(data []byte) [32]byte {
	return sumOf(data)
}

func sumOf(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the named digests of internal/jcs"
}

var Text = textutil.Sum
`,
	"internal/textutil/sum.go": `package textutil

import "crypto/sha256"

func Sum(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the named digests of internal/textutil"
}
`,
	"internal/strictjson/hash.go": `package strictjson

import (
	. "crypto/sha256" // want "imports crypto/sha256 by"
	"hash/fnv" // want "imports hash/fnv, a hash package, into internal/strictjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash" // want "imports github.com/btcsuite/btcd/chaincfg/chainhash, a dependency not on importableDependencies"
	"golang.org/x/crypto/sha3" // want "imports golang.org/x/crypto/sha3, a dependency not on importableDependencies"
	"github.com/knadh/koanf/v2"
)

func Sum(data []byte) ([32]byte, uint32) {
	h := fnv.New32()
	_, _ = h.Write(data)
	_, _, _ = chainhash.DoubleHashH(data), sha3.Sum256(data), koanf.New(".")
	return Sum256(data), h.Sum32()
}
`,
	"internal/verifier/derive_records.go": `package verifier

import (
	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
)

func itemHash(encoded []byte) ([32]byte, [32]byte, [32]byte, [64]byte, [32]byte) {
	preimage := append(append([]byte(nil), 17), encoded...)
	return specdata.SumOf(preimage), jcs.SumOf(preimage), specdata.Hasher(preimage), specdata.Sum512(preimage), jcs.Text(preimage)
}
`,
	"cmd/hash.go": `package cmd

import (
	"crypto/sha256"
	"crypto/sha512"
)

func Hash(data []byte) ([32]byte, [64]byte) {
	return sha256.Sum256(data), sha512.Sum512(data)
}
`,
}

// Check 12 holds each package of the module the new verifier may import,
// directly or through another such package, and not itself scanned, to
// its named digests: crypto/sha256 in no other function, method or
// declaration, exported or not, and no other hash package; DERIVE's own
// files then show nothing, since the hash is the importable package's.
// The module's own importedDigests each use crypto/sha256 and have their
// one call, and the packages internal/verifier lists are importable.
// Mutants killed: the rule as it was, an exported function's own use of
// crypto/sha256 alone (the wrappers' sumOf in specdata and jcs, Hasher,
// Digester's method and the import of crypto/sha512 and hash/fnv pass);
// the use rule reading functions alone (Hasher passes); a method taken
// for a named digest by its name (Digester's acceptedDigests passes);
// importableDirs not following an importable package's imports
// (textutil's Sum passes); the hash-import rule left out (sum512.go's and
// hash.go's imports pass); the dot import let through (hash.go's
// crypto/sha256 passes); the dependency rule left out (hash.go's chainhash
// and golang.org/x/crypto/sha3 imports pass, while its koanf import is on
// the list); acceptedDigests or placeholderFor left off importedDigests
// (their own uses give findings).
func TestRule2_TheImportablePackagesHashOnlyInTheirNamedDigests(t *testing.T) {
	fset, files, want := parsePlanted(t, importedDigestPlanted, "want")
	if len(want) != 12 {
		t.Fatalf("%d planted lines, want 12", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)

	_, module := parseModule(t, repoRoot)
	importable := importableDirs(module, scannedPackages)
	for _, dir := range []string{"internal/config", "internal/httpclient", "internal/jcs", "internal/specdata", "internal/strictjson"} {
		if !importable[dir] {
			t.Errorf("%s is not among the packages the new verifier may import", dir)
		}
	}
	uses, calls := map[contentDigest]int{}, map[contentDigest]int{}
	for _, f := range module {
		sha256Name := importName(f.file, "crypto/sha256")
		ast.Inspect(f.file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.SelectorExpr:
				if x, ok := n.X.(*ast.Ident); ok && sha256Name != "" && x.Name == sha256Name && inImportedDigest(f, n.Pos()) {
					uses[contentDigest{f.rel, enclosingFunc(f.file, n.Pos())}]++
				}
			case *ast.CallExpr:
				for _, d := range importedDigests {
					if ident, ok := n.Fun.(*ast.Ident); ok && ident.Name == d.function && path.Dir(f.rel) == path.Dir(d.file) &&
						(contentDigest{f.rel, enclosingFunc(f.file, n.Pos())}) == digestCalls[d] {
						calls[d]++
					}
				}
			}
			return true
		})
	}
	for _, d := range importedDigests {
		if uses[d] == 0 || calls[d] != 1 {
			t.Errorf("%s's %s uses crypto/sha256 %d times and has %d calls in %v, want some and one", d.file, d.function, uses[d], calls[d], digestCalls[d])
		}
	}
}
