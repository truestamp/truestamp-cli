// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package wschannel implements a minimal Phoenix Channels V2 client
// targeted at long-lived authenticated connections (TUI panes, daemons).
//
// The wire format is the Phoenix V2 array form:
//
//	[join_ref, ref, topic, event, payload]
//
// All five elements are present in every message. join_ref and ref may be
// JSON null. Topics, events, and payloads are arbitrary JSON.
//
// This package is intentionally small, it does not depend on the Go
// "phx" community library so that we control reconnection behaviour and
// keep the dependency tree shallow. The codec is exposed for unit tests
// and for callers that want to drive a raw connection.
package wschannel

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Frame is a single Phoenix Channel message in V2 array form.
type Frame struct {
	JoinRef *string         // optional; null in JSON when absent
	Ref     *string         // optional; null for server pushes
	Topic   string          // e.g. "console:lobby" or "phoenix" (heartbeat)
	Event   string          // e.g. "phx_join", "subscribe", "stream"
	Payload json.RawMessage // raw JSON object
}

// MarshalJSON encodes the frame as the canonical 5-element array.
func (f Frame) MarshalJSON() ([]byte, error) {
	payload := f.Payload
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	arr := [5]any{f.JoinRef, f.Ref, f.Topic, f.Event, json.RawMessage(payload)}
	return json.Marshal(arr)
}

// UnmarshalJSON decodes a 5-element Phoenix array into a Frame.
func (f *Frame) UnmarshalJSON(data []byte) error {
	var arr [5]json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("frame: unmarshal array: %w", err)
	}
	if err := unmarshalNullableString(arr[0], &f.JoinRef); err != nil {
		return fmt.Errorf("frame[0] join_ref: %w", err)
	}
	if err := unmarshalNullableString(arr[1], &f.Ref); err != nil {
		return fmt.Errorf("frame[1] ref: %w", err)
	}
	if err := json.Unmarshal(arr[2], &f.Topic); err != nil {
		return fmt.Errorf("frame[2] topic: %w", err)
	}
	if err := json.Unmarshal(arr[3], &f.Event); err != nil {
		return fmt.Errorf("frame[3] event: %w", err)
	}
	f.Payload = arr[4]
	return nil
}

// PhxReply is the payload shape Phoenix sends back for messages that have
// a ref, i.e. anything that expects a reply.
//
//	{"status":"ok","response":{...}}    or    {"status":"error","response":{...}}
type PhxReply struct {
	Status   string          `json:"status"`
	Response json.RawMessage `json:"response"`
}

// ParseReply extracts the PhxReply from a "phx_reply" frame.
func ParseReply(f Frame) (PhxReply, error) {
	if f.Event != "phx_reply" {
		return PhxReply{}, errors.New("not a phx_reply frame")
	}
	var r PhxReply
	if err := json.Unmarshal(f.Payload, &r); err != nil {
		return PhxReply{}, fmt.Errorf("phx_reply payload: %w", err)
	}
	return r, nil
}

func unmarshalNullableString(raw json.RawMessage, dst **string) error {
	if len(raw) == 0 || string(raw) == "null" {
		*dst = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	*dst = &s
	return nil
}

// Helper for callers: build a payload-less frame.
func newFrame(joinRef, ref *string, topic, event string, payload any) (Frame, error) {
	var raw json.RawMessage
	if payload == nil {
		raw = json.RawMessage("{}")
	} else {
		b, err := json.Marshal(payload)
		if err != nil {
			return Frame{}, fmt.Errorf("encode payload: %w", err)
		}
		raw = b
	}
	return Frame{JoinRef: joinRef, Ref: ref, Topic: topic, Event: event, Payload: raw}, nil
}
