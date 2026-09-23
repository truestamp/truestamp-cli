// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/bitcoin"
	"github.com/truestamp/truestamp-cli/internal/external"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// horizonAnswer is what the stubbed Horizon returns for one transaction.
type horizonAnswer struct {
	memoHex   string
	ledger    int
	createdAt string
}

// withStubbedHorizon points both Horizon endpoints at a stub that answers
// a transaction lookup for each hash in answers and returns 503 for every
// other request, which the verifier grades as unavailable. Call it after
// withStubbedSources, whose cleanup restores the endpoints.
func withStubbedHorizon(t *testing.T, answers map[string]horizonAnswer) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hash, ok := strings.CutPrefix(r.URL.Path, "/transactions/")
		a, known := answers[hash]
		if !ok || !known {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		memo, err := hex.DecodeString(a.memoHex)
		if err != nil {
			t.Errorf("stub memo %q: %v", a.memoHex, err)
		}
		body := map[string]any{
			"hash":      hash,
			"memo_type": "hash",
			"memo":      base64.StdEncoding.EncodeToString(memo),
			"ledger":    a.ledger,
		}
		if a.createdAt != "" {
			body["created_at"] = a.createdAt
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	external.HorizonPublicURL, external.HorizonTestnetURL = srv.URL, srv.URL
}

// assertNoCarriedTime checks that no row presents a Submitted before edge
// other than want ("" for none), and that no Submitted Before row, and no
// line of the printed report, shows any of the carried timestamps.
func assertNoCarriedTime(t *testing.T, r *Report, want string, carried ...string) {
	t.Helper()
	for _, s := range r.Steps {
		msg := strings.ToLower(s.Message)
		if strings.Contains(msg, "submitted before") && (want == "" || !strings.HasPrefix(s.Message, "Submitted before "+want+":")) {
			t.Errorf("%s row %s %q presents an edge; want %q\n%s", s.Group, s.Status, s.Message, want, render(r.Steps))
		}
	}
	printed := Render(r, false)
	for _, ts := range carried {
		for _, s := range rowsOf(r, groupSubmittedBefore) {
			if strings.Contains(s.Message, ts) {
				t.Errorf("Submitted Before row %s %q shows the carried timestamp %s", s.Status, s.Message, ts)
			}
		}
		if strings.Contains(printed, ts) {
			t.Errorf("the printed report shows the carried timestamp %s\n%s", ts, printed)
		}
	}
}

// onlySubmittedBeforeRow returns the one Submitted Before row, failing when
// there is not exactly one.
func onlySubmittedBeforeRow(t *testing.T, r *Report) Step {
	t.Helper()
	rows := rowsOf(r, groupSubmittedBefore)
	if len(rows) != 1 {
		t.Fatalf("Submitted Before rows = %d, want 1\n%s", len(rows), render(r.Steps))
	}
	return rows[0]
}

// TestSubmittedBefore_CarriedTimeIsNeverAnEdge pins E01's second half on
// the A2.1 fixture, whose Bitcoin entry carries a real txoutproof of an
// unrelated transaction and the timestamp 2001-01-01. Neither that time
// nor the Stellar entry's carried time is ever shown as the edge. With
// Horizon answering for the Stellar entry, the edge is the close time
// Horizon returned, not the entry's timestamp.
func TestSubmittedBefore_CarriedTimeIsNeverAnEdge(t *testing.T) {
	data := testfixtures.Read(t, testfixtures.TamperDir, testfixtures.BitcoinUnrelatedTxoutproof)
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	var stellar, bitcoinEntry *proof.Commitment
	for i := range bundle.Commitments {
		switch c := &bundle.Commitments[i]; c.Chain {
		case "stellar":
			stellar = c
		case "bitcoin":
			bitcoinEntry = c
		}
	}
	if stellar == nil || bitcoinEntry == nil || bundle.SigningKeyEvent == nil || len(bundle.SigningKeyEvent.Commitments) != 1 {
		t.Fatal("the fixture no longer carries one Stellar entry, one Bitcoin entry and one key event commitment")
	}
	if bitcoinEntry.Timestamp != "2001-01-01T00:00:00Z" {
		t.Fatalf("the fixture's Bitcoin timestamp is %q, want 2001-01-01T00:00:00Z", bitcoinEntry.Timestamp)
	}
	keyEvent := bundle.SigningKeyEvent.Commitments[0]
	carried := []string{bitcoinEntry.Timestamp, "2001-01-01", stellar.Timestamp, keyEvent.Timestamp}

	t.Run("offline", func(t *testing.T) {
		r := mustRun(t, data, Options{SkipExternal: true})
		row := onlySubmittedBeforeRow(t, r)
		if row.Status != StatusInfo || !strings.HasPrefix(row.Message, "not established: no chain was read in this run (offline)") {
			t.Errorf("Submitted Before = %s %q, want an offline not-established info", row.Status, row.Message)
		}
		assertNoCarriedTime(t, r, "", carried...)
	})

	t.Run("online, no Horizon answer", func(t *testing.T) {
		stub := &blockstreamStub{blockHash: unrelatedBlockHash, height: unrelatedBlockHeight}
		withStubbedSources(t, stub)
		r := mustRun(t, data, Options{})
		row := onlySubmittedBeforeRow(t, r)
		if row.Status != StatusInfo || !strings.HasPrefix(row.Message, "not established: no commitment was confirmed on its chain in this run") {
			t.Errorf("Submitted Before = %s %q, want a not-established info", row.Status, row.Message)
		}
		assertNoCarriedTime(t, r, "", carried...)
	})

	// Horizon answers for the Stellar entry at 12:00:05, seven seconds
	// before the entry's carried 12:00:12, and for the key event's own
	// commitment at an earlier time still, which must not move the edge.
	const stubbed = "2026-07-24T12:00:05Z"
	answers := map[string]horizonAnswer{
		stellar.TransactionHash:  {memoHex: stellar.EpochMerkleRoot, ledger: stellar.Ledger, createdAt: stubbed},
		keyEvent.TransactionHash: {memoHex: keyEvent.EpochMerkleRoot, ledger: keyEvent.Ledger, createdAt: "2026-07-24T11:00:01Z"},
	}

	t.Run("online, stubbed Horizon read", func(t *testing.T) {
		stub := &blockstreamStub{blockHash: unrelatedBlockHash, height: unrelatedBlockHeight}
		withStubbedSources(t, stub)
		withStubbedHorizon(t, answers)
		r := mustRun(t, data, Options{})
		if !r.Passed() {
			t.Errorf("verdict FAILED\n%s", render(r.Steps))
		}
		stellarPassed := false
		for _, s := range rowsOf(r, groupStellar) {
			if s.Status == StatusPass && strings.HasPrefix(s.Message, "Transaction "+stellar.TransactionHash+" confirmed") {
				stellarPassed = true
			}
		}
		if !stellarPassed {
			t.Fatalf("the stubbed Horizon did not confirm the Stellar entry\n%s", render(r.Steps))
		}
		assertNoPass(t, r, groupBitcoin)

		row := onlySubmittedBeforeRow(t, r)
		const wantEdge = "2026-07-24T12:00:05.000Z"
		if row.Status != StatusPass || !strings.HasPrefix(row.Message, "Submitted before "+wantEdge+": the close time of ledger 51234567, read from the public Horizon in this run") {
			t.Errorf("Submitted Before = %s %q, want a pass at the stubbed time %s", row.Status, row.Message, wantEdge)
		}
		ms, ok := isoMs(stubbed)
		if !ok || formatMs(ms) != wantEdge {
			t.Fatalf("stubbed time %s does not format as %s", stubbed, wantEdge)
		}
		assertNoCarriedTime(t, r, wantEdge, carried...)
	})

	t.Run("online, Horizon returns no time", func(t *testing.T) {
		stub := &blockstreamStub{blockHash: unrelatedBlockHash, height: unrelatedBlockHeight}
		withStubbedSources(t, stub)
		untimed := map[string]horizonAnswer{
			stellar.TransactionHash: {memoHex: stellar.EpochMerkleRoot, ledger: stellar.Ledger},
		}
		withStubbedHorizon(t, untimed)
		r := mustRun(t, data, Options{})
		row := onlySubmittedBeforeRow(t, r)
		if row.Status != StatusInfo || !strings.Contains(row.Message, "Horizon returned no usable close time") {
			t.Errorf("Submitted Before = %s %q, want a not-established info naming the missing close time\n%s", row.Status, row.Message, render(r.Steps))
		}
		assertNoCarriedTime(t, r, "", carried...)
	})
}

// TestSubmittedBefore_BitcoinGivesInfoOnly runs the staging bundle whose
// Bitcoin entry carries the whole byte chain, relabelled as mainnet (the
// signature does not cover `network`) so the stubbed chain confirms it.
// A confirmed Bitcoin commitment never yields the edge, and its carried
// timestamp is never shown in its place.
func TestSubmittedBefore_BitcoinGivesInfoOnly(t *testing.T) {
	var m map[string]any
	dec := json.NewDecoder(strings.NewReader(string(testfixtures.Read(t, testfixtures.StagingDir, testfixtures.ProdComplete))))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	entry := m["commitments"].([]any)[0].(map[string]any)
	if entry["chain"] != "bitcoin" {
		t.Fatalf("staging commitments[0] is %v, want bitcoin", entry["chain"])
	}
	entry["network"] = "mainnet"
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	headerHash, err := bitcoin.ParseBlockHash(entry["txoutproof"].(string))
	if err != nil {
		t.Fatal(err)
	}
	height, err := entry["block_height"].(json.Number).Int64()
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	stellar := bundle.Commitments[1]
	if stellar.Chain != "stellar" {
		t.Fatalf("staging commitments[1] is %s, want stellar", stellar.Chain)
	}
	carried := []string{bundle.Commitments[0].Timestamp, stellar.Timestamp}
	keyring := testfixtures.Path(testfixtures.StagingDir, testfixtures.ProdKeyring)

	bitcoinConfirmed := func(t *testing.T, r *Report) {
		t.Helper()
		want := "Block " + headerHash + " confirmed on mainnet"
		for _, s := range rowsOf(r, groupBitcoin) {
			if s.Status == StatusPass && strings.HasPrefix(s.Message, want) {
				return
			}
		}
		t.Fatalf("the Bitcoin commitment did not confirm\n%s", render(r.Steps))
	}

	t.Run("Bitcoin alone confirmed", func(t *testing.T) {
		withStubbedSources(t, &blockstreamStub{blockHash: headerHash, height: int(height)})
		r := mustRun(t, data, Options{KeyringFile: keyring})
		bitcoinConfirmed(t, r)
		row := onlySubmittedBeforeRow(t, r)
		if row.Status != StatusInfo || row.Message != "not established: "+bitcoinNoEdge+"; a commitment's carried timestamp is not signed and never stands in for this edge" {
			t.Errorf("Submitted Before = %s %q, want the Bitcoin info only", row.Status, row.Message)
		}
		assertNoCarriedTime(t, r, "", carried...)
	})

	// The Stellar read, 18:20:00, is later than the Bitcoin entry's carried
	// 18:15:06, which must not displace it.
	t.Run("Stellar and Bitcoin confirmed", func(t *testing.T) {
		withStubbedSources(t, &blockstreamStub{blockHash: headerHash, height: int(height)})
		withStubbedHorizon(t, map[string]horizonAnswer{
			stellar.TransactionHash: {memoHex: stellar.EpochMerkleRoot, ledger: stellar.Ledger, createdAt: "2026-09-03T18:20:00Z"},
		})
		r := mustRun(t, data, Options{KeyringFile: keyring})
		bitcoinConfirmed(t, r)
		const wantEdge = "2026-09-03T18:20:00.000Z"
		rows := rowsOf(r, groupSubmittedBefore)
		if len(rows) != 2 ||
			rows[0].Status != StatusPass || !strings.HasPrefix(rows[0].Message, "Submitted before "+wantEdge+": ") ||
			rows[1].Status != StatusInfo || rows[1].Message != bitcoinNoEdge {
			t.Errorf("Submitted Before rows:\n%s\nwant a pass at %s and the Bitcoin info", render(rows), wantEdge)
		}
		assertNoCarriedTime(t, r, wantEdge, carried...)
	})
}
