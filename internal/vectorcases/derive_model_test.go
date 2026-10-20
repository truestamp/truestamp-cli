// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// An independent model of DERIVE (SPEC.md section 10.2, step 9 (a) to (e),
// with the key events of section 10.3 and the Bitcoin evidence of
// bitcoin_model_test.go), and the value classes of
// spec/value-classes.json (section 3.7), for the runner. The model shares
// no code with internal/verifier or internal/registry: it reads each prefix
// byte from spec/prefixes.json and each small-order key from
// spec/small-order-keys.json, hashes with crypto/sha256 over internal/jcs's
// encoding, verifies a handover with crypto/ed25519 after its own steps 2
// and 3 of section 6.2, finds each position from SPEC.md's tables and the
// proof's own members, and walks each inclusion proof by the steps of
// section 5.3 as written, so a defect in the verifier's layout, registry,
// walk or key checks cannot hide in the values it is compared with. It
// ports the reference's test support elixir/test/support/value_classes.ex
// and the application's test/support/proof_derive_model.ex and
// proof_value_classes.ex, and the failure values of the application's
// derive_key_events_test.exs.

package vectorcases_test

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
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

// tracedConstructions are the rows of the constructions text DERIVE's
// trace rows name, sorted: every row a sound trace holds.
var tracedConstructions = []string{"bitcoin", "block_hash", "claims_hash", "entropy_hash", "epoch_hash", "genesis", "handover", "inclusion",
	"item_hash", "key_event", "observation_hash", "witnesses"}

// modelCheck is one check of the model: its part and constructions row,
// the pointers it reads, sorted (lessPointer), the value P carries at its
// part, what it compares that with, whether it gives a trace row, its
// status and code, and, for a refused key or signature, the step of
// SPEC.md section 6.2 that refused it, with uncompared for the check of a
// key's bytes, whose failing row holds no recomputed value, and, for a
// Bitcoin structure that is not well formed, the first rule it breaks.
type modelCheck struct {
	part, construction string
	inputs             []string
	declared           any
	recomputed         any
	traced             bool
	status, code       string
	refused            string
	uncompared         bool
	rule               string
}

// equal is the traced check that declared equals recomputed, failing with
// code.
func equal(part, construction string, inputs []string, declared, recomputed any, code string) modelCheck {
	m := modelCheck{part: part, construction: construction, inputs: inputs, declared: declared, recomputed: recomputed, traced: true, status: "pass"}
	if !reflect.DeepEqual(declared, recomputed) {
		m.status, m.code = "fail", code
	}
	return m
}

// order is the untraced order link at part: a pass when holds, and
// otherwise ordering_violation.
func order(part, construction string, inputs []string, declared, bound any, holds bool) modelCheck {
	m := modelCheck{part: part, construction: construction, inputs: sortedPointers(inputs...), declared: declared, recomputed: bound, status: "pass"}
	if !holds {
		m.status, m.code = "fail", "ordering_violation"
	}
	return m
}

// values are the values SPEC.md section 10.5 gives the check's row: its
// construction, and, when it fails, the rule a structure breaks for a
// check of well-formedness, which compares nothing, and otherwise the
// value it compares as declared, the model's recomputed value unless the
// check compares none, and the refusal's step when it names one.
func (m modelCheck) values() map[string]any {
	out := map[string]any{"construction": m.construction}
	if m.status != "fail" {
		return out
	}
	if m.rule != "" {
		out["rule"] = m.rule
		return out
	}
	out["declared"] = m.declared
	if !m.uncompared {
		out["recomputed"] = m.recomputed
	}
	if m.refused != "" {
		out["refused"] = m.refused
	}
	return out
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

// model returns the model's checks of P, signed by the key whose B64U32
// is k, in a run whose attribution is attribution: (a) the subject
// hashes, (b) each block node's block_hash and, with metadata.genesis, its
// height and previous_block_hash, then the checks of the key event its
// record carries, (c) each commitment entry's epoch_hash, (d) each
// Bitcoin entry's evidence (modelBitcoin), and the links:
// LINK_INCLUSION's root and size of each inclusion proof, LINK_WITNESS,
// LINK_HEAD, LINK_KEY_EVENT and LINK_CHAIN_ORDER. The runner compares them
// by part and construction; of the checks at one part with one
// construction, such as a beacon's roots at /block/record/merkle_root or
// the check of a key's bytes and LINK_KEY_EVENT at the signing key event's
// public_key, the model gives them in the order of step 9's parts and, for
// the roots, in the document order of their proofs, by the names P holds,
// sorted.
func model(t *testing.T, proof map[string]any, k, attribution string) []modelCheck {
	t.Helper()
	var out []modelCheck
	hash := func(part, name, preimage string) {
		out = append(out, equal(part, name, []string{preimage}, get(proof, part), modelHash(t, name, get(proof, preimage)), "derivation_mismatch"))
	}
	inclusion := func(q, leafName, leafAt, r string) {
		leaf := modelHash(t, leafName, get(proof, leafAt))
		out = append(out,
			equal(r+"/merkle_root", "inclusion", sortedPointers(q, leafAt), get(proof, r+"/merkle_root"), modelWalk(leaf, get(proof, q), 32), "derivation_mismatch"),
			equal(q+"/tree_size", "inclusion", []string{r + "/leaf_count"}, get(proof, q+"/tree_size"), get(proof, r+"/leaf_count"), "derivation_mismatch"))
	}
	var holders []string
	var links []func()
	typ, _ := proof["type"].(string)
	switch {
	case typ == "item":
		hash("/subject/record/claims_hash", "claims_hash", "/subject/claims")
		hash("/subject/item_hash", "item_hash", "/subject/record")
		links = append(links, func() { inclusion("/subject/inclusion_proof", "item_hash", "/subject/record", "/block/record") })
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
		links = append(links, func() { inclusion(at+"/inclusion_proof", "observation_hash", at+"/record", "/block/record") })
	}
	blocks := []string{"/block", "/signing_key_event/block"}
	if typ == "item" {
		blocks = append(blocks, "/witnesses/block")
	}
	for _, b := range blocks {
		if get(proof, b) == nil {
			continue
		}
		hash(b+"/block_hash", "block_hash", b+"/record")
		if genesis := get(proof, b+"/record/metadata/genesis"); genesis != nil {
			out = append(out,
				equal(b+"/record/height", "genesis", []string{b + "/record/metadata/genesis"}, get(proof, b+"/record/height"), strictjson.Number(0), "derivation_mismatch"),
				equal(b+"/record/previous_block_hash", "genesis", []string{b + "/record/metadata/genesis"},
					get(proof, b+"/record/previous_block_hash"), modelHash(t, "genesis_previous_block_hash", genesis), "derivation_mismatch"))
		}
		out = append(out, modelKeyEvent(t, proof, b)...)
	}
	for _, base := range []string{"", "/signing_key_event"} {
		commitments, _ := get(proof, base+"/commitments").(map[string]any)
		for _, chain := range slices.Sorted(maps.Keys(commitments)) {
			n := base + "/commitments/" + chain
			hash(n+"/epoch/epoch_hash", "epoch_hash", n+"/epoch/record")
			links = append(links, func() { inclusion(n+"/inclusion_proof", "block_hash", base+"/block/record", n+"/epoch/record") })
		}
	}
	for _, base := range []string{"", "/signing_key_event"} {
		if get(proof, base+"/commitments/bitcoin") != nil {
			out = append(out, modelBitcoin(t, proof, base+"/commitments/bitcoin")...)
		}
	}
	for _, link := range links {
		link()
	}
	if typ == "item" {
		hashes, _ := get(proof, "/subject/record/witness_hashes").(map[string]any)
		for _, name := range slices.Sorted(maps.Keys(hashes)) {
			at, kind := "/witnesses/"+name, "entropy_hash"
			if name == "block" {
				at, kind = "/witnesses/block/record", "block_hash"
			}
			out = append(out, equal("/subject/record/witness_hashes/"+name, "witnesses", []string{at}, hashes[name], modelHash(t, kind, get(proof, at)), "derivation_mismatch"))
		}
		witness, block := modelHeight(get(proof, "/witnesses/block/record")), modelHeight(get(proof, "/block/record"))
		part := "/witnesses/block/record/height"
		out = append(out, order(part, "height", []string{part, "/block/record/height"}, get(proof, part), block, witness < block))
	}
	link := "/signing_key_event/block/record/metadata/key_event/public_key"
	if declared := get(proof, link); declared != nil {
		m := modelCheck{part: link, construction: "key_event", inputs: []string{link}, declared: declared, recomputed: k, status: "pass"}
		switch {
		case !slices.Equal(b64u(t, declared), b64u(t, k)):
			m.status, m.code = "fail", "key_event_mismatch"
		case attribution != "truestamp":
			m.status, m.code = "info", "not_attributable"
		}
		out = append(out, m)
	}
	for _, e := range modelEntropyObjects(proof) {
		top, _ := get(proof, "/commitments/"+e.source).(map[string]any)
		if top == nil || !reflect.DeepEqual(top["network"], get(proof, e.at+"/network")) {
			continue
		}
		switch e.source {
		case "stellar":
			part := e.at + "/payload/sequence"
			sequence, ledger := get(proof, part).(strictjson.Number), top["ledger"].(strictjson.Number)
			out = append(out, order(part, "chain_order", []string{part, "/commitments/stellar/ledger"}, sequence, ledger, sequence < ledger))
		case "bitcoin":
			part := e.at + "/payload/height"
			height, committed := get(proof, part).(strictjson.Number), top["height"].(strictjson.Number)
			out = append(out, order(part, "chain_order", []string{part, "/commitments/bitcoin/height"}, height, committed, height <= committed))
		}
	}
	return out
}

// modelHeight is the height a block record counts as (SPEC.md section
// 10.3): 0 when it carries metadata.genesis, and otherwise its height.
func modelHeight(record any) strictjson.Number {
	if get(record, "/metadata/genesis") != nil {
		return 0
	}
	return get(record, "/height").(strictjson.Number)
}

// modelEntropy is an entropy object of P at its pointer, with the source
// its position names.
type modelEntropy struct{ at, source string }

// modelEntropyObjects are the entropy objects P carries (SPEC.md section
// 10.2, step 9): /subject/entropy in an entropy type, each
// /witnesses/entropy_X in an item and each /observations/entropy_X/entropy
// in a beacon, by the names P holds.
func modelEntropyObjects(proof map[string]any) []modelEntropy {
	typ, _ := proof["type"].(string)
	var out []modelEntropy
	add := func(base, suffix string) {
		holder, _ := get(proof, base).(map[string]any)
		for _, name := range slices.Sorted(maps.Keys(holder)) {
			if source, ok := strings.CutPrefix(name, "entropy_"); ok {
				out = append(out, modelEntropy{base + "/" + name + suffix, source})
			}
		}
	}
	switch {
	case strings.HasPrefix(typ, "entropy_"):
		out = append(out, modelEntropy{"/subject/entropy", strings.TrimPrefix(typ, "entropy_")})
	case typ == "item":
		add("/witnesses", "")
	case typ == "beacon":
		add("/observations", "/entropy")
	}
	return out
}

// modelKeyEvent is the model's checks of the key event the block node at b
// carries, by the table of SPEC.md section 10.3: none for a revocation, or
// when the record carries no event.
func modelKeyEvent(t *testing.T, proof map[string]any, b string) []modelCheck {
	t.Helper()
	e := b + "/record/metadata/key_event"
	event, _ := get(proof, e).(map[string]any)
	keyID := func(member string) string { return modelKeyID(t, b64u(t, event[member])) }
	mismatch := func(part, construction string, inputs []string, declared, recomputed any) modelCheck {
		return equal(part, construction, inputs, declared, recomputed, "key_event_mismatch")
	}
	switch event["type"] {
	case "genesis", "rotation", "emergency_rotation":
		bytesCheck := modelCheck{part: e + "/public_key", construction: "key_event", inputs: []string{e + "/public_key"},
			declared: event["public_key"], uncompared: true, status: "pass"}
		if refused := modelKeyRefusal(t, b64u(t, event["public_key"])); refused != "" {
			bytesCheck.status, bytesCheck.code, bytesCheck.refused = "fail", "key_event_mismatch", refused
		}
		out := []modelCheck{
			bytesCheck,
			mismatch(e+"/key_id", "key_event", []string{e + "/public_key"}, event["key_id"], keyID("public_key")),
			mismatch(b+"/record/signing_key_id", "key_event", []string{e + "/public_key"}, get(proof, b+"/record/signing_key_id"), keyID("public_key")),
		}
		if event["type"] == "genesis" {
			return out
		}
		out = append(out,
			mismatch(e+"/previous_key_id", "handover", []string{e + "/previous_public_key"}, event["previous_key_id"], keyID("previous_public_key")),
			mismatch(e+"/height", "handover", []string{b + "/record/height"}, event["height"], get(proof, b+"/record/height")),
			mismatch(e+"/previous_block_hash", "handover", []string{b + "/record/previous_block_hash"}, event["previous_block_hash"],
				get(proof, b+"/record/previous_block_hash")))
		signature, signed := event["handover_signature"]
		if !signed {
			return out
		}
		unsigned := maps.Clone(event)
		delete(unsigned, "handover_signature")
		message := modelHash(t, "handover", unsigned)
		m := modelCheck{part: e + "/handover_signature", construction: "handover", inputs: sortedPointers(e, e+"/previous_public_key"),
			declared: signature, recomputed: message, traced: true, status: "pass"}
		digest, _ := hex.DecodeString(message)
		if refused := modelVerify(t, b64u(t, event["previous_public_key"]), digest, b64u(t, signature)); refused != "" {
			m.status, m.code, m.refused = "fail", "key_event_mismatch", refused
		}
		return append(out, m)
	case "prerotation":
		return []modelCheck{mismatch(b+"/record/signing_key_id", "key_event", []string{e + "/active_key_id"},
			get(proof, b+"/record/signing_key_id"), event["active_key_id"])}
	}
	return nil
}

// b64u decodes a base64url text P carries, with no padding.
func b64u(t *testing.T, text any) []byte {
	t.Helper()
	s, _ := text.(string)
	out, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("%v is not base64url: %v", text, err)
	}
	return out
}

// modelKeyID is the key id of SPEC.md section 3.3: the hex of the first 4
// bytes of SHA-256 over the byte spec/prefixes.json gives key_id and the
// key's 32 bytes.
func modelKeyID(t *testing.T, key []byte) string {
	t.Helper()
	prefixes, err := prefixBytes()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(append([]byte{prefixes["key_id"]}, key...))
	return hex.EncodeToString(sum[:4])
}

// fieldPrime is p = 2^255 - 19, and groupOrder L, of RFC 8032 section 5.1.
var (
	fieldPrime = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))
	groupOrder = func() *big.Int {
		l, _ := new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
		return l
	}()
)

// littleEndian reads octets as an unsigned little-endian integer.
func littleEndian(octets []byte) *big.Int {
	bigEndian := slices.Clone(octets)
	slices.Reverse(bigEndian)
	return new(big.Int).SetBytes(bigEndian)
}

// modelKeyRefusal is the step of SPEC.md section 6.2 that refuses the
// 32 octets key as a key's bytes: y_not_below_p (step 2) when y, key with
// its top bit cleared, is not below p, small_order_key (step 3) when key is
// one of spec/small-order-keys.json, and "" otherwise.
func modelKeyRefusal(t *testing.T, key []byte) string {
	t.Helper()
	y := littleEndian(key)
	y.SetBit(y, 255, 0)
	if y.Cmp(fieldPrime) >= 0 {
		return "y_not_below_p"
	}
	data, err := os.ReadFile(filepath.Join(vectorcases.Dir(), "..", "spec", "small-order-keys.json"))
	if err != nil {
		t.Fatal(err)
	}
	var keys []struct {
		Hex string `json:"hex"`
	}
	if err := json.Unmarshal(data, &keys); err != nil || len(keys) != 10 {
		t.Fatalf("spec/small-order-keys.json: %d keys, %v", len(keys), err)
	}
	for _, small := range keys {
		if small.Hex == hex.EncodeToString(key) {
			return "small_order_key"
		}
	}
	return ""
}

// modelVerify is Verify of SPEC.md section 6.2 over message under key:
// "" when signature verifies, and otherwise the first step that refuses:
// the key's steps 2 and 3, not_on_curve when no point has the encoding key
// (step 4, RFC 8032 section 5.1.3), s_not_below_l when the last 32 octets
// are not below L (step 5), and equation (step 7).
func modelVerify(t *testing.T, key, message, signature []byte) string {
	t.Helper()
	if refused := modelKeyRefusal(t, key); refused != "" {
		return refused
	}
	if ed25519.Verify(key, message, signature) {
		return ""
	}
	y := littleEndian(key)
	sign := y.Bit(255)
	y.SetBit(y, 255, 0)
	// x^2 = (y^2 - 1) / (d y^2 + 1), d = -121665 / 121666.
	d := new(big.Int).Mul(big.NewInt(-121665), new(big.Int).ModInverse(big.NewInt(121666), fieldPrime))
	y2 := new(big.Int).Mul(y, y)
	u := new(big.Int).Sub(y2, big.NewInt(1))
	v := new(big.Int).Add(new(big.Int).Mul(d, y2), big.NewInt(1))
	x2 := new(big.Int).Mul(u, new(big.Int).ModInverse(v.Mod(v, fieldPrime), fieldPrime))
	x2.Mod(x2, fieldPrime)
	x := new(big.Int).ModSqrt(x2, fieldPrime)
	switch {
	case x == nil, x.Sign() == 0 && sign == 1:
		return "not_on_curve"
	case littleEndian(signature[32:]).Cmp(groupOrder) >= 0:
		return "s_not_below_l"
	}
	return "equation"
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

// checkDerive holds a run that reached DERIVE to the model, given the K of
// the case's document and the run's attribution: its DERIVE rows are
// exactly the model's checks, by part and construction, each with the
// model's status and code, and the values SPEC.md section 10.5 gives it
// (modelCheck.values): a pass or info row its construction alone, a
// failing row its construction, the value P carries at its part as
// declared, the model's recomputed value unless the check compares none,
// and the step of section 6.2 that refused a key or a signature; its trace
// has one row for each traced DERIVE row, in the rows' order, at a CHECKED
// part (spec/value-classes.json), with the row's construction and status,
// the value P carries at its part, the model's recomputed value and
// inputs, each input a value P holds, in document order, and no untraced
// row is at a CHECKED part but one LINK_HEAD shares with the genesis
// check; and each trace row, as its Value writes it, has the JCS of the
// model's row {construction, declared, inputs, part, recomputed, status},
// declared the value P carries at the part. Mutants killed by the whole
// comparison: Value writing the status pass for every row (a failing row
// writes pass); Value writing declared from Recomputed (a failing row
// declares the recomputed value); a refusal named for the wrong step (the
// small-order outgoing keys of key_events-identity_previous_public_key-*);
// the check of a key's bytes giving a recomputed value, or traced.
func checkDerive(t *testing.T, c vectorcases.Case, run *verifier.Run) {
	t.Helper()
	proof := proofOf(t, c)
	groups := map[string][]modelCheck{}
	for _, m := range model(t, proof, keyOf(t, c), run.Attribution) {
		key := m.part + " " + m.construction
		groups[key] = append(groups[key], m)
	}
	seen := map[string]int{}
	traced := 0
	for _, row := range run.Rows {
		if row.Step != "DERIVE" {
			continue
		}
		construction, _ := row.Values["construction"].(string)
		key := *row.Part + " " + construction
		checks := groups[key]
		if seen[key] >= len(checks) {
			t.Fatalf("a DERIVE row %s that the model does not make", key)
		}
		m := checks[seen[key]]
		seen[key]++
		if string(row.Status) != m.status || string(row.Code) != m.code || !reflect.DeepEqual(row.Values, m.values()) {
			t.Fatalf("%s: %s %s %v, the model %s %s %v", key, row.Status, row.Code, row.Values, m.status, m.code, m.values())
		}
		if !m.traced {
			if classOf(t, proof, m.part) == "CHECKED" && m.construction != "height" {
				t.Fatalf("an untraced row at %s, whose class is CHECKED", m.part)
			}
			continue
		}
		if traced >= len(run.Trace) {
			t.Fatalf("no trace row for %s", key)
		}
		trace := run.Trace[traced]
		traced++
		if trace.Part != *row.Part || trace.Construction != construction || string(trace.Status) != m.status ||
			!reflect.DeepEqual(trace.Declared, get(proof, trace.Part)) || !reflect.DeepEqual(trace.Recomputed, m.recomputed) ||
			!slices.Equal(trace.Inputs, m.inputs) {
			t.Fatalf("trace row %d %+v, the row %s and the model %+v", traced-1, trace, key, m)
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
			t.Fatalf("trace row %d: %v", traced-1, err)
		}
		modelRow, err := jcs.Encode(map[string]any{
			"construction": m.construction, "declared": get(proof, m.part), "inputs": inputs,
			"part": m.part, "recomputed": m.recomputed, "status": m.status,
		})
		if err != nil {
			t.Fatal(err)
		}
		if string(written) != string(modelRow) {
			t.Fatalf("trace row %d writes %s, the model %s", traced-1, written, modelRow)
		}
	}
	if traced != len(run.Trace) {
		t.Fatalf("%d traced rows, %d trace rows", traced, len(run.Trace))
	}
	for key, checks := range groups {
		if seen[key] != len(checks) {
			t.Fatalf("the model's check %s gives %d rows, the run %d", key, len(checks), seen[key])
		}
	}
}

// keyOf is K of a case's document, the B64U32 its signature carries.
func keyOf(t *testing.T, c vectorcases.Case) string {
	t.Helper()
	data, err := strictjson.ReadFile(c.ProofPath())
	if err != nil {
		t.Fatal(err)
	}
	document, err := strictjson.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return get(document, "/signature/public_key").(string)
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
