# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.12.0] — 2026-07-28

Conformance pass against Appendix E of the Truestamp whitepaper, the
normative specification for an independent proof verifier. The headline
change is that a bundle's verdict is now a function of the bundle and
nothing else: not the filename it arrived under, not which wire encoding
carried it, and not whether a third-party service happened to answer.

### Added

- **`--json` now carries the complete step record.** A new top-level
  `steps` array reports every check with its `group`, `category`,
  `status` and `message`, in Appendix E.22's category order. The two
  filtered views beside it drop passing rows (`issues` keeps failures,
  warnings and skips; `verification_notes` keeps the Verification Notes
  group plus every info row), so three of the whitepaper's fourteen
  reference rows previously survived `--json` as nothing but a count.
  `hash_comparison` is now always emitted and gained distinct `supplied`
  and `matched` booleans, because "no expected hash was given" and "one
  was given and did not match" are different facts. Also new for script
  consumers: a top-level `signatures_checked` boolean, an `info` count
  inside `summary`, an `external_check` string
  (`confirmed` / `skipped` / `failed`) on each commitment, and a
  `skipped` value for `issues[].severity`.
- **`--json` reports a structural refusal as data.** A bundle refused
  before any check runs now emits
  `{"result":"rejected","rejection":{"code":...,"detail":...}}` with a
  stable identifier from the whitepaper's error taxonomy
  (`not_a_json_object`, `invalid_subject_type_code`, `missing_block`,
  ...) instead of an English sentence, so independent verifiers can be
  compared on the identifier. Exit code is still 1, and human output is
  unchanged.
- **Bare-map CBOR proof bundles are accepted.** Appendix E.3 requires a
  verifier to read CBOR both wrapped in the self-describing tag 55799
  and as a bare map. Only the tagged form was recognized before, so an
  untagged bundle fell through to the JSON parser and was refused.
  `verify`, `convert proof` and every other CBOR reader now take both,
  and the JSON, tag-55799 CBOR and bare-map CBOR encodings of one
  bundle produce byte-identical reports.
- **Portability reporting for large integers.** A document carrying an
  integer outside +/- 2^53 is not portably verifiable by a strict
  RFC 8785 implementation, which Appendix E.4 requires a verifier to
  say rather than hide. `truestamp jcs` and `truestamp hash --jcs`
  print an advisory stderr line naming the offending literals (exit
  code unchanged, suppressed under `--silent`, replaced by an
  `oversized_integers` field under `--json`), and `truestamp verify`
  surfaces it as a warning row on the Subject Data step.
- **`truestamp create` refuses claims it could never verify back.** An
  integer outside +/- (2^53 - 1) anywhere in the claims is rejected
  before the network call, naming every offending path, because a
  verifier that parses JSON numbers as IEEE-754 doubles would derive a
  different `claims_hash`. `--json` emits
  `{"error":"unsafe_integer","message":...,"violations":[...]}` with
  the value and the bounds as JSON strings, so a consumer that parses
  with doubles cannot round the very numbers being reported. Claims
  that cannot be canonicalized under RFC 8785 at all (a float literal
  overflowing a double, for instance) are refused for the same reason,
  and a claims document carrying trailing content after the object, or
  a bare `null`, is now an error instead of a silently accepted prefix.
- **Two informational rows naming which temporal edge a bundle can
  establish.** Appendix E.20 requires a verifier that has not retrieved
  the previous block and its entropy leaves to report the
  submitted-after edge as *not established from the bundle*, never as
  established. `Submitted Before` names the commitment transaction
  that, once confirmed on chain, establishes the earlier edge;
  `Submitted After` says plainly that the later edge cannot be
  established from a bundle alone. Both are `info` rows and cannot move
  a verdict. The Submission Window pass also now carries E.20's
  qualifier, "asserted by Truestamp, not externally verified".

### Changed

- **Report step groups follow the whitepaper's normative table.**
  `Temporal Window` is now `Submission Window`; the keyring cross-check
  moved out of `Signing Key` into its own `Key Binding` group (decoding
  a public key and binding it to Truestamp are separate claims, and
  only the second needs the network); a `Temporal Info` timing row
  carries the display timeline; and the "file hash not verified"
  advisory moved from `Verification Notes` onto `Hash Comparison`.
  Scripts that match on group names in `--json` output must be updated.
- **`truestamp verify` no longer infers `--type` from the input
  filename or URL basename.** A subject type is read only from the
  bundle's signed `t` field (Appendix E.24). The previous behaviour
  made the verdict a function of the filename in both directions: a
  name matching the `truestamp-<stem>-<id>.<ext>` download convention
  silently asserted a type, failing a cryptographically sound proof
  when the two disagreed, and renaming away from that convention
  silently dropped the assertion. A file named
  `truestamp-beacon-<id>.json` may legitimately carry `t = 10`: the
  beacon show page generates exactly that, and so does the sample
  bundle shipped in this repo, so the common case was a false failure.
  It also reached the wire: the inferred type was posted as `data.type` on
  `--remote`, where the server's rejection surfaced as an opaque API
  error with no report at all. Pass `--type` explicitly to keep the
  assertion; with it unset no Subject Type row is emitted, nothing is
  posted, and the `(inferred --type ...)` stderr hint is gone. The swap
  the inference claimed to catch is already caught cryptographically:
  `t` is inside the signed payload, so relabelling a bundle breaks its
  signature.
- **Hash fields must carry Appendix E.4's lowercase-hex encoding.** Go's
  hex decoder accepts either case, so a bundle whose `mh`, `ph`, `mr`
  or `kid` had been upper-cased verified as sound while its bytes
  differed from the canonical form the producer emits. Uppercase is now
  a graded step failure naming the field and the offset rather than a
  parse abort, so the rest of the report still renders.
- **`--hash` on a subject that commits to no file hash is a skip, not a
  failure.** Supplying `--hash` for an entropy or block proof reports a
  visible skip explaining that the flag does not apply, and contributes
  nothing to the verdict.
- **External confirmation is tri-state.** A commitment records whether
  its external lookup was confirmed, skipped, or ran and failed; the
  terminal render distinguishes `(external verification skipped)` from
  `(external verification failed)`. Previously any non-confirmation
  rendered identically, and `--json`'s `externally_verified` was
  unconditionally `true` on the remote path.
- **The proof-bundle hash-shape check reports as info, not pass**,
  matching the whitepaper's reference report.
- **The `Structure` step now asserts Appendix E.8's `v == 1` and nothing
  else.** The four other propositions it used to carry (registered `t`,
  block present, block id and Merkle root present, non-empty `cx`) all
  became hard rejections under E.6, where they abort before any step
  runs. **`Signing Key` makes no comparison** against the bundle's
  stored key ids either, because key rotation legitimately makes them
  differ; subject-kid tampering is still caught, because `kid` is an
  input to the composite hash.
- **`verify --remote` asserts `--type` client-side** and forwards it to
  the server only when the assertion already holds. Forwarding a
  mismatching type made the server answer 4xx and the caller got a bare
  error string with no report, the opposite of what the flag is for. A
  mismatch now produces the same `Subject Type` failure row local mode
  produces, alongside everything else the server checked.

### Removed

- **Dead cryptographic code that implemented the wrong thing.**
  `tscrypto.ComputeProofHash` built the *pre-restructure* signature
  payload and sat in the same file, under a near-identical doc comment,
  as the correct one, with zero production callers and six test
  references, so the wrong format was actively pinned by the suite. A
  future editor who called it would have got a silently wrong signature
  payload with green tests. Removed alongside `ComputeCommitmentHash`,
  `ComputeCommitmentDataHash`, `ComputeEntropyMetadataHash`,
  `LenPrefix`, `proof.CheckHexShape` and `proof.InspectBundleType`, all
  of which had the same profile. The `0x12` / `0x22` / `0x33` / `0x34`
  / `0x35` prefix constants stay: they are producer-side entries of a
  frozen registry mirrored whole on purpose, and a constant cannot be
  called with the wrong arguments.

### Fixed

- **Malformed CBOR-only bundles that verified as sound proofs are now
  refused.** All predate this work. The type code `t` was read into a
  16-bit field, so `t = 65546` truncated to `10` and a bundle claiming
  an unregistered subject type verified, with exit 0, as a block. A
  non-integer `v` (`1.9`) or `t` (`20.5`) passed the version and type
  checks that their JSON equivalents fail. Wrapping `s.d` in the
  self-describing tag 55799 produced an unbounded family of
  byte-distinct bundles all valid under one signature. Duplicate map
  keys inside `s.d` gave one signed blob two readings, and the JSON
  path rejected the same document. Hex *text* was accepted where
  Appendix E.3 requires a byte string, and indefinite-length strings
  gave every field a second spelling. Invalid UTF-8 inside an `s.d`
  text string was silently replaced with U+FFFD and fed to the `0x11`
  preimage. The CBOR and JSON paths now accept the same value space.
- **`verify --remote` no longer fails open.** Four separate paths let a
  server-reported failure print `VERIFIED` at exit 0: a step object
  carrying no `status` key scored as a pass (Go's zero value), a status
  outside Appendix E.22's five silently became `info`, the server's own
  top-level `passed` verdict was parsed and never read, and a `result`
  carrying zero steps was treated as a clean run. Each is now a failure
  with a `Server Verdict` row saying which. Separately, `--remote`
  never performed E.7's `--hash` comparison at all:
  `hash_comparison.matched: false` and `result: "verified"` coexisted
  in one document. The client now performs the comparison itself, on
  the bundle it parsed, and treats the server's `hash_matched` as
  corroboration rather than as the answer.
- **`--skip-signatures` no longer prints "VERIFIED - proof is valid" for
  a bundle with a forged signature.** The verdict line now discloses
  that the signature was not checked, and a warn row states that the
  run establishes nothing about who issued the proof. The Signing Key
  step still runs under the flag, deliberately: skipping it would let
  an undecodable `pk` produce a report with no failure in it.
- **The report attributes the signature to the key that actually signed
  it.** It reported `b.kid`, the block's stored key id, which under the
  key rotation Appendix E.9 explicitly blesses is a different key from
  the one the signature verifies against. The signer is now the
  `pk`-derived key id everywhere it is described as such; `b.kid` is
  still reported verbatim where the block itself is being described.
- **`--hash` reports what was supplied separately from whether it
  matched.** `hash_comparison.supplied` was `false` on every run where
  the comparison did not take place, including runs where a hash *was*
  supplied, the exact conflation the field was added to prevent.
  Supplying `--hash` for an item that timestamped its claims content
  directly (and therefore carries no `s.d.hash`) also fabricated a
  `HASH MISMATCH` against an empty string; it is now a skip that says
  why.
- **An upstream service that answered about the wrong thing no longer
  decides a verdict.** A Blockstream response naming a *different*
  block was reported as `confirmed` with `externally_verified: true`,
  while a 2xx body omitting `height` failed a sound proof. A Horizon
  transaction response carrying no `memo_type` was graded as a chain
  disagreement. A 200 that is not a keyring document at all (`{}`,
  `null`, `{"version":"1"}`) failed the proof, where the reference
  verifier skips. Each is now an identity or shape check that grades
  the response unusable, a skip, leaving only a source that answered
  *about the right object* and disagreed able to fail a proof.
- **An unreachable external service no longer fails a sound proof.**
  Keyring, Horizon, Blockstream and NIST lookups are graded on a typed
  availability taxonomy: only a service that answers and *disagrees*
  fails a step. Transport errors, timeouts, 5xx, 429, and (for an
  entropy source) 404 are reported as skips. A Stellar commitment whose
  network is absent or unrecognized resolves to the testnet Horizon
  rather than failing unverified.
- **Fail messages no longer assert things the run did not establish.**
  The E.7 advisory claimed "the proof itself is verified" inside
  reports that went on to fail and in runs where no signature was
  checked. A signature step that aborted before verifying anything was
  explained as "may have been tampered with". A Bitcoin merkle failure
  was explained as an item-inclusion failure, in reports whose
  Inclusion Proof row passed. A skipped Bitcoin lookup said "no public
  API for " with an empty network name. An Inclusion Proof pass named
  an absent block (`Inclusion proof to block  (2 steps)`). E.20's
  ordering violation read `X is AFTER committed block time X` at second
  resolution because both timestamps were rendered without their
  milliseconds.
- **The Block Hash step checks preimage width, not just field
  presence.** A bundle with short `b` fields produced "Block hash
  derived (0x32)" over a 141-byte preimage where Appendix E.14 frames
  157.
- **Bundles the parser used to refuse now produce a report.** Only the
  hard rejections Appendix E.6 enumerates abort verification; a wrong
  or missing bundle version, a malformed key or signature, and
  wrong-sized hash fields are now graded as failing report steps
  instead of aborting with no report at all. Conversely a missing `ep`,
  a missing commitment root key, and several other structural defects
  that used to slip through are now refused outright.
- **Hash and digest comparisons are constant-time**, including E.21's
  NIST `outputValue`, which was also still case-sensitive. The Bitcoin
  OP_RETURN extractor now takes the push data of the first `0x6a`
  output, per Appendix E.19, rather than scanning past a malformed or
  oversized one to a later output that happens to look well-formed.
- **`truestamp jcs` and `truestamp hash --jcs` no longer canonicalize
  RFC-8259-invalid JSON into a different valid document.** Unescaped
  control characters inside strings, and documents nested past the
  decoder's frame limit, are now rejected rather than silently
  rewritten.
- **`truestamp jcs` and `truestamp hash --jcs` no longer round integers
  larger than 2^53.** Both are documented as the way to recompute a
  `claims_hash` locally, and both round-tripped every JSON number
  through an IEEE-754 double, so `9007199254740993` canonicalized as
  `9007199254740992` and produced the wrong digest. Canonicalization
  now runs through the new `internal/jcs` package, which preserves such
  literals verbatim (matching the Truestamp producer and the
  whitepaper's Appendix C.2a vectors) while delegating key ordering,
  string escaping and float formatting unchanged to the underlying
  RFC 8785 library. Output for any document inside the safe range is
  byte-identical to before, so no existing digest moves.
- **`truestamp convert proof --to json` no longer rewrites numbers or
  reorders keys.** Pretty output is the default and it re-parsed the
  bundle through a generic decoder, rounding integers above 2^53 and
  re-sorting object members alphabetically. It now re-indents the
  marshaled bytes without re-encoding them, so the output is a faithful
  rendering of the wire bundle and agrees with `--compact`.
- **`truestamp verify` no longer invents a portability warning for a
  CBOR bundle carrying a large float**, and such a bundle survives a
  CBOR to JSON to CBOR round trip instead of becoming non-re-encodable.

### Developer experience / tests

- **A conformance suite that pins the report, not just the exit code.**
  `TestAppendixD4_Conformance` runs the vendored Appendix D.1 bundle
  and checks Appendix E.25's one-way containment against the D.4 table;
  a golden report additionally pins every row's message text, which
  containment by construction cannot see; and a set of status-direction
  tests assert that a `fail` cannot be downgraded to a `skip`, that a
  `skip` cannot be promoted to a `pass`, and that no failure message
  reads as a positive assertion.

### Documentation

- Corrected several long-stale claims in `CLAUDE.md`: the entry point is
  `cmd/truestamp/main.go` (not a repo-root `main.go`); the global-flag
  table documented `--api-url` and `--keyring-url`, which the binary
  rejects in favour of the single `--base-url` origin every service URL
  derives from (consolidated back in 0.8.0); the logging env vars are
  `TRUESTAMP_LOGGING_LEVEL` / `TRUESTAMP_LOGGING_FILE`, named after the
  config section rather than the flag; Stellar's `net` is not strict;
  the fuzz-target inventory was 12 targets short; and
  `internal/hashing`'s 14-algorithm registry and `internal/tscrypto`'s
  12-entry wire registry are deliberately different and must not be
  reconciled. The same flag corrections were applied to `README.md` and
  `EXAMPLES.md`.
- A second documentation pass checked every documented flag, environment
  variable, command and file path against the built binary and the
  filesystem, and corrected what disagreed: the Verification Steps list
  described a `Signing Key` step that compares key ids and skips under
  `--skip-signatures` (it does neither) and a `Structure` step carrying
  four checks it no longer makes; the `download --type` vocabulary was
  documented as `auto|item|entropy|block|beacon`, three of which the
  binary rejects, and the same file contradicted itself two hundred
  lines later; the signature payload's `kid` slot was documented as
  `b.kid` hex-decoded, which is what Appendix E.9 explicitly forbids and
  would reject every rotated-key proof; the default log file was named
  `console.log` rather than `truestamp.log`; and `README.md` showed a
  `truestamp download <uuidv7>` invocation that exits with an error.
- **The egress table now lists the NIST Randomness Beacon.** `CLAUDE.md`'s
  "External API Calls" table and `EXAMPLES.md`'s offline section both
  listed three outbound destinations. There are four: an `entropy_nist`
  bundle is verified against `beacon.nist.gov` by default. That table is
  the one place a security reviewer looks to answer "what does this
  binary talk to?".
- **Every cross-repository reference was re-pointed.** The
  `truestamp-v2/docs/` tree these documents cited no longer exists; its
  contents moved to that repo's `kb/` knowledge base and `whitepaper/`
  directory. `CLAUDE.md` now names `whitepaper/whitepaper.typ`
  (Appendix E) as the normative specification and
  `whitepaper/verify_proof.exs` as the reference verifier, with the
  byte-prefix registry, wire format, beacon API and console WebSocket
  protocol pointing at their `kb/` locations. Every path was verified to
  exist.

## [0.11.1] — 2026-07-23

### Changed
- **Dependency and toolchain refresh.** Brought the bundled Go modules up to
  their latest compatible versions — notably `golang.org/x/crypto` and the
  Charm TUI libraries (bubbletea / bubbles / lipgloss) — and migrated the
  Bitcoin commitment parsing onto btcsuite/btcd's v0.26.0 `/v2` submodules.
  The btcd change is a behavior-preserving module restructure (same package
  APIs), verified byte-identical against the real-transaction, txoutproof, and
  BIP 37 partial-merkle-tree test vectors. No user-facing behavior changes.
- **Build and CI hardening.** Refreshed the pinned developer toolchain
  (goreleaser, cosign, syft, gosec, govulncheck) and the GitHub Actions, and
  made CI run the mise-pinned lint / vulnerability tools through the Taskfile
  tasks instead of `go install ...@latest`, so CI and local checks share a
  single source of truth for tool versions and can no longer drift.

## [0.11.0] — 2026-07-23

### Changed
- **Entropy observation timestamp is now `inserted_at`, following a server
  rename.** The Truestamp service renamed the entropy observation's record
  timestamp from `captured_at` to `inserted_at` (the Ash default). It is the
  same value — the moment Truestamp recorded the observation — and no hash or
  signature covers it, so proof verification is completely unaffected.
  `truestamp verify --remote` now reads the server's `inserted_at` temporal
  key, and the `--json` verification output emits the timeline timestamp under
  `inserted_at` instead of `captured_at`. No dual-key back-compat is kept.
  Scripts that parse `.timeline.captured_at` from `--json` output must switch
  to `.timeline.inserted_at`. The entropy subject's own `captured_at` (the
  upstream NIST pulse / Bitcoin block / Stellar close time) is a distinct
  value the server did not rename and keeps its name.

### Security
- **Go toolchain bumped to 1.26.5.** Picks up the standard-library
  `crypto/tls` fix for GO-2026-5856 (an Encrypted Client Hello privacy leak).
  Release binaries and `go install` builds are now produced with the patched
  toolchain.

## [0.10.0] — 2026-06-24

### Added
- **Create teams from the CLI and the console.** A new `truestamp team
  create [name]` subcommand and a keyboard-driven create-team modal in the
  console Teams pane (open with `c`) both create a team via
  `POST /api/json/teams` — the authenticated user becomes the owner and the
  owner membership is created server-side. The subcommand takes
  `--ownership-model creator_retains|team_retains`, `--set` (make the new
  team active), and `--json` (including a parseable error object on
  failure); with no name on an interactive terminal it prompts for the name
  and ownership model. Creation is plan-gated server-side: the CLI attempts
  the create and renders the server's self-describing error, discriminating
  the plan-limit rejection (`teams.ErrTeamLimitReached`) from the
  ownership-entitlement rejection (`teams.ErrOwnershipNotEntitled`, marked by
  an `ownership_model` JSON:API `source.pointer`) and surfacing the server's
  actionable detail under a tailored banner. (Previously team creation was
  web-only.)

### Fixed
- **Console: keys typed into the create-team name field no longer switch
  panes.** While the create modal is open it is fully modal — the numeric
  pane-jump keys (`1`-`4`), `[` / `]`, and `?` land in the name field (so a
  team named "Q3" is typable); `ctrl+tab` / `ctrl+shift+tab` still switch
  panes, and `ctrl+c` still quits.

## [0.9.0] — 2026-06-24

### Added
- **OAuth 2.1 browser sign-in, now the primary authentication method.**
  `truestamp auth login` opens the browser and runs a loopback
  Authorization Code + PKCE (S256) flow, storing a short-lived access
  token plus a rotating refresh token in the OS keychain (with a 0600
  file fallback). A new `internal/auth` package unifies OAuth and the
  long-lived API key behind a single process-wide `Authorizer`:
  discovery-driven endpoints (RFC 8414) validated with issuer and
  same-origin pinning, a self-managed token cache that refreshes
  proactively and on demand, automatic single-use refresh-token
  rotation, and a reactive 401 → refresh → retry-once transport on the
  shared HTTP client. The six existing API call sites (`auth`, `team`,
  `beacon`, `create`, `download`, `verify --remote`) and the `console`
  WebSocket all authenticate through it. `auth login --api-key` keeps
  the interactive paste-a-key path; `auth logout` revokes the refresh
  token (RFC 7009) and clears the session; `auth status` and
  `config show` report the active credential, scopes, and token expiry.
- **Console WebSocket OAuth with an in-band keep-alive.** The console
  authenticates on the upgrade with an `access_token` query param and
  keeps a long session alive across token rotation without
  reconnecting: it proactively refreshes and pushes `token.refresh`
  over the live socket before expiry, with the server's `token_expired`
  push as a reactive refresh-and-redial safety net and a
  stop-and-prompt-re-login path for a dead session.

### Changed
- **Authentication is OAuth-first, API-key-second.** A long-lived API
  key remains the CI/headless path via `--api-key` or
  `TRUESTAMP_API_KEY`; an explicitly-provided key takes precedence over
  a stored OAuth session so automated environments stay deterministic.
  Help text, `config show`, and the "not authenticated" guidance across
  `team` / `beacon` / `create` / `download` / `verify` / `console` were
  reworded for the new model. Secret redaction was extended to OAuth
  access/refresh tokens, authorization codes, and PKCE verifiers in both
  query-string and JSON forms. New fuzz targets `FuzzDiscoveryValidate`
  and `FuzzSessionDecode` harden the discovery-document and token-store
  parsers.

### Security
- New dependencies `golang.org/x/oauth2`, `github.com/zalando/go-keyring`,
  and `github.com/cli/browser`. Refresh tokens live in the OS keychain
  (0600 file fallback), never in `config.toml`; access tokens are
  short-lived; the loopback callback enforces the PKCE `state` before
  accepting an authorization code; and the discovery document's endpoints
  are pinned to the configured origin.

## [0.8.2] — 2026-06-11

### Security
- **Go toolchain 1.26.3 → 1.26.4.** Picks up three stdlib CVE fixes
  that all sit on this CLI's HTTPS client path: CVE-2026-42504
  (net/textproto — attacker-controlled server could inject unescaped
  content into errors surfaced by net/http clients reading MIME
  headers), CVE-2026-42507 (mime — CPU exhaustion decoding malicious
  MIME headers), and CVE-2026-27145 (crypto/x509 — quadratic
  `VerifyHostname` cost on large DNS SAN lists during TLS
  verification). `task vuln-check` is clean on 1.26.4.

### Changed
- **Toolchain pins bumped in `.tool-versions`** after vetting each
  upstream changelog: goreleaser 2.15.2 → 2.16.0 (XZ archive support,
  secret-redaction hardening in 2.15.3; the newly-deprecated `brews:`
  config section does not affect us — `.goreleaser.yaml` already uses
  `homebrew_casks`, and `goreleaser check` validates clean), cosign
  3.0.6 → 3.1.1 (Sigstore bundle format now the default; legacy
  detached-signature `verify-blob` used by `truestamp upgrade` and
  `install.sh` keeps working in v3 — flag removals land in cosign v4),
  syft 1.42.4 → 1.45.1, caddy 2.11.2 → 2.11.4 (upstream security
  patches; dev-docs server only). shellcheck and websocat were already
  current.
- **Go module dependencies updated across the board**: bubbletea
  v2.0.7 (mouse-handling race fixes that directly benefit
  `truestamp console`), koanf v2.3.5, x/crypto v0.53.0, x/mod v0.37.0,
  x/term v0.44.0, and notable indirects btcec v2.5.0 + btcutil v1.2.0
  (first releases on btcsuite's post-module-split graph — the new
  `chainhash/v2` module enters the graph via btcec; the breaking
  `*/v2.0.0` btcsuite line was deliberately *not* taken), fsnotify
  v1.10.1, x/sys v0.46.0. No breaking changes per upstream release
  notes. Verified with the full test suite between each update group,
  `task precommit-full`, and `task fuzz-deep` across all 64 fuzz
  targets with no new reproducers.
- **Go lint/vuln tooling pinned via mise** (`staticcheck`, `gosec`,
  `govulncheck` in `.tool-versions`) instead of `go install ...@latest`,
  keeping `task lint` and `task vuln-check` reproducible across
  machines and Go toolchain bumps (#19).

## [0.8.1] — 2026-05-27

### Fixed
- **Self-upgrade backup vanished for binaries older than 7 days.**
  `selfupgrade.Replace` creates the timestamped `.bak.<rfc3339>`
  rollback file via `os.Rename(destPath, backupPath)`. On Unix,
  `rename(2)` leaves the file's mtime untouched, so the backup
  arrived carrying the previous binary's install-time mtime. When
  the previous install was more than 7 days old — the common case
  for users upgrading between releases — the `pruneOldBackups` call
  that ends `Replace` reaped the freshly-created backup before
  `Replace` returned. The CLI continued to announce the backup path
  on stderr while the file was already gone, silently breaking the
  manual-rollback safety net. Fix: `os.Chtimes(backupPath, now, now)`
  immediately after the backup is in place, so its mtime reflects
  the actual backup-creation time. Regression test
  `TestReplace_BackupSurvivesAgedPriorBinary` backdates the source
  file 30 days before calling `Replace` and asserts the backup
  survives with the old contents and a refreshed mtime — the
  failure shape the previous test
  (`TestReplace_WithExistingBinary_CreatesBackup`) couldn't catch
  because its `os.WriteFile`'d destination always had a near-now
  mtime well inside the 7-day window. Discovered while verifying
  the v0.8.0 release upgrade against a 33-day-old `install.sh`
  binary.

## [0.8.0] — 2026-05-27

### Added
- **`truestamp console` interactive TUI.** New subcommand opens a
  long-lived, authenticated WebSocket to the Truestamp backend
  (multiplexed Phoenix Channels: `console:lobby` for commands +
  stream events, `console:clock` for server-time ticks) and renders
  four panes — **Monitor** (toggleable stream subscriptions with a
  scrollable, reversible event waterfall; 24h time-windowed
  retention, 100k hard cap), **New Item** (form + live
  `items.created → items.committed` lifecycle card with a leading
  "Submission mode" Select for external-hash vs.
  claims-as-source-of-truth), **Teams** (membership list + in-place
  team switch via `scope.switch_team`), and **Connection**
  (diagnostics with reconnect countdown, server-time clock, push
  counters, log file path, and a live third-party health-check
  table). Reconnect-with-backoff (1→2→5→10→30s), server-side
  first-event-immediate event coalescing into `<resource>.burst`
  summaries, two-stage readiness gate (socketReady / sessionReady).
  Architecture and wire-protocol notes in
  [docs/engineering/console.md](docs/engineering/console.md).
- **`truestamp beacon` subcommand.** Read-only access to the
  Truestamp beacons JSON:API at `/api/json/beacons/*`: `beacon`
  (default = `latest`), `beacon list`, `beacon get <uuidv7>`,
  `beacon by-hash <64-hex>`. Output is plain Unix-friendly text by
  default with optional `--json` / `--hash-only` / `-s` modes. The
  single-beacon card emits two shareable public-web links
  (`Details → {host}/beacons/<hash>` and
  `Verify → {host}/verify/beacon/<id>`) and renders unconditionally
  — localhost and plain-http hosts too — so links work when
  developing against a local server.
- **`truestamp team` subcommand.** `team list / show / set /
  unset`. `team list` emits a four-column `★/ID/NAME/ROLE` table
  over `GET /api/json/memberships?include=team`; `team set <id>`
  validates by reading `/teams/{id}` first so a typo or revoked
  membership refuses to write; `team set` with no arg opens an
  interactive picker; `team unset` reverts to the server's
  personal-team auto-fallback. Active selection persists in the
  top-level `team` key in `~/.config/truestamp/config.toml`. The
  console Teams pane (key `3`) is wired to the same source of
  truth, and `config show` / `auth status` both surface `Team Name`
  / `Team Role` rows alongside the bare id.
- **Claims-as-source-of-truth submission mode for `truestamp
  create`.** A second submission mode where no external file is
  required — the claims content itself is what gets timestamped,
  gated by a server-side meaningful-content rule (≥ 32-char
  description or non-empty metadata). The `--hash` / `--hash-type`
  pair is now co-required: both supplied (external-hash mode) or
  both omitted (claims-content mode). Mirrored in the console New
  Item pane via a leading `Submission mode` Select; hash + hash_type
  fields auto-hide in claims-content mode and the description field
  enforces the ≥ 32-char rule inline.
- **CLI-wide JSON logger with redaction and panic recovery.**
  `internal/logging` extends a single structured `slog` JSON logger
  across every subcommand. `internal/redact` scrubs api_key
  patterns from log payloads (security-critical — covered by the
  `TestRedactSecrets` fuzz target). A top-level panic recovery
  installs a `slog.Error("panic", ...)` handler that captures
  goroutine stacks before the process exits, so the console's
  Bubble Tea event loop never leaks an uncaught panic to a black
  terminal.
- **Third-party health checks in the Connection pane.** Probes
  Truestamp service (`{base_url}/health`), Truestamp keyring,
  NIST Beacon, Stellar Horizon (mainnet), and Blockstream every
  minute while the pane is open. Results sort problems first
  (failed → degraded → checking → unknown → ok). Manual refresh
  rate-limited to one run per 3 seconds.
- **`/release` Claude Code skill.** Canonical end-to-end release
  flow for this repo, documented at
  [.claude/skills/release/SKILL.md](.claude/skills/release/SKILL.md).
  Walks the full playbook (pre-flight quality gate, GoReleaser
  dry-run, CHANGELOG update, release PR + CI, signed annotated tag,
  release.yml watch, post-release artifact verification) and
  delegates partial-failure recovery to a sibling reference doc.
  `CONTRIBUTING.md` points at the skill as the normal way to ship.

### Changed
- **URL config consolidated into a single `base_url`.** Previous
  configs surfaced multiple endpoint URLs (api_url, keyring_url,
  websocket_url, health_url) that all derived from the same
  Truestamp deployment. The new `base_url` (default
  `https://www.truestamp.com`) is the single dial — `api_url`,
  `keyring_url`, the console WebSocket URL, and the health-check
  URL are all derived from it at config load, with the per-URL
  flags / env vars preserved for explicit override. `config show`
  renders the derived values alongside `base_url` so the source of
  truth is obvious.
- **Console monitor waterfall redesigned.** Table-driven layout
  via `lipgloss/v2/table`, mouse-wheel scrolling, deterministic
  reverse-chronological event ordering, ID column no longer
  truncates the primary id, scroll indicator anchored to the title
  row, footer state-aware to whichever pane is focused. Default
  focus on app launch goes straight to the events waterfall so
  `j/k`/arrows scroll without a leading Tab. Header status trimmed
  — plan + stream count moved to the Connection pane to declutter
  the Monitor.
- **New Item form ergonomics.** Migrated to `huh/v2`, the
  hash-type Select uses the canonical Display names shared with
  the watching screen, validation runs inline at the offending
  field with the three checks (algorithm registered, even-length
  hex, length matches algorithm) mandatory in every code path,
  `?` toggles help by growing the body (no longer resizing the
  screen), watching screen drops the card border + fixes line
  wrap, esc returns cleanly to the form, and quit prompts for
  confirmation so an accidental ctrl-C doesn't lose a submission
  in flight.
- **Go toolchain pinned to 1.26.3.** `.tool-versions` (single
  source of truth for CI via `mise`) and `go.mod` both bumped from
  1.26.2 — see Security below.

### Fixed
- **Health-check latency flake on fast loopback.** The Connection
  pane's `checkHealthTarget` rounded `time.Since(start)` to
  milliseconds at storage time. On a fast macOS runner the
  in-process httptest.Server round-trip completes in under 500µs,
  so the value collapsed to `0s` and broke
  `TestCheckHealthTarget_OK`'s positive-latency assertion. The
  renderer in `connection.go` already rounds at display time, and
  `TestRenderHealthSection_IconsAndDetails` documents the storage
  contract by setting `r.Latency = 42_000_000` (42ms in
  nanoseconds). Storage-time rounding was wrong on both counts:
  it lost sub-ms signal and made "too-fast-to-measure"
  indistinguishable from "never measured". Removed.

### Security
- **Go 1.26.2 → 1.26.3** clears two stdlib advisories surfaced by
  `task vuln-check`: **GO-2026-4971** (panic in `net.Dial` /
  `LookupPort` on Windows NUL bytes; reachable via
  `httpclient.DownloadBytesCtx`) and **GO-2026-4918** (infinite
  loop in `net/http/internal/http2` on a bad
  `SETTINGS_MAX_FRAME_SIZE`). Both fixed in 1.26.3. The
  `.tool-versions` comment block records the bump rationale so
  future Go bumps can compare against a known baseline.

### Developer experience / tests
- **wschannel chaos test suite.** New `internal/wschannel` chaos
  tests under a `chaos` build tag exercise reconnect-with-backoff,
  topic rejoin replay, pending-call drain on disconnect, and
  api_key redaction under failure. `task test-chaos` runs them
  locally; the CI workflow runs them on every PR alongside the
  main test matrix.
- **wschannel codec fuzz tests.** Three new `FuzzXxx` targets
  cover Phoenix V2 array-form `Frame` unmarshal, reply parsing,
  and the api_key redactor. Seeds committed; CI replays them on
  every run plus runs each for 30s of active mutation per push as
  part of the existing fuzz step.
- **Phased TUI refactor (six phases).** Migration to lipgloss/v2 +
  bubbletea/v2 + huh/v2 broken into reviewable slices: chrome
  package + keymap-driven footer (Phase 1), mouse wheel scroll on
  Monitor (Phase 2), canonical event projector + lipgloss/v2/table
  for the waterfall (Phase 3), New Item form migration to huh/v2
  (Phase 4), Connection pane polish via lipgloss/v2/table (Phase
  5), tests for chrome / events projector / keymaps (Phase 6).

### Dependencies
- `golang.org/x/crypto` 0.50.0 → 0.52.0
- `golang.org/x/mod` 0.35.0 → 0.36.0
- `github.com/btcsuite/btcd/chaincfg/chainhash` 1.1.0 → 1.2.0
- `github.com/fxamacker/cbor/v2` 2.9.1 → 2.9.2
- `github.com/knadh/koanf/parsers/toml/v2` 2.2.0 → 2.2.1
- `github/codeql-action` (workflow) 4.35.2 → 4.35.3
- `goreleaser/goreleaser-action` (workflow) patch bump

## [0.7.1] — 2026-04-23

### Changed
- **Release workflow gated on full CI.** `release.yml` now runs the
  `ci.yml` job as a prerequisite (via `workflow_call`) on the tagged
  SHA before GoReleaser starts. If the tag SHA fails lint, tests,
  staticcheck, gosec, or govulncheck, no artifacts publish, no tap
  PR opens, no half-released state to clean up. Adds a two-layer
  quality gate: ruleset-enforced CI on `main` pre-tag, plus a
  re-run on the tag SHA itself at release time.
- **Release pre-flight steps in GoReleaser job.** Before the real
  `goreleaser release`, the workflow now runs `goreleaser check`
  (validates `.goreleaser.yaml` syntax — catches a malformed config
  in under a second) and `goreleaser release --snapshot --clean`
  (full cross-compile dry-run: SBOM generation, archive creation,
  cask template rendering — surfaces platform-specific build
  breakage before anything touches GitHub Release or cosign).
  Tag-push → published release now takes ~7-9 min instead of ~4,
  in exchange for a half-publish-proof pipeline.
- **CI adds `staticcheck` and `gosec` steps.** Previously only run
  locally via `task lint`; now enforced on every PR and every
  `main` push. gosec exclusion list is kept in sync with
  `Taskfile.yml`'s `task lint` (same 11 exclusions, same
  justifications). Adds ~15s to CI runtime.
- **`ci.yml` is a reusable workflow.** Added `workflow_call:` to
  its triggers so `release.yml` can invoke it as a gate job.
  External-facing behavior on `push: main` and `pull_request`
  events is unchanged.

### Fixed
- **Stale flag references in docs.** `README.md:168` used
  `--output silent`; the actual flag is `--silent` / `-s`. Fixed.
  `Taskfile.yml`'s `task run` description hinted `[--verbose]` as
  an example `verify` flag; no such flag exists. Replaced with
  `[--skip-external] [--skip-signatures] [--json]`.
- **`task release-snapshot` fails offline.** The task ran
  `goreleaser release --snapshot --clean` without the `--skip=sign,publish`
  flags that the `.goreleaser.yaml` header comment documents as the
  correct local dry-run invocation. Without the skips, cosign tries
  keyless signing, which requires a GitHub OIDC token only the
  release workflow has — the local task sat waiting for a device-flow
  OAuth code it couldn't receive, then failed after five minutes.
  Fixed: task now matches the documented dry-run.
- **`CONTRIBUTING.md` release procedure updated for the
  `protect-main` ruleset.** The documented `jj git push --bookmark main`
  flow was written before the ruleset existed and is now blocked by
  the required-status-checks rule. Procedure rewritten to route the
  release commit through a PR (`release-vX.Y.Z` branch → rebase-merge)
  so CI runs on the exact SHA before it lands on `main` and then
  gets tagged. Added prose explaining the two-layer
  tests-green-on-this-SHA guarantee (ruleset at merge time +
  `release.yml` CI gate at tag time), a `git tag -v` verification
  step, and the expected shape of `release.yml`'s two-job flow.
- **Bug-report issue template.** `.github/ISSUE_TEMPLATE/bug_report.yml`
  was carrying a stale recommendation to pass `--output debug`
  (doesn't exist anywhere in the CLI). Template overhauled: added
  a subcommand dropdown, an install-method dropdown covering all
  six real install paths, an OS/arch dropdown, a subject-ID text
  input (ULID / UUIDv7) so maintainers can correlate bugs with
  server logs, a steps-to-reproduce field, and guidance on which
  values to redact before submitting. The stale `--output debug`
  hint is replaced with `--json` (which does exist).

### Security
- **Branch-protection ruleset (`protect-main`) active on `main`.**
  Not a code change — this is a repo-settings change — but
  documenting it here because it's load-bearing for the release
  flow: pushes to `main` (direct or PR-merged) now require the
  `Test (ubuntu-latest)` and `Test (macos-latest)` checks to be
  green on the exact SHA. Linear history is enforced; deletion and
  non-fast-forward pushes are blocked; no bypass actors. Pairs
  with the `release.yml` CI gate so a release tag can only point
  at a commit whose CI went green twice (once at merge time, once
  at tag time).

### Developer experience
- **Copyright / SPDX headers on every source file.** Previously
  only `.go` files carried the canonical two-line header; 11
  non-Go source files (GitHub workflow YAMLs, issue templates,
  `dependabot.yml`, `.goreleaser.yaml`, `Taskfile.yml`, the
  install shell scripts, and `defaults/config.toml`) did not.
  Now uniform across the tree — SBOM / license scanners see a
  consistent per-file license declaration. The TOML header
  propagates into user configs generated by `truestamp config init`.
- **Copyright year range rolled to 2019-2026.** `2021-2026` was
  the stale start-of-coverage in the canonical header; 2019 is
  the correct founding year. Applied uniformly across all 151
  header-carrying files.

## [0.7.0] — 2026-04-23

### Added
- **`cosign_path` config setting.** Operators can pin the absolute
  path to the `cosign` binary used by `truestamp upgrade` for
  release-artifact signature verification, defending against
  `$PATH` hijacking in hardened environments. Settable in
  `config.toml` as `cosign_path = "/abs/path"`, via the
  `TRUESTAMP_COSIGN_PATH` env var, or left empty for the default
  `$PATH` lookup. Validated at config load (must be absolute; empty
  is valid); existence and the executable bit are re-checked at
  use time inside `selfupgrade.resolveCosignBinary`. Surfaces in
  `truestamp config show` and is included in the first-generation
  config template written by `truestamp config init`.
- **HTTP `User-Agent` header** on every outbound request, stamped
  centrally from `httpclient.SetUserAgent` as
  `truestamp-cli/<version> (<os>/<arch>)`. Caller-set User-Agent
  headers are preserved (the GitHub client's existing
  `truestamp-cli-upgrade` UA is unchanged).
- **`inputsrc.ErrNoTTY`.** The interactive file picker (`--file`
  with no path) and URL prompt (`--url` with no URL) now fail fast
  with a clear error when stdin is not a terminal, instead of
  crashing inside the huh renderer.
- **Parallel multi-file hashing.** `truestamp hash a b c …` uses a
  bounded `runtime.NumCPU()` worker pool when three or more inputs
  are supplied. Arg order is preserved in the output; stdin
  sources force serial execution (pipes aren't concurrent-safe).
- **`truestamp download --help` exit codes.** The help text now
  explicitly documents exit-0 on success and exit-1 on any error.

### Changed
- **Self-upgrade tar extraction rejects non-regular entries.**
  `internal/selfupgrade.ExtractBinary` now refuses tar entries
  whose `Typeflag` is not `tar.TypeReg` (symlinks, hardlinks,
  devices, etc.). GoReleaser tarballs only ship regular files; the
  explicit rejection is defence in depth if archive provenance
  ever broadens.
- **`hasParentTraversal` simplified.** Removed an unused
  `filepath.SplitList` loop (which splits on PATH separators, not
  path segments, and whose body was empty anyway). The working
  `filepath.Clean` round-trip plus `slices.Contains(segments, "..")`
  check is retained and unchanged in behaviour.
- **`http_timeout` validation.** Zero and negative durations are
  now rejected at `config.Load` with a clear error. `http.Client`
  treats those as "no timeout" which would silently disable the
  guard.
- **`config show` layout.** New "Cosign Path" row under the
  General section, rendering `(auto: $PATH lookup)` when unset
  and the pinned absolute path otherwise.
- **`truestamp hash` worker pool.** The per-input hash path is
  unchanged for a single file or a stdin source; it's the
  multi-file case that now parallelises across cores.
- Release workflow: direct `gh pr merge --merge` on the
  homebrew-tap PR instead of `gh pr merge --auto --merge`. The
  tap's `protect-main` ruleset only blocks branch deletion and
  non-fast-forward pushes — no required checks or reviews — so
  nothing gates mergeability, and `--auto` rejected the
  instantly-clean PR with "Pull request is in clean status". The
  simplification removes a manual merge step that the 0.6.0
  release tripped over.

### Fixed
- **Keyring error chain.** `external.VerifyKeyring` now returns a
  `keyringNetError` wrapper that keeps the user-facing classified
  message as `.Error()` while `.Unwrap()` exposes the underlying
  `net.DNSError`, `net.OpError`, `tls.CertificateVerificationError`,
  `context.DeadlineExceeded`, etc. for `errors.Is`/`errors.As`
  callers. Previously the classification `string` replaced the
  chain entirely.
- **Cosign `$PATH` lookup hardened.** `selfupgrade.VerifyCosign`
  no longer falls back to `$PATH` when a pinned path is
  configured — an operator who set `cosign_path` gets their
  chosen binary or a clear error, never a silent $PATH
  substitution. SHA-256 verification remains mandatory regardless
  of cosign status; cosign stays best-effort unless
  `TRUESTAMP_REQUIRE_COSIGN=1`.

### Security
- **Trust model documented for `external.VerifyKeyring`.** The
  function's doc comment now states explicitly that the keyring's
  authenticity is rooted entirely in TLS chain verification to
  the configured `keyring-url` — there is no in-band signature
  over the keyring payload — and spells out the implications for
  operators choosing a `keyring_url` value.
- **Defensive abs-path check on pinned cosign.** The path stored
  in `cosign_path` / `TRUESTAMP_COSIGN_PATH` must be absolute;
  relative paths are rejected at config load. This blocks a
  subtle CWD-bait class where `cosign_path = "cosign"` would
  otherwise resolve against whatever directory the CLI happened
  to run from. The `gosec` G703 warning on the consequent
  `os.Stat(pinned)` is suppressed with a targeted `#nosec` and
  justification — the input has been reduced to an
  operator-chosen absolute path on a CLI running with the user's
  own privileges, and the stat result only gates whether we
  return that exact path as a command to exec.

### Developer experience / tests
- **`t.Parallel()` added to 143 pure-logic unit tests** across
  `internal/{encoding,hashing,tscrypto,bitcoin,proof,selfupgrade}`.
  The two tests that use `t.Setenv` on shared keys
  (`TestVerifyCosign_missingBinary_*`) stay serial. Full suite is
  race-clean under `go test -race ./...`.
- **New regression tests:**
  `TestResolveCosignBinary_pinnedPath` (valid exec, missing,
  directory, non-executable, relative — five cases),
  `TestExtractBinary_rejectsSymlink`, `TestDefaultPickFile_NoTTY`,
  `TestDefaultPromptURL_NoTTY`, `TestLoad_NonPositiveHTTPTimeout`
  (both 0s and -1s), `TestLoad_CosignPathAbsolute` (four
  sub-cases), `TestLoad_CosignPathFromEnv`,
  `TestSetUserAgent_StampedOnOutboundRequests`.
- **Shared convert-subcommand flag helper.** `cmd/convert.go`
  exposes `addConvertCommonFlags(cmd)` so the five convert
  children (`time`, `id`, `keyid`, `merkle`, `proof`) register
  the shared `--json` and `--silent` flags through one call site,
  keeping defaults and help text in sync. `convert proof` still
  registers `--json` locally because its semantics ("JSON
  envelope", not "raw JSON output") warrant a different help
  string.
- **Documentation hygiene.** `CLAUDE.md`'s fuzz-target count
  updated 45 → 64 with a per-package breakdown (cmd:8,
  tscrypto:11, proof:7, bitcoin:6, selfupgrade:5, encoding:5,
  hashing:4, beacons:4, items:3, config:3, verify:2,
  upgradecheck:2, inputsrc:2, external:2). `Taskfile.yml`'s
  `fuzz-deep` description now spells out that reproducers the Go
  fuzz engine writes under `testdata/fuzz/FuzzXxx/` MUST be
  committed so `task test` replays them as permanent regression
  corpus. `README.md` and `CLAUDE.md` document the new
  `cosign_path` setting in their respective Global Flags tables,
  and the README's Upgrading section calls out the pin option
  alongside `TRUESTAMP_REQUIRE_COSIGN`.

## [0.6.0] — 2026-04-23

### Added
- **Proof format restructure.** The subject-kind discriminator moves
  from the string `s.src` to a top-level integer `t` that identifies
  both the subject and every external-commitment chain. The byte is
  emitted as a 2-byte big-endian u16 right after the version byte in
  the Ed25519 signing pre-image, so flipping `t` on a bundle without
  re-signing breaks signature verification — real cryptographic
  domain separation between subject kinds. Frozen type-code registry
  (single source of truth at `internal/proof/ptype/ptype.go`):
  - `10` block · `20` item · `30` entropy_nist · `31` entropy_stellar
    · `32` entropy_bitcoin · `40` commitment_stellar · `41` commitment_bitcoin.
  The wire field `v` remains `1`; this is a structure-only refresh,
  not a new bundle version. Parser tightened: unknown `t` rejected,
  `cx` required non-empty, block proofs must not carry `s` / `ip`,
  non-block proofs must, every hash field decodes to the exact byte
  count declared by the CDDL schema (32 for hashes, 4 for kid, 32
  for pk, 64 for sig). Opaque failures three steps into the pipeline
  now surface as "pk must be 32 bytes, got 31" at parse time.
- **Beacons are first-class proof subjects (`t = 11`).** Beacon and
  block (`t = 10`) share the same wire shape (no `s`, no `ip`,
  `subject_hash == block_hash`) but are cryptographically distinct
  because `t` is in the signed payload — a block bundle and a beacon
  bundle over the same underlying block have different signatures.
  Verify dispatches both through the same pipeline via the new
  `ptype.IsBlockLikeSubject` / `ProofBundle.IsBlockLike()` predicates;
  the report's Type row reads "Beacon" for `t=11`, "Block" for `t=10`.
- **`truestamp beacon {latest,list,get,by-hash}`** — read-only client
  for the Truestamp Beacons JSON:API (`GET /api/json/beacons/*`).
  Shared `--json` / `--hash-only` / `--silent` flags, client-side
  UUIDv7 and 64-hex validation before the network round-trip, and
  typed sentinel errors (`Unauthorized`, `NotFound`, `BadRequest`,
  `RateLimited`, `Server`) that preserve the server's JSON:API error
  envelope. `truestamp beacon` alone defaults to `latest`.
- **`truestamp download --type`** extended to the full six-value
  enum matching the server's strict `/proof/generate` contract:
  `item | entropy_nist | entropy_stellar | entropy_bitcoin | block | beacon`.
  No `auto`, no bare `entropy`. Smart default from id shape: a ULID
  id defaults to `--type item` (the only unambiguous case); a UUIDv7
  id without `--type` fails fast listing the five valid choices,
  since the server cannot infer the subtype from id alone. Downloaded
  filenames use the `truestamp-<stem>-<id>.<ext>` convention with
  hyphenated stems (`truestamp-entropy-nist-<id>.json`) while the
  wire value stays underscored to match the server enum.
- **`truestamp verify --type`** — assert the expected subject type
  locally and remotely, defending against swapped-file confused-deputy
  scenarios (block and beacon are wire-identical apart from the `t`
  byte, so both verify cleanly on their own). Local mode surfaces a
  mismatch as a "Subject Type" failure step in the report. Remote
  mode forwards the value to the server's `/proof/verify type` arg;
  the server rejects with HTTP 4xx + `meta.code=subject_type_mismatch`
  if the posted bundle's `t` disagrees. When `--type` is omitted the
  CLI infers it from the input filename / URL basename using the
  download convention (`truestamp-beacon-019d…json` → beacon), prints
  a faint stderr hint so the inference is traceable in transcripts,
  and lets an explicit `--type` override. Inference goes through a
  dedicated fuzz target (`FuzzInferTypeFromFilename`) with a 20-seed
  corpus.
- **Live entropy-source consistency checks.** The Entropy Source
  verification step now contacts the canonical upstream source for
  each subject type and byte-compares the bundle's stored value:
  - NIST Beacon (`t=30`) — `beacon.nist.gov/beacon/2.0/chain/{c}/pulse/{p}`,
    compare `outputValue` and `timeStamp`.
  - Stellar ledger (`t=31`) — Horizon `/ledgers/{seq}` on the network
    derived from the bundle's Stellar commitment; compare `hash` and
    `closed_at`.
  - Bitcoin block (`t=32`) — `blockstream.info/api/block/{hash}` pinned
    to mainnet (the authoritative public-randomness source, even in
    dev deployments that commit to regtest/testnet); compare `height`
    and `time`.
  Skipped under `--skip-external`.
- **Dual Details + Verify URLs on every post-action card** (beacon,
  download, create, and the verify report's Proof section). URL shape
  comes from the shared `ui.SubjectDetailURL` / `ui.SubjectVerifyURL`
  / `ui.BeaconDetailURL` / `ui.BeaconVerifyURL` helpers, routed
  through one `publicWebBase` that strips a trailing `/api/json`.
  URLs render unconditionally — localhost, 127.0.0.1, and plain-http
  hosts too — so the links are visible when developing against a
  local server.
- `internal/beacons` — HTTP client for the Beacons JSON:API (Latest /
  List / Get / ByHash) with typed errors, 429 `Retry-After` surfacing,
  and a fuzz suite over the parser + UUIDv7 / 64-hex validators.

### Changed
- **Homegrown semver parser replaced with `golang.org/x/mod/semver`**
  (the same package `go mod` uses). ~100 lines of parser + comparator
  deleted; pre-release identifier comparison (numeric vs alphanumeric,
  shortest-prefix rules from SemVer §11) now comes from battle-tested
  upstream code. Public API (`Semver`, `ParseSemver`, `Compare`,
  `Display`, `IsPreRelease`) is preserved. `go.mod` gains a direct
  `golang.org/x/mod` dependency — already transitive, so effectively
  zero footprint growth in real-world installs.
- **Upgrade-check no longer mis-ranks git-describe dev builds.** A
  locally-built `0.5.0-4-g356ee75-dirty` binary is conceptually
  4 commits AHEAD of v0.5.0, but SemVer §11 ranks it as a pre-release
  BELOW the tag. Two concrete fixes: new
  `selfupgrade.IsGitDescribeDev` predicate regex-detects the
  `<N>-g<SHA>[-dirty]` shape (accepts bare post-tag and post-release
  suffixes, rejects plain pre-releases like `rc.1`); new
  `selfupgrade.UpgradeAvailable` delegates to the normal comparator
  but compares MAJOR.MINOR.PATCH cores only for dev builds, so
  `0.5.0-4-g...-dirty → v0.5.0` reports no upgrade (would downgrade)
  while → v0.5.1 / v0.6.0 / v1.0.0 still does. Both `truestamp
  upgrade --check` and the passive upgrade-check nag now use this
  predicate.
- **Narrow-TTY layout fix.** `lipgloss.JoinVertical(Left, …)` pads
  every line to the widest line across all inputs; one over-wide
  section pushed every other row past the terminal width, which the
  terminal then hard-wrapped, producing phantom blank lines after
  every table row. Switched to `strings.Join(…, "\n")` at six call
  sites (`internal/verify/presenter.go`, `cmd/config.go`,
  `cmd/beacon.go`, `cmd/beacon_list.go`, `cmd/create.go`,
  `cmd/download.go`). Each site carries a comment pointing back to
  the root-cause note.
- **Tight vertical spacing across every card.** `lipgloss.HiddenBorder()`
  emits invisible top/bottom border rows that stack with explicit
  `""` separators in `strings.Join`, doubling the apparent gap
  between a section header and its first row. New
  `ui.CompactTable()` helper returns a table with `HiddenBorder`
  plus `BorderTop/Bottom/Left/Right(false)` — content is flush to
  whatever comes before and after, and callers control inter-section
  spacing explicitly. Applied at 16 call sites across `cmd/` and
  `internal/verify/presenter.go`.
- **Full-hash display everywhere.** Removed `truncateHash()` and its
  seven call sites in `internal/verify/presenter.go`; diagnostic
  contexts (hash-mismatch diffs, entropy-source mismatch, Bitcoin
  fetch/height errors) now emit the full 64-char hex. Two different
  hashes sharing prefix + suffix used to read as visually identical
  — no longer. Commitments section already did, so this is the
  convergence pass. `cmd/beacon_list.go` also drops its local
  `truncateHashShort` in favour of TTY-aware width logic.
- **Centralized timestamp truncation.** Promoted the package-local
  `verify.truncateToSecond()` to exported `ui.TruncateToSecond()`,
  applied at every human-display site (beacon list TIMESTAMP column,
  beacon card, verify report Timeline section). `--json` output and
  timestamp-extraction sites (`convert time`, `convert id`) keep full
  microsecond precision; `verify` Stellar `closed_at` / Bitcoin `time`
  mismatch errors keep full precision so a sub-second diff isn't
  masked.
- **Stellar `net` strict.** Only `"testnet"` and `"public"` accepted
  — the previous tolerance of other values is gone.
- **Legacy `s.kid == b.kid` equality check removed.** Legitimate key
  rotation can produce divergent kids; subject-kid tampering is
  still detected because `kid` is an input to the 0x13 / 0x23
  composite hash.
- Verify report groups renamed to match the authoritative spec:
  Subject Data, Block Hash, Epoch Proof, Temporal Window,
  Entropy Source.
- Subject-card URL rows now render as rows of the SAME table as
  labels (Details / Verify inherit the right-aligned-label column),
  and the `→` arrow between label and URL was dropped as redundant
  visual noise.
- CBOR parser now accepts `t ∈ {10, 11}` for no-`s` / no-`ip` proofs
  matching the JSON parser. Marshaller emits no `s` / `ip` for both.

### Removed
- `--type auto` and bare `--type entropy` on `download` — rejected
  client-side before any I/O. The server's strict six-value enum is
  the contract.
- Old `s.src` string discriminator. All callers branch on the `t`
  integer exclusively.

### Security
- Build pulls `golang.org/x/mod` direct; `task vuln-check` clean.
- Dependabot: `github-actions` group updates ([#4](https://github.com/truestamp/truestamp-cli/pull/4)).

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
- SPDX-style `Copyright (c) 2019-2026 Truestamp, Inc.` +
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

[Unreleased]: https://github.com/truestamp/truestamp-cli/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.6.0
[0.5.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.5.0
[0.4.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.4.0
[0.3.3]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.3
[0.3.2]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.2
[0.3.1]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.1
[0.3.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.3.0
[0.2.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.2.0
[0.1.0]: https://github.com/truestamp/truestamp-cli/releases/tag/v0.1.0
