// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package beacons is a thin client for the Truestamp Beacons JSON:API
// surface (GET /api/json/beacons, /beacons/latest, /beacons/:id,
// /beacons/by-hash/:hash). A Beacon is a compact "proof of life"
// projection of a finalized block: {id, hash, timestamp, previous_hash}.
// The server side is documented in
// truestamp-v2/docs/BEACONS_API_IMPLEMENTERS_GUIDE.md.
package beacons

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
	"github.com/truestamp/truestamp-cli/internal/httpclient"
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

// Errors surfaced by the client. CLI layers may errors.Is these to pick
// an exit code and user-facing message.
var (
	ErrUnauthorized = errors.New("not authenticated")
	ErrNotFound     = errors.New("beacon not found")
	ErrBadRequest   = errors.New("bad request")
	ErrRateLimited  = errors.New("rate limited")
	ErrServer       = errors.New("server error")
)

// APIError carries HTTP status + preserved `errors[].detail` from the
// JSON:API envelope for display to the user. Wraps one of the sentinel
// errors above so callers can errors.Is() the class while still showing
// the detail text.
type APIError struct {
	Status     int
	Detail     string // preferred; falls back to Title
	RetryAfter string // verbatim Retry-After header on 429
	sentinel   error
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("HTTP %d: %s", e.Status, e.Detail)
	}
	return fmt.Sprintf("HTTP %d", e.Status)
}

func (e *APIError) Unwrap() error { return e.sentinel }

// Config carries the subset of runtime configuration needed for a request.
// Kept small to avoid importing the top-level config package.
type Config struct {
	APIURL string // e.g. https://www.truestamp.com/api/json
	APIKey string // Bearer token (never logged)
	Team   string // optional tenant id
}

// Latest fetches the most recent finalized/committed beacon.
func Latest(ctx context.Context, cfg Config) (*Beacon, error) {
	body, err := doGet(ctx, cfg, "/beacons/latest")
	if err != nil {
		return nil, err
	}
	return unmarshalBeacon(body)
}

// List fetches up to `limit` most-recent beacons, newest first. The server
// accepts 1..100; callers should pre-clamp if they want a friendly error,
// but the server's own 400 detail surfaces cleanly through APIError too.
func List(ctx context.Context, cfg Config, limit int) ([]Beacon, error) {
	path := "/beacons"
	if limit > 0 {
		path = path + "?limit=" + strconv.Itoa(limit)
	}
	body, err := doGet(ctx, cfg, path)
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
	body, err := doGet(ctx, cfg, "/beacons/"+url.PathEscape(id))
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
	body, err := doGet(ctx, cfg, "/beacons/by-hash/"+url.PathEscape(hash))
	if err != nil {
		return nil, err
	}
	return unmarshalBeacon(body)
}

// doGet issues an authenticated GET and returns the response body on 2xx.
// On non-2xx returns an *APIError wrapping one of the class sentinels.
func doGet(ctx context.Context, cfg Config, path string) ([]byte, error) {
	if cfg.APIKey == "" {
		return nil, &APIError{Status: 401, Detail: "API key not set", sentinel: ErrUnauthorized}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.APIURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
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

// parseAPIError extracts `errors[].detail` (or `title`) from the JSON:API
// error envelope and wraps the appropriate class sentinel.
func parseAPIError(status int, body []byte) *APIError {
	e := &APIError{Status: status, sentinel: sentinelFor(status)}
	var envelope struct {
		Errors []struct {
			Detail string `json:"detail"`
			Title  string `json:"title"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		switch {
		case first.Detail != "":
			e.Detail = first.Detail
		case first.Title != "":
			e.Detail = first.Title
		}
	}
	if e.Detail == "" && len(body) > 0 && body[0] == '<' {
		e.Detail = "server returned HTML error page"
	}
	if e.Detail == "" {
		e.Detail = httpclient.Truncate(string(body), 200)
	}
	return e
}

func sentinelFor(status int) error {
	switch {
	case status == http.StatusUnauthorized:
		return ErrUnauthorized
	case status == http.StatusNotFound:
		return ErrNotFound
	case status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status >= 400 && status < 500:
		return ErrBadRequest
	case status >= 500:
		return ErrServer
	}
	return errors.New("unexpected status")
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

// hashRE matches 64 lowercase hex chars.
var hashRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidateHash returns nil iff s is exactly 64 lowercase hex characters.
// Exported so cobra commands can run this client-side before the network.
func ValidateHash(s string) error {
	if !hashRE.MatchString(s) {
		return fmt.Errorf("hash must be 64 lowercase hex characters, got %q", s)
	}
	return nil
}

// ValidateUUIDv7 returns nil iff s parses as a UUID whose version nibble is 7.
func ValidateUUIDv7(s string) error {
	u, err := uuid.FromString(s)
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}
	if u.Version() != 7 {
		return fmt.Errorf("id must be a UUIDv7 (version=7), got version=%d", u.Version())
	}
	return nil
}
