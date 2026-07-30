# Limits and bounds

Defense-in-depth values, none of which should fire under normal
interactive use.

## Server-side

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

## Client-side

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

## Memory profile (steady state)

- WebSocket connection: about 30 KB after `hibernate_after` kicks in.
- Event waterfall at 10 events/min over 24 h: roughly 14 400 events at
  about 200 B each, so **about 3 MB resident**.
- Worst case at sustained burst rates (very rare): up to the 100 000
  hard cap at about 250 B each, so **about 25 MB resident**.
- Log file: about 10 MB before rotation, capped at about 10 MB current
  plus about 2 MB per gzipped backup times 5, so **about 20 MB on
  disk**.

A days-long session in normal operation uses well under 50 MB total.

