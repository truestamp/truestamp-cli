# Truestamp CLI

[![CI](https://github.com/truestamp/truestamp-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/truestamp/truestamp-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/truestamp/truestamp-cli)](https://github.com/truestamp/truestamp-cli/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/truestamp/truestamp-cli.svg)](https://pkg.go.dev/github.com/truestamp/truestamp-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Standalone Go CLI for cryptographic timestamping with [Truestamp](https://truestamp.com). Verifies Truestamp proof bundles end to end: user claims, hash chains, Merkle inclusion, Ed25519 signatures, and commitments to public blockchains. Every hash, Merkle walk and signature check is recomputed locally, so a proof can be checked without running or trusting the Truestamp application.

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

The three main commands (`create`, `download`, `verify`) form the full lifecycle of a Truestamp item. Commands that talk to the Truestamp API (`create`, `download`, `beacon`, `team`, `console`, `verify --remote`) need credentials: run `truestamp auth login` for the browser OAuth flow, or set `TRUESTAMP_API_KEY` / `--api-key` for headless and CI use. Without a credential they exit non-zero with a "Not authenticated" hint. Plain `verify` computes locally and needs no credentials at all.

### Create an item

Truestamp supports two submission modes. Pick whichever fits the
shape of the thing you're timestamping.

**External-hash mode**, for files you can keep around. The file
never leaves your device; only its SHA-256 is submitted.

```sh
truestamp create document.pdf
```

Under the hood this computes SHA-256 of the file, uses the filename
as the item name, and registers the hash with the Truestamp API so
it'll be included in the next block.

**Claims-as-source-of-truth mode**, for things that don't have a
file. Written statements, invention disclosures, dated facts,
release notes. The claims content itself is what gets timestamped,
so no file needs to be preserved alongside the proof.

```sh
truestamp create -n "Invention" \
  -d "On this day I claim the following novel approach as my own original work."
```

The server requires the claims content to be meaningful in this
mode: at least a 32-character description (or non-empty
`--metadata`). The CLI checks this locally before any network
round-trip.

Other input styles:

```sh
truestamp create --file document.pdf                     # External hash: explicit file
truestamp create --file                                  # External hash: interactive picker
truestamp create -c claims.json                          # Either mode: claims from JSON file
cat claims.json | truestamp create -C                    # Either mode: claims from stdin
truestamp create -n "Q1 Report" --hash <64-hex> \        # External hash: build from flags
  -v public -t finance,reports
truestamp create -n "Title" --metadata '{"k":"v"}'       # Claims-only: metadata satisfies the rule
```

`--hash` and `--hash-type` travel together in the submitted claims:
both present selects external-hash mode, both absent selects
claims-as-source-of-truth mode. `--hash-type` on its own is rejected
(`claims hash is required when hash_type is supplied`). `--hash` on
its own is accepted: the CLI fills in `hash_type = "sha256"` for you,
which is also the flag's default, and then validates that the hash is
hex of the right length for that algorithm.

JSON output for scripting:

```sh
truestamp create document.pdf --json
```

In claims-as-source-of-truth mode the JSON output omits the `hash`
and `hash_type` keys; scripts can use `jq 'has("hash")'` to branch
on the mode.

### Download a proof bundle

After an item has been committed to a block, download its proof by ID. Item IDs are ULIDs, so a ULID with no `--type` defaults to `--type item`:

```sh
truestamp download 01KNN33GX5E470CB9TRWAYF9DD
```

Pick a format and output path:

```sh
truestamp download -f cbor -o proof.cbor 01KNN33GX5E470CB9TRWAYF9DD
truestamp download -o /tmp/proof.json 01KNN33GX5E470CB9TRWAYF9DD
```

Every other subject type uses a UUIDv7, and entropy observations, blocks and beacons are indistinguishable by id shape, so `--type` is **required** for a UUIDv7. Omitting it exits with an error listing the five valid values rather than guessing:

```sh
truestamp download --type entropy_stellar 019d6a32-13e6-72b0-97e5-3779231ea97b
truestamp download --type block  019db7cd-efc0-7196-b763-682a84d71919
truestamp download --type beacon 019db7cd-efc0-7196-b763-682a84d71919
```

Valid `--type` values are exactly `item | entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon`. There is no `auto` and no bare `entropy`, and the hyphenated spellings (`entropy-nist`) are rejected: flag values use underscores. Generated filenames go the other way and use hyphens, so `--type entropy_nist` writes `truestamp-entropy-nist-<id>.json`.

The id shape is checked before any network call: `--type item` requires a ULID, and every other type requires a UUIDv7.

### Verify a proof

```sh
truestamp verify proof.json
```

Exit code `0` on success, `1` on failure or structural error.

Offline verification (no calls to Truestamp, Stellar, or Bitcoin APIs):

```sh
truestamp verify proof.json --skip-external
```

Silent mode for scripting:

```sh
truestamp verify proof.json --silent && echo valid || echo invalid
```

Other input sources:

```sh
truestamp verify https://example.com/proof.json   # URL
truestamp verify --file                            # Interactive file picker
truestamp verify --url                             # Interactive URL prompt
cat proof.json | truestamp verify                  # stdin pipe
```

## Commands

```
truestamp create [file]              Create a new Truestamp item (submit claims / file hash)
truestamp download <id>              Download a proof bundle (--type required for UUIDv7 ids)
truestamp verify [proof]             Verify a Truestamp proof bundle
truestamp hash [path ...]            Compute cryptographic digests (SHA-2 / SHA-3 / BLAKE2 / MD5 / SHA-1)
truestamp encode [file]              Encode raw bytes into hex / base64 / base64url
truestamp decode [file]              Decode hex / base64 / base64url into raw bytes
truestamp jcs [file]                 Canonicalize JSON per RFC 8785
truestamp convert time [input]       Convert timestamps across zones / Unix formats
truestamp convert proof [file]       Convert a proof bundle between JSON and CBOR
truestamp convert id [value]         Extract the embedded timestamp from a ULID or UUIDv7
truestamp convert keyid [pubkey]     Derive the 4-byte Truestamp kid from an Ed25519 public key
truestamp convert merkle [compact]   Decode a compact base64url Merkle proof
truestamp auth login|logout|status   Manage authentication (browser OAuth by default; --api-key for CI)
truestamp beacon [latest|list|...]   Inspect Truestamp block beacons (bare `beacon` = `beacon latest`)
truestamp team [list|show|set|...]   Manage the active team context (list / show / create / set / unset)
truestamp console                    Interactive TUI over an authenticated WebSocket
truestamp upgrade                    Upgrade the CLI to the latest release (install-method aware)
truestamp config path                Print the config file path
truestamp config show                Print the resolved configuration (API key masked)
truestamp config init                Create a default config file
truestamp version                    Print detailed build and runtime info (includes detected install method)
truestamp --version                  Terse one-line version (also `-v`)
truestamp completion <shell>         Generate shell completions (bash, zsh, fish, powershell)
```

Run `truestamp <command> --help` for per-command flags.

> **[See EXAMPLES.md](./EXAMPLES.md) for an exhaustive per-command tour plus real-world pipeline recipes.** The examples below are a taste.

### Composable pipelines

Every command reads stdin and prints to stdout, and the file-oriented ones (`verify`, `hash`, `encode`, `decode`, `jcs`, `convert proof`) also take `--file` / `--url` with an optional path. So the commands compose as Unix pipes and replace a pile of external tools (`sha256sum`, `shasum`, `xxd`, `base64`, `jq`, `date`):

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

# Round-trip a proof between wire formats and verify end-to-end
truestamp convert proof --to cbor proof.json | truestamp verify --skip-external

# Derive the 4-byte kid fingerprint from an Ed25519 pubkey
truestamp convert keyid CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=

# Timezone math without shelling out to `date`
truestamp convert time 1700000000 --to-zone America/New_York
truestamp convert time "2024-06-15T12:00:00Z" --to-zone Asia/Kolkata

# ULID / UUIDv7 timestamp extraction
truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD
truestamp convert id 019cf813-99b8-730a-84f1-5a711a9c355e --to-zone Local
```

`verify`, `hash`, `encode`, `decode`, `jcs`, every `convert` sub-command, every `beacon` sub-command and `team list` / `team show` / `team create` all support `--json` (structured output for scripting) and `-s` / `--silent` (exit code only). `create` has `--json` but no `--silent`; `auth`, `config`, `download`, `upgrade`, `console` and `version` have neither. `truestamp hash` defaults to GNU `sha256sum`-compatible output, `--style bsd` switches to BSD `shasum --tag` format.

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
truestamp upgrade --yes              # skip the interactive confirmation prompt (also -y)
truestamp upgrade --version vX.Y.Z   # pin to a specific release tag (also the opt-in path for pre-releases)
```

`--check` exit codes: `0` up-to-date, `1` upgrade available, `2` network error, `3` the latest release is a pre-release (will not auto-install; pass `--version <tag>` to install one explicitly).

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

`--base-url` takes an **origin only**: scheme plus host, no path (for example `https://www.truestamp.com`). The API (`/api/json`), keyring (`/.well-known/keyring.json`), console WebSocket (`/console/websocket`) and health (`/health`) URLs are all derived from it, so there is **no `--api-url` and no `--keyring-url`**; passing either is an `unknown flag` error, and the retired `api_url` / `keyring_url` config keys produce a one-time "no longer recognized" warning on stderr.

`cosign_path` pins the `cosign` binary used by `truestamp upgrade` for release-artifact signature verification. Empty (the default) means "use `$PATH` lookup"; set this to an absolute path (e.g. `/opt/cosign/bin/cosign`) in hardened environments to avoid `$PATH` hijacking. Relative paths are rejected at config load. Setting has no effect unless you actually run `truestamp upgrade`.

### Verify-specific flags

| Flag | Env var | Default |
| ---- | ------- | ------- |
| `--file [path]` |   |   |
| `--url [url]` |   |   |
| `--hash` |   |   |
| `--type` |   |   |
| `--remote` | `TRUESTAMP_VERIFY_REMOTE` | `false` |
| `--silent` / `-s` | `TRUESTAMP_VERIFY_SILENT` | `false` |
| `--json` | `TRUESTAMP_VERIFY_JSON` | `false` |
| `--skip-external` | `TRUESTAMP_VERIFY_SKIP_EXTERNAL` | `false` |
| `--skip-signatures` | `TRUESTAMP_VERIFY_SKIP_SIGNATURES` | `false` |

`--type` asserts which subject type you expected (`item | entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon`); a disagreement with the bundle's own signed `t` is reported as a failing `Subject Type` step and exits 1. It has no default and is never inferred: **the filename is never consulted**, so renaming a proof can never change a verdict.

Because every `verify` flag is also a config key, an ambient `TRUESTAMP_VERIFY_SKIP_SIGNATURES` (or `[verify] skip_signatures = true` in `config.toml`) silently weakens a run that still exits 0. Scripts that need a full check should pass the flags explicitly and read the report, not just the exit code.

## What gets verified

`truestamp verify` implements Appendix E of the [Truestamp whitepaper](https://github.com/truestamp/truestamp-v2/blob/main/whitepaper/whitepaper.pdf), which is the normative specification for a conforming verifier. Every run produces a report of graded steps:

1. **Hash Comparison**: does the value you passed to `--hash` match the hash the proof commits to? When the bundle commits to an external file hash and you did not pass `--hash`, this group emits a single warning saying the file hash was not checked. A warning does not fail the proof. Bundles that carry no external hash (claims-as-source-of-truth items, and block-like subjects) produce no Hash Comparison row at all.
2. **Subject Type**: only when `--type` is supplied, does the bundle's own signed `t` agree with what you asserted?
3. **Signing Key**: `pk` decodes to a 32-byte Ed25519 key and yields a 4-byte key id.
4. **Key Binding**: that key id and `pk` are cross-checked against Truestamp's published keyring. This is a separate step from the one above, because parsing a key says nothing about whose key it is. It reads the keyring over the network.
5. **Structure**: the bundle format version is the one this verifier understands.
6. **Subject Data**: subject hash (`0x11` claims / `0x21` entropy) and the composite fingerprint (`0x13` item / `0x23` observation) are recomputed from the bundle's own bytes.
7. **Inclusion Proof**: RFC 6962 Merkle walk from the subject hash to the block's Merkle root.
8. **Block Hash**: `0x32` derivation from the five block fields.
9. **Epoch Proof**: one per commitment, mapping the block hash to the value committed on each public chain.
10. **Proof Signature**: Ed25519 over the fixed-width binary payload.
11. **Submission Window**: the subject's own timestamp precedes the committed block's. This ordering is asserted by Truestamp, not established externally.
12. **Submitted Before** and **Submitted After**: the two edges of the window, always informational. One names the earliest commitment whose on-chain confirmation would ground the submitted-before edge; the other names what a bundle cannot ground on its own.
13. **Temporal Info**: the submitted and committed timeline, rendered for the reader.
14. **Entropy Source**: for entropy subjects only, the captured value is byte-compared against the upstream publisher (NIST beacon, Stellar Horizon, or Blockstream).
15. **Stellar Commitment** and **Bitcoin Commitment**: the on-chain transactions. Bitcoin is verified locally first (OP_RETURN extraction, txid recomputation, partial Merkle tree), then optionally confirmed against Blockstream.

For block (`t=10`) and beacon (`t=11`) bundles the block is its own subject, so there is no subject hash to derive and no leaf to prove: Subject Data, Submission Window, Submitted Before and Submitted After emit no rows at all, Inclusion Proof reports a `skip`, and Temporal Info carries only the committed time.

`--json` renders the same report as structured output. Alongside the summary keys (`result`, `subject_type`, `subject_id`, `signatures_checked`, `subject`, `hash_comparison`, `timeline`, `commitments`, `verification_notes`, `issues`, `summary`) it carries a `steps` array, one object per graded step, each with `group`, `category`, `status` and `message`. That array is the machine-readable form of the list above.

Two rules run through the whole report and are worth knowing before you read one:

- **A step that could not run is a `skip`, never a `fail`.** An unreachable keyring, a Horizon timeout or a chain with no public API establishes nothing either way, and none of them can make a sound proof report as defective.
- **A `skip` never changes the verdict.** Only a step that ran and disagreed fails a proof.

`--skip-external` skips every network step (Key Binding, Entropy Source, and the two commitment confirmations). `--skip-signatures` skips the Ed25519 Proof Signature check and the keyring cross-check; the verdict line then reads `VERIFIED - but the signature was NOT checked (--skip-signatures)`, because a run that did not check the signature has not established who issued the proof. The Signing Key step still runs under both flags, and `--json` reports `"signatures_checked": false`.

## Exit codes

| Code | Meaning |
| ---- | ------- |
| `0`  | Success. For `verify`, the proof is valid. For `upgrade --check`, the CLI is up to date. |
| `1`  | Error. Failed verification, network failure, invalid input, or any other runtime error. For `upgrade --check`, a newer release is available. |
| `2`  | An unrecovered panic (matching Go's own convention, so `[ $? -eq 2 ]` pipelines keep working). For `upgrade --check`, a network error prevented the check. |
| `3`  | For `upgrade --check` only: the latest release is a pre-release and will not auto-install. Pass `--version <tag>` to install one explicitly. |

Usage and flag-parse errors (`unknown flag`, unknown sub-command) exit `1`, not `2`. Scripts that branch on specific codes should check only `upgrade --check`'s documented codes; for other commands, treat any non-zero as failure.

## Contributing

Dev setup, testing, and release process are in [`CONTRIBUTING.md`](./CONTRIBUTING.md). Security issues go through [`SECURITY.md`](./SECURITY.md). Conduct expectations are in [`CODE_OF_CONDUCT.md`](./CODE_OF_CONDUCT.md).

## Related projects

- [`truestamp/truestamp-v2`](https://github.com/truestamp/truestamp-v2): the Truestamp service that generates the proofs this CLI verifies.
- [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap): the Homebrew tap this CLI publishes to.

## License

MIT. See [LICENSE](./LICENSE).

Copyright (c) 2019-2026 [Truestamp, Inc.](https://truestamp.com) All rights reserved.
