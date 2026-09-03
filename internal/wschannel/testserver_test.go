// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// testPhoenixServer is a controllable Phoenix-Channel-V2 server used by
// the wschannel unit tests and chaos tests. It accepts WebSocket upgrades,
// parses V2 frames, and replies to phx_join and heartbeats by default. Tests
// can override behavior (hang on phx_join, kill the connection, broadcast
// pushes) and assert against per-event counters.
//
// One server instance is reused across reconnect cycles in the same test;
// the per-connection state is reset on each Accept.
type testPhoenixServer struct {
	t   *testing.T
	srv *httptest.Server

	mu       sync.Mutex
	conn     *websocket.Conn // current upgraded conn, or nil between connections
	behavior behaviorMode    // controls how the next/current conn responds

	// Atomic counters, race-safe reads from test goroutines.
	Connects       atomic.Int64
	PhxJoins       atomic.Int64
	Heartbeats     atomic.Int64
	FramesReceived atomic.Int64

	// writeMu serializes Write calls on the active conn, coder/websocket
	// is not safe for concurrent writers.
	writeMu sync.Mutex

	// connClosed fires every time the per-connection goroutine returns.
	// Tests use this to wait for KillConnection to take effect before
	// triggering the next phase.
	connClosed chan struct{}
}

type behaviorMode int

const (
	// behaviorNormal, default: reply {"status":"ok","response":{}} to
	// every phx_join and ack heartbeats. All other events are read and
	// counted but produce no reply (so tests can drive synthetic replies
	// via SendReply, or verify rate-limiting / drainPending behavior).
	behaviorNormal behaviorMode = iota

	// behaviorHangOnPhxJoin, accept frames, count them, but never reply
	// to phx_join. Used to simulate a server that's reachable but stuck.
	behaviorHangOnPhxJoin
)

func newTestPhoenixServer(t *testing.T) *testPhoenixServer {
	t.Helper()
	s := &testPhoenixServer{
		t:          t,
		behavior:   behaviorNormal,
		connClosed: make(chan struct{}, 16),
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Close)
	return s
}

// URL returns a ws:// URL suitable for passing to wschannel.New.
func (s *testPhoenixServer) URL() string {
	u, _ := url.Parse(s.srv.URL)
	u.Scheme = "ws"
	u.Path = "/console/websocket"
	return u.String()
}

// SetBehavior changes how subsequent connections (and the current one, for
// the next frame) respond. Safe to call from any goroutine.
func (s *testPhoenixServer) SetBehavior(b behaviorMode) {
	s.mu.Lock()
	s.behavior = b
	s.mu.Unlock()
}

// AcceptNext is sugar for SetBehavior(behaviorNormal).
func (s *testPhoenixServer) AcceptNext() { s.SetBehavior(behaviorNormal) }

// AcceptButHangOnPhxJoin is sugar for SetBehavior(behaviorHangOnPhxJoin).
func (s *testPhoenixServer) AcceptButHangOnPhxJoin() { s.SetBehavior(behaviorHangOnPhxJoin) }

// KillConnection closes the active connection abnormally, forcing the
// client into reconnect. Returns once the server-side handler goroutine
// has observed the close (so tests can sequence reliably).
func (s *testPhoenixServer) KillConnection() {
	s.mu.Lock()
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()
	if conn != nil {
		_ = conn.Close(websocket.StatusAbnormalClosure, "test kill")
	}
	// Wait briefly for the read goroutine to exit; tests use this to
	// ensure they don't race ahead of the close.
	select {
	case <-s.connClosed:
	case <-time.After(2 * time.Second):
	}
}

// BroadcastFrame sends a server-initiated push to the active connection.
// Used for ticks / synthetic stream events. Returns an error if no conn
// is live or the write fails.
func (s *testPhoenixServer) BroadcastFrame(topic, event string, payload any) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return errNoConn
	}
	frame, err := newFrame(nil, nil, topic, event, payload)
	if err != nil {
		return err
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, data)
}

var errNoConn = errNoConnT("test server has no active connection")

type errNoConnT string

func (e errNoConnT) Error() string { return string(e) }

// SendReply sends a synthetic phx_reply for the given joinRef/ref pair.
// Used by chaos cases that need to control reply timing.
func (s *testPhoenixServer) SendReply(joinRef, ref, topic string, status string, response any) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return websocket.CloseError{Code: websocket.StatusAbnormalClosure}
	}
	payload := map[string]any{"status": status, "response": response}
	jr, r := joinRef, ref
	frame, err := newFrame(&jr, &r, topic, "phx_reply", payload)
	if err != nil {
		return err
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, data)
}

// Close shuts the test server down. Called automatically via t.Cleanup.
func (s *testPhoenixServer) Close() {
	s.mu.Lock()
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()
	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "test cleanup")
	}
	s.srv.Close()
}

// handle is the http.HandlerFunc bound to httptest.Server. Each upgraded
// connection runs in its own goroutine; the read loop parses V2 frames,
// updates counters, and dispatches to the current behavior handler.
func (s *testPhoenixServer) handle(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Allow any origin (httptest.Server uses an arbitrary port).
		InsecureSkipVerify: true,
	})
	if err != nil {
		s.t.Logf("testserver: accept failed: %v", err)
		return
	}

	// Install the conn BEFORE incrementing Connects so any test that
	// waits on Connects (awaitConnects, waitForBoth) sees a fully
	// initialized server state when it returns.
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	s.Connects.Add(1)

	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "")
		s.mu.Lock()
		if s.conn == conn {
			s.conn = nil
		}
		s.mu.Unlock()
		// Non-blocking notify: tests waiting on KillConnection wake up.
		select {
		case s.connClosed <- struct{}{}:
		default:
		}
	}()

	ctx := r.Context()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		s.FramesReceived.Add(1)

		var f Frame
		if err := json.Unmarshal(data, &f); err != nil {
			continue
		}

		switch f.Event {
		case "phx_join":
			s.PhxJoins.Add(1)
			s.mu.Lock()
			behavior := s.behavior
			s.mu.Unlock()
			if behavior == behaviorHangOnPhxJoin {
				continue
			}
			// Reply with status=ok and an empty response object.
			ref := ""
			if f.Ref != nil {
				ref = *f.Ref
			}
			joinRef := ""
			if f.JoinRef != nil {
				joinRef = *f.JoinRef
			}
			_ = s.SendReply(joinRef, ref, f.Topic, "ok", map[string]any{})

		case "heartbeat":
			s.Heartbeats.Add(1)
			ref := ""
			if f.Ref != nil {
				ref = *f.Ref
			}
			_ = s.SendReply("", ref, "phoenix", "ok", map[string]any{})

		default:
			// Unhandled events: counted via FramesReceived but not auto-replied.
			// Tests can call SendReply explicitly to simulate a server response.
		}
	}
}

// awaitConnect blocks until the test server has accepted at least n
// connections, or times out.
func (s *testPhoenixServer) awaitConnects(n int64, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s.Connects.Load() >= n {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// =============================================================================
// Self-tests for the harness, proving it works before A/B/C depend on it.
// =============================================================================

// TestHarnessAcceptAndJoin verifies the server accepts a connection,
// receives a phx_join, and replies with status=ok.
func TestHarnessAcceptAndJoin(t *testing.T) {
	t.Parallel()

	s := newTestPhoenixServer(t)

	c, err := New(Options{URL: s.URL(), APIKey: "test_key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	welcome, err := c.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !strings.Contains(string(welcome), "{}") {
		t.Errorf("welcome should be empty object, got: %s", welcome)
	}
	if got := s.Connects.Load(); got != 1 {
		t.Errorf("Connects = %d, want 1", got)
	}
	if got := s.PhxJoins.Load(); got != 1 {
		t.Errorf("PhxJoins = %d, want 1", got)
	}
}

// TestHarnessKillAndReconnect verifies KillConnection forces the client
// into reconnect, and the server accepts a fresh connection.
func TestHarnessKillAndReconnect(t *testing.T) {
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

	// Kill the server-side connection; client should reconnect.
	s.KillConnection()

	// Wait for the second Connects increment.
	if !s.awaitConnects(2, 5*time.Second) {
		t.Fatalf("server did not see reconnect within 5s, Connects=%d", s.Connects.Load())
	}
	// Wait for the second phx_join.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && s.PhxJoins.Load() < 2 {
		time.Sleep(10 * time.Millisecond)
	}
	if got := s.PhxJoins.Load(); got < 2 {
		t.Errorf("PhxJoins = %d after reconnect, want >= 2", got)
	}
}

// TestHarnessBroadcastFrame verifies the server can push a frame to the
// connected client.
func TestHarnessBroadcastFrame(t *testing.T) {
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

	if err := s.BroadcastFrame("console:lobby", "stream", map[string]any{"hello": "world"}); err != nil {
		t.Fatalf("BroadcastFrame: %v", err)
	}

	select {
	case p := <-c.Pushes():
		if p.Event != "stream" {
			t.Errorf("event = %q, want stream", p.Event)
		}
		if !strings.Contains(string(p.Payload), "hello") {
			t.Errorf("payload missing hello: %s", p.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive broadcast frame within 2s")
	}
}
