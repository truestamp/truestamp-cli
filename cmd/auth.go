// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"charm.land/huh/v2"
	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Truestamp API authentication",
	Long:  "Log in to store your Truestamp API key in the config file, or log out to remove it.",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Store an API key in the config file",
	Long: `Log in by pasting a Truestamp API key.

Visit the API keys page in the Truestamp web app, create a new key, and copy
it immediately — existing keys cannot be copied after creation. Paste the
new key into the interactive prompt. The key is stored in your local config
file with 0600 permissions. For security, there is no command-line flag to
accept the key — it must be pasted into the hidden-input prompt.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove the stored API key from the config file",
	Args:  cobra.NoArgs,
	RunE:  runAuthLogout,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report whether an API key is configured and valid",
	Long: `Report whether an API key is configured and validate it by calling
the Truestamp API. This is inherently an online operation; no offline mode
is offered.

Exit codes:
  0  valid (key is set and accepted by the API)
  1  no key, invalid key, or network error`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runAuthStatus,
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	if !stdinIsTerminal() {
		return fmt.Errorf("auth login requires an interactive terminal")
	}

	keysURL, err := apiKeysURL(appConfig.APIURL)
	if err != nil {
		return err
	}

	labelStyle := lipgloss.NewStyle().Foreground(ui.Label)
	valueStyle := lipgloss.NewStyle().Foreground(ui.Value).Bold(true)
	accent := lipgloss.NewStyle().Foreground(ui.Accent)

	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  "+accent.Render("Create and copy a new API key at:"))
	fmt.Fprintln(out, "    "+valueStyle.Render(keysURL))
	fmt.Fprintln(out, "  "+labelStyle.Render("    (existing keys cannot be copied — create a new one now)"))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  "+labelStyle.Render("Then paste the key below. Input is hidden."))
	fmt.Fprintln(out)

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

	fmt.Fprintln(out, ui.SuccessBanner("Logged in — API key saved to "+config.ConfigFilePath()))
	return nil
}

func runAuthLogout(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	if appConfig.APIKey == "" {
		fmt.Fprintln(out, ui.FaintStyle().Render("  No API key is currently stored."))
		return nil
	}

	if !stdinIsTerminal() {
		return fmt.Errorf("auth logout requires an interactive terminal")
	}

	var confirmed bool
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Remove the stored API key?").
				Description("This clears api_key in " + config.ConfigFilePath()).
				Affirmative("Yes, log out").
				Negative("Cancel").
				Value(&confirmed),
		),
	).WithTheme(ui.HuhTheme()).Run()
	if err != nil {
		return fmt.Errorf("confirmation: %w", err)
	}

	if !confirmed {
		fmt.Fprintln(out, ui.FaintStyle().Render("  Cancelled."))
		return nil
	}

	if err := config.SetAPIKey(""); err != nil {
		return err
	}

	fmt.Fprintln(out, ui.SuccessBanner("Logged out — API key removed"))
	return nil
}

func runAuthStatus(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	cfg := appConfig
	apiURL := cfg.APIURL
	checkURL := apiURL + "/users"

	labelStyle := lipgloss.NewStyle().Foreground(ui.Label)

	t := table.New().
		Row("Config File", config.ConfigFilePath()).
		Row("API URL", apiURL).
		Row("Check URL", checkURL).
		Row("API Key", maskAPIKey(cfg.APIKey)).
		Row("Team In Scope", teamInScope(cfg.Team)).
		Border(lipgloss.HiddenBorder()).
		StyleFunc(configStyleFunc)

	fmt.Fprintln(out, ui.HeaderBox("Truestamp Auth Status", "Validating with the API"))
	fmt.Fprintln(out)
	fmt.Fprintln(out, t.String())

	if cfg.APIKey == "" {
		fmt.Fprintln(out, ui.FailureBanner("Not logged in"))
		fmt.Fprintln(out, labelStyle.Render("    Run 'truestamp auth login' to store an API key."))
		return errSilentFail
	}

	ctx := cmd.Context()

	userResult, err := checkAPIKey(ctx, apiURL, cfg.APIKey, cfg.Team)
	if err != nil {
		fmt.Fprintln(out, ui.FailureBanner("Could not reach the API"))
		fmt.Fprintln(out, labelStyle.Render("    "+err.Error()))
		return errSilentFail
	}

	switch {
	case userResult.unauthorized:
		fmt.Fprintln(out, ui.FailureBanner("API key rejected by the server"))
		if userResult.message != "" {
			fmt.Fprintln(out, labelStyle.Render("    "+userResult.message))
		} else {
			fmt.Fprintf(out, "    %s\n", labelStyle.Render(fmt.Sprintf("HTTP %d — run 'truestamp auth login' to replace the key.", userResult.httpStatus)))
		}
		return errSilentFail

	case !userResult.ok:
		fmt.Fprintln(out, ui.FailureBanner(fmt.Sprintf("Unexpected API response (HTTP %d)", userResult.httpStatus)))
		if userResult.message != "" {
			fmt.Fprintln(out, labelStyle.Render("    "+userResult.message))
		}
		return errSilentFail
	}

	// User is authenticated. Resolve the team when one is configured, and
	// surface an auth-style failure if the team is not accessible.
	var teamResult *teamCheckResult
	if cfg.Team != "" {
		teamResult, err = fetchTeam(ctx, apiURL, cfg.APIKey, cfg.Team)
		if err != nil {
			fmt.Fprintln(out, ui.FailureBanner("Could not look up team"))
			fmt.Fprintln(out, labelStyle.Render("    "+err.Error()))
			return errSilentFail
		}
		if !teamResult.found {
			fmt.Fprintln(out, ui.FailureBanner("Team "+cfg.Team+" is not accessible"))
			if teamResult.message != "" {
				fmt.Fprintln(out, labelStyle.Render("    "+teamResult.message))
			}
			fmt.Fprintln(out, labelStyle.Render(fmt.Sprintf("    HTTP %d — the team id may be wrong, or this user is not a member.", teamResult.httpStatus)))
			return errSilentFail
		}
	}

	fmt.Fprintln(out, ui.SuccessBanner("Authenticated as "+formatUserIdentity(userResult)))
	fmt.Fprintln(out, labelStyle.Render("    Team: "+formatTeam(cfg.Team, teamResult)))
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

// formatTeam renders the resolved team context shown on success.
func formatTeam(teamID string, r *teamCheckResult) string {
	if teamID == "" {
		return "personal team (no tenant header sent)"
	}
	if r == nil || r.name == "" {
		return teamID
	}
	label := r.name
	if r.personal {
		label += " (personal)"
	}
	return fmt.Sprintf("%s  [%s]", label, teamID)
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

// teamCheckResult summarizes the outcome of the /teams/{id} probe. found
// is true only on a 2xx response; 401/403/404 all return found=false.
type teamCheckResult struct {
	found      bool
	httpStatus int
	name       string
	personal   bool
	message    string
}

// checkAPIKey sends GET {apiURL}/users with the given bearer token and
// interprets the response. Returns a non-nil error only for transport-level
// failures (DNS, timeout, connection refused). All other outcomes — 2xx,
// 4xx, 5xx — are reported in the result.
func checkAPIKey(ctx context.Context, apiURL, apiKey, team string) (*authCheckResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Keep the response small and sparse-fielded — an admin caller would
	// otherwise see every user on the system here.
	reqURL := apiURL + "/users?page[limit]=1&fields[user]=email,first_name,last_name,full_name"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
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
// can see that team. A 401/403/404 is treated as "not accessible" (the
// caller could not find it or lacks membership) rather than a transport
// error. 5xx surfaces in message with found=false.
func fetchTeam(ctx context.Context, apiURL, apiKey, teamID string) (*teamCheckResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reqURL := apiURL + "/teams/" + url.PathEscape(teamID) + "?fields[team]=name,personal"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
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
// array. Prefers the server-computed `full_name` attribute, then falls
// back to `first_name last_name`. Missing fields return empty strings.
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
// been configured. The API assigns requests without a tenant to the
// caller's personal team.
func teamInScope(team string) string {
	if team == "" {
		return "(personal team — set TRUESTAMP_TEAM or --team to override)"
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

// apiKeysURL derives the web app's API keys page from the API base URL by
// keeping the scheme and host and replacing the path with /api-keys. A
// default `https://www.truestamp.com/api/json` therefore maps to
// `https://www.truestamp.com/api-keys`, and a local `http://localhost:4000/api/json`
// maps to `http://localhost:4000/api-keys`.
func apiKeysURL(apiURL string) (string, error) {
	u, err := url.Parse(apiURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid api_url %q", apiURL)
	}
	return (&url.URL{Scheme: u.Scheme, Host: u.Host, Path: "/api-keys"}).String(), nil
}
