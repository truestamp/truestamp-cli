// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds, as compiled tables, the data files the reference
// verifier's configuration and readers compile at build time: the pattern
// classes of spec/encodings.json (read by its Truestamp.Verifier.Keyring and
// Truestamp.Verifier.Config, elixir/lib/truestamp/verifier/keyring.ex and
// config.ex in truestamp-verification), and spec/networks.json and
// spec/network-pins.json with the environments (Config). Those modules are
// ported with the steps that use them, and read these tables.

package specdata

import (
	"maps"
	"slices"
)

// Encoding is one pattern class of spec/encodings.json (SPEC.md section
// 2.4). Each member is the file's member of that name, and is the zero value
// when the class has none.
type Encoding struct {
	// Pattern is the class's regular expression as the file writes it,
	// with no ^ or $.
	Pattern string
	// DecodesToOctets is the length of what a base64url class decodes to.
	DecodesToOctets int
	// CapsInOctets bounds HEXBYTES, by the member it appears at.
	CapsInOctets map[string]int
	// Values lists the values of ENV.
	Values []string
	// ValuesByChain lists the values of NETWORK, by chain.
	ValuesByChain map[string][]string
	// Range bounds INT.
	Range *IntRange
	// Schema names the schema definition of MERKLE_PROOF.
	Schema string
	// RoundTrip says a TIME or NISTTIME value must equal its own
	// re-formatting.
	RoundTrip bool
}

// IntRange is the minimum and maximum of a class of integers.
type IntRange struct {
	Minimum, Maximum int64
}

// asciiPattern is ASCII128's and ASCII16's character class: printable ASCII
// but the quote and the backslash.
const asciiPattern = `[\x20\x21\x23-\x5b\x5d-\x7e]`

// encodings is spec/encodings.json.
var encodings = map[string]Encoding{
	"ASCII128": {Pattern: asciiPattern + `{1,128}`},
	"ASCII16":  {Pattern: asciiPattern + `{1,16}`},
	"B64U32":   {Pattern: `[A-Za-z0-9_-]{42}[AEIMQUYcgkosw048]`, DecodesToOctets: 32},
	"B64U64":   {Pattern: `[A-Za-z0-9_-]{85}[AQgw]`, DecodesToOctets: 64},
	"ENV":      {Values: []string{"dev", "staging", "test"}},
	"HEX64":    {Pattern: `[0-9a-f]{64}`},
	"HEXBYTES": {
		Pattern:      `(?:[0-9a-f]{2})+`,
		CapsInOctets: map[string]int{"raw_transaction": RawTxMaxOctets, "txoutproof": TxoutproofMaxOctets},
	},
	"INT":          {Range: &IntRange{Minimum: 0, Maximum: MaxInteger}},
	"KID":          {Pattern: `[0-9a-f]{8}`},
	"MERKLE_PROOF": {Schema: "schema.json#/$defs/MERKLE_PROOF"},
	"NETWORK": {ValuesByChain: map[string][]string{
		"bitcoin": {"mainnet", "testnet", "regtest"},
		"stellar": {"public", "testnet"},
	}},
	"NISTHEX": {Pattern: `[0-9A-F]{128}`},
	"NISTTIME": {
		Pattern:   `[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}Z`,
		RoundTrip: true,
	},
	"TIME": {
		Pattern:   `[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z`,
		RoundTrip: true,
	},
	"UUID7": {Pattern: `[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`},
}

// Encodings returns the pattern classes of spec/encodings.json, by name.
// Each call returns a new map, sharing nothing with the table.
func Encodings() map[string]Encoding {
	out := make(map[string]Encoding, len(encodings))
	for name, class := range encodings {
		out[name] = class.clone()
	}
	return out
}

func (e Encoding) clone() Encoding {
	e.CapsInOctets = maps.Clone(e.CapsInOctets)
	e.Values = slices.Clone(e.Values)
	e.ValuesByChain = cloneLists(e.ValuesByChain)
	if e.Range != nil {
		bounds := *e.Range
		e.Range = &bounds
	}
	return e
}

// networks is spec/networks.json: the network names of each chain, in the
// file's order.
var networks = map[string][]string{
	"bitcoin": {"mainnet", "testnet", "regtest"},
	"stellar": {"public", "testnet"},
}

// Networks returns the network names of each chain, by chain, as
// spec/networks.json lists them. Each call returns a new map.
func Networks() map[string][]string {
	return cloneLists(networks)
}

// ChainNetworks is a network for each chain: for a commitment pin, the
// chain's; for an entropy pin, the network of the source of that name.
type ChainNetworks struct {
	Bitcoin string
	Stellar string
}

// NetworkPins is one row of spec/network-pins.json (SPEC.md section 9): the
// default commitment pins and entropy pins of one E_v.
type NetworkPins struct {
	Commitment ChainNetworks
	Entropy    ChainNetworks
}

// networkPins is spec/network-pins.json.
var networkPins = map[string]NetworkPins{
	"dev": {
		Commitment: ChainNetworks{Bitcoin: "regtest", Stellar: "testnet"},
		Entropy:    ChainNetworks{Bitcoin: "mainnet", Stellar: "testnet"},
	},
	"prod": {
		Commitment: ChainNetworks{Bitcoin: "mainnet", Stellar: "public"},
		Entropy:    ChainNetworks{Bitcoin: "mainnet", Stellar: "public"},
	},
	"staging": {
		Commitment: ChainNetworks{Bitcoin: "regtest", Stellar: "testnet"},
		Entropy:    ChainNetworks{Bitcoin: "mainnet", Stellar: "testnet"},
	},
	"test": {
		Commitment: ChainNetworks{Bitcoin: "regtest", Stellar: "testnet"},
		Entropy:    ChainNetworks{Bitcoin: "regtest", Stellar: "testnet"},
	},
}

// Environments returns every E_v (SPEC.md section 6.4): prod, then the
// values of ENV in the order spec/encodings.json lists them. Each call
// returns a new slice.
func Environments() []string {
	return append([]string{"prod"}, encodings["ENV"].Values...)
}

// DefaultNetworkPins returns the default network pins of environment, an
// E_v, from spec/network-pins.json, and whether it is one.
func DefaultNetworkPins(environment string) (NetworkPins, bool) {
	pins, ok := networkPins[environment]
	return pins, ok
}

func cloneLists(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	for key, list := range in {
		out[key] = slices.Clone(list)
	}
	return out
}
