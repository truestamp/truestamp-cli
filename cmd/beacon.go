// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/beacons"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// beaconCmd is the parent for the `truestamp beacon ...` subtree. Invoking
// it without a subcommand behaves like `beacon latest`.
var beaconCmd = &cobra.Command{
	Use:   "beacon",
	Short: "Inspect Truestamp block beacons",
	Long: `Query the Truestamp Beacons JSON:API surface.

A Beacon is a compact projection of a finalized Truestamp block:
{id, hash, timestamp, previous_hash}. It commits to every item and
entropy observation finalized inside a minute window and makes a great
"proof of life" anchor.

Sub-commands:
  latest     Show the current head beacon
  list       Show the most recent N beacons (default 25, max 100)
  get        Show a beacon by UUIDv7 id
  by-hash    Show a beacon by 64-hex-char hash

Invoking 'truestamp beacon' with no subcommand is an alias for 'beacon latest'.

Shared flags:
  --json         Print the raw JSON response, pretty-printed
  --hash-only    Print only the hash field + newline (for shell substitution)
  -s, --silent   No output, exit code only

Requires --api-key to be set (via flag, env, or config file).`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runBeaconLatest, // default = latest
}

var beaconLatestCmd = &cobra.Command{
	Use:           "latest",
	Short:         "Show the current head beacon",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runBeaconLatest,
}

func runBeaconLatest(cmd *cobra.Command, _ []string) error {
	jsonOut, hashOnly, silent, err := beaconSharedFlags(cmd)
	if err != nil {
		return err
	}

	cfg, err := beaconConfig(cmd)
	if err != nil {
		return err
	}

	b, err := beacons.Latest(cmd.Context(), cfg)
	if err != nil {
		return beaconRenderError(cmd, err, silent)
	}
	return renderBeacon(cmd, b, jsonOut, hashOnly, silent)
}

// ---------------- shared plumbing ----------------

// beaconSharedFlags reads --json / --hash-only / --silent and enforces the
// mutual-exclusion rules that apply across all beacon subcommands.
func beaconSharedFlags(cmd *cobra.Command) (jsonOut, hashOnly, silent bool, err error) {
	jsonOut, _ = cmd.Flags().GetBool("json")
	hashOnly, _ = cmd.Flags().GetBool("hash-only")
	silent, _ = cmd.Flags().GetBool("silent")

	if silent && jsonOut {
		return false, false, false, fmt.Errorf("--silent and --json are mutually exclusive")
	}
	if silent && hashOnly {
		return false, false, false, fmt.Errorf("--silent and --hash-only are mutually exclusive")
	}
	if jsonOut && hashOnly {
		return false, false, false, fmt.Errorf("--json and --hash-only are mutually exclusive")
	}
	return jsonOut, hashOnly, silent, nil
}

// beaconConfig pulls the values the beacons client needs from the resolved
// application config. Returns errSilentFail when no API key is set, after
// first printing a "not authenticated" banner to stderr (unless silent).
func beaconConfig(cmd *cobra.Command) (beacons.Config, error) {
	cfg := appConfig
	if cfg.APIKey == "" {
		silent, _ := cmd.Flags().GetBool("silent")
		if !silent {
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp auth login' to store an API key."))
		}
		return beacons.Config{}, errSilentFail
	}
	return beacons.Config{
		APIURL: cfg.APIURL,
		APIKey: cfg.APIKey,
		Team:   cfg.Team,
	}, nil
}

// beaconRenderError converts a client error into a user-facing message and
// an appropriate non-zero exit. Preserves `errors[].detail` verbatim.
// 401 → errSilentFail after printing the Not-authenticated banner; other
// statuses → a plain error so the root Execute() prints it to stderr.
func beaconRenderError(cmd *cobra.Command, err error, silent bool) error {
	if errors.Is(err, beacons.ErrUnauthorized) {
		if !silent {
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			fmt.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp auth login' to store an API key."))
		}
		return errSilentFail
	}
	if silent {
		return errSilentFail
	}
	// Surface the API detail via the stringer.
	var apiErr *beacons.APIError
	if errors.As(err, &apiErr) {
		switch {
		case errors.Is(err, beacons.ErrNotFound):
			return fmt.Errorf("beacon not found")
		case errors.Is(err, beacons.ErrRateLimited) && apiErr.RetryAfter != "":
			return fmt.Errorf("rate limited (Retry-After: %s): %s", apiErr.RetryAfter, apiErr.Detail)
		default:
			return fmt.Errorf("%s", apiErr.Error())
		}
	}
	return err
}

// renderBeacon emits the single-beacon card, --json, or --hash-only.
func renderBeacon(cmd *cobra.Command, b *beacons.Beacon, jsonOut, hashOnly, silent bool) error {
	if silent {
		return nil
	}
	if hashOnly {
		fmt.Fprintln(cmd.OutOrStdout(), b.Hash)
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), b)
	}
	renderBeaconCard(cmd.OutOrStdout(), appConfig.APIURL, b)
	return nil
}

// renderBeaconCard prints the 4-field human-readable card. The "Verify"
// link is suppressed under --silent and --json (both handled upstream).
func renderBeaconCard(w io.Writer, apiURL string, b *beacons.Beacon) {
	header := ui.AccentBoldStyle().Render("  Beacon")
	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("Hash", b.Hash).
		Row("Timestamp", timestampWithRelative(b.Timestamp)).
		Row("ID", b.ID).
		Row("Previous", b.PreviousHash)

	fmt.Fprintln(w, lipgloss.JoinVertical(lipgloss.Left, header, "", tbl.String()))
	if link := beaconWebURL(apiURL, b.Hash); link != "" {
		fmt.Fprintln(w, ui.FaintStyle().Render("    Verify → "+link))
	}
}

// timestampWithRelative appends a coarse "N minutes ago" hint to an ISO
// timestamp when it parses cleanly. Falls back to the original string.
func timestampWithRelative(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
	}
	if err != nil {
		return ts
	}
	// Render the timestamp at second precision for readability, then a
	// rough relative label.
	display := t.UTC().Format("2006-01-02T15:04:05Z")
	return display + "  (" + humanizeAge(time.Since(t)) + ")"
}

func humanizeAge(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	secs := int(math.Round(d.Seconds()))
	switch {
	case secs < 60:
		return fmt.Sprintf("%ds ago", secs)
	case secs < 3600:
		return fmt.Sprintf("%dm ago", secs/60)
	case secs < 86400:
		return fmt.Sprintf("%dh ago", secs/3600)
	default:
		return fmt.Sprintf("%dd ago", secs/86400)
	}
}

// beaconWebURL derives the public web URL for a beacon hash from the
// configured API URL. Returns "" when the API URL looks non-standard
// (e.g. a dev host), in which case the card omits the link. We don't
// emit localhost or internal-looking URLs as "Verify" links.
func beaconWebURL(apiURL, hash string) string {
	// Trim the /api/json suffix if present and keep https://<host>.
	base := strings.TrimSuffix(apiURL, "/")
	base = strings.TrimSuffix(base, "/api/json")
	if !strings.HasPrefix(base, "https://") {
		return ""
	}
	if strings.Contains(base, "localhost") || strings.Contains(base, "127.0.0.1") {
		return ""
	}
	return base + "/beacons/" + hash
}

// emitJSONMarshal is a small shim so subcommands can render either a
// single beacon or a list via emitJSON() (shared with codec subcommands).
func emitJSONMarshal(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// Ensure helpers don't go unused when a later edit simplifies callers.
var _ = emitJSONMarshal
var _ = context.Background

func init() {
	// Shared flags on the parent (inherited through cobra's local flag
	// lookup in each RunE) and on each subcommand for --help clarity.
	for _, c := range []*cobra.Command{beaconCmd, beaconLatestCmd} {
		f := c.Flags()
		f.Bool("json", false, "Print the raw JSON response, pretty-printed")
		f.Bool("hash-only", false, "Print only the beacon hash + newline (for shell substitution)")
		f.BoolP("silent", "s", false, "No output, exit code only")
	}

	beaconCmd.AddCommand(beaconLatestCmd)
	rootCmd.AddCommand(beaconCmd)
}
