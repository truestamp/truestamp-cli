// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package keyring holds a keyring and the load checks of SPEC.md section
// 10.2, step 0 (CONFIGURATION), which every keyring the verifier loads
// passes before any key in it can bind (sections 8.3 (External verifiers)
// and 9 (What a verifier pins)). It is the one loader for every keyring a
// run loads: the compiled pin, a keyring file, a configured keyring, and,
// online, a keyring read from www.truestamp.com or from the origin an
// explicit --base-url names; a test holds the compiled pin to it too.
//
// It ports the reference verifier's Truestamp.Verifier.Keyring
// (elixir/lib/truestamp/verifier/keyring.ex in truestamp-verification),
// checked against the Truestamp application's port of its load checks,
// Truestamp.Proof.KeyringRules and Truestamp.Proof.KeyringLoader
// (lib/truestamp/proof/keyring_rules.ex and keyring_loader.ex in
// truestamp-v2). [Load] reads a keyring's bytes under the parsing rules of
// section 10.1 (J1 to J6, with internal/strictjson), then makes exactly
// these checks, and no others (no [L]A check):
//
//  1. the keyring is an object with exactly two members, keys, an array,
//     and version, the string "1.0";
//  2. each element of keys is an object with exactly active (a boolean),
//     key_id (KID), public_key (B64U32) and sequence (INT);
//  3. each key_id is the key id of its public_key (section 3.3);
//  4. no two keys share a public_key or a key_id;
//  5. each public_key passes the checks of a key's bytes (section 6.2): its
//     y is below 2^255 - 19, and it is not one of SMALL_ORDER_KEYS;
//  6. no key is VECTOR_KEY unless E_v is test;
//  7. in a production keyring, no key is in spec/keys.nonproduction.json.
//
// A production keyring is any keyring loaded in a run whose E_v is prod,
// whatever its source, a keyring file included (section 6.4 (The
// environment marker)), so check 7 applies exactly when the environment
// [Load] is given is prod.
//
// Each check is made over every key before the next check, as in the
// reference, so a keyring with two defects is named by the earlier check
// of the list above. The application names the first key that fails and
// the first check it fails; both refuse exactly the same keyrings, and
// SPEC.md fixes no order. Adapted from the reference: a key's public key is
// a [32]byte, a failure also names the index of the key that fails, and a
// loaded keyring holds no digest: trust_root's digest d (section 8.3) is
// made from the bytes as read where the report's trust_root is.
package keyring

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/ed25519"
)

// Key is one key of a loaded keyring.
type Key struct {
	Active bool
	// KeyID is the key id of PublicKey, 8 lowercase hex characters (KID).
	KeyID string
	// PublicKey is the key's 32 octets.
	PublicKey [32]byte
	// Sequence is an INT: an integer from 0 to MAX_INTEGER.
	Sequence int64
}

// Keyring is a keyring that passed the load checks: its keys in the
// file's order, and Location, the file or URL it came from, or "" when it
// came from neither.
type Keyring struct {
	Keys     []Key
	Location string
}

// Find returns the key of k whose 32 octets equal publicKey, and whether
// there is one. Only the full comparison binds (SPEC.md section 6.1): no
// key id is read.
func (k *Keyring) Find(publicKey [32]byte) (Key, bool) {
	for _, key := range k.Keys {
		if key.PublicKey == publicKey {
			return key, true
		}
	}
	return Key{}, false
}

// Check names a load check, by the order of the package's list.
type Check int

// The load checks a keyring can fail.
const (
	// CheckJSON: the bytes break a parsing rule of SPEC.md section 10.1.
	CheckJSON Check = iota + 1
	// CheckKeyringMembers is check 1.
	CheckKeyringMembers
	// CheckKeyMembers is check 2.
	CheckKeyMembers
	// CheckKeyID is check 3.
	CheckKeyID
	// CheckRepeatedPublicKey and CheckRepeatedKeyID are check 4.
	CheckRepeatedPublicKey
	CheckRepeatedKeyID
	// CheckYNotBelowP and CheckSmallOrderKey are check 5.
	CheckYNotBelowP
	CheckSmallOrderKey
	// CheckVectorKey is check 6.
	CheckVectorKey
	// CheckNonproductionKey is check 7.
	CheckNonproductionKey
)

// Failure is the load check a keyring fails.
type Failure struct {
	Check Check
	// Key is the index in keys of the key that fails, or -1 for a check of
	// the keyring as a whole (CheckJSON, CheckKeyringMembers). For a
	// repeat, it is the later of the two keys.
	Key int
	// Rejection is the parsing rule the bytes break, for CheckJSON.
	Rejection *strictjson.Rejection
}

// Error describes the failure, as [Failure.Describe] does.
func (f *Failure) Error() string { return "keyring: " + f.Describe() }

// Describe is a plain sentence saying which load check f names, for the
// message of configuration_error, which carries no sub_reason.
func (f *Failure) Describe() string {
	key := fmt.Sprintf("the key at index %d", f.Key)
	switch f.Check {
	case CheckJSON:
		if f.Rejection != nil && f.Rejection.Code == specdata.ReasonInputTooLarge {
			return fmt.Sprintf("it is larger than %d octets", strictjson.MaxReadOctets)
		}
		sub := specdata.SubReason("")
		if f.Rejection != nil {
			sub = f.Rejection.SubReason
		}
		return fmt.Sprintf("it is not JSON that every verifier reads alike (%s)", sub)
	case CheckKeyringMembers:
		return `it is not an object of exactly "keys", an array, and "version", the string "1.0"`
	case CheckKeyMembers:
		return key + ` is not an object of exactly "active", "key_id", "public_key" and "sequence", each of its class`
	case CheckKeyID:
		return key + `'s "key_id" is not the key id of its "public_key"`
	case CheckRepeatedPublicKey:
		return key + " repeats the public key of an earlier key"
	case CheckRepeatedKeyID:
		return key + " repeats the key id of an earlier key"
	case CheckYNotBelowP:
		return key + "'s public key has a y that is not below 2^255 - 19"
	case CheckSmallOrderKey:
		return key + "'s public key is one of SMALL_ORDER_KEYS"
	case CheckVectorKey:
		return key + " is VECTOR_KEY, whose seed is public, and the run's environment is not test"
	case CheckNonproductionKey:
		return "it is a production keyring, since the run's environment is prod, and " + key + " belongs to a non-production environment"
	}
	return "it fails a load check"
}

// ErrUnknownEnvironment marks a call of [Load] with an environment that is
// not an E_v (SPEC.md section 6.4), a defect in the caller.
var ErrUnknownEnvironment = errors.New("keyring: not an environment a verifier knows")

// Load loads the keyring whose bytes are data in a run whose E_v is
// environment, with the checks of the package's list: the keyring, or a
// [*Failure] naming the first check that fails. location names where the
// bytes came from, for trust_root (SPEC.md section 10.5), or is "".
func Load(data []byte, environment string, location string) (*Keyring, error) {
	if !slices.Contains(specdata.Environments(), environment) {
		return nil, ErrUnknownEnvironment
	}
	value, err := strictjson.Decode(data)
	if err != nil {
		var rejection *strictjson.Rejection
		if errors.As(err, &rejection) {
			return nil, &Failure{Check: CheckJSON, Key: -1, Rejection: rejection}
		}
		return nil, err
	}
	keys, failure := form(value)
	if failure != nil {
		return nil, failure
	}
	if failure := each(keys, keyIDOfItsKey); failure != nil {
		return nil, failure
	}
	if failure := unique(keys); failure != nil {
		return nil, failure
	}
	if failure := each(keys, keyBytes); failure != nil {
		return nil, failure
	}
	if failure := each(keys, func(key Key) Check { return vectorKey(key, environment) }); failure != nil {
		return nil, failure
	}
	if failure := each(keys, func(key Key) Check { return nonproductionKey(key, environment) }); failure != nil {
		return nil, failure
	}
	return &Keyring{Keys: keys, Location: location}, nil
}

// each makes check over every key, in order: the failure of the first key
// it refuses, or nil.
func each(keys []Key, check func(Key) Check) *Failure {
	for i, key := range keys {
		if refused := check(key); refused != 0 {
			return &Failure{Check: refused, Key: i}
		}
	}
	return nil
}

// form makes checks 1 and 2: the members and their classes.
func form(value any) ([]Key, *Failure) {
	keyring, ok := value.(map[string]any)
	if !ok || len(keyring) != 2 {
		return nil, &Failure{Check: CheckKeyringMembers, Key: -1}
	}
	elements, isArray := keyring["keys"].([]any)
	version, isString := keyring["version"].(string)
	if !isArray || !isString || version != "1.0" {
		return nil, &Failure{Check: CheckKeyringMembers, Key: -1}
	}
	keys := make([]Key, 0, len(elements))
	for i, element := range elements {
		key, ok := keyOf(element)
		if !ok {
			return nil, &Failure{Check: CheckKeyMembers, Key: i}
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// keyOf reads one element of keys, when it is an object of exactly the
// four members, each of its class.
func keyOf(element any) (Key, bool) {
	object, ok := element.(map[string]any)
	if !ok || len(object) != 4 {
		return Key{}, false
	}
	active, isBool := object["active"].(bool)
	keyID, isKeyID := object["key_id"].(string)
	text, isText := object["public_key"].(string)
	sequence, isNumber := object["sequence"].(strictjson.Number)
	if !isBool || !isKeyID || !isText || !isNumber || !kid.MatchString(keyID) {
		return Key{}, false
	}
	// INT: an integer from 0 to MAX_INTEGER; Int64 refuses any other.
	n, isInteger := sequence.Int64()
	if !isInteger || n < 0 {
		return Key{}, false
	}
	publicKey, err := registry.DecodeB64U32(text)
	if err != nil {
		return Key{}, false
	}
	return Key{Active: active, KeyID: keyID, PublicKey: publicKey, Sequence: n}, true
}

// keyIDOfItsKey is check 3.
func keyIDOfItsKey(key Key) Check {
	if registry.KeyIDOf(key.PublicKey) != key.KeyID {
		return CheckKeyID
	}
	return 0
}

// unique is check 4. After check 3, two keys that share a public key share
// a key id too, so the public keys are compared first; two keys share only
// a key id when their ids collide.
func unique(keys []Key) *Failure {
	for i := range keys {
		for j := range i {
			if keys[j].PublicKey == keys[i].PublicKey {
				return &Failure{Check: CheckRepeatedPublicKey, Key: i}
			}
		}
	}
	for i := range keys {
		for j := range i {
			if keys[j].KeyID == keys[i].KeyID {
				return &Failure{Check: CheckRepeatedKeyID, Key: i}
			}
		}
	}
	return nil
}

// keyBytes is check 5.
func keyBytes(key Key) Check {
	switch err := ed25519.CheckKeyBytes(key.PublicKey); {
	case errors.Is(err, ed25519.ErrYNotBelowP):
		return CheckYNotBelowP
	case errors.Is(err, ed25519.ErrSmallOrderKey):
		return CheckSmallOrderKey
	}
	return 0
}

// vectorKey is check 6.
func vectorKey(key Key, environment string) Check {
	if key.PublicKey == vector && environment != "test" {
		return CheckVectorKey
	}
	return 0
}

// nonproductionKey is check 7: every keyring loaded under E_v prod is a
// production keyring (SPEC.md section 6.4).
func nonproductionKey(key Key, environment string) Check {
	if environment == "prod" && slices.Contains(nonproduction, key.PublicKey) {
		return CheckNonproductionKey
	}
	return 0
}

// CompiledPin returns the bytes of the compiled pin,
// spec/keyring.production.json as built in (SPEC.md section 8.3). A run
// loads it through [Load] under prod, as it loads every keyring.
func CompiledPin() []byte {
	data, err := specdata.File("keyring.production.json")
	if err != nil {
		// The file is embedded at build time, and specdata's tests hold the
		// embedded files to the spec's, so it is always there.
		panic(err)
	}
	return data
}

// NonproductionKeys returns the 32-octet keys of
// spec/keys.nonproduction.json, in the file's order.
func NonproductionKeys() [][32]byte {
	return slices.Clone(nonproduction)
}

// VectorKey returns VECTOR_KEY (SPEC.md section 9) as its 32 octets.
func VectorKey() [32]byte { return vector }

// kid is the KID class of spec/encodings.json, matched against a whole
// string.
var kid = regexp.MustCompile(`\A(?:` + specdata.Encodings()["KID"].Pattern + `)\z`)

// vector is VECTOR_KEY, from specdata's transcription of
// spec/constants.json.
var vector = func() [32]byte {
	decoded, err := hex.DecodeString(specdata.VectorKey)
	if err != nil || len(decoded) != 32 {
		panic("keyring: VECTOR_KEY is not 32 bytes of hex")
	}
	return [32]byte(decoded)
}()

// nonproduction is the keys of spec/keys.nonproduction.json, read once
// through the strict reader. The file is embedded at build time, so a file
// this cannot read is a defect of the build, and stops it at start-up.
var nonproduction = func() [][32]byte {
	data, err := specdata.File("keys.nonproduction.json")
	if err != nil {
		panic(err)
	}
	value, err := strictjson.Decode(data)
	if err != nil {
		panic(fmt.Sprintf("keyring: spec/keys.nonproduction.json: %v", err))
	}
	document, _ := value.(map[string]any)
	listed, ok := document["keys"].([]any)
	if !ok || len(listed) == 0 {
		panic("keyring: spec/keys.nonproduction.json holds no keys")
	}
	keys := make([][32]byte, 0, len(listed))
	for _, element := range listed {
		object, _ := element.(map[string]any)
		text, _ := object["public_key"].(string)
		key, err := registry.DecodeB64U32(text)
		if err != nil {
			panic(fmt.Sprintf("keyring: spec/keys.nonproduction.json: %q is not a B64U32", text))
		}
		keys = append(keys, key)
	}
	return keys
}()
