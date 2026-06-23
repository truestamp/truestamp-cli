# `truestamp console` — interactive TUI architecture

The `truestamp console` subcommand opens an interactive Bubble Tea
TUI that holds a long-lived authenticated WebSocket to the Truestamp
backend. It's the live counterpart to the JSON:API: instead of
polling, you subscribe to streams and watch events flow in real time.

This document covers the **client** side — how the TUI is structured,
what the WebSocket client guarantees, where logs go, and what the
limits are. The wire protocol and server-side authoritative reference
live in [truestamp-v2/docs/console_channel.md](https://github.com/truestamp/truestamp-v2/blob/main/docs/console_channel.md).

## Anatomy

```
cmd/console.go                       Cobra registration + flag plumbing; passes auth.Default() as Options.Authorizer
internal/console/                    The TUI
  app.go                             Bubble Tea root model, header/footer, pane switching; maps Options.Authorizer → wschannel OAuth/API-key options
  monitor.go                         Monitor pane: stream toggles + scrollable waterfall
  newitem.go                         New Item pane: form + lifecycle card
  connection.go                      Connection pane: scope, push counts, reconnect summary, log path
  connerror.go                       Dial-error classifier; a dead/absent OAuth session routes to the "re-authenticate" hint
  messages.go                        tea.Msg types + waitForPush bridge; routes tokenRefreshingMsg / authFailedMsg
internal/auth/                       OAuth 2.1 client + Authorizer abstraction (see CLAUDE.md "Authentication")
internal/wschannel/                  Phoenix Channel V2 client (homegrown)
  client.go                          Connection lifecycle, multi-topic, reconnect, OAuth Bearer auth + token_expired recovery, redaction
  codec.go                           Frame encoder/decoder
  redact_test.go                     api_key / Bearer token never leaks
  smoke_test.go                      Opt-in live-server tests (build tag: smoke)
internal/redact/redact.go            Single redaction source (api_key, Bearer, OAuth tokens/code/verifier)
internal/logging/logging.go          slog + lumberjack file logger (redacts via internal/redact)
```

The **server** documentation in `truestamp-v2/docs/console_channel.md`
is authoritative for everything that flows over the wire (catalog,
commands, events, burst shape). Whenever the wire shape changes, that
file changes first; this document trails.

## Three panes

The TUI is a single Bubble Tea program with three switchable panes
sharing one WebSocket:

1. **Monitor** — left column lists the stream catalog with
   checkboxes; right column shows a scrollable waterfall of
   server-pushed events. All catalog streams are toggled on at
   startup. Focus moves between left and right with `←/→`; arrow keys
   navigate the focused side; `space` toggles the cursor stream;
   `r` flips chronological order.

2. **New Item** — a form (huh-style text inputs) for creating a
   timestamped item. Submit fires `items.create` over the WS; on
   success the channel auto-watches the item and the pane renders a
   card with live state transitions (`created → processing →
   committed`) flowing in below.

3. **Connection** — diagnostics: scope summary (user, team, plan),
   uptime, push counts by event, reconnect history, and the log file
   path. No per-event noise; "what's actually wrong" is the sort of
   thing a user can act on, and that goes inline next to the action,
   not here.

Header shows the active pane tabs on the left, status (`connected • <plan>
• <N streams>`) plus the server-time clock on the right. Footer shows
context-sensitive key hints.

## WebSocket client (`internal/wschannel`)

Homegrown Phoenix Channels V2 client (`client.go` ~1k LOC). The wire
format is so small a third-party client (e.g. `nshafer/phx`) is more code
than the problem; controlling reconnect/heartbeat behavior — and the
OAuth `?access_token=` upgrade + `token_expired` recovery below — precisely matters
for a long-running TUI.

### Authentication (OAuth Bearer + `token_expired` recovery)

The console draws its credential from the process-wide `auth.Authorizer`
(`cmd/console.go` passes `auth.Default()` as `Options.Authorizer`;
`internal/console/app.go` maps it onto the wschannel options). Two modes:

- **OAuth** (`Authorizer.Mode() == ModeOAuth`): the access token is sent
  as the `?access_token=<jwt>` query param on the WebSocket **upgrade**
  (a Phoenix upgrade can't expose the `Authorization` header to the
  socket's `connect/3` — `:x_headers` only captures `x-*` headers — so
  the token rides the query like `?api_key=` does). The client wires
  `Options.BearerToken = Authorizer.BearerToken` and pulls a *fresh*
  token on every (re)dial, so a reconnect after a token refresh
  automatically carries the new credential.
- **API key** (`ModeAPIKey`): unchanged — the key is resolved once and
  sent as the `?api_key=` query param to the server's `connect/3`
  callback.

OAuth access tokens are short-lived; the token is validated at *connect*
(not at channel join), so re-authenticating means re-dialling the whole
socket. The recovery flow on a server `token_expired` push:

1. Emit a synthetic `token_refreshing` push (the TUI shows a transient
   "refreshing session" hint via `tokenRefreshingMsg`).
2. Call `Options.ForceRefresh` (the authorizer's `ForceRefresh`) **now**,
   so the upcoming reconnect dials with a genuinely new access token
   rather than re-presenting the just-rejected one (which would loop
   under client/server clock skew).
3. `dropConn` → the normal reconnect-with-backoff path re-dials with the
   refreshed token and re-joins every topic.

**Fatal dead-session stop**: if the forced refresh fails because the
refresh token is expired/revoked/reused (`invalid_grant` →
`auth.ErrSessionExpired`), `Options.FatalDialErr` classifies it as
permanently fatal. The client sets `authDead`, stops retrying, and emits
`auth_failed` (`authFailedMsg`) instead of looping forever re-presenting a
locally-"valid" but server-rejected token. The TUI surfaces a re-login
prompt; `connerror.go` also maps `ErrSessionExpired` / `ErrNoCredentials`
on the dial to the "re-authenticate" hint rather than a network error.

**In-band keep-alive** (`keepAliveLoop` + `inBandRefresh`). When the caller
wires `wschannel.Options.AccessTokenExpiry` (the console does, from
`auth.Authorizer.AccessTokenExpiry`), a background loop polls and, ~60s
before the access token expires, force-refreshes it and pushes
`token.refresh {"access_token": <jwt>}` on `console:lobby` over the *live*
socket. The server re-validates the new token and reschedules its
disconnect timer (`{:ok, %{exp}}`), so a long session re-authenticates
**without** dropping/reconnecting and `token_expired` never fires. On any
failure — dead session (`{:error, invalid_token}` → stop + re-login),
rejected token, or a delivery hiccup — it falls back to the reactive
`token_expired` → re-dial path above. The `token_expired` path remains the
safety net for the asleep-past-expiry case.

### What it guarantees

- **Multi-topic on one socket**. `Connect` joins the primary topic
  (default `console:lobby`); `JoinTopic(ctx, topic)` joins any
  additional topic on the same WS. The TUI uses this to attach
  `console:clock` after the lobby.
- **Heartbeats**. 30-second heartbeat loop on the reserved `phoenix`
  topic. Configurable via `Options.HeartbeatInterval` (lowering is
  primarily a test affordance).
- **Ref correlation**. Every `Push(ctx, topic, event, payload)`
  allocates a unique ref and blocks for the matching `phx_reply`.
  Safe to call concurrently from multiple goroutines. Returns the
  `PhxReply{Status, Response}` directly; status `error` does NOT
  return a Go error — the caller decides whether the domain error is
  fatal.
- **Reconnect**. Exponential backoff (`1s → 2s → 5s → 10s → 30s`,
  capped). On each successful redial, every previously-joined topic
  is rejoined automatically and a synthetic `rejoined` push is
  emitted per topic over `Pushes()` so the application can replay any
  in-channel state the server doesn't remember (subscriptions, item
  watches).
- **Reconnect status events**. Before each dial attempt the client
  emits a synthetic `reconnecting` push with `{attempt, next_attempt_at}`
  the TUI uses for its countdown header.
- **Drained pending on disconnect**. In-flight `Push` calls receive
  a synthetic `phx_reply` with `{status: "error", reason: "connection
  lost during reconnect"}` so they unblock immediately rather than
  hanging on the original ref.
- **Two-stage readiness gates**. Internally the client distinguishes
  "socket alive" (rejoin path can send) from "session ready"
  (application calls can send). Application `Push`/`JoinTopic` waits
  on the session gate so callers never race the rejoin replay.
- **Drop-on-full inbound**. The pushes channel is buffered at 256;
  if the consumer falls behind, frames are dropped (logged at
  `warn`) rather than blocking the reader (which would also block
  the heartbeat).
- **Secret redaction**. The websocket library's dial errors echo the
  upgrade URL (and, in OAuth mode, the `Authorization` header) verbatim,
  including `api_key=…` and `Bearer <jwt>`. `wschannel` flows every
  error through `internal/redact` before returning to the caller AND
  before logging via the slog logger. Neither the `truestamp_…` key nor
  an OAuth access/refresh token can reach the UI, the log file, or any
  stderr the host process owns. `redact` is the single source of truth
  (shared with `internal/logging`); see CLAUDE.md "Authentication" for
  the full pattern set.

### What it does NOT do

- **OAuth**: token *expiry* is recovered automatically (see
  Authentication above — `token_expired` → force-refresh → re-dial →
  re-join). A genuinely dead session (refresh token revoked/expired)
  stops reconnect and prompts re-login rather than looping.
- **API key**: no automatic reauthentication. If the server rejects the
  key, reconnect attempts keep failing — fix the key, restart the CLI.
- No exponential backoff jitter. Fine at single-user scale; would
  matter at thousand-client thundering-herd scale.
- No subscription persistence across CLI restarts. Each launch
  starts fresh and auto-subscribes to the full catalog.

### Sentinel push events

`internal/wschannel` exports synthetic event-name constants used by
application code to special-case reconnect and OAuth lifecycle:

| Constant                      | Wire value         | When                                                                |
| ----------------------------- | ------------------ | ------------------------------------------------------------------- |
| `wschannel.ReconnectingEvent` | `"reconnecting"`   | Emitted before each dial attempt during a reconnect cycle.          |
| `wschannel.ReconnectedEvent`  | `"rejoined"`       | Emitted per topic after a successful redial+rejoin.                 |
| `wschannel.TokenRefreshEvent` | `"token_refreshing"`| Emitted on a server `token_expired` push while the client force-refreshes the OAuth token and re-dials. |
| `wschannel.AuthFailedEvent`   | `"auth_failed"`    | Emitted when the OAuth session is permanently dead (`invalid_grant`); reconnect stops and the user must re-authenticate. |

None are sent by the server; all are synthetic, injected by the client
into `Pushes()` for the application to observe. (The server's own
`token_expired` push is the *input* that triggers `token_refreshing` /
`auth_failed`.)

## Logging (`internal/logging`)

Stdlib `log/slog` JSON handler over
`gopkg.in/natefinch/lumberjack.v2` for size-rotated, age-retained,
gzip-compressed log files. The TUI never writes to stdout/stderr —
those belong to Bubble Tea's renderer.

### Default destination (per platform)

| Platform | Path                                                |
| -------- | --------------------------------------------------- |
| macOS    | `~/Library/Caches/truestamp/console.log`            |
| Linux    | `~/.cache/truestamp/console.log`                    |
| Windows  | `%LOCALAPPDATA%\truestamp\Cache\console.log`        |

Override with `--log-file <path>`.

### Defaults

| Knob          | Default       | Override                                |
| ------------- | ------------- | --------------------------------------- |
| Log level     | `info`        | `--log-level debug|info|warn|error`     |
| Rotation size | 10 MB         | (constant, not currently flag-exposed)  |
| Retention     | 14 days       | (constant)                              |
| Backups       | 5 files       | (constant)                              |
| Compression   | gzip          | (constant)                              |

Worst-case disk: ~`MaxSize × (MaxBackups + 1)` after compression,
roughly 10–15 MB.

### What gets logged

Routed through `logging` from `wschannel.Client` and the panes:

- `info` — session start (with `ws_url`, `log_path`, `version`),
  successful reconnects (with `downtime`, `attempts`).
- `info` — `reconnect dial failed` for each individual backoff
  attempt during an outage. Routine.
- `warn` — `ws read failed`, `ws write failed`, `frame decode failed`,
  `dropped push (consumer slow)`, `subscribe replay after reconnect
  failed`, `auto-subscribe-all failed`, `clock topic join failed`.
- `debug` — outbound rate-limit hits.

Errors that the user can act on (form validation, auth failures,
server-rejected commands) do **not** flow through this logger —
they're surfaced inline next to the action that triggered them. The
logger is a transport diagnostic, not a global error log.

### Inspecting at runtime

```bash
tail -f ~/Library/Caches/truestamp/console.log | jq .
```

The Connection pane shows the live log file path with a hint pointing
at this command.

### Adding more sinks later

Slog's handler model is composable; a future debug overlay (toggleable
key in the TUI) could wrap the file handler with an in-memory ring
buffer and surface the last N entries on demand without changing any
emit-site call.

## Limits and bounds

Defense-in-depth values, none of which should fire under normal
interactive use.

### Server-side (defined in `lib/truestamp_web/channels/console_channel.ex`)

| Limit                                  | Default              |
| -------------------------------------- | -------------------- |
| Max active streams per connection      | 32                   |
| Inbound command rate limit             | 60 / 10s             |
| Outbound stream-push rate limit        | 200 / s              |
| Per-payload encoded size cap           | 16 384 bytes         |
| Coalesce window                        | 500 ms               |
| Phoenix `max_frame_size` (inbound)     | 65 536 bytes         |
| `hibernate_after`                      | 15 000 ms            |

### Client-side (defined in `internal/wschannel/client.go` and `internal/console/`)

| Limit                                  | Default              | Where                                           |
| -------------------------------------- | -------------------- | ----------------------------------------------- |
| Per-frame inbound read limit           | 1 MB                 | `wschannel.Client.Connect`                      |
| `Pushes()` buffer (drop-on-full)       | 256 frames           | `wschannel.Options.PushBufferSize`              |
| Outbound queue                         | 64 frames            | `wschannel.Client.out`                          |
| Reconnect backoff                      | 1, 2, 5, 10, 30s     | `wschannel.reconnectBackoff`                    |
| Heartbeat interval                     | 30 s                 | `wschannel.Options.HeartbeatInterval`           |
| Event waterfall retention              | 24 hours             | `internal/console/monitor.go::eventRetention`   |
| Event waterfall hard cap               | 100 000 events       | `internal/console/monitor.go::eventHardCap`     |
| New Item lifecycle log cap             | 100 transitions      | `internal/console/newitem.go::maxTransitions`   |
| Connection-pane error log cap          | 50 (oldest evicted)  | `internal/console/connection.go`                |

### Memory profile (steady state)

- WebSocket connection: ~30 KB after `hibernate_after` kicks in.
- Event waterfall at 10 events/min × 24 h: ~14 400 events × ~200 B =
  **~3 MB resident**.
- Worst case at sustained burst rates (very rare): up to the 100 000
  hard cap × ~250 B ≈ **~25 MB resident**.
- Log file: ~10 MB before rotation, capped at ~10 MB current + ~2 MB
  per gzipped backup × 5 ≈ **~20 MB on disk**.

A days-long session in normal operation uses well under 50 MB total.

## Coalescing, on the client side

The server may emit `<resource>.burst` summary pushes when many
events of the same stream arrive within 500 ms. The client renders
these as a single waterfall row using the same color scheme as the
underlying resource:

```
14:42:11.500  item.burst              [items.team]   437 events in 500ms  created=250 deleted=37 updated=150
```

`summarizeBurst` decodes the `data.count`, `data.window_ms`, and
`data.by_kind` fields and renders a verb-only breakdown (the resource
is already obvious from the kind column). `data.by_state` is parsed
but not currently rendered inline; it's available on the wire for
future "live counters" features.

The client never decides whether to coalesce — that's a server-side
decision. It just renders any `*.burst` kind that arrives, with the
same row template as everything else. See
`truestamp-v2/docs/console_channel.md` § Coalescing for the
authoritative behavior.

## Reconnection UX

When the wschannel session loop detects a disconnect:

1. The header status flips to `reconnecting in Ns (attempt N)` with a
   live 1-second countdown driven by `reconnectTickMsg`.
2. An `⚠ server.down` outage marker is dropped into the Monitor
   waterfall immediately, repeated every 10 seconds while the outage
   continues.
3. On successful reconnect:
   - A closing `✓ server.up` marker shows the total downtime.
   - All previously active streams are re-subscribed (via
     `monitor.replayAfterReconnect`).
   - The Connection pane's "Reconnects" counter increments, total
     downtime accumulates.
   - The header reverts to the `connected` state.

Outage markers (`⚠`) use a sentinel internal stream id `_outage` so
they're visually distinct and never collide with subscriptions. They
participate in normal scrollback so the user can scroll back through
days of history and see exactly when data went missing.

## Extension points

The architecture has deliberate hooks for future growth without
touching the wire shape:

- **New stream id**. Add to `ConsoleStreams.@global_streams` (server)
  + `pubsub_topics/2` mapping. Client picks it up automatically from
  the welcome envelope and renders any matching `<resource>.*` events
  using the existing color scheme.
- **New burst-eligible stream**. Already universal — every stream
  flows through the coalescer. The first-event-immediate rule means
  bursts only emerge when input rate genuinely warrants them. No
  config change needed.
- **New pane**. Add a `*.go` file in `internal/console/`, register it
  in `app.go`'s pane enum and the `Update`/`View` switches, and route
  any pushes it cares about via its `handlePush` method. The wschannel
  client doesn't need to know about new panes.
- **New command**. Add a `dispatch/3` clause server-side; call
  `client.Push(ctx, lobbyTopic, "<event>", payload)` client-side.

## Testing

### Unit + offline tests

```bash
task test          # everything
task precommit     # full gate (gofmt + vet + staticcheck + gosec + tests + build)
```

`internal/wschannel/redact_test.go` (plus `internal/redact/redact_test.go`,
the source-of-truth redactor it relies on) is the security-critical test:
asserts that neither the API key nor an OAuth access/refresh token leaks
into logs OR into errors returned to callers, even when the underlying
websocket library echoes the upgrade URL or `Authorization` header
verbatim.

### Live smoke tests (gated behind `smoke` build tag)

```bash
WSURL=ws://localhost:4010/console/websocket \
APIKEY=truestamp_... \
go test -tags=smoke -run TestSmoke ./internal/wschannel -v
```

Three smoke tests:

- `TestSmokeConsoleLobby` — connect + subscribe + ping + subscriptions.
- `TestSmokeClockTopic` — confirms `console:clock` ticks arrive.
- `TestSmokeLiveBlock` — subscribes to `blocks` and waits up to 90 s
  for a real cron-emitted block (validates the full
  PubSub → channel → wire path with live data).

`TestSmokeReconnect` requires a manual server restart during the
test window:

```bash
RECONNECT=1 WSURL=... APIKEY=... go test -tags=smoke -run TestSmokeReconnect ./internal/wschannel -v -count=1 -timeout=120s &
# In another shell:
mcp_preview restart phx     # or task serve restart, or whatever kills the server
```

The test confirms the client reconnects, rejoins all topics, and
receives ticks again — without dropping or hanging.

### Hand-rolled testing with `websocat`

The wire protocol is hand-writable. See
`truestamp-v2/docs/console_channel.md` § "Hand-rolled testing with
websocat" for a step-by-step recipe that doesn't involve this CLI at
all. This is the canonical way to debug protocol-layer issues without
the Bubble Tea event loop in the way.

## See also

- **Server-side authoritative reference**:
  [truestamp-v2/docs/console_channel.md](https://github.com/truestamp/truestamp-v2/blob/main/docs/console_channel.md)
- **Server PubSub patterns**:
  [truestamp-v2/docs/pubsub_architecture.md](https://github.com/truestamp/truestamp-v2/blob/main/docs/pubsub_architecture.md)
- **Item ownership / authorization**:
  [truestamp-v2/docs/item_ownership.md](https://github.com/truestamp/truestamp-v2/blob/main/docs/item_ownership.md)
