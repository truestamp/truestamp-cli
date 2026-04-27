// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestStatusInitialState verifies a freshly New()'d client reports
// StatusInit before Connect is called.
func TestStatusInitialState(t *testing.T) {
	t.Parallel()

	c, err := New(Options{URL: "ws://example.invalid/ws", APIKey: "test_key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if got := c.Status(); got != StatusInit {
		t.Errorf("Status() = %s, want %s", got, StatusInit)
	}
}

// TestStatusDuringConnect drives a server that accepts the upgrade but
// never replies to phx_join. Status() must report StatusConnecting (never
// StatusConnected) until Connect aborts.
func TestStatusDuringConnect(t *testing.T) {
	t.Parallel()

	s := newTestPhoenixServer(t)
	s.AcceptButHangOnPhxJoin()

	c, err := New(Options{URL: s.URL(), APIKey: "test_key", HeartbeatInterval: 5 * time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	// Spawn Connect with a short ctx so it aborts after the join hangs.
	connectCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	connectErr := make(chan error, 1)
	go func() {
		_, err := c.Connect(connectCtx)
		connectErr <- err
	}()

	// While Connect is in flight, Status must never reach StatusConnected.
	// Sample a handful of times across the 500ms window.
	deadline := time.Now().Add(450 * time.Millisecond)
	var sawConnecting atomic.Bool
	for time.Now().Before(deadline) {
		st := c.Status()
		if st == StatusConnected {
			t.Fatalf("Status() reached %s while phx_join was hanging", st)
		}
		if st == StatusConnecting {
			sawConnecting.Store(true)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !sawConnecting.Load() {
		t.Errorf("never observed StatusConnecting during a hung connect")
	}

	// Connect should ultimately fail with ctx deadline.
	select {
	case err := <-connectErr:
		if err == nil {
			t.Errorf("Connect succeeded unexpectedly when phx_join was hung")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Connect did not return after ctx deadline")
	}
}

// TestStatusAfterConnect verifies the steady-state transition after a
// successful Connect.
func TestStatusAfterConnect(t *testing.T) {
	t.Parallel()

	s := newTestPhoenixServer(t)
	c, err := New(Options{URL: s.URL(), APIKey: "test_key", HeartbeatInterval: 5 * time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if got := c.Status(); got != StatusConnected {
		t.Errorf("Status() after Connect = %s, want %s", got, StatusConnected)
	}
}

// TestStatusDuringReconnect verifies the state machine handles a
// disconnect: StatusConnected → StatusReconnecting → StatusConnected.
func TestStatusDuringReconnect(t *testing.T) {
	t.Parallel()

	s := newTestPhoenixServer(t)
	c, err := New(Options{URL: s.URL(), APIKey: "test_key", HeartbeatInterval: 5 * time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if got := c.Status(); got != StatusConnected {
		t.Fatalf("pre-kill Status() = %s, want %s", got, StatusConnected)
	}

	s.KillConnection()

	// Wait for the transition to StatusReconnecting (the session loop
	// closes the gate and starts backing off — this is event-driven now,
	// so it should land within a few ms).
	if !waitForStatus(c, StatusReconnecting, 500*time.Millisecond) {
		t.Fatalf("Status() did not transition to %s after kill, still %s",
			StatusReconnecting, c.Status())
	}

	// Should reconnect within the 1s initial backoff + dial latency.
	if !waitForStatus(c, StatusConnected, 5*time.Second) {
		t.Fatalf("Status() did not return to %s after reconnect, still %s",
			StatusConnected, c.Status())
	}
}

// TestStatusAfterClose verifies Close transitions the client to
// StatusClosed and the call is idempotent.
func TestStatusAfterClose(t *testing.T) {
	t.Parallel()

	s := newTestPhoenixServer(t)
	c, err := New(Options{URL: s.URL(), APIKey: "test_key", HeartbeatInterval: 5 * time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := c.Status(); got != StatusClosed {
		t.Errorf("Status() after Close = %s, want %s", got, StatusClosed)
	}

	// Idempotent: calling Close again is safe.
	if err := c.Close(); err != nil {
		t.Errorf("second Close returned error: %v", err)
	}
	if got := c.Status(); got != StatusClosed {
		t.Errorf("Status() after second Close = %s, want %s", got, StatusClosed)
	}
}

// waitForStatus polls Status() until it matches `want` or the timeout
// expires. Returns true on match.
func waitForStatus(c *Client, want Status, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.Status() == want {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return c.Status() == want
}
