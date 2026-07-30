# Authentication

Authentication is **OAuth-first, API-key-second**, unified behind a single `auth.Authorizer` ([`internal/auth/auth.go`](../internal/auth/auth.go)). One `Authorizer` is resolved once in `cmd/root.go`'s `PersistentPreRunE` and installed process-wide via `auth.SetDefault`; every authenticated call site draws from `auth.Default()`.

**Precedence** ([`internal/auth.Resolve`](../internal/auth/resolve.go)):

1. **Explicit API key**: `--api-key` flag or `TRUESTAMP_API_KEY` env (`Credentials.APIKeyExplicit`). Wins outright so CI/headless is deterministic.
2. **Stored OAuth session**: access token, auto-refreshed via the rotating refresh token.
3. **Config-file `api_key`**: a key sitting in config.toml (not "explicit").
4. **None**: unauthenticated; public requests still go out, the server 401s on protected ones.

Both an OAuth access token and an API key are presented to the JSON API as `Authorization: Bearer <value>`; the server accepts either.

**OAuth flow**: browser-based loopback Authorization Code + PKCE (S256 only), `golang.org/x/oauth2`. Public client: the `client_id` `019ef661-6737-71ec-abd0-ac8f4684ce45` (a UUIDv7: the server's `oauth_clients` PK type rejects non-v7 ids) is baked in ([`auth.ClientID`](../internal/auth/auth.go), `token_endpoint_auth_method=none`: PKCE is the per-flow secret); every endpoint (issuer / authorize / token / revocation) comes from RFC 8414 discovery ([`discovery.go`](../internal/auth/discovery.go)), validated with issuer-match + same-origin endpoint pinning. The loopback redirect binds **fixed ports** `127.0.0.1:8976` (primary) → `:8765` (fallback), exact-matched server-side ([`login.go`](../internal/auth/login.go) `loopbackPorts`). Requested `Scopes` = `api:read api:write console:read console:write` (no `mcp:*`).

**Commands** ([`cmd/auth.go`](../cmd/auth.go)):

- `auth login`: browser OAuth by default; `--api-key` switches to an interactive paste-a-key path that writes the key to config.toml (0600).
- `auth logout`: best-effort RFC 7009 `/oauth/revoke` of the refresh token + clear the local session; `--api-key` additionally removes the stored config-file key. Idempotent.
- `auth status`: mode-aware: renders **Auth Mode**, plus (in OAuth mode) scopes and token expiry, then validates the credential against `/users` and resolves the team. `config show` also gained an **Auth Mode** row (`cmd/config.go` `authModeDisplay`).

**Storage** ([`store.go`](../internal/auth/store.go)): `Session` (access + rotating refresh token + cached endpoint URLs + scope/expiry) persisted per server origin in the OS keychain (`zalando/go-keyring`), transparently falling back to a 0600 JSON file under the user cache dir (`truestamp/oauth.json`) on a keychain-less host. The refresh token rotates on every refresh; the rotated value is persisted best-effort after each grant ([`resolve.go`](../internal/auth/resolve.go) `oauthAuthorizer.token`).

**Reactive 401 retry**: `auth.NewRetryTransport(nil)` is layered onto the shared HTTP client (`httpclient.SetTransport`, wired in `cmd/root.go`). In OAuth mode a `401` on a request *we* authenticated triggers one `ForceRefresh` + retry; the server's OAuth→API-key fallback means an expired token can surface as a bare 401 with no `WWW-Authenticate`, so the contract is "any 401 in OAuth mode ⇒ refresh-and-retry-once", centralized here for every call site. A dead session (`invalid_grant` ⇒ `ErrSessionExpired`) is not retried.

**Call-site routing**: the six former Bearer-header sites (`cmd/auth.go` `checkAuth`/`fetchTeam`, `internal/teams`, `internal/beacons`, `internal/items`, `internal/proof/download.go`, `internal/verify/remote.go`) now authorize through `auth.AuthorizeRequest` / `auth.Default()`; their `Config`/params no longer carry an api key.

**WebSocket**: see [console/websocket.md](console/websocket.md): OAuth sends the access token as an `?access_token=` query param on the upgrade (a Phoenix upgrade can't read the `Authorization` header; api key stays `?api_key=`), with a `token_expired` → force-refresh → re-dial → re-join recovery and a dead-session stop.

**Secret redaction** ([`internal/redact`](../internal/redact/redact.go)): the single redaction source (used by `internal/logging` and `internal/wschannel`) was extended to scrub `access_token` / `refresh_token` / `code` / `code_verifier` (query-string and JSON forms) alongside the existing `api_key=…` / `Bearer …` patterns, plus `redact.WrapError` (redacts `Error()` while preserving `errors.Is`/`errors.As` through the chain; used so a token error stays `errors.Is(ErrSessionExpired)` for the WS fatal-session check without leaking bytes).

**In-band keep-alive**: `internal/wschannel`'s `keepAliveLoop` polls and, ~60s before the access token expires, force-refreshes it and pushes `token.refresh {"access_token": <jwt>}` on `console:lobby` over the live socket. The server re-validates and reschedules its disconnect timer (`{:ok, %{exp}}`), so a long console session re-authenticates **without** reconnecting and `token_expired` never fires. Any failure (dead session, rejected token, delivery hiccup) falls back to the reactive `token_expired` → re-dial path. Wired via `wschannel.Options.AccessTokenExpiry` ← `auth.Authorizer.AccessTokenExpiry`.

