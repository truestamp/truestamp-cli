// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// This file keeps every online `truestamp verify` of this package's tests
// off the network, and holds the runs that need a source to answer.
//
//   - The binary every test runs is built with the verifier's two
//     compiled-in addresses, Truestamp's keyring (www.truestamp.com) and
//     NIST's Randomness Beacon, set at link time (-ldflags -X) to
//     deadEndpoints, a loopback port nothing listens on; a test that needs
//     them to answer builds its own binary with buildVerifyBinary, which
//     takes loopback addresses only. No flag, configuration key or
//     environment variable reaches either address, so the shipped binary
//     reads www.truestamp.com and https://beacon.nist.gov/beacon/2.0
//     (TestShippedBuilds_SetNoVerifierAddress).
//   - Every run of `truestamp verify` goes through verifyCLI, which refuses
//     an online run that leaves a network its E_v pins without a loopback
//     source, names a source or a --base-url off loopback, or would leave a
//     pinned network to its built-in source ([checkOnline]); and
//     TestVerifyRuns_GoThroughVerifyCLI scans this package's tests for a
//     run of verify, or of the test binary, that does not go through it.

// verifierEndpoints are the verifier's two compiled-in addresses, as the
// linker sets them.
type verifierEndpoints struct {
	keyring, nist string
}

// deadEndpoints point both addresses at loopback port 9, where nothing
// listens.
var deadEndpoints = verifierEndpoints{
	keyring: "http://127.0.0.1:9/.well-known/keyring.json",
	nist:    "http://127.0.0.1:9/nist",
}

// verifierEndpointFlags are the linker flags that set e.
func verifierEndpointFlags(e verifierEndpoints) string {
	const pkg = "github.com/truestamp/truestamp-cli/internal/verifier"
	return fmt.Sprintf("-X %s.currentKeyringRead=%s -X %s.nistBeaconBase=%s", pkg, e.keyring, pkg, e.nist)
}

// builtBinaries are the binaries buildVerifyBinary built, which verifyCLI
// runs beside binaryPath.
var builtBinaries sync.Map

// buildVerifyBinary builds the CLI with the verifier's addresses set to e,
// both on loopback, with tags as its build tags and extra as more linker
// flags, and gives its path.
func buildVerifyBinary(t *testing.T, e verifierEndpoints, tags string, extra ...string) string {
	t.Helper()
	for _, raw := range []string{e.keyring, e.nist} {
		if !loopback(raw) {
			t.Fatalf("buildVerifyBinary: %s is not on loopback", raw)
		}
	}
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "truestamp-verify")
	args := []string{"build", "-buildvcs=false", "-ldflags", strings.Join(append([]string{verifierEndpointFlags(e)}, extra...), " ")}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	cmd := exec.Command("go", append(args, "-o", out, "./cmd/truestamp")...)
	cmd.Dir = modRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the binary: %v\n%s", err, b)
	}
	builtBinaries.Store(out, true)
	return out
}

// loopback reports whether raw is an http or https URL whose host is
// loopback.
func loopback(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	return u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
}

// verifyCLI runs `truestamp verify` with args through bin, binaryPath or a
// binary of buildVerifyBinary, with env added to cleanEnv and stdin as its
// standard input, and gives its stdout, its stderr and its exit code, as
// [verifyCLITo] runs it.
func verifyCLI(t *testing.T, bin string, env []string, stdin io.Reader, args ...string) (string, string, int) {
	t.Helper()
	var stdout bytes.Buffer
	stderr, code := verifyCLITo(t, &stdout, bin, env, stdin, args...)
	return stdout.String(), stderr, code
}

// verifyCLITo runs `truestamp verify` as verifyCLI does, with stdout as
// its standard output, and gives its stderr and its exit code. It refuses
// an online run that does not keep to loopback ([checkOnline]), and fails
// the test for an online run whose --json report names a built-in source
// in its CONFIGURATION row ([builtInSources]).
func verifyCLITo(t *testing.T, stdout io.Writer, bin string, env []string, stdin io.Reader, args ...string) (string, int) {
	t.Helper()
	if _, built := builtBinaries.Load(bin); bin != binaryPath && !built {
		t.Fatalf("verifyCLI: %s is not a binary of this package's tests", bin)
	}
	online := checkOnline(t, env, args, stdin)
	cmd := exec.Command(bin, append([]string{"verify"}, args...)...)
	cmd.Env = append(cleanEnv(), env...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	var stderr, copied bytes.Buffer
	cmd.Stdout, cmd.Stderr = stdout, &stderr
	if online {
		cmd.Stdout = io.MultiWriter(stdout, &copied)
	}
	err := cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	case err != nil:
		t.Fatalf("running verify: %v", err)
	}
	if online {
		builtInSources(t, copied.String(), args)
	}
	return stderr.String(), code
}

// checkOnline fails the test for a run of verify, with args, env and stdin,
// that could read off loopback, and reports whether the run is online and
// names an input. Every source it gives, from a flag, a TRUESTAMP_VERIFY_*
// variable of env, or the configuration file --config names (its source
// keys and its [verify.sources.<chain>.<network>] tables), must be on
// loopback, and so must a --base-url, which moves the keyring read, unless
// the run is offline. Online, every network its E_v pins (internal/specdata's
// default pins: one Stellar network, and one Bitcoin network under prod and
// test, two under dev and staging) needs a loopback source: a table for
// exactly that chain and network, or a floating source of the chain (one
// configured for no network) that serves it. Of the floating sources, an
// unreachable one serves every network no other serves, and one that
// answers, a stub of newChainStub's, serves the network its identity read
// gives, unless an earlier one serves it (SPEC.md section 9):
// under dev and staging both Bitcoin sources, an Esplora server and a
// Bitcoin Core node, so that each pinned Bitcoin network has one. A run
// whose source gives an identity no pinned network allows is refused as
// configuration_error before any built-in source is read, and one asking
// for help, naming no input, or naming an E_v that is none, is refused
// before anything is read.
func checkOnline(t *testing.T, env, args []string, stdin io.Reader) bool {
	t.Helper()
	given := map[string]string{}
	offline, input := false, stdin != nil
	for i := 0; i < len(args); i++ {
		name, value, hasValue := strings.Cut(args[i], "=")
		switch {
		case name == "--help" || name == "-h":
			return false
		case name == "--offline":
			offline = !hasValue || value == "true"
		case name == "--file" || name == "--url":
			input = input || !hasValue || value != ""
		case slices.Contains([]string{"--environment", "--horizon", "--esplora", "--bitcoin-core", "--config", "--base-url"}, name):
			if !hasValue && i+1 < len(args) {
				i++
				value = args[i]
			}
			given[name] = value
		case !strings.HasPrefix(args[i], "-") && args[i] != "":
			input = true
		}
	}
	floating := map[string][]string{}
	for _, flag := range []string{"--horizon", "--esplora", "--bitcoin-core"} {
		if v := given[flag]; v != "" {
			floating[flag] = append(floating[flag], v)
		}
	}
	for _, e := range env {
		name, value, _ := strings.Cut(e, "=")
		switch name {
		case "TRUESTAMP_VERIFY_OFFLINE":
			offline = offline || value == "true" || value == "1"
		case "TRUESTAMP_VERIFY_HORIZON":
			floating["--horizon"] = append(floating["--horizon"], value)
		case "TRUESTAMP_VERIFY_ESPLORA":
			floating["--esplora"] = append(floating["--esplora"], value)
		case "TRUESTAMP_VERIFY_BITCOIN_CORE":
			floating["--bitcoin-core"] = append(floating["--bitcoin-core"], value)
		}
	}
	tables := map[[2]string][]string{}
	if path := given["--config"]; path != "" {
		cfg, err := config.Load(path, nil)
		if err != nil {
			t.Fatalf("checkOnline: the configuration file: %v", err)
		}
		offline = offline || cfg.Verify.Offline
		for flag, v := range map[string]string{"--horizon": cfg.Verify.Horizon, "--esplora": cfg.Verify.Esplora, "--bitcoin-core": cfg.Verify.BitcoinCore} {
			if v != "" {
				floating[flag] = append(floating[flag], v)
			}
		}
		for chain, networks := range cfg.Verify.Sources {
			for network, n := range networks {
				for _, entry := range n.Sources {
					for _, v := range entry {
						tables[[2]string{chain, network}] = append(tables[[2]string{chain, network}], v)
					}
				}
			}
		}
	}
	for flag, urls := range floating {
		for _, raw := range urls {
			if !loopbackWithUserinfo(raw) {
				t.Fatalf("checkOnline: the source %s of %s is not on loopback", raw, flag)
			}
		}
	}
	for at, urls := range tables {
		for _, raw := range urls {
			if !loopbackWithUserinfo(raw) {
				t.Fatalf("checkOnline: the source %s of [verify.sources.%s.%s] is not on loopback", raw, at[0], at[1])
			}
		}
	}
	if base, named := given["--base-url"]; named && !offline && !loopback(base) {
		t.Fatalf("checkOnline: an online run of verify %q reads its keyring from --base-url %s, which is not on loopback", args, base)
	}
	if offline || !input {
		return false
	}
	environment, named := given["--environment"]
	if !named {
		environment = "prod"
	}
	pins, ok := specdata.DefaultNetworkPins(environment)
	if !ok {
		return false
	}
	pinned := map[string][]string{
		verifierChainStellar: slices.Compact([]string{pins.Commitment.Stellar, pins.Entropy.Stellar}),
		verifierChainBitcoin: slices.Compact([]string{pins.Commitment.Bitcoin, pins.Entropy.Bitcoin}),
	}
	// The floating sources of each chain, in the order the run reads them,
	// one of each kind: a flag's over a variable's over the file's.
	floatingOf := map[string][][2]string{}
	for _, flag := range []string{"--horizon", "--esplora", "--bitcoin-core"} {
		if urls := floating[flag]; len(urls) > 0 {
			chain := verifierChainBitcoin
			if flag == "--horizon" {
				chain = verifierChainStellar
			}
			floatingOf[chain] = append(floatingOf[chain], [2]string{flag, urls[0]})
		}
	}
	var unserved []string
	for _, chain := range []string{verifierChainStellar, verifierChainBitcoin} {
		var open []string
		for _, network := range pinned[chain] {
			if len(tables[[2]string{chain, network}]) == 0 {
				open = append(open, network)
			}
		}
		served := map[string]bool{}
		unreachable := false
		for _, s := range floatingOf[chain] {
			network, answers := stubNetwork(s[0], s[1])
			switch {
			case !answers:
				unreachable = true
			case !slices.Contains(pinned[chain], network):
				return true
			case slices.Contains(open, network):
				served[network] = true
			}
		}
		for _, network := range open {
			if !served[network] && !unreachable {
				unserved = append(unserved, chain+" "+network)
			}
		}
	}
	if len(unserved) > 0 {
		t.Fatalf("checkOnline: an online run of verify %q gives no loopback source for the pinned %s, so its built-in source would be read", args, strings.Join(unserved, " and "))
	}
	return true
}

// The chains of a source, as internal/verifier names them.
const (
	verifierChainStellar = "stellar"
	verifierChainBitcoin = "bitcoin"
)

// stubNetwork is the network a floating source of flag at raw shows: for a
// stub of newChainStub's, the network of the identity it gives that kind
// of source, and true; for any other address, which no test serves a chain
// from, false, as for an unreachable source.
func stubNetwork(flag, raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	found, ok := chainStubs.Load(u.Scheme + "://" + u.Host)
	if !ok {
		return "", false
	}
	s := found.(*chainStub)
	s.mu.Lock()
	defer s.mu.Unlock()
	identity := s.genesis
	switch flag {
	case "--horizon":
		identity = s.passphrase
	case "--bitcoin-core":
		identity = s.coreIdentity()
	}
	return map[string]string{
		publicStellar: "public", testnetStellar: "testnet",
		mainnetGenesis: "mainnet", regtestGenesis: "regtest",
	}[identity], true
}

// builtInSources fails the test when stdout is a report whose
// CONFIGURATION row names a source from a built-in default, which an
// online run of these tests never reads.
func builtInSources(t *testing.T, stdout string, args []string) {
	t.Helper()
	var report verifyReport
	if json.Unmarshal([]byte(stdout), &report) != nil {
		return
	}
	for _, row := range report.Rows {
		if row.Step != "CONFIGURATION" {
			continue
		}
		sources, _ := row.Values["sources"].([]any)
		for _, s := range sources {
			if record, _ := s.(map[string]any); record["from"] == "default" {
				t.Errorf("an online run of verify %q read the built-in source %v", args, record["url"])
			}
		}
	}
}

// loopbackWithUserinfo reports whether raw is a URL on loopback once any
// user information is set aside, as a test that gives a source URL
// carrying credentials, which the verifier refuses before any read, does.
func loopbackWithUserinfo(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	u.User = nil
	return loopback(u.String())
}

// chainStub is a loopback server standing in for Horizon (under
// /horizon), Esplora (under /esplora), a Bitcoin Core node (at /core) and
// Truestamp's keyring (/.well-known/keyring.json), answering the reads of
// the proof testProof rebuilds: its Stellar transaction holds memo, and
// its genesis.bitcoin is the block at height 212.
type chainStub struct {
	srv *httptest.Server

	mu sync.Mutex
	// passphrase and genesis are the identity values the stub gives: the
	// Stellar network passphrase and the hash of Bitcoin's block at height
	// 0, which the Esplora server gives, and the Bitcoin Core node too
	// when coreGenesis is "".
	passphrase, genesis string
	// coreGenesis is the hash of the block at height 0 the Bitcoin Core
	// node gives, when it is not genesis: a stub whose two Bitcoin
	// sources show two networks, as an online run under dev or staging
	// needs (checkOnline).
	coreGenesis string
	// memo is the memo of the Stellar transaction, 32 bytes.
	memo []byte
	// keyring is the keyring the stub serves.
	keyring []byte
	// requests are the requests made, "<method> <path>", and
	// authorizations the Authorization header of each Bitcoin Core call.
	requests, authorizations []string
}

// The answers of the stub's chains.
const (
	stubLedger      = 1000327
	stubClosedAt    = "2026-09-26T01:57:15Z"
	stubLedgerHash  = "39ac4b6d995a7d80558fa336d5aa11d2071692e4e65cbff913c0db7d54d0825a"
	stubHeaderHash  = "266bee81d2822091039a920177dbd44b264e95905e9356d868ff9cffe07a19da"
	stubMedianTime  = 1790386620
	stubTipHeight   = 1000
	regtestGenesis  = "0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206"
	mainnetGenesis  = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	publicStellar   = "Public Global Stellar Network ; September 2015"
	testnetStellar  = "Test SDF Network ; September 2015"
	stubGenesisPath = "/esplora/block-height/212"
)

// chainStubs are the stubs newChainStub started, by URL, whose answers
// checkOnline reads.
var chainStubs sync.Map

// newChainStub starts a stub giving the production identities.
func newChainStub(t *testing.T) *chainStub {
	t.Helper()
	s := &chainStub{passphrase: publicStellar, genesis: mainnetGenesis}
	s.srv = httptest.NewServer(http.HandlerFunc(s.serve))
	chainStubs.Store(s.srv.URL, s)
	t.Cleanup(func() {
		chainStubs.Delete(s.srv.URL)
		s.srv.Close()
	})
	return s
}

// coreIdentity is the hash of the block at height 0 the stub's Bitcoin
// Core node gives; its caller holds the lock.
func (s *chainStub) coreIdentity() string {
	if s.coreGenesis != "" {
		return s.coreGenesis
	}
	return s.genesis
}

// set changes the stub's answers under its lock.
func (s *chainStub) set(change func(*chainStub)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	change(s)
}

// seen gives the requests and Bitcoin Core authorizations the stub saw.
func (s *chainStub) seen() (requests, authorizations []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests), slices.Clone(s.authorizations)
}

func (s *chainStub) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests = append(s.requests, r.Method+" "+r.URL.Path)
	passphrase, genesis, coreGenesis, memo, keyring := s.passphrase, s.genesis, s.coreIdentity(), s.memo, s.keyring
	if r.URL.Path == "/core" {
		s.authorizations = append(s.authorizations, r.Header.Get("Authorization"))
	}
	s.mu.Unlock()
	answer := func(v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, v)
	}
	block := map[string]any{"height": 212, "mediantime": stubMedianTime, "timestamp": stubMedianTime, "tx_count": 1}
	switch path := r.URL.Path; {
	case path == "/.well-known/keyring.json" && keyring != nil:
		_, _ = w.Write(keyring)
	case path == "/horizon/" || path == "/horizon":
		answer(map[string]any{"network_passphrase": passphrase})
	case strings.HasPrefix(path, "/horizon/transactions/") && memo != nil:
		answer(map[string]any{"successful": true, "memo_type": "hash", "memo": base64.StdEncoding.EncodeToString(memo), "ledger": stubLedger})
	case path == "/horizon/ledgers/"+strconv.Itoa(stubLedger):
		answer(map[string]any{"closed_at": stubClosedAt, "hash": stubLedgerHash})
	case path == "/esplora/block-height/0":
		_, _ = io.WriteString(w, genesis)
	case path == "/esplora/blocks/tip/height":
		_, _ = io.WriteString(w, strconv.Itoa(stubTipHeight))
	case path == stubGenesisPath:
		_, _ = io.WriteString(w, stubHeaderHash)
	case path == "/esplora/block/"+stubHeaderHash:
		answer(block)
	case path == "/core" && r.Method == http.MethodPost:
		var call struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		if err := json.UnmarshalRead(r.Body, &call); err != nil {
			http.Error(w, "bad call", http.StatusBadRequest)
			return
		}
		var result any
		switch {
		case call.Method == "getblockhash" && len(call.Params) == 1 && call.Params[0] == float64(0):
			result = coreGenesis
		case call.Method == "getblockhash" && len(call.Params) == 1 && call.Params[0] == float64(212):
			result = stubHeaderHash
		case call.Method == "getblockcount":
			result = stubTipHeight
		case call.Method == "getblockheader":
			result = map[string]any{"height": 212, "mediantime": stubMedianTime, "time": stubMedianTime, "nTx": 1}
		default:
			w.WriteHeader(http.StatusInternalServerError)
			answer(map[string]any{"result": nil, "error": map[string]any{"code": -8, "message": "no"}, "id": call.ID})
			return
		}
		answer(map[string]any{"result": result, "error": nil, "id": call.ID})
	default:
		http.NotFound(w, r)
	}
}

// URL is the stub's base URL, http://127.0.0.1:<port>.
func (s *chainStub) URL() string { return s.srv.URL }

// productionProof is the proof testProof rebuilds under prod, signed by
// the key of testOnlySeed, with its networks the production ones.
var productionProof = testProofOptions{seed: testOnlySeed, stellar: "public", bitcoin: "mainnet"}

// truestampKeyring is a production keyring holding the key of seed, as
// www.truestamp.com would serve it.
func truestampKeyring(seed [32]byte) []byte {
	return []byte(fmt.Sprintf(`{"keys":[{"active":true,"key_id":%q,"public_key":%q,"sequence":0}],"version":"1.0"}`, keyIDOf(seed), publicKeyOf(seed)))
}

// truestampStub is a stub serving the keyring of testOnlySeed and the
// chains of productionProof, and a binary whose keyring read and NIST base
// go to it.
func truestampStub(t *testing.T) (*chainStub, string) {
	t.Helper()
	stub := newChainStub(t)
	memo, err := hex.DecodeString(testEpochHash(t, productionProof))
	if err != nil {
		t.Fatal(err)
	}
	stub.set(func(s *chainStub) { s.memo, s.keyring = memo, truestampKeyring(testOnlySeed) })
	bin := buildVerifyBinary(t, verifierEndpoints{keyring: stub.URL() + "/.well-known/keyring.json", nist: stub.URL() + "/nist"}, "")
	return stub, bin
}

// --- the online endings: verified and internally-consistent ---

// A production proof signed by a key of the keyring read from Truestamp
// (the stub, by the binary's link-time address), whose Stellar commitment
// and genesis.bitcoin the operator's Horizon and Esplora servers confirm,
// is verified, exit 0: attribution truestamp, trust_root current_keyring
// at that address with the digest of the bytes read, KEY_EVENT_COMMITMENT
// pass, and the submitted-before edge the ledger's close time, read from
// the Horizon server in this run, which the text names as "read from
// <source> in this run" (SPEC.md section 11.5); --silent carries the 0.
// With both sources unreachable, the same proof is internally-consistent,
// exit 7. Mutants killed: Options.Config leaving ReadChain nil (every
// CHAIN row is skip source_unreachable, and the run is
// internally-consistent); Configure reading CurrentKeyringURL rather than
// currentKeyringRead (the guard refuses www.truestamp.com, and the run is
// failed keyring_unavailable); testProof leaving the genesis block's
// previous_block_hash as carried (DERIVE fails under mainnet).
func TestVerify_VerifiedAndInternallyConsistent(t *testing.T) {
	stub, bin := truestampStub(t)
	path := writeTestProof(t, productionProof)
	sources := []string{"--horizon", stub.URL() + "/horizon", "--esplora", stub.URL() + "/esplora"}

	stdout, stderr, code := verifyCLI(t, bin, nil, nil, append([]string{path, "--json"}, sources...)...)
	report := parseReport(t, stdout)
	digest := sha256.Sum256(truestampKeyring(testOnlySeed))
	if code != 0 || report.Label != "verified" || !report.Verified || report.Window != "established" || report.Attribution != "truestamp" {
		t.Fatalf("exit %d, %+v\n%s", code, report, stderr)
	}
	if report.TrustRoot["kind"] != "current_keyring" || report.TrustRoot["location"] != stub.URL()+"/.well-known/keyring.json" || report.TrustRoot["digest"] != hex.EncodeToString(digest[:]) {
		t.Errorf("trust_root %v", report.TrustRoot)
	}
	if got := rowsWith(report, "CHAIN"); !slices.Contains(got, "pass CHAIN /signing_key_event") {
		t.Errorf("CHAIN %q", got)
	}
	if before := report.SubmittedBefore; before["instant"] != stubClosedAt || before["basis"] != "stellar_ledger_close_read" || before["source"] != stub.URL() || before["cached"] != false {
		t.Errorf("submitted_before %v", before)
	}

	text, _, code := verifyCLI(t, bin, nil, nil, append([]string{path}, sources...)...)
	for _, want := range []string{
		"Verdict:      verified\n",
		"Submitted before: " + stubClosedAt + " (stellar_ledger_close_read, read from " + stub.URL() + " in this run)\n",
		"Trust root:   current_keyring, from " + stub.URL() + "/.well-known/keyring.json, SHA-256 " + hex.EncodeToString(digest[:]),
	} {
		if code != 0 || !strings.Contains(text, want) {
			t.Errorf("exit %d; the text lacks %q\n%s", code, want, text)
		}
	}
	if stdout, stderr, code := verifyCLI(t, bin, nil, nil, append([]string{path, "--silent"}, sources...)...); code != 0 || stdout+stderr != "" {
		t.Errorf("--silent: exit %d, %q", code, stdout+stderr)
	}

	dead := []string{"--horizon", "http://127.0.0.1:9", "--esplora", "http://127.0.0.1:9"}
	stdout, _, code = verifyCLI(t, bin, nil, nil, append([]string{path, "--json"}, dead...)...)
	if report := parseReport(t, stdout); code != 7 || report.Label != "internally-consistent" || report.Window != "not_established" {
		t.Errorf("sources unreachable: exit %d, %+v", code, report)
	}
}

// An online run under staging, which pins Bitcoin's regtest network for
// commitments and its mainnet for entropy, reads each from one of its two
// floating Bitcoin sources, a stub whose Esplora server gives regtest's
// block at height 0 and whose Bitcoin Core node gives mainnet's: the
// CONFIGURATION row lists the Esplora server pinned to regtest and the
// node to mainnet, and no built-in source, which verifyCLI also checks.
// Mutant killed: the stub's node giving the Esplora server's block for
// height 0 (both sources show regtest, the node serves nothing, and the
// run reads mainnet's built-in Esplora server off loopback, which
// verifyCLI and the guard refuse).
func TestVerify_AStagingRunReadsBothBitcoinNetworksFromItsSources(t *testing.T) {
	stub := newChainStub(t)
	testIdentities(stub)
	stub.set(func(s *chainStub) { s.coreGenesis = mainnetGenesis })
	path := writeTestProof(t, testProofOptions{seed: vectorSeed, environment: "staging", stellar: "testnet", bitcoin: "regtest"})
	stdout, stderr, code := verifyCLI(t, binaryPath, nil, nil, path, "--json", "--environment", "staging", "--no-keyring",
		"--horizon", stub.URL()+"/horizon", "--esplora", stub.URL()+"/esplora", "--bitcoin-core", stub.URL()+"/core")
	report := parseReport(t, stdout)
	pinned := map[string]string{}
	for _, row := range report.Rows {
		if row.Step != "CONFIGURATION" {
			continue
		}
		sources, _ := row.Values["sources"].([]any)
		for _, s := range sources {
			record, _ := s.(map[string]any)
			pins, _ := record["pins"].([]any)
			var networks []string
			for _, pin := range pins {
				networks = append(networks, pin.(map[string]any)["network"].(string))
			}
			pinned[record["provider"].(string)] = strings.Join(slices.Compact(networks), " ")
		}
	}
	if code == 1 || code == 8 || pinned["esplora"] != "regtest" || pinned["bitcoin_core"] != "mainnet" {
		t.Errorf("exit %d, the sources' networks %v\n%s", code, pinned, stderr)
	}
}

// --- faults (SPEC.md sections 10.5 and 12.2) ---

// In a binary built with the tag truestamp_fault, the linker flag that sets
// the verifier's faultStep raises a fault before the step it names: one
// before steps 0 to 3 have
// passed, or at VERDICT, gives no report and exits 2, with nothing on
// stdout and the reason on stderr (and nothing at all under --silent);
// one from ENVIRONMENT to EXPECTED_HASH is an internal_error row at that
// step, part null, then skip prior_failure for each later step up to
// EXPECTED_HASH, label failed, exit 3, and at ENVIRONMENT the attribution
// none. Mutants killed: admit without its recover (the binary panics, and
// the guard of Execute exits 2 with a panic on stderr, not the sentence);
// runStep without its recover (exit 2 at SHAPE).
func TestVerify_Faults(t *testing.T) {
	for _, c := range []struct {
		step        string
		code        int
		attribution string
		after       []string
	}{
		{"CONFIGURATION", 2, "", nil},
		{"READ", 2, "", nil},
		{"ENVELOPE", 2, "", nil},
		{"ENVIRONMENT", 3, "none", []string{"skip KEY_BINDING prior_failure", "skip SIGNATURE prior_failure", "skip SHAPE prior_failure"}},
		{"SHAPE", 3, "keyring:", []string{"skip POLICY prior_failure", "skip DERIVE prior_failure", "skip CHAIN prior_failure"}},
		{"EXPECTED_HASH", 3, "keyring:", nil},
		{"VERDICT", 2, "", nil},
	} {
		t.Run(c.step, func(t *testing.T) {
			bin := buildVerifyBinary(t, deadEndpoints, "truestamp_fault", "-X github.com/truestamp/truestamp-cli/internal/verifier.faultStep="+c.step)
			stdout, stderr, code := verifyCLI(t, bin, nil, nil, append([]string{soundItem(), "--json"}, underTest()...)...)
			if code != c.code {
				t.Fatalf("exit %d, want %d\n%s\n%s", code, c.code, stdout, stderr)
			}
			if c.code == 2 {
				if stdout != "" || !strings.Contains(stderr, "The verifier stopped on a fault") || strings.Contains(stderr, "goroutine") {
					t.Errorf("stdout %q, stderr %q", stdout, stderr)
				}
				if stdout, stderr, code := verifyCLI(t, bin, nil, nil, append([]string{soundItem(), "--silent"}, underTest()...)...); code != 2 || stdout+stderr != "" {
					t.Errorf("--silent: exit %d, %q", code, stdout+stderr)
				}
				return
			}
			report := parseReport(t, stdout)
			at := slices.IndexFunc(report.Rows, func(r verifyRow) bool { return r.Status == "fail" })
			if report.Label != "failed" || *report.Code != "internal_error" || at < 0 || report.Rows[at].String() != "fail "+c.step+" internal_error" ||
				!strings.HasPrefix(report.Attribution, c.attribution) || report.Rows[len(report.Rows)-1].Step != "EXPECTED_HASH" {
				t.Fatalf("%+v", report)
			}
			for i, want := range c.after {
				if got := report.Rows[at+1+i].String(); got != want {
					t.Errorf("row %d after the fault: %q, want %q", i+1, got, want)
				}
			}
		})
	}
}

// --- the verifier's compiled-in addresses ---

// The binary every test runs is built with both compiled-in addresses at
// deadEndpoints and the fixed BuildYear of testBuildYearFlag, as its build
// information records (testBinaryLDFlags). An online run of it
// under prod reads its keyring there: keyring_unavailable, with
// trust_root's location the dead address. An online run of the vectors'
// sound item proof, which carries a NIST pulse, under E_v test with the
// vectors' keyring and dead Stellar and Bitcoin sources, reads NIST there:
// its NIST SOURCES row is skip source_unreachable, from the dead address
// as a report names a source (scheme://host:port). Neither makes a
// request off loopback. Mutants killed: runTests building binaryPath
// without the linker flags (the build information lacks them, and the
// read goes to www.truestamp.com, which the guard refuses and records);
// nistBeaconBase initialized by a call, `fmt.Sprint(NISTBeaconURL)`, which
// the linker cannot set (the NIST row's source is
// https://beacon.nist.gov); runTests building binaryPath without
// testBuildYearFlag (the build information lacks it).
func TestVerify_TheTestBinaryIsRedirected(t *testing.T) {
	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	var ldflags string
	for _, s := range info.Settings {
		if s.Key == "-ldflags" {
			ldflags = s.Value
		}
	}
	if ldflags != testBinaryLDFlags() {
		t.Errorf("the test binary's -ldflags are %q", ldflags)
	}
	path := writeTestProof(t, productionProof)
	stdout, _, code := verifyCLI(t, binaryPath, nil, nil, path, "--json", "--horizon", "http://127.0.0.1:9", "--esplora", "http://127.0.0.1:9")
	report := parseReport(t, stdout)
	if code != 3 || *report.Code != "keyring_unavailable" || report.TrustRoot["location"] != deadEndpoints.keyring {
		t.Errorf("exit %d, %+v", code, report)
	}

	nist, err := url.Parse(deadEndpoints.nist)
	if err != nil {
		t.Fatal(err)
	}
	stdout, _, code = verifyCLI(t, binaryPath, nil, nil, soundItem(), "--json", "--environment", "test", "--keyring", testKeyring(),
		"--horizon", "http://127.0.0.1:9", "--esplora", "http://127.0.0.1:9")
	report = parseReport(t, stdout)
	var nistRow *verifyRow
	for i, row := range report.Rows {
		if row.Step == "SOURCES" && row.Part != nil && *row.Part == "/witnesses/entropy_nist" {
			nistRow = &report.Rows[i]
		}
	}
	if code != 6 || nistRow == nil || nistRow.String() != "skip SOURCES source_unreachable /witnesses/entropy_nist" {
		t.Fatalf("exit %d; the NIST row is %v", code, nistRow)
	}
	outcomes, _ := nistRow.Values["outcomes"].([]any)
	if len(outcomes) != 1 || outcomes[0].(map[string]any)["source"] != nist.Scheme+"://"+nist.Host {
		t.Errorf("NIST was read from %v, want %s://%s", outcomes, nist.Scheme, nist.Host)
	}
}

// No shipped build sets either compiled-in address: neither the Taskfile's
// build targets nor the release configuration passes a linker flag for
// internal/verifier, so a released binary reads Truestamp's keyring at
// www.truestamp.com and NIST at https://beacon.nist.gov/beacon/2.0, the
// values internal/verifier's TestEndpoints_ShippedValues holds. Mutant
// killed: the Taskfile's LDFLAGS gaining an -X for internal/verifier.
func TestShippedBuilds_SetNoVerifierAddress(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"Taskfile.yml", ".goreleaser.yaml"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(data, []byte("internal/version.Version")) {
			t.Fatalf("%s sets no linker flag at all, so this test shows nothing", name)
		}
		if bytes.Contains(data, []byte("internal/verifier")) {
			t.Errorf("%s names internal/verifier", name)
		}
	}
}

// --- every run of verify keeps to loopback ---

// No test of this package runs `truestamp verify`, or the test binary, but
// through verifyCLI, which keeps an online run on loopback. The scan reads
// every test file of the package, but verifyCLI and verifyCLITo
// themselves:
//   - a call that passes the literal "verify" as any of its arguments is
//     verifyCLI's, verifyCLITo's, helpText's, or a help run (it passes
//     "--help");
//   - a list that holds the literal "verify" in any position names
//     --offline or --help, or is a `help` command's;
//   - an exec.Command or exec.CommandContext of the test binary (binaryPath,
//     or a binary of buildVerifyBinary or truestampStub) whose arguments,
//     each local variable read as the string or the list the function
//     assigns it, hold "verify" is a help run: an offline run goes through
//     verifyCLI too. A list written elsewhere (a table's, a range's, the
//     function's own variadic arguments) is held to the first two rules
//     where it is written. Any other exec.Command is held to the first
//     rule.
//
// Mutants killed: a test running exec.Command(binaryPath, "verify", path);
// one running `args := []string{"--json=true", "verify", path, ...}` and
// exec.Command(binaryPath, args...); one running `bin := binaryPath` and
// exec.Command(bin, "verify", path); one running
// exec.Command(filepath.Join(dir, "truestamp"), "verify", path) (the scan
// names its file and line).
func TestVerifyRuns_GoThroughVerifyCLI(t *testing.T) {
	const verifyCommand, helpFlag = "verify", "--help"
	files, err := filepath.Glob(filepath.Join(repoRoot(t), "cmd", "*_test.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no test files: %v", err)
	}
	fset := token.NewFileSet()
	isLit := func(e ast.Expr, want ...string) bool {
		lit, ok := e.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return false
		}
		value, err := strconv.Unquote(lit.Value)
		return err == nil && slices.Contains(want, value)
	}
	holds := func(list []ast.Expr, want ...string) bool {
		return slices.ContainsFunc(list, func(e ast.Expr) bool { return isLit(e, want...) })
	}
	isExec := func(call *ast.CallExpr) ([]ast.Expr, bool) {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !isIdent(sel.X, "exec") || (sel.Sel.Name != "Command" && sel.Sel.Name != "CommandContext") {
			return nil, false
		}
		if sel.Sel.Name == "CommandContext" && len(call.Args) > 0 {
			return call.Args[1:], true
		}
		return call.Args, true
	}
	runs, binaryRuns := 0, 0
	for _, file := range files {
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Name.Name == "verifyCLI" || fn.Name.Name == "verifyCLITo" {
				continue
			}
			// binaries and assigned are the function's binaries of
			// buildVerifyBinary or truestampStub, and the string or list
			// it assigns each local variable.
			binaries := map[string]bool{}
			assigned := map[string][]ast.Expr{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				assign, ok := n.(*ast.AssignStmt)
				if !ok || len(assign.Rhs) != 1 {
					return true
				}
				switch rhs := assign.Rhs[0].(type) {
				case *ast.CallExpr:
					if callee, ok := rhs.Fun.(*ast.Ident); ok && (callee.Name == "buildVerifyBinary" || callee.Name == "truestampStub") {
						for _, lhs := range assign.Lhs {
							if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" {
								binaries[id.Name] = true
							}
						}
					}
				case *ast.CompositeLit:
					if id, ok := assign.Lhs[0].(*ast.Ident); ok && len(assign.Lhs) == 1 {
						assigned[id.Name] = append(assigned[id.Name], rhs.Elts...)
					}
				case *ast.BasicLit:
					if id, ok := assign.Lhs[0].(*ast.Ident); ok && len(assign.Lhs) == 1 {
						assigned[id.Name] = append(assigned[id.Name], rhs)
					}
				}
				return true
			})
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CallExpr:
					if args, ok := isExec(n); ok && len(args) > 0 && (isIdent(args[0], "binaryPath") || binaries[identName(args[0])]) {
						binaryRuns++
						var read []ast.Expr
						for _, arg := range args[1:] {
							read = append(read, arg)
							read = append(read, assigned[identName(arg)]...)
						}
						if holds(read, verifyCommand) && !holds(read, helpFlag) {
							t.Errorf("%s: runs verify through the test binary but not through verifyCLI", fset.Position(n.Pos()))
						}
						return true
					}
					if !holds(n.Args, verifyCommand) {
						return true
					}
					runs++
					if callee, ok := n.Fun.(*ast.Ident); ok && (callee.Name == "verifyCLI" || callee.Name == "verifyCLITo" || callee.Name == "helpText") {
						return true
					}
					if holds(n.Args, helpFlag) {
						return true
					}
					t.Errorf("%s: runs verify but not through verifyCLI", fset.Position(n.Pos()))
				case *ast.CompositeLit:
					if holds(n.Elts, verifyCommand) && !holds(n.Elts, "--offline", helpFlag) && !(len(n.Elts) > 0 && isLit(n.Elts[0], "help")) {
						t.Errorf("%s: a run of verify that is neither offline nor help", fset.Position(n.Pos()))
					}
				}
				return true
			})
		}
	}
	if runs == 0 || binaryRuns == 0 {
		t.Fatalf("the scan found %d runs of verify and %d of the test binary, so it shows nothing", runs, binaryRuns)
	}
}

// identName is the name of e when it is an identifier, and "" otherwise.
func identName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// isIdent reports whether e is the identifier name.
func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}
