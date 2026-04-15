// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print detailed version, build, and runtime information",
	Long:  "Print detailed version info including module path, config path, Go toolchain, platform, commit, and build date.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		configPath := configFile
		if configPath == "" {
			configPath = config.ConfigFilePath()
		}

		lines := []struct{ label, value string }{
			{"version", version.Version},
			{"path", version.Path},
			{"config path", configPath},
			{"go", version.GoFor()},
			{"commit", version.GitCommit},
			{"built", version.BuildDate},
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
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}