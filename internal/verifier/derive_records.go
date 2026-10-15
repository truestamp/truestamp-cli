// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.Derive.Records
// (elixir/lib/truestamp/verifier/derive/records.ex in
// truestamp-verification), checked against the Truestamp application's
// port, Truestamp.Proof.Verifier.Derive.Records
// (lib/truestamp/proof/verifier/derive/records.ex in truestamp-v2): the
// record hashes of DERIVE (SPEC.md section 10.2, step 9), parts (a) to (c),
// each a check that recomputes a value from the preimage P carries and
// compares it with the value P carries for it.
//
// (a) Subject hashes, by type, the only member that selects what runs
// (SPEC.md section 10.6, rule 3):
//
//	| Recomputed                                  | In                | Part                                         | Row of 7.3       |
//	|---------------------------------------------|-------------------|----------------------------------------------|------------------|
//	| H(0x10, /subject/claims)                    | item              | /subject/record/claims_hash                  | claims_hash      |
//	| H(0x11, /subject/record)                    | item              | /subject/item_hash                           | item_hash        |
//	| H(0x20, /subject/entropy)                   | the entropy types | /subject/record/entropy_hash                 | entropy_hash     |
//	| H(0x21, /subject/record)                    | the entropy types | /subject/observation_hash                    | observation_hash |
//	| H(0x20, /observations/entropy_X/entropy)    | beacon            | /observations/entropy_X/record/entropy_hash  | entropy_hash     |
//	| H(0x21, /observations/entropy_X/record)     | beacon            | /observations/entropy_X/observation_hash     | observation_hash |
//
// (b) Block nodes. For each block node B P carries (layout.go's
// blockNodes):
//
//	| Check                                                                       | Part                     | Row of 7.3 |
//	|-----------------------------------------------------------------------------|--------------------------|------------|
//	| H(0x31, B/record) = B/block_hash                                            | B/block_hash             | block_hash |
//	| with B/record/metadata/genesis: B/record/height = 0                          | B/record/height          | genesis    |
//	| with B/record/metadata/genesis: H(0x30, B/record/metadata/genesis) = B/record/previous_block_hash | B/record/previous_block_hash | genesis |
//
// (c) Epochs. For each commitment entry N, H(0x40, N/epoch/record) =
// N/epoch/epoch_hash, at that part (row epoch_hash).
//
// Every prefix byte comes from the compiled registry: each hash is
// internal/registry's RecordHash under a registry constant this file names
// at its call, or under the parameter kind of recordCheck, which every
// caller gives a registry constant (SPEC.md section 10.6, rule 2). Every
// check is traced: each part above is a CHECKED value (section 3.7).
//
// Each block node's checks are followed by those of the key event its
// record carries (section 10.3), which derive_key_events.go makes, so that
// in (b) a block's checks come in the table's order, then its key event's.
// recordHeight is the reference's height/1: the height a block record
// counts as, which the genesis check fixes at 0 and LINK_HEAD compares.
//
// Adapted from the reference: SHAPE has held every preimage to the schema,
// so the registry refuses none in a run; a refusal (a preimage that is not
// an object) is a fault of the verifier, which these functions return as an
// error and [Derive] reports as internal_error.

package verifier

import (
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// recordChecks are the checks of (a), (b) and (c), in that order.
func recordChecks(proof map[string]any) ([]deriveCheck, error) {
	var out []deriveCheck
	for _, part := range []func(map[string]any) ([]deriveCheck, error){subjectChecks, blockNodeChecks, epochChecks} {
		checks, err := part(proof)
		if err != nil {
			return nil, err
		}
		out = append(out, checks...)
	}
	return out, nil
}

// (a)

// subjectChecks are the subject hashes of P's type.
func subjectChecks(proof map[string]any) ([]deriveCheck, error) {
	subject := Pointer{Name("subject")}
	typ := proof["type"]
	if typ == "item" {
		claims, err := recordCheck(proof, under(subject, "record", "claims_hash"), "claims_hash", registry.ClaimsHash, under(subject, "claims"))
		if err != nil {
			return nil, err
		}
		item, err := recordCheck(proof, under(subject, "item_hash"), "item_hash", registry.ItemHash, under(subject, "record"))
		if err != nil {
			return nil, err
		}
		return []deriveCheck{claims, item}, nil
	}
	if typ == "beacon" {
		var out []deriveCheck
		for _, source := range layoutSources {
			at := Pointer{Name("observations"), Name("entropy_" + source)}
			if _, present := valueAt(proof, at).(map[string]any); !present {
				continue
			}
			checks, err := observationChecks(proof, at)
			if err != nil {
				return nil, err
			}
			out = append(out, checks...)
		}
		return out, nil
	}
	for _, source := range layoutSources {
		if typ == "entropy_"+source {
			return observationChecks(proof, subject)
		}
	}
	return nil, nil
}

// observationChecks are the two hashes of the observation at at: its
// entropy object's entropy_hash and its record's observation_hash.
func observationChecks(proof map[string]any, at Pointer) ([]deriveCheck, error) {
	entropy, err := recordCheck(proof, under(at, "record", "entropy_hash"), "entropy_hash", registry.EntropyHash, under(at, "entropy"))
	if err != nil {
		return nil, err
	}
	observation, err := recordCheck(proof, under(at, "observation_hash"), "observation_hash", registry.ObservationHash, under(at, "record"))
	if err != nil {
		return nil, err
	}
	return []deriveCheck{entropy, observation}, nil
}

// recordCheck is one subject hash: the value at part against H(kind, the
// value at preimage), by the constructions row construction.
func recordCheck(proof map[string]any, part Pointer, construction string, kind registry.Construction, preimage Pointer) (deriveCheck, error) {
	recomputed, err := recordHashAt(proof, kind, preimage)
	if err != nil {
		return deriveCheck{}, err
	}
	return equalCheck(part, construction, []Pointer{preimage}, valueAt(proof, part), recomputed, "The "+construction), nil
}

// recordHashAt is H(kind, the value at at) as HEX64, or the registry's
// refusal of it, which names the pointer.
func recordHashAt(proof map[string]any, kind registry.Construction, at Pointer) (string, error) {
	record, _ := valueAt(proof, at).(map[string]any)
	recomputed, err := registry.RecordHashHex(kind, record)
	if err != nil {
		return "", fmt.Errorf("the record at %s: %w", at.Encode(), err)
	}
	return recomputed, nil
}

// (b)

// blockNodeChecks are the checks of each block node P carries, in the
// layout's order.
func blockNodeChecks(proof map[string]any) ([]deriveCheck, error) {
	var out []deriveCheck
	for _, block := range blockNodes(proof) {
		recordAt := under(block.at, "record")
		blockHash, err := recordHashAt(proof, registry.BlockHash, recordAt)
		if err != nil {
			return nil, err
		}
		out = append(out, equalCheck(under(block.at, "block_hash"), "block_hash", []Pointer{recordAt},
			block.node["block_hash"], blockHash, "The block hash"))
		genesis, err := genesisChecks(proof, recordAt)
		if err != nil {
			return nil, err
		}
		keyEvent, err := keyEventChecks(block.at, recordAtOf(proof, recordAt))
		if err != nil {
			return nil, err
		}
		out = append(append(out, genesis...), keyEvent...)
	}
	return out, nil
}

// recordHeight is the height the block record at at counts as where a
// check compares it (SPEC.md section 10.2, step 9 (b), and section 10.3):
// 0 for a record that carries metadata.genesis, whatever it declares, and
// otherwise the height it carries. A record with neither, which SHAPE has
// made impossible, is a fault of the verifier.
func recordHeight(proof map[string]any, at Pointer) (strictjson.Number, error) {
	record := recordAtOf(proof, at)
	metadata, _ := record["metadata"].(map[string]any)
	if _, genesis := metadata["genesis"]; genesis {
		return 0, nil
	}
	height, ok := record["height"].(strictjson.Number)
	if !ok {
		return 0, fmt.Errorf("the block record at %s carries no height", at.Encode())
	}
	return height, nil
}

// genesisChecks are the two checks of a block record at recordAt that
// carries metadata.genesis, and none for one that does not: its height is
// 0, and its previous_block_hash is H(0x30, genesis).
func genesisChecks(proof map[string]any, recordAt Pointer) ([]deriveCheck, error) {
	metadata, _ := valueAt(proof, under(recordAt, "metadata")).(map[string]any)
	if _, present := metadata["genesis"]; !present {
		return nil, nil
	}
	genesisAt := under(recordAt, "metadata", "genesis")
	previous, err := recordHashAt(proof, registry.GenesisPreviousBlockHash, genesisAt)
	if err != nil {
		return nil, err
	}
	inputs := []Pointer{genesisAt}
	return []deriveCheck{
		equalCheck(under(recordAt, "height"), "genesis", inputs, valueAt(proof, under(recordAt, "height")),
			strictjson.Number(0), "The genesis block's height"),
		equalCheck(under(recordAt, "previous_block_hash"), "genesis", inputs, valueAt(proof, under(recordAt, "previous_block_hash")),
			previous, "The genesis block's previous_block_hash"),
	}, nil
}

// (c)

// epochChecks are the epoch_hash of each commitment entry P carries.
func epochChecks(proof map[string]any) ([]deriveCheck, error) {
	var out []deriveCheck
	for _, entry := range commitmentEntries(proof) {
		recordAt := under(entry.at, "epoch", "record")
		epochHash, err := recordHashAt(proof, registry.EpochHash, recordAt)
		if err != nil {
			return nil, err
		}
		part := under(entry.at, "epoch", "epoch_hash")
		out = append(out, equalCheck(part, "epoch_hash", []Pointer{recordAt}, valueAt(proof, part), epochHash, "The epoch hash"))
	}
	return out, nil
}
