// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/redact"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var teamCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new team",
	Long: `Create a new team owned by you. You become the team owner, and the
team is created under your account (not within your current active team).

The name may be given as the positional argument or with --name. With
neither on an interactive terminal, a short form prompts for the name and
ownership model. The --ownership-model flag chooses how ownership is held:

  creator_retains  Items you create stay yours — you keep them if you
                   leave the team (the default).
  team_retains     Items you create belong to the team — they stay with
                   the team if you leave or are removed. Requires a plan
                   entitlement.

Team creation is plan-limited. If you've reached your plan's cap (or the
chosen ownership model isn't in your plan) the server explains the limit
and the CLI surfaces it with a clear, actionable message.

Examples:
  truestamp team create "Acme Engineering"
  truestamp team create --name "Acme" --ownership-model team_retains --set
  truestamp team create                       # interactive prompt`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runTeamCreate,
}

// createTeamCtx is a seam letting the test suite stub the network without
// exporting the underlying client. Defaults to the real client at runtime.
var createTeamCtx = func(ctx context.Context, cfg teams.Config, name, ownership string) (*teams.Team, error) {
	return teams.CreateTeam(ctx, cfg, name, ownership)
}

func runTeamCreate(cmd *cobra.Command, args []string) error {
	silent, _ := cmd.Flags().GetBool("silent")
	jsonOut, _ := cmd.Flags().GetBool("json")
	setActive, _ := cmd.Flags().GetBool("set")
	nameFlag, _ := cmd.Flags().GetString("name")
	ownershipFlag, _ := cmd.Flags().GetString("ownership-model")

	cfg, err := teamConfig(cmd)
	if err != nil {
		return err
	}

	name := strings.TrimSpace(nameFlag)
	if name == "" && len(args) > 0 {
		name = strings.TrimSpace(args[0])
	}
	ownership, err := normalizeOwnership(ownershipFlag)
	if err != nil {
		return err
	}

	// Prompt for the name (and ownership model) when missing on a TTY. In
	// --json/--silent or non-interactive use, a missing name is a hard error
	// so scripted callers get a clear failure instead of a hang. The server
	// is the authority on plan entitlement — there is no pre-flight check.
	if name == "" {
		if jsonOut || silent || !stdinIsTerminal() {
			return fmt.Errorf("team name is required (pass a name argument or --name)")
		}
		picked, pickedOwnership, perr := promptTeamCreate(name, ownership)
		if perr != nil {
			return perr
		}
		if picked == "" {
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render("  Cancelled."))
			return nil
		}
		name, ownership = picked, pickedOwnership
	}

	team, err := createTeamCtx(cmd.Context(), cfg, name, ownership)
	if err != nil {
		if jsonOut {
			return printTeamCreateErrorJSON(cmd, err)
		}
		return renderCreateError(cmd, cfg.APIURL, err, ownership, silent)
	}

	if setActive {
		if serr := config.SetTeam(team.ID); serr != nil {
			return fmt.Errorf("team created (%s) but writing active team to config failed: %w", team.ID, serr)
		}
	}

	if jsonOut {
		return printTeamCreateJSON(cmd, team, setActive)
	}
	if silent {
		return nil
	}
	renderTeamCreateCard(cmd, cfg.APIURL, team, setActive)
	return nil
}

// normalizeOwnership maps user-friendly --ownership-model input to the
// canonical wire value, accepting short aliases. An empty value stays empty
// so the server applies its own default.
func normalizeOwnership(in string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case "":
		return "", nil
	case teams.OwnershipCreatorRetains, "creator", "creator-retains":
		return teams.OwnershipCreatorRetains, nil
	case teams.OwnershipTeamRetains, "team", "team-retains":
		return teams.OwnershipTeamRetains, nil
	}
	return "", fmt.Errorf("invalid --ownership-model %q (want %q or %q)",
		in, teams.OwnershipCreatorRetains, teams.OwnershipTeamRetains)
}

// promptTeamCreate runs a small huh form for the name and ownership model.
// Both ownership models are offered; the server is the authority on whether
// the chosen one is entitled. Returns an empty name when the user cancels.
func promptTeamCreate(name, ownership string) (string, string, error) {
	if ownership == "" {
		ownership = teams.OwnershipCreatorRetains
	}
	models := teams.OwnershipModels()
	opts := make([]huh.Option[string], 0, len(models))
	for _, m := range models {
		opts = append(opts, huh.NewOption(teams.OwnershipLabel(m)+" — "+teams.OwnershipDescription(m), m))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Team name").
				Description("A label for the new team.").
				CharLimit(200).
				Validate(requiredTrimmed("team name")).
				Value(&name),
			huh.NewSelect[string]().
				Title("Ownership model").
				Options(opts...).
				Value(&ownership),
		),
	).WithTheme(ui.HuhTheme())

	if err := form.Run(); err != nil {
		// Treat any Run() error as cancel — the user can re-run.
		return "", "", nil
	}
	return strings.TrimSpace(name), ownership, nil
}

func requiredTrimmed(field string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
}

// renderCreateError maps the two create-specific policy rejections to
// tailored messages (preferring the server's self-describing detail) and
// otherwise defers to the shared team error renderer.
func renderCreateError(cmd *cobra.Command, apiURL string, err error, ownership string, silent bool) error {
	if silent {
		return errSilentFail
	}
	detail := apiErrorDetail(err)
	switch {
	case errors.Is(err, teams.ErrTeamLimitReached):
		fmt.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Team limit reached"))
		fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
			"    "+orDefault(detail, "Your plan does not allow creating additional teams.")))
		if url := ui.TeamCreateURL(apiURL); url != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render("    Manage teams & plans: "+url))
		}
		return errSilentFail
	case errors.Is(err, teams.ErrOwnershipNotEntitled):
		fmt.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Ownership model not available"))
		fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render("    "+orDefault(detail, fmt.Sprintf(
			"The %q ownership model isn't included in your plan.", teams.OwnershipLabel(ownership)))))
		fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
			"    Use --ownership-model creator_retains, or upgrade your plan."))
		return errSilentFail
	}
	return teamRenderError(cmd, err, silent)
}

// apiErrorDetail returns the (redacted) server detail of a *teams.APIError,
// or "" for any other error. Redaction at the presentation boundary is
// defense in depth against a reflected credential in an error body.
func apiErrorDetail(err error) string {
	var ae *teams.APIError
	if errors.As(err, &ae) {
		return redact.String(ae.Detail)
	}
	return ""
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func renderTeamCreateCard(cmd *cobra.Command, apiURL string, team *teams.Team, setActive bool) {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, ui.SuccessBanner("Team created"))
	// A freshly created team makes you its owner.
	renderTeamCard(w, apiURL, team, "team_owner", setActive)
	fmt.Fprintln(w)
	if setActive {
		fmt.Fprintln(w, ui.FaintStyle().Render(
			"  Set as your active team in "+config.ConfigFilePath()+"."))
	} else {
		fmt.Fprintln(w, ui.FaintStyle().Render(
			"  Run 'truestamp team set "+team.ID+"' to make it your active team."))
	}
}

func printTeamCreateJSON(cmd *cobra.Command, team *teams.Team, setActive bool) error {
	out := map[string]any{
		"id":              team.ID,
		"name":            team.Name,
		"personal":        team.Personal,
		"ownership_model": team.OwnershipModel,
		"created_at":      team.CreatedAt,
		"active":          setActive,
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// printTeamCreateErrorJSON emits a structured, parseable error object on
// stdout under --json (so a script's `| jq` doesn't choke on empty input)
// and returns errSilentFail for the non-zero exit.
func printTeamCreateErrorJSON(cmd *cobra.Command, err error) error {
	code := "create_failed"
	switch {
	case errors.Is(err, teams.ErrTeamLimitReached):
		code = "team_limit_reached"
	case errors.Is(err, teams.ErrOwnershipNotEntitled):
		code = "ownership_not_entitled"
	case errors.Is(err, teams.ErrUnauthorized):
		code = "unauthenticated"
	case errors.Is(err, teams.ErrForbidden):
		code = "forbidden"
	}
	out := map[string]any{
		"error": redact.String(err.Error()),
		"code":  code,
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
	return errSilentFail
}

func init() {
	f := teamCreateCmd.Flags()
	f.StringP("name", "n", "", "Team name (also accepted as the positional argument)")
	f.String("ownership-model", "", "Ownership model: creator_retains (default) or team_retains")
	f.Bool("set", false, "Set the new team as the active team after creating it")
	f.Bool("json", false, "Output the created team as JSON")
	f.BoolP("silent", "s", false, "No output, exit code only")

	teamCmd.AddCommand(teamCreateCmd)
}
