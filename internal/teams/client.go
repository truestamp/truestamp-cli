// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package teams is a thin client for the Truestamp Teams + Memberships
// JSON:API surfaces (GET /api/json/teams, /api/json/memberships). It
// exposes the small subset of operations the CLI needs to discover a
// user's team memberships and validate that a configured team is
// accessible.
//
// The membership read policy on the server side filters to the actor's
// own memberships, so `ListMyMemberships` returns "the teams I'm a
// member of" with no extra filtering. See
// truestamp-v2/lib/truestamp/teams/membership.ex policy block.
package teams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// Team is the subset of a JSON:API team resource the CLI consumes.
type Team struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Personal       bool   `json:"personal"`
	OwnershipModel string `json:"ownership_model"`
	CreatedAt      string `json:"inserted_at"`
}

// Membership pairs a user's role with the team it applies to. Team
// is populated from a parallel `GET /teams` call, joined client-side
// (see ListMyMemberships). JSON tags are snake_case to match the
// shape of every other --json surface in the CLI; without explicit
// tags the Go field names ("ID", "TeamID") would leak.
type Membership struct {
	ID     string `json:"id"`
	TeamID string `json:"team_id"`
	Role   string `json:"role"` // "team_owner" | "team_admin" | "team_member" | "team_viewer"
	Team   *Team  `json:"team,omitempty"`
}

// Errors surfaced by the client. CLI layers may errors.Is these to pick
// an exit code and user-facing message.
//
// ErrUnauthorized covers 401 (the API key itself is invalid or expired).
// ErrForbidden covers 403 — auth was accepted but the actor isn't allowed
// to read this resource (typically: the tenant header points to a team
// the actor isn't a member of). Distinguishing the two matters because
// the user-facing remediation is completely different: 401 → run `auth
// login`; 403 → check the team id, ask for membership.
var (
	ErrUnauthorized = errors.New("not authenticated")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrBadRequest   = errors.New("bad request")
	ErrRateLimited  = errors.New("rate limited")
	ErrServer       = errors.New("server error")
)

// APIError carries HTTP status + preserved `errors[].detail` from the
// JSON:API envelope for display to the user. Wraps one of the sentinel
// errors above so callers can errors.Is() the class while still showing
// the detail text.
type APIError struct {
	Status     int
	Detail     string // preferred; falls back to Title
	RetryAfter string // verbatim Retry-After header on 429
	sentinel   error
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("HTTP %d: %s", e.Status, e.Detail)
	}
	return fmt.Sprintf("HTTP %d", e.Status)
}

func (e *APIError) Unwrap() error { return e.sentinel }

// Config carries the subset of runtime configuration needed for a request.
// Kept small to avoid importing the top-level config package. The
// credential is supplied out-of-band by the process-wide [auth.Authorizer]
// (set in cmd/root); only the tenant scoping lives here.
type Config struct {
	APIURL string // e.g. https://www.truestamp.com/api/json
	Team   string // optional tenant id; sent verbatim as the `tenant` header
}

// ListMyMemberships returns one Membership row per team the API
// key's user has access to, with the user's role on that team.
//
// The implementation anchors on `GET /teams` (source of truth for
// "teams I can read", per the server-side `relates_to_actor_via(:members)`
// READ policy) rather than `/memberships` because:
//
//  1. `/memberships` can return rows whose team_id no longer exists
//     (orphaned dev seed data, mid-cascade-delete races).
//  2. Under admin bypass policies, `/memberships` may return
//     memberships from other users that the actor doesn't actually
//     hold. Anchoring on the team list ensures every returned row
//     represents a team the actor can actually read.
//  3. `?include=team` on the memberships endpoint is unreliable —
//     the included array is sparse for reasons we haven't fully
//     diagnosed (possibly Ash's per-response include limits or
//     relationship-load authorization filtering).
//
// Roles are joined from a parallel `/memberships` request and
// deduplicated by team_id (when multiple memberships exist for the
// same team — e.g. different users on the same team under admin
// bypass — we take the first match deterministically).
//
// Both endpoints walk `links.next` so users with many memberships
// or teams aren't silently truncated by the server's default page size.
func ListMyMemberships(ctx context.Context, cfg Config) ([]Membership, error) {
	teamsByID, err := listReadableTeams(ctx, cfg)
	if err != nil {
		return nil, err
	}

	memberships, err := listMembershipsRaw(ctx, cfg)
	if err != nil {
		// Soft-fail the role lookup: return one row per team with an
		// empty role so the user still sees their teams. Better than
		// erroring the whole listing on a transient /memberships
		// outage.
		return teamsToMembershipsNoRole(teamsByID), nil
	}

	// Build team_id → first matching membership lookup.
	roleByTeam := make(map[string]string, len(memberships))
	memIDByTeam := make(map[string]string, len(memberships))
	for _, m := range memberships {
		if _, seen := roleByTeam[m.TeamID]; seen {
			continue
		}
		if _, valid := teamsByID[m.TeamID]; !valid {
			// Drop memberships whose team isn't readable — these are
			// either orphaned (team deleted) or admin-bypass leakage
			// from other users' teams. The team list is the source
			// of truth.
			continue
		}
		roleByTeam[m.TeamID] = m.Role
		memIDByTeam[m.TeamID] = m.ID
	}

	out := make([]Membership, 0, len(teamsByID))
	for id, t := range teamsByID {
		out = append(out, Membership{
			ID:     memIDByTeam[id],
			TeamID: id,
			Role:   roleByTeam[id],
			Team:   t,
		})
	}
	// Sort by team id so callers see a stable order — the map
	// iteration above is non-deterministic. The CLI's own renderers
	// sort by privilege rank afterwards; this just gives JSON
	// output and tests a reproducible baseline.
	sort.Slice(out, func(i, j int) bool { return out[i].TeamID < out[j].TeamID })
	return out, nil
}

// teamsToMembershipsNoRole is the soft-fail fallback when the
// /memberships endpoint is unreachable but /teams succeeded. Returns
// one row per team with empty Role, sorted by team id for stable
// callers.
func teamsToMembershipsNoRole(teamsByID map[string]*Team) []Membership {
	out := make([]Membership, 0, len(teamsByID))
	for id, t := range teamsByID {
		out = append(out, Membership{TeamID: id, Team: t})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TeamID < out[j].TeamID })
	return out
}

// listMembershipsRaw walks the `/memberships` paginated endpoint and
// returns every page concatenated. No include — see the note on
// ListMyMemberships for why.
func listMembershipsRaw(ctx context.Context, cfg Config) ([]Membership, error) {
	pageSize := 200
	path := fmt.Sprintf("/memberships?fields[membership]=role,team_id&page[limit]=%d", pageSize)
	var out []Membership
	if err := walkPages(ctx, cfg, path, func(body []byte) (string, error) {
		batch, next, err := parseMembershipsPage(body)
		if err != nil {
			return "", err
		}
		out = append(out, batch...)
		return next, nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

// listReadableTeams walks the `/teams` paginated endpoint and returns
// a map keyed by team id, ready for client-side joining onto
// memberships. The team READ policy on the server filters this to
// teams the actor is a member of, so for a normal user this is
// exactly the set of teams they need.
func listReadableTeams(ctx context.Context, cfg Config) (map[string]*Team, error) {
	pageSize := 200
	path := fmt.Sprintf("/teams?fields[team]=name,personal,ownership_model,inserted_at&page[limit]=%d", pageSize)
	out := make(map[string]*Team)
	if err := walkPages(ctx, cfg, path, func(body []byte) (string, error) {
		batch, next, err := parseTeamsPage(body)
		if err != nil {
			return "", err
		}
		for _, t := range batch {
			out[t.ID] = t
		}
		return next, nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

// walkPages drives a paginated GET loop. Each fetched page is handed
// to onPage which returns the URL of the next page (relative or
// absolute) or "" when the walk is done. Capped at maxPages so a
// runaway server doesn't loop forever; 50 pages × 200 rows = 10k
// rows, more than enough for any realistic CLI use.
const maxPages = 50

func walkPages(ctx context.Context, cfg Config, firstPath string, onPage func([]byte) (string, error)) error {
	path := firstPath
	for page := 0; page < maxPages; page++ {
		body, err := doGet(ctx, cfg, path)
		if err != nil {
			return err
		}
		next, err := onPage(body)
		if err != nil {
			return err
		}
		if next == "" {
			return nil
		}
		// links.next from JSON:API is a full URL. Strip the base so
		// doGet can re-attach it.
		path = stripAPIBase(cfg.APIURL, next)
	}
	return nil
}

// stripAPIBase converts a full URL from `links.next` back into a path
// relative to cfg.APIURL so doGet can compose it. If `full` doesn't
// share the cfg.APIURL prefix (unexpected — would only happen if the
// server returns a different host), we return the URL as-is and let
// doGet handle it.
func stripAPIBase(apiURL, full string) string {
	if strings.HasPrefix(full, apiURL) {
		return strings.TrimPrefix(full, apiURL)
	}
	return full
}

// parseMembershipsPage decodes one page of memberships plus the
// `links.next` cursor. Used by the pagination walker.
func parseMembershipsPage(body []byte) ([]Membership, string, error) {
	var doc struct {
		Data  []jsonAPIResource `json:"data"`
		Links jsonAPILinks      `json:"links"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, "", fmt.Errorf("parsing memberships page: %w", err)
	}
	out := make([]Membership, 0, len(doc.Data))
	for i := range doc.Data {
		r := &doc.Data[i]
		if r.Type != "membership" {
			continue
		}
		role, _ := decodeStringAttr(r.Attributes, "role")
		teamID, _ := decodeStringAttr(r.Attributes, "team_id")
		if teamID == "" {
			if rel, ok := r.Relationships["team"]; ok && rel.Data != nil {
				teamID = rel.Data.ID
			}
		}
		out = append(out, Membership{
			ID:     r.ID,
			TeamID: teamID,
			Role:   role,
		})
	}
	return out, doc.Links.Next, nil
}

// parseTeamsPage decodes one page of teams plus the `links.next`
// cursor.
func parseTeamsPage(body []byte) ([]*Team, string, error) {
	var doc struct {
		Data  []jsonAPIResource `json:"data"`
		Links jsonAPILinks      `json:"links"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, "", fmt.Errorf("parsing teams page: %w", err)
	}
	out := make([]*Team, 0, len(doc.Data))
	for i := range doc.Data {
		r := &doc.Data[i]
		if r.Type != "team" {
			continue
		}
		t, err := teamFromResource(r)
		if err != nil {
			return nil, "", fmt.Errorf("team %s: %w", r.ID, err)
		}
		out = append(out, t)
	}
	return out, doc.Links.Next, nil
}

// jsonAPILinks captures the subset of the JSON:API top-level `links`
// object the CLI uses for pagination. `next` is `null` when there
// are no more pages — UnmarshalJSON on the wrapper type coerces it
// to an empty string so the walk loop can use a single `next == ""`
// terminator.
type jsonAPILinks struct {
	Next string
}

func (l *jsonAPILinks) UnmarshalJSON(data []byte) error {
	var raw struct {
		Next *string `json:"next"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Next != nil {
		l.Next = *raw.Next
	}
	return nil
}

// GetTeam fetches a single team by id. Useful for `truestamp team set
// <id>` validation: a 2xx confirms the user can read it (which by
// policy means they have a membership), 4xx surfaces the JSON:API
// detail string.
func GetTeam(ctx context.Context, cfg Config, id string) (*Team, error) {
	if id == "" {
		return nil, fmt.Errorf("team id is empty")
	}
	path := "/teams/" + url.PathEscape(id) +
		"?fields[team]=name,personal,ownership_model,inserted_at"
	body, err := doGet(ctx, cfg, path)
	if err != nil {
		return nil, err
	}
	return parseTeam(body)
}

// GetMyRoleOnTeam returns the role string for the API key's user on
// the given team, or the empty string if the user has no membership
// (without distinguishing the "no membership" case from "lookup
// failed"). Callers that need the failure distinction should call
// ListMyMemberships and search the slice.
func GetMyRoleOnTeam(ctx context.Context, cfg Config, teamID string) (string, error) {
	memberships, err := ListMyMemberships(ctx, cfg)
	if err != nil {
		return "", err
	}
	for _, m := range memberships {
		if m.TeamID == teamID {
			return m.Role, nil
		}
	}
	return "", nil
}

// doGet issues an authenticated GET and returns the response body on 2xx.
// On non-2xx returns an *APIError wrapping one of the class sentinels.
func doGet(ctx context.Context, cfg Config, path string) ([]byte, error) {
	if auth.Default().Mode() == auth.ModeNone {
		return nil, &APIError{Status: 401, Detail: "not authenticated", sentinel: ErrUnauthorized}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.APIURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	if err := auth.AuthorizeRequest(ctx, req); err != nil {
		return nil, &APIError{Status: 401, Detail: err.Error(), sentinel: ErrUnauthorized}
	}
	if cfg.Team != "" {
		req.Header.Set("tenant", cfg.Team)
	}

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading API response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}

	apiErr := parseAPIError(resp.StatusCode, body)
	if resp.StatusCode == http.StatusTooManyRequests {
		apiErr.RetryAfter = resp.Header.Get("Retry-After")
	}
	return nil, apiErr
}

func parseAPIError(status int, body []byte) *APIError {
	e := &APIError{Status: status, sentinel: sentinelFor(status)}
	var envelope struct {
		Errors []struct {
			Detail string `json:"detail"`
			Title  string `json:"title"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		switch {
		case first.Detail != "":
			e.Detail = first.Detail
		case first.Title != "":
			e.Detail = first.Title
		}
	}
	if e.Detail == "" && len(body) > 0 && body[0] == '<' {
		e.Detail = "server returned HTML error page"
	}
	if e.Detail == "" {
		e.Detail = httpclient.Truncate(string(body), 200)
	}
	return e
}

func sentinelFor(status int) error {
	switch {
	case status == http.StatusUnauthorized:
		return ErrUnauthorized
	case status == http.StatusForbidden:
		return ErrForbidden
	case status == http.StatusNotFound:
		return ErrNotFound
	case status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status >= 400 && status < 500:
		return ErrBadRequest
	case status >= 500:
		return ErrServer
	}
	return errors.New("unexpected status")
}

type jsonAPISingle struct {
	Data jsonAPIResource `json:"data"`
}

type jsonAPIResource struct {
	Type          string                            `json:"type"`
	ID            string                            `json:"id"`
	Attributes    map[string]json.RawMessage        `json:"attributes"`
	Relationships map[string]jsonAPIRelationshipObj `json:"relationships"`
}

type jsonAPIRelationshipObj struct {
	Data *jsonAPIResourceID `json:"data"`
}

type jsonAPIResourceID struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// parseMembershipsWithIncluded is retained as a thin alias over
// parseMembershipsPage so the fuzz corpus and the legacy
// AttributeFallbackForTeamID test keep covering the parser. The
// production data path now goes through parseMembershipsPage +
// parseTeamsPage; this function ignores `included` because the
// production code no longer requests it.
func parseMembershipsWithIncluded(body []byte) ([]Membership, error) {
	out, _, err := parseMembershipsPage(body)
	return out, err
}

func parseTeam(body []byte) (*Team, error) {
	var single jsonAPISingle
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, fmt.Errorf("parsing team response: %w", err)
	}
	if single.Data.ID == "" {
		return nil, fmt.Errorf("malformed team response: missing data.id")
	}
	return teamFromResource(&single.Data)
}

func teamFromResource(r *jsonAPIResource) (*Team, error) {
	name, _ := decodeStringAttr(r.Attributes, "name")
	createdAt, _ := decodeStringAttr(r.Attributes, "inserted_at")
	ownership, _ := decodeStringAttr(r.Attributes, "ownership_model")

	var personal bool
	if raw, ok := r.Attributes["personal"]; ok {
		// Tolerate both bool and null; default false.
		_ = json.Unmarshal(raw, &personal)
	}

	return &Team{
		ID:             r.ID,
		Name:           name,
		Personal:       personal,
		OwnershipModel: ownership,
		CreatedAt:      createdAt,
	}, nil
}

func decodeStringAttr(m map[string]json.RawMessage, key string) (string, error) {
	raw, ok := m[key]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("attribute %q is not a string: %w", key, err)
	}
	return s, nil
}

// FormatRole returns a human-friendly title-cased role label suitable
// for table cells: "team_owner" -> "Owner", "team_admin" -> "Admin",
// "team_member" -> "Member", "team_viewer" -> "Viewer". Unknown
// inputs pass through unchanged so a future server-side role addition
// shows up verbatim instead of being silently dropped.
func FormatRole(role string) string {
	switch role {
	case "team_owner":
		return "Owner"
	case "team_admin":
		return "Admin"
	case "team_member":
		return "Member"
	case "team_viewer":
		return "Viewer"
	}
	return role
}

// PrivilegeRank returns a sort key for the given role. Lower values
// rank higher (Owner = 0, Viewer = 3). Unknown roles sort last so a
// future server-side addition isn't silently grouped with viewers.
func PrivilegeRank(role string) int {
	switch role {
	case "team_owner":
		return 0
	case "team_admin":
		return 1
	case "team_member":
		return 2
	case "team_viewer":
		return 3
	}
	return 4
}
