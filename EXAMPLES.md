# Truestamp CLI — Examples

A hands-on tour of every sub-command, followed by real-world pipeline recipes.
Every example is copy-pastable; outputs shown here were captured from the
actual binary.

Run `truestamp <command> --help` at any time for exhaustive flag documentation.

---

## Table of contents

- [External tools used in these examples](#external-tools-used-in-these-examples)
- [Conventions](#conventions)
- [`truestamp auth`](#truestamp-auth) — **start here: prerequisite for `create` / `download` / `verify --remote`**
- [`truestamp config`](#truestamp-config)
- [Lifecycle: the three-step flow](#lifecycle-the-three-step-flow)
- [`truestamp create`](#truestamp-create)
- [`truestamp download`](#truestamp-download)
- [`truestamp verify`](#truestamp-verify)
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
`--json` + `convert` + `jcs` primitives cover most parse / transform needs —
`jq` appears in examples only because it's the industry-standard fallback.

## Conventions

Every sub-command that reads input supports the same **six modes**:

| How | Syntax |
| --- | --- |
| Positional argument | `truestamp verify proof.json` |
| Explicit file path | `truestamp verify --file proof.json` |
| Interactive file picker | `truestamp verify --file` (no path — TUI picker opens) |
| Explicit URL | `truestamp verify --url https://example.com/proof.json` |
| Interactive URL prompt | `truestamp verify --url` (no URL — TUI prompt opens) |
| Stdin pipe | `cat proof.json \| truestamp verify` |
| Stdin (explicit `-`) | `truestamp hash -` (Unix convention) |

Truly global (persistent) flags, available on every sub-command:

- `--no-color` — strip ANSI (also respects `NO_COLOR=1`)
- `--config <path>` — override the config file location
- `--no-upgrade-check` — suppress the passive "new version available" notice
- `--api-url` / `--api-key` / `--team` / `--keyring-url` / `--http-timeout` — network settings

Widely-available (per-command where meaningful) flags you'll see repeatedly:

- `--json` — machine-readable output; supported by `verify`, `hash`, `encode`, `decode`, `jcs`, every `convert` sub-command, `create`, and `config show`
- `-s` / `--silent` — exit code only, no output; supported by `verify`, `hash`, `encode`, `decode`, `jcs`, every `convert` sub-command

Configuration resolution order (highest wins):

1. Compiled defaults
2. `~/.config/truestamp/config.toml` (or `$XDG_CONFIG_HOME/...`)
3. Environment variables (`TRUESTAMP_*` prefix)
4. CLI flags

Exit code convention: **`0` = success, `1` = failure**. Specialized commands add
codes (`upgrade --check` uses 0–3; see its help).

---

## `truestamp auth`

**Start here.** `create`, `download`, and `verify --remote` all require an API
key. Everything else in this document — `hash`, `encode`, `decode`, `jcs`,
`verify` (local), every `convert` subcommand — works without authentication and
without a network. If you only need local verification and inspection, you can
skip this section.

Get an API key from <https://www.truestamp.com>, then:

```sh
# Interactive login (prompts for the API key, persists it to the config file
# at the right permissions — 0600, because the key is a secret)
truestamp auth login

# Confirm who you are, what team the key belongs to, and whether the API
# accepts the stored key. Exit 0 = valid, exit 1 = missing / invalid / network.
truestamp auth status

# Clear the stored credentials
truestamp auth logout
```

The API key is written to `~/.config/truestamp/config.toml` (or
`$XDG_CONFIG_HOME/truestamp/config.toml`). You can also set it ad-hoc via the
`TRUESTAMP_API_KEY` environment variable or the `--api-key` flag on any
command. Environment and flag values override the config file.

## `truestamp config`

Inspect and manage the resolved configuration. Settings are merged from
compiled defaults → config file → `TRUESTAMP_*` env vars → CLI flags, with
later sources overriding earlier ones.

```sh
# Where is the config file?
truestamp config path

# Print the fully-resolved config (API key masked)
truestamp config show

# Create a default config.toml if it doesn't yet exist
truestamp config init

# Override any setting for a single invocation via an env var
TRUESTAMP_API_URL=https://www.truestamp.com/api/json truestamp config show

# Or via a flag
truestamp config show --api-url https://www.truestamp.com/api/json
```

Defaults worth knowing:

- `api_url` → `https://www.truestamp.com/api/json`
- `keyring_url` → `https://www.truestamp.com/.well-known/keyring.json`
- `http_timeout` → `10s`
- `hash.algorithm` → `sha256`
- `hash.encoding` → `hex`
- `hash.style` → `gnu` (sha256sum-compatible)
- `convert.time_zone` → `UTC`

See the full env-var reference in [CI / scripting conventions](#ci--scripting-conventions).

---

## Lifecycle: the three-step flow

The canonical Truestamp workflow is **create → download → verify**. Everything
else in the CLI supports, inspects, or extends this flow.

```sh
# 1. Create an item (hashes the file locally, submits claim to API)
truestamp create contract.pdf
# → prints the new item's ID and claims_hash

# 2. Later, after the item is committed to a block, download its proof
truestamp download 01KNN33GX5E470CB9TRWAYF9DD -o contract.proof.json

# 3. Verify the proof end-to-end
truestamp verify contract.proof.json
# → walks signing key, merkle proof, block hash, commitments; exits 0 on success
```

---

## `truestamp create`

Submit a new timestamp item. Needs `--api-key` (via flag, env, or config).

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

# Submit a precomputed claims JSON (see proof.livemd for the shape)
truestamp create --claims claims.json

# Claims via stdin
jq -c '.' claims.json | truestamp create -C

# Flag-only: provide the hash and name directly
truestamp create -n "Q1 report" --hash aa11bb22...f00d --hash-type sha256

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

Filenames use hyphens; wire values use underscores (the CLI translates
between them so filenames stay readable).

```sh
# ULID id — defaults to --type item, produces truestamp-item-<ulid>.json
truestamp download 01KNN33GX5E470CB9TRWAYF9DD

# Same, with an explicit type (identical behaviour)
truestamp download --type item 01KNN33GX5E470CB9TRWAYF9DD

# Override the auto-generated filename
truestamp download 01KNN33GX5E470CB9TRWAYF9DD -o contract.proof.json

# CBOR — smaller, deterministic, ideal for embedding in another file
truestamp download 01KNN33GX5E470CB9TRWAYF9DD -f cbor -o contract.proof.cbor

# Entropy proof — UUIDv7 ids require an explicit --type (three subtypes)
truestamp download --type entropy_stellar 019cf813-99b8-730a-84f1-5a711a9c355e
truestamp download --type entropy_nist    019cf813-99b8-730a-84f1-5a711a9c355e
truestamp download --type entropy_bitcoin 019cf813-99b8-730a-84f1-5a711a9c355e

# Block proof (t=10) for a committed block
truestamp download --type block 019db7cd-efc0-7196-b763-682a84d71919

# Beacon proof (t=11) for the same block — structurally identical to a
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

UUIDv7 ids are ambiguous — entropy observations, blocks, and beacons
all use UUIDv7 — so the CLI cannot infer what you want. A UUIDv7
without `--type` exits with a helpful error listing the five valid
types. Use `truestamp beacon by-hash <hash>` first if you only have a
hash and need the id.

---

## `truestamp verify`

Verify the full cryptographic chain: signing key → claims hash → item hash →
Merkle proof → block hash → Stellar / Bitcoin commitments.

```sh
# Local file
truestamp verify contract.proof.json

# URL — auto-detected from the positional argument
truestamp verify https://example.com/proof.json

# Stdin pipe
cat contract.proof.json | truestamp verify

# Interactive picker
truestamp verify --file

# Pin the expected claims hash and fail if it doesn't match
truestamp verify contract.proof.json \
  --hash e08764deac64ca9a1046901c5b23674941f1e86f0e2d0429ee07c5e311a15ce7

# Assert the expected subject type — guards against verifying the wrong
# file. Useful because block (t=10) and beacon (t=11) proofs are
# structurally identical on the wire, so a renamed or swapped file
# would still verify on its own. --type <t> fails the verify if the
# bundle's t doesn't match the requested type.
truestamp verify --type beacon truestamp-beacon-019d....json
truestamp verify --type item   truestamp-item-01K....json
truestamp verify --type entropy_stellar truestamp-entropy-stellar-019c....json

# Smart default: when you verify a file whose name matches the
# `truestamp-<stem>-<id>.<ext>` convention `truestamp download` emits,
# the CLI infers --type from the filename and asserts automatically.
# A faint stderr hint tells you the inference happened. To override,
# pass --type explicitly; to skip the assertion, rename the file.
truestamp verify truestamp-beacon-019d....json          # infers --type beacon
truestamp verify truestamp-entropy-nist-019d....json    # infers --type entropy_nist

# Skip the public-blockchain checks (offline / restricted networks)
truestamp verify contract.proof.json --skip-external

# Skip all signature verification (structural check only)
truestamp verify contract.proof.json --skip-signatures

# Script-friendly modes
truestamp verify contract.proof.json --json        # structured output
truestamp verify contract.proof.json --silent      # exit code only
```

Use `--remote` to delegate verification to the Truestamp server (requires
API key). Local verification is the default and needs no credentials.
When `--type` is combined with `--remote`, the value is forwarded to the
server's `/proof/verify` endpoint — a server-side assertion rather than
a CLI-only one — and a mismatch returns HTTP 4xx with
`meta.code=subject_type_mismatch`.

---

## `truestamp hash`

Multi-algorithm digest tool. Default output is **byte-identical to
`sha256sum`** so it drops into existing scripts.

```sh
# SHA-256 a file (default algorithm)
truestamp hash contract.pdf
# → ba7816bf8f01cfea...ad  contract.pdf

# Multiple files at once
truestamp hash a.bin b.bin c.bin

# Stdin (filename shows as "-", matching sha256sum)
echo -n "abc" | truestamp hash
# → ba7816bf...ad  -

# List supported algorithms
truestamp hash --list

# Pick any supported algorithm
truestamp hash -a sha3-256 contract.pdf
truestamp hash -a blake2b-512 contract.pdf
truestamp hash -a md5 contract.pdf             # warns: legacy algorithm

# BSD-style tagged output (shasum --tag compatible)
truestamp hash -a sha256 --style bsd contract.pdf
# → SHA256 (contract.pdf) = ba78...ad

# Bare digest (no filename, no separator)
truestamp hash -a sha256 --style bare contract.pdf

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

---

## `truestamp encode` / `truestamp decode`

Pipe-friendly byte-encoding primitives. `encode` takes raw bytes and produces
text; `decode` takes text and produces raw bytes. Both support text→text
conversion via `--from` and `--to`.

Supported encodings: `binary`, `hex`, `base64`, `base64url`.

```sh
# Encode raw bytes to hex (default)
echo -n "hello" | truestamp encode
# → 68656c6c6f

# Encode a file as base64url
truestamp encode --to base64url contract.pdf > contract.b64u

# Decode base64 back to binary
echo "SGVsbG8=" | truestamp decode --from base64
# → Hello

# Text-to-text: hex → base64 without an intermediate binary file
echo "68656c6c6f" | truestamp encode --from hex --to base64
# → aGVsbG8=

# JSON envelope for scripting
echo -n "hello" | truestamp encode --to hex --json
# → {"from":"binary","to":"hex","input_bytes":5,"output_bytes":10,"output":"68656c6c6f"}
```

---

## `truestamp jcs`

Apply **RFC 8785 JSON Canonicalization**. Output is the byte-stable form that
Truestamp uses when computing claims / entropy / metadata hashes.

```sh
# Canonicalize (sorts keys, normalizes whitespace and number formatting)
echo '{"b":2,"a":1,"c":[3,1,2]}' | truestamp jcs
# → {"a":1,"b":2,"c":[3,1,2]}

# Append a trailing newline (for appending to a stream)
truestamp jcs --newline < claims.json

# JSON envelope
truestamp jcs --json < claims.json
```

---

## `truestamp convert time`

Bidirectional time-format tool. Replaces most uses of `date` for parsing,
reformatting, and zone conversion. Accepts RFC 3339 or Unix seconds /
milliseconds / microseconds / nanoseconds; auto-detects by default.

```sh
# Current time in UTC
truestamp convert time now

# Unix seconds → RFC 3339 UTC
truestamp convert time 1700000000
# → 2023-11-14T22:13:20Z

# Convert to another time zone
truestamp convert time 1700000000 --to-zone America/New_York
# → 2023-11-14T17:13:20-05:00

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

---

## `truestamp convert proof`

Convert a proof bundle between JSON and CBOR. The CBOR output uses **RFC 8949
§4.2 core deterministic encoding** and is prefixed with the self-describing
tag 55799 so `truestamp verify` auto-detects it.

```sh
# JSON → CBOR
truestamp convert proof --to cbor proof.json > proof.cbor

# CBOR → JSON (auto-detected input format)
truestamp convert proof --to json proof.cbor | jq .

# Force the input format (error out if the bytes don't match)
truestamp convert proof --from json --to cbor < proof.json

# Compact JSON (minified)
truestamp convert proof --to json --compact proof.cbor

# Round-trip verification (the CBOR output must verify end-to-end)
truestamp convert proof --to cbor proof.json | truestamp verify --skip-external
```

---

## `truestamp convert id`

Extract the embedded timestamp from a ULID (item IDs) or UUIDv7 (block and
entropy IDs). Truestamp uses ULIDs for item-style subjects and UUIDv7 for
blocks + entropy observations.

```sh
# ULID → embedded timestamp (UTC)
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD
# → 2026-04-07T23:05:39.493Z

# UUIDv7 → timestamp
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

---

## `truestamp convert keyid`

Derive the 4-byte Truestamp **key fingerprint** (`kid`) from an Ed25519 public
key. Formula: `truncate4(SHA256(0x51 || pubkey))`. Useful for confirming which
signing key a proof was issued under.

```sh
# Standard base64 Ed25519 public key → 8-char hex kid
truestamp convert keyid CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=
# → 4ceefa4a

# Hex input
truestamp convert keyid --from hex 093c0ca83667...b4

# Auto-detect the encoding (hex / base64 / base64url)
truestamp convert keyid <pubkey-from-anywhere>

# From stdin
jq -r .pk proof.json | truestamp convert keyid
```

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

---

## `truestamp beacon`

Inspect Truestamp block beacons via the read-only JSON:API at
`/api/json/beacons/*`. A **beacon** is a compact projection of a
finalized block — four fields only: `{id, hash, timestamp,
previous_hash}`. It's a "proof of life" commitment: every item and
entropy observation finalized inside that minute window is covered by
the beacon's hash.

Full verifiable proof bundles for a beacon are a separate artefact
fetched via `truestamp download --type beacon <id>` (see above).

```sh
# Current head beacon (no subcommand is an alias for `latest`)
truestamp beacon
truestamp beacon latest

# Most-recent N beacons, newest first (server caps at 100)
truestamp beacon list
truestamp beacon list --limit 3

# Look up by UUIDv7 id
truestamp beacon get 019db8b5-90a1-7015-a62c-48e5038f2306

# Look up by 64-hex-char hash — useful when all you have is a hash
# (e.g. printed on a receipt or read from photo metadata)
truestamp beacon by-hash 9f0be4446c4bfb8faa9a13766e3635b2c27913f35cec4eafcc20cd10af663feb
```

Shared flags (all four subcommands): `--json` (raw JSON, pipeline
friendly), `--hash-only` (prints just the hash + newline for shell
substitution), `--silent` / `-s` (exit code only). `--silent` +
`--json`, `--silent` + `--hash-only`, and `--json` + `--hash-only` are
mutually exclusive. `--hash-only` is not valid on `beacon list` (a list
has no single hash).

```sh
# Capture the current chain head as a moment-in-time anchor
MOMENT=$(truestamp beacon --hash-only)
echo "beacon hash: $MOMENT"
# → beacon hash: 9f0be4446c4bfb8faa9a13766e3635b2c27913f35cec4eafcc20cd10af663feb

# Pipeline-friendly JSON
truestamp beacon list --limit 10 --json | jq -r '.[].hash'

# Round-trip: id → hash → id (demonstrates by-hash lookup)
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

All beacon subcommands require `--api-key` (via flag, env, or config
file); a missing key prints a "Not authenticated" banner to stderr and
exits non-zero.

---

## `truestamp upgrade`

Self-upgrade the binary. The command is **install-method aware** — Homebrew
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

---

## `truestamp version`

```sh
# Detailed build info (version, commit, build date, toolchain, OS/arch,
# detected install method)
truestamp version

# Terse one-liner (just the version number)
truestamp --version
```

---

## Pipeline recipes

Real-world compositions that solve a specific problem.

### Reproduce the protocol's `claims_hash` intermediate

The Truestamp protocol computes an internal intermediate,
`claims_hash = SHA256(0x11 || JCS(claims))`, while deriving the item_hash
that gets signed into the block. The value itself is **not serialized** into
the proof bundle (the proof stores `s.d` — the raw claims — and `s.mh` —
the *metadata*_hash under prefix `0x12`, which is a different value).

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
intermediate — compare the file's plain SHA-256 against `s.d.hash`, or
pass `--hash <sha256-hex>` to `truestamp verify` (see the next two recipes).

### Convert a JSON proof to CBOR and verify

```sh
truestamp convert proof --to cbor proof.json \
  | truestamp verify --skip-external
```

### Derive a kid from a proof's embedded public key, without `jq`

```sh
grep -o '"pk":"[^"]*"' proof.json | cut -d'"' -f4 \
  | truestamp convert keyid
```

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
# Hash the local file, compare against the proof's claims hash.
# Exit 0 if they match; 1 otherwise.
expected="$(jq -r .s.d.hash proof.json)"
actual="$(truestamp hash -a sha256 --style bare contract.pdf)"
[ "$expected" = "$actual" ] && echo "match" || echo "MISMATCH"
```

### Verify a proof while passing the expected hash inline

```sh
truestamp verify proof.json --hash "$(truestamp hash -a sha256 --style bare contract.pdf)"
```

### Batch-verify every proof in a directory, silently

```sh
find proofs -name '*.json' -print0 \
  | xargs -0 -I{} sh -c 'truestamp verify --silent --skip-external "{}" || echo "FAIL: {}"'
```

### Hash a file with a dozen algorithms in one pass

```sh
for alg in $(truestamp hash --list | awk '{print $1}'); do
  truestamp hash -a "$alg" --style bsd contract.pdf
done
```

### Round-trip test a proof you just received

```sh
# JSON → CBOR → JSON → compare against original (modulo canonical key order)
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
# Pull the claims_hash found in the proof (when --hash was supplied on the
# verify invocation to populate the comparison block)
truestamp verify proof.json --json --hash "<expected>" \
  | jq -r .hash_comparison.found

# Extract the verify result ("verified" / "failed")
truestamp verify proof.json --json | jq -r .result

# Compute a digest and pipe it into another command's --hash flag
expected="$(truestamp hash -a sha256 --json contract.pdf | jq -r .digest.hex)"
truestamp verify proof.json --hash "$expected"

# Compare the claims.hash field in a proof against a fresh local hash
proof_hash="$(jq -r .s.d.hash proof.json)"
fresh_hash="$(truestamp hash -a sha256 --json contract.pdf | jq -r .digest.hex)"
[ "$proof_hash" = "$fresh_hash" ] && echo "✓ match" || echo "✗ MISMATCH"

# Tag-style report of every commitment's type, network, and timestamp
jq -r '.cx[] | "\(.t) (\(.net)) committed at \(.ts)"' proof.json
```

---

## CI / scripting conventions

For automated pipelines, use the following conventions:

```sh
# Silent verification — exit code 0 = pass, 1 = fail
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
| `TRUESTAMP_API_URL` | API endpoint override |
| `TRUESTAMP_API_KEY` | Auth token for `create` / `auth status` / `verify --remote` |
| `TRUESTAMP_TEAM` | Multi-tenant team ID |
| `TRUESTAMP_KEYRING_URL` | Signing-key registry URL |
| `TRUESTAMP_HTTP_TIMEOUT` | HTTP timeout (`30s`, `1m`) |
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

Everything except the three commands that explicitly talk to the Truestamp
API (`create`, `download`, `auth`, `verify --remote`) works without network:

```sh
# Fully offline verification — no calls to Truestamp, Stellar, or Bitcoin APIs
truestamp verify proof.json --skip-external

# All the convert / hash / encode / decode / jcs primitives are offline-only;
# they never touch the network.
truestamp hash -a sha256 file.bin
truestamp jcs < claims.json
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD
truestamp convert proof --to cbor proof.json
```

`truestamp verify` without `--skip-external` performs three classes of
outbound requests:

1. Fetches the Truestamp keyring at `https://www.truestamp.com/.well-known/keyring.json` to validate the signing key.
2. If a Stellar commitment is present, hits the Horizon API (`horizon.stellar.org` or testnet).
3. If a Bitcoin commitment is present, hits the Blockstream API (`blockstream.info`).

Skip all three with `--skip-external`. Skip only the signing-key step with
`--skip-signatures`. All local cryptographic verification (item hash, Merkle
proof, block hash, proof signature) is always performed.

---

**See also:**
`README.md` for install instructions, `CLAUDE.md` for architecture notes,
`./build/truestamp <command> --help` for per-command flag documentation.
