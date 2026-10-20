// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the constants of SPEC.md section 9 (What a verifier
// pins), each named after its constant, transcribed from
// spec/constants.json and held equal to it by TestAgreement_Constants.
// SMALL_ORDER_KEYS has a file of its own, small_order_keys.go.

package specdata

// The constants of SPEC.md section 9, each named after the name
// spec/constants.json gives it. A test holds each to the file and to
// SPEC.md's table.
const (
	// BitcoinFinality is BITCOIN_FINALITY.
	BitcoinFinality = 10
	// BitcoinMaxTotalTxs is BITCOIN_MAX_TOTAL_TXS.
	BitcoinMaxTotalTxs = 16666
	// BitcoinMedianSpan is BITCOIN_MEDIAN_SPAN.
	BitcoinMedianSpan = 11
	// ClaimsMaxOctets is CLAIMS_MAX_OCTETS.
	ClaimsMaxOctets = 20480
	// MaxDepth is MAX_DEPTH, counting the document object as 1.
	MaxDepth = 32
	// MaxDocumentOctets is MAX_DOCUMENT_OCTETS.
	MaxDocumentOctets = 65536
	// MaxInteger is MAX_INTEGER, 2^53 - 1.
	MaxInteger = 9007199254740991
	// MaxPathSteps is MAX_PATH_STEPS.
	MaxPathSteps = 32
	// MaxReadOctets is MAX_READ_OCTETS.
	MaxReadOctets = 262144
	// RawTxMaxOctets is RAW_TX_MAX_OCTETS.
	RawTxMaxOctets = 1000
	// StellarIndexMargin is STELLAR_INDEX_MARGIN, in ledgers.
	StellarIndexMargin = 60
	// TxoutproofMaxOctets is TXOUTPROOF_MAX_OCTETS.
	TxoutproofMaxOctets = 1024
	// VectorKey is VECTOR_KEY, in hex: the public key of the seed of 32
	// bytes of 0x42, valid only when the verifier's environment is test.
	VectorKey = "2152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12"
)

// Constants returns every constant by its name in spec/constants.json, an
// integer as an int64 and VECTOR_KEY as its hex string. Each call returns a
// new map.
func Constants() map[string]any {
	return map[string]any{
		"BITCOIN_FINALITY":      int64(BitcoinFinality),
		"BITCOIN_MAX_TOTAL_TXS": int64(BitcoinMaxTotalTxs),
		"BITCOIN_MEDIAN_SPAN":   int64(BitcoinMedianSpan),
		"CLAIMS_MAX_OCTETS":     int64(ClaimsMaxOctets),
		"MAX_DEPTH":             int64(MaxDepth),
		"MAX_DOCUMENT_OCTETS":   int64(MaxDocumentOctets),
		"MAX_INTEGER":           int64(MaxInteger),
		"MAX_PATH_STEPS":        int64(MaxPathSteps),
		"MAX_READ_OCTETS":       int64(MaxReadOctets),
		"RAW_TX_MAX_OCTETS":     int64(RawTxMaxOctets),
		"STELLAR_INDEX_MARGIN":  int64(StellarIndexMargin),
		"TXOUTPROOF_MAX_OCTETS": int64(TxoutproofMaxOctets),
		"VECTOR_KEY":            VectorKey,
	}
}
