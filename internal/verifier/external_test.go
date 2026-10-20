// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// The external gate and the rows of CHAIN and SOURCES under a closed gate
// (SPEC.md sections 10.2, the external gate, 10.3, the key-history rows,
// and 10.4, "Rows" and "Offline"). The runs are deriveRun's, past SHAPE,
// over the positive proofs, each signed (as far as DERIVE reads) by
// VECTOR_KEY under a supplied keyring's attribution.

// throughDerive is the run of proof past DERIVE, online when online is
// true, with extra added after SHAPE's row.
func throughDerive(proof map[string]any, online bool, extra ...Row) *Run {
	run := deriveRun(proof, deriveAttribution)
	run.Config.Online = online
	for _, row := range extra {
		run.AddRow(row)
	}
	Derive(run)
	return run
}

// stepRows writes the run's rows of step as rowSummary does.
func stepRows(run *Run, step string) string {
	var rows []Row
	for _, row := range run.Rows {
		if row.Step == step {
			rows = append(rows, row)
		}
	}
	return rowSummary(rows)
}

// policyFailure is a failing POLICY row.
var policyFailure = NewRow("POLICY", specdata.ReasonNetworkMismatch, Part("/commitments/bitcoin/network"), "another network", nil)

// The gate is decided once, when DERIVE ends: open when the caller did not
// ask for an offline run and no row so far is fail; offline when the
// caller asked for one and none is; and prior_failure when a row before
// CHAIN failed, at POLICY or at DERIVE, online or offline. An info row is
// no failure. Mutants killed: decideExternalGate ignoring Config.Online
// (the online sound run is offline); ignoring a failed row (the online
// runs with a failure open); reading only DERIVE's rows (the POLICY
// failure opens the gate); counting an info row as a failure (each sound
// run, whose LINK_KEY_EVENT row is info not_attributable, is
// prior_failure).
func TestExternalGate_DecidedWhenDeriveEnds(t *testing.T) {
	sound := positiveProofs(t)["item"]
	broken := put(sound, Pointer{Name("subject"), Name("item_hash")}, strings.Repeat("ab", 32))
	for _, c := range []struct {
		name   string
		proof  map[string]any
		online bool
		extra  []Row
		want   ExternalGate
	}{
		{"sound, online", sound, true, nil, ExternalGateOpen},
		{"sound, offline", sound, false, nil, ExternalGateOffline},
		{"a DERIVE failure, online", broken, true, nil, ExternalGatePriorFailure},
		{"a DERIVE failure, offline", broken, false, nil, ExternalGatePriorFailure},
		{"a POLICY failure, online", sound, true, []Row{policyFailure}, ExternalGatePriorFailure},
		{"a POLICY failure, offline", sound, false, []Row{policyFailure}, ExternalGatePriorFailure},
	} {
		run := throughDerive(c.proof, c.online, c.extra...)
		if run.ExternalGate != c.want {
			t.Errorf("%s: the gate is %q, want %q", c.name, run.ExternalGate, c.want)
		}
		if c.want == ExternalGatePriorFailure {
			continue
		}
		infos := slices.ContainsFunc(run.Rows, func(row Row) bool {
			return row.Status == StatusInfo && row.Code == specdata.ReasonNotAttributable
		})
		if !infos {
			t.Errorf("%s: the sound run has no info row", c.name)
		}
	}
}

// Under a closed gate, CHAIN gives one row for each commitment entry and
// each genesis.bitcoin, at its pointer, in document order, then
// KEY_EVENT_COMMITMENT at /signing_key_event, and SOURCES one row for each
// entropy object, in document order; each row is skip offline when the
// caller asked for an offline run and no row failed, and skip
// prior_failure when a row before CHAIN failed. block's genesis.bitcoin at
// /block comes before its commitment entries, and item-genesis-witness's
// genesis.bitcoin at /signing_key_event/block before that key event's
// entries and its witness block's after them. Mutants killed: Chain
// without its sort (block's genesis row comes after the entries); Chain
// leaving out genesis.bitcoin (no genesis row); KEY_EVENT_COMMITMENT
// placed among the entries by document order (it comes before the
// /signing_key_event/commitments rows); closedRow giving offline for
// prior_failure (the failed runs' rows are skip offline); Sources reading
// the commitment entries' positions (its parts are wrong).
func TestChainAndSources_ClosedGateRows(t *testing.T) {
	proofs := positiveProofs(t)
	for _, c := range []struct {
		proof          string
		chain, sources []string
	}{
		{"item", []string{"/commitments/bitcoin", "/commitments/stellar", "/signing_key_event/commitments/bitcoin", "/signing_key_event/commitments/stellar", "/signing_key_event"},
			[]string{"/witnesses/entropy_bitcoin", "/witnesses/entropy_nist", "/witnesses/entropy_stellar"}},
		{"block", []string{"/block/record/metadata/genesis/bitcoin", "/commitments/bitcoin", "/commitments/stellar", "/signing_key_event/commitments/bitcoin", "/signing_key_event/commitments/stellar", "/signing_key_event"},
			nil},
		{"item-genesis-witness", []string{"/commitments/stellar", "/signing_key_event/block/record/metadata/genesis/bitcoin", "/signing_key_event/commitments/stellar",
			"/witnesses/block/record/metadata/genesis/bitcoin", "/signing_key_event"}, nil},
		{"beacon", []string{"/commitments/bitcoin", "/commitments/stellar", "/signing_key_event/commitments/bitcoin", "/signing_key_event/commitments/stellar", "/signing_key_event"},
			[]string{"/observations/entropy_bitcoin/entropy", "/observations/entropy_nist/entropy", "/observations/entropy_stellar/entropy"}},
		{"entropy_nist", []string{"/commitments/bitcoin", "/commitments/stellar", "/signing_key_event/commitments/bitcoin", "/signing_key_event/commitments/stellar", "/signing_key_event"},
			[]string{"/subject/entropy"}},
	} {
		for _, failed := range []bool{false, true} {
			var extra []Row
			code := "offline"
			if failed {
				extra, code = []Row{policyFailure}, "prior_failure"
			}
			run := throughDerive(proofs[c.proof], false, extra...)
			Chain(run)
			Sources(run)
			summary := func(step string, parts []string) string {
				var out []string
				for _, part := range parts {
					out = append(out, "["+code+" "+part+" skip "+step+"]")
				}
				return strings.Join(out, " ")
			}
			if got, want := stepRows(run, "CHAIN"), summary("CHAIN", c.chain); got != want {
				t.Errorf("%s, failed %v: CHAIN %s\nwant %s", c.proof, failed, got, want)
			}
			if got, want := stepRows(run, "SOURCES"), summary("SOURCES", c.sources); got != want {
				t.Errorf("%s, failed %v: SOURCES %s\nwant %s", c.proof, failed, got, want)
			}
			if run.Ended {
				t.Errorf("%s: the run ended", c.proof)
			}
		}
	}
}

// KEY_EVENT_COMMITMENT is reported only when P carries signing_key_event
// (SPEC.md section 10.3). Mutant killed: Chain adding KEY_EVENT_COMMITMENT
// whatever P carries (the proof without signing_key_event has one).
func TestChain_NoKeyEventRowWithoutASigningKeyEvent(t *testing.T) {
	proof := clone(positiveProofs(t)["item"]).(map[string]any)
	delete(proof, "signing_key_event")
	run := throughDerive(proof, false)
	Chain(run)
	if got := stepRows(run, "CHAIN"); got != "[offline /commitments/bitcoin skip CHAIN] [offline /commitments/stellar skip CHAIN]" {
		t.Fatalf("CHAIN %s", got)
	}
}

// A skip offline row's message names the read that would confirm it, each
// value it names written whole, its integers in decimal; a skip
// prior_failure row's says a check failed; neither row has values. Mutants
// killed: carried writing a number as fmt.Sprint does (the ledger reads
// 1.000255e+06); entropyRead naming the commitment pin's network for a
// Stellar entropy object's read (its message names testnet); commitmentRead
// giving a signing key event's entry the candidate read (its message names
// the ledger's close time); closedRow writing the offline message on a
// prior_failure row.
func TestChainAndSources_OfflineMessagesNameTheRead(t *testing.T) {
	proof := positiveProofs(t)["item"]
	run := throughDerive(proof, false)
	run.Config.Networks = specdata.NetworkPins{
		Commitment: specdata.ChainNetworks{Bitcoin: "regtest", Stellar: "testnet"},
		Entropy:    specdata.ChainNetworks{Bitcoin: "mainnet", Stellar: "public"},
	}
	Chain(run)
	Sources(run)
	stellar := valueAt(proof, Pointer{Name("commitments"), Name("stellar")}).(map[string]any)
	payload := func(source string, name string) string {
		return carried(valueAt(proof, Pointer{Name("witnesses"), Name("entropy_" + source), Name("payload"), Name(name)}))
	}
	want := map[string][]string{
		"/commitments/stellar":                   {stellar["transaction_hash"].(string), "ledger 1000255", "Stellar testnet", "close time"},
		"/commitments/bitcoin":                   {"height 222", "Bitcoin regtest", "median time past"},
		"/signing_key_event/commitments/stellar": {"ledger 1000255"},
		"/signing_key_event":                     {"/signing_key_event"},
		"/witnesses/entropy_stellar":             {"ledger " + payload("stellar", "sequence"), "Stellar public"},
		"/witnesses/entropy_bitcoin":             {payload("bitcoin", "hash"), "Bitcoin mainnet"},
		"/witnesses/entropy_nist":                {"chainIndex 2", "pulseIndex 1200018"},
	}
	seen := 0
	for _, row := range run.Rows {
		if row.Step != "CHAIN" && row.Step != "SOURCES" {
			continue
		}
		if len(row.Values) != 0 || !strings.HasPrefix(row.Message, "Not read: this run is offline. Confirming it needs ") {
			t.Errorf("%s: values %v, message %q", *row.Part, row.Values, row.Message)
		}
		for _, text := range want[*row.Part] {
			seen++
			if !strings.Contains(row.Message, text) {
				t.Errorf("%s: the message %q does not name %q", *row.Part, row.Message, text)
			}
		}
		if *row.Part == "/signing_key_event/commitments/stellar" && strings.Contains(row.Message, "close time") {
			t.Errorf("a signing key event's entry gives no candidate: %q", row.Message)
		}
	}
	if seen != 15 {
		t.Fatalf("%d named values checked", seen)
	}
	failed := throughDerive(proof, false, policyFailure)
	Chain(failed)
	for _, row := range failed.Rows {
		if row.Step == "CHAIN" && (row.Message != "Not read: a check before CHAIN failed, so this run reads no chain or source." || len(row.Values) != 0) {
			t.Errorf("%s: %q, %v", *row.Part, row.Message, row.Values)
		}
	}
}

// Under an open gate CHAIN gives its rows, and a failure CHAIN itself
// gives does not close the gate for SOURCES: nothing decides it again, so
// SOURCES reads each entropy object and gives its row, no skip
// prior_failure row among them. WINDOW's rule is not in this package, so
// it gives no row here. The item proof's two Stellar entries name one
// transaction, so both fail on its one answer, and so does
// KEY_EVENT_COMMITMENT; its Bitcoin entries pass, and so do its three
// witnesses, each read from the sources of the entropy pins, the Stellar
// witness from the source whose transaction answer failed CHAIN. Mutants
// killed: Chain deciding the gate again when it ends (SOURCES gives skip
// prior_failure rows after CHAIN's failure); closedGate deciding from
// Run.Failed (the same); Chain treating an open gate as offline (CHAIN
// gives skip offline rows); Sources giving no row under an open gate (no
// SOURCES row follows); Window drawing its row under an open gate (a
// WINDOW row follows).
func TestExternalGate_AChainFailureLeavesItOpenForSources(t *testing.T) {
	item := positiveProofs(t)["item"]
	stand := &chainStandIn{answers: map[string]SourceRead{}}
	stand.answer(soundStellar(t, item, topStellar, itemClosed), soundTx(map[string]any{"successful": false}), SourceRead{}, "a")
	stand.answerEntropy(t, entropyEntryAt(t, item, witnessNIST), "", SourceRead{})
	stand.answerEntropy(t, entropyEntryAt(t, item, witnessStellar), "testnet", SourceRead{}, "a")
	stand.answerEntropy(t, entropyEntryAt(t, item, witnessBitcoin), "regtest", SourceRead{}, "btc")
	run := openItemRun(t, item, deriveAttribution, standInSet(SourcePolicy{Kind: SourcePolicyFirst}, "a"), stand)
	run.Config.Networks.Entropy = specdata.ChainNetworks{Bitcoin: "regtest", Stellar: "testnet"}
	before := len(run.Rows)
	Chain(run)
	Sources(run)
	Window(run)
	want := "[null /commitments/bitcoin pass CHAIN] [chain_mismatch /commitments/stellar fail CHAIN] [null /signing_key_event/commitments/bitcoin pass CHAIN] " +
		"[chain_mismatch /signing_key_event/commitments/stellar fail CHAIN] [chain_mismatch /signing_key_event fail CHAIN] " +
		"[null /witnesses/entropy_bitcoin pass SOURCES] [null /witnesses/entropy_nist pass SOURCES] [null /witnesses/entropy_stellar pass SOURCES]"
	if got := rowSummary(run.Rows[before:]); got != want || run.ExternalGate != ExternalGateOpen {
		t.Fatalf("rows after DERIVE %s, gate %q; want %s", got, run.ExternalGate, want)
	}
}

// CHAIN, SOURCES and WINDOW follow DERIVE, so a run whose gate is
// undecided is a defect in the caller, and each panics. Mutant killed:
// closedGate taking an undecided gate as offline (no panic, and skip
// offline rows).
func TestExternalGate_UndecidedPanics(t *testing.T) {
	for name, step := range map[string]func(*Run){"CHAIN": Chain, "SOURCES": Sources, "WINDOW": Window} {
		run := deriveRun(positiveProofs(t)["item"], deriveAttribution)
		func() {
			defer func() {
				if r := recover(); r == nil || !strings.Contains(r.(string), name+" before DERIVE decided the external gate") {
					t.Errorf("%s: recovered %v", name, r)
				}
			}()
			step(run)
		}()
	}
}

// A run that has ended passes through CHAIN, SOURCES, WINDOW and
// EXPECTED_HASH unchanged, its rows of those steps already skip
// prior_failure, part null. Mutant killed: any of the four without its
// ended-run guard (the ended run gains rows, or panics on its undecided
// gate).
func TestLaterSteps_AnEndedRunIsPassedThrough(t *testing.T) {
	run := deriveRun(positiveProofs(t)["item"], deriveAttribution)
	run.Rows = run.Rows[:len(run.Rows)-1]
	run.AddRow(NewRow("SHAPE", specdata.ReasonUnsupportedFormat, Part("/format"), "no", nil))
	run.EndRun()
	hash := "69e3941192b086bfdc0e4e120c18ff12"
	run.Config.ExpectedHash = &hash
	rows := slices.Clone(run.Rows)
	Derive(run)
	Chain(run)
	Sources(run)
	Window(run)
	ExpectedHash(run)
	if rowSummary(run.Rows) != rowSummary(rows) || run.ExternalGate != "" {
		t.Fatalf("rows %s, gate %q", rowSummary(run.Rows), run.ExternalGate)
	}
}
