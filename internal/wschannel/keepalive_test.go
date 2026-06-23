// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"testing"
	"time"
)

func TestKeepAliveDue(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	lead := 60 * time.Second
	cases := []struct {
		name string
		exp  time.Time
		want bool
	}{
		{"zero expiry (unknown) is not due", time.Time{}, false},
		{"far future is not due", now.Add(10 * time.Minute), false},
		{"just past the lead window is not due", now.Add(61 * time.Second), false},
		{"within the lead window is due", now.Add(30 * time.Second), true},
		{"exactly at the lead boundary is due", now.Add(60 * time.Second), true},
		{"already expired is due", now.Add(-time.Minute), true},
	}
	for _, c := range cases {
		if got := keepAliveDue(c.exp, now, lead); got != c.want {
			t.Errorf("%s: keepAliveDue=%v want %v", c.name, got, c.want)
		}
	}
}
