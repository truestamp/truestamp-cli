# Truestamp CLI - Examples

A hands-on tour of every sub-command, followed by real-world pipeline recipes.
Every example is copy-pastable once you substitute your own paths and ids;
outputs shown here were captured from the actual binary. A literal `...`
inside a value marks a long string elided for readability.

Run `truestamp <command> --help` at any time for exhaustive flag documentation.

---

## Table of contents

- [External tools used in these examples](#external-tools-used-in-these-examples)
- [Conventions](#conventions)
- [`truestamp auth`](#truestamp-auth) - **start here: prerequisite for `create` / `download` / `beacon` / `team` / `console` / `verify --remote`**
- [`truestamp config`](#truestamp-config)
- [Lifecycle: the three-step flow](#lifecycle-the-three-step-flow)
- [`truestamp create`](#truestamp-create)
- [`truestamp download`](#truestamp-download)
- [`truestamp verify`](#truestamp-verify)
- [`truestamp team`](#truestamp-team)
- [`truestamp console`](#truestamp-console)
- [`truestamp hash`](#truestamp-hash)
- [`truestamp encode` / `truestamp decode`](#truestamp-encode--truestamp-decode)
- [`truestamp jcs`](#truestamp-jcs)
- [`truestamp convert time`](#truestamp-convert-time)
- [`truestamp convert proof`](#truestamp-convert-proof)
- [`truestamp convert id`](#truestamp-convert-id)
- [`truestamp convert keyid`](#truestamp-convert-keyid)
- [`truestamp convert merkle`](#truestamp-convert-merkle)
- [`truestamp beacon`](#truestamp-beacon)
- [`truestamp upgrade`](#truestamp-upgrade)
- [`truestamp version`](#truestamp-version)
- [Pipeline recipes](#pipeline-recipes)
- [Scripting with `--json` and `jq`](#scripting-with---json-and-jq)
- [CI / scripting conventions](#ci--scripting-conventions)
- [Offline / air-gapped use](#offline--air-gapped-use)

---

## External tools used in these examples

The examples below are written in POSIX shell syntax and compose `truestamp`
with standard Unix utilities (plus `jq` where pipeline scripting benefits
from it). **Only `truestamp` itself is required** to use the CLI; the
others appear purely because they're how you glue Unix commands together.
If a given tool isn't on your system (especially on Windows or a minimal
container), the example using it won't run, but you can substitute an
equivalent.

| Tool | Used for | Typical availability |
| --- | --- | --- |
| `cat` / `echo` / `printf` / `read` / `grep` / `cut` / `find` / `xargs` / `awk` / `for` / `while` | Shell plumbing and text manipulation | POSIX-standard. Built-in on macOS, Linux, BSD, WSL, Git-Bash on Windows |
| `curl` | Fetch remote files / proofs | Pre-installed on macOS and most Linux; install via your package manager if missing |
| `date` | Format or print the current time | POSIX-standard; GNU `date` and BSD `date` have different `-d` / `-v` flag syntax |
| `jq` | Parse and transform JSON | Third-party; [install from jqlang.org](https://jqlang.org/). Highly recommended for any scripting against Truestamp's `--json` output |

Windows users running outside WSL should install Git-Bash or a similar POSIX
shell to run the examples. PowerShell-native equivalents work too, but the
pipe / redirection syntax will differ.

If you want to run the pipelines without installing `jq`, `truestamp`'s own
`--json` + `convert` + `jcs` primitives cover most parse / transform needs.
`jq` appears in examples only because it's the industry-standard fallback.

## Conventions

Every sub-command that reads input supports the same **six modes**, plus the
Unix `-` alias for stdin:

| How | Syntax |
| --- | --- |
| Positional argument | `truestamp verify proof.json` |
| Explicit file path | `truestamp verify --file proof.json` |
| Interactive file picker | `truestamp verify --file` (no path, TUI picker opens) |
| Explicit URL | `truestamp verify --url https://example.com/proof.json` |
| Interactive URL prompt | `truestamp verify --url` (no URL, TUI prompt opens) |
| Stdin pipe | `cat proof.json \| truestamp verify` |
| Stdin (explicit `-`) | `truestamp hash -` (Unix convention) |

### `--file` / `--claims` / `--url` take an *optional* value

These flags are declared with an optional argument so that `--file` alone can
open the interactive picker. That has a consequence worth internalizing: a
space-separated value is parsed as a **positional argument**, not as the flag's
value.

`verify`, `hash`, `encode`, `decode`, `jcs` and `convert proof` all fall back to
the positional argument when `--file` / `--url` were given with no value, so
both spellings work there:

```sh
truestamp verify --file proof.json     # works (falls back to the positional)
truestamp verify --file=proof.json     # works (unambiguous)
```

**`truestamp create` has no such fallback.** Its `--claims` / `-c` and
`--file` / `-f` flags *must* use the `=` form, or the CLI will try to open an
interactive picker and ignore the path you typed:

```sh
truestamp create --claims=claims.json  # correct
truestamp create -c=claims.json        # correct
truestamp create --claims claims.json  # WRONG - opens the file picker
truestamp create -c claims.json        # WRONG - opens the file picker
```

Truly global (persistent) flags, available on every sub-command:

- `--no-color` strips ANSI (also respects `NO_COLOR=1`)
- `--config <path>` overrides the config file location
- `--no-upgrade-check` suppresses the passive "new version available" notice
- `--log-level` / `--log-file` control the shared JSON log file
- `--base-url` / `--api-key` / `--team` / `--http-timeout` are the network settings. Every service URL (API, keyring, console WebSocket, health) is derived from the single `--base-url` origin; there is no `--api-url` and no `--keyring-url`.

`--base-url` wants an **origin only** (scheme + host, optionally a port), for
example `https://www.truestamp.com` or `http://localhost:4000`. Any path,
query or fragment you append is discarded during config load, so pass the bare
origin and let the CLI compose `/api/json`, `/.well-known/keyring.json`,
`/console/websocket` and `/health` itself.

Widely-available (per-command where meaningful) flags you'll see repeatedly:

- `--json` for machine-readable output; supported by `verify`, `hash`, `encode`, `decode`, `jcs`, every `convert` sub-command, `create`, `beacon` and `team`
- `-s` / `--silent` for exit code only, no output; supported by `verify`, `hash`, `encode`, `decode`, `jcs`, every `convert` sub-command, `beacon` and `team`

`config show` renders a styled table only; it has no `--json` mode.

Configuration resolution order (highest wins):

1. Compiled defaults
2. `~/.config/truestamp/config.toml` (or `$XDG_CONFIG_HOME/...`)
3. Environment variables (`TRUESTAMP_*` prefix)
4. CLI flags

Exit code convention: **`0` = success, `1` = failure**. Specialized commands add
codes (`upgrade --check` uses 0 through 3; see its help).

---

## `truestamp auth`

**Start here.** `create`, `download`, `beacon`, `team`, `console` and
`verify --remote` all require credentials. Everything else in this document,
`hash`, `encode`, `decode`, `jcs`, `verify` (local) and every `convert`
subcommand, works without authentication and without a network. If you only
need local verification and inspection, you can skip this section.

Authentication is **OAuth-first, API-key-second**:

```sh
# Default: opens your browser for an OAuth 2.1 Authorization Code + PKCE
# flow. The resulting access + refresh token pair is stored in your OS
# keychain (0600-file fallback) and refreshed automatically, never in
# config.toml.
truestamp auth login

# Headless / CI alternative: paste a long-lived API key, stored in
# config.toml at 0600.
truestamp auth login --api-key

# Confirm the auth mode, scopes, token expiry, and the resolved team, then
# validate the credential against the API.
# Exit 0 = valid, exit 1 = missing / invalid / network.
truestamp auth status

# Revoke the OAuth session and clear it locally (idempotent).
truestamp auth logout

# Also remove the stored API key from config.toml.
truestamp auth logout --api-key
```

Precedence, highest first:

1. An **explicit** API key: the `--api-key` flag or `TRUESTAMP_API_KEY` env. Wins outright, so CI is deterministic.
2. The stored **OAuth session**, auto-refreshed.
3. An `api_key` sitting in `config.toml`.
4. Nothing. Public requests still go out; the server 401s on protected ones.

Both credential types are presented to the API as `Authorization: Bearer
<value>`. For CI, prefer the env var:

```sh
export TRUESTAMP_API_KEY=...   # no browser, no keychain, no interactive step
```

`truestamp auth status` prints the resolved **Auth Mode** (`OAuth`, `API key`,
or `none`), the API and check URLs, the team in scope, and, in OAuth mode, the
granted scopes and the access-token expiry.

## `truestamp config`

Inspect and manage the resolved configuration. Settings are merged from
compiled defaults, then the config file, then `TRUESTAMP_*` env vars, then CLI
flags, with later sources overriding earlier ones.

```sh
# Where is the config file?
truestamp config path

# Print the fully-resolved config (API key masked)
truestamp config show

# Create a default config.toml if it doesn't yet exist
truestamp config init

# Override any setting for a single invocation via an env var
TRUESTAMP_BASE_URL=https://www.truestamp.com truestamp config show

# Or via a flag
truestamp config show --base-url https://www.truestamp.com
```

`config show` renders the General block (Config File, API URL, Auth Mode, API
Key, Team, Team Name, Keyring URL, HTTP Timeout, Cosign Path) followed by the
Verification, Hash and Convert sections. Config File is the file actually in
effect, so it reflects `--config` when one was supplied.

Defaults worth knowing:

- `base_url` is `https://www.truestamp.com` (API, keyring, console and health URLs are all derived from it)
- `http_timeout` is `10s`
- `hash.algorithm` is `sha256`
- `hash.encoding` is `hex`
- `hash.style` is `gnu` (sha256sum-compatible)
- `convert.time_zone` is `UTC`
- `logging.level` is `info`, and `logging.file` defaults to `<user cache dir>/truestamp/truestamp.log`

See the full env-var reference in [CI / scripting conventions](#ci--scripting-conventions).

---

## Lifecycle: the three-step flow

The canonical Truestamp workflow is **create, download, verify**. Everything
else in the CLI supports, inspects, or extends this flow.

```sh
# 1. Create an item (hashes the file locally, submits claims to the API)
truestamp create contract.pdf
# prints an "Item Created" card: ID, Name, Hash, Visibility, and the
# public Details / Verify links

# 2. Later, after the item is committed to a block, download its proof
truestamp download 01KNN33GX5E470CB9TRWAYF9DD -o contract.proof.json

# 3. Verify the proof end-to-end
truestamp verify contract.proof.json
# walks signing key, merkle proof, block hash, commitments; exits 0 on success
```

---

## `truestamp create`

Submit a new timestamp item. Requires authentication: run
`truestamp auth login`, or set `TRUESTAMP_API_KEY` / `--api-key` for
headless and CI use.

Truestamp supports two submission modes. Both produce
byte-shape-identical proofs; the only wire-level difference is
whether `claims.hash` / `claims.hash_type` are populated.

* **External-hash mode** for files you keep on your own device.
  The file's SHA-256 is submitted; the file itself never leaves your
  machine.
* **Claims-as-source-of-truth mode** for things that don't have a
  file (statements, invention disclosures, dated facts, release
  notes). The claims content is what gets timestamped, so there's
  nothing to preserve alongside the proof.

`claims.hash` and `claims.hash_type` are a co-required pair: both present
(external-hash mode) or both absent (claims-as-source-of-truth mode). Supplying
exactly one is rejected. At the flag layer, `--hash-type` carries a default of
`sha256`, so `--hash <hex>` on its own still lands in external-hash mode, while
`--hash-type <algo>` on its own is rejected.

Remember that `--claims` and `--file` need the `=` form here; see
[Conventions](#conventions).

### External-hash mode

```sh
# Hash a file and submit in one step (filename becomes the item name,
# SHA-256 becomes the hash)
truestamp create contract.pdf

# Same, machine-readable output
truestamp create contract.pdf --json

# Pick a file interactively
truestamp create --file

# Provide the content via stdin (name required separately)
curl -fsSL https://example.com/data.bin | truestamp create -F -n "data.bin"

# Submit a precomputed claims JSON (see truestamp-v2/kb/items/submit-an-item.md for the shape)
truestamp create --claims=claims.json

# Claims via stdin
jq -c '.' claims.json | truestamp create -C

# Flag-only: provide the hash and name directly. The hash must be valid
# hex of the length --hash-type implies (64 chars for sha256).
truestamp create -n "Q1 report" \
  --hash ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad \
  --hash-type sha256

# Rich metadata
truestamp create contract.pdf \
  --name "Contract v2" \
  --description "Final signed version" \
  --url https://example.com/contract.pdf \
  --timestamp 2026-04-21T12:00:00Z \
  --location 37.7749,-122.4194 \
  --metadata '{"department":"legal","project":"alpha"}' \
  --tags legal,q2 \
  --visibility private
```

`--url` on `create` is a *claims* field (a link associated with the item), not
an input source, and it must start with `https://`. `--visibility` accepts
`private` (default), `team`, or `public`.

### Claims-as-source-of-truth mode

The claims content IS the timestamped data, so the server requires
the submission to carry meaningful content. Either:

* `claims.description` of at least 32 non-whitespace characters
  (after trimming), or
* a non-empty `claims.metadata` object.

The CLI enforces this locally before any network round-trip.

```sh
# Simplest claims-only submission: name plus a description
truestamp create -n "Invention" \
  -d "On this day I claim the following novel approach as my own original work."

# Same, machine-readable. hash / hash_type keys are omitted from the
# JSON output entirely (use jq 'has("hash")' to branch on the mode).
truestamp create -n "Invention" \
  -d "On this day I claim the following novel approach as my own original work." \
  --json

# Metadata escape hatch: no long description needed if metadata
# carries the content
truestamp create -n "Release v1.2" \
  --metadata '{"version":"1.2.0","sha":"deadbeef","notes":"..."}'

# Claims JSON file with no hash/hash_type: the server treats both as
# absent and the proof commits to the claims bytes directly
truestamp create --claims=claim-only.json
```

A claims-only `claim-only.json` looks like:

```json
{
  "name": "Invention",
  "description": "On this day I claim the following novel approach as my own original work.",
  "timestamp": "2026-05-20T15:00:00Z"
}
```

Note that `hash` and `hash_type` are absent. If your file carries
those keys, you're in external-hash mode and the values must satisfy
the usual hex/length checks.

### Common errors

| You did this                                             | You'll see                                                             |
| -------------------------------------------------------- | ---------------------------------------------------------------------- |
| `--hash-type sha256` without `--hash`                    | `claims hash is required when hash_type is supplied`                   |
| A claims file with `hash` and a blank `hash_type`        | `claims hash_type is required when hash is supplied`                   |
| No `--name`, and none derivable from the input           | `claims name is required (use --name or provide via file/auto-hash)`   |
| Claims-only with short description, no metadata          | `claims content is required: provide --description of at least 32 ...` |
| `--metadata 'not json'`                                  | `--metadata must be valid JSON: ...`                                   |
| `--visibility` outside the allowed set                   | `--visibility must be private, team, or public, got "..."`             |
| Claims-only exceeding the plan byte budget               | a server-side error naming the team owner's plan limit                 |

Every row except the last is caught client-side before any network
round-trip. The plan-budget error comes back from the server, because
the CLI can't predict the team owner's plan locally.

---

## `truestamp download`

Fetch a proof bundle for an already-committed subject. `--type` declares
which kind of proof you want; there is no auto-detection at the server,
so every UUIDv7 id needs an explicit `--type`. ULID ids default to
`--type item` client-side (the only zero-flag shortcut).

The six `--type` values map 1:1 to server subject codes:

| `--type` | Wire value | Returned `t` | Filename stem |
| --- | --- | --- | --- |
| `item` | `item` | 20 | `item` |
| `entropy_nist` | `entropy_nist` | 30 | `entropy-nist` |
| `entropy_stellar` | `entropy_stellar` | 31 | `entropy-stellar` |
| `entropy_bitcoin` | `entropy_bitcoin` | 32 | `entropy-bitcoin` |
| `block` | `block` | 10 | `block` |
| `beacon` | `beacon` | 11 | `beacon` |

Flag values use **underscores**; generated filenames use **hyphens** (the CLI
translates between them so filenames stay readable). `--type entropy-nist`,
bare `--type entropy` and `--type auto` are all rejected.

```sh
# ULID id, defaults to --type item, produces truestamp-item-<ulid>.json
truestamp download 01KNN33GX5E470CB9TRWAYF9DD

# Same, with an explicit type (identical behaviour)
truestamp download --type item 01KNN33GX5E470CB9TRWAYF9DD

# Override the auto-generated filename
truestamp download 01KNN33GX5E470CB9TRWAYF9DD -o contract.proof.json

# CBOR: smaller, deterministic, ideal for embedding in another file
truestamp download 01KNN33GX5E470CB9TRWAYF9DD -f cbor -o contract.proof.cbor

# Entropy proof: UUIDv7 ids require an explicit --type (three subtypes)
truestamp download --type entropy_stellar 019cf813-99b8-730a-84f1-5a711a9c355e
truestamp download --type entropy_nist    019cf813-99b8-730a-84f1-5a711a9c355e
truestamp download --type entropy_bitcoin 019cf813-99b8-730a-84f1-5a711a9c355e

# Block proof (t=10) for a committed block
truestamp download --type block 019db7cd-efc0-7196-b763-682a84d71919

# Beacon proof (t=11) for the same block: structurally identical to a
# block proof but carries a distinct type code and a different signature
# (the `t` byte is part of the signing payload). The CLI downloads a
# self-describing t=11 bundle.
truestamp download --type beacon 019db7cd-efc0-7196-b763-682a84d71919

# Resulting files (default naming: truestamp-<stem>-<id>.<ext>):
#   truestamp-item-01KNN33GX5E470CB9TRWAYF9DD.json
#   truestamp-entropy-stellar-019cf813-99b8-730a-84f1-5a711a9c355e.json
#   truestamp-block-019db7cd-efc0-7196-b763-682a84d71919.json
#   truestamp-beacon-019db7cd-efc0-7196-b763-682a84d71919.json
```

The id shape is validated before the network call. `--type item` requires a
ULID; every other type requires a UUIDv7. Mismatches fail fast:

```sh
truestamp download --type item 019db7cd-efc0-7196-b763-682a84d71919
# --type item requires a ULID id (e.g. 01KNN33GX5E470CB9TRWAYF9DD); got a UUIDv7

truestamp download 019db7cd-efc0-7196-b763-682a84d71919
# --type is required for UUIDv7 ids (entropy, block, and beacon all use UUIDv7).
# One of: entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon
```

UUIDv7 ids are ambiguous, because entropy observations, blocks, and beacons
all use UUIDv7, so the CLI cannot infer what you want. Use
`truestamp beacon by-hash <hash>` first if you only have a hash and need the id.

---

## `truestamp verify`

Verify the full cryptographic chain: signing key, claims hash, item hash,
Merkle proof, block hash, then the Stellar / Bitcoin commitments.

```sh
# Local file
truestamp verify contract.proof.json

# URL, auto-detected from the positional argument
truestamp verify https://example.com/proof.json

# Stdin pipe
cat contract.proof.json | truestamp verify

# Interactive picker
truestamp verify --file

# Pin the expected subject hash and fail if it doesn't match
truestamp verify contract.proof.json \
  --hash e08764deac64ca9a1046901c5b23674941f1e86f0e2d0429ee07c5e311a15ce7

# The subject type always comes from the bundle's own signed `t` field.
# The filename is never consulted: a file named truestamp-beacon-<id>.json
# may legitimately carry a t=10 block proof, and renaming a file can
# never change a verdict.
truestamp verify truestamp-beacon-019d....json          # reports whatever t says

# Pass --type to additionally assert which type you expected. The verify
# fails if the bundle's t disagrees. Useful as a guard when you fetched a
# proof for a specific subject and want to be told if you got another.
truestamp verify --type beacon truestamp-beacon-019d....json
truestamp verify --type item   truestamp-item-01K....json
truestamp verify --type entropy_stellar truestamp-entropy-stellar-019c....json

# Skip the public-blockchain checks (offline / restricted networks)
truestamp verify contract.proof.json --skip-external

# Skip all signature verification (structural check only)
truestamp verify contract.proof.json --skip-signatures

# Script-friendly modes
truestamp verify contract.proof.json --json        # structured output
truestamp verify contract.proof.json --silent      # exit code only
```

`--hash` takes the hex digest the proof commits to for the subject, that is,
the value the bundle carries at `.s.d.hash` for an external-hash item. It is
compared byte-for-byte and reported under a `Hash Comparison` step.

The rendered report groups its steps into Proof, Item Claims, Timeline,
Commitments, Verification Notes, Issues and Verification Summary blocks.
Under `--json` the top-level keys are `result`, `summary`, `steps`,
`subject`, `subject_id`, `subject_type`, `timeline`, `commitments`,
`issues`, `hash_comparison`, `signatures_checked` and `verification_notes`.
Each entry in `steps` carries `group`, `category`, `status` and `message`.

Use `--remote` to delegate verification to the Truestamp server (requires
authentication: `truestamp auth login` or `TRUESTAMP_API_KEY`). Local
verification is the default and needs no credentials.

Delegating does not mean trusting blindly. Even in `--remote` mode the CLI
still does four things itself:

- **Parses the bundle first**, so a structurally invalid file is refused locally and never posted.
- **Asserts `--type` locally**, then forwards `data.type` only when the assertion already holds. A mismatch is reported as a `Subject Type` failure alongside the server's own steps; forwarding it instead would make the server answer 4xx and you would get an error string with no report at all.
- **Performs the `--hash` comparison itself**, treating the server's `hash_matched` as corroboration and failing on a disagreement in either direction. The client holds both values, so a server that ignored `expected_hash` cannot turn a mismatch into a pass.
- **Fails closed on a report it cannot read.** A step with no `status`, a status outside the five this verifier knows, a result carrying zero steps, or a server verdict that contradicts its own step list. Each is reported under a `Server Verdict` row rather than silently scored as passing.

### Structurally malformed bundles

A bundle that is malformed at the structural level is refused before any
check runs, so there is no report to render. Under `--json` the refusal is
reported as a stable identifier rather than an English sentence, so two
independent verifiers can be compared on it:

```sh
echo '{"v":1,"t":99}' | truestamp verify --json
```

```json
{
  "result": "rejected",
  "rejection": {
    "code": "invalid_subject_type_code",
    "detail": "invalid subject type code: 99"
  }
}
```

Exit code `1`. The identifiers are `not_a_json_object`, `missing_type_code`,
`invalid_subject_type_code`, `missing_block`, `no_external_commitments`,
`invalid_external_commitment_entry`, `unexpected_subject_fields_for_block_like`,
`missing_subject`, `missing_inclusion_proof` and `invalid_subject_data`.

```sh
# Branch on the reason in a script
code=$(truestamp verify proof.json --json 2>/dev/null | jq -r '.rejection.code // empty')
[ -n "$code" ] && echo "bundle refused: $code"
```

---

## `truestamp team`

Discover, switch between, and persist the active team the CLI sends as the
multitenancy context. The id is stored under the top-level `team` key in
`config.toml`, so it applies across invocations. Requires authentication.

```sh
# List every team you're a member of (no subcommand is an alias for `list`)
truestamp team
truestamp team list

# Detail card for the active team, or a specific one by id
truestamp team show
truestamp team show 019de4e3-3150-7e29-aa6c-20d93f50e24e

# Create a team. With no name on a TTY, an interactive prompt opens.
truestamp team create "Acme Legal"
truestamp team create "Acme Legal" --ownership-model team_retains --set
truestamp team create -n "Acme Legal" --json

# Set the active team. The id is read back from the API before it is
# persisted, so a typo or a revoked membership refuses to write.
truestamp team set 019de4e3-3150-7e29-aa6c-20d93f50e24e

# No id on a TTY opens an interactive picker; Esc cancels without writing
truestamp team set

# Clear the active team (the server falls back to your personal team)
truestamp team unset
```

`team list` renders a four-column table (`active marker`, `NAME`, `ROLE`,
`TEAM ID`) with a star on the active row. `team show` adds Personal,
Ownership, Created and a public Details link.

`--ownership-model` accepts exactly `creator_retains` (the default) or
`team_retains`; anything else is rejected client-side. `team_retains`
requires a plan entitlement, which only the server can evaluate, so the
CLI attempts the create and renders the server's own error when the
entitlement or the team-count limit blocks it.

All five subcommands accept `--json` and `-s` / `--silent`.

---

## `truestamp console`

Interactive Bubble Tea TUI backed by an authenticated WebSocket to the
Truestamp server. Four panes share one long-lived connection:

- **Monitor** with toggleable subscriptions to live event streams (block
  lifecycle, internal/external commitments, NIST/Stellar/Bitcoin
  entropy observations, item events for your team) plus a scrollable,
  reversible event waterfall. Newest at top by default.
- **New Item**, a form (name, description, hash, hash type) that
  creates a timestamped item over the same socket and shows its live
  state transitions (`item.created`, `item.updated`, `item.committed`)
  below the card as they arrive.
- **Teams**, the same membership table `truestamp team list` renders,
  with `enter` to switch the active team over the live socket and `c` to
  open a create-team modal.
- **Connection** with a scope summary, push counts by event, reconnect
  history, health-target status, and the log file path so you can
  `tail -f` it for live transport diagnostics.

Requires authentication (run `truestamp auth login` first, or set
`TRUESTAMP_API_KEY`; everything in the console talks to the same endpoint as
the JSON:API).

```sh
# Launch (uses your configured base_url + credentials)
truestamp console

# Point at a non-default backend (e.g. local dev)
truestamp console --ws-url ws://localhost:4000/console/websocket

# Crank the log file to debug if something goes wrong
truestamp console --log-level debug

# Send the log somewhere other than the default cache dir
truestamp console --log-file /tmp/truestamp-console.log
```

`--ws-url` is the only console-specific flag. `--log-level` and `--log-file`
are global flags that every subcommand honors; the console simply writes more
to that shared file. With `--ws-url` unset, the WebSocket URL is derived from
`--base-url` as `ws(s)://<host>/console/websocket`.

### Keys

Press `?` at any time for the live, authoritative key list. The footer and
help overlay are generated from the same bindings the app consumes, so they
cannot drift from the table below.

| Key                     | Action                                                               |
| ----------------------- | -------------------------------------------------------------------- |
| `1` / `2` / `3` / `4`   | Jump to Monitor / New Item / Teams / Connection                      |
| `]` / `[`               | Next / previous pane (also `ctrl+tab` / `ctrl+shift+tab`)            |
| `?`                     | Toggle the full help overlay                                         |
| `q` / `ctrl+c`          | Quit                                                                 |
| **Monitor pane:**       |                                                                      |
| `left` / `right` (`h`/`l`) | Switch focus between the Streams list and the Events waterfall    |
| `up` / `down` (`k`/`j`) | Move within the focused side                                         |
| `space`                 | Toggle the cursor stream's subscription on/off (Streams list focused)|
| `pgup` / `pgdn`         | Page through the waterfall                                           |
| `g` / `G` (`home`/`end`)| Jump to top / bottom of the waterfall                                |
| `r`                     | Reverse chronological order (newest first or oldest first)           |
| `d`                     | Toggle the event detail panel                                        |
| **New Item pane:**      |                                                                      |
| `tab` / `shift+tab`     | Move between form fields (plain `tab` is pane-local, not pane-switch)|
| `enter`                 | Advance, then submit on the last field                               |
| `esc`                   | Clear the form                                                       |
| `n` (after submit)      | Reset for another item                                               |
| **Teams pane:**         |                                                                      |
| `up` / `down` (`k`/`j`) | Move through the memberships table                                   |
| `enter`                 | Set the team under the cursor as active                              |
| `c`                     | Open the create-team modal                                           |
| `r`                     | Refresh the membership list                                          |

### What you see by default

On launch every catalog stream is auto-subscribed, so events start
flowing immediately. The header right side shows the current
connection state and the server-time clock. The footer shows the
context-relevant key hints for the active pane.

The Monitor pane's left column shows each stream as `[x] <id>` when
active, `[ ] <id>` when inactive, and `[~] <id>` while a
subscribe/unsubscribe is in flight. Toggling with `space` sends the
request over the WebSocket; the optimistic local state flips
immediately and the server's reply reconciles any rejection.

The waterfall itself is a three-column table: `Time`, `Kind`, `ID`.
Press `d` to open the detail panel for the selected row and see the
full payload.

### Bursts (server-side coalescing)

When the server sees many events of the same stream within a 500 ms
window (the typical case during a block close, when thousands of
items and commitments fan out at once), it coalesces them into a single
`<resource>.burst` summary push, for example `item.burst` or
`entropy.bitcoin.burst`. The waterfall renders that as one row whose
`ID` column holds the plain integer event count; the window and the
per-kind breakdown live in the detail panel (`d`).

Slow streams (blocks, entropy, external commitments) almost never
trigger this. The first-event-immediate rule means a burst only
emerges when input rate genuinely warrants summarization.

### Reconnect

If the network blips or the server restarts, the client reconnects
automatically with exponential backoff (`1s`, `2s`, `5s`, `10s`, `30s`,
capped). The header shows a live countdown such as `reconnecting in 7s
(attempt 4)`, and `server.down` markers drop into the waterfall every
10 seconds during the outage so you can scroll back later and see
exactly when data went missing. On reconnect, all previously active
subscriptions are re-issued automatically.

### Logs

Transport diagnostics (read EOFs during a server restart, dial
attempts during reconnect, frame decode errors) write to a rotated
JSON-lines log file rather than the UI. It is the same file every
other subcommand logs to:

```sh
# macOS
tail -f ~/Library/Caches/truestamp/truestamp.log | jq .

# Linux
tail -f ~/.cache/truestamp/truestamp.log | jq .
```

Defaults: 10 MB rotation, 14-day retention, 5 backups, gzip-compressed.
The Connection pane shows the live path. **Credentials are redacted**
before any error or log line touches the file or screen, covering the
API key, OAuth access and refresh tokens, and the PKCE code and
verifier.

### Hand-rolled testing

The wire protocol is plain JSON arrays over Phoenix Channels V2 and
is fully driveable from `websocat` / `wscat`. See
[kb/console/websocket.md](kb/console/websocket.md) for the
client-side architecture details and
[truestamp-v2/kb/api/console-websocket.md](https://github.com/truestamp/truestamp-v2/blob/main/kb/api/console-websocket.md)
for the authoritative wire protocol reference (commands, events,
catalog, limits, hand-rolled testing recipe).

---

## `truestamp hash`

Multi-algorithm digest tool. Default output is **byte-identical to
`sha256sum`** so it drops into existing scripts.

```sh
# SHA-256 a file (default algorithm)
truestamp hash contract.pdf
# ba7816bf8f01cfea...ad  contract.pdf

# Multiple files at once
truestamp hash a.bin b.bin c.bin

# Stdin (filename shows as "-", matching sha256sum)
echo -n "abc" | truestamp hash
# ba7816bf...ad  -

# Same, via the explicit Unix "-" alias
echo -n "abc" | truestamp hash -

# List supported algorithms
truestamp hash --list

# Pick any supported algorithm
truestamp hash -a sha3-256 contract.pdf
truestamp hash -a blake2b-512 contract.pdf
truestamp hash -a md5 contract.pdf             # warns: legacy algorithm

# BSD-style tagged output (shasum --tag compatible)
truestamp hash -a sha256 --style bsd contract.pdf
# SHA256 (contract.pdf) = ba78...ad

# Bare digest (no filename, no separator)
truestamp hash -a sha256 --style bare contract.pdf

# GNU binary mode: "<hex> *<filename>" instead of two spaces
truestamp hash --binary contract.pdf

# Drop the filename column entirely from gnu/bsd output
truestamp hash --no-filename contract.pdf

# Pick the output encoding (default: hex)
truestamp hash -a sha256 --encoding base64 contract.pdf
truestamp hash -a sha256 --encoding base64url contract.pdf

# Apply an RFC 8785 JCS canonicalization before hashing (input must be JSON)
truestamp hash --jcs -a sha256 < claims.json

# Prepend a single domain-separation byte before hashing
truestamp hash --prefix 0x11 < payload.bin

# The Truestamp "claims_hash" one-liner: SHA256(0x11 || JCS(claims))
truestamp hash --prefix 0x11 --jcs -a sha256 --style bare < claims.json

# JSON output with all three digest encodings
truestamp hash -a sha256 --json contract.pdf
```

`--list` prints 14 algorithms with their digest sizes, any aliases, and a
`[legacy]` marker on `md5` and `sha1`:

```
md5            16 bytes  [legacy]
sha1           20 bytes  [legacy]
sha224         28 bytes
sha256         32 bytes
sha384         48 bytes
sha512         64 bytes
sha3-224       28 bytes  (aliases: sha3_224)
sha3-256       32 bytes  (aliases: sha3_256)
sha3-384       48 bytes  (aliases: sha3_384)
sha3-512       64 bytes  (aliases: sha3_512)
blake2s-256    32 bytes  (aliases: blake2s)
blake2b-256    32 bytes
blake2b-384    48 bytes
blake2b-512    64 bytes  (aliases: blake2b)
```

Selecting `md5` or `sha1` writes a one-line notice to stderr
(`warning: md5 is cryptographically broken and unsuitable for security uses`).
It is suppressed under `--json` and `--silent`.

`--json` emits the digest in all three encodings alongside the input
description:

```json
{
  "algorithm": "sha256",
  "digest": {
    "hex": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
    "base64": "ungWv48Bz+pBQUDeXa4iI7ADYaOWF3qctBD/YfIAFa0=",
    "base64url": "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0"
  },
  "encoded": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
  "encoding": "hex",
  "size_bytes": 3,
  "input": { "type": "file", "path": "abc.txt" }
}
```

---

## `truestamp encode` / `truestamp decode`

Pipe-friendly byte-encoding primitives. `encode` takes raw bytes and produces
text; `decode` takes text and produces raw bytes. Both support text-to-text
conversion via `--from` and `--to`.

Supported encodings: `binary`, `hex`, `base64`, `base64url`.
`encode` defaults to `--from binary --to hex`; `decode` defaults to
`--from hex --to binary`.

```sh
# Encode raw bytes to hex (default)
echo -n "hello" | truestamp encode
# 68656c6c6f

# Encode a file as base64url
truestamp encode --to base64url contract.pdf > contract.b64u

# Decode base64 back to binary
echo "SGVsbG8=" | truestamp decode --from base64
# Hello

# Text-to-text: hex to base64 without an intermediate binary file
echo "68656c6c6f" | truestamp encode --from hex --to base64
# aGVsbG8=

# JSON envelope for scripting (pretty-printed, 2-space indent)
echo -n "hello" | truestamp encode --to hex --json
```

```json
{
  "from": "binary",
  "to": "hex",
  "input_bytes": 5,
  "output_bytes": 10,
  "output": "68656c6c6f"
}
```

`base64url` output is unpadded; `decode --from base64url` accepts either the
padded or the unpadded form and tolerates a trailing newline.

---

## `truestamp jcs`

Apply **RFC 8785 JSON Canonicalization**. Output is the byte-stable form that
Truestamp uses when computing claims / entropy / metadata hashes.

```sh
# Canonicalize (sorts keys, normalizes whitespace and number formatting)
echo '{"b":2,"a":1,"c":[3,1,2]}' | truestamp jcs
# {"a":1,"b":2,"c":[3,1,2]}

# Append a trailing newline (for appending to a stream)
truestamp jcs --newline < claims.json

# JSON envelope
truestamp jcs --json < claims.json
```

### Large integers

Truestamp emits integers at full precision, so canonicalization preserves an
integer literal exactly as written rather than round-tripping it through an
IEEE-754 double the way a strict RFC 8785 reading would. Without that, every
integer above 2^53 would silently change and the recomputed hash would be
wrong.

```sh
printf '{"n":9007199254740992}' | truestamp jcs
# {"n":9007199254740992}

printf '{"n":9007199254740993}' | truestamp jcs
# {"n":9007199254740993}
# stderr: warning: preserved 1 integer literal(s) larger than 2^53,
#         e.g. 9007199254740993; this JSON is not portably verifiable by a
#         strict RFC 8785 implementation
```

The warning is advisory. The digest is the correct one and the exit code
stays `0`. It goes to stderr, so pipelines are unaffected. `--silent`
suppresses it; `--json` replaces it with an `oversized_integers` array:

```sh
printf '{"n":9007199254740993}' | truestamp jcs --json | jq .oversized_integers
# [ "9007199254740993" ]
```

`truestamp hash --jcs` reports the same signal (labelled with the filename
when several inputs are hashed, and as a per-input `oversized_integers` field
under `--json`), and `truestamp verify` surfaces it as a warning row on the
Subject Data step.

---

## `truestamp convert time`

Bidirectional time-format tool. Replaces most uses of `date` for parsing,
reformatting, and zone conversion. Accepts RFC 3339 or Unix seconds /
milliseconds / microseconds / nanoseconds; auto-detects by default.

```sh
# Current time in UTC
truestamp convert time now

# Unix seconds to RFC 3339 UTC
truestamp convert time 1700000000
# 2023-11-14T22:13:20Z

# Convert to another time zone
truestamp convert time 1700000000 --to-zone America/New_York
# 2023-11-14T17:13:20-05:00

# Force the interpretation of numeric input
truestamp convert time 1700000000000 --from unix-ms
truestamp convert time 1700000000000000000 --from unix-ns

# Change the output format
truestamp convert time "2026-04-21T12:00:00Z" --format unix-s
truestamp convert time "2026-04-21T12:00:00Z" --format unix-ms
truestamp convert time "2026-04-21T12:00:00Z" --format "2006-01-02 15:04:05"  # Go layout

# Read the timestamp from stdin (useful in pipelines)
date -u +%s | truestamp convert time --to-zone Asia/Kolkata

# JSON output with all representations
truestamp convert time 1700000000 --json
```

`--from` accepts `auto` (default), `rfc3339`, `unix-s`, `unix-ms`, `unix-us`
and `unix-ns`. `--format` accepts the same Unix variants plus `rfc3339`
(default) or any Go time layout. The `--json` envelope carries `input`,
`parsed_utc`, `output_zone`, `output`, `unix_s` and `unix_ms`.

---

## `truestamp convert proof`

Convert a proof bundle between JSON and CBOR. The CBOR output uses **RFC 8949
section 4.2 core deterministic encoding** and is prefixed with the
self-describing tag 55799 so `truestamp verify` auto-detects it.

```sh
# JSON to CBOR
truestamp convert proof --to cbor proof.json > proof.cbor

# CBOR to JSON (auto-detected input format)
truestamp convert proof --to json proof.cbor | jq .

# Force the input format (error out if the bytes don't match)
truestamp convert proof --from json --to cbor < proof.json

# Compact JSON (minified; the default is a 2-space indent)
truestamp convert proof --to json --compact proof.cbor

# Round-trip verification (the CBOR output must verify end-to-end)
truestamp convert proof --to cbor proof.json | truestamp verify --skip-external
```

`--to` is required. `--from` accepts `auto` (default), `json` or `cbor`.

---

## `truestamp convert id`

Extract the embedded timestamp from a ULID (item IDs) or UUIDv7 (block and
entropy IDs). Truestamp uses ULIDs for item-style subjects and UUIDv7 for
blocks and entropy observations.

```sh
# ULID to embedded timestamp (UTC)
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD
# 2026-04-07T23:05:39.493Z

# UUIDv7 to timestamp
truestamp convert id 019cf813-99b8-730a-84f1-5a711a9c355e

# Convert to a specific zone
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD --to-zone Local

# Force the parser if you have an unusual form (hyphenless UUID, etc.)
truestamp convert id 019d6a3213e672b097e53779231ea97b --type uuid7

# Extract raw bytes as hex instead of the timestamp
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD --extract raw

# JSON output with every representation
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD --json
```

`--type` accepts `auto` (default), `ulid` or `uuid7`. `--extract` accepts
`time` (default) or `raw`.

---

## `truestamp convert keyid`

Derive the 4-byte Truestamp **key fingerprint** (`kid`) from an Ed25519 public
key. Formula: `truncate4(SHA256(0x51 || pubkey))`. Useful for confirming which
signing key a proof was issued under.

```sh
# Standard base64 Ed25519 public key to an 8-char hex kid
truestamp convert keyid CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=
# 4ceefa4a

# Hex input (the same key, hex-encoded, yields the same kid)
truestamp convert keyid --from hex \
  093c0ca836673ddfd04cb4aaf1a4de483ddafa3d8343129c19f861218250eb96
# 4ceefa4a

# Auto-detect the encoding (hex / base64 / base64url)
truestamp convert keyid <pubkey-from-anywhere>

# From stdin
jq -r .pk proof.json | truestamp convert keyid
```

`--from` accepts `auto` (default), `hex`, `base64` or `base64url`.

---

## `truestamp convert merkle`

Decode a compact base64url Merkle proof (the `ip` field of an item proof or
the `ep` field inside each `cx` commitment) into a human-readable sibling list.

```sh
# Positional argument
truestamp convert merkle "BAKbGnC2S9wB-uoc..."

# From stdin (common when fed from jq)
jq -r .ip proof.json | truestamp convert merkle

# JSON envelope (depth, siblings with position + hash)
jq -r .ip proof.json | truestamp convert merkle --json
```

The text form prints `depth: N` followed by one indexed line per sibling with
its `left` / `right` position and hex hash. The `--json` form carries
`compact_base64url`, `depth`, and a `siblings` array of
`{position, hash_hex}` objects.

---

## `truestamp beacon`

Inspect Truestamp block beacons via the read-only JSON:API at
`/api/json/beacons/*`. A **beacon** is a compact projection of a
finalized block, four fields only: `{id, hash, timestamp,
previous_hash}`. It's a "proof of life" commitment: every item and
entropy observation finalized inside that minute window is covered by
the beacon's hash.

Full verifiable proof bundles for a beacon are a separate artefact
fetched via `truestamp download --type beacon <id>` (see above).

```sh
# Current head beacon (no subcommand is an alias for `latest`)
truestamp beacon
truestamp beacon latest

# Most-recent N beacons, newest first (default 25, server caps at 100)
truestamp beacon list
truestamp beacon list --limit 3

# Look up by UUIDv7 id
truestamp beacon get 019db8b5-90a1-7015-a62c-48e5038f2306

# Look up by 64-hex-char hash, useful when all you have is a hash
# (e.g. printed on a receipt or read from photo metadata)
truestamp beacon by-hash 9f0be4446c4bfb8faa9a13766e3635b2c27913f35cec4eafcc20cd10af663feb
```

Shared flags (all four subcommands): `--json` (raw JSON, pipeline
friendly), `--hash-only` (prints just the hash + newline for shell
substitution), `--silent` / `-s` (exit code only). `--silent` +
`--json`, `--silent` + `--hash-only`, and `--json` + `--hash-only` are
mutually exclusive. `--hash-only` is not valid on `beacon list` (a list
has no single hash).

The single-beacon card prints Hash, Timestamp (with a relative age), ID,
Previous, and two shareable public-web links (Details and Verify).
`beacon list` renders a `TIMESTAMP` / `HASH` / `ID` table.

```sh
# Capture the current chain head as a moment-in-time commitment
MOMENT=$(truestamp beacon --hash-only)
echo "beacon hash: $MOMENT"
# beacon hash: 9f0be4446c4bfb8faa9a13766e3635b2c27913f35cec4eafcc20cd10af663feb

# Pipeline-friendly JSON (a top-level array for `list`, an object otherwise)
truestamp beacon list --limit 10 --json | jq -r '.[].hash'

# Round-trip: id to hash to id (demonstrates by-hash lookup)
ID=$(truestamp beacon latest --json | jq -r .id)
HASH=$(truestamp beacon get "$ID" --hash-only)
truestamp beacon by-hash "$HASH" --json | jq .id
```

Client-side validation catches obvious typos without hitting the
network:

```sh
truestamp beacon get not-a-uuid
# invalid UUID: uuid: incorrect UUID length 10 in string "not-a-uuid"

truestamp beacon by-hash ABCDEF
# hash must be 64 lowercase hex characters, got "ABCDEF"
```

All beacon subcommands require authentication. Run `truestamp auth login`,
or set `TRUESTAMP_API_KEY` / `--api-key` for headless and CI use. Without a
credential the command prints a "Not authenticated" banner to stderr and
exits `1`.

---

## `truestamp upgrade`

Self-upgrade the binary. The command is **install-method aware**. Homebrew
users are shown `brew upgrade`, `go install` users are shown the
`go install ...@latest` incantation, and `install.sh` users get a native-Go
in-place upgrade that mirrors the install script (SHA-256 mandatory, cosign
best-effort, atomic rename with timestamped backup).

```sh
# Check if an upgrade is available; exit 0 up-to-date, 1 available, 2 network,
# 3 pre-release latest
truestamp upgrade --check

# Perform the upgrade (install.sh-installed binaries only)
truestamp upgrade

# Skip the confirmation prompt
truestamp upgrade --yes

# Pin to a specific version (the opt-in path for pre-releases)
truestamp upgrade --version v0.5.0
```

`truestamp version` reports the detected install method, which is what
decides between the "print instructions" and "replace in place" paths.

---

## `truestamp version`

```sh
# Detailed build info, one label per line: version, module path, config
# path, detected install method, Go toolchain + OS/arch, commit, build date
truestamp version

# One-liner: version plus the short commit and the build date
truestamp --version
# truestamp <version> (commit: <short-sha>, built: <rfc3339>)
```

---

## Pipeline recipes

Real-world compositions that solve a specific problem.

### Reproduce the protocol's `claims_hash` intermediate

The Truestamp protocol computes an internal intermediate,
`claims_hash = SHA256(0x11 || JCS(claims))`, while deriving the item_hash
that gets signed into the block. The value itself is **not serialized** into
the proof bundle (the proof stores `s.d`, the raw claims, and `s.mh`, the
*metadata*_hash under prefix `0x12`, which is a different value).

If you want to reproduce or audit that intermediate against a claims JSON
document:

```sh
truestamp hash --prefix 0x11 --jcs \
  -a sha256 --style bare \
  < claims.json
```

Or the explicit two-step form (useful when you want to inspect the
canonical JSON bytes in between):

```sh
truestamp jcs < claims.json \
  | truestamp hash --prefix 0x11 -a sha256 --style bare
```

**To confirm that a file matches what was timestamped**, you don't need this
intermediate. Compare the file's plain SHA-256 against `s.d.hash`, or
pass `--hash <sha256-hex>` to `truestamp verify` (see the next two recipes).

### Convert a JSON proof to CBOR and verify

```sh
truestamp convert proof --to cbor proof.json \
  | truestamp verify --skip-external
```

### Derive a kid from a proof's embedded public key, without `jq`

Canonicalize first so the `"pk"` field is guaranteed to be compact, then
slice it out with plain `grep` / `cut`:

```sh
truestamp jcs < proof.json \
  | grep -o '"pk":"[^"]*"' | cut -d'"' -f4 \
  | truestamp convert keyid
```

Skipping the `jcs` step only works on an already-minified bundle; a
pretty-printed proof writes `"pk": "..."` with a space, which the pattern
above would miss.

With `jq` (cleaner):

```sh
jq -r .pk proof.json | truestamp convert keyid
```

### Extract the item timestamp from a proof

```sh
jq -r .s.id proof.json | truestamp convert id
```

### Extract the block commit time from a proof (in your local zone)

```sh
jq -r .b.id proof.json | truestamp convert id --to-zone Local
```

### Confirm a downloaded file matches what you timestamped

```sh
# Hash the local file, compare against the hash the proof commits to.
# Exit 0 if they match; 1 otherwise.
expected="$(jq -r .s.d.hash proof.json)"
actual="$(truestamp hash -a sha256 --style bare contract.pdf)"
[ "$expected" = "$actual" ] && echo "match" || echo "MISMATCH"
```

This recipe is **external-hash mode only**. For
claims-as-source-of-truth proofs there's no separate file to
compare, because the claims content itself is what the proof commits to.
You can spot a claims-only proof at a glance with
`jq '.s.d | has("hash") | not' proof.json`; the verify report
also omits the Hash row for claims-only items.

### Verify a proof while passing the expected hash inline

```sh
truestamp verify proof.json --hash "$(truestamp hash -a sha256 --style bare contract.pdf)"
```

### Batch-verify every proof in a directory, silently

```sh
find proofs -name '*.json' -print0 \
  | xargs -0 -I{} sh -c 'truestamp verify --silent --skip-external "{}" || echo "FAIL: {}"'
```

### Hash a file with every supported algorithm in one pass

```sh
for alg in $(truestamp hash --list | awk '{print $1}'); do
  truestamp hash -a "$alg" --style bsd contract.pdf
done
```

### Round-trip test a proof you just received

```sh
# JSON to CBOR to JSON, then compare against the original (modulo
# canonical key order)
orig="$(truestamp jcs < proof.json)"
round="$(truestamp convert proof --to cbor proof.json \
        | truestamp convert proof --from cbor --to json \
        | truestamp jcs)"
[ "$orig" = "$round" ] && echo "round-trip stable" || echo "DRIFT"
```

### Inspect every Merkle sibling in a proof's inclusion path

```sh
jq -r .ip proof.json | truestamp convert merkle
```

### Convert every `cx[].ts` commitment timestamp to your local zone

```sh
jq -r '.cx[].ts' proof.json \
  | while read -r ts; do
      truestamp convert time "$ts" --to-zone Local
    done
```

### Show the embedded item time and block time side-by-side

```sh
printf "item:  "; jq -r .s.id proof.json | truestamp convert id
printf "block: "; jq -r .b.id proof.json | truestamp convert id
```

---

## Scripting with `--json` and `jq`

Every inspection command supports `--json`. Combined with `jq`, you can
build sophisticated pipelines with no parsing glue code.

```sh
# Pull the hash found in the proof (when --hash was supplied on the
# verify invocation to populate the comparison block)
truestamp verify proof.json --json --hash "<expected>" \
  | jq -r .hash_comparison.found

# Extract the verify result ("verified" / "failed" / "rejected")
truestamp verify proof.json --json | jq -r .result

# Compute a digest and pipe it into another command's --hash flag
expected="$(truestamp hash -a sha256 --json contract.pdf | jq -r .digest.hex)"
truestamp verify proof.json --hash "$expected"

# Compare the claims.hash field in a proof against a fresh local hash.
# Works for external-hash proofs only; claims-only proofs have no
# .s.d.hash field (jq returns "null") and don't need this comparison,
# because the claims content is already in s.d.
proof_hash="$(jq -r .s.d.hash proof.json)"
fresh_hash="$(truestamp hash -a sha256 --json contract.pdf | jq -r .digest.hex)"
[ "$proof_hash" = "$fresh_hash" ] && echo "match" || echo "MISMATCH"

# Branch on submission mode at the jq layer
jq 'if .s.d | has("hash") then "external-hash mode" else "claims-only mode" end' proof.json

# Tag-style report of every commitment's type, network, and timestamp
jq -r '.cx[] | "\(.t) (\(.net)) committed at \(.ts)"' proof.json
```

---

## CI / scripting conventions

For automated pipelines, use the following conventions:

```sh
# Silent verification: exit code 0 = pass, 1 = fail
if truestamp verify --silent --skip-external proof.json; then
  echo "valid"
else
  echo "invalid"
  exit 1
fi

# JSON-structured output for parsers
truestamp verify proof.json --json > verify-report.json
jq '.result' verify-report.json

# Suppress the once-per-day passive upgrade notice in CI
export TRUESTAMP_NO_UPGRADE_CHECK=1

# Force deterministic (non-color) output for log ingestion
export NO_COLOR=1
# or
truestamp --no-color verify proof.json
```

Environment variables for CI:

| Variable | Purpose |
| --- | --- |
| `TRUESTAMP_BASE_URL` | Service origin (scheme + host). API, keyring, console and health URLs all derive from it. |
| `TRUESTAMP_API_KEY` | Auth token for `create` / `download` / `beacon` / `team` / `console` / `auth status` / `verify --remote` |
| `TRUESTAMP_TEAM` | Multi-tenant team ID |
| `TRUESTAMP_HTTP_TIMEOUT` | HTTP timeout (`30s`, `1m`) |
| `TRUESTAMP_COSIGN_PATH` | Absolute path to the `cosign` binary used by `truestamp upgrade` (empty = `$PATH` lookup) |
| `TRUESTAMP_LOGGING_LEVEL` / `TRUESTAMP_LOGGING_FILE` | Log level and log file path (named after the `[logging]` config section, not the flag) |
| `TRUESTAMP_LOGGING_MAX_SIZE_MB` / `TRUESTAMP_LOGGING_MAX_BACKUPS` / `TRUESTAMP_LOGGING_MAX_AGE_DAYS` | Log rotation budget |
| `TRUESTAMP_VERIFY_SILENT` / `TRUESTAMP_VERIFY_JSON` | Defaults for `verify --silent` / `--json` |
| `TRUESTAMP_VERIFY_SKIP_EXTERNAL` / `TRUESTAMP_VERIFY_SKIP_SIGNATURES` / `TRUESTAMP_VERIFY_REMOTE` | Defaults for the corresponding `verify` flags |
| `TRUESTAMP_HASH_ALGORITHM` | Default algorithm for `truestamp hash` |
| `TRUESTAMP_HASH_ENCODING` | Default digest encoding (`hex` / `base64` / `base64url`) |
| `TRUESTAMP_HASH_STYLE` | Default output style (`gnu` / `bsd` / `bare`) |
| `TRUESTAMP_CONVERT_TIME_ZONE` | Default `--to-zone` for `convert time` / `convert id` |
| `TRUESTAMP_NO_UPGRADE_CHECK` | Disable the passive upgrade nag |
| `NO_COLOR` | Industry-standard: strip all ANSI colors |

Truestamp also auto-detects common CI environments and silences the passive
upgrade notice there: `CI`, `GITHUB_ACTIONS`, `GITLAB_CI`, `CIRCLECI`,
`BUILDKITE`, `JENKINS_HOME`, `TF_BUILD`.

---

## Offline / air-gapped use

Everything except the commands that explicitly talk to the Truestamp API
(`create`, `download`, `auth`, `beacon`, `team`, `console`, `verify --remote`)
works without network:

```sh
# Fully offline verification: no calls to Truestamp, Stellar, or Bitcoin APIs
truestamp verify proof.json --skip-external

# All the convert / hash / encode / decode / jcs primitives are offline-only;
# they never touch the network.
truestamp hash -a sha256 file.bin
truestamp jcs < claims.json
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD
truestamp convert proof --to cbor proof.json
```

`truestamp verify` without `--skip-external` performs four classes of
outbound requests:

1. Fetches the Truestamp keyring at `{base-url}/.well-known/keyring.json` (default `https://www.truestamp.com/...`) to cross-check the signing key. This one happens for every bundle.
2. If a Stellar commitment is present, hits the Horizon API (`horizon.stellar.org` or `horizon-testnet.stellar.org`).
3. If a Bitcoin commitment is present on mainnet or testnet, hits the Blockstream API (`blockstream.info`). Regtest has no public API, so that case is local-only.
4. For an entropy subject, hits the upstream publisher the observation came from, to byte-compare the captured value: the **NIST Randomness Beacon** (`beacon.nist.gov`) for `entropy_nist`, Horizon for `entropy_stellar`, Blockstream mainnet for `entropy_bitcoin`.

`--skip-external` skips all four. `--skip-signatures` skips the Ed25519
**Proof Signature** check and the **Key Binding** keyring cross-check. Note
that it does *not* skip the **Signing Key** step, which still decodes `pk`
and derives its key id, so an undecodable public key still fails. A run with
`--skip-signatures` says so on its verdict line
(`VERIFIED - but the signature was NOT checked (--skip-signatures)`), because
it establishes nothing about who issued the proof. Everything else local
(subject hash, Merkle inclusion, block hash, epoch proofs) is always
performed.

---

**See also:**
`README.md` for install instructions, `kb/` for architecture and format
reference (start at `kb/README.md`),
`./build/truestamp <command> --help` for per-command flag documentation.
