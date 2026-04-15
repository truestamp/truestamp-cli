# Contributing

Thanks for your interest in `truestamp-cli`. This guide covers everything you need to hack on the CLI locally and, if you're a maintainer, to cut a release.

Before contributing code or discussion, please read [`CODE_OF_CONDUCT.md`](./CODE_OF_CONDUCT.md). For security issues, follow [`SECURITY.md`](./SECURITY.md) — do not open a public issue.

## Development setup

This repo uses [mise](https://mise.jdx.dev/) for tool versions and [Task](https://taskfile.dev/) for the developer workflow. A one-liner bootstrap:

```sh
mise install    # Installs Go, GoReleaser, cosign, shellcheck, syft from .tool-versions
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

Releases are driven entirely by a git tag matching `v*`. Pushing the tag triggers [`.github/workflows/release.yml`](./.github/workflows/release.yml), which runs GoReleaser to build the platform archives, generate `checksums.txt`, publish a GitHub Release, and update the Homebrew cask at [`truestamp/homebrew-tap`](https://github.com/truestamp/homebrew-tap).

### Prerequisites (one-time)

- Repository secret `HOMEBREW_TAP_GITHUB_TOKEN` on `truestamp/truestamp-cli`. **This must be a fine-grained PAT scoped to `truestamp/homebrew-tap` only, with `Contents: Read and write`.** Do not use a classic `repo`-scoped PAT — the classic scope is broader than the release pipeline needs and should not be reintroduced.
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

## [0.2.0] — 2026-04-20

### Added
- ...
```

### Commit and push the CHANGELOG

This repo is a jj colocated workspace. Commit the CHANGELOG edit as a normal change and advance `main`:

```sh
jj describe -m "Prep release v0.2.0"
jj bookmark move main --to @
jj git push --bookmark main
```

### Tag and push

jj does not create annotated tags itself — use the git CLI in the same working copy (the jj repo is colocated with `.git/`):

```sh
git tag -a v0.2.0 -m "v0.2.0 - one-line summary of the headline change"
git push origin v0.2.0
```

The tag must point at the exact commit that `main` now holds, and must start with `v` so GoReleaser's trigger (`push: tags: ['v*']`) fires.

### Watch the release

```sh
run_id=$(gh run list --workflow=release.yml --limit 1 --json databaseId -q '.[].databaseId')
gh run watch "$run_id" --exit-status

# Verify artifacts landed.
gh release view v0.2.0 --json tagName,assets -q '{tag: .tagName, assets: (.assets | length)}'

# Confirm the tap cask updated.
gh api repos/truestamp/homebrew-tap/contents/Casks/truestamp-cli.rb -q '.content' | base64 -d | head
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
go install github.com/truestamp/truestamp-cli/cmd/truestamp@v0.2.0
truestamp version

# Direct tarball.
os=$(uname -s | tr A-Z a-z)
arch=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -sSL "https://github.com/truestamp/truestamp-cli/releases/download/v0.2.0/truestamp-cli_0.2.0_${os}_${arch}.tar.gz" | tar -xz
./truestamp version
```

### If the release fails partway

GoReleaser is mostly idempotent, but a partial failure (for example a tap push rejected) leaves the GitHub Release in place while the tap cask is out of date. To redo cleanly:

```sh
gh release delete v0.2.0 -y
git push origin :refs/tags/v0.2.0
git tag -d v0.2.0

# Fix the problem in a new commit, push to main, then retag from the fixed commit.
git tag -a v0.2.0 -m "v0.2.0 - ..."
git push origin v0.2.0
```

Do **not** re-tag a version that has already propagated to `proxy.golang.org` — the proxy caches tagged module versions forever. Bump the patch version (`v0.2.1`) instead.
