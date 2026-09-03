// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package verify runs the Appendix E verification pipeline against a parsed
// proof bundle and produces a [Report] of graded step results.
//
// Every step below corresponds to a numbered section of Appendix E
// ("Cryptographic Proof Verification Reference", normative) of
// `truestamp-v2/whitepaper/whitepaper.typ`, and the reference verifier
// `whitepaper/verify_proof.exs` is the behavioral oracle: step order,
// status choices and row wording follow it, so that the two produce reports
// whose statuses match per E.25 on every bundle.
//
// Offline operation is first class: every step from E.6 through E.16, the
// witness step E.17a and the submission window E.20 run with no network
// access. The network-dependent steps (E.18, E.19's binding lookup, E.21,
// the key event's chain confirmations, and the E.17 keyring fetch) run
// only when the caller allows them and report `skip` on any failure to
// reach a source; a skipped check never fails a proof.
package verify

import (
	"fmt"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/proof"
)

// Options holds the caller's choices for a verification run.
type Options struct {
	// ExpectedHash is the hash of the file the caller holds, compared
	// against subject.claims.hash for an item subject (E.7). Trimmed and
	// lowercased before use.
	ExpectedHash string

	// ExpectedSubjectType, when non-empty, asserts the bundle's signed
	// `type`. A mismatch is the hard rejection `subject_type_mismatch`.
	ExpectedSubjectType string

	// SkipExternal runs offline: no Horizon, Blockstream, NIST or keyring
	// lookups. Every network-dependent step reports skip.
	SkipExternal bool

	// SkipSignatures leaves E.16's Ed25519 check and the keyring
	// cross-check unperformed; the report discloses it.
	SkipSignatures bool

	// KeyringFile pins a local copy of /.well-known/keyring.json for E.17.
	// It takes precedence over KeyringURL and needs no network.
	KeyringFile string

	// KeyringURL is fetched for E.17 when no file is pinned and the run is
	// online. Empty means no keyring is fetched.
	KeyringURL string
}

// Run executes the full verification pipeline on a proof file.
func Run(filename string, opts Options) (*Report, error) {
	bundle, err := proof.Parse(filename)
	if err != nil {
		return nil, err
	}
	return runBundle(bundle, filename, proof.FileSize(filename), opts)
}

// RunFromBytes executes the full verification pipeline on raw proof bytes.
func RunFromBytes(data []byte, displayName string, opts Options) (*Report, error) {
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		return nil, err
	}
	return runBundle(bundle, displayName, proof.FileSizeFromData(data), opts)
}

// RunBundle executes the pipeline on an already parsed bundle.
func RunBundle(bundle *proof.Bundle, displayName string, opts Options) (*Report, error) {
	return runBundle(bundle, displayName, int64(len(bundle.JSON)), opts)
}

// runBundle runs the steps in dependency order. Rows are appended in the
// reference verifier's emission order, so a report grouped by category
// lines up with the reference's row for row.
func runBundle(bundle *proof.Bundle, filename string, fileSize int64, opts Options) (*Report, error) {
	// The --type assertion is a hard rejection, named the way the server
	// names it, because E.25 forbids a verifier adding a `fail` row for a
	// step Appendix D.4 does not report, and because the mismatch is a
	// statement about the caller's expectation rather than about the
	// bundle.
	if opts.ExpectedSubjectType != "" && opts.ExpectedSubjectType != bundle.Type {
		return nil, proof.Rejectf(proof.CodeSubjectTypeMismatch,
			"the bundle's signed type is %s but --type %s was asserted", bundle.Type, opts.ExpectedSubjectType)
	}

	r := newReport(bundle, filename, fileSize, opts)
	blockLike := bundle.IsBlockLike()

	// [E.7] Expected-hash comparison, then [E.8] version, then [E.4] the
	// hex sweep before any hash is derived.
	stepExpectedHash(r, bundle, opts)
	stepVersion(r, bundle)
	stepEncoding(r, bundle)

	// [E.9] The public key, and the key id derived from it.
	publicKey, keyID := stepPublicKey(r, bundle)

	// [E.10/E.11] The subject hash. Block-like subjects have none of their
	// own: their subject hash IS the block hash, assigned after E.14.
	var subjectHash string
	if !blockLike {
		subjectHash = stepSubjectHash(r, bundle)
	}

	// [E.13] Does the subject sit in the block's Merkle tree?
	stepInclusionProof(r, bundle, subjectHash)

	// [E.14] The block hash, recomputed from the block's five fields.
	blockHash := stepBlockHash(r, bundle)
	if blockLike {
		subjectHash = blockHash
	}

	// [E.15] Does the block sit in each epoch tree that was committed?
	for i := range bundle.Commitments {
		walkEpochProof(r, &bundle.Commitments[i], blockHash, groupEpoch, CatCryptographic)
	}

	// [E.16] The signature over everything derived above.
	stepSignature(r, bundle, publicKey, keyID, subjectHash, blockHash, opts)

	// [E.17] Is that key Truestamp's? The keyring path and the carried key
	// event path each get their own group.
	stepKeyring(r, bundle, publicKey, keyID, opts)
	stepSigningKeyEvent(r, bundle, publicKey, keyID, opts)

	// [E.17a] The witnesses that open the submitted-after edge.
	witnesses := stepWitnesses(r, bundle)

	// [E.18/E.19/E.21] The network-dependent steps.
	confirmedChains, confirmedWitnesses := stepExternal(r, bundle, witnesses, opts)

	// [E.20] The submission window.
	stepSubmissionWindow(r, bundle, witnesses, confirmedChains, confirmedWitnesses)

	return r, nil
}

func newReport(bundle *proof.Bundle, filename string, fileSize int64, opts Options) *Report {
	r := &Report{
		Filename:        filename,
		FileSize:        fileSize,
		Format:          "json",
		ProofVersion:    bundle.Version,
		VersionLiteral:  literal(bundle.VersionLiteral),
		SubjectType:     bundle.Type,
		SubjectID:       bundle.SubjectID(),
		BlockID:         bundle.Block.ID,
		GeneratedAt:     bundle.GeneratedAt,
		KeyEventCarried: bundle.SigningKeyEvent != nil,
		SkippedExternal: opts.SkipExternal,
	}
	if bundle.FromCBOR {
		r.Format = "cbor"
	}
	if bundle.Subject != nil {
		r.WitnessesCarried = bundle.Subject.WitnessNamesCarried()
	}
	for _, c := range bundle.Commitments {
		r.Commitments = append(r.Commitments, CommitmentSummary{Chain: c.Chain, Network: c.Network})
	}
	if h := strings.ToLower(strings.TrimSpace(opts.ExpectedHash)); h != "" {
		r.ExpectedHash = h
	}
	return r
}

// literal renders a raw JSON value for a message, or "absent".
func literal(raw []byte) string {
	if len(raw) == 0 {
		return "absent"
	}
	s := strings.Join(strings.Fields(string(raw)), " ")
	if len(s) > 64 {
		return s[:64] + "..."
	}
	return s
}

// short abbreviates a hex value the way the reference verifier does in its
// offline skip rows.
func short(value string) string {
	if value == "" {
		return "(absent)"
	}
	if len(value) <= 16 {
		return value + "..."
	}
	return value[:16] + "..."
}

func stepsLabel(n int) string {
	if n == 1 {
		return "1 step"
	}
	return fmt.Sprintf("%d steps", n)
}

func blocksLabel(n int) string {
	if n == 1 {
		return "1 block"
	}
	return fmt.Sprintf("%d blocks", n)
}
