// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The verifier's sources and keyrings, through a real `truestamp verify`
// process: where each option may come from (SPEC.md section 9 (What a
// verifier pins); sections 6.4 and 8.3 for the run-only options), where a
// Bitcoin Core node's credentials go, and that nothing the CLI prints
// carries them. Every online run reads only the loopback stub of
// verify_online_test.go, or a dead loopback port.

// The planted credentials of these tests: stand-in values, never a real
// credential. The password holds %, /, #, ?, @ and :, which a URL would
// have to escape.
const (
	plantedUser     = "planted-user-3k"
	plantedPassword = "planted-pass-9z/#?@:%"
)

// noSecret fails the test when any of outputs carries a planted
// credential.
func noSecret(t *testing.T, what string, outputs ...string) {
	t.Helper()
	for _, out := range outputs {
		for _, secret := range []string{plantedUser, plantedPassword, "planted-pass-9z"} {
			if strings.Contains(out, secret) {
				t.Errorf("%s carries %q:\n%s", what, secret, out)
			}
		}
	}
}

// writeConfig writes text to a configuration file of the test's.
func writeConfig(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// testIdentities makes stub give the identities of the vectors' test
// networks, Stellar testnet and Bitcoin regtest.
func testIdentities(stub *chainStub) {
	stub.set(func(s *chainStub) { s.passphrase, s.genesis = testnetStellar, regtestGenesis })
}

// readBoth reports whether the stub saw an identity read of each chain:
// of Horizon's root (which a Stellar read that finds nothing reads again)
// and of Esplora's block at height 0.
func readBoth(stub *chainStub) bool {
	requests, _ := stub.seen()
	return slices.Contains(requests, "GET /horizon/") && slices.Contains(requests, "GET /esplora/block-height/0")
}

// verify --help names the three source flags, and no flag takes a Bitcoin
// Core node's credentials: --bitcoin-core-user and --bitcoin-core-password
// are unknown flags (exit 1), and so are a NIST source and a network pin.
// Mutant killed: RegisterFlags registering --bitcoin-core-password.
func TestCLI_Verify_SourceFlagsAndNoCredentialFlag(t *testing.T) {
	help := helpText(t, "verify")
	for _, want := range []string{"--horizon string", "--esplora string", "--bitcoin-core string", "--environment string", "--no-keyring", "--keyring string", "--trace string"} {
		if !strings.Contains(help, want) {
			t.Errorf("verify --help lacks %q\n%s", want, help)
		}
	}
	for _, flag := range []string{"--bitcoin-core-user", "--bitcoin-core-password", "--nist", "--network", "--commitment-pin"} {
		_, stderr, code := verifyCLI(t, binaryPath, nil, nil, append([]string{soundItem(), flag, "x"}, underTest()...)...)
		if code != 1 || !strings.Contains(stderr, "unknown flag: "+flag) {
			t.Errorf("%s: exit %d\n%s", flag, code, stderr)
		}
	}
}

// The TRUESTAMP_VERIFY_* variables and the configuration keys of the
// sources move an online run's identity reads to the servers they name:
// with each, the stub sees a read of Horizon's root and one of Esplora's
// block at height 0. The legacy TRUESTAMP_HORIZON_URL,
// TRUESTAMP_BLOCKSTREAM_URL and TRUESTAMP_NIST_URL move nothing: with them
// naming the stub and the operator's sources dead, the stub sees no
// request. Mutant killed: OptionsFromCommand ignoring cfg.Verify.Horizon
// (no Horizon identity read).
func TestCLI_Verify_SourceVariablesAndKeysMoveTheIdentityReads(t *testing.T) {
	online := []string{soundItem(), "--environment", "test", "--keyring", testKeyring(), "--json"}

	stub := newChainStub(t)
	testIdentities(stub)
	env := []string{"TRUESTAMP_VERIFY_HORIZON=" + stub.URL() + "/horizon", "TRUESTAMP_VERIFY_ESPLORA=" + stub.URL() + "/esplora"}
	if _, stderr, code := verifyCLI(t, binaryPath, env, nil, online...); code == 1 || code == 8 || !readBoth(stub) {
		t.Errorf("the variables: exit %d\n%s", code, stderr)
	}

	keys := newChainStub(t)
	testIdentities(keys)
	path := writeConfig(t, "[verify]\nhorizon = \""+keys.URL()+"/horizon\"\nesplora = \""+keys.URL()+"/esplora\"\n")
	if _, stderr, code := verifyCLI(t, binaryPath, nil, nil, append(online, "--config", path)...); code == 1 || code == 8 || !readBoth(keys) {
		t.Errorf("the configuration keys: exit %d\n%s", code, stderr)
	}

	legacy := newChainStub(t)
	variables := []string{"TRUESTAMP_HORIZON_URL=" + legacy.URL(), "TRUESTAMP_BLOCKSTREAM_URL=" + legacy.URL(), "TRUESTAMP_NIST_URL=" + legacy.URL()}
	dead := []string{"--horizon", "http://127.0.0.1:9", "--esplora", "http://127.0.0.1:9"}
	if _, _, code := verifyCLI(t, binaryPath, variables, nil, append(online, dead...)...); code != 6 {
		t.Errorf("the legacy variables: exit %d, want 6", code)
	}
	if requests, _ := legacy.seen(); len(requests) != 0 {
		t.Errorf("a legacy variable moved a read: %q", requests)
	}
}

// No flag, configuration key or environment variable sets E_v, keyless
// mode or a network pin (SPEC.md sections 6.4, 8.3 and 9): with spellings
// of each set to the test environment, keyless mode and the production
// networks, a run is still under prod and keyed (the test keyring, which
// holds VECTOR_KEY, is refused under prod: exit 8; and with no keyring
// the test-marked proof fails environment_mismatch: exit 3, not 5), and
// under --environment test its networks are still test's (exit 6: POLICY
// passes). Mutant killed: OptionsFromCommand taking E_v from
// TRUESTAMP_VERIFY_ENVIRONMENT (the first run is supplied-keyring).
func TestCLI_Verify_NoOptionSetsTheEnvironmentKeylessOrAPin(t *testing.T) {
	env := []string{
		"TRUESTAMP_ENVIRONMENT=test", "TRUESTAMP_VERIFY_ENVIRONMENT=test",
		"TRUESTAMP_NO_KEYRING=true", "TRUESTAMP_VERIFY_NO_KEYRING=true",
		"TRUESTAMP_VERIFY_NETWORKS_BITCOIN=mainnet", "TRUESTAMP_VERIFY_COMMITMENT_BITCOIN=mainnet",
		"TRUESTAMP_VERIFY_PINS_COMMITMENT_STELLAR=public",
	}
	path := writeConfig(t, "environment = \"test\"\nno_keyring = true\n\n[verify]\nenvironment = \"test\"\nno_keyring = true\n\n"+
		"[verify.pins.commitment]\nbitcoin = \"mainnet\"\nstellar = \"public\"\n")
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{soundItem(), "--offline", "--keyring", testKeyring(), "--config", path}, 8},
		{[]string{soundItem(), "--offline", "--config", path}, 3},
		{append([]string{soundItem(), "--config", path}, underTest()...), 6},
	} {
		if _, stderr, code := verifyCLI(t, binaryPath, env, nil, c.args...); code != c.code {
			t.Errorf("%q: exit %d, want %d\n%s", c.args, code, c.code, stderr)
		}
	}
}

// A --bitcoin-core URL that carries user information is refused before any
// read, exit 1, from a flag, a variable or the configuration file, and
// neither stdout nor stderr carries the planted user or password. Mutant
// killed: the refusal quoting the URL.
func TestCLI_Verify_ACredentialInASourceURLIsRefused(t *testing.T) {
	withUser := "http://" + plantedUser + ":planted-pass-9z@127.0.0.1:9"
	sources := []string{"--horizon", "http://127.0.0.1:9", "--esplora", "http://127.0.0.1:9", "--environment", "test", "--keyring", testKeyring()}
	for name, run := range map[string]struct {
		env  []string
		args []string
	}{
		"a flag":     {nil, append([]string{soundItem(), "--bitcoin-core", withUser}, sources...)},
		"a variable": {[]string{"TRUESTAMP_VERIFY_BITCOIN_CORE=" + withUser}, append([]string{soundItem()}, sources...)},
		"the file":   {nil, append([]string{soundItem(), "--config", writeConfig(t, "[verify]\nbitcoin_core = \""+withUser+"\"\n")}, sources...)},
	} {
		stdout, stderr, code := verifyCLI(t, binaryPath, run.env, nil, run.args...)
		if code != 1 || !strings.Contains(stderr, "the URL carries user information") {
			t.Errorf("%s: exit %d\n%s", name, code, stderr)
		}
		noSecret(t, name, stdout, stderr)
	}
}

// A Bitcoin Core node's credentials, from the environment, reach the node
// as Basic authentication on every call, and nothing the CLI writes
// carries them: not the --json report, the text, stderr, the --trace file,
// nor config show as text or --json. Mutants killed: the credentials left
// off the node's calls (no Authorization header); BitcoinCoreUser tagged
// json:"bitcoin_core_user" (config show --json carries the user).
func TestCLI_Verify_CoreCredentialsReachOnlyTheNode(t *testing.T) {
	stub, bin := truestampStub(t)
	path := writeTestProof(t, productionProof)
	env := []string{"TRUESTAMP_VERIFY_BITCOIN_CORE_USER=" + plantedUser, "TRUESTAMP_VERIFY_BITCOIN_CORE_PASSWORD=" + plantedPassword}
	trace := filepath.Join(t.TempDir(), "trace.json")
	args := []string{path, "--horizon", stub.URL() + "/horizon", "--bitcoin-core", stub.URL() + "/core", "--trace", trace}

	stdout, stderr, code := verifyCLI(t, bin, env, nil, append(args, "--json")...)
	if report := parseReport(t, stdout); code != 0 || report.Label != "verified" {
		t.Fatalf("exit %d, %s\n%s", code, report.Label, stderr)
	}
	text, textErr, _ := verifyCLI(t, bin, env, nil, args...)
	traced, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	noSecret(t, "the run", stdout, stderr, text, textErr, string(traced))

	_, authorizations := stub.seen()
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte(plantedUser+":"+plantedPassword))
	if len(authorizations) < 4 || slices.ContainsFunc(authorizations, func(a string) bool { return a != want }) {
		t.Errorf("the node's calls carried %q, want each %q", authorizations, want)
	}

	for _, show := range [][]string{{"config", "show"}, {"config", "show", "--json"}} {
		cmd := exec.Command(binaryPath, show...)
		cmd.Env = append(cleanEnv(), append(env, "TRUESTAMP_VERIFY_BITCOIN_CORE="+stub.URL()+"/core")...)
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), stub.URL()+"/core") {
			t.Errorf("%q: %v\n%s", show, err, out)
		}
		noSecret(t, strings.Join(show, " "), string(out))
	}
}

// config show, as text and as --json, shows the verifier's sources, each
// network's several sources with its policy, and the configured keyrings,
// and never a Bitcoin Core node's user or password, planted in the file,
// nor user information in a source URL, a source key's or a
// [verify.sources] table's, which it shows masked, a password typed with a
// raw slash (a URL that parses with a host and a path) among them.
// Mutants killed: the text table leaving the sources out; maskUserinfo
// returning the URL whole; newConfigRecord copying a table's URLs without
// maskUserinfo (--json carries the table's credentials); sourceRows
// listing a table's URLs without it (the text does); maskUserinfo showing
// whole a URL that parses with no user information (the slashed password
// shows).
func TestCLI_ConfigShow_TheVerifierSources(t *testing.T) {
	path := writeConfig(t, `[verify]
horizon = "https://horizon.config.example"
esplora = "https://esplora.config.example/api"
bitcoin_core = "http://`+plantedUser+`:planted-pass-9z@core.config.example:8332"
bitcoin_core_user = "`+plantedUser+`"
bitcoin_core_password = "planted-pass-9z"

[verify.keyrings]
staging = "/keyrings/staging.json"

[verify.sources.bitcoin.mainnet]
policy = "at_least 2"
sources = [{ esplora = "https://one.config.example" }, { esplora = "https://two.config.example" }]

[verify.sources.bitcoin.regtest]
policy = "all"
sources = [{ bitcoin_core = "http://`+plantedUser+`:planted-pass-9z@core.sources.example:8332" }, { bitcoin_core = "http://u:12/ss-planted@core.slash.example:8332" }]
`)
	for _, show := range [][]string{{"config", "show", "--config", path}, {"config", "show", "--config", path, "--json"}} {
		out, code := runCLIText(t, show...)
		for _, want := range []string{
			"https://horizon.config.example", "https://esplora.config.example/api", "[masked]@core.config.example:8332",
			"/keyrings/staging.json", "at_least 2", "https://one.config.example", "https://two.config.example",
			"http://[masked]@core.sources.example:8332", "http://[masked]@core.slash.example:8332",
		} {
			if code != 0 || !strings.Contains(out, want) {
				t.Errorf("%q: exit %d, lacks %q\n%s", show, code, want, out)
			}
		}
		if strings.Contains(out, "12/ss-planted") {
			t.Errorf("%q: shows the slashed password\n%s", show, out)
		}
		noSecret(t, strings.Join(show, " "), out)
	}
	out, _ := runCLIText(t, "config", "show", "--config", path, "--json")
	var record struct {
		Verify map[string]json.RawMessage `json:"verify"`
	}
	if err := json.Unmarshal([]byte(out), &record); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"horizon", "esplora", "bitcoin_core", "keyrings", "sources"} {
		if _, ok := record.Verify[key]; !ok {
			t.Errorf("config show --json has no verify.%s", key)
		}
	}
	for _, key := range []string{"keyring", "bitcoin_core_user", "bitcoin_core_password"} {
		if _, ok := record.Verify[key]; ok {
			t.Errorf("config show --json has verify.%s", key)
		}
	}
}

// A source whose identity read gives a value its pins forbid is
// configuration_error, exit 8: under E_v test, which pins Bitcoin to
// regtest alone, an Esplora server that gives mainnet's block at height 0.
// An unreachable one is not: the rows it serves skip, and the sound proof
// is supplied-keyring, exit 6. Mutant killed: the command exiting 3 for a
// configuration_error.
func TestCLI_Verify_AMismatchedSourceIsConfigurationError(t *testing.T) {
	stub := newChainStub(t)
	stub.set(func(s *chainStub) { s.passphrase, s.genesis = testnetStellar, mainnetGenesis })
	args := []string{soundItem(), "--json", "--horizon", stub.URL() + "/horizon", "--esplora", stub.URL() + "/esplora", "--environment", "test", "--keyring", testKeyring()}
	stdout, _, code := verifyCLI(t, binaryPath, nil, nil, args...)
	var rejection map[string]string
	if err := json.Unmarshal([]byte(stdout), &rejection); err != nil || code != 8 || rejection["code"] != "configuration_error" {
		t.Errorf("a mismatched source: exit %d, %v, %s", code, err, stdout)
	}
	_, _, code = verifyCLI(t, binaryPath, nil, nil, soundItem(), "--horizon", "http://127.0.0.1:9", "--esplora", "http://127.0.0.1:9", "--environment", "test", "--keyring", testKeyring())
	if code != 6 {
		t.Errorf("an unreachable source: exit %d, want 6", code)
	}
}

// verify.keyring and TRUESTAMP_VERIFY_KEYRING are retired: neither gives
// a run its keyring, so a run under E_v test with only them has no keyring
// source (configuration_error, exit 8). A keyring configured for an
// environment under [verify.keyrings] is used when --environment names
// that environment: the sound proof is supplied-keyring, exit 6, with
// that file as trust_root's location; under prod it is not used, and the
// compiled pin governs (exit 3). Mutant killed: Options.Config taking a
// configured keyring under prod (the test keyring, which holds VECTOR_KEY,
// is refused there: exit 8).
func TestCLI_Verify_TheConfiguredKeyrings(t *testing.T) {
	retired := writeConfig(t, "[verify]\nkeyring = \""+testKeyring()+"\"\n")
	_, stderr, code := verifyCLI(t, binaryPath, []string{"TRUESTAMP_VERIFY_KEYRING=" + testKeyring()}, nil, soundItem(), "--offline", "--environment", "test", "--config", retired)
	if code != 8 {
		t.Errorf("the retired key and variable: exit %d, want 8\n%s", code, stderr)
	}
	configured := writeConfig(t, "[verify.keyrings]\ntest = \""+testKeyring()+"\"\n")
	stdout, _, code := verifyCLI(t, binaryPath, nil, nil, soundItem(), "--offline", "--environment", "test", "--config", configured, "--json")
	if report := parseReport(t, stdout); code != 6 || report.TrustRoot["location"] != testKeyring() {
		t.Errorf("a configured keyring: exit %d, %+v", code, report.TrustRoot)
	}
	_, _, code = verifyCLI(t, binaryPath, nil, nil, soundItem(), "--offline", "--config", configured)
	if code != 3 {
		t.Errorf("a configured keyring under prod: exit %d, want 3 (the compiled pin governs)", code)
	}
}

// A network's several sources come from the configuration file, in their
// order, under their policy (SPEC.md section 9): bitcoin's regtest under
// at_least 3 with two sources is configuration_error, exit 8, before any
// read; under all, both are read. Mutant killed: OptionsFromCommand leaving
// the networks out (the at_least 3 run reads the operator's Horizon and
// is configuration_error only for regtest's missing source, which reads
// the stub first).
func TestCLI_Verify_SeveralSourcesForANetwork(t *testing.T) {
	stub := newChainStub(t)
	testIdentities(stub)
	table := func(policy string) string {
		return writeConfig(t, "[verify.sources.bitcoin.regtest]\npolicy = \""+policy+"\"\nsources = [{ esplora = \""+stub.URL()+"/esplora\" }, { bitcoin_core = \""+stub.URL()+"/core\" }]\n")
	}
	base := []string{soundItem(), "--json", "--horizon", stub.URL() + "/horizon", "--environment", "test", "--keyring", testKeyring()}
	stdout, _, code := verifyCLI(t, binaryPath, nil, nil, append(base, "--config", table("at_least 3"))...)
	var rejection map[string]string
	if err := json.Unmarshal([]byte(stdout), &rejection); err != nil || code != 8 || !strings.Contains(rejection["message"], "at_least") {
		t.Errorf("at_least 3 of two: exit %d, %s", code, stdout)
	}
	if requests, _ := stub.seen(); len(requests) != 0 {
		t.Errorf("a refused policy read %q", requests)
	}
	_, stderr, code := verifyCLI(t, binaryPath, nil, nil, append(base, "--config", table("all"))...)
	if requests, _ := stub.seen(); code == 1 || code == 8 || !slices.Contains(requests, "GET /esplora/block-height/0") || !slices.Contains(requests, "POST /core") {
		t.Errorf("all: exit %d, requests %q\n%s", code, requests, stderr)
	}
}

// helpText returns a subcommand's --help output.
func helpText(t *testing.T, sub string) string {
	t.Helper()
	cmd := exec.Command(binaryPath, sub, "--help")
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s --help: %v\n%s", sub, err, out)
	}
	return string(out)
}
