// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCheckHealthTarget_OK confirms a healthy 200 response classifies
// as healthOK and records the latency. Uses a real httptest.Server so
// the path through net/http stays exercised end to end.
func TestCheckHealthTarget_OK(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := checkHealthTarget(context.Background(), HealthTarget{Name: "test", URL: srv.URL})
	if r.State != healthOK {
		t.Errorf("expected healthOK, got state=%d (err=%v)", r.State, r.Err)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", r.StatusCode)
	}
	if r.Latency <= 0 {
		t.Errorf("expected positive latency, got %v", r.Latency)
	}
}

// TestCheckHealthTarget_FallbackToGET ensures a server that rejects
// HEAD with 405 still gets probed correctly via GET.
func TestCheckHealthTarget_FallbackToGET(t *testing.T) {
	t.Parallel()

	var sawGET bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		sawGET = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := checkHealthTarget(context.Background(), HealthTarget{Name: "test", URL: srv.URL})
	if r.State != healthOK {
		t.Errorf("expected fallback to succeed; got state=%d", r.State)
	}
	if !sawGET {
		t.Errorf("expected GET fallback to fire")
	}
}

// TestCheckHealthTarget_Degraded covers the 4xx/5xx branch:
// reachable host but the server returned an error code.
func TestCheckHealthTarget_Degraded(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	r := checkHealthTarget(context.Background(), HealthTarget{Name: "test", URL: srv.URL})
	if r.State != healthDegraded {
		t.Errorf("expected healthDegraded, got %d", r.State)
	}
	if r.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", r.StatusCode)
	}
}

// TestCheckHealthTarget_Unreachable covers the network-failure
// branch. A guaranteed-unreachable URL ensures we hit the
// healthFailed path with a populated Err.
func TestCheckHealthTarget_Unreachable(t *testing.T) {
	t.Parallel()

	r := checkHealthTarget(context.Background(),
		HealthTarget{Name: "test", URL: "http://127.0.0.1:1"})
	if r.State != healthFailed {
		t.Errorf("expected healthFailed for unreachable target, got %d (err=%v)", r.State, r.Err)
	}
	if r.Err == nil {
		t.Errorf("expected non-nil err on healthFailed result")
	}
}

// TestDefaultHealthTargets_HonorsConfig verifies the canonical list
// uses caller-provided URLs (so user overrides flow through) and
// always pins the third-party verification endpoints. The Truestamp
// row points at the host's /health endpoint, derived from api_url.
func TestDefaultHealthTargets_HonorsConfig(t *testing.T) {
	t.Parallel()

	got := DefaultHealthTargets("https://api.example/api/json", "https://kr.example/keyring.json")

	wantURLs := map[string]bool{
		"https://api.example/health":                     false,
		"https://kr.example/keyring.json":                false,
		"https://beacon.nist.gov/beacon/2.0":             false,
		"https://horizon.stellar.org":                    false,
		"https://blockstream.info/api/blocks/tip/height": false,
	}
	for _, target := range got {
		if _, ok := wantURLs[target.URL]; ok {
			wantURLs[target.URL] = true
		}
	}
	for url, found := range wantURLs {
		if !found {
			t.Errorf("DefaultHealthTargets missing %q", url)
		}
	}
}

// TestDeriveHealthURL covers the canonical mappings users expect:
// the production API, a local preview server on a non-standard
// port, and a deployment with a custom path. Each one should map
// to the same host's /health endpoint.
func TestDeriveHealthURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
	}{
		{"https://www.truestamp.com/api/json", "https://www.truestamp.com/health"},
		{"http://localhost:4010/api/json", "http://localhost:4010/health"},
		{"http://localhost:4000", "http://localhost:4000/health"},
		{"https://staging.example.com:8443/v1/api?x=y", "https://staging.example.com:8443/health"},
		{"", ""},
		{"not-a-url", ""},         // no scheme or host
		{"file:///etc/hosts", ""}, // no host
	}
	for _, tc := range cases {
		if got := deriveHealthURL(tc.in); got != tc.want {
			t.Errorf("deriveHealthURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestDefaultHealthTargets_OmitsEmpty: if the user has no api_url
// configured we shouldn't add an empty-URL row to the table.
func TestDefaultHealthTargets_OmitsEmpty(t *testing.T) {
	t.Parallel()

	got := DefaultHealthTargets("", "")
	for _, target := range got {
		if target.URL == "" {
			t.Errorf("DefaultHealthTargets emitted an empty URL row: %+v", target)
		}
	}
}

// TestSortHealthResultsPrioritizesProblems checks the surface
// invariant that failures sort to the top. We can't observe the
// sort directly via the table renderer — sort a synthetic slice
// instead.
func TestSortHealthResultsPrioritizesProblems(t *testing.T) {
	t.Parallel()

	rs := []healthResult{
		{Target: HealthTarget{Name: "z-ok"}, State: healthOK},
		{Target: HealthTarget{Name: "a-fail"}, State: healthFailed},
		{Target: HealthTarget{Name: "m-degraded"}, State: healthDegraded},
		{Target: HealthTarget{Name: "n-checking"}, State: healthChecking},
	}
	sortHealthResults(rs)

	wantOrder := []string{"a-fail", "m-degraded", "n-checking", "z-ok"}
	for i, want := range wantOrder {
		if rs[i].Target.Name != want {
			t.Errorf("sort order at index %d: got %q, want %q", i, rs[i].Target.Name, want)
		}
	}
}

// TestRenderHealthSection_IconsAndDetails feeds a model with a mix
// of states and confirms each renders with its expected icon plus
// detail fragment.
func TestRenderHealthSection_IconsAndDetails(t *testing.T) {
	t.Parallel()

	targets := []HealthTarget{
		{Name: "Ok target", URL: "http://ok.example"},
		{Name: "Bad gateway", URL: "http://502.example"},
		{Name: "Down host", URL: "http://down.example"},
		{Name: "Pending", URL: "http://slow.example"},
	}
	m := newConnectionModel("", "", targets)
	m.healthResults[0].State = healthOK
	m.healthResults[0].Latency = 42_000_000 // 42ms in nanoseconds
	m.healthResults[1].State = healthDegraded
	m.healthResults[1].StatusCode = 502
	m.healthResults[2].State = healthFailed
	m.healthResults[2].Err = errors.New("connection refused")
	m.healthResults[3].State = healthChecking

	out := stripANSI(m.renderHealthSection())

	wantParts := []string{
		"External services",
		"✓",
		"✗",
		"!",
		"Ok target",
		"42ms",
		"HTTP 502",
		"unreachable",
		"checking",
	}
	for _, want := range wantParts {
		if !strings.Contains(out, want) {
			t.Errorf("health section missing %q\n--- rendered ---\n%s", want, out)
		}
	}
}

// TestCanRunManualHealthCheckThrottles confirms the rate limiter
// flips back and forth as time advances. Uses internal state poke
// rather than time.Sleep so the test stays fast.
func TestCanRunManualHealthCheckThrottles(t *testing.T) {
	t.Parallel()

	m := newConnectionModel("", "", []HealthTarget{{Name: "x", URL: "http://x"}})
	if !m.canRunManualHealthCheck() {
		t.Errorf("zero-value lastHealthRunAt should always allow a manual run")
	}
	m.dispatchAllChecks(context.Background()) // sets lastHealthRunAt
	if m.canRunManualHealthCheck() {
		t.Errorf("should be throttled immediately after a run")
	}
}
