// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chainhash/v2"
	"github.com/truestamp/truestamp-cli/internal/bitcoin"
	"github.com/truestamp/truestamp-cli/internal/external"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// --- [E.18 / E.19 / E.21] Step 11: the network-dependent steps ---
//
// The epoch proof (E.15) already proved the block hash sits under the root
// that the bundle SAYS was committed. What these steps add is confirmation
// that this root really appears in the named transaction on the named
// chain, and that each carried witness really exists in its source's
// records. Without them the cryptography holds but nothing ties it to a
// clock outside Truestamp. Every one of them reports `skip` when it cannot
// reach its source; a skipped check never fails a proof (E.22).

// stepExternal runs the commitment confirmations and the entropy re-fetches
// in the reference verifier's row order, returning the chains that
// confirmed and the entropy witnesses that confirmed, which E.20 consumes.
func stepExternal(r *Report, bundle *proof.Bundle, witnesses witnessSet, opts Options) (confirmedChains, confirmedWitnesses map[string]bool) {
	confirmedChains = map[string]bool{}
	confirmedWitnesses = map[string]bool{}

	for i := range bundle.Commitments {
		entry := &bundle.Commitments[i]
		switch entry.Chain {
		case ptype.ChainStellar:
			if entry.EpochMerkleRoot != "" && r.Temporal.StellarCommit == "" {
				r.Temporal.StellarCommit = commitmentTime(entry)
			}
			if opts.SkipExternal {
				r.skip(groupStellar, CatBlockchain, fmt.Sprintf(
					"not checked offline: confirm memo %s in tx %s on Horizon",
					short(entry.EpochMerkleRoot), short(entry.TransactionHash)))
				continue
			}
			if confirmStellar(r, entry, groupStellar, CatBlockchain) {
				confirmedChains[ptype.ChainStellar] = true
			}
		case ptype.ChainBitcoin:
			if entry.EpochMerkleRoot != "" && r.Temporal.BitcoinCommit == "" {
				r.Temporal.BitcoinCommit = commitmentTime(entry)
			}
			if confirmBitcoin(r, entry, groupBitcoin, CatBlockchain, opts) {
				confirmedChains[ptype.ChainBitcoin] = true
			}
		}
	}

	if bundle.IsEntropy() {
		if opts.SkipExternal {
			r.skip(groupEntropySource, CatBlockchain,
				"not checked offline: re-fetch the source and compare subject.entropy (a match proves the value exists, never that it was fresh)")
		} else {
			status, msg := fetchEntropySource(bundle.Type, bundle.Subject.Entropy, stellarNetwork(bundle))
			r.add(groupEntropySource, CatBlockchain, status, msg)
		}
	}

	for _, name := range witnesses.sortedNames() {
		e := witnesses[name]
		if !proof.IsEntropyWitnessName(name) || !e.Matched {
			continue
		}
		if opts.SkipExternal {
			r.skip(groupEntropySource, CatBlockchain, fmt.Sprintf(
				"%s witness source check skipped (offline): re-fetching it would confirm the after edge it opens", name))
			continue
		}
		status, msg := fetchEntropySource(name, e.Payload, stellarNetwork(bundle))
		r.add(groupEntropySource, CatBlockchain, status, fmt.Sprintf("%s witness: %s", name, msg))
		if status == StatusPass {
			confirmedWitnesses[name] = true
		}
	}
	return confirmedChains, confirmedWitnesses
}

// commitmentTime is the entry's timestamp, or "unknown" when it carries
// none, which is how the server records it.
func commitmentTime(entry *proof.Commitment) string {
	if entry.Timestamp == "" {
		return "unknown"
	}
	return entry.Timestamp
}

// stellarNetwork returns the network named by the bundle's first Stellar
// commitment. A Truestamp deployment uses one Stellar network for both
// entropy observation and commitment, and E.18's endpoint rule maps an
// absent name to the testnet instance.
func stellarNetwork(bundle *proof.Bundle) string {
	for _, c := range bundle.Commitments {
		if c.Chain == ptype.ChainStellar {
			return c.Network
		}
	}
	return ""
}

// --- [E.18] Stellar on-chain confirmation ---

// confirmStellar looks the transaction up on Horizon and grades the memo
// and ledger. It reports two rows on success, the way the server does, and
// returns whether the entry confirmed. Only a substantive disagreement with
// the chain may fail the group; every transport outcome is a skip.
func confirmStellar(r *Report, entry *proof.Commitment, group, category string) bool {
	if entry.TransactionHash == "" {
		r.skip(group, category, "Stellar commitment unconfirmed: the entry carries no transaction id to look up")
		return false
	}
	network := entry.Network
	networkLabel := external.NetworkLabel(network)

	result, err := external.VerifyStellar(entry.TransactionHash, entry.EpochMerkleRoot, network, 0)
	switch external.Classify(err) {
	case external.OutcomeOK:
		r.pass(group, category, "Stellar memo matches expected epoch root")
	case external.OutcomeMismatch:
		r.fail(group, category, fmt.Sprintf("Stellar commitment does not match the chain: %s", err))
		return false
	case external.OutcomeNotFound:
		// A 404 is only a definitive absence when the entry named the
		// network it was looking in. When `network` is absent or
		// unrecognised the lookup fell through to E.18's testnet default,
		// and a public-network transaction 404s there; failing on that
		// would let a missing optional field sink a sound proof.
		if external.IsDefaultedNetwork(network) {
			r.skip(group, category, fmt.Sprintf(
				"Stellar commitment unconfirmed: transaction %s is not on the default (testnet) Horizon, and %s",
				entry.TransactionHash, defaultedNetworkReason(network)))
		} else {
			r.fail(group, category, fmt.Sprintf("Transaction %s not found on %s", entry.TransactionHash, networkLabel))
		}
		return false
	default:
		r.skip(group, category, fmt.Sprintf("Stellar commitment unconfirmed: %s", err))
		return false
	}

	// The ledger, when carried, must match. A present value that is not an
	// integer is graded here rather than coerced.
	switch {
	case entry.Fields.Has("ledger") && !entry.HasLedger:
		r.fail(group, category, fmt.Sprintf(
			"Transaction %s confirmed on %s, but the entry's ledger %s is not an integer",
			entry.TransactionHash, networkLabel, entry.Fields.Literal("ledger")))
		return false
	case entry.HasLedger && result.Ledger != entry.Ledger:
		r.fail(group, category, fmt.Sprintf(
			"Transaction %s is in ledger %d on %s, not the ledger %d the entry names",
			entry.TransactionHash, result.Ledger, networkLabel, entry.Ledger))
		return false
	}
	r.pass(group, category, fmt.Sprintf("Transaction %s confirmed on %s (ledger %d)",
		entry.TransactionHash, networkLabel, result.Ledger))
	return true
}

// defaultedNetworkReason says WHY E.18's endpoint rule fell through to the
// testnet default.
func defaultedNetworkReason(network string) string {
	if network == "" {
		return "the entry names no network"
	}
	return fmt.Sprintf("the entry names an unrecognised network (%q)", network)
}

// --- [E.19] Bitcoin confirmation ---
//
// E.19(b) splits the work in two. The offline steps run over bytes the
// bundle itself supplies, and they establish the internal consistency of
// those bytes ONLY: a fabricated low-difficulty header over a
// one-transaction tree with any 32-byte OP_RETURN passes all of them. They
// are recorded as `info` on success and `fail` on a contradiction, in the
// server's own rows and words, because a commitment group's `pass` is
// reserved for a genuine on-chain confirmation. A mandatory binding step
// then confirms the recomputed block hash against something outside the
// bundle, and it is the only branch that may report the commitment as
// confirmed. The server has no networked Bitcoin lookup and always skips
// the binding; the Blockstream lookup here is the stricter verifier E.1
// and E.19 permit.
//
// E.19(c) makes every offline input optional: an absent txoutproof skips
// the Merkle rows, an absent raw_transaction skips the OP_RETURN and txid
// rows, an absent block_merkle_root skips the cross-check, and an absent
// transaction_hash or block_height leaves the binding step with nothing
// to look up. Absence is always a skip, never a failure.

func confirmBitcoin(r *Report, entry *proof.Commitment, group, category string, opts Options) bool {
	if entry.RawTransaction == "" && entry.Txoutproof == "" && entry.BlockMerkleRoot == "" {
		if opts.SkipExternal {
			r.skip(group, category, fmt.Sprintf(
				"not checked offline: confirm OP_RETURN %s in tx %s at height %s",
				short(entry.EpochMerkleRoot), short(entry.TransactionHash), entry.Fields.Literal("block_height")))
			return false
		}
		r.skip(group, category,
			"Bitcoin commitment unconfirmed: the entry carries no raw transaction, txoutproof or block merkle root to check")
		return false
	}

	checkBitcoinRawTx(r, entry, group, category)
	headerBlockHash := checkBitcoinTxOutProof(r, entry, group, category)
	return bindBitcoinHeader(r, entry, group, category, headerBlockHash, opts)
}

// consistency records an offline internal-consistency check the way the
// server does: info on success, and on a contradiction a fail carrying the
// same message behind "Check failed: ".
func consistency(r *Report, group, category string, ok bool, msg string) {
	if ok {
		r.info(group, category, msg)
	} else {
		r.fail(group, category, "Check failed: "+msg)
	}
}

// checkBitcoinRawTx runs E.19(b) steps 1 and 2: the OP_RETURN payload of
// the first nulldata output MUST equal epoch_merkle_root, and the txid
// recomputed per BIP 141 MUST equal transaction_hash.
func checkBitcoinRawTx(r *Report, entry *proof.Commitment, group, category string) {
	if entry.RawTransaction == "" {
		r.skip(group, category, "OP_RETURN and txid checks skipped (no raw transaction)")
		return
	}
	if extracted, err := bitcoin.ExtractOpReturn(entry.RawTransaction); err != nil {
		r.fail(group, category, fmt.Sprintf("OP_RETURN extraction failed: %s", err))
	} else {
		consistency(r, group, category, tscrypto.SecureEqual(extracted, entry.EpochMerkleRoot),
			"OP_RETURN in the supplied raw transaction matches the epoch root (internal consistency)")
	}
	computedTxid, err := bitcoin.ComputeTxID(entry.RawTransaction)
	switch {
	case err != nil:
		r.fail(group, category, fmt.Sprintf("Txid computation failed: %s", err))
	case entry.TransactionHash == "":
		r.skip(group, category, "Transaction id check skipped (no transaction id)")
	default:
		consistency(r, group, category, tscrypto.SecureEqual(computedTxid, entry.TransactionHash),
			fmt.Sprintf("Transaction id %s recomputed from the supplied raw transaction (internal consistency)", entry.TransactionHash))
	}
}

// checkBitcoinTxOutProof runs E.19(b) steps 3 to 5 and the
// block_merkle_root cross-check: parse the txoutproof, walk its partial
// Merkle tree and confirm transaction_hash is in the matched set (one row,
// as the server reports it), then compare the header's Merkle root with
// the entry's block_merkle_root. Returns the block hash recomputed from
// the 80-byte header (step 6) in display byte order, "" when no header
// could be recovered; it feeds the binding step and has no row of its own.
func checkBitcoinTxOutProof(r *Report, entry *proof.Commitment, group, category string) string {
	if entry.Txoutproof == "" {
		r.skip(group, category, "Bitcoin Merkle proof skipped (no txoutproof)")
		return ""
	}
	mb, err := bitcoin.DecodeTxOutProof(entry.Txoutproof)
	if err != nil {
		r.fail(group, category, fmt.Sprintf("Txoutproof parse failed: %s", err))
		return ""
	}
	merkleResult := bitcoin.VerifyPartialMerkleTree(mb.Hashes, mb.Flags, mb.Transactions, &mb.Header.MerkleRoot)

	switch {
	case entry.TransactionHash == "":
		r.skip(group, category, "Bitcoin Merkle proof placement skipped (no transaction id)")
	default:
		expectedTxid, err := chainhash.NewHashFromStr(entry.TransactionHash)
		if err != nil {
			r.fail(group, category, "Bitcoin merkle verification failed: malformed transaction id")
			break
		}
		inMatched := false
		for _, m := range merkleResult.MatchedTxIDs {
			if bitcoin.HashEqual(m, expectedTxid) {
				inMatched = true
				break
			}
		}
		consistency(r, group, category, merkleResult.Valid && inMatched,
			"Supplied txoutproof places the transaction under the supplied block Merkle root (internal consistency)")
	}

	// Both operands are display byte order (E.19(a)), so this is a plain
	// comparison and not a reversal.
	if entry.BlockMerkleRoot == "" {
		r.skip(group, category, "Bitcoin block_merkle_root not supplied; cannot cross-check against txoutproof header")
	} else {
		consistency(r, group, category, tscrypto.SecureEqual(entry.BlockMerkleRoot, mb.Header.MerkleRoot.String()),
			"Commitment block_merkle_root matches the supplied txoutproof header (internal consistency)")
	}
	return mb.Header.BlockHash().String()
}

// bindBitcoinHeader runs E.19(b)'s mandatory binding step: the block hash
// recomputed from the bundle's own header MUST be confirmed against
// something outside the bundle before the commitment may be reported as
// passing. When no confirmation point is available the commitment is
// reported skip, never pass.
func bindBitcoinHeader(r *Report, entry *proof.Commitment, group, category, headerBlockHash string, opts Options) bool {
	switch {
	case opts.SkipExternal:
		r.skip(group, category, "Bitcoin commitment unconfirmed: external confirmation skipped (offline)")
		return false
	case headerBlockHash == "":
		r.skip(group, category, "Bitcoin commitment unconfirmed: no block header was recovered to confirm against the chain")
		return false
	case !entry.HasBlockHeight:
		r.skip(group, category, "Bitcoin commitment unconfirmed: the entry carries no block_height to confirm the header at")
		return false
	}

	result, skipped, err := external.VerifyBitcoinBlock(headerBlockHash, entry.Network)
	switch {
	case skipped:
		r.skip(group, category, fmt.Sprintf("Bitcoin commitment unconfirmed: %s", external.BitcoinNetworkSkipReason(entry.Network)))
		r.info(group, category, bitcoinNetworkNote)
		return false
	case err != nil:
		if external.Classify(err) == external.OutcomeNotFound {
			r.fail(group, category, fmt.Sprintf("Bitcoin commitment not on chain: block %s not found on %s", headerBlockHash, entry.Network))
			return false
		}
		r.skip(group, category, fmt.Sprintf("Bitcoin commitment unconfirmed: %s", err))
		return false
	case result.Height != entry.BlockHeight:
		r.fail(group, category, fmt.Sprintf(
			"Bitcoin block height mismatch: the entry claims height %d, the chain reports %d", entry.BlockHeight, result.Height))
		return false
	default:
		r.pass(group, category, fmt.Sprintf("Block %s confirmed on %s (height %d)", headerBlockHash, entry.Network, result.Height))
		return true
	}
}

// bitcoinNetworkNote discloses that the field which suppressed the on-chain
// lookup is not covered by the signature payload: a rewritten `network`
// suppresses the lookup without disturbing the signature, so an unconfirmed
// commitment is not evidence about any chain.
const bitcoinNetworkNote = "`network` is not covered by the proof signature, so an unconfirmed commitment is not evidence about any chain: a rewritten `network` suppresses this lookup without disturbing the signature"

// --- [E.21] Entropy source re-fetch ---
//
// Re-fetch the original source and compare the reported value. A successful
// re-fetch establishes that the value exists in the source's records. It
// does NOT establish that the value was fresh at the moment of capture: a
// source that freezes yields a genuine, re-fetchable, stale value.

// fetchEntropySource re-fetches one payload from its source and grades it:
// pass on agreement, fail on a value mismatch, skip when the source cannot
// be reached or does not hold the record. sourceName is the subject type or
// witness name (`entropy_nist`, `entropy_stellar`, `entropy_bitcoin`).
func fetchEntropySource(sourceName string, payload json.RawMessage, stellarNet string) (Status, string) {
	switch sourceName {
	case proof.WitnessEntropyNIST:
		return fetchNIST(payload)
	case proof.WitnessEntropyStellar:
		return fetchStellarLedger(payload, stellarNet)
	case proof.WitnessEntropyBitcoin:
		// Bitcoin entropy is always captured from mainnet, even by
		// deployments that commit to testnet or regtest.
		return fetchBitcoinBlock(payload, "mainnet")
	}
	return StatusFail, fmt.Sprintf("unsupported entropy source %s", sourceName)
}

func fetchNIST(payload json.RawMessage) (Status, string) {
	var env struct {
		Pulse *struct {
			ChainIndex  int    `json:"chainIndex"`
			PulseIndex  int    `json:"pulseIndex"`
			OutputValue string `json:"outputValue"`
			TimeStamp   string `json:"timeStamp"`
		} `json:"pulse"`
		ChainIndex  int    `json:"chainIndex"`
		PulseIndex  int    `json:"pulseIndex"`
		OutputValue string `json:"outputValue"`
		TimeStamp   string `json:"timeStamp"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return StatusFail, fmt.Sprintf("cannot parse a NIST pulse from the payload: %s", err)
	}
	chainIdx, pulseIdx, outputValue, timeStamp := env.ChainIndex, env.PulseIndex, env.OutputValue, env.TimeStamp
	if env.Pulse != nil {
		chainIdx, pulseIdx, outputValue, timeStamp = env.Pulse.ChainIndex, env.Pulse.PulseIndex, env.Pulse.OutputValue, env.Pulse.TimeStamp
	}
	if outputValue == "" {
		return StatusFail, "NIST payload carries no outputValue"
	}
	remote, err := external.GetNISTPulse(chainIdx, pulseIdx)
	if err != nil {
		switch external.Classify(err) {
		case external.OutcomeBadInput:
			return StatusFail, fmt.Sprintf("NIST pulse index is unusable: %s", err)
		case external.OutcomeNotFound:
			return StatusSkip, fmt.Sprintf("source unconfirmed: NIST Beacon has no pulse %d on chain %d (HTTP 404)", pulseIdx, chainIdx)
		default:
			return StatusSkip, fmt.Sprintf("source unconfirmed: NIST Beacon unavailable (%s)", err)
		}
	}
	// The beacon publishes its outputValue in uppercase; the comparison
	// against an outside party's value folds case, the wire value is never
	// re-cased.
	if !tscrypto.HexEqual(remote.OutputValue, outputValue) {
		return StatusFail, fmt.Sprintf("NIST outputValue mismatch at pulse %d: upstream %s, proof %s", pulseIdx, remote.OutputValue, outputValue)
	}
	if !timestampsEqual(remote.TimeStamp, timeStamp) {
		return StatusFail, fmt.Sprintf("NIST timeStamp mismatch at pulse %d: upstream %s, proof %s", pulseIdx, remote.TimeStamp, timeStamp)
	}
	return StatusPass, "External consistency verified via NIST Beacon API"
}

func fetchStellarLedger(payload json.RawMessage, network string) (Status, string) {
	var ledger struct {
		Hash     string `json:"hash"`
		Sequence int    `json:"sequence"`
		ClosedAt string `json:"closed_at"`
		Ledger   *struct {
			Hash     string `json:"hash"`
			Sequence int    `json:"sequence"`
			ClosedAt string `json:"closed_at"`
		} `json:"ledger"`
	}
	if err := json.Unmarshal(payload, &ledger); err != nil {
		return StatusFail, fmt.Sprintf("cannot parse a Stellar ledger from the payload: %s", err)
	}
	hash, sequence, closedAt := ledger.Hash, ledger.Sequence, ledger.ClosedAt
	if ledger.Ledger != nil {
		hash, sequence, closedAt = ledger.Ledger.Hash, ledger.Ledger.Sequence, ledger.Ledger.ClosedAt
	}
	if sequence == 0 || hash == "" {
		return StatusFail, "Stellar payload carries no sequence or hash"
	}
	remote, err := external.GetStellarLedger(sequence, network)
	if err != nil {
		switch external.Classify(err) {
		case external.OutcomeBadInput:
			return StatusFail, fmt.Sprintf("Stellar ledger sequence is unusable: %s", err)
		case external.OutcomeNotFound:
			return StatusSkip, fmt.Sprintf("source unconfirmed: ledger %d is not in the %s Horizon history (HTTP 404)", sequence, external.NetworkLabel(network))
		default:
			return StatusSkip, fmt.Sprintf("source unconfirmed: Stellar Horizon unavailable (%s)", err)
		}
	}
	if !tscrypto.HexEqual(remote.Hash, hash) {
		return StatusFail, fmt.Sprintf("Stellar ledger hash mismatch at seq %d: upstream %s, proof %s", sequence, remote.Hash, hash)
	}
	if !timestampsEqual(remote.ClosedAt, closedAt) {
		return StatusFail, fmt.Sprintf("Stellar ledger closed_at mismatch at seq %d: upstream %s, proof %s", sequence, remote.ClosedAt, closedAt)
	}
	return StatusPass, "External consistency verified via Stellar Horizon API"
}

func fetchBitcoinBlock(payload json.RawMessage, network string) (Status, string) {
	var block struct {
		Hash   string `json:"hash"`
		Height int    `json:"height"`
		Time   int64  `json:"time"`
		Block  *struct {
			Hash   string `json:"hash"`
			Height int    `json:"height"`
			Time   int64  `json:"time"`
		} `json:"block"`
	}
	if err := json.Unmarshal(payload, &block); err != nil {
		return StatusFail, fmt.Sprintf("cannot parse a Bitcoin block from the payload: %s", err)
	}
	hash, height, btime := block.Hash, block.Height, block.Time
	if block.Block != nil {
		hash, height, btime = block.Block.Hash, block.Block.Height, block.Block.Time
	}
	if hash == "" {
		return StatusFail, "Bitcoin payload carries no hash"
	}
	remote, skipped, err := external.GetBitcoinBlockHeader(hash, network)
	if skipped {
		return StatusSkip, fmt.Sprintf("source unconfirmed: %s", external.BitcoinNetworkSkipReason(network))
	}
	if err != nil {
		switch external.Classify(err) {
		case external.OutcomeBadInput:
			return StatusFail, fmt.Sprintf("Bitcoin block hash is unusable: %s", err)
		case external.OutcomeNotFound:
			return StatusSkip, fmt.Sprintf("source unconfirmed: Blockstream has no block %s on %s (HTTP 404)", hash, network)
		default:
			return StatusSkip, fmt.Sprintf("source unconfirmed: Blockstream unavailable (%s)", err)
		}
	}
	if !tscrypto.HexEqual(remote.Hash, hash) {
		// The lookup was BY hash, so a disagreement is an answer about
		// some other block, not a defect in the bundle.
		return StatusSkip, fmt.Sprintf("source unconfirmed: Blockstream answered about block %s, not the block %s that was asked for", remote.Hash, hash)
	}
	if height != 0 && remote.Height != height {
		return StatusFail, fmt.Sprintf("Bitcoin height mismatch at hash %s: upstream %d, proof %d", hash, remote.Height, height)
	}
	if btime != 0 && remote.Time != btime {
		return StatusFail, fmt.Sprintf("Bitcoin block time mismatch at height %d: upstream %d, proof %d", remote.Height, remote.Time, btime)
	}
	return StatusPass, "External consistency verified via blockstream"
}

// timestampsEqual compares two ISO 8601 timestamps after parsing, so
// "2026-04-22T19:45:00Z" and "2026-04-22T19:45:00.000Z" are equal.
func timestampsEqual(a, b string) bool {
	if a == b {
		return true
	}
	ta, errA := time.Parse(time.RFC3339Nano, strings.TrimSpace(a))
	tb, errB := time.Parse(time.RFC3339Nano, strings.TrimSpace(b))
	if errA != nil || errB != nil {
		return false
	}
	return ta.Equal(tb)
}
