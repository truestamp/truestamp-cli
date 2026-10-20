// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Tests of step READ (SPEC.md section 10.2, step 1), J1 of section 10.1.

package strictjson

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// requireRejection fails the test unless err is a *Rejection with code and
// sub, and returns it.
func requireRejection(t *testing.T, err error, code specdata.Reason, sub specdata.SubReason) *Rejection {
	t.Helper()
	var rejection *Rejection
	if !errors.As(err, &rejection) {
		t.Fatalf("error = %v, want the rejection %s (%s)", err, code, sub)
	}
	if rejection.Code != code || rejection.SubReason != sub {
		t.Fatalf("rejection = %s (%s), want %s (%s)", rejection.Code, rejection.SubReason, code, sub)
	}
	return rejection
}

// The limits are SPEC.md section 9's, from internal/specdata. Mutant
// killed: MaxReadOctets set to its own literal off by one.
func TestLimitsAreSpecData(t *testing.T) {
	if MaxReadOctets != 262144 || MaxDepth != 32 || MaxInteger != 9007199254740991 {
		t.Fatalf("limits = %d, %d, %d", MaxReadOctets, MaxDepth, MaxInteger)
	}
	constants := specdata.Constants()
	if constants["MAX_READ_OCTETS"] != int64(MaxReadOctets) || constants["MAX_DEPTH"] != int64(MaxDepth) ||
		constants["MAX_INTEGER"] != int64(MaxInteger) {
		t.Fatalf("the limits differ from specdata: %v", constants)
	}
}

// J1 on bytes: MAX_READ_OCTETS octets pass, one more is input_too_large.
// Mutant killed: CheckSize comparing with >= in place of >.
func TestCheckSize_Boundary(t *testing.T) {
	if err := CheckSize(make([]byte, MaxReadOctets)); err != nil {
		t.Fatalf("CheckSize(MAX_READ_OCTETS octets) = %v, want nil", err)
	}
	r := requireRejection(t, CheckSize(make([]byte, MaxReadOctets+1)), specdata.ReasonInputTooLarge, "")
	if r.Error() != "input_too_large: the input is over 262144 octets" {
		t.Fatalf("message = %q", r.Error())
	}
}

// long is a reader of 4 x MAX_READ_OCTETS spaces, counting what it gives.
type long struct{ given int }

func (l *long) Read(p []byte) (int, error) {
	if l.given >= 4*MaxReadOctets {
		return 0, io.EOF
	}
	n := min(len(p), 4*MaxReadOctets-l.given)
	for i := range n {
		p[i] = ' '
	}
	l.given += n
	return n, nil
}

// Read stops after MAX_READ_OCTETS + 1 octets, which decides J1, however
// long the input. Mutant killed: Read without its io.LimitReader.
func TestRead_ReadsAtMostOneOctetPastTheLimit(t *testing.T) {
	source := &long{}
	_, err := Read(source)
	requireRejection(t, err, specdata.ReasonInputTooLarge, "")
	if source.given > MaxReadOctets+1 {
		t.Fatalf("Read took %d octets from the source", source.given)
	}
}

// Read gives back exactly MAX_READ_OCTETS octets, and an empty input as
// empty. Mutant killed: Read returning nil data on success.
func TestRead_AtTheLimit(t *testing.T) {
	want := bytes.Repeat([]byte{' '}, MaxReadOctets-2)
	want = append([]byte{'['}, append(want, ']')...)
	got, err := Read(bytes.NewReader(want))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("Read = %d octets, %v; want %d octets", len(got), err, len(want))
	}
	if got, err := Read(bytes.NewReader(nil)); err != nil || len(got) != 0 {
		t.Fatalf("Read(empty) = %q, %v", got, err)
	}
}

type failingReader struct{}

var errBroken = errors.New("broken pipe")

func (failingReader) Read([]byte) (int, error) { return 0, errBroken }

// A read that fails is ErrUnreadable, never a rejection, and keeps its
// cause. Mutant killed: Read returning the bare cause without ErrUnreadable.
func TestRead_FailureIsUnreadable(t *testing.T) {
	_, err := Read(io.MultiReader(bytes.NewReader([]byte("{")), failingReader{}))
	var rejection *Rejection
	if !errors.Is(err, ErrUnreadable) || !errors.Is(err, errBroken) || errors.As(err, &rejection) {
		t.Fatalf("error = %v, want ErrUnreadable wrapping the cause", err)
	}
}

// ReadFile makes J1 over a file: the boundary on both sides, and a file
// that cannot be opened, or is a directory, is ErrUnreadable. Mutant killed:
// ReadFile reading the whole file with os.ReadFile and no J1.
func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	atLimit := filepath.Join(dir, "at-limit.json")
	overLimit := filepath.Join(dir, "over-limit.json")
	if err := os.WriteFile(atLimit, bytes.Repeat([]byte{' '}, MaxReadOctets), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overLimit, bytes.Repeat([]byte{' '}, MaxReadOctets+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, err := ReadFile(atLimit); err != nil || len(data) != MaxReadOctets {
		t.Fatalf("ReadFile(at the limit) = %d octets, %v", len(data), err)
	}
	_, err := ReadFile(overLimit)
	requireRejection(t, err, specdata.ReasonInputTooLarge, "")

	for _, name := range []string{filepath.Join(dir, "missing.json"), dir} {
		_, err := ReadFile(name)
		var rejection *Rejection
		if !errors.Is(err, ErrUnreadable) || errors.As(err, &rejection) {
			t.Fatalf("ReadFile(%s) error = %v, want ErrUnreadable", name, err)
		}
	}
}

// Decode makes J1 before J2: a text over the limit is input_too_large even
// when it is also malformed. Mutant killed: Decode calling Parse alone.
func TestDecode_StepOrder(t *testing.T) {
	over := append([]byte{0xff}, bytes.Repeat([]byte{' '}, MaxReadOctets)...)
	_, err := Decode(over)
	requireRejection(t, err, specdata.ReasonInputTooLarge, "")
	_, err = Decode(over[:MaxReadOctets])
	requireRejection(t, err, specdata.ReasonMalformedJSON, specdata.SubReasonInvalidUTF8)
	v, err := Decode([]byte(`{"a":[1]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := v.(map[string]any)["a"].([]any)[0]; got != Number(1) {
		t.Fatalf("Decode value = %#v", got)
	}
}
