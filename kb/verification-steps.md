<!--
Copyright (c) 2019-2026 Truestamp, Inc.
SPDX-License-Identifier: Apache-2.0
-->

# Verification steps

`truestamp verify` runs `internal/verifier`, the CLI's verifier of SPEC.md section 10 (The verification algorithm), built from SPEC.md and its conformance vectors. It makes the steps of SPEC.md section 10.2 in order, and each file names the sections it implements. Read SPEC.md section 10 before changing anything here: SPEC.md fixes which rows a run reports, their parts, their statuses, their codes and their order, and conformance compares every row (SPEC.md section 13.1). The entry point is `verifier.Verify` in [`outcome.go`](../internal/verifier/outcome.go); `cmd/verify.go` sets its configuration from the command's options and `cmd/verify_render.go` writes the report.

## The steps

| Step | Name | Where | What it checks |
| ---- | ---- | ----- | -------------- |
| 0 | CONFIGURATION | `config.go`, `options.go`, `sources.go`, `keyring_read.go`, `gate.go` | E_v, the network pins, the sources and the keyring that governs, before the file is read; online, each Stellar and Bitcoin source's identity read |
| 1 to 3 | READ, PARSE, ENVELOPE | `internal/strictjson`, `gate.go` | J1 (at most 262,144 octets), J2 to J6, and the envelope of SPEC.md section 3.1 with the bytes of K |
| 4 | ENVIRONMENT | `gate.go` | the environment marker against E_v |
| 5 | KEY_BINDING | `gate.go`, `internal/verifier/keyring` | K against the keyring that governs; keyless mode is a skip, `key_binding_not_performed` |
| 6 | SIGNATURE | `gate.go`, `internal/verifier/ed25519` | V under K over `proof_hash`, by the strict rule of SPEC.md section 6.2 |
| 7 | SHAPE | `shape.go`, `schema.go` | `format`, `version`, the schema `spec/schema.json`, the checks the schema cannot make, and `constructions` |
| 8 | POLICY | `policy.go` | every `network` against the pins |
| 9 | DERIVE | `derive*.go`, `internal/verifier/bitcoin` | every recomputation and every link, offline: the records' hashes, the inclusion walks, the Bitcoin evidence and the key events (SPEC.md section 10.3) |
| 10 | CHAIN | `chain*.go` | each commitment entry and `genesis.bitcoin`, read from its chain, then KEY_EVENT_COMMITMENT |
| 11 | SOURCES | `entropy.go` | each entropy object, read from its source |
| 12 | WINDOW | `window.go` | the submission window of SPEC.md section 11 |
| 13 | EXPECTED_HASH | `expected_hash.go` | `--expected-hash` against `claims.hash`, when given |
| 14 | VERDICT | `verdict.go` | the label, attribution and window; adds no row |

**The gate.** A failure at steps 0 to 3 is a bare rejection: `{code, message, sub_reason}`, with `sub_reason` only for `malformed_json` and `invalid_envelope`, and no report and no verdict (`rejection.go`). A failure at steps 4 to 6 ends the run: the label is `failed`, the attribution `none`, and each later step up to EXPECTED_HASH reports one row, `skip prior_failure`, part null. So nothing in the proof but its `environment` is interpreted until the key is bound and the signature verified. No option skips the signature.

**The external gate.** The gate is decided once, when DERIVE ends: CHAIN and SOURCES read only when the run is online and no row so far is `fail` (`external.go`); otherwise each of their rows is `skip prior_failure` or `skip offline`. Offline, every check that needs a read is a skip, never a failure, so an offline run is never `verified`. A source that gives no usable answer serves its rows as `skip source_unreachable`; one that answers and contradicts the proof fails its row. Several sources for one network are combined under the policy `first`, `at_least N` or `all` (SPEC.md section 10.4, `source_outcomes.go`).

## Rows and the report

A row is `{code, message, part, status, step, values}` (SPEC.md section 10.5, `row.go`): `code` a reason code of SPEC.md section 12.1, absent exactly on a pass; `part` an RFC 6901 pointer into the proof, or null; `status` one of `pass`, `fail`, `skip`, `warn`, `info`; and `values` what the run computed or read. Rows are in step order and, within a step, in the document order of their parts. A step's code is that of its first failing row, and the run's code that of its first failing step. The 36 reason codes, each with its step and status, are `truestamp schema get reason-codes`, generated from `spec/reason-codes.json` through `internal/specdata`. The CLI adds no row SPEC.md does not define and drops none it defines; a check the CLI needs that SPEC.md lacks is raised against SPEC.md first.

The report (`report.go`) has `attribution` (`truestamp`, `keyring:<d>`, `unattributed` or `none`), `code` (when `failed`), `environment`, `label`, `rows`, `subject_id`, `submitted_after`, `submitted_before`, `trust_root` (`{digest, kind, location}`), `type`, `verified` and `window`. `verify --json` writes exactly that, or the bare rejection. The label is the first that applies of `failed`, `unattributed`, `supplied-keyring`, `internally-consistent` and `verified`, and the process exits with SPEC.md section 12.2's code for it: 0 `verified`, 1 a runtime error outside verification, 2 a fault the verifier stopped on, 3 `failed`, 4 a bare rejection other than `configuration_error`, 5 `unattributed`, 6 `supplied-keyring`, 7 `internally-consistent`, 8 `configuration_error` (`internal/specdata`, held to `spec/exit-codes.json`; `truestamp schema get exit-codes`).

The edges come only from `submitted_after` and `submitted_before`, captioned "Submitted after" and "Submitted before", each naming the sources it was read from; no window is shown for a `failed` run, and every other time appears only as recorded or declared (SPEC.md section 11.5). For an item proof the text also prints the `item_hash` recomputed from the subject's record and, beside it, its item code (SPEC.md section 3.9), captioned as a name for the item and not a check. The item code is never a member or row of the report, never shown with a pass status, and `--json` does not carry it.

`--trace FILE` writes the run's trace (SPEC.md section 10.6): one row for each recomputed value, its declared value and what it was compared with.

## The key and its id

K's key id, the hex of the first 4 bytes of SHA-256(0x60 || K), is only an index into the keyring. A `key_id` or `signing_key_id` the document carries never chooses or checks the signer: DERIVE checks each one where SPEC.md section 3.3's key-event table names it, against its event's key, and a mismatch is `key_event_mismatch`. A block record's `signing_key_id` is a preimage input and recorded provenance only.

The keyring that governs a run is chosen by the run's options alone (SPEC.md section 8.3): `--no-keyring` (keyless), else `--keyring FILE`, else, online under `prod`, the keyring read from `https://www.truestamp.com/.well-known/keyring.json` at CONFIGURATION, and offline the compiled pin. A failed online read binds no key, and KEY_BINDING fails with `keyring_unavailable`; it never falls back to the compiled pin. See [configuration.md](configuration.md) for the whole order.

## Conformance

[`internal/vectorcases/conformance_test.go`](../internal/vectorcases/conformance_test.go) runs every case of `testdata/verification/vectors/cases.json`, the synced copy of the spec's vectors, through the one runner (`internal/vectorcases/runner_test.go`), each online case answered from its replay file, and compares what SPEC.md section 13.1 says: a bare rejection's `code` and `sub_reason`; a report's `label`, its `code` when the case gives one, its rows reduced to `(code, part, status, step)` in order, and, online, each edge's instant and basis. It also runs every case through `verifier.Verify`, the path `truestamp verify` runs, and requires the same encoded report or rejection byte for byte but for the clock-derived `fetched_at`. It logs the claim of conformance: the spec commit `testdata/verification/SOURCE` names and the SHA-256 of `testdata/verification/MANIFEST.sha256`. The reference verifier, `Truestamp.Verify` in truestamp-verify-ex, runs the same cases; a disagreement at the same spec commit is a defect in one of the two, and SPEC.md decides which.

## Remote mode

`verify --remote` also posts the file's bytes, exactly as read, to `POST {base-url}/api/json/proof/raw/verify` (with `offline=true` on an offline run, and the expected hash when one is given), and prints the label of the service's report on stderr beside this run's report (`internal/verify/remote.go`). The service's verdict is never a row, never a reason code, and changes neither the report, `--json` stdout nor the exit code, which are this run's own. The request waits for the larger of `--http-timeout` and 75 seconds. A refusal or an answer that is not a report is printed as the service giving no verdict.
