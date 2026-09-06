# Truestamp CLI - Examples

A hands-on tour of every sub-command, followed by real-world pipeline recipes.
Every example is copy-pastable once you substitute your own paths and ids;
outputs shown here were captured from the actual binary. A literal `...`
inside a value marks a long string elided for readability.

Run `truestamp <command> --help` at any time for exhaustive flag documentation,
or `truestamp schema get commands --json` for the entire command tree - every
path, flag, type, default and enum - as machine-readable JSON.

---

## Table of contents

- [External tools used in these examples](#external-tools-used-in-these-examples)
- [Conventions](#conventions) (input modes, optional-value flags, [paging lists](#paging-lists))
- [`truestamp auth`](#truestamp-auth) - **start here: prerequisite for `items` / `proofs get` / `blocks` / `beacons` / `entropy` / `teams` / `console` / `verify --remote`**
- [`truestamp config`](#truestamp-config)
- [Lifecycle: the three-step flow](#lifecycle-the-three-step-flow)
- [`truestamp items create`](#truestamp-items-create)
- [`truestamp proofs get`](#truestamp-proofs-get)
- [`truestamp verify`](#truestamp-verify)
- [`truestamp inspect`](#truestamp-inspect)
- [`truestamp teams`](#truestamp-teams)
- [`truestamp console`](#truestamp-console)
- [`truestamp hash`](#truestamp-hash)
- [`truestamp encode` / `truestamp decode`](#truestamp-encode--truestamp-decode)
- [`truestamp jcs`](#truestamp-jcs)
- [`truestamp convert time`](#truestamp-convert-time)
- [`truestamp proofs convert`](#truestamp-proofs-convert)
- [`truestamp convert id`](#truestamp-convert-id)
- [`truestamp convert keyid`](#truestamp-convert-keyid)
- [`truestamp convert merkle`](#truestamp-convert-merkle)
- [`truestamp beacons`](#truestamp-beacons)
- [`truestamp items` (list, get, update)](#truestamp-items-list-get-update)
- [`truestamp blocks`](#truestamp-blocks)
- [`truestamp entropy`](#truestamp-entropy)
- [`truestamp keys`](#truestamp-keys)
- [`truestamp schema`](#truestamp-schema)
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
| `cat` / `echo` / `printf` / `export` / `read` / `grep` / `cut` / `head` / `tail` / `find` / `xargs` / `awk` / `sh` / `for` / `while` / `if` | Shell plumbing and text manipulation | POSIX-standard. Built-in on macOS, Linux, BSD, WSL, Git-Bash on Windows |
| `curl` | Fetch a remote file to submit, or download the published keyring for offline verification | Pre-installed on macOS and most Linux; install via your package manager if missing |
| `date` | Print the current Unix time to pipe into `convert time` | POSIX-standard; GNU `date` and BSD `date` have different `-d` / `-v` flag syntax, which is why the examples only ever use it as `date -u +%s` |
| `jq` | Parse and transform JSON | Third-party; [install from jqlang.org](https://jqlang.org/). Highly recommended for any scripting against Truestamp's `--json` output |

Windows users running outside WSL should install Git-Bash or a similar POSIX
shell to run the examples. PowerShell-native equivalents work too, but the
pipe / redirection syntax will differ.

If you want to run the pipelines without installing `jq`, `truestamp`'s own
`--json` + `convert` + `jcs` primitives cover most parse / transform needs.
`jq` appears in examples only because it's the industry-standard fallback.

## Conventions

`verify`, `inspect`, `encode`, `decode`, `jcs` and `proofs convert` share one
input resolver and accept the same **six modes**, plus the Unix `-` alias for
stdin:

| How | Syntax |
| --- | --- |
| Positional argument | `truestamp verify proof.json` |
| Explicit file path | `truestamp verify --file proof.json` |
| Interactive file picker | `truestamp verify --file` (no path, TUI picker opens) |
| Explicit URL | `truestamp verify --url https://example.com/proof.json` |
| Interactive URL prompt | `truestamp verify --url` (no URL, TUI prompt opens) |
| Stdin pipe | `cat proof.json \| truestamp verify` |
| Stdin (explicit `-`) | `truestamp verify -` (Unix convention) |

Three families sit outside that resolver:

- `hash` takes the same six modes (`truestamp hash -` included) but accepts
  **many** positional files (`truestamp hash a.txt b.txt`), so `--file` /
  `--url` must use the `=` form there — see below.
- `convert time`, `convert id`, `convert keyid` and `convert merkle` take their
  value positionally or on stdin only. They have no `--file` / `--url`, and `-`
  is read as a value rather than as stdin.
- `items create` has its own documented priority order
  (`truestamp items create --help`).

The two interactive modes need a terminal. Under a pipe or redirect they fail
fast with a named reason rather than hanging:

```sh
truestamp verify --file < /dev/null
# --file without a path: interactive prompt requires a terminal (stdin is piped or redirected)
# exit 1

truestamp verify --url < /dev/null
# --url without a URL: interactive prompt requires a terminal (stdin is piped or redirected)
# exit 1
```

### `--file` / `--claims` / `--url` take an *optional* value

These flags are declared with an optional argument so that `--file` alone can
open the interactive picker. That has a consequence worth internalizing: a
space-separated value is parsed as a **positional argument**, not as the flag's
value.

`verify`, `inspect`, `encode`, `decode`, `jcs` and `proofs convert` all fall
back to the positional argument when `--file` / `--url` were given with no
value, so both spellings work there:

```sh
truestamp verify --file proof.json     # works (falls back to the positional)
truestamp verify --file=proof.json     # works (unambiguous)
```

**`truestamp hash` has no such fallback**, because for `hash` a positional is a
file to digest and there may be several of them. Combining the flag with a
positional is refused rather than guessed at:

```sh
truestamp hash --file=doc.pdf          # correct
truestamp hash doc.pdf                 # correct (positional, and repeatable)
truestamp hash --file doc.pdf
# --file/--url cannot be combined with positional file arguments
# exit 1
```

**`truestamp items create` has no such fallback either.** Its `--claims` / `-c`
and `--file` / `-f` flags *must* use the `=` form, or the CLI rejects the
invocation up front, naming the `=` form as the fix (a space-separated path
would otherwise be read as a file to hash, not as claims):

```sh
truestamp items create --claims=claims.json  # correct
truestamp items create -c=claims.json        # correct
truestamp items create --claims claims.json  # ERROR - suggests --claims=claims.json
truestamp items create -c claims.json        # ERROR - suggests --claims=claims.json
truestamp items create --file doc.pdf        # ERROR - suggests --file=doc.pdf, or dropping the flag
```

Truly global (persistent) flags, available on every sub-command:

- `--no-color` strips ANSI (also respects `NO_COLOR=1`)
- `--config <path>` overrides the config file location
- `--no-upgrade-check` suppresses the passive "new version available" notice
- `--log-level` / `--log-file` control the shared JSON log file
- `--base-url` / `--api-key` / `--team` / `--http-timeout` are the network settings. Every service URL (API, keyring, console WebSocket, health) is derived from the single `--base-url` origin; there is no `--api-url` and no `--keyring-url` (both are rejected with `unknown flag`, exit 1). One exception to "available everywhere": `auth login` declares its own local `--api-key`, a boolean that switches it to interactive key entry, and that shadows the persistent string flag on that one command.

`--json` and `--silent` are deliberately *not* persistent flags: a persistent
flag would appear on `auth login`, `console` and `completion`, which render no
record and have no such output mode. They are registered per command instead.

`--base-url` wants an **origin only** (scheme + host, optionally a port), for
example `https://www.truestamp.com` or `http://localhost:4000`. Any path,
query or fragment you append is discarded during config load, so pass the bare
origin and let the CLI compose `/api/json`, `/.well-known/keyring.json`,
`/console/websocket` and `/health` itself:

```sh
truestamp --base-url 'https://www.truestamp.com/ignored/path?q=1' config show --json \
  | grep -E '"(base_url|api_url|keyring_url)"'
#   "api_url": "https://www.truestamp.com/api/json",
#   "base_url": "https://www.truestamp.com",
#   "keyring_url": "https://www.truestamp.com/.well-known/keyring.json",
```

Widely-available (per-command where meaningful) flags you'll see repeatedly:

- `--json` for machine-readable output, and `-s` / `--silent` for
  exit-code-only runs. They are mutually exclusive, and every command that
  renders a *record* carries both.
- Both are also CLI-wide *settings*: put `json = true` or `silent = true` at the
  top level of `config.toml`, or set `TRUESTAMP_JSON` / `TRUESTAMP_SILENT`, and
  record-rendering commands pick it up without the flag. The pipeline
  primitives (`encode`, `decode`, `jcs`, `convert time|id|keyid|merkle`) and
  `hash` deliberately ignore the ambient setting — wrapping their output would
  break every pipe, and `hash --style` stays a byte-identical `sha256sum`
  drop-in. They still carry their own `--json` / `-s` flags when you ask
  explicitly.

```sh
truestamp verify --json --silent --offline proof.json
# --silent and --json are mutually exclusive
# exit 1
```

`config show` renders a styled table by default and the fully resolved settings
as JSON under `--json`; `config path` keeps a deliberate stdout/stderr split so
that stdout stays a single line — the labelled path goes to stdout and whether
that file exists goes to stderr. The first line below is stdout, the second is
stderr — so stdout carries the label as well as the path, and a `$()` capture
is one line but not a bare path:

```sh
NO_COLOR=1 truestamp config path
# Config Path  /Users/glenn/.config/truestamp/config.toml
#              exists
```

Configuration resolution order (highest wins):

1. Compiled defaults
2. `~/.config/truestamp/config.toml` (or `$XDG_CONFIG_HOME/...`)
3. Environment variables (`TRUESTAMP_*` prefix)
4. CLI flags

```sh
# Prove it with a throwaway config file, leaving your real one alone.
printf 'json = true\n' > /tmp/ts.toml

truestamp --config /tmp/ts.toml version                    # JSON  (2 beats 1)
TRUESTAMP_JSON=false truestamp --config /tmp/ts.toml version   # table (3 beats 2)

printf 'json = false\n' > /tmp/ts.toml
truestamp --config /tmp/ts.toml version --json             # JSON  (4 beats 2)
```

Exit code convention: **`0` = success, `1` = failure**, `2` = an unrecovered
panic (matching Go's own convention). Specialized commands add codes
(`upgrade --check --exit-code` uses 0 through 3). The authoritative table is
generated from the constants that produce it:

```sh
truestamp schema get exit-codes
#   Exit codes
#
#   CODE    MEANING
#   0       the command did what it was asked
#   1       it did not: failed verification, network failure, invalid input, or any other runtime error
#   2       an unrecovered panic, matching Go's own convention
#
#   upgrade --check --exit-code
#
#   CODE    MEANING
#   0       up to date
#   1       an upgrade is available
#   2       a network error prevented the check
#   3       the latest release is a pre-release and will not auto-install
```

### Paging lists

`items list`, `blocks list` and `entropy list` page the same way, so learn it
once:

| Flag | What it does |
| --- | --- |
| `--limit N` | Page size (default 25; a page above 250 is clamped to 250, and the listing says so) |
| `--after <cursor>` | Continue forward from the cursor a page printed |
| `--before <cursor>` | Continue backward from the cursor a page printed |
| `--oldest-first` | Start at the beginning (genesis, the first observation, the oldest item) instead of the newest row |
| `--max N` | Follow cursors in the chosen direction until N rows have been fetched |
| `--count` | Add the server's total to the heading (`(2 shown, 45,934 total)`) and `"total"` to `--json` |

A text page that continues ends with `More: --after <cursor>` and, when it
can go back, `Back: --before <cursor>`; `--json` is one envelope,
`{"<noun>": [...], "next_cursor": "...", "prev_cursor": "..."}`. There is no
`--all`: the tables behind these lists grow by the minute, so following pages
costs a cap you wrote down, and `--max` sizes its last request to the rows
still wanted so the cursor it hands back continues exactly.

```sh
truestamp blocks list --limit 2                  # newest two, plus a More: cursor
truestamp blocks list --limit 2 --after <cursor> # the next two, plus More: and Back:
truestamp blocks list --oldest-first --limit 2   # genesis first
truestamp entropy list --max 500 --json | jq -r '.observations[].id'
truestamp items list --count --limit 1           # Items (1 shown, 10 total)
```

Every collection clamps a page above 250 rows to 250 rather than refusing
it; the listing says so, and `--max` follows the cursor past it.

---

## `truestamp auth`

**Start here.** `items`, `proofs`, `blocks`, `beacons`, `teams`, `console` and
`verify --remote` all require credentials. Everything else in this document,
`hash`, `encode`, `decode`, `jcs`, `verify` (local), `inspect`, `schema` and
every `convert` subcommand, works without authentication and without a network.
`keys` sits in between: it needs the network to fetch the public keyring, but
no credential. If you only need local verification and inspection, you can skip
this section.

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

# Machine-readable. `ok` mirrors the exit code, so a caller can branch on
# either. Empty fields are omitted rather than emitted as null.
truestamp auth status --json
# {
#   "ok": true,
#   "config_file": "...",
#   "api_url": ".../api/json",
#   "auth_mode": "API key",
#   "api_key": "...",
#   "user_id": "...",
#   "email": "..."
# }

# A credential the API rejects: exit 1, plus a stable `reason` identifier
# (never a sentence) and a human-readable `message`. The reason vocabulary
# is not_authenticated, api_unreachable, credential_rejected,
# unexpected_api_response, team_lookup_failed, team_not_accessible.
truestamp auth status --json --api-key not-a-real-key
# {
#   "ok": false,
#   "reason": "credential_rejected",
#   "message": "Authentication required. Provide an API key as ...",
#   "config_file": "...",
#   "api_url": ".../api/json",
#   "auth_mode": "API key",
#   "api_key": "not-...-key",
#   "http_status": 401
# }

# Exit code only, nothing on stdout
truestamp auth status --silent

# Revoke the OAuth session and clear it locally (idempotent).
truestamp auth logout

# Also remove the stored API key from config.toml.
truestamp auth logout --api-key
```

Precedence, highest first:

1. An **explicit** API key: the `--api-key` flag or `TRUESTAMP_API_KEY` env. Wins outright, so CI is deterministic.
2. The stored **OAuth session**, auto-refreshed.
3. An `api_key` sitting in `config.toml`.
4. Nothing. Public reads still work (`keys current`, `schema`), but a command that needs a credential refuses locally rather than eliciting a 401: `blocks latest` and `teams list` print `Not authenticated`, and `verify --remote` errors before any request goes out.

Both credential types are presented to the API as `Authorization: Bearer
<value>`. For CI, prefer the env var:

```sh
export TRUESTAMP_API_KEY=...   # no browser, no keychain, no interactive step
```

`truestamp auth status` prints the resolved **Auth Mode** (`OAuth (browser
sign-in)`, `API key`, or `none, run 'truestamp auth login'`), the API and check
URLs, the API key masked to its first and last four characters, the team in
scope, and, in OAuth mode, the granted scopes and the access-token expiry.
There is no offline mode: the whole point of the command is to ask the API
whether the credential is still good.

## `truestamp config`

Inspect and manage the resolved configuration. Settings are merged from
compiled defaults, then the config file, then `TRUESTAMP_*` env vars, then CLI
flags, with later sources overriding earlier ones.

```sh
# Where is the config file, and does it exist yet?
truestamp config path
# Config Path  .../truestamp/config.toml
#              exists
# (or: does not exist, run 'truestamp config init' to create it)

# The styled form puts a label on stdout, so `$(truestamp config path)`
# captures "Config Path  /path" rather than the path. Use --json when a
# script needs the value itself.
truestamp config path --json
# {
#   "exists": true,
#   "path": "/Users/you/.config/truestamp/config.toml"
# }
truestamp config path --json | jq -r .path
# /Users/you/.config/truestamp/config.toml

# Print the fully-resolved config (API key masked)
truestamp config show

# Same, machine-readable
truestamp config show --json

# Create a default config.toml if it doesn't yet exist. Idempotent: an
# existing file is left untouched and the command still exits 0, printing
# "Config already exists at <path>".
truestamp config init

# Write the default file somewhere else instead of the platform default
truestamp config init --config /path/to/truestamp.toml

# Open the file in effect in your editor: $VISUAL, else $EDITOR, else a
# platform default (vi on Unix, notepad on Windows). The value may carry
# arguments, so EDITOR="code -w" works. The file must already exist, and
# the command needs a terminal, so it refuses in a pipe or in CI rather
# than launching vi against a pipe and blocking forever.
truestamp config edit

# Override any setting for a single invocation via an env var
TRUESTAMP_BASE_URL=https://www.truestamp.com truestamp config show

# Or via a flag
truestamp config show --base-url https://www.truestamp.com
```

`config show` renders a General block (Config File, API URL, Auth Mode, API
Key, Team, Keyring URL, HTTP Timeout, Cosign Path, Silent, JSON) followed by
the Verification, Hash and Convert sections. Two further rows, **Team Name**
and **Team Role**, appear directly under Team only when a team id *and* a
usable credential are both present; that lookup is best-effort, so `config
show` stays useful offline. It is not dropped silently: when the API is
unreachable, Team Name reads `(unavailable, try 'truestamp auth status')`,
Team Role is omitted, and the command still exits 0.
Config File is the file actually in effect, so it reflects `--config` when one
was supplied.

`--json` shows values the styled table does not — `base_url`, `verify.keyring`
and a `logging` block — and nests `verify`, `hash`, `convert` and `logging` as
objects (`truestamp config show --json | jq -r .base_url`). It is not a strict
superset in either direction: that `logging` object carries only `level` and
`file`, not the three rotation keys, and the best-effort **Team Name** and
**Team Role** rows are table-only, so the JSON reports a team as the bare
`team` id and nothing more. As everywhere else, `--json` and `-s` / `--silent`
are mutually exclusive; passing both is an error and exits 1.

Defaults worth knowing:

- `base_url` is `https://www.truestamp.com` (API, keyring, console and health URLs are all derived from it)
- `http_timeout` is `10s`
- `cosign_path` is empty, meaning "look `cosign` up on `$PATH`"
- `silent` and `json` are both `false`
- `hash.algorithm` is `sha256`
- `hash.encoding` is `hex`
- `hash.style` is `gnu` (sha256sum-compatible)
- `convert.time_zone` is `UTC`
- `logging.level` is `info`, and `logging.file` is empty, meaning the platform default (`~/Library/Caches/truestamp/truestamp.log` on macOS, `~/.cache/truestamp/truestamp.log` on Linux, `%LOCALAPPDATA%\truestamp\truestamp.log` on Windows)
- the log rotates at `logging.max_size_mb` = `10`, keeping `logging.max_backups` = `5` compressed backups for at most `logging.max_age_days` = `14` days

Every key above has a `TRUESTAMP_`-prefixed env var, named in a comment above
it in the file `config init` writes. See the full env-var reference in
[CI / scripting conventions](#ci--scripting-conventions).

---

## Lifecycle: the three-step flow

The canonical Truestamp workflow is **`items create`, `proofs get`, `verify`**. Everything
else in the CLI supports, inspects, or extends this flow.

```sh
# 1. Create an item (hashes the file locally, submits claims to the API)
truestamp items create contract.pdf
# prints an "Item Created" card: ID, Name, Hash, Visibility (plus Tags and
# Team when set) and the public Details / Verify links

# 2. Later, after the item is committed to a block, download its proof
truestamp proofs get 01KNN33GX5E470CB9TRWAYF9DD -o contract.proof.json

# 3. Verify the proof end-to-end
truestamp verify contract.proof.json
# recomputes every hash, walks the Merkle and epoch proofs, checks the
# signature, the witnesses and the submission window, confirms the
# commitments on chain; exits 0 when the proof passes
```

---

## `truestamp items create`

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
exactly one is rejected. The flags that populate them are `--data-hash` and
`--hash-type` (there is no `--hash` flag on `create`). At the flag layer,
`--hash-type` carries a default of `sha256`, so `--data-hash <hex>` on its own
still lands in external-hash mode, while `--hash-type <algo>` on its own is
rejected.

Remember that `--claims` and `--file` need the `=` form here; see
[Conventions](#conventions).

### External-hash mode

```sh
# Hash a file and submit in one step (filename becomes the item name,
# SHA-256 becomes the hash)
truestamp items create contract.pdf

# Same, machine-readable output
truestamp items create contract.pdf --json

# Pick a file interactively
truestamp items create --file

# Provide the content via stdin (name required separately)
curl -fsSL https://example.com/data.bin | truestamp items create -F -n "data.bin"

# Submit a precomputed claims JSON (`truestamp items create --help` lists the fields it may carry)
truestamp items create --claims=claims.json

# Claims via stdin
jq -c '.' claims.json | truestamp items create -C

# Flag-only: provide the hash and name directly. The hash must be valid
# hex of the length --hash-type implies (64 chars for sha256).
truestamp items create -n "Q1 report" \
  --data-hash ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad \
  --hash-type sha256

# Rich metadata
truestamp items create contract.pdf \
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
truestamp items create -n "Invention" \
  -d "On this day I claim the following novel approach as my own original work."

# Same, machine-readable. hash / hash_type keys are omitted from the
# JSON output entirely (use jq 'has("hash")' to branch on the mode).
truestamp items create -n "Invention" \
  -d "On this day I claim the following novel approach as my own original work." \
  --json

# Metadata escape hatch: no long description needed if metadata
# carries the content
truestamp items create -n "Release v1.2" \
  --metadata '{"version":"1.2.0","sha":"deadbeef","notes":"..."}'

# Claims JSON file with no hash/hash_type: the server treats both as
# absent and the proof commits to the claims bytes directly
truestamp items create --claims=claim-only.json
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
| `--hash-type sha256` without `--data-hash`               | `claims hash is required when hash_type is supplied`                   |
| A claims file with `hash` and a blank `hash_type`        | `claims hash_type is required when hash is supplied`                   |
| No `--name`, and none derivable from the input           | `claims name is required (use --name or provide via file/auto-hash)`   |
| Claims-only with short description, no metadata          | `claims content is required: provide --description of at least 32 ...` |
| `--metadata 'not json'`                                  | `--metadata must be valid JSON: ...`                                   |
| `--visibility` outside the allowed set                   | `--visibility must be private, team, or public, got "..."`             |
| `--url` not starting with `https://`                     | `claims url must start with https://`                                  |
| Claims-only exceeding the plan byte budget               | a server-side error naming the team owner's plan limit                 |

Every row except the last is caught client-side before any network
round-trip. The plan-budget error comes back from the server, because
the CLI can't predict the team owner's plan locally.

---

## `truestamp proofs get`

Fetch a proof bundle for an already-committed subject. `--type` is
**optional**: a ULID is unambiguously an item, so it is classified
client-side with no extra call. A UUIDv7 could be an entropy observation, a
block or a beacon, so the CLI resolves it against the server in one extra
round trip. Pass `--type` to skip that call, or when an id is verifiable as
more than one subject type — that case is refused rather than guessed.

The six `--type` values map 1:1 to the registry names the bundle carries in
its `type` field, and to the frozen code bound into the signed payload:

| `--type` | Wire value | Returned `type` (code) | Filename stem |
| --- | --- | --- | --- |
| `item` | `item` | `item` (20) | `item` |
| `entropy_nist` | `entropy_nist` | `entropy_nist` (30) | `entropy-nist` |
| `entropy_stellar` | `entropy_stellar` | `entropy_stellar` (31) | `entropy-stellar` |
| `entropy_bitcoin` | `entropy_bitcoin` | `entropy_bitcoin` (32) | `entropy-bitcoin` |
| `block` | `block` | `block` (10) | `block` |
| `beacon` | `beacon` | `beacon` (11) | `beacon` |

Flag values use **underscores**; generated filenames use **hyphens** (the CLI
translates between them so filenames stay readable). `--type entropy-nist`,
bare `--type entropy` and `--type auto` are all rejected. The `type` inside
the file is authoritative; the filename never is.

**Where the bundle goes.** With no output flag the bundle is written to
**stdout**, so it can be piped. `-o` / `--out` writes it to the path you
name. `--to-file` writes it to a conventionally-named file in the current
directory, `truestamp-<stem>-<id><variant>.<ext>`. `--out` and `--to-file`
are mutually exclusive. Writing CBOR straight to a terminal is refused
(`refusing to write CBOR to a terminal: redirect it, or pass -o <path> or
--to-file`); redirecting or naming a file is fine, so the refusal never
fires in a pipeline.

**`--witnesses`.** `--witnesses` selects which witness details an item
bundle carries. A witness is a public record that existed before the
submission and that the item's fingerprint commits to: the head block of
the ledger at submission, and the NIST, Stellar and Bitcoin entropy
observations captured then. The committed hashes always ride in
`subject.metadata`; the flag decides whether the details ride alongside
under `subject.witnesses`.

| `--witnesses` | Bundle | Filename |
| --- | --- | --- |
| `all` (default) | complete: every committed witness's detail, plus the signing key event | `truestamp-item-<id>.json` |
| `none` | compact: hashes only (the `signing_key_event` is dropped too) | `truestamp-item-<id>-compact.json` |
| `block,entropy_nist,...` | partial: the named subset | `truestamp-item-<id>-partial.json` |

Valid names: `block`, `entropy_stellar`, `entropy_nist`, `entropy_bitcoin`,
`signing_key_event`. All three variants are ordinary version 1 bundles and
all three verify. The only difference a verifier sees is how many Witnesses
and Submitted After rows it can report: verifying a compact bundle still
reaches `VERDICT: PASSED`, but every Witnesses row is a `skip` reading
`<name> committed in the subject metadata but not carried in this bundle`,
and Submitted After is an `info` reading `not established from this bundle:
no witness details carried; ...`. A compact bundle cannot open the
submitted-after edge of the submission window on its own.

The rule underneath is general: a witness the bundle cannot speak to is
reported as a check this run could not make, never as one that failed. A
bundle whose `subject.metadata` commits to no witness at all reads

```text
  [SKIP]  Witnesses            no witnesses: this subject's metadata commits to none and none is carried
  [INFO]  Submitted After      not established from this bundle: this subject's metadata commits to no witness
```

and still reaches a verdict on everything else. That is verifier
robustness rather than a shape to expect in your inbox: the service does
not currently emit bundles with no `witnesses` key, because an item
committed before the format was redefined around witnesses is refused at
generation with `subject_not_recomputable` instead (see the table above).

**Item proofs.** Item ids are ULIDs, so `--type` is inferred and no
resolution round trip is made. The ULID below is a **placeholder**:
substitute one of your own committed items (`truestamp items list`). No
output is shown for these lines because the placeholder id is not a real
item.

```sh
# Written to the conventional filename in the cwd,
# truestamp-item-<ulid>.json
truestamp proofs get 01KNN33GX5E470CB9TRWAYF9DD --to-file

# Same, with the type stated rather than inferred (identical behaviour)
truestamp proofs get --type item --to-file 01KNN33GX5E470CB9TRWAYF9DD

# The compact and a partial variant (-compact / -partial filename suffixes)
truestamp proofs get --witnesses none --to-file 01KNN33GX5E470CB9TRWAYF9DD
truestamp proofs get --witnesses block,entropy_nist --to-file 01KNN33GX5E470CB9TRWAYF9DD

# Override the auto-generated filename
truestamp proofs get 01KNN33GX5E470CB9TRWAYF9DD -o contract.proof.json

# CBOR: smaller, deterministic, ideal for embedding in another file
truestamp proofs get 01KNN33GX5E470CB9TRWAYF9DD -f cbor -o contract.proof.cbor
```

**Block, beacon and entropy proofs.** Blocks and beacons share one
structural shape — the same bundle keys — but carry distinct type codes in
the signed payload, so a block proof and a beacon proof for the same block
have different signatures. The bundle's own `type` says which one you got.

```sh
# Block proof for a committed block
truestamp proofs get --type block 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429 --to-file
#   Proof Downloaded
#
#        File  truestamp-block-01a06cfc-c5b6-77d5-a9c6-4ed42fef6429.json
#      Format  JSON (5,304 bytes)
#          ID  01a06cfc-c5b6-77d5-a9c6-4ed42fef6429
#        Type  block
#   Witnesses  all
#     Details  https://.../blocks/01a06cfc-c5b6-77d5-a9c6-4ed42fef6429
#      Verify  https://.../verify/block/01a06cfc-c5b6-77d5-a9c6-4ed42fef6429

# Beacon proof for the same block
truestamp proofs get --type beacon 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429 --to-file
#        File  truestamp-beacon-01a06cfc-c5b6-77d5-a9c6-4ed42fef6429.json
#      Format  JSON (5,305 bytes)
#          ID  01a06cfc-c5b6-77d5-a9c6-4ed42fef6429
#        Type  beacon
#   ...

# Same subject, CBOR: the extension follows --format
truestamp proofs get --type beacon -f cbor 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429 --to-file
#        File  truestamp-beacon-01a06cfc-c5b6-77d5-a9c6-4ed42fef6429.cbor
#      Format  CBOR (4,031 bytes)
#   ...

# --witnesses applies to the filename for every type; a block-like bundle
# has only the signing key event to drop, which takes it from 5,304 bytes
# down to 2,656
truestamp proofs get --type block --witnesses none 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429 --to-file
#        File  truestamp-block-01a06cfc-c5b6-77d5-a9c6-4ed42fef6429-compact.json
#      Format  JSON (2,656 bytes)
#          ID  01a06cfc-c5b6-77d5-a9c6-4ed42fef6429
#        Type  block
#   Witnesses  none
#   ...

truestamp proofs get --type block --witnesses block,entropy_nist 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429 --to-file
#        File  truestamp-block-01a06cfc-c5b6-77d5-a9c6-4ed42fef6429-partial.json
#      Format  JSON (2,656 bytes)
#          ID  01a06cfc-c5b6-77d5-a9c6-4ed42fef6429
#        Type  block
#   Witnesses  block,entropy_nist
#   ...

# -o names the path yourself
truestamp proofs get --type block 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429 -o block.proof.json
#        File  block.proof.json
#      Format  JSON (5,304 bytes)
#   ...

# Bare UUIDv7: one extra round trip resolves it, then the proof is fetched.
# With no output flag the bundle goes to stdout, so it redirects and pipes.
truestamp proofs get 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429 > block.proof.json

truestamp proofs get 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429 | truestamp verify --offline
# Bundle: (stdin) (json, 5304 bytes)
#   version=1  type=block
#   ...
#   12 passed   0 failed   0 warned   8 skipped   6 info
#   file hash provided: no
#
#   VERDICT: PASSED

# Entropy proofs take the UUIDv7 id of an entropy observation. The id below
# is illustrative only — no CLI command lists observation ids, so substitute
# one you already hold. Under --to-file each subtype writes
# truestamp-entropy-<subtype>-<uuidv7>.json.
truestamp proofs get --type entropy_stellar 019cf813-99b8-730a-84f1-5a711a9c355e
truestamp proofs get --type entropy_nist    019cf813-99b8-730a-84f1-5a711a9c355e
truestamp proofs get --type entropy_bitcoin 019cf813-99b8-730a-84f1-5a711a9c355e
```

**Failing fast.** The id shape is validated before the network call.
`--type item` requires a ULID; every other type requires a UUIDv7.
Mismatches, bad type names and conflicting output flags all fail locally
with exit code 1:

```sh
truestamp proofs get --type item 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429
# --type item requires a ULID id (e.g. 01KNN33GX5E470CB9TRWAYF9DD); got a UUIDv7

truestamp proofs get --type block 01KNN33GX5E470CB9TRWAYF9DD
# --type block requires a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b); got a ULID

truestamp proofs get --type entropy-nist 019cf813-99b8-730a-84f1-5a711a9c355e
# --type must be one of item | entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon, got "entropy-nist"

truestamp proofs get --type block -o x.json --to-file 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429
# --out and --to-file are mutually exclusive: --out names a path, --to-file picks the conventional name
```

A UUIDv7 that the server cannot resolve is reported with both ways out:

```sh
truestamp proofs get 019d6a32-13e6-72b0-97e5-3779231ea97b
# 019d6a32-13e6-72b0-97e5-3779231ea97b does not resolve to anything you can fetch a proof for; pass --type explicitly if you believe it should
# (or pass --type explicitly: entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon)
```

A proof exists only after the subject's first public-chain commitment.
Items commit to a Truestamp block within about a minute of submission and
to Stellar within about five; until then the server answers
`no_external_commitments` and the CLI says to try again after the next
epoch commit.

The refusals differ in whether waiting helps, and the CLI says which:

| `meta.code` | Meaning | Retry? |
| --- | --- | --- |
| `no_external_commitments` | not committed to a public chain yet | **yes**, clears in minutes |
| `subject_not_ready` | not yet in a state a proof can be built from | **yes**, clears on its own |
| `subject_not_recomputable` | the subject's stored data no longer reproduces the hash committed at submission, so no bundle could verify | **no**, permanent |
| `generation_failed` | the server built a bundle and its own self-verification rejected it | no; report it |

`subject_not_recomputable` also carries `meta.drifted` — `claims`,
`metadata`, or `claims and metadata` — and the CLI prints which half moved.
The commitment on the public chain is unaffected in that case; what is lost
is the deployment's ability to reproduce the preimage it committed to.
`generation_failed` carries `meta.failed_steps`, the failing verifier step
messages, which the CLI prints so the cause is named rather than bisected.

Read the code from `meta.code`, not from the JSON:API `code` member: the
latter is the generic `"invalid"` on every one of these. The two retryable
codes above are a closed set — anything else is terminal, and the CLI adds
no retry advice to a code it does not recognise.

If you only have a beacon hash and need its id, resolve it first with
`truestamp beacons get <hash>`, which accepts either shape.

---

## `truestamp verify`

Verify the full cryptographic chain of a proof bundle, JSON or CBOR: every
hash is recomputed from bytes the bundle carries (the subject's claims or
entropy payload, the subject and block metadata maps, the witness details),
the Merkle inclusion proof and each epoch proof are walked, the Ed25519
signature is checked over the derived values, the witnesses that open the
submitted-after edge are recomputed, the signing key event is checked, and
online the Stellar and Bitcoin commitments, each entropy witness's source,
and the keyring are consulted.

```sh
# Local file, JSON or CBOR (detected by content)
truestamp verify contract.proof.json
truestamp verify contract.proof.cbor

# URL, auto-detected from the positional argument
truestamp verify https://example.com/proof.json

# Stdin pipe
cat contract.proof.json | truestamp verify

# Interactive picker (needs a terminal; with stdin redirected it exits 1 with
# "--file without a path: interactive prompt requires a terminal (stdin is
# piped or redirected)")
truestamp verify --file

# Compare the hash of the file you hold against the hash the item's
# claims commit to; a mismatch fails the run
truestamp verify contract.proof.json \
  --expected-hash e08764deac64ca9a1046901c5b23674941f1e86f0e2d0429ee07c5e311a15ce7

# Offline: no calls to Truestamp, Stellar, Bitcoin or the entropy sources.
# Every cryptographic step still runs; each check that needs a source is
# reported as skipped, never failed.
truestamp verify contract.proof.json --offline

# Pin a local copy of the keyring so the key binding check works offline
curl -sO https://www.truestamp.com/.well-known/keyring.json
truestamp verify contract.proof.json --offline --keyring keyring.json
#   [PASS]  Key Binding          key_id 3c19f776 found in the pinned keyring (sequence 0, active true)

# The subject type always comes from the bundle's own signed `type`.
# The filename is never consulted: a file named truestamp-beacon-<id>.json
# may legitimately carry a `type: block` bundle, and renaming a file can
# never change a verdict.
truestamp verify truestamp-beacon-019d....json          # reports whatever type says
#   version=1  type=block

# Pass --type to assert which type you expected. A disagreement with the
# bundle's signed type is the hard rejection subject_type_mismatch.
truestamp verify --type beacon truestamp-beacon-019d....json
#   REJECTED: subject_type_mismatch
#   the bundle's signed type is block but --type beacon was asserted
#   The bundle's signed type is authoritative. Drop --type, or pass the type the
#   bundle carries; the filename is never consulted.
truestamp verify --type item   truestamp-item-01K....json

# Skip the signature check (disclosed in the report; never a full verify)
truestamp verify contract.proof.json --skip-signatures
#   [SKIP]  Proof Signature      proof signature verification skipped (--skip-signatures)
#   [WARN]  Proof Signature      this run establishes nothing about who signed the
#                                proof: the Ed25519 signature was not checked
#                                (--skip-signatures)

# Script-friendly modes
truestamp verify contract.proof.json --json        # structured output
truestamp verify contract.proof.json --silent      # exit code only
```

`--expected-hash` takes the hex digest of the file you hold; it is
trimmed, lowercased, and compared in constant time against the value the
bundle carries at `.subject.claims.hash`, reported under `Hash Comparison`.
Its four outcomes:

```text
[PASS]  Hash Comparison      supplied file hash matches subject.claims.hash
[FAIL]  Hash Comparison      supplied file hash does NOT match subject.claims.hash (proof commits to b47cc0f1...)
[WARN]  Hash Comparison      an expected hash was supplied but this proof commits to no file hash
[SKIP]  Hash Comparison      --expected-hash ignored: only an item subject commits to a file hash
```

The `WARN` is what an item that timestamped its claims content itself
produces: a warning, never a failure. The `SKIP` is what a block, beacon or
entropy bundle produces, since none of them commits to a file hash.

The report is the whitepaper's reference verifier's shape: a short bundle
header, then five categories in a fixed order (Data Integrity,
Cryptographic, Structural, Timing, Blockchain), each row a `[PASS]`,
`[FAIL]`, `[WARN]`, `[SKIP]` or `[INFO]` badge with its group and message,
then the counts, whether a file hash was provided, and the verdict. A
category with no rows is not printed, so a run with no `--expected-hash`
against a block-like bundle starts at `Cryptographic`.
A proof passes when no row fails; warnings, skips and infos never fail
it, and any skip is a check the run did not perform, not a check that
failed.

Under `--json` the document uses the field names the Truestamp API's
`/proof/verify` result uses, so a CLI report and an API report are directly
diffable: `passed`, `id`, `source`, `steps` (each with `group`, `status`,
`category`, `message`, in category order), `temporal`, `pass_count`,
`failed_count`, `warn_count`, `skip_count`, `info_count`, `hash_provided`,
`expected_hash_provided`, `hash_matched`, `proof_version`,
`skipped_external`, `generated_at`, plus `verifier` (this CLI's name and
version) and `signatures_checked`.

Use `--remote` to also ask the Truestamp server to verify the bundle
(requires authentication: `truestamp auth login` or `TRUESTAMP_API_KEY`).
The bundle header then reads `mode: remote (steps reported by the Truestamp
server)`. Local verification is the default, needs no credentials, and never
depends on the server: the server's verifier is not part of the proof's
independence argument. Even in `--remote` mode the CLI still:

- **Parses the bundle first**, applying the same structural gates, so a bundle it would refuse is never posted, and posts a CBOR input as its JSON conversion.
- **Asserts `--type` locally** as the same `subject_type_mismatch` rejection, then forwards it.
- **Performs the `--expected-hash` comparison itself**, treating the server's `hash_matched` as corroboration and failing on a disagreement in either direction.
- **Fails closed on a report it cannot read**: a step with no `status`, a status outside the five, zero steps, or a server verdict that contradicts its own step list is reported under a `Server Verdict` row rather than scored as passing.
- **Renders a server-side rejection** (HTTP 400 `invalid_proof` with an Appendix E.23 reason) exactly like a local one.

### Structurally malformed bundles

A bundle that is malformed at the structural level is refused before any
check runs, so there is no report to render, only the identifier, the
detail and one line of advice. Under `--json` the refusal is reported as
the stable identifier, so two independent verifiers can be compared on it:

```sh
echo '{"version":1,"type":"nope","block":{}}' | truestamp verify --json
```

```json
{
  "verifier": {
    "name": "truestamp-cli",
    "version": "..."
  },
  "passed": false,
  "rejection": {
    "code": "invalid_subject_type",
    "detail": "type is \"nope\"; expected one of block, beacon, item, entropy_nist, entropy_stellar, entropy_bitcoin",
    "advice": "This input is not a well-formed proof bundle, so no steps were run."
  },
  "steps": [],
  "pass_count": 0,
  "failed_count": 0,
  "warn_count": 0,
  "skip_count": 0,
  "info_count": 0,
  "hash_provided": null,
  "expected_hash_provided": false,
  "hash_matched": false,
  "proof_version": 0,
  "skipped_external": false,
  "signatures_checked": false
}
```

Without `--json` the same refusal prints as three lines:

```text
  REJECTED: invalid_subject_type
  type is "nope"; expected one of block, beacon, item, entropy_nist, entropy_stellar, entropy_bitcoin
  This input is not a well-formed proof bundle, so no steps were run.
```

Exit code `1`. The identifiers, in the order the gates apply, are
`not_a_json_object`, `invalid_subject_data` (CBOR only), `unsupported_layout`,
`invalid_subject_type`, `missing_block`,
`unexpected_subject_fields_for_block_like`, `missing_subject`,
`missing_inclusion_proof`, `missing_metadata`, `no_external_commitments`
and `invalid_commitment_entry`, plus the CLI-side `subject_type_mismatch`
for a `--type` that disagrees with the bundle.

`unsupported_layout` is the one to know: a bundle carrying a top-level `v`
or `t` key is in the pre-publication draft layout that version 1 replaced.
No verifier reads it. Ask the holder to regenerate the proof.

```sh
truestamp verify old-layout.json
#   REJECTED: unsupported_layout
#   a top-level v or t key is present: this is the pre-publication draft layout,
#   which version 1 replaced
#   This file uses the pre-publication draft layout, which version 1 replaced.
#   Ask the holder to regenerate the proof; no verifier reads that layout.

# Branch on the reason in a script
code=$(truestamp verify proof.json --json 2>/dev/null | jq -r '.rejection.code // empty')
[ -n "$code" ] && echo "bundle refused: $code"
#   bundle refused: unsupported_layout
```

---

## `truestamp inspect`

Print what a bundle carries without verifying anything: the format,
version and type, the ids, the witnesses the subject metadata commits to
and the details carried, the block, the commitments, and the signing key
event. No hash is recomputed and no signature is checked; the structural
gates still apply, so a malformed or draft-layout bundle is rejected with
the same identifier `verify` would report.

```sh
truestamp inspect proof.json
truestamp inspect proof.cbor --json
cat proof.json | truestamp inspect

# Which witness details does this bundle carry?
truestamp inspect proof.json --json | jq -r '.subject.carried_witnesses[]'

# Is a signing key event carried?
truestamp inspect proof.json --json | jq 'has("signing_key_event")'
```

```text
  Proof Bundle

  Source                 proof.json (json, 4356 bytes)
  Version                1
  Type                   item
  Generated at           2026-09-03T16:20:22Z
  Public key             1hHbF5H5u8LiSp+nVMRb8duR2eGkOo5Q1JYfcmAtF28=
  Derived key id         3c19f776

  Subject
  ──────────────────────────────────────────
  ID                     01M1M0V3SE3C5P32TRAJSNX6QF
  Signing key id         3c19f776
  Claims keys            description, name
  Committed witnesses    block  e36f824dea9508d5...
                         entropy_bitcoin  712b47e1ea379e09...
                         entropy_nist  ad333a58c6b61eac...
                         entropy_stellar  fd6df57b6afa30f6...
  Carried witnesses      block, entropy_bitcoin, entropy_nist, entropy_stellar
  Inclusion proof        5 steps

  Block
  ──────────────────────────────────────────
  Block id               01a0680e-42b6-712a-9137-5b59dc891f19
  Previous block hash    e36f824dea9508d5...
  Merkle root            d175d1ef04595ffb...
  Signing key id         3c19f776
  Metadata keys          {} (empty)

  Commitments
  ──────────────────────────────────────────
  Commitment 1           stellar public
    Epoch root           52c7d34ebc4291db...
    Epoch proof          3 steps
    Transaction          9a58b44146076484...
    Ledger               64256842
    Timestamp            2026-09-03T16:20:05Z

  Signing Key Event
  ──────────────────────────────────────────
  Carried                yes
  Key event              type genesis, sequence 0, key id 3c19f776
  Block id               019fcf1d-b17b-7897-adb4-3dd23adc9d0e
  Previous block hash    96118892a96ced7d...
  Merkle root            e3b0c44298fc1c14...
  Signing key id         3c19f776
  Metadata keys          key_event
  Commitment 1           stellar public
    Epoch root           b32c37da5b9f74b8...
    Epoch proof          3 steps
    Transaction          56efb2b977758ac0...
    Ledger               63802260
    Timestamp            2026-08-04T23:35:05Z
```

`--json` prints the same material as an object keyed
`source`, `format`, `bytes`, `version`, `type`, `generated_at`,
`public_key`, `key_id`, `subject`, `inclusion_proof`, `block`,
`commitments` and (when carried) `signing_key_event`. A CBOR bundle
reports `"format": "cbor"` and its own byte count. A compact bundle
carries no signing key event, so the `has("signing_key_event")` probe
prints `false` there and `true` on a complete one.

The structural gates run before anything is printed, so a bundle in the
pre-publication draft layout is refused here exactly as `verify` refuses
it:

```sh
truestamp inspect draft-layout.json
```

```text
  REJECTED: unsupported_layout
  a top-level v or t key is present: this is the pre-publication draft layout, which version 1 replaced
  This file uses the pre-publication draft layout, which version 1 replaced.
  Ask the holder to regenerate the proof; no verifier reads that layout.
```

Exit code `0` when the bundle parses, `1` when it is rejected.

---

## `truestamp teams`

Discover, switch between, and persist the active team the CLI sends as the
multitenancy context. The id is stored under the top-level `team` key in
`config.toml`, so it applies across invocations. Requires authentication.

```sh
# List every team you're a member of. A bare `truestamp teams` prints
# help: a group is a namespace, never a command.
truestamp teams list

# Detail card for the team the CLI is pointed at, or a specific one by id
truestamp teams current
truestamp teams get 019fcf1c-0f5f-7136-bde5-e1676bcdfdc8

# Create a team. With no name on a TTY, an interactive prompt opens.
truestamp teams create "Acme Legal"
truestamp teams create "Acme Legal" --ownership-model team_retains --use
truestamp teams create -n "Acme Legal" --json

# Set the active team. The id is read back from the API before it is
# persisted, so a typo or a revoked membership refuses to write.
truestamp teams use 019fcf1c-0f5f-7136-bde5-e1676bcdfdc8

# No id on a TTY opens an interactive picker; Esc cancels without writing
truestamp teams use

# Clear the active team (the server falls back to your personal team)
truestamp teams use --clear
```

```text
  Teams (1)

       NAME                   ROLE     TEAM ID
       Personal (personal)    Owner    019fcf1c-0f5f-7136-bde5-e1676bcdfdc8
```

`teams list` renders a four-column table (`active marker`, `NAME`, `ROLE`,
`TEAM ID`). The marker column is blank on every row, as above, when no
team is configured; once a team is named, by `teams use`, `--team` or
`TRUESTAMP_TEAM`, that row leads with `★`. `teams get` and `teams current`
render the same detail card, which adds Personal, Ownership, Created and a
public Details link (the link's host follows `base_url`, so the card below
shows a staging origin):

```text
  Team

         ID  019fcf1c-0f5f-7136-bde5-e1676bcdfdc8
       Name  Personal
       Role  Owner
   Personal  yes
  Ownership  creator_retains
    Created  2026-08-04T23:29:10Z
    Details  https://staging.truestamp.com/teams/019fcf1c-0f5f-7136-bde5-e1676bcdfdc8
```

`teams current` heads the identical card `Team (active)`, and exits `1`
when nothing names a team, neither `config.toml` nor `--team` nor
`TRUESTAMP_TEAM`:

```text
  No team configured
    Run 'truestamp teams use' to pick one interactively, or 'truestamp teams list' to see all available teams.
```

The server is the authority on which teams you can read, and each rejection
gets its own banner because each has a different fix: an unknown id gives
`Team not found`, a team you have no membership in gives `Access denied`,
and a rejected credential gives `Not authenticated`. All three exit `1`.

```text
  Team not found
    No team exists with that id. Run 'truestamp teams list' to see valid options.
```

`--ownership-model` accepts `creator_retains` (the default) or
`team_retains`, plus the short aliases `creator` / `team` and the
hyphenated `creator-retains` / `team-retains`, case-insensitively;
anything else is rejected client-side, before any request is sent:

```sh
truestamp teams create "Acme Legal" --ownership-model bogus_model
# invalid --ownership-model "bogus_model" (want "creator_retains" or "team_retains")
```

`team_retains` requires a plan entitlement, which only the server can
evaluate, so the CLI attempts the create and renders the server's own
error when the entitlement or the team-count limit blocks it.

All five sub-commands (`list`, `get`, `current`, `create`, `use`) accept
`--json` and `-s` / `--silent`.

---

## `truestamp console`

Interactive Bubble Tea TUI backed by an authenticated WebSocket to the
Truestamp server. Four panes share one long-lived connection:

- **Monitor** with toggleable subscriptions to live event streams (block
  lifecycle, internal/external commitments, NIST/Stellar/Bitcoin
  entropy observations, item events for your team) plus a scrollable,
  reversible event waterfall. Newest at top by default.
- **New Item**, a two-mode form (submission mode, name, description,
  plus hash type and hash in external-hash mode, hidden in
  claims-as-source-of-truth mode, where the description must be ≥ 32
  characters) that creates a timestamped item over the same socket and
  shows its live state transitions (`item.created`, `item.updated`,
  `item.committed`) below the card as they arrive.
- **Teams**, the same membership table `truestamp teams list` renders,
  with `enter` to switch the active team over the live socket and `c` to
  open a create-team modal.
- **Connection** with a scope summary (user, team, role, plan, server
  version, connected-for, active stream count), push counts by event,
  reconnect history, an external-services health table, and the log file
  path so you can `tail -f` it for live transport diagnostics.

Requires authentication (run `truestamp auth login` first, or set
`TRUESTAMP_API_KEY`). With no credential configured it refuses to start
before it opens a socket:

```sh
truestamp console
# not authenticated, run `truestamp auth login` (or set TRUESTAMP_API_KEY)
# exit 1
```

```sh
# Launch (uses your configured base_url + credentials)
truestamp console

# Point at a non-default backend (e.g. local dev)
truestamp console --ws-url ws://localhost:4000/console/websocket

# Send the log somewhere other than the default cache dir
truestamp console --log-file /tmp/truestamp-console.log

# Widen the shared log file. Note that nothing in the console path emits
# at debug today, so this mostly widens what the rest of the CLI and its
# dependencies write; the console's own transport diagnostics are
# info/warn and are already captured at the default level.
truestamp console --log-level debug
```

`--ws-url` is the only console-specific flag. `--log-level` and `--log-file`
are persistent root flags that every subcommand honors; the console consumes
the logger the root already built, tagged `component=console`. With `--ws-url`
unset, the WebSocket URL is derived from `--base-url` as
`ws(s)://<host>/console/websocket` (`http` maps to `ws`, `https` to `wss`) —
the same single origin every other service URL comes from.

The TUI needs a controlling terminal. Under CI, or anywhere `/dev/tty` is not
available, it exits 1 before it dials:

```text
bubbletea: error opening TTY: bubbletea: could not open TTY: open /dev/tty: device not configured
```

Before the TUI starts, when no team is configured **and** stdin is a TTY, the
console opens the same interactive team picker `truestamp teams use` uses and
persists the choice to `config.toml`. Esc dismisses it without writing,
leaving the server's personal-team fallback in place.

### Keys

Press `?` at any time to expand the footer into the full key list for the
active pane. The footer and its expanded form are rendered by `bubbles/help`
from the same bindings the panes consume, so the live hints are always
authoritative; the table below is a snapshot of them.

| Key                     | Action                                                               |
| ----------------------- | -------------------------------------------------------------------- |
| `1` / `2` / `3` / `4`   | Jump to Monitor / New Item / Teams / Connection                      |
| `]` / `[`               | Next / previous pane (also `ctrl+tab` / `ctrl+shift+tab`)            |
| `?`                     | Expand / collapse the footer help                                    |
| `q` / `ctrl+c`          | Raise the "Really quit?" prompt (`y` / `enter` confirms, any other key cancels, a second `ctrl+c` quits outright) |
| **Monitor pane:**       |                                                                      |
| `left`/`h`, `right`/`l` | Focus the Streams list / the Events waterfall (waterfall at launch)  |
| `up`/`k`, `down`/`j`    | Move within the focused side                                         |
| `space`                 | Toggle the cursor stream's subscription on/off (Streams list focused)|
| `pgup`/`K`, `pgdn`/`J`  | Page through the waterfall                                           |
| `home`/`g`, `end`/`G`   | Jump to top / bottom of the waterfall                                |
| `r`                     | Reverse chronological order (newest first or oldest first)           |
| `d`                     | Toggle the event detail panel                                        |
| **New Item pane:**      |                                                                      |
| `tab`/`down`, `shift+tab`/`up` | Move between form fields (plain `tab` is pane-local, not pane-switch) |
| `enter`                 | Advance, then submit on the last field                               |
| `esc`                   | Clear the form                                                       |
| `n` / `esc` (after submit) | Reset for another item                                            |
| **Teams pane:**         |                                                                      |
| `up`/`k`, `down`/`j`    | Move through the memberships table                                   |
| `enter`                 | Set the team under the cursor as active                              |
| `c`                     | Open the create-team modal                                           |
| `r`                     | Refresh the membership list                                          |
| **Connection pane:**    |                                                                      |
| `r`                     | Re-run the external-service health probes (one manual run per 3s)    |

`r` and `d` on the Monitor pane work from either focus. While the create-team
modal is open it is fully modal: it captures every key except `ctrl+c`,
`ctrl+tab` and `ctrl+shift+tab`, so `1`-`4`, `[`, `]`, `?` and `q` land in the
name field instead of switching panes (a team named "Q3" has to be typable).
For the same reason `q` never raises the quit prompt while a form field has
focus.

### What you see by default

On launch every catalog stream is auto-subscribed, so events start
flowing immediately. The header shows the numbered pane tabs on the
left and, on the right, the active team label, a liveness status pill
(`connecting…`, `connected`, `reconnecting in Ns (attempt N)`,
`disconnected`, or a classified error) and the server-time clock. The
footer shows the context-relevant key hints for the active pane.

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
`ID` column holds the plain integer event count; the rest of the
payload (`window_ms`, `first_at`, `last_at`, `by_kind`, `by_state`)
lives in the detail panel (`d`), with the maps flattened one level
deep so you see `by_kind.item.created` rather than a nested blob.

Slow streams (blocks, entropy, external commitments) almost never
trigger this. The first-event-immediate rule means a burst only
emerges when input rate genuinely warrants summarization. The client
never decides whether to coalesce: it renders any `*.burst` kind that
arrives with the same row template as everything else.

### Reconnect

If the network blips or the server restarts, the client reconnects
automatically over a fixed capped backoff schedule (`1s`, `2s`, `5s`,
`10s`, `30s`, with the last value repeating). The header shows a live
countdown such as `reconnecting in 7s (attempt 4)`, and `server.down`
markers drop into the waterfall every 10 seconds during the outage so
you can scroll back later and see exactly when data went missing. On
reconnect a closing `server.up` marker records the total downtime in
the `ID` column, every previously active subscription is re-issued
automatically, and the Connection pane's reconnect counter increments.

### Logs

Transport diagnostics (read EOFs during a server restart, dial
attempts during reconnect, frame decode errors) write to a rotated
JSON-lines log file rather than the UI. It is the same file every
other subcommand logs to, so filter on `component`:

```sh
# macOS
tail -f ~/Library/Caches/truestamp/truestamp.log | jq 'select(.component == "console")'

# Linux
tail -f ~/.cache/truestamp/truestamp.log | jq 'select(.component == "console")'
```

A launch, right down to the derived WebSocket URL, is visible in the file
before the first frame is drawn:

```sh
truestamp console --log-file /tmp/truestamp-console.log
jq -c 'select(.component == "console")' /tmp/truestamp-console.log
# {"time":"...","level":"INFO","msg":"command_start","component":"console","cmd":"truestamp console","argc":0,"version":"0.13.0-...","install_method":"unknown"}
# {"time":"...","level":"INFO","msg":"console session start","component":"console","ws_url":"wss://staging.truestamp.com/console/websocket","log_path":"...","version":"0.13.0-..."}
# {"time":"...","level":"ERROR","msg":"command_end","component":"console","duration_ms":4,"exit":1,"err":"bubbletea: error opening TTY: ..."}
```

(`ws_url` is whatever your `--base-url` derives to; the run above was pointed
at staging, and ended early because it had no controlling terminal.)

Defaults: 10 MB rotation, 14-day retention, 5 backups, gzip-compressed.
The Connection pane shows the live path. **Credentials are redacted**
before any error or log line touches the file or screen, covering the
API key, OAuth access and refresh tokens, and the PKCE code and
verifier — including the `?api_key=` / `?access_token=` query params a
WebSocket dial error would otherwise echo back verbatim.

### Hand-rolled testing

The wire protocol is plain JSON arrays over Phoenix Channels V2 and
is fully driveable from `websocat`. See
[kb/console/websocket.md](kb/console/websocket.md) for the
client-side architecture (auth on the upgrade, heartbeats, ref
correlation, reconnect, redaction) and the parts of the wire protocol the
client relies on, and [kb/console/panes.md](kb/console/panes.md) for the
pane and key-binding reference. The server's full protocol reference
(commands, events, catalog, limits) is not published.

---

## `truestamp hash`

Multi-algorithm digest tool. Default output is **byte-identical to
`sha256sum`** so it drops into existing scripts.

```sh
# SHA-256 a file (default algorithm)
truestamp hash contract.pdf
# <hex>  contract.pdf

# Multiple files at once, one line per file
truestamp hash a.bin b.bin c.bin

# Stdin (filename shows as "-", matching sha256sum)
echo -n "abc" | truestamp hash
# ba7816bf8f01cfea...15ad  -

# Same, via the explicit Unix "-" stdin convention
echo -n "abc" | truestamp hash -

# List supported algorithms
truestamp hash --list

# Pick any supported algorithm
truestamp hash -a sha3-256 contract.pdf
truestamp hash -a blake2b-512 contract.pdf
truestamp hash -a md5 contract.pdf             # warns on stderr: legacy algorithm

# BSD-style tagged output (shasum --tag compatible)
truestamp hash -a sha256 --style bsd contract.pdf
# SHA256 (contract.pdf) = <hex>

# Bare digest (no filename, no separator)
truestamp hash -a sha256 --style bare contract.pdf

# GNU binary mode: "<hex> *<filename>" instead of two spaces
truestamp hash --binary contract.pdf

# Drop the filename column entirely from gnu/bsd output
truestamp hash --no-filename contract.pdf

# Pick the output encoding (default: hex); -e is the short form
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

Other input modes match `truestamp verify`: `--file <path>` (or `--file`
alone for the interactive picker) and `--url <url>` (or `--url` alone for the
interactive prompt).

`--list` prints 14 algorithms with their digest sizes, any aliases, and a
`[legacy]` marker on `md5` and `sha1`:

```text
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
(`warning: md5 is cryptographically broken and unsuitable for security uses`);
stdout and the exit code are unaffected. It is suppressed under `--json` and
`--silent`.

An unknown `-a` value is rejected before any input is read, with the
supported list echoed back:

```sh
truestamp hash -a nope contract.pdf                # exit 1
```

```text
unknown hash algorithm "nope" (supported: md5, sha1, sha224, sha256, sha384, sha512, sha3-224, sha3-256, sha3-384, sha3-512, blake2s-256, blake2b-256, blake2b-384, blake2b-512)
```

`--json` emits the digest in all three encodings alongside the input
description:

```sh
printf abc > abc.txt
truestamp hash -a sha256 --json abc.txt
```

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
  "input": {
    "type": "file",
    "path": "abc.txt"
  }
}
```

`encoded` repeats whichever of the three encodings `--encoding` selected, and
`input` becomes `{"type": "stdin", "path": "-"}` when the bytes came from a
pipe. Exit code is 0 when every input hashed, 1 if any input failed;
`--json` and `--silent` are mutually exclusive.

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

# Same file, via the explicit --file flag
truestamp encode --file contract.pdf --to base64

# Decode base64 back to binary (raw bytes, no trailing newline added)
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
padded or the unpadded form and tolerates a trailing newline. `input_bytes`
counts the bytes actually read, so a trailing newline the decoder then
ignores is still included in that count.

---

## `truestamp jcs`

Apply **RFC 8785 JSON Canonicalization**. Output is the byte-stable form that
Truestamp uses when computing claims / entropy / metadata hashes.

```sh
# Canonicalize (sorts keys, normalizes whitespace and number formatting)
echo '{"b":2,"a":1,"c":[3,1,2]}' | truestamp jcs
# {"a":1,"b":2,"c":[3,1,2]}

# Plain output carries NO trailing newline, so it pipes straight into a hash.
# --newline appends one (for appending to a line-oriented stream)
truestamp jcs --newline < claims.json

# JSON envelope
echo '{"b":2,"a":1,"c":[3,1,2]}' | truestamp jcs --json
# {
#   "input_bytes": 26,
#   "output_bytes": 25,
#   "output": "{\"a\":1,\"b\":2,\"c\":[3,1,2]}"
# }
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
suppresses all output including the warning; `--json` replaces it with an
`oversized_integers` array and writes nothing to stderr:

```sh
printf '{"n":9007199254740993}' | truestamp jcs --json | jq -c .oversized_integers
# ["9007199254740993"]
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
# Current time in UTC (microsecond precision)
truestamp convert time now
# 2026-09-04T15:19:57.496987Z

# Unix seconds to RFC 3339 UTC
truestamp convert time 1700000000
# 2023-11-14T22:13:20Z

# Convert to another time zone
truestamp convert time 1700000000 --to-zone America/New_York
# 2023-11-14T17:13:20-05:00

# Force the interpretation of numeric input
truestamp convert time 1700000000000 --from unix-ms
# 2023-11-14T22:13:20Z
truestamp convert time 1700000000000000000 --from unix-ns
# 2023-11-14T22:13:20Z

# Change the output format
truestamp convert time "2026-04-21T12:00:00Z" --format unix-s
# 1776772800
truestamp convert time "2026-04-21T12:00:00Z" --format unix-ms
# 1776772800000
truestamp convert time "2026-04-21T12:00:00Z" --format "2006-01-02 15:04:05"  # Go layout
# 2026-04-21 12:00:00

# Read the timestamp from stdin (useful in pipelines)
date -u +%s | truestamp convert time --to-zone Asia/Kolkata
# 2026-09-04T20:50:02+05:30

# JSON output with all representations
truestamp convert time 1700000000 --json
# {
#   "input": "1700000000",
#   "parsed_utc": "2023-11-14T22:13:20Z",
#   "output_zone": "UTC",
#   "output": "2023-11-14T22:13:20Z",
#   "unix_s": 1700000000,
#   "unix_ms": 1700000000000
# }
```

`--from` accepts `auto` (default), `rfc3339`, `unix-s`, `unix-ms`, `unix-us`
and `unix-ns`. `--format` accepts the same Unix variants plus `rfc3339`
(default) or any Go time layout. `--to-zone` takes any IANA zone name
(including `Local`) and defaults to `UTC`. The `--json` envelope carries
`input`, `parsed_utc`, `output_zone`, `output`, `unix_s` and `unix_ms`.

---

## `truestamp proofs convert`

Convert a proof bundle between JSON and CBOR. The conversion applies
Appendix E.3's field-type correspondence: `public_key` and `signature` are
byte strings in CBOR and padded base64 in JSON; every hash slot
(`signing_key_id`, `previous_block_hash`, `merkle_root`,
`epoch_merkle_root`, `transaction_hash`, `block_merkle_root`) is a byte
string in CBOR and lowercase hex in JSON; every map that is a JCS preimage
(claims, entropy, both metadata maps, witness payloads) keeps the JSON value
space in both. Every number survives exactly. The CBOR output uses **RFC 8949
section 4.2 core deterministic encoding** and is prefixed with the
self-describing tag 55799 so `truestamp verify` auto-detects it; JSON output
preserves the input's key order.

```sh
# JSON to CBOR
truestamp proofs convert --to cbor proof.json > proof.cbor
# the first three bytes are the 55799 self-describing tag:
#   $ xxd proof.cbor | head -1
#   00000000: d9d9 f7aa 6474 7970 6564 6974 656d 6562  ....dtypeditemeb

# CBOR to JSON (auto-detected input format)
truestamp proofs convert --to json proof.cbor | jq .

# Force the input format (error out if the bytes don't match)
truestamp proofs convert --from json --to cbor < proof.json
# feeding it the wrong format is a clean exit 1:
#   $ truestamp proofs convert --from cbor --to json proof.json
#   parsing CBOR proof: not_a_json_object: input is not a CBOR map

# Compact JSON (minified; the default is a 2-space indent)
truestamp proofs convert --to json --compact proof.cbor
# {"type":"item","block":{"id":"01a0680e-42b6-712a-9137-5b59dc891f19",...

# Round-trip verification (the CBOR output must verify end-to-end)
truestamp proofs convert --to cbor proof.json | truestamp verify --offline
# ...
#   VERDICT: PASSED

# Envelope mode: metadata plus the output as a base64url string
truestamp proofs convert --to cbor --json proof.json
# {
#   "input_format": "json",
#   "output_format": "cbor",
#   "input_bytes": 4356,
#   "output_bytes": 2961,
#   "canonical_cbor": true,
#   "output_wrapped": "2dn3qmR0eXBlZGl0ZW1lYmxvY2..."
# }
```

`--to` is required (omitting it exits 1 with `--to is required (json or
cbor)`). `--from` accepts `auto` (default), `json` or `cbor`. The input comes
from the positional path, `--file`, `--url` or stdin, exactly as for
`truestamp verify`.

---

## `truestamp convert id`

Extract the embedded timestamp from a ULID (item IDs) or UUIDv7 (block,
beacon and entropy IDs). Truestamp uses ULIDs for item-style subjects and
UUIDv7 for blocks, beacons and entropy observations.

```sh
# ULID to embedded timestamp (UTC)
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD
# 2026-04-07T23:05:39.493Z

# UUIDv7 to timestamp
truestamp convert id 019cf813-99b8-730a-84f1-5a711a9c355e
# 2026-03-16T19:16:00.056Z

# Convert to a specific zone
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD --to-zone Local
# 2026-04-07T19:05:39.493-04:00   (depends on your local zone)

# Force the parser if you have an unusual form (hyphenless UUID, etc.)
truestamp convert id 019d6a3213e672b097e53779231ea97b --type uuid7
# 2026-04-07T23:06:00.038Z

# Extract raw bytes as hex instead of the timestamp
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD --extract raw
# 019d6a31c3a5710e062d3ac715e7a5ad

# JSON output with every representation
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD --json
# {
#   "type": "ulid",
#   "value": "01KNN33GX5E470CB9TRWAYF9DD",
#   "timestamp_ms": 1775603139493,
#   "timestamp_utc": "2026-04-07T23:05:39.493Z",
#   "zone": "UTC",
#   "local": "2026-04-07T23:05:39.493Z"
# }

# From stdin (common when fed from jq)
jq -r .subject.id proof.json | truestamp convert id
# 2026-09-03T16:15:14.222Z
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

# Auto-detect the encoding (hex / base64 / base64url); same key, base64url
truestamp convert keyid CTwMqDZnPd_QTLSq8aTeSD3a-j2DQxKcGfhhIYJQ65Y
# 4ceefa4a

# JSON output carries all three encodings alongside the kid
truestamp convert keyid --json CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=
# {
#   "public_key_hex": "093c0ca836673ddfd04cb4aaf1a4de483ddafa3d8343129c19f861218250eb96",
#   "public_key_base64": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
#   "public_key_base64url": "CTwMqDZnPd_QTLSq8aTeSD3a-j2DQxKcGfhhIYJQ65Y",
#   "kid_hex": "4ceefa4a"
# }

# From stdin
jq -r .public_key proof.json | truestamp convert keyid
# 3c19f776
```

`--from` accepts `auto` (default), `hex`, `base64` or `base64url`.

---

## `truestamp convert merkle`

Decode a compact base64url Merkle proof (the `inclusion_proof` field of an
item or entropy proof, or the `epoch_proof` inside each commitment entry)
into a human-readable sibling list.

```sh
# Positional argument
truestamp convert merkle "$(jq -r .inclusion_proof proof.json)"
# depth: 5
#    0  right  7251f2b6c01afe3d1f1b4723566641f7f2590e8caa86be1706b8b291d71dc386
#    1  right  9a56135841b89f111a2c9eb344c7023fdbc24eb3e7504a00f0bd67255a2dffe3
#    2  right  265788d3f3f9a03ce8452be58c9cbfc85b739cf1f5216c0eef3b147309bd5f45
#    3  right  845b98c0469e9bfc51fcd6756131c6b612624bffdb4d729808ab1359cc948647
#    4  right  33010ae04a22ef2505c2e4079f633491eae67f6a39e828922935cf56776eb494

# From stdin (common when fed from jq)
jq -r .inclusion_proof proof.json | truestamp convert merkle

# The epoch proof inside a commitment entry decodes the same way
jq -r '.commitments[0].epoch_proof' proof.json | truestamp convert merkle
# depth: 3
#    0  right  f007826901b88c81414827f12ca8a01b85f7545de9e88c5dc714e7511847555a
#    1  right  cce754aceea4171af6a274ae860db6ba9e04fc4a4a57d037a2645668aa528e0e
#    2  right  0985f048a86ae95dfca0940bd06e69eaf39ce2ff9f524d28fdd164a0fec11ca6

# JSON envelope (depth, siblings with position + hash)
jq -r .inclusion_proof proof.json | truestamp convert merkle --json
# {
#   "compact_base64url": "BR9yUfK2wBr-PR8bRyNWZkH38lkOjKqGvhcGuLKR1x3D...",
#   "depth": 5,
#   "siblings": [
#     {
#       "position": "right",
#       "hash_hex": "7251f2b6c01afe3d1f1b4723566641f7f2590e8caa86be17..."
#     },
#     ...
#   ]
# }
```

The text form prints `depth: N` followed by one indexed line per sibling with
its `left` / `right` position and hex hash. The `--json` form carries
`compact_base64url`, `depth`, and a `siblings` array of
`{position, hash_hex}` objects. A value that is not valid base64url exits 1
with `base64url decode failed: illegal base64 data at input byte N`; a value
that decodes cleanly but is the wrong length for its declared depth exits 1
with `expected 162 bytes for depth 5 proof, got 160`.

---

## `truestamp beacons`

Inspect Truestamp block beacons via the read-only JSON:API at
`/api/json/beacons/*`. A **beacon** is a compact projection of a
finalized block, four fields only: `{id, hash, timestamp,
previous_hash}`. It's a "proof of life" commitment: every item and
entropy observation finalized inside that minute window is covered by
the beacon's hash.

Full verifiable proof bundles for a beacon are a separate artefact
fetched via `truestamp proofs get --type beacon <id>` (see above).

```sh
# Most recent finalized beacon. A bare `truestamp beacons` prints help:
# a group is a namespace, never a command.
truestamp beacons latest
#   Beacon
#
#        Hash  79ac0abb5f8c295054e58be910f6c334e279eae249346a2c6addd3a33ff95d8d
#   Timestamp  2026-09-04T15:53:00Z  (6s ago)
#          ID  01a06d1f-8fb4-710a-894d-3412252cdcd8
#    Previous  f30efc591419a999a40927298a75432a05feaa9fb5fc858c9c84cda6993669f0
#   (two more rows follow, Details and Verify — see below)

# Most-recent N beacons, newest first (default 25, at most 250 per page).
# Paging is the same as every other list, see
# Conventions → Paging lists: --after / --before, --oldest-first (the
# genesis beacon first), --max and --count.
truestamp beacons list
truestamp beacons list --limit 3
truestamp beacons list --oldest-first --limit 3
#   Beacons (3)
#
#   TIMESTAMP               HASH                                                                ID
#   2026-09-04T15:53:00Z    79ac0abb5f8c295054e58be910f6c334e279eae249346a2c6addd3a33ff95d8d    01a06d1f-8fb4-710a-894d-3412252cdcd8
#   2026-09-04T15:52:00Z    f30efc591419a999a40927298a75432a05feaa9fb5fc858c9c84cda6993669f0    01a06d1e-a589-7947-b856-15e7b6a3779d
#   2026-09-04T15:51:00Z    1b7a533329e0cbcf5b8a744c2eac1e0ddc1f8fecd90a869e9506942dd9ed58f0    01a06d1d-bb56-7999-972c-48add5ca8ba9

# Look up by UUIDv7 id
truestamp beacons get 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429

# Look up by 64-hex-char hash, useful when all you have is a hash
# (e.g. printed on a receipt or read from photo metadata). The two id
# shapes are disjoint, so both of these address the same record and
# print the same card.
truestamp beacons get c001cf0c7eb2575b72cf3a56685f6a6218b0068b4ef79c7ce9f93bf1c578327f
```

Shared flags (all three subcommands): `--json` (raw JSON, pipeline
friendly), `--silent` / `-s` (exit code only). On `latest` and `get`
only: `--hash-only` (prints just the hash + newline for shell
substitution). `--silent` + `--json`, `--silent` + `--hash-only`, and
`--json` + `--hash-only` are each rejected as mutually exclusive.
`--hash-only` is not a flag on `beacons list` at all (a list has no
single hash), so passing it is `unknown flag: --hash-only`.

The single-beacon card prints Hash, Timestamp (with a relative age), ID
and Previous, then two shareable public-web links: Details at
`/beacons/<hash>` and Verify at `/verify/beacon/<id>`. Both are built
from `base_url`, so they name whichever origin the CLI is pointed at —
`https://www.truestamp.com` under the default configuration.
`beacons list` renders a `TIMESTAMP` / `HASH` / `ID` table.

```sh
# Capture the current chain head as a moment-in-time commitment.
# --hash-only lives on the subcommand, not on the bare group.
MOMENT=$(truestamp beacons latest --hash-only)
echo "beacon hash: $MOMENT"
# beacon hash: 79ac0abb5f8c295054e58be910f6c334e279eae249346a2c6addd3a33ff95d8d

# Pipeline-friendly JSON (the paged envelope for `list`, an object otherwise)
truestamp beacons list --limit 10 --json | jq -r '.beacons[].hash'
# 79ac0abb5f8c295054e58be910f6c334e279eae249346a2c6addd3a33ff95d8d
# f30efc591419a999a40927298a75432a05feaa9fb5fc858c9c84cda6993669f0
# 1b7a533329e0cbcf5b8a744c2eac1e0ddc1f8fecd90a869e9506942dd9ed58f0
# ...

# Round-trip: id to hash to id (demonstrates by-hash lookup)
ID=$(truestamp beacons latest --json | jq -r .id)
HASH=$(truestamp beacons get "$ID" --hash-only)
truestamp beacons get "$HASH" --json | jq .id
# "01a06d1f-8fb4-710a-894d-3412252cdcd8"
```

Client-side validation catches obvious typos without hitting the
network:

```sh
truestamp beacons get not-a-uuid
# invalid UUID: uuid: incorrect UUID length

truestamp beacons get ABCDEF
# "ABCDEF" is neither a UUIDv7 id nor a 64-hex-char beacon hash
```

An id that is well-formed but unknown is a server answer, not a
client-side one:

```sh
truestamp beacons get 019db8b5-90a1-7015-a62c-48e5038f2306
# beacon not found
# exit 1
```

All beacon subcommands require authentication. Run `truestamp auth login`,
or set `TRUESTAMP_API_KEY` / `--api-key` for headless and CI use. Without a
credential the command prints a "Not authenticated" banner to stderr and
exits `1`.

---

## `truestamp items` (list, get, update)

Beyond `items create`, the group reads and updates items. Commitment state
is the question most often asked, so it is a column and a filter rather
than a separate command.

```sh
# The items in the team the CLI is pointed at
truestamp items list
#   Items (3)
#
#   01M1PHJRN546DKV7QYAW7RAP2B  ✓ committed  contract.pdf
#   01M1PHJR9R51VQ7JPTR9RNAMS3  ✓ committed  contract.pdf
#   01M1M74XBVCAMMWGY8SZJD7YPZ  ✓ committed  CLI parity round trip

# Filter on commitment state: only the ones a proof can be generated
# for, or only the ones still waiting for a block. Every item on this
# team is already committed, so --committed repeats the three rows above
# and --pending finds nothing.
truestamp items list --committed
truestamp items list --pending
#   No items.

# Page explicitly. A short page prints the cursor that continues it.
truestamp items list --limit 2
#   Items (2)
#
#   01M1PHJRN546DKV7QYAW7RAP2B  ✓ committed  contract.pdf
#   01M1PHJR9R51VQ7JPTR9RNAMS3  ✓ committed  contract.pdf
#     More: --after g2wAAAABbQAAABowMU0xUEhKUjlSNTFWUTdKUFRSOVJOQU1TM2o=

truestamp items list --after g2wAAAABbQAAABowMU0xUEhKUjlSNTFWUTdKUFRSOVJOQU1TM2o=
truestamp items list --limit 100

# ...or follow cursors until a cap you choose. There is no unbounded walk:
# the tables behind these lists grow by the minute, so --max is the price
# of following pages. Each page is --limit rows; the last asks for only the
# rows still wanted, so the cursor printed at the end continues exactly.
truestamp items list --max 200 --json | jq -r '.items[].id'
# 01M1PHJRN546DKV7QYAW7RAP2B
# 01M1PHJR9R51VQ7JPTR9RNAMS3
# ...

# --count adds the server's total to the heading (and "total" to --json)
truestamp items list --limit 2 --count
#   Items (2 shown, 3 total)

# Walk from the beginning instead, and step back with the Back: cursor a
# page prints (--before is the mirror of --after)
truestamp items list --oldest-first --limit 2
truestamp items list --before g2wAAAABbQAAABowMU0xUEhKUjlSNTFWUTdKUFRSOVJOQU1TM2o=

# One item, including whether a proof is available yet
truestamp items get 01M1M74XBVCAMMWGY8SZJD7YPZ
#   Item
#
#            ID  01M1M74XBVCAMMWGY8SZJD7YPZ
#         State  committed  (a proof can be generated)
#    Visibility  private
#          Name  CLI parity round trip
#   Claims Hash  29e9ecea180843632df386dd546535b26cf6f3d8fb1654112fbca79b9073bf1c
#     Item Hash  8286350eb51f5b1ed51b4d109b6e04f3ea0c89dd520cd1510102281fa055af46
#          Team  019fcf1c-0f5f-7136-bde5-e1676bcdfdc8
#       Created  2026-09-03T18:05:26Z  (23h ago)
#   (one more row follows, Details — see below)
#   A Tags row appears here too when the item carries any.
```

The item card closes with the same kind of link the beacon card does: a
Details URL at `/items/<id>`, on whichever origin `base_url` names.

`items list --json`, like every keyset-paged list (`blocks list`,
`beacons list`, `entropy list`), is an object: `{"items": [...], "next_cursor": "...",
"prev_cursor": "..."}`, plus `"total"` under `--count`, with `next_cursor`
empty once the last page has been read and `prev_cursor` empty on the
first. Reach for `.items[]` in `jq`, not `.[]`.
`--limit` is bounded on one side only: the
CLI refuses `--limit 0` locally, because the API contract states a minimum
of 1, and forwards everything else for the server to accept or refuse. The
ceiling is the server's and it names its own when it declines.

Only three attributes are mutable — visibility, tags, and the owning team.
Claims are immutable because `claims_hash` is signed, and there is no flag
on `update` that can reach them. Substitute one of your own item ids for
the placeholder below: these three lines change stored state.

```sh
truestamp items update 01KNN33GX5E470CB9TRWAYF9DD --visibility public
truestamp items update 01KNN33GX5E470CB9TRWAYF9DD --tags q3,contracts

# Move it to another team. --to-team, not the root --team: that one says
# which tenant scopes the request.
truestamp items update 01KNN33GX5E470CB9TRWAYF9DD --to-team 019dbd00-0000-7000-8000-000000000000
```

A server-reported state is a claim. The authoritative answer is a proof
you check yourself — again with your own item id in place of the
placeholder:

```sh
truestamp proofs get 01KNN33GX5E470CB9TRWAYF9DD | truestamp verify
```

## `truestamp blocks`

Read-only access to Truestamp's block chain. A block is the full signed
record — Merkle root, state, signature, key id, chain links — where a
beacon is its four-field public projection.

```sh
truestamp blocks latest       # the head block, whatever its state
truestamp blocks genesis      # the first block, where every chain walk ends
truestamp blocks list --limit 10
truestamp blocks list --oldest-first --limit 3            # genesis first
truestamp blocks list --limit 100 --max 1000 --count --json | jq -r '.blocks[].id'
truestamp blocks get 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429
truestamp blocks get c001cf0c7eb2575b72cf3a56685f6a6218b0068b4ef79c7ce9f93bf1c578327f --json
```

Either address form works: a UUIDv7 has hyphens, a block hash is exactly
64 lowercase hex characters, so the two shapes are disjoint. `blocks list`
pages like every other list (see [Paging lists](#paging-lists)): `--after`
and `--before` continue from a printed cursor, `--oldest-first` starts at
genesis, `--max` caps a multi-page walk, and `--count` reports the chain's
length as the total. `--limit` defaults to 25 and is forwarded as given;
only `--limit 0` is refused locally. An id that does not resolve is an
error, not an empty result:

```sh
truestamp blocks get 019db702-b08c-73dc-a7cd-2c5e011f1dad
# block not found      (exit 1)
```

`blocks` and `beacons` address the same rows, but not the same set of
them: only finalized or committed blocks project as beacons, so
`blocks latest` (the head block, routinely not yet finalized) and
`beacons latest` (the newest finalized block) usually return different
rows, and `blocks list` includes rows `beacons list` does not. They
differ in shape too. A block carries the
whole signed record (`id`, `state`, `block_hash`, `merkle_root`,
`previous_block_id`, `previous_block_hash`, `signing_key_id`,
`signature`, `inserted_at`); a beacon is the four-field projection
(`id`, `hash`, `timestamp`, `previous_hash`).

A block has a proof of its own, but only once it has been committed to a
public chain. Feeding `blocks latest` into `proofs get` is therefore
unreliable: the head is often still `finalized`, and `proofs get` on a
subject with no commitment yet exits 1 with "Subject has not yet been
committed to a public blockchain". Pin a block that has already reached
`committed`:

```sh
truestamp proofs get 01a06cfc-c5b6-77d5-a9c6-4ed42fef6429 | truestamp verify --offline
# Bundle: (stdin) (json, 5304 bytes)
#   version=1  type=block
#   ...
#   VERDICT: PASSED
```

## `truestamp entropy`

Inspect the entropy observations Truestamp witnesses, via the read-only
JSON:API at `/api/json/entropy_observations`. An **entropy observation**
is a public random value captured from an independent source together
with the moment it was captured: a NIST Randomness Beacon pulse
(`entropy_nist`), a Stellar ledger close (`entropy_stellar`), a Bitcoin
block (`entropy_bitcoin`). Every item commits to the newest observation
per source at submission, which opens the **submitted-after** edge of its
submission window: the item cannot have been submitted before a value
that did not yet exist.

The same observations are published on the web at `<base-url>/entropy`.
The `--source` vocabulary is the wire names, identical to the
`entropy_*` proof subject types, so one word names both; the card and the
list drop the `entropy_` prefix for display (`stellar`), and `--json`
keeps the wire name.

```sh
# Newest observation from any source. A bare `truestamp entropy` prints help.
truestamp entropy latest
#   Entropy Observation
#
#             ID  01a07349-c2b3-7290-8cbe-e392bf94cd9d
#         Source  stellar
#          State  created
#   Entropy Hash  27bce1e57b76e5092704be389273259b05870d3c705a38c7fe0dd41f407d39c2
#      Published  2026-09-05T20:36:47Z  (5s ago)
#       Captured  2026-09-05T20:36:48Z  (3s ago)
#    Signing Key  96b1cd2f
#      closed_at  2026-09-05T20:36:47Z
#           hash  e1e983b5b0ea0281e081a48efa33e9ff868be82cb5db0198999acafe5552be2d
#   paging_token  19427872246595584
#       sequence  4523404
#        Details  https://www.truestamp.com/entropy/01a07349-c2b3-7290-8cbe-e392bf94cd9d
#         Verify  https://www.truestamp.com/verify/entropy_stellar/01a07349-c2b3-7290-8cbe-e392bf94cd9d

# Newest from one source: what a newly submitted item would commit to for it
truestamp entropy latest --source entropy_nist
truestamp entropy latest --source entropy_bitcoin --json | jq -r .entropy_hash

# Most recent observations, newest first, across sources or from one
truestamp entropy list --limit 4
#   Entropy Observations (4)
#
#   PUBLISHED               SOURCE     STATE      ID                                      HASH
#   2026-09-05T20:36:47Z    stellar    created    01a07349-c2b3-7290-8cbe-e392bf94cd9d    27bce1e5…407d39c2
#   2026-09-05T20:36:42Z    stellar    created    01a07349-af28-75fc-8edc-f62b8097010d    a5f4dd3e…6be8edb3
#   2026-09-05T20:36:37Z    stellar    created    01a07349-9b9e-7d40-90f2-493759b1865a    bdcb9659…8a4e8d40
#   2026-09-05T20:36:00Z    nist       created    01a07349-8a08-7862-8064-48350f95c400    c4b3baf8…89bc82a7
truestamp entropy list --source entropy_bitcoin --limit 2

# Page like every other list: --after and --before continue from a printed
# cursor in either direction, --oldest-first starts at the first observation
# ever captured, --max follows cursors up to a cap, --count adds the total
truestamp entropy list --oldest-first --limit 3
truestamp entropy list --source entropy_bitcoin --limit 50 --max 200 --json | jq -r '.observations[].entropy_hash'
truestamp entropy list --count --limit 1
#   Entropy Observations (1 shown, 593,419 total)

# Look up by UUIDv7 id, or by the 64-hex entropy hash
truestamp entropy get 01a07335-b8fe-7ef2-856c-c9b4eca99850
truestamp entropy get 4142459a2859a57a5e5d6540579b977215d6329110adc0a890fdce6f05054f2d --json
```

The card shows the observation's identity, its state (`created` until a
block commits it, then `committed`, with the block's id as a `Block` row),
the `Entropy Hash`, when the source published the value and when
Truestamp captured it, then the source's own record under the source's
own field names: a Stellar ledger's `sequence`, `closed_at`, `hash` and
`paging_token`; a NIST pulse as `pulse.pulseIndex`, `pulse.outputValue`
and so on; a Bitcoin block's fields likewise. Nothing is renamed. The
`HASH` column of the list is shortened; the full value is on the card and
in `--json`, and `entropy get` accepts it whole.

The entropy hash is the value an item's metadata commits to under
`subject.metadata.witnesses`, which makes `get` by hash the way to trace a
witness named in a proof back to the observation it came from, and each
observation is a proof subject in its own right:

```sh
# From an item proof to the Stellar observation it witnessed
truestamp entropy get "$(jq -r .subject.metadata.witnesses.entropy_stellar proof.json)"

# From an observation to its own proof, verified
truestamp proofs get 01a07335-b8fe-7ef2-856c-c9b4eca99850 | truestamp verify
#   [PASS]  Entropy Source       External consistency verified via Stellar Horizon API
#   VERDICT: PASSED
```

`--source` outside the three wire names is refused before any request
(`--source must be one of entropy_nist | entropy_stellar | entropy_bitcoin,
got "nist"`), a malformed id is refused locally, and a well-formed unknown
one is a server answer: `entropy observation not found`, exit `1`. All
three subcommands carry `--json` and `--silent` / `-s`.

All entropy subcommands require authentication, like `beacons`: run
`truestamp auth login`, or set `TRUESTAMP_API_KEY` / `--api-key` for
headless and CI use.

---

## `truestamp keys`

Read-only access to the published signing keyring. This is the only
API-backed group that needs no credential: the keyring is served
unauthenticated from `/.well-known/keyring.json`, and it is the same
document `verify` consults for the key binding step.

```sh
truestamp keys list
truestamp keys current                 # the key signing right now
truestamp keys get 96b1cd2f            # by its 8-hex key id
truestamp keys current --json | jq -r .public_key
# sYfRKf1R/ILtfRv1KjBgk6F6kgIrXDfVbz3Pl+kgE2U=

truestamp keys get 3c19f776
# no published key with id "3c19f776"      (exit 1)
```

`current`, not `latest`: the keyring is populated by chain replay
including prerotation events, so a published-but-not-yet-active key would
make the two diverge.

The keyring answers the one question a proof bundle cannot answer about
itself — *is this key Truestamp's* — because a forged bundle is
self-consistent by construction and carries its forger's own public key.
It does not tell you a key was uncompromised, valid at signing time, or
authorized; it carries no revocation flag, no validity interval and no
timestamps.

```sh
# Derive a key id from a public key you already hold, and look it up
truestamp keys get $(truestamp convert keyid sYfRKf1R/ILtfRv1KjBgk6F6kgIrXDfVbz3Pl+kgE2U=)
```

## `truestamp schema`

Machine-readable descriptions of the CLI's own interface, generated by
walking the live command tree and the in-code registries, so they cannot
drift from the binary that prints them. Every document renders as text by
default and as JSON with `--json`, so pipe through `--json` before `jq`.

```sh
truestamp schema list
#   Schema documents
#
#      algorithms  Hash algorithms accepted by `truestamp hash -a`
#        commands  The full command tree: every path, flag, type, default and enum
#      exit-codes  What each process exit status means
#   subject-types  The frozen proof subject type registry (name to wire code)
#       witnesses  Witness names accepted by `--witnesses`, and which are committed

truestamp schema get commands --json | jq -r '.subcommands[].path'
truestamp schema get algorithms --json | jq -r '.algorithms[].name'
truestamp schema get subject-types
truestamp schema get exit-codes
truestamp schema get witnesses
```

`schema get commands --json` is the one call that answers "what can this
CLI do, and with which flags" without parsing help text. It carries each
flag's name, usage, type, default, shorthand, whether it is inherited,
and its closed value set where the flag has one.

## `truestamp upgrade`

Self-upgrade the binary. The command is **install-method aware**. Homebrew
users are shown `brew upgrade`, `go install` users are shown the
`go install ...@latest` incantation, and `install.sh` users get a native-Go
in-place upgrade that mirrors the install script (SHA-256 mandatory, cosign
best-effort, atomic rename with timestamped backup).

```sh
# Is a newer release available? Prints the answer and exits 0 either way.
truestamp upgrade --check
# truestamp is up to date (0.13.0-...)

# The same check with the result in the exit status: 0 up to date,
# 1 upgrade available, 2 network error, 3 latest release is a pre-release
truestamp upgrade --check --exit-code

# Perform the upgrade (install.sh-installed binaries only)
truestamp upgrade

# Skip the confirmation prompt
truestamp upgrade --yes

# Pin to a specific version (the opt-in path for pre-releases)
truestamp upgrade --version v0.13.0
```

`truestamp version` reports the detected install method, which is what
decides between the "print instructions" and "replace in place" paths.

---

## `truestamp version`

```sh
# Detailed build info, one label per line: version, module path, config
# path, detected install method, Go toolchain + OS/arch, commit, build date
truestamp version

# The same fields as JSON
truestamp version --json

# One-liner: version plus the short commit and the build date
truestamp --version
# truestamp <version> (commit: <short-sha>, built: <rfc3339>)
```

---

## Pipeline recipes

Real-world compositions that solve a specific problem.

Where a recipe below shows sample output, it was captured from one real
complete item bundle, a claims-as-source-of-truth proof. Two blocks are
drawn from elsewhere and say so where they appear: the `claims_hash`
worked example and the Hash Comparison outcome table both use the
whitepaper's Appendix D bundle, which is the one that carries a file hash.
Your own `proof.json` will carry different values but the same shape.

### Reproduce the protocol's `claims_hash` intermediate

The Truestamp protocol computes an internal intermediate,
`claims_hash = SHA256(0x11 || JCS(claims))`, while deriving the item_hash
that gets signed into the block. The value itself is **not serialized** into
the proof bundle: the proof stores `subject.claims`, the raw claims, and
`subject.metadata`, the witness map whose `0x12` digest is a different
intermediate. A verifier recomputes both.

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

Both forms produce the same digest. A self-contained check you can paste
verbatim, using the claims of the whitepaper's Appendix D worked example
(note that the keys are deliberately out of order here: JCS sorts them, so
the result is unaffected):

```sh
echo '{"name":"Appendix D worked example","description":"Illustrative item for the Truestamp whitepaper. Not a production record.","hash":"b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380","hash_type":"sha256"}' \
  | truestamp hash --prefix 0x11 --jcs -a sha256 --style bare
# ae5cdc73a52359a4fb0e335f004a6cd0cb1247024e812263e362368915a4b924

# The canonical bytes that digest is taken over, if you want to see them:
echo '{"name":"Appendix D worked example","description":"Illustrative item for the Truestamp whitepaper. Not a production record.","hash":"b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380","hash_type":"sha256"}' \
  | truestamp jcs
# {"description":"Illustrative item for the Truestamp whitepaper. Not a production record.","hash":"b47cc0f1...","hash_type":"sha256","name":"Appendix D worked example"}
```

`ae5cdc73...` is the `claims_hash 0x11` value the whitepaper publishes for
that bundle, so a matching digest confirms your `jcs` + `hash` composition
agrees with the protocol.

**To confirm that a file matches what was timestamped**, you don't need this
intermediate. Compare the file's plain SHA-256 against
`subject.claims.hash`, or pass `--expected-hash <sha256-hex>` to
`truestamp verify` (see the next two recipes).

### Convert a JSON proof to CBOR and verify

```sh
truestamp proofs convert --to cbor proof.json \
  | truestamp verify --offline
```

`proofs convert` prefixes its CBOR output with the self-describing tag
55799, and `verify` detects the format from that tag, so the pipeline needs
no format flag on the `verify` side (`verify` has no `--from`; that flag
belongs to `proofs convert`). The report header names the input as
`Bundle: (stdin) (cbor, <N> bytes)`. Exit code 0 when the verdict is
PASSED, 1 when it is FAILED.

### Derive a kid from a proof's embedded public key, without `jq`

Canonicalize first so the `"public_key"` field is guaranteed to be compact,
then slice it out with plain `grep` / `cut` (the first match is the
bundle's own key; a signing key event carries the same value again):

```sh
truestamp jcs < proof.json \
  | grep -o '"public_key":"[^"]*"' | head -1 | cut -d'"' -f4 \
  | truestamp convert keyid
# 8 lowercase hex characters, e.g. 3c19f776
```

JCS sorts the top-level keys alphabetically, which puts `public_key` ahead
of `signing_key_event`, so `head -1` reliably selects the bundle's own key.

Skipping the `jcs` step only works on an already-minified bundle; a
pretty-printed proof writes `"public_key": "..."` with a space, which the
pattern above would miss. It fails loudly rather than silently, because
`convert keyid` refuses an empty input:

```sh
grep -o '"public_key":"[^"]*"' pretty-proof.json | head -1 | cut -d'"' -f4 \
  | truestamp convert keyid
# empty stdin input
# exit 1
```

With `jq` (cleaner):

```sh
jq -r .public_key proof.json | truestamp convert keyid
```

Both spellings print the same kid.

### Extract the block commit time from a proof (in your local zone)

```sh
jq -r .block.id proof.json | truestamp convert id --to-zone Local
# 2026-09-03T12:16:00.182-04:00
```

### Confirm a downloaded file matches what you timestamped

```sh
# Hash the local file, compare against the hash the proof commits to.
# Prints "match" or "MISMATCH".
expected="$(jq -r .subject.claims.hash proof.json)"
actual="$(truestamp hash -a sha256 --style bare contract.pdf)"
[ "$expected" = "$actual" ] && echo "match" || echo "MISMATCH"
```

This recipe is **external-hash mode only**. For
claims-as-source-of-truth proofs there's no separate file to
compare, because the claims content itself is what the proof commits to.
You can spot a claims-only proof at a glance with
`jq '.subject.claims | has("hash") | not' proof.json` (`true` means
claims-only); the verify report's Hash Comparison row reads
`[SKIP]  Hash Comparison      no file hash in this proof` for it.

### Verify a proof while passing the expected hash inline

```sh
truestamp verify proof.json --expected-hash "$(truestamp hash -a sha256 --style bare contract.pdf)"
```

Three outcomes, all driven by the same Data Integrity row. The first two
are from the whitepaper's Appendix D bundle, the only fixture here that
carries a file hash; the third is from the claims-only item bundle. Rows
are wrapped for width, and the `b47cc0f1...` digest is elided:

```text
# match     [PASS]  Hash Comparison      supplied file hash matches
#                                        subject.claims.hash
#                                        summary: "file hash matched: yes"
#                                        VERDICT: PASSED, exit 0
# mismatch  [FAIL]  Hash Comparison      supplied file hash does NOT match
#                                        subject.claims.hash (proof commits
#                                        to b47cc0f1...)
#                                        VERDICT: FAILED, exit 1
# claims-only proof
#           [WARN]  Hash Comparison      an expected hash was supplied but
#                                        this proof commits to no file hash
#                                        VERDICT: PASSED, exit 0
```

A supplied hash that does not match is a hard failure; supplying one to a
claims-as-source-of-truth proof only warns, because there is nothing for it
to be compared against.

### Batch-verify every proof in a directory, silently

```sh
find proofs -name '*.json' -print0 \
  | xargs -0 -I{} sh -c 'truestamp verify --silent --offline "{}" || echo "FAIL: {}"'
# FAIL: proofs/tamper-claims.json
```

`--silent` suppresses the report entirely, so the only output is one line
per proof that failed.

### Hash a file with every supported algorithm in one pass

```sh
for alg in $(truestamp hash --list | awk '{print $1}'); do
  truestamp hash -a "$alg" --style bsd contract.pdf
done
# MD5 (contract.pdf) = <hex>
# SHA1 (contract.pdf) = <hex>
# SHA224 (contract.pdf) = <hex>
# ... 14 lines, one per algorithm, MD5 first and BLAKE2b last ...
# BLAKE2b (contract.pdf) = <hex>
```

`--list` prints one algorithm per line with no header, so `awk '{print $1}'`
yields exactly the 14 canonical names, in registry order: `md5`, `sha1`,
`sha224`, `sha256`, `sha384`, `sha512`, `sha3-224`, `sha3-256`, `sha3-384`,
`sha3-512`, `blake2s-256`, `blake2b-256`, `blake2b-384`, `blake2b-512`.
`md5` and `sha1` each print a "cryptographically broken" warning to
**stderr** first; redirect with `2>/dev/null` if the loop's output is being
parsed.

### Round-trip test a proof you just received

```sh
# JSON to CBOR to JSON, then compare against the original (modulo
# canonical key order)
orig="$(truestamp jcs < proof.json)"
round="$(truestamp proofs convert --to cbor proof.json \
        | truestamp proofs convert --from cbor --to json \
        | truestamp jcs)"
[ "$orig" = "$round" ] && echo "round-trip stable" || echo "DRIFT"
# round-trip stable
```

### Convert every commitment timestamp to your local zone

```sh
jq -r '.commitments[].timestamp' proof.json \
  | while read -r ts; do
      truestamp convert time "$ts" --to-zone Local
    done
# 2026-09-03T12:20:05-04:00
```

### Show the embedded item time and block time side-by-side

```sh
printf "item:  "; jq -r .subject.id proof.json | truestamp convert id
printf "block: "; jq -r .block.id proof.json | truestamp convert id
# item:  2026-09-03T16:15:14.222Z
# block: 2026-09-03T16:16:00.182Z
```

The gap between the two is how long the item waited before being committed
into a block; both are submission-side times, and neither says anything
about when the underlying content was created.

---

## Scripting with `--json` and `jq`

Every inspection command supports `--json`. Combined with `jq`, you can
build sophisticated pipelines with no parsing glue code.

```sh
# The verdict: true when no step failed
truestamp verify proof.json --json | jq -r .passed
# true

# Compute a digest and pipe it into another command's --expected-hash flag.
# --expected-hash is compared against .subject.claims.hash, so it only says
# anything on an external-hash bundle - hence a second placeholder here.
expected="$(truestamp hash -a sha256 --json contract.pdf | jq -r .digest.hex)"
truestamp verify external-proof.json --expected-hash "$expected"

# Whether an expected hash was supplied, and whether it matched. E.7 keeps
# the two apart so "not provided" can never read as "mismatch". The output
# below is the matching case, which assumes contract.pdf really is the file
# external-proof.json commits to; substitute your own pair.
truestamp verify external-proof.json --json --expected-hash "$expected" \
  | jq '{expected_hash_provided, hash_matched}'
# {
#   "expected_hash_provided": true,
#   "hash_matched": true
# }

# Every failing row, if any. Shown against a deliberately corrupted bundle,
# since the passing proof.json above emits no fail rows at all.
truestamp verify tampered.json --json --offline | jq -r '.steps[] | select(.status=="fail") | "\(.group): \(.message)"'
# Inclusion Proof: derived root does not match block.merkle_root (derived fd796ea4...)
# Proof Signature: proof signature invalid (Ed25519)

# The submission window the run established (server field names). The
# *_commit keys are present only for the chains the bundle commits to.
truestamp verify proof.json --json | jq .temporal
# {
#   "submitted_at": "2026-09-03T18:05:26Z",
#   "committed_at": "2026-09-03T18:06:00Z",
#   "stellar_commit": "2026-09-03T18:10:02Z",
#   "bitcoin_commit": "2026-09-03T18:15:06Z"
# }

# Branch on submission mode at the jq layer
jq 'if .subject.claims | has("hash") then "external-hash mode" else "claims-only mode" end' proof.json
# "claims-only mode"

# Tag-style report of every commitment's chain, network, and timestamp
jq -r '.commitments[] | "\(.chain) (\(.network)) committed at \(.timestamp)"' proof.json
# bitcoin (regtest) committed at 2026-09-03T18:15:06Z
# stellar (testnet) committed at 2026-09-03T18:10:02Z

# The witnesses an item commits to, and which details this bundle carries
jq '.subject.metadata.witnesses' proof.json
# {
#   "block": "b2605d453ac09843...",
#   "entropy_bitcoin": "d715b11fc28f43d3...",
#   "entropy_nist": "ea09f883c7970891...",
#   "entropy_stellar": "51e1ab3f18179564..."
# }
truestamp inspect proof.json --json | jq '.subject.carried_witnesses'
# [
#   "block",
#   "entropy_bitcoin",
#   "entropy_nist",
#   "entropy_stellar"
# ]

# A refused bundle carries a rejection object and an empty steps array
truestamp verify draft.json --json | jq -r '.rejection.code // "accepted"'
# unsupported_layout
```

---

## CI / scripting conventions

For automated pipelines, use the following conventions:

```sh
# Silent verification: exit code 0 = pass, 1 = fail or rejected
if truestamp verify --silent --offline proof.json; then
  echo "valid"
else
  echo "invalid"
  exit 1
fi

# JSON-structured output for parsers
truestamp verify proof.json --json > verify-report.json
jq '.passed' verify-report.json

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
| `TRUESTAMP_API_KEY` | Auth token for `items` / `proofs get` / `blocks` / `beacons` / `teams` / `console` / `auth status` / `verify --remote`. `keys` and `schema` need no credential. |
| `TRUESTAMP_TEAM` | Multi-tenant team ID |
| `TRUESTAMP_HTTP_TIMEOUT` | HTTP timeout (`30s`, `1m`) |
| `TRUESTAMP_COSIGN_PATH` | Absolute path to the `cosign` binary used by `truestamp upgrade` (empty = `$PATH` lookup) |
| `TRUESTAMP_REQUIRE_COSIGN` | `1` makes a missing `cosign` binary or signature bundle a hard error during an in-place `truestamp upgrade` instead of a warning (SHA-256 is always enforced) |
| `TRUESTAMP_LOGGING_LEVEL` / `TRUESTAMP_LOGGING_FILE` | Log level and log file path (named after the `[logging]` config section, not the flag) |
| `TRUESTAMP_LOGGING_MAX_SIZE_MB` / `TRUESTAMP_LOGGING_MAX_BACKUPS` / `TRUESTAMP_LOGGING_MAX_AGE_DAYS` | Log rotation budget |
| `TRUESTAMP_SILENT` / `TRUESTAMP_JSON` | CLI-wide defaults for `--silent` / `--json` (setting both is an error) |
| `TRUESTAMP_VERIFY_OFFLINE` / `TRUESTAMP_VERIFY_SKIP_SIGNATURES` / `TRUESTAMP_VERIFY_REMOTE` | Defaults for the corresponding `verify` flags |
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
(`items`, `proofs get`, `auth`, `blocks`, `beacons`, `teams`, `console`,
`verify --remote`) works without network. `truestamp keys` is the one
network-bound group that needs no credential: it reads the published
keyring at `{base-url}/.well-known/keyring.json`. `truestamp schema` is
served out of the binary and is unconditionally local.

One command reaches a *different* network service: `truestamp upgrade
--check` always queries the GitHub Releases API (with `--exit-code`, exit 2
on network error), as does the in-place upgrade path; on a Homebrew or `go
install` binary plain `truestamp upgrade` only prints instructions and stays
offline.

```sh
# Fully offline verification: no calls to Truestamp, Stellar, Bitcoin or
# the entropy sources. Pin a keyring copied earlier so the key binding
# check runs too.
truestamp verify proof.json --offline --keyring keyring.json
# [PASS]  Key Binding          key_id 96b1cd2f found in the pinned keyring (sequence 0, active true)
# 23 passed   0 failed   0 warned   8 skipped   12 info
# VERDICT: PASSED

# The convert / hash / encode / decode / jcs / inspect primitives compute
# locally and make no network calls - unless you explicitly pass --url,
# which downloads the input (hash, encode, decode, jcs, inspect and
# proofs convert accept it; convert time / id / keyid / merkle have no
# --url and are unconditionally offline).
truestamp hash -a sha256 contract.pdf
truestamp jcs < claims.json
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD
# 2026-04-07T23:05:39.493Z
truestamp proofs convert --to cbor proof.json
truestamp inspect proof.json
```

Offline operation is a first-class mode of the whitepaper's verifier
(Appendix E.2): every step from the structural gates through the signature,
the witnesses and the submission window runs with zero network access. What
an offline run cannot do is tie the proof to a clock outside Truestamp: the
Stellar and Bitcoin commitments, each entropy witness's source, the key
event's own commitments and the keyring (unless `--keyring` pins a local
copy) are each reported as skipped, and the two edges of the submission
window stay informational rather than passing. A skipped check is a check
the run did not perform, never a check that failed, and the verdict is
PASSED when nothing failed.

`truestamp verify` without `--offline` performs five classes of outbound
requests:

1. Fetches the Truestamp keyring at `{base-url}/.well-known/keyring.json` (default `https://www.truestamp.com/...`) to bind the signing key, unless `--keyring <file>` pins a local copy.
2. If a Stellar commitment is present (in the bundle or in the signing key event), hits the Horizon API (`horizon.stellar.org` for `network: public`, otherwise `horizon-testnet.stellar.org`).
3. If a Bitcoin commitment is present on mainnet or testnet, hits the Blockstream API (`blockstream.info`) to bind the header recomputed from the bundle's own bytes. Regtest has no public API, so that case is local-only and stays unconfirmed.
4. For an entropy subject, and for every entropy witness an item bundle carries, hits the upstream publisher the observation came from, to compare the captured value: the **NIST Randomness Beacon** (`beacon.nist.gov`) for `entropy_nist`, Horizon for `entropy_stellar`, Blockstream mainnet for `entropy_bitcoin` (Bitcoin entropy is always captured from mainnet, even by deployments that commit to testnet or regtest).
5. With `--remote`, posts the bundle to the Truestamp API's `/proof/verify`.

`--offline` skips the first four. `--skip-signatures` skips the Ed25519
**Proof Signature** check, and the **Key Binding** cross-check with it
unless `--keyring` pins a local copy; `README.md` §What gets verified
spells out exactly what each flag skips and what still runs. Everything
else local (subject hash, Merkle inclusion, block hash, epoch proofs,
witnesses, submission window) is always performed.

---

**See also:**
`README.md` for install instructions, `kb/` for architecture and format
reference (start at `kb/README.md`),
`./build/truestamp <command> --help` for per-command flag documentation.
