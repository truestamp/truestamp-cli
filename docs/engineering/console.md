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
cmd/console.go                       Cobra registration + flag plumbing
internal/console/                    The TUI
  app.go                             Bubble Tea root model, header/footer, pane switching
  monitor.go                         Monitor pane: stream toggles + scrollable waterfall
  newitem.go                         New Item pane: form + lifecycle card
  connection.go                      Connection pane: scope, push counts, reconnect summary, log path
  messages.go                        tea.Msg types + waitForPush bridge
internal/wschannel/                  Phoenix Channel V2 client (homegrown, ~600 LOC)
  client.go                          Connection lifecycle, multi-topic, reconnect, redaction
  codec.go                           Frame encoder/decoder
  redact_test.go                     api_key never leaks
  smoke_test.go                      Opt-in live-server tests (build tag: smoke)
internal/logging/logging.go          slog + lumberjack file logger
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

Homegrown ~600-line Phoenix Channels V2 client. The wire format is so
small a third-party client (e.g. `nshafer/phx`) is more code than the
problem; controlling reconnect/heartbeat behavior precisely matters
for a long-running TUI.

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
- **API-key redaction**. The websocket library's dial errors echo the
  upgrade URL verbatim, including `api_key=…`. `wschannel` redacts
  every error before returning to the caller AND before logging via
  the slog logger. The `truestamp_…` token can never reach the UI,
  the log file, or any stderr the host process owns.

### What it does NOT do

- No automatic reauthentication on a `4401` close. If the server
  rejects the API key, reconnect attempts will keep failing — fix
  the key, restart the CLI.
- No exponential backoff jitter. Fine at single-user scale; would
  matter at thousand-client thundering-herd scale.
- No subscription persistence across CLI restarts. Each launch
  starts fresh and auto-subscribes to the full catalog.

### Sentinel push events

`internal/wschannel` exports two synthetic event-name constants used
by application code to special-case reconnect lifecycle:

| Constant                      | Wire value      | When                                                                |
| ----------------------------- | --------------- | ------------------------------------------------------------------- |
| `wschannel.ReconnectingEvent` | `"reconnecting"`| Emitted before each dial attempt during a reconnect cycle.          |
| `wschannel.ReconnectedEvent`  | `"rejoined"`    | Emitted per topic after a successful redial+rejoin.                 |

Neither is sent by the server; both are synthetic, injected by the
client into `Pushes()` for the application to observe.

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

`internal/wschannel/redact_test.go` is the security-critical test:
asserts the API key never leaks into logs OR into errors returned
to callers, even when the underlying websocket library echoes the
upgrade URL verbatim.

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
