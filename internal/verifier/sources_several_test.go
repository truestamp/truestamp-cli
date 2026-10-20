// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// Several sources for one network and their source policy (SPEC.md
// sections 9, "Several sources for one network" and "Which pin a source
// serves", and 10.2, step 0). As in sources_test.go, every answer comes
// from a table ([valuesReader]) or from an httptest server on loopback.

// configured is the configuration of network of chain under policy, one
// source for each URL, a Horizon server for Stellar and an Esplora server
// for Bitcoin, in the URLs' order.
func configured(chain, network string, policy SourcePolicy, urls ...string) NetworkSources {
	provider := source.Esplora
	if chain == ChainStellar {
		provider = source.Horizon
	}
	n := NetworkSources{Chain: chain, Network: network, Policy: policy}
	for _, u := range urls {
		n.Sources = append(n.Sources, Source{Chain: chain, Provider: provider, URL: u})
	}
	return n
}

// valuesReader answers each identity read from values, by URL, and each
// URL it does not hold as [honestIdentities] does; a value "" is a read
// with no usable answer.
func valuesReader(values map[string]string) IdentityReader {
	return func(s Source) (string, error) {
		value, ok := values[s.URL]
		switch {
		case !ok:
			return honestIdentities(s)
		case value == "":
			return "", errors.New("HTTP 503")
		}
		return value, nil
	}
}

// timesRead is how many of the reads in log are of u.
func timesRead(log []string, u string) int {
	n := 0
	for _, read := range log {
		if read == u {
			n++
		}
	}
	return n
}

// setOf writes a source set as "<network> <policy> <required>: <url>
// <status> <pins>, ...", for comparison.
func setOf(set SourceSet) string {
	var sources []string
	for _, r := range set.Sources {
		sources = append(sources, strings.Join([]string{r.Source.URL, string(r.Status), string(r.From), r.Source.Network, pinsOf(r)}, " "))
	}
	return fmt.Sprintf("%s %s %d: %s", set.Network, set.Policy, set.Required(), strings.Join(sources, ", "))
}

const (
	h1 = "https://h1.truestamp.example"
	h2 = "https://h2.truestamp.example"
	h3 = "https://h3.truestamp.example"
	e1 = "https://e1.truestamp.example"
	e2 = "https://e2.truestamp.example"
	e3 = "https://e3.truestamp.example"
)

// Each policy's configuration: under test, Stellar testnet configured with
// three sources under first, at_least 1, at_least 2, at_least 3 (n equal to
// the number of sources) and all, and Bitcoin regtest with one under first,
// each run passes CONFIGURATION with the network's three sources in their
// order, each confirmed, from the operator, configured for testnet and
// serving its two pins, under the policy given, whose required is 1, 1, 2,
// 3 and 3; each source is read once and no default is read. An at_least n
// whose n is 0, -1 or 4, above the three sources, is configuration_error
// naming the policy and the count, before any read: no identity read and,
// under prod with the keyring of www.truestamp.com, no keyring read, offline
// as online, and for a configuration of a network the run does not pin as
// for one it pins. Mutants killed: checkPolicies refusing an n equal to the
// number of sources (at_least 3 of 3 is refused); checkPolicies without
// its n < 1 check (at_least 0 passes); checkPolicies made only online, or
// after the reads (offline passes, or the keyring is read); checkPolicies
// only for pinned networks (Bitcoin mainnet's at_least 3 passes);
// Required giving N under all (all's required is 0) or the count under
// first (first's is 3); assign giving every network the policy first
// (at_least 2's set is under first).
func TestSources_EachPolicysConfiguration(t *testing.T) {
	values := map[string]string{h1: testnetPassphrase, h2: testnetPassphrase, h3: testnetPassphrase, e1: regtestGenesisHash}
	for _, c := range []struct {
		policy   SourcePolicy
		required int
	}{
		{SourcePolicy{Kind: SourcePolicyFirst}, 1},
		{SourcePolicy{Kind: SourcePolicyAtLeast, N: 1}, 1},
		{SourcePolicy{Kind: SourcePolicyAtLeast, N: 2}, 2},
		{SourcePolicy{Kind: SourcePolicyAtLeast, N: 3}, 3},
		{SourcePolicy{Kind: SourcePolicyAll}, 3},
	} {
		var log readLog
		cfg := online(t, "test", log.through(valuesReader(values)))
		cfg.NetworkSources = []NetworkSources{
			configured(ChainStellar, "testnet", c.policy, h1, h2, h3),
			configured(ChainBitcoin, "regtest", SourcePolicy{Kind: SourcePolicyFirst}, e1),
		}
		run, rejection := configure(t, cfg)
		if rejection != nil {
			t.Fatalf("%s: %v", c.policy, rejection)
		}
		want := fmt.Sprintf("testnet %s %d: "+
			h1+" confirmed operator testnet commitment:testnet entropy:testnet, "+
			h2+" confirmed operator testnet commitment:testnet entropy:testnet, "+
			h3+" confirmed operator testnet commitment:testnet entropy:testnet", c.policy, c.required)
		for _, pin := range []string{PinCommitment, PinEntropy} {
			set, ok := run.SourcesFor(pin, ChainStellar)
			if !ok || set.Policy != c.policy || setOf(set) != want {
				t.Fatalf("%s: the Stellar %s pin: %q, %v\nwant %q", c.policy, pin, setOf(set), ok, want)
			}
		}
		if bitcoin, _ := run.SourcesFor(PinCommitment, ChainBitcoin); setOf(bitcoin) != "regtest first 1: "+e1+" confirmed operator regtest commitment:regtest entropy:regtest" {
			t.Fatalf("%s: the Bitcoin pin: %q", c.policy, setOf(bitcoin))
		}
		reads := log.read()
		if len(reads) != 4 || timesRead(reads, h1) != 1 || timesRead(reads, h2) != 1 || timesRead(reads, h3) != 1 || timesRead(reads, e1) != 1 {
			t.Fatalf("%s: reads %q", c.policy, reads)
		}
	}

	for _, n := range []int{0, -1, 4} {
		policy := SourcePolicy{Kind: SourcePolicyAtLeast, N: n}
		for _, mode := range []string{"online", "offline", "prod online", "not pinned"} {
			var log readLog
			keyrings := &recorder{data: []byte("{}")}
			environment, network, chain, urls := "test", "testnet", ChainStellar, []string{h1, h2, h3}
			switch mode {
			case "prod online":
				environment, network = "prod", "public"
			case "not pinned":
				chain, network, urls = ChainBitcoin, "mainnet", []string{e1, e2, e3}
			}
			cfg := online(t, environment, log.through(valuesReader(values)))
			cfg.ReadKeyring = keyrings.read
			if environment == "prod" {
				cfg.Keyring = KeyringSource{}
			}
			cfg.Online = mode != "offline"
			cfg.NetworkSources = []NetworkSources{configured(chain, network, policy, urls...)}
			if mode == "not pinned" {
				cfg.NetworkSources = append([]NetworkSources{configured(ChainStellar, "testnet", SourcePolicy{Kind: SourcePolicyFirst}, h1)}, cfg.NetworkSources...)
			}
			_, rejection := configure(t, cfg)
			wantMessage := fmt.Sprintf("network have the policy at_least %d, and the network has 3 sources", n)
			if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || !strings.Contains(rejection.Message, wantMessage) {
				t.Errorf("at_least %d, %s: %v, want configuration_error: ...%s...", n, mode, rejection, wantMessage)
			}
			if len(log.read()) != 0 || len(keyrings.urls) != 0 {
				t.Errorf("at_least %d, %s: identity reads %q, keyring reads %q", n, mode, log.read(), keyrings.urls)
			}
		}
	}
}

// A source whose identity read gives a value step 0 forbids for its
// network is configuration_error whatever the policy and whatever the
// network's other sources give (SPEC.md section 10.2, step 0): of Stellar
// testnet's three sources, under first, at_least 2 and all, the one that
// gives the production passphrase is the first, the second or the third,
// and the other two each confirm or each give no usable answer; every
// source is read once, and the message names the forbidden source's
// scheme, host and port and its place in the network's order. Under prod,
// of Bitcoin mainnet's two sources the second gives regtest's hash beside
// a first that gives mainnet's. Mutants killed: assign stopping a
// network's judgement at its first confirmed source under first (the
// second and third forbidden sources pass under first); assign judging a
// network's first source alone (every forbidden source after the first
// passes); a forbidden source recorded unreachable (every run passes);
// sourceAt leaving out the place (no message names it); sourceAt counting
// places from 0 (the first source is named source 0).
func TestSources_AForbiddenIdentityFromAnySourceIsAConfigurationError(t *testing.T) {
	urls := []string{h1, h2, h3}
	for _, policy := range []SourcePolicy{{Kind: SourcePolicyFirst}, {Kind: SourcePolicyAtLeast, N: 2}, {Kind: SourcePolicyAll}} {
		for forbidden := range urls {
			for _, others := range []string{testnetPassphrase, ""} {
				values := map[string]string{e1: regtestGenesisHash}
				for i, u := range urls {
					values[u] = others
					if i == forbidden {
						values[u] = ProductionPassphrase
					}
				}
				var log readLog
				cfg := online(t, "test", log.through(valuesReader(values)))
				cfg.NetworkSources = []NetworkSources{
					configured(ChainStellar, "testnet", policy, urls...),
					configured(ChainBitcoin, "regtest", SourcePolicy{Kind: SourcePolicyFirst}, e1),
				}
				_, rejection := configure(t, cfg)
				want := fmt.Sprintf("The Stellar source at %s, source %d of the 3 sources of Stellar's testnet network, gives the network passphrase "+
					"of Stellar's production network, public, and the run pins it to testnet, so the run gives no verdict.", urls[forbidden], forbidden+1)
				if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || rejection.Message != want {
					t.Errorf("%s, source %d forbidden, the others %q: %v\nwant %s", policy, forbidden+1, others, rejection, want)
				}
				reads := log.read()
				if timesRead(reads, h1) != 1 || timesRead(reads, h2) != 1 || timesRead(reads, h3) != 1 {
					t.Errorf("%s, source %d forbidden: reads %q", policy, forbidden+1, reads)
				}
			}
		}
	}
	cfg := online(t, "prod", valuesReader(map[string]string{e1: ProductionGenesisHash, e2: regtestGenesisHash}))
	cfg.NetworkSources = []NetworkSources{configured(ChainBitcoin, "mainnet", SourcePolicy{Kind: SourcePolicyAll}, e1, e2)}
	_, rejection := configure(t, cfg)
	if rejection == nil || rejection.Message != "The Bitcoin source at "+e2+", source 2 of the 2 sources of Bitcoin's mainnet network, "+
		"gives a hash of the block at height 0 other than that of Bitcoin's production network, mainnet, which the run pins it to, so the run gives no verdict." {
		t.Fatalf("prod, Bitcoin mainnet's second source on regtest: %v", rejection)
	}
}

// Under dev and staging, which pin Bitcoin to regtest and mainnet, a
// source of a network's configuration is held to that network alone,
// never to another network the run pins on its chain (SPEC.md section
// 10.2, step 0): regtest configured with one source that gives the
// production hash is configuration_error, and it is the only read; mainnet
// configured with one source that gives regtest's hash, beside an Esplora
// server given for no network that shows regtest, is configuration_error,
// and those two are the only reads, so no default is read. Each message is
// exact, naming the configured source with no place, since its network has
// one source. Mutants killed: assign holding a configured source to any
// pinned network of its chain (both runs pass, the mainnet pin served by a
// regtest source); sourceAt naming a place for a network's one source
// (each message names source 1 of 1).
func TestSources_AConfiguredSourceIsHeldToItsOwnNetworkUnderTwoPins(t *testing.T) {
	floating := Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://floating.truestamp.example"}
	for _, environment := range []string{"dev", "staging"} {
		for _, c := range []struct {
			network, value string
			sources        []Source
			want           string
			reads          []string
		}{
			{"regtest", ProductionGenesisHash, nil,
				"The Bitcoin source at " + e1 + " gives the hash of the block at height 0 of Bitcoin's production network, mainnet, " +
					"and the run pins it to regtest, so the run gives no verdict.", []string{e1}},
			{"mainnet", regtestGenesisHash, []Source{floating},
				"The Bitcoin source at " + e1 + " gives a hash of the block at height 0 other than that of Bitcoin's production network, " +
					"mainnet, which the run pins it to, so the run gives no verdict.", []string{e1, floating.URL}},
		} {
			var log readLog
			cfg := online(t, environment, log.through(valuesReader(map[string]string{e1: c.value, floating.URL: regtestGenesisHash})), c.sources...)
			cfg.NetworkSources = []NetworkSources{configured(ChainBitcoin, c.network, SourcePolicy{Kind: SourcePolicyFirst}, e1)}
			_, rejection := configure(t, cfg)
			if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || rejection.Message != c.want {
				t.Errorf("%s, %s configured: %v\nwant %s", environment, c.network, rejection, c.want)
			}
			reads := log.read()
			slices.Sort(reads)
			if !slices.Equal(reads, c.reads) {
				t.Errorf("%s, %s configured: reads %q, want %q", environment, c.network, reads, c.reads)
			}
		}
	}
}

// The at_least range message names the network, its policy and its
// number of sources, reads well for one source, and names no source:
// under test, Bitcoin regtest configured with one, two or three Esplora
// servers, a token planted in each URL's path, under at_least 0 and
// at_least one above the count, is configuration_error with the exact
// message, which holds neither the token nor any host, and no source is
// read. Mutants killed: checkPolicies writing one source as "1 sources"
// and "from 1 to 1" (the one-source message differs); checkPolicies
// giving the one-source message for every count (two's message differs);
// checkPolicies quoting the network's first source's URL (the token is
// printed).
func TestSources_TheAtLeastRangeMessage(t *testing.T) {
	const token = "/tok-RANGE-3a7e"
	for count := 1; count <= 3; count++ {
		urls := []string{e1 + token, e2 + token, e3 + token}[:count]
		for _, n := range []int{0, count + 1} {
			var log readLog
			cfg := online(t, "test", log.through(valuesReader(nil)))
			cfg.NetworkSources = []NetworkSources{configured(ChainBitcoin, "regtest", SourcePolicy{Kind: SourcePolicyAtLeast, N: n}, urls...)}
			_, rejection := configure(t, cfg)
			want := fmt.Sprintf("The sources of Bitcoin's regtest network have the policy at_least %d, and the network has %d sources, "+
				"so the run gives no verdict: give at_least a number from 1 to %d.", n, count, count)
			if count == 1 {
				want = fmt.Sprintf("Bitcoin's regtest network has one source, under the policy at_least %d, "+
					"so the run gives no verdict: give it the policy first or at_least 1.", n)
			}
			if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || rejection.Message != want {
				t.Errorf("%d sources, at_least %d: %v\nwant %s", count, n, rejection, want)
				continue
			}
			if strings.Contains(rejection.Message, token) || strings.Contains(rejection.Message, ".example") || len(log.read()) != 0 {
				t.Errorf("%d sources, at_least %d: %s, reads %q", count, n, rejection.Message, log.read())
			}
		}
	}
}

// One identity read for each source: under dev, Bitcoin regtest configured
// with two sources under all, Stellar testnet with two under first, and a
// configuration of Bitcoin testnet, which dev does not pin, with a third;
// beside them an Esplora server given for no network that shows mainnet,
// and a Horizon server given for no network. Each of the four configured
// sources of a pinned network and the Esplora server is read exactly once;
// the testnet configuration's source and the Horizon server, which can
// serve nothing since Stellar's one pinned network has a configuration,
// are never read and have no record; no default is read, since every pin
// has a source. Mutants killed: chainPlan's reads listing a configured
// network's sources twice (each is read twice); planChain planning a
// configuration of a network the run does not pin (e3 is read); planChain
// keeping the sources given for no network when every pinned network of
// their chain has a configuration (the Horizon server is read and
// recorded); a default read for a network with a configuration (Stellar
// testnet's default is read).
func TestSources_OneIdentityReadPerSource(t *testing.T) {
	floatingEsplora := Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://floating-esplora.truestamp.example"}
	floatingHorizon := Source{Chain: ChainStellar, Provider: source.Horizon, URL: "https://floating-horizon.truestamp.example"}
	values := map[string]string{
		e1: regtestGenesisHash, e2: regtestGenesisHash, e3: testnetGenesisHash, h1: testnetPassphrase, h2: testnetPassphrase,
		floatingEsplora.URL: ProductionGenesisHash, floatingHorizon.URL: testnetPassphrase,
	}
	var log readLog
	cfg := online(t, "dev", log.through(valuesReader(values)), floatingEsplora, floatingHorizon)
	cfg.NetworkSources = []NetworkSources{
		configured(ChainBitcoin, "regtest", SourcePolicy{Kind: SourcePolicyAll}, e1, e2),
		configured(ChainStellar, "testnet", SourcePolicy{Kind: SourcePolicyFirst}, h1, h2),
		configured(ChainBitcoin, "testnet", SourcePolicy{Kind: SourcePolicyFirst}, e3),
	}
	run, rejection := configure(t, cfg)
	if rejection != nil {
		t.Fatal(rejection)
	}
	reads := log.read()
	for _, u := range []string{e1, e2, h1, h2, floatingEsplora.URL} {
		if timesRead(reads, u) != 1 {
			t.Errorf("%s read %d times", u, timesRead(reads, u))
		}
	}
	if len(reads) != 5 {
		t.Errorf("reads %q, want each of five sources once", reads)
	}
	var recorded []string
	for _, r := range run.Sources {
		recorded = append(recorded, r.Source.URL+" "+pinsOf(r))
	}
	want := []string{
		e1 + " commitment:regtest", e2 + " commitment:regtest", floatingEsplora.URL + " entropy:mainnet",
		h1 + " commitment:testnet entropy:testnet", h2 + " commitment:testnet entropy:testnet",
	}
	if !slices.Equal(recorded, want) {
		t.Fatalf("records %q\nwant    %q", recorded, want)
	}
	if entropy, _ := run.SourcesFor(PinEntropy, ChainBitcoin); setOf(entropy) != "mainnet first 1: "+floatingEsplora.URL+" confirmed operator  entropy:mainnet" {
		t.Fatalf("the Bitcoin entropy pin: %q", setOf(entropy))
	}
}

// A source with no usable identity answer, with several sources (SPEC.md
// section 9, "Which pin a source serves"): a source configured for a
// network stays among that network's sources, unreachable with a reason,
// serving its pins, and no default is read in its place, so an outage is
// never configuration_error. Under dev, Bitcoin regtest's two sources
// under all both give no usable answer, and the run passes with both in
// the set, mainnet's entropy pin keeping its default; at_least 2 of a
// source that gives no usable answer and one that confirms passes, the
// two in order. Beside a regtest configuration whose one source gives no
// usable answer, a Bitcoin Core node given for no network that gives none
// either serves mainnet, which no other source serves, so mainnet's
// default is not read, and it is not among regtest's sources. Under
// staging, Bitcoin mainnet configured with one source that gives no usable
// answer keeps it, and an Esplora server given for no network that shows
// regtest serves regtest. Mutants killed: assign leaving a configured
// source with no usable answer out of its network's sources (regtest has
// one source, or none); confirmSources taking a configured network whose
// sources all gave no usable answer as unserved (regtest has no default,
// so the run is configuration_error, and mainnet's default is read); the
// unreachable source given for no network serving a network with a
// configuration (regtest's set holds the node).
func TestSources_ANetworkSourceWithNoUsableAnswerStaysAmongItsSources(t *testing.T) {
	node := Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: "https://node.truestamp.example"}
	floating := Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://floating.truestamp.example"}
	for _, c := range []struct {
		name, environment string
		values            map[string]string
		sources           []Source
		configured        NetworkSources
		want              map[string]string
		notRead           []string
	}{
		{
			"both of all unreachable", "dev", map[string]string{e1: "", e2: ""}, nil,
			configured(ChainBitcoin, "regtest", SourcePolicy{Kind: SourcePolicyAll}, e1, e2),
			map[string]string{
				PinCommitment: "regtest all 2: " + e1 + " unreachable operator regtest commitment:regtest, " + e2 + " unreachable operator regtest commitment:regtest",
				PinEntropy:    "mainnet first 1: https://blockstream.info/api confirmed default mainnet entropy:mainnet",
			}, nil,
		},
		{
			"at_least 2, one unreachable", "dev", map[string]string{e1: "", e2: regtestGenesisHash}, nil,
			configured(ChainBitcoin, "regtest", SourcePolicy{Kind: SourcePolicyAtLeast, N: 2}, e1, e2),
			map[string]string{
				PinCommitment: "regtest at_least 2 2: " + e1 + " unreachable operator regtest commitment:regtest, " + e2 + " confirmed operator regtest commitment:regtest",
			}, nil,
		},
		{
			"a node for no network beside a configuration", "dev", map[string]string{e1: "", node.URL: ""}, []Source{node},
			configured(ChainBitcoin, "regtest", SourcePolicy{Kind: SourcePolicyFirst}, e1),
			map[string]string{
				PinCommitment: "regtest first 1: " + e1 + " unreachable operator regtest commitment:regtest",
				PinEntropy:    "mainnet first 1: " + node.URL + " unreachable operator  entropy:mainnet",
			}, []string{"https://blockstream.info/api"},
		},
		{
			"staging, mainnet configured", "staging", map[string]string{e1: "", floating.URL: regtestGenesisHash}, []Source{floating},
			configured(ChainBitcoin, "mainnet", SourcePolicy{Kind: SourcePolicyFirst}, e1),
			map[string]string{
				PinCommitment: "regtest first 1: " + floating.URL + " confirmed operator  commitment:regtest",
				PinEntropy:    "mainnet first 1: " + e1 + " unreachable operator mainnet entropy:mainnet",
			}, []string{"https://blockstream.info/api"},
		},
	} {
		var log readLog
		cfg := online(t, c.environment, log.through(valuesReader(c.values)), c.sources...)
		cfg.NetworkSources = []NetworkSources{c.configured}
		run, rejection := configure(t, cfg)
		if rejection != nil {
			t.Fatalf("%s: %v", c.name, rejection)
		}
		for pin, want := range c.want {
			set, _ := run.SourcesFor(pin, ChainBitcoin)
			if setOf(set) != want {
				t.Errorf("%s: the %s pin: %q\nwant %q", c.name, pin, setOf(set), want)
			}
			for _, r := range set.Sources {
				if (r.Status == SourceUnreachable) != (r.Reason != "") {
					t.Errorf("%s: %+v", c.name, r)
				}
			}
		}
		for _, u := range c.notRead {
			if timesRead(log.read(), u) != 0 {
				t.Errorf("%s: %s was read", c.name, u)
			}
		}
	}
}

// One source under every policy is the run of a single source (SPEC.md
// section 10.4, "Several sources"): with Bitcoin regtest and Stellar
// testnet each configured with one source, confirmed or giving no usable
// answer, under first, at_least 1 and all, CONFIGURATION's row, its message
// and its values, and every record equal those of the same source given
// for no network, and each set's required is 1. Mutants killed:
// checkPolicies refusing an n equal to the number of sources (at_least 1
// of 1 is refused); a configured source with no usable answer recorded
// differently from one given for no network (its values differ); Required
// under all giving the count plus one.
func TestSources_OneSourceUnderEveryPolicyIsTheSingleSourceRun(t *testing.T) {
	for _, stellarValue := range []string{testnetPassphrase, ""} {
		values := map[string]string{e1: regtestGenesisHash, h1: stellarValue}
		single := online(t, "test", valuesReader(values),
			Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: e1}, Source{Chain: ChainStellar, Provider: source.Horizon, URL: h1})
		base, rejection := configure(t, single)
		if rejection != nil {
			t.Fatal(rejection)
		}
		wantRow, _ := json.Marshal([]any{base.Rows[0].Message, base.Rows[0].Values})
		for _, policy := range []SourcePolicy{{Kind: SourcePolicyFirst}, {Kind: SourcePolicyAtLeast, N: 1}, {Kind: SourcePolicyAll}} {
			cfg := online(t, "test", valuesReader(values))
			cfg.NetworkSources = []NetworkSources{
				configured(ChainBitcoin, "regtest", policy, e1),
				configured(ChainStellar, "testnet", policy, h1),
			}
			run, rejection := configure(t, cfg)
			if rejection != nil {
				t.Fatalf("%s: %v", policy, rejection)
			}
			gotRow, _ := json.Marshal([]any{run.Rows[0].Message, run.Rows[0].Values})
			if string(gotRow) != string(wantRow) {
				t.Errorf("%s, Stellar %q: CONFIGURATION\n got %s\nwant %s", policy, stellarValue, gotRow, wantRow)
			}
			for i, r := range run.Sources {
				b := base.Sources[i]
				if r.Status != b.Status || r.Identity != b.Identity || r.Reason != b.Reason || pinsOf(r) != pinsOf(b) || r.Source.URL != b.Source.URL {
					t.Errorf("%s: record %d %+v, single %+v", policy, i, r, b)
				}
			}
			for _, chain := range chains {
				if set, ok := run.SourcesFor(PinCommitment, chain); !ok || set.Required() != 1 || len(set.Sources) != 1 || set.Policy != policy {
					t.Errorf("%s: %s's set %+v", policy, chain, set)
				}
			}
		}
	}
}

// A network configuration no run can take is an invalid configuration,
// whose error never quotes a source's URL or host, though a token is
// planted in the path of every source: NIST, which is one source; no
// chain; a network not of its chain; no source; a policy that is none of
// first, at_least and all, the empty one included; a number under first
// or all; a source of another chain or configured for another network; a
// source checkSource refuses, one with user information among them; one
// server twice, as given, with a trailing slash, with its host in
// capitals, with its scheme's default port (443 for https, 80 for http)
// on one side, or with a trailing dot on its host; and two configurations
// of one network. An error about one source names its place in the
// network's order, from 1. Allowed: an Esplora server and a Bitcoin Core
// node at one URL, one host at two ports or two paths, at_least with any
// n (its range is configuration_error at step 0), and a source that names
// its own network. Mutants killed, one for each refusal:
// checkNetworkSources without its network check (a network of another
// chain passes), its count, its policy switch's default or its check of a
// number under first or all, its check of a source's chain or network,
// its call of sourceFault, or its check of one server twice; sameServer
// comparing URLs as given (the trailing slash and the capitals pass);
// sameServer comparing the host and port as written (the default ports
// and the trailing dot pass); sameServer with no default port (the
// default ports pass); sameServer without the trailing dot dropped (the
// trailing dot passes); sameServer ignoring the port or the path (two
// ports or two paths are refused); sameServer ignoring the provider
// (Esplora and Core at one URL are refused); Validate without its check
// of two configurations of one network; an error about one source naming
// no place, or a place counted from 0; the URL quoted in the error of one
// server twice, of a source of another chain or network, or of a source
// sourceFault refuses (the token is printed).
func TestConfig_NetworkSourcesRefusals(t *testing.T) {
	const secret = "planted-secret-77c1"
	const token = "/tok-REFUSED-8b4d"
	first, all := SourcePolicy{Kind: SourcePolicyFirst}, SourcePolicy{Kind: SourcePolicyAll}
	esploraAt := func(u string) Source { return Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: u} }
	with := func(n NetworkSources, edit func(*NetworkSources)) NetworkSources {
		n.Sources = slices.Clone(n.Sources)
		edit(&n)
		return n
	}
	good := configured(ChainBitcoin, "regtest", all, e1+token, e2+token)
	second := func(u string) func(*NetworkSources) { return func(n *NetworkSources) { n.Sources[1].URL = u } }
	const atSecond, bothFirstAndSecond = "source 2 of Bitcoin's regtest network", "sources 1 and 2 of Bitcoin's regtest network name one esplora server"
	refused := map[string]struct {
		networks []NetworkSources
		names    string
	}{
		"NIST":                       {[]NetworkSources{{Chain: "nist", Network: "mainnet", Policy: first, Sources: []Source{esploraAt(e1 + token)}}}, ""},
		"no chain":                   {[]NetworkSources{with(good, func(n *NetworkSources) { n.Chain = "" })}, ""},
		"a network of another chain": {[]NetworkSources{with(good, func(n *NetworkSources) { n.Network = "public" })}, ""},
		"no source":                  {[]NetworkSources{with(good, func(n *NetworkSources) { n.Sources = nil })}, ""},
		"an unknown policy":          {[]NetworkSources{with(good, func(n *NetworkSources) { n.Policy = SourcePolicy{Kind: "most"} })}, ""},
		"no policy":                  {[]NetworkSources{with(good, func(n *NetworkSources) { n.Policy = SourcePolicy{} })}, ""},
		"a number under first":       {[]NetworkSources{with(good, func(n *NetworkSources) { n.Policy = SourcePolicy{Kind: SourcePolicyFirst, N: 1} })}, ""},
		"a number under all":         {[]NetworkSources{with(good, func(n *NetworkSources) { n.Policy = SourcePolicy{Kind: SourcePolicyAll, N: 2} })}, ""},
		"a source of another chain": {[]NetworkSources{with(good, func(n *NetworkSources) {
			n.Sources[1] = Source{Chain: ChainStellar, Provider: source.Horizon, URL: h1 + token}
		})}, atSecond + " is configured for another chain or network"},
		"a source of another network": {[]NetworkSources{with(good, func(n *NetworkSources) { n.Sources[1].Network = "mainnet" })},
			atSecond + " is configured for another chain or network"},
		"user information": {[]NetworkSources{with(good, second("https://u:"+secret+"@e.truestamp.example"+token))}, atSecond + ": the esplora URL:"},
		"credentials on Esplora": {[]NetworkSources{with(good, func(n *NetworkSources) {
			n.Sources[1].Credentials = source.NewCredentials("u", secret)
		})}, atSecond + ": only a Bitcoin Core node takes RPC credentials"},
		"one server twice":                   {[]NetworkSources{with(good, second(e1+token))}, bothFirstAndSecond},
		"one server twice, a trailing slash": {[]NetworkSources{with(good, second(e1+token+"/"))}, bothFirstAndSecond},
		"one server twice, capitals":         {[]NetworkSources{with(good, second("HTTPS://E1.TRUESTAMP.EXAMPLE"+token))}, bothFirstAndSecond},
		"one server twice, https's default port": {[]NetworkSources{with(good, second("https://e1.truestamp.example:443"+token+"/"))},
			bothFirstAndSecond},
		"one server twice, http's default port": {[]NetworkSources{with(good, func(n *NetworkSources) {
			n.Sources[0].URL, n.Sources[1].URL = "http://e1.truestamp.example"+token, "http://e1.truestamp.example:80"+token
		})}, bothFirstAndSecond},
		"one server twice, a trailing dot": {[]NetworkSources{with(good, second("https://e1.truestamp.example."+token))}, bothFirstAndSecond},
		"two configurations of one network": {[]NetworkSources{good, configured(ChainBitcoin, "regtest", first, e3+token)},
			"two configurations of the sources of Bitcoin's regtest network"},
	}
	for name, c := range refused {
		cfg, _ := NewConfig("test")
		cfg.NetworkSources = c.networks
		err := cfg.Validate()
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if text := err.Error(); strings.Contains(text, secret) || strings.Contains(text, token) || strings.Contains(text, ".example") ||
			!strings.Contains(text, c.names) {
			t.Errorf("%s: %v, want ...%s...", name, err, c.names)
		}
	}
	allowed := map[string]NetworkSources{
		"Esplora and Core at one URL": with(good, func(n *NetworkSources) {
			n.Sources[1] = Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: e1 + token}
		}),
		"one host at two ports": with(good, second("https://e1.truestamp.example:8443"+token)),
		"one host at two paths": with(good, second(e1+token+"/two")),
		"at_least 9 of two":     with(good, func(n *NetworkSources) { n.Policy = SourcePolicy{Kind: SourcePolicyAtLeast, N: 9} }),
		"at_least 0":            with(good, func(n *NetworkSources) { n.Policy = SourcePolicy{Kind: SourcePolicyAtLeast} }),
		"its own network named": with(good, func(n *NetworkSources) { n.Sources[0].Network = "regtest" }),
	}
	for name, n := range allowed {
		cfg, _ := NewConfig("test")
		cfg.NetworkSources = []NetworkSources{n, configured(ChainBitcoin, "mainnet", first, e3)}
		if err := cfg.Validate(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Several Bitcoin Core nodes for one network, each with its own RPC
// credentials and an API token in its URL's path, given through
// Options.NetworkSources: each node is read once, with its own
// credentials as HTTP Basic and its own path; no planted user, password
// or Basic encoding appears in CONFIGURATION's values or message, the
// records, the source sets, the configuration printed with every fmt verb
// and as JSON, or the mismatch that the second node's production hash
// gives, which names the node's place among regtest's sources; and no
// token appears in what a report or a message prints, CONFIGURATION's
// values and message and the mismatch's message, each of which prints a
// node's URL as its scheme://host:port. Options.Config gives the run its
// own copy of the configuration, and OptionsFromCommand sets none.
// Mutants killed: Options.Config dropping NetworkSources (regtest has no
// source, and the run is configuration_error); Options.Config sharing the
// options' slices (a later change to the options moves the run's source);
// sourceValues printing the whole URL (the token is printed); mismatch
// printing the whole URL; the credentials of the network's first node
// sent to every node.
func TestSources_SeveralCoreNodesKeepTheirCredentialsAndPrintNoPath(t *testing.T) {
	users := []string{"rpcuser-one-4d1f", "rpcuser-two-9a3c"}
	secrets := []string{"planted-secret-one-61b0", "planted-secret-two-e27d"}
	tokens := []string{"tok-ONE-c41a", "tok-TWO-5f3e"}
	for _, mismatch := range []bool{false, true} {
		var nodes []*node
		var sources []Source
		for i := range users {
			value := regtestGenesisHash
			if mismatch && i == 1 {
				value = ProductionGenesisHash
			}
			n := pathAnswering(newNode(t, value))
			nodes = append(nodes, n)
			sources = append(sources, Source{Chain: ChainBitcoin, Provider: source.BitcoinCore,
				URL: n.srv.URL + "/" + tokens[i] + "/", Credentials: source.NewCredentials(users[i], secrets[i])})
		}
		horizonNode := newNode(t, testnetPassphrase)
		options := Options{Environment: "test", NoKeyring: true, Horizon: horizonNode.srv.URL, NetworkSources: []NetworkSources{
			{Chain: ChainBitcoin, Network: "regtest", Policy: SourcePolicy{Kind: SourcePolicyAll}, Sources: sources},
		}}
		cfg, err := options.Config(nil, loopbackIdentities(t))
		if err != nil {
			t.Fatal(err)
		}
		given := sources[0].URL
		options.NetworkSources[0].Sources[0].URL = "https://moved.truestamp.example"
		if cfg.NetworkSources[0].Sources[0].URL != given {
			t.Fatalf("the run's source moved with the options: %s", cfg.NetworkSources[0].Sources[0].URL)
		}
		run, rejection, err := Configure(cfg)
		if err != nil {
			t.Fatal(err)
		}
		for i, n := range nodes {
			basic := base64.StdEncoding.EncodeToString([]byte(users[i] + ":" + secrets[i]))
			if n.requests.Load() != 1 || n.auth.Load() != "Basic "+basic {
				t.Fatalf("node %d: %d requests, Authorization %v", i+1, n.requests.Load(), n.auth.Load())
			}
		}
		var printed []string
		if mismatch {
			host, _ := url.Parse(nodes[1].srv.URL)
			if rejection == nil || !strings.Contains(rejection.Message, "source at http://"+host.Host+", source 2 of the 2 sources of Bitcoin's regtest network, gives the hash") {
				t.Fatalf("the second node's mismatch: %v", rejection)
			}
			printed = append(printed, rejection.Message)
		} else {
			if rejection != nil {
				t.Fatal(rejection)
			}
			values, _ := json.Marshal(run.Rows[0].Values)
			sets, _ := json.Marshal(run.SourceSets)
			printed = append(printed, string(values), string(sets), run.Rows[0].Message,
				fmt.Sprintf("%v %+v %#v", run.Sources, run.SourceSets, *run))
			for _, n := range nodes {
				if !strings.Contains(string(values), `"url":"`+n.srv.URL+`"`) {
					t.Fatalf("the values do not print %s: %s", n.srv.URL, values)
				}
			}
			if set, _ := run.SourcesFor(PinCommitment, ChainBitcoin); len(set.Sources) != 2 || set.Required() != 2 {
				t.Fatalf("regtest's sources %+v", set)
			}
		}
		configJSON, _ := json.Marshal(cfg.NetworkSources)
		printed = append(printed, string(configJSON), fmt.Sprintf("%v %+v %#v %s %q %x", cfg, cfg, cfg, cfg.NetworkSources, cfg.NetworkSources, cfg.NetworkSources))
		text := strings.Join(printed, "\n")
		report := printed[0]
		if !mismatch {
			report = printed[0] + "\n" + printed[2]
		}
		for i := range users {
			basic := base64.StdEncoding.EncodeToString([]byte(users[i] + ":" + secrets[i]))
			for _, planted := range []string{users[i], secrets[i], basic} {
				if strings.Contains(text, planted) {
					t.Fatalf("%q is printed:\n%s", planted, text)
				}
			}
			if strings.Contains(report, tokens[i]) {
				t.Fatalf("the token %q is printed:\n%s", tokens[i], report)
			}
		}
	}
	clearTruestampEnv(t)
	_, options := loadOptions(t, "", "--environment", "test", "--horizon", "https://h.example", "--esplora", "https://e.example", "--bitcoin-core", "http://127.0.0.1:18443")
	if options.NetworkSources != nil {
		t.Fatalf("OptionsFromCommand set %+v", options.NetworkSources)
	}
}

// SourcesFor finds a pin's set by its chain and the run's network for that
// pin, with all of its sources in their order: under staging, Bitcoin's
// entropy pin is mainnet's set of two under all, and, with its commitment
// pin moved to testnet, that pin is Bitcoin testnet's set, not Stellar
// testnet's, listed before it; a pin no set serves has none. Mutants
// killed: SourcesFor reading the commitment network for every pin (the
// entropy pin finds testnet's set); SourcesFor matching the network alone
// (the commitment pin finds Stellar's set).
func TestRun_SourcesForFindsThePinsSet(t *testing.T) {
	cfg, err := NewConfig("staging")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Networks.Commitment.Bitcoin = "testnet"
	record := func(chain, u, network string) SourceRecord {
		return SourceRecord{Source: Source{Chain: chain, Provider: source.Esplora, URL: u, Network: network}}
	}
	run := &Run{Config: cfg, SourceSets: []SourceSet{
		{Chain: ChainStellar, Network: "testnet", Policy: SourcePolicy{Kind: SourcePolicyFirst}, Sources: []SourceRecord{record(ChainStellar, h1, "testnet")}},
		{Chain: ChainBitcoin, Network: "testnet", Policy: SourcePolicy{Kind: SourcePolicyFirst}, Sources: []SourceRecord{record(ChainBitcoin, e3, "testnet")}},
		{Chain: ChainBitcoin, Network: "mainnet", Policy: SourcePolicy{Kind: SourcePolicyAll}, Sources: []SourceRecord{
			record(ChainBitcoin, e1, "mainnet"), record(ChainBitcoin, e2, "mainnet"),
		}},
	}}
	if set, ok := run.SourcesFor(PinEntropy, ChainBitcoin); !ok || set.Network != "mainnet" || set.Required() != 2 || len(set.Sources) != 2 ||
		set.Sources[0].Source.URL != e1 || set.Sources[1].Source.URL != e2 {
		t.Fatalf("the entropy pin's set: %+v, %v", set, ok)
	}
	if set, ok := run.SourcesFor(PinCommitment, ChainBitcoin); !ok || set.Chain != ChainBitcoin || set.Network != "testnet" ||
		len(set.Sources) != 1 || set.Sources[0].Source.URL != e3 {
		t.Fatalf("the commitment pin's set: %+v, %v", set, ok)
	}
	run.SourceSets = run.SourceSets[1:]
	if set, ok := run.SourcesFor(PinCommitment, ChainStellar); ok {
		t.Fatalf("Stellar's pin with no set gave %+v", set)
	}
}
