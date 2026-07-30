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


## Documents in this directory

| Document | Covers | Read it when |
| -------- | ------ | ------------ |
| [panes.md](panes.md) | Command surface, the four panes, key bindings, client-side coalescing, reconnection UX | Touching `internal/console` |
| [websocket.md](websocket.md) | The Phoenix Channels V2 client: OAuth on the upgrade, `token_expired` recovery, guarantees, sentinel push events | Touching `internal/wschannel` |
| [logging.md](logging.md) | Log destinations per platform, defaults, what gets logged, runtime inspection | Touching `internal/logging` |
| [limits.md](limits.md) | Server-side and client-side bounds, steady-state memory profile | Changing retention, caps, or buffers |
| [testing.md](testing.md) | Unit + offline tests, `smoke`-tagged live tests, hand-rolled `websocat` testing | Adding or debugging console tests |

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
internal/auth/                       OAuth 2.1 client + Authorizer abstraction (see kb/authentication.md)
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

## See also

- **Console wire surface (public reference)**:
  [truestamp-v2/kb/api/console-websocket.md](https://github.com/truestamp/truestamp-v2/blob/main/kb/api/console-websocket.md)
- **Console server wiring (internal reference)**:
  [truestamp-v2/kb/internal/console-wiring.md](https://github.com/truestamp/truestamp-v2/blob/main/kb/internal/console-wiring.md)
- **Server PubSub patterns**:
  [truestamp-v2/kb/internal/pubsub-architecture.md](https://github.com/truestamp/truestamp-v2/blob/main/kb/internal/pubsub-architecture.md)
- **Item ownership / authorization**:
  [truestamp-v2/kb/items/ownership-vs-authorship.md](https://github.com/truestamp/truestamp-v2/blob/main/kb/items/ownership-vs-authorship.md)
