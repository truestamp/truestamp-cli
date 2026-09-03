<!--
Copyright (c) 2019-2026 Truestamp, Inc.
SPDX-License-Identifier: MIT
-->

# Proof Bundle Format

Version 1 is the first published Truestamp proof bundle format. Normative reference: **Appendix E of `truestamp-v2/whitepaper/whitepaper.typ`** (E.3 serializations, E.5 the wire key reference, E.6 hard rejections, E.24 the type registry), with the field reference and a full example in `truestamp-v2/kb/verification/proof-bundle-format.md` and the CBOR schema in `truestamp-v2/priv/cddl/proof.cddl`. The model lives in [`internal/proof/types.go`](../internal/proof/types.go); the parsers in `parse.go` (JSON), `cbor_decode.go` and `cbor_encode.go`.

## The draft layout is refused

Version 1 replaced the pre-publication draft layout (short keys `v`, `t`, `s`, `b`, `cx`, `ip`, carried `metadata_hash` and block-hash fields) in place, under the same `version: 1`. A bundle carrying a top-level `v` or `t` key is the hard rejection `unsupported_layout` and the holder is told to regenerate the proof. No verifier keeps a second code path for the old layout, and neither does this CLI: nothing it downloads, inspects or verifies can be trusted unless it speaks the published layout.

## Nothing is opaque

The bundle carries `subject.metadata` and every block's `metadata` (the maps themselves) and carries **no** `metadata_hash` and **no** block hash anywhere. A verifier recomputes every hash from bytes the bundle carries: the subject metadata hash under `0x12` (item) or `0x22` (entropy), each block metadata hash under `0x33`, each block hash under `0x32`, each witness hash under `0x32` or `0x21`. The one hash a bundle does state, `epoch_merkle_root`, is not taken on trust either: E.15 walks the epoch proof to it and E.18/E.19 compare it against the on-chain value. If you find yourself holding a digest you cannot recompute, you have misread the format.

## Top-level keys

| Key | Type | Presence | Rule |
| --- | ---- | -------- | ---- |
| `version` | integer | always | must equal 1 (graded by E.8, not a rejection) |
| `type` | string | always | a registry name: `block`, `beacon`, `item`, `entropy_nist`, `entropy_stellar`, `entropy_bitcoin`; anything else is `invalid_subject_type` |
| `generated_at` | string | always | ISO 8601, whole seconds; supplies `ts_ms` in the signed payload |
| `public_key` | string / bstr | always | padded base64 in JSON, 32 raw bytes in CBOR |
| `signature` | string / bstr | always | padded base64 in JSON, 64 raw bytes in CBOR; Ed25519 over `SHA-256(0x61 || payload)` |
| `subject` | map | absent for `block` and `beacon` | see below |
| `inclusion_proof` | string | absent for `block` and `beacon` | compact proof, base64url without padding, subject hash to `block.merkle_root` |
| `block` | map | always | a block map; the block hash is never carried |
| `block_path` | array of block maps | optional | ordered oldest first; walked in E.17a |
| `commitments` | array | always, non-empty | commitment entries; empty or absent is `no_external_commitments` |
| `signing_key_event` | map | optional | `{block, commitments}`: the key-event block plus its own commitments |

Block-like subjects (`block`, `beacon`) carry no `subject` and no `inclusion_proof`: the block is its own subject, so the subject hash equals the block hash. Carrying either is `unexpected_subject_fields_for_block_like`.

## The subject map

| Key | Presence | Rule |
| --- | -------- | ---- |
| `id` | always | 26-character ULID for `item`, 36-character UUIDv7 for an entropy subject |
| `claims` | `item` only | the claims map, hashed under `0x11` |
| `entropy` | entropy only | the captured source payload map, hashed under `0x21` |
| `metadata` | always | the metadata map itself, hashed under `0x12` (item) or `0x22` (entropy; always `{}`) |
| `signing_key_id` | always | 8 lowercase hex characters, consumed verbatim in the composite preimage |
| `witnesses` | optional, `item` only | witness name to witness detail |

An item's `metadata` is the **witness map** and is never empty (`block` is always present): `{"witnesses": {"block": <hex>, "entropy_stellar": <hex>, "entropy_nist": <hex>, "entropy_bitcoin": <hex>}}`. Each value is the hash of a public record that existed before the submission and that the composite fingerprint commits to. An entropy witness key is present only when that source had a captured observation at submission time; it is absent, never null. The values are lowercase hex **text** in both JSON and CBOR, because the map around them is a JCS preimage.

## Block maps

The top-level `block`, each `block_path` entry, the head block carried as `subject.witnesses.block`, and `signing_key_event.block` all have exactly this shape and derive their hash by one procedure (`blockHashFromMap` in `internal/verify`):

| Key | Rule |
| --- | ---- |
| `id` | 36-character UUIDv7 |
| `previous_block_hash` | 64 lowercase hex (JSON) / 32 bytes (CBOR) |
| `merkle_root` | 64 lowercase hex / 32 bytes |
| `metadata` | the block metadata map, hashed under `0x33`: `{}` for an ordinary block, `{"key_event": {...}}` for a key-event block |
| `signing_key_id` | 8 lowercase hex / 4 bytes |

```text
metadata_hash = SHA-256(0x33 || JCS(metadata))
block_hash    = SHA-256(0x32 || len32(id) || id || len32(ph) || ph || len32(mr) || mr
                          || len32(metadata_hash) || metadata_hash || len32(kid) || kid)
```

The preimage is 157 bytes. The empty-metadata constant `SHA-256(0x33 || "{}")` is `14fe55ee...e582ebe1`; it is derived, never substituted. `block_path`, when present, links the head block to the containing block: `block_path[0].previous_block_hash` is the head block hash, each later entry's is the hash of the entry before it, and `block.previous_block_hash` is the hash of the last entry. It is absent when the head block is the direct predecessor (the common case).

## Commitment entries

| Key | Tier | Chains | Rule |
| --- | ---- | ------ | ---- |
| `chain` | required | both | `"stellar"` or `"bitcoin"` |
| `epoch_merkle_root` | required | both | 64 lowercase hex / 32 bytes: the value written on chain (Stellar memo, Bitcoin `OP_RETURN`) |
| `epoch_proof` | required | both | compact proof, base64url no padding, block hash to epoch root |
| `transaction_hash` | confirmation | both | 64 lowercase hex / 32 bytes, display byte order |
| `network` | optional | both | `public` or `testnet` (Stellar); `mainnet`, `testnet`, `regtest` (Bitcoin) |
| `timestamp` | optional | both | ISO 8601: the Stellar ledger close time or the Bitcoin header time |
| `ledger` | optional | Stellar | ledger sequence, an integer |
| `block_height` | confirmation | Bitcoin | an integer |
| `txoutproof` | optional | Bitcoin | text, hex or base64url: the `CMerkleBlock` |
| `raw_transaction` | optional | Bitcoin | text, hex or base64url: the full raw transaction |
| `block_merkle_root` | optional | Bitcoin | 64 lowercase hex / 32 bytes, display byte order |

Presence tiers are graded by consequence. A missing *required* field is the hard rejection `invalid_commitment_entry`. A missing *confirmation* field means the chain's confirmation step has nothing to look up and reports `skip`, never `fail`. A missing *optional* field narrows what a confirmation can establish and is never a failure. **One root key on both chains**: `chain` is the only field that says which chain an entry belongs to; never dispatch on which root key is present. A present `ledger` or `block_height` that is not a JSON integer is not coerced: the parser records it as unusable and the E.18 confirmation row fails on it (the server grades it the same way).

## The witness registry

| Name | Committed in item metadata | Detail carried | Hash rule | Published time |
| ---- | -------------------------- | -------------- | --------- | -------------- |
| `block` | always | the head block map | block hash (0x32) of the map | UUIDv7 ms of `detail.id` |
| `entropy_stellar` | when captured | the observation's payload map | `SHA-256(0x21 \|\| JCS(payload))` | `detail.closed_at` |
| `entropy_nist` | when captured | the observation's payload map | same | `detail.pulse.timeStamp` |
| `entropy_bitcoin` | when captured | the observation's payload map | same | `detail.time` seconds |
| `signing_key_event` | never (top level) | `{block, commitments}` | block hash recomputed; epoch proofs walked | none |

Payloads travel exactly as captured and are never reshaped (reshaping changes the JCS encoding and therefore the `0x21` hash). Entropy payloads may carry uppercase hex (the NIST `outputValue`) and large integers; both are canonicalized verbatim. The registry is in [`internal/proof/witnesses.go`](../internal/proof/witnesses.go). Extensibility is normative: an unknown name in `metadata.witnesses` or `subject.witnesses` is reported as a `skip` and never fails; names are never renamed or removed.

Witness details are selectable at generation time (`truestamp download --witnesses all|none|<list>`): absent means all (the **complete** bundle), `[]` means none (the **compact** bundle), a list selects a subset (a **partial** bundle). All three are ordinary version 1 bundles; the only difference a verifier sees is how many rows the Witnesses and Submitted After steps can report. Artifact filenames append `-compact` or `-partial`.

## Type registry (frozen; never renumbered)

| Name | Code | Where |
| ---- | ---- | ----- |
| `block` | 10 | `type`; code in the signed payload |
| `beacon` | 11 | `type`; code in the signed payload |
| `item` | 20 | `type`; code in the signed payload |
| `entropy_nist` | 30 | `type`; code in the signed payload |
| `entropy_stellar` | 31 | `type`; code in the signed payload |
| `entropy_bitcoin` | 32 | `type`; code in the signed payload |
| `stellar` | 40 | `commitments[].chain`; code reserved, never on the wire |
| `bitcoin` | 41 | `commitments[].chain`; code reserved, never on the wire |

`type` travels as a name; the signed payload carries the frozen integer. The registry is [`internal/proof/ptype/ptype.go`](../internal/proof/ptype/ptype.go) (`ptype.FromName`, `ptype.Name`). Block and beacon share one wire shape and are distinguished only by the code bound into the signature, so relabelling one as the other without re-signing breaks verification. Read `type` from the bundle, never from the filename (E.24): a file named `truestamp-beacon-<id>.json` may carry `type: block`.

## Hard rejections (E.6), in this order

A hard rejection aborts before any step result exists and produces an error, not a report. The order is normative so two verifiers name the same first defect. Every abort is a `proof.RejectionError` carrying its E.23 identifier, readable via `proof.RejectionCode(err)`, with operator advice from `proof.RejectionAdvice`.

1. input does not decode, or is not a map: `not_a_json_object`
2. (CBOR only) a value outside the JSON value space inside any hashed map: `invalid_subject_data`
3. a top-level `v` or `t` key is present (even null): `unsupported_layout`
4. `type` absent, not a string, or not a registry name: `invalid_subject_type`
5. `block` is not a map: `missing_block`
6. a block-like type carries `subject` or `inclusion_proof`: `unexpected_subject_fields_for_block_like`
7. a non-block-like type is missing `subject`: `missing_subject`
8. a non-block-like type is missing `inclusion_proof` (or it is not a string): `missing_inclusion_proof`
9. `block.metadata` is not a map: `missing_metadata`
10. a non-block-like type whose `subject.metadata` is not a map: `missing_metadata`
11. `commitments` is not a list, or is empty: `no_external_commitments`
12. an entry is not a map, or its `chain` is not `"stellar"` or `"bitcoin"`, or it lacks `epoch_proof` (as a string), or it lacks `epoch_merkle_root`: `invalid_commitment_entry`

One CLI-side rejection sits outside E.6: `verify --type X` on a bundle whose signed `type` differs is `subject_type_mismatch`, the name the server uses for the same condition on `/proof/verify`. It is a rejection rather than a graded row because E.25 forbids adding a `fail` row for a step Appendix D.4 does not report.

Everything else is graded, never rejected: a wrong or absent `version` (E.8), an undecodable `public_key` (E.9), a mis-cased hash (E.4), an undecodable proof (E.12), an unknown optional field (ignored), an unknown witness name (a visible `skip`).

## CBOR

A CBOR bundle is accepted wrapped in the self-describing tag 55799 (`d9 d9 f7`) or as a bare map, definite or indefinite length. `IsCBORProof` detects either by the first byte. Decoding converts the CBOR into the equivalent JSON document ([`CBORToJSON`](../internal/proof/cbor_decode.go)) and then runs the same JSON parser, so the two serializations grade one logical bundle identically. The correspondence is driven by the field's position, not its type:

- `public_key` and `signature` byte strings become padded base64 text;
- every hash slot (`signing_key_id` on the subject and on every block map; every block map's `previous_block_hash` and `merkle_root`; every commitment's `epoch_merkle_root`, `transaction_hash`, `block_merkle_root`, in `commitments` and in `signing_key_event.commitments`) becomes lowercase hex text. A text value in such a slot passes through unchanged, as the server's decoder does, and the E.4 sweep then grades it;
- every map that is a JCS preimage (`subject.claims`, `subject.entropy`, `subject.metadata`, every block `metadata`, every entropy witness payload) is converted strictly within the JSON value space: integers exactly, floats per their IEEE-754 value, text as UTF-8, members in wire order. A byte string, a tag, `undefined`, a simple value other than true/false/null, a non-finite float, a non-text key or invalid UTF-8 inside one is `invalid_subject_data`, naming the path;
- `inclusion_proof`, `epoch_proof`, `txoutproof` and `raw_transaction` are text on the wire (the server never emits them as byte strings). A byte string in one of those slots is not the field and renders as null, so the required two are refused by the E.6 gates the reference verifier applies (`missing_inclusion_proof`, `invalid_commitment_entry`) and the optional two read as absent, which the Bitcoin step reports as a skip;
- everything else keeps its JSON type. Outside a hashed map a byte string renders as hex and a tag is unwrapped, so an unknown optional field is carried rather than refused. Duplicate map keys anywhere are `not_a_json_object` (RFC 8949 section 5.6).

Encoding ([`JSONToCBOR`](../internal/proof/cbor_encode.go)) is the exact inverse: hex slots become byte strings (uppercase hex is refused rather than laundered into a byte string the verifier would accept), base64 slots become byte strings, every number is preserved exactly (an integer beyond 64 bits is an error naming the key), and the output is RFC 8949 core-deterministic with the tag prepended. `truestamp convert proof` round-trips either way; `internal/proof/parse_test.go` pins that JSON to CBOR to JSON is value-identical and that `proof-complete.cbor` decodes to the same values as `proof-complete.json`.

## Hex encoding (E.4)

The closed list of hex fields that MUST be lowercase: every `signing_key_id` (subject and every block map), every block map's `previous_block_hash` and `merkle_root` (top-level `block`, each `block_path` entry, `signing_key_event.block`), every value of `subject.metadata.witnesses`, and every commitment's `epoch_merkle_root`, `transaction_hash` and `block_merkle_root` (in `commitments` and in `signing_key_event.commitments`). A violation is a graded `fail` carrying `invalid_hex_encoding`, reported once under **Structure** naming every offender in wire order (the reference verifier's names: `subject.signing_key_id`, `block.merkle_root`, `commitments[0].transaction_hash`, `subject.metadata.witnesses.block`, ...) and again at the point of use in the consuming step. The sweep emits no row on a conforming bundle.

The head block carried as the `block` witness is deliberately not swept; its hex fields are graded where its hash is derived, in E.17a, as `witness_hash_mismatch`. The exclusions are normative too: `public_key`/`signature` (base64) and `inclusion_proof`/`epoch_proof` (base64url) are case-significant; `raw_transaction` and `txoutproof` carry either base64url or hex, so a lowercase rule there is undefined (pinned by `TestCLI_RawTxAndTxOutProof_CaseIsNotGraded`); ids, `type`, `chain`, `network` and timestamps are not hashes; and everything inside a hashed map is canonicalized verbatim, `subject.claims.hash` included (the E.7 comparison of a caller-supplied expected hash is the one case-insensitive comparison, because that argument is an operator's typed input).
