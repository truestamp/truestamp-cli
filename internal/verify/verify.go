// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package verify runs the end-to-end cryptographic verification pipeline
// against a parsed proof bundle: signing-key lookup against the public
// keyring, Ed25519 signature check, Merkle-inclusion proof, block hash
// chain, and optional public-blockchain commitments (Stellar, Bitcoin).
// Callers receive a [Report] summarizing every check.
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
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// Options holds CLI flags for the verifier.
type Options struct {
	KeyringURL     string
	APIURL         string // optional; populates Report.APIURL so the presenter can emit subject-detail + verify web links
	SkipExternal   bool
	SkipSignatures bool
	ExpectedHash   string // hex hash to compare against claims.hash

	// ExpectedSubjectType, when non-empty, asserts that the parsed
	// bundle's subject type name (ptype.Name) matches. A mismatch
	// surfaces as a StatusFail step in the "Subject Type" group —
	// crypto steps still run, the report still renders, and the
	// mismatch appears in the Issues section. Mirrors --type in the
	// download command so users can guard against verifying the wrong
	// file. Must be one of: item, entropy_nist, entropy_stellar,
	// entropy_bitcoin, block, beacon.
	ExpectedSubjectType string
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
	block := bundle.Block
	commits := bundle.Commitments
	// Block and beacon (t ∈ {10, 11}) share the same wire shape — no subject,
	// no inclusion proof, subject_hash == block_hash. Pipeline guards switch
	// on "block-like" rather than strict IsBlock so beacon proofs take the
	// same code path.
	isBlockLike := bundle.IsBlockLike()
	isItem := bundle.IsItem()
	isEntropy := bundle.IsEntropy()

	subjectType := ptype.Name(bundle.T)

	// Subject-type assertion (client-side; mirrors the server's
	// /proof/verify type check added alongside the t=11 beacon work).
	// Surface as a StatusFail step so the report still renders and the
	// crypto pipeline still runs — same UX precedent as --hash
	// mismatches. Exit code is non-zero via Report.Passed().
	subjectTypeAssertionFailed := false
	if opts.ExpectedSubjectType != "" && opts.ExpectedSubjectType != subjectType {
		subjectTypeAssertionFailed = true
	}

	r := &Report{
		Filename:        filename,
		FileSize:        fileSize,
		ProofVersion:    bundle.Version,
		SubjectType:     subjectType,
		APIURL:          opts.APIURL,
		GeneratedAt:     bundle.Timestamp,
		SkippedExternal: opts.SkipExternal,
		ChainLength:     1,
		SigningKeyID:    block.SigningKeyID,
	}

	if isBlockLike {
		r.SubjectID = block.ID
	} else if bundle.Subject != nil {
		r.SubjectID = bundle.Subject.ID
	}

	if isEntropy && bundle.Subject != nil {
		r.Source = subjectType
		r.EntropySubject = parseEntropySubject(bundle.T, bundle.Subject.Data)
	}

	if isItem {
		r.Claims = parseClaims(bundle.RawData)
	}

	// Subject-type assertion (from --type flag). Check early so it
	// surfaces at the top of the Issues block alongside structural
	// failures; crypto still runs so users see whether the proof is
	// otherwise valid.
	if subjectTypeAssertionFailed {
		r.fail(groupSubjectType, CatStructural, fmt.Sprintf(
			"Proof is %s (t=%d) but --type %s was requested",
			subjectType, bundle.T, opts.ExpectedSubjectType))
	}

	// Hash comparison (item proofs only)
	if opts.ExpectedHash != "" && !isItem {
		r.fail(groupHashComparison, CatDataIntegrity,
			fmt.Sprintf("--hash flag is not applicable to %s proofs", subjectType))
	}
	if isItem {
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
	verifyStructure(r, bundle, block)
	verifyVersion(r, bundle)

	// Steps 3-6: Subject-hash derivation (non-block-like subjects only)
	var subjectHash string
	switch {
	case isItem:
		claimsHash := deriveClaimsHash(r, bundle.RawData)
		validateClaimsHashType(r, r.Claims)
		verifyClaimsTimestamp(r, bundle.Subject)
		subjectHash = deriveItemHash(r, bundle.Subject, claimsHash)
	case isEntropy:
		entropyHash := deriveEntropyHash(r, bundle.RawData)
		subjectHash = deriveObservationHash(r, bundle.Subject, entropyHash)
	}

	// Step 7: Inclusion Proof (skipped for block-like subjects)
	if !isBlockLike {
		verifyInclusionProof(r, subjectHash, bundle.InclusionProof, block)
	}

	// Step 8: Block Hash Derivation
	blockHash := deriveBlockHash(r, block)

	// For block-like subjects, subject_hash == block_hash by construction.
	if isBlockLike {
		subjectHash = blockHash
	}

	// Step 9 (dropped): subject.kid == block.kid equality check. Legitimate
	// key rotation can produce divergent kids; subject-kid tampering is still
	// caught because s.kid is an input to the 0x13 / 0x23 composite hash.

	// Step 10: Epoch proofs (cx entries)
	epochRoots := verifyEpochProofs(r, commits, blockHash)

	// Step 11: Proof Signature
	verifyProofSignature(r, bundle, pubkeyBytes, keyID, subjectHash, blockHash, epochRoots, opts)

	// Step 12: Temporal checks + info
	if !isBlockLike {
		verifySubjectTemporalWindow(r, bundle.T, bundle.Subject, block)
	}
	addTemporalInfo(r, bundle.T, bundle.Subject, block)

	// Step 13: Entropy source consistency (entropy subjects only, external, skippable)
	if isEntropy {
		verifyEntropySource(r, bundle.T, bundle.Subject, commits, opts)
	}

	// Step 14: Stellar Commitment
	verifyStellarCommitments(r, commits, blockHash, opts)

	// Step 15: Bitcoin Commitment
	verifyBitcoinCommitments(r, commits, blockHash, opts)

	return r, nil
}

const (
	groupHashComparison = "Hash Comparison"
	groupSigningKey     = "Signing Key"
	groupStructure      = "Structure"
	groupSubjectType    = "Subject Type"
	groupSubjectData    = "Subject Data"
	groupInclusion      = "Inclusion Proof"
	groupBlockHash      = "Block Hash"
	groupProof          = "Proof Signature"
	groupEpoch          = "Epoch Proof"
	groupStellar        = "Stellar Commitment"
	groupBitcoin        = "Bitcoin Commitment"
	groupTemporal       = "Temporal Window"
	groupEntropySource  = "Entropy Source"
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

func verifyStructure(r *Report, bundle *proof.ProofBundle, block proof.Block) {
	r.check(groupStructure, CatStructural, ptype.IsValidSubject(bundle.T),
		fmt.Sprintf("Subject type %d (%s) is registered", uint16(bundle.T), ptype.Name(bundle.T)))
	r.check(groupStructure, CatStructural, block.ID != "", "Block present with ID")
	r.check(groupStructure, CatStructural, block.MerkleRoot != "", "Block has merkle_root")
	r.check(groupStructure, CatStructural, len(bundle.Commitments) > 0, "At least one external commitment")
}

func verifyVersion(r *Report, bundle *proof.ProofBundle) {
	r.check(groupStructure, CatStructural, bundle.Version == expectedVersion,
		fmt.Sprintf("Proof version %d (expected %d)", bundle.Version, expectedVersion))
}

// --- Step 3: Claims Hash ---

func deriveClaimsHash(r *Report, rawClaims json.RawMessage) string {
	if len(rawClaims) == 0 || string(rawClaims) == "null" {
		r.skip(groupSubjectData, CatCryptographic, "Claims hash skipped (redacted)")
		return ""
	}

	canonical, err := jcs.Transform(rawClaims)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic, fmt.Sprintf("Claims JCS failed: %s", err))
		return ""
	}

	hash := tscrypto.BytesToHex(tscrypto.DomainHash(tscrypto.PrefixItemClaims, canonical))
	r.pass(groupSubjectData, CatCryptographic, "Claims hash derived (0x11)")
	return hash
}

// --- Step 4: Claims Hash Type ---

func validateClaimsHashType(r *Report, claims Claims) {
	if claims.Hash == "" || claims.HashType == "" {
		return
	}
	if err := tscrypto.ValidateClaimsHash(claims.Hash, claims.HashType); err != nil {
		r.warn(groupSubjectData, CatDataIntegrity, fmt.Sprintf("Claims hash validation: %s", err))
	} else {
		r.pass(groupSubjectData, CatDataIntegrity, fmt.Sprintf("Claims hash length valid for %s", claims.HashType))
	}
}

// --- Step 5: Claims Timestamp ---

func verifyClaimsTimestamp(r *Report, subject *proof.Subject) {
	if subject == nil {
		return
	}
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
		r.warn(groupSubjectData, CatTiming, "Claims timestamp is not before submission time (future-dated claim)")
	} else if submittedTime.Sub(claimedTime) > 7*24*time.Hour {
		days := int(submittedTime.Sub(claimedTime).Hours() / 24)
		r.Claims.TimestampStatus = TimestampStale
		r.Claims.TimestampNote = fmt.Sprintf("Claims timestamp is %d days before submission (stale claim)", days)
		r.warn(groupSubjectData, CatTiming, fmt.Sprintf("Claims timestamp is %d days before submission (stale claim)", days))
	}
}

// --- Step 6: Item Hash ---

func deriveItemHash(r *Report, subject *proof.Subject, claimsHash string) string {
	if subject == nil {
		r.fail(groupSubjectData, CatCryptographic, "Cannot derive item hash (missing subject)")
		return ""
	}
	if claimsHash == "" || subject.MetadataHash == "" || subject.SigningKeyID == "" {
		r.fail(groupSubjectData, CatCryptographic, "Cannot derive item hash (missing inputs)")
		return ""
	}

	hash, err := tscrypto.ComputeItemHash(subject.ID, claimsHash, subject.MetadataHash, subject.SigningKeyID)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic, fmt.Sprintf("Item hash computation failed: %s", err))
		return ""
	}

	r.pass(groupSubjectData, CatCryptographic, "Item hash derived (0x13)")
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
		r.fail(groupBlockHash, CatCryptographic, fmt.Sprintf("Block hash computation failed: %s", err))
		return ""
	}

	r.pass(groupBlockHash, CatCryptographic, "Block hash derived (0x32)")
	return computed
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

		target := epochTarget(cx)
		if target == "" {
			r.fail(groupEpoch, CatCryptographic, fmt.Sprintf("Epoch proof %d: missing target for commitment type %d", i, uint16(cx.Type)))
			epochRoots = append(epochRoots, "")
			continue
		}

		valid, err := tscrypto.VerifyMerkleProof(blockHash, proofList, target)
		if err != nil {
			r.fail(groupEpoch, CatCryptographic, fmt.Sprintf("Epoch proof %d error: %s", i, err))
			epochRoots = append(epochRoots, "")
			continue
		}

		r.check(groupEpoch, CatCryptographic, valid,
			fmt.Sprintf("Epoch proof %d (%s): block hash maps to committed value (%d steps)",
				i, ptype.Humanize(cx.Type), len(proofList)))
		epochRoots = append(epochRoots, target)
	}

	return epochRoots
}

// epochTarget returns the committed value for an external commitment:
// memo for stellar (t=40), op_return for bitcoin (t=41).
func epochTarget(cx ExternalCommit) string {
	switch cx.Type {
	case ptype.CommitmentStellar:
		return cx.MemoHash
	case ptype.CommitmentBitcoin:
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

	tsTime, err := time.Parse(time.RFC3339, bundle.Timestamp)
	if err != nil {
		r.fail(groupProof, CatCryptographic, fmt.Sprintf("Cannot parse proof timestamp: %s", err))
		return
	}
	timestampMs := uint64(tsTime.UnixMilli())

	proofHashBytes, err := tscrypto.BuildCompactProofPayload(
		byte(bundle.Version), uint16(bundle.T), keyID, timestampMs, subjectHash, blockHash, epochRoots,
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

func verifySubjectTemporalWindow(r *Report, t ptype.Code, subject *proof.Subject, block proof.Block) {
	if subject == nil {
		return
	}

	var subjectTime time.Time
	var sErr error
	if t == ptype.Item {
		subjectTime, sErr = tscrypto.ExtractULIDTimestamp(subject.ID)
	} else {
		subjectTime, sErr = tscrypto.ExtractUUIDv7Timestamp(subject.ID)
	}
	if sErr != nil {
		return
	}

	blockTime, bErr := tscrypto.ExtractUUIDv7Timestamp(block.ID)
	if bErr != nil {
		return
	}

	ok := !subjectTime.After(blockTime)
	label := "Subject"
	switch {
	case t == ptype.Item:
		label = "Item submitted"
	case ptype.IsEntropySubject(t):
		label = "Entropy captured"
	}
	r.check(groupTemporal, CatTiming,
		ok,
		fmt.Sprintf("%s before committed block (%s)", label, blockTime.Format(time.RFC3339)))
}

func addTemporalInfo(r *Report, t ptype.Code, subject *proof.Subject, block proof.Block) {
	switch {
	case t == ptype.Item && subject != nil:
		r.Temporal.SubmittedAt = tscrypto.FormatItemTime(subject.ID)
		r.Temporal.CommittedAt = tscrypto.FormatBlockTime(block.ID)
	case ptype.IsEntropySubject(t) && subject != nil:
		r.Temporal.CapturedAt = tscrypto.FormatBlockTime(subject.ID)
		r.Temporal.CommittedAt = tscrypto.FormatBlockTime(block.ID)
	default:
		// Block subject: no subject timestamp, just the block.
		r.Temporal.CommittedAt = tscrypto.FormatBlockTime(block.ID)
	}
}

// --- Step 13: Entropy Source consistency ---

// verifyEntropySource confirms that s.d matches the canonical published
// value at the upstream source. Each entropy type fetches a specific
// upstream document and byte-compares identifier fields. For NIST Beacon
// we intentionally do not validate the pulse's X.509 signature chain —
// the Truestamp service stores only the minimal pulse fields
// (chainIndex/pulseIndex/outputValue/timeStamp/version), so the subject
// hash it signed is already over exactly the slice of data we compare.
//
// Network selection:
//   - entropy_stellar: matches the deployment's Stellar network. A given
//     Truestamp deployment uses one Stellar network for both entropy
//     observation and commitment, so we derive the network from the
//     bundle's Stellar commitment in cx[]. Falls back to "public" if
//     no Stellar commitment is present.
//   - entropy_bitcoin: always Bitcoin mainnet. The server captures
//     mainnet Bitcoin blocks as the authoritative public-randomness
//     source even in dev/test/staging deployments that commit to
//     testnet/regtest.
//   - entropy_nist: NIST Beacon is a single global public source.
func verifyEntropySource(r *Report, t ptype.Code, subject *proof.Subject, commits []proof.ExternalCommit, opts Options) {
	if opts.SkipExternal {
		r.skip(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Entropy source verification skipped for %s (--skip-external)", ptype.Humanize(t)))
		return
	}
	if subject == nil || len(subject.Data) == 0 {
		r.fail(groupEntropySource, CatBlockchain, "Missing entropy subject data")
		return
	}

	switch t {
	case ptype.EntropyNIST:
		verifyEntropyNIST(r, subject.Data)
	case ptype.EntropyStellar:
		verifyEntropyStellar(r, subject.Data, entropyNetwork(commits, ptype.CommitmentStellar, "public"))
	case ptype.EntropyBitcoin:
		// Bitcoin entropy is always captured from mainnet — even in dev
		// environments that commit to regtest/testnet. Do not derive
		// from cx[] commitments.
		verifyEntropyBitcoin(r, subject.Data, "mainnet")
	default:
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Unsupported entropy type code %d", uint16(t)))
	}
}

// entropyNetwork returns the network string for the first cx[] entry
// matching the given commitment code, or the fallback if none match.
func entropyNetwork(commits []proof.ExternalCommit, code ptype.Code, fallback string) string {
	for _, c := range commits {
		if c.Type == code && c.Network != "" {
			return c.Network
		}
	}
	return fallback
}

func verifyEntropyNIST(r *Report, rawData json.RawMessage) {
	// NIST pulses may arrive wrapped in {"pulse": {...}} or flat.
	var env struct {
		Pulse *struct {
			ChainIndex  int    `json:"chainIndex"`
			PulseIndex  int    `json:"pulseIndex"`
			OutputValue string `json:"outputValue"`
			TimeStamp   string `json:"timeStamp"`
			Version     string `json:"version"`
		} `json:"pulse"`
		ChainIndex  int    `json:"chainIndex"`
		PulseIndex  int    `json:"pulseIndex"`
		OutputValue string `json:"outputValue"`
		TimeStamp   string `json:"timeStamp"`
		Version     string `json:"version"`
	}
	if err := json.Unmarshal(rawData, &env); err != nil {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Cannot parse NIST pulse from subject data: %s", err))
		return
	}
	chainIdx, pulseIdx, outputValue, timeStamp := env.ChainIndex, env.PulseIndex, env.OutputValue, env.TimeStamp
	if env.Pulse != nil {
		chainIdx = env.Pulse.ChainIndex
		pulseIdx = env.Pulse.PulseIndex
		outputValue = env.Pulse.OutputValue
		timeStamp = env.Pulse.TimeStamp
	}
	if outputValue == "" {
		r.fail(groupEntropySource, CatBlockchain, "NIST entropy subject missing outputValue")
		return
	}

	remote, err := external.GetNISTPulse(chainIdx, pulseIdx)
	if err != nil {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("NIST Beacon fetch failed (chain %d, pulse %d): %s", chainIdx, pulseIdx, err))
		return
	}
	if remote.OutputValue != outputValue {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("NIST outputValue mismatch at pulse %d: upstream %s, proof %s",
				pulseIdx, remote.OutputValue, outputValue))
		return
	}
	if !timestampsEqual(remote.TimeStamp, timeStamp) {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("NIST timeStamp mismatch at pulse %d: upstream %s, proof %s",
				pulseIdx, remote.TimeStamp, timeStamp))
		return
	}
	r.pass(groupEntropySource, CatBlockchain,
		fmt.Sprintf("NIST Beacon pulse %d (chain %d) confirmed upstream", pulseIdx, chainIdx))
}

func verifyEntropyStellar(r *Report, rawData json.RawMessage, network string) {
	var stellar struct {
		Hash     string `json:"hash"`
		Sequence int    `json:"sequence"`
		ClosedAt string `json:"closed_at"`
	}
	if err := json.Unmarshal(rawData, &stellar); err != nil {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Cannot parse Stellar ledger from subject data: %s", err))
		return
	}
	if stellar.Sequence == 0 || stellar.Hash == "" {
		r.fail(groupEntropySource, CatBlockchain, "Stellar entropy subject missing sequence or hash")
		return
	}

	remote, err := external.GetStellarLedger(stellar.Sequence, network)
	if err != nil {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Stellar ledger fetch failed (sequence %d on %s): %s", stellar.Sequence, network, err))
		return
	}
	if !tscrypto.HexEqual(remote.Hash, stellar.Hash) {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Stellar ledger hash mismatch at seq %d: upstream %s, proof %s",
				stellar.Sequence, remote.Hash, stellar.Hash))
		return
	}
	if !timestampsEqual(remote.ClosedAt, stellar.ClosedAt) {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Stellar ledger closed_at mismatch at seq %d: upstream %s, proof %s",
				stellar.Sequence, remote.ClosedAt, stellar.ClosedAt))
		return
	}
	r.pass(groupEntropySource, CatBlockchain,
		fmt.Sprintf("Stellar ledger %d confirmed on %s", stellar.Sequence, network))
}

func verifyEntropyBitcoin(r *Report, rawData json.RawMessage, network string) {
	var btc struct {
		Hash   string `json:"hash"`
		Height int    `json:"height"`
		Time   int64  `json:"time"`
	}
	if err := json.Unmarshal(rawData, &btc); err != nil {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Cannot parse Bitcoin block from subject data: %s", err))
		return
	}
	if btc.Hash == "" {
		r.fail(groupEntropySource, CatBlockchain, "Bitcoin entropy subject missing hash")
		return
	}

	remote, skipped, err := external.GetBitcoinBlockHeader(btc.Hash, network)
	if skipped {
		r.skip(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Bitcoin entropy source verification skipped: no public API for %s", network))
		return
	}
	if err != nil {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Bitcoin block fetch failed (hash %s on %s): %s", btc.Hash, network, err))
		return
	}
	if !tscrypto.HexEqual(remote.Hash, btc.Hash) {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Bitcoin block hash mismatch: upstream %s, proof %s",
				remote.Hash, btc.Hash))
		return
	}
	if btc.Height != 0 && remote.Height != btc.Height {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Bitcoin height mismatch at hash %s: upstream %d, proof %d",
				btc.Hash, remote.Height, btc.Height))
		return
	}
	if btc.Time != 0 && remote.Time != btc.Time {
		r.fail(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Bitcoin block time mismatch at height %d: upstream %d, proof %d",
				remote.Height, remote.Time, btc.Time))
		return
	}
	r.pass(groupEntropySource, CatBlockchain,
		fmt.Sprintf("Bitcoin block %d confirmed on %s", remote.Height, network))
}

// timestampsEqual compares two ISO 8601 timestamp strings after parsing,
// so "2026-04-22T19:45:00Z" and "2026-04-22T19:45:00.000Z" are treated
// as equal. Falls back to string equality on parse failure.
func timestampsEqual(a, b string) bool {
	if a == b {
		return true
	}
	ta, errA := time.Parse(time.RFC3339Nano, a)
	tb, errB := time.Parse(time.RFC3339Nano, b)
	if errA != nil || errB != nil {
		return false
	}
	return ta.Equal(tb)
}

// --- Step 14: Stellar Commitments ---

func verifyStellarCommitments(r *Report, commits []proof.ExternalCommit, blockHash string, opts Options) {
	for i := range commits {
		if commits[i].Type != ptype.CommitmentStellar {
			continue
		}
		verifySingleStellar(r, &commits[i], blockHash, opts)
	}
}

func verifySingleStellar(r *Report, cx *proof.ExternalCommit, blockHash string, opts Options) {
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
	} else if cx.Network != "testnet" && cx.Network != "public" {
		r.fail(groupStellar, CatBlockchain,
			fmt.Sprintf("Stellar net %q is not a recognised Stellar network (expected testnet or public)", cx.Network))
	} else {
		result, err := external.VerifyStellar(cx.TransactionHash, cx.MemoHash, cx.Network, cx.Ledger)
		if err != nil {
			r.fail(groupStellar, CatBlockchain, fmt.Sprintf("Stellar external verification failed: %s", err))
		} else {
			r.pass(groupStellar, CatBlockchain, fmt.Sprintf("Transaction %s confirmed on %s (ledger %d)", cx.TransactionHash, cx.Network, result.Ledger))
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
		if commits[i].Type != ptype.CommitmentBitcoin {
			continue
		}
		verifySingleBitcoin(r, &commits[i], blockHash, opts)
	}
}

func verifySingleBitcoin(r *Report, cx *proof.ExternalCommit, blockHash string, opts Options) {
	extractedOpReturn, err := bitcoin.ExtractOpReturn(cx.RawTxHex)
	if err != nil {
		r.fail(groupBitcoin, CatBlockchain, fmt.Sprintf("OP_RETURN extraction failed: %s", err))
	} else {
		r.check(groupBitcoin, CatBlockchain,
			tscrypto.HexEqual(extractedOpReturn, cx.OpReturn),
			"OP_RETURN extracted from raw transaction matches")
	}

	computedTxid, err := bitcoin.ComputeTxID(cx.RawTxHex)
	if err != nil {
		r.fail(groupBitcoin, CatBlockchain, fmt.Sprintf("Txid computation failed: %s", err))
	} else {
		r.check(groupBitcoin, CatBlockchain,
			tscrypto.HexEqual(computedTxid, cx.TransactionHash),
			fmt.Sprintf("Transaction %s verified from raw tx", cx.TransactionHash))
	}

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
			if result.Timestamp != "" {
				ci.Timestamp = result.Timestamp
			}
		}
	}

	r.CommitmentInfos = append(r.CommitmentInfos, ci)
}

// --- Entropy Hash ---

func deriveEntropyHash(r *Report, rawData json.RawMessage) string {
	if len(rawData) == 0 || string(rawData) == "null" {
		r.fail(groupSubjectData, CatCryptographic, "Entropy data missing")
		return ""
	}

	canonical, err := jcs.Transform(rawData)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic, fmt.Sprintf("Entropy JCS failed: %s", err))
		return ""
	}

	hash := tscrypto.ComputeEntropyHash(canonical)
	r.pass(groupSubjectData, CatCryptographic, "Entropy hash derived (0x21)")
	return hash
}

// --- Observation Hash ---

func deriveObservationHash(r *Report, subject *proof.Subject, entropyHash string) string {
	if subject == nil {
		r.fail(groupSubjectData, CatCryptographic, "Cannot derive observation hash (missing subject)")
		return ""
	}
	if entropyHash == "" || subject.MetadataHash == "" || subject.SigningKeyID == "" {
		r.fail(groupSubjectData, CatCryptographic, "Cannot derive observation hash (missing inputs)")
		return ""
	}

	hash, err := tscrypto.ComputeObservationHash(subject.ID, entropyHash, subject.MetadataHash, subject.SigningKeyID)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic, fmt.Sprintf("Observation hash computation failed: %s", err))
		return ""
	}

	r.pass(groupSubjectData, CatCryptographic, "Observation hash derived (0x23)")
	return hash
}

// --- Entropy Subject Parsing (display only) ---

func parseEntropySubject(t ptype.Code, rawEntropy json.RawMessage) EntropySubject {
	subject := EntropySubject{RawSource: ptype.Name(t), Source: ptype.Humanize(t)}

	if len(rawEntropy) == 0 {
		return subject
	}

	switch t {
	case ptype.EntropyNIST:
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

	case ptype.EntropyBitcoin:
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

	case ptype.EntropyStellar:
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
	}

	return subject
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
