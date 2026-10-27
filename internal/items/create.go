// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package items provides API operations for Truestamp items.
package items

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/jsonapi"
)

// CreateItemResponse holds the parsed JSON:API response for a created item.
type CreateItemResponse struct {
	ID         string
	State      string
	ClaimsHash string
	// ClaimsJCS is the server's claims_jcs: the canonical form (RFC 8785)
	// of the claims it accepted, the bytes ClaimsHash is computed over.
	ClaimsJCS  string
	ItemHash   string
	Visibility string
	Tags       []string
	TeamID     string
	Name       string
	Hash       string
	HashType   string
}

// CodeItemCodeUnavailable is the server's code for a submission it could
// not give an item code. Nothing was created, and the same request may be
// sent again.
const CodeItemCodeUnavailable = "item_code_unavailable"

// ErrItemCodeUnavailable is the class of a 409 item_code_unavailable
// refusal: nothing was created, and sending the same request again is
// expected to succeed.
var ErrItemCodeUnavailable = errors.New("the server could not issue an item code")

// claimsPointer is where the JSON:API create puts the claims, and so the
// prefix of every source.pointer that names a member of them.
const claimsPointer = "/data/attributes/claims"

// ClaimsReason is one claims rule the server found broken: its code (such
// as claims_surrounding_whitespace), its message, which names the fix, and
// Pointer, the RFC 6901 pointer into the claims at the member that breaks
// it ("" for the claims as a whole).
type ClaimsReason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Pointer string `json:"pointer"`
}

// ClaimsRefusedError is a create the server refused because the claims
// break one or more of its rules: every rule broken, in the server's order.
// Others holds the detail of any error in the same response that names no
// member of the claims, so that nothing the server said is dropped.
type ClaimsRefusedError struct {
	Status  int
	Reasons []ClaimsReason
	Others  []string
}

func (e *ClaimsRefusedError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "HTTP %d: the server refused the claims", e.Status)
	for _, r := range e.Reasons {
		fmt.Fprintf(&b, "; %s at %q: %s", r.Code, r.Pointer, r.Message)
	}
	for _, o := range e.Others {
		fmt.Fprintf(&b, "; %s", o)
	}
	return b.String()
}

// CreateItemCtx sends a JSON:API POST request to create a new item.
// claimsJSON is the claims as a JSON text, one value the caller has read
// with the strict reader (internal/strictjson); it is written into the
// request body byte for byte as the value of attributes.claims, never
// decoded and re-encoded, so the server reads exactly the text the user
// gave. visibility and tags are item attributes. ctx cancels the in-flight
// request. The credential is applied by the process-wide [auth.Authorizer].
//
// A refusal of the claims is a [*ClaimsRefusedError]; a 409
// item_code_unavailable is a [*jsonapi.APIError] whose Sentinel is
// [ErrItemCodeUnavailable]; any other non-2xx status is a
// [*jsonapi.APIError].
func CreateItemCtx(ctx context.Context, apiURL, team string, claimsJSON []byte, visibility string, tags []string) (*CreateItemResponse, error) {
	body, err := createBody(claimsJSON, visibility, tags)
	if err != nil {
		return nil, err
	}

	resp, respBody, err := jsonapi.DoRaw(ctx, jsonapi.Config{APIURL: apiURL, Team: team}, http.MethodPost, "/items", body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return parseResponse(respBody)
	}
	if refused := parseClaimsRefusal(resp.StatusCode, respBody); refused != nil {
		return nil, refused
	}
	apiErr := jsonapi.ErrorFromResponse(resp, respBody)
	if resp.StatusCode == http.StatusConflict && apiErr.Code == CodeItemCodeUnavailable {
		apiErr.Sentinel = ErrItemCodeUnavailable
	}
	return nil, apiErr
}

// createBody writes the JSON:API document with claimsJSON spliced in
// verbatim. encoding/json is not used for the claims: json.RawMessage is
// compacted and HTML-escaped when marshaled, which would change the text.
func createBody(claimsJSON []byte, visibility string, tags []string) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(`{"data":{"type":"item","attributes":{"claims":`)
	b.Write(claimsJSON)
	if visibility != "" {
		v, err := json.Marshal(visibility)
		if err != nil {
			return nil, fmt.Errorf("encoding request: %w", err)
		}
		b.WriteString(`,"visibility":`)
		b.Write(v)
	}
	if len(tags) > 0 {
		t, err := json.Marshal(tags)
		if err != nil {
			return nil, fmt.Errorf("encoding request: %w", err)
		}
		b.WriteString(`,"tags":`)
		b.Write(t)
	}
	b.WriteString(`}}}`)
	return b.Bytes(), nil
}

// parseClaimsRefusal reads a 422 whose errors name members of the claims
// (source.pointer under /data/attributes/claims) as a ClaimsRefusedError,
// one reason per such error with its pointer made relative to the claims.
// It returns nil for any other response.
func parseClaimsRefusal(status int, body []byte) *ClaimsRefusedError {
	if status != http.StatusUnprocessableEntity {
		return nil
	}
	var envelope struct {
		Errors []struct {
			Code   string `json:"code"`
			Detail string `json:"detail"`
			Title  string `json:"title"`
			Source struct {
				Pointer string `json:"pointer"`
			} `json:"source"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil
	}
	refused := &ClaimsRefusedError{Status: status}
	for _, e := range envelope.Errors {
		message := e.Detail
		if message == "" {
			message = e.Title
		}
		pointer, inClaims := claimsMemberPointer(e.Source.Pointer)
		if !inClaims {
			other := message
			if e.Code != "" {
				other = e.Code + ": " + message
			}
			refused.Others = append(refused.Others, other)
			continue
		}
		refused.Reasons = append(refused.Reasons, ClaimsReason{Code: e.Code, Message: message, Pointer: pointer})
	}
	if len(refused.Reasons) == 0 {
		return nil
	}
	return refused
}

// claimsMemberPointer returns source, a pointer into the request document,
// as a pointer into the claims, and whether it points into them at all.
func claimsMemberPointer(source string) (string, bool) {
	if source == claimsPointer {
		return "", true
	}
	if rest, ok := strings.CutPrefix(source, claimsPointer); ok && strings.HasPrefix(rest, "/") {
		return rest, true
	}
	return "", false
}

// parseResponse extracts item data from a JSON:API response envelope.
func parseResponse(body []byte) (*CreateItemResponse, error) {
	var envelope struct {
		Data struct {
			ID         string         `json:"id"`
			Type       string         `json:"type"`
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	attrs := envelope.Data.Attributes
	r := &CreateItemResponse{
		ID:         envelope.Data.ID,
		State:      getString(attrs, "state"),
		ClaimsHash: getString(attrs, "claims_hash"),
		ClaimsJCS:  getString(attrs, "claims_jcs"),
		ItemHash:   getString(attrs, "item_hash"),
		Visibility: getString(attrs, "visibility"),
		TeamID:     getString(attrs, "team_id"),
		Name:       getString(attrs, "display_name"),
	}

	// Extract hash and hash_type from claims (the user's data hash)
	if claimsRaw, ok := attrs["claims"]; ok {
		if claimsMap, ok := claimsRaw.(map[string]any); ok {
			r.Hash = getString(claimsMap, "hash")
			r.HashType = getString(claimsMap, "hash_type")
		}
	}

	if tagsRaw, ok := attrs["tags"]; ok {
		if tagSlice, ok := tagsRaw.([]any); ok {
			for _, t := range tagSlice {
				if s, ok := t.(string); ok {
					r.Tags = append(r.Tags, s)
				}
			}
		}
	}

	return r, nil
}

func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
