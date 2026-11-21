// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package console

import (
	"context"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/truestamp/truestamp-cli/internal/wschannel"
)

// lobbyCatalog is the stream catalog the server's welcome envelope
// carries for a signed-in scope with a team, in the server's order: the
// base streams of TruestampWeb.ConsoleStreams (@base_streams, in
// lib/truestamp_web/channels/console_streams.ex of truestamp-v2) and then
// items.team (catalog_for/1). Their scopes are the atoms the server
// serializes as strings.
var lobbyCatalog = []StreamMeta{
	{ID: "blocks", Scope: "global", Title: "Block lifecycle (created, committed)"},
	{ID: "blocks.healing", Scope: "global", Title: "Block healing transitions (forward / reverse)"},
	{ID: "beacons", Scope: "global", Title: "Beacons (public projection of new blocks)"},
	{ID: "commitments.subjects", Scope: "global_and_team", Title: "Commitments into blocks: every entropy observation's and the current team's items'"},
	{ID: "commitments.blocks", Scope: "global", Title: "Commitments of blocks into epochs on public chains (Stellar, Bitcoin)"},
	{ID: "entropy.nist", Scope: "global", Title: "NIST beacon entropy observations"},
	{ID: "entropy.stellar", Scope: "global", Title: "Stellar entropy observations"},
	{ID: "entropy.bitcoin", Scope: "global", Title: "Bitcoin entropy observations"},
	{ID: "items.team", Scope: "team", Title: "Items in current team"},
}

// lobbyServer is a loopback stand-in for the server's console:lobby
// channel (TruestampWeb.ConsoleChannel, in
// lib/truestamp_web/channels/console_channel.ex of truestamp-v2) over
// Phoenix Channels V2 frames. It answers phx_join with the welcome
// envelope (welcome_envelope/1), a subscribe with the reply of
// dispatch("subscribe", ...) for every stream asked for, and a
// heartbeat, and records the streams each subscribe named.
type lobbyServer struct {
	srv *httptest.Server

	mu         sync.Mutex
	conn       *websocket.Conn
	subscribes [][]string
}

func startLobbyServer(t *testing.T) *lobbyServer {
	t.Helper()
	s := &lobbyServer{}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *lobbyServer) url() string {
	return "ws" + strings.TrimPrefix(s.srv.URL, "http") + "/console/websocket"
}

func (s *lobbyServer) handle(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}
		var f wschannel.Frame
		if json.Unmarshal(data, &f) != nil {
			continue
		}
		switch f.Event {
		case "phx_join":
			s.reply(f, map[string]any{
				"scope":   map[string]any{"user_id": "019a3f2c-4e00-7a00-8b00-0000000000a1", "team_id": "019a3f2c-4e00-7a00-8b00-0000000000b2", "plan": "free"},
				"streams": lobbyCatalog,
				"server":  map[string]any{"version": "0.1.0"},
			})
		case "subscribe":
			var p struct {
				Streams []string `json:"streams"`
			}
			_ = json.Unmarshal(f.Payload, &p)
			s.mu.Lock()
			s.subscribes = append(s.subscribes, p.Streams)
			s.mu.Unlock()
			s.reply(f, map[string]any{"subscribed": p.Streams, "rejected": []string{}, "active": p.Streams})
		case "heartbeat":
			s.reply(f, map[string]any{})
		}
	}
}

// reply answers frame f with status ok and response.
func (s *lobbyServer) reply(f wschannel.Frame, response any) {
	payload, _ := json.Marshal(map[string]any{"status": "ok", "response": response}, json.Deterministic(true))
	s.write(wschannel.Frame{JoinRef: f.JoinRef, Ref: f.Ref, Topic: f.Topic, Event: "phx_reply", Payload: payload})
}

// pushAtLayout is the server's "at" format: RFC 3339 in UTC with
// microseconds.
const pushAtLayout = "2006-01-02T15:04:05.000000Z"

// push sends a stream push as the channel's push_event/4 does: on the
// lobby topic, event "stream", {stream, kind, at, data}, with at the
// given time in the server's format.
func (s *lobbyServer) push(t *testing.T, stream, kind string, at time.Time, data map[string]any) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"stream": stream, "kind": kind, "at": at.UTC().Format(pushAtLayout), "data": data}, json.Deterministic(true))
	if err != nil {
		t.Fatal(err)
	}
	s.write(wschannel.Frame{Topic: lobbyTopic, Event: "stream", Payload: payload})
}

func (s *lobbyServer) write(f wschannel.Frame) {
	data, _ := json.Marshal(f, json.Deterministic(true))
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.conn.Write(ctx, websocket.MessageText, data)
}

func (s *lobbyServer) subscribed() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.subscribes)
}

// commitmentRow is a commitment row as the server's console projection
// sends it (TruestampWeb.ConsoleProjector.project/1, in
// lib/truestamp_web/channels/console_projector.ex of truestamp-v2): one
// shape for the three kinds, the links a kind does not have null.
func commitmentRow(id, subjectKind string) map[string]any {
	row := map[string]any{
		"id": id, "subject_kind": subjectKind, "tree_kind": "block", "chain": nil,
		"item_id": nil, "entropy_observation_id": nil, "subject_block_id": nil,
		"tree_block_id": nil, "epoch_id": nil, "inserted_at": "2026-10-27T16:00:00.000000Z",
	}
	switch subjectKind {
	case "item":
		row["item_id"], row["tree_block_id"] = "019a3f2c-6a00-7c11-8e22-0f1e2d3c4b5a", "019a3f2c-5f00-7a00-8b00-000000000001"
	case "observation":
		row["entropy_observation_id"], row["tree_block_id"] = "019a3f2c-6900-7b22-9c33-1a2b3c4d5e6f", "019a3f2c-5f00-7a00-8b00-000000000001"
	case "block":
		row["tree_kind"], row["chain"] = "epoch", "bitcoin"
		row["subject_block_id"], row["epoch_id"] = "019a3f2c-5f00-7a00-8b00-000000000001", "019a3f2c-7000-7d44-8a55-6b7c8d9e0f12"
	}
	return row
}

// The Monitor subscribes to every stream of the server's catalog, the two
// commitment streams among them, and renders the one commitment event
// both emit, commitment.created, as a row naming its subject_kind: an
// item and an observation on commitments.subjects, a block on
// commitments.blocks (TruestampWeb.ConsoleStreams.classify/3). Each
// row is stamped one minute before the current time, inside the
// Monitor's retention, and a first event stamped an hour beyond the
// retention ages out. The server is a loopback stand-in speaking
// Phoenix V2 frames. Mutants killed: subscribeAllStreams leaving any
// one catalog stream out (here commitments.subjects or
// commitments.blocks); compactKind without its commitment branch (all
// three rows "commitment"); evictAged with a cutoff that drops every
// event (no rows) or keeps every event (the aged event rendered as a
// fourth row).
func TestMonitor_FollowsTheServersCommitmentStreams(t *testing.T) {
	srv := startLobbyServer(t)
	client, err := wschannel.New(wschannel.Options{URL: srv.url(), APIKey: "test-key", HeartbeatInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	var w Welcome
	if err := json.Unmarshal(raw, &w); err != nil {
		t.Fatalf("welcome: %v", err)
	}

	m := newMonitorModel(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	m.applyWelcome(w)
	cmd := m.subscribeAllStreams()
	if cmd == nil {
		t.Fatal("no subscribe was sent for the catalog")
	}
	m.Update(cmd())

	want := make([]string, 0, len(lobbyCatalog))
	for _, s := range lobbyCatalog {
		want = append(want, s.ID)
	}
	if got := srv.subscribed(); len(got) != 1 || !slices.Equal(got[0], want) {
		t.Fatalf("subscribes sent %q; want one naming %q", got, want)
	}
	for _, id := range want {
		if !m.active[id] {
			t.Errorf("stream %s is not active after the server's reply", id)
		}
	}

	rows := []struct{ stream, id, subjectKind, wantKind string }{
		{"commitments.subjects", "019a3f2c-6b1e-7d40-9a51-3c2b1d0e4f61", "item", "commitment.item"},
		{"commitments.subjects", "019a3f2c-6b1e-7d40-9a51-3c2b1d0e4f62", "observation", "commitment.observation"},
		{"commitments.blocks", "019a3f2c-6b1e-7d40-9a51-3c2b1d0e4f63", "block", "commitment.block"},
	}
	deliver := func(stream, subjectKind, id string, at time.Time) {
		t.Helper()
		srv.push(t, stream, "commitment.created", at, commitmentRow(id, subjectKind))
		select {
		case p := <-client.Pushes():
			m.handlePush(pushMsg{Topic: p.Topic, Event: p.Event, Payload: p.Payload})
		case <-ctx.Done():
			t.Fatalf("the %s push never arrived", subjectKind)
		}
	}
	deliver("commitments.subjects", "item", "019a3f2c-6b1e-7d40-9a51-3c2b1d0e4f60", time.Now().Add(-eventRetention-time.Hour))
	for _, r := range rows {
		deliver(r.stream, r.subjectKind, r.id, time.Now().Add(-time.Minute))
	}
	if len(m.events) != len(rows) {
		t.Fatalf("%d rows rendered; want %d", len(m.events), len(rows))
	}
	for i, r := range rows {
		e := m.events[i]
		if e.Kind != r.wantKind || e.RawKind != "commitment.created" || e.Stream != r.stream || e.ID != r.id {
			t.Errorf("row %d: kind %q, raw kind %q, stream %q, id %q; want %q, commitment.created, %q, %q",
				i, e.Kind, e.RawKind, e.Stream, e.ID, r.wantKind, r.stream, r.id)
		}
	}
}

// The Detail Panel lists a commitment row's fields in the server's
// order of meaning: the kinds and the chain first, then the subject's
// and the tree's links, then the time. Mutant killed: orderedPayloadKeys
// without subject_kind, tree_kind and chain in its priority list (they
// sort last, alphabetically, among the links).
func TestOrderedPayloadKeys_CommitmentRow(t *testing.T) {
	t.Parallel()

	got := orderedPayloadKeys(commitmentRow("019a3f2c-6b1e-7d40-9a51-3c2b1d0e4f63", "block"), map[string]bool{"id": true})
	want := []string{
		"subject_kind", "tree_kind", "chain",
		"item_id", "entropy_observation_id", "subject_block_id", "tree_block_id", "epoch_id",
		"inserted_at",
	}
	if !slices.Equal(got, want) {
		t.Errorf("keys %q; want %q", got, want)
	}
}
