// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/install"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print detailed version, build, and runtime information",
	Long:  "Print detailed version info including module path, config path, install method, Go toolchain, platform, commit, and build date.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// The file in effect, not the platform default: `truestamp
		// version --config X` must report X.
		configPath := config.ActivePath()

		jsonOut, silent := outputMode(cmd)
		if silent {
			return nil
		}
		// One record feeds both renderings, so the text and --json forms
		// cannot disagree about a value.
		rec := versionRecord{
			Version:    version.Version,
			Path:       version.Path,
			ConfigPath: configPath,
			Install:    install.Detect().String(),
			Go:         version.GoFor(),
			Commit:     version.GitCommit,
			Built:      version.BuildDate,
		}
		if jsonOut {
			return emitJSON(cmd.OutOrStdout(), rec)
		}

		lines := []struct{ label, value string }{
			{"version", rec.Version},
			{"path", rec.Path},
			{"config path", rec.ConfigPath},
			{"install", rec.Install},
			{"go", rec.Go},
			{"commit", rec.Commit},
			{"built", rec.Built},
		}

		labelStyle := lipgloss.NewStyle().Foreground(ui.Label).Width(11)
		valueStyle := lipgloss.NewStyle().Foreground(ui.Value)

		lipgloss.Println("truestamp")
		for _, ln := range lines {
			lipgloss.Println(fmt.Sprintf("  %s  %s",
				labelStyle.Render(ln.label),
				valueStyle.Render(ln.value),
			))
		}
		return nil
	},
}

// versionRecord is the --json shape of `truestamp version`. Field names
// match the text labels so the two renderings are diffable by eye.
type versionRecord struct {
	Version    string `json:"version"`
	Path       string `json:"path"`
	ConfigPath string `json:"config_path"`
	Install    string `json:"install"`
	Go         string `json:"go"`
	Commit     string `json:"commit"`
	Built      string `json:"built"`
}

func init() {
	addRecordOutputFlags(versionCmd)
	versionCmd.GroupID = groupOther
	rootCmd.AddCommand(versionCmd)
}
