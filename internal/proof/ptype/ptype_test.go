// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package ptype

import "testing"

// TestRegistryFrozen anchors the exact integer values assigned to each code.
// These values are a long-lived wire contract — renumbering any of them
// breaks every previously-issued proof bundle, so this test is intentionally
// a spec-lock. A change here must be accompanied by a version-bump
// conversation with downstream verifier implementers.
func TestRegistryFrozen(t *testing.T) {
	cases := []struct {
		code Code
		want uint16
	}{
		{Block, 10},
		{Beacon, 11},
		{Item, 20},
		{EntropyNIST, 30},
		{EntropyStellar, 31},
		{EntropyBitcoin, 32},
		{CommitmentStellar, 40},
		{CommitmentBitcoin, 41},
	}
	for _, c := range cases {
		if uint16(c.code) != c.want {
			t.Errorf("code %q = %d, want %d", Name(c.code), uint16(c.code), c.want)
		}
	}
}

func TestIsValidSubject(t *testing.T) {
	valid := []Code{Block, Beacon, Item, EntropyNIST, EntropyStellar, EntropyBitcoin}
	for _, c := range valid {
		if !IsValidSubject(c) {
			t.Errorf("IsValidSubject(%d) = false, want true", c)
		}
	}
	invalid := []Code{0, 1, 12, 19, 22, 29, 33, 40, 41, 50, 9999}
	for _, c := range invalid {
		if IsValidSubject(c) {
			t.Errorf("IsValidSubject(%d) = true, want false", c)
		}
	}
}

func TestIsBlockLikeSubject(t *testing.T) {
	// Block and Beacon are block-like; everything else is not.
	for _, c := range []Code{Block, Beacon} {
		if !IsBlockLikeSubject(c) {
			t.Errorf("IsBlockLikeSubject(%d) = false, want true", c)
		}
	}
	for _, c := range []Code{Item, EntropyNIST, EntropyStellar, EntropyBitcoin, CommitmentStellar, CommitmentBitcoin, 0, 12, 19, 99} {
		if IsBlockLikeSubject(c) {
			t.Errorf("IsBlockLikeSubject(%d) = true, want false", c)
		}
	}
}

func TestIsValidExternalCommitment(t *testing.T) {
	valid := []Code{CommitmentStellar, CommitmentBitcoin}
	for _, c := range valid {
		if !IsValidExternalCommitment(c) {
			t.Errorf("IsValidExternalCommitment(%d) = false, want true", c)
		}
	}
	invalid := []Code{0, 10, 20, 30, 42, 49, 50, 9999}
	for _, c := range invalid {
		if IsValidExternalCommitment(c) {
			t.Errorf("IsValidExternalCommitment(%d) = true, want false", c)
		}
	}
}

func TestIsEntropySubject(t *testing.T) {
	if !IsEntropySubject(EntropyNIST) || !IsEntropySubject(EntropyStellar) || !IsEntropySubject(EntropyBitcoin) {
		t.Error("all entropy codes should be entropy subjects")
	}
	for _, c := range []Code{Block, Item, CommitmentStellar, CommitmentBitcoin, 0, 99} {
		if IsEntropySubject(c) {
			t.Errorf("IsEntropySubject(%d) = true, want false", c)
		}
	}
}

func TestNameAndHumanize(t *testing.T) {
	cases := []struct {
		code  Code
		name  string
		human string
	}{
		{Block, "block", "Block"},
		{Beacon, "beacon", "Beacon"},
		{Item, "item", "Item"},
		{EntropyNIST, "entropy_nist", "NIST Beacon"},
		{EntropyStellar, "entropy_stellar", "Stellar Ledger"},
		{EntropyBitcoin, "entropy_bitcoin", "Bitcoin Block"},
		{CommitmentStellar, "commitment_stellar", "Stellar"},
		{CommitmentBitcoin, "commitment_bitcoin", "Bitcoin"},
	}
	for _, c := range cases {
		if got := Name(c.code); got != c.name {
			t.Errorf("Name(%d) = %q, want %q", c.code, got, c.name)
		}
		if got := Humanize(c.code); got != c.human {
			t.Errorf("Humanize(%d) = %q, want %q", c.code, got, c.human)
		}
	}
}

func TestNameUnknown(t *testing.T) {
	if Name(Code(999)) != "unknown" {
		t.Error("Name for unknown code must be \"unknown\"")
	}
	if Humanize(Code(999)) != "unknown" {
		t.Error("Humanize for unknown code must be \"unknown\"")
	}
}
