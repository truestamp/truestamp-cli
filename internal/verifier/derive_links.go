// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the links of DERIVE (SPEC.md section 10.2, step 9 (e),
// and section 10.3), in this order:
//
//	| Link             | Rule | Part | Failure | Row of 7.3 | Traced |
//	|------------------|------|------|---------|------------|--------|
//	| LINK_INCLUSION   | each inclusion proof walks from its recomputed leaf to its record's merkle_root, and its tree_size equals the record's leaf_count | the record's merkle_root; the proof's tree_size | derivation_mismatch | inclusion | yes |
//	| LINK_WITNESS     | each witness_hashes entry equals the recomputed hash of its witness | /subject/record/witness_hashes/W | derivation_mismatch | witnesses | yes |
//	| LINK_HEAD        | /witnesses/block/record/height is below /block/record/height | /witnesses/block/record/height | ordering_violation | height | no |
//	| LINK_KEY_EVENT   | the key event at /signing_key_event/block introduces K | that event's public_key | key_event_mismatch | key_event | no |
//	| LINK_CHAIN_ORDER | a Stellar entropy sequence is below the top-level Stellar ledger, and a Bitcoin entropy height at most the top-level Bitcoin height, on the same network | that sequence or height | ordering_violation | chain_order | no |
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
// LINK_WITNESS, in an item, for each name W of
// /subject/record/witness_hashes, from this file's compiled list of names
// (witnessNames), never the names the object holds: the entry equals
// H(0x31, /witnesses/block/record) when W is block, and H(0x20,
// /witnesses/entropy_X) when W is entropy_X.
//
// LINK_HEAD, in an item. A block record that carries metadata.genesis
// counts as height 0 here, whatever it declares, /block included
// (recordHeight, SPEC.md section 10.3 and section 10.6, rule 4). A failing
// row's declared is the height the witness block carries, and its
// recomputed the height, so counted, of /block, which the witness block's
// had to be below.
//
// LINK_KEY_EVENT, when P carries signing_key_event: the 32 octets the
// event's public_key decodes to must equal K's. It fails when they differ,
// in every run; otherwise it is pass when the run's attribution is
// truestamp, and info not_attributable when it is not (section 10.3;
// section 10.6, rule 7). The info row is neither a pass nor a failure: it
// never fails the run, so it never closes the external gate nor makes a
// later row skip prior_failure. Its recomputed is K, as B64U32.
//
// LINK_CHAIN_ORDER pairs each Stellar or Bitcoin entropy object with the
// top-level entry of its chain when their network values are equal; every
// other pair, a /signing_key_event/commitments entry's included, reports no
// row. A failing row's recomputed is the ledger or height it was held to.
// The link compares chain positions only and uses no clock (section 11).
//
// The carried MERKLE_PROOF is read by internal/registry's
// ReadInclusionProof, and a proof it refuses, which SHAPE has held to the
// schema, gives no root, as a walk that fails does (SPEC.md section 5.3);
// each witness hash is read through valueAt by its compiled name; and a
// value SHAPE has held to its class that is not of it (a key that does not
// decode, a height, sequence or ledger that is not a number) is a fault of
// the verifier, returned as an error, which [Derive] reports as
// internal_error.

package verifier

import (
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// inclusionWalk is one inclusion proof Q, the record whose hash is its
// recomputed leaf v, at leafAt, and the record R it walks into.
type inclusionWalk struct {
	q, leafAt, r Pointer
	leaf         [32]byte
}

// linkChecks are the link checks, in the order of SPEC.md section 10.3's
// table. publicKey is the 32 octets of K, and attribution the run's
// (section 10.5).
func linkChecks(proof map[string]any, publicKey [32]byte, attribution string) ([]deriveCheck, error) {
	var out []deriveCheck
	for _, link := range []func() ([]deriveCheck, error){
		func() ([]deriveCheck, error) { return inclusionChecks(proof) },
		func() ([]deriveCheck, error) { return witnessChecks(proof) },
		func() ([]deriveCheck, error) { return headChecks(proof) },
		func() ([]deriveCheck, error) { return keyEventLink(proof, publicKey, attribution) },
		func() ([]deriveCheck, error) { return chainOrderChecks(proof) },
	} {
		checks, err := link()
		if err != nil {
			return nil, err
		}
		out = append(out, checks...)
	}
	return out, nil
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

// LINK_WITNESS

// witnessNames are the names witness_hashes may hold (SPEC.md section
// 3.6): block, and entropy_X for each entropy source, in JCS order.
var witnessNames = func() []string {
	names := []string{"block"}
	for _, source := range layoutSources {
		names = append(names, "entropy_"+source)
	}
	return names
}()

// witnessChecks are LINK_WITNESS's checks, one for each name of
// witnessNames the item record's witness_hashes holds; none outside an
// item.
func witnessChecks(proof map[string]any) ([]deriveCheck, error) {
	if proof["type"] != "item" {
		return nil, nil
	}
	hashesAt := Pointer{Name("subject"), Name("record"), Name("witness_hashes")}
	var out []deriveCheck
	for _, name := range witnessNames {
		part := under(hashesAt, name)
		declared := valueAt(proof, part)
		if declared == nil {
			continue
		}
		var at Pointer
		var recomputed string
		var err error
		if name == "block" {
			at = Pointer{Name("witnesses"), Name("block"), Name("record")}
			recomputed, err = recordHashAt(proof, registry.BlockHash, at)
		} else {
			at = Pointer{Name("witnesses"), Name(name)}
			recomputed, err = recordHashAt(proof, registry.EntropyHash, at)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, equalCheck(part, "witnesses", []Pointer{at}, declared, recomputed, "The witness hash"))
	}
	return out, nil
}

// LINK_HEAD

// headChecks is LINK_HEAD's check, in an item: the witness block's height
// is below that of /block, each counted by recordHeight.
func headChecks(proof map[string]any) ([]deriveCheck, error) {
	if proof["type"] != "item" {
		return nil, nil
	}
	part := Pointer{Name("witnesses"), Name("block"), Name("record"), Name("height")}
	blockHeightAt := Pointer{Name("block"), Name("record"), Name("height")}
	witness, err := recordHeight(proof, Pointer{Name("witnesses"), Name("block"), Name("record")})
	if err != nil {
		return nil, err
	}
	block, err := recordHeight(proof, Pointer{Name("block"), Name("record")})
	if err != nil {
		return nil, err
	}
	return []deriveCheck{orderCheck(part, "height", []Pointer{part, blockHeightAt}, valueAt(proof, part), block, witness < block,
		"The witness block's height is below the height of the block.",
		"The witness block, the head of the chain when the item was submitted, is not below the block that includes the item.")}, nil
}

// LINK_KEY_EVENT

// signingKeyEventKeyAt is the part of LINK_KEY_EVENT, the public_key of the
// key event at /signing_key_event/block.
var signingKeyEventKeyAt = Pointer{Name("signing_key_event"), Name("block"), Name("record"), Name("metadata"), Name("key_event"), Name("public_key")}

// keyEventLink is LINK_KEY_EVENT's check, when P carries
// signing_key_event: the event's key is K, and the row's status is set by
// attribution.
func keyEventLink(proof map[string]any, publicKey [32]byte, attribution string) ([]deriveCheck, error) {
	declared := valueAt(proof, signingKeyEventKeyAt)
	if declared == nil {
		return nil, nil
	}
	text, _ := declared.(string)
	key, err := registry.DecodeB64U32(text)
	if err != nil {
		return nil, fmt.Errorf("the key at %s: %w", signingKeyEventKeyAt.Encode(), err)
	}
	c := newCheck(signingKeyEventKeyAt, "key_event", []Pointer{signingKeyEventKeyAt}, text, registry.EncodeB64U(publicKey[:]),
		"The key event at /signing_key_event/block introduced the key that signed this proof.")
	switch {
	case key != publicKey:
		c.code = specdata.ReasonKeyEventMismatch
		c.message = "The key event at /signing_key_event/block does not introduce the key that signed this proof."
	case attribution != AttributionTruestamp:
		c.code = specdata.ReasonNotAttributable
		c.message = "The key event at /signing_key_event/block introduces the key that signed this proof, but this run " +
			"does not attribute the proof to Truestamp, so it shows no Truestamp key history."
	}
	return []deriveCheck{c}, nil
}

// LINK_CHAIN_ORDER

// chainOrderChecks are LINK_CHAIN_ORDER's checks: each Stellar or Bitcoin
// entropy object P carries, in the layout's order, held to the top-level
// entry of its chain on its network.
func chainOrderChecks(proof map[string]any) ([]deriveCheck, error) {
	var out []deriveCheck
	for _, object := range entropyObjects(proof) {
		if !slices.Contains(chains, object.name) {
			continue
		}
		entryAt := Pointer{Name("commitments"), Name(object.name)}
		entry, ok := valueAt(proof, entryAt).(map[string]any)
		if !ok || !sameValue(entry["network"], object.node["network"]) {
			continue
		}
		var c deriveCheck
		var err error
		if object.name == ChainStellar {
			c, err = chainOrderCheck(proof, under(object.at, "payload", "sequence"), under(entryAt, "ledger"), entry["ledger"],
				func(position, limit strictjson.Number) bool { return position < limit },
				"The Stellar entropy's sequence is below the committed ledger.",
				"The Stellar entropy's sequence is not below the ledger of the top-level Stellar commitment.")
		} else {
			c, err = chainOrderCheck(proof, under(object.at, "payload", "height"), under(entryAt, "height"), entry["height"],
				func(position, limit strictjson.Number) bool { return position <= limit },
				"The Bitcoin entropy's height is at most the committed height.",
				"The Bitcoin entropy's height is above the height of the top-level Bitcoin commitment.")
		}
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// chainOrderCheck is one LINK_CHAIN_ORDER check: the entropy's position at
// part held by holds to bound, the top-level entry's value at boundAt.
func chainOrderCheck(proof map[string]any, part, boundAt Pointer, bound any, holds func(position, limit strictjson.Number) bool,
	passing, failing string) (deriveCheck, error) {
	position, ok := valueAt(proof, part).(strictjson.Number)
	limit, okLimit := bound.(strictjson.Number)
	if !ok || !okLimit {
		return deriveCheck{}, fmt.Errorf("the chain positions at %s and %s are not numbers", part.Encode(), boundAt.Encode())
	}
	return orderCheck(part, "chain_order", []Pointer{part, boundAt}, position, limit, holds(position, limit), passing, failing), nil
}

// orderCheck is an untraced order link at part: a pass with passing when
// holds, and otherwise ordering_violation with failing, its declared the
// value at part and its recomputed the bound it was held to.
func orderCheck(part Pointer, construction string, inputs []Pointer, declared, bound any, holds bool, passing, failing string) deriveCheck {
	c := newCheck(part, construction, inputs, declared, bound, passing)
	if !holds {
		c.code, c.message = specdata.ReasonOrderingViolation, failing
	}
	return c
}
