// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the reads CHAIN makes of an Esplora server or a Bitcoin
// Core node for a Bitcoin commitment entry and for genesis.bitcoin, and
// the read SOURCES makes for a Bitcoin entropy object (SPEC.md section
// 10.4, "Bitcoin commitment entries", "genesis.bitcoin", "Entropy objects"
// and its informative table, "Common sources"), each giving its answer in
// the form of a replay record (SPEC.md section 13.2, "Replay files"):
//
//	| Read                    | Calls                                                     | Fields of the answer's value                 |
//	|-------------------------|-----------------------------------------------------------|----------------------------------------------|
//	| bitcoin_tip             | Esplora GET <base>/blocks/tip/height                      | height                                       |
//	|                         | Bitcoin Core getblockcount                                |                                              |
//	| bitcoin_block_at_height | Esplora GET <base>/block-height/<height>, then            | hash, median_time_past (mediantime),         |
//	|                         | GET <base>/block/<hash>                                   | transaction_count (tx_count)                 |
//	|                         | Bitcoin Core getblockhash <height>, then                  | hash, median_time_past (mediantime),         |
//	|                         | getblockheader <hash>                                     | transaction_count (nTx)                      |
//	| bitcoin_block           | Esplora GET <base>/block/<hash>                           | height, median_time_past, time               |
//	|                         | Bitcoin Core getblockheader <hash>                        |                                              |
//
// The block at a height is the best-chain block there: its hash is the
// first call's answer, and its median time past and its transaction count
// are those the second call gives of the block with that hash, Esplora's
// mediantime and tx_count and Bitcoin Core's mediantime and nTx. The block
// by its hash is the block the source holds with that hash, on its best
// chain or not (section 10.4: the block of an entropy object need not be
// on the best chain, and Bitcoin Core's getblockheader answers for a block
// off it too), one call: its height, its header's nTime as time, and its
// median time past. Section 10.4 lets a verifier take a median time past
// from the source's report ("Median time past"), which is what these reads
// give.
//
// A read's answer is one of SPEC.md section 10.4's three outcomes
// ("Answers"):
//
//   - found: the tip's height, an INT (Esplora's text of decimal digits,
//     Bitcoin Core's result); or the block at a height, whose value holds
//     the hash, a HEX64 (Esplora's text, Bitcoin Core's result), and
//     median_time_past and transaction_count, each when the second call
//     gives it as an INT; or the block by its hash, whose value holds
//     height, median_time_past and time, each when the source gives it as
//     an INT. A field the second call does not give as an INT,
//     and both when that call gives no answer at all (a failure, a 404 or
//     a Bitcoin Core error included, or a body that is not a JSON object),
//     is absent from the value: the answer holds what the source gave, so
//     the rule that reads only the hash, genesis.bitcoin's, is judged on
//     it, and a rule that reads another field finds it unusable (section
//     10.4, "Answers": an answer is judged on the fields its rule reads at
//     that point, and a read made of several calls whose later call
//     answers not found or gives no usable answer, one that does not parse
//     included, is found with the fields its earlier calls gave). A field in another form is left out rather than copied, so a
//     value never holds what a server wrote in place of a number, such as
//     the credentials a node echoes back, and the rule judges it unusable
//     either way. A block by its hash whose one call gives no answer is
//     unreachable, as any read with no answer is: unlike the block at a
//     height, nothing of it has been given;
//   - not found: the first call of a block at a height, when Esplora
//     answers 404 or Bitcoin Core answers its error -8 (a height out of
//     range) or -5 (no such block): the source holds no best-chain block
//     at that height; and the block by its hash, when Esplora answers 404
//     or Bitcoin Core one of those errors: the source holds no block with
//     that hash. A tip is never not found (SPEC.md section 13.2), so a 404
//     or a Bitcoin Core error to it is unreachable;
//   - unreachable: every other answer, a 3xx, a 429, a 503, a challenge
//     page and a body over its bound among them, a tip or a hash that is
//     not in its form, any other Bitcoin Core error and an answer to
//     another call; a block by its hash whose answer is not a JSON object;
//     and a read whose height is not an INT, or whose hash is not a HEX64,
//     which is made to no server. Its reason is a fixed text, as the identity reads' are
//     ([fromRead]), quoting nothing a server sent.
//
// A Bitcoin Core call is a JSON-RPC 1.0 POST to the node's URL, with the
// node's credentials in its Authorization header and nowhere else, as
// [CoreIdentity] makes it. The node answers {result, error, id}, with
// error null and id this call's, and answers an error with an HTTP 404 or
// 500 whose body is that answer with the error in it (httpclient's
// PostRPC), so its error's code is read there; nothing else of the error
// is read. Esplora's reads send no credential.
//
// Every JSON answer is read whole by the strict reader
// (internal/strictjson), so a body it refuses, for a number past
// MAX_INTEGER in a member no read uses too, gives its call no answer
// (SPEC.md section 10.4, "Answers": an answer in JSON parses only when it
// meets J2 to J6 of section 10.1 in every member).
//
// No answer is compared with the height or the hash it was asked for:
// section 10.4 ("Answers") lets a verifier treat as unreachable an answer
// that does not name the object asked for, and these reads take the
// answer the source gives to the call they made. A Bitcoin Core answer
// must still be to this call, by its JSON-RPC id.
//
// The source's URL is the operator's or a compiled-in default, never the
// document's: a read's parameters name only the height or the hash asked
// for.

package source

import (
	"bytes"
	"context"
	"net/http"
	"regexp"
	"strconv"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// The kinds of Bitcoin chain read this file makes (SPEC.md section 13.2's
// table).
const (
	KindBitcoinTip           = "bitcoin_tip"
	KindBitcoinBlockAtHeight = "bitcoin_block_at_height"
	KindBitcoinBlock         = "bitcoin_block"
)

// BitcoinOctets bounds the body of each Bitcoin answer: a height, a hash,
// or a block's header fields, which Esplora and Bitcoin Core each give in
// about a kilobyte.
const BitcoinOctets = 65536

// The Bitcoin Core error codes that say a node holds no such block: -8, a
// height out of range, and -5, a block it does not have.
const (
	coreHeightOutOfRange = -8
	coreBlockNotFound    = -5
)

// decimalPattern is an integer in decimal digits, with no sign and no
// leading zero.
var decimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

// EsploraTip reads an Esplora server's GET <base>/blocks/tip/height, the
// height of its tip.
func EsploraTip(ctx context.Context, reads *httpclient.SingleRead, base string) Answer {
	body, err := reads.Get(ctx, endpoint(base, "/blocks/tip/height"), BitcoinOctets)
	if err != nil {
		return unreachableFrom(err)
	}
	height, ok := decimalInt(string(body))
	if !ok {
		return unreachableAnswer("Esplora's tip is not the decimal digits of an integer from 0 to 2^53 - 1")
	}
	return Answer{Outcome: OutcomeFound, Value: map[string]any{"height": height}}
}

// EsploraBlockAtHeight reads the best-chain block at height of an Esplora
// server: GET <base>/block-height/<height>, the block's hash, then GET
// <base>/block/<hash>, for its mediantime and tx_count.
func EsploraBlockAtHeight(ctx context.Context, reads *httpclient.SingleRead, base string, height any) Answer {
	h, ok := intParam(height)
	if !ok {
		return unreachableAnswer("the height is not an integer from 0 to 2^53 - 1")
	}
	body, err := reads.Get(ctx, endpoint(base, "/block-height/"+strconv.FormatInt(h, 10)), BitcoinOctets)
	if err != nil {
		if httpclient.Status(err) == http.StatusNotFound {
			return Answer{Outcome: OutcomeNotFound}
		}
		return unreachableFrom(err)
	}
	hash := string(body)
	if !hex64(hash) {
		return unreachableAnswer("Esplora's answer is not the 64 lowercase hex digits of a block hash")
	}
	answer := Answer{Outcome: OutcomeFound, Value: map[string]any{"hash": hash}}
	header, err := reads.Get(ctx, endpoint(base, "/block/"+hash), BitcoinOctets)
	if err != nil {
		return answer
	}
	block, _ := strictjson.Parse(header)
	addInts(answer.Value, block, map[string]string{"mediantime": "median_time_past", "tx_count": "transaction_count"})
	return answer
}

// CoreTip calls a Bitcoin Core node's getblockcount at url, the height of
// its tip, with creds.
func CoreTip(ctx context.Context, reads *httpclient.SingleRead, url string, creds Credentials) Answer {
	result, answer, ok := coreCall(ctx, reads, url, creds, "getblockcount", "")
	if !ok {
		if answer.Outcome == OutcomeNotFound {
			return unreachableAnswer("Bitcoin Core answered with an error")
		}
		return answer
	}
	height, isInt := intValue(result)
	if !isInt {
		return unreachableAnswer("Bitcoin Core's result is not an integer from 0 to 2^53 - 1")
	}
	return Answer{Outcome: OutcomeFound, Value: map[string]any{"height": height}}
}

// CoreBlockAtHeight reads the best-chain block at height of a Bitcoin Core
// node at url, with creds: getblockhash <height>, the block's hash, then
// getblockheader <hash>, for its mediantime and nTx.
func CoreBlockAtHeight(ctx context.Context, reads *httpclient.SingleRead, url string, creds Credentials, height any) Answer {
	h, ok := intParam(height)
	if !ok {
		return unreachableAnswer("the height is not an integer from 0 to 2^53 - 1")
	}
	result, answer, ok := coreCall(ctx, reads, url, creds, "getblockhash", strconv.FormatInt(h, 10))
	if !ok {
		return answer
	}
	hash, isString := result.(string)
	if !isString || !hex64(hash) {
		return unreachableAnswer("Bitcoin Core's result is not the 64 lowercase hex digits of a block hash")
	}
	answer = Answer{Outcome: OutcomeFound, Value: map[string]any{"hash": hash}}
	header, _, ok := coreCall(ctx, reads, url, creds, "getblockheader", `"`+hash+`"`)
	if ok {
		addInts(answer.Value, header, map[string]string{"mediantime": "median_time_past", "nTx": "transaction_count"})
	}
	return answer
}

// EsploraBlock reads the block whose hash is hash of an Esplora server,
// GET <base>/block/<hash>, for its height, timestamp and mediantime.
func EsploraBlock(ctx context.Context, reads *httpclient.SingleRead, base string, hash any) Answer {
	text, ok := hashParam(hash)
	if !ok {
		return unreachableAnswer("the block hash is not 64 lowercase hex digits")
	}
	body, err := reads.Get(ctx, endpoint(base, "/block/"+text), BitcoinOctets)
	if err != nil {
		if httpclient.Status(err) == http.StatusNotFound {
			return Answer{Outcome: OutcomeNotFound}
		}
		return unreachableFrom(err)
	}
	block, err := strictjson.Parse(body)
	if _, isObject := block.(map[string]any); err != nil || !isObject {
		return unreachableAnswer("Esplora's answer is not a JSON object")
	}
	return blockByHash(block, map[string]string{"height": "height", "mediantime": "median_time_past", "timestamp": "time"})
}

// CoreBlock reads the block whose hash is hash of a Bitcoin Core node at
// url, with creds: getblockheader <hash>, for its height, time and
// mediantime.
func CoreBlock(ctx context.Context, reads *httpclient.SingleRead, url string, creds Credentials, hash any) Answer {
	text, ok := hashParam(hash)
	if !ok {
		return unreachableAnswer("the block hash is not 64 lowercase hex digits")
	}
	result, answer, ok := coreCall(ctx, reads, url, creds, "getblockheader", `"`+text+`"`)
	if !ok {
		return answer
	}
	if _, isObject := result.(map[string]any); !isObject {
		return unreachableAnswer("Bitcoin Core's result is not a JSON object")
	}
	return blockByHash(result, map[string]string{"height": "height", "mediantime": "median_time_past", "time": "time"})
}

// blockByHash is the found answer of a block by its hash, holding each
// field of block that names gives a field of the value's name, when block
// holds it as an INT.
func blockByHash(block any, names map[string]string) Answer {
	value := map[string]any{}
	addInts(value, block, names)
	return Answer{Outcome: OutcomeFound, Value: value}
}

// coreCall calls method with params, the JSON text of its one parameter
// or "" for none, on the Bitcoin Core node at url with creds, and gives
// its result; or, when there is none, the answer the call stands for:
// not found for the error -8 or -5, and otherwise unreachable, with a
// fixed reason.
func coreCall(ctx context.Context, reads *httpclient.SingleRead, url string, creds Credentials, method, params string) (any, Answer, bool) {
	call := []byte(`{"jsonrpc":"1.0","id":"` + coreCallID + `","method":"` + method + `","params":[` + params + `]}`)
	header := creds.header()
	header.Set("Content-Type", "application/json")
	body, err := reads.PostRPC(ctx, url, call, header, BitcoinOctets)
	if err != nil {
		return nil, unreachableFrom(err), false
	}
	value, err := strictjson.Parse(bytes.TrimSpace(body))
	answer, isObject := value.(map[string]any)
	switch {
	case err != nil || !isObject:
		return nil, unreachableAnswer("Bitcoin Core's answer is not a JSON object"), false
	case answer["id"] != coreCallID:
		return nil, unreachableAnswer("Bitcoin Core's answer is not to this call"), false
	}
	if rpcError, present := answer["error"]; present && rpcError != nil {
		object, _ := rpcError.(map[string]any)
		if code, ok := intOf(object["code"]); ok && (code == coreHeightOutOfRange || code == coreBlockNotFound) {
			return nil, Answer{Outcome: OutcomeNotFound}, false
		}
		return nil, unreachableAnswer("Bitcoin Core answered with an error"), false
	}
	return answer["result"], Answer{}, true
}

// addInts adds to value each field of object, a JSON object, that names
// gives a field of the value's name, when object holds it as an INT.
func addInts(value map[string]any, object any, names map[string]string) {
	fields, _ := object.(map[string]any)
	for from, to := range names {
		if n, ok := intValue(fields[from]); ok {
			value[to] = n
		}
	}
}

// intValue is v as an INT (SPEC.md section 2.4), the strict reader's
// number from 0 to 2^53 - 1, and whether it is one.
func intValue(v any) (strictjson.Number, bool) {
	n, ok := intOf(v)
	if !ok || n < 0 {
		return 0, false
	}
	return strictjson.Number(float64(n)), true
}

// intOf is v as an integer of magnitude at most 2^53 - 1, and whether it
// is one.
func intOf(v any) (int64, bool) {
	number, isNumber := v.(strictjson.Number)
	n, isInt := number.Int64()
	return n, isNumber && isInt
}

// intParam is a read's integer parameter, a height or a NIST index, as an
// INT, and whether it is one.
func intParam(param any) (int64, bool) {
	n, ok := intOf(param)
	return n, ok && n >= 0
}

// hashParam is a read's hash parameter as a HEX64, and whether it is one.
func hashParam(param any) (string, bool) {
	text, ok := param.(string)
	return text, ok && hex64(text)
}

// decimalInt is text, an integer in decimal digits, as an INT, and
// whether it is one.
func decimalInt(text string) (strictjson.Number, bool) {
	if !decimalPattern.MatchString(text) {
		return 0, false
	}
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil || n > strictjson.MaxInteger {
		return 0, false
	}
	return strictjson.Number(float64(n)), true
}

// unreachableFrom is the unreachable answer of a read that failed, with
// [fromRead]'s fixed reason.
func unreachableFrom(err error) Answer {
	return unreachableAnswer(fromRead(err).(*Unreachable).Reason)
}
