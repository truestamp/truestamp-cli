// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// End-to-end proofs for Appendix E.18 / E.19 / E.21's availability
// grading, run through a real `truestamp verify` process.
//
// WHY THIS EXISTS. The Horizon, Blockstream and NIST base URLs are package
// vars in internal/external, so a claim about what an unreachable Horizon
// or a 404 from Blockstream does to a report would otherwise rest on an
// in-package httptest that calls the lookup function directly. That leaves
// the whole chain above it unproven: cobra flag and config resolution, the
// shared HTTP client, external.Classify, the pipeline's grading switch, the
// exit code, and the --json projection.
//
// HOW THE ENDPOINTS ARE REDIRECTED, AND WHY IT IS NOT A SHIPPED FLAG. The
// redirection is done at LINK time: this test builds its own binary with
// `-ldflags -X internal/external.<Var>=<httptest URL>`. Nothing is added to
// the production source, so a released binary has no way to be pointed at
// an attacker-controlled Horizon or Blockstream. E.18/E.19/E.21 exist
// precisely to consult a third party, and a supported override would let a
// caller supply both the proof and the "independent" chain that confirms
// it. TestCLI_ExternalEndpoints_NoAmbientOverride pins that property.

// externalStub is the httptest server standing in for Horizon, Blockstream,
// NIST and the keyring. Roles are separated by path prefix, and behaviour
// is switched through mode.
type externalStub struct {
	mu   sync.Mutex
	mode string
	srv  *httptest.Server
}

func (s *externalStub) setMode(m string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = m
}

func (s *externalStub) currentMode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

// The Appendix D bundle's derived key id and public_key, served by the stub
// keyring so E.17's Key Binding row passes and the only external row in
// play is the one under test.
const (
	appendixDKeyID  = "f2c39df9"
	appendixDPubKey = "IVL40Zt5HSRFMkLhXy6rbLfP+ntqXtMAl5YOBpiB2xI="
)

// entropyBlockHash is the entropy.hash of the fabricated entropy_bitcoin
// bundle below: the block the E.21 lookup asks Blockstream about.
const entropyBlockHash = "0000000000000000000123456789abcdef0123456789abcdef0123456789abcd"

func (s *externalStub) handler(w http.ResponseWriter, r *http.Request) {
	mode := s.currentMode()
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.URL.Path == "/.well-known/keyring.json":
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": "1.0",
			"keys": []map[string]any{
				{"key_id": appendixDKeyID, "public_key": appendixDPubKey, "sequence": 0, "active": true},
			},
		})

	case strings.HasPrefix(r.URL.Path, "/horizon/transactions/"):
		switch mode {
		case "stellar_mismatch":
			// A well-formed transaction that IS the one asked for, but
			// whose memo commits to different bytes: E.18's one failing
			// outcome, the chain answered and disagreed.
			hash := strings.TrimPrefix(r.URL.Path, "/horizon/transactions/")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"hash":       hash,
				"memo_type":  "hash",
				"memo":       "ERERERERERERERERERERERERERERERERERERERERERE=",
				"ledger":     51234567,
				"created_at": "2026-07-24T12:00:12Z",
			})
		default:
			http.Error(w, `{"detail":"not configured"}`, http.StatusInternalServerError)
		}

	case strings.HasPrefix(r.URL.Path, "/blockstream/block/"):
		switch mode {
		case "btc_404":
			http.Error(w, `{"error":"Block not found"}`, http.StatusNotFound)
		case "btc_height_mismatch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          entropyBlockHash,
				"height":      999999, // the bundle says 870000
				"timestamp":   1750000000,
				"merkle_root": strings.Repeat("ab", 32),
			})
		default:
			http.Error(w, `{"error":"not configured"}`, http.StatusInternalServerError)
		}

	default:
		http.NotFound(w, r)
	}
}

// entropyBitcoinProofJSON is a fabricated entropy_bitcoin bundle. Its
// Merkle values and signature are invented, so the run as a whole fails;
// the assertions are on the Entropy Source ROW's status, which E.21 grades
// independently of every other step. The Bitcoin commitment names regtest
// deliberately: regtest has no public API, so the commitment row needs no
// network and the only outbound lookup in the run is the E.21 one.
var entropyBitcoinProofJSON = fmt.Sprintf(`{
  "version": 1,
  "type": "entropy_bitcoin",
  "public_key": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "signature": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "generated_at": "2026-04-06T23:25:06Z",
  "subject": {
    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
    "entropy": {"hash": "%s", "height": 870000, "time": 1750000000},
    "metadata": {},
    "signing_key_id": "4ceefa4a"
  },
  "inclusion_proof": "AA",
  "block": {
    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
    "previous_block_hash": "1111111111111111111111111111111111111111111111111111111111111111",
    "merkle_root": "2222222222222222222222222222222222222222222222222222222222222222",
    "metadata": {},
    "signing_key_id": "4ceefa4a"
  },
  "commitments": [{"chain": "bitcoin", "network": "regtest", "transaction_hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "epoch_merkle_root": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "block_height": 1, "epoch_proof": "AA"}]
}`, entropyBlockHash)

func TestCLI_ExternalEndpoints_EndToEnd(t *testing.T) {
	stub := &externalStub{}
	stub.srv = httptest.NewServer(http.HandlerFunc(stub.handler))
	defer stub.srv.Close()

	// A server started and immediately closed: its address refuses
	// connections, the closest stand-in for "Horizon is unreachable" that
	// does not depend on the host's real network.
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	bin := buildWithEndpoints(t, map[string]string{
		// The Appendix D bundle's Stellar commitment names "public", so it
		// resolves here, at the dead address.
		"HorizonPublicURL": deadURL,
		// The same bundle with `network` rewritten to "testnet" resolves
		// here instead, at the live stub.
		"HorizonTestnetURL":     stub.srv.URL + "/horizon",
		"BlockstreamMainnetURL": stub.srv.URL + "/blockstream",
		"BlockstreamTestnetURL": stub.srv.URL + "/blockstream",
		"NISTBeaconURL":         stub.srv.URL + "/nist",
	})

	// The keyring URL is derived from --base-url, so pointing the origin
	// at the stub serves /.well-known/keyring.json from it.
	baseURL := stub.srv.URL

	// E.18: an unreachable Horizon SKIPS, and does not fail a proof that
	// is otherwise sound. The entropy witnesses' sources are the dead
	// address too (NIST at the stub's unconfigured path, Horizon dead,
	// Blockstream unconfigured), so every Entropy Source row skips.
	t.Run("unreachable Horizon skips and exits 0", func(t *testing.T) {
		stub.setMode("unused")
		out, code := runVerifyJSON(t, bin, conformanceVectorPath(t), "--base-url", baseURL, "--expected-hash", appendixDClaimsHash)
		if code != 0 {
			t.Errorf("exit code: got %d, want 0: an unreachable Horizon must not fail a sound proof\n%s",
				code, formatCLISteps(out.Steps))
		}
		st := stepStatuses(out, "Stellar Commitment")
		if !st["skip"] || st["fail"] {
			t.Errorf("Stellar Commitment: want a skip and no fail, got %v\n%s", st, formatCLISteps(out.Steps))
		}
		if es := stepStatuses(out, "Entropy Source"); es["fail"] || es["pass"] {
			t.Errorf("Entropy Source with unreachable sources: got %v\n%s", es, formatCLISteps(out.Steps))
		}
		// E.17 ran for real against the stub keyring.
		if kb := stepStatuses(out, "Key Binding"); !kb["pass"] {
			t.Errorf("Key Binding: want a pass against the stub keyring, got %v\n%s", kb, formatCLISteps(out.Steps))
		}
		// With nothing confirmed, both edges stay informational.
		if sa := stepStatuses(out, "Submitted After"); sa["pass"] {
			t.Errorf("Submitted After must not pass without a confirmed witness: %v", sa)
		}
		if sb := stepStatuses(out, "Submitted Before"); sb["pass"] {
			t.Errorf("Submitted Before must not pass without a confirmed chain: %v", sb)
		}
	})

	// E.18: a chain that answers and DISAGREES still fails. Same bundle,
	// one field rewritten so the lookup lands on the live stub; `network`
	// is not part of the signed payload, so every cryptographic check
	// stays intact and the difference in outcome is the endpoint's answer.
	t.Run("Stellar memo mismatch fails and exits 1", func(t *testing.T) {
		stub.setMode("stellar_mismatch")
		path := rewriteBundle(t, conformanceVectorPath(t), func(m map[string]any) {
			m["commitments"].([]any)[0].(map[string]any)["network"] = "testnet"
		})
		out, code := runVerifyJSON(t, bin, path, "--base-url", baseURL)
		if code == 0 {
			t.Errorf("exit code 0: a chain that answered and disagreed must fail\n%s", formatCLISteps(out.Steps))
		}
		if st := stepStatuses(out, "Stellar Commitment"); !st["fail"] {
			t.Errorf("Stellar Commitment: want a fail, got %v\n%s", st, formatCLISteps(out.Steps))
		}
		if !strings.Contains(strings.ToLower(rawIssueText(out)), "memo") {
			t.Errorf("the failure should name the memo:\n%s", formatCLISteps(out.Steps))
		}
	})

	// E.21 on an entropy SUBJECT: a 404 from Blockstream SKIPS. E.21 fails
	// only on a value mismatch; a 404 yields no upstream value to compare.
	entropyPath := writeProofFile(t, entropyBitcoinProofJSON)

	t.Run("Blockstream 404 on an entropy source skips", func(t *testing.T) {
		stub.setMode("btc_404")
		out, _ := runVerifyJSON(t, bin, entropyPath, "--skip-signatures", "--base-url", baseURL)
		st := stepStatuses(out, "Entropy Source")
		if !st["skip"] || st["fail"] {
			t.Errorf("Entropy Source: want a skip and no fail, got %v\n%s", st, formatCLISteps(out.Steps))
		}
		if !strings.Contains(rawIssueText(out), "HTTP 404") {
			t.Errorf("the skip should name the 404:\n%s", formatCLISteps(out.Steps))
		}
	})

	t.Run("Blockstream height mismatch on an entropy source fails", func(t *testing.T) {
		stub.setMode("btc_height_mismatch")
		out, code := runVerifyJSON(t, bin, entropyPath, "--skip-signatures", "--base-url", baseURL)
		if code == 0 {
			t.Errorf("exit code 0 for a bundle whose entropy source disagrees\n%s", formatCLISteps(out.Steps))
		}
		if st := stepStatuses(out, "Entropy Source"); !st["fail"] {
			t.Errorf("Entropy Source: want a fail, got %v\n%s", st, formatCLISteps(out.Steps))
		}
		if !strings.Contains(strings.ToLower(rawIssueText(out)), "height mismatch") {
			t.Errorf("the failure should name the height mismatch:\n%s", formatCLISteps(out.Steps))
		}
	})
}

// TestCLI_ExternalEndpoints_NoAmbientOverride pins the property that makes
// the link-time redirection above acceptable: the SHIPPED binary offers no
// supported way to retarget an external lookup.
func TestCLI_ExternalEndpoints_NoAmbientOverride(t *testing.T) {
	help := helpText(t, "verify")
	for _, needle := range []string{"horizon", "blockstream", "beacon.nist", "nist-url", "stellar-url", "bitcoin-url"} {
		if strings.Contains(strings.ToLower(help), needle) {
			t.Errorf("`verify --help` advertises %q: external endpoints must not be caller-selectable", needle)
		}
	}
	for _, kv := range []string{
		"TRUESTAMP_HORIZON_URL=http://127.0.0.1:1",
		"TRUESTAMP_BLOCKSTREAM_URL=http://127.0.0.1:1",
		"TRUESTAMP_NIST_URL=http://127.0.0.1:1",
	} {
		cmd := exec.Command(binaryPath, "verify", conformanceVectorPath(t), "--offline", "--json")
		cmd.Env = append(cleanEnv(), kv)
		if _, err := cmd.Output(); err != nil {
			t.Errorf("%s changed the outcome of a verify run: %v", kv, err)
		}
	}
}

// bitcoinOfflineEvidence is a real regtest Bitcoin commitment (raw
// transaction, txoutproof, block merkle root, txid, OP_RETURN payload) as
// the server emitted it, carried on a fabricated item bundle. The
// signature and Merkle values around it are invented, so the epoch walk
// and signature fail; the six E.19 offline steps read only these fields.
const bitcoinOfflineEvidence = `{
  "chain": "bitcoin", "network": "regtest",
  "epoch_merkle_root": "d6d2ff0348fa867830430e7052d7603db87d582b072302b4a116105a490f9644",
  "epoch_proof": "AA",
  "transaction_hash": "a612b0d3f77c77471760c875a2d58b27b07db38a024ae9a5e5e28067d07e754f",
  "block_height": 12834,
  "timestamp": "2026-04-22T20:00:02Z",
  "block_merkle_root": "68daa6850344036b216a37d3d8ff6a3d075319f7fd07f39c7a24d0b616f2c296",
  "raw_transaction": "02000000000102f8623f43f7003a90e649db693d64c5032f318927c8f2c571c38a469c19d787880000000000fdfffffffae8beba326bdc8178d2b21f042b5856f6a7c5ebc3ed53285452fb1b945e1b520100000000fdffffff0258c30000000000002251204c25f8ac4a9dac64e4148e94cb5b3ec2611f8e5a508a9a4bbf74ee37101c3f8a0000000000000000226a20d6d2ff0348fa867830430e7052d7603db87d582b072302b4a116105a490f96440140cc05bafb377f446d335d9936e74d8f4e6c72e097ab1709703b876e6d0bd1b968d6f2808780441a70b7c841cfad5a4bb4b826cad84f292d3f2feebf9ea24163fe01408fee265e9f3c79dd50b0f7c84f9f04cd10e87e5119bb67d9ac460bf3fdea311b8df672cde47f8c61ab4ab732021c48cb770d1225b2c90c39bba5b1bde144be9100000000",
  "txoutproof": "000000203c1321eb58f04f38880a4e01260e4437ff8d035db29fe417bb23a768fbc5780196c2f216b6d0247a9cf307fdf71953073d6affd8d3376a216b03440385a6da68c228e969ffff7f2001000000020000000249136ad66b7812f76ebaa01e2b571febb62266c15c6c407d08cae15e1e6661c74f757ed06780e2e5a5e94a028ab37db0278bd5a275c8601747777cf7d3b012a60105"
}`

// bitcoinEvidenceBundlePath writes a fabricated item bundle carrying the
// real Bitcoin offline evidence.
func bitcoinEvidenceBundlePath(t *testing.T) string {
	t.Helper()
	return rewriteBundle(t, conformanceVectorPath(t), func(m map[string]any) {
		var entry map[string]any
		if err := json.Unmarshal([]byte(bitcoinOfflineEvidence), &entry); err != nil {
			t.Fatal(err)
		}
		m["commitments"] = []any{entry}
	})
}

// TestCLI_BitcoinOfflineEvidence pins E.19(b): the offline steps grade the
// bundle's own bytes as internal consistency, in the server's four rows
// and words (info on success, "Check failed: " on contradiction), never as
// a passing commitment, and the binding step skips for a network with no
// public API.
func TestCLI_BitcoinOfflineEvidence(t *testing.T) {
	out, _ := runVerifyJSON(t, binaryPath, bitcoinEvidenceBundlePath(t), "--offline")
	rows := bitcoinRows(out)
	st := stepStatuses(out, "Bitcoin Commitment")
	if st["pass"] || st["fail"] || !st["info"] || !st["skip"] {
		t.Errorf("Bitcoin Commitment statuses = %v, want info rows plus a skipped binding and no pass/fail\n%s", st, strings.Join(rows, "\n"))
	}
	wantInfo := []string{
		"info | OP_RETURN in the supplied raw transaction matches the epoch root (internal consistency)",
		"info | Transaction id a612b0d3f77c77471760c875a2d58b27b07db38a024ae9a5e5e28067d07e754f recomputed from the supplied raw transaction (internal consistency)",
		"info | Supplied txoutproof places the transaction under the supplied block Merkle root (internal consistency)",
		"info | Commitment block_merkle_root matches the supplied txoutproof header (internal consistency)",
	}
	if len(rows) < 5 || strings.Join(rows[:4], "\n") != strings.Join(wantInfo, "\n") {
		t.Errorf("Bitcoin consistency rows:\ngot:\n%s\nwant:\n%s", strings.Join(rows, "\n"), strings.Join(wantInfo, "\n"))
	}
	if !strings.Contains(strings.Join(rows, "\n"), "skip | Bitcoin commitment unconfirmed") {
		t.Errorf("binding not skipped:\n%s", strings.Join(rows, "\n"))
	}
	// A contradicted OP_RETURN fails.
	bad := rewriteBundle(t, bitcoinEvidenceBundlePath(t), func(m map[string]any) {
		m["commitments"].([]any)[0].(map[string]any)["epoch_merkle_root"] = strings.Repeat("00", 32)
	})
	out, _ = runVerifyJSON(t, binaryPath, bad, "--offline")
	if st := stepStatuses(out, "Bitcoin Commitment"); !st["fail"] || !strings.Contains(strings.Join(bitcoinRows(out), "\n"), "fail | Check failed: OP_RETURN in the supplied raw transaction") {
		t.Errorf("a contradicted OP_RETURN did not fail with the server's wording: %v\n%s", st, strings.Join(bitcoinRows(out), "\n"))
	}
}

// TestCLI_RawTxAndTxOutProof_CaseIsNotGraded pins the one hex-valued pair
// deliberately left OUT of E.4's lowercase-hex enforcement: raw_transaction
// and txoutproof carry either base64url or hex, so a lowercase rule there
// is undefined, and neither is trusted as a value (their derived txid,
// OP_RETURN and Merkle root are compared against fields the rule does
// cover). A case flip decodes to identical bytes and changes nothing.
func TestCLI_RawTxAndTxOutProof_CaseIsNotGraded(t *testing.T) {
	sample := bitcoinEvidenceBundlePath(t)
	baseline, baseCode := runVerifyJSON(t, binaryPath, sample, "--offline")
	wantRows := bitcoinRows(baseline)
	if len(wantRows) == 0 {
		t.Fatal("the bundle carries no Bitcoin Commitment rows, so this test asserts nothing")
	}
	for _, field := range []string{"raw_transaction", "txoutproof"} {
		t.Run(field, func(t *testing.T) {
			path := rewriteBundle(t, sample, func(m map[string]any) {
				c := m["commitments"].([]any)[0].(map[string]any)
				c[field] = strings.ToUpper(c[field].(string))
			})
			out, code := runVerifyJSON(t, binaryPath, path, "--offline")
			if code != baseCode {
				t.Errorf("uppercasing %s changed the exit code: %d -> %d", field, baseCode, code)
			}
			if got := bitcoinRows(out); strings.Join(got, "\n") != strings.Join(wantRows, "\n") {
				t.Errorf("uppercasing %s changed the Bitcoin Commitment grading:\ngot:\n%s\nwant:\n%s",
					field, strings.Join(got, "\n"), strings.Join(wantRows, "\n"))
			}
			if st := stepStatuses(out, "Structure"); st["fail"] {
				t.Errorf("uppercasing %s tripped the E.4 sweep", field)
			}
		})
	}
	// The covered fields DO trip the sweep.
	path := rewriteBundle(t, sample, func(m map[string]any) {
		c := m["commitments"].([]any)[0].(map[string]any)
		c["transaction_hash"] = strings.ToUpper(c["transaction_hash"].(string))
	})
	out, _ := runVerifyJSON(t, binaryPath, path, "--offline")
	if st := stepStatuses(out, "Structure"); !st["fail"] || !strings.Contains(rawIssueText(out), "commitments[0].transaction_hash") {
		t.Errorf("uppercase transaction_hash was not named by the sweep: %v\n%s", st, formatCLISteps(out.Steps))
	}
}

// bitcoinRows renders the Bitcoin Commitment rows as comparable strings.
func bitcoinRows(out conformanceOutput) []string {
	var rows []string
	for _, s := range out.Steps {
		if strings.HasPrefix(s.Group, "Bitcoin") {
			rows = append(rows, s.Status+" | "+s.Message)
		}
	}
	return rows
}

// --- helpers ---

// buildWithEndpoints compiles the CLI with internal/external's endpoint
// vars overridden through -ldflags -X, and returns the binary path.
func buildWithEndpoints(t *testing.T, endpoints map[string]string) string {
	t.Helper()
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatalf("locating module root: %v", err)
	}
	const pkg = "github.com/truestamp/truestamp-cli/internal/external"
	var ld []string
	for name, url := range endpoints {
		ld = append(ld, fmt.Sprintf("-X %s.%s=%s", pkg, name, url))
	}
	out := filepath.Join(t.TempDir(), "truestamp-endpoints")
	cmd := exec.Command("go", "build", "-ldflags", strings.Join(ld, " "), "-o", out, "./cmd/truestamp")
	cmd.Dir = modRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building endpoint-redirected binary: %v\n%s", err, b)
	}
	return out
}

// rewriteBundle decodes a bundle, applies mutate, and writes it to a new
// temp file, returning the path. Numbers are never rounded.
func rewriteBundle(t *testing.T, path string, mutate func(map[string]any)) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var m map[string]any
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("decoding %s: %v", path, err)
	}
	mutate(m)
	encoded, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("encoding bundle: %v", err)
	}
	out := filepath.Join(t.TempDir(), "rewritten.json")
	if err := os.WriteFile(out, encoded, 0o600); err != nil {
		t.Fatalf("writing %s: %v", out, err)
	}
	return out
}

// helpText returns a subcommand's --help output.
func helpText(t *testing.T, sub string) string {
	t.Helper()
	cmd := exec.Command(binaryPath, sub, "--help")
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s --help: %v\n%s", sub, err, out)
	}
	return string(out)
}

var _ = testfixtures.Root
