// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// stubReleaseServer stands in for the GitHub Releases API. It serves
// whatever Release the test puts on it, under both /releases/latest and
// /releases/tags/:tag. Tests install it via setReleaseServer(t, srv).
func stubReleaseServer(t *testing.T, latest Release, byTag map[string]Release) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(latest)
	})
	mux.HandleFunc("/releases/tags/", func(w http.ResponseWriter, r *http.Request) {
		tag := r.URL.Path[len("/releases/tags/"):]
		rel, ok := byTag[tag]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rel)
	})
	return httptest.NewServer(mux)
}

// setReleaseServer points the package-level URL vars at srv for the
// duration of the test. Restored on t.Cleanup.
func setReleaseServer(t *testing.T, srv *httptest.Server) {
	t.Helper()
	savedLatest := LatestReleaseURL
	savedTag := TagReleaseURLFormat
	LatestReleaseURL = srv.URL + "/releases/latest"
	TagReleaseURLFormat = srv.URL + "/releases/tags/%s"
	t.Cleanup(func() {
		LatestReleaseURL = savedLatest
		TagReleaseURLFormat = savedTag
	})
}

func TestCheck_upgradeAvailable(t *testing.T) {
	srv := stubReleaseServer(t, Release{TagName: "v0.4.0"}, nil)
	defer srv.Close()
	setReleaseServer(t, srv)

	got, err := Check(context.Background(), Options{CurrentVersion: "v0.3.0"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !got.UpgradeAvail {
		t.Errorf("UpgradeAvail = false, want true (v0.3.0 < v0.4.0)")
	}
	if got.LatestVersion != "v0.4.0" {
		t.Errorf("LatestVersion = %q, want v0.4.0", got.LatestVersion)
	}
	if got.PreRelease {
		t.Error("PreRelease = true, want false")
	}
}

func TestCheck_upToDate(t *testing.T) {
	srv := stubReleaseServer(t, Release{TagName: "v0.4.0"}, nil)
	defer srv.Close()
	setReleaseServer(t, srv)

	got, err := Check(context.Background(), Options{CurrentVersion: "v0.4.0"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got.UpgradeAvail {
		t.Error("UpgradeAvail = true, want false (same version)")
	}
}

func TestCheck_newerCurrent(t *testing.T) {
	// User is on a newer version than "latest" (e.g., ran a dev build).
	srv := stubReleaseServer(t, Release{TagName: "v0.3.0"}, nil)
	defer srv.Close()
	setReleaseServer(t, srv)

	got, err := Check(context.Background(), Options{CurrentVersion: "v0.4.0"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got.UpgradeAvail {
		t.Error("UpgradeAvail = true, want false (current > latest)")
	}
}

// TestCheck_preReleaseLatest_layer2 locks in the Layer-2 defense: if
// "latest" resolves to something with a semver pre-release suffix AND
// the caller didn't pin --version, Check must return ErrPreRelease
// regardless of what GitHub said about the prerelease flag.
func TestCheck_preReleaseLatest_layer2(t *testing.T) {
	// GitHub's prerelease flag is false, but the tag carries a
	// semver -rc.1 suffix. Our Go code rejects it.
	srv := stubReleaseServer(t, Release{TagName: "v1.0.0-rc.1", Prerelease: false}, nil)
	defer srv.Close()
	setReleaseServer(t, srv)

	got, err := Check(context.Background(), Options{CurrentVersion: "v0.3.0"})
	if !errors.Is(err, ErrPreRelease) {
		t.Fatalf("Check: err = %v, want ErrPreRelease", err)
	}
	if got == nil || got.LatestVersion != "v1.0.0-rc.1" {
		t.Errorf("result does not carry latest version: %+v", got)
	}
	if !got.PreRelease {
		t.Error("PreRelease flag should be set in the returned result")
	}
}

// TestCheck_preReleaseLatest_layer1 is the GitHub-flag path: the tag
// itself doesn't carry a suffix, but the Release is marked as
// prerelease=true. Should still be caught.
func TestCheck_preReleaseLatest_layer1(t *testing.T) {
	srv := stubReleaseServer(t, Release{TagName: "v1.0.0", Prerelease: true}, nil)
	defer srv.Close()
	setReleaseServer(t, srv)

	_, err := Check(context.Background(), Options{CurrentVersion: "v0.3.0"})
	if !errors.Is(err, ErrPreRelease) {
		t.Errorf("Check with Prerelease=true: err = %v, want ErrPreRelease", err)
	}
}

// TestCheck_preRelease_explicitPin confirms that --version pinning
// bypasses the pre-release defense — the user asked for it by name.
func TestCheck_preRelease_explicitPin(t *testing.T) {
	byTag := map[string]Release{
		"v1.0.0-rc.1": {TagName: "v1.0.0-rc.1"},
	}
	srv := stubReleaseServer(t, Release{TagName: "v0.3.0"}, byTag)
	defer srv.Close()
	setReleaseServer(t, srv)

	got, err := Check(context.Background(), Options{
		CurrentVersion: "v0.3.0",
		TargetVersion:  "v1.0.0-rc.1",
	})
	if err != nil {
		t.Fatalf("Check with explicit pre-release pin: err = %v, want nil", err)
	}
	if got.LatestVersion != "v1.0.0-rc.1" {
		t.Errorf("LatestVersion = %q, want v1.0.0-rc.1", got.LatestVersion)
	}
	if !got.PreRelease {
		t.Error("PreRelease flag should still be set in result")
	}
}

func TestCheck_unparseableCurrent(t *testing.T) {
	// "dev" doesn't parse as semver. Check should still return a
	// useful result with UpgradeAvail=true — we always prefer
	// over-reporting to leaving dev builds stale.
	srv := stubReleaseServer(t, Release{TagName: "v0.4.0"}, nil)
	defer srv.Close()
	setReleaseServer(t, srv)

	got, err := Check(context.Background(), Options{CurrentVersion: "dev"})
	if err != nil {
		t.Fatalf("Check with dev version: err = %v", err)
	}
	if !got.UpgradeAvail {
		t.Error("UpgradeAvail = false for dev version, want true (over-report)")
	}
}

func TestCheck_networkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()
	setReleaseServer(t, srv)

	_, err := Check(context.Background(), Options{CurrentVersion: "v0.3.0"})
	if err == nil {
		t.Fatal("Check on 429: expected error, got nil")
	}
}
