# Truestamp CLI

[![CI](https://github.com/truestamp/truestamp-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/truestamp/truestamp-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/truestamp/truestamp-cli)](https://github.com/truestamp/truestamp-cli/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/truestamp/truestamp-cli.svg)](https://pkg.go.dev/github.com/truestamp/truestamp-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Standalone Go CLI for cryptographic timestamping with [Truestamp](https://truestamp.com). Verifies Truestamp proof bundles end to end — user claims, hash chains, Merkle inclusion, Ed25519 signatures, and public-blockchain commitments — with no dependency on the Truestamp service.

Ships as a single static binary. No runtime required.

## Install

### Homebrew (macOS and Linux)

```sh
brew install truestamp/tap/truestamp-cli
```

Upgrades:

```sh
brew upgrade truestamp/tap/truestamp-cli
```

### Go install

```sh
go install github.com/truestamp/truestamp-cli@latest
```

Produces a binary at `$GOBIN` (default `~/go/bin/truestamp`). Requires Go 1.22 or newer.

### Direct download

Grab the archive for your platform from the [Releases page](https://github.com/truestamp/truestamp-cli/releases/latest):

- `truestamp-cli_<version>_darwin_arm64.tar.gz` — Apple Silicon
- `truestamp-cli_<version>_darwin_amd64.tar.gz` — Intel Mac
- `truestamp-cli_<version>_linux_amd64.tar.gz`
- `truestamp-cli_<version>_linux_arm64.tar.gz`
- `truestamp-cli_<version>_windows_amd64.zip`
- `truestamp-cli_<version>_windows_arm64.zip`

Extract and place `truestamp` somewhere on your `PATH`. Verify `checksums.txt` with `sha256sum -c` before running.

## Quick start

```sh
truestamp verify proof.json
```

Exit code `0` on success, `1` on failure or structural error.

Offline verification (no network calls to Truestamp, Stellar, or Bitcoin APIs):

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

## Development

This repo uses [mise](https://mise.jdx.dev/) for tool versions and [Task](https://taskfile.dev/) for the developer workflow.

```sh
mise install                 # Installs Go and GoReleaser from .tool-versions
task build                   # Build for current platform → build/truestamp
task test                    # Run all tests
task precommit               # fmt + vet + test + build-all
task release-snapshot        # Local GoReleaser dry-run → dist/
```

The Go module path is `github.com/truestamp/truestamp-cli`; releases are tagged `vX.Y.Z` and drive both the GitHub Release and the Homebrew Cask publication to [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap) via GoReleaser.

## Related projects

- [`truestamp/truestamp-v2`](https://github.com/truestamp/truestamp-v2) — the Truestamp service that generates the proofs this CLI verifies.
- [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap) — the Homebrew tap this CLI publishes to.

## License

MIT. See [LICENSE](./LICENSE).
