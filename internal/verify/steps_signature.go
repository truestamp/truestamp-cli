// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/truestamp/truestamp-cli/internal/external"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// --- [E.16] Step 8: signature verification ---
//
// Everything above reproduced values. This is the step that establishes
// somebody committed to them. The payload is built from the DERIVED values,
// not from anything the bundle carries directly, which is why tampering
// anywhere upstream surfaces here:
//
//	offset  size  field
//	     0     1  version    uint8
//	     1     2  type code  uint16 big-endian, the registry integer
//	     3     4  kid        raw, DERIVED from public_key
//	     7     8  ts_ms      uint64 big-endian, milliseconds
//	    15    32  subject hash
//	    47    32  block hash
//	    79     2  N          uint16 big-endian, number of epoch roots
//	    81   32N  epoch roots, concatenated, in commitments array order
//
//	proof_hash = SHA-256(0x61 || payload)
//	verify Ed25519 over that 32-byte digest, using public_key

func stepSignature(r *Report, bundle *proof.Bundle, publicKey []byte, keyID, subjectHash, blockHash string, opts Options) {
	switch {
	case publicKey == nil:
		r.fail(groupProof, CatCryptographic,
			"cannot verify proof signature (missing derived data): no usable public key")
		return
	case subjectHash == "":
		r.fail(groupProof, CatCryptographic,
			"cannot verify proof signature (missing derived data): no subject hash")
		return
	case blockHash == "":
		r.fail(groupProof, CatCryptographic,
			"cannot verify proof signature (missing derived data): no block hash")
		return
	}

	tsMs, tsOK := isoMs(bundle.GeneratedAt)
	signature, sigErr := base64.StdEncoding.DecodeString(bundle.Signature)
	if !tsOK || sigErr != nil || bundle.Signature == "" {
		r.fail(groupProof, CatCryptographic,
			"cannot verify proof signature (missing derived data): generated_at or signature is unreadable")
		return
	}

	// The epoch roots are the one input here read straight off the wire,
	// so they are the one input that can refuse to decode.
	roots := make([]string, 0, len(bundle.Commitments))
	for _, c := range bundle.Commitments {
		if hexErr := firstHexError(hexField{name: "epoch root", obj: c.Fields, key: "epoch_merkle_root"}); hexErr != "" {
			r.fail(groupProof, CatCryptographic,
				"invalid_hex_encoding: cannot rebuild the signed payload: "+hexErr)
			return
		}
		roots = append(roots, c.EpochMerkleRoot)
	}

	proofHash, err := tscrypto.BuildCompactProofPayload(
		byte(bundle.Version), uint16(bundle.Code), keyID, uint64(tsMs), subjectHash, blockHash, roots)
	if err != nil {
		r.fail(groupProof, CatCryptographic,
			fmt.Sprintf("cannot verify proof signature (missing derived data): %s", err))
		return
	}
	payloadSize := 81 + 32*len(roots)

	if opts.SkipSignatures {
		r.skip(groupProof, CatCryptographic,
			"proof signature verification skipped (--skip-signatures)")
		// E.25 lists the steps a verifier MAY skip and still call the run
		// verified; E.16 is not among them, so the omission is disclosed
		// as a warn: a skip cannot move the verdict, and nothing else in
		// the report says the signature went unchecked.
		r.warn(groupProof, CatCryptographic,
			"this run establishes nothing about who signed the proof: the Ed25519 signature was not checked (--skip-signatures)")
		return
	}

	if len(signature) != 64 {
		r.fail(groupProof, CatCryptographic,
			fmt.Sprintf("proof signature invalid (Ed25519): signature is %d bytes, expected 64", len(signature)))
		return
	}
	valid, err := tscrypto.VerifyEd25519(proofHash, bundle.Signature, publicKey)
	if err != nil {
		r.fail(groupProof, CatCryptographic, fmt.Sprintf("proof signature invalid (Ed25519): %s", err))
		return
	}
	r.check(groupProof, CatCryptographic, valid,
		fmt.Sprintf("Ed25519 valid over the 0x61 proof hash (%d-byte payload, type code %d, N=%d)",
			payloadSize, uint16(bundle.Code), len(roots)),
		"proof signature invalid (Ed25519)")
}

// --- [E.17] Step 9: binding the key to Truestamp ---
//
// E.16 established that the holder of the private key matching the
// `public_key` IN THIS BUNDLE signed this bundle. That is self-consistent
// by construction: an attacker who fabricates a bundle can sign it with
// their own key and embed their own public key, and every step so far
// still passes. What turns "someone signed this" into "Truestamp signed
// this" is checking the key against an independently obtained keyring.
// Even a successful check establishes only that Truestamp published this
// key: the keyring carries no validity intervals and no revocation flag.

const keyringNotSupplied = "no keyring supplied: this run establishes only that SOME key signed this bundle, not that it is Truestamp's"

func stepKeyring(r *Report, bundle *proof.Bundle, publicKey []byte, keyID string, opts Options) {
	if publicKey == nil {
		r.skip(groupKeyBinding, CatCryptographic, "no public key to check")
		return
	}

	var keyring *external.KeyringResponse
	var source, sourceLabel string
	switch {
	case opts.KeyringFile != "":
		data, err := os.ReadFile(opts.KeyringFile)
		if err == nil {
			keyring, err = external.ParseKeyring(data)
		}
		if err != nil {
			r.skip(groupKeyBinding, CatCryptographic,
				fmt.Sprintf("could not read a {\"keys\": [...]} keyring from %s", opts.KeyringFile))
			return
		}
		source, sourceLabel = "pinned "+opts.KeyringFile, "pinned keyring"
	case opts.SkipExternal || opts.SkipSignatures || opts.KeyringURL == "":
		r.skip(groupKeyBinding, CatCryptographic, keyringNotSupplied)
		return
	default:
		var err error
		keyring, err = external.FetchKeyring(opts.KeyringURL)
		if err != nil {
			// An unreachable or unintelligible keyring establishes
			// nothing either way; E.22 forbids a skipped external check
			// from failing a proof.
			r.skip(groupKeyBinding, CatCryptographic,
				fmt.Sprintf("keyring cross-check not performed: %s", err))
			return
		}
		// E.17 asks for an independently obtained, pinned keyring and
		// treats a live fetch as weaker evidence, so the row says which
		// it was.
		source, sourceLabel = "fetched "+opts.KeyringURL, "keyring fetched from "+opts.KeyringURL
	}
	r.KeyringSource = source

	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)
	if entry, ok := keyring.Find(keyID, publicKeyB64); ok {
		r.pass(groupKeyBinding, CatCryptographic,
			fmt.Sprintf("key_id %s found in the %s (sequence %d, active %t)", keyID, sourceLabel, entry.Sequence, entry.Active))
		return
	}
	r.fail(groupKeyBinding, CatCryptographic,
		fmt.Sprintf("key_id %s is NOT in the %s: this bundle was not signed by a published Truestamp key", keyID, sourceLabel))
}

// --- [E.17] Step 9b: the signing key event ---
//
// The signing key event is a witness of the SIGNATURE rather than of the
// submission, which is why it rides at the top level instead of inside the
// item's metadata. What it adds is independent of the keyring: the keyring
// says "Truestamp publishes this key today"; the key event says "this key
// was introduced by a block in the ledger, and that block was itself
// committed to a public chain". A bundle without one is not defective.

var keyEventTypes = map[string]bool{"genesis": true, "rotation": true, "emergency_rotation": true}

func stepSigningKeyEvent(r *Report, bundle *proof.Bundle, publicKey []byte, keyID string, opts Options) {
	event := bundle.SigningKeyEvent
	if event == nil {
		r.skip(groupSigningKeyEvent, CatCryptographic,
			"no signing key event carried: this bundle does not bind its signing key to a chain-recorded key event")
		return
	}
	if !event.IsMap {
		r.fail(groupSigningKeyEvent, CatCryptographic,
			fmt.Sprintf("signing_key_event_mismatch: signing key event is not a map (%s)", literal(bundle.Fields.Raw("signing_key_event"))))
		return
	}
	if !event.Block.IsMap() {
		r.fail(groupSigningKeyEvent, CatCryptographic,
			"signing_key_event_mismatch: the key event carries no block map")
		return
	}
	keyEvent, ok := event.KeyEvent()
	if !ok || !keyEventTypes[keyEvent.Type] {
		r.fail(groupSigningKeyEvent, CatCryptographic,
			"signing_key_event_mismatch: the carried block's metadata holds no genesis, rotation or emergency_rotation key event")
		return
	}

	// The event only says anything about THIS bundle if the key it
	// introduces is the key that signed the bundle. Both halves are
	// compared: the published key material itself, and the key id derived
	// from it.
	publicKeyB64 := ""
	if publicKey != nil {
		publicKeyB64 = base64.StdEncoding.EncodeToString(publicKey)
	}
	r.check(groupSigningKeyEvent, CatCryptographic,
		publicKeyB64 != "" && keyEvent.PublicKey == publicKeyB64,
		"the key event introduces the public key that signed this bundle",
		"signing_key_event_mismatch: the key event introduces a different public key than the one that signed this bundle")
	r.check(groupSigningKeyEvent, CatCryptographic,
		keyID != "" && keyEvent.KeyID == keyID,
		fmt.Sprintf("key event key_id %s equals the key id derived from the bundle's public key", keyEvent.KeyID),
		"signing_key_event_mismatch: the key event key_id does not equal the key id derived from the bundle's public key")
	r.info(groupSigningKeyEvent, CatCryptographic,
		fmt.Sprintf("key event type %s, sequence %s", keyEvent.Type, literal(keyEvent.Sequence)))

	blockHash, _, reason := blockHashFromMap(event.Block)
	if reason != "" {
		r.fail(groupSigningKeyEvent, CatCryptographic,
			"signing_key_event_mismatch: cannot derive the key event block hash: "+reason)
		return
	}
	r.pass(groupSigningKeyEvent, CatCryptographic,
		"key event block hash derived (0x32) from the carried block map")

	switch {
	case event.CommitmentCount == 0:
		r.skip(groupSigningKeyEvent, CatCryptographic,
			"the key event carries no commitments: the key event block is not shown to be publicly committed")
	case len(event.Commitments) == 0:
		r.fail(groupSigningKeyEvent, CatCryptographic,
			"signing_key_event_mismatch: no key event commitment carries a chain, an epoch proof and an epoch root")
	default:
		for i := range event.Commitments {
			entry := &event.Commitments[i]
			walkEpochProof(r, entry, blockHash, groupSigningKeyEvent, CatCryptographic)
			confirmKeyEventCommitment(r, entry, opts)
		}
	}
}

// confirmKeyEventCommitment runs the unchanged E.18 / E.19 procedures over
// one of the key event's own commitments, reporting under the Signing Key
// Event group.
func confirmKeyEventCommitment(r *Report, entry *proof.Commitment, opts Options) {
	if opts.SkipExternal {
		r.skip(groupSigningKeyEvent, CatCryptographic, fmt.Sprintf(
			"not checked offline: confirm the key event's %s commitment %s in tx %s",
			entry.Chain, short(entry.EpochMerkleRoot), short(entry.TransactionHash)))
		return
	}
	switch entry.Chain {
	case "stellar":
		confirmStellar(r, entry, groupSigningKeyEvent, CatCryptographic)
	case "bitcoin":
		confirmBitcoin(r, entry, groupSigningKeyEvent, CatCryptographic, opts)
	}
}
