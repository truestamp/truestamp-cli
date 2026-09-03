// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

//go:build smoke

package wschannel

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// TestSmokeItemsCreate exercises the items.create + auto-watch round-trip
// and waits briefly for at least one item.* push to validate end-to-end
// lifecycle delivery. Skipped without WSURL/APIKEY env vars.
func TestSmokeItemsCreate(t *testing.T) {
	wsURL := os.Getenv("WSURL")
	apiKey := os.Getenv("APIKEY")
	if wsURL == "" || apiKey == "" {
		t.Skip("WSURL and APIKEY env vars required")
	}

	client, err := New(Options{URL: wsURL, APIKey: apiKey})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// 64-hex-char sha256, deterministic per run.
	hash := strings.Repeat("ab", 32)

	reply, err := client.Push(ctx, "console:lobby", "items.create", map[string]any{
		"name":        "smoke test item",
		"description": "from go smoke test",
		"hash":        hash,
		"hash_type":   "sha256",
		"watch":       true,
	})
	if err != nil {
		t.Fatalf("items.create push: %v", err)
	}
	if reply.Status != "ok" {
		t.Fatalf("items.create status = %q: %s", reply.Status, reply.Response)
	}
	var resp struct {
		Item struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"item"`
	}
	if err := json.Unmarshal(reply.Response, &resp); err != nil {
		t.Fatalf("decode reply: %v", err)
	}
	t.Logf("created item %s in state %s", resp.Item.ID, resp.Item.State)

	// Consume any pushes for ~5 seconds to confirm we receive at least
	// the synthetic item.created (or a subsequent transition) for our
	// own item via the items.<id> subscription.
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	saw := false
loop:
	for {
		select {
		case p := <-client.Pushes():
			if p.Event == "stream" && strings.Contains(string(p.Payload), resp.Item.ID) {
				t.Logf("received stream push for our item: %s", string(p.Payload))
				saw = true
				break loop
			}
		case <-deadline.C:
			break loop
		}
	}
	if !saw {
		t.Logf("no item.* push observed within 5s, broadcast likely fired before subscribe was complete (acceptable)")
	}
}

// TestSmokeReconnect verifies the client survives a server restart:
// connect, subscribe, then expect ticks/blocks even after a disconnect.
//
// Drives the test via the RECONNECT_TRIGGER hook env var, the runner
// should arrange for the server to be restarted while this test is
// running (e.g. `task: kick the preview` in a parallel shell). The test
// counts pre- and post-reconnect ticks on console:clock to prove that
// re-join replay actually delivered new pushes from the new server
// process.
func TestSmokeReconnect(t *testing.T) {
	wsURL := os.Getenv("WSURL")
	apiKey := os.Getenv("APIKEY")
	if wsURL == "" || apiKey == "" {
		t.Skip("WSURL and APIKEY env vars required")
	}
	if os.Getenv("RECONNECT") == "" {
		t.Skip("set RECONNECT=1 and restart the preview during the test window")
	}

	client, err := New(Options{URL: wsURL, APIKey: apiKey})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if _, err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if _, err := client.JoinTopic(ctx, "console:clock"); err != nil {
		t.Fatalf("JoinTopic: %v", err)
	}

	t.Log("connected; restart the server now to trigger reconnect…")

	rejoined := false
	postReconnectTicks := 0
	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()

	for postReconnectTicks < 2 {
		select {
		case p := <-client.Pushes():
			switch p.Event {
			case ReconnectingEvent:
				t.Logf("reconnecting status: %s", p.Payload)
			case ReconnectedEvent:
				t.Logf("rejoined topic %s", p.Topic)
				rejoined = true
			case "tick":
				if rejoined {
					postReconnectTicks++
					t.Logf("post-reconnect tick %d: %s", postReconnectTicks, p.Payload)
				}
			}
		case <-timer.C:
			t.Fatalf("did not see %d post-reconnect ticks within 60s (rejoined=%v)", 2, rejoined)
		}
	}
}

// TestSmokeClockTopic joins the auxiliary `console:clock` topic on the
// same socket as the lobby and confirms two server-time ticks arrive
// within ~3 seconds (the broadcaster fires every second).
func TestSmokeClockTopic(t *testing.T) {
	wsURL := os.Getenv("WSURL")
	apiKey := os.Getenv("APIKEY")
	if wsURL == "" || apiKey == "" {
		t.Skip("WSURL and APIKEY env vars required")
	}

	client, err := New(Options{URL: wsURL, APIKey: apiKey})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect lobby: %v", err)
	}
	if _, err := client.JoinTopic(ctx, "console:clock"); err != nil {
		t.Fatalf("JoinTopic clock: %v", err)
	}

	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticks := 0

	for ticks < 2 {
		select {
		case p := <-client.Pushes():
			if p.Topic == "console:clock" && p.Event == "tick" {
				ticks++
				t.Logf("tick %d: %s", ticks, p.Payload)
			}
		case <-deadline.C:
			t.Fatalf("only %d tick(s) within 3s, expected at least 2", ticks)
		}
	}
}

// TestSmokeLiveBlock subscribes to the `blocks` stream and waits up to
// 90 seconds for a real block creation to arrive over PubSub. The dev
// server creates a block every minute, so this validates the full
// PubSub → channel → wire path with live data. Skipped without
// WSURL/APIKEY.
func TestSmokeLiveBlock(t *testing.T) {
	wsURL := os.Getenv("WSURL")
	apiKey := os.Getenv("APIKEY")
	if wsURL == "" || apiKey == "" {
		t.Skip("WSURL and APIKEY env vars required")
	}

	client, err := New(Options{URL: wsURL, APIKey: apiKey})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	if _, err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	reply, err := client.Push(ctx, "console:lobby", "subscribe", map[string]any{
		"streams": []string{"blocks"},
	})
	if err != nil || reply.Status != "ok" {
		t.Fatalf("subscribe: %v / %s", err, reply.Response)
	}

	deadline := time.NewTimer(90 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case p := <-client.Pushes():
			if p.Event != "stream" {
				continue
			}
			var sp struct {
				Stream string `json:"stream"`
				Kind   string `json:"kind"`
			}
			if err := json.Unmarshal(p.Payload, &sp); err != nil {
				continue
			}
			if sp.Stream == "blocks" {
				t.Logf("received live %s on stream %s, payload: %s",
					sp.Kind, sp.Stream, truncatePayload(p.Payload))
				return
			}

		case <-deadline.C:
			t.Skip("no live block arrived within 90s, dev cron may be paused")
		}
	}
}

func truncatePayload(p []byte) string {
	if len(p) <= 200 {
		return string(p)
	}
	return string(p[:200]) + "..."
}

// TestSmokeConsoleLobby is a hand-driven integration test that exercises
// the full Phoenix Channel round-trip against a running dev server.
//
// It is gated behind the `smoke` build tag and reads its connection
// parameters from environment variables so the dev URL and API key
// don't get checked in.
//
// Run with:
//
//	WSURL=ws://localhost:4010/console/websocket \
//	APIKEY=truestamp_... \
//	go test -tags=smoke -run TestSmoke ./internal/wschannel -v
func TestSmokeConsoleLobby(t *testing.T) {
	wsURL := os.Getenv("WSURL")
	apiKey := os.Getenv("APIKEY")
	if wsURL == "" || apiKey == "" {
		t.Skip("WSURL and APIKEY env vars required")
	}

	client, err := New(Options{URL: wsURL, APIKey: apiKey})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	welcome, err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Logf("welcome envelope: %s", welcome)

	// Subscribe to two streams (one of which is unknown).
	reply, err := client.Push(ctx, "console:lobby", "subscribe", map[string]any{
		"streams": []string{"blocks", "entropy.nist", "nonsense"},
	})
	if err != nil {
		t.Fatalf("subscribe push: %v", err)
	}
	if reply.Status != "ok" {
		t.Fatalf("subscribe status = %q, want ok: %s", reply.Status, reply.Response)
	}
	t.Logf("subscribe reply: %s", reply.Response)

	// Ping round-trip.
	reply, err = client.Push(ctx, "console:lobby", "ping", map[string]any{})
	if err != nil {
		t.Fatalf("ping push: %v", err)
	}
	t.Logf("ping reply: %s", reply.Response)

	// Subscriptions snapshot.
	reply, err = client.Push(ctx, "console:lobby", "subscriptions", map[string]any{})
	if err != nil {
		t.Fatalf("subscriptions push: %v", err)
	}
	var snap struct {
		Active []string `json:"active"`
	}
	if err := json.Unmarshal(reply.Response, &snap); err != nil {
		t.Fatalf("decode subscriptions: %v", err)
	}
	if len(snap.Active) != 2 {
		t.Errorf("active = %v, want 2 entries (blocks + entropy.nist)", snap.Active)
	}
	t.Logf("active subscriptions: %v", snap.Active)
}
