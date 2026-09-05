// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package entropy is a thin client for the Truestamp entropy observations
// JSON:API surface (GET /api/json/entropy_observations and
// /entropy_observations/:id).
//
// An entropy observation is a witness: a public random value Truestamp
// captured from an independent source (a NIST Randomness Beacon pulse, a
// Stellar ledger close, a Bitcoin block) together with the moment it was
// captured. Item submission commits the newest observation per source into
// the item's metadata, which is what opens the submitted-after edge of the
// submission window: the item cannot have been submitted before a value
// that did not yet exist. Each observation is also a proof subject in its
// own right (`proofs get --type entropy_*`), and this package is the
// discovery path to the ids those proofs are asked for by.
//
// Two things the server does not offer, worked around here the way
// internal/blocks does:
//
//   - There is no /latest route: latest is a sort plus a limit of one.
//   - There is no by-hash route: a hash lookup is filter[entropy_hash], and
//     the hex shape is validated before the request is sent.
package entropy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/ids"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
)

// Observation is the public shape of one entropy observation. Entropy is
// the source's own record under its own field names (a Stellar ledger's
// sequence and closed_at, a NIST pulse's index, a Bitcoin block's height)
// and is rendered as published, never renamed. Its numbers are decoded as
// json.Number so a ledger sequence is never re-printed as a float.
type Observation struct {
	ID                string         `json:"id"`
	Source            string         `json:"source"`
	State             string         `json:"state"`
	EntropyHash       string         `json:"entropy_hash"`
	ObservationHash   string         `json:"observation_hash"`
	MetadataHash      string         `json:"metadata_hash"`
	SigningKeyID      string         `json:"signing_key_id"`
	Signature         string         `json:"signature"`
	CaptureMethod     string         `json:"capture_method"`
	SourcePublishedAt string         `json:"source_published_at"`
	InsertedAt        string         `json:"inserted_at"`
	BlockID           string         `json:"block_id,omitempty"`
	Entropy           map[string]any `json:"entropy"`
	Metadata          map[string]any `json:"metadata"`
}

// Sources is the closed set of entropy sources, in the order the CLI
// lists them. They are the wire names, identical to the proof subject
// types `proofs get --type` takes, so one vocabulary names both.
var Sources = []string{"entropy_nist", "entropy_stellar", "entropy_bitcoin"}

// ValidateSource rejects a --source value outside Sources.
func ValidateSource(s string) error {
	for _, ok := range Sources {
		if s == ok {
			return nil
		}
	}
	return fmt.Errorf("--source must be one of %s, got %q", strings.Join(Sources, " | "), s)
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

// ErrAmbiguousHash is returned when a by-hash lookup matches more than one
// row. The server does not assume hash uniqueness, so this is reported
// rather than resolved by picking one.
var ErrAmbiguousHash = errors.New("more than one entropy observation matches that hash")

// ValidateHash rejects anything that is not exactly 64 lowercase hex
// characters, before it can reach filter[entropy_hash].
func ValidateHash(h string) error { return ids.ValidateHash64(h) }

// ValidateUUIDv7 rejects an id that is not a UUIDv7.
func ValidateUUIDv7(id string) error { return ids.ValidateUUIDv7(id) }

// defaultLimit is sent when a caller asks for no particular page size, so
// an unbounded GET never reaches a table that grows by the minute.
const defaultLimit = 25

// List fetches up to limit observations, newest first. An empty source
// means every source; otherwise it must be one of Sources.
func List(ctx context.Context, cfg Config, source string, limit int) ([]Observation, error) {
	if source != "" {
		if err := ValidateSource(source); err != nil {
			return nil, err
		}
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	// No client-side ceiling; the server owns it. See cmd/limits.go.
	q := url.Values{}
	q.Set("sort", "-id")
	q.Set("page[limit]", strconv.Itoa(limit))
	if source != "" {
		q.Set("filter[source]", source)
	}
	body, err := jsonapi.Get(ctx, cfg, "/entropy_observations?"+q.Encode())
	if err != nil {
		return nil, err
	}
	return unmarshalList(body)
}

// Get fetches one observation by UUIDv7 id.
func Get(ctx context.Context, cfg Config, id string) (*Observation, error) {
	if err := ValidateUUIDv7(id); err != nil {
		return nil, err
	}
	body, err := jsonapi.Get(ctx, cfg, "/entropy_observations/"+url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	return unmarshalOne(body)
}

// ByHash fetches one observation by its 64-hex entropy hash. There is no
// by-hash route, so this filters; see the package doc.
func ByHash(ctx context.Context, cfg Config, hash string) (*Observation, error) {
	if err := ValidateHash(hash); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("filter[entropy_hash]", hash)
	q.Set("page[limit]", "2") // 2, so "more than one" is detectable
	body, err := jsonapi.Get(ctx, cfg, "/entropy_observations?"+q.Encode())
	if err != nil {
		return nil, err
	}
	list, err := unmarshalList(body)
	if err != nil {
		return nil, err
	}
	switch len(list) {
	case 0:
		return nil, jsonapi.NotFound("no entropy observation with that hash")
	case 1:
		return &list[0], nil
	default:
		return nil, ErrAmbiguousHash
	}
}

// Latest fetches the newest observation: the most recently captured from
// any source, or from one source when source is set.
func Latest(ctx context.Context, cfg Config, source string) (*Observation, error) {
	list, err := List(ctx, cfg, source, 1)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, jsonapi.NotFound("no entropy observations")
	}
	return &list[0], nil
}

// resource is the JSON:API resource object: identity at the top,
// everything else under attributes.
type resource struct {
	ID         string          `json:"id"`
	Attributes json.RawMessage `json:"attributes"`
}

func fromResource(r resource) (Observation, error) {
	var o Observation
	if len(r.Attributes) > 0 {
		dec := json.NewDecoder(bytes.NewReader(r.Attributes))
		dec.UseNumber()
		if err := dec.Decode(&o); err != nil {
			return Observation{}, fmt.Errorf("parsing entropy observation: %w", err)
		}
	}
	o.ID = r.ID
	return o, nil
}

func unmarshalOne(body []byte) (*Observation, error) {
	var env struct {
		Data resource `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parsing entropy observation response: %w", err)
	}
	if env.Data.ID == "" {
		return nil, fmt.Errorf("API response is not an entropy observation")
	}
	o, err := fromResource(env.Data)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func unmarshalList(body []byte) ([]Observation, error) {
	var env struct {
		Data []resource `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parsing entropy observation list: %w", err)
	}
	out := make([]Observation, 0, len(env.Data))
	for i, r := range env.Data {
		o, err := fromResource(r)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
		out = append(out, o)
	}
	return out, nil
}
