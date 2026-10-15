// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file is the part of step 0 (CONFIGURATION, SPEC.md section 10.2)
// that only an online run makes: which Stellar and Bitcoin source serves
// each network pin, and each source's identity, confirmed both ways
// before the file is read (SPEC.md sections 9 (What a verifier pins), 10.2,
// step 0, and 10.4 (Chain and source reads)). No reference module ports
// it, since the reference verifier is offline. It follows the Truestamp
// application's step 0, Truestamp.Proof.Verifier.Frame.confirm/1
// (lib/truestamp/proof/verifier/frame.ex in truestamp-v2, as committed with
// its online frame), with Truestamp.SourceCache.Sources and
// Truestamp.SourceCache.Identity (lib/truestamp/source_cache/sources.ex and
// identity.ex), adapted to an external verifier's sources:
//
//   - the application configures a source for each pin; the CLI has a
//     compiled-in default for each network but regtest ([DefaultSource]),
//     and the operator may pin a Horizon server, an Esplora server and a
//     Bitcoin Core node ([Config.Sources]), each serving the pins of its
//     chain whose network its identity read shows (SPEC.md section 9,
//     "Which pin a source serves"); a source a conformance case's replay
//     file stands for names its one network instead ([Source.Network]);
//   - an identity read with no usable answer records the source
//     unreachable, and the run reads nothing else from it; the rows it
//     serves are made by CHAIN and SOURCES, so this file records only the
//     outcome, in [Run.Sources];
//   - a source whose identity contradicts the pin it serves, a pinned
//     network with no source, and two sources for one pin are each
//     configuration_error, a bare rejection;
//   - a source's URL is printed, in a message and in CONFIGURATION's
//     values, as its scheme, host and port alone ([printedURL]), and no
//     unreachable reason names it, since a hosted node's provider may
//     carry an API key in the path and a report is made to be shared; the
//     reads use the whole URL.
//
// SPEC.md section 9 does not settle which pins a source with no usable
// identity answer serves: its rule reads the network from that answer.
// The rule of step 4 below is this CLI's own: such a source serves the
// pins it may serve that no source that answered serves, the first of two
// such sources in the options' order taking them and the other none, so
// no pin is served twice, no default is read in an operator source's
// place, and an outage stays an outage rather than a configuration_error
// (SPEC.md section 10.2, step 0).
//
// NIST's Randomness Beacon has no network and no identity read: SOURCES
// reads it at its one compiled-in address.
//
// The steps, for an online run:
//
//  1. each chain's pinned networks, from the run's commitment and entropy
//     pins in that order; an operator source that names a network not
//     pinned serves nothing and is not read;
//  2. before any read: two operator sources that name one network, or more
//     operator sources of a chain than it has networks for them to serve,
//     are two sources for one pin; a pinned network that no operator
//     source can serve and that has no default has no source;
//  3. every operator source's identity read, all at once;
//  4. each operator source that answered serves the pinned networks its
//     answer allows, of those it may serve (its own network, or every
//     network of its chain no source names): a production value the
//     chain's production network, any other value its other networks; one
//     that may serve none of them contradicts its pin; two that serve one
//     network are two sources for one pin; then each operator source with
//     no usable answer, in the order of [Config.Sources] (the options'
//     order: Horizon, Esplora, Bitcoin Core), serves every network it may
//     serve that neither a source that answered nor such a source before
//     it serves, so under dev or staging, of an Esplora server and a
//     Bitcoin Core node that both give no usable answer, the Esplora
//     server serves regtest and mainnet and the node serves nothing;
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

// SourceReadDeadline bounds each identity read, the whole read, as the
// Truestamp application bounds each of its verification reads.
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
	// Network is the one network the source is pinned to, or "" for a
	// source that serves the pins of its chain whose network its identity
	// read shows.
	Network string
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
	// SourceConfirmed: the source answered with a value its pins allow.
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
	// Pins are the network pins the source serves.
	Pins []ServedPin
}

// Serves reports whether the record serves pin of its chain.
func (r SourceRecord) Serves(pin, network string) bool {
	return slices.Contains(r.Pins, ServedPin{Pin: pin, Network: network})
}

// SourceFor returns the record of the source that serves the run's pin
// (commitment or entropy) of chain, and whether one does: every pin of an
// online run that passed CONFIGURATION has one, and an offline run has
// none.
func (r *Run) SourceFor(pin, chain string) (SourceRecord, bool) {
	network := r.Config.pinNetwork(pin, chain)
	for _, record := range r.Sources {
		if record.Source.Chain == chain && record.Serves(pin, network) {
			return record, true
		}
	}
	return SourceRecord{}, false
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

// identityAnswer is one identity read's outcome.
type identityAnswer struct {
	value string
	err   error
}

// readIdentities makes the identity read of each source through reader,
// all at once, and returns their answers in the sources' order. A nil
// reader reads nothing, so each source gives no usable answer.
func readIdentities(reader IdentityReader, sources []Source) []identityAnswer {
	answers := make([]identityAnswer, len(sources))
	var wg sync.WaitGroup
	for i, s := range sources {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if reader == nil {
				answers[i] = identityAnswer{err: fmt.Errorf("no identity reader")}
				return
			}
			value, err := reader(s)
			answers[i] = identityAnswer{value: value, err: err}
		}()
	}
	wg.Wait()
	return answers
}

// chainPlan is what step 0 knows of one chain's operator sources before
// any read: the sources pinned to one network, by network, and the
// sources that serve whatever their identity read shows.
type chainPlan struct {
	chain    string
	networks []string
	fixed    []Source
	floating []Source
	// open are the pinned networks no fixed source names, which the
	// floating sources may serve.
	open []string
}

// confirmSources makes the online part of step 0 (see the file comment):
// the record of each source, in the order of [chains], each chain's
// operator sources first; or configuration_error.
func confirmSources(cfg Config) ([]SourceRecord, *Rejection) {
	plans := make([]chainPlan, 0, len(chains))
	var operator []Source
	for _, chain := range chains {
		plan, rejection := planChain(cfg, chain)
		if rejection != nil {
			return nil, rejection
		}
		plans = append(plans, plan)
		operator = append(operator, plan.fixed...)
		operator = append(operator, plan.floating...)
	}

	answers := readIdentities(cfg.ReadIdentity, operator)
	var records []SourceRecord
	var defaults []Source
	for _, plan := range plans {
		chainRecords, rejection := assign(cfg, plan, answers[:len(plan.fixed)+len(plan.floating)])
		if rejection != nil {
			return nil, rejection
		}
		answers = answers[len(plan.fixed)+len(plan.floating):]
		records = append(records, chainRecords...)
		for _, network := range plan.networks {
			if served(chainRecords, network) {
				continue
			}
			s, ok := DefaultSource(plan.chain, network)
			if !ok {
				return nil, noSource(plan.chain, network)
			}
			defaults = append(defaults, s)
		}
	}

	for i, answer := range readIdentities(cfg.ReadIdentity, defaults) {
		s := defaults[i]
		record := SourceRecord{Source: s, From: FromDefault, Pins: cfg.pinsFor(s.Chain, []string{s.Network})}
		switch {
		case answer.err != nil:
			record.Status, record.Reason = SourceUnreachable, answer.err.Error()
		case !allows(s.Chain, s.Network, answer.value):
			return nil, mismatch(s, []string{s.Network}, answer.value)
		default:
			record.Status, record.Identity = SourceConfirmed, answer.value
		}
		records = append(records, record)
	}
	slices.SortStableFunc(records, func(a, b SourceRecord) int {
		return slices.Index(chains, a.Source.Chain) - slices.Index(chains, b.Source.Chain)
	})
	return records, nil
}

// planChain sorts chain's operator sources before any read, and refuses
// two sources for one pin and a pinned network with no source that the
// reads cannot change.
func planChain(cfg Config, chain string) (chainPlan, *Rejection) {
	plan := chainPlan{chain: chain, networks: cfg.networksOf(chain)}
	for _, s := range cfg.Sources {
		switch {
		case s.Chain != chain:
		case s.Network == "":
			plan.floating = append(plan.floating, s)
		case !slices.Contains(plan.networks, s.Network):
			// A source pinned to a network the run does not pin serves
			// nothing, and is not read.
		case slices.ContainsFunc(plan.fixed, func(f Source) bool { return f.Network == s.Network }):
			return chainPlan{}, twoSources(chain, []string{s.Network})
		default:
			plan.fixed = append(plan.fixed, s)
		}
	}
	for _, network := range plan.networks {
		if !slices.ContainsFunc(plan.fixed, func(f Source) bool { return f.Network == network }) {
			plan.open = append(plan.open, network)
		}
	}
	if len(plan.floating) > len(plan.open) {
		return chainPlan{}, twoSources(chain, plan.open)
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

// assign gives each of a chain's operator sources the pins it serves from
// its identity read's answer (answers holds the fixed sources' answers,
// then the floating sources'): first each source that answered, then each
// that did not, in order, the pins it may serve that no record before it
// serves.
func assign(cfg Config, plan chainPlan, answers []identityAnswer) ([]SourceRecord, *Rejection) {
	var records []SourceRecord
	operator := append(slices.Clone(plan.fixed), plan.floating...)
	var unreachable []int
	for i, s := range operator {
		may := plan.open
		if s.Network != "" {
			may = []string{s.Network}
		}
		record := SourceRecord{Source: s, From: FromOperator}
		if err := answers[i].err; err != nil {
			record.Status, record.Reason = SourceUnreachable, err.Error()
			records = append(records, record)
			unreachable = append(unreachable, len(records)-1)
			continue
		}
		value := answers[i].value
		var serves []string
		for _, network := range may {
			if allows(plan.chain, network, value) {
				serves = append(serves, network)
			}
		}
		if len(serves) == 0 {
			return nil, mismatch(s, may, value)
		}
		for _, network := range serves {
			if served(records, network) {
				return nil, twoSources(plan.chain, []string{network})
			}
		}
		record.Status, record.Identity, record.Pins = SourceConfirmed, value, cfg.pinsFor(plan.chain, serves)
		records = append(records, record)
	}
	for _, i := range unreachable {
		may := plan.open
		if network := records[i].Source.Network; network != "" {
			may = []string{network}
		}
		var serves []string
		for _, network := range may {
			if !served(records, network) {
				serves = append(serves, network)
			}
		}
		records[i].Pins = cfg.pinsFor(plan.chain, serves)
	}
	return records, nil
}

// served reports whether a record serves a pin of network.
func served(records []SourceRecord, network string) bool {
	return slices.ContainsFunc(records, func(r SourceRecord) bool {
		return slices.ContainsFunc(r.Pins, func(p ServedPin) bool { return p.Network == network })
	})
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

// twoSources is configuration_error for two sources of one pin.
func twoSources(chain string, networks []string) *Rejection {
	return ConfigurationError(fmt.Sprintf(
		"Two %s sources serve the pin of %s's %s network, so the run gives no verdict: give one source for each network.",
		chainName(chain), chainName(chain), strings.Join(networks, " or ")))
}

// mismatch is configuration_error for a source whose identity value no
// network it may serve allows.
func mismatch(s Source, networks []string, value string) *Rejection {
	chain := chainName(s.Chain)
	if value == productionIdentity[s.Chain] {
		return ConfigurationError(fmt.Sprintf(
			"The %s source at %s gives the %s of %s's production network, %s, and the run pins it to %s, "+
				"so the run gives no verdict.",
			chain, printedURL(s.URL), identityName(s.Chain), chain, productionNetwork[s.Chain], strings.Join(networks, " or ")))
	}
	return ConfigurationError(fmt.Sprintf(
		"The %s source at %s gives a %s other than that of %s's production network, %s, which the run pins it to, "+
			"so the run gives no verdict.",
		chain, printedURL(s.URL), identityName(s.Chain), chain, productionNetwork[s.Chain]))
}

// printedURL is a source's URL as a message or a report prints it:
// scheme://host, with the port when the URL names one. Its path, which may
// hold a provider's API key, is never printed; [Config.Validate] has refused user
// information, a query and a fragment before any source is read.
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

// sourceValues are the run's sources as CONFIGURATION's row holds them,
// each URL as [printedURL] prints it: two sources stay told apart by their
// chain, provider and pins.
func sourceValues(records []SourceRecord) []any {
	values := make([]any, 0, len(records))
	for _, r := range records {
		pins := make([]any, 0, len(r.Pins))
		for _, p := range r.Pins {
			pins = append(pins, map[string]any{"network": p.Network, "pin": p.Pin})
		}
		value := map[string]any{
			"chain": r.Source.Chain, "from": string(r.From), "pins": pins,
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
