// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package items provides API operations for Truestamp items.
package items

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/truestamp/truestamp-cli/internal/jsonapi"
)

// CreateItemResponse holds the parsed JSON:API response for a created item.
type CreateItemResponse struct {
	ID         string
	State      string
	ClaimsHash string
	ItemHash   string
	Visibility string
	Tags       []string
	TeamID     string
	Name       string
	Hash       string
	HashType   string
}

// CreateItem calls [CreateItemCtx] with [context.Background].
func CreateItem(apiURL, team string, claims map[string]any, visibility string, tags []string) (*CreateItemResponse, error) {
	return CreateItemCtx(context.Background(), apiURL, team, claims, visibility, tags)
}

// CreateItemCtx sends a JSON:API POST request to create a new item. claims
// is the nested claims map (hash, hash_type, name, etc.); visibility and
// tags are top-level item attributes. ctx cancels the in-flight request.
// The credential is applied by the process-wide [auth.Authorizer].
func CreateItemCtx(ctx context.Context, apiURL, team string, claims map[string]any, visibility string, tags []string) (*CreateItemResponse, error) {
	// Build JSON:API envelope
	attributes := map[string]any{
		"claims": claims,
	}
	if visibility != "" {
		attributes["visibility"] = visibility
	}
	if len(tags) > 0 {
		attributes["tags"] = tags
	}

	body := map[string]any{
		"data": map[string]any{
			"type":       "item",
			"attributes": attributes,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	respBody, err := jsonapi.Do(ctx, jsonapi.Config{APIURL: apiURL, Team: team}, http.MethodPost, "/items", bodyBytes)
	if err != nil {
		return nil, err
	}
	return parseResponse(respBody)
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
