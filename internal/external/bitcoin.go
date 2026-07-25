// Copyright (c) 2019-2026 Truestamp, Inc.
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
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// blockstreamBlock is the subset of the Blockstream block resource the
// E.19(b) binding step reads.
//
// `id` is not decoration: it is the only field that ties the answer to
// the question. Height is a pointer so an absent `height` is
// distinguishable from a real height of 0 — without that, a body that
// says nothing about height decodes to 0 and the caller reports "the
// chain reports 0" as a positive statement about Bitcoin.
type blockstreamBlock struct {
	ID        string `json:"id"`
	Height    *int   `json:"height"`
	Timestamp int64  `json:"timestamp"`
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

// Bitcoin network names carried in a commitment's `net` field.
const (
	bitcoinMainnet = "mainnet"
	bitcoinTestnet = "testnet"
	bitcoinRegtest = "regtest"
)

// blockstreamBaseURL resolves the `net` field to a public Blockstream
// instance, reporting whether one exists at all.
func blockstreamBaseURL(network string) (string, bool) {
	switch network {
	case bitcoinMainnet:
		return BlockstreamMainnetURL, true
	case bitcoinTestnet:
		return BlockstreamTestnetURL, true
	default:
		return "", false
	}
}

// BitcoinNetworkSkipReason explains why a `net` value yields no
// networked Bitcoin lookup, so a report can state the reason it actually
// has instead of asserting that a name it does not recognise is a
// Bitcoin network that happens to lack a public API. It returns "" for a
// network that IS looked up.
//
// The grading is settled by E.5: `net` is optional, and "E.19 names no
// default endpoint, and a verifier MUST NOT guess a Bitcoin network from
// an absent `net`: if it cannot determine which chain to query it has no
// networked lookup available, which is already one of E.19's stated
// `skip` conditions for the binding step." Absent, regtest and
// unrecognised all land there.
//
// Known consequence: `net` is not in the 0x61 signature payload, so this
// is an unauthenticated downgrade lever — a bundle whose header is
// genuinely absent from mainnet can have `net` rewritten and the
// definitive on-chain refutation becomes a skip. It cannot be closed by
// grading: regtest is a legitimate Truestamp network with no public API,
// and E.5 forbids guessing a chain. What a verifier can do is refuse to
// dress the skip up as a statement about a chain it never consulted.
func BitcoinNetworkSkipReason(network string) string {
	switch network {
	case bitcoinMainnet, bitcoinTestnet:
		return ""
	case bitcoinRegtest:
		return "no public API for regtest"
	case "":
		return "the entry names no Bitcoin network to look the header up on"
	default:
		return fmt.Sprintf("the entry names an unrecognised Bitcoin network (%q)", network)
	}
}

// VerifyBitcoinBlock runs E.19(b)'s networked binding lookup: it asks a
// public Blockstream instance for the block named by blockHash and
// returns that block's height and timestamp. Returns a skip indicator
// when `net` resolves to no public endpoint (see
// [BitcoinNetworkSkipReason]). Errors are typed for [Classify].
func VerifyBitcoinBlock(blockHash, network string) (*BitcoinResult, bool, error) {
	baseURL, ok := blockstreamBaseURL(network)
	if !ok {
		return nil, true, nil // skipped
	}

	if _, err := hex.DecodeString(blockHash); err != nil || len(blockHash) != 64 {
		return nil, false, &BadInputError{Field: "block hash", Detail: "not a 64-character hex string"}
	}

	url := fmt.Sprintf("%s/block/%s", baseURL, blockHash)
	body, err := httpclient.GetJSON(url)
	if err != nil {
		return nil, false, fmt.Errorf("fetching Bitcoin block: %w", err)
	}

	var block blockstreamBlock
	if err := json.Unmarshal(body, &block); err != nil {
		return nil, false, &MalformedResponseError{Source: "Blockstream", Detail: "block response is not valid JSON", Err: err}
	}

	// E.19(b): this step exists to confirm the recomputed block hash
	// "against something outside the bundle", so the answer MUST be
	// about the block that was asked for. Without this guard a 200 body
	// describing any other block is reported as `pass ... confirmed on
	// mainnet` with externally_verified true — a binding the lookup
	// never established.
	//
	// A non-matching id is graded Malformed, not Mismatch: the bundle
	// contributed only the lookup key, which was sent verbatim, so an
	// answer about a different block is an upstream property and cannot
	// be evidence against the bundle. E.22 skips such an answer.
	if block.ID == "" {
		return nil, false, &MalformedResponseError{Source: "Blockstream", Detail: "block response carries no block id"}
	}
	if !tscrypto.HexEqual(block.ID, blockHash) {
		return nil, false, &MalformedResponseError{
			Source: "Blockstream",
			Detail: fmt.Sprintf("block response describes block %s, not the requested %s", block.ID, blockHash),
		}
	}

	// E.19(b) binds the header at height `h`, and the caller compares
	// this height against the entry's. A body carrying no height said
	// nothing about it; reporting the zero value would fail a sound
	// proof with "the chain reports 0".
	if block.Height == nil {
		return nil, false, &MalformedResponseError{Source: "Blockstream", Detail: "block response carries no height"}
	}

	var ts string
	if block.Timestamp > 0 {
		ts = time.Unix(block.Timestamp, 0).UTC().Format(time.RFC3339)
	}

	return &BitcoinResult{
		Height:    *block.Height,
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
// Errors are typed for [Classify].
func GetBitcoinBlockHeader(blockHash, network string) (*BitcoinBlockHeader, bool, error) {
	baseURL, ok := blockstreamBaseURL(network)
	if !ok {
		return nil, true, nil
	}
	if _, err := hex.DecodeString(blockHash); err != nil || len(blockHash) != 64 {
		return nil, false, &BadInputError{Field: "block hash", Detail: "not a 64-character hex string"}
	}
	url := fmt.Sprintf("%s/block/%s", baseURL, blockHash)
	body, err := httpclient.GetJSON(url)
	if err != nil {
		return nil, false, fmt.Errorf("fetching Bitcoin block header: %w", err)
	}
	var h blockstreamHeader
	if err := json.Unmarshal(body, &h); err != nil {
		return nil, false, &MalformedResponseError{Source: "Blockstream", Detail: "block response is not valid JSON", Err: err}
	}
	if h.ID == "" {
		return nil, false, &MalformedResponseError{Source: "Blockstream", Detail: "block response carries no block id"}
	}
	return &BitcoinBlockHeader{
		Hash:       h.ID,
		Height:     h.Height,
		Time:       h.Timestamp,
		MerkleRoot: h.MerkleRoot,
	}, false, nil
}
