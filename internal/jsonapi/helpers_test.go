// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package jsonapi

import (
	"bytes"
	"io"
	"strconv"
)

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
