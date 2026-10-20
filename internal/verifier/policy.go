// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds step 8 of SPEC.md section 10.2, POLICY, offline, in
// every run that reaches it. Each network member of P must equal the run's
// pin, else network_mismatch at that network:
//
//	| network of                                                                  | Pin                                |
//	|-----------------------------------------------------------------------------|------------------------------------|
//	| each commitment entry, at /commitments and /signing_key_event/commitments   | the commitment pin for its chain   |
//	| each Stellar or Bitcoin entropy object, at /subject/entropy, /witnesses/entropy_X and /observations/entropy_X/entropy | the entropy pin for its source |
//	| each metadata.genesis.bitcoin, at any block position                        | the Bitcoin commitment pin         |
//
// The positions are layout.go's. The pins are the run's, [Config.Networks]
// (SPEC.md section 9), which CONFIGURATION fixed: the row of
// spec/network-pins.json for E_v, which no flag, configuration key or
// environment variable changes, or, in a conformance test, a case's
// networks (SPEC.md section 13.2), set on the Config in place of that row.
// No pin comes from the document, and a network the proof lacks is never
// supplied: SHAPE has held every position above to carry one. The proof's
// network is compared as the string it is.
//
// Each comparison reports one row, in the document order of its part
// ([SortBy]). Its values hold the chain or source its position names, the
// pin and the pin's kind, and on a failure the proof's network under
// declared (SPEC.md section 10.5). A failing row's message never quotes
// the proof's network. A failure here does not end the run: DERIVE still
// runs, and the external gate stays closed. A run that has ended passes
// through.
//
// The pins are a [specdata.NetworkPins], a struct with a member for each
// chain, so [Config.pinNetwork], which the online run's sources also read,
// names the member a position's chain and kind select; and a message names
// a chain as Bitcoin or Stellar.

package verifier

import (
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// policyComparison is one network of P: its pointer, the kind of pin it
// is compared with, the chain or source its position names, and the
// network P carries.
type policyComparison struct {
	at      Pointer
	kind    string
	chain   string
	network any
}

// Policy is step 8, POLICY: one row for each network of P, against the
// run's pins (see the file's doc).
func Policy(run *Run) {
	if run.Ended {
		return
	}
	comparisons := policyComparisons(run.Proof)
	SortBy(comparisons, func(c policyComparison) Pointer { return c.at })
	for _, c := range comparisons {
		run.AddRow(policyRow(c, run.Config))
	}
}

// policyComparisons returns every network of P, by the layout: the
// commitment entries', the Stellar and Bitcoin entropy objects' and each
// genesis.bitcoin's.
func policyComparisons(proof map[string]any) []policyComparison {
	var out []policyComparison
	for _, entry := range commitmentEntries(proof) {
		out = append(out, policyComparison{with(entry.at, Name("network")), PinCommitment, entry.name, entry.node["network"]})
	}
	for _, object := range entropyObjects(proof) {
		if object.name == ChainBitcoin || object.name == ChainStellar {
			out = append(out, policyComparison{with(object.at, Name("network")), PinEntropy, object.name, object.node["network"]})
		}
	}
	for _, genesis := range genesisBitcoin(proof) {
		out = append(out, policyComparison{with(genesis.at, Name("network")), PinCommitment, ChainBitcoin, genesis.node["network"]})
	}
	return out
}

// policyRow compares one network with the run's pin of its kind and chain:
// a chain's commitment pin, or the entropy pin of the source of that name.
func policyRow(c policyComparison, config Config) Row {
	pin := config.pinNetwork(c.kind, c.chain)
	part := Part(c.at.Encode())
	values := map[string]any{"chain": c.chain, "pin": pin, "pin_kind": c.kind}
	if network, ok := c.network.(string); ok && network == pin {
		return PassRow("POLICY", part, fmt.Sprintf("The %s network is the pinned %s.", chainName(c.chain), pin), values)
	}
	values["declared"] = c.network
	noun := "commitment entries"
	if c.kind == PinEntropy {
		noun = "entropy objects"
	}
	return NewRow("POLICY", specdata.ReasonNetworkMismatch, part,
		fmt.Sprintf("This proof names a %s network other than the %s network this run pins for its %s.", chainName(c.chain), pin, noun),
		values)
}
