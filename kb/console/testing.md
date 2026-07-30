# Testing

## Unit + offline tests

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

## Live smoke tests (gated behind the `smoke` build tag)

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

## Hand-rolled testing with `websocat`

The wire protocol is hand-writable. The truestamp-v2 knowledge base
concept [`kb/internal/console-wiring.md`](https://github.com/truestamp/truestamp-v2/blob/main/kb/internal/console-wiring.md)
carries a step-by-step `websocat` recipe that doesn't involve this CLI
at all. That is the canonical way to debug protocol-layer issues
without the Bubble Tea event loop in the way.

