// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package source

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// Horizon's reads of CHAIN (SPEC.md section 10.4, "Stellar commitment
// entries" and "Common sources"), against httptest servers on loopback.

const (
	txHash  = "860d4085c0af0b5f8cadff2bb0647cf75bbd16901817684dc4e180c200c70681"
	memoHex = "488e6583a192071810724dd392b551c4ca119ca6688deb9034683eb42eba1364"
)

// memoBase64 is memoHex's 32 bytes as Horizon writes a hash memo, the
// padded standard base64, which holds a "+".
var memoBase64 = func() string {
	raw, _ := hex.DecodeString(memoHex)
	return base64.StdEncoding.EncodeToString(raw)
}()

// num is n as the strict reader gives it.
func num(n float64) strictjson.Number { return strictjson.Number(n) }

// Each read makes its call of SPEC.md section 10.4's table, once, a GET
// with no credential, and gives its found answer in the form of a replay
// record: the root's history_latest_ledger and history_elder_ledger as
// latest and elder, a transaction's successful, memo_type and ledger as
// they are and its hash memo's base64 as the hex of its 32 bytes, and a
// ledger's closed_at and hash; every other member Horizon sends is left
// out, and a base with a path and a trailing slash is joined once.
// Mutants killed: the root's names not renamed (the value holds
// history_latest_ledger); the memo copied as Horizon writes it (base64 in
// the value); the memo kept for a memo_type of text; StellarLedger
// writing the sequence in float form (the path is /ledgers/1.000267e+06);
// a field copied that the read does not use (paging_token in the value).
func TestStellarReads_EachCallAndItsAnswer(t *testing.T) {
	root := `{"network_passphrase":"` + passphrase + `","history_latest_ledger":1000396,"history_elder_ledger":1000208,"core_latest_ledger":1000397}`
	tx := `{"hash":"` + txHash + `","successful":true,"ledger":1000255,"memo_type":"hash","memo":"` + memoBase64 + `","paging_token":"1","envelope_xdr":"AAAA"}`
	ledger := `{"sequence":1000267,"closed_at":"2026-09-26T01:52:15Z","hash":"0e8e2010d79bb6b859b5884ee90f83092d7320e18b12740119c7b734dcab4ea7","tx_set_operation_count":3}`
	text := `{"successful":false,"ledger":7,"memo_type":"text","memo":"` + memoBase64 + `"}`
	for _, tc := range []struct {
		name, body, path string
		read             map[string]any
		want             Answer
	}{
		{"the root", root, "/v1/", map[string]any{"kind": KindStellarRoot, "network": "testnet"},
			Answer{Outcome: OutcomeFound, Value: map[string]any{"latest": num(1000396), "elder": num(1000208)}}},
		{"a transaction", tx, "/v1/transactions/" + txHash, map[string]any{"kind": KindStellarTransaction, "network": "testnet", "hash": txHash},
			Answer{Outcome: OutcomeFound, Value: map[string]any{"successful": true, "ledger": num(1000255), "memo_type": "hash", "memo": memoHex}}},
		{"a text memo", text, "/v1/transactions/" + txHash, map[string]any{"kind": KindStellarTransaction, "network": "public", "hash": txHash},
			Answer{Outcome: OutcomeFound, Value: map[string]any{"successful": false, "ledger": num(7), "memo_type": "text"}}},
		{"a ledger", ledger, "/v1/ledgers/1000267", map[string]any{"kind": KindStellarLedger, "network": "testnet", "sequence": num(1000267)},
			Answer{Outcome: OutcomeFound, Value: map[string]any{"closed_at": "2026-09-26T01:52:15Z", "hash": "0e8e2010d79bb6b859b5884ee90f83092d7320e18b12740119c7b734dcab4ea7"}}},
	} {
		srv, seen := recording(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tc.body)) })
		got := Read(context.Background(), reads(), Horizon, srv.URL+"/v1/", tc.read)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
		if s := seen(); len(s) != 1 || s[0] != (request{"GET", tc.path, "", "", ""}) {
			t.Errorf("%s: the server saw %+v, want one GET of %s", tc.name, s, tc.path)
		}
	}
}

// A found answer holds only the fields Horizon gave, as it gave them: a
// transaction with no ledger has none, and one whose successful is a
// string holds that string, for the rule to judge (SPEC.md section 10.4,
// "Answers"); a hash memo that is not the padded standard base64 of 32
// bytes, 31 bytes, unpadded, with a line break, with its unused bits set,
// in URL-safe base64, as hex or not a string, is left out. Mutants killed:
// memoBytes without its pattern (the memo with a line break decodes, the
// decoder skipping it); the memo copied only for a memo_type of hash (a
// return memo's hex absent).
func TestStellarReads_AFoundAnswerHoldsWhatHorizonGave(t *testing.T) {
	memo := func(m string) string {
		return `{"successful":true,"ledger":5,"memo_type":"hash","memo":` + m + `}`
	}
	raw, _ := hex.DecodeString(memoHex)
	thirtyOne := base64.StdEncoding.EncodeToString(raw[:31])
	lastBitSet := memoBase64[:42] + "B="
	for _, tc := range []struct {
		name, body string
		want       map[string]any
	}{
		{"no ledger", `{"successful":true,"memo_type":"hash","memo":"` + memoBase64 + `"}`, map[string]any{"successful": true, "memo_type": "hash", "memo": memoHex}},
		{"successful a string", `{"successful":"true","ledger":5,"memo_type":"none"}`, map[string]any{"successful": "true", "ledger": num(5), "memo_type": "none"}},
		{"a return memo", `{"successful":true,"ledger":5,"memo_type":"return","memo":"` + memoBase64 + `"}`, map[string]any{"successful": true, "ledger": num(5), "memo_type": "return", "memo": memoHex}},
		{"31 bytes", memo(`"` + thirtyOne + `"`), map[string]any{"successful": true, "ledger": num(5), "memo_type": "hash"}},
		{"unpadded", memo(`"` + strings.TrimSuffix(memoBase64, "=") + `"`), map[string]any{"successful": true, "ledger": num(5), "memo_type": "hash"}},
		{"a line break", memo(`"` + memoBase64[:20] + `\n` + memoBase64[20:] + `"`), map[string]any{"successful": true, "ledger": num(5), "memo_type": "hash"}},
		{"unused bits set", memo(`"` + lastBitSet + `"`), map[string]any{"successful": true, "ledger": num(5), "memo_type": "hash"}},
		{"URL-safe", memo(`"` + strings.NewReplacer("+", "-", "/", "_").Replace(memoBase64) + `"`), map[string]any{"successful": true, "ledger": num(5), "memo_type": "hash"}},
		{"hex", memo(`"` + memoHex + `"`), map[string]any{"successful": true, "ledger": num(5), "memo_type": "hash"}},
		{"a number", memo(`5`), map[string]any{"successful": true, "ledger": num(5), "memo_type": "hash"}},
	} {
		srv, _ := recording(t, text(tc.body))
		got := StellarTransaction(context.Background(), reads(), srv.URL, txHash)
		if got.Outcome != OutcomeFound || !reflect.DeepEqual(got.Value, tc.want) {
			t.Errorf("%s: %+v, want found %v", tc.name, got, tc.want)
		}
	}
	if !strings.Contains(memoBase64, "+") {
		t.Fatal("the URL-safe fixture equals the standard one")
	}
}

// A ledger or root answer that lacks fields is found with exactly the
// fields it holds, for the rule to judge on the fields it reads (SPEC.md
// section 10.4, "Answers"): a ledger with closed_at alone, one with hash
// alone and one with neither; a root with history_latest_ledger and
// history_elder_ledger alone, and one with neither. Mutants killed:
// StellarLedger refusing an answer without hash (the ledger with closed_at
// alone is unreachable); StellarLedger refusing one without closed_at (the
// ledger with hash alone is unreachable); StellarRoot refusing one without
// core_latest_ledger (the root with latest and elder alone is
// unreachable).
func TestStellarReads_APartialAnswerIsFound(t *testing.T) {
	ledgerRead := map[string]any{"kind": KindStellarLedger, "network": "testnet", "sequence": num(1000255)}
	rootRead := map[string]any{"kind": KindStellarRoot, "network": "testnet"}
	for _, tc := range []struct {
		name, body string
		read       map[string]any
		want       map[string]any
	}{
		{"a ledger with closed_at alone", `{"closed_at":"2026-09-26T01:52:15Z"}`, ledgerRead, map[string]any{"closed_at": "2026-09-26T01:52:15Z"}},
		{"a ledger with hash alone", `{"hash":"` + memoHex + `"}`, ledgerRead, map[string]any{"hash": memoHex}},
		{"a ledger with neither", `{"sequence":1000255}`, ledgerRead, map[string]any{}},
		{"a root with latest and elder alone", `{"history_latest_ledger":9,"history_elder_ledger":3}`, rootRead, map[string]any{"latest": num(9), "elder": num(3)}},
		{"a root with neither", `{"core_latest_ledger":10}`, rootRead, map[string]any{}},
	} {
		srv, seen := recording(t, text(tc.body))
		got := Read(context.Background(), reads(), Horizon, srv.URL, tc.read)
		if got.Outcome != OutcomeFound || !reflect.DeepEqual(got.Value, tc.want) || got.Reason != "" || len(seen()) != 1 {
			t.Errorf("%s: %+v after %d requests, want found %v", tc.name, got, len(seen()), tc.want)
		}
	}
}

// A 404 to a transaction or a ledger is not found, and to the root, which
// is never not found (SPEC.md section 13.2), unreachable; a 3xx, a 429, a
// 503, a challenge page (a 403 or a 200 that is HTML), an answer that is
// not a JSON object and one over its bound are unreachable, with a reason
// that quotes no body, each after one request; an answer of exactly the
// bound is read. Mutants killed: a 404 to the root not found; any status
// read as not found (a 503 is not found); a not-found answer given a value
// (a 404 with a body holds one); TransactionOctets passed as limit - 1.
func TestStellarReads_NotFoundAndUnreachable(t *testing.T) {
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
	redirect := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	}
	reads404 := map[string]Outcome{KindStellarRoot: OutcomeUnreachable, KindStellarTransaction: OutcomeNotFound, KindStellarLedger: OutcomeNotFound}
	readOf := map[string]map[string]any{
		KindStellarRoot:        {"kind": KindStellarRoot, "network": "testnet"},
		KindStellarTransaction: {"kind": KindStellarTransaction, "network": "testnet", "hash": txHash},
		KindStellarLedger:      {"kind": KindStellarLedger, "network": "testnet", "sequence": num(5)},
	}
	for kind, read := range readOf {
		for name, tc := range map[string]struct {
			answer http.HandlerFunc
			want   Outcome
		}{
			"404":             {status(404), reads404[kind]},
			"3xx":             {redirect, OutcomeUnreachable},
			"429":             {status(429), OutcomeUnreachable},
			"503":             {status(503), OutcomeUnreachable},
			"a 403 challenge": {challenge(403), OutcomeUnreachable},
			"a 200 challenge": {challenge(200), OutcomeUnreachable},
			"an array":        {text(`[1]`), OutcomeUnreachable},
			"a repeated name": {text(`{"ledger":1,"ledger":2}`), OutcomeUnreachable},
			"over the bound":  {text(`{}` + strings.Repeat(" ", TransactionOctets)), OutcomeUnreachable},
			"a 404 with a value": {func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(404)
				_, _ = w.Write([]byte(`{"ledger":5}`))
			}, reads404[kind]},
		} {
			srv, seen := recording(t, tc.answer)
			got := Read(context.Background(), reads(), Horizon, srv.URL, read)
			if got.Outcome != tc.want || got.Value != nil || strings.Contains(got.Reason, "quoted-body") || (tc.want == OutcomeUnreachable) != (got.Reason != "") || len(seen()) != 1 {
				t.Errorf("%s, %s: %+v after %d requests; want %s, no value, no body quoted", kind, name, got, len(seen()), tc.want)
			}
		}
	}
	exact := `{"ledger":5}`
	srv, _ := recording(t, text(exact+strings.Repeat(" ", TransactionOctets-len(exact))))
	if got := StellarTransaction(context.Background(), reads(), srv.URL, txHash); got.Outcome != OutcomeFound || got.Value["ledger"] != num(5) {
		t.Fatalf("an answer of exactly %d octets: %+v", TransactionOctets, got)
	}
}

// A read is made to no server when its parameter is not in its class, a
// transaction hash that is not a HEX64 or a ledger sequence that is not an
// INT, or when its kind is not one this provider serves: each is
// unreachable, with a fixed reason. Mutants killed: StellarTransaction
// without its hex64 check (an uppercase hash, or one with a slash, is
// requested); StellarLedger taking a negative sequence; Read sending a
// Stellar kind to an Esplora server.
func TestStellarReads_NothingIsReadForARefusedParameter(t *testing.T) {
	srv, seen := recording(t, text(`{}`))
	for name, tc := range map[string]struct {
		provider Provider
		read     map[string]any
	}{
		"an uppercase hash":       {Horizon, map[string]any{"kind": KindStellarTransaction, "hash": strings.ToUpper(txHash)}},
		"a hash with a path":      {Horizon, map[string]any{"kind": KindStellarTransaction, "hash": txHash[:62] + "/x"}},
		"no hash":                 {Horizon, map[string]any{"kind": KindStellarTransaction}},
		"a negative sequence":     {Horizon, map[string]any{"kind": KindStellarLedger, "sequence": num(-1)}},
		"a fractional sequence":   {Horizon, map[string]any{"kind": KindStellarLedger, "sequence": num(1.5)}},
		"a sequence string":       {Horizon, map[string]any{"kind": KindStellarLedger, "sequence": "5"}},
		"an unknown kind":         {Horizon, map[string]any{"kind": "bitcoin_tip"}},
		"a Stellar kind, Esplora": {Esplora, map[string]any{"kind": KindStellarRoot}},
	} {
		got := Read(context.Background(), reads(), tc.provider, srv.URL, tc.read)
		if got.Outcome != OutcomeUnreachable || got.Reason == "" || got.Value != nil {
			t.Errorf("%s: %+v", name, got)
		}
	}
	if len(seen()) != 0 {
		t.Fatalf("requests made: %+v", seen())
	}
}

// A read ends at its SingleRead's deadline when Horizon never answers:
// with a 300 ms deadline it is unreachable, naming the deadline, well
// before a 5 s guard. Mutant killed: getObject giving the transport's
// error text as the reason (the reason is not the deadline's fixed text).
func TestStellarReads_AReadEndsAtItsDeadline(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)
	done := make(chan Answer, 1)
	start := time.Now()
	go func() {
		done <- StellarTransaction(context.Background(), httpclient.NewSingleRead(nil, 300*time.Millisecond), srv.URL, txHash)
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
