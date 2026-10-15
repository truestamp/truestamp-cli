// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.Config
// (elixir/lib/truestamp/verifier/config.ex in truestamp-verification),
// checked against the Truestamp application's port,
// Truestamp.Proof.Verifier.Config (lib/truestamp/proof/verifier/config.ex
// in truestamp-v2): a run's configuration, what step CONFIGURATION (SPEC.md
// section 10.2, step 0) fixes before the file is read.
//
// Adapted from the reference, which is offline and takes its members from
// its command's options and from a conformance case: the CLI also runs
// online, so a Config says whether the run is online and holds the seam an
// online keyring read goes through; its keyring sources add a configured
// keyring read from a file the configuration file names (SPEC.md section
// 8.3), and the keyring read from the origin an explicit --base-url names;
// and it holds no expected hash yet, which joins with step EXPECTED_HASH.
// options.go sets a Config from the command's options.

package verifier

import (
	"errors"
	"fmt"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/specdata"
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

// KeyringReader reads the keyring at url online and gives its bytes as
// read, or an error for a read that gave none (a transport failure, an
// HTTP error, a redirect). It is the seam every online keyring read goes
// through.
type KeyringReader func(url string) ([]byte, error)

// Config is a run's configuration (SPEC.md section 10.2, step 0). E_v,
// keyless mode and --keyring FILE come only from the run's options
// (sections 6.4 and 8.3). A configured keyring comes from the
// configuration file, one for each non-production environment, and is
// used only when --environment names it. Whether the run is offline may
// also come from the configuration file or an environment variable. The
// network pins are the defaults of E_v (section 9). No document sets any
// member.
type Config struct {
	// Environment is E_v (SPEC.md section 6.4): prod, or an ENV value.
	Environment string
	// Keyring is the keyring source.
	Keyring KeyringSource
	// Networks are the network pins (SPEC.md section 9): by default the row
	// of spec/network-pins.json for E_v.
	Networks specdata.NetworkPins
	// Online says whether the run reads outside the document.
	Online bool
	// ReadKeyring reads a keyring online. A nil reader reads nothing, so an
	// online keyring read through it fails and binds no key.
	ReadKeyring KeyringReader
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
// no network of its chain, a keyring source without its file or URL, or a
// keyring read online in an offline run.
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
	return nil
}
