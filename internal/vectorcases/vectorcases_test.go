// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package vectorcases_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// Load reads a case's sources, {<chain>: {<network>: {policy, required,
// sources}}} (SPEC.md section 13.2): 18 cases name them, each online, each
// naming only networks the case pins, as a commitment or an entropy pin,
// and sources-at_least_2-one_unreachable's Stellar testnet has the policy
// at_least, required 2, over the sources a, b and c, in that order, and
// names no Bitcoin sources. Mutants killed: Config's Sources decoded from
// another name, or NetworkSources's Required left undecoded (cases.json's
// sources or required is an unknown member, and Load refuses the file);
// Load dropping a case's sources once it has checked them (no case names
// sources).
func TestLoad_ReadsSources(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	named := 0
	for _, c := range cases {
		if c.Config.Sources == nil {
			continue
		}
		named++
		if c.Config.Online == nil {
			t.Errorf("%s names sources offline", c.ID)
		}
		for chain, networks := range c.Config.Sources {
			for network := range networks {
				if c.Config.Networks.Commitment[chain] != network && c.Config.Networks.Entropy[chain] != network {
					t.Errorf("%s names sources of %s %s, which it does not pin", c.ID, chain, network)
				}
			}
		}
		if c.ID != "sources-at_least_2-one_unreachable" {
			continue
		}
		stellar, ok := c.Config.Sources["stellar"]["testnet"]
		if _, bitcoin := c.Config.Sources["bitcoin"]; !ok || bitcoin || len(c.Config.Sources) != 1 || stellar.Policy != vectorcases.PolicyAtLeast ||
			stellar.Required == nil || *stellar.Required != 2 || len(stellar.Sources) != 3 ||
			stellar.Sources[0] != "a" || stellar.Sources[1] != "b" || stellar.Sources[2] != "c" {
			t.Errorf("%s: sources %+v", c.ID, c.Config.Sources)
		}
	}
	if named != 18 {
		t.Fatalf("%d cases name sources, want 18", named)
	}
}

// checkSources refuses a case's sources SPEC.md section 13.2 does not
// allow, and allows each form it does. Mutants killed, one for each
// refusal: sources taken offline; a network not of its chain, or of a
// chain other than stellar or bitcoin; no names; a name that is not an
// ASCII16 (empty, or of 17 characters); a name twice; an unknown policy;
// required under first or all; at_least without required, or with a
// required of 0 or above the number of names; a break after a chain's first
// network (a bad network after a good one, Bitcoin testnet after mainnet,
// is allowed); a return after the first chain (a bad chain after a good
// one, Stellar after Bitcoin, is allowed).
func TestCheckSources_Refusals(t *testing.T) {
	const online = `"online": "corpus/replay/sources-all-agree.json"`
	const networks = `"networks": {"commitment": {"bitcoin": "regtest", "stellar": "testnet"}, "entropy": {"bitcoin": "mainnet", "stellar": "testnet"}}`
	config := func(t *testing.T, text string) vectorcases.Config {
		t.Helper()
		var cfg vectorcases.Config
		if err := json.Unmarshal([]byte(text), &cfg); err != nil {
			t.Fatal(err)
		}
		return cfg
	}
	sources := func(body string) string {
		return `{"environment": "test", ` + networks + `, ` + online + `, "sources": ` + body + `}`
	}
	for name, body := range map[string]string{
		"first":                   `{"stellar": {"testnet": {"policy": "first", "sources": ["a", "b"]}}}`,
		"all, Bitcoin regtest":    `{"bitcoin": {"regtest": {"policy": "all", "sources": ["a", "b"]}}}`,
		"Bitcoin mainnet":         `{"bitcoin": {"mainnet": {"policy": "first", "sources": ["a"]}}}`,
		"at_least 1 of one":       `{"stellar": {"testnet": {"policy": "at_least", "required": 1, "sources": ["a"]}}}`,
		"at_least 3 of three":     `{"stellar": {"testnet": {"policy": "at_least", "required": 3, "sources": ["a", "b", "c"]}}}`,
		"a name of 16 characters": `{"stellar": {"testnet": {"policy": "first", "sources": ["abcdefghijklmnop"]}}}`,
		"both chains":             `{"bitcoin": {"regtest": {"policy": "first", "sources": ["a"]}}, "stellar": {"testnet": {"policy": "all", "sources": ["x", "y"]}}}`,
	} {
		if err := vectorcases.CheckSources(config(t, sources(body))); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	refused := map[string]string{
		"a chain with no sources option": sources(`{"nist": {"testnet": {"policy": "first", "sources": ["a"]}}}`),
		"a network of another chain":     sources(`{"stellar": {"regtest": {"policy": "first", "sources": ["a"]}}}`),
		"no names":                       sources(`{"stellar": {"testnet": {"policy": "first", "sources": []}}}`),
		"no sources member":              sources(`{"stellar": {"testnet": {"policy": "first"}}}`),
		"an empty name":                  sources(`{"stellar": {"testnet": {"policy": "first", "sources": [""]}}}`),
		"a name of 17 characters":        sources(`{"stellar": {"testnet": {"policy": "first", "sources": ["abcdefghijklmnopq"]}}}`),
		"a name twice":                   sources(`{"stellar": {"testnet": {"policy": "all", "sources": ["a", "b", "a"]}}}`),
		"an unknown policy":              sources(`{"stellar": {"testnet": {"policy": "most", "sources": ["a", "b"]}}}`),
		"required under first":           sources(`{"stellar": {"testnet": {"policy": "first", "required": 1, "sources": ["a", "b"]}}}`),
		"required under all":             sources(`{"stellar": {"testnet": {"policy": "all", "required": 2, "sources": ["a", "b"]}}}`),
		"at_least without required":      sources(`{"stellar": {"testnet": {"policy": "at_least", "sources": ["a", "b"]}}}`),
		"at_least 0":                     sources(`{"stellar": {"testnet": {"policy": "at_least", "required": 0, "sources": ["a", "b"]}}}`),
		"at_least 3 of two":              sources(`{"stellar": {"testnet": {"policy": "at_least", "required": 3, "sources": ["a", "b"]}}}`),
		"offline":                        `{"environment": "test", ` + networks + `, "sources": {"stellar": {"testnet": {"policy": "first", "sources": ["a"]}}}}`,
		"a bad network after a good one": sources(`{"bitcoin": {"mainnet": {"policy": "first", "sources": ["a"]}, "testnet": {"policy": "most", "sources": ["a", "b"]}}}`),
		"a bad chain after a good one":   sources(`{"bitcoin": {"regtest": {"policy": "first", "sources": ["a"]}}, "stellar": {"testnet": {"policy": "first", "required": 1, "sources": ["a", "b"]}}}`),
	}
	for name, text := range refused {
		if err := vectorcases.CheckSources(config(t, text)); err == nil {
			t.Errorf("%s: allowed", name)
		}
	}
	if err := vectorcases.CheckSources(config(t, `{"environment": "test", `+networks+`}`)); err != nil {
		t.Errorf("no sources, offline: %v", err)
	}
}

// load refuses a case whose sources SPEC.md section 13.2 does not allow,
// here required under first, and its error names the case's id, while the
// same case without required loads. Mutant killed: Load's call of
// checkSources deleted (the case loads).
func TestLoad_RefusesACaseSources(t *testing.T) {
	file := func(sources string) []byte {
		return []byte(`[{"id": "sources-refused", "proof": "positive/block.proof.json", "config": {"environment": "test",
			"networks": {"commitment": {"bitcoin": "regtest", "stellar": "testnet"}, "entropy": {"bitcoin": "mainnet", "stellar": "testnet"}},
			"online": "corpus/replay/sources-all-agree.json", "sources": {"stellar": {"testnet": ` + sources + `}}},
			"expected": {"code": "configuration_error"}}]`)
	}
	if cases, err := vectorcases.LoadBytes(file(`{"policy": "first", "sources": ["a", "b"]}`)); err != nil || len(cases) != 1 {
		t.Fatalf("the allowed case: %d cases, %v", len(cases), err)
	}
	cases, err := vectorcases.LoadBytes(file(`{"policy": "first", "required": 1, "sources": ["a", "b"]}`))
	if err == nil {
		t.Fatalf("loaded %+v", cases)
	}
	if !strings.Contains(err.Error(), "sources-refused") {
		t.Errorf("the error %q does not name the case", err)
	}
}
