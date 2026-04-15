// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package httpclient provides a shared HTTP client for all external API calls.
// The client is safe for concurrent use and reuses connections.
// Call Init once during startup to set the timeout; the default is 10s.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// httpClient is the shared HTTP client.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// MaxResponseSize limits HTTP response bodies to 1 MB to prevent OOM.
const MaxResponseSize = 1 << 20

// Init creates a new HTTP client with the given timeout.
// Must be called once during startup before any external calls.
func Init(timeout time.Duration) {
	httpClient = &http.Client{Timeout: timeout}
}

// Do executes an HTTP request using the shared client. The request's
// existing [context.Context] (if any) is respected; callers that want
// cancellation should attach one via [http.Request.WithContext] before
// calling.
func Do(req *http.Request) (*http.Response, error) {
	return httpClient.Do(req)
}

// GetJSON performs a GET request with [context.Background] and returns the
// response body. Prefer [GetJSONCtx] when a cancellable context is
// available (e.g. from Cobra's cmd.Context()).
func GetJSON(url string) ([]byte, error) {
	return GetJSONCtx(context.Background(), url)
}

// GetJSONCtx performs a context-aware GET request and returns the response
// body. Returns an error for non-2xx status codes or on ctx cancellation.
func GetJSONCtx(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyStr := string(body)
		if len(bodyStr) > 0 && bodyStr[0] == '<' {
			return nil, fmt.Errorf("HTTP %d (server returned HTML error page)", resp.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, Truncate(bodyStr, 80))
	}

	return body, nil
}

// Truncate shortens a string to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
