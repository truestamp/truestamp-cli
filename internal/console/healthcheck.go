// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// healthState classifies a single health-check outcome. The states are
// ordered roughly by severity so a sort by state surfaces problems
// first when we render the section.
type healthState int

const (
	healthUnknown  healthState = iota // not yet checked this cycle
	healthChecking                    // request in flight
	healthOK                          // 2xx / 3xx response
	healthDegraded                    // 4xx / 5xx response (host up, content not what we expect)
	healthFailed                      // network failure (DNS, refused, timeout, TLS, etc.)
)

// HealthTarget is a service the CLI talks to during normal operation.
// The Connection pane probes each target and reports whether it's
// reachable so a user diagnosing a problem can tell at a glance
// whether their network is broken, the Truestamp service is down, or
// a third-party verification source is unhealthy.
type HealthTarget struct {
	// Name is a short, human-readable label rendered in the table,
	// e.g. "Truestamp API" or "NIST Beacon (entropy)".
	Name string

	// URL is the address we probe. The check uses HEAD and falls
	// back to GET if the server doesn't allow HEAD (some endpoints
	// reply 405 Method Not Allowed).
	URL string
}

// healthResult carries one row's worth of state. The pane renders
// these directly; the trailing fields default to zero values during
// the "checking" phase and are populated once the request returns.
type healthResult struct {
	Target     HealthTarget
	State      healthState
	StatusCode int
	Latency    time.Duration
	Err        error
	At         time.Time
}

// healthCheckTimeout is the per-target ceiling. Five seconds is long
// enough to ride out a slow handshake on a busy network without
// holding the user hostage if a server is genuinely unreachable.
const healthCheckTimeout = 5 * time.Second

// healthCheckPollInterval is the cadence at which we re-probe each
// target while the Connection pane is open. Thirty seconds is a
// usability/respect tradeoff: long enough to avoid hammering third
// parties when a user lingers on the pane, short enough that a
// transient outage clears within one poll after it resolves.
const healthCheckPollInterval = 30 * time.Second

// healthCheckMinInterval rate-limits manual refreshes so a user
// holding down `r` can't hammer external services.
const healthCheckMinInterval = 3 * time.Second

// DefaultHealthTargets returns the canonical list of services the
// CLI calls during normal operation. Caller passes in the resolved
// API and keyring URLs from config so we honor user overrides.
//
// For the Truestamp service we point at the dedicated /health
// endpoint rather than the api_url itself: the server returns 200
// only when the database is reachable AND the app is up, so a green
// row here is a much stronger signal than "any HTTP response from
// the api_url host". /health is derived from the api_url's host so
// users running against staging or a local preview server still get
// a meaningful probe.
//
// Third-party verification endpoints (NIST, Stellar, Blockstream)
// use the production URLs by design — we don't have testnet URLs
// in the CLI's config surface, and a user hitting a Truestamp
// mainnet item would always pull from the production third parties
// anyway.
//
// Exported because cmd/console.go calls it to populate
// console.Options before invoking console.Run.
func DefaultHealthTargets(apiURL, keyringURL string) []HealthTarget {
	targets := []HealthTarget{}
	if healthURL := deriveHealthURL(apiURL); healthURL != "" {
		targets = append(targets, HealthTarget{Name: "Truestamp service", URL: healthURL})
	}
	if keyringURL != "" {
		targets = append(targets, HealthTarget{Name: "Truestamp keyring", URL: keyringURL})
	}
	targets = append(targets,
		HealthTarget{Name: "NIST randomness beacon", URL: "https://beacon.nist.gov/beacon/2.0"},
		HealthTarget{Name: "Stellar network (verification)", URL: "https://horizon.stellar.org"},
		HealthTarget{Name: "Bitcoin network (verification)", URL: "https://blockstream.info/api/blocks/tip/height"},
	)
	return targets
}

// deriveHealthURL maps the user's api_url to the corresponding
// /health endpoint by replacing the path. The /health route is
// mounted at the server root in TruestampWeb.Router (above the
// /api/json scope), so the transformation is uniform across all
// deployments:
//
//	https://www.truestamp.com/api/json → https://www.truestamp.com/health
//	http://localhost:4010/api/json    → http://localhost:4010/health
//	https://staging.example.com/v1     → https://staging.example.com/health
//
// Returns "" if the input doesn't parse or lacks a scheme/host —
// the caller suppresses the row in that case so the pane doesn't
// surface a meaningless probe.
func deriveHealthURL(apiURL string) string {
	if apiURL == "" {
		return ""
	}
	u, err := url.Parse(apiURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.Path = "/health"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// checkHealthTarget performs a single probe and returns a populated
// result. It uses the package-shared httpclient so it inherits the
// configured timeout AND the User-Agent header, but layers a
// per-call deadline on top so a slow target can't outlive
// healthCheckTimeout regardless of the global setting.
//
// Probe shape: HEAD first. If the server returns 405 Method Not
// Allowed (common on simple JSON endpoints), retry with GET. We
// don't read the body — we only care that the host responded.
func checkHealthTarget(ctx context.Context, t HealthTarget) healthResult {
	r := healthResult{Target: t, At: time.Now()}

	if _, err := url.Parse(t.URL); err != nil {
		r.State = healthFailed
		r.Err = fmt.Errorf("invalid URL: %w", err)
		return r
	}

	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	start := time.Now()
	statusCode, err := probeOnce(checkCtx, http.MethodHead, t.URL)
	if err == nil && statusCode == http.StatusMethodNotAllowed {
		statusCode, err = probeOnce(checkCtx, http.MethodGet, t.URL)
	}
	r.Latency = time.Since(start).Round(time.Millisecond)

	// We're probing for liveness ("is this host reachable from
	// here?"), not for endpoint correctness. Any HTTP response
	// (200, 301, 404, 418, …) means the host is up and answering;
	// only 5xx counts as degraded because that's the server
	// signalling its own ill health. This matches what the user
	// actually wants to know on this pane: "is my internet down,
	// or is the service down?"
	switch {
	case err != nil:
		r.State = healthFailed
		r.Err = err
	case statusCode >= 500:
		r.State = healthDegraded
		r.StatusCode = statusCode
	default:
		r.State = healthOK
		r.StatusCode = statusCode
	}
	return r
}

// probeOnce issues a single HTTP request and returns the status code
// (or an error). The body is discarded — we don't care about content
// for liveness probing.
func probeOnce(ctx context.Context, method, target string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return 0, err
	}
	resp, err := httpclient.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}

// sortHealthResults orders results so problems surface first
// (failed → degraded → checking → unknown → ok) and then by name
// within each bucket for a deterministic render order.
func sortHealthResults(rs []healthResult) {
	priority := map[healthState]int{
		healthFailed:   0,
		healthDegraded: 1,
		healthChecking: 2,
		healthUnknown:  3,
		healthOK:       4,
	}
	sort.SliceStable(rs, func(i, j int) bool {
		pi, pj := priority[rs[i].State], priority[rs[j].State]
		if pi != pj {
			return pi < pj
		}
		return rs[i].Target.Name < rs[j].Target.Name
	})
}

// runHealthCheckCmd returns a tea.Cmd that probes a single target on
// a background goroutine and emits a healthCheckResultMsg with the
// outcome. The Bubble Tea runtime invokes it asynchronously; we
// fan out one Cmd per target so the pane updates row-by-row as
// each request returns rather than waiting for the slowest target.
func runHealthCheckCmd(ctx context.Context, index int, target HealthTarget) tea.Cmd {
	return func() tea.Msg {
		return healthCheckResultMsg{
			Index:  index,
			Result: checkHealthTarget(ctx, target),
		}
	}
}

// healthCheckTickCmd schedules a healthCheckTickMsg after
// healthCheckPollInterval. The Connection pane re-arms this command
// each time it processes a tick while it is the active pane;
// tabbing away breaks the chain and pauses polling, which is
// exactly the "while the page is open" semantic we want.
func healthCheckTickCmd() tea.Cmd {
	return tea.Tick(healthCheckPollInterval, func(time.Time) tea.Msg {
		return healthCheckTickMsg{}
	})
}
