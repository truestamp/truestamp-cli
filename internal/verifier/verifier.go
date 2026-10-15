// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package verifier is the CLI's verifier of SPEC.md section 10 (The
// verification algorithm), built beside internal/verify and ported module
// by module from the reference verifier in truestamp-verification
// (elixir/lib/truestamp/verifier/), each file naming the module it ports
// and checked against the Truestamp application's port of it
// (lib/truestamp/proof/verifier/ in truestamp-v2).
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
// them carries its attribution and trust_root (section 10.5).
//
//   - config.go: a run's [Config] (the reference's config.ex);
//   - options.go: the options of `truestamp verify` that set a Config,
//     which only the run's own options may set (sections 6.4 and 8.3);
//   - gate.go: the steps (gate.ex);
//   - keyring_read.go: the online keyring read of step 0, which no
//     reference module ports (the reference is offline);
//   - sources.go: the rest of step 0 in an online run, which no reference
//     module ports either: which Stellar and Bitcoin source serves each
//     network pin, and each source's identity, confirmed both ways before
//     the file is read (sections 9 and 10.2, step 0), through the readers
//     of internal/verifier/source;
//   - run.go, row.go, rejection.go and pointer.go: the run's state, its
//     rows, the bare rejection and JSON Pointers into the proof (run.ex,
//     row.ex, rejection.ex and pointer.ex).
//
// The keyring, its load checks and its digest are internal/verifier/keyring,
// and the checks of a key's bytes and strict Ed25519 are
// internal/verifier/ed25519.
//
// An online run's sources are the compiled-in defaults of its pinned
// networks (none for regtest) and the Horizon, Esplora and Bitcoin Core
// servers the operator pins; the document never chooses one. Each source's
// identity read goes through [Config.ReadIdentity], and one with no usable
// answer is recorded unreachable, so nothing else is read from it.
//
// The CLI is an external verifier (SPEC.md section 8.3). Online, under E_v
// prod and with no supplied keyring, the keyring read from
// www.truestamp.com governs; such a read, and the read of a keyring from
// the origin an explicit --base-url names, go through [Config.ReadKeyring],
// and a read that does not give a keyring passing the load checks binds no
// key, whatever the compiled pin holds. Offline, under prod, the compiled
// pin governs.
package verifier
