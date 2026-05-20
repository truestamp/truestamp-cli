# Briefing: Claims-as-source-of-truth submission mode

> Companion prompt for an LLM session run inside the `truestamp/truestamp-cli`
> repository. Describes a change just shipped in the
> [`truestamp/truestamp-v2`](https://github.com/truestamp/truestamp-v2) backend
> that the Go CLI needs to catch up to. Tells you **what** changed on the
> server and **why**. Does not prescribe **how** to change the Go code — pick
> the implementation that fits the CLI's conventions best.

## 1. What changed in the backend

Until now, every Truestamp Item submission required the user to compute a
cryptographic hash of some external file on their own device and submit that
hash (plus `hash_type`) inside `claims`. The CLI's `truestamp create`
command, the JSON:API `/items` endpoint it calls, the GraphQL `create_item`
mutation, the flat webhook controller, and the console-channel
`items.create` event all assumed `claims.hash` + `claims.hash_type` were
required fields.

The backend now supports a second submission mode:

- **External-hash mode** (the existing flow). The user keeps the original
  file on their own device, computes its hash locally, and submits the
  hash. `hash` and `hash_type` are present in the claims payload. Nothing
  about this mode changes for the CLI.

- **Claims-as-source-of-truth mode** (new). The user submits an Item where
  the **claims content itself** is the data being timestamped. No external
  file exists. The proof commits to the claims bytes directly. Typical use
  cases: written invention disclosures, public statements, release notes,
  short-form factual claims, structured records that have no separate
  underlying file. In this mode `claims.hash` and `claims.hash_type` are
  absent from the submission.

## 2. Why this matters

The original "always submit a hash" framing forced friction on users whose
"data" was a sentence, a paragraph, or a small structured record. They had
to fabricate a file, hash it locally, keep that file pristine for the
lifetime of the proof, and figure out how to share it with verifiers. For
many real workflows (claim of invention, dated statement, release-note
timestamping, simple business records) that's overkill. The claims content
IS the data — committing it directly is the natural shape.

The change is intentionally not a full redesign: the cryptographic pipeline
already hashed the **entire** claims map as the Layer-1 input
(`claims_hash = SHA256(0x11 || JCS(claims))`), so submitting claims without
`hash`/`hash_type` simply changes which fields contribute to that hash. The
proof bundle format, the Merkle tree, block hashes, external commitments,
and the Ed25519 signatures are all unchanged. A proof of a claims-only Item
is byte-shape-identical to a proof of an external-hash Item; the only
difference is whether `s.d.hash` and `s.d.hash_type` are populated.

## 3. The new wire-level contract

The fields of the JSON:API / GraphQL / flat-webhook / console-channel
`create item` request payload change as follows. All four surfaces share the
same Ash `:create` action server-side, so they all behave identically.

| Field                | Before                   | After                                                           |
| -------------------- | ------------------------ | --------------------------------------------------------------- |
| `claims.name`        | required (1–64 chars)    | **unchanged** (still required)                                  |
| `claims.hash`        | required (32–128 hex)    | **optional**, but **co-required with `hash_type`**              |
| `claims.hash_type`   | required (sha256, …)     | **optional**, but **co-required with `hash`**                   |
| `claims.description` | optional (≤256 chars)    | **unchanged** (but see "Meaningful content" rule below)         |
| `claims.url`         | optional                 | unchanged                                                       |
| `claims.location`    | optional                 | unchanged                                                       |
| `claims.metadata`    | optional, plan-gated     | unchanged (but see "Meaningful content" rule below)             |
| `claims.timestamp`   | optional                 | unchanged                                                       |

### The co-required rule

Submitting exactly one of `hash` / `hash_type` (without the other) is
rejected with HTTP 422. The pair must be either both supplied or both
omitted. This is a hard rule, enforced server-side, and CLI code that fills
in a default `hash_type` whenever a hash is present (or vice versa) must
also handle the both-absent case cleanly — sending `hash_type: "sha256"`
when no hash is supplied now produces an error where it used to produce a
successful item.

The server treats empty strings and whitespace-only strings the same as
nil/absent for the purpose of this check.

### The "meaningful content" rule (claims-only mode only)

When **both** `hash` and `hash_type` are omitted, the submission must
contain at least one of:

- a `description` with at least **32** non-whitespace characters after
  trimming, OR
- a non-empty `metadata` object (any non-empty JSON object will do)

A claims-only Item containing only `name` is rejected with HTTP 422. The
purpose of this rule is to prevent meaningless empty submissions; the
specific threshold (32 chars) and the OR-with-metadata escape hatch are
intentional, so that "the claims have content" is enforced without
forcing users into a particular shape.

### The plan-gated size budget (claims-only mode only)

When in claims-only mode, the entire JCS-encoded `claims` object must fit
within a new per-plan limit: `max_claims_bytes_without_hash`. Starting
values:

| Plan         | Budget   |
| ------------ | -------- |
| Free         | 256 B    |
| Starter      | 1 KB     |
| Pro          | 4 KB     |
| Enterprise   | 16 KB    |
| Unlimited    | 16 KB    |

The budget is resolved against the **team owner's** plan, not the
submitting actor's plan. A pro-plan user submitting into a free-plan-owned
team gets the free 256 B budget. This is identical to how
`max_metadata_bytes` is resolved today; the CLI does not need to predict
the limit, but it should anticipate the error and surface it usefully.

This budget does not apply when an external `hash` is supplied (in that
mode the claims payload is effectively bounded by the existing per-field
constraints in `ClaimsType`).

### Error shape

Validation errors come back through the existing JSON:API error envelope.
The new error messages are user-facing strings:

- Co-required violation:
  `"hash is required when hash_type is supplied"` or
  `"hash_type is required when hash is supplied"`
- Meaningful-content violation:
  `"Items submitted without an external hash must include either a description of at least 32 characters or non-empty metadata."`
- Budget violation:
  `"Claims content size (<actual>) exceeds the team owner's plan limit of <max> for items without an external hash. Provide an external hash or upgrade the team owner's plan."`

Tests / golden files that check for the old "hash is required" or
"hash_type is required" errors need to be reviewed — those errors no longer
fire on the legitimate claims-only path. They still fire if exactly one of
the pair is supplied.

## 4. Cryptography is unchanged

This is a wire-shape change, not a protocol change.

- `claims_hash = SHA256(0x11 || JCS(claims))` runs over whatever claims
  were submitted. JCS canonicalization (RFC 8785) handles arbitrary
  key sets deterministically, so omitting `hash`/`hash_type` simply
  produces a smaller canonical encoding and a different `claims_hash` —
  not a malformed one.
- `metadata_hash`, `item_hash`, the Merkle tree, the block hash, the
  Ed25519 signature, the epoch external commitments — all unchanged.
- The proof bundle JSON / CBOR format is unchanged. `s.d` still carries
  the claims map. `s.d.hash` and `s.d.hash_type` may simply be absent in
  some proofs; verifiers MUST NOT require them to be present. The
  Truestamp spec at `docs/proofs.md` step 9 explicitly notes this.
- No proof version bump. Existing proofs continue to verify unchanged.

The CLI's `truestamp verify` already does the right thing here. Its
`validateClaimsHashType` step at `internal/verify/verify.go` already
early-returns when `claims.Hash == "" || claims.HashType == ""`. The
`Claims` struct in `internal/verify/report.go` uses string fields whose
zero value is `""`, which deserializes cleanly from a proof that omits
those keys. The presenter only renders the hash row when `r.Claims.Hash !=
""`. Verification of claims-only proofs should "just work"; the audit work
on the verify path is mostly confirming this and updating presentation /
test data, not changing logic.

## 5. CLI surfaces this affects (overview, not prescription)

These are the places where the CLI currently assumes external-hash mode is
the only mode. They are starting points for your investigation; you decide
the right shape of the changes.

1. **`truestamp create` (`cmd/create.go`)**. The current command requires
   `claims.hash` and `claims.hash_type` to be set before sending. Several
   code paths feed into this:

   - `autoHashFile` / `autoHashFileChecked` / `autoHashStdin` always
     produce a `hash` + `hash_type` pair from a real file (unchanged use
     case).
   - The positional-argument form `truestamp create file.pdf` auto-hashes
     and is the most common path. Stays unchanged.
   - `overlayFlags` auto-fills `hash_type: "sha256"` whenever `hash` is
     set on `claims` but `hash_type` is not. This rule is fine for the
     external-hash mode but must not fire when both are absent.
   - `validateClaims` currently rejects any submission missing `hash`,
     `hash_type`, or `name`. It needs to allow the both-absent case
     and (ideally) mirror the server's "meaningful content" rule
     locally for fast feedback. Locally enforcing the 32-char /
     non-empty-metadata rule on `validateClaims` produces a better
     error before a round-trip.
   - `--claims` / `--claims-stdin` paths (loading claims JSON from file
     or stdin) become more useful in the new mode — users who want
     claims-as-source-of-truth will typically come in through these
     paths, since they're already supplying a structured claims
     object. Existing JSON files that include `hash`/`hash_type`
     continue to work.
   - The flag-only mode `truestamp create -n "Doc" -d "long description ..."`
     becomes a new ergonomic surface — describe what shape it should
     take. (No new flag may be needed; absence of `--hash` already
     signals intent.)
   - Output presenter (`presentCreate`, `printCreateJSON`) reads
     `Hash` / `HashType` from the response. Both are already
     conditional on non-empty, so the table will degrade gracefully,
     but the JSON output currently emits them as empty strings — you
     may want to omit them entirely when absent.
   - Tests under `cmd/create_test.go` assert specific error strings
     like `"hash is required"`; those expectations now reflect only
     the partial-pair case, not the both-absent case. Audit and
     update.

2. **`truestamp console` New Item pane (`internal/console/newitem.go`)**.
   The Bubble Tea form currently has a mandatory hash input with strict
   `validateHash` enforcement. To support claims-as-source-of-truth from
   the TUI you need a path where the hash can be empty. Options
   include: a toggle ("Use claims content as source of truth"), an
   "Optional" annotation on the hash field with a longer description
   field promoted, or a separate sub-flow. Pick what fits the TUI's
   conventions. Whatever you do, the form should also validate the
   description-length rule locally before submitting so users see the
   error inline.

3. **API client (`internal/items/create.go`)**. The Go struct
   `CreateItemResponse` has `Hash` and `HashType` fields. Both stay valid;
   they'll just be empty strings for claims-only items. JSON Marshal /
   Unmarshal handles this cleanly; the question is whether the
   downstream consumers of the struct gracefully handle empty values.
   `parseResponse` already does the right thing.

4. **Help text, examples, and docs**. The CLI's own `README.md`,
   `EXAMPLES.md`, `CLAUDE.md`, and per-command `--help` strings all
   describe `create` as a hash-then-submit flow. Add the second mode
   alongside, with a worked example or two. Don't replace the existing
   examples — both modes are first-class.

5. **`truestamp verify`**. As noted in §4, the verify pipeline is
   architecturally ready. Action items here are limited to:
   - Confirm presentation degrades gracefully when `s.d.hash` / `s.d.hash_type`
     are absent.
   - Add a fixture / golden file for a claims-only proof so regression
     tests cover the path.
   - The "Confirm a downloaded file matches what you timestamped"
     recipe in `EXAMPLES.md` is intrinsically external-hash-only; a
     brief note that claims-only proofs don't need this step
     (because the data is already in `s.d`) is worth adding.

6. **Anything else that touches `claims.hash` / `claims.hash_type`**.
   Greps across the repo will find a handful of places (`cmd/convert*.go`,
   tests with hard-coded claim fixtures, etc.). Most should already work
   because Go zero-values are friendly here, but it's worth a sweep.

## 6. UX considerations

- The CLI must not silently add a `hash_type` to a claims-only submission
  (the server will reject the partial pair). The cleanest approach: only
  populate `hash_type` in the outgoing claims when `hash` is non-empty.

- Local pre-validation of the meaningful-content rule (description >= 32
  chars OR non-empty metadata) gives users instant feedback. Match the
  server's wording or paraphrase clearly. The 32-char threshold is the
  server's load-bearing constant — don't drift from it.

- The plan-gated size budget can't be enforced locally without knowing the
  team owner's plan, which the CLI generally doesn't. The right move is
  probably: don't pre-check size, just surface the server's error message
  cleanly when it comes back (HTTP 422).

- Help text should be clear that the two modes are alternatives, not
  combinatorial. A user picks one per submission.

- For scripted / CI usage, the existing positional-file form
  (`truestamp create document.pdf`) stays the most ergonomic external-hash
  entry. The claims-only mode is most naturally reached via
  `truestamp create --claims claims.json` with a JSON file that contains
  `name` + `description`, or via a flag-only form
  (`truestamp create -n "Title" -d "Long description …"`). Pick whichever
  flag-only spelling reads best.

## 7. Out of scope (no changes needed)

- Cryptographic verification logic (`internal/tscrypto`,
  `internal/verify/verify.go`'s core pipeline). Already content-agnostic.
- Proof generation, signing, encoding, CBOR / JSON conversion. All
  unchanged on the server side.
- Beacons, blocks, entropy observations. Not affected by this change.
- The flat webhook endpoint URL, JSON:API URL, GraphQL URL, headers, or
  auth. Unchanged.
- The `truestamp hash`, `truestamp jcs`, `truestamp convert`,
  `truestamp encode`, `truestamp decode` family. Pure-byte utilities;
  unaffected.

## 8. Reference material

- Server-side change: see the latest commit in
  [`truestamp/truestamp-v2`](https://github.com/truestamp/truestamp-v2)
  ("Allow items without an external hash"). Look at:
  - `lib/truestamp/items/types/claims_type.ex` — `hash` / `hash_type` made
    `allow_nil?: true`.
  - `lib/truestamp/items/validations/claims_hash_validation.ex` — adds
    the co-required rule.
  - `lib/truestamp/items/validations/validate_claims_only_content.ex` —
    new module implementing meaningful-content + size-budget rules.
  - `lib/truestamp/accounts/billing/features.ex` — new
    `max_claims_bytes_without_hash` per-plan limit.
  - `lib/truestamp_web/controllers/webhook_controller.ex` — updated
    docstring covering both modes.
  - `lib/truestamp_web/channels/console_channel.ex` — the
    `items.create` handler stops auto-filling `hash_type` when no hash
    is supplied.
- Updated server documentation:
  - `docs/proofs.md` — step 9 explicitly states verifiers must not
    require `s.d.hash` / `s.d.hash_type`.
  - `docs/webhook_api.md` — field reference now lists `hash` and
    `hash_type` as conditionally required, with a "Two submission
    modes" subsection.
  - `docs/console_channel.md` — `items.create` table marks the fields
    optional; example payloads show both modes.
  - `docs/product.md` and `docs/whitepaper/whitepaper.typ` —
    public-facing framing acknowledges both modes.

## 9. Acceptance criteria for the CLI work

A round-trip in claims-only mode should succeed:

1. `truestamp create -n "Invention" -d "On this day I claim the following novel approach as my own original work."`
   sends an Item with no `hash` / `hash_type`, the server creates it, and
   the CLI prints a successful result.
2. `truestamp create -n "Doc"` (no description, no hash) fails with a
   clear local or server-side error that names the meaningful-content
   rule (either is acceptable; local is friendlier).
3. `truestamp create --hash abc... -n "Doc"` (hash without hash_type, and
   the default-`hash_type` rule has been adjusted to fire only when a
   hash is present) still works and continues to default to `sha256`,
   matching prior behavior.
4. Downloading the resulting proof and running `truestamp verify` on
   the claims-only Item passes all the same checks as a hashed Item,
   with the hash / hash-type display row simply omitted.
5. The TUI New Item flow has a path to submit a claims-only Item.

No protocol-level changes, no spec version bump, no breaking change for
existing scripts or CI pipelines.
