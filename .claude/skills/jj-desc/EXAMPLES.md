# Commit Message Examples

These examples are drawn from this repo's actual commit history and the
shape of changes it typically receives (CLI subcommands, crypto code,
GoReleaser config, CI pipelines).

## Good Examples

### Feature Addition

```
Add --type flag to verify subcommand

Accept an expected subject type (item, entropy_*, block, beacon)
and surface a "Subject Type" failure step when the proof's t code
disagrees. In --remote mode the flag is forwarded to the server's
/proof/verify endpoint, which rejects with subject_type_mismatch.

- Wire --type through cmd/verify.go to verify.Run options
- Infer --type from the truestamp-<stem>-<id>.<ext> filename
  convention when the flag is omitted
- Emit a faint stderr hint when the type is inferred so a
  mismatch is traceable to the inference

Lets users assert what they expected to verify, catching swapped
or renamed bundles before the cryptographic walk runs.
```

Why it works: 35-char subject in imperative mood, clear motivation,
specific bullets, benefit explained in closing.

### Security Fix

```
Fix self-upgrade backup vanishing for old binaries

The 7-day prune in replace_unix.go ran before the new backup was
written, so upgrades of binaries older than 7 days deleted the
just-created .bak.<rfc3339> file and left no rollback path.

- Move the prune step to run after the atomic rename completes
- Add regression test covering a 30-day-old binary upgrade
- Keep prune best-effort; surface failures only at -v
```

### Simple Bug Fix

```
Fix detached HEAD warning under jj workflow

The CLI repo is managed with jj; git reports "Not currently on
any branch" as the steady-state. Don't gate commits on that.
```

### Refactoring

```
Extract input resolver into internal/inputsrc

Move the six-mode input pattern (positional / --file [path] /
--file picker / --url [url] / --url prompt / stdin pipe) out of
cmd/verify.go into a shared package so create, hash, encode,
decode, jcs, and convert children can reuse it.

- Define Resolve and ResolveStream entry points
- Use FilePickSentinel and URLPromptSentinel constants for the
  pflag NoOptDefVal trick
- Cover all six modes with unit tests
```

### Mechanical Update

```
Bump Go toolchain 1.26.2 -> 1.26.3 to clear two stdlib CVEs
```

Single subject line, no body needed. The motivation IS the subject.

### Code Change with Dependency Bump

```
Add deterministic CBOR marshal to ProofBundle

Encode proof bundles to CBOR using CoreDetEncOptions (RFC 8949
§4.2: sorted keys, shortest-form ints, definite-length) with the
self-describing tag 55799 so round-trips are byte-stable and
IsCBORProof still recognises the output.

- Add MarshalCBOR method and per-field byte-string policy
- Re-encode pk, sig, all hashes, ep, ip as major-type-2 bytes
- Pin fxamacker/cbor/v2 in go.mod

A first pass through ParseCBOR -> MarshalCBOR may normalize a
non-deterministic source; a second pass is stable.
```

The dependency line gets one bullet — no need to enumerate every
`go.sum` change.

### Multiple Unrelated Changes

If a changeset has truly unrelated changes, consider splitting via
`jj split`. If that is not possible:

```
Refresh CI matrix and tighten goreleaser config

CI matrix:
- Drop go 1.25 from the test matrix
- Add windows-latest to the build job

GoReleaser:
- Pin homebrew tap commit SHA in release.yml
- Add Linux ARM64 to archives target list
```

### Large Refactoring

```
Restructure verify package for testability

Split the monolithic verify package into pure-logic and I/O
layers so the verification steps can be unit-tested without
touching the network.

- verify.go now returns a Report with no I/O
- presenter.go renders Reports via lipgloss
- json_output.go handles --json mode
- remote.go isolates the /proof/verify HTTP path

No functional changes; all existing CLI integration tests pass.
```

## Bad Examples

### Too Long Subject (61 chars)

```
Refactored the verify subcommand to support remote API verification
```

Fix: "Add remote verification mode to verify" (38 chars)

### Past Tense

```
Added passive upgrade check
```

Fix: "Add passive upgrade check"

### Gerund

```
Adding fuzz coverage to bitcoin parsers
```

Fix: "Add fuzz coverage to bitcoin parsers"

### Markdown in Body

```
Update upgrade subcommand

## Changes

**Added**:
- Install-method detection
- Cosign verification
```

Fix: Use plain text with "- " bullets, no headers or bold.

### Attribution Line

```
Fix vuln-check exit code

Return 2 on network error instead of 1.

Co-Authored-By: Claude <noreply@anthropic.com>
```

Fix: Remove attribution line entirely. The commit message is the
developer's own voice. Never credit AI tools in commit messages.

### Vague Subject

```
Update Taskfile
```

Fix: Be specific — "Add vuln-check to precommit task"
