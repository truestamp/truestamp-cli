// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Status pinning.
//
// Appendix E is, more than anything else, a specification of which
// condition earns which status: E.18 settles that a Stellar commitment
// absent from the chain FAILS while an unreachable Horizon SKIPS; E.21
// settles the same split for entropy sources; E.22 forbids a skip from
// reading as a positive assertion. A suite that asserts only message text
// cannot tell those apart — every one of those pairs shares its wording
// with the branch it must be distinguished from, and a `fail` silently
// downgraded to a `skip` stops failing the run while the report still
// reads the same.
//
// A mutation sweep over internal/verify/verify.go (each `r.fail(` ->
// `r.skip(`, each `r.skip(` -> `r.pass(`, each `r.check` condition ->
// `true`) measured that gap: 31 of 60 fail sites, 14 of 34 skip sites and
// one check condition survived the whole suite. The tests in this file
// exist to kill those mutants. Each case names the condition, the status
// Appendix E assigns it, and asserts BOTH — a message assertion alone is
// what let the mutants live.
//
// The helpers deliberately require a UNIQUE message match: a status
// assertion against an ambiguous row pins nothing.

package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// stepMatching returns the single step whose message contains substr.
// Zero matches means the branch under test was not reached; several means
// the substring does not identify one row, and a status asserted against
// either is worthless.
func stepMatching(t *testing.T, r *Report, substr string) Step {
	t.Helper()
	var got []Step
	for _, s := range r.Steps {
		if strings.Contains(s.Message, substr) {
			got = append(got, s)
		}
	}
	if len(got) != 1 {
		t.Fatalf("want exactly 1 step whose message contains %q, got %d; report was:\n%s",
			substr, len(got), stepMessages(r))
	}
	return got[0]
}

// assertStepStatus pins the (group, category, status) of the row that
// substr identifies. Group and category are asserted alongside the status
// because E.22 defines the report surface as that triple: a row that
// migrates to another group is as much a conformance break as one that
// changes status, and D.4 containment is checked on the pair.
func assertStepStatus(t *testing.T, r *Report, substr, wantGroup, wantCat string, want Status) {
	t.Helper()
	s := stepMatching(t, r, substr)
	if s.Status != want {
		t.Errorf("step %q: status = %v, want %v (message: %q)", substr, s.Status, want, s.Message)
	}
	if s.Group != wantGroup {
		t.Errorf("step %q: group = %q, want %q", substr, s.Group, wantGroup)
	}
	if s.Category != wantCat {
		t.Errorf("step %q: category = %q, want %q", substr, s.Category, wantCat)
	}
}

// --- E.10 / E.13 / E.14 / E.15 / E.16: derivation failures are fails ---

// TestDerivationFailures_AreFailsNotSkips pins the item pipeline's error
// arms. Every one of these reports that a derivation the appendix
// REQUIRES could not be performed, which E.22 grades fail: a skip would
// say the step did not apply, and a proof missing its block hash would
// then verify.
//
// Each case drives the unexported step function directly. That is the
// only way to reach several of these arms at all — E.6 rejects the
// bundles that would produce them at parse time, which is why they went
// unpinned — and it keeps one case per branch instead of one crafted
// bundle per branch.
func TestDerivationFailures_AreFailsNotSkips(t *testing.T) {
	const hex32 = "1111111111111111111111111111111111111111111111111111111111111111"
	const notHex32 = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	const ulid = "01JQMHK4Z1V7Q0J8N2WJXG3B5C"
	const uuid7 = "019cf813-99b8-730a-84f1-5a711a9c355e"

	cases := []struct {
		name  string
		run   func(r *Report)
		want  string // unique message substring
		group string
		cat   string
	}{
		{
			// E.10: s.d present but not canonicalizable. jcs.Canonicalize
			// is strict RFC 8259, so a truncated object reaches the error
			// arm rather than hashing whatever it could parse.
			name:  "claims JCS failure",
			run:   func(r *Report) { deriveClaimsHash(r, json.RawMessage(`{"a":`)) },
			want:  "Claims JCS failed",
			group: groupSubjectData, cat: CatCryptographic,
		},
		{
			name:  "item hash with no subject",
			run:   func(r *Report) { deriveItemHash(r, nil, hex32) },
			want:  "Cannot derive item hash: no subject is present",
			group: groupSubjectData, cat: CatCryptographic,
		},
		{
			// Widths are right so E.10's framing guard passes; the field
			// simply is not hex, so ComputeItemHash cannot build the
			// preimage. Nothing was derived, so nothing may be asserted.
			name: "item hash computation failure",
			run: func(r *Report) {
				deriveItemHash(r, &proof.Subject{ID: ulid, MetadataHash: notHex32, SigningKeyID: "aabbccdd"}, hex32)
			},
			want:  "Item hash computation failed",
			group: groupSubjectData, cat: CatCryptographic,
		},
		{
			// E.13's walk needs a leaf. An upstream derivation failure
			// leaves it with none, and reporting that as a skip would let
			// a bundle suppress the inclusion proof by breaking the step
			// before it.
			name:  "inclusion proof with no subject hash",
			run:   func(r *Report) { verifyInclusionProof(r, "", "AA", proof.Block{MerkleRoot: hex32}) },
			want:  "Cannot verify inclusion proof (no subject hash)",
			group: groupInclusion, cat: CatCryptographic,
		},
		{
			name: "inclusion proof walk error",
			run: func(r *Report) {
				verifyInclusionProof(r, notHex32, "AA", proof.Block{MerkleRoot: hex32})
			},
			want:  "Inclusion proof error",
			group: groupInclusion, cat: CatCryptographic,
		},
		{
			name:  "inclusion proof decode failure",
			run:   func(r *Report) { verifyInclusionProof(r, hex32, "!!!!", proof.Block{MerkleRoot: hex32}) },
			want:  "Inclusion proof decode failed",
			group: groupInclusion, cat: CatCryptographic,
		},
		{
			name: "block hash computation failure",
			run: func(r *Report) {
				deriveBlockHash(r, proof.Block{
					ID: uuid7, PreviousBlockHash: hex32, MerkleRoot: notHex32,
					MetadataHash: hex32, SigningKeyID: "aabbccdd",
				})
			},
			want:  "Block hash computation failed",
			group: groupBlockHash, cat: CatCryptographic,
		},
		{
			// E.15 requires a result per cx entry. A `ep` that is not
			// base64url is a defect in the bundle, not an absence.
			name: "epoch proof decode failure",
			run: func(r *Report) {
				verifyEpochProofs(r, []ExternalCommit{{
					Type: ptype.CommitmentStellar, MemoHash: hex32, EpochProof: "!!!!",
				}}, hex32)
			},
			want:  "Epoch proof 0 decode failed",
			group: groupEpoch, cat: CatCryptographic,
		},
		{
			// No memo on a t=40 entry: there is no committed value to
			// walk to, so the entry's epoch root cannot enter the 0x61
			// payload and the signature step must see the gap.
			name: "epoch proof with no target",
			run: func(r *Report) {
				verifyEpochProofs(r, []ExternalCommit{{
					Type: ptype.CommitmentStellar, EpochProof: "AA",
				}}, hex32)
			},
			want:  "Epoch proof 0: missing target for commitment type 40",
			group: groupEpoch, cat: CatCryptographic,
		},
		{
			name: "epoch proof walk error",
			run: func(r *Report) {
				verifyEpochProofs(r, []ExternalCommit{{
					Type: ptype.CommitmentStellar, MemoHash: hex32, EpochProof: "AA",
				}}, notHex32)
			},
			want:  "Epoch proof 0 error",
			group: groupEpoch, cat: CatCryptographic,
		},
		{
			// E.16 puts ts_ms in the signed payload. A timestamp that
			// does not parse means the payload cannot be rebuilt, so the
			// signature is unverifiable rather than inapplicable.
			name: "unparseable proof timestamp",
			run: func(r *Report) {
				verifyProofSignature(r,
					&proof.ProofBundle{Version: 1, T: ptype.Item, Timestamp: "not-a-timestamp"},
					make([]byte, 32), "aabbccdd", hex32, hex32, []string{hex32}, Options{})
			},
			want:  "Cannot parse proof timestamp",
			group: groupProof, cat: CatCryptographic,
		},
		{
			// Same step, one field further on: the payload builder
			// rejects a key id that is not 4 bytes.
			name: "proof hash computation failure",
			run: func(r *Report) {
				verifyProofSignature(r,
					&proof.ProofBundle{Version: 1, T: ptype.Item, Timestamp: "2026-04-22T20:00:00Z"},
					make([]byte, 32), "zz", hex32, hex32, []string{hex32}, Options{})
			},
			want:  "Proof hash computation failed",
			group: groupProof, cat: CatCryptographic,
		},
		{
			name:  "entropy hash with absent subject data",
			run:   func(r *Report) { deriveEntropyHash(r, nil) },
			want:  "no entropy hash can be derived",
			group: groupSubjectData, cat: CatCryptographic,
		},
		{
			name:  "entropy JCS failure",
			run:   func(r *Report) { deriveEntropyHash(r, json.RawMessage(`{"a":`)) },
			want:  "Entropy JCS failed",
			group: groupSubjectData, cat: CatCryptographic,
		},
		{
			name:  "observation hash with no subject",
			run:   func(r *Report) { deriveObservationHash(r, nil, hex32) },
			want:  "Cannot derive observation hash: no subject is present",
			group: groupSubjectData, cat: CatCryptographic,
		},
		{
			name: "observation hash computation failure",
			run: func(r *Report) {
				deriveObservationHash(r, &proof.Subject{ID: uuid7, MetadataHash: notHex32, SigningKeyID: "aabbccdd"}, hex32)
			},
			want:  "Observation hash computation failed",
			group: groupSubjectData, cat: CatCryptographic,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{}
			tc.run(r)
			assertStepStatus(t, r, tc.want, tc.group, tc.cat, StatusFail)
			// A derivation failure must move the verdict. This is the
			// property the fail->skip mutation removes while leaving
			// every message intact.
			if r.Passed() {
				t.Errorf("the run passed despite %q:\n%s", tc.want, stepMessages(r))
			}
		})
	}
}

// --- E.21: entropy subject defects are fails ---

// TestEntropySubjectDefects_AreFails pins the E.21 arms that report a
// subject the verifier cannot use. Unlike an unreachable upstream (which
// E.21 settles as a skip), these are defects in the bundle itself, and
// grading them skip would let a proof carrying an unparseable entropy
// subject verify.
//
// The upstream is never contacted on any of these paths, so no stub is
// needed — reaching the network would itself be the bug.
func TestEntropySubjectDefects_AreFails(t *testing.T) {
	cases := []struct {
		name string
		run  func(r *Report)
		want string
	}{
		{
			name: "no entropy subject at all",
			run:  func(r *Report) { verifyEntropySource(r, ptype.EntropyNIST, nil, nil, Options{}) },
			want: "Missing entropy subject data",
		},
		{
			// A subject type code outside E.2's entropy range. The
			// verifier has no source to consult, so it must not report
			// the entropy source as merely unchecked.
			name: "unsupported entropy type code",
			run: func(r *Report) {
				verifyEntropySource(r, ptype.Item,
					&proof.Subject{Data: json.RawMessage(`{"a":1}`)}, nil, Options{})
			},
			want: "Unsupported entropy type code 20",
		},
		{
			name: "NIST subject not parseable",
			run:  func(r *Report) { verifyEntropyNIST(r, json.RawMessage(`["not","an","object"]`)) },
			want: "Cannot parse NIST pulse from subject data",
		},
		{
			name: "NIST subject missing outputValue",
			run:  func(r *Report) { verifyEntropyNIST(r, json.RawMessage(`{"chainIndex":1,"pulseIndex":2}`)) },
			want: "NIST entropy subject missing outputValue",
		},
		{
			name: "Stellar subject not parseable",
			run:  func(r *Report) { verifyEntropyStellar(r, json.RawMessage(`["nope"]`), "testnet") },
			want: "Cannot parse Stellar ledger from subject data",
		},
		{
			name: "Stellar subject missing sequence or hash",
			run:  func(r *Report) { verifyEntropyStellar(r, json.RawMessage(`{"closed_at":"x"}`), "testnet") },
			want: "Stellar entropy subject missing sequence or hash",
		},
		{
			// Past the presence guard (sequence != 0) but not a ledger
			// any Horizon can be asked about. E.21 grades an unusable
			// subject value a defect in the bundle, not an
			// unavailability: no lookup is attempted, so no stub is
			// needed and reaching the network here would be the bug.
			name: "Stellar subject sequence is unusable",
			run: func(r *Report) {
				verifyEntropyStellar(r, json.RawMessage(
					`{"hash":"`+strings.Repeat("ab", 32)+`","sequence":-1,"closed_at":"x"}`), "testnet")
			},
			want: "Stellar ledger sequence is unusable",
		},
		{
			name: "Bitcoin subject not parseable",
			run:  func(r *Report) { verifyEntropyBitcoin(r, json.RawMessage(`["nope"]`), "mainnet") },
			want: "Cannot parse Bitcoin block from subject data",
		},
		{
			name: "Bitcoin subject missing hash",
			run:  func(r *Report) { verifyEntropyBitcoin(r, json.RawMessage(`{"height":1}`), "mainnet") },
			want: "Bitcoin entropy subject missing hash",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{}
			tc.run(r)
			assertStepStatus(t, r, tc.want, groupEntropySource, CatBlockchain, StatusFail)
			if r.Passed() {
				t.Errorf("the run passed despite %q:\n%s", tc.want, stepMessages(r))
			}
		})
	}
}

// TestEntropySourceDisagreements_AreFailsNotSkips pins E.21's other half:
// when the upstream IS reachable and reports a different value, that is a
// refutation and must fail. Downgrading any of these to a skip turns a
// proof whose entropy subject does not exist upstream into a verified one.
func TestEntropySourceDisagreements_AreFailsNotSkips(t *testing.T) {
	t.Run("NIST timeStamp", func(t *testing.T) {
		nistStub(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"pulse":{"chainIndex":1,"pulseIndex":2847561,"outputValue":"AABB","timeStamp":"2026-04-22T19:45:00.000Z","version":"2.0"}}`))
		})
		r := &Report{}
		verifyEntropyNIST(r, json.RawMessage(
			`{"chainIndex":1,"pulseIndex":2847561,"outputValue":"AABB","timeStamp":"2020-01-01T00:00:00Z"}`))
		assertStepStatus(t, r, "NIST timeStamp mismatch at pulse", groupEntropySource, CatBlockchain, StatusFail)
	})

	t.Run("Stellar ledger hash", func(t *testing.T) {
		horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"hash":"` + strings.Repeat("ff", 32) +
				`","sequence":51234567,"closed_at":"2026-04-22T20:00:02Z"}`))
		})
		r := &Report{}
		verifyEntropyStellar(r, json.RawMessage(
			`{"hash":"`+strings.Repeat("ab", 32)+`","sequence":51234567,"closed_at":"2026-04-22T20:00:02Z"}`), "testnet")
		assertStepStatus(t, r, "Stellar ledger hash mismatch", groupEntropySource, CatBlockchain, StatusFail)
	})

	t.Run("Stellar closed_at", func(t *testing.T) {
		horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"hash":"` + strings.Repeat("ab", 32) +
				`","sequence":51234567,"closed_at":"2026-04-22T20:00:02Z"}`))
		})
		r := &Report{}
		verifyEntropyStellar(r, json.RawMessage(
			`{"hash":"`+strings.Repeat("ab", 32)+`","sequence":51234567,"closed_at":"2020-01-01T00:00:00Z"}`), "testnet")
		assertStepStatus(t, r, "Stellar ledger closed_at mismatch", groupEntropySource, CatBlockchain, StatusFail)
	})

	t.Run("Bitcoin height", func(t *testing.T) {
		blockstreamStub(t, func(w http.ResponseWriter, req *http.Request) {
			blockstreamBlockBody(t, w, req, 999999, 1700000000)
		})
		r := &Report{}
		verifyEntropyBitcoin(r, json.RawMessage(
			`{"hash":"`+strings.Repeat("ab", 32)+`","height":850000,"time":1700000000}`), "mainnet")
		assertStepStatus(t, r, "Bitcoin height mismatch at hash", groupEntropySource, CatBlockchain, StatusFail)
	})

	t.Run("Bitcoin time", func(t *testing.T) {
		blockstreamStub(t, func(w http.ResponseWriter, req *http.Request) {
			blockstreamBlockBody(t, w, req, 850000, 1699999999)
		})
		r := &Report{}
		verifyEntropyBitcoin(r, json.RawMessage(
			`{"hash":"`+strings.Repeat("ab", 32)+`","height":850000,"time":1700000000}`), "mainnet")
		assertStepStatus(t, r, "Bitcoin block time mismatch", groupEntropySource, CatBlockchain, StatusFail)
	})
}

// TestEntropySource_SkipsAreNotPasses pins the inverse: an entropy source
// that could not be consulted is reported skip, never pass. E.22 forbids
// a row that establishes nothing from reading as one that did, and D.4
// carries the --skip-external entropy row as a skip.
func TestEntropySource_SkipsAreNotPasses(t *testing.T) {
	t.Run("--skip-external", func(t *testing.T) {
		r := &Report{}
		verifyEntropySource(r, ptype.EntropyNIST,
			&proof.Subject{Data: json.RawMessage(nistSubject)}, nil, Options{SkipExternal: true})
		assertStepStatus(t, r, "Entropy source verification skipped", groupEntropySource, CatBlockchain, StatusSkip)
	})

	t.Run("bitcoin network with no public API", func(t *testing.T) {
		r := &Report{}
		verifyEntropyBitcoin(r, json.RawMessage(
			`{"hash":"`+strings.Repeat("ab", 32)+`","height":850000,"time":1700000000}`), "regtest")
		assertStepStatus(t, r, "Entropy source unconfirmed", groupEntropySource, CatBlockchain, StatusSkip)
	})
}

// --- E.18: Stellar commitment grading ---

// TestStellarCommitment_UnconfirmedRowsAreSkips pins the two E.18 arms
// that establish nothing. Promoting either to a pass publishes an
// on-chain confirmation that never happened; both messages already say
// "unconfirmed", which is why only a status assertion catches it.
func TestStellarCommitment_UnconfirmedRowsAreSkips(t *testing.T) {
	t.Run("no transaction id to look up", func(t *testing.T) {
		cx := stellarSample()
		cx.TransactionHash = ""
		r := &Report{}
		verifySingleStellar(r, &cx, Options{})
		assertStepStatus(t, r, "carries no transaction id to look up", groupStellar, CatBlockchain, StatusSkip)
		if ci := onlyCommitment(t, r); ci.ExternalCheck == ExternalConfirmed {
			t.Error("a commitment with no transaction id was reported confirmed")
		}
	})

	t.Run("horizon unavailable", func(t *testing.T) {
		horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})
		cx := stellarSample()
		r := &Report{}
		verifySingleStellar(r, &cx, Options{})
		assertStepStatus(t, r, "Stellar commitment unconfirmed", groupStellar, CatBlockchain, StatusSkip)
		if ci := onlyCommitment(t, r); ci.ExternalCheck == ExternalConfirmed {
			t.Error("an unreachable Horizon was reported as a confirmation")
		}
	})
}

// --- E.19: Bitcoin commitment grading ---

// TestBitcoinCommitment_AbsenceRowsAreSkipsNotPasses covers every E.19(c)
// absence branch: a piece of optional evidence the entry does not carry.
// E.5 forbids an absent optional field from failing a sound proof, and
// E.22 forbids the resulting row from reading as a check that succeeded —
// so each of these MUST be a skip, and promoting any of them to a pass is
// the defect E.22 exists to prevent. All fifteen survived the suite
// before this test.
func TestBitcoinCommitment_AbsenceRowsAreSkipsNotPasses(t *testing.T) {
	cases := []struct {
		name string
		mut  func(cx *proof.ExternalCommit)
		opts Options
		want string
	}{
		{
			name: "no offline evidence at all",
			mut: func(cx *proof.ExternalCommit) {
				cx.RawTxHex, cx.TxoutproofHex, cx.BlockMerkleRoot = "", "", ""
			},
			want: "carries no raw transaction, txoutproof or block merkle root",
		},
		{
			name: "no raw transaction",
			mut:  func(cx *proof.ExternalCommit) { cx.RawTxHex = "" },
			want: "OP_RETURN and txid checks skipped",
		},
		{
			name: "no txoutproof",
			mut:  func(cx *proof.ExternalCommit) { cx.TxoutproofHex = "" },
			want: "Bitcoin merkle proof skipped",
		},
		{
			name: "no block merkle root",
			mut:  func(cx *proof.ExternalCommit) { cx.BlockMerkleRoot = "" },
			want: "the entry carries no block merkle root (bmr)",
		},
		{
			name: "no txoutproof header to cross-check bmr against",
			mut:  func(cx *proof.ExternalCommit) { cx.TxoutproofHex = "" },
			want: "no txoutproof header to compare against",
		},
		{
			name: "no transaction id to compare the recomputed txid to",
			mut:  func(cx *proof.ExternalCommit) { cx.TransactionHash = "" },
			want: "Txid comparison skipped",
		},
		{
			name: "no transaction id to place in the matched set",
			mut:  func(cx *proof.ExternalCommit) { cx.TransactionHash = "" },
			want: "Bitcoin merkle proof placement skipped",
		},
		{
			name: "no block header recovered to confirm against the chain",
			mut:  func(cx *proof.ExternalCommit) { cx.TxoutproofHex = "" },
			want: "no block header was recovered",
		},
		{
			name: "no block height to confirm the header at",
			mut:  func(cx *proof.ExternalCommit) { cx.BlockHeight = 0; cx.Network = "mainnet" },
			want: "carries no block height (h)",
		},
		{
			name: "--skip-external",
			mut:  func(cx *proof.ExternalCommit) { cx.Network = "mainnet" },
			opts: Options{SkipExternal: true},
			want: "External Bitcoin verification skipped (--skip-external)",
		},
		{
			// regtest has no public API. E.19 grades that skip, and the
			// accompanying disclosure info row is what stops the skip
			// being read as evidence.
			name: "network with no public API",
			mut:  func(cx *proof.ExternalCommit) { cx.Network = "regtest" },
			want: "Bitcoin commitment unconfirmed: no public API for regtest",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cx := btcSample()
			tc.mut(&cx)
			r := &Report{}
			verifySingleBitcoin(r, &cx, tc.opts)
			assertStepStatus(t, r, tc.want, groupBitcoin, CatBlockchain, StatusSkip)
			if ci := onlyCommitment(t, r); ci.ExternalCheck == ExternalConfirmed {
				t.Errorf("commitment reported confirmed on the %q branch:\n%s", tc.name, stepMessages(r))
			}
		})
	}
}

// TestBitcoinCommitment_UnavailableChainIsSkipNotPass covers the arm the
// table above cannot reach without a stub: a named network whose explorer
// answers 5xx. E.18/E.19's settled rule routes transport failures to skip.
func TestBitcoinCommitment_UnavailableChainIsSkipNotPass(t *testing.T) {
	blockstreamStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	cx := btcSample()
	cx.Network = "mainnet"
	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{})
	assertStepStatus(t, r, "Bitcoin commitment unconfirmed: fetching Bitcoin block",
		groupBitcoin, CatBlockchain, StatusSkip)
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalSkipped {
		t.Errorf("ExternalCheck = %v, want ExternalSkipped", ci.ExternalCheck)
	}
}

// TestBitcoinCommitment_DefectsAreFails pins E.19(b)'s "present but
// unusable" arms. Each is a defect in bytes the bundle chose to carry, so
// unlike an absence it must fail: a skip here would let a bundle publish
// a corrupt txoutproof and still verify.
func TestBitcoinCommitment_DefectsAreFails(t *testing.T) {
	cases := []struct {
		name string
		mut  func(cx *proof.ExternalCommit)
		want string
	}{
		{
			// Valid hex, not a transaction: the parser reaches the
			// OP_RETURN scan and finds no output to read.
			name: "OP_RETURN extraction failure",
			mut:  func(cx *proof.ExternalCommit) { cx.RawTxHex = "00" },
			want: "OP_RETURN extraction failed",
		},
		{
			name: "txid computation failure",
			mut:  func(cx *proof.ExternalCommit) { cx.RawTxHex = "00" },
			want: "Txid computation failed",
		},
		{
			name: "txoutproof parse failure",
			mut:  func(cx *proof.ExternalCommit) { cx.TxoutproofHex = "00ff" },
			want: "Txoutproof parse failed",
		},
		{
			// A cx.tx that is not a usable transaction id: present, so
			// not an absence, and chainhash cannot look for it.
			name: "unusable transaction id for placement",
			mut:  func(cx *proof.ExternalCommit) { cx.TransactionHash = "not-a-txid" },
			want: "cx.tx is not a usable transaction id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cx := btcSample()
			tc.mut(&cx)
			r := &Report{}
			verifySingleBitcoin(r, &cx, Options{})
			assertStepStatus(t, r, tc.want, groupBitcoin, CatBlockchain, StatusFail)
			if r.Passed() {
				t.Errorf("the run passed despite %q:\n%s", tc.want, stepMessages(r))
			}
		})
	}
}

// TestBitcoinCommitment_ChainRefutationsAreFails pins E.19(b)'s binding
// step: an explorer that answers definitively and disagrees refutes the
// commitment. Both arms already carried a message; neither carried a
// pinned status, so both survived a downgrade to skip — which would turn
// a header the chain does not contain into a verified commitment.
func TestBitcoinCommitment_ChainRefutationsAreFails(t *testing.T) {
	t.Run("header not on chain", func(t *testing.T) {
		blockstreamStub(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		cx := btcSample()
		cx.Network = "mainnet"
		r := &Report{}
		verifySingleBitcoin(r, &cx, Options{})
		assertStepStatus(t, r, "Bitcoin commitment not on chain", groupBitcoin, CatBlockchain, StatusFail)
		if r.Passed() {
			t.Errorf("a header absent from the chain left the run passing:\n%s", stepMessages(r))
		}
	})

	t.Run("height disagrees", func(t *testing.T) {
		blockstreamStub(t, func(w http.ResponseWriter, req *http.Request) {
			blockstreamBlockBody(t, w, req, 999999, 1700000000)
		})
		cx := btcSample()
		cx.Network = "mainnet"
		r := &Report{}
		verifySingleBitcoin(r, &cx, Options{})
		assertStepStatus(t, r, "Bitcoin block height mismatch", groupBitcoin, CatBlockchain, StatusFail)
		if r.Passed() {
			t.Errorf("a height the chain contradicts left the run passing:\n%s", stepMessages(r))
		}
	})
}

// mutatedTxOutProofHex is btcSample()'s txoutproof with the first nibble
// of its first hash flipped (0x82... -> 0x92...). That hash is the
// sibling, not the matched txid, so the structure still parses, cx.tx is
// still placed in the matched set, and the header — hence the block hash
// and the bmr cross-check — is untouched. The ONLY thing that changes is
// that the partial tree no longer derives the header's merkle root.
const mutatedTxOutProofHex = "0000002026cf000a60782b1b57c018b6482e6837c1165d5e2aa06f92d199163481b6c07b" +
	"150edc6623b16f07e3239de42b6f67d08f5d71b4d06e35685dcaab533f951d5e395ab869ffff7f2000000000" +
	"02000000029205b7caabd985090d0ba024a7d2ce998e53d49122989f8eac68b6f25558bbee" +
	"52877124271894f27729ada539c4834cfdf4b3972d39a0e189ff108c578ec3290105"

// TestBitcoinPartialMerkleTree_InvalidWalkFails pins E.19(b) step 5: the
// partial Merkle tree MUST derive the txoutproof header's merkle root.
// Forcing that condition to a literal true left the whole suite green, so
// the one cryptographic binding between the txoutproof's hash list and
// its header was an unconditional pass as far as the tests were concerned.
func TestBitcoinPartialMerkleTree_InvalidWalkFails(t *testing.T) {
	cx := btcSample()
	cx.TxoutproofHex = mutatedTxOutProofHex

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{})

	assertStepStatus(t, r, "Bitcoin partial merkle proof invalid",
		groupBitcoin, CatBlockchain, StatusFail)
	if r.Passed() {
		t.Errorf("a partial merkle tree that derives the wrong root left the run passing:\n%s", stepMessages(r))
	}

	// The mutation is surgical on purpose: if it also broke the header
	// the failure above would be over-determined and would not pin the
	// walk. The header-derived rows must still be the healthy ones.
	if !strings.Contains(stepMessages(r), "Bitcoin block merkle root matches the txoutproof header") {
		t.Errorf("the bmr cross-check should be unaffected by a sibling-hash mutation:\n%s", stepMessages(r))
	}
	if !strings.Contains(stepMessages(r), "is in the txoutproof matched set") {
		t.Errorf("placement should be unaffected by a sibling-hash mutation:\n%s", stepMessages(r))
	}

	// Control: the unmutated fixture passes the same row. Without this
	// the test above would also pass if the walk always failed.
	healthy := btcSample()
	rh := &Report{}
	verifySingleBitcoin(rh, &healthy, Options{})
	assertStepStatus(t, rh, "Bitcoin partial merkle tree derives the txoutproof header merkle root",
		groupBitcoin, CatBlockchain, StatusPass)
}

// TestBitcoinRecomputedTxid_MustEqualCxTx pins E.19(b) step 2. Forcing
// its condition to a literal true left the whole suite green: the sample
// bundle's tampered cx.tx is also caught by the redundant placement check
// at step 5, so an end-to-end assertion on the verdict cannot tell the
// two apart. This asserts the step 2 row itself, which is the only thing
// that distinguishes "the raw transaction hashes to cx.tx" from "some
// other row objected".
func TestBitcoinRecomputedTxid_MustEqualCxTx(t *testing.T) {
	cx := btcSample()
	// A well-formed txid that is not this transaction's. Same shape, so
	// nothing earlier rejects it as unusable.
	cx.TransactionHash = strings.Repeat("ab", 32)

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{})

	assertStepStatus(t, r, "Recomputed txid does NOT match cx.tx",
		groupBitcoin, CatBlockchain, StatusFail)
	if r.Passed() {
		t.Errorf("a raw transaction that does not hash to cx.tx left the run passing:\n%s", stepMessages(r))
	}

	// Control: the same fixture untouched produces the pass row, so the
	// assertion above cannot be satisfied by a step 2 that always fails.
	healthy := btcSample()
	rh := &Report{}
	verifySingleBitcoin(rh, &healthy, Options{})
	assertStepStatus(t, rh, "verified from raw tx", groupBitcoin, CatBlockchain, StatusPass)
}

// --- E.17: keyring cross-check ---

// TestKeyBinding_NoKeyringPublished_IsSkipNotPass pins E.17's 404 arm.
// E.25 lists the keyring cross-check among the steps a run may omit and
// still be called verified, which is exactly why the omission must not
// read as a completed check: promoting this row to a pass would publish
// "this key is in the published keyring" for a host that publishes no
// keyring at all.
func TestKeyBinding_NoKeyringPublished_IsSkipNotPass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := &Report{}
	verifyKeyBinding(r, testPubKeyB64, decodeTestPubKey(t), testKeyID, Options{KeyringURL: srv.URL})

	assertStepStatus(t, r, "no keyring published at", groupKeyBinding, CatCryptographic, StatusSkip)
	// The disclosure row is what stops the skip reading as a completed
	// cross-check; a pass would make it self-contradictory.
	if !strings.Contains(stepMessages(r), keyBindingUnestablishedNote) {
		t.Errorf("the unestablished-binding disclosure is missing:\n%s", stepMessages(r))
	}
}

// --- F18 / E.22: no fail message reads as a positive assertion ---

// positiveAssertionFragments are the phrases this verifier uses to state
// that a check SUCCEEDED, taken from the pass arms of the pipeline's own
// r.check / r.pass call sites. None of them may appear in a message a
// caller is shown as the REASON a proof was refused: E.22 forbids a
// result from reading as an assertion the run did not establish, and the
// sharpest version of that is a rejection worded like a confirmation.
//
// The list is a conservative allow-list complement, not a grammar. It
// catches the concrete defect F18 exists to prevent — a call site that
// hands check() the same string for both arms, or a fail message written
// by copying the pass message and forgetting the negation — which is a
// real mistake the pipeline made before this pass. Each fragment includes
// the surrounding spacing or punctuation that keeps a legitimate negation
// ("does NOT match cx.bmr", "Proof signature invalid (Ed25519)") from
// tripping it.
var positiveAssertionFragments = []string{
	" matches ",
	"verified from raw tx",
	"confirmed on",
	" valid (Ed25519)",
	"derives the txoutproof header merkle root",
	"is in the txoutproof matched set",
	"maps to committed value",
	" derived (0x",
	"Public key valid,",
}

// TestFailMessages_NeverReadAsPositiveAssertions collects every fail
// message the pipeline emits across a broad corpus — the Appendix D
// tamper matrix plus the crafted defect scenarios in this file — and
// asserts none of them reads as a confirmation.
//
// This is the pipeline-wide property report_test.go's F18 primitive test
// only claimed to have. The corpus is deliberately assembled from
// scenarios that are individually asserted elsewhere, so a branch that
// stops being reached shows up there rather than silently shrinking the
// corpus here; the count assertion below guards against the corpus
// collapsing to nothing.
func TestFailMessages_NeverReadAsPositiveAssertions(t *testing.T) {
	var failures []string

	collect := func(r *Report) {
		for _, s := range r.Steps {
			if s.Status == StatusFail {
				failures = append(failures, s.Message)
			}
		}
	}

	// 1. Every tamper of the published conformance vector.
	base := fixtureBytes(t, "appendix-d-item.json")
	for _, mutate := range []func(map[string]any){
		func(m map[string]any) { m["s"].(map[string]any)["mh"] = strings.Repeat("ab", 32) },
		func(m map[string]any) { m["s"].(map[string]any)["kid"] = "deadbeef" },
		func(m map[string]any) { m["b"].(map[string]any)["mr"] = strings.Repeat("cd", 32) },
		func(m map[string]any) { m["b"].(map[string]any)["ph"] = strings.Repeat("ef", 32) },
		func(m map[string]any) { m["cx"].([]any)[0].(map[string]any)["memo"] = strings.Repeat("23", 32) },
		func(m map[string]any) { m["ts"] = "2026-07-24T12:00:01Z" },
		func(m map[string]any) { m["v"] = 2 },
		func(m map[string]any) {
			ip := m["ip"].(string)
			m["ip"] = ip[:len(ip)-8]
		},
	} {
		report, err := RunFromBytes(mutateBundle(t, base, mutate), "tamper.json", Options{
			SkipExternal: true,
			ExpectedHash: "0000000000000000000000000000000000000000000000000000000000000000",
		})
		if err != nil {
			continue // an E.6 hard rejection prints no steps at all
		}
		collect(report)
	}

	// 2. The Bitcoin and entropy defect scenarios, which the tamper
	//    matrix cannot reach: the D.4 vector's t=41 entry carries no
	//    offline evidence and its entropy subject is an item.
	for _, mut := range []func(cx *proof.ExternalCommit){
		func(cx *proof.ExternalCommit) { cx.RawTxHex = "00" },
		func(cx *proof.ExternalCommit) { cx.TxoutproofHex = "00ff" },
		func(cx *proof.ExternalCommit) { cx.TransactionHash = "not-a-txid" },
		func(cx *proof.ExternalCommit) { cx.TransactionHash = strings.Repeat("ab", 32) },
		func(cx *proof.ExternalCommit) { cx.OpReturn = strings.Repeat("cd", 32) },
		func(cx *proof.ExternalCommit) { cx.BlockMerkleRoot = strings.Repeat("ef", 32) },
		func(cx *proof.ExternalCommit) { cx.TxoutproofHex = mutatedTxOutProofHex },
	} {
		cx := btcSample()
		mut(&cx)
		r := &Report{}
		verifySingleBitcoin(r, &cx, Options{})
		collect(r)
	}
	for _, raw := range []string{
		`["not","an","object"]`,
		`{"chainIndex":1,"pulseIndex":2}`,
	} {
		r := &Report{}
		verifyEntropyNIST(r, json.RawMessage(raw))
		collect(r)
	}

	// A corpus that shrank to nothing would make every assertion below
	// vacuously true, which is the failure mode this whole file exists
	// to close.
	if len(failures) < 15 {
		t.Fatalf("corpus collapsed: only %d fail messages collected", len(failures))
	}

	for _, msg := range failures {
		for _, frag := range positiveAssertionFragments {
			if strings.Contains(msg, frag) {
				t.Errorf("a fail message reads as a positive assertion (contains %q): %q", frag, msg)
			}
		}
	}
	t.Logf("checked %d distinct fail messages", len(failures))
}

// --- E.22: confirmations are passes, disclosures are info ---

// TestExternalConfirmations_ArePasses pins the positive arms. Demoting a
// confirmed on-chain lookup to an info leaves the run passing and the
// message intact, so nothing but a status assertion notices — yet the
// report then reports a completed external check as a neutral note and
// the summary's "N of M checks passed" silently loses a check.
func TestExternalConfirmations_ArePasses(t *testing.T) {
	t.Run("stellar entropy ledger", func(t *testing.T) {
		horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"hash":"` + strings.Repeat("ab", 32) +
				`","sequence":51234567,"closed_at":"2026-04-22T20:00:02Z"}`))
		})
		r := &Report{}
		verifyEntropyStellar(r, json.RawMessage(
			`{"hash":"`+strings.Repeat("ab", 32)+`","sequence":51234567,"closed_at":"2026-04-22T20:00:02Z"}`), "testnet")
		assertStepStatus(t, r, "confirmed on testnet", groupEntropySource, CatBlockchain, StatusPass)
	})

	t.Run("bitcoin entropy block with no comparable field", func(t *testing.T) {
		blockstreamStub(t, func(w http.ResponseWriter, req *http.Request) {
			blockstreamBlockBody(t, w, req, 850000, 1700000000)
		})
		r := &Report{}
		// Neither `height` nor `time`, so the hash is the only operand:
		// the arm that reports what it could confirm and names what it
		// could not, without asserting a height the bundle never claimed.
		verifyEntropyBitcoin(r, json.RawMessage(
			`{"hash":"`+strings.Repeat("ab", 32)+`"}`), "mainnet")
		assertStepStatus(t, r, "the subject carries no height or time to compare against it",
			groupEntropySource, CatBlockchain, StatusPass)
	})

	t.Run("stellar commitment", func(t *testing.T) {
		cx := stellarSample()
		horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(horizonTxBody(t, cx.TransactionHash, cx.MemoHash, cx.Ledger)))
		})
		r := &Report{}
		verifySingleStellar(r, &cx, Options{})
		assertStepStatus(t, r, "confirmed on testnet", groupStellar, CatBlockchain, StatusPass)
		if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalConfirmed {
			t.Errorf("ExternalCheck = %v, want ExternalConfirmed", ci.ExternalCheck)
		}
	})

	t.Run("bitcoin commitment", func(t *testing.T) {
		blockstreamStub(t, func(w http.ResponseWriter, req *http.Request) {
			blockstreamBlockBody(t, w, req, 9312, 1700000000)
		})
		cx := btcSample()
		cx.Network = "mainnet"
		r := &Report{}
		verifySingleBitcoin(r, &cx, Options{})
		assertStepStatus(t, r, "confirmed on mainnet", groupBitcoin, CatBlockchain, StatusPass)
		if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalConfirmed {
			t.Errorf("ExternalCheck = %v, want ExternalConfirmed", ci.ExternalCheck)
		}
	})

	t.Run("observation hash 0x23", func(t *testing.T) {
		r := &Report{}
		got := deriveObservationHash(r, &proof.Subject{
			ID:           "019cf813-99b8-730a-84f1-5a711a9c355e",
			MetadataHash: strings.Repeat("11", 32),
			SigningKeyID: "aabbccdd",
		}, strings.Repeat("22", 32))
		if got == "" {
			t.Fatalf("no observation hash derived:\n%s", stepMessages(r))
		}
		assertStepStatus(t, r, "Observation hash derived (0x23)", groupSubjectData, CatCryptographic, StatusPass)
	})
}

// TestDisclosureRows_AreInfoNotPass pins the other direction. Each of
// these rows exists to say a binding was NOT established; publishing it
// as a pass would both assert something the run did not establish (E.22)
// and add a pass row E.25's containment forbids.
func TestDisclosureRows_AreInfoNotPass(t *testing.T) {
	t.Run("key binding: no keyring published", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()
		r := &Report{}
		verifyKeyBinding(r, testPubKeyB64, decodeTestPubKey(t), testKeyID, Options{KeyringURL: srv.URL})
		assertStepStatus(t, r, keyBindingUnestablishedNote, groupKeyBinding, CatCryptographic, StatusInfo)
	})

	t.Run("key binding: keyring unreachable", func(t *testing.T) {
		r := &Report{}
		verifyKeyBinding(r, testPubKeyB64, decodeTestPubKey(t), testKeyID,
			Options{KeyringURL: "http://127.0.0.1:1/nonexistent"})
		assertStepStatus(t, r, keyBindingUnestablishedNote, groupKeyBinding, CatCryptographic, StatusInfo)
	})

	t.Run("bitcoin: rewritten net suppresses the lookup", func(t *testing.T) {
		cx := btcSample()
		cx.Network = "regtest"
		r := &Report{}
		verifySingleBitcoin(r, &cx, Options{})
		assertStepStatus(t, r, bitcoinNetworkDowngradeNote, groupBitcoin, CatBlockchain, StatusInfo)
	})

	t.Run("submission window: the submitted-before edge", func(t *testing.T) {
		r := &Report{}
		reportSubmissionWindowEdges(r, []proof.ExternalCommit{{
			Type: ptype.CommitmentStellar, Timestamp: "2026-07-24T12:00:12Z",
		}})
		assertStepStatus(t, r, "confirming that transaction on chain establishes the submitted-before edge",
			groupSubmittedBefore, CatTiming, StatusInfo)
	})

	t.Run("submission window: no commitment timestamp", func(t *testing.T) {
		r := &Report{}
		reportSubmissionWindowEdges(r, []proof.ExternalCommit{{Type: ptype.CommitmentStellar}})
		assertStepStatus(t, r, "no cx entry carries a transaction timestamp",
			groupSubmittedBefore, CatTiming, StatusInfo)
	})
}
