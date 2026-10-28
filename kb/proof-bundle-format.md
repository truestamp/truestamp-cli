<!--
Copyright (c) 2019-2026 Truestamp, Inc.
SPDX-License-Identifier: Apache-2.0
-->

# Proof format

A Truestamp proof is one JSON document, SPEC.md's `version` 1, and nothing older is recognized. Normative reference: **SPEC.md section 3 (The proof document)** in [`truestamp/truestamp-verification`](https://github.com/truestamp/truestamp-verification), with its schema, `spec/schema.json`, and its value classes, `spec/value-classes.json`. The CLI holds every file of `spec/` as the embedded copy in `internal/specdata` (synced by `task verification-sync`, never edited by hand). This page says where the CLI reads each part of the document; SPEC.md says what each part is, and governs wherever the two differ.

## No layout is recognized

A document that is not exactly the envelope of SPEC.md section 3.1 is the bare rejection `invalid_envelope`, whose `sub_reason` names the first check of SPEC.md section 10.2, step 3 (ENVELOPE), that fails (`document`, `proof`, `signature`, `alg`, `public_key`, `value`), with the message "This file is not a Truestamp proof." (`internal/verifier/gate.go`). No member's name selects a layout, no reason code names an earlier one, and no code path exists for one: a file in another layout is refused by the first check it fails, like any other malformed document. A binary encoding of a proof is not JSON, so the reader refuses it at PARSE (`malformed_json`).

## The envelope

```
{"proof": {...}, "signature": {"alg": "truestamp-ed25519-sha256-jcs", "public_key": K, "value": V}}
```

- `proof` holds all signed content; `signature` is closed and has exactly its three members. `alg` is compared with its literal and never selects behavior. `public_key` and `value` are matched against their pattern classes, B64U32 and B64U64 (SPEC.md section 2.4), before they are decoded.
- The signature is Ed25519 over the 32 bytes of `proof_hash` = SHA-256(0x50 || JCS(proof)) (SPEC.md section 6.1). `proof_hash` is computed from the parsed `proof` value and is never carried, so every formatting of one value verifies alike, and a proof is identified by its `proof_hash`, never by its file or its signature string (SPEC.md section 3.1).
- **Served form.** Truestamp serves JCS(envelope), at most 65,536 octets (MAX_DOCUMENT_OCTETS), as `application/vnd.truestamp.proof+json` with the extension `.proof.json`. `truestamp proofs get` writes the served bytes unchanged, to stdout, to `-o`'s path, or under `--to-file` to `<CODE>.proof.json` (an item, by its item code) or `<uuidv7>.proof.json` (every other subject) (`internal/proof/download.go`). A verifier reads at most 262,144 octets (MAX_READ_OCTETS, rule J1), so a reformatted proof still reads.
- The only other signature a proof carries is a rotation's `handover_signature`, inside its key event (SPEC.md section 3.3). Nothing else in a proof is signed: no block signature, no commitment signature and no signature on an item or an entropy observation.

## Members of `proof`

SPEC.md section 3.2 is the table. In short: `format` (`"truestamp-proof"`), `version` (`1`), `type`, `block` (a BLOCK_NODE: the block that includes the subject, or for `block` and `beacon` the subject itself), `commitments`, and, by type, `subject`, `witnesses` (an item) and `observations` (a beacon); optionally `environment` (present exactly when a non-production environment signed the proof, SPEC.md section 6.4), `constructions` (the text, its reference URL, or absent, SPEC.md section 7) and `signing_key_event` (`{block, commitments}`: the committed block whose key event introduced `signature.public_key`, provenance only). A proof carries no generation time, no issuer, no proof-level key id, no `proof_hash`, no download variant, no ledger or block close time, and no chain of blocks between `/witnesses/block` and `/block`.

Every object is closed (SPEC.md section 2.3): it has exactly the members SPEC.md lists for it. The only open region is `subject.claims`. An optional member with no value is absent, never `null`.

## Types and layouts

A proof's `type` is one of the six names of the schema's closed registry, `item`, `entropy_nist`, `entropy_stellar`, `entropy_bitcoin`, `block` and `beacon` (SPEC.md section 3.2), compared exactly. A type is its name: it has no number, and nothing maps it to one. It is the only member that selects what runs, and only once SHAPE has checked it (SPEC.md section 10.6, rule 3). Every other member that varies, an entropy object's `source`, a key event's `type` or the chain a commitment entry is keyed by, chooses only among the alternatives the schema lists for its position, and the alternative chosen is checked in full. The filename is never consulted.

The layouts (SPEC.md section 3.6):

| Type | Beside the common members |
| ---- | ------------------------- |
| `item` | `subject`: `claims` (SUBMITTED, never checked as true), `record` (`claims_hash`, `id`, `witness_hashes`), `item_hash` = H(0x11, record), and `inclusion_proof` from the recomputed `item_hash` to `/block/record/merkle_root`; `witnesses`: exactly the names of `witness_hashes`, `block` always, each `entropy_X` when that source had a current observation |
| `entropy_nist`, `entropy_stellar`, `entropy_bitcoin` | `subject`: `entropy` (the ENTROPY_OBJECT, its `source` matching the type), `record` (`entropy_hash`, `id`), `observation_hash` = H(0x21, record), and `inclusion_proof` |
| `block` | nothing: the subject is `block.record`, and only the submitted-before edge applies |
| `beacon` | `observations`: for each source with a leaf in the block, the leaf with the latest source time; an empty object leaves the submitted-after edge absent |

Block and beacon proofs of one block carry the same `block`, `commitments` and `signing_key_event`, and differ in `type` (and a beacon's `observations`), which the signature covers, so relabelling one as the other breaks the signature.

## Records and their hashes

Each node is a record and its hash, and no record contains its own hash (SPEC.md section 3.7). A BLOCK_NODE is `{block_hash, record}` with `block_hash` = H(0x31, record) and a record of `height`, `id`, `leaf_count`, `merkle_root`, `metadata` (`{}`, `{"key_event": ...}`, or at genesis `{"genesis": ..., "key_event": ...}`) and `previous_block_hash` and `signing_key_id` (SPEC.md section 3.3). The server stores and sends a block's `height`; `blocks list` shows it as a column and `blocks get` as a row. An ENTROPY_OBJECT is exactly the preimage of its `entropy_hash`, H(0x20, object) (SPEC.md section 3.5). The byte prefixes are SPEC.md section 4's registry, held in `internal/registry` and pinned to `spec/prefixes.json`.

An item's, an observation's and a block's id are each a UUIDv7 in lowercase (the UUID7 class). An item also has an **item code** (SPEC.md section 3.9), written from the leftmost 75 bits of its recomputed `item_hash` and a check symbol: no proof carries it, no step reads it, and it is never evidence (`internal/itemcode`).

## Commitments: one kind, at two levels

A commitment binds a record into the Merkle tree one level up, and there is one kind of it:

- an item's or entropy observation's record into its block: the subject's `inclusion_proof`, walked from the recomputed `item_hash` or `observation_hash` to `/block/record/merkle_root`;
- a block into an epoch: each entry of `commitments`, an object keyed by chain (`stellar`, `bitcoin` or both), whose entry carries the epoch as an EPOCH_NODE (`{epoch_hash, record: {leaf_count, merkle_root}}`, `epoch_hash` = H(0x40, record)) and the `inclusion_proof` of the recomputed `block_hash` in that epoch's tree (SPEC.md section 3.4). The `epoch_hash` is the value written to the chain: a Stellar transaction's hash memo, a Bitcoin transaction's OP_RETURN output.

`/commitments` holds the commitments of `/block`, and `/signing_key_event/commitments` those of the key-event block, with the same shape at both. Only `/commitments` feeds the submitted-before edge. Every member of an entry is required: a Stellar entry adds `ledger`, `network` and `transaction_hash`; a Bitcoin entry, present only once final, adds `header_hash`, `header_merkle_root`, `header_time`, `height`, `network`, `raw_transaction`, `transaction_hash` and `txoutproof`, whose consistency DERIVE checks offline (SPEC.md section 10.2, step 9 (d), `internal/verifier/bitcoin`). Nothing in a commitment is signed, and a block's commitment to a public chain is not a second kind of commitment.

## Value classes and pattern classes

Every value outside `subject.claims` matches the pattern class SPEC.md section 2.4 gives its member (HEX64, KID, B64U32, B64U64, TIME, UUID7, INT and the rest), checked at SHAPE, so a hash is lowercase hex because its class says so, and an uppercase one fails SHAPE's row for its member. The NIST `outputValue` is NISTHEX, uppercase as NIST publishes it. Each value is also treated as its value class says (SPEC.md section 3.7): PINNED values are compared with pinned values before anything uses them, CHECKED values are recomputed and compared, CHAIN POINTER values support a pass only after a read made in the run, and RECORDED values are never a verdict input. `truestamp inspect` prints every member by its class (`internal/verifier/inspect.go`), and `cmd/inspect_test.go` holds those classes to `spec/value-classes.json`.

## Size

SPEC.md section 3.8 gives the typical and worst sizes of each type in each constructions mode; every committed subject has a proof under MAX_DOCUMENT_OCTETS.
