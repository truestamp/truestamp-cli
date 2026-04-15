// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/oklog/ulid/v2"
)

// IDType represents the detected ID format.
type IDType string

const (
	IDTypeItem    IDType = "item"
	IDTypeEntropy IDType = "entropy"
)

// DetectIDType determines whether an ID is a ULID (item) or UUIDv7 (entropy).
func DetectIDType(id string) (IDType, error) {
	if _, err := ulid.ParseStrict(strings.ToUpper(id)); err == nil {
		return IDTypeItem, nil
	}
	if _, err := uuid.FromString(id); err == nil {
		return IDTypeEntropy, nil
	}
	return "", fmt.Errorf("invalid ID format %q: expected ULID (item, e.g. 01KNN33GX5E470CB9TRWAYF9DD) or UUIDv7 (entropy, e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)", id)
}