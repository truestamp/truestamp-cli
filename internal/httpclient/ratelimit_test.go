// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"", 0, false},
		{"37", 37 * time.Second, true},
		{" 1 ", time.Second, true},
		{"0", 0, true},
		{"-5", 0, false},
		{"soon", 0, false},
		{"1.5", 0, false},
		{now.Add(90 * time.Second).Format(http.TimeFormat), 90 * time.Second, true},
		{now.Add(-90 * time.Second).Format(http.TimeFormat), 0, true},
	}
	for _, tc := range cases {
		got, ok := ParseRetryAfter(tc.in, now)
		if ok != tc.ok || got != tc.want {
			t.Errorf("ParseRetryAfter(%q) = %v, %v; want %v, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestBoundRetry(t *testing.T) {
	cases := []struct {
		delay time.Duration
		named bool
		want  time.Duration
		ok    bool
	}{
		{0, false, DefaultRetryAfter, true},
		{5 * time.Minute, false, DefaultRetryAfter, true},
		{30 * time.Second, true, 30 * time.Second, true},
		{MaxRetryAfter, true, MaxRetryAfter, true},
		{MaxRetryAfter + time.Second, true, MaxRetryAfter + time.Second, false},
	}
	for _, tc := range cases {
		got, ok := BoundRetry(tc.delay, tc.named)
		if ok != tc.ok || got != tc.want {
			t.Errorf("BoundRetry(%v, %v) = %v, %v; want %v, %v", tc.delay, tc.named, got, ok, tc.want, tc.ok)
		}
	}
}

// stubSleep replaces the sleeper for one test and records every wait, so
// a 30 s Retry-After is observed rather than spent.
func stubSleep(t *testing.T) *[]time.Duration {
	t.Helper()
	var waits []time.Duration
	orig := sleep
	sleep = func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)
		return ctx.Err()
	}
	t.Cleanup(func() { sleep = orig })
	return &waits
}

// refuseThenAccept is a server that answers the first n requests with
// 429 (and the given Retry-After, when non-empty) and every later one
// with 200 echoing the request body.
func refuseThenAccept(t *testing.T, n int32, retryAfter string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if calls.Add(1) <= n {
			if retryAfter != "" {
				w.Header().Set("Retry-After", retryAfter)
			}
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"slow_down","error_description":"Rate limit exceeded; retry after 30 seconds"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestRetryAfterTransport_RetriesOnceAfterHeader(t *testing.T) {
	waits := stubSleep(t)
	srv, calls := refuseThenAccept(t, 1, "30")
	client := &http.Client{Transport: NewRetryAfterTransport(nil)}

	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader("grant_type=refresh_token"))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 after the retry", resp.StatusCode)
	}
	if string(body) != "grant_type=refresh_token" {
		t.Errorf("the retry must resend the body, server saw %q", body)
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want exactly 2", calls.Load())
	}
	if len(*waits) != 1 || (*waits)[0] != 30*time.Second {
		t.Errorf("waits = %v, want [30s] from Retry-After", *waits)
	}
}

func TestRetryAfterTransport_DefaultWaitWhenNoHeader(t *testing.T) {
	waits := stubSleep(t)
	srv, calls := refuseThenAccept(t, 1, "")
	client := &http.Client{Transport: NewRetryAfterTransport(nil)}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || calls.Load() != 2 {
		t.Errorf("status %d after %d calls, want 200 after 2", resp.StatusCode, calls.Load())
	}
	if len(*waits) != 1 || (*waits)[0] != DefaultRetryAfter {
		t.Errorf("waits = %v, want [%v]", *waits, DefaultRetryAfter)
	}
}

func TestRetryAfterTransport_SecondRefusalIsReturned(t *testing.T) {
	stubSleep(t)
	srv, calls := refuseThenAccept(t, 2, "5")
	client := &http.Client{Transport: NewRetryAfterTransport(nil)}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, want the second 429 surfaced", resp.StatusCode)
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want 2: one retry, never more", calls.Load())
	}
}

func TestRetryAfterTransport_GivesUpOverCap(t *testing.T) {
	waits := stubSleep(t)
	srv, calls := refuseThenAccept(t, 1, "3600")
	client := &http.Client{Transport: NewRetryAfterTransport(nil)}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests || calls.Load() != 1 {
		t.Errorf("status %d after %d calls, want the 429 surfaced at once", resp.StatusCode, calls.Load())
	}
	if len(*waits) != 0 {
		t.Errorf("a wait over MaxRetryAfter must not be slept: %v", *waits)
	}
	if got := resp.Header.Get("Retry-After"); got != "3600" {
		t.Errorf("the refusal's Retry-After must reach the caller, got %q", got)
	}
}

// bodyOnly hides the concrete reader type so http.NewRequest cannot
// derive GetBody: the body is streamed and cannot be sent twice.
type bodyOnly struct{ io.Reader }

func TestRetryAfterTransport_UnrewindableBodyIsNotRetried(t *testing.T) {
	waits := stubSleep(t)
	srv, calls := refuseThenAccept(t, 1, "1")
	client := &http.Client{Transport: NewRetryAfterTransport(nil)}

	req, _ := http.NewRequest(http.MethodPost, srv.URL, bodyOnly{strings.NewReader("once")})
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests || calls.Load() != 1 || len(*waits) != 0 {
		t.Errorf("a body that cannot be rewound must surface the 429: status %d, calls %d, waits %v",
			resp.StatusCode, calls.Load(), *waits)
	}
}

func TestRetryAfterTransport_OtherStatusesUntouched(t *testing.T) {
	waits := stubSleep(t)
	for _, code := range []int{400, 401, 403, 404, 500, 503} {
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(code)
		}))
		client := &http.Client{Transport: NewRetryAfterTransport(nil)}
		resp, err := client.Get(srv.URL)
		srv.Close()
		if err != nil {
			t.Fatalf("HTTP %d: %v", code, err)
		}
		resp.Body.Close()
		if resp.StatusCode != code || calls.Load() != 1 {
			t.Errorf("HTTP %d: status %d after %d calls, want untouched", code, resp.StatusCode, calls.Load())
		}
	}
	if len(*waits) != 0 {
		t.Errorf("no status but 429 may wait: %v", *waits)
	}
}

func TestRetryAfterTransport_CancelledDuringWait(t *testing.T) {
	// The real sleeper: a cancelled context ends the wait with its error.
	srv, calls := refuseThenAccept(t, 1, "30")
	client := &http.Client{Transport: NewRetryAfterTransport(nil)}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	_, err := client.Do(req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled from the wait", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1: the retry never went out", calls.Load())
	}
}

func TestRewind_NilAndNoBodyRequests(t *testing.T) {
	for _, body := range []io.Reader{nil, http.NoBody} {
		req, _ := http.NewRequest(http.MethodGet, "http://example.invalid/", body)
		if _, ok := Rewind(req); !ok {
			t.Errorf("a request with body %v must rewind", body)
		}
	}
}
