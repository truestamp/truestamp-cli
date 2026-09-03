// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// Outcome grades what an external lookup actually established, so a
// caller can report the right verification status. Whitepaper Appendix
// E.22 is unambiguous that "a skipped external check MUST NOT fail a
// proof": without this distinction an unreachable Horizon and a
// transaction that is genuinely absent from the chain are the same
// opaque error, and the verifier fails a sound proof for want of a
// network.
type Outcome int

const (
	// OutcomeOK means the lookup succeeded and agreed with the bundle.
	OutcomeOK Outcome = iota
	// OutcomeMismatch means the source answered and substantively
	// disagreed with the bundle. This is the only networked outcome
	// that may fail a step.
	OutcomeMismatch
	// OutcomeNotFound means the source answered HTTP 404: a definitive
	// "no such record here". Whether that is a failure is a per-step
	// decision (E.18 grades it against the commitment; E.21 does not).
	OutcomeNotFound
	// OutcomeUnavailable means no answer was obtained: transport
	// failure, DNS, TLS, timeout, cancellation, 429, 5xx, or any other
	// non-2xx status.
	OutcomeUnavailable
	// OutcomeMalformed means a 2xx response whose body this client
	// cannot interpret. It is an upstream property, not a bundle
	// property.
	OutcomeMalformed
	// OutcomeBadInput means the bundle's own field is unusable, so
	// there was nothing to look up.
	OutcomeBadInput
)

func (o Outcome) String() string {
	switch o {
	case OutcomeOK:
		return "ok"
	case OutcomeMismatch:
		return "mismatch"
	case OutcomeNotFound:
		return "not_found"
	case OutcomeUnavailable:
		return "unavailable"
	case OutcomeMalformed:
		return "malformed"
	case OutcomeBadInput:
		return "bad_input"
	default:
		return fmt.Sprintf("Outcome(%d)", int(o))
	}
}

// MismatchError reports that the source answered and the value it
// returned disagrees with the bundle.
type MismatchError struct {
	Field    string
	Expected string
	Got      string
}

func (e *MismatchError) Error() string {
	return fmt.Sprintf("%s mismatch: expected %s, got %s", e.Field, e.Expected, e.Got)
}

// BadInputError reports that a field taken from the bundle is unusable,
// so no lookup was attempted.
type BadInputError struct {
	Field  string
	Detail string
}

func (e *BadInputError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Detail)
}

// MalformedResponseError reports a 2xx response this client could not
// interpret: unparseable, or missing the field the comparison needs.
type MalformedResponseError struct {
	Source string
	Detail string
	Err    error
}

func (e *MalformedResponseError) Error() string {
	return fmt.Sprintf("%s response malformed: %s", e.Source, e.Detail)
}

func (e *MalformedResponseError) Unwrap() error { return e.Err }

// KeyBindingError reports that the keyring answered and does not vouch
// for the key: either it is absent, or it is published with different
// bytes. Both are substantive disagreements, so both grade as
// OutcomeMismatch.
type KeyBindingError struct {
	KeyID  string
	Reason string
}

func (e *KeyBindingError) Error() string {
	return fmt.Sprintf("key %s %s", e.KeyID, e.Reason)
}

// Classify maps any error returned by this package's lookups onto an
// Outcome, unwrapping through %w chains. The default is
// OutcomeUnavailable on purpose: an unrecognized error must skip a
// step, never fail one (E.22).
func Classify(err error) Outcome {
	if err == nil {
		return OutcomeOK
	}

	var mismatch *MismatchError
	if errors.As(err, &mismatch) {
		return OutcomeMismatch
	}
	var binding *KeyBindingError
	if errors.As(err, &binding) {
		return OutcomeMismatch
	}
	var badInput *BadInputError
	if errors.As(err, &badInput) {
		return OutcomeBadInput
	}
	var malformed *MalformedResponseError
	if errors.As(err, &malformed) {
		return OutcomeMalformed
	}
	// Our own 1MB cap cutting a well-formed body short is the same kind
	// of "we cannot interpret this" as junk bytes would be.
	if httpclient.IsTruncated(err) {
		return OutcomeMalformed
	}
	if code := httpclient.Status(err); code != 0 {
		if code == http.StatusNotFound {
			return OutcomeNotFound
		}
		return OutcomeUnavailable
	}

	// Everything else, transport failure, DNS, TLS, client timeout,
	// context cancellation, or an error shape we do not recognize,
	// means no answer was obtained.
	return OutcomeUnavailable
}
