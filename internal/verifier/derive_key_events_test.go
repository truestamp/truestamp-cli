// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"bytes"
	goed25519 "crypto/ed25519"
	"encoding/hex"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// The key events of DERIVE and its links after LINK_INCLUSION (SPEC.md
// section 10.2, step 9 (b) and (e), and section 10.3), on the positive
// proofs and mutants of them, beside the key_events, handover_position,
// order and self_signed cases of internal/vectorcases' runner. It ports
// the application's derive_key_events_test.exs (the reference's
// derive_test.exs reaches these checks through the vectors alone):
//
//   - proofs built here, which no case takes through DERIVE: LINK_HEAD at a
//     genesis /block, an emergency rotation with and without a handover, a
//     revocation, a genesis key and an outgoing key whose y is p, an
//     incoming rotation key whose y is p or of small order, a record
//     whose signing_key_id is not the key id of its event's key, and the
//     handover signature's other refusals;
//   - each link failing at its own part (SPEC.md section 10.2's one rule);
//   - the rules of SPEC.md section 10.6 these checks reach (rule 2 and
//     rule 7 for LINK_KEY_EVENT are derive_test.go's);
//   - the order of the checks that share a part;
//   - the members the handover signs, against spec/schema.json.

// keyEventAt is the pointer of the key event block carries.
func keyEventAt(block Pointer) Pointer { return under(block, "record", "metadata", "key_event") }

// rehashed returns proof with the block_hash of the block node at block
// recomputed here from its record, so that only the checks a mutant of the
// record targets fail at the block, and the epoch walks from it.
func rehashed(t *testing.T, proof map[string]any, block Pointer) map[string]any {
	t.Helper()
	return put(proof, under(block, "block_hash"), modelHash(t, "block_hash", valueAt(proof, under(block, "record"))))
}

// pastPolicy runs proof, signed again by shapeSigner, keyless under E_v
// test with the networks of the sound cases pinned, through the gate, SHAPE
// and POLICY, and fails the test unless every row passes, so that a run
// reaches DERIVE with it.
func pastPolicy(t *testing.T, proof map[string]any) {
	t.Helper()
	cfg, err := NewConfig("test")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Keyring = KeyringSource{Kind: SourceKeyless}
	chains := specdata.ChainNetworks{Bitcoin: "regtest", Stellar: "testnet"}
	cfg.Networks = specdata.NetworkPins{Commitment: chains, Entropy: chains}
	run := gate(t, cfg, signed(t, proof, shapeSigner))
	Shape(run)
	Policy(run)
	if run.Ended || run.Failed() {
		t.Fatalf("the mutant does not pass SHAPE and POLICY: %s", rowSummary(run.Rows))
	}
}

// failingWith returns the failing DERIVE rows of run whose code is code.
func failingWith(run *Run, code specdata.Reason) []Row {
	var out []Row
	for _, row := range deriveRows(run) {
		if row.Status == StatusFail && row.Code == code {
			out = append(out, row)
		}
	}
	return out
}

// yIsP is the 32 octets whose y is p itself: step 2 of SPEC.md section 6.2
// refuses them, and they are not one of SMALL_ORDER_KEYS.
func yIsP(t *testing.T) [32]byte {
	t.Helper()
	var key [32]byte
	key[0] = 0xED
	for i := 1; i < 31; i++ {
		key[i] = 0xFF
	}
	key[31] = 0x7F
	for _, small := range specdata.SmallOrderKeys() {
		if small.Key == key {
			t.Fatal("y = p is a small-order key")
		}
	}
	return key
}

// The event without handover_signature that the handover signs is each
// other member the rotation carries, as carried: for every rotation and
// emergency rotation of the positive proofs, at every block position, and
// with prerotation_commitment present and absent, unsignedHandover equals
// the event with that one member deleted here. Mutants killed: a member
// the schema lists left out (reason dropped from unsignedHandover);
// prerotation_commitment written when it is absent (a null member);
// handover_signature kept.
func TestKeyEvents_TheUnsignedHandoverIsTheEventWithoutItsSignature(t *testing.T) {
	seen := 0
	for _, name := range sortedProofs(positiveProofs(t)) {
		proof := positiveProofs(t)[name]
		for _, block := range blockPositions {
			event, ok := valueAt(proof, keyEventAt(block)).(map[string]any)
			if !ok || (event["type"] != "rotation" && event["type"] != "emergency_rotation") {
				continue
			}
			for _, variant := range []map[string]any{
				event,
				put(event, Pointer{Name("prerotation_commitment")}, strings.Repeat("ef", 32)),
				remove(put(event, Pointer{Name("prerotation_commitment")}, "x"), Pointer{Name("prerotation_commitment")}),
			} {
				want := clone(variant).(map[string]any)
				delete(want, "handover_signature")
				if got := unsignedHandover(variant); !reflect.DeepEqual(got, want) {
					t.Errorf("%s %s: %v, want %v", name, block.Encode(), got, want)
				}
				seen++
			}
		}
	}
	if seen < 9 {
		t.Fatalf("%d rotation events seen", seen)
	}
}

// An emergency rotation with a handover passes its handover row, whose
// trace row's recomputed is the hex of the 32 octets it verified over; the
// sound emergency rotation, which carries none, makes no handover row, and
// every other row of its event passes. Mutants killed: the handover made
// only in a rotation (the signed emergency rotation has no row at its
// handover_signature); a handover row made for an emergency rotation that
// carries none (a row at its handover_signature, and a fault).
func TestKeyEvents_AnEmergencyRotationWithAndWithoutAHandover(t *testing.T) {
	proofs := positiveProofs(t)
	at := keyEventAt(Pointer{Name("block")})
	outgoing := goed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x43}, 32))
	event := clone(valueAt(proofs["block-rotation"], at)).(map[string]any)
	if event["previous_public_key"] != registry.EncodeB64U(outgoing.Public().(goed25519.PublicKey)) {
		t.Fatalf("block-rotation's outgoing key is not the seed of 0x43: %v", event["previous_public_key"])
	}
	event["type"] = "emergency_rotation"
	delete(event, "handover_signature")
	message, err := hex.DecodeString(modelHash(t, "handover", event))
	if err != nil {
		t.Fatal(err)
	}
	signature := registry.EncodeB64U(goed25519.Sign(outgoing, message))
	event["handover_signature"] = signature
	proof := rehashed(t, put(proofs["block-rotation"], at, event), Pointer{Name("block")})
	run := derived(proof)
	part := under(at, "handover_signature").Encode()
	// The new block_hash is a leaf of neither epoch's tree.
	if got := failing(run); !slices.Equal(got, []string{"derivation_mismatch /commitments/bitcoin/epoch/record/merkle_root",
		"derivation_mismatch /commitments/stellar/epoch/record/merkle_root"}) {
		t.Fatalf("failing %q", got)
	}
	want := []TraceRow{{Construction: "handover", Declared: signature, Inputs: []string{at.Encode(), under(at, "previous_public_key").Encode()},
		Part: part, Recomputed: hex.EncodeToString(message), Status: StatusPass}}
	if got := traceAt(run, part); !reflect.DeepEqual(got, want) {
		t.Fatalf("the handover's trace %+v, want %+v", got, want)
	}

	without := derived(proofs["block-emergency-rotation"])
	if _, carried := valueAt(proofs["block-emergency-rotation"], at).(map[string]any)["handover_signature"]; carried {
		t.Fatal("the sound emergency rotation carries a handover")
	}
	var parts []string
	for _, row := range deriveRows(without) {
		if strings.HasPrefix(*row.Part, at.Encode()+"/") {
			parts = append(parts, *row.Part)
			if row.Status != StatusPass {
				t.Errorf("%s %s", *row.Part, row.Status)
			}
		}
	}
	var wantParts []string
	for _, member := range []string{"height", "key_id", "previous_block_hash", "previous_key_id", "public_key"} {
		wantParts = append(wantParts, under(at, member).Encode())
	}
	if !slices.Equal(parts, wantParts) || len(failing(without)) != 0 {
		t.Fatalf("the emergency rotation's rows %q, failing %q", parts, failing(without))
	}
}

// A revocation, at /block and at /witnesses/block, introduces no key and
// is not checked: no row under its event or at its record's signing_key_id,
// and none in the trace, while the block's own hash is still checked.
// Mutants killed: a revocation checked as an event that introduces a key (a
// fault, since it carries no public_key); a revocation checked as a
// prerotation (a row at signing_key_id).
func TestKeyEvents_ARevocationIsNotChecked(t *testing.T) {
	proofs := positiveProofs(t)
	for _, c := range []struct {
		name  string
		block Pointer
	}{{"block-rotation", Pointer{Name("block")}}, {"item-rotation-witness", Pointer{Name("witnesses"), Name("block")}}} {
		at := keyEventAt(c.block)
		signingKeyID := under(c.block, "record", "signing_key_id").Encode()
		event := valueAt(proofs[c.name], at).(map[string]any)
		if event["type"] != "rotation" {
			t.Fatalf("%s: %v", c.name, event["type"])
		}
		inEvent := func(run *Run) []string {
			var out []string
			for _, row := range deriveRows(run) {
				if *row.Part == signingKeyID || strings.HasPrefix(*row.Part, at.Encode()+"/") {
					out = append(out, *row.Part)
				}
			}
			for _, row := range run.Trace {
				if row.Part == signingKeyID || strings.HasPrefix(row.Part, at.Encode()+"/") {
					out = append(out, "trace "+row.Part)
				}
			}
			return out
		}
		if !slices.Contains(inEvent(derived(proofs[c.name])), signingKeyID) {
			t.Fatalf("%s: the rotation gives no row at %s", c.name, signingKeyID)
		}
		revocation := map[string]any{"revoked_key_id": event["previous_key_id"], "type": "revocation"}
		proof := rehashed(t, put(proofs[c.name], at, revocation), c.block)
		run := derived(proof)
		if got := inEvent(run); len(got) != 0 || run.Ended {
			t.Fatalf("%s: rows %q, ended %v", c.name, got, run.Ended)
		}
		if !strings.Contains(rowSummary(deriveRows(run)), "[null "+under(c.block, "block_hash").Encode()+" pass DERIVE]") {
			t.Fatalf("%s: the block's hash is not checked: %s", c.name, rowSummary(deriveRows(run)))
		}
	}
}

// A genesis key whose y is p fails its bytes check, named y_not_below_p,
// with the values {construction, declared, refused}; SHAPE and POLICY pass
// the mutant, and the new block_hash is a leaf of neither epoch's tree, so
// only the two epoch roots fail besides. Mutants killed: the checks of a
// key's bytes refusing only the small-order keys (the row passes); a
// refused key named small_order_key; the key bytes row given a recomputed.
func TestKeyEvents_AGenesisKeyWhoseYIsPFailsItsBytes(t *testing.T) {
	block := Pointer{Name("block")}
	at := keyEventAt(block)
	key := yIsP(t)
	text := registry.EncodeB64U(key[:])
	proof := put(positiveProofs(t)["block"], under(at, "public_key"), text)
	if valueAt(proof, under(at, "type")) != "genesis" {
		t.Fatal("block's /block carries no genesis event")
	}
	proof = put(put(proof, under(at, "key_id"), modelKeyID(t, text)), under(block, "record", "signing_key_id"), modelKeyID(t, text))
	proof = rehashed(t, proof, block)
	pastPolicy(t, proof)
	part := under(at, "public_key").Encode()
	assertRefused(t, derived(proof), part, map[string]any{"construction": "key_event", "declared": text, "refused": "y_not_below_p"})
}

// A rotation and an emergency rotation hold the key they bring in to the
// checks of a key's bytes, as a genesis does. From block-rotation, /block's
// event, of each type, is given an incoming public_key whose y is p, or the
// first of SMALL_ORDER_KEYS, with its key_id and the record's
// signing_key_id that key's key id, and its handover signed again by the
// outgoing key over the event without its signature. SHAPE and POLICY pass
// the mutant, and the key fails at E/public_key, named y_not_below_p or
// small_order_key, with the values {construction, declared, refused}.
// Mutants killed: the check of a key's bytes passing for every event that
// is not a genesis (every subtest fails); passing for an emergency
// rotation only (its two subtests fail); passing for a rotation only (its
// two subtests fail).
func TestKeyEvents_AnIncomingRotationKeyWhoseYIsPFailsItsBytes(t *testing.T) {
	block := Pointer{Name("block")}
	at := keyEventAt(block)
	sound := positiveProofs(t)["block-rotation"]
	outgoing := goed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x43}, 32))
	if valueAt(sound, under(at, "type")) != "rotation" ||
		valueAt(sound, under(at, "previous_public_key")) != registry.EncodeB64U(outgoing.Public().(goed25519.PublicKey)) {
		t.Fatal("block-rotation's /block is not a rotation from the seed of 0x43")
	}
	for _, typ := range []string{"rotation", "emergency_rotation"} {
		for _, c := range []struct {
			key     [32]byte
			refused string
		}{{yIsP(t), "y_not_below_p"}, {specdata.SmallOrderKeys()[0].Key, "small_order_key"}} {
			t.Run(typ+"/"+c.refused, func(t *testing.T) {
				text := registry.EncodeB64U(c.key[:])
				event := clone(valueAt(sound, at)).(map[string]any)
				event["type"] = typ
				event["public_key"] = text
				event["key_id"] = modelKeyID(t, text)
				delete(event, "handover_signature")
				message, err := hex.DecodeString(modelHash(t, "handover", event))
				if err != nil {
					t.Fatal(err)
				}
				event["handover_signature"] = registry.EncodeB64U(goed25519.Sign(outgoing, message))
				proof := put(put(sound, at, event), under(block, "record", "signing_key_id"), modelKeyID(t, text))
				proof = rehashed(t, proof, block)
				pastPolicy(t, proof)
				assertRefused(t, derived(proof), under(at, "public_key").Encode(),
					map[string]any{"construction": "key_event", "declared": text, "refused": c.refused})
			})
		}
	}
}

// unsignedHandover writes the members spec/schema.json lists for a
// rotation and an emergency rotation, less handover_signature, each as
// carried: over an event of each type that carries every member of its
// properties, exactly those members less handover_signature; over one that
// carries only its required members, exactly those less
// handover_signature. The schema is the embedded spec's, read here.
// Mutants killed: an optional member added to a rotation event in a copy
// of the schema (the first comparison lacks it); a required member made
// optional (the second writes it, as null); a name dropped from
// unsignedHandover's list (the first lacks it); a name added to that list
// (both write it).
func TestKeyEvents_TheUnsignedHandoverWritesTheSchemasMembers(t *testing.T) {
	defs := specSchemaValue(t)["$defs"].(map[string]any)
	names := func(m map[string]any) []string {
		out := make([]string, 0, len(m))
		for name := range m {
			out = append(out, name)
		}
		slices.Sort(out)
		return out
	}
	for _, def := range []string{"ROTATION_EVENT", "EMERGENCY_ROTATION_EVENT"} {
		schema := defs[def].(map[string]any)
		listed := names(schema["properties"].(map[string]any))
		if !slices.Contains(listed, "handover_signature") {
			t.Fatalf("%s lists no handover_signature: %q", def, listed)
		}
		var required []string
		for _, name := range schema["required"].([]any) {
			required = append(required, name.(string))
		}
		for _, c := range []struct {
			carried string
			members []string
		}{{"every member", listed}, {"the required members", required}} {
			event := map[string]any{}
			for _, member := range c.members {
				event[member] = "the value of " + member
			}
			want := clone(event).(map[string]any)
			delete(want, "handover_signature")
			if got := unsignedHandover(event); !reflect.DeepEqual(got, want) {
				t.Errorf("%s with %s: writes %q, want %q (%v)", def, c.carried, names(got), names(want), got)
			}
		}
	}
}

// assertRefused holds run to one failing key-event row, at part, with
// values, and the two epoch roots a re-hashed /block no longer reaches.
func assertRefused(t *testing.T, run *Run, part string, values map[string]any) {
	t.Helper()
	want := []string{"key_event_mismatch " + part, "derivation_mismatch /commitments/bitcoin/epoch/record/merkle_root",
		"derivation_mismatch /commitments/stellar/epoch/record/merkle_root"}
	if got := failing(run); !slices.Equal(got, want) {
		t.Fatalf("failing %q, want %q", got, want)
	}
	rows := failingWith(run, specdata.ReasonKeyEventMismatch)
	if !reflect.DeepEqual(rows[0].Values, values) {
		t.Fatalf("values %v, want %v", rows[0].Values, values)
	}
}

// A rotation's outgoing key whose y is p fails the handover, named
// y_not_below_p, with the values {construction, declared, recomputed,
// refused}, recomputed the hex of the 32 octets the signature was checked
// over. Mutants killed: Verify without the checks of a key's bytes (the
// equation refusal); a refused outgoing key named small_order_key; the
// handover's recomputed left out of a failing row.
func TestKeyEvents_AnOutgoingKeyWhoseYIsPFailsTheHandover(t *testing.T) {
	block := Pointer{Name("block")}
	at := keyEventAt(block)
	key := yIsP(t)
	text := registry.EncodeB64U(key[:])
	proof := put(positiveProofs(t)["block-rotation"], under(at, "previous_public_key"), text)
	proof = rehashed(t, put(proof, under(at, "previous_key_id"), modelKeyID(t, text)), block)
	pastPolicy(t, proof)
	event := clone(valueAt(proof, at)).(map[string]any)
	signature := event["handover_signature"]
	delete(event, "handover_signature")
	assertRefused(t, derived(proof), under(at, "handover_signature").Encode(), map[string]any{
		"construction": "handover", "declared": signature, "recomputed": modelHash(t, "handover", event), "refused": "y_not_below_p",
	})
}

// A handover signature that a key passing steps 2 and 3 does not verify
// names the step of SPEC.md section 6.2 that refused it: s at or above L is
// s_not_below_l (step 5), and a flipped bit of R is equation (step 7); the
// event's other rows pass. Mutants killed: the refusal named for every
// failure as one step (not_verified); a failing handover passing.
func TestKeyEvents_TheHandoverRefusalNamesItsStep(t *testing.T) {
	at := keyEventAt(Pointer{Name("block")})
	sound := positiveProofs(t)["block-rotation"]
	value, err := registry.DecodeB64U64(valueAt(sound, under(at, "handover_signature")).(string))
	if err != nil {
		t.Fatal(err)
	}
	l, _ := new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	highS := value
	le := l.Bytes()
	slices.Reverse(le)
	copy(highS[32:], le)
	flippedR := value
	flippedR[0] ^= 0x01
	for _, c := range []struct {
		signature [64]byte
		refused   string
	}{{highS, "s_not_below_l"}, {flippedR, "equation"}} {
		proof := put(sound, under(at, "handover_signature"), registry.EncodeB64U(c.signature[:]))
		rows := failingWith(derived(proof), specdata.ReasonKeyEventMismatch)
		if len(rows) != 1 || *rows[0].Part != under(at, "handover_signature").Encode() || rows[0].Values["refused"] != c.refused {
			t.Errorf("%s: %s %v", c.refused, rowSummary(rows), rows)
		}
	}
}

// A record whose signing_key_id is not the key id of its event's key fails
// key_event_mismatch at that signing_key_id, at a genesis, a rotation and
// a prerotation, recomputed the key id this run computed from the event's
// key (or, for a prerotation, its active_key_id), while the event's own
// key_id still passes. Mutant killed: its failure given
// derivation_mismatch. (A record's signing_key_id compared with the
// event's carried key_id is Rule 4's.)
func TestKeyEvents_ASigningKeyIDThatIsNotItsEventsKeysID(t *testing.T) {
	proofs := positiveProofs(t)
	for _, c := range []struct {
		name  string
		block Pointer
	}{
		{"block", Pointer{Name("block")}},
		{"block-rotation", Pointer{Name("block")}},
		{"item-rotation-witness", Pointer{Name("witnesses"), Name("block")}},
		{"beacon-empty", Pointer{Name("block")}},
	} {
		at := keyEventAt(c.block)
		event := valueAt(proofs[c.name], at).(map[string]any)
		recomputed := event["active_key_id"]
		if event["type"] != "prerotation" {
			recomputed = modelKeyID(t, event["public_key"])
		}
		part := under(c.block, "record", "signing_key_id")
		run := derived(rehashed(t, put(proofs[c.name], part, "00000000"), c.block))
		rows := failingWith(run, specdata.ReasonKeyEventMismatch)
		want := map[string]any{"construction": "key_event", "declared": "00000000", "recomputed": recomputed}
		if len(rows) != 1 || *rows[0].Part != part.Encode() || !reflect.DeepEqual(rows[0].Values, want) {
			t.Errorf("%s %s: %s %v", c.name, c.block.Encode(), rowSummary(rows), rows)
		}
		if _, introduces := event["public_key"]; introduces && !strings.Contains(rowSummary(deriveRows(run)), "[null "+under(at, "key_id").Encode()+" pass DERIVE]") {
			t.Errorf("%s: the event's key_id does not pass", c.name)
		}
	}
}

// Each link fails at its own part, the carried value it finds wrong (SPEC.md
// section 10.2's one rule, as section 10.3's table gives it), with its
// code, whichever of its two sides a mutant moves: LINK_INCLUSION at the
// record's merkle_root (its size at the proof's tree_size is
// derive_test.go's), LINK_WITNESS at the
// witness_hashes entry, LINK_HEAD at /witnesses/block/record/height,
// LINK_KEY_EVENT at the signing key event's public_key, and
// LINK_CHAIN_ORDER at the entropy's sequence or height, each the one
// failing row of its construction. Mutants killed: LINK_WITNESS at the
// witness it hashes; LINK_HEAD at /block/record/height; LINK_CHAIN_ORDER at
// the commitment's ledger or height; a link failing with
// derivation_mismatch where its table gives another code.
func TestKeyEvents_EachLinkFailsAtItsPart(t *testing.T) {
	proofs := positiveProofs(t)
	item := proofs["item"]
	ledger := valueAt(item, Pointer{Name("commitments"), Name("stellar"), Name("ledger")}).(strictjson.Number)
	height := valueAt(item, Pointer{Name("commitments"), Name("bitcoin"), Name("height")}).(strictjson.Number)
	blockHeight := valueAt(item, Pointer{Name("block"), Name("record"), Name("height")})
	path := clone(valueAt(item, Pointer{Name("subject"), Name("inclusion_proof"), Name("path")})).([]any)
	path[0] = strings.Repeat("00", 32)
	for _, c := range []struct {
		link, construction string
		proof              map[string]any
		key                [32]byte
		code               specdata.Reason
		part               string
	}{
		{"LINK_INCLUSION root", "inclusion", put(item, Pointer{Name("subject"), Name("inclusion_proof"), Name("path")}, path), publicOf(vectorSigner),
			specdata.ReasonDerivationMismatch, "/block/record/merkle_root"},
		{"LINK_WITNESS", "witnesses", put(item, Pointer{Name("witnesses"), Name("entropy_nist"), Name("moved")}, "x"), publicOf(vectorSigner),
			specdata.ReasonDerivationMismatch, "/subject/record/witness_hashes/entropy_nist"},
		{"LINK_HEAD", "height", put(item, Pointer{Name("witnesses"), Name("block"), Name("record"), Name("height")}, blockHeight), publicOf(vectorSigner),
			specdata.ReasonOrderingViolation, "/witnesses/block/record/height"},
		{"LINK_KEY_EVENT", "key_event", item, publicOf(signerOf("another key")),
			specdata.ReasonKeyEventMismatch, signingKeyEventKeyAt.Encode()},
		{"LINK_CHAIN_ORDER stellar", "chain_order", put(item, Pointer{Name("witnesses"), Name("entropy_stellar"), Name("payload"), Name("sequence")}, ledger),
			publicOf(vectorSigner), specdata.ReasonOrderingViolation, "/witnesses/entropy_stellar/payload/sequence"},
		{"LINK_CHAIN_ORDER bitcoin", "chain_order", put(item, Pointer{Name("witnesses"), Name("entropy_bitcoin"), Name("payload"), Name("height")}, height+1),
			publicOf(vectorSigner), specdata.ReasonOrderingViolation, "/witnesses/entropy_bitcoin/payload/height"},
	} {
		run := deriveRunBy(c.proof, c.key, deriveAttribution)
		Derive(run)
		var got []string
		for _, row := range deriveRows(run) {
			if row.Status == StatusFail && row.Values["construction"] == c.construction {
				got = append(got, string(row.Code)+" "+*row.Part)
			}
		}
		if want := []string{string(c.code) + " " + c.part}; !slices.Equal(got, want) {
			t.Errorf("%s: failing %q, want %q", c.link, got, want)
		}
	}
}

// LINK_CHAIN_ORDER pairs an entropy object only with the top-level entry
// on its network: the item's Bitcoin and Stellar witnesses each have a row;
// on another network the Bitcoin witness has none, and no
// /signing_key_event/commitments entry stands in for one; a Bitcoin height
// equal to the entry's passes, one above it fails, with the entry's height
// as the bound. Mutants killed: the networks not compared (the mainnet
// entry still pairs); the signing key event's entry paired; the Bitcoin
// bound strict (equality fails).
func TestKeyEvents_ChainOrderPairsOnlyOnItsNetwork(t *testing.T) {
	order := func(proof map[string]any) []string {
		var out []string
		for _, row := range deriveRows(derived(proof)) {
			if row.Values["construction"] == "chain_order" {
				out = append(out, rowSummary([]Row{row}))
			}
		}
		return out
	}
	bitcoin := Pointer{Name("witnesses"), Name("entropy_bitcoin"), Name("payload"), Name("height")}
	stellar := "[null /witnesses/entropy_stellar/payload/sequence pass DERIVE]"
	item := positiveProofs(t)["item"]
	if _, ok := valueAt(item, Pointer{Name("signing_key_event"), Name("commitments"), Name("bitcoin")}).(map[string]any); !ok {
		t.Fatal("item's signing key event has no Bitcoin entry")
	}
	if got := order(item); !slices.Equal(got, []string{"[null " + bitcoin.Encode() + " pass DERIVE]", stellar}) {
		t.Fatalf("item: %q", got)
	}
	if got := order(put(item, Pointer{Name("commitments"), Name("bitcoin"), Name("network")}, "mainnet")); !slices.Equal(got, []string{stellar}) {
		t.Fatalf("another network: %q", got)
	}
	committed := valueAt(item, Pointer{Name("commitments"), Name("bitcoin"), Name("height")}).(strictjson.Number)
	if got := order(put(item, bitcoin, committed)); !slices.Equal(got, []string{"[null " + bitcoin.Encode() + " pass DERIVE]", stellar}) {
		t.Fatalf("an equal height: %q", got)
	}
	above := put(item, bitcoin, committed+1)
	rows := failingWith(derived(above), specdata.ReasonOrderingViolation)
	want := map[string]any{"construction": "chain_order", "declared": committed + 1, "recomputed": committed}
	if len(rows) != 1 || *rows[0].Part != bitcoin.Encode() || !reflect.DeepEqual(rows[0].Values, want) {
		t.Fatalf("a height above: %s %v", rowSummary(rows), rows)
	}
}

// LINK_HEAD at a genesis /block counts its height as 0, whatever it
// declares: with the genesis witness's metadata on /block, which SHAPE and
// POLICY pass, the link fails ordering_violation with the values
// {construction height, declared, recomputed 0}, the genesis check fails
// /block/record/height, and no trace row is LINK_HEAD's. Mutants killed:
// LINK_HEAD reading /block's declared height (the link passes); LINK_HEAD
// traced.
func TestKeyEvents_LinkHeadCountsAGenesisBlockAsZero(t *testing.T) {
	sound := positiveProofs(t)["item-genesis-witness"]
	metadata := valueAt(sound, Pointer{Name("witnesses"), Name("block"), Name("record"), Name("metadata")}).(map[string]any)
	if _, genesis := metadata["genesis"]; !genesis {
		t.Fatal("item-genesis-witness's witness block is not the genesis block")
	}
	proof := put(sound, Pointer{Name("block"), Name("record"), Name("metadata")}, metadata)
	pastPolicy(t, proof)
	run := derived(proof)
	part := "/witnesses/block/record/height"
	var head []Row
	for _, row := range deriveRows(run) {
		if *row.Part == part && row.Values["construction"] == "height" {
			head = append(head, row)
		}
	}
	want := map[string]any{"construction": "height", "declared": valueAt(proof, pointerOf(t, part)), "recomputed": strictjson.Number(0)}
	if len(head) != 1 || head[0].Code != specdata.ReasonOrderingViolation || !reflect.DeepEqual(head[0].Values, want) {
		t.Fatalf("LINK_HEAD %s %v", rowSummary(head), head)
	}
	if !slices.Contains(failing(run), "derivation_mismatch /block/record/height") {
		t.Fatalf("failing %q", failing(run))
	}
	for _, row := range run.Trace {
		if row.Construction == "height" {
			t.Fatalf("a trace row of LINK_HEAD %+v", row)
		}
	}
}

// Rule 3: each key event type is checked in full at its position. A
// rotation relabeled an emergency rotation still has its handover checked,
// and fails it, since the signature covers the type; without a handover,
// the rest still runs. Mutant killed: the handover made only for the type
// rotation.
func TestKeyEvents_Rule3_EachKeyEventTypeIsCheckedInFull(t *testing.T) {
	sound := positiveProofs(t)["block-rotation"]
	at := keyEventAt(Pointer{Name("block")})
	parts := func(run *Run) []string {
		var out []string
		for _, row := range deriveRows(run) {
			if strings.HasPrefix(*row.Part, at.Encode()) {
				out = append(out, *row.Part)
			}
		}
		return out
	}
	emergency := derived(put(sound, under(at, "type"), "emergency_rotation"))
	if !slices.Equal(parts(emergency), parts(derived(sound))) ||
		!slices.Contains(failing(emergency), "key_event_mismatch "+under(at, "handover_signature").Encode()) {
		t.Fatalf("relabeled: %q, failing %q", parts(emergency), failing(emergency))
	}
	without := remove(put(sound, under(at, "type"), "emergency_rotation"), under(at, "handover_signature"))
	want := slices.DeleteFunc(parts(derived(sound)), func(p string) bool { return p == under(at, "handover_signature").Encode() })
	if got := parts(derived(without)); !slices.Equal(got, want) {
		t.Fatalf("without a handover: %q, want %q", got, want)
	}
}

// Rule 4: a key id is recomputed from its key, and a declared one is only
// compared. A genesis event's replaced key_id fails at its own part alone
// among the key events, and the record's signing_key_id, compared with the
// key id recomputed from the key, still passes. Mutant killed: the
// signing_key_id compared with the event's carried key_id (it fails too).
func TestKeyEvents_Rule4_AKeyIDIsRecomputedFromItsKey(t *testing.T) {
	block := Pointer{Name("block")}
	at := keyEventAt(block)
	run := derived(rehashed(t, put(positiveProofs(t)["block"], under(at, "key_id"), "00000000"), block))
	var parts []string
	for _, row := range failingWith(run, specdata.ReasonKeyEventMismatch) {
		parts = append(parts, *row.Part)
	}
	if !slices.Equal(parts, []string{under(at, "key_id").Encode()}) ||
		!strings.Contains(rowSummary(deriveRows(run)), "[null /block/record/signing_key_id pass DERIVE]") {
		t.Fatalf("key_event_mismatch at %q: %s", parts, rowSummary(deriveRows(run)))
	}
}

// keyIDCollision returns two 32-octet keys with one key id, found here by
// a birthday search; the last octet 0x01 keeps y below p and each key out
// of SMALL_ORDER_KEYS.
func keyIDCollision(t *testing.T) ([32]byte, [32]byte) {
	t.Helper()
	seen := map[string][32]byte{}
	for n := uint64(0); n < 1<<24; n++ {
		var key [32]byte
		for i := range 8 {
			key[i] = byte(n >> (8 * i))
		}
		key[31] = 0x01
		id := modelKeyID(t, registry.EncodeB64U(key[:]))
		if earlier, ok := seen[id]; ok {
			return earlier, key
		}
		seen[id] = key
	}
	t.Fatal("no key id collision")
	return [32]byte{}, [32]byte{}
}

// LINK_KEY_EVENT compares all 32 octets of K, never its key id: an event
// that introduces one of two keys sharing a key id fails when K is the
// other, and does not when K is the one; and the genuine key with its last
// octet flipped is another key. Under every attribution. Mutants killed:
// the comparison by key id (the collision partner passes); a comparison of
// the first 31 octets (the flipped key passes).
func TestKeyEvents_LinkKeyEventComparesAll32Octets(t *testing.T) {
	a, b := keyIDCollision(t)
	if a == b {
		t.Fatal("the collision keys are equal")
	}
	link := signingKeyEventKeyAt.Encode()
	linkRow := func(run *Run) string {
		rows := deriveRows(run)
		var last []Row
		for _, row := range rows {
			if *row.Part == link {
				last = []Row{row}
			}
		}
		return rowSummary(last)
	}
	fails := "[key_event_mismatch " + link + " fail DERIVE]"
	event := put(positiveProofs(t)["item"], signingKeyEventKeyAt, registry.EncodeB64U(a[:]))
	flipped := publicOf(vectorSigner)
	flipped[31] ^= 0x01
	for _, attribution := range []string{AttributionTruestamp, deriveAttribution, AttributionUnattributed} {
		other := deriveRunBy(event, b, attribution)
		Derive(other)
		same := deriveRunBy(event, a, attribution)
		Derive(same)
		flip := deriveRunBy(positiveProofs(t)["item"], flipped, attribution)
		Derive(flip)
		if linkRow(other) != fails || linkRow(same) == fails || linkRow(flip) != fails {
			t.Fatalf("under %s: the partner %s, the key %s, flipped %s", attribution, linkRow(other), linkRow(same), linkRow(flip))
		}
	}
}

// The checks that share a part keep the order of their checks: at a
// genesis /witnesses/block's height, the genesis check of step 9 (b), then
// LINK_HEAD of (e), in deriveChecks and in the rows (the
// declared-genesis_height-witnesses case, whose genesis check fails and
// whose link passes); at the signing key event's public_key, the check of
// its bytes, then LINK_KEY_EVENT, both untraced. Mutant killed: deriveChecks
// giving the links before the records (each order swaps).
func TestKeyEvents_TheChecksThatShareAPartKeepTheirOrder(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(cases, func(c vectorcases.Case) bool { return c.ID == "declared-genesis_height-witnesses" })
	if i < 0 {
		t.Fatal("no case declared-genesis_height-witnesses")
	}
	data, err := strictjson.ReadFile(cases[i].ProofPath())
	if err != nil {
		t.Fatal(err)
	}
	document, err := strictjson.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	proof := document.(map[string]any)["proof"].(map[string]any)
	key, err := registry.DecodeB64U32(document.(map[string]any)["signature"].(map[string]any)["public_key"].(string))
	if err != nil {
		t.Fatal(err)
	}
	at := func(checks []deriveCheck, part string) []string {
		var out []string
		for _, c := range checks {
			if c.part.Encode() == part {
				out = append(out, c.construction+" "+string(c.row().Status)+" "+map[bool]string{true: "traced", false: "untraced"}[c.traced])
			}
		}
		return out
	}
	checks, err := deriveChecks(proof, key, deriveAttribution)
	if err != nil {
		t.Fatal(err)
	}
	head := "/witnesses/block/record/height"
	if got := at(checks, head); !slices.Equal(got, []string{"genesis fail traced", "height pass untraced"}) {
		t.Fatalf("the checks at %s: %q", head, got)
	}
	run := deriveRunBy(proof, key, deriveAttribution)
	Derive(run)
	var rows []Row
	for _, row := range deriveRows(run) {
		if *row.Part == head {
			rows = append(rows, row)
		}
	}
	if got := rowSummary(rows); got != "[derivation_mismatch "+head+" fail DERIVE] [null "+head+" pass DERIVE]" {
		t.Fatalf("the rows at %s: %s", head, got)
	}
	if got := at(checks, signingKeyEventKeyAt.Encode()); !slices.Equal(got, []string{"key_event pass untraced", "key_event info untraced"}) {
		t.Fatalf("the checks at the signing key event's key: %q", got)
	}
}
