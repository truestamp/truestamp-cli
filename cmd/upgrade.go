// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/install"
	"github.com/truestamp/truestamp-cli/internal/selfupgrade"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/version"
)

// Exit codes for --check. The user-facing contract lives in cmd so
// selfupgrade can stay free of CLI concerns.
const (
	checkExitUpToDate     = 0
	checkExitUpgradeAvail = 1
	checkExitNetworkErr   = 2
	checkExitPreRelease   = 3
)

var (
	upgradeFlagCheck    bool
	upgradeFlagYes      bool
	upgradeFlagVersion  string
	upgradeFlagNoVerify bool
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the truestamp CLI to the latest release",
	Long: `Upgrade truestamp to the latest release.

The behavior depends on how the binary was installed:

  homebrew   → prints the correct 'brew upgrade' command to run
  go install → prints the 'go install ...@latest' command to run
  install.sh → downloads the latest release, verifies its SHA-256 and
               cosign signature, and atomically replaces the binary in
               place. Requires write access to the binary's directory.

Run 'truestamp version' to see the detected install method.

Examples:
  truestamp upgrade                       # upgrade to the latest release
  truestamp upgrade --check               # only print whether an upgrade is available
  truestamp upgrade --yes                 # upgrade without interactive confirmation
  truestamp upgrade --version v0.4.0      # pin to a specific release tag`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runUpgrade,
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeFlagCheck, "check", false, "Only check for a newer version; do not install (exit 0=up-to-date, 1=upgrade available, 2=network error, 3=pre-release)")
	upgradeCmd.Flags().BoolVarP(&upgradeFlagYes, "yes", "y", false, "Skip the interactive confirmation prompt")
	upgradeCmd.Flags().StringVar(&upgradeFlagVersion, "version", "", "Pin to a specific release tag (e.g. v0.4.0). Bypasses the pre-release filter.")
	upgradeCmd.Flags().BoolVar(&upgradeFlagNoVerify, "no-verify", false, "Skip cosign signature verification (SHA-256 is still enforced)")
	_ = upgradeCmd.Flags().MarkHidden("no-verify")

	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	method := install.Detect()
	current := version.Version

	// --check always queries the API so scripts can learn about
	// available upgrades regardless of install method. For the default
	// upgrade path, package-manager users short-circuit here.
	if !upgradeFlagCheck {
		if msg, ok := upgradeInstructionFor(method); ok {
			printUpgradeInstruction(cmd.OutOrStdout(), method, msg)
			return nil
		}
	}

	opts := selfupgrade.Options{
		TargetVersion:  upgradeFlagVersion,
		CurrentVersion: current,
		RequireCosign:  os.Getenv("TRUESTAMP_REQUIRE_COSIGN") == "1",
		SkipCosign:     upgradeFlagNoVerify,
		CosignPath:     appConfig.CosignPath,
		Logger: func(msg string) {
			fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", msg)
		},
	}

	if upgradeFlagCheck {
		return runCheck(ctx, cmd.OutOrStdout(), opts)
	}

	return runInPlaceUpgrade(ctx, cmd, opts)
}

// upgradeInstructionFor returns the message for package-manager installs,
// or ("", false) to signal that the caller should perform the in-place
// upgrade itself. Windows always gets go-install (install.sh doesn't
// support Windows, Homebrew doesn't exist there).
func upgradeInstructionFor(method install.Method) (string, bool) {
	if runtime.GOOS == "windows" {
		return "go install " + selfupgrade.ReleasesRepo + "/cmd/truestamp@latest", true
	}
	switch method {
	case install.Homebrew:
		return method.UpgradeCommand(), true
	case install.GoInstall:
		return method.UpgradeCommand(), true
	case install.Unknown:
		// Writable unknown paths fall through to the in-place flow.
		if install.BinaryWritable() {
			return "", false
		}
		return "curl -fsSL https://get.truestamp.com/install.sh | sh", true
	}
	// InstallScript falls through to in-place upgrade.
	return "", false
}

func printUpgradeInstruction(out io.Writer, method install.Method, command string) {
	labelStyle := lipgloss.NewStyle().Foreground(ui.Label)
	valueStyle := lipgloss.NewStyle().Foreground(ui.Value).Bold(true)
	accent := lipgloss.NewStyle().Foreground(ui.Accent)

	fmt.Fprintln(out)
	fmt.Fprintf(out, "  %s  %s\n", labelStyle.Render("detected install"), accent.Render(method.String()))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  To upgrade, run:")
	fmt.Fprintf(out, "    %s\n", valueStyle.Render(command))
	fmt.Fprintln(out)
}

func runCheck(ctx context.Context, out io.Writer, opts selfupgrade.Options) error {
	result, err := selfupgrade.Check(ctx, opts)
	if errors.Is(err, selfupgrade.ErrPreRelease) {
		latest := selfupgrade.Display(result.LatestVersion)
		fmt.Fprintf(out, "note: latest release %s is a pre-release; pass --version %s to install it explicitly, or wait for the next stable release.\n", latest, result.LatestVersion)
		return exitWith(checkExitPreRelease)
	}
	if err != nil {
		fmt.Fprintf(out, "upgrade check failed: %v\n", err)
		return exitWith(checkExitNetworkErr)
	}
	if result.UpgradeAvail {
		fmt.Fprintf(out, "truestamp %s is available (current: %s)\n", selfupgrade.Display(result.LatestVersion), selfupgrade.Display(result.CurrentVersion))
		return exitWith(checkExitUpgradeAvail)
	}
	fmt.Fprintf(out, "truestamp is up to date (%s)\n", selfupgrade.Display(result.CurrentVersion))
	return nil
}

func runInPlaceUpgrade(ctx context.Context, cmd *cobra.Command, opts selfupgrade.Options) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	result, err := selfupgrade.Check(ctx, opts)
	if errors.Is(err, selfupgrade.ErrPreRelease) {
		fmt.Fprintf(errOut, "note: latest release %s is a pre-release; pass --version %s to install it explicitly, or wait for the next stable release.\n", selfupgrade.Display(result.LatestVersion), result.LatestVersion)
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve latest: %w", err)
	}
	if !result.UpgradeAvail && opts.TargetVersion == "" {
		fmt.Fprintf(out, "truestamp is up to date (%s)\n", selfupgrade.Display(result.CurrentVersion))
		return nil
	}

	target := result.LatestVersion
	if opts.TargetVersion != "" {
		target = opts.TargetVersion
	}

	if !upgradeFlagYes && stdinIsTerminal() {
		fmt.Fprintf(out, "Upgrade truestamp %s → %s? [Y/n] ", selfupgrade.Display(result.CurrentVersion), selfupgrade.Display(target))
		if !readYes(cmd.InOrStdin()) {
			fmt.Fprintln(out, "aborted")
			return nil
		}
	}

	installed, backup, err := selfupgrade.Upgrade(ctx, opts)
	if errors.Is(err, selfupgrade.ErrAlreadyCurrent) {
		fmt.Fprintf(out, "truestamp is already at %s\n", selfupgrade.Display(installed))
		return nil
	}
	if err != nil {
		return err
	}

	successStyle := lipgloss.NewStyle().Foreground(ui.Green).Bold(true)
	fmt.Fprintf(out, "%s upgraded %s → %s\n", successStyle.Render("✓"), selfupgrade.Display(result.CurrentVersion), selfupgrade.Display(installed))
	if backup != "" {
		dim := lipgloss.NewStyle().Foreground(ui.Dim)
		fmt.Fprintf(out, "%s\n", dim.Render("  previous binary: "+backup))
	}
	return nil
}

// readYes reads one line from r. Empty, "y", or "yes" (case-insensitive)
// are yes; anything else is no. EOF defaults to yes to match the "[Y/n]"
// prompt convention used by most installers.
func readYes(r io.Reader) bool {
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		return true
	}
	resp := strings.ToLower(strings.TrimSpace(sc.Text()))
	return resp == "" || resp == "y" || resp == "yes"
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// exitCodeErr carries a specific exit code through to Execute() without
// triggering cobra's default error printing. Execute() extracts the code
// and calls os.Exit — but currently the CLI just returns non-zero via
// os.Exit(1), so we set silent via SilenceErrors on the command itself.
type exitCodeErr struct{ code int }

func (e exitCodeErr) Error() string { return fmt.Sprintf("exit %d", e.code) }

// ExitCode returns the code to pass to os.Exit for a given error.
// Falls back to 1 for generic errors, 0 for nil, or the stored code
// when err wraps an exitCodeErr.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ec exitCodeErr
	if errors.As(err, &ec) {
		return ec.code
	}
	return 1
}

func exitWith(code int) error {
	if code == 0 {
		return nil
	}
	return exitCodeErr{code: code}
}
