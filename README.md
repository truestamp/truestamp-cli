# Truestamp CLI

[![CI](https://github.com/truestamp/truestamp-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/truestamp/truestamp-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/truestamp/truestamp-cli)](https://github.com/truestamp/truestamp-cli/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/truestamp/truestamp-cli.svg)](https://pkg.go.dev/github.com/truestamp/truestamp-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Standalone Go CLI for cryptographic timestamping with [Truestamp](https://truestamp.com). Verifies Truestamp proof bundles end to end — user claims, hash chains, Merkle inclusion, Ed25519 signatures, and public-blockchain commitments — with no dependency on the Truestamp service.

Ships as a single static binary. No runtime required.

## Install

### Install script (macOS, Linux)

```sh
curl -fsSL https://get.truestamp.com/install.sh | sh
```

The script detects your OS/architecture (darwin/linux × amd64/arm64), resolves the latest release, verifies the SHA-256 checksum, installs the binary to `/usr/local/bin` (or `~/.local/bin` if the former isn't writable), and clears the macOS quarantine attribute so the binary runs without a Gatekeeper prompt. Upgrade by re-running the same command.

Pin a specific version:

```sh
curl -fsSL https://get.truestamp.com/install.sh | TRUESTAMP_VERSION=v0.3.0 sh
```

Install to a custom directory:

```sh
curl -fsSL https://get.truestamp.com/install.sh | TRUESTAMP_INSTALL_DIR=~/bin sh
```

Landing page with these same instructions: [get.truestamp.com](https://get.truestamp.com).

### Homebrew (macOS and Linux)

```sh
brew install truestamp/tap/truestamp-cli
```

Upgrades:

```sh
brew upgrade truestamp/tap/truestamp-cli
```

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

Produces a binary at `$GOBIN/truestamp` (default `~/go/bin/truestamp`). Requires Go 1.22 or newer.

The `/cmd/truestamp` suffix is required so the `go` toolchain names the binary `truestamp` rather than `truestamp-cli` (Go derives the binary name from the package path's last element).

### Direct download

Grab the archive for your platform from the [Releases page](https://github.com/truestamp/truestamp-cli/releases/latest):

- `truestamp-cli_<version>_darwin_arm64.tar.gz` — Apple Silicon
- `truestamp-cli_<version>_darwin_amd64.tar.gz` — Intel Mac
- `truestamp-cli_<version>_linux_amd64.tar.gz`
- `truestamp-cli_<version>_linux_arm64.tar.gz`
- `truestamp-cli_<version>_windows_amd64.zip`
- `truestamp-cli_<version>_windows_arm64.zip`

Extract and place `truestamp` somewhere on your `PATH`.

## Verifying a download

Every GitHub Release publishes a `checksums.txt` alongside the archives. To verify a download manually:

```sh
# From the directory containing the downloaded archive and checksums.txt.
sha256sum -c checksums.txt --ignore-missing   # GNU coreutils
# or on macOS without coreutils:
shasum -a 256 -c checksums.txt --ignore-missing
```

The `install.sh` installer and Homebrew cask both verify the SHA-256 automatically — this section is only needed if you downloaded the tarball yourself.

## Quick start

The three main commands — `create`, `download`, `verify` — form the full lifecycle of a Truestamp item. Commands that talk to the Truestamp API (`create`, `download`) need an API key (`--api-key`, `TRUESTAMP_API_KEY`, or the config file). `verify` works entirely locally by default.

### Create an item

Hash a file and submit it in one step:

```sh
truestamp create document.pdf
```

Under the hood this computes SHA-256 of the file, uses the filename as the item name, and registers it with the Truestamp API so it'll be included in the next block.

Other input styles:

```sh
truestamp create --file document.pdf                     # Explicit file path
truestamp create --file                                  # Interactive file picker
truestamp create -c claims.json                          # Claims from a JSON file
cat claims.json | truestamp create -C                    # Claims from stdin
truestamp create -n "Q1 Report" --hash abc123... \       # Build claims from flags
  -v public -t finance,reports
```

JSON output for scripting:

```sh
truestamp create document.pdf --json
```

### Download a proof bundle

After an item has been committed to a block, download its proof by ID. Item IDs are ULIDs; entropy observation IDs are UUIDv7s; the command auto-detects which from the format:

```sh
truestamp download 01KNN33GX5E470CB9TRWAYF9DD
```

Pick a format and output path:

```sh
truestamp download -f cbor -o proof.cbor 01KNN33GX5E470CB9TRWAYF9DD
truestamp download -o /tmp/proof.json 01KNN33GX5E470CB9TRWAYF9DD
```

Download an entropy proof (UUIDv7 triggers entropy proof mode):

```sh
truestamp download 019d6a32-13e6-72b0-97e5-3779231ea97b
```

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
truestamp verify proof.json --output silent && echo valid || echo invalid
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
truestamp create [file]      Create a new Truestamp item (submit claims / file hash)
truestamp download <id>      Download a proof bundle for an item or entropy observation
truestamp verify [proof]     Verify a Truestamp proof bundle
truestamp config path        Print the config file path
truestamp config show        Print the resolved configuration (API key masked)
truestamp config init        Create a default config file
truestamp version            Print detailed build and runtime info
truestamp --version          Terse one-line version
truestamp completion <shell> Generate shell completions (bash, zsh, fish)
```

Run `truestamp <command> --help` for per-command flags.

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
| `--api-url` | `TRUESTAMP_API_URL` | `https://www.truestamp.com/api/json` |
| `--api-key` | `TRUESTAMP_API_KEY` |   |
| `--keyring-url` | `TRUESTAMP_KEYRING_URL` | `https://www.truestamp.com/.well-known/keyring.json` |
| `--http-timeout` | `TRUESTAMP_HTTP_TIMEOUT` | `10s` |
| `--no-color` | `NO_COLOR` | `false` |

### Verify-specific flags

| Flag | Env var | Default |
| ---- | ------- | ------- |
| `--file [path]` |   |   |
| `--url [url]` |   |   |
| `--hash` |   |   |
| `--silent` / `-s` | `TRUESTAMP_VERIFY_SILENT` | `false` |
| `--json` | `TRUESTAMP_VERIFY_JSON` | `false` |
| `--skip-external` | `TRUESTAMP_VERIFY_SKIP_EXTERNAL` | `false` |
| `--skip-signatures` | `TRUESTAMP_VERIFY_SKIP_SIGNATURES` | `false` |

## What gets verified

1. Signing key against the published keyring
2. Proof structure (required fields, block reference)
3. Subject hash — claims hash (`0x11`), timestamp validation, item hash (`0x13`)
4. RFC 6962 Merkle inclusion proof against the block root
5. Block hash (`0x32`) derivation
6. Epoch proofs: block hash → each public-blockchain commitment root
7. Ed25519 proof signature over the binary payload
8. Temporal ordering (item submission before block)
9. Stellar commitment via Horizon API (memo + ledger)
10. Bitcoin commitment via local crypto (OP_RETURN, txid, partial Merkle tree) plus optional Blockstream API

Skipped selectively with `--skip-external` and `--skip-signatures`.

## Exit codes

| Code | Meaning |
| ---- | ------- |
| `0`  | Success. For `verify`, the proof is valid. |
| `1`  | Error. Failed verification, network failure, invalid input, or any other runtime error. |

Exit code `2` is reserved for future use (usage / flag-parse errors). Scripts should branch on `0` vs non-zero rather than matching specific non-zero codes.

## Contributing

Dev setup, testing, and release process are in [`CONTRIBUTING.md`](./CONTRIBUTING.md). Security issues go through [`SECURITY.md`](./SECURITY.md). Conduct expectations are in [`CODE_OF_CONDUCT.md`](./CODE_OF_CONDUCT.md).

## Related projects

- [`truestamp/truestamp-v2`](https://github.com/truestamp/truestamp-v2) — the Truestamp service that generates the proofs this CLI verifies.
- [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap) — the Homebrew tap this CLI publishes to.

## License

MIT. See [LICENSE](./LICENSE).

Copyright (c) 2021-2026 [Truestamp, Inc.](https://truestamp.com) All rights reserved.
