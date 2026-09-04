// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package blocks is a thin client for the Truestamp Blocks JSON:API
// surface (GET /api/json/blocks, /blocks/:id).
//
// A block is the full signed record: Merkle root, state, signature, key
// id and chain links. The same block projected to four public fields is
// a *beacon*, served by [internal/beacons]. They are two views of one
// row, and they are separate here for the same reason they are separate
// commands: only finalized or committed blocks project as beacons, so
// "the head block" and "the most recent beacon" are different questions
// most of the time — the chain advances about once a minute and the head
// is routinely not yet finalized.
//
// Two things the server does not offer, and that this package therefore
// works around rather than assumes:
//
//   - There is no /blocks/latest or /blocks/genesis route, though the Ash
//     actions exist. Both are a sort plus a limit of one.
//   - There is no by-hash route. Addressing by hash goes through
//     filter[block_hash], which — unlike the beacons by-hash action — has
//     NO server-side shape guard, and an unguarded cast raises a
//     Ecto.Query.CastError, a 500 that leaks SQL. So the hex shape is
//     validated here, before the request is sent.
package blocks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/gofrs/uuid/v5"
	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// Block is the subset of a block's public attributes the CLI renders.
// The server pins exactly eleven public attributes with a test; these are
// the ones that identify a block and place it in the chain.
//
// There is deliberately no Height field. block_height is a runtime
// COUNT(*) calculation, explicitly non-sortable and non-filterable, and
// withheld from the wire. A block is addressed by UUIDv7, which is also
// its ordering handle.
type Block struct {
	ID                string `json:"id"`
	BlockHash         string `json:"block_hash"`
	MerkleRoot        string `json:"merkle_root"`
	State             string `json:"state"`
	PreviousBlockID   string `json:"previous_block_id"`
	PreviousBlockHash string `json:"previous_block_hash"`
	SigningKeyID      string `json:"signing_key_id"`
	Signature         string `json:"signature"`
	InsertedAt        string `json:"inserted_at"`
}

// Errors surfaced by the client, mirroring internal/beacons so CLI layers
// can treat the two the same way.
var (
	ErrUnauthorized = errors.New("not authenticated")
	ErrNotFound     = errors.New("block not found")
	ErrBadRequest   = errors.New("bad request")
	ErrRateLimited  = errors.New("rate limited")
	ErrServer       = errors.New("server error")
	// ErrAmbiguousHash is returned when a by-hash lookup matches more than
	// one row. The server does not assume block-hash uniqueness, so this
	// is reported rather than resolved by picking one.
	ErrAmbiguousHash = errors.New("more than one block matches that hash")
)

// APIError carries HTTP status plus the preserved JSON:API `errors[].detail`.
type APIError struct {
	Status     int
	Detail     string
	RetryAfter string
	sentinel   error
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("API error (HTTP %d): %s", e.Status, e.Detail)
	}
	return fmt.Sprintf("API error (HTTP %d)", e.Status)
}

func (e *APIError) Unwrap() error { return e.sentinel }

// Config carries the subset of runtime configuration a request needs. The
// credential is supplied out of band by the process-wide auth.Authorizer.
type Config struct {
	APIURL string
	Team   string
}

// hashRe is the client-side guard standing in for the server-side one the
// blocks filter does not have.
var hashRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidateHash rejects anything that is not exactly 64 lowercase hex
// characters, before it can reach filter[block_hash].
func ValidateHash(h string) error {
	if !hashRe.MatchString(h) {
		return fmt.Errorf("block hash must be exactly 64 lowercase hex characters, got %q", h)
	}
	return nil
}

// ValidateUUIDv7 rejects an id that is not a UUIDv7.
func ValidateUUIDv7(id string) error {
	u, err := uuid.FromString(id)
	if err != nil {
		return fmt.Errorf("block id must be a UUIDv7, got %q", id)
	}
	if u.Version() != 7 {
		return fmt.Errorf("block id must be a UUIDv7, got a UUIDv%d", u.Version())
	}
	return nil
}

// defaultLimit matches the beacons convention. The server's block :read
// action declares no default_limit and no max_page_size, against a table
// growing about 1,440 rows a day, so an unbounded GET /blocks is a real
// hazard: always send one.
const defaultLimit = 25

// List fetches up to limit blocks, newest first.
func List(ctx context.Context, cfg Config, limit int) ([]Block, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	// No client-side ceiling; the server owns it. See cmd/limits.go.
	q := url.Values{}
	q.Set("sort", "-id")
	q.Set("page[limit]", strconv.Itoa(limit))
	body, err := doGet(ctx, cfg, "/blocks?"+q.Encode())
	if err != nil {
		return nil, err
	}
	return unmarshalList(body)
}

// Get fetches one block by UUIDv7 id.
func Get(ctx context.Context, cfg Config, id string) (*Block, error) {
	if err := ValidateUUIDv7(id); err != nil {
		return nil, err
	}
	body, err := doGet(ctx, cfg, "/blocks/"+url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	return unmarshalOne(body)
}

// ByHash fetches one block by its 64-hex block hash. There is no by-hash
// route, so this filters; see the package doc for why the shape is
// validated here first.
func ByHash(ctx context.Context, cfg Config, hash string) (*Block, error) {
	if err := ValidateHash(hash); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("filter[block_hash]", hash)
	q.Set("page[limit]", "2") // 2, so "more than one" is detectable
	body, err := doGet(ctx, cfg, "/blocks?"+q.Encode())
	if err != nil {
		return nil, err
	}
	list, err := unmarshalList(body)
	if err != nil {
		return nil, err
	}
	switch len(list) {
	case 0:
		return nil, &APIError{Status: 404, Detail: "no block with that hash", sentinel: ErrNotFound}
	case 1:
		return &list[0], nil
	default:
		return nil, ErrAmbiguousHash
	}
}

// Latest fetches the head block: the newest row, whatever its state. This
// is NOT the same as the most recent beacon, which is the newest
// *finalized* block.
func Latest(ctx context.Context, cfg Config) (*Block, error) {
	return firstOf(ctx, cfg, "-id", ErrNotFound)
}

// Genesis fetches the first block, the trust root every chain walk
// terminates at. It is identifiable by id == previous_block_id.
func Genesis(ctx context.Context, cfg Config) (*Block, error) {
	b, err := firstOf(ctx, cfg, "id", ErrNotFound)
	if err != nil {
		return nil, err
	}
	if b.PreviousBlockID != "" && b.PreviousBlockID != b.ID {
		return nil, fmt.Errorf(
			"oldest block %s does not look like genesis (previous_block_id %s); the chain may be partially visible to this credential",
			b.ID, b.PreviousBlockID)
	}
	return b, nil
}

func firstOf(ctx context.Context, cfg Config, sort string, notFound error) (*Block, error) {
	q := url.Values{}
	q.Set("sort", sort)
	q.Set("page[limit]", "1")
	body, err := doGet(ctx, cfg, "/blocks?"+q.Encode())
	if err != nil {
		return nil, err
	}
	list, err := unmarshalList(body)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, &APIError{Status: 404, Detail: "no blocks", sentinel: notFound}
	}
	return &list[0], nil
}

func doGet(ctx context.Context, cfg Config, path string) ([]byte, error) {
	if auth.Default().Mode() == auth.ModeNone {
		return nil, &APIError{Status: 401, Detail: "not authenticated", sentinel: ErrUnauthorized}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.APIURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	if err := auth.AuthorizeRequest(ctx, req); err != nil {
		return nil, &APIError{Status: 401, Detail: err.Error(), sentinel: ErrUnauthorized}
	}
	if cfg.Team != "" {
		req.Header.Set("tenant", cfg.Team)
	}

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading API response: %w", err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}
	apiErr := parseAPIError(resp.StatusCode, body)
	if resp.StatusCode == http.StatusTooManyRequests {
		apiErr.RetryAfter = resp.Header.Get("Retry-After")
	}
	return nil, apiErr
}

func parseAPIError(status int, body []byte) *APIError {
	e := &APIError{Status: status, sentinel: sentinelFor(status)}
	var envelope struct {
		Errors []struct {
			Detail string `json:"detail"`
			Title  string `json:"title"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		e.Detail = envelope.Errors[0].Detail
		if e.Detail == "" {
			e.Detail = envelope.Errors[0].Title
		}
	}
	return e
}

func sentinelFor(status int) error {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return ErrUnauthorized
	case status == http.StatusNotFound:
		return ErrNotFound
	case status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status >= 400 && status < 500:
		return ErrBadRequest
	default:
		return ErrServer
	}
}

// unmarshalOne accepts both a bare object and a JSON:API `{"data": {...}}`
// envelope, matching how the beacons client is tolerant of both.
func unmarshalOne(body []byte) (*Block, error) {
	var b Block
	if err := json.Unmarshal(unwrap(body), &b); err != nil {
		return nil, fmt.Errorf("parsing block: %w", err)
	}
	if b.ID == "" {
		return nil, fmt.Errorf("API response is not a block")
	}
	return &b, nil
}

func unmarshalList(body []byte) ([]Block, error) {
	var list []Block
	if err := json.Unmarshal(unwrap(body), &list); err != nil {
		return nil, fmt.Errorf("parsing block list: %w", err)
	}
	return list, nil
}

// unwrap peels a JSON:API `{"data": …}` envelope when present, and also
// flattens `{"data": {"attributes": {...}, "id": "..."}}` into a single
// object, because the two shapes differ across this API's routes.
func unwrap(body []byte) []byte {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Data) == 0 {
		return body
	}
	return flattenAttributes(envelope.Data)
}

// flattenAttributes merges a JSON:API resource object's `attributes` up
// into the object itself, so `{"id":…, "attributes":{"state":…}}` decodes
// into Block. Non-resource shapes pass through untouched.
func flattenAttributes(raw json.RawMessage) json.RawMessage {
	// Try a list first.
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err == nil {
		out := make([]json.RawMessage, 0, len(items))
		for _, it := range items {
			out = append(out, flattenOne(it))
		}
		merged, err := json.Marshal(out)
		if err != nil {
			return raw
		}
		return merged
	}
	return flattenOne(raw)
}

func flattenOne(raw json.RawMessage) json.RawMessage {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw
	}
	attrs, ok := obj["attributes"]
	if !ok {
		return raw
	}
	var inner map[string]json.RawMessage
	if err := json.Unmarshal(attrs, &inner); err != nil {
		return raw
	}
	for k, v := range inner {
		if _, exists := obj[k]; !exists {
			obj[k] = v
		}
	}
	delete(obj, "attributes")
	merged, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return merged
}
