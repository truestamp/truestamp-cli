// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package vectorcases loads the conformance cases of SPEC.md section 13.2
// (The vectors), testdata/verification/vectors/cases.json, for the tests that
// run them. It is test support: no command imports it.
//
// A case decodes into typed structs, the members of a case that section
// 13.2 lists. The one runner over the cases is this package's
// runner_test.go, which runs every case through every step of section
// 10.2, each online case answered from its replay file by the replay
// adapter (replay.go). The conformance test of section 13.1,
// conformance_test.go's TestConformance, runs every case through that
// runner and compares exactly what section 13.1 compares; the per-step
// tests beside it compare more.
//
// cases.json is decoded by encoding/json/v2, which shares no code with the
// strict reader the runner tests, so a defect in that reader cannot hide in
// the cases it is compared with. The decode refuses unknown members, a
// repeated member name and invalid UTF-8, and the runner also parses
// cases.json with the strict reader.
// cases.json is larger than MAX_READ_OCTETS, which binds a document and not
// a data file (SPEC.md section 9), so that parse does not make J1.
package vectorcases

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// Steps are the steps of SPEC.md section 10.2, in their order.
var Steps = []string{
	"CONFIGURATION", "READ", "PARSE", "ENVELOPE", "ENVIRONMENT", "KEY_BINDING",
	"SIGNATURE", "SHAPE", "POLICY", "DERIVE", "CHAIN", "SOURCES", "WINDOW",
	"EXPECTED_HASH", "VERDICT",
}

// StepIndex returns the position of step in [Steps], or -1.
func StepIndex(step string) int {
	return slices.Index(Steps, step)
}

// Case is one case of cases.json (SPEC.md section 13.2).
type Case struct {
	// ID is the case's name, unique in the file.
	ID string `json:"id"`
	// Proof is the path of its proof, relative to vectors/.
	Proof    string   `json:"proof"`
	Config   Config   `json:"config"`
	Expected Expected `json:"expected"`
}

// Config is a case's configuration.
type Config struct {
	// Environment is the verifier's environment, E_v.
	Environment string `json:"environment"`
	// ExpectedHash is the caller's hash, or nil when none is given.
	ExpectedHash *string `json:"expected_hash"`
	// Keyring is test, none, unavailable or the path of a keyring file, or
	// nil when neither a keyring nor keyless mode is asked for.
	Keyring *string `json:"keyring"`
	// Networks gives every pin for the case.
	Networks Networks `json:"networks"`
	// Online is the path of the case's replay file under corpus/replay/, or
	// nil for an offline case.
	Online *string `json:"online"`
	// Sources gives, by chain and then network, the several sources of
	// each Stellar or Bitcoin network the case pins that has them (SPEC.md
	// section 9), only beside Online; a network it does not name has one
	// source, under the policy first (SPEC.md section 13.2).
	Sources map[string]map[string]NetworkSources `json:"sources"`
}

// The source policies of SPEC.md section 9.
const (
	PolicyFirst   = "first"
	PolicyAtLeast = "at_least"
	PolicyAll     = "all"
)

// NetworkSources is one network's sources in a case's Sources (SPEC.md
// section 13.2): Sources, the names of the network's sources in their
// order, each an ASCII16, one or more, no two the same; Policy, first,
// at_least or all; and Required, the n of at_least n, from 1 to the number
// of sources, present exactly when Policy is at_least.
type NetworkSources struct {
	Policy   string   `json:"policy"`
	Required *int     `json:"required"`
	Sources  []string `json:"sources"`
}

// Networks is a case's commitment pin per chain and entropy pin per source.
type Networks struct {
	Commitment map[string]string `json:"commitment"`
	Entropy    map[string]string `json:"entropy"`
}

// Expected is a case's expected output (SPEC.md section 13.2): a bare
// rejection's Code and SubReason, with no Label; or a report's Label and
// Rows, with Code, the report's code, for a case whose point is which
// failing row decides it, and the empty string otherwise.
type Expected struct {
	Code      string `json:"code"`
	SubReason string `json:"sub_reason"`
	Label     string `json:"label"`
	Rows      []Row  `json:"rows"`
	// Trace is the path of a trace file, for a case that gives a sound
	// proof's trace.
	Trace string `json:"trace"`
	// SubmittedAfter and SubmittedBefore are an online case's edges, each
	// null or {basis, instant}, kept as their JSON.
	SubmittedAfter  jsontext.Value `json:"submitted_after"`
	SubmittedBefore jsontext.Value `json:"submitted_before"`
}

// Bare reports whether the case expects a bare rejection: a code and no
// label (SPEC.md section 13.2), since a report case may give its code too.
func (e Expected) Bare() bool { return e.Code != "" && e.Label == "" }

// Row is a report row reduced as SPEC.md section 13.1 compares it:
// (code, part, status, step), where a nil Code or Part is null.
type Row struct {
	Code   *string
	Part   *string
	Status string
	Step   string
}

// UnmarshalJSON reads a row from its four-element array.
func (r *Row) UnmarshalJSON(data []byte) error {
	var parts []*string
	if err := json.Unmarshal(data, &parts); err != nil {
		return err
	}
	if len(parts) != 4 || parts[2] == nil || parts[3] == nil {
		return fmt.Errorf("vectorcases: a row is not [code, part, status, step]: %s", data)
	}
	*r = Row{Code: parts[0], Part: parts[1], Status: *parts[2], Step: *parts[3]}
	return nil
}

// String writes the row as cases.json does, for messages.
func (r Row) String() string {
	text := func(s *string) string {
		if s == nil {
			return "null"
		}
		return fmt.Sprintf("%q", *s)
	}
	return fmt.Sprintf("[%s, %s, %q, %q]", text(r.Code), text(r.Part), r.Status, r.Step)
}

// Family is the first hyphen-separated word of a case's id, such as parse
// for parse-depth-33.
func (c Case) Family() string {
	family, _, _ := strings.Cut(c.ID, "-")
	return family
}

// ProofPath is the absolute path of the case's proof.
func (c Case) ProofPath() string {
	return Path(c.Proof)
}

// Dir returns the absolute path of testdata/verification/vectors.
func Dir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("vectorcases: cannot locate source file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "verification", "vectors")
}

// Path joins a path relative to vectors/, as cases.json writes one, under
// [Dir].
func Path(relative string) string {
	return filepath.Join(Dir(), filepath.FromSlash(relative))
}

// Load reads every case of cases.json, in the file's order. A member the
// structs above do not name, a case without an id, proof or environment,
// two cases with one id, and a case's sources that SPEC.md section 13.2
// does not allow ([checkSources]) are errors.
func Load() ([]Case, error) {
	data, err := os.ReadFile(Path("cases.json"))
	if err != nil {
		return nil, err
	}
	return load(data)
}

// load decodes the bytes of a cases.json and checks its cases as [Load]
// says.
func load(data []byte) ([]Case, error) {
	// Unmarshal refuses content after the array, a repeated member name and
	// invalid UTF-8; RejectUnknownMembers refuses a member Case does not name.
	var cases []Case
	if err := json.Unmarshal(data, &cases, json.RejectUnknownMembers(true)); err != nil {
		return nil, fmt.Errorf("vectorcases: cases.json: %w", err)
	}
	seen := make(map[string]bool, len(cases))
	for _, c := range cases {
		if c.ID == "" || c.Proof == "" || c.Config.Environment == "" {
			return nil, fmt.Errorf("vectorcases: a case lacks its id, proof or environment: %+v", c)
		}
		if seen[c.ID] {
			return nil, fmt.Errorf("vectorcases: two cases are named %s", c.ID)
		}
		seen[c.ID] = true
		if err := checkSources(c.Config); err != nil {
			return nil, fmt.Errorf("vectorcases: %s: %w", c.ID, err)
		}
	}
	return cases, nil
}

// checkSources checks a case's sources as SPEC.md section 13.2 writes them:
// present only beside online; each network one of its chain's, a network of
// stellar or bitcoin, the chains specdata.Networks gives; one or more
// names, each an ASCII16, no two the same; the policy first, at_least or
// all; and required present exactly when the policy is at_least, an integer
// from 1 to the number of names. Chains and networks are checked in sorted
// order, so the refusal returned is the first in that order. That each
// network is one the case pins is held by the package's tests over
// cases.json.
func checkSources(cfg Config) error {
	if cfg.Sources == nil {
		return nil
	}
	if cfg.Online == nil {
		return errors.New("sources without online")
	}
	ascii16 := regexp.MustCompile(`^` + specdata.Encodings()["ASCII16"].Pattern + `$`)
	for _, chain := range slices.Sorted(maps.Keys(cfg.Sources)) {
		networks := cfg.Sources[chain]
		for _, network := range slices.Sorted(maps.Keys(networks)) {
			named := networks[network]
			if !slices.Contains(specdata.Networks()[chain], network) {
				return fmt.Errorf("sources of %s %s, not a network of a Stellar or Bitcoin chain", chain, network)
			}
			if len(named.Sources) == 0 {
				return fmt.Errorf("sources of %s %s name no source", chain, network)
			}
			for i, name := range named.Sources {
				if !ascii16.MatchString(name) || slices.Contains(named.Sources[:i], name) {
					return fmt.Errorf("sources of %s %s: the name %q is not an ASCII16 named once", chain, network, name)
				}
			}
			switch named.Policy {
			case PolicyFirst, PolicyAll:
				if named.Required != nil {
					return fmt.Errorf("sources of %s %s: required under %s", chain, network, named.Policy)
				}
			case PolicyAtLeast:
				if named.Required == nil || *named.Required < 1 || *named.Required > len(named.Sources) {
					return fmt.Errorf("sources of %s %s: at_least without a required from 1 to %d", chain, network, len(named.Sources))
				}
			default:
				return fmt.Errorf("sources of %s %s: the policy %q", chain, network, named.Policy)
			}
		}
	}
	return nil
}
