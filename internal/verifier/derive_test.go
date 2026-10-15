// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// Step 9, DERIVE (SPEC.md section 10.2), parts (a) to (c) and
// LINK_INCLUSION, on the 11 positive proofs and mutants of them, beside the
// vector cases of internal/vectorcases' runner. It ports the parts of the
// reference's elixir/test/verifier/derive_test.exs these checks reach,
// checked against the application's derive_test.exs:
//
//   - the committed leaf count (SPEC.md section 5.4) at the block and the
//     epoch level: a tree_size that is not the record's leaf_count, and a
//     moved leaf index (the last of three leaves as index 1 of two,
//     section 5.3), each with a walk that still reaches the root;
//   - one test for each rule of SPEC.md section 10.6;
//   - the trace (section 10.6): its rows, the genesis height's row, a
//     recomputed null where a walk gives no root;
//   - the engine, and what DERIVE does to the run.
//
// DERIVE reads P alone, so these runs hold a P past POLICY with no
// signature over it; every mutant is otherwise its positive proof
// unchanged.

// deriveAttribution is the attribution of these runs: a supplied keyring's.
const deriveAttribution = "keyring:0000000000000000000000000000000000000000000000000000000000000000"

// deriveRun is a run past POLICY holding proof, with the attribution
// attribution.
func deriveRun(proof map[string]any, attribution string) *Run {
	var rows []Row
	for _, step := range []string{"CONFIGURATION", "READ", "PARSE", "ENVELOPE", "ENVIRONMENT", "KEY_BINDING", "SIGNATURE", "SHAPE"} {
		rows = append(rows, PassRow(step, nil, "held", nil))
	}
	typ, _ := proof["type"].(string)
	return &Run{Proof: proof, Attribution: attribution, Type: typ, Rows: rows}
}

// derived is the run of proof after DERIVE.
func derived(proof map[string]any) *Run {
	run := deriveRun(proof, deriveAttribution)
	Derive(run)
	return run
}

// deriveRows are the run's DERIVE rows.
func deriveRows(run *Run) []Row {
	var out []Row
	for _, row := range run.Rows {
		if row.Step == deriveStep {
			out = append(out, row)
		}
	}
	return out
}

// failing writes each failing DERIVE row as "code part".
func failing(run *Run) []string {
	var out []string
	for _, row := range deriveRows(run) {
		if row.Status == StatusFail {
			part := "null"
			if row.Part != nil {
				part = *row.Part
			}
			out = append(out, string(row.Code)+" "+part)
		}
	}
	return out
}

// traceAt returns the run's trace rows at part.
func traceAt(run *Run, part string) []TraceRow {
	var out []TraceRow
	for _, row := range run.Trace {
		if row.Part == part {
			out = append(out, row)
		}
	}
	return out
}

// inclusionAt is an inclusion proof Q of a positive proof, the record R
// it walks into, and its level, block or epoch.
type inclusionAt struct {
	name  string
	q, r  Pointer
	level string
}

// inclusionPositions are the inclusion proofs of the positive proofs whose
// leaf_index and tree_size are index and size, read here without the
// layout.
func inclusionPositions(t *testing.T, index, size int) []inclusionAt {
	t.Helper()
	proofs := positiveProofs(t)
	var out []inclusionAt
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		var holders []Pointer
		switch proof["type"] {
		case "block":
		case "beacon":
			for _, source := range sortedNames(proof["observations"].(map[string]any)) {
				holders = append(holders, Pointer{Name("observations"), Name(source)})
			}
		default:
			holders = append(holders, Pointer{Name("subject")})
		}
		var all []inclusionAt
		for _, h := range holders {
			all = append(all, inclusionAt{name, under(h, "inclusion_proof"), Pointer{Name("block"), Name("record")}, "block"})
		}
		for _, base := range []Pointer{{Name("commitments")}, {Name("signing_key_event"), Name("commitments")}} {
			for _, chain := range []string{"bitcoin", "stellar"} {
				if _, ok := valueAt(proof, under(base, chain)).(map[string]any); ok {
					all = append(all, inclusionAt{name, under(base, chain, "inclusion_proof"), under(base, chain, "epoch", "record"), "epoch"})
				}
			}
		}
		for _, at := range all {
			q := valueAt(proof, at.q).(map[string]any)
			if q["leaf_index"] == strictjson.Number(index) && q["tree_size"] == strictjson.Number(size) {
				out = append(out, at)
			}
		}
	}
	return out
}

// levels names the levels of positions.
func levels(positions []inclusionAt) []string {
	var out []string
	for _, p := range positions {
		if !slices.Contains(out, p.level) {
			out = append(out, p.level)
		}
	}
	slices.Sort(out)
	return out
}

// Every positive proof passes DERIVE: each of its rows passes, each row has
// its trace row, and the run does not end. Mutant killed: block_hash
// recomputed under the epoch_hash byte (every block_hash row fails).
func TestDerive_EveryPositiveProofPasses(t *testing.T) {
	for name, proof := range positiveProofs(t) {
		run := derived(proof)
		rows := deriveRows(run)
		if len(rows) == 0 || len(failing(run)) != 0 || run.Ended || len(run.Trace) != len(rows) {
			t.Errorf("%s: failing %q, %d rows, %d trace rows, ended %v", name, failing(run), len(rows), len(run.Trace), run.Ended)
		}
	}
}

// A tree_size that is not the record's leaf_count fails at that size alone,
// at both levels, though the walk still reaches the root: index 0 of three
// leaves walks the same two nodes as index 0 of four (SPEC.md sections 5.3
// and 5.4). The size's trace row is {inclusion, 4, [R/leaf_count], Q/
// tree_size, 3, fail}. Mutants killed: the size row left out (no failure);
// the size compared with the proof's own tree_size (the size passes).
func TestDerive_ATreeSizeThatIsNotTheLeafCount(t *testing.T) {
	found := inclusionPositions(t, 0, 3)
	if !slices.Equal(levels(found), []string{"block", "epoch"}) {
		t.Fatalf("index 0 of 3 at the levels %q", levels(found))
	}
	proofs := positiveProofs(t)
	for _, at := range found {
		run := derived(put(proofs[at.name], under(at.q, "tree_size"), strictjson.Number(4)))
		size := under(at.q, "tree_size").Encode()
		if got := failing(run); !slices.Equal(got, []string{"derivation_mismatch " + size}) {
			t.Errorf("%s %s: failing %q", at.name, at.q.Encode(), got)
		}
		root := traceAt(run, under(at.r, "merkle_root").Encode())
		if len(root) == 0 || root[0].Status != StatusPass {
			t.Errorf("%s %s: the root's trace %+v", at.name, at.q.Encode(), root)
		}
		want := []TraceRow{{Construction: "inclusion", Declared: strictjson.Number(4), Inputs: []string{under(at.r, "leaf_count").Encode()},
			Part: size, Recomputed: strictjson.Number(3), Status: StatusFail}}
		if got := traceAt(run, size); !reflect.DeepEqual(got, want) {
			t.Errorf("%s %s: the size's trace %+v, want %+v", at.name, at.q.Encode(), got, want)
		}
	}
}

// A moved leaf index, the last of three leaves presented as index 1 of two,
// fails at its size alone, at both levels: the walk alone accepts the move
// (SPEC.md section 5.3), so only the committed count catches it. Mutant
// killed: the size row left out (the moved proof passes).
func TestDerive_AMovedLeafIndex(t *testing.T) {
	found := inclusionPositions(t, 2, 3)
	if !slices.Equal(levels(found), []string{"block", "epoch"}) {
		t.Fatalf("index 2 of 3 at the levels %q", levels(found))
	}
	proofs := positiveProofs(t)
	for _, at := range found {
		sound := proofs[at.name]
		moved := put(put(sound, under(at.q, "leaf_index"), strictjson.Number(1)), under(at.q, "tree_size"), strictjson.Number(2))
		run := derived(moved)
		rootPart := under(at.r, "merkle_root").Encode()
		var walk *TraceRow
		for _, row := range traceAt(run, rootPart) {
			if slices.Contains(row.Inputs, at.q.Encode()) {
				walk = &row
			}
		}
		if walk == nil || walk.Status != StatusPass || walk.Recomputed != valueAt(sound, under(at.r, "merkle_root")) {
			t.Errorf("%s %s: the walk's trace %+v", at.name, at.q.Encode(), walk)
		}
		if got := failing(run); !slices.Equal(got, []string{"derivation_mismatch " + under(at.q, "tree_size").Encode()}) {
			t.Errorf("%s %s: failing %q", at.name, at.q.Encode(), got)
		}
	}
}

// A record whose leaf_count is 0 fails every inclusion proof into it, at
// its size (SPEC.md section 10.3). Mutant killed: the size row left out.
func TestDerive_ALeafCountOfZeroFailsEveryProofIntoIt(t *testing.T) {
	run := derived(put(positiveProofs(t)["item"], Pointer{Name("block"), Name("record"), Name("leaf_count")}, strictjson.Number(0)))
	if !slices.Contains(failing(run), "derivation_mismatch /subject/inclusion_proof/tree_size") {
		t.Fatalf("failing %q", failing(run))
	}
}

// longPath is a path of n nodes from leaf index 0 of a tree of 2^n leaves,
// with the root SPEC.md section 5.3's walk gives from leaf along it, with
// no cap on its length, computed here.
func longPath(leaf [32]byte, n int) (map[string]any, string) {
	path := make([]any, n)
	r := sha256.Sum256(append([]byte{0}, leaf[:]...))
	for i := range n {
		node := sha256.Sum256([]byte{byte(i + 1)})
		path[i] = hex.EncodeToString(node[:])
		r = sha256.Sum256(append(append([]byte{1}, r[:]...), node[:]...))
	}
	return map[string]any{"leaf_index": strictjson.Number(0), "path": path, "tree_size": strictjson.Number(float64(uint64(1) << n))}, hex.EncodeToString(r[:])
}

// modelHash is H(p, R) of SPEC.md section 4.1, with p the byte
// spec/prefixes.json gives the row name, computed here.
func modelHash(t *testing.T, name string, record any) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(vectorcases.Dir(), "..", "spec", "prefixes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var prefixes map[string]struct {
		Byte string `json:"byte"`
	}
	if err := json.Unmarshal(data, &prefixes); err != nil {
		t.Fatal(err)
	}
	b, err := strconv.ParseUint(strings.TrimPrefix(prefixes[name].Byte, "0x"), 16, 8)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	encoded, err := jcs.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(append([]byte{byte(b)}, encoded...))
	return hex.EncodeToString(sum[:])
}

// A subject's 33-node path that reaches its root under no cap fails at the
// root, recomputed null in its row and its trace row, since the walk takes
// at most MAX_PATH_STEPS (32) nodes and gives no root; its size, which the
// record commits, passes; a 32-node path to its root passes there.
// Mutants killed: the walk's cap left out (the 33-node root passes); the
// cap one lower (the 32-node root fails); a failed walk's root given as the
// empty string (recomputed is not null).
func TestDerive_AWalkThatGivesNoRootIsRecomputedNull(t *testing.T) {
	item := positiveProofs(t)["item"]
	leafHex := modelHash(t, "item_hash", valueAt(item, Pointer{Name("subject"), Name("record")}))
	leaf, _ := hex.DecodeString(leafHex)
	inclusion, root := longPath([32]byte(leaf), 33)
	proof := put(item, Pointer{Name("subject"), Name("inclusion_proof")}, inclusion)
	proof = put(proof, Pointer{Name("block"), Name("record"), Name("merkle_root")}, root)
	proof = put(proof, Pointer{Name("block"), Name("record"), Name("leaf_count")}, inclusion["tree_size"])
	run := derived(proof)
	var at []Row
	for _, row := range deriveRows(run) {
		if *row.Part == "/block/record/merkle_root" {
			at = append(at, row)
		}
	}
	want := map[string]any{"construction": "inclusion", "declared": root, "recomputed": nil}
	if len(at) != 1 || at[0].Status != StatusFail || !reflect.DeepEqual(at[0].Values, want) {
		t.Fatalf("the root's rows %+v", at)
	}
	trace := traceAt(run, "/block/record/merkle_root")
	if len(trace) != 1 || trace[0].Recomputed != nil || trace[0].Status != StatusFail {
		t.Fatalf("the root's trace %+v", trace)
	}
	encoded, err := jcs.Encode(trace[0].Value())
	if err != nil || !strings.Contains(string(encoded), `"recomputed":null`) {
		t.Fatalf("the trace row writes %s, %v", encoded, err)
	}
	for _, row := range deriveRows(run) {
		if *row.Part == "/subject/inclusion_proof/tree_size" && row.Status != StatusPass {
			t.Fatalf("the size fails: %+v", row)
		}
	}
	at32, root32 := longPath([32]byte(leaf), 32)
	proof = put(put(put(item, Pointer{Name("subject"), Name("inclusion_proof")}, at32),
		Pointer{Name("block"), Name("record"), Name("merkle_root")}, root32),
		Pointer{Name("block"), Name("record"), Name("leaf_count")}, at32["tree_size"])
	for _, row := range deriveRows(derived(proof)) {
		if *row.Part == "/block/record/merkle_root" && row.Status != StatusPass {
			t.Fatalf("a 32-node path fails at its root: %+v", row)
		}
	}
}

// The genesis checks of SPEC.md section 10.2, step 9 (b), each give a
// trace row: a genesis /witnesses/block's height row, recomputed 0, is in
// the trace though LINK_HEAD's row at the same part will not be (section
// 10.6), and a genesis height other than 0 fails, recomputed 0. Mutants
// killed: the genesis rows untraced (no trace row at the height); the
// height compared with the value it declares (the mutant passes).
func TestDerive_TheGenesisHeightRowIsInTheTrace(t *testing.T) {
	proof := positiveProofs(t)["item-genesis-witness"]
	height := Pointer{Name("witnesses"), Name("block"), Name("record"), Name("height")}
	run := derived(proof)
	want := []TraceRow{{Construction: "genesis", Declared: strictjson.Number(0),
		Inputs: []string{"/witnesses/block/record/metadata/genesis"}, Part: height.Encode(), Recomputed: strictjson.Number(0), Status: StatusPass}}
	if got := traceAt(run, height.Encode()); !reflect.DeepEqual(got, want) {
		t.Fatalf("the height's trace %+v, want %+v", got, want)
	}
	mutant := derived(put(proof, height, strictjson.Number(5)))
	got := traceAt(mutant, height.Encode())
	if len(got) != 1 || got[0].Status != StatusFail || got[0].Recomputed != strictjson.Number(0) || got[0].Declared != strictjson.Number(5) {
		t.Fatalf("a genesis height of 5: %+v", got)
	}
	if !slices.Contains(failing(mutant), "derivation_mismatch "+height.Encode()) {
		t.Fatalf("failing %q", failing(mutant))
	}
}

// The rules of SPEC.md section 10.6.

// Rule 1: DERIVE gives the same rows and trace whatever constructions
// holds, or when it is absent; constructions_static_test.go holds the
// module to SHAPE's one read. Mutant killed: DERIVE giving no rows to a
// proof without constructions.
func TestDerive_Rule1_TheSameWhateverConstructionsHolds(t *testing.T) {
	proofs := positiveProofs(t)
	for _, name := range []string{"item", "block-rotation", "beacon"} {
		proof := proofs[name]
		base := derived(proof)
		reference := constructionsReferencePrefix + strings.Repeat("0", 64) + constructionsReferenceSuffix
		var others []*Run
		for _, value := range []any{reference, map[string]any{"not": "the text"}, strictjson.Number(1), nil} {
			others = append(others, derived(put(proof, Pointer{Name("constructions")}, value)))
		}
		others = append(others, derived(remove(proof, Pointer{Name("constructions")})))
		for i, other := range others {
			if !reflect.DeepEqual(other.Rows, base.Rows) || !reflect.DeepEqual(other.Trace, base.Trace) {
				t.Errorf("%s, value %d: rows %s, want %s", name, i, rowSummary(deriveRows(other)), rowSummary(deriveRows(base)))
			}
		}
	}
}

// Rule 2: prefix bytes come only from the compiled registry. Each record
// hash DERIVE recomputes is H(p, R) under the byte spec/prefixes.json gives
// its construction, computed here from the file; the static checks of
// internal/registry/rule2_test.go hold DERIVE to registry constants at
// each call and to no member set read from a document. Mutant killed: the
// subject's item_hash recomputed under the claims_hash byte (the trace's
// recomputed differs from the model's).
func TestDerive_Rule2_EachHashIsUnderItsRowsByte(t *testing.T) {
	seen := map[string]bool{}
	for name, proof := range positiveProofs(t) {
		run := derived(proof)
		for _, row := range run.Trace {
			kind := row.Construction
			switch {
			case kind == "inclusion" || (kind == "genesis" && strings.HasSuffix(row.Part, "/height")):
				continue
			case kind == "genesis":
				kind = "genesis_previous_block_hash"
			}
			if len(row.Inputs) != 1 {
				t.Fatalf("%s %s: inputs %q", name, row.Part, row.Inputs)
			}
			preimage := valueAt(proof, pointerOf(t, row.Inputs[0]))
			if want := modelHash(t, kind, preimage); row.Recomputed != want {
				t.Errorf("%s %s: recomputed %v, the model %s", name, row.Part, row.Recomputed, want)
			}
			seen[kind] = true
		}
	}
	if len(seen) != 7 {
		t.Fatalf("the model checked %d kinds: %v", len(seen), seen)
	}
}

// pointerOf reads an RFC 6901 pointer that holds no escaped name and no
// index.
func pointerOf(t *testing.T, text string) Pointer {
	t.Helper()
	if !strings.HasPrefix(text, "/") || strings.Contains(text, "~") {
		t.Fatalf("pointer %q", text)
	}
	var p Pointer
	for _, name := range strings.Split(text[1:], "/") {
		p = append(p, Name(name))
	}
	return p
}

// Rule 3: type selects the whole composition, never the members P holds.
// An item, a beacon and an entropy proof relabeled block give no row
// under /subject, /witnesses or /observations, and an observation a
// beacon holds under a name the schema does not list gives none either.
// Mutants killed: the subject's checks selected by the presence of
// /subject (the relabeled item and entropy proof give rows there); a
// beacon's observations read from its members (the unlisted name gives
// rows).
func TestDerive_Rule3_TypeSelectsTheChecks(t *testing.T) {
	proofs := positiveProofs(t)
	for _, name := range []string{"item", "beacon", "entropy_nist"} {
		run := derived(put(proofs[name], Pointer{Name("type")}, "block"))
		for _, row := range deriveRows(run) {
			if part := *row.Part; strings.HasPrefix(part, "/subject") || strings.HasPrefix(part, "/witnesses") || strings.HasPrefix(part, "/observations") {
				t.Errorf("%s relabeled block: a row at %s", name, part)
			}
		}
	}
	beacon := proofs["beacon"]
	observation := valueAt(beacon, Pointer{Name("observations"), Name("entropy_nist")})
	run := derived(put(beacon, Pointer{Name("observations"), Name("entropy_other")}, observation))
	for _, row := range deriveRows(run) {
		if strings.Contains(*row.Part, "entropy_other") {
			t.Errorf("a row at %s", *row.Part)
		}
	}
	if !reflect.DeepEqual(run.Rows, derived(beacon).Rows) {
		t.Errorf("the unlisted observation changed the rows")
	}
}

// Rule 4: a declared value is only compared. Each value replaced fails at
// its own part alone: the walks that take it as their leaf, and the epoch
// walks from a block's hash, start from its recomputation instead. Mutants
// killed: the epoch walks starting from the block_hash P carries (the
// replaced /block/block_hash and /signing_key_event/block/block_hash also
// fail their epoch roots); the subject's walk starting from the item_hash
// or observation_hash P carries (the replaced value also fails
// /block/record/merkle_root).
func TestDerive_Rule4_DeclaredValuesAreOnlyCompared(t *testing.T) {
	proofs := positiveProofs(t)
	other := strings.Repeat("ab", 32)
	for _, c := range []struct {
		name string
		at   Pointer
	}{
		{"item", Pointer{Name("block"), Name("block_hash")}},
		{"item", Pointer{Name("subject"), Name("item_hash")}},
		{"item", Pointer{Name("witnesses"), Name("block"), Name("block_hash")}},
		{"item", Pointer{Name("commitments"), Name("bitcoin"), Name("epoch"), Name("epoch_hash")}},
		{"item", Pointer{Name("signing_key_event"), Name("block"), Name("block_hash")}},
		{"beacon", Pointer{Name("observations"), Name("entropy_nist"), Name("observation_hash")}},
		{"entropy_stellar", Pointer{Name("subject"), Name("observation_hash")}},
	} {
		run := derived(put(proofs[c.name], c.at, other))
		if got := failing(run); !slices.Equal(got, []string{"derivation_mismatch " + c.at.Encode()}) {
			t.Errorf("%s %s: failing %q", c.name, c.at.Encode(), got)
		}
	}
}

// Rule 5: PINNED values are compared with their pins before anything uses
// them, so a run that ended before DERIVE, at SHAPE, is passed through
// unchanged, whatever its P holds. Mutant killed: Derive without its
// ended-run guard (the ended run gains rows).
func TestDerive_Rule5_AnEndedRunIsPassedThrough(t *testing.T) {
	run := deriveRun(put(positiveProofs(t)["item"], Pointer{Name("format")}, "another-format"), deriveAttribution)
	run.Rows = run.Rows[:len(run.Rows)-1]
	run.AddRow(NewRow("SHAPE", specdata.ReasonUnsupportedFormat, Part("/format"), "no", nil))
	run.EndRun()
	before := *run
	before.Rows = slices.Clone(run.Rows)
	Derive(run)
	if !reflect.DeepEqual(*run, before) || run.Trace != nil {
		t.Fatalf("an ended run changed: %s", rowSummary(run.Rows))
	}
}

// Rule 6: no value the proof carries is ever an edge. DERIVE draws none: a
// run past it differs from the run before only by its DERIVE rows and its
// trace, even when a check fails, and it does not end. Mutants killed:
// Derive ending the run at a failing row (the run ends, with skip rows);
// Derive giving the run its type from P (a run that held none takes one).
func TestDerive_Rule6_DeriveSetsOnlyItsRowsAndTrace(t *testing.T) {
	proof := put(positiveProofs(t)["item"], Pointer{Name("commitments"), Name("stellar"), Name("epoch"), Name("epoch_hash")}, strings.Repeat("cd", 32))
	run := deriveRun(proof, deriveAttribution)
	run.Type = ""
	before := *run
	before.Rows = slices.Clone(run.Rows)
	Derive(run)
	if len(failing(run)) != 1 {
		t.Fatalf("failing %q", failing(run))
	}
	after := *run
	after.Rows, after.Trace = after.Rows[:len(before.Rows)], nil
	if !reflect.DeepEqual(after, before) || run.Ended {
		t.Fatalf("DERIVE changed more than its rows and trace: ended %v, %+v", run.Ended, after)
	}
	for _, row := range run.Rows[len(before.Rows):] {
		if row.Step != deriveStep {
			t.Fatalf("DERIVE added a %s row", row.Step)
		}
	}
}

// Rule 7: a key-history row passes only when the attribution is truestamp.
// None of these checks is one: their rows and trace are the same under
// every attribution, and none is at LINK_KEY_EVENT's part. Mutant killed:
// Derive leaving out the trace when the attribution is not truestamp.
func TestDerive_Rule7_NoRowOfTheseReadsTheAttribution(t *testing.T) {
	proof := positiveProofs(t)["item"]
	var runs []*Run
	for _, attribution := range []string{AttributionTruestamp, deriveAttribution, AttributionUnattributed} {
		run := deriveRun(proof, attribution)
		Derive(run)
		runs = append(runs, run)
	}
	for _, run := range runs[1:] {
		if !reflect.DeepEqual(run.Rows, runs[0].Rows) || !reflect.DeepEqual(run.Trace, runs[0].Trace) {
			t.Fatalf("under %s: %s, under truestamp %s", run.Attribution, rowSummary(deriveRows(run)), rowSummary(deriveRows(runs[0])))
		}
	}
	for _, row := range deriveRows(runs[0]) {
		if strings.HasSuffix(*row.Part, "/key_event/public_key") {
			t.Fatalf("a row at %s", *row.Part)
		}
	}
}

// The engine orders rows and trace by part in document order, rows
// sharing a part in the order of their checks, gives a trace row only for
// a traced check, and does not reorder the checks it is given; a failing
// row holds its construction, the declared and the recomputed value, and
// a pass row its construction alone. Mutants killed: the sort not stable
// (the two roots at /block/record/merkle_root swap); a trace row for an
// untraced check; the engine sorting its argument in place (the given
// checks are reordered); a failing row without recomputed.
func TestDerive_TheEngine(t *testing.T) {
	merkleRoot := Pointer{Name("block"), Name("record"), Name("merkle_root")}
	checks := []deriveCheck{
		equalCheck(Pointer{Name("subject"), Name("item_hash")}, "item_hash", []Pointer{{Name("subject"), Name("record")}}, "x", "x", "a"),
		equalCheck(merkleRoot, "inclusion", nil, "a", "b", "a"),
		equalCheck(Pointer{Name("block"), Name("block_hash")}, "block_hash", nil, "c", "c", "a"),
		equalCheck(merkleRoot, "inclusion", []Pointer{{Name("subject"), Name("record")}, {Name("subject"), Name("inclusion_proof")}}, "d", "d", "a"),
	}
	untraced := equalCheck(Pointer{Name("block"), Name("block_hash")}, "height", nil, "e", "f", "a")
	untraced.traced = false
	given := append(slices.Clone(checks), untraced)
	rows, trace := deriveEngine(given)
	if got := rowSummary(rows); got != "[null /block/block_hash pass DERIVE] [derivation_mismatch /block/block_hash fail DERIVE] "+
		"[derivation_mismatch /block/record/merkle_root fail DERIVE] [null /block/record/merkle_root pass DERIVE] [null /subject/item_hash pass DERIVE]" {
		t.Fatalf("rows %s", got)
	}
	var parts []string
	for _, row := range trace {
		parts = append(parts, fmt.Sprint(row.Part, " ", row.Declared, " ", row.Status))
	}
	if want := []string{"/block/block_hash c pass", "/block/record/merkle_root a fail", "/block/record/merkle_root d pass", "/subject/item_hash x pass"}; !slices.Equal(parts, want) {
		t.Fatalf("trace %q, want %q", parts, want)
	}
	if !reflect.DeepEqual(trace[2].Inputs, []string{"/subject/inclusion_proof", "/subject/record"}) {
		t.Fatalf("inputs %q, want them in document order", trace[2].Inputs)
	}
	if given[0].part.Encode() != "/subject/item_hash" {
		t.Fatalf("the engine reordered the checks it was given")
	}
	if want := map[string]any{"construction": "inclusion", "declared": "a", "recomputed": "b"}; !reflect.DeepEqual(rows[2].Values, want) {
		t.Fatalf("a failing row's values %v", rows[2].Values)
	}
	if want := map[string]any{"construction": "block_hash"}; !reflect.DeepEqual(rows[0].Values, want) || rows[0].Code != "" {
		t.Fatalf("a pass row's values %v", rows[0].Values)
	}
}

// A comparison holds only between two equal strings, two equal numbers or
// two nulls, as the reference's === holds; a check may name only the 14
// rows of SPEC.md sections 10.2, step 9, and 10.3, each a row of every
// accepted constructions text. Mutants killed: sameValue holding for the
// number 0 and the string "0" (fmt.Sprint equality); equalCheck accepting
// any construction (merkle_path, a row of the text DERIVE never names,
// passes); a name added to deriveConstructions that is not a row of the
// text.
func TestDerive_ComparisonsAndConstructionNames(t *testing.T) {
	for _, c := range []struct {
		a, b any
		same bool
	}{
		{"x", "x", true}, {strictjson.Number(0), strictjson.Number(0), true}, {nil, nil, true},
		{strictjson.Number(0), "0", false}, {"0", strictjson.Number(0), false}, {nil, "", false}, {"", nil, false},
		{strictjson.Number(3), strictjson.Number(4), false}, {true, true, false}, {map[string]any{}, map[string]any{}, false},
	} {
		if got := sameValue(c.a, c.b); got != c.same {
			t.Errorf("sameValue(%#v, %#v) = %v", c.a, c.b, got)
		}
	}
	if !panics(func() { equalCheck(Pointer{Name("block")}, "merkle_path", nil, "a", "a", "a") }) {
		t.Error("equalCheck named merkle_path")
	}
	paths, err := filepath.Glob(filepath.Join(vectorcases.Dir(), "..", "spec", "constructions", "*.json"))
	if err != nil || len(paths) != len(specdata.Accepted()) {
		t.Fatalf("constructions texts %q, %v", paths, err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var rows map[string]any
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatal(err)
		}
		for _, name := range deriveConstructions {
			if _, ok := rows[name]; !ok {
				t.Errorf("%s has no row %s", filepath.Base(path), name)
			}
		}
		if len(rows) != 28 {
			t.Errorf("%s has %d rows, SPEC.md section 7.3's text 28", filepath.Base(path), len(rows))
		}
	}
	if len(deriveConstructions) != 14 || !slices.IsSorted(deriveConstructions) {
		t.Fatalf("deriveConstructions %q", deriveConstructions)
	}
}

// A preimage the registry refuses, which SHAPE makes impossible, is a fault
// of the verifier: one DERIVE row fail internal_error, part null, then one
// skip prior_failure row of each later step up to EXPECTED_HASH, no trace,
// and the run ends (SPEC.md section 10.5). The claims and the epoch record
// are hashed only by their own checks, a block record by its check and by
// the epoch walks, and a genesis object by its genesis check. Mutants
// killed: the refusal dropped (a check of an empty preimage is made, or a
// row passes); the run not ended (no skip rows).
func TestDerive_ARefusedPreimageIsAFault(t *testing.T) {
	proofs := positiveProofs(t)
	for _, c := range []struct {
		name string
		at   Pointer
	}{
		{"item", Pointer{Name("subject"), Name("claims")}},
		{"item", Pointer{Name("commitments"), Name("stellar"), Name("epoch"), Name("record")}},
		{"item", Pointer{Name("signing_key_event"), Name("block"), Name("record")}},
		{"item", Pointer{Name("subject"), Name("record")}},
		{"item-genesis-witness", Pointer{Name("witnesses"), Name("block"), Name("record"), Name("metadata"), Name("genesis")}},
		{"beacon", Pointer{Name("observations"), Name("entropy_nist"), Name("entropy")}},
	} {
		run := derived(put(proofs[c.name], c.at, "not an object"))
		want := "[internal_error null fail DERIVE] [prior_failure null skip CHAIN] [prior_failure null skip SOURCES] " +
			"[prior_failure null skip WINDOW] [prior_failure null skip EXPECTED_HASH]"
		if got := rowSummary(run.Rows[8:]); got != want || !run.Ended || run.Trace != nil {
			t.Errorf("%s %s: %s, ended %v, trace %v", c.name, c.at.Encode(), got, run.Ended, run.Trace)
		}
	}
}
