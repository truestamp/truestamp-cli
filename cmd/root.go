// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package cmd wires up the cobra command tree for the Truestamp CLI. The
// main entrypoint (cmd/truestamp/main.go) calls [Execute]; everything else
// here registers subcommands, flags, and the shared PersistentPreRunE that
// loads the resolved configuration into [appConfig].
package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/install"
	"github.com/truestamp/truestamp-cli/internal/logging"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/upgradecheck"
	"github.com/truestamp/truestamp-cli/internal/version"
)

// panicExitCode is the conventional Go runtime exit code for an
// unrecovered panic. We match it on purpose so existing CI / shell
// pipelines that branch on `[ $? -eq 2 ]` keep working.
const panicExitCode = 2

// errSilentFail signals a failure that should exit non-zero without any
// output. Commands use it for modes like `verify --silent` where the user
// has explicitly asked for no output.
var errSilentFail = errors.New("silent failure")

// appConfig holds the resolved configuration, available to all subcommands.
var appConfig *config.Config

// configFile is the --config flag value (overrides default path).
var configFile string

// appLogger is the process-wide JSON logger constructed in
// PersistentPreRunE and used by every subcommand. Always non-nil after
// the pre-run hook fires (falls back to logging.Discard() if file
// creation failed). Subcommands access it via the package var so
// Bubble Tea programs and goroutines can log without threading a
// context through every helper.
var appLogger *slog.Logger = logging.Discard()

// appLogPath is the resolved log file path, surfaced in diagnostic UIs
// (the Connection pane in the console, `truestamp config show`, etc.).
// Empty when no file logger could be constructed.
var appLogPath string

// appStartTime is captured in PersistentPreRunE so command_end can log
// the wall-clock duration of the invocation.
var appStartTime time.Time

var rootCmd = &cobra.Command{
	Use:     "truestamp",
	Short:   "Truestamp CLI — tools for cryptographic timestamping",
	Long:    "Truestamp CLI — tools for cryptographic timestamping\n\n" + version.Copyright(),
	Version: version.Short(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for completion/help (no config/HTTP/log
		// needed). Check os.Args directly: cobra's __complete command
		// has DisableFlagParsing=true, so the command hierarchy may
		// not be fully initialized when PersistentPreRunE fires.
		if len(os.Args) > 1 {
			switch os.Args[1] {
			case "completion", "__complete", "__completeNoDesc", "help":
				return nil
			}
		}

		noColor, _ := cmd.Flags().GetBool("no-color")
		ui.Init(noColor)

		cfg, err := config.Load(configFile, cmd.Flags())
		if err != nil {
			return err
		}
		appConfig = cfg
		httpclient.Init(cfg.Timeout())
		httpclient.SetUserAgent(version.Version)

		// Build the process-wide logger once config has been resolved
		// so user-supplied --log-file / TRUESTAMP_LOGGING_FILE / TOML
		// settings are honored. File creation failures are non-fatal —
		// we fall back to a discard logger and emit a one-line stderr
		// warning so a read-only home directory doesn't break commands.
		logger, logPath, lerr := logging.New(logging.Options{
			Path:       cfg.Logging.File,
			Level:      cfg.Logging.Level,
			MaxSizeMB:  cfg.Logging.MaxSizeMB,
			MaxBackups: cfg.Logging.MaxBackups,
			MaxAgeDays: cfg.Logging.MaxAgeDays,
			Component:  cmd.Name(),
		})
		if lerr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: log file disabled: %v\n", lerr)
		}
		appLogger = logger
		appLogPath = logPath
		appStartTime = time.Now()

		// Stash the logger on the command's context so Bubble Tea
		// programs and downstream callers that already accept a
		// context can pull it out without a package-level dependency.
		cmd.SetContext(loggerContext(cmd.Context(), logger))

		appLogger.Info("command_start",
			"cmd", cmd.CommandPath(),
			"argc", len(args),
			"version", version.Version,
			"install_method", install.Detect().String(),
		)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		maybeEmitUpgradeNotice(cmd)
	},
}

func init() {
	rootCmd.SetVersionTemplate(version.Full() + "\n")

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to config file (default: "+config.ConfigFilePath()+")")
	rootCmd.PersistentFlags().String("base-url", "", "Origin of the Truestamp service (scheme + host, e.g. https://www.truestamp.com)")
	rootCmd.PersistentFlags().String("api-key", "", "API key for authenticating with the Truestamp API")
	rootCmd.PersistentFlags().String("team", "", "Team ID for multi-tenant API operations")
	rootCmd.PersistentFlags().String("http-timeout", "", "HTTP timeout for external API calls (e.g. 10s, 30s, 1m)")
	rootCmd.PersistentFlags().String("log-file", "", "JSON log file path (default: "+logging.DefaultPath()+")")
	rootCmd.PersistentFlags().String("log-level", "", "Log level: debug | info | warn | error (default: info)")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable color output")
	rootCmd.PersistentFlags().Bool("no-upgrade-check", false, "Disable the once-per-day 'new version available' notice")
}

// loggerContextKey is the context.Value key for the process logger.
// Unexported so external packages can't fish it out by guessing the
// key — they must call LoggerFrom or read appLogger directly (only
// possible from the cmd package).
type loggerContextKey struct{}

// loggerContext returns ctx with logger attached at the loggerContextKey.
// Public-ish helper for symmetry with LoggerFrom; primarily used by
// PersistentPreRunE.
func loggerContext(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFrom returns the logger attached to ctx, or a discard logger if
// no logger was injected (e.g. tests that bypass PersistentPreRunE, or
// the completion / help fast paths).
func LoggerFrom(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return logging.Discard()
	}
	if v, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && v != nil {
		return v
	}
	return logging.Discard()
}

// maybeEmitUpgradeNotice runs after any successful subcommand and may
// write a one-line "upgrade available" notice to stderr. It is a no-op
// when the upgrade-check is disabled (flag, env var, CI, non-TTY), when
// the invoked subcommand is itself about upgrades/version/help, or when
// no newer release is found. All failures are swallowed; this path is
// never load-bearing.
func maybeEmitUpgradeNotice(cmd *cobra.Command) {
	// Only run for subcommands where a nag wouldn't be annoying. Also
	// skip for the root command itself (help output) and for hidden
	// completion helpers.
	name := cmd.Name()
	switch name {
	case "truestamp", "upgrade", "version", "help", "completion",
		"__complete", "__completeNoDesc":
		return
	}

	// Don't emit on non-TTY or CI — the Disabled() check handles those.
	flagDisabled, _ := cmd.Flags().GetBool("no-upgrade-check")
	upgradecheck.MaybeNotify(cmd.ErrOrStderr(), flagDisabled, version.Version)
}

// Execute runs the root command. Commands set SilenceErrors so cobra does
// not print their errors; Execute is the single place errors reach stderr.
// A command that needs silent-on-error UX (e.g. `verify --silent`) returns
// errSilentFail instead of the real error to opt out of printing. The
// upgrade --check flow uses exitCodeErr to return a specific exit code
// without also printing an error line.
//
// An unrecovered panic anywhere under rootCmd.Execute() is caught by the
// deferred recover below: the panic value and stack are written to the
// log file as a `command_end` record at error level (so a "the CLI
// crashed" support report has a forensic trail), the same stack is
// re-emitted on stderr in the runtime's default `panic: <value>\n\n
// <stack>` format so the operator sees what they would have seen
// without our wrapper, and the process exits with code 2 (matching the
// Go runtime's default panic exit code).
func Execute() (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		stack := debug.Stack()
		var dur time.Duration
		if !appStartTime.IsZero() {
			dur = time.Since(appStartTime)
		}
		// appLogger is initialized to logging.Discard() at package
		// scope so this is safe to call even when the panic fired
		// before PersistentPreRunE built the real logger.
		appLogger.Error("command_end",
			"duration_ms", dur.Milliseconds(),
			"exit", panicExitCode,
			"panic", fmt.Sprint(r),
			"stack", string(stack),
		)
		fmt.Fprintf(os.Stderr, "panic: %v\n\n%s", r, stack)
		os.Exit(panicExitCode)
	}()

	err = rootCmd.Execute()
	// Mirror the final outcome to the log so a single grep over
	// truestamp.log shows every invocation's exit shape — even when
	// the failure short-circuited PersistentPostRun.
	if !appStartTime.IsZero() {
		dur := time.Since(appStartTime)
		switch {
		case err == nil:
			appLogger.Info("command_end", "duration_ms", dur.Milliseconds(), "exit", 0)
		case errors.Is(err, errSilentFail):
			appLogger.Info("command_end", "duration_ms", dur.Milliseconds(), "exit", 1, "silent", true)
		default:
			var ec exitCodeErr
			code := 1
			if errors.As(err, &ec) {
				code = ec.code
			}
			appLogger.Error("command_end",
				"duration_ms", dur.Milliseconds(),
				"exit", code,
				"err", err.Error(),
			)
		}
	}
	if err == nil {
		return nil
	}
	var ec exitCodeErr
	if errors.As(err, &ec) {
		return err
	}
	if !errors.Is(err, errSilentFail) {
		fmt.Fprintln(os.Stderr, err)
	}
	return err
}
