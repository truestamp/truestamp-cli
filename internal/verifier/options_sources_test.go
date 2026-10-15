// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"bytes"
	"errors"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// The source options of SPEC.md section 9, "Sources", on a throwaway
// command: the verify command does not register them yet.

// A source URL comes from a flag, the configuration file or an
// environment variable, a flag over a variable over the file, for each of
// --horizon, --esplora and --bitcoin-core; the Bitcoin Core node's RPC
// credentials come from the file or the environment, a variable over the
// file; and Config gives the run one operator source of each provider,
// pinned to no one network, the credentials on the Bitcoin Core node's
// alone. Mutants killed: flagKeyMap without "bitcoin-core" (the flag is
// ignored, the variable wins); OptionsFromCommand taking Esplora's URL from
// verify.horizon; Config giving the credentials to every source (Validate
// refuses them); Config pinning an operator source to the first pinned
// network.
func TestOptions_SourcesComeFromAFlagTheFileOrTheEnvironment(t *testing.T) {
	const file = "[verify]\nhorizon = \"https://horizon.file.example\"\nesplora = \"https://esplora.file.example\"\n" +
		"bitcoin_core = \"http://core.file.example:8332\"\nbitcoin_core_user = \"file-user\"\nbitcoin_core_password = \"file-password\"\n"
	type urls struct{ horizon, esplora, core string }
	cases := []struct {
		name  string
		toml  string
		env   map[string]string
		args  []string
		want  urls
		creds source.Credentials
	}{
		{"none", "", nil, nil, urls{}, source.Credentials{}},
		{"the file", file, nil, nil,
			urls{"https://horizon.file.example", "https://esplora.file.example", "http://core.file.example:8332"},
			source.NewCredentials("file-user", "file-password")},
		{"the environment over the file", file, map[string]string{
			"TRUESTAMP_VERIFY_HORIZON":               "https://horizon.env.example",
			"TRUESTAMP_VERIFY_ESPLORA":               "https://esplora.env.example",
			"TRUESTAMP_VERIFY_BITCOIN_CORE":          "http://core.env.example:8332",
			"TRUESTAMP_VERIFY_BITCOIN_CORE_USER":     "env-user",
			"TRUESTAMP_VERIFY_BITCOIN_CORE_PASSWORD": "env-password",
		}, nil, urls{"https://horizon.env.example", "https://esplora.env.example", "http://core.env.example:8332"},
			source.NewCredentials("env-user", "env-password")},
		{"a flag over the environment", file, map[string]string{
			"TRUESTAMP_VERIFY_HORIZON":      "https://horizon.env.example",
			"TRUESTAMP_VERIFY_ESPLORA":      "https://esplora.env.example",
			"TRUESTAMP_VERIFY_BITCOIN_CORE": "http://core.env.example:8332",
		}, []string{"--horizon", "https://horizon.flag.example", "--esplora", "https://esplora.flag.example", "--bitcoin-core", "http://127.0.0.1:18443"},
			urls{"https://horizon.flag.example", "https://esplora.flag.example", "http://127.0.0.1:18443"},
			source.NewCredentials("file-user", "file-password")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clearTruestampEnv(t)
			for name, value := range c.env {
				t.Setenv(name, value)
			}
			_, options := loadOptions(t, c.toml, c.args...)
			if got := (urls{options.Horizon, options.Esplora, options.BitcoinCore}); got != c.want || options.BitcoinCoreCredentials != c.creds {
				t.Fatalf("options = %+v, want %+v", got, c.want)
			}
			cfg, err := options.Config(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			var want []Source
			if c.want.horizon != "" {
				want = []Source{
					{Chain: ChainStellar, Provider: source.Horizon, URL: c.want.horizon},
					{Chain: ChainBitcoin, Provider: source.Esplora, URL: c.want.esplora},
					{Chain: ChainBitcoin, Provider: source.BitcoinCore, URL: c.want.core, Credentials: c.creds},
				}
			}
			if len(cfg.Sources) != len(want) {
				t.Fatalf("sources = %+v", cfg.Sources)
			}
			for i := range want {
				if cfg.Sources[i] != want[i] {
					t.Fatalf("source %d = %+v, want %+v", i, cfg.Sources[i], want[i])
				}
			}
		})
	}
}

// runThrowaway runs a throwaway command that registers the verifier's
// options, as `truestamp verify` will, with args and with the
// configuration file text toml, and returns its error and everything it
// printed, on stdout, on stderr and to the standard library's two loggers,
// with the error's text.
func runThrowaway(t *testing.T, toml string, args ...string) (string, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	var logged bytes.Buffer
	savedSlog, savedLog := slog.Default(), log.Writer()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelDebug})))
	log.SetOutput(&logged)
	defer func() { slog.SetDefault(savedSlog); log.SetOutput(savedLog) }()

	cmd := &cobra.Command{
		Use: "verify",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(path, cmd.Flags())
			if err != nil {
				return err
			}
			options, err := OptionsFromCommand(cmd.Flags(), cfg)
			if err != nil {
				return err
			}
			_, err = options.Config(nil, nil)
			return err
		},
	}
	cmd.Flags().String(FlagBaseURL, "", "")
	cmd.Flags().String(FlagKeyring, "", "")
	cmd.Flags().Bool("offline", false, "")
	RegisterFlags(cmd.Flags())
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	printed := stdout.String() + stderr.String() + logged.String()
	if err != nil {
		printed += err.Error()
	}
	return printed, err
}

// Bitcoin Core's RPC credentials are never taken from the command line:
// no flag takes them, so --bitcoin-core-user and --bitcoin-core-password
// are unknown flags; and a source URL that carries user information is
// refused wherever it comes from, the command line, the environment or the
// configuration file, with an error that names the option and never the
// URL. A planted secret is absent from every byte of stdout, stderr, the
// error and the standard library's loggers, including for a URL that does
// not parse. Mutants killed: checkSourceURL without the user information
// check (the run is configured); CheckURL quoting the URL in its error;
// CheckURL returning url.Parse's error, which quotes the URL; a
// --bitcoin-core-password flag registered (the secret is taken).
func TestOptions_CoreCredentialsOnTheCommandLineAreRefused(t *testing.T) {
	const secret = "planted-secret-7f3c1e"
	for name, c := range map[string]struct {
		toml string
		env  map[string]string
		args []string
	}{
		"--bitcoin-core with user information":           {args: []string{"--bitcoin-core", "http://rpcuser:" + secret + "@127.0.0.1:8332"}},
		"--bitcoin-core with a user alone":               {args: []string{"--bitcoin-core=http://" + secret + "@127.0.0.1:8332"}},
		"--horizon with user information":                {args: []string{"--horizon", "https://user:" + secret + "@horizon.example"}},
		"--esplora with user information":                {args: []string{"--esplora", "https://user:" + secret + "@esplora.example"}},
		"--bitcoin-core that does not parse":             {args: []string{"--bitcoin-core", "http://rpcuser:" + secret + "@[::1"}},
		"--bitcoin-core-password":                        {args: []string{"--bitcoin-core-password=" + secret}},
		"--bitcoin-core-user":                            {args: []string{"--bitcoin-core-user", secret}},
		"a URL with user information in the environment": {env: map[string]string{"TRUESTAMP_VERIFY_BITCOIN_CORE": "http://u:" + secret + "@127.0.0.1:8332"}},
		"a URL with user information in the file":        {toml: "[verify]\nbitcoin_core = \"http://u:" + secret + "@127.0.0.1:8332\"\n"},
	} {
		t.Run(name, func(t *testing.T) {
			clearTruestampEnv(t)
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			printed, err := runThrowaway(t, c.toml, c.args...)
			if err == nil {
				t.Fatal("the run was configured")
			}
			if strings.Contains(printed, secret) {
				t.Fatalf("the secret is printed:\n%s", printed)
			}
			if strings.HasPrefix(name, "--bitcoin-core-") {
				if !strings.Contains(err.Error(), "unknown flag") {
					t.Fatalf("error = %v, want an unknown flag", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("error = %v, want ErrInvalidConfig", err)
			}
		})
	}
	// A Bitcoin Core URL with no user information, the credentials in the
	// environment, is configured, and nothing is printed.
	clearTruestampEnv(t)
	t.Setenv("TRUESTAMP_VERIFY_BITCOIN_CORE_PASSWORD", secret)
	if printed, err := runThrowaway(t, "", "--bitcoin-core", "http://127.0.0.1:8332"); err != nil || printed != "" {
		t.Fatalf("the credentials in the environment: %v, %q", err, printed)
	}
}

// RegisterFlags registers the five flags of the verifier's options and no
// other; none takes a credential, and the three source flags are config
// keys while --environment and --no-keyring are not. Mutant killed: a
// --bitcoin-core-user flag registered.
func TestRegisterFlags_TheSourceFlagsAndNoCredential(t *testing.T) {
	flags := pflag.NewFlagSet("verify", pflag.ContinueOnError)
	RegisterFlags(flags)
	var names []string
	flags.VisitAll(func(f *pflag.Flag) { names = append(names, f.Name) })
	want := []string{FlagBitcoinCore, FlagEnvironment, FlagEsplora, FlagHorizon, FlagNoKeyring}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Fatalf("flags = %v, want %v", names, want)
	}
	for _, name := range names {
		for _, word := range []string{"user", "password", "credential", "secret", "auth"} {
			if strings.Contains(name, word) {
				t.Errorf("--%s takes a credential", name)
			}
		}
	}
}
