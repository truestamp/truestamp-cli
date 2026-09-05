// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
	"net/http"
	"strings"
)

// ResolveSubjectType asks the server what a bare id refers to, so
// `proofs get <id>` does not have to require --type.
//
// This is the one place id-shape dispatch cannot work. A ULID is
// unambiguously an item, but blocks, beacons and entropy observations all
// use UUIDv7, so nothing client-side can tell them apart. The server can:
// POST /utilities/resolve-id returns each match's `proof_type`, which is
// exactly the value --type takes.
//
// It is deliberately a fallback, not the default path. Passing --type is
// one fewer round trip, and it keeps an id-classification service out of
// the loop for a tool whose whole thesis is not taking the server's word
// for things. The classification only decides which proof to ASK for; the
// bundle that comes back is still verified against its own signed type,
// and `verify --type` still asserts independently.
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("could not resolve %s (HTTP %d): pass --type explicitly", id, resp.StatusCode)
	}

	var env struct {
		Result struct {
			Matches []struct {
				Kind      string  `json:"kind"`
				ProofType *string `json:"proof_type"`
			} `json:"matches"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return "", fmt.Errorf("parsing resolve-id response: %w", err)
	}

	var types []string
	for _, m := range env.Result.Matches {
		if m.ProofType != nil && *m.ProofType != "" {
			types = append(types, *m.ProofType)
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
		// A block is also verifiable as a beacon, so more than one match is
		// legitimate. Refuse rather than pick: which one the caller meant
		// changes what the proof commits to.
		return "", fmt.Errorf(
			"%s is verifiable as more than one subject type (%s); pass --type to say which you want",
			id, strings.Join(types, ", "))
	}
}
