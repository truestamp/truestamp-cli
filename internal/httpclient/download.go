// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

// DefaultMaxDownloadSize caps a single DownloadCtx response at 200 MB —
// comfortably larger than any truestamp release tarball today, small
// enough to prevent a runaway redirect from filling the disk.
const DefaultMaxDownloadSize = 200 << 20

// DownloadCtx streams the body of a GET request to destPath. The
// destination is created (or truncated) with 0644 permissions. The
// response body is read through an [io.LimitReader] whose cap is the
// larger of maxBytes and DefaultMaxDownloadSize; pass 0 to use the
// default.
//
// Unlike GetJSONCtx, this function is safe for multi-MB responses and
// never buffers the full body in memory.
//
// Returns the number of bytes written on success.
func DownloadCtx(ctx context.Context, url, destPath string, maxBytes int64) (int64, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxDownloadSize
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	stampUserAgent(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read a small slice of the body for error context without
		// exhausting a large error page.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, Truncate(string(snippet), 120))
	}

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", destPath, err)
	}

	n, copyErr := io.Copy(out, io.LimitReader(resp.Body, maxBytes+1))
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(destPath)
		return n, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(destPath)
		return n, closeErr
	}
	if n > maxBytes {
		_ = os.Remove(destPath)
		return n, fmt.Errorf("response exceeded %d byte cap (got at least %d)", maxBytes, n)
	}

	return n, nil
}

// DownloadBytesCtx is like DownloadCtx but returns the body in memory.
// Intended for small artifacts (checksums.txt, signature bundles) that
// exceed MaxResponseSize occasionally but are still known to be small
// (<1 MB). Pass 0 for maxBytes to default to 1 MB.
func DownloadBytesCtx(ctx context.Context, url string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = MaxResponseSize
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	stampUserAgent(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, Truncate(string(snippet), 120))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("response exceeded %d byte cap", maxBytes)
	}
	return body, nil
}
