// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// The Bitcoin evidence of DERIVE (SPEC.md section 10.2, step 9 (d)) on the
// positive item proof and mutants of it, beside the bitcoin cases of
// internal/vectorcases' runner: the transaction's OP_RETURN against the
// recomputed epoch_hash, the rules of well-formedness, the checks a
// malformed structure leaves unmade, and the order of DERIVE's checks.
// Each mutant changes the bytes of /commitments/bitcoin and, where it
// says so, recomputes transaction_hash for the new bytes, so the rows that
// fail are only those the change breaks.

// topBitcoin is the top-level Bitcoin entry.
var topBitcoin = Pointer{Name("commitments"), Name("bitcoin")}

// rawOf is the bytes of the raw_transaction of the entry at n.
func rawOf(t *testing.T, proof map[string]any, n Pointer) []byte {
	t.Helper()
	b, err := hex.DecodeString(valueAt(proof, under(n, "raw_transaction")).(string))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// displayedSHA256d is rev(SHA-256(SHA-256(b))) as hex, computed here.
func displayedSHA256d(b []byte) string {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	slices.Reverse(second[:])
	return hex.EncodeToString(second[:])
}

// withRaw is proof with raw as the raw_transaction of the entry at n, and,
// when rehash is true, its transaction_hash recomputed for raw.
func withRaw(proof map[string]any, n Pointer, raw []byte, rehash bool) map[string]any {
	out := put(proof, under(n, "raw_transaction"), hex.EncodeToString(raw))
	if rehash {
		out = put(out, under(n, "transaction_hash"), displayedSHA256d(raw))
	}
	return out
}

// opReturnOutput is the last output of the positive item's transaction:
// its 8-octet value, its length octet 0x22 and its script, 0x6a 0x20 and
// the epoch hash, before the 4-octet lock time.
func opReturnOutput(t *testing.T, raw []byte) []byte {
	t.Helper()
	out := raw[len(raw)-4-43 : len(raw)-4]
	if out[8] != 0x22 || out[9] != 0x6a || out[10] != 0x20 {
		t.Fatalf("the last output is %x, not an OP_RETURN of 32 bytes", out)
	}
	return out
}

// rowsAt are the run's DERIVE rows at part.
func rowsAt(run *Run, part string) []Row {
	var out []Row
	for _, row := range deriveRows(run) {
		if *row.Part == part {
			out = append(out, row)
		}
	}
	return out
}

// The 32 bytes of the OP_RETURN that differ from the recomputed epoch_hash
// fail at raw_transaction with derivation_mismatch (SPEC.md section 5.4),
// declared as the bytes pushed and recomputed as H(0x40, epoch.record),
// and the row of the matched hash fails too, since the txoutproof matches
// the transaction as it was; transaction_hash, recomputed for the new
// bytes, passes. Mutants killed: the OP_RETURN row left out (no
// raw_transaction failure); its declared value taken from the member at
// the part (the whole raw_transaction hex).
func TestBitcoinEvidence_AnOpReturnThatIsNotTheEpochHash(t *testing.T) {
	proof := positiveProofs(t)["item"]
	raw := rawOf(t, proof, topBitcoin)
	copy(raw[len(raw)-4-32:len(raw)-4], make([]byte, 32))
	run := derived(withRaw(proof, topBitcoin, raw, true))
	want := []string{"derivation_mismatch /commitments/bitcoin/raw_transaction", "derivation_mismatch /commitments/bitcoin/txoutproof"}
	if got := failing(run); !slices.Equal(got, want) {
		t.Fatalf("failing %q, want %q", got, want)
	}
	rows := rowsAt(run, "/commitments/bitcoin/raw_transaction")
	epochHash := modelHash(t, "epoch_hash", valueAt(proof, under(topBitcoin, "epoch", "record")))
	wantValues := map[string]any{"construction": "bitcoin", "declared": strings.Repeat("00", 32), "recomputed": epochHash}
	if len(rows) != 2 || rows[0].Status != StatusPass || !reflect.DeepEqual(rows[1].Values, wantValues) {
		t.Fatalf("raw_transaction rows %+v, want a pass and %v", rows, wantValues)
	}
}

// The OP_RETURN is compared with the epoch_hash this run recomputes from
// epoch.record, never with the one the entry carries (SPEC.md sections 5.4
// and 10.3, and section 10.6, rule 4): an epoch record whose leaf_count is
// changed, beside the genuine epoch_hash and OP_RETURN, fails at
// epoch_hash, at the inclusion proof's tree_size and at the OP_RETURN,
// whose recomputed value is the changed record's hash; and the same record
// with its epoch_hash declared as that hash fails at the OP_RETURN still.
// Mutant killed: the OP_RETURN compared with N/epoch/epoch_hash (the first
// passes it).
func TestBitcoinEvidence_TheOpReturnIsComparedWithTheRecomputedEpochHash(t *testing.T) {
	proof := positiveProofs(t)["item"]
	recordAt := under(topBitcoin, "epoch", "record", "leaf_count")
	count := valueAt(proof, recordAt).(strictjson.Number)
	changed := put(proof, recordAt, count+1)
	recomputed := modelHash(t, "epoch_hash", valueAt(changed, under(topBitcoin, "epoch", "record")))
	carried := valueAt(proof, under(topBitcoin, "epoch", "epoch_hash"))
	for _, c := range []struct {
		name  string
		proof map[string]any
		want  []string
	}{
		{"the genuine epoch_hash", changed, []string{"derivation_mismatch /commitments/bitcoin/epoch/epoch_hash",
			"derivation_mismatch /commitments/bitcoin/inclusion_proof/tree_size", "derivation_mismatch /commitments/bitcoin/raw_transaction"}},
		{"the changed record's epoch_hash", put(changed, under(topBitcoin, "epoch", "epoch_hash"), recomputed),
			[]string{"derivation_mismatch /commitments/bitcoin/inclusion_proof/tree_size", "derivation_mismatch /commitments/bitcoin/raw_transaction"}},
	} {
		run := derived(c.proof)
		if got := failing(run); !slices.Equal(got, c.want) {
			t.Errorf("%s: failing %q, want %q", c.name, got, c.want)
			continue
		}
		row := rowsAt(run, "/commitments/bitcoin/raw_transaction")[1]
		if row.Values["declared"] != carried || row.Values["recomputed"] != recomputed {
			t.Errorf("%s: the OP_RETURN row's values %v, want declared %v and recomputed %s", c.name, row.Values, carried, recomputed)
		}
	}
}

// A raw_transaction that is not well formed fails at raw_transaction with
// bitcoin_evidence_invalid alone, its values the constructions row and the
// rule it breaks, and the OP_RETURN row, which reads a value out of it, is
// not made; transaction_hash, which reads only the bytes, is still checked,
// and so is the hash the txoutproof matches (SPEC.md section 10.2, step 9
// (d)). Every OP_RETURN is read, never only the first, and a witness
// serialization is refused, never stripped:
//
//   - two OP_RETURN outputs, transaction_hash recomputed: op_return;
//   - an OP_RETURN that pushes its 32 bytes with OP_PUSHDATA1,
//     transaction_hash recomputed: op_return;
//   - the transaction serialized with a witness (BIP 144), transaction_hash
//     its txid as carried: parse, and transaction_hash fails, recomputed as
//     rev(SHA-256d) of the bytes as carried, the witness hash;
//   - a byte after the lock time, transaction_hash recomputed: parse.
//
// Mutants killed: the OP_RETURN row made for a malformed transaction (a
// second row at raw_transaction); the failing row's values holding
// declared and recomputed in place of the rule; transaction_hash
// recomputed over the transaction with its witness stripped (the witness
// case's transaction_hash passes).
func TestBitcoinEvidence_AMalformedTransactionLeavesItsOpReturnUnread(t *testing.T) {
	proof := positiveProofs(t)["item"]
	raw := rawOf(t, proof, topBitcoin)
	output := opReturnOutput(t, raw)
	body := raw[:len(raw)-4]
	twoReturns := slices.Concat(body[:46], []byte{0x03}, body[47:], output, raw[len(raw)-4:])
	if raw[46] != 0x02 {
		t.Fatalf("the transaction's output count is %#x, want 2", raw[46])
	}
	pushdata := slices.Concat(raw[:len(raw)-4-35], []byte{0x23, 0x6a, 0x4c, 0x20}, raw[len(raw)-4-32:])
	witness := slices.Concat(raw[:4], []byte{0x00, 0x01}, raw[4:len(raw)-4], []byte{0x01, 0x00}, raw[len(raw)-4:])
	trailing := slices.Concat(raw, []byte{0x00})
	txid := valueAt(proof, under(topBitcoin, "transaction_hash"))
	for _, c := range []struct {
		name   string
		proof  map[string]any
		rule   string
		failed []string
	}{
		{"two OP_RETURN outputs", withRaw(proof, topBitcoin, twoReturns, true), "op_return",
			[]string{"bitcoin_evidence_invalid /commitments/bitcoin/raw_transaction", "derivation_mismatch /commitments/bitcoin/txoutproof"}},
		{"OP_PUSHDATA1", withRaw(proof, topBitcoin, pushdata, true), "op_return",
			[]string{"bitcoin_evidence_invalid /commitments/bitcoin/raw_transaction", "derivation_mismatch /commitments/bitcoin/txoutproof"}},
		{"a witness serialization", withRaw(proof, topBitcoin, witness, false), "parse",
			[]string{"bitcoin_evidence_invalid /commitments/bitcoin/raw_transaction", "derivation_mismatch /commitments/bitcoin/transaction_hash",
				"derivation_mismatch /commitments/bitcoin/txoutproof"}},
		{"a byte after the lock time", withRaw(proof, topBitcoin, trailing, true), "parse",
			[]string{"bitcoin_evidence_invalid /commitments/bitcoin/raw_transaction", "derivation_mismatch /commitments/bitcoin/txoutproof"}},
	} {
		run := derived(c.proof)
		if got := failing(run); !slices.Equal(got, c.failed) {
			t.Errorf("%s: failing %q, want %q", c.name, got, c.failed)
			continue
		}
		rows := rowsAt(run, "/commitments/bitcoin/raw_transaction")
		if want := map[string]any{"construction": "bitcoin", "rule": c.rule}; len(rows) != 1 || !reflect.DeepEqual(rows[0].Values, want) {
			t.Errorf("%s: raw_transaction rows %+v, want one with %v", c.name, rows, want)
		}
		matched := rowsAt(run, "/commitments/bitcoin/txoutproof")[1]
		raw := rawOf(t, c.proof, topBitcoin)
		if matched.Values["declared"] != txid || matched.Values["recomputed"] != displayedSHA256d(raw) {
			t.Errorf("%s: the matched hash's row %v, want declared %v and recomputed %s", c.name, matched.Values, txid, displayedSHA256d(raw))
		}
		if c.rule == "parse" && strings.HasPrefix(c.name, "a witness") {
			row := rowsAt(run, "/commitments/bitcoin/transaction_hash")[0]
			if row.Values["recomputed"] != displayedSHA256d(witness) || row.Values["declared"] != txid {
				t.Errorf("%s: transaction_hash row %v", c.name, row.Values)
			}
		}
	}
}

// A txoutproof that is not well formed fails at txoutproof with
// bitcoin_evidence_invalid alone, and the four checks that read it, the
// matched hash and the three header values, are not made, so the trace has
// no header row (SPEC.md section 10.2, step 9 (d)): the positive item's
// flag octet with its last, unused, bit set (partial_tree), an unused flag
// bit set being refused whatever the count of flag octets; and a byte after
// the flags (parse). Mutant killed: the header rows made for a malformed
// txoutproof (rows at header_hash, header_merkle_root and header_time).
func TestBitcoinEvidence_AMalformedTxoutproofLeavesItsHeaderUnread(t *testing.T) {
	proof := positiveProofs(t)["item"]
	proofAt := under(topBitcoin, "txoutproof")
	b, err := hex.DecodeString(valueAt(proof, proofAt).(string))
	if err != nil {
		t.Fatal(err)
	}
	unusedBit := slices.Clone(b)
	unusedBit[len(unusedBit)-1] |= 0x80
	if b[len(b)-1]&0x80 != 0 {
		t.Fatalf("the flag octet %#x uses its last bit", b[len(b)-1])
	}
	for _, c := range []struct {
		name string
		b    []byte
		rule string
	}{{"an unused flag bit", unusedBit, "partial_tree"}, {"a byte after the flags", slices.Concat(b, []byte{0x00}), "parse"}} {
		run := derived(put(proof, proofAt, hex.EncodeToString(c.b)))
		if got, want := failing(run), []string{"bitcoin_evidence_invalid /commitments/bitcoin/txoutproof"}; !slices.Equal(got, want) {
			t.Errorf("%s: failing %q, want %q", c.name, got, want)
			continue
		}
		rows := rowsAt(run, proofAt.Encode())
		if want := map[string]any{"construction": "bitcoin", "rule": c.rule}; len(rows) != 1 || !reflect.DeepEqual(rows[0].Values, want) {
			t.Errorf("%s: txoutproof rows %+v, want one with %v", c.name, rows, want)
		}
		for _, member := range []string{"header_hash", "header_merkle_root", "header_time"} {
			part := under(topBitcoin, member).Encode()
			if len(rowsAt(run, part)) != 0 || len(traceAt(run, part)) != 0 {
				t.Errorf("%s: a row or trace row at %s", c.name, part)
			}
		}
		if len(rowsAt(run, under(topBitcoin, "transaction_hash").Encode())) != 1 {
			t.Errorf("%s: no transaction_hash row", c.name)
		}
	}
}

// The eight checks of one Bitcoin entry come in the order of SPEC.md
// section 10.2, step 9 (d)'s table, each at its part, reading its inputs,
// with the four CHECKED values traced and the rows at raw_transaction and
// txoutproof, which are PREIMAGE, untraced; each names the constructions
// row bitcoin. Mutants killed: a header row reading raw_transaction; the
// OP_RETURN row without the epoch record among its inputs; the matched
// hash's row traced; the well-formedness and OP_RETURN rows swapped.
func TestBitcoinEvidence_TheEightChecksOfAnEntry(t *testing.T) {
	proof := positiveProofs(t)["item"]
	checks, err := bitcoinEntryChecks(proof, topBitcoin)
	if err != nil {
		t.Fatal(err)
	}
	n := "/commitments/bitcoin/"
	want := []string{
		"raw_transaction [/commitments/bitcoin/raw_transaction] false pass",
		"raw_transaction [/commitments/bitcoin/epoch/record /commitments/bitcoin/raw_transaction] false pass",
		"transaction_hash [/commitments/bitcoin/raw_transaction] true pass",
		"txoutproof [/commitments/bitcoin/txoutproof] false pass",
		"txoutproof [/commitments/bitcoin/raw_transaction /commitments/bitcoin/txoutproof] false pass",
		"header_hash [/commitments/bitcoin/txoutproof] true pass",
		"header_merkle_root [/commitments/bitcoin/txoutproof] true pass",
		"header_time [/commitments/bitcoin/txoutproof] true pass",
	}
	var got []string
	for _, c := range checks {
		var inputs []string
		for _, input := range c.inputs {
			inputs = append(inputs, input.Encode())
		}
		status := "pass"
		if c.code != "" {
			status = string(c.code)
		}
		if c.construction != "bitcoin" {
			t.Errorf("%s names the row %s", c.part.Encode(), c.construction)
		}
		got = append(got, strings.TrimPrefix(c.part.Encode(), n)+" ["+strings.Join(inputs, " ")+"] "+map[bool]string{false: "false", true: "true"}[c.traced]+" "+status)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the entry's checks\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// deriveChecks gives the record checks, then the Bitcoin evidence's, then
// the links' (SPEC.md section 10.2, step 9, parts (a) to (c), (d) and (e)),
// each part's checks in its own order, for every positive proof: exactly
// recordChecks, bitcoinEvidenceChecks and linkChecks concatenated, the
// records and links never empty, and eight Bitcoin checks for each of the
// 19 Bitcoin entries the proofs carry. The engine's stable sort hides this
// order wherever the parts differ, which they do between the three, so the
// checks are compared before it. Mutants killed: deriveChecks giving the
// Bitcoin evidence after the links, or before the records.
func TestDerive_TheChecksAreTheRecordsThenTheBitcoinEvidenceThenTheLinks(t *testing.T) {
	key := publicOf(vectorSigner)
	summary := func(checks []deriveCheck) []string {
		var out []string
		for _, c := range checks {
			out = append(out, c.construction+" "+c.part.Encode())
		}
		return out
	}
	withEvidence := 0
	for _, name := range sortedProofs(positiveProofs(t)) {
		proof := positiveProofs(t)[name]
		checks, err := deriveChecks(proof, key, deriveAttribution)
		if err != nil {
			t.Fatal(err)
		}
		records, err1 := recordChecks(proof)
		evidence, err2 := bitcoinEvidenceChecks(proof)
		links, err3 := linkChecks(proof, key, deriveAttribution)
		entries := 0
		for _, n := range []Pointer{topBitcoin, {Name("signing_key_event"), Name("commitments"), Name("bitcoin")}} {
			if valueAt(proof, n) != nil {
				entries++
			}
		}
		if err1 != nil || err2 != nil || err3 != nil || len(records) == 0 || len(evidence) != 8*entries || len(links) == 0 {
			t.Fatalf("%s: %d, %d, %d checks for %d Bitcoin entries; %v %v %v", name, len(records), len(evidence), len(links), entries, err1, err2, err3)
		}
		withEvidence += entries
		if got, want := summary(checks), summary(slices.Concat(records, evidence, links)); !slices.Equal(got, want) {
			t.Errorf("%s: deriveChecks\n  %s\nwant\n  %s", name, strings.Join(got, "\n  "), strings.Join(want, "\n  "))
		}
	}
	if withEvidence != 19 {
		t.Fatalf("the positive proofs carry %d Bitcoin entries, want 19", withEvidence)
	}
}

// A raw_transaction or txoutproof that is not a string, or does not decode
// as hex, which SHAPE has made impossible, is a fault of the verifier: one
// row fail internal_error, part null, the run ended, and no trace (SPEC.md
// section 10.5). Mutant killed: hexBytesAt's decoding error ignored (the
// run goes on with no bytes, and its rows fail bitcoin_evidence_invalid).
func TestBitcoinEvidence_HexThatDoesNotDecodeIsAFault(t *testing.T) {
	proof := positiveProofs(t)["item"]
	want := "[internal_error null fail DERIVE] [prior_failure null skip CHAIN] [prior_failure null skip SOURCES] " +
		"[prior_failure null skip WINDOW] [prior_failure null skip EXPECTED_HASH]"
	for _, member := range []string{"raw_transaction", "txoutproof"} {
		for _, value := range []any{"zz", strictjson.Number(1)} {
			run := derived(put(proof, under(topBitcoin, member), value))
			if got := rowSummary(run.Rows[8:]); got != want || !run.Ended || run.Trace != nil {
				t.Errorf("%s %v: %s, ended %v, trace %v", member, value, got, run.Ended, run.Trace)
			}
		}
	}
}
