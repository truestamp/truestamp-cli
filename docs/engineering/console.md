# `truestamp console`: interactive TUI architecture

The `truestamp console` subcommand opens an interactive Bubble Tea
TUI that holds a long-lived authenticated WebSocket to the Truestamp
backend. It's the live counterpart to the JSON:API: instead of
polling, you subscribe to streams and watch events flow in real time.

This document covers the **client** side: how the TUI is structured,
what the WebSocket client guarantees, where logs go, and what the
limits are. The wire protocol and server-side authoritative reference
live in the truestamp-v2 knowledge base
([`kb/api/console-websocket.md`](https://github.com/truestamp/truestamp-v2/blob/main/kb/api/console-websocket.md)
and [`kb/internal/console-wiring.md`](https://github.com/truestamp/truestamp-v2/blob/main/kb/internal/console-wiring.md)),
backed by `lib/truestamp_web/channels/console_{socket,channel,streams,projector}.ex`.

## Anatomy

```
cmd/console.go                       Cobra registration + --ws-url flag; first-run team picker;
                                     passes auth.Default() as Options.Authorizer and
                                     console.DefaultHealthTargets(...) as Options.HealthTargets
internal/console/                    The TUI
  app.go                             Bubble Tea root model, pane enum + switching, quit
                                     confirmation, reconnect bookkeeping, header/footer render;
                                     maps Options.Authorizer onto the wschannel OAuth/API-key options
  monitor.go                         Monitor pane: stream toggles + waterfall + detail panel
  newitem.go                         New Item pane: two-mode huh form + lifecycle card
  team.go                            Teams pane: membership table, set-active, access-loss banner
  teamcreate.go                      Create-team modal driven from the Teams pane
  connection.go                      Connection pane: scope, push counts, reconnects,
                                     external-service health table, log path, and the
                                     classified connect-failure diagnostic
  healthcheck.go                     Health-probe targets, HEAD-then-GET prober, poll cadence
  connerror.go                       Dial-error classifier; a dead/absent OAuth session routes
                                     to the "re-authenticate" hint
  scope.go                           activeScope: the shared "what team am I on" state
  messages.go                        tea.Msg types + waitForPush bridge; routes clock ticks,
                                     rejoin/reconnect events, tokenRefreshingMsg / authFailedMsg
  keys/keys.go                       Every keybinding, one struct per pane (help.KeyMap impls)
  events/events.go                   Wire push -> Row projection, compact kind, burst/outage rows
  chrome/                            Header (tabs, team label, status pill, clock), footer
                                     (bubbles/help), page layout, theme
internal/auth/                       OAuth 2.1 client + Authorizer abstraction (see CLAUDE.md "Authentication")
internal/wschannel/                  Phoenix Channel V2 client (homegrown)
  client.go                          Connection lifecycle, multi-topic, reconnect, OAuth
                                     access-token auth + token_expired recovery, in-band
                                     token.refresh keep-alive, redaction
  codec.go                           Frame encoder/decoder + ParseReply
  scope.go                           SwitchTeam helper (scope.switch_team push + reply decoding)
  redact_test.go                     api_key / Bearer token never leaks
  smoke_test.go                      Opt-in live-server tests (build tag: smoke)
internal/redact/redact.go            Single redaction source (api_key, Bearer, OAuth tokens/code/verifier)
internal/logging/logging.go          slog + lumberjack file logger (redacts via internal/redact)
```

The truestamp-v2 knowledge base is authoritative for everything that
flows over the wire (catalog, commands, events, burst shape). Whenever
the wire shape changes, that repo changes first; this document trails.

## Command surface

`truestamp console` takes no positional arguments. It refuses to start
when no credential is configured, pointing the user at
`truestamp auth login` or `TRUESTAMP_API_KEY`.

| Flag | Scope | Default | Notes |
| ---- | ----- | ------- | ----- |
| `--ws-url` | console only | derived from `--base-url` | Escape hatch for debugging or a non-standard WS path. The default is `<scheme>://<host>/console/websocket` computed in `internal/config` (`http` maps to `ws`, `https` to `wss`). |
| `--base-url` | global | `https://www.truestamp.com` | Origin the WebSocket URL and the Teams pane's JSON:API calls are derived from. |
| `--api-key` | global | `""` | Headless/CI credential. An explicit key outranks a stored OAuth session. |
| `--team` | global | `""` | Active team id. Also read from `config.toml`. |
| `--log-file` | global | `<user cache dir>/truestamp/truestamp.log` | Shared by every subcommand, not console-specific. |
| `--log-level` | global | `info` | `debug` \| `info` \| `warn` \| `error`. |
| `--http-timeout` | global | `10s` | Applies to the Teams pane's REST calls and the Connection pane's health probes. |
| `--no-color` | global | `false` | |

`--log-file` and `--log-level` are persistent root flags, not console
flags: the console consumes the logger the root `PersistentPreRunE`
already built, tagged with `component=console`.

Before the TUI starts, when no team is configured **and** stdin is a
TTY, `cmd/console.go` runs the same `huh` team picker
`truestamp team set` uses and persists the choice to `config.toml`.
Cancelling (Esc) leaves the server's personal-team fallback in place.

## Four panes

The TUI is a single Bubble Tea program with four switchable panes
sharing one WebSocket (`internal/console/app.go`, `pane` enum +
`numPanes = 4`):

1. **Monitor** (key `1`): left column lists the stream catalog with
   checkboxes; right column shows a scrollable waterfall of
   server-pushed events plus a Detail Panel underneath rendering the
   full untruncated payload of the selected row. All catalog streams
   are subscribed automatically once the welcome envelope arrives
   (`subscribeAllStreams`). Default focus is the waterfall, and the
   default order is reverse-chronological (newest at top).

2. **New Item** (key `2`): a `huh` form for registering a timestamped
   item, split into two groups so it covers both submission modes:

   - Group A (always shown): `Submission mode` select, `Name`,
     `Description`.
   - Group B (external-hash mode only): `Hash type`, `Hash`. The whole
     group is skipped when the mode is "Claims content as source of
     truth", and the description validator enforces the 32-character
     minimum inline in that mode.

   Submit pushes `items.create` with `watch: true` over the lobby, so
   the server auto-watches the new item. The pane then swaps into a
   watching state showing a summary card with live `item.*` state
   transitions flowing in below.

3. **Teams** (key `3`): the membership table (`GET /memberships`),
   with the active team marked from the shared `activeScope`. `enter`
   sets the cursor's team active: it pushes `scope.switch_team` over
   the live socket and persists via `config.SetTeam` on success. `c`
   opens the create-team modal. The pane auto-activates when the
   startup access check finds the configured team unreachable,
   surfacing a "membership lost" banner.

4. **Connection** (key `4`): diagnostics: scope summary (user, team,
   role, plan, active stream count), uptime, push counts by event,
   reconnect history, an external-service health table, and the log
   file path. On a hard connect failure the pane instead leads with a
   classified diagnostic (headline, the WS URL tried, "what to try"
   hints, the settings-file path, and the raw error in dim italic).
   No per-event noise; actionable errors go inline next to the action
   that produced them, not here.

The header renders the numbered pane tabs on the left and, on the
right, the active team label, a liveness status pill, and the
server-time clock. The status pill carries **liveness only**
(`connecting…`, `connected`, `reconnecting in Ns (attempt N)`,
`disconnected`, or a classified error); plan tier and stream count
live on the Connection pane where the user looks them up deliberately.
The footer is a `bubbles/help` component driven by the active pane's
`KeyMap`, so it is never hand-maintained.

### Key bindings

All bindings live in `internal/console/keys/keys.go`.

**Global** (`AppKeys`, consumed by the root model):

| Key | Action |
| --- | ------ |
| `]` / `ctrl+tab` | next pane (cyclic) |
| `[` / `ctrl+shift+tab` | previous pane (cyclic) |
| `1` / `2` / `3` / `4` | jump to Monitor / New Item / Teams / Connection |
| `?` | expand or collapse the footer help |
| `q` / `ctrl+c` | raise the "Really quit?" prompt (`y` or `enter` confirms, any other key cancels; a second `ctrl+c` quits outright) |

Plain `tab` is deliberately **not** handled at the root so the New
Item form can use it for field navigation. `q` is only consumed when
the active pane isn't accepting typed text, so a team or item name
containing "q" stays typable.

**Monitor** (`MonitorKeys` + `monitorModel.Update`):

| Key | Action |
| --- | ------ |
| `left` / `h` | focus the stream list |
| `right` / `l` | focus the waterfall |
| `up` / `k`, `down` / `j` | move the cursor on the focused side |
| `pgup` / `K`, `pgdn` / `J` | page the waterfall |
| `home` / `g`, `end` / `G` | jump to the top / bottom of the waterfall |
| `space` | toggle the cursor stream's subscription (list focus) |
| `r` | reverse chronological order (works from either focus) |
| `d` | show or hide the Detail Panel (works from either focus) |

**New Item** (`NewItemKeys` while entering, `NewItemWatchingKeys` after
submit):

| Key | Action |
| --- | ------ |
| `tab` / `down` | next field |
| `shift+tab` / `up` | previous field |
| `enter` | submit |
| `esc` | clear the form (entry state) |
| `n` / `esc` | start a fresh form (watching state) |

**Teams** (`TeamKeys`):

| Key | Action |
| --- | ------ |
| `up` / `k`, `down` / `j` | move the cursor |
| `enter` | set the cursor's team active (single press, no confirm step) |
| `c` | open the create-team modal |
| `r` | refresh memberships + active-team details |

**Create-team modal** (`TeamCreateKeys`): while open it is fully modal
and captures every key except `ctrl+c`, so pane-nav digits and
brackets land in the name field.

| Key | Action |
| --- | ------ |
| `tab` / `shift+tab` / `up` / `down` | move between name, ownership radio, buttons |
| `left` / `right` | change the ownership model |
| `enter` | create |
| `esc` | cancel |
| `ctrl+c` | quit (the only key that still leaves the modal for the app) |

**Connection** (`ConnectionKeys`):

| Key | Action |
| --- | ------ |
| `r` | re-run the external-service health probes (rate-limited to one manual run per 3s) |

## WebSocket client (`internal/wschannel`)

Homegrown Phoenix Channels V2 client (`client.go`, roughly 1.2k LOC).
The wire format is so small a third-party client (e.g. `nshafer/phx`)
is more code than the problem; controlling reconnect/heartbeat
behavior, and the OAuth `?access_token=` upgrade + `token_expired`
recovery below, matters precisely for a long-running TUI.

### Authentication (OAuth access token + `token_expired` recovery)

The console draws its credential from the process-wide `auth.Authorizer`
(`cmd/console.go` passes `auth.Default()` as `Options.Authorizer`;
`internal/console/app.go` maps it onto the wschannel options). Two modes:

- **OAuth** (`Authorizer.Mode() == ModeOAuth`): the access token is sent
  as the `?access_token=<jwt>` query param on the WebSocket **upgrade**
  (a Phoenix upgrade can't expose the `Authorization` header to the
  socket's `connect/3`, since `:x_headers` only captures `x-*` headers,
  so the token rides the query like `?api_key=` does). The client wires
  `Options.BearerToken = Authorizer.BearerToken` and pulls a *fresh*
  token on every (re)dial, so a reconnect after a token refresh
  automatically carries the new credential.
- **API key** (`ModeAPIKey`): unchanged. The key is resolved once in
  `console.Run` and sent as the `?api_key=` query param to the server's
  `connect/3` callback.

OAuth access tokens are short-lived; the token is validated at *connect*
(not at channel join), so re-authenticating means re-dialling the whole
socket. The recovery flow on a server `token_expired` push:

1. Emit a synthetic `token_refreshing` push (the TUI logs it and keeps
   pumping via `tokenRefreshingMsg`).
2. Call `Options.ForceRefresh` (the authorizer's `ForceRefresh`) **now**,
   so the upcoming reconnect dials with a genuinely new access token
   rather than re-presenting the just-rejected one (which would loop
   under client/server clock skew).
3. `dropConn`, then the normal reconnect-with-backoff path re-dials with
   the refreshed token and re-joins every topic.

**Fatal dead-session stop**: if the forced refresh fails because the
refresh token is expired/revoked/reused (`invalid_grant` produces
`auth.ErrSessionExpired`), `Options.FatalDialErr` classifies it as
permanently fatal. The client sets `authDead`, stops retrying, and emits
`auth_failed` (`authFailedMsg`) instead of looping forever re-presenting a
locally-"valid" but server-rejected token. The TUI flips to the
Connection pane with a re-login prompt; `connerror.go` also maps
`ErrSessionExpired` / `ErrNoCredentials` on the dial to the
"re-authenticate" hint rather than a network error.

**In-band keep-alive** (`keepAliveLoop` + `inBandRefresh`). When the caller
wires `wschannel.Options.AccessTokenExpiry` (the console does, from
`auth.Authorizer.AccessTokenExpiry`), a background loop polls every 15s
and, within 60s of the access token's expiry, force-refreshes it and
pushes `token.refresh {"access_token": <jwt>}` on `console:lobby` over
the *live* socket. The server re-validates the new token and reschedules
its disconnect timer, so a long session re-authenticates **without**
dropping/reconnecting and `token_expired` never fires. On any failure
(dead session, rejected token, or a delivery hiccup) it falls back to the
reactive `token_expired` -> re-dial path above. That path remains the
safety net for the asleep-past-expiry case.

### What it guarantees

- **Multi-topic on one socket**. `Connect` joins the primary topic
  (default `console:lobby`); `JoinTopic(ctx, topic)` joins any
  additional topic on the same WS. The TUI uses this to attach
  `console:clock` after the lobby; a clock-join failure is non-fatal
  and only costs the header clock.
- **Heartbeats**. 30-second heartbeat loop on the reserved `phoenix`
  topic. Configurable via `Options.HeartbeatInterval` (lowering is
  primarily a test affordance).
- **Ref correlation**. Every `Push(ctx, topic, event, payload)`
  allocates a unique ref and blocks for the matching `phx_reply`.
  Safe to call concurrently from multiple goroutines. Returns the
  `PhxReply{Status, Response}` directly; status `error` does NOT
  return a Go error, so the caller decides whether the domain error is
  fatal.
- **Reconnect**. Backoff over a fixed capped schedule
  (`1s, 2s, 5s, 10s, 30s`; the last value repeats). On each successful
  redial, every previously-joined topic is rejoined automatically and a
  synthetic `rejoined` push is emitted per topic over `Pushes()` so the
  application can replay any in-channel state the server doesn't
  remember (subscriptions, item watches).
- **Reconnect status events**. Before each dial attempt the client
  emits a synthetic `reconnecting` push with `{attempt, next_attempt_at}`
  the TUI uses for its countdown header.
- **Drained pending on disconnect**. In-flight `Push` calls receive
  a synthetic `phx_reply` with `{status: "error", reason: "connection
  lost during reconnect"}` so they unblock immediately rather than
  hanging on the original ref.
- **Two-stage readiness gates**. Internally the client distinguishes
  "socket alive" (`socketReady`, the rejoin path can send) from
  "session ready" (`sessionReady`, application calls can send).
  Application `Push`/`JoinTopic` waits on the session gate so callers
  never race the rejoin replay.
- **Drop-on-full inbound**. The pushes channel is buffered at 256;
  if the consumer falls behind, frames are dropped (logged at
  `warn`) rather than blocking the reader (which would also block
  the heartbeat).
- **Secret redaction**. The websocket library's dial errors echo the
  upgrade URL verbatim, including `api_key=…` and `access_token=…`.
  `wschannel` flows every error through `internal/redact` before
  returning to the caller AND before logging via the slog logger. A
  token error from the authorizer is wrapped with `redact.WrapError`
  so it stays `errors.Is(ErrSessionExpired)` for the fatal-session
  check while never leaking token bytes. Neither the `truestamp_…` key
  nor an OAuth access/refresh token can reach the UI, the log file, or
  any stderr the host process owns. `redact` is the single source of
  truth (shared with `internal/logging`); see CLAUDE.md
  "Authentication" for the full pattern set.

### What it does NOT do

- **OAuth**: token *expiry* is recovered automatically (see
  Authentication above: in-band `token.refresh` first, then
  `token_expired` -> force-refresh -> re-dial -> re-join). A genuinely
  dead session (refresh token revoked/expired) stops reconnect and
  prompts re-login rather than looping.
- **API key**: no automatic reauthentication. If the server rejects the
  key, reconnect attempts keep failing. Fix the key, restart the CLI.
- No exponential backoff jitter. The schedule is a fixed table. Fine at
  single-user scale; would matter at thousand-client thundering-herd
  scale.
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
gzip-compressed log files. The TUI never writes to stdout/stderr:
those belong to Bubble Tea's renderer.

The log file is **shared by every `truestamp` subcommand**, not
console-specific. Records emitted while the console is running carry
`component=console`, so filter on that when reading.

### Default destination (per platform)

| Platform | Path                                                  |
| -------- | ----------------------------------------------------- |
| macOS    | `~/Library/Caches/truestamp/truestamp.log`            |
| Linux    | `~/.cache/truestamp/truestamp.log`                    |
| Windows  | `%LOCALAPPDATA%\truestamp\Cache\truestamp.log`        |

Override with `--log-file <path>`. `logging.DefaultPath()` computes the
same value and is what the flag's help text prints.

The file is opened lazily on the first write (`lazyWriter`), so an
invocation that emits no records never touches the filesystem.

### Defaults

| Knob          | Default       | Override                                |
| ------------- | ------------- | --------------------------------------- |
| Log level     | `info`        | `--log-level debug\|info\|warn\|error`  |
| Rotation size | 10 MB         | `logging.Options.MaxSizeMB` (no flag)   |
| Retention     | 14 days       | `logging.Options.MaxAgeDays` (no flag)  |
| Backups       | 5 files       | `logging.Options.MaxBackups` (no flag)  |
| Compression   | gzip          | (constant, always on)                   |

Worst-case disk: roughly `MaxSize x (MaxBackups + 1)` before
compression, and 10 to 15 MB after.

### What gets logged

Routed through `logging` from `wschannel.Client` and the panes:

- `info`: session start (with `ws_url`, `log_path`, `version`),
  `console connected`, successful reconnects (with `downtime`,
  `attempts`), in-band OAuth token refreshes.
- `info`: `reconnect dial failed` for each individual backoff
  attempt during an outage, and `ws read failed` when a socket ends.
  Both are routine during a reconnect cycle.
- `warn`: `ws write failed`, `frame decode failed`, `dropped push
  (consumer slow)`, `dropped rejoin notice`, `reconnect rejoin
  failed`, `fatal auth error; stopping reconnect`, `subscribe replay
  after reconnect failed`, `auto-subscribe-all failed`, `clock topic
  join failed`, `console connect failed`, `oauth session dead;
  reconnect stopped`, and the `in-band refresh: …` failure family.

Nothing in the console path emits at `debug` today, so
`--log-level debug` only widens what other subcommands and
dependencies produce.

Errors that the user can act on (form validation, auth failures,
server-rejected commands) do **not** flow through this logger:
they're surfaced inline next to the action that triggered them. The
logger is a transport diagnostic, not a global error log.

### Inspecting at runtime

```bash
tail -f ~/Library/Caches/truestamp/truestamp.log | jq 'select(.component == "console")'
```

The Connection pane's Logs section shows the live log file path with a
hint pointing at this command. The config file path is surfaced too,
but only inside the connect-failure diagnostic, where a user who can't
sign in needs it.

### Adding more sinks later

Slog's handler model is composable; a future debug overlay (toggleable
key in the TUI) could wrap the file handler with an in-memory ring
buffer and surface the last N entries on demand without changing any
emit-site call.

## Limits and bounds

Defense-in-depth values, none of which should fire under normal
interactive use.

### Server-side

| Limit                                  | Default              | Where                              |
| -------------------------------------- | -------------------- | ---------------------------------- |
| Max active streams per connection      | 32                   | `console_channel.ex` `@max_streams` |
| Max streams per subscribe/unsubscribe call | 64               | `console_channel.ex` `@max_streams_per_call` |
| Inbound command rate limit             | 60 / 10s             | `console_channel.ex` `@cmd_window_ms` + `:console_cmd_limit` |
| Outbound stream-push rate limit        | 200 / s              | `console_channel.ex` `@out_rate_window_ms` + `:console_out_rate_limit` |
| Per-payload encoded size cap           | 16 384 bytes         | `console_channel.ex` `@payload_max_bytes` |
| Coalesce window                        | 500 ms               | `console_channel.ex` `@coalesce_window_ms` |
| Phoenix `max_frame_size` (inbound)     | 65 536 bytes         | `truestamp_web/endpoint.ex`        |
| `hibernate_after`                      | 15 000 ms            | `console_socket.ex`                |

### Client-side

| Limit                                  | Default              | Where                                           |
| -------------------------------------- | -------------------- | ----------------------------------------------- |
| Per-frame inbound read limit           | 1 MB                 | `wschannel.Client.dial` (`SetReadLimit`)        |
| `Pushes()` buffer (drop-on-full)       | 256 frames           | `wschannel.Options.PushBufferSize`              |
| Outbound queue                         | 64 frames            | `wschannel.Client.out`                          |
| Reconnect backoff                      | 1, 2, 5, 10, 30s     | `wschannel.reconnectBackoff`                    |
| Heartbeat interval                     | 30 s                 | `wschannel.Options.HeartbeatInterval`           |
| Keep-alive poll / refresh lead         | 15 s / 60 s          | `wschannel.Client.keepAliveLoop`                |
| Connect timeout                        | 15 s                 | `console.Run`                                   |
| Event waterfall retention              | 24 hours             | `internal/console/monitor.go::eventRetention`   |
| Event waterfall hard cap               | 100 000 events       | `internal/console/monitor.go::eventHardCap`     |
| New Item lifecycle log cap             | 100 transitions      | `internal/console/newitem.go::maxTransitions`   |
| Health probe timeout                   | 5 s                  | `internal/console/healthcheck.go::healthCheckTimeout` |
| Health probe poll cadence              | 1 minute             | `internal/console/healthcheck.go::healthCheckPollInterval` |
| Manual health refresh floor            | 3 s                  | `internal/console/healthcheck.go::healthCheckMinInterval` |

The Connection pane keeps no error log of its own: individual
transport errors go to the file logger, and hard connect failures
render as a single classified diagnostic section.

### Memory profile (steady state)

- WebSocket connection: about 30 KB after `hibernate_after` kicks in.
- Event waterfall at 10 events/min over 24 h: roughly 14 400 events at
  about 200 B each, so **about 3 MB resident**.
- Worst case at sustained burst rates (very rare): up to the 100 000
  hard cap at about 250 B each, so **about 25 MB resident**.
- Log file: about 10 MB before rotation, capped at about 10 MB current
  plus about 2 MB per gzipped backup times 5, so **about 20 MB on
  disk**.

A days-long session in normal operation uses well under 50 MB total.

## Coalescing, on the client side

The server may emit `<resource>.burst` summary pushes when many
events of the same stream arrive within 500 ms. The client has no
special burst renderer: `events.Project` tags any kind ending in
`.burst` with `SeverityBurst`, puts the payload's `count` in the ID
column, and the waterfall renders it as an ordinary Time / Kind / ID
row with a dim count badge in the resource's own accent color.

```
14:42:11.500  item.burst                                                437
```

For entropy bursts the stream id is spliced into the kind
(`entropy.burst` on stream `entropy.bitcoin` renders as
`entropy.bitcoin.burst`, via `events.burstKind`).

The rest of the burst payload (`window_ms`, `first_at`, `last_at`,
`by_kind`, `by_state`, and the nested `latest` projection) is
preserved verbatim on `Row.Payload` and rendered by the Detail Panel
when the row is selected: maps are flattened one level deep
(`by_kind.created`, `by_state.processing`), and `orderedPayloadKeys`
puts the burst fields near the top of the panel's canonical order.

The client never decides whether to coalesce, that's a server-side
decision. It just renders any `*.burst` kind that arrives, with the
same row template as everything else. See the truestamp-v2 knowledge
base for the authoritative coalescer behavior.

## Reconnection UX

When the wschannel session loop detects a disconnect:

1. The header status flips to `reconnecting in Ns (attempt N)` with a
   live 1-second countdown driven by `reconnectTickMsg`.
2. A `server.down` outage marker is dropped into the Monitor
   waterfall immediately, repeated every 10 seconds
   (`outageMarkerInterval`) while the outage continues.
3. On successful reconnect:
   - A closing `server.up` marker shows the total downtime in the ID
     column.
   - All previously active streams are re-subscribed (via
     `monitor.replayAfterReconnect`).
   - The Connection pane's reconnect counter increments and total
     downtime accumulates.
   - The header reverts to the `connected` state.

Outage markers are synthetic `events.Row` values built by
`events.Outage`: `SeverityOutage` renders them italic in the error
color, they carry no stream id (so they can never collide with a real
subscription), and their payload holds `since`, `duration`, and a
plain-language `message` for the Detail Panel. They participate in
normal scrollback so the user can scroll back through days of history
and see exactly when data went missing.

## Extension points

The architecture has deliberate hooks for future growth without
touching the wire shape:

- **New stream id**. Add to `ConsoleStreams.@global_streams` (server)
  plus the `pubsub_topics/2` mapping. Client picks it up automatically
  from the welcome envelope and renders any matching `<resource>.*`
  events using the existing color scheme.
- **New burst-eligible stream**. Already universal: every stream
  flows through the coalescer. The first-event-immediate rule means
  bursts only emerge when input rate genuinely warrants them. No
  config change needed.
- **New pane**. Add a `*.go` file in `internal/console/`, register it
  in `app.go`'s pane enum, bump `numPanes`, extend `pane.title()` and
  the `Update`/`View` switches, add a `KeyMap` in
  `internal/console/keys/`, and route any pushes it cares about via
  its `handlePush` method. The wschannel client doesn't need to know
  about new panes.
- **New command**. Add a `dispatch/3` clause server-side; call
  `client.Push(ctx, lobbyTopic, "<event>", payload)` client-side.

## Testing

### Unit + offline tests

```bash
task test          # everything
task precommit     # full gate (fmt + lint + vuln-check + tests + build)
```

`task lint` is itself `go vet` + `staticcheck` + `gosec`.

`internal/wschannel/redact_test.go` (plus `internal/redact/redact_test.go`,
the source-of-truth redactor it relies on) is the security-critical test:
it asserts that neither the API key nor an OAuth access/refresh token
leaks into logs OR into errors returned to callers, even when the
underlying websocket library echoes the upgrade URL verbatim.

Other offline coverage worth knowing about: `chaos_test.go` and
`testserver_test.go` drive a fake Phoenix server through disconnects,
`fuzz_codec_test.go` fuzzes the V2 frame decoder, `keepalive_test.go`
pins `keepAliveDue`, and the console package has renderer, key, form,
health-check and connection-error tests.

### Live smoke tests (gated behind the `smoke` build tag)

```bash
WSURL=ws://localhost:4010/console/websocket \
APIKEY=truestamp_... \
go test -tags=smoke -run TestSmoke ./internal/wschannel -v
```

Every smoke test skips unless both `WSURL` and `APIKEY` are set:

- `TestSmokeConsoleLobby`: connect + subscribe + ping + subscriptions.
- `TestSmokeClockTopic`: confirms `console:clock` ticks arrive (two
  ticks within about 3 seconds).
- `TestSmokeItemsCreate`: `items.create` with auto-watch, then waits
  briefly for at least one `item.*` push.
- `TestSmokeLiveBlock`: subscribes to `blocks` and waits for a real
  cron-emitted block, validating the full PubSub to channel to wire
  path with live data.

`TestSmokeReconnect` additionally requires `RECONNECT=1` and a manual
server restart during the test window:

```bash
RECONNECT=1 WSURL=... APIKEY=... go test -tags=smoke -run TestSmokeReconnect ./internal/wschannel -v -count=1 -timeout=120s &
# In another shell, restart the running Truestamp server.
```

The test counts pre- and post-reconnect ticks on `console:clock` to
prove the client reconnected, rejoined all topics, and is receiving
pushes from the new server process, without dropping or hanging.

### Hand-rolled testing with `websocat`

The wire protocol is hand-writable. The truestamp-v2 knowledge base
concept [`kb/internal/console-wiring.md`](https://github.com/truestamp/truestamp-v2/blob/main/kb/internal/console-wiring.md)
carries a step-by-step `websocat` recipe that doesn't involve this CLI
at all. That is the canonical way to debug protocol-layer issues
without the Bubble Tea event loop in the way.

## See also

- **Console wire surface (public reference)**:
  [truestamp-v2/kb/api/console-websocket.md](https://github.com/truestamp/truestamp-v2/blob/main/kb/api/console-websocket.md)
- **Console server wiring (internal reference)**:
  [truestamp-v2/kb/internal/console-wiring.md](https://github.com/truestamp/truestamp-v2/blob/main/kb/internal/console-wiring.md)
- **Server PubSub patterns**:
  [truestamp-v2/kb/internal/pubsub-architecture.md](https://github.com/truestamp/truestamp-v2/blob/main/kb/internal/pubsub-architecture.md)
- **Item ownership / authorization**:
  [truestamp-v2/kb/items/ownership-vs-authorship.md](https://github.com/truestamp/truestamp-v2/blob/main/kb/items/ownership-vs-authorship.md)
