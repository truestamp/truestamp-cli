// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The per-step runner over the conformance cases of SPEC.md section 13.2,
// as the reference verifier's elixir/test/verifier/gate_cases_test.exs runs
// them, here through steps CONFIGURATION, READ, PARSE and ENVELOPE (SPEC.md
// section 10.2), with internal/verifier.

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
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// The per-step runner. It runs each case of cases.json through the steps
// built so far and compares what SPEC.md section 13.1 compares, cut at the
// last of them: a case decided at a later step is held only to passing
// these. lastBuilt names that last step, and run runs the steps from
// firstBuilt to it.
//
// An online case's step 0 also confirms each pinned source's identity,
// which is not built, so an online case that CONFIGURATION decides is not
// run, and every other online case runs from READ, its rows compared from
// there, after a CONFIGURATION made offline under its environment and
// keyring.

// lastBuilt is the last step the runner runs.
const lastBuilt = "ENVELOPE"

// firstBuilt is the first.
const firstBuilt = "CONFIGURATION"

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

// caseConfig is the configuration a case gives (SPEC.md section 13.2): its
// E_v and pins, and its keyring: test, the vectors' keyring.test.json; none,
// keyless mode; a path, that keyring file; and no keyring option. An online
// case's configuration is made offline, and its keyring unavailable, the
// failed online read, is no keyring option.
func caseConfig(t *testing.T, c vectorcases.Case) verifier.Config {
	t.Helper()
	cfg, err := verifier.NewConfig(c.Config.Environment)
	if err != nil {
		t.Fatalf("%s: %v", c.ID, err)
	}
	pins := c.Config.Networks
	cfg.Networks = specdata.NetworkPins{
		Commitment: specdata.ChainNetworks{Bitcoin: pins.Commitment["bitcoin"], Stellar: pins.Commitment["stellar"]},
		Entropy:    specdata.ChainNetworks{Bitcoin: pins.Entropy["bitcoin"], Stellar: pins.Entropy["stellar"]},
	}
	switch keyring := c.Config.Keyring; {
	case keyring == nil || *keyring == "unavailable":
	case *keyring == "test":
		cfg.Keyring = verifier.KeyringSource{Kind: verifier.SourceFile, Path: vectorcases.Path("keyring.test.json")}
	case *keyring == "none":
		cfg.Keyring = verifier.KeyringSource{Kind: verifier.SourceKeyless}
	default:
		cfg.Keyring = verifier.KeyringSource{Kind: verifier.SourceFile, Path: vectorcases.Path(*keyring)}
	}
	return cfg
}

func rowsOf(run *verifier.Run) []vectorcases.Row {
	var out []vectorcases.Row
	for _, row := range run.Rows {
		code, part, status, step := row.Reduce()
		out = append(out, vectorcases.Row{Code: code, Part: part, Status: status, Step: step})
	}
	return out
}

func bareOf(rejection *verifier.Rejection) outcome {
	return outcome{code: string(rejection.Code), subReason: string(rejection.SubReason)}
}

// run runs the built steps over the case's proof file: an offline case
// from CONFIGURATION, and an online case from READ.
func run(t *testing.T, c vectorcases.Case) outcome {
	t.Helper()
	cfg := caseConfig(t, c)
	if c.Config.Online == nil {
		r, rejection, err := verifier.AdmitFile(cfg, c.ProofPath())
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		if rejection != nil {
			return bareOf(rejection)
		}
		return outcome{rows: rowsOf(r)}
	}
	r, rejection, err := verifier.Configure(cfg)
	if err != nil || rejection != nil {
		t.Fatalf("%s: the offline CONFIGURATION of an online case: %v, %v", c.ID, rejection, err)
	}
	data, err := strictjson.ReadFile(c.ProofPath())
	var tooLarge *strictjson.Rejection
	switch {
	case errors.As(err, &tooLarge):
		return outcome{code: string(tooLarge.Code)}
	case err != nil:
		t.Fatalf("%s: the proof file cannot be read: %v", c.ID, err)
	}
	for _, step := range []func() *verifier.Rejection{
		func() *verifier.Rejection { return verifier.Read(r, data) },
		func() *verifier.Rejection { return verifier.Parse(r, data) },
		func() *verifier.Rejection { return verifier.Envelope(r) },
	} {
		if rejection := step(); rejection != nil {
			return bareOf(rejection)
		}
	}
	return outcome{rows: rowsOf(r)[1:]}
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

// rowsFrom returns the rows of the steps from first to lastBuilt.
func rowsFrom(first string, rows []vectorcases.Row) []vectorcases.Row {
	from, last := vectorcases.StepIndex(first), vectorcases.StepIndex(lastBuilt)
	var out []vectorcases.Row
	for _, row := range rows {
		if i := vectorcases.StepIndex(row.Step); from <= i && i <= last {
			out = append(out, row)
		}
	}
	return out
}

// rowsThrough returns the rows of the steps from firstBuilt to lastBuilt.
func rowsThrough(rows []vectorcases.Row) []vectorcases.Row {
	return rowsFrom(firstBuilt, rows)
}

func rowsEqual(a, b []vectorcases.Row) bool {
	return slices.EqualFunc(a, b, func(x, y vectorcases.Row) bool { return x.String() == y.String() })
}

// Every case of cases.json through the built steps: a case they decide
// gives exactly its code and sub_reason; every other case passes them, and
// a report's rows of those steps are exactly its expected rows; an online
// case CONFIGURATION decides is not run. Mutants killed: parseNumber
// without its call to profileNumber (the parse-integer-* and parse-number-*
// cases are accepted); skipWhitespace without the carriage return (the
// sameness-format-* cases, which end their lines in CRLF, are refused);
// Envelope without the signature's member count (the
// envelope-signature-extra-key_id case passes); the keyring's VECTOR_KEY
// check left out (configuration-vector_key-under-staging passes); a
// non-production run with no keyring source governed by the compiled pin
// (configuration-no_keyring_source passes).
func TestCases_ThroughEnvelope(t *testing.T) {
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
		online := c.Config.Online != nil
		switch {
		case online && step == "CONFIGURATION":
			tally["not run, online and decided at CONFIGURATION"]++
			continue
		case step != "" && vectorcases.StepIndex(step) <= vectorcases.StepIndex(lastBuilt):
			tally["rejected at "+step]++
		case step != "":
			tally["passes, then rejected at "+step]++
		case online:
			tally["online report, from READ"]++
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
			case step == "" && online:
				if want := rowsFrom("READ", c.Expected.Rows); !rowsEqual(got.rows, want) {
					t.Fatalf("rows = %v, want %v", got.rows, want)
				}
			case step == "":
				if want := rowsThrough(c.Expected.Rows); !rowsEqual(got.rows, want) {
					t.Fatalf("rows = %v, want %v", got.rows, want)
				}
			}
		})
	}

	want := map[string]int{
		"rejected at CONFIGURATION":                    11,
		"not run, online and decided at CONFIGURATION": 5,
		"rejected at READ":                             1,
		"rejected at PARSE":                            15,
		"rejected at ENVELOPE":                         32,
		"report":                                       315,
		"online report, from READ":                     64,
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

// The families cut at ENVELOPE are SPEC.md section 13.3's: the 32
// envelope-* cases, each invalid_envelope with the sub_reason of its
// check; the 11 offline configuration-* cases, each configuration_error;
// and the 9 gates-* cases, each a report, so none ends by step 3. Every
// report's rows of steps 0 to 3 are four part-null passes. Mutant killed:
// Envelope naming every failure of K's bytes value (the small-order cases
// give value).
func TestCases_FamiliesCutAtEnvelope(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	subReasons := map[string]int{}
	families := map[string]int{}
	for _, c := range cases {
		online := c.Config.Online != nil
		switch family := c.Family(); {
		case family == "envelope":
			if c.Expected.Code != "invalid_envelope" || online {
				t.Fatalf("%s: %+v", c.ID, c.Expected)
			}
			subReasons[c.Expected.SubReason]++
			families[family]++
		case family == "configuration" && !online:
			if c.Expected.Code != "configuration_error" {
				t.Fatalf("%s: %+v", c.ID, c.Expected)
			}
			families[family]++
		case family == "gates":
			if c.Expected.Bare() {
				t.Fatalf("%s ends in a bare rejection", c.ID)
			}
			families[family]++
		}
		if !c.Expected.Bare() {
			want := []vectorcases.Row{passRow("CONFIGURATION"), passRow("READ"), passRow("PARSE"), passRow("ENVELOPE")}
			if got := rowsThrough(c.Expected.Rows); !rowsEqual(got, want) {
				t.Fatalf("%s: rows of steps 0 to 3 = %v", c.ID, got)
			}
		}
	}
	if want := map[string]int{"envelope": 32, "configuration": 11, "gates": 9}; len(families) != 3 ||
		families["envelope"] != want["envelope"] || families["configuration"] != want["configuration"] || families["gates"] != want["gates"] {
		t.Fatalf("families = %v, want %v", families, want)
	}
	want := map[string]int{"document": 5, "proof": 2, "signature": 3, "alg": 3, "public_key": 15, "value": 4}
	for _, sub := range verifier.EnvelopeChecks() {
		if subReasons[string(sub)] != want[string(sub)] {
			t.Fatalf("envelope sub_reasons = %v, want %v", subReasons, want)
		}
	}
	for _, c := range cases {
		if c.Family() != "envelope" || (!strings.Contains(c.ID, "small_order") && !strings.Contains(c.ID, "y_not_below_p")) {
			continue
		}
		if got := run(t, c); got.subReason != "public_key" {
			t.Fatalf("%s: %+v", c.ID, got)
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
			got := slices.DeleteFunc(rowsFrom("READ", c.Expected.Rows), func(row vectorcases.Row) bool { return row.Step == "ENVELOPE" })
			if !rowsEqual(got, want) {
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
