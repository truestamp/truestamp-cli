// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package beacons is a thin client for the Truestamp Beacons JSON:API
// surface (GET /api/json/beacons, /beacons/latest, /beacons/:id,
// /beacons/by-hash/:hash). A Beacon is a compact "proof of life"
// projection of a finalized block: {id, hash, timestamp, previous_hash}.
// The server side is documented in
// truestamp-v2/kb/api/beacon-api.md.
package beacons

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/truestamp/truestamp-cli/internal/ids"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
)

// Beacon is the JSON shape returned by every beacon endpoint. All four
// fields are always present on success; a missing field is treated as a
// parse error rather than recovered from.
type Beacon struct {
	ID           string `json:"id"`            // UUIDv7
	Hash         string `json:"hash"`          // 64 lowercase hex
	Timestamp    string `json:"timestamp"`     // ISO 8601 UTC
	PreviousHash string `json:"previous_hash"` // 64 lowercase hex
}

// The transport, the class sentinels and APIError live in
// internal/jsonapi; these aliases keep this client's surface stable for
// the commands that errors.Is its classes.
type (
	Config   = jsonapi.Config
	APIError = jsonapi.APIError
)

var (
	ErrUnauthorized = jsonapi.ErrUnauthorized
	ErrForbidden    = jsonapi.ErrForbidden
	ErrNotFound     = jsonapi.ErrNotFound
	ErrBadRequest   = jsonapi.ErrBadRequest
	ErrRateLimited  = jsonapi.ErrRateLimited
	ErrServer       = jsonapi.ErrServer
)

// Latest fetches the most recent finalized/committed beacon.
func Latest(ctx context.Context, cfg Config) (*Beacon, error) {
	body, err := jsonapi.Get(ctx, cfg, "/beacons/latest")
	if err != nil {
		return nil, err
	}
	return unmarshalBeacon(body)
}

// List fetches up to `limit` most-recent beacons, newest first.
//
// The ceiling is the server's to enforce and to name: it refuses an
// over-large page with "must be less than or equal to 100", which surfaces
// cleanly through APIError. Do not pre-clamp here -- the server's OpenAPI
// document states no maximum, so any number written into this client is
// unbacked and drifts silently. cmd/limits.go owns the floor, which the
// contract does state.
func List(ctx context.Context, cfg Config, limit int) ([]Beacon, error) {
	path := "/beacons"
	if limit > 0 {
		path = path + "?limit=" + strconv.Itoa(limit)
	}
	body, err := jsonapi.Get(ctx, cfg, path)
	if err != nil {
		return nil, err
	}
	return unmarshalBeaconList(body)
}

// Get fetches a single beacon by UUIDv7 id.
func Get(ctx context.Context, cfg Config, id string) (*Beacon, error) {
	if err := ValidateUUIDv7(id); err != nil {
		return nil, err
	}
	body, err := jsonapi.Get(ctx, cfg, "/beacons/"+url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	return unmarshalBeacon(body)
}

// ByHash fetches a single beacon by its 64-char lowercase hex hash.
func ByHash(ctx context.Context, cfg Config, hash string) (*Beacon, error) {
	if err := ValidateHash(hash); err != nil {
		return nil, err
	}
	body, err := jsonapi.Get(ctx, cfg, "/beacons/by-hash/"+url.PathEscape(hash))
	if err != nil {
		return nil, err
	}
	return unmarshalBeacon(body)
}

// unmarshalBeacon handles both bare-object and {"result": …} envelopes.
func unmarshalBeacon(body []byte) (*Beacon, error) {
	unwrapped := unwrap(body)
	var b Beacon
	if err := json.Unmarshal(unwrapped, &b); err != nil {
		return nil, fmt.Errorf("parsing beacon response: %w", err)
	}
	if err := validateShape(&b); err != nil {
		return nil, err
	}
	return &b, nil
}

// unmarshalBeaconList handles both bare-array and {"result": [...]} envelopes.
func unmarshalBeaconList(body []byte) ([]Beacon, error) {
	unwrapped := unwrap(body)
	var list []Beacon
	if err := json.Unmarshal(unwrapped, &list); err != nil {
		return nil, fmt.Errorf("parsing beacon list response: %w", err)
	}
	for i := range list {
		if err := validateShape(&list[i]); err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
	}
	return list, nil
}

// unwrap returns the inner payload if body is {"result": <x>}, else body.
func unwrap(body []byte) []byte {
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Result) > 0 {
		return envelope.Result
	}
	return body
}

func validateShape(b *Beacon) error {
	if err := ValidateUUIDv7(b.ID); err != nil {
		return fmt.Errorf("invalid beacon id: %w", err)
	}
	if err := ValidateHash(b.Hash); err != nil {
		return fmt.Errorf("invalid beacon hash: %w", err)
	}
	if err := ValidateHash(b.PreviousHash); err != nil {
		return fmt.Errorf("invalid previous_hash: %w", err)
	}
	if b.Timestamp == "" {
		return fmt.Errorf("missing timestamp")
	}
	return nil
}

// ValidateHash returns nil iff s is exactly 64 lowercase hex characters.
func ValidateHash(s string) error { return ids.ValidateHash64(s) }

// ValidateUUIDv7 returns nil iff s parses as a UUID whose version is 7.
func ValidateUUIDv7(s string) error { return ids.ValidateUUIDv7(s) }
