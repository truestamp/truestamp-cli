# Console command surface and panes

The `truestamp console` TUI: how it is launched, the four panes, key bindings, client-side event coalescing, and reconnection behaviour. The socket underneath is documented in [websocket.md](websocket.md).

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
and captures every key except `ctrl+c`, `ctrl+tab` and
`ctrl+shift+tab`, so pane-nav digits and brackets land in the name
field.

| Key | Action |
| --- | ------ |
| `tab` / `shift+tab` / `up` / `down` | move between name, ownership radio, buttons |
| `left` / `right` | change the ownership model |
| `enter` | create |
| `esc` | cancel |
| `ctrl+c` | quit (raises the quit-confirmation prompt) |
| `ctrl+tab` / `ctrl+shift+tab` | switch panes — the only other bindings the modal doesn't capture |

**Connection** (`ConnectionKeys`):

| Key | Action |
| --- | ------ |
| `r` | re-run the external-service health probes (rate-limited to one manual run per 3s) |

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

