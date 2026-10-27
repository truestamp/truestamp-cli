// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier"
	"github.com/truestamp/truestamp-cli/internal/verify"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [file-or-url]",
	Short: "Verify a Truestamp proof",
	Long: `Verify a Truestamp proof, one JSON document, as SPEC.md of
truestamp/truestamp-verification defines it.

The run makes the steps of SPEC.md section 10.2 in order: it fixes the
environment it checks, the network pins, the sources and the keyring
before the file is read; reads and parses the file; checks the envelope,
the environment marker, the signer's key against the keyring and the
signature; then the schema, every network, and every hash, inclusion
proof and link, recomputed from the values the proof carries. Online, it
reads each Stellar and Bitcoin commitment and each entropy source, and
draws the submitted-after and submitted-before edges of the submission
window from what it read. --offline reads nothing: each check that needs
a read is a skip, never a failure.

Proof input can be provided as:
  truestamp verify proof.json           Local file path
  truestamp verify https://host/p.json  URL (auto-detected)
  truestamp verify --file proof.json    Explicit file path
  truestamp verify --file               Interactive file picker
  truestamp verify --url https://...    Explicit URL download
  truestamp verify --url                Interactive URL prompt
  cat proof.json | truestamp verify     Pipe from stdin

The keyring that governs the signer's key, for this run only:
  (none)                 online: the keyring read from www.truestamp.com;
                         offline: the keys built into this CLI
  --keyring FILE         a keyring file you supply
  --no-keyring           keyless: the signature is checked only against
                         the key the proof carries
  --environment NAME     check proofs of Truestamp's dev, staging or test
                         environment, with --keyring FILE, a keyring
                         configured for it, or --no-keyring

--expected-hash compares the hash of a file you hold with the hash the
proof's claims carry. --trace FILE writes the run's trace (SPEC.md section
10.6) to FILE. --horizon, --esplora and --bitcoin-core read Stellar and
Bitcoin from servers you name; a Bitcoin Core node's credentials come from
the configuration file or the environment, never the command line.

The report is labelled verified, internally-consistent, supplied-keyring,
unattributed or failed; --json writes it exactly as SPEC.md section 10.5
gives it. --remote also asks the Truestamp service, whose verdict is
printed beside the report and never changes the exit code; it waits for
the service's answer for the larger of --http-timeout and 75s.

Exit codes (SPEC.md section 12.2): 0 verified; 1 a runtime error, such as
an invalid command line or an input that cannot be read; 2 the verifier
stopped on a fault; 3 failed; 4 a bare rejection other than
configuration_error; 5 unattributed; 6 supplied-keyring; 7
internally-consistent; 8 configuration_error. Only 0 says that Truestamp
signed the proof and that its submission window is established.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runVerify,
}

func init() {
	f := verifyCmd.Flags()
	f.String("file", "", "Path to proof file (interactive picker if no path given)")
	f.String("url", "", "URL to download proof from (interactive prompt if no URL given)")
	f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
	f.Lookup("url").NoOptDefVal = inputsrc.URLPromptSentinel
	f.String("expected-hash", "", "Hash (hex) of the file you hold, compared with the hash the proof's claims carry")
	f.String(verifier.FlagKeyring, "", "Check the signer's key against this keyring file, for this run only")
	verifier.RegisterFlags(f)
	f.String("trace", "", "Write the run's trace (SPEC.md section 10.6) to this file")
	f.BoolP("silent", "s", false, "No output, exit code only")
	f.Bool("json", false, "Output the report as JSON, exactly as SPEC.md section 10.5 gives it")
	f.Bool("offline", false, "Read nothing outside the proof: each check that needs a read is reported as skipped")
	f.Bool("remote", false, "Also ask the Truestamp service to verify the proof (requires authentication)")
	verifyCmd.GroupID = groupVerification
	rootCmd.AddCommand(verifyCmd)
}

// runVerify is `truestamp verify`: it builds the run from the command's
// options, runs it through internal/verifier, writes what the run gave and
// exits with the code of how it ended (SPEC.md section 12.2).
func runVerify(cmd *cobra.Command, args []string) error {
	cfg := appConfig
	positional := ""
	if len(args) > 0 {
		positional = args[0]
	}
	fileFlag, _ := cmd.Flags().GetString("file")
	urlFlag, _ := cmd.Flags().GetString("url")
	runtimeError := func(err error) error {
		if cfg.Silent {
			return errSilentFail
		}
		return err
	}
	if positional == "" && fileFlag == "" && urlFlag == "" && !inputsrc.IsStdinPipe() {
		return runtimeError(fmt.Errorf("no proof was given: name a file or a URL, or pipe the proof on standard input\n\n%s", strings.TrimRight(cmd.UsageString(), "\n")))
	}
	options, err := verifier.OptionsFromCommand(cmd.Flags(), cfg)
	if err != nil {
		return runtimeError(err)
	}
	run, err := options.Config(verifier.DefaultKeyringReader(), verifier.DefaultIdentityReader())
	if err != nil {
		return runtimeError(err)
	}
	if flag := cmd.Flags().Lookup("expected-hash"); flag.Changed {
		value := flag.Value.String()
		run.ExpectedHash = &value
	}
	trace, _ := cmd.Flags().GetString("trace")
	if trace != "" {
		if info, statErr := os.Stat(filepath.Dir(trace)); statErr != nil || !info.IsDir() {
			return runtimeError(fmt.Errorf("--trace names a file in a directory that does not exist"))
		}
	}
	if cfg.Verify.Remote && !authConfigured() {
		return runtimeError(errors.New("not authenticated: --remote verification needs `truestamp auth login`, or TRUESTAMP_API_KEY / --api-key for headless use"))
	}

	var data []byte
	var src inputsrc.Source
	outcome := verifier.Verify(run, func() ([]byte, error) {
		stream, resolved, err := inputsrc.ResolveStream(cmd.Context(), inputsrc.Options{
			PositionalArg:        positional,
			FileFlag:             fileFlag,
			URLFlag:              urlFlag,
			AllowStdin:           true,
			AutoDetectURL:        true,
			PickerTitle:          "Select proof file",
			PickerExts:           []string{".json"},
			URLPromptTitle:       "Enter proof URL",
			URLPromptPlaceholder: "https://example.com/proof.json",
			ReadAtMost:           strictjson.MaxReadOctets + 1,
		})
		if err != nil {
			return nil, err
		}
		defer func() { _ = stream.Close() }()
		src = resolved
		data, err = strictjson.Read(stream)
		return data, err
	})
	appLogger.Info("verify_completed",
		"source", string(src.Type),
		"ending", string(outcome.Ending),
		"exit", outcome.ExitCode(),
	)
	if err := writeVerification(cmd, outcome, trace); err != nil {
		return err
	}
	if cfg.Verify.Remote && data != nil {
		askRemote(cmd, data, run.ExpectedHash, !run.Online)
	}
	return exitWith(outcome.ExitCode())
}

// writeVerification writes what the run gave: the trace to the file trace
// names, when it names one and the run gave a report or a bare rejection;
// then, unless --silent, the report or the bare rejection on standard
// output, as JSON under --json and as text otherwise, or why there is
// none on standard error. A trace that cannot be written is a runtime
// error (exit code 1) and nothing else is written; a report or rejection
// that cannot be encoded is a fault while the report is written (exit code
// 2); one that is encoded but cannot be written to standard output is a
// runtime error (exit code 1), so that a script never reads a code for a
// report it was not given.
func writeVerification(cmd *cobra.Command, outcome verifier.Outcome, trace string) error {
	cfg := appConfig
	stdout, stderr := cmd.OutOrStdout(), cmd.ErrOrStderr()
	switch outcome.Ending {
	case specdata.EndingRuntimeError:
		if cfg.Silent {
			return errSilentFail
		}
		return outcome.Err
	case specdata.EndingFault:
		if !cfg.Silent {
			_, _ = fmt.Fprintf(stderr, "The verifier stopped on a fault before it could give a report: %v. No report.\n", outcome.Err)
		}
		return exitWith(outcome.ExitCode())
	}
	if trace != "" {
		rows := make([]any, 0, len(outcome.Trace))
		for _, row := range outcome.Trace {
			rows = append(rows, row.Value())
		}
		text, err := jcs.CanonicalForm(rows)
		if err == nil {
			err = os.WriteFile(trace, text, 0o644)
		}
		if err != nil {
			if cfg.Silent {
				return errSilentFail
			}
			return fmt.Errorf("the trace cannot be written to %s: %w; no report", trace, err)
		}
	}
	encoded, err := encodeOutcome(outcome)
	if err != nil {
		if !cfg.Silent {
			_, _ = fmt.Fprintf(stderr, "The verifier stopped on a fault while it wrote the report: %v. No report.\n", err)
		}
		code, _ := specdata.ExitCode(specdata.EndingFault)
		return exitWith(code)
	}
	var written error
	switch {
	case cfg.Silent:
	case cfg.JSON:
		_, written = stdout.Write(append(encoded, '\n'))
	case outcome.Report != nil:
		written = renderReport(stdout, outcome.Report)
	default:
		written = renderRejection(stdout, outcome.Rejection)
	}
	if written != nil {
		return fmt.Errorf("the report cannot be written to standard output: %w", written)
	}
	return nil
}

// encodeOutcome is the report or the bare rejection written with JCS, as
// --json writes it, made whatever the output so that a report that cannot
// be written is found before anything is.
func encodeOutcome(outcome verifier.Outcome) ([]byte, error) {
	if outcome.Report != nil {
		return outcome.Report.Encode()
	}
	return outcome.Rejection.Encode()
}

// askRemote asks the Truestamp service to verify the proof too, posting
// the bytes read, and prints its verdict, the label of its report, on
// standard error, beside the report and never as a row of it or a reason
// code. It never changes the exit code, which is the local run's: a
// verdict that disagrees with the report and a request that gives no
// verdict are printed and nothing more. Under --json standard output stays
// exactly the report. The log line of a request that gives no verdict
// carries the service's reason code when it refused the request; the
// printed line never does.
func askRemote(cmd *cobra.Command, data []byte, expectedHash *string, offline bool) {
	cfg := appConfig
	label, err := verify.AskRemoteCtx(cmd.Context(), data, remoteOptions(cfg, expectedHash, offline))
	if err != nil {
		attrs := []any{"err", err.Error()}
		var rejection *verify.RemoteRejectionError
		if errors.As(err, &rejection) {
			attrs = append(attrs, "code", rejection.Code)
		}
		appLogger.Info("verify_remote_no_verdict", attrs...)
	} else {
		appLogger.Info("verify_remote_verdict", "label", label)
	}
	if cfg.Silent {
		return
	}
	var w io.Writer = cmd.ErrOrStderr()
	if err != nil {
		_, _ = fmt.Fprintf(w, "The Truestamp service gave no verdict, which changes neither the report nor the exit code: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(w, "The Truestamp service's verdict, beside this report and not part of it: %s.\n", label)
}

// remoteOptions is what askRemote asks the service with: the configured
// API, team and http_timeout (the request waits for the larger of it and
// the service's floor), and the run's offline flag and expected hash.
func remoteOptions(cfg *config.Config, expectedHash *string, offline bool) verify.RemoteOptions {
	return verify.RemoteOptions{APIURL: cfg.APIURL, Team: cfg.Team, Offline: offline, ExpectedHash: expectedHash, HTTPTimeout: cfg.Timeout()}
}
