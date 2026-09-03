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
- `cosign_path` (TOML) / `TRUESTAMP_COSIGN_PATH` has **no flag**. Absolute path to the cosign binary used by `truestamp upgrade`; empty = `$PATH` lookup; relative paths are rejected at config load.

There is **no `--api-url` and no `--keyring-url`**: both flags, and the `api_url` / `keyring_url` TOML keys, were retired in favour of the single `base_url`. `config.Load` warns once ("no longer recognized") when it finds the old TOML keys, and cobra rejects the old flags outright with `unknown flag`.

## Verify Flags, the two that carry contracts

`truestamp verify --help` lists all of them. The two with semantics `--help` cannot convey:

- `--type item|entropy_nist|entropy_stellar|entropy_bitcoin|block|beacon` asserts the expected subject type. A mismatch surfaces as a "Subject Type" failure step in **both** modes. `--remote` asserts it client-side against the bytes it is about to post and forwards `data.type` only when the assertion already holds: forwarding a mismatching type makes the server answer 4xx and the caller gets a bare error string with no report at all, which is the opposite of what the flag is for. **There is no default and no inference**: the subject type is always read from the bundle's own signed `t`, and the filename is never consulted (whitepaper Appendix E.24). With `--type` unset, no Subject Type row is emitted and nothing is posted as `data.type`.
- `--remote` delegates verification to the server's `/proof/verify` instead of computing locally, and requires authentication. The CLI still does three things itself: it parses the bundle (so an E.6 rejection never reaches the wire), it performs E.7's `--hash` comparison locally and treats the server's `hash_matched` as corroboration (failing on a disagreement in either direction) and it **fails closed** on a report it cannot read (a step with no `status`, a status outside E.22's five, a `result` carrying zero steps, or a server `passed` verdict that contradicts its own step list).

## Hash subcommand

Flags and the 14-algorithm inventory: `truestamp hash --help` and `truestamp hash --list`.

MD5 and SHA-1 emit a one-line stderr warning when selected, suppressed under `--json` or `--silent`. Algorithm output is byte-identical to `sha256sum`, `md5sum`, and `shasum --tag` for the corresponding vectors; tests in `internal/hashing/hashing_test.go` pin against the canonical NIST FIPS 180-4, FIPS 202, RFC 6234, RFC 7693 vectors.

## Download subcommand

Flags: `truestamp download --help`. `--type` is sent verbatim to the server's `/proof/generate` `type` field; with `--output` unset the filename is `truestamp-<stem>-<id>.<ext>`.

When `--type` is omitted the CLI applies a **client-side smart default** based on the id shape: ULID ids default to `--type item` (the only unambiguous case); UUIDv7 ids fail fast with a helpful error listing the five valid types (`entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon`). There is no `"auto"`: the server's strict-type cutover rejects it, and the CLI follows the same contract.

**Filename stem convention**: wire values use underscores (`entropy_nist`) to match the server enum; filename stems translate `_` → `-` for readable filenames (`truestamp-entropy-nist-<id>.json`). Other types (`item`, `block`, `beacon`) contain no underscores and pass through unchanged.

**Pre-flight id-shape validation**: `--type item` requires a ULID; every other type requires a UUIDv7. Mismatches are caught client-side before the network call with a targeted error instead of a generic server 422.

**Post-download card** emits two public-web hint lines:

- `Details → {host}/<subject-path>/<id>`: the subject's own detail page (`/items/`, `/entropy/`, `/blocks/`). Beacon downloads route to `/blocks/<id>` because the hash-keyed `/beacons/<hash>` form requires computing the block hash from bundle bytes; the beacon listing card uses that form instead since it has the hash directly from the API.
- `Verify → {host}/verify/<type>/<id>`: the typed-sub-path verify landing page from the t=11 cutover. Same URL format the `create` card and the `verify` report emit.

## Encode / Decode / JCS / Convert

Flags: `truestamp <encode|decode|jcs> --help` and `truestamp convert <time|proof|id|keyid|merkle> --help`. `encode`, `decode` and `jcs` all use the same six-mode input convention as `verify`.

Env vars: `TRUESTAMP_CONVERT_TIME_ZONE` sets the default `--to-zone` for `convert time` and `convert id`.

