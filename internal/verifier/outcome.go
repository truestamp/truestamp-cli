// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file runs every step of SPEC.md section 10.2 over one document, as
// `truestamp verify` does, and says how the run ended, which decides the
// command's exit code (SPEC.md section 12.2):
//
//	| Ending                   | The run                                                                 | Exit |
//	|--------------------------|-------------------------------------------------------------------------|------|
//	| a label                  | a report, labelled by VERDICT (section 10.5)                            | 0, 3, 5, 6 or 7 |
//	| runtime error            | an invalid configuration, or an input or keyring file that cannot be read; no report | 1 |
//	| fault                    | a fault caught before steps 0 to 3 passed, in VERDICT or while the report is written; no report | 2 |
//	| bare rejection           | a bare rejection other than configuration_error (section 10.2, the gate) | 4 |
//	| configuration_error      | the bare rejection of step 0                                            | 8 |
//
// A fault the run catches from ENVIRONMENT on is an internal_error row, part
// null, at the step it was caught in, followed by one row skip
// prior_failure for each later step up to EXPECTED_HASH (SPEC.md section
// 10.5), so the run's label is failed and it exits 3; at ENVIRONMENT,
// KEY_BINDING or SIGNATURE the attribution is then none, as for a failure
// there. A fault after EXPECTED_HASH, the last step that reports rows, has
// no report to hold its row (section 12.2).
//
// The steps' own checks catch the faults they can name (a value SHAPE
// should have refused, for one); [Verify] catches the rest, a panic of any
// step, so that no fault ends the command without its exit code.

package verifier

import (
	"errors"
	"fmt"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// Outcome is how one run of [Verify] ended (see the file's doc).
type Outcome struct {
	// Ending is the row of SPEC.md section 12.2 the run ended on.
	Ending specdata.Ending
	// Report is the run's report, for a run VERDICT labelled; nil
	// otherwise.
	Report *Report
	// Rejection is the bare rejection, for a run that ended at steps 0 to
	// 3; nil otherwise.
	Rejection *Rejection
	// Trace is the run's trace (SPEC.md section 10.6), for a run VERDICT
	// labelled; nil for a run that ended before DERIVE or in it, and for
	// every run with no report.
	Trace []TraceRow
	// Err says why, for a runtime error or a fault; nil otherwise.
	Err error
}

// ExitCode is the run's exit code (SPEC.md section 12.2).
func (o Outcome) ExitCode() int {
	code, ok := specdata.ExitCode(o.Ending)
	if !ok {
		panic(fmt.Sprintf("verifier: the ending %q has no exit code", o.Ending))
	}
	return code
}

// ErrFault marks a fault of the verifier that left no report (SPEC.md
// section 12.2, exit code 2).
var ErrFault = errors.New("the verifier stopped on a fault")

// step is one step of SPEC.md section 10.2 that adds rows, by name.
type step struct {
	name string
	run  func(*Run)
}

// laterSteps are steps 4 to 13, ENVIRONMENT to EXPECTED_HASH, in order.
var laterSteps = []step{
	{"ENVIRONMENT", Environment}, {"KEY_BINDING", KeyBinding}, {"SIGNATURE", Signature},
	{"SHAPE", Shape}, {"POLICY", Policy}, {"DERIVE", Derive}, {"CHAIN", Chain},
	{"SOURCES", Sources}, {"WINDOW", Window}, {"EXPECTED_HASH", ExpectedHash},
}

// gateSteps are the steps whose failure gives the attribution none
// (SPEC.md section 10.5).
var gateSteps = []string{"ENVIRONMENT", "KEY_BINDING", "SIGNATURE"}

// faultAt is called with each step's name before the step runs. It is nil
// in every build but one made with the build tag truestamp_fault (a test
// binary, fault_tag.go), where it raises the fault the build names.
var faultAt func(step string)

// Verify runs steps 0 to 14 of SPEC.md section 10.2 under cfg over the
// document read gives, which it calls after CONFIGURATION, and gives the
// run's [Outcome]. read gives the document's bytes, at most MAX_READ_OCTETS
// + 1 of them, as [strictjson.Read] does: a *strictjson.Rejection from it
// is J1's input_too_large, and any other error an input that cannot be
// read.
func Verify(cfg Config, read func() ([]byte, error)) (outcome Outcome) {
	run, rejection, err := admit(cfg, read)
	switch {
	case errors.Is(err, ErrFault):
		return Outcome{Ending: specdata.EndingFault, Err: err}
	case err != nil:
		return Outcome{Ending: specdata.EndingRuntimeError, Err: err}
	case rejection != nil && rejection.Code == specdata.ReasonConfigurationError:
		return Outcome{Ending: specdata.EndingConfigurationError, Rejection: rejection}
	case rejection != nil:
		return Outcome{Ending: specdata.EndingBareRejection, Rejection: rejection}
	}
	for _, s := range laterSteps {
		runStep(run, s)
	}
	defer func() {
		if r := recover(); r != nil {
			outcome = Outcome{Ending: specdata.EndingFault, Err: fmt.Errorf("%w at VERDICT: %v", ErrFault, r)}
		}
	}()
	fault("VERDICT")
	Verdict(run)
	report := NewReport(run)
	return Outcome{Ending: labelEndings[report.Label], Report: report, Trace: run.Trace}
}

// labelEndings are the endings of the five labels (SPEC.md section 12.2).
var labelEndings = map[string]specdata.Ending{
	LabelVerified:             specdata.EndingVerified,
	LabelInternallyConsistent: specdata.EndingInternallyConsistent,
	LabelSuppliedKeyring:      specdata.EndingSuppliedKeyring,
	LabelUnattributed:         specdata.EndingUnattributed,
	LabelFailed:               specdata.EndingFailed,
}

// admit makes steps 0 to 3, reading the document after CONFIGURATION: the
// run when each passes, a bare rejection, or an error, wrapping ErrFault
// for a fault caught in them.
func admit(cfg Config, read func() ([]byte, error)) (run *Run, rejection *Rejection, err error) {
	at := "CONFIGURATION"
	defer func() {
		if r := recover(); r != nil {
			run, rejection, err = nil, nil, fmt.Errorf("%w at %s, before steps 0 to 3 passed: %v", ErrFault, at, r)
		}
	}()
	fault(at)
	run, rejection, err = Configure(cfg)
	if rejection != nil || err != nil {
		return nil, rejection, err
	}
	data, err := read()
	var tooLarge *strictjson.Rejection
	switch {
	case errors.As(err, &tooLarge):
		return nil, fromReader(tooLarge), nil
	case err != nil:
		return nil, nil, fmt.Errorf("the proof cannot be read: %w", err)
	}
	for _, s := range []struct {
		name string
		step func() *Rejection
	}{
		{"READ", func() *Rejection { return Read(run, data) }},
		{"PARSE", func() *Rejection { return Parse(run, data) }},
		{"ENVELOPE", func() *Rejection { return Envelope(run) }},
	} {
		at = s.name
		fault(at)
		if rejection := s.step(); rejection != nil {
			return nil, rejection, nil
		}
	}
	return run, nil, nil
}

// runStep runs s over run, and turns a fault it raises into an
// internal_error row at that step, after the rows the step gave before it,
// and ends the run (see the file's doc).
func runStep(run *Run, s step) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if slices.Contains(gateSteps, s.name) {
			run.Attribution = AttributionNone
		}
		run.AddRow(NewRow(s.name, specdata.ReasonInternalError, nil, "The verifier stopped on a fault in this step.", nil))
		run.EndRun()
	}()
	fault(s.name)
	s.run(run)
}

// fault raises the fault a truestamp_fault build asks for at step.
func fault(step string) {
	if faultAt != nil {
		faultAt(step)
	}
}
