// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file reads a raw_transaction (SPEC.md section 3.4), for the Bitcoin
// evidence of DERIVE (section 10.2, step 9 (d)). The bytes are well formed
// when these rules hold, checked in this order, the first that fails
// refusing them:
//
//	| Rule      | Holds when                                                                                                     |
//	|-----------|----------------------------------------------------------------------------------------------------------------|
//	| parse     | they parse completely as a transaction serialized without witness data (BIP 144): the version, at least one input, the outputs and the lock time, with every compact size in its shortest form and no byte after the lock time |
//	| length    | they are not exactly 64 octets long, so they cannot pass for an inner node of a Bitcoin Merkle tree (CVE-2017-12842) |
//	| op_return | exactly one of the output scripts begins with 0x6a, and that script is exactly 0x6a 0x20 followed by 32 bytes |
//
// Those 32 bytes are the chain's copy of the epoch hash (section 5.4),
// which DERIVE compares with the epoch_hash it recomputed. A transaction
// serialized with its witness has the marker octet 0x00 where the count of
// inputs stands, so it reads as a transaction with no input and fails
// parse: its bytes are judged as carried, and the witness is never
// stripped.
//
// [TransactionHash] is SHA-256d of the bytes as carried, whatever they
// hold: transaction_hash is rev of it (section 3.4), and the hash a
// txoutproof matches is compared with it.

package bitcoin

// The octets of the fixed fields of a transaction serialized without
// witness data.
const (
	versionOctets  = 4
	outpointOctets = 36 // the previous output's 32-octet hash and 4-octet index
	sequenceOctets = 4
	valueOctets    = 8
	lockTimeOctets = 4
)

// innerNodeOctets is the length of an inner node of a Bitcoin Merkle tree,
// two 32-octet hashes, which a raw_transaction must not have.
const innerNodeOctets = 64

// opReturn is the opcode an OP_RETURN output script begins with, and
// push32 the opcode that pushes the next 32 bytes.
const (
	opReturn = 0x6a
	push32   = 0x20
)

// TransactionHash is SHA-256d(raw), in the order it is hashed in, of the
// bytes as carried; rev of it is transaction_hash (SPEC.md section 3.4).
func TransactionHash(raw []byte) [32]byte {
	return sha256d(raw)
}

// ParseTransaction judges raw as a raw_transaction (see the file's doc):
// the 32 bytes its one OP_RETURN script pushes, in the order pushed, when
// it is well formed, and otherwise the first rule it breaks, a [Refusal].
func ParseTransaction(raw []byte) ([32]byte, error) {
	scripts, ok := outputScripts(raw)
	if !ok {
		return [32]byte{}, RefusedParse
	}
	if len(raw) == innerNodeOctets {
		return [32]byte{}, RefusedLength
	}
	var found [][]byte
	for _, script := range scripts {
		if len(script) > 0 && script[0] == opReturn {
			found = append(found, script)
		}
	}
	if len(found) != 1 || len(found[0]) != 2+32 || found[0][1] != push32 {
		return [32]byte{}, RefusedOpReturn
	}
	return [32]byte(found[0][2:]), nil
}

// outputScripts parses raw completely as a transaction serialized without
// witness data: the output scripts, in order, or ok false.
func outputScripts(raw []byte) (scripts [][]byte, ok bool) {
	_, rest, ok := take(raw, versionOctets)
	if !ok {
		return nil, false
	}
	inputs, rest, ok := compactSize(rest)
	if !ok || inputs < 1 {
		return nil, false
	}
	for range inputs {
		if _, rest, ok = take(rest, outpointOctets); !ok {
			return nil, false
		}
		var length uint64
		if length, rest, ok = compactSize(rest); !ok {
			return nil, false
		}
		if _, rest, ok = take(rest, length); !ok {
			return nil, false
		}
		if _, rest, ok = take(rest, sequenceOctets); !ok {
			return nil, false
		}
	}
	outputs, rest, ok := compactSize(rest)
	if !ok {
		return nil, false
	}
	for range outputs {
		if _, rest, ok = take(rest, valueOctets); !ok {
			return nil, false
		}
		var length uint64
		if length, rest, ok = compactSize(rest); !ok {
			return nil, false
		}
		var script []byte
		if script, rest, ok = take(rest, length); !ok {
			return nil, false
		}
		scripts = append(scripts, script)
	}
	return scripts, len(rest) == lockTimeOctets
}
