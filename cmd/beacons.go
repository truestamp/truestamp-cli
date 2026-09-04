// Copyright (c) 2019-2026 Truestamp, Inc.
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

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/beacons"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// beaconsCmd is the parent for the `truestamp beacons ...` subtree. Like
// every group it has no RunE: a bare `truestamp beacons` prints help.
// It used to run `latest`, which made the group name mean two things and
// hid the word `latest` from the people who most needed to learn it.
var beaconsCmd = &cobra.Command{
	Use:   "beacons",
	Short: "Read-only: public randomness over finalized blocks",
	Long: `Query the Truestamp Beacons JSON:API surface.

A beacon is the four-field public view of a finalized block:
{id, hash, timestamp, previous_hash}. It commits to every item and
entropy observation finalized inside a minute window and makes a good
"proof of life" commitment.

Only finalized or committed blocks project as beacons, so
'beacons latest' and 'blocks latest' answer different questions: the head
block is routinely not yet finalized. For a block's internals — Merkle
root, state, signature, key id, chain links — use 'truestamp blocks get'.

Sub-commands:
  latest     Show the most recent finalized beacon
  list       Show the most recent N beacons (default 25, max 100)
  get        Show a beacon by UUIDv7 id or by 64-hex-char hash

Shared flags:
  --json         Print the raw JSON response, pretty-printed
  -s, --silent   No output, exit code only

On 'latest' and 'get' only:
  --hash-only    Print only the hash field + newline (for shell substitution)

Requires authentication, run 'truestamp auth login', or set TRUESTAMP_API_KEY / --api-key for headless/CI use.`,
	Args: cobra.NoArgs,
}

var beaconsLatestCmd = &cobra.Command{
	Use:           "latest",
	Short:         "Show the most recent finalized beacon",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runBeaconsLatest,
}

func runBeaconsLatest(cmd *cobra.Command, _ []string) error {
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
	// --json and --silent are resolved from the merged config so a
	// config-file or environment default applies here too; their mutual
	// exclusion is enforced once in config.Load.
	jsonOut, silent = outputMode(cmd)
	// --hash-only is not registered on `list`, where a single hash makes
	// no sense; GetBool returns false there, which is what we want.
	hashOnly, _ = cmd.Flags().GetBool("hash-only")

	if silent && hashOnly {
		return false, false, false, fmt.Errorf("--silent and --hash-only are mutually exclusive")
	}
	if jsonOut && hashOnly {
		return false, false, false, fmt.Errorf("--json and --hash-only are mutually exclusive")
	}
	return jsonOut, hashOnly, silent, nil
}

// beaconConfig pulls the values the beacons client needs from the resolved
// application config. Returns errSilentFail when no credential is
// configured (neither an OAuth session nor an API key), after first
// printing a "not authenticated" banner to stderr (unless silent).
func beaconConfig(cmd *cobra.Command) (beacons.Config, error) {
	cfg := appConfig
	if !authConfigured() {
		_, silent := outputMode(cmd)
		if !silent {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp auth login' to sign in (or set TRUESTAMP_API_KEY)."))
		}
		return beacons.Config{}, errSilentFail
	}
	return beacons.Config{
		APIURL: cfg.APIURL,
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
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    Run 'truestamp auth login' to sign in (or set TRUESTAMP_API_KEY)."))
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
		ui.Fprintln(cmd.OutOrStdout(), b.Hash)
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), b)
	}
	renderBeaconCard(cmd.OutOrStdout(), appConfig.APIURL, b)
	return nil
}

// renderBeaconCard prints the 4-field human-readable card followed by
// two public-web links (suppressed under --silent / --json upstream):
//
//	Details, the beacon detail page, keyed by hash (useful when a
//	          user has only the hash, e.g. printed on a receipt)
//	Verify , the verify page, keyed by the typed sub-path
//	          /verify/beacon/<id> (the shareable "go verify this"
//	          URL format introduced in the t=11 cutover)
func renderBeaconCard(w io.Writer, apiURL string, b *beacons.Beacon) {
	header := ui.AccentBoldStyle().Render("  Beacon")
	tbl := ui.CompactTable().
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("Hash", b.Hash).
		Row("Timestamp", timestampWithRelative(b.Timestamp)).
		Row("ID", b.ID).
		Row("Previous", b.PreviousHash)

	// Append URL rows to the SAME table so they inherit the
	// right-aligned-label / value-column alignment. Adding as separate
	// faint-styled lines would leave the labels dangling at a fixed
	// left indent and visually break the column grid.
	if detail := ui.BeaconDetailURL(apiURL, b.Hash); detail != "" {
		tbl = tbl.Row("Details", detail)
	}
	if verify := ui.BeaconVerifyURL(apiURL, b.ID); verify != "" {
		tbl = tbl.Row("Verify", verify)
	}

	// Plain newline-join, see note in internal/verify/presenter.go
	// Present(). lipgloss.JoinVertical pads every line to match the
	// widest line, which can blow up vertical spacing when a long line
	// forces terminal wrap on every row.
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

// timestampWithRelative appends a coarse "N minutes ago" hint to an ISO
// timestamp when it parses cleanly. Falls back to the input string.
// Delegates the second-precision truncation to the shared ui helper so
// all CLI surfaces render timestamps consistently.
func timestampWithRelative(ts string) string {
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, ts)
	}
	if err != nil {
		return ts
	}
	return ui.TruncateToSecond(ts) + "  (" + humanizeAge(time.Since(parsed)) + ")"
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

// URL helpers (publicWebBase, SubjectDetailURL, SubjectVerifyURL,
// BeaconDetailURL, BeaconVerifyURL, TeamDetailURL, TeamCreateURL) all
// live in internal/ui/weburls.go, they're shared across the beacon
// card, download card, create card, and team card, so centralization
// avoids drift.

// emitJSONMarshal is a small shim so subcommands can render either a
// single beacon or a list via emitJSON() (shared with codec subcommands).
func emitJSONMarshal(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = ui.Fprintln(w, string(data))
	return err
}

// Ensure helpers don't go unused when a later edit simplifies callers.
var _ = emitJSONMarshal
var _ = context.Background

func init() {
	f := beaconsLatestCmd.Flags()
	f.Bool("hash-only", false, "Print only the beacon hash + newline")
	addRecordOutputFlags(beaconsLatestCmd)

	beaconsCmd.AddCommand(beaconsLatestCmd)
	beaconsCmd.GroupID = groupResources
	rootCmd.AddCommand(asGroup(beaconsCmd))
}
