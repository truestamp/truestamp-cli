// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// An independent model of DERIVE's built checks (SPEC.md section 10.2,
// step 9 (a) to (c), and LINK_INCLUSION of section 10.3), and the value
// classes of spec/value-classes.json (section 3.7), for the runner. The
// model shares no code with internal/verifier or internal/registry: it
// reads each prefix byte from spec/prefixes.json, hashes with
// crypto/sha256 over internal/jcs's encoding, finds each position from
// SPEC.md's tables and the proof's own members, and walks each inclusion
// proof by the steps of section 5.3 as written, so a defect in the
// verifier's layout, registry or walk cannot hide in the values it is
// compared with. It ports the reference's test support
// elixir/test/support/value_classes.ex and the application's
// test/support/proof_derive_model.ex and proof_value_classes.ex.

package vectorcases_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// builtConstructions are the rows of the constructions text the built
// checks name, sorted.
var builtConstructions = []string{"block_hash", "claims_hash", "entropy_hash", "epoch_hash", "genesis", "inclusion", "item_hash", "observation_hash"}

// linkHeadPart is LINK_HEAD's part, which the genesis check of a genesis
// /witnesses/block shares.
const linkHeadPart = "/witnesses/block/record/height"

// builtParts are the parts of the built checks' rows.
var builtParts = func() []*regexp.Regexp {
	blocks := `(?:/block|/witnesses/block|/signing_key_event/block)`
	var out []*regexp.Regexp
	for _, pattern := range []string{
		`^/subject/(?:record/claims_hash|item_hash|record/entropy_hash|observation_hash)$`,
		`^/observations/entropy_[a-z]+/(?:record/entropy_hash|observation_hash)$`,
		`^` + blocks + `/block_hash$`,
		`^` + blocks + `/record/(?:height|previous_block_hash)$`,
		`/epoch/epoch_hash$`,
		`/record/merkle_root$`,
		`/inclusion_proof/tree_size$`,
	} {
		out = append(out, regexp.MustCompile(pattern))
	}
	return out
}()

// builtPart reports whether part is the part of a built check's row.
func builtPart(part string) bool {
	return slices.ContainsFunc(builtParts, func(r *regexp.Regexp) bool { return r.MatchString(part) })
}

// modelCheck is one check of the model: its part and constructions row,
// the pointers it reads, sorted (lessPointer), the value P carries at its
// part, and what it compares that with.
type modelCheck struct {
	part, construction string
	inputs             []string
	declared           any
	recomputed         any
}

// status is the model's status of the check.
func (m modelCheck) status() string {
	if reflect.DeepEqual(m.declared, m.recomputed) {
		return "pass"
	}
	return "fail"
}

// prefixBytes are the bytes of spec/prefixes.json, by row name.
var prefixBytes = sync.OnceValues(func() (map[string]byte, error) {
	data, err := os.ReadFile(filepath.Join(vectorcases.Dir(), "..", "spec", "prefixes.json"))
	if err != nil {
		return nil, err
	}
	var rows map[string]struct {
		Byte string `json:"byte"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	out := map[string]byte{}
	for name, row := range rows {
		b, err := strconv.ParseUint(strings.TrimPrefix(row.Byte, "0x"), 16, 8)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out[name] = byte(b)
	}
	return out, nil
})

// modelHash is H(p, R) (SPEC.md section 4.1) with p the byte of the row
// name of spec/prefixes.json, as HEX64.
func modelHash(t *testing.T, name string, record any) string {
	t.Helper()
	prefixes, err := prefixBytes()
	if err != nil {
		t.Fatal(err)
	}
	b, ok := prefixes[name]
	if !ok {
		t.Fatalf("spec/prefixes.json has no row %s", name)
	}
	encoded, err := jcs.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(append([]byte{b}, encoded...))
	return hex.EncodeToString(sum[:])
}

// modelWalk is merkle_path(start, proof) of SPEC.md section 5.3, step by
// step, from the HEX64 start along a MERKLE_PROOF as P carries it, with at
// most maxSteps path nodes: the root as HEX64, or nil when it fails.
func modelWalk(start string, proof any, maxSteps int) any {
	object, _ := proof.(map[string]any)
	index, okIndex := object["leaf_index"].(strictjson.Number)
	size, okSize := object["tree_size"].(strictjson.Number)
	path, okPath := object["path"].([]any)
	leaf, err := hex.DecodeString(start)
	if !okIndex || !okSize || !okPath || err != nil || index >= size || len(path) > maxSteps {
		return nil
	}
	fn, sn := uint64(index), uint64(size)-1
	r := sha256.Sum256(append([]byte{0x00}, leaf...))
	for _, node := range path {
		text, _ := node.(string)
		p, err := hex.DecodeString(text)
		if err != nil || len(p) != 32 || sn == 0 {
			return nil
		}
		if fn%2 == 1 || fn == sn {
			r = sha256.Sum256(append(append([]byte{0x01}, p...), r[:]...))
			for fn%2 == 0 && fn != 0 {
				fn, sn = fn>>1, sn>>1
			}
		} else {
			r = sha256.Sum256(append(append([]byte{0x01}, r[:]...), p...))
		}
		fn, sn = fn>>1, sn>>1
	}
	if sn != 0 {
		return nil
	}
	return hex.EncodeToString(r[:])
}

// get is the value at the RFC 6901 pointer part in value, read here
// without the verifier, or nil; part holds no escaped name.
func get(value any, part string) any {
	if part == "" {
		return value
	}
	for _, segment := range strings.Split(strings.TrimPrefix(part, "/"), "/") {
		switch v := value.(type) {
		case map[string]any:
			value = v[segment]
		case []any:
			i, err := strconv.Atoi(segment)
			if err != nil || i < 0 || i >= len(v) {
				return nil
			}
			value = v[i]
		default:
			return nil
		}
	}
	return value
}

// lessPointer orders two pointers that hold no escaped name and no index
// in document order (SPEC.md section 10.2): a prefix first, and otherwise
// the first differing segment, by its bytes, which for these ASCII names
// is JCS order.
func lessPointer(a, b string) bool {
	x, y := strings.Split(a, "/"), strings.Split(b, "/")
	for i := range min(len(x), len(y)) {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return len(x) < len(y)
}

// sortedPointers returns pointers sorted by lessPointer.
func sortedPointers(pointers ...string) []string {
	out := slices.Clone(pointers)
	sort.SliceStable(out, func(i, j int) bool { return lessPointer(out[i], out[j]) })
	return out
}

// model returns the model's checks of P: (a) the subject hashes, (b) each
// block node's block_hash and, with metadata.genesis, its height and
// previous_block_hash, (c) each commitment entry's epoch_hash, and
// LINK_INCLUSION's root and size of each inclusion proof. The runner
// compares them by part and construction; of the checks at one part with
// one construction, such as a beacon's roots at /block/record/merkle_root,
// the model gives them in the document order of their proofs, by the
// names P holds, sorted.
func model(t *testing.T, proof map[string]any) []modelCheck {
	t.Helper()
	var out []modelCheck
	hash := func(part, name, preimage string) {
		out = append(out, modelCheck{part, name, []string{preimage}, get(proof, part), modelHash(t, name, get(proof, preimage))})
	}
	inclusion := func(q, leafName, leafAt, r string) {
		leaf := modelHash(t, leafName, get(proof, leafAt))
		out = append(out,
			modelCheck{r + "/merkle_root", "inclusion", sortedPointers(q, leafAt), get(proof, r+"/merkle_root"), modelWalk(leaf, get(proof, q), 32)},
			modelCheck{q + "/tree_size", "inclusion", []string{r + "/leaf_count"}, get(proof, q+"/tree_size"), get(proof, r+"/leaf_count")})
	}
	var holders []string
	switch typ, _ := proof["type"].(string); {
	case typ == "item":
		hash("/subject/record/claims_hash", "claims_hash", "/subject/claims")
		hash("/subject/item_hash", "item_hash", "/subject/record")
		inclusion("/subject/inclusion_proof", "item_hash", "/subject/record", "/block/record")
	case typ == "beacon":
		observations, _ := proof["observations"].(map[string]any)
		for _, name := range slices.Sorted(maps.Keys(observations)) {
			holders = append(holders, "/observations/"+name)
		}
	case strings.HasPrefix(typ, "entropy_"):
		holders = append(holders, "/subject")
	}
	for _, at := range holders {
		hash(at+"/record/entropy_hash", "entropy_hash", at+"/entropy")
		hash(at+"/observation_hash", "observation_hash", at+"/record")
		inclusion(at+"/inclusion_proof", "observation_hash", at+"/record", "/block/record")
	}
	blocks := []string{"/block", "/signing_key_event/block"}
	if proof["type"] == "item" {
		blocks = append(blocks, "/witnesses/block")
	}
	for _, b := range blocks {
		if get(proof, b) == nil {
			continue
		}
		hash(b+"/block_hash", "block_hash", b+"/record")
		if genesis := get(proof, b+"/record/metadata/genesis"); genesis != nil {
			out = append(out,
				modelCheck{b + "/record/height", "genesis", []string{b + "/record/metadata/genesis"}, get(proof, b+"/record/height"), strictjson.Number(0)},
				modelCheck{b + "/record/previous_block_hash", "genesis", []string{b + "/record/metadata/genesis"},
					get(proof, b+"/record/previous_block_hash"), modelHash(t, "genesis_previous_block_hash", genesis)})
		}
	}
	for _, base := range []string{"", "/signing_key_event"} {
		commitments, _ := get(proof, base+"/commitments").(map[string]any)
		for _, chain := range slices.Sorted(maps.Keys(commitments)) {
			n := base + "/commitments/" + chain
			hash(n+"/epoch/epoch_hash", "epoch_hash", n+"/epoch/record")
			inclusion(n+"/inclusion_proof", "block_hash", base+"/block/record", n+"/epoch/record")
		}
	}
	return out
}

// valueClass is one member of spec/value-classes.json.
type valueClass struct {
	Class     string `json:"class"`
	Otherwise string `json:"otherwise"`
	Pointer   string `json:"pointer"`
	When      string `json:"when"`
	Document  string `json:"document"`
}

// valueClasses are the members of spec/value-classes.json.
var valueClasses = sync.OnceValues(func() ([]valueClass, error) {
	data, err := os.ReadFile(filepath.Join(vectorcases.Dir(), "..", "spec", "value-classes.json"))
	if err != nil {
		return nil, err
	}
	var file struct {
		Members []valueClass `json:"members"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return file.Members, nil
})

// classOf is the value class of the member of P at part, by
// spec/value-classes.json, or "" when it lists none: a member's pointer
// may hold * for any one segment and entropy_* for a segment that begins
// entropy_, and a member with a condition has its class when the condition
// holds in P, and its otherwise class when it does not.
func classOf(t *testing.T, proof map[string]any, part string) string {
	t.Helper()
	members, err := valueClasses()
	if err != nil {
		t.Fatal(err)
	}
	segments := strings.Split(part, "/")
	for _, m := range members {
		if m.Document != "" {
			continue
		}
		pattern := strings.Split(m.Pointer, "/")
		if len(pattern) != len(segments) {
			continue
		}
		matches := true
		for i, want := range pattern {
			if want != "*" && want != segments[i] && !(want == "entropy_*" && strings.HasPrefix(segments[i], "entropy_")) {
				matches = false
			}
		}
		if !matches {
			continue
		}
		if m.When == "" {
			return m.Class
		}
		if conditionHolds(t, proof, segments, m.When) {
			return m.Class
		}
		return m.Otherwise
	}
	return ""
}

// conditionHolds reads one of spec/value-classes.json's conditions in P,
// for the member at segments.
func conditionHolds(t *testing.T, proof map[string]any, segments []string, condition string) bool {
	t.Helper()
	record := func() any {
		for _, block := range []string{"/signing_key_event/block", "/witnesses/block", "/block"} {
			if strings.HasPrefix(strings.Join(segments, "/")+"/", block+"/") {
				return get(proof, block+"/record")
			}
		}
		return nil
	}
	switch condition {
	case "genesis_block":
		_, ok := get(record(), "/metadata/genesis").(map[string]any)
		return ok
	case "introduces_key_or_prerotation":
		typ, _ := get(record(), "/metadata/key_event/type").(string)
		return slices.Contains([]string{"genesis", "rotation", "emergency_rotation", "prerotation"}, typ)
	case "walked":
		switch proof["type"] {
		case "block":
			return false
		case "beacon":
			observations, _ := proof["observations"].(map[string]any)
			return len(observations) > 0
		}
		return true
	case "stellar_source":
		return get(proof, strings.Join(segments[:len(segments)-2], "/")+"/source") == "stellar"
	}
	t.Fatalf("spec/value-classes.json names the condition %s, which this model does not read", condition)
	return false
}

// checkDerive holds a run that reached DERIVE to the model: its DERIVE
// rows are exactly the model's checks, by part and construction, each
// with the model's status, a pass row's values its construction alone and
// a failing row's its construction, the value P carries at its part as
// declared and the model's recomputed value (SPEC.md section 10.5), and
// the code derivation_mismatch; its trace has one row for each DERIVE row,
// in the rows' order, at a CHECKED part (spec/value-classes.json), with
// the row's construction and status, the value P carries at its part, the
// model's recomputed value and inputs, each input a value P holds, in
// document order; and each trace row, as its Value writes it, has the JCS
// of the model's row {construction, declared, inputs, part, recomputed,
// status}, declared the value P carries at the part. Mutants killed by the
// whole comparison: Value writing the status pass for every row (a failing
// row writes pass); Value writing declared from Recomputed (a failing row
// declares the recomputed value).
func checkDerive(t *testing.T, c vectorcases.Case, run *verifier.Run) {
	t.Helper()
	proof := proofOf(t, c)
	groups := map[string][]modelCheck{}
	for _, m := range model(t, proof) {
		key := m.part + " " + m.construction
		groups[key] = append(groups[key], m)
	}
	var rows []verifier.Row
	for _, row := range run.Rows {
		if row.Step == "DERIVE" {
			rows = append(rows, row)
		}
	}
	if len(rows) != len(run.Trace) {
		t.Fatalf("%d DERIVE rows, %d trace rows", len(rows), len(run.Trace))
	}
	seen := map[string]int{}
	for i, row := range rows {
		construction, _ := row.Values["construction"].(string)
		key := *row.Part + " " + construction
		checks := groups[key]
		if seen[key] >= len(checks) {
			t.Fatalf("a DERIVE row %s that the model does not make", key)
		}
		m := checks[seen[key]]
		seen[key]++
		want := map[string]any{"construction": m.construction}
		code := ""
		if m.status() == "fail" {
			want["declared"], want["recomputed"], code = m.declared, m.recomputed, "derivation_mismatch"
		}
		if string(row.Status) != m.status() || string(row.Code) != code || !reflect.DeepEqual(row.Values, want) {
			t.Fatalf("%s: %s %s %v, the model %s %v", key, row.Status, row.Code, row.Values, m.status(), want)
		}
		trace := run.Trace[i]
		if trace.Part != *row.Part || trace.Construction != construction || string(trace.Status) != m.status() ||
			!reflect.DeepEqual(trace.Declared, get(proof, trace.Part)) || !reflect.DeepEqual(trace.Recomputed, m.recomputed) ||
			!slices.Equal(trace.Inputs, m.inputs) {
			t.Fatalf("trace row %d %+v, the row %s and the model %+v", i, trace, key, m)
		}
		for _, input := range trace.Inputs {
			if get(proof, input) == nil {
				t.Fatalf("%s: the input %s holds nothing", key, input)
			}
		}
		if class := classOf(t, proof, trace.Part); class != "CHECKED" {
			t.Fatalf("a trace row at %s, whose class is %q", trace.Part, class)
		}
		inputs := make([]any, len(m.inputs))
		for j, input := range m.inputs {
			inputs[j] = input
		}
		written, err := jcs.Encode(trace.Value())
		if err != nil {
			t.Fatalf("trace row %d: %v", i, err)
		}
		modelRow, err := jcs.Encode(map[string]any{
			"construction": m.construction, "declared": get(proof, m.part), "inputs": inputs,
			"part": m.part, "recomputed": m.recomputed, "status": m.status(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if string(written) != string(modelRow) {
			t.Fatalf("trace row %d writes %s, the model %s", i, written, modelRow)
		}
	}
	for key, checks := range groups {
		if seen[key] != len(checks) {
			t.Fatalf("the model's check %s gives %d rows, the run %d", key, len(checks), seen[key])
		}
	}
}

// traceFile is a sound proof's trace file, each row as its JCS.
func traceFile(t *testing.T, relative string) []string {
	t.Helper()
	data, err := os.ReadFile(vectorcases.Path(relative))
	if err != nil {
		t.Fatal(err)
	}
	value, err := strictjson.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, row := range value.([]any) {
		encoded, err := jcs.Encode(row)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, string(encoded))
	}
	return out
}
