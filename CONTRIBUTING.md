# Contributing

Thanks for your interest in `truestamp-cli`. This guide covers everything you need to hack on the CLI locally and, if you're a maintainer, to cut a release.

Before contributing code or discussion, please read [`CODE_OF_CONDUCT.md`](./CODE_OF_CONDUCT.md). For security issues, follow [`SECURITY.md`](./SECURITY.md) — do not open a public issue.

## Development setup

This repo uses [mise](https://mise.jdx.dev/) for tool versions and [Task](https://taskfile.dev/) for the developer workflow. A one-liner bootstrap:

```sh
mise install    # Installs Go, GoReleaser, cosign, shellcheck, syft, caddy from .tool-versions
task build      # Build for current platform → build/truestamp
```

Useful tasks:

```sh
task test              # All tests with -race and coverage
task precommit         # fmt + vet + test + build-all
task release-check     # Validate .goreleaser.yaml
task release-snapshot  # Local GoReleaser dry-run → dist/
```

The Go module path is `github.com/truestamp/truestamp-cli`. Minimum Go is pinned in `.tool-versions` and `go.mod`.

## Commit conventions

- **Sign every commit and tag.** The repo is configured for `commit.gpgsign=true` / `tag.gpgsign=true`; CI will flag unsigned commits in the next cycle.
- Keep the first line under 72 characters and written in the imperative (`Add X`, not `Added X`).
- Reference issues where relevant (`Fixes #123`). Use the body to explain *why*, not *what* — the diff shows the what.

## Tests

- All tests must pass under `task test` (which runs `go test -race -coverprofile=coverage.out ./...`).
- Prefer table-driven unit tests. Integration tests that shell out to the binary should be gated with a build tag so they can be skipped.
- New `internal/*` packages should get at least one `_test.go` file alongside them.

Run a single test file:

```sh
go test ./internal/verify/...
go test ./internal/verify -run TestInclusionProof
```

## Pull requests

- Branch from `main`, keep the change focused (one logical change per PR).
- Include a short description of the motivation and the observable behaviour change.
- Update `CHANGELOG.md` under `## [Unreleased]` using the Keep-a-Changelog groupings (`Added` / `Changed` / `Fixed` / `Removed`).
- CI must be green before a reviewer will look at the PR.

## Cutting a release (maintainer)

Releases are driven entirely by a git tag matching `v*`. Pushing the tag triggers [`.github/workflows/release.yml`](./.github/workflows/release.yml), which:

1. Runs GoReleaser to cross-compile the platform archives, generate `checksums.txt`, and publish a GitHub Release (including the cosign `.sigstore` bundle, SBOMs, and build-provenance attestation).
2. GoReleaser opens a PR on [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap) from a branch named `goreleaser-<version>` into `main`, updating `Casks/truestamp-cli.rb` with the new version and per-platform SHA-256s.
3. A follow-up `gh pr merge --auto --merge --delete-branch` step in the release workflow queues that PR for auto-merge. With no required reviews or status checks on the tap's `main` branch, the merge fires as soon as GitHub registers the PR as mergeable — typically within seconds of the GoReleaser step finishing.

The PR flow (introduced in 0.3.0) preserves an audit trail of every cask update; the auto-merge step (added in 0.3.3) removes the human-click step that used to be required.

### Prerequisites (one-time)

- Repository secret `HOMEBREW_TAP_GITHUB_TOKEN` on `truestamp/truestamp-cli`. **This must be a fine-grained PAT scoped to `truestamp/homebrew-tap` only, with `Contents: Read and write` + `Pull requests: Read and write`.** Do not use a classic `repo`-scoped PAT — the classic scope is broader than the release pipeline needs and should not be reintroduced. The `Pull requests` scope is what lets GoReleaser open the cask update PR and what lets the follow-up step auto-merge it.
- Repo setting `Allow auto-merge` enabled on `truestamp/homebrew-tap` (one-time). Without it, `gh pr merge --auto` rejects with `auto merge is not allowed for this repository`. Verify via `gh api repos/truestamp/homebrew-tap --jq '.allow_auto_merge'` — should return `true`.
- `mise install` locally so `task release-check` and `task release-snapshot` work for pre-flight testing.

### Pre-flight checklist

```sh
# Working copy is clean and on top of the latest origin/main.
jj git fetch
jj log -r 'main@origin..@'   # expect 0 commits not on origin

# All quality gates pass.
task precommit               # fmt + vet + test + build-all

# GoReleaser can build the full artifact set with ldflags intact.
task release-check           # validates .goreleaser.yaml
task release-snapshot        # rm -rf dist/ && goreleaser release --snapshot --clean

# Inspect the generated cask before tagging.
cat dist/homebrew/Casks/truestamp-cli.rb
```

### Update `CHANGELOG.md`

Move entries from `## [Unreleased]` into a new section for the version you're about to cut. Use today's date and the Keep-a-Changelog groupings. Add a matching link reference at the bottom.

```md
## [Unreleased]

## [X.Y.Z] — YYYY-MM-DD

### Added
- ...
```

### Commit and push the CHANGELOG

This repo is a jj colocated workspace. Commit the CHANGELOG edit as a normal change and advance `main`:

```sh
jj describe -m "Prep release vX.Y.Z"
jj bookmark move main --to @
jj git push --bookmark main
```

### Tag and push

jj does not create annotated tags itself — use the git CLI in the same working copy (the jj repo is colocated with `.git/`):

```sh
git tag -a vX.Y.Z -m "vX.Y.Z - one-line summary of the headline change"
git push origin vX.Y.Z
```

The tag must point at the exact commit that `main` now holds, and must start with `v` so GoReleaser's trigger (`push: tags: ['v*']`) fires.

### Watch the release

```sh
run_id=$(gh run list --workflow=release.yml --limit 1 --json databaseId -q '.[].databaseId')
gh run watch "$run_id" --exit-status

# Verify artifacts landed.
gh release view vX.Y.Z --json tagName,assets -q '{tag: .tagName, assets: (.assets | length)}'

# Confirm auto-merge ran and no tap PRs are dangling.
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
- **GoReleaser succeeded, tap auto-merge failed.** The GitHub Release is in place, but the tap PR is still open. The auto-merge step is `continue-on-error: true`, so the overall workflow goes green and you get a visible warning rather than a hard failure. Check `gh pr list --repo truestamp/homebrew-tap --state open` — if you see a stale `goreleaser-<ver>` PR, either fix whatever blocked it (e.g. a conflict from an overlapping release) and merge manually, or re-enable auto-merge with `gh pr merge <branch> --repo truestamp/homebrew-tap --auto --merge --delete-branch`.

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
