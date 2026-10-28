// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package proof

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/jsonapi"
)

// ErrSeveralTypes is a subject the server says is verifiable as more
// than one subject type: [ResolveSubjectType] refuses rather than picks.
var ErrSeveralTypes = errors.New("verifiable as more than one subject type")

// ResolveSubjectType asks the server what a bare UUIDv7 refers to, so
// `proofs get <id>` does not have to require --type.
//
// This is the one place id-shape dispatch cannot work. Items, blocks,
// beacons and entropy observations all use UUIDv7, so nothing client-side
// can tell them apart. The server can: POST /utilities/resolve-id returns
// one match for a UUIDv7 it can show, the first of an item, a block, an
// entropy observation and a team, with its `proof_type`, exactly the
// value --type takes, and `also_verifiable_as`, the other types the same
// subject is verifiable as: a block's is ["beacon"], since a block is
// also verifiable as a beacon. id is sent as the caller gives it, which
// cmd/proofs_get.go makes the UUIDv7's canonical lowercase text.
//
// The types a match names, its proof_type and then each of its
// also_verifiable_as, are the answer: one is the type returned, and more
// than one, a block's, is refused with [ErrSeveralTypes], since which
// proof the caller meant changes what the proof commits to: a block id
// needs --type block or --type beacon. A match with no proof_type, a
// team, names none.
//
// It is deliberately a fallback, not the default path. Passing --type is
// one fewer round trip, and it keeps an id-classification service out of
// the loop for a tool whose whole thesis is not taking the server's word
// for things. The classification only decides which proof to ASK for; the
// proof that comes back is still verified against its own signed type.
//
// Resolution is visibility-gated server-side: an id you cannot see
// returns no match rather than revealing that it exists.
func ResolveSubjectType(ctx context.Context, apiURL, team, id string) (string, error) {
	// AshJsonApi generic routes take their arguments under `data`, not at
	// the top level: a flat `{"id": …}` is refused with
	// `Required … source.pointer /data`.
	payload, err := json.Marshal(map[string]any{"data": map[string]any{"id": id}})
	if err != nil {
		return "", fmt.Errorf("encoding request: %w", err)
	}
	resp, body, err := jsonapi.DoRaw(ctx, jsonapi.Config{APIURL: apiURL, Team: team},
		http.MethodPost, "/utilities/resolve-id", payload)
	if err != nil {
		return "", fmt.Errorf("resolving id: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		// A rate limit is not an unresolvable id: carry the classified
		// refusal so the command can say "wait", not "pass --type".
		return "", fmt.Errorf("resolving id: %w", jsonapi.ErrorFromResponse(resp, body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("could not resolve %s (HTTP %d): pass --type explicitly", id, resp.StatusCode)
	}

	var env struct {
		Result struct {
			Matches []struct {
				Kind             string   `json:"kind"`
				ProofType        *string  `json:"proof_type"`
				AlsoVerifiableAs []string `json:"also_verifiable_as"`
			} `json:"matches"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return "", fmt.Errorf("parsing resolve-id response: %w", err)
	}

	var types []string
	for _, m := range env.Result.Matches {
		if m.ProofType == nil || *m.ProofType == "" {
			continue
		}
		for _, typ := range append([]string{*m.ProofType}, m.AlsoVerifiableAs...) {
			if typ != "" && !slices.Contains(types, typ) {
				types = append(types, typ)
			}
		}
	}
	switch len(types) {
	case 0:
		return "", fmt.Errorf(
			"%s does not resolve to anything you can fetch a proof for; pass --type explicitly if you believe it should",
			id)
	case 1:
		return types[0], nil
	default:
		return "", fmt.Errorf("%s is %w, %s, which is refused rather than guessed: pass --type %s",
			id, ErrSeveralTypes, strings.Join(types, " | "), strings.Join(types, " or --type "))
	}
}
