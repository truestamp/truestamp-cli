// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// seedKey is the public key of the Ed25519 seed of 32 bytes of seed.
func seedKey(seed byte) [32]byte {
	public := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, ed25519.SeedSize)).Public().(ed25519.PublicKey)
	return [32]byte(public)
}

// A key id is the hex of the first 4 bytes of SHA-256(0x60 || the key),
// whatever the key. Mutants killed: the id cut to 5 bytes; the id taken
// from the last 4 bytes; KeyID's byte 0x60 changed to 0x51.
func TestKeyIDOf_IsTheFirstFourBytesOfSHA256OfItsByteAndTheKey(t *testing.T) {
	for _, key := range [][32]byte{{}, [32]byte(bytes.Repeat([]byte{7}, 32)), [32]byte(bytes.Repeat([]byte{0xff}, 32))} {
		sum := sha256.Sum256(append([]byte{0x60}, key[:]...))
		if got, want := KeyIDOf(key), hex.EncodeToString(sum[:4]); got != want {
			t.Errorf("KeyIDOf(%x) = %s, want %s", key, got, want)
		}
	}
}

// The key ids of SPEC.md: the published seeds' keys (section 9),
// VECTOR_KEY of the worked example (section 6.3), every key of
// spec/keys.nonproduction.json and of the vectors' test keyring. Mutant
// killed: KeyIDOf hashing under the prerotation commitment's byte.
func TestKeyIDOf_TheKeysOfTheSpecAndTheVectors(t *testing.T) {
	for seed, id := range map[byte]string{0x41: "f12cfabb", 0x42: "fbd680e8", 0x43: "9602c03e", 0x44: "d0b1cefe"} {
		if got := KeyIDOf(seedKey(seed)); got != id {
			t.Errorf("the key of seed %#02x has key id %s, want %s", seed, got, id)
		}
	}
	vectorKey, err := hex.DecodeString(specdata.VectorKey)
	if err != nil {
		t.Fatal(err)
	}
	example := mustParseFile(t, filepath.Join(vectorsDir, "worked-example.json")).(map[string]any)
	if got := KeyIDOf([32]byte(vectorKey)); got != example["key_id"] {
		t.Errorf("VECTOR_KEY has key id %s, the worked example %v", got, example["key_id"])
	}
	checked := 0
	for _, keys := range []any{
		mustParseSpecFile(t, "keys.nonproduction.json"),
		mustParseFile(t, filepath.Join(vectorsDir, "keyring.test.json")),
	} {
		for _, entry := range keys.(map[string]any)["keys"].([]any) {
			key := entry.(map[string]any)
			public, err := DecodeB64U32(key["public_key"].(string))
			if err != nil {
				t.Fatal(err)
			}
			if got := KeyIDOf(public); got != key["key_id"] {
				t.Errorf("key %v has key id %s", key["public_key"], got)
			}
			checked++
		}
	}
	if checked < 2 {
		t.Fatalf("checked %d keyring keys", checked)
	}
}

// keyIDRow says whether a trace row of the sound proofs recomputes a key
// id from one key, as the reference's record_hash_test.exs reads them: a
// key event's key_id, its signing_key_id when that is taken from its own
// public_key, and a handover's previous_key_id.
func keyIDRow(construction, part, input string) bool {
	switch {
	case construction == "key_event" && lastSegment(part) == "key_id":
		return true
	case construction == "key_event" && lastSegment(part) == "signing_key_id":
		return lastSegment(input) == "public_key"
	case construction == "handover" && lastSegment(part) == "previous_key_id":
		return true
	}
	return false
}

// Every key id the sound proofs' traces recompute is KeyIDOf its input
// key, and equals the id the proof carries: 16 key_id, 16 signing_key_id
// and 13 previous_key_id rows. Mutant killed: KeyIDOf returning the hex of
// the key's own first 4 bytes.
func TestKeyIDOf_TraceRowsOfTheSoundProofs(t *testing.T) {
	counts := map[string]int{}
	for _, sound := range soundProofs(t) {
		for _, row := range sound.trace {
			construction, _ := row["construction"].(string)
			part, _ := row["part"].(string)
			in := inputs(t, row)
			if len(in) != 1 || !keyIDRow(construction, part, in[0]) {
				continue
			}
			key, err := DecodeB64U32(stringAt(t, sound.proof, in[0]))
			if err != nil {
				t.Fatalf("%s: %s: %v", sound.name, in[0], err)
			}
			got := KeyIDOf(key)
			if got != row["recomputed"] || got != stringAt(t, sound.proof, part) {
				t.Errorf("%s %s: key id %s, the trace recomputes %v", sound.name, part, got, row["recomputed"])
			}
			counts[lastSegment(part)]++
		}
	}
	if counts["key_id"] != 16 || counts["signing_key_id"] != 16 || counts["previous_key_id"] != 13 || len(counts) != 3 {
		t.Fatalf("checked %v, want 16 key_id, 16 signing_key_id and 13 previous_key_id rows", counts)
	}
}

// A prerotation commitment is the hex of SHA-256(0x61 || the next key),
// all 32 bytes of it. Mutants killed: PrerotationCommitmentOf hashing under
// KeyID's byte; its hex cut to the first 4 bytes.
func TestPrerotationCommitmentOf_IsSHA256OfItsByteAndTheKey(t *testing.T) {
	for _, key := range [][32]byte{{}, seedKey(0x41), seedKey(0x42)} {
		sum := sha256.Sum256(append([]byte{0x61}, key[:]...))
		if got, want := PrerotationCommitmentOf(key), hex.EncodeToString(sum[:]); got != want {
			t.Errorf("PrerotationCommitmentOf(%x) = %s, want %s", key, got, want)
		}
	}
}

// Each prerotation_commitment the sound proofs carry, four in all, is the
// prerotation commitment of a key one of them carries: the vectors'
// generator commits each to a key a later event introduces. Mutant killed:
// the prerotation commitment's byte 0x61 changed to 0x52.
func TestPrerotationCommitmentOf_TheSoundProofs(t *testing.T) {
	commitments := map[string][]string{}
	committed := map[string]string{}
	var collect func(name, at string, value any)
	collect = func(name, at string, value any) {
		switch typed := value.(type) {
		case map[string]any:
			for member, inner := range typed {
				collect(name, at+"/"+member, inner)
			}
		case string:
			switch {
			case strings.HasSuffix(at, "/prerotation_commitment"):
				commitments[typed] = append(commitments[typed], name+" "+at)
			case strings.HasSuffix(at, "public_key"):
				key, err := DecodeB64U32(typed)
				if err != nil {
					t.Fatalf("%s %s: %v", name, at, err)
				}
				committed[PrerotationCommitmentOf(key)] = typed
			}
		}
	}
	for _, sound := range soundProofs(t) {
		collect(sound.name, "", sound.document)
	}
	found := 0
	for commitment, sites := range commitments {
		if _, ok := committed[commitment]; !ok {
			t.Errorf("no key of the sound proofs commits to %s, carried at %v", commitment, sites)
		}
		found += len(sites)
	}
	if found != 4 {
		t.Fatalf("the sound proofs carry %d prerotation commitments, want 4", found)
	}
}
