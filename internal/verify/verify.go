// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package verify holds what `truestamp verify --remote` and `truestamp
// inspect` read through the earlier bundle parser, internal/proof:
// [RunRemoteBytesCtx] asks the Truestamp service to verify a proof and
// gives its verdict, which `verify --remote` prints beside the report and
// never as a row of it; and [BuildJSONRejection] and [PresentRejection]
// write that parser's rejection for `inspect`.
//
// This CLI's own verifier, the verification of SPEC.md section 10.2 that
// `truestamp verify` runs, is internal/verifier; nothing here takes part
// in it.
package verify
