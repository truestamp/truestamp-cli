// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
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
// that says it does.

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
//
// Map members are written sorted by name (json.Deterministic), so one value
// prints one way. jsontext.PreserveRawStrings keeps a string a MarshalJSON
// method wrote exactly as written: without it the encoder rewrites each
// string in its shortest form, which would undo the \uXXXX escapes
// claimsName.MarshalJSON writes for the invisible and directional code
// points of SPEC.md section 3.7 (inspect_test.go checks both).
func emitJSON(w io.Writer, v any) error {
	out, err := json.Marshal(v,
		jsontext.WithIndent("  "),
		json.Deterministic(true),
		jsontext.PreserveRawStrings(true),
	)
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	recordWritten = true
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
// shape the resource groups share: a 401 is the shared banner (no
// credential, or the one sent was rejected), a 403 an "Access denied"
// banner naming the team when the server refused the tenant and the record
// otherwise, a 404 one line naming the record, anything else one line
// carrying the server's detail, and nothing at all under --silent. Under
// --json each is a *cliError, which Execute writes as the one failure
// object. noun names the record ("beacon", "block", "item", "team").
func renderAPIError(cmd *cobra.Command, err error, noun string) error {
	if errors.Is(err, jsonapi.ErrUnauthorized) {
		if !authConfigured() {
			return failNotAuthenticated(cmd)
		}
		return failCredentialRejected(cmd, err)
	}
	var apiErr *jsonapi.APIError
	isAPI := errors.As(err, &apiErr)
	if errors.Is(err, jsonapi.ErrForbidden) {
		if isAPI && isTenantRefusal(apiErr) {
			return failTeamRefused(cmd, err)
		}
		msg := "You're authenticated, but you do not have access to that " + noun + "."
		return failWith(cmd, &cliError{code: codeForbidden, message: msg, status: 403, cause: err},
			"Access denied", msg)
	}
	if _, silent := outputMode(cmd); silent {
		return errSilentFail
	}
	if !isAPI {
		return err
	}
	e := &cliError{code: apiFailureCode(apiErr), message: apiErr.Error(), status: apiErr.Status, cause: err}
	switch {
	case errors.Is(err, jsonapi.ErrNotFound):
		e.message = noun + " not found"
	case apiErr.Code == jsonapi.CodeInvalidKeyset:
		// A cursor is opaque and only meaningful to the listing that
		// printed it; say that rather than echoing "invalid keyset".
		e.message = "the cursor was not recognised: use one printed by a previous page of this listing (More: --after, Back: --before)"
	case errors.Is(err, jsonapi.ErrRateLimited):
		e.message = describeRateLimit(apiErr)
	}
	return e
}

// describeRateLimit is the one rendering of a 429 that is still standing
// after the one retry jsonapi.Send makes: what the server said, how long
// it asked the holder to wait (or that no wait will admit the request),
// and the limit it named. The limit's window is server configuration and
// is not carried, so the number is shown as the server sent it, never as
// "per minute".
func describeRateLimit(e *jsonapi.APIError) string {
	var notes []string
	wait, named := e.RetryAfterDelay()
	switch {
	case e.NeverAdmitted:
		notes = append(notes, "this request is over the limit on its own and no wait will admit it")
	case named:
		notes = append(notes, "retry after "+formatWait(wait))
	}
	if e.Limit > 0 {
		notes = append(notes, fmt.Sprintf("limit %d", e.Limit))
	}
	msg := "rate limited"
	if len(notes) > 0 {
		msg += " (" + strings.Join(notes, ", ") + ")"
	}
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	return msg
}

// formatWait renders a wait in whole seconds, rounded up and never below
// one, the way the server's own detail phrases it.
func formatWait(d time.Duration) string {
	secs := int64(math.Ceil(d.Seconds()))
	if secs < 1 {
		secs = 1
	}
	return fmt.Sprintf("%ds", secs)
}
