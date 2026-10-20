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
// keyring read, a source's identity read and the reads of CHAIN and SOURCES
// go through, and the Stellar and
// Bitcoin sources the operator pins, with each network's several sources and
// their source policy where the operator configures them (SPEC.md section 9,
// "Sources" and "Several sources for one network"); its keyring
// sources include a configured keyring read from a file the configuration
// file names (SPEC.md section 8.3), and the keyring read from the origin an
// explicit --base-url names; and it holds the caller's expected hash, which
// step EXPECTED_HASH compares (SPEC.md section 10.2, step 13). options.go
// sets a Config from the command's options, and sources.go confirms its
// sources.

package verifier

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"

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
	// it to a case's networks (SPEC.md section 13.2), which in some cases
	// differ from the defaults of the case's E_v.
	Networks specdata.NetworkPins
	// Online says whether the run reads outside the document.
	Online bool
	// ReadKeyring reads a keyring online. A nil reader reads nothing, so an
	// online keyring read through it fails and binds no key.
	ReadKeyring KeyringReader
	// Sources are the Stellar and Bitcoin sources the operator gives for
	// no network, at most one of each provider in the command's options,
	// in the order of those options, each in place of the compiled-in
	// default of the pinned networks with no configuration that its
	// identity read shows it serves (SPEC.md section 9); one whose answer
	// only networks with a configuration allow serves nothing. Each has
	// Network "".
	Sources []Source
	// NetworkSources are the operator's configurations of a network's
	// several sources, at most one for each network, each with its
	// sources in order and its source policy (SPEC.md section 9, "Several
	// sources for one network"). A pinned network with one serves no other
	// source and reads no default. No document sets any.
	NetworkSources []NetworkSources
	// ReadIdentity reads a source's identity in an online run. A nil
	// reader reads nothing, so each source gives no usable answer.
	ReadIdentity IdentityReader
	// ReadChain makes the reads CHAIN and SOURCES make of a source whose
	// identity was confirmed (SPEC.md section 10.4). A nil reader reads
	// nothing, so each such read gives no usable answer.
	ReadChain ChainReader
	// ExpectedHash is the caller's hash for step EXPECTED_HASH (SPEC.md
	// section 10.2, step 13), as the caller gave it, or nil when none is
	// given. Any string is taken: one that is not hex of the claims'
	// hash_type's length is that step's failing row, not an invalid
	// configuration.
	ExpectedHash *string
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
// keyring read online in an offline run, a source whose provider is not
// one of its chain's, whose URL [source.CheckURL] refuses or whose host is
// neither an IP address nor a valid DNS name, that names a network not of
// its chain, or that carries credentials and is not a Bitcoin Core node, a
// source of [Config.Sources] that names a network, a network configuration
// [checkNetworkSources] refuses, and two configurations of one network.
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
		if s.Network != "" {
			return fmt.Errorf("%w: a %s source given for no network names the network %q; a network's own sources are its NetworkSources",
				ErrInvalidConfig, chainName(s.Chain), s.Network)
		}
		if err := checkSource(s); err != nil {
			return err
		}
	}
	for i, n := range c.NetworkSources {
		if err := checkNetworkSources(n); err != nil {
			return err
		}
		if slices.ContainsFunc(c.NetworkSources[:i], func(m NetworkSources) bool { return m.Chain == n.Chain && m.Network == n.Network }) {
			return fmt.Errorf("%w: two configurations of the sources of %s's %s network", ErrInvalidConfig, chainName(n.Chain), n.Network)
		}
	}
	return nil
}

// checkNetworkSources refuses a network's configuration no run can take
// (see [Config.Validate]): a network that is not one of a Stellar or
// Bitcoin chain's in spec/networks.json, so any of NIST's, the Randomness
// Beacon being one source (SPEC.md section 9); no source; a policy other
// than first, at_least and all, or an n under first or all; a source of
// another chain or network, or one [checkSource] refuses; and one server
// twice ([sameServer]), which would count one server's outcome twice under
// at_least and all. An at_least n out of range is no invalid configuration
// but configuration_error at step 0 ([checkPolicies]). An error about a
// source names it by its place in the network's order, from 1, and no
// error quotes a source's URL.
func checkNetworkSources(n NetworkSources) error {
	if !slices.Contains(specdata.Networks()[n.Chain], n.Network) {
		return fmt.Errorf("%w: a configuration of sources names the chain %q and the network %q, which is not one of the Stellar or Bitcoin networks; "+
			"NIST's Randomness Beacon is one source", ErrInvalidConfig, n.Chain, n.Network)
	}
	if len(n.Sources) == 0 {
		return fmt.Errorf("%w: the configuration of %s's %s network names no source", ErrInvalidConfig, chainName(n.Chain), n.Network)
	}
	switch n.Policy.Kind {
	case SourcePolicyFirst, SourcePolicyAll:
		if n.Policy.N != 0 {
			return fmt.Errorf("%w: the policy %s of %s's %s network takes no number", ErrInvalidConfig, n.Policy.Kind, chainName(n.Chain), n.Network)
		}
	case SourcePolicyAtLeast:
	default:
		return fmt.Errorf("%w: the sources of %s's %s network have the policy %q, not first, at_least or all", ErrInvalidConfig, chainName(n.Chain), n.Network, n.Policy.Kind)
	}
	for i, s := range n.Sources {
		place := fmt.Sprintf("source %d of %s's %s network", i+1, chainName(n.Chain), n.Network)
		if s.Chain != n.Chain || (s.Network != "" && s.Network != n.Network) {
			return fmt.Errorf("%w: %s is configured for another chain or network", ErrInvalidConfig, place)
		}
		if err := sourceFault(s); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrInvalidConfig, place, err)
		}
		if j := slices.IndexFunc(n.Sources[:i], func(t Source) bool { return sameServer(s, t) }); j >= 0 {
			return fmt.Errorf("%w: sources %d and %d of %s's %s network name one %s server", ErrInvalidConfig, j+1, i+1, chainName(n.Chain), n.Network, s.Provider)
		}
	}
	return nil
}

// sameServer reports whether two sources are one server given twice: one
// provider, one scheme, one host, lowercased with one trailing dot dropped
// as [hostOf] gives it, one port, the scheme's default (443 for https, 80
// for http) when the URL names none, and one path, with any trailing
// slashes dropped, as the reads join a path to it. It catches one server
// spelled two ways; two names for one machine are beyond any comparison
// of URLs.
func sameServer(a, b Source) bool {
	server := func(s Source) string {
		u, err := url.Parse(s.URL)
		host, ok := hostOf(s.URL)
		if err != nil || !ok {
			return s.URL
		}
		port := u.Port()
		if port == "" {
			port = map[string]string{"https": "443", "http": "80"}[strings.ToLower(u.Scheme)]
		}
		return strings.ToLower(u.Scheme) + "://" + net.JoinHostPort(host, port) + strings.TrimRight(u.EscapedPath(), "/")
	}
	return a.Provider == b.Provider && server(a) == server(b)
}

// checkSource refuses a source no run can read; see [Config.Validate].
func checkSource(s Source) error {
	if err := sourceFault(s); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return nil
}

// sourceFault is why no run can read source s, or nil (see
// [checkSource]). It never quotes the source's URL.
func sourceFault(s Source) error {
	if s.Chain == "" || s.Provider.Chain() != s.Chain {
		return fmt.Errorf("a source of the chain %q has the provider %q, which does not serve it", s.Chain, s.Provider)
	}
	if s.Network != "" && !slices.Contains(specdata.Networks()[s.Chain], s.Network) {
		return fmt.Errorf("a %s source names %q, which is not one of its networks", chainName(s.Chain), s.Network)
	}
	if err := checkSourceURL(s.URL); err != nil {
		return fmt.Errorf("the %s URL: %w", s.Provider, err)
	}
	if !s.Credentials.Empty() && s.Provider != source.BitcoinCore {
		return errors.New("only a Bitcoin Core node takes RPC credentials")
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
