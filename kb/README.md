# Knowledge base

Reference documentation for `truestamp-cli`, split out of `CLAUDE.md` so it can be read on demand instead of loaded in full for every task.

`CLAUDE.md` at the repo root stays deliberately small: orientation, the hard rules, and this index. Everything below is detail you read when you're about to touch the corresponding code.

These files are **not** published to <https://get.truestamp.com/>, that site is served from `docs/`. This directory is repo-internal, matching the `kb/` convention used in [`truestamp/truestamp-v2`](https://github.com/truestamp/truestamp-v2).

## Index

What each document covers, and when to read it, is the table in [`CLAUDE.md`](../CLAUDE.md) §"Knowledge base": that copy is loaded into every session, so it is the one kept current, and a second table here had already drifted from it. The files:

- [command-tree.md](command-tree.md), the noun-first tree and the verb vocabulary
- [proof-bundle-format.md](proof-bundle-format.md), the published wire format
- [verification-steps.md](verification-steps.md), the ordered Appendix E steps
- [architecture.md](architecture.md), code organization and CLI behavior
- [upgrade-and-install.md](upgrade-and-install.md), `truestamp upgrade`
- [configuration.md](configuration.md), flags, env vars, config.toml
- [authentication.md](authentication.md), OAuth, token storage, credential precedence
- [jcs-canonicalization.md](jcs-canonicalization.md), RFC 8785 and the safe-integer thresholds
- [external-apis.md](external-apis.md), every third-party service this binary talks to
- [team-management.md](team-management.md), `truestamp teams` and the console Teams pane
- [console/](console/README.md), `truestamp console`, split by package

## Normative sources

This CLI is written against Appendix E of the Truestamp whitepaper in [`truestamp/truestamp-v2`](https://github.com/truestamp/truestamp-v2). When these documents and the whitepaper disagree, the whitepaper wins. See `CLAUDE.md` §"Relationship to the Truestamp service" for the full pointer list.
