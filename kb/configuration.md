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

- `--base-url` / `TRUESTAMP_BASE_URL` (default `https://www.truestamp.com`) is the **origin only** (scheme + host). Every service URL is **derived** from it during `config.Load`: API `/api/json`, keyring `/.well-known/keyring.json`, console WebSocket `/console/websocket`, health `/health`. Paths, queries and fragments are stripped, so only the origin survives. Since the API key, an OAuth token and the console WebSocket all go to it, `config.Load` accepts only `https`, or plain `http` to a loopback host (`localhost`, a name under `.localhost`, `127.0.0.0/8`, `::1`), and refuses any other scheme and plain http to any other host, exit 1 before any request is built. A bad `base_url` in the file stops every command that loads the configuration, `config show` and `config edit` included; `--base-url https://...` or `TRUESTAMP_BASE_URL` overrides the file for that run.
- `--api-key` / `TRUESTAMP_API_KEY` is the CI/headless fallback, not the default path. Supplying it explicitly (flag or env var) overrides any stored OAuth session; a config-file `api_key` ranks *below* the OAuth session. Prefer `truestamp auth login`.
- `--log-level` / `--log-file` take their env var names from the TOML **section** (`[logging] level` → `TRUESTAMP_LOGGING_LEVEL`), not from the flag name. The level must be `debug`, `info`, `warn` (or `warning`) or `error`, in any case, from whichever source; `config.Load` refuses anything else with exit 1, as it refuses a bad `http_timeout`. `--log-level info` overrides a bad value in the file, `config edit` included.
- `--json` / `-s, --silent` are **not root persistent flags**: each record-rendering command registers the pair itself (`cmd/output.go`), so they go after the command name. `TRUESTAMP_JSON` / `TRUESTAMP_SILENT` and the top-level `json` / `silent` config keys set them CLI-wide; they are mutually exclusive, enforced once in `config.Load`. The pipeline primitives (`encode`, `decode`, `jcs`, `convert *`, `hash`) carry the flags but read them directly, so the ambient setting cannot silence a pipe (kb/command-tree.md R10).
- `cosign_path` (TOML) / `TRUESTAMP_COSIGN_PATH` has **no flag**. Absolute path to the cosign binary used by `truestamp upgrade`; empty = `$PATH` lookup; relative paths are rejected at config load.

There is **no `--api-url` and no `--keyring-url`**, and no `api_url` / `keyring_url` TOML key: the single `base_url` is the origin the API commands derive every URL from, so no second setting can point one of them elsewhere. It never moves verify's keyring read, which is read from another origin only when the run is given an explicit `--base-url` (below). `config.Load` warns once ("no longer recognized") when it finds either TOML key, and cobra rejects either flag with `unknown flag`.

Any other key the file sets that no setting reads (a renamed or retired one such as `verify.keyring`, or a typo) is not refused but reported: `config.Load` returns it in `Config.Warnings`, built from the struct's `koanf` tags (every table and leaf, and anything under the `verify.sources` map), and `cmd/root.go` prints `warning: <file>: unknown config key "<key>" has no effect` on stderr unless the run is silent. Case is compared as Unmarshal compares it (`reachesSetting`): it reads a key spelled exactly as the lowercase tag and folds case only when there is none, and the defaults give nearly every leaf a lowercase key, so `BASE_URL`, or `OFFLINE` under `[VERIFY]`, is dropped and reported; an upper-case spelling with no lowercase one beside it (such as `[verify.SOURCES.stellar.pubnet]`) is read and not reported. The legacy warning reads only `api_url` and `keyring_url` spelled so, and any other spelling of them is reported here. Only the file is checked: environment variables share the `TRUESTAMP_` prefix with variables that are not settings, and cobra refuses an unknown flag itself. The legacy warning above prints even under `--silent`; this one does not.

## Verify options, and where each may come from

`truestamp verify --help` lists them. Where each may come from is the contract (SPEC.md sections 6.4, 8.3 and 9; `internal/verifier/options.go`):

- **The run's options only**: `--environment` (E_v, `prod` by default), `--no-keyring` (keyless mode), `--keyring FILE` and `--trace FILE`. No configuration key and no environment variable sets any of them; `verify.keyring` and `TRUESTAMP_VERIFY_KEYRING` are retired and set nothing (the file key draws the unknown-key warning above).
- **The configuration file only**: `[verify.keyrings]` with `dev`, `staging` and `test`, a keyring file for each non-production environment, used only when `--environment` names it; and `[verify.sources.<chain>.<network>]`, a network's several sources in order with a `policy` of `first`, `at_least N` or `all`, each source a table `{ horizon = URL }`, `{ esplora = URL }` or `{ bitcoin_core = URL }`. Load drops every environment variable that would reach either table.
- **A flag, the environment or the file**: `--horizon`, `--esplora`, `--bitcoin-core` (`TRUESTAMP_VERIFY_HORIZON`, `_ESPLORA`, `_BITCOIN_CORE`; `verify.horizon`, `verify.esplora`, `verify.bitcoin_core`), a flag over a variable over the file; and `--offline` / `TRUESTAMP_VERIFY_OFFLINE` / `verify.offline`, `--remote` / `TRUESTAMP_VERIFY_REMOTE` / `verify.remote`.
- **The environment or the file, never the command line**: a Bitcoin Core node's RPC credentials, `verify.bitcoin_core_user` / `verify.bitcoin_core_password` and `TRUESTAMP_VERIFY_BITCOIN_CORE_USER` / `_PASSWORD`. No flag takes them, and a source URL with user information is refused from every origin with an error that never quotes it. `config show` never shows them, and shows everything before a source URL's last `@`, but its scheme, as `[masked]`, since any `@` after the `//` could end user information.
- **Nothing**: no option sets a network pin; the pins are E_v's defaults (SPEC.md section 9). No option selects a NIST source.

`config init` creates `config.toml` readable by its owner alone (0600) in a directory only its owner can enter (0700), since it can hold an API key and a node's password.

The keyring that governs, for a run: `--no-keyring`, keyless; else `--keyring FILE`; else, online, the keyring read from the origin an explicit `--base-url` names, when that host is not www.truestamp.com; else, under a non-production E_v, its configured keyring; else, under `prod`, the keyring read from www.truestamp.com online and the compiled pin offline, and under any other E_v `configuration_error`. The configured base URL (`base_url`, `TRUESTAMP_BASE_URL`) points the API commands and never moves the keyring read.

`--expected-hash` is read by its Changed state: `--expected-hash ""` is a value given, and fails EXPECTED_HASH. The value is lowercased and never trimmed (SPEC.md section 10.2, step 13).

`--remote` also posts the file's bytes to the service's `POST /api/json/proof/raw/verify` and prints the label of its report on stderr; stdout, `--json` and the exit code are the local run's. The request waits for the larger of `--http-timeout` and 75 seconds.

## Hash subcommand

Flags and the 14-algorithm inventory: `truestamp hash --help` and `truestamp hash --list` (with `--json`, the same `{"algorithms": [...]}` document as `truestamp schema get algorithms --json`).

MD5 and SHA-1 emit a one-line stderr warning when selected, suppressed under `--json` or `--silent`. Algorithm output is byte-identical to `sha256sum`, `md5sum`, and `shasum --tag` for the corresponding vectors; tests in `internal/hashing/hashing_test.go` pin against the canonical NIST FIPS 180-4, FIPS 202, RFC 6234, RFC 7693 vectors.

## proofs get

Flags: `truestamp proofs get --help`. The proof is read from `GET /api/json/proof/raw/{type}/{ref}?constructions={full|reference|none}`, `--type` in the path, and its bytes are written exactly as served: to stdout unless `-o/--out <path>` names a file or `--to-file` picks the conventional name, `<CODE>.proof.json` for an item (by its item code, whatever the constructions mode) and `<uuidv7>.proof.json` for every other subject, as the app names its downloads. A block's proof and its beacon's proof share one id and so one name, and the later write replaces the earlier (`os.WriteFile`); `-o` keeps both. The two flags are mutually exclusive and the receipt card goes to stderr (kb/command-tree.md R10).

**What names a subject.** Nothing is trimmed: a UUIDv7 or an `item_hash` is read in either case, and an item code as SPEC.md section 3.9 lets it be typed (any case, `I` and `L` read as `1` and `O` as `0`, and one space, hyphen or underscore allowed between its groups of four) and is sent in its one spelling; a UUID's 32-hex, braced and `urn:` forms are refused before any request. An item code (16 symbols of SPEC.md section 3.9) is an item, with no round trip; a 64-hex value is an item's `item_hash` and needs `--type item` (without it, it is refused with that hint); a UUIDv7 with no `--type` is resolved against `POST /api/json/utilities/resolve-id` in one extra round trip, and one verifiable as more than one subject type, a block's id (a block is also verifiable as a beacon), is refused as `block | beacon` rather than guessed. Passing `--type` skips the round trip. `--type item` takes a UUIDv7, a code or an `item_hash`; every other type takes a UUIDv7, and a code or a hash with it is refused before the request. There is no `"auto"` and no bare `entropy`: flag values are exactly the six types, with underscores.

**The served item is checked before anything is written**: for an item proof asked for by its UUIDv7, the record it carries must have that id; by its code, the `item_hash` recomputed from its record must give that code; by its `item_hash`, it must be that whole value. Otherwise the run refuses with "the server returned a proof for a different item" and writes nothing (`internal/proof.CheckServedItem`). A matching code shows only that the recomputed `item_hash` begins with the 75 bits the code is written from.

**Receipt card** (on stderr when a file is written) gives the file, its size, for an item the recomputed `item_hash` with its item code, the type, the constructions mode, and two public-web lines:

- `Details`: the subject's own detail page, `/items/<CODE>` for an item and `/entropy/<id>` or `/blocks/<id>` otherwise. Beacon proofs route to `/blocks/<id>`, because the hash-keyed `/beacons/<hash>` form needs the block hash; the beacon listing card uses that form, since it has the hash from the API.
- `Verify`: the typed verify page, `/verify/<type>/<CODE>` for an item and `/verify/<type>/<id>` otherwise, as the `items create` card gives it.

## items get and items update

Each takes an item by its UUIDv7 id, its item code or its `item_hash`, read as `proofs get` reads it. `items get` reads `GET /api/json/items/{ref}` by whichever is given, and the item the server answers with must carry the code or the whole `item_hash` given. `items update` sends `PATCH /api/json/items/{id}` by the UUIDv7 only: a code or an `item_hash` is first read with one GET, held to the same check, and the update is sent by the id that item gives.

## Encode / Decode / JCS / Convert

Flags: `truestamp <encode|decode|jcs> --help` and `truestamp convert <time|id|keyid> --help`. `encode`, `decode` and `jcs` all use the same six-mode input convention as `verify`.

Env vars: `TRUESTAMP_CONVERT_TIME_ZONE` sets the default `--to-zone` for `convert time` and `convert id`.

