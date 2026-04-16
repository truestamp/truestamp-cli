// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// Release is the subset of the GitHub Releases API response we consume.
type Release struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
	HTMLURL    string `json:"html_url"`
}

// ReleasesRepo is the GitHub slug truestamp release artifacts come from.
const ReleasesRepo = "truestamp/truestamp-cli"

// LatestReleaseURL is the REST endpoint returning the most recent
// non-prerelease, non-draft release. Unauthenticated GitHub requests are
// capped at 60/hr per IP; callers can set GITHUB_TOKEN to lift that to
// 5000/hr. Declared as var (not const) so tests can redirect to an
// httptest server; do not reassign in production code.
var LatestReleaseURL = "https://api.github.com/repos/" + ReleasesRepo + "/releases/latest"

// TagReleaseURLFormat is a printf format for fetching a specific tag.
// Same var-for-testability rationale as LatestReleaseURL.
var TagReleaseURLFormat = "https://api.github.com/repos/" + ReleasesRepo + "/releases/tags/%s"

// ErrReleaseNotFound is returned when the requested tag has no release.
var ErrReleaseNotFound = errors.New("release not found")

// FetchLatest returns the current latest non-prerelease, non-draft
// release. Respects ctx cancellation.
func FetchLatest(ctx context.Context) (*Release, error) {
	return fetchRelease(ctx, LatestReleaseURL)
}

// FetchByTag returns the release for a specific tag (e.g. "v0.3.0").
// Returns ErrReleaseNotFound if GitHub returns 404.
func FetchByTag(ctx context.Context, tag string) (*Release, error) {
	url := fmt.Sprintf(TagReleaseURLFormat, tag)
	rel, err := fetchRelease(ctx, url)
	if err != nil && isHTTP404(err) {
		return nil, ErrReleaseNotFound
	}
	return rel, err
}

func fetchRelease(ctx context.Context, url string) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// GitHub recommends setting a user-agent. The API returns HTTP 403
	// without one.
	req.Header.Set("User-Agent", "truestamp-cli-upgrade")
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var rel Release
	// We deliberately do NOT call DisallowUnknownFields — the GitHub
	// payload is much larger than our shape and we only care about the
	// handful of fields declared on Release.
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release JSON: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release JSON missing tag_name")
	}
	return &rel, nil
}

func isHTTP404(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "HTTP 404"
}
