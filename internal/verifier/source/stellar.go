// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the reads CHAIN makes of a Horizon server for a Stellar
// commitment entry (SPEC.md section 10.4, "Stellar commitment entries",
// and its informative table, "Common sources"), each giving its answer in
// the form of a replay record (SPEC.md section 13.2, "Replay files"):
//
//	| Read                    | Call                          | Fields of the answer's value                          |
//	|-------------------------|-------------------------------|-------------------------------------------------------|
//	| stellar_root            | GET <base>/                   | latest (history_latest_ledger), elder (history_elder_ledger) |
//	| stellar_transaction     | GET <base>/transactions/<hash>| ledger, memo_type, successful, memo                   |
//	| stellar_ledger          | GET <base>/ledgers/<sequence> | closed_at, hash                                       |
//
// A read's answer is one of SPEC.md section 10.4's three outcomes
// ("Answers"):
//
//   - found: a 200 OK whose body is a JSON object; the value holds each
//     field of the table that the object holds, as it holds it, and the
//     memo, which Horizon writes as the standard base64 of its 32 bytes,
//     with padding, as the lowercase hex of those bytes, present only when
//     memo_type is hash or return and the memo is the base64 of 32 bytes.
//     A field the object lacks is absent from the value, and so is a memo
//     in another form. This file judges no field's class: the rule that
//     reads the answer judges each field it uses at that point, so an
//     answer lacking a field, or giving one in another form, is unusable
//     only where its rule uses that field (SPEC.md section 10.4, "Answers");
//   - not found: a 404 to a read of a transaction or a ledger, the answer
//     Horizon gives for a hash or a sequence it holds no record of. The
//     root is never not found (SPEC.md section 13.2), so a 404 to it is
//     unreachable;
//   - unreachable: every other answer, a 3xx, a 429, a 503, a challenge
//     page and a body over its bound among them, and a read whose hash is
//     not a HEX64 or whose sequence is not an INT, which is made to no
//     server. Its reason is a fixed text, as the identity reads' are
//     ([fromRead]), quoting nothing a server sent.
//
// The source's URL is the operator's or a compiled-in default, never the
// document's: a read's parameters name only the record asked for.

package source

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// Outcome is a chain read's outcome (SPEC.md section 10.4, "Answers").
type Outcome string

// The outcomes of a chain read.
const (
	// OutcomeFound: the source returned the object asked for.
	OutcomeFound Outcome = "found"
	// OutcomeNotFound: the source answered that it holds no such object.
	OutcomeNotFound Outcome = "not_found"
	// OutcomeUnreachable: there is no usable answer.
	OutcomeUnreachable Outcome = "unreachable"
)

// Answer is one chain read's answer, in the form of a replay record
// (SPEC.md section 13.2): Outcome; Value, for a found answer, the fields
// the read gives, each a value internal/jcs can write; and Reason, for an
// unreachable one, a fixed text that quotes nothing a server sent.
type Answer struct {
	Outcome Outcome
	Value   map[string]any
	Reason  string
}

// The kinds of chain read this file makes (SPEC.md section 13.2's table).
const (
	KindStellarRoot        = "stellar_root"
	KindStellarTransaction = "stellar_transaction"
	KindStellarLedger      = "stellar_ledger"
)

// The bounds of a Horizon answer's body: its root and a ledger are each a
// few kilobytes, and a transaction carries its envelope and result, whose
// XDR can run to some tens of kilobytes.
const (
	RootOctets        = 65536
	LedgerOctets      = 65536
	TransactionOctets = 1 << 20
)

// Read makes the chain read named read, {kind, network, ...} as a replay
// record names it, of the source of provider at base, through reads, and
// gives its answer. A kind this provider does not serve is unreachable,
// with no request made.
func Read(ctx context.Context, reads *httpclient.SingleRead, provider Provider, base string, read map[string]any) Answer {
	kind, _ := read["kind"].(string)
	if provider == Horizon {
		switch kind {
		case KindStellarRoot:
			return StellarRoot(ctx, reads, base)
		case KindStellarTransaction:
			return StellarTransaction(ctx, reads, base, read["hash"])
		case KindStellarLedger:
			return StellarLedger(ctx, reads, base, read["sequence"])
		}
	}
	return unreachableAnswer("no read of this kind from this provider")
}

// StellarRoot reads a Horizon server's root, GET <base>/, for
// history_latest_ledger and history_elder_ledger, given as latest and
// elder.
func StellarRoot(ctx context.Context, reads *httpclient.SingleRead, base string) Answer {
	object, answer, ok := getObject(ctx, reads, endpoint(base, "/"), RootOctets, false)
	if !ok {
		return answer
	}
	return found(object, map[string]string{"history_latest_ledger": "latest", "history_elder_ledger": "elder"})
}

// StellarTransaction reads the transaction whose hash is hash, GET
// <base>/transactions/<hash>, for successful, memo_type, memo and ledger.
func StellarTransaction(ctx context.Context, reads *httpclient.SingleRead, base string, hash any) Answer {
	text, isString := hash.(string)
	if !isString || !hex64(text) {
		return unreachableAnswer("the transaction hash is not 64 lowercase hex digits")
	}
	object, answer, ok := getObject(ctx, reads, endpoint(base, "/transactions/"+text), TransactionOctets, true)
	if !ok {
		return answer
	}
	answer = found(object, map[string]string{"successful": "successful", "memo_type": "memo_type", "ledger": "ledger"})
	if memoType, _ := object["memo_type"].(string); memoType == "hash" || memoType == "return" {
		if memo, isMemo := memoBytes(object["memo"]); isMemo {
			answer.Value["memo"] = hex.EncodeToString(memo)
		}
	}
	return answer
}

// StellarLedger reads the ledger whose sequence is sequence, GET
// <base>/ledgers/<sequence>, for closed_at and hash.
func StellarLedger(ctx context.Context, reads *httpclient.SingleRead, base string, sequence any) Answer {
	number, isNumber := sequence.(strictjson.Number)
	n, isInt := number.Int64()
	if !isNumber || !isInt || n < 0 {
		return unreachableAnswer("the ledger sequence is not an integer from 0 to 2^53 - 1")
	}
	object, answer, ok := getObject(ctx, reads, endpoint(base, "/ledgers/"+strconv.FormatInt(n, 10)), LedgerOctets, true)
	if !ok {
		return answer
	}
	return found(object, map[string]string{"closed_at": "closed_at", "hash": "hash"})
}

// getObject reads url and gives the JSON object a 200 OK holds, or the
// answer it stands for instead: not found for a 404 when notFound is true,
// and otherwise unreachable.
func getObject(ctx context.Context, reads *httpclient.SingleRead, url string, limit int, notFound bool) (map[string]any, Answer, bool) {
	body, err := reads.Get(ctx, url, limit)
	if err != nil {
		var status *httpclient.StatusError
		if notFound && errors.As(err, &status) && status.StatusCode == http.StatusNotFound {
			return nil, Answer{Outcome: OutcomeNotFound}, false
		}
		var unreachable *Unreachable
		errors.As(fromRead(err), &unreachable)
		return nil, unreachableAnswer(unreachable.Reason), false
	}
	value, err := strictjson.Parse(body)
	object, isObject := value.(map[string]any)
	if err != nil || !isObject {
		return nil, unreachableAnswer("Horizon's answer is not a JSON object"), false
	}
	return object, Answer{}, true
}

// found is the found answer whose value holds each field of object that
// names gives a field of the value's name, as object holds it.
func found(object map[string]any, names map[string]string) Answer {
	value := map[string]any{}
	for horizon, field := range names {
		if v, present := object[horizon]; present {
			value[field] = v
		}
	}
	return Answer{Outcome: OutcomeFound, Value: value}
}

// memoPattern is the standard base64 of 32 bytes, with its padding.
var memoPattern = regexp.MustCompile(`^[A-Za-z0-9+/]{42}[AEIMQUYcgkosw048]=$`)

// memoBytes gives the 32 bytes of a memo Horizon writes as their standard
// base64, with padding, and whether memo is one. The pattern refuses what
// the decoder would skip, such as a line break, and a final symbol whose
// unused bits are set, so one memo has one spelling.
func memoBytes(memo any) ([]byte, bool) {
	text, isString := memo.(string)
	if !isString || !memoPattern.MatchString(text) {
		return nil, false
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(text)
	return decoded, err == nil && len(decoded) == 32
}

// unreachableAnswer is an unreachable answer with reason.
func unreachableAnswer(reason string) Answer {
	return Answer{Outcome: OutcomeUnreachable, Reason: reason}
}
