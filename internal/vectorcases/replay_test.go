// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package vectorcases_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
	"github.com/truestamp/truestamp-cli/internal/verifier/keyring"
)

// The replay adapter answers a keyring read (SPEC.md section 13.2's kind
// keyring) from the case's file: key_binding-unavailable-unreachable's read
// of the keyring of www.truestamp.com is unreachable, with its reason, and
// key_binding-unavailable-load_check's is found, a keyring value that holds
// VECTOR_KEY, so it fails the load checks under prod and loads under test;
// a read of another URL is one neither file holds. Mutants killed:
// Replay.Keyring giving an unreachable answer as found (no error);
// Replay.Keyring answering by kind alone (the staging URL is answered);
// the found value given whole, {keyring}, not its keyring member (the
// value fails check 1 under test too).
func TestReplay_KeyringReads(t *testing.T) {
	unreachable, err := vectorcases.LoadReplay("corpus/replay/key_binding-unavailable-unreachable.json")
	if err != nil {
		t.Fatal(err)
	}
	if value, err := unreachable.Keyring(verifier.CurrentKeyringURL); !errors.Is(err, vectorcases.ErrUnreachable) ||
		!strings.Contains(err.Error(), "HTTP 503") || value != nil {
		t.Errorf("unreachable: %v, %v", value, err)
	}
	found, err := vectorcases.LoadReplay("corpus/replay/key_binding-unavailable-load_check.json")
	if err != nil {
		t.Fatal(err)
	}
	value, err := found.Keyring(verifier.CurrentKeyringURL)
	if err != nil {
		t.Fatal(err)
	}
	var failure *keyring.Failure
	if _, err := keyring.LoadValue(value, "prod", verifier.CurrentKeyringURL); !errors.As(err, &failure) || failure.Check != keyring.CheckVectorKey {
		t.Errorf("under prod: %v, want VECTOR_KEY refused", err)
	}
	loaded, err := keyring.LoadValue(value, "test", verifier.CurrentKeyringURL)
	if err != nil {
		t.Fatalf("under test: %v", err)
	}
	if _, ok := loaded.Find(keyring.VectorKey()); !ok || loaded.Digest != "" || len(loaded.Keys) != 1 {
		t.Errorf("under test: %+v", loaded)
	}
	const staging = "https://staging.truestamp.com/.well-known/keyring.json"
	for _, replay := range []*vectorcases.Replay{unreachable, found} {
		if _, err := replay.Keyring(staging); !errors.Is(err, vectorcases.ErrNoAnswer) {
			t.Errorf("a read of %s: %v, want no answer", staging, err)
		}
	}
}

// Every case's replay file loads, and every file under corpus/replay/ is
// one case's; two of them hold a keyring read, each of
// www.truestamp.com, and no other does; and each holds the identity read
// of every commitment network its case pins, but the case whose point is a
// network with no source, found from a read built with its members in
// another order than the file's. Mutant killed: readKey keying a read by
// its kind alone (a file's two reads of one kind collide as a repeat).
func TestReplay_EveryCaseFileLoads(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	named := map[string]bool{}
	keyringReads := 0
	for _, c := range cases {
		if c.Config.Online == nil {
			continue
		}
		named[filepath.Base(*c.Config.Online)] = true
		replay, err := vectorcases.LoadReplay(*c.Config.Online)
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		if _, err := replay.Keyring(verifier.CurrentKeyringURL); !errors.Is(err, vectorcases.ErrNoAnswer) {
			keyringReads++
			if !strings.HasPrefix(c.ID, "key_binding-unavailable-") {
				t.Errorf("%s holds a keyring read", c.ID)
			}
		}
		for chain, network := range c.Config.Networks.Commitment {
			if _, ok := replay.Answer(map[string]any{"network": network, "kind": chain + "_identity"}); !ok && !strings.HasPrefix(c.ID, "configuration-online-") {
				t.Errorf("%s: no identity read of %s %s", c.ID, chain, network)
			}
		}
	}
	entries, err := os.ReadDir(vectorcases.Path("corpus/replay"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(named) || len(named) != 67 || keyringReads != 2 {
		t.Fatalf("%d files, %d named by cases, %d keyring reads", len(entries), len(named), keyringReads)
	}
	for _, entry := range entries {
		if !named[entry.Name()] {
			t.Errorf("%s is no case's replay file", entry.Name())
		}
	}
}

// ParseReplay refuses a file SPEC.md section 13.2 does not allow: two
// records of one read, a keyring read not_found, a keyring value that is
// not exactly {keyring}, an unreachable answer with no reason, a member
// beside reads, a record with a third member and an unknown outcome; it
// reads a well-formed one. Mutants killed: the repeat check left out; the
// never-not_found kinds check left out; the {keyring} member count left
// out; the unreachable reason check left out; the file's member count left
// out; the record's member count left out; the outcome switch's default
// accepting.
func TestReplay_ParseRefusals(t *testing.T) {
	const keyringRead = `"read": {"kind": "keyring", "url": "https://www.truestamp.com/.well-known/keyring.json"}`
	const found = `{"answer": {"outcome": "found", "value": {"keyring": {"keys": [], "version": "1.0"}}}, ` + keyringRead + `}`
	if replay, err := vectorcases.ParseReplay([]byte(`{"reads": [` + found + `]}`)); err != nil {
		t.Fatalf("a well-formed file: %v", err)
	} else if value, err := replay.Keyring(verifier.CurrentKeyringURL); err != nil || value == nil {
		t.Fatalf("its keyring: %v, %v", value, err)
	}
	refused := map[string]string{
		"a repeated read":            `{"reads": [` + found + `, ` + found + `]}`,
		"a keyring read not_found":   `{"reads": [{"answer": {"outcome": "not_found"}, ` + keyringRead + `}]}`,
		"a keyring value beside":     `{"reads": [{"answer": {"outcome": "found", "value": {"keyring": {}, "url": "x"}}, ` + keyringRead + `}]}`,
		"unreachable with no reason": `{"reads": [{"answer": {"outcome": "unreachable"}, ` + keyringRead + `}]}`,
		"a member beside reads":      `{"reads": [], "version": 1}`,
		"a record's third member":    `{"reads": [{"answer": {"outcome": "unreachable", "reason": "x"}, "note": "x", ` + keyringRead + `}]}`,
		"an unknown outcome":         `{"reads": [{"answer": {"outcome": "timeout", "reason": "x"}, ` + keyringRead + `}]}`,
	}
	for name, text := range refused {
		if _, err := vectorcases.ParseReplay([]byte(text)); err == nil {
			t.Errorf("%s: read", name)
		}
	}
}

// The replay adapter answers an identity read (SPEC.md section 13.2's
// kinds stellar_identity and bitcoin_identity) from the case's file, by its
// chain and network: found, the value's one field; unreachable, with its
// reason; a network the file holds no identity read of, no answer. Sources
// lists each network whose identity read the file holds, found or
// unreachable. Mutants killed: Identity answering by kind alone (regtest
// is answered for mainnet); Identity giving an unreachable answer as found;
// Sources leaving out an unreachable identity read.
func TestReplay_IdentityReads(t *testing.T) {
	mismatch, err := vectorcases.LoadReplay("corpus/replay/configuration-online-regtest-mainnet_block0.json")
	if err != nil {
		t.Fatal(err)
	}
	if value, err := mismatch.Identity("bitcoin", "regtest"); err != nil || value != verifier.ProductionGenesisHash {
		t.Errorf("bitcoin regtest: %q, %v", value, err)
	}
	if value, err := mismatch.Identity("stellar", "testnet"); err != nil || value != "Test SDF Network ; September 2015" {
		t.Errorf("stellar testnet: %q, %v", value, err)
	}
	if _, err := mismatch.Identity("bitcoin", "mainnet"); !errors.Is(err, vectorcases.ErrNoAnswer) {
		t.Errorf("bitcoin mainnet: %v, want no answer", err)
	}
	unreachable, err := vectorcases.LoadReplay("corpus/replay/reads-identity-bitcoin-unreachable.json")
	if err != nil {
		t.Fatal(err)
	}
	if value, err := unreachable.Identity("bitcoin", "regtest"); !errors.Is(err, vectorcases.ErrUnreachable) || value != "" || !strings.Contains(err.Error(), "timeout") {
		t.Errorf("unreachable: %q, %v", value, err)
	}
	if got := unreachable.Sources(); len(got) != 2 || !slices.Contains(got, vectorcases.Source{Chain: "bitcoin", Network: "regtest"}) ||
		!slices.Contains(got, vectorcases.Source{Chain: "stellar", Network: "testnet"}) {
		t.Errorf("sources = %+v", got)
	}
	none, err := vectorcases.LoadReplay("corpus/replay/configuration-online-no_source-regtest.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := none.Sources(); len(got) != 1 || got[0] != (vectorcases.Source{Chain: "stellar", Network: "testnet"}) {
		t.Errorf("no_source-regtest's sources = %+v", got)
	}
}

// ParseReplay refuses an identity read SPEC.md section 13.2 does not
// allow: one not_found, one with a parameter beside its network, one whose
// network is not of its chain, and a found value that is not exactly its
// one field in its class (a passphrase that is not an ASCII128, a hash that
// is not a HEX64, a field beside it). Mutants killed: identityRead without
// its member count; without its network check; without its value check.
func TestReplay_IdentityRefusals(t *testing.T) {
	record := func(answer, read string) string {
		return `{"reads": [{"answer": ` + answer + `, "read": ` + read + `}]}`
	}
	const bitcoinRead = `{"kind": "bitcoin_identity", "network": "regtest"}`
	const hash = `"0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206"`
	if _, err := vectorcases.ParseReplay([]byte(record(`{"outcome": "found", "value": {"hash": `+hash+`}}`, bitcoinRead))); err != nil {
		t.Fatalf("a well-formed identity read: %v", err)
	}
	for name, text := range map[string]string{
		"not_found":                  record(`{"outcome": "not_found"}`, bitcoinRead),
		"a parameter beside":         record(`{"outcome": "found", "value": {"hash": `+hash+`}}`, `{"kind": "bitcoin_identity", "network": "regtest", "height": 0}`),
		"a network of another chain": record(`{"outcome": "found", "value": {"hash": `+hash+`}}`, `{"kind": "bitcoin_identity", "network": "public"}`),
		"a hash in uppercase":        record(`{"outcome": "found", "value": {"hash": "0F9188F13CB7B2C71F2A335E3A4FC328BF5BEB436012AFCA590B1A11466E2206"}}`, bitcoinRead),
		"a field beside":             record(`{"outcome": "found", "value": {"hash": `+hash+`, "height": 0}}`, bitcoinRead),
		"a passphrase with a quote":  record(`{"outcome": "found", "value": {"network_passphrase": "a\"b"}}`, `{"kind": "stellar_identity", "network": "testnet"}`),
		"a passphrase as a hash":     record(`{"outcome": "found", "value": {"hash": `+hash+`}}`, `{"kind": "stellar_identity", "network": "testnet"}`),
	} {
		if _, err := vectorcases.ParseReplay([]byte(text)); err == nil {
			t.Errorf("%s: read", name)
		}
	}
}
