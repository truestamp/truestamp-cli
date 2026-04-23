// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Subject-type flag values for `truestamp download --type`. Four of these
// map 1:1 to the server's /proof/generate `type` enum; "beacon" is a
// CLIENT-SIDE alias — the server does not yet accept type="beacon" and
// rejects it with 400. We send wire type="block" and label the output
// file with stem "beacon" instead.
//
// TODO(server-beacon): once the server accepts type="beacon" natively,
// stop the alias and pass it through verbatim (see BEACONS_API_IMPLEMENTERS_GUIDE.md).
const (
	downloadTypeAuto    = "auto"
	downloadTypeItem    = "item"
	downloadTypeEntropy = "entropy"
	downloadTypeBlock   = "block"
	downloadTypeBeacon  = "beacon"
)

var downloadTypeValues = []string{
	downloadTypeAuto, downloadTypeItem, downloadTypeEntropy,
	downloadTypeBlock, downloadTypeBeacon,
}

var downloadCmd = &cobra.Command{
	Use:   "download <id>",
	Short: "Download a Truestamp proof bundle",
	Long: `Download a cryptographic proof bundle for a Truestamp subject.

The subject type is auto-detected by the server from the ID format when
--type is omitted (default). Override with --type to force a specific
class of proof.

  ULID   (e.g. 01KNN33GX5E470CB9TRWAYF9DD)         -> item proof
  UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b) -> block (tried first) then entropy

Use --type beacon to download a block proof saved with a 'beacon'
filename label — beacon and block proofs are byte-identical on the wire.

Output filename (when -o is not set):
  --type auto     -> truestamp-<item|entropy|block>-<id>.<ext>   (from returned t)
  --type item     -> truestamp-item-<ulid>.<ext>
  --type entropy  -> truestamp-entropy-<uuidv7>.<ext>
  --type block    -> truestamp-block-<uuidv7>.<ext>
  --type beacon   -> truestamp-beacon-<uuidv7>.<ext>

Examples:
  truestamp download 01KNN33GX5E470CB9TRWAYF9DD
  truestamp download --type block 019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp download --type beacon 019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp download --type beacon -f cbor 019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp download -o proof.json 01KNN33GX5E470CB9TRWAYF9DD

Requires --api-key to be set (via flag, env, or config file).`,
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
		if !validDownloadType(typeFlag) {
			return fmt.Errorf("--type must be one of %s, got %q",
				strings.Join(downloadTypeValues, " | "), typeFlag)
		}

		// Pre-flight id shape validation — sensible errors before hitting the network.
		if _, err := proof.DetectIDType(id); err != nil {
			return err
		}

		// Client-side beacon alias: wire sends type=block (the server does
		// not yet accept "beacon"), but the output file is labelled "beacon".
		wireType := typeFlag
		if wireType == downloadTypeBeacon {
			wireType = downloadTypeBlock
		}

		data, err := proof.GenerateCtx(cmd.Context(), cfg.APIURL, cfg.APIKey, cfg.Team, id, wireType, format)
		if err != nil {
			return err
		}

		stem, err := downloadStem(typeFlag, data)
		if err != nil {
			return err
		}

		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			output = fmt.Sprintf("truestamp-%s-%s.%s", stem, id, format)
		}

		if err := os.WriteFile(output, data, 0644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		presentDownload(output, format, id, stem, len(data))
		return nil
	},
}

// downloadStem picks the filename stem based on the user's --type choice
// and (for --type auto) the returned proof's `t` field.
func downloadStem(typeFlag string, data []byte) (string, error) {
	switch typeFlag {
	case downloadTypeItem, downloadTypeEntropy, downloadTypeBlock, downloadTypeBeacon:
		return typeFlag, nil
	case downloadTypeAuto, "":
		t, err := proof.InspectBundleType(data)
		if err != nil {
			return "", fmt.Errorf("inspecting returned proof: %w", err)
		}
		switch t {
		case ptype.Item:
			return downloadTypeItem, nil
		case ptype.EntropyNIST, ptype.EntropyStellar, ptype.EntropyBitcoin:
			return downloadTypeEntropy, nil
		case ptype.Block:
			return downloadTypeBlock, nil
		}
		return "", fmt.Errorf("unexpected subject type %d in returned proof", t)
	}
	return "", fmt.Errorf("unexpected --type value %q", typeFlag)
}

func validDownloadType(v string) bool {
	return slices.Contains(downloadTypeValues, v)
}

func presentDownload(filename, format, id, stem string, size int) {
	header := ui.AccentBoldStyle().Render("  Proof Downloaded")

	formatDisplay := strings.ToUpper(format)

	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("File", filename).
		Row("Format", fmt.Sprintf("%s (%s bytes)", formatDisplay, formatSize(size))).
		Row("ID", id).
		Row("Type", stem)

	lipgloss.Println(lipgloss.JoinVertical(lipgloss.Left,
		header, "",
		tbl.String(),
	))
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
	f.String("type", downloadTypeAuto,
		fmt.Sprintf(`Subject type: %s (beacon is a client-side alias for block)`,
			strings.Join(downloadTypeValues, " | ")))
	rootCmd.AddCommand(downloadCmd)
}
