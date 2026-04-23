# Failure recovery

Scenarios where a `/release` invocation fails partway through, and the exact recipe to get back to a clean state.

## Quick diagnosis

Start here. Run these three checks:

```bash
# 1. Does the GitHub Release exist?
gh release view vX.Y.Z --json tagName,assets,publishedAt 2>&1 | head -5

# 2. Is the tag on origin?
git ls-remote --tags origin 'refs/tags/vX.Y.Z'

# 3. Is there an open tap PR for this version?
gh pr list --repo truestamp/homebrew-tap --state open --json number,title,headRefName | \
  tee /tmp/tap-prs.json
```

Read the results:

| GitHub Release | Tag on origin | Tap PR open | Scenario |
| -------------- | ------------- | ----------- | -------- |
| ❌ missing     | ❌ missing    | ❌ none     | Nothing published. Start the playbook from Step 2. |
| ❌ missing     | ✅ present    | ❌ none     | **Scenario 1** — tag pushed, `ci` gate failed (or pipeline crashed early). |
| ❌ missing     | ✅ present    | ❌ none     | **Scenario 2a** — GoReleaser failed before publish. Same recipe as scenario 1. |
| ✅ present     | ✅ present    | ✅ dangling | **Scenario 2b** — tap merge step failed (marked `continue-on-error`). Recover without re-tagging. |
| ✅ present     | ✅ present    | ❌ merged   | Release is healthy — re-verify Step 11 of the playbook. If a specific check reports wrong (e.g., asset count < 14), **scenario 3**. |

## Scenario 1 — Tag pushed, `ci` gate failed

**Diagnosis:** tag is on origin, `release.yml` failed during the `ci` job (or any pre-GoReleaser step), no artifacts published.

**Recovery is safe** because nothing reached users. Delete the tag and restart cleanly.

```bash
# Remove the GitHub Release (no-op if one wasn't created).
gh release delete vX.Y.Z -y 2>/dev/null || true

# Delete the tag on origin.
git push origin :refs/tags/vX.Y.Z

# Delete the tag locally.
git tag -d vX.Y.Z
```

Then fix the root cause of the CI failure on `main` via a normal PR, wait for it to merge, restart `/release` from Step 2.

## Scenario 2 — Tag pushed, `ci` passed, GoReleaser failed

Three sub-scenarios based on where inside the GoReleaser job it fell over.

### 2a — Before publish (cross-compile, goreleaser check, snapshot, real build)

**Diagnosis:** `gh release view vX.Y.Z` reports no release. No tap PR was created.

Same recovery as Scenario 1. Safe to delete tag + restart.

### 2b — Publish succeeded, tap merge step failed

**Diagnosis:** `gh release view vX.Y.Z` succeeds and reports 14 assets. But `gh pr list --repo truestamp/homebrew-tap --state open` shows a dangling `goreleaser-X.Y.Z` PR, and the tap cask on main still shows the *previous* version.

The `Merge homebrew-tap PR` step in `release.yml` is marked `continue-on-error: true` by design — a flaky tap merge (e.g. GitHub API rate-limit hiccup, PAT token expiry) should NOT roll back a successful release. The recovery is a one-liner: merge the PR manually.

```bash
# Identify the dangling PR number and branch.
pr=$(gh pr list --repo truestamp/homebrew-tap --state open --head 'goreleaser-X.Y.Z' --json number -q '.[0].number')

# Merge it with --delete-branch, matching the workflow's behaviour.
gh pr merge "$pr" --repo truestamp/homebrew-tap --merge --delete-branch
```

Then rerun Step 11 of the playbook to confirm the cask is now at `NEW_VERSION`.

If the merge fails (e.g. `HOMEBREW_TAP_GITHUB_TOKEN` has expired, permission error):
1. Don't tamper with the tap cask file directly. GoReleaser owns it.
2. Rotate the PAT (see `CONTRIBUTING.md` "Prerequisites (one-time)" — fine-grained, scoped to `truestamp/homebrew-tap` only, `Contents: Read and write` + `Pull requests: Read and write`).
3. Retry the merge with the new token.

### 2c — Attestation step failed

**Diagnosis:** `gh release view vX.Y.Z` succeeds, tap PR merged, but `gh api repos/truestamp/truestamp-cli/attestations/vX.Y.Z` returns an error or empty list.

The `Attest build provenance` step runs *after* the release is already public. Its failure is the lowest-stakes — users get binaries + checksums + cosign bundle normally; the only missing piece is the SLSA build-provenance attestation that confirms which Actions workflow built the artifacts.

Options:
1. Re-run the failed step: `gh run rerun <run_id> --repo truestamp/truestamp-cli --failed`. This re-runs only failed steps in the existing workflow run.
2. Accept the gap for this release and document in the next patch release's CHANGELOG under `### Fixed`.

Don't delete the release — users already have the binaries.

## Scenario 3 — Step 11 verification discrepancy

**Diagnosis:** the workflow completed successfully (`gh run view $run_id --json conclusion -q '.conclusion'` returns `"success"`) but a specific Step 11 check reports unexpected state.

| Check that failed | Likely cause | Recovery |
| ----------------- | ------------ | -------- |
| Asset count ≠ 14 | `.goreleaser.yaml` changed the artifact set without updating docs | Update the playbook's Step 11 expected count, or restore the missing artifacts; depends on whether the change was intentional |
| Tap cask version wrong | Scenario 2b — tap merge silently failed despite workflow green | Apply Scenario 2b recipe |
| Tag shows unsigned on GitHub | Signing key misconfigured when tag was created | Delete tag (Scenario 1 recipe), fix `git config --get tag.gpgsign` / `git config --get user.signingkey`, re-tag and re-push |
| Signatures in `checksums.txt.sigstore` don't verify with cosign | OIDC token issue at signing time — rare | Re-run the release: delete everything per Scenario 1, investigate logs |

## Scenario 4 — Go proxy cached a broken version

**Diagnosis:** the release went public, got fetched by `go install …@vX.Y.Z` by at least one external user or CI job, the Go module proxy at `proxy.golang.org` cached the version as immutable, AND you want to publish a fix.

**You CANNOT safely re-tag the same version.** The proxy never evicts cached versions. Even if you delete the GitHub Release + tag and push a new tag with the same name, anyone resolving `@vX.Y.Z` will get the cached (broken) version.

Recovery: bump the patch and release again.

```bash
# If v0.7.1 is broken and cached by the proxy, cut v0.7.2 instead.
# Add a note in CHANGELOG for v0.7.2 explaining the reason (refer users to upgrade from v0.7.1 → v0.7.2).
```

Check whether the proxy has cached the broken version:

```bash
curl -s "https://proxy.golang.org/github.com/truestamp/truestamp-cli/@v/vX.Y.Z.info" | head
# If this returns a JSON object with a Version field, the proxy has cached it.
# If it returns 410 Gone or 404, you still have a chance to re-tag.
```

In practice, if you catch the broken release within a few minutes of the tag push AND nobody else has run `go install …@vX.Y.Z`, the proxy probably hasn't cached it yet. Delete fast (Scenario 1 recipe) and try again. But if `curl` shows a cached version, bump the patch.

## Last resort — manual everything

For any unrecoverable state, fall back to the recipe in [CONTRIBUTING.md](../../../CONTRIBUTING.md) "If the release fails partway" section. That's the authoritative manual recovery procedure; this file is its detailed complement, not its replacement.

## Prevention (for future releases)

Most failures caught here would have been caught earlier by:

- `task release-check` — malformed `.goreleaser.yaml` (caught in ~1s).
- `task release-snapshot` — cross-compile failures, missing SBOM tools, cask template errors (caught in ~1min).
- `task precommit-full` — lint / test / vuln regressions (caught in ~5min).

If a step in the `/release` playbook was skipped to "save time", this file is the cost. Run every pre-flight step every time.
