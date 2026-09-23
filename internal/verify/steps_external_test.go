// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/bitcoin"
	"github.com/truestamp/truestamp-cli/internal/external"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// The mainnet block the header in testfixtures.BitcoinUnrelatedTxoutproof's
// txoutproof recomputes to.
const (
	unrelatedBlockHash   = "0000000000000000000015bfa786d8b26c489171c41c897c15a9197f1879a2a3"
	unrelatedBlockHeight = 870456
)

// blockstreamStub answers a Blockstream header lookup for exactly one
// block, the way the public chain answers it, and counts the lookups.
type blockstreamStub struct {
	blockHash string
	height    int

	mu      sync.Mutex
	lookups int
}

func (s *blockstreamStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/block/"+s.blockHash {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	s.lookups++
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"id":%q,"height":%d,"timestamp":1731702183}`, s.blockHash, s.height)
}

func (s *blockstreamStub) headerLookups() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lookups
}

// withStubbedSources points both Blockstream endpoints at stub and every
// Horizon and NIST endpoint at an address that refuses connections, so an
// online run consults nothing outside the test.
func withStubbedSources(t *testing.T, stub *blockstreamStub) {
	t.Helper()
	srv := httptest.NewServer(stub)
	t.Cleanup(srv.Close)
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	saved := []*string{
		&external.BlockstreamMainnetURL, &external.BlockstreamTestnetURL,
		&external.HorizonPublicURL, &external.HorizonTestnetURL, &external.NISTBeaconURL,
	}
	values := make([]string, len(saved))
	for i, p := range saved {
		values[i] = *p
	}
	t.Cleanup(func() {
		for i, p := range saved {
			*p = values[i]
		}
	})
	external.BlockstreamMainnetURL, external.BlockstreamTestnetURL = srv.URL, srv.URL
	external.HorizonPublicURL, external.HorizonTestnetURL, external.NISTBeaconURL = deadURL, deadURL, deadURL
}

// rowsOf returns the rows of one group.
func rowsOf(r *Report, group string) []Step {
	var out []Step
	for _, s := range r.Steps {
		if s.Group == group {
			out = append(out, s)
		}
	}
	return out
}

func assertNoPass(t *testing.T, r *Report, group string) {
	t.Helper()
	for _, s := range rowsOf(r, group) {
		if s.Status == StatusPass {
			t.Errorf("%s passed: %q\n%s", group, s.Message, render(r.Steps))
		}
	}
}

// TestBitcoinUnrelatedTxoutproof pins E01: a real txoutproof of an
// unrelated transaction, with no raw transaction, never yields a Bitcoin
// pass, even when the chain confirms the header it carries at the height
// the entry names.
func TestBitcoinUnrelatedTxoutproof(t *testing.T) {
	data := testfixtures.Read(t, testfixtures.TamperDir, testfixtures.BitcoinUnrelatedTxoutproof)

	t.Run("offline", func(t *testing.T) {
		r := mustRun(t, data, Options{SkipExternal: true})
		assertNoPass(t, r, groupBitcoin)
		assertNoPass(t, r, groupSubmittedBefore)
		// The fixture exercises the E01 path: the proof is otherwise
		// sound, and its txoutproof really does place the txid it names.
		if !r.Passed() {
			t.Errorf("verdict FAILED, want the rest of the bundle to verify\n%s", render(r.Steps))
		}
		placed := false
		for _, s := range rowsOf(r, groupBitcoin) {
			if s.Status == StatusInfo && strings.HasPrefix(s.Message, "Supplied txoutproof places the transaction") {
				placed = true
			}
		}
		if !placed {
			t.Errorf("the fixture's txoutproof does not place its transaction\n%s", render(r.Steps))
		}
	})

	t.Run("online, header confirmed by the chain", func(t *testing.T) {
		stub := &blockstreamStub{blockHash: unrelatedBlockHash, height: unrelatedBlockHeight}
		withStubbedSources(t, stub)
		r := mustRun(t, data, Options{})
		assertNoPass(t, r, groupBitcoin)
		assertNoPass(t, r, groupSubmittedBefore)
		rows := rowsOf(r, groupBitcoin)
		if len(rows) == 0 {
			t.Fatalf("no Bitcoin Commitment rows\n%s", render(r.Steps))
		}
		last := rows[len(rows)-1]
		if last.Status != StatusSkip || !strings.Contains(last.Message, "not looked up") {
			t.Errorf("binding row = %s %q, want a skip that says the header was not looked up\n%s", last.Status, last.Message, render(r.Steps))
		}
		if n := stub.headerLookups(); n != 0 {
			t.Errorf("the header was looked up %d times; the binding must not run until every link holds", n)
		}
	})
}

// TestBitcoinFullChainConfirms is the control for the test above: the
// staging bundle's Bitcoin entry carries the whole chain (raw transaction,
// OP_RETURN, txid, txoutproof). Relabelled as mainnet, which the signature
// does not cover, and with the chain confirming its header at the entry's
// height, it passes. Without its raw transaction, or with any one other
// link broken, it does not, and the header is not looked up.
func TestBitcoinFullChainConfirms(t *testing.T) {
	data := testfixtures.Read(t, testfixtures.StagingDir, testfixtures.ProdComplete)
	keyring := testfixtures.Path(testfixtures.StagingDir, testfixtures.ProdKeyring)

	var entry map[string]any
	relabel := func(drop ...string) []byte {
		var m map[string]any
		dec := json.NewDecoder(strings.NewReader(string(data)))
		dec.UseNumber()
		if err := dec.Decode(&m); err != nil {
			t.Fatal(err)
		}
		entry = m["commitments"].([]any)[0].(map[string]any)
		if entry["chain"] != "bitcoin" {
			t.Fatalf("staging commitments[0] is %v, want bitcoin", entry["chain"])
		}
		entry["network"] = "mainnet"
		for _, k := range drop {
			delete(entry, k)
		}
		out, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	full := relabel()
	headerHash, err := bitcoin.ParseBlockHash(entry["txoutproof"].(string))
	if err != nil {
		t.Fatal(err)
	}
	height, err := entry["block_height"].(json.Number).Int64()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("whole chain", func(t *testing.T) {
		stub := &blockstreamStub{blockHash: headerHash, height: int(height)}
		withStubbedSources(t, stub)
		r := mustRun(t, full, Options{KeyringFile: keyring})
		want := fmt.Sprintf("Block %s confirmed on mainnet (height %d)", headerHash, height)
		found := false
		for _, s := range rowsOf(r, groupBitcoin) {
			if s.Status == StatusFail {
				t.Errorf("Bitcoin Commitment failed: %q\n%s", s.Message, render(r.Steps))
			}
			if s.Status == StatusPass && s.Message == want {
				found = true
			}
		}
		if !found {
			t.Errorf("no %q pass\n%s", want, render(r.Steps))
		}
	})

	t.Run("no raw transaction", func(t *testing.T) {
		stub := &blockstreamStub{blockHash: headerHash, height: int(height)}
		withStubbedSources(t, stub)
		r := mustRun(t, relabel("raw_transaction"), Options{KeyringFile: keyring})
		assertNoPass(t, r, groupBitcoin)
		if n := stub.headerLookups(); n != 0 {
			t.Errorf("the header was looked up %d times without a raw transaction", n)
		}
	})

	// Each case below breaks exactly one link of the chain while every
	// other check holds, and the chain would confirm the header its
	// txoutproof carries at the entry's height. confirmBitcoin runs on the
	// parsed entry, so which fields the signature covers does not matter.
	// The first case is the control: with every link holding, the header
	// is looked up once and confirmed.
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	staged := bundle.Commitments[0]
	staged.Network = "mainnet"

	// The staging block holds two transactions, the coinbase and the
	// commitment. Its txoutproof ends with one flag byte, 05, which
	// matches only the second leaf; 07 matches both.
	const coinbaseTxid = "008ef46f954307fd8aa3f9bcb243396b72c53c5270278add565fa33f2a5d3bf7"
	if !strings.HasSuffix(staged.Txoutproof, "0105") {
		t.Fatalf("staging txoutproof does not end with the one flag byte 05: %s", staged.Txoutproof)
	}

	unrelated, err := proof.ParseBytes(testfixtures.Read(t, testfixtures.TamperDir, testfixtures.BitcoinUnrelatedTxoutproof))
	if err != nil {
		t.Fatal(err)
	}
	var unrelatedTxoutproof string
	for _, c := range unrelated.Commitments {
		if c.Chain == "bitcoin" {
			unrelatedTxoutproof = c.Txoutproof
		}
	}
	unrelatedBlock, err := bitcoin.DecodeTxOutProof(unrelatedTxoutproof)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name   string
		breaks func(e *proof.Commitment) // nil for the control
		header string                    // the block the stub confirms
		failed string                    // the start of the one fail row
	}{
		{name: "every link holds", header: headerHash},
		{
			name:   "OP_RETURN differs from the epoch root",
			breaks: func(e *proof.Commitment) { e.EpochMerkleRoot = strings.Repeat("ab", 32) },
			header: headerHash,
			failed: "Check failed: OP_RETURN in the supplied raw transaction matches the epoch root",
		},
		{
			// The txoutproof places the coinbase txid, and the raw
			// transaction, whose OP_RETURN still matches, does not hash
			// to it.
			name: "recomputed txid differs from a placed transaction_hash",
			breaks: func(e *proof.Commitment) {
				e.Txoutproof = strings.TrimSuffix(e.Txoutproof, "05") + "07"
				e.TransactionHash = coinbaseTxid
			},
			header: headerHash,
			failed: "Check failed: Transaction id " + coinbaseTxid + " recomputed",
		},
		{
			// A real txoutproof of an unrelated mainnet transaction, with
			// block_merkle_root set to its header's root so the
			// cross-check holds.
			name: "txoutproof does not place transaction_hash",
			breaks: func(e *proof.Commitment) {
				e.Txoutproof = unrelatedTxoutproof
				e.BlockMerkleRoot = unrelatedBlock.Header.MerkleRoot.String()
			},
			header: unrelatedBlockHash,
			failed: "Check failed: Supplied txoutproof places the transaction",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := staged
			if tc.breaks != nil {
				tc.breaks(&e)
			}
			stub := &blockstreamStub{blockHash: tc.header, height: e.BlockHeight}
			withStubbedSources(t, stub)
			r := &Report{}
			confirmed := confirmBitcoin(r, &e, groupBitcoin, CatBlockchain, Options{})

			var fails []Step
			for _, s := range r.Steps {
				if s.Status == StatusFail {
					fails = append(fails, s)
				}
			}
			if tc.breaks == nil {
				if !confirmed || stub.headerLookups() != 1 || len(fails) != 0 {
					t.Errorf("confirmed = %v after %d lookups, want the header looked up once and confirmed\n%s",
						confirmed, stub.headerLookups(), render(r.Steps))
				}
				return
			}
			if confirmed {
				t.Errorf("confirmed with a broken link\n%s", render(r.Steps))
			}
			assertNoPass(t, r, groupBitcoin)
			if n := stub.headerLookups(); n != 0 {
				t.Errorf("the header was looked up %d times; the binding must not run until every link holds", n)
			}
			if len(fails) != 1 || !strings.HasPrefix(fails[0].Message, tc.failed) {
				t.Errorf("rows = \n%s, want exactly one fail, starting %q", render(r.Steps), tc.failed)
			}
		})
	}
}

// TestCheckBitcoinRawTx_SixtyFourBytes pins CVE-2017-12842: a transaction
// whose txid preimage is 64 bytes has the length of an inner Merkle node,
// so it never completes the chain, even when every other check on it
// would hold. For a witness transaction the preimage is the serialization
// without witness data.
func TestCheckBitcoinRawTx_SixtyFourBytes(t *testing.T) {
	input := "01" + strings.Repeat("00", 32) + "ffffffff" + "00" + "ffffffff" // one input, empty scriptSig
	output := "01" + "0000000000000000" + "04" + "6a02abcd"                   // one output: OP_RETURN, push 2
	for _, tc := range []struct {
		name   string
		raw    string
		bytes  int    // as supplied
		starts string // how the fail row starts
	}{
		{
			name:   "64 bytes as supplied",
			raw:    "01000000" + input + output + "00000000",
			bytes:  64,
			starts: "Raw transaction rejected: it is exactly 64 bytes,",
		},
		{
			// Marker and flag 0001, and one witness item of one byte.
			name:   "64 bytes without witness data",
			raw:    "01000000" + "0001" + input + output + "010100" + "00000000",
			bytes:  69,
			starts: "Raw transaction rejected: without its witness data, the bytes its txid hashes, it is exactly 64 bytes,",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := bitcoin.DecodeTransaction(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			if got, stripped := len(tc.raw)/2, tx.SerializeSizeStripped(); got != tc.bytes || stripped != 64 {
				t.Fatalf("test transaction is %d bytes, %d without witness data; want %d and 64", got, stripped, tc.bytes)
			}
			txid, err := bitcoin.ComputeTxID(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			entry := &proof.Commitment{Chain: "bitcoin", RawTransaction: tc.raw, TransactionHash: txid, EpochMerkleRoot: "abcd"}
			r := &Report{}
			if checkBitcoinRawTx(r, entry, groupBitcoin, CatBlockchain) {
				t.Errorf("the transaction completed the chain\n%s", render(r.Steps))
			}
			if len(r.Steps) != 1 || r.Steps[0].Status != StatusFail || !strings.HasPrefix(r.Steps[0].Message, tc.starts) {
				t.Errorf("rows = \n%s, want one fail starting %q", render(r.Steps), tc.starts)
			}
		})
	}
}
