// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// Step 0's sources (SPEC.md sections 9 and 10.2, step 0). Every HTTP
// answer here is an httptest server's, on loopback, read through the real
// readers ([DefaultIdentityReader]); a test that reads no server uses
// [honestIdentities], which answers from a table and reads nothing. No
// test reads a compiled-in default's URL: [loopbackIdentities] fails a test
// that tries.

// The identity values of the test networks, as their sources give them.
const (
	testnetPassphrase  = "Test SDF Network ; September 2015"
	regtestGenesisHash = "0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206"
	testnetGenesisHash = "000000000933ea01ad0ee984209779baaec3ced90fa3f408719526f8d77f4943"
)

// regtestEsplora is the URL of a regtest Esplora server that only
// [honestIdentities] answers.
const regtestEsplora = "https://regtest.truestamp.example"

// honestValues are the identity values [honestIdentities] gives, by URL:
// each compiled-in default's own network's, and regtest's for
// regtestEsplora.
var honestValues = map[string]string{
	"https://horizon.stellar.org":          ProductionPassphrase,
	"https://horizon-testnet.stellar.org":  testnetPassphrase,
	"https://blockstream.info/api":         ProductionGenesisHash,
	"https://blockstream.info/testnet/api": testnetGenesisHash,
	regtestEsplora:                         regtestGenesisHash,
}

// honestIdentities answers each identity read from honestValues, reading
// nothing; a URL it does not know has no usable answer.
func honestIdentities(s Source) (string, error) {
	if value, ok := honestValues[s.URL]; ok {
		return value, nil
	}
	return "", fmt.Errorf("no answer for %s", s.URL)
}

// readLog records the URL of each identity read a reader makes, which
// step 0 makes all at once.
type readLog struct {
	mu   sync.Mutex
	urls []string
}

// through returns reader, recording each read in the log.
func (l *readLog) through(reader IdentityReader) IdentityReader {
	return func(s Source) (string, error) {
		l.mu.Lock()
		l.urls = append(l.urls, s.URL)
		l.mu.Unlock()
		return reader(s)
	}
}

// read returns the URLs read so far.
func (l *readLog) read() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.urls)
}

// esploraFor is the --esplora a run under environment needs for its
// regtest pin: regtestEsplora outside prod, none under prod.
func esploraFor(environment string) string {
	if environment == "prod" {
		return ""
	}
	return regtestEsplora
}

// withRegtestSource gives an online run under a non-production E_v, which
// pins Bitcoin regtest, an operator Esplora source for it, and
// [honestIdentities] as its identity reader.
func withRegtestSource(cfg Config) Config {
	if cfg.Online && cfg.Environment != "prod" {
		cfg.Sources = append(cfg.Sources, Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: regtestEsplora})
		cfg.ReadIdentity = honestIdentities
	}
	return cfg
}

// node is an httptest server that answers as a source of each provider
// would for one network: Horizon's root, Esplora's block-height/0 and
// Bitcoin Core's getblockhash 0, each from its identity value, or with
// answer when it is set. It counts every request it is sent.
type node struct {
	srv      *httptest.Server
	value    string
	answer   http.HandlerFunc
	requests atomic.Int64
	// auth is the Authorization header of the last request.
	auth atomic.Value
}

func newNode(t *testing.T, value string) *node {
	t.Helper()
	n := &node{value: value}
	n.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.requests.Add(1)
		n.auth.Store(r.Header.Get("Authorization"))
		if n.answer != nil {
			n.answer(w, r)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			_ = json.NewEncoder(w).Encode(map[string]any{"network_passphrase": n.value, "history_latest_ledger": 1})
		case r.Method == http.MethodGet && r.URL.Path == "/block-height/0":
			_, _ = w.Write([]byte(n.value))
		case r.Method == http.MethodPost && r.URL.Path == "/":
			_, _ = fmt.Fprintf(w, `{"result":%q,"error":null,"id":"truestamp-verify"}`, n.value)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(n.srv.Close)
	return n
}

// loopbackIdentities is [DefaultIdentityReader], the real reader, failing
// the test for a source that is not on loopback, so no test reads a
// compiled-in default's URL.
func loopbackIdentities(t *testing.T) IdentityReader {
	t.Helper()
	real := DefaultIdentityReader()
	return func(s Source) (string, error) {
		u, err := url.Parse(s.URL)
		if ip := net.ParseIP(u.Hostname()); err != nil || ip == nil || !ip.IsLoopback() {
			t.Errorf("an identity read of %s, which is not on loopback", s.URL)
			return "", errors.New("not on loopback")
		}
		return real(s)
	}
}

// online is the configuration of an online, keyless run under environment
// with sources, read through reader.
func online(t *testing.T, environment string, reader IdentityReader, sources ...Source) Config {
	t.Helper()
	cfg, err := NewConfig(environment)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Online, cfg.Keyring, cfg.Sources, cfg.ReadIdentity = true, KeyringSource{Kind: SourceKeyless}, sources, reader
	return cfg
}

func horizon(n *node) Source {
	return Source{Chain: ChainStellar, Provider: source.Horizon, URL: n.srv.URL}
}
func esplora(n *node) Source {
	return Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: n.srv.URL}
}
func core(n *node) Source {
	return Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: n.srv.URL}
}

// firstSource is the first of the sources of the run's pin of chain, its
// one source when it has one, and whether it has any.
func firstSource(run *Run, pin, chain string) (SourceRecord, bool) {
	set, ok := run.SourcesFor(pin, chain)
	if !ok || len(set.Sources) == 0 {
		return SourceRecord{}, false
	}
	return set.Sources[0], true
}

// pinsOf writes a record's pins as "commitment:regtest entropy:regtest".
func pinsOf(record SourceRecord) string {
	var out []string
	for _, p := range record.Pins {
		out = append(out, p.Pin+":"+p.Network)
	}
	return strings.Join(out, " ")
}

// An identity mismatch, each way, on each chain, Esplora and Bitcoin Core
// both, through the real readers: a source pinned to the production
// network that gives another network's value, and a source pinned to
// another network that gives the production value, each
// configuration_error with a message that names the source's URL; the
// other chain's operator source is honest, and each mismatched source is
// read once. Mutants killed: allows comparing the network alone (every
// value passes); allows without its production-network side (a regtest
// hash under prod passes); mismatch leaving the source's URL out of its
// message.
func TestSources_AnIdentityMismatchEachWayOnEachChain(t *testing.T) {
	for _, c := range []struct {
		name, environment string
		chain             string
		value             string
		as                func(*node) Source
	}{
		{"Stellar public, another passphrase", "prod", ChainStellar, testnetPassphrase, horizon},
		{"Stellar testnet, the production passphrase", "test", ChainStellar, ProductionPassphrase, horizon},
		{"Esplora mainnet, another hash", "prod", ChainBitcoin, regtestGenesisHash, esplora},
		{"Esplora regtest, the production hash", "test", ChainBitcoin, ProductionGenesisHash, esplora},
		{"Bitcoin Core mainnet, another hash", "prod", ChainBitcoin, testnetGenesisHash, core},
		{"Bitcoin Core regtest, the production hash", "test", ChainBitcoin, ProductionGenesisHash, core},
	} {
		t.Run(c.name, func(t *testing.T) {
			wrong := newNode(t, c.value)
			var other *node
			var sources []Source
			if c.chain == ChainStellar {
				other = newNode(t, map[string]string{"prod": ProductionGenesisHash, "test": regtestGenesisHash}[c.environment])
				sources = []Source{c.as(wrong), esplora(other)}
			} else {
				other = newNode(t, map[string]string{"prod": ProductionPassphrase, "test": testnetPassphrase}[c.environment])
				sources = []Source{c.as(wrong), horizon(other)}
			}
			_, rejection := configure(t, online(t, c.environment, loopbackIdentities(t), sources...))
			if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || !strings.Contains(rejection.Message, wrong.srv.URL) {
				t.Fatalf("%v, want configuration_error naming %s", rejection, wrong.srv.URL)
			}
			if wrong.requests.Load() != 1 {
				t.Fatalf("the source was sent %d requests, want 1", wrong.requests.Load())
			}
		})
	}
}

// An identity read with no usable answer, by a transport failure, HTTP
// 429, HTTP 503, a 3xx, a 404 and an answer that does not parse, from a
// Horizon server, an Esplora server and a Bitcoin Core node, through the
// real readers: never configuration_error, the source recorded unreachable
// with the pins it serves and the reason, one request made and none to
// the 3xx's target, which nothing more is read from; CONFIGURATION passes
// and says how many sources gave no usable answer. Mutants killed: an
// unreachable source given as configuration_error; the redirect followed
// (the target is read); fromRead quoting the body (the reason names it);
// an unreachable source recorded with no pins.
func TestSources_AnIdentityReadWithNoUsableAnswerIsUnreachable(t *testing.T) {
	target := newNode(t, ProductionPassphrase)
	failures := map[string]http.HandlerFunc{
		"a transport failure": func(w http.ResponseWriter, _ *http.Request) {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close()
			}
		},
		"HTTP 429": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "body-of-the-429", http.StatusTooManyRequests)
		},
		"HTTP 503": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "body-of-the-503", http.StatusServiceUnavailable)
		},
		"a 3xx": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.srv.URL+r.URL.Path, http.StatusFound)
		},
		"a 404": func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
		"an answer that does not parse": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html>Just a moment...</html>"))
		},
	}
	for name, answer := range failures {
		for _, provider := range []source.Provider{source.Horizon, source.Esplora, source.BitcoinCore} {
			t.Run(name+", "+string(provider), func(t *testing.T) {
				failing := newNode(t, "")
				failing.answer = answer
				s := Source{Chain: provider.Chain(), Provider: provider, URL: failing.srv.URL}
				honest := newNode(t, testnetPassphrase)
				if s.Chain == ChainStellar {
					honest = newNode(t, regtestGenesisHash)
				}
				sources := []Source{s, {Chain: ChainStellar, Provider: source.Horizon, URL: honest.srv.URL}}
				if s.Chain == ChainStellar {
					sources[1] = Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: honest.srv.URL}
				}
				before := target.requests.Load()
				run, rejection := configure(t, online(t, "test", loopbackIdentities(t), sources...))
				if rejection != nil {
					t.Fatalf("%v, want no rejection", rejection)
				}
				record, ok := firstSource(run, PinCommitment, s.Chain)
				if !ok || record.Status != SourceUnreachable || record.Source.URL != s.URL || record.Reason == "" ||
					pinsOf(record) != map[string]string{ChainBitcoin: "commitment:regtest entropy:regtest", ChainStellar: "commitment:testnet entropy:testnet"}[s.Chain] {
					t.Fatalf("record = %+v, %v", record, ok)
				}
				if strings.Contains(record.Reason, "body-of-the") || strings.Contains(record.Reason, "Just a moment") {
					t.Fatalf("the reason quotes the body: %q", record.Reason)
				}
				if failing.requests.Load() != 1 || target.requests.Load() != before {
					t.Fatalf("%d requests to the source, %d to the redirect's target, want 1 and 0",
						failing.requests.Load(), target.requests.Load()-before)
				}
				if !strings.Contains(run.Rows[0].Message, "1 gave no usable answer") {
					t.Fatalf("CONFIGURATION's message: %s", run.Rows[0].Message)
				}
			})
		}
	}
}

// An online run that pins regtest with no Bitcoin source is
// configuration_error under every E_v that pins it, before any identity
// read, the operator's Horizon server's included; so is one whose only
// Bitcoin source shows mainnet under staging, once it is read, since
// regtest then has no source. Offline, the same run reads nothing and
// passes CONFIGURATION. Mutants killed: DefaultSource giving regtest a
// default; planChain without its check before any read (the Horizon
// server is read first); confirmSources taking an unserved network with no
// default as served.
func TestSources_AnOnlineRunThatPinsRegtestWithNoBitcoinSource(t *testing.T) {
	operatorHorizon := Source{Chain: ChainStellar, Provider: source.Horizon, URL: "https://horizon.truestamp.example"}
	for _, environment := range []string{"dev", "staging", "test"} {
		var log readLog
		reader := log.through(honestIdentities)
		_, rejection := configure(t, online(t, environment, reader, operatorHorizon))
		if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || !strings.Contains(rejection.Message, "regtest") || len(log.read()) != 0 {
			t.Errorf("%s: %v, reads %q", environment, rejection, log.read())
		}
		offline := online(t, environment, reader, operatorHorizon)
		offline.Online = false
		if run, rejection := configure(t, offline); rejection != nil || len(run.Sources) != 0 || len(log.read()) != 0 {
			t.Errorf("%s offline: %v, %+v, reads %q", environment, rejection, run, log.read())
		}
	}
	mainnet := newNode(t, ProductionGenesisHash)
	_, rejection := configure(t, online(t, "staging", func(s Source) (string, error) {
		if s.URL == mainnet.srv.URL {
			return loopbackIdentities(t)(s)
		}
		return honestIdentities(s)
	}, esplora(mainnet)))
	if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || !strings.Contains(rejection.Message, "regtest") {
		t.Fatalf("staging, an Esplora source on mainnet only: %v", rejection)
	}
}

// Sources configured for no network that show one network are its
// sources, in the order of the options, under the policy first, and none
// is refused: under prod, which pins Bitcoin to mainnet alone, an Esplora
// server and a Bitcoin Core node both showing mainnet are both read and
// both serve mainnet's two pins, the Esplora server first, Stellar public
// keeping its one default; under staging, both showing regtest serve
// regtest, Esplora first, and mainnet keeps its default. Each source is
// read once. Mutants killed: planChain refusing more sources configured for
// no network than open networks before any read (prod is
// configuration_error and reads nothing); assign refusing a network that
// two answering sources serve (both runs are configuration_error); assign
// keeping only the first answering source of a network (each set has one
// source); the sets ordered against the options (Bitcoin Core comes
// first).
func TestSources_SourcesForNoNetworkThatShowOneNetworkAreItsSources(t *testing.T) {
	esp := Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://esplora.truestamp.example"}
	cor := Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: "https://core.truestamp.example"}
	for _, c := range []struct {
		environment, value, network string
		others                      []string
	}{
		{"prod", ProductionGenesisHash, "mainnet", []string{"https://blockstream.info/api"}},
		{"staging", regtestGenesisHash, "regtest", nil},
	} {
		var log readLog
		reader := log.through(func(s Source) (string, error) {
			if s.URL == esp.URL || s.URL == cor.URL {
				return c.value, nil
			}
			return honestIdentities(s)
		})
		run, rejection := configure(t, online(t, c.environment, reader, esp, cor))
		if rejection != nil {
			t.Fatalf("%s: %v", c.environment, rejection)
		}
		set, ok := run.SourcesFor(PinCommitment, ChainBitcoin)
		if !ok || set.Network != c.network || set.Policy != (SourcePolicy{Kind: SourcePolicyFirst}) || set.Required() != 1 || len(set.Sources) != 2 ||
			set.Sources[0].Source.URL != esp.URL || set.Sources[1].Source.URL != cor.URL ||
			set.Sources[0].Status != SourceConfirmed || set.Sources[1].Status != SourceConfirmed {
			t.Fatalf("%s: the Bitcoin commitment pin's sources %+v, %v", c.environment, set, ok)
		}
		if c.environment == "prod" {
			if entropy, _ := run.SourcesFor(PinEntropy, ChainBitcoin); len(entropy.Sources) != 2 || entropy.Network != "mainnet" {
				t.Fatalf("prod: the Bitcoin entropy pin's sources %+v", entropy)
			}
		} else if entropy, _ := run.SourcesFor(PinEntropy, ChainBitcoin); len(entropy.Sources) != 1 || entropy.Sources[0].From != FromDefault {
			t.Fatalf("staging: the Bitcoin entropy pin's sources %+v", entropy)
		}
		reads := log.read()
		for _, url := range []string{esp.URL, cor.URL} {
			if n := len(slices.DeleteFunc(slices.Clone(reads), func(u string) bool { return u != url })); n != 1 {
				t.Errorf("%s: %s read %d times, reads %q", c.environment, url, n, reads)
			}
		}
	}
}

// recordsOf writes each of the run's records as "<url> <status> <pins>".
func recordsOf(run *Run) []string {
	var out []string
	for _, r := range run.Sources {
		out = append(out, strings.TrimSpace(r.Source.URL+" "+string(r.Status)+" "+pinsOf(r)))
	}
	return out
}

// A source given for no network whose confirmed answer only pinned
// networks with their own configuration allow serves nothing and is
// recorded confirmed with no pins, whatever other networks are open, and
// no configuration_error comes of it (SPEC.md sections 9 and 10.2, step
// 0); the networks with no configuration keep their other sources or their
// defaults. Under dev: mainnet configured under at_least 2, beside an
// Esplora server showing mainnet and a Bitcoin Core node showing regtest,
// in both orders of the options, the Esplora server serves nothing and the
// node serves regtest; regtest configured under all, beside the same two,
// the Esplora server serves mainnet and the node nothing; regtest
// configured with one source, confirmed or with no usable answer, beside
// an Esplora server showing regtest, which serves nothing, so mainnet's
// default is read. With both networks configured under dev, or mainnet
// configured under prod, the source is never read and has no record. A
// source given for no network whose answer no pinned network of its chain
// allows is still configuration_error beside a configuration: with
// Bitcoin pinned to testnet and regtest, testnet configured, one showing
// mainnet is refused, and the message names both pinned networks. Mutants
// killed: assign holding such a source to the open networks alone (each
// run with one open network is configuration_error); assign never
// refusing such a source (the last run is configuration_error for
// regtest's missing source instead); assign recording it unreachable (it
// then serves mainnet, whose default is not read); assign leaving it out
// of the records; assign giving it the pins of the configured networks it
// fits; mismatch naming the open networks alone (the message names
// regtest only).
func TestSources_ASourceForNoNetworkThatOnlyConfiguredNetworksAllowServesNothing(t *testing.T) {
	esp := Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://floating-esplora.truestamp.example"}
	cor := Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: "https://floating-core.truestamp.example"}
	const blockstream = "https://blockstream.info/api"
	first := SourcePolicy{Kind: SourcePolicyFirst}
	atLeast2 := SourcePolicy{Kind: SourcePolicyAtLeast, N: 2}
	stellarTestnet := "https://horizon-testnet.stellar.org confirmed commitment:testnet entropy:testnet"
	mainnetValues := map[string]string{e1: ProductionGenesisHash, e2: ProductionGenesisHash, esp.URL: ProductionGenesisHash, cor.URL: regtestGenesisHash}
	mainnetSet := "mainnet at_least 2 2: " + e1 + " confirmed operator mainnet entropy:mainnet, " + e2 + " confirmed operator mainnet entropy:mainnet"
	for _, c := range []struct {
		name, environment   string
		values              map[string]string
		sources             []Source
		configured          []NetworkSources
		records             []string
		commitment, entropy string
		read, notRead       []string
	}{
		{"dev, mainnet configured, Esplora then Core", "dev", mainnetValues, []Source{esp, cor},
			[]NetworkSources{configured(ChainBitcoin, "mainnet", atLeast2, e1, e2)},
			[]string{e1 + " confirmed entropy:mainnet", e2 + " confirmed entropy:mainnet", esp.URL + " confirmed", cor.URL + " confirmed commitment:regtest", stellarTestnet},
			"regtest first 1: " + cor.URL + " confirmed operator  commitment:regtest", mainnetSet,
			[]string{esp.URL, cor.URL}, []string{blockstream}},
		{"dev, mainnet configured, Core then Esplora", "dev", mainnetValues, []Source{cor, esp},
			[]NetworkSources{configured(ChainBitcoin, "mainnet", atLeast2, e1, e2)},
			[]string{e1 + " confirmed entropy:mainnet", e2 + " confirmed entropy:mainnet", cor.URL + " confirmed commitment:regtest", esp.URL + " confirmed", stellarTestnet},
			"regtest first 1: " + cor.URL + " confirmed operator  commitment:regtest", mainnetSet,
			[]string{esp.URL, cor.URL}, []string{blockstream}},
		{"dev, regtest configured, Esplora then Core", "dev",
			map[string]string{e1: regtestGenesisHash, e2: regtestGenesisHash, esp.URL: ProductionGenesisHash, cor.URL: regtestGenesisHash},
			[]Source{esp, cor},
			[]NetworkSources{configured(ChainBitcoin, "regtest", SourcePolicy{Kind: SourcePolicyAll}, e1, e2)},
			[]string{e1 + " confirmed commitment:regtest", e2 + " confirmed commitment:regtest", esp.URL + " confirmed entropy:mainnet", cor.URL + " confirmed", stellarTestnet},
			"regtest all 2: " + e1 + " confirmed operator regtest commitment:regtest, " + e2 + " confirmed operator regtest commitment:regtest",
			"mainnet first 1: " + esp.URL + " confirmed operator  entropy:mainnet",
			[]string{esp.URL, cor.URL}, []string{blockstream}},
		{"dev, regtest configured, one source", "dev",
			map[string]string{e1: regtestGenesisHash, esp.URL: regtestGenesisHash}, []Source{esp},
			[]NetworkSources{configured(ChainBitcoin, "regtest", first, e1)},
			[]string{e1 + " confirmed commitment:regtest", esp.URL + " confirmed", blockstream + " confirmed entropy:mainnet", stellarTestnet},
			"regtest first 1: " + e1 + " confirmed operator regtest commitment:regtest",
			"mainnet first 1: " + blockstream + " confirmed default mainnet entropy:mainnet",
			[]string{esp.URL, blockstream}, nil},
		{"dev, regtest configured, its one source unreachable", "dev",
			map[string]string{e1: "", esp.URL: regtestGenesisHash}, []Source{esp},
			[]NetworkSources{configured(ChainBitcoin, "regtest", first, e1)},
			[]string{e1 + " unreachable commitment:regtest", esp.URL + " confirmed", blockstream + " confirmed entropy:mainnet", stellarTestnet},
			"regtest first 1: " + e1 + " unreachable operator regtest commitment:regtest",
			"mainnet first 1: " + blockstream + " confirmed default mainnet entropy:mainnet",
			[]string{esp.URL, blockstream}, nil},
		{"dev, both networks configured", "dev",
			map[string]string{e1: regtestGenesisHash, e2: ProductionGenesisHash, esp.URL: regtestGenesisHash}, []Source{esp},
			[]NetworkSources{configured(ChainBitcoin, "regtest", first, e1), configured(ChainBitcoin, "mainnet", first, e2)},
			[]string{e1 + " confirmed commitment:regtest", e2 + " confirmed entropy:mainnet", stellarTestnet},
			"regtest first 1: " + e1 + " confirmed operator regtest commitment:regtest",
			"mainnet first 1: " + e2 + " confirmed operator mainnet entropy:mainnet",
			nil, []string{esp.URL, blockstream}},
		{"prod, mainnet configured", "prod",
			map[string]string{e1: ProductionGenesisHash, esp.URL: ProductionGenesisHash}, []Source{esp},
			[]NetworkSources{configured(ChainBitcoin, "mainnet", first, e1)},
			[]string{e1 + " confirmed commitment:mainnet entropy:mainnet", "https://horizon.stellar.org confirmed commitment:public entropy:public"},
			"mainnet first 1: " + e1 + " confirmed operator mainnet commitment:mainnet entropy:mainnet",
			"mainnet first 1: " + e1 + " confirmed operator mainnet commitment:mainnet entropy:mainnet",
			nil, []string{esp.URL, blockstream}},
	} {
		var log readLog
		cfg := online(t, c.environment, log.through(valuesReader(c.values)), c.sources...)
		cfg.NetworkSources = c.configured
		run, rejection := configure(t, cfg)
		if rejection != nil {
			t.Errorf("%s: %v", c.name, rejection)
			continue
		}
		if got := recordsOf(run); !slices.Equal(got, c.records) {
			t.Errorf("%s: records\n got %q\nwant %q", c.name, got, c.records)
		}
		commitment, _ := run.SourcesFor(PinCommitment, ChainBitcoin)
		entropy, _ := run.SourcesFor(PinEntropy, ChainBitcoin)
		if setOf(commitment) != c.commitment || setOf(entropy) != c.entropy {
			t.Errorf("%s: the Bitcoin pins\n got %q, %q\nwant %q, %q", c.name, setOf(commitment), setOf(entropy), c.commitment, c.entropy)
		}
		reads := log.read()
		for _, u := range c.read {
			if timesRead(reads, u) != 1 {
				t.Errorf("%s: %s read %d times, want 1", c.name, u, timesRead(reads, u))
			}
		}
		for _, u := range c.notRead {
			if timesRead(reads, u) != 0 {
				t.Errorf("%s: %s was read", c.name, u)
			}
		}
	}

	var log readLog
	cfg := online(t, "test", log.through(valuesReader(map[string]string{e1: testnetGenesisHash, esp.URL: ProductionGenesisHash})), esp)
	cfg.Networks.Commitment.Bitcoin, cfg.Networks.Entropy.Bitcoin = "testnet", "regtest"
	cfg.NetworkSources = []NetworkSources{configured(ChainBitcoin, "testnet", first, e1)}
	_, rejection := configure(t, cfg)
	want := "The Bitcoin source at " + esp.URL + " gives the hash of the block at height 0 of Bitcoin's production network, mainnet, " +
		"and the run pins it to testnet or regtest, so the run gives no verdict."
	if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || rejection.Message != want {
		t.Fatalf("testnet configured, a source for no network showing mainnet: %v\nwant %s", rejection, want)
	}
	if reads := log.read(); len(reads) != 2 || timesRead(reads, e1) != 1 || timesRead(reads, esp.URL) != 1 {
		t.Fatalf("reads %q", reads)
	}
}

// A source given for no network whose answer fits only a configured
// network serves no other network's pin, so an open network it is the
// only candidate for is left with no source (SPEC.md sections 9 and 10.2,
// step 0). Under dev and under staging, with mainnet configured with its
// own source and one Esplora server given for no network alone, showing
// mainnet, regtest has no source and the run is configuration_error for
// it, with the exact message; the configured source and the Esplora
// server are each read once, and nothing else is read, no default
// either. Mutant killed: assign giving a source whose answer fits only
// configured networks the open networks that have no default and that no
// other source serves (the Esplora server serves regtest, and the run
// passes).
func TestSources_ASourceForNoNetworkThatFitsOnlyAConfiguredNetworkLeavesRegtestWithNoSource(t *testing.T) {
	esp := Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://floating-esplora.truestamp.example"}
	values := map[string]string{e1: ProductionGenesisHash, esp.URL: ProductionGenesisHash}
	want := "The run pins Bitcoin's regtest network, and no source serves it, so the run gives no verdict: " +
		"this verifier has no built-in source for regtest, so run again with --esplora URL or --bitcoin-core URL for it."
	for _, environment := range []string{"dev", "staging"} {
		var log readLog
		cfg := online(t, environment, log.through(valuesReader(values)), esp)
		cfg.NetworkSources = []NetworkSources{configured(ChainBitcoin, "mainnet", SourcePolicy{Kind: SourcePolicyFirst}, e1)}
		_, rejection := configure(t, cfg)
		if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || rejection.Message != want {
			t.Errorf("%s: %v\nwant %s", environment, rejection, want)
		}
		if reads := log.read(); len(reads) != 2 || timesRead(reads, e1) != 1 || timesRead(reads, esp.URL) != 1 {
			t.Errorf("%s: reads %q", environment, reads)
		}
	}
}

// A forbidden source given for no network beside one of its chain that
// answers with a value the run's pin allows is configuration_error,
// whatever the other source gives (SPEC.md section 10.2, step 0): under
// prod, an Esplora server confirming mainnet beside a Bitcoin Core node
// giving regtest's hash, and under test, an Esplora server confirming
// regtest beside a node giving the production hash, each in both orders
// of the options. Each message is exact and names the node, and each
// source is read once. Mutant killed: assign recording a forbidden source
// given for no network unreachable when another such source of its chain
// answered (every run passes).
func TestSources_AForbiddenSourceForNoNetworkBesideAnAnsweringOne(t *testing.T) {
	esp := Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://floating-esplora.truestamp.example"}
	cor := Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: "https://floating-core.truestamp.example"}
	for _, c := range []struct {
		environment, good, forbidden, want string
	}{
		{"prod", ProductionGenesisHash, regtestGenesisHash, "The Bitcoin source at " + cor.URL + " gives a hash of the block at height 0 " +
			"other than that of Bitcoin's production network, mainnet, which the run pins it to, so the run gives no verdict."},
		{"test", regtestGenesisHash, ProductionGenesisHash, "The Bitcoin source at " + cor.URL + " gives the hash of the block at height 0 " +
			"of Bitcoin's production network, mainnet, and the run pins it to regtest, so the run gives no verdict."},
	} {
		for _, sources := range [][]Source{{esp, cor}, {cor, esp}} {
			var log readLog
			cfg := online(t, c.environment, log.through(valuesReader(map[string]string{esp.URL: c.good, cor.URL: c.forbidden})), sources...)
			_, rejection := configure(t, cfg)
			if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || rejection.Message != c.want {
				t.Errorf("%s, %s first: %v\nwant %s", c.environment, sources[0].Provider, rejection, c.want)
			}
			if reads := log.read(); timesRead(reads, esp.URL) != 1 || timesRead(reads, cor.URL) != 1 {
				t.Errorf("%s, %s first: reads %q", c.environment, sources[0].Provider, reads)
			}
		}
	}
}

// Which pin each source serves, for every E_v (SPEC.md section 9): under
// prod one Stellar source serves both public pins and one Bitcoin source
// both mainnet pins; under dev and staging the regtest commitment pin and
// the mainnet entropy pin of Bitcoin are served by different sources, an
// operator source by what its identity read shows and a default for the
// other, whichever of Esplora and Bitcoin Core shows which; under test one
// source serves each chain's two pins. Each record says where it came
// from, and SourcesFor finds each pin's one source, which serves it.
// Mutants killed: allows inverted for floating sources (the regtest node
// serves mainnet); pinsFor giving every pin of the chain (the regtest node
// serves the mainnet entropy pin); SourcesFor reading the commitment
// network for every pin (staging's entropy source is the regtest node).
func TestSources_WhichPinEachSourceServesForEveryEnvironment(t *testing.T) {
	regtestCore := Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: "https://core.truestamp.example"}
	mainnetEsplora := Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://esplora.truestamp.example"}
	values := map[string]string{regtestCore.URL: regtestGenesisHash, mainnetEsplora.URL: ProductionGenesisHash}
	reader := func(s Source) (string, error) {
		if value, ok := values[s.URL]; ok {
			return value, nil
		}
		return honestIdentities(s)
	}
	cases := []struct {
		environment string
		sources     []Source
		want        []string
	}{
		{"prod", nil, []string{
			"bitcoin esplora https://blockstream.info/api default commitment:mainnet entropy:mainnet",
			"stellar horizon https://horizon.stellar.org default commitment:public entropy:public",
		}},
		{"prod", []Source{mainnetEsplora}, []string{
			"bitcoin esplora https://esplora.truestamp.example operator commitment:mainnet entropy:mainnet",
			"stellar horizon https://horizon.stellar.org default commitment:public entropy:public",
		}},
		{"dev", []Source{regtestCore}, []string{
			"bitcoin bitcoin_core https://core.truestamp.example operator commitment:regtest",
			"bitcoin esplora https://blockstream.info/api default entropy:mainnet",
			"stellar horizon https://horizon-testnet.stellar.org default commitment:testnet entropy:testnet",
		}},
		{"staging", []Source{mainnetEsplora, regtestCore}, []string{
			"bitcoin esplora https://esplora.truestamp.example operator entropy:mainnet",
			"bitcoin bitcoin_core https://core.truestamp.example operator commitment:regtest",
			"stellar horizon https://horizon-testnet.stellar.org default commitment:testnet entropy:testnet",
		}},
		{"test", []Source{regtestCore}, []string{
			"bitcoin bitcoin_core https://core.truestamp.example operator commitment:regtest entropy:regtest",
			"stellar horizon https://horizon-testnet.stellar.org default commitment:testnet entropy:testnet",
		}},
	}
	for _, c := range cases {
		run, rejection := configure(t, online(t, c.environment, reader, c.sources...))
		if rejection != nil {
			t.Fatalf("%s: %v", c.environment, rejection)
		}
		var got []string
		for _, r := range run.Sources {
			if r.Status != SourceConfirmed {
				t.Fatalf("%s: %+v", c.environment, r)
			}
			got = append(got, strings.Join([]string{r.Source.Chain, string(r.Source.Provider), r.Source.URL, string(r.From), pinsOf(r)}, " "))
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s:\n got %q\nwant %q", c.environment, got, c.want)
		}
		for _, pin := range []string{PinCommitment, PinEntropy} {
			for _, chain := range chains {
				network := run.Config.pinNetwork(pin, chain)
				set, ok := run.SourcesFor(pin, chain)
				if !ok || set.Network != network || len(set.Sources) != 1 ||
					!slices.Contains(set.Sources[0].Pins, ServedPin{Pin: pin, Network: network}) {
					t.Errorf("%s: no source for the %s pin of %s", c.environment, pin, chain)
				}
			}
		}
		if c.environment == "staging" {
			if record, _ := firstSource(run, PinEntropy, ChainBitcoin); record.Source.URL != mainnetEsplora.URL {
				t.Errorf("staging's Bitcoin entropy source: %+v", record)
			}
		}
	}
}

// Under dev and staging, an operator source whose identity read gets no
// usable answer serves every Bitcoin network no source that answered
// serves, so no default is read in its place, and the run reads nothing
// else from it; beside a source that answered, it serves only the other
// network. Mutant killed: assign giving an unreachable source no pins (the
// run takes the default for mainnet).
func TestSources_AnUnreachableOperatorSourceServesWhatNoOtherSourceServes(t *testing.T) {
	down := Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: "https://down.truestamp.example"}
	var log readLog
	reader := log.through(func(s Source) (string, error) {
		if s.URL == down.URL {
			return "", errors.New("HTTP 503")
		}
		return honestIdentities(s)
	})
	run, rejection := configure(t, online(t, "staging", reader, down))
	if rejection != nil {
		t.Fatal(rejection)
	}
	record, _ := firstSource(run, PinEntropy, ChainBitcoin)
	if record.Source.URL != down.URL || record.Status != SourceUnreachable || pinsOf(record) != "commitment:regtest entropy:mainnet" ||
		slices.Contains(log.read(), "https://blockstream.info/api") {
		t.Fatalf("record %+v, reads %q", record, log.read())
	}
	run, rejection = configure(t, online(t, "staging", reader, down, Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: regtestEsplora}))
	if rejection != nil {
		t.Fatal(rejection)
	}
	if record, _ := firstSource(run, PinEntropy, ChainBitcoin); record.Source.URL != down.URL || pinsOf(record) != "entropy:mainnet" {
		t.Fatalf("beside a regtest source: %+v", record)
	}
}

// The run's record of its sources is in CONFIGURATION's values, each
// source with its chain, provider, URL, where it came from, its status,
// the pins it serves, its identity or reason, and its identity read's
// fetched_at, by the verifier's clock when that read came back, and
// cached, always false (SPEC.md sections 10.4, "What a row read", and
// 10.5), as values JCS writes; each record holds its fetched_at too; an
// offline run reads no source and records none. Here the operator's
// Bitcoin source is read first and the default Stellar source after it, so
// the clock's two readings fall to them in that order. Mutants killed:
// sourceValues leaving out the pins; readIdentities stamping no time (both
// fetched_at are the zero time's); one reading taken when step 0 starts
// for every record (the Stellar source's fetched_at is the Bitcoin
// source's); cached written true.
func TestSources_TheRunRecordsEachSource(t *testing.T) {
	first, second := time.Date(2026, 10, 20, 9, 30, 1, 250e6, time.UTC), time.Date(2026, 10, 20, 9, 30, 2, 500e6, time.UTC)
	useClock(t, first, second)
	run, rejection := configure(t, online(t, "test", func(s Source) (string, error) {
		if s.Chain == ChainStellar {
			return "", errors.New("HTTP 429")
		}
		return honestIdentities(s)
	}, Source{Chain: ChainBitcoin, Provider: source.Esplora, URL: regtestEsplora}))
	if rejection != nil {
		t.Fatal(rejection)
	}
	data, err := json.Marshal(run.Rows[0].Values["sources"])
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"cached":false,"chain":"bitcoin","fetched_at":"2026-10-20T09:30:01.250Z","from":"operator","identity":"` + regtestGenesisHash + `","pins":[{"network":"regtest","pin":"commitment"},{"network":"regtest","pin":"entropy"}],"provider":"esplora","status":"confirmed","url":"` + regtestEsplora + `"},` +
		`{"cached":false,"chain":"stellar","fetched_at":"2026-10-20T09:30:02.500Z","from":"default","pins":[{"network":"testnet","pin":"commitment"},{"network":"testnet","pin":"entropy"}],"provider":"horizon","reason":"HTTP 429","status":"unreachable","url":"https://horizon-testnet.stellar.org"}]`
	if string(data) != want {
		t.Fatalf("sources = %s\nwant      %s", data, want)
	}
	if len(run.Sources) != 2 || !run.Sources[0].FetchedAt.Equal(first) || !run.Sources[1].FetchedAt.Equal(second) {
		t.Fatalf("the records' fetched_at: %+v", run.Sources)
	}
}

// The compiled-in defaults are one for each network of spec/networks.json
// but regtest, a Horizon server for Stellar's and an Esplora server for
// Bitcoin's, each on https and pinned to its own network. Mutant killed: a
// default for regtest.
func TestSources_EveryNetworkButRegtestHasADefault(t *testing.T) {
	for chain, networks := range specdata.Networks() {
		for _, network := range networks {
			s, ok := DefaultSource(chain, network)
			if ok != (network != "regtest") {
				t.Errorf("%s %s: a default %v", chain, network, ok)
			}
			if ok && (s.Chain != chain || s.Network != network || s.Provider.Chain() != chain || !strings.HasPrefix(s.URL, "https://") || checkSource(s) != nil) {
				t.Errorf("%s %s: %+v", chain, network, s)
			}
		}
	}
}

// The production values are those the vectors' configuration-online
// cases expect a source to be held to: the replay of
// configuration-online-regtest-mainnet_block0 gives the mainnet hash, and
// that of configuration-online-testnet-production_passphrase the public
// passphrase. Mutant killed: ProductionGenesisHash with one digit
// changed.
func TestSources_TheProductionValues(t *testing.T) {
	if !allows(ChainBitcoin, "mainnet", ProductionGenesisHash) || allows(ChainBitcoin, "regtest", ProductionGenesisHash) ||
		!allows(ChainStellar, "public", ProductionPassphrase) || allows(ChainStellar, "testnet", ProductionPassphrase) {
		t.Fatal("allows")
	}
	if ProductionGenesisHash != "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f" ||
		ProductionPassphrase != "Public Global Stellar Network ; September 2015" {
		t.Fatal("the production values differ from SPEC.md section 10.2, step 0")
	}
}

// echoing answers a request with raw, given the Authorization header the
// request carried, written straight to the connection, so the answer can
// be anything, HTTP or not.
func echoing(raw func(auth string) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = buf.WriteString(raw(auth))
		_ = buf.Flush()
	}
}

// Planted secrets, the Bitcoin Core node's RPC user and password, are
// absent from every byte the CLI could print when the credentials come
// from the environment: the node receives them, as HTTP Basic credentials,
// and echoes the Authorization header in a 401's body, in a 200 answer
// that does not parse, in a malformed status line, in a malformed header
// line, and in a JSON-RPC answer's error.message, result and id, or fails
// in transport (its connection hijacked and closed), while the run, its
// CONFIGURATION row, each record, the configuration printed with every
// fmt verb and as JSON, and every reason hold neither the user, the
// password nor their Basic encoding. Mutants killed: a fromRead
// that gives "no answer: " and the transport error's text (the status
// line's and the header line's echoes are printed); fromRead quoting a
// status error's body (the 401's echo is printed); CoreIdentity quoting
// the answer in the reason of one that does not parse, of a Core error,
// of a result that is no hash or of an answer to another id (each echo is
// printed); the credentials not sent (the node sees no Authorization).
func TestSources_CoreCredentialsNeverReachAnOutput(t *testing.T) {
	const user, secret = "rpcuser-planted-3b9c", "planted-secret-0d2a5f"
	basic := base64.StdEncoding.EncodeToString([]byte(user + ":" + secret))
	for name, answer := range map[string]http.HandlerFunc{
		"a 401 echoing the header": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "unauthorized "+r.Header.Get("Authorization"), http.StatusUnauthorized)
		},
		"a 200 echoing the header": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json " + r.Header.Get("Authorization")))
		},
		"the status line echoing the header": echoing(func(auth string) string { return auth + "\r\n\r\n" }),
		"a header line echoing the header":   echoing(func(auth string) string { return "HTTP/1.1 200 OK\r\n" + auth + "\r\n\r\n" }),
		"error.message echoing the header": func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, `{"result":null,"error":{"code":-1,"message":%q},"id":"truestamp-verify"}`, r.Header.Get("Authorization"))
		},
		"result echoing the header": func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, `{"result":%q,"error":null,"id":"truestamp-verify"}`, r.Header.Get("Authorization"))
		},
		"id echoing the header": func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, `{"result":%q,"error":null,"id":%q}`, regtestGenesisHash, r.Header.Get("Authorization"))
		},
		"a transport failure": echoing(func(string) string { return "" }),
	} {
		t.Run(name, func(t *testing.T) {
			clearTruestampEnv(t)
			n := newNode(t, "")
			n.answer = answer
			t.Setenv("TRUESTAMP_VERIFY_BITCOIN_CORE_USER", user)
			t.Setenv("TRUESTAMP_VERIFY_BITCOIN_CORE_PASSWORD", secret)
			_, options := loadOptions(t, "", "--environment", "test", "--no-keyring", "--bitcoin-core", n.srv.URL, "--horizon", newNode(t, testnetPassphrase).srv.URL)
			cfg, err := options.Config(nil, loopbackIdentities(t))
			if err != nil {
				t.Fatal(err)
			}
			run, rejection, err := Configure(cfg)
			if err != nil || rejection != nil {
				t.Fatal(rejection, err)
			}
			if got, _ := n.auth.Load().(string); got != "Basic "+basic {
				t.Fatalf("the node saw Authorization %q", got)
			}
			record, _ := firstSource(run, PinCommitment, ChainBitcoin)
			if record.Status != SourceUnreachable {
				t.Fatalf("record = %+v", record)
			}
			rowValues, err := json.Marshal(run.Rows[0].Values)
			if err != nil {
				t.Fatal(err)
			}
			configJSON, err := json.Marshal([]any{cfg.Sources, options, run.Sources})
			if err != nil {
				t.Fatal(err)
			}
			printed := strings.Join([]string{
				fmt.Sprintf("%v %+v %#v %s %q %x", cfg, cfg, cfg, cfg.Sources, cfg.Sources, cfg.Sources),
				fmt.Sprintf("%v %+v %#v", options, options, options),
				fmt.Sprintf("%v %+v %#v", run.Sources, run.Sources, *run),
				string(rowValues), string(configJSON), run.Rows[0].Message, record.Reason,
			}, "\n")
			for _, planted := range []string{user, secret, basic} {
				if strings.Contains(printed, planted) {
					t.Fatalf("%q is printed:\n%s", planted, printed)
				}
			}
			if !strings.Contains(printed, source.Masked) {
				t.Fatalf("the credentials are not shown masked:\n%s", printed)
			}
		})
	}
}

// DefaultIdentityReader builds each read's SingleRead when the read is
// made, never before, with the default transport and SourceReadDeadline,
// 9 s, the bound of each of the Truestamp application's verification
// reads. Mutants killed: DefaultIdentityReader building its SingleRead
// once, when it is called (a reader is built with no read made);
// DefaultIdentityReader reading through a SingleRead built at package init
// (no read is built through the seam); SourceReadDeadline of 30 s.
func TestSources_TheDefaultReaderBuildsEachReadWithItsDeadline(t *testing.T) {
	var built []string
	saved := newSingleRead
	newSingleRead = func(transport http.RoundTripper, deadline time.Duration) *httpclient.SingleRead {
		built = append(built, fmt.Sprintf("%v %v", transport, deadline))
		return saved(transport, deadline)
	}
	defer func() { newSingleRead = saved }()
	reader := DefaultIdentityReader()
	if len(built) != 0 {
		t.Fatalf("a reader built %d reads before any read", len(built))
	}
	n := newNode(t, regtestGenesisHash)
	for range 2 {
		if value, err := reader(esplora(n)); err != nil || value != regtestGenesisHash {
			t.Fatalf("%q, %v", value, err)
		}
	}
	if SourceReadDeadline != 9*time.Second || len(built) != 2 || built[0] != "<nil> 9s" || built[1] != built[0] {
		t.Fatalf("reads built %q", built)
	}
}

// A source no run can read is an invalid configuration, whose error never
// quotes the source's URL: a provider of another chain, a network not of
// its chain, a URL with user information, a query or a host that is not a
// DNS name, credentials on a source that is not a Bitcoin Core node, and a
// source given for no network that names one, even one of its chain, and
// NIST's beacon as a source of any chain or of none, since it is one
// source at its compiled-in address and no configuration names another
// (SPEC.md section 9). Mutants killed: checkSource without its provider
// check; without its credentials check; checkSourceURL without its host
// check; Validate without its refusal of a network on a source of
// Config.Sources (the source naming testnet is allowed); the beacon's
// provider serving a chain named nist (the beacon of the chain nist is
// allowed).
func TestConfig_SourceRefusals(t *testing.T) {
	const secret = "planted-secret-2e8a"
	for name, s := range map[string]Source{
		"Horizon for Bitcoin":        {Chain: ChainBitcoin, Provider: source.Horizon, URL: "https://h.example"},
		"no chain":                   {Provider: source.Esplora, URL: "https://e.example"},
		"a network of another chain": {Chain: ChainStellar, Provider: source.Horizon, URL: "https://h.example", Network: "mainnet"},
		"a network named":            {Chain: ChainStellar, Provider: source.Horizon, URL: "https://h.example", Network: "testnet"},
		"user information":           {Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: "http://u:" + secret + "@127.0.0.1:8332"},
		"a query":                    {Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://e.example/?key=" + secret},
		"a host that is no DNS name": {Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://" + secret + "..example"},
		"credentials on Esplora": {Chain: ChainBitcoin, Provider: source.Esplora, URL: "https://e.example",
			Credentials: source.NewCredentials("u", secret)},
		"the NIST beacon":             {Chain: SourceNIST, Provider: source.NISTBeacon, URL: "https://beacon.example"},
		"the NIST beacon for Bitcoin": {Chain: ChainBitcoin, Provider: source.NISTBeacon, URL: "https://beacon.example"},
		"the NIST beacon of no chain": {Provider: source.NISTBeacon, URL: "https://beacon.example"},
	} {
		cfg, _ := NewConfig("prod")
		cfg.Sources = []Source{s}
		err := cfg.Validate()
		if !errors.Is(err, ErrInvalidConfig) || strings.Contains(err.Error(), secret) {
			t.Errorf("%s: %v", name, err)
		}
	}
	cfg, _ := NewConfig("prod")
	cfg.Sources = []Source{{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: "http://127.0.0.1:8332", Credentials: source.NewCredentials("u", secret)}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("a Bitcoin Core node with credentials: %v", err)
	}
}

// A compiled-in default is held to its chain's production identity both
// ways, as an operator source is: an identity reader answers a default's
// URL with the wrong value, and each run is configuration_error with a
// message naming the source. Under prod, horizon.stellar.org gives the
// testnet passphrase, and the mainnet Esplora default a regtest hash;
// under dev, beside a regtest operator Esplora server, the testnet Horizon
// default gives the production passphrase, and the mainnet Esplora default
// a regtest hash; under test, the testnet Horizon default gives the
// production passphrase. Mutants killed: a default's identity left
// unchecked; a default checked on one side only, that a default of the
// production network gives the production value (a test network's default
// giving it passes), or that a default of another network does not (a
// production default giving another network's value passes); a default's
// mismatch recorded unreachable.
func TestSources_ADefaultIsCheckedBothWays(t *testing.T) {
	for _, c := range []struct{ environment, url, value string }{
		{"prod", "https://horizon.stellar.org", testnetPassphrase},
		{"prod", "https://blockstream.info/api", regtestGenesisHash},
		{"dev", "https://horizon-testnet.stellar.org", ProductionPassphrase},
		{"dev", "https://blockstream.info/api", regtestGenesisHash},
		{"test", "https://horizon-testnet.stellar.org", ProductionPassphrase},
	} {
		var log readLog
		reader := log.through(func(s Source) (string, error) {
			if s.URL == c.url {
				return c.value, nil
			}
			return honestIdentities(s)
		})
		var sources []Source
		if c.environment != "prod" {
			sources = []Source{{Chain: ChainBitcoin, Provider: source.Esplora, URL: regtestEsplora}}
		}
		_, rejection := configure(t, online(t, c.environment, reader, sources...))
		if rejection == nil || rejection.Code != specdata.ReasonConfigurationError || !strings.Contains(rejection.Message, printedURL(c.url)+" gives") ||
			!slices.Contains(log.read(), c.url) {
			t.Errorf("%s, %s giving %s: %v, reads %q", c.environment, c.url, c.value, rejection, log.read())
		}
	}
}

// pathAnswering makes n answer every request as its provider would,
// whatever the path, so a source URL with a path reaches it.
func pathAnswering(n *node) *node {
	n.answer = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = fmt.Fprintf(w, `{"result":%q,"error":null,"id":"truestamp-verify"}`, n.value)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"network_passphrase": n.value})
	}
	return n
}

// A token planted in a source URL's path, given through the options and
// Options.Config, is read with (the node sees it in the request's path)
// and printed nowhere: not in CONFIGURATION's values or message, a
// mismatch's message, an unreachable reason or an error. Each printed URL
// is scheme://host:port, and the two sources stay told apart by their
// provider and pins. Mutants killed: sourceValues printing the whole URL;
// mismatch printing the whole URL; a fromRead whose unreachable
// reason quotes the transport error, which names the whole URL;
// printedURL dropping the port.
func TestSources_APrintedURLIsItsSchemeHostAndPort(t *testing.T) {
	const token = "tok-PLANTED-5e21"
	for _, c := range []struct {
		name   string
		value  string
		closed bool
	}{
		{"confirmed", regtestGenesisHash, false},
		{"a mismatch", ProductionGenesisHash, false},
		{"unreachable", "", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			clearTruestampEnv(t)
			horizonNode := pathAnswering(newNode(t, testnetPassphrase))
			coreNode := pathAnswering(newNode(t, c.value))
			var mu sync.Mutex
			var paths []string
			answer := coreNode.answer
			coreNode.answer = func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				paths = append(paths, r.URL.Path)
				mu.Unlock()
				answer(w, r)
			}
			if c.closed {
				coreNode.srv.Close()
			}
			_, options := loadOptions(t, "", "--environment", "test", "--no-keyring",
				"--bitcoin-core", coreNode.srv.URL+"/"+token+"/", "--horizon", horizonNode.srv.URL+"/"+token+"/h/")
			cfg, err := options.Config(nil, loopbackIdentities(t))
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range cfg.Sources {
				if !strings.Contains(s.URL, token) {
					t.Fatalf("the configuration's source URL %q lost its path", s.URL)
				}
			}
			run, rejection, err := Configure(cfg)
			printed := []string{fmt.Sprint(err)}
			switch {
			case c.name == "a mismatch":
				if rejection == nil || !strings.Contains(rejection.Message, "source at "+coreNode.srv.URL+" gives") {
					t.Fatalf("%v, %v", rejection, err)
				}
				printed = append(printed, rejection.Message)
			case rejection != nil || err != nil:
				t.Fatalf("%v, %v", rejection, err)
			default:
				values, _ := json.Marshal(run.Rows[0].Values)
				printed = append(printed, string(values), run.Rows[0].Message)
				var got []string
				for _, r := range run.Sources {
					printed = append(printed, r.Reason)
					got = append(got, strings.Join([]string{string(r.Source.Provider), printedURL(r.Source.URL), string(r.Status), pinsOf(r)}, " "))
				}
				want := []string{
					"bitcoin_core " + coreNode.srv.URL + " confirmed commitment:regtest entropy:regtest",
					"horizon " + horizonNode.srv.URL + " confirmed commitment:testnet entropy:testnet",
				}
				if c.closed {
					want[0] = "bitcoin_core " + coreNode.srv.URL + " unreachable commitment:regtest entropy:regtest"
				}
				if !slices.Equal(got, want) || !strings.Contains(string(values), `"url":"`+coreNode.srv.URL+`"`) ||
					!strings.Contains(string(values), `"url":"`+horizonNode.srv.URL+`"`) {
					t.Fatalf("sources %q, values %s; want %q", got, values, want)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if !c.closed && !slices.Equal(paths, []string{"/" + token + "/"}) {
				t.Fatalf("the node saw the paths %q", paths)
			}
			for _, p := range printed {
				if strings.Contains(p, token) {
					t.Fatalf("the token is printed: %s", p)
				}
			}
		})
	}
}

// Under dev and staging, which pin Bitcoin to regtest and mainnet, of an
// Esplora server and a Bitcoin Core node that both give no usable identity
// answer (HTTP 503 and HTTP 500), the first in the options' order, the
// Esplora server, serves both Bitcoin pins and the node serves none: each
// is recorded unreachable, each was sent one request, and no default is
// read in their place. Mutants killed: assign giving each unreachable
// source every network no source that answered serves (the node serves
// both pins too); the order reversed (the node serves them).
func TestSources_OfTwoUnreachableBitcoinSourcesTheFirstServesThePins(t *testing.T) {
	for _, environment := range []string{"dev", "staging"} {
		t.Run(environment, func(t *testing.T) {
			clearTruestampEnv(t)
			esploraNode, coreNode := newNode(t, ""), newNode(t, "")
			esploraNode.answer = func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusServiceUnavailable) }
			coreNode.answer = func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusInternalServerError) }
			_, options := loadOptions(t, "", "--environment", environment, "--no-keyring",
				"--bitcoin-core", coreNode.srv.URL, "--esplora", esploraNode.srv.URL, "--horizon", newNode(t, testnetPassphrase).srv.URL)
			cfg, err := options.Config(nil, loopbackIdentities(t))
			if err != nil {
				t.Fatal(err)
			}
			run, rejection := configure(t, cfg)
			if rejection != nil {
				t.Fatal(rejection)
			}
			var got []string
			for _, r := range run.Sources {
				if r.Source.Chain == ChainBitcoin {
					got = append(got, strings.Join([]string{string(r.Source.Provider), string(r.Status), pinsOf(r)}, " "))
				}
			}
			want := []string{"esplora unreachable commitment:regtest entropy:mainnet", "bitcoin_core unreachable "}
			if !slices.Equal(got, want) {
				t.Fatalf("Bitcoin sources %q, want %q", got, want)
			}
			for _, pin := range []string{PinCommitment, PinEntropy} {
				if record, ok := firstSource(run, pin, ChainBitcoin); !ok || record.Source.Provider != source.Esplora {
					t.Fatalf("the %s pin's source: %+v", pin, record)
				}
			}
			if esploraNode.requests.Load() != 1 || coreNode.requests.Load() != 1 {
				t.Fatalf("%d and %d requests, want 1 each", esploraNode.requests.Load(), coreNode.requests.Load())
			}
		})
	}
}
