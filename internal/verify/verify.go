// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package verify holds what `truestamp verify --remote` and `truestamp
// inspect` use beside this CLI's own verifier: [AskRemoteCtx] posts a
// proof file's bytes to the Truestamp service and gives the label of the
// report it answers with, which `verify --remote` prints beside its own
// report and never as a row of it or a reason code; and
// [BuildJSONRejection] and [PresentRejection] write a rejection of the
// earlier bundle parser, internal/proof, for `inspect`.
//
// This CLI's own verifier, the verification of SPEC.md section 10.2 that
// `truestamp verify` runs, is internal/verifier; nothing here takes part
// in it.
package verify
