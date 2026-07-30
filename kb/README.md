# Knowledge base

Reference documentation for `truestamp-cli`, split out of `CLAUDE.md` so it can be read on demand instead of loaded in full for every task.

`CLAUDE.md` at the repo root stays deliberately small: orientation, the hard rules, and this index. Everything below is detail you read when you're about to touch the corresponding code.

These files are **not** published to <https://get.truestamp.com/> — that site is served from `docs/`. This directory is repo-internal, matching the `kb/` convention used in [`truestamp/truestamp-v2`](https://github.com/truestamp/truestamp-v2).

## Index

| Document | Covers | Read it when |
| -------- | ------ | ------------ |
| [proof-bundle-format.md](proof-bundle-format.md) | Bundle wire shapes for every subject type, the frozen type-code registry, E.6 hard rejections, the E.4 hex-encoding rule and its normative exclusions | Touching `internal/proof`, parsing, or CBOR |
| [verification-steps.md](verification-steps.md) | The ordered E.8–E.21 step definitions, E.22 group names, E.25 containment, tri-state external confirmation | Touching `internal/verify` or `cmd/verify.go` |
| [architecture.md](architecture.md) | Appendix E conformance machinery, code organization, CLI behavior, data handling, testing — grouped under those five headings. Terminal output, post-action cards, web-URL construction and styling (`internal/ui`) are under "CLI behavior" | Any structural change, or touching `internal/ui` |
| [upgrade-and-install.md](upgrade-and-install.md) | `truestamp upgrade`: install-method detection, the two-layer pre-release defense, passive notices, Windows print-only, exit-code contract | Touching `internal/selfupgrade`, `internal/upgradecheck`, `internal/install` |
| [configuration.md](configuration.md) | Resolution order, global and per-subcommand flags, env vars, config.toml, and the semantics `--help` cannot convey | Touching `internal/config` or adding a flag |
| [authentication.md](authentication.md) | OAuth 2.1 loopback + PKCE, token storage and rotation, credential precedence, the reactive 401-retry transport | Touching `internal/auth` or any authenticated call site |
| [jcs-canonicalization.md](jcs-canonicalization.md) | RFC 8785 canonicalization, the deliberate oversized-integer deviation, the two safe-integer thresholds and why they differ by one | Touching `internal/jcs` or anything that hashes claims |
| [external-apis.md](external-apis.md) | Every third-party service this binary talks to, and when | Adding or changing egress |
| [team-management.md](team-management.md) | `truestamp team`, the console Teams pane, team creation, ownership models | Touching `internal/teams` or team surfaces |
| [console/](console/) | `truestamp console`, split by package: [panes](console/panes.md) (`internal/console`), [websocket](console/websocket.md) (`internal/wschannel`), [logging](console/logging.md) (`internal/logging`), [limits](console/limits.md), [testing](console/testing.md) | Touching any console subsystem — start at [console/README.md](console/README.md) |

## Normative sources

This CLI is written against Appendix E of the Truestamp whitepaper in [`truestamp/truestamp-v2`](https://github.com/truestamp/truestamp-v2). When these documents and the whitepaper disagree, the whitepaper wins. See `CLAUDE.md` §"Relationship to the Truestamp service" for the full pointer list.
