// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package proof

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/itemcode"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// The constructions modes of SPEC.md section 7.2 (Modes), as the raw
// proof route's `constructions` parameter spells them: how the proof
// carries the constructions text. Each mode is its own signed document,
// and the verdict never depends on it.
const (
	// ConstructionsFull carries the text itself, the default.
	ConstructionsFull = "full"
	// ConstructionsReference carries the text's URL, which names its
	// SHA-256.
	ConstructionsReference = "reference"
	// ConstructionsNone carries neither.
	ConstructionsNone = "none"
)

// ConstructionsParameter is the name of the raw proof route's query
// parameter that gives the constructions mode, and of the `proofs get`
// flag that sets it. It names a request's parameter, never the proof's
// member: SPEC.md section 7.2, item 2, lets a verifier read that member
// only at SHAPE, and this constant is never an index into a document.
const ConstructionsParameter = "constructions"

// ConstructionsModes is every constructions mode, in the order help text
// lists them.
var ConstructionsModes = []string{ConstructionsFull, ConstructionsReference, ConstructionsNone}

// FetchRawCtx reads the proof Truestamp serves of the subject id of
// subjectType, one of the six registry names, in the constructions mode
// constructions, from the raw proof route,
//
//	GET {apiURL}/proof/raw/{subjectType}/{id}?constructions={constructions}
//
// and returns the body exactly as served: the served form of SPEC.md
// section 3.1 (The envelope), never decoded, re-encoded or re-indented, so
// a file written from it holds the bytes Truestamp signed and served. Only
// a 200 is a proof. A refusal is a [*GenerateAPIError] when it names a
// detail, a rate limit the [*jsonapi.APIError] of a 429, and a body over
// MAX_READ_OCTETS, which no verifier reads, an error. The credential is
// applied by the process-wide auth.Authorizer and team is sent as the
// tenant header.
func FetchRawCtx(ctx context.Context, apiURL, team, subjectType, id, constructions string) ([]byte, error) {
	query := url.Values{}
	query.Set(ConstructionsParameter, constructions)
	path := "/proof/raw/" + url.PathEscape(subjectType) + "/" + url.PathEscape(id) + "?" + query.Encode()
	resp, body, err := jsonapi.DoRaw(ctx, jsonapi.Config{APIURL: apiURL, Team: team}, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		// A rate-limit refusal is the shared envelope (errors[].code
		// `rate_limited`, the wait in meta.retry_after_ms), not a generate
		// refusal, and it has already been retried once by jsonapi.Send.
		return nil, jsonapi.ErrorFromResponse(resp, body)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, parseGenerateError(resp.StatusCode, body)
	}
	if len(body) == 0 {
		return nil, errors.New("the server answered with an empty proof")
	}
	if len(body) > strictjson.MaxReadOctets {
		return nil, fmt.Errorf("the served proof is over %d octets, more than a verifier reads", strictjson.MaxReadOctets)
	}
	return body, nil
}

// FileName is the name `proofs get --to-file` gives the proof served of
// the subject id of subjectType: `<CODE>.proof.json` for an item, with
// CODE its item code (SPEC.md section 3.9 (The item code)), and
// `<id>.proof.json` for every other subject, with `.proof.json` the
// extension SPEC.md section 3.1 (The envelope) gives a proof file. The
// constructions mode adds nothing to the name.
//
// An item's code is computed from its recomputed item_hash, H(0x11,
// JCS(proof.subject.record)) as SPEC.md section 4.3 (The table) gives
// it, of the served bytes, never from the id given or from any hash the
// document carries; a served document that is not an item proof with a
// record gives no name. An id that holds a path separator, or is empty,
// "." or "..", gives no name either, so a file is only ever written in
// the current directory.
func FileName(subjectType, id string, served []byte) (string, error) {
	if subjectType != "item" {
		if id == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\`) {
			return "", fmt.Errorf("the id %q cannot name a file", id)
		}
		return id + ".proof.json", nil
	}
	code, err := servedItemCode(served)
	if err != nil {
		return "", fmt.Errorf("no file name for the served proof: %w", err)
	}
	return code + ".proof.json", nil
}

// servedItemCode is the item code of the item proof served, from the
// item_hash recomputed over its subject's record.
func servedItemCode(served []byte) (string, error) {
	value, err := strictjson.Parse(served)
	if err != nil {
		return "", err
	}
	envelope, _ := value.(map[string]any)
	document, _ := envelope["proof"].(map[string]any)
	if document["type"] != "item" {
		return "", errors.New("it is not an item proof")
	}
	subject, _ := document["subject"].(map[string]any)
	record, _ := subject["record"].(map[string]any)
	if record == nil {
		return "", errors.New("it carries no item record")
	}
	return itemcode.OfItemRecord(record)
}

// Codes the server sends in `meta.code` when it refuses to serve a proof,
// the generate action's refusals.
//
// These are deliberately not reason codes of SPEC.md section 12.1 (Reason
// codes): those say why a verifier refuses or fails a proof in hand, and
// these say why the server could not serve one.
const (
	// GenerateCodeNoBlockCommitment means the subject's block has no
	// commitment on a public chain yet. Retryable.
	GenerateCodeNoBlockCommitment = "no_block_commitment"
	// GenerateCodeCommitmentNotFinal means the block's commitment is
	// recorded and none is final yet. Retryable.
	GenerateCodeCommitmentNotFinal = "commitment_not_final"
	// GenerateCodeEvidenceUnconfirmed means Truestamp's own chain reads did
	// not confirm the evidence the proof would carry. Retryable.
	GenerateCodeEvidenceUnconfirmed = "evidence_unconfirmed"
	// GenerateCodeSubjectNotReady means the subject is not committed in a
	// block yet. Retryable.
	GenerateCodeSubjectNotReady = "subject_not_ready"
	// GenerateCodeWitnessUnavailable means a witness the item's fingerprint
	// commits to could not be found, so no proof was served rather than one
	// without it. Retryable.
	GenerateCodeWitnessUnavailable = "witness_unavailable"
	// GenerateCodeSigningUnavailable means Truestamp's signer refused, so
	// no proof was served. Retryable.
	GenerateCodeSigningUnavailable = "signing_unavailable"
	// GenerateCodeLookupFailed means a record the proof is built from could
	// not be read. Retryable.
	GenerateCodeLookupFailed = "lookup_failed"
	// GenerateCodeSubjectRedacted means the item's claims are redacted, so
	// it has no proof until it is restored.
	GenerateCodeSubjectRedacted = "subject_redacted"
	// GenerateCodeProofTooLarge means the document would be larger than
	// the largest Truestamp serves.
	GenerateCodeProofTooLarge = "proof_too_large"
	// GenerateCodeGenerationFailed means Truestamp's records do not give a
	// proof its own verifier accepts.
	GenerateCodeGenerationFailed = "generation_failed"
)

// GenerateAPIError is a refusal of the raw proof route carrying the
// server's `meta.code` when one was sent, for example
// `no_block_commitment` for a subject whose block is not committed yet.
type GenerateAPIError struct {
	StatusCode int
	Code       string
	Detail     string
}

func (e *GenerateAPIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("API error (HTTP %d, %s): %s", e.StatusCode, e.Code, e.Detail)
	}
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Detail)
}

// parseGenerateError extracts a meaningful error message from the API response.
func parseGenerateError(statusCode int, body []byte) error {
	var envelope struct {
		Errors []struct {
			Detail string `json:"detail"`
			Title  string `json:"title"`
			Meta   struct {
				Code string `json:"code"`
			} `json:"meta"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		detail := first.Detail
		if detail == "" {
			detail = first.Title
		}
		if detail != "" {
			return &GenerateAPIError{StatusCode: statusCode, Code: first.Meta.Code, Detail: detail}
		}
	}

	bodyStr := string(body)
	if len(bodyStr) > 0 && bodyStr[0] == '<' {
		return fmt.Errorf("API error (HTTP %d): server returned HTML error page", statusCode)
	}
	return fmt.Errorf("API error (HTTP %d): %s", statusCode, httpclient.Truncate(bodyStr, 200))
}
