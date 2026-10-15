// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/keyring"
)

// The online keyring read (SPEC.md sections 8.3 and 10.2, step 0) through
// the real reader, NewKeyringReader over httpclient's SingleRead. Every
// answer is an httptest server's, on loopback: testNet sends a read of
// www.truestamp.com to the test's own server and refuses every host that
// is not a loopback address, so no read reaches a real host. It sends each
// read it allows on through savedTransport, the transport TestMain's
// refusal on http.DefaultTransport saved.

// testNet is the transport of these tests' reads.
type testNet struct {
	// www answers the reads of www.truestamp.com, or nil.
	www *httptest.Server
	// sent counts the requests sent on to a server, and refused those to a
	// host that is not loopback.
	sent, refused atomic.Int64
}

func (n *testNet) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Hostname()
	switch ip := net.ParseIP(host); {
	case host == "www.truestamp.com" && n.www != nil:
		out := req.Clone(req.Context())
		out.URL.Scheme, out.URL.Host = "http", n.www.Listener.Addr().String()
		n.sent.Add(1)
		return savedTransport.RoundTrip(out)
	case ip != nil && ip.IsLoopback():
		n.sent.Add(1)
		return savedTransport.RoundTrip(req)
	}
	n.refused.Add(1)
	return nil, fmt.Errorf("the test refuses a read of %s, which is not loopback", host)
}

// realReader is the real keyring reader over net, each read within
// deadline.
func realReader(net *testNet, deadline time.Duration) KeyringReader {
	return NewKeyringReader(httpclient.NewSingleRead(net, deadline))
}

// server is an httptest server that answers with handler and counts the
// requests it is sent, by path.
func server(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/keyring.json" {
			http.NotFound(w, r)
			return
		}
		requests.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

// serve answers with data, 200 OK.
func serve(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }
}

// The answers a keyring read fails on, each named as a test case: the
// server closes the connection with no answer; HTTP 503; a 302 to another
// server that serves a keyring holding the key; a challenge page served
// with 200 OK, such as a CDN's bot check; a keyring that holds the key and
// fails a load check, VECTOR_KEY beside it under prod; and a server slower
// than the read's deadline.
func failingAnswers(t *testing.T, sound []byte, holding [32]byte) (map[string]http.HandlerFunc, *atomic.Int64) {
	t.Helper()
	target, reached := server(t, serve(sound))
	return map[string]http.HandlerFunc{
		"a transport failure": func(w http.ResponseWriter, _ *http.Request) {
			if conn, _, err := http.NewResponseController(w).Hijack(); err == nil {
				_ = conn.Close()
			}
		},
		"HTTP 503": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		},
		"a 302 to another server": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL+"/.well-known/keyring.json", http.StatusFound)
		},
		"a challenge page served with 200": serve([]byte(`<!DOCTYPE html><html><head><title>Just a moment...</title></head>` +
			`<body>Checking your browser before accessing www.truestamp.com.</body></html>`)),
		"a keyring failing a load check": serve(keyringBytes(keyringEntry(holding, true, 0), keyringEntry(keyring.VectorKey(), true, 1))),
		"a server slower than the deadline": func(_ http.ResponseWriter, r *http.Request) {
			// It answers nothing until the reader gives up, or 10 s pass.
			select {
			case <-r.Context().Done():
			case <-time.After(10 * time.Second):
			}
		},
	}, reached
}

// readDeadline is the deadline of these tests' reads: a loopback answer
// takes milliseconds, and a slow server is cut after it.
const readDeadline = 1 * time.Second

// Online under prod with no keyring option, the keyring read from
// www.truestamp.com governs alone (SPEC.md section 8.3): read through the
// real reader, a sound keyring binds the key, attributed truestamp, with
// the digest of the bytes served; and a read that fails, by transport, by
// HTTP 503, by a 3xx (unfollowed: the server it points at, which serves a
// keyring holding the key, is sent nothing), by a challenge page served
// with 200, by a keyring that fails a load check, or by a server slower
// than the deadline, fails KEY_BINDING keyring_unavailable, while the
// compiled pin holds the key. Each case makes exactly one request, of
// www.truestamp.com's keyring, which the test's server answers, and no
// request reaches the shared client or a host that is not loopback.
// Mutants killed: a failed online read falling back to the compiled pin
// (each failure binds); SingleRead following redirects (the 302 binds);
// the message of keyring_unavailable without its offline sentence; the
// keyring read with no deadline (the slow case does not end).
func TestKeyringRead_AFailedReadFromTruestampBindsNoKeyWhateverThePinHolds(t *testing.T) {
	before := refusals.Load()
	usePinHolding(t, pinKey)
	proof := soundProof(t)
	delete(proof, "environment")
	document := signed(t, proof, pinSigner)
	sound := keyringBytes(keyringEntry(pinKey, true, 0))
	failing, reached := failingAnswers(t, sound, pinKey)

	answers := map[string]http.HandlerFunc{"a sound keyring": serve(sound)}
	for name, handler := range failing {
		answers[name] = handler
	}
	for name, handler := range answers {
		www, requests := server(t, handler)
		net := &testNet{www: www}
		cfg, err := Options{Environment: "prod"}.Config(realReader(net, readDeadline), nil)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		run := gate(t, cfg, document)
		took := time.Since(start)
		if requests.Load() != 1 || net.sent.Load() != 1 || net.refused.Load() != 0 {
			t.Errorf("%s: %d requests of the keyring, %d sent, %d refused", name, requests.Load(), net.sent.Load(), net.refused.Load())
		}
		if took > readDeadline+2*time.Second {
			t.Errorf("%s: the read took %v", name, took)
		}
		message := run.Rows[5].Message
		if name == "a sound keyring" {
			if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [null null pass SIGNATURE]" ||
				run.Attribution != AttributionTruestamp || run.TrustRoot != (TrustRoot{Digest: digestOf(sound), Kind: KindCurrentKeyring, Location: CurrentKeyringURL}) {
				t.Errorf("%s: rows %s, attribution %q, trust_root %+v", name, got, run.Attribution, run.TrustRoot)
			}
			continue
		}
		if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [keyring_unavailable null fail KEY_BINDING] [prior_failure null skip SIGNATURE]" {
			t.Errorf("%s: rows %s", name, got)
		}
		if run.Attribution != AttributionNone || run.Keyring != nil || !run.KeyringUnavailable ||
			run.TrustRoot != (TrustRoot{Kind: KindCurrentKeyring, Location: CurrentKeyringURL}) {
			t.Errorf("%s: attribution %q, keyring %v, trust_root %+v", name, run.Attribution, run.Keyring != nil, run.TrustRoot)
		}
		for _, want := range []string{"could not be read from www.truestamp.com", "--offline", "only against the keys built into this verifier", "revoked since"} {
			if !strings.Contains(message, want) {
				t.Errorf("%s: the message %q does not say %q", name, message, want)
			}
		}
	}
	if reached.Load() != 0 {
		t.Errorf("the redirect's target was sent %d requests", reached.Load())
	}
	if got := refusals.Load() - before; got != 0 {
		t.Errorf("%d requests reached the shared client", got)
	}
}

// A keyring read from the origin an explicit --base-url names is a
// supplied keyring that governs alone (SPEC.md sections 6.4 and 8.3): a
// sound one binds, attributed keyring:<d> with the digest of the bytes
// served, located at the origin's keyring URL; one whose read fails by
// HTTP 503, by bytes that fail a load check (a key_id that is not its
// key's) or by any of the other failures is keyring_unavailable, never
// configuration_error, while the compiled pin holds the key. Mutants
// killed: a failed read from an origin given as configuration_error (the
// gate gives a bare rejection); an origin's keyring attributed truestamp.
func TestKeyringRead_AnExplicitBaseURLsOriginFailsTheSameWay(t *testing.T) {
	clearTruestampEnv(t)
	usePinHolding(t, pinKey)
	proof := soundProof(t)
	delete(proof, "environment")
	document := signed(t, proof, pinSigner)
	sound := keyringBytes(keyringEntry(pinKey, true, 0))
	wrongID := []byte(strings.Replace(string(sound), registry.KeyIDOf(pinKey), "00000000", 1))
	failing, _ := failingAnswers(t, sound, pinKey)
	failing["a key_id that is not its key's"] = serve(wrongID)

	answers := map[string]http.HandlerFunc{"a sound keyring": serve(sound)}
	for name, handler := range failing {
		answers[name] = handler
	}
	for name, handler := range answers {
		origin, requests := server(t, handler)
		_, options := loadOptions(t, "", "--base-url", origin.URL)
		url := origin.URL + "/.well-known/keyring.json"
		if options.OriginKeyringURL != url {
			t.Fatalf("%s: the origin's keyring URL is %q", name, options.OriginKeyringURL)
		}
		net := &testNet{}
		cfg, err := options.Config(realReader(net, readDeadline), nil)
		if err != nil || cfg.Keyring != (KeyringSource{Kind: SourceOrigin, URL: url}) {
			t.Fatalf("%s: %+v, %v", name, cfg.Keyring, err)
		}
		run, rejection, err := Gate(cfg, document)
		if err != nil || rejection != nil {
			t.Fatalf("%s: %v, %v; want a report", name, rejection, err)
		}
		if requests.Load() != 1 || net.refused.Load() != 0 {
			t.Errorf("%s: %d requests, %d refused", name, requests.Load(), net.refused.Load())
		}
		if name == "a sound keyring" {
			want := TrustRoot{Digest: digestOf(sound), Kind: KindSuppliedKeyring, Location: url}
			if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [null null pass SIGNATURE]" ||
				run.Attribution != "keyring:"+digestOf(sound) || run.TrustRoot != want {
				t.Errorf("%s: rows %s, attribution %q, trust_root %+v", name, got, run.Attribution, run.TrustRoot)
			}
			continue
		}
		if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [keyring_unavailable null fail KEY_BINDING] [prior_failure null skip SIGNATURE]" ||
			run.TrustRoot != (TrustRoot{Kind: KindSuppliedKeyring, Location: url}) || !strings.Contains(run.Rows[5].Message, "could not be read from "+url) {
			t.Errorf("%s: rows %s, trust_root %+v, message %q", name, got, run.TrustRoot, run.Rows[5].Message)
		}
	}
}

// A run asked to be offline is governed by the compiled pin alone (SPEC.md
// section 8.3), here a keyring holding pinKey through the test seam, and
// reads no keyring, an explicit --base-url given or not: the key the pin
// holds binds, attributed truestamp, with trust_root compiled_pin and the
// pin's digest; any other fails key_binding_failed, with the message to
// update the verifier or run online. Mutants killed: Configure loading
// keyring.CompiledPin() rather than the seam (pinKey does not bind); an
// offline run reading the keyring online; the compiled pin's
// key_binding_failed message without "update the verifier or run online".
func TestKeyringRead_OfflineTheCompiledPinGoverns(t *testing.T) {
	clearTruestampEnv(t)
	pin := usePinHolding(t, pinKey)
	production := soundProof(t)
	delete(production, "environment")
	for _, args := range [][]string{{"--offline"}, {"--offline", "--base-url", "http://127.0.0.1:1"}} {
		_, options := loadOptions(t, "", args...)
		cfg, err := options.Config(byteReader(func(url string) ([]byte, error) {
			t.Errorf("%v: an offline run read %s", args, url)
			return nil, errors.New("read")
		}), nil)
		if err != nil || cfg.Online {
			t.Fatalf("%v: %+v, %v", args, cfg, err)
		}
		run := gate(t, cfg, signed(t, production, pinSigner))
		if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [null null pass SIGNATURE]" ||
			run.Attribution != AttributionTruestamp || run.TrustRoot != (TrustRoot{Digest: digestOf(pin), Kind: KindCompiledPin}) {
			t.Errorf("%v, the pin's key: rows %s, attribution %q, trust_root %+v", args, got, run.Attribution, run.TrustRoot)
		}
		run = gate(t, cfg, signed(t, production, signerOf("a key the pin lacks")))
		if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [key_binding_failed null fail KEY_BINDING] [prior_failure null skip SIGNATURE]" ||
			!strings.Contains(run.Rows[5].Message, "update the verifier or run online") {
			t.Errorf("%v, another key: rows %s, message %q", args, got, run.Rows[5].Message)
		}
	}
}

// A base URL whose host is not a valid DNS name, such as
// www.truestamp.com.. with two trailing dots, which Options.Config reads as
// another origin than www.truestamp.com, gives a keyring read the real
// reader refuses with no request made: KEY_BINDING fails
// keyring_unavailable. Mutant killed: NewKeyringReader without its URL
// check (the read is sent, and the test's transport refuses its host).
func TestKeyringRead_AHostThatIsNotADNSNameIsRefused(t *testing.T) {
	clearTruestampEnv(t)
	usePinHolding(t, pinKey)
	proof := soundProof(t)
	delete(proof, "environment")
	_, options := loadOptions(t, "", "--base-url", "https://www.truestamp.com..")
	net := &testNet{}
	cfg, err := options.Config(realReader(net, readDeadline), nil)
	if err != nil || cfg.Keyring.Kind != SourceOrigin {
		t.Fatalf("%+v, %v", cfg.Keyring, err)
	}
	run := gate(t, cfg, signed(t, proof, pinSigner))
	if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [keyring_unavailable null fail KEY_BINDING] [prior_failure null skip SIGNATURE]" {
		t.Errorf("rows %s", got)
	}
	if net.sent.Load() != 0 || net.refused.Load() != 0 {
		t.Errorf("%d sent, %d refused; want no request", net.sent.Load(), net.refused.Load())
	}
}

// The keyring URLs a read is made to: http or https, no user information,
// and a host that is an IP address or a DNS name of labels of 1 to 63
// letters, digits and hyphens, none at either end, at most 253 octets, with
// at most one trailing dot. Mutants killed: validDNSName without its empty
// label check (two trailing dots, and a..b, pass); without the trailing dot
// trimmed (a name with one is refused); with a label of 64 octets allowed;
// with a name of 254 octets allowed; without the hyphen check; with no
// character check; checkKeyringURL without its scheme check or its user
// information check.
func TestKeyringRead_TheURLCheck(t *testing.T) {
	label63 := strings.Repeat("a", 63)
	long := strings.TrimSuffix(strings.Repeat(label63+".", 4), ".")
	allowed := []string{
		"https://www.truestamp.com/.well-known/keyring.json",
		"https://www.truestamp.com./.well-known/keyring.json",
		"http://localhost:4000/.well-known/keyring.json",
		"https://staging-1.truestamp.example:8443/.well-known/keyring.json",
		"http://127.0.0.1:54321/.well-known/keyring.json",
		"http://[::1]:54321/.well-known/keyring.json",
		"https://" + label63 + ".example/.well-known/keyring.json",
		"https://" + long[:253-len(".example")] + ".example/k.json",
	}
	refused := []string{
		"https://www.truestamp.com../.well-known/keyring.json",
		"https://a..b/.well-known/keyring.json",
		"https://.truestamp.com/.well-known/keyring.json",
		"https://-www.truestamp.com/.well-known/keyring.json",
		"https://www-.truestamp.com/.well-known/keyring.json",
		"https://www_1.truestamp.com/.well-known/keyring.json",
		"https://" + label63 + "a.example/.well-known/keyring.json",
		"https://" + long[:254-len(".example")] + ".example/k.json",
		"https://" + long + "/k.json",
		"ftp://www.truestamp.com/.well-known/keyring.json",
		"https://user:pass@www.truestamp.com/.well-known/keyring.json",
		"https:///.well-known/keyring.json",
		"https://www.truestamp.com:x/.well-known/keyring.json",
	}
	for _, raw := range allowed {
		if err := checkKeyringURL(raw); err != nil {
			t.Errorf("%s: %v", raw, err)
		}
	}
	for _, raw := range refused {
		if err := checkKeyringURL(raw); err == nil {
			t.Errorf("%s: allowed", raw)
		}
	}
}

// A keyring answered from a replay is a value with no bytes of its own
// (SPEC.md section 13.2): read from www.truestamp.com it loads under the
// run's E_v and binds, attributed truestamp, with trust_root's digest null
// (section 10.5); one that fails a load check binds nothing; and as the
// answer to a read from another origin, a supplied keyring whose
// attribution names the digest of its bytes, it is a failed read. Mutants
// killed: a replayed value loaded by encoding its canonical form and
// loading the bytes (trust_root gets a digest); the origin check left out
// (the origin's replayed keyring binds, attributed "keyring:" with no
// digest); a replayed value loaded under a fixed "test" (VECTOR_KEY
// binds under prod).
func TestKeyringRead_AReplayedKeyringHasNoDigest(t *testing.T) {
	proof := soundProof(t)
	delete(proof, "environment")
	document := signed(t, proof, pinSigner)
	value := func(data []byte) any {
		parsed, err := strictjson.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	replayed := func(v any) KeyringReader {
		return func(string) (KeyringRead, error) { return KeyringRead{Replayed: true, Value: v}, nil }
	}
	sound := value(keyringBytes(keyringEntry(pinKey, true, 0)))
	withVector := value(keyringBytes(keyringEntry(pinKey, true, 0), keyringEntry(keyring.VectorKey(), true, 1)))
	const origin = "https://keys.truestamp.example/.well-known/keyring.json"
	cases := []struct {
		name   string
		source KeyringSource
		value  any
		rows   string
		root   TrustRoot
	}{
		{"from www.truestamp.com", KeyringSource{}, sound,
			"[null /environment pass ENVIRONMENT] [null null pass KEY_BINDING] [null null pass SIGNATURE]",
			TrustRoot{Kind: KindCurrentKeyring, Location: CurrentKeyringURL}},
		{"failing a load check", KeyringSource{}, withVector,
			"[null /environment pass ENVIRONMENT] [keyring_unavailable null fail KEY_BINDING] [prior_failure null skip SIGNATURE]",
			TrustRoot{Kind: KindCurrentKeyring, Location: CurrentKeyringURL}},
		{"from another origin", KeyringSource{Kind: SourceOrigin, URL: origin}, sound,
			"[null /environment pass ENVIRONMENT] [keyring_unavailable null fail KEY_BINDING] [prior_failure null skip SIGNATURE]",
			TrustRoot{Kind: KindSuppliedKeyring, Location: origin}},
	}
	for _, c := range cases {
		cfg, _ := NewConfig("prod")
		cfg.Online, cfg.Keyring, cfg.ReadKeyring = true, c.source, replayed(c.value)
		run := gate(t, cfg, document)
		if got := gateRows(run); got != c.rows || run.TrustRoot != c.root {
			t.Errorf("%s: rows %s, trust_root %+v", c.name, got, run.TrustRoot)
		}
		if c.name == "from www.truestamp.com" && run.Attribution != AttributionTruestamp {
			t.Errorf("%s: attribution %q", c.name, run.Attribution)
		}
	}
}

// The CLI's own reader, DefaultKeyringReader, reads through
// http.DefaultTransport, which TestMain's loopbackOnly lets reach a server
// on loopback: it reads a keyring served there, and follows no redirect:
// the server a 301 points at is sent nothing, and the read fails. Mutant
// killed: DefaultKeyringReader reading through the shared client,
// httpclient.GetJSON (which follows redirects, and which this package's
// TestMain refuses on every host).
func TestKeyringRead_TheDefaultReader(t *testing.T) {
	sound := keyringBytes(keyringEntry(pinKey, true, 0))
	srv, requests := server(t, serve(sound))
	answer, err := DefaultKeyringReader()(srv.URL + "/.well-known/keyring.json")
	if err != nil || string(answer.Bytes) != string(sound) || answer.Replayed || requests.Load() != 1 {
		t.Fatalf("%+v, %v", answer, err)
	}
	moved, _ := server(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/.well-known/keyring.json", http.StatusMovedPermanently)
	})
	if _, err := DefaultKeyringReader()(moved.URL + "/.well-known/keyring.json"); httpclient.Status(err) != http.StatusMovedPermanently {
		t.Fatalf("a 301: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatal("the redirect was followed")
	}
}

// Online under prod with no keyring option, the keyring read from
// www.truestamp.com replaces the compiled pin and adds nothing to it
// (SPEC.md section 8.3): with the pin holding pinKey, and
// www.truestamp.com serving a sound keyring that lacks it, a proof pinKey
// signed fails KEY_BINDING key_binding_failed, attributed to no one, with
// trust_root the keyring read and its digest, whether the read is made by
// the real reader or answered as bytes. Mutants killed: the pin's keys
// added to the keyring read (pinKey binds, attributed truestamp);
// KEY_BINDING looking a key the keyring read lacks up in the pin (pinKey
// binds, attributed truestamp).
func TestKeyringRead_TheKeyringReadReplacesThePin(t *testing.T) {
	usePinHolding(t, pinKey)
	proof := soundProof(t)
	delete(proof, "environment")
	document := signed(t, proof, pinSigner)
	lacking := keyringBytes(keyringEntry(publicOf(signerOf("a key of the keyring read")), true, 0))
	www, requests := server(t, serve(lacking))
	net := &testNet{www: www}
	readers := map[string]KeyringReader{
		"the real reader": realReader(net, readDeadline),
		"bytes": byteReader(func(url string) ([]byte, error) {
			if url != CurrentKeyringURL {
				t.Errorf("bytes: a read of %s", url)
			}
			return lacking, nil
		}),
	}
	for name, reader := range readers {
		cfg, err := Options{Environment: "prod"}.Config(reader, nil)
		if err != nil {
			t.Fatal(err)
		}
		run := gate(t, cfg, document)
		if got := gateRows(run); got != "[null /environment pass ENVIRONMENT] [key_binding_failed null fail KEY_BINDING] [prior_failure null skip SIGNATURE]" {
			t.Errorf("%s: rows %s, attribution %q", name, got, run.Attribution)
		}
		if run.Attribution != AttributionNone || run.KeyringUnavailable ||
			run.TrustRoot != (TrustRoot{Digest: digestOf(lacking), Kind: KindCurrentKeyring, Location: CurrentKeyringURL}) {
			t.Errorf("%s: attribution %q, unavailable %t, trust_root %+v", name, run.Attribution, run.KeyringUnavailable, run.TrustRoot)
		}
		if message := run.Rows[5].Message; !strings.Contains(message, "not in the keyring read from www.truestamp.com") {
			t.Errorf("%s: the message %q", name, message)
		}
	}
	if requests.Load() != 1 || net.sent.Load() != 1 || net.refused.Load() != 0 {
		t.Errorf("%d requests of the keyring, %d sent, %d refused", requests.Load(), net.sent.Load(), net.refused.Load())
	}
}

// DefaultKeyringReader's reads are built with no transport of their own,
// so they go through http.DefaultTransport, and with KeyringReadDeadline,
// 9 seconds, the Truestamp application's bound on each of its verification
// reads (Truestamp.SourceCache.ReadBounds). Mutants killed:
// KeyringReadDeadline at 9 hours; DefaultKeyringReader building its reads
// with a deadline of its own.
func TestKeyringRead_TheDefaultReadersDeadline(t *testing.T) {
	if KeyringReadDeadline != 9*time.Second {
		t.Fatalf("KeyringReadDeadline = %v, want 9s", KeyringReadDeadline)
	}
	transport, deadline := defaultReadsOf(t)
	if transport != nil || deadline != KeyringReadDeadline {
		t.Fatalf("DefaultKeyringReader's reads are built with %v and %v, want nil and %v", transport, deadline, KeyringReadDeadline)
	}
}

// The real reader reads a body to at most MAX_READ_OCTETS octets (SPEC.md
// section 10.1, J1): www.truestamp.com serving exactly MAX_READ_OCTETS + 1
// octets gives a truncated read and no bytes, and serving exactly
// MAX_READ_OCTETS octets gives them all, read whole. Mutants killed: the
// reader's bound at 2 x MAX_READ_OCTETS, or at MAX_READ_OCTETS + 1, one
// octet too many (the longer body is read whole); the bound at
// MAX_READ_OCTETS - 1 (the body of MAX_READ_OCTETS is truncated).
func TestKeyringRead_TheBodyIsBoundedAtMaxReadOctets(t *testing.T) {
	for _, tc := range []struct {
		name      string
		served    int
		truncated bool
	}{
		{"one octet over the bound", strictjson.MaxReadOctets + 1, true},
		{"exactly the bound", strictjson.MaxReadOctets, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(strings.Repeat(" ", tc.served))
			www, requests := server(t, serve(body))
			net := &testNet{www: www}
			answer, err := realReader(net, readDeadline)(CurrentKeyringURL)
			switch {
			case tc.truncated && (!httpclient.IsTruncated(err) || answer.Bytes != nil || answer.Replayed):
				t.Fatalf("%d octets served: %d read, replayed %t, %v; want a truncated read", tc.served, len(answer.Bytes), answer.Replayed, err)
			case !tc.truncated && (err != nil || !bytes.Equal(answer.Bytes, body) || answer.Replayed):
				t.Fatalf("%d octets served: %d read, replayed %t, %v; want them read whole", tc.served, len(answer.Bytes), answer.Replayed, err)
			}
			if requests.Load() != 1 || net.sent.Load() != 1 || net.refused.Load() != 0 {
				t.Errorf("%d requests of the keyring, %d sent, %d refused", requests.Load(), net.sent.Load(), net.refused.Load())
			}
		})
	}
}
