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

// VerifyStellar checks the Stellar Horizon API to confirm the transaction exists,
// the memo matches, and the ledger number matches the expected value.
// Returns the ledger number, the transaction timestamp, or an error.
func VerifyStellar(transactionHash, expectedMemoHash, network string, expectedLedger int) (*StellarResult, error) {
	if _, err := hex.DecodeString(transactionHash); err != nil || len(transactionHash) != 64 {
		return nil, fmt.Errorf("invalid transaction hash format")
	}

	horizonURL := "https://horizon-testnet.stellar.org"
	if network == "public" {
		horizonURL = "https://horizon.stellar.org"
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