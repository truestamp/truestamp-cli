// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/items"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
	"github.com/truestamp/truestamp-cli/internal/redact"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Failures under --json, in one place so a new command cannot invent a
// fifth shape. See kb/command-tree.md R10b.
//
// A command run with --json that fails prints exactly one object on
// standard output: {"code", "message"}, plus "http_status" when an HTTP
// refusal caused the failure, plus the members a few failures carry
// (items create's claims_reasons and other_errors, its retryable, the
// sub_reason of a malformed_json input). code is one of errorCodes, the
// closed registry `truestamp schema get error-codes` publishes, and
// message is the sentence the text rendering prints.
//
// The object is built by failureRecord and written by Execute, nowhere
// else. A command or renderer that fails under --json returns a *cliError
// and prints nothing; Execute writes any other error it is handed through
// the same classification. Under --silent nothing is printed. Two commands
// keep their own record: auth status, whose record with ok and reason is
// its answer, and verify, whose report and bare rejection SPEC.md sections
// 10.5 and 12.1 fix (ownJSONFailure). A command that has already written
// its record (hash over several inputs, some of which failed) is given no
// second object.

// The codes of errorCodes, as constants for the places that name one.
const (
	codeNotAuthenticated     = "not_authenticated"
	codeCredentialRejected   = "credential_rejected" // #nosec G101 -- the name of an error code, not a credential
	codeTeamNotAccessible    = "team_not_accessible"
	codeForbidden            = "forbidden"
	codeNotFound             = "not_found"
	codeNoTeamConfigured     = "no_team_configured"
	codeRateLimited          = "rate_limited"
	codeInvalidCursor        = "invalid_cursor"
	codeRequestRefused       = "request_refused"
	codeServerError          = "server_error"
	codeNetworkError         = "network_error"
	codeClaimsRefused        = "claims_refused"
	codeItemCodeUnavailable  = items.CodeItemCodeUnavailable
	codeTeamLimitReached     = "team_limit_reached"
	codeOwnershipNotEntitled = "ownership_not_entitled"
	codeMalformedJSON        = "malformed_json"
	codeInputTooLarge        = "input_too_large"
	codeInvalidUsage         = "invalid_usage"
	codeCommandFailed        = "command_failed"
)

// errorCode is one entry of the failure registry.
type errorCode struct {
	Code    string `json:"code"`
	Meaning string `json:"meaning"`
}

// errorCodes is the closed registry of the code a --json failure carries.
// TestFailure_CodesAreTheRegistry fails when a code is used that is not
// here, and schema get error-codes publishes this slice as it stands.
var errorCodes = []errorCode{
	{codeNotAuthenticated, "no credential is configured: no OAuth session and no API key"},
	{codeCredentialRejected, "the server refused the credential the request carried (HTTP 401)"},
	{codeTeamNotAccessible, "the server refused the team the request named as its tenant (HTTP 403): the caller is not a member of it, or the id is wrong"},
	{codeForbidden, "the caller is authenticated but may not read or change that record (HTTP 403)"},
	{codeNotFound, "the record does not exist or the caller cannot see it (HTTP 404), or no entry of a published list has that id"},
	{codeNoTeamConfigured, "the command reads the team in scope and none is configured"},
	{codeRateLimited, "the server refused the request for its rate, and again after the one retry (HTTP 429)"},
	{codeInvalidCursor, "a --after or --before cursor that no page of this listing printed"},
	{codeRequestRefused, "the server refused the request for another reason; http_status and message give its answer (HTTP 4xx)"},
	{codeServerError, "the server failed to answer the request (HTTP 5xx)"},
	{codeNetworkError, "the server could not be reached, or gave no HTTP answer"},
	{codeClaimsRefused, "items create: the server refused the claims; claims_reasons lists each rule broken, and nothing was created"},
	{codeItemCodeUnavailable, "items create: no item code could be issued and nothing was created; the same request may be sent again (retryable)"},
	{codeTeamLimitReached, "teams create: the plan allows no more teams"},
	{codeOwnershipNotEntitled, "teams create: the plan does not include the ownership model asked for"},
	{codeMalformedJSON, "the input is not JSON as a Truestamp verifier reads it (SPEC.md section 10.1); sub_reason names the rule it breaks"},
	{codeInputTooLarge, "the input is over the size a Truestamp verifier reads (SPEC.md section 10.1)"},
	{codeInvalidUsage, "the command line or the configuration was refused before the command ran"},
	{codeCommandFailed, "any other failure; message says what failed"},
}

// cliError is a failure that carries its registry code. Error is the
// message, the text the non-JSON rendering prints, so returning one in
// place of a plain error changes nothing outside --json.
type cliError struct {
	code    string
	message string
	status  int            // the HTTP status of the refusal behind it, 0 when none
	extra   map[string]any // members a few failures add to the object
	cause   error
}

func (e *cliError) Error() string { return e.message }
func (e *cliError) Unwrap() error { return e.cause }

// ownJSONFailure are the commands whose --json failures are their own
// record, which Execute never replaces: verify's bare rejection and report
// are SPEC.md's, and an object of this shape beside them could be read as a
// rejection with a code SPEC.md does not define.
var ownJSONFailure = map[string]bool{"truestamp verify": true}

// recordWritten is set by emitJSON, so Execute can tell a command that has
// already written its record from one that wrote nothing.
var recordWritten bool

// jsonFailure reports whether a failure of c is written as the one JSON
// object: c carries --json and it is on, by its own flag or, except on
// rawOutputCommands, by the ambient setting. --json and --silent given
// together are a usage error that names neither mode, and is printed as
// text.
func jsonFailure(c *cobra.Command) bool {
	if c == nil || ownJSONFailure[c.CommandPath()] {
		return false
	}
	f := c.Flags().Lookup("json")
	if f == nil {
		return false
	}
	if s := c.Flags().Lookup("silent"); s != nil && s.Changed {
		if v, _ := c.Flags().GetBool("silent"); v {
			return false
		}
	}
	if f.Changed {
		v, _ := c.Flags().GetBool("json")
		return v
	}
	return !rawOutputCommands[c.CommandPath()] && appConfig != nil && appConfig.JSON
}

// writeFailure is Execute's rendering of a failure it is to print: the one
// JSON object under --json, the error's text on stderr otherwise.
func writeFailure(c *cobra.Command, err error) {
	if !jsonFailure(c) {
		ui.Fprintln(os.Stderr, err)
		return
	}
	if recordWritten {
		return
	}
	_ = emitJSON(c.OutOrStdout(), failureRecord(err))
}

// failureRecord is the one builder of the --json failure object. The
// message is redacted, since stdout is the stream most often kept in logs
// and CI artifacts, and a transport or OAuth error can echo a token or an
// api_key query.
func failureRecord(err error) map[string]any {
	e := classifyFailure(err)
	rec := map[string]any{"code": e.code, "message": redact.String(e.message)}
	if e.status != 0 {
		rec["http_status"] = e.status
	}
	for k, v := range e.extra {
		rec[k] = v
	}
	return rec
}

// classifyFailure gives any error its registry code. A *cliError has one
// already; an HTTP refusal is classed by its status; a refusal of the
// strict JSON reader keeps its SPEC.md code; and everything else is
// command_failed, but an error raised before the configuration loaded,
// which is the command line's or the configuration's.
func classifyFailure(err error) *cliError {
	var ce *cliError
	if errors.As(err, &ce) {
		return ce
	}
	if appConfig == nil {
		return &cliError{code: codeInvalidUsage, message: err.Error(), cause: err}
	}
	var apiErr *jsonapi.APIError
	if errors.As(err, &apiErr) {
		return &cliError{code: apiFailureCode(apiErr), message: err.Error(), status: apiErr.Status, cause: err}
	}
	var rej *strictjson.Rejection
	if errors.As(err, &rej) {
		e := &cliError{code: codeMalformedJSON, message: err.Error(), cause: err}
		if string(rej.Code) == codeInputTooLarge {
			e.code = codeInputTooLarge
		}
		if rej.SubReason != "" {
			e.extra = map[string]any{"sub_reason": string(rej.SubReason)}
		}
		return e
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) || httpclient.IsTransport(err) {
		return &cliError{code: codeNetworkError, message: err.Error(), cause: err}
	}
	return &cliError{code: codeCommandFailed, message: err.Error(), cause: err}
}

// apiFailureCode classes an HTTP refusal by its status, and a 403 by
// whether it refused the team or the record.
func apiFailureCode(e *jsonapi.APIError) string {
	switch {
	case e.Status == 401 && !authConfigured():
		return codeNotAuthenticated
	case e.Status == 401:
		return codeCredentialRejected
	case e.Status == 403 && isTenantRefusal(e):
		return codeTeamNotAccessible
	case e.Status == 403:
		return codeForbidden
	case e.Status == 404:
		return codeNotFound
	case e.Status == 429:
		return codeRateLimited
	case e.Code == jsonapi.CodeInvalidKeyset:
		return codeInvalidCursor
	case e.Status >= 500:
		return codeServerError
	}
	return codeRequestRefused
}

// failWith renders a failure the one way every command does: nothing under
// --silent; under --json (as jsonFailure reads it, so only on a command
// that carries the flag), e itself, which Execute writes as the object;
// otherwise the banner and its lines on stderr. Every failure banner in
// the tree but auth status's goes through here, which is what keeps a
// banner off a --json run (TestFailure_BannersGoThroughFailWith).
func failWith(cmd *cobra.Command, e *cliError, banner string, lines ...string) error {
	if _, silent := outputMode(cmd); silent {
		return errSilentFail
	}
	if jsonFailure(cmd) {
		return e
	}
	w := cmd.ErrOrStderr()
	ui.Fprintln(w, ui.FailureBanner(banner))
	for _, l := range lines {
		ui.Fprintln(w, ui.FaintStyle().Render("    "+l))
	}
	return errSilentFail
}

// notAuthenticatedHint is what to do when no credential is configured, the
// same sentence auth status gives as its message.
const notAuthenticatedHint = "Run 'truestamp auth login' to sign in (or set TRUESTAMP_API_KEY)."

// failNotAuthenticated is the one rendering of "no credential is
// configured". A credential the server rejected is failCredentialRejected.
func failNotAuthenticated(cmd *cobra.Command) error {
	return failWith(cmd, &cliError{code: codeNotAuthenticated, message: notAuthenticatedHint},
		"Not authenticated", notAuthenticatedHint)
}

// credentialRejectedHint names the credential the server refused, by the
// mode the run resolved, and what to do about it.
func credentialRejectedHint() string {
	if auth.Default().Mode() == auth.ModeOAuth {
		return "The server rejected the OAuth session. Run 'truestamp auth login' to sign in again."
	}
	return "The server rejected the API key (from --api-key, TRUESTAMP_API_KEY or api_key in the config file). " +
		"Check the key, or run 'truestamp auth login' to sign in instead."
}

// failCredentialRejected is the one rendering of a 401 on a request that
// carried a credential, for every resource group, teams included. Its
// banner is auth status's.
func failCredentialRejected(cmd *cobra.Command, err error) error {
	hint := credentialRejectedHint()
	return failWith(cmd, &cliError{code: codeCredentialRejected, message: hint, status: 401, cause: err},
		"Credential rejected", hint)
}

// tenantRefusalDetail is the detail of the server's refusal of a tenant
// header naming a team the caller cannot use. A server that names the
// header in source.header is recognised by that; this string is the
// fallback for one that does not, and must stay beside it.
const tenantRefusalDetail = "Invalid or unauthorized tenant specified"

// isTenantRefusal reports whether a 403 refused the team the request sent
// as its tenant header, rather than the record: a team is in scope, and
// the refusal names the tenant header, or carries the server's tenant
// refusal detail.
func isTenantRefusal(e *jsonapi.APIError) bool {
	if appConfig == nil || appConfig.Team == "" || e.Status != 403 {
		return false
	}
	return e.Header == "tenant" || e.Detail == tenantRefusalDetail
}

// failTeamRefused is the rendering of a tenant refusal: the team in scope
// is named, since the record the command asked for was never looked at.
func failTeamRefused(cmd *cobra.Command, err error) error {
	team := appConfig.Team
	return failWith(cmd, &cliError{
		code:    codeTeamNotAccessible,
		message: "The server refused team " + team + ": you are not a member of it, or the id is wrong.",
		status:  403,
		cause:   err,
	}, "Access denied",
		"You're authenticated, but the server refused team "+team+": you are not a member of it, or the id is wrong.",
		"Run 'truestamp teams list' to see your teams, or 'truestamp teams use' to change the team in scope.")
}
