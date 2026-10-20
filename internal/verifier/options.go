// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file sets a run's [Config] from the options of `truestamp verify`,
// each from where SPEC.md sections 6.4 (The environment marker), 8.3
// (External verifiers) and 9 (What a verifier pins) let it come from:
//
//   - E_v comes only from the run's --environment, never from the
//     configuration file, an environment variable, the base URL or the
//     document;
//   - keyless mode comes only from the run's --no-keyring, and --keyring
//     FILE comes only from the run's flag; the two together are refused;
//   - a configured keyring comes from the configuration file alone, one
//     for each non-production environment, and is used only when the
//     run's --environment names that environment; internal/config drops
//     every environment variable that would reach the table or a key
//     under it;
//   - the keyring is read from another origin only when the run is given
//     an explicit --base-url whose host is not www.truestamp.com; the
//     configured base URL (the configuration file or TRUESTAMP_BASE_URL)
//     points the CLI's API commands, and never moves the keyring read, so
//     a run under E_v prod reads it from www.truestamp.com;
//   - whether the run is offline may come from the configuration file or an
//     environment variable as well as --offline (verify.offline);
//   - the Stellar and Bitcoin sources the operator pins (SPEC.md section 9
//     (What a verifier pins), "Sources") may come from a flag, the
//     configuration file or an environment variable: --horizon,
//     --esplora and --bitcoin-core, verify.horizon, verify.esplora and
//     verify.bitcoin_core, TRUESTAMP_VERIFY_HORIZON,
//     TRUESTAMP_VERIFY_ESPLORA and TRUESTAMP_VERIFY_BITCOIN_CORE, a flag
//     over a variable over the file, as internal/config resolves them;
//   - a Bitcoin Core node's RPC credentials come from the configuration
//     file or the environment only (verify.bitcoin_core_user and
//     verify.bitcoin_core_password, TRUESTAMP_VERIFY_BITCOIN_CORE_USER and
//     TRUESTAMP_VERIFY_BITCOIN_CORE_PASSWORD), never from the command
//     line: no flag of theirs exists, a source URL that carries user
//     information is refused wherever it comes from, no error quotes a
//     source URL, and a report prints one as its scheme, host and port
//     alone, never its path;
//   - no option sets a network pin: the pins are E_v's defaults.
//
// A repeated option is not refused: a flag set holds one value for each
// flag.

package verifier

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/pflag"

	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// The flags of `truestamp verify` that set a run's keyring, environment
// and sources, and the root flag whose explicit use moves the keyring read.
const (
	FlagEnvironment = "environment"
	FlagKeyring     = "keyring"
	FlagNoKeyring   = "no-keyring"
	FlagBaseURL     = "base-url"
	FlagHorizon     = "horizon"
	FlagEsplora     = "esplora"
	FlagBitcoinCore = "bitcoin-core"
)

// RegisterFlags adds to flags --environment and --no-keyring, the two run
// options only a run may set (SPEC.md sections 6.4 and 8.3), neither of
// them a config key, and --horizon, --esplora and --bitcoin-core, the
// sources the operator pins (section 9), which internal/config also reads
// from the configuration file and the environment. No flag takes a Bitcoin
// Core node's RPC credentials.
func RegisterFlags(flags *pflag.FlagSet) {
	flags.String(FlagEnvironment, "prod",
		"Check proofs of this Truestamp environment, for this run only: prod, dev, staging or test")
	flags.Bool(FlagNoKeyring, false,
		"Keyless mode, for this run only: check the signature only against the key the proof carries")
	flags.String(FlagHorizon, "",
		"Read Stellar from this Horizon server, in place of the built-in one for the network it serves")
	flags.String(FlagEsplora, "",
		"Read Bitcoin from this Esplora server, in place of the built-in one for the network it serves")
	flags.String(FlagBitcoinCore, "",
		"Read Bitcoin from the Bitcoin Core node at this JSON-RPC URL; "+
			"its RPC credentials come only from the configuration file or the environment")
}

// Options are the options of one run of `truestamp verify` that decide its
// [Config].
type Options struct {
	// Environment is the run's --environment, or prod when none is given.
	Environment string
	// KeyringFile is the run's --keyring FILE, or "".
	KeyringFile string
	// NoKeyring is the run's --no-keyring.
	NoKeyring bool
	// OriginKeyringURL is the keyring URL of the origin an explicit
	// --base-url names, or "" when the run was given none.
	OriginKeyringURL string
	// Offline says the run reads nothing outside the document.
	Offline bool
	// Configured are the configured keyrings, by environment.
	Configured config.KeyringsConfig
	// Horizon, Esplora and BitcoinCore are the URLs of the sources the
	// operator pins, each "" when none is given.
	Horizon, Esplora, BitcoinCore string
	// BitcoinCoreCredentials are the Bitcoin Core node's RPC credentials,
	// from the configuration file or the environment; they print masked.
	BitcoinCoreCredentials source.Credentials
}

// OptionsFromCommand reads a run's options from the flags of the command
// that runs it and from the resolved configuration, which config.Load
// resolved with those flags. Only the flags given for the run are read for
// E_v, the keyring file, keyless mode and the keyring's origin; cfg gives
// the configured keyrings, the offline flag, the keyring URL an explicit
// --base-url composes, the sources' URLs (a flag over an environment
// variable over the configuration file) and the Bitcoin Core node's RPC
// credentials (an environment variable over the configuration file). A
// flag the command does not register is read as not given.
func OptionsFromCommand(flags *pflag.FlagSet, cfg *config.Config) (Options, error) {
	options := Options{
		Environment:            "prod",
		Offline:                cfg.Verify.Offline,
		Configured:             cfg.Verify.Keyrings,
		Horizon:                cfg.Verify.Horizon,
		Esplora:                cfg.Verify.Esplora,
		BitcoinCore:            cfg.Verify.BitcoinCore,
		BitcoinCoreCredentials: source.NewCredentials(cfg.Verify.BitcoinCoreUser, cfg.Verify.BitcoinCorePassword),
	}
	if flag := flags.Lookup(FlagEnvironment); flag != nil && flag.Changed {
		options.Environment = flag.Value.String()
	}
	if flag := flags.Lookup(FlagKeyring); flag != nil && flag.Changed {
		options.KeyringFile = flag.Value.String()
		if options.KeyringFile == "" {
			return Options{}, fmt.Errorf("%w: --keyring needs the path of a keyring file", ErrInvalidConfig)
		}
	}
	if flag := flags.Lookup(FlagNoKeyring); flag != nil && flag.Changed {
		noKeyring, err := flags.GetBool(FlagNoKeyring)
		if err != nil {
			return Options{}, fmt.Errorf("%w: --no-keyring: %w", ErrInvalidConfig, err)
		}
		options.NoKeyring = noKeyring
	}
	if flag := flags.Lookup(FlagBaseURL); flag != nil && flag.Changed {
		// config.Load gives a flag the user set precedence over the
		// configuration file and the environment, so the keyring URL it
		// composed is the explicit origin's.
		options.OriginKeyringURL = cfg.KeyringURL
	}
	return options, nil
}

// Config returns the run's configuration, with keyrings for its online
// keyring reads and identities for its sources' identity reads, or an
// error wrapping [ErrInvalidConfig] for an environment that is not an E_v,
// for --keyring with --no-keyring, or for a source URL [Config.Validate]
// refuses, one carrying user information among them, whose error names
// the option and never quotes the URL. Each source the options give is
// one the operator pins, serving whichever pins of its chain its identity
// read shows. The keyring source is the first of these the options give:
//
//  1. --no-keyring: keyless mode;
//  2. --keyring FILE;
//  3. online, an explicit --base-url whose host is not www.truestamp.com:
//     the keyring read from that origin;
//  4. under a non-production E_v, the configured keyring of that
//     environment;
//  5. none: under prod, the keyring read from www.truestamp.com online and
//     the compiled pin offline; under any other E_v, configuration_error
//     at CONFIGURATION.
//
// An offline run reads no keyring, so an explicit --base-url gives an
// offline run no keyring source. Nor does one whose host is
// www.truestamp.com, under any scheme or port: its keyring is Truestamp's,
// which a run under E_v prod reads as the default and a run under a
// non-production E_v must not use (SPEC.md section 6.4).
func (o Options) Config(keyrings KeyringReader, identities IdentityReader) (Config, error) {
	cfg, err := NewConfig(o.Environment)
	if err != nil {
		return Config{}, err
	}
	if o.KeyringFile != "" && o.NoKeyring {
		return Config{}, fmt.Errorf("%w: --keyring and --no-keyring cannot be given together", ErrInvalidConfig)
	}
	cfg.Online = !o.Offline
	cfg.ReadKeyring, cfg.ReadIdentity = keyrings, identities
	for _, pinned := range []struct {
		raw, option string
		s           Source
	}{
		{o.Horizon, "--horizon (verify.horizon, TRUESTAMP_VERIFY_HORIZON)", Source{Chain: ChainStellar, Provider: source.Horizon}},
		{o.Esplora, "--esplora (verify.esplora, TRUESTAMP_VERIFY_ESPLORA)", Source{Chain: ChainBitcoin, Provider: source.Esplora}},
		{o.BitcoinCore, "--bitcoin-core (verify.bitcoin_core, TRUESTAMP_VERIFY_BITCOIN_CORE)",
			Source{Chain: ChainBitcoin, Provider: source.BitcoinCore, Credentials: o.BitcoinCoreCredentials}},
	} {
		if pinned.raw == "" {
			continue
		}
		if err := checkSourceURL(pinned.raw); err != nil {
			return Config{}, fmt.Errorf("%w: the URL of %s: %w%s", ErrInvalidConfig, pinned.option, err, credentialsHint(pinned.s.Provider))
		}
		pinned.s.URL = pinned.raw
		cfg.Sources = append(cfg.Sources, pinned.s)
	}
	switch configured := o.Configured.For(o.Environment); {
	case o.NoKeyring:
		cfg.Keyring = KeyringSource{Kind: SourceKeyless}
	case o.KeyringFile != "":
		cfg.Keyring = KeyringSource{Kind: SourceFile, Path: o.KeyringFile}
	case cfg.Online && o.OriginKeyringURL != "" && !sameHost(o.OriginKeyringURL, CurrentKeyringURL):
		cfg.Keyring = KeyringSource{Kind: SourceOrigin, URL: o.OriginKeyringURL}
	case o.Environment != "prod" && configured != "":
		cfg.Keyring = KeyringSource{Kind: SourceConfigured, Path: configured}
	}
	return cfg, cfg.Validate()
}

// sameHost reports whether two URLs name one host, under any scheme or
// port: their hostnames compared ignoring case, each with one trailing dot
// trimmed, as DNS reads a name. www.truestamp.com's keyring is the one at
// that host however a URL spells it, https://www.truestamp.com:443,
// https://www.truestamp.com. and http://www.truestamp.com among them, so
// none of those is another origin's keyring.
func sameHost(a, b string) bool {
	x, okX := hostOf(a)
	y, okY := hostOf(b)
	return okX && okY && x == y
}

// hostOf is the hostname of raw, lowercased with one trailing dot trimmed,
// and whether raw is a URL that names one.
func hostOf(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", false
	}
	return strings.TrimSuffix(strings.ToLower(u.Hostname()), "."), true
}

// credentialsHint is where a refused source URL's error says credentials
// go: for a Bitcoin Core node, the configuration file or the environment.
func credentialsHint(provider source.Provider) string {
	if provider != source.BitcoinCore {
		return ""
	}
	return "; give a Bitcoin Core node's RPC credentials in the configuration file " +
		"(verify.bitcoin_core_user and verify.bitcoin_core_password) or the environment " +
		"(TRUESTAMP_VERIFY_BITCOIN_CORE_USER and TRUESTAMP_VERIFY_BITCOIN_CORE_PASSWORD), never in the URL or on the command line"
}
