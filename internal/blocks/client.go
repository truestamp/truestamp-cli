// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package blocks is a thin client for the Truestamp Blocks JSON:API
// surface (GET /api/json/blocks, /blocks/:id).
//
// A block is the full signed record: Merkle root, state, signature, key
// id and chain links. The same block projected to four public fields is
// a *beacon*, served by [internal/beacons]. They are two views of one
// row, and they are separate here for the same reason they are separate
// commands: only finalized or committed blocks project as beacons, so
// "the head block" and "the most recent beacon" are different questions
// most of the time: the chain advances about once a minute and the head
// is routinely not yet finalized.
//
// Two things the server does not offer, and that this package therefore
// works around rather than assumes:
//
//   - There is no /blocks/latest or /blocks/genesis route, though the Ash
//     actions exist. Both are a sort plus a limit of one.
//   - There is no by-hash route. Addressing by hash goes through
//     filter[block_hash], which, unlike the beacons by-hash action, has
//     NO server-side shape guard, and an unguarded cast raises a
//     Ecto.Query.CastError, a 500 that leaks SQL. So the hex shape is
//     validated here, before the request is sent.
package blocks

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/url"

	"github.com/truestamp/truestamp-cli/internal/ids"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
)

// Block is the subset of a block's public attributes the CLI renders.
// The server pins its twelve public attributes with a test; these are the
// ones that identify a block and place it in the chain.
//
// Height is the server's stored height, the block's position in the
// chain: 0 at genesis, then one more per block. It orders the chain, so
// the lists and the head and genesis reads sort by it, never by id: a
// UUIDv7's time is when its id was minted, and the server orders blocks
// by height whatever their ids. A block is addressed by its UUIDv7.
type Block struct {
	ID                string `json:"id"`
	Height            int64  `json:"height"`
	BlockHash         string `json:"block_hash"`
	MerkleRoot        string `json:"merkle_root"`
	State             string `json:"state"`
	PreviousBlockID   string `json:"previous_block_id"`
	PreviousBlockHash string `json:"previous_block_hash"`
	SigningKeyID      string `json:"signing_key_id"`
	Signature         string `json:"signature"`
	InsertedAt        string `json:"inserted_at"`
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

// ErrAmbiguousHash is returned when a by-hash lookup matches more than
// one row. The server does not assume block-hash uniqueness, so this is
// reported rather than resolved by picking one.
var ErrAmbiguousHash = errors.New("more than one block matches that hash")

// ValidateHash rejects anything that is not exactly 64 lowercase hex
// characters, before it can reach filter[block_hash].
func ValidateHash(h string) error { return ids.ValidateHash64(h) }

// ValidateUUIDv7 rejects an id that is not a UUIDv7.
func ValidateUUIDv7(id string) error { return ids.ValidateUUIDv7(id) }

// defaultLimit is the page size List asks for when the caller gives
// none. The blocks index route's action requires pagination, defaults to
// 25 and clamps to 250, so a request without a limit is still bounded;
// sending one keeps the page size explicit.
const defaultLimit = 25

// ListOptions configures a list request; the paging fields are the ones
// every keyset-paged list shares (jsonapi.SetPageQuery). The list is in
// height order, highest first unless OldestFirst.
type ListOptions struct {
	Limit       int
	After       string // continue forward from a Page.NextCursor
	Before      string // continue backward from a Page.PrevCursor
	OldestFirst bool   // walk from the beginning instead of the newest row
	Count       bool
}

// Page is one page of blocks plus the cursor for the next and, when asked
// for, the server's total.
type Page struct {
	Blocks     []Block
	NextCursor string
	PrevCursor string
	Total      int
	Limit      int // the page size the server actually used
}

// The sorts the blocks route is asked for: by height, highest first or
// lowest first.
const (
	sortHighestFirst = "-height"
	sortLowestFirst  = "height"
)

// List fetches one page of blocks, highest first.
func List(ctx context.Context, cfg Config, opts ListOptions) (*Page, error) {
	if opts.Limit <= 0 {
		opts.Limit = defaultLimit
	}
	// No client-side ceiling; the server owns it. See cmd/limits.go.
	q := url.Values{}
	sort := sortHighestFirst
	if opts.OldestFirst {
		sort = sortLowestFirst
	}
	q.Set("sort", sort)
	jsonapi.SetPageQuery(q, opts.Limit, opts.After, opts.Before, opts.Count)
	body, err := jsonapi.Get(ctx, cfg, "/blocks?"+q.Encode())
	if err != nil {
		return nil, err
	}
	list, err := unmarshalList(body)
	if err != nil {
		return nil, err
	}
	info := jsonapi.ParsePage(body)
	return &Page{Blocks: list, NextCursor: info.NextCursor, PrevCursor: info.PrevCursor, Total: info.Total, Limit: info.Limit}, nil
}

// Get fetches one block by UUIDv7 id.
func Get(ctx context.Context, cfg Config, id string) (*Block, error) {
	if err := ValidateUUIDv7(id); err != nil {
		return nil, err
	}
	body, err := jsonapi.Get(ctx, cfg, "/blocks/"+url.PathEscape(id))
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
	body, err := jsonapi.Get(ctx, cfg, "/blocks?"+q.Encode())
	if err != nil {
		return nil, err
	}
	list, err := unmarshalList(body)
	if err != nil {
		return nil, err
	}
	switch len(list) {
	case 0:
		return nil, jsonapi.NotFound("no block with that hash")
	case 1:
		return &list[0], nil
	default:
		return nil, ErrAmbiguousHash
	}
}

// Latest fetches the head block: the block with the greatest height,
// whatever its state. This is NOT the same as the most recent beacon,
// which is the newest *finalized* block.
func Latest(ctx context.Context, cfg Config) (*Block, error) {
	return firstOf(ctx, cfg, sortHighestFirst)
}

// Genesis fetches the first block, at height 0, the trust root every
// chain walk terminates at. It is identifiable by id == previous_block_id.
func Genesis(ctx context.Context, cfg Config) (*Block, error) {
	b, err := firstOf(ctx, cfg, sortLowestFirst)
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

func firstOf(ctx context.Context, cfg Config, sort string) (*Block, error) {
	q := url.Values{}
	q.Set("sort", sort)
	q.Set("page[limit]", "1")
	body, err := jsonapi.Get(ctx, cfg, "/blocks?"+q.Encode())
	if err != nil {
		return nil, err
	}
	list, err := unmarshalList(body)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, jsonapi.NotFound("no blocks")
	}
	return &list[0], nil
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
		Data jsontext.Value `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Data) == 0 {
		return body
	}
	return flattenAttributes(envelope.Data)
}

// flattenAttributes merges a JSON:API resource object's `attributes` up
// into the object itself, so `{"id":…, "attributes":{"state":…}}` decodes
// into Block. Non-resource shapes pass through untouched.
func flattenAttributes(raw jsontext.Value) jsontext.Value {
	// Try a list first.
	var items []jsontext.Value
	if err := json.Unmarshal(raw, &items); err == nil {
		out := make([]jsontext.Value, 0, len(items))
		for _, it := range items {
			out = append(out, flattenOne(it))
		}
		merged, err := json.Marshal(out, json.Deterministic(true))
		if err != nil {
			return raw
		}
		return merged
	}
	return flattenOne(raw)
}

func flattenOne(raw jsontext.Value) jsontext.Value {
	var obj map[string]jsontext.Value
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw
	}
	attrs, ok := obj["attributes"]
	if !ok {
		return raw
	}
	var inner map[string]jsontext.Value
	if err := json.Unmarshal(attrs, &inner); err != nil {
		return raw
	}
	for k, v := range inner {
		if _, exists := obj[k]; !exists {
			obj[k] = v
		}
	}
	delete(obj, "attributes")
	merged, err := json.Marshal(obj, json.Deterministic(true))
	if err != nil {
		return raw
	}
	return merged
}
