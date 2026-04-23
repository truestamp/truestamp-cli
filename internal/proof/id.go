// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/oklog/ulid/v2"
)

// IDType is the syntactic shape of a subject id. It is NOT the subject's
// semantic kind (item / entropy / block / beacon) — the server decides that
// from the id on /proof/generate. Use DetectIDType for pre-flight validation
// and for sanity-checking positional args; rely on the server's returned `t`
// for authoritative subject labelling.
type IDType string

const (
	IDTypeULID   IDType = "ulid"   // 26-char Crockford base32 (items)
	IDTypeUUIDv7 IDType = "uuidv7" // UUIDv7, with or without hyphens (entropy / block)
)

// DetectIDType returns the syntactic shape of id. Returns an error for
// values that are neither a valid ULID nor a parseable UUID.
func DetectIDType(id string) (IDType, error) {
	if _, err := ulid.ParseStrict(strings.ToUpper(id)); err == nil {
		return IDTypeULID, nil
	}
	if _, err := uuid.FromString(id); err == nil {
		return IDTypeUUIDv7, nil
	}
	return "", fmt.Errorf("invalid ID format %q: expected ULID (e.g. 01KNN33GX5E470CB9TRWAYF9DD) or UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)", id)
}
