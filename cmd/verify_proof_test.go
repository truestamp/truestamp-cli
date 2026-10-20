// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// This file makes the proofs the verify tests run that the shared vectors
// do not hold: a proof signed by a key that exists only inside the test,
// so that a run can be attributed to Truestamp (SPEC.md section 10.5) and
// labelled verified or internally-consistent, and a proof marked with
// another environment (SPEC.md section 6.4). Each is the sound proof
// vectors/positive/item-genesis-witness.proof.json rebuilt from its own
// values: the genesis block's key event, the networks, the marker and the
// signer change, and every value that depends on them is recomputed by the
// constructions text (SPEC.md section 7.3): the genesis block's
// previous_block_hash and block_hash, the item's item_hash, the next
// block's merkle_root and block_hash, the epoch's merkle_root and
// epoch_hash, both inclusion paths, and the signature over proof_hash.
// TestTestProof_RebuildsTheVector holds the rebuild to the vector's own
// bytes.

// testProofOptions say how to rebuild the proof.
type testProofOptions struct {
	// seed is the 32-byte Ed25519 seed of the signer, and of the key the
	// genesis block introduces.
	seed [32]byte
	// environment is the marker, or "" for none (a production proof).
	environment string
	// stellar and bitcoin are the networks of the Stellar commitment
	// entries and of genesis.bitcoin.
	stellar, bitcoin string
}

// vectorSeed is the seed of 32 bytes of 0x42, whose key is VECTOR_KEY
// (SPEC.md section 9).
var vectorSeed = [32]byte(bytes.Repeat([]byte{0x42}, 32))

// testOnlySeed is the seed of a key that exists only in these tests: no
// keyring of the spec holds it, and spec/keys.nonproduction.json does not
// list it, so a production keyring may hold it.
var testOnlySeed = [32]byte(bytes.Repeat([]byte{0x7d}, 32))

// keyIDOf is the key id of the public key of seed, the hex of the first 4
// bytes of SHA-256(0x60 || K) (SPEC.md section 3.3).
func keyIDOf(seed [32]byte) string {
	sum := prefixed(0x60, ed25519.NewKeyFromSeed(seed[:]).Public().(ed25519.PublicKey))
	return hex.EncodeToString(sum[:4])
}

// publicKeyOf is the public key of seed as a B64U32.
func publicKeyOf(seed [32]byte) string {
	return base64.RawURLEncoding.EncodeToString(ed25519.NewKeyFromSeed(seed[:]).Public().(ed25519.PublicKey))
}

// prefixed is SHA-256(prefix || data).
func prefixed(prefix byte, data []byte) []byte {
	sum := sha256.Sum256(append([]byte{prefix}, data...))
	return sum[:]
}

// recordHash is SHA-256(prefix || JCS(value)), as hex.
func recordHash(t *testing.T, prefix byte, value any) string {
	t.Helper()
	encoded, err := jcs.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(prefixed(prefix, encoded))
}

// leafOf is a Merkle leaf, SHA-256(0x00 || the bytes of hash), as hex
// (SPEC.md section 5.1).
func leafOf(t *testing.T, hash string) string {
	t.Helper()
	raw, err := hex.DecodeString(hash)
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(prefixed(0x00, raw))
}

// nodeOf is a Merkle node, SHA-256(0x01 || left || right), as hex.
func nodeOf(t *testing.T, left, right string) string {
	t.Helper()
	l, err := hex.DecodeString(left)
	if err != nil {
		t.Fatal(err)
	}
	r, err := hex.DecodeString(right)
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(prefixed(0x01, append(l, r...)))
}

// object is the member name of m, as an object.
func object(t *testing.T, m any, names ...string) map[string]any {
	t.Helper()
	for _, name := range names {
		next, ok := m.(map[string]any)[name]
		if !ok {
			t.Fatalf("the proof has no member %q", name)
		}
		m = next
	}
	return m.(map[string]any)
}

// testProof is the rebuilt envelope (see the file's doc), written with
// JCS.
func testProof(t *testing.T, o testProofOptions) []byte {
	t.Helper()
	raw, err := os.ReadFile(vectorcases.Path("positive/item-genesis-witness.proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := strictjson.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	document := parsed.(map[string]any)
	proof := object(t, document, "proof")
	if o.environment == "" {
		delete(proof, "environment")
	} else {
		proof["environment"] = o.environment
	}
	keyID := keyIDOf(o.seed)

	genesis := object(t, proof, "signing_key_event", "block")
	record := object(t, genesis, "record")
	metadata := object(t, record, "metadata")
	object(t, metadata, "genesis", "bitcoin")["network"] = o.bitcoin
	event := object(t, metadata, "key_event")
	event["public_key"], event["key_id"] = publicKeyOf(o.seed), keyID
	record["signing_key_id"] = keyID
	record["previous_block_hash"] = recordHash(t, 0x30, metadata["genesis"])
	genesisHash := recordHash(t, 0x31, record)
	genesis["block_hash"] = genesisHash
	witness, err := strictjson.Parse(mustEncode(t, genesis))
	if err != nil {
		t.Fatal(err)
	}
	object(t, proof, "witnesses")["block"] = witness

	subject := object(t, proof, "subject")
	object(t, subject, "record", "witness_hashes")["block"] = genesisHash
	itemHash := recordHash(t, 0x11, object(t, subject, "record"))
	subject["item_hash"] = itemHash

	block := object(t, proof, "block")
	blockRecord := object(t, block, "record")
	blockRecord["previous_block_hash"], blockRecord["signing_key_id"] = genesisHash, keyID
	blockRecord["merkle_root"] = leafOf(t, itemHash)
	blockHash := recordHash(t, 0x31, blockRecord)
	block["block_hash"] = blockHash

	epochRecord := map[string]any{"leaf_count": strictjson.Number(2), "merkle_root": nodeOf(t, leafOf(t, genesisHash), leafOf(t, blockHash))}
	epochHash := recordHash(t, 0x40, epochRecord)
	for _, entry := range []struct {
		at   []string
		path string
	}{
		{[]string{"commitments", "stellar"}, leafOf(t, genesisHash)},
		{[]string{"signing_key_event", "commitments", "stellar"}, leafOf(t, blockHash)},
	} {
		commitment := object(t, proof, entry.at...)
		commitment["network"] = o.stellar
		epoch := object(t, commitment, "epoch")
		epoch["epoch_hash"] = epochHash
		epoch["record"] = map[string]any{"leaf_count": strictjson.Number(2), "merkle_root": epochRecord["merkle_root"]}
		object(t, commitment, "inclusion_proof")["path"] = []any{entry.path}
	}

	proofHash, err := hex.DecodeString(recordHash(t, 0x50, proof))
	if err != nil {
		t.Fatal(err)
	}
	signature := object(t, document, "signature")
	signature["public_key"] = publicKeyOf(o.seed)
	signature["value"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(o.seed[:]), proofHash))
	return mustEncode(t, document)
}

// testEpochHash is the epoch_hash of the proof testProof rebuilds with o,
// the memo its Stellar transaction holds.
func testEpochHash(t *testing.T, o testProofOptions) string {
	t.Helper()
	parsed, err := strictjson.Parse(testProof(t, o))
	if err != nil {
		t.Fatal(err)
	}
	return object(t, parsed, "proof", "commitments", "stellar", "epoch")["epoch_hash"].(string)
}

// mustEncode is v written with JCS.
func mustEncode(t *testing.T, v any) []byte {
	t.Helper()
	encoded, err := jcs.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

// writeTestProof writes the proof testProof rebuilds with o to a file of
// the test's, and gives its path.
func writeTestProof(t *testing.T, o testProofOptions) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proof.json")
	if err := os.WriteFile(path, testProof(t, o), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The rebuild, given the vector's own key, marker and networks, gives the
// vector's own document, value for value, so each value it recomputes is
// computed as the vector's generator computed it. Mutants killed: the
// leaf prefix 0x00 written 0x01; the item_hash prefix 0x11 written 0x13.
// The genesis block's previous_block_hash, which the vector's own networks
// leave unchanged, is held by TestVerify_VerifiedAndInternallyConsistent.
func TestTestProof_RebuildsTheVector(t *testing.T) {
	raw, err := os.ReadFile(vectorcases.Path("positive/item-genesis-witness.proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := strictjson.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := mustEncode(t, parsed)
	got := testProof(t, testProofOptions{seed: vectorSeed, environment: "test", stellar: "testnet", bitcoin: "regtest"})
	if !bytes.Equal(got, want) {
		t.Fatalf("the rebuild differs from the vector:\n got %s\nwant %s", got, want)
	}
}
