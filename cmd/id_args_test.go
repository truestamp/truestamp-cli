// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"
)

// uuidRefusal is the refusal of internal/ids.ValidateUUIDv7, which blocks,
// entropy and beacons get give a value with a hyphen that is no UUIDv7.
func uuidRefusal(quoted string) string {
	return "invalid UUID " + quoted + ": an id is the 36-character text of a UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)"
}

// blocks get, entropy get and beacons get read their id or hash exactly as
// typed, by the one UUIDv7 rule DetectIDType reads every other id by: a
// space, tab or line feed around a UUIDv7 or a hash is refused, never
// trimmed, and so are a UUID's braced, urn: and 32-hex forms, each quoted
// as given and before any request. Mutants killed: a strings.TrimSpace on
// the argument of blocks get, entropy get or beacons get (the padded
// values are sent, and the stand-in's 404 is "not found"); ValidateUUIDv7
// reading a UUID by gofrs FromString, which takes the braced and urn:
// forms (those values are sent).
func TestCLI_RecordIDsAreReadExactlyAsTyped(t *testing.T) {
	const id = testBeaconID
	const hex32 = "019db702b08c73dca7cd2c5e011f1dad"
	url, requests := countingServer(t)
	for _, noun := range []string{"block", "entropy", "beacon"} {
		group := map[string]string{"block": "blocks", "entropy": "entropy", "beacon": "beacons"}[noun]
		neither := " is neither a UUIDv7 id nor a 64-hex-char " + noun + " hash"
		for typed, want := range map[string]string{
			" " + id:                               uuidRefusal(`" ` + id + `"`),
			id + " ":                               uuidRefusal(`"` + id + ` "`),
			id + "\n":                              uuidRefusal(`"` + id + `\n"`),
			"\t" + id:                              uuidRefusal(`"\t` + id + `"`),
			"{" + id + "}":                         uuidRefusal(`"{` + id + `}"`),
			"urn:uuid:" + id:                       uuidRefusal(`"urn:uuid:` + id + `"`),
			hex32:                                  `"` + hex32 + `"` + neither,
			testBeaconHash + "\n":                  `"` + testBeaconHash + `\n"` + neither,
			" " + testBeaconHash:                   `" ` + testBeaconHash + `"` + neither,
			strings.ToUpper(hex32):                 `"` + strings.ToUpper(hex32) + `"` + neither,
			"550e8400-e29b-41d4-a716-446655440000": uuidRefusal(`"550e8400-e29b-41d4-a716-446655440000"`),
			id[:9] + "\t" + id[9:]:                 uuidRefusal(`"` + id[:9] + `\t` + id[9:] + `"`),
		} {
			stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", group, "get", typed)
			if exit != 1 || stdout != "" || stderr != want+"\n" {
				t.Errorf("%s get %q: exit %d, stdout %q, stderr %q; want exit 1 and %q", group, typed, exit, stdout, stderr, want)
			}
		}
	}
	if n := requests(); n != 0 {
		t.Errorf("%d requests sent, want none", n)
	}
}

// keys get reads its key id exactly as typed, and teams use and teams get
// their team id: a space, tab or line feed around a key id or a team's
// UUIDv7 is refused, never trimmed, and so is a value of another form, a
// UUID's braced, urn: and 32-hex forms among them, each quoted as given
// and before any request. Mutants killed: a strings.TrimSpace on the
// argument of keys get or teams use (the padded values are sent: the
// keyring is read, or the team is); the key id check removed from keys
// get, or the UUIDv7 check from teams use or teams get (the other forms
// are sent).
func TestCLI_KeyAndTeamIDsAreReadExactlyAsTyped(t *testing.T) {
	const kid = "96b1cd2f"
	const team = "019dbd00-0000-7000-8000-000000000000"
	url, requests := countingServer(t)
	keyRefusal := func(quoted string) string {
		return quoted + " is no key id: keys get takes an 8-hex key id (e.g. 96b1cd2f)"
	}
	teamRefusal := func(quoted string) string {
		return quoted + " is no team id: teams use takes a team's UUIDv7 id (e.g. 019dbd00-0000-7000-8000-000000000000)"
	}
	teamGetRefusal := func(quoted string) string {
		return quoted + " is no team id: teams get takes a team's UUIDv7 id (e.g. 019dbd00-0000-7000-8000-000000000000)"
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"keys", "get", " " + kid}, keyRefusal(`" ` + kid + `"`)},
		{[]string{"keys", "get", kid + " "}, keyRefusal(`"` + kid + ` "`)},
		{[]string{"keys", "get", kid + "\n"}, keyRefusal(`"` + kid + `\n"`)},
		{[]string{"keys", "get", "\t" + kid}, keyRefusal(`"\t` + kid + `"`)},
		{[]string{"keys", "get", "{" + kid + "}"}, keyRefusal(`"{` + kid + `}"`)},
		{[]string{"keys", "get", "urn:uuid:" + team}, keyRefusal(`"urn:uuid:` + team + `"`)},
		{[]string{"keys", "get", "019dbd00000070008000000000000000"}, keyRefusal(`"019dbd00000070008000000000000000"`)},
		{[]string{"keys", "get", "96b1cd2"}, keyRefusal(`"96b1cd2"`)},
		{[]string{"teams", "use", " " + team}, teamRefusal(`" ` + team + `"`)},
		{[]string{"teams", "use", team + " "}, teamRefusal(`"` + team + ` "`)},
		{[]string{"teams", "use", team + "\n"}, teamRefusal(`"` + team + `\n"`)},
		{[]string{"teams", "use", " "}, teamRefusal(`" "`)},
		{[]string{"teams", "use", "{" + team + "}"}, teamRefusal(`"{` + team + `}"`)},
		{[]string{"teams", "use", "urn:uuid:" + team}, teamRefusal(`"urn:uuid:` + team + `"`)},
		{[]string{"teams", "use", "019dbd00000070008000000000000000"}, teamRefusal(`"019dbd00000070008000000000000000"`)},
		{[]string{"teams", "get", " " + team}, teamGetRefusal(`" ` + team + `"`)},
		{[]string{"teams", "get", team + "\n"}, teamGetRefusal(`"` + team + `\n"`)},
		{[]string{"teams", "get", "{" + team + "}"}, teamGetRefusal(`"{` + team + `}"`)},
		{[]string{"teams", "get", "urn:uuid:" + team}, teamGetRefusal(`"urn:uuid:` + team + `"`)},
		{[]string{"teams", "get", "019dbd00000070008000000000000000"}, teamGetRefusal(`"019dbd00000070008000000000000000"`)},
	} {
		stdout, stderr, exit := runCLI(t, append([]string{"--base-url", url, "--api-key", "k"}, tc.args...)...)
		if exit != 1 || stdout != "" || stderr != tc.want+"\n" {
			t.Errorf("%q: exit %d, stdout %q, stderr %q; want exit 1 and %q", tc.args, exit, stdout, stderr, tc.want)
		}
	}
	if n := requests(); n != 0 {
		t.Errorf("%d requests sent, want none", n)
	}
	// The forms each command takes are still sent: a key id in either case
	// reads the keyring, and a team's UUIDv7 reads the team, one request
	// each.
	for _, args := range [][]string{{"keys", "get", kid}, {"keys", "get", strings.ToUpper(kid)}, {"teams", "use", team}, {"teams", "get", team}} {
		url, requests := countingServer(t)
		if _, stderr, _ := runCLI(t, append([]string{"--base-url", url, "--api-key", "k"}, args...)...); requests() != 1 {
			t.Errorf("%q: %d requests sent, want 1\n%s", args, requests(), stderr)
		}
	}
}
