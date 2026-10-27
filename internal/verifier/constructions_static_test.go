// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// SPEC.md section 10.6, rule 1: constructions is read once, at SHAPE, and
// only compared with CONSTRUCTIONS_ACCEPTED (section 7.2). This static
// test is the test that section 7.2, item 2, asks of each of Truestamp's
// verifiers: no file of the program but SHAPE's names the member, and SHAPE
// reads it once. Its scope is the whole program: every non-test Go file of
// the module, the commands, the shared packages and the verifier's own,
// each type-checked (module_static_test.go). It finds every use of the
// member's name: a string literal "constructions", or any expression whose
// constant value is that string, so a key built by concatenation, in
// parentheses or from a constant of any package of the module is seen. The
// exemptions are two. internal/specdata/constructions.go's constant
// constructionsDir, the directory of the texts under spec/ (section 9), its
// declaration and its one use, as an argument of the call of
// acceptedDigests in mustAcceptedDigests, matched by the file, the
// constant's name and that call. And internal/proof/download.go's constant
// ConstructionsParameter, the name of the raw proof route's query
// parameter and of the `proofs get` flag that sets it, which name a
// request's parameter and never the proof's member: its declaration in
// that file, and each use of the constant itself, by its name, that is not
// a read, a key or a part, in the three files that name the parameter or
// the flag, download.go, cmd/proofs_get.go and cmd/schema.go, each with
// its exact count. A use of the constant in any other file, a comparison,
// a case or an accessor's argument as much as a read, is refused. A name
// built at run time, from
// values that are not constants, is out of the test's sight; planted files
// show each shape it refuses.

// constructionsMember is the member's name.
const constructionsMember = "constructions"

// The exemption: the constant constructionsDir of its file, and its one
// use, an argument of the call of acceptedDigests in mustAcceptedDigests.
const (
	constructionsDirFile   = "internal/specdata/constructions.go"
	constructionsDirName   = "constructionsDir"
	constructionsDirCaller = "mustAcceptedDigests"
	constructionsDirCallee = "acceptedDigests"
)

// The second exemption: the constant ConstructionsParameter of its file,
// in internal/proof, and each use of it by name that is not a read, a key
// or a part, in the files of constructionsParameterUses alone.
const (
	constructionsParameterFile = "internal/proof/download.go"
	constructionsParameterName = "ConstructionsParameter"
)

// constructionsParameterUses is each file whose uses of
// ConstructionsParameter by name are exempt, with the exact count of
// them: download.go's declaration and its use as the query parameter,
// cmd/proofs_get.go's two as the flag's name, where the flag is read and
// where it is declared, and cmd/schema.go's one, the key of the flag's
// values in shell completion.
var constructionsParameterUses = map[string]int{
	constructionsParameterFile: 2,
	"cmd/proofs_get.go":        2,
	"cmd/schema.go":            1,
}

// constructionsParameter is the object of the constant
// ConstructionsParameter of internal/proof in m, or nil when m has no such
// constant.
func constructionsParameter(m *typedModule) types.Object {
	pkg := m.packages[modulePath+"/internal/proof"]
	if pkg == nil {
		return nil
	}
	if object, ok := pkg.Scope().Lookup(constructionsParameterName).(*types.Const); ok {
		return object
	}
	return nil
}

// refersTo reports whether e is object by its name: an identifier, a
// package-qualified selector or either in parentheses.
func refersTo(m *typedModule, e ast.Expr, object types.Object) bool {
	if object == nil {
		return false
	}
	switch x := e.(type) {
	case *ast.Ident:
		return m.info.Uses[x] == object
	case *ast.SelectorExpr:
		return m.info.Uses[x.Sel] == object
	case *ast.ParenExpr:
		return refersTo(m, x.X, object)
	}
	return false
}

// memberUses sorts the uses of the member's name in one file:
//
//   - reads: an index of a map, as proof["constructions"], each with the
//     function it sits in;
//   - keys: a key of a map literal, as a row's values {"constructions":
//     "full"};
//   - parts: the argument of Name in a Pointer literal given as a field
//     part, as a failure's part Pointer{Name("constructions")};
//   - exempt: the declaration of constructionsDir in its file, and its one
//     use there, as an argument of acceptedDigests in mustAcceptedDigests;
//     the declaration of ConstructionsParameter in its file, and each use
//     of it by name that is none of the three above, in a file of
//     constructionsParameterUses;
//   - unclassified: any other place, such as a constant, a comparison, a
//     case, or a Name given to valueAt, each a read this test cannot
//     classify, and so refuses, and every use of ConstructionsParameter
//     by name in any other file.
type memberUses struct {
	reads                      []string
	keys, parts, exempt, other int
}

// isMemberName reports whether e is the member's name: a string literal
// "constructions", or an expression m's type check gives that constant
// value.
func isMemberName(m *typedModule, e ast.Expr) bool {
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		if value, err := strconv.Unquote(lit.Value); err == nil && value == constructionsMember {
			return true
		}
	}
	tv, ok := m.info.Types[e]
	return ok && tv.Value != nil && tv.Value.Kind() == constant.String && constant.StringVal(tv.Value) == constructionsMember
}

// classifyMemberUses sorts each use of the member's name in f, one of m's
// files. A use is the largest expression that is the name, so the
// literals of a concatenation are not counted again.
func classifyMemberUses(m *typedModule, f moduleFile) memberUses {
	var uses memberUses
	classified := map[ast.Expr]bool{}
	exempt := map[ast.Expr]bool{}
	var dir types.Object
	if f.rel == constructionsDirFile {
		for _, decl := range f.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value := spec.(*ast.ValueSpec)
				for i, name := range value.Names {
					if name.Name == constructionsDirName && i < len(value.Values) {
						exempt[value.Values[i]] = true
						dir = m.info.Defs[name]
					}
				}
			}
		}
	}
	parameter := constructionsParameter(m)
	_, parameterFile := constructionsParameterUses[f.rel]
	if f.rel == constructionsParameterFile && parameter != nil {
		for _, decl := range f.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value := spec.(*ast.ValueSpec)
				for i, name := range value.Names {
					if m.info.Defs[name] == parameter && i < len(value.Values) {
						exempt[value.Values[i]] = true
					}
				}
			}
		}
	}
	use := map[*ast.Ident]bool{}
	for _, decl := range f.file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if dir == nil || !ok || fn.Name.Name != constructionsDirCaller || fn.Recv != nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if callee, ok := call.Fun.(*ast.Ident); ok && callee.Name == constructionsDirCallee {
					for _, arg := range call.Args {
						if ident, ok := arg.(*ast.Ident); ok && m.info.Uses[ident] == dir {
							use[ident] = true
						}
					}
				}
			}
			return true
		})
	}
	for _, decl := range f.file.Decls {
		function := ""
		if fn, ok := decl.(*ast.FuncDecl); ok {
			function = fn.Name.Name
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.IndexExpr:
				if isMemberName(m, x.Index) {
					uses.reads = append(uses.reads, function)
					classified[x.Index] = true
				}
			case *ast.CompositeLit:
				if _, isMap := x.Type.(*ast.MapType); isMap {
					for _, element := range x.Elts {
						if kv, ok := element.(*ast.KeyValueExpr); ok && isMemberName(m, kv.Key) {
							uses.keys++
							classified[kv.Key] = true
						}
					}
				}
			case *ast.KeyValueExpr:
				key, ok := x.Key.(*ast.Ident)
				pointer, isLit := x.Value.(*ast.CompositeLit)
				if !ok || key.Name != "part" || !isLit {
					return true
				}
				if typ, ok := pointer.Type.(*ast.Ident); !ok || typ.Name != "Pointer" || len(pointer.Elts) != 1 {
					return true
				}
				if call, ok := pointer.Elts[0].(*ast.CallExpr); ok && len(call.Args) == 1 {
					if fn, ok := call.Fun.(*ast.Ident); ok && fn.Name == "Name" && isMemberName(m, call.Args[0]) {
						uses.parts++
						classified[call.Args[0]] = true
					}
				}
			}
			return true
		})
	}
	ast.Inspect(f.file, func(n ast.Node) bool {
		e, ok := n.(ast.Expr)
		if !ok || !isMemberName(m, e) {
			return true
		}
		ident, isIdent := e.(*ast.Ident)
		switch {
		case classified[e]:
		case exempt[e] || isIdent && use[ident] || parameterFile && refersTo(m, e, parameter):
			uses.exempt++
		default:
			uses.other++
		}
		return false
	})
	return uses
}

// No non-test file of the module but internal/verifier/shape.go uses the
// member's name, beside constructionsDir's exemption, whose declaration and
// one use are both in place, and ConstructionsParameter's, declared in its
// file and used there once, as the query parameter, twice in
// cmd/proofs_get.go, as the flag's name, and once in cmd/schema.go, as the
// key of the flag's values; no other file has an exempt use, and each of
// those files has exactly its count; and shape.go reads the member exactly
// once, at check 6 (shapeConstructions), every other use there being a key
// of a row's values or the part of a failure. Mutants killed: a second use of
// constructionsDir in specdata (its file's uses are not the two); a
// specdata use of constructionsDir outside mustAcceptedDigests's call (an
// unclassified use); Policy reading the member (policy.go holds the
// name), by a literal, a concatenation ("construct"+"ions") or a constant
// of internal/specdata naming constructionsDir; a second read in Shape,
// by a literal or a parenthesized one (two reads); a read through valueAt
// with Pointer{Name("constructions")} (an unclassified use); a read in
// cmd/ or in internal/registry/ (the file holds the name); the walk held
// to internal/verifier (it finds no internal/registry/walk.go or
// cmd/verify.go, and those two reads pass); the flag named by a literal
// in cmd/proofs_get.go (the file holds the name); a comparison, a case
// and an accessor's argument through proof.ConstructionsParameter planted
// in a file of internal/verifier (the file holds the name), and with the
// exemption covering every file as well (the file has exempt uses); a
// second use of ConstructionsParameter in cmd/schema.go (its count is not
// one).
func TestConstructions_ReadOnlyAtShape(t *testing.T) {
	m := typedModuleOf(t)
	byPath := map[string]moduleFile{}
	for _, f := range m.files {
		byPath[f.rel] = f
	}
	for _, rel := range []string{"internal/verifier/shape.go", "internal/verifier/policy.go", "internal/verifier/keyring/keyring.go",
		"internal/registry/walk.go", constructionsDirFile, constructionsParameterFile, "internal/jcs/jcs.go", "cmd/verify.go",
		"cmd/proofs_get.go", "cmd/schema.go", "cmd/truestamp/main.go"} {
		if _, ok := byPath[rel]; !ok {
			t.Fatalf("the walk found %d files, without %s", len(m.files), rel)
		}
	}
	var holders []string
	for _, f := range m.files {
		uses := classifyMemberUses(m, f)
		if len(uses.reads)+uses.keys+uses.parts+uses.other > 0 {
			holders = append(holders, f.rel)
		}
		exempt := constructionsParameterUses[f.rel]
		if f.rel == constructionsDirFile {
			exempt = 2
		}
		if uses.exempt != exempt || exempt > 0 && len(uses.reads)+uses.keys+uses.parts+uses.other > 0 {
			t.Errorf("%s: %+v, want %d exempt uses and no other", f.rel, uses, exempt)
		}
	}
	if !slices.Equal(holders, []string{"internal/verifier/shape.go"}) {
		t.Errorf("files using the name %q: %q, want internal/verifier/shape.go alone", constructionsMember, holders)
	}
	uses := classifyMemberUses(m, byPath["internal/verifier/shape.go"])
	if !slices.Equal(uses.reads, []string{"shapeConstructions"}) || uses.other != 0 || uses.keys == 0 || uses.parts == 0 {
		t.Errorf("shape.go: reads in %q, %d keys, %d parts, %d unclassified; want one read, in shapeConstructions", uses.reads, uses.keys, uses.parts, uses.other)
	}
}

// specdataImport is the import of internal/specdata, for the planted
// files.
const specdataImport = `import "` + modulePath + `/internal/specdata"`

// constructionsPlanted are files whose uses of "constructions" the
// classifier must sort, by their slash path from a module root, each with
// the uses it holds; a file want does not name holds none.
var constructionsPlanted = []struct {
	name    string
	sources map[string]string
	want    map[string]memberUses
}{
	{"the shapes shape.go uses", map[string]string{"internal/p/p.go": `package p
func shapeConstructions(proof map[string]any) (map[string]any, *f) {
	value, present := proof["constructions"]
	_ = value
	_ = present
	return map[string]any{"constructions": "none"}, &f{part: Pointer{Name("constructions")}}
}`}, map[string]memberUses{"internal/p/p.go": {reads: []string{"shapeConstructions"}, keys: 1, parts: 1}}},
	{"a second read", map[string]string{"internal/p/p.go": `package p
func a(proof map[string]any) { _ = proof["constructions"] }
func b(proof map[string]any) { _, _ = proof["constructions"] }`}, map[string]memberUses{"internal/p/p.go": {reads: []string{"a", "b"}}}},
	{"a read through valueAt", map[string]string{"internal/p/p.go": `package p
func a(proof map[string]any) { _ = valueAt(proof, Pointer{Name("constructions")}) }`}, map[string]memberUses{"internal/p/p.go": {other: 1}}},
	{"a constant", map[string]string{"internal/p/p.go": `package p
const c = "constructions"`}, map[string]memberUses{"internal/p/p.go": {other: 1}}},
	{"a comparison over the proof's names", map[string]string{"internal/p/p.go": `package p
func a(proof map[string]any) {
	for name := range proof {
		if name == "constructions" {
		}
	}
}`}, map[string]memberUses{"internal/p/p.go": {other: 1}}},
	{"a case", map[string]string{"internal/p/p.go": `package p
func a(name string) {
	switch name {
	case "constructions":
	}
}`}, map[string]memberUses{"internal/p/p.go": {other: 1}}},
	{"a part given to another field", map[string]string{"internal/p/p.go": `package p
var x = f{at: Pointer{Name("constructions")}}`}, map[string]memberUses{"internal/p/p.go": {other: 1}}},
	{"other words", map[string]string{"internal/p/p.go": `package p
var a = "constructions text"
var b = "https://www.truestamp.com/.well-known/constructions-"`}, nil},
	{"a key built by concatenation", map[string]string{"internal/p/p.go": `package p
func a(proof map[string]any) { _ = proof["construct"+"ions"] }`}, map[string]memberUses{"internal/p/p.go": {reads: []string{"a"}}}},
	{"a key in parentheses", map[string]string{"internal/p/p.go": `package p
func a(proof map[string]any) { _ = proof[("constructions")] }`}, map[string]memberUses{"internal/p/p.go": {reads: []string{"a"}}}},
	{"a key from a constant of the file", map[string]string{"internal/p/p.go": `package p
const member = "construct" + "ions"
func a(proof map[string]any) { _ = proof[member] }`}, map[string]memberUses{"internal/p/p.go": {reads: []string{"a"}, other: 1}}},
	{"a key from a constant of another package", map[string]string{
		"internal/specdata/member.go": `package specdata
const Member = "construct" + "ions"`,
		"internal/p/p.go": `package p
` + specdataImport + `
func a(proof map[string]any) { _ = proof[specdata.Member] }`,
	}, map[string]memberUses{"internal/specdata/member.go": {other: 1}, "internal/p/p.go": {reads: []string{"a"}}}},
	{"a read in a function declared twice, as in two files for two systems", map[string]string{
		"internal/p/p_unix.go": `package p
func a(proof map[string]any) { _ = proof["construct"+"ions"] }`,
		"internal/p/p_windows.go": `package p
func a(proof map[string]any) { _ = proof["construct"+"ions"] }`,
	}, map[string]memberUses{"internal/p/p_unix.go": {reads: []string{"a"}}, "internal/p/p_windows.go": {reads: []string{"a"}}}},
	{"constructionsDir and its one use", map[string]string{constructionsDirFile: `package specdata
const constructionsDir = "constructions"
func acceptedDigests(fsys any, dir string) {}
func mustAcceptedDigests() { acceptedDigests(nil, constructionsDir) }`}, map[string]memberUses{constructionsDirFile: {exempt: 2}}},
	{"constructionsDir's other uses in its file", map[string]string{
		constructionsDirFile: `package specdata
const constructionsDir = "constructions"
const Member = constructionsDir
func open(dir string) {}
func acceptedDigests(fsys any, dir string) {}
func load() { open(constructionsDir) }
func mustAcceptedDigests() { open(constructionsDir); acceptedDigests(nil, constructionsDir) }
func other() { acceptedDigests(nil, constructionsDir) }`,
		"internal/p/p.go": `package p
` + specdataImport + `
func a(proof map[string]any) { _ = proof[specdata.Member] }`,
	}, map[string]memberUses{constructionsDirFile: {exempt: 2, other: 4}, "internal/p/p.go": {reads: []string{"a"}}}},
	{"a read through constructionsDir in its file", map[string]string{constructionsDirFile: `package specdata
const constructionsDir = "constructions"
func a(proof map[string]any) { _ = proof[constructionsDir] }`}, map[string]memberUses{constructionsDirFile: {reads: []string{"a"}, exempt: 1}}},
	{"another constant in constructionsDir's file", map[string]string{constructionsDirFile: `package specdata
const constructionsDir = "constructions"
const member = "constructions"`}, map[string]memberUses{constructionsDirFile: {exempt: 1, other: 1}}},
	{"a constructionsDir in another file", map[string]string{"internal/registry/dir.go": `package registry
const constructionsDir = "constructions"
func open(dir string) {}
func load() { open(constructionsDir) }`}, map[string]memberUses{"internal/registry/dir.go": {other: 2}}},
	{"ConstructionsParameter and its uses by name in its three files", map[string]string{
		constructionsParameterFile: `package proof
const ConstructionsParameter = "constructions"
func query(name string) {}
func get() { query(ConstructionsParameter) }`,
		"cmd/proofs_get.go": `package cmd
` + proofImport + `
func flag(name string) {}
func a() { flag(proof.ConstructionsParameter); flag((proof.ConstructionsParameter)) }`,
		"cmd/schema.go": `package cmd
` + proofImport + `
type values map[string][]string
var completion = values{proof.ConstructionsParameter: nil}`,
	}, map[string]memberUses{constructionsParameterFile: {exempt: 2}, "cmd/proofs_get.go": {exempt: 2}, "cmd/schema.go": {exempt: 1}}},
	{"reads through ConstructionsParameter in a third file", map[string]string{
		constructionsParameterFile: `package proof
const ConstructionsParameter = "constructions"`,
		"internal/verifier/read.go": `package verifier
` + proofImport + `
type object map[string]any
func (o object) value(name string) any { return o[name] }
func a(doc object) any {
	for k, v := range doc {
		if k == proof.ConstructionsParameter {
			return v
		}
		switch k {
		case proof.ConstructionsParameter:
			return v
		}
	}
	return doc.value(proof.ConstructionsParameter)
}`,
	}, map[string]memberUses{constructionsParameterFile: {exempt: 1}, "internal/verifier/read.go": {other: 3}}},
	{"a read, a key and a part through ConstructionsParameter", map[string]string{
		constructionsParameterFile: `package proof
const ConstructionsParameter = "constructions"`,
		"internal/p/p.go": `package p
` + proofImport + `
func a(doc map[string]any) { _ = doc[proof.ConstructionsParameter] }
var b = map[string]any{proof.ConstructionsParameter: 1}
var c = f{part: Pointer{Name(proof.ConstructionsParameter)}}`,
	}, map[string]memberUses{constructionsParameterFile: {exempt: 1}, "internal/p/p.go": {reads: []string{"a"}, keys: 1, parts: 1}}},
	{"another constant in ConstructionsParameter's file", map[string]string{constructionsParameterFile: `package proof
const ConstructionsParameter = "constructions"
const member = "construct" + "ions"`}, map[string]memberUses{constructionsParameterFile: {exempt: 1, other: 1}}},
	{"a ConstructionsParameter in another package", map[string]string{
		constructionsParameterFile: `package proof
const ConstructionsParameter = "constructions"`,
		"internal/registry/parameter.go": `package registry
const ConstructionsParameter = "constructions"
func flag(name string) {}
func a() { flag(ConstructionsParameter) }`,
	}, map[string]memberUses{constructionsParameterFile: {exempt: 1}, "internal/registry/parameter.go": {other: 2}}},
}

// proofImport is the import of internal/proof, for the planted files.
const proofImport = `import "` + modulePath + `/internal/proof"`

// The classifier sorts each planted shape: shape.go's three shapes, a
// read in each of two functions, five reads it cannot classify, and a key
// built by concatenation, in parentheses or from a constant of its file or
// of another package, each a read, even in a function declared twice;
// constructionsDir's declaration and its one use, as acceptedDigests's
// argument in mustAcceptedDigests, alone are exempt, never as a key;
// ConstructionsParameter's declaration and its uses by name are exempt in
// its three files alone, and a comparison, a case and an accessor's
// argument through it in a third file are each unclassified; and a string
// that only holds the word is no use of the member.
// Mutants killed: classifyMemberUses counting a Name argument as a part
// whatever field it is given to (the planted at passes); the walk counting
// only index reads (the constant passes); isMemberName reading string
// literals alone (each built key passes); the importer giving every
// package an empty one, the module's included (specdata.Member passes);
// the exemption matched by the constant's name alone (the registry's
// constructionsDir passes), by the file alone (member passes), covering
// a key (the read through constructionsDir passes) or covering every use in
// its file (Member's, load's and other's pass, and mustAcceptedDigests's
// open); a use counted again inside a concatenation (member's literals
// count); ConstructionsParameter's exemption matched by the constant's name
// alone (the registry's ConstructionsParameter passes), by its file alone
// (its file's member passes), covering a read, a key or a part (those
// through it pass) or blind to a use in parentheses (that use counts), or
// covering every file (the third file's comparison, case and accessor's
// argument are exempt).
func TestConstructions_TheClassifierSortsEachPlantedShape(t *testing.T) {
	for _, c := range constructionsPlanted {
		m := typedSources(t, c.sources)
		for _, f := range m.files {
			got, want := classifyMemberUses(m, f), c.want[f.rel]
			if !slices.Equal(got.reads, want.reads) || got.keys != want.keys || got.parts != want.parts || got.exempt != want.exempt || got.other != want.other {
				t.Errorf("%s, %s: %+v, want %+v", c.name, f.rel, got, want)
			}
		}
	}
}

// After SHAPE, POLICY gives the same rows whatever constructions holds: the
// accepted text, a reference to it, a text no one accepted, a value of
// another type, or nothing. Mutant killed: Policy giving no rows to a proof
// without constructions.
func TestConstructions_PolicyIsTheSameWhateverTheMemberHolds(t *testing.T) {
	item := positiveProofs(t)["item"]
	shaped := shapeRun(t, item)
	d := specdata.Accepted()[0]
	var first string
	for i, value := range []any{item["constructions"], constructionsReferencePrefix + d + constructionsReferenceSuffix,
		map[string]any{"x": "y"}, strictjson.Number(0), nil} {
		run := *shaped
		run.Rows = slices.Clone(shaped.Rows)
		run.Proof = put(item, Pointer{Name("constructions")}, value)
		if value == nil {
			run.Proof = remove(item, Pointer{Name("constructions")})
		}
		Policy(&run)
		got := rowSummary(run.Rows[8:])
		if i == 0 {
			first = got
		}
		if got != first || !strings.Contains(got, "POLICY") {
			t.Errorf("constructions %#v: %s, want %s", value, got, first)
		}
	}
}
