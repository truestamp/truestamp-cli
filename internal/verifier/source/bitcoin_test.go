// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package source

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// Esplora's and Bitcoin Core's reads of CHAIN (SPEC.md section 10.4,
// "Bitcoin commitment entries", "genesis.bitcoin" and "Common sources"),
// against httptest servers on loopback.

const (
	blockHash  = "066ee8ed37e3b69aaef26bca28f2818674e61412e1be5bc8f17b0e4cc5971386"
	coreUser   = "rpcuser-planted-77c1"
	corePasswd = "planted-secret-77c1"
)

// coreBasic is the Authorization header coreUser and corePasswd send.
var coreBasic = "Basic " + base64.StdEncoding.EncodeToString([]byte(coreUser+":"+corePasswd))

// coreCreds are the credentials the Bitcoin Core tests send.
var coreCreds = NewCredentials(coreUser, corePasswd)

// esplora is an Esplora server: its tip, the hash of each height it holds,
// a 404 for any other, and the block for each hash, by path.
func esplora(tip string, hashes, blocks map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api")
		switch {
		case path == "/blocks/tip/height":
			_, _ = w.Write([]byte(tip))
		case strings.HasPrefix(path, "/block-height/") && hashes[strings.TrimPrefix(path, "/block-height/")] != "":
			_, _ = w.Write([]byte(hashes[strings.TrimPrefix(path, "/block-height/")]))
		case strings.HasPrefix(path, "/block/") && blocks[strings.TrimPrefix(path, "/block/")] != "":
			_, _ = w.Write([]byte(blocks[strings.TrimPrefix(path, "/block/")]))
		default:
			http.Error(w, "Block not found", http.StatusNotFound)
		}
	}
}

// coreCallOf is a JSON-RPC call a test node was sent.
type coreCallOf struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params []any  `json:"params"`
}

// core is a Bitcoin Core node that answers each method with the JSON text
// answers gives it, with status 200 OK, or, for a text that starts with
// "500 ", status 500 and the rest of the text, as Core answers an error.
func core(answers map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var call coreCallOf
		_ = json.NewDecoder(r.Body).Decode(&call)
		answer, ok := answers[call.Method]
		if !ok {
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"result":null,"error":{"code":-32601,"message":"Method not found"},"id":"truestamp-verify"}`))
			return
		}
		if rest, isError := strings.CutPrefix(answer, "500 "); isError {
			w.WriteHeader(500)
			answer = rest
		}
		_, _ = w.Write([]byte(answer))
	}
}

// result is Core's answer with result.
func result(value string) string {
	return `{"result":` + value + `,"error":null,"id":"truestamp-verify"}` + "\n"
}

// rpcError is Core's error answer with code, at status 500.
func rpcError(code int) string {
	return fmt.Sprintf(`500 {"result":null,"error":{"code":%d,"message":"no"},"id":"truestamp-verify"}`, code)
}

// heightRead and tipRead are the reads of a Bitcoin source.
func heightRead(h float64) map[string]any {
	return map[string]any{"kind": KindBitcoinBlockAtHeight, "network": "regtest", "height": num(h)}
}

var tipRead = map[string]any{"kind": KindBitcoinTip, "network": "regtest"}

// hashRead is the read of the block blockHash by its hash.
var hashRead = map[string]any{"kind": KindBitcoinBlock, "network": "regtest", "hash": blockHash}

// Each read makes its calls of SPEC.md section 10.4's table and gives its
// found answer in the form of a replay record: Esplora's tip text as
// height, and for the block at a height the hash GET
// <base>/block-height/<height> gives and the mediantime and tx_count of
// GET <base>/block/<hash> as median_time_past and transaction_count, with
// no credential sent, a base with a path and a trailing slash joined once;
// Bitcoin Core's getblockcount as height, and getblockhash <height> then
// getblockheader "<hash>" for hash, median_time_past (mediantime) and
// transaction_count (nTx), each a JSON-RPC 1.0 POST with its Content-Type
// and the credentials as HTTP Basic; every other field of the block left
// out. Mutants killed: tx_count and mediantime not renamed (the value
// holds tx_count); Bitcoin Core's nTx read as tx_count (no
// transaction_count); a field copied that the read does not use (the
// value holds height); getblockheader called with the height (its params
// are [232]); Read giving an Esplora source Bitcoin Core's reads
// (Esplora is sent POSTs with the credentials).
func TestBitcoinReads_EachCallAndItsAnswer(t *testing.T) {
	block := `{"id":"` + blockHash + `","height":232,"tx_count":5,"mediantime":1790387220,"timestamp":1790387300,"size":900}`
	es, esSeen := recording(t, esplora("233", map[string]string{"232": blockHash}, map[string]string{blockHash: block}))
	header := result(`{"hash":"` + blockHash + `","height":232,"nTx":5,"mediantime":1790387220,"time":1790387300,"confirmations":2}`)
	node, nodeSeen := recording(t, core(map[string]string{"getblockcount": result("233"), "getblockhash": result(`"` + blockHash + `"`), "getblockheader": header}))
	full := Answer{Outcome: OutcomeFound, Value: map[string]any{"hash": blockHash, "median_time_past": num(1790387220), "transaction_count": num(5)}}
	tip := Answer{Outcome: OutcomeFound, Value: map[string]any{"height": num(233)}}
	for _, tc := range []struct {
		name     string
		provider Provider
		base     string
		read     map[string]any
		want     Answer
	}{
		{"Esplora's tip", Esplora, es.URL + "/api/", tipRead, tip},
		{"Esplora's block at a height", Esplora, es.URL + "/api/", heightRead(232), full},
		{"Bitcoin Core's tip", BitcoinCore, node.URL, tipRead, tip},
		{"Bitcoin Core's block at a height", BitcoinCore, node.URL, heightRead(232), full},
	} {
		if got := Read(context.Background(), reads(), tc.provider, tc.base, coreCreds, tc.read); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
	wantEsplora := []request{
		{"GET", "/api/blocks/tip/height", "", "", ""},
		{"GET", "/api/block-height/232", "", "", ""},
		{"GET", "/api/block/" + blockHash, "", "", ""},
	}
	if got := esSeen(); !reflect.DeepEqual(got, wantEsplora) {
		t.Errorf("Esplora saw %+v, want %+v", got, wantEsplora)
	}
	call := func(method, params string) request {
		return request{"POST", "/", "application/json", coreBasic, `{"jsonrpc":"1.0","id":"truestamp-verify","method":"` + method + `","params":[` + params + `]}`}
	}
	wantCore := []request{call("getblockcount", ""), call("getblockhash", "232"), call("getblockheader", `"`+blockHash+`"`)}
	if got := nodeSeen(); !reflect.DeepEqual(got, wantCore) {
		t.Errorf("Bitcoin Core saw %+v, want %+v", got, wantCore)
	}
}

// A block at a height the source does not hold is not found: Esplora's
// 404 to GET <base>/block-height/<height>, and Bitcoin Core's error -8 (a
// height out of range) or -5 (no such block) to getblockhash, which Core
// answers with status 500, with no second call made and no value. A tip
// is never not found (SPEC.md section 13.2): Esplora's 404 and Core's -8
// to getblockcount are unreachable. Mutants killed: coreCall calling
// Post, which reads no error body (Core's -8 is unreachable); coreCall taking any error code as not
// found (a -28 is not found); CoreTip giving a not-found answer (its -8 is
// not found); EsploraTip reading a 404 as not found.
func TestBitcoinReads_NotFound(t *testing.T) {
	es, esSeen := recording(t, esplora("", nil, nil))
	if got := EsploraBlockAtHeight(context.Background(), reads(), es.URL, num(300)); !reflect.DeepEqual(got, Answer{Outcome: OutcomeNotFound}) || len(esSeen()) != 1 {
		t.Errorf("Esplora's 404: %+v after %d requests", got, len(esSeen()))
	}
	missing, _ := recording(t, http.NotFound)
	if got := EsploraTip(context.Background(), reads(), missing.URL); got.Outcome != OutcomeUnreachable || got.Reason != "HTTP 404" {
		t.Errorf("Esplora's tip 404: %+v", got)
	}
	for code, want := range map[int]Outcome{-8: OutcomeNotFound, -5: OutcomeNotFound, -28: OutcomeUnreachable, -1: OutcomeUnreachable} {
		node, seen := recording(t, core(map[string]string{"getblockhash": rpcError(code), "getblockcount": rpcError(code)}))
		got := CoreBlockAtHeight(context.Background(), reads(), node.URL, coreCreds, num(300))
		if got.Outcome != want || got.Value != nil || len(seen()) != 1 {
			t.Errorf("Core's %d: %+v after %d requests, want %s", code, got, len(seen()), want)
		}
		if tip := CoreTip(context.Background(), reads(), node.URL, coreCreds); tip.Outcome != OutcomeUnreachable || tip.Reason != "Bitcoin Core answered with an error" {
			t.Errorf("Core's tip with %d: %+v", code, tip)
		}
	}
}

// Every other answer to a tip or to a block's first call is unreachable,
// with a fixed reason that quotes no body, one request made: a 3xx, a
// 429, a 503 and a challenge page at 403 and at 200 from each provider; a
// body over BitcoinOctets; Esplora's tip that is not the decimal digits of
// an INT (a leading zero, a sign, an exponent, a final newline, past
// MAX_INTEGER) and a hash that is not a HEX64; Bitcoin Core's answer to
// another call, its answer that is not an object, a 404 body that is not
// one of its errors, a tip that is not an INT and a hash that is not a
// HEX64. An answer of exactly BitcoinOctets is read. Mutants killed:
// decimalPattern taking a leading zero; decimalInt without its
// MAX_INTEGER bound; coreCall without its id check; intValue taking a
// negative (a tip of -1 is found); BitcoinOctets passed as limit - 1.
func TestBitcoinReads_Unreachable(t *testing.T) {
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
	common := map[string]http.HandlerFunc{
		"3xx": redirect, "429": status(429), "503": status(503),
		"a 403 challenge": challenge(403), "a 200 challenge": challenge(200),
		"over the bound": text(strings.Repeat("1", BitcoinOctets+1)),
	}
	for name, answer := range common {
		for _, provider := range []Provider{Esplora, BitcoinCore} {
			for _, read := range []map[string]any{tipRead, heightRead(232)} {
				srv, seen := recording(t, answer)
				got := Read(context.Background(), reads(), provider, srv.URL, coreCreds, read)
				if got.Outcome != OutcomeUnreachable || got.Value != nil || got.Reason == "" || strings.Contains(got.Reason, "quoted-body") || len(seen()) != 1 {
					t.Errorf("%s, %s %s: %+v after %d requests", provider, read["kind"], name, got, len(seen()))
				}
			}
		}
	}
	for name, tc := range map[string]struct {
		provider Provider
		answer   http.HandlerFunc
		read     map[string]any
	}{
		"a leading zero":          {Esplora, text("0233"), tipRead},
		"a sign":                  {Esplora, text("+233"), tipRead},
		"an exponent":             {Esplora, text("2e2"), tipRead},
		"a final newline":         {Esplora, text("233\n"), tipRead},
		"past MAX_INTEGER":        {Esplora, text("9007199254740992"), tipRead},
		"an uppercase hash":       {Esplora, esplora("1", map[string]string{"232": strings.ToUpper(blockHash)}, nil), heightRead(232)},
		"a short hash":            {Esplora, esplora("1", map[string]string{"232": blockHash[1:]}, nil), heightRead(232)},
		"another call's answer":   {BitcoinCore, text(`{"result":233,"error":null,"id":"another"}`), tipRead},
		"no id":                   {BitcoinCore, text(`{"result":233,"error":null}`), tipRead},
		"an array":                {BitcoinCore, text(`[233]`), tipRead},
		"not JSON":                {BitcoinCore, text(`quoted-body`), tipRead},
		"a 404 not an RPC answer": {BitcoinCore, status(404), heightRead(232)},
		"a negative tip":          {BitcoinCore, core(map[string]string{"getblockcount": result("-1")}), tipRead},
		"a fractional tip":        {BitcoinCore, core(map[string]string{"getblockcount": result("2.5")}), tipRead},
		"a tip string":            {BitcoinCore, core(map[string]string{"getblockcount": result(`"233"`)}), tipRead},
		"a hash number":           {BitcoinCore, core(map[string]string{"getblockhash": result("5")}), heightRead(232)},
		"an uppercase Core hash":  {BitcoinCore, core(map[string]string{"getblockhash": result(`"` + strings.ToUpper(blockHash) + `"`)}), heightRead(232)},
	} {
		srv, seen := recording(t, tc.answer)
		got := Read(context.Background(), reads(), tc.provider, srv.URL, coreCreds, tc.read)
		if got.Outcome != OutcomeUnreachable || got.Value != nil || got.Reason == "" || strings.Contains(got.Reason, "quoted-body") || len(seen()) != 1 {
			t.Errorf("%s: %+v after %d requests", name, got, len(seen()))
		}
	}
	padded := result("233")
	padded += strings.Repeat(" ", BitcoinOctets-len(padded))
	node, _ := recording(t, core(map[string]string{"getblockcount": padded}))
	if got := CoreTip(context.Background(), reads(), node.URL, coreCreds); got.Outcome != OutcomeFound || got.Value["height"] != num(233) {
		t.Fatalf("an answer of exactly %d octets: %+v", BitcoinOctets, got)
	}
}

// When the first call of a block at a height gives the hash and the second
// gives no answer, or not every field as an INT, the answer is found with
// the hash and only the fields the second call gave as INTs, so the rule
// that reads only the hash is judged on it and one that reads another
// field finds it absent (SPEC.md section 10.4, "Answers"): a 503 or a
// Core error, a 404 or Core's -5, a challenge page and a body that is not
// an object; a block missing a field or both; and fields in another form,
// a string, a negative and a fraction.
// Mutants killed: a second call's failure making the whole read
// unreachable; addInts copying a field as it is (the string is kept);
// intValue taking a negative.
func TestBitcoinReads_ASecondCallWithNoAnswer(t *testing.T) {
	status := func(code int) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "busy", code) }
	}
	hashOnly := map[string]any{"hash": blockHash}
	for name, tc := range map[string]struct {
		block  http.HandlerFunc
		header string
		want   map[string]any
	}{
		"a 503, a Core error":     {status(503), rpcError(-28), hashOnly},
		"a 404, Core's -5":        {status(404), rpcError(-5), hashOnly},
		"a challenge page":        {text("<html>Just a moment...</html>"), "500 <html>Just a moment...</html>", hashOnly},
		"not an object":           {text("[1]"), result("[1]"), hashOnly},
		"no count":                {text(`{"mediantime":1790387220}`), result(`{"mediantime":1790387220}`), map[string]any{"hash": blockHash, "median_time_past": num(1790387220)}},
		"a count string":          {text(`{"tx_count":"5","mediantime":1790387220}`), result(`{"nTx":"5","mediantime":1790387220}`), map[string]any{"hash": blockHash, "median_time_past": num(1790387220)}},
		"a negative time":         {text(`{"tx_count":5,"mediantime":-1}`), result(`{"nTx":5,"mediantime":-1}`), map[string]any{"hash": blockHash, "transaction_count": num(5)}},
		"a fractional count":      {text(`{"tx_count":5.5,"mediantime":1790387220}`), result(`{"nTx":5.5,"mediantime":1790387220}`), map[string]any{"hash": blockHash, "median_time_past": num(1790387220)}},
		"neither field, a header": {text(`{"height":232}`), result(`{"height":232}`), hashOnly},
	} {
		es, _ := recording(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/block-height/232" {
				_, _ = w.Write([]byte(blockHash))
				return
			}
			tc.block(w, r)
		})
		node, _ := recording(t, core(map[string]string{"getblockhash": result(`"` + blockHash + `"`), "getblockheader": tc.header}))
		want := Answer{Outcome: OutcomeFound, Value: tc.want}
		for provider, base := range map[Provider]string{Esplora: es.URL, BitcoinCore: node.URL} {
			if got := Read(context.Background(), reads(), provider, base, coreCreds, heightRead(232)); !reflect.DeepEqual(got, want) {
				t.Errorf("%s, %s: %+v, want %+v", provider, name, got, want)
			}
		}
	}
}

// A Bitcoin Core node that echoes the Authorization header it was sent,
// into a malformed status line, an error's message, getblockcount's
// result, getblockheader's fields or the call's id, and one that echoes it
// into getblockhash's result or its error's message, with the code -1 or
// -8, gives answers whose reason and value hold neither the planted user,
// the planted password nor their Basic encoding, nor the node's URL, each
// read its own exact answer in its own count of calls, each call with the
// credentials: the status line's tip and block unreachable, with no
// answer, in one call each; a node echoing after getblockhash answers
// gives its tip unreachable with its fixed reason in one call, and its
// block found with the hash alone in two; getblockhash's echoed result is
// unreachable, not a hash, its -1 message unreachable, an error, and its
// -8 message not found, each in one call. Esplora is sent no Authorization
// at all, its tip and its block found, in three requests. Mutants killed:
// coreCall quoting the error's message in its reason; addInts copying a
// field in another form into the value (the echoed header in
// median_time_past); fromRead quoting the transport's error (the status
// line's echo is printed); CoreBlockAtHeight quoting getblockhash's
// result in its reason (the Basic encoding is printed); source.Read
// answering a Bitcoin read before making any request (no read has its
// answer or its calls).
func TestBitcoinReads_NothingANodeEchoesReachesAReasonOrValue(t *testing.T) {
	afterHash := func(rewrite func(auth string) string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var call coreCallOf
			_ = json.NewDecoder(r.Body).Decode(&call)
			if call.Method == "getblockhash" {
				_, _ = w.Write([]byte(result(`"` + blockHash + `"`)))
				return
			}
			_, _ = w.Write([]byte(rewrite(r.Header.Get("Authorization"))))
		}
	}
	always := func(rewrite func(auth string) string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(rewrite(r.Header.Get("Authorization"))))
		}
	}
	rpcEcho := func(code int) func(string) string {
		return func(auth string) string {
			return fmt.Sprintf(`{"result":null,"error":{"code":%d,"message":%q},"id":"truestamp-verify"}`, code, auth)
		}
	}
	unreachable := func(reason string) Answer { return Answer{Outcome: OutcomeUnreachable, Reason: reason} }
	notInt := unreachable("Bitcoin Core's result is not an integer from 0 to 2^53 - 1")
	coreError := unreachable("Bitcoin Core answered with an error")
	hashOnly := Answer{Outcome: OutcomeFound, Value: map[string]any{"hash": blockHash}}
	type read struct {
		want  Answer
		calls int
	}
	for _, tc := range []struct {
		name       string
		node       http.HandlerFunc
		tip, block read
	}{
		{"the status line", echoing(func(auth string) string { return auth + "\r\n\r\n" }), read{unreachable(reasonNoAnswer), 1}, read{unreachable(reasonNoAnswer), 1}},
		{"error.message", afterHash(rpcEcho(-1)), read{coreError, 1}, read{hashOnly, 2}},
		{"the -8 message", afterHash(rpcEcho(-8)), read{coreError, 1}, read{hashOnly, 2}},
		{"result", afterHash(func(auth string) string { return result(fmt.Sprintf("%q", auth)) }), read{notInt, 1}, read{hashOnly, 2}},
		{"the fields", afterHash(func(auth string) string {
			return result(fmt.Sprintf(`{"mediantime":%q,"nTx":%q,"hash":%q}`, auth, auth, auth))
		}), read{notInt, 1}, read{hashOnly, 2}},
		{"the id", afterHash(func(auth string) string { return fmt.Sprintf(`{"result":233,"error":null,"id":%q}`, auth) }),
			read{unreachable("Bitcoin Core's answer is not to this call"), 1}, read{hashOnly, 2}},
		{"getblockhash's result", always(func(auth string) string { return result(fmt.Sprintf("%q", auth)) }),
			read{notInt, 1}, read{unreachable("Bitcoin Core's result is not the 64 lowercase hex digits of a block hash"), 1}},
		{"getblockhash's -1 message", always(rpcEcho(-1)), read{coreError, 1}, read{coreError, 1}},
		{"getblockhash's -8 message", always(rpcEcho(-8)), read{coreError, 1}, read{Answer{Outcome: OutcomeNotFound}, 1}},
	} {
		for _, r := range []struct {
			read map[string]any
			want read
		}{{tipRead, tc.tip}, {heightRead(232), tc.block}} {
			srv, seen := recording(t, tc.node)
			got := Read(context.Background(), reads(), BitcoinCore, srv.URL, coreCreds, r.read)
			printed, _ := json.Marshal(got)
			for _, planted := range []string{coreUser, corePasswd, coreBasic, strings.TrimPrefix(srv.URL, "http://")} {
				if strings.Contains(string(printed)+fmt.Sprintf("%+v", got), planted) {
					t.Errorf("%s, %s: %q is printed: %s", tc.name, r.read["kind"], planted, printed)
				}
			}
			if !reflect.DeepEqual(got, r.want.want) || len(seen()) != r.want.calls {
				t.Errorf("%s, %s: %+v after %d calls, want %+v after %d", tc.name, r.read["kind"], got, len(seen()), r.want.want, r.want.calls)
			}
			for _, call := range seen() {
				if call.auth != coreBasic {
					t.Errorf("%s: a call without the credentials: %+v", tc.name, call)
				}
			}
		}
	}
	es, esSeen := recording(t, esplora("233", map[string]string{"232": blockHash}, map[string]string{blockHash: `{"tx_count":5}`}))
	esTip := Read(context.Background(), reads(), Esplora, es.URL, coreCreds, tipRead)
	esBlock := Read(context.Background(), reads(), Esplora, es.URL, coreCreds, heightRead(232))
	if want := (Answer{Outcome: OutcomeFound, Value: map[string]any{"height": num(233)}}); !reflect.DeepEqual(esTip, want) {
		t.Errorf("Esplora's tip: %+v, want %+v", esTip, want)
	}
	if want := (Answer{Outcome: OutcomeFound, Value: map[string]any{"hash": blockHash, "transaction_count": num(5)}}); !reflect.DeepEqual(esBlock, want) {
		t.Errorf("Esplora's block: %+v, want %+v", esBlock, want)
	}
	if len(esSeen()) != 3 {
		t.Errorf("Esplora saw %d requests, want 3", len(esSeen()))
	}
	for _, r := range esSeen() {
		if r.auth != "" {
			t.Errorf("Esplora was sent %q", r.auth)
		}
	}
}

// A block read is made to no server when its height is not an INT (a
// negative, a fraction, a string, none), nor a block by its hash whose
// hash is not a HEX64 (uppercase, short, a number, with a path, none), nor
// a Bitcoin read of a provider that does not serve it (Horizon): each is
// unreachable, with a fixed reason. Mutants killed: intParam taking a
// negative; hashParam without its HEX64 check (an uppercase hash, or one
// with a path, is requested); Read sending a Bitcoin read to Horizon.
func TestBitcoinReads_NothingIsReadForARefusedParameter(t *testing.T) {
	srv, seen := recording(t, text(`{}`))
	for name, tc := range map[string]struct {
		provider Provider
		read     map[string]any
	}{
		"a negative height, Esplora":    {Esplora, heightRead(-1)},
		"a fractional height, Esplora":  {Esplora, heightRead(1.5)},
		"a height string, Esplora":      {Esplora, map[string]any{"kind": KindBitcoinBlockAtHeight, "height": "5"}},
		"no height, Esplora":            {Esplora, map[string]any{"kind": KindBitcoinBlockAtHeight}},
		"a negative height, Core":       {BitcoinCore, heightRead(-1)},
		"no height, Core":               {BitcoinCore, map[string]any{"kind": KindBitcoinBlockAtHeight}},
		"a Bitcoin kind, Horizon":       {Horizon, tipRead},
		"a Bitcoin block kind, Horizon": {Horizon, heightRead(1)},
		"a Stellar kind, Bitcoin Core":  {BitcoinCore, map[string]any{"kind": KindStellarRoot}},
		"an uppercase hash, Esplora":    {Esplora, map[string]any{"kind": KindBitcoinBlock, "hash": strings.ToUpper(blockHash)}},
		"a short hash, Esplora":         {Esplora, map[string]any{"kind": KindBitcoinBlock, "hash": blockHash[1:]}},
		"a hash with a path, Esplora":   {Esplora, map[string]any{"kind": KindBitcoinBlock, "hash": blockHash[:62] + "/x"}},
		"a hash number, Core":           {BitcoinCore, map[string]any{"kind": KindBitcoinBlock, "hash": num(5)}},
		"no hash, Core":                 {BitcoinCore, map[string]any{"kind": KindBitcoinBlock}},
		"the block by hash, Horizon":    {Horizon, map[string]any{"kind": KindBitcoinBlock, "hash": blockHash}},
	} {
		got := Read(context.Background(), reads(), tc.provider, srv.URL, coreCreds, tc.read)
		if got.Outcome != OutcomeUnreachable || got.Reason == "" || got.Value != nil {
			t.Errorf("%s: %+v", name, got)
		}
	}
	if len(seen()) != 0 {
		t.Fatalf("requests made: %+v", seen())
	}
}

// The control of the chain reads on http.DefaultTransport: each Bitcoin
// read of CHAIN and of SOURCES, each Stellar one and the NIST beacon's
// pulse, through a SingleRead built with no
// transport of its own, of a host under .invalid, is refused in process,
// once, and is unreachable. Mutants killed: TestMain without the
// DefaultTransport swap (the test stops before any read); Read reading
// through a transport of its own (no read is refused).
func TestMain_RefusesNetworkOnEveryChainRead(t *testing.T) {
	if _, ok := http.DefaultTransport.(loopbackOnly); !ok {
		t.Fatalf("http.DefaultTransport is %T, not the loopback-only refusal", http.DefaultTransport)
	}
	for _, tc := range []struct {
		provider Provider
		read     map[string]any
	}{
		{Esplora, tipRead}, {Esplora, heightRead(232)}, {BitcoinCore, tipRead}, {BitcoinCore, heightRead(232)},
		{Esplora, hashRead}, {BitcoinCore, hashRead}, {NISTBeacon, pulseRead},
		{Horizon, map[string]any{"kind": KindStellarRoot}},
		{Horizon, map[string]any{"kind": KindStellarTransaction, "hash": txHash}},
		{Horizon, map[string]any{"kind": KindStellarLedger, "sequence": num(5)}},
	} {
		before := refusals.Load()
		got := Read(context.Background(), reads(), tc.provider, "https://"+string(tc.provider)+".invalid", coreCreds, tc.read)
		if got.Outcome != OutcomeUnreachable || refusals.Load()-before != 1 {
			t.Fatalf("%s %s: %+v, %d refused; want unreachable after one refusal", tc.provider, tc.read["kind"], got, refusals.Load()-before)
		}
	}
}

// A Bitcoin read ends at its SingleRead's deadline when the source never
// answers: with a 300 ms deadline each of Esplora's and Bitcoin Core's
// tip and block reads is unreachable, naming the deadline, well before a
// 5 s guard. Mutant killed: unreachableFrom giving a reason of its own in
// place of fromRead's (the reason is not the deadline's fixed text).
func TestBitcoinReads_AReadEndsAtItsDeadline(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)
	for _, provider := range []Provider{Esplora, BitcoinCore} {
		for _, read := range []map[string]any{tipRead, heightRead(232)} {
			done := make(chan Answer, 1)
			start := time.Now()
			go func() {
				done <- Read(context.Background(), httpclient.NewSingleRead(nil, 300*time.Millisecond), provider, srv.URL, coreCreds, read)
			}()
			select {
			case got := <-done:
				if got.Outcome != OutcomeUnreachable || got.Reason != reasonDeadline || time.Since(start) > 3*time.Second {
					t.Fatalf("%s %s: %+v after %v; want the deadline's reason", provider, read["kind"], got, time.Since(start))
				}
			case <-time.After(5 * time.Second):
				t.Fatalf("%s %s: the read was not cut at its deadline", provider, read["kind"])
			}
		}
	}
}

// The block by its hash is one call of SPEC.md section 10.4's table, and
// its found answer is in the form of a replay record: Esplora's GET
// <base>/block/<hash>, with no credential, its height, timestamp and
// mediantime as height, time and median_time_past; Bitcoin Core's
// getblockheader "<hash>", a JSON-RPC 1.0 POST with the credentials as
// HTTP Basic, its height, time and mediantime, a header off the best
// chain (confirmations -1) found like any other; every other field left
// out. A block that lacks a field, or gives one in another form, is found
// with the fields it gives as INTs, so the rule judges it (section 10.4,
// "Answers"). Mutants killed: Esplora's timestamp not renamed (the value
// holds timestamp, not time); Bitcoin Core's nTx read in place of time;
// a field copied that the read does not use (the value holds tx_count);
// EsploraBlock calling /block-height (the path is wrong and the read is
// not found); CoreBlock giving getblockheader the hash unquoted (its
// params are not a string); Read giving a by-hash read the block at a
// height (two calls). A password that holds %, /, #, ?, @ and : reaches
// Bitcoin Core as HTTP Basic alone, in one call that finds the block, and
// no request's URL carries userinfo. Mutants killed: the credentials
// written into the URL's userinfo unescaped, with strings.Replace on
// "://" (the URL no longer names the node, and the read is unreachable);
// the credentials set as url.UserPassword userinfo, which net/http sends
// as the same Basic header (the request's URL carries userinfo).
func TestBitcoinReads_TheBlockByItsHash(t *testing.T) {
	block := `{"id":"` + blockHash + `","height":232,"timestamp":1790387300,"tx_count":5,"mediantime":1790387220,"size":900}`
	es, esSeen := recording(t, esplora("", nil, map[string]string{blockHash: block}))
	header := result(`{"hash":"` + blockHash + `","height":232,"time":1790387300,"nTx":5,"mediantime":1790387220,"confirmations":-1}`)
	node, nodeSeen := recording(t, core(map[string]string{"getblockheader": header}))
	full := Answer{Outcome: OutcomeFound, Value: map[string]any{"height": num(232), "median_time_past": num(1790387220), "time": num(1790387300)}}
	for provider, base := range map[Provider]string{Esplora: es.URL + "/api/", BitcoinCore: node.URL} {
		if got := Read(context.Background(), reads(), provider, base, coreCreds, hashRead); !reflect.DeepEqual(got, full) {
			t.Errorf("%s: %+v, want %+v", provider, got, full)
		}
	}
	if got := esSeen(); !reflect.DeepEqual(got, []request{{"GET", "/api/block/" + blockHash, "", "", ""}}) {
		t.Errorf("Esplora saw %+v", got)
	}
	call := request{"POST", "/", "application/json", coreBasic, `{"jsonrpc":"1.0","id":"truestamp-verify","method":"getblockheader","params":["` + blockHash + `"]}`}
	if got := nodeSeen(); !reflect.DeepEqual(got, []request{call}) {
		t.Errorf("Bitcoin Core saw %+v", got)
	}
	for name, tc := range map[string]struct {
		block, header string
		want          map[string]any
	}{
		"no median":         {`{"height":232,"timestamp":1790387300}`, `{"height":232,"time":1790387300}`, map[string]any{"height": num(232), "time": num(1790387300)}},
		"a time string":     {`{"height":232,"timestamp":"1790387300","mediantime":1790387220}`, `{"height":232,"time":"1790387300","mediantime":1790387220}`, map[string]any{"height": num(232), "median_time_past": num(1790387220)}},
		"a negative height": {`{"height":-1,"timestamp":1790387300,"mediantime":1790387220}`, `{"height":-1,"time":1790387300,"mediantime":1790387220}`, map[string]any{"median_time_past": num(1790387220), "time": num(1790387300)}},
		"no field":          {`{}`, `{}`, map[string]any{}},
	} {
		es, _ := recording(t, esplora("", nil, map[string]string{blockHash: tc.block}))
		node, _ := recording(t, core(map[string]string{"getblockheader": result(tc.header)}))
		want := Answer{Outcome: OutcomeFound, Value: tc.want}
		for provider, base := range map[Provider]string{Esplora: es.URL, BitcoinCore: node.URL} {
			if got := Read(context.Background(), reads(), provider, base, coreCreds, hashRead); !reflect.DeepEqual(got, want) {
				t.Errorf("%s, %s: %+v, want %+v", provider, name, got, want)
			}
		}
	}
	const awkward = "p%41ss/w#o?r@d:77c1"
	awkwardNode, awkwardSeen := recording(t, core(map[string]string{"getblockheader": header}))
	var userinfo []bool
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		userinfo = append(userinfo, r.URL.User != nil)
		return http.DefaultTransport.RoundTrip(r)
	})
	got := Read(context.Background(), httpclient.NewSingleRead(transport, 5*time.Second), BitcoinCore, awkwardNode.URL, NewCredentials(coreUser, awkward), hashRead)
	if !reflect.DeepEqual(got, full) {
		t.Errorf("a password URLs mangle: %+v, want %+v", got, full)
	}
	basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(coreUser+":"+awkward))
	if seen := awkwardSeen(); len(seen) != 1 || seen[0].auth != basic {
		t.Errorf("a password URLs mangle: Bitcoin Core saw %+v, want one call with Authorization %q", seen, basic)
	}
	if !reflect.DeepEqual(userinfo, []bool{false}) {
		t.Errorf("a password URLs mangle: whether each request's URL carried userinfo: %v, want [false]", userinfo)
	}
}

// roundTripFunc is an [http.RoundTripper] that is a function.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// A block the source does not hold by that hash is not found, with no
// value, in one call: Esplora's 404, and Bitcoin Core's error -5 (no such
// block), or -8, to getblockheader. Every other answer is unreachable, with
// a fixed reason that quotes no body, in one call: a 3xx, a 429, a 503 and
// a challenge page at 403 and at 200 from each provider; a body over
// BitcoinOctets; Esplora's answer that is not a JSON object, or that holds
// a number the strict reader refuses (2^53, in a member the read does not
// use, which SPEC.md section 10.1's profile refuses); Bitcoin Core's
// result that is not an object and any other error of it. Mutants killed:
// EsploraBlock reading a 404 as unreachable; EsploraBlock reading any
// status as not found (a 503 is not found); CoreBlock taking a result that
// is not an object as a block (it is found, empty); EsploraBlock taking a
// body that parses but is not an object (the array is found, empty).
func TestBitcoinReads_ABlockByItsHashNotFoundAndUnreachable(t *testing.T) {
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
	for name, tc := range map[string]struct {
		provider Provider
		answer   http.HandlerFunc
		want     Outcome
	}{
		"Esplora's 404":               {Esplora, esplora("", nil, nil), OutcomeNotFound},
		"Core's -5":                   {BitcoinCore, core(map[string]string{"getblockheader": rpcError(-5)}), OutcomeNotFound},
		"Core's -8":                   {BitcoinCore, core(map[string]string{"getblockheader": rpcError(-8)}), OutcomeNotFound},
		"Core's -28":                  {BitcoinCore, core(map[string]string{"getblockheader": rpcError(-28)}), OutcomeUnreachable},
		"Core's result a string":      {BitcoinCore, core(map[string]string{"getblockheader": result(`"quoted-body"`)}), OutcomeUnreachable},
		"Esplora's array":             {Esplora, esplora("", nil, map[string]string{blockHash: `[232]`}), OutcomeUnreachable},
		"Esplora's 2^53":              {Esplora, esplora("", nil, map[string]string{blockHash: `{"height":232,"timestamp":1790387300,"mediantime":1790387220,"difficulty":9007199254740992}`}), OutcomeUnreachable},
		"Esplora's 3xx":               {Esplora, redirect, OutcomeUnreachable},
		"Core's 3xx":                  {BitcoinCore, redirect, OutcomeUnreachable},
		"Esplora's 429":               {Esplora, status(429), OutcomeUnreachable},
		"Core's 429":                  {BitcoinCore, status(429), OutcomeUnreachable},
		"Esplora's 503":               {Esplora, status(503), OutcomeUnreachable},
		"Core's 503":                  {BitcoinCore, status(503), OutcomeUnreachable},
		"Esplora's 403 challenge":     {Esplora, challenge(403), OutcomeUnreachable},
		"Core's 200 challenge":        {BitcoinCore, challenge(200), OutcomeUnreachable},
		"Esplora's 200 challenge":     {Esplora, challenge(200), OutcomeUnreachable},
		"Esplora's body over a bound": {Esplora, text(`{}` + strings.Repeat(" ", BitcoinOctets)), OutcomeUnreachable},
	} {
		srv, seen := recording(t, tc.answer)
		got := Read(context.Background(), reads(), tc.provider, srv.URL, coreCreds, hashRead)
		if got.Outcome != tc.want || got.Value != nil || strings.Contains(got.Reason, "quoted-body") || (tc.want == OutcomeUnreachable) != (got.Reason != "") || len(seen()) != 1 {
			t.Errorf("%s: %+v after %d calls; want %s, no value, no body quoted", name, got, len(seen()), tc.want)
		}
	}
}

// A Bitcoin Core node that echoes the Authorization header it was sent
// into getblockheader's fields, its error's message or the call's id,
// gives an answer to the block by its hash whose reason and value hold
// neither the planted user, the planted password nor their Basic
// encoding, each read in one call with the credentials: the fields' echo
// found with no field, the message's unreachable with Core's fixed reason
// and the id's unreachable as an answer to another call. Mutant killed:
// blockByHash copying a field as it is (the echo is in the value).
func TestBitcoinReads_NothingANodeEchoesReachesABlockByItsHash(t *testing.T) {
	echo := func(rewrite func(auth string) string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(rewrite(r.Header.Get("Authorization"))))
		}
	}
	for name, tc := range map[string]struct {
		node http.HandlerFunc
		want Answer
	}{
		"the fields": {echo(func(auth string) string {
			return result(fmt.Sprintf(`{"height":%q,"time":%q,"mediantime":%q}`, auth, auth, auth))
		}), Answer{Outcome: OutcomeFound, Value: map[string]any{}}},
		"error.message": {echo(func(auth string) string {
			return fmt.Sprintf(`{"result":null,"error":{"code":-1,"message":%q},"id":"truestamp-verify"}`, auth)
		}), Answer{Outcome: OutcomeUnreachable, Reason: "Bitcoin Core answered with an error"}},
		"the id": {echo(func(auth string) string { return fmt.Sprintf(`{"result":{},"error":null,"id":%q}`, auth) }),
			Answer{Outcome: OutcomeUnreachable, Reason: "Bitcoin Core's answer is not to this call"}},
	} {
		srv, seen := recording(t, tc.node)
		got := Read(context.Background(), reads(), BitcoinCore, srv.URL, coreCreds, hashRead)
		printed, _ := json.Marshal(got)
		for _, planted := range []string{coreUser, corePasswd, coreBasic} {
			if strings.Contains(string(printed)+fmt.Sprintf("%+v", got), planted) {
				t.Errorf("%s: %q is printed: %s", name, planted, printed)
			}
		}
		if !reflect.DeepEqual(got, tc.want) || len(seen()) != 1 || seen()[0].auth != coreBasic {
			t.Errorf("%s: %+v after %+v, want %+v", name, got, seen(), tc.want)
		}
	}
}
