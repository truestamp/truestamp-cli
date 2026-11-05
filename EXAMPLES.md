<!--
Copyright (c) 2019-2026 Truestamp, Inc.
SPDX-License-Identifier: Apache-2.0
-->

# Truestamp CLI - Examples

A hands-on tour of every sub-command, followed by real-world pipeline recipes.
Every example is copy-pastable once you substitute your own paths and ids.

Every output shown here was captured from the binary, whole: no hash, id or
key is shortened, and where an output is long the example pipes it through
`head` rather than cutting it. Three things are worth knowing about how they
were captured:

- The proofs are the conformance vectors of
  [truestamp/truestamp-verification](https://github.com/truestamp/truestamp-verification):
  `proof.json` is its `vectors/positive/item.proof.json`, an item proof from
  Truestamp's **test** environment, and `keyring.test.json` is its
  `vectors/keyring.test.json`. Those proofs are checked offline against that
  keyring (`--offline --environment test --keyring keyring.test.json`), so a
  sound one ends in the label `supplied-keyring`. A production proof, written
  `contract.proof.json` here, is checked with neither `--environment` nor
  `--keyring`, and only an online run can end in `verified`; where that
  matters the text says what an online run adds.
- The failing proofs are the vectors' negative proofs, each signed as it
  stands so that the check it targets is the one that fails:
  `tampered.json` and `proofs/tamper-claims.json` are
  `vectors/negative/declared-claims_hash.proof.json`, and
  `claims-only.proof.json` is
  `vectors/negative/expected_hash-no_hash.proof.json`. A proof whose
  content you edit by hand fails at SIGNATURE, since its signature no
  longer covers what it says. Some edits fail sooner: a changed or removed
  environment marker at ENVIRONMENT, a changed `signature.public_key` at
  KEY_BINDING, and an added envelope member as the bare rejection
  `invalid_envelope`.
- The commands that talk to the Truestamp API (`items`, `proofs get`,
  `blocks`, `beacons`, `entropy`, `keys`, `teams`, `auth status`) were run
  against a local stand-in that answers as the service's routes do, serving
  the same vectors' records. The links they print are shown on the default
  origin, `https://www.truestamp.com`, and a home directory as `/Users/you`.
- Table rows are shown without the trailing spaces that pad them to a column
  width.

Run `truestamp <command> --help` at any time for exhaustive flag documentation,
or `truestamp schema get commands --json` for the entire command tree (every
path, flag, type, default and enum) as machine-readable JSON.

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
- [`truestamp convert id`](#truestamp-convert-id)
- [`truestamp convert keyid`](#truestamp-convert-keyid)
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

`verify`, `inspect`, `encode`, `decode` and `jcs` share one input resolver and
accept the same **six modes**, plus the Unix `-` alias for stdin:

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
  `--url` must use the `=` form there, as shown below.
- `convert time`, `convert id` and `convert keyid` take their value
  positionally or on stdin only. They have no `--file` / `--url`, and `-`
  is read as a value rather than as stdin.
- `items create` has its own documented priority order
  (`truestamp items create --help`).

The two interactive modes need a terminal. Under a pipe or redirect they fail
fast with a named reason rather than hanging. (`verify` is shown with
`--offline` here: an online run reads its keyring and checks its chain
sources first, before it reads any input.)

```sh
truestamp verify --offline --file < /dev/null
# the proof cannot be read: --file without a path: interactive prompt requires a terminal (stdin is piped or redirected)
# exit 1

truestamp verify --offline --url < /dev/null
# the proof cannot be read: --url without a URL: interactive prompt requires a terminal (stdin is piped or redirected)
# exit 1
```

### `--file` / `--claims` / `--url` take an *optional* value

These flags are declared with an optional argument so that `--file` alone can
open the interactive picker. That has a consequence worth internalizing: a
space-separated value is parsed as a **positional argument**, not as the flag's
value.

`verify`, `inspect`, `encode`, `decode` and `jcs` all fall back to the
positional argument when `--file` / `--url` were given with no value, so both
spellings work there:

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

The refusal names the fix:

```text
--claims takes its path with an equals sign: did you mean --claims=claims.json ? (a space-separated path is read as a file to hash, not as claims)
--file takes its path with an equals sign: did you mean --file=doc.pdf ? (or drop the flag: `truestamp items create doc.pdf` hashes it directly)
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
#   "base_url": "https://www.truestamp.com",
#   "api_url": "https://www.truestamp.com/api/json",
#   "keyring_url": "https://www.truestamp.com/.well-known/keyring.json",
```

Widely-available (per-command where meaningful) flags you'll see repeatedly:

- `--json` for machine-readable output, and `-s` / `--silent` for
  exit-code-only runs. They are mutually exclusive, and every command that
  renders a *record* carries both.
- Both are also CLI-wide *settings*: put `json = true` or `silent = true` at the
  top level of `config.toml`, or set `TRUESTAMP_JSON` / `TRUESTAMP_SILENT`, and
  record-rendering commands pick it up without the flag. The pipeline
  primitives (`encode`, `decode`, `jcs`, `convert time|id|keyid`) and
  `hash` deliberately ignore the ambient setting: wrapping their output would
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
that stdout stays a single line: the labelled path goes to stdout and whether
that file exists goes to stderr. The first line below is stdout and the
second is stderr, so stdout carries the label as well as the path, and a
`$()` capture is one line but not a bare path:

```sh
NO_COLOR=1 truestamp config path
# Config Path  /Users/you/.config/truestamp/config.toml
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
panic (matching Go's own convention). Specialized commands add codes:
`verify` exits with the codes of SPEC.md section 12.2, 0 through 8, and
`upgrade --check --exit-code` uses 0 through 3. The authoritative table is
generated from the constants that produce it:

```sh
truestamp schema get exit-codes
#   Exit codes of every other command
#
#   CODE    MEANING
#   0       the command did what it was asked
#   1       it did not: a network failure, invalid input, or any other runtime error
#   2       an unrecovered panic, matching Go's own convention
#
#   verify (SPEC.md section 12.2)
#
#   CODE    MEANING
#   0       label verified
#   1       a runtime error outside verification, such as an invalid command line or an input that cannot be read; no report
#   2       the verifier stopped on a fault it did not catch, or caught one before steps 0 to 3 passed; no report
#   3       label failed
#   4       a bare rejection other than configuration_error
#   5       label unattributed
#   6       label supplied-keyring
#   7       label internally-consistent
#   8       configuration_error
#
#   upgrade --check --exit-code
#
#   CODE    MEANING
#   0       up to date
#   1       an upgrade is available
#   2       a network error prevented the check
#   3       the latest release is a pre-release and will not auto-install
#
#     exit_codes are every command's but verify's and upgrade's under --check --exit-code; verify exits only with verify_exit_codes (SPEC.md section 12.2), and upgrade under --check --exit-code only with upgrade_check_exit_codes.
```

### Paging lists

`items list`, `blocks list`, `beacons list` and `entropy list` page the same
way, so learn it once:

| Flag | What it does |
| --- | --- |
| `--limit N` | Page size (default 25; a page above 250 is clamped to 250, and the listing says so) |
| `--after <cursor>` | Continue forward from the cursor a page printed |
| `--before <cursor>` | Continue backward from the cursor a page printed |
| `--oldest-first` | Start at the beginning (genesis, the first observation, the oldest item) instead of the newest row |
| `--max N` | Follow cursors in the chosen direction until N rows have been fetched |
| `--count` | Add the server's total to the heading (`(1 shown, 3 total)`) and `"total"` to `--json` |

A text page that continues ends with `More: --after <cursor>` and, when it
can go back, `Back: --before <cursor>`. Both go to **stderr**, and only when
stdout is a terminal: they are guidance about the listing, not the listing, so
`truestamp items list | wc -l` counts rows and `... > f` writes a file the next
command can read. `--json` is one envelope,
`{"<noun>": [...], "next_cursor": "...", "prev_cursor": "..."}`. There is no
`--all`: the tables behind these lists grow by the minute, so following pages
costs a cap you wrote down, and `--max` sizes its last request to the rows
still wanted so the cursor it hands back continues exactly.

```sh
truestamp blocks list --limit 2                  # the two highest, plus a More: cursor
truestamp blocks list --limit 2 --after <cursor> # the next two down, plus More: and Back:
truestamp blocks list --oldest-first --limit 2   # genesis first
truestamp entropy list --max 500 --json | jq -r '.observations[].id'
truestamp items list --count --limit 1           # Items (1 shown, 3 total)
```

Every collection clamps a page above 250 rows to 250 rather than refusing
it. The listing says so when a page follows, and `--max` follows the cursor
past it; `--json` carries `"page_limit": 250` whenever the server used a
smaller page than asked for, even when nothing follows (a short collection
fetched with `--limit 500` still reports it), so read `next_cursor`, not
`page_limit`, to learn whether there is more.

---

## `truestamp auth`

**Start here.** `items`, `proofs`, `blocks`, `beacons`, `entropy`, `teams`,
`console` and `verify --remote` all require credentials. Everything else in
this document, `hash`, `encode`, `decode`, `jcs`, `verify` (local),
`inspect`, `schema` and every `convert` subcommand, works without
authentication and without a network. `keys` sits in between: it needs the
network to fetch the public keyring, but no credential. If you only need
local verification and inspection, you can skip this section.

Authentication is **OAuth-first, API-key-second**:

```sh
# Default: opens your browser for an OAuth 2.1 Authorization Code + PKCE
# flow. The resulting access + refresh token pair is stored in your OS
# keychain (0600-file fallback) and refreshed automatically, never in
# config.toml.
truestamp auth login

# Headless / CI alternative: paste a long-lived API key, stored in
# config.toml at 0600. It needs a terminal to type into.
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
#   "config_file": "/Users/you/.config/truestamp/config.toml",
#   "api_url": "https://www.truestamp.com/api/json",
#   "auth_mode": "API key",
#   "api_key": "ts_s...0000",
#   "user_id": "01a0db60-0000-7000-8000-0000000000aa",
#   "email": "you@example.com",
#   "full_name": "Ada Lovelace"
# }

# A credential the API rejects: exit 1, plus a stable `reason` identifier
# (never a sentence) and a human-readable `message`. The reason vocabulary
# is not_authenticated, api_unreachable, credential_rejected,
# unexpected_api_response, team_lookup_failed, team_not_accessible.
truestamp auth status --json --api-key not-a-real-key
# {
#   "ok": false,
#   "reason": "credential_rejected",
#   "message": "Authentication required. Provide an API key as 'Authorization: Bearer \u003capi_key\u003e'. See https://www.truestamp.com/llms-full.txt",
#   "config_file": "/Users/you/.config/truestamp/config.toml",
#   "api_url": "https://www.truestamp.com/api/json",
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
4. Nothing. Public reads still work (`keys current`, `schema`), but a command that needs a credential refuses locally rather than eliciting a 401: `blocks latest` and `teams list` print `Not authenticated`, and `verify --remote` refuses before it reads the proof, so the proof is never sent.

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
whether the credential is still good. With an API key in the environment,
`auth logout` has nothing stored to clear and says so:

```sh
truestamp auth logout
#   Authenticated via TRUESTAMP_API_KEY/--api-key, nothing is stored to clear.
#   Unset it in your environment to sign out.
```

## `truestamp config`

Inspect and manage the resolved configuration. Settings are merged from
compiled defaults, then the config file, then `TRUESTAMP_*` env vars, then CLI
flags, with later sources overriding earlier ones.

```sh
# Where is the config file, and does it exist yet?
truestamp config path
# Config Path  /Users/you/.config/truestamp/config.toml
#              exists

# Before `config init`, the second line reads "does not exist, run
# 'truestamp config init' to create it".

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
# than launching vi against a pipe and blocking forever:
# "config edit needs a terminal; edit <path> directly instead".
truestamp config edit

# Override any setting for a single invocation via an env var
TRUESTAMP_BASE_URL=https://www.truestamp.com truestamp config show

# Or via a flag
truestamp config show --base-url https://www.truestamp.com
```

`config show` renders a General block (Config File, API URL, Auth Mode, API
Key, Team, Keyring URL, HTTP Timeout, Cosign Path, Silent, JSON) followed by
the Verification, Hash and Convert sections. Verification shows Offline and
Remote, the verifier's Horizon, Esplora and Bitcoin Core sources, each
network's several sources with its policy, and the keyring configured for
each non-production environment; a Bitcoin Core node's user and password
are never shown, and everything before a source URL's last `@`, but its
scheme, is shown as `[masked]`, since any `@` could end user information.
Two further rows, **Team Name** and **Team Role**, appear directly under
Team only when a team id *and* a usable credential are both present:

```sh
truestamp config show --team 01a0db60-0000-7000-8000-000000000001 | grep -E '^ +Team'
#              Team  01a0db60-0000-7000-8000-000000000001
#         Team Name  Personal (personal)
#         Team Role  Owner
```

That lookup is best-effort, so `config show` stays useful offline. It is not
dropped silently: when the API is unreachable, Team Name reads
`(unavailable, try 'truestamp auth status')`, Team Role is omitted, and the
command still exits 0. Config File is the file actually in effect, so it
reflects `--config` when one was supplied.

`--json` shows values the styled table does not, `base_url`,
`websocket_url`, `health_url` and a `logging` block among them, and nests
`verify`, `hash`, `convert` and `logging` as objects
(`truestamp config show --json | jq -r .base_url`); `logging` carries the
level and file and the three rotation keys. It is not a strict superset of
the table: the best-effort **Team Name** and **Team Role** rows are
table-only, so the JSON reports a team as the bare `team` id and nothing
more. As everywhere else, `--json` and `-s` / `--silent` are mutually
exclusive; passing both is an error and exits 1.

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
# prints an "Item Created" card: ID, Name, Hash, Visibility, Team, the
# claims in canonical form and their claims hash, the item hash and its
# item code, a Details link, whose page needs sign-in, and the public
# Verify link

# 2. Later, after the item's block is committed to a public chain, download
# its proof by the item code step 1 printed (or its UUIDv7 id, or its
# item_hash)
truestamp proofs get <CODE> -o contract.proof.json

# 3. Verify the proof end-to-end
truestamp verify contract.proof.json
# checks the envelope and the signer's key against Truestamp's keyring, the
# signature, every recomputed hash, inclusion proof and link, then reads each
# Stellar and Bitcoin commitment and each entropy source to draw the
# submission window; exits 0 when the label is verified
```

---

## `truestamp items create`

Submit a new timestamp item. Requires authentication: run
`truestamp auth login`, or set `TRUESTAMP_API_KEY` / `--api-key` for
headless and CI use.

Truestamp supports two submission modes. Both produce proofs of the same
shape; the only difference in the claims is whether `hash` / `hash_type`
are present.

* **External-hash mode** for files you keep on your own device.
  The file's SHA-256 is submitted; the file itself never leaves your
  machine.
* **Claims-as-source-of-truth mode** for things that don't have a
  file (statements, invention disclosures, dated facts, release
  notes). The claims content is what gets timestamped, so there's
  nothing to preserve alongside the proof.

The claims are sent exactly as given, and the server commits their
canonical form (`claims_jcs`, RFC 8785). A claims file (`--claims=`) or
claims on stdin (`-C`) goes byte for byte: the CLI reads it only to refuse a
text no verifier reads alike (a member name repeated in one object, or a
number outside the range every verifier reads exactly), and trims,
reorders and fills in nothing. The claim flags (`--name`, `--data-hash`,
`--hash-type`, `--description`, `--url`, `--timestamp`, `--metadata`,
`--location`) build the claims instead, so they are refused beside a claims
file.

`hash` and `hash_type` go together: both present (external-hash mode) or
both absent (claims-as-source-of-truth mode). The flags that set them are
`--data-hash` and `--hash-type` (there is no `--hash` flag on `create`), and
neither is filled in for the other: a bare `--data-hash` sends no
`hash_type`, and the server refuses the half pair, as it refuses a hash that
is not hex of the length its algorithm gives. Beside a hashed file both are
refused, since the file supplies the hash and its `hash_type`.

Remember that `--claims` and `--file` need the `=` form here; see
[Conventions](#conventions).

### External-hash mode

```sh
# Hash a file and submit in one step (filename becomes the item name,
# SHA-256 becomes the hash)
truestamp items create contract.pdf
#   Item Created
#            ID  01a17d3b-59a0-7a37-badf-6bb603cf6ac6
#          Name  contract.pdf
#          Hash  3494d2947b7929676faeb13ff604d686ad200a2e577130856cc4b63b1324350f (sha256)
#    Visibility  private
#          Team  01a0db60-0000-7000-8000-000000000001
#    Claims JCS  {"hash":"3494d2947b7929676faeb13ff604d686ad200a2e577130856cc4b63b1324350f","hash_type":"sha256","name":"contract.pdf"}
#   Claims hash  9967655a7ccc860d040e50dd80eb2bda6f81032ac39ad18cba284d63dfdc2f63
#     Item hash  fc05138c6e53e7f5f16a4ea832438a082d9be2b0907e3eb1d20813a27ad62e96
#     Item code  ZG2H733EAFKZBWB0
#       Details  https://www.truestamp.com/items/ZG2H733EAFKZBWB0
#        Verify  https://www.truestamp.com/verify/item/ZG2H733EAFKZBWB0

# Same, machine-readable output
truestamp items create contract.pdf --json
# {
#   "claims_hash": "9967655a7ccc860d040e50dd80eb2bda6f81032ac39ad18cba284d63dfdc2f63",
#   "claims_jcs": "{\"hash\":\"3494d2947b7929676faeb13ff604d686ad200a2e577130856cc4b63b1324350f\",\"hash_type\":\"sha256\",\"name\":\"contract.pdf\"}",
#   "hash": "3494d2947b7929676faeb13ff604d686ad200a2e577130856cc4b63b1324350f",
#   "hash_type": "sha256",
#   "id": "01a17d3b-5d88-7508-b87e-92cfcbf7a809",
#   "item_code": "JREEKECJ224AD2EV",
#   "item_hash": "961ce9b9921088a689d50592e3c8d4ef4fcba5441210f9ca53b967df1ca800d1",
#   "name": "contract.pdf",
#   "team_id": "01a0db60-0000-7000-8000-000000000001",
#   "visibility": "private"
# }

# Pick a file interactively (needs a terminal: in a pipe it refuses with
# "interactive file picker requires a terminal: pass --file=<path> or --claims=<path>")
truestamp items create --file

# Provide the content via stdin (name required separately)
curl -fsSL https://example.com/data.bin | truestamp items create -F -n "data.bin"

# Submit a precomputed claims JSON (`truestamp items create --help` lists the fields it may carry)
truestamp items create --claims=claims.json

# Claims via stdin
jq -c '.' claims.json | truestamp items create -C

# Flag-only: provide the hash and name directly, and the hash type with it.
# The server refuses a hash that is not hex of the length --hash-type
# implies (64 characters for sha256).
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

Every submission gives back the new item's `id`, `item_hash` and
`item_code`: each submission is a new item, with its own UUIDv7 id and its
own item code, even for the same file.

`--url` on `create` is a *claims* field (a link associated with the item), not
an input source, and it must start with `https://`. `--visibility` accepts
`private` (default), `team`, or `public`. `--timestamp` is a time you claim,
never an edge of the submission window; the CLI refuses one outside the
grammar the service checks (`YYYY`, `YYYY-MM`, `YYYY-MM-DD`, or a whole date
with `Thh:mm`, optional `:ss` and fraction, and an optional `Z` or offset) or
one naming a day that does not exist.

### Claims-as-source-of-truth mode

The claims content IS the timestamped data, so the server requires
the submission to carry meaningful content. Either:

* `claims.description` of at least 32 characters (code points), or
* a non-empty `claims.metadata` object.

The server checks this, with every other claims rule the CLI does not, and
says which rule a submission broke.

```sh
# Simplest claims-only submission: name plus a description
truestamp items create -n "Invention" \
  -d "On this day I claim the following novel approach as my own original work."

# Same, machine-readable. hash / hash_type keys are omitted from the
# JSON output entirely (use jq 'has("hash")' to branch on the mode).
truestamp items create -n "Invention" \
  -d "On this day I claim the following novel approach as my own original work." \
  --json
# {
#   "claims_hash": "d43e06ebc262150d85b744a69ae02b1bf97082e75999cf725958bb04ca0baafc",
#   "claims_jcs": "{\"description\":\"On this day I claim the following novel approach as my own original work.\",\"name\":\"Invention\"}",
#   "id": "01a17d3b-78e0-7420-9725-1badbc5982e4",
#   "item_code": "T9M2VBW186WVMFPZ",
#   "item_hash": "d2682daf8141b9ba3edcce33bbb8d3ee3114a87cbe296f44bef5300499a86549",
#   "name": "Invention",
#   "team_id": "01a0db60-0000-7000-8000-000000000001",
#   "visibility": "private"
# }

# Metadata escape hatch: no long description needed if metadata
# carries the content
truestamp items create -n "Release v1.2" \
  --metadata '{"version":"1.2.0","sha":"deadbeef","notes":"first release"}'

# Claims JSON file with no hash/hash_type: the proof commits to the
# claims themselves
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

| You did this | You'll see |
| --- | --- |
| `--hash-type sha256` without `--data-hash` | `claims_hash_pair: /hash is missing, and /hash_type needs it. Add hash, or remove hash_type.` |
| `--data-hash` without `--hash-type` | `claims_hash_pair: /hash_type is missing, and /hash needs it. Add hash_type, or remove hash.` |
| `--data-hash abc --hash-type sha256` | `claims_hash: /hash has 3 code points, and a sha256 hash is 64 hexadecimal digits. Send the whole hash.` |
| Claims with no `name` | `claims_name: /name is missing. Add a name of 1 to 64 code points.` |
| Claims-only with short description, no metadata | `claims_content: Claims without /hash need a /description of at least 32 code points, or a non-empty /metadata. Add one, or add hash and hash_type.` |
| `--url` not starting with `https://` | `claims_url: /url does not begin with https://. Give an https:// URL.` |
| Claims-only exceeding the plan byte budget | `claims_plan_size`, naming the plan's limit |
| `--metadata 'not json'` | `--metadata must be valid JSON: malformed_json (syntax) at octet 0` |
| `--visibility` outside the allowed set | `--visibility must be private, team, or public, got "secret"` |
| `--timestamp 2026-02-30` | `--timestamp names 2026-02-30, a date that does not exist` |
| A claim flag beside `--claims=` | `--name cannot be combined with --claims or --claims-stdin: the claims are sent exactly as given, so put every claim in the claims JSON itself` |
| A member name twice in claims on stdin (`-C`) | `parsing claims JSON from stdin: malformed_json (duplicate_name) at octet 12: a member name appears twice in one object; keep one` |

The rows with a `claims_` code come back from the server, each with its JSON
pointer into the claims; the CLI prints them all and creates nothing (exit
1):

```sh
truestamp items create -n "Q1 report" --hash-type sha256
#   Claims refused
#     pointer "/hash", claims_hash_pair: /hash is missing, and /hash_type needs it. Add hash, or remove hash_type.
#     pointer "", claims_content: Claims without /hash need a /description of at least 32 code points, or a non-empty /metadata. Add one, or add hash and hash_type.
#     Nothing was created. The server answered HTTP 422.
```

The other rows are refused by the CLI before any request: the flags it
parses itself (`--metadata`, `--visibility`, `--timestamp`, a claim flag
beside a claims file) and a claims text no verifier reads alike.

---

## `truestamp proofs get`

Fetch the proof of a subject whose block is committed to a public chain:
one JSON document, the envelope of SPEC.md section 3.1, exactly as the
service serves it. The subject is named by its UUIDv7 id, and an item also
by its item code or its `item_hash`. `--type` is **optional**:

- an item code names only an item, so it is `--type item` with no extra call;
- an `item_hash` (64 hex characters) needs `--type item`, since nothing else
  is named by a hash here;
- a UUIDv7 could be an item, an entropy observation, a block or a beacon, so
  the CLI asks the server what it is, in one extra round trip.

Pass `--type` to skip that call. A block's id is verifiable as a block and
as a beacon, which is refused rather than guessed.

The six `--type` values are the six types of SPEC.md section 3.2, the
`type` the proof carries:

| `--type` | The subject | `--to-file` writes |
| --- | --- | --- |
| `item` | an item, by its UUIDv7, its item code or its `item_hash` | `<CODE>.proof.json`, by its item code |
| `entropy_nist` | a NIST Randomness Beacon observation, by its UUIDv7 | `<uuidv7>.proof.json` |
| `entropy_stellar` | a Stellar ledger observation, by its UUIDv7 | `<uuidv7>.proof.json` |
| `entropy_bitcoin` | a Bitcoin block observation, by its UUIDv7 | `<uuidv7>.proof.json` |
| `block` | a block, by its UUIDv7 | `<uuidv7>.proof.json` |
| `beacon` | a block's beacon, by the block's UUIDv7 | `<uuidv7>.proof.json` |

Flag values use **underscores**: `--type entropy-nist`, bare
`--type entropy` and `--type auto` are all rejected. The `type` inside the
file is authoritative; the filename never is.

**Where the proof goes.** With no output flag the proof is written to
**stdout**, so it can be piped. `-o` / `--out` writes it to the path you
name. `--to-file` writes it to the conventional name in the current
directory, from the table above, the names the app gives its downloads.
`--out` and `--to-file` are mutually exclusive. When a file is written, a
receipt card goes to **stderr**, so stdout stays pipeable.

**`--constructions`** selects how the proof carries the constructions text
(SPEC.md section 7.2): `full`, the default, carries the text itself,
`reference` its URL, which names its SHA-256, and `none` neither. Each mode
is its own signed document, the verdict never depends on it, and
`--to-file` names them alike.

**Item proofs.** An item is named by its item code, its UUIDv7 id or its
whole `item_hash`. Substitute one of your own committed items
(`truestamp items list` prints both the id and the code).

```sh
# Written to the conventional file name in the cwd, <CODE>.proof.json
truestamp proofs get WX495M26THGQBJ68 --to-file
#   Proof Downloaded
#
#            File  WX495M26THGQBJ68.proof.json
#            Size  16,412 bytes
#       Item hash  e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c
#       Item code  WX495M26THGQBJ68
#            Type  item
#   Constructions  full
#         Details  https://www.truestamp.com/items/WX495M26THGQBJ68
#          Verify  https://www.truestamp.com/verify/item/WX495M26THGQBJ68

# Same, with the type stated rather than inferred (identical behaviour)
truestamp proofs get --type item --to-file WX495M26THGQBJ68

# The code may be typed in any case, with I and L read as 1 and O as 0,
# and with one space, hyphen or underscore between its groups of four; it
# is sent in its one spelling
truestamp proofs get wx49-5m26-thgq-bj68 -o contract.proof.json

# By the item's UUIDv7 id: one extra round trip resolves it as an item
truestamp proofs get 01a0db66-c2e0-78a4-b8cc-12b11bc97493 -o contract.proof.json

# By the whole item_hash, which needs --type item
truestamp proofs get --type item e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c -o contract.proof.json

# The constructions text by reference rather than in full; the same file name
truestamp proofs get --constructions reference --to-file WX495M26THGQBJ68
```

Before anything is written, the served proof is checked against the item
asked for: the record it carries must have the UUIDv7 given, or the
`item_hash` recomputed from that record must give the item code given, or
be the whole `item_hash` given. Otherwise the run refuses with "the server
returned a proof for a different item" and writes nothing. A matching code
shows only that the recomputed `item_hash` begins with the 75 bits the code
is written from.

**Block, beacon and entropy proofs.** Blocks and beacons share one
structural shape but carry distinct types in the signed proof, so a block
proof and a beacon proof of the same block have different signatures. The
proof's own `type` says which one you got.

```sh
# Block proof for a committed block
truestamp proofs get --type block 01a0db68-2270-796b-a428-e3946340a287 --to-file
#   Proof Downloaded
#
#            File  01a0db68-2270-796b-a428-e3946340a287.proof.json
#            Size  14,436 bytes
#              ID  01a0db68-2270-796b-a428-e3946340a287
#            Type  block
#   Constructions  full
#         Details  https://www.truestamp.com/blocks/01a0db68-2270-796b-a428-e3946340a287
#          Verify  https://www.truestamp.com/verify/block/01a0db68-2270-796b-a428-e3946340a287

# A beacon proof. A block's proof and its beacon's proof share one id, so
# --to-file gives them one name and the later write replaces the earlier:
# name one with -o when you keep both
truestamp proofs get --type beacon 01a0db64-03c0-7bd4-9d0c-7991f7ebfc5b -o beacon.proof.json
#   Proof Downloaded
#
#            File  beacon.proof.json
#            Size  15,880 bytes
#              ID  01a0db64-03c0-7bd4-9d0c-7991f7ebfc5b
#            Type  beacon
#   Constructions  full
#         Details  https://www.truestamp.com/blocks/01a0db64-03c0-7bd4-9d0c-7991f7ebfc5b
#          Verify  https://www.truestamp.com/verify/beacon/01a0db64-03c0-7bd4-9d0c-7991f7ebfc5b

# A block's bare UUIDv7 is verifiable as a block and as a beacon, so it
# needs --type
truestamp proofs get 01a0db68-2270-796b-a428-e3946340a287
# 01a0db68-2270-796b-a428-e3946340a287 is verifiable as more than one subject type, block | beacon, which is refused rather than guessed: pass --type block or --type beacon

# With no output flag the proof goes to stdout, so it redirects and pipes.
# An item's UUIDv7 resolves in one round trip.
truestamp proofs get 01a0db66-c2e0-78a4-b8cc-12b11bc97493 | truestamp verify --offline --environment test --keyring keyring.test.json | head -9
# Verdict:      supplied-keyring (environment test, not production)
#               The proof is intact and signed by a key in the keyring the operator supplied; this says nothing about Truestamp.
# Attribution:  keyring:e06820401e366a5325b9ec9788440fd4bc63f584e08a0fc7b33b4f2c5123c1fd
# Trust root:   supplied_keyring, from keyring.test.json, SHA-256 e06820401e366a5325b9ec9788440fd4bc63f584e08a0fc7b33b4f2c5123c1fd
# Environment:  test, not production: this run checks proofs of Truestamp's test environment
# Type:         item
# Subject id:   01a0db66-c2e0-78a4-b8cc-12b11bc97493
# Item hash:    e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c (recomputed from the subject's record)
# Item code:    WX495M26THGQBJ68 (of that item_hash: a name for the item, not a check)

# Entropy proofs take the UUIDv7 id of an entropy observation, which
# `truestamp entropy list` prints. An observation's id resolves as its own
# source, so --type is optional here too; under --to-file each writes
# <uuidv7>.proof.json.
truestamp proofs get --type entropy_stellar 01a0db66-6138-74af-aa5e-a51083f649fc > stellar.proof.json
truestamp proofs get --type entropy_nist    01a0db65-ec08-72a4-b5c3-508641e22274 > nist.proof.json
truestamp proofs get --type entropy_bitcoin 01a0db63-8e90-75fe-9c63-cd091cfb1cac > bitcoin.proof.json
```

**Failing fast.** The id is checked before the network call. `--type item`
takes a UUIDv7, an item code or an `item_hash`; every other type takes a
UUIDv7. Nothing is trimmed, and a UUID's 32-hex, braced and `urn:` forms
are refused. Mismatches, bad type names and conflicting output flags all
fail locally with exit code 1:

```sh
truestamp proofs get --type block WX495M26THGQBJ68
# --type block requires a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b); got an item code, which names only an item

truestamp proofs get e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c
# "e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c" is 64 hex characters, which names a subject only as an item's item_hash: pass --type item to fetch that item's proof

truestamp proofs get 01a0db66c2e078a4b8cc12b11bc97493
# "01a0db66c2e078a4b8cc12b11bc97493" is 32 hex characters, which is no id: proofs get takes a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b), an item code (e.g. 9WBPKNV7SCP87VZW) or, with --type item, an item_hash

truestamp proofs get WX495M26THGQBJ69
# "WX495M26THGQBJ69" has the shape of an item code but is not one: a symbol is mistyped or two are exchanged

truestamp proofs get --type entropy-nist 01a0db65-ec08-72a4-b5c3-508641e22274
# --type must be one of item | entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon, got "entropy-nist"

truestamp proofs get --type block -o x.json --to-file 01a0db68-2270-796b-a428-e3946340a287
# --out and --to-file are mutually exclusive: --out names a path, --to-file picks the conventional name
```

A UUIDv7 that the server cannot resolve is reported with both ways out:

```sh
truestamp proofs get 019d6a32-13e6-72b0-97e5-3779231ea97b
# 019d6a32-13e6-72b0-97e5-3779231ea97b does not resolve to anything you can fetch a proof for; pass --type explicitly if you believe it should
# (or pass --type explicitly: item | entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon)
```

A proof exists only once the subject's block is committed to a public
chain. Items reach a Truestamp block within about a minute of submission
and Stellar within about five; until then the server refuses with
`no_block_commitment`, and the CLI says that waiting helps:

```sh
truestamp proofs get --type block 01a0db65-6350-741d-be1f-442cbe5a8c0d
#   Proof not available yet
#     The subject's block has not been committed to a public blockchain yet. Try again after the next commitment.
#     A proof exists once the subject's block is committed to a public chain.
#     Items reach a Truestamp block within about a minute and Stellar within
#     about five.
#     This clears on its own: try again shortly.
#     The server said no_block_commitment (HTTP 400).
```

The refusals differ in whether waiting helps, and the CLI says which:

| `meta.code` | Meaning | Retry? |
| --- | --- | --- |
| `no_block_commitment` | the subject's block is not committed to a public chain yet | **yes**, clears in minutes |
| `commitment_not_final` | it is, and no commitment of it is final yet | **yes** |
| `evidence_unconfirmed` | Truestamp's own chain reads have not confirmed the evidence the proof would carry | **yes** |
| `subject_not_ready` | the item or observation is not committed in a block yet | **yes**, clears on its own |
| `witness_unavailable` | a witness the item's fingerprint commits to could not be found, so no proof is served without it | **yes** |
| `signing_unavailable` | Truestamp's signer refused | **yes** |
| `lookup_failed` | a record the proof is built from could not be read | **yes** |
| `subject_redacted` | the item's claims are redacted; restoring the item makes its proof available again | **no** |
| `proof_too_large` | the proof would be larger than the largest document Truestamp serves | **no**; report the subject id |
| `generation_failed` | Truestamp's records do not give a proof its own verifier accepts | **no**; report it |

Read the code from `meta.code`, not from the JSON:API `code` member: the
latter is the generic `"invalid"` on every one of these. The set above is
closed: any other code, such as `subject_type_mismatch` for an
observation fetched as another source's type, is printed as the server
gave it under "Proof request refused", with no advice about retrying:

```sh
truestamp proofs get --type entropy_nist 01a0db66-6138-74af-aa5e-a51083f649fc
#   Proof request refused
#     Requested type entropy_nist but subject 01a0db66-6138-74af-aa5e-a51083f649fc has source entropy_stellar
#     The server said subject_type_mismatch (HTTP 400).
```

If you only have a block hash and need the block's id, resolve it first with
`truestamp blocks get <hash>` or `truestamp beacons get <hash>`, which
accept either shape.

---

## `truestamp verify`

Verify a Truestamp proof, one JSON document, as an external verifier of
SPEC.md in truestamp/truestamp-verification: the run fixes its
environment, network pins, sources and keyring before the file is read,
then reads and parses the file, checks the envelope, the environment
marker, the signer's key and the signature, the schema, every network and
every recomputed hash, inclusion proof and link, and, online, reads each
Stellar and Bitcoin commitment and each entropy observation from its source
and draws the submission window from what it read (SPEC.md section 10.2).

```sh
# Local file
truestamp verify contract.proof.json

# URL, auto-detected from the positional argument
truestamp verify https://example.com/proof.json

# Stdin pipe
cat contract.proof.json | truestamp verify

# Interactive picker (needs a terminal; with stdin redirected it exits 1 with
# "the proof cannot be read: --file without a path: interactive prompt
# requires a terminal (stdin is piped or redirected)")
truestamp verify --file

# Compare the hash of the file you hold with the hash the item's claims
# carry; a mismatch fails the run
truestamp verify contract.proof.json \
  --expected-hash e08764deac64ca9a1046901c5b23674941f1e86f0e2d0429ee07c5e311a15ce7

# Offline: no calls to Truestamp, Stellar, Bitcoin or the entropy sources.
# Every cryptographic step still runs; each check that needs a read is
# reported as skipped, never failed. The proof's key must be among the
# keys built into the CLI: a sound proof then ends internally-consistent,
# exit 7. When the key is not among them, key binding fails, exit 3. An
# offline run is never verified.
truestamp verify contract.proof.json --offline

# Check the signer's key against a keyring file you supply, such as the
# downloaded keyring: a sound proof signed by a key in it ends
# supplied-keyring, exit 6
curl -sO https://www.truestamp.com/.well-known/keyring.json
truestamp verify contract.proof.json --offline --keyring keyring.json

# A proof of another Truestamp environment, against that environment's
# keyring. The examples here check the conformance vectors' test proof,
# proof.json, so.
truestamp verify proof.json --offline --environment test --keyring keyring.test.json

# Keyless: the signature checked only against the key the proof carries
truestamp verify proof.json --offline --environment test --no-keyring

# Read Stellar and Bitcoin from servers you name
truestamp verify contract.proof.json --horizon https://horizon.stellar.org --esplora https://blockstream.info/api

# Write the run's trace (SPEC.md section 10.6): one row per recomputed value
truestamp verify proof.json --offline --environment test --keyring keyring.test.json --trace proof.trace.json

# Script-friendly modes
truestamp verify contract.proof.json --json        # the report, exactly
truestamp verify contract.proof.json --silent      # exit code only
```

The type always comes from the proof's own signed `type`, and the filename
is never consulted: renaming a file can never change a verdict.

The text output names the label and what it says, the attribution, the
trust root, the environment, the type and subject id, for an item the
`item_hash` recomputed from its record and its item code, each edge of the
submission window, and every row that does not pass. This is the
conformance vectors' item proof, checked offline against the vectors'
keyring:

```sh
truestamp verify proof.json --offline --environment test --keyring keyring.test.json
```

```text
Verdict:      supplied-keyring (environment test, not production)
              The proof is intact and signed by a key in the keyring the operator supplied; this says nothing about Truestamp.
Attribution:  keyring:e06820401e366a5325b9ec9788440fd4bc63f584e08a0fc7b33b4f2c5123c1fd
Trust root:   supplied_keyring, from keyring.test.json, SHA-256 e06820401e366a5325b9ec9788440fd4bc63f584e08a0fc7b33b4f2c5123c1fd
Environment:  test, not production: this run checks proofs of Truestamp's test environment
Type:         item
Subject id:   01a0db66-c2e0-78a4-b8cc-12b11bc97493
Item hash:    e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c (recomputed from the subject's record)
Item code:    WX495M26THGQBJ68 (of that item_hash: a name for the item, not a check)
Submitted after:  not established in this run (offline at /witnesses/entropy_bitcoin, offline at /witnesses/entropy_nist, offline at /witnesses/entropy_stellar)
Submitted before: not established in this run (offline at /commitments/bitcoin, offline at /commitments/stellar)
Rows:         75 (64 pass, 0 fail, 9 skip, 0 warn, 2 info)
  info DERIVE not_attributable at /signing_key_event/block/record/metadata/key_event/public_key: The key event at /signing_key_event/block introduces the key that signed this proof, but this run does not attribute the proof to Truestamp, so it shows no Truestamp key history.
  skip CHAIN offline at /commitments/bitcoin: Not read: this run is offline. Confirming it needs reading the best-chain block at height 222 from a source of the pinned Bitcoin regtest network, with the tip at least 10 blocks above it, and then the block 10 above it, for its median time past.
  skip CHAIN offline at /commitments/stellar: Not read: this run is offline. Confirming it needs reading the transaction 860d4085c0af0b5f8cadff2bb0647cf75bbd16901817684dc4e180c200c70681 from a source of the pinned Stellar testnet network, which must be in ledger 1000255, and then that ledger, for its close time.
  skip CHAIN offline at /signing_key_event/commitments/bitcoin: Not read: this run is offline. Confirming it needs reading the best-chain block at height 222 from a source of the pinned Bitcoin regtest network, with the tip at least 10 blocks above it.
  skip CHAIN offline at /signing_key_event/commitments/stellar: Not read: this run is offline. Confirming it needs reading the transaction 860d4085c0af0b5f8cadff2bb0647cf75bbd16901817684dc4e180c200c70681 from a source of the pinned Stellar testnet network, which must be in ledger 1000255.
  skip CHAIN offline at /signing_key_event: Not read: this run is offline. Confirming it needs reading each commitment entry under /signing_key_event, and its genesis.bitcoin when it carries one, from the chains.
  skip SOURCES offline at /witnesses/entropy_bitcoin: Not read: this run is offline. Confirming it needs reading the block 4688b3b0f3274af83aeb59fe154f7eeda0091fd9d746dca9bd3f05863d50f16d from a source of the pinned Bitcoin regtest network.
  skip SOURCES offline at /witnesses/entropy_nist: Not read: this run is offline. Confirming it needs reading the pulse at chainIndex 2 and pulseIndex 1200018 from NIST's Randomness Beacon.
  skip SOURCES offline at /witnesses/entropy_stellar: Not read: this run is offline. Confirming it needs reading ledger 1000222 from a source of the pinned Stellar testnet network.
  info WINDOW window_not_established
  skip EXPECTED_HASH expected_hash_not_checked: No expected hash was given, so the claims' hash was not compared with one.
```

It exits `6`, the code of `supplied-keyring`. An online run of a production
proof does what each skip row above says it needs: it reads the keyring
from www.truestamp.com, so the trust root is `current_keyring` and the
attribution `truestamp`; it reads each commitment from its chain and each
witness from its source, so both edges of the window are established, each
naming the source it was read from ("read from <source> in this run"); and a
sound proof then ends in `verified`, exit `0`. When a network has several
sources, <source> also names its place in the network's order, as in
`read from https://btc.example, source 2 of the 2 sources of its network,
in this run`, since the CLI shows a source by its scheme, host and port
alone and two sources on one host would read alike.

Under any environment but `prod` the verdict line carries it, as above, and
a keyless run says so on a line of its own:

```sh
truestamp verify proof.json --offline --environment test --no-keyring | head -6
# Verdict:      unattributed (environment test, not production)
#               The proof is intact under the key it carries, which was checked against no keyring, so anyone could have made it.
# Attribution:  unattributed
# Trust root:   none
# Environment:  test, not production: this run checks proofs of Truestamp's test environment
# Keyless:      this run checked the signature only against the key the proof carries, against no keyring
```

A run labelled `failed` shows no window, and names the code of the first
failing check. A test proof checked as a production one fails at its
environment marker:

```sh
truestamp verify proof.json --offline | head -8
# Verdict:      failed
#               A check failed, and its row says which.
# Code:         environment_mismatch
# Attribution:  none
# Trust root:   compiled_pin, SHA-256 4e57588da2d0b7c34202d866664057aac36b5ee581dceb1eec9a0a35df8f9a6f
# Environment:  prod
# Window:       none is shown, since a check failed
# Rows:         14 (4 pass, 1 fail, 9 skip, 0 warn, 0 info)
```

`--expected-hash` takes the hex digest of the file you hold, lowercased and
never trimmed, and compares it with `claims.hash` (SPEC.md section 10.2,
step 13). Given, even as `--expected-hash ""`, it is a value: a proof with
no `claims.hash` of a listed type, a value that is not hex of the type's
length, and a hash that differs each fail the row `expected_hash_mismatch`.
Without it the row is `skip expected_hash_not_checked`.

Under `--json` stdout is exactly the report of SPEC.md section 10.5:
`attribution`, `code` (for a failed run), `environment`, `label`, `rows`
(each `{code, message, part, status, step, values}`), `subject_id`,
`submitted_after`, `submitted_before`, `trust_root`, `type`, `verified` and
`window`, written with JCS. It carries no item code.

```sh
truestamp verify proof.json --offline --environment test --keyring keyring.test.json --json | jq -r '.label, .window'
# supplied-keyring
# not_established
```

Use `--remote` to also ask the Truestamp service to verify the proof
(requires authentication: `truestamp auth login` or `TRUESTAMP_API_KEY`). Its
verdict is printed on stderr, beside the report and never as a row of it;
stdout and the exit code are this CLI's own run.

### Bare rejections

A file that is too large, is not one strict JSON text, or is not exactly the
envelope `{proof, signature}` is a bare rejection: no report and no verdict,
only `{code, message}`, with `sub_reason` beside them for `malformed_json`
and `invalid_envelope` alone, and exit code `4`. A configuration the
run cannot take before the file is read, such as `--environment staging`
with no keyring source, or a source that answers for the wrong network, is
`configuration_error`, exit code `8`.

```sh
echo '{"proof":{}}' | truestamp verify --offline --json
```

```json
{"code":"invalid_envelope","message":"This file is not a Truestamp proof.","sub_reason":"document"}
```

```sh
truestamp verify proof.json --environment staging --offline --json
```

```json
{"code":"configuration_error","message":"A run whose environment is staging checks proofs of that environment against a supplied keyring, and none was given: run again with --keyring FILE, with a keyring configured for staging, or with --no-keyring to check the signature only against the key the proof carries."}
```

```sh
# Branch on the exit code in a script
truestamp verify proof.json --silent; case $? in
  0) echo verified ;;
  3) echo failed ;;
  4|8) echo "not checked" ;;
  5|6|7) echo "intact, but not verified" ;;
  *) echo "other: see the report" ;;
esac
```

---

## `truestamp inspect`

Print what a proof carries without verifying it: the format, version, type,
environment marker and constructions mode, the signature, every member the
proof carries by its value class (SPEC.md section 3.7), `proof_hash`, the
key id of the signer's key, each value DERIVE recomputes beside the value
carried for it, for an item proof the item code, and the times and records
the signer declares. It reads the proof through the same reader and SHAPE
checks `verify` uses, so a malformed proof is refused with the code `verify`
would give, but it checks no key, no signature and no chain, and gives no
verdict.

```sh
truestamp inspect proof.json
truestamp inspect proof.json --json
cat proof.json | truestamp inspect

# The item code, computed from the recomputed item_hash
truestamp inspect proof.json --json | jq -r .item_code
# WX495M26THGQBJ68

# The item_hash, as carried and as recomputed
truestamp inspect proof.json --json | jq '.recomputed[] | select(.construction == "item_hash")'
# {
#   "construction": "item_hash",
#   "part": "/subject/item_hash",
#   "carried": "e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c",
#   "value": "e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c"
# }
```

The text starts with the proof's header and its signature:

```sh
truestamp inspect proof.json | head -14
```

```text
  Proof

  Source                   proof.json (16412 octets)
  Format                   truestamp-proof, version 1
  Type                     item
  Environment marker       test
  Constructions            full, digest c0d70826d7501d5089184d1947bad3dcc33c3172269d425f16fc0025ed07df49

  Signature
  ──────────────────────────────────────────
  Alg                      truestamp-ed25519-sha256-jcs
  Public key               IVL40Zt5HSRFMkLhXy6rbLfP-ntqXtMAl5YOBpiB2xI
  Value                    KeN18PcVrGiSwLNNBP4e2xg8NCLUR9spnN2LODRJWNcu_P1diXIyl7VZMkTmJkOg7kkWZdZN7M2dur14S8a-BA

```

Then come the sections "Carried by the proof", "Recomputed from what the
proof carries" (each value with its carried and recomputed forms, and the
item code after the recomputed `item_hash`), and "Declared by the signer
(not attributed to Truestamp)", which ends by saying that none of those
values is confirmed and no time among them is an edge of a submission
window. The last line is always `Nothing was verified. Run 'truestamp
verify' to verify the proof.`

`--json` prints the same material as an object keyed `source`, `octets`,
`rejection`, `signature`, `key_id`, `proof_hash`, `proof`, `recomputed`,
`declared_times`, `recorded` and `item_code` (null but for an item proof).

A proof that does not pass the reader and SHAPE is refused, with the step
and the code `verify` would report:

```sh
echo '{"proof":{}}' | truestamp inspect
```

```text
  Proof

  Source                   - (13 octets)
  Refused                  invalid_envelope (ENVELOPE, document)
                           This file is not a Truestamp proof.

  Nothing was verified. Run 'truestamp verify' to verify the proof.
```

Exit code `0` when the proof is read, `1` when it is refused.

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
truestamp teams get 01a0db60-0000-7000-8000-000000000001

# Create a team. With no name on a TTY, an interactive prompt opens.
truestamp teams create "Acme Research"
truestamp teams create "Acme Legal Two" --ownership-model team_retains --use
truestamp teams create -n "Acme Ops" --json

# Set the active team. The id is read back from the API before it is
# persisted, so a typo or a revoked membership refuses to write.
truestamp teams use 01a0db60-0000-7000-8000-000000000001

# No id on a TTY opens an interactive picker; Esc cancels without writing
truestamp teams use

# Clear the active team (the server falls back to your personal team)
truestamp teams use --clear
```

```text
  Teams (2)

       NAME                   ROLE     TEAM ID
       Personal (personal)    Owner    01a0db60-0000-7000-8000-000000000001
       Acme Legal             Admin    01a0db60-0000-7000-8000-000000000002
```

`teams list` renders a four-column table (`active marker`, `NAME`, `ROLE`,
`TEAM ID`). The marker column is blank on every row, as above, when no
team is configured; once a team is named, by `teams use`, `--team` or
`TRUESTAMP_TEAM`, that row leads with `★`. `teams get` and `teams current`
render the same detail card, which adds Personal, Ownership, Created and a
Details link, whose page needs sign-in (the link's host follows
`base_url`):

```text
  Team

         ID  01a0db60-0000-7000-8000-000000000001
       Name  Personal
       Role  Owner
   Personal  yes
  Ownership  creator_retains
    Created  2026-09-26T01:40:00Z
    Details  https://www.truestamp.com/teams/01a0db60-0000-7000-8000-000000000001
```

`teams current` heads the identical card `Team (active)`, and exits `1`
when nothing names a team, neither `config.toml` nor `--team` nor
`TRUESTAMP_TEAM`:

```text
  No team configured
    Run 'truestamp teams use' to pick one interactively, or 'truestamp teams list' to see all available teams.
```

`teams use <id>` prints the active team's card and where it was stored:

```text
  Active team updated
  Team (active)

         ID  01a0db60-0000-7000-8000-000000000001
       Name  Personal
       Role  Owner
   Personal  yes
  Ownership  creator_retains
    Created  2026-09-26T01:40:00Z
    Details  https://www.truestamp.com/teams/01a0db60-0000-7000-8000-000000000001

  Stored under 'team' in /Users/you/.config/truestamp/config.toml.
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

- **Monitor** with toggleable subscriptions to live event streams (blocks
  and their healing, beacons, commitments of entropy observations and your
  team's items into blocks and of blocks into epochs on Stellar and
  Bitcoin, NIST/Stellar/Bitcoin entropy observations, item events for your
  team) plus a scrollable, reversible event waterfall. Newest at top by
  default.
- **New Item**, a two-mode form (submission mode, name, description,
  plus hash type and hash in external-hash mode, hidden in
  claims-as-source-of-truth mode, where the description must be at least
  32 characters) that creates a timestamped item over the same socket and
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
`ws(s)://<host>/console/websocket` (`http` maps to `ws`, `https` to `wss`),
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
flowing immediately. The catalog is the server's: `blocks`,
`blocks.healing`, `beacons`, `commitments.subjects` (every entropy
observation's commitment into its block and your team's items'),
`commitments.blocks` (blocks into epochs on Stellar and Bitcoin),
`entropy.nist`, `entropy.stellar` and `entropy.bitcoin`, and `items.team`
when a team is in scope. The header shows the numbered pane tabs on the
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
`entropy.bitcoin.burst`. A `commitment.burst` on `commitments.blocks`
holds only block rows and renders as `commitment.block.burst`; one on
`commitments.subjects` may mix items and observations and stays
`commitment.burst`. The waterfall renders a burst as one row whose `ID`
column holds the plain integer event count; the rest of the payload
(`window_ms`, `first_at`, `last_at`, `by_kind`, `by_state`) lives in the
detail panel (`d`), with the maps flattened one level deep so you see
`by_kind.item.created` rather than a nested blob.

Slow streams (blocks, entropy, the commitments of blocks into epochs)
almost never trigger this. The first-event-immediate rule means a burst
only emerges when input rate genuinely warrants summarization. The client
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
# {"time":"2026-10-27T23:51:38.097996-04:00","level":"INFO","msg":"command_start","component":"console","cmd":"truestamp console","argc":0,"version":"dev","install_method":"unknown"}
# {"time":"2026-10-27T23:51:38.098204-04:00","level":"INFO","msg":"console session start","component":"console","ws_url":"wss://www.truestamp.com/console/websocket","log_path":"/tmp/truestamp-console.log","version":"dev"}
# {"time":"2026-10-27T23:51:38.10205-04:00","level":"ERROR","msg":"command_end","component":"console","duration_ms":4,"exit":1,"err":"bubbletea: error opening TTY: bubbletea: could not open TTY: open /dev/tty: device not configured"}
```

(`ws_url` is whatever your `--base-url` derives to, and `version` is the
binary's own, `dev` for a build from source; the run above ended early
because it had no controlling terminal.)

Defaults: 10 MB rotation, 14-day retention, 5 backups, gzip-compressed.
The Connection pane shows the live path. **Credentials are redacted**
before any error or log line touches the file or screen, covering the
API key, OAuth access and refresh tokens, and the PKCE code and
verifier, including the `?api_key=` / `?access_token=` query params a
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
# 3494d2947b7929676faeb13ff604d686ad200a2e577130856cc4b63b1324350f  contract.pdf

# Multiple files at once, one line per file
truestamp hash a.bin b.bin c.bin

# Stdin (filename shows as "-", matching sha256sum)
echo -n "abc" | truestamp hash
# ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad  -

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
# SHA256 (contract.pdf) = 3494d2947b7929676faeb13ff604d686ad200a2e577130856cc4b63b1324350f

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
truestamp hash --prefix 0x10 < payload.bin

# The Truestamp claims_hash one-liner: SHA-256(0x10 || JCS(claims)). The
# vectors' item proof carries its claims, and the record holds their hash.
jq -c .proof.subject.claims proof.json | truestamp hash --prefix 0x10 --jcs -a sha256 --style bare
# bd4341f1a42c9b0c085b7ae89df18bdab66eeb77284c1e53077e4ae45320f4f3
jq -r .proof.subject.record.claims_hash proof.json
# bd4341f1a42c9b0c085b7ae89df18bdab66eeb77284c1e53077e4ae45320f4f3

# JSON output with all three digest encodings
truestamp hash -a sha256 --json contract.pdf
```

Other input modes match `truestamp verify`: `--file=<path>` (or `--file`
alone for the interactive picker) and `--url=<url>` (or `--url` alone for
the interactive prompt).

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
Truestamp uses when computing claims / entropy / record hashes.

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

`truestamp jcs` reads its input as a Truestamp verifier reads a proof
(SPEC.md section 10.1). A number is read as its nearest IEEE-754 double and
written as RFC 8785 writes that double, and an integer of magnitude above
2^53 - 1 and below 10^21 is refused however it is written, because no two
verifiers would agree on it. Send such a value as a string instead.

```sh
printf '{"n":9007199254740991}' | truestamp jcs
# {"n":9007199254740991}

printf '{"n":9007199254740992}' | truestamp jcs
# stderr: JCS canonicalization: malformed_json (number) at octet 5
# exit 1

printf '{"n":1e16}' | truestamp jcs
# stderr: JCS canonicalization: malformed_json (number) at octet 5
# exit 1
```

`truestamp hash --jcs` reads its input the same way and refuses the same
values, naming the input that holds one
(`truestamp hash: -: JCS canonicalization: malformed_json (number) at octet 5`).

---

## `truestamp convert time`

Bidirectional time-format tool. Replaces most uses of `date` for parsing,
reformatting, and zone conversion. Accepts RFC 3339 or Unix seconds /
milliseconds / microseconds / nanoseconds; auto-detects by default.

```sh
# Current time in UTC (microsecond precision)
truestamp convert time now
# 2026-10-28T03:51:38.816347Z

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
# 2026-10-28T09:21:38+05:30

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

## `truestamp convert id`

Read the time Truestamp recorded in a UUIDv7 id. Every Truestamp record id
is a UUIDv7: items, blocks (and so beacons) and entropy observations. The
time is the one Truestamp wrote into the id when it made the record, a
recorded time, never a proven one: only a proof establishes a submission
window. An item code carries no time.

```sh
# UUIDv7 to its recorded time (UTC)
truestamp convert id 01a0db66-c2e0-78a4-b8cc-12b11bc97493
# 2026-09-26T01:49:00Z

# Another UUIDv7, whose milliseconds are not zero
truestamp convert id 019cf813-99b8-730a-84f1-5a711a9c355e
# 2026-03-16T19:16:00.056Z

# Convert to a specific zone (or Local)
truestamp convert id 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --to-zone America/New_York
# 2026-09-25T21:49:00-04:00

# Name the id type: auto (the default) or uuid7
truestamp convert id 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --type uuid7
# 2026-09-26T01:49:00Z

# Extract the raw bytes as hex instead of the time
truestamp convert id 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --extract raw
# 01a0db66c2e078a4b8cc12b11bc97493

# JSON output with every representation
truestamp convert id 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --json
# {
#   "type": "uuid7",
#   "value": "01a0db66-c2e0-78a4-b8cc-12b11bc97493",
#   "timestamp_ms": 1790387340000,
#   "timestamp_utc": "2026-09-26T01:49:00Z",
#   "zone": "UTC",
#   "local": "2026-09-26T01:49:00Z"
# }

# From stdin (common when fed from jq): the item's id in its proof
jq -r .proof.subject.record.id proof.json | truestamp convert id
# 2026-09-26T01:49:00Z
```

`--type` accepts `auto` (default) or `uuid7`. `--extract` accepts `time`
(default) or `raw`. A UUIDv7 is read as its 36-character text, so its
32-hex form is refused, and so is an item code:

```sh
truestamp convert id 01a0db66c2e078a4b8cc12b11bc97493 --type uuid7
# "01a0db66c2e078a4b8cc12b11bc97493" is 32 hex characters, which is no id: convert id takes a UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)

truestamp convert id WX495M26THGQBJ68
# WX495M26THGQBJ68 is an item code: an item code carries no time
```

---

## `truestamp convert keyid`

Derive the 4-byte Truestamp **key id** from an Ed25519 public key: the hex
of the first 4 bytes of SHA-256(0x60 || its 32 bytes) (SPEC.md section 3.3),
the `key_id` and `signing_key_id` a proof gives its signer, and the id
`truestamp keys get` looks up. Useful for confirming which signing key a
proof was issued under. The key below is the conformance vectors' test key.

```sh
# Base64url Ed25519 public key (the encoding a proof uses) to an 8-char hex key id
truestamp convert keyid IVL40Zt5HSRFMkLhXy6rbLfP-ntqXtMAl5YOBpiB2xI
# fbd680e8

# Hex input (the same key, hex-encoded, yields the same key id)
truestamp convert keyid --from hex \
  2152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12
# fbd680e8

# Auto-detect the encoding (hex / base64 / base64url); same key, standard base64
truestamp convert keyid IVL40Zt5HSRFMkLhXy6rbLfP+ntqXtMAl5YOBpiB2xI=
# fbd680e8

# JSON output carries all three encodings alongside the key id
truestamp convert keyid --json IVL40Zt5HSRFMkLhXy6rbLfP-ntqXtMAl5YOBpiB2xI
# {
#   "public_key_hex": "2152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12",
#   "public_key_base64": "IVL40Zt5HSRFMkLhXy6rbLfP+ntqXtMAl5YOBpiB2xI=",
#   "public_key_base64url": "IVL40Zt5HSRFMkLhXy6rbLfP-ntqXtMAl5YOBpiB2xI",
#   "kid_hex": "fbd680e8"
# }

# From stdin: the signer's key of a proof
jq -r .signature.public_key proof.json | truestamp convert keyid
# fbd680e8
```

`--from` accepts `auto` (default), `hex`, `base64` or `base64url`.

---

## `truestamp beacons`

Inspect Truestamp block beacons via the read-only JSON:API at
`/api/json/beacons/*`. A **beacon** is a compact projection of a
finalized block, four fields only: `{id, hash, timestamp,
previous_hash}`. It's a "proof of life" commitment: every item and
entropy observation committed inside that block is covered by the
beacon's hash.

A verifiable proof of a beacon is a separate document, fetched with
`truestamp proofs get --type beacon <block-id>` (see above).

```sh
# Most recent finalized beacon. A bare `truestamp beacons` prints help:
# a group is a namespace, never a command.
truestamp beacons latest
#   Beacon
#
#        Hash  22aff8c1347ada55bd0e4d23b10f62acef0e242abfb810d0f7ec7b53c042822b
#   Timestamp  2026-09-26T01:52:00Z  (32d ago)
#          ID  01a0db69-8200-7dd2-9ff8-ea5d21b285f2
#    Previous  38029526ddf5d8904f2b325b2a31e628bd49ca8fd9a8b772acf9359633a25552
#     Details  https://www.truestamp.com/beacons/22aff8c1347ada55bd0e4d23b10f62acef0e242abfb810d0f7ec7b53c042822b
#      Verify  https://www.truestamp.com/verify/beacon/01a0db69-8200-7dd2-9ff8-ea5d21b285f2

# Most-recent N beacons, newest first (default 25, at most 250 per page).
# Paging is the same as every other list, see
# Conventions, Paging lists: --after / --before, --oldest-first (the
# genesis beacon first), --max and --count.
truestamp beacons list
truestamp beacons list --limit 3
truestamp beacons list --oldest-first --limit 3
#   Beacons (3)
#
#   TIMESTAMP               HASH                                                                ID
#   2026-09-26T01:44:00Z    0e988763cc952d83ea9f887a66281a674e8898df50d49a1cb32fa21e4a7c8298    01a0db62-2f00-7272-bcdc-eaa719a3cbdc
#   2026-09-26T01:46:00Z    2ff6ae1fcc13afb761508b1cd84a251e0a182f34f5ad0df595e63cc83016b52d    01a0db64-03c0-7bd4-9d0c-7991f7ebfc5b
#   2026-09-26T01:47:00Z    bf93c2bf45b206b25c37f95c78b6a71fbdb04eeba2800acc18bf01574c7208d5    01a0db64-ee20-7419-8b82-ad06f52666ac

# Look up by UUIDv7 id
truestamp beacons get 01a0db67-ad40-773c-8f68-92f749710110

# Look up by 64-hex-char hash, useful when all you have is a hash
# (e.g. printed on a receipt or read from photo metadata). The two id
# shapes are disjoint, so both of these address the same record and
# print the same card.
truestamp beacons get 5e6d728fd156c04e24c56fc19339506c85e0c497f295181812d35b5325528d31
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
from `base_url`, so they name whichever origin the CLI is pointed at:
`https://www.truestamp.com` under the default configuration.
`beacons list` renders a `TIMESTAMP` / `HASH` / `ID` table.

```sh
# Capture the current chain head as a moment-in-time commitment.
# --hash-only lives on the subcommand, not on the bare group.
MOMENT=$(truestamp beacons latest --hash-only)
echo "beacon hash: $MOMENT"
# beacon hash: 22aff8c1347ada55bd0e4d23b10f62acef0e242abfb810d0f7ec7b53c042822b

# Pipeline-friendly JSON (the paged envelope for `list`, an object otherwise)
truestamp beacons list --limit 3 --json | jq -r '.beacons[].hash'
# 22aff8c1347ada55bd0e4d23b10f62acef0e242abfb810d0f7ec7b53c042822b
# 38029526ddf5d8904f2b325b2a31e628bd49ca8fd9a8b772acf9359633a25552
# 5e6d728fd156c04e24c56fc19339506c85e0c497f295181812d35b5325528d31

# Round-trip: id to hash to id (demonstrates by-hash lookup)
ID=$(truestamp beacons latest --json | jq -r .id)
HASH=$(truestamp beacons get "$ID" --hash-only)
truestamp beacons get "$HASH" --json | jq .id
# "01a0db69-8200-7dd2-9ff8-ea5d21b285f2"
```

Client-side validation catches obvious typos without hitting the
network:

```sh
truestamp beacons get not-a-uuid
# invalid UUID "not-a-uuid": an id is the 36-character text of a UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)

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
than a separate command. Every command here takes an item by its UUIDv7
id, its item code or its whole `item_hash`.

```sh
# The items in the team the CLI is pointed at, newest first
truestamp items list
#   Items (3)
#
#   ID                                      CODE                STATE          NAME
#   01a0db6d-2b80-7928-9225-f8760ae1a795    HN56SH5HC26EZ9R4    ✓ committed    the first item of the second chain
#   01a0db69-0cd0-73d0-9535-27f94d79d287    RM3XZA6CN92D7N4H    ✓ committed    an item submitted after the rotation
#   01a0db66-c2e0-78a4-b8cc-12b11bc97493    WX495M26THGQBJ68    ✓ committed    Q3 report "final", saved as C:\reports\q3.pdf

# Filter on commitment state: only the ones in a block, or only the ones
# still waiting for one. Every item on this team is already committed, so
# --committed repeats the three rows above and --pending finds nothing.
truestamp items list --committed
truestamp items list --pending
#   No items.

# Page explicitly. A page that continues prints the cursor that continues
# it, on stderr when stdout is a terminal, and as next_cursor in --json.
truestamp items list --limit 2
#   Items (2)
#
#   ID                                      CODE                STATE          NAME
#   01a0db6d-2b80-7928-9225-f8760ae1a795    HN56SH5HC26EZ9R4    ✓ committed    the first item of the second chain
#   01a0db69-0cd0-73d0-9535-27f94d79d287    RM3XZA6CN92D7N4H    ✓ committed    an item submitted after the rotation

truestamp items list --limit 2 --after "$(truestamp items list --limit 2 --json | jq -r .next_cursor)"
#   Items (1)
#
#   ID                                      CODE                STATE          NAME
#   01a0db66-c2e0-78a4-b8cc-12b11bc97493    WX495M26THGQBJ68    ✓ committed    Q3 report "final", saved as C:\reports\q3.pdf
truestamp items list --limit 100

# ...or follow cursors until a cap you choose. There is no unbounded walk:
# the tables behind these lists grow by the minute, so --max is the price
# of following pages. Each page is --limit rows; the last asks for only the
# rows still wanted, so the cursor printed at the end continues exactly.
truestamp items list --max 200 --json | jq -r '.items[].item_code'
# HN56SH5HC26EZ9R4
# RM3XZA6CN92D7N4H
# WX495M26THGQBJ68

# --count adds the server's total to the heading (and "total" to --json)
truestamp items list --limit 2 --count | head -1
#   Items (2 shown, 3 total)

# Walk from the beginning instead, and step back with the Back: cursor a
# page prints (--before is the mirror of --after)
truestamp items list --oldest-first --limit 2
truestamp items list --oldest-first --before <cursor>

# One item, by its item code, including whether it is in a block yet
truestamp items get WX495M26THGQBJ68
#   Item
#
#            ID  01a0db66-c2e0-78a4-b8cc-12b11bc97493
#         State  committed  (in a block; its proof follows once the block is committed to a public chain)
#    Visibility  private
#          Name  Q3 report "final", saved as C:\reports\q3.pdf
#   Claims hash  bd4341f1a42c9b0c085b7ae89df18bdab66eeb77284c1e53077e4ae45320f4f3
#     Item hash  e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c
#     Item code  WX495M26THGQBJ68
#          Team  01a0db60-0000-7000-8000-000000000001
#     Submitted  2026-09-26T01:49:00Z  (32d ago)
#       Details  https://www.truestamp.com/items/WX495M26THGQBJ68

# A Tags row appears after Name when the item carries any.

# The same item by its UUIDv7 id or its item_hash
truestamp items get 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --json
truestamp items get e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c
```

The item card closes with a Details URL at `/items/<CODE>`, by the item's
code, on whichever origin `base_url` names. Unlike the beacon card's
links, that page needs sign-in.

`items list --json`, like every keyset-paged list (`blocks list`,
`beacons list`, `entropy list`), is an object: `{"items": [...], "next_cursor": "...",
"prev_cursor": "..."}`, plus `"total"` under `--count`, with `next_cursor`
empty once the last page has been read and `prev_cursor` empty on the
first. Reach for `.items[]` in `jq`, not `.[]`.
`--limit` is bounded on one side only: the
CLI refuses `--limit 0` locally, because the API contract states a minimum
of 1, and forwards everything else. The ceiling is the server's: it lowers
a page above 250 to 250 rather than refusing it, and the listing says so
when a page follows (see [Paging lists](#paging-lists)).

Only three attributes are mutable: visibility, tags, and the owning team.
Claims are immutable because `claims_hash` is bound into the item's hash,
and there is no flag on `update` that can reach them. `update` takes the
item by its UUIDv7 id, its item code or its `item_hash`, resolves a code or
a hash with one read, and changes the item by its UUIDv7 id. Substitute one of your own items for the
one below: these three lines change stored state.

```sh
truestamp items update WX495M26THGQBJ68 --visibility public
truestamp items update WX495M26THGQBJ68 --tags q3,contracts

# Move it to another team. --to-team, not the root --team: that one says
# which tenant scopes the request.
truestamp items update WX495M26THGQBJ68 --to-team 01a0db60-0000-7000-8000-000000000002
```

Each prints the item card as it now stands. With none of the three flags it
refuses: `nothing to update: pass --visibility, --tags or --to-team`.

A server-reported state is a claim. The authoritative answer is a proof
you check yourself, again with your own item in place of this one:

```sh
truestamp proofs get WX495M26THGQBJ68 | truestamp verify
```

## `truestamp blocks`

Read-only access to Truestamp's block chain. A block is the full record
(height, state, block hash, Merkle root, chain links, signing key id),
where a beacon is its four-field public projection.

```sh
truestamp blocks latest       # the head block, the one of greatest height
truestamp blocks genesis      # the first block, height 0, where every chain walk ends
truestamp blocks list --limit 10
truestamp blocks list --oldest-first --limit 3            # genesis first
truestamp blocks list --limit 100 --max 1000 --count --json | jq -r '.blocks[].id'
truestamp blocks get 01a0db67-ad40-773c-8f68-92f749710110
truestamp blocks get 5e6d728fd156c04e24c56fc19339506c85e0c497f295181812d35b5325528d31 --json
```

```text
  Block

           ID  01a0db67-ad40-773c-8f68-92f749710110
       Height  4
        State  committed
   Block Hash  5e6d728fd156c04e24c56fc19339506c85e0c497f295181812d35b5325528d31
  Merkle Root  a72265e95bd9aa269da573f251092c2c85fd0026c7115e9476ab074ed579f982
     Previous  940829eb0fc4717dac15092189394ef52fbbfda555ef5ff0f4c3878fb374a98e
  Signing Key  9602c03e
      Created  2026-09-26T01:50:00Z  (32d ago)
```

`blocks list` orders blocks by height, the highest first, and leads each
row with it:

```sh
truestamp blocks list --limit 3
#   Blocks (3)
#
#   HEIGHT    STATE        ID                                      HASH
#   6         committed    01a0db69-8200-7dd2-9ff8-ea5d21b285f2    22aff8c1347ada55bd0e4d23b10f62acef0e242abfb810d0f7ec7b53c042822b
#   5         committed    01a0db68-2270-796b-a428-e3946340a287    38029526ddf5d8904f2b325b2a31e628bd49ca8fd9a8b772acf9359633a25552
#   4         committed    01a0db67-ad40-773c-8f68-92f749710110    5e6d728fd156c04e24c56fc19339506c85e0c497f295181812d35b5325528d31
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

`blocks` and `beacons` address the same rows: every block is created
finalized, so each block projects as a beacon and `blocks latest` and
`beacons latest` name the same head. They differ in shape. A block carries
the whole record (`id`, `height`, `state`, `block_hash`, `merkle_root`,
`previous_block_id`, `previous_block_hash`, `signing_key_id`, `signature`,
`inserted_at`, in its `--json`); a beacon is the four-field projection
(`id`, `hash`, `timestamp`, `previous_hash`).

A block has a proof of its own, but only once it has been committed to a
public chain. Feeding `blocks latest` into `proofs get` is therefore
unreliable: the head is often not committed yet, and `proofs get` then
exits 1 with `no_block_commitment`:

```sh
truestamp proofs get --type block "$(truestamp blocks latest --json | jq -r .id)"
#   Proof not available yet
#     The subject's block has not been committed to a public blockchain yet. Try again after the next commitment.
#     A proof exists once the subject's block is committed to a public chain.
#     Items reach a Truestamp block within about a minute and Stellar within
#     about five.
#     This clears on its own: try again shortly.
#     The server said no_block_commitment (HTTP 400).
```

Pin a block that has already been committed:

```sh
truestamp proofs get --type block 01a0db68-2270-796b-a428-e3946340a287 | truestamp verify --offline --environment test --keyring keyring.test.json | head -9
# Verdict:      supplied-keyring (environment test, not production)
#               The proof is intact and signed by a key in the keyring the operator supplied; this says nothing about Truestamp.
# Attribution:  keyring:e06820401e366a5325b9ec9788440fd4bc63f584e08a0fc7b33b4f2c5123c1fd
# Trust root:   supplied_keyring, from keyring.test.json, SHA-256 e06820401e366a5325b9ec9788440fd4bc63f584e08a0fc7b33b4f2c5123c1fd
# Environment:  test, not production: this run checks proofs of Truestamp's test environment
# Type:         block
# Subject id:   01a0db68-2270-796b-a428-e3946340a287
# Submitted after:  none: this proof carries nothing that could give this edge
# Submitted before: not established in this run (offline at /commitments/bitcoin, offline at /commitments/stellar)
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
`entropy_*` proof types, so one word names both; the card and the
list drop the `entropy_` prefix for display (`stellar`), and `--json`
keeps the wire name.

```sh
# Newest observation from any source. A bare `truestamp entropy` prints help.
truestamp entropy latest
#   Entropy Observation
#
#             ID  01a0db66-6138-74af-aa5e-a51083f649fc
#         Source  stellar
#        Network  testnet
#          State  committed
#   Entropy Hash  2bfcd684973d6e7234cb0d27174cd2b317cf53219ddd8609d6728a286b046dab
#      Published  2026-09-26T01:48:30Z  (32d ago)
#       Captured  2026-09-26T01:48:35Z  (32d ago)
#          Block  01a0db67-ad40-773c-8f68-92f749710110
#      closed_at  2026-09-26T01:48:30Z
#           hash  84e52c6a9487098304fe1c3ccd550a0c323e330db2e8854c17145b217a94c6e7
#       sequence  1000222
#        Details  https://www.truestamp.com/entropy/01a0db66-6138-74af-aa5e-a51083f649fc
#         Verify  https://www.truestamp.com/verify/entropy_stellar/01a0db66-6138-74af-aa5e-a51083f649fc

# Newest from one source: what a newly submitted item would commit to for it
truestamp entropy latest --source entropy_nist
truestamp entropy latest --source entropy_bitcoin --json | jq -r .entropy_hash
# c5e73cea68aaf7999c859639b8b7816f356be9b40d0ff8304b9001377ae040ca

# Most recent observations, newest first, across sources or from one
truestamp entropy list --limit 4
#   Entropy Observations (3)
#
#   PUBLISHED               SOURCE     STATE        ID                                      HASH
#   2026-09-26T01:48:30Z    stellar    committed    01a0db66-6138-74af-aa5e-a51083f649fc    2bfcd684973d6e7234cb0d27174cd2b317cf53219ddd8609d6728a286b046dab
#   2026-09-26T01:48:00Z    nist       committed    01a0db65-ec08-72a4-b5c3-508641e22274    1d2d1432a7ee7221ec781b943d017779b83930b6478f7e58b567e08587fe1482
#   2026-09-26T01:45:00Z    bitcoin    committed    01a0db63-8e90-75fe-9c63-cd091cfb1cac    c5e73cea68aaf7999c859639b8b7816f356be9b40d0ff8304b9001377ae040ca
truestamp entropy list --source entropy_bitcoin --limit 2

# Page like every other list: --after and --before continue from a printed
# cursor in either direction, --oldest-first starts at the first observation
# ever captured, --max follows cursors up to a cap, --count adds the total
truestamp entropy list --oldest-first --limit 3
truestamp entropy list --source entropy_bitcoin --limit 50 --max 200 --json | jq -r '.observations[].entropy_hash'
truestamp entropy list --count --limit 1 | head -1
#   Entropy Observations (1 shown, 3 total)

# Look up by UUIDv7 id, or by the 64-hex entropy hash
truestamp entropy get 01a0db66-6138-74af-aa5e-a51083f649fc
truestamp entropy get 1d2d1432a7ee7221ec781b943d017779b83930b6478f7e58b567e08587fe1482 --json
```

The card shows the observation's identity, its source and, for Stellar and
Bitcoin, its network, its state (`created` until a block commits it, then
`committed`, with the block's id as a `Block` row), the `Entropy Hash`,
when the source published the value and when Truestamp captured it, then
the source's own record under the source's own field names: a Stellar
ledger's `closed_at`, `hash` and `sequence`; a NIST pulse as
`pulse.chainIndex`, `pulse.outputValue`, `pulse.pulseIndex` and so on; a
Bitcoin block's fields likewise. Nothing is renamed, and the list prints
every hash whole, as the card and `--json` do.

The entropy hash is the value an item's record commits to under
`witness_hashes`, which makes `get` by hash the way to trace a witness named
in a proof back to the observation it came from, and each observation is a
proof subject in its own right:

```sh
# From an item proof to the Stellar observation it witnessed
truestamp entropy get "$(jq -r .proof.subject.record.witness_hashes.entropy_stellar proof.json)"

# From an observation to its own proof, verified
truestamp proofs get 01a0db66-6138-74af-aa5e-a51083f649fc | truestamp verify --offline --environment test --keyring keyring.test.json | head -9
# Verdict:      supplied-keyring (environment test, not production)
#               The proof is intact and signed by a key in the keyring the operator supplied; this says nothing about Truestamp.
# Attribution:  keyring:e06820401e366a5325b9ec9788440fd4bc63f584e08a0fc7b33b4f2c5123c1fd
# Trust root:   supplied_keyring, from keyring.test.json, SHA-256 e06820401e366a5325b9ec9788440fd4bc63f584e08a0fc7b33b4f2c5123c1fd
# Environment:  test, not production: this run checks proofs of Truestamp's test environment
# Type:         entropy_stellar
# Subject id:   01a0db66-6138-74af-aa5e-a51083f649fc
# Submitted after:  not established in this run (offline at /subject/entropy)
# Submitted before: not established in this run (offline at /commitments/bitcoin, offline at /commitments/stellar)
```

Online, the run reads the observation from its source (Horizon for a
Stellar ledger) and compares it with what the proof carries, which is what
gives the observation its submitted-after edge.

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
unauthenticated from `/.well-known/keyring.json`, and online, under the
production environment, `verify` reads the same document for its key
binding step. The examples below read the conformance vectors' test
keyring, whose one key is `fbd680e8`.

```sh
truestamp keys list
#   Signing Keys (1)
#
#   0  fbd680e8  IVL40Zt5HSRFMkLhXy6rbLfP-ntqXtMAl5YOBpiB2xI  (active)

truestamp keys current                 # the key signing right now
truestamp keys get fbd680e8            # by its 8-hex key id
#   Signing Key
#
#       Key ID  fbd680e8
#   Public Key  IVL40Zt5HSRFMkLhXy6rbLfP-ntqXtMAl5YOBpiB2xI
#     Sequence  0
#       Active  true

truestamp keys current --json | jq -r .public_key
# IVL40Zt5HSRFMkLhXy6rbLfP-ntqXtMAl5YOBpiB2xI

truestamp keys get 3c19f776
# no published key with id "3c19f776"      (exit 1)
```

`current`, not `latest`: the keyring is populated by chain replay
including prerotation events, so a published-but-not-yet-active key would
make the two diverge.

The keyring answers the one question a proof cannot answer about itself,
*is this key Truestamp's*, because a forged proof is self-consistent by
construction and carries its forger's own public key. It does not tell you
a key was uncompromised, valid at signing time, or authorized; it carries
no revocation flag, no validity interval and no timestamps.

```sh
# Derive a key id from a public key you already hold, and look it up
truestamp keys get $(truestamp convert keyid IVL40Zt5HSRFMkLhXy6rbLfP-ntqXtMAl5YOBpiB2xI)
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
#     algorithms  Hash algorithms accepted by `truestamp hash -a`
#       commands  The full command tree: every path, flag, type, default and enum
#     exit-codes  What each process exit status means, by command
#      item-code  The item code (SPEC.md section 3.9): its alphabet, and each member, row and name that carries it
#   reason-codes  The reason codes of a verify report row or bare rejection (SPEC.md section 12.1)

truestamp schema get commands --json | jq -r '.subcommands[].path'
truestamp schema get algorithms --json | jq -r '.algorithms[].name'
truestamp schema get item-code
truestamp schema get reason-codes
truestamp schema get exit-codes
```

`schema get commands --json` is the one call that answers "what can this
CLI do, and with which flags" without parsing help text. It carries each
flag's name, usage, type, default, shorthand, whether it is inherited,
and its closed value set where the flag has one. `schema get item-code`
names the item code's alphabet and length, the commands that take one, and
every place the CLI writes one:

```sh
truestamp schema get item-code | head -6
#   The item code (SPEC.md section 3.9)
#
#   alphabet  0123456789ABCDEFGHJKMNPQRSTVWXYZ
#   length    16
#   taken by  items get, items update, proofs get
#
```

## `truestamp upgrade`

Self-upgrade the binary. The command is **install-method aware**. Homebrew
users are shown `brew upgrade`, `go install` users are shown the
`go install ...@latest` incantation, and `install.sh` users get a native-Go
in-place upgrade that mirrors the install script (SHA-256 mandatory, cosign
best-effort, atomic rename with timestamped backup).

```sh
# Is a newer release available? Prints the answer and exits 0 either way.
truestamp upgrade --check

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

`--check` asks the GitHub Releases API, so it needs the network, and prints
one line saying whether this binary is up to date or which release is
newer. `truestamp version` reports the detected install method, which is
what decides between the "print instructions" and "replace in place" paths.

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
# truestamp dev (commit: unknown, built: unknown)
```

That is a binary built from source without the release flags; a release
build prints its version, its short commit and its build date in their
place.

---

## Pipeline recipes

Real-world compositions that solve a specific problem.

Where a recipe below shows sample output, it was captured from the
conformance vectors' item proof, `proof.json` (an external-hash proof whose
claims carry an `md5` hash), and, for the pasteable `claims_hash` check, the
vectors' `item-genesis-witness.proof.json`. Your own `proof.json` will carry
different values but the same shape: the envelope
`{"proof": {...}, "signature": {...}}`, with the item's claims, record and
item hash under `.proof.subject`, its block under `.proof.block`, its
commitments under `.proof.commitments` and its witnesses under
`.proof.witnesses`.

### Reproduce the protocol's `claims_hash`

The Truestamp protocol computes `claims_hash = SHA-256(0x10 || JCS(claims))`,
and the item's record carries it: `.proof.subject.record.claims_hash`, which
the item hash, `SHA-256(0x11 || JCS(record))`, then commits to. A verifier
recomputes both from the claims the proof carries. To reproduce
`claims_hash` from a claims JSON document:

```sh
truestamp hash --prefix 0x10 --jcs \
  -a sha256 --style bare \
  < claims.json
```

Or the explicit two-step form (useful when you want to inspect the
canonical JSON bytes in between):

```sh
truestamp jcs < claims.json \
  | truestamp hash --prefix 0x10 -a sha256 --style bare
```

Both forms produce the same digest. A self-contained check you can paste
verbatim, using the claims of the vectors' `item-genesis-witness` proof
(note that the keys are deliberately out of order here: JCS sorts them, so
the result is unaffected):

```sh
echo '{"name":"the first item of the second chain","hash_type":"sha256","hash":"0be5be687aa93ca2f437e810631742601bd9534be176678c7750f295ecac58e2"}' \
  | truestamp hash --prefix 0x10 --jcs -a sha256 --style bare
# 8a3d0d33ba8f9c47f8e283e9f3bcc87ada35aed0ce21f96150478e31be865b2f

# The canonical bytes that digest is taken over, if you want to see them:
echo '{"name":"the first item of the second chain","hash_type":"sha256","hash":"0be5be687aa93ca2f437e810631742601bd9534be176678c7750f295ecac58e2"}' \
  | truestamp jcs
# {"hash":"0be5be687aa93ca2f437e810631742601bd9534be176678c7750f295ecac58e2","hash_type":"sha256","name":"the first item of the second chain"}
```

`8a3d0d33ba8f9c47f8e283e9f3bcc87ada35aed0ce21f96150478e31be865b2f` is the
`claims_hash` that proof's record carries, so a matching digest confirms
your `jcs` + `hash` composition agrees with the protocol. The same check
runs on any item proof you hold:

```sh
jq -c .proof.subject.claims proof.json | truestamp hash --prefix 0x10 --jcs -a sha256 --style bare
# bd4341f1a42c9b0c085b7ae89df18bdab66eeb77284c1e53077e4ae45320f4f3
jq -r .proof.subject.record.claims_hash proof.json
# bd4341f1a42c9b0c085b7ae89df18bdab66eeb77284c1e53077e4ae45320f4f3
```

**To confirm that a file matches what was timestamped**, you don't need this
value. Compare the file's hash, of the claims' `hash_type`, against
`.proof.subject.claims.hash`, or pass `--expected-hash <hex>` to
`truestamp verify` (see the next two recipes).

### Derive a key id from a proof's signer key, without `jq`

Canonicalize first so the `"public_key"` member is guaranteed to be compact,
then slice it out with plain `grep` / `cut`. JCS sorts the envelope's
members, so `signature` comes after `proof` and the last match is the
signer's key, `signature.public_key` (a signing key event carries the same
key again):

```sh
truestamp jcs < proof.json \
  | grep -o '"public_key":"[^"]*"' | tail -1 | cut -d'"' -f4 \
  | truestamp convert keyid
# fbd680e8
```

Skipping the `jcs` step only works on an already-minified proof; a
pretty-printed proof writes `"public_key": "..."` with a space, which the
pattern above would miss. It fails loudly rather than silently, because
`convert keyid` refuses an empty input:

```sh
grep -o '"public_key":"[^"]*"' pretty-proof.json | tail -1 | cut -d'"' -f4 \
  | truestamp convert keyid
# empty stdin input
# exit 1
```

With `jq` (cleaner):

```sh
jq -r .signature.public_key proof.json | truestamp convert keyid
# fbd680e8
```

Both spellings print the same key id.

### Read the time recorded in a proof's block id (in your local zone)

The block's UUIDv7 id carries the time Truestamp recorded when it made the
block, a recorded time and never an edge of the submission window:

```sh
jq -r .proof.block.record.id proof.json | truestamp convert id --to-zone America/New_York
# 2026-09-25T21:50:00-04:00
```

### Confirm a downloaded file matches what you timestamped

```sh
# Hash the local file with the claims' hash_type, and compare it with the
# hash the proof commits to. Prints "match" or "MISMATCH".
expected="$(jq -r .proof.subject.claims.hash proof.json)"
algo="$(jq -r .proof.subject.claims.hash_type proof.json)"
actual="$(truestamp hash -a "$algo" --style bare contract.pdf 2>/dev/null)"
[ "$expected" = "$actual" ] && echo "match" || echo "MISMATCH"
# MISMATCH
```

(`contract.pdf` here is not the file the vectors' item was made from, so it
does not match.) This recipe is **external-hash mode only**. For
claims-as-source-of-truth proofs there's no separate file to compare,
because the claims content itself is what the proof commits to. You can
spot a claims-only proof at a glance with
`jq '.proof.subject.claims | has("hash") | not' proof.json` (`true` means
claims-only).

### Verify a proof while passing the expected hash inline

```sh
truestamp verify proof.json --offline --environment test --keyring keyring.test.json --expected-hash "$(truestamp hash -a sha256 --style bare contract.pdf)"
```

The EXPECTED_HASH row says how the comparison went (SPEC.md section
10.2, step 13). The text report lists every row that did not pass:

```text
# match: the row passes, and the run ends as it would without the flag
# mismatch: the row fails, the label is failed, exit 3
#   fail EXPECTED_HASH expected_hash_mismatch at /subject/claims/hash: The expected hash differs from the claims' hash.
# a value that is not hex of the claims' hash type, as a sha256 digest against the md5 hash of proof.json:
#   fail EXPECTED_HASH expected_hash_mismatch at /subject/claims/hash: The expected hash was not matched: it is not md5 hex, which is 32 hex characters.
# a claims-only proof, claims-only.proof.json:
#   fail EXPECTED_HASH expected_hash_mismatch at /subject/claims/hash: The expected hash was not matched: this proof's claims hold no hash of a type the comparison lists.
```

A supplied hash that does not match is a failure, and so is one supplied
for a claims-as-source-of-truth proof, which holds no hash to compare it
with. A match against an `md5` or `sha1` claims hash warns
`expected_hash_weak_algorithm`, since collisions are practical for both:

```sh
truestamp verify proof.json --offline --environment test --keyring keyring.test.json \
  --expected-hash 69e3941192b086bfdc0e4e120c18ff12 --json \
  | jq -r '.rows[] | select(.step=="EXPECTED_HASH") | [.status, .code // empty] | join(" ")'
# warn expected_hash_weak_algorithm
```

### Batch-verify every proof in a directory, silently

```sh
find proofs -name '*.json' -print0 \
  | xargs -0 -I{} sh -c 'truestamp verify --silent --offline --environment test --keyring keyring.test.json "{}"; code=$?; [ $code -eq 6 ] || echo "exit $code: {}"'
# exit 3: proofs/tamper-claims.json
```

`--silent` suppresses the report entirely and keeps the exit code, so the
only output is one line per proof that did not end as expected. The
directory above holds the vectors' positive proofs and, as
`tamper-claims.json`, their `negative/declared-claims_hash.proof.json`,
checked against the vectors' keyring, so a sound one ends in
`supplied-keyring`, exit code `6`. For production proofs, drop
`--environment` and `--keyring`. An offline run reads no chain, and its
keyring is the keys built into the CLI: when the proof's key is among them,
a sound production proof ends in `internally-consistent`, exit code `7`;
when it is not, key binding fails, exit code `3`, and `--keyring` with a
downloaded keyring gives `supplied-keyring`, exit code `6`. `0`,
`verified`, needs an online run. `3` is `failed`, `4` a bare rejection and
`8` a `configuration_error` (SPEC.md section 12.2;
[Exit codes](README.md#exit-codes)).

### Hash a file with every supported algorithm in one pass

```sh
for alg in $(truestamp hash --list | awk '{print $1}'); do
  truestamp hash -a "$alg" --style bsd contract.pdf
done
# MD5 (contract.pdf) = 619947b72f5e7fff68ebf56e4e4a848e
# SHA1 (contract.pdf) = 29db3e9faf72a596fc487e9f627b3807f77c179e
# SHA224 (contract.pdf) = 7af44813022b64cb4e0c725a82c9abe9dd509e82d029ff3b87e4c86f
# SHA256 (contract.pdf) = 3494d2947b7929676faeb13ff604d686ad200a2e577130856cc4b63b1324350f
# SHA384 (contract.pdf) = d52ccbc75077bb3ef7da5bda1277cd64b785f6bc6a14e045e5d9cdcda0650cb044d1a21c6f49778ca7924e96162018c9
# SHA512 (contract.pdf) = e53cc32c7c1c3e3d6c1c16fe8299ae129aeb494064fd5ca74b6cec5c6f4d5fa4039cb69c5520f7629cea721dd0cf4af9c90c2eff7a8e29ea3bd88218664f40c7
# SHA3-224 (contract.pdf) = 6735b2899373fc8148e362eceb12c108b4d23a4349c6374f94b8180a
# SHA3-256 (contract.pdf) = 9ec5a5c981913dfaffd6e865f86253eb6383eef62648729e5004be762bc96547
# SHA3-384 (contract.pdf) = 4bbe47402e4c909ea8906dac1995e00044594235dee2219b838bde4d98aa393e0055909b40a34353d8b8ff6ea2793edc
# SHA3-512 (contract.pdf) = a1ae70fab8df208df18556732a53b85f912208372f47f052278bebec17876238a07d976f6d11d557057a24d770dc229badeb17567aab94166655fc9bccda99f4
# BLAKE2s (contract.pdf) = 190ca4941fa7f0d384efb28ccdee68c11cf13e6f84fae640b974767c61f7e271
# BLAKE2b-256 (contract.pdf) = 50b30b076e4b5a998f1ea5feb754551a027671dad45e15a4cf1505e03d72b8a5
# BLAKE2b-384 (contract.pdf) = 3e53843b50f091be5213c0c61f74b4c119263ac69e68129055b8f6c1f9e35ac4eec9a20a5735f7a26e256e2b66f907b7
# BLAKE2b (contract.pdf) = 20ae2d609cf21a2ccfb2885a1a6b7dd1c528bf8c270e133e98c98b516533b1bf7bdf2e37097c8fc36deeacc115475e6d2abb9b24cae3464418800bb3fbdf3245
```

`--list` prints one algorithm per line with no header, so `awk '{print $1}'`
yields exactly the 14 canonical names, in registry order: `md5`, `sha1`,
`sha224`, `sha256`, `sha384`, `sha512`, `sha3-224`, `sha3-256`, `sha3-384`,
`sha3-512`, `blake2s-256`, `blake2b-256`, `blake2b-384`, `blake2b-512`.
`md5` and `sha1` each print a "cryptographically broken" warning to
**stderr** first; redirect with `2>/dev/null` if the loop's output is being
parsed.

### Convert the times a proof's witnesses carry to your local zone

An item proof carries each witness observation's payload as its source
published it, the NIST pulse's `timeStamp` and the Stellar ledger's
`closed_at` among them. These are values the proof declares, never edges:
only an online `verify` reads each from its source and draws the
submitted-after edge from what it read.

```sh
jq -r '.proof.witnesses | .entropy_nist.payload.pulse.timeStamp, .entropy_stellar.payload.closed_at' proof.json \
  | while read -r ts; do
      truestamp convert time "$ts" --to-zone America/New_York
    done
# 2026-09-25T21:48:00-04:00
# 2026-09-25T21:48:30-04:00
```

### Show the recorded item time and block time side-by-side

```sh
printf "item:  "; jq -r .proof.subject.record.id proof.json | truestamp convert id
printf "block: "; jq -r .proof.block.record.id proof.json | truestamp convert id
# item:  2026-09-26T01:49:00Z
# block: 2026-09-26T01:50:00Z
```

The gap between the two is how long the item waited before being committed
into a block; both are times Truestamp recorded on its side, and neither
says anything about when the underlying content was created.

---

## Scripting with `--json` and `jq`

Every inspection command supports `--json`. Combined with `jq`, you can
build sophisticated pipelines with no parsing glue code.

```sh
# The label, and whether the submission window is established. Offline
# against the vectors' keyring; an online run of a sound production proof
# prints verified and established.
truestamp verify proof.json --offline --environment test --keyring keyring.test.json --json | jq -r '.label, .window'
# supplied-keyring
# not_established

# Compute a digest and pipe it into another command's --expected-hash flag.
# --expected-hash is compared against .proof.subject.claims.hash, so it only
# says anything on an external-hash proof, and only for a digest of the
# claims' hash_type.
expected="$(truestamp hash -a sha256 --json contract.pdf | jq -r .digest.hex)"
truestamp verify external-proof.json --expected-hash "$expected"

# Whether the expected hash was checked: the EXPECTED_HASH row passes when
# it matches, fails expected_hash_mismatch when it does not, and is
# skip expected_hash_not_checked without --expected-hash. Its code is
# expected_hash_weak_algorithm on an md5 or sha1 match, as here.
truestamp verify external-proof.json --offline --environment test --keyring keyring.test.json --json --expected-hash 69e3941192b086bfdc0e4e120c18ff12 \
  | jq -r '.rows[] | select(.step=="EXPECTED_HASH") | [.status, .code // empty] | join(" ")'
# warn expected_hash_weak_algorithm

# Every failing row, if any, as its step and code. Shown against
# tampered.json, the vectors' negative/declared-claims_hash.proof.json,
# since a sound proof has no fail row. A failed run ends at its first
# failing check; .code names it.
truestamp verify tampered.json --json --offline --environment test --keyring keyring.test.json | jq -r '.rows[] | select(.status=="fail") | "\(.step) \(.code)"'
# DERIVE derivation_mismatch

# The submitted-after edge the run established, null when it has none; the
# submitted-before edge is .submitted_before. A failed run shows neither.
truestamp verify proof.json --offline --environment test --keyring keyring.test.json --json | jq .submitted_after
# null

# Branch on submission mode at the jq layer
jq 'if .proof.subject.claims | has("hash") then "external-hash mode" else "claims-only mode" end' proof.json
# "external-hash mode"

# Each commitment's chain, network and the epoch hash it commits
jq -r '.proof.commitments | to_entries[] | "\(.key) (\(.value.network)), epoch \(.value.epoch.epoch_hash)"' proof.json
# bitcoin (regtest), epoch 488e6583a192071810724dd392b551c4ca119ca6688deb9034683eb42eba1364
# stellar (testnet), epoch 488e6583a192071810724dd392b551c4ca119ca6688deb9034683eb42eba1364

# The witnesses an item's record commits to, and the witnesses the proof
# carries, which are always the same names
jq '.proof.subject.record.witness_hashes' proof.json
# {
#   "block": "940829eb0fc4717dac15092189394ef52fbbfda555ef5ff0f4c3878fb374a98e",
#   "entropy_bitcoin": "c5e73cea68aaf7999c859639b8b7816f356be9b40d0ff8304b9001377ae040ca",
#   "entropy_nist": "1d2d1432a7ee7221ec781b943d017779b83930b6478f7e58b567e08587fe1482",
#   "entropy_stellar": "2bfcd684973d6e7234cb0d27174cd2b317cf53219ddd8609d6728a286b046dab"
# }
jq -r '.proof.witnesses | keys[]' proof.json
# block
# entropy_bitcoin
# entropy_nist
# entropy_stellar

# A bare rejection is {code, message}, plus sub_reason for malformed_json
# and invalid_envelope, and no report: it has a code
# and no label (exit code 4, or 8 for configuration_error)
truestamp verify not-a-proof.json --offline --json | jq -r 'if has("label") then .label else "rejected: \(.code)" end'
# rejected: invalid_envelope
```

---

## CI / scripting conventions

For automated pipelines, use the following conventions:

```sh
# Silent verification: exit code 0 = verified, and only that; 1 a runtime
# error, 2 a fault, 3 failed, 4 a bare rejection, 5 unattributed, 6
# supplied-keyring, 7 internally-consistent, 8 configuration_error (SPEC.md
# section 12.2). An online run is the only one that can end in 0.
if truestamp verify --silent proof.json; then
  echo "verified"
else
  echo "not verified (exit $?)"
  exit 1
fi

# JSON-structured output for parsers: the report of SPEC.md section 10.5
truestamp verify proof.json --json > verify-report.json
jq -r '.label' verify-report.json

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
| `TRUESTAMP_BASE_URL` | Service origin (scheme + host). API, keyring, console and health URLs all derive from it; it never moves the keyring `verify` reads. |
| `TRUESTAMP_API_KEY` | Auth token for `items` / `proofs get` / `blocks` / `beacons` / `entropy` / `teams` / `console` / `auth status` / `verify --remote`. `keys` and `schema` need no credential. |
| `TRUESTAMP_TEAM` | Multi-tenant team ID |
| `TRUESTAMP_HTTP_TIMEOUT` | HTTP timeout (`30s`, `1m`) |
| `TRUESTAMP_COSIGN_PATH` | Absolute path to the `cosign` binary used by `truestamp upgrade` (empty = `$PATH` lookup) |
| `TRUESTAMP_REQUIRE_COSIGN` | `1` makes a missing `cosign` binary or signature bundle a hard error during an in-place `truestamp upgrade` instead of a warning (SHA-256 is always enforced) |
| `TRUESTAMP_LOGGING_LEVEL` / `TRUESTAMP_LOGGING_FILE` | Log level and log file path (named after the `[logging]` config section, not the flag) |
| `TRUESTAMP_LOGGING_MAX_SIZE_MB` / `TRUESTAMP_LOGGING_MAX_BACKUPS` / `TRUESTAMP_LOGGING_MAX_AGE_DAYS` | Log rotation budget |
| `TRUESTAMP_SILENT` / `TRUESTAMP_JSON` | CLI-wide defaults for `--silent` / `--json` (setting both is an error) |
| `TRUESTAMP_VERIFY_OFFLINE` / `TRUESTAMP_VERIFY_REMOTE` | Defaults for the corresponding `verify` flags |
| `TRUESTAMP_VERIFY_HORIZON` / `TRUESTAMP_VERIFY_ESPLORA` / `TRUESTAMP_VERIFY_BITCOIN_CORE` | The Stellar and Bitcoin sources `verify` reads, as `--horizon`, `--esplora` and `--bitcoin-core` |
| `TRUESTAMP_VERIFY_BITCOIN_CORE_USER` / `TRUESTAMP_VERIFY_BITCOIN_CORE_PASSWORD` | A Bitcoin Core node's RPC credentials, which no flag takes |
| `TRUESTAMP_HASH_ALGORITHM` | Default algorithm for `truestamp hash` |
| `TRUESTAMP_HASH_ENCODING` | Default digest encoding (`hex` / `base64` / `base64url`) |
| `TRUESTAMP_HASH_STYLE` | Default output style (`gnu` / `bsd` / `bare`) |
| `TRUESTAMP_CONVERT_TIME_ZONE` | Default `--to-zone` for `convert time` / `convert id` |
| `TRUESTAMP_NO_UPGRADE_CHECK` | Disable the passive upgrade nag |
| `NO_COLOR` | Industry-standard: strip all ANSI colors |

The verifier's environment and keyless mode are never set by an
environment variable or the configuration file: `--environment` and
`--no-keyring` are flags of the run only. A keyring file comes from
`--keyring`, or, for a non-production environment that `--environment`
names, from `[verify.keyrings]` in `config.toml`.

Truestamp also auto-detects common CI environments and silences the passive
upgrade notice there: `CI`, `GITHUB_ACTIONS`, `GITLAB_CI`, `CIRCLECI`,
`BUILDKITE`, `JENKINS_HOME`, `TF_BUILD`.

---

## Offline / air-gapped use

Everything except the commands that explicitly talk to the Truestamp API
(`items`, `proofs get`, `auth`, `blocks`, `beacons`, `entropy`, `teams`,
`console`, `verify --remote`) works without network. `truestamp keys` is the
one network-bound group that needs no credential: it reads the published
keyring at `{base-url}/.well-known/keyring.json`. `truestamp schema` is
served out of the binary and is unconditionally local.

One command reaches a *different* network service: `truestamp upgrade
--check` always queries the GitHub Releases API (with `--exit-code`, exit 2
on network error), as does the in-place upgrade path; on a Homebrew or `go
install` binary plain `truestamp upgrade` only prints instructions and stays
offline.

```sh
# Fully offline verification: no calls to Truestamp, Stellar, Bitcoin or
# the entropy sources. Pin a keyring file so the key binding check runs
# against it; here, the vectors' test proof against their test keyring.
truestamp verify proof.json --offline --environment test --keyring keyring.test.json | head -3
# Verdict:      supplied-keyring (environment test, not production)
#               The proof is intact and signed by a key in the keyring the operator supplied; this says nothing about Truestamp.
# Attribution:  keyring:e06820401e366a5325b9ec9788440fd4bc63f584e08a0fc7b33b4f2c5123c1fd

# The convert / hash / encode / decode / jcs / inspect primitives compute
# locally and make no network calls, unless you explicitly pass --url,
# which downloads the input (hash, encode, decode, jcs and inspect accept
# it; convert time / id / keyid have no --url and are unconditionally
# offline).
truestamp hash -a sha256 contract.pdf
truestamp jcs < claims.json
truestamp convert id 01a0db66-c2e0-78a4-b8cc-12b11bc97493
# 2026-09-26T01:49:00Z
truestamp inspect proof.json
```

Offline verification is one of the verifier's two modes (SPEC.md section
10.2): every step from the reader and the envelope through the signature,
the schema and every recomputed hash, inclusion proof and link runs with
zero network access. What an offline run cannot do is tie the proof to a
clock outside Truestamp: the Stellar and Bitcoin commitments, each entropy
witness's source and the key event's own commitments are each reported as
`skip offline`, and both edges of the submission window stay not
established. A skipped check is a check the run did not perform, never a
check that failed. Offline, the keyring is the keys built into the CLI, so
the proof's key must be among them: a sound production proof then ends in
`internally-consistent` (exit `7`). When the key is not among them, key
binding fails (exit `3`), and `--keyring` with a downloaded keyring gives
`supplied-keyring` (exit `6`). Only an online run ends in `verified`.

`truestamp verify` without `--offline` makes these outbound requests
(`kb/external-apis.md` lists them in full):

1. Under the production environment, with no `--keyring` and outside keyless mode, it reads the Truestamp keyring at `https://www.truestamp.com/.well-known/keyring.json` to bind the signer's key, or at `{origin}/.well-known/keyring.json` when an explicit `--base-url` names another origin, as a supplied keyring. A failed read binds no key, and never falls back to the keys built into the CLI.
2. Before the file is read, it confirms each Stellar and Bitcoin source's network: Horizon's network passphrase (`horizon.stellar.org` for the public network, `horizon-testnet.stellar.org` for testnet) and the Bitcoin block at height 0 (Blockstream's Esplora for mainnet and testnet, or a Bitcoin Core node you name; regtest has no built-in source). A source that answers for the wrong network is `configuration_error`.
3. For each Stellar and Bitcoin commitment, it reads the transaction, ledger or block the commitment names from those sources, to draw the submitted-before edge.
4. For each entropy subject, and each entropy witness an item proof carries, it reads the observation from its source to compare it with what the proof carries and draw the submitted-after edge: the **NIST Randomness Beacon** (`beacon.nist.gov`) for a NIST pulse, and the Stellar and Bitcoin sources above for a ledger or a block.
5. With `--remote`, it posts the proof's bytes to the Truestamp API's `/api/json/proof/raw/verify`.

`--offline` skips the first four; `README.md` §What gets verified spells
out exactly what it skips and what still runs. Everything local (the
reader, the envelope, the strict Ed25519 signature, the schema, every
recomputed hash, Merkle inclusion proof and link) is always performed.

---

**See also:**
`README.md` for install instructions, `kb/` for architecture and format
reference (start at `kb/README.md`),
`./build/truestamp <command> --help` for per-command flag documentation.
