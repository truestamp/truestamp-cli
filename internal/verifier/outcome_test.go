// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// countedRead gives the bytes of the file at path, and counts each call.
func countedRead(path string, calls *int) func() ([]byte, error) {
	return func() ([]byte, error) {
		*calls++
		return strictjson.ReadFile(path)
	}
}

// Each way a run ends, and its exit code (SPEC.md section 12.2): a label's
// code, with the report and, past DERIVE, the trace; a bare rejection's,
// 4, and configuration_error's, 8, with the rejection and no report; an
// invalid configuration and an input that cannot be read, 1. The file is
// read after CONFIGURATION, once, and not at all when CONFIGURATION
// fails. Mutants killed: Verify reading the document before Configure
// (the configuration_error run reads it); labelEndings giving
// supplied-keyring the ending of unattributed (exit 5 for a supplied
// keyring).
func TestVerify_Endings(t *testing.T) {
	sound := vectorcases.Path("positive/item.proof.json")
	keyless := testConfig(t)
	keyless.Keyring = KeyringSource{Kind: SourceKeyless}
	staging, err := NewConfig("staging")
	if err != nil {
		t.Fatal(err)
	}
	invalid := testConfig(t)
	invalid.Environment = "production"
	for _, c := range []struct {
		name   string
		cfg    Config
		read   func(*int) func() ([]byte, error)
		ending specdata.Ending
		code   int
		reads  int
	}{
		{"supplied-keyring", testConfig(t), func(n *int) func() ([]byte, error) { return countedRead(sound, n) }, specdata.EndingSuppliedKeyring, 6, 1},
		{"unattributed", keyless, func(n *int) func() ([]byte, error) { return countedRead(sound, n) }, specdata.EndingUnattributed, 5, 1},
		{"failed", testConfig(t), func(n *int) func() ([]byte, error) {
			return countedRead(vectorcases.Path("negative/key_binding-failed-other_key.proof.json"), n)
		}, specdata.EndingFailed, 3, 1},
		{"a bare rejection", testConfig(t), func(n *int) func() ([]byte, error) {
			return func() ([]byte, error) {
				*n++
				return strictjson.Read(bytes.NewReader(make([]byte, strictjson.MaxReadOctets+1)))
			}
		}, specdata.EndingBareRejection, 4, 1},
		{"configuration_error", staging, func(n *int) func() ([]byte, error) { return countedRead(sound, n) }, specdata.EndingConfigurationError, 8, 0},
		{"an invalid configuration", invalid, func(n *int) func() ([]byte, error) { return countedRead(sound, n) }, specdata.EndingRuntimeError, 1, 0},
		{"an input that cannot be read", testConfig(t), func(n *int) func() ([]byte, error) {
			return countedRead(filepath.Join(t.TempDir(), "missing.json"), n)
		}, specdata.EndingRuntimeError, 1, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			reads := 0
			outcome := Verify(c.cfg, c.read(&reads))
			if outcome.Ending != c.ending || outcome.ExitCode() != c.code || reads != c.reads {
				t.Fatalf("ending %q, exit %d, %d reads; want %q, %d, %d (%v)", outcome.Ending, outcome.ExitCode(), reads, c.ending, c.code, c.reads, outcome.Err)
			}
			switch c.code {
			case 3, 5, 6:
				if outcome.Report == nil || outcome.Rejection != nil || outcome.Err != nil || outcome.Report.Label != strings.TrimPrefix(string(c.ending), "label ") {
					t.Errorf("%+v", outcome)
				}
			case 4, 8:
				if outcome.Report != nil || outcome.Rejection == nil || outcome.Trace != nil {
					t.Errorf("%+v", outcome)
				}
			default:
				if outcome.Report != nil || outcome.Rejection != nil || outcome.Err == nil {
					t.Errorf("%+v", outcome)
				}
			}
		})
	}
	outcome := Verify(testConfig(t), func() ([]byte, error) { return strictjson.ReadFile(sound) })
	if len(outcome.Trace) == 0 {
		t.Error("a run past DERIVE gives no trace")
	}
}

// A fault raised before a step runs: at CONFIGURATION, READ, PARSE or
// ENVELOPE, before steps 0 to 3 have passed, and at VERDICT, the run
// gives no report and ends on the fault, exit 2, its error wrapping
// ErrFault; from ENVIRONMENT to EXPECTED_HASH, it is one fail
// internal_error row at that step, part null, after the rows before it,
// then one skip prior_failure, part null, for each later step up to
// EXPECTED_HASH, and the run is failed, exit 3, its code internal_error,
// with the attribution none for a fault at the gate (SPEC.md sections 10.5
// and 12.2). Mutants killed: runStep without its recover (the test
// panics); runStep leaving the attribution after a gate fault; runStep
// not ending the run (the later steps give their own rows); admit's
// recover dropped (the test panics).
func TestVerify_Faults(t *testing.T) {
	sound := vectorcases.Path("positive/item.proof.json")
	read := func() ([]byte, error) { return strictjson.ReadFile(sound) }
	for _, step := range []string{"CONFIGURATION", "READ", "PARSE", "ENVELOPE", "VERDICT"} {
		t.Run(step, func(t *testing.T) {
			useFault(t, step)
			outcome := Verify(testConfig(t), read)
			if outcome.Ending != specdata.EndingFault || outcome.ExitCode() != 2 || !errors.Is(outcome.Err, ErrFault) ||
				outcome.Report != nil || outcome.Rejection != nil || !strings.Contains(outcome.Err.Error(), step) {
				t.Errorf("%+v", outcome)
			}
		})
	}
	for _, c := range []struct {
		step, attribution string
	}{
		{"ENVIRONMENT", AttributionNone},
		{"SIGNATURE", AttributionNone},
		{"DERIVE", "keyring:"},
		{"WINDOW", "keyring:"},
		{"EXPECTED_HASH", "keyring:"},
	} {
		t.Run(c.step, func(t *testing.T) {
			useFault(t, c.step)
			outcome := Verify(testConfig(t), read)
			report := outcome.Report
			if outcome.Ending != specdata.EndingFailed || outcome.ExitCode() != 3 || report == nil || report.Code != specdata.ReasonInternalError ||
				!strings.HasPrefix(report.Attribution, c.attribution) || report.SubmittedAfter != nil || report.SubmittedBefore != nil {
				t.Fatalf("%+v", outcome)
			}
			at := slices.IndexFunc(report.Rows, func(r Row) bool { return r.Status == StatusFail })
			if row := report.Rows[at]; row.Step != c.step || row.Code != specdata.ReasonInternalError || row.Part != nil || at == 0 || report.Rows[at-1].Step == c.step {
				t.Fatalf("the fault's row %d: %+v", at, row)
			}
			later := rowSteps[slices.Index(rowSteps, c.step)+1:]
			if len(report.Rows) != at+1+len(later) {
				t.Fatalf("%d rows after the fault, want %d", len(report.Rows)-at-1, len(later))
			}
			for i, step := range later {
				if row := report.Rows[at+1+i]; row.Step != step || row.Code != specdata.ReasonPriorFailure || row.Part != nil {
					t.Errorf("row %d after the fault: %+v", i+1, row)
				}
			}
		})
	}
}

// A fault a step raises after it has given rows keeps them: the
// internal_error row follows them, at that step, and the run ends there.
// Mutant killed: runStep dropping the rows the step gave before the fault.
func TestVerify_AFaultInsideAStepKeepsItsRows(t *testing.T) {
	useStep(t, "POLICY", func(run *Run) {
		run.AddRow(PassRow("POLICY", Part("/commitments/bitcoin/network"), "a row before the fault", nil))
		panic("a fault inside POLICY")
	})
	outcome := Verify(testConfig(t), func() ([]byte, error) { return strictjson.ReadFile(vectorcases.Path("positive/item.proof.json")) })
	var policy []string
	for _, row := range outcome.Report.Rows {
		if row.Step == "POLICY" {
			policy = append(policy, string(row.Status)+" "+string(row.Code)+" "+row.Message)
		}
	}
	if want := []string{"pass  a row before the fault", "fail internal_error The verifier stopped on a fault in this step."}; !slices.Equal(policy, want) {
		t.Errorf("POLICY rows %q, want %q", policy, want)
	}
}

// --- the compiled-in addresses ---

// In every build but the cmd package's test binary, which sets them at
// link time, Configure reads Truestamp's keyring from
// https://www.truestamp.com/.well-known/keyring.json and SOURCES reads
// NIST from https://beacon.nist.gov/beacon/2.0: this package's test
// binary is built as a shipped one is, with no linker flag for either.
// Mutants killed: either variable initialized to another address.
func TestEndpoints_ShippedValues(t *testing.T) {
	if currentKeyringRead != "https://www.truestamp.com/.well-known/keyring.json" {
		t.Errorf("the keyring is read from %q", currentKeyringRead)
	}
	if nistBeaconBase != "https://beacon.nist.gov/beacon/2.0" || nistSet().Sources[0].Source.URL != nistBeaconBase {
		t.Errorf("NIST is read from %q", nistBeaconBase)
	}
}

// Each link-time address has one reader in the package's code:
// currentKeyringRead is read only in Configure and nistBeaconBase only in
// nistSet, and neither is assigned anywhere, so no flag, configuration
// key, environment variable or document can reach either. A reference in
// a package-level declaration, a var's initializer among them, is a
// reader too, named for the declaration. Mutants killed: governs,
// CONFIGURATION's message, reading currentKeyringRead; a package-level
// `var shownKeyringURL = currentKeyringRead`, read at init.
func TestEndpoints_EachHasOneReader(t *testing.T) {
	readers := map[string]string{"currentKeyringRead": "Configure", "nistBeaconBase": "nistSet"}
	uses := map[string][]string{}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, name, data, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			var reader string
			var declared map[*ast.Ident]bool
			switch d := decl.(type) {
			case *ast.FuncDecl:
				reader = d.Name.Name
			case *ast.GenDecl:
				reader = "package-level " + d.Tok.String()
				declared = map[*ast.Ident]bool{}
				for _, spec := range d.Specs {
					if v, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range v.Names {
							declared[name] = true
							reader += " " + name.Name
						}
					}
				}
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.Ident:
					if _, watched := readers[n.Name]; watched && !declared[n] {
						uses[n.Name] = append(uses[n.Name], reader)
					}
				case *ast.AssignStmt:
					for _, lhs := range n.Lhs {
						if id, ok := lhs.(*ast.Ident); ok {
							if _, watched := readers[id.Name]; watched {
								t.Errorf("%s: %s is assigned", fset.Position(id.Pos()), id.Name)
							}
						}
					}
				}
				return true
			})
		}
	}
	for name, reader := range readers {
		if !slices.Equal(uses[name], []string{reader}) {
			t.Errorf("%s is read in %q, want only in %s", name, uses[name], reader)
		}
	}
}
