// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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
// Checks 6 and 8 read only the packages of scannedPackages, the new
// verifier's, and check 7 only those of them it marks: older code, such as
// internal/tscrypto's walk, still applies 0x00 and 0x01 itself.
//
//  6. crypto/sha256 used outside the registry's digest, the one function
//     that applies a prefix byte, and outside contentDigest, the one
//     function of the new verifier that makes a content digest (SPEC.md
//     sections 4.1 and 8.3: trust_root's digest of a keyring's bytes as
//     read); in internal/registry, digest given a
//     construction that is not a registry constant (or RecordHash's
//     checked kind), or the leaf or node; a hex integer literal, or a byte
//     slice or array literal holding a literal, outside the table; and an
//     import by ".", which hides its uses from these checks;
//  7. in the registry, and in DERIVE and its layout once they are built,
//     as the reference's derive_test.exs scans them (never SHAPE, the gate,
//     strictjson or jcs, which read a document's members by design), a
//     member set read from a value: the maps or reflect packages, a range
//     over a map parameter or over a map a type assertion gives, such a map
//     indexed by a name that is not a string literal, and a string literal
//     named like a registry member ("byte", "prefix", "prefixes",
//     "members", "fields"), as part 5 of the app's rule bans;
//  8. an import that is not on the package's list in scannedPackages, so
//     that nothing in internal/registry hashes but digest and the walk, and
//     nothing in another package of the new verifier hashes at all: no
//     list but the registry's holds crypto/sha256, and none holds a hash
//     package (crypto, crypto/sha1, crypto/sha512, crypto/md5,
//     crypto/hmac, crypto/sha3, hash or a hash/ package), internal/tscrypto
//     or internal/hashing, whose generic hashers and prefix tables the
//     other checks do not see. Another crypto/ package, such as
//     crypto/ed25519, may be listed, and so may crypto/sha256 on the list
//     of contentDigest's package, which check 6 confines to that one
//     function.
//
// go/ast sees syntax, not data flow, so it cannot see a byte or a name that
// reaches these sites through a variable or a call; a planted file of each
// shape shows each check firing.

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
}

// scannedPackages are the packages of the new verifier that checks 6 to 8
// read, by their slash path from the module root. A package of the new
// verifier joins the list in the commit that adds it; check 9 fails one
// that imports internal/registry without being on it.
var scannedPackages = map[string]scannedPackage{
	registryDir: {imports: registryImports, memberSets: true},
	"internal/verifier": {imports: []string{
		"cmp", "encoding/hex", "errors", "fmt", "net/url", "slices", "strconv", "strings", "unicode/utf16",
		"github.com/spf13/pflag",
		"github.com/truestamp/truestamp-cli/internal/config",
		registryPath, specdataPath, strictjsonPath,
		"github.com/truestamp/truestamp-cli/internal/verifier/ed25519",
		"github.com/truestamp/truestamp-cli/internal/verifier/keyring",
	}},
	"internal/verifier/keyring": {imports: []string{
		"crypto/sha256", "encoding/hex", "errors", "fmt", "regexp", "slices",
		registryPath, specdataPath, strictjsonPath,
		"github.com/truestamp/truestamp-cli/internal/verifier/ed25519",
	}},
	"internal/verifier/ed25519": {imports: []string{"crypto/ed25519", "errors", "math/big", specdataPath}},
}

// contentDigest is the one function of the new verifier outside the
// registry that may use crypto/sha256: keyring.Digest, trust_root's digest
// d, the SHA-256 of a keyring's bytes as read, a content digest SPEC.md
// section 4.1 puts outside the registry (section 8.3). Check 6 allows the
// use in that function of that file and nowhere else in its package, and
// check 8 lets that package alone list crypto/sha256 beside the registry.
var contentDigest = struct{ file, function string }{"internal/verifier/keyring/digest.go", "Digest"}

// inContentDigest reports whether pos, in f, lies in contentDigest.
func inContentDigest(f goFile, pos token.Pos) bool {
	return f.rel == contentDigest.file && enclosingFunc(f.file, pos) == contentDigest.function
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

func isMapAssertion(expr ast.Expr) bool {
	assertion, ok := expr.(*ast.TypeAssertExpr)
	if !ok {
		return false
	}
	_, isMap := assertion.Type.(*ast.MapType)
	return isMap
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
	for _, f := range files {
		dir := path.Dir(f.rel)
		inRegistry := dir == registryDir
		pkg, inScanned := scanned[dir]
		memberSets := inScanned && pkg.memberSets
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
		if memberSets {
			out = append(out, memberSetFindings(fset, f)...)
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

// memberSetFindings reports, in each function of f, a range over a map
// parameter or a map a type assertion gives, and such a map indexed by a
// name that is not a string literal.
func memberSetFindings(fset *token.FileSet, f goFile) []finding {
	var out []finding
	for _, decl := range f.file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		maps := map[string]bool{}
		for _, field := range fn.Type.Params.List {
			if _, isMap := field.Type.(*ast.MapType); isMap {
				for _, name := range field.Names {
					maps[name.Name] = true
				}
			}
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if assign, ok := node.(*ast.AssignStmt); ok && len(assign.Rhs) == 1 && isMapAssertion(assign.Rhs[0]) {
				if name, ok := assign.Lhs[0].(*ast.Ident); ok {
					maps[name.Name] = true
				}
			}
			return true
		})
		isDocumentMap := func(expr ast.Expr) bool {
			if ident, ok := expr.(*ast.Ident); ok {
				return maps[ident.Name]
			}
			return isMapAssertion(expr)
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.RangeStmt:
				if isDocumentMap(n.X) {
					out = append(out, finding{rel: f.rel, line: fset.Position(n.Pos()).Line, what: "ranges over the members of an object"})
				}
			case *ast.IndexExpr:
				if literal, ok := n.Index.(*ast.BasicLit); isDocumentMap(n.X) && (!ok || literal.Kind != token.STRING) {
					out = append(out, finding{rel: f.rel, line: fset.Position(n.Pos()).Line, what: "reads a member of an object by a name that is not a literal"})
				}
			}
			return true
		})
	}
	return out
}

// The module holds to checks 1 to 5 and 9, and the scanned packages to
// checks 6 to 8. The scan is not vacuous: it reads the commands and every
// internal package, each scanned package, the walk's own use of the
// library, and digest. Mutants killed: a 0x01 literal hashed in key_id.go;
// the library's hasher named in a second function of walk.go; a
// specdata.PrefixByte call added to cmd/hash.go; a scanned package's path
// misspelt in scannedPackages.
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
	_ = object[name] // want "by a name that is not a literal"
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
// CLI's hash command's registry. Each import is off the package's list.
const verifierPlanted = `package verifier

import (
	"crypto" // want "imports crypto, which is not among the imports of internal/verifier"
	"crypto/sha256" // want "imports crypto/sha256, which is not among the imports of internal/verifier"
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
	if len(want) != 8 {
		t.Fatalf("%d planted lines, want 8", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)
}

// hasherPackages are the hash packages no package of the new verifier but
// the registry may import (check 8): the generic crypto package, each
// hash of the standard library, and the CLI's own hashers.
var hasherPackages = []string{
	"crypto", "crypto/sha256", "crypto/sha1", "crypto/sha512", "crypto/md5", "crypto/hmac", "crypto/sha3",
	"github.com/truestamp/truestamp-cli/internal/tscrypto", "github.com/truestamp/truestamp-cli/internal/hashing",
}

// hasher reports whether p is a hash package: one of hasherPackages, hash
// or a hash/ package.
func hasher(p string) bool {
	return slices.Contains(hasherPackages, p) || p == "hash" || strings.HasPrefix(p, "hash/")
}

// hasherImports names each hash package a list of packages other than the
// registry's allows, but crypto/sha256 on the list of contentDigest's
// package.
func hasherImports(packages map[string]scannedPackage) []string {
	var found []string
	for dir, pkg := range packages {
		if dir == registryDir {
			continue
		}
		for _, p := range pkg.imports {
			if hasher(p) && !(dir == path.Dir(contentDigest.file) && p == "crypto/sha256") {
				found = append(found, dir+" may import "+p)
			}
		}
	}
	sort.Strings(found)
	return found
}

// No list of a package of the new verifier but the registry's names a
// hash package, but contentDigest's package's crypto/sha256: crypto/sha256
// is otherwise the registry's alone, and crypto, the other hashes of
// crypto/, hash and the hash/ packages, internal/tscrypto and
// internal/hashing are no one's; another crypto/ package, such as
// crypto/ed25519, may be listed; and every package under internal/verifier
// is on the list. Mutants killed: crypto/sha256 added to
// internal/verifier's list; hasher banning every crypto/ package (the
// planted list's crypto/ed25519 is named); the content digest's allowance
// made for every package (the planted internal/verifier/x's crypto/sha256
// is not named) or for every hash package of its own (the planted
// keyring's crypto/sha512 is not named).
func TestRule2_NoOtherPackageOfTheNewVerifierMayImportAHasher(t *testing.T) {
	if found := hasherImports(scannedPackages); len(found) != 0 {
		t.Errorf("hash packages listed: %q", found)
	}
	planted := map[string]scannedPackage{
		registryDir:                 {imports: []string{"crypto/sha256"}},
		"internal/verifier/ed25519": {imports: []string{"crypto/ed25519", "crypto/subtle", "errors", "math/big"}},
		"internal/verifier/keyring": {imports: []string{"crypto/sha256", "crypto/sha512"}},
		"internal/verifier/x": {imports: []string{
			"crypto", "crypto/sha256", "crypto/sha1", "crypto/sha512", "crypto/md5", "crypto/hmac", "crypto/sha3",
			"hash", "hash/fnv", "github.com/truestamp/truestamp-cli/internal/tscrypto",
			"github.com/truestamp/truestamp-cli/internal/hashing",
		}},
	}
	want := []string{"internal/verifier/keyring may import crypto/sha512"}
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

// contentDigestPlanted are files of the content digest's package and of
// another package of the new verifier that use crypto/sha256: the one
// function check 6 allows, a second function in its file, a function of its
// name in another file of its package, and one of its name in another
// package, whose import check 8 also refuses.
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
	"internal/verifier/digest.go": `package verifier

import "crypto/sha256" // want "imports crypto/sha256, which is not among the imports of internal/verifier"

func Digest(data []byte) [32]byte {
	return sha256.Sum256(data) // want "uses crypto/sha256 outside the registry's digest"
}
`,
}

// Check 6 allows crypto/sha256 in exactly one function outside the
// registry, contentDigest, by its file and its name, and the module's own
// keyring.Digest is that function and uses it. Mutants killed: the
// allowance matched by the function's name alone (keyring.go's Digest
// passes), by its file alone (another passes), or in every scanned package
// by its name (internal/verifier's Digest passes its check 6 finding).
func TestRule2_AllowsTheOneContentDigest(t *testing.T) {
	fset, files, want := parsePlanted(t, contentDigestPlanted, "want")
	if len(want) != 4 {
		t.Fatalf("%d planted lines, want 4", len(want))
	}
	assertFindings(t, rule2Findings(fset, files, scannedPackages), want)

	_, module := parseModule(t, repoRoot)
	uses := 0
	for _, f := range module {
		if f.rel != contentDigest.file {
			continue
		}
		sha256Name := importName(f.file, "crypto/sha256")
		ast.Inspect(f.file, func(node ast.Node) bool {
			if sel, ok := node.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == sha256Name && inContentDigest(f, sel.Pos()) {
					uses++
				}
			}
			return true
		})
	}
	if uses != 1 {
		t.Fatalf("%s's %s uses crypto/sha256 %d times, want once", contentDigest.file, contentDigest.function, uses)
	}
}
