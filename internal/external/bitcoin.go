// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package external fetches supplementary data from public blockchains (Stellar
// Horizon, Bitcoin block explorers) to corroborate proofs against real-world
// public records.
package external

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

type blockstreamBlock struct {
	Height    int   `json:"height"`
	Timestamp int64 `json:"timestamp"`
}

// BitcoinResult holds the verification result from the Blockstream API.
type BitcoinResult struct {
	Height    int
	Timestamp string // ISO 8601 block timestamp from Blockstream
}

// BlockstreamMainnetURL / BlockstreamTestnetURL are the Blockstream API
// endpoints consulted by VerifyBitcoinBlock. Exposed as vars so tests
// can redirect them; do not mutate in production code.
var (
	BlockstreamMainnetURL = "https://blockstream.info/api"
	BlockstreamTestnetURL = "https://blockstream.info/testnet/api"
)

// VerifyBitcoinBlock checks the Blockstream API to confirm a Bitcoin block exists.
// Returns the block height and timestamp, or an error. For regtest, returns a skip indicator.
func VerifyBitcoinBlock(blockHash, network string) (*BitcoinResult, bool, error) {
	var baseURL string
	switch network {
	case "mainnet":
		baseURL = BlockstreamMainnetURL
	case "testnet":
		baseURL = BlockstreamTestnetURL
	default:
		return nil, true, nil // skipped
	}

	if _, err := hex.DecodeString(blockHash); err != nil || len(blockHash) != 64 {
		return nil, false, fmt.Errorf("invalid block hash format")
	}

	url := fmt.Sprintf("%s/block/%s", baseURL, blockHash)
	body, err := httpclient.GetJSON(url)
	if err != nil {
		return nil, false, fmt.Errorf("fetching Bitcoin block: %w", err)
	}

	var block blockstreamBlock
	if err := json.Unmarshal(body, &block); err != nil {
		return nil, false, fmt.Errorf("parsing Blockstream response: %w", err)
	}

	var ts string
	if block.Timestamp > 0 {
		ts = time.Unix(block.Timestamp, 0).UTC().Format(time.RFC3339)
	}

	return &BitcoinResult{
		Height:    block.Height,
		Timestamp: ts,
	}, false, nil
}

// blockstreamHeader is the subset of fields the CLI reads for
// entropy_bitcoin verification. Blockstream returns many more fields.
type blockstreamHeader struct {
	ID         string `json:"id"`
	Height     int    `json:"height"`
	Timestamp  int64  `json:"timestamp"`
	MerkleRoot string `json:"merkle_root"`
}

// BitcoinBlockHeader is the CLI-facing return shape for GetBitcoinBlockHeader.
type BitcoinBlockHeader struct {
	Hash       string
	Height     int
	Time       int64
	MerkleRoot string
}

// GetBitcoinBlockHeader fetches a Bitcoin block header from Blockstream by
// block hash. Returns (nil, skipped=true, nil) for regtest (no public API).
func GetBitcoinBlockHeader(blockHash, network string) (*BitcoinBlockHeader, bool, error) {
	var baseURL string
	switch network {
	case "mainnet":
		baseURL = BlockstreamMainnetURL
	case "testnet":
		baseURL = BlockstreamTestnetURL
	default:
		return nil, true, nil
	}
	if _, err := hex.DecodeString(blockHash); err != nil || len(blockHash) != 64 {
		return nil, false, fmt.Errorf("invalid block hash format")
	}
	url := fmt.Sprintf("%s/block/%s", baseURL, blockHash)
	body, err := httpclient.GetJSON(url)
	if err != nil {
		return nil, false, fmt.Errorf("fetching Bitcoin block header: %w", err)
	}
	var h blockstreamHeader
	if err := json.Unmarshal(body, &h); err != nil {
		return nil, false, fmt.Errorf("parsing Blockstream block response: %w", err)
	}
	if h.ID == "" {
		return nil, false, fmt.Errorf("blockstream response missing block id")
	}
	return &BitcoinBlockHeader{
		Hash:       h.ID,
		Height:     h.Height,
		Time:       h.Timestamp,
		MerkleRoot: h.MerkleRoot,
	}, false, nil
}
