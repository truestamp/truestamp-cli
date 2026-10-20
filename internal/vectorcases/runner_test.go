// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The per-step runner over the conformance cases of SPEC.md section 13.2,
// through steps CONFIGURATION to DERIVE (SPEC.md section 10.2), the gate,
// SHAPE, POLICY and DERIVE, with internal/verifier, comparing what SPEC.md
// section 13.1 compares and more: for steps 0 to 6, each case's bare
// rejection or rows; for SHAPE and POLICY, their rows, with the check of
// where each value of their rows comes from (section 10.5); and for DERIVE,
// its rows, their values and the trace (section 10.6).

package vectorcases_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// The per-step runner. It runs each case of cases.json through the steps
// built so far and compares what SPEC.md section 13.1 compares, cut at the
// last of them: a case decided at a later step is held only to passing
// these. lastBuilt names that last step, and run runs the steps from
// firstBuilt to it. A run whose gate or SHAPE fails has ended, and every
// row it will report is made, so all of its rows are compared, the skip
// prior_failure rows of the later steps included. A failure at POLICY or
// DERIVE does not end the run, so a case either fails is held, as any
// other open run is, to its rows through DERIVE.
//
// DERIVE makes every check of step 9, parts (a) to (e), and its rows are
// compared whole, each with the values and trace rows of
// derive_model_test.go's model.
//
// An online case runs online, every read it makes answered from its replay
// file by the replay adapter (vectorcases.Replay, SPEC.md section 13.2), so
// an online case under prod with no keyring option, whose keyring is
// unavailable, reads the keyring of www.truestamp.com from its file at
// CONFIGURATION, and its failed read fails KEY_BINDING. An online case's
// step 0 also confirms each pinned source's identity: the file stands as
// the operator-pinned source of each network whose identity read it holds,
// each pinned to that network, and answers each identity read. A case runs
// under the pins its networks give, never the defaults of its E_v.

// lastBuilt is the last step the runner runs.
const lastBuilt = "DERIVE"

// firstBuilt is the first.
const firstBuilt = "CONFIGURATION"

// laterShapeChecks are the cases a check of SHAPE after check 3 decides,
// by that check: 4, the witnesses of an item; 5, the TIME and NISTTIME
// round trips; 6, the constructions digest. Each passes checks 1 to 3, so
// the schema finds nothing in it.
var laterShapeChecks = map[string]int{
	"shape-witnesses-missing":                 4,
	"shape-witnesses-extra":                   4,
	"shape-time-hour_24":                      5,
	"shape-time-february_30":                  5,
	"shape-time-second_60":                    5,
	"shape-time-february_29-1900":             5,
	"shape-time-february_29-2100":             5,
	"shape-first_of_two-times":                5,
	"shape-first_of_two-times-mirror":         5,
	"shape-constructions-byte_changed":        6,
	"shape-constructions-row_missing":         6,
	"shape-constructions-merkle_path_swapped": 6,
	"shape-reference-not_accepted":            6,
}

// outcome is what the built steps give for a case: a bare rejection, or
// the rows of the steps that passed, the values of each row of SHAPE and
// POLICY, whether the gate or SHAPE ended the run, whether a row failed,
// its attribution, the type and subject id it took at SHAPE, and the first
// row whose values JCS cannot write.
type outcome struct {
	run         *verifier.Run
	code        string
	subReason   string
	rows        []vectorcases.Row
	values      []stepValues
	ended       bool
	failed      bool
	attribution string
	typ         string
	subjectID   string
	badValues   error
}

// stepValues are one row of SHAPE or POLICY, reduced, with its values.
type stepValues struct {
	row    vectorcases.Row
	values map[string]any
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
// keyless mode; a path, that keyring file; and no keyring option, which
// unavailable is too. A case is offline unless it names a replay file; an
// online one reads its keyring, when it reads one, through the replay's
// keyring reader, and its sources' identities through its identity
// reader, its sources the networks whose identity read the file holds.
func caseConfig(t *testing.T, c vectorcases.Case, replay *caseReplay) verifier.Config {
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
	if replay != nil {
		cfg.Online, cfg.ReadKeyring, cfg.ReadIdentity = true, replay.keyrings, replay.identities
		cfg.Sources = replay.sources
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
	rows := rowsOf(run)
	o := outcome{run: run, rows: rows[from:], ended: run.Ended, failed: run.Failed(), attribution: run.Attribution, typ: run.Type, subjectID: run.SubjectID}
	for i, row := range run.Rows {
		if _, err := jcs.Encode(row.Values); err != nil && o.badValues == nil {
			o.badValues = err
		}
		if row.Step == "SHAPE" || row.Step == "POLICY" {
			o.values = append(o.values, stepValues{rows[i], row.Values})
		}
	}
	return o
}

// caseReplay is the replay adapter of one online case: its keyring
// reader, its identity reader, the sources its replay file stands for, and
// the reads each reader made.
type caseReplay struct {
	keyrings   verifier.KeyringReader
	identities verifier.IdentityReader
	sources    []verifier.Source

	mu sync.Mutex
	// urls are the keyring reads made, and identityReads the identity
	// reads, each "<chain> <network>".
	urls, identityReads []string
}

// replayOf is the replay adapter of an online case, or nil for an offline
// one. Each keyring read is answered from the case's replay file, a found
// keyring as a value with no bytes of its own (SPEC.md section 13.2), and
// an unreachable one as a failed read; each identity read of a network is
// answered from the file's identity read of that network, an unreachable
// one as a read with no usable answer. The file stands as the
// operator-pinned source of each network whose identity read it holds: a
// Horizon server for Stellar and an Esplora server for Bitcoin, pinned to
// that network, at a URL under replay.invalid that no read reaches. A read
// the file does not hold fails the test.
func replayOf(t *testing.T, c vectorcases.Case) *caseReplay {
	t.Helper()
	if c.Config.Online == nil {
		return nil
	}
	replay, err := vectorcases.LoadReplay(*c.Config.Online)
	if err != nil {
		t.Fatalf("%s: %v", c.ID, err)
	}
	adapter := &caseReplay{}
	adapter.keyrings = func(url string) (verifier.KeyringRead, error) {
		adapter.mu.Lock()
		adapter.urls = append(adapter.urls, url)
		adapter.mu.Unlock()
		value, err := replay.Keyring(url)
		if errors.Is(err, vectorcases.ErrNoAnswer) {
			t.Errorf("%s: %v", c.ID, err)
		}
		if err != nil {
			return verifier.KeyringRead{}, err
		}
		return verifier.KeyringRead{Replayed: true, Value: value}, nil
	}
	adapter.identities = func(s verifier.Source) (string, error) {
		adapter.mu.Lock()
		adapter.identityReads = append(adapter.identityReads, s.Chain+" "+s.Network)
		adapter.mu.Unlock()
		value, err := replay.Identity(s.Chain, s.Network)
		if errors.Is(err, vectorcases.ErrNoAnswer) {
			t.Errorf("%s: %v", c.ID, err)
		}
		return value, err
	}
	for _, s := range replay.Sources() {
		provider := source.Esplora
		if s.Chain == verifier.ChainStellar {
			provider = source.Horizon
		}
		adapter.sources = append(adapter.sources, verifier.Source{
			Chain: s.Chain, Provider: provider, Network: s.Network,
			URL: "https://replay.invalid/" + c.ID + "/" + s.Chain + "/" + s.Network,
		})
	}
	return adapter
}

// reads returns the identity reads the adapter made, sorted.
func (a *caseReplay) reads() []string {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Sorted(slices.Values(a.identityReads))
}

// run runs the built steps over the case's proof file, from CONFIGURATION
// to DERIVE, an online case's keyring and identity reads answered by the
// replay adapter. Each online case under prod with no keyring option reads the
// keyring of www.truestamp.com once, and no other case reads one. Each
// online case that passes CONFIGURATION has a source for every pin,
// confirmed unless its file's identity read of that network is
// unreachable, and reads each pinned network's identity once; an offline
// case reads none.
func run(t *testing.T, c vectorcases.Case) outcome {
	t.Helper()
	o, _ := runCase(t, c)
	return o
}

// runCase is run, also giving the replay adapter of an online case.
func runCase(t *testing.T, c vectorcases.Case) (outcome, *caseReplay) {
	t.Helper()
	replay := replayOf(t, c)
	cfg := caseConfig(t, c, replay)
	r, rejection, err := verifier.GateFile(cfg, c.ProofPath())
	if err != nil {
		t.Fatalf("%s: %v", c.ID, err)
	}
	wantReads := 0
	if c.Config.Online != nil && c.Config.Environment == "prod" && (c.Config.Keyring == nil || keyringUnavailable(c)) {
		wantReads = 1
	}
	var urls []string
	if replay != nil {
		urls = replay.urls
	}
	if len(urls) != wantReads || (wantReads == 1 && urls[0] != verifier.CurrentKeyringURL) {
		t.Errorf("%s: keyring reads %q, want %d of %s", c.ID, urls, wantReads, verifier.CurrentKeyringURL)
	}
	if rejection != nil {
		return bareOf(rejection), replay
	}
	checkSources(t, c, r, replay)
	verifier.Shape(r)
	verifier.Policy(r)
	verifier.Derive(r)
	return reportOf(r, 0), replay
}

// checkSources holds a run that passed CONFIGURATION to its case's
// sources: offline, none; online, a source for each pin, confirmed or, for
// a network whose replayed identity read is unreachable, unreachable, and
// one identity read of each pinned network.
func checkSources(t *testing.T, c vectorcases.Case, r *verifier.Run, replay *caseReplay) {
	t.Helper()
	if replay == nil {
		if len(r.Sources) != 0 {
			t.Errorf("%s: an offline run has sources %+v", c.ID, r.Sources)
		}
		return
	}
	var pinned []string
	for _, pin := range []struct {
		name     string
		networks map[string]string
	}{{verifier.PinCommitment, c.Config.Networks.Commitment}, {verifier.PinEntropy, c.Config.Networks.Entropy}} {
		for chain, network := range pin.networks {
			if !slices.Contains(pinned, chain+" "+network) {
				pinned = append(pinned, chain+" "+network)
			}
			record, ok := r.SourceFor(pin.name, chain)
			want := verifier.SourceConfirmed
			if strings.HasPrefix(c.ID, "reads-identity-"+chain+"-unreachable") {
				want = verifier.SourceUnreachable
			}
			if !ok || record.Status != want || record.Source.Network != network || record.From != verifier.FromOperator {
				t.Errorf("%s: the %s pin of %s: %+v, %v; want %s", c.ID, pin.name, chain, record, ok, want)
			}
		}
	}
	if got := replay.reads(); !slices.Equal(got, slices.Sorted(slices.Values(pinned))) {
		t.Errorf("%s: identity reads %q, want one of each of %q", c.ID, got, pinned)
	}
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

// expectedThrough returns a case's expected rows from firstBuilt to
// lastBuilt.
func expectedThrough(c vectorcases.Case) []vectorcases.Row {
	return rowsThrough(c.Expected.Rows)
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

// gateFails reports whether rows, a case's expected rows, fail at the gate.
func gateFails(rows []vectorcases.Row) bool {
	return slices.ContainsFunc(rowsBetween("ENVIRONMENT", "SIGNATURE", rows), func(row vectorcases.Row) bool {
		return row.Status == "fail"
	})
}

// shapeRow returns a case's expected SHAPE row.
func shapeRow(t *testing.T, c vectorcases.Case) vectorcases.Row {
	t.Helper()
	rows := rowsBetween("SHAPE", "SHAPE", c.Expected.Rows)
	if len(rows) != 1 {
		t.Fatalf("%s: SHAPE rows %v", c.ID, rows)
	}
	return rows[0]
}

// proofOf returns P of a case's proof, read by the strict reader.
func proofOf(t *testing.T, c vectorcases.Case) map[string]any {
	t.Helper()
	value, err := strictjson.ReadFile(c.ProofPath())
	if err == nil {
		var parsed any
		if parsed, err = strictjson.Parse(value); err == nil {
			return parsed.(map[string]any)["proof"].(map[string]any)
		}
	}
	t.Fatalf("%s: %v", c.ID, err)
	return nil
}

// typeAndSubject returns the type and subject id P carries (SPEC.md
// section 10.5), read here without the verifier.
func typeAndSubject(proof map[string]any) (string, string) {
	typ, _ := proof["type"].(string)
	holder := "subject"
	if typ == "block" || typ == "beacon" {
		holder = "block"
	}
	node, _ := proof[holder].(map[string]any)
	record, _ := node["record"].(map[string]any)
	id, _ := record["id"].(string)
	return typ, id
}

// constructionsMode is the mode of P's constructions (SPEC.md section 7.2),
// read here without the verifier: full for an object, reference for a
// string, none when absent.
func constructionsMode(proof map[string]any) string {
	switch proof["constructions"].(type) {
	case map[string]any:
		return "full"
	case string:
		return "reference"
	case nil:
		if _, present := proof["constructions"]; !present {
			return "none"
		}
	}
	return "other"
}

// constructionsDigestOf is d of P's full constructions text, SHA-256 of
// its JCS, computed here without the verifier or specdata, or "".
func constructionsDigestOf(t *testing.T, proof map[string]any) string {
	t.Helper()
	text, ok := proof["constructions"].(map[string]any)
	if !ok {
		return ""
	}
	encoded, err := jcs.Encode(text)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// valueSourced reports whether one value of a row of SHAPE or POLICY,
// named name, is one the run computed or holds (SPEC.md section 10.5), by
// its name: each is recomputed here from the case's configuration and
// proof, or is one of the verifier's own words, and a value the proof
// carries appears only under declared, in a failing row. A name not listed
// is not sourced, so a value added to a row must say where it comes from. A
// keyword is "false", the boolean schema's, or one of the keywords of
// SPEC.md section 9's subset, and a pin_kind is the kind of pin its row's
// position takes (pinKindAt).
func valueSourced(t *testing.T, c vectorcases.Case, proof map[string]any, row stepValues, name string, value any) bool {
	t.Helper()
	text, isText := value.(string)
	switch name {
	case "keyword":
		return isText && (text == "false" || slices.Contains(verifier.SchemaKeywords(), text))
	case "schema":
		return isText && strings.HasPrefix(text, "#/")
	case "rule":
		return text == "witnesses_named_by_witness_hashes"
	case "class":
		return text == "TIME" || text == "NISTTIME"
	case "declared":
		return row.row.Status == "fail"
	case "constructions":
		return text == constructionsMode(proof)
	case "constructions_digest":
		return isText && text == constructionsDigestOf(t, proof) && text != ""
	case "chain":
		return text == "bitcoin" || text == "stellar"
	case "pin_kind":
		return isText && text == pinKindAt(row.row.Part)
	case "pin":
		pins := map[any]map[string]string{"commitment": c.Config.Networks.Commitment, "entropy": c.Config.Networks.Entropy}
		chain, _ := row.values["chain"].(string)
		return isText && text == pins[row.values["pin_kind"]][chain] && text != ""
	}
	return false
}

// pinKindAt is the kind of pin that governs the network at part, read from
// the pointer without the layout (SPEC.md section 9): a commitment entry's,
// under commitments, and a genesis.bitcoin's, under genesis, take the
// commitment pin, and an entropy object's the entropy pin.
func pinKindAt(part *string) string {
	if part == nil {
		return ""
	}
	segments := strings.Split(*part, "/")
	if slices.Contains(segments, "commitments") || slices.Contains(segments, "genesis") {
		return "commitment"
	}
	return "entropy"
}

// unsourced returns each value of a row of SHAPE or POLICY that
// valueSourced does not find computed, as step, part, name and value.
func unsourced(t *testing.T, c vectorcases.Case, o outcome) []string {
	t.Helper()
	proof := proofOf(t, c)
	var out []string
	for _, row := range o.values {
		for name, value := range row.values {
			if !valueSourced(t, c, proof, row, name, value) {
				out = append(out, fmt.Sprintf("%s %s: %s = %v", row.row.Step, row.row, name, value))
			}
		}
	}
	slices.Sort(out)
	return out
}

// Every case of cases.json through the built steps: a case they decide
// gives exactly its code and sub_reason; every other case passes steps 0
// to 3, and a report's rows of the built steps are exactly its expected
// rows, all of them when the gate or SHAPE ends the run, whose label is
// then failed, with the attribution none when the gate ended it and the
// attribution SIGNATURE gave it when SHAPE did; a run the gate opens has
// the attribution its label needs; a run SHAPE passes takes the type and
// subject id its proof carries, stays open through DERIVE, and has a
// failing row exactly when its expected rows of POLICY and of DERIVE do,
// and any other run takes neither; a run that reaches DERIVE has exactly
// the rows and trace of DERIVE that the model of derive_model_test.go
// gives (checkDerive), and a run that ended
// before it has no trace; every row's values write with JCS, and each
// value of a row of SHAPE or POLICY is one the run computed or holds (valueSourced),
// every SHAPE row's keyword "false" or one of verifier.SchemaKeywords()
// among them; the five online cases CONFIGURATION decides give
// configuration_error through the replay adapter; every other online case
// has a source for each pin and reads each pinned network's identity once;
// and the two online cases whose keyring is unavailable,
// key_binding-unavailable-unreachable and
// key_binding-unavailable-load_check, read the keyring of
// www.truestamp.com through the replay adapter and fail KEY_BINDING
// keyring_unavailable. Mutants killed: a failed online keyring read
// binding against the compiled pin (both online key_binding-* cases fail
// key_binding_failed); a replayed keyring loaded under a fixed "test"
// (key_binding-unavailable-load_check's VECTOR_KEY binds); parseNumber
// without its call to profileNumber (the parse-integer-* and parse-number-* cases
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
// differ); Configure without its source confirmation (the five
// configuration-online cases pass CONFIGURATION); the runner without its
// call to Shape (every SHAPE row is missing) or to Policy (every POLICY
// row is missing); Shape giving the subject id of /subject/record/id to
// every type (the block and beacon cases take none, and panic); a SHAPE
// failure setting the attribution none (the 63 that fail at SHAPE); Policy
// ending the run at a failure (the 15 POLICY failures get skip rows);
// shapeSchema giving the failure's schema location as its keyword (every
// check-3 row's keyword is unsourced); policyRow's values giving the
// proof's network as the pin (each POLICY failure's pin is unsourced);
// policyComparisons comparing genesis.bitcoin with the entropy pin (each
// genesis row's pin_kind is unsourced, though every case with a genesis
// row pins one Bitcoin network for both kinds); the runner without its
// call to Derive (every DERIVE row is missing); Derive ending the run at
// a failing row (the 140 cases that fail at DERIVE get skip rows);
// deriveChecks giving the links before the records (in
// declared-genesis_height-witnesses LINK_HEAD's pass comes before the
// genesis height's failure at /witnesses/block/record/height, and in every
// case with a signing key event LINK_KEY_EVENT's row before the check of
// its key's bytes).
func TestCases_ThroughDerive(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 443 {
		t.Fatalf("cases.json holds %d cases, want 443", len(cases))
	}
	tally := map[string]int{}
	names := map[string]bool{}
	for _, c := range cases {
		step := decidedAt(t, c)
		online := c.Config.Online != nil

		t.Run(c.ID, func(t *testing.T) {
			got := run(t, c)
			switch {
			case step != "" && vectorcases.StepIndex(step) <= vectorcases.StepIndex(lastBuilt):
				tally[map[bool]string{false: "rejected at ", true: "online, rejected at "}[online]+step]++
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
			if bad := unsourced(t, c, got); len(bad) != 0 {
				t.Fatalf("values the run did not compute: %q", bad)
			}
			for _, row := range got.values {
				for name := range row.values {
					names[name] = true
				}
			}
			on := map[bool]string{false: "", true: "online "}[online]
			want := c.Expected.Rows
			switch {
			case got.ended && gateFails(want):
				tally[on+"report, the gate fails"]++
				if c.Expected.Label != "failed" || got.attribution != verifier.AttributionNone {
					t.Fatalf("the gate ended the run with attribution %q; the case's label is %s", got.attribution, c.Expected.Label)
				}
			case got.ended:
				tally[on+"report, SHAPE fails"]++
				if c.Expected.Label != "failed" || got.attribution == verifier.AttributionNone || !attributionFits(got.attribution, "failed") {
					t.Fatalf("SHAPE ended the run with attribution %q; the case's label is %s", got.attribution, c.Expected.Label)
				}
			default:
				want = expectedThrough(c)
				fails := func(step string) bool {
					return slices.ContainsFunc(rowsBetween(step, step, want), func(row vectorcases.Row) bool { return row.Status == "fail" })
				}
				switch {
				case fails("POLICY"):
					tally[on+"report, POLICY fails"]++
				case fails("DERIVE"):
					tally[on+"report, DERIVE fails"]++
				default:
					tally[on+"report, passes through DERIVE"]++
				}
				if got.failed != (fails("POLICY") || fails("DERIVE")) {
					t.Fatalf("the run failed %v; its expected rows through DERIVE fail %v", got.failed, fails("POLICY") || fails("DERIVE"))
				}
				if !attributionFits(got.attribution, c.Expected.Label) {
					t.Fatalf("attribution %q, label %s", got.attribution, c.Expected.Label)
				}
			}
			if !rowsEqual(got.rows, want) {
				t.Fatalf("rows = %v, want %v", got.rows, want)
			}
			passed := !got.ended
			typ, subject := typeAndSubject(proofOf(t, c))
			if passed && (got.typ != typ || got.subjectID != subject || subject == "") {
				t.Fatalf("type %q, subject id %q; the proof carries %q, %q", got.typ, got.subjectID, typ, subject)
			}
			if !passed && (got.typ != "" || got.subjectID != "") {
				t.Fatalf("an ended run took the type %q and subject id %q", got.typ, got.subjectID)
			}
			if passed {
				checkDerive(t, c, got.run)
			} else if got.run.Trace != nil {
				t.Fatalf("a run that ended before DERIVE has a trace %v", got.run.Trace)
			}
		})
	}

	want := map[string]int{
		"rejected at CONFIGURATION":            11,
		"online, rejected at CONFIGURATION":    5,
		"rejected at READ":                     1,
		"rejected at PARSE":                    15,
		"rejected at ENVELOPE":                 32,
		"report, the gate fails":               30,
		"report, SHAPE fails":                  63,
		"report, POLICY fails":                 14,
		"report, DERIVE fails":                 139,
		"report, passes through DERIVE":        69,
		"online report, passes through DERIVE": 60,
		"online report, POLICY fails":          1,
		"online report, DERIVE fails":          1,
		"online report, the gate fails":        2,
	}
	if len(tally) != len(want) {
		t.Fatalf("tally = %v, want %v", tally, want)
	}
	for k, n := range want {
		if tally[k] != n {
			t.Fatalf("tally = %v, want %v", tally, want)
		}
	}
	reached := slices.Sorted(maps.Keys(names))
	if wantNames := []string{"chain", "class", "constructions", "constructions_digest", "declared", "keyword",
		"pin", "pin_kind", "rule", "schema"}; !slices.Equal(reached, wantNames) {
		t.Fatalf("the cases reach the value names %q of SHAPE and POLICY, want %q", reached, wantNames)
	}
}

// failingTimes returns each header_time, closed_at and timeStamp of P,
// found here by a walk of P's members, whose class's round trip
// verifier.ParseTime refuses.
func failingTimes(proof map[string]any) []string {
	var out []string
	var walk func(value any)
	walk = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			for name, x := range v {
				class, timed := map[string]verifier.TimeClass{"header_time": verifier.ClassTIME, "closed_at": verifier.ClassTIME,
					"timeStamp": verifier.ClassNISTTIME}[name]
				if _, err := verifier.ParseTime(class, x); timed && err != nil {
					out = append(out, name)
				}
				walk(x)
			}
		case []any:
			for _, x := range v {
				walk(x)
			}
		}
	}
	walk(proof)
	return out
}

// The families cut at SHAPE are SPEC.md section 13.3's, past a gate each
// passes: the 57 shape-* cases, three unsupported_format, two
// unsupported_version, 38 shape_invalid at check 3, the 13 of checks 4 to 6
// (laterShapeChecks), whose proofs the schema finds nothing in, two of
// check 4 at /witnesses, seven of check 5 at a header_time, closed_at or
// timeStamp, and four of check 6, construction_mismatch at /constructions,
// and shape-time-february_29-0000, which passes; the 6 empty_root-* cases, one
// with leaves under the empty root and one with no leaves under another
// root at each of /block, /witnesses/block and /signing_key_event/block,
// each shape_invalid at that record's merkle_root by the schema's
// empty-root rule; and gates-shape, unsupported_version. Each case decided
// by checks 1 to 3 gives its row, and check 3's row is the first in
// document order of the schema's failures; the two shape-first_of_two-schema
// cases each carry two failures of check 3 and give the first; every case
// gives its SHAPE row, checks 4 to 6 included, and the two
// shape-first_of_two-times cases each carry two failures of check 5 and
// give the first in document order. Mutants killed: checks 1 and 2 made after the schema (shape-format-absent and
// shape-version-absent fail shape_invalid); the schema's if applying then
// whatever its schema gives (the empty_root-*-leaves cases pass); the
// failures left in the order the checker meets them
// (shape-first_of_two-schema-mirror fails at previous_block_hash); not
// passing what its schema fails (every proof fails at a not); check 5
// taking the first failing time in the layout's order
// (shape-first_of_two-times-mirror fails at its other time); the leap-year
// rule without its century clause (shape-time-february_29-1900 and -2100
// pass); check 6 accepting every reference
// (shape-reference-not_accepted passes).
func TestCases_FamiliesCutAtShape(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	const emptyRoot = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	checks := map[string]map[string]int{}
	positions := map[string]bool{}
	for _, c := range cases {
		family := c.Family()
		if family != "shape" && family != "empty_root" && c.ID != "gates-shape" {
			continue
		}
		if c.Config.Online != nil || c.Expected.Bare() || gateFails(c.Expected.Rows) {
			t.Fatalf("%s is not an offline report past the gate", c.ID)
		}
		if checks[family] == nil {
			checks[family] = map[string]int{}
		}
		row := shapeRow(t, c)
		proof := proofOf(t, c)
		failures := verifier.ProofFailures(proof)
		code := "pass"
		if row.Code != nil {
			code = *row.Code
		}
		got := run(t, c)
		if check, later := laterShapeChecks[c.ID]; later {
			checks[family][fmt.Sprintf("check %d", check)]++
			part := *row.Part
			ok := len(failures) == 0 && map[int]bool{
				4: part == "/witnesses" && code == "shape_invalid",
				5: code == "shape_invalid" && (strings.HasSuffix(part, "/closed_at") || strings.HasSuffix(part, "/timeStamp") || strings.HasSuffix(part, "/header_time")),
				6: part == "/constructions" && code == "construction_mismatch",
			}[check]
			if !ok {
				t.Errorf("%s: %v, schema failures %v; not a case of check %d", c.ID, row, failures, check)
			}
		} else {
			checks[family][code]++
		}
		if gotShape := rowsBetween("SHAPE", "SHAPE", got.rows); len(gotShape) != 1 || gotShape[0].String() != row.String() {
			t.Errorf("%s: SHAPE rows %v, want %v", c.ID, gotShape, row)
		}
		if strings.HasPrefix(c.ID, "shape-first_of_two-times") {
			if n := len(failingTimes(proof)); n != 2 {
				t.Errorf("%s: %d times fail their round trip, want 2", c.ID, n)
			}
		}
		if _, later := laterShapeChecks[c.ID]; later {
			continue
		}
		switch code {
		case "pass":
			if len(failures) != 0 {
				t.Errorf("%s: schema failures %v", c.ID, failures)
			}
		case "shape_invalid":
			if len(failures) == 0 || failures[0].Part.Encode() != *row.Part {
				t.Errorf("%s: schema failures %v, want the first at %s", c.ID, failures, *row.Part)
			}
		}
		if strings.HasPrefix(c.ID, "shape-first_of_two-schema") {
			parts := map[string]bool{}
			for _, f := range failures {
				parts[f.Part.Encode()] = true
			}
			if len(parts) < 2 {
				t.Errorf("%s: schema failures %v, want two parts", c.ID, failures)
			}
		}
		if family == "empty_root" {
			position, kind, _ := strings.Cut(strings.TrimPrefix(c.ID, "empty_root-"), "-")
			at := map[string][]string{
				"block":                   {"block"},
				"witnesses_block":         {"witnesses", "block"},
				"signing_key_event_block": {"signing_key_event", "block"},
			}[position]
			record := proof
			for _, name := range append(slices.Clone(at), "record") {
				record, _ = record[name].(map[string]any)
			}
			count, _ := record["leaf_count"].(strictjson.Number)
			root, _ := record["merkle_root"].(string)
			wantPart := "/" + strings.Join(at, "/") + "/record/merkle_root"
			holds := map[string]bool{"leaves": count >= 1 && root == emptyRoot, "root": count == 0 && root != emptyRoot && len(root) == 64}[kind]
			if !holds || *row.Part != wantPart || code != "shape_invalid" ||
				!strings.HasPrefix(failures[0].Schema, "#/$defs/BLOCK_RECORD/allOf/0/") {
				t.Errorf("%s: leaf_count %v, merkle_root %q, row %v, failures %v", c.ID, count, root, row, failures)
			}
			positions[position+" "+kind] = true
		}
	}
	want := map[string]map[string]int{
		"shape": {"unsupported_format": 3, "unsupported_version": 2, "shape_invalid": 38, "pass": 1,
			"check 4": 2, "check 5": 7, "check 6": 4},
		"empty_root": {"shape_invalid": 6},
		"gates":      {"unsupported_version": 1},
	}
	for family, codes := range want {
		if len(checks[family]) != len(codes) {
			t.Fatalf("%s: %v, want %v", family, checks[family], codes)
		}
		for code, n := range codes {
			if checks[family][code] != n {
				t.Fatalf("%s: %v, want %v", family, checks[family], codes)
			}
		}
	}
	if len(positions) != 6 {
		t.Fatalf("empty_root positions and kinds: %v", positions)
	}
}

// The 38 sameness-* cases (SPEC.md section 13.3) are cut at SHAPE: each
// passes it, and its SHAPE row's values name its constructions mode, full
// with the digest of its text, reference, or none, as its proof carries
// it: 11 sameness-format-* and the five other spellings in full mode, 11
// sameness-reference-* in reference mode and 11 sameness-none-* with none.
// Every one of them passes SHAPE and POLICY whole, so its rows through
// POLICY are those of TestCases_ThroughDerive. Mutants killed: check 6
// giving the full mode's values for a reference (each
// sameness-reference-* row names full); the pass row with no values (each
// row names no mode); d computed over encoding/json's Marshal of the text
// rather than its JCS (every full-mode case fails construction_mismatch,
// since Marshal escapes the text's < and >).
func TestCases_SamenessCutAtShape(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	modes := map[string]int{}
	for _, c := range cases {
		if c.Family() != "sameness" {
			continue
		}
		got := run(t, c)
		proof := proofOf(t, c)
		var shape []stepValues
		for _, row := range got.values {
			if row.row.Step == "SHAPE" {
				shape = append(shape, row)
			}
		}
		mode := constructionsMode(proof)
		want := map[string]any{"constructions": mode}
		if mode == "full" {
			want["constructions_digest"] = constructionsDigestOf(t, proof)
		}
		if len(shape) != 1 || shape[0].row.String() != passRow("SHAPE").String() || fmt.Sprint(shape[0].values) != fmt.Sprint(want) {
			t.Errorf("%s: SHAPE %v, want a pass with %v", c.ID, shape, want)
		}
		if got.ended || got.failed {
			t.Errorf("%s: the run ended %v, failed %v", c.ID, got.ended, got.failed)
		}
		switch _, kind, _ := strings.Cut(c.ID, "-"); {
		case strings.HasPrefix(kind, "reference-") && mode != "reference",
			strings.HasPrefix(kind, "none-") && mode != "none",
			strings.HasPrefix(kind, "format-") && mode != "full":
			t.Errorf("%s: mode %s", c.ID, mode)
		}
		modes[mode]++
	}
	if want := map[string]int{"full": 16, "reference": 11, "none": 11}; fmt.Sprint(modes) != fmt.Sprint(want) {
		t.Fatalf("sameness modes %v, want %v", modes, want)
	}
}

// The families cut at POLICY are SPEC.md section 13.3's: the 13 policy-*
// cases, one network_mismatch at each network member position, 12 in all,
// at /commitments and /signing_key_event/commitments for each chain, at
// /witnesses/entropy_X and /observations/entropy_X/entropy for Stellar and
// Bitcoin and at /subject/entropy in an entropy_stellar and an
// entropy_bitcoin proof, and at genesis.bitcoin in each of the three
// block positions; and gates-policy and gates-online-policy, the second
// through the replay adapter, each network_mismatch at
// /commitments/bitcoin/network. Each is held to its POLICY rows whole, the
// passes beside its one failure in document order, with the run open after
// POLICY; its failing row's part is the position its id names, and its
// values give the case's pin for that position and the proof's network as
// declared. Six of the policy cases pin an entropy network that differs
// from their E_v's default, as 25 cases of the file do. Mutants killed:
// caseConfig taking the pins from the case's E_v, test, instead of its
// networks (the six entropy cases' POLICY rows differ); pinNetwork giving the
// commitment pin for every kind (the same six cases' rows differ);
// genesisBitcoin left out of policyComparisons (the three genesis cases
// lose their rows); Policy leaving its rows in the layout's order (in
// policy-observations-* /observations sorts before /signing_key_event,
// and in the genesis cases a block position's genesis row sorts among the
// commitments); Policy taking the default pins of the run's E_v (the six
// entropy cases' rows differ).
func TestCases_FamiliesCutAtPolicy(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	differ := 0
	positions := map[string]bool{}
	seen := 0
	for _, c := range cases {
		defaults, _ := specdata.DefaultNetworkPins(c.Config.Environment)
		if c.Config.Networks.Commitment["bitcoin"] != defaults.Commitment.Bitcoin || c.Config.Networks.Commitment["stellar"] != defaults.Commitment.Stellar ||
			c.Config.Networks.Entropy["bitcoin"] != defaults.Entropy.Bitcoin || c.Config.Networks.Entropy["stellar"] != defaults.Entropy.Stellar {
			differ++
		}
		if c.Family() != "policy" && c.ID != "gates-policy" && c.ID != "gates-online-policy" {
			continue
		}
		seen++
		got := run(t, c)
		want := rowsBetween("POLICY", "POLICY", c.Expected.Rows)
		if !rowsEqual(rowsBetween("POLICY", "POLICY", got.rows), want) || got.ended || !got.failed {
			t.Errorf("%s: POLICY rows %v, ended %v, failed %v; want %v", c.ID, rowsBetween("POLICY", "POLICY", got.rows), got.ended, got.failed, want)
			continue
		}
		var failing []stepValues
		for _, row := range got.values {
			if row.row.Step == "POLICY" && row.row.Status == "fail" {
				failing = append(failing, row)
			}
		}
		if len(failing) != 1 || *failing[0].row.Code != "network_mismatch" {
			t.Errorf("%s: failing rows %v", c.ID, failing)
			continue
		}
		part := *failing[0].row.Part
		positions[part] = true
		named := strings.ReplaceAll(strings.TrimPrefix(c.ID, "policy-"), "-", "/")
		named = strings.NewReplacer("witnesses_block", "witnesses/block", "signing_key_event_block", "signing_key_event/block",
			"genesis", "record/metadata/genesis/bitcoin", "observations/entropy_bitcoin", "observations/entropy_bitcoin/entropy",
			"observations/entropy_stellar", "observations/entropy_stellar/entropy", "subject/entropy_stellar", "subject/entropy",
			"subject/entropy_bitcoin", "subject/entropy").Replace(named)
		if c.Family() == "gates" {
			named = "commitments/bitcoin"
		}
		if part != "/"+named+"/network" {
			t.Errorf("%s: fails at %s, its id names /%s/network", c.ID, part, named)
		}
		values := failing[0].values
		declared, _ := values["declared"].(string)
		if valueAt := proofValue(proofOf(t, c), part); declared == "" || declared != valueAt || declared == values["pin"] {
			t.Errorf("%s: values %v, the proof's network %q", c.ID, values, valueAt)
		}
	}
	if seen != 15 || len(positions) != 12 || differ != 25 {
		t.Fatalf("%d cases, %d positions %v, %d cases with pins other than their E_v's", seen, len(positions), positions, differ)
	}
}

// proofValue returns the string at the JSON Pointer part in P, read here
// without the verifier; part holds no escaped name.
func proofValue(proof map[string]any, part string) string {
	var value any = proof
	for _, name := range strings.Split(strings.TrimPrefix(part, "/"), "/") {
		object, _ := value.(map[string]any)
		value = object[name]
	}
	text, _ := value.(string)
	return text
}

// The families cut at SIGNATURE are SPEC.md section 13.3's: the 11
// environment-* cases, each environment_mismatch; the 12 signature-*
// cases, each signature_invalid; the 2 mixed_order-* cases, one passing
// and one signature_invalid, both keyless; the 5 offline key_binding-*
// cases, three key_binding_failed, the inactive key passing and the
// keyless skip; the 2 online key_binding-* cases, each
// keyring_unavailable; and the 9 gates-* cases, three of them failing at
// the gate and six passing it, three of those online. Each runs to its
// expected rows of the gate, the online ones through the replay adapter.
// Mutants killed: keyless mode skipping SIGNATURE (the four keyless
// signature-* cases and mixed_order-k_not_multiple_of_8 pass); a failed
// online keyring read binding against the compiled pin (the two online
// key_binding-* cases fail key_binding_failed).
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
		o := run(t, c)
		if want := rowsBetween("ENVIRONMENT", "SIGNATURE", c.Expected.Rows); !rowsEqual(rowsBetween("ENVIRONMENT", "SIGNATURE", o.rows), want) {
			t.Errorf("%s: gate rows %v, want %v", c.ID, o.rows, want)
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

// The five configuration-online-* cases (SPEC.md sections 10.2, step 0,
// and 13.2) give configuration_error through the replay adapter, each for
// its own reason and after its own identity reads, under the pins the case
// gives, which differ from its E_v's defaults in two of them: a Stellar
// source pinned to public whose passphrase is another network's, and a
// Bitcoin source pinned to mainnet whose height-0 hash is another
// network's, each read with the other chain's source; a Stellar testnet
// source that gives the production passphrase and a Bitcoin regtest source
// that gives the production hash; and regtest pinned with no source, which
// is refused before any identity read. The two reads-identity-* cases pass
// CONFIGURATION, their one unreachable source recorded with the pins it
// serves. Mutants killed: caseConfig taking the pins from the case's E_v,
// test, instead of its networks (public-other_passphrase reads Stellar
// testnet's default, which its file does not hold, and
// mainnet-other_block0 has no regtest source); allows comparing the
// network alone (no mismatch is found).
func TestCases_TheOnlineConfigurationCases(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		message string
		reads   []string
	}{
		"configuration-online-no_source-regtest": {"The run pins Bitcoin's regtest network, and no source serves it", nil},
		"configuration-online-public-other_passphrase": {"gives a network passphrase other than that of Stellar's production network, public",
			[]string{"bitcoin regtest", "stellar public"}},
		"configuration-online-mainnet-other_block0": {"gives a hash of the block at height 0 other than that of Bitcoin's production network, mainnet",
			[]string{"bitcoin mainnet", "stellar testnet"}},
		"configuration-online-testnet-production_passphrase": {"gives the network passphrase of Stellar's production network, public, and the run pins it to testnet",
			[]string{"bitcoin regtest", "stellar testnet"}},
		"configuration-online-regtest-mainnet_block0": {"gives the hash of the block at height 0 of Bitcoin's production network, mainnet, and the run pins it to regtest",
			[]string{"bitcoin regtest", "stellar testnet"}},
	}
	seen := 0
	for _, c := range cases {
		if !strings.HasPrefix(c.ID, "configuration-online-") && !strings.HasPrefix(c.ID, "reads-identity-") {
			continue
		}
		replay := replayOf(t, c)
		r, rejection, err := verifier.Configure(caseConfig(t, c, replay))
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		expected, configurationCase := want[c.ID]
		if !configurationCase {
			chain := strings.TrimSuffix(strings.TrimPrefix(c.ID, "reads-identity-"), "-unreachable")
			record, ok := r.SourceFor(verifier.PinEntropy, chain)
			if rejection != nil || !ok || record.Status != verifier.SourceUnreachable || len(record.Pins) != 2 || record.Reason == "" {
				t.Errorf("%s: %v, %+v", c.ID, rejection, record)
			}
			continue
		}
		seen++
		if rejection == nil || string(rejection.Code) != c.Expected.Code || rejection.Code != specdata.ReasonConfigurationError ||
			!strings.Contains(rejection.Message, expected.message) {
			t.Errorf("%s: %v, want configuration_error: ...%s...", c.ID, rejection, expected.message)
		}
		if got := replay.reads(); !slices.Equal(got, expected.reads) {
			t.Errorf("%s: identity reads %q, want %q", c.ID, got, expected.reads)
		}
	}
	if seen != len(want) {
		t.Fatalf("%d configuration-online cases, want %d", seen, len(want))
	}
}

// failingDerive returns a run's failing DERIVE rows.
func failingDerive(run *verifier.Run) []verifier.Row {
	var out []verifier.Row
	for _, row := range run.Rows {
		if row.Step == "DERIVE" && row.Status == verifier.StatusFail {
			out = append(out, row)
		}
	}
	return out
}

// derivePartsOf writes rows' parts.
func derivePartsOf(rows []verifier.Row) []string {
	var out []string
	for _, row := range rows {
		out = append(out, *row.Part)
	}
	return out
}

// The families cut at DERIVE are SPEC.md section 13.3's: the 42 declared-*
// cases, the 20 paths-*, the 18 offline leaf_count-* and the 2 graft-*,
// and gates-derive, each run through TestCases_ThroughDerive's comparison
// (its rows through DERIVE and its values against the
// model); each family fails these checks in some case, and only with
// derivation_mismatch; gates-derive fails at /subject/item_hash alone. And
// the cases of the committed leaf count (SPEC.md section 5.4) and of
// the graft (section 10.3), at their parts:
//
//   - graft-genuine_block_hash fails at /block/block_hash, recomputed as
//     the model's block_hash, and at its Stellar epoch root, whose walk
//     starts from that recomputation, while the carried block_hash walks
//     to the root;
//   - graft-forged_block_hash fails at its Stellar epoch root alone;
//   - leaf_count-record-subject and -observation, whose block records lie
//     about their leaf count, fail at both epoch roots;
//   - the six leaf_count-tree_size-* cases each fail one row, at an
//     inclusion proof's tree_size, recomputed as its record's leaf_count;
//   - the six leaf_count-moved_index-* cases each fail only at a
//     tree_size, while the walk from the recomputed leaf reaches the
//     record's root, and the same index and path under the committed leaf
//     count reach none (section 5.3).
//
// Mutants killed: the epoch walks starting from the block_hash P carries
// (graft-genuine_block_hash's epoch root passes); the size compared with
// the walk's own size (the tree_size cases pass); the size row left out
// (the moved_index and tree_size cases pass).
func TestCases_FamiliesCutAtDerive(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	families := map[string]int{}
	codes := map[string]map[string]bool{}
	byID := map[string]vectorcases.Case{}
	for _, c := range cases {
		byID[c.ID] = c
		family := c.Family()
		if c.ID == "gates-derive" {
			family = c.ID
		}
		switch family {
		case "declared", "paths", "leaf_count", "graft", "gates-derive":
		default:
			continue
		}
		if c.Config.Online != nil {
			continue
		}
		families[family]++
		got := run(t, c)
		if got.bare() || got.ended || !rowsEqual(got.rows, expectedThrough(c)) {
			t.Fatalf("%s: rows %v, want %v", c.ID, got.rows, expectedThrough(c))
		}
		checkDerive(t, c, got.run)
		if codes[family] == nil {
			codes[family] = map[string]bool{}
		}
		for _, row := range failingDerive(got.run) {
			codes[family][string(row.Code)] = true
		}
	}
	if fmt.Sprint(families) != fmt.Sprint(map[string]int{"declared": 42, "paths": 20, "leaf_count": 18, "graft": 2, "gates-derive": 1}) {
		t.Fatalf("families %v", families)
	}
	for family, found := range codes {
		if fmt.Sprint(found) != "map[derivation_mismatch:true]" {
			t.Errorf("%s fails these checks with %v", family, found)
		}
	}
	single := func(id string) verifier.Row {
		failing := failingDerive(run(t, byID[id]).run)
		if len(failing) != 1 {
			t.Fatalf("%s: failing %q", id, derivePartsOf(failing))
		}
		return failing[0]
	}
	if row := single("gates-derive"); *row.Part != "/subject/item_hash" {
		t.Errorf("gates-derive fails at %s", *row.Part)
	}

	genuine := byID["graft-genuine_block_hash"]
	proof := proofOf(t, genuine)
	failing := failingDerive(run(t, genuine).run)
	stellarRoot := "/commitments/stellar/epoch/record/merkle_root"
	if parts := derivePartsOf(failing); !slices.Equal(parts, []string{"/block/block_hash", stellarRoot}) {
		t.Fatalf("graft-genuine_block_hash fails at %q", parts)
	}
	recomputed := modelHash(t, "block_hash", get(proof, "/block/record"))
	inclusion := get(proof, "/commitments/stellar/inclusion_proof")
	if failing[0].Values["recomputed"] != recomputed || failing[0].Values["declared"] != get(proof, "/block/block_hash") ||
		failing[1].Values["recomputed"] != modelWalk(recomputed, inclusion, 32) ||
		modelWalk(get(proof, "/block/block_hash").(string), inclusion, 32) != failing[1].Values["declared"] {
		t.Fatalf("graft-genuine_block_hash: %v, %v", failing[0].Values, failing[1].Values)
	}
	if row := single("graft-forged_block_hash"); *row.Part != stellarRoot {
		t.Errorf("graft-forged_block_hash fails at %s", *row.Part)
	}

	for _, id := range []string{"leaf_count-record-subject", "leaf_count-record-observation"} {
		if parts := derivePartsOf(failingDerive(run(t, byID[id]).run)); !slices.Equal(parts,
			[]string{"/commitments/bitcoin/epoch/record/merkle_root", stellarRoot}) {
			t.Errorf("%s fails at %q", id, parts)
		}
	}

	// recordOf is the record R an inclusion proof Q walks into.
	recordOf := func(q string) string {
		base, chain, isCommitment := strings.Cut(q, "/commitments/")
		if !isCommitment {
			return "/block/record"
		}
		return base + "/commitments/" + strings.TrimSuffix(chain, "/inclusion_proof") + "/epoch/record"
	}
	sizes, moved := 0, 0
	for _, c := range cases {
		kind := ""
		switch {
		case strings.HasPrefix(c.ID, "leaf_count-tree_size-"):
			kind, sizes = "tree_size", sizes+1
		case strings.HasPrefix(c.ID, "leaf_count-moved_index-"):
			kind, moved = "moved_index", moved+1
		default:
			continue
		}
		proof := proofOf(t, c)
		r := run(t, c).run
		row := single(c.ID)
		q, isSize := strings.CutSuffix(*row.Part, "/tree_size")
		if !isSize || row.Values["declared"] != get(proof, *row.Part) || row.Values["recomputed"] != get(proof, recordOf(q)+"/leaf_count") ||
			row.Values["declared"] == row.Values["recomputed"] {
			t.Errorf("%s: %s %v", c.ID, *row.Part, row.Values)
			continue
		}
		if kind != "moved_index" {
			continue
		}
		rootPart := recordOf(q) + "/merkle_root"
		passes := 0
		for _, trace := range r.Trace {
			if trace.Part == rootPart && slices.Contains(trace.Inputs, q) && trace.Status == verifier.StatusPass {
				passes++
			}
		}
		leafAt := strings.TrimSuffix(q, "/inclusion_proof") + "/record"
		leafName := "observation_hash"
		switch {
		case strings.Contains(q, "/commitments/"):
			leafAt, leafName = strings.Split(q, "/commitments/")[0]+"/block/record", "block_hash"
		case proof["type"] == "item":
			leafName = "item_hash"
		}
		leaf := modelHash(t, leafName, get(proof, leafAt))
		inclusion := get(proof, q).(map[string]any)
		committed := map[string]any{"leaf_index": inclusion["leaf_index"], "path": inclusion["path"], "tree_size": get(proof, recordOf(q)+"/leaf_count")}
		if passes != 1 || modelWalk(leaf, inclusion, 32) != get(proof, rootPart) || modelWalk(leaf, committed, 32) != nil {
			t.Errorf("%s: %d passing walks to %s; under the committed leaf count the walk gives %v", c.ID, passes, rootPart, modelWalk(leaf, committed, 32))
		}
	}
	if sizes != 6 || moved != 6 {
		t.Fatalf("%d tree_size and %d moved_index cases, want 6 of each", sizes, moved)
	}
}

// The 11 sound traces (SPEC.md sections 10.6 and 13.3): each case that
// names a trace file, one for each of vectors/positive/*.trace.json, gives
// exactly the rows of that file, each compared whole, all six members
// written as JCS, inputs included, and in order; every row passes; across
// the 11 the rows name every construction a trace row can name; and the
// files hold 76 rows of the Bitcoin evidence, 4 for each of the 19 Bitcoin
// entries they carry (transaction_hash, header_hash, header_merkle_root
// and header_time), every one of which the run gives. Mutants killed: the
// Bitcoin evidence's transaction_hash or header rows untraced (the 76 rows
// are missing); its raw_transaction or txoutproof rows traced (rows no
// file holds); a header row's inputs naming raw_transaction (every header
// row differs); the transaction_hash in the order hashed rather than
// display order (every transaction_hash row differs); the trace's inputs
// left unsorted (the epoch roots' inputs, /commitments/C/inclusion_proof
// before /block/record, are out of document order); the trace's recomputed
// root given as the leaf (every inclusion root row differs); the genesis
// height's trace row left out (item-genesis-witness's and block's differ);
// a handover signature's recomputed given as the signature or its inputs
// without the event (every handover_signature row differs); the key bytes
// check traced (a row at each key event's public_key that no trace holds);
// LINK_WITNESS untraced (the witness_hashes rows are missing).
func TestCases_TheSoundTraces(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(vectorcases.Dir(), "positive", "*.trace.json"))
	if err != nil || len(files) != 11 {
		t.Fatalf("trace files %q, %v", files, err)
	}
	traced := map[string]bool{}
	constructions := map[string]bool{}
	bitcoinRows := map[string]int{}
	entries := 0
	for _, c := range cases {
		if c.Expected.Trace == "" {
			continue
		}
		got := run(t, c)
		var rows []string
		for _, row := range got.run.Trace {
			encoded, err := jcs.Encode(row.Value())
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, string(encoded))
			constructions[row.Construction] = true
			if row.Status != verifier.StatusPass {
				t.Errorf("%s: a trace row %s fails", c.ID, row.Part)
			}
		}
		want := traceFile(t, c.Expected.Trace)
		bitcoin := 0
		for _, row := range want {
			var parsed struct {
				Construction string `json:"construction"`
			}
			if err := json.Unmarshal([]byte(row), &parsed); err != nil {
				t.Fatal(err)
			}
			if parsed.Construction == "bitcoin" {
				bitcoin++
			}
		}
		if len(want) == 0 || !slices.Equal(rows, want) {
			t.Errorf("%s: trace\n  %s\nwant\n  %s", c.ID, strings.Join(rows, "\n  "), strings.Join(want, "\n  "))
		}
		bitcoinRows[c.Expected.Trace] = bitcoin
		if !traced[c.Expected.Trace] {
			for _, n := range []string{"/commitments/bitcoin", "/signing_key_event/commitments/bitcoin"} {
				if get(proofOf(t, c), n) != nil {
					entries++
				}
			}
		}
		traced[c.Expected.Trace] = true
	}
	if len(traced) != 11 {
		t.Fatalf("%d trace files named by the cases, want 11", len(traced))
	}
	for _, file := range files {
		relative, _ := filepath.Rel(vectorcases.Dir(), file)
		if !traced[filepath.ToSlash(relative)] {
			t.Errorf("no case names %s", relative)
		}
	}
	total := 0
	for _, n := range bitcoinRows {
		total += n
	}
	if total != 76 || entries != 19 {
		t.Fatalf("the sound traces hold %d Bitcoin rows for %d Bitcoin entries, want 76 for 19: %v", total, entries, bitcoinRows)
	}
	if got := slices.Sorted(maps.Keys(constructions)); !slices.Equal(got, tracedConstructions) {
		t.Fatalf("the sound traces reach %q, want %q", got, tracedConstructions)
	}
}

// keyEventFamilies are the families of the key events and the links after
// LINK_INCLUSION (SPEC.md section 13.3), with their sizes.
var keyEventFamilies = map[string]int{"key_events": 26, "handover_position": 4, "order": 12, "self_signed": 1}

// keyEventPositions name a case's block position, and keyEventMembers the
// part its member names under that block node.
var (
	keyEventPositions = map[string]string{"block": "/block", "witnesses": "/witnesses/block", "signing_key_event": "/signing_key_event/block"}
	keyEventMembers   = map[string]string{
		"handover_signature":           "/record/metadata/key_event/handover_signature",
		"identity_previous_public_key": "/record/metadata/key_event/handover_signature",
		"key_id":                       "/record/metadata/key_event/key_id",
		"previous_key_id":              "/record/metadata/key_event/previous_key_id",
		"signing_key_id":               "/record/signing_key_id",
		"public_key_small_order":       "/record/metadata/key_event/public_key",
		"prerotation_active_key_id":    "/record/signing_key_id",
	}
	orderPositions = map[string]string{"subject": "/subject/entropy", "witness": "/witnesses/entropy_%s", "observation": "/observations/entropy_%s/entropy"}
)

// signingKeyEventKey is LINK_KEY_EVENT's part.
const signingKeyEventKey = "/signing_key_event/block/record/metadata/key_event/public_key"

// expectedFailure is the code and part a key-event family case's name
// gives its one failing row, or "" for a case that fails nothing.
func expectedFailure(t *testing.T, id string) string {
	t.Helper()
	family, rest, _ := strings.Cut(id, "-")
	switch {
	case id == "key_events-link_key_event":
		return "key_event_mismatch " + signingKeyEventKey
	case family == "key_events":
		parts := strings.Split(rest, "-")
		return "key_event_mismatch " + keyEventPositions[parts[len(parts)-1]] + keyEventMembers[parts[0]]
	case family == "handover_position":
		member, position, _ := strings.Cut(rest, "-")
		return "key_event_mismatch " + keyEventPositions[position] + "/record/metadata/key_event/" + member
	case strings.HasPrefix(id, "order-head-"):
		return "ordering_violation /witnesses/block/record/height"
	case strings.HasPrefix(id, "order-bitcoin_height_equal-"), id == "self_signed-genesis":
		return ""
	case family == "order":
		what, where, _ := strings.Cut(rest, "-")
		source, member := "bitcoin", "height"
		if strings.Contains(what, "stellar") {
			source, member = "stellar", "sequence"
		}
		object := orderPositions[where]
		if strings.Contains(object, "%s") {
			object = fmt.Sprintf(object, source)
		}
		return "ordering_violation " + object + "/payload/" + member
	}
	t.Fatalf("%s is not a case of the key-event families", id)
	return ""
}

// The key_events (26), handover_position (4), order (12) and self_signed
// (1) cases, cut at DERIVE, each run through TestCases_ThroughDerive's
// comparison (its rows through DERIVE, and its values and
// trace against the model): each fails only at the part and with the code
// its name gives (none for the equal Bitcoin heights and self_signed); each
// has LINK_KEY_EVENT's row info not_attributable, since every one runs
// under a supplied keyring or keyless, but key_events-link_key_event, whose
// event introduces another key and fails; no run ends or takes the
// attribution truestamp, and the info row fails no run; and the refusals
// name the step of SPEC.md section 6.2 that refused the key or the
// signature: small_order_key for the identity outgoing key and the
// small-order genesis key, equation for a changed handover signature.
// Mutants killed: LINK_KEY_EVENT passing under a supplied keyring (no info
// row); the info row counted as a failure (self_signed-genesis and the
// equal heights fail); the handover's height compared with the event's
// own (the handover_position height cases pass); a refused handover named
// not_verified, as one step (the changed signature's refusal is not
// equation).
func TestCases_KeyEventFamiliesCutAtDerive(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	families := map[string]int{}
	refusals := map[string]string{}
	for _, c := range cases {
		if _, ok := keyEventFamilies[c.Family()]; !ok {
			continue
		}
		families[c.Family()]++
		got := run(t, c)
		if got.bare() || got.ended || !rowsEqual(got.rows, expectedThrough(c)) {
			t.Fatalf("%s: rows %v, want %v", c.ID, got.rows, expectedThrough(c))
		}
		checkDerive(t, c, got.run)
		var failures, infos []string
		for _, row := range got.run.Rows {
			if row.Step != "DERIVE" {
				continue
			}
			switch row.Status {
			case verifier.StatusFail:
				failures = append(failures, string(row.Code)+" "+*row.Part)
				if refused, ok := row.Values["refused"].(string); ok {
					refusals[c.ID] = refused
				}
			case verifier.StatusInfo:
				infos = append(infos, string(row.Code)+" "+*row.Part)
			}
		}
		var want []string
		if failure := expectedFailure(t, c.ID); failure != "" {
			want = []string{failure}
		}
		if !slices.Equal(failures, want) || got.failed != (len(want) > 0) {
			t.Errorf("%s: failing %q, failed %v; want %q", c.ID, failures, got.failed, want)
		}
		wantInfos := []string{"not_attributable " + signingKeyEventKey}
		if c.ID == "key_events-link_key_event" {
			wantInfos = nil
		}
		if !slices.Equal(infos, wantInfos) || got.attribution == verifier.AttributionTruestamp {
			t.Errorf("%s: info rows %q under %s", c.ID, infos, got.attribution)
		}
	}
	if fmt.Sprint(families) != fmt.Sprint(keyEventFamilies) {
		t.Fatalf("families %v, want %v", families, keyEventFamilies)
	}
	for id, refused := range map[string]string{
		"key_events-identity_previous_public_key-block":   "small_order_key",
		"key_events-public_key_small_order-witnesses":     "small_order_key",
		"key_events-handover_signature-signing_key_event": "equation",
	} {
		if refusals[id] != refused {
			t.Errorf("%s: refused %q, want %q", id, refusals[id], refused)
		}
	}
}

// bitcoinRules are the bitcoin cases (SPEC.md section 13.3), by the part
// of their one failing row and, for a structure that is not well formed,
// the rule it breaks.
var bitcoinRules = map[string][2]string{
	"bitcoin-op_return_other_value-commitments":                {"/commitments/bitcoin/raw_transaction", ""},
	"bitcoin-op_return_other_value-signing_key_event":          {"/signing_key_event/commitments/bitcoin/raw_transaction", ""},
	"bitcoin-raw_transaction_trailing_bytes-commitments":       {"/commitments/bitcoin/raw_transaction", "parse"},
	"bitcoin-raw_transaction_trailing_bytes-signing_key_event": {"/signing_key_event/commitments/bitcoin/raw_transaction", "parse"},
	"bitcoin-txoutproof_trailing_bytes-commitments":            {"/commitments/bitcoin/txoutproof", "parse"},
	"bitcoin-txoutproof_trailing_bytes-signing_key_event":      {"/signing_key_event/commitments/bitcoin/txoutproof", "parse"},
	"bitcoin-raw_transaction_64_octets":                        {"/commitments/bitcoin/raw_transaction", "length"},
	"bitcoin-raw_transaction_no_input":                         {"/commitments/bitcoin/raw_transaction", "parse"},
	"bitcoin-raw_transaction_two_op_returns":                   {"/commitments/bitcoin/raw_transaction", "op_return"},
	"bitcoin-raw_transaction_op_return_pushdata":               {"/commitments/bitcoin/raw_transaction", "op_return"},
	"bitcoin-txoutproof_duplicate_pair":                        {"/commitments/bitcoin/txoutproof", "duplicate_pair"},
	"bitcoin-txoutproof_unused_hash":                           {"/commitments/bitcoin/txoutproof", "partial_tree"},
	"bitcoin-txoutproof_unused_flag_bit":                       {"/commitments/bitcoin/txoutproof", "partial_tree"},
	"bitcoin-txoutproof_total_txs_0":                           {"/commitments/bitcoin/txoutproof", "total_txs"},
	"bitcoin-txoutproof_total_txs_above_max":                   {"/commitments/bitcoin/txoutproof", "total_txs"},
	"bitcoin-txoutproof_header_above_target":                   {"/commitments/bitcoin/txoutproof", "target"},
	"bitcoin-txoutproof_nbits_overflow":                        {"/commitments/bitcoin/txoutproof", "target"},
	"bitcoin-txoutproof_nbits_negative":                        {"/commitments/bitcoin/txoutproof", "target"},
	"bitcoin-unmatched_transaction":                            {"/commitments/bitcoin/txoutproof", ""},
}

// The 19 bitcoin cases (SPEC.md section 13.3), cut at DERIVE, each run
// through TestCases_ThroughDerive's comparison (its rows through DERIVE,
// and its values and trace against the model): each fails at DERIVE only
// at the part bitcoinRules gives it, with bitcoin_evidence_invalid and the
// rule named when the structure is not well formed, every rule of a
// raw_transaction and a txoutproof among them, and with
// derivation_mismatch otherwise; a malformed structure has one row at its
// part, its OP_RETURN or its matched hash unread, and a malformed
// txoutproof no header row; and no run ends. And the OP_RETURN is held to
// the recomputed epoch_hash (SPEC.md section 5.4): in
// leaf_count-record-commitments-bitcoin and
// leaf_count-record-signing_key_event-bitcoin, whose epoch records lie
// about their leaf count with epoch_hash declared as the lying record's
// hash, the OP_RETURN row alone fails, declared as the 32 bytes the
// OP_RETURN pushes and recomputed as the model's hash of the record; and
// in declared-epoch_hash-commitments-bitcoin and
// declared-epoch_hash-signing_key_event-bitcoin, whose epoch_hash alone is
// changed, the OP_RETURN row passes beside the failing epoch_hash. Mutants
// killed: the OP_RETURN compared with the carried epoch_hash (the two
// declared-epoch_hash cases fail it); a well-formedness row's rule
// misnamed or left out; the OP_RETURN or header rows made for a malformed
// structure (a second row at its part, or header rows a malformed
// txoutproof should not give).
func TestCases_BitcoinFamilyCutAtDerive(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	seen, rules := 0, map[string]bool{}
	epochCases := 0
	for _, c := range cases {
		id := c.ID
		epochCase := strings.HasPrefix(id, "leaf_count-record-") && strings.HasSuffix(id, "-bitcoin") ||
			strings.HasPrefix(id, "declared-epoch_hash-") && strings.HasSuffix(id, "-bitcoin")
		if c.Family() != "bitcoin" && !epochCase {
			continue
		}
		got := run(t, c)
		if got.bare() || got.ended || !rowsEqual(got.rows, expectedThrough(c)) {
			t.Fatalf("%s: rows %v, want %v", id, got.rows, expectedThrough(c))
		}
		checkDerive(t, c, got.run)
		failed := failingDerive(got.run)
		if epochCase {
			epochCases++
			proof := proofOf(t, c)
			base := "/commitments/bitcoin"
			if strings.Contains(id, "-signing_key_event-") {
				base = "/signing_key_event" + base
			}
			var opReturn []verifier.Row
			for _, row := range got.run.Rows {
				if row.Step == "DERIVE" && *row.Part == base+"/raw_transaction" {
					opReturn = append(opReturn, row)
				}
			}
			raw := get(proof, base+"/raw_transaction").(string)
			at := strings.Index(raw, "226a20")
			if at < 0 || strings.Count(raw, "226a20") != 1 {
				t.Fatalf("%s: no one output script 0x6a 0x20 in %s", id, raw)
			}
			pushed, recomputed := raw[at+6:at+6+64], modelHash(t, "epoch_hash", get(proof, base+"/epoch/record"))
			wantFailing := []string{base + "/raw_transaction"}
			if strings.HasPrefix(id, "declared-") {
				wantFailing = []string{base + "/epoch/epoch_hash"}
				if len(opReturn) != 2 || opReturn[1].Status != verifier.StatusPass || pushed != recomputed || pushed == get(proof, base+"/epoch/epoch_hash") {
					t.Errorf("%s: the OP_RETURN rows %+v", id, opReturn)
				}
			} else if len(opReturn) != 2 || opReturn[1].Values["declared"] != pushed || opReturn[1].Values["recomputed"] != recomputed ||
				pushed == recomputed || recomputed != get(proof, base+"/epoch/epoch_hash") {
				t.Errorf("%s: the OP_RETURN rows %+v; it pushes %s", id, opReturn, pushed)
			}
			if got := derivePartsOf(failed); !slices.Equal(got, wantFailing) {
				t.Errorf("%s: failing %q, want %q", id, got, wantFailing)
			}
			continue
		}
		seen++
		want, ok := bitcoinRules[id]
		if !ok {
			t.Fatalf("%s is not among bitcoinRules", id)
		}
		code := "derivation_mismatch"
		if want[1] != "" {
			code = "bitcoin_evidence_invalid"
			rules[want[1]] = true
		}
		if len(failed) != 1 || *failed[0].Part != want[0] || string(failed[0].Code) != code || (want[1] != "" && failed[0].Values["rule"] != want[1]) {
			t.Errorf("%s: failing %q %v, want %s %s rule %q", id, derivePartsOf(failed), failed, code, want[0], want[1])
			continue
		}
		atPart := 0
		for _, row := range got.run.Rows {
			if row.Step == "DERIVE" && *row.Part == want[0] {
				atPart++
			}
		}
		if wantRows := map[bool]int{false: 2, true: 1}[want[1] != ""]; atPart != wantRows {
			t.Errorf("%s: %d rows at %s, want %d", id, atPart, want[0], wantRows)
		}
		if strings.HasSuffix(want[0], "/txoutproof") && want[1] != "" {
			for _, row := range got.run.Rows {
				if row.Step == "DERIVE" && strings.HasPrefix(*row.Part, strings.TrimSuffix(want[0], "txoutproof")+"header_") {
					t.Errorf("%s: a header row at %s", id, *row.Part)
				}
			}
		}
	}
	allRules := []string{"duplicate_pair", "length", "op_return", "parse", "partial_tree", "target", "total_txs"}
	if seen != 19 || epochCases != 4 || !slices.Equal(slices.Sorted(maps.Keys(rules)), allRules) {
		t.Fatalf("%d bitcoin cases, %d epoch cases, rules %v; want 19, 4, %q", seen, epochCases, rules, allRules)
	}
}
