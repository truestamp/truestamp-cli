<!--
Copyright (c) 2019-2026 Truestamp, Inc.
SPDX-License-Identifier: Apache-2.0
-->

# Configuration

Settings are resolved in priority order (highest priority last):
1. Compiled defaults
2. Config file (`~/.config/truestamp/config.toml` or `$XDG_CONFIG_HOME/truestamp/config.toml`)
3. Environment variables (`TRUESTAMP_` prefix)
4. CLI flags (only explicitly set flags override)

Authentication is OAuth-first: the access/refresh token pair lives in the **OS keychain** (0600-file fallback), *not* in config.toml. The config-file `api_key` and the explicit `--api-key` / `TRUESTAMP_API_KEY` are the CI/headless path only; an explicit key (env or flag, tracked as `config.Config.APIKeyExplicit`, computed in `config.Load`) overrides a stored OAuth session, while a config-file key is consulted only after the OAuth session. See [authentication.md](authentication.md) for the full precedence.

## Global Flags, the non-obvious ones

Run `truestamp --help` for the full persistent-flag list. What `--help` does not say:

- `--base-url` / `TRUESTAMP_BASE_URL` (default `https://www.truestamp.com`) is the **origin only** (scheme + host). Every service URL is **derived** from it during `config.Load`: API `/api/json`, keyring `/.well-known/keyring.json`, console WebSocket `/console/websocket`, health `/health`. Paths, queries and fragments are stripped, so only the origin survives.
- `--api-key` / `TRUESTAMP_API_KEY` is the CI/headless fallback, not the default path. Supplying it explicitly (flag or env var) overrides any stored OAuth session; a config-file `api_key` ranks *below* the OAuth session. Prefer `truestamp auth login`.
- `--log-level` / `--log-file` take their env var names from the TOML **section** (`[logging] level` → `TRUESTAMP_LOGGING_LEVEL`), not from the flag name.
- `--json` / `-s, --silent` are **not root persistent flags**: each record-rendering command registers the pair itself (`cmd/output.go`), so they go after the command name. `TRUESTAMP_JSON` / `TRUESTAMP_SILENT` and the top-level `json` / `silent` config keys set them CLI-wide; they are mutually exclusive, enforced once in `config.Load`. The pipeline primitives (`encode`, `decode`, `jcs`, `convert *`, `hash`) carry the flags but read them directly, so the ambient setting cannot silence a pipe (kb/command-tree.md R10).
- `cosign_path` (TOML) / `TRUESTAMP_COSIGN_PATH` has **no flag**. Absolute path to the cosign binary used by `truestamp upgrade`; empty = `$PATH` lookup; relative paths are rejected at config load.

There is **no `--api-url` and no `--keyring-url`**: both flags, and the `api_url` / `keyring_url` TOML keys, were retired in favour of the single `base_url`. `config.Load` warns once ("no longer recognized") when it finds the old TOML keys, and cobra rejects the old flags outright with `unknown flag`.

## Verify options, and where each may come from

`truestamp verify --help` lists them. Where each may come from is the contract (SPEC.md sections 6.4, 8.3 and 9; `internal/verifier/options.go`):

- **The run's options only**: `--environment` (E_v, `prod` by default), `--no-keyring` (keyless mode), `--keyring FILE` and `--trace FILE`. No configuration key and no environment variable sets any of them; `verify.keyring` and `TRUESTAMP_VERIFY_KEYRING` are retired and set nothing.
- **The configuration file only**: `[verify.keyrings]` with `dev`, `staging` and `test`, a keyring file for each non-production environment, used only when `--environment` names it; and `[verify.sources.<chain>.<network>]`, a network's several sources in order with a `policy` of `first`, `at_least N` or `all`, each source a table `{ horizon = URL }`, `{ esplora = URL }` or `{ bitcoin_core = URL }`. Load drops every environment variable that would reach either table.
- **A flag, the environment or the file**: `--horizon`, `--esplora`, `--bitcoin-core` (`TRUESTAMP_VERIFY_HORIZON`, `_ESPLORA`, `_BITCOIN_CORE`; `verify.horizon`, `verify.esplora`, `verify.bitcoin_core`), a flag over a variable over the file; and `--offline` / `TRUESTAMP_VERIFY_OFFLINE` / `verify.offline`, `--remote` / `TRUESTAMP_VERIFY_REMOTE` / `verify.remote`.
- **The environment or the file, never the command line**: a Bitcoin Core node's RPC credentials, `verify.bitcoin_core_user` / `verify.bitcoin_core_password` and `TRUESTAMP_VERIFY_BITCOIN_CORE_USER` / `_PASSWORD`. No flag takes them, and a source URL with user information is refused from every origin with an error that never quotes it. `config show` never shows them, and shows everything before a source URL's last `@`, but its scheme, as `[masked]`, since any `@` after the `//` could end user information.
- **Nothing**: no option sets a network pin; the pins are E_v's defaults (SPEC.md section 9). No option selects a NIST source.

`config init` creates `config.toml` readable by its owner alone (0600) in a directory only its owner can enter (0700), since it can hold an API key and a node's password.

The keyring that governs, for a run: `--no-keyring`, keyless; else `--keyring FILE`; else, online, the keyring read from the origin an explicit `--base-url` names, when that host is not www.truestamp.com; else, under a non-production E_v, its configured keyring; else, under `prod`, the keyring read from www.truestamp.com online and the compiled pin offline, and under any other E_v `configuration_error`. The configured base URL (`base_url`, `TRUESTAMP_BASE_URL`) points the API commands and never moves the keyring read.

`--expected-hash` is read by its Changed state: `--expected-hash ""` is a value given, and fails EXPECTED_HASH. The value is lowercased and never trimmed (SPEC.md section 10.2, step 13).

`--remote` also asks the server's `/proof/verify` and prints its verdict on stderr; stdout and the exit code are the local run's.

## Hash subcommand

Flags and the 14-algorithm inventory: `truestamp hash --help` and `truestamp hash --list`.

MD5 and SHA-1 emit a one-line stderr warning when selected, suppressed under `--json` or `--silent`. Algorithm output is byte-identical to `sha256sum`, `md5sum`, and `shasum --tag` for the corresponding vectors; tests in `internal/hashing/hashing_test.go` pin against the canonical NIST FIPS 180-4, FIPS 202, RFC 6234, RFC 7693 vectors.

## proofs get

Flags: `truestamp proofs get --help`. `--type` is sent verbatim to the server's `/proof/generate` `type` field. The bundle goes to stdout unless `-o/--out <path>` names a file or `--to-file` picks the conventional `truestamp-<stem>-<id>.<ext>` name; the two are mutually exclusive and the receipt card goes to stderr (kb/command-tree.md R10).

When `--type` is omitted: a ULID id is an item (the only unambiguous shape); a UUIDv7 id is resolved against `POST /utilities/resolve-id` in one extra round trip, and an id verifiable as more than one subject type (a block is also verifiable as a beacon) is refused with the candidates listed rather than guessed. Passing `--type` skips the round trip. There is no `"auto"`: the server's strict-type cutover rejects it, and the CLI follows the same contract.

**Filename stem convention**: wire values use underscores (`entropy_nist`) to match the server enum; filename stems translate `_` → `-` for readable filenames (`truestamp-entropy-nist-<id>.json`). Other types (`item`, `block`, `beacon`) contain no underscores and pass through unchanged.

**Pre-flight id-shape validation**: `--type item` requires a ULID; every other type requires a UUIDv7. Mismatches are caught client-side before the network call with a targeted error instead of a server refusal.

**Receipt card** (on stderr when a file is written) emits two public-web hint lines:

- `Details → {host}/<subject-path>/<id>`: the subject's own detail page (`/items/`, `/entropy/`, `/blocks/`). Beacon proofs route to `/blocks/<id>` because the hash-keyed `/beacons/<hash>` form requires computing the block hash from bundle bytes; the beacon listing card uses that form instead since it has the hash directly from the API.
- `Verify → {host}/verify/<type>/<id>`: the typed-sub-path verify landing page from the t=11 cutover. Same URL format the `items create` card and the `verify` report emit.

## Encode / Decode / JCS / Convert

Flags: `truestamp <encode|decode|jcs> --help` and `truestamp convert <time|id|keyid> --help`. `encode`, `decode` and `jcs` all use the same six-mode input convention as `verify`.

Env vars: `TRUESTAMP_CONVERT_TIME_ZONE` sets the default `--to-zone` for `convert time` and `convert id`.

