// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The per-step runner over the conformance cases of SPEC.md section 13.2,
// as the reference verifier's elixir/test/verifier/gate_cases_test.exs runs
// them, here through steps READ and PARSE (SPEC.md section 10.2).

package vectorcases_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// The per-step runner. It runs each case of cases.json through the steps
// built so far and compares what SPEC.md section 13.1 compares, cut at the
// last of them: a case decided at a later step is held only to passing
// these. lastBuilt names that last step, and run runs the steps from
// firstBuilt to it.

// lastBuilt is the last step the runner runs.
const lastBuilt = "PARSE"

// firstBuilt is the first: CONFIGURATION comes before it and is not built,
// so a case it decides is not run.
const firstBuilt = "READ"

// outcome is what the built steps give for a case: a bare rejection, or the
// rows of the steps that passed.
type outcome struct {
	code      string
	subReason string
	rows      []vectorcases.Row
}

func (o outcome) bare() bool { return o.code != "" }

func passRow(step string) vectorcases.Row {
	return vectorcases.Row{Status: "pass", Step: step}
}

// run runs steps READ and PARSE over the case's proof file.
func run(t *testing.T, c vectorcases.Case) outcome {
	t.Helper()
	var rejection *strictjson.Rejection
	data, err := strictjson.ReadFile(c.ProofPath())
	if errors.As(err, &rejection) {
		return outcome{code: string(rejection.Code), subReason: string(rejection.SubReason)}
	}
	if err != nil {
		t.Fatalf("%s: the proof file cannot be read: %v", c.ID, err)
	}
	rows := []vectorcases.Row{passRow("READ")}

	if _, err := strictjson.Parse(data); errors.As(err, &rejection) {
		return outcome{code: string(rejection.Code), subReason: string(rejection.SubReason)}
	} else if err != nil {
		t.Fatalf("%s: %v", c.ID, err)
	}
	return outcome{rows: append(rows, passRow("PARSE"))}
}

// decidedAt returns the step at which a case ends in a bare rejection, or
// the empty string for a report.
func decidedAt(t *testing.T, c vectorcases.Case) string {
	t.Helper()
	if !c.Expected.Bare() {
		return ""
	}
	entry, ok := specdata.LookupReason(specdata.Reason(c.Expected.Code))
	if !ok || entry.Status != specdata.StatusBare || vectorcases.StepIndex(entry.Step) < 0 {
		t.Fatalf("%s: %s is not a bare rejection of one step", c.ID, c.Expected.Code)
	}
	return entry.Step
}

// rowsThrough returns the rows of the steps from firstBuilt to lastBuilt.
func rowsThrough(rows []vectorcases.Row) []vectorcases.Row {
	first, last := vectorcases.StepIndex(firstBuilt), vectorcases.StepIndex(lastBuilt)
	var out []vectorcases.Row
	for _, row := range rows {
		if i := vectorcases.StepIndex(row.Step); first <= i && i <= last {
			out = append(out, row)
		}
	}
	return out
}

func rowsEqual(a, b []vectorcases.Row) bool {
	return slices.EqualFunc(a, b, func(x, y vectorcases.Row) bool { return x.String() == y.String() })
}

// Every case of cases.json through READ and PARSE: a case those steps
// decide gives exactly its code and sub_reason; every other case passes
// them, and a report's rows of those steps are exactly its expected rows;
// a case CONFIGURATION decides is not run. Mutants killed: parseNumber
// without its call to profileNumber (the parse-integer-* and parse-number-*
// cases are accepted), and skipWhitespace without the carriage return (the
// sameness-format-* cases, which end their lines in CRLF, are refused).
func TestCases_ThroughParse(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 443 {
		t.Fatalf("cases.json holds %d cases, want 443", len(cases))
	}
	tally := map[string]int{}
	for _, c := range cases {
		step := decidedAt(t, c)
		switch {
		case step != "" && vectorcases.StepIndex(step) < vectorcases.StepIndex(firstBuilt):
			tally["not run, decided at "+step]++
			continue
		case step != "" && vectorcases.StepIndex(step) <= vectorcases.StepIndex(lastBuilt):
			tally["rejected at "+step]++
		case step != "":
			tally["passes, then rejected at "+step]++
		default:
			tally["report"]++
		}

		t.Run(c.ID, func(t *testing.T) {
			got := run(t, c)
			switch {
			case step != "" && vectorcases.StepIndex(step) <= vectorcases.StepIndex(lastBuilt):
				if got.code != c.Expected.Code || got.subReason != c.Expected.SubReason {
					t.Fatalf("got %s (%s), want %s (%s)", got.code, got.subReason, c.Expected.Code, c.Expected.SubReason)
				}
			case got.bare():
				t.Fatalf("rejected %s (%s) at a built step; the case is decided later", got.code, got.subReason)
			case step == "":
				if want := rowsThrough(c.Expected.Rows); !rowsEqual(got.rows, want) {
					t.Fatalf("rows = %v, want %v", got.rows, want)
				}
			}
		})
	}

	want := map[string]int{
		"not run, decided at CONFIGURATION": 16,
		"rejected at READ":                  1,
		"rejected at PARSE":                 15,
		"passes, then rejected at ENVELOPE": 32,
		"report":                            379,
	}
	if len(tally) != len(want) {
		t.Fatalf("tally = %v, want %v", tally, want)
	}
	for k, n := range want {
		if tally[k] != n {
			t.Fatalf("tally = %v, want %v", tally, want)
		}
	}
}

// The families cut at PARSE are exactly SPEC.md section 13.3's: one READ
// case and 15 PARSE cases, each with its sub_reason, and every report's
// rows of READ and PARSE are two part-null passes. Mutant killed:
// strictjson.MaxReadOctets one less than MAX_READ_OCTETS (the READ case's
// file would be two octets over the limit).
func TestCases_FamiliesCutAtParse(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	subReasons := map[string]int{}
	for _, c := range cases {
		switch c.Family() {
		case "read":
			if c.ID != "read-input-too-large" || c.Expected.Code != "input_too_large" {
				t.Fatalf("%s: %+v", c.ID, c.Expected)
			}
			info, err := os.Stat(c.ProofPath())
			if err != nil || info.Size() != strictjson.MaxReadOctets+1 {
				t.Fatalf("%s is %v octets (%v), want MAX_READ_OCTETS + 1", c.ID, info.Size(), err)
			}
		case "parse":
			if c.Expected.Code != "malformed_json" {
				t.Fatalf("%s: %+v", c.ID, c.Expected)
			}
			subReasons[c.Expected.SubReason]++
		}
		if !c.Expected.Bare() {
			want := []vectorcases.Row{passRow("READ"), passRow("PARSE")}
			if got := rowsThrough(c.Expected.Rows); !rowsEqual(got, want) {
				t.Fatalf("%s: rows of READ and PARSE = %v", c.ID, got)
			}
		}
	}
	want := map[string]int{"duplicate_name": 3, "lone_surrogate": 1, "invalid_utf8": 1, "syntax": 2, "number": 7, "depth": 1}
	for k, n := range want {
		if subReasons[k] != n {
			t.Fatalf("parse sub_reasons = %v, want %v", subReasons, want)
		}
	}
	for _, sub := range specdata.SubReasons(specdata.ReasonMalformedJSON) {
		if subReasons[string(sub)] == 0 {
			t.Fatalf("no parse case for the sub_reason %s", sub)
		}
	}
}

// Every keyring a case supplies, and keyring.test.json, reads through J1 to
// J6 as a keyring load reads it (SPEC.md section 10.1, last paragraph), and
// the one keyring whose case is a repeated name is refused for it. Mutant
// killed: parseObject without its duplicate check (the duplicate_name
// keyring would load).
func TestCases_KeyringsReadUnderTheRules(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{"keyring.test.json": true}
	for _, c := range cases {
		if k := c.Config.Keyring; k != nil && strings.HasSuffix(*k, ".json") {
			paths[*k] = true
		}
	}
	if len(paths) != 11 {
		t.Fatalf("keyring files = %d, want 11", len(paths))
	}
	for path := range paths {
		data, err := strictjson.ReadFile(vectorcases.Path(path))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		_, err = strictjson.Parse(data)
		var rejection *strictjson.Rejection
		if path == "negative/configuration-keyring-duplicate_name.keyring.json" {
			if !errors.As(err, &rejection) || rejection.SubReason != specdata.SubReasonDuplicateName {
				t.Fatalf("%s: error = %v, want duplicate_name", path, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
}

// cases.json itself reads under J2 to J6, and to the same cases as
// encoding/json reads. Mutant killed: Parse making J1 itself, which binds
// a document, not a data file such as cases.json.
func TestCasesJSON_ReadsUnderTheStrictReader(t *testing.T) {
	data, err := os.ReadFile(vectorcases.Path("cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) <= strictjson.MaxReadOctets {
		t.Fatalf("cases.json is %d octets, not more than MAX_READ_OCTETS (%d), so this test cannot show that Parse does not make J1", len(data), strictjson.MaxReadOctets)
	}
	value, err := strictjson.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	parsed := value.([]any)
	if len(parsed) != len(cases) {
		t.Fatalf("the strict reader reads %d cases, encoding/json %d", len(parsed), len(cases))
	}
	for i, c := range cases {
		object := parsed[i].(map[string]any)
		if object["id"] != c.ID || object["proof"] != c.Proof {
			t.Fatalf("case %d: %v, %v", i, object["id"], c.ID)
		}
	}
}
