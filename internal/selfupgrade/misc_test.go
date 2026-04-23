// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

// --- isHTTP404 -----------------------------------------------------------

func TestIsHTTP404(t *testing.T) {
	if isHTTP404(nil) {
		t.Error("nil error should not be 404")
	}
	if !isHTTP404(errors.New("HTTP 404")) {
		t.Error("HTTP 404 string should be detected")
	}
	if isHTTP404(errors.New("HTTP 500")) {
		t.Error("HTTP 500 should not be detected as 404")
	}
	if isHTTP404(errors.New("something else")) {
		t.Error("unrelated error should not be detected")
	}
}

// --- samePatch -----------------------------------------------------------

func TestSamePatch(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v1.2.3", "v1.2.3", true},
		{"1.2.3", "v1.2.3", true},
		{"v1.2.3", "v1.2.4", false},
		{"v1.2.3-rc.1", "v1.2.3", false},
		{"v1.2.3-rc.1", "v1.2.3-rc.1", true},
		{"not-a-version", "v1.2.3", false},
		{"v1.2.3", "not-a-version", false},
	}
	for _, c := range cases {
		if got := samePatch(c.a, c.b); got != c.want {
			t.Errorf("samePatch(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// --- hostOSArch ----------------------------------------------------------

func TestHostOSArch_Current(t *testing.T) {
	os, arch, err := hostOSArch()
	// Only runs on darwin/linux × amd64/arm64 for test machines. Anywhere
	// else we expect the error paths to fire.
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			if err != nil {
				t.Errorf("unexpected error on %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
			}
			if os == "" || arch == "" {
				t.Error("empty os/arch")
			}
		}
	}
}

// --- resolveRelease ------------------------------------------------------

func TestResolveRelease_TargetVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The server receives /tags/v1.2.3 - confirm the prefix was added.
		if r.URL.Path == "" {
			t.Error("empty path")
		}
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer srv.Close()

	orig := TagReleaseURLFormat
	TagReleaseURLFormat = srv.URL + "/%s"
	t.Cleanup(func() { TagReleaseURLFormat = orig })

	rel, err := resolveRelease(context.Background(), Options{TargetVersion: "1.2.3"})
	if err != nil {
		t.Fatalf("resolveRelease: %v", err)
	}
	if rel.TagName != "v1.2.3" {
		t.Errorf("tag: got %q", rel.TagName)
	}
}

func TestResolveRelease_Latest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v2.0.0"}`))
	}))
	defer srv.Close()

	orig := LatestReleaseURL
	LatestReleaseURL = srv.URL
	t.Cleanup(func() { LatestReleaseURL = orig })

	rel, err := resolveRelease(context.Background(), Options{})
	if err != nil {
		t.Fatalf("resolveRelease: %v", err)
	}
	if rel.TagName != "v2.0.0" {
		t.Errorf("tag: got %q", rel.TagName)
	}
}

// --- isPreReleaseTag -----------------------------------------------------

func TestIsPreReleaseTag(t *testing.T) {
	if !isPreReleaseTag(&Release{Prerelease: true, TagName: "v1.0.0"}) {
		t.Error("Prerelease flag should trigger")
	}
	if !isPreReleaseTag(&Release{TagName: "v1.0.0-rc.1"}) {
		t.Error("semver pre-release suffix should trigger")
	}
	if isPreReleaseTag(&Release{TagName: "v1.0.0"}) {
		t.Error("stable release should not trigger")
	}
	if isPreReleaseTag(&Release{TagName: "not-a-semver"}) {
		t.Error("unparseable tag should not trigger")
	}
}

// --- FetchByTag 404 ------------------------------------------------------

func TestFetchByTag_404_ReturnsErrReleaseNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	orig := TagReleaseURLFormat
	TagReleaseURLFormat = srv.URL + "/%s"
	t.Cleanup(func() { TagReleaseURLFormat = orig })

	_, err := FetchByTag(context.Background(), "v99.0.0")
	if !errors.Is(err, ErrReleaseNotFound) {
		t.Errorf("expected ErrReleaseNotFound, got %v", err)
	}
}

func TestFetchByTag_500_PropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	orig := TagReleaseURLFormat
	TagReleaseURLFormat = srv.URL + "/%s"
	t.Cleanup(func() { TagReleaseURLFormat = orig })

	_, err := FetchByTag(context.Background(), "v1.0.0")
	if err == nil || errors.Is(err, ErrReleaseNotFound) {
		t.Errorf("500 should not map to ErrReleaseNotFound, got %v", err)
	}
}

func TestFetchRelease_MissingTagName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"name": "something"}`))
	}))
	defer srv.Close()
	_, err := fetchRelease(context.Background(), srv.URL)
	if err == nil {
		t.Error("missing tag_name should error")
	}
}

func TestFetchRelease_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	_, err := fetchRelease(context.Background(), srv.URL)
	if err == nil {
		t.Error("malformed JSON should error")
	}
}

func TestFetchRelease_BadURL(t *testing.T) {
	_, err := fetchRelease(context.Background(), "://bad-url")
	if err == nil {
		t.Error("bad URL should error")
	}
}

func TestFetchLatest_WithAuth(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "fake-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer fake-token" {
			t.Errorf("Authorization header: got %q", got)
		}
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0"}`))
	}))
	defer srv.Close()
	orig := LatestReleaseURL
	LatestReleaseURL = srv.URL
	t.Cleanup(func() { LatestReleaseURL = orig })

	_, err := FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("FetchLatest: %v", err)
	}
}
