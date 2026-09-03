// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package bitcoin

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/wire/v2"
)

// buildRawTx serializes a transaction with one dummy input and one output per
// supplied pkScript, returning its raw hex. Values and the input are
// irrelevant to ExtractOpReturn, only the output scripts matter.
func buildRawTx(t *testing.T, scripts ...[]byte) string {
	t.Helper()
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(wire.NewTxIn(&wire.OutPoint{Index: 0}, nil, nil))
	for _, s := range scripts {
		tx.AddTxOut(wire.NewTxOut(0, s))
	}
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		t.Fatalf("serialize tx: %s", err)
	}
	return hex.EncodeToString(buf.Bytes())
}

// mustDecodeHex is a test-local hex decoder for hand-written scripts.
func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode script hex: %s", err)
	}
	return b
}

const (
	payloadA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 32 bytes
	payloadB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" // 32 bytes
	// A P2TR output, i.e. the change output a real commitment transaction
	// carries alongside its OP_RETURN.
	p2trScript = "5120d2972046141d369b5eb51ad6fd4efd463c3458e9eb416ecccb521eb5ef38188d"
)

// canonicalOpReturn builds the grammar generation emits: OP_RETURN (0x6a),
// OP_PUSHBYTES_32 (0x20), then 32 bytes.
func canonicalOpReturn(payloadHex string) string { return "6a20" + payloadHex }

// TestExtractOpReturnFirstOutputWins pins the Appendix E.19 selection rule:
// the payload is the push data of the FIRST output whose script begins with
// 0x6a, parsed on that output alone. Every adversarial case below has a
// perfectly well-formed OP_RETURN in a LATER output; before the selection
// rule was enforced, each of them silently returned that later payload, so a
// crafted transaction could present bytes generation never emitted.
func TestExtractOpReturnFirstOutputWins(t *testing.T) {
	t.Parallel()

	decoy := mustDecodeHex(t, canonicalOpReturn(payloadB))

	cases := []struct {
		name       string
		scripts    []string
		wantHex    string
		wantErrSub string
	}{
		{
			name:    "canonical single output",
			scripts: []string{canonicalOpReturn(payloadA)},
			wantHex: payloadA,
		},
		{
			name:    "skips non-OP_RETURN outputs",
			scripts: []string{p2trScript, canonicalOpReturn(payloadA)},
			wantHex: payloadA,
		},
		{
			// OP_PUSHBYTES_32 announced but only 31 bytes follow. IsNullData
			// rejects this script, so the old scan fell through to the decoy.
			name:       "truncated push in the first OP_RETURN",
			scripts:    []string{"6a20" + payloadA[:62], canonicalOpReturn(payloadB)},
			wantErrSub: "does not parse",
		},
		{
			// OP_PUSHDATA1 of 81 bytes: valid script grammar, but over
			// txscript.MaxDataCarrierSize, which is a relay policy limit and
			// not a parse failure. E.19 selects on the leading opcode, so the
			// oversize payload is what gets returned.
			name:    "oversize push in the first OP_RETURN",
			scripts: []string{"6a4c51" + strings.Repeat("aa", 81), canonicalOpReturn(payloadB)},
			wantHex: strings.Repeat("aa", 81),
		},
		{
			// A bare OP_RETURN is valid nulldata with zero pushes. The old
			// `len(pushes) > 0` guard silently walked past it to the decoy.
			name:       "bare OP_RETURN with no push data",
			scripts:    []string{"6a", canonicalOpReturn(payloadB)},
			wantErrSub: "carries no push data",
		},
		{
			// Two pushes in one output. IsNullData rejects multi-push
			// nulldata, so the old scan fell through; E.19 takes the first
			// push of the first 0x6a output.
			name:    "multi-push first OP_RETURN",
			scripts: []string{canonicalOpReturn(payloadA) + "0199", canonicalOpReturn(payloadB)},
			wantHex: payloadA,
		},
		{
			name:       "no OP_RETURN output at all",
			scripts:    []string{p2trScript},
			wantErrSub: "no OP_RETURN output found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scripts := make([][]byte, 0, len(tc.scripts))
			for _, s := range tc.scripts {
				scripts = append(scripts, mustDecodeHex(t, s))
			}
			got, err := ExtractOpReturn(buildRawTx(t, scripts...))

			if tc.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got payload %q", tc.wantErrSub, got)
				}
				if !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Errorf("error = %q, want it to contain %q", err, tc.wantErrSub)
				}
				if got == hex.EncodeToString(decoy[2:]) {
					t.Error("returned the later output's payload; selection must stop at the first 0x6a output")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got != tc.wantHex {
				t.Errorf("payload = %s, want %s", got, tc.wantHex)
			}
		})
	}
}

// TestExtractOpReturnEmptyScript pins that a zero-length pkScript is skipped
// rather than indexed into.
func TestExtractOpReturnEmptyScript(t *testing.T) {
	t.Parallel()
	raw := buildRawTx(t, []byte{}, mustDecodeHex(t, canonicalOpReturn(payloadA)))
	got, err := ExtractOpReturn(raw)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got != payloadA {
		t.Errorf("payload = %s, want %s", got, payloadA)
	}
}

func TestExtractOpReturnInvalidHex(t *testing.T) {
	t.Parallel()
	if _, err := ExtractOpReturn("not-hex"); err == nil {
		t.Error("expected an error for non-hex transaction input")
	}
}
