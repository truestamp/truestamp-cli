// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"os"
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

// TestCLI_Inspect_HonorsCLIWideOutputSettings.
//
// `--json` and `--silent` are CLI-wide settings, resolvable from
// config.toml and from TRUESTAMP_JSON / TRUESTAMP_SILENT, not just from a
// flag typed on the line. inspect renders a record -- it sits in the same
// help group as verify, which honors them -- but it read the raw cobra
// flags instead of the resolved config, so it was the one
// record-rendering command that silently ignored both. It now goes
// through outputMode, like everything else that prints a record.
func TestCLI_Inspect_HonorsCLIWideOutputSettings(t *testing.T) {
	bundle := testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete)

	t.Run("TRUESTAMP_JSON", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "inspect", bundle)
		cmd.Env = append(os.Environ(), "TRUESTAMP_JSON=true")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("inspect with TRUESTAMP_JSON: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("TRUESTAMP_JSON=true did not produce JSON: %v\ngot: %s",
				err, firstLine(string(out)))
		}
		if got["source"] == nil {
			t.Error("JSON summary has no source field")
		}
	})

	t.Run("TRUESTAMP_SILENT", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "inspect", bundle)
		cmd.Env = append(os.Environ(), "TRUESTAMP_SILENT=true")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("inspect with TRUESTAMP_SILENT: %v", err)
		}
		if len(out) != 0 {
			t.Errorf("TRUESTAMP_SILENT=true still printed %d bytes:\n%s",
				len(out), firstLine(string(out)))
		}
	})

	// Asking for both is incoherent whatever the two settings came from,
	// and config.Load rejects the pair centrally rather than each RunE
	// checking. What matters here is that inspect answers exactly the way
	// verify and the resource commands do, so this asserts the two side by
	// side rather than asserting inspect alone.
	t.Run("json plus silent is rejected, the same way verify rejects it", func(t *testing.T) {
		for _, args := range [][]string{
			{"inspect", bundle, "--json"},
			{"verify", bundle, "--offline", "--json"},
		} {
			cmd := exec.Command(binaryPath, args...)
			cmd.Env = append(os.Environ(), "TRUESTAMP_SILENT=true")
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("%v with TRUESTAMP_SILENT: expected an error, got none", args)
				continue
			}
			if !strings.Contains(string(out), "mutually exclusive") {
				t.Errorf("%v with TRUESTAMP_SILENT: got %q, want a mutual-exclusion error",
					args, firstLine(string(out)))
			}
		}
	})
}
