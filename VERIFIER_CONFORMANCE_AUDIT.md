# Verifier Conformance Audit: truestamp-cli vs Whitepaper Appendix E

Status when written: findings only, nothing changed.

**Status now: SUPERSEDED as a description of the code.** The findings below were subsequently
remediated, adversarially judged, and re-remediated in the same working copy this file lives in.
Read this document as the historical delta that motivated those changes, not as a description of
current behaviour — every code excerpt, line number and "what the CLI does" paragraph describes
the pre-remediation tree. The current behaviour is pinned by the tests, chiefly
`internal/verify/conformance_test.go` (Appendix D.4 containment, the tamper suite) and
`cmd/external_e2e_test.go` (E.18/E.19/E.21 availability grading through the shipped pipeline).

Audit date: 2026-07-24. Audited commit: whatever was checked out at that time; re-verify line
numbers before editing, they will drift.

---

## 1. What this is and why it exists

The Truestamp whitepaper's **Appendix E is the normative verification reference**. It defines,
step by step, what a conforming proof verifier must do: which conditions abort parsing outright,
which produce a failing step inside a report, which must be reported as skipped, and what the
resulting report must look like.

Appendix E was recently amended, and a 16-commit stack brought five surfaces into line with it:
the Elixir production verifier, the LiveView verify page, the knowledge base, the reference
Livebook, and a new dependency-free standalone reference verifier. **The Go CLI was not part of
that pass.** This document is the delta between the CLI and the amended appendix.

The amendments that matter most to this repo:

- **E.5's per-chain field table was regraded into three columns**: structurally required (E.6
  rejects), required for external confirmation (absence must be a `skip`), and optional.
  `cx[].tx`, `cx[].bmr`, `cx[].rtx` and `cx[].txp` all moved out of "required".
- **E.22 gained a `Key Binding` group** and a documented Subject Data category exception.
- **E.25's "match D.4" was clarified to one-way containment**: extra `skip` and `info` rows are
  conformant, extra `fail`/`pass`/`warn` rows are not.
- **The unknown-`t` rule was made consistent.**

### Authority files (absolute paths)

| File | Role |
|---|---|
| `/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/whitepaper.typ` | **Normative.** Appendix E starts around line 6100. Read it directly. |
| `/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/verify_proof.exs` | Dependency-free reference verifier written straight to Appendix E. **This is the behavioral oracle.** |
| `/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/vectors/bundle.json` | The Appendix D worked bundle. E.25 makes reproducing its report a conformance requirement. |
| `/Users/glenn/src/github.com/truestamp/truestamp-v2/lib/truestamp/proof/verification.ex` | The just-fixed Elixir production verifier. A working reference implementation for most findings here. |
| `/Users/glenn/src/github.com/truestamp/truestamp-v2/priv/cddl/proof.cddl` | The wire schema. Note it declares `txp`/`rtx`/`bmr` as nullable. |

**Read Appendix E directly. Do not trust this summary alone.** Every finding below cites a
`whitepaper.typ:LINE`, but line numbers drift and prose carries nuance this document compresses.

### Elixir fix commits worth reading

Where a finding mirrors a defect already fixed on the Elixir side, the commit is named in the
finding. The full stack, newest first:

```
ffe6cbaf16  Align the reference Livebook with whitepaper Appendix E
c6b32b562a  Report locally-checked Bitcoin commitments as unconfirmed on verify
5badf3a4cf  Fix OP_RETURN output selection and the entropy report group
0b32a0b3ee  Reject CBOR subject data that cannot round-trip to JSON
177ef14d16  Report the E.17 key binding check instead of omitting it
6ab5d23fb7  Scope the expected-hash comparison to item subjects
f125261688  Hard-reject cx entries with no epoch proof or epoch root
25c1a2cf28  Report unreachable external checks as skip, not fail
ed0f6f017b  Report, never raise, on legitimate and malformed proof bundle shapes
48351de3cb  Resolve the Appendix E verifier conformance contradictions
3430087a08  Add a standalone reference proof verifier
32ba10b989  Cross-check Bitcoin bmr against txoutproof header
a6cfb1ca6d  Report Bitcoin commitments as unconfirmed until externally checked
```

### Summary

| Severity | Count | IDs |
|---|---|---|
| Critical | 5 | F1 F2 F3 F4 F5 |
| High | 8 | F6 F7 F8 F9 F10 F11 F12 F13 |
| Medium | 8 | F14 F15 F16 F17 F18 F19 F20 F21 |
| Low | 5 | F22 F23 F24 F25 F26 |

"Critical" means a valid proof is rejected, or an invalid one accepted.

The headline result: **`truestamp verify` cannot verify the whitepaper's own published worked
example.** It rejects it at parse time (F1), and once that is patched around it reports the
bundle FAILED (F2). The reference verifier passes the same bytes.

---

## 2. How to reproduce

### Build

```bash
cd /Users/glenn/src/github.com/truestamp/truestamp-cli
go build -o /tmp/truestamp ./cmd/truestamp
```

Every command below is written as `truestamp ...`; substitute `/tmp/truestamp`.

### Run the oracle

Elixir is not on the default PATH via the mise shim. Export it first:

```bash
export PATH="/Users/glenn/.local/share/mise/installs/elixir/1.20.2-otp-29/bin:/Users/glenn/.local/share/mise/installs/erlang/29.0.2/bin:$PATH"
cd /Users/glenn/src/github.com/truestamp/truestamp-v2
elixir whitepaper/verify_proof.exs whitepaper/vectors/bundle.json
```

Optional flags: `--expected-hash HEX`, `--keyring FILE`.

### The oracle idea

**For any bundle, `verify_proof.exs` is the intended behavior.** It is dependency-free, written
directly from Appendix E, and passes the Appendix D vector. Where the CLI and it disagree on the
same bytes, that disagreement is a finding. Run both side by side:

```bash
B=/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/vectors/bundle.json
/tmp/truestamp verify "$B" --skip-external ; echo "cli exit=$?"
elixir /Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/verify_proof.exs "$B" ; echo "ref exit=$?"
```

Observed today:

```
parsing proof: cx[1].bmr is required
cli exit=1

  9 passed   0 failed   1 warned   3 skipped   4 info
  VERDICT: PASSED
ref exit=0
```

### Fixtures

Every reproduction in section 3 uses a file that lives in one of the two repos, so nothing
depends on a scratch directory that may have been cleaned up:

| Path | Shape |
|---|---|
| `truestamp-v2/whitepaper/vectors/bundle.json` | Appendix D worked bundle. `t = 20` item. Bitcoin `cx` entry with **no** `rtx`, `txp` or `bmr`. |
| `truestamp-cli/samples/truestamp-item-01KPVAP639RSVPZCW2CBS51CTV.json` | `t = 20` item with a full Bitcoin artifact set (`rtx`, `txp`, `bmr`, `tx`). |
| `truestamp-cli/samples/truestamp-beacon-019db753-...json` | **`t = 10`, a block, despite the `-beacon-` filename.** This is verbatim the artifact E.24 names as legitimate. |
| `truestamp-cli/samples/*.cbor` | Tagged CBOR (`d9 d9 f7` prefix) equivalents. |
| `truestamp-cli/internal/verify/testdata/fixtures/{item,block}.{json,cbor}` | Unit-test fixtures. |

A handful of findings need a bundle shape none of these carry (a large integer in `s.d`, a `cx`
entry with no `ep`). Those are reproduced by editing a fixture with `python3`, shown inline.

---

## 3. Findings, severity-ordered

### F1 (critical): `cx[].tx` and `cx[].bmr` are parse-time hard rejections, so the Appendix D bundle never produces a report

**What the CLI does.** `validateSizes` (`internal/proof/parse.go:155`) loops every `cx` entry and
calls `expectHexBytes(c.TransactionHash, 32, pfx+".tx")` at **`internal/proof/parse.go:191`**,
before the `switch c.Type`, so it applies to Stellar and Bitcoin alike. For Bitcoin entries it
also calls `expectHexBytes(c.BlockMerkleRoot, 32, pfx+".bmr")` at
**`internal/proof/parse.go:203`**. `expectHexBytes` opens with
`if s == "" { return fmt.Errorf("%s is required", field) }` at
**`internal/proof/parse.go:213-216`**, so absence is indistinguishable from a shape error.

The error unwinds out of `ParseBytes` and `internal/verify/verify.go:55-58` returns before
`runBundle` is entered. **No `Report` object is ever constructed**, in text or `--json` mode.
`validateSizes` is reached from all four parse entry points (`parse.go:119`, `parse.go:144`,
`binary.go:103`, `binary.go:129`), so both JSON and CBOR are affected. The fields are non-pointer
`string` with `omitempty` (`internal/proof/types.go:123-132`), so a JSON `null` hits the same
branch as an absent key.

**What Appendix E requires.** E.5's regraded three-column table
(`whitepaper.typ:6251-6252`) places, for Bitcoin (`cx[].t = 41`), only `t`, `op`, `ep` in the
structurally-required column; `tx` and `h` in required-for-external-confirmation; and
`net`, `txp`, `rtx`, `bmr`, `ts` in Optional. The presence note is explicit:

> Absence of a required-for-external-confirmation field is NOT a rejection and NOT a step failure:
> it means that chain's confirmation step (E.18 for Stellar, E.19 for Bitcoin) has nothing to look
> up and MUST be reported `skip`, exactly as a network failure is.
> (`whitepaper.typ:6263-6265`)

> the offline-verification payload `txp`, `rtx`, and `bmr` is optional ... marking any of these
> required would reject bundles the system legitimately emits.
> (`whitepaper.typ:6266-6269`)

E.6's hard-rejection table (`whitepaper.typ:6300-6310`) contains neither `tx` nor `bmr`. Its only
per-`cx`-entry rejections are a missing `ep` and a missing chain root key (`memo` for 40, `op` for
41). E.18 (`whitepaper.typ:6635-6637`) and E.19(c) (`whitepaper.typ:6683-6685`) both name an
absent `tx` as a `skip` condition.

The CLI is not deferring to the CDDL: `priv/cddl/proof.cddl:99-107` declares
`"bmr": bstr .size 32 / null` and comments that the offline payload keys "are always present but
carry null when the offline data was not stored". Feeding the CLI that CDDL-blessed null shape
still yields `cx[1].bmr is required`.

**Reproduction.**

```bash
B=/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/vectors/bundle.json
/tmp/truestamp verify "$B" --skip-external ; echo "exit=$?"
```

```
parsing proof: cx[1].bmr is required
exit=1
```

The Stellar side, on a bundle that does carry `bmr`:

```bash
python3 - <<'EOF'
import json
d = json.load(open('samples/truestamp-item-01KPVAP639RSVPZCW2CBS51CTV.json'))
for c in d['cx']:
    if c.get('t') == 40: del c['tx']
json.dump(d, open('/tmp/no_stellar_tx.json','w'))
EOF
/tmp/truestamp verify /tmp/no_stellar_tx.json --skip-external --type item ; echo "exit=$?"
```

```
parsing proof: cx[0].tx is required
exit=1
```

**User-visible consequence.** Any bundle whose Bitcoin epoch has not had its offline artifacts
captured, or whose Stellar or Bitcoin commitment has not yet been submitted, is unverifiable. The
user gets a bare parse error with no report, no per-step detail, and no indication that the
proof's cryptography is intact. The whitepaper's own published example is among them.

**Fix.** In `internal/proof/parse.go`, add

```go
func expectOptionalHexBytes(s string, nBytes int, field string) error {
    if s == "" { return nil }
    return expectHexBytes(s, nBytes, field)
}
```

and use it for `c.TransactionHash` (line 191) and `c.BlockMerkleRoot` (line 203). Keep the
mandatory `expectHexBytes` only for `c.MemoHash` (t=40, line 196) and `c.OpReturn` (t=41,
line 200), which E.6 genuinely does hard-reject. Empty string covers both an absent key and a
JSON `null` given the non-pointer field types. Absence then reaches E.18/E.19, which must report
`skip` (see **F2** and **F6**).

**Elixir reference.** `ed0f6f017b` "Report, never raise, on legitimate and malformed proof bundle
shapes".

---

### F2 (critical): E.19(c) absence branches are unimplemented, so an absent `rtx`/`txp` produces three fail rows and erases the Bitcoin commitment

**What the CLI does.** `verifySingleBitcoin` (`internal/verify/verify.go:856-925`) calls

- `bitcoin.ExtractOpReturn(cx.RawTxHex)` at **`internal/verify/verify.go:857`**
- `bitcoin.ComputeTxID(cx.RawTxHex)` at **`internal/verify/verify.go:866`**
- `bitcoin.DecodeTxOutProof(cx.TxoutproofHex)` at **`internal/verify/verify.go:875`**

unconditionally. There is no presence guard on `cx.RawTxHex` or `cx.TxoutproofHex` anywhere in the
function. With the field absent the argument is `""`, the wire parsers return `EOF`, and each
error arm calls `r.fail` (lines 859, 868, 877).

Worse, the `txp` arm ends in a bare `return` at **`internal/verify/verify.go:878`**, ahead of both
the `--skip-external` branch (line 907) and the `r.CommitmentInfos` append (line 923). An absent
`txp` therefore also **removes the Bitcoin commitment from the report entirely**.

`--skip-external` does not suppress any of this: the flag is first consulted at line 908, after
all three fail sites.

**What Appendix E requires.**

> Absence branches, per E.5: an absent `txp` skips steps 3, 4, and 5; an absent `rtx` skips steps
> 1, 2, and 6; an absent `bmr` skips step 4. A bundle missing all three carries no offline Bitcoin
> evidence, and its Bitcoin commitment is reported `skip`.
> (`whitepaper.typ:6681-6685`)

Appendix D.4 (`whitepaper.typ:6028`) pins the exact expected row for this bundle:
`Bitcoin Commitment / Blockchain / skip`, overall verdict `passed`. E.25's containment rule
(`whitepaper.typ:6881-6888`) states that no step D.4 reports may be absent, and that adding a
`fail` row for a step D.4 does not report is non-conformant.

**Reproduction.** Patch a `bmr` in so F1 does not mask the result:

```bash
python3 - <<'EOF'
import json
B='/Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/vectors/bundle.json'
d = json.load(open(B))
for c in d['cx']:
    if c.get('t') == 41: c['bmr'] = '00'*32
json.dump(d, open('/tmp/appD_plus_bmr.json','w'))
EOF
/tmp/truestamp verify /tmp/appD_plus_bmr.json --skip-external --type item ; echo "exit=$?"
```

```
x OP_RETURN extraction failed: parsing transaction: EOF
x Txid computation failed: parsing transaction: EOF
x Txoutproof parse failed: parsing merkle block: EOF
FAILED - proof verification failed
exit=1
```

`--json` on the same file shows `commitments` containing only `truestamp` and `stellar`: no
`bitcoin` object at all.

**User-visible consequence.** A valid, spec-conforming proof is reported as a forgery.

**Fix.** In `verifySingleBitcoin`, guard each sub-chain on presence:

- `if cx.RawTxHex == ""`: emit one `r.skip(groupBitcoin, CatBlockchain, ...)` covering E.19(b)
  steps 1, 2 and 6, and do not call `ExtractOpReturn` or `ComputeTxID`.
- `if cx.TxoutproofHex == ""`: emit one `r.skip` covering steps 3, 4 and 5, skip the merkle block
  entirely, and **fall through** to the `--skip-external` branch and the `CommitmentInfo` append
  rather than returning at line 878. Note `mb` is nil on that path, so lines 884
  (`mb.Hashes`/`mb.Flags`/`mb.Header.MerkleRoot`) and 904 (`mb.Header.BlockHash()`) must not run;
  `CommitmentInfo.BlockHash` needs a nil-safe source.
- If `rtx`, `txp` and `bmr` are all absent, emit the single Bitcoin Commitment `skip` D.4 names.

**Elixir reference.** The production verifier's Bitcoin path in
`lib/truestamp/proof/verification.ex` already branches this way. See `a6cfb1ca6d` and `c6b32b562a`.

---

### F3 (critical): JCS silently rounds integers outside the exactly representable double range, reporting a sound proof as tampered

**What the CLI does.** All canonicalization goes through `github.com/gowebpki/jcs` v1.0.1
(`go.mod:19`), which parses every JSON number with `strconv.ParseFloat` and errors only on
NaN/Infinity, never on loss of precision. Unguarded call sites:

- **`internal/verify/verify.go:314`** (`deriveClaimsHash`, the `0x11` item-claims digest)
- **`internal/verify/verify.go:935`** (`deriveEntropyHash`, the `0x21` entropy digest)
- **`cmd/codec.go:287`** (`truestamp jcs`)
- **`cmd/hash.go:298`** (`truestamp hash --jcs`)

A repo-wide grep for `UseNumber`, `json.Number`, `9007199254740991`, `1<<53`, `MaxSafe`,
`representab` and `portab` finds no guard anywhere in the verification path. On `err == nil` the
CLI emits `r.pass(... "Claims hash derived (0x11)")` over rounded digits, and the corruption first
surfaces two steps later as an inclusion-walk failure and a signature failure whose detail reads
"The proof may have been tampered with or signed with a different key."

**What Appendix E requires.**

> a number in the input that is not exactly representable as a double makes the bundle unverifiable
> and MUST be reported as such rather than silently rounded (entropy data from Stellar and Bitcoin
> routinely carries large integers: ledger sequences, nonces, difficulty)
> (`whitepaper.typ:6188-6190`)

E.22 (`whitepaper.typ:6776`) adds that a `warn` must not fail a proof. The reference verifier emits
`[WARN] Subject Data  not portably verifiable: s.d carries 1 integer(s) outside the exactly
representable range` and still returns `VERDICT: PASSED`.

**The producer side signs the exact literal.** `deps/jcs/lib/jcs.ex:219-220` and
`whitepaper/verify_proof.exs:121` both encode integers with `Integer.to_string/1` at arbitrary
precision, and truestamp-v2 has no safe-integer guard outside the reference verifier. So Truestamp
byte-exactly signs any user-submitted Item claim carrying a 64-bit integer: a snowflake id, a
nanosecond epoch timestamp, a 64-bit database key. **Every such proof is permanently unverifiable
by this CLI and is reported to its holder as possibly tampered.**

**Reproduction (mechanism, no fixture needed).**

```bash
printf '{"n":9007199254740993,"ledgerSequence":18446744073709551615}' > /tmp/big.json
/tmp/truestamp jcs /tmp/big.json ; echo " exit=$?"
```

```
{"ledgerSequence":18446744073709552000,"n":9007199254740992} exit=0
```

**Reproduction (end to end).** Build an entropy or item bundle whose `s.d` carries
`9007199254740993`, correctly signed. The CLI reports `FAILED` with inclusion-proof and signature
fail rows; the reference verifier reports `VERDICT: PASSED` with one warn. Swapping the value to
`9007199254740992` makes the CLI pass, isolating the cause to the single literal.

**Fix.** Before each `jcs.Transform`, scan the raw bytes with `json.Decoder` + `UseNumber()`. For
every `json.Number` with no fraction or exponent, confirm it round-trips through `float64`
without loss (compare the literal against `strconv.FormatFloat(f, 'f', -1, 64)`, or check
`strconv.ParseInt` stays within 2^53). On any lossy value:

- emit a `warn` step naming the offending key ("not portably verifiable"), and
- report the dependent Inclusion Proof and Proof Signature steps as **`skip`** with
  "cannot verify: s.d is not portably canonicalizable".

Never `fail`, and never proceed silently on the rounded value. Apply at all four sites.

**Scope the guard to integers only.** Section 7 records a 28-case cross-implementation
measurement showing floats are fully portable between this CLI and Elixir, including every
notation-switch boundary. It also gives the exact threshold to use (`|n| > 2^53`, not
`2^53 - 1`, and why the producer and verifier thresholds differ). Read it before implementing.

See also **F20** for the `convert proof` variant of the same root cause.

---

### F4 (critical): a Bitcoin commitment is reported "externally verified" when the binding lookup never ran

**What the CLI does.** `CommitmentInfo.Skipped` is populated from `opts.SkipExternal` alone
(**`internal/verify/verify.go:905`** for Bitcoin, **`internal/verify/verify.go:821`** for Stellar),
never from the outcome of the binding step. `external.VerifyBitcoinBlock` returns
`skipped = true`, before any HTTP call, for any `cx[].net` outside `{"mainnet","testnet"}`
(`internal/external/bitcoin.go:41-48`), covering regtest, an unknown value, and an absent `net`.

On that path the *step row* is a correct `skip` (verify.go:913) but `ci.Skipped` stays false, so
`internal/verify/json_output.go:145` emits `"externally_verified": true` and
`internal/verify/presenter.go:388-389` prints the unqualified Commitments row
`Block <h> on <net> (<ts>)` instead of the `(external verification skipped)` variant that exists at
line 386. The same field also reads `true` when the lookup ran and **failed** (verify.go:915,
and verify.go:833 for Stellar).

The `--json` surface has no compensating signal, because it drops every `skip` row (see **F12**),
so `externally_verified` is the only external-confirmation signal a machine consumer has.

**What Appendix E requires.** E.1's proof-of-work qualification
(`whitepaper.typ:6272-6278`):

> Where a chain's commitment is established from bundle-carried bytes alone (Bitcoin, E.19), the
> verifier MUST obtain one external confirmation point (a networked lookup, or an
> operator-supplied pinned header set or checkpoint) before reporting that commitment as passing,
> and MUST otherwise report it as skipped.

E.19(b) (`whitepaper.typ:6673-6676`): "When neither is available, the Bitcoin commitment MUST be
reported `skip`, never `pass`." The whitepaper is explicit that a fabricated low-difficulty header
over a one-transaction tree with any 32-byte `OP_RETURN` passes all six offline steps.

**Reproduction.** Take any `t = 20` bundle with a self-consistent Bitcoin artifact set, set
`cx[].net` to `"regtest"` (or delete it), and run with `--skip-signatures`:

```bash
/tmp/truestamp verify <bundle> --skip-signatures --json | python3 -c \
  "import json,sys; print(json.load(sys.stdin)['commitments']['bitcoin'])"
```

`"externally_verified": true`, exit 0, with zero network calls made. The reference verifier on the
same bytes reports `[SKIP] Bitcoin Commitment  not checked offline`.

**User-visible consequence.** A machine consumer archiving the `--json` document records a Bitcoin
commitment as externally confirmed when nothing external was consulted, on a header the bundle
itself supplied. This is exactly the misreading E.1 exists to prevent. Note the presented
timestamp on that path is the bundle-supplied `cx[].ts`, rendered into a field documented at
`internal/verify/report.go:154` as "Public blockchain timestamp".

**Fix.** Set the commitment's external status from the binding step's own outcome, not from the
flag. Replace `CommitmentInfo.Skipped` with a tri-state (confirmed / skipped / failed) assigned
inside each branch of `internal/verify/verify.go:908-922` (Bitcoin) and
`internal/verify/verify.go:825-840` (Stellar), and emit `"externally_verified": true` at
`json_output.go:145` only on the confirmed branch.

Do **not** additionally downgrade the offline rows at verify.go:861/870/894 to `info`. They are
honestly worded internal-consistency checks reported alongside the external skip, and E.25's
prohibition on extra `pass` rows is scoped to a run against the Appendix D bundle, which carries
no `rtx`/`txp` and so never produces them.

**Elixir reference.** `a6cfb1ca6d` "Report Bitcoin commitments as unconfirmed until externally
checked" and `c6b32b562a` "Report locally-checked Bitcoin commitments as unconfirmed on verify".

---

### F5 (critical): `--type` is inferred from the filename, so a byte-identical bundle passes or fails depending on what the file is called

**What the CLI does.** When the user passes no `--type`, `cmd/verify.go:127-133` runs
`inferTypeFromFilename(displayName)` (defined at **`cmd/verify.go:256-283`**, matching the
`truestamp-<stem>-<id>.<ext>` download convention over the stems in `filenameStems`,
`cmd/verify.go:239`) and assigns the result to `typeFlag`. That flows into
`Options.ExpectedSubjectType` (cmd/verify.go:164, 172-173) with **no marker distinguishing it from
a user-supplied flag**, on both the local and the `--remote` path.
`internal/verify/verify.go:82-84` compares it against `ptype.Name(bundle.T)` and
**`internal/verify/verify.go:117-121`** emits
`r.fail(groupSubjectType, CatStructural, "Proof is %s (t=%d) but --type %s was requested")`.

`Report.Passed()` (`internal/verify/report.go:206`) returns false on any fail, and
`cmd/verify.go:224` turns that into a non-zero exit. The explanatory stderr hint
(`cmd/verify.go:218-222`) is suppressed under both `--silent` and `--json`, so automation sees a
hard failure attributed to a flag the user never passed.

There is no opt-out: `--type ""` trims to empty and the inference still fires.

**What Appendix E requires.**

> A verifier MUST read `t` from the bundle, never from the downloaded filename. A file named
> `truestamp-beacon-<id>.json` may carry `t = 10` from the beacon show page while
> `/verify/beacon/:id` emits a genuine `t = 11` bundle, and only the signed `t` is authoritative.
> (`whitepaper.typ:6841-6843`)

Also `whitepaper.typ:4408`: "A verifier should not infer a subject type from a file name."
Additionally, D.4 (`whitepaper.typ:6008-6035`) has no Subject Type row, and E.25
(`whitepaper.typ:6887-6888`) states that adding a `fail` row for a step D.4 does not report is not
conformant.

**Reproduction.** **This repo ships the failing artifact.** `samples/truestamp-beacon-019db753-...json`
carries `t = 10`, which is verbatim the case E.24 names as legitimate:

```bash
cd /Users/glenn/src/github.com/truestamp/truestamp-cli
/tmp/truestamp verify samples/truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503.json --skip-external
echo "exit=$?"
```

```
  Structural
    x Proof is block (t=10) but --type beacon was requested

  FAILED - proof verification failed
  13 of 17 cryptographic checks passed  1 failed  3 skipped
  (inferred --type beacon from filename "truestamp-beacon-019db753-...json"; pass --type explicitly to override)
exit=1
```

The identical bytes with `--type block` exit 0. The reference verifier on the same bytes returns
`VERDICT: PASSED`, exit 0.

**User-visible consequence.** Renaming a file, or downloading it through a browser that appends
` (1)`, flips a cryptographically sound proof from verified to failed. Under `--json` the failure
is reported with no disclosure that the "requested" type came from the filename.

**Fix.** Delete the inference at `cmd/verify.go:127-133` so `typeFlag` stays empty unless the user
passes `--type`, and keep `r.fail` at `internal/verify/verify.go:117-121` only for an explicit
flag. If the filename hint is worth keeping, record it with **`r.info(...)`**
(`internal/verify/report.go:320`), never `r.warn(...)`: E.25 bars an additive `warn` row on the
same grounds as `fail` and `pass`, and permits only additive `skip` and `info`.

---

### F6 (high): Stellar confirmation reports transport failures, and an absent `net`, as `fail`

**What the CLI does.** `verifySingleStellar` has two defects.

1. **`internal/verify/verify.go:826-829`** short-circuits on
   `cx.Network != "testnet" && cx.Network != "public"` straight into
   `r.fail(groupStellar, CatBlockchain, "Stellar net %q is not a recognised Stellar network ...")`,
   before any lookup is attempted. The empty string satisfies that condition, and `net` is never
   validated at parse, so an absent `net` reaches it. `external.HorizonTestnetURL` already exists
   at `internal/external/stellar.go:34`.
2. **`internal/verify/verify.go:831-834`** maps *every* error from `external.VerifyStellar` to
   `r.fail`. `internal/external/stellar.go:57-83` returns one untyped error for transport failure,
   any non-2xx status, unparseable body, `memo_type` mismatch, memo mismatch and ledger mismatch
   alike, so the call site cannot discriminate availability from substance.

**What Appendix E requires.**

> A mismatch fails the step. A network failure, a `skip_external` request, or an entry that carries
> no `tx` to look up MUST be reported as skipped, never failed (E.5).
> (`whitepaper.typ:6635-6637`)

> `net` is optional on both chains, and an absent `net` on its own MUST NOT skip either
> confirmation step. E.18 resolves the Horizon endpoint through the default it already states, the
> non-`public` instance, so a Stellar entry with no `net` is still looked up and its outcome is
> graded as any other lookup would be.
> (`whitepaper.typ:6271-6274`)

Note the absent-`net` case is doubly non-conformant: the spec forbids even a `skip` there, and the
CLI does worse than skip.

**Reproduction.** Transport half:

```bash
/tmp/truestamp verify samples/truestamp-item-01KPVAP639RSVPZCW2CBS51CTV.json \
  --type item --http-timeout 1ms
```

```
x Stellar external verification failed: fetching Stellar transaction: Get
  "https://horizon.stellar.org/transactions/5e5e...": net/http: request canceled
FAILED   exit=1
```

Absent-`net` half: delete `net` from the `t=40` entry and run online. Output:
`x Stellar net "" is not a recognised Stellar network (expected testnet or public)`, exit 1, with
no HTTP request issued.

**User-visible consequence.** A cryptographically sound proof is reported FAILED with exit 1
because Horizon was unreachable, rate-limited, or because the user was behind a captive portal.
The verdict becomes a function of a third party's availability.

**Fix.** Give `external.VerifyStellar` typed or sentinel-wrapped errors so transport failures and
5xx/429 are distinguishable from substantive mismatch (the wrapping already uses `%w`, so
`errors.As` against `*net.OpError` / `*net.DNSError` is viable). At `internal/verify/verify.go:832`
route those to `r.skip(groupStellar, CatBlockchain, "Stellar commitment unconfirmed: <reason>")`
and reserve `r.fail` for `memo_type != "hash"`, memo mismatch and ledger mismatch. Keep a
definitive 404 as a `fail`: that means the transaction is absent from the chain, a substantive
confirmation failure. At `internal/verify/verify.go:827` default an empty `net` to
`HorizonTestnetURL` and grade the lookup normally; `internal/external/stellar.go:45-53` has its own
`default:` arm that errors on an unrecognised network and must be relaxed the same way.

**Elixir reference.** `25c1a2cf28` "Report unreachable external checks as skip, not fail". See
`lib/truestamp/proof/verification.ex:752-806`, which routes every transport outcome to
`Report.skip` while reserving `fail` for a 404 and for genuine mismatches, and
`Truestamp.Stellar.Horizon.resolve_network/1` (`horizon.ex:240`), whose catch-all defaults to
testnet.

---

### F7 (high): entropy source unavailability is reported `fail` on all three sources

**What the CLI does.** All three entropy source verifiers treat a failed re-fetch as a
verification failure:

- **`internal/verify/verify.go:680-683`** `NIST Beacon fetch failed (chain %d, pulse %d): %s`
- **`internal/verify/verify.go:718-721`** `Stellar ledger fetch failed (sequence %d on %s): %s`
- **`internal/verify/verify.go:761-764`** `Bitcoin block fetch failed (hash %s on %s): %s`

each `r.fail(groupEntropySource, CatBlockchain, ...)`. The correct pattern already exists two lines
above the last one: `internal/verify/verify.go:756-759` emits `r.skip` for the no-public-API case,
and `internal/verify/verify.go:609-613` correctly skips the whole step under `--skip-external`.
Only the transport branch is wrong.

**What Appendix E requires.**

> A mismatch fails; unavailability or a `skip_external` request MUST be reported as skipped.
> (`whitepaper.typ:6724`)

E.22 (`whitepaper.typ:6778-6782`) adds "a skipped external check MUST NOT fail a proof" and names
E.21 among the steps skippable for lack of network access.

**Reproduction.**

```bash
HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 \
  /tmp/truestamp verify <entropy t=30 bundle> ; echo "exit=$?"
```

```
x NIST Beacon fetch failed (chain 1, pulse 2847561): fetching NIST pulse:
  Get "https://beacon.nist.gov/...": proxyconnect tcp: connect: connection refused
FAILED - proof verification failed
exit=1
```

The same bundle with `--skip-external` reports that step as a `skip` and exits 0.

**User-visible consequence.** During the documented US budget lapse, or any NIST/Blockstream/Horizon
outage, every entropy proof is reported as failed.

**Fix.** Do not blanket-skip the whole `err` branch. `external.GetNISTPulse`,
`external.GetStellarLedger` (`internal/external/stellar.go:111-138`) and
`external.GetBitcoinBlockHeader` (`internal/external/bitcoin.go:94-124`) currently conflate genuine
unavailability with unknown network, invalid sequence or hash format, a malformed JSON body, and a
response missing the expected field. `internal/httpclient/client.go:89-112` collapses transport
errors and every non-2xx into one opaque error, so the typed-error work belongs there. Grade only
transport failure, timeout, 5xx and 429 as `r.skip(groupEntropySource, CatBlockchain, "Entropy
source unconfirmed: <reason>")`. Malformed-response and invalid-input errors keep failing, and the
comparison branches that already hold both values
(`internal/verify/verify.go:685-696`, `:723-734`, `:766-783`) keep failing on mismatch.

**Elixir reference.** `25c1a2cf28`; also `5badf3a4cf` for the entropy report group.

---

### F8 (high): keyring transport failure fails the proof, there is no `Key Binding` group, and the failure carries a positive assertion as its detail

Three related defects in `verifySigningKey`.

**8a. Transport failure is a `fail`.** **`internal/verify/verify.go:277-278`** maps every non-nil
error from `external.VerifyKeyring` to `r.fail`, discarding the classification
`internal/external/keyring.go` already performs: `keyring.go:126` wraps transport errors in
`*keyringNetError` after `classifyNetworkError` (`keyring.go:45-72`) has separated DNS,
connection-refused, timeout and TLS, while the substantive cases are plain errors at
`keyring.go:142` ("key %s not found in keyring") and `keyring.go:145` ("public key mismatch").

**8b. No `Key Binding` group.** All four keyring outcomes are filed under `groupSigningKey`
(`internal/verify/verify.go:236`), at lines 272, 274, 278 and 282, plus two `info` rows at 279-280.
A repo-wide grep for `"Key Binding"` returns nothing.

**8c. The failure detail is a positive assertion.** `internal/verify/json_output.go:181` sets
`Detail: lookupFailureDetail(s.Message)` on every fail/warn issue, and
`internal/verify/presenter.go:583` maps the keyword `keyring` to
`"The keyring confirms the signing key is a trusted Truestamp key."` on a run where no binding was
established.

**What Appendix E requires.**

> MUST report the check as skipped (not passed) when it is not performed. The result is reported as
> a step under the *Key Binding* group (E.22), so that a report can never be read as having
> established a binding it never attempted.
> (`whitepaper.typ:6607-6611`)

E.22's group table (`whitepaper.typ:6752`) lists `Key Binding (E.17)` separately from
`Signing Key (E.9)`. E.25 item 11 (`whitepaper.typ:6868-6869`): "Cross-check `pk` against a pinned
keyring, **or report the check as skipped**, under the Key Binding group". D.4
(`whitepaper.typ:6026`) carries the row `Key Binding / Cryptographic / skip`, and E.25's
containment rule forbids omitting a step D.4 reports.

**Reproduction.**

```bash
TRUESTAMP_BASE_URL=http://127.0.0.1:1 /tmp/truestamp verify \
  samples/truestamp-item-01KPVAP639RSVPZCW2CBS51CTV.json --type item --json
echo "exit=$?"
```

```json
{"result": "failed",
 "summary": {"failed": 1},
 "issues": [{"severity": "error",
             "category": "cryptographic",
             "message": "Keyring verification failed: could not connect to the keyring server; the server may be offline",
             "detail": "The keyring confirms the signing key is a trusted Truestamp key."}]}
exit=1
```

The same bundle with `--skip-external` exits 0.

**Fix.**

1. Export a predicate over the unexported `*keyringNetError` (e.g.
   `func external.IsKeyringUnreachable(err error) bool`; package `verify` cannot `errors.As`
   against an unexported type). Branch on it at `internal/verify/verify.go:277` and emit `r.skip`
   in D.4's shape ("Keyring cross-check not performed: <reason>"), keeping `r.fail` only for a
   keyring that answered and disagreed.
2. Add `groupKeyBinding = "Key Binding"` to the const block at `internal/verify/verify.go:234-257`
   and move **six** call sites onto it: 272, 274, 278, 279, 280, 282. Leave the
   `Public key valid, key_id: %s` pass at line 269 under `groupSigningKey`, which is where E.22
   puts E.9. Keep `CatCryptographic` on all of them.
3. Reword or drop the `"keyring"` entry in `failureDetails` (`internal/verify/presenter.go:583`) so
   it describes the failure rather than the success it would have established.

**Existing test to invert.** `internal/verify/verify_test.go:312-329`
(`TestVerifySigningKey_ExternalFails`) points `KeyringURL` at `http://127.0.0.1:1/nonexistent` and
asserts "unreachable keyring should produce a failure". That test pins the wrong behavior.

**Elixir reference.** `177ef14d16` "Report the E.17 key binding check instead of omitting it"
(see `@group_key_binding "Key Binding"` at `lib/truestamp/proof/verification.ex:1231`) and
`25c1a2cf28` for the skip-not-fail half.

---

### F9 (high): `--hash` on a non-item subject is a `fail`, and the run reports "verified" while exiting 1

**What the CLI does.** **`internal/verify/verify.go:124-127`**:

```go
if opts.ExpectedHash != "" && !isItem {
    r.fail(groupHashComparison, CatDataIntegrity,
        fmt.Sprintf("--hash flag is not applicable to %s proofs", subjectType))
}
```

The whole rest of E.7 lives inside `if isItem` (verify.go:128-169), and `r.HashProvided` is
assigned only at **`internal/verify/verify.go:135`**, inside that branch. Consequences:

- `Report.ProofPassed()` (`internal/verify/report.go:284-291`) excludes the `"Hash Comparison"`
  group, so `computeResult` (`internal/verify/json_output.go:205-220`) returns `"verified"`.
- `Report.Passed()` (`internal/verify/report.go:206`) does **not** exclude it, so
  `cmd/verify.go:224` exits 1.
- The `hash_comparison` JSON object is gated on `r.HashProvided != ""`
  (`json_output.go:111`) and is therefore absent.

For `t != 20` with **no** `--hash`, no Hash Comparison row of any status is emitted either.

**What Appendix E requires.**

> the comparison MUST run only when `t = 20`. For every other `t`, including the entropy codes
> {30, 31, 32} as well as the block-like codes {10, 11}, the step is not applicable, and a verifier
> MUST NOT fail it ... When a caller supplies an expected hash for a non-item subject, a verifier
> MUST report a visible `skip` for this step rather than silently discarding the argument.
> (`whitepaper.typ:6329-6335`)

E.22 (`whitepaper.typ:6783-6786`) makes the E.7 inapplicability skip REQUIRED and "reported, not
omitted". E.22 (`whitepaper.typ:6788-6789`) adds that "provided" must be reportable separately from
"matched".

**Reproduction.**

```bash
/tmp/truestamp verify samples/truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503.json \
  --skip-external --type block \
  --hash b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380 --json
```

```
result = verified
summary = {'passed': 13, 'failed': 1, 'warnings': 0, 'skipped': 3, 'total': 17}
issues = [{'severity': 'error', 'category': 'data_integrity',
           'message': '--hash flag is not applicable to block proofs'}]
```

and:

```bash
/tmp/truestamp verify ...same... --silent ; echo "silent exit=$?"
```

```
silent exit=1
```

The human presenter is equally split: it prints `VERIFIED - proof is valid` and `1 failed` in the
same summary block, then exits 1. The reference verifier prints
`[SKIP] Hash Comparison  --expected-hash ignored: only an item subject (t=20) commits to a file
hash` and exits 0.

**User-visible consequence.** A CI job that pipes a valid entropy or beacon proof plus a habitual
`--hash` argument fails the build. A consumer keying on `result` and a consumer keying on the exit
code reach opposite conclusions about the same run.

**Fix.** Replace the `r.fail` at `internal/verify/verify.go:125` with
`r.skip(groupHashComparison, CatDataIntegrity, ...)` naming the inapplicability. That single change
removes the `StatusFail`, restores exit 0, and keeps `"result": "verified"`.

Two clauses to **not** apply:

- Do **not** also emit a skip when `!isItem` and no `--hash` was supplied. The reference verifier
  emits no row in that case (`whitepaper/verify_proof.exs:713-725`), and E.7's MUST-skip is
  conditioned on the caller having supplied a hash.
- Do **not** set `r.HashProvided` outside the `isItem` branch on its own. `computeResult` would
  then see `hashProvided && !hashOK` (a `skip` is not a `StatusPass`, so `HashMatched()` stays
  false) and return `"hash_mismatch"` for a sound non-item proof. If you want the E.22
  provided-vs-matched exposure, change `computeResult`/`HashMatched` in the same commit.

Separately worth doing: reconcile `Passed()` and `ProofPassed()` so the exit code and the printed
verdict can never disagree.

**Elixir reference.** `6ab5d23fb7` "Scope the expected-hash comparison to item subjects".

---

### F10 (high): `cx[].bmr` is required to parse and then never compared, so E.19(b) step 4 is unimplemented

**What the CLI does.** `internal/verify/verify.go:882-884` calls
`bitcoin.VerifyPartialMerkleTree(mb.Hashes, mb.Flags, mb.Transactions, &mb.Header.MerkleRoot)`.
`mb` is decoded from `cx.TxoutproofHex` alone (`internal/bitcoin/parse.go:16-26`), so the
comparison at `internal/bitcoin/merkle.go:56` checks the txoutproof's tree against the
**txoutproof's own header**. `cx.BlockMerkleRoot` is never read in `internal/verify`: a repo-wide
grep returns only `internal/proof/types.go:132` (struct tag), `internal/proof/binary.go:220` (CBOR
decode), `internal/proof/marshal_cbor.go:204-205` (encode) and `internal/proof/parse.go:203` (the
F1 rejection).

**What Appendix E requires.** E.19(b) step 4 (`whitepaper.typ:6664-6667`): "verify the partial
Merkle tree to a root, which MUST equal `bmr`". E.19(c) (`whitepaper.typ:6681-6683`): "an absent
`bmr` skips step 4."

**Reproduction.**

```bash
cd /Users/glenn/src/github.com/truestamp/truestamp-cli
python3 - <<'EOF'
import json
d = json.load(open('samples/truestamp-item-01KPVAP639RSVPZCW2CBS51CTV.json'))
for c in d['cx']:
    if c.get('t') == 41: c['bmr'] = 'de'*32
json.dump(d, open('/tmp/badbmr.json','w'))
EOF
/tmp/truestamp verify samples/truestamp-item-01KPVAP639RSVPZCW2CBS51CTV.json --skip-external --type item > /tmp/c.txt 2>&1
/tmp/truestamp verify /tmp/badbmr.json --skip-external --type item > /tmp/b.txt 2>&1
diff /tmp/c.txt /tmp/b.txt && echo "IDENTICAL: bmr is never compared"
tail -3 /tmp/b.txt
```

```
IDENTICAL: bmr is never compared
  VERIFIED - proof is valid
  18 of 22 cryptographic checks passed  1 warning  3 skipped
```

**User-visible consequence.** The field is simultaneously over-required at parse time and unchecked
at verify time. The one shape the CLI rejects is the legitimate absent one; the one shape it
accepts is the tampered one. `bmr` is not in the `0x61` signed payload, so nothing else catches it.

**Fix.** After the `VerifyPartialMerkleTree` call at `internal/verify/verify.go:882-884`, compare
`mb.Header.MerkleRoot` in display byte order (`chainhash.Hash.String()`) against
`cx.BlockMerkleRoot` when the latter is non-empty, as its own `r.check` step. When it is empty emit
`r.skip` for that step with "no bmr supplied". Pair this with dropping the presence requirement at
`internal/proof/parse.go:203` (**F1**); emitting a `skip` row for the absent case is conformant
under E.25's containment rule, which matters because the Appendix D Bitcoin entry carries no `bmr`.

**Elixir reference.** `32ba10b989` "Cross-check Bitcoin bmr against txoutproof header". See
`lib/truestamp/proof/verification.ex:1029-1053` (`verify_bitcoin_block_merkle_root/3`), whose
absent-`bmr` branch emits a skip.

---

### F11 (high): none of E.3's `s.d` CBOR value-space rejections is implemented, and two of them collide distinct bundles onto one `0x11` digest

**What the CLI does.** `convertForJSON` (**`internal/proof/binary.go:361-391`**) is declared
`func convertForJSON(v any) any`, with **no error return**, so rejection is structurally
impossible. It is reached from `decodeSubjectCBOR` via `anyToJSON(m["d"])`
(`internal/proof/binary.go:149`, `:352`); the result becomes `bundle.RawData`
(`internal/proof/binary.go:128`), which `internal/verify/verify.go:182` JCS-canonicalizes and
hashes under `0x11`. The coercion therefore feeds the signed digest.

Observed behavior, verified against `fxamacker/cbor` v2.9.2:

| CBOR value in `s.d` | CLI result | Line |
|---|---|---|
| byte string `h'deadbeef'` | ACCEPT, `"deadbeef"` | binary.go:385-387 |
| text string `"deadbeef"` | ACCEPT, `"deadbeef"` (collides with the row above) | |
| integer map key `1: "x"` | ACCEPT, key silently dropped | binary.go:366-370 |
| text key `""` (legal JSON) | ACCEPT, key silently dropped (same guard) | binary.go:367 |
| `undefined` (0xf7) | ACCEPT, becomes `null` | binary.go:388-389 |
| `simple(99)` | ACCEPT, becomes the number `99` | |
| tag 1 (epoch time) | ACCEPT, RFC 3339 string **in local time** | |
| tag 0 | ACCEPT, RFC 3339 string | |
| tag 100 (unknown) | ACCEPT, becomes `{"Number":100,"Content":42}` (leaks Go struct field names into the hashed preimage) | |
| bignum (tag 2) | ACCEPT, becomes `{}` (value destroyed) | |

**What Appendix E requires.**

> map keys MUST be text strings; a non-text key is a hard rejection (E.6)
> (`whitepaper.typ:6149`)

> byte strings, tags other than the 55799 wrapper, `undefined`, and simple values other than
> `true` / `false` / `null` inside `s.d` MUST cause rejection, because they cannot round-trip to
> JSON
> (`whitepaper.typ:6150-6152`)

Plus the closing invariant that "the data hash derived from a CBOR bundle MUST equal the data hash
derived from the equivalent JSON bundle" (`whitepaper.typ:6153-6154`), and E.4's "no pre-processing,
no key filtering, and no value coercion" (`whitepaper.typ:6180-6181`).

**Reproduction (collision, on the shipped worked example).** Appendix D's `s.d.hash` is already a
64-hex-char text string. Re-encoding just that one value as a 32-byte CBOR byte string yields two
files with distinct wire digests whose converted JSON is byte-identical, and `truestamp verify`
prints identical output for both, including "Proof signature was verified using the embedded public
key". One signature covers two distinct wire bundles.

**Reproduction (invariant break).** `internal/proof/parse.go:141` sets
`bundle.RawData = subject.Data`, preserving the original JSON bytes, so a bundle carrying
`s.d[""] = "v"` keeps the key on the JSON path and drops it on the CBOR path. The two paths derive
different data hashes from the same logical bundle.

**Fix.** Change `convertForJSON` to return `(any, error)` and propagate through `anyToJSON` and
`decodeSubjectCBOR`. Return an error for `[]byte`, `cbor.Tag` / `cbor.RawTag`, `time.Time`
(a decoded tag 0/1), `cbor.SimpleValue` other than the true/false/null encodings, and `undefined`.
For maps, error on any non-string key rather than `continue`, and **test the type assertion's `ok`
value rather than the resulting string**, so a legal empty-string key survives while a non-string
key rejects. Surface the error out of `ParseCBOR` **and** the `convert proof` path, otherwise the
lossy form can be laundered into a JSON bundle that then verifies.

**Elixir reference.** `0b32a0b3ee` "Reject CBOR subject data that cannot round-trip to JSON".

---

### F12 (high): `--json` reduces every `skip` row to a bare count, so three D.4 rows are absent

**What the CLI does.** `JSONOutput` (`internal/verify/json_output.go:8-19`) has no `steps` array.
The `issues` loop skips every step that is not `StatusFail` or `StatusWarn` at
**`internal/verify/json_output.go:162-164`**, and `verification_notes`
(`internal/verify/json_output.go:188-200`) admits only steps in the `"Verification Notes"` group. A
`skip` matches neither, so it survives only as `summary.skipped`, a count with no group, category,
or message.

The text presenter behaves differently: `renderIssues` (`internal/verify/presenter.go:483-495`)
collects every non-pass step except Verification Notes, so it names each skip. The two surfaces
disagree.

**What Appendix E requires.** E.25's containment rule (`whitepaper.typ:6881-6888`): "no step that
D.4 reports may be absent from it". D.4 reports three `skip` rows: Key Binding
(`whitepaper.typ:6026`), Stellar Commitment (`:6027`), Bitcoin Commitment (`:6028`). E.17
(`whitepaper.typ:6607-6611`) supplies the reason the Key Binding row must exist: "so that a report
can never be read as having established a binding it never attempted."

**Reproduction.**

```bash
/tmp/truestamp verify samples/truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503.json \
  --skip-external --type block --json | python3 -c \
  "import json,sys; d=json.load(sys.stdin); print(sorted(d.keys())); print(d['summary'])"
```

```
['commitments', 'result', 'subject', 'subject_id', 'subject_type', 'summary', 'timeline']
{'passed': 13, 'failed': 0, 'warnings': 0, 'skipped': 3, 'total': 16}
```

No `issues` key at all, and no way to learn which three checks were skipped. The text run of the
same file names all three.

**User-visible consequence.** The Stellar and Bitcoin skips are partially recoverable from
`commitments.*.externally_verified` (though see **F4**, where that field lies). The **Key Binding
skip is not recoverable from the JSON at any point**, so a machine consumer gating on
`result == "verified"` cannot tell whether the embedded public key was ever cross-checked against
Truestamp's keyring.

**Fix.** Add a `steps` array to `JSONOutput` carrying `{group, category, status, message}` for
every entry in `Report.Steps`, built in `BuildJSONOutput` alongside `issues` in the E.22 category
display order (data_integrity, cryptographic, structural, timing, blockchain). `Step.Group` is
already json-tagged `"group"` at `internal/verify/report.go:96`. Keep `issues` and
`verification_notes` as filtered convenience views for backward compatibility.

---

### F13 (high): bare-map CBOR is rejected on the verify path, though E.3 requires accepting it and the package already can

**What the CLI does.** `IsCBORProof` (**`internal/proof/binary.go:21-23`**) returns true only when
the first three bytes are `0xd9 0xd9 0xf7`. `ParseBytes` calls `ParseCBOR` only when that sniff
succeeds (**`internal/proof/parse.go:34-36`**); everything else falls through to `json.Unmarshal`
at `internal/proof/parse.go:49`. `ParseCBOR` itself is tag-transparent
(`cborDecMode.Unmarshal` into `map[string]any`, `internal/proof/binary.go:36`) and decodes a bare
map fine, so **only the dispatcher is wrong**.

Wider than the verify path: `convert proof --from auto` (the default when `--from` is omitted)
calls `ParseBytes` at `cmd/convert_proof.go:71` **before** its `IsCBORProof || !looksLikeJSON` test
at line 75, so it fails identically. `InspectBundleType` (`internal/proof/download.go:166`) repeats
the same tag-only sniff.

**What Appendix E requires.**

> CBOR MUST be accepted both wrapped in the self-describing tag 55799 (byte prefix
> `0xd9 0xd9 0xf7`) and as a bare map.
> (`whitepaper.typ:6113-6115`)

E.25 item 1 (`whitepaper.typ:6851`): "Accept JSON and CBOR (tagged and bare)".

**Reproduction.**

```bash
cd /Users/glenn/src/github.com/truestamp/truestamp-cli
tail -c +4 samples/truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503.cbor > /tmp/bare.cbor
xxd -l 4 /tmp/bare.cbor            # a7 61 62 a5   -> CBOR major type 5, bare map
/tmp/truestamp verify /tmp/bare.cbor --skip-external --type block
/tmp/truestamp convert proof --from cbor --to json /tmp/bare.cbor | head -2
```

```
parsing proof: parsing JSON: invalid character '§' looking for beginning of value
{
  "b": {
```

One command in the binary reads the bytes and the verifier refuses them, with an error naming a
stray Latin-1 character and no hint that the input was well-formed CBOR.

**User-visible consequence.** No Truestamp-issued artifact is affected, because `MarshalCBOR`
always prepends the tag (`internal/proof/marshal_cbor.go:85-86`). This bites bundles from
third-party producers, which is precisely the interoperability case E.3's bare-map clause exists
for.

**Fix.** Smallest correct change: extend `IsCBORProof` (`internal/proof/binary.go:21-23`) to also
accept CBOR major type 5 (first byte in `0xa0`-`0xbf`) in addition to the tag prefix. That cannot
collide with JSON, since no JSON document begins with a byte in that range, and it repairs
`verify`, `convert --from auto`, and `download.go:166` at once. Keep the tag-only test available
under a different name for the places that genuinely want to know whether the tag is present.

**Elixir reference.** `Truestamp.Proof.Binary.decode/1` documents "Accepts both the self-describing
tag 55_799 form and a bare map."

---

### F14 (medium): a `cx` entry missing `ep` is accepted and produces a full report, where E.6 requires a hard rejection

**What the CLI does.** Nothing in either parse path requires `cx[].ep`. The JSON per-entry loop
(**`internal/proof/parse.go:92-96`**) validates only `cx[].t`; `validateSizes`
(**`internal/proof/parse.go:189-207`**) checks `tx`, `memo`, `op` and `bmr` but never
`EpochProof`; the CBOR loop (`internal/proof/binary.go:76-80`) is identical, and its `ep` decode
at `internal/proof/binary.go:200-208` is guarded by `if ep, ok := m["ep"]; ok`, so an absent key
leaves `EpochProof` at `""`.

That `""` reaches `internal/verify/verify.go:442`, decoding fails with "empty proof data", the CLI
records a `fail` and appends an `""` placeholder at `internal/verify/verify.go:445`, and
`verifyProofSignature` bails at `internal/verify/verify.go:501-506`.

**What Appendix E requires.** E.6's table row (`whitepaper.typ:6306`) maps "a `cx` entry is missing
`ep`" to `invalid_external_commitment_entry`. E.6's preamble (`whitepaper.typ:6285-6287`): these
"MUST abort before any report exists ... a hard rejection returns an error and produces no step
results". The rationale (`whitepaper.typ:6317-6320`) ties the gate to E.16's ability to assert
`N == length(cx)` unconditionally, and E.16 (`whitepaper.typ:6576-6579`) states outright: "A
verifier MUST NOT emit a placeholder for a rootless entry".

**Reproduction.** Delete `cx[0].ep` from any bundle the CLI accepts:

```bash
/tmp/truestamp verify /tmp/no_ep.json --skip-external --type item --json | python3 -c \
  "import json,sys; print(json.load(sys.stdin)['summary'])"
```

```
{'passed': 16, 'failed': 2, 'warnings': 1, 'skipped': 3, 'total': 22}
```

The reference verifier prints `REJECTED: :invalid_external_commitment_entry` and produces no report.

**User-visible consequence.** **Not a soundness hole.** The placeholder discipline keeps
`N == length(cx)` intact and the guard at verify.go:501-506 refuses to build a payload, so no
rootless entry becomes signature-transparent and the verdict is still `failed` with exit 1. The
harm is the E.23 error vocabulary and 16 passing step results attached to a bundle that should
never have been graded.

**Fix.** In the existing per-entry loop at `internal/proof/parse.go:92-96`, add
`if cx.EpochProof == "" { return nil, fmt.Errorf("cx[%d]: invalid external commitment entry: missing ep", i) }`,
and mirror it in `internal/proof/binary.go:76-80` so both serializations abort identically. The
chain root key half of the E.6 pair is already enforced by `validateSizes`. Once the gate lands, the
placeholder branches at verify.go:445/452/459 and the guard at 501-506 become unreachable
defensive code.

**Elixir reference.** `f125261688` "Hard-reject cx entries with no epoch proof or epoch root".

---

### F15 (medium): an absent or zero `v` is a hard rejection, though E.6 names `v` as its single explicit exception

**What the CLI does.** `internal/proof/parse.go:40` declares `Version int` (a value, not a pointer
like `T *uint16` on the very next line), so `encoding/json` leaves both an absent `v` and an
explicit `"v": 0` at zero. **`internal/proof/parse.go:55-57`** aborts both with
`missing required field: v`, which is factually wrong for the latter. The CBOR path repeats it at
`internal/proof/binary.go:41-43` via `toInt(raw["v"])`, whose `default: return 0`
(`internal/proof/binary.go:314-327`) collapses the two cases the same way. A third site,
`internal/proof/download.go:49,57`, applies `shape.Version == 0` as a shape gate on remotely
fetched bundles.

`v: 2` is handled correctly: it parses, and `verifyVersion` (`internal/verify/verify.go:301-304`)
fails it as a step.

**What Appendix E requires.**

> *Exception.* The version `v` is NOT a hard gate. A wrong `v` produces a failing step (E.8), not a
> rejection.
> (`whitepaper.typ:6322-6323`)

E.8 (`whitepaper.typ:6347`): "`v` MUST equal 1. Otherwise this step fails. Failure here does not
abort the run." E.6's table contains no `v` row. E.23 (`whitepaper.typ:6817`) types `version fail`
as a step fail raised by E.8.

**Reproduction.**

```bash
python3 -c "import json;d=json.load(open('samples/truestamp-item-01KPVAP639RSVPZCW2CBS51CTV.json'));d['v']=0;json.dump(d,open('/tmp/v0.json','w'))"
/tmp/truestamp verify /tmp/v0.json --skip-external --type item ; echo "exit=$?"
```

```
parsing proof: missing required field: v
exit=1
```

The reference verifier on the same file emits
`[FAIL] Structure  unsupported proof format version 0 (expected 1)` inside a full report.

**Fix.** Change `internal/proof/parse.go:40` to `Version *int` and carry the decoded value,
including 0, into `bundle.Version`; drop the rejection at lines 55-57. Replace the `version == 0`
check at `internal/proof/binary.go:41-43` with a presence test on `raw["v"]`, and relax the
`shape.Version == 0` clause at `internal/proof/download.go:57`. `verifyVersion` already fails any
non-1 value, and `internal/verify/verify.go:516` uses `byte(bundle.Version)`, so a nil-defaulted 0
builds a payload and yields a signature step fail inside a normal report.

**Note.** For an entirely absent `v` the reference verifier itself currently crashes
(`whitepaper/verify_proof.exs:1185`, `ArgumentError` on a nil integer segment). That is a defect in
the reference script, not evidence about the CLI. `v: 0` is the clean oracle-backed case.

---

### F16 (medium): ten parse-time rejections that Appendix E specifies as reported step results

**What the CLI does.** `expectHexBytes` (`internal/proof/parse.go:213-216`) conflates "field is
empty" with "field is required" for every hex field it touches, and `ParseBytes` separately rejects
empty `pk`/`sig`/`s.id`/`s.mh`/`s.kid`. All abort before `runBundle` builds a `Report`:

| Condition | Line |
|---|---|
| `pk` / `sig` absent | `internal/proof/parse.go:65-70` |
| `s.id` / `s.mh` / `s.kid` absent | `internal/proof/parse.go:133-135` |
| `pk` / `sig` wrong decoded length | `internal/proof/parse.go:156-165` |
| `b.ph` / `b.mr` / `b.mh` / `b.kid` absent or wrong size | `internal/proof/parse.go:167-178` |
| `s.mh` / `s.kid` wrong size | `internal/proof/parse.go:180-187` |

**What Appendix E requires.** None appears in E.6's table (`whitepaper.typ:6300-6310`), and E.6's
preamble makes the classification itself normative: "The distinction between a hard rejection and a
failed step (E.8 onward) is itself normative: a hard rejection returns an error and produces no step
results" (`whitepaper.typ:6285-6287`). Specifically:

- E.10 (`whitepaper.typ:6410`): "Any missing input field fails this step and MUST cause E.13 and
  E.16 to report 'cannot verify' rather than pass." E.10's inputs are `s.d`, `s.id`, `s.mh`,
  `s.kid`, `t` (`whitepaper.typ:6375`).
- E.23 (`whitepaper.typ:6818`) types `public key not 32 bytes` as a step fail raised by E.9;
  E.9 (`whitepaper.typ:6369-6371`) requires it to "abort the signature check (E.16) with an
  explicit 'cannot verify' result, never a silent pass".
- E.14 (`whitepaper.typ:6508`): "Inputs: `b.id`, `b.ph`, `b.mr`, `b.mh`, `b.kid`. All five MUST be
  present"; E.16 closes with "An upstream derivation failure MUST produce 'cannot verify proof
  signature (missing derived data)', never a pass".

**Reproduction.** Each is one deleted key:

```
del s.mh   -> parsing proof: subject missing required fields (id, mh, kid)   exit 1
del b.ph   -> parsing proof: b.ph is required                                exit 1
del pk     -> parsing proof: missing required field: pk                      exit 1
pk="AAAA"  -> parsing proof: pk must be 32 bytes, got 3                      exit 1
```

The reference verifier produces a full report for each: a Data Integrity `[FAIL] Subject Data
missing fields for the composite subject hash` for the first, `[FAIL] Block Hash  block is missing
one of the five required fields` for the second, `[FAIL] Signing Key` plus the E.16 cannot-verify
for the last two.

**User-visible consequence.** No valid proof is rejected and no invalid one accepted: the CLI is
strictly stricter and exits 1 either way. The cost is diagnostic precision, cross-verifier report
comparability (which E.23 makes a MUST), and one structurally unreachable result (see **F26**).

**Fix.** Split `expectHexBytes` into a presence-agnostic shape check that returns `nil` for `""`,
and delete the checks at `internal/proof/parse.go:65-70`, `:133-135` and `:156-165`. Most of the
step machinery already exists: `verifySigningKey` (`internal/verify/verify.go:261-266`) already
emits a Cryptographic step fail for an undecodable or wrong-length `pk`, and `deriveBlockHash`
(`internal/verify/verify.go:418-423`) already emits one when `ComputeBlockHash` errors. The only
genuinely new work is re-keying the missing-composite-fields branch of `deriveItemHash`
(`internal/verify/verify.go:377-380`) to `CatDataIntegrity` (see **F26**).

**Elixir reference.** `ed0f6f017b` "Report, never raise, on legitimate and malformed proof bundle
shapes".

---

### F17 (medium): no E.23 error name appears anywhere in the parser, and one rejection leaks the parser's private struct

**What the CLI does.** `grep` for any of E.23's nine hard-rejection identifiers across the whole
repo returns zero hits. Every abort is an English sentence. Two cases are worse than merely
non-standard:

- A non-object JSON input surfaces the raw `encoding/json` type error from
  `internal/proof/parse.go:50-51`, which prints the entire anonymous decode struct from
  `internal/proof/parse.go:39-49` into user-facing output.
- JSON literal `null` unmarshals cleanly into the struct as a no-op and falls through to the `v`
  gate, so it is reported as a missing field rather than as a non-object.

**What Appendix E requires.**

> Independent verifiers MUST agree on the vocabulary for rejections and failures so their reports
> are comparable.
> (`whitepaper.typ:6793-6794`)

The nine snake_case names are at `whitepaper.typ:6808-6816`; E.6 row 1
(`whitepaper.typ:6300`) maps "input is not an object" to `not_a_json_object`.

**Reproduction.**

```bash
printf '[1,2]' > /tmp/jarr.json && /tmp/truestamp verify /tmp/jarr.json
printf 'null'  > /tmp/jnull.json && /tmp/truestamp verify /tmp/jnull.json
```

```
parsing proof: parsing JSON: json: cannot unmarshal array into Go value of type
  struct { Version int "json:\"v\""; T *uint16 "json:\"t\""; Timestamp string "json:\"ts\""; ... }
parsing proof: missing required field: v
```

The reference verifier returns `REJECTED: :not_a_json_object` for both. Adding `--json` changes
nothing: the same prose, exit 1, with no structured code for a consumer to compare.

**Fix.** Introduce `proof.RejectionError{Code string; Detail string}` carrying the E.23 identifier
and return it from every abort site in `ParseBytes` / `ParseCBOR` / `validateSizes` /
`expectHexBytes`. Before the `json.Unmarshal` at `internal/proof/parse.go:50`, check that the first
non-whitespace byte is `{` and return `not_a_json_object` otherwise, which catches the `null`,
array, string and number cases without leaking the struct type. Surface `Code` in `--json`.

---

### F18 (medium): failing steps carry the passing step's message verbatim

**What the CLI does.** `Report.check` (**`internal/verify/report.go:324-330`**) routes the same
`msg` string to `r.pass` (line 304) and `r.fail` (line 308):

```go
func (r *Report) check(group, category string, ok bool, msg string) {
    if ok { r.pass(group, category, msg) } else { r.fail(group, category, msg) }
}
```

All 12 `r.check` call sites in `internal/verify/verify.go` inherit this: lines 294, 296, 297, 298,
302, 413, 463, 534, 568, 861, 870, 894. The pretty renderer masks it behind a red glyph;
`--json` (`internal/verify/json_output.go:169-182`) copies the raw string next to
`"severity": "error"`.

**What Appendix E requires.**

> a failing generic message MUST be prefixed so it never reads as a positive assertion
> (`whitepaper.typ:6787`)

**Reproduction.** Flip one nibble of `b.mr` in any accepted bundle:

```json
{"severity": "error", "category": "cryptographic",
 "message": "Inclusion proof to block 019f93ff-2600-7c30-8000-000000000c30 (0 steps)"}
{"severity": "error", "category": "cryptographic",
 "message": "Proof signature valid (Ed25519)",
 "detail": "The proof may have been tampered with or signed with a different key."}
```

The `detail` field mitigates only some rows: `lookupFailureDetail`
(`internal/verify/presenter.go:595`) matches substrings from `failureDetails`
(`internal/verify/presenter.go:578-593`), which has `"proof signature"` but not
`"inclusion proof"`. So the inclusion-proof failure ships with **no negating text anywhere in the
record**.

**User-visible consequence.** The verdict, exit code and `severity` field are all correct, so no
proof is misjudged. The exposure is any log line, screenshot, or machine consumer that reads
`message` without `severity`, which then reads a forged proof as verified.

**Fix.** Extend the signature to `check(group, category string, ok bool, passMsg, failMsg string)`
and give each of the 12 call sites an explicit negative phrasing, mirroring the reference verifier's
`Report.check/6` (`whitepaper/verify_proof.exs:440-444`, whose fail arms read "proof signature
invalid (Ed25519)" and "derived root does not match block merkle root"). A minimal alternative is to
prefix the fail arm in `check` at `internal/verify/report.go:328`. Note the `"keyring"` entry in
`failureDetails` is itself positively phrased even when attached to a failure (see **F8c**).

---

### F19 (medium): E.13's required inapplicability skip is omitted for block-like subjects

**What the CLI does.** **`internal/verify/verify.go:191-194`**:

```go
// Step 7: Inclusion Proof (skipped for block-like subjects)
if !isBlockLike {
    verifyInclusionProof(r, subjectHash, bundle.InclusionProof, block)
}
```

No `else`. Every emission of `groupInclusion` lives inside `verifyInclusionProof`
(`internal/verify/verify.go:397, 403, 409, 413`), so for `t` in {10, 11} no Inclusion Proof step of
any status is appended.

**What Appendix E requires.**

> For block-like subjects this step MUST be reported as skipped ("not applicable"), not passed and
> not failed.
> (`whitepaper.typ:6497-6498`)

E.22 (`whitepaper.typ:6783-6786`) makes it REQUIRED and "reported, not omitted, so a reader can tell
a check that did not apply from a check that was never reached." E.25 item 7
(`whitepaper.typ:6861`) restates it as a conformance MUST.

**Reproduction.**

```bash
/tmp/truestamp verify samples/truestamp-beacon-019db753-4188-7692-b487-9f8c5b805503.json \
  --skip-external --type block 2>&1 | grep -ci inclusion
```

```
0
```

The reference verifier on the same fixture emits `[SKIP]  Inclusion Proof   not applicable: a
block-like subject IS the block, so there is no leaf to prove`.

**Fix.** Add the `else` arm at `internal/verify/verify.go:194`:

```go
} else {
    r.skip(groupInclusion, CatCryptographic,
        "Inclusion proof not applicable to block-like subjects (subject hash is the block hash)")
}
```

Both helpers already exist (`r.skip` at `internal/verify/report.go:312`, `groupInclusion` at
`internal/verify/verify.go:240`). Pair with **F12** so the row is machine-visible.

**Do not** also add an unconditional Hash Comparison skip for non-item subjects with no `--hash`.
E.7 scopes that MUST to the hash-supplied case, and the reference verifier emits no row there.

---

### F20 (medium): `truestamp convert proof` re-parses `s.d` through `float64` and silently rounds, on both the CBOR and the pretty-JSON paths

Same root cause as **F3**, different call sites and a different fix.

**What the CLI does.**

- `MarshalCBOR` reconstructs the subject data by unmarshalling the preserved `RawData` into an
  untyped `any` at **`internal/proof/marshal_cbor.go:104-113`**, so `encoding/json` fills every
  number as `float64`. `fxamacker`'s shortest-float option then downgrades further: inspecting the
  emitted bytes, an integer is written as `fa 5a000000` (float32) and small integers as float16.
- `prettyJSON` (**`cmd/convert_proof.go:175-181`**) performs the identical
  `json.Unmarshal(raw, &v)` before `json.MarshalIndent`. Pretty is the default for `--to json`, so
  that path rounds too, even though `MarshalJSON` and `internal/proof/parse.go:141` are both
  byte-preserving. `--compact` bypasses it.

**Reproduction.**

```bash
/tmp/truestamp convert proof --to cbor <bundle with a >2^53 integer in s.d> > /tmp/e30.cbor
/tmp/truestamp convert proof --from cbor --to json --compact /tmp/e30.cbor | grep -o '"ledgerSequence":[0-9]*'
```

Emits `...992` where the input carried `...993`, exit 0, empty stderr.

**User-visible consequence.** The CLI hands the user a file whose `s.d` no longer hashes to what its
own `sig` covers. Run against the reference verifier, the original bundle is `VERDICT: PASSED` with
one warn and the converted file is `VERDICT: FAILED`. One silent command moves a bundle from passing
to failing.

The documented guarantee at `internal/proof/marshal_cbor.go:32-34` ("`cbor -> Parse -> MarshalCBOR`
is byte-stable for inputs that are themselves deterministically encoded") is false for this input.

**Fix.** At `internal/proof/marshal_cbor.go:104`, decode with `json.Decoder` + `UseNumber()` so
integers survive as `json.Number`, and map each to a CBOR integer rather than a float. At
`cmd/convert_proof.go:175-181`, either apply the same treatment or drop the reparse in favour of
`json.Indent`, which reformats bytes without touching numbers. Where a value genuinely cannot be
represented exactly, return an error naming the offending key rather than emitting a corrupted
bundle at exit 0.

---

### F21 (medium): the JSON and CBOR paths implement the E.6 block-like shape rule differently

**What the CLI does.** Two readings of one rule:

| Path | "carries `s`" | "carries `ip`" |
|---|---|---|
| JSON, `internal/proof/parse.go:113` and `:116` | present and not `null` | present and non-empty |
| CBOR, `internal/proof/binary.go:97` and `:100` | key present at all | present and not `null` |

**What Appendix E requires.** E.6 row (`whitepaper.typ:6308`) states one rule: "a block-like `t`
(10, 11) carries `s` or `ip`" maps to `unexpected_subject_fields_for_block_like`. Both Elixir
surfaces resolve "carries" identically: `whitepaper/verify_proof.exs:623` uses
`block_like? and (not is_nil(bundle["s"]) or not is_nil(bundle["ip"]))`, and
`lib/truestamp/proof/verification.ex:190` uses `if is_nil(map["s"]) and is_nil(map["ip"])`. So the
normative reading is: **an explicit `null` counts as absent for both fields, and any non-null value
counts as carried, including an empty string.**

Measured against that, each Go path deviates on a different field:

| Variant | Oracle | CLI JSON | CLI CBOR |
|---|---|---|---|
| `s: null` | ACCEPT | ACCEPT (match) | REJECT (deviates) |
| `ip: ""` | REJECT | ACCEPT (deviates) | REJECT (match) |
| `ip: null` | ACCEPT | ACCEPT | ACCEPT |

**User-visible consequence.** Low exposure: the CLI's own encoders omit `s` and `ip` entirely for
block-like bundles (`internal/proof/marshal_cbor.go:65`, `internal/proof/types.go:83`), so no bundle
round-tripped through Truestamp tooling can hit either branch. It bites a third-party CBOR producer
that emits explicit nulls for absent fields, a plausible habit given that `priv/cddl/proof.cddl`
already uses null-for-absent for the Bitcoin offline payload.

**Fix.** Extract one helper and call it from both `ParseBytes` (`internal/proof/parse.go:111-118`)
and `ParseCBOR` (`internal/proof/binary.go:95-102`), implementing the oracle rule: treat a field as
absent when the key is missing OR its value is `null`, and as carried for any other value including
an empty string or empty byte string. Concretely, CBOR must change
`if _, ok := raw["s"]; ok` to also require a non-nil value, and JSON must change
`*raw.InclusionProof != ""` to reject on any non-null `ip`. **Do not** adopt the JSON reading
wholesale: making an empty `ip` count as absent would put both paths in conflict with the oracle.

---

### F22 (low): no constant-time comparison anywhere

**What the CLI does.** `HexEqual` (**`internal/tscrypto/hash.go:296-314`**) returns early on a
length mismatch (lines 298-300) and again on the first differing character (lines 309-311), so its
running time is data-dependent. `crypto/subtle` is imported nowhere in the repo. Every hash
comparison routes through it: the Merkle root at `internal/tscrypto/merkle.go:50` (reached from
`internal/verify/verify.go:407` and `:456`), the E.7 expected-hash compare at
`internal/verify/verify.go:136`, the E.19 OP_RETURN and txid compares at
`internal/verify/verify.go:862` and `:871`, and the E.21 entropy re-fetch compares at
`internal/verify/verify.go:723` and `:766`. The Bitcoin partial-Merkle path additionally uses the
short-circuiting `chainhash.Hash.IsEqual` (`internal/bitcoin/merkle.go:56`, `:114`,
`internal/verify/verify.go:888`).

**What Appendix E requires.**

> *Constant-time comparison*: all hash and digest comparisons MUST be constant-time.
> (`whitepaper.typ:6171`)

Restated inside two of the steps E.1 defines as conforming behavior: E.13
(`whitepaper.typ:6496`) and E.7 (`whitepaper.typ:6338`).

**User-visible consequence.** For an offline CLI every operand is public and the attacker already
holds the bundle, so there is no practical timing oracle. The concrete failure is conformance. Note
the "networked context" caveat is already present rather than hypothetical: verify.go:723 and :766
compare a bundle-carried hash against a value just fetched from
`external.GetStellarLedger` / `external.GetBitcoinBlockHeader`.

**Fix.** Replace the body of `HexEqual` with a hex decode of both operands followed by
`subtle.ConstantTimeCompare`, keeping the length check outside the constant-time region (a length
difference is not secret) and doing case folding during hex decode rather than during comparison.
This mirrors `whitepaper/verify_proof.exs:244-247`, which does the length check outside and
`:crypto.hash_equals/2` inside.

**Note.** E.25's checklist does not enumerate this MUST, because that checklist collects the MUSTs
an implementer can self-certify against an Appendix C or D vector, and timing behavior is not
vector-testable. That explains why no fixture catches it; it is not a carve-out from E.4.

---

### F23 (low): `ValidateClaimsHash` checks hex length only, never the lowercase-hex character set

**What the CLI does.** `ValidateClaimsHash` (**`internal/tscrypto/hash.go:354-370`**) looks the
algorithm up in `hashTypes` and compares `len(hash)` to `info.Bytes * 2`. It never inspects a
character. `validateClaimsHashType` (`internal/verify/verify.go:331-334`) then emits
`r.pass(groupSubjectData, CatDataIntegrity, "Claims hash length valid for %s")`.

**What Appendix E requires.**

> check that the hex length equals twice the algorithm's output size and that the character set is
> lowercase hex, for one of the twelve registered algorithms
> (`whitepaper.typ:6418-6419`)

E.11's disposition is fixed at `whitepaper.typ:6415-6416`: "All checks in E.11 MUST warn and MUST
NOT fail." The twelve algorithms and widths in `hashTypes` (`internal/tscrypto/hash.go:337-350`) are
exactly E.11's twelve and are correct; only the charset half is missing.

**Reproduction.** Set `s.d.hash` to 64 repetitions of `z`, or uppercase the genuine hash, with
`hash_type: "sha256"`. The CLI emits zero warnings; the default text output prints
`Hash zzzz...zzzz (sha256) - not verified` with no indication the value is not hex. The reference
verifier warns: `claimed hash is not 64 lowercase hex characters for sha256`
(implemented at `whitepaper/verify_proof.exs:942`).

**Fix.** After the length check at `internal/tscrypto/hash.go:367`, scan the string and return an
error naming the offending character when any byte falls outside `0-9a-f`. Keep the result warn-only
at `internal/verify/verify.go:332` per E.11.

---

### F24 (low): `ExtractOpReturn` selects the first output that parses as standard nulldata, not the first output whose script begins `0x6a`

**What the CLI does.** **`internal/bitcoin/parse.go:56-63`** gates each output on
`txscript.IsNullData(pkScript)` and then on `len(pushes) > 0`, continuing the scan past any output
that fails either. btcd's `isNullDataScript` additionally requires the script to tokenise to exactly
one push, the opcode to be a small int or at most `OP_PUSHDATA4`, and the data to be at most
`MaxDataCarrierSize` (80 bytes). Three classes of earlier output are therefore silently skipped:

- an output starting `0x6a` that `IsNullData` rejects (truncated push, oversize push, multi-push)
- an output whose push data does not parse
- a canonical bare `0x6a` with zero pushes, which `IsNullData` accepts but the `len(pushes) > 0`
  clause skips

**What Appendix E requires.**

> the `OP_RETURN` payload is the push data of the FIRST output whose script begins with `0x6a`,
> with the expected grammar `OP_RETURN 0x6a`, `OP_PUSHBYTES_32 0x20`, then 32 bytes. A verifier that
> scans last-match or requires exactly-one output disagrees with generation
> (`whitepaper.typ:6657-6659`)

**Reproduction.** Verified by compiling the function verbatim against the pinned btcd modules. For
first outputs `6a20`+31 bytes, `6a4c51`+81 bytes, bare `6a`, and `6a20<32>0199`, the CLI returns a
later output's payload in every case, while a first-`0x6a` selector returns that first output's
payload or an error. A single canonical nulldata output (what generation emits) agrees.

**User-visible consequence.** Unreachable for Truestamp-generated bundles, so it needs an
adversarially supplied `rtx`, and the payload the CLI selects is still the genuine epoch root
already matched at E.15. Only the per-check line at `internal/verify/verify.go:857-864` diverges.

**Fix.** In `internal/bitcoin/parse.go:56-63`, select the output with
`bytes.HasPrefix(txout.PkScript, []byte{0x6a})` first (`bytes` is already imported at line 7), run
`txscript.PushedData` on that output alone, and return its parse error instead of continuing. Return
a distinct error when that output yields zero pushes, and "no OP_RETURN output found" only when no
output starts with `0x6a`.

**Elixir reference.** `5badf3a4cf` "Fix OP_RETURN output selection and the entropy report group".

---

### F25 (low): `b.id` presence is not enforced, so Block Hash reports `pass` over a 121-byte preimage

**What the CLI does.** `validateSizes` checks `b.ph`, `b.mr`, `b.mh` and `b.kid`
(`internal/proof/parse.go:167-178`) but not `b.id`. `deriveBlockHash`
(**`internal/verify/verify.go:418-427`**) calls `tscrypto.ComputeBlockHash` with no presence check,
and `ComputeBlockHash` (`internal/tscrypto/hash.go:127`) frames an empty id as `len32(0)` via
`HexToBytes` (`internal/tscrypto/hash.go:36-41` returns an empty slice with a nil error). The result
is a 121-byte hashed preimage instead of 157, a nil error, and
`r.pass(groupBlockHash, CatCryptographic, "Block hash derived (0x32)")` at line 425.

**What Appendix E requires.** E.14 (`whitepaper.typ:6508-6510, 6525-6530`): "Inputs: `b.id`, `b.ph`,
`b.mr`, `b.mh`, `b.kid`. All five MUST be present" and "The preimage is 157 bytes, unconditionally
... There is no nil branch".

**Reproduction.** Delete `b.id`. The run returns verdict `failed` with three errors (the structural
"Block present with ID", the epoch walk, and the signature). The epoch entry reports the walk
mismatch, **not** `internal/verify/verify.go:435-437`'s "Cannot verify epoch proofs (no block
hash)", which proves `deriveBlockHash` returned a non-empty digest and emitted its pass.

**User-visible consequence.** `verifyStructure` (`internal/verify/verify.go:296`) fails
`block.ID != ""` in the same run, so the verdict stays FAILED. The defect is that a derivation step
asserts a success it did not achieve, and verdict correctness depends on an unrelated structural
check. Note neither renderer prints pass rows, so this is visible as the aggregate count rather than
as a printed line.

**Fix.** Add a presence guard at the top of `deriveBlockHash` covering all five fields, emitting
`r.fail(groupBlockHash, CatCryptographic, "Cannot derive block hash (missing inputs)")` and
returning `""` so the downstream epoch and signature steps report "cannot verify", matching the
reference verifier. Optionally assert the 157-byte budget inside `ComputeBlockHash`.

---

### F26 (low, conditional): the composite-hash "missing inputs" failure is filed under Cryptographic where E.22 assigns Data Integrity

**Read section 5 before touching this.** The Go CLI is the incumbent authority for report
categories, and this is the one row where the amended exception does not match the CLI.

**What the CLI does.** `internal/verify/verify.go:379` (item) and `internal/verify/verify.go:954`
(entropy) hardcode `CatCryptographic` for "Cannot derive item hash (missing inputs)" /
"Cannot derive observation hash (missing inputs)". Each of those single predicates conflates two
distinct spec cases:

- `claimsHash == ""` (subject data absent or redacted). E.22 assigns **Cryptographic**. The CLI is
  correct here, and this is the only reachable trigger today.
- `subject.MetadataHash == ""` or `subject.SigningKeyID == ""`. E.22 assigns **Data Integrity**.
  Unreachable today, because `internal/proof/parse.go:133-135` rejects the bundle first (**F16**).

The other two limbs of the exception already match: `internal/verify/verify.go:332` and `:334` file
E.11's hash-shape check under `CatDataIntegrity`, and `verifyClaimsTimestamp` files stale and
future-dated claim warnings under `CatTiming`.

**What Appendix E requires.** The Subject Data category exception
(`whitepaper.typ:6762-6773`) names "exactly two" Data Integrity cases: E.11's hash-shape check, and
"E.10's failure when the fields needed for the composite subject hash are missing, so no composite
hash is derived".

This is **not** a transcription artifact of an Elixir-only rule. `whitepaper/verify_proof.exs:817-826`
hardcodes `:data_integrity` for that branch with a comment tying it to E.22, and
`lib/truestamp/proof/verification.ex:384` and `:404` emit "Missing fields for composite hash", routed
to `data_integrity`.

**Fix (only if F16 is fixed).** Split the `internal/verify/verify.go:379` and `:954` predicates so
the `mh`/`kid` limb reports `CatDataIntegrity` while the absent-subject-data limb stays
`CatCryptographic`. Leave `internal/verify/verify.go:375` and `:950` as `CatCryptographic`. If F16 is
not fixed, this branch remains unreachable and no change is warranted.

---

## 4. What the CLI already gets right

This section exists so the next author does not re-audit settled ground.

### Verified conformant by direct testing

- **All twelve tamper classes detected.** `s.d` edited, `s.mh`, `s.kid`, `s.id`, `b.mr`, `b.ph`,
  `b.mh`, `b.kid`, a tampered epoch root, reordered `cx` entries, `ts` shifted, `t` changed,
  `sig` replaced, `pk` replaced, `ip` truncated. Every one FAILED on both the CLI and the reference
  verifier.
- **E.6 hard rejections**, all of them except the two filed as **F14** (`ep`) and **F15** (`v: 0`):
  missing `t`, `t = 99`, `t = 21`, `t = 12`, `t = 41` used as a subject code, missing `b`, empty
  `cx`, `cx` not a list, `cx[].t = 42`, missing `memo` on a `t=40` entry, missing `op` on a `t=41`
  entry, a block-like subject carrying `s` or `ip`, a non-block-like subject missing `s`, and one
  missing `ip`. All REJECT on both verifiers.
- **E.12 compact-proof decoding.** `internal/tscrypto/merkle.go:61-114` uses
  `base64.RawURLEncoding` (unpadded), enforces `maxProofDepth = 64` (const at line 12, checked at
  line 81), enforces the exact size equation `1 + (depth+7)/8 + depth*32` with a `!=` rejection at
  line 88, and reads the bit as `(bitfield[byteIdx] >> bitIdx) & 1` (little-endian). Depth-65,
  trailing-byte and short-`ip` cases all agree with the oracle.
- **E.15 epoch-root dispatch.** `epochTarget` (`internal/verify/verify.go:474-483`) switches on
  `cx.Type` returning `MemoHash` / `OpReturn`, which is exactly E.15's "MUST dispatch on `t`, not on
  key presence".
- **E.16 `N == length(cx)`.** The placeholder discipline at `internal/verify/verify.go:445/452/459`
  keeps `len(epochRoots) == len(commits)`, and the guard at `internal/verify/verify.go:501-506`
  refuses to build a payload when any root is empty. No rootless `cx` entry is signature-transparent.
- **E.18 unknown-`t` rejection.** `internal/proof/parse.go:62-64` and
  `internal/proof/binary.go:50-52` both reject an unknown subject code before any report exists.
- **JCS key ordering.** UTF-16 code-unit ordering is correct: `truestamp jcs` orders
  `a` < U+1F600 < U+FB00, matching the reference.
- **`null` inside `s.d` is not dropped**, and floats in `s.d` canonicalize correctly.
- **All five report statuses exist** (`internal/verify/report.go:16-32`) and `warn`, `info` and
  `skip` are all actually emitted (`r.warn` 4x, `r.info` 3x, `r.skip` 9x).
- **Report categories.** The CLI's Blockchain / Cryptographic / Structural / Data Integrity /
  Timing assignments match E.22's table, including Entropy Source under Blockchain at all 24 call
  sites. See section 5.
- **Temporal info.** D.4's amended preamble (`whitepaper.typ:5993-5997`) blesses recording
  `submitted_at` / `committed_at` in a temporal field rather than as a step, which is what
  `internal/verify/verify.go:573-585` does.
- **`--skip-external` handling** is correct everywhere it is reached: it produces `skip` rows, not
  fails. The defects in F6, F7 and F8 are all in the transport-error branch, not the flag branch.

### Elixir audit defect ids that do NOT apply to the CLI

The Elixir audit is at
`/Users/glenn/src/github.com/truestamp/truestamp-v2/tmp/verifier-audit/DEFECT-REGISTER.md`.

**Cleared (the CLI is already correct):**

| Id | Subject | Why cleared |
|---|---|---|
| D04 | Crash on malformed `b.mr` | Go has no exception path, and `internal/proof/parse.go:170` rejects nil or short up front. |
| D08 | Compact-proof decoder | Enforces base64url-unpadded, depth <= 64, the exact size equation and the little-endian bit index. |
| D18 | Unknown `t` accepted | Rejected at parse on both serializations. |
| D24 | Epoch roots dispatched on `memo \|\| op` | The CLI dispatches on `cx[].t`. |
| D30 | Missing report statuses | All five exist and are emitted. |
| D32 | Temporal Info step | D.4's amended preamble blesses the CLI's `temporal` field. |
| D35 | Entropy Source category | Filed under Blockchain at every call site. |

**Not applicable (Elixir, Livebook or whitepaper-only surfaces):** D05, D06, D07, D09, D14, D15,
D16, D17, D19, D20, D21, D22, D23, D27, D28, D31, D33, D34.

Two nuances:

- **D07 and D21**'s substance (optional `cx` fields treated as required) does recur in the Go
  parser and is covered by **F1**.
- **D14**'s substance (a fabricated Bitcoin header accepted as external confirmation) does **not**
  recur in the same form: the CLI implements the real E.19(b) binding lookup at
  `internal/external/bitcoin.go:39-74` and skips when it is unavailable. The related **F4** is a
  narrower reporting defect in `CommitmentInfo.Skipped`, not a missing lookup.
- **D10**'s soundness half does not recur. The Elixir surfaces silently dropped a rootless `cx`
  entry, shrinking `N` below `length(cx)`; the CLI keeps a positional placeholder and refuses to
  build the payload. Only the report-shape half recurs, as **F14**.

---

## 5. Notes on authority: two things not to "fix"

### 5.1 Report categories: the Go CLI is the incumbent

Appendix E was **amended to document the Go CLI's categorization**, not the other way round. The
E.22 Subject Data category exception (`whitepaper.typ:6762-6773`) exists because the Elixir audit
deliberately re-keyed several Subject Data steps to Data Integrity and Timing "to match the Go
CLI's categorization".

**Do not change any category assignment in this repo to match some other implementation.** The
CLI's Blockchain / Cryptographic / Structural / Data Integrity / Timing mapping is the reference.

The single exception is **F26**, and it is filed conditionally: the CLI disagrees with the exception
*as now written* on exactly one row (the missing-composite-fields case), and that row is unreachable
today because the parser rejects the bundle first. Fix it only if you also fix **F16**.

Note also that E.22 states the exception is "verdict-neutral: the verdict rule below reads each
step's status, never its category". Every other finding in this document is about a step's **status**
(`fail` vs `skip` vs `pass`) or about whether a step exists at all. None of them asks you to change a
category, and none of them is shielded by the incumbency.

### 5.2 Places where the spec was clarified rather than the code changed

The following were adjudicated during the Elixir pass and are **settled**. Do not re-litigate them,
and do not "fix" the CLI toward the pre-amendment reading:

- **E.5's three-column grading.** Deciding that `tx`, `bmr`, `rtx` and `txp` are not structurally
  required was a deliberate resolution, recorded in
  `/Users/glenn/src/github.com/truestamp/truestamp-v2/tmp/verifier-audit/DECISIONS.md` (S1-S5). The
  older reading, in which the CDDL's presence of a key implied a requirement, is retired.
- **E.25's one-way containment.** "Match D.4" means containment, not row-for-row identity. Extra
  `skip` and `info` rows are conformant; extra `fail`, `pass` or `warn` rows are not. This is why
  several fixes above say to use `r.info` or `r.skip` and explicitly **not** `r.warn`.
- **E.13 and E.7 inapplicability skips are required rows**, not optional. That was clarified, not
  invented.
- **The E.19(b) offline steps are internal-consistency checks only.** They terminate at a block
  hash recomputed from a header the bundle itself carries. Reporting them as `pass` rows is fine, as
  long as the *commitment* is not reported as externally confirmed (**F4**). Do not downgrade those
  step rows.
- **`skip` is not a failure.** An offline run reports "verified" without chain badges rather than
  "failed" (`whitepaper.typ:6779-6782`). Several findings here (F6, F7, F8) exist purely because the
  CLI treats an unreachable third party as a proof defect.

---

## 6. Conformance checklist

Derived from E.25 (`whitepaper.typ:6845-6879`), in execution order, with the finding that covers each
gap. Work top to bottom; earlier items block later ones from even being reachable.

- [ ] **1. Accept JSON and CBOR (tagged and bare); apply the CBOR-to-JSON mapping; reject
      undecodable input.** (E.3; C.2, D.3)
      Gaps: **F13** (bare map rejected), **F11** (no `s.d` value-space rejections).
- [ ] **2. Apply the E.6 hard rejections before producing any step result.** (E.6)
      Gaps: **F1** (rejects conditions E.6 does not list), **F14** (`ep` not rejected),
      **F15** (`v` rejected though E.6 exempts it), **F16** (ten unauthorized rejections),
      **F17** (no E.23 vocabulary), **F21** (JSON and CBOR disagree on one row).
- [ ] **3. Canonicalize `s.d` exactly as parsed, with no null-dropping and no value coercion.**
      (E.4; C.2a)
      Gaps: **F3** (integers silently rounded), **F11** (keys dropped, values coerced),
      **F20** (`convert proof` rounds), **F22** (comparisons not constant-time).
      Conformant already: `null` preserved, UTF-16 key ordering, floats.
- [ ] **4. Derive `key_id = SHA-256(0x51 || pk)[0..3]` from a 32-byte `pk` and use only that
      derived id downstream.** (E.9; Appendix C)
      Conformant. Note **F16** turns the wrong-length `pk` case into a parse abort instead of the
      E.9 step fail.
- [ ] **5. Derive the subject hash with the 111-byte (`0x13`) or 121-byte (`0x23`) preimage.**
      (E.10; C, D.3)
      Conformant. See **F26** for the category of its missing-inputs failure.
- [ ] **6. Decode the compact proof with the little-endian bit-index formula and the exact size
      equation; reject depth over 64.** (E.12; C.7)
      Conformant. No gap.
- [ ] **7. Walk the inclusion proof to `b.mr`; report block-like subjects as skipped.** (E.13; C.7)
      Gap: **F19** (the required skip row is omitted).
- [ ] **8. Derive the 157-byte block hash; assign `subject_hash := block_hash` for block-like
      subjects.** (E.14; C.5, D.3)
      Gap: **F25** (`b.id` presence not enforced, so the preimage can be 121 bytes and still pass).
- [ ] **9. Walk each epoch proof to its chain-specific root, dispatching on `cx[].t`.** (E.15; D.3)
      Conformant. No gap.
- [ ] **10. Rebuild the 81 + 32N byte payload with the big-endian `t`, hash under `0x61`, and verify
      pure Ed25519 over the digest; assert `N == length(cx)`.** (E.16; D.3)
      Conformant, including the `N == length(cx)` assertion. No gap.
- [ ] **11. Cross-check `pk` against a pinned keyring, or report the check as skipped, under the Key
      Binding group.** (E.17; E.22, D.4)
      Gaps: **F8a** (transport failure fails the proof), **F8b** (no Key Binding group),
      **F8c** (positive-assertion detail on a failure), **F12** (the skip is invisible in `--json`).
- [ ] **12. Confirm the Stellar memo and, for Bitcoin, obtain one external confirmation point before
      reporting `pass`; otherwise `skip`.** (E.18, E.19; D.4)
      Gaps: **F2** (absence branches unimplemented), **F4** (`externally_verified: true` with no
      lookup), **F6** (Stellar transport failure and absent `net` fail), **F10** (E.19(b) step 4
      missing), **F24** (OP_RETURN output selection).
- [ ] **13. Determine the submission window without asserting the submitted-after edge from a bundle
      that does not carry it.** (E.20)
      Conformant. No gap found.
- [ ] **14. Re-fetch entropy sources, reporting existence and never freshness.** (E.21)
      Gap: **F7** (unavailability reported as `fail`).
- [ ] **15. Report per E.22: five statuses, fixed category order, warnings and skipped external
      checks never fail a proof.** (E.22)
      Gaps: **F9** (`--hash` on non-item fails, and the verdict surfaces contradict),
      **F12** (`--json` cannot express a skip), **F18** (failing steps carry passing messages),
      **F19** (required inapplicability skip missing), **F23** (E.11 charset check missing),
      **F26** (one category row, conditional).
- [ ] **16. Read `t` from the bundle, never from the filename.** (E.24)
      Gap: **F5**.
- [ ] **17. Run against the Appendix D bundle: reproduce every intermediate value in D.3 and produce
      a report whose statuses match D.4 under one-way containment.** (E.25)
      Currently impossible: blocked by **F1** (parse rejection) and then **F2** (three fail rows on
      a row D.4 reports as `skip`). **This is the single acceptance test for the whole effort.**
      When `truestamp verify /Users/glenn/src/github.com/truestamp/truestamp-v2/whitepaper/vectors/bundle.json`
      produces a report matching D.4 with exit 0, the critical findings are closed.

---

## 7. Appendix: what is actually unportable, measured

F3 is the subtlest finding here, and it is easy to overcorrect on. This section records a direct
cross-implementation measurement so you know exactly what to change and, just as importantly, what
to leave alone.

Method: a 28-case battery, deliberately clustered at the notation-switch boundaries, canonicalized
by Elixir `Jcs.encode/1` (OTP 29, Elixir 1.20.1) and by `truestamp jcs` (gowebpki/jcs v1.0.1), and
compared byte for byte.

### Floats are portable. Do NOT add a float guard.

All 25 float cases agreed exactly, including every case most likely to break:

| input | both sides emit |
| --- | --- |
| `1.0e20` | `100000000000000000000` |
| `1.0e21` | `1e+21` |
| `1.0e15`, `1.0e16`, `9.007199254740994e15` | positional, identical |
| `-1.0e22` | `-1e+22` |
| `-0.0` | `0` |
| `5.0e-324` (denormal) | `5e-324` |
| `1.0e-6` / `1.0e-7` | `0.000001` / `1e-7` |

The `1.0e15` and `1.0e16` rows are the interesting ones. OTP 25's `float_to_binary(F, [short])`
uses Ryū and **forces scientific notation outside (-2^53, 2^53)**, so Elixir's canonicalizer
receives `1.0e16` and must convert it back to positional under ECMA-262 7.1.12.1
(`deps/jcs/lib/jcs.ex:378-389`). It does, and the result matches Go. The `jcs` package moduledoc
carries a caveat that `float_to_binary` "seems to have differing results depending on the OTP
release"; that did not manifest. Ryū yields the unique shortest round-tripping digit sequence and
ECMA-262 determines the notation, so the output is fully determined.

So: scope your fix to integers only, exactly as F3's fix text says (`json.Number` with no fraction
and no exponent). A guard that also inspects floats would reject valid, portable bundles.

### The threshold, and why ours is one value stricter than yours should be

RFC 8785 section 3.2.2.3 sets no integer bound; it defers entirely to ECMA-262 7.1.12.1. The bound
is in **Appendix B**, as a SHOULD:

> values to be interpreted as true integers SHOULD be in the range
> -9007199254740991 to 9007199254740991

That is +/- (2^53 - 1), JavaScript's `Number.MAX_SAFE_INTEGER`. But the measured first divergence
is 2^53 + 1, because 2^53 itself is exactly representable and round-trips cleanly:

| input | Elixir | Go | agree |
| --- | --- | --- | --- |
| `9007199254740991` (2^53 - 1) | `9007199254740991` | `9007199254740991` | yes |
| `9007199254740992` (2^53) | `9007199254740992` | `9007199254740992` | yes |
| `9007199254740993` (2^53 + 1) | `9007199254740993` | `9007199254740992` | **no** |
| `18446744073709551615` | `18446744073709551615` | `18446744073709552000` | **no** |
| `-9007199254740993` | `-9007199254740993` | `-9007199254740992` | **no** |

Truestamp is adding a producer-side guard that REJECTS a new Item whose claims carry
`|n| > 2^53 - 1`, following Appendix B's SHOULD. That is deliberately one value stricter than the
verifier-side threshold, which flags only `|n| > 2^53`, so a verifier never raises a false alarm on
a bundle that every conforming implementation can actually check. Be strict in what you emit,
lenient in what you accept.

**For the CLI, use the verifier threshold: warn only when `|n| > 2^53`.** Reporting a bundle
carrying exactly 2^53 as unverifiable would be a false positive, since you and we both reproduce it.

### Reproducing the battery

```bash
printf '{"v":9007199254740993}' > /tmp/a.json && truestamp jcs /tmp/a.json
printf '{"v":1.0e16}'           > /tmp/b.json && truestamp jcs /tmp/b.json
printf '{"v":-0.0}'             > /tmp/c.json && truestamp jcs /tmp/c.json
```

Against the Elixir side, from a truestamp-v2 checkout:

```bash
elixir -pa _build/dev/lib/jcs/ebin \
  -e 'IO.puts(Jcs.encode(JSON.decode!(File.read!("/tmp/a.json"))))'
```

---

## 8. Outcome, and what is still open

Written 2026-07-25, after the remediation passes. This section describes the CURRENT tree; sections
1-7 describe the tree as it was when the audit was written.

### What was done

All 26 findings (F1-F26) were independently re-verified against Appendix E and the reference
verifier before implementation. **None was refuted; 17 needed corrections**, several of which would
have caused damage if implemented as written:

- The audit's single acceptance test could not pass as specified. D.4's run supplies the file hash,
  so the correct command needs `--hash b47cc0f1…4380`; without it E.7 requires a `warn` where D.4
  records a `pass`.
- F3's prescription (downgrade Inclusion Proof and Proof Signature to `skip` on an oversized
  integer) contradicts the reference verifier, which warns and passes both — and would itself break
  E.25 containment. The threshold is `|n| > 2^53`, not `2^53 - 1` (see section 7).
- F22's prescribed rewrite of `HexEqual` silently flips `HexEqual("zz","zz")` from true to false.
  Measured, not theorised, and both operands are bundle-controlled.
- F1's fix text keeps a rejection E.6 does not authorize (a present-but-malformed `memo`/`op`),
  contradicting F16's own principle. The root-key gate is now presence-only; a malformed root key
  falls through to an E.15 step failure.

An adversarial judging pass over the result found **76 further confirmed defects**, which were then
remediated. Most were pre-existing rather than caused by the conformance work: the pre-effort binary
(commit `83ebaadc`) also accepts CBOR `t=65546` and `v=1.9` at exit 0.

### Current state, measured

- **E.25 acceptance passes.** `truestamp verify <appendix-D vector> --skip-external --hash <s.d.hash>`
  exits 0 with one-way containment against all 14 D.4 rows, on JSON and on both CBOR forms
  (tagged and bare), byte-identical down to message text.
- **Tamper detection is not weakened**: 15 classes x 3 encodings, all exit non-zero; the reference
  verifier agrees on all 15.
- `task precommit` (fmt, vet, staticcheck, gosec, govulncheck, tests) passes.

### Still open

1. **`cx[].rtx` / `cx[].txp` accept uppercase hex.** Deliberate, pinned by
   `TestCLI_RawTxAndTxOutProof_CaseIsNotGraded`. Two reasons: neither field is trusted as a value
   (both are decoded, and every value derived from them is compared against something the signature
   chain pins, so a case flip changes nothing derived), and E.3 files them as text fields carrying
   "either base64url or hex" — base64url is case-significant, so a lowercase rule would reject
   bundles the wire format allows. Revisit jointly with truestamp-v2 if E.4 is decided to cover them.
2. **Reference-verifier defect, filed 2026-07-27, awaiting a ruling.** `whitepaper/verify_proof.exs`'s
   `unhex/1` (`Base.decode16!(s, case: :lower)`) raises an uncaught `ArgumentError` on uppercase hex
   in any of eight fields, instead of a graded result or a named E.23 rejection. Both implementations
   *refuse* such bundles — that disagreement is purely the form. Written up at
   `truestamp-v2/bug_hex.md` for that repo's maintainers, together with two boundary questions the
   same measurement pass surfaced: `cx[].tx` and `cx[].bmr` are a live **outcome** divergence (this
   CLI grades them `fail`, the reference exits 0 and never reads them), and `cx[].rtx` / `cx[].txp`
   are the deliberate exclusion in item 1 above, put to them for confirmation.
3. **Endpoint constants are not test-covered.** `cmd/external_e2e_test.go` exercises E.18/E.19/E.21
   availability grading through the shipped pipeline by substituting five base-URL constants at link
   time, so a typo in a real Horizon / Blockstream / NIST hostname would not be caught. A live-network
   test would cover it at the cost of making `go test` internet-dependent.
4. ~~**No committed entropy fixture.**~~ **CLOSED.** `internal/verify/testdata/fixtures/`
   now carries a signed bundle for each of `t ∈ {30,31,32}`, and the Entropy Source step is
   exercised end to end with the signature check enabled.

   The blocker recorded here, that a signed fixture requires a private key, was wrong. Appendix D's
   own vector is signed with a **fixed, published illustrative** Ed25519 seed
   (`whitepaper/proof_vectors.exs`), not a production key, so an independent Elixir generator can
   sign entropy bundles the same way. `testdata/fixtures/gen_entropy_fixtures.exs` does exactly
   that, and the fixtures are derived by Elixir and verified by Go, so a passing fixture cannot
   merely encode a Go bug. E.17 remains unestablished for them, correctly: the illustrative key is
   not in Truestamp's keyring, which
   `TestEntropyFixture_IllustrativeKeyIsNotBound` pins against a stub keyring that answers.
5. **CLAUDE.md documentation drift beyond this effort's scope.** The Global Flags table and pipeline
   recipes still describe `--api-url` and `--keyring-url`, which cobra now rejects outright; line 131
   of the same file contradicts them. Wants a `/docs-audit` pass.
6. **E.25 has a blind spot the appendix cannot close.** A report that drops a D.4 row and appends a
   fabricated one carrying the same (group, category, status) still satisfies containment, because
   the rule grades statuses and not messages. Covered for this implementation by the golden report
   at `internal/verify/testdata/golden/appendix-d4-report.txt`; it remains a gap in the appendix.
