// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package jsonapi is the one transport every Truestamp JSON:API client
// shares: the authenticated request carrying the tenant header, the
// response size cap, and the classification of the error envelope into a
// small set of sentinels a command can errors.Is. The resource packages
// (internal/beacons, internal/blocks, internal/teams, internal/items,
// internal/proof) own only their routes and their decoding; each used to
// carry its own copy of this file, and the copies had started to differ
// in which statuses mapped to which class.
package jsonapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/redact"
)

// Config carries what a request needs beyond the credential, which the
// process-wide auth.Authorizer installed in cmd/root supplies out of band.
type Config struct {
	APIURL string // e.g. https://www.truestamp.com/api/json
	Team   string // optional tenant id; sent verbatim as the `tenant` header
}

// Class sentinels. An *APIError wraps exactly one of them, so a caller can
// errors.Is the class while still showing the server's detail text.
var (
	ErrUnauthorized = errors.New("not authenticated")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrBadRequest   = errors.New("bad request")
	ErrRateLimited  = errors.New("rate limited")
	ErrServer       = errors.New("server error")
)

// APIError carries the HTTP status and the preserved `errors[].detail`
// (falling back to `title`) from the JSON:API error envelope.
type APIError struct {
	Status     int
	Pointer    string // errors[].source.pointer, when present
	Detail     string
	RetryAfter string // verbatim Retry-After header on 429
	// Sentinel is the class this error belongs to; Unwrap returns it. A
	// resource package may narrow it to one of its own domain sentinels
	// once it has read the structural discriminators, as teams does for
	// the plan-limit and entitlement rejections on create.
	Sentinel error
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("HTTP %d: %s", e.Status, e.Detail)
	}
	return fmt.Sprintf("HTTP %d", e.Status)
}

func (e *APIError) Unwrap() error { return e.Sentinel }

// NotFound is the error a client returns when a filter-style lookup came
// back empty and there was no 404 to classify.
func NotFound(detail string) *APIError {
	return &APIError{Status: http.StatusNotFound, Detail: detail, Sentinel: ErrNotFound}
}

// Get issues an authenticated GET and returns the body on 2xx.
func Get(ctx context.Context, cfg Config, path string) ([]byte, error) {
	return Do(ctx, cfg, http.MethodGet, path, nil)
}

// Post marshals payload as the request body and POSTs it.
func Post(ctx context.Context, cfg Config, path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}
	return Do(ctx, cfg, http.MethodPost, path, body)
}

// Do issues an authenticated request and returns the body on 2xx. Any
// other status is an *APIError wrapping its class sentinel, with the
// Retry-After header preserved on 429.
func Do(ctx context.Context, cfg Config, method, path string, body []byte) ([]byte, error) {
	resp, respBody, err := DoRaw(ctx, cfg, method, path, body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}
	apiErr := ParseError(resp.StatusCode, respBody)
	if resp.StatusCode == http.StatusTooManyRequests {
		apiErr.RetryAfter = resp.Header.Get("Retry-After")
	}
	return nil, apiErr
}

// DoRaw is Do without the status classification: the response (its body
// already read, capped and closed) comes back for a client with its own
// error envelope to parse, which proof generation's `meta.code` needs. A
// missing credential is still an *APIError, because no request is sent.
func DoRaw(ctx context.Context, cfg Config, method, path string, body []byte) (*http.Response, []byte, error) {
	if auth.Default().Mode() == auth.ModeNone {
		return nil, nil, &APIError{Status: http.StatusUnauthorized, Detail: "not authenticated", Sentinel: ErrUnauthorized}
	}
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, cfg.APIURL+path, rdr)
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/vnd.api+json")
	}
	if err := auth.AuthorizeRequest(ctx, req); err != nil {
		return nil, nil, &APIError{Status: http.StatusUnauthorized, Detail: err.Error(), Sentinel: ErrUnauthorized}
	}
	if cfg.Team != "" {
		req.Header.Set("tenant", cfg.Team)
	}

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return nil, nil, fmt.Errorf("reading API response: %w", err)
	}
	return resp, respBody, nil
}

// ParseError classifies a non-2xx response. It keeps `errors[].detail`
// (or `title`) and the `source.pointer` of the first error that carries
// one: the server can return several errors at once (a free-plan user
// requesting team_retains trips both the plan-limit and the entitlement
// rejection), and the pointer is the structural discriminator, so it wins
// over array position.
func ParseError(status int, body []byte) *APIError {
	e := &APIError{Status: status, Sentinel: sentinelFor(status)}
	var envelope struct {
		Errors []struct {
			Detail string `json:"detail"`
			Title  string `json:"title"`
			Source struct {
				Pointer string `json:"pointer"`
			} `json:"source"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		chosen := envelope.Errors[0]
		for i := range envelope.Errors {
			if envelope.Errors[i].Source.Pointer != "" {
				chosen = envelope.Errors[i]
				break
			}
		}
		e.Pointer = chosen.Source.Pointer
		switch {
		case chosen.Detail != "":
			e.Detail = chosen.Detail
		case chosen.Title != "":
			e.Detail = chosen.Title
		}
	}
	if e.Detail == "" {
		// Defense in depth: a server- or attacker-controlled raw body
		// (including a reflected request) is truncated AND run through the
		// secret redactor before it can reach a log or the terminal.
		trimmed := bytes.TrimSpace(body)
		if len(trimmed) > 0 && trimmed[0] == '<' {
			e.Detail = "server returned HTML error page"
		} else {
			e.Detail = redact.String(httpclient.Truncate(string(body), 200))
		}
	}
	return e
}

func sentinelFor(status int) error {
	switch {
	case status == http.StatusUnauthorized:
		return ErrUnauthorized
	case status == http.StatusForbidden:
		return ErrForbidden
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
