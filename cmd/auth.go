// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"charm.land/huh/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Truestamp authentication",
	Long: `Sign in to Truestamp.

By default 'auth login' opens your browser for a secure OAuth sign-in
(Authorization Code + PKCE). The resulting session uses a short-lived
access token that the CLI refreshes automatically, and the refresh token
is stored in your OS keychain (with a 0600 file fallback).

For CI and other headless environments, set a long-lived API key via the
TRUESTAMP_API_KEY env var or the --api-key flag; an explicitly-provided API
key takes precedence over an OAuth session. 'auth login --api-key' stores a
key in your config file interactively.`,
}

var authLoginAPIKey bool

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in via your browser (OAuth), or store an API key with --api-key",
	Long: `Sign in to Truestamp.

Default: opens your browser to authorize the CLI (OAuth 2.1 Authorization
Code + PKCE). On success a session is stored in your OS keychain (0600 file
fallback) and refreshed automatically.

--api-key: instead prompts for a long-lived API key and stores it in the
config file with 0600 permissions (the headless/CI path).`,
	Args: cobra.NoArgs,
	RunE: runAuthLogin,
}

var authLogoutAPIKey bool

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out, revoke the OAuth session and/or remove the stored API key",
	Args:  cobra.NoArgs,
	RunE:  runAuthLogout,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report the active credential and validate it against the API",
	Long: `Report which credential is active (OAuth session or API key) and
validate it by calling the Truestamp API. This is inherently an online
operation; no offline mode is offered.

Exit codes:
  0  valid (a credential is active and accepted by the API)
  1  no credential, invalid credential, or network error

--json emits one record: "ok", and on failure a "reason" identifier
(not_authenticated, api_unreachable, credential_rejected,
unexpected_api_response, team_lookup_failed, team_not_accessible) that a
script can branch on.`,
	Args: cobra.NoArgs,
	RunE: runAuthStatus,
}

func init() {
	authLoginCmd.Flags().BoolVar(&authLoginAPIKey, "api-key", false, "Store a long-lived API key interactively instead of OAuth sign-in")
	authLogoutCmd.Flags().BoolVar(&authLogoutAPIKey, "api-key", false, "Remove the stored API key (in addition to clearing any OAuth session)")
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	addRecordOutputFlags(authStatusCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.GroupID = groupSetup
	rootCmd.AddCommand(asGroup(authCmd))
}

// runAuthLogin dispatches to the browser OAuth flow (default) or the
// interactive API-key paste path (--api-key).
func runAuthLogin(cmd *cobra.Command, _ []string) error {
	if authLoginAPIKey {
		return runAPIKeyLogin(cmd)
	}
	return runOAuthLogin(cmd)
}

// runOAuthLogin runs the browser-based loopback OAuth flow.
func runOAuthLogin(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	cfg := appConfig
	store := auth.NewStore(cfg.BaseURL)

	// The browser flow needs an interactive session to be useful, but we
	// don't hard-require a TTY: the URL is printed so a user on a headless
	// box can copy it. We do warn if an explicit API key will shadow the
	// session for actual API calls.
	if cfg.APIKeyExplicit {
		ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
			"note: TRUESTAMP_API_KEY/--api-key is set and takes precedence over an OAuth session for API calls."))
	}

	ui.Fprintln(out, ui.HeaderBox("Truestamp Sign-In", "Authorizing via your browser"))
	ui.Fprintln(out)

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	sess, err := auth.Login(ctx, cfg.BaseURL, store, auth.LoginOptions{Out: out})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	appLogger.Info("auth_login_oauth", "issuer", sess.Issuer, "scope", sess.Scope)

	// Re-resolve the process authorizer so subsequent calls (and the probe
	// below) see the new session.
	auth.SetDefault(auth.Resolve(
		auth.Credentials{APIKey: cfg.APIKey, APIKeyExplicit: cfg.APIKeyExplicit}, store))

	// Best-effort identity probe using the freshly-minted OAuth session
	// specifically (ignoring any api-key precedence) so the confirmation
	// reflects who you just signed in as.
	probe := auth.Resolve(auth.Credentials{}, store)
	identity := ""
	if res, perr := checkAuth(ctx, probe, cfg.APIURL, cfg.Team); perr == nil && res.ok {
		identity = formatUserIdentity(res)
	}

	labelStyle := lipgloss.NewStyle().Foreground(ui.Label)
	if identity != "" {
		ui.Fprintln(out, ui.SuccessBanner("Signed in as "+identity))
	} else {
		ui.Fprintln(out, ui.SuccessBanner("Signed in"))
	}
	if sess.Scope != "" {
		ui.Fprintln(out, labelStyle.Render("    Scopes:  "+sess.Scope))
	}
	if !sess.Expiry.IsZero() {
		ui.Fprintln(out, labelStyle.Render("    Expires: "+sess.Expiry.Local().Format(time.RFC1123)+" (auto-refreshed)"))
	}
	ui.Fprintln(out, labelStyle.Render("    Stored:  "+store.Location()))
	return nil
}

// runAPIKeyLogin is the legacy interactive paste-a-key path (--api-key).
func runAPIKeyLogin(cmd *cobra.Command) error {
	if !inputsrc.IsStdinTerminal() {
		return fmt.Errorf("auth login --api-key requires an interactive terminal (set TRUESTAMP_API_KEY directly in CI)")
	}

	keysURL, err := apiKeysURL(appConfig.APIURL)
	if err != nil {
		return err
	}

	labelStyle := lipgloss.NewStyle().Foreground(ui.Label)
	valueStyle := lipgloss.NewStyle().Foreground(ui.Value).Bold(true)
	accent := lipgloss.NewStyle().Foreground(ui.Accent)

	out := cmd.OutOrStdout()
	ui.Fprintln(out)
	ui.Fprintln(out, "  "+accent.Render("Create and copy a new API key at:"))
	ui.Fprintln(out, "    "+valueStyle.Render(keysURL))
	ui.Fprintln(out, "  "+labelStyle.Render("    (existing keys cannot be copied, create a new one now)"))
	ui.Fprintln(out)
	ui.Fprintln(out, "  "+labelStyle.Render("Then paste the key below. Input is hidden."))
	ui.Fprintln(out)

	var apiKey string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("API Key").
				EchoMode(huh.EchoModePassword).
				Value(&apiKey).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("API key cannot be empty")
					}
					return nil
				}),
		),
	).WithTheme(ui.HuhTheme()).Run()
	if err != nil {
		return fmt.Errorf("API key input: %w", err)
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("no API key provided")
	}

	if err := config.SetAPIKey(apiKey); err != nil {
		return err
	}

	// Log the action, never the key bytes.
	appLogger.Info("auth_login_apikey", "config_path", config.ActivePath())
	ui.Fprintln(out, ui.SuccessBanner("API key saved to "+config.ActivePath()))
	return nil
}

// runAuthLogout revokes any OAuth session and optionally clears a stored
// API key. By default it clears whatever credential is active; --api-key
// additionally removes the stored config-file key.
func runAuthLogout(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	cfg := appConfig
	store := auth.NewStore(cfg.BaseURL)

	_, oauthErr := store.Load()
	hasOAuth := oauthErr == nil
	hasFileKey := cfg.APIKey != "" && !cfg.APIKeyExplicit

	if !hasOAuth && !hasFileKey {
		if cfg.APIKeyExplicit {
			ui.Fprintln(out, ui.FaintStyle().Render(
				"  Authenticated via TRUESTAMP_API_KEY/--api-key, nothing is stored to clear."))
			ui.Fprintln(out, ui.FaintStyle().Render(
				"  Unset it in your environment to sign out."))
		} else {
			ui.Fprintln(out, ui.FaintStyle().Render("  Not logged in."))
		}
		return nil
	}

	if inputsrc.IsStdinTerminal() {
		var confirmed bool
		desc := logoutDescription(hasOAuth, hasFileKey || authLogoutAPIKey)
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Sign out?").
					Description(desc).
					Affirmative("Yes, sign out").
					Negative("Cancel").
					Value(&confirmed),
			),
		).WithTheme(ui.HuhTheme()).Run(); err != nil {
			return fmt.Errorf("confirmation: %w", err)
		}
		if !confirmed {
			ui.Fprintln(out, ui.FaintStyle().Render("  Cancelled."))
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
	defer cancel()

	if hasOAuth {
		revoked, err := auth.Logout(ctx, store)
		if err != nil {
			return fmt.Errorf("clearing OAuth session: %w", err)
		}
		appLogger.Info("auth_logout_oauth", "revoked", revoked)
		if revoked {
			ui.Fprintln(out, ui.SuccessBanner("Signed out, OAuth session revoked and cleared"))
		} else {
			ui.Fprintln(out, ui.SuccessBanner("Signed out, OAuth session cleared (server revocation best-effort)"))
		}
	}

	if hasFileKey || authLogoutAPIKey {
		if err := config.SetAPIKey(""); err != nil {
			return err
		}
		appLogger.Info("auth_logout_apikey", "config_path", config.ActivePath())
		ui.Fprintln(out, ui.SuccessBanner("Stored API key removed from "+config.ActivePath()))
	} else if hasOAuth && cfg.APIKey != "" {
		ui.Fprintln(out, ui.FaintStyle().Render(
			"  Note: a config-file API key is still set; run 'truestamp auth logout --api-key' to remove it."))
	}

	auth.SetDefault(auth.Resolve(
		auth.Credentials{APIKey: "", APIKeyExplicit: cfg.APIKeyExplicit}, store))
	return nil
}

func logoutDescription(hasOAuth, clearsKey bool) string {
	switch {
	case hasOAuth && clearsKey:
		return "Revoke the OAuth session and remove the stored API key."
	case hasOAuth:
		return "Revoke and clear the OAuth session."
	default:
		return "Clear the stored API key in " + config.ActivePath() + "."
	}
}

// authStatusRecord is the --json shape of `auth status`. `ok` is the same
// answer as the exit code, so a caller can branch on either. `reason` is a
// stable identifier, never a sentence: an agent branches on it and a human
// reads `message`.
type authStatusRecord struct {
	OK          bool   `json:"ok"`
	Reason      string `json:"reason,omitempty"`
	Message     string `json:"message,omitempty"`
	ConfigFile  string `json:"config_file"`
	APIURL      string `json:"api_url"`
	AuthMode    string `json:"auth_mode"`
	Scopes      string `json:"scopes,omitempty"`
	TokenExpiry string `json:"token_expiry,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	Email       string `json:"email,omitempty"`
	FullName    string `json:"full_name,omitempty"`
	TeamID      string `json:"team_id,omitempty"`
	TeamName    string `json:"team_name,omitempty"`
	TeamRole    string `json:"team_role,omitempty"`
	HTTPStatus  int    `json:"http_status,omitempty"`
}

// finishAuthStatus is the single exit point for the JSON and silent
// renderings. Keeping it separate from the text path means the two cannot
// drift on which outcomes count as failures: both derive from rec.OK.
func finishAuthStatus(cmd *cobra.Command, rec authStatusRecord) (handled bool, err error) {
	jsonOut, silent := outputMode(cmd)
	switch {
	case silent:
		if !rec.OK {
			return true, errSilentFail
		}
		return true, nil
	case jsonOut:
		if werr := emitJSON(cmd.OutOrStdout(), rec); werr != nil {
			return true, werr
		}
		if !rec.OK {
			return true, errSilentFail
		}
		return true, nil
	}
	return false, nil
}

func runAuthStatus(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	cfg := appConfig
	apiURL := cfg.APIURL
	checkURL := apiURL + "/users"
	azr := auth.Default()

	labelStyle := lipgloss.NewStyle().Foreground(ui.Label)

	// The record is built alongside the table so the two renderings can
	// never disagree about what was found. Every early return below fills
	// in a reason and goes through finishAuthStatus first.
	rec := authStatusRecord{
		ConfigFile: config.ActivePath(),
		APIURL:     apiURL,
		AuthMode:   authModeDisplay(),
	}

	t := ui.CompactTable().
		StyleFunc(configStyleFunc).
		Row("Config File", config.ActivePath()).
		Row("API URL", apiURL).
		Row("Check URL", checkURL).
		Row("Auth Mode", authModeDisplay())

	// OAuth session detail rows, only when OAuth is the ACTIVE credential,
	// so the displayed scope/expiry always matches the Auth Mode (an
	// explicit API key can override a still-stored session).
	store := auth.NewStore(cfg.BaseURL)
	if azr.Mode() == auth.ModeOAuth {
		if sess, serr := store.Load(); serr == nil {
			if sess.Scope != "" {
				t = t.Row("Scopes", sess.Scope)
				rec.Scopes = sess.Scope
			}
			rec.TokenExpiry = formatTokenExpiry(sess.Expiry)
			t = t.Row("Token Expiry", rec.TokenExpiry)
		}
	}
	if cfg.APIKey != "" {
		rec.APIKey = maskAPIKey(cfg.APIKey)
		t = t.Row("API Key", rec.APIKey)
	}
	t = t.Row("Team In Scope", teamInScope(cfg.Team))
	rec.TeamID = cfg.Team

	// textHeader prints the card that precedes every text-mode outcome.
	// Deferred until after the first finishAuthStatus call so --json and
	// --silent emit nothing before their own output.
	textHeader := func() {
		ui.Fprintln(out, ui.HeaderBox("Truestamp Auth Status", "Validating with the API"))
		ui.Fprintln(out)
		ui.Fprintln(out, t.String())
	}

	// fail is the single failure epilogue. The JSON and silent renderings
	// go first, because they derive from rec and print nothing else; the
	// text rendering is the card, a banner and the detail lines. Every
	// failure path ends with errSilentFail so Execute prints nothing more.
	fail := func(banner string, details ...string) error {
		if handled, err := finishAuthStatus(cmd, rec); handled {
			return err
		}
		textHeader()
		ui.Fprintln(out, ui.FailureBanner(banner))
		for _, d := range details {
			if d != "" {
				ui.Fprintln(out, labelStyle.Render("    "+d))
			}
		}
		return errSilentFail
	}

	if azr.Mode() == auth.ModeNone {
		rec.Reason = "not_authenticated"
		rec.Message = "Run 'truestamp auth login' to sign in (or set TRUESTAMP_API_KEY)."
		return fail("Not authenticated", rec.Message)
	}

	ctx := cmd.Context()

	userResult, err := checkAuth(ctx, azr, apiURL, cfg.Team)
	if err != nil {
		rec.Reason = "api_unreachable"
		rec.Message = err.Error()
		return fail("Could not reach the API", err.Error())
	}

	switch {
	case userResult.unauthorized:
		rec.Reason = "credential_rejected"
		rec.HTTPStatus = userResult.httpStatus
		rec.Message = userResult.message
		if rec.Message == "" {
			rec.Message = fmt.Sprintf("HTTP %d, run 'truestamp auth login' to re-authenticate.", userResult.httpStatus)
		}
		return fail("Credential rejected by the server", rec.Message)

	case !userResult.ok:
		rec.Reason = "unexpected_api_response"
		rec.HTTPStatus = userResult.httpStatus
		rec.Message = userResult.message
		return fail(fmt.Sprintf("Unexpected API response (HTTP %d)", userResult.httpStatus), userResult.message)
	}

	rec.UserID = userResult.userID
	rec.Email = userResult.email
	rec.FullName = userResult.fullName

	// Authenticated. Resolve the team when one is configured.
	var teamResult *teamCheckResult
	if cfg.Team != "" {
		teamResult, err = fetchTeam(ctx, azr, apiURL, cfg.Team)
		if err != nil {
			rec.Reason = "team_lookup_failed"
			rec.Message = err.Error()
			return fail("Could not look up team", err.Error())
		}
		if !teamResult.found {
			rec.Reason = "team_not_accessible"
			rec.HTTPStatus = teamResult.httpStatus
			rec.Message = teamResult.message
			if rec.Message == "" {
				rec.Message = fmt.Sprintf("HTTP %d, the team id may be wrong, or this user is not a member.", teamResult.httpStatus)
			}
			return fail("Team "+cfg.Team+" is not accessible", teamResult.message,
				fmt.Sprintf("HTTP %d, the team id may be wrong, or this user is not a member.", teamResult.httpStatus))
		}
		role, _ := teams.GetMyRoleOnTeam(ctx, teams.Config{
			APIURL: apiURL, Team: cfg.Team,
		}, cfg.Team)
		teamResult.role = role
		rec.TeamName = teamResult.name
		rec.TeamRole = role
	}

	rec.OK = true
	if handled, ferr := finishAuthStatus(cmd, rec); handled {
		return ferr
	}
	textHeader()
	ui.Fprintln(out, ui.SuccessBanner("Authenticated as "+formatUserIdentity(userResult)))
	for _, line := range formatTeamLines(cfg.Team, teamResult) {
		ui.Fprintln(out, labelStyle.Render("    "+line))
	}
	return nil
}

// formatUserIdentity renders "Full Name <email>", falling back to email,
// then user id, then a generic placeholder.
func formatUserIdentity(r *authCheckResult) string {
	switch {
	case r.fullName != "" && r.email != "":
		return fmt.Sprintf("%s <%s>", r.fullName, r.email)
	case r.email != "":
		return r.email
	case r.userID != "":
		return r.userID
	default:
		return "(identity not returned)"
	}
}

// Removed: formatTeam, the one-line "Name  [id]" renderer. formatTeamLines
// replaced it when `auth status` grew the separate Team Id / Team Name /
// Team Role rows that `config show` and `teams list` also render, and
// nothing had called the single-line form since. Its only remaining caller
// was its own test, which meant a change to the team display could not
// break it.

// formatTeamLines renders the team context as up-to-three lines for the
// success-banner subblock.
func formatTeamLines(teamID string, r *teamCheckResult) []string {
	if teamID == "" {
		return []string{"Team: personal team (no tenant header sent)"}
	}
	out := []string{"Team Id:   " + teamID}
	if r != nil && r.name != "" {
		name := r.name
		if r.personal {
			name += " (personal)"
		}
		out = append(out, "Team Name: "+name)
	}
	if r != nil && r.role != "" {
		out = append(out, "Team Role: "+teams.FormatRole(r.role))
	}
	return out
}

// formatTokenExpiry renders the access-token expiry for `auth status`.
// Access tokens are short-lived (~1h) and auto-refreshed, so a past
// timestamp is normal, annotate it so it doesn't read as a broken session.
func formatTokenExpiry(exp time.Time) string {
	switch {
	case exp.IsZero():
		return "(unknown), auto-refreshes on next use"
	case exp.Before(time.Now()):
		return "expired, auto-refreshes on next use"
	default:
		return exp.Local().Format(time.RFC1123) + " (auto-refreshed)"
	}
}

// authCheckResult summarizes the outcome of the /users probe.
type authCheckResult struct {
	ok           bool
	unauthorized bool
	httpStatus   int
	message      string
	userID       string
	email        string
	fullName     string
}

// teamCheckResult summarizes the outcome of the /teams/{id} probe.
type teamCheckResult struct {
	found      bool
	httpStatus int
	name       string
	personal   bool
	role       string
	message    string
}

// checkAuth sends GET {apiURL}/users authorized by azr and interprets the
// response. Returns a non-nil error only for transport-level failures and
// for a dead/unconfigured credential (azr.Authorize failing). All HTTP
// outcomes, 2xx, 4xx, 5xx, are reported in the result.
func checkAuth(ctx context.Context, azr auth.Authorizer, apiURL, team string) (*authCheckResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reqURL := apiURL + "/users?page[limit]=1&fields[user]=email,first_name,last_name,full_name"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	if err := azr.Authorize(ctx, req); err != nil {
		return nil, err
	}
	if team != "" {
		req.Header.Set("tenant", team)
	}

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	result := &authCheckResult{httpStatus: resp.StatusCode}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		result.ok = true
		result.userID, result.email, result.fullName = extractUserIdentity(body)
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		result.unauthorized = true
		result.message = extractAPIErrorMessage(body)
	default:
		result.message = extractAPIErrorMessage(body)
	}
	return result, nil
}

// fetchTeam sends GET {apiURL}/teams/{id} and reports whether the caller
// can see that team.
func fetchTeam(ctx context.Context, azr auth.Authorizer, apiURL, teamID string) (*teamCheckResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reqURL := apiURL + "/teams/" + url.PathEscape(teamID) + "?fields[team]=name,personal"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	if err := azr.Authorize(ctx, req); err != nil {
		return nil, err
	}
	req.Header.Set("tenant", teamID)

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	result := &teamCheckResult{httpStatus: resp.StatusCode}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.found = true
		result.name, result.personal = extractTeamAttrs(body)
		return result, nil
	}
	result.message = extractAPIErrorMessage(body)
	return result, nil
}

// extractUserIdentity pulls id, email, and a display name out of a
// JSON:API response whose `data` may be either a single resource or an
// array.
func extractUserIdentity(body []byte) (id, email, fullName string) {
	first := firstJSONAPIResource(body)
	if first == nil {
		return "", "", ""
	}
	email = stringAttr(first.Attributes, "email")
	fullName = stringAttr(first.Attributes, "full_name")
	if fullName == "" {
		fn := stringAttr(first.Attributes, "first_name")
		ln := stringAttr(first.Attributes, "last_name")
		fullName = strings.TrimSpace(fn + " " + ln)
	}
	return first.ID, email, fullName
}

// extractTeamAttrs pulls the team name and personal flag from a JSON:API
// single-resource response.
func extractTeamAttrs(body []byte) (name string, personal bool) {
	first := firstJSONAPIResource(body)
	if first == nil {
		return "", false
	}
	name = stringAttr(first.Attributes, "name")
	if v, ok := first.Attributes["personal"].(bool); ok {
		personal = v
	}
	return name, personal
}

// jsonAPIResource is the minimal JSON:API resource shape we consume.
type jsonAPIResource struct {
	ID         string         `json:"id"`
	Attributes map[string]any `json:"attributes"`
}

// firstJSONAPIResource tolerates both `data: {}` and `data: [{}, ...]`
// shapes and returns the first resource, or nil for malformed or empty
// responses.
func firstJSONAPIResource(body []byte) *jsonAPIResource {
	var single struct {
		Data jsonAPIResource `json:"data"`
	}
	if err := json.Unmarshal(body, &single); err == nil && single.Data.ID != "" {
		return &single.Data
	}
	var list struct {
		Data []jsonAPIResource `json:"data"`
	}
	if err := json.Unmarshal(body, &list); err == nil && len(list.Data) > 0 {
		return &list.Data[0]
	}
	return nil
}

// extractAPIErrorMessage returns the first JSON:API error's detail/title, or
// a short snippet of the body when it isn't a recognizable error envelope.
func extractAPIErrorMessage(body []byte) string {
	var envelope struct {
		Errors []struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		if first.Detail != "" {
			return first.Detail
		}
		if first.Title != "" {
			return first.Title
		}
	}
	s := strings.TrimSpace(string(body))
	if s == "" {
		return ""
	}
	if s[0] == '<' {
		return "server returned HTML error page"
	}
	return httpclient.Truncate(s, 200)
}

// teamInScope returns the tenant header value the CLI will send, or a
// placeholder noting that the personal team will be used when no team has
// been configured.
func teamInScope(team string) string {
	if team == "" {
		return "(personal team, set TRUESTAMP_TEAM or --team to override)"
	}
	return team
}

func stringAttr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// apiKeysURL derives the web app's API keys page from the API base URL.
func apiKeysURL(apiURL string) (string, error) {
	u, err := url.Parse(apiURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid api_url %q", apiURL)
	}
	return (&url.URL{Scheme: u.Scheme, Host: u.Host, Path: "/api-keys"}).String(), nil
}
