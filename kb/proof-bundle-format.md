# Proof Bundle Format

All Truestamp proofs use a compact format with short keys. Bundle version `v` is `1`. The top-level integer field `t` discriminates subject types. Normative reference: **Appendix E of `truestamp-v2/whitepaper/whitepaper.typ`**, with the wire shape also described in `truestamp-v2/kb/verification/proof-bundle-format.md`. (The `docs/PROOF_FORMAT*.md` files this used to cite no longer exist; see `CLAUDE.md` §"Relationship to the Truestamp service" for where every truestamp-v2 reference moved.)

## Type code registry (frozen; never renumbered)

| Code | Name | Category |
| ---- | ---- | -------- |
| 10 | block | subject |
| 11 | beacon | subject |
| 20 | item | subject |
| 30 | entropy_nist | subject |
| 31 | entropy_stellar | subject |
| 32 | entropy_bitcoin | subject |
| 40 | commitment_stellar | external commitment |
| 41 | commitment_bitcoin | external commitment |

The single source of truth lives at [`internal/proof/ptype/ptype.go`](../internal/proof/ptype/ptype.go). Every caller branches on these integers, never on strings.

Block (`t=10`) and beacon (`t=11`) share the same wire shape (no `s`, no `ip`, `b` + `cx` only); the type code is the only discriminator. Verify-pipeline guards use `ptype.IsBlockLikeSubject` / `ProofBundle.IsBlockLike()`; there is deliberately no strict `IsBlock()` / `IsBeacon()` predicate, because every structural guard is a block-like guard and the places that need the exact code compare `bundle.T` against `ptype.Block` / `ptype.Beacon` directly. Because the `t` byte is part of the signing payload, a `t=10` and `t=11` bundle for the same underlying block produce **different signatures**; this is intentional cryptographic domain separation.

## Item / entropy bundle shape (t ∈ {20, 30, 31, 32})

```json
{
  "v": 1,
  "t": 20,
  "ts": "2026-04-06T23:25:06Z",
  "pk": "base64(32-byte Ed25519 pubkey)",
  "sig": "base64(64-byte Ed25519 sig over proof_hash)",
  "s":  { "id": "ULID|UUIDv7", "d": { subject_data... }, "mh": "hex64", "kid": "hex8" },
  "ip": "base64url(compact Merkle proof)",
  "b":  { "id": "uuid", "ph": "hex64", "mr": "hex64", "mh": "hex64", "kid": "hex8" },
  "cx": [
    { "t": 40, "net": "testnet|public", "tx": "hex64", "memo": "hex64", "l": 1800000, "ts": "iso8601", "ep": "base64url" },
    { "t": 41, "net": "regtest|testnet|mainnet", "tx": "hex64", "op": "hex64", "h": 850000, "rtx": "hex_var", "txp": "hex_var", "bmr": "hex64", "ep": "base64url" }
  ]
}
```

Item proofs (`t=20`) use domain prefixes `0x11` / `0x13` for subject-data / composite. Entropy proofs (`t ∈ {30,31,32}`) use `0x21` / `0x23`.

## Block-like bundle shape (t ∈ {10, 11})

Block (`t=10`) and beacon (`t=11`) share the same wire shape (no `s` key, no `ip` key) because the block IS the subject, so `subject_hash == block_hash` in the signed payload.

```json
{
  "v": 1,
  "t": 10,                           // or 11 for beacon
  "ts": "2026-04-06T23:25:06Z",
  "pk": "base64(...)",
  "sig": "base64(...)",              // differs between t=10 and t=11 for the same block
  "b":  { "id": "uuid", "ph": "hex64", "mr": "hex64", "mh": "hex64", "kid": "hex8" },
  "cx": [ ... at least one commitment ... ]
}
```

## Beacons are first-class proof bundles

A **beacon** is now its own subject type code (`t=11`) alongside plain block (`t=10`). Both share the structural shape above but are cryptographically distinct: the `t` byte lives in the signing payload, so a block and beacon bundle for the same underlying block have different signatures. Flipping `t` from `10` to `11` on a bundle without re-signing breaks verification.

- `truestamp beacon {latest|list|get|by-hash}` reads the compact metadata projection `{id, hash, timestamp, previous_hash}` at `GET /api/json/beacons/*` (see `truestamp-v2/kb/api/beacon-api.md`). The single-beacon card prints two shareable public-web links: `Details → {host}/beacons/<hash>` (the beacon detail page keyed by hash) and `Verify → {host}/verify/beacon/<id>` (the typed-sub-path verify landing page introduced in the t=11 cutover). URLs render unconditionally (localhost and plain-http hosts too) so the links are visible when developing against a local server; the accepted tradeoff is documented in [architecture.md](architecture.md) §"CLI behavior" → "Post-action card URL shape".
- `truestamp download --type beacon <uuidv7>` fetches a full `t=11` proof bundle labelled `truestamp-beacon-*` on disk. The wire request sends `data.type = "beacon"` verbatim; the server returns a `t=11` bundle that self-describes.
- `truestamp verify` dispatches beacon proofs through the same pipeline as block proofs (no subject-hash derivation, no inclusion-proof walk, `subject_hash == block_hash`). The Type row of the report reads "Beacon" for `t=11` and "Block" for `t=10`.

## Structural requirements

The parser aborts on **only** the hard rejections Appendix E.6 enumerates. Everything else parses cleanly so the pipeline can grade it as a report step: a verifier that refuses to produce a report cannot be compared with one that does. Every abort returns a `proof.RejectionError` carrying its E.23 identifier, readable via `proof.RejectionCode(err)`.

Hard rejections (parse aborts, no report):

- input is not a JSON object → `not_a_json_object`
- `t` absent or not an integer → `missing_type_code`; `t ∉ {10, 11, 20, 30, 31, 32}` → `invalid_subject_type_code`
- `b` absent or not a map → `missing_block`
- `cx` absent, not a list, or empty → `no_external_commitments`
- a `cx[i]` that is not a map, carries an unregistered `t`, or is missing `ep` or its chain-specific root key (`memo` for `t=40`, `op` for `t=41`) → `invalid_external_commitment_entry`
- `t ∈ {10, 11}` (block-like) carrying `s` or `ip` → `unexpected_subject_fields_for_block_like`
- `t ∉ {10, 11}` missing `s` → `missing_subject`; missing `ip` (or `ip` not a string) → `missing_inclusion_proof`
- a CBOR `s.d` value with no JSON counterpart → `invalid_subject_data`

Graded as report steps, **not** rejections:

- `v`: a missing, wrong or non-integer version carries through (as `0` when unreadable) and fails the E.8 Structure step. `v` is deliberately not a gate.
- `pk`, `sig`, `ts`: carried verbatim; E.9 / E.16 report what they find.
- hash and `kid` field sizes, an empty `ip`, and Stellar's `net` value. `net` is **not** strict: an absent or unrecognized network resolves to the testnet Horizon per E.5/E.18 and is graded, not rejected (`external.IsDefaultedNetwork` marks the defaulted case so a 404 there is a skip rather than a fail).
- Subject `kid` may differ from block `kid` under legitimate key rotation; no equality assertion is made. Subject-kid tampering is still detected because `kid` is an input to the 0x13 / 0x23 composite hash.
- **Hex case.** E.6 gates on the presence and type of structure, never on the lexical form of a value, so a wrongly encoded field is graded rather than rejected — the same treatment E.9 gives a `pk` that does not decode to 32 bytes and E.12 gives a non-base64url `ip`. See "Hex encoding (E.4)" below for the field list.

## Hex encoding (E.4)

E.4 fixes a **closed set of ten** hex-encoded JSON fields that MUST be lowercase: `s.mh`, `s.kid`, `b.ph`, `b.mr`, `b.mh`, `b.kid`, `cx[].memo`, `cx[].op`, `cx[].tx`, `cx[].bmr`. That is exactly the set E.3 files as CBOR byte strings, less `pk` and `sig` (byte strings in CBOR, base64 in JSON). A violation is a **step `fail` carrying E.23's `invalid_hex_encoding` identifier and naming the offending field** — never an E.6 abort.

Both reporting points are required, and `internal/verify` emits both:

1. **The sweep** (`verifyHexEncoding`, run before any hash is derived) — one `fail` under `Structure` naming every offender in wire order. It is what reaches `cx[].tx` and `cx[].bmr`, which no verifier decodes. It emits **nothing** on a conforming bundle: a `pass` row D.4 does not carry would itself break E.25 containment.
2. **The point of use** — the consuming step fails too (`Subject Data`, `Block Hash`, `Inclusion Proof`, `Epoch Proof`), so an encoding defect is never reported as a generic root mismatch. The decoders in `internal/tscrypto` enforce it and name the wire key, not the parameter: `metadata_hash` alone cannot say whether `s.mh` or `b.mh` is at fault.

The **exclusions are normative too** — applying the rule outside the ten rejects conforming bundles:

| Excluded | Why |
| -------- | --- |
| `pk`, `sig` (base64), `ip`, `ep` (base64url) | Case-significant alphabets; preserve exactly |
| `cx[].rtx`, `cx[].txp` | E.3 files both as **text** carrying *either* base64url or hex, and hex is a subset of the base64url alphabet, so the rule is **not well defined** for them. No consequence either: neither is in E.16's signed payload, and both are decoded only so their derived values can be compared against `tx`, `op`, `bmr` — which the rule does cover |
| `s.id`, `b.id`, `net`, `ts` | Not hashes; a ULID is uppercase Crockford Base32 by construction. Case is already bound cryptographically — ids enter their preimage as UTF-8 bytes under `len32`, so a re-cased id derives a different subject/block hash and fails E.13/E.14 on its own |
| Everything inside `s.d`, incl. `s.d.hash` | Canonicalized verbatim; normalizing case would derive a different 0x11/0x21 digest and report a valid proof as forged. The one exception is E.7's comparison of a caller-supplied `--hash` against `s.d.hash`, which MAY be case-insensitive (an operator's typed argument, not a wire field) — `tscrypto.HexEqual` folds case for exactly that reason |

Pinned by `TestAppendixD_UppercaseHexFieldsAreRejected` (all ten, both reporting points, identifier, field naming), `TestHexEncodingExclusionsAreNotGraded`, `TestConformingBundleEmitsNoEncodingRow`, and `TestCLI_RawTxAndTxOutProof_CaseIsNotGraded`. History: truestamp-cli found the rule and the boundary; truestamp-v2 commit `4b1beaff0d` made it normative and brought all three of its verifiers into line (their reference verifier had crashed with an uncaught `ArgumentError`; their production verifier had accepted uppercase outright). The exchange is in that repo's `COMMS.md`, "Round 3".

