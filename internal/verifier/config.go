// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds a run's configuration, what step CONFIGURATION (SPEC.md
// section 10.2, step 0) fixes before the file is read.
//
// A Config takes its members from the command's options or, in a
// conformance test, from a case's configuration (SPEC.md section 13.2 (The
// vectors)). An external verifier makes its keyring read and its source
// reads only in an online run (SPEC.md sections 8.3 and 10.4), so a
// Config says whether the run is online and holds the seams an online
// keyring read and a source's identity read go through, and the Stellar and
// Bitcoin sources the operator pins (SPEC.md section 9); its keyring
// sources include a configured keyring read from a file the configuration
// file names (SPEC.md section 8.3), and the keyring read from the origin an
// explicit --base-url names; and it holds no expected hash yet, which joins
// with step EXPECTED_HASH. options.go sets a Config from the command's
// options, and sources.go confirms its sources.

package verifier

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier/source"
)

// CurrentKeyringURL is where an external verifier reads Truestamp's
// keyring online (SPEC.md section 8.3).
const CurrentKeyringURL = "https://www.truestamp.com/.well-known/keyring.json"

// SourceKind is how a run's keyring is chosen.
type SourceKind int

// The keyring sources of a run (SPEC.md section 8.3).
const (
	// SourceDefault: no keyring is asked for. Under E_v prod, the keyring
	// read from www.truestamp.com governs an online run and the compiled
	// pin an offline one; under any other E_v the run stops at
	// CONFIGURATION with configuration_error (section 6.4).
	SourceDefault SourceKind = iota
	// SourceKeyless: keyless mode (--no-keyring); no keyring governs.
	SourceKeyless
	// SourceFile: a supplied keyring read from a file (--keyring FILE).
	SourceFile
	// SourceConfigured: a configured keyring, a supplied keyring read from
	// the file the configuration file names for the run's E_v.
	SourceConfigured
	// SourceOrigin: a supplied keyring read online from the origin an
	// explicit --base-url names.
	SourceOrigin
)

// KeyringSource is the keyring a run asks for.
type KeyringSource struct {
	Kind SourceKind
	// Path is the keyring file, for SourceFile and SourceConfigured.
	Path string
	// URL is the keyring's URL, for SourceOrigin.
	URL string
}

// KeyringRead is what an online keyring read gives: the keyring's bytes as
// read, which load under SPEC.md section 10.1 and give trust_root its
// digest d (section 8.3); or, from a conformance case's replay file
// (section 13.2 (The vectors)), Replayed and the keyring as Value, a value
// the replay's own reader already read under section 10.1, which has no
// bytes of its own, so trust_root's digest is null (section 10.5).
type KeyringRead struct {
	// Bytes are the keyring's bytes as read, when Replayed is false.
	Bytes []byte
	// Replayed says the answer is Value, with no bytes.
	Replayed bool
	// Value is the replayed keyring, when Replayed is true.
	Value any
}

// KeyringReader reads the keyring at url online, or gives an error for a
// read that gave no keyring (a transport failure, an HTTP error, a
// redirect, a host that is not a valid DNS name, an unreachable answer in a
// replay). It is the seam every online keyring read goes through:
// [NewKeyringReader] reads over HTTP, and a conformance test answers from a
// case's replay file.
type KeyringReader func(url string) (KeyringRead, error)

// Config is a run's configuration (SPEC.md section 10.2, step 0). E_v,
// keyless mode and --keyring FILE come only from the run's options
// (sections 6.4 and 8.3). A configured keyring comes from the
// configuration file, one for each non-production environment, and is
// used only when --environment names it. Whether the run is offline, and
// the sources the operator pins, may also come from the configuration
// file or an environment variable, and a Bitcoin Core node's RPC
// credentials come only from one of those two (section 9). The network
// pins are the defaults of E_v (section 9); no option sets them, and only
// a conformance test replaces them with a case's. No document sets any
// member.
type Config struct {
	// Environment is E_v (SPEC.md section 6.4): prod, or an ENV value.
	Environment string
	// Keyring is the keyring source.
	Keyring KeyringSource
	// Networks are the network pins (SPEC.md section 9), which POLICY
	// compares every network of the proof with: the row of
	// spec/network-pins.json for E_v, which [NewConfig] sets and nothing a
	// user gives replaces, since no flag, configuration key or environment
	// variable reaches this member ([Options.Config] starts from
	// NewConfig). It is also the conformance seam: a conformance test sets
	// it to a case's networks (SPEC.md section 13.2), which in 25 cases
	// differ from the defaults of the case's E_v.
	Networks specdata.NetworkPins
	// Online says whether the run reads outside the document.
	Online bool
	// ReadKeyring reads a keyring online. A nil reader reads nothing, so an
	// online keyring read through it fails and binds no key.
	ReadKeyring KeyringReader
	// Sources are the Stellar and Bitcoin sources the operator pins, at
	// most one of each provider, each in place of the compiled-in default
	// of the networks it serves (SPEC.md section 9).
	Sources []Source
	// ReadIdentity reads a source's identity in an online run. A nil
	// reader reads nothing, so each source gives no usable answer.
	ReadIdentity IdentityReader
}

// ErrInvalidConfig marks a configuration no run can take, such as an
// environment that is not an E_v, or two keyring sources. It is an invalid
// command line (exit code 1, SPEC.md section 12.2), not a verification
// result.
var ErrInvalidConfig = errors.New("invalid verifier configuration")

// NewConfig returns the configuration of an offline run under E_v
// environment, with no keyring option and the default pins of that E_v,
// or an error wrapping [ErrInvalidConfig] when environment is not an E_v.
func NewConfig(environment string) (Config, error) {
	pins, ok := specdata.DefaultNetworkPins(environment)
	if !ok {
		return Config{}, fmt.Errorf("%w: %q is not an environment a verifier knows (one of %v)", ErrInvalidConfig, environment, specdata.Environments())
	}
	return Config{Environment: environment, Networks: pins}, nil
}

// Validate reports a configuration no run can take, wrapping
// [ErrInvalidConfig]: an environment that is not an E_v, a pin that names
// no network of its chain, a keyring source without its file or URL, a
// keyring read online in an offline run, or a source whose provider is not
// one of its chain's, whose URL [source.CheckURL] refuses or whose host is
// neither an IP address nor a valid DNS name, that names a network not of
// its chain, or that carries credentials and is not a Bitcoin Core node.
// No error quotes a source's URL.
func (c Config) Validate() error {
	if !slices.Contains(specdata.Environments(), c.Environment) {
		return fmt.Errorf("%w: %q is not an environment a verifier knows (one of %v)", ErrInvalidConfig, c.Environment, specdata.Environments())
	}
	networks := specdata.Networks()
	for _, pin := range []struct{ name, chain, network string }{
		{"commitment", "bitcoin", c.Networks.Commitment.Bitcoin},
		{"commitment", "stellar", c.Networks.Commitment.Stellar},
		{"entropy", "bitcoin", c.Networks.Entropy.Bitcoin},
		{"entropy", "stellar", c.Networks.Entropy.Stellar},
	} {
		if !slices.Contains(networks[pin.chain], pin.network) {
			return fmt.Errorf("%w: the %s pin of %s, %q, is not one of its networks", ErrInvalidConfig, pin.name, pin.chain, pin.network)
		}
	}
	switch c.Keyring.Kind {
	case SourceDefault, SourceKeyless:
		if c.Keyring.Path != "" || c.Keyring.URL != "" {
			return fmt.Errorf("%w: a keyring source with no file or URL names one", ErrInvalidConfig)
		}
	case SourceFile, SourceConfigured:
		if c.Keyring.Path == "" || c.Keyring.URL != "" {
			return fmt.Errorf("%w: a keyring file source names no file", ErrInvalidConfig)
		}
	case SourceOrigin:
		if c.Keyring.URL == "" || c.Keyring.Path != "" {
			return fmt.Errorf("%w: a keyring read from an origin names no URL", ErrInvalidConfig)
		}
		if !c.Online {
			return fmt.Errorf("%w: an offline run reads no keyring from an origin", ErrInvalidConfig)
		}
	default:
		return fmt.Errorf("%w: an unknown keyring source", ErrInvalidConfig)
	}
	for _, s := range c.Sources {
		if err := checkSource(s); err != nil {
			return err
		}
	}
	return nil
}

// checkSource refuses a source no run can read; see [Config.Validate].
func checkSource(s Source) error {
	if s.Chain == "" || s.Provider.Chain() != s.Chain {
		return fmt.Errorf("%w: a source of the chain %q has the provider %q, which does not serve it", ErrInvalidConfig, s.Chain, s.Provider)
	}
	if s.Network != "" && !slices.Contains(specdata.Networks()[s.Chain], s.Network) {
		return fmt.Errorf("%w: a %s source names %q, which is not one of its networks", ErrInvalidConfig, chainName(s.Chain), s.Network)
	}
	if err := checkSourceURL(s.URL); err != nil {
		return fmt.Errorf("%w: the %s URL: %w", ErrInvalidConfig, s.Provider, err)
	}
	if !s.Credentials.Empty() && s.Provider != source.BitcoinCore {
		return fmt.Errorf("%w: only a Bitcoin Core node takes RPC credentials", ErrInvalidConfig)
	}
	return nil
}

// checkSourceURL refuses a source URL as [source.CheckURL] does, and one
// whose host is neither an IP address nor a valid DNS name. Its error
// never quotes the URL.
func checkSourceURL(raw string) error {
	if err := source.CheckURL(raw); err != nil {
		return err
	}
	u, _ := url.Parse(raw)
	if host := u.Hostname(); net.ParseIP(host) == nil && !validDNSName(host) {
		return errors.New("the URL's host is not a valid DNS name")
	}
	return nil
}
