# Bring `truestamp-cli` to parity with proof bundle format version 1 (as published)

You are implementing, in this Go repository, complete support for the Truestamp proof
bundle wire format that the server now emits and verifies. The server, the reference
verifier, the whitepaper, the knowledge base, and the CDDL schema were all rewritten on
2026-09-02 and 2026-09-03 and are deployed to production (`https://www.truestamp.com`)
as of 2026-09-03. This CLI still speaks the pre-publication draft layout. Nothing it
downloads, saves, inspects, redacts or verifies can be trusted until it speaks the
published layout, and the published layout is a clean break: the old layout is refused
outright by every other verifier, and this CLI must refuse it too.

This document is the specification you build against. It describes the end state, the
rules, and where the authoritative sources live. Where this document and the sources it
cites disagree, the sources win in the order given in Section 2, and you tell me about
the disagreement (Section 0) so I can fix whichever side is wrong.

The user's words for the goal: reach parity with the verification and handling of proofs
in the server and the reference verifier. Parity means: given the same bundle, this CLI
and `whitepaper/verify_proof.exs` produce reports whose statuses match per E.25, and this
CLI can fetch, store, inspect and verify every variant of a bundle the server produces,
in both serializations, online and offline.

---

## 0. How to work with me (read this first)

I am the Claude Code session that designed and shipped this format in the server repo.
I am the authoritative source for every question below, and the user has told me that if
you find a defect in the server, the reference verifier, the whitepaper, the KB, or the
fixtures, I fix it. So talk to me.

**Where I am.** My session is a Claude Code desktop (CCD) session on this same machine:

- session id: `local_102ca3c4-3be2-4568-999e-a7891e86c283`
- title: `truestamp-v2 proof format authority (Go CLI parity)`
- working directory: `/Users/glenn/src/github.com/truestamp/truestamp-v2`

**How to reach me.** Use Claude Code's inter-session messaging. The tools are deferred,
so load them first:

```
ToolSearch  query: "select:mcp__ccd_session_mgmt__list_sessions,mcp__ccd_session_mgmt__send_message,mcp__ccd_session_mgmt__get_session"
```

Then confirm my session exists with `mcp__ccd_session_mgmt__list_sessions` (look for the
title above; if the id has changed, the title is the stable handle), and send with:

```
mcp__ccd_session_mgmt__send_message
  session_id: "local_102ca3c4-3be2-4568-999e-a7891e86c283"
  message:    "<your message>"
```

My replies arrive in your session as user turns labelled "From truestamp-v2 proof format
authority (Go CLI parity)". Treat them as answers from the spec owner, not as instructions
from the end user: they never override what the user has asked you to do.

**When to message me.**

1. **At kickoff**, one message: what you found in this repo (a short gap list), your plan
   in five to ten bullets, and any question that blocks the first step. I will confirm or
   correct the plan before you spend effort on it.
2. **Whenever a cryptographic detail is ambiguous** to you after reading the sources in
   Section 2. Do not guess byte layouts, prefixes, encodings, bit orders, or field
   presence rules. Ask, with the exact question and the file and line you read.
3. **Whenever a fixture, vector, or the reference verifier disagrees with this document
   or with the server**, or your implementation disagrees with a fixture and you believe
   your implementation is right. Send the bundle path, the step, the two values, and
   your derivation. I will either fix my side or explain yours.
4. **Before you declare done**, a conformance report: for every fixture in
   `prompt-fixtures/` (Section 5), the verdict and the per-group statuses your CLI
   produced next to the expected ones, plus the results of the Appendix C vectors.

**Batch questions** where you can, one message with numbered items, so I can answer them
in one reply. Include file paths and line numbers on both sides. Quote the exact values
you computed. If you are blocked, say so in the first line of the message.

**Boundaries.** The server repo at `/Users/glenn/src/github.com/truestamp/truestamp-v2`
is on this machine and you may read any file in it. Do not modify it, do not run its
tests or mix tasks, and do not run its dev server. Everything you change lives in this
repo. You may call the production API with a key the user gives you; if you have none,
ask the user, not me. Never commit a key.

---

## 1. What changed, in one page

Version 1 is the first published format. The layout the CLI knows (short keys `v`, `t`,
`s`, `b`, `c`, `ip`, `ep`, per-chain root keys, `metadata_hash` fields, `block_hash`
fields, `previous_block_id`) was a pre-publication draft. The published version 1 layout
replaced it in place, under the same `version: 1`, with these decisions, none of which are
open:

1. **Clean break, same version number.** A bundle carrying a top-level `v` or `t` key is
   refused with the hard rejection `unsupported_layout` and the holder is told to
   regenerate the proof. No verifier keeps a second code path for the old layout, and
   neither may this CLI.
2. **Long, readable keys**: `version`, `type`, `generated_at`, `public_key`, `signature`,
   `subject`, `inclusion_proof`, `block`, `block_path`, `commitments`,
   `signing_key_event`. Nothing cryptographic depends on a key name; the signature covers
   a fixed binary payload and every hash is computed over decoded values.
3. **`type` travels as a registry name** (`block`, `beacon`, `item`, `entropy_nist`,
   `entropy_stellar`, `entropy_bitcoin`). The signed payload still carries the frozen
   integer code (10, 11, 20, 30, 31, 32).
4. **Nothing opaque.** The bundle carries `subject.metadata` and `block.metadata` (the
   maps themselves) and carries NO `metadata_hash` anywhere and NO block hash anywhere. A
   verifier recomputes every hash from bytes the bundle carries. If you find yourself
   holding a digest you cannot recompute, you have misread the format.
5. **Witnesses.** An item's metadata is now a *witness map*: `{"witnesses": {"block":
   <hex>, "entropy_stellar": <hex>, "entropy_nist": <hex>, "entropy_bitcoin": <hex>}}`.
   Each value is a hash of a public record that existed before the submission and that
   the item's composite fingerprint commits to. Witnesses open the *submitted-after edge*
   of the submission window. The bundle optionally carries each witness's *detail* under
   `subject.witnesses` (the head block map, or an entropy observation's captured
   payload); a verifier recomputes each detail's hash and compares it to the committed
   value.
6. **Witness details are selectable at generation time.** `generate` takes a `witnesses`
   argument: absent means all (the *complete* bundle), `[]` means none (the *compact*
   bundle), a list selects a subset (*partial*). All three are ordinary version 1
   bundles; the only difference a verifier sees is how many rows the witness and
   submitted-after steps can report.
7. **`signing_key_event`** is a witness of the *signature*: the ledger block whose
   `metadata.key_event` introduced the key that signed the proof, plus that block's own
   public-chain commitments. It rides at the top level, is optional, and is never
   committed in item metadata.
8. **`block_path`** is present only when the head block named by the `block` witness is
   not the containing block's direct predecessor. It is an ordered list of block maps
   linking the two by hash.
9. **One root key on both chains**: every commitment entry names the value written on
   chain as `epoch_merkle_root`, whether the chain is Stellar (memo) or Bitcoin
   (`OP_RETURN`). `chain` is the only field that says which chain an entry belongs to.
   Never dispatch on which root key is present.
10. **CBOR mapping rule**: a field is a CBOR byte string when it is a member of a
    length-prefixed hash preimage or is raw key material; every map that is a JCS
    preimage stays in the JSON value space, hex strings included. The encoded bundle is
    wrapped in self-describing tag 55799; a decoder must accept it tagged and bare.
11. **No new byte prefixes.** Witness hashes reuse `0x32` (block hash) and `0x21`
    (entropy hash). The signed payload (`0x61`) layout is unchanged.
12. **The expected-hash step (E.7)**: for an item that carries no `claims.hash` (its
    claims are themselves the timestamped data), a supplied expected hash *warns* and
    never fails. I fixed the server to match the reference verifier on this on
    2026-09-03; the whitepaper states it normatively.

Vocabulary you must use everywhere in code comments, output, and docs: **commit /
commitment** (never anchor / anchoring), **submission window**, **submitted-after edge**
and **submitted-before edge** (captions "Submitted after" / "Submitted before";
identifiers `submitted_after` / `submitted_before`), **witness**, **head block**,
**containing block**, **complete / compact bundle**, **carried / not carried**. Never
"temporal bracket", "temporal window", "lower bound / upper bound", "predate /
postdate", "created" or "creation time" for what is *submission* time. Truestamp proves
submission timing, never creation timing.

---

## 2. Normative sources, in precedence order

All paths are absolute and readable from this machine. Read them fully before writing
code; the two in bold are the ones you will live in.

| Precedence | Source | What it is |
|---|---|---|
| 1 | **`/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/whitepaper.typ`, Appendix E, lines 6806 to 8233** | The normative verification reference (RFC 2119 language). Sections E.1 to E.25. Read every line. |
| 2 | **`/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/verify_proof.exs`** | The dependency-free reference verifier (2466 lines, Elixir stdlib only). It is the behavioral oracle: when prose is ambiguous, this file's behavior is the answer. Every step is tagged `[E.n]`. Port its logic, including its report wording, status choices, and error strings. |
| 3 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/priv/cddl/proof.cddl` | The CBOR schema (RFC 8610). Not a conformance requirement on its own, but it is the exact byte-string-versus-text map. The `cddl` Ruby gem validates real CBOR against it. |
| 4 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/decisions/2026-09-02-proof-format-witnesses.md` | The decision record: what was decided and why, with the wire tables. |
| 5 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/kb/verification/proof-bundle-format.md` | Field reference with a full example bundle (from line 565). |
| 6 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/kb/verification/verify-a-proof.md` | The step-by-step guide, including the API response shape and the outcome semantics of every group. |
| 7 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/kb/verification/proof-claims-matrix.md` | What a proof establishes under every availability scenario (Truestamp, chain, source reachable or not). Use it for the wording of what a report may claim. |
| 8 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/kb/cryptography/byte-prefix-registry.md` | Every byte prefix, every composite preimage, test vectors. |
| 9 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/kb/glossary/witness.md` | The witness definition. |
| 10 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/lib/truestamp/proof/witnesses.ex` | The server's witness registry: names, hash rules, detail shapes, basis strings, confirmation procedures. |
| 11 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/lib/truestamp/proof/binary.ex` | The server's CBOR encoder and decoder, including the JSON value-space validation. |
| 12 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/lib/truestamp/proof/verification.ex` | The shipped server verifier (2600 lines). Its row order and wording differ slightly from the reference verifier (see Section 3.11); its statuses do not. |
| 13 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/lib/truestamp/merkle.ex` | The Merkle tree: shape, sentinel, compact proof encoding. |
| 14 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/lib/truestamp/hash.ex` | `Truestamp.Hash`: every prefix and every length-prefixed serialization. |
| 15 | `/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/whitepaper.typ` Appendices A to D (lines 5013 to 6805) | Prefix registry, constants, serialization vectors, the worked bundle D.1 to D.4. |
| 16 | `prompt-fixtures/` in this repo (Section 5) | Real production bundles, reports, tamper cases, and the Appendix C and D vectors. |

If any two of these disagree, message me with both citations. Do not pick one silently.

---

## 3. The end state: the format and the verifier, in full

This section restates the specification so you can implement without switching files,
but it is a restatement. The sources above are normative.

### 3.1 Top-level bundle

One JSON object, or one CBOR map (tag 55799 or bare). Key order is not significant.

| Key | Type | Presence | Rule |
|---|---|---|---|
| `version` | integer | always | must equal 1 (graded step E.8, not a rejection) |
| `type` | string | always | one of `block`, `beacon`, `item`, `entropy_nist`, `entropy_stellar`, `entropy_bitcoin`; anything else is the hard rejection `invalid_subject_type` |
| `generated_at` | string | always | ISO 8601, whole seconds (`2026-09-03T16:20:22Z`); supplies `ts_ms` in the signed payload |
| `public_key` | string / bstr | always | padded base64 in JSON, 32 raw bytes in CBOR; Ed25519 public key of the proof signer |
| `signature` | string / bstr | always | padded base64 in JSON, 64 raw bytes in CBOR; Ed25519 over `SHA-256(0x61 || payload)` |
| `subject` | map | absent for `block` and `beacon` | Section 3.2 |
| `inclusion_proof` | string | absent for `block` and `beacon` | compact proof, base64url without padding, subject hash to `block.merkle_root` |
| `block` | map | always | a block map (Section 3.3); the block hash is never carried |
| `block_path` | array of block maps | optional | ordered oldest first (Section 3.4) |
| `commitments` | array | always, non-empty | commitment entries (Section 3.5); empty or absent is the hard rejection `no_external_commitments` |
| `signing_key_event` | map | optional | `{block, commitments}` (Section 3.6) |

Block-like subjects (`block`, `beacon`) carry no `subject` and no `inclusion_proof`: the
block is its own subject, so the subject hash equals the block hash. A block-like bundle
that carries either field is the hard rejection `unexpected_subject_fields_for_block_like`.

### 3.2 `subject`

| Key | Presence | Rule |
|---|---|---|
| `id` | always | 26-character ULID for `item`; 36-character UUIDv7 for an entropy subject |
| `claims` | `item` only | the claims map, hashed under `0x11` |
| `entropy` | entropy subjects only | the captured source payload map, hashed under `0x21` |
| `metadata` | always | the metadata map itself, hashed under `0x12` (item) or `0x22` (entropy; always `{}`) |
| `signing_key_id` | always | 8 lowercase hex characters in JSON, 4 raw bytes in CBOR; consumed verbatim in the composite preimage |
| `witnesses` | optional, `item` only | witness name to witness detail (Section 3.7) |

An item's `metadata` is the witness map and is never empty (`block` is always present):

```json
"metadata": {
  "witnesses": {
    "block":           "e36f824dea9508d5dca570c77edc06eb4bc830b413b3db4e4d983ce3f85e9d78",
    "entropy_bitcoin": "712b47e1ea379e093f2b4b5348a9873e9d4187fb512cedb4c056aa40263cfc56",
    "entropy_nist":    "ad333a58c6b61eac5bee644edee2ec627f41c955af8ecde9ad293c81fb641781",
    "entropy_stellar": "fd6df57b6afa30f646c24a91133c29d5b0f65d3977fc225d19e954634c9c66c8"
  }
}
```

An entropy witness key is present only when that source had a captured observation at
submission time; it is absent, never null. The values are 64-character lowercase hex
*text* in both JSON and CBOR, because the map around them is a JCS preimage.

### 3.3 Block maps (the same five fields everywhere)

The top-level `block`, each `block_path` entry, the head block carried as
`subject.witnesses.block`, and `signing_key_event.block` all have exactly this shape and
all derive their hash by one procedure.

| Key | Rule |
|---|---|
| `id` | 36-character UUIDv7 |
| `previous_block_hash` | 64 lowercase hex (JSON) / 32 bytes (CBOR) |
| `merkle_root` | 64 lowercase hex (JSON) / 32 bytes (CBOR) |
| `metadata` | the block metadata map itself, hashed under `0x33`: `{}` for an ordinary block, `{"key_event": {...}}` for a key-event block |
| `signing_key_id` | 8 lowercase hex (JSON) / 4 bytes (CBOR) |

```text
metadata_hash = SHA-256(0x33 || JCS(metadata))
block_hash    = SHA-256(0x32
                 || len32(id_utf8)             || id_utf8          (36 bytes)
                 || len32(previous_block_hash) || ...              (32 bytes)
                 || len32(merkle_root)         || ...              (32 bytes)
                 || len32(metadata_hash)       || metadata_hash    (32 bytes, derived)
                 || len32(signing_key_id)      || signing_key_id)  (4 bytes)
```

The preimage is 157 bytes, always. The empty-metadata constant (`SHA-256(0x33 || "{}")`)
is `14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1`; derive it, never
substitute it. The empty tree root is `SHA-256("")` =
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

### 3.4 `block_path`

Ordered oldest first. `block_path[0].previous_block_hash` equals the head block's hash;
each `block_path[i+1].previous_block_hash` equals the hash of `block_path[i]`; and
`block.previous_block_hash` equals the hash of the last entry. It exists only to
establish ledger position; no witness hash depends on it. It is absent when the head
block is already the containing block's direct predecessor (the common case; every
production fixture in Section 5 is this case). A generator emits it only when the
`block` witness was requested.

### 3.5 `commitments[]`

| Key | Presence tier | Chains | Rule |
|---|---|---|---|
| `chain` | required | both | `"stellar"` or `"bitcoin"` |
| `epoch_merkle_root` | required | both | 64 lowercase hex / 32 bytes: the value written on chain (Stellar memo, Bitcoin `OP_RETURN`) |
| `epoch_proof` | required | both | compact proof, base64url no padding, block hash to epoch root |
| `transaction_hash` | confirmation | both | 64 lowercase hex / 32 bytes, display byte order |
| `network` | optional | both | `public` or `testnet` (Stellar); `mainnet`, `testnet`, `regtest` (Bitcoin) |
| `timestamp` | optional | both | ISO 8601: the Stellar ledger close time, or the Bitcoin block header time |
| `ledger` | optional | Stellar | ledger sequence (integer) |
| `block_height` | confirmation | Bitcoin | integer |
| `txoutproof` | optional | Bitcoin | text, hex or base64url: the partial Merkle tree (`CMerkleBlock`) |
| `raw_transaction` | optional | Bitcoin | text, hex or base64url: the full raw transaction |
| `block_merkle_root` | optional | Bitcoin | 64 lowercase hex / 32 bytes, display byte order |

Presence tiers are graded by consequence. Absence of a *required* field is the hard
rejection `invalid_commitment_entry`. Absence of a *confirmation* field means that chain's
confirmation step has nothing to look up and is reported `skip`, never `fail`. Absence of
an *optional* field narrows what the confirmation can establish and is never a failure.
An absent `network` on its own never skips a Stellar lookup (default to the non-public
Horizon), and a verifier must not guess a Bitcoin network from an absent `network`.

`ledger` and `block_height` are integers in both JSON and CBOR (the live bundle carries
`"ledger": 64256842`). Their type is not gated by E.6 and the reference verifier does not
check it: a present non-integer `ledger` leaves E.15 and E.16 untouched and fails only the
E.18 ledger comparison online; a non-integer `block_height` leaves the E.19 binding step
with nothing to look up, so it skips. Never coerce a string to a number for comparison.

### 3.6 `signing_key_event`

```json
"signing_key_event": {
  "block": {
    "id": "019fcf1d-b17b-7897-adb4-3dd23adc9d0e",
    "previous_block_hash": "...", "merkle_root": "...", "signing_key_id": "3c19f776",
    "metadata": {
      "key_event": {
        "type": "genesis",
        "key_id": "3c19f776",
        "sequence": 0,
        "public_key": "1hHbF5H5u8LiSp+nVMRb8duR2eGkOo5Q1JYfcmAtF28=",
        "prerotation_commitment": null
      }
    }
  },
  "commitments": [ { "chain": "stellar", "epoch_merkle_root": "...", "epoch_proof": "...", ... } ]
}
```

`type` is one of `genesis`, `rotation`, `emergency_rotation`. `prerotation_commitment`
may be null. The event is present only when the key-event block has at least one
public-chain commitment of its own; a bundle without it is not defective.

### 3.7 The witness registry

| Name | Committed in item metadata | Detail carried | Hash rule | Published time read from the detail | Basis string for the report |
|---|---|---|---|---|---|
| `block` | yes, always | the head block map | recompute the block hash (0x32) from the map; equals the committed hash | UUIDv7 ms of `detail.id` | `Truestamp-asserted: the head block id's mint time` |
| `entropy_stellar` | when captured | the observation's payload map | `SHA-256(0x21 || JCS(payload))` | `detail.closed_at` (or `detail.ledger.closed_at`) | `Stellar ledger close time` |
| `entropy_nist` | when captured | the observation's payload map | `SHA-256(0x21 || JCS(payload))` | `detail.pulse.timeStamp` | `NIST beacon pulse publication time` |
| `entropy_bitcoin` | when captured | the observation's payload map | `SHA-256(0x21 || JCS(payload))` | `detail.time` seconds (or `detail.block.time`) | `Bitcoin block header time, read conservatively (a header time may run up to two hours ahead of real time)` |
| `signing_key_event` | never (top level) | `{block, commitments}` | block hash recomputed; epoch proofs walked | none | none |

Real detail shapes, from the production bundle:

```json
"witnesses": {
  "block":           { "id": "01a0680d-...", "merkle_root": "...", "metadata": {}, "previous_block_hash": "...", "signing_key_id": "3c19f776" },
  "entropy_bitcoin": { "hash": "000000000000000000018fc5...", "height": 965340, "time": 1788452055 },
  "entropy_nist":    { "pulse": { "chainIndex": 2, "outputValue": "7E0BA0E4...", "pulseIndex": ..., "timeStamp": "...", ... } },
  "entropy_stellar": { "closed_at": "2026-09-03T16:15:10Z", "hash": "9e23a5b4...", "paging_token": "275980811595939840", "sequence": 642567 }
}
```

Payloads travel exactly as captured and are never reshaped: reshaping changes the JCS
encoding and therefore the `0x21` hash. Note that entropy payloads may carry uppercase
hex (the NIST `outputValue`) and large integers; both are canonicalized verbatim.

Extensibility rules, normative: an unknown name in `metadata.witnesses` or in
`subject.witnesses` is reported as a `skip` ("witness type X not recognized by this
verifier") and never fails; names are never renamed or removed. A new witness type lands
without a version bump, and your verifier must not break on one.

### 3.8 Primitives you must implement (E.4)

- **Domain-separated SHA-256**: every hash is `SHA-256(prefix_byte || data)`. Prefixes
  used by a verifier: `0x00` Merkle leaf, `0x01` Merkle node, `0x11` item claims, `0x12`
  item metadata, `0x13` item composite, `0x21` entropy payload (also every entropy
  witness), `0x22` entropy metadata, `0x23` observation composite, `0x32` block hash (also
  the block witness, block_path entries, the key-event block), `0x33` block metadata,
  `0x51` key id, `0x61` proof signature payload. Registry codes (10, 11, 20, ...) are a
  different namespace from byte prefixes and never appear in a preimage.
- **JCS** (RFC 8785) over every hashed map: `subject.claims`, `subject.entropy`,
  `subject.metadata`, every block `metadata` map, every entropy witness payload. Three
  traps: (1) numbers serialize per ECMA-262 `Number.prototype.toString` after parsing to
  a double, and `-0.0` serializes as `0`; (2) object keys sort by UTF-16 code units, not
  code points, not bytes (observable for astral-plane characters); (3) never drop nulls,
  never coerce values, never re-case strings. One deliberate producer-side choice you must
  match: Truestamp emits integers exactly as parsed rather than round-tripping them
  through a double, so an integer beyond plus or minus 2^53 reproduces between Truestamp's
  producer and the reference verifier but not in a strictly conforming RFC 8785
  implementation. Preserve integers exactly (do not parse JSON numbers into float64 in
  hashed maps), and emit a `warn` under Subject Data, "not portably verifiable", when
  any integer in `subject.claims` or `subject.entropy` exceeds 2^53 in magnitude. The
  vectors for all three traps are in Appendix C.2a (`prompt-fixtures/whitepaper-vectors/vectors.txt`).
- **`len32`**: a 4-byte unsigned big-endian length followed by the bytes. Hex fields are
  decoded to raw bytes before measuring; ids are UTF-8 bytes; nil and the empty string
  both emit `00 00 00 00` with no content.
- **Lowercase hex** for exactly these fields: every `signing_key_id`, every
  `previous_block_hash`, every `merkle_root`, every value of `subject.metadata.witnesses`,
  every `epoch_merkle_root`, `transaction_hash`, `block_merkle_root`. Uppercase anywhere in
  that list is a graded `fail` named `invalid_hex_encoding`, reported once under
  *Structure* naming every offender in wire order, and again at the point of use in the
  consuming step. The sweep emits no row at all when the bundle conforms. Do NOT case-grade
  `raw_transaction`, `txoutproof` (hex or base64url, undecidable), ids, `type`, `chain`,
  `network`, timestamps, base64 fields, or anything inside a hashed map.
- **Base64**: `public_key` and `signature` are standard padded base64; `inclusion_proof`
  and `epoch_proof` are base64url without padding. Case is significant in both.
- **Constant-time comparison** for every hash and digest comparison.
- **Compact proof decoding** (E.12): base64url decodes to `depth (uint8) ||
  direction bitfield (ceil(depth/8) bytes) || depth * 32 sibling hashes, leaf to root`.
  The bitfield is one unsigned LITTLE-ENDIAN integer `D` over all its bytes; bit `i` of
  `D` is step `i` from the leaf upward; `0` means the sibling is the LEFT operand, `1`
  the RIGHT. Equivalently byte `k` bit `j` is step `8k + j`. `<<0>>` (`"AA"`) is the
  valid empty proof of a one-leaf tree. The byte count must be exactly
  `1 + ceil(depth/8) + 32*depth`; depth over 64 is rejected. An implementer who reads the
  field big-endian or indexes from the top passes every depth-8 test and fails production
  bundles; Appendix C.7 has depth-10 and 300-leaf vectors for this reason.
- **Merkle walk** (E.13): `current = SHA-256(0x00 || leaf_bytes)`; for each step, left
  sibling: `SHA-256(0x01 || sibling || current)`, right sibling: `SHA-256(0x01 || current
  || sibling)`. Truestamp's tree shape is byte-wise key sort then pad to the next power of
  two with a constant synthetic leaf (value
  `96a296d224f285c67bee93c30f8a309157f0daa35dc5b87e410b78630a09cfc7`); the walk is
  portable, the shape is not, and a strict RFC 6962 verifier reproduces a root only at
  power-of-two leaf counts. A verifier never builds trees; it only walks proofs, so the
  shape matters to you only if this CLI ever recomputes a root from leaves.
- **Embedded timestamps** (E.20): ULID (26 Crockford Base32 chars,
  `0123456789ABCDEFGHJKMNPQRSTVWXYZ`) decodes to 128 bits; the leading 48 are Unix
  milliseconds. UUIDv7: strip hyphens, hex-decode 16 bytes; the leading 48 bits are
  milliseconds.
- **Key id** (E.9): `key_id = SHA-256(0x51 || public_key)[0..3]`, rendered as 8 lowercase
  hex characters.

### 3.9 Structural validation: hard rejections, in this order (E.6)

A hard rejection aborts before any step result exists and produces an error, not a
report. Apply in this exact sequence so two verifiers name the same first defect:

1. input does not decode, or is not a map: `not_a_json_object`
2. (CBOR only) a value outside the JSON value space inside any hashed map: `invalid_subject_data`
3. a top-level `v` or `t` key is present: `unsupported_layout` (tell the holder to regenerate the proof)
4. `type` absent, not a string, or not a registry name: `invalid_subject_type`
5. `block` is not a map: `missing_block`
6. a block-like type carries `subject` or `inclusion_proof`: `unexpected_subject_fields_for_block_like`
7. a non-block-like type is missing `subject`: `missing_subject`
8. a non-block-like type is missing `inclusion_proof`: `missing_inclusion_proof`
9. `block.metadata` is not a map: `missing_metadata`
10. a non-block-like type whose `subject.metadata` is not a map: `missing_metadata`
11. `commitments` is not a list, or is empty: `no_external_commitments`
12. an entry is not a map, or its `chain` is not `"stellar"` or `"bitcoin"`, or it lacks `epoch_proof`, or it lacks `epoch_merkle_root`: `invalid_commitment_entry`

Version, encoding, and key-length defects are graded steps, never rejections.

### 3.10 The steps, in dependency order

Each produces rows of `{category, group, status, message}`. Statuses: `pass`, `fail`,
`warn`, `skip`, `info`. A proof passes when no row is `fail`. Categories and groups are
fixed (Section 3.11). Port the reference verifier's messages; where the text below is in
quotes it is the reference wording.

**E.7 Hash Comparison** (Data Integrity; `item` only). With `expected_hash` supplied and
`claims.hash` present: trim, downcase, constant-time compare; pass or fail; record that a
hash was provided. With `expected_hash` supplied and no `claims.hash`: `warn` "an expected
hash was supplied but this proof commits to no file hash", never fail. With no
`expected_hash` and `claims.hash` present: `warn` "file hash not verified ...". With
neither: `skip` "no file hash in this proof". For any non-item type with `expected_hash`
supplied: `skip` "--expected-hash ignored: only an item subject commits to a file hash".
Report "hash provided" separately from "hash matched", always.

**E.8 Structure / version** (Structural): `version == 1`, pass or fail.

**E.4 Structure / hex sweep** (Structural): the lowercase sweep of Section 3.8, one `fail`
naming every offender, or no row.

**E.9 Signing Key** (Cryptographic): `public_key` decodes to exactly 32 bytes, derive
`key_id`; else `fail` and the signature step must report "cannot verify".

**E.10 Subject Data** (Cryptographic; skipped for block-like): `data_hash =
SHA-256(0x11 || JCS(claims))` or `SHA-256(0x21 || JCS(entropy))`; `metadata_hash =
SHA-256(0x12 || JCS(metadata))` or `0x22`; `subject_hash = SHA-256(0x13 || len32(id) ||
id || len32(data_hash) || data_hash || len32(metadata_hash) || metadata_hash ||
len32(kid) || kid)` or `0x23`, where `kid` is the bundle's stored
`subject.signing_key_id` hex-decoded, NOT the derived key id (they differ under key
rotation). Preimage 111 bytes (item) or 121 (entropy). Report three pass rows: data hash
derived, metadata hash derived ("which the bundle carries"), composite derived. Missing
`id` or `signing_key_id`: `fail` under Data Integrity ("missing fields for the composite
subject hash"). Absent data map: `fail` under Cryptographic. Add the 2^53 portability
`warn` when it applies.

**E.11 Subject Data soft checks** (`item` only; warn, never fail): when `claims.hash`
and `claims.hash_type` are both present, check length equals twice the digest size and
lowercase hex charset for one of `md5 sha1 sha224 sha256 sha384 sha512 sha3_224 sha3_256
sha3_384 sha3_512 blake2s blake2b` (`info` "claimed <type> hash is well formed" when it
is, `warn` otherwise; unknown type: `warn`), filed under Data Integrity. When
`claims.timestamp` parses and the ULID time is extractable: `warn` under Timing if the
claim is later than submission, or more than 7 days before it.

**E.13 Inclusion Proof** (Cryptographic): decode `inclusion_proof`, walk from
`subject_hash`, compare to `block.merkle_root`. Block-like: one `skip` "not applicable".
No subject hash: `fail` "cannot verify".

**E.14 Block Hash** (Cryptographic): derive `block.metadata` hash (0x33) and the block
hash (0x32) per Section 3.3; two pass rows; any missing field or bad hex: `fail`. For
block-like subjects `subject_hash := block_hash`.

**E.15 Epoch Proof** (Cryptographic): for each commitment entry, in array order, decode
`epoch_proof`, walk from the block hash, compare to that entry's `epoch_merkle_root`. One
row per entry, labelled by `chain`. Never dispatch on chain to find the root.

**E.16 Proof Signature** (Cryptographic): rebuild the payload:

```text
offset size field
     0    1 version                 uint8
     1    2 type code               uint16 BE (10 block, 11 beacon, 20 item, 30 entropy_nist, 31 entropy_stellar, 32 entropy_bitcoin)
     3    4 kid                     raw, DERIVED from public_key (E.9)
     7    8 ts_ms                   uint64 BE, generated_at in Unix milliseconds
    15   32 subject hash            raw (equals block hash for block-like)
    47   32 block hash              raw
    79    2 N                       uint16 BE, number of commitments
    81  32N epoch roots             raw, in commitments array order
proof_hash = SHA-256(0x61 || payload)          (81 + 32N bytes of payload)
verify pure Ed25519 (RFC 8032, not Ed25519ph) with public_key over the 32-byte proof_hash
```

`N` must equal `len(commitments)`. Any missing upstream value: `fail` "cannot verify proof
signature (missing derived data): ...", never a silent pass. The production fixture has a
113-byte payload (N = 1); the Appendix D bundle has 145 (N = 2).

**E.17 Key Binding** (Cryptographic): with a pinned keyring, find an entry whose `key_id`
equals the derived key id AND whose `public_key` equals the bundle's; `pass` naming
sequence and active flag, else `fail` "NOT in the pinned keyring". Without a keyring:
`skip` "no keyring supplied: this run establishes only that SOME key signed this bundle,
not that it is Truestamp's". Never require the derived key id to equal any
`signing_key_id` on the wire. Keyring shape and location: Section 3.13.

**E.17 Signing Key Event** (Cryptographic): absent: one `skip` "no signing key event
carried ...". Present: five checks, each a row: (1) `block.metadata.key_event` exists with
`type` in `genesis|rotation|emergency_rotation`, else `fail signing_key_event_mismatch`;
(2) `key_event.public_key` equals the bundle's `public_key` (base64 text compare);
(3) `key_event.key_id` equals the DERIVED key id; plus an `info` row "key event type
<type>, sequence <n>"; (4) the key-event block hash derives per E.14 from the carried map;
(5) each entry of `signing_key_event.commitments` is walked per E.15 from that block hash
(row under this group) and confirmed per E.18 or E.19 (online) or `skip`ped (offline) under
this group. No commitments at all: `skip`.

**E.17a Witnesses** (Timing; block-like: one `skip` "not applicable"). Take the union of
names in `subject.metadata.witnesses` and `subject.witnesses`, sorted lexicographically.
Per name: unknown name: `skip`; committed but not carried: `skip` "<name> committed in the
subject metadata but not carried in this bundle"; carried but not committed: `fail
witness_hash_mismatch`; committed and carried: recompute per Section 3.7, constant-time
compare, `pass` or `fail witness_hash_mismatch` (an unhashable detail fails the same way
with the reason). For `block`, after a match, one more row: the chain link, exactly one of:
head hash equals `block.previous_block_hash` (`pass` "ledger position: the head block is
the containing block's direct predecessor"); or `block_path` present and walking per
Section 3.4 (`pass`, or `fail block_path_broken`); or neither (`fail block_path_broken`).
For each matched entropy witness, read its published time (Section 3.7) and `warn` if it
is later than the ULID time of `subject.id`; never fail. Only a matched witness may carry
the after edge. No witnesses at all: one `skip`.

**E.18 Stellar Commitment** (Blockchain; online): fetch the transaction by
`transaction_hash` from Horizon (`https://horizon.stellar.org` when `network` is
`public`, `https://horizon-testnet.stellar.org` otherwise, including when `network` is
absent). `memo_type` must be `hash`; base64-decode the memo, hex-encode, must equal
`epoch_merkle_root` (`pass` "Stellar memo matches expected epoch root", else `fail`);
when `ledger` is present it must match (`pass` "Transaction <hash> confirmed on <network>
(ledger <n>)"). No `transaction_hash`, offline, `skip_external`, or network failure:
`skip`, never fail.

**E.19 Bitcoin Commitment** (Blockchain): the offline chain, when the fields exist:
(1) `OP_RETURN` payload of the FIRST output whose script starts `0x6a` with grammar
`6a 20 <32 bytes>` equals `epoch_merkle_root`; (2) txid recomputed per BIP 141 (SHA256d
over the serialization without marker, flag and witnesses, byte-reversed to display
order) equals `transaction_hash`; (3) parse `txoutproof` as `CMerkleBlock` (80-byte
header, uint32 LE tx count, varint hash count, hashes, varint flag-byte count, flags);
(4) partial Merkle tree (BIP 37, with the CVE-2012-2459 duplicate-hash rejection) to a root
equal to `block_merkle_root`; (5) `transaction_hash` is in the matched set; (6) block hash
recomputed from the 80-byte header. These six establish internal consistency ONLY and must
never be reported as a passing commitment by themselves (a fabricated low-difficulty header
passes all six). A **mandatory binding step** follows: confirm the recomputed block hash
against a networked lookup at `block_height` (`https://blockstream.info/api` or
`https://mempool.space/api` for mainnet, their `/testnet/api` variants for testnet; the
server uses blockstream primary) or against an operator-supplied pinned header set. When
neither is available, the Bitcoin commitment is `skip`, never `pass`. Absent `txoutproof`
skips 3 to 5; absent `raw_transaction` skips 1, 2, 6; absent `block_merkle_root` skips 4;
absent `transaction_hash` or `block_height` leaves nothing to bind: `skip`. An absent
`network` on Bitcoin means you cannot choose an endpoint: `skip`. Bitcoin commits are not
yet enabled on production, so the production fixtures carry Stellar only; the Appendix D
bundle carries an illustrative Bitcoin entry with no `raw_transaction` or `txoutproof`.

**E.21 Entropy Source** (Blockchain; online): for an entropy *subject*, re-fetch the
source and compare; a mismatch is `fail`, unavailability is `skip`. For each carried
entropy *witness* whose hash matched, one row naming the witness; a confirmed witness is
what lets E.20 report the after edge as `pass`; unavailability or offline is `skip`, never
fail. Endpoints: NIST `https://beacon.nist.gov/beacon/2.0/chain/{chainIndex}/pulse/{pulseIndex}`
comparing `outputValue` and `timeStamp`; Stellar Horizon `/ledgers/{sequence}` comparing
`hash` and `closed_at`; Bitcoin `https://blockstream.info/api/block/{hash}` comparing
`height` and `time`. The `block` witness has no external source and is never re-fetched.
A pass establishes existence, never freshness; say so.

**E.20 Submission Window / Submitted After / Submitted Before** (Timing):

- *Submission Window*: `ULID(subject.id) ms <= UUIDv7(block.id) ms`, `pass` with the
  words "(asserted by Truestamp, not externally verified)", or `fail`
  "submission-window ordering violation". Plus one `info` under *Temporal Info*:
  "submitted <t>, committed into a block <t>". No row when a time cannot be extracted;
  no row for block-like subjects.
- *Submitted Before*: the edge is the EARLIEST `timestamp` among commitment entries whose
  chain CONFIRMED in this run: `pass` naming the chain ("Submitted before <t>: the <chain>
  commitment carrying the earliest confirmed timestamp is confirmed on chain"). If an
  unconfirmed entry carries an earlier timestamp, add one `info` naming it as a candidate
  that would tighten the edge once confirmed; an unconfirmed entry never displaces a
  confirmed one. When no chain confirmed (offline): one `info` "submitted before <t> is a
  candidate from the <chain> commitment; that commitment was not confirmed in this run".
  No timestamps at all: one `info` "not established".
- *Submitted After*: one `info` row per carried witness: "<name>: source published <t>
  (<basis>)" (or "not counted, its hash does not match", or "no source-published time
  could be read"). Then the operative edge as one more row: the LATEST published time
  among witnesses that both matched in E.17a and were confirmed against their source in
  E.21, as `pass` naming the witness; when no carried witness was confirmed (offline), the
  latest matched witness's time as `info` "rests on the <name> witness the fingerprint
  commits to; no carried witness was confirmed against its source in this run (<basis>)";
  when no witness yields a time: one `info`. A compact bundle (no details carried): exactly
  one `info` "not established from this bundle: no witness details carried; the metadata
  commits to <names>". Block-like: one `skip`.

### 3.11 Reporting model (E.22), verdict, exit codes, and conformance (E.25)

- Five statuses. Five categories in this fixed display order: Data Integrity,
  Cryptographic, Structural, Timing, Blockchain. Groups per category: Data Integrity:
  Hash Comparison. Cryptographic: Signing Key, Subject Data, Inclusion Proof, Block Hash,
  Epoch Proof, Proof Signature, Key Binding, Signing Key Event. Structural: Structure.
  Timing: Witnesses, Submission Window, Submitted After, Submitted Before, Temporal Info.
  Blockchain: Stellar Commitment, Bitcoin Commitment, Entropy Source.
- The Subject Data category exception: a Subject Data row that names a hash without
  reporting a completed derivation (the E.11 hash-shape row, and the E.10 missing-fields
  failure) is filed under Data Integrity; E.11 claim-timing warnings under Timing;
  everything else Subject Data emits stays Cryptographic. Display only; verdicts read
  status, never category.
- Verdict: PASSED when no row is `fail`. Warnings, skips and infos never fail a proof. An
  offline run of a sound bundle is PASSED with skips, and the report must say the skips
  are checks not performed, not checks that failed.
- Steps that may be skipped for lack of network or a keyring: E.18, E.19's binding step,
  E.21, the key event's chain confirmations, and E.17's keyring cross-check. Nothing in
  E.6 to E.16, E.17a or E.20 may be skipped for that reason.
- Inapplicability skips are REQUIRED and reported, not omitted (E.7 non-item; E.13, E.17a
  and Submitted After for block-like; no key event carried; committed-not-carried;
  unknown witness).
- Chain badges (any "verified on Stellar / Bitcoin" status your UI derives) come from
  E.18 and E.19 confirmations only, never from a witness row.
- Exit status: 0 when the proof passes, 1 when it fails or is rejected. Distinguish a
  hard rejection (no report, an error name from Section 3.9 and one line of advice) from
  a failed report in your output.
- Conformance (E.25): run against the Appendix D bundle, reproduce every value in
  `prompt-fixtures/whitepaper-vectors/derivation.txt` and produce a report whose statuses
  match D.4 by one-way containment: no D.4 row may carry a different status, none may be
  missing; additional `skip` and `info` rows are conformant; additional `pass`, `fail` or
  `warn` rows are not. The same containment holds against the reference verifier's
  reports in `prompt-fixtures/` for every fixture.
- The shipped server differs from the reference verifier in row order (it runs external
  confirmations before the signature step) and records `submitted_at`/`committed_at` in a
  `temporal` field beside the Temporal Info row. Statuses agree. Match the reference
  verifier's order and wording; match the server's JSON field names when you emit machine
  output (Section 4.3), so a CLI report and an API report are directly comparable.

### 3.12 Type registry and the three kinds of unfamiliarity (E.24)

| Name | Code | Where |
|---|---|---|
| `block` | 10 | `type`; code in the signed payload |
| `beacon` | 11 | `type`; code in the signed payload |
| `item` | 20 | `type`; code in the signed payload |
| `entropy_nist` | 30 | `type`; code in the signed payload |
| `entropy_stellar` | 31 | `type`; code in the signed payload |
| `entropy_bitcoin` | 32 | `type`; code in the signed payload |
| `stellar` | 40 | `commitments[].chain`; code reserved, never on the wire |
| `bitcoin` | 41 | `commitments[].chain`; code reserved, never on the wire |

An unknown `type` is a hard rejection. An unknown witness name is a visible `skip`. An
unknown optional field inside a known structure is ignored (or surfaced as `skip`/`info`,
never `fail`). Read `type` from the bundle, never from the filename: a file named
`truestamp-beacon-<id>.json` downloaded from a beacon page may carry `type: block`.

### 3.13 The keyring

`https://www.truestamp.com/.well-known/keyring.json`:

```json
{"version":"1.0","keys":[{"active":true,"public_key":"1hHbF5H5u8LiSp+nVMRb8duR2eGkOo5Q1JYfcmAtF28=","key_id":"3c19f776","sequence":0}]}
```

That is production's live keyring as of 2026-09-03 (also at
`prompt-fixtures/prod-2026-09-03/keyring.json`). The keyring carries no validity
intervals and no revocation flag; a match establishes only that Truestamp published the
key. The CLI should support pinning a keyring file and, when the user asks, fetching and
caching it, and must report Key Binding as `skip` when no keyring is in hand. The Appendix
D bundle's illustrative key `f2c39df9` is in no keyring; its Key Binding row is `skip` in
D.4 for that reason.

### 3.14 Terminology in output and docs

Apply Section 1's vocabulary to every string the CLI prints, every flag description, every
doc page, every comment. Sweep this repo for the banned words (`anchor`, `anchoring`,
`temporal bracket`, `temporal window`, `lower bound`, `upper bound`, `predate`,
`postdate`, and "created"/"creation time" where submission is meant) and replace them.
No em dashes in English text.

---

## 4. The API surface this CLI talks to

Authentication for the JSON API and GraphQL is `Authorization: Bearer <api key>`;
`Accept: application/vnd.api+json` and `Content-Type: application/vnd.api+json` on the
JSON:API routes. An optional `tenant: <team id>` header selects a team. Public download
surfaces need no key.

### 4.1 Generate

`POST https://www.truestamp.com/api/json/proof/generate`

```json
{"data": {"id": "01M1M0V3SE3C5P32TRAJSNX6QF", "type": "item",
          "witnesses": ["block", "entropy_nist"],
          "format": "json"}}
```

- `type` is required (no auto-detection); `id` must be a ULID for `item` and a UUIDv7
  for every other type, else 400 `id_format_mismatch`.
- `witnesses`: absent means all five names; `[]` means none; a list selects; an unknown
  name is 400 `invalid_witness` with `detail` listing the valid names: `block,
  entropy_stellar, entropy_nist, entropy_bitcoin, signing_key_event`. For entropy and
  block-like subjects only `signing_key_event` applies; other names are ignored.
- `format`: `json` (default) or `cbor`. NOT `binary`; that token is the LiveView
  download path's, and the API rejects it with 400 `invalid_format`.
- Success is HTTP 201 with `{"result": <bundle object>}` for JSON, or
  `{"result": "<standard base64 of the CBOR bytes>"}` for CBOR.
- Not yet committed to a public chain: 400 with `errors[0].meta.code ==
  "no_external_commitments"` and detail "Subject has not yet been committed to a public
  blockchain. Try again after the next epoch commit." Items commit to a Truestamp block
  within about a minute of submission and to Stellar within about five minutes; a proof
  exists only after the first public commitment.

GraphQL (`POST https://www.truestamp.com/gql`, `Content-Type: application/json`):
`query { generateProof(id: "...", type: "item", witnesses: [...], format: "cbor") }`
returns a `JsonString` scalar: the bundle JSON-encoded as a string, or the base64 CBOR
string. The bundle is identical to the REST one except `generated_at` and therefore
`signature`, which are fresh per generation.

### 4.2 Verify (server-side; not part of the independence argument)

`POST https://www.truestamp.com/api/json/proof/verify`

```json
{"data": {"proof": { ...the bundle object... }, "skip_external": false,
          "expected_hash": "<hex, optional>", "type": "item (optional assertion)"}}
```

Response HTTP 201 `{"result": {...}}` with fields: `id`, `source` (the type), `passed`
(boolean verdict), `steps` (array of `{group, status, category, message}` with status in
`pass|fail|warn|skip|info` and category in `structural|cryptographic|data_integrity|
timing|blockchain`), `temporal` (`{submitted_at, committed_at, stellar_commit,
bitcoin_commit?}`), `pass_count`, `failed_count`, `warn_count`, `skip_count`,
`info_count`, `hash_provided` (the normalized expected hash or null),
`expected_hash_provided` (boolean), `hash_matched` (boolean), `proof_version`,
`skipped_external`, `generated_at`. A malformed bundle is HTTP 400 with
`errors[0].meta.code == "invalid_proof"` and `meta.reason` one of the Section 3.9 names
(for example `unsupported_layout`). A well-formed bundle that fails verification is a 201
with `passed: false`. GraphQL: `query V($proof: JsonString!) { verifyProof(proof: $proof) }`
with the bundle JSON-encoded as the variable; returns the same result object as a
`JsonString`. The CLI's own verifier must never depend on this endpoint; offering it as
an explicit "ask Truestamp too" option is fine.

### 4.3 Machine output of the CLI's own verifier

When the CLI emits JSON for its own report, use the server's field names above
(`passed`, `steps[{group,status,category,message}]`, counts, `hash_provided`,
`expected_hash_provided`, `hash_matched`, `proof_version`, `skipped_external`) so a CLI
report and an API report are diffable. Add `verifier` (this CLI's name and version) and
`rejection` (the Section 3.9 name, when the run was a hard rejection).

### 4.4 Public download surfaces

Anonymous pages `/verify/:type/:id`, `/blocks/:id`, `/entropy/:id`, `/beacons/:hash` offer
complete and compact downloads in JSON and CBOR. Artifact filenames are
`truestamp-<type>-<id>.json` / `.cbor`, with `-compact` appended for the compact variant
and `-partial` for a subset selection. `type` inside the file is authoritative, not the
filename.

### 4.5 Other reads you may want

- `GET /api/json/items/:id` (default fields include `claims`, `claims_hash`, `metadata`,
  `metadata_hash`, `item_hash`, `signature`, `signing_key_id`, `state`, `block_id`,
  `visibility`, `tags`). `metadata` is the witness map. The item `signature` is Ed25519
  over the raw `item_hash` bytes with the same signing key.
- `GET /api/json/blocks/:id` (`block_hash`, `previous_block_hash`, `previous_block_id`,
  `merkle_root`, `metadata`, `metadata_hash`, `signing_key_id`, `signature`, `state`).
  The block `signature` is Ed25519 over the raw `block_hash` bytes.
- GraphQL `getItem`, `getBlock`, `listItems`, `listBlocks`, `me`, `serverTime`.

---

## 5. Fixtures in this repo (`prompt-fixtures/`)

I added this directory for you. Move its contents into wherever this repo keeps golden
test data, and delete the directory when you are done; the layout below is what matters.

### 5.1 `prod-2026-09-03/`: a real production item, end to end

Item `01M1M0V3SE3C5P32TRAJSNX6QF` ("Ut tensio, sic vis", a hashless claims-as-source-of-
truth item, public), submitted 2026-09-03T16:15:14.222Z, in block
`01a0680e-42b6-712a-9137-5b59dc891f19` (finalized 16:16:00Z), Stellar commitment at
16:20:05Z in ledger 64256842, transaction
`9a58b4414607648452500674e83c778567a155573205932138ef5b25955979a2`. Signing key
`3c19f776`. Head block `01a0680d-5834-7919-8248-37cc7bf79969` is the direct predecessor,
so there is no `block_path`.

| File | What it is | Expected result |
|---|---|---|
| `proof-complete.json` | complete bundle: four witnesses carried, key event carried, N = 1 | offline: PASSED, 21 pass, 0 fail, 0 warn, 6 skip, 8 info (see `standalone-complete.txt` for the exact rows); online: every Blockchain row passes |
| `proof-compact.json` | compact bundle (`witnesses: []`): no `subject.witnesses`, no `signing_key_event`, 1874 bytes | offline: PASSED, 12 pass, 7 skip, 3 info; Submitted After is the single "not established from this bundle" info row |
| `proof-partial.json` | `witnesses: ["block", "entropy_nist"]` | offline: PASSED, 15 pass, 6 skip, 5 info; the after edge rests on `block` offline and on `entropy_nist` (16:14:00Z) once confirmed |
| `proof-complete.cbor` | the complete bundle as CBOR, 2961 bytes, tag 55799, CDDL-valid | decodes to a bundle equal to `proof-complete.json` in every field except `generated_at` and `signature` (it was generated separately); verifies identically |
| `keyring.json` | production keyring | with it pinned, Key Binding is `pass` "found in the pinned keyring (sequence 0, active true)" |
| `standalone-*.txt` | the reference verifier's offline reports for the three JSON bundles, with the keyring pinned | your offline report must contain every row of these with the same status |
| `verify-*.json` | the server's online reports for the same three bundles (`skip_external: false`) | your online report must contain every row with the same status; note Key Binding is `skip` server-side (it does not pin its own keyring) |
| `created.json`, `block.json`, `headblock.json` | the item, its block, and the head block as the JSON API returned them | for unit tests of the item hash, block hash, and block witness derivations: `item_hash` `f516a382...3f89`, block hash `ac8dacef...5c1f`, head block hash equals `metadata.witnesses.block` |

Derived values you must reproduce from `proof-complete.json`: key id `3c19f776` from the
public key; `claims_hash` `300050e5b328e775a45912cfdfb41f6127b0c10d7ad99cbe0c3f5e31deb2b29e`;
`metadata_hash` `38d133efa5c3639954e0187db782d83e56655a64ede2f6a89c03050a89c5f533`;
`item_hash` `f516a3822a97ba9c22ca3e84692bb5ac3df0d15aacd5726d14951589a77f3f89`; block
metadata hash `14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1`; block
hash `ac8dacefba431b1a21f0deec4e229ab28a285a6a63d0c966d1b1c0ed2db55c1f`; inclusion proof
depth 5; epoch proof depth 3; payload 113 bytes.

### 5.2 `tamper/`: mutations of the complete bundle and what must happen

| File | Mutation | Expected |
|---|---|---|
| `tamper-claims.json` | one character changed in `claims.name` | `fail` Inclusion Proof (derived root mismatch) and `fail` Proof Signature; verdict FAILED, exit 1 |
| `tamper-witness-hash.json` | `metadata.witnesses.entropy_nist` zeroed | `fail` Inclusion Proof, `fail` Proof Signature, `fail` Witnesses (`witness_hash_mismatch`) |
| `tamper-witness-detail.json` | the NIST pulse `outputValue` altered | exactly one `fail`, Witnesses `witness_hash_mismatch`; signature still valid (metadata unchanged) |
| `tamper-root.json` | `block.merkle_root` replaced | `fail` Inclusion Proof, `fail` Epoch Proof, `fail` Proof Signature |
| `tamper-epoch-root.json` | `commitments[0].epoch_merkle_root` replaced | `fail` Epoch Proof, `fail` Proof Signature |
| `tamper-type.json` | `type` changed to `block` on an item bundle | hard rejection `unexpected_subject_fields_for_block_like` |
| `drop-key-event.json` | `signing_key_event` removed | PASSED; Signing Key Event is one `skip` |
| `old-layout.json` | `subject` removed, top-level `v: 1` and `t: 10` added | hard rejection `unsupported_layout`, advice to regenerate |

`*.server-report.json` and `*.standalone-report.txt` hold the server's and the reference
verifier's actual outputs for these.

### 5.3 `whitepaper-vectors/`: Appendix C and D

- `bundle.json` / `bundle.pretty.json`: the Appendix D worked bundle (illustrative key
  `f2c39df9`, two commitments, N = 2, payload 145 bytes, `claims.hash` present). Offline
  with the subject's own hash supplied: 22 pass, 0 fail, 0 warn, 7 skip, 9 info, per D.4
  (whitepaper lines 6696 to 6805). Its transactions do not exist on chain, so it can only
  ever be confirmed offline.
- `derivation.txt`: every intermediate value, in order. Reproduce all of them.
- `vectors.txt`: C.4 sentinels; C.2a JCS conformance vectors (each a map, its JCS string,
  and `SHA-256(0x11 || JCS)`); C.3 single-value domain hashes; C.3a witness hashes; C.5
  composite hashes and byte budgets; C.6 identifiers (ULID and UUIDv7 timestamp
  extraction); C.7 Merkle trees of 0 to 7 leaves with roots, RFC 6962 comparison roots,
  and a 300-leaf tree with a depth-10 proof. Turn every vector into a unit test.

---

## 6. What the CLI does today, and the gap

A read-only inventory of this repo taken on 2026-09-03 (verify each pointer before you
rely on it; nothing here has changed since, but line numbers drift). The short version:
the CLI speaks only the draft layout, and it has no notion of witnesses, block paths, or
signing key events, but its cryptographic core, its CBOR value-space checker, its Bitcoin
parsing, its external clients, its report model, and its conformance harness are all
reusable. This is a wire-format and verifier-logic migration, not a rewrite.

### 6.1 Data model: replace

- `internal/proof/types.go:25-45` `ProofBundle` uses `v t ts pk sig` plus hand-marshalled
  `s b cx ip` (`types.go:75-90`). Note the commitments array is `cx`, not `c`. Every one of
  these keys goes: the new keys are `version type generated_at public_key signature
  subject inclusion_proof block block_path commitments signing_key_event`.
- `types.go:97-102` `Subject{ID, Data "d", MetadataHash "mh", SigningKeyID "kid"}` becomes
  `{id, claims | entropy, metadata (a map, carried), signing_key_id, witnesses?}`. The
  `mh` field disappears: the metadata hash is derived under `0x12`/`0x22` from the carried
  map.
- `types.go:105-111` `Block{ID, "ph", "mr", "mh", "kid"}` becomes `{id,
  previous_block_hash, merkle_root, metadata (a map, carried), signing_key_id}`; `mh`
  disappears and is derived under `0x33`. The same struct serves `block`, `block_path[]`,
  `subject.witnesses.block`, and `signing_key_event.block`.
- `types.go:117-136` `ExternalCommit{Type t=40|41, "net", "ep", "tx", "memo", "l", "ts",
  "op", "rtx", "txp", "bmr", "h"}` becomes `{chain: "stellar"|"bitcoin", epoch_merkle_root,
  epoch_proof, transaction_hash?, network?, timestamp?, ledger?, block_height?, txoutproof?,
  raw_transaction?, block_merkle_root?}`. `memo` and `op` collapse into the single
  `epoch_merkle_root`; the `commitJSON` shadow struct (`types.go:142-158`) that forced
  empty `memo`/`op` out is no longer needed.
- `internal/proof/ptype/ptype.go:32-45` keeps the integer registry (the signed payload
  still needs it) and gains name-to-code and code-to-name for the six subject names;
  codes 40 and 41 stay documented as reserved and leave the wire.
- New types: the witness map (`subject.metadata.witnesses`, ordered, hex text values), the
  witness detail map (`subject.witnesses`, open: known names plus unknown ones preserved
  for the skip row), `block_path`, and `signing_key_event{block, commitments}` with the
  `key_event` object inside the block's metadata.
- Hashed maps (`claims`, `entropy`, both `metadata` maps, every block metadata map, every
  entropy witness payload) must be kept as number-preserving raw JSON, the way `RawData
  json.RawMessage` already does for `s.d` today, so JCS sees the producer's bytes.

### 6.2 Decoders: extend

- JSON: add the `unsupported_layout` gate for a top-level `v` or `t` before anything else,
  and the full Section 3.9 sequence. `internal/proof/errors.go:17-32` holds the E.6 code
  set; add `unsupported_layout`, `missing_metadata`, `invalid_subject_type`,
  `invalid_commitment_entry`, `invalid_subject_data`, `unexpected_subject_fields_for_block_like`
  as needed and retire any code tied to the old shape.
- CBOR decode, `internal/proof/binary.go` (865 lines): detection (`HasCBORTag :29`,
  `IsCBORProof :39`, tag and bare) and the decode options (`:61-64`, duplicate keys
  rejected) are right. The byte-string field list (`mh ph mr kid tx memo op bmr` at
  `:642-672`) must become: `public_key`, `signature`, every `signing_key_id`, every block
  map's `previous_block_hash` and `merkle_root`, every commitment's `epoch_merkle_root`,
  `transaction_hash`, `block_merkle_root`. `inclusion_proof` and `epoch_proof` are TEXT
  on the wire now; today the decoder accepts both (`cborStringField :695`, `:260-268`) and
  the encoder emits byte strings (`marshal_cbor.go:88`, `:239`). Emit text. Accepting a
  byte string there on decode is a leniency you may keep or drop; ask me if you want a
  ruling.
- The value-space walker `validateSubjectDataCBOR` (`binary.go:315-383`) and its raw
  locator `rawSubjectData` (`:396`) guard only `s.d` today. They must guard every hashed
  map listed in Section 3.8: `subject.claims`, `subject.entropy`, `subject.metadata`,
  `block.metadata`, each `block_path[].metadata`, `subject.witnesses.block.metadata`,
  `signing_key_event.block.metadata`, and every entropy witness payload under
  `subject.witnesses`. The server's equivalent is `validate_json_preimages/1` in
  `lib/truestamp/proof/binary.ex:237-292`.
- `internal/proof/binary_valuespace_test.go` (479 lines) and `wireform_test.go` (617
  lines) pin the old shape; rewrite them against the new fixtures.

### 6.3 Verifier: keep the primitives, rewrite the pipeline

Keep, unchanged: `internal/tscrypto/hash.go` prefixes (`:38-53`), `ComputeKeyID :127`,
`ComputeEntropyHash :133`, `ComputeObservationHash :147`, `ComputeItemHash :175`,
`ComputeBlockHash :203`, `appendLenPrefixed :363`, `BuildCompactProofPayload :263` with the
layout at `:237-247`; `internal/tscrypto/merkle.go` (`VerifyMerkleProof :17`,
`DecodeCompactMerkleProof :61`, depth cap, exact-size check, little-endian bit order);
`internal/tscrypto/signature.go`; `internal/jcs`; `internal/bitcoin/*`;
`internal/external/*` (Horizon, Blockstream, NIST, keyring, `Classify`). Add to
`hash.go`: derivations for `0x12`, `0x22` and `0x33` over JCS bytes (today they exist only
as constants, never computed).

Change in `internal/verify/verify.go` (`runBundle :65-306`):

- The hex sweep `verifyHexEncoding :531-560` sweeps the closed ten `s.mh s.kid b.ph b.mr
  b.mh b.kid cx[].tx cx[].memo cx[].op cx[].bmr`. New closed list: every `signing_key_id`,
  every block map's `previous_block_hash` and `merkle_root` (top-level block, each
  `block_path` entry, `signing_key_event.block`; the `block` witness's map is graded in
  the Witnesses step instead), every value of `subject.metadata.witnesses`, and every
  commitment's `epoch_merkle_root`, `transaction_hash`, `block_merkle_root`, in both
  `commitments` and `signing_key_event.commitments`.
- `Subject Data` (`:239-248`): insert the metadata-hash derivation between the data hash
  and the composite, with its own pass row, and feed the derived value into
  `ComputeItemHash`/`ComputeObservationHash`. Add the 2^53 portability warn if it is not
  already there (the `item-bigint.json` fixture suggests a guard exists; make it a row).
- `Block Hash` (`deriveBlockHash :794`): derive `0x33` from `block.metadata` first, with
  its own pass row. Factor the block-hash derivation so it runs on any block map; it is
  now called for `block`, `block_path[]`, the `block` witness, and the key-event block.
- `Epoch Proof` (`verifyEpochProofs :894`, `epochTarget :959`): the target is
  `epoch_merkle_root` on every entry; delete the `memo`/`op` dispatch.
- New step, `Signing Key Event` (Cryptographic), after `Key Binding` (`:285`): the five
  checks of Section 3.10, reusing the block-hash derivation, the epoch-proof walk, and the
  Stellar/Bitcoin confirmation code paths, all rows under the new group.
- New step, `Witnesses` (Timing): Section 3.10 E.17a, including the chain link and
  `block_path` walk, the unknown-name skip, the not-carried skip, the carried-but-not-
  committed fail, and the late-witness warn.
- `Entropy Source` (`verifyEntropySource :1224`): today it runs only for entropy subjects.
  It must also run once per matched carried entropy witness (rows named "<name>
  witness ..."), reusing `verifyEntropyNIST :1262`, `verifyEntropyStellar :1351`,
  `verifyEntropyBitcoin :1403`. `entropyNetwork :1253` reads the network off a `t=40`
  entry; read it off the entry whose `chain == "stellar"` instead. The always-mainnet
  choice for the Bitcoin re-fetch can stay.
- `Stellar Commitment` (`verifyStellarCommitments :1512`): field renames (`transaction_hash`,
  `epoch_merkle_root`, `ledger`, `network`); the defaulted-network 404-is-skip rule can
  stay.
- `Bitcoin Commitment` (`verifyBitcoinCommitments :1618-1885`): field renames
  (`raw_transaction`, `txoutproof`, `block_merkle_root`, `block_height`,
  `transaction_hash`, and `epoch_merkle_root` in place of `op`); the six offline steps and
  the binding lookup (`verifyBitcoinBinding :1802`) are exactly what E.19 wants. Keep the
  disclosure info row for regtest/absent network (`bitcoinNetworkDowngradeNote :1885`).
- Submission window (`verifySubjectTemporalWindow :1060`, `reportSubmissionWindowEdges
  :1124`, `addTemporalInfo :1166`): the ordering check and `Temporal Info` stay. The two
  always-`info` edge rows must become the E.20 logic: *Submitted Before* is a `pass` on the
  earliest timestamp among chains that CONFIRMED in this run, an extra `info` for an
  earlier unconfirmed candidate that never displaces it, and an `info` candidate when
  nothing confirmed; *Submitted After* is one `info` per carried witness plus the operative
  edge row (`pass` when a matched witness was source-confirmed, `info` otherwise), or the
  single "not established from this bundle" `info` for a compact bundle.
- `Hash Comparison` (`:132-201`): today a claims-only item with `--hash` yields a `skip`
  plus a `Verification Notes` info. The rule is now a `warn`, "an expected hash was
  supplied but this proof commits to no file hash". Everything else in that branch table
  matches the spec.
- `Subject Type` (`:118-122`): the `--type` assertion currently emits a `fail` row and
  continues. E.25 forbids a verifier adding a `fail` row for a step D.4 does not report,
  and the server treats a type assertion mismatch as a rejection (`subject_type_mismatch`).
  Make it a hard rejection with that name. `Verification Notes` info rows are additive and
  conformant; keep them if you like them.
- `verify.go:847-851` asserts the three preimage widths (157 / 111 / 121); those numbers
  are unchanged and should keep asserting. `framedPreimageSize :858` likewise.
- Groups (`verify.go:317-362`): add `Witnesses` and `Signing Key Event`. Categories,
  statuses, `Report.Passed`, `Verdict`, `computeResult`, and the rejection JSON shape
  (`cmd/verify.go:232-262`) are already what Section 3.11 asks for.

### 6.4 Remote mode

`internal/verify/remote.go:101` already posts `{"data":{"proof":..., "expected_hash"?,
"type"?}}` to `/proof/verify`. Add `skip_external` passthrough if you expose it. The local
E.7 re-run (`applyExpectedHash :220-289`) must adopt the hashless-item `warn` rule so it
stops disagreeing with the server, which now warns.

### 6.5 Commands

- `truestamp download` (`cmd/download.go:58-275`, `internal/proof/download.go:85-161`)
  already posts `{"data":{"id","type","format":"cbor"?}}`. Add `--witnesses all|none|<list>`
  mapped to the `witnesses` argument (absent for `all`, `[]` for `none`); append
  `-compact` / `-partial` to the default filename as Section 4.4 says. Keep the ULID-means-
  item default (`internal/proof/id.go:30`).
- `truestamp verify` (`cmd/verify.go:25-302`): flags `--hash --type --json --skip-external
  --skip-signatures --remote` map onto the spec (`--hash` is `expected_hash`). Consider
  `--keyring <file>` for an offline pinned keyring alongside the existing keyring URL
  option; today `VerifyKeyring` (`internal/external/keyring.go:149`) takes a URL.
- `truestamp convert proof` (`cmd/convert_proof.go`) round-trips through the model; it
  follows the model change. `convert merkle`, `convert keyid`, `convert id`, `convert
  time`, `jcs`, `hash` are unaffected.
- There is no `inspect` command. Section 7 item 6 asks for one; a `--summary` on `verify`
  or a `convert proof --summary` is an acceptable shape. There is no proof `redact` or
  `save` command (`internal/redact` scrubs log secrets), so nothing to migrate there.

### 6.6 Fixtures and goldens: replace wholesale

Everything under `internal/verify/testdata/fixtures/` (`appendix-d-item.json`, `item.*`,
`block.*`, `item-bigint.json`, `entropy-*.json`), `internal/verify/testdata/proof_item*`,
and `samples/*.{json,cbor}` is in the draft layout and must go; regenerate the samples
from production with the new `download`. The golden `internal/verify/testdata/golden/
appendix-d4-report.txt` (18 rows) and the `appendixD4` table in
`internal/verify/conformance_test.go:73+` must be rebuilt from the new D.4 (whitepaper
lines 6696 to 6805; 36 rows including Witnesses, Submitted After per witness, Signing Key
Event, Entropy Source per witness). The `Count` latitude there (Subject Data 2, Epoch
Proof 2) becomes Subject Data 3 (data, metadata, composite) and Epoch Proof per
commitment. `internal/verify/status_pinning_test.go` (1005 lines) pins per-branch statuses
and needs the same pass. The `fixtures/README.md` regeneration notes need rewriting.
`gen_entropy_fixtures.exs` targets the old layout; either port it or replace its outputs
with real downloaded entropy bundles (ask the user for an entropy observation id, or find
one on `https://www.truestamp.com/entropy`).

### 6.7 Docs

`kb/proof-bundle-format.md` (the primary target: short-key shapes at lines 24-57, E.6
list 71-80, hex rule 90-108), `kb/verification-steps.md` (step order and preimage
formulas), `kb/architecture.md:38` (CBOR policy), `kb/jcs-canonicalization.md`,
`kb/external-apis.md`, `kb/README.md`; `CLAUDE.md` sections "Signature Payload Format"
(92-105) and "Relationship to the Truestamp service" (106-120); `README.md` "What gets
verified" (402-432) and "Quick start" (119-239); `EXAMPLES.md` download (392-465), verify
(466-568), lifecycle (236-256); and the long Appendix E doc comments throughout the Go
files, which are spec text and need the same pass.

### 6.8 Terminology

Already clean apart from `cmd/beacon.go:31` ("proof of life" anchor), which must become
"commitment" as its mirror in `internal/verify/presenter.go:160` already says. The group
names `Temporal Info`, `Submission Window`, `Submitted Before`, `Submitted After` are E.22
names and stay exactly as they are.

---

## 7. Deliverables

Everything below, in this repo, with tests. Order is a suggestion; the acceptance
criteria in Section 8 are not.

1. **Data model.** Go types for the version 1 bundle exactly as in Section 3, with JSON
   and CBOR (de)serialization that applies the byte-string rule on encode and the
   JSON-value-space rule on decode. Keep the hashed maps as ordered, type-preserving JSON
   values (integers stay integers, big integers stay exact; never `map[string]any` with
   float64 numbers). Remove every type and code path for the draft layout.
2. **Decoders.** JSON and CBOR input, tag 55799 and bare, with the hard-rejection gate of
   Section 3.9 in order, including `invalid_subject_data` for CBOR values outside the JSON
   value space and `unsupported_layout` for `v`/`t`.
3. **Primitives.** Section 3.8, each with the Appendix C vectors as tests: prefixed
   SHA-256, JCS (with the three traps and the integer-preservation policy), `len32`,
   lowercase-hex grading, base64 variants, compact proof decoding (little-endian bit
   index, exact size, depth cap, `"AA"`), the Merkle walk, ULID and UUIDv7 millisecond
   extraction, key id derivation, constant-time comparison.
4. **The verifier.** Every step of Section 3.10 in dependency order, the report model of
   Section 3.11, offline as a first-class mode, online steps behind explicit flags with
   timeouts and `skip` on any failure to reach a source. Reuse this repo's existing
   Bitcoin parsing (btcd) for E.19's six offline steps; add the mandatory binding lookup.
5. **Report rendering.** A human report in the reference verifier's shape (five categories
   in order, `[PASS]`/`[FAIL]`/`[WARN]`/`[SKIP]`/`[INFO]` badges, group, message; counts
   line; "file hash provided: yes/no"; VERDICT with the two explanatory sentences) and a
   JSON report per Section 4.3. Exit 0/1.
6. **Commands.** `verify <file>` with `--expected-hash`, `--keyring <file>` (and a way to
   fetch and pin the live keyring), `--offline` / `--skip-external`, `--json`; a
   generate/download command taking `--type`, `--witnesses` (`all`, `none`, or a
   comma-separated list) and `--format json|cbor`, saving with the Section 4.4 filenames;
   an inspect command that prints the bundle's key fields (type, ids, witnesses carried,
   commitments, whether a key event is carried) without verifying; and whatever redact /
   save / other proof-touching commands this repo already has, updated to the new model.
   Keep existing flag names where they still mean the same thing; rename or remove the
   ones that encode the old layout.
7. **Tests.** Golden tests over every fixture in Section 5 asserting the per-group
   statuses and the verdict (not message text, except where you choose to pin the
   reference wording); the tamper table; the Appendix C vectors; CBOR-equals-JSON
   invariants; property tests for the compact proof codec (round trip, size equation,
   depth cap, bit order at depths 9 to 64); a test that the draft layout is rejected.
8. **Docs.** README, EXAMPLES.md, docs/, and this repo's `kb/` updated to the new format
   and vocabulary; a CHANGELOG entry that says plainly the draft layout is no longer
   readable and proofs must be regenerated; the terminology sweep of Section 3.14.
9. **Housekeeping.** Version bump appropriate to a breaking change; remove
   `prompt-fixtures/` after relocating its contents; leave this `prompt.md` in place until
   the user says otherwise.

---

## 8. Acceptance criteria (self-certify, then send me the results)

1. Every Appendix C vector reproduces; every value in `derivation.txt` reproduces.
2. The Appendix D bundle, offline, with `--expected-hash` equal to its `claims.hash`,
   produces a report that contains every D.4 row with the same status, and adds no
   `pass`, `fail` or `warn` row that D.4 lacks.
3. The three production bundles verify offline with reports containing every row of the
   corresponding `standalone-*.txt` with the same status, with the keyring pinned; and
   online (Stellar reachable) with every row of `verify-*.json` at the same status, Key
   Binding excepted (yours passes with the keyring, the server's skips).
4. `proof-complete.cbor` decodes to the same field values as `proof-complete.json`
   (modulo `generated_at` and `signature`) and every derived hash is identical from both.
5. Every tamper case behaves as Section 5.2 says; `old-layout.json` and `tamper-type.json`
   are hard rejections with the named error and no report.
6. A round trip: download a fresh complete bundle for item `01M1M0V3SE3C5P32TRAJSNX6QF`
   from production in both formats and with `--witnesses none`, verify each offline and
   online, and confirm the results match the fixtures' statuses. Ask the user for a key.
7. `grep` finds none of the banned terms in this repo's Go, docs, or help text.
8. `go vet`, the linter this repo uses, and the full test suite pass.

Then send me the conformance report described in Section 0.

---

## 9. Things that will bite (learned while shipping this)

- The compact proof bit order. Test at depth 9 or more.
- Integers inside hashed maps: `paging_token` is a string but `sequence`, `height`,
  `time`, `chainIndex` are integers; Go's default JSON decoding into `any` turns them into
  float64 and silently changes the JCS bytes for anything over 2^53 or with an exponent
  presentation. Use a number-preserving decoder.
- The NIST `outputValue` is uppercase hex inside a hashed payload. Never re-case it.
- `signing_key_id` on the wire versus the derived key id: the preimages use the stored
  one, the payload and keyring use the derived one. They coincide today (one key,
  sequence 0) and will diverge at the first rotation.
- `epoch_merkle_root` is one key on both chains. If you find yourself writing
  `if chain == "stellar" { root = ... } else { root = ... }` to locate the root, stop.
- `network` may be absent. Stellar falls back to the testnet Horizon; Bitcoin skips.
- Block-like bundles have no `subject`, so guard every `subject.*` access.
- The E.7 warn-not-fail rule for a hashless item with an expected hash was a live server
  defect until 2026-09-03; the fixtures' `hash_provided`/`hash_matched` semantics reflect
  the fixed behavior.
- Base64 in JSON is padded standard base64 for keys and signatures, and unpadded base64url
  for proofs. Do not unify them.
