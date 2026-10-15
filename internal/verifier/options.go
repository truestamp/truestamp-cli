// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the option handling of the reference verifier's
// Truestamp.Verifier.Config (elixir/lib/truestamp/verifier/config.ex in
// truestamp-verification), new/1, which takes a run's members from its
// command's options and refuses an unknown or repeated one, adapted to the
// options of `truestamp verify` and to what SPEC.md sections 6.4 (The
// environment marker) and 8.3 (External verifiers) let each come from:
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
//     environment variable as well as --offline (verify.offline).
//
// The reference's repeated-option refusal has no counterpart: a flag set
// holds one value for each flag.

package verifier

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/pflag"

	"github.com/truestamp/truestamp-cli/internal/config"
)

// The flags of `truestamp verify` that set a run's keyring and
// environment, and the root flag whose explicit use moves the keyring read.
const (
	FlagEnvironment = "environment"
	FlagKeyring     = "keyring"
	FlagNoKeyring   = "no-keyring"
	FlagBaseURL     = "base-url"
)

// RegisterFlags adds --environment and --no-keyring to flags, the two run
// options only a run may set (SPEC.md sections 6.4 and 8.3). Neither is a
// config key: internal/config maps no flag of these names.
func RegisterFlags(flags *pflag.FlagSet) {
	flags.String(FlagEnvironment, "prod",
		"Check proofs of this Truestamp environment, for this run only: prod, dev, staging or test")
	flags.Bool(FlagNoKeyring, false,
		"Keyless mode, for this run only: check the signature only against the key the proof carries")
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
}

// OptionsFromCommand reads a run's options from the flags of the command
// that runs it and from the resolved configuration. Only the flags given
// for the run are read for E_v, the keyring file, keyless mode and the
// keyring's origin; cfg gives only the configured keyrings, the offline
// flag, and the keyring URL an explicit --base-url composes. A flag the
// command does not register is read as not given.
func OptionsFromCommand(flags *pflag.FlagSet, cfg *config.Config) (Options, error) {
	options := Options{
		Environment: "prod",
		Offline:     cfg.Verify.Offline,
		Configured:  cfg.Verify.Keyrings,
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

// Config returns the run's configuration, with reader for its online
// keyring reads, or an error wrapping [ErrInvalidConfig] for an
// environment that is not an E_v or for --keyring with --no-keyring. The
// keyring source is the first of these the options give:
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
func (o Options) Config(reader KeyringReader) (Config, error) {
	cfg, err := NewConfig(o.Environment)
	if err != nil {
		return Config{}, err
	}
	if o.KeyringFile != "" && o.NoKeyring {
		return Config{}, fmt.Errorf("%w: --keyring and --no-keyring cannot be given together", ErrInvalidConfig)
	}
	cfg.Online = !o.Offline
	cfg.ReadKeyring = reader
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
