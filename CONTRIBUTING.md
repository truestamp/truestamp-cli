# Contributing

Thanks for your interest in `truestamp-cli`. This guide covers everything you need to hack on the CLI locally and, if you're a maintainer, to cut a release.

Before contributing code or discussion, please read [`CODE_OF_CONDUCT.md`](./CODE_OF_CONDUCT.md). For security issues, follow [`SECURITY.md`](./SECURITY.md) — do not open a public issue.

## Development setup

This repo uses [mise](https://mise.jdx.dev/) for tool versions and [Task](https://taskfile.dev/) for the developer workflow. A one-liner bootstrap:

```sh
mise install    # Installs Go, GoReleaser, cosign, shellcheck, syft, caddy from .tool-versions
task build      # Build for current platform → build/truestamp
```

Optional static-analysis and vuln-scan tools (needed by `task lint` and `task vuln-check`):

```sh
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

The Go module path is `github.com/truestamp/truestamp-cli`. Minimum Go is pinned in `.tool-versions` and `go.mod`; bumping Go there should always be followed by re-running `task vuln-check` to confirm no new stdlib CVEs surface.

## When to run which task

| Task | What it runs | Typical duration | When to use it |
| ---- | ------------ | ---------------- | -------------- |
| `task test` | `go test ./...` — every `TestXxx` + `FuzzXxx` seed replay across 17 packages | ~2–8 s | While iterating on code |
| `task precommit` | `fmt` + `lint` + `test` + `build` | <10 s hot cache | Before every commit |
| `task precommit-full` | Adds `test-race` + active fuzz + `vuln-check` + `build-all` | ~3–5 min | Before opening a PR or cutting a release |
| `task test-race` | Full suite under the race detector (`-race`) | ~60 s | When touching goroutines or package-level state |
| `task test-coverage` | Per-package coverage summary | ~5 s | Quick "where are the gaps?" check |
| `task test-coverage-full` | Coverage including CLI subprocess tests + HTML report | ~20 s | Before investing in more tests |
| `task bench` | Every `BenchmarkXxx` with `-benchmem` | ~30 s | Before merging a change that may affect hot paths |
| `task bench-compare` | Same, with `-count=5`, writes a baseline file for `benchstat` | ~2 min | A/B comparing performance between branches |
| `task fuzz` | Smoke-runs every `FuzzXxx` with its seed corpus (no mutation) | ~5 s | Explicit fuzz-seed pass (subsumed by `task test`) |
| `task fuzz-deep` | Active mutation fuzzing, default 15 s per target × 59 targets | ~15 min | Hardening pass before a release; override with `DURATION=1m task fuzz-deep` |
| `task fuzz-list` | Print the fuzz-target inventory | instant | Discover what's covered |
| `task lint` | `go vet` + `gofmt -l` + `staticcheck` + `gosec` | ~5–10 s | Part of `precommit`; rarely run standalone |
| `task vuln-check` | `govulncheck` against `go.mod` + stdlib | ~10 s | After a `go.mod` change or Go toolchain bump |
| `task release-check` | Validate `.goreleaser.yaml` | <5 s | Maintainer pre-release gate |
| `task release-snapshot` | Local GoReleaser dry-run → `dist/` | ~60 s | Maintainer pre-release gate |

Run a single test or a focused subset:

```sh
go test ./internal/verify/...                 # Every test in a package subtree
go test ./internal/verify -run TestInclusionProof
go test ./internal/hashing -bench=. -benchmem
go test -run=^$ -fuzz=FuzzParseCBOR -fuzztime=30s ./internal/proof/
```

## Commit conventions

- **Sign every commit and tag.** The repo is configured for `commit.gpgsign=true` / `tag.gpgsign=true`; CI will flag unsigned commits in the next cycle.
- Keep the first line under 72 characters and written in the imperative (`Add X`, not `Added X`).
- Reference issues where relevant (`Fixes #123`). Use the body to explain *why*, not *what* — the diff shows the what.

## Tests

New code is expected to ship with tests. The repo has **seven categories** of tests, each with a defined purpose:

### Unit and integration tests (`TestXxx`)

- **~650 functions across 17 packages.** Plain `go test` semantics — one test per invariant.
- Table-driven tests are preferred for parser / validator / encoder code.
- `cmd/` integration tests use a `TestMain` that builds the CLI binary in a tempdir once and then runs it as a subprocess for each test. This gives real exit-code + real stdout/stderr assertions without paying subprocess costs per-test. See `cmd/verify_test.go` for the pattern.
- New `internal/*` packages should ship at least one `_test.go` file alongside them.

### Golden-output snapshot tests (`cmd/golden_test.go`)

- Pin every user-facing CLI output (help text, `--list`, `--json` envelopes) byte-for-byte to committed fixtures under `cmd/testdata/golden/`.
- Catch silent wording / formatting / JSON-schema drift — the class of change that quietly breaks downstream scripts.
- Regenerate with `UPDATE_GOLDEN=1 go test ./cmd -run Golden` after an intentional output change.
- When you add a flag that affects output, add (or update) a golden test.

### Fuzz tests (`FuzzXxx`)

- **59 targets across 13 packages** covering every parser that touches attacker-controlled bytes: proof JSON + CBOR, encoding decoders, compact Merkle proofs, Bitcoin tx + txoutproof, TOML config, tar.gz extraction (path-traversal defense), ID / timestamp / URL / public-key parsers.
- Go's native fuzz framework calls your target in-process (no subprocess cost). Seed corpus lives in `f.Add()` calls; `go test` replays it as regression tests on every run. Active mutation kicks in only with `-fuzz=...`.
- Add a fuzz target whenever you write a parser that consumes external bytes. Assert at minimum "no panic"; add stronger invariants (round-trip, bounded output, etc.) where the semantics support it. See `internal/selfupgrade/fuzz_test.go`'s `FuzzExtractBinary` for a direct path-traversal assertion inside the fuzz callback.
- Crashing inputs discovered during fuzzing are auto-saved under `<pkg>/testdata/fuzz/FuzzXxx/` and become permanent regression seeds. Commit these.

### Benchmarks (`BenchmarkXxx`)

- **20+ targets** on hot paths: hashing across all 14 algorithms, proof parse / marshal (JSON + CBOR), encoding round-trip, Merkle proof decode + verify, domain-prefixed hashing.
- Run with `task bench` or `go test -bench=.`. `b.SetBytes` is used where throughput matters so `go test` reports MB/s alongside ns/op.
- Before merging a change to any parser or crypto primitive, capture a baseline with `task bench-compare` and diff with [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat).

### Race detector (`task test-race`)

- Runs the full suite under `-race`. Currently zero-finding on `main`; keep it that way. Any new goroutine, any new package-level mutable state, any test that swaps a package-level var should stay green under this task.
- Runs in `precommit-full` but not `precommit`, so PR authors should run it before opening a PR.

### Coverage (`task test-coverage` / `task test-coverage-full`)

- `task test-coverage` — fast per-package summary, no subprocess instrumentation.
- `task test-coverage-full` — builds the CLI binary with `-cover` so subprocess runs in `cmd/*_test.go` are counted too; merges test-process + subprocess covdata; emits `coverage.out` and `coverage.html`. This is the honest number.
- Target is 90%+ per package where reachable; packages below that threshold have structural reasons documented inline (interactive TTY, platform-specific branches, side-effect-heavy upgrade pipeline).

### Static analysis (`task lint`) and vulnerability scan (`task vuln-check`)

- `go vet` + `gofmt -l` + `staticcheck` + `gosec`. Lint exclusions (`G104`, `G115`, `G304`, etc.) are documented inline in the Taskfile with rationale — if you disagree with one, argue the case in the PR.
- `govulncheck` is run by `precommit-full` and must be clean before any release. Re-run it after every Go toolchain bump.

## Pull requests

- Branch from `main`, keep the change focused (one logical change per PR).
- Include a short description of the motivation and the observable behaviour change.
- Update `CHANGELOG.md` under `## [Unreleased]` using the Keep-a-Changelog groupings (`Added` / `Changed` / `Fixed` / `Removed`).
- CI must be green before a reviewer will look at the PR.

## Cutting a release (maintainer)

Releases are driven entirely by a git tag matching `v*`. Pushing the tag triggers [`.github/workflows/release.yml`](./.github/workflows/release.yml), which:

1. Runs GoReleaser to cross-compile the platform archives, generate `checksums.txt`, and publish a GitHub Release (including the cosign `.sigstore` bundle, SBOMs, and build-provenance attestation).
2. GoReleaser opens a PR on [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap) from a branch named `goreleaser-<version>` into `main`, updating `Casks/truestamp-cli.rb` with the new version and per-platform SHA-256s.
3. A follow-up `gh pr merge --merge --delete-branch` step in the release workflow merges that PR directly. The tap's `protect-main` ruleset only blocks branch deletion and non-fast-forward pushes — no required status checks or reviews — so there is nothing to gate mergeability, and `--auto` (which queues until some pending check / review clears) rejects the PR as already clean. Direct merge is the right call.

The PR flow (introduced in 0.3.0) preserves an audit trail of every cask update; the auto-merge step (added in 0.3.3, simplified to a direct merge in 0.6.0 after auto-merge kept rejecting already-clean PRs) removes the human-click step that used to be required.

### Prerequisites (one-time)

- Repository secret `HOMEBREW_TAP_GITHUB_TOKEN` on `truestamp/truestamp-cli`. **This must be a fine-grained PAT scoped to `truestamp/homebrew-tap` only, with `Contents: Read and write` + `Pull requests: Read and write`.** Do not use a classic `repo`-scoped PAT — the classic scope is broader than the release pipeline needs and should not be reintroduced. The `Pull requests` scope is what lets GoReleaser open the cask update PR and what lets the follow-up step merge it.
- `mise install` locally so `task release-check` and `task release-snapshot` work for pre-flight testing.
- `protect-main` ruleset (repo Settings → Rules → Rulesets). Enforces linear history, blocks force-pushes and deletions, and **requires `Test (ubuntu-latest)` + `Test (macos-latest)` green on the exact SHA** before anything merges to `main`. The release flow below routes the CHANGELOG commit through a PR specifically to satisfy that rule. Release tags then trigger a second CI re-run (via `release.yml`'s `workflow_call` into `ci.yml`) before GoReleaser starts — two layers of "tests green on this SHA" before artifacts publish.

### Pre-flight checklist

```sh
# Working copy is clean and on top of the latest origin/main.
jj git fetch
jj log -r 'main@origin..@'   # expect the empty WC change, nothing else

# Full quality gate — race detector + active fuzz + vuln scan + all-platform build.
# Takes ~3-5 minutes; use this at the release boundary, not for every commit.
task precommit-full

# GoReleaser can build the full artifact set with ldflags intact.
task release-check           # validates .goreleaser.yaml (<1s)
task release-snapshot        # rm -rf dist/ && goreleaser release --snapshot --clean --skip=sign,publish

# Inspect the generated cask before tagging.
cat dist/homebrew/Casks/truestamp-cli.rb
```

`release-snapshot` skips `sign` and `publish` because cosign keyless signing requires a GitHub OIDC token (only available inside the release workflow), and no dry-run should touch the GitHub Release API. Expect the `version` in the rendered cask to read `X.Y.Z-SNAPSHOT-<shortsha>` — that gets replaced with the real tag during the actual release.

### Update `CHANGELOG.md`

Move entries from `## [Unreleased]` into a new section for the version you're about to cut. Use today's date and the Keep-a-Changelog groupings.

```md
## [Unreleased]

## [X.Y.Z] — YYYY-MM-DD

### Added
- ...
```

### Open a release PR for the CHANGELOG commit

The `protect-main` ruleset (see Prerequisites) requires CI checks to pass on the exact SHA before `main` accepts it, so the release commit must land via PR — direct `jj git push --bookmark main` is rejected with `GH013: Repository rule violations found`. This is by design: the PR gives CI a chance to run on the SHA that's about to be tagged.

```sh
# Commit the CHANGELOG edit.
jj describe -m "Release vX.Y.Z"

# Push to a release branch instead of directly to main.
jj bookmark create release-vX.Y.Z -r @
jj git push --bookmark release-vX.Y.Z

# Open the PR. Keep the title exactly "Release vX.Y.Z" — it's what
# the changelog and commit history expect.
gh pr create --base main --head release-vX.Y.Z \
  --title "Release vX.Y.Z" \
  --body "See CHANGELOG.md for the full release notes."

# Wait for CI to go green on the PR, then merge via rebase so the
# signed tag below points at the exact CHANGELOG commit (merge commits
# would introduce a different SHA, which the linear-history rule also
# rejects anyway).
gh pr checks <pr> --watch --repo truestamp/truestamp-cli
gh pr merge <pr> --rebase --delete-branch --repo truestamp/truestamp-cli

# Sync jj to the post-merge main.
jj git fetch
jj bookmark set main -r main@origin
```

### Tag and push

jj does not create annotated tags itself — use the git CLI in the same working copy (the jj repo is colocated with `.git/`). Run `git tag -v` afterwards to confirm the tag was signed; the repo has `tag.gpgsign=true` + `user.signingkey` set, so plain `git tag -a` auto-signs.

```sh
git tag -a vX.Y.Z -m "vX.Y.Z - one-line summary of the headline change"
git tag -v vX.Y.Z   # expect "Good 'git' signature ..." — abort if unsigned.
git push origin vX.Y.Z
```

The tag must point at the exact commit that `main` now holds, and must start with `v` so GoReleaser's trigger (`push: tags: ['v*']`) fires.

### Watch the release

The tag push triggers `release.yml`, which runs two top-level jobs:

1. `ci` — `workflow_call` into `ci.yml`, re-running the full lint + test matrix on the tagged SHA. If this fails, nothing publishes.
2. `goreleaser` (needs: ci) — runs `goreleaser check`, then a `--snapshot --clean` dry-run (local cross-compile + SBOM + cask template render, surfaces platform-specific breakage before the real publish), then the real `goreleaser release --clean`, then the homebrew-tap PR merge, then build-provenance attestation.

Total runtime: ~7-9 minutes (CI gate 3-5 min + snapshot ~1 min + real release ~2-3 min).

```sh
run_id=$(gh run list --workflow=release.yml --limit 1 --json databaseId -q '.[].databaseId')
gh run watch "$run_id" --exit-status

# Verify artifacts landed (expect 14 assets: checksums.txt +
# checksums.txt.sigstore + 6 platform archives + 6 SBOMs).
gh release view vX.Y.Z --json tagName,assets -q '{tag: .tagName, assets: (.assets | length)}'

# Confirm the tap PR merged and none are dangling.
gh pr list --repo truestamp/homebrew-tap --state open    # expect empty
gh pr list --repo truestamp/homebrew-tap --state merged --limit 1   # expect the goreleaser-<ver> PR

# Confirm the tap cask on main has the new version.
gh api repos/truestamp/homebrew-tap/contents/Casks/truestamp-cli.rb -q '.content' | base64 -d | grep '^  version'
```

### Smoke-test the install channels

```sh
# install.sh (get.truestamp.com).
curl -fsSL https://get.truestamp.com/install.sh | TRUESTAMP_INSTALL_DIR=/tmp sh
/tmp/truestamp version

# Homebrew (macOS / Linux).
brew update
brew upgrade truestamp/tap/truestamp-cli
xattr -cr "$(brew --caskroom)/truestamp-cli"   # macOS Gatekeeper, first run only
truestamp version

# Go install.
go install github.com/truestamp/truestamp-cli/cmd/truestamp@vX.Y.Z
truestamp version

# Direct tarball.
os=$(uname -s | tr A-Z a-z)
arch=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -sSL "https://github.com/truestamp/truestamp-cli/releases/download/vX.Y.Z/truestamp-cli_X.Y.Z_${os}_${arch}.tar.gz" | tar -xz
./truestamp version
```

### If the release fails partway

GoReleaser is mostly idempotent, but partial failures are possible. The two common modes:

- **GoReleaser step failed outright** (cross-compile broke, cosign signing flaked, tap PAT expired). No GitHub Release, no tap PR. Redo cleanly with the recipe below.
- **GoReleaser succeeded, tap merge failed.** The GitHub Release is in place, but the tap PR is still open. The merge step is `continue-on-error: true`, so the overall workflow goes green and you get a visible warning rather than a hard failure. Check `gh pr list --repo truestamp/homebrew-tap --state open` — if you see a stale `goreleaser-<ver>` PR, either fix whatever blocked it (e.g. a conflict from an overlapping release) and merge manually with `gh pr merge <branch> --repo truestamp/homebrew-tap --merge --delete-branch`.

To redo a release cleanly:

```sh
gh release delete vX.Y.Z -y
git push origin :refs/tags/vX.Y.Z
git tag -d vX.Y.Z

# Fix the problem in a new commit, push to main, then retag from the fixed commit.
git tag -a vX.Y.Z -m "vX.Y.Z - ..."
git push origin vX.Y.Z
```

Do **not** re-tag a version that has already propagated to `proxy.golang.org` — the proxy caches tagged module versions forever. Bump the patch version (e.g. `vX.Y.Z+1`) instead.
