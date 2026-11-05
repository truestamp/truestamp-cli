// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package specdata holds what a verifier pins (SPEC.md section 9): every
// file under the spec repository's spec/, the 12 files and the constructions
// texts under constructions/, embedded byte for byte from verification/spec/
// beside this file, with verification/SOURCE naming the spec commit they
// came from.
//
// task verification-sync writes verification/ here and testdata/verification/
// at the module root from one commit on the spec's main, and nothing else may
// change either copy; task verification-check compares both with that commit.
// The tests hold testdata/verification/ to its MANIFEST.sha256, the embedded
// files and verification/ to that copy, and each table this package holds
// to the file it is transcribed from (SPEC.md section 13.1, "Beyond the
// vectors").
//
// A file comes back as its bytes, from [File]: nothing here decodes JSON, so
// a file is read under the parsing rules of SPEC.md section 10.1 by the
// verifier's own strict reader. What a verifier needs as constants is
// written here as Go values, transcribed from its file and held equal to it
// by a test:
//
//   - the constants of constants.json, as Go constants, and [Constants];
//   - SMALL_ORDER_KEYS, [SmallOrderKeys];
//   - the pattern classes of encodings.json, [Encodings];
//   - the network names of networks.json, [Networks];
//   - the default network pins of network-pins.json, [DefaultNetworkPins]
//     and [Environments];
//   - the reason codes of reason-codes.json, with their sub_reasons,
//     [ReasonCodes];
//   - the exit codes of exit-codes.json, [ExitCodes] and [ExitCode];
//   - CONSTRUCTIONS_ACCEPTED, the digests of the texts under constructions/,
//     computed from the embedded bytes, [Accepted].
//
// The contents of keyring.production.json, the compiled keyring pin of
// SPEC.md section 8.3, are read only through [File]: it holds Truestamp's
// production key, and no table of this package transcribes it.
package specdata
