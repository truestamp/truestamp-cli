<!--
Copyright (c) 2019-2026 Truestamp, Inc.
SPDX-License-Identifier: Apache-2.0
-->

# Truestamp CLI

[![CI](https://github.com/truestamp/truestamp-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/truestamp/truestamp-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/truestamp/truestamp-cli)](https://github.com/truestamp/truestamp-cli/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/truestamp/truestamp-cli.svg)](https://pkg.go.dev/github.com/truestamp/truestamp-cli)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](./LICENSE)

Standalone Go CLI for cryptographic timestamping with [Truestamp](https://www.truestamp.com). Verifies Truestamp proof bundles end to end: user claims, hash chains, Merkle inclusion, Ed25519 signatures, and commitments to public blockchains. Every hash, Merkle walk and signature check is recomputed locally, so a proof can be checked without running or trusting the Truestamp application.

Ships as a single self-contained binary. Release builds are pure Go with cgo disabled, so there is no interpreter or language runtime to install.

## Documentation

- **[EXAMPLES.md](./EXAMPLES.md)**: hands-on tour of every sub-command with real, copy-pastable examples. Includes pipeline recipes, `--json` / `jq` patterns, CI conventions, and offline / air-gapped usage. **Start here to see what the CLI can do.**
- **[CONTRIBUTING.md](./CONTRIBUTING.md)**: development setup, test categories, and task reference.
- **[CHANGELOG.md](./CHANGELOG.md)**: release notes.
- Per-command help: `truestamp <command> --help`.

## Install

### Install script (macOS, Linux)

```sh
curl -fsSL https://get.truestamp.com/install.sh | sh
```

The script detects your OS/architecture (darwin/linux x amd64/arm64), resolves the latest release, verifies the SHA-256 checksum, installs the binary to `/usr/local/bin` (or `~/.local/bin` if the former isn't writable), and clears the macOS quarantine attribute so the binary runs without a Gatekeeper prompt. It also verifies the keyless cosign signature over `checksums.txt` when `cosign` is on `$PATH`, and skips that step silently when it is not. To upgrade later, run `truestamp upgrade`; it matches the install method, and for install-script users it downloads the new release, verifies SHA-256 plus cosign, and atomically replaces the binary in place. Re-running the curl pipeline also works.

Pin a specific version:

```sh
curl -fsSL https://get.truestamp.com/install.sh | TRUESTAMP_VERSION=vX.Y.Z sh
```

Install to a custom directory:

```sh
curl -fsSL https://get.truestamp.com/install.sh | TRUESTAMP_INSTALL_DIR=~/bin sh
```

Refuse to install unless the cosign signature verifies:

```sh
curl -fsSL https://get.truestamp.com/install.sh | TRUESTAMP_REQUIRE_COSIGN=1 sh
```

Landing page with these same instructions: [get.truestamp.com](https://get.truestamp.com). The script itself lives at [`docs/install.sh`](./docs/install.sh) in this repo.

### Homebrew (macOS and Linux)

The CLI is published as a Homebrew **cask** (not a formula) to [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap):

```sh
brew install --cask truestamp/tap/truestamp-cli
```

Upgrades:

```sh
brew upgrade --cask truestamp/tap/truestamp-cli
```

This is the exact command `truestamp upgrade` prints when it detects a Homebrew install.

> **macOS Gatekeeper note.** The binary is not yet signed with an Apple Developer ID, so the first time you run `truestamp` after a `brew install` or `brew upgrade` macOS will show a dialog titled _"truestamp" Not Opened_ and kill the process. Clear the quarantine attribute once per install to avoid it:
>
> ```sh
> xattr -cr "$(brew --caskroom)/truestamp-cli"
> ```
>
> The same instruction is printed by `brew` as a caveat on install. Signed and notarized builds are on the roadmap; once they ship this step will not be needed.

### Go install

```sh
go install github.com/truestamp/truestamp-cli/cmd/truestamp@latest
```

Produces a binary at `$GOBIN/truestamp` (default `~/go/bin/truestamp`). Requires the Go toolchain named by the `go` directive in [`go.mod`](./go.mod), or newer; that file is the authority on the exact minimum.

The `/cmd/truestamp` suffix is required so the `go` toolchain names the binary `truestamp` rather than `truestamp-cli` (Go derives the binary name from the package path's last element).

### Direct download

Grab the archive for your platform from the [Releases page](https://github.com/truestamp/truestamp-cli/releases/latest):

- `truestamp-cli_<version>_darwin_arm64.tar.gz` (Apple Silicon)
- `truestamp-cli_<version>_darwin_amd64.tar.gz` (Intel Mac)
- `truestamp-cli_<version>_linux_amd64.tar.gz`
- `truestamp-cli_<version>_linux_arm64.tar.gz`
- `truestamp-cli_<version>_windows_amd64.zip`
- `truestamp-cli_<version>_windows_arm64.zip`

`<version>` carries no leading `v` in the archive name even though the release tag does. Extract and place `truestamp` somewhere on your `PATH`.

## Verifying a download

Every GitHub Release publishes a `checksums.txt` alongside the archives. To verify a download manually:

```sh
# From the directory containing the downloaded archive and checksums.txt.
sha256sum -c checksums.txt --ignore-missing   # GNU coreutils
# or on macOS without coreutils:
shasum -a 256 -c checksums.txt --ignore-missing
```

Releases also publish `checksums.txt.sigstore`, a keyless cosign bundle over `checksums.txt`. If you have `cosign` installed you can check that the checksum list itself is authentic:

```sh
cosign verify-blob \
  --bundle checksums.txt.sigstore \
  --certificate-identity-regexp '^https://github\.com/truestamp/truestamp-cli/\.github/workflows/release\.yml@' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

The `install.sh` installer and the Homebrew cask both verify the SHA-256 automatically, so this section is only needed if you downloaded the tarball yourself.

## Quick start

The three main commands (`items create`, `proofs get`, `verify`) form the full lifecycle of a Truestamp item. Commands that talk to the Truestamp API (`items`, `proofs get`, `blocks`, `beacons`, `entropy`, `teams`, `console`, `verify --remote`) need credentials: run `truestamp auth login` for the browser OAuth flow, or set `TRUESTAMP_API_KEY` / `--api-key` for headless and CI use. Without a credential they exit non-zero with a "Not authenticated" hint. Plain `verify` computes locally and needs no credentials at all.

### Create an item

Truestamp supports two submission modes. Pick whichever fits the
shape of the thing you're timestamping.

**External-hash mode**, for files you can keep around. The file
never leaves your device; only its SHA-256 is submitted.

```sh
truestamp items create document.pdf
```

Under the hood this computes SHA-256 of the file, uses the filename
as the item name, and registers the hash with the Truestamp API so
it'll be included in the next block.

**Claims-as-source-of-truth mode**, for things that don't have a
file. Written statements, invention disclosures, dated facts,
release notes. The claims content itself is what gets timestamped,
so no file needs to be preserved alongside the proof.

```sh
truestamp items create -n "Invention" \
  -d "On this day I claim the following novel approach as my own original work."
```

The server requires the claims content to be meaningful in this
mode: at least a 32-character description (or non-empty
`--metadata`). The CLI checks this locally before any network
round-trip.

Other input styles:

```sh
truestamp items create --file=document.pdf                     # External hash: explicit file
truestamp items create --file                                  # External hash: interactive picker
truestamp items create -c=claims.json                          # Either mode: claims from JSON file
cat claims.json | truestamp items create -C                    # Either mode: claims from stdin
truestamp items create -n "Q1 Report" --data-hash <64-hex> \        # External hash: build from flags
  -v public -t finance,reports
truestamp items create -n "Title" --metadata '{"k":"v"}'       # Claims-only: metadata satisfies the rule
```

`--data-hash` and `--hash-type` travel together in the submitted claims:
both present selects external-hash mode, both absent selects
claims-as-source-of-truth mode. `--hash-type` on its own is rejected
(`claims hash is required when hash_type is supplied`). `--data-hash` on
its own is accepted: the CLI fills in `hash_type = "sha256"` for you,
which is also the flag's default, and then validates that the hash is
hex of the right length for that algorithm.

JSON output for scripting:

```sh
truestamp items create document.pdf --json
```

In claims-as-source-of-truth mode the JSON output omits the `hash`
and `hash_type` keys; scripts can use `jq 'has("hash")'` to branch
on the mode.

### Download a proof bundle

After an item has been committed to a block and to a public chain, download its proof by ID. Item IDs are ULIDs, so a ULID with no `--type` defaults to `--type item`.

The bundle goes to **stdout** by default, so it composes:

```sh
truestamp proofs get 01KNN33GX5E470CB9TRWAYF9DD | truestamp verify --offline
truestamp proofs get 01KNN33GX5E470CB9TRWAYF9DD > proof.json
```

Or name a path with `-o` / `--out`, or let `--to-file` pick the conventional name in the current directory. The two are mutually exclusive, and the receipt card is written to stderr so stdout stays pipeable:

```sh
truestamp proofs get -o /tmp/proof.json 01KNN33GX5E470CB9TRWAYF9DD
truestamp proofs get --to-file 01KNN33GX5E470CB9TRWAYF9DD          # <CODE>.proof.json
```

Choose how the proof carries the constructions text (SPEC.md section 7.2): `full`, the default, carries the text itself, `reference` its URL, which names its SHA-256, and `none` neither. Each mode is its own signed document, the verdict never depends on it, and `--to-file` names them alike.

```sh
truestamp proofs get --constructions reference --to-file 01KNN33GX5E470CB9TRWAYF9DD   # <CODE>.proof.json
```

Every other subject type uses a UUIDv7, and entropy observations, blocks and beacons are indistinguishable by id shape, so `--type` cannot be inferred from a UUIDv7's shape. Omitting it costs one extra round trip: the CLI asks the server what the id refers to. Passing `--type` skips that call, and is required when an id is verifiable as more than one subject type (a block is also verifiable as a beacon):

```sh
truestamp proofs get --type entropy_stellar 019d6a32-13e6-72b0-97e5-3779231ea97b
truestamp proofs get --type block  019db7cd-efc0-7196-b763-682a84d71919
truestamp proofs get --type beacon 019db7cd-efc0-7196-b763-682a84d71919
```

Valid `--type` values are exactly `item | entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon`. There is no `auto` and no bare `entropy`, and the hyphenated spellings (`entropy-nist`) are rejected: flag values use underscores. `--to-file` names an item's proof `<CODE>.proof.json`, by its item code, and every other subject's `<id>.proof.json`. The `type` inside the file is authoritative; the filename never is.

The id shape is checked before any network call: `--type item` requires a ULID, and every other type requires a UUIDv7.

### Verify a proof

```sh
truestamp verify proof.json
```

A proof is one JSON document. The run ends in a report labelled `verified`, `internally-consistent`, `supplied-keyring`, `unattributed` or `failed`, or in a bare rejection with no report, and exits with the code of [Exit codes](#exit-codes): `0` only for `verified`.

Offline verification (no calls to Truestamp, Stellar, Bitcoin or the entropy sources; every cryptographic check still runs, and every check that needs a read is reported as skipped, never failed):

```sh
truestamp verify proof.json --offline
```

Compare the hash of a file you hold with the hash the item's claims carry, and check the signer's key against a keyring file you supply:

```sh
curl -sO https://www.truestamp.com/.well-known/keyring.json
truestamp verify proof.json --offline --keyring keyring.json \
  --expected-hash "$(truestamp hash -a sha256 --style bare contract.pdf)"
```

A proof from Truestamp's staging environment is checked with `--environment staging` and a staging keyring: `--keyring FILE`, or one configured under `[verify.keyrings]`. `--no-keyring` checks the signature only against the key the proof carries, and the report says so.

```sh
truestamp verify staging-proof.json --offline --environment staging --keyring staging-keyring.json
```

Silent mode for scripting:

```sh
truestamp verify proof.json --silent && echo verified || echo "not verified (exit $?)"
```

Other input sources:

```sh
truestamp verify https://example.com/proof.json   # URL
truestamp verify --file                            # Interactive file picker
truestamp verify --url                             # Interactive URL prompt
cat proof.json | truestamp verify                  # stdin pipe
```

See what a bundle carries without verifying it:

```sh
truestamp inspect proof.json
truestamp inspect proof.cbor --json
```

## Commands

```
Verify a proof:
  truestamp verify [proof]              Verify a proof (one JSON document), offline or online
  truestamp inspect [proof]             Print what a bundle carries, without verifying it

Truestamp resources:
  truestamp items create [file]         Create a timestamped item (claims, or a hash of a local file)
  truestamp items list                  List items, newest first (--limit/--after/--before/--oldest-first/--max/--count, --committed/--pending)
  truestamp items get <id>              Show one item, including whether a proof can be generated
  truestamp items update <id>           Change visibility, tags, or the owning team (--to-team)
  truestamp proofs get <id>             Fetch a proof bundle to stdout (-o <path> or --to-file to write it)
  truestamp proofs convert [file]       Convert a bundle between JSON and CBOR
  truestamp blocks list|get|latest|genesis    Read-only: the block chain (latest = head block)
  truestamp beacons list|get|latest     Read-only: public randomness over finalized blocks
  truestamp entropy list|get|latest     Read-only: public entropy observations (NIST, Stellar, Bitcoin)
  truestamp keys list|get|current       Read-only: the published signing keyring (no credential needed)
  truestamp teams list|get|current|create|use  List, create, and switch teams

Local tools:
  truestamp hash [path ...]             Compute digests (SHA-2 / SHA-3 / BLAKE2 / MD5 / SHA-1)
  truestamp encode [file]               Encode raw bytes into hex / base64 / base64url
  truestamp decode [file]               Decode hex / base64 / base64url into raw bytes
  truestamp jcs [file]                  Canonicalize JSON per RFC 8785
  truestamp convert time [input]        Convert timestamps across zones / Unix formats
  truestamp convert id [value]          Extract the embedded timestamp from a ULID or UUIDv7
  truestamp convert keyid [pubkey]      Derive the 4-byte Truestamp kid from an Ed25519 public key
  truestamp convert merkle [compact]    Decode a compact base64url Merkle proof

Setup:
  truestamp auth login|logout|status    Manage authentication (browser OAuth; --api-key for CI)
  truestamp config path|show|init|edit  Inspect, create and edit the config file

Other:
  truestamp console                     Interactive TUI over an authenticated WebSocket
  truestamp schema list|get <name>      Machine-readable registries, including the whole command tree
  truestamp upgrade                     Upgrade the CLI in place (install-method aware)
  truestamp version                     Detailed build and runtime info
  truestamp completion <shell>          Shell completions (bash, zsh, fish, powershell)

A bare group prints help: `truestamp items` tells you what you can do with items,
and works before you have signed in.

Cross-cutting reference lives where it is owned: the vocabulary in the Truestamp whitepaper (not yet published; this README will link it when it is), the output contract and exit codes in this file, and machine-readable registries in `truestamp schema`. Each command's own `--help` carries the constraints that apply to it.

Run `truestamp <command> --help` for per-command flags.

> **[See EXAMPLES.md](./EXAMPLES.md) for an exhaustive per-command tour plus real-world pipeline recipes.** The examples below are a taste.

### Composable pipelines

Every command reads stdin and prints to stdout, and the file-oriented ones (`verify`, `inspect`, `hash`, `encode`, `decode`, `jcs`, `proofs convert`) also take `--file` / `--url` with an optional path. So the commands compose as Unix pipes and replace a pile of external tools (`sha256sum`, `shasum`, `xxd`, `base64`, `jq`, `date`):

```sh
# SHA-256 a file, byte-identical to sha256sum / shasum output
truestamp hash doc.pdf

# Pick a different algorithm (14 supported; see `truestamp hash --list`)
truestamp hash -a blake2b-512 doc.pdf
truestamp hash -a sha3-256 --style bsd doc.pdf

# Recompute a Truestamp claims_hash locally: the flagship use case
truestamp hash --prefix 0x11 --jcs -a sha256 --style bare --no-filename < claims.json
# equivalently, as an explicit pipeline:
truestamp jcs < claims.json | truestamp hash --prefix 0x11 -a sha256 --style bare --no-filename

# verify reads JSON only, and proofs convert reads only the earlier proof
# layout, so no CBOR proof verifies until the CLI reads the new format

# Derive the 4-byte kid fingerprint from an Ed25519 pubkey
truestamp convert keyid CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=

# Timezone math without shelling out to `date`
truestamp convert time 1700000000 --to-zone America/New_York
truestamp convert time "2024-06-15T12:00:00Z" --to-zone Asia/Kolkata

# ULID / UUIDv7 timestamp extraction
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD
truestamp convert id 019cf813-99b8-730a-84f1-5a711a9c355e --to-zone Local
```

Every list (`items list`, `blocks list`, `beacons list`, `entropy list`) pages the same way: `--limit` is the page size (a page above 250 is clamped, and the listing says so); `--after <cursor>` and `--before <cursor>` continue from a cursor a previous page printed, forward or back toward the start (`More: --after …` and `Back: --before …` on **stderr**, printed only when stdout is a terminal, so a pipe or a redirect carries rows and nothing else; `next_cursor` and `prev_cursor` in `--json`); `--oldest-first` starts at the beginning instead of the newest row; `--max N` follows cursors in the chosen direction until N rows have been fetched; and `--count` adds the server's total. There is no unbounded `--all`: the tables behind these lists grow by the minute, so following pages costs a cap you wrote down. Their `--json` is one envelope, `{"<noun>": [...], "next_cursor": "...", "prev_cursor": "..."}`, plus `"total"` under `--count` and `"page_limit"` whenever the server used a smaller page than `--limit` asked for (the human listing mentions the clamp only when a page follows; the JSON key reports the page size used, whether or not one does).

`--json` (structured output for scripting) and `-s` / `--silent` (exit code only) are **CLI-wide, mutually exclusive settings**. Every command that renders a *record* carries them, including `auth status`, `config show`, `config path` and `version`. They can be set once via `config.toml`, `TRUESTAMP_JSON` or `TRUESTAMP_SILENT`.

The exemptions are deliberate: the pipeline primitives (`encode`, `decode`, `jcs`, `convert time|id|keyid|merkle`, `hash`) still carry `--json` and `--silent` as explicit flags, but ignore the ambient config-file and environment setting. They print one bare value or raw bytes, and `truestamp hash` defaults to GNU `sha256sum`-compatible output with `--style bsd` for BSD `shasum --tag` format, so a `silent = true` in `config.toml` must not be able to silence a pipe built on them. `proofs get` emits a payload rather than a record and follows the stdout / `-o` / `--to-file` triad instead.

**More examples:** [EXAMPLES.md](./EXAMPLES.md) covers every sub-command with copy-pastable recipes, scripting patterns, CI conventions, and offline usage.

## Upgrading

The `truestamp upgrade` command is install-method aware: it detects how the binary was installed (Homebrew, `go install`, or install.sh / manual tarball) and does the right thing for each:

| Install method | `truestamp upgrade` behavior |
| -------------- | ---------------------------- |
| Homebrew       | Prints `brew upgrade --cask truestamp/tap/truestamp-cli` (does not touch the Homebrew prefix). |
| `go install`   | Prints `go install github.com/truestamp/truestamp-cli/cmd/truestamp@latest`. |
| install.sh / manual | Downloads the latest release tarball, verifies SHA-256 (mandatory, pure Go) and cosign signature (best-effort; required if `TRUESTAMP_REQUIRE_COSIGN=1`; `cosign` is located on `$PATH` by default, or pin an absolute path with `cosign_path` in config or `TRUESTAMP_COSIGN_PATH` env var to defend against `$PATH` hijacking), extracts the binary, atomically replaces the running executable, and clears the macOS quarantine xattr. A `.bak.<timestamp>` backup of the previous binary is kept for 7 days. |
| Unknown, not writable | Prints `curl -fsSL https://get.truestamp.com/install.sh \| sh`. |
| Windows (any method) | Prints `go install ...@latest`. In-place upgrade is not supported on Windows in this release. |

An unknown install location that *is* writable falls through to the same in-place upgrade as install.sh.

Check the detected install method at any time:

```sh
truestamp version           # output includes an `install` row naming the method
```

Flags:

```sh
truestamp upgrade --check            # only report whether an upgrade is available (does not install)
truestamp upgrade --check --exit-code  # ... and encode the answer in the exit status
truestamp upgrade --yes              # skip the interactive confirmation prompt (also -y)
truestamp upgrade --version vX.Y.Z   # pin to a specific release tag (also the opt-in path for pre-releases)
truestamp upgrade --no-verify        # skip the cosign signature check even when cosign is installed (SHA-256 still enforced)
```

`--check` always exits `0`, whatever it finds, so a check in a script never fails the script. Add `--exit-code` to encode the answer in the status instead; the codes are listed under [Exit codes](#exit-codes). A pre-release latest never auto-installs; pass `--version <tag>` to install one explicitly.

### Passive upgrade notices

Once every 24 hours (cached at `$XDG_CACHE_HOME/truestamp/upgrade-check.json`, defaulting to `~/.cache/truestamp/upgrade-check.json`, and `%LOCALAPPDATA%` on Windows), other commands print a one-line note on stderr if a newer release is available. The notice is automatically suppressed in CI environments (`CI`, `GITHUB_ACTIONS`, `GITLAB_CI`, `CIRCLECI`, `BUILDKITE`, `JENKINS_HOME`, `TF_BUILD`), when stderr is not a TTY, when the current version is a local `dev` build, and when the resolved latest is a pre-release. To opt out:

```sh
truestamp --no-upgrade-check verify proof.json
# or persistently:
export TRUESTAMP_NO_UPGRADE_CHECK=1
```

The notice is always on stderr, so it never pollutes stdout (`truestamp verify proof.json > out.json` is safe for scripting).

## Configuration

Settings are resolved in this order (later overrides earlier):

1. Compiled defaults
2. Config file (`~/.config/truestamp/config.toml` by default)
3. Environment variables (`TRUESTAMP_*`)
4. CLI flags

> The config file may contain an API key. It is stored in plaintext, so restrict permissions on a shared machine:
>
> ```sh
> chmod 600 ~/.config/truestamp/config.toml
> ```

### Global flags

| Flag | Env var | Default |
| ---- | ------- | ------- |
| `--config` |   | `~/.config/truestamp/config.toml` |
| `--base-url` | `TRUESTAMP_BASE_URL` | `https://www.truestamp.com` |
| `--api-key` | `TRUESTAMP_API_KEY` |   |
| `--team` | `TRUESTAMP_TEAM` |   |
| `--http-timeout` | `TRUESTAMP_HTTP_TIMEOUT` | `10s` |
| `--log-level` | `TRUESTAMP_LOGGING_LEVEL` | `info` |
| `--log-file` | `TRUESTAMP_LOGGING_FILE` | `<user cache dir>/truestamp/truestamp.log` |
| `--no-color` | `NO_COLOR` | `false` |
| `--no-upgrade-check` | `TRUESTAMP_NO_UPGRADE_CHECK` | `false` |
| (config file / env only: `cosign_path`) | `TRUESTAMP_COSIGN_PATH` |   |

`--json` and `-s` / `--silent` are not root flags: every record-rendering command registers the pair itself, so they go after the command name. `TRUESTAMP_JSON` / `TRUESTAMP_SILENT` (default `false`) and the top-level `json` / `silent` config keys set them CLI-wide.

`--base-url` takes an **origin only**: scheme plus host, no path (for example `https://www.truestamp.com`). The API (`/api/json`), keyring (`/.well-known/keyring.json`), console WebSocket (`/console/websocket`) and health (`/health`) URLs are all derived from it, so there is **no `--api-url` and no `--keyring-url`**; passing either is an `unknown flag` error, and the retired `api_url` / `keyring_url` config keys produce a one-time "no longer recognized" warning on stderr.

`cosign_path` pins the `cosign` binary used by `truestamp upgrade` for release-artifact signature verification. Empty (the default) means "use `$PATH` lookup"; set this to an absolute path (e.g. `/opt/cosign/bin/cosign`) in hardened environments to avoid `$PATH` hijacking. Relative paths are rejected at config load. Setting has no effect unless you actually run `truestamp upgrade`.

### Verify-specific flags

| Flag | Env var | Config key | Default |
| ---- | ------- | ---------- | ------- |
| `--file [path]` |   |   |   |
| `--url [url]` |   |   |   |
| `--expected-hash` |   |   | none |
| `--keyring FILE` |   |   | none |
| `--environment` |   |   | `prod` |
| `--no-keyring` |   |   | `false` |
| `--trace FILE` |   |   | none |
| `--horizon` | `TRUESTAMP_VERIFY_HORIZON` | `verify.horizon` | the built-in Horizon |
| `--esplora` | `TRUESTAMP_VERIFY_ESPLORA` | `verify.esplora` | the built-in Esplora |
| `--bitcoin-core` | `TRUESTAMP_VERIFY_BITCOIN_CORE` | `verify.bitcoin_core` | none |
|   | `TRUESTAMP_VERIFY_BITCOIN_CORE_USER` | `verify.bitcoin_core_user` | none |
|   | `TRUESTAMP_VERIFY_BITCOIN_CORE_PASSWORD` | `verify.bitcoin_core_password` | none |
|   |   | `[verify.keyrings]` `dev`, `staging`, `test` | none |
|   |   | `[verify.sources.<chain>.<network>]` | none |
| `--remote` | `TRUESTAMP_VERIFY_REMOTE` | `verify.remote` | `false` |
| `--offline` | `TRUESTAMP_VERIFY_OFFLINE` | `verify.offline` | `false` |

`--expected-hash` is the hash of a file you hold, compared with the proof's `claims.hash` (SPEC.md section 10.2, step 13). Given, even empty, it is a value: a proof with no `claims.hash` of a listed type, a value that is not hex of the type's length, and a hash that differs each fail the row. Without it the row is skipped.

The keyring that governs the signer's key is chosen for the run only, never by the configuration file or an environment variable: `--keyring FILE`, a keyring file you supply; `--no-keyring`, keyless mode, which checks the signature only against the key the proof carries and says so in every report; and otherwise, online, the keyring read from www.truestamp.com, and offline the keys built into this CLI. `--environment` (`prod`, `dev`, `staging` or `test`) names the Truestamp environment whose proofs the run checks, again for the run only. A run under another environment than `prod` needs `--keyring FILE`, `--no-keyring`, or a keyring configured for that environment under `[verify.keyrings]` in `config.toml`, which is used only when `--environment` names it.

`--horizon`, `--esplora` and `--bitcoin-core` read Stellar and Bitcoin from servers you name, each in place of the built-in source of the network its identity read shows; regtest has no built-in source. A flag overrides the environment, which overrides the file. A Bitcoin Core node's RPC credentials come from the configuration file or the environment only, never the command line, and a source URL that carries a user name or password is refused. Several sources for one network, in an order and under a policy (`first`, `at_least N` or `all`), are set in `config.toml` alone:

```toml
[verify.sources.bitcoin.mainnet]
policy = "at_least 2"
sources = [
  { esplora = "https://blockstream.info/api" },
  { bitcoin_core = "http://127.0.0.1:8332" },
]
```

`--trace FILE` writes the run's trace (SPEC.md section 10.6) to FILE: one row for each recomputed value, its declared value and what it was compared with. No option selects a NIST source or a network pin, and the filename is never consulted.

Because `--offline` and `--remote` are also config keys, an ambient `TRUESTAMP_VERIFY_OFFLINE` (or `[verify] offline = true` in `config.toml`) makes every run offline, and an offline run is never `verified` (exit `7` at best). A script that needs a full check reads the exit code, which is `0` only for `verified`.

## What gets verified

`truestamp verify` is an external verifier of SPEC.md in [truestamp/truestamp-verification](https://github.com/truestamp/truestamp-verification), and runs its steps in order (SPEC.md section 10.2):

- Step 0, **CONFIGURATION**: the environment the run checks, the network pins, the sources and the keyring that governs, before the file is read. Online, each Stellar and Bitcoin source's network is confirmed first: a source that answers for the wrong network is `configuration_error`.
- Steps 1 to 3, **READ**, **PARSE** and **ENVELOPE**: the file is at most 262,144 octets, is one strict JSON text, and is exactly the envelope `{proof, signature}`. A failure at these steps is a bare rejection: `{code, message, sub_reason}` and no report.
- Steps 4 to 6, **ENVIRONMENT**, **KEY_BINDING** and **SIGNATURE**: the environment marker against the run's environment, the signer's key against the keyring, and the strict Ed25519 signature over `proof_hash`. A failure here ends the run.
- Steps 7 to 9, **SHAPE**, **POLICY** and **DERIVE**: the schema, every network against the pins, and every hash, inclusion proof and link recomputed from the values the proof carries.
- Steps 10 to 12, **CHAIN**, **SOURCES** and **WINDOW**: online, each Stellar and Bitcoin commitment and each entropy observation read from its source, and the submitted-after and submitted-before edges drawn from what was read. Offline, each is a skip.
- Step 13, **EXPECTED_HASH**: `--expected-hash` against `claims.hash`.
- Step 14, **VERDICT**: the label.

Each check gives one row, `pass`, `fail`, `skip`, `warn` or `info`, with a reason code. A skip is a check this run did not make, never a failure: an offline run, an unreachable source or history a source no longer holds leaves an edge absent and fails nothing. The text output names the label, the attribution, the trust root, the environment (beside the verdict when it is not `prod`), keyless mode in words, each edge with the source it was read from in this run, and every row that does not pass. No window is shown for a `failed` run. `--json` writes the report exactly as SPEC.md section 10.5 gives it (`attribution`, `code`, `environment`, `label`, `rows`, `subject_id`, `submitted_after`, `submitted_before`, `trust_root`, `type`, `verified`, `window`), or the bare rejection.

`--remote` also asks the Truestamp service to verify the proof and prints its verdict on stderr, beside the report and never as a row of it; the exit code is this run's own.

## Rate limits

The Truestamp API refuses an over-limit request with HTTP `429` and an error whose code is `rate_limited`. The refusal names how long to wait: a `Retry-After` header (whole seconds) on refusals from the per-surface request-rate limit, or `meta.retry_after_ms` on refusals raised inside an action (an item submission over the per-user rate, for example). The CLI waits that long, plus up to two seconds of random jitter, and repeats the request once, saying why it has gone quiet when stderr is a terminal. The jitter is deliberate: the per-address guard described below is a window aligned to the clock minute, so every client it refuses in the same minute is told the same second, and one that retried on exactly that second would arrive at the boundary together with all the others. The per-caller budget is a token bucket, so its `Retry-After` is your own short wait for the next token, and the jitter costs at most two seconds there. A second refusal, a wait of more than a minute, or a `retry_after_ms` of `null` (the request is over the limit on its own, and no wait admits it) is reported as `rate limited (retry after Ns, limit L): <server detail>`, and the command exits `1`. The OAuth token and revocation endpoints get the same one retry, keyed on the `429` status and `Retry-After`; a rate-limited token refresh is reported as a rate limit and never as an expired session, so wait rather than signing in again.

The limits themselves are server configuration and can change without a CLI release. At the time of writing: the per-caller budget for the JSON:API is a token bucket of 120 requests refilling at 120 per minute; each API surface also admits at most 1,000 requests per minute per client address, checked before any credential is examined, so a flood of missing or bad credentials is refused with the same `429`; the OAuth endpoints allow 60 per minute per IP; item submission is limited to 1,000 per minute per user; proof generation (`proofs get`) to 300 per minute per account; and proof verification (`verify --remote`) to 30 per minute per account, with a fleet-wide ceiling of 120 per minute on verifications that contact the public chains (`--offline` skips that ceiling). The limit a refusal names (`meta.limit`) is the one that applied.

## Exit codes

| Code | Meaning |
| ---- | ------- |
| `0`  | Success. |
| `1`  | Error. Network failure, invalid input, or any other runtime error. |
| `2`  | An unrecovered panic (matching Go's own convention, so `[ $? -eq 2 ]` pipelines keep working). |

`verify` exits with the codes of SPEC.md section 12.2, which `--silent` carries too:

| Code | The run |
| ---- | ------- |
| `0`  | label `verified` |
| `1`  | a runtime error outside verification, such as an invalid command line or an input that cannot be read; no report |
| `2`  | the verifier stopped on a fault it did not catch, or caught one before steps 0 to 3 passed; no report |
| `3`  | label `failed` |
| `4`  | a bare rejection other than `configuration_error` |
| `5`  | label `unattributed` |
| `6`  | label `supplied-keyring` |
| `7`  | label `internally-consistent` |
| `8`  | `configuration_error` |

Only `0` says that Truestamp signed the proof and that its submission window is established. A script that needs both checks for `0` alone.

`upgrade --check` exits `0` whatever it finds; the answer is the line it prints. Pass `--exit-code` to encode the result in the status instead: `1` an upgrade is available, `2` a network error prevented the check, `3` the latest release is a pre-release and will not auto-install.

Usage and flag-parse errors (`unknown flag`, unknown sub-command) exit `1`, not `2`. Scripts that branch on specific codes should check only `upgrade --check`'s documented codes; for other commands, treat any non-zero as failure.

## Contributing

Dev setup, testing, and release process are in [`CONTRIBUTING.md`](./CONTRIBUTING.md). Security issues go through [`SECURITY.md`](./SECURITY.md). Conduct expectations are in [`CODE_OF_CONDUCT.md`](./CODE_OF_CONDUCT.md).

## Related projects

- [Truestamp](https://www.truestamp.com): the service that generates the proofs this CLI verifies. Its API reference is at [www.truestamp.com/api/json/redoc](https://www.truestamp.com/api/json/redoc), with a plain-text edition for LLMs at [www.truestamp.com/llms-full.txt](https://www.truestamp.com/llms-full.txt).
- [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap): the Homebrew tap this CLI publishes to.

## License

Apache License 2.0. See [LICENSE](./LICENSE).

Copyright (c) 2019-2026 [Truestamp, Inc.](https://www.truestamp.com)
