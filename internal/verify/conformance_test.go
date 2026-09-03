// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// rowKey is the (group, status) pair Appendix E.25's containment is checked
// on.
type rowKey struct {
	Group  string
	Status Status
}

func (k rowKey) String() string { return fmt.Sprintf("%s/%s", k.Group, k.Status) }

func keysOf(steps []Step) map[rowKey]int {
	m := map[rowKey]int{}
	for _, s := range steps {
		m[rowKey{s.Group, s.Status}]++
	}
	return m
}

// assertContainment applies E.25's one-way containment: every row of want
// must appear in got with the same status (as a multiset), and got may add
// only skip and info rows beyond want. except names groups left out of
// the comparison.
func assertContainment(t *testing.T, got, want []Step, except ...string) {
	t.Helper()
	skip := map[string]bool{}
	for _, g := range except {
		skip[g] = true
	}
	gotKeys, wantKeys := keysOf(got), keysOf(want)
	for k, n := range wantKeys {
		if skip[k.Group] {
			continue
		}
		if gotKeys[k] < n {
			t.Errorf("missing or differently graded row: want %d x %s, got %d\n%s", n, k, gotKeys[k], render(got))
		}
	}
	for k, n := range gotKeys {
		if skip[k.Group] || k.Status == StatusSkip || k.Status == StatusInfo {
			continue
		}
		if wantKeys[k] < n {
			t.Errorf("additive %s row beyond the reference: got %d x %s, want %d\n%s", k.Status, n, k, wantKeys[k], render(got))
		}
	}
}

func render(steps []Step) string {
	var b strings.Builder
	for _, s := range steps {
		fmt.Fprintf(&b, "  %s %-20s %s\n", s.Status.Badge(), s.Group, s.Message)
	}
	return b.String()
}

var referenceRow = regexp.MustCompile(`^  \[(PASS|FAIL|WARN|SKIP|INFO)\]  (.{20}) (.*)$`)

// parseReferenceReport reads the reference verifier's printed report back
// into steps: a category heading line, then rows of badge, a 20-column
// group and the message.
func parseReferenceReport(t *testing.T, text string) []Step {
	t.Helper()
	var steps []Step
	category := ""
	for _, line := range strings.Split(text, "\n") {
		switch strings.TrimSpace(line) {
		case "Data Integrity":
			category = CatDataIntegrity
			continue
		case "Cryptographic":
			category = CatCryptographic
			continue
		case "Structural":
			category = CatStructural
			continue
		case "Timing":
			category = CatTiming
			continue
		case "Blockchain":
			category = CatBlockchain
			continue
		}
		m := referenceRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		status, err := StatusFromString(strings.ToLower(m[1]))
		if err != nil {
			t.Fatal(err)
		}
		steps = append(steps, Step{Group: strings.TrimSpace(m[2]), Status: status, Category: category, Message: m[3]})
	}
	if len(steps) == 0 {
		t.Fatal("no rows parsed from the reference report")
	}
	return steps
}

// parseServerReport reads the server's /proof/verify result.
func parseServerReport(t *testing.T, data []byte) (steps []Step, passed bool) {
	t.Helper()
	var env struct {
		Result struct {
			Passed bool   `json:"passed"`
			Steps  []Step `json:"steps"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatal(err)
	}
	return env.Result.Steps, env.Result.Passed
}

func prodBundle(t *testing.T, name string) []byte {
	t.Helper()
	return testfixtures.Read(t, testfixtures.ProdDir, name)
}

func offlineWithKeyring() Options {
	return Options{SkipExternal: true, KeyringFile: testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdKeyring)}
}

func mustRun(t *testing.T, data []byte, opts Options) *Report {
	t.Helper()
	r, err := RunFromBytes(data, "test", opts)
	if err != nil {
		t.Fatalf("RunFromBytes: %v", err)
	}
	return r
}

func assertCounts(t *testing.T, r *Report, pass, fail, warn, skip, info int) {
	t.Helper()
	c := r.Counts()
	if c.Passed != pass || c.Failed != fail || c.Warned != warn || c.Skipped != skip || c.Info != info {
		t.Errorf("counts = %d pass %d fail %d warn %d skip %d info, want %d/%d/%d/%d/%d\n%s",
			c.Passed, c.Failed, c.Warned, c.Skipped, c.Info, pass, fail, warn, skip, info, render(r.Steps))
	}
}

// TestConformance_ProductionOffline runs the three production bundles
// offline with the production keyring pinned and checks E.25 containment
// against the reference verifier's own reports, plus the exact counts the
// fixtures document.
func TestConformance_ProductionOffline(t *testing.T) {
	cases := []struct {
		bundle, report               string
		pass, fail, warn, skip, info int
	}{
		{testfixtures.ProdComplete, "standalone-complete.txt", 21, 0, 0, 6, 8},
		{testfixtures.ProdCompact, "standalone-compact.txt", 12, 0, 0, 7, 3},
		{testfixtures.ProdPartial, "standalone-partial.txt", 15, 0, 0, 6, 5},
	}
	for _, tc := range cases {
		t.Run(tc.bundle, func(t *testing.T) {
			r := mustRun(t, prodBundle(t, tc.bundle), offlineWithKeyring())
			want := parseReferenceReport(t, string(prodBundle(t, tc.report)))
			assertContainment(t, r.Steps, want)
			if !r.Passed() {
				t.Errorf("verdict FAILED, want PASSED\n%s", render(r.Steps))
			}
			assertCounts(t, r, tc.pass, tc.fail, tc.warn, tc.skip, tc.info)
			// The rows are meant to line up one for one with the reference,
			// group and status alike, in category order.
			gotOrdered := orderedSteps(r)
			if len(gotOrdered) != len(want) {
				t.Errorf("row count %d, reference %d", len(gotOrdered), len(want))
			} else {
				for i := range want {
					if gotOrdered[i].Group != want[i].Group || gotOrdered[i].Status != want[i].Status {
						t.Errorf("row %d: got %s/%s, reference %s/%s", i, gotOrdered[i].Group, gotOrdered[i].Status, want[i].Group, want[i].Status)
					}
				}
			}
		})
	}
}

// TestConformance_ProductionOffline_NoKeyring pins that without a keyring
// the Key Binding row is a skip and nothing else moves.
func TestConformance_ProductionOffline_NoKeyring(t *testing.T) {
	r := mustRun(t, prodBundle(t, testfixtures.ProdComplete), Options{SkipExternal: true})
	found := false
	for _, s := range r.Steps {
		if s.Group == groupKeyBinding {
			found = true
			if s.Status != StatusSkip || !strings.Contains(s.Message, "no keyring supplied") {
				t.Errorf("Key Binding without a keyring = %s %q", s.Status, s.Message)
			}
		}
	}
	if !found || !r.Passed() {
		t.Errorf("found=%v passed=%v", found, r.Passed())
	}
	assertCounts(t, r, 20, 0, 0, 7, 8)
}

// TestConformance_CBOREqualsJSON is E.3's invariant at the report level:
// the CBOR bundle produces the same rows, messages included, as the JSON
// bundle it was generated alongside.
func TestConformance_CBOREqualsJSON(t *testing.T) {
	fromJSON := mustRun(t, prodBundle(t, testfixtures.ProdComplete), offlineWithKeyring())
	fromCBOR := mustRun(t, prodBundle(t, testfixtures.ProdCBOR), offlineWithKeyring())
	if fromCBOR.Format != "cbor" || fromJSON.Format != "json" {
		t.Errorf("formats: %s / %s", fromJSON.Format, fromCBOR.Format)
	}
	if len(fromJSON.Steps) != len(fromCBOR.Steps) {
		t.Fatalf("row counts differ: %d vs %d", len(fromJSON.Steps), len(fromCBOR.Steps))
	}
	for i := range fromJSON.Steps {
		if fromJSON.Steps[i] != fromCBOR.Steps[i] {
			t.Errorf("row %d differs:\n json: %+v\n cbor: %+v", i, fromJSON.Steps[i], fromCBOR.Steps[i])
		}
	}
	if !fromCBOR.Passed() {
		t.Error("CBOR bundle did not pass")
	}
}

// appendixD4 is Appendix D.4's table: every row a conforming verifier must
// reproduce, offline, with the subject's own file hash supplied.
var appendixD4 = []rowKey{
	{groupHashComparison, StatusPass},
	{groupSigningKey, StatusPass},
	{groupSubjectData, StatusPass}, {groupSubjectData, StatusPass}, {groupSubjectData, StatusPass},
	{groupInclusion, StatusPass},
	{groupBlockHash, StatusPass}, {groupBlockHash, StatusPass},
	{groupEpoch, StatusPass}, {groupEpoch, StatusPass},
	{groupProof, StatusPass},
	{groupKeyBinding, StatusSkip},
	{groupSigningKeyEvent, StatusPass}, {groupSigningKeyEvent, StatusPass}, {groupSigningKeyEvent, StatusInfo},
	{groupSigningKeyEvent, StatusPass}, {groupSigningKeyEvent, StatusPass}, {groupSigningKeyEvent, StatusSkip},
	{groupStructure, StatusPass},
	{groupWitnesses, StatusPass}, {groupWitnesses, StatusPass}, {groupWitnesses, StatusPass},
	{groupWitnesses, StatusPass}, {groupWitnesses, StatusPass},
	{groupSubmissionWindow, StatusPass},
	{groupTemporalInfo, StatusInfo},
	{groupSubmittedAfter, StatusInfo}, {groupSubmittedAfter, StatusInfo}, {groupSubmittedAfter, StatusInfo},
	{groupSubmittedAfter, StatusInfo}, {groupSubmittedAfter, StatusInfo},
	{groupSubmittedBefore, StatusInfo},
	{groupStellar, StatusSkip},
	{groupBitcoin, StatusSkip},
	{groupEntropySource, StatusSkip}, {groupEntropySource, StatusSkip}, {groupEntropySource, StatusSkip},
}

const appendixDClaimsHash = "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"

func TestConformance_AppendixD4(t *testing.T) {
	data := testfixtures.Read(t, testfixtures.WhitepaperDir, testfixtures.AppendixD)
	r := mustRun(t, data, Options{SkipExternal: true, ExpectedHash: appendixDClaimsHash})
	want := make([]Step, 0, len(appendixD4))
	for _, k := range appendixD4 {
		want = append(want, Step{Group: k.Group, Status: k.Status})
	}
	assertContainment(t, r.Steps, want)
	if !r.Passed() {
		t.Errorf("verdict FAILED\n%s", render(r.Steps))
	}
	// D.4: twenty-two passes, no failures, no warnings, seven skips, nine
	// informational rows (the ninth info is E.11's hash-shape row).
	assertCounts(t, r, 22, 0, 0, 7, 9)
	if !r.HashProvided() || !r.HashMatched() {
		t.Errorf("hash provided=%v matched=%v", r.HashProvided(), r.HashMatched())
	}
	// The same bundle without the expected hash warns and never fails.
	r = mustRun(t, data, Options{SkipExternal: true})
	for _, s := range r.Steps {
		if s.Group == groupHashComparison && (s.Status != StatusWarn || !strings.Contains(s.Message, "file hash not verified")) {
			t.Errorf("Hash Comparison without a hash = %s %q", s.Status, s.Message)
		}
	}
	if !r.Passed() {
		t.Error("verdict FAILED without an expected hash")
	}
	// A wrong hash fails the run, and only that row.
	r = mustRun(t, data, Options{SkipExternal: true, ExpectedHash: strings.Repeat("00", 32)})
	if r.Passed() || r.FailedCount() != 1 || r.HashMatched() {
		t.Errorf("wrong hash: passed=%v failed=%d matched=%v", r.Passed(), r.FailedCount(), r.HashMatched())
	}
}

// TestTamper runs the mutated bundles of the tamper table and pins which
// groups fail, the verdict, and containment against the reference and
// server reports where they exist.
func TestTamper(t *testing.T) {
	cases := []struct {
		file       string
		failGroups map[string]int
	}{
		{"tamper-claims.json", map[string]int{groupInclusion: 1, groupProof: 1}},
		{"tamper-witness-hash.json", map[string]int{groupInclusion: 1, groupProof: 1, groupWitnesses: 1}},
		{"tamper-witness-detail.json", map[string]int{groupWitnesses: 1}},
		{"tamper-root.json", map[string]int{groupInclusion: 1, groupEpoch: 1, groupProof: 1}},
		{"tamper-epoch-root.json", map[string]int{groupEpoch: 1, groupProof: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			data := testfixtures.Read(t, testfixtures.TamperDir, tc.file)
			r := mustRun(t, data, offlineWithKeyring())
			if r.Passed() {
				t.Fatalf("verdict PASSED, want FAILED\n%s", render(r.Steps))
			}
			got := map[string]int{}
			for _, s := range r.Steps {
				if s.Status == StatusFail {
					got[s.Group]++
				}
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.failGroups) {
				t.Errorf("failing groups = %v, want %v\n%s", got, tc.failGroups, render(r.Steps))
			}
			if ref, err := os.ReadFile(testfixtures.Path(testfixtures.TamperDir, strings.TrimSuffix(tc.file, ".json")+".standalone-report.txt")); err == nil {
				assertContainment(t, r.Steps, parseReferenceReport(t, string(ref)))
			}
			// The server's online report: every row it grades pass must
			// pass here too when the run is online; offline, only the
			// failing rows are comparable. Fail rows must agree exactly.
			server, passed := parseServerReport(t, testfixtures.Read(t, testfixtures.TamperDir, strings.TrimSuffix(tc.file, ".json")+".server-report.json"))
			if passed {
				t.Error("server report says passed")
			}
			serverFails := map[string]int{}
			for _, s := range server {
				if s.Status == StatusFail {
					serverFails[s.Group]++
				}
			}
			if fmt.Sprint(serverFails) != fmt.Sprint(got) {
				t.Errorf("failing groups differ from the server: server %v, cli %v", serverFails, got)
			}
		})
	}

	t.Run("drop-key-event.json", func(t *testing.T) {
		r := mustRun(t, testfixtures.Read(t, testfixtures.TamperDir, "drop-key-event.json"), offlineWithKeyring())
		if !r.Passed() {
			t.Errorf("verdict FAILED\n%s", render(r.Steps))
		}
		n := 0
		for _, s := range r.Steps {
			if s.Group == groupSigningKeyEvent {
				n++
				if s.Status != StatusSkip {
					t.Errorf("Signing Key Event row = %s %q", s.Status, s.Message)
				}
			}
		}
		if n != 1 {
			t.Errorf("Signing Key Event rows = %d, want 1", n)
		}
	})

	for file, code := range map[string]string{"old-layout.json": proof.CodeUnsupportedLayout, "tamper-type.json": proof.CodeUnexpectedSubjectFieldsForBlockLike} {
		t.Run(file, func(t *testing.T) {
			_, err := RunFromBytes(testfixtures.Read(t, testfixtures.TamperDir, file), file, offlineWithKeyring())
			if proof.RejectionCode(err) != code {
				t.Errorf("rejection = %v, want %s", err, code)
			}
		})
	}
}

// TestTypeAssertion pins the --type rejection.
func TestTypeAssertion(t *testing.T) {
	data := prodBundle(t, testfixtures.ProdComplete)
	if _, err := RunFromBytes(data, "x", Options{SkipExternal: true, ExpectedSubjectType: "block"}); proof.RejectionCode(err) != proof.CodeSubjectTypeMismatch {
		t.Errorf("mismatch rejection = %v", err)
	}
	if _, err := RunFromBytes(data, "x", Options{SkipExternal: true, ExpectedSubjectType: "item"}); err != nil {
		t.Errorf("matching assertion rejected: %v", err)
	}
}

// TestConformance_ProductionOnline compares an online run against the
// server's own reports. It needs the public Horizon, NIST and Blockstream
// endpoints and runs only when TRUESTAMP_ONLINE_TESTS is set.
func TestConformance_ProductionOnline(t *testing.T) {
	if os.Getenv("TRUESTAMP_ONLINE_TESTS") == "" {
		t.Skip("set TRUESTAMP_ONLINE_TESTS=1 to run against the public chains")
	}
	cases := []struct{ bundle, report string }{
		{testfixtures.ProdComplete, "verify-complete.json"},
		{testfixtures.ProdCompact, "verify-compact.json"},
		{testfixtures.ProdPartial, "verify-partial.json"},
	}
	for _, tc := range cases {
		t.Run(tc.bundle, func(t *testing.T) {
			opts := Options{KeyringFile: testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdKeyring)}
			r := mustRun(t, prodBundle(t, tc.bundle), opts)
			want, passed := parseServerReport(t, prodBundle(t, tc.report))
			if !passed || !r.Passed() {
				t.Errorf("passed: server %v, cli %v\n%s", passed, r.Passed(), render(r.Steps))
			}
			// Key Binding excepted: this run pins the keyring and passes;
			// the server does not pin its own and skips.
			assertContainment(t, r.Steps, want, groupKeyBinding)
			t.Logf("\n%s", render(orderedSteps(r)))
		})
	}
}

// TestConformance_StagingBitcoin runs the staging bundle that carries a
// real Bitcoin regtest commitment (raw transaction, txoutproof and block
// merkle root, listed ahead of its Stellar entry, so N=2 with Bitcoin
// first). Offline, the four internal-consistency rows are the server's, as
// info, the binding is a skip, and the signature covers both roots.
func TestConformance_StagingBitcoin(t *testing.T) {
	data := testfixtures.Read(t, testfixtures.StagingDir, testfixtures.ProdComplete)
	keyring := testfixtures.Path(testfixtures.StagingDir, testfixtures.ProdKeyring)
	r := mustRun(t, data, Options{SkipExternal: true, KeyringFile: keyring})
	if !r.Passed() {
		t.Fatalf("verdict FAILED\n%s", render(r.Steps))
	}
	var bitcoin []string
	for _, s := range r.Steps {
		if s.Group == groupBitcoin {
			bitcoin = append(bitcoin, s.Status.String()+" | "+s.Message)
		}
	}
	want := []string{
		"info | OP_RETURN in the supplied raw transaction matches the epoch root (internal consistency)",
		"info | Transaction id 3ebe32a3cd2fb711405f5d7789309375a877fdc0e3ba5d7b63e182082a278e95 recomputed from the supplied raw transaction (internal consistency)",
		"info | Supplied txoutproof places the transaction under the supplied block Merkle root (internal consistency)",
		"info | Commitment block_merkle_root matches the supplied txoutproof header (internal consistency)",
		"skip | Bitcoin commitment unconfirmed: external confirmation skipped (offline)",
	}
	if strings.Join(bitcoin, "\n") != strings.Join(want, "\n") {
		t.Errorf("Bitcoin Commitment rows:\ngot:\n%s\nwant:\n%s", strings.Join(bitcoin, "\n"), strings.Join(want, "\n"))
	}
	found := false
	for _, s := range r.Steps {
		if s.Group == groupProof && strings.Contains(s.Message, "145-byte payload, type code 20, N=2") {
			found = true
		}
	}
	if !found {
		t.Errorf("signature row does not report the 145-byte N=2 payload\n%s", render(r.Steps))
	}
	if r.Temporal.BitcoinCommit != "2026-09-03T18:15:06Z" || r.Temporal.StellarCommit != "2026-09-03T18:10:02Z" {
		t.Errorf("temporal = %+v", r.Temporal)
	}
	// The production keyring must NOT vouch for staging's key.
	r = mustRun(t, data, Options{SkipExternal: true, KeyringFile: testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdKeyring)})
	if r.Passed() {
		t.Error("staging bundle passed with the production keyring pinned")
	}
	for _, s := range r.Steps {
		if s.Group == groupKeyBinding && (s.Status != StatusFail || !strings.Contains(s.Message, "NOT in the pinned keyring")) {
			t.Errorf("Key Binding against the wrong keyring = %s %q", s.Status, s.Message)
		}
	}
	// Online (gated): every row of the server's report at the same status,
	// Key Binding excepted; the server's regtest skip and this verifier's
	// regtest skip are both skips.
	if os.Getenv("TRUESTAMP_ONLINE_TESTS") == "" {
		return
	}
	r = mustRun(t, data, Options{KeyringFile: keyring})
	serverSteps, passed := parseServerReport(t, testfixtures.Read(t, testfixtures.StagingDir, "verify-complete.json"))
	if !passed || !r.Passed() {
		t.Errorf("passed: server %v, cli %v\n%s", passed, r.Passed(), render(r.Steps))
	}
	assertContainment(t, r.Steps, serverSteps, groupKeyBinding)
}
