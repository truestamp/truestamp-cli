// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package items provides API operations for Truestamp items.
package items

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
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

// CreateItem sends a JSON:API POST request to create a new item.
// claims is the nested claims map (hash, hash_type, name, etc.).
// visibility and tags are top-level item attributes.
func CreateItem(apiURL, apiKey, team string, claims map[string]any, visibility string, tags []string) (*CreateItemResponse, error) {
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

	reqURL := apiURL + "/items"
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if team != "" {
		req.Header.Set("tenant", team)
	}

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseError(resp.StatusCode, respBody)
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

// parseError extracts a user-friendly message from a JSON:API error response.
func parseError(statusCode int, body []byte) error {
	var envelope struct {
		Errors []struct {
			Status string `json:"status"`
			Detail string `json:"detail"`
			Title  string `json:"title"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		if first.Detail != "" {
			return fmt.Errorf("API error (HTTP %d): %s", statusCode, first.Detail)
		}
		if first.Title != "" {
			return fmt.Errorf("API error (HTTP %d): %s", statusCode, first.Title)
		}
	}

	bodyStr := string(body)
	if len(bodyStr) > 0 && bodyStr[0] == '<' {
		return fmt.Errorf("API error (HTTP %d): server returned HTML error page", statusCode)
	}
	return fmt.Errorf("API error (HTTP %d): %s", statusCode, httpclient.Truncate(bodyStr, 200))
}