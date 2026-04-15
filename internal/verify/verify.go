// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/gowebpki/jcs"
	"github.com/truestamp/truestamp-cli/internal/bitcoin"
	"github.com/truestamp/truestamp-cli/internal/external"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// Options holds CLI flags for the verifier.
type Options struct {
	KeyringURL     string
	SkipExternal   bool
	SkipSignatures bool
	ExpectedHash   string // hex hash to compare against claims.hash
}

// Run executes the full verification pipeline on a proof file.
func Run(filename string, opts Options) (*Report, error) {
	bundle, err := proof.Parse(filename)
	if err != nil {
		return nil, fmt.Errorf("parsing proof: %w", err)
	}
	return runBundle(bundle, filename, proof.FileSize(filename), opts)
}

// RunFromBytes executes the full verification pipeline on raw proof bytes.
func RunFromBytes(data []byte, displayName string, opts Options) (*Report, error) {
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parsing proof: %w", err)
	}
	return runBundle(bundle, displayName, proof.FileSizeFromData(data), opts)
}

// runBundle runs the verification pipeline on a parsed proof bundle.
func runBundle(bundle *proof.ProofBundle, filename string, fileSize int64, opts Options) (*Report, error) {
	subject := bundle.Subject
	block := bundle.Block
	commits := bundle.Commitments
	isEntropy := bundle.IsEntropy()

	subjectType := "item"
	if isEntropy {
		subjectType = "entropy"
	}

	r := &Report{
		Filename:        filename,
		FileSize:        fileSize,
		ProofVersion:    bundle.Version,
		SubjectID:       subject.ID,
		SubjectType:     subjectType,
		GeneratedAt:     bundle.Timestamp,
		SkippedExternal: opts.SkipExternal,
		ChainLength:     1,
		SigningKeyID:    block.SigningKeyID,
	}

	if isEntropy {
		r.Source = subject.Source
		r.EntropySubject = parseEntropySubject(subject.Source, subject.Data)
	}

	// Parse claims for display (item proofs only)
	if !isEntropy {
		r.Claims = parseClaims(bundle.RawData)
	}

	// Hash comparison (item proofs only)
	if opts.ExpectedHash != "" && isEntropy {
		r.fail(groupHashComparison, CatDataIntegrity,
			fmt.Sprintf("--hash flag is not applicable to entropy proofs (source: %s)", subject.Source))
	}
	if !isEntropy {
		if opts.ExpectedHash != "" {
			r.HashProvided = opts.ExpectedHash
			if tscrypto.HexEqual(opts.ExpectedHash, r.Claims.Hash) {
				r.pass(groupHashComparison, CatDataIntegrity, "Provided hash matches claims.hash")
			} else {
				r.fail(groupHashComparison, CatDataIntegrity, fmt.Sprintf(
					"Provided hash does not match claims.hash (expected: %s, proof: %s)",
					opts.ExpectedHash, r.Claims.Hash))
			}
		} else if r.Claims.Hash != "" {
			hashType := r.Claims.HashType
			if hashType == "" {
				hashType = "sha256"
			}
			r.warn(groupHashComparison, CatDataIntegrity, fmt.Sprintf(
				"Claims hash not verified (use --hash <%s_hash_of_your_data> to confirm)", hashType))
		}
	}

	// Step 1: Signing Key
	pubkeyBytes, keyID := verifySigningKey(r, bundle, opts)

	// Step 2: Structure + Version
	verifyStructure(r, block)
	verifyVersion(r, bundle)

	// Steps 3-6: Subject hash derivation (branched by proof type)
	var subjectHash string
	if isEntropy {
		entropyHash := deriveEntropyHash(r, bundle.RawData)
		subjectHash = deriveObservationHash(r, subject, entropyHash)
	} else {
		claimsHash := deriveClaimsHash(r, bundle.RawData)
		validateClaimsHashType(r, r.Claims)
		verifyClaimsTimestamp(r, subject)
		subjectHash = deriveItemHash(r, subject, claimsHash)
	}

	// Step 7: Inclusion Proof (Merkle proof from subject to block)
	verifyInclusionProof(r, subjectHash, bundle.InclusionProof, block)

	// Step 8: Block Hash Derivation
	blockHash := deriveBlockHash(r, block)

	// Step 9: Key consistency (subject vs block)
	if !isEntropy {
		verifyItemKeyMatchesBlock(r, subject, block)
	}

	// Step 10: Epoch proofs (cx entries)
	epochRoots := verifyEpochProofs(r, commits, blockHash)

	// Step 11: Proof Signature (with timestamp)
	verifyProofSignature(r, bundle, pubkeyBytes, keyID, subjectHash, blockHash, epochRoots, opts)

	// Step 12: Temporal checks
	if isEntropy {
		verifyEntropyTemporalWindow(r, subject, block)
	} else {
		verifyItemTemporalWindow(r, subject, block)
	}

	// Step 13: Temporal Info
	if isEntropy {
		addEntropyTemporalInfo(r, subject, block)
	} else {
		addTemporalInfo(r, subject, block)
	}

	// Step 14: Stellar Commitment (external verification)
	verifyStellarCommitments(r, commits, blockHash, opts)

	// Step 15: Bitcoin Commitment (external verification)
	verifyBitcoinCommitments(r, commits, blockHash, opts)

	return r, nil
}

const (
	groupHashComparison = "Hash Comparison"
	groupSigningKey     = "Signing Key"
	groupStructure      = "Structure"
	groupInclusion      = "Inclusion Proof"
	groupBlock          = "Block"
	groupProof          = "Proof Signature"
	groupEpoch          = "Epoch Proofs"
	groupStellar        = "Stellar Commitment"
	groupBitcoin        = "Bitcoin Commitment"
)

// --- Step 1: Signing Key ---

func verifySigningKey(r *Report, bundle *proof.ProofBundle, opts Options) ([]byte, string) {
	pubkey, err := tscrypto.DecodePublicKey(bundle.PublicKey)
	if err != nil {
		r.fail(groupSigningKey, CatCryptographic, fmt.Sprintf("Failed to decode public key: %s", err))
		return nil, ""
	}

	keyID := tscrypto.ComputeKeyID(pubkey)
	r.pass(groupSigningKey, CatCryptographic, fmt.Sprintf("Public key valid, key_id: %s", keyID))

	if opts.SkipSignatures {
		r.skip(groupSigningKey, CatCryptographic, "Keyring check skipped (--skip-signatures)")
	} else if opts.SkipExternal {
		r.skip(groupSigningKey, CatCryptographic, "Keyring check skipped (--skip-external)")
	} else {
		signingKeys := map[string]string{keyID: bundle.PublicKey}
		if err := external.VerifyKeyring(signingKeys, opts.KeyringURL); err != nil {
			r.fail(groupSigningKey, CatCryptographic, fmt.Sprintf("Keyring verification failed: %s", err))
			r.info(groupSigningKey, CatCryptographic, "Proof signature was verified using the embedded public key")
			r.info(groupSigningKey, CatCryptographic, "Use --skip-external or --skip-signatures to skip this check")
		} else {
			r.pass(groupSigningKey, CatCryptographic, fmt.Sprintf("Key %s confirmed via keyring", keyID))
		}
	}

	return pubkey, keyID
}

// --- Step 2: Structure + Version ---

const expectedVersion = 1

func verifyStructure(r *Report, block proof.Block) {
	r.check(groupStructure, CatStructural, block.ID != "", "Block present with ID")
	r.check(groupStructure, CatStructural, block.MerkleRoot != "", "Block has merkle_root")
}

func verifyVersion(r *Report, bundle *proof.ProofBundle) {
	r.check(groupStructure, CatStructural, bundle.Version == expectedVersion,
		fmt.Sprintf("Proof version %d (expected %d)", bundle.Version, expectedVersion))
}

// --- Step 3: Claims Hash ---

func deriveClaimsHash(r *Report, rawClaims json.RawMessage) string {
	group := "Item"
	if len(rawClaims) == 0 || string(rawClaims) == "null" {
		r.skip(group, CatCryptographic, "Claims hash skipped (redacted)")
		return ""
	}

	canonical, err := jcs.Transform(rawClaims)
	if err != nil {
		r.fail(group, CatCryptographic, fmt.Sprintf("Claims JCS failed: %s", err))
		return ""
	}

	hash := tscrypto.BytesToHex(tscrypto.DomainHash(tscrypto.PrefixItemClaims, canonical))
	r.pass(group, CatCryptographic, "Claims hash derived (0x11)")
	return hash
}

// --- Step 4: Claims Hash Type ---

func validateClaimsHashType(r *Report, claims Claims) {
	if claims.Hash == "" || claims.HashType == "" {
		return
	}
	group := "Item"
	if err := tscrypto.ValidateClaimsHash(claims.Hash, claims.HashType); err != nil {
		r.warn(group, CatDataIntegrity, fmt.Sprintf("Claims hash validation: %s", err))
	} else {
		r.pass(group, CatDataIntegrity, fmt.Sprintf("Claims hash length valid for %s", claims.HashType))
	}
}

// --- Step 5: Claims Timestamp ---

func verifyClaimsTimestamp(r *Report, subject proof.Subject) {
	group := "Item"
	ts := extractClaimsTimestamp(subject.Data)
	if ts == "" {
		r.Claims.TimestampStatus = TimestampMissing
		return
	}

	r.Claims.TimestampStatus = TimestampOK
	r.Temporal.ClaimedAt = truncateToSecond(ts)

	claimedTime, cErr := time.Parse(time.RFC3339, ts)
	submittedTime, sErr := tscrypto.ExtractULIDTimestamp(subject.ID)
	if cErr != nil || sErr != nil {
		return
	}

	if !claimedTime.Before(submittedTime) {
		r.Claims.TimestampStatus = TimestampFuture
		r.Claims.TimestampNote = "Claims timestamp is not before submission time (future-dated claim)"
		r.warn(group, CatTiming, "Claims timestamp is not before submission time (future-dated claim)")
	} else if submittedTime.Sub(claimedTime) > 7*24*time.Hour {
		days := int(submittedTime.Sub(claimedTime).Hours() / 24)
		r.Claims.TimestampStatus = TimestampStale
		r.Claims.TimestampNote = fmt.Sprintf("Claims timestamp is %d days before submission (stale claim)", days)
		r.warn(group, CatTiming, fmt.Sprintf("Claims timestamp is %d days before submission (stale claim)", days))
	}
}

// --- Step 6: Item Hash ---

func deriveItemHash(r *Report, subject proof.Subject, claimsHash string) string {
	group := "Item"
	if claimsHash == "" || subject.MetadataHash == "" || subject.SigningKeyID == "" {
		r.fail(group, CatCryptographic, "Cannot derive item hash (missing inputs)")
		return ""
	}

	hash, err := tscrypto.ComputeItemHash(subject.ID, claimsHash, subject.MetadataHash, subject.SigningKeyID)
	if err != nil {
		r.fail(group, CatCryptographic, fmt.Sprintf("Item hash computation failed: %s", err))
		return ""
	}

	r.pass(group, CatCryptographic, "Item hash derived (0x13)")
	return hash
}

// --- Step 7: Inclusion Proof ---

func verifyInclusionProof(r *Report, subjectHash, inclusionProof string, block proof.Block) {
	if subjectHash == "" {
		r.fail(groupInclusion, CatCryptographic, "Cannot verify inclusion proof (no subject hash)")
		return
	}

	proofList, err := tscrypto.DecodeCompactMerkleProof(inclusionProof)
	if err != nil {
		r.fail(groupInclusion, CatCryptographic, fmt.Sprintf("Inclusion proof decode failed: %s", err))
		return
	}

	valid, err := tscrypto.VerifyMerkleProof(subjectHash, proofList, block.MerkleRoot)
	if err != nil {
		r.fail(groupInclusion, CatCryptographic, fmt.Sprintf("Inclusion proof error: %s", err))
		return
	}

	r.check(groupInclusion, CatCryptographic, valid, fmt.Sprintf("Inclusion proof to block %s (%d steps)", block.ID, len(proofList)))
}

// --- Step 8: Block Hash Derivation ---

func deriveBlockHash(r *Report, block proof.Block) string {
	computed, err := tscrypto.ComputeBlockHash(block.ID, block.PreviousBlockHash, block.MerkleRoot, block.MetadataHash, block.SigningKeyID)
	if err != nil {
		r.fail(groupBlock, CatCryptographic, fmt.Sprintf("Block hash computation failed: %s", err))
		return ""
	}

	r.pass(groupBlock, CatCryptographic, "Block hash derived (0x32)")
	return computed
}

// --- Step 9: Item Key Matches Block ---

func verifyItemKeyMatchesBlock(r *Report, subject proof.Subject, block proof.Block) {
	group := "Item"
	if tscrypto.HexEqual(subject.SigningKeyID, block.SigningKeyID) {
		r.pass(group, CatDataIntegrity, "Item signing key matches block signing key")
	} else {
		r.info(group, CatDataIntegrity, "Item signing key differs from block (key rotation)")
	}
}

// --- Step 10: Epoch Proofs ---

func verifyEpochProofs(r *Report, commits []ExternalCommit, blockHash string) []string {
	if len(commits) == 0 {
		return nil
	}
	if blockHash == "" {
		r.fail(groupEpoch, CatCryptographic, "Cannot verify epoch proofs (no block hash)")
		return nil
	}

	var epochRoots []string
	for i, cx := range commits {
		proofList, err := tscrypto.DecodeCompactMerkleProof(cx.EpochProof)
		if err != nil {
			r.fail(groupEpoch, CatCryptographic, fmt.Sprintf("Epoch proof %d decode failed: %s", i, err))
			epochRoots = append(epochRoots, "")
			continue
		}

		valid, err := tscrypto.VerifyMerkleProof(blockHash, proofList, epochTarget(cx))
		if err != nil {
			r.fail(groupEpoch, CatCryptographic, fmt.Sprintf("Epoch proof %d error: %s", i, err))
			epochRoots = append(epochRoots, "")
			continue
		}

		root := epochTarget(cx)
		r.check(groupEpoch, CatCryptographic, valid,
			fmt.Sprintf("Epoch proof %d (%s): block hash maps to committed value (%d steps)", i, cx.Type, len(proofList)))
		epochRoots = append(epochRoots, root)
	}

	return epochRoots
}

// epochTarget returns the committed value for an external commitment:
// memo for stellar, op_return for bitcoin.
func epochTarget(cx ExternalCommit) string {
	switch cx.Type {
	case "stellar":
		return cx.MemoHash
	case "bitcoin":
		return cx.OpReturn
	default:
		return ""
	}
}

// ExternalCommit is a type alias for convenience
type ExternalCommit = proof.ExternalCommit

// --- Step 11: Proof Signature ---

func verifyProofSignature(r *Report, bundle *proof.ProofBundle, pubkeyBytes []byte, keyID, subjectHash, blockHash string, epochRoots []string, opts Options) {
	if pubkeyBytes == nil || subjectHash == "" {
		r.fail(groupProof, CatCryptographic, "Cannot verify proof signature (missing derived data)")
		return
	}

	if blockHash == "" {
		r.fail(groupProof, CatCryptographic, "Cannot verify proof signature (missing block hash)")
		return
	}

	for i, er := range epochRoots {
		if er == "" {
			r.fail(groupProof, CatCryptographic, fmt.Sprintf("Cannot verify proof signature (missing epoch root %d)", i))
			return
		}
	}

	// Parse timestamp from proof ts field
	tsTime, err := time.Parse(time.RFC3339, bundle.Timestamp)
	if err != nil {
		r.fail(groupProof, CatCryptographic, fmt.Sprintf("Cannot parse proof timestamp: %s", err))
		return
	}
	timestampMs := uint64(tsTime.UnixMilli())

	proofHashBytes, err := tscrypto.BuildCompactProofPayload(
		byte(bundle.Version), keyID, timestampMs, subjectHash, blockHash, epochRoots,
	)
	if err != nil {
		r.fail(groupProof, CatCryptographic, fmt.Sprintf("Proof hash computation failed: %s", err))
		return
	}

	if opts.SkipSignatures {
		r.skip(groupProof, CatCryptographic, "Proof signature verification skipped (--skip-signatures)")
		return
	}

	sigValid, err := tscrypto.VerifyEd25519(proofHashBytes, bundle.Signature, pubkeyBytes)
	if err != nil {
		r.fail(groupProof, CatCryptographic, fmt.Sprintf("Proof signature verification error: %s", err))
		return
	}

	r.check(groupProof, CatCryptographic, sigValid, "Proof signature valid (Ed25519)")
}

// --- Step 12: Temporal Checks ---

func verifyItemTemporalWindow(r *Report, subject proof.Subject, block proof.Block) {
	group := "Item"
	itemTime, err := tscrypto.ExtractULIDTimestamp(subject.ID)
	if err != nil {
		return
	}

	blockTime, bErr := tscrypto.ExtractUUIDv7Timestamp(block.ID)
	if bErr != nil {
		return
	}

	ok := !itemTime.After(blockTime)
	r.check(group, CatTiming, ok, fmt.Sprintf("Item submitted before committed block (%s)", blockTime.Format(time.RFC3339)))
}

func verifyEntropyTemporalWindow(r *Report, subject proof.Subject, block proof.Block) {
	group := "Entropy"
	obsTime, err := tscrypto.ExtractUUIDv7Timestamp(subject.ID)
	if err != nil {
		return
	}

	blockTime, bErr := tscrypto.ExtractUUIDv7Timestamp(block.ID)
	if bErr != nil {
		return
	}

	ok := !obsTime.After(blockTime)
	r.check(group, CatTiming, ok, fmt.Sprintf("Entropy captured before committed block (%s)", blockTime.Format(time.RFC3339)))
}

// --- Step 13: Temporal Info ---

func addTemporalInfo(r *Report, subject proof.Subject, block proof.Block) {
	r.Temporal.SubmittedAt = tscrypto.FormatItemTime(subject.ID)
	r.Temporal.CommittedAt = tscrypto.FormatBlockTime(block.ID)
}

func addEntropyTemporalInfo(r *Report, subject proof.Subject, block proof.Block) {
	r.Temporal.CapturedAt = tscrypto.FormatBlockTime(subject.ID)
	r.Temporal.CommittedAt = tscrypto.FormatBlockTime(block.ID)
}

// --- Step 14: Stellar Commitments ---

func verifyStellarCommitments(r *Report, commits []proof.ExternalCommit, blockHash string, opts Options) {
	for i := range commits {
		if commits[i].Type != "stellar" {
			continue
		}
		verifySingleStellar(r, &commits[i], blockHash, opts)
	}
}

func verifySingleStellar(r *Report, cx *proof.ExternalCommit, blockHash string, opts Options) {
	// Build commitment info for display
	ci := CommitmentInfo{
		Method:        "stellar",
		Network:       cx.Network,
		Ledger:        cx.Ledger,
		TxHash:        cx.TransactionHash,
		CommittedHash: cx.MemoHash,
		Skipped:       opts.SkipExternal,
		Timestamp:     cx.Timestamp,
	}

	if opts.SkipExternal {
		r.skip(groupStellar, CatBlockchain, "External Stellar verification skipped (--skip-external)")
	} else {
		result, err := external.VerifyStellar(cx.TransactionHash, cx.MemoHash, cx.Network, cx.Ledger)
		if err != nil {
			r.fail(groupStellar, CatBlockchain, fmt.Sprintf("Stellar external verification failed: %s", err))
		} else {
			r.pass(groupStellar, CatBlockchain, fmt.Sprintf("Transaction %s confirmed on %s (ledger %d)", cx.TransactionHash, cx.Network, result.Ledger))
			// Use externally confirmed timestamp as the gold standard
			if result.Timestamp != "" {
				ci.Timestamp = result.Timestamp
			}
		}
	}

	r.CommitmentInfos = append(r.CommitmentInfos, ci)
}

// --- Step 15: Bitcoin Commitments ---

func verifyBitcoinCommitments(r *Report, commits []proof.ExternalCommit, blockHash string, opts Options) {
	for i := range commits {
		if commits[i].Type != "bitcoin" {
			continue
		}
		verifySingleBitcoin(r, &commits[i], blockHash, opts)
	}
}

func verifySingleBitcoin(r *Report, cx *proof.ExternalCommit, blockHash string, opts Options) {
	// OP_RETURN extraction from raw transaction
	extractedOpReturn, err := bitcoin.ExtractOpReturn(cx.RawTxHex)
	if err != nil {
		r.fail(groupBitcoin, CatBlockchain, fmt.Sprintf("OP_RETURN extraction failed: %s", err))
	} else {
		r.check(groupBitcoin, CatBlockchain,
			tscrypto.HexEqual(extractedOpReturn, cx.OpReturn),
			"OP_RETURN extracted from raw transaction matches")
	}

	// Txid computation (handles segwit non-witness serialization)
	computedTxid, err := bitcoin.ComputeTxID(cx.RawTxHex)
	if err != nil {
		r.fail(groupBitcoin, CatBlockchain, fmt.Sprintf("Txid computation failed: %s", err))
	} else {
		r.check(groupBitcoin, CatBlockchain,
			tscrypto.HexEqual(computedTxid, cx.TransactionHash),
			fmt.Sprintf("Transaction %s verified from raw tx", cx.TransactionHash))
	}

	// Parse txoutproof and verify partial merkle tree (BIP 37)
	mb, err := bitcoin.DecodeTxOutProof(cx.TxoutproofHex)
	if err != nil {
		r.fail(groupBitcoin, CatBlockchain, fmt.Sprintf("Txoutproof parse failed: %s", err))
		return
	}

	expectedTxid, _ := chainhash.NewHashFromStr(cx.TransactionHash)
	merkleResult := bitcoin.VerifyPartialMerkleTree(
		mb.Hashes, mb.Flags, mb.Transactions, &mb.Header.MerkleRoot,
	)

	txidInMatched := false
	for _, m := range merkleResult.MatchedTxIDs {
		if m.IsEqual(expectedTxid) {
			txidInMatched = true
			break
		}
	}

	r.check(groupBitcoin, CatBlockchain, merkleResult.Valid && txidInMatched, "Bitcoin merkle proof valid")

	// Build commitment info for display
	timestamp := cx.Timestamp
	ci := CommitmentInfo{
		Method:        "bitcoin",
		Network:       cx.Network,
		Height:        cx.BlockHeight,
		TxHash:        cx.TransactionHash,
		CommittedHash: cx.OpReturn,
		BlockHash:     mb.Header.BlockHash().String(),
		Timestamp:     timestamp,
		Skipped:       opts.SkipExternal,
	}

	if opts.SkipExternal {
		r.skip(groupBitcoin, CatBlockchain, "External Bitcoin verification skipped (--skip-external)")
	} else {
		result, skipped, err := external.VerifyBitcoinBlock(mb.Header.BlockHash().String(), cx.Network)
		if skipped {
			r.skip(groupBitcoin, CatBlockchain, fmt.Sprintf("Bitcoin external verification skipped: no public API for %s", cx.Network))
		} else if err != nil {
			r.fail(groupBitcoin, CatBlockchain, fmt.Sprintf("Bitcoin external verification failed: %s", err))
		} else {
			r.pass(groupBitcoin, CatBlockchain, fmt.Sprintf("Block confirmed on %s (height %d)", cx.Network, result.Height))
			// Use externally confirmed timestamp as the gold standard
			if result.Timestamp != "" {
				ci.Timestamp = result.Timestamp
			}
		}
	}

	r.CommitmentInfos = append(r.CommitmentInfos, ci)
}

// --- Entropy Hash ---

func deriveEntropyHash(r *Report, rawData json.RawMessage) string {
	group := "Entropy"
	if len(rawData) == 0 || string(rawData) == "null" {
		r.fail(group, CatCryptographic, "Entropy data missing")
		return ""
	}

	canonical, err := jcs.Transform(rawData)
	if err != nil {
		r.fail(group, CatCryptographic, fmt.Sprintf("Entropy JCS failed: %s", err))
		return ""
	}

	hash := tscrypto.ComputeEntropyHash(canonical)
	r.pass(group, CatCryptographic, "Entropy hash derived (0x21)")
	return hash
}

// --- Observation Hash ---

func deriveObservationHash(r *Report, subject proof.Subject, entropyHash string) string {
	group := "Entropy"
	if entropyHash == "" || subject.MetadataHash == "" || subject.SigningKeyID == "" {
		r.fail(group, CatCryptographic, "Cannot derive observation hash (missing inputs)")
		return ""
	}

	hash, err := tscrypto.ComputeObservationHash(subject.ID, entropyHash, subject.MetadataHash, subject.SigningKeyID)
	if err != nil {
		r.fail(group, CatCryptographic, fmt.Sprintf("Observation hash computation failed: %s", err))
		return ""
	}

	r.pass(group, CatCryptographic, "Observation hash derived (0x23)")
	return hash
}

// --- Entropy Subject Parsing ---

func parseEntropySubject(source string, rawEntropy json.RawMessage) EntropySubject {
	subject := EntropySubject{RawSource: source, Source: humanizeSource(source)}

	if len(rawEntropy) == 0 {
		return subject
	}

	switch source {
	case "nist_beacon":
		var nist struct {
			Pulse struct {
				TimeStamp   string `json:"timeStamp"`
				PulseIndex  int    `json:"pulseIndex"`
				ChainIndex  int    `json:"chainIndex"`
				Version     string `json:"version"`
				OutputValue string `json:"outputValue"`
			} `json:"pulse"`
		}
		if json.Unmarshal(rawEntropy, &nist) == nil {
			subject.CapturedAt = nist.Pulse.TimeStamp
			subject.PulseIndex = nist.Pulse.PulseIndex
			subject.ChainIndex = nist.Pulse.ChainIndex
			subject.Version = nist.Pulse.Version
			subject.OutputValue = nist.Pulse.OutputValue
		}

	case "bitcoin_block":
		var btc struct {
			Hash   string `json:"hash"`
			Height int    `json:"height"`
			Time   int64  `json:"time"`
		}
		if json.Unmarshal(rawEntropy, &btc) == nil {
			subject.BlockHash = btc.Hash
			subject.BlockHeight = btc.Height
			subject.BlockTime = btc.Time
			if btc.Time > 0 {
				subject.CapturedAt = time.Unix(btc.Time, 0).UTC().Format(time.RFC3339)
			}
		}

	case "stellar_ledger":
		var stellar struct {
			Hash     string `json:"hash"`
			Sequence int    `json:"sequence"`
			ClosedAt string `json:"closed_at"`
		}
		if json.Unmarshal(rawEntropy, &stellar) == nil {
			subject.LedgerHash = stellar.Hash
			subject.LedgerSequence = stellar.Sequence
			subject.LedgerClosedAt = stellar.ClosedAt
			subject.CapturedAt = stellar.ClosedAt
		}

	default:
		subject.Source = "Unknown entropy source: " + source
	}

	return subject
}

// humanizeSource converts an entropy source identifier to a human-readable name.
func humanizeSource(source string) string {
	switch source {
	case "nist_beacon":
		return "NIST Beacon"
	case "bitcoin_block":
		return "Bitcoin Block"
	case "stellar_ledger":
		return "Stellar Ledger"
	default:
		return source
	}
}

// --- Helpers ---

func extractClaimsTimestamp(claims json.RawMessage) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(claims, &m); err != nil {
		return ""
	}
	raw, ok := m["timestamp"]
	if !ok {
		return ""
	}
	var ts string
	if err := json.Unmarshal(raw, &ts); err != nil {
		return ""
	}
	return ts
}

func parseClaims(raw json.RawMessage) Claims {
	var c Claims
	if len(raw) == 0 || string(raw) == "null" {
		return c
	}
	json.Unmarshal(raw, &c)

	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) == nil {
		if v, ok := m["metadata"]; ok && string(v) != "null" && string(v) != "{}" {
			c.HasMetadata = true
			c.RawMetadata = v
		}
	}
	return c
}
