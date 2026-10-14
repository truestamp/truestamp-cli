// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package netguardcontrol holds the control that proves the no-network guard
// of tools/nonet fires. It is its own module, so it never touches the CLI's
// packages. It passes only when each attempt is refused with EPERM, the
// sandbox's answer; a reachable host or any other error fails it.
package netguardcontrol

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestControl_LoopbackStillWorks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") }))
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("loopback httptest GET failed: %v", err)
	}
	b, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if string(b) != "ok" {
		t.Fatalf("loopback body = %q", b)
	}
	l, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Fatalf("listening on [::1]: %v", err)
	}
	srv6 := httptest.NewUnstartedServer(http.NotFoundHandler())
	srv6.Listener = l
	srv6.Start()
	defer srv6.Close()
	res6, err := http.Get(srv6.URL)
	if err != nil {
		t.Fatalf("loopback ::1 GET failed: %v", err)
	}
	_ = res6.Body.Close()
}

func TestControl_DialRealHostIsRefused(t *testing.T) {
	// By address, so no DNS is involved.
	for _, addr := range []string{"1.1.1.1:443", "[2606:4700:4700::1111]:443"} {
		c, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err == nil {
			_ = c.Close()
			t.Fatalf("GUARD ABSENT: connected to %s", addr)
		}
		if !errors.Is(err, syscall.EPERM) {
			t.Fatalf("dial %s failed, but not by the guard (want EPERM): %v", addr, err)
		}
	}
	c, err := net.DialTimeout("udp", "8.8.8.8:53", 5*time.Second)
	if err == nil {
		_, err = c.Write([]byte{0})
		_ = c.Close()
	}
	if !errors.Is(err, syscall.EPERM) {
		t.Fatalf("UDP to 8.8.8.8:53 not refused by the guard: %v", err)
	}
}

func TestControl_DNSIsRefused(t *testing.T) {
	for _, pref := range []bool{false, true} {
		r := &net.Resolver{PreferGo: pref}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		addrs, err := r.LookupHost(ctx, "www.truestamp.com")
		cancel()
		if err == nil {
			t.Fatalf("GUARD ABSENT: resolved www.truestamp.com (PreferGo=%v): %v", pref, addrs)
		}
		t.Logf("PreferGo=%v lookup refused: %v", pref, err)
	}
}

func TestControl_HTTPGetRealHostIsRefused(t *testing.T) {
	cl := &http.Client{Timeout: 10 * time.Second}
	_, err := cl.Get("https://1.1.1.1/")
	if err == nil || !errors.Is(err, syscall.EPERM) {
		t.Fatalf("GET https://1.1.1.1/ not refused by the guard: %v", err)
	}
	_, err = cl.Get("https://www.truestamp.com/.well-known/keyring.json")
	if err == nil {
		t.Fatalf("GUARD ABSENT: fetched the production keyring")
	}
	t.Logf("named-host GET refused: %v", err)
}

func TestControl_LocalServicesAreRefused(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:4000", "127.0.0.1:5432", "[::1]:4000"} {
		c, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			_ = c.Close()
			t.Fatalf("GUARD ABSENT: connected to local service %s", addr)
		}
		if !errors.Is(err, syscall.EPERM) {
			t.Fatalf("dial %s failed, but not by the guard (want EPERM): %v", addr, err)
		}
	}
}

// TestControl_KeychainToolIsRefused runs the keychain tool with `help`,
// which reads nothing, so an absent guard costs nothing but this failure.
func TestControl_KeychainToolIsRefused(t *testing.T) {
	err := exec.Command("/usr/bin/security", "help").Run()
	if err == nil {
		t.Fatal("GUARD ABSENT: /usr/bin/security ran")
	}
	if !errors.Is(err, syscall.EPERM) {
		t.Fatalf("/usr/bin/security failed, but not by the guard (want EPERM): %v", err)
	}
}

// TestControl_ServiceSocketsAreRefused listens on a Unix socket at a path
// each service-socket rule matches, in a directory of its own, and dials
// it: the guard must refuse it while a socket beside it, matching no rule,
// still connects. The real sockets, where they exist, must be refused too.
// The directory is under /tmp because a Unix socket path is limited to
// 104 bytes.
func TestControl_ServiceSocketsAreRefused(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "nonet-control-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	listen := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		l, err := net.Listen("unix", p)
		if err != nil {
			t.Fatalf("listening on %s: %v", p, err)
		}
		t.Cleanup(func() { _ = l.Close() })
	}
	other := filepath.Join(dir, "other.sock")
	listen(other)
	c, err := net.DialTimeout("unix", other, 2*time.Second)
	if err != nil {
		t.Fatalf("a Unix socket no rule matches was refused: %v", err)
	}
	_ = c.Close()

	refused := []string{filepath.Join(dir, ".docker", "run", "docker.sock"), filepath.Join(dir, ".s.PGSQL.5432")}
	for _, p := range refused {
		listen(p)
	}
	for _, p := range []string{"/var/run/docker.sock", "/tmp/.s.PGSQL.5432"} {
		if _, err := os.Stat(p); err == nil {
			refused = append(refused, p)
		}
	}
	for _, p := range refused {
		c, err := net.DialTimeout("unix", p, 2*time.Second)
		if err == nil {
			_ = c.Close()
			t.Fatalf("GUARD ABSENT: connected to the Unix socket %s", p)
		}
		if !errors.Is(err, syscall.EPERM) {
			t.Fatalf("dial %s failed, but not by the guard (want EPERM): %v", p, err)
		}
	}
}
