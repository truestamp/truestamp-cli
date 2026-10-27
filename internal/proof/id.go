// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package proof

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/ids"
	"github.com/truestamp/truestamp-cli/internal/itemcode"
)

// IDType is the syntactic shape of a subject id. It is NOT the subject's
// semantic kind (item / entropy / block / beacon): the raw proof route,
// GET /proof/raw/{type}/{id}, takes the type in its path, and
// ResolveSubjectType asks the server for it when the caller gives none
// (see cmd/proofs_get.go's id-shape default). The
// authoritative label for a proof in hand is its own signed `type`
// (SPEC.md section 3.2 (Members of `proof`)), which internal/verifier reads
// once the strict reader of internal/strictjson has parsed the file, and
// checks at SHAPE (SPEC.md section 10.2, step 7). Use DetectIDType for
// pre-flight validation and for sanity-checking positional args.
type IDType string

const (
	// IDTypeUUIDv7 is the 36-character text of a UUIDv7 (RFC 9562 section
	// 5.7), the id of every item, block, entropy observation and beacon
	// (SPEC.md section 2.4, the UUID7 class).
	IDTypeUUIDv7 IDType = "uuidv7"
	// IDTypeItemCode is an item code as SPEC.md section 3.9 (The item
	// code) lets a reader take it, which names only an item.
	IDTypeItemCode IDType = "item_code"
	// IDTypeItemHash is 64 hex characters, which names a subject only as
	// an item's item_hash, so a caller takes it only where the subject is
	// known to be an item.
	IDTypeItemHash IDType = "item_hash"
)

// Why DetectIDType refused a value, for errors.Is.
var (
	// ErrNotAnID is a value that is no id, item code or item_hash.
	ErrNotAnID = errors.New("not an id")
	// ErrItemCodeTypo is a value with the shape of an item code that
	// holds a U or fails the code's check.
	ErrItemCodeTypo = errors.New("an item code with a typo")
)

// idError is a refusal of DetectIDType: its message names the value, and
// it unwraps to the kind of refusal.
type idError struct {
	kind error
	msg  string
}

func (e *idError) Error() string { return e.msg }
func (e *idError) Unwrap() error { return e.kind }

func refuseID(kind error, format string, args ...any) error {
	return &idError{kind: kind, msg: fmt.Sprintf(format, args...)}
}

var (
	uuidText  = regexp.MustCompile(`\A[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\z`)
	hex64Text = regexp.MustCompile(`\A[0-9a-fA-F]{64}\z`)
	hexText   = regexp.MustCompile(`\A[0-9a-fA-F]{32,128}\z`)
)

// DetectIDType returns the syntactic shape of id and its canonical form: a
// UUIDv7 or an item_hash in lowercase, an item code in its one uppercase
// spelling.
//
// id is classified exactly as given, as the app's Truestamp.IdShape does:
// nothing is trimmed and no character is mapped, so a leading space or a
// trailing line feed makes a value no id. A UUID is only its 36-character
// text with hyphens, in either case, and only of version 7 with the RFC
// 9562 variant, by ids.IsUUIDv7Text, the one UUIDv7 rule of every command:
// its 32-hex, braced and urn: forms and every other version are no id. An
// item code is read by itemcode.Parse, as SPEC.md section 3.9 (The item
// code) lets a reader read one; a value with its shape that holds
// a U or fails the check is [ErrItemCodeTypo]. Every other value is
// [ErrNotAnID], whose message says what the value is and not what is
// expected: each command takes its own forms, so the caller names them.
func DetectIDType(id string) (IDType, string, error) {
	switch {
	case ids.IsUUIDv7Text(id):
		return IDTypeUUIDv7, strings.ToLower(id), nil
	case uuidText.MatchString(id):
		return "", "", refuseID(ErrNotAnID, "%q is a UUID of another version", id)
	case hex64Text.MatchString(id):
		return IDTypeItemHash, strings.ToLower(id), nil
	case hexText.MatchString(id):
		return "", "", refuseID(ErrNotAnID, "%q is %d hex characters, which is no id", id, len(id))
	}
	code, err := itemcode.Parse(id)
	switch {
	case err == nil:
		return IDTypeItemCode, code, nil
	case errors.Is(err, itemcode.ErrTypo):
		return "", "", refuseID(ErrItemCodeTypo, "%q has the shape of an item code but is not one: a symbol is mistyped or two are exchanged", id)
	}
	return "", "", refuseID(ErrNotAnID, "invalid ID format %q", id)
}
