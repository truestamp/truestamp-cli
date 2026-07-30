# WebSocket client (`internal/wschannel`)

Homegrown Phoenix Channels V2 client (`client.go`, roughly 1.2k LOC).
The wire format is so small a third-party client (e.g. `nshafer/phx`)
is more code than the problem; controlling reconnect/heartbeat
behavior, and the OAuth `?access_token=` upgrade + `token_expired`
recovery below, matters precisely for a long-running TUI.

## Authentication (OAuth access token + `token_expired` recovery)

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

## What it guarantees

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
  truth (shared with `internal/logging`); see `kb/authentication.md`
  for the full pattern set.

## What it does NOT do

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

## Sentinel push events

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

