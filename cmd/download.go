// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var downloadCmd = &cobra.Command{
	Use:   "download <id>",
	Short: "Download a Truestamp proof bundle",
	Long: `Download a cryptographic proof bundle for an item or entropy observation.

The subject type is auto-detected from the ID format:
  ULID   (e.g. 01KNN33GX5E470CB9TRWAYF9DD)     -> item proof
  UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-...)     -> entropy proof

Examples:
  truestamp download 01KNN33GX5E470CB9TRWAYF9DD
  truestamp download --format cbor 01KNN33GX5E470CB9TRWAYF9DD
  truestamp download -o proof.json 01KNN33GX5E470CB9TRWAYF9DD
  truestamp download -f cbor -o proof.cbor 019d6a32-13e6-72b0-97e5-3779231ea97b

Requires --api-key to be set (via flag, env, or config file).`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			cmd.Help()
			return nil
		}

		cfg := appConfig
		id := args[0]

		// Validate API key
		if cfg.APIKey == "" {
			return fmt.Errorf("API key required (use --api-key or set TRUESTAMP_API_KEY)")
		}

		// Validate format
		format, _ := cmd.Flags().GetString("format")
		format = strings.ToLower(strings.TrimSpace(format))
		if format != "json" && format != "cbor" {
			return fmt.Errorf("--format must be \"json\" or \"cbor\", got %q", format)
		}

		// Detect ID type (also validates the ID format)
		idType, err := proof.DetectIDType(id)
		if err != nil {
			return err
		}

		// Download the proof
		data, err := proof.Generate(cfg.APIURL, cfg.APIKey, cfg.Team, id, format)
		if err != nil {
			return err
		}

		// Determine output filename
		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			ext := format
			if ext == "json" {
				ext = "json"
			} else {
				ext = "cbor"
			}
			output = fmt.Sprintf("truestamp-%s-%s.%s", idType, id, ext)
		}

		// Write file
		if err := os.WriteFile(output, data, 0644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		// Display success
		presentDownload(output, format, id, idType, len(data))

		return nil
	},
}

func presentDownload(filename, format, id string, idType proof.IDType, size int) {
	header := ui.AccentBoldStyle().Render("  Proof Downloaded")

	formatDisplay := strings.ToUpper(format)

	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(downloadStyleFunc).
		Row("File", filename).
		Row("Format", fmt.Sprintf("%s (%s bytes)", formatDisplay, formatSize(size))).
		Row("ID", id).
		Row("Type", string(idType))

	lipgloss.Println(lipgloss.JoinVertical(lipgloss.Left,
		header, "",
		tbl.String(),
	))
}

func downloadStyleFunc(row, col int) lipgloss.Style {
	if col == 0 {
		return lipgloss.NewStyle().
			Foreground(ui.Label).
			PaddingLeft(2).
			Align(lipgloss.Right).
			PaddingRight(1)
	}
	return lipgloss.NewStyle().Foreground(ui.Value)
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
	rootCmd.AddCommand(downloadCmd)
}
