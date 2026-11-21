// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package proof fetches proofs from the Truestamp service: the exact bytes
// of a proof from the raw proof route, the name `proofs get --to-file`
// gives a proof, the service's refusals to serve one, and the shape of a
// subject id and the server's resolution of an id to its subject type. It
// reads no proof's members to verify or show it: `truestamp verify` and
// `truestamp inspect` read a proof through internal/verifier.
package proof

import (
	"context"
	"encoding/hex"
	"encoding/json/v2"
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
// An item's code is computed from its recomputed item_hash
// ([ServedItemHash]), never from the id given or from any hash the
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
	itemHash, err := ServedItemHash(served)
	if err != nil {
		return "", fmt.Errorf("no file name for the served proof: %w", err)
	}
	return itemcode.FromItemHash(itemHash) + ".proof.json", nil
}

// ServedItemHash is the item_hash of the item proof served, recomputed
// from its subject's record as H(0x11, JCS(proof.subject.record)), the
// row item_hash of SPEC.md section 4.3 (The table) that DERIVE recomputes
// (SPEC.md section 10.2, step 9), never the item_hash the document
// carries. A served document the strict reader refuses, or that is not an
// item proof with a record, gives an error.
func ServedItemHash(served []byte) ([32]byte, error) {
	record, err := servedItemRecord(served)
	if err != nil {
		return [32]byte{}, err
	}
	return itemcode.ItemHashOfRecord(record)
}

// servedItemRecord is /proof/subject/record of the item proof served, as
// the strict reader parses it. A served document the strict reader
// refuses, or that is not an item proof with a record, gives an error.
func servedItemRecord(served []byte) (map[string]any, error) {
	value, err := strictjson.Parse(served)
	if err != nil {
		return nil, err
	}
	envelope, _ := value.(map[string]any)
	document, _ := envelope["proof"].(map[string]any)
	if document["type"] != "item" {
		return nil, errors.New("it is not an item proof")
	}
	subject, _ := document["subject"].(map[string]any)
	record, _ := subject["record"].(map[string]any)
	if record == nil {
		return nil, errors.New("it carries no item record")
	}
	return record, nil
}

// ErrDifferentItem is a served item proof that is not of the item asked
// for: [CheckServedItem]'s refusal.
var ErrDifferentItem = errors.New("the server returned a proof for a different item")

// CheckServedItem checks that served, the proof served for a subject of
// subjectType named by ref, is a proof of that item, before anything is
// written. shape is ref's shape and ref its canonical form, as
// [DetectIDType] gives them. Only an item is checked: for any other
// subjectType it gives nil. For a UUIDv7, the id of the served record,
// /proof/subject/record/id (SPEC.md section 3.6 (Layouts by type)), must
// equal ref, and nothing else of the proof is read; for an item code, the
// item code of the recomputed item_hash ([ServedItemHash]) must; for an
// item_hash, the recomputed item_hash itself must, all 64 hex characters.
// The item_hash the document carries is never read. A mismatch is
// [ErrDifferentItem]. A served document that is not an item proof with a
// record is an error too, and so is a record with no string id, for a
// UUIDv7, or one whose item_hash cannot be recomputed, for a code or an
// item_hash.
//
// A matching code shows only that the recomputed item_hash begins with
// the 75 bits the code is written from (SPEC.md section 3.9), so a code
// checks the item a person named, and an item_hash checks it whole.
func CheckServedItem(subjectType string, shape IDType, ref string, served []byte) error {
	if subjectType != "item" {
		return nil
	}
	if shape == IDTypeUUIDv7 {
		record, err := servedItemRecord(served)
		if err != nil {
			return fmt.Errorf("the served proof cannot be checked against %s, so nothing was written: %w", ref, err)
		}
		id, ok := record["id"].(string)
		if !ok {
			return fmt.Errorf("the served proof cannot be checked against %s, so nothing was written: its item record has no string id", ref)
		}
		if id != ref {
			return fmt.Errorf("%w: its record's id is %q, not %s; nothing was written", ErrDifferentItem, id, ref)
		}
		return nil
	}
	itemHash, err := ServedItemHash(served)
	if err != nil {
		return fmt.Errorf("the served proof cannot be checked against %s, so nothing was written: %w", ref, err)
	}
	recomputed := hex.EncodeToString(itemHash[:])
	if shape == IDTypeItemHash {
		if recomputed != ref {
			return fmt.Errorf("%w: its recomputed item_hash is %s, not %s; nothing was written", ErrDifferentItem, recomputed, ref)
		}
		return nil
	}
	if code := itemcode.FromItemHash(itemHash); code != ref {
		return fmt.Errorf("%w: its recomputed item_hash %s has the item code %s, not %s; nothing was written", ErrDifferentItem, recomputed, code, ref)
	}
	return nil
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
