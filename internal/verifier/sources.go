// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file is the part of step 0 (CONFIGURATION, SPEC.md section 10.2)
// that concerns the Stellar and Bitcoin sources: which sources each
// network pin has, in what order and under what source policy, and each
// source's identity, confirmed both ways before the file is read (SPEC.md
// sections 9 (What a verifier pins), "Sources", "Several sources for one
// network" and "Which pin a source serves", 10.2, step 0, and 10.4 (Chain
// and source reads), "Several sources"). For the CLI's sources:
//
//   - the CLI has a compiled-in default for each network but regtest
//     ([DefaultSource]); the operator may give sources configured for no
//     network ([Config.Sources]: a Horizon server, an Esplora server and a
//     Bitcoin Core node), each serving, when it answers its identity read,
//     only the pins of its chain of networks with no configuration that
//     the answer shows, so that one whose answer fits only configured
//     networks serves nothing, and may configure a network's sources, one
//     or more in an order, with a source policy, first, at_least n or all
//     ([Config.NetworkSources], [SourcePolicy]); a network with no such
//     configuration has, under the policy first, its compiled-in default
//     or the sources configured for no network that serve it, so a run
//     with one source per network is the run of a single source. SPEC.md
//     section 9 gives such a network one source, so of several sources
//     configured for no network that may serve it, only the first in the
//     order of [Config.Sources] serves it (step 4 below);
//   - each source has its own identity read, made once, all of them at
//     once, before anything else is read from any source;
//   - an identity read with no usable answer records the source
//     unreachable, and the run reads nothing else from it; the rows it
//     serves are made by CHAIN and SOURCES, which combine its outcome
//     with the network's other sources' under the policy, so this file
//     records only the outcome, in [Run.Sources], and each pinned
//     network's sources and policy, the source map those steps read, in
//     [Run.SourceSets];
//   - configuration_error, a bare rejection: a source whose identity read
//     gives a value step 0 forbids for the network it serves, whatever the
//     policy and whatever the network's other sources give; a pinned
//     network with no source; and, in every run, online or offline, a
//     policy at_least n whose n is 0 or above the number of its network's
//     sources ([checkPolicies]);
//   - a source's URL is printed, in a message and in CONFIGURATION's
//     values, as its scheme, host and port alone ([printedURL]), and no
//     unreachable reason names it, since a hosted node's provider may
//     carry an API key in the path and a report is made to be shared; the
//     reads use the whole URL. No source takes credentials from the
//     command line: a Bitcoin Core node's credentials are a
//     source.Credentials, which prints masked.
//
// A source with no usable identity answer serves the pins SPEC.md section
// 9, "Which pin a source serves", gives it: a source configured for one
// network, a compiled-in default or a source of the network's
// configuration, serves that network's pins and stays among its sources,
// so each of their rows it makes is skip source_unreachable; a source
// configured for no network serves each pin of its chain the run pins that
// no other source serves, and of two such sources the first in the order
// of [Config.Sources], the order of the verifier's source options, serves
// it; and no compiled-in default is read in such a source's place, so an
// outage is never configuration_error (SPEC.md section 10.2, step 0).
//
// NIST's Randomness Beacon is one source, with no network and no identity
// read (SPEC.md section 9): SOURCES reads it at its one compiled-in
// address, and no configuration names another.
//
// The steps, for an online run:
//
//  1. each chain's pinned networks, from the run's commitment and entropy
//     pins in that order; a network's configuration whose network the run
//     does not pin serves nothing and is not read, and neither is a
//     source configured for no network when every pinned network of its
//     chain has a configuration;
//  2. before any read: a pinned network with no configuration that no
//     source configured for no network can serve and that has no default
//     has no source;
//  3. the identity read of every source of every pinned network's
//     configuration and of every source configured for no network, all at
//     once;
//  4. each source of a network's configuration that answered is held to
//     that network alone: a value the network forbids is
//     configuration_error, and the message names the source's place in
//     the network's order when the network has several; then each source
//     configured for no network that answered serves the pinned networks
//     of its chain with no configuration that its answer allows, a
//     production value the chain's production network and any other value
//     its other networks, but none that such a source before it in the
//     order of [Config.Sources] serves, so that such a network has one
//     source (SPEC.md section 9). One whose answer only pinned
//     networks with a configuration, or networks a source before it
//     serves, allow serves nothing and is recorded with no pins, whatever
//     other networks are open, and one whose answer no pinned network of
//     its chain allows is configuration_error; SPEC.md section 10.2, step
//     0, lists no other case, and the first is the CLI's own reading of
//     section 9; then each source configured for no network with no
//     usable answer, in that order, serves every such network it may
//     serve that no source before it serves, so under dev
//     or staging, of an Esplora server and a Bitcoin Core node that both
//     give no usable answer, the Esplora server serves regtest and mainnet
//     and the node serves nothing;
//  5. each pinned network still unserved: its default, if it has one, or
//     no source;
//  6. every such default's identity read, all at once, each checked
//     against its one network.
//
// So step 0 waits for at most two read deadlines, whatever the sources do.

package verifier

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// SourceReadDeadline bounds each identity read and each chain read of
// [DefaultChainReader], the whole read, every call it makes included (a
// best-chain block at a height takes two calls, which end within it
// together), as the Truestamp application bounds each of its verification
// reads.
const SourceReadDeadline = 9 * time.Second

// The production values of the identity checks (SPEC.md section 10.2,
// step 0). No test network's value is pinned.
const (
	// ProductionPassphrase is the network passphrase of Stellar's
	// production network, public.
	ProductionPassphrase = "Public Global Stellar Network ; September 2015"
	// ProductionGenesisHash is the hash of the block at height 0 of
	// Bitcoin's production network, mainnet.
	ProductionGenesisHash = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
)

// The two kinds of network pin (SPEC.md section 9).
const (
	PinCommitment = "commitment"
	PinEntropy    = "entropy"
)

// The chains a source serves.
const (
	ChainBitcoin = "bitcoin"
	ChainStellar = "stellar"
)

// chains are the chains with sources, in the order step 0 confirms them.
var chains = []string{ChainBitcoin, ChainStellar}

// productionNetwork is each chain's production network.
var productionNetwork = map[string]string{ChainBitcoin: "mainnet", ChainStellar: "public"}

// productionIdentity is each chain's production identity value.
var productionIdentity = map[string]string{ChainBitcoin: ProductionGenesisHash, ChainStellar: ProductionPassphrase}

// Source is one Stellar or Bitcoin source a run may read.
type Source struct {
	// Chain is stellar or bitcoin.
	Chain string
	// Provider is the kind of server: Horizon for Stellar, Esplora or
	// Bitcoin Core for Bitcoin.
	Provider source.Provider
	// URL is the server's base URL, or a Bitcoin Core node's JSON-RPC URL.
	// It never carries user information. The reads use it whole, and a
	// message or a report prints it as [printedURL] does, with no path.
	URL string
	// Credentials are a Bitcoin Core node's RPC credentials; they print
	// masked.
	Credentials source.Credentials
	// Network is the one network the source is configured for, a
	// compiled-in default's or that of the [NetworkSources] it is one of,
	// or "" for a source configured for no network, which, when it answers
	// its identity read, serves only the pins of its chain of networks with
	// no configuration that the answer shows. Step 0 sets it on each source
	// of a network's configuration.
	Network string
}

// SourcePolicyKind names a source policy (SPEC.md section 9, "Several
// sources for one network").
type SourcePolicyKind string

// The source policies.
const (
	// SourcePolicyFirst: the sources are asked in their order, and a row
	// takes the first conclusive outcome.
	SourcePolicyFirst SourcePolicyKind = "first"
	// SourcePolicyAtLeast: every source is asked, and a row passes or
	// fails when n of them agree.
	SourcePolicyAtLeast SourcePolicyKind = "at_least"
	// SourcePolicyAll: at_least the number of sources.
	SourcePolicyAll SourcePolicyKind = "all"
)

// SourcePolicy is a network's source policy (SPEC.md sections 9 and 10.4,
// "Several sources"): first, at_least n with N its n, or all. Only the
// operator sets it, never the document. N is 0 under first and all; under
// at_least, an N that is 0 or above the number of the network's sources is
// configuration_error at step 0.
type SourcePolicy struct {
	Kind SourcePolicyKind
	N    int
}

// Required is the policy's n for a network of count sources, the number of
// conclusive outcomes it asks for (SPEC.md section 10.4, "What a row
// read"): 1 under first, N under at_least and count under all.
func (p SourcePolicy) Required(count int) int {
	switch p.Kind {
	case SourcePolicyAtLeast:
		return p.N
	case SourcePolicyAll:
		return count
	}
	return 1
}

// String writes the policy as a message names it: first, at_least n or
// all.
func (p SourcePolicy) String() string {
	if p.Kind == SourcePolicyAtLeast {
		return fmt.Sprintf("%s %d", p.Kind, p.N)
	}
	return string(p.Kind)
}

// NetworkSources is the operator's configuration of one Stellar or Bitcoin
// network's sources (SPEC.md section 9, "Several sources for one
// network"): the chain and network, the policy, and the sources, one or
// more, in the order the operator gives them, each configured for that
// network. Each has its own identity read, held to that network (SPEC.md
// section 10.2, step 0). NIST's Randomness Beacon is one source, so no
// configuration names it.
type NetworkSources struct {
	Chain   string
	Network string
	Policy  SourcePolicy
	Sources []Source
}

// IdentityReader reads a source's identity live (SPEC.md section 10.2,
// step 0): Stellar's network passphrase, or the hash of Bitcoin's block at
// height 0, or an error for a read with no usable answer. It is the seam
// every identity read goes through: [DefaultIdentityReader] reads over
// HTTP, and a conformance test answers from a case's replay file.
type IdentityReader func(Source) (string, error)

// DefaultIdentityReader returns the reader of each source's identity over
// HTTP, through the default transport: each read builds its own
// SingleRead, within [SourceReadDeadline], when it is made.
func DefaultIdentityReader() IdentityReader {
	return func(s Source) (string, error) {
		reads := newSingleRead(nil, SourceReadDeadline)
		return source.Identity(context.Background(), reads, s.Provider, s.URL, s.Credentials)
	}
}

// defaultSources are the compiled-in sources, by chain and network.
// Regtest has none (SPEC.md section 9).
var defaultSources = map[[2]string]Source{
	{ChainStellar, "public"}:  {Chain: ChainStellar, Provider: source.Horizon, URL: "https://horizon.stellar.org", Network: "public"},
	{ChainStellar, "testnet"}: {Chain: ChainStellar, Provider: source.Horizon, URL: "https://horizon-testnet.stellar.org", Network: "testnet"},
	{ChainBitcoin, "mainnet"}: {Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://blockstream.info/api", Network: "mainnet"},
	{ChainBitcoin, "testnet"}: {Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://blockstream.info/testnet/api", Network: "testnet"},
}

// DefaultSource returns the compiled-in source of network of chain, and
// whether there is one.
func DefaultSource(chain, network string) (Source, bool) {
	s, ok := defaultSources[[2]string{chain, network}]
	return s, ok
}

// SourceFrom says where a run's source came from.
type SourceFrom string

// Where a source comes from.
const (
	// FromDefault: a compiled-in default.
	FromDefault SourceFrom = "default"
	// FromOperator: a source the operator pins, or one a conformance
	// case's replay file stands for.
	FromOperator SourceFrom = "operator"
)

// SourceStatus is the outcome of a source's identity read.
type SourceStatus string

// The outcomes of an identity read that is no configuration_error.
const (
	// SourceConfirmed: the source answered with a value its pins allow;
	// a source configured for no network whose answer only pinned networks
	// with a configuration allow is confirmed with no pins.
	SourceConfirmed SourceStatus = "confirmed"
	// SourceUnreachable: the source gave no usable answer, and the run
	// reads nothing else from it.
	SourceUnreachable SourceStatus = "unreachable"
)

// ServedPin is one network pin a source serves: its kind, commitment or
// entropy, and its network.
type ServedPin struct {
	Pin     string
	Network string
}

// SourceRecord is one source of an online run and its identity read.
type SourceRecord struct {
	Source Source
	From   SourceFrom
	Status SourceStatus
	// Identity is the value the source gave, when confirmed.
	Identity string
	// Reason says why the source gave no usable answer, when unreachable.
	Reason string
	// Pins are the network pins the source serves, none for a source that
	// serves nothing.
	Pins []ServedPin
	// FetchedAt is when the identity read's answer, or its failure, came
	// back, by the verifier's clock (SPEC.md section 10.4, "What a row
	// read"): a row whose source was asked nothing else gives it. The read
	// is always live, so it is never cached.
	FetchedAt time.Time
}

// SourceSet is one pinned network's sources, in their order, each with its
// identity read's outcome, and its policy: what CHAIN and SOURCES read
// that network from, each row's rule made once for each source and the
// outcomes combined under the policy (SPEC.md section 10.4, "Several
// sources"). A network with no configuration has the policy first and
// either its compiled-in default or the sources configured for no network
// that serve it, one or more, in the order of [Config.Sources]: SPEC.md
// section 9 gives such a network one source, and several is the CLI's own
// rule, which [assign] applies.
type SourceSet struct {
	Chain   string
	Network string
	Policy  SourcePolicy
	Sources []SourceRecord
}

// Required is the set's n: the number of conclusive outcomes its policy
// asks for (SPEC.md section 10.4, "What a row read").
func (s SourceSet) Required() int { return s.Policy.Required(len(s.Sources)) }

// SourcesFor returns the sources of the run's pin (commitment or entropy)
// of chain, all of them in their order under the network's policy, and
// whether it has any: every pin of an online run that passed CONFIGURATION
// has one or more, and an offline run has none.
func (r *Run) SourcesFor(pin, chain string) (SourceSet, bool) {
	network := r.Config.pinNetwork(pin, chain)
	for _, set := range r.SourceSets {
		if set.Chain == chain && set.Network == network {
			return set, true
		}
	}
	return SourceSet{}, false
}

// pinNetwork is the network of the run's pin of chain.
func (c Config) pinNetwork(pin, chain string) string {
	networks := c.Networks.Commitment
	if pin == PinEntropy {
		networks = c.Networks.Entropy
	}
	if chain == ChainStellar {
		return networks.Stellar
	}
	return networks.Bitcoin
}

// pinsOf returns the pins of chain, commitment then entropy.
func (c Config) pinsOf(chain string) []ServedPin {
	return []ServedPin{
		{Pin: PinCommitment, Network: c.pinNetwork(PinCommitment, chain)},
		{Pin: PinEntropy, Network: c.pinNetwork(PinEntropy, chain)},
	}
}

// networksOf returns the networks chain is pinned to, once each, in the
// order of its pins.
func (c Config) networksOf(chain string) []string {
	var networks []string
	for _, pin := range c.pinsOf(chain) {
		if !slices.Contains(networks, pin.Network) {
			networks = append(networks, pin.Network)
		}
	}
	return networks
}

// pinsFor returns the pins of chain whose network is one of networks.
func (c Config) pinsFor(chain string, networks []string) []ServedPin {
	var pins []ServedPin
	for _, pin := range c.pinsOf(chain) {
		if slices.Contains(networks, pin.Network) {
			pins = append(pins, pin)
		}
	}
	return pins
}

// allows reports whether a source of chain whose identity read gave value
// may serve network: a production value for the production network, and
// any other value for any other network (SPEC.md section 10.2, step 0).
func allows(chain, network, value string) bool {
	return (network == productionNetwork[chain]) == (value == productionIdentity[chain])
}

// identityAnswer is one identity read's outcome, and when it came back.
type identityAnswer struct {
	value string
	err   error
	at    time.Time
}

// readIdentities makes the identity read of each source through reader,
// all at once, and returns their answers in the sources' order, each
// stamped by the verifier's clock when it came back. A nil reader reads
// nothing, so each source gives no usable answer.
func readIdentities(reader IdentityReader, sources []Source) []identityAnswer {
	answers := make([]identityAnswer, len(sources))
	var wg sync.WaitGroup
	for i, s := range sources {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if reader == nil {
				answers[i] = identityAnswer{err: fmt.Errorf("no identity reader"), at: stamp()}
				return
			}
			value, err := reader(s)
			answers[i] = identityAnswer{value: value, err: err, at: stamp()}
		}()
	}
	wg.Wait()
	return answers
}

// checkPolicies is configuration_error for the first of the run's network
// configurations, in their order, whose policy is at_least n with n 0 or
// above the number of its sources (SPEC.md sections 9 and 10.2, step 0),
// or nil. Step 0 makes it in every run, online or offline, before any
// read, and for every configuration, pinned or not, since it concerns the
// configuration alone. The message names no source.
func checkPolicies(cfg Config) *Rejection {
	for _, n := range cfg.NetworkSources {
		if n.Policy.Kind != SourcePolicyAtLeast || (n.Policy.N >= 1 && n.Policy.N <= len(n.Sources)) {
			continue
		}
		if len(n.Sources) == 1 {
			return ConfigurationError(fmt.Sprintf(
				"%s's %s network has one source, under the policy %s, so the run gives no verdict: "+
					"give it the policy first or at_least 1.",
				chainName(n.Chain), n.Network, n.Policy))
		}
		return ConfigurationError(fmt.Sprintf(
			"The sources of %s's %s network have the policy %s, and the network has %d sources, "+
				"so the run gives no verdict: give at_least a number from 1 to %d.",
			chainName(n.Chain), n.Network, n.Policy, len(n.Sources), len(n.Sources)))
	}
	return nil
}

// chainPlan is what step 0 knows of one chain's sources before any read.
type chainPlan struct {
	chain string
	// networks are the chain's pinned networks, in the order of its pins.
	networks []string
	// configured are the operator's configurations of pinned networks, in
	// the order of networks, each source's Network set to its network.
	configured []NetworkSources
	// floating are the sources configured for no network, in the order of
	// [Config.Sources]; none when open is empty.
	floating []Source
	// open are the pinned networks with no configuration, which the
	// floating sources may serve.
	open []string
}

// reads are the plan's sources whose identity step 0 reads first: each
// configured network's sources in order, then the floating sources.
func (p chainPlan) reads() []Source {
	var out []Source
	for _, n := range p.configured {
		out = append(out, n.Sources...)
	}
	return append(out, p.floating...)
}

// confirmSources makes the online part of step 0 (see the file comment):
// the record of each source, in the order of [chains], each chain's
// configured sources first, then those configured for no network, then its
// defaults; and the sources of each pinned network, in the order of
// [chains] and of each chain's pins; or configuration_error.
func confirmSources(cfg Config) ([]SourceRecord, []SourceSet, *Rejection) {
	plans := make([]chainPlan, 0, len(chains))
	var operator []Source
	for _, chain := range chains {
		plan, rejection := planChain(cfg, chain)
		if rejection != nil {
			return nil, nil, rejection
		}
		plans = append(plans, plan)
		operator = append(operator, plan.reads()...)
	}

	answers := readIdentities(cfg.ReadIdentity, operator)
	var records []SourceRecord
	var defaults []Source
	sets := map[[2]string]SourceSet{}
	for _, plan := range plans {
		count := len(plan.reads())
		chainRecords, chainSets, rejection := assign(cfg, plan, answers[:count])
		if rejection != nil {
			return nil, nil, rejection
		}
		answers = answers[count:]
		records = append(records, chainRecords...)
		for _, set := range chainSets {
			sets[[2]string{set.Chain, set.Network}] = set
		}
		for _, network := range plan.networks {
			if _, ok := sets[[2]string{plan.chain, network}]; ok {
				continue
			}
			s, ok := DefaultSource(plan.chain, network)
			if !ok {
				return nil, nil, noSource(plan.chain, network)
			}
			defaults = append(defaults, s)
		}
	}

	for i, answer := range readIdentities(cfg.ReadIdentity, defaults) {
		s := defaults[i]
		record := SourceRecord{Source: s, From: FromDefault, Pins: cfg.pinsFor(s.Chain, []string{s.Network}), FetchedAt: answer.at}
		switch {
		case answer.err != nil:
			record.Status, record.Reason = SourceUnreachable, answer.err.Error()
		case !allows(s.Chain, s.Network, answer.value):
			return nil, nil, mismatch(s, sourceAt(s, 1, 1), []string{s.Network}, answer.value)
		default:
			record.Status, record.Identity = SourceConfirmed, answer.value
		}
		records = append(records, record)
		sets[[2]string{s.Chain, s.Network}] = SourceSet{
			Chain: s.Chain, Network: s.Network, Policy: SourcePolicy{Kind: SourcePolicyFirst}, Sources: []SourceRecord{record},
		}
	}
	slices.SortStableFunc(records, func(a, b SourceRecord) int {
		return slices.Index(chains, a.Source.Chain) - slices.Index(chains, b.Source.Chain)
	})
	var ordered []SourceSet
	for _, plan := range plans {
		for _, network := range plan.networks {
			ordered = append(ordered, sets[[2]string{plan.chain, network}])
		}
	}
	return records, ordered, nil
}

// planChain sorts chain's sources before any read, and refuses a pinned
// network with no source that the reads cannot change.
func planChain(cfg Config, chain string) (chainPlan, *Rejection) {
	plan := chainPlan{chain: chain, networks: cfg.networksOf(chain)}
	for _, network := range plan.networks {
		i := slices.IndexFunc(cfg.NetworkSources, func(n NetworkSources) bool { return n.Chain == chain && n.Network == network })
		if i < 0 {
			plan.open = append(plan.open, network)
			continue
		}
		n := cfg.NetworkSources[i]
		n.Sources = slices.Clone(n.Sources)
		for j := range n.Sources {
			n.Sources[j].Network = network
		}
		plan.configured = append(plan.configured, n)
	}
	if len(plan.open) > 0 {
		for _, s := range cfg.Sources {
			if s.Chain == chain {
				plan.floating = append(plan.floating, s)
			}
		}
	}
	if len(plan.floating) == 0 {
		for _, network := range plan.open {
			if _, ok := DefaultSource(chain, network); !ok {
				return chainPlan{}, noSource(chain, network)
			}
		}
	}
	return plan, nil
}

// assign holds each of a chain's operator sources to what its identity
// read's answer allows (answers holds the configured sources' answers, in
// order, then the floating sources'), and gives the sources of each
// network it serves: each configured network's, under its policy, each
// source held to that network alone; then each open network that floating
// sources serve: each floating source that answered, in order, serves the
// open networks its answer allows that no floating source before it
// serves, and then each that did not answer, in order, serves the open
// networks no floating source before it serves, so that an open network
// has one source, under the policy first, as SPEC.md section 9 gives a
// network with no configuration. A floating source that answered and
// whose answer allows only networks a configuration or an earlier
// floating source serves serves nothing and is recorded with no pins, and
// one whose answer no pinned network of the chain allows is
// configuration_error.
func assign(cfg Config, plan chainPlan, answers []identityAnswer) ([]SourceRecord, []SourceSet, *Rejection) {
	var records []SourceRecord
	var sets []SourceSet
	for _, n := range plan.configured {
		set := SourceSet{Chain: plan.chain, Network: n.Network, Policy: n.Policy}
		for place, s := range n.Sources {
			answer := answers[0]
			answers = answers[1:]
			record := SourceRecord{Source: s, From: FromOperator, Pins: cfg.pinsFor(plan.chain, []string{n.Network}), FetchedAt: answer.at}
			switch {
			case answer.err != nil:
				record.Status, record.Reason = SourceUnreachable, answer.err.Error()
			case !allows(plan.chain, n.Network, answer.value):
				return nil, nil, mismatch(s, sourceAt(s, place+1, len(n.Sources)), []string{n.Network}, answer.value)
			default:
				record.Status, record.Identity = SourceConfirmed, answer.value
			}
			records = append(records, record)
			set.Sources = append(set.Sources, record)
		}
		sets = append(sets, set)
	}

	floating := make([]SourceRecord, len(plan.floating))
	var serves [][]string
	for i, s := range plan.floating {
		floating[i] = SourceRecord{Source: s, From: FromOperator, FetchedAt: answers[i].at}
		var networks []string
		if err := answers[i].err; err != nil {
			floating[i].Status, floating[i].Reason = SourceUnreachable, err.Error()
		} else {
			value := answers[i].value
			allowed := false
			for _, network := range plan.open {
				if allows(plan.chain, network, value) {
					allowed = true
					if !slices.ContainsFunc(serves, func(earlier []string) bool { return slices.Contains(earlier, network) }) {
						networks = append(networks, network)
					}
				}
			}
			if !allowed && !slices.ContainsFunc(plan.networks, func(network string) bool { return allows(plan.chain, network, value) }) {
				return nil, nil, mismatch(s, sourceAt(s, 1, 1), plan.networks, value)
			}
			floating[i].Status, floating[i].Identity = SourceConfirmed, value
		}
		serves = append(serves, networks)
	}
	served := func(network string) bool {
		return slices.ContainsFunc(serves, func(networks []string) bool { return slices.Contains(networks, network) })
	}
	for i := range floating {
		if floating[i].Status != SourceUnreachable {
			continue
		}
		var networks []string
		for _, network := range plan.open {
			if !served(network) {
				networks = append(networks, network)
			}
		}
		serves[i] = networks
	}
	for i := range floating {
		floating[i].Pins = cfg.pinsFor(plan.chain, serves[i])
	}
	records = append(records, floating...)
	for _, network := range plan.open {
		set := SourceSet{Chain: plan.chain, Network: network, Policy: SourcePolicy{Kind: SourcePolicyFirst}}
		for i, record := range floating {
			if slices.Contains(serves[i], network) {
				set.Sources = append(set.Sources, record)
			}
		}
		if len(set.Sources) > 0 {
			sets = append(sets, set)
		}
	}
	return records, sets, nil
}

// chainName is a chain's name in a message.
func chainName(chain string) string {
	if chain == ChainStellar {
		return "Stellar"
	}
	return "Bitcoin"
}

// identityName names what a chain's identity read gives.
func identityName(chain string) string {
	if chain == ChainStellar {
		return "network passphrase"
	}
	return "hash of the block at height 0"
}

// noSource is configuration_error for a pinned network with no source.
func noSource(chain, network string) *Rejection {
	return ConfigurationError(fmt.Sprintf(
		"The run pins %s's %s network, and no source serves it, so the run gives no verdict: "+
			"this verifier has no built-in source for %s, so run again with %s for it.",
		chainName(chain), network, network, operatorOptions(chain)))
}

// mismatch is configuration_error for source s, which the message names as
// at ([sourceAt]), whose identity value no network it may serve allows;
// networks are the networks the run pins it to.
func mismatch(s Source, at string, networks []string, value string) *Rejection {
	chain := chainName(s.Chain)
	if value == productionIdentity[s.Chain] {
		return ConfigurationError(fmt.Sprintf(
			"The %s source at %s gives the %s of %s's production network, %s, and the run pins it to %s, "+
				"so the run gives no verdict.",
			chain, at, identityName(s.Chain), chain, productionNetwork[s.Chain], strings.Join(networks, " or ")))
	}
	return ConfigurationError(fmt.Sprintf(
		"The %s source at %s gives a %s other than that of %s's production network, %s, which the run pins it to, "+
			"so the run gives no verdict.",
		chain, at, identityName(s.Chain), chain, productionNetwork[s.Chain]))
}

// sourceAt names source s in a message: its URL as [printedURL] prints it
// and, for one of count sources of a network's configuration, count above
// one, its place in that network's order, from 1, since two sources of one
// provider and host print alike.
func sourceAt(s Source, place, count int) string {
	at := printedURL(s.URL)
	if count > 1 {
		at += fmt.Sprintf(", source %d of the %d sources of %s's %s network,", place, count, chainName(s.Chain), s.Network)
	}
	return at
}

// printedURL is a source's URL as a message or a report prints it:
// scheme://host, with the port when the URL names one. Its path, which may
// hold a provider's API key, is never printed; [Config.Validate] has
// refused user information, a query and a fragment before any source is
// read.
func printedURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "a URL that does not parse"
	}
	return u.Scheme + "://" + u.Host
}

// operatorOptions names the options that pin a source of chain.
func operatorOptions(chain string) string {
	if chain == ChainStellar {
		return "--horizon URL"
	}
	return "--esplora URL or --bitcoin-core URL"
}

// sourceValues are the run's sources as CONFIGURATION's row holds them, in
// the order of [Run.Sources], each URL as [printedURL] prints it, and each
// with its identity read's fetched_at and cached, always false, since the
// read is a value this run read (SPEC.md section 10.5). Two
// sources of one provider and host that serve one network print alike,
// since no path is printed, and only their places in that order tell them
// apart.
func sourceValues(records []SourceRecord) []any {
	values := make([]any, 0, len(records))
	for _, r := range records {
		pins := make([]any, 0, len(r.Pins))
		for _, p := range r.Pins {
			pins = append(pins, map[string]any{"network": p.Network, "pin": p.Pin})
		}
		value := map[string]any{
			"cached": false, "chain": r.Source.Chain, "fetched_at": fetchedAt(r.FetchedAt), "from": string(r.From), "pins": pins,
			"provider": string(r.Source.Provider), "status": string(r.Status), "url": printedURL(r.Source.URL),
		}
		if r.Status == SourceConfirmed {
			value["identity"] = r.Identity
		} else {
			value["reason"] = r.Reason
		}
		values = append(values, value)
	}
	return values
}

// sourcesSummary says, for CONFIGURATION's message, how many sources
// confirmed their network and how many gave no usable answer.
func sourcesSummary(records []SourceRecord) string {
	unreachable := 0
	for _, r := range records {
		if r.Status == SourceUnreachable {
			unreachable++
		}
	}
	return fmt.Sprintf("Of its %d Stellar and Bitcoin sources, %d confirmed the networks they serve and %d gave no usable answer, "+
		"so nothing else is read from those.", len(records), len(records)-unreachable, unreachable)
}
