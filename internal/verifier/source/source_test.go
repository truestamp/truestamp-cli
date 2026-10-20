// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package source

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// Every answer here is an httptest server's, on loopback.

const (
	production = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	passphrase = "Test SDF Network ; September 2015"
)

// request is what a server saw of one request.
type request struct {
	method, path, contentType, auth, body string
}

// recording is an httptest server that answers every request with answer,
// which reads the request's body again, and records what it saw of each.
func recording(t *testing.T, answer http.HandlerFunc) (*httptest.Server, func() []request) {
	t.Helper()
	var seen atomic.Value
	seen.Store([]request{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen.Store(append(seen.Load().([]request), request{r.Method, r.URL.Path, r.Header.Get("Content-Type"), r.Header.Get("Authorization"), string(body)}))
		r.Body = io.NopCloser(bytes.NewReader(body))
		answer(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []request { return seen.Load().([]request) }
}

func text(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }
}

func reads() *httpclient.SingleRead { return httpclient.NewSingleRead(nil, 5*time.Second) }

// Each identity read makes the call of SPEC.md section 10.4's table, once:
// Horizon's GET <base>/ and Esplora's GET <base>/block-height/0, a base
// with a path and a trailing slash joined once, with no Authorization; and
// Bitcoin Core's getblockhash 0, a JSON-RPC POST with its Content-Type and
// the credentials as HTTP Basic, the call's id echoed in the answer.
// Mutants killed: endpoint without its trim of trailing slashes (the path
// is /api//block-height/0); CoreIdentity without its Content-Type;
// CoreIdentity calling getblockcount; the Basic token without the colon.
func TestIdentity_EachReadAndWhatItSends(t *testing.T) {
	creds := NewCredentials("rpcuser", "rpcpassword")
	horizon, horizonSeen := recording(t, text(`{"network_passphrase": "`+passphrase+`", "history_latest_ledger": 5}`))
	if got, err := Identity(context.Background(), reads(), Horizon, horizon.URL, creds); err != nil || got != passphrase {
		t.Fatalf("Horizon: %q, %v", got, err)
	}
	esplora, esploraSeen := recording(t, text(production))
	if got, err := Identity(context.Background(), reads(), Esplora, esplora.URL+"/api/", creds); err != nil || got != production {
		t.Fatalf("Esplora: %q, %v", got, err)
	}
	core, coreSeen := recording(t, text(`{"result":"`+production+`","error":null,"id":"truestamp-verify"}`+"\n"))
	if got, err := Identity(context.Background(), reads(), BitcoinCore, core.URL, creds); err != nil || got != production {
		t.Fatalf("Bitcoin Core: %q, %v", got, err)
	}
	basic := "Basic " + base64.StdEncoding.EncodeToString([]byte("rpcuser:rpcpassword"))
	want := map[string]request{
		"horizon": {"GET", "/", "", "", ""},
		"esplora": {"GET", "/api/block-height/0", "", "", ""},
		"core":    {"POST", "/", "application/json", basic, `{"jsonrpc":"1.0","id":"truestamp-verify","method":"getblockhash","params":[0]}`},
	}
	for name, seen := range map[string][]request{"horizon": horizonSeen(), "esplora": esploraSeen(), "core": coreSeen()} {
		if len(seen) != 1 || seen[0] != want[name] {
			t.Errorf("%s saw %+v, want %+v", name, seen, want[name])
		}
	}
	if _, err := Identity(context.Background(), reads(), BitcoinCore, core.URL, Credentials{}); err != nil || coreSeen()[1].auth != "" {
		t.Errorf("Bitcoin Core with no credentials: %v, %+v", err, coreSeen())
	}
	if _, err := Identity(context.Background(), reads(), Provider("electrum"), core.URL, Credentials{}); err == nil {
		t.Error("an unknown provider is read")
	}
}

// An identity read is found or unreachable, never not found: a 404, a 3xx,
// a 429 and a 503 from each provider, an answer over IdentityOctets, and
// every answer not in its form are an *Unreachable whose reason quotes no
// body; an answer of exactly IdentityOctets is read. Mutants killed:
// ascii128 accepting the quote; hex64 accepting uppercase; CoreIdentity
// without its error check (a Core error with a result passes); CoreIdentity
// without its id check; IdentityOctets passed as limit - 1 (the answer of
// exactly the bound is refused).
func TestIdentity_EveryAnswerWithoutAValueInItsFormIsUnreachable(t *testing.T) {
	status := func(code int) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "quoted-body", code) }
	}
	redirect := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/elsewhere", http.StatusMovedPermanently)
	}
	core := func(result, rpcError, id string) http.HandlerFunc {
		return text(`{"result":` + result + `,"error":` + rpcError + `,"id":` + id + `}`)
	}
	cases := map[Provider]map[string]http.HandlerFunc{
		Horizon: {
			"404": status(404), "3xx": redirect, "429": status(429), "503": status(503),
			"not JSON":            text("quoted-body"),
			"an array":            text(`["` + passphrase + `"]`),
			"no passphrase":       text(`{"history_latest_ledger": 5}`),
			"a number":            text(`{"network_passphrase": 5}`),
			"129 characters":      text(`{"network_passphrase": "` + strings.Repeat("a", 129) + `"}`),
			"empty":               text(`{"network_passphrase": ""}`),
			"a quote":             text(`{"network_passphrase": "a\"b"}`),
			"a control character": text(`{"network_passphrase": "a\u0001b"}`),
			"over the bound":      text(`{"network_passphrase": "` + passphrase + `"}` + strings.Repeat(" ", IdentityOctets)),
			"a repeated name":     text(`{"network_passphrase": "x", "network_passphrase": "` + passphrase + `"}`),
		},
		Esplora: {
			"404": status(404), "3xx": redirect, "429": status(429), "503": status(503),
			"uppercase":      text(strings.ToUpper(production)),
			"63 digits":      text(production[1:]),
			"a newline":      text(production + "\n"),
			"quoted":         text(`"` + production + `"`),
			"over the bound": text(strings.Repeat("0", IdentityOctets+1)),
		},
		BitcoinCore: {
			"404": status(404), "3xx": redirect, "429": status(429), "503": status(503),
			"a -8 with 500": func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(500)
				_, _ = w.Write([]byte(`{"result":null,"error":{"code":-8,"message":"Block height out of range"},"id":"truestamp-verify"}`))
			},
			"not JSON":              text("quoted-body"),
			"an error and a result": core(`"`+production+`"`, `{"code":-28,"message":"Loading"}`, `"truestamp-verify"`),
			"another id":            core(`"`+production+`"`, `null`, `"another"`),
			"no id":                 text(`{"result":"` + production + `","error":null}`),
			"a number":              core(`0`, `null`, `"truestamp-verify"`),
			"uppercase":             core(`"`+strings.ToUpper(production)+`"`, `null`, `"truestamp-verify"`),
		},
	}
	for provider, answers := range cases {
		for name, answer := range answers {
			srv, seen := recording(t, answer)
			value, err := Identity(context.Background(), reads(), provider, srv.URL, NewCredentials("u", "p"))
			var unreachable *Unreachable
			if !errors.As(err, &unreachable) || value != "" || strings.Contains(err.Error(), "quoted-body") || len(seen()) != 1 {
				t.Errorf("%s, %s: %q, %v, %d requests; want unreachable, one request, no body quoted", provider, name, value, err, len(seen()))
			}
		}
	}
	exact := `{"network_passphrase": "` + passphrase + `"}`
	srv, _ := recording(t, text(exact+strings.Repeat(" ", IdentityOctets-len(exact))))
	if value, err := StellarIdentity(context.Background(), reads(), srv.URL); err != nil || value != passphrase {
		t.Fatalf("an answer of exactly %d octets: %q, %v", IdentityOctets, value, err)
	}
}

// ascii128 and hex64 accept exactly what spec/encodings.json's ASCII128
// and HEX64 patterns match, for every byte, at both ends of the length.
// Mutant killed: ascii128 accepting 129 characters.
func TestIdentity_TheFormsAreTheSpecsPatterns(t *testing.T) {
	encodings := specdata.Encodings()
	ascii := regexp.MustCompile(`^` + encodings["ASCII128"].Pattern + `$`)
	hex := regexp.MustCompile(`^` + encodings["HEX64"].Pattern + `$`)
	for b := 0; b < 256; b++ {
		c := string([]byte{byte(b)})
		for _, s := range []string{c, strings.Repeat("a", 127) + c, strings.Repeat("a", 128) + c} {
			if ascii128(s) != ascii.MatchString(s) {
				t.Fatalf("ascii128(%q) = %v", s, ascii128(s))
			}
		}
		for _, s := range []string{production[:63] + c, production + c} {
			if hex64(s) != hex.MatchString(s) {
				t.Fatalf("hex64(%q) = %v", s, hex64(s))
			}
		}
	}
	if ascii128("") || hex64("") {
		t.Fatal("the empty string")
	}
}

// Credentials print masked, whatever prints them: every fmt verb, inside a
// struct printed with %v, %+v and %#v, encoding/json and MarshalText; and
// Empty is true only for no user and no password. Mutants killed: Format
// left out (%d prints the members); Empty true when only the password is
// set.
func TestCredentials_PrintMasked(t *testing.T) {
	const secret = "planted-secret-91be"
	creds := NewCredentials("rpcuser-"+secret, secret)
	holder := struct {
		Name  string
		Creds Credentials
	}{"core", creds}
	var printed []string
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d", "%T"} {
		printed = append(printed, fmt.Sprintf(verb, creds), fmt.Sprintf(verb, holder), fmt.Sprintf(verb, &holder))
	}
	data, err := json.Marshal(holder)
	if err != nil {
		t.Fatal(err)
	}
	textForm, _ := creds.MarshalText()
	printed = append(printed, string(data), string(textForm), creds.String(), creds.GoString())
	for _, p := range printed {
		if strings.Contains(p, secret) || strings.Contains(p, "rpcuser") {
			t.Fatalf("printed %q", p)
		}
	}
	if string(data) != `{"Name":"core","Creds":"`+Masked+`"}` {
		t.Fatalf("json = %s", data)
	}
	if !(Credentials{}).Empty() || NewCredentials("", secret).Empty() || NewCredentials("u", "").Empty() {
		t.Fatal("Empty")
	}
}

// CheckURL refuses what no read may be made to, and its error never quotes
// the URL: a URL that does not parse, a scheme other than http and https,
// no host, user information, a query and a fragment. Mutants killed:
// CheckURL without the user information check; without the query check.
func TestCheckURL_RefusesAndQuotesNothing(t *testing.T) {
	const secret = "planted-secret-c4d1"
	for _, raw := range []string{
		"http://u:" + secret + "@[::1",
		"ftp://" + secret + ".example",
		"https://",
		"https://u:" + secret + "@node.example",
		"https://" + secret + "@node.example",
		"https://node.example/?" + secret,
		"https://node.example/?",
		"https://node.example/#" + secret,
		secret,
	} {
		err := CheckURL(raw)
		if err == nil || strings.Contains(err.Error(), secret) {
			t.Errorf("%q: %v", raw, err)
		}
	}
	for _, raw := range []string{"https://horizon.stellar.org", "http://127.0.0.1:8332", "https://blockstream.info/api/", "http://[::1]:18443"} {
		if err := CheckURL(raw); err != nil {
			t.Errorf("%q: %v", raw, err)
		}
	}
}

// Each provider names its chain, and an unknown one none. Mutant killed:
// Chain giving bitcoin for Horizon.
func TestProvider_Chain(t *testing.T) {
	if Horizon.Chain() != "stellar" || Esplora.Chain() != "bitcoin" || BitcoinCore.Chain() != "bitcoin" || Provider("x").Chain() != "" {
		t.Fatal("Chain")
	}
}

// echoing answers a request with raw, given the Authorization header the
// request carried, written straight to the connection, so the answer can
// be anything, HTTP or not.
func echoing(raw func(auth string) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = buf.WriteString(raw(auth))
		_ = buf.Flush()
	}
}

// coreEchoes are Bitcoin Core nodes that echo the Authorization header
// they were sent back in their answer, each where a reason could quote
// it, and one that fails in transport.
var coreEchoes = map[string]http.HandlerFunc{
	"the status line": echoing(func(auth string) string { return auth + "\r\n\r\n" }),
	"a header line":   echoing(func(auth string) string { return "HTTP/1.1 200 OK\r\n" + auth + "\r\n\r\n" }),
	"error.message": func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"result":null,"error":{"code":-1,"message":%q},"id":"truestamp-verify"}`, r.Header.Get("Authorization"))
	},
	"result": func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"result":%q,"error":null,"id":"truestamp-verify"}`, r.Header.Get("Authorization"))
	},
	"id": func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"result":%q,"error":null,"id":%q}`, production, r.Header.Get("Authorization"))
	},
	"a transport failure": echoing(func(string) string { return "" }),
}

// A Bitcoin Core node that echoes the Authorization header it was sent,
// into a malformed status line, a malformed header line, a JSON-RPC
// answer's error.message, its result or its id, and one whose connection
// is hijacked and closed, give an *Unreachable whose reason, error and
// every fmt verb of it hold neither the planted user, the planted password
// nor their Basic encoding, nor the node's URL, and no value. Mutants
// killed: a fromRead that gives "no answer: " and the transport
// error's text (the status line's and the header line's echoes are
// printed, and the transport failure's reason names the URL); CoreIdentity
// quoting the Core error, the result, or the answer whose id is not this
// call's, in its reason (each echo is printed).
func TestIdentity_NothingANodeEchoesReachesAReason(t *testing.T) {
	const user, secret = "rpcuser-planted-41f7", "planted-secret-41f7"
	basic := base64.StdEncoding.EncodeToString([]byte(user + ":" + secret))
	for name, answer := range coreEchoes {
		t.Run(name, func(t *testing.T) {
			srv, seen := recording(t, answer)
			value, err := Identity(context.Background(), reads(), BitcoinCore, srv.URL, NewCredentials(user, secret))
			var unreachable *Unreachable
			if !errors.As(err, &unreachable) || value != "" || len(seen()) != 1 || seen()[0].auth != "Basic "+basic {
				t.Fatalf("%q, %v, %+v; want unreachable after one request with the credentials", value, err, seen())
			}
			printed := fmt.Sprintf("%v|%+v|%s|%q|%s", err, err, err, err, unreachable.Reason)
			for _, planted := range []string{user, secret, basic, strings.TrimPrefix(srv.URL, "http://")} {
				if strings.Contains(printed, planted) {
					t.Fatalf("%q is printed: %s", planted, printed)
				}
			}
		})
	}
}

// A Bitcoin Core read, a POST, ends at its SingleRead's deadline when the
// node never answers: with a 300 ms deadline it is unreachable, the
// deadline named and the transport's context.DeadlineExceeded its cause,
// well before a 5 s guard. Mutant killed: SingleRead's read applying its
// deadline to a GET alone (the POST waits on the node until the guard).
func TestIdentity_ACoreReadEndsAtItsDeadline(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, err := CoreIdentity(context.Background(), httpclient.NewSingleRead(nil, 300*time.Millisecond), srv.URL, Credentials{})
		done <- err
	}()
	select {
	case err := <-done:
		var unreachable *Unreachable
		if !errors.As(err, &unreachable) || unreachable.Reason != reasonDeadline || !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 3*time.Second {
			t.Fatalf("%v after %v; want the deadline's reason", err, time.Since(start))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the Core read was not cut at its deadline")
	}
}
