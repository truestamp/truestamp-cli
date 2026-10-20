// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file is the replay adapter of the online cases: it answers each
// read a verifier makes in an online case from the case's replay file,
// in place of the network (SPEC.md section 13.2 (The vectors), "Replay
// files"): the kinds a verifier's step 0 reads, keyring and the sources'
// identities, stellar_identity and bitcoin_identity, each through its own
// method, and every read of a chain or an entropy source, NIST's pulse
// among them, through [Replay.Answer]. The file stands as the
// operator-pinned source of each Stellar and Bitcoin network whose
// identity read it holds, or as each named source whose identity read it
// holds ([Replay.Sources]), so a case that pins such a network, regtest
// included, has a source for it.
//
// A record is {answer, read} or {answer, read, source}. Its source, present
// only in a record of a read of a Stellar or Bitcoin network that the case's
// sources name, is the name of the source that gives the answer, and a
// record without one answers for its network's one source (SPEC.md section
// 13.2), so a read is found by the read and the source together, and no two
// records hold the same read and the same source or both none.
// [LoadCaseReplay] also holds each record's source to the case's sources.
//
// A file is read with the strict reader (internal/strictjson), so a found
// answer's value is in the values every later step reads, and a keyring
// answered from it is a value with no bytes of its own, whose digest is
// never compared (section 13.2). A read is found by building the same
// object the file holds, compared as its encoding/json form with its
// members sorted, so the order of a read's members carries no meaning. A
// read the file does not hold has no answer (section 13.2):
// [Replay.Keyring] and [Replay.Identity] give [ErrNoAnswer], which the test
// that makes the read fails on.
//
// A case's config names no origin, so every keyring read a case makes is of
// www.truestamp.com, and a replayed keyring is a found answer the verifier
// loads only as the keyring of www.truestamp.com. The CLI's --base-url, an
// option to read the keyring from another origin, gives a supplied keyring
// that no case can name, so it is tested in the verifier's own tests
// (SPEC.md section 13.2, "Replay files", and section 8.3).

package vectorcases

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

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

// Replay is a replay file's reads, by read and source.
type Replay struct {
	answers map[replayKey]Answer
	records []Record
	sources []Source
}

// replayKey finds a record: its read's encoding/json text and its source,
// the empty string for a record without one.
type replayKey struct {
	read, source string
}

// Record is what a record holds besides its answer: the kind and the
// network of its read, the network the empty string for a read with none,
// and its source, the empty string for a record without one.
type Record struct {
	Kind    string
	Network string
	Source  string
}

// Source is a Stellar or Bitcoin source whose identity read a replay file
// holds, found or unreachable: Name is the empty string for the record of a
// network's one source, and the source's name for a named one. The file
// stands as that network's operator-pinned source, or as that named source
// (SPEC.md section 13.2).
type Source struct {
	Chain   string
	Network string
	Name    string
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
// reads, is an array of {answer, read} and {answer, read, source} records,
// no two with the same read and the same source or both without one; a
// source is an ASCII16, and only a read of a Stellar or Bitcoin network,
// one whose kind begins stellar_ or bitcoin_, has one; each answer is
// {outcome: found, value}, {outcome: not_found} or {outcome: unreachable,
// reason}, and a kind that is never not_found has no such answer; a keyring
// read's found value is exactly {keyring}; and an identity read is exactly
// {kind, network}, a network of its chain, whose found value is exactly
// {network_passphrase}, an ASCII128, for Stellar, and {hash}, a HEX64, for
// Bitcoin (SPEC.md section 13.2's table).
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

// LoadCaseReplay reads the replay file of the online case c, as
// [LoadReplay] does, and holds each record's source to c's sources (SPEC.md
// section 13.2): a record of a read of a network c's sources name has a
// source, one of that network's names there, and no other record has one.
func LoadCaseReplay(c Case) (*Replay, error) {
	if c.Config.Online == nil {
		return nil, fmt.Errorf("vectorcases: %s is offline", c.ID)
	}
	replay, err := LoadReplay(*c.Config.Online)
	if err != nil {
		return nil, err
	}
	for i, record := range replay.records {
		chain, _, _ := strings.Cut(record.Kind, "_")
		named, hasNames := c.Config.Sources[chain][record.Network]
		switch {
		case hasNames && !slices.Contains(named.Sources, record.Source):
			return nil, fmt.Errorf("vectorcases: %s: record %d, a read of %s %s, has the source %q, not one of %q", *c.Config.Online, i, chain, record.Network, record.Source, named.Sources)
		case !hasNames && record.Source != "":
			return nil, fmt.Errorf("vectorcases: %s: record %d has the source %q, though the case names no sources of its read's network", *c.Config.Online, i, record.Source)
		}
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
	ascii16 := regexp.MustCompile(`^` + specdata.Encodings()["ASCII16"].Pattern + `$`)
	replay := &Replay{answers: make(map[replayKey]Answer, len(reads))}
	for i, element := range reads {
		record, ok := element.(map[string]any)
		read, isRead := record["read"].(map[string]any)
		answer, isAnswer := record["answer"].(map[string]any)
		source, hasSource := record["source"]
		name, isName := source.(string)
		if !ok || !isRead || !isAnswer || len(record) != map[bool]int{false: 2, true: 3}[hasSource] {
			return fail("record %d is not {answer, read} or {answer, read, source}", i)
		}
		kind, _ := read["kind"].(string)
		network, _ := read["network"].(string)
		if hasSource && (!isName || !ascii16.MatchString(name)) {
			return fail("record %d's source is not an ASCII16", i)
		}
		if hasSource && !strings.HasPrefix(kind, "stellar_") && !strings.HasPrefix(kind, "bitcoin_") {
			return fail("record %d has a source, though its read, of kind %q, is of no Stellar or Bitcoin network", i, kind)
		}
		parsed, err := answerOf(answer, kind)
		if err != nil {
			return fail("record %d: %v", i, err)
		}
		if chain, isIdentity := identityChain[kind]; isIdentity {
			network, err := identityRead(read, chain, parsed)
			if err != nil {
				return fail("record %d: %v", i, err)
			}
			replay.sources = append(replay.sources, Source{Chain: chain, Network: network, Name: name})
		}
		text, err := readKey(read)
		if err != nil {
			return fail("record %d: %v", i, err)
		}
		key := replayKey{read: text, source: name}
		if _, repeated := replay.answers[key]; repeated {
			return fail("record %d repeats the read %s from the source %q", i, text, name)
		}
		replay.answers[key] = parsed
		replay.records = append(replay.records, Record{Kind: kind, Network: network, Source: name})
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

// Answer returns the replayed answer to read from source, the empty string
// for a network's one source and for a read of no network, and whether the
// file holds it.
func (r *Replay) Answer(read map[string]any, source string) (Answer, bool) {
	text, err := readKey(read)
	if err != nil {
		return Answer{}, false
	}
	answer, ok := r.answers[replayKey{read: text, source: source}]
	return answer, ok
}

// Records returns what each record holds besides its answer, in the file's
// order.
func (r *Replay) Records() []Record { return slices.Clone(r.records) }

// Keyring answers the keyring read at url (SPEC.md section 13.2's keyring
// kind): the keyring value read, which the load checks then judge, for a
// found answer; an error wrapping [ErrUnreachable], with the reason, for an
// unreachable one; and [ErrNoAnswer] when the file holds no read of url.
func (r *Replay) Keyring(url string) (any, error) {
	answer, ok := r.Answer(map[string]any{"kind": "keyring", "url": url}, "")
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

// Sources returns the sources whose identity read the file holds, in the
// file's order: the file stands as each (SPEC.md section 13.2).
func (r *Replay) Sources() []Source { return slices.Clone(r.sources) }

// Identity answers the identity read of network of chain from source, the
// empty string for the network's one source (SPEC.md section 13.2's kinds
// stellar_identity and bitcoin_identity): the value read, Stellar's
// network_passphrase or the hash of Bitcoin's block at height 0, for a
// found answer; an error wrapping [ErrUnreachable], with the reason, for an
// unreachable one; and [ErrNoAnswer] when the file holds no such read from
// that source.
func (r *Replay) Identity(chain, network, source string) (string, error) {
	answer, ok := r.Answer(map[string]any{"kind": chain + "_identity", "network": network}, source)
	switch {
	case !ok:
		return "", fmt.Errorf("%w: the identity of %s %s from the source %q", ErrNoAnswer, chain, network, source)
	case answer.Outcome == OutcomeUnreachable:
		return "", fmt.Errorf("%w: %s", ErrUnreachable, answer.Reason)
	}
	value, _ := answer.Value[identityField[chain]].(string)
	return value, nil
}
