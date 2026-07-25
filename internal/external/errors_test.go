// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Outcome
	}{
		{"nil", nil, OutcomeOK},
		{"mismatch", &MismatchError{Field: "memo", Expected: "aa", Got: "bb"}, OutcomeMismatch},
		{"key binding", &KeyBindingError{KeyID: "4ceefa4a", Reason: "not found in keyring"}, OutcomeMismatch},
		{"bad input", &BadInputError{Field: "transaction hash", Detail: "empty"}, OutcomeBadInput},
		{"malformed", &MalformedResponseError{Source: "Horizon", Detail: "not JSON"}, OutcomeMalformed},
		{"truncated body", &httpclient.TruncatedError{Limit: 10}, OutcomeMalformed},
		{"404", &httpclient.StatusError{StatusCode: 404}, OutcomeNotFound},
		{"429", &httpclient.StatusError{StatusCode: 429}, OutcomeUnavailable},
		{"500", &httpclient.StatusError{StatusCode: 500}, OutcomeUnavailable},
		{"503", &httpclient.StatusError{StatusCode: 503}, OutcomeUnavailable},
		{"418", &httpclient.StatusError{StatusCode: 418}, OutcomeUnavailable},
		{"transport", &httpclient.TransportError{Err: errors.New("dial tcp: connection refused")}, OutcomeUnavailable},
		{"timeout", &httpclient.TransportError{Err: context.DeadlineExceeded}, OutcomeUnavailable},
		{"unknown bare error", errors.New("something else entirely"), OutcomeUnavailable},

		// Every fetcher wraps its transport failure with %w, so Classify
		// has to unwrap rather than type-assert.
		{"wrapped 404", fmt.Errorf("fetching Stellar transaction: %w", &httpclient.StatusError{StatusCode: 404}), OutcomeNotFound},
		{"wrapped transport", fmt.Errorf("fetching NIST pulse: %w", &httpclient.TransportError{Err: errors.New("no route to host")}), OutcomeUnavailable},
		{"wrapped malformed", fmt.Errorf("outer: %w", &MalformedResponseError{Source: "keyring", Detail: "junk"}), OutcomeMalformed},
		{"doubly wrapped mismatch", fmt.Errorf("a: %w", fmt.Errorf("b: %w", &MismatchError{Field: "ledger"})), OutcomeMismatch},

		// The friendly keyring wrapper must not hide the typing beneath it.
		{"keyring wrapper over 503", &keyringNetError{
			friendly: "could not reach the keyring server",
			inner:    &httpclient.StatusError{StatusCode: 503},
		}, OutcomeUnavailable},
		{"keyring wrapper over 404", &keyringNetError{
			friendly: "could not reach the keyring server",
			inner:    &httpclient.StatusError{StatusCode: 404},
		}, OutcomeNotFound},
		{"keyring wrapper over transport", &keyringNetError{
			friendly: "could not connect to the keyring server",
			inner:    &httpclient.TransportError{Err: &net.DNSError{Err: "no such host"}},
		}, OutcomeUnavailable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.err); got != tc.want {
				t.Errorf("Classify(%v) = %s, want %s", tc.err, got, tc.want)
			}
		})
	}
}

// A non-nil error must never grade OutcomeOK: that is the invariant that
// keeps an unrecognized failure from being read as a successful lookup.
func TestClassify_NonNilNeverOK(t *testing.T) {
	for _, err := range []error{
		errors.New(""),
		errors.New("x"),
		fmt.Errorf("wrapped: %w", errors.New("y")),
		&keyringNetError{friendly: "z", inner: errors.New("z")},
	} {
		if got := Classify(err); got == OutcomeOK {
			t.Errorf("Classify(%q) returned OutcomeOK for a non-nil error", err)
		}
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{&MismatchError{Field: "memo", Expected: "aa", Got: "bb"}, "memo mismatch: expected aa, got bb"},
		{&BadInputError{Field: "ledger sequence", Detail: "0 is not a valid sequence"}, "ledger sequence: 0 is not a valid sequence"},
		{&MalformedResponseError{Source: "Horizon", Detail: "not JSON"}, "Horizon response malformed: not JSON"},
		{&KeyBindingError{KeyID: "4ceefa4a", Reason: "not found in keyring"}, "key 4ceefa4a not found in keyring"},
	}
	for _, tc := range tests {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
	}
}

func TestMalformedResponseError_Unwrap(t *testing.T) {
	inner := errors.New("boom")
	err := &MalformedResponseError{Source: "keyring", Detail: "junk", Err: inner}
	if !errors.Is(err, inner) {
		t.Error("MalformedResponseError should unwrap to its cause")
	}
}

func TestOutcomeString(t *testing.T) {
	want := map[Outcome]string{
		OutcomeOK:          "ok",
		OutcomeMismatch:    "mismatch",
		OutcomeNotFound:    "not_found",
		OutcomeUnavailable: "unavailable",
		OutcomeMalformed:   "malformed",
		OutcomeBadInput:    "bad_input",
	}
	for o, s := range want {
		if o.String() != s {
			t.Errorf("Outcome(%d).String() = %q, want %q", int(o), o.String(), s)
		}
	}
}

// E.18/E.5: "public" selects the public Horizon instance and everything
// else — including an absent net — falls through to the testnet
// instance, with no error branch.
func TestHorizonEndpointResolution(t *testing.T) {
	origPublic, origTestnet := HorizonPublicURL, HorizonTestnetURL
	HorizonPublicURL = "https://public.example"
	HorizonTestnetURL = "https://testnet.example"
	t.Cleanup(func() { HorizonPublicURL, HorizonTestnetURL = origPublic, origTestnet })

	tests := []struct {
		network   string
		wantURL   string
		defaulted bool
		label     string
	}{
		{"public", "https://public.example", false, "public"},
		{"testnet", "https://testnet.example", false, "testnet"},
		{"", "https://testnet.example", true, "testnet"},
		{"futurenet", "https://testnet.example", true, "testnet"},
		{"PUBLIC", "https://testnet.example", true, "testnet"},
	}
	for _, tc := range tests {
		t.Run(tc.network, func(t *testing.T) {
			if got := horizonBaseURL(tc.network); got != tc.wantURL {
				t.Errorf("horizonBaseURL(%q) = %q, want %q", tc.network, got, tc.wantURL)
			}
			if got := IsDefaultedNetwork(tc.network); got != tc.defaulted {
				t.Errorf("IsDefaultedNetwork(%q) = %v, want %v", tc.network, got, tc.defaulted)
			}
			if got := NetworkLabel(tc.network); got != tc.label {
				t.Errorf("NetworkLabel(%q) = %q, want %q", tc.network, got, tc.label)
			}
		})
	}
}
