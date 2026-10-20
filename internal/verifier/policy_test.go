// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// Step 8, POLICY (SPEC.md sections 9 and 10.2), on every network member of
// the 11 positive proofs, beside the vectors' own cases in
// internal/vectorcases' runner: one row for each network member, in
// document order; each network changed fails at exactly its own row, and
// each pin changed fails exactly the rows it governs; a failure does not
// end the run; and the pins are the run's, which nothing but the
// conformance seam, Config.Networks, sets apart from E_v's defaults.

// policyNetwork is a network member of P, by the test's own walk: its
// pointer, the kind of pin that governs it and its chain or source.
type policyNetwork struct {
	at    Pointer
	kind  string
	chain string
}

// networksOf returns every network member of proof, in document order,
// read from its pointer without the layout: under genesis or commitments a
// commitment pin, Bitcoin for genesis and the member's parent's name for a
// commitment entry, and otherwise an entropy pin for the object's source.
func networksOf(proof map[string]any) []policyNetwork {
	var out []policyNetwork
	for _, n := range nodes(proof) {
		if len(n.at) == 0 || n.at[len(n.at)-1].isIndex || n.at[len(n.at)-1].name != "network" {
			continue
		}
		if _, ok := n.value.(string); !ok {
			continue
		}
		names := map[string]bool{}
		for _, s := range n.at {
			names[s.name] = true
		}
		switch parent := n.at[:len(n.at)-1]; {
		case names["genesis"]:
			out = append(out, policyNetwork{n.at, PinCommitment, ChainBitcoin})
		case names["commitments"]:
			out = append(out, policyNetwork{n.at, PinCommitment, parent[len(parent)-1].name})
		default:
			source, _ := valueAt(proof, with(parent, Name("source"))).(string)
			out = append(out, policyNetwork{n.at, PinEntropy, source})
		}
	}
	return out
}

// pinsOfProof returns the pins under which every network of proof passes:
// test's defaults, with each pin a network of proof names set to it.
func pinsOfProof(t *testing.T, proof map[string]any) specdata.NetworkPins {
	t.Helper()
	pins, _ := specdata.DefaultNetworkPins("test")
	for _, n := range networksOf(proof) {
		setPin(&pins, n.kind, n.chain, valueAt(proof, n.at).(string))
	}
	return pins
}

// pinFor returns the pin of kind for chain, read here without pinNetwork.
func pinFor(pins specdata.NetworkPins, kind, chain string) string {
	return map[string]string{
		PinCommitment + " " + ChainBitcoin: pins.Commitment.Bitcoin, PinCommitment + " " + ChainStellar: pins.Commitment.Stellar,
		PinEntropy + " " + ChainBitcoin: pins.Entropy.Bitcoin, PinEntropy + " " + ChainStellar: pins.Entropy.Stellar,
	}[kind+" "+chain]
}

// setPin sets the pin of kind for chain.
func setPin(pins *specdata.NetworkPins, kind, chain, network string) {
	networks := &pins.Commitment
	if kind == PinEntropy {
		networks = &pins.Entropy
	}
	if chain == ChainStellar {
		networks.Stellar = network
	} else {
		networks.Bitcoin = network
	}
}

// otherNetwork returns a network of chain other than network.
func otherNetwork(chain, network string) string {
	for _, other := range specdata.Networks()[chain] {
		if other != network {
			return other
		}
	}
	panic("no other network of " + chain)
}

// policyRows runs POLICY over proof under pins, after a SHAPE that passed,
// and returns its rows.
func policyRows(proof map[string]any, pins specdata.NetworkPins) []Row {
	run := &Run{Config: Config{Environment: "test", Networks: pins}, Proof: proof,
		Rows: []Row{PassRow("SHAPE", nil, "held", nil)}}
	Policy(run)
	return run.Rows[1:]
}

// The positive proofs are consistent, one network for each pin, and POLICY
// gives one pass row for each network member, in document order, under the
// pins they name; the rows cover the commitments at both positions, every
// entropy object of a Stellar or Bitcoin source and genesis.bitcoin at
// every block position. Mutants killed: Policy leaving its rows in the
// layout's order (a beacon's /observations rows come after
// /signing_key_event's); genesisBitcoin left out of policyComparisons (the
// genesis rows are missing); entropyObjects skipping /observations (a
// beacon's entropy rows are missing); blockNodes leaving out
// /signing_key_event/block (its genesis row is missing).
func TestPolicy_OneRowForEachNetworkInDocumentOrder(t *testing.T) {
	proofs := positiveProofs(t)
	parts := map[string]bool{}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		pins := pinsOfProof(t, proof)
		var want []string
		for _, n := range networksOf(proof) {
			if got := valueAt(proof, n.at); got != pinFor(pins, n.kind, n.chain) {
				t.Errorf("%s %s: %v, two networks for one pin", name, n.at.Encode(), got)
			}
			want = append(want, "[null "+n.at.Encode()+" pass POLICY]")
		}
		if got := rowSummary(policyRows(proof, pins)); got != strings.Join(want, " ") || len(want) == 0 {
			t.Errorf("%s: %s, want %s", name, got, strings.Join(want, " "))
		}
		for _, row := range policyRows(proof, pins) {
			parts[*row.Part] = true
		}
	}
	for _, part := range []string{
		"/commitments/bitcoin/network", "/commitments/stellar/network",
		"/signing_key_event/commitments/bitcoin/network", "/signing_key_event/commitments/stellar/network",
		"/subject/entropy/network", "/witnesses/entropy_bitcoin/network", "/witnesses/entropy_stellar/network",
		"/observations/entropy_bitcoin/entropy/network", "/observations/entropy_stellar/entropy/network",
		"/block/record/metadata/genesis/bitcoin/network", "/witnesses/block/record/metadata/genesis/bitcoin/network",
		"/signing_key_event/block/record/metadata/genesis/bitcoin/network",
	} {
		if !parts[part] {
			t.Errorf("no row at %s", part)
		}
	}
}

// Every network member position against its pin, under the default pins
// of each E_v (prod, dev, staging and test): a value equal to the pin
// passes; a value different from it fails network_mismatch at exactly its
// own row, every other row as it was; and the member absent fails SHAPE's
// schema at that member, so POLICY makes no comparison of it and supplies
// no network. Each row's values give its chain, its pin and the pin's
// kind, and a failure the proof's network as declared, which its message
// does not quote. Mutants killed: pinNetwork giving the commitment pin for
// every kind (under dev and staging a Bitcoin entropy network of mainnet
// fails against regtest); pinNetwork giving the Bitcoin pin for every chain
// (each Stellar row fails); a genesis.bitcoin network compared with the
// entropy pin (under dev and staging the genesis rows fail at regtest);
// policyRow comparing case-folded (testnet against TESTNET passes, the
// schema having been skipped here).
func TestPolicy_EachPositionEqualDifferentOrAbsentUnderEachEnvironment(t *testing.T) {
	proofs := positiveProofs(t)
	for _, environment := range specdata.Environments() {
		pins, _ := specdata.DefaultNetworkPins(environment)
		for _, name := range sortedProofs(proofs) {
			proof := proofs[name]
			networks := networksOf(proof)
			for _, n := range networks {
				pin := pinFor(pins, n.kind, n.chain)
				aligned := clone(proof).(map[string]any)
				for _, m := range networks {
					aligned = put(aligned, m.at, pinFor(pins, m.kind, m.chain))
				}
				for kind, value := range map[string]string{"equal": pin, "different": otherNetwork(n.chain, pin), "upper": strings.ToUpper(pin)} {
					rows := policyRows(put(aligned, n.at, value), pins)
					if len(rows) != len(networks) {
						t.Fatalf("%s %s: %d rows for %d networks", environment, name, len(rows), len(networks))
					}
					for i, row := range rows {
						m := networks[i]
						wantFail := kind != "equal" && Compare(m.at, n.at) == 0
						values := map[string]any{"chain": m.chain, "pin": pinFor(pins, m.kind, m.chain), "pin_kind": m.kind}
						if wantFail {
							values["declared"] = value
						}
						if *row.Part != m.at.Encode() || (row.Code == specdata.ReasonNetworkMismatch) != wantFail ||
							(row.Status == StatusFail) != wantFail || string(encode(t, row.Values)) != string(encode(t, values)) ||
							wantFail && strings.Contains(row.Message, value) {
							t.Errorf("%s %s %s %s at %s: %s %s %q", environment, name, kind, n.at.Encode(), m.at.Encode(),
								rowSummary([]Row{row}), encode(t, row.Values), row.Message)
						}
					}
				}
			}
		}
	}
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		for _, n := range networksOf(proof) {
			run := shapeRun(t, remove(proof, n.at))
			Policy(run)
			if got := rowSummary(run.Rows[7:9]); got != failsAt(specdata.ReasonShapeInvalid, n.at)+" [prior_failure null skip POLICY]" {
				t.Errorf("%s without %s: %s", name, n.at.Encode(), got)
			}
		}
	}
}

// A pin changed fails exactly the rows it governs: a chain's commitment
// pin its commitment entries and, for Bitcoin, each genesis.bitcoin; a
// source's entropy pin its entropy objects; never another. Mutant killed:
// pinNetwork reading the entropy pins for a commitment of Stellar (the
// Stellar commitment rows follow the entropy pin).
func TestPolicy_APinChangedFailsExactlyTheRowsItGoverns(t *testing.T) {
	proofs := positiveProofs(t)
	for _, name := range sortedProofs(proofs) {
		proof := proofs[name]
		own := pinsOfProof(t, proof)
		for _, kind := range []string{PinCommitment, PinEntropy} {
			for _, chain := range chains {
				pins := own
				setPin(&pins, kind, chain, otherNetwork(chain, pinFor(own, kind, chain)))
				for i, row := range policyRows(proof, pins) {
					n := networksOf(proof)[i]
					governed := n.kind == kind && n.chain == chain
					if (row.Status == StatusFail) != governed {
						t.Errorf("%s, the %s pin of %s changed: %s", name, kind, chain, rowSummary([]Row{row}))
					}
				}
			}
		}
	}
}

// Under staging's default pins Bitcoin commits to regtest and reads
// entropy from mainnet, so a positive item, whose Bitcoin entropy is
// regtest's, fails at its Bitcoin witness alone, and passes once that
// network is mainnet. Mutant killed: pinNetwork giving the commitment pin for
// every kind (the witness passes).
func TestPolicy_UnderStagingBitcoinCommitsToRegtestAndReadsEntropyFromMainnet(t *testing.T) {
	item := positiveProofs(t)["item"]
	pins, _ := specdata.DefaultNetworkPins("staging")
	if pins.Commitment.Bitcoin != "regtest" || pins.Entropy.Bitcoin != "mainnet" {
		t.Fatalf("staging's pins: %+v", pins)
	}
	witness := Pointer{Name("witnesses"), Name("entropy_bitcoin"), Name("network")}
	var failing []string
	for _, row := range policyRows(item, pins) {
		if row.Status == StatusFail {
			failing = append(failing, *row.Part)
		}
	}
	if !slices.Equal(failing, []string{witness.Encode()}) {
		t.Errorf("failing rows %q, want %s alone", failing, witness.Encode())
	}
	for _, row := range policyRows(put(item, witness, "mainnet"), pins) {
		if row.Status != StatusPass {
			t.Errorf("with a mainnet witness: %s", rowSummary([]Row{row}))
		}
	}
}

// A failure does not end the run, which still fails; the run's pins, not
// the default pins of its E_v, decide; and an ended run passes through.
// Mutants killed: Policy ending the run at a failure (skip rows follow);
// Policy taking the default pins of the run's environment (the run's own
// pins pass the commitment rows that the defaults fail); Policy running
// over an ended run (rows are added).
func TestPolicy_AFailureDoesNotEndTheRun(t *testing.T) {
	item := positiveProofs(t)["item"]
	pins := pinsOfProof(t, item)
	pins.Commitment.Stellar = "public"
	run := &Run{Config: Config{Environment: "test", Networks: pins}, Proof: item, Rows: []Row{PassRow("SHAPE", nil, "held", nil)}}
	Policy(run)
	if run.Ended || !run.Failed() || len(run.Rows) != 1+len(networksOf(item)) {
		t.Fatalf("ended %v, failed %v, rows %s", run.Ended, run.Failed(), rowSummary(run.Rows))
	}
	for _, row := range run.Rows[1:] {
		if row.Status == StatusFail && (row.Values["pin"] != "public" || row.Values["declared"] != "testnet") {
			t.Errorf("a failing row's values: %v", row.Values)
		}
	}
	defaults, _ := specdata.DefaultNetworkPins("prod")
	under := &Run{Config: Config{Environment: "prod", Networks: pinsOfProof(t, item)}, Proof: item}
	Policy(under)
	if under.Failed() || defaults == pinsOfProof(t, item) {
		t.Errorf("under prod with the proof's own pins: %s", rowSummary(under.Rows))
	}
	ended := &Run{Config: Config{Networks: pins}, Proof: item, Ended: true}
	Policy(ended)
	if len(ended.Rows) != 0 {
		t.Errorf("Policy over an ended run added %s", rowSummary(ended.Rows))
	}
}

// The conformance pins seam is Config.Networks, and no flag, configuration
// key or environment variable reaches it: under every E_v, with every
// option a run takes given (--offline, the keyring options, the source
// options) and with every variable and configuration key that might name a
// pin set (network, networks, pins, the commitment and entropy pins by
// chain), the run's pins are the row of spec/network-pins.json for its E_v.
// And the non-test files of the module refer to the member exactly where
// networksReviewed lists, so a new reference, a read, a write or its
// address taken, fails until someone has reviewed it and listed it. The
// scan sees references to the member only: a whole Config replaced by
// another (NewConfig for another E_v assigned over cfg) names no Networks,
// so the run-level half of this test is what would catch it. Mutants
// killed: Options.Config setting a pin from an environment
// variable (TRUESTAMP_VERIFY_NETWORK moves the Bitcoin commitment pin);
// NewConfig taking the prod row for every E_v (the dev pins are prod's);
// Options.Config writing a pin through &cfg.Networks, under an --esplora
// URL that names testnet (a reference in Config), or passing &cfg.Networks
// to a function that writes one (the same).
func TestPins_NoFlagKeyOrVariableReachesConfigNetworks(t *testing.T) {
	clearTruestampEnv(t)
	for _, variable := range []string{"TRUESTAMP_NETWORK", "TRUESTAMP_NETWORKS", "TRUESTAMP_VERIFY_NETWORK", "TRUESTAMP_VERIFY_NETWORKS",
		"TRUESTAMP_VERIFY_PINS", "TRUESTAMP_VERIFY_NETWORKS_COMMITMENT_BITCOIN", "TRUESTAMP_VERIFY_NETWORKS_ENTROPY_STELLAR",
		"TRUESTAMP_VERIFY_COMMITMENT_BITCOIN", "TRUESTAMP_VERIFY_ENTROPY_BITCOIN", "TRUESTAMP_BITCOIN_NETWORK", "TRUESTAMP_STELLAR_NETWORK"} {
		t.Setenv(variable, "testnet")
	}
	toml := `network = "testnet"
networks = "testnet"

[verify]
network = "testnet"
pins = "testnet"
bitcoin_network = "testnet"
stellar_network = "testnet"

[verify.networks.commitment]
bitcoin = "testnet"
stellar = "public"

[verify.networks.entropy]
bitcoin = "testnet"
stellar = "public"
`
	for _, environment := range specdata.Environments() {
		args := []string{"--offline", "--environment", environment, "--no-keyring",
			"--horizon", "https://horizon.example", "--esplora", "https://esplora.example", "--bitcoin-core", "http://127.0.0.1:18443"}
		_, options := loadOptions(t, toml, args...)
		cfg, err := options.Config(nil, nil)
		want, _ := specdata.DefaultNetworkPins(environment)
		if err != nil || cfg.Networks != want {
			t.Errorf("%s: pins %+v, %v; want %+v", environment, cfg.Networks, err, want)
		}
	}
	if got := networksReferences(t, typedModuleOf(t)); !slices.Equal(got, networksReviewed) {
		t.Errorf("Config.Networks is referred to at %q, want %q", got, networksReviewed)
	}
}

// networksReviewed are the non-test references to Config's Networks
// member, one for each, by file and function, sorted: NewConfig sets it,
// from E_v's row; Validate reads each of its four pins; and pinNetwork
// reads the commitment or the entropy pins, for POLICY and for the
// sources of an online run.
var networksReviewed = []string{
	"internal/verifier/config.go NewConfig",
	"internal/verifier/config.go Validate", "internal/verifier/config.go Validate",
	"internal/verifier/config.go Validate", "internal/verifier/config.go Validate",
	"internal/verifier/sources.go pinNetwork", "internal/verifier/sources.go pinNetwork",
}

// networksReferences returns each reference of m's files to Config's
// Networks member, as its file and function, one entry for each, sorted:
// a selector or a composite literal's key that the type check resolves to
// the member, and a Config literal without keys, which sets every member.
// specdata.Networks, the function, is told apart by the object it names;
// a selector or a key named Networks that resolves to neither, or that the
// check could not resolve, is listed too, so the scan never passes over
// what it cannot tell.
func networksReferences(t *testing.T, m *typedModule) []string {
	t.Helper()
	verifierPackage, specdataPackage := m.packages[modulePath+"/internal/verifier"], m.packages[modulePath+"/internal/specdata"]
	if verifierPackage == nil || specdataPackage == nil {
		t.Fatal("the type check gave no internal/verifier or internal/specdata")
	}
	config := verifierPackage.Scope().Lookup("Config")
	var member types.Object
	if structure, ok := config.Type().Underlying().(*types.Struct); ok {
		for field := range structure.Fields() {
			if field.Name() == "Networks" {
				member = field
			}
		}
	}
	networksFunc := specdataPackage.Scope().Lookup("Networks")
	if member == nil || networksFunc == nil {
		t.Fatal("no Config.Networks member, or no specdata.Networks function")
	}
	var out []string
	for _, f := range m.files {
		ast.Inspect(f.file, func(n ast.Node) bool {
			var name *ast.Ident
			switch x := n.(type) {
			case *ast.SelectorExpr:
				name = x.Sel
			case *ast.KeyValueExpr:
				name, _ = x.Key.(*ast.Ident)
			case *ast.CompositeLit:
				if tv, ok := m.info.Types[x]; ok && types.Identical(tv.Type, config.Type()) && len(x.Elts) > 0 {
					if _, keyed := x.Elts[0].(*ast.KeyValueExpr); !keyed {
						out = append(out, f.rel+" "+enclosingFunction(f.file, x.Pos())+" (a Config literal without keys)")
					}
				}
			}
			if name == nil || name.Name != "Networks" {
				return true
			}
			switch m.info.Uses[name] {
			case member:
				out = append(out, f.rel+" "+enclosingFunction(f.file, name.Pos()))
			case networksFunc:
			default:
				out = append(out, f.rel+" "+enclosingFunction(f.file, name.Pos())+" (not Config's member)")
			}
			return true
		})
	}
	slices.Sort(out)
	return out
}

// pinsPlanted are files of a module whose references to Config's Networks
// member networksReferences must list: the reviewed shapes, beside a
// write through the member's address, the address passed to a function, a
// Config literal without keys, another type's member of that name and a
// reference from another package; and specdata.Networks, which it must
// not list.
var pinsPlanted = map[string]string{
	"internal/specdata/pins.go": `package specdata
type NetworkPins struct{ Commitment, Entropy struct{ Bitcoin, Stellar string } }
func Networks() map[string][]string { return nil }`,
	"internal/verifier/config.go": `package verifier
` + specdataImport + `
type Config struct {
	Environment string
	Networks    specdata.NetworkPins
}
func NewConfig() Config { return Config{Environment: "test", Networks: specdata.NetworkPins{}} }
func (c Config) Validate() bool { return len(specdata.Networks()) > 0 && c.Networks.Entropy.Bitcoin != "" }`,
	"internal/verifier/options.go": `package verifier
` + specdataImport + `
func through(c *Config) {
	if pins := &c.Networks; c.Environment == "dev" {
		pins.Commitment.Bitcoin = "testnet"
	}
}
func passed(c *Config) { set(&c.Networks) }
func set(pins *specdata.NetworkPins) { pins.Entropy.Stellar = "public" }
func unkeyed() Config { return Config{"test", specdata.NetworkPins{}} }
func another(x struct{ Networks int }) int { return x.Networks }`,
	"cmd/c.go": `package cmd
import "` + modulePath + `/internal/verifier"
func f(c verifier.Config) string { return c.Networks.Commitment.Stellar }`,
}

// networksReferences lists each planted reference to the member, by file
// and function, and no call of specdata.Networks. Mutants killed: the scan
// reading composite literals' keys alone, as a scan for setters of the
// member might (through's and passed's addresses pass); specdata's
// Networks not told apart (Validate's call is listed); a Config literal
// without keys not seen (unkeyed passes); a Networks of another type left
// out (another passes); the scan held to internal/verifier (cmd's read
// passes).
func TestPins_TheReferenceScanListsEachPlantedShape(t *testing.T) {
	want := []string{
		"cmd/c.go f",
		"internal/verifier/config.go NewConfig",
		"internal/verifier/config.go Validate",
		"internal/verifier/options.go another (not Config's member)",
		"internal/verifier/options.go passed",
		"internal/verifier/options.go through",
		"internal/verifier/options.go unkeyed (a Config literal without keys)",
	}
	if got := networksReferences(t, typedSources(t, pinsPlanted)); !slices.Equal(got, want) {
		t.Errorf("the planted references: %q, want %q", got, want)
	}
}
