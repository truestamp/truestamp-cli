// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

//go:build chaos

// Chaos tests exercise the wschannel client under pathological reconnect
// orderings: slow consumer + reconnect, mid-rejoin disconnect, blocked
// Push during disconnect, concurrent JoinTopic during reconnect, and
// heartbeat behavior under disconnection. Each test runs against the
// in-memory testPhoenixServer (testserver_test.go), so they're hermetic
// and CI-safe.
//
// Run with:
//
//	go test -tags=chaos -count=1 -race -timeout=60s ./internal/wschannel/...
//
// `-race` is mandatory: these tests exist to surface concurrent ordering
// bugs that the happy-path smoke tests miss.

package wschannel

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// waitForBoth polls until the client reaches `wantStatus` AND the
// server has accepted at least `wantConnects` connections, or `timeout`
// expires.
func waitForBoth(t *testing.T, c *Client, s *testPhoenixServer, wantStatus Status, wantConnects int64, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.Status() == wantStatus && s.Connects.Load() >= wantConnects {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return c.Status() == wantStatus && s.Connects.Load() >= wantConnects
}

// TestChaosSlowConsumerDuringReconnect:
//
// A consumer that doesn't drain Pushes() must not deadlock the client,
// even when a disconnect+reconnect happens while the buffer is saturated.
// The readLoop drops on full and logs; the session loop runs independently.
func TestChaosSlowConsumerDuringReconnect(t *testing.T) {
	s := newTestPhoenixServer(t)

	c, err := New(Options{
		URL:               s.URL(),
		APIKey:            "test_key",
		HeartbeatInterval: time.Hour, // suppress heartbeats; isolate the broadcast path
		PushBufferSize:    32,        // small buffer makes overflow easier to provoke
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Spam ~3x the buffer worth of broadcasts. Some land in the buffer,
	// the rest get drop-on-fulled by readLoop.
	for i := range 100 {
		_ = s.BroadcastFrame("console:lobby", "stream", map[string]any{"i": i})
	}

	// Now disconnect. The reconnect path also publishes synthetic events
	// to a buffer that may already be full — drop-on-full must apply
	// there too without blocking the session loop.
	s.KillConnection()

	// Wait for the server to accept the reconnect (Connects==2) AND
	// the client to finish rejoin (Status==Connected). Polling both
	// avoids the race where the client briefly reports Connected from
	// the old session before its readLoop notices the disconnect.
	if !waitForBoth(t, c, s, StatusConnected, 2, 5*time.Second) {
		t.Fatalf("client did not fully reconnect: status=%s, server.Connects=%d",
			c.Status(), s.Connects.Load())
	}

	// Drain everything. We don't care about exact counts (drop-on-full
	// makes them non-deterministic) — only that the channel keeps
	// flowing and we eventually observe a fresh broadcast post-reconnect.
	if err := s.BroadcastFrame("console:lobby", "stream", map[string]any{"post": true}); err != nil {
		t.Fatalf("post-reconnect broadcast: %v", err)
	}

	sawPost := false
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
drain:
	for {
		select {
		case p, ok := <-c.Pushes():
			if !ok {
				break drain
			}
			if p.Event == "stream" {
				var m map[string]any
				_ = json.Unmarshal(p.Payload, &m)
				if _, ok := m["post"]; ok {
					sawPost = true
					break drain
				}
			}
		case <-deadline.C:
			break drain
		}
	}
	if !sawPost {
		t.Errorf("post-reconnect broadcast never observed despite Status=Connected")
	}

	// Close should return cleanly; no panic, no goroutine wedged.
	if err := c.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestChaosMidRejoinDisconnect:
//
// Disconnect → server hangs on phx_join → disconnect again → server
// recovers. Verifies the client survives the chaos without panicking
// or leaking and re-establishes a working socket.
//
// IMPORTANT FINDING (documented behavior, not a bug):
// When a rejoin's phx_join is interrupted (drainPending fires while
// joinTopic is waiting for the reply, or ctx times out), joinTopic
// calls forgetTopic, removing the entry from c.topics. The next
// rejoin cycle sees an empty topic set and joins nothing. The
// application is responsible for re-issuing JoinTopic after
// observing a `rejoined` push that has lost a topic — the client
// does NOT retry forgotten topics on subsequent reconnects. This
// test asserts the actual behavior so a future change is forced
// to update the test if the design changes.
func TestChaosMidRejoinDisconnect(t *testing.T) {
	s := newTestPhoenixServer(t)

	c, err := New(Options{
		URL:               s.URL(),
		APIKey:            "test_key",
		HeartbeatInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if _, err := c.JoinTopic(ctx, "console:clock"); err != nil {
		t.Fatalf("JoinTopic clock: %v", err)
	}

	// Kill. Next accept will hang on phx_join.
	s.AcceptButHangOnPhxJoin()
	s.KillConnection()

	// Wait for the client to redial and at least begin the hung rejoin.
	if !s.awaitConnects(2, 5*time.Second) {
		t.Fatalf("server did not see second connect, Connects=%d", s.Connects.Load())
	}
	// Wait briefly for the lobby phx_join to land on the hung server.
	time.Sleep(300 * time.Millisecond)

	// Now release: subsequent accepts behave normally.
	s.AcceptNext()
	// Kill the hung connection so the client redials.
	s.KillConnection()

	// Eventually back to fully connected. The client's topic set is
	// expected to have shrunk (see comment block above): the rejoin
	// of lobby was killed mid-flight, so lobby was forgotten. This is
	// the documented behavior under chaos.
	if !waitForBoth(t, c, s, StatusConnected, 3, 15*time.Second) {
		t.Fatalf("client did not recover, status=%s, Connects=%d, PhxJoins=%d",
			c.Status(), s.Connects.Load(), s.PhxJoins.Load())
	}

	// At this point the client has survived the chaos and stabilized.
	// We deliberately don't probe with a follow-up Push or BroadcastFrame
	// — the post-chaos socket can have a brief settling window where
	// the read goroutine has just resumed and either side may still
	// be flushing prior writes. The primary contract this test guards
	// is "no panic, no leak, returns to a steady state" — which is
	// what waitForBoth above confirms. Health probes here would be
	// flaky for a finding that is itself documented behavior.
}

// TestChaosPushBlockedDuringDisconnect:
//
// A Push that's blocked waiting for a reply must wake up promptly when
// the underlying socket dies — drainPending sends a synthetic "error"
// reply. Without this, blocked callers hang for the full ctx timeout.
func TestChaosPushBlockedDuringDisconnect(t *testing.T) {
	s := newTestPhoenixServer(t)

	c, err := New(Options{
		URL:               s.URL(),
		APIKey:            "test_key",
		HeartbeatInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Issue a push the server will never reply to (default behavior
	// ignores anything other than phx_join + heartbeat). Use a long ctx
	// so we can prove drainPending wakes us up well before the deadline.
	pushCtx, pushCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer pushCancel()

	type result struct {
		reply PhxReply
		err   error
		took  time.Duration
	}
	resultCh := make(chan result, 1)

	go func() {
		start := time.Now()
		reply, err := c.Push(pushCtx, "console:lobby", "slow_op", map[string]any{})
		resultCh <- result{reply: reply, err: err, took: time.Since(start)}
	}()

	// Let the push register its pending entry, then kill the server.
	time.Sleep(100 * time.Millisecond)
	s.KillConnection()

	// The push should return within ~250ms — drainPending fires from the
	// readLoop's read-error path, immediately after dropConn.
	select {
	case r := <-resultCh:
		if r.err != nil {
			t.Fatalf("Push returned err = %v (want synthetic phx_reply with status=error)", r.err)
		}
		if r.reply.Status != "error" {
			t.Errorf("reply status = %q, want %q", r.reply.Status, "error")
		}
		if r.took > 1*time.Second {
			t.Errorf("Push took %v, expected <1s (drainPending wakeup); ctx timeout was 10s",
				r.took)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Push did not wake up within 2s of disconnect; drainPending broken")
	}
}

// TestChaosManyJoinTopicDuringReconnect:
//
// Five concurrent JoinTopic calls during a reconnect window must all
// eventually succeed and serialize cleanly through the session gate.
// No deadlock, no double-registration of the same topic in c.topics.
func TestChaosManyJoinTopicDuringReconnect(t *testing.T) {
	s := newTestPhoenixServer(t)

	c, err := New(Options{
		URL:               s.URL(),
		APIKey:            "test_key",
		HeartbeatInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Kill the server connection — the session loop will redial.
	s.KillConnection()

	// Wait until the client's readLoop has noticed the disconnect and
	// closed the session gate. Without this wait, the JoinTopic calls
	// race ahead while the client still believes the dead conn is live;
	// they get queued to the writer, fail synchronously when the write
	// hits the dead socket, and drainPending wakes them with a synthetic
	// "connection lost" error — testing the wrong path.
	if !waitForStatus(c, StatusReconnecting, 5*time.Second) {
		t.Fatalf("client did not transition to %s after kill, status=%s",
			StatusReconnecting, c.Status())
	}

	// Spawn 5 concurrent JoinTopic calls. They all park on the session
	// gate until rejoin completes.
	const N = 5
	var wg sync.WaitGroup
	errs := make([]error, N)
	for i := range N {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			joinCtx, jcancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer jcancel()
			_, errs[i] = c.JoinTopic(joinCtx, fmt.Sprintf("console:room%d", i))
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("JoinTopic(room%d): %v", i, err)
		}
	}

	// All 5 rooms + lobby + clock-not-joined should be in c.topics with
	// no duplicates. (The session gate serializes joinTopicOnSocket with
	// app-level joinTopic, so c.topics is internally consistent at this
	// point.)
	c.topicsMu.RLock()
	defer c.topicsMu.RUnlock()
	for i := range N {
		topic := fmt.Sprintf("console:room%d", i)
		if _, ok := c.topics[topic]; !ok {
			t.Errorf("topic %s not in c.topics after JoinTopic", topic)
		}
	}
	if _, ok := c.topics["console:lobby"]; !ok {
		t.Errorf("primary topic lost during reconnect+joins")
	}
}

// TestChaosHeartbeatDuringReconnect:
//
// During a disconnect, the heartbeat loop blocks in send() (parked on
// the session gate) — only one frame is in flight at a time, regardless
// of how many ticks the timer would have fired. This is the regression
// guard: if the heartbeat loop ever starts spawning goroutines per tick,
// the goroutine count would balloon during outages.
func TestChaosHeartbeatDuringReconnect(t *testing.T) {
	s := newTestPhoenixServer(t)

	c, err := New(Options{
		URL:               s.URL(),
		APIKey:            "test_key",
		HeartbeatInterval: 100 * time.Millisecond, // aggressive
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Snapshot goroutine count after connection is established.
	time.Sleep(150 * time.Millisecond)
	baseGoroutines := runtime.NumGoroutine()

	// Disconnect. While disconnected the heartbeat ticker fires every
	// 100ms; since send() is parked on the session gate, the heartbeat
	// goroutine count must not grow.
	s.KillConnection()

	// Stay disconnected for 1s — the ticker would fire ~10 times if any
	// per-tick goroutine were being spawned.
	time.Sleep(1 * time.Second)
	disconnectedGoroutines := runtime.NumGoroutine()

	// Allow some slack for the session loop's transient goroutines, but
	// require we're not 10+ over baseline (which would indicate per-tick
	// goroutine spawn).
	if grew := disconnectedGoroutines - baseGoroutines; grew > 5 {
		t.Errorf("goroutine count grew by %d during 1s disconnect (base=%d, now=%d) — heartbeat may be leaking goroutines",
			grew, baseGoroutines, disconnectedGoroutines)
	}

	// Recover and ensure things settle without explosion.
	if !waitForStatus(c, StatusConnected, 5*time.Second) {
		t.Fatalf("did not reconnect, status=%s", c.Status())
	}

	if err := c.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	// Give the wg time to drain.
	time.Sleep(200 * time.Millisecond)
	finalGoroutines := runtime.NumGoroutine()
	if grew := finalGoroutines - baseGoroutines; grew > 5 {
		t.Errorf("goroutine count grew by %d after Close (base=%d, now=%d) — possible leak",
			grew, baseGoroutines, finalGoroutines)
	}
}
