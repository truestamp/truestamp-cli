// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// The NIST beacon's read of SOURCES (SPEC.md section 10.4, "Entropy
// objects" and "Common sources"), against httptest servers on loopback.

const (
	pulseOutput = "FDBA0E7DA81C11481C72F9B01CBAE7C364C53270C8EDFFB34537C8F7028D75163D70C142873DD560A6E985101A99E02BD7C7EA6ABA8BD36F3C581CB65C2F6216"
	pulseStamp  = "2026-09-26T01:48:00.000Z"
)

// pulseRead is the read of the pulse at chainIndex 2 and pulseIndex
// 1200018.
var pulseRead = map[string]any{"kind": KindNISTPulse, "chainIndex": num(2), "pulseIndex": num(1200018)}

// beacon is a NIST beacon that answers GET <path>/chain/2/pulse/1200018
// with pulse, the JSON text of the pulse object, wrapped as the beacon
// wraps it, and any other path with a 404.
func beacon(pulse string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.URL.Path, "/beacon/2.0") != "/chain/2/pulse/1200018" {
			http.Error(w, "Pulse Not Found", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"pulse":` + pulse + `}`))
	}
}

// soundPulse is a pulse as the beacon writes it, with members the read
// does not use.
const soundPulse = `{"uri":"https://beacon.nist.gov/beacon/2.0/chain/2/pulse/1200018","version":"2.0","cipherSuite":0,"period":60000,` +
	`"chainIndex":2,"pulseIndex":1200018,"timeStamp":"` + pulseStamp + `","localRandomValue":"AB","statusCode":0,` +
	`"signatureValue":"CD","outputValue":"` + pulseOutput + `"}`

// The read makes the one call of SPEC.md section 10.4's table, a GET of
// <base>/chain/<chainIndex>/pulse/<pulseIndex> with no credential, a base
// with a path and a trailing slash joined once, and gives its found answer
// in the form of a replay record: the pulse's outputValue and timeStamp,
// as the pulse holds them, every other member of the pulse left out.
// Mutants killed: the path's indices swapped (a 404, not found); a field
// copied that the read does not use (the value holds version); the
// credentials sent to the beacon (an Authorization is seen).
func TestNISTReads_TheCallAndItsAnswer(t *testing.T) {
	srv, seen := recording(t, beacon(soundPulse))
	got := Read(context.Background(), reads(), NISTBeacon, srv.URL+"/beacon/2.0/", coreCreds, pulseRead)
	want := Answer{Outcome: OutcomeFound, Value: map[string]any{"outputValue": pulseOutput, "timeStamp": pulseStamp}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v, want %+v", got, want)
	}
	if requests := seen(); !reflect.DeepEqual(requests, []request{{"GET", "/beacon/2.0/chain/2/pulse/1200018", "", "", ""}}) {
		t.Fatalf("the beacon saw %+v", requests)
	}
}

// A pulse the beacon does not hold is not found: its 404, with no value,
// in one request. Every other answer is unreachable, with a fixed reason
// that quotes no body, in one request: a 3xx, a 429, a 503 and a challenge
// page at 403 and at 200; a body over NISTOctets; a body that is not a
// JSON object, or an object with no pulse object; a pulse that names
// another chainIndex or pulseIndex, lacks either or gives one as a string,
// so the answer is not about the pulse asked for. An answer of exactly
// NISTOctets is read. Mutants killed: NISTPulse reading any status as not
// found (a 503 is not found); NISTPulse without its check of the pulse's
// indices (another pulse is found); the check of the pulseIndex alone (a
// pulse of another chain is found); NISTOctets passed as limit - 1.
func TestNISTReads_NotFoundAndUnreachable(t *testing.T) {
	status := func(code int) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "quoted-body", code) }
	}
	challenge := func(code int) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(code)
			_, _ = w.Write([]byte("<html><title>Just a moment...</title>quoted-body</html>"))
		}
	}
	redirect := func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/elsewhere", http.StatusFound) }
	other := strings.Replace(soundPulse, `"pulseIndex":1200018`, `"pulseIndex":1200019`, 1)
	otherChain := strings.Replace(soundPulse, `"chainIndex":2`, `"chainIndex":1`, 1)
	noIndex := strings.Replace(soundPulse, `"pulseIndex":1200018,`, ``, 1)
	stringIndex := strings.Replace(soundPulse, `"chainIndex":2`, `"chainIndex":"2"`, 1)
	for name, tc := range map[string]struct {
		answer http.HandlerFunc
		want   Outcome
	}{
		"404":                    {status(404), OutcomeNotFound},
		"3xx":                    {redirect, OutcomeUnreachable},
		"429":                    {status(429), OutcomeUnreachable},
		"503":                    {status(503), OutcomeUnreachable},
		"a 403 challenge":        {challenge(403), OutcomeUnreachable},
		"a 200 challenge":        {challenge(200), OutcomeUnreachable},
		"over the bound":         {text(`{"pulse":` + soundPulse + `}` + strings.Repeat(" ", NISTOctets)), OutcomeUnreachable},
		"an array":               {text(`[` + soundPulse + `]`), OutcomeUnreachable},
		"no pulse":               {text(soundPulse), OutcomeUnreachable},
		"a pulse string":         {text(`{"pulse":"quoted-body"}`), OutcomeUnreachable},
		"another pulseIndex":     {beacon(other), OutcomeUnreachable},
		"another chainIndex":     {beacon(otherChain), OutcomeUnreachable},
		"no pulseIndex":          {beacon(noIndex), OutcomeUnreachable},
		"a chainIndex as string": {beacon(stringIndex), OutcomeUnreachable},
	} {
		srv, seen := recording(t, tc.answer)
		got := Read(context.Background(), reads(), NISTBeacon, srv.URL+"/beacon/2.0", Credentials{}, pulseRead)
		if got.Outcome != tc.want || got.Value != nil || strings.Contains(got.Reason, "quoted-body") || (tc.want == OutcomeUnreachable) != (got.Reason != "") || len(seen()) != 1 {
			t.Errorf("%s: %+v after %d requests; want %s, no value, no body quoted", name, got, len(seen()), tc.want)
		}
	}
	exact := `{"pulse":` + soundPulse + `}`
	srv, _ := recording(t, text(exact+strings.Repeat(" ", NISTOctets-len(exact))))
	if got := NISTPulse(context.Background(), reads(), srv.URL, num(2), num(1200018)); got.Outcome != OutcomeFound || got.Value["outputValue"] != pulseOutput {
		t.Fatalf("an answer of exactly %d octets: %+v", NISTOctets, got)
	}
}

// A pulse that lacks a field the read gives, or gives one in another form,
// is found with what it holds, so the rule that reads the answer judges it
// (SPEC.md section 10.4, "Answers"): no outputValue gives the timeStamp
// alone, neither gives an empty value, and a timeStamp written as a number
// is kept as the pulse holds it. Mutants killed: NISTPulse making a pulse
// that lacks a field unreachable; found copying only strings (the numeric
// timeStamp is dropped).
func TestNISTReads_APartialPulseIsFound(t *testing.T) {
	noOutput := `{"chainIndex":2,"pulseIndex":1200018,"timeStamp":"` + pulseStamp + `"}`
	neither := `{"chainIndex":2,"pulseIndex":1200018}`
	numeric := `{"chainIndex":2,"pulseIndex":1200018,"timeStamp":5,"outputValue":"` + pulseOutput + `"}`
	for name, tc := range map[string]struct {
		pulse string
		want  map[string]any
	}{
		"no outputValue":      {noOutput, map[string]any{"timeStamp": pulseStamp}},
		"neither":             {neither, map[string]any{}},
		"a numeric timeStamp": {numeric, map[string]any{"outputValue": pulseOutput, "timeStamp": num(5)}},
	} {
		srv, _ := recording(t, beacon(tc.pulse))
		got := NISTPulse(context.Background(), reads(), srv.URL, num(2), num(1200018))
		if want := (Answer{Outcome: OutcomeFound, Value: tc.want}); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %+v, want %+v", name, got, want)
		}
	}
}

// A pulse read is made to no server when its chainIndex or pulseIndex is
// not an INT (a negative, a fraction, a string, none), nor a NIST read of
// a provider that does not serve it, nor another kind of read of the
// beacon: each is unreachable, with a fixed reason. Mutants killed:
// NISTPulse taking a negative index; Read sending a NIST read to an
// Esplora server; Read sending a Bitcoin read to the beacon.
func TestNISTReads_NothingIsReadForARefusedParameter(t *testing.T) {
	srv, seen := recording(t, text(`{}`))
	for name, tc := range map[string]struct {
		provider Provider
		read     map[string]any
	}{
		"a negative chainIndex":   {NISTBeacon, map[string]any{"kind": KindNISTPulse, "chainIndex": num(-1), "pulseIndex": num(5)}},
		"a fractional pulseIndex": {NISTBeacon, map[string]any{"kind": KindNISTPulse, "chainIndex": num(2), "pulseIndex": num(1.5)}},
		"a pulseIndex string":     {NISTBeacon, map[string]any{"kind": KindNISTPulse, "chainIndex": num(2), "pulseIndex": "5"}},
		"no chainIndex":           {NISTBeacon, map[string]any{"kind": KindNISTPulse, "pulseIndex": num(5)}},
		"a NIST read, Esplora":    {Esplora, pulseRead},
		"a Bitcoin read, NIST":    {NISTBeacon, map[string]any{"kind": KindBitcoinBlock, "hash": blockHash}},
	} {
		got := Read(context.Background(), reads(), tc.provider, srv.URL, Credentials{}, tc.read)
		if got.Outcome != OutcomeUnreachable || got.Reason == "" || got.Value != nil {
			t.Errorf("%s: %+v", name, got)
		}
	}
	if len(seen()) != 0 {
		t.Fatalf("requests made: %+v", seen())
	}
}

// A pulse read ends at its SingleRead's deadline when the beacon never
// answers: with a 300 ms deadline it is unreachable, naming the deadline,
// well before a 5 s guard. Mutant killed: NISTPulse giving a reason of its
// own for a failed read (the reason is not the deadline's fixed text).
func TestNISTReads_AReadEndsAtItsDeadline(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)
	done := make(chan Answer, 1)
	start := time.Now()
	go func() {
		done <- Read(context.Background(), httpclient.NewSingleRead(nil, 300*time.Millisecond), NISTBeacon, srv.URL, Credentials{}, pulseRead)
	}()
	select {
	case got := <-done:
		if got.Outcome != OutcomeUnreachable || got.Reason != reasonDeadline || time.Since(start) > 3*time.Second {
			t.Fatalf("%+v after %v; want the deadline's reason", got, time.Since(start))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the read was not cut at its deadline")
	}
}
