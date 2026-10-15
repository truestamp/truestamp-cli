// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports LINK_INCLUSION of the reference verifier's
// Truestamp.Verifier.Derive.Links (elixir/lib/truestamp/verifier/derive/links.ex
// in truestamp-verification), checked against the Truestamp application's
// port, Truestamp.Proof.Verifier.Derive.Links
// (lib/truestamp/proof/verifier/derive/links.ex in truestamp-v2): the first
// link of DERIVE (SPEC.md section 10.2, step 9 (e), and section 10.3).
// LINK_WITNESS, LINK_HEAD, LINK_KEY_EVENT and LINK_CHAIN_ORDER join it
// after, in that order, with the key-event checks.
//
//	| Link           | Rule | Part | Failure | Row of 7.3 | Traced |
//	|----------------|------|------|---------|------------|--------|
//	| LINK_INCLUSION | each inclusion proof walks from its recomputed leaf to its record's merkle_root, and its tree_size equals the record's leaf_count | the record's merkle_root; the proof's tree_size | derivation_mismatch | inclusion | yes |
//
// LINK_INCLUSION covers every inclusion proof Q, at both tree levels
// (SPEC.md section 5.4), in the document order of Q, and gives two checks
// for each:
//
//  1. the root, at R/merkle_root, where the walk of section 5.3 from the
//     recomputed leaf v must succeed and give R's merkle_root (recomputed
//     is nil when it gives no root);
//  2. the size, at Q/tree_size, which must equal R's leaf_count whether or
//     not the walk succeeded (recomputed is that count, the committed leaf
//     count of section 5.4).
//
//	| Q                                                 | v, recomputed                                   | R                                            |
//	|---------------------------------------------------|-------------------------------------------------|----------------------------------------------|
//	| /subject/inclusion_proof, in an item              | H(0x11, /subject/record)                        | /block/record                                |
//	| /subject/inclusion_proof, in an entropy type      | H(0x21, /subject/record)                        | /block/record                                |
//	| /observations/entropy_X/inclusion_proof           | H(0x21, /observations/entropy_X/record)         | /block/record                                |
//	| /commitments/C/inclusion_proof                    | H(0x31, /block/record)                          | /commitments/C/epoch/record                  |
//	| /signing_key_event/commitments/C/inclusion_proof  | H(0x31, /signing_key_event/block/record)        | /signing_key_event/commitments/C/epoch/record |
//
// The walk is internal/registry's Walk, RFC 9162 section 2.1.3.2, with at
// most MAX_PATH_STEPS path nodes; the leaf and node bytes 0x00 and 0x01 are
// used only inside it (SPEC.md section 10.6, rule 2). Every walk starts
// from a value this run recomputed, never from the one P carries (rule 4),
// so a graft fails where section 10.3 says: v is recomputed under a
// registry constant this file names at its call, the hash each record's
// own check makes.
//
// Adapted from the reference: the carried MERKLE_PROOF is read by
// internal/registry's ReadInclusionProof, and a proof it refuses, which
// SHAPE has held to the schema, gives no root, as a walk that fails does.

package verifier

import (
	"encoding/hex"

	"github.com/truestamp/truestamp-cli/internal/registry"
)

// inclusionWalk is one inclusion proof Q, the record whose hash is its
// recomputed leaf v, at leafAt, and the record R it walks into.
type inclusionWalk struct {
	q, leafAt, r Pointer
	leaf         [32]byte
}

// linkChecks are the link checks, in the order of SPEC.md section 10.3's
// table: LINK_INCLUSION, so far.
func linkChecks(proof map[string]any) ([]deriveCheck, error) {
	return inclusionChecks(proof)
}

// inclusionChecks are LINK_INCLUSION's two checks of each inclusion proof,
// in the document order of the proofs.
func inclusionChecks(proof map[string]any) ([]deriveCheck, error) {
	walks, err := inclusionWalks(proof)
	if err != nil {
		return nil, err
	}
	SortBy(walks, func(w inclusionWalk) Pointer { return w.q })
	var out []deriveCheck
	for _, w := range walks {
		out = append(out, inclusionCheck(proof, w)...)
	}
	return out, nil
}

// inclusionWalks are the inclusion proofs of P: the subject's or the
// observations' by type, into /block/record, and each commitment entry's,
// into its epoch record.
func inclusionWalks(proof map[string]any) ([]inclusionWalk, error) {
	var out []inclusionWalk
	block := Pointer{Name("block"), Name("record")}
	subject := Pointer{Name("subject")}
	typ := proof["type"]
	switch {
	case typ == "item":
		w, err := inclusionWalkOf(proof, registry.ItemHash, subject, block)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	case typ == "beacon":
		for _, source := range layoutSources {
			at := Pointer{Name("observations"), Name("entropy_" + source)}
			if _, present := valueAt(proof, at).(map[string]any); !present {
				continue
			}
			w, err := inclusionWalkOf(proof, registry.ObservationHash, at, block)
			if err != nil {
				return nil, err
			}
			out = append(out, w)
		}
	default:
		for _, source := range layoutSources {
			if typ != "entropy_"+source {
				continue
			}
			w, err := inclusionWalkOf(proof, registry.ObservationHash, subject, block)
			if err != nil {
				return nil, err
			}
			out = append(out, w)
		}
	}
	for _, entry := range commitmentEntries(proof) {
		// The block beside the commitments: /block for /commitments/C, and
		// /signing_key_event/block for /signing_key_event/commitments/C.
		beside := under(entry.at[:len(entry.at)-2], "block", "record")
		leaf, err := registry.RecordHash(registry.BlockHash, recordAtOf(proof, beside))
		if err != nil {
			return nil, err
		}
		out = append(out, inclusionWalk{q: under(entry.at, "inclusion_proof"), leafAt: beside, r: under(entry.at, "epoch", "record"), leaf: leaf})
	}
	return out, nil
}

// inclusionWalkOf is the inclusion proof of the leaf holder at at, whose
// record at at/record is hashed under kind for v, walking into r.
func inclusionWalkOf(proof map[string]any, kind registry.Construction, at, r Pointer) (inclusionWalk, error) {
	leafAt := under(at, "record")
	leaf, err := registry.RecordHash(kind, recordAtOf(proof, leafAt))
	if err != nil {
		return inclusionWalk{}, err
	}
	return inclusionWalk{q: under(at, "inclusion_proof"), leafAt: leafAt, r: r, leaf: leaf}, nil
}

// recordAtOf is the object at at, or nil, which the registry refuses.
func recordAtOf(proof map[string]any, at Pointer) map[string]any {
	record, _ := valueAt(proof, at).(map[string]any)
	return record
}

// inclusionCheck is w's two checks: the root, then the size.
func inclusionCheck(proof map[string]any, w inclusionWalk) []deriveCheck {
	inclusion, _ := valueAt(proof, w.q).(map[string]any)
	record, _ := valueAt(proof, w.r).(map[string]any)
	return []deriveCheck{
		equalCheck(under(w.r, "merkle_root"), "inclusion", []Pointer{w.q, w.leafAt}, record["merkle_root"], walkRoot(w.leaf, inclusion),
			"The merkle_root the inclusion proof at "+w.q.Encode()+" walks to"),
		equalCheck(under(w.q, "tree_size"), "inclusion", []Pointer{under(w.r, "leaf_count")}, inclusion["tree_size"], record["leaf_count"],
			"The tree_size"),
	}
}

// walkRoot is the root the walk of SPEC.md section 5.3 gives from leaf
// along inclusion, as HEX64, or nil when the walk fails.
func walkRoot(leaf [32]byte, inclusion map[string]any) any {
	p, err := registry.ReadInclusionProof(inclusion)
	if err != nil {
		return nil
	}
	root, err := registry.Walk(leaf, p)
	if err != nil {
		return nil
	}
	return hex.EncodeToString(root[:])
}
