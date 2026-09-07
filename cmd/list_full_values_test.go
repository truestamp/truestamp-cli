// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/truestamp/truestamp-cli/internal/beacons"
	"github.com/truestamp/truestamp-cli/internal/blocks"
	"github.com/truestamp/truestamp-cli/internal/entropy"
)

// Hashes and ids are rendered whole, everywhere, in every output mode.
//
// A listing exists to be copied, pasted, grepped and piped into the next
// command. An abbreviated hash serves none of those, and the abbreviation
// is silent: nothing in the output says a value was shortened, so a reader
// cannot tell a truncated hash from a short one. `entropy list` and
// `blocks list` both shipped truncated hashes behind a comment arguing the
// full value was available elsewhere; that argument is rejected here.
//
// The one place an abbreviated hex value is correct is the verify report's
// offline skip rows, where `internal/verify.short()` reproduces the
// reference verifier (whitepaper/verify_proof.exs) byte for byte and the
// conformance fixtures pin it. That is conformance, not display, and it
// lives in a different package on purpose.
//
// This test renders each list from a fixture and asserts every hash and id
// it was given survives whole, and that no elision marker reaches the
// output.
func TestListRenderers_RenderHashesAndIDsWhole(t *testing.T) {
	t.Parallel()

	const (
		hashA = "4142459a2859a57a5e5d6540579b977215d6329110adc0a890fdce6f05054f2d"
		hashB = "e36f824dea9508d5dca570c77edc06eb4bc830b413b3db4e4d983ce3f85e9d78"
		hashC = "d175d1ef04595ffbea81cf9864780889615a73bdb0184c6efa43de184f5c24a2"
		idA   = "01a07342-b988-7b3a-a707-8b1653bf175c"
		idB   = "01a0680e-42b6-712a-9137-5b59dc891f19"
	)

	render := func(t *testing.T, fn func(*cobra.Command, *bytes.Buffer) error) string {
		t.Helper()
		var out bytes.Buffer
		c := &cobra.Command{}
		addRecordOutputFlags(c)
		c.SetOut(&out)
		c.SetErr(&bytes.Buffer{})
		if err := fn(c, &out); err != nil {
			t.Fatalf("render: %v", err)
		}
		return out.String()
	}

	cases := []struct {
		name string
		want []string
		out  func(t *testing.T) string
	}{
		{
			name: "entropy list",
			want: []string{hashA, idA},
			out: func(t *testing.T) string {
				return render(t, func(c *cobra.Command, _ *bytes.Buffer) error {
					return renderObservationList(c, []entropy.Observation{{
						ID: idA, Source: "entropy_stellar", State: "committed",
						EntropyHash: hashA, SourcePublishedAt: "2026-09-03T18:10:02Z",
					}}, listPage{})
				})
			},
		},
		{
			name: "blocks list",
			want: []string{hashB, idB},
			out: func(t *testing.T) string {
				return render(t, func(c *cobra.Command, _ *bytes.Buffer) error {
					return renderBlockList(c, []blocks.Block{{
						ID: idB, State: "committed", BlockHash: hashB, MerkleRoot: hashC,
					}}, listPage{})
				})
			},
		},
		{
			name: "beacons list",
			want: []string{hashB, idB},
			out: func(t *testing.T) string {
				var out bytes.Buffer
				renderBeaconList(&out, []beacons.Beacon{{
					ID: idB, Hash: hashB, Timestamp: "2026-09-03T18:10:02.123456Z",
				}}, listPage{})
				return out.String()
			},
		},
	}

	// Any run of hex followed by an elision marker, in either spelling.
	elided := regexp.MustCompile(`[0-9a-f]{4,}\s*(…|\.\.\.)`)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.out(t)
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("%s must render %q whole, got:\n%s", tc.name, w, got)
				}
			}
			if m := elided.FindString(got); m != "" {
				t.Errorf("%s elided a value (%q); hashes and ids render whole, got:\n%s", tc.name, m, got)
			}
		})
	}
}
