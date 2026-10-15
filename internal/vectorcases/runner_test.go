// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The per-step runner over the conformance cases of SPEC.md section 13.2,
// as the reference verifier's elixir/test/verifier/gate_cases_test.exs runs
// them, here through steps CONFIGURATION to SIGNATURE (SPEC.md section
// 10.2), the gate, with internal/verifier.

package vectorcases_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// The per-step runner. It runs each case of cases.json through the steps
// built so far and compares what SPEC.md section 13.1 compares, cut at the
// last of them: a case decided at a later step is held only to passing
// these. lastBuilt names that last step, and run runs the steps from
// firstBuilt to it. A run whose gate fails has ended, and every row it
// will report is made, so all of its rows are compared, the skip
// prior_failure rows of the later steps included.
//
// An online case's step 0 also confirms each pinned source's identity,
// which is not built, so an online case that CONFIGURATION decides is not
// run, and every other online case runs from READ, its rows compared from
// there, after a CONFIGURATION made offline under its environment and
// keyring. An online case whose keyring is unavailable, a failed online
// read of the keyring, is compared through ENVELOPE: its KEY_BINDING row
// needs its replay file's keyring read, which the replay adapter answers.

// lastBuilt is the last step the runner runs.
const lastBuilt = "SIGNATURE"

// firstBuilt is the first.
const firstBuilt = "CONFIGURATION"

// outcome is what the built steps give for a case: a bare rejection, or
// the rows of the steps that passed, whether the gate ended the run, its
// attribution, and the first row whose values JCS cannot write.
type outcome struct {
	code        string
	subReason   string
	rows        []vectorcases.Row
	ended       bool
	attribution string
	badValues   error
}

func (o outcome) bare() bool { return o.code != "" }

func passRow(step string) vectorcases.Row {
	return vectorcases.Row{Status: "pass", Step: step}
}

// keyringUnavailable reports whether the case's keyring is unavailable.
func keyringUnavailable(c vectorcases.Case) bool {
	return c.Config.Keyring != nil && *c.Config.Keyring == "unavailable"
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

// reportOf is the outcome of a run that passed steps 0 to 3, its rows from
// the one at index from.
func reportOf(run *verifier.Run, from int) outcome {
	o := outcome{rows: rowsOf(run)[from:], ended: run.Ended, attribution: run.Attribution}
	for _, row := range run.Rows {
		if _, err := jcs.Encode(row.Values); err != nil && o.badValues == nil {
			o.badValues = err
		}
	}
	return o
}

// run runs the built steps over the case's proof file: an offline case
// from CONFIGURATION, and an online case from READ.
func run(t *testing.T, c vectorcases.Case) outcome {
	t.Helper()
	cfg := caseConfig(t, c)
	if c.Config.Online == nil {
		r, rejection, err := verifier.GateFile(cfg, c.ProofPath())
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		if rejection != nil {
			return bareOf(rejection)
		}
		return reportOf(r, 0)
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
	if !keyringUnavailable(c) {
		verifier.Environment(r)
		verifier.KeyBinding(r)
		verifier.Signature(r)
	}
	return reportOf(r, 1)
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

// rowsBetween returns the rows of the steps from first to last.
func rowsBetween(first, last string, rows []vectorcases.Row) []vectorcases.Row {
	from, to := vectorcases.StepIndex(first), vectorcases.StepIndex(last)
	var out []vectorcases.Row
	for _, row := range rows {
		if i := vectorcases.StepIndex(row.Step); from <= i && i <= to {
			out = append(out, row)
		}
	}
	return out
}

// rowsFrom returns the rows of the steps from first to lastBuilt.
func rowsFrom(first string, rows []vectorcases.Row) []vectorcases.Row {
	return rowsBetween(first, lastBuilt, rows)
}

// rowsThrough returns the rows of the steps from firstBuilt to lastBuilt.
func rowsThrough(rows []vectorcases.Row) []vectorcases.Row {
	return rowsFrom(firstBuilt, rows)
}

func rowsEqual(a, b []vectorcases.Row) bool {
	return slices.EqualFunc(a, b, func(x, y vectorcases.Row) bool { return x.String() == y.String() })
}

// attributionFits reports whether a run's attribution after a gate that
// passed fits the label its case expects (SPEC.md section 10.5): a label
// that names the attribution needs it, and failed fits any.
func attributionFits(attribution, label string) bool {
	switch {
	case attribution == verifier.AttributionUnattributed:
		return label == "unattributed" || label == "failed"
	case strings.HasPrefix(attribution, "keyring:") && len(attribution) == len("keyring:")+64:
		return label == "supplied-keyring" || label == "failed"
	case attribution == verifier.AttributionTruestamp:
		return label == "verified" || label == "internally-consistent" || label == "failed"
	}
	return false
}

// Every case of cases.json through the built steps: a case they decide
// gives exactly its code and sub_reason; every other case passes steps 0
// to 3, and a report's rows of the built steps are exactly its expected
// rows, all of them when the gate ends the run, whose label is then failed
// and whose attribution is none; a run the gate opens has the attribution
// its label needs; every row's values write with JCS; an online case
// CONFIGURATION decides is not run. Mutants killed: parseNumber without
// its call to profileNumber (the parse-integer-* and parse-number-* cases
// are accepted); skipWhitespace without the carriage return (the
// sameness-format-* cases, which end their lines in CRLF, are refused);
// Envelope without the signature's member count (the
// envelope-signature-extra-key_id case passes); the keyring's VECTOR_KEY
// check left out (configuration-vector_key-under-staging passes); a
// non-production run with no keyring source governed by the compiled pin
// (configuration-no_keyring_source passes); ENVIRONMENT passing any marker
// under a non-production E_v (environment-absent-under-staging passes);
// keyless mode skipping SIGNATURE (signature-other_key-keyless passes);
// the proof_hash prefix 0x50 replaced by 0x31 (every sound case fails
// SIGNATURE); EndRun leaving out the skip rows (every gate failure's rows
// differ).
func TestCases_ThroughSignature(t *testing.T) {
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
		if online && step == "CONFIGURATION" {
			tally["not run, online and decided at CONFIGURATION"]++
			continue
		}

		t.Run(c.ID, func(t *testing.T) {
			got := run(t, c)
			switch {
			case step != "" && vectorcases.StepIndex(step) <= vectorcases.StepIndex(lastBuilt):
				tally["rejected at "+step]++
				if got.code != c.Expected.Code || got.subReason != c.Expected.SubReason {
					t.Fatalf("got %s (%s), want %s (%s)", got.code, got.subReason, c.Expected.Code, c.Expected.SubReason)
				}
				return
			case got.bare():
				t.Fatalf("rejected %s (%s) at a built step; the case is decided later", got.code, got.subReason)
			case step != "":
				t.Fatalf("no case is decided by a bare rejection after %s", lastBuilt)
			}
			if got.badValues != nil {
				t.Fatalf("a row's values: %v", got.badValues)
			}
			want := c.Expected.Rows
			if online {
				want = rowsBetween("READ", "VERDICT", want)
			}
			switch {
			case online && keyringUnavailable(c):
				tally["online report, from READ through ENVELOPE"]++
				want = rowsBetween("READ", "ENVELOPE", want)
			case got.ended:
				tally[map[bool]string{false: "report, the gate fails", true: "online report, from READ, the gate fails"}[online]]++
				if c.Expected.Label != "failed" || got.attribution != verifier.AttributionNone {
					t.Fatalf("the gate ended the run with attribution %q; the case's label is %s", got.attribution, c.Expected.Label)
				}
			default:
				tally[map[bool]string{false: "report, the gate passes", true: "online report, from READ, the gate passes"}[online]]++
				want = rowsThrough(want)
				if !attributionFits(got.attribution, c.Expected.Label) {
					t.Fatalf("attribution %q, label %s", got.attribution, c.Expected.Label)
				}
			}
			if !rowsEqual(got.rows, want) {
				t.Fatalf("rows = %v, want %v", got.rows, want)
			}
		})
	}

	want := map[string]int{
		"rejected at CONFIGURATION":                    11,
		"not run, online and decided at CONFIGURATION": 5,
		"rejected at READ":                             1,
		"rejected at PARSE":                            15,
		"rejected at ENVELOPE":                         32,
		"report, the gate fails":                       30,
		"report, the gate passes":                      285,
		"online report, from READ, the gate passes":    62,
		"online report, from READ through ENVELOPE":    2,
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

// The families cut at SIGNATURE are SPEC.md section 13.3's: the 11
// environment-* cases, each environment_mismatch; the 12 signature-*
// cases, each signature_invalid; the 2 mixed_order-* cases, one passing
// and one signature_invalid, both keyless; the 5 offline key_binding-*
// cases, three key_binding_failed, the inactive key passing and the
// keyless skip; and the 9 gates-* cases, three of them failing at the gate
// and six passing it, three of those online. Each offline one runs to its
// expected rows of the gate. Mutant killed: keyless mode skipping
// SIGNATURE (the four keyless signature-* cases and
// mixed_order-k_not_multiple_of_8 pass).
func TestCases_FamiliesCutAtSignature(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	gateCode := func(c vectorcases.Case) string {
		var codes []string
		for _, row := range rowsBetween("ENVIRONMENT", "SIGNATURE", c.Expected.Rows) {
			if row.Code != nil && *row.Code != "prior_failure" {
				codes = append(codes, *row.Code)
			}
		}
		return strings.Join(codes, " ")
	}
	got := map[string]map[string]int{}
	for _, c := range cases {
		family := c.Family()
		if family == "key_binding" && c.Config.Online != nil {
			family = "key_binding online"
		}
		switch family {
		case "environment", "signature", "mixed_order", "key_binding", "key_binding online", "gates":
		default:
			continue
		}
		if got[family] == nil {
			got[family] = map[string]int{}
		}
		got[family][gateCode(c)]++
		if c.Config.Online == nil {
			o := run(t, c)
			if want := rowsBetween("ENVIRONMENT", "SIGNATURE", c.Expected.Rows); !rowsEqual(rowsBetween("ENVIRONMENT", "SIGNATURE", o.rows), want) {
				t.Errorf("%s: gate rows %v, want %v", c.ID, o.rows, want)
			}
		}
	}
	want := map[string]map[string]int{
		"environment": {"environment_mismatch": 11},
		"signature":   {"signature_invalid": 8, "key_binding_not_performed signature_invalid": 4},
		"mixed_order": {"key_binding_not_performed": 1, "key_binding_not_performed signature_invalid": 1},
		"key_binding": {"key_binding_failed": 3, "": 1, "key_binding_not_performed": 1},
		"gates":       {"environment_mismatch": 1, "key_binding_failed": 1, "signature_invalid": 1, "": 6},
	}
	for family, codes := range want {
		for code, n := range codes {
			if got[family][code] != n || len(got[family]) != len(codes) {
				t.Fatalf("%s: %v, want %v", family, got[family], codes)
			}
		}
	}
	if got["key_binding online"]["keyring_unavailable"] != 2 {
		t.Fatalf("online key_binding: %v", got["key_binding online"])
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
			if got := rowsBetween("CONFIGURATION", "ENVELOPE", c.Expected.Rows); !rowsEqual(got, want) {
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
			got := rowsBetween("READ", "PARSE", c.Expected.Rows)
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
