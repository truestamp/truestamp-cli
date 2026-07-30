# Logging (`internal/logging`)

Stdlib `log/slog` JSON handler over
`gopkg.in/natefinch/lumberjack.v2` for size-rotated, age-retained,
gzip-compressed log files. The TUI never writes to stdout/stderr:
those belong to Bubble Tea's renderer.

The log file is **shared by every `truestamp` subcommand**, not
console-specific. Records emitted while the console is running carry
`component=console`, so filter on that when reading.

## Default destination (per platform)

| Platform | Path                                                  |
| -------- | ----------------------------------------------------- |
| macOS    | `~/Library/Caches/truestamp/truestamp.log`            |
| Linux    | `~/.cache/truestamp/truestamp.log`                    |
| Windows  | `%LOCALAPPDATA%\truestamp\Cache\truestamp.log`        |

Override with `--log-file <path>`. `logging.DefaultPath()` computes the
same value and is what the flag's help text prints.

The file is opened lazily on the first write (`lazyWriter`), so an
invocation that emits no records never touches the filesystem.

## Defaults

| Knob          | Default       | Override                                |
| ------------- | ------------- | --------------------------------------- |
| Log level     | `info`        | `--log-level debug\|info\|warn\|error`  |
| Rotation size | 10 MB         | `logging.Options.MaxSizeMB` (no flag)   |
| Retention     | 14 days       | `logging.Options.MaxAgeDays` (no flag)  |
| Backups       | 5 files       | `logging.Options.MaxBackups` (no flag)  |
| Compression   | gzip          | (constant, always on)                   |

Worst-case disk: roughly `MaxSize x (MaxBackups + 1)` before
compression, and 10 to 15 MB after.

## What gets logged

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

## Inspecting at runtime

```bash
tail -f ~/Library/Caches/truestamp/truestamp.log | jq 'select(.component == "console")'
```

The Connection pane's Logs section shows the live log file path with a
hint pointing at this command. The config file path is surfaced too,
but only inside the connect-failure diagnostic, where a user who can't
sign in needs it.

## Adding more sinks later

Slog's handler model is composable; a future debug overlay (toggleable
key in the TUI) could wrap the file handler with an in-memory ring
buffer and surface the last N entries on demand without changing any
emit-site call.

