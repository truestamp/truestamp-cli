// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

type horizonTx struct {
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

// VerifyStellar checks the Stellar Horizon API to confirm the transaction exists,
// the memo matches, and the ledger number matches the expected value.
// Returns the ledger number, the transaction timestamp, or an error.
func VerifyStellar(transactionHash, expectedMemoHash, network string, expectedLedger int) (*StellarResult, error) {
	if _, err := hex.DecodeString(transactionHash); err != nil || len(transactionHash) != 64 {
		return nil, fmt.Errorf("invalid transaction hash format")
	}

	var horizonURL string
	switch network {
	case "public":
		horizonURL = HorizonPublicURL
	case "testnet":
		horizonURL = HorizonTestnetURL
	default:
		return nil, fmt.Errorf("unknown Stellar network %q (expected \"testnet\" or \"public\")", network)
	}

	url := fmt.Sprintf("%s/transactions/%s", horizonURL, transactionHash)
	body, err := httpclient.GetJSON(url)
	if err != nil {
		return nil, fmt.Errorf("fetching Stellar transaction: %w", err)
	}

	var tx horizonTx
	if err := json.Unmarshal(body, &tx); err != nil {
		return nil, fmt.Errorf("parsing Horizon response: %w", err)
	}

	if tx.MemoType != "hash" {
		return nil, fmt.Errorf("expected memo_type 'hash', got '%s'", tx.MemoType)
	}

	// Horizon returns memo as base64
	memoBytes, err := base64.StdEncoding.DecodeString(tx.Memo)
	if err != nil {
		return nil, fmt.Errorf("decoding memo base64: %w", err)
	}

	decodedMemo := hex.EncodeToString(memoBytes)
	if !strings.EqualFold(decodedMemo, expectedMemoHash) {
		return nil, fmt.Errorf("memo mismatch: expected %s, got %s", expectedMemoHash, decodedMemo)
	}

	if expectedLedger > 0 && tx.Ledger != expectedLedger {
		return nil, fmt.Errorf("ledger mismatch: expected %d, got %d", expectedLedger, tx.Ledger)
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
// against the entropy subject data; network selection (testnet|public)
// is the caller's responsibility.
func GetStellarLedger(sequence int, network string) (*StellarLedger, error) {
	var horizonURL string
	switch network {
	case "public":
		horizonURL = HorizonPublicURL
	case "testnet":
		horizonURL = HorizonTestnetURL
	default:
		return nil, fmt.Errorf("unknown Stellar network %q (expected \"testnet\" or \"public\")", network)
	}
	if sequence <= 0 {
		return nil, fmt.Errorf("invalid sequence: %d", sequence)
	}

	url := fmt.Sprintf("%s/ledgers/%d", horizonURL, sequence)
	body, err := httpclient.GetJSON(url)
	if err != nil {
		return nil, fmt.Errorf("fetching Stellar ledger: %w", err)
	}
	var l horizonLedger
	if err := json.Unmarshal(body, &l); err != nil {
		return nil, fmt.Errorf("parsing Horizon ledger response: %w", err)
	}
	if l.Hash == "" {
		return nil, fmt.Errorf("horizon response missing ledger hash")
	}
	return &StellarLedger{
		Sequence: l.Sequence,
		Hash:     l.Hash,
		ClosedAt: l.ClosedAt,
	}, nil
}
