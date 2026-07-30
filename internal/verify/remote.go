// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/bitcoin"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// RemoteOptions holds configuration for remote verification. The
// credential is applied by the process-wide [auth.Authorizer]; only the
// tenant scoping is carried here.
type RemoteOptions struct {
	APIURL       string
	Team         string // team ID, sent as tenant header
	ExpectedHash string // hex hash to compare against claims.hash

	// ExpectedSubjectType, when non-empty, is sent to the server as
	// `data.type` on /proof/verify. The server then asserts the posted
	// bundle's `t` matches; a mismatch returns a structured 422 with
	// meta.code == "subject_type_mismatch" (see truestamp-v2
	// kb/verification/proof-bundle-format.md). Mirrors
	// Options.ExpectedSubjectType for the local path.
	ExpectedSubjectType string
}

// apiEnvelope wraps the top-level API response.
type apiEnvelope struct {
	Result *apiResult `json:"result"`
	Errors []apiError `json:"errors"`
}

// apiResult is the verification result from the server.
type apiResult struct {
	ProofVersion    *int            `json:"proof_version"`
	SubjectID       *string         `json:"subject_id"`
	SubjectType     *string         `json:"subject_type"`
	GeneratedAt     *string         `json:"generated_at"`
	Source          *string         `json:"source"`
	Passed          bool            `json:"passed"`
	Temporal        TemporalSummary `json:"temporal"`
	HashProvided    *string         `json:"hash_provided"`
	HashMatched     bool            `json:"hash_matched"`
	SkippedExternal bool            `json:"skipped_external"`
	Steps           []Step          `json:"steps"`
	ItemID          *string         `json:"item_id"` // backward compat
}

// apiError represents an error from the JSON:API response.
type apiError struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// RunRemote calls [RunRemoteCtx] with [context.Background].
func RunRemote(filename string, opts RemoteOptions) (*Report, error) {
	return RunRemoteCtx(context.Background(), filename, opts)
}

// RunRemoteCtx sends the proof to the Truestamp API for server-side
// verification and returns a Report compatible with the local verification
// output. ctx cancels the in-flight request and bounds any token refresh.
func RunRemoteCtx(ctx context.Context, filename string, opts RemoteOptions) (*Report, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading proof file: %w", err)
	}

	// Single parse to extract type and preserve raw JSON for the API request
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parsing proof: %w", err)
	}

	// If the input is CBOR, convert to JSON for the API request
	jsonData := data
	if proof.IsCBORProof(data) {
		jsonData, err = bundle.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("converting CBOR proof to JSON: %w", err)
		}
	}

	// Route to the unified proof verification endpoint
	endpoint := "/proof/verify"

	// Build request body
	var raw json.RawMessage = jsonData
	dataFields := map[string]any{
		"proof": raw,
	}
	if opts.ExpectedHash != "" {
		dataFields["expected_hash"] = opts.ExpectedHash
	}
	// --type is asserted HERE, against the same bytes about to be
	// posted, and only forwarded when it already holds.
	//
	// Forwarding a mismatching type makes the server answer 4xx with
	// meta.code="subject_type_mismatch", which this client can only
	// surface as a bare error string: no Report, no steps, no --json
	// document at all. Local mode instead grades a Subject Type fail and
	// runs everything else, which is the whole point of the flag — see
	// whether the file is otherwise a sound proof of something else.
	// RunRemoteCtx promises "a Report compatible with the local
	// verification output", so the two modes must agree on the one
	// condition --type exists to detect.
	//
	// When the assertion holds the type is still forwarded, so the
	// server re-checks it against its own reading of the bundle.
	subjectTypeMismatch := ""
	if opts.ExpectedSubjectType != "" {
		if actual := ptype.Name(bundle.T); actual != opts.ExpectedSubjectType {
			subjectTypeMismatch = fmt.Sprintf(
				"Proof is %s (t=%d) but --type %s was requested",
				actual, bundle.T, opts.ExpectedSubjectType)
		} else {
			dataFields["type"] = opts.ExpectedSubjectType
		}
	}
	requestBody := map[string]any{
		"data": dataFields,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encoding request body: %w", err)
	}

	// POST to the API
	url := opts.APIURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	if err := auth.AuthorizeRequest(ctx, req); err != nil {
		return nil, err
	}
	if opts.Team != "" {
		req.Header.Set("tenant", opts.Team)
	}

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, httpclient.MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading API response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("parsing API response: %w", err)
	}
	if envelope.Result == nil {
		return nil, fmt.Errorf("API response missing 'result' field")
	}

	report := mapToReport(filename, int64(len(data)), envelope.Result, opts)
	report.APIURL = opts.APIURL // for presenter's subject-detail + verify web links

	// Extract display data from the already-parsed proof bundle
	populateFromBundle(report, bundle)

	// Mirror local mode's Subject Type row so both modes report the same
	// assertion the same way.
	if subjectTypeMismatch != "" {
		report.fail(groupSubjectType, CatStructural, subjectTypeMismatch)
	}

	// E.7 is enforced by this process, on the bundle it just parsed.
	applyExpectedHash(report, bundle, opts, envelope.Result)

	// Finally, reconcile the server's own top-level verdict with the
	// step list it sent.
	applyServerVerdict(report, envelope.Result)

	return report, nil
}

// applyExpectedHash performs Appendix E.7's comparison locally, on the
// same two operands the local pipeline uses: the caller's --hash and the
// `s.d.hash` of the bundle this process parsed and posted.
//
// --remote used to forward `expected_hash` to the server and then never
// consult it again, dropping `hash_matched` on the floor. A server that
// ignored the field produced hash_comparison.supplied:false at exit 0; a
// server that echoed a mismatch produced a document that said
// matched:false and result:"verified" in the same breath. The identical
// invocation in local mode exits 1. --hash exists to answer "is my local
// file the timestamped one", and the CLI holds both operands, so the
// answer is computed here rather than taken on trust.
//
// The branch structure mirrors runBundle's E.7 switch deliberately: the
// two modes must reach the same status on the same bundle, including the
// two REQUIRED inapplicability skips (a non-item subject, and an item
// that timestamped its claims content and carries no s.d.hash).
func applyExpectedHash(r *Report, bundle *proof.ProofBundle, opts RemoteOptions, result *apiResult) {
	if opts.ExpectedHash == "" {
		return
	}

	var status Status
	var msg string
	switch {
	case !bundle.IsItem():
		status, msg = StatusSkip, fmt.Sprintf(
			"--hash not applicable: only an item subject (t=20) commits to a file hash (this proof is %s)",
			ptype.Name(bundle.T))
	case r.Claims.Hash == "":
		status, msg = StatusSkip,
			"--hash not applicable: this item timestamped its claims content itself and carries no s.d.hash to compare a local file against"
	case tscrypto.HexEqual(opts.ExpectedHash, r.Claims.Hash):
		status, msg = StatusPass, "Provided hash matches claims.hash"
	default:
		status, msg = StatusFail, fmt.Sprintf(
			"Provided hash does not match claims.hash (expected: %s, proof: %s)",
			opts.ExpectedHash, r.Claims.Hash)
	}

	// Snapshot the server's own rows in this group before adding ours,
	// so the two are distinguishable afterwards.
	agrees, serverRowClaimsMatch := false, false
	for _, s := range r.Steps {
		if s.Group != groupHashComparison {
			continue
		}
		agrees = agrees || s.Status == status
		serverRowClaimsMatch = serverRowClaimsMatch || s.Status == StatusPass
	}

	// Emitting a second row that agrees with the server's would
	// double-report one check, so the local row is added only when the
	// server reported nothing or reported something else. When they
	// disagree both rows stand, and the fail among them decides the
	// verdict — [Report.HashMatched] treats a fail in the group as
	// decisive against a match.
	if !agrees {
		r.Steps = append(r.Steps, Step{
			Group: groupHashComparison, Category: CatDataIntegrity,
			Status: status, Message: msg,
		})
	}

	// The server's own `hash_matched` is read rather than discarded, but
	// only when it acknowledged receiving an expected hash: the field is
	// a bare bool, so an absent one is indistinguishable from false and
	// would refute every genuine match.
	echoed := result.HashProvided != nil && *result.HashProvided != ""
	serverClaimsMatch := serverRowClaimsMatch || (echoed && result.HashMatched)

	switch {
	case serverClaimsMatch && status != StatusPass:
		// The fail-open direction, and the one that matters: the server
		// asserts the caller's file is the timestamped one over a
		// comparison this process either refuted or could not make.
		r.fail(groupHashComparison, CatDataIntegrity, fmt.Sprintf(
			"Expected-hash disagreement: the server reports the supplied hash as matching, this verifier's own comparison against the posted bundle reports %s",
			statusStrings[status]))
	case echoed && status == StatusPass && !result.HashMatched:
		// The converse. One of the two verifiers is wrong about the
		// caller's data either way, which is not a state to publish as
		// verified.
		r.fail(groupHashComparison, CatDataIntegrity,
			"Expected-hash disagreement: the server reports the supplied hash as NOT matching, this verifier's own comparison against the posted bundle reports pass")
	}
}

// applyServerVerdict reconciles the server's top-level `passed` field
// with the step list it sent alongside it.
//
// `passed` was parsed and never read, so the verdict came only from the
// steps — and a response that reported passed:false with no failing step
// (or with no steps at all) rendered "VERIFIED - proof is valid" at exit
// 0. E.22's verdict rule reads step statuses, so the fix is to give the
// server's own verdict a step of its own rather than to bypass the rule.
func applyServerVerdict(r *Report, result *apiResult) {
	// The server's OWN step list, not r.Steps: by this point the CLI may
	// have appended its E.7 row or a Subject Type row, and an empty
	// server response must not be rescued by a row the CLI added.
	if len(result.Steps) == 0 {
		r.fail(groupServerVerdict, CatStructural,
			"Server returned no verification steps: this run establishes nothing about the proof")
		return
	}
	// Likewise the server's own steps, not r.Steps: a failure the CLI
	// raised for its own reasons (an E.7 mismatch, a --type assertion)
	// does not stand in for the server's verdict. Without this the
	// report could read "HASH MISMATCH - proof is valid" over a proof
	// the server had just reported as not verified.
	serverFailed := false
	for _, s := range result.Steps {
		if s.Status == StatusFail {
			serverFailed = true
			break
		}
	}
	if !result.Passed && !serverFailed {
		r.fail(groupServerVerdict, CatStructural,
			"Server reported the proof as NOT verified (passed: false) but sent no failing step: the reported outcome and the reported steps disagree")
	}
}

// groupServerVerdict carries results derived from the server's own
// top-level verdict fields rather than from a step it graded. It is a
// CLI-specific group, reachable only on the --remote path, and the local
// pipeline never emits it.
const groupServerVerdict = "Server Verdict"

// mapToReport converts the API result to a Report struct.
func mapToReport(filename string, fileSize int64, result *apiResult, opts RemoteOptions) *Report {
	r := &Report{
		Filename:        filename,
		FileSize:        fileSize,
		Temporal:        result.Temporal,
		Steps:           result.Steps,
		Remote:          true,
		SkippedExternal: result.SkippedExternal,
	}

	if result.ProofVersion != nil {
		r.ProofVersion = *result.ProofVersion
	}
	if result.SubjectID != nil {
		r.SubjectID = *result.SubjectID
	} else if result.ItemID != nil {
		r.SubjectID = *result.ItemID
	}
	if result.SubjectType != nil {
		r.SubjectType = *result.SubjectType
	} else {
		r.SubjectType = "item"
	}
	if result.GeneratedAt != nil {
		r.GeneratedAt = *result.GeneratedAt
	}
	if result.Source != nil {
		r.Source = *result.Source
	}
	// E.22 requires "an expected hash was supplied" to be readable
	// separately from "it matched", and the caller is the authority on
	// the first: taking it only from the server's echo published
	// supplied:false for a run that passed --hash to a server that
	// ignored the field.
	if opts.ExpectedHash != "" {
		r.HashProvided = opts.ExpectedHash
	} else if result.HashProvided != nil {
		r.HashProvided = *result.HashProvided
	}

	return r
}

// populateFromBundle extracts display data from a parsed proof bundle
// for presenter parity with local mode.
func populateFromBundle(r *Report, bundle *proof.ProofBundle) {
	subject := bundle.Subject
	t := bundle.T

	r.SubjectType = ptype.Name(t)

	if r.SubjectID == "" {
		// Block-like (plain block or beacon): SubjectID falls back to the
		// block id. Otherwise use the Subject.ID (item or entropy).
		if bundle.IsBlockLike() {
			r.SubjectID = bundle.Block.ID
		} else if subject != nil {
			r.SubjectID = subject.ID
		}
	}

	// Claims (item proofs)
	if bundle.IsItem() && subject != nil {
		r.Claims = parseClaims(bundle.RawData)

		// Derive TimestampStatus from steps (server already validated)
		for _, s := range r.Steps {
			if s.Status == StatusWarn {
				if strings.Contains(s.Message, "future-dated claim") {
					r.Claims.TimestampStatus = TimestampFuture
					r.Claims.TimestampNote = s.Message
				} else if strings.Contains(s.Message, "stale claim") {
					r.Claims.TimestampStatus = TimestampStale
					r.Claims.TimestampNote = s.Message
				}
			}
		}

		if r.Temporal.SubmittedAt == "" {
			r.Temporal.SubmittedAt = tscrypto.FormatItemTime(subject.ID)
		}

		if r.Temporal.ClaimedAt == "" {
			if ts := extractClaimsTimestamp(subject.Data); ts != "" {
				r.Temporal.ClaimedAt = truncateToSecond(ts)
			}
		}
	}

	if bundle.IsEntropy() && subject != nil {
		r.EntropySubject = parseEntropySubject(t, subject.Data)

		if r.Temporal.CapturedAt == "" {
			r.Temporal.CapturedAt = tscrypto.FormatBlockTime(subject.ID)
		}
	}

	r.ChainLength = 1

	// Name the key that actually signed the bundle, which E.9/E.16 make
	// the pk-DERIVED key id — not the stored b.kid, which E.9 blesses
	// diverging from it under key rotation. The local path derives the
	// same value from the same field; reading b.kid here made the two
	// modes attribute one signature to two different keys.
	// A `pk` that does not decode yields no derived key id, and the
	// local path leaves the field empty rather than substituting b.kid —
	// naming b.kid as the signer is the very claim this is fixing.
	r.BlockSigningKeyID = bundle.Block.SigningKeyID
	r.SigningKeyID = ""
	if pubkey, err := tscrypto.DecodePublicKey(bundle.PublicKey); err == nil {
		r.SigningKeyID = tscrypto.ComputeKeyID(pubkey)
	}

	if r.Temporal.CommittedAt == "" {
		r.Temporal.CommittedAt = tscrypto.FormatBlockTime(bundle.Block.ID)
	}

	for i := range bundle.Commitments {
		cx := &bundle.Commitments[i]

		switch cx.Type {
		case ptype.CommitmentStellar:
			ci := CommitmentInfo{
				Method:        "stellar",
				Network:       cx.Network,
				Ledger:        cx.Ledger,
				TxHash:        cx.TransactionHash,
				CommittedHash: cx.MemoHash,
				Timestamp:     cx.Timestamp,
				ExternalCheck: remoteExternalCheck(r, groupStellar),
			}
			r.CommitmentInfos = append(r.CommitmentInfos, ci)

		case ptype.CommitmentBitcoin:
			ci := CommitmentInfo{
				Method:        "bitcoin",
				Network:       cx.Network,
				Height:        cx.BlockHeight,
				TxHash:        cx.TransactionHash,
				CommittedHash: cx.OpReturn,
				Timestamp:     cx.Timestamp,
				ExternalCheck: remoteExternalCheck(r, groupBitcoin),
			}
			if cx.TxoutproofHex != "" {
				if blockHash, err := bitcoin.ParseBlockHash(cx.TxoutproofHex); err == nil {
					ci.BlockHash = blockHash
				}
			}
			r.CommitmentInfos = append(r.CommitmentInfos, ci)
		}
	}
}

// remoteExternalCheck derives a commitment's external-confirmation
// status from the server's own step rows for that group, because
// /proof/verify reports no per-commitment status on the wire.
//
// The rule is deliberately conservative: confirmed only when the group
// carries at least one pass and nothing else unresolved. E.19 requires a
// commitment whose binding could not be established to be reported
// skipped, never passing, and the server emits a skip row for exactly
// that case (a regtest commitment, or a run with external checks off).
// Reading "some row passed" as confirmed would reintroduce, one level
// removed, the same fabricated confirmation the local path just lost.
func remoteExternalCheck(r *Report, group string) ExternalStatus {
	if r.SkippedExternal {
		return ExternalSkipped
	}
	sawPass, sawUnresolved := false, false
	for _, s := range r.Steps {
		if s.Group != group {
			continue
		}
		switch s.Status {
		case StatusFail:
			// A failure anywhere in the group is decisive.
			return ExternalFailed
		case StatusPass:
			sawPass = true
		case StatusSkip, StatusWarn:
			sawUnresolved = true
		}
	}
	if sawPass && !sawUnresolved {
		return ExternalConfirmed
	}
	return ExternalSkipped
}

// parseAPIError extracts a meaningful error message from the API response body.
func parseAPIError(statusCode int, body []byte) error {
	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		if first.Detail != "" {
			return fmt.Errorf("API error (HTTP %d): %s", statusCode, first.Detail)
		}
		if first.Title != "" {
			return fmt.Errorf("API error (HTTP %d): %s", statusCode, first.Title)
		}
	}

	bodyStr := string(body)
	if len(bodyStr) > 0 && bodyStr[0] == '<' {
		return fmt.Errorf("API error (HTTP %d): server returned HTML error page", statusCode)
	}

	return fmt.Errorf("API error (HTTP %d): %s", statusCode, httpclient.Truncate(bodyStr, 200))
}
