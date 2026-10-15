// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file is the replay adapter of the online cases: it answers each
// read a verifier makes in an online case from the case's replay file,
// in place of the network (SPEC.md section 13.2 (The vectors), "Replay
// files"). It ports the Truestamp application's test reader,
// Truestamp.ProofReplayReader (test/support/proof_replay_reader.ex in
// truestamp-v2), so far for the kinds a verifier's step 0 reads: keyring,
// and the sources' identities, stellar_identity and bitcoin_identity; the
// reads of the chains and the entropy sources join with the steps that
// make them. The file stands as the operator-pinned source of each Stellar
// and Bitcoin network whose identity read it holds ([Replay.Sources]), so
// a case that pins such a network, regtest included, has a source for it.
//
// Adapted from the application: a file is read with the strict reader
// (internal/strictjson), so a found answer's value is in the values every
// later step reads, and a keyring answered from it is a value with no bytes
// of its own, whose digest is never compared (section 13.2). A read is
// found by building the same object the file holds, compared as its
// encoding/json form with its members sorted, so the order of a read's
// members carries no meaning. Where the application's reader raises on a
// read the file does not hold, [Replay.Keyring] and [Replay.Identity] give
// [ErrNoAnswer], which the test that makes the read fails on.
//
// A known limit: a replayed keyring is a found answer the verifier loads
// only as the keyring of www.truestamp.com. As the answer to a read of the
// origin an explicit --base-url names, it is a failed read
// (keyring_unavailable), though section 13.2 lets a keyring read name that
// origin. Such a keyring is a supplied keyring, whose attribution
// keyring:<d> names the digest of its bytes as read (sections 8.3 and
// 10.5), and a replayed value has no bytes; section 13.2 does not say what
// the attribution of a replayed supplied keyring is. No case's
// configuration can name a --base-url, so no case reaches it.

package vectorcases

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// The outcomes of a replayed answer (SPEC.md section 13.2).
const (
	OutcomeFound       = "found"
	OutcomeNotFound    = "not_found"
	OutcomeUnreachable = "unreachable"
)

// Answer is one replayed answer: Outcome, with Value for a found answer and
// Reason, informational only, for an unreachable one.
type Answer struct {
	Outcome string
	Value   map[string]any
	Reason  string
}

// Replay is a replay file's reads, by read.
type Replay struct {
	answers map[string]Answer
	sources []Source
}

// Source is a Stellar or Bitcoin network whose identity read a replay file
// holds, found or unreachable: the file stands as that network's
// operator-pinned source (SPEC.md section 13.2).
type Source struct {
	Chain   string
	Network string
}

// identityChain is the chain of each identity kind.
var identityChain = map[string]string{"stellar_identity": "stellar", "bitcoin_identity": "bitcoin"}

// identityField is the one field of each chain's found identity value.
var identityField = map[string]string{"stellar": "network_passphrase", "bitcoin": "hash"}

// ErrNoAnswer marks a read the replay file does not hold: under an online
// case it has no answer, and the test that makes it fails (SPEC.md section
// 13.2).
var ErrNoAnswer = errors.New("vectorcases: a read the replay file does not hold")

// ErrUnreachable marks a read whose replayed answer is unreachable.
var ErrUnreachable = errors.New("vectorcases: the replayed answer is unreachable")

// neverNotFound are the kinds of read that are never not_found: a source's
// identity, the Stellar root, the Bitcoin tip and the keyring.
var neverNotFound = map[string]bool{
	"stellar_identity": true, "bitcoin_identity": true, "stellar_root": true, "bitcoin_tip": true, "keyring": true,
}

// LoadReplay reads the replay file at relative, a path under vectors/ as a
// case's online member writes it. The file is an object whose one member,
// reads, is an array of {answer, read} records, no two with the same read;
// each answer is {outcome: found, value}, {outcome: not_found} or
// {outcome: unreachable, reason}, and a kind that is never not_found has
// no such answer; a keyring read's found value is exactly {keyring}; and an
// identity read is exactly {kind, network}, a network of its chain, whose
// found value is exactly {network_passphrase}, an ASCII128, for Stellar,
// and {hash}, a HEX64, for Bitcoin (SPEC.md section 13.2's table).
func LoadReplay(relative string) (*Replay, error) {
	data, err := os.ReadFile(Path(relative))
	if err != nil {
		return nil, err
	}
	replay, err := ParseReplay(data)
	if err != nil {
		return nil, fmt.Errorf("vectorcases: %s: %w", relative, err)
	}
	return replay, nil
}

// ParseReplay reads a replay file's bytes, as [LoadReplay] does.
func ParseReplay(data []byte) (*Replay, error) {
	value, err := strictjson.Parse(data)
	if err != nil {
		return nil, err
	}
	fail := func(format string, args ...any) (*Replay, error) {
		return nil, fmt.Errorf(format, args...)
	}
	file, ok := value.(map[string]any)
	reads, isArray := file["reads"].([]any)
	if !ok || len(file) != 1 || !isArray {
		return fail("not an object of exactly reads, an array")
	}
	replay := &Replay{answers: make(map[string]Answer, len(reads))}
	for i, element := range reads {
		record, ok := element.(map[string]any)
		read, isRead := record["read"].(map[string]any)
		answer, isAnswer := record["answer"].(map[string]any)
		if !ok || len(record) != 2 || !isRead || !isAnswer {
			return fail("record %d is not {answer, read}", i)
		}
		kind, _ := read["kind"].(string)
		parsed, err := answerOf(answer, kind)
		if err != nil {
			return fail("record %d: %v", i, err)
		}
		if chain, isIdentity := identityChain[kind]; isIdentity {
			network, err := identityRead(read, chain, parsed)
			if err != nil {
				return fail("record %d: %v", i, err)
			}
			replay.sources = append(replay.sources, Source{Chain: chain, Network: network})
		}
		key, err := readKey(read)
		if err != nil {
			return fail("record %d: %v", i, err)
		}
		if _, repeated := replay.answers[key]; repeated {
			return fail("record %d repeats the read %s", i, key)
		}
		replay.answers[key] = parsed
	}
	return replay, nil
}

// answerOf checks one answer of a read of kind.
func answerOf(answer map[string]any, kind string) (Answer, error) {
	outcome, _ := answer["outcome"].(string)
	switch outcome {
	case OutcomeFound:
		value, ok := answer["value"].(map[string]any)
		if len(answer) != 2 || !ok {
			return Answer{}, errors.New("a found answer is not {outcome, value}")
		}
		if _, hasKeyring := value["keyring"]; kind == "keyring" && (len(value) != 1 || !hasKeyring) {
			return Answer{}, errors.New("a keyring read's value is not {keyring}")
		}
		return Answer{Outcome: outcome, Value: value}, nil
	case OutcomeNotFound:
		if len(answer) != 1 || neverNotFound[kind] {
			return Answer{}, fmt.Errorf("a not_found answer to a read of kind %q", kind)
		}
		return Answer{Outcome: outcome}, nil
	case OutcomeUnreachable:
		reason, ok := answer["reason"].(string)
		if len(answer) != 2 || !ok {
			return Answer{}, errors.New("an unreachable answer is not {outcome, reason}")
		}
		return Answer{Outcome: outcome, Reason: reason}, nil
	}
	return Answer{}, fmt.Errorf("the outcome %q", outcome)
}

// readKey is the form a read is found by: its encoding/json text, whose
// object members encoding/json writes sorted by name.
func readKey(read map[string]any) (string, error) {
	key, err := json.Marshal(read)
	if err != nil {
		return "", fmt.Errorf("a read encoding/json cannot write: %w", err)
	}
	return string(key), nil
}

// Answer returns the replayed answer to read, and whether the file holds
// it.
func (r *Replay) Answer(read map[string]any) (Answer, bool) {
	key, err := readKey(read)
	if err != nil {
		return Answer{}, false
	}
	answer, ok := r.answers[key]
	return answer, ok
}

// Keyring answers the keyring read at url (SPEC.md section 13.2's keyring
// kind): the keyring value read, which the load checks then judge, for a
// found answer; an error wrapping [ErrUnreachable], with the reason, for an
// unreachable one; and [ErrNoAnswer] when the file holds no read of url.
func (r *Replay) Keyring(url string) (any, error) {
	answer, ok := r.Answer(map[string]any{"kind": "keyring", "url": url})
	switch {
	case !ok:
		return nil, fmt.Errorf("%w: the keyring at %s", ErrNoAnswer, url)
	case answer.Outcome == OutcomeUnreachable:
		return nil, fmt.Errorf("%w: %s", ErrUnreachable, answer.Reason)
	}
	return answer.Value["keyring"], nil
}

// identityRead checks an identity read of chain and its answer, and
// returns the read's network.
func identityRead(read map[string]any, chain string, answer Answer) (string, error) {
	network, _ := read["network"].(string)
	if len(read) != 2 || !slices.Contains(specdata.Networks()[chain], network) {
		return "", fmt.Errorf("an identity read is not {kind, network} with a network of %s", chain)
	}
	if answer.Outcome != OutcomeFound {
		return network, nil
	}
	field := identityField[chain]
	value, isString := answer.Value[field].(string)
	class := map[string]string{"stellar": "ASCII128", "bitcoin": "HEX64"}[chain]
	pattern := regexp.MustCompile(`^` + specdata.Encodings()[class].Pattern + `$`)
	if len(answer.Value) != 1 || !isString || !pattern.MatchString(value) {
		return "", fmt.Errorf("a %s identity's value is not exactly {%s}, a %s", chain, field, class)
	}
	return network, nil
}

// Sources returns the networks whose identity read the file holds, in the
// file's order: the file stands as the operator-pinned source of each.
func (r *Replay) Sources() []Source { return slices.Clone(r.sources) }

// Identity answers the identity read of network of chain (SPEC.md section
// 13.2's kinds stellar_identity and bitcoin_identity): the value read,
// Stellar's network_passphrase or the hash of Bitcoin's block at height 0,
// for a found answer; an error wrapping [ErrUnreachable], with the reason,
// for an unreachable one; and [ErrNoAnswer] when the file holds no such
// read.
func (r *Replay) Identity(chain, network string) (string, error) {
	answer, ok := r.Answer(map[string]any{"kind": chain + "_identity", "network": network})
	switch {
	case !ok:
		return "", fmt.Errorf("%w: the identity of %s %s", ErrNoAnswer, chain, network)
	case answer.Outcome == OutcomeUnreachable:
		return "", fmt.Errorf("%w: %s", ErrUnreachable, answer.Reason)
	}
	value, _ := answer.Value[identityField[chain]].(string)
	return value, nil
}
