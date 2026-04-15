// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// Download fetches a proof bundle from a URL and validates its basic structure.
// Returns the raw bytes suitable for passing to ParseBytes.
func Download(rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("URL must include a host")
	}

	data, err := httpclient.GetJSON(rawURL)
	if err != nil {
		return nil, fmt.Errorf("downloading proof: %w", err)
	}

	// Quick-validate: check that the response looks like a proof bundle (compact format)
	var shape struct {
		Version   int             `json:"v"`
		PublicKey string          `json:"pk"`
		Signature string          `json:"sig"`
		Subject   json.RawMessage `json:"s"`
		Block     json.RawMessage `json:"b"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return nil, fmt.Errorf("response is not valid JSON: %w", err)
	}
	if shape.Version == 0 || shape.PublicKey == "" || shape.Signature == "" || len(shape.Subject) == 0 || len(shape.Block) == 0 {
		return nil, fmt.Errorf("response does not appear to be a Truestamp proof bundle (missing v, pk, sig, s, or b)")
	}

	return data, nil
}

// Generate requests a proof bundle from the Truestamp API for the given subject ID.
// The server auto-detects item (ULID) vs entropy (UUIDv7) from the ID format.
// format should be "json" or "cbor".
// Returns raw bytes ready to write to a file (pretty JSON or decoded CBOR binary).
func Generate(apiURL, apiKey, team, id, format string) ([]byte, error) {
	dataFields := map[string]string{"id": id}
	if format != "" && format != "json" {
		dataFields["format"] = format
	}
	requestBody := map[string]any{"data": dataFields}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encoding request body: %w", err)
	}

	reqURL := apiURL + "/proof/generate"
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
		return nil, fmt.Errorf("reading API response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseGenerateError(resp.StatusCode, respBody)
	}

	// Parse the response envelope {"result": ...}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("parsing API response: %w", err)
	}
	if len(envelope.Result) == 0 {
		return nil, fmt.Errorf("API response missing 'result' field")
	}

	if format == "cbor" {
		// Result is a base64-encoded CBOR binary string (JSON string with quotes)
		var b64 string
		if err := json.Unmarshal(envelope.Result, &b64); err != nil {
			return nil, fmt.Errorf("parsing CBOR base64 from response: %w", err)
		}
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("decoding CBOR base64: %w", err)
		}
		return decoded, nil
	}

	// JSON format: pretty-print the result map
	var raw any
	if err := json.Unmarshal(envelope.Result, &raw); err != nil {
		return nil, fmt.Errorf("parsing proof JSON: %w", err)
	}
	pretty, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("formatting proof JSON: %w", err)
	}
	return append(pretty, '\n'), nil
}

// parseGenerateError extracts a meaningful error message from the API response.
func parseGenerateError(statusCode int, body []byte) error {
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
