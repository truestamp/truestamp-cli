// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	goed25519 "crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier/keyring"
)

// vectorSigner is the private key of the vector seed, 32 octets of 0x42,
// whose public key is VECTOR_KEY (SPEC.md section 9).
var vectorSigner = goed25519.NewKeyFromSeed([]byte(strings.Repeat("B", 32)))

// signerOf is a private key no environment uses, made from seed.
func signerOf(seed string) goed25519.PrivateKey {
	key := testKey(seed)
	return goed25519.NewKeyFromSeed(key[:])
}

func publicOf(private goed25519.PrivateKey) [32]byte {
	return [32]byte(private.Public().(goed25519.PublicKey))
}

// soundProof is P of the sound item proof, which a test edits and signs
// again.
func soundProof(t *testing.T) map[string]any {
	t.Helper()
	return soundDocument(t)["proof"].(map[string]any)
}

// signed returns the served document of proof signed by private, as
// SPEC.md section 6.1 signs: Ed25519 over the 32 octets of H(0x50, proof).
func signed(t *testing.T, proof map[string]any, private goed25519.PrivateKey) []byte {
	t.Helper()
	proofHash, err := registry.RecordHash(registry.ProofHash, proof)
	if err != nil {
		t.Fatal(err)
	}
	public := publicOf(private)
	return encode(t, map[string]any{
		"proof": proof,
		"signature": map[string]any{
			"alg":        Alg,
			"public_key": registry.EncodeB64U(public[:]),
			"value":      registry.EncodeB64U(goed25519.Sign(private, proofHash[:])),
		},
	})
}

// gate runs steps 0 to 6 over data and fails the test on a bare rejection
// or an error.
func gate(t *testing.T, cfg Config, data []byte) *Run {
	t.Helper()
	run, rejection, err := Gate(cfg, data)
	if err != nil || rejection != nil {
		t.Fatalf("Gate: %v, %v", rejection, err)
	}
	return run
}

// gateRows are the rows of ENVIRONMENT, KEY_BINDING and SIGNATURE, as
// rowSummary writes them.
func gateRows(run *Run) string {
	return rowSummary(run.Rows[4:7])
}

// keyringFile writes data to a keyring file of its own and returns its
// path.
func keyringFile(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "keyring.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// keyringEntry writes one key of a keyring.
func keyringEntry(key [32]byte, active bool, sequence int) string {
	return fmt.Sprintf(`{"active": %t, "key_id": %q, "public_key": %q, "sequence": %d}`,
		active, registry.KeyIDOf(key), registry.EncodeB64U(key[:]), sequence)
}

func keyringBytes(entries ...string) []byte {
	return []byte(`{"keys": [` + strings.Join(entries, ", ") + `], "version": "1.0"}`)
}

// The message of environment_mismatch never quotes the marker: a marker
// that is not dev, staging or test, a test marker with a newline among
// them, gets the one sentence that says no Truestamp environment uses it,
// with no hint; a marker of dev, staging or test is named, with SPEC.md
// section 6.4's hint, an offline run with that environment and a keyring
// file; and a proof with no marker is called a production proof, with no
// hint. Mutants killed: the unknown marker's message quoting it with %q
// (the newline's \n shows); the hint added for every marker; the hint
// without "offline"; prod among the known markers (the marker prod is
// named); the staging message's purpose given as development.
func TestEnvironment_TheMessageNeverQuotesTheMarker(t *testing.T) {
	const unknown = "This proof carries an environment marker that no Truestamp environment uses."
	for _, e := range specdata.Environments() {
		for _, marker := range []any{
			"test\n", "prod", "production", "Staging", " dev", "staging ", "", nil,
			strictjson.Number(1), true, []any{"test"}, map[string]any{"environment": "test"},
		} {
			if got := environmentMessage(e, marker, true); got != unknown {
				t.Errorf("E_v %s, marker %#v: %q", e, marker, got)
			}
		}
	}
	cases := []struct{ e, marker, want string }{
		{"prod", "staging", "This proof is marked as issued by Truestamp's staging environment, which is for testing. " +
			"It is not a production proof. To check it against the staging keyring, run again offline with " +
			"--environment staging and --keyring FILE."},
		{"prod", "dev", "This proof is marked as issued by Truestamp's dev environment, which is for development. " +
			"It is not a production proof. To check it against the dev keyring, run again offline with " +
			"--environment dev and --keyring FILE."},
		{"prod", "test", "This proof is marked as issued by Truestamp's test environment, which is for testing. " +
			"It is not a production proof. To check it against the test keyring, run again offline with " +
			"--environment test and --keyring FILE."},
		{"staging", "dev", "This proof is marked as issued by Truestamp's dev environment, and this run checks " +
			"proofs of its staging environment. To check it against the dev keyring, run again offline with " +
			"--environment dev and --keyring FILE."},
		{"test", "staging", "This proof is marked as issued by Truestamp's staging environment, and this run checks " +
			"proofs of its test environment. To check it against the staging keyring, run again offline with " +
			"--environment staging and --keyring FILE."},
	}
	for _, c := range cases {
		if got := environmentMessage(c.e, c.marker, true); got != c.want {
			t.Errorf("E_v %s, marker %s:\n got %q\nwant %q", c.e, c.marker, got, c.want)
		}
	}
	for _, e := range []string{"dev", "staging", "test"} {
		got := environmentMessage(e, nil, false)
		if !strings.Contains(got, "carries no environment marker, so it is a production proof") ||
			!strings.Contains(got, "Truestamp's "+e+" environment") || strings.Contains(got, "--") {
			t.Errorf("E_v %s, no marker: %q", e, got)
		}
	}
}

// ENVIRONMENT's one row, at /environment: a marker equal to E_v passes
// outside prod and no marker passes under prod; anything else fails
// environment_mismatch and ends the run, the row's values holding E_v and
// the declared marker as the proof carries it, whatever JSON value it is,
// null included, written with JCS; the gate's later rows skip and the
// attribution is none. A test marker with a newline fails under test, a
// marker that differs from E_v only in case fails, and under prod an empty
// marker fails, since it is present. Mutants killed: the comparison made
// after trimming whitespace (the test marker with a newline passes); the
// comparison folding case (Test passes under test, STAGING under staging);
// an empty marker read as absent under prod (it passes); the declared value
// left out for a null marker; E_v prod passing any marker that is not a
// string.
func TestEnvironment_Rows(t *testing.T) {
	keyless := func(e string) Config {
		cfg, err := NewConfig(e)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Keyring = KeyringSource{Kind: SourceKeyless}
		return cfg
	}
	cases := []struct {
		e      string
		marker any
		absent bool
		pass   bool
	}{
		{e: "test", marker: "test", pass: true},
		{e: "prod", absent: true, pass: true},
		{e: "test", marker: "test\n"},
		{e: "test", marker: "Test"},
		{e: "staging", marker: "STAGING"},
		{e: "prod", marker: ""},
		{e: "test", absent: true},
		{e: "test", marker: "staging"},
		{e: "staging", marker: "test"},
		{e: "prod", marker: "prod"},
		{e: "prod", marker: "test"},
		{e: "prod", marker: nil},
		{e: "prod", marker: strictjson.Number(0)},
		{e: "prod", marker: false},
		{e: "test", marker: map[string]any{}},
	}
	for _, c := range cases {
		proof := soundProof(t)
		if c.absent {
			delete(proof, "environment")
		} else {
			proof["environment"] = c.marker
		}
		run := gate(t, keyless(c.e), signed(t, proof, vectorSigner))
		row := run.Rows[4]
		if row.Step != "ENVIRONMENT" || row.Part == nil || *row.Part != "/environment" {
			t.Fatalf("%s, %#v: row %+v", c.e, c.marker, row)
		}
		if c.pass {
			if row.Status != StatusPass || run.Ended || len(row.Values) != 1 || row.Values["environment"] != c.e {
				t.Errorf("%s, %#v: %+v", c.e, c.marker, row)
			}
			continue
		}
		declared, hasDeclared := row.Values["declared"]
		if row.Code != specdata.ReasonEnvironmentMismatch || !run.Ended || run.Attribution != AttributionNone ||
			row.Values["environment"] != c.e || hasDeclared == c.absent {
			t.Errorf("%s, %#v: %+v, ended %v, attribution %q", c.e, c.marker, row, run.Ended, run.Attribution)
			continue
		}
		if !c.absent {
			want, _ := jcs.Encode(c.marker)
			if got, err := jcs.Encode(declared); err != nil || string(got) != string(want) {
				t.Errorf("%s, %#v: declared %s, %v", c.e, c.marker, got, err)
			}
		}
		if _, err := jcs.Encode(row.Values); err != nil {
			t.Errorf("%s, %#v: values %v", c.e, c.marker, err)
		}
		if got := gateRows(run); got != "[environment_mismatch /environment fail ENVIRONMENT] [prior_failure null skip KEY_BINDING] [prior_failure null skip SIGNATURE]" {
			t.Errorf("%s, %#v: rows %s", c.e, c.marker, got)
		}
	}
}

// A supplied keyring holding the signing key as inactive beside an active
// key binds it, and SIGNATURE passes with the keyring's attribution.
// Mutant killed: Find limited to active keys.
func TestKeyBinding_AnInactiveKeyBinds(t *testing.T) {
	data := keyringBytes(keyringEntry(testKey("active"), true, 1), keyringEntry(keyring.VectorKey(), false, 0))
	cfg := testConfig(t)
	cfg.Keyring.Path = keyringFile(t, data)
	run := gate(t, cfg, encode(t, soundDocument(t)))
	if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [null null pass SIGNATURE]" {
		t.Fatalf("rows %s", got)
	}
	if run.BoundKey == nil || run.BoundKey.Active || run.BoundKey.PublicKey != keyring.VectorKey() {
		t.Fatalf("bound key %+v", run.BoundKey)
	}
	if run.Attribution != "keyring:"+digestOf(data) {
		t.Fatalf("attribution %q", run.Attribution)
	}
}

// KEY_BINDING compares all 32 octets, so a keyring holding K with one bit
// flipped, at either end of any of its 32 octets (bits 0 and 7, the sign
// bit among them), binds nothing: each of the 64 is key_binding_failed,
// and K itself binds. Mutants killed: Find comparing the first 31 octets;
// Find comparing every bit but the sign bit.
func TestKeyBinding_ComparesAll32Octets(t *testing.T) {
	document := encode(t, soundDocument(t))
	k := keyring.VectorKey()
	cfg := testConfig(t)
	cfg.Keyring.Path = writeKeyring(t, k)
	if run := gate(t, cfg, document); run.BoundKey == nil {
		t.Fatalf("K does not bind: %s", gateRows(run))
	}
	flips := 0
	for octet := range 32 {
		for _, bit := range []uint{0, 7} {
			other := k
			other[octet] ^= 1 << bit
			cfg.Keyring.Path = writeKeyring(t, other)
			run := gate(t, cfg, document)
			if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [key_binding_failed null fail KEY_BINDING] [prior_failure null skip SIGNATURE]" ||
				run.BoundKey != nil || run.Attribution != AttributionNone {
				t.Errorf("octet %d, bit %d: %s", octet, bit, got)
			}
			flips++
		}
	}
	if flips != 64 {
		t.Fatalf("%d flips", flips)
	}
}

// collisionSigners returns two private keys whose public keys share a key
// id, found by a birthday search over keys made from a counter: a key id
// has 32 bits, so a pair turns up after about 2^16 keys.
func collisionSigners(t *testing.T) (goed25519.PrivateKey, goed25519.PrivateKey) {
	t.Helper()
	seen := map[string]goed25519.PrivateKey{}
	var seed [32]byte
	for i := uint64(0); i < 1<<21; i++ {
		binary.BigEndian.PutUint64(seed[:], i)
		private := goed25519.NewKeyFromSeed(seed[:])
		id := registry.KeyIDOf(publicOf(private))
		if other, ok := seen[id]; ok {
			return other, private
		}
		seen[id] = private
	}
	t.Fatal("no key-id collision found")
	return nil, nil
}

// A keyring holding a key whose key id equals K's, with other bytes, binds
// nothing, and the same keyring with K binds: only the full comparison
// binds (SPEC.md section 6.1). Mutant killed: KeyBinding finding the key
// whose key id is K's.
func TestKeyBinding_AKeyIDCollisionPartnerBindsNothing(t *testing.T) {
	signer, partner := collisionSigners(t)
	k, other := publicOf(signer), publicOf(partner)
	if k == other || registry.KeyIDOf(k) != registry.KeyIDOf(other) {
		t.Fatal("not a collision")
	}
	document := signed(t, soundProof(t), signer)
	cfg := testConfig(t)
	cfg.Keyring.Path = writeKeyring(t, testKey("a"), other)
	run := gate(t, cfg, document)
	if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [key_binding_failed null fail KEY_BINDING] [prior_failure null skip SIGNATURE]" {
		t.Fatalf("the collision partner: %s", got)
	}
	if run.Rows[5].Values["key_id"] != registry.KeyIDOf(k) {
		t.Fatalf("values %v", run.Rows[5].Values)
	}
	cfg.Keyring.Path = writeKeyring(t, testKey("a"), k)
	if run := gate(t, cfg, document); run.BoundKey == nil || run.BoundKey.PublicKey != k || run.Rows[6].Status != StatusPass {
		t.Fatalf("K: %s", gateRows(run))
	}
}

// Each keyring that can govern, and the attribution and trust_root it
// gives (SPEC.md sections 8.3 and 10.5): a keyring file, keyring:<d> with d
// the SHA-256 of the file's bytes and its path; keyless mode, unattributed
// with no digest or location; the keyring read from www.truestamp.com,
// truestamp with the digest of the bytes read and its URL; the compiled
// pin, which does not hold K, key_binding_failed with its digest and the
// message to update the verifier or run online; and a failed read from
// www.truestamp.com or from another origin, keyring_unavailable whatever K
// is, with no digest. trust_root after the gate is the one CONFIGURATION
// fixed, whatever fails. Mutants killed: a supplied keyring attributed
// truestamp; the keyring read from www.truestamp.com attributed by its
// digest; a failed online read binding against the keyring it would have
// loaded (K binds); keyring_unavailable's message for another origin naming
// www.truestamp.com; gateFailure clearing trust_root.
func TestKeyBinding_EachKeyringAndItsAttribution(t *testing.T) {
	signer := signerOf("online")
	k := publicOf(signer)
	production := soundProof(t)
	delete(production, "environment")
	prodDocument := signed(t, production, signer)
	online := keyringBytes(keyringEntry(k, true, 0))
	origin := "https://keys.truestamp.example/.well-known/keyring.json"
	read := func(data []byte, err error) KeyringReader {
		return byteReader(func(string) ([]byte, error) { return data, err })
	}
	prod := func(online bool, source KeyringSource, reader KeyringReader) Config {
		cfg, _ := NewConfig("prod")
		cfg.Online, cfg.Keyring, cfg.ReadKeyring = online, source, reader
		return cfg
	}
	file := writeKeyring(t, k)
	cases := []struct {
		name        string
		cfg         Config
		rows        string
		root        TrustRoot
		attribution string
		message     string
	}{
		{"a keyring file", prod(false, KeyringSource{Kind: SourceFile, Path: file}, nil),
			"[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [null null pass SIGNATURE]",
			TrustRoot{Digest: fileDigest(t, file), Kind: KindSuppliedKeyring, Location: file},
			"keyring:" + fileDigest(t, file), "in the supplied keyring"},
		{"keyless", prod(false, KeyringSource{Kind: SourceKeyless}, nil),
			"[null /environment pass ENVIRONMENT] [key_binding_not_performed null skip KEY_BINDING] [null null pass SIGNATURE]",
			TrustRoot{Kind: KindNone}, AttributionUnattributed, "only against the key the proof carries"},
		{"the keyring read from www.truestamp.com", prod(true, KeyringSource{}, read(online, nil)),
			"[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [null null pass SIGNATURE]",
			TrustRoot{Digest: digestOf(online), Kind: KindCurrentKeyring, Location: CurrentKeyringURL},
			AttributionTruestamp, "one of Truestamp's keys"},
		{"the compiled pin", prod(false, KeyringSource{}, nil),
			"[null /environment pass ENVIRONMENT] [key_binding_failed null fail KEY_BINDING] [prior_failure null skip SIGNATURE]",
			TrustRoot{Digest: digestOf(keyring.CompiledPin()), Kind: KindCompiledPin},
			AttributionNone, "update the verifier or run online"},
		{"a failed read from www.truestamp.com", prod(true, KeyringSource{}, read(online, errors.New("HTTP 503"))),
			"[null /environment pass ENVIRONMENT] [keyring_unavailable null fail KEY_BINDING] [prior_failure null skip SIGNATURE]",
			TrustRoot{Kind: KindCurrentKeyring, Location: CurrentKeyringURL},
			AttributionNone, "could not be read from www.truestamp.com"},
		{"a failed read from another origin", prod(true, KeyringSource{Kind: SourceOrigin, URL: origin}, read(online, errors.New("HTTP 503"))),
			"[null /environment pass ENVIRONMENT] [keyring_unavailable null fail KEY_BINDING] [prior_failure null skip SIGNATURE]",
			TrustRoot{Kind: KindSuppliedKeyring, Location: origin},
			AttributionNone, "could not be read from " + origin},
	}
	for _, c := range cases {
		configured, rejection, err := Configure(c.cfg)
		if err != nil || rejection != nil {
			t.Fatalf("%s: %v, %v", c.name, rejection, err)
		}
		run := gate(t, c.cfg, prodDocument)
		if got := gateRows(run); got != c.rows {
			t.Errorf("%s: rows %s", c.name, got)
		}
		if run.TrustRoot != c.root || configured.TrustRoot != c.root {
			t.Errorf("%s: trust_root %+v after the gate, %+v at CONFIGURATION, want %+v", c.name, run.TrustRoot, configured.TrustRoot, c.root)
		}
		if run.Attribution != c.attribution || !strings.Contains(run.Rows[5].Message, c.message) {
			t.Errorf("%s: attribution %q, message %q", c.name, run.Attribution, run.Rows[5].Message)
		}
		if run.Rows[5].Values["key_id"] != registry.KeyIDOf(k) {
			t.Errorf("%s: values %v", c.name, run.Rows[5].Values)
		}
	}
}

// A run under the compiled pin whose key binds is attributed truestamp
// (SPEC.md section 10.5). The pin built into this verifier holds
// Truestamp's production key, which no test can sign with, so the test
// admits a production proof offline under prod, which fixes
// trust_root compiled_pin, and gives the run, in place of the pin, a
// keyring loaded under prod that holds a key only this test uses, the key
// that signed the proof. Mutant killed: compiled_pin left out of the
// truestamp attribution (the run is unattributed).
func TestSignature_TheCompiledPinIsAttributedTruestamp(t *testing.T) {
	signer := signerOf("compiled pin")
	proof := soundProof(t)
	delete(proof, "environment")
	cfg, err := NewConfig("prod")
	if err != nil {
		t.Fatal(err)
	}
	run, rejection, err := Admit(cfg, signed(t, proof, signer))
	if err != nil || rejection != nil {
		t.Fatalf("Admit: %v, %v", rejection, err)
	}
	if run.TrustRoot.Kind != KindCompiledPin {
		t.Fatalf("trust_root %+v", run.TrustRoot)
	}
	pin, err := keyring.Load(keyringBytes(keyringEntry(publicOf(signer), true, 0)), "prod", "")
	if err != nil {
		t.Fatal(err)
	}
	run.Keyring = pin
	Environment(run)
	KeyBinding(run)
	Signature(run)
	if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [null null pass SIGNATURE]" {
		t.Fatalf("rows %s", got)
	}
	if run.Ended || run.Attribution != AttributionTruestamp || run.TrustRoot.Kind != KindCompiledPin {
		t.Fatalf("ended %v, attribution %q, trust_root %+v", run.Ended, run.Attribution, run.TrustRoot)
	}
	if !strings.Contains(run.Rows[5].Message, "one of Truestamp's keys") {
		t.Fatalf("message %q", run.Rows[5].Message)
	}
}

// offCurveKey is 0x02 followed by 31 zero octets: its y, 2, is below p and
// it is not one of SMALL_ORDER_KEYS, so it passes the checks of a key's
// bytes, but no x has x^2 = (y^2 - 1) / (d y^2 + 1), so it is no point of
// the curve (SPEC.md section 6.2, step 4).
var offCurveKey = [32]byte{2}

// A key off the curve passes the load checks under every E_v, which check a
// key's bytes and no more, so a supplied keyring can hold it and bind it;
// SIGNATURE then fails it, named not_on_curve, and the run ends with the
// attribution none. Mutants killed: KEY_BINDING refusing a K off the curve
// (the row is key_binding_failed); the load checks refusing a key off the
// curve (Load fails).
func TestSignature_AKeyOffTheCurveBindsAndFails(t *testing.T) {
	data := keyringBytes(keyringEntry(offCurveKey, true, 0))
	for _, e := range specdata.Environments() {
		if _, err := keyring.Load(data, e, ""); err != nil {
			t.Errorf("E_v %s: %v", e, err)
		}
	}
	document := soundDocument(t)
	proofHash, err := registry.RecordHash(registry.ProofHash, document["proof"].(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	signatureOf(document)["public_key"] = registry.EncodeB64U(offCurveKey[:])
	signatureOf(document)["value"] = registry.EncodeB64U(goed25519.Sign(vectorSigner, proofHash[:]))
	cfg := testConfig(t)
	cfg.Keyring.Path = keyringFile(t, data)
	run := gate(t, cfg, encode(t, document))
	if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [signature_invalid null fail SIGNATURE]" {
		t.Fatalf("rows %s", got)
	}
	if run.BoundKey == nil || run.BoundKey.PublicKey != offCurveKey {
		t.Fatalf("bound key %+v", run.BoundKey)
	}
	if row := run.Rows[6]; row.Values["refused"] != "not_on_curve" || !run.Ended || run.Attribution != AttributionNone {
		t.Fatalf("row %+v, ended %v, attribution %q", row, run.Ended, run.Attribution)
	}
}

// SIGNATURE's one row: on a pass its values hold proof_hash as hex,
// computed from P as parsed, and the run holds it; on a failure, the step
// of SPEC.md section 6.2 that refused as well. Each signature-* proof that
// the case files carry fails, named by its step. Mutants killed: the
// refusal left out of the values; proof_hash computed over the document
// rather than P.
func TestSignature_RowsAndRefusals(t *testing.T) {
	document := soundDocument(t)
	run := gate(t, testConfig(t), encode(t, document))
	want, err := registry.RecordHashHex(registry.ProofHash, document["proof"].(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	row := run.Rows[6]
	if row.Status != StatusPass || row.Values["proof_hash"] != want || hex.EncodeToString(run.ProofHash[:]) != want || len(row.Values) != 1 {
		t.Fatalf("row %+v, proof_hash %x", row, run.ProofHash)
	}
	keyless := testConfig(t)
	keyless.Keyring = KeyringSource{Kind: SourceKeyless}
	for file, refused := range map[string]string{
		"negative/signature-over-m.proof.json":                   "equation",
		"negative/signature-s_plus_l.proof.json":                 "s_not_below_l",
		"negative/signature-s_equal_l.proof.json":                "s_not_below_l",
		"negative/signature-key_not_on_curve-keyless.proof.json": "not_on_curve",
		"negative/mixed_order-k_not_multiple_of_8.proof.json":    "equation",
	} {
		data, err := os.ReadFile(vectorcases.Path(file))
		if err != nil {
			t.Fatal(err)
		}
		run := gate(t, keyless, data)
		row := run.Rows[6]
		if row.Code != specdata.ReasonSignatureInvalid || row.Values["refused"] != refused || !run.Ended ||
			run.Attribution != AttributionNone || len(row.Values["proof_hash"].(string)) != 64 {
			t.Errorf("%s: %+v, attribution %q", file, row, run.Attribution)
		}
	}
}

// A proof_hash that cannot be computed is a fault of the verifier, never a
// pass or a signature_invalid: SIGNATURE fails internal_error, the later
// steps skip and the attribution is none (SPEC.md section 10.5). No parsed
// proof gives one, so the test hands SIGNATURE a P that JCS refuses.
// Mutant killed: the error of the record hash ignored (the zero hash is
// checked, and the row is signature_invalid).
func TestSignature_AFaultIsInternalError(t *testing.T) {
	run := &Run{Proof: map[string]any{"x": make(chan int)}, TrustRoot: TrustRoot{Kind: KindNone}, PublicKey: keyring.VectorKey()}
	Signature(run)
	if len(run.Rows) != 8 || run.Rows[0].Code != specdata.ReasonInternalError || run.Rows[0].Status != StatusFail ||
		run.Rows[0].Step != "SIGNATURE" || run.Rows[1].Code != specdata.ReasonPriorFailure || !run.Ended || run.Attribution != AttributionNone {
		t.Fatalf("rows %s, attribution %q", rowSummary(run.Rows), run.Attribution)
	}
}

// The worked example of SPEC.md section 6.3 through the gate: a test run
// whose keyring holds VECTOR_KEY passes ENVIRONMENT, KEY_BINDING and
// SIGNATURE, and a run whose E_v is not test fails it at ENVIRONMENT; the
// re-indented document with CRLF line ends, its members reordered, a \u
// escape and 1.0 for 1 verifies alike, and a bare rejection gives no run.
// Mutants killed: proof_hash computed under the byte of block_hash (both
// spellings fail SIGNATURE); Gate making steps 4 to 6 on a rejected run.
func TestGate_TheWorkedExample(t *testing.T) {
	data, err := os.ReadFile(vectorcases.Path("worked-example.json"))
	if err != nil {
		t.Fatal(err)
	}
	var example struct {
		Document string `json:"document"`
		K        string `json:"K"`
		V        string `json:"V"`
	}
	if err := json.Unmarshal(data, &example); err != nil {
		t.Fatal(err)
	}
	reformatted := fmt.Sprintf("{\r\n  \"signature\": {\r\n    \"value\": %q,\r\n    \"public_key\": %q,\r\n    \"alg\": \"truestamp-ed25519-sha256-jcs\"\r\n  },\r\n"+
		"  \"proof\": {\r\n    \"version\": 1.0,\r\n    \"type\": \"bl\\u006fck\",\r\n    \"format\": \"truestamp-proof\",\r\n    \"environment\": \"test\"\r\n  }\r\n}\r\n",
		example.V, example.K)
	for _, document := range []string{example.Document, reformatted} {
		run := gate(t, testConfig(t), []byte(document))
		if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [null null pass SIGNATURE]" ||
			run.Rows[6].Values["proof_hash"] != "9f46e27f4a5e1d937189020719959df9d4f7b65948b616d4bde9d187688e21b2" {
			t.Errorf("rows %s, values %v", got, run.Rows[6].Values)
		}
	}
	for _, e := range []string{"prod", "dev", "staging"} {
		cfg, _ := NewConfig(e)
		cfg.Keyring = KeyringSource{Kind: SourceKeyless}
		if run := gate(t, cfg, []byte(example.Document)); run.Rows[4].Code != specdata.ReasonEnvironmentMismatch {
			t.Errorf("E_v %s: %s", e, gateRows(run))
		}
	}
	if run, rejection, err := Gate(testConfig(t), []byte("[]")); run != nil || err != nil || rejection == nil ||
		rejection.SubReason != specdata.SubReasonDocument {
		t.Errorf("a bare rejection through Gate: %v, %v, %v", run, rejection, err)
	}
}
