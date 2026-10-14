// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// recordKinds are the rows of SPEC.md section 4.3 that hash JCS of a value.
var recordKinds = []Construction{
	ClaimsHash, ItemHash, EntropyHash, ObservationHash,
	GenesisPreviousBlockHash, BlockHash, EpochHash, ProofHash, Handover,
}

func sha256Hex(parts ...[]byte) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write(part)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func mustObject(t *testing.T, text string) map[string]any {
	t.Helper()
	object, ok := mustParse(t, "record", []byte(text)).(map[string]any)
	if !ok {
		t.Fatalf("%s is not an object", text)
	}
	return object
}

// H(p, R) is SHA-256 over the kind's byte and then JCS of the record, for
// every kind under the rule. Mutants killed: digest hashing a byte one
// above the row's; digest leaving the byte out.
func TestRecordHash_HashesTheKindsByteThenJCS(t *testing.T) {
	record := mustObject(t, `{"b": [1, "x"], "a": {"c": null}}`)
	canonical := `{"a":{"c":null},"b":[1,"x"]}`
	for _, kind := range recordKinds {
		want := sha256Hex([]byte{specBytes[kind]}, []byte(canonical))
		sum, err := RecordHash(kind, record)
		if err != nil {
			t.Fatalf("%v: %v", kind, err)
		}
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("RecordHash(%v) = %s, want %s", kind, got, want)
		}
		if got, err := RecordHashHex(kind, record); err != nil || got != want {
			t.Errorf("RecordHashHex(%v) = %s, %v, want %s", kind, got, err, want)
		}
	}
}

// The rule covers exactly the nine kinds of SPEC.md section 4.3 that hash
// JCS of a value, and refuses the four outside it (section 4.1) and a value
// that names no row. Mutants killed: Handover left out of isRecordKind;
// KeyID added to it.
func TestRecordHash_CoversExactlyTheNineRecordKinds(t *testing.T) {
	if got := RecordKinds(); !slices.Equal(got, recordKinds) {
		t.Fatalf("RecordKinds() = %v, want %v", got, recordKinds)
	}
	for _, kind := range append(constructions(), 0, Handover+1) {
		_, err := RecordHash(kind, map[string]any{})
		if slices.Contains(recordKinds, kind) {
			if err != nil {
				t.Errorf("RecordHash(%v) refused: %v", kind, err)
			}
		} else if !errors.Is(err, ErrNotRecordKind) {
			t.Errorf("RecordHash(%v) error = %v, want ErrNotRecordKind", kind, err)
		}
	}
	for _, kind := range []Construction{MerkleLeaf, MerkleNode, KeyID, PrerotationCommitment} {
		if _, err := RecordHashHex(kind, map[string]any{}); !errors.Is(err, ErrNotRecordKind) {
			t.Errorf("RecordHashHex(%v) error = %v, want ErrNotRecordKind", kind, err)
		}
	}
}

// A nil record is refused, never hashed as {}, while a parsed {}, which is
// never nil, and an empty map literal are hashed as {}. Mutant killed:
// RecordHash without its nil check, which hashes nil as H(p, {}).
func TestRecordHash_RefusesANilRecordAndHashesTheEmptyObject(t *testing.T) {
	var absent map[string]any
	for _, kind := range recordKinds {
		if sum, err := RecordHash(kind, absent); !errors.Is(err, ErrNotObject) {
			t.Errorf("RecordHash(%v, nil) = %x, %v, want ErrNotObject", kind, sum, err)
		}
		if text, err := RecordHashHex(kind, nil); !errors.Is(err, ErrNotObject) || text != "" {
			t.Errorf("RecordHashHex(%v, nil) = %q, %v, want ErrNotObject", kind, text, err)
		}
	}
	if _, err := RecordHash(KeyID, nil); !errors.Is(err, ErrNotRecordKind) {
		t.Errorf("RecordHash(KeyID, nil) error = %v, want ErrNotRecordKind first", err)
	}
	parsed := mustObject(t, `{}`)
	if parsed == nil {
		t.Fatal("the strict reader parsed {} as a nil map")
	}
	// One member the document holds as null and one it lacks: each type
	// assertion gives a nil map, the record that was never parsed.
	doc := mustObject(t, `{"record": null}`)
	fromNull, _ := doc["record"].(map[string]any)
	fromAbsent, _ := doc["absent"].(map[string]any)
	for _, kind := range recordKinds {
		want := sha256Hex([]byte{specBytes[kind]}, []byte("{}"))
		for name, record := range map[string]map[string]any{"parsed {}": parsed, "map literal": {}} {
			sum, err := RecordHash(kind, record)
			if err != nil {
				t.Fatalf("RecordHash(%v, %s): %v", kind, name, err)
			}
			if got := hex.EncodeToString(sum[:]); got != want {
				t.Errorf("RecordHash(%v, %s) = %s, want H(p, {}) = %s", kind, name, got, want)
			}
		}
		for name, record := range map[string]map[string]any{"a null member": fromNull, "an absent member": fromAbsent} {
			if _, err := RecordHash(kind, record); !errors.Is(err, ErrNotObject) {
				t.Errorf("RecordHash(%v, %s) error = %v, want ErrNotObject", kind, name, err)
			}
		}
	}
}

// A record that is not within the value profile is refused with JCS's
// error, never hashed. Mutant killed: RecordHash ignoring Encode's error.
func TestRecordHash_RefusesAValueOutsideTheProfile(t *testing.T) {
	for _, record := range []map[string]any{
		{"n": strictjson.Number(math.NaN())},
		{"n": strictjson.Number(9007199254740992)},
		{"n": 1},
	} {
		if sum, err := RecordHash(BlockHash, record); !errors.Is(err, jcs.ErrOutsideProfile) {
			t.Errorf("RecordHash(%v) = %x, %v, want jcs.ErrOutsideProfile", record, sum, err)
		}
	}
}

// The hash depends on the parsed value, not on how its file was written,
// and JCS orders member names by their UTF-16 code units. Mutant killed:
// RecordHash hashing encoding/json's Marshal of the record, which orders
// the names by their UTF-8 bytes.
func TestRecordHash_DependsOnTheValueNotItsSpelling(t *testing.T) {
	one := mustObject(t, `{"a":1,"é":"é","😀":0,"｡":0}`)
	two := mustObject(t, "{ \"｡\" : 0, \"\U0001F600\" : 0.0,\r\n \"é\" : \"é\" , \"a\" : 1.0e0 }")
	first, err := RecordHash(ItemHash, one)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RecordHash(ItemHash, two)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("one value, two hashes: %x and %x", first, second)
	}
	// JCS orders the names by UTF-16 code units: U+1F600 (0xD83D ...)
	// before U+FF61.
	canonical := "{\"a\":1,\"é\":\"é\",\"\U0001F600\":0,\"｡\":0}"
	if got, want := hex.EncodeToString(first[:]), sha256Hex([]byte{0x11}, []byte(canonical)); got != want {
		t.Fatalf("RecordHash = %s, want %s", got, want)
	}
}

// The worked example of SPEC.md section 6.3: proof_hash is H(0x50, proof)
// over its JCS M. Mutant killed: ProofHash's byte 0x50 changed to 0x60.
func TestRecordHash_WorkedExampleProofHash(t *testing.T) {
	example := mustParseFile(t, vectorsDir+"/worked-example.json").(map[string]any)
	proof := example["proof"].(map[string]any)
	canonical, err := jcs.Encode(proof)
	if err != nil {
		t.Fatal(err)
	}
	if string(canonical) != example["M"] {
		t.Fatalf("JCS(proof) = %s, want M %s", canonical, example["M"])
	}
	if got, err := RecordHashHex(ProofHash, proof); err != nil || got != example["proof_hash"] {
		t.Fatalf("H(0x50, proof) = %s, %v, want %s", got, err, example["proof_hash"])
	}
}

// recordKindOfRow is the kind a trace row of the sound proofs recomputes as
// a record hash from its one input, or 0 for a row that is not one, as the
// reference's record_hash_test.exs reads them.
func recordKindOfRow(construction, part string) Construction {
	switch construction {
	case "witnesses":
		if lastSegment(part) == "block" {
			return BlockHash
		}
		return EntropyHash
	case "genesis":
		if lastSegment(part) == "previous_block_hash" {
			return GenesisPreviousBlockHash
		}
		return 0
	}
	for _, kind := range recordKinds {
		if kind != Handover && kind.String() == construction {
			return kind
		}
	}
	return 0
}

// Every record hash the sound proofs' traces recompute from one input is H
// of that input, and equals the value the proof declares; seven kinds are
// among them. Mutants killed: the table's bytes of block_hash (0x31 to
// 0x30) and of claims_hash (0x10 to 0x11).
func TestRecordHash_TraceRowsOfTheSoundProofs(t *testing.T) {
	seen := map[Construction]int{}
	for _, sound := range soundProofs(t) {
		for _, row := range sound.trace {
			construction, _ := row["construction"].(string)
			part, _ := row["part"].(string)
			in := inputs(t, row)
			kind := recordKindOfRow(construction, part)
			if kind == 0 || len(in) != 1 {
				continue
			}
			record, ok := pointer(t, sound.proof, in[0]).(map[string]any)
			if !ok {
				t.Fatalf("%s: %s is not an object", sound.name, in[0])
			}
			got, err := RecordHashHex(kind, record)
			if err != nil {
				t.Fatalf("%s: %v", sound.name, err)
			}
			if got != row["recomputed"] || got != row["declared"] {
				t.Errorf("%s %s: H(%v, %s) = %s, the trace recomputes %v and declares %v",
					sound.name, part, kind, in[0], got, row["recomputed"], row["declared"])
			}
			seen[kind]++
		}
	}
	want := []Construction{ClaimsHash, ItemHash, EntropyHash, ObservationHash, GenesisPreviousBlockHash, BlockHash, EpochHash}
	for _, kind := range want {
		if seen[kind] == 0 {
			t.Errorf("no trace row recomputes %v", kind)
		}
	}
	if len(seen) != len(want) {
		t.Errorf("the rows recompute %v, want exactly %v", seen, want)
	}
}

// Every handover signature of the sound proofs verifies under its
// previous_public_key over H(0x62, the event without handover_signature),
// whose hex the trace records, and over no other kind's hash. Mutant
// killed: Handover's byte 0x62 changed to 0x52.
func TestRecordHash_HandoverSignaturesOfTheSoundProofs(t *testing.T) {
	checked := 0
	for _, sound := range soundProofs(t) {
		for _, row := range sound.trace {
			in := inputs(t, row)
			if row["construction"] != "handover" || len(in) != 2 {
				continue
			}
			event := map[string]any{}
			for name, value := range pointer(t, sound.proof, in[0]).(map[string]any) {
				if name != "handover_signature" {
					event[name] = value
				}
			}
			message, err := RecordHash(Handover, event)
			if err != nil {
				t.Fatal(err)
			}
			if got := hex.EncodeToString(message[:]); got != row["recomputed"] {
				t.Errorf("%s: H(0x62, event) = %s, the trace recomputes %v", sound.name, got, row["recomputed"])
			}
			key, err := DecodeB64U32(stringAt(t, sound.proof, in[1]))
			if err != nil {
				t.Fatal(err)
			}
			signature, err := DecodeB64U64(row["declared"].(string))
			if err != nil {
				t.Fatal(err)
			}
			if !ed25519.Verify(key[:], message[:], signature[:]) {
				t.Errorf("%s: the handover signature does not verify over H(0x62, event)", sound.name)
			}
			other, _ := RecordHash(BlockHash, event)
			if ed25519.Verify(key[:], other[:], signature[:]) {
				t.Errorf("%s: the handover signature verifies over H(0x31, event)", sound.name)
			}
			checked++
		}
	}
	if checked != 12 {
		t.Fatalf("checked %d handover signatures, want 12", checked)
	}
}

// Each sound proof is signed by its key over H(0x50, proof), and not over
// the hash of the proof with one member changed. Mutant killed: RecordHash
// hashing JCS of the record without its prefix byte.
func TestRecordHash_SoundProofsAreSignedOverTheirProofHash(t *testing.T) {
	for _, sound := range soundProofs(t) {
		signature := sound.document["signature"].(map[string]any)
		key, err := DecodeB64U32(signature["public_key"].(string))
		if err != nil {
			t.Fatal(err)
		}
		value, err := DecodeB64U64(signature["value"].(string))
		if err != nil {
			t.Fatal(err)
		}
		message, err := RecordHash(ProofHash, sound.proof)
		if err != nil {
			t.Fatal(err)
		}
		if !ed25519.Verify(key[:], message[:], value[:]) {
			t.Errorf("%s: the signature does not verify over H(0x50, proof)", sound.name)
		}
		altered := map[string]any{}
		for name, member := range sound.proof {
			altered[name] = member
		}
		altered["version"] = strictjson.Number(2)
		message, _ = RecordHash(ProofHash, altered)
		if ed25519.Verify(key[:], message[:], value[:]) {
			t.Errorf("%s: the signature verifies over the altered proof", sound.name)
		}
	}
}
