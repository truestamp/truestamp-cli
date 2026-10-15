// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package ed25519

import (
	goed25519 "crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// corpusEntry is one entry of vectors/corpus/ed25519.json: a public key, a
// message and a signature, whether Verify passes them, and for a failure
// the step of SPEC.md section 6.2 that refuses first.
type corpusEntry struct {
	About     string `json:"about"`
	ID        string `json:"id"`
	Message   string `json:"message"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
	Step      int    `json:"step"`
	Verify    string `json:"verify"`
}

// refusalOfStep is the refusal Verify gives for each step of SPEC.md
// section 6.2 that can refuse.
var refusalOfStep = map[int]error{
	2: ErrYNotBelowP,
	3: ErrSmallOrderKey,
	4: ErrNotOnCurve,
	5: ErrSNotBelowL,
	7: ErrEquation,
}

func readCorpus(t *testing.T) []corpusEntry {
	t.Helper()
	data, err := os.ReadFile(vectorcases.Path("corpus/ed25519.json"))
	if err != nil {
		t.Fatal(err)
	}
	var entries []corpusEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatal(err)
	}
	return entries
}

// Every entry of vectors/corpus/ed25519.json: Verify passes exactly the
// five that pass, the mixed-order key with a challenge that is a multiple
// of 8 among them, and refuses each other entry with the refusal of the
// step the entry names, the key and signature decoded through the
// registry's base64url. Mutants killed: Verify without CheckKeyBytes (the
// identity written with y = p + 1 and the small-order keys of order 1 and
// 4 pass, as the library alone passes them); the point check made after
// the s check (step_4-not_on_curve-s_equal_l is named s_not_below_l);
// onCurve testing for no root by ModSqrt of x2 + 1 (the key not on the
// curve is named equation); s compared with L by > (step_5-s_equal_l is
// named equation); Verify passing every signature the key checks pass.
func TestVerify_TheEd25519Corpus(t *testing.T) {
	entries := readCorpus(t)
	if len(entries) != 28 {
		t.Fatalf("%d entries, want 28", len(entries))
	}
	steps := map[int]int{}
	passes := 0
	for _, entry := range entries {
		key, err := registry.DecodeB64U32(entry.PublicKey)
		if err != nil {
			t.Fatalf("%s: public_key: %v", entry.ID, err)
		}
		signature, err := registry.DecodeB64U64(entry.Signature)
		if err != nil {
			t.Fatalf("%s: signature: %v", entry.ID, err)
		}
		message, err := hex.DecodeString(entry.Message)
		if err != nil || len(message) != 32 {
			t.Fatalf("%s: message: %v", entry.ID, err)
		}
		got := Verify(key, message, signature)
		switch entry.Verify {
		case "pass":
			passes++
			if got != nil {
				t.Errorf("%s: %v, want a pass", entry.ID, got)
			}
		case "fail":
			steps[entry.Step]++
			want, ok := refusalOfStep[entry.Step]
			if !ok {
				t.Fatalf("%s: step %d refuses nothing", entry.ID, entry.Step)
			}
			if !errors.Is(got, want) {
				t.Errorf("%s: %v, want %v (step %d)", entry.ID, got, want, entry.Step)
			}
		default:
			t.Fatalf("%s: verify %q", entry.ID, entry.Verify)
		}
	}
	if passes != 5 || steps[2] != 3 || steps[3] != 10 || steps[4] != 2 || steps[5] != 2 || steps[7] != 6 {
		t.Fatalf("passes %d, refusals by step %v", passes, steps)
	}
}

// workedExample is vectors/worked-example.json, SPEC.md section 6.3.
type workedExample struct {
	A         string         `json:"A"`
	K         string         `json:"K"`
	M         string         `json:"M"`
	S         string         `json:"S"`
	V         string         `json:"V"`
	Document  string         `json:"document"`
	KeyID     string         `json:"key_id"`
	Proof     map[string]any `json:"proof"`
	ProofHash string         `json:"proof_hash"`
	SK        string         `json:"sk"`
}

func mustHex(t *testing.T, text string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(text)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

// The worked example of SPEC.md section 6.3: from its proof, read through
// the strict reader, M is its JCS and proof_hash is H(0x50, proof); the
// seed's key is A, written as K, with its key id; the seed's signature
// over proof_hash's 32 octets is S, written as V; the served document is
// JCS of the envelope; and Verify passes S under A over proof_hash and
// refuses it, at step 7, over every other message the section names: M
// itself, SHA-256(M), the hex of proof_hash, SHA-256(0x31 || M),
// SHA-256(0x62 || M), and proof_hash with the environment changed to dev or
// removed. Mutant killed: Verify passing every signature the key checks
// pass (each other message passes).
func TestVerify_TheWorkedExample(t *testing.T) {
	data, err := os.ReadFile(vectorcases.Path("worked-example.json"))
	if err != nil {
		t.Fatal(err)
	}
	var example workedExample
	if err := json.Unmarshal(data, &example); err != nil {
		t.Fatal(err)
	}
	parsed, err := strictjson.Parse([]byte(example.M))
	if err != nil {
		t.Fatal(err)
	}
	proof := parsed.(map[string]any)
	if len(proof) != len(example.Proof) || proof["environment"] != example.Proof["environment"] {
		t.Fatalf("M reads as %v, the example's proof is %v", proof, example.Proof)
	}
	m, err := jcs.Encode(proof)
	if err != nil || string(m) != example.M || len(m) != 76 {
		t.Fatalf("JCS(proof) = %s, %v", m, err)
	}
	proofHash, err := registry.RecordHash(registry.ProofHash, proof)
	if err != nil || hex.EncodeToString(proofHash[:]) != example.ProofHash {
		t.Fatalf("proof_hash = %x, %v", proofHash, err)
	}

	private := goed25519.NewKeyFromSeed(mustHex(t, example.SK))
	key := [32]byte(private.Public().(goed25519.PublicKey))
	if hex.EncodeToString(key[:]) != example.A || registry.EncodeB64U(key[:]) != example.K || registry.KeyIDOf(key) != example.KeyID {
		t.Fatalf("A = %x, K %s, key id %s", key, registry.EncodeB64U(key[:]), registry.KeyIDOf(key))
	}
	signature := [64]byte(goed25519.Sign(private, proofHash[:]))
	if hex.EncodeToString(signature[:]) != example.S || registry.EncodeB64U(signature[:]) != example.V {
		t.Fatalf("S = %x", signature)
	}
	envelope := map[string]any{
		"proof":     proof,
		"signature": map[string]any{"alg": "truestamp-ed25519-sha256-jcs", "public_key": example.K, "value": example.V},
	}
	if served, err := jcs.Encode(envelope); err != nil || string(served) != example.Document || len(served) != 293 {
		t.Fatalf("JCS(envelope) = %s, %v", served, err)
	}

	if err := Verify(key, proofHash[:], signature); err != nil {
		t.Fatalf("Verify over proof_hash: %v", err)
	}
	prefixed := func(b byte) []byte {
		sum := sha256.Sum256(append([]byte{b}, m...))
		return sum[:]
	}
	shaM := sha256.Sum256(m)
	others := map[string][]byte{
		"M":                     m,
		"SHA-256(M)":            shaM[:],
		"the hex of proof_hash": []byte(example.ProofHash),
		"SHA-256(0x31 || M)":    prefixed(0x31),
		"SHA-256(0x62 || M)":    prefixed(0x62),
	}
	for _, environment := range []string{"dev", ""} {
		changed := map[string]any{}
		for name, value := range proof {
			changed[name] = value
		}
		if environment == "" {
			delete(changed, "environment")
		} else {
			changed["environment"] = environment
		}
		sum, err := registry.RecordHash(registry.ProofHash, changed)
		if err != nil {
			t.Fatal(err)
		}
		others["proof_hash with the environment "+environment] = sum[:]
	}
	for name, message := range others {
		if err := Verify(key, message, signature); !errors.Is(err, ErrEquation) {
			t.Errorf("Verify over %s: %v, want equation", name, err)
		}
	}
}

// L and d are RFC 8032's: L = 2^252 + 27742317777372353535851937790883648493
// and d = -121665/121666 mod p, written as RFC 8032 section 5.1 and the
// reference write them. Mutants killed: L built with 2^253; d built with
// +121665.
func TestConstants_LAndD(t *testing.T) {
	if got := l.Text(16); got != "1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3ed" {
		t.Errorf("L = %s", got)
	}
	if got := d.String(); got != "37095705934669439343138083508754565189542113879843219016388785533085940283555" {
		t.Errorf("d = %s", got)
	}
}
