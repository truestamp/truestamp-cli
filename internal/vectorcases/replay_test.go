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

// Every case's replay file loads with its case's sources, and every file
// under corpus/replay/ is one case's: 85 files, 18 of them the sources-*
// cases', which alone hold records with a source; two of them hold a
// keyring read, each of www.truestamp.com, and no other does; and each
// holds the identity read of every source of every Stellar and Bitcoin
// network its case pins, as a commitment or an entropy pin, a named source's
// under its name and a network's one source's under none, but the one read
// SPEC.md section 13.2 excepts, that of the case whose point is a network
// with no source, configuration-online-no_source-regtest's Bitcoin regtest,
// each found from a read built with its members in another order than the
// file's. Mutants killed: readKey keying a read by its kind alone (a file's
// two reads of one kind collide as a repeat); the replay's key without the
// source (each sources-* file's records of one read from two sources
// collide as a repeat); Identity's read made with no source (a named
// source's identity read is not found); another configuration-online case's
// identity record dropped (configuration-online-mainnet-other_block0
// without its stellar_identity record has no identity read of Stellar
// testnet).
func TestReplay_EveryCaseFileLoads(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	named := map[string]bool{}
	keyringReads, withSources := 0, 0
	for _, c := range cases {
		if c.Config.Online == nil {
			continue
		}
		named[filepath.Base(*c.Config.Online)] = true
		replay, err := vectorcases.LoadCaseReplay(c)
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		if _, err := replay.Keyring(verifier.CurrentKeyringURL); !errors.Is(err, vectorcases.ErrNoAnswer) {
			keyringReads++
			if !strings.HasPrefix(c.ID, "key_binding-unavailable-") {
				t.Errorf("%s holds a keyring read", c.ID)
			}
		}
		sourced := slices.ContainsFunc(replay.Records(), func(r vectorcases.Record) bool { return r.Source != "" })
		if sourced {
			withSources++
		}
		if sourced != (c.Config.Sources != nil) {
			t.Errorf("%s: records with a source %v, sources %v", c.ID, sourced, c.Config.Sources)
		}
		for _, pins := range []map[string]string{c.Config.Networks.Commitment, c.Config.Networks.Entropy} {
			for chain, network := range pins {
				names := []string{""}
				if networkSources, ok := c.Config.Sources[chain][network]; ok {
					names = networkSources.Sources
				}
				for _, name := range names {
					_, err := replay.Identity(chain, network, name)
					if errors.Is(err, vectorcases.ErrNoAnswer) && !noSource(c.ID, chain, network) {
						t.Errorf("%s: no identity read of %s %s from the source %q", c.ID, chain, network, name)
					}
				}
				if _, ok := replay.Answer(map[string]any{"network": network, "kind": chain + "_identity"}, names[0]); !ok && !noSource(c.ID, chain, network) {
					t.Errorf("%s: no identity read of %s %s found from a read in another order", c.ID, chain, network)
				}
			}
		}
	}
	entries, err := os.ReadDir(vectorcases.Path("corpus/replay"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(named) || len(named) != 85 || keyringReads != 2 || withSources != 18 {
		t.Fatalf("%d files, %d named by cases, %d keyring reads, %d with sources", len(entries), len(named), keyringReads, withSources)
	}
	for _, entry := range entries {
		if !named[entry.Name()] {
			t.Errorf("%s is no case's replay file", entry.Name())
		}
	}
}

// noSource says whether a case's pin is the network with no source whose
// identity read SPEC.md section 13.2 lets a replay file leave out.
func noSource(id, chain, network string) bool {
	return id == "configuration-online-no_source-regtest" && chain == "bitcoin" && network == "regtest"
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
	if value, err := mismatch.Identity("bitcoin", "regtest", ""); err != nil || value != verifier.ProductionGenesisHash {
		t.Errorf("bitcoin regtest: %q, %v", value, err)
	}
	if value, err := mismatch.Identity("stellar", "testnet", ""); err != nil || value != "Test SDF Network ; September 2015" {
		t.Errorf("stellar testnet: %q, %v", value, err)
	}
	if _, err := mismatch.Identity("bitcoin", "mainnet", ""); !errors.Is(err, vectorcases.ErrNoAnswer) {
		t.Errorf("bitcoin mainnet: %v, want no answer", err)
	}
	unreachable, err := vectorcases.LoadReplay("corpus/replay/reads-identity-bitcoin-unreachable.json")
	if err != nil {
		t.Fatal(err)
	}
	if value, err := unreachable.Identity("bitcoin", "regtest", ""); !errors.Is(err, vectorcases.ErrUnreachable) || value != "" || !strings.Contains(err.Error(), "timeout") {
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

// A record is {answer, read} or {answer, read, source} (SPEC.md section
// 13.2): a file may hold one read from a network's one source, with no
// source, and from named sources, each under its name, and each answers
// only for its own source; Sources lists each identity read with its
// source's name, the empty string for none; Records gives each record's
// kind, network and source. In sources-first-past_identity_unreachable, the
// identity read of Stellar testnet is unreachable from the source a and
// found from b, and Bitcoin regtest's one source has no name. Mutants
// killed: the record's member count fixed at two (every record with a
// source is refused); the replay's key without the source (the second and
// third records repeat the first); Identity or Answer ignoring the source
// asked for (a gives b's answer, or the source c, which the file does not
// hold, is answered); Sources without the name; Records without the
// source.
func TestReplay_RecordsWithAndWithoutSource(t *testing.T) {
	const read = `"read": {"kind": "stellar_identity", "network": "testnet"}`
	record := func(passphrase, source string) string {
		text := `{"answer": {"outcome": "found", "value": {"network_passphrase": "` + passphrase + `"}}, ` + read
		if source != "" {
			text += `, "source": "` + source + `"`
		}
		return text + `}`
	}
	replay, err := vectorcases.ParseReplay([]byte(`{"reads": [` + record("one", "") + `, ` + record("from a", "a") + `, ` + record("from b", "b") + `]}`))
	if err != nil {
		t.Fatalf("one read from no source, a and b: %v", err)
	}
	for source, want := range map[string]string{"": "one", "a": "from a", "b": "from b"} {
		if got, err := replay.Identity("stellar", "testnet", source); err != nil || got != want {
			t.Errorf("the identity from %q: %q, %v; want %q", source, got, err, want)
		}
	}
	if got, err := replay.Identity("stellar", "testnet", "c"); !errors.Is(err, vectorcases.ErrNoAnswer) {
		t.Errorf("the identity from c: %q, %v; want no answer", got, err)
	}
	if _, ok := replay.Answer(map[string]any{"network": "testnet", "kind": "stellar_identity"}, "c"); ok {
		t.Error("Answer from c: found")
	}
	wantSources := []vectorcases.Source{{Chain: "stellar", Network: "testnet"}, {Chain: "stellar", Network: "testnet", Name: "a"}, {Chain: "stellar", Network: "testnet", Name: "b"}}
	if got := replay.Sources(); !slices.Equal(got, wantSources) {
		t.Errorf("Sources() = %+v, want %+v", got, wantSources)
	}
	wantRecords := []vectorcases.Record{{Kind: "stellar_identity", Network: "testnet"}, {Kind: "stellar_identity", Network: "testnet", Source: "a"},
		{Kind: "stellar_identity", Network: "testnet", Source: "b"}}
	if got := replay.Records(); !slices.Equal(got, wantRecords) {
		t.Errorf("Records() = %+v, want %+v", got, wantRecords)
	}

	file, err := vectorcases.LoadReplay("corpus/replay/sources-first-past_identity_unreachable.json")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := file.Identity("stellar", "testnet", "a"); !errors.Is(err, vectorcases.ErrUnreachable) || got != "" || !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Stellar testnet from a: %q, %v; want unreachable, timeout", got, err)
	}
	if got, err := file.Identity("stellar", "testnet", "b"); err != nil || got != "Test SDF Network ; September 2015" {
		t.Errorf("Stellar testnet from b: %q, %v", got, err)
	}
	if _, err := file.Identity("stellar", "testnet", ""); !errors.Is(err, vectorcases.ErrNoAnswer) {
		t.Errorf("Stellar testnet from no source: %v, want no answer", err)
	}
	if got, err := file.Identity("bitcoin", "regtest", ""); err != nil || got != "0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206" {
		t.Errorf("Bitcoin regtest: %q, %v", got, err)
	}
	if _, err := file.Identity("bitcoin", "regtest", "a"); !errors.Is(err, vectorcases.ErrNoAnswer) {
		t.Errorf("Bitcoin regtest from a: %v, want no answer", err)
	}
	wantFileSources := []vectorcases.Source{{Chain: "bitcoin", Network: "regtest"}, {Chain: "stellar", Network: "testnet", Name: "a"}, {Chain: "stellar", Network: "testnet", Name: "b"}}
	if got := file.Sources(); !slices.Equal(got, wantFileSources) {
		t.Errorf("the file's Sources() = %+v, want %+v", got, wantFileSources)
	}
}

// ParseReplay refuses a source SPEC.md section 13.2 does not allow: two
// records of one read from one source, or both from none; a source that is
// not an ASCII16 (a number, null, the empty string, 17 characters, a
// quote); a source on a read of no Stellar or Bitcoin network (a keyring
// read, a NIST pulse); and a record of four members. It reads one read from
// a named source and from none. Mutants killed: the repeat check left out;
// the source's ASCII16 check left out; the check that only a read of a
// Stellar or Bitcoin network has a source left out; the record's member
// count taking any number above two.
func TestReplay_SourceRefusals(t *testing.T) {
	const identity = `"read": {"kind": "bitcoin_identity", "network": "regtest"}`
	const found = `"answer": {"outcome": "found", "value": {"hash": "0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206"}}`
	record := func(read, source string) string {
		if source == "" {
			return `{` + found + `, ` + read + `}`
		}
		return `{` + found + `, ` + read + `, "source": ` + source + `}`
	}
	file := func(records ...string) []byte { return []byte(`{"reads": [` + strings.Join(records, ", ") + `]}`) }
	if _, err := vectorcases.ParseReplay(file(record(identity, `"a"`), record(identity, ""))); err != nil {
		t.Fatalf("one read from a and from none: %v", err)
	}
	const keyring = `"read": {"kind": "keyring", "url": "https://www.truestamp.com/.well-known/keyring.json"}`
	const keyringFound = `"answer": {"outcome": "found", "value": {"keyring": {}}}`
	const nist = `"read": {"chainIndex": 2, "kind": "nist_pulse", "pulseIndex": 1}`
	const nistFound = `"answer": {"outcome": "found", "value": {"outputValue": "00", "timeStamp": "x"}}`
	for name, text := range map[string][]byte{
		"one read from a twice":     file(record(identity, `"a"`), record(identity, `"a"`)),
		"one read from none twice":  file(record(identity, ""), record(identity, "")),
		"a source that is a number": file(record(identity, `1`)),
		"a source that is null":     file(record(identity, `null`)),
		"an empty source":           file(record(identity, `""`)),
		"a source of 17 characters": file(record(identity, `"abcdefghijklmnopq"`)),
		"a source with a quote":     file(record(identity, `"a\"b"`)),
		"a keyring read's source":   []byte(`{"reads": [{` + keyringFound + `, ` + keyring + `, "source": "a"}]}`),
		"a NIST read's source":      []byte(`{"reads": [{` + nistFound + `, ` + nist + `, "source": "a"}]}`),
		"a fourth member":           []byte(`{"reads": [{` + found + `, ` + identity + `, "source": "a", "note": "x"}]}`),
	} {
		if _, err := vectorcases.ParseReplay(text); err == nil {
			t.Errorf("%s: read", name)
		}
	}
	if _, err := vectorcases.ParseReplay(file(record(identity, `"abcdefghijklmnop"`))); err != nil {
		t.Errorf("a source of 16 characters: %v", err)
	}
}

// LoadCaseReplay holds each record's source to its case's sources (SPEC.md
// section 13.2): in sources-all-agree, whose sources are a and b for
// Bitcoin regtest and Stellar testnet, every record of those networks has
// one; given sources that leave b out, a record from b is refused; given
// no sources of Stellar, a Stellar record with a source is refused; and in
// gates-online-derive, whose case names no sources, a record without one is
// refused once its case names sources of that network. An offline case has
// no replay. Mutants killed: the check that a record's source is one of its
// network's names left out; the check that a record of a network the case
// names no sources of has none left out; a named network's record taken
// without a source.
func TestReplay_CaseSources(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]vectorcases.Case{}
	for _, c := range cases {
		byID[c.ID] = c
	}
	agree, sound := byID["sources-all-agree"], byID["gates-online-derive"]
	if agree.Config.Online == nil || sound.Config.Online == nil || sound.Config.Sources != nil {
		t.Fatalf("sources-all-agree %+v, gates-online-derive %+v", agree.Config, sound.Config)
	}
	if _, err := vectorcases.LoadCaseReplay(agree); err != nil {
		t.Fatalf("sources-all-agree: %v", err)
	}
	if _, err := vectorcases.LoadCaseReplay(sound); err != nil {
		t.Fatalf("gates-online-derive: %v", err)
	}
	withSources := func(c vectorcases.Case, sources map[string]map[string]vectorcases.NetworkSources) vectorcases.Case {
		c.Config.Sources = sources
		return c
	}
	two := vectorcases.NetworkSources{Policy: vectorcases.PolicyAll, Sources: []string{"a", "b"}}
	for name, c := range map[string]vectorcases.Case{
		"b left out": withSources(agree, map[string]map[string]vectorcases.NetworkSources{
			"bitcoin": {"regtest": two}, "stellar": {"testnet": {Policy: vectorcases.PolicyFirst, Sources: []string{"a"}}},
		}),
		"no sources of Stellar":  withSources(agree, map[string]map[string]vectorcases.NetworkSources{"bitcoin": {"regtest": two}}),
		"sources the file lacks": withSources(sound, map[string]map[string]vectorcases.NetworkSources{"stellar": {"testnet": two}}),
		"offline":                byID["shape-witnesses-swapped"],
	} {
		if _, err := vectorcases.LoadCaseReplay(c); err == nil {
			t.Errorf("%s: read", name)
		}
	}
}
