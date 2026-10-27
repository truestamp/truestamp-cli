// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package external

import "fmt"

// MalformedResponseError reports a 2xx response this client could not
// interpret as a keyring: a body the strict reader refuses, or a value
// that does not have a keyring's shape. Err, when set, is the reader's
// refusal.
type MalformedResponseError struct {
	Source string
	Detail string
	Err    error
}

func (e *MalformedResponseError) Error() string {
	return fmt.Sprintf("%s response malformed: %s", e.Source, e.Detail)
}

func (e *MalformedResponseError) Unwrap() error { return e.Err }
