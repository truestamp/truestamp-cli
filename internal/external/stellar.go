// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// horizonTx is the subset of Horizon's transaction resource E.18 reads.
// `hash` is what ties the answer to the question: Horizon echoes the
// transaction hash it was asked for, and without comparing it a body
// describing some other transaction is graded as though it described
// this one.
type horizonTx struct {
	Hash      string `json:"hash"`
	MemoType  string `json:"memo_type"`
	Memo      string `json:"memo"`
	Ledger    int    `json:"ledger"`
	CreatedAt string `json:"created_at"`
}

// StellarResult holds the verification result from the Stellar Horizon API.
type StellarResult struct {
	Ledger    int
	Timestamp string // ISO 8601 ledger close timestamp from Horizon
}

// HorizonPublicURL / HorizonTestnetURL are the Stellar Horizon endpoints
// consulted by VerifyStellar. Exposed as vars so tests can point them at
// an httptest server; do not mutate in production code.
var (
	HorizonPublicURL  = "https://horizon.stellar.org"
	HorizonTestnetURL = "https://horizon-testnet.stellar.org"
)

// Stellar network names carried in a commitment's `net` field.
const (
	stellarPublic  = "public"
	stellarTestnet = "testnet"
)

// horizonBaseURL implements whitepaper E.18's endpoint rule verbatim:
// "public" selects the public instance, anything else, including an
// absent `net`, selects the testnet instance. There is deliberately no
// error branch. E.5 forbids the absence of an optional field from
// skipping or failing a confirmation step, so an unnamed network is
// looked up against the default and graded like any other lookup.
func horizonBaseURL(network string) string {
	if network == stellarPublic {
		return HorizonPublicURL
	}
	return HorizonTestnetURL
}

// IsDefaultedNetwork reports whether the entry named no Horizon
// instance, so the lookup fell through to E.18's default. A 404 from a
// defaulted lookup is not a definitive absence from the chain: the
// transaction may simply live on the network the entry failed to name.
func IsDefaultedNetwork(network string) bool {
	return network != stellarPublic && network != stellarTestnet
}

// NetworkLabel names the Horizon instance actually queried, so a report
// says which chain answered rather than echoing an empty `net`.
func NetworkLabel(network string) string {
	if network == stellarPublic {
		return stellarPublic
	}
	return stellarTestnet
}

// VerifyStellar checks the Stellar Horizon API to confirm the transaction exists,
// the memo matches, and the ledger number matches the expected value.
// Returns the ledger number, the transaction timestamp, or an error.
// Errors are typed for [Classify]: the caller decides which outcomes
// fail the step and which only leave it unconfirmed.
func VerifyStellar(transactionHash, expectedMemoHash, network string, expectedLedger int) (*StellarResult, error) {
	if transactionHash == "" {
		return nil, &BadInputError{Field: "transaction hash", Detail: "the entry carries no transaction id to look up"}
	}
	if _, err := hex.DecodeString(transactionHash); err != nil || len(transactionHash) != 64 {
		return nil, &BadInputError{Field: "transaction hash", Detail: "not a 64-character hex string"}
	}

	url := fmt.Sprintf("%s/transactions/%s", horizonBaseURL(network), transactionHash)
	body, err := httpclient.GetJSON(url)
	if err != nil {
		return nil, fmt.Errorf("fetching Stellar transaction: %w", err)
	}

	var tx horizonTx
	if err := json.Unmarshal(body, &tx); err != nil {
		return nil, &MalformedResponseError{Source: "Horizon", Detail: "response is not valid JSON", Err: err}
	}

	// The answer MUST be about the transaction that was asked for. A
	// body carrying no `hash`, or naming a different transaction, did
	// not answer this question. Graded Malformed rather than Mismatch:
	// the bundle contributed only the lookup key, which was sent
	// verbatim, so a mis-addressed answer is an upstream property and
	// cannot be evidence against the bundle (E.22 skips it).
	if tx.Hash == "" {
		return nil, &MalformedResponseError{Source: "Horizon", Detail: "transaction response carries no hash"}
	}
	if !tscrypto.HexEqual(tx.Hash, transactionHash) {
		return nil, &MalformedResponseError{
			Source: "Horizon",
			Detail: fmt.Sprintf("transaction response describes %s, not the requested %s", tx.Hash, transactionHash),
		}
	}

	// E.18: the transaction's memo_type MUST be `hash`. A transaction
	// that carries another memo type cannot be the commitment, but an
	// ABSENT memo_type is not a transaction saying its memo is not a
	// hash, it is a body with no memo_type in it. Every sibling lookup
	// grades a missing expected field Malformed (see [GetStellarLedger],
	// [GetNISTPulse], [GetBitcoinBlockHeader]); grading it Mismatch here
	// would fail a sound proof with "memo_type mismatch: expected hash,
	// got ", a positive claim about the chain from a body that carried
	// no transaction.
	if tx.MemoType == "" {
		return nil, &MalformedResponseError{Source: "Horizon", Detail: "transaction response carries no memo_type"}
	}
	if tx.MemoType != "hash" {
		return nil, &MismatchError{Field: "memo_type", Expected: "hash", Got: tx.MemoType}
	}

	// Horizon returns memo as base64
	memoBytes, err := base64.StdEncoding.DecodeString(tx.Memo)
	if err != nil {
		return nil, &MalformedResponseError{Source: "Horizon", Detail: "memo base64 is undecodable", Err: err}
	}

	decodedMemo := hex.EncodeToString(memoBytes)
	if !tscrypto.HexEqual(decodedMemo, expectedMemoHash) {
		return nil, &MismatchError{Field: "memo", Expected: expectedMemoHash, Got: decodedMemo}
	}

	if expectedLedger > 0 && tx.Ledger != expectedLedger {
		return nil, &MismatchError{
			Field:    "ledger",
			Expected: fmt.Sprintf("%d", expectedLedger),
			Got:      fmt.Sprintf("%d", tx.Ledger),
		}
	}

	return &StellarResult{
		Ledger:    tx.Ledger,
		Timestamp: tx.CreatedAt,
	}, nil
}

// horizonLedger captures the fields the CLI compares against entropy
// subject data. Horizon returns many more; we only read what we need.
type horizonLedger struct {
	Sequence int    `json:"sequence"`
	Hash     string `json:"hash"`
	ClosedAt string `json:"closed_at"`
}

// StellarLedger is the CLI-facing return shape for GetStellarLedger.
type StellarLedger struct {
	Sequence int
	Hash     string
	ClosedAt string
}

// GetStellarLedger fetches a specific ledger from Stellar Horizon at
// /ledgers/{sequence}. The caller compares returned hash + closed_at
// against the entropy subject data. Endpoint selection follows E.18's
// rule (see [horizonBaseURL]), so an entry that names no network is
// still looked up against the testnet instance.
func GetStellarLedger(sequence int, network string) (*StellarLedger, error) {
	if sequence <= 0 {
		return nil, &BadInputError{Field: "ledger sequence", Detail: fmt.Sprintf("%d is not a valid sequence", sequence)}
	}

	url := fmt.Sprintf("%s/ledgers/%d", horizonBaseURL(network), sequence)
	body, err := httpclient.GetJSON(url)
	if err != nil {
		return nil, fmt.Errorf("fetching Stellar ledger: %w", err)
	}
	var l horizonLedger
	if err := json.Unmarshal(body, &l); err != nil {
		return nil, &MalformedResponseError{Source: "Horizon", Detail: "ledger response is not valid JSON", Err: err}
	}
	if l.Hash == "" {
		return nil, &MalformedResponseError{Source: "Horizon", Detail: "ledger response carries no hash"}
	}
	// The caller byte-compares this hash against the entropy subject's
	// (E.21), so an answer about a different ledger would be reported as
	// a value mismatch and fail a sound proof. Horizon echoes the
	// sequence it was asked for; a disagreement is upstream, not the
	// bundle's, and must skip.
	if l.Sequence != sequence {
		return nil, &MalformedResponseError{
			Source: "Horizon",
			Detail: fmt.Sprintf("ledger response describes sequence %d, not the requested %d", l.Sequence, sequence),
		}
	}
	return &StellarLedger{
		Sequence: l.Sequence,
		Hash:     l.Hash,
		ClosedAt: l.ClosedAt,
	}, nil
}
