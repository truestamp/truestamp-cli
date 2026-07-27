# Contributing

Thanks for your interest in `truestamp-cli`. This guide covers everything you need to hack on the CLI locally and, if you're a maintainer, to cut a release.

Before contributing code or discussion, please read [`CODE_OF_CONDUCT.md`](./CODE_OF_CONDUCT.md). For security issues, follow [`SECURITY.md`](./SECURITY.md): do not open a public issue.

## Development setup

This repo uses [mise](https://mise.jdx.dev/) for tool versions and [Task](https://taskfile.dev/) for the developer workflow. A one-liner bootstrap:

```sh
mise install    # Installs every tool pinned in .tool-versions (including `task` itself)
task build      # Build for current platform -> build/truestamp
```

`mise install` is the only setup step. [`.tool-versions`](./.tool-versions) pins the full toolchain, and `mise install` provides all of it:

| Tool | Used by |
| ---- | ------- |
| `go` | everything |
| `go-task` (the `task` runner) | every task below, including in CI |
| `staticcheck`, `gosec` | `task lint` |
| `govulncheck` | `task vuln-check` |
| `goreleaser`, `cosign`, `syft` | `task release-check`, `task release-snapshot`, the release workflow |
| `shellcheck` | CI's `docs/install.sh` lint |
| `caddy` | `task docs-serve` |
| `websocat` | manual `truestamp console` wire-protocol testing |

Do **not** `go install` the linters at `@latest`. CI deliberately invokes the same Taskfile tasks you run locally so that tool versions come from `.tool-versions` and nothing can drift between a developer machine and CI. A `go install`-ed binary also lives in the active Go toolchain's bin directory and is shadowed the moment the Go version moves.

The Go module path is `github.com/truestamp/truestamp-cli`, and the main package is `./cmd/truestamp` (not the repo root, so a bare `go build .` will not work). The Go floor is `go 1.26.5`, stated in both [`go.mod`](./go.mod) and [`.tool-versions`](./.tool-versions); the two must stay in agreement. Bumping Go should always be followed by re-running `task vuln-check` to confirm no new stdlib CVEs surface.

## Version control: this repo uses Jujutsu

The repo is managed with [Jujutsu (jj)](https://jj-vcs.github.io/), colocated on top of the git repo. The working copy is always a jj change, so **`git status` reporting `Not currently on any branch` (detached HEAD) is the normal steady state**. Do not try to "fix" it by creating a git branch.

Read-only git commands (`git log`, `git show`, `git diff`, `git blame`, `gh ...`) are fine, since jj and git share the same object store. Prefer jj for anything that mutates state so the jj operation log and bookmarks stay coherent:

| Need | Use | Avoid |
| ---- | --- | ----- |
| See working-copy state | `jj st` / `jj log` | `git status` |
| See the diff | `jj diff` | `git diff` (works, but jj is canonical) |
| Set or amend the commit message | `jj desc -r @ -m '...'` | `git commit -m`, `git commit --amend` |
| Start a new logical change | `jj new` | `git checkout -b` |
| Move work between changes | `jj squash` / `jj split` | `git rebase -i`, `git reset` |
| Push a branch | `jj bookmark create` + `jj git push` | `git push` |

The one exception is annotated tags: jj does not create them, so the release flow below uses `git tag -a` in the colocated working copy.

## When to run which task

`task` with no argument (or `task --list-all`) prints every available target. The ones you will actually reach for:

| Task | What it runs | Typical duration | When to use it |
| ---- | ------------ | ---------------- | -------------- |
| `task build` | Builds `./cmd/truestamp` with version ldflags to `build/truestamp` | ~1 s | Iterating on the binary |
| `task build-all` | Cross-compiles all 6 release targets into `build/` | ~10 s | Checking a platform-specific change |
| `task test` | `go test ./...`, every `TestXxx` plus `FuzzXxx` seed replay across 29 test packages | ~11 s cold, <1 s fully cached | While iterating on code |
| `task precommit` | `fmt` + `lint` + `vuln-check` + `test` + `build` | seconds on a hot cache | Before every commit |
| `task precommit-full` | `fmt` + `lint` + `test-race` + `fuzz` (seed corpus) + `vuln-check` + `build-all` | seconds hot cache, minutes cold | Before opening a PR or cutting a release |
| `task test-race` | Full suite under the race detector (`-race`) | ~15 s | When touching goroutines or package-level state |
| `task test-chaos` | `wschannel` reconnect-under-stress cases behind the `chaos` build tag, under `-race` | ~11 s | When touching the WebSocket session loop |
| `task test-coverage` | Per-package coverage summary | ~12 s | Quick "where are the gaps?" check |
| `task test-coverage-full` | Coverage including CLI subprocess tests, plus an HTML report | ~15 s | Before investing in more tests |
| `task bench` | Every `BenchmarkXxx` with `-benchmem` | ~30 s | Before merging a change that may affect hot paths |
| `task bench-compare` | Same, with `-count=5`, writes a baseline file for `benchstat` | ~2 min | A/B comparing performance between branches |
| `task fuzz` | Smoke-runs every `FuzzXxx` with its seed corpus (no mutation) | ~2 s | Explicit fuzz-seed pass (subsumed by `task test`) |
| `task fuzz-codec` | Active mutation fuzzing of the 3 `wschannel` codec/redactor targets, 30 s each | ~90 s | When touching the Phoenix frame codec or redactor |
| `task fuzz-deep` | Active mutation fuzzing, default 15 s per target across 79 targets | ~20 min | Hardening pass before a release; override with `DURATION=1m task fuzz-deep` |
| `task fuzz-list` | Print the fuzz-target inventory | instant | Discover what's covered |
| `task lint` | `task vet` + `task fmt-check` + `task staticcheck` + `task gosec` | ~3 s | Part of `precommit`; the four sub-tasks also run standalone |
| `task vuln-check` | `govulncheck ./...` against `go.mod` plus the stdlib | ~5 s | After a `go.mod` change or Go toolchain bump |
| `task release-check` | Validate `.goreleaser.yaml` | <1 s | Maintainer pre-release gate |
| `task release-snapshot` | Local GoReleaser dry-run into `dist/` | ~60 s | Maintainer pre-release gate |

Durations are warm-cache measurements on Apple Silicon; treat them as orders of magnitude, not guarantees. A cold Go build cache dominates the first run of anything.

CI ([`ci.yml`](./.github/workflows/ci.yml)) invokes `task fmt-check`, `task vet`, `task staticcheck`, `task gosec`, and `task vuln-check` directly, so a green `task lint && task vuln-check` locally means the same tool versions and the same exclusion lists ran that CI will run.

Run a single test or a focused subset:

```sh
go test ./internal/verify/...                 # Every test in a package subtree
go test ./internal/verify -run TestVerifyInclusionProof
go test ./internal/hashing -bench=. -benchmem
go test -run=^$ -fuzz=FuzzParseCBOR -fuzztime=30s ./internal/proof/
```

## Commit conventions

- **Sign every commit and tag.** The repo is configured with `commit.gpgsign=true` / `tag.gpgsign=true` and `gpg.format=ssh`, so commits and annotated tags are signed with the configured ED25519 SSH key automatically.
- Keep the first line under 72 characters and written in the imperative (`Add X`, not `Added X`).
- Reference issues where relevant (`Fixes #123`). Use the body to explain *why*, not *what*: the diff shows the what.

## Tests

New code is expected to ship with tests. The repo has **eight categories** of tests, each with a defined purpose:

### Unit and integration tests (`TestXxx`)

- **~1,230 functions across 29 test packages.** Plain `go test` semantics, one test per invariant.
- Table-driven tests are preferred for parser / validator / encoder code.
- `cmd/` integration tests use a `TestMain` (in `cmd/verify_test.go`) that builds the CLI binary in a tempdir once and then runs it as a subprocess for each test. This gives real exit-code plus real stdout/stderr assertions without paying subprocess build costs per-test.
- New `internal/*` packages should ship at least one `_test.go` file alongside them.

### Golden-output snapshot tests (`cmd/golden_test.go`)

- Pin user-facing CLI output (help text, `--list`, `--json` envelopes) byte-for-byte to committed fixtures under `cmd/testdata/golden/`.
- Catch silent wording / formatting / JSON-schema drift, the class of change that quietly breaks downstream scripts.
- Regenerate with `UPDATE_GOLDEN=1 go test ./cmd -run Golden` after an intentional output change, or pass the `-update-golden` flag.
- When you add a flag that affects one of the pinned outputs, add (or update) a golden test.

### Fuzz tests (`FuzzXxx`)

- **79 targets across 20 packages** covering every parser that touches attacker-controlled bytes: proof JSON and CBOR, encoding decoders, compact Merkle proofs, Bitcoin tx and txoutproof, TOML config, tar.gz extraction (path-traversal defense), OAuth discovery and session decoding, Phoenix frames, and ID / timestamp / URL / public-key parsers. Run `task fuzz-list` for the current inventory.
- Go's native fuzz framework calls your target in-process (no subprocess cost). Seed corpus lives in `f.Add()` calls; `go test` replays it as regression tests on every run. Active mutation kicks in only with `-fuzz=...`.
- Add a fuzz target whenever you write a parser that consumes external bytes. Assert at minimum "no panic"; add stronger invariants (round-trip, bounded output, and so on) where the semantics support it. See `internal/selfupgrade/fuzz_test.go`'s `FuzzExtractBinary` for a direct path-traversal assertion inside the fuzz callback.
- Crashing inputs discovered during fuzzing are auto-saved under `<pkg>/testdata/fuzz/FuzzXxx/` and become permanent regression seeds. Commit these. After a `task fuzz-deep` session, `find . -path '*/testdata/fuzz/*' -type f` lists what to track.

### Chaos tests (`task test-chaos`)

- Reconnect-under-stress cases for `internal/wschannel`, gated behind the `chaos` build tag so they don't pad every plain `go test` run.
- They take ~10 s because of deliberate reconnect-backoff sleeps. CI runs them on every PR, so run `task test-chaos` before pushing a change to the WebSocket session loop.

### Benchmarks (`BenchmarkXxx`)

- **20 targets across 4 packages** on hot paths: hashing across all 14 algorithms (`internal/hashing`), proof parse / marshal for JSON and CBOR (`internal/proof`), encoding round-trip (`internal/encoding`), and Merkle proof decode / verify plus domain-prefixed hashing (`internal/tscrypto`).
- Run with `task bench` or `go test -bench=.`. `b.SetBytes` is used where throughput matters so `go test` reports MB/s alongside ns/op.
- Before merging a change to any parser or crypto primitive, capture a baseline with `task bench-compare` and diff with [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat).

### Race detector (`task test-race`)

- Runs the full suite under `-race`, and is clean on `main`. Keep it that way. Any new goroutine, any new package-level mutable state, and any test that swaps a package-level var should stay green under this task.
- Runs in `precommit-full` but not `precommit`, so PR authors should run it before opening a PR. CI runs the whole matrix with `-race` on every PR.

### Coverage (`task test-coverage` / `task test-coverage-full`)

- `task test-coverage` is a fast per-package summary with no subprocess instrumentation. It **understates `cmd/` badly** (~23%), because the CLI integration tests exercise a subprocess whose coverage it cannot see.
- `task test-coverage-full` builds the CLI binary with `-cover` so those subprocess runs are counted too, merges the test-process and subprocess covdata, and emits `coverage.out` plus `coverage.html`. This is the honest number: it moves `cmd/` to ~71% and reports a repo total around 72%.
- Coverage is uneven by design rather than uniformly high. The crypto, parsing, and wire-format packages that decide a proof's verdict sit at 90% and above (`tscrypto`, `bitcoin`, `proof`, `external`, `httpclient`, `items`, `encoding`, `config`, `jcs`, `verify`). The packages that sit lower are dominated by interactive TTY rendering, platform-specific branches, and long-lived network sessions (`console/*`, `ui`, `wschannel`, `auth`, `selfupgrade`). When adding tests, prefer raising a verdict-bearing package over chasing a repo-wide percentage.

### Static analysis (`task lint`) and vulnerability scan (`task vuln-check`)

- `task lint` runs `go vet`, `gofmt -l`, `staticcheck`, and `gosec`. The gosec exclusions (`G104`, `G115`, `G304`, and others) are documented inline in `Taskfile.yml` with rationale. If you disagree with one, argue the case in the PR. Both the tool versions and the exclusion list are single-sourced in `.tool-versions` and `Taskfile.yml`, and CI invokes those same tasks.
- `govulncheck` runs in `precommit`, in `precommit-full`, and in CI, and must be clean before any release. Re-run it after every Go toolchain bump. Pinning govulncheck does not reduce CVE coverage: it queries the live vuln.go.dev database at run time.

## Pull requests

- Start a new change with `jj new` on top of the latest `main`, and keep the change focused (one logical change per PR).
- Push with `jj bookmark create <name> -r @` followed by `jj git push --bookmark <name>`, then open the PR with `gh pr create`.
- Include a short description of the motivation and the observable behaviour change.
- Update `CHANGELOG.md` under `## [Unreleased]` using the Keep-a-Changelog groupings (`Added` / `Changed` / `Fixed` / `Removed`).
- CI must be green before a reviewer will look at the PR. The `protect-main` ruleset will not let the PR merge otherwise.

## Cutting a release (maintainer)

Releases are driven entirely by a git tag matching `v*`. Pushing the tag triggers [`.github/workflows/release.yml`](./.github/workflows/release.yml), which:

1. Re-runs the full CI matrix on the tagged SHA via `workflow_call` into `ci.yml`. If this fails, nothing publishes.
2. Runs GoReleaser to cross-compile the platform archives, generate `checksums.txt`, and publish a GitHub Release (including the cosign `.sigstore` bundle, per-archive SBOMs, and a build-provenance attestation).
3. GoReleaser opens a PR on [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap) from a branch named `goreleaser-<version>` into `main`, updating `Casks/truestamp-cli.rb` with the new version and per-platform SHA-256s. This preserves an audit trail of every cask update.
4. A follow-up `gh pr merge --merge --delete-branch` step merges that PR directly. The tap's `protect-main` ruleset only blocks branch deletion and non-fast-forward pushes, with no required status checks or reviews, so there is nothing to gate mergeability, and `--auto` (which queues until some pending check or review clears) rejects the PR as already clean. Direct merge is the right call.

### How to actually cut a release: use `/release`

**The canonical way to ship a release is to run the [`/release` skill](.claude/skills/release/SKILL.md) from Claude Code.** It walks the entire flow end-to-end (pre-flight gates, CHANGELOG update, release PR plus CI wait, signed tag, release.yml verification, GitHub Release plus Homebrew tap plus cosign / SLSA-attestation checks) and produces either a structured success report or actionable failure diagnostics keyed to [`.claude/skills/release/references/failure-recovery.md`](.claude/skills/release/references/failure-recovery.md).

The sections below document the underlying steps for reference. You will need them when diagnosing a failure, reviewing the skill, or cutting a release without Claude Code, but they are not the recommended day-to-day procedure: typos and skipped steps are much more likely by hand, and the `protect-main` ruleset will reject any direct-push flow anyway.

If you are cutting a release from a shell alone, follow the sections below exactly as written. The skill derives every command from them.

### Prerequisites (one-time)

- Repository secret `HOMEBREW_TAP_GITHUB_TOKEN` on `truestamp/truestamp-cli`. **This must be a fine-grained PAT scoped to `truestamp/homebrew-tap` only, with `Contents: Read and write` plus `Pull requests: Read and write`.** Do not use a classic `repo`-scoped PAT: the classic scope is broader than the release pipeline needs. The `Pull requests` scope is what lets GoReleaser open the cask update PR and what lets the follow-up step merge it.
- `mise install` locally, so `task release-check` and `task release-snapshot` work for pre-flight testing.
- `protect-main` ruleset (repo Settings, then Rules, then Rulesets). It enforces linear history, blocks force-pushes and deletions, and **requires `Test (ubuntu-latest)` plus `Test (macos-latest)` green on the exact SHA** before anything merges to `main`. The release flow below routes the CHANGELOG commit through a PR specifically to satisfy that rule. Release tags then trigger a second CI re-run before GoReleaser starts, giving two layers of "tests green on this SHA" before artifacts publish.

### Pre-flight checklist

```sh
# Working copy is clean and on top of the latest origin/main.
jj git fetch
jj log -r 'main@origin..@'   # expect the empty WC change, nothing else

# Full quality gate: race detector + fuzz seed replay + vuln scan + all-platform build.
task precommit-full

# GoReleaser can build the full artifact set with ldflags intact.
task release-check           # validates .goreleaser.yaml (<1s)
task release-snapshot        # goreleaser release --snapshot --clean --skip=sign,publish

# Inspect the generated cask before tagging.
cat dist/homebrew/Casks/truestamp-cli.rb
```

`release-snapshot` skips `sign` and `publish` because cosign keyless signing requires a GitHub OIDC token (only available inside the release workflow), and no dry-run should touch the GitHub Release API. Expect the `version` in the rendered cask to read `X.Y.Z-SNAPSHOT-<shortsha>`; that gets replaced with the real tag during the actual release.

### Update `CHANGELOG.md`

Move entries from `## [Unreleased]` into a new section for the version you're about to cut. Use today's date and the Keep-a-Changelog groupings.

```md
## [Unreleased]

## [X.Y.Z] - YYYY-MM-DD

### Added
- ...
```

### Open a release PR for the CHANGELOG commit

The `protect-main` ruleset (see Prerequisites) requires CI checks to pass on the exact SHA before `main` accepts it, so the release commit must land via PR. A direct `jj git push --bookmark main` is rejected with `GH013: Repository rule violations found`. This is by design: the PR gives CI a chance to run on the SHA that is about to be tagged.

```sh
# Describe the CHANGELOG change.
jj desc -r @ -m "Release vX.Y.Z"

# Push to a release branch instead of directly to main.
jj bookmark create release-vX.Y.Z -r @
jj git push --bookmark release-vX.Y.Z

# Open the PR. Keep the title exactly "Release vX.Y.Z" so it matches
# the changelog and commit history.
gh pr create --base main --head release-vX.Y.Z \
  --title "Release vX.Y.Z" \
  --body "See CHANGELOG.md for the full release notes."

# Wait for CI to go green on the PR, then merge via rebase so the
# signed tag below points at the exact CHANGELOG commit. Merge commits
# would introduce a different SHA, which the linear-history rule also
# rejects anyway.
gh pr checks <pr> --watch --repo truestamp/truestamp-cli
gh pr merge <pr> --rebase --delete-branch --repo truestamp/truestamp-cli

# Sync jj to the post-merge main.
jj git fetch
jj bookmark set main -r main@origin
```

### Tag and push

jj does not create annotated tags itself, so use the git CLI in the same working copy (the jj repo is colocated with `.git/`). Pin the tag to the merged release SHA rather than to whatever the working copy currently points at. The repo has `tag.gpgsign=true`, `gpg.format=ssh`, and `user.signingkey` set, so plain `git tag -a` auto-signs.

```sh
RELEASE_SHA=$(git rev-parse main@{upstream})
git tag -a vX.Y.Z -m "vX.Y.Z - one-line summary of the headline change" "$RELEASE_SHA"
git tag -v vX.Y.Z   # expect: Good "git" signature ... Abort if unsigned.
git push origin vX.Y.Z
```

The tag must point at the exact commit that `main` now holds, and must start with `v` so GoReleaser's trigger (`push: tags: ['v*']`) fires.

Signing the tag may require an interactive approval from your key agent, so this step is run by a human even when the rest of the flow is driven by `/release`.

### Watch the release

The tag push triggers `release.yml`, which runs two top-level jobs:

1. `ci`, a `workflow_call` into `ci.yml`, re-running the full lint plus test matrix on the tagged SHA. If this fails, nothing publishes.
2. `goreleaser` (`needs: ci`), which runs `goreleaser check`, then a `--snapshot --clean` dry-run (local cross-compile, SBOM, and cask template render, surfacing platform-specific breakage before the real publish), then the real `goreleaser release --clean`, then the homebrew-tap PR merge, then build-provenance attestation.

Total runtime is roughly 7 to 9 minutes.

```sh
run_id=$(gh run list --workflow=release.yml --limit 1 --json databaseId -q '.[].databaseId')
gh run watch "$run_id" --exit-status

# Verify artifacts landed. Expect 14 assets: checksums.txt +
# checksums.txt.sigstore + 6 platform archives (4 tar.gz, 2 Windows zip)
# + 6 SBOMs.
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

The `/release` skill routes failures to [`.claude/skills/release/references/failure-recovery.md`](.claude/skills/release/references/failure-recovery.md), which covers every scenario observed in practice (CI gate failed before publish, GoReleaser published but the tap merge flaked, the Go proxy cached a broken version, and others) with specific recipes. Prefer that reference when diagnosing; the outline below is the fallback if you are recovering without Claude Code.

GoReleaser is mostly idempotent, but partial failures are possible. The two common modes:

- **GoReleaser step failed outright** (cross-compile broke, cosign signing flaked, tap PAT expired). No GitHub Release, no tap PR. Redo cleanly with the recipe below.
- **GoReleaser succeeded, tap merge failed.** The GitHub Release is in place, but the tap PR is still open. The merge step is `continue-on-error: true`, so the overall workflow goes green and you get a visible warning rather than a hard failure. Check `gh pr list --repo truestamp/homebrew-tap --state open`; if you see a stale `goreleaser-<ver>` PR, fix whatever blocked it (for example a conflict from an overlapping release) and merge manually with `gh pr merge <branch> --repo truestamp/homebrew-tap --merge --delete-branch`.

To redo a release cleanly:

```sh
gh release delete vX.Y.Z -y
git push origin :refs/tags/vX.Y.Z
git tag -d vX.Y.Z

# Fix the problem in a new change and land it on main through a PR
# (protect-main rejects a direct push), then retag from the fixed SHA.
git tag -a vX.Y.Z -m "vX.Y.Z - ..." "$(git rev-parse main@{upstream})"
git push origin vX.Y.Z
```

Do **not** re-tag a version that has already propagated to `proxy.golang.org`. The proxy caches tagged module versions forever. Bump the patch version instead.
