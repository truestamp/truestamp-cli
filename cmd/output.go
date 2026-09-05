// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// The R10 output contract, in one place so a new command cannot invent a
// variation of it. See kb/command-tree.md R10 for the rules and for the
// list of deliberate exemptions (the pipeline primitives that print one
// bare value or raw bytes, and `hash --style`, which is a byte-identical
// sha256sum drop-in).
//
// --json and --silent are NOT root persistent flags. A persistent flag
// appears on every command, including `auth login`, `console` and
// `completion`, which render no record and would then advertise an output
// mode they do not have. They are registered per command instead, from
// here, so the set of commands that carry the pair is exactly the set
// that says it does. `proofs convert` is the one command that registers
// its own --json, because there the flag means "a JSON envelope" and
// needs its own help text.

// addRecordOutputFlags registers --json and --silent on a command.
// config.Load enforces their mutual exclusion once, for every command, so
// callers do not repeat that check. Whether a command honors the ambient
// config-file / environment setting for the pair is decided where it
// reads them: record commands go through outputMode, the pipeline
// primitives read the raw flags (R10's exemptions).
func addRecordOutputFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.Bool("json", false, "Output as JSON")
	f.BoolP("silent", "s", false, "No output, exit code only")
}

// emitJSON writes v as indented JSON. It is the single JSON writer for
// every --json rendering, so the shape of that output is decided in one
// place rather than by whichever encoder each command reached for.
func emitJSON(w io.Writer, v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(out))
	return err
}

// outputMode reads the resolved output settings for the current
// invocation.
//
// It prefers appConfig, which config.Load has already merged from
// defaults, config file, environment and flags in precedence order — that
// is what makes `silent = true` in config.toml work at all. An explicitly
// changed flag is consulted directly as well, because it wins under that
// same precedence, and because unit tests that call a RunE without going
// through PersistentPreRunE have no resolved config to read.
func outputMode(cmd *cobra.Command) (jsonOut, silent bool) {
	if appConfig != nil {
		jsonOut, silent = appConfig.JSON, appConfig.Silent
	}
	if f := cmd.Flags().Lookup("json"); f != nil && f.Changed {
		jsonOut, _ = cmd.Flags().GetBool("json")
	}
	if f := cmd.Flags().Lookup("silent"); f != nil && f.Changed {
		silent, _ = cmd.Flags().GetBool("silent")
	}
	return jsonOut, silent
}

// printNotAuthenticated is the one rendering of the "Not authenticated"
// banner and its remediation hint, so the wording, the stream and the
// silent gating cannot drift between the resource groups. It prints
// nothing under --silent.
func printNotAuthenticated(cmd *cobra.Command) {
	if _, silent := outputMode(cmd); silent {
		return
	}
	ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Not authenticated"))
	ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
		"    Run 'truestamp auth login' to sign in (or set TRUESTAMP_API_KEY)."))
}

// failNotAuthenticated prints the banner and returns errSilentFail. It is
// the right response both when no credential is configured and when the
// server rejects the one that is (a 401 on a request that did carry one).
func failNotAuthenticated(cmd *cobra.Command) error {
	printNotAuthenticated(cmd)
	return errSilentFail
}

// requireAuth is the pre-flight gate every command that needs a credential
// runs before its first request: it refuses early, with the banner, when
// neither an OAuth session nor an API key is configured.
func requireAuth(cmd *cobra.Command) error {
	if authConfigured() {
		return nil
	}
	return failNotAuthenticated(cmd)
}

// renderAPIError turns a jsonapi client error into the message and exit
// shape the resource groups share: a 401 is the shared banner, a 403 an
// "Access denied" banner, a 404 one line naming the record, anything else
// one line carrying the server's detail, and nothing at all under
// --silent. noun names the record ("beacon", "block", "item", "team").
func renderAPIError(cmd *cobra.Command, err error, noun string) error {
	if errors.Is(err, jsonapi.ErrUnauthorized) {
		return failNotAuthenticated(cmd)
	}
	_, silent := outputMode(cmd)
	if errors.Is(err, jsonapi.ErrForbidden) {
		if !silent {
			ui.Fprintln(cmd.ErrOrStderr(), ui.FailureBanner("Access denied"))
			ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
				"    You're authenticated, but you do not have access to that "+noun+"."))
		}
		return errSilentFail
	}
	if silent {
		return errSilentFail
	}
	var apiErr *jsonapi.APIError
	if errors.As(err, &apiErr) {
		switch {
		case errors.Is(err, jsonapi.ErrNotFound):
			return fmt.Errorf("%s not found", noun)
		case apiErr.Code == jsonapi.CodeInvalidKeyset:
			// A cursor is opaque and only meaningful to the listing that
			// printed it; say that rather than echoing "invalid keyset".
			return errors.New("the cursor was not recognised: use one printed by a previous page of this listing (More: --after, Back: --before)")
		case errors.Is(err, jsonapi.ErrRateLimited) && apiErr.RetryAfter != "":
			return fmt.Errorf("rate limited (Retry-After: %s): %s", apiErr.RetryAfter, apiErr.Detail)
		}
		return errors.New(apiErr.Error())
	}
	return err
}
