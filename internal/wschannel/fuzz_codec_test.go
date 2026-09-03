// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/redact"
)

// FuzzFrameUnmarshal feeds arbitrary bytes to Frame.UnmarshalJSON and
// asserts the only outcomes are (a) a non-panicking error or (b) a
// usable Frame whose round-trip is stable.
//
// The Phoenix V2 wire format is sourced from the network, so any bytes a
// remote peer can choose to send must not crash the client. The
// happy-path JSON shapes are seeded; the fuzzer mutates from there.
//
// Run:
//
//	go test -run=^$ -fuzz=FuzzFrameUnmarshal -fuzztime=30s ./internal/wschannel/...
func FuzzFrameUnmarshal(f *testing.F) {
	seeds := []string{
		`["1","1","console:lobby","phx_join",{}]`,
		`[null,null,"console:lobby","stream",{"stream":"blocks"}]`,
		`[null,"h1","phoenix","heartbeat",{}]`,
		`["2","42","console:lobby","items.create",{"name":"x","hash":"deadbeef"}]`,
		`["1","1","console:lobby","phx_reply",{"status":"ok","response":{}}]`,
		// Common malformed shapes the SafeV2Serializer fix targeted,
		// these MUST surface as errors, not panics.
		`[`,
		`null`,
		`{"oops":true}`,
		`["1","1","topic","event"]`,            // 4-element, missing payload
		`["1","1","topic","event",{},"extra"]`, // 6-element
		`[1,1,"topic","event",{}]`,             // numeric refs (Phoenix uses string)
		`[null,null,null,null,null]`,           // all-null
		`["a","b",123,"event",{}]`,             // non-string topic
		"[null,null,\"t\",\"e\",{}]\n[null,null]\n", // multi-line paste
		"\n",
		"",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var frame Frame
		err := json.Unmarshal(data, &frame)
		if err != nil {
			return // any error is fine; we only forbid panics
		}
		// If decode succeeded, encoding it back must not panic and must
		// produce something that decodes again.
		out, err := json.Marshal(frame)
		if err != nil {
			t.Fatalf("re-marshal failed for accepted frame %q: %v", data, err)
		}
		var f2 Frame
		if err := json.Unmarshal(out, &f2); err != nil {
			t.Fatalf("round-trip decode failed for %q: %v (encoded as %q)",
				data, err, out)
		}
		// Topic and Event must round-trip exactly, strings of arbitrary
		// bytes including invalid UTF-8 should still match themselves.
		if f2.Topic != frame.Topic {
			t.Fatalf("topic round-trip mismatch: %q -> %q", frame.Topic, f2.Topic)
		}
		if f2.Event != frame.Event {
			t.Fatalf("event round-trip mismatch: %q -> %q", frame.Event, f2.Event)
		}
	})
}

// FuzzParseReply feeds arbitrary phx_reply payloads to ParseReply,
// confirming malformed status/response objects produce errors rather
// than panics. Unlike FuzzFrameUnmarshal, this exercises only the
// inner payload decoder, the frame envelope is fixed.
//
// Run:
//
//	go test -run=^$ -fuzz=FuzzParseReply -fuzztime=30s ./internal/wschannel/...
func FuzzParseReply(f *testing.F) {
	seeds := []string{
		`{"status":"ok","response":{}}`,
		`{"status":"error","response":{"reason":"forbidden"}}`,
		`{"status":"ok","response":{"item":{"id":"01H","state":"created"}}}`,
		`{}`,
		`{"status":42}`,
		`{"status":"ok","response":null}`,
		`{"status":"ok"}`,
		`null`,
		`[]`,
		`"not an object"`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		frame := Frame{
			Topic:   "console:lobby",
			Event:   "phx_reply",
			Payload: json.RawMessage(payload),
		}
		// Either a valid reply or an error, never a panic.
		_, _ = ParseReply(frame)
	})
}

// FuzzRedactSecrets ensures the api_key redaction (now provided by
// internal/redact) is correct across arbitrary URL-shaped inputs.
// Specifically:
//   - never panics
//   - if the input contains "api_key=truestamp_..." the output must
//     not contain that literal token verbatim
//
// The shared regex `api_key=[^&"\s]*` is bounded (negated character
// class, no backtracking), so this is mostly a regression net rather
// than a bug hunt, it pins behavior so a future "improvement" can't
// accidentally reintroduce a leak. Lives in wschannel as well as in
// the redact package itself because the WebSocket dial path is the
// historically attested leak path the regex was added to plug.
//
// Run:
//
//	go test -run=^$ -fuzz=FuzzRedactSecrets -fuzztime=30s ./internal/wschannel/...
func FuzzRedactSecrets(f *testing.F) {
	seeds := []string{
		`api_key=truestamp_secret`,
		`dial ws://x/y?api_key=truestamp_abc&vsn=2.0.0`,
		`Get "http://x?api_key=truestamp_xyz"`,
		`no key here`,
		`api_key=`,
		`api_key=   `,
		`api_key=truestamp_`,
		``,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		out := redact.String(in)

		// Invariant: any "api_key=truestamp_..." token in the input must
		// not appear verbatim in the output.
		idx := 0
		for {
			at := strings.Index(in[idx:], "api_key=truestamp_")
			if at < 0 {
				break
			}
			start := idx + at
			end := start + len("api_key=truestamp_")
			// Walk forward to the natural token boundary the regex uses:
			// a character in the negated class [^&"\s]
			for end < len(in) {
				c := in[end]
				if c == '&' || c == '"' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
					break
				}
				end++
			}
			token := in[start:end]
			if token != "api_key=truestamp_" && strings.Contains(out, token) {
				t.Fatalf("redact.String leaked token %q from input %q (output %q)",
					token, in, out)
			}
			idx = end
		}
	})
}
