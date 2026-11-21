// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package keyring

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier/ed25519"
)

// entry is one key as a test writes it into a keyring.
type entry struct {
	active    string // a JSON value
	keyID     string // a JSON value
	publicKey string // a JSON value
	sequence  string // a JSON value
	extra     string // further members, written as is
}

func (e entry) json() string {
	text := fmt.Sprintf(`{"active": %s, "key_id": %s, "public_key": %s, "sequence": %s`, e.active, e.keyID, e.publicKey, e.sequence)
	if e.extra != "" {
		text += ", " + e.extra
	}
	return text + "}"
}

// sound is the entry of a key that passes every load check.
func sound(key [32]byte, sequence int) entry {
	return entry{
		active:    "true",
		keyID:     fmt.Sprintf("%q", registry.KeyIDOf(key)),
		publicKey: fmt.Sprintf("%q", registry.EncodeB64U(key[:])),
		sequence:  fmt.Sprint(sequence),
	}
}

func keyringOf(entries ...entry) []byte {
	var parts []string
	for _, e := range entries {
		parts = append(parts, e.json())
	}
	return []byte(`{"keys": [` + strings.Join(parts, ", ") + `], "version": "1.0"}`)
}

// testKey is a key no environment uses, made from seed.
func testKey(seed string) [32]byte {
	return sha256.Sum256([]byte("keyring test key " + seed))
}

func mustHex32(t *testing.T, text string) [32]byte {
	t.Helper()
	decoded, err := hex.DecodeString(text)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("%q is not 32 bytes of hex", text)
	}
	return [32]byte(decoded)
}

func failureOf(t *testing.T, err error) *Failure {
	t.Helper()
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("error = %v, want a load check's failure", err)
	}
	return failure
}

// Every keyring the conformance cases supply loads, or fails the check its
// case is named for, under the case's environment; keyring.test.json, the
// keyring of VECTOR_KEY, loads under test only. Mutants killed: check 1
// without the version comparison (the version case loads); check 2 without
// the member count (the key_extra_member case loads); check 6 not applied
// under staging (keyring.test.json loads under staging).
func TestLoad_TheCasesKeyrings(t *testing.T) {
	cases := []struct {
		file        string
		environment string
		check       Check
		key         int
	}{
		{"keyring.test.json", "test", 0, 0},
		{"keyring.test.json", "staging", CheckVectorKey, 0},
		{"keyring.test.json", "prod", CheckVectorKey, 0},
		{"negative/configuration-keyring-duplicate_name.keyring.json", "test", CheckJSON, -1},
		{"negative/configuration-keyring-extra_member.keyring.json", "test", CheckKeyringMembers, -1},
		{"negative/configuration-keyring-version.keyring.json", "test", CheckKeyringMembers, -1},
		{"negative/configuration-keyring-key_extra_member.keyring.json", "test", CheckKeyMembers, 0},
		{"negative/configuration-keyring-wrong_key_id.keyring.json", "test", CheckKeyID, 0},
		{"negative/configuration-keyring-repeated_key.keyring.json", "test", CheckRepeatedPublicKey, 1},
		{"negative/configuration-keyring-y_not_below_p.keyring.json", "test", CheckYNotBelowP, 1},
		{"negative/configuration-keyring-small_order_key.keyring.json", "test", CheckSmallOrderKey, 1},
		{"negative/configuration-nonproduction_key-under-prod.keyring.json", "prod", CheckNonproductionKey, 0},
		{"negative/configuration-nonproduction_key-under-prod.keyring.json", "test", 0, 0},
	}
	for _, c := range cases {
		data, err := os.ReadFile(vectorcases.Path(c.file))
		if err != nil {
			t.Fatal(err)
		}
		keyring, err := Load(data, c.environment, c.file)
		if c.check == 0 {
			if err != nil {
				t.Errorf("%s under %s: %v", c.file, c.environment, err)
			} else if keyring.Location != c.file || len(keyring.Keys) != 1 {
				t.Errorf("%s under %s: %+v", c.file, c.environment, keyring)
			}
			continue
		}
		failure := failureOf(t, err)
		if failure.Check != c.check || failure.Key != c.key {
			t.Errorf("%s under %s: check %d at key %d, want %d at %d", c.file, c.environment, failure.Check, failure.Key, c.check, c.key)
		}
	}
}

// Check 7 is a production keyring's alone: a keyring holding any key of
// spec/keys.nonproduction.json but VECTOR_KEY loads under dev and under
// staging, and is refused under prod. Mutant killed: check 7 made under
// every E_v but test.
func TestLoad_ANonproductionKeyOutsideProd(t *testing.T) {
	for i, key := range NonproductionKeys() {
		if key == VectorKey() {
			continue
		}
		data := keyringOf(sound(testKey("a"), 0), sound(key, 1))
		for _, environment := range []string{"dev", "staging"} {
			if keyring, err := Load(data, environment, ""); err != nil || len(keyring.Keys) != 2 {
				t.Errorf("key %d under %s: %v", i, environment, err)
			}
		}
		failure := failureOf(t, func() error { _, err := Load(data, "prod", ""); return err }())
		if failure.Check != CheckNonproductionKey || failure.Key != 1 {
			t.Errorf("key %d under prod: check %d at key %d", i, failure.Check, failure.Key)
		}
	}
}

// The compiled pin passes the load checks as a production keyring, through
// the same loader every keyring a run loads goes through.
func TestCompiledPin_PassesTheLoadChecksUnderProd(t *testing.T) {
	if _, err := Load(CompiledPin(), "prod", ""); err != nil {
		t.Fatalf("the compiled pin: %v", err)
	}
	if _, err := specdata.File("keyring.production.json"); err != nil {
		t.Fatal(err)
	}
}

// productionKey and stagingKey are Truestamp's production and staging
// keys, as spec/keyring.production.json and spec/keys.nonproduction.json
// give them, with their key ids under SPEC.md section 3.3.
const (
	productionKey   = "1hHbF5H5u8LiSp-nVMRb8duR2eGkOo5Q1JYfcmAtF28"
	productionKeyID = "77e51f02"
	stagingKey      = "sYfRKf1R_ILtfRv1KjBgk6F6kgIrXDfVbz3Pl-kgE2U"
	stagingKeyID    = "0df0ebbb"
)

// The compiled pin, loaded under prod, holds exactly one key: Truestamp's
// production key, active, at sequence 0, under its key id. Mutants killed:
// the pin synced from a spec whose keyring.production.json holds no key; a
// pin whose one key is any other.
func TestCompiledPin_HoldsTheProductionKey(t *testing.T) {
	want, err := registry.DecodeB64U32(productionKey)
	if err != nil {
		t.Fatal(err)
	}
	if got := registry.KeyIDOf(want); got != productionKeyID {
		t.Fatalf("the production key's key id is %s, want %s", got, productionKeyID)
	}
	pin, err := Load(CompiledPin(), "prod", "")
	if err != nil {
		t.Fatalf("the compiled pin: %v", err)
	}
	expected := []Key{{Active: true, KeyID: productionKeyID, PublicKey: want, Sequence: 0}}
	if len(pin.Keys) != len(expected) || pin.Keys[0] != expected[0] {
		t.Fatalf("the compiled pin holds %+v, want %+v", pin.Keys, expected)
	}
}

// A keyring holding only Truestamp's staging key loads under staging and
// is refused under prod by check 7, at that key. Mutants killed: the
// staging key left out of spec/keys.nonproduction.json; check 7 made under
// no E_v.
func TestLoad_TheStagingKeyAloneUnderProd(t *testing.T) {
	key, err := registry.DecodeB64U32(stagingKey)
	if err != nil {
		t.Fatal(err)
	}
	if got := registry.KeyIDOf(key); got != stagingKeyID {
		t.Fatalf("the staging key's key id is %s, want %s", got, stagingKeyID)
	}
	data := keyringOf(sound(key, 0))
	if keyring, err := Load(data, "staging", ""); err != nil || len(keyring.Keys) != 1 || keyring.Keys[0].KeyID != stagingKeyID {
		t.Fatalf("under staging: %+v, %v", keyring, err)
	}
	failure := failureOf(t, func() error { _, err := Load(data, "prod", ""); return err }())
	if failure.Check != CheckNonproductionKey || failure.Key != 0 {
		t.Fatalf("under prod: check %d at key %d, want %d at 0", failure.Check, failure.Key, CheckNonproductionKey)
	}
}

// Check 5 checks a key's bytes and no more, so a key off the curve loads
// under every E_v, and SIGNATURE refuses it later (SPEC.md section 6.2,
// step 4): 0x02 followed by 31 zero octets, whose y is 2, is below p and
// not one of SMALL_ORDER_KEYS, and ed25519's Verify names it not_on_curve.
// Mutant killed: the load checks refusing a key off the curve.
func TestLoad_AKeyOffTheCurveLoads(t *testing.T) {
	offCurve := [32]byte{2}
	if err := ed25519.Verify(offCurve, nil, [64]byte{}); !errors.Is(err, ed25519.ErrNotOnCurve) {
		t.Fatalf("Verify: %v, want not_on_curve", err)
	}
	for _, environment := range specdata.Environments() {
		keyring, err := Load(keyringOf(sound(offCurve, 0)), environment, "")
		if err != nil || len(keyring.Keys) != 1 || keyring.Keys[0].PublicKey != offCurve {
			t.Errorf("under %s: %v", environment, err)
		}
	}
}

// Each per-key check is made on every key, not the first alone: a
// keyring whose first two keys are sound and whose third fails is refused,
// naming the third. Mutant killed: each checking keys[0] alone (every row
// here loads).
func TestLoad_ChecksKeysOtherThanTheFirst(t *testing.T) {
	a, b := sound(testKey("a"), 0), sound(testKey("b"), 1)
	small := specdata.SmallOrderKeys()[4].Key
	highY := mustHex32(t, "edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")
	nonproduction := NonproductionKeys()[0]
	wrongID := sound(testKey("c"), 2)
	wrongID.keyID = fmt.Sprintf("%q", registry.KeyIDOf(testKey("d")))
	numericID := sound(testKey("c"), 2)
	numericID.keyID = "12345678"
	cases := []struct {
		name        string
		third       entry
		environment string
		check       Check
	}{
		{"a key id that is not a string", numericID, "test", CheckKeyMembers},
		{"a key id not its key's", wrongID, "test", CheckKeyID},
		{"y not below p", sound(highY, 2), "test", CheckYNotBelowP},
		{"a small-order key", sound(small, 2), "test", CheckSmallOrderKey},
		{"VECTOR_KEY", sound(VectorKey(), 2), "dev", CheckVectorKey},
		{"a non-production key", sound(nonproduction, 2), "prod", CheckNonproductionKey},
	}
	for _, c := range cases {
		failure := failureOf(t, func() error { _, err := Load(keyringOf(a, b, c.third), c.environment, ""); return err }())
		if failure.Check != c.check || failure.Key != 2 {
			t.Errorf("%s: check %d at key %d, want %d at key 2", c.name, failure.Check, failure.Key, c.check)
		}
	}
}

// Repeats are found wherever they are, not only side by side, and a key
// other than the first is compared with every later key. Mutants killed:
// unique comparing each key with the one before it alone (a, b, c, a
// loads); unique comparing each public key with the first key's alone (a,
// b, c, b names the repeated key id, not the repeated public key).
func TestLoad_RepeatsThatAreNotAdjacent(t *testing.T) {
	a, b, c := sound(testKey("a"), 0), sound(testKey("b"), 1), sound(testKey("c"), 2)
	for name, again := range map[string]entry{"a again": sound(testKey("a"), 3), "b again": sound(testKey("b"), 3)} {
		failure := failureOf(t, func() error { _, err := Load(keyringOf(a, b, c, again), "test", ""); return err }())
		if failure.Check != CheckRepeatedPublicKey || failure.Key != 3 {
			t.Errorf("%s: check %d at key %d, want a repeated public key at key 3", name, failure.Check, failure.Key)
		}
	}
}

var (
	collisionOnce sync.Once
	collisionA    [32]byte
	collisionB    [32]byte
)

// collision returns two keys whose key ids are equal, found by a birthday
// search over keys made from a counter: a key id has 32 bits, so a pair
// turns up after about 2^16 keys.
func collision(t *testing.T) ([32]byte, [32]byte) {
	t.Helper()
	collisionOnce.Do(func() {
		seen := map[string][32]byte{}
		var counter [8]byte
		for i := uint64(0); i < 1<<22; i++ {
			binary.BigEndian.PutUint64(counter[:], i)
			key := sha256.Sum256(counter[:])
			id := registry.KeyIDOf(key)
			if other, ok := seen[id]; ok {
				collisionA, collisionB = other, key
				return
			}
			seen[id] = key
		}
	})
	if collisionA == collisionB {
		t.Fatal("no key-id collision found")
	}
	return collisionA, collisionB
}

// Check 4's second half: two keys with different public keys and one key
// id are refused, which only a key-id collision can give once check 3 has
// passed, with another key between them. Mutants killed: unique without
// its key id comparison (the pair loads); unique comparing each key id
// with the one before it alone (the separated pair loads).
func TestLoad_TwoKeysThatShareOnlyAKeyID(t *testing.T) {
	first, second := collision(t)
	if registry.KeyIDOf(first) != registry.KeyIDOf(second) || first == second {
		t.Fatal("not a collision")
	}
	data := keyringOf(sound(first, 0), sound(testKey("a"), 1), sound(second, 2))
	failure := failureOf(t, func() error { _, err := Load(data, "test", ""); return err }())
	if failure.Check != CheckRepeatedKeyID || failure.Key != 2 {
		t.Fatalf("check %d at key %d, want a repeated key id at key 2", failure.Check, failure.Key)
	}
}

// Each check is made over every key before the next, the order keyring.go
// gives them (SPEC.md section 10.2, step 0, fixes none): a keyring whose
// first key is small-order and whose second has the wrong key id fails
// check 3, not check 5. The order of the first key's first failing check
// names check 5. Mutant killed: check 5 made before check 3.
func TestLoad_ChecksInTheReferencesOrder(t *testing.T) {
	small := sound(specdata.SmallOrderKeys()[0].Key, 0)
	wrongID := sound(testKey("b"), 1)
	wrongID.keyID = fmt.Sprintf("%q", registry.KeyIDOf(testKey("c")))
	failure := failureOf(t, func() error { _, err := Load(keyringOf(small, wrongID), "test", ""); return err }())
	if failure.Check != CheckKeyID || failure.Key != 1 {
		t.Fatalf("check %d at key %d, want check 3 at key 1", failure.Check, failure.Key)
	}
}

// Check 2's classes, one member at a time: a value outside its class is
// refused, and the edges of INT and the spellings the strict reader reads
// alike are accepted. Mutants killed: sequence allowed below 0; key_id
// matched without \A and \z (a ninth digit or a final newline passes);
// public_key decoded leniently (the padded spelling passes); active
// accepted as any value.
func TestLoad_KeyMemberClasses(t *testing.T) {
	key := testKey("a")
	refused := map[string]func(*entry){
		"active a string":           func(e *entry) { e.active = `"true"` },
		"key_id uppercase":          func(e *entry) { e.keyID = strings.ToUpper(e.keyID) },
		"key_id a ninth digit":      func(e *entry) { e.keyID = e.keyID[:9] + `0"` },
		"key_id a newline":          func(e *entry) { e.keyID = e.keyID[:9] + `\n"` },
		"public_key padded":         func(e *entry) { e.publicKey = e.publicKey[:44] + `="` },
		"public_key a number":       func(e *entry) { e.publicKey = "1" },
		"sequence negative":         func(e *entry) { e.sequence = "-1" },
		"sequence a fraction":       func(e *entry) { e.sequence = "1.5" },
		"sequence a string":         func(e *entry) { e.sequence = `"0"` },
		"sequence null":             func(e *entry) { e.sequence = "null" },
		"a fifth member":            func(e *entry) { e.extra = `"environment": "test"` },
		"sequence past MAX_INTEGER": func(e *entry) { e.sequence = "1e300" },
	}
	for name, edit := range refused {
		e := sound(key, 0)
		edit(&e)
		failure := failureOf(t, func() error { _, err := Load(keyringOf(e), "test", ""); return err }())
		if failure.Check != CheckKeyMembers || failure.Key != 0 {
			t.Errorf("%s: check %d at key %d, want check 2 at key 0", name, failure.Check, failure.Key)
		}
	}
	accepted := map[string]string{
		"MAX_INTEGER":     fmt.Sprint(specdata.MaxInteger),
		"zero as 0.0":     "0.0",
		"zero as -0":      "-0",
		"one as 1e0":      "1e0",
		"one as 10e-1":    "10e-1",
		"inactive key, 0": "0",
	}
	for name, sequence := range accepted {
		e := sound(key, 0)
		e.sequence = sequence
		if name == "inactive key, 0" {
			e.active = "false"
		}
		keyring, err := Load(keyringOf(e), "test", "")
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got, ok := keyring.Find(key); !ok || got.Active != (name != "inactive key, 0") {
			t.Errorf("%s: %+v", name, keyring)
		}
	}
}

// Check 1, the keyring's own members, and J1 to J6 before any check.
// Mutants killed: check 1 without its member count (an extra member
// passes); keys accepted as any value (keys an object passes); the bytes
// decoded without J1 (a keyring of MAX_READ_OCTETS + 1 octets loads).
func TestLoad_TheKeyringsOwnForm(t *testing.T) {
	one := string(keyringOf(sound(testKey("a"), 0)))
	cases := map[string]struct {
		text  string
		check Check
	}{
		"an array":            {`[]`, CheckKeyringMembers},
		"no version":          {`{"keys": []}`, CheckKeyringMembers},
		"version a number":    {`{"keys": [], "version": 1.0}`, CheckKeyringMembers},
		"keys an object":      {`{"keys": {}, "version": "1.0"}`, CheckKeyringMembers},
		"a third member":      {`{"keys": [], "version": "1.0", "x": 1}`, CheckKeyringMembers},
		"a key not an object": {`{"keys": [1], "version": "1.0"}`, CheckKeyMembers},
		"trailing content":    {one + " {}", CheckJSON},
		"a byte order mark":   {"\ufeff" + one, CheckJSON},
	}
	for name, c := range cases {
		failure := failureOf(t, func() error { _, err := Load([]byte(c.text), "test", ""); return err }())
		if failure.Check != c.check {
			t.Errorf("%s: check %d, want %d", name, failure.Check, c.check)
		}
	}
	atLimit := one + strings.Repeat(" ", strictjson.MaxReadOctets-len(one))
	if _, err := Load([]byte(atLimit), "test", ""); err != nil {
		t.Errorf("a keyring of MAX_READ_OCTETS octets: %v", err)
	}
	over := one + strings.Repeat(" ", strictjson.MaxReadOctets+1-len(one))
	failure := failureOf(t, func() error { _, err := Load([]byte(over), "test", ""); return err }())
	if failure.Rejection == nil || failure.Rejection.Code != specdata.ReasonInputTooLarge || !strings.Contains(failure.Describe(), "larger than") {
		t.Errorf("over MAX_READ_OCTETS: %+v, %q", failure, failure.Describe())
	}
	if _, err := Load([]byte(`{"keys": [], "version": "1.0"}`), "test", ""); err != nil {
		t.Errorf("an empty keyring: %v", err)
	}
}

// Find compares all 32 octets: a key that differs from a keyring's key in
// one bit of its first or its last octet is not found. Mutant killed: Find
// comparing a prefix of the key.
func TestKeyring_FindComparesAll32Octets(t *testing.T) {
	key := testKey("a")
	keyring, err := Load(keyringOf(sound(key, 0)), "test", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := keyring.Find(key); !ok {
		t.Fatal("the key is not found")
	}
	for _, i := range []int{0, 31} {
		for bit := range 8 {
			other := key
			other[i] ^= 1 << bit
			if _, ok := keyring.Find(other); ok {
				t.Errorf("a key with bit %d of octet %d flipped binds", bit, i)
			}
		}
	}
}

// NonproductionKeys is spec/keys.nonproduction.json's keys, read here with
// encoding/json, which shares no code with the strict reader the package
// reads it with; VECTOR_KEY is spec/constants.json's and is among them.
// Mutant killed: the last key of the file dropped.
func TestNonproductionKeys_AreTheFilesKeys(t *testing.T) {
	data, err := specdata.File("keys.nonproduction.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Keys []struct {
			PublicKey string `json:"public_key"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	got := NonproductionKeys()
	if len(got) != len(file.Keys) || len(got) < 8 {
		t.Fatalf("%d keys, the file has %d", len(got), len(file.Keys))
	}
	vectorListed := false
	for i, listed := range file.Keys {
		if registry.EncodeB64U(got[i][:]) != listed.PublicKey {
			t.Errorf("key %d = %x, the file has %s", i, got[i], listed.PublicKey)
		}
		vectorListed = vectorListed || got[i] == VectorKey()
	}
	vector := VectorKey()
	if hex.EncodeToString(vector[:]) != specdata.VectorKey || !vectorListed {
		t.Errorf("VECTOR_KEY = %x, listed %v", VectorKey(), vectorListed)
	}
}

// Load takes only an E_v, and every failure says which check refused.
func TestLoad_UnknownEnvironmentAndDescriptions(t *testing.T) {
	if _, err := Load(keyringOf(), "production", ""); !errors.Is(err, ErrUnknownEnvironment) {
		t.Errorf("Load under production: %v", err)
	}
	seen := map[string]bool{}
	for check := CheckJSON; check <= CheckNonproductionKey; check++ {
		text := (&Failure{Check: check, Key: 1, Rejection: &strictjson.Rejection{Code: specdata.ReasonMalformedJSON, SubReason: specdata.SubReasonSyntax}}).Describe()
		if text == "" || text == "it fails a load check" || seen[text] {
			t.Errorf("check %d: %q", check, text)
		}
		seen[text] = true
	}
}

// A loaded keyring's digest d is the lowercase hex SHA-256 of the bytes it
// was loaded from as read (SPEC.md section 8.3), a content digest with no
// prefix byte: two spellings of one keyring, compact and indented with CRLF
// line ends and its members reordered, load to the same keys and two
// digests, each the SHA-256 of its own bytes. Mutants killed: the digest
// made over JCS of the parsed keyring (the two spellings give one digest);
// the digest written in upper case; Load leaving the digest out.
func TestLoad_TheDigestIsOfTheBytesAsRead(t *testing.T) {
	key := testKey("a")
	compact := keyringOf(sound(key, 0))
	indented := []byte(fmt.Sprintf("{\r\n  \"version\": \"1.0\",\r\n  \"keys\": [\r\n    {\"sequence\": 0, \"public_key\": %q, \"key_id\": %q, \"active\": true}\r\n  ]\r\n}\r\n",
		registry.EncodeB64U(key[:]), registry.KeyIDOf(key)))
	digests := map[string]bool{}
	for _, data := range [][]byte{compact, indented} {
		keyring, err := Load(data, "test", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(keyring.Keys) != 1 || keyring.Keys[0].PublicKey != key {
			t.Fatalf("keys %+v", keyring.Keys)
		}
		sum := sha256.Sum256(data)
		if want := hex.EncodeToString(sum[:]); keyring.Digest != want || Digest(data) != want {
			t.Errorf("digest %q, Digest %q, want %q", keyring.Digest, Digest(data), want)
		}
		digests[keyring.Digest] = true
	}
	if len(digests) != 2 {
		t.Fatalf("two spellings of one keyring give %d digests", len(digests))
	}
}

// LoadValue makes exactly Load's checks over a keyring already read, the
// form a replay file answers a keyring read with (SPEC.md section 13.2):
// each of the cases' keyrings but the one whose bytes the strict reader
// refuses gives, read and then loaded as a value, the verdict Load gives
// its bytes, under each E_v, and a value that loads has no digest, since it
// has no bytes of its own; an unknown E_v is refused. Mutants killed: Load
// giving LoadValue a fixed "test" (keyring.test.json loads from its bytes
// under staging and prod); LoadValue setting a digest of the value;
// LoadValue without its own environment check (production is taken).
func TestLoadValue_MakesLoadsChecksOverAValue(t *testing.T) {
	files := []string{
		"keyring.test.json",
		"negative/configuration-keyring-extra_member.keyring.json",
		"negative/configuration-keyring-version.keyring.json",
		"negative/configuration-keyring-key_extra_member.keyring.json",
		"negative/configuration-keyring-wrong_key_id.keyring.json",
		"negative/configuration-keyring-repeated_key.keyring.json",
		"negative/configuration-keyring-y_not_below_p.keyring.json",
		"negative/configuration-keyring-small_order_key.keyring.json",
		"negative/configuration-nonproduction_key-under-prod.keyring.json",
	}
	loaded := 0
	for _, file := range files {
		data, err := os.ReadFile(vectorcases.Path(file))
		if err != nil {
			t.Fatal(err)
		}
		value, err := strictjson.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		for _, environment := range specdata.Environments() {
			fromBytes, errBytes := Load(data, environment, file)
			fromValue, errValue := LoadValue(value, environment, file)
			if (errBytes == nil) != (errValue == nil) {
				t.Errorf("%s under %s: Load %v, LoadValue %v", file, environment, errBytes, errValue)
				continue
			}
			if errBytes != nil {
				if b, v := failureOf(t, errBytes), failureOf(t, errValue); *b != *v {
					t.Errorf("%s under %s: Load %+v, LoadValue %+v", file, environment, b, v)
				}
				continue
			}
			loaded++
			if fromValue.Digest != "" || fromValue.Location != file || len(fromValue.Keys) != len(fromBytes.Keys) || fromValue.Keys[0] != fromBytes.Keys[0] {
				t.Errorf("%s under %s: %+v from the value, %+v from the bytes", file, environment, fromValue, fromBytes)
			}
		}
	}
	if loaded != 4 {
		t.Fatalf("%d loads, want 4: keyring.test.json under test, the non-production keyring under dev, staging and test", loaded)
	}
	if _, err := LoadValue(map[string]any{}, "production", ""); !errors.Is(err, ErrUnknownEnvironment) {
		t.Errorf("LoadValue under production: %v", err)
	}
}
