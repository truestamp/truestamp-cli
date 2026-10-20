// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the read SOURCES makes of NIST's Randomness Beacon for a
// NIST entropy object (SPEC.md section 10.4, "Entropy objects", and its
// informative table, "Common sources"), giving its answer in the form of a
// replay record (SPEC.md section 13.2, "Replay files"):
//
//  | Read       | Call                                               | Fields of the answer's value      |
//  |------------|----------------------------------------------------|-----------------------------------|
//  | nist_pulse | GET <base>/chain/<chainIndex>/pulse/<pulseIndex>   | outputValue, timeStamp (of pulse) |
//
// NIST's Randomness Beacon is one source, with no network and no identity
// read (SPEC.md section 9): its base is the verifier's one compiled-in
// address, and no configuration names another, so [Provider.Chain] gives
// [NISTBeacon] no chain and a source configured with it is refused.
//
// A read's answer is one of SPEC.md section 10.4's three outcomes
// ("Answers"):
//
//   - found: a 200 OK whose body is a JSON object holding pulse, an object
//     whose chainIndex and pulseIndex are the ones asked for, so the
//     answer is of the pulse asked for; the value holds pulse's
//     outputValue and timeStamp, each as the pulse holds it and absent when
//     it lacks one. This file judges neither field's class: the rule that
//     reads the answer judges each field it uses (section 10.4: an answer
//     is judged on the fields its rule reads);
//   - not found: a 404, the answer the beacon gives for a pulse it does not
//     hold;
//   - unreachable: every other answer, a 3xx, a 429, a 503, a challenge
//     page, a body over its bound and a pulse of another index among them,
//     and a read whose chainIndex or pulseIndex is not an INT, which is
//     made to no server. Its reason is a fixed text, as the identity
//     reads' are ([fromRead]), quoting nothing the server sent. A pulse that
//     names another index is no answer about the pulse asked for: it is
//     neither that pulse, which is what found needs, nor an answer that the
//     beacon holds no such pulse.
//
// The read's parameters name only the pulse asked for, never a URL: the
// document chooses no source.

package source

import (
	"context"
	"net/http"
	"strconv"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// NISTBeacon is NIST's Randomness Beacon, read over its 2.0 API. It
// serves no chain ([Provider.Chain]).
const NISTBeacon Provider = "nist_beacon"

// KindNISTPulse is the kind of the read of a NIST pulse (SPEC.md section
// 13.2's table).
const KindNISTPulse = "nist_pulse"

// NISTOctets bounds the body of a pulse: the beacon writes one in about
// five kilobytes, its signature and its list values included.
const NISTOctets = 65536

// NISTPulse reads the pulse at chainIndex and pulseIndex of the beacon at
// base, GET <base>/chain/<chainIndex>/pulse/<pulseIndex>, for its
// outputValue and timeStamp.
func NISTPulse(ctx context.Context, reads *httpclient.SingleRead, base string, chainIndex, pulseIndex any) Answer {
	chain, isChain := intParam(chainIndex)
	pulse, isPulse := intParam(pulseIndex)
	if !isChain || !isPulse {
		return unreachableAnswer("the chainIndex or the pulseIndex is not an integer from 0 to 2^53 - 1")
	}
	path := "/chain/" + strconv.FormatInt(chain, 10) + "/pulse/" + strconv.FormatInt(pulse, 10)
	body, err := reads.Get(ctx, endpoint(base, path), NISTOctets)
	if err != nil {
		if httpclient.Status(err) == http.StatusNotFound {
			return Answer{Outcome: OutcomeNotFound}
		}
		return unreachableFrom(err)
	}
	value, err := strictjson.Parse(body)
	object, _ := value.(map[string]any)
	read, isObject := object["pulse"].(map[string]any)
	if err != nil || !isObject {
		return unreachableAnswer("NIST's answer is not a JSON object holding a pulse object")
	}
	gotChain, chainOK := intOf(read["chainIndex"])
	gotPulse, pulseOK := intOf(read["pulseIndex"])
	if !chainOK || !pulseOK || gotChain != chain || gotPulse != pulse {
		return unreachableAnswer("NIST's pulse is not the pulse asked for")
	}
	return found(read, map[string]string{"outputValue": "outputValue", "timeStamp": "timeStamp"})
}
