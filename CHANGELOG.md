# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Version strings in `truestamp upgrade` output now render with a
  consistent style regardless of whether the source is a ldflags-
  injected build version (no `v` prefix) or a GitHub release tag
  (`v` prefix). Previously the "from" and "to" sides of the upgrade
  line — e.g. `0.3.1 → v0.3.2` — could mix prefixes. A new
  `selfupgrade.Display()` helper strips any leading `v` and is used
  everywhere user-facing version strings are printed (check output,
  confirmation prompt, success line, "already at" message, pre-release
  notices, and the passive once-per-day "new version available"
  notice on stderr). The `--version` flag in help text still references
  tags with the `v` prefix to match what users see on the GitHub
  Releases page.

## [0.3.2] — 2026-04-16

### Added
- Retroactive 0.3.1 entry in this CHANGELOG documenting the
  `truestamp upgrade` subcommand that shipped in that release. No
  functional changes in 0.3.2 itself; this release is primarily a
  smoke test of the new in-place upgrade flow introduced in 0.3.1
  (download archive, verify SHA-256, verify cosign bundle, atomic
  replace, darwin quarantine clear).

## [0.3.1] — 2026-04-16

### Added
- `truestamp upgrade` subcommand with install-method detection.
  Homebrew installs print `brew upgrade --cask truestamp/tap/truestamp-cli`,
  `go install` binaries print `go install github.com/truestamp/truestamp-cli/cmd/truestamp@latest`,
  and install.sh / manual installs perform a native-Go in-place
  upgrade that mirrors `docs/install.sh`: download tarball, verify
  SHA-256 (mandatory, pure Go), verify cosign bundle (best-effort via
  shell-out to `cosign` when on `PATH`; required when
  `TRUESTAMP_REQUIRE_COSIGN=1`), extract, atomic replace with
  `.bak.<rfc3339>` backup, clear the macOS quarantine xattr. A
  7-day-old-backup prune runs on every successful upgrade.
- `--check` flag on `upgrade` — report status without installing.
  Exit codes: `0` up-to-date, `1` upgrade available, `2` network
  error, `3` latest release is a pre-release.
- `--yes` and `--version <tag>` flags on `upgrade` for non-interactive
  and pinned upgrades. Passing `--version` is the opt-in path for
  installing pre-release tags; without it, pre-releases are refused.
- Pre-release defense is two-layer: GitHub's `/releases/latest`
  endpoint already filters releases flagged `prerelease: true`, and
  our Go code additionally rejects any resolved tag with a semver
  pre-release suffix (e.g. `v1.0.0-rc.1`) unless `--version` was
  passed explicitly.
- Passive "new version available" notice on stderr, emitted at most
  once per 24 hours (cached at
  `$XDG_CACHE_HOME/truestamp/upgrade-check.json`). Suppressed when
  stderr is not a TTY, when the running binary is a `dev` build, when
  the resolved latest is a pre-release, and under any of seven CI env
  vars (`CI`, `GITHUB_ACTIONS`, `GITLAB_CI`, `CIRCLECI`, `BUILDKITE`,
  `JENKINS_HOME`, `TF_BUILD`).
- `--no-upgrade-check` persistent flag and `TRUESTAMP_NO_UPGRADE_CHECK`
  env var to opt out of the passive notice.
- New `install` line in `truestamp version` output showing the
  detected install method.
- `docs-serve` Taskfile task —
  `mise exec -- caddy file-server --listen :8080 --root docs` —
  for previewing `docs/index.html` and testing `docs/install.sh`
  changes locally before they reach `get.truestamp.com`.
- `caddy` entry in `.tool-versions` so `mise install` bootstraps the
  binary used by `task docs-serve` via the Aqua backend
  (`caddyserver/caddy`).
- New internal packages:
  - `internal/install` — classify the running binary by resolved
    executable path plus `runtime/debug.BuildInfo`. `sameDir`
    resolves symlinks so `/tmp → /private/tmp` and other macOS
    symlinked prefixes classify correctly.
  - `internal/selfupgrade` — orchestrator plus SemVer,
    GitHub Releases client, SHA-256 + cosign verification, tar.gz
    extraction with path-traversal rejection, and Unix/Windows
    atomic-replace implementations.
  - `internal/upgradecheck` — passive check runner with JSON cache
    and all suppression rules.
- `DownloadCtx` and `DownloadBytesCtx` helpers in `internal/httpclient`
  (bounded, context-aware). `DownloadCtx` streams to disk with a
  200 MB default cap.
- Test suite additions: 111 total passing cases across the touched
  packages — path-heuristic coverage for all four install methods, a
  regression test for the `sameDir` symlink bug, httptest-stubbed
  `selfupgrade.Check()` tests locking in both layers of the
  pre-release defense, tar-extraction path-traversal rejection, all
  `Disabled()` suppression rules, cache round-trip, and
  `cmd/upgrade`'s routing + `readYes` + exit-code unwrapping.

### Changed
- `truestamp version` now includes a new `install` line reporting
  the detected install method (`homebrew`, `go install`,
  `install.sh`, or `unknown`).
- Root command's `Execute()` now recognises the internal
  `exitCodeErr` sentinel so `upgrade --check` can exit with a
  specific non-zero code without cobra printing an error line.

### Notes
- Windows is print-only for `upgrade` in this release —
  rename-running-exe-to-`.bak` is deferred to a future minor version.
  Windows users always get `go install …@latest` printed regardless
  of detected method.

## [0.3.0] — 2026-04-15

### Added
- Cosign keyless signing of every release's `checksums.txt`, published
  as `checksums.txt.sigstore` (Sigstore bundle format). Signing identity
  is the release workflow, the OIDC issuer is GitHub Actions, and the
  signing event is logged to the public Rekor transparency log.
- Per-archive SPDX JSON Software Bill of Materials (SBOM) generated by
  syft and attached to each GitHub Release.
- SLSA build-provenance attestation for `checksums.txt` via
  `actions/attest-build-provenance`, queryable with
  `gh attestation verify`.
- `docs/install.sh` now performs best-effort Cosign signature
  verification of `checksums.txt` when `cosign` is on `PATH`, and
  refuses to install when `TRUESTAMP_REQUIRE_COSIGN=1` is set and
  verification fails or the bundle is missing.
- `--help` / `-h` flag on `docs/install.sh` so
  `curl … | sh -s -- --help` prints the env-var reference.
- `docs/install.test.sh` end-to-end installer smoke test, wired into
  CI alongside shellcheck.
- `govulncheck` step in CI catches Go stdlib and dependency
  vulnerabilities on every push.
- `SECURITY.md`, `CONTRIBUTING.md`, and `CODE_OF_CONDUCT.md`.
- README sections for verifying a download, pinning a specific
  version, config-file permissions, and exit codes.
- Package-level godoc on every `internal/*` package.
- `cmd/timestamp_test.go` table-driven tests for the new
  `--timestamp` ISO 8601 validator.
- `cosign`, `shellcheck`, and `syft` to `.tool-versions` so
  `mise install` bootstraps every tool the repo uses.
- `-buildvcs=true` build flag so `go version -m` on the released
  binary reports the source commit.

### Changed
- Homebrew cask updates now land via a pull request to
  `truestamp/homebrew-tap` instead of a direct push to `main`,
  reducing the blast radius of the publish token.
- `--timestamp` values are validated locally and normalised to UTC
  RFC 3339 before being sent to the API, rejecting bad inputs with a
  clear local error.
- Every `uses:` reference in the GitHub Actions workflows is pinned to
  a full commit SHA with a version comment for Dependabot.
- `cmd/verify.go` no longer calls `os.Exit`; failures propagate as
  errors. A silent-mode sentinel keeps `verify --silent` exit-code 1
  without producing any output.
- Duplicate file-picker and table-style helpers in `cmd/` consolidated
  into `internal/ui` (`PickFile`, `LabelValueStyleFunc`).
- HTTP calls now accept a `context.Context` end to end, plumbed from
  each Cobra `cmd.Context()`.
- Contributor-facing content moved out of `README.md` into
  `CONTRIBUTING.md`; the README is now focused on install and use.

### Fixed
- Removed an unreachable init-time `panic` in `internal/proof/binary.go`
  that could only fire for an impossible CBOR option combination.
- Replaced the `":pick"` / `":prompt"` flag sentinels with NUL-byte
  values so that a filename like `:pick` can never collide with the
  picker trigger.

### Security
- All release artifacts now have a cryptographically verifiable chain
  of custody: signed Sigstore bundle → `checksums.txt` SHA-256 →
  platform archives. Tampering anywhere in that chain is detected.
- The `HOMEBREW_TAP_GITHUB_TOKEN` PAT is expected to be fine-grained,
  scoped to `truestamp/homebrew-tap` only, with
  `Contents: Read and write` + `Pull requests: Read and write`. The
  previous classic-PAT guidance has been corrected.

## [0.2.0] — 2026-04-15

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
- SPDX-style `Copyright (c) 2021-2026 Truestamp, Inc.` +
  `SPDX-License-Identifier: MIT` header on all 48 Go source files under
  `cmd/` and `internal/`.
- Copyright footer in `README.md` and the `get.truestamp.com` landing
  page.

### Changed
- Updated `truestamp --help` footer copyright start year from 2019 to
  2021 to match the repository LICENSE file.

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

[Unreleased]: https://github.com/truestamp/truestamp-cli/compare/v0.3.2...HEAD
[0.3.2]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.2
[0.3.1]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.1
[0.3.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.0
[0.2.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.2.0
[0.1.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.1.0
