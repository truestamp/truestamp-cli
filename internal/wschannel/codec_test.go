// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"encoding/json"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		joinRef *string
		ref     *string
		topic   string
		event   string
	}{
		{
			name:    "phx_join with both refs",
			input:   `["1","1","console:lobby","phx_join",{}]`,
			joinRef: ptr("1"),
			ref:     ptr("1"),
			topic:   "console:lobby",
			event:   "phx_join",
		},
		{
			name:    "server push with null refs",
			input:   `[null,null,"console:lobby","stream",{"stream":"blocks"}]`,
			joinRef: nil,
			ref:     nil,
			topic:   "console:lobby",
			event:   "stream",
		},
		{
			name:    "heartbeat",
			input:   `[null,"h1","phoenix","heartbeat",{}]`,
			joinRef: nil,
			ref:     ptr("h1"),
			topic:   "phoenix",
			event:   "heartbeat",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var f Frame
			if err := json.Unmarshal([]byte(tc.input), &f); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if f.Topic != tc.topic {
				t.Errorf("topic = %q, want %q", f.Topic, tc.topic)
			}
			if f.Event != tc.event {
				t.Errorf("event = %q, want %q", f.Event, tc.event)
			}
			if !strPtrEq(f.JoinRef, tc.joinRef) {
				t.Errorf("join_ref mismatch: got %v want %v", deref(f.JoinRef), deref(tc.joinRef))
			}
			if !strPtrEq(f.Ref, tc.ref) {
				t.Errorf("ref mismatch: got %v want %v", deref(f.Ref), deref(tc.ref))
			}

			out, err := json.Marshal(f)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			// Round-trip through unmarshal again to confirm equivalence
			// (canonical JSON ordering may differ from input).
			var f2 Frame
			if err := json.Unmarshal(out, &f2); err != nil {
				t.Fatalf("unmarshal round-trip: %v", err)
			}
			if f2.Topic != f.Topic || f2.Event != f.Event {
				t.Errorf("round-trip lost fields: %+v -> %+v", f, f2)
			}
		})
	}
}

func TestParseReply(t *testing.T) {
	t.Parallel()

	frame := Frame{
		Topic:   "console:lobby",
		Event:   "phx_reply",
		Payload: json.RawMessage(`{"status":"ok","response":{"pong":12345}}`),
	}

	r, err := ParseReply(frame)
	if err != nil {
		t.Fatalf("ParseReply: %v", err)
	}
	if r.Status != "ok" {
		t.Errorf("status = %q, want ok", r.Status)
	}
	var resp struct {
		Pong int `json:"pong"`
	}
	if err := json.Unmarshal(r.Response, &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Pong != 12345 {
		t.Errorf("pong = %d, want 12345", resp.Pong)
	}
}

func TestParseReplyRejectsNonReply(t *testing.T) {
	t.Parallel()

	frame := Frame{Event: "stream", Payload: json.RawMessage(`{}`)}
	if _, err := ParseReply(frame); err == nil {
		t.Fatalf("expected error for non phx_reply frame")
	}
}

func ptr[T any](v T) *T { return &v }

func strPtrEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func deref(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
