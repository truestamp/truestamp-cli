// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"encoding/json"
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// NISTBeaconURL is the NIST Randomness Beacon v2.0 base URL. Exposed as a
// var so tests can redirect it; do not mutate in production code.
var NISTBeaconURL = "https://beacon.nist.gov/beacon/2.0"

// NISTPulse holds the minimum set of NIST Beacon pulse fields the CLI
// byte-compares against entropy subject data. Signature-chain verification
// is not performed: the Truestamp service stores only these five fields,
// so the entropy subject hash it signed is already over the same slice of
// data we compare here.
type NISTPulse struct {
	ChainIndex  int    `json:"chainIndex"`
	PulseIndex  int    `json:"pulseIndex"`
	OutputValue string `json:"outputValue"`
	TimeStamp   string `json:"timeStamp"`
	Version     string `json:"version"`
}

// pulseEnvelope matches the shape NIST returns: a top-level object with a
// single "pulse" key wrapping the pulse map.
type pulseEnvelope struct {
	Pulse NISTPulse `json:"pulse"`
}

// GetNISTPulse fetches a specific pulse from the NIST Randomness Beacon
// v2.0 API at /chain/{chainIndex}/pulse/{pulseIndex}.
//
// NIST Beacon reliability caveat (per truestamp-v2 docs): during US
// federal shutdowns the beacon may stop publishing new pulses and repeat
// the last pulse indefinitely. Callers that care about freshness should
// apply their own staleness policy on top of this fetch.
// Errors are typed for [Classify] so the caller can tell an unusable
// bundle field from an unreachable beacon.
func GetNISTPulse(chainIndex, pulseIndex int) (*NISTPulse, error) {
	if chainIndex < 0 {
		return nil, &BadInputError{Field: "chainIndex", Detail: fmt.Sprintf("%d is not a valid chain index", chainIndex)}
	}
	if pulseIndex < 0 {
		return nil, &BadInputError{Field: "pulseIndex", Detail: fmt.Sprintf("%d is not a valid pulse index", pulseIndex)}
	}
	url := fmt.Sprintf("%s/chain/%d/pulse/%d", NISTBeaconURL, chainIndex, pulseIndex)
	body, err := httpclient.GetJSON(url)
	if err != nil {
		return nil, fmt.Errorf("fetching NIST pulse: %w", err)
	}
	var env pulseEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, &MalformedResponseError{Source: "NIST Beacon", Detail: "pulse response is not valid JSON", Err: err}
	}
	if env.Pulse.OutputValue == "" {
		return nil, &MalformedResponseError{Source: "NIST Beacon", Detail: "pulse response carries no outputValue"}
	}
	// The caller byte-compares outputValue and timeStamp against the
	// entropy subject (E.21), so an answer about a different pulse would
	// be reported as a value mismatch and fail a sound proof. The beacon
	// echoes both indices in every pulse; a disagreement means the
	// response is not about the pulse that was requested, which is an
	// upstream property and must skip, never fail.
	if env.Pulse.ChainIndex != chainIndex || env.Pulse.PulseIndex != pulseIndex {
		return nil, &MalformedResponseError{
			Source: "NIST Beacon",
			Detail: fmt.Sprintf("pulse response describes chain %d pulse %d, not the requested chain %d pulse %d",
				env.Pulse.ChainIndex, env.Pulse.PulseIndex, chainIndex, pulseIndex),
		}
	}
	return &env.Pulse, nil
}
