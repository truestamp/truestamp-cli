// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// TestCLI_Inspect_ReportsOnlyWhatTheBundleCarries.
//
// inspect used to emit `type_code`, the numeric subject type. The bundle
// does not carry it: it carries the type NAME, in both JSON and CBOR, and
// the server never returns the number. The code exists only inside the
// signature preimage and is reachable only through the frozen registry,
// so reporting it here was inspect asserting something the bundle does
// not say. `schema get subject-types` has the mapping.
func TestCLI_Inspect_ReportsOnlyWhatTheBundleCarries(t *testing.T) {
	bundle := testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete)

	out, err := exec.Command(binaryPath, "inspect", bundle, "--json").Output()
	if err != nil {
		t.Fatalf("inspect --json: %v", err)
	}
	var got map[string]any
	if jErr := json.Unmarshal(out, &got); jErr != nil {
		t.Fatalf("not JSON: %v", jErr)
	}
	if _, present := got["type_code"]; present {
		t.Error("type_code is not carried by the bundle; inspect must not report it")
	}
	if got["type"] != "item" {
		t.Errorf("the carried type name should still be reported, got %v", got["type"])
	}

	text, err := exec.Command(binaryPath, "inspect", bundle).Output()
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if strings.Contains(string(text), "code 20") {
		t.Errorf("the text rendering still reports the numeric code:\n%s", text)
	}

	// key_id stays, and stays labelled. It is a different thing: a pure
	// function of a value the bundle carries (truncate4(SHA256(0x51||pk))),
	// so any holder can recompute it, and it is what the signature preimage
	// actually uses — comparing it against a stored signing_key_id is how
	// key rotation becomes visible.
	if !strings.Contains(string(text), "Derived key id") {
		t.Error("the derived key id should still be shown, and labelled as derived")
	}
}
