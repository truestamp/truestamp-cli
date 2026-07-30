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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chainhash/v2"
	"github.com/truestamp/truestamp-cli/internal/bitcoin"
	"github.com/truestamp/truestamp-cli/internal/external"
	"github.com/truestamp/truestamp-cli/internal/jcs"
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

	// Hash comparison (E.7).
	//
	// E.22 requires "an expected hash was supplied" to be readable
	// separately from "it matched", so the caller's argument is recorded
	// on EVERY branch, including the ones that cannot compare it. A
	// consumer that infers "none was supplied" from the field's absence
	// otherwise cannot tell a skip from a run with no --hash at all,
	// which is the one distinction the exposure exists to carry.
	if opts.ExpectedHash != "" {
		r.HashProvided = opts.ExpectedHash
	}
	switch {
	case !isItem:
		// E.7 scopes this step to t=20 and forbids failing it for any
		// other subject: an entropy or block-like payload carries no
		// s.d.hash, so treating the caller's argument as a mismatch
		// would report a sound proof as forged. The skip is required,
		// not optional — an ignored argument must stay visible.
		if opts.ExpectedHash != "" {
			r.skip(groupHashComparison, CatDataIntegrity, fmt.Sprintf(
				"--hash not applicable: only an item subject (t=20) commits to a file hash (this proof is %s)",
				subjectType))
		}

	case r.Claims.Hash == "":
		// Claims-as-source-of-truth item (no s.d.hash). Make the mode
		// explicit so the absence of a Hash row in the Item Claims
		// section is explained rather than left as a silent gap.
		r.info(groupVerificationNotes, CatDataIntegrity,
			"Claims-only item: the claims content itself is what was timestamped — no external file to compare")
		// E.7's "no s.d.hash" rule is about the subject, not about `t`:
		// this item is a t=20 that timestamped its claims content
		// directly, so there is no file hash on the proof side to
		// compare the caller's argument against. Comparing anyway made
		// HexEqual(x, "") false and reported a sound proof as covering
		// different data than the caller's — a mismatch against an
		// operand the proof never carried.
		if opts.ExpectedHash != "" {
			r.skip(groupHashComparison, CatDataIntegrity,
				"--hash not applicable: this item timestamped its claims content itself and carries no s.d.hash to compare a local file against")
		}

	case opts.ExpectedHash != "":
		// User explicitly supplied a hash to compare; pass/fail stays
		// under groupHashComparison so the rest of the presenter
		// (HashMatched, ProofPassed, hash-mismatch detail) continues to
		// find it where it expects.
		if tscrypto.HexEqual(opts.ExpectedHash, r.Claims.Hash) {
			r.pass(groupHashComparison, CatDataIntegrity, "Provided hash matches claims.hash")
		} else {
			r.fail(groupHashComparison, CatDataIntegrity, fmt.Sprintf(
				"Provided hash does not match claims.hash (expected: %s, proof: %s)",
				opts.ExpectedHash, r.Claims.Hash))
		}

	default:
		// External-hash item, no --hash supplied. E.7 names this warn as
		// a Hash Comparison result, so it is filed there rather than
		// under the CLI's own Verification Notes group: a consumer
		// keying on the group E.22 defines has to find it.
		//
		// The message states only what this branch established — which
		// hash the proof commits to, and that nothing was compared
		// against it. It used to assert "the proof itself is verified",
		// which is emitted here at pipeline position 2, before the
		// signature, inclusion, epoch and block-hash steps have run, and
		// so appeared verbatim inside reports whose verdict was `failed`
		// and inside --skip-signatures runs that checked no signature at
		// all. E.22 forbids a message that reads as an assertion its
		// branch did not establish.
		hashType := r.Claims.HashType
		if hashType == "" {
			hashType = "sha256"
		}
		r.warn(groupHashComparison, CatDataIntegrity, fmt.Sprintf(
			"File hash not verified: the proof commits to %s (%s) but no --hash was supplied to compare a local file against it",
			r.Claims.Hash, hashType))
	}

	// Step 0b: E.4's hex-encoding sweep. It runs here rather than beside
	// verifyVersion because E.4 requires the sweep "before deriving any
	// hash", and E.9's key-id derivation below is one. No swept field
	// feeds that derivation — `pk` is base64, not one of the ten — so the
	// old placement was compliant on the substance; this placement is
	// compliant on the letter as well, and costs nothing.
	verifyHexEncoding(r, bundle)

	// Step 1: Signing Key
	pubkeyBytes, keyID := verifySigningKey(r, bundle)

	// The report names the key that actually signed this bundle, which
	// E.9/E.16 make the pk-DERIVED key id — the value fed to the 0x61
	// payload — not the stored `b.kid`. E.9 blesses key rotation, under
	// which the two differ, and this field used to carry b.kid: a
	// rotated proof printed "Public key valid, key_id: 0c15f1f3" next to
	// "signed with key 11223344" and named the wrong key as the signer.
	// b.kid stays a verbatim input to the 0x32 preimage (E.14); it is
	// not an assertion about who signed.
	r.SigningKeyID = keyID

	// ...and separately records the block's own `b.kid`, which is a
	// verbatim input to E.14's 0x32 preimage rather than a claim about
	// who signed. Remote mode has always populated it; local mode did
	// not, so every display site that means "the block's key" fell back
	// to the DERIVED signer id. On a rotated bundle the two disagree,
	// and the two modes then published different `signing_key` values
	// for the same block.
	r.BlockSigningKeyID = block.SigningKeyID

	// Step 2: Structure. E.22 files two things under this group — E.8's
	// version check here, and E.4's hex-encoding sweep, already run above.
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

	// Step 7: Inclusion Proof (not applicable to block-like subjects)
	if !isBlockLike {
		verifyInclusionProof(r, subjectHash, bundle.InclusionProof, block)
	} else {
		// E.13 requires this as a reported skip rather than an omission,
		// so a reader can tell a check that did not apply from one that
		// was never reached (E.22).
		r.skip(groupInclusion, CatCryptographic,
			"Inclusion proof not applicable: a block-like subject IS the block, so there is no leaf to prove")
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

	// Step 11a: Key Binding. Deliberately reported after the signature
	// check: E.16 establishes only that SOME key signed this bundle, and
	// E.17 is the step that turns that into "Truestamp signed it".
	// Reporting it earlier inverts the argument, and its absence would
	// let a report read as having established a binding it never
	// attempted.
	verifyKeyBinding(r, bundle.PublicKey, pubkeyBytes, keyID, opts)

	// Step 12: Temporal checks + info
	if !isBlockLike {
		verifySubjectTemporalWindow(r, bundle.T, bundle.Subject, block)
		reportSubmissionWindowEdges(r, commits)
	}
	addTemporalInfo(r, bundle.T, bundle.Subject, block)

	// Step 13: Entropy source consistency (entropy subjects only, external, skippable)
	if isEntropy {
		verifyEntropySource(r, bundle.T, bundle.Subject, commits, opts)
	}

	// Step 14: Stellar Commitment
	verifyStellarCommitments(r, commits, opts)

	// Step 15: Bitcoin Commitment
	verifyBitcoinCommitments(r, commits, opts)

	return r, nil
}

// Step group names. Every group Appendix E.22 defines carries E.22's
// spelling verbatim, because a consumer that keys on a group name is
// reading the appendix, not this file. groupSubjectType,
// groupVerificationNotes and (in remote.go) groupServerVerdict are the
// CLI-specific groups E.22 does not define; groupSubmittedBefore and
// groupSubmittedAfter are outside E.22's table too, but take their
// spellings from Section 3's public captions, which the reference
// verifier shares. None of the five emits a `pass` or a `warn`, so none
// of them can break E.25's containment rule.
const (
	groupHashComparison = "Hash Comparison"
	groupSigningKey     = "Signing Key"
	// groupKeyBinding carries E.17's keyring cross-check. It is separate
	// from groupSigningKey on purpose: E.9 establishes that `pk` decodes
	// and yields a key id, which says nothing about whose key it is, and
	// merging the two lets a report be read as having bound a key to
	// Truestamp when it only parsed one.
	groupKeyBinding       = "Key Binding"
	groupStructure        = "Structure"
	groupSubjectType      = "Subject Type"
	groupSubjectData      = "Subject Data"
	groupInclusion        = "Inclusion Proof"
	groupBlockHash        = "Block Hash"
	groupProof            = "Proof Signature"
	groupEpoch            = "Epoch Proof"
	groupStellar          = "Stellar Commitment"
	groupBitcoin          = "Bitcoin Commitment"
	groupSubmissionWindow = "Submission Window"
	groupTemporalInfo     = "Temporal Info"
	groupEntropySource    = "Entropy Source"

	// groupSubmittedBefore and groupSubmittedAfter carry E.20's two edge
	// statements. They are separate groups rather than extra Submission
	// Window rows because E.20 makes them different propositions from the
	// ordering check: the ordering constraint is asserted by Truestamp
	// and graded, while these two name which edge of the submission
	// window a bundle can and cannot ground outside Truestamp. Both are
	// always `info`. The spellings are Section 3's public captions
	// ("Submitted after" / "Submitted before"), which the reference
	// verifier also uses, so the two implementations' reports line up.
	groupSubmittedBefore = "Submitted Before"
	groupSubmittedAfter  = "Submitted After"

	// groupVerificationNotes holds workflow-level observations about the
	// verify session that no Appendix E step defines — currently only
	// the claims-only acknowledgment, which explains why the report
	// shows no Hash row rather than leaving the absence silent. The
	// presenter renders these in their own section rather than under
	// "Issues" so they read as context, not failed checks.
	//
	// E.7's "file hash not verified" warn used to live here and does
	// not any more: it is a Hash Comparison result the appendix names,
	// so it is filed under that group where a consumer can find it.
	groupVerificationNotes = "Verification Notes"
)

// --- Step 1: Signing Key ---

// verifySigningKey covers E.9 alone: `pk` base64-decodes to 32 bytes and
// yields a derived key id. Whose key it is belongs to E.17, reported by
// verifyKeyBinding after the signature check.
func verifySigningKey(r *Report, bundle *proof.ProofBundle) ([]byte, string) {
	pubkey, err := tscrypto.DecodePublicKey(bundle.PublicKey)
	if err != nil {
		r.fail(groupSigningKey, CatCryptographic, fmt.Sprintf("Failed to decode public key: %s", err))
		return nil, ""
	}

	keyID := tscrypto.ComputeKeyID(pubkey)
	r.pass(groupSigningKey, CatCryptographic, fmt.Sprintf("Public key valid, key_id: %s", keyID))
	return pubkey, keyID
}

// --- Step 11a: Key Binding (E.17) ---

// verifyKeyBinding cross-checks the derived key id and `pk` against the
// published keyring. It always emits exactly one graded row, whatever
// happened, because E.22 requires this result be reported rather than
// omitted: a report with no Key Binding row reads as if the binding were
// never in question, which is precisely the reading E.17 exists to
// prevent.
//
// Only a keyring that answered and does not vouch for the key may fail
// the proof. An unreachable, absent or unintelligible keyring
// establishes nothing either way, and E.22 forbids a skipped external
// check from failing a proof — before this grading a network outage
// reported a sound proof as defective.
func verifyKeyBinding(r *Report, publicKeyB64 string, pubkey []byte, keyID string, opts Options) {
	switch {
	case pubkey == nil:
		// E.9's decode failure already failed its own step; there is no
		// usable key to look up, so this one cannot even be attempted.
		r.skip(groupKeyBinding, CatCryptographic,
			"Key binding not checked: no usable public key")
		return

	case opts.SkipSignatures:
		r.skip(groupKeyBinding, CatCryptographic,
			"Keyring cross-check not performed (--skip-signatures); pk is not bound to Truestamp by this run")
		return

	case opts.SkipExternal:
		r.skip(groupKeyBinding, CatCryptographic,
			"Keyring cross-check not performed (--skip-external); pk is not bound to Truestamp by this run")
		return
	}

	err := external.VerifyKeyring(map[string]string{keyID: publicKeyB64}, opts.KeyringURL)
	switch external.Classify(err) {
	case external.OutcomeOK:
		r.pass(groupKeyBinding, CatCryptographic, fmt.Sprintf("Key %s confirmed via keyring", keyID))

	case external.OutcomeMismatch:
		// The keyring answered and either does not carry this key id or
		// publishes different bytes for it. This is the one branch that
		// distinguishes an attacker's self-consistent bundle from a
		// Truestamp one, so it fails.
		r.fail(groupKeyBinding, CatCryptographic, fmt.Sprintf("Key binding failed: %s", err))

	case external.OutcomeNotFound:
		r.skip(groupKeyBinding, CatCryptographic, fmt.Sprintf(
			"Keyring cross-check not performed: no keyring published at %s (HTTP 404)", opts.KeyringURL))
		r.info(groupKeyBinding, CatCryptographic, keyBindingUnestablishedNote)

	default:
		// Unavailable or Malformed: no answer, or one this client cannot
		// read. Either way nothing was established.
		r.skip(groupKeyBinding, CatCryptographic,
			fmt.Sprintf("Keyring cross-check not performed: %s", err))
		r.info(groupKeyBinding, CatCryptographic, keyBindingUnestablishedNote)
	}
}

// keyBindingUnestablishedNote explains why an unreachable keyring is not
// a proof failure. It states only what is true on that branch — the
// signature step runs against the bundle's own embedded key regardless
// of its outcome — rather than asserting that the signature verified.
const keyBindingUnestablishedNote = "The signature check reads the bundle's own embedded public key; only that key's binding to Truestamp is unestablished"

// --- Step 2: Structure ---

const expectedVersion = 1

// verifyVersion is the only unconditional Structure row, and deliberately
// so. E.4's hex sweep (verifyHexEncoding, below) is the group's only other
// writer, and it stays silent unless a field breaks the lowercase rule.
//
// E.8's version check is the one structural proposition the appendix asks a
// report to carry, and E.25's containment rule forbids a `pass` row for a
// step D.4 does not report. Four checks that used to live beside it were
// removed rather than demoted, because each was either dead or duplicated:
//
//   - a registered `t` and a non-empty `cx` are E.6 hard rejections applied
//     at parse, so neither row could ever have emitted anything but `pass`
//     by the time it ran — a step asserting a success it cannot fail to
//     achieve, which is what E.22's "never read as a positive assertion"
//     rule exists to prevent, seen from the other side;
//   - `b.id` and `b.mr` presence is E.14's own five-field precondition and
//     is reported there, so a bundle missing either produced two fail rows
//     for one defect.
func verifyVersion(r *Report, bundle *proof.ProofBundle) {
	r.check(groupStructure, CatStructural, bundle.Version == expectedVersion,
		fmt.Sprintf("Proof version %d (expected %d)", bundle.Version, expectedVersion),
		fmt.Sprintf("Unsupported proof format version %d (expected %d)", bundle.Version, expectedVersion))
}

// --- Step 2a: Hex Field Encoding (E.4) ---

// verifyHexEncoding is E.4's lowercase-hex sweep: it grades the ten
// hex-encoded wire fields the appendix enumerates and emits a row only
// when one breaks the rule. A conforming bundle produces no step here at
// all — deliberately not a `pass`, because a `pass` row D.4 does not
// carry is itself nonconformant under E.25 — so the D.4 report is
// unchanged. A non-conforming one gets one Structural failure naming
// every offender in wire order.
//
// E.4 fixes the field list as a closed set, and it is exactly the set E.3
// files as CBOR byte strings less `pk` and `sig` (byte strings in CBOR,
// base64 in JSON): s.mh, s.kid, b.ph, b.mr, b.mh, b.kid, cx[].memo,
// cx[].op, cx[].tx, cx[].bmr. Do not add to it or remove from it here
// without the appendix moving first.
//
// Graded, not rejected: E.6 gates on the presence and type of structure,
// never on the lexical form of a value, and says so explicitly. This puts
// hex case with the other encoding failures E already grades at the point
// of use — a `pk` that does not decode to 32 bytes fails E.9, an `ip` that
// is not valid base64url fails E.12. A hard rejection would produce no
// step results at all, leaving an operator holding a merely mis-encoded
// bundle knowing only that it was refused, not which field to fix.
//
// Both reporting points are required by E.4, not merely permitted. This
// sweep is one of them; the other is the consuming step failing when a
// decoder refuses a value at its point of use. The sweep is what reaches
// cx[].tx and cx[].bmr, which no verifier decodes — they are string-
// compared against values derived from rtx and txp, so without a rule here
// they are the only fields where an uppercase spelling still reaches a
// passing report. The point-of-use failure is what keeps a generic root
// mismatch from being reported when the real defect is the encoding.
//
// The exclusions are normative too, and applying the rule outside the ten
// rejects bundles Truestamp legitimately emits:
//
//   - `pk`/`sig` (base64) and `ip`/`ep` (base64url) are case-significant
//     alphabets; case MUST be preserved exactly.
//   - `cx[].rtx` and `cx[].txp` carry EITHER base64url or hex (E.3, E.5),
//     and the hex alphabet is a subset of the base64url one, so a verifier
//     cannot tell which it holds: the rule is not well defined for them and
//     MUST NOT be applied. Their case has no consequence either — neither
//     is in E.16's signed payload, and both are decoded only so the values
//     derived from them can be compared against tx, op and bmr, which the
//     rule does cover. Pinned by TestCLI_RawTxAndTxOutProof_CaseIsNotGraded.
//   - `s.id`, `b.id`, `net`, `ts` are not hashes, and a ULID is uppercase
//     Crockford Base32 by construction. Case in an id is already bound
//     cryptographically: ids enter their preimage as UTF-8 bytes under
//     len32 rather than being hex-decoded, so a re-cased id derives a
//     different subject or block hash and fails E.13/E.14 on its own.
//   - Values inside `s.d`, including an Item's `s.d.hash`, are canonicalized
//     verbatim. Normalizing their case would derive a different 0x11 or
//     0x21 digest and report a valid proof as forged; case there is bound
//     by the JCS digest. The one exception is E.7's comparison of a
//     caller-supplied expected hash against s.d.hash, which MAY be
//     case-insensitive because that argument is an operator's typed input
//     rather than a wire field — see [tscrypto.HexEqual].
func verifyHexEncoding(r *Report, bundle *proof.ProofBundle) {
	var offenders []string
	check := func(name, value string) {
		if err := tscrypto.ValidateLowercaseHex(value); err != nil {
			offenders = append(offenders, fmt.Sprintf("%s: %s", name, err))
		}
	}

	if bundle.Subject != nil {
		check("s.mh", bundle.Subject.MetadataHash)
		check("s.kid", bundle.Subject.SigningKeyID)
	}
	check("b.ph", bundle.Block.PreviousBlockHash)
	check("b.mr", bundle.Block.MerkleRoot)
	check("b.mh", bundle.Block.MetadataHash)
	check("b.kid", bundle.Block.SigningKeyID)
	for i, cx := range bundle.Commitments {
		check(fmt.Sprintf("cx[%d].tx", i), cx.TransactionHash)
		check(fmt.Sprintf("cx[%d].memo", i), cx.MemoHash)
		check(fmt.Sprintf("cx[%d].op", i), cx.OpReturn)
		check(fmt.Sprintf("cx[%d].bmr", i), cx.BlockMerkleRoot)
	}

	if len(offenders) == 0 {
		return
	}
	r.fail(groupStructure, CatStructural, withCode(codeInvalidHexEncoding, fmt.Sprintf(
		"Hash fields do not carry E.4's required lowercase-hex encoding — %s",
		strings.Join(offenders, "; "))))
}

// codeInvalidHexEncoding is Appendix E.23's identifier for a value that is
// present and of the right type but not lowercase hex. E.4 makes carrying
// it a MUST on every such failure — the sweep above and the point-of-use
// refusals in E.10, E.14 and E.15 alike — because the identifier is what
// lets two independent verifiers be compared on the same bundle without
// diffing prose that differs between them.
//
// It is deliberately NOT part of proof.RejectionError's taxonomy, which
// carries E.6's hard rejections and surfaces under --json as
// {"result":"rejected", …}. This one is a step fail: the run continues,
// produces a full report, and the identifier rides in the step's message
// the way the reference verifier emits it.
const codeInvalidHexEncoding = "invalid_hex_encoding"

// withCode prefixes a step message with the E.23 identifier the failure
// carries.
func withCode(code, msg string) string { return code + ": " + msg }

// hexEncodingAware prefixes msg with E.23's invalid_hex_encoding identifier
// when err is E.4's lowercase-hex refusal, and returns msg untouched
// otherwise. The point-of-use sites share one error path for every decode
// failure — a short hash, an odd-length field, a non-hex byte — and only
// the encoding case is what E.23 names.
func hexEncodingAware(err error, msg string) string {
	if errors.Is(err, tscrypto.ErrNotLowercaseHex) {
		return withCode(codeInvalidHexEncoding, msg)
	}
	return msg
}

// --- Step 3: Claims Hash ---

func deriveClaimsHash(r *Report, rawClaims json.RawMessage) string {
	if len(rawClaims) == 0 || string(rawClaims) == "null" {
		// E.10: any missing input fails this step. Reporting it as a
		// skip would say the derivation did not apply, when in fact it
		// applied and could not be performed. The result names no hash,
		// so E.22's Data Integrity exception does not reach it.
		r.fail(groupSubjectData, CatCryptographic,
			"Subject data s.d is absent, so no claims hash can be derived")
		return ""
	}

	canonical, oversized, err := jcs.Canonicalize(rawClaims)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic, fmt.Sprintf("Claims JCS failed: %s", err))
		return ""
	}

	hash := tscrypto.BytesToHex(tscrypto.DomainHash(tscrypto.PrefixItemClaims, canonical))
	r.pass(groupSubjectData, CatCryptographic, "Claims hash derived (0x11)")
	reportNumberPortability(r, oversized, "claims hash")
	return hash
}

// reportNumberPortability emits E.4's non-portability warn. It runs only
// on the successful-derivation path and changes no other step's status:
// the digest this run derived is the one the producer signed, so the
// inclusion walk and the signature genuinely verify. What a reader needs
// to know is that a strict RFC 8785 implementation, which rounds every
// number through an IEEE-754 double, would derive a different one.
func reportNumberPortability(r *Report, oversized []string, hashName string) {
	if len(oversized) == 0 {
		return
	}
	r.warn(groupSubjectData, CatCryptographic, fmt.Sprintf(
		"Subject data is not portably verifiable: s.d carries %d integer(s) outside the exactly representable range (e.g. %s); a verifier that parses numbers into IEEE-754 doubles per RFC 8785 will derive a different %s",
		len(oversized), oversized[0], hashName))
}

// --- Step 4: Claims Hash Type ---

func validateClaimsHashType(r *Report, claims Claims) {
	if claims.Hash == "" || claims.HashType == "" {
		return
	}
	if err := tscrypto.ValidateClaimsHash(claims.Hash, claims.HashType); err != nil {
		r.warn(groupSubjectData, CatDataIntegrity, fmt.Sprintf("Claims hash validation: %s", err))
	} else {
		// Info, not pass. E.11's shape check inspects a value the proof
		// carries rather than reproducing a derivation, and Appendix D.4
		// carries it as an additive info row with no row of its own in
		// the reference report. Emitting it as a pass would add a pass
		// row E.25's containment check does not allow. The category is
		// Data Integrity on both arms — E.22's Subject Data exception
		// files a result that names a hash without completing a
		// derivation there whatever its status.
		r.info(groupSubjectData, CatDataIntegrity,
			fmt.Sprintf("Claims hash well formed for %s", claims.HashType))
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
	// Unreachable through Run: E.6 hard-rejects a non-block-like bundle
	// whose `s` is absent or is not a map, so the parser never hands
	// this function a nil subject. It stays because it is a crash guard
	// on an exported-in-package helper the tests drive directly, not an
	// assertion about the bundle — deleting it would trade a graded
	// failure for a nil dereference. Same for the nil guards in
	// deriveObservationHash, verifyClaimsTimestamp and
	// verifySubjectTemporalWindow.
	if subject == nil {
		r.fail(groupSubjectData, CatCryptographic, "Cannot derive item hash: no subject is present")
		return ""
	}
	if claimsHash == "" {
		// E.22: a Subject Data failure that names no hash stays under
		// Cryptographic. Nothing was derived because the data hash it
		// consumes was never derived either.
		r.fail(groupSubjectData, CatCryptographic, "Cannot derive item hash: no claims hash was derived")
		return ""
	}
	if unusable := unusableFields(
		namedField{"s.id", subject.ID}, namedField{"s.mh", subject.MetadataHash},
		namedField{"s.kid", subject.SigningKeyID},
	); len(unusable) > 0 {
		// E.22's Subject Data exception: this result names the composite
		// subject hash and reports no completed derivation, which is one
		// of exactly two cases the appendix files under Data Integrity.
		// s.id is part of the predicate because it is an input to the
		// 0x13 preimage — without it ComputeItemHash frames len32(0) and
		// hashes a short preimage rather than failing.
		r.fail(groupSubjectData, CatDataIntegrity, fmt.Sprintf(
			"Cannot derive item hash: no usable value for %s (E.10's composite subject hash requires s.id, s.mh and s.kid)",
			strings.Join(unusable, ", ")))
		return ""
	}
	// Same width rule as E.14's block preimage: E.10 quotes the item
	// path as 111 bytes for a 26-character ULID id, and len32 frames a
	// short s.mh or s.kid without complaint.
	if size, ok := framedPreimageSize(subject.ID, claimsHash, subject.MetadataHash, subject.SigningKeyID); ok && size != itemPreimageBytes {
		r.fail(groupSubjectData, CatDataIntegrity, fmt.Sprintf(
			"Cannot derive item hash: E.10 frames a %d-byte preimage from s.id, the claims hash, s.mh and s.kid, this subject frames %d (a field is present but the wrong width)",
			itemPreimageBytes, size))
		return ""
	}

	hash, err := tscrypto.ComputeItemHash(subject.ID, claimsHash, subject.MetadataHash, subject.SigningKeyID)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic,
			hexEncodingAware(err, fmt.Sprintf("Item hash computation failed: %s", err)))
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

	// The walk's root comparison folds ASCII case, so without this guard a
	// b.mr spelled in uppercase would derive the same root and this step
	// would report `pass` — asserting a match against a value the bundle
	// does not carry in the encoding E.4 requires, and disagreeing with the
	// reference verifier, whose secure_equal?/2 is a raw binary compare and
	// grades the same bundle a failure here.
	if err := tscrypto.ValidateLowercaseHex(block.MerkleRoot); err != nil {
		r.fail(groupInclusion, CatCryptographic, withCode(codeInvalidHexEncoding, fmt.Sprintf(
			"Cannot verify inclusion proof: b.mr is %s", err)))
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

	// A block that carries no usable id is not named. The walk still
	// establishes what it establishes — the derived root equals b.mr —
	// but "Inclusion proof to block  (2 steps)" names an absent block,
	// which E.22 forbids a message from doing.
	passMsg := fmt.Sprintf("Inclusion proof to the block merkle root (%d steps); the block carries no usable id to name it by", len(proofList))
	if block.ID != "" {
		passMsg = fmt.Sprintf("Inclusion proof to block %s (%d steps)", block.ID, len(proofList))
	}

	// The fail wording is Appendix E.13's, quoted: independent verifiers
	// must name this failure the same way (E.23).
	r.check(groupInclusion, CatCryptographic, valid, passMsg,
		"Inclusion proof derived root does not match block merkle root")
}

// --- Step 8: Block Hash Derivation ---

func deriveBlockHash(r *Report, block proof.Block) string {
	// E.14: all five inputs MUST be present and the preimage is 157
	// bytes unconditionally — there is no nil branch. Without this guard
	// an absent field is framed as len32(0), ComputeBlockHash returns a
	// digest over a short preimage with no error, and the step asserts a
	// derivation it never performed. Returning "" instead routes the
	// dependent epoch walks and the signature step to their "cannot
	// verify" arms, which is what E.16 requires of an upstream
	// derivation failure.
	//
	// "no usable value", not "missing": a field the bundle carries with
	// the wrong JSON/CBOR type reaches this step as the zero value too,
	// and a message reading "missing" would be false about a field the
	// file demonstrably has.
	if unusable := unusableFields(
		namedField{"b.id", block.ID}, namedField{"b.ph", block.PreviousBlockHash},
		namedField{"b.mr", block.MerkleRoot}, namedField{"b.mh", block.MetadataHash},
		namedField{"b.kid", block.SigningKeyID},
	); len(unusable) > 0 {
		r.fail(groupBlockHash, CatCryptographic, fmt.Sprintf(
			"Cannot derive block hash: no usable value for %s (E.14 requires all five of id, ph, mr, mh, kid)",
			strings.Join(unusable, ", ")))
		return ""
	}

	// E.14 fixes the widths as well as the presence: a 36-character id
	// plus 32/32/32/4 hex-decoded bytes, "157 bytes, unconditionally".
	// len32 frames a short field without complaint, so a present-but-
	// truncated b.mh yields a digest over a 141-byte preimage and the
	// step used to assert E.14's derivation over it.
	if size, ok := framedPreimageSize(block.ID, block.PreviousBlockHash, block.MerkleRoot, block.MetadataHash, block.SigningKeyID); ok && size != blockPreimageBytes {
		r.fail(groupBlockHash, CatCryptographic, fmt.Sprintf(
			"Cannot derive block hash: E.14 frames a %d-byte preimage from the five block fields, this block frames %d (a field is present but the wrong width)",
			blockPreimageBytes, size))
		return ""
	}

	computed, err := tscrypto.ComputeBlockHash(block.ID, block.PreviousBlockHash, block.MerkleRoot, block.MetadataHash, block.SigningKeyID)
	if err != nil {
		r.fail(groupBlockHash, CatCryptographic,
			hexEncodingAware(err, fmt.Sprintf("Block hash computation failed: %s", err)))
		return ""
	}

	r.pass(groupBlockHash, CatCryptographic, "Block hash derived (0x32)")
	return computed
}

// Preimage widths fixed by the appendix: E.14's block hash (0x32) and
// E.10's two composite subject hashes (0x13 over a 26-character ULID,
// 0x23 over a 36-character UUIDv7). Each is quoted in the appendix as a
// single unconditional number, so a preimage of any other size is not the
// derivation the step claims to have performed.
const (
	blockPreimageBytes   = 157 // E.14: 1 + (4+36) + (4+32)*3 + (4+4)
	itemPreimageBytes    = 111 // E.10 item path: 1 + (4+26) + (4+32)*2 + (4+4)
	entropyPreimageBytes = 121 // E.10 entropy path: 1 + (4+36) + (4+32)*2 + (4+4)
)

// framedPreimageSize returns the byte length of the length-prefixed
// preimage tscrypto builds from these fields: one domain-prefix byte,
// then len32 + payload for the id (UTF-8 bytes) and for each hex field
// (decoded bytes). ok is false when a hex field does not decode at all,
// which the compute call reports on its own with a better message.
func framedPreimageSize(id string, hexFields ...string) (int, bool) {
	size := 1 + 4 + len(id)
	for _, f := range hexFields {
		b, err := tscrypto.HexToBytes(f)
		if err != nil {
			return 0, false
		}
		size += 4 + len(b)
	}
	return size, true
}

// namedField pairs a bundle field with the name a report calls it by.
type namedField struct {
	Name  string
	Value string
}

// unusableFields returns the names of the fields that carry no value, in
// the order given — which callers pass in the appendix's own field order,
// so a message lists them the way E.10 and E.14 do. A field is unusable
// whether the bundle omits it or carries it with a type the parser cannot
// read as a string; the report says only that this step found nothing to
// work with, which is what both cases have in common.
func unusableFields(fields ...namedField) []string {
	var out []string
	for _, f := range fields {
		if f.Value == "" {
			out = append(out, f.Name)
		}
	}
	return out
}

// --- Step 10: Epoch Proofs ---

func verifyEpochProofs(r *Report, commits []ExternalCommit, blockHash string) []string {
	if len(commits) == 0 {
		return nil
	}
	if blockHash == "" {
		// E.15: each cx entry produces its own step result. One row for
		// the whole array would report N commitments as a single
		// unverifiable thing and silently drop the per-entry accounting
		// the appendix requires.
		for i, cx := range commits {
			r.fail(groupEpoch, CatCryptographic, fmt.Sprintf(
				"Epoch proof %d (%s): cannot verify, no block hash to walk from", i, ptype.Humanize(cx.Type)))
		}
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
		// Same guard as the E.13 walk: the comparison folds case, so a
		// committed root spelled in uppercase would otherwise pass here and
		// then be fed to the 0x61 payload, where E.16's decode rejects it —
		// two rows disagreeing about the same field. Refuse it once, at the
		// step that owns the field, and contribute no epoch root so the
		// signature step reports what it is actually missing.
		if err := tscrypto.ValidateLowercaseHex(target); err != nil {
			r.fail(groupEpoch, CatCryptographic, withCode(codeInvalidHexEncoding, fmt.Sprintf(
				"Epoch proof %d (%s): cx[%d].%s is %s",
				i, ptype.Humanize(cx.Type), i, epochTargetKey(cx), err)))
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
				i, ptype.Humanize(cx.Type), len(proofList)),
			fmt.Sprintf("Epoch proof %d (%s): derived epoch root does not match the committed value",
				i, ptype.Humanize(cx.Type)))
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

// epochTargetKey returns the wire key [epochTarget] read the value from, so
// a message about that value names the key the bundle actually spells
// rather than the generic word "target".
func epochTargetKey(cx ExternalCommit) string {
	switch cx.Type {
	case ptype.CommitmentStellar:
		return "memo"
	case ptype.CommitmentBitcoin:
		return "op"
	default:
		return "epoch root"
	}
}

// ExternalCommit is a type alias for convenience
type ExternalCommit = proof.ExternalCommit

// --- Step 11: Proof Signature ---

func verifyProofSignature(r *Report, bundle *proof.ProofBundle, pubkeyBytes []byte, keyID, subjectHash, blockHash string, epochRoots []string, opts Options) {
	// Every arm below carries E.16's required phrase verbatim, with the
	// specific missing input appended. E.9 additionally requires that an
	// undecodable `pk` abort this step with an explicit "cannot verify"
	// rather than a silent pass, which is the first case here.
	switch {
	case pubkeyBytes == nil:
		r.fail(groupProof, CatCryptographic,
			"Cannot verify proof signature (missing derived data): no usable public key")
		return
	case subjectHash == "":
		r.fail(groupProof, CatCryptographic,
			"Cannot verify proof signature (missing derived data): no subject hash")
		return
	case blockHash == "":
		r.fail(groupProof, CatCryptographic,
			"Cannot verify proof signature (missing derived data): no block hash")
		return
	}

	for i, er := range epochRoots {
		if er == "" {
			r.fail(groupProof, CatCryptographic, fmt.Sprintf(
				"Cannot verify proof signature (missing derived data): no epoch root %d", i))
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
		// E.25 lists the steps a verifier MAY skip and still call the
		// run verified — the external ones and E.17's keyring
		// cross-check. E.16 is not among them, so a run that omits it
		// must not let its verdict be read as a signature-checked one.
		// The verdict itself is derived from step statuses alone and a
		// skip cannot move it, so the disclosure is carried as a warn:
		// the same bundle exits 1 with "Proof signature invalid
		// (Ed25519)" once the flag is dropped, and nothing else in the
		// report says so.
		r.warn(groupProof, CatCryptographic,
			"This run establishes nothing about who signed the proof: the Ed25519 signature was not checked (--skip-signatures)")
		return
	}

	sigValid, err := tscrypto.VerifyEd25519(proofHashBytes, bundle.Signature, pubkeyBytes)
	if err != nil {
		r.fail(groupProof, CatCryptographic, fmt.Sprintf("Proof signature verification error: %s", err))
		return
	}

	r.check(groupProof, CatCryptographic, sigValid,
		"Proof signature valid (Ed25519)", "Proof signature invalid (Ed25519)")
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
	// E.20's second bullet: this ordering constraint "is asserted by
	// Truestamp ... and MUST NOT be presented as externally verified".
	// Both operands are ids Truestamp minted, so the pass says whose
	// claim it is. The fail renders at millisecond resolution because
	// that is the resolution the comparison above runs at: RFC3339
	// truncates to the second, and a sub-second violation printed the
	// same instant twice — a message asserting X is after X.
	r.check(groupSubmissionWindow, CatTiming,
		ok,
		fmt.Sprintf("%s before committed block (%s), asserted by Truestamp, not externally verified",
			label, blockTime.Format(time.RFC3339)),
		fmt.Sprintf("Submission-window ordering violation: %s at %s is AFTER committed block time %s",
			label, formatMillis(subjectTime), formatMillis(blockTime)))
}

// formatMillis renders an instant at the resolution E.20's ordering check
// compares at. The reference verifier prints the same resolution on the
// same row.
func formatMillis(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// reportSubmissionWindowEdges states which edge of the submission window
// this bundle grounds and which it does not.
//
// E.20's third bullet is a MUST with a negative object: establishing the
// submitted-after edge needs the referenced previous block and its
// entropy leaves, which a bundle does not carry, and "a verifier that
// does not [retrieve them out of band] MUST report the submitted-after
// edge as not established from the bundle, never as established".
// Silence does not satisfy it — a reader of a bundle-only run cannot
// tell an edge the verifier found unestablished from one it never
// considered. Both rows are `info`: neither grades anything, E.25
// permits additive info rows, and the reference verifier emits the same
// pair.
func reportSubmissionWindowEdges(r *Report, commits []proof.ExternalCommit) {
	if ts := earliestCommitmentTimestamp(commits); ts != "" {
		r.info(groupSubmittedBefore, CatTiming, fmt.Sprintf(
			"The earliest commitment names %s; confirming that transaction on chain establishes the submitted-before edge", ts))
	} else {
		r.info(groupSubmittedBefore, CatTiming,
			"Not established from this bundle: no cx entry carries a transaction timestamp, so the submitted-before edge rests on confirming a commitment on chain")
	}
	r.info(groupSubmittedAfter, CatTiming,
		"Not established from this bundle: the submitted-after edge needs the referenced previous block and its entropy leaves, which a bundle does not carry")
}

// earliestCommitmentTimestamp returns the earliest parseable `ts` across
// the cx entries — E.20's "earliest cx transaction carrying an epoch root
// that covers this block". Entries with no timestamp, or one that does
// not parse, contribute nothing rather than sorting first.
func earliestCommitmentTimestamp(commits []proof.ExternalCommit) string {
	var best time.Time
	var bestRaw string
	for _, cx := range commits {
		if cx.Timestamp == "" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, cx.Timestamp)
		if err != nil {
			continue
		}
		if bestRaw == "" || ts.Before(best) {
			best, bestRaw = ts, cx.Timestamp
		}
	}
	return bestRaw
}

// addTemporalInfo records the temporal bracket the report displays and
// emits it as an E.22 Timing step.
//
// The step is required, not decorative: Appendix D.4 carries a Temporal
// Info row and E.25 says no step D.4 reports may be absent from a
// conforming verifier's report. It is deliberately `info` — it asserts
// nothing, it records. The one temporal assertion a bundle supports is
// E.20's ordering check, which is its own Submission Window step.
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

	if parts := temporalInfoParts(r.Temporal); len(parts) > 0 {
		r.info(groupTemporalInfo, CatTiming, "Timeline: "+strings.Join(parts, ", "))
	}
}

// temporalInfoParts renders the bracket's populated edges. An id whose
// embedded milliseconds cannot be read formats as "unknown", which is
// dropped rather than printed — a row reading "committed into a block
// unknown" records nothing, and an empty parts list suppresses the step
// entirely.
func temporalInfoParts(ts TemporalSummary) []string {
	var parts []string
	add := func(label, value string) {
		if value == "" || value == "unknown" {
			return
		}
		parts = append(parts, label+" "+value)
	}
	add("submitted", ts.SubmittedAt)
	add("captured", ts.CapturedAt)
	add("committed into a block", ts.CommittedAt)
	return parts
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
		switch external.Classify(err) {
		case external.OutcomeBadInput:
			// The bundle's own field is unusable, so no lookup was
			// attempted. That is a defect in the bundle, graded like
			// the parse failures above.
			r.fail(groupEntropySource, CatBlockchain,
				fmt.Sprintf("NIST pulse index is unusable: %s", err))
		// There is deliberately no OutcomeMismatch arm here, nor in the
		// two sibling entropy verifiers. GetNISTPulse, GetStellarLedger
		// and GetBitcoinBlockHeader are fetchers: they return
		// BadInputError, MalformedResponseError or a wrapped transport
		// error, and only VerifyStellar and VerifyKeyring construct the
		// *MismatchError / *KeyBindingError that Classify grades as a
		// disagreement. The arms that used to sit here asserted
		// "<source> disagrees with the entropy subject" from a branch
		// that could never run and that performed no comparison — an
		// unreachable assertion is the same defect as a message that
		// overstates its branch, and a later refactor would have
		// trusted it as a safety net. The comparisons that CAN
		// disagree are below, after the fetch.
		case external.OutcomeNotFound:
			// E.21 fails only on a value mismatch. A 404 yields no
			// upstream value to compare, and NIST retires chain
			// indices, so an absent pulse leaves the step unconfirmed.
			r.skip(groupEntropySource, CatBlockchain,
				fmt.Sprintf("Entropy source unconfirmed: NIST Beacon has no pulse %d on chain %d (HTTP 404)", pulseIdx, chainIdx))
		default:
			r.skip(groupEntropySource, CatBlockchain,
				fmt.Sprintf("Entropy source unconfirmed: NIST Beacon unavailable (%s)", err))
		}
		return
	}
	// E.4's constant-time MUST covers every digest comparison, and
	// HexEqual is the primitive both sibling branches already use. A
	// bare `!=` also made the comparison case-sensitive, so a subject
	// carrying the same 64 bytes in the opposite hex case was graded a
	// value mismatch and failed the proof; E.21 fails only on a genuine
	// mismatch.
	if !tscrypto.HexEqual(remote.OutputValue, outputValue) {
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
		switch external.Classify(err) {
		case external.OutcomeBadInput:
			r.fail(groupEntropySource, CatBlockchain,
				fmt.Sprintf("Stellar ledger sequence is unusable: %s", err))
		// No OutcomeMismatch arm: see verifyEntropyNIST.
		case external.OutcomeNotFound:
			// E.21 fails only on a value mismatch. Stellar testnet is
			// reset periodically, so a 404 on an old ledger sequence is
			// expected and must not fail a sound proof.
			r.skip(groupEntropySource, CatBlockchain,
				fmt.Sprintf("Entropy source unconfirmed: ledger %d is not in the %s Horizon history (HTTP 404)",
					stellar.Sequence, external.NetworkLabel(network)))
		default:
			r.skip(groupEntropySource, CatBlockchain,
				fmt.Sprintf("Entropy source unconfirmed: Stellar Horizon unavailable (%s)", err))
		}
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
		// external owns the reason string so the report never dresses a
		// name it does not recognise up as a Bitcoin network that
		// happens to lack a public API.
		r.skip(groupEntropySource, CatBlockchain, fmt.Sprintf(
			"Entropy source unconfirmed: %s", external.BitcoinNetworkSkipReason(network)))
		return
	}
	if err != nil {
		switch external.Classify(err) {
		case external.OutcomeBadInput:
			r.fail(groupEntropySource, CatBlockchain,
				fmt.Sprintf("Bitcoin block hash is unusable: %s", err))
		// No OutcomeMismatch arm: see verifyEntropyNIST.
		case external.OutcomeNotFound:
			// E.21 fails only on a value mismatch. Blockstream can move
			// or retire an endpoint, and a 404 yields no upstream value
			// to compare against.
			r.skip(groupEntropySource, CatBlockchain,
				fmt.Sprintf("Entropy source unconfirmed: Blockstream has no block %s on %s (HTTP 404)", btc.Hash, network))
		default:
			r.skip(groupEntropySource, CatBlockchain,
				fmt.Sprintf("Entropy source unconfirmed: Blockstream unavailable (%s)", err))
		}
		return
	}
	if !tscrypto.HexEqual(remote.Hash, btc.Hash) {
		// The lookup was BY btc.Hash, so a disagreement here is an
		// answer about some other block, not a defect in the bundle:
		// nothing the subject asserts can make a well-behaved endpoint
		// return a different block's header for it. E.22 forbids an
		// uninterpretable upstream answer from failing a proof, so this
		// leaves the step unconfirmed. Tampering with s.d.hash cannot
		// reach this branch — a rewritten hash either 404s or names a
		// real block whose height and time are then compared below.
		r.skip(groupEntropySource, CatBlockchain,
			fmt.Sprintf("Entropy source unconfirmed: Blockstream answered about block %s, not the block %s that was asked for",
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
	// Name the fields this branch actually compared. The block was
	// looked up BY its hash, so `height` and `time` are the only
	// independent operands, and both are optional in s.d — a subject
	// carrying neither used to produce "Bitcoin block <upstream height>
	// confirmed on mainnet", asserting a height the bundle never
	// claimed and reading as a confirmation of it.
	compared := []string{"hash"}
	if btc.Height != 0 {
		compared = append(compared, "height")
	}
	if btc.Time != 0 {
		compared = append(compared, "time")
	}
	if len(compared) == 1 {
		r.pass(groupEntropySource, CatBlockchain, fmt.Sprintf(
			"Bitcoin block %s is on %s; the subject carries no height or time to compare against it", btc.Hash, network))
		return
	}
	r.pass(groupEntropySource, CatBlockchain, fmt.Sprintf(
		"Bitcoin block %s confirmed on %s: the subject's %s match upstream",
		btc.Hash, network, strings.Join(compared, ", ")))
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

func verifyStellarCommitments(r *Report, commits []proof.ExternalCommit, opts Options) {
	for i := range commits {
		if commits[i].Type != ptype.CommitmentStellar {
			continue
		}
		verifySingleStellar(r, &commits[i], opts)
	}
}

// verifySingleStellar grades one t=40 commitment. Like its Bitcoin
// sibling it takes no Truestamp block hash: E.15's epoch walk is what
// binds the block hash to cx.memo, and it has already run.
func verifySingleStellar(r *Report, cx *proof.ExternalCommit, opts Options) {
	// Start unconfirmed. E.19's rule ("MUST otherwise report it as
	// skipped") is a floor for every chain: only a lookup that ran and
	// agreed may upgrade this, so a branch added later that forgets to
	// assign it publishes "unconfirmed" rather than a fabricated
	// confirmation.
	ci := CommitmentInfo{
		Method:        "stellar",
		Network:       cx.Network,
		Ledger:        cx.Ledger,
		TxHash:        cx.TransactionHash,
		CommittedHash: cx.MemoHash,
		ExternalCheck: ExternalSkipped,
		Timestamp:     cx.Timestamp,
	}

	switch {
	case opts.SkipExternal:
		r.skip(groupStellar, CatBlockchain, "External Stellar verification skipped (--skip-external)")

	case cx.TransactionHash == "":
		// E.18: an entry that carries no `tx` to look up MUST be
		// reported as skipped, never failed.
		r.skip(groupStellar, CatBlockchain,
			"Stellar commitment unconfirmed: the entry carries no transaction id to look up")

	default:
		// E.18 resolves the Horizon endpoint itself — "public" selects
		// the public instance, anything else (including an absent
		// `net`) selects testnet — so there is deliberately no
		// unrecognised-network branch here. E.5 forbids the absence of
		// an optional field from failing a sound proof.
		result, err := external.VerifyStellar(cx.TransactionHash, cx.MemoHash, cx.Network, cx.Ledger)
		switch external.Classify(err) {
		case external.OutcomeOK:
			r.pass(groupStellar, CatBlockchain,
				fmt.Sprintf("Transaction %s confirmed on %s (ledger %d)",
					cx.TransactionHash, external.NetworkLabel(cx.Network), result.Ledger))
			ci.ExternalCheck = ExternalConfirmed
			if result.Timestamp != "" {
				ci.Timestamp = result.Timestamp
			}

		case external.OutcomeMismatch:
			// E.23 "Stellar / Bitcoin memo or root mismatch": Horizon
			// answered and disagrees with the bundle.
			r.fail(groupStellar, CatBlockchain,
				fmt.Sprintf("Stellar commitment does not match the chain: %s", err))
			ci.ExternalCheck = ExternalFailed

		case external.OutcomeNotFound:
			// A 404 is only a definitive absence when the entry named
			// the network it was looking in. When `net` is absent or
			// unrecognised the lookup fell through to E.18's testnet
			// default, and a public-network transaction 404s there —
			// failing on that would let a missing optional field sink a
			// sound proof, which E.5 forbids.
			if external.IsDefaultedNetwork(cx.Network) {
				r.skip(groupStellar, CatBlockchain,
					fmt.Sprintf("Stellar commitment unconfirmed: transaction %s is not on the default (testnet) Horizon, and %s",
						cx.TransactionHash, defaultedNetworkReason(cx.Network)))
			} else {
				r.fail(groupStellar, CatBlockchain,
					fmt.Sprintf("Stellar commitment not on chain: transaction %s not found on %s",
						cx.TransactionHash, external.NetworkLabel(cx.Network)))
				ci.ExternalCheck = ExternalFailed
			}

		default:
			// Unavailable, Malformed, BadInput. E.22: a skipped
			// external check MUST NOT fail a proof.
			r.skip(groupStellar, CatBlockchain,
				fmt.Sprintf("Stellar commitment unconfirmed: %s", err))
		}
	}

	r.CommitmentInfos = append(r.CommitmentInfos, ci)
}

// defaultedNetworkReason says WHY E.18's endpoint rule fell through to
// the testnet default. [external.IsDefaultedNetwork] is true for an
// absent `net` and for a present-but-unrecognised one alike, and a
// single message covering both told the reader the entry named no
// network while CommitmentInfo.Network in the same report showed the
// name it did carry.
func defaultedNetworkReason(network string) string {
	if network == "" {
		return "the entry names no network"
	}
	return fmt.Sprintf("the entry names an unrecognised network (%q)", network)
}

// --- Step 15: Bitcoin Commitments ---

func verifyBitcoinCommitments(r *Report, commits []proof.ExternalCommit, opts Options) {
	for i := range commits {
		if commits[i].Type != ptype.CommitmentBitcoin {
			continue
		}
		verifySingleBitcoin(r, &commits[i], opts)
	}
}

// verifySingleBitcoin grades one t=41 commitment.
//
// E.19(b) splits the work in two. Six offline steps run over bytes the
// bundle itself supplies, and they establish the internal consistency of
// those bytes ONLY — a fabricated low-difficulty header over a
// one-transaction tree with any 32-byte OP_RETURN passes all six. A
// mandatory binding step then confirms the recomputed block hash against
// something outside the bundle, and it is the only branch here that may
// report the commitment as externally confirmed.
//
// E.19(c) makes every offline input optional: an absent txp skips steps
// 3-5, an absent rtx skips steps 1-2, an absent bmr skips step 4, and an
// absent tx or h leaves the binding step with nothing to look up. Absence
// is always a skip, never a failure, and every branch falls through to
// the CommitmentInfo append so a thin entry never erases the commitment
// from the report.
//
// The Truestamp block hash is deliberately NOT a parameter: nothing here
// compares against it. The epoch walk (E.15) is what binds the Truestamp
// block hash to cx.op, and it runs before this step. An unread parameter
// carrying a 32-byte value of a different kind is an invitation to
// compare two hashes that have nothing to do with each other.
func verifySingleBitcoin(r *Report, cx *proof.ExternalCommit, opts Options) {
	// Start unconfirmed: only the binding step may upgrade this.
	ci := CommitmentInfo{
		Method:        "bitcoin",
		Network:       cx.Network,
		Height:        cx.BlockHeight,
		TxHash:        cx.TransactionHash,
		CommittedHash: cx.OpReturn,
		Timestamp:     cx.Timestamp,
		ExternalCheck: ExternalSkipped,
	}

	// E.19(c): an entry carrying none of rtx/txp/bmr has no offline
	// Bitcoin evidence at all, and no recomputed header for the binding
	// step either. One skip row reports the whole commitment.
	if cx.RawTxHex == "" && cx.TxoutproofHex == "" && cx.BlockMerkleRoot == "" {
		r.skip(groupBitcoin, CatBlockchain,
			"Bitcoin commitment unconfirmed: the entry carries no raw transaction, txoutproof or block merkle root to check")
		r.CommitmentInfos = append(r.CommitmentInfos, ci)
		return
	}

	verifyBitcoinRawTx(r, cx)

	// Steps 3-6 all read the txoutproof. headerMerkleRoot and
	// headerBlockHash stay empty when it is absent or unparseable, which
	// is what downstream steps key their skips off.
	headerMerkleRoot, headerBlockHash := verifyBitcoinTxOutProof(r, cx)
	ci.BlockHash = headerBlockHash

	// Step 4: the tree root recomputed above MUST equal cx.bmr. The
	// header's own merkle root is the bundle's assertion; bmr is the
	// bundle's second, independent assertion of the same value, and
	// cross-checking them is what makes a swapped header detectable.
	switch {
	case cx.BlockMerkleRoot == "":
		r.skip(groupBitcoin, CatBlockchain,
			"Bitcoin block merkle root cross-check skipped: the entry carries no block merkle root (bmr)")
	case headerMerkleRoot == "":
		r.skip(groupBitcoin, CatBlockchain,
			"Bitcoin block merkle root cross-check skipped: no txoutproof header to compare against")
	default:
		// Both operands are display byte order (E.19(a)), so this is a
		// plain hex comparison and not a reversal.
		r.check(groupBitcoin, CatBlockchain,
			tscrypto.HexEqual(cx.BlockMerkleRoot, headerMerkleRoot),
			"Bitcoin block merkle root matches the txoutproof header",
			fmt.Sprintf("Bitcoin merkle proof root does NOT match cx.bmr (txoutproof header carries %s)", headerMerkleRoot))
	}

	verifyBitcoinBinding(r, cx, &ci, headerBlockHash, opts)

	r.CommitmentInfos = append(r.CommitmentInfos, ci)
}

// verifyBitcoinRawTx runs E.19(b) steps 1 and 2 — extract the OP_RETURN
// payload (MUST equal cx.op) and recompute the txid (MUST equal cx.tx).
// Both read cx.rtx, so E.19(c) skips them together when it is absent.
func verifyBitcoinRawTx(r *Report, cx *proof.ExternalCommit) {
	if cx.RawTxHex == "" {
		r.skip(groupBitcoin, CatBlockchain,
			"OP_RETURN and txid checks skipped: the entry carries no raw transaction (rtx)")
		return
	}

	// Step 1
	if extracted, err := bitcoin.ExtractOpReturn(cx.RawTxHex); err != nil {
		r.fail(groupBitcoin, CatBlockchain, fmt.Sprintf("OP_RETURN extraction failed: %s", err))
	} else {
		r.check(groupBitcoin, CatBlockchain,
			tscrypto.HexEqual(extracted, cx.OpReturn),
			"OP_RETURN extracted from raw transaction matches",
			"OP_RETURN extracted from raw transaction does NOT match cx.op")
	}

	// Step 2
	computedTxid, err := bitcoin.ComputeTxID(cx.RawTxHex)
	switch {
	case err != nil:
		r.fail(groupBitcoin, CatBlockchain, fmt.Sprintf("Txid computation failed: %s", err))
	case cx.TransactionHash == "":
		r.skip(groupBitcoin, CatBlockchain,
			"Txid comparison skipped: the entry carries no transaction id (tx)")
	default:
		r.check(groupBitcoin, CatBlockchain,
			tscrypto.HexEqual(computedTxid, cx.TransactionHash),
			fmt.Sprintf("Transaction %s verified from raw tx", cx.TransactionHash),
			fmt.Sprintf("Recomputed txid does NOT match cx.tx (computed %s)", computedTxid))
	}
}

// verifyBitcoinTxOutProof runs E.19(b) steps 3, 5 and 6: parse the
// txoutproof, walk its partial Merkle tree, confirm cx.tx is in the
// matched set, and recompute the block hash from the 80-byte header.
// Returns the header's merkle root and block hash in display byte order,
// both empty when no header could be recovered. Step 4 (the cx.bmr
// cross-check) is graded by the caller because it can be skipped
// independently.
func verifyBitcoinTxOutProof(r *Report, cx *proof.ExternalCommit) (merkleRoot, blockHash string) {
	if cx.TxoutproofHex == "" {
		r.skip(groupBitcoin, CatBlockchain,
			"Bitcoin merkle proof skipped: the entry carries no txoutproof (txp)")
		return "", ""
	}

	mb, err := bitcoin.DecodeTxOutProof(cx.TxoutproofHex)
	if err != nil {
		// Present but unparseable is a defect in the bundle, not an
		// absence, so it stays a failure.
		r.fail(groupBitcoin, CatBlockchain, fmt.Sprintf("Txoutproof parse failed: %s", err))
		return "", ""
	}

	merkleResult := bitcoin.VerifyPartialMerkleTree(
		mb.Hashes, mb.Flags, mb.Transactions, &mb.Header.MerkleRoot,
	)
	r.check(groupBitcoin, CatBlockchain, merkleResult.Valid,
		"Bitcoin partial merkle tree derives the txoutproof header merkle root",
		"Bitcoin partial merkle proof invalid: the derived root does not match the txoutproof header merkle root")

	// Step 5: placement. An absent cx.tx is nothing to look for, which
	// E.19(c) reports as a skip; a malformed one is a bundle defect.
	// chainhash.NewHashFromStr accepts "" and returns the all-zero hash,
	// so the empty case must be caught before the parse or placement
	// silently fails against a hash no tree contains.
	if cx.TransactionHash == "" {
		r.skip(groupBitcoin, CatBlockchain,
			"Bitcoin merkle proof placement skipped: the entry carries no transaction id (tx)")
	} else if expectedTxid, err := chainhash.NewHashFromStr(cx.TransactionHash); err != nil {
		r.fail(groupBitcoin, CatBlockchain,
			fmt.Sprintf("Bitcoin merkle proof placement failed: cx.tx is not a usable transaction id: %s", err))
	} else {
		txidInMatched := false
		for _, m := range merkleResult.MatchedTxIDs {
			if bitcoin.HashEqual(m, expectedTxid) {
				txidInMatched = true
				break
			}
		}
		r.check(groupBitcoin, CatBlockchain, txidInMatched,
			fmt.Sprintf("Transaction %s is in the txoutproof matched set", cx.TransactionHash),
			"Bitcoin merkle proof does not place cx.tx in the matched transaction set")
	}

	// Step 6
	return mb.Header.MerkleRoot.String(), mb.Header.BlockHash().String()
}

// verifyBitcoinBinding runs E.19(b)'s mandatory binding step: the block
// hash recomputed from the bundle's own header MUST be confirmed against
// something outside the bundle before the commitment may be reported as
// passing. When no confirmation point is available the commitment is
// reported skip, never pass.
func verifyBitcoinBinding(r *Report, cx *proof.ExternalCommit, ci *CommitmentInfo, headerBlockHash string, opts Options) {
	switch {
	case opts.SkipExternal:
		r.skip(groupBitcoin, CatBlockchain, "External Bitcoin verification skipped (--skip-external)")
		return
	case headerBlockHash == "":
		r.skip(groupBitcoin, CatBlockchain,
			"Bitcoin commitment unconfirmed: no block header was recovered to confirm against the chain")
		return
	case cx.BlockHeight <= 0:
		// E.19(c): an absent `h` leaves the binding step with nothing to
		// look up. BlockHeight is a plain int with `omitempty`, so a
		// non-positive value is the only absence signal available —
		// safe, because the genesis block is not a plausible Truestamp
		// commitment height.
		r.skip(groupBitcoin, CatBlockchain,
			"Bitcoin commitment unconfirmed: the entry carries no block height (h) to confirm the header at")
		return
	}

	result, skipped, err := external.VerifyBitcoinBlock(headerBlockHash, cx.Network)
	switch {
	case skipped:
		// external owns the reason: an absent `net`, regtest, and a name
		// this client does not recognise are three different facts, and
		// the old single message reported all three as "no public API
		// for <value>" — asserting that whatever the entry carried was a
		// Bitcoin network.
		r.skip(groupBitcoin, CatBlockchain, fmt.Sprintf(
			"Bitcoin commitment unconfirmed: %s", external.BitcoinNetworkSkipReason(cx.Network)))
		r.info(groupBitcoin, CatBlockchain, bitcoinNetworkDowngradeNote)

	case err != nil:
		switch external.Classify(err) {
		// No OutcomeMismatch arm. VerifyBitcoinBlock returns
		// BadInputError, MalformedResponseError or a wrapped transport
		// error, and grades an answer about a different block as
		// malformed rather than as a disagreement — deliberately, since
		// a mis-addressed upstream answer establishes nothing about the
		// bundle. The arm that used to sit here said "Bitcoin
		// commitment does not match the chain" from a branch that could
		// not run and had compared nothing.
		case external.OutcomeNotFound:
			// VerifyBitcoinBlock only reaches the network for an
			// explicitly named mainnet or testnet, so a 404 is a
			// definitive "this header is not in that chain" rather than
			// an artefact of guessing the network.
			r.fail(groupBitcoin, CatBlockchain,
				fmt.Sprintf("Bitcoin commitment not on chain: block %s not found on %s", headerBlockHash, cx.Network))
			ci.ExternalCheck = ExternalFailed
		default:
			// Unavailable, Malformed, BadInput. E.22: a skipped
			// external check MUST NOT fail a proof.
			r.skip(groupBitcoin, CatBlockchain,
				fmt.Sprintf("Bitcoin commitment unconfirmed: %s", err))
		}

	case result.Height != cx.BlockHeight:
		// E.19(b) binds the header at height `h`. Confirming the hash
		// while ignoring `h` would let a bundle publish any height it
		// liked next to a genuine header.
		r.fail(groupBitcoin, CatBlockchain,
			fmt.Sprintf("Bitcoin block height mismatch: the entry claims height %d, the chain reports %d",
				cx.BlockHeight, result.Height))
		ci.ExternalCheck = ExternalFailed

	default:
		r.pass(groupBitcoin, CatBlockchain,
			fmt.Sprintf("Block %s confirmed on %s (height %d)", headerBlockHash, cx.Network, result.Height))
		ci.ExternalCheck = ExternalConfirmed
		if result.Timestamp != "" {
			ci.Timestamp = result.Timestamp
		}
	}
}

// bitcoinNetworkDowngradeNote discloses that the field which suppressed
// the on-chain lookup is not covered by the 0x61 signature payload. E.5
// forbids guessing a chain from an absent `net` and regtest is a
// legitimate Truestamp network with no public API, so grading cannot
// separate an honest unconfirmable commitment from a mainnet one whose
// `net` was rewritten to suppress a refutation. What the report can do
// is refuse to let the resulting skip read as evidence of anything.
const bitcoinNetworkDowngradeNote = "`net` is not covered by the proof signature, so an unconfirmed commitment is not evidence about any chain: a rewritten `net` suppresses this lookup without disturbing the signature"

// --- Entropy Hash ---

func deriveEntropyHash(r *Report, rawData json.RawMessage) string {
	if len(rawData) == 0 || string(rawData) == "null" {
		r.fail(groupSubjectData, CatCryptographic,
			"Subject data s.d is absent, so no entropy hash can be derived")
		return ""
	}

	canonical, oversized, err := jcs.Canonicalize(rawData)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic, fmt.Sprintf("Entropy JCS failed: %s", err))
		return ""
	}

	hash := tscrypto.ComputeEntropyHash(canonical)
	r.pass(groupSubjectData, CatCryptographic, "Entropy hash derived (0x21)")
	// Entropy payloads are where E.4's hazard actually bites: ledger
	// sequences, nonces and difficulty values routinely exceed 2^53.
	reportNumberPortability(r, oversized, "entropy hash")
	return hash
}

// --- Observation Hash ---

func deriveObservationHash(r *Report, subject *proof.Subject, entropyHash string) string {
	if subject == nil {
		r.fail(groupSubjectData, CatCryptographic, "Cannot derive observation hash: no subject is present")
		return ""
	}
	if entropyHash == "" {
		r.fail(groupSubjectData, CatCryptographic, "Cannot derive observation hash: no entropy hash was derived")
		return ""
	}
	if unusable := unusableFields(
		namedField{"s.id", subject.ID}, namedField{"s.mh", subject.MetadataHash},
		namedField{"s.kid", subject.SigningKeyID},
	); len(unusable) > 0 {
		// Same E.22 Subject Data exception as the item composite: this
		// names the composite subject hash and completed no derivation.
		r.fail(groupSubjectData, CatDataIntegrity, fmt.Sprintf(
			"Cannot derive observation hash: no usable value for %s (E.10's composite subject hash requires s.id, s.mh and s.kid)",
			strings.Join(unusable, ", ")))
		return ""
	}
	if size, ok := framedPreimageSize(subject.ID, entropyHash, subject.MetadataHash, subject.SigningKeyID); ok && size != entropyPreimageBytes {
		r.fail(groupSubjectData, CatDataIntegrity, fmt.Sprintf(
			"Cannot derive observation hash: E.10 frames a %d-byte preimage from s.id, the entropy hash, s.mh and s.kid, this subject frames %d (a field is present but the wrong width)",
			entropyPreimageBytes, size))
		return ""
	}

	hash, err := tscrypto.ComputeObservationHash(subject.ID, entropyHash, subject.MetadataHash, subject.SigningKeyID)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic,
			hexEncodingAware(err, fmt.Sprintf("Observation hash computation failed: %s", err)))
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
