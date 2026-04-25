// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Push is a server-initiated channel event delivered to the application
// loop (Bubble Tea, etc.) via Pushes().
type Push struct {
	Topic   string          // e.g. "console:lobby" or "console:clock"
	Event   string          // e.g. "stream", "tick", "error"
	Payload json.RawMessage // raw JSON object
}

// Reconnected is a synthetic event the client emits over Pushes() each
// time it has successfully re-dialled the socket and re-joined a topic.
// Applications observe it to replay in-channel state (subscriptions,
// item watches) that the server doesn't remember across the restart.
//
// Carried inside a Push as event "rejoined" with the welcome envelope
// in Payload, on the topic that was rejoined.
const ReconnectedEvent = "rejoined"

// Reconnecting is a synthetic event emitted before every dial attempt
// while the session is down. Payload carries `attempt` (1-indexed) and
// `next_attempt_at` (RFC3339-nano UTC). Applications can use this to
// drive countdown UIs and outage markers. Topic is empty (the event is
// not associated with any channel topic).
const ReconnectingEvent = "reconnecting"

// Client is a Phoenix Channel session multiplexing one or more topics
// over a single WebSocket. It maintains the connection, a heartbeat,
// ref → reply correlation, and a fan-out of server pushes from every
// joined topic.
//
// Lifecycle:
//
//  1. New(opts) returns a configured client (no I/O).
//  2. Connect(ctx) opens the socket and joins the primary topic; returns
//     the join reply.
//  3. JoinTopic(ctx, topic) joins additional topics on the same socket.
//  4. Push(ctx, topic, event, payload) sends a command and blocks for
//     the matching phx_reply; safe to call concurrently from multiple
//     goroutines, across any joined topic.
//  5. Pushes() yields server-initiated events from every joined topic;
//     consumers route on Push.Topic.
//  6. Close() shuts everything down cleanly.
//
// Reconnect-with-backoff is intentionally NOT in V1; the caller treats a
// closed Pushes() channel as an explicit signal to retry.
type Client struct {
	url               string
	apiKey            string
	primaryTopic      string
	heartbeatInterval time.Duration
	log               *slog.Logger

	// connMu guards the active *websocket.Conn pointer. The session
	// loop swaps it on each (re)connect; readers/writers take a fresh
	// snapshot for each I/O call.
	connMu sync.RWMutex
	conn   *websocket.Conn

	// Two-stage readiness gate:
	//
	//   socketReady — closed once the active WebSocket can carry frames.
	//                 Rejoin uses this to send phx_join during reconnect
	//                 before the application gate opens.
	//   sessionReady — closed once the socket is live AND every previously
	//                 active topic has been re-joined. Application calls
	//                 (Push, JoinTopic) wait on this so they never race
	//                 against the rejoin replay.
	//
	// Both are recreated on disconnect.
	gatesMu      sync.Mutex
	socketReady  chan struct{}
	sessionReady chan struct{}

	// topics maps each joined topic to its join_ref. New joins allocate
	// a fresh ref; we use it as both the topic's join_ref and the
	// per-message join_ref for outbound frames on that topic. The set
	// also drives automatic re-join on reconnect.
	topicsMu sync.RWMutex
	topics   map[string]string

	refMu   sync.Mutex
	nextRef int64
	pending map[string]chan Frame

	pushes chan Push

	closeOnce sync.Once
	done      chan struct{}
	wg        sync.WaitGroup

	// runCtx scopes goroutine-internal I/O to the client's lifetime;
	// cancelled on Close so reads/writes/heartbeats abort cleanly.
	runCtx    context.Context
	runCancel context.CancelFunc

	// Buffered channel for outbound writes; the writer goroutine is the
	// single owner of the WS write side, avoiding the need for a write
	// mutex on the connection.
	out chan Frame
}

// Options configures a new Client.
type Options struct {
	// URL is the base WS endpoint, e.g. "wss://www.truestamp.com/console/websocket".
	// vsn=2.0.0 and api_key are appended automatically.
	URL string

	// APIKey is the Truestamp API key. Sent as a query parameter to the
	// Socket.connect/3 callback on the server.
	APIKey string

	// Topic is the primary channel topic Connect joins. Defaults to
	// "console:lobby" when empty. Additional topics can be joined later
	// via JoinTopic.
	Topic string

	// HeartbeatInterval — Phoenix's default is 30s; set lower for tests.
	// Zero or negative falls back to 30s.
	HeartbeatInterval time.Duration

	// PushBufferSize sets the capacity of the Pushes channel. Defaults to 256.
	PushBufferSize int

	// Logger receives transport diagnostics (read EOFs, dial failures,
	// frame decode errors, push channel overflow). When nil, logs are
	// discarded. The TUI is the typical caller and should pass a
	// file-backed logger from internal/logging — these messages are
	// noisy by design and would clutter the UI.
	Logger *slog.Logger
}

// New constructs a Client without performing any network I/O.
func New(opts Options) (*Client, error) {
	if opts.URL == "" {
		return nil, errors.New("wschannel: URL is required")
	}
	if opts.APIKey == "" {
		return nil, errors.New("wschannel: APIKey is required")
	}
	topic := opts.Topic
	if topic == "" {
		topic = "console:lobby"
	}
	pushBuf := opts.PushBufferSize
	if pushBuf <= 0 {
		pushBuf = 256
	}
	hb := opts.HeartbeatInterval
	if hb <= 0 {
		hb = 30 * time.Second
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	runCtx, runCancel := context.WithCancel(context.Background())
	c := &Client{
		url:               opts.URL,
		apiKey:            opts.APIKey,
		primaryTopic:      topic,
		heartbeatInterval: hb,
		log:               logger,
		topics:            make(map[string]string),
		pending:           make(map[string]chan Frame),
		pushes:            make(chan Push, pushBuf),
		done:              make(chan struct{}),
		runCtx:            runCtx,
		runCancel:         runCancel,
		out:               make(chan Frame, 64),
		socketReady:       make(chan struct{}),
		sessionReady:      make(chan struct{}),
	}
	return c, nil
}

// reconnect backoff schedule (capped). Modest jitter is fine for a
// human-driven TUI; we don't have thundering-herd concerns at this scale.
var reconnectBackoff = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
}

// Connect opens the WebSocket, joins the primary channel topic, and
// installs a session loop that auto-reconnects on transport failure.
// Returns the initial join reply for the primary topic.
//
// On reconnect the client re-dials, replays every previously-joined
// topic, and emits a synthetic `rejoined` push (event ReconnectedEvent)
// per topic so the application can resync any in-channel state the
// server doesn't remember (subscriptions, item watches).
func (c *Client) Connect(ctx context.Context) (json.RawMessage, error) {
	if err := c.dial(ctx); err != nil {
		return nil, err
	}

	c.wg.Add(2)
	go c.readLoop()
	go c.writeLoop()

	c.wg.Add(1)
	go c.heartbeatLoop(c.heartbeatInterval)

	c.wg.Add(1)
	go c.sessionLoop()

	c.openSocketGate()
	welcome, err := c.joinTopicOnSocket(ctx, c.primaryTopic)
	if err != nil {
		return nil, err
	}
	c.openSessionGate()
	return welcome, nil
}

// dial opens a single WebSocket and installs it as the active conn.
// The session loop is responsible for retrying until the client is closed.
func (c *Client) dial(ctx context.Context) error {
	u, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	q := u.Query()
	q.Set("vsn", "2.0.0")
	q.Set("api_key", c.apiKey)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		// The library's error embeds the upgrade URL — including our
		// api_key — verbatim. Redact eagerly so the cleartext token
		// can't reach an upstream caller's UI or logger.
		return fmt.Errorf("dial: %s", redactSecrets(err.Error()))
	}
	conn.SetReadLimit(1 << 20)

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()
	return nil
}

// activeConn returns the live socket or nil if we're between sockets.
func (c *Client) activeConn() *websocket.Conn {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn
}

// openSocketGate releases callers using sendOnSocket (rejoin path). Called
// after a successful dial.
func (c *Client) openSocketGate() { closeOnce(&c.gatesMu, &c.socketReady) }

// openSessionGate releases the application's Push/JoinTopic callers.
// Called only after rejoinAllTopics has finished, so apps never see an
// empty topics map mid-replay.
func (c *Client) openSessionGate() { closeOnce(&c.gatesMu, &c.sessionReady) }

// resetGates installs fresh (un-closed) gates if the current ones are
// already open. Idempotent — calling twice in a row is a no-op so it's
// safe to invoke from both dropConn and the session loop.
func (c *Client) resetGates() {
	c.gatesMu.Lock()
	defer c.gatesMu.Unlock()
	if isClosed(c.socketReady) {
		c.socketReady = make(chan struct{})
	}
	if isClosed(c.sessionReady) {
		c.sessionReady = make(chan struct{})
	}
}

func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func (c *Client) socketGate() <-chan struct{} {
	c.gatesMu.Lock()
	defer c.gatesMu.Unlock()
	return c.socketReady
}

func (c *Client) sessionGate() <-chan struct{} {
	c.gatesMu.Lock()
	defer c.gatesMu.Unlock()
	return c.sessionReady
}

func closeOnce(mu *sync.Mutex, ch *chan struct{}) {
	mu.Lock()
	defer mu.Unlock()
	select {
	case <-*ch:
		// already open
	default:
		close(*ch)
	}
}

// sessionLoop watches for socket loss (signalled by the read loop
// closing the active conn) and re-dials with exponential backoff. On
// each successful redial it spins up new read/write goroutines and
// re-joins every topic that was active when the socket dropped.
func (c *Client) sessionLoop() {
	defer c.wg.Done()

	for {
		// Wait for either the read loop to drop the active conn, or
		// the client to be closed.
		select {
		case <-c.done:
			return
		case <-c.runCtx.Done():
			return
		case <-c.disconnectSignal():
			// Disconnected — fall through to reconnect.
		}

		c.resetGates()

		attempt := 0
		for {
			if c.isClosed() {
				return
			}
			delay := reconnectBackoff[min(attempt, len(reconnectBackoff)-1)]
			attempt++
			c.emitReconnecting(attempt, time.Now().Add(delay))

			select {
			case <-c.done:
				return
			case <-time.After(delay):
			}

			dialCtx, cancel := context.WithTimeout(c.runCtx, 15*time.Second)
			err := c.dial(dialCtx)
			cancel()
			if err != nil {
				c.logErr(slog.LevelInfo, "reconnect dial failed", err, "attempt", attempt)
				continue
			}

			// Persistent read/write loops are still running; they
			// were parked on the socket gate or in c.out's blocking
			// receive. Opening the socket gate releases them onto
			// the new conn.
			c.openSocketGate()

			if rerr := c.rejoinAllTopics(); rerr != nil {
				c.logErr(slog.LevelWarn, "reconnect rejoin failed", rerr)
				c.dropConn(websocket.StatusAbnormalClosure, "rejoin failed")
				c.resetGates()
				continue
			}
			c.openSessionGate()
			break
		}
	}
}

// disconnectSignal returns a channel that fires when the read loop has
// observed the socket dying. Implemented by polling the active-conn
// pointer; cheap because the read loop nils it on exit.
func (c *Client) disconnectSignal() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-c.done:
				close(ch)
				return
			case <-t.C:
				if c.activeConn() == nil {
					close(ch)
					return
				}
			}
		}
	}()
	return ch
}

func (c *Client) isClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

// rejoinAllTopics re-issues phx_join for every topic that was active
// before the disconnect, in the order they were originally joined. The
// primary topic goes first so applications observing the rejoin event
// see them in a predictable order.
func (c *Client) rejoinAllTopics() error {
	c.topicsMu.RLock()
	prev := make([]string, 0, len(c.topics))
	if _, ok := c.topics[c.primaryTopic]; ok {
		prev = append(prev, c.primaryTopic)
	}
	for t := range c.topics {
		if t != c.primaryTopic {
			prev = append(prev, t)
		}
	}
	c.topicsMu.RUnlock()

	c.topicsMu.Lock()
	c.topics = make(map[string]string)
	c.topicsMu.Unlock()

	for _, t := range prev {
		ctx, cancel := context.WithTimeout(c.runCtx, 10*time.Second)
		welcome, err := c.joinTopicOnSocket(ctx, t)
		cancel()
		if err != nil {
			return fmt.Errorf("rejoin %s: %w", t, err)
		}
		select {
		case c.pushes <- Push{Topic: t, Event: ReconnectedEvent, Payload: welcome}:
		default:
			c.log.Warn("dropped rejoin notice", "topic", t)
		}
	}
	return nil
}

// dropConn closes the active websocket, clears the pointer (so the
// session loop's disconnect signal fires), and resets the readiness
// gates so persistent read/write loops park until reconnect succeeds.
// Safe to call multiple times — only the first call has an effect.
func (c *Client) dropConn(code websocket.StatusCode, reason string) {
	c.connMu.Lock()
	conn := c.conn
	c.conn = nil
	c.connMu.Unlock()
	if conn != nil {
		_ = conn.Close(code, reason)
	}
	c.resetGates()
}

// JoinTopic joins an additional channel topic on the existing socket.
// Returns the raw join reply payload. Safe to call after Connect.
func (c *Client) JoinTopic(ctx context.Context, topic string) (json.RawMessage, error) {
	return c.joinTopic(ctx, topic, c.send)
}

// joinTopicOnSocket is the internal variant used during the initial
// primary-topic join and during rejoin replays. It bypasses the session
// gate (which can't be open yet) but still respects the socket gate.
func (c *Client) joinTopicOnSocket(ctx context.Context, topic string) (json.RawMessage, error) {
	return c.joinTopic(ctx, topic, c.sendOnSocket)
}

func (c *Client) joinTopic(ctx context.Context, topic string, sendFn func(context.Context, Frame) error) (json.RawMessage, error) {
	if topic == "" {
		return nil, errors.New("wschannel: topic is required")
	}

	joinRef := c.allocRef()
	joinReplyCh := make(chan Frame, 1)
	c.refMu.Lock()
	c.pending[joinRef] = joinReplyCh
	c.refMu.Unlock()

	c.topicsMu.Lock()
	c.topics[topic] = joinRef
	c.topicsMu.Unlock()

	jr := joinRef
	frame, _ := newFrame(&jr, &jr, topic, "phx_join", map[string]any{})
	if err := sendFn(ctx, frame); err != nil {
		c.forgetTopic(topic)
		return nil, fmt.Errorf("send phx_join: %w", err)
	}

	select {
	case rf := <-joinReplyCh:
		reply, err := ParseReply(rf)
		if err != nil {
			c.forgetTopic(topic)
			return nil, err
		}
		if reply.Status != "ok" {
			c.forgetTopic(topic)
			return nil, fmt.Errorf("join %s rejected: %s", topic, string(reply.Response))
		}
		return reply.Response, nil

	case <-ctx.Done():
		c.forgetTopic(topic)
		return nil, ctx.Err()

	case <-c.done:
		return nil, errors.New("connection closed before join reply")
	}
}

// Push sends an event on the given (already-joined) topic and blocks
// until the matching phx_reply arrives, or ctx is cancelled.
func (c *Client) Push(ctx context.Context, topic, event string, payload any) (PhxReply, error) {
	c.topicsMu.RLock()
	joinRef, ok := c.topics[topic]
	c.topicsMu.RUnlock()
	if !ok {
		return PhxReply{}, fmt.Errorf("topic %q not joined", topic)
	}

	ref := c.allocRef()
	replyCh := make(chan Frame, 1)
	c.refMu.Lock()
	c.pending[ref] = replyCh
	c.refMu.Unlock()
	defer func() {
		c.refMu.Lock()
		delete(c.pending, ref)
		c.refMu.Unlock()
	}()

	jr, r := joinRef, ref
	frame, err := newFrame(&jr, &r, topic, event, payload)
	if err != nil {
		return PhxReply{}, err
	}
	if err := c.send(ctx, frame); err != nil {
		return PhxReply{}, err
	}

	select {
	case rf := <-replyCh:
		return ParseReply(rf)
	case <-ctx.Done():
		return PhxReply{}, ctx.Err()
	case <-c.done:
		return PhxReply{}, errors.New("connection closed")
	}
}

// Pushes returns the channel of server-initiated events from every
// joined topic. The channel is closed when the connection ends (clean
// Close() or a transport error).
func (c *Client) Pushes() <-chan Push { return c.pushes }

// Close shuts down the client and underlying socket cleanly. Safe to
// call multiple times.
func (c *Client) Close() error {
	c.shutdown(websocket.StatusNormalClosure, "client close")
	c.wg.Wait()
	return nil
}

// ===========================================================================
// Internal
// ===========================================================================

// send is the application-facing send path. It waits for the session
// gate (socket live AND topics replayed) so callers never see a
// transient reconnect window.
func (c *Client) send(ctx context.Context, f Frame) error {
	if err := waitGate(ctx, c.done, c.sessionGate()); err != nil {
		return err
	}
	return c.queueFrame(ctx, f)
}

// sendOnSocket is the rejoin path: it waits only for the socket gate,
// not the session gate, so it can fire phx_join during the replay.
func (c *Client) sendOnSocket(ctx context.Context, f Frame) error {
	if err := waitGate(ctx, c.done, c.socketGate()); err != nil {
		return err
	}
	return c.queueFrame(ctx, f)
}

func (c *Client) queueFrame(ctx context.Context, f Frame) error {
	select {
	case c.out <- f:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return errors.New("connection closed")
	}
}

func waitGate(ctx context.Context, done <-chan struct{}, gate <-chan struct{}) error {
	select {
	case <-gate:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return errors.New("connection closed")
	}
}

func (c *Client) forgetTopic(topic string) {
	c.topicsMu.Lock()
	delete(c.topics, topic)
	c.topicsMu.Unlock()
}

// writeLoop is persistent across the Client's lifetime. Each frame
// pulls the current active conn at write time; if no conn is live, the
// write blocks on the socket gate until reconnect installs one. A write
// failure drops the conn and goes back to waiting.
func (c *Client) writeLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.done:
			return
		case f := <-c.out:
			if err := c.writeFrame(f); err != nil {
				c.logErr(slog.LevelWarn, "ws write failed", err, "topic", f.Topic, "event", f.Event)
				c.dropConn(websocket.StatusInternalError, "write error")
				// Continue: the next frame will block on the socket
				// gate until reconnect, or be aborted on c.done.
			}
		}
	}
}

func (c *Client) writeFrame(f Frame) error {
	// Wait for an active conn, with reasonable timeout — a hard-down
	// server shouldn't queue indefinitely on out.
	ctx, cancel := context.WithTimeout(c.runCtx, 30*time.Second)
	defer cancel()
	if err := waitGate(ctx, c.done, c.socketGate()); err != nil {
		return fmt.Errorf("wait socket: %w", err)
	}

	conn := c.activeConn()
	if conn == nil {
		return errors.New("no active conn")
	}

	data, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("encode frame: %w", err)
	}

	wctx, wcancel := context.WithTimeout(c.runCtx, 10*time.Second)
	defer wcancel()
	if err := conn.Write(wctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("ws write: %w", err)
	}
	return nil
}

// readLoop is persistent across the Client's lifetime. On read error
// it drops the conn (signalling the session loop to reconnect), drains
// any pending replies, then waits on the socket gate for a fresh conn.
func (c *Client) readLoop() {
	defer c.wg.Done()

	for {
		if c.isClosed() {
			return
		}

		conn := c.activeConn()
		if conn == nil {
			// No live conn — wait for the session loop to dial a new
			// one (which then opens the socket gate).
			select {
			case <-c.socketGate():
				continue
			case <-c.done:
				return
			}
		}

		_, data, err := conn.Read(c.runCtx)
		if err != nil {
			if c.isClosed() {
				return
			}
			c.logErr(slog.LevelInfo, "ws read failed", err)
			c.dropConn(websocket.StatusAbnormalClosure, "read error")
			c.drainPending()
			continue
		}

		var f Frame
		if err := json.Unmarshal(data, &f); err != nil {
			c.logErr(slog.LevelWarn, "frame decode failed", err)
			continue
		}

		// Reply correlation: phx_reply frames carry the ref of the
		// original request.
		if f.Event == "phx_reply" && f.Ref != nil {
			c.refMu.Lock()
			ch, ok := c.pending[*f.Ref]
			if ok {
				delete(c.pending, *f.Ref)
			}
			c.refMu.Unlock()
			if ok {
				ch <- f
				continue
			}
		}

		// Ignore phx_close / phx_error here for V1 — the read error
		// path will handle disconnects.
		if f.Event == "phx_close" || f.Event == "phx_error" {
			continue
		}

		// Server pushes for the application to handle.
		select {
		case c.pushes <- Push{Topic: f.Topic, Event: f.Event, Payload: f.Payload}:
		default:
			// Drop if the consumer is slow — better than blocking the
			// reader (and thus the heartbeat).
			c.log.Warn("dropped push (consumer slow)", "topic", f.Topic, "event", f.Event)
		}
	}
}

func (c *Client) heartbeatLoop(interval time.Duration) {
	defer c.wg.Done()

	if interval <= 0 {
		interval = 30 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-t.C:
			ref := c.allocRef()
			r := ref
			frame, _ := newFrame(nil, &r, "phoenix", "heartbeat", map[string]any{})
			ctx, cancel := context.WithTimeout(c.runCtx, 5*time.Second)
			_ = c.send(ctx, frame)
			cancel()
		}
	}
}

func (c *Client) allocRef() string {
	c.refMu.Lock()
	c.nextRef++
	n := c.nextRef
	c.refMu.Unlock()
	return strconv.FormatInt(n, 10)
}

// apiKeyRe matches the `api_key=…` query-string fragment that
// websocket.Dial errors echo verbatim from the upgrade URL. Without
// redaction the user's key would land in the log and in any wrapped
// error a caller persists.
var apiKeyRe = regexp.MustCompile(`api_key=[^&"\s]*`)

func redactSecrets(s string) string {
	return apiKeyRe.ReplaceAllString(s, "api_key=REDACTED")
}

// logErr is the canonical "something went wrong but it's not fatal"
// path inside the client. All transport noise flows through here so
// it never leaks into the UI; redaction guarantees the api_key never
// hits disk in cleartext.
func (c *Client) logErr(level slog.Level, msg string, err error, attrs ...any) {
	out := []any{"err", redactSecrets(err.Error())}
	out = append(out, attrs...)
	c.log.Log(c.runCtx, level, msg, out...)
}

// emitReconnecting injects a synthetic Push describing an upcoming
// reconnect attempt. Drop-on-full is fine — the next attempt will emit
// a fresh status, and the consumer only needs the latest value.
func (c *Client) emitReconnecting(attempt int, nextAttemptAt time.Time) {
	payload, _ := json.Marshal(map[string]any{
		"attempt":         attempt,
		"next_attempt_at": nextAttemptAt.UTC().Format(time.RFC3339Nano),
	})
	select {
	case c.pushes <- Push{Event: ReconnectingEvent, Payload: payload}:
	default:
	}
}

// drainPending fails every in-flight reply correlation with a synthetic
// "phx_error" reply so blocked Push callers wake up promptly when the
// underlying connection drops. The new server has no memory of the old
// refs, so leaving them parked would hang the call until ctx timeout.
func (c *Client) drainPending() {
	c.refMu.Lock()
	pending := c.pending
	c.pending = make(map[string]chan Frame)
	c.refMu.Unlock()

	for ref, ch := range pending {
		errPayload, _ := json.Marshal(map[string]any{
			"status":   "error",
			"response": map[string]any{"reason": "connection lost during reconnect"},
		})
		r := ref
		select {
		case ch <- Frame{Event: "phx_reply", Ref: &r, Payload: errPayload}:
		default:
			// Channel buffer (1) full — caller must be gone.
		}
	}
}

func (c *Client) shutdown(code websocket.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		close(c.done)
		c.runCancel()
		c.dropConn(code, reason)
		// Pushes() callers see the channel closed, signalling
		// permanent end-of-stream (distinct from a transient
		// disconnect, which the session loop handles transparently).
		close(c.pushes)
	})
}
