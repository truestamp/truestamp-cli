// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package teams

import "testing"

// FuzzParseMemberships exercises the JSON:API memberships+included
// parser against garbage bytes. Must never panic and must reject
// malformed responses with an ordinary error. Anchors the teams
// package to the repo's fuzz coverage requirement (CLAUDE.md "Fuzz
// coverage on every parser").
func FuzzParseMemberships(f *testing.F) {
	f.Add([]byte(jsonAPIMembershipsBody))
	f.Add([]byte(`{"data":[]}`))
	f.Add([]byte(`{"data":[],"included":[]}`))
	f.Add([]byte(`{"data":[{"type":"other","id":"x"}]}`))
	f.Add([]byte(`{"data":[{"type":"membership","id":"x","attributes":null}]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[`))
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = parseMembershipsWithIncluded(data)
	})
}

// FuzzParseTeam exercises the single-team JSON:API parser.
func FuzzParseTeam(f *testing.F) {
	f.Add([]byte(jsonAPISingleTeamBody))
	f.Add([]byte(`{"data":{"type":"team","id":"x","attributes":{}}}`))
	f.Add([]byte(`{"data":{}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[`))
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = parseTeam(data)
	})
}

// FuzzFormatRole confirms the role label helper never panics on
// arbitrary input. It's a pure switch but treating it as a parser
// keeps the convention uniform.
func FuzzFormatRole(f *testing.F) {
	f.Add("team_owner")
	f.Add("")
	f.Add("unknown")
	f.Fuzz(func(_ *testing.T, s string) {
		_ = FormatRole(s)
	})
}
