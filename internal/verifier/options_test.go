// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// commandFlags is the flag set of `truestamp verify` as these options see
// it, the root's --base-url among them, parsed from args.
func commandFlags(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()
	flags := pflag.NewFlagSet("verify", pflag.ContinueOnError)
	flags.String("base-url", "", "")
	flags.String("keyring", "", "")
	flags.Bool("offline", false, "")
	RegisterFlags(flags)
	if err := flags.Parse(args); err != nil {
		t.Fatal(err)
	}
	return flags
}

// loadOptions resolves the configuration as the CLI does, from the TOML
// file text (none when empty), the environment the test set and the flags
// args gives, and returns it with the run's options.
func loadOptions(t *testing.T, toml string, args ...string) (*config.Config, Options) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	flags := commandFlags(t, args...)
	cfg, err := config.Load(path, flags)
	if err != nil {
		t.Fatal(err)
	}
	options, err := OptionsFromCommand(flags, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, options
}

// recorder is a fake keyring reader: it answers every read with data and
// records each URL it is asked for.
type recorder struct {
	data []byte
	err  error
	urls []string
}

func (r *recorder) read(url string) ([]byte, error) {
	r.urls = append(r.urls, url)
	return r.data, r.err
}

// clearTruestampEnv removes every TRUESTAMP_* variable for the test, so
// the developer's own settings (TRUESTAMP_VERIFY_OFFLINE,
// TRUESTAMP_BASE_URL, TRUESTAMP_HTTP_TIMEOUT and the rest) neither break
// nor steer it; t.Setenv restores each when the test ends. A test calls it
// first, before it sets any variable of its own, and never through
// loadOptions, which would erase the variables the test set.
func clearTruestampEnv(t *testing.T) {
	t.Helper()
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "TRUESTAMP_") {
			t.Setenv(name, "")
			_ = os.Unsetenv(name)
		}
	}
}

// configureOptions makes CONFIGURATION from options with a recorder that
// answers with a sound keyring.
func configureOptions(t *testing.T, options Options) (*Run, *Rejection, *recorder) {
	t.Helper()
	data, err := os.ReadFile(writeKeyring(t, testKey("online")))
	if err != nil {
		t.Fatal(err)
	}
	return configureOptionsWith(t, options, data)
}

// configureOptionsWith makes CONFIGURATION from options with a recorder
// that answers every read with data.
func configureOptionsWith(t *testing.T, options Options, data []byte) (*Run, *Rejection, *recorder) {
	t.Helper()
	reader := &recorder{data: data}
	cfg, err := options.Config(reader.read)
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	run, rejection := configure(t, cfg)
	return run, rejection, reader
}

// E_v and keyless mode come only from the run's options: the
// configuration file and environment variables that ask for an
// environment or for keyless mode, under any spelling, and a keyring
// config key or environment variable, select nothing (SPEC.md sections 6.4
// and 8.3). Mutants killed: OptionsFromCommand taking the keyring file
// from cfg.Verify.Keyring; keyless mode taken from
// TRUESTAMP_VERIFY_NO_KEYRING.
func TestOptions_EnvironmentAndKeylessComeOnlyFromTheRun(t *testing.T) {
	clearTruestampEnv(t)
	for _, name := range []string{"TRUESTAMP_ENVIRONMENT", "TRUESTAMP_VERIFY_ENVIRONMENT"} {
		t.Setenv(name, "staging")
	}
	for _, name := range []string{"TRUESTAMP_NO_KEYRING", "TRUESTAMP_VERIFY_NO_KEYRING", "TRUESTAMP_VERIFY_NO-KEYRING"} {
		t.Setenv(name, "true")
	}
	t.Setenv("TRUESTAMP_VERIFY_KEYRING", vectorcases.Path("keyring.test.json"))
	toml := `environment = "staging"
no_keyring = true
"no-keyring" = true

[verify]
environment = "staging"
no_keyring = true
"no-keyring" = true
keyring = "` + vectorcases.Path("keyring.test.json") + `"
`
	cfg, options := loadOptions(t, toml, "--offline")
	if cfg.Verify.Keyring == "" {
		t.Fatal("the keyring config key did not load, so this test shows nothing")
	}
	if options != (Options{Environment: "prod", Offline: true}) {
		t.Fatalf("options = %+v", options)
	}
	run, rejection, _ := configureOptions(t, options)
	if rejection != nil || run.Config.Environment != "prod" || run.TrustRoot.Kind != KindCompiledPin {
		t.Fatalf("CONFIGURATION: %v, %+v", rejection, run)
	}

	_, options = loadOptions(t, toml, "--offline", "--environment", "staging", "--no-keyring")
	run, rejection, _ = configureOptions(t, options)
	if rejection != nil || run.Config.Environment != "staging" || run.TrustRoot.Kind != KindNone {
		t.Fatalf("given for the run: %v, %+v", rejection, run)
	}
}

// The configured base URL, in the configuration file or
// TRUESTAMP_BASE_URL, points the API commands but never moves the keyring
// read or E_v: an online run under prod reads the keyring from
// www.truestamp.com, through the fake reader, and no request leaves the
// test. Mutant killed: OptionsFromCommand taking the keyring URL from the
// configured base URL as well as from an explicit --base-url.
func TestOptions_TheConfiguredBaseURLNeverMovesTheKeyringRead(t *testing.T) {
	clearTruestampEnv(t)
	before := refusals.Load()
	for name, setup := range map[string]func() string{
		"the configuration file": func() string { return `base_url = "https://staging.truestamp.example"` + "\n" },
		"TRUESTAMP_BASE_URL": func() string {
			t.Setenv("TRUESTAMP_BASE_URL", "https://staging.truestamp.example")
			return ""
		},
	} {
		cfg, options := loadOptions(t, setup())
		if cfg.KeyringURL != "https://staging.truestamp.example/.well-known/keyring.json" {
			t.Fatalf("%s: the configured base URL did not load: %s", name, cfg.KeyringURL)
		}
		run, rejection, reader := configureOptions(t, options)
		if rejection != nil || run.Config.Environment != "prod" || !run.Config.Online {
			t.Fatalf("%s: %v, %+v", name, rejection, run)
		}
		if len(reader.urls) != 1 || reader.urls[0] != CurrentKeyringURL || run.TrustRoot.Kind != KindCurrentKeyring || run.Keyring == nil {
			t.Fatalf("%s: read %q, trust root %+v", name, reader.urls, run.TrustRoot)
		}
	}
	if got := refusals.Load() - before; got != 0 {
		t.Fatalf("%d requests reached the shared client", got)
	}
}

// An explicit --base-url names the origin of the keyring read, as a
// supplied keyring loaded under the run's E_v, and a failed read from it
// binds no key and is never configuration_error; one whose host is
// www.truestamp.com, under any scheme or port and with or without a
// trailing dot, is the default read under prod and no keyring source under
// a non-production E_v, which must not use it; --keyring FILE comes before
// it; offline, no keyring is read from an origin. Mutants killed: an
// explicit --base-url naming www.truestamp.com read as a supplied keyring;
// an explicit --base-url read in an offline run; the origin compared by
// its raw scheme and Host (https://www.truestamp.com:443, the trailing dot
// and http are read as another origin); the origin's keyring loaded under
// a fixed "prod" (the staging origin's non-production key is refused); the
// origin's source put before --keyring FILE.
func TestOptions_AnExplicitBaseURLMovesTheKeyringRead(t *testing.T) {
	clearTruestampEnv(t)
	before := refusals.Load()
	_, options := loadOptions(t, "", "--base-url", "https://staging.truestamp.example/some/path")
	run, rejection, reader := configureOptions(t, options)
	origin := "https://staging.truestamp.example/.well-known/keyring.json"
	if rejection != nil || run.TrustRoot != (TrustRoot{Kind: KindSuppliedKeyring, Location: origin}) ||
		len(reader.urls) != 1 || reader.urls[0] != origin || run.Keyring == nil {
		t.Fatalf("explicit origin: %v, %+v, read %q", rejection, run.TrustRoot, reader.urls)
	}

	cfg, err := options.Config((&recorder{err: errors.New("HTTP 503")}).read)
	if err != nil {
		t.Fatal(err)
	}
	if run, rejection := configure(t, cfg); rejection != nil || !run.KeyringUnavailable || run.Keyring != nil {
		t.Fatalf("a failed read from the origin: %v, %+v", rejection, run)
	}

	_, options = loadOptions(t, "", "--base-url", "HTTPS://WWW.Truestamp.com")
	if run, _, reader := configureOptions(t, options); run.TrustRoot.Kind != KindCurrentKeyring || reader.urls[0] != CurrentKeyringURL {
		t.Fatalf("www.truestamp.com named: %+v, %q", run.TrustRoot, reader.urls)
	}
	_, options = loadOptions(t, "", "--base-url", "https://www.truestamp.com", "--environment", "staging")
	if _, rejection, reader := configureOptions(t, options); rejection == nil || rejection.Code != specdata.ReasonConfigurationError || len(reader.urls) != 0 {
		t.Fatalf("www.truestamp.com under staging: %v, %q", rejection, reader.urls)
	}
	_, options = loadOptions(t, "", "--base-url", "https://staging.truestamp.example", "--offline")
	if run, _, reader := configureOptions(t, options); run.TrustRoot.Kind != KindCompiledPin || len(reader.urls) != 0 {
		t.Fatalf("offline: %+v, %q", run.TrustRoot, reader.urls)
	}

	for _, www := range []string{
		"https://www.truestamp.com:443",
		"https://www.truestamp.com.",
		"http://www.truestamp.com",
		"HTTPS://WWW.TRUESTAMP.COM.:443/x",
		"https://www.truestamp.com:8443",
	} {
		_, options = loadOptions(t, "", "--base-url", www)
		if run, rejection, reader := configureOptions(t, options); rejection != nil ||
			run.TrustRoot != (TrustRoot{Kind: KindCurrentKeyring, Location: CurrentKeyringURL}) ||
			len(reader.urls) != 1 || reader.urls[0] != CurrentKeyringURL {
			t.Errorf("%s under prod: %v, %+v, read %q", www, rejection, run, reader.urls)
		}
		_, options = loadOptions(t, "", "--base-url", www, "--environment", "staging")
		if _, rejection, reader := configureOptions(t, options); rejection == nil ||
			rejection.Code != specdata.ReasonConfigurationError || len(reader.urls) != 0 {
			t.Errorf("%s under staging: %v, read %q", www, rejection, reader.urls)
		}
	}
	staging := writeKeyring(t, testKey("staging"))
	_, options = loadOptions(t, "[verify.keyrings]\nstaging = \""+staging+"\"\n",
		"--base-url", "https://www.truestamp.com:443", "--environment", "staging")
	if run, rejection, reader := configureOptions(t, options); rejection != nil ||
		run.TrustRoot != (TrustRoot{Kind: KindSuppliedKeyring, Location: staging}) || len(reader.urls) != 0 {
		t.Errorf("www.truestamp.com:443 beside a configured staging keyring: %v, %+v, read %q", rejection, run, reader.urls)
	}

	key := nonproductionKey(t)
	data, err := os.ReadFile(writeKeyring(t, key))
	if err != nil {
		t.Fatal(err)
	}
	_, options = loadOptions(t, "", "--base-url", "https://staging.truestamp.example", "--environment", "staging")
	run, rejection, reader = configureOptionsWith(t, options, data)
	if rejection != nil || run.TrustRoot != (TrustRoot{Kind: KindSuppliedKeyring, Location: origin}) ||
		len(reader.urls) != 1 || run.Keyring == nil || run.KeyringUnavailable {
		t.Fatalf("a staging origin's non-production key under staging: %v, %+v, read %q", rejection, run, reader.urls)
	}
	if _, ok := run.Keyring.Find(key); !ok {
		t.Fatal("the staging origin's key is not in the run's keyring")
	}

	file := writeKeyring(t, testKey("file"))
	for _, environment := range []string{"prod", "staging"} {
		_, options = loadOptions(t, "", "--keyring", file, "--base-url", "https://staging.truestamp.example",
			"--environment", environment)
		if run, rejection, reader := configureOptions(t, options); rejection != nil ||
			run.TrustRoot != (TrustRoot{Kind: KindSuppliedKeyring, Location: file}) || len(reader.urls) != 0 {
			t.Errorf("--keyring FILE and --base-url under %s: %v, %+v, read %q", environment, rejection, run, reader.urls)
		}
	}
	if got := refusals.Load() - before; got != 0 {
		t.Fatalf("%d requests reached the shared client", got)
	}
}

// A configured keyring serves only the environment the configuration file
// names it for: a run under --environment staging loads the staging
// keyring, and a run under prod, offline or online, ignores it. --keyring
// FILE and --no-keyring given for the run come first, and a configured
// keyring that fails a load check is configuration_error. Mutant killed:
// the configured staging keyring used whatever E_v is.
func TestOptions_AConfiguredKeyringServesOnlyItsEnvironment(t *testing.T) {
	clearTruestampEnv(t)
	staging := writeKeyring(t, testKey("staging"))
	toml := "[verify.keyrings]\nstaging = \"" + staging + "\"\n"

	cfg, options := loadOptions(t, toml, "--environment", "staging", "--offline")
	if cfg.Verify.Keyrings.Staging != staging || options.Configured.For("staging") != staging {
		t.Fatalf("the configured keyring did not load: %+v", cfg.Verify.Keyrings)
	}
	run, rejection, _ := configureOptions(t, options)
	if rejection != nil || run.TrustRoot != (TrustRoot{Kind: KindSuppliedKeyring, Location: staging}) {
		t.Fatalf("under staging: %v, %+v", rejection, run)
	}
	if _, ok := run.Keyring.Find(testKey("staging")); !ok {
		t.Fatal("the staging key is not in the run's keyring")
	}

	_, options = loadOptions(t, toml, "--offline")
	if run, rejection, _ := configureOptions(t, options); rejection != nil || run.TrustRoot.Kind != KindCompiledPin {
		t.Fatalf("under prod offline: %v, %+v", rejection, run.TrustRoot)
	}
	_, options = loadOptions(t, toml)
	if run, _, reader := configureOptions(t, options); run.TrustRoot.Kind != KindCurrentKeyring || reader.urls[0] != CurrentKeyringURL {
		t.Fatalf("under prod online: %+v", run.TrustRoot)
	}
	_, options = loadOptions(t, toml, "--environment", "dev", "--offline")
	if _, rejection, _ := configureOptions(t, options); rejection == nil || rejection.Code != specdata.ReasonConfigurationError {
		t.Fatalf("under dev, which has none: %v", rejection)
	}

	other := writeKeyring(t, testKey("other"))
	_, options = loadOptions(t, toml, "--environment", "staging", "--keyring", other)
	if run, _, _ := configureOptions(t, options); run.TrustRoot.Location != other {
		t.Fatalf("--keyring FILE: %+v", run.TrustRoot)
	}
	_, options = loadOptions(t, toml, "--environment", "staging", "--no-keyring")
	if run, _, _ := configureOptions(t, options); run.TrustRoot.Kind != KindNone {
		t.Fatalf("--no-keyring: %+v", run.TrustRoot)
	}

	vector := "[verify.keyrings]\nstaging = \"" + vectorcases.Path("keyring.test.json") + "\"\n"
	_, options = loadOptions(t, vector, "--environment", "staging")
	if _, rejection, _ := configureOptions(t, options); rejection == nil || rejection.Code != specdata.ReasonConfigurationError {
		t.Fatalf("a configured keyring holding VECTOR_KEY under staging: %v", rejection)
	}
	missing := "[verify.keyrings]\nstaging = \"" + filepath.Join(t.TempDir(), "missing.json") + "\"\n"
	_, options = loadOptions(t, missing, "--environment", "staging")
	cfg2, err := options.Config(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, rejection, err := Configure(cfg2); !errors.Is(err, strictjson.ErrUnreadable) || rejection != nil {
		t.Fatalf("a configured keyring that cannot be read: %v, %v", rejection, err)
	}
}

// The command line refusals: --keyring with --no-keyring, an
// --environment that is not an E_v, and an empty --keyring. Mutant killed:
// Config without the --keyring and --no-keyring refusal (keyless wins).
func TestOptions_Refusals(t *testing.T) {
	clearTruestampEnv(t)
	for name, args := range map[string][]string{
		"--keyring and --no-keyring": {"--keyring", vectorcases.Path("keyring.test.json"), "--no-keyring"},
		"--environment production":   {"--environment", "production"},
		"--environment empty":        {"--environment", ""},
	} {
		flags := commandFlags(t, args...)
		cfg, err := config.Load(filepath.Join(t.TempDir(), "none.toml"), flags)
		if err != nil {
			t.Fatal(err)
		}
		options, err := OptionsFromCommand(flags, cfg)
		if err == nil {
			_, err = options.Config(nil)
		}
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("%s: %v", name, err)
		}
	}
	flags := commandFlags(t, "--keyring", "")
	cfg, err := config.Load(filepath.Join(t.TempDir(), "none.toml"), flags)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OptionsFromCommand(flags, cfg); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("--keyring empty: %v", err)
	}
}

// A flag set without the run options, as the verify command's is today,
// reads as none given. Mutant killed: OptionsFromCommand without its check
// that Lookup found the flag (the bare flag set panics).
func TestOptions_FlagsTheCommandDoesNotRegister(t *testing.T) {
	flags := pflag.NewFlagSet("bare", pflag.ContinueOnError)
	options, err := OptionsFromCommand(flags, &config.Config{})
	if err != nil || options != (Options{Environment: "prod", Offline: false}) {
		t.Fatalf("options = %+v, %v", options, err)
	}
}
