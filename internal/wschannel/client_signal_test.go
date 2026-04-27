// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestDisconnectSignalFiresOnDropConn verifies that calling dropConn on an
// idle client publishes exactly one wake-up on c.disconnect within ~10ms.
// Replaces the prior 200ms polling loop in disconnectSignal().
func TestDisconnectSignalFiresOnDropConn(t *testing.T) {
	t.Parallel()

	c, err := New(Options{URL: "ws://example.invalid/ws", APIKey: "test_key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	// dropConn on a client with no active conn still fires the signal —
	// the session loop relies on this idempotency.
	c.dropConn(websocket.StatusAbnormalClosure, "test")

	select {
	case <-c.disconnect:
		// good
	case <-time.After(50 * time.Millisecond):
		t.Fatal("disconnect signal did not fire within 50ms of dropConn")
	}
}

// TestDisconnectSignalCoalesces verifies that multiple dropConn calls in a
// tight loop produce only one wake-up. The session loop processes
// reconnects serially, so coalescing is required to avoid extra dial
// attempts.
func TestDisconnectSignalCoalesces(t *testing.T) {
	t.Parallel()

	c, err := New(Options{URL: "ws://example.invalid/ws", APIKey: "test_key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	for range 5 {
		c.dropConn(websocket.StatusAbnormalClosure, "test")
	}

	// Drain one signal.
	select {
	case <-c.disconnect:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("expected one signal in buffer")
	}

	// Buffer should be empty now — no further signals are queued.
	select {
	case <-c.disconnect:
		t.Fatal("multiple dropConn calls leaked signals; coalescing broken")
	case <-time.After(20 * time.Millisecond):
		// good
	}
}

// TestDisconnectSignalDoesNotBlock verifies that dropConn never blocks on
// a full buffer. The session loop may be busy reconnecting when a write
// failure triggers another dropConn; this must not deadlock.
func TestDisconnectSignalDoesNotBlock(t *testing.T) {
	t.Parallel()

	c, err := New(Options{URL: "ws://example.invalid/ws", APIKey: "test_key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	// Fill the buffer by directly sending (simulating a prior dropConn
	// whose signal hasn't been consumed yet).
	c.disconnect <- struct{}{}

	done := make(chan struct{})
	go func() {
		// This call must return promptly via the default branch of the
		// non-blocking send, not park forever.
		c.dropConn(websocket.StatusAbnormalClosure, "second drop")
		close(done)
	}()

	select {
	case <-done:
		// good
	case <-time.After(100 * time.Millisecond):
		t.Fatal("dropConn blocked when disconnect buffer was full")
	}
}
