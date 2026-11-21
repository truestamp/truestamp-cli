// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier/keyring"
)

// testConfig is the configuration of the conformance cases that use the
// test keyring: E_v test, keyring.test.json supplied, offline.
func testConfig(t *testing.T) Config {
	t.Helper()
	cfg, err := NewConfig("test")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Keyring = KeyringSource{Kind: SourceFile, Path: vectorcases.Path("keyring.test.json")}
	return cfg
}

// soundDocument is the parsed envelope of the sound item proof, which a
// test edits and writes back with JCS.
func soundDocument(t *testing.T) map[string]any {
	t.Helper()
	data, err := strictjson.ReadFile(vectorcases.Path("positive/item.proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := strictjson.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return value.(map[string]any)
}

func signatureOf(document map[string]any) map[string]any {
	return document["signature"].(map[string]any)
}

func encode(t *testing.T, value any) []byte {
	t.Helper()
	data, err := jcs.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// admitRejection runs steps 0 to 3 over data under testConfig and returns
// the bare rejection they give, failing when they give none.
func admitRejection(t *testing.T, data []byte) *Rejection {
	t.Helper()
	run, rejection, err := Admit(testConfig(t), data)
	if err != nil || rejection == nil {
		t.Fatalf("Admit = %v, %v, %v; want a bare rejection", run, rejection, err)
	}
	return rejection
}

func rowSummary(rows []Row) string {
	var parts []string
	for _, row := range rows {
		code, part, status, step := row.Reduce()
		text := func(s *string) string {
			if s == nil {
				return "null"
			}
			return *s
		}
		parts = append(parts, fmt.Sprintf("[%s %s %s %s]", text(code), text(part), status, step))
	}
	return strings.Join(parts, " ")
}

// The sound proof passes steps 0 to 3, each with one pass row, part null,
// and the run holds P, K and V; every row's values are values JCS writes.
// E_v stays the run's own when the proof's environment marker names
// another environment (SPEC.md section 6.4). Mutants killed: Envelope not
// setting the run's proof; the ENVELOPE row's key id computed from V's
// first 32 octets; E_v taken from the proof's environment marker after
// ENVELOPE.
func TestAdmit_TheSoundProofPassesStepsZeroToThree(t *testing.T) {
	document := soundDocument(t)
	run, rejection, err := Admit(testConfig(t), encode(t, document))
	if err != nil || rejection != nil {
		t.Fatalf("Admit: %v, %v", rejection, err)
	}
	if got := rowSummary(run.Rows); got != "[null null pass CONFIGURATION] [null null pass READ] [null null pass PARSE] [null null pass ENVELOPE]" {
		t.Fatalf("rows = %s", got)
	}
	for _, row := range run.Rows {
		if _, err := jcs.Encode(row.Values); err != nil {
			t.Errorf("%s values %v: %v", row.Step, row.Values, err)
		}
	}
	key, _ := registry.DecodeB64U32(signatureOf(document)["public_key"].(string))
	value, _ := registry.DecodeB64U64(signatureOf(document)["value"].(string))
	if run.PublicKey != key || run.Signature != value || run.Proof == nil {
		t.Fatalf("the run holds K %x, V %x, P %v", run.PublicKey, run.Signature, run.Proof != nil)
	}
	if run.Rows[3].Values["key_id"] != registry.KeyIDOf(key) || run.Proof["type"] != "item" {
		t.Fatalf("ENVELOPE values = %v", run.Rows[3].Values)
	}
	if run.TrustRoot.Kind != KindSuppliedKeyring || run.Keyring == nil || len(run.Keyring.Keys) != 1 {
		t.Fatalf("trust root %+v, keyring %+v", run.TrustRoot, run.Keyring)
	}

	marked := soundDocument(t)
	marked["proof"].(map[string]any)["environment"] = "staging"
	run, rejection, err = Admit(testConfig(t), encode(t, marked))
	if err != nil || rejection != nil {
		t.Fatalf("Admit, the marker staging: %v, %v", rejection, err)
	}
	if run.Config.Environment != "test" || run.Rows[0].Values["environment"] != "test" {
		t.Fatalf("E_v = %s after ENVELOPE, CONFIGURATION values %v", run.Config.Environment, run.Rows[0].Values)
	}
}

// Each check of ENVELOPE fails with its own sub_reason, a newline on K or V
// among them, and the message is the one SPEC.md fixes, which quotes no
// member name the document holds. A member renamed, so the count of
// members is unchanged, fails the check of the object that holds it.
// Mutants killed: Envelope trimming a final newline from K (the newline
// case passes); the signature's member count not checked (key_id beside
// the three passes); alg compared ignoring case; each of the document's
// proof and the signature's alg, public_key and value not required by
// name, applied as (false && !hasProof) and likewise (the rename names
// proof, alg, public_key or value).
func TestEnvelope_EachCheckAndItsSubReason(t *testing.T) {
	cases := []struct {
		name string
		edit func(map[string]any) any
		want specdata.SubReason
	}{
		{"a top-level array", func(d map[string]any) any { return []any{d} }, specdata.SubReasonDocument},
		{"a signatures array", func(d map[string]any) any {
			d["signatures"] = []any{d["signature"]}
			delete(d, "signature")
			return d
		}, specdata.SubReasonDocument},
		{"a third member", func(d map[string]any) any { d["proof_hash"] = "x"; return d }, specdata.SubReasonDocument},
		{"no proof", func(d map[string]any) any { delete(d, "proof"); return d }, specdata.SubReasonDocument},
		{"a string", func(d map[string]any) any { return "proof" }, specdata.SubReasonDocument},
		{"proof null", func(d map[string]any) any { d["proof"] = nil; return d }, specdata.SubReasonProof},
		{"proof an array", func(d map[string]any) any { d["proof"] = []any{}; return d }, specdata.SubReasonProof},
		{"signature an array", func(d map[string]any) any { d["signature"] = []any{}; return d }, specdata.SubReasonSignature},
		{"signature with key_id", func(d map[string]any) any { signatureOf(d)["key_id"] = "fbd680e8"; return d }, specdata.SubReasonSignature},
		{"signature without alg", func(d map[string]any) any { delete(signatureOf(d), "alg"); return d }, specdata.SubReasonSignature},
		{"proof renamed", func(d map[string]any) any { return rename(d, "proof") }, specdata.SubReasonDocument},
		{"alg renamed", func(d map[string]any) any { rename(signatureOf(d), "alg"); return d }, specdata.SubReasonSignature},
		{"public_key renamed", func(d map[string]any) any { rename(signatureOf(d), "public_key"); return d }, specdata.SubReasonSignature},
		{"value renamed", func(d map[string]any) any { rename(signatureOf(d), "value"); return d }, specdata.SubReasonSignature},
		{"alg in capitals", func(d map[string]any) any { signatureOf(d)["alg"] = strings.ToUpper(Alg); return d }, specdata.SubReasonAlg},
		{"alg a number", func(d map[string]any) any { signatureOf(d)["alg"] = strictjson.Number(1); return d }, specdata.SubReasonAlg},
		{"a newline on K", func(d map[string]any) any {
			signatureOf(d)["public_key"] = signatureOf(d)["public_key"].(string) + "\n"
			return d
		}, specdata.SubReasonPublicKey},
		{"K padded", func(d map[string]any) any {
			signatureOf(d)["public_key"] = signatureOf(d)["public_key"].(string) + "="
			return d
		}, specdata.SubReasonPublicKey},
		{"K a number", func(d map[string]any) any { signatureOf(d)["public_key"] = strictjson.Number(0); return d }, specdata.SubReasonPublicKey},
		{"K small-order", func(d map[string]any) any {
			key := specdata.SmallOrderKeys()[9].Key
			signatureOf(d)["public_key"] = registry.EncodeB64U(key[:])
			return d
		}, specdata.SubReasonPublicKey},
		{"a newline on V", func(d map[string]any) any {
			signatureOf(d)["value"] = signatureOf(d)["value"].(string) + "\n"
			return d
		}, specdata.SubReasonValue},
		{"V padded", func(d map[string]any) any {
			signatureOf(d)["value"] = signatureOf(d)["value"].(string) + "=="
			return d
		}, specdata.SubReasonValue},
		{"V a bool", func(d map[string]any) any { signatureOf(d)["value"] = true; return d }, specdata.SubReasonValue},
	}
	for _, c := range cases {
		rejection := admitRejection(t, encode(t, c.edit(soundDocument(t))))
		if rejection.Code != specdata.ReasonInvalidEnvelope || rejection.SubReason != c.want {
			t.Errorf("%s: %s (%s), want invalid_envelope (%s)", c.name, rejection.Code, rejection.SubReason, c.want)
		}
		if rejection.Message != "This file is not a Truestamp proof." {
			t.Errorf("%s: message %q", c.name, rejection.Message)
		}
	}
}

// rename moves object's member name to x, keeping the count of members.
func rename(object map[string]any, name string) map[string]any {
	object["x"] = object[name]
	delete(object, name)
	return object
}

// The first failing check names the rejection, in SPEC.md's order, and a
// member's name never reaches the message. Mutants killed: public_key
// checked after value (a document whose K and V both fail names value);
// alg checked after public_key; the signature checked before the proof (a
// document whose proof and signature are both null names signature).
func TestEnvelope_TheFirstFailingCheckNamesIt(t *testing.T) {
	if r := admitRejection(t, encode(t, map[string]any{"proof": nil, "signature": nil})); r.SubReason != specdata.SubReasonProof {
		t.Errorf("proof and signature both null: %s", r.SubReason)
	}
	both := soundDocument(t)
	signatureOf(both)["public_key"] = "x"
	signatureOf(both)["value"] = "y"
	if r := admitRejection(t, encode(t, both)); r.SubReason != specdata.SubReasonPublicKey {
		t.Errorf("K and V both bad: %s", r.SubReason)
	}
	algAndKey := soundDocument(t)
	signatureOf(algAndKey)["alg"] = "ed25519"
	signatureOf(algAndKey)["public_key"] = "x"
	if r := admitRejection(t, encode(t, algAndKey)); r.SubReason != specdata.SubReasonAlg {
		t.Errorf("alg and K both bad: %s", r.SubReason)
	}
	named := soundDocument(t)
	named["<script>a member named to be echoed</script>"] = true
	r := admitRejection(t, encode(t, named))
	if r.SubReason != specdata.SubReasonDocument || strings.Contains(r.Error(), "script") || strings.Contains(r.Error(), "echoed") {
		t.Errorf("rejection %v", r)
	}
	if got := EnvelopeChecks(); len(got) != 6 || got[0] != specdata.SubReasonDocument || got[5] != specdata.SubReasonValue {
		t.Errorf("EnvelopeChecks = %v", got)
	}
}

// READ and PARSE give the strict reader's rejections, and READ's row
// counts the octets. Mutant killed: Read without J1 (the oversized file is
// parsed and passes).
func TestAdmit_ReadAndParse(t *testing.T) {
	sound := encode(t, soundDocument(t))
	padded := append(append([]byte{}, sound...), []byte(strings.Repeat(" ", strictjson.MaxReadOctets+1-len(sound)))...)
	if r := admitRejection(t, padded); r.Code != specdata.ReasonInputTooLarge || r.SubReason != "" {
		t.Errorf("one octet over the limit: %v", r)
	}
	atLimit := padded[:strictjson.MaxReadOctets]
	run, rejection, err := Admit(testConfig(t), atLimit)
	if err != nil || rejection != nil || run.Rows[1].Values["octets"] != strictjson.Number(strictjson.MaxReadOctets) {
		t.Errorf("at the limit: %v, %v", rejection, err)
	}
	for text, sub := range map[string]specdata.SubReason{
		`{"a": 1, "a": 2}`:   specdata.SubReasonDuplicateName,
		``:                   specdata.SubReasonSyntax,
		"\ufeff{}":           specdata.SubReasonSyntax,
		`[9007199254740992]`: specdata.SubReasonNumber,
	} {
		if r := admitRejection(t, []byte(text)); r.Code != specdata.ReasonMalformedJSON || r.SubReason != sub {
			t.Errorf("%q: %v, want malformed_json (%s)", text, r, sub)
		}
	}
}

// AdmitFile reads the proof only after CONFIGURATION: a configuration that
// fails is rejected even when the proof cannot be read, and a proof that
// cannot be read is an error, not a rejection. A proof file over the limit
// is input_too_large. Mutant killed: AdmitFile reading the file before
// Configure (the missing proof under a failing configuration is an error).
func TestAdmitFile_ReadsTheProofAfterConfiguration(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.proof.json")
	staging, err := NewConfig("staging")
	if err != nil {
		t.Fatal(err)
	}
	if _, rejection, err := AdmitFile(staging, missing); err != nil || rejection == nil || rejection.Code != specdata.ReasonConfigurationError {
		t.Fatalf("a failing configuration and a missing proof: %v, %v", rejection, err)
	}
	if _, rejection, err := AdmitFile(testConfig(t), missing); !errors.Is(err, strictjson.ErrUnreadable) || rejection != nil {
		t.Fatalf("a missing proof: %v, %v", rejection, err)
	}
	large := filepath.Join(t.TempDir(), "large.proof.json")
	if err := os.WriteFile(large, []byte(strings.Repeat(" ", strictjson.MaxReadOctets+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, rejection, err := AdmitFile(testConfig(t), large); err != nil || rejection == nil || rejection.Code != specdata.ReasonInputTooLarge {
		t.Fatalf("a large proof: %v, %v", rejection, err)
	}
	run, rejection, err := AdmitFile(testConfig(t), vectorcases.Path("positive/item.proof.json"))
	if err != nil || rejection != nil || len(run.Rows) != 4 {
		t.Fatalf("the sound proof: %v, %v", rejection, err)
	}
}

// An online AdmitFile confirms the sources at CONFIGURATION, before it
// reads the proof (SPEC.md section 10.2, step 0): a compiled-in default
// whose identity contradicts its pin, under prod, is configuration_error,
// and no error, when the proof's path names no file; and a proof that an
// identity reader writes during its read is read after it and admitted,
// with the four rows of steps 0 to 3. Mutants killed: the sources
// confirmed after the file is read (the missing proof gives an error where
// step 0 gives configuration_error); the file opened before
// CONFIGURATION, its error kept for after it (the proof written during the
// reads is not found).
func TestAdmitFile_ConfirmsTheSourcesBeforeTheFileIsRead(t *testing.T) {
	mismatched := func(s Source) (string, error) {
		if s.URL == "https://horizon.stellar.org" {
			return testnetPassphrase, nil
		}
		return honestIdentities(s)
	}
	missing := filepath.Join(t.TempDir(), "missing.proof.json")
	if _, rejection, err := AdmitFile(online(t, "prod", mismatched), missing); err != nil || rejection == nil ||
		rejection.Code != specdata.ReasonConfigurationError {
		t.Fatalf("a mismatched default and a missing proof: %v, %v", rejection, err)
	}
	data, err := os.ReadFile(vectorcases.Path("positive/item.proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	late := filepath.Join(t.TempDir(), "late.proof.json")
	var once sync.Once
	writer := func(s Source) (string, error) {
		once.Do(func() {
			if err := os.WriteFile(late, data, 0o600); err != nil {
				t.Error(err)
			}
		})
		return honestIdentities(s)
	}
	if run, rejection, err := AdmitFile(online(t, "prod", writer), late); err != nil || rejection != nil || len(run.Rows) != 4 {
		t.Fatalf("a proof written during the identity reads: %v, %v", rejection, err)
	}
}

// writeKeyring writes a keyring of keys to a file of its own and returns
// its path.
func writeKeyring(t *testing.T, keys ...[32]byte) string {
	t.Helper()
	var entries []string
	for i, key := range keys {
		entries = append(entries, fmt.Sprintf(`{"active": true, "key_id": %q, "public_key": %q, "sequence": %d}`,
			registry.KeyIDOf(key), registry.EncodeB64U(key[:]), i))
	}
	path := filepath.Join(t.TempDir(), "keyring.json")
	if err := os.WriteFile(path, []byte(`{"keys": [`+strings.Join(entries, ", ")+`], "version": "1.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// testKey is a key no environment uses, made from seed.
func testKey(seed string) [32]byte { return sha256.Sum256([]byte("verifier test key " + seed)) }

// digestOf is the lowercase hex SHA-256 of data, made here apart from
// keyring.Digest: trust_root's digest d of a keyring read as data.
func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// fileDigest is [digestOf] the bytes of the file at path.
func fileDigest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return digestOf(data)
}

func configure(t *testing.T, cfg Config) (*Run, *Rejection) {
	t.Helper()
	run, rejection, err := Configure(cfg)
	if err != nil {
		t.Fatalf("Configure: %v", err)
	}
	return run, rejection
}

// A run under another environment than prod with no keyring is
// configuration_error, and its message names every way to supply one that
// the run could take: online, an explicit --base-url too, whose origin's
// keyring is a supplied keyring; offline, not, since no keyring is read.
// Mutant killed: the message offering --base-url offline, or leaving it out
// online (the dev server run found an online `--environment dev` told only
// of --keyring, a configured keyring and --no-keyring).
func TestConfigure_ANonProductionRunWithNoKeyringNamesEachWay(t *testing.T) {
	for _, online := range []bool{true, false} {
		cfg, err := NewConfig("dev")
		if err != nil {
			t.Fatal(err)
		}
		cfg.Online = online
		_, rejection := configure(t, cfg)
		if rejection == nil || rejection.Code != specdata.ReasonConfigurationError {
			t.Fatalf("online %v: %+v, want configuration_error", online, rejection)
		}
		for _, way := range []string{"--keyring FILE", "a keyring configured for dev", "--no-keyring"} {
			if !strings.Contains(rejection.Message, way) {
				t.Errorf("online %v: the message %q does not name %s", online, rejection.Message, way)
			}
		}
		if got := strings.Contains(rejection.Message, "--base-url"); got != online {
			t.Errorf("online %v: the message names --base-url: %v, want %v (%q)", online, got, online, rejection.Message)
		}
	}
}

// A keyring file is loaded under the run's E_v, so under prod it is a
// production keyring: one holding a non-production key is
// configuration_error, as is one that fails any load check or is over
// MAX_READ_OCTETS, while a file that cannot be read at all is an error (exit
// code 1), not a verification result. Mutants killed: a keyring file
// loaded under test whatever E_v is (the non-production key binds under
// prod); a keyring file over the limit given as input_too_large, the
// document's rejection; that failure built without the reader's rejection
// (its message no longer says the keyring is larger than the limit); the
// keyring file's digest left out of trust_root.
func TestConfigure_AKeyringFile(t *testing.T) {
	prod, err := NewConfig("prod")
	if err != nil {
		t.Fatal(err)
	}
	nonproduction := keyring.NonproductionKeys()[0]
	prod.Keyring = KeyringSource{Kind: SourceFile, Path: writeKeyring(t, testKey("a"), nonproduction)}
	if _, rejection := configure(t, prod); rejection == nil || rejection.Code != specdata.ReasonConfigurationError ||
		!strings.Contains(rejection.Message, "non-production") {
		t.Errorf("a non-production key under prod: %v", rejection)
	}

	prod.Keyring.Path = writeKeyring(t, testKey("a"))
	run, rejection := configure(t, prod)
	if rejection != nil || run.TrustRoot != (TrustRoot{Digest: fileDigest(t, prod.Keyring.Path), Kind: KindSuppliedKeyring, Location: prod.Keyring.Path}) {
		t.Fatalf("a sound keyring under prod: %v, %+v", rejection, run)
	}

	large := filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(large, []byte(strings.Repeat(" ", strictjson.MaxReadOctets+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	prod.Keyring.Path = large
	if _, rejection := configure(t, prod); rejection == nil || rejection.Code != specdata.ReasonConfigurationError ||
		!strings.Contains(rejection.Message, fmt.Sprintf("larger than %d octets", strictjson.MaxReadOctets)) {
		t.Errorf("a keyring over MAX_READ_OCTETS: %v", rejection)
	}

	for _, unreadable := range []string{filepath.Join(t.TempDir(), "missing.json"), t.TempDir()} {
		prod.Keyring.Path = unreadable
		if run, rejection, err := Configure(prod); !errors.Is(err, strictjson.ErrUnreadable) || rejection != nil || run != nil {
			t.Errorf("%s: %v, %v, %v; want an unreadable input", unreadable, run, rejection, err)
		}
	}
}

// Without a keyring source and outside keyless mode, a run under a
// non-production E_v is configuration_error, offline and online, while
// keyless mode is not; under prod an offline run is governed by the
// compiled pin. Mutant killed: a non-production run without a keyring
// falling back to the compiled pin.
func TestConfigure_WithoutAKeyringSource(t *testing.T) {
	for _, environment := range []string{"dev", "staging", "test"} {
		for _, online := range []bool{false, true} {
			cfg, _ := NewConfig(environment)
			cfg.Online = online
			cfg = withRegtestSource(cfg)
			cfg.ReadKeyring = byteReader(func(string) ([]byte, error) { t.Fatal("a keyring was read"); return nil, nil })
			if _, rejection := configure(t, cfg); rejection == nil || rejection.Code != specdata.ReasonConfigurationError {
				t.Errorf("%s, online %v: %v", environment, online, rejection)
			}
			cfg.Keyring = KeyringSource{Kind: SourceKeyless}
			if run, rejection := configure(t, cfg); rejection != nil || run.TrustRoot.Kind != KindNone || run.Keyring != nil {
				t.Errorf("%s keyless: %v", environment, rejection)
			}
		}
	}
	prod, _ := NewConfig("prod")
	run, rejection := configure(t, prod)
	if rejection != nil || run.TrustRoot.Kind != KindCompiledPin || run.Keyring == nil || run.KeyringUnavailable {
		t.Fatalf("prod offline: %v, %+v", rejection, run)
	}
	if got := run.Rows[0].Values; got["environment"] != "prod" || got["trust_root"] != "compiled_pin" {
		t.Errorf("CONFIGURATION values = %v", got)
	}
}

// Online under prod with no supplied keyring, the keyring is read from
// www.truestamp.com through the run's reader; a read that fails for any
// reason (the reader's error, no reader, bytes over MAX_READ_OCTETS or not
// JSON, a failed load check) is no rejection but leaves no keyring to bind.
// The keyring read is loaded under the run's E_v, prod, so one holding a
// key of spec/keys.nonproduction.json binds none. No request leaves the
// test: the shared client's refusals do not move. Mutants killed: a failed
// online read given as configuration_error; a failed online read falling
// back to the compiled pin (the run holds a keyring); the online keyring
// loaded under a fixed "staging" (the non-production key binds); the
// digest of the bytes read left out of trust_root.
func TestConfigure_TheOnlineKeyringRead(t *testing.T) {
	before := refusals.Load()
	good, err := os.ReadFile(writeKeyring(t, testKey("a")))
	if err != nil {
		t.Fatal(err)
	}
	vector, err := os.ReadFile(vectorcases.Path("keyring.test.json"))
	if err != nil {
		t.Fatal(err)
	}
	nonproduction, err := os.ReadFile(writeKeyring(t, nonproductionKey(t)))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		reader func(string) ([]byte, error)
		bound  bool
	}{
		{"a sound keyring", func(string) ([]byte, error) { return good, nil }, true},
		{"the reader fails", func(string) ([]byte, error) { return nil, errors.New("HTTP 503") }, false},
		{"no reader", nil, false},
		{"over MAX_READ_OCTETS", func(string) ([]byte, error) {
			return append(append([]byte{}, good...), []byte(strings.Repeat(" ", strictjson.MaxReadOctets))...), nil
		}, false},
		{"a challenge page", func(string) ([]byte, error) { return []byte("<html>Just a moment...</html>"), nil }, false},
		{"VECTOR_KEY under prod", func(string) ([]byte, error) { return vector, nil }, false},
		{"a non-production key under prod", func(string) ([]byte, error) { return nonproduction, nil }, false},
	}
	for _, c := range cases {
		var read []string
		cfg, _ := NewConfig("prod")
		cfg.Online = true
		if c.reader != nil {
			reader := c.reader
			cfg.ReadKeyring = byteReader(func(url string) ([]byte, error) { read = append(read, url); return reader(url) })
		}
		run, rejection := configure(t, cfg)
		if rejection != nil {
			t.Errorf("%s: %v", c.name, rejection)
			continue
		}
		if c.reader != nil && (len(read) != 1 || read[0] != CurrentKeyringURL) {
			t.Errorf("%s: read %q", c.name, read)
		}
		want := TrustRoot{Kind: KindCurrentKeyring, Location: CurrentKeyringURL}
		if c.bound {
			want.Digest = digestOf(good)
		}
		if run.TrustRoot != want || (run.Keyring != nil) != c.bound || run.KeyringUnavailable == c.bound {
			t.Errorf("%s: %+v, keyring %v, unavailable %v", c.name, run.TrustRoot, run.Keyring != nil, run.KeyringUnavailable)
		}
	}
	if got := refusals.Load() - before; got != 0 {
		t.Fatalf("%d requests reached the shared client", got)
	}
}

// nonproductionKey is a key of spec/keys.nonproduction.json other than
// VECTOR_KEY, which check 6 refuses under every E_v but test.
func nonproductionKey(t *testing.T) [32]byte {
	t.Helper()
	for _, key := range keyring.NonproductionKeys() {
		if key != keyring.VectorKey() {
			return key
		}
	}
	t.Fatal("spec/keys.nonproduction.json holds no key but VECTOR_KEY")
	return [32]byte{}
}

// Check 7 is a production keyring's alone: a keyring holding a key of
// spec/keys.nonproduction.json loads under dev and under staging, as a
// keyring file, as a configured keyring and read online, and is refused
// under prod (configuration_error for a file, no keyring for a read); a
// keyring read online under test holding VECTOR_KEY loads. Mutants killed:
// check 7 made under every E_v but test (the dev and staging keyrings are
// refused); the online keyring loaded under a fixed "prod" (the dev and
// staging reads bind nothing); the online keyring loaded under a fixed
// "staging" (the test read of VECTOR_KEY binds nothing).
func TestConfigure_ANonproductionKeyLoadsOutsideProd(t *testing.T) {
	key := nonproductionKey(t)
	path := writeKeyring(t, testKey("a"), key)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const url = "https://staging.truestamp.example/.well-known/keyring.json"
	sources := map[string]KeyringSource{
		"a keyring file":       {Kind: SourceFile, Path: path},
		"a configured keyring": {Kind: SourceConfigured, Path: path},
		"an origin's keyring":  {Kind: SourceOrigin, URL: url},
	}
	for _, environment := range []string{"dev", "staging", "prod"} {
		for name, source := range sources {
			cfg, _ := NewConfig(environment)
			cfg.Keyring, cfg.Online = source, source.Kind == SourceOrigin
			cfg = withRegtestSource(cfg)
			cfg.ReadKeyring = byteReader(func(string) ([]byte, error) { return data, nil })
			run, rejection := configure(t, cfg)
			switch {
			case environment != "prod":
				if rejection != nil || run.Keyring == nil || run.KeyringUnavailable {
					t.Errorf("%s under %s: %v, %+v", name, environment, rejection, run)
				} else if _, ok := run.Keyring.Find(key); !ok {
					t.Errorf("%s under %s: the key is not in the keyring", name, environment)
				}
			case source.Kind == SourceOrigin:
				if rejection != nil || run.Keyring != nil || !run.KeyringUnavailable {
					t.Errorf("%s under prod: %v, %+v", name, rejection, run)
				}
			default:
				if rejection == nil || rejection.Code != specdata.ReasonConfigurationError ||
					!strings.Contains(rejection.Message, "non-production") {
					t.Errorf("%s under prod: %v", name, rejection)
				}
			}
		}
	}

	vector, err := os.ReadFile(vectorcases.Path("keyring.test.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := NewConfig("test")
	cfg.Keyring, cfg.Online = KeyringSource{Kind: SourceOrigin, URL: url}, true
	cfg = withRegtestSource(cfg)
	cfg.ReadKeyring = byteReader(func(string) ([]byte, error) { return vector, nil })
	run, rejection := configure(t, cfg)
	if rejection != nil || run.Keyring == nil || run.KeyringUnavailable {
		t.Fatalf("an origin's VECTOR_KEY under test: %v, %+v", rejection, run)
	}
	if _, ok := run.Keyring.Find(keyring.VectorKey()); !ok {
		t.Fatal("VECTOR_KEY is not in the keyring read under test")
	}
}

// Config refusals: an environment that is not an E_v, a pin that is no
// network of its chain, a keyring source without its file, and an origin's
// keyring in an offline run. Mutant killed: Validate not checking the pins.
func TestConfig_Refusals(t *testing.T) {
	if _, err := NewConfig("production"); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("NewConfig(production): %v", err)
	}
	edits := map[string]func(*Config){
		"E_v production":          func(c *Config) { c.Environment = "production" },
		"a Bitcoin pin of public": func(c *Config) { c.Networks.Commitment.Bitcoin = "public" },
		"an empty Stellar pin":    func(c *Config) { c.Networks.Entropy.Stellar = "" },
		"a file source, no file":  func(c *Config) { c.Keyring = KeyringSource{Kind: SourceFile} },
		"an origin offline":       func(c *Config) { c.Keyring = KeyringSource{Kind: SourceOrigin, URL: "https://x.test/k.json"} },
		"an unknown source":       func(c *Config) { c.Keyring = KeyringSource{Kind: SourceOrigin + 1} },
	}
	for name, edit := range edits {
		cfg, _ := NewConfig("test")
		edit(&cfg)
		if _, rejection, err := Configure(cfg); !errors.Is(err, ErrInvalidConfig) || rejection != nil {
			t.Errorf("%s: %v, %v", name, rejection, err)
		}
	}
	for _, environment := range specdata.Environments() {
		cfg, err := NewConfig(environment)
		pins, _ := specdata.DefaultNetworkPins(environment)
		if err != nil || cfg.Networks != pins || cfg.Validate() != nil {
			t.Errorf("NewConfig(%s) = %+v, %v", environment, cfg, err)
		}
	}
}
