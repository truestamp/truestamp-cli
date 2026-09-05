// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package items

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/truestamp/truestamp-cli/internal/jsonapi"
)

// Item is the subset of an item's attributes the CLI reads back.
//
// Three of these fields — InsertedAt, UpdatedAt, ExpiresAt — are NOT in
// the resource's json_api default_fields, so every request that wants
// them must ask for them explicitly via fields[item]. See requestFields.
type Item struct {
	ID          string         `json:"id"`
	State       string         `json:"state"`
	Claims      map[string]any `json:"claims,omitempty"`
	ClaimsHash  string         `json:"claims_hash,omitempty"`
	ItemHash    string         `json:"item_hash,omitempty"`
	Visibility  string         `json:"visibility,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	TeamID      string         `json:"team_id,omitempty"`
	DisplayName string         `json:"display_name,omitempty"`
	InsertedAt  string         `json:"inserted_at,omitempty"`
	UpdatedAt   string         `json:"updated_at,omitempty"`
	ExpiresAt   string         `json:"expires_at,omitempty"`
}

// Committed reports whether a proof can be generated for this item.
// Proof generation hard-requires the committed state server-side, so this
// is the difference between `proofs get <id>` working and being refused
// with no_external_commitments.
func (i Item) Committed() bool { return i.State == "committed" }

// Page is one page of a list response plus the cursor for the next.
type Page struct {
	Items []Item
	// NextCursor is empty when there are no more pages.
	NextCursor string
	// Total is the server's count of matching items, only when asked for.
	Total int
}

// DefaultLimit matches the server's own default for the paginated read.
const DefaultLimit = 25

// requestFields asks for the attributes the CLI renders. inserted_at,
// updated_at and expires_at are absent from the resource's
// json_api default_fields, so omitting this leaves them empty with no
// error — a silent hole rather than a failure.
const requestFields = "claims,claims_hash,item_hash,visibility,state,tags,team_id,display_name,inserted_at,updated_at,expires_at"

// ListOptions configures a list request.
type ListOptions struct {
	Limit int
	// After is a keyset cursor from a previous Page.NextCursor.
	After string
	// Count asks the server for the total, reported as Page.Total.
	Count bool
	// Committed and Pending filter on commitment state. Both false means
	// no filter; both true is rejected by the caller.
	Committed bool
	Pending   bool
}

// List fetches one page of items, newest first.
func List(ctx context.Context, apiURL, team string, opts ListOptions) (*Page, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	// No client-side ceiling. The server's OpenAPI document declares
	// page.limit with "minimum": 1 and no maximum at all, so a constant
	// here would be an unbacked second source of truth; the server refuses
	// an over-large page and names its own cap. See cmd/limits.go.

	q := url.Values{}
	jsonapi.SetPageQuery(q, limit, opts.After, opts.Count)
	q.Set("fields[item]", requestFields)
	// Newest first, which is what this command's help promises and what
	// `blocks list` and `beacons list` already do. Without it the server
	// applies its default ascending order and `items list` was the one
	// list verb in the tree that answered oldest-first.
	//
	// Sent on every page, not just the first. The server does echo `sort`
	// back in the `next` link, but this client never follows that link: it
	// rebuilds the query itself and lifts only the cursor out, so the sort
	// has to be re-supplied here or page two would silently revert.
	q.Set("sort", "-id")
	switch {
	case opts.Committed && opts.Pending:
		return nil, fmt.Errorf("--committed and --pending are mutually exclusive")
	case opts.Committed:
		q.Set("filter[state]", "committed")
	case opts.Pending:
		// "Pending" is every state that is not yet committed. The server
		// has no such filter, so this is a client-side exclusion after the
		// fetch rather than a query the server can index.
	}

	body, err := jsonapi.Do(ctx, jsonapi.Config{APIURL: apiURL, Team: team}, http.MethodGet, "/items?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	page, err := parseList(body)
	if err != nil {
		return nil, err
	}
	if opts.Pending {
		kept := page.Items[:0]
		for _, it := range page.Items {
			if !it.Committed() {
				kept = append(kept, it)
			}
		}
		page.Items = kept
	}
	return page, nil
}

// Get fetches one item by ULID.
func Get(ctx context.Context, apiURL, team, id string) (*Item, error) {
	q := url.Values{}
	q.Set("fields[item]", requestFields)
	body, err := jsonapi.Do(ctx, jsonapi.Config{APIURL: apiURL, Team: team}, http.MethodGet, "/items/"+url.PathEscape(id)+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	return parseOne(body)
}

// UpdateOptions carries the mutable attributes.
//
// The server's :update action is `accept [:team_id, :visibility, :tags]`,
// described there as "considered mutable and not included in the item's
// hash". name and description live inside claims and are immutable, so
// there is deliberately no way to reach a signed field from here.
type UpdateOptions struct {
	Visibility *string
	Tags       *[]string
	TeamID     *string
}

// Empty reports whether the caller asked for no change at all.
func (o UpdateOptions) Empty() bool {
	return o.Visibility == nil && o.Tags == nil && o.TeamID == nil
}

// Update patches an item's mutable attributes.
func Update(ctx context.Context, apiURL, team, id string, opts UpdateOptions) (*Item, error) {
	if opts.Empty() {
		return nil, fmt.Errorf("nothing to update: pass --visibility, --tags or --to-team")
	}
	attrs := map[string]any{}
	if opts.Visibility != nil {
		attrs["visibility"] = *opts.Visibility
	}
	if opts.Tags != nil {
		attrs["tags"] = *opts.Tags
	}
	if opts.TeamID != nil {
		attrs["team_id"] = *opts.TeamID
	}
	payload := map[string]any{
		"data": map[string]any{
			"type":       "item",
			"id":         id,
			"attributes": attrs,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}
	body, err := jsonapi.Do(ctx, jsonapi.Config{APIURL: apiURL, Team: team}, http.MethodPatch, "/items/"+url.PathEscape(id), raw)
	if err != nil {
		return nil, err
	}
	return parseOne(body)
}

// resourceObject is the JSON:API shape both the single and list responses
// use: identity at the top, everything else under attributes.
type resourceObject struct {
	ID         string          `json:"id"`
	Attributes json.RawMessage `json:"attributes"`
}

func itemFromResource(r resourceObject) (Item, error) {
	var it Item
	if len(r.Attributes) > 0 {
		if err := json.Unmarshal(r.Attributes, &it); err != nil {
			return Item{}, fmt.Errorf("parsing item attributes: %w", err)
		}
	}
	it.ID = r.ID
	return it, nil
}

func parseOne(body []byte) (*Item, error) {
	var env struct {
		Data resourceObject `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parsing item: %w", err)
	}
	if env.Data.ID == "" {
		return nil, fmt.Errorf("API response is not an item")
	}
	it, err := itemFromResource(env.Data)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func parseList(body []byte) (*Page, error) {
	var env struct {
		Data []resourceObject `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parsing item list: %w", err)
	}
	page := &Page{Items: make([]Item, 0, len(env.Data))}
	for _, r := range env.Data {
		it, err := itemFromResource(r)
		if err != nil {
			return nil, err
		}
		page.Items = append(page.Items, it)
	}
	info := jsonapi.ParsePage(body)
	page.NextCursor, page.Total = info.NextCursor, info.Total
	return page, nil
}
