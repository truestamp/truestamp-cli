# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Curl-bash installer hosted at `https://get.truestamp.com/install.sh` for
  macOS and Linux, amd64 and arm64. Detects OS/architecture, resolves the
  latest GitHub release (or a pinned tag via `TRUESTAMP_VERSION=vX.Y.Z`),
  verifies SHA-256 against the release `checksums.txt`, installs to
  `/usr/local/bin` or `$HOME/.local/bin`, and clears the macOS quarantine
  attribute so the binary runs without a Gatekeeper dialog.
- Terminal-themed landing page at `https://get.truestamp.com` with plain
  green-on-black monospace install instructions.
- `shellcheck` step in CI to keep `docs/install.sh` portable POSIX sh.

## [0.1.0] — 2026-04-14

### Added
- First release of the Go rewrite of the Truestamp CLI.
- `truestamp verify` — end-to-end proof bundle verification including signing key
  resolution against the published keyring, RFC 6962 Merkle inclusion proof,
  Ed25519 signature verification, Stellar Horizon and Bitcoin Blockstream
  external commitment checks, and temporal ordering.
- `truestamp config path|show|init` for managing the TOML config file at
  `~/.config/truestamp/config.toml`.
- `truestamp version` detailed build/runtime info and `--version` one-liner.
- `truestamp completion` for bash, zsh, and fish shells.
- Config resolution order: defaults → config file → env vars (`TRUESTAMP_*`) → flags.
- Output modes: normal, silent (`-s`), JSON (`--json`), verbose, debug.
- Selective skip flags: `--skip-external`, `--skip-signatures`.
- Input modes: positional path or URL, `--file`, `--url`, interactive pickers, stdin pipe.
- Distribution channels: Homebrew (`brew install truestamp/tap/truestamp-cli`),
  `go install github.com/truestamp/truestamp-cli@latest`, and direct binary
  downloads from GitHub Releases for darwin/linux/windows × amd64/arm64.

### Removed
- The prior TypeScript/Deno CLI that previously occupied this repository at
  versions through v1.1.0. Those releases and tags have been removed. This
  v0.1.0 is the first release of a standalone Go codebase; the two share
  nothing beyond the repository name.

[Unreleased]: https://github.com/truestamp/truestamp-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.1.0
