// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package verifier is the CLI's verifier of SPEC.md section 10 (The
// verification algorithm), built beside internal/verify from SPEC.md and
// its conformance vectors (section 13.2 (The vectors)), each file naming
// the sections of SPEC.md it implements.
//
// It holds steps 0 to 6 of SPEC.md section 10.2 (Steps), the gate. Steps 0
// to 3 decide whether a run gives a report at all: CONFIGURATION, which
// fixes E_v, the network pins and the keyring that governs key binding
// before the file is read; READ (J1); PARSE (J2 to J6); and ENVELOPE, the
// envelope of section 3.1. A failure at any of them is a bare rejection, a
// [*Rejection]; a run that passes them is a [*Run] with one pass row of
// each step, part null. Steps 4 to 6 add one row each: ENVIRONMENT, the
// marker of section 6.4 against E_v; KEY_BINDING, K against the keyring
// that governs, all 32 octets compared (section 6.1); and SIGNATURE, V
// under K over proof_hash by the strict rule of section 6.2. A failure at
// any of them ends the run with the attribution none; a run that passes
// them carries its attribution and trust_root (section 10.5). Past the
// gate it holds step 7, SHAPE: format, version, the schema of
// spec/schema.json, through the package's own interpreter of SPEC.md
// section 9's keyword subset, and the three checks the schema cannot make,
// the witnesses of an item, the TIME and NISTTIME round trips and the
// constructions digest; a failure ends the run, and a pass gives the run
// its type and subject id. Then step 8, POLICY: every network member of the
// proof against the run's network pins, one row each; a failure there
// does not end the run. Then step 9, DERIVE: every record hash of the
// subject, the block nodes and the epochs recomputed and compared, each key
// event a block node carries checked as the table of section 3.3 requires,
// every inclusion proof walked from its recomputed leaf, its tree_size held
// to its record's committed leaf_count, and the other links of section
// 10.3, LINK_WITNESS, LINK_HEAD, LINK_KEY_EVENT and LINK_CHAIN_ORDER,
// through one engine that gives the step's rows and the run's trace
// (SPEC.md section 10.6); a failure there does not end the run either, and
// LINK_KEY_EVENT's info row under an attribution other than truestamp is
// no failure. When DERIVE ends, the run takes its external gate, open only
// when the caller did not ask for an offline run and no row has failed.
// Then step 10, CHAIN, and step 11, SOURCES: under a closed gate, each
// commitment entry, genesis.bitcoin and entropy object gets its row skip
// offline or skip prior_failure, and KEY_EVENT_COMMITMENT ends CHAIN.
// Under an open gate CHAIN reads each Stellar commitment entry by the rule
// of section 10.4, through source_outcomes.go's frame, made once for each
// source of its network and combined under the network's source policy,
// and keeps each passing row's candidate for WINDOW; the Bitcoin rule and
// the sources' rules are not in the package yet, so their rows, and
// WINDOW's, are not given under an open gate. Then step 12, WINDOW, which
// a closed gate leaves not established; step
// 13, EXPECTED_HASH; and step 14, VERDICT, the label and the run's code of
// section 10.5.
//
//   - config.go: a run's [Config], what step 0 fixes (section 10.2);
//   - options.go: the options of `truestamp verify` that set a Config,
//     which only the run's own options may set (sections 6.4 and 8.3);
//   - gate.go: the steps (section 10.2, the gate);
//   - keyring_read.go: the online keyring read of step 0, of the keyring
//     that governs an external verifier's online run by default (section
//     8.3);
//   - sources.go: the rest of step 0 for the sources: each network's
//     source policy, held to its number of sources in every run, and in
//     an online run which Stellar and Bitcoin sources serve each network
//     pin, in what order, and each source's identity, confirmed both ways
//     before the file is read (sections 9 and 10.2, step 0), through the
//     readers of internal/verifier/source;
//   - shape.go: step 7, SHAPE, whose check 5 parses each time through
//     time.go, the classes TIME and NISTTIME (section 2.4);
//   - policy.go: step 8, POLICY;
//   - derive.go: step 9, DERIVE, and its one engine (section 10.6), whose
//     checks are derive_check.go's: the record hashes of derive_records.go
//     (step 9 (a) to (c)), the key events of derive_key_events.go and the
//     links of derive_links.go (section 10.3);
//   - external.go: the external gate, and steps 10 and 11, CHAIN and
//     SOURCES, under a closed gate (sections 10.2, 10.3 and 10.4);
//   - chain.go: CHAIN under an open gate, KEY_EVENT_COMMITMENT's statuses
//     from its inputs, and the [ChainReader] its reads go through
//     (sections 10.3 and 10.4); chain_stellar.go: the rule of a Stellar
//     commitment entry (sections 10.4 and 11.1);
//   - source_outcomes.go: the frame a CHAIN or SOURCES row read under an
//     open gate runs through: each source's outcome, their combination
//     under the source policy, and the row's values (section 10.4);
//   - window.go: step 12, WINDOW, under a closed gate (section 11.3);
//   - expected_hash.go: step 13, EXPECTED_HASH;
//   - verdict.go: step 14, VERDICT's label and the run's code (section
//     10.5);
//   - schema.go: the checker of spec/schema.json, SHAPE's check 3, which
//     refuses a schema outside SPEC.md section 9's subset when it loads;
//   - layout.go: where the proof holds its block nodes, commitment
//     entries, entropy objects and genesis.bitcoin objects, and the value
//     at a pointer into it (sections 3.3 to 3.6);
//   - run.go, row.go, rejection.go and pointer.go: the run's state, its
//     rows, the bare rejection and JSON Pointers into the proof (sections
//     10.2 and 10.5).
//
// The keyring, its load checks and its digest are internal/verifier/keyring,
// and the checks of a key's bytes and strict Ed25519 are
// internal/verifier/ed25519.
//
// An online run's sources are the compiled-in defaults of its pinned
// networks (none for regtest), the Horizon, Esplora and Bitcoin Core
// servers the operator pins, and the several sources the operator
// configures for a network, in an order, under a source policy (SPEC.md
// section 9); the document never chooses one, nor a policy. Each source's
// identity read goes through [Config.ReadIdentity], and one with no usable
// answer is recorded unreachable, so nothing else is read from it; every
// other read of a source goes through [Config.ReadChain].
//
// The CLI is an external verifier (SPEC.md section 8.3). Online, under E_v
// prod and with no supplied keyring, the keyring read from
// www.truestamp.com governs; such a read, and the read of a keyring from
// the origin an explicit --base-url names, go through [Config.ReadKeyring],
// and a read that does not give a keyring passing the load checks binds no
// key, whatever the compiled pin holds. Offline, under prod, the compiled
// pin governs.
package verifier
