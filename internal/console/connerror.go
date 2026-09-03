// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"errors"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/auth"
)

// connErrorKind classifies common WebSocket connect-time failures so
// the UI can render a short, friendly status pill plus a longer
// explanation with concrete next steps. We keep the categories small
// and additive, anything that doesn't match falls through to
// connErrorUnknown which still shows the raw error for debugging.
type connErrorKind int

const (
	connErrorUnknown connErrorKind = iota
	connErrorBadURL
	connErrorDNS
	connErrorRefused
	connErrorTimeout
	connErrorTLS
	connErrorAuth
	connErrorNotFound
	connErrorServerError
	connErrorProtocol
	connErrorDecodeWelcome
)

// classifyConnError pattern-matches the error chain from
// wschannel.Connect to a kind. Substring matching against err.Error()
// is intentional: Go's net package and coder/websocket both expose
// stable English fragments ("no such host", "connection refused",
// "i/o timeout", "expected handshake response status code 101 but
// got NNN") as part of their public contract.
func classifyConnError(err error) connErrorKind {
	if err == nil {
		return connErrorUnknown
	}
	// A dead or absent OAuth session is an auth failure, not a generic
	// transport error, route it to the auth kind so the user is told to
	// re-authenticate rather than to check their network.
	if errors.Is(err, auth.ErrSessionExpired) || errors.Is(err, auth.ErrNoCredentials) {
		return connErrorAuth
	}
	s := err.Error()

	// Order matters: the more specific patterns first.
	switch {
	case strings.Contains(s, "decode welcome"):
		return connErrorDecodeWelcome
	case strings.Contains(s, "parse url"):
		return connErrorBadURL
	case strings.Contains(s, "no such host"):
		return connErrorDNS
	case strings.Contains(s, "connection refused"):
		return connErrorRefused
	case strings.Contains(s, "i/o timeout"),
		strings.Contains(s, "context deadline exceeded"):
		return connErrorTimeout
	case strings.Contains(s, "tls:"),
		strings.Contains(s, "x509:"),
		strings.Contains(s, "certificate"):
		return connErrorTLS
	case strings.Contains(s, "got 401"),
		strings.Contains(s, "got 403"):
		return connErrorAuth
	case strings.Contains(s, "got 404"):
		return connErrorNotFound
	case strings.Contains(s, "got 500"),
		strings.Contains(s, "got 502"),
		strings.Contains(s, "got 503"),
		strings.Contains(s, "got 504"):
		return connErrorServerError
	case strings.Contains(s, "expected handshake response status code 101"):
		return connErrorProtocol
	}
	return connErrorUnknown
}

// shortStatus is the header-pill label. Capped at 24 chars so the
// status pill stays compact even on narrow terminals.
func (k connErrorKind) shortStatus() string {
	switch k {
	case connErrorBadURL:
		return "invalid server URL"
	case connErrorDNS:
		return "host not found"
	case connErrorRefused:
		return "server unreachable"
	case connErrorTimeout:
		return "connection timed out"
	case connErrorTLS:
		return "TLS handshake failed"
	case connErrorAuth:
		return "authentication failed"
	case connErrorNotFound:
		return "console not found"
	case connErrorServerError:
		return "server error"
	case connErrorProtocol:
		return "unexpected response"
	case connErrorDecodeWelcome:
		return "bad welcome reply"
	}
	return "connection failed"
}

// title is the one-line headline for the error section on the
// Connection pane. More descriptive than shortStatus, but still a
// single sentence.
func (k connErrorKind) title() string {
	switch k {
	case connErrorBadURL:
		return "The configured server URL could not be parsed."
	case connErrorDNS:
		return "Could not resolve the server hostname."
	case connErrorRefused:
		return "The server refused the connection."
	case connErrorTimeout:
		return "The connection attempt timed out."
	case connErrorTLS:
		return "TLS / certificate verification failed."
	case connErrorAuth:
		return "The server rejected your credentials."
	case connErrorNotFound:
		return "The server is reachable but does not expose a console endpoint at this URL."
	case connErrorServerError:
		return "The server returned an error response during the upgrade handshake."
	case connErrorProtocol:
		return "The server did not complete a WebSocket upgrade."
	case connErrorDecodeWelcome:
		return "The server connected but sent a welcome reply we could not decode."
	}
	return "The console could not connect to the server."
}

// hints returns one or more short "what to try" lines tailored to the
// failure mode. They are written for everyday users, no jargon
// (no "DNS", "TLS handshake", "WebSocket upgrade", "reverse proxy",
// "preview server") and no developer-only commands like `task serve`.
// Each bullet is a concrete action a non-technical user can take.
//
// Action commands kept on purpose: `truestamp auth login` (the
// canonical way to refresh credentials, surfaced in our CLI help)
// and `truestamp config show` (the canonical way to inspect the
// active config from the command line).
func (k connErrorKind) hints() []string {
	switch k {
	case connErrorBadURL:
		return []string{
			"The server address in your settings doesn't look like a valid web address.",
			"Run `truestamp config show` to see the current address.",
			"A normal address starts with `https://` (for example `https://www.truestamp.com`).",
		}
	case connErrorDNS:
		return []string{
			"Check that you're connected to the internet.",
			"Try opening the server address in a web browser to see if it loads.",
			"If you changed the server address recently, double-check it for typos.",
		}
	case connErrorRefused:
		return []string{
			"Check that you're connected to the internet.",
			"Try opening the server address in a web browser, if it doesn't load there either, the server is offline.",
			"Make sure your settings point at the official Truestamp server (`https://www.truestamp.com`) unless someone gave you a different address to use.",
		}
	case connErrorTimeout:
		return []string{
			"Check that you're connected to the internet.",
			"If you're on a work or school network, ask whether it blocks outgoing connections, try again on a different network (for example, your home Wi-Fi or phone hotspot).",
			"Try again in a moment; the server may just be slow to respond.",
		}
	case connErrorTLS:
		return []string{
			"Check that the date and time on your computer are correct, a wrong clock breaks secure connections.",
			"If your network requires a VPN or special security software, make sure it's running.",
			"Try again in a few minutes, or contact support if it keeps happening.",
		}
	case connErrorAuth:
		return []string{
			"Your sign-in has expired or isn't valid for this server.",
			"Run `truestamp auth login` to sign in again.",
			"If someone gave you an API key to set manually, double-check it was copied correctly.",
		}
	case connErrorNotFound:
		return []string{
			"The address responded, but it doesn't look like a Truestamp server.",
			"Run `truestamp config show` and confirm the address is set to `https://www.truestamp.com` (unless you were told to use a different one).",
		}
	case connErrorServerError:
		return []string{
			"The Truestamp server is having a problem on its end, this is not something you can fix on your computer.",
			"Wait a few minutes and try again.",
			"If it doesn't recover, contact support and include the error shown below.",
		}
	case connErrorProtocol:
		return []string{
			"Something on the network is blocking the connection, often a work or school firewall, or a VPN.",
			"Try the same command on a different network (for example, your home Wi-Fi or phone hotspot) to see if that helps.",
		}
	case connErrorDecodeWelcome:
		return []string{
			"The server replied with something this version of truestamp doesn't understand.",
			"Make sure you're running the latest version (run `truestamp upgrade` to update).",
			"If you're already up to date, contact support and include the error shown below.",
		}
	}
	return []string{
		"Check that you're connected to the internet.",
		"Run `truestamp auth login` to refresh your sign-in.",
		"If the problem keeps happening, contact support and include the error shown below.",
	}
}
