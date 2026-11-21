// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package vectorcases_test

import (
	"encoding/json/v2"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// Load reads a case's sources, {<chain>: {<network>: {policy, required,
// sources}}} (SPEC.md section 13.2): 19 cases name them, each online, each
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
	if named != 19 {
		t.Fatalf("%d cases name sources, want 19", named)
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
// one, Stellar after Bitcoin, is allowed); checkSources checking only the
// last network of each chain (a bad network before a good one, Bitcoin
// mainnet before regtest, is allowed); checkSources checking only the last
// chain (a bad chain before a good one, Bitcoin before Stellar, is
// allowed).
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
		"a chain with no sources option":  sources(`{"nist": {"testnet": {"policy": "first", "sources": ["a"]}}}`),
		"a network of another chain":      sources(`{"stellar": {"regtest": {"policy": "first", "sources": ["a"]}}}`),
		"no names":                        sources(`{"stellar": {"testnet": {"policy": "first", "sources": []}}}`),
		"no sources member":               sources(`{"stellar": {"testnet": {"policy": "first"}}}`),
		"an empty name":                   sources(`{"stellar": {"testnet": {"policy": "first", "sources": [""]}}}`),
		"a name of 17 characters":         sources(`{"stellar": {"testnet": {"policy": "first", "sources": ["abcdefghijklmnopq"]}}}`),
		"a name twice":                    sources(`{"stellar": {"testnet": {"policy": "all", "sources": ["a", "b", "a"]}}}`),
		"an unknown policy":               sources(`{"stellar": {"testnet": {"policy": "most", "sources": ["a", "b"]}}}`),
		"required under first":            sources(`{"stellar": {"testnet": {"policy": "first", "required": 1, "sources": ["a", "b"]}}}`),
		"required under all":              sources(`{"stellar": {"testnet": {"policy": "all", "required": 2, "sources": ["a", "b"]}}}`),
		"at_least without required":       sources(`{"stellar": {"testnet": {"policy": "at_least", "sources": ["a", "b"]}}}`),
		"at_least 0":                      sources(`{"stellar": {"testnet": {"policy": "at_least", "required": 0, "sources": ["a", "b"]}}}`),
		"at_least 3 of two":               sources(`{"stellar": {"testnet": {"policy": "at_least", "required": 3, "sources": ["a", "b"]}}}`),
		"offline":                         `{"environment": "test", ` + networks + `, "sources": {"stellar": {"testnet": {"policy": "first", "sources": ["a"]}}}}`,
		"a bad network after a good one":  sources(`{"bitcoin": {"mainnet": {"policy": "first", "sources": ["a"]}, "testnet": {"policy": "most", "sources": ["a", "b"]}}}`),
		"a bad chain after a good one":    sources(`{"bitcoin": {"regtest": {"policy": "first", "sources": ["a"]}}, "stellar": {"testnet": {"policy": "first", "required": 1, "sources": ["a", "b"]}}}`),
		"a bad network before a good one": sources(`{"bitcoin": {"mainnet": {"policy": "most", "sources": ["a"]}, "regtest": {"policy": "first", "sources": ["a", "b"]}}}`),
		"a bad chain before a good one":   sources(`{"bitcoin": {"regtest": {"policy": "all", "required": 2, "sources": ["a", "b"]}}, "stellar": {"testnet": {"policy": "first", "sources": ["a"]}}}`),
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

// Bare is true exactly for an expected output with a code and no label
// (SPEC.md section 13.2): a bare rejection, with or without its
// sub_reason, is bare, and a report is not, whether or not it gives the
// report's code. In cases.json every case is one or the other, and exactly
// one report gives its code: two_steps-policy-derive, failed with
// network_mismatch, the code of its first failing row, of the two its rows
// fail with. Mutants killed: Bare as a code alone (the report that gives
// network_mismatch is bare); Bare as no label alone (an expected output
// with neither is bare).
func TestExpected_Bare(t *testing.T) {
	for text, want := range map[string]bool{
		`{"code": "configuration_error"}`:                             true,
		`{"code": "malformed_json", "sub_reason": "duplicate_name"}`:  true,
		`{"code": "network_mismatch", "label": "failed", "rows": []}`: false,
		`{"label": "verified", "rows": []}`:                           false,
		`{}`:                                                          false,
	} {
		var expected vectorcases.Expected
		if err := json.Unmarshal([]byte(text), &expected); err != nil {
			t.Fatal(err)
		}
		if expected.Bare() != want {
			t.Errorf("%s: Bare() = %v, want %v", text, expected.Bare(), want)
		}
	}

	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	var coded []string
	for _, c := range cases {
		e := c.Expected
		if e.Bare() == (e.Label != "") || e.Bare() != (e.Code != "" && e.Rows == nil) {
			t.Errorf("%s: code %q, label %q, %d rows: neither a bare rejection nor a report", c.ID, e.Code, e.Label, len(e.Rows))
		}
		if e.Bare() || e.Code == "" {
			continue
		}
		coded = append(coded, c.ID)
		var failing []string
		for _, row := range e.Rows {
			if row.Status == "fail" {
				failing = append(failing, *row.Code)
			}
		}
		if e.Label != "failed" || e.Code != "network_mismatch" || !slices.Equal(failing, []string{"network_mismatch", "derivation_mismatch"}) {
			t.Errorf("%s: label %q, code %q, failing rows %q", c.ID, e.Label, e.Code, failing)
		}
	}
	if !slices.Equal(coded, []string{"two_steps-policy-derive"}) {
		t.Fatalf("the reports that give their code: %q, want two_steps-policy-derive alone", coded)
	}
}

// When two entries of a case's sources are refused, checkSources gives the
// refusal of the first in sorted order, chain by chain and then network by
// network: of Bitcoin mainnet's unknown policy and Bitcoin regtest's
// sources with no names, mainnet's; of Bitcoin regtest's required under
// all and Stellar testnet's name given twice, Bitcoin's. Each error is
// exactly that refusal's. Mutants killed: checkSources ranging over the
// networks in reverse order (regtest's refusal is given), over the chains
// in reverse order (Stellar's is given), or in map order unsorted (one of
// the two orders is given in some runs, so the check is made 50 times).
func TestCheckSources_TheFirstRefusalInSortedOrder(t *testing.T) {
	const prefix = `{"environment": "test", "networks": {"commitment": {"bitcoin": "regtest", "stellar": "testnet"}, "entropy": {"bitcoin": "mainnet", "stellar": "testnet"}}, "online": "corpus/replay/sources-all-agree.json", "sources": `
	for body, want := range map[string]string{
		`{"bitcoin": {"regtest": {"policy": "first", "sources": []}, "mainnet": {"policy": "most", "sources": ["a"]}}}`:                                       `sources of bitcoin mainnet: the policy "most"`,
		`{"stellar": {"testnet": {"policy": "all", "sources": ["a", "a"]}}, "bitcoin": {"regtest": {"policy": "all", "required": 2, "sources": ["a", "b"]}}}`: `sources of bitcoin regtest: required under all`,
	} {
		var cfg vectorcases.Config
		if err := json.Unmarshal([]byte(prefix+body+`}`), &cfg); err != nil {
			t.Fatal(err)
		}
		for range 50 {
			if err := vectorcases.CheckSources(cfg); err == nil || err.Error() != want {
				t.Fatalf("%s: %v, want %q", body, err, want)
			}
		}
	}
}
