---
name: release
description: Cut a new truestamp-cli release end-to-end with full determinism. Walks through every documented step — pre-flight quality gate, GoReleaser dry-run, CHANGELOG update, release PR + CI, signed annotated tag, release.yml workflow verification, post-release GitHub Release / Homebrew tap / install.sh smoke checks — and reports status or actionable failure diagnostics at each step. Use this whenever the user says anything like "cut a release", "release this", "bump the version", "tag v<X>", "ship the next version", "time to release", "/release", or otherwise signals they want to publish a new version of truestamp-cli — even when they don't mention a specific version. This skill is the canonical way to ship a release; `CONTRIBUTING.md` points at it as the normal flow.
---

# /release — cut a new truestamp-cli release

The release process has many steps, and most of them are reversible only until the tag is pushed. After that, the release is visible to every user on every install channel (GitHub Releases, Homebrew tap, `install.sh`, Go proxy). This skill's job is to walk that gauntlet deterministically so every release goes out the same way, and to leave clear tracks when something fails mid-flight.

## Scope

### Use this skill when

The user wants to publish a new version to users. Phrasings that should trigger it: "cut a release", "release vX.Y.Z", "let's ship the next version", "bump the version", "tag this and release", "/release", "release truestamp-cli".

### Do NOT use this skill when

- The user is just pushing a code change to `main` (that's a regular PR, not a release).
- A prior release failed and the tag is already on origin — use `references/failure-recovery.md` directly; don't run the whole playbook again.
- The user wants to cut a pre-release / release candidate (`vX.Y.Z-rc.N`, `vX.Y.Z-beta`, etc.). The project's `release.yml` refuses pre-release tags by design (two-layer defence — GitHub `/releases/latest` filter + local semver-suffix check). Pause and discuss with the user before proceeding.
- The user wants to cut a MAJOR version bump (X.0.0). Pause and confirm — a major break needs a migration story, not just a tag push.

## Context this skill assumes

Before starting, know these facts about the truestamp-cli repo:

- **Colocated jj + git.** Use `jj` for local work (describe, bookmark, push); use `git` for annotated tags (jj doesn't create them).
- **Origin:** `truestamp/truestamp-cli`. **Homebrew tap:** `truestamp/homebrew-tap`.
- **`protect-main` ruleset active.** `main` rejects any push whose SHA doesn't have `Test (ubuntu-latest)` + `Test (macos-latest)` green. Linear history is enforced; force-push and deletion blocked. **This is why the release commit must go through a PR even when the only change is the CHANGELOG** — direct `jj git push --bookmark main` will be rejected.
- **Release trigger:** pushing a tag matching `v*` to origin triggers `.github/workflows/release.yml`, which runs:
  1. `ci` — reusable `workflow_call` into `ci.yml` on the tagged SHA (matrix: ubuntu + macos).
  2. `goreleaser` (gated on `needs: ci`) — `goreleaser check` → snapshot dry-run → real `goreleaser release` → Homebrew tap PR merge → SLSA build-provenance attestation.
- **Expected release.yml runtime:** 7–9 minutes total.
- **Tag signing:** `.git/config` has `tag.gpgsign=true` + `user.signingkey` set (ED25519 SSH key). `git tag -a` auto-signs; `git tag -v` verifies.

## The canonical playbook

Do every step in order. After each step, check the outcome and branch to failure handling if needed. Do NOT skip steps or batch them. Determinism matters — every release should follow exactly the same path.

### Step 0 — Detect current state

Before touching anything, figure out whether a release is already in flight. This makes the skill safe to re-invoke after a partial run.

```bash
git describe --tags --abbrev=0                          # last released version
git tag -l 'v*' | sort -V | tail -5                     # recent tags
jj status                                               # WC clean?
jj log -r 'main@origin' -T 'commit_id.short() ++ " " ++ description.first_line() ++ "\n"' --no-graph
gh pr list --state open --head 'release-v*' --json number,title,headRefName --repo truestamp/truestamp-cli
gh run list --workflow=release.yml --limit 3 --json databaseId,name,conclusion,status,headBranch
```

Interpret the results:

| Finding | Action |
| ------- | ------ |
| Open `release-vX.Y.Z` PR exists | Jump to Step 7 (wait-for-CI-and-merge); skip ahead |
| Tag `vX.Y.Z` already on origin and release.yml run is in progress | Jump to Step 10 (watch); skip ahead |
| Tag `vX.Y.Z` on origin, release.yml completed, artifacts missing | Go to `references/failure-recovery.md`; do NOT re-run the playbook |
| Clean state (no in-flight release) | Continue to Step 0.5 below |

Tell the user what you found before proceeding.

### Step 0.5 — Decide the version

Read `## [Unreleased]` in `CHANGELOG.md`.

- **If empty**: ask the user what they want to release. Either they haven't written release notes yet (stop and ask them to write them), or there's nothing meaningful to ship (stop).
- **If populated**: summarize the entries to the user (not every bullet — the *kinds* of changes: new features? bug fixes? infra? breaking?) and ask which version bump:
  - **PATCH** (increment Z): bug fixes, doc-only changes, internal hardening, dependency bumps with no API effect.
  - **MINOR** (increment Y, reset Z): new features, new subcommands, new config settings, new flags, backward-compatible behavior changes.
  - **MAJOR** (increment X, reset Y/Z): breaking changes. Pause and require explicit confirmation + discussion of migration path.

Compute `NEW_VERSION = X.Y.Z` (no leading `v`) based on the last tag (`git describe --tags --abbrev=0`) + the chosen bump. Confirm `NEW_VERSION` with the user before proceeding. This is a contract — the CLI is on semver and users rely on it.

### Step 1 — Pre-flight sanity checks

```bash
jj git fetch
jj status
jj log -r 'main..@' -T 'description.first_line() ++ "\n"' --no-graph
```

Required state:
- `jj status` reports `The working copy has no changes`.
- `jj log -r 'main..@'` shows only the default empty WC change (no unpushed real commits).
- `jj log -r main@origin` matches `jj log -r main` (local main is synced).

If anything is off, STOP and report to the user. The most common cause is uncommitted local work — that needs to land via normal PR before a release can proceed.

### Step 2 — Quality gate: `task precommit-full`

```bash
task precommit-full
```

This runs fmt + vet + staticcheck + gosec + race tests + fuzz seed-corpus replay + govulncheck + all-platform build. Takes 3–5 minutes. The whole point is that the release SHA has already been through this gate locally before it touches origin.

**On failure:** identify which step failed (the output names the task — `fmt`, `vet`, `lint`, `test-race`, `fuzz`, `vuln-check`, `build-all`). Report the failing step verbatim to the user with a one-line suggestion. Do NOT proceed.

| Failed task | Likely cause | Fix |
| ----------- | ------------ | --- |
| `fmt-check` | File not gofmt'd | `task fmt` |
| `vet` | Likely bug caught by stdlib vet | Read the diagnostic |
| `lint` (staticcheck) | Code-quality issue | Read the diagnostic |
| `lint` (gosec) | New security finding; check if it's a real issue or needs `#nosec` with justification | See existing gosec exclusions in `Taskfile.yml` `task lint` for the format |
| `test-race` | Real race or flake — run locally a few times to distinguish | Fix the race; don't paper over |
| `fuzz` | A seed input now triggers a regression | Investigate; don't silence |
| `vuln-check` | New CVE in a dep or in the stdlib (re-run after Go bump) | Bump the dep or the Go toolchain via `.tool-versions` |
| `build-all` | Platform-specific compile failure | Fix for the specific GOOS/GOARCH that broke |

### Step 3 — GoReleaser pre-flight

```bash
task release-check     # .goreleaser.yaml syntax — <1s
task release-snapshot  # full dry-run with --skip=sign,publish — ~1 min
```

`release-snapshot` produces `dist/` including the rendered Homebrew cask at `dist/homebrew/Casks/truestamp-cli.rb`.

**On failure:**

- `release-check` fails → `.goreleaser.yaml` has a syntax error. The CI release would also fail its `goreleaser check` step. Fix and restart.
- `release-snapshot` fails during cross-compile → a platform-specific build is broken. Report the failing target (`darwin_arm64`, `linux_amd64`, etc.) to the user.
- `release-snapshot` hangs at cosign signing → the task is missing `--skip=sign,publish` (should have been fixed in v0.7.1). Inspect `Taskfile.yml`; if the `release-snapshot` task still reads just `goreleaser release --snapshot --clean`, the fix was lost. Run directly: `mise exec -- goreleaser release --snapshot --clean --skip=sign,publish`.

### Step 4 — Inspect the rendered cask

```bash
cat dist/homebrew/Casks/truestamp-cli.rb
```

Required shape:
- `version "X.Y.Z-SNAPSHOT-<shortsha>"` — the `X.Y.Z` prefix matches the previous release (snapshot mode doesn't know the next tag; real release substitutes `NEW_VERSION`).
- Four `url` + `sha256` pairs: `darwin arm64`, `darwin amd64`, `linux arm64`, `linux amd64`.
- `binary "truestamp"`.
- Gatekeeper `caveats` block preserved (references `xattr -cr #{staged_path}`).

If any of these are missing, `.goreleaser.yaml` has drifted. Pause, investigate, and do NOT tag.

Clean up before committing:

```bash
rm -rf dist/ build/
```

### Step 5 — Update `CHANGELOG.md`

Replace:

```markdown
## [Unreleased]

## [PREV_VERSION] — …
```

with:

```markdown
## [Unreleased]

## [NEW_VERSION] — YYYY-MM-DD

<move the former [Unreleased] content here>

## [PREV_VERSION] — …
```

Where `YYYY-MM-DD` is today's UTC date.

**Do not add or edit release-note content in this step.** Only move what's already in `[Unreleased]`. If the user wants to revise the notes, they should stop the skill, edit `CHANGELOG.md`, and restart.

### Step 6 — Release commit + release branch

```bash
jj describe -m "Release vX.Y.Z"
jj bookmark create release-vX.Y.Z -r @
jj git push --bookmark release-vX.Y.Z
```

Critical: do NOT advance the local `main` bookmark, and do NOT try `jj git push --bookmark main` directly. The `protect-main` ruleset rejects pushes without green CI checks on the target SHA (the push returns `GH013: Repository rule violations found`). The CHANGELOG commit must land through a rebase-merge PR.

### Step 7 — Open PR, wait for CI, merge

**Checkpoint: confirm with the user before opening the PR.** This is the first action visible on GitHub.

```bash
gh pr create --base main --head release-vX.Y.Z \
  --title "Release vX.Y.Z" \
  --body "See CHANGELOG.md for the full release notes."
```

Record the returned PR number. Watch CI:

```bash
gh pr checks <PR> --watch --repo truestamp/truestamp-cli
```

Required green checks: `Test (ubuntu-latest)`, `Test (macos-latest)`. Other checks (`CodeQL`, `Analyze (Go)`, `Socket Security: Project Report`, `Socket Security: Pull Request Alerts`) will also run — CodeQL and the two Test jobs are ruleset-enforced.

**On CI failure on the release PR:** don't merge. Options:
1. Fix-forward: push more commits to `release-vX.Y.Z`.
2. Re-run a flaky check: `gh run rerun <run-id> --repo truestamp/truestamp-cli`.
3. Abandon the release if the fix needs to land on main first. Close the PR + delete the branch.

When CI is green:

```bash
gh pr merge <PR> --rebase --delete-branch --repo truestamp/truestamp-cli
RELEASE_SHA=$(gh pr view <PR> --json mergeCommit -q '.mergeCommit.oid')
echo "RELEASE_SHA=$RELEASE_SHA"
```

Rebase-merge (NOT merge-commit — the `required_linear_history` rule blocks merge commits).

### Step 8 — Sync jj to post-merge main

```bash
jj git fetch
jj bookmark set main -r main@origin --allow-backwards   # only needed if local main was ahead
jj log -r main -T 'commit_id.short() ++ " " ++ description.first_line() ++ "\n"' --no-graph
```

The log output's commit SHA must match `RELEASE_SHA`. If it doesn't, STOP — something went wrong with the merge.

Clean up orphan commits: `jj abandon <change_id>` for any divergent duplicates (jj logs them as `(divergent)` after a rebase-merge rewrites the SHA). Safe to abandon the pre-rebase version of the release commit; it's no longer reachable from main.

### Step 9 — Create and push the signed tag

**Checkpoint: confirm with the user before pushing the tag.** This is the point of no return — the tag triggers `release.yml` and the Go module proxy may cache the version.

```bash
# Tag at exactly RELEASE_SHA.
git tag -a vX.Y.Z -m "vX.Y.Z - one-line summary" $RELEASE_SHA

# Verify the tag is signed.
git tag -v vX.Y.Z 2>&1 | head -5
```

The `git tag -v` output MUST start with `Good "git" signature`. If it doesn't, STOP — the repo's `tag.gpgsign=true` + `user.signingkey` should auto-sign. Unsigned release tags are a trust regression.

One-line summary pattern: lead with the version, followed by the headline change. Under 70 chars. Examples from prior releases:
- `v0.6.0 - beacon as first-class subject + proof format t-byte restructure`
- `v0.7.0 - cosign_path config + release hardening + test parallelisation`
- `v0.7.1 - release workflow CI gate + docs hygiene`

Push:

```bash
git push origin vX.Y.Z
```

### Step 10 — Watch `release.yml`

```bash
run_id=$(gh run list --workflow=release.yml --limit 1 --json databaseId -q '.[0].databaseId')
echo "run_id=$run_id"
gh run watch "$run_id" --exit-status
```

Expected job structure (verify with `gh run view $run_id --json jobs`):

1. `CI gate / Test (ubuntu-latest)` — matrix job from the reusable `ci.yml` call.
2. `CI gate / Test (macos-latest)` — same.
3. `GoReleaser` — blocked by `needs: ci`; starts only after both above succeed.

Expected GoReleaser step sequence (verify with `gh run view $run_id --json jobs -q '.jobs[] | select(.name == "GoReleaser") | .steps[] | "\(.conclusion) \(.name)"'`):

1. `Checkout`
2. `Install tools via mise`
3. `goreleaser check`
4. `goreleaser release --snapshot (dry run)`
5. `Run GoReleaser` (the real release)
6. `Merge homebrew-tap PR` (marked `continue-on-error: true` — a flaky tap merge doesn't fail the whole release)
7. `Attest build provenance`

**On workflow failure:** route by which job failed.

- `ci` gate failed → nothing was published. Clean up per `references/failure-recovery.md` scenario 1.
- `GoReleaser` failed → partial state is possible. See `references/failure-recovery.md` scenario 2.

### Step 11 — Verify every release output

Run all of these and report each result to the user:

```bash
# 1. GitHub Release — expect 14 assets.
gh release view vX.Y.Z --json tagName,assets -q '{tag: .tagName, asset_count: (.assets | length), assets: [.assets[].name]}'
# Expected: tag == "vX.Y.Z", asset_count == 14. Assets:
#   checksums.txt + checksums.txt.sigstore
#   + 6 archives (darwin arm64/amd64 .tar.gz, linux arm64/amd64 .tar.gz, windows arm64/amd64 .zip)
#   + 6 matching .sbom.json files

# 2. Homebrew tap: no dangling PRs.
gh pr list --repo truestamp/homebrew-tap --state open --json number
# Expected: []

# 3. Homebrew tap: last merged PR is the one for THIS release.
gh pr list --repo truestamp/homebrew-tap --state merged --limit 1 --json title,mergedAt
# Expected title contains "vX.Y.Z".

# 4. Homebrew tap cask on main matches NEW_VERSION.
gh api repos/truestamp/homebrew-tap/contents/Casks/truestamp-cli.rb -q '.content' | base64 -d | grep '^  version'
# Expected: version "X.Y.Z"

# 5. Tag still signed on origin.
git fetch origin --tags
git tag -v vX.Y.Z 2>&1 | head -3
# Expected: Good "git" signature ...

# 6. Every GoReleaser step succeeded.
gh run view "$run_id" --json jobs -q '.jobs[] | select(.name == "GoReleaser") | .steps[] | select(.conclusion != "success") | {name, conclusion}'
# Expected: (empty output — no failures)

# 7. Build-provenance attestation landed.
gh api repos/truestamp/truestamp-cli/attestations/vX.Y.Z 2>&1 | head -3 || true
```

If any check fails, say which one, and point at the failure-recovery reference.

### Step 12 — Success report

When every Step 11 check passes, produce this structured report to the user:

```
✅ truestamp-cli vX.Y.Z released

Release commit : <RELEASE_SHA>
Tag            : vX.Y.Z (signed, ED25519)
Workflow run   : https://github.com/truestamp/truestamp-cli/actions/runs/<run_id>
GitHub Release : https://github.com/truestamp/truestamp-cli/releases/tag/vX.Y.Z
Assets         : 14 (checksums + sigstore bundle + 6 archives + 6 SBOMs)
Homebrew tap   : version "X.Y.Z" on main, PR #<NN> merged
Attestation    : SLSA build-provenance published

Install / upgrade channels:
  brew upgrade truestamp/tap/truestamp-cli
  curl -fsSL https://get.truestamp.com/install.sh | sh
  go install github.com/truestamp/truestamp-cli/cmd/truestamp@vX.Y.Z
  Direct tarballs: <GitHub Release URL>

Total wall-clock time from tag push to "done": <actual minutes> (expected 7–9).
```

If any Step 11 check failed but the release is recoverable, also include:

```
⚠ Partial release detected: <which check failed>
See .claude/skills/release/references/failure-recovery.md scenario <N>.
Recommended action: <one-line recovery step>.
```

## Checkpoints requiring user confirmation

These are points where Claude must pause and get an explicit OK from the user before continuing, because the action is visible to others or hard to reverse:

1. **Version bump decision** (Step 0.5) — user must confirm `NEW_VERSION`.
2. **Opening the PR** (Step 7) — the PR appears on GitHub.
3. **Merging the PR** (Step 7) — the CHANGELOG commit lands on main.
4. **Pushing the tag** (Step 9) — triggers release.yml; the Go proxy may cache the version.

Every other step is either local-only or read-only and can run without confirmation.

## Determinism invariants

The skill relies on these being true throughout:

- `RELEASE_SHA` (the merged release commit SHA) is computed once at Step 7 and reused through Step 9. Don't recompute from `main` at a later step — if anyone else pushes to main between merge and tag, your local and remote will diverge.
- The tag is created with `git tag -a` at exactly `RELEASE_SHA`, not at `HEAD`. This matters in case the working tree has moved.
- Every verification in Step 11 passes before reporting success. Do NOT report success on partial verification; route to `references/failure-recovery.md` instead.

## Failure recovery

See [references/failure-recovery.md](references/failure-recovery.md) for:

- Tag pushed, `ci` gate failed (scenario 1).
- Tag pushed, GoReleaser failed (scenarios 2a/2b/2c).
- Verification discrepancy in Step 11 (scenario 3).
- Go module proxy cached a broken version (cannot re-tag — scenario 4).

## What this skill deliberately doesn't do

- **Major version bumps** — pause for user input; a major requires migration docs, not just a tag.
- **Pre-release tags** — release.yml has two layers of defence that refuse them.
- **Backporting** — if a hotfix needs to go to an older release line, that's a separate manual process.
- **Windows in-place upgrade** — upgrade.go stub-errors on Windows; users there must go through `go install`.
- **Manual tap cask edits** — GoReleaser owns the cask. Never hand-edit `truestamp/homebrew-tap/Casks/truestamp-cli.rb`.
