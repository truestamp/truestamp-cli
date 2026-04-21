# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] — 2026-04-21

### Added
- Five new pipe-friendly sub-commands that replace the ad-hoc external
  tool chain (`sha256sum`, `shasum`, `xxd`, `base64`, `jq`, `date`)
  with built-ins that behave identically across macOS, Linux, BSD,
  WSL, and Git-Bash on Windows:
  - `truestamp hash` — SHA-2 / SHA-3 / BLAKE2 / MD5 / SHA-1 digests
    for files, stdin, URLs, or a picked file. Default output is
    byte-identical to GNU coreutils' `sha256sum` (including the
    standard `\`-escaping for filenames with newlines/backslashes).
    `--style bsd` matches `shasum --tag`; `--style bare` is
    digest-only. Legacy algorithms (MD5, SHA-1) emit a one-line
    stderr warning unless `--json` or `--silent` is set. Supports
    `--prefix 0xNN` (prepend a single domain-separation byte before
    hashing) and `--jcs` (apply RFC 8785 canonicalization first), so
    the Truestamp claims-hash intermediate is a one-liner:
    `truestamp hash --prefix 0x11 --jcs -a sha256 --style bare < claims.json`.
  - `truestamp encode` and `truestamp decode` — byte-encoding
    conversions among `hex`, `base64`, `base64url`, and `binary`,
    with strict cross-encoding-alphabet rejection. Both support
    text-to-text conversion via `--from`/`--to`.
  - `truestamp jcs` — apply RFC 8785 JSON Canonicalization to the
    input. Pipes directly into `truestamp hash` for hashing
    pipelines; `truestamp hash --jcs` is the shortcut.
  - `truestamp convert` — umbrella for domain-specific conversions:
    - `convert time` — parse and re-format timestamps across zones
      and Unix formats (auto / rfc3339 / unix-{s,ms,us,ns}, IANA
      zones, Go time layouts as `--format`).
    - `convert proof` — switch a proof bundle between JSON and CBOR
      wire formats. CBOR output uses RFC 8949 §4.2 core
      deterministic encoding and prepends the self-describing tag
      55799 so `truestamp verify` auto-detects the format.
    - `convert id` — extract the embedded millisecond timestamp
      from a ULID (item IDs) or UUIDv7 (block and entropy IDs).
      Auto-detects the type; supports `--to-zone` for display.
    - `convert keyid` — derive the 4-byte Truestamp `kid`
      fingerprint (`truncate4(SHA256(0x51 || pubkey))`) from an
      Ed25519 public key in hex, base64, or base64url.
    - `convert merkle` — decode a compact base64url Merkle proof
      (`ip` / `ep` fields) into a human-readable sibling list with
      position + hash per step.
- `internal/inputsrc` — shared six-mode input resolver (positional,
  `--file [path]`, `--file` picker, `--url [url]`, `--url` prompt,
  stdin pipe). Used uniformly by `verify`, `create`, and every new
  sub-command. `pflag` `NoOptDefVal` sentinels are now readable
  `(pick)` / `(prompt)` strings so `--help` renders cleanly; `-`
  as a positional is accepted as the Unix-standard stdin alias.
- `internal/encoding` — RFC 4648 hex/base64/base64url round-trip with
  tolerance for trailing whitespace and rejection of mismatched
  alphabets.
- `internal/hashing` — 14-algorithm registry built on `crypto/{md5,
  sha1,sha256,sha512}` + `golang.org/x/crypto/{sha3,blake2b,blake2s}`,
  streaming `Compute`, and `sha256sum` / `shasum --tag` output
  formatters with proper filename escaping.
- `ProofBundle.MarshalCBOR` on `internal/proof` — deterministic CBOR
  encoding (RFC 8949 §4.2) with the 0xd9d9f7 self-describing tag.
  Byte-valued fields (`pk`, `sig`, hashes, epoch/inclusion proofs)
  are re-encoded as CBOR major-type-2 byte strings per an explicit
  per-field policy; round-trips stabilize on the second pass.
- `[hash]` and `[convert]` config sections in
  `~/.config/truestamp/config.toml`, plus matching env-var overrides
  (`TRUESTAMP_HASH_ALGORITHM`, `TRUESTAMP_HASH_ENCODING`,
  `TRUESTAMP_HASH_STYLE`, `TRUESTAMP_CONVERT_TIME_ZONE`).
- Comprehensive test infrastructure:
  - **59 fuzz targets** (`FuzzXxx`) across 13 packages covering every
    parser that touches attacker-controlled bytes. Seed corpus lives
    in `f.Add()` calls; `go test` replays them on every run. Active
    mutation via `task fuzz-deep` (default 15s per target, 59
    targets, ~320M inputs per full pass).
  - **20+ benchmarks** (`BenchmarkXxx`) on hot paths: hashing across
    all 14 algorithms (SHA-256 ~2.9 GB/s, BLAKE2b ~1.1 GB/s on M3
    Max), encoding round-trip, proof parse / marshal (JSON + CBOR),
    Merkle decode + verify, domain-prefixed hashing.
  - **9 golden-output snapshot tests** — `testdata/golden/` fixtures
    for `--help` output on root/verify/hash, `hash --list`, and JSON
    envelopes for `hash`, `encode`, and `convert {time,id,keyid}`.
    JSON is canonicalized before diffing so tests don't flake on
    Go's map-iteration order. Regenerate with
    `UPDATE_GOLDEN=1 go test ./cmd -run Golden`.
  - Coverage raised from 47.6% to 81.2% across 17 packages. Most
    packages are now >90%; the handful below that ceiling have
    structural reasons (interactive TTY, platform-specific Windows
    branches on a macOS/Linux runner, self-upgrade pipeline that
    needs a real release tarball + cosign binary).
- New Taskfile entry points:
  - `task test-coverage-full` — covers CLI subprocess runs too by
    building the binary with `-cover -coverpkg=./...` and routing
    its `GOCOVERDIR` through a task-controlled directory (works
    around `go test -cover` clobbering `GOCOVERDIR` in the test
    process's environment).
  - `task test-race` — full suite under the race detector; currently
    zero-finding.
  - `task bench` / `task bench-compare` — benchmarks with `-benchmem`
    and a `-count=5` baseline suitable for `benchstat` comparison.
  - `task fuzz` / `task fuzz-deep` / `task fuzz-list`.
  - `task lint` — `go vet` + `gofmt -l` + `staticcheck` + `gosec`
    with documented exclusions for CLI-standard patterns
    (user-specified file paths, supported-with-warning legacy
    hashes, Unix-standard file permissions, hard-coded subprocess
    names).
  - `task vuln-check` — `govulncheck` against `go.mod` and stdlib.
  - `task precommit-full` — strict pre-release gate (fmt + lint +
    test-race + fuzz seeds + vuln-check + build-all, ~3-5 min).
- `EXAMPLES.md` — new, ~800 lines. Per-sub-command tour with
  copy-pastable examples, ~15 pipeline recipes, `--json` + `jq`
  scripting patterns, CI conventions, and an offline/air-gapped
  usage section. Every example was exercised end-to-end against a
  live dev server to catch documentation drift. Linked prominently
  from the top of `README.md`.

### Changed
- Go toolchain pinned to **1.26.2** in both `.tool-versions` and
  `go.mod` (previously `latest`, which resolved to 1.26.1). See
  Security below. All other entries in `.tool-versions` are now
  explicitly version-pinned rather than tracking `latest` to avoid
  silent drift.
- `task precommit` slimmed to a **fast** hot-cache pass — `fmt` +
  `lint` + `test` + `build` (single-platform). ~2 s hot, ~8 s cold.
  Fuzz seed replay happens automatically as part of `task test` so
  no separate fuzz step is needed here. The comprehensive gate
  (race, active fuzz, vuln-check, cross-compile) lives in
  `precommit-full` and is intended for PR/release boundaries.
- `cmd/verify.go` and `cmd/create.go` now consume the shared
  `internal/inputsrc` resolver; the old duplicated six-mode input
  logic in each command is gone.
- `hash --style bare` now unconditionally omits the filename column
  (previously only did so when `--no-filename` was also passed,
  making `bare` accidentally byte-identical to `gnu` when a filename
  was available). The three styles now produce three distinct
  shapes:
  - `gnu` — `<hex>  <filename>` (sha256sum-compatible)
  - `bsd` — `<ALGO> (<filename>) = <hex>` (shasum --tag)
  - `bare` — `<hex>` (digest only, always)
- `internal/config.ConfigDir` and `internal/upgradecheck.cacheDir`
  split into `*_unix.go` / `*_windows.go` via build tags so each
  platform's branch is counted for coverage only on the platform
  where it can execute.
- `ProofBundle.MarshalJSON` now writes `ts`, `pk`, `sig`, `s`, `b`,
  `ip`, `cx` in a stable map — with `encoding/json`'s alphabetical
  key ordering this yields a canonical JSON form suitable for
  round-trip comparisons against JCS.
- `CONTRIBUTING.md` significantly expanded with a "When to run which
  task" table covering every Taskfile entry point, their durations,
  and recommended usage. The `Tests` section is now broken out by
  category (unit/integration, golden snapshots, fuzz, bench, race,
  coverage, lint, vuln-check) with guidance on what kind of test to
  add when.
- `CLAUDE.md` updated with new package structure entries, the
  deterministic-CBOR policy, the shared inputsrc pattern, and a
  note on the 59-target fuzz coverage.

### Security
- `golang.org/x/crypto` (already an indirect dependency) promoted to
  direct; used for SHA-3 and BLAKE2 implementations in
  `internal/hashing`. `task vuln-check` after the bump is clean.
- Go toolchain bump to 1.26.2 eliminates **six standard-library
  vulnerabilities** flagged by `govulncheck` in 1.26.1: five in
  `crypto/x509` (various certificate-parsing issues reachable via
  `io.Copy` through our hash-streaming path, though not exploitable
  through the CLI's actual call graph) and one XSS-class bug in
  `html/template` (reachable from `fmt.Fprintln` → `template.Error`,
  again not exploitable from the CLI's call graph but eliminated on
  principle). `task vuln-check` now reports zero findings.

### Fixed
- One real `staticcheck` finding addressed: an unused variable
  assignment in `cmd/coverage_extra_test.go` that the prior test
  suite never caught.

## [0.4.0] — 2026-04-17

### Added
- New `truestamp auth` parent command with `login`, `logout`, and
  `status` subcommands for managing the API key stored in
  `~/.config/truestamp/config.toml`.
  - `truestamp auth login` prints the web app's API-keys URL (derived
    from the configured `api_url` — so a local `http://localhost:4000/api/json`
    maps to `http://localhost:4000/api-keys`) and prompts for the key
    via a hidden-input field (`huh.EchoModePassword`). There is
    intentionally no `--api-key` flag; the key must be pasted into the
    prompt. The help text and on-screen hint both instruct the user to
    **create and copy a new key** — Truestamp does not allow re-copying
    existing keys after initial creation. The resulting config file is
    written with 0600 permissions.
  - `truestamp auth logout` confirms via an interactive `huh.Confirm`
    and clears `api_key` in the config (fast no-op when no key is set).
  - `truestamp auth status` is an always-online command: it renders a
    table of the resolved config (config path, API URL, probe URL,
    masked key, team in scope) and then calls
    `GET {api_url}/users?page[limit]=1&fields[user]=email,first_name,last_name,full_name`
    with `Authorization: Bearer <key>` (and `tenant: <team>` when a
    team is configured). On 2xx, the success banner shows
    `Authenticated as <full name> <email>`. When a team is configured,
    `auth status` additionally calls
    `GET {api_url}/teams/{id}?fields[team]=name,personal` to resolve
    and display the team's friendly name alongside its id
    (e.g. `Team: Acme Corp  [team_42]`). A 401/403 on the user probe
    is reported as "API key rejected by the server"; a 401/403/404 on
    the team probe is reported as "Team <id> is not accessible" —
    both exit 1, as does any transport-level failure. No offline mode
    is offered.
- New `internal/config.SetAPIKey` helper persists the API key to the
  on-disk config. It edits the `api_key` line in the top-level TOML
  scope in place (preserving comments and other settings), creating
  the config from the embedded default when it does not yet exist,
  and tightens file permissions to 0600 because the file now holds a
  secret.

### Changed
- `CONTRIBUTING.md`'s "Cutting a release" section updated to match
  the actual release flow: GoReleaser opens a PR on
  `truestamp/homebrew-tap` (since 0.3.0) and a follow-up workflow
  step auto-merges it (since 0.3.3). The `HOMEBREW_TAP_GITHUB_TOKEN`
  PAT scope list now correctly includes `Pull requests: Read and
  write` (CHANGELOG 0.3.0 had the right value but CONTRIBUTING had
  not been updated). Added a prerequisite to enable the
  `Allow auto-merge` repo setting on `truestamp/homebrew-tap`.
  Watch-the-release checks now verify the tap PR auto-merged
  rather than assuming a direct push. Partial-failure recipes now
  call out auto-merge failures (soft-fail under
  `continue-on-error: true`) as a distinct, recoverable mode.
- `CONTRIBUTING.md` development-setup now lists `caddy` alongside
  the other tools `mise install` bootstraps (added in 0.3.1 for the
  `task docs-serve` workflow).

## [0.3.3] — 2026-04-16

### Changed
- Release workflow now auto-merges the GoReleaser-generated PR on
  `truestamp/homebrew-tap` instead of requiring a manual click.
  Immediately after the `goreleaser release` step succeeds, a new
  step runs `gh pr merge --auto --merge --delete-branch` against the
  `goreleaser-<version>` branch using the existing
  `HOMEBREW_TAP_GITHUB_TOKEN` PAT. The step is `continue-on-error`
  so that a merge failure (conflict, rate limit, etc.) doesn't mask
  an otherwise-successful release — the tap PR can always be merged
  manually. Addresses the friction of having two open cask PRs
  stacking up during rapid back-to-back releases (see
  truestamp/homebrew-tap#3 and #4 for the last instance where PR #3
  blocked PR #4 with a conflict).
- Release workflow now requests `pull-requests: write` permission in
  addition to `contents: write`, `id-token: write`, and
  `attestations: write`. The auto-merge step uses the fine-grained
  `HOMEBREW_TAP_GITHUB_TOKEN` PAT, so the workflow permission is
  belt-and-braces and does not grant the default `GITHUB_TOKEN` any
  additional reach into `truestamp-cli` itself beyond its own PRs.

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

[Unreleased]: https://github.com/truestamp/truestamp-cli/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.5.0
[0.4.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.4.0
[0.3.3]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.3
[0.3.2]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.2
[0.3.1]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.1
[0.3.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.0
[0.2.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.2.0
[0.1.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.1.0
