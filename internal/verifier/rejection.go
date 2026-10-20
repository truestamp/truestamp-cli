// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds a bare rejection (SPEC.md section 10.2, the gate), a
// failure at steps 0 to 3, which gives only {code, message, sub_reason},
// with no report and no verdict. The codes are internal/specdata's (SPEC.md
// section 12.1), and a rejection of steps READ and PARSE is built from the
// strictjson.Rejection the strict reader gives.

package verifier

import (
	"fmt"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// Rejection is a bare rejection. Code is one of the four codes whose status
// is bare in SPEC.md section 12.1: configuration_error, input_too_large,
// malformed_json or invalid_envelope. SubReason is set for malformed_json
// and invalid_envelope only, to one of the names section 12.1 lists for the
// code, and is "" otherwise.
type Rejection struct {
	Code      specdata.Reason
	Message   string
	SubReason specdata.SubReason
}

// Error names the code and the sub_reason, with the message.
func (r *Rejection) Error() string {
	if r.SubReason != "" {
		return fmt.Sprintf("%s (%s): %s", r.Code, r.SubReason, r.Message)
	}
	return fmt.Sprintf("%s: %s", r.Code, r.Message)
}

// NewRejection returns a bare rejection with code, message and, when the
// code has one, sub. It panics for a code that is not bare, a sub_reason
// the code does not list, or a missing one.
func NewRejection(code specdata.Reason, message string, sub specdata.SubReason) *Rejection {
	entry, ok := specdata.LookupReason(code)
	if !ok || entry.Status != specdata.StatusBare {
		panic(fmt.Sprintf("verifier: %q is not a bare code of SPEC.md section 12.1", code))
	}
	listed := entry.SubReasons
	if (len(listed) == 0 && sub != "") || (len(listed) > 0 && !slices.Contains(listed, sub)) {
		panic(fmt.Sprintf("verifier: %q is not a sub_reason of %s", sub, code))
	}
	return &Rejection{Code: code, Message: message, SubReason: sub}
}

// ConfigurationError is configuration_error, with message saying which
// check of step 0 failed.
func ConfigurationError(message string) *Rejection {
	return NewRejection(specdata.ReasonConfigurationError, message, "")
}

// InputTooLarge is input_too_large (step READ, J1).
func InputTooLarge() *Rejection {
	return NewRejection(specdata.ReasonInputTooLarge,
		fmt.Sprintf("The file is larger than %d octets, the most a verifier reads.", strictjson.MaxReadOctets), "")
}

// MalformedJSON is malformed_json with the rule the file breaks (step
// PARSE, J2 to J6).
func MalformedJSON(sub specdata.SubReason) *Rejection {
	return NewRejection(specdata.ReasonMalformedJSON, "The file is not JSON that every verifier reads alike.", sub)
}

// envelopeMessage is the message of invalid_envelope, which SPEC.md
// section 10.2, step 3, fixes. It never names a member the document holds.
const envelopeMessage = "This file is not a Truestamp proof."

// InvalidEnvelope is invalid_envelope with the first check of step
// ENVELOPE that fails (SPEC.md section 10.2, step 3).
func InvalidEnvelope(sub specdata.SubReason) *Rejection {
	return NewRejection(specdata.ReasonInvalidEnvelope, envelopeMessage, sub)
}

// fromReader returns the rejection of READ or PARSE that the strict
// reader's rejection is.
func fromReader(rejection *strictjson.Rejection) *Rejection {
	if rejection.Code == specdata.ReasonInputTooLarge {
		return InputTooLarge()
	}
	return MalformedJSON(rejection.SubReason)
}
