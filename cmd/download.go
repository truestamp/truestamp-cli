// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Subject-type flag values for `truestamp download --type`. These map 1:1
// to the server's /proof/generate `type` string enum. There is no "auto"
// and no bare "entropy" — both were removed in the server's strict-type
// cutover. Callers that don't pass --type get a client-side smart default:
// a ULID id defaults to --type item; a UUIDv7 id errors asking for an
// explicit --type.
//
// Wire values preserve the underscore form (entropy_nist) to match the
// server enum exactly; filename stems translate underscores to hyphens
// for friendlier filenames (truestamp-entropy-nist-<id>.json).
const (
	downloadTypeItem           = "item"
	downloadTypeEntropyNIST    = "entropy_nist"
	downloadTypeEntropyStellar = "entropy_stellar"
	downloadTypeEntropyBitcoin = "entropy_bitcoin"
	downloadTypeBlock          = "block"
	downloadTypeBeacon         = "beacon"
)

// downloadTypeValues lists every accepted --type value in the order they
// appear in user-facing help text.
var downloadTypeValues = []string{
	downloadTypeItem,
	downloadTypeEntropyNIST,
	downloadTypeEntropyStellar,
	downloadTypeEntropyBitcoin,
	downloadTypeBlock,
	downloadTypeBeacon,
}

// downloadTypesForUUIDv7 lists the --type values that may be used with a
// UUIDv7 id (everything except "item").
var downloadTypesForUUIDv7 = []string{
	downloadTypeEntropyNIST,
	downloadTypeEntropyStellar,
	downloadTypeEntropyBitcoin,
	downloadTypeBlock,
	downloadTypeBeacon,
}

var downloadCmd = &cobra.Command{
	Use:   "download <id>",
	Short: "Download a Truestamp proof bundle",
	Long: `Download a cryptographic proof bundle for a Truestamp subject.

The server requires an explicit --type. For zero-flag convenience, if
--type is omitted the CLI falls back to the following rules based on
the ID format:

  ULID    (e.g. 01KNN33GX5E470CB9TRWAYF9DD)     -> --type item
  UUIDv7  (e.g. 019d6a32-13e6-72b0-97e5-...)    -> must specify --type

UUIDv7 ids are ambiguous (entropy observations, blocks, and beacons all
use UUIDv7), so an explicit --type is required for them.

Output filename (when -o is not set): truestamp-<stem>-<id>.<ext> where
<stem> is the --type value with underscores translated to hyphens:

  --type item              -> truestamp-item-<ulid>.<ext>
  --type entropy_nist      -> truestamp-entropy-nist-<uuidv7>.<ext>
  --type entropy_stellar   -> truestamp-entropy-stellar-<uuidv7>.<ext>
  --type entropy_bitcoin   -> truestamp-entropy-bitcoin-<uuidv7>.<ext>
  --type block             -> truestamp-block-<uuidv7>.<ext>
  --type beacon            -> truestamp-beacon-<uuidv7>.<ext>

Block (t=10) and beacon (t=11) proofs share the same structural shape
but carry distinct type codes on the wire. A block and beacon proof for
the same underlying block have different signatures — the ` + "`t`" + ` byte is
part of the signing payload, providing cryptographic domain separation.

Examples:
  truestamp download 01KNN33GX5E470CB9TRWAYF9DD
  truestamp download --type block   019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp download --type beacon  019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp download --type beacon -f cbor 019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp download --type entropy_stellar 019cf813-99b8-730a-84f1-5a711a9c355e
  truestamp download -o proof.json 01KNN33GX5E470CB9TRWAYF9DD

Requires --api-key to be set (via flag, env, or config file).

Exit code 0 on success, 1 on any error (validation failure, network
error, missing API key, server rejection, or failure to write the
output file).`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		cfg := appConfig
		id := args[0]

		if cfg.APIKey == "" {
			return fmt.Errorf("API key required (use --api-key or set TRUESTAMP_API_KEY)")
		}

		format, _ := cmd.Flags().GetString("format")
		format = strings.ToLower(strings.TrimSpace(format))
		if format != "json" && format != "cbor" {
			return fmt.Errorf("--format must be \"json\" or \"cbor\", got %q", format)
		}

		typeFlag, _ := cmd.Flags().GetString("type")
		typeFlag = strings.ToLower(strings.TrimSpace(typeFlag))

		// Pre-flight id-shape validation — catches obvious typos before
		// the network round-trip and is also the basis for the smart
		// --type default.
		shape, err := proof.DetectIDType(id)
		if err != nil {
			return err
		}

		// Resolve --type when not specified.
		if typeFlag == "" {
			switch shape {
			case proof.IDTypeULID:
				typeFlag = downloadTypeItem
			case proof.IDTypeUUIDv7:
				return fmt.Errorf(
					"--type is required for UUIDv7 ids (entropy, block, and beacon all use UUIDv7). One of: %s",
					strings.Join(downloadTypesForUUIDv7, " | "))
			default:
				// DetectIDType should have errored already; belt-and-suspenders.
				return fmt.Errorf("unrecognised id shape %q", shape)
			}
		}

		if !validDownloadType(typeFlag) {
			return fmt.Errorf("--type must be one of %s, got %q",
				strings.Join(downloadTypeValues, " | "), typeFlag)
		}

		// Shape vs type cross-check — surfaces obvious mismatches locally
		// before the server's 422 id_format_mismatch fires.
		if err := validateTypeVsShape(typeFlag, shape); err != nil {
			return err
		}

		// Wire-send typeFlag verbatim. Beacon is a first-class server type
		// now (returns t=11), not a client-side alias.
		data, err := proof.GenerateCtx(cmd.Context(), cfg.APIURL, cfg.APIKey, cfg.Team, id, typeFlag, format)
		if err != nil {
			return err
		}

		stem := downloadStem(typeFlag)

		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			output = fmt.Sprintf("truestamp-%s-%s.%s", stem, id, format)
		}

		if err := os.WriteFile(output, data, 0644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		presentDownload(output, format, id, typeFlag, len(data))
		return nil
	},
}

// downloadStem converts a resolved --type value to a filename stem. Wire
// values use underscores (entropy_nist) to match the server enum;
// filename stems use hyphens (entropy-nist) for friendlier filenames.
func downloadStem(typeFlag string) string {
	return strings.ReplaceAll(typeFlag, "_", "-")
}

// validDownloadType reports whether v is one of the six canonical --type
// values.
func validDownloadType(v string) bool {
	return slices.Contains(downloadTypeValues, v)
}

// validateTypeVsShape ensures --type matches the syntactic shape of the
// positional id. `item` requires a ULID; every other type requires a
// UUIDv7. Runs locally before the network call so users get a targeted
// error rather than a generic server 422.
func validateTypeVsShape(typeFlag string, shape proof.IDType) error {
	switch typeFlag {
	case downloadTypeItem:
		if shape != proof.IDTypeULID {
			return fmt.Errorf("--type %s requires a ULID id (e.g. 01KNN33GX5E470CB9TRWAYF9DD); got a UUIDv7", typeFlag)
		}
	default:
		// entropy_*, block, beacon — all UUIDv7.
		if shape != proof.IDTypeUUIDv7 {
			return fmt.Errorf("--type %s requires a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b); got a ULID", typeFlag)
		}
	}
	return nil
}

func presentDownload(filename, format, id, typeFlag string, size int) {
	header := ui.AccentBoldStyle().Render("  Proof Downloaded")

	formatDisplay := strings.ToUpper(format)

	tbl := ui.CompactTable().
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("File", filename).
		Row("Format", fmt.Sprintf("%s (%s bytes)", formatDisplay, formatSize(size))).
		Row("ID", id).
		Row("Type", typeFlag)

	// Append URL rows to the SAME table so they share the
	// right-aligned-label / value column alignment. subjectDetailURL
	// routes beacon downloads to /blocks/<id> because the id we have
	// is the block id; the hash-keyed /beacons/<hash> form lives on
	// the beacon listing card where the hash comes directly from the
	// API response.
	if detail := ui.SubjectDetailURL(appConfig.APIURL, typeFlag, id); detail != "" {
		tbl = tbl.Row("Details", detail)
	}
	if verify := ui.SubjectVerifyURL(appConfig.APIURL, typeFlag, id); verify != "" {
		tbl = tbl.Row("Verify", verify)
	}

	// Plain newline-join — see note in internal/verify/presenter.go
	// Present(). Long filenames (e.g. truestamp-entropy-bitcoin-<uuidv7>.cbor)
	// won't inflate every other table row on narrow terminals.
	lipgloss.Println(strings.Join([]string{header, "", tbl.String()}, "\n"))
}

func formatSize(size int) string {
	switch {
	case size >= 1_000_000:
		return fmt.Sprintf("%d,%03d,%03d", size/1_000_000, (size/1_000)%1_000, size%1_000)
	case size >= 1_000:
		return fmt.Sprintf("%d,%03d", size/1_000, size%1_000)
	default:
		return fmt.Sprintf("%d", size)
	}
}

func init() {
	f := downloadCmd.Flags()
	f.StringP("format", "f", "json", `Output format: "json" or "cbor"`)
	f.StringP("output", "o", "", "Output file path (default: auto-generated from ID)")
	f.String("type", "",
		fmt.Sprintf(`Subject type (required for UUIDv7 ids; auto-defaults to "item" for ULID). One of: %s`,
			strings.Join(downloadTypeValues, " | ")))
	rootCmd.AddCommand(downloadCmd)
}
