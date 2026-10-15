// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file is the CLI's online keyring read, which no reference module
// ports, since the reference verifier is offline (SPEC.md section 8.3
// (External verifiers)): the read of step 0 (CONFIGURATION, SPEC.md
// section 10.2) of the keyring that governs an online run, from
// www.truestamp.com under E_v prod with no supplied keyring, or from the
// origin an explicit --base-url names. Its rules are SPEC.md's:
//
//   - the keyring read replaces the compiled pin, and a read that does not
//     give a keyring passing the load checks, for any reason, binds no key
//     (section 8.3); this file gives the bytes or the reason, and
//     [Configure] loads them;
//   - the read follows no redirect and ends within one deadline,
//     [KeyringReadDeadline], over the whole read, through
//     internal/httpclient's SingleRead, as the Truestamp application's
//     verification reads do (Truestamp.SourceCache.ReadBounds,
//     lib/truestamp/source_cache/read_bounds.ex in truestamp-v2): a 3xx is
//     an HTTP error, and so is any status but 200 OK;
//   - the body is read to at most MAX_READ_OCTETS octets (section 10.1,
//     J1), so a larger answer is a failed read unread past the bound;
//   - the URL's host is a valid DNS name or an IP address, so a host such
//     as www.truestamp.com.. (two trailing dots), which names no DNS name
//     and which [Options.Config] reads as another origin than
//     www.truestamp.com, is a failed read with no request made.

package verifier

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// KeyringReadDeadline bounds each online keyring read, the whole read, as
// the Truestamp application bounds each of its verification reads.
const KeyringReadDeadline = 9 * time.Second

// DefaultKeyringReader returns the reader of an online run's keyring over
// HTTP, through the default transport, each read within
// [KeyringReadDeadline].
func DefaultKeyringReader() KeyringReader {
	return NewKeyringReader(newSingleRead(nil, KeyringReadDeadline))
}

// newSingleRead builds the reads of [DefaultKeyringReader] and
// [DefaultIdentityReader]. It is httpclient.NewSingleRead, and a seam a
// test replaces to see the transport and deadline the reads are built
// with.
var newSingleRead = httpclient.NewSingleRead

// NewKeyringReader returns the reader of an online run's keyring through
// reads: a URL whose host is not a valid DNS name or IP address, or whose
// scheme is not http or https, is refused before any request; otherwise
// one GET with no redirect followed, within the reader's deadline, whose
// 200 OK body of at most MAX_READ_OCTETS octets is the keyring's bytes as
// read.
func NewKeyringReader(reads *httpclient.SingleRead) KeyringReader {
	return func(raw string) (KeyringRead, error) {
		if err := checkKeyringURL(raw); err != nil {
			return KeyringRead{}, err
		}
		data, err := reads.Get(context.Background(), raw, strictjson.MaxReadOctets)
		if err != nil {
			return KeyringRead{}, err
		}
		return KeyringRead{Bytes: data}, nil
	}
}

// checkKeyringURL refuses a keyring URL a read must not be made to: one
// that does not parse, whose scheme is not http or https, that carries
// user information, or whose host is neither an IP address nor a valid DNS
// name.
func checkKeyringURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("the keyring URL does not parse: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("the keyring URL's scheme %q is not http or https", u.Scheme)
	}
	if u.User != nil {
		return fmt.Errorf("the keyring URL carries user information")
	}
	if host := u.Hostname(); net.ParseIP(host) == nil && !validDNSName(host) {
		return fmt.Errorf("the keyring URL's host %q is not a valid DNS name", host)
	}
	return nil
}

// validDNSName reports whether name is a DNS host name (RFC 1123 section
// 2.1): labels of 1 to 63 letters, digits and hyphens, none beginning or
// ending with a hyphen, at most 253 octets in all, with at most one
// trailing dot, which names the root. So www.truestamp.com. is one, and
// www.truestamp.com.., whose last label is empty, is not.
func validDNSName(name string) bool {
	name = strings.TrimSuffix(name, ".")
	if name == "" || len(name) > 253 {
		return false
	}
	for _, label := range strings.Split(name, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range []byte(label) {
			if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
