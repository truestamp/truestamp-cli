// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package console

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/items"
)

// TestHashTypeDisplayMatchesServerCanonical pins each algorithm's
// Display name to the v2 server's @hash_types[*].name field. The
// dropdown picker, the watching-screen summary, and any future
// surface that renders a hash type all flow through this Display
// value, so drift between client and server canonical names breaks
// the user's mental model.
func TestHashTypeDisplayMatchesServerCanonical(t *testing.T) {
	t.Parallel()

	// Mirror of lib/truestamp/hash.ex's @hash_types[*].name strings.
	serverNames := map[string]string{
		"md5":      "MD5",
		"sha1":     "SHA-1",
		"sha224":   "SHA-224",
		"sha256":   "SHA-256",
		"sha384":   "SHA-384",
		"sha512":   "SHA-512",
		"sha3_224": "SHA3-224",
		"sha3_256": "SHA3-256",
		"sha3_384": "SHA3-384",
		"sha3_512": "SHA3-512",
		"blake2s":  "BLAKE2s",
		"blake2b":  "BLAKE2b",
	}

	for _, h := range hashTypeOptions {
		want, ok := serverNames[h.Value]
		if !ok {
			t.Errorf("hashTypeOptions has %q which is not in the server map", h.Value)
			continue
		}
		if h.Display != want {
			t.Errorf("%s Display = %q, want %q (server canonical name)",
				h.Value, h.Display, want)
		}
	}
}

// TestSelectLabelStartsWithDisplayName verifies the dropdown label
// is derived from the canonical Display name, the algorithm name
// the user sees first in the picker is the same string that appears
// on the watching screen.
func TestSelectLabelStartsWithDisplayName(t *testing.T) {
	t.Parallel()

	for _, h := range hashTypeOptions {
		if !strings.HasPrefix(h.selectLabel(), h.Display) {
			t.Errorf("%s selectLabel = %q does not start with Display %q",
				h.Value, h.selectLabel(), h.Display)
		}
	}
}

// TestDisplayHashTypeFallsBackToWireValue confirms the lookup is
// best-effort: an unknown wire value renders as itself rather than
// blanking the watching-screen field.
func TestDisplayHashTypeFallsBackToWireValue(t *testing.T) {
	t.Parallel()

	if got := displayHashType("sha256"); got != "SHA-256" {
		t.Errorf("displayHashType(\"sha256\") = %q, want SHA-256", got)
	}
	if got := displayHashType("ripemd160"); got != "ripemd160" {
		t.Errorf("displayHashType of unknown should pass through, got %q", got)
	}
}

// TestHashTypeOptionsCoverServerAcceptedTypes pins the canonical set
// the v2 backend accepts (lib/truestamp/hash.ex @hash_types). If the
// server adds an algorithm and we forget to mirror it, the new type
// is silently unselectable in the form. This test is the canary.
func TestHashTypeOptionsCoverServerAcceptedTypes(t *testing.T) {
	t.Parallel()

	// Mirror of the server's @hash_types map (just the wire keys).
	// Update both sides if the server's set changes.
	wantTypes := []string{
		"md5", "sha1", "sha224", "sha256", "sha384", "sha512",
		"sha3_224", "sha3_256", "sha3_384", "sha3_512",
		"blake2s", "blake2b",
	}

	got := make(map[string]bool, len(hashTypeOptions))
	for _, h := range hashTypeOptions {
		got[h.Value] = true
	}

	for _, want := range wantTypes {
		if !got[want] {
			t.Errorf("hashTypeOptions missing %q (server accepts it)", want)
		}
	}

	if len(hashTypeOptions) != len(wantTypes) {
		t.Errorf("hashTypeOptions has %d entries, server has %d, drift",
			len(hashTypeOptions), len(wantTypes))
	}
}

// TestHashTypeHexLengthsMatchByteSizes verifies the hex-length entry
// is exactly 2× the canonical byte size of each algorithm. Wrong
// lengths would let the user submit an unparseable hash that the
// server then rejects with a less helpful error.
func TestHashTypeHexLengthsMatchByteSizes(t *testing.T) {
	t.Parallel()

	wantBytes := map[string]int{
		"md5":      16,
		"sha1":     20,
		"sha224":   28,
		"sha256":   32,
		"sha384":   48,
		"sha512":   64,
		"sha3_224": 28,
		"sha3_256": 32,
		"sha3_384": 48,
		"sha3_512": 64,
		"blake2s":  32,
		"blake2b":  64,
	}

	for _, h := range hashTypeOptions {
		want, ok := wantBytes[h.Value]
		if !ok {
			t.Errorf("hashTypeOptions has unexpected %q", h.Value)
			continue
		}
		if h.HexLen != want*2 {
			t.Errorf("%s HexLen = %d, want %d (= 2 × %d bytes)",
				h.Value, h.HexLen, want*2, want)
		}
	}
}

// TestValidateHashAcrossAllTypes confirms validateHash accepts a
// correctly-sized hex digest for every algorithm and rejects
// off-by-one length errors.
func TestValidateHashAcrossAllTypes(t *testing.T) {
	t.Parallel()

	for _, h := range hashTypeOptions {
		t.Run(h.Value, func(t *testing.T) {
			ht := h.Value
			validate := validateHash(&ht)

			good := strings.Repeat("a", h.HexLen)
			if err := validate(good); err != nil {
				t.Errorf("validateHash rejected a %d-char hex digest for %s: %v",
					h.HexLen, h.Value, err)
			}

			tooShort := strings.Repeat("a", h.HexLen-1)
			if err := validate(tooShort); err == nil {
				t.Errorf("validateHash accepted a %d-char digest for %s (expected %d)",
					h.HexLen-1, h.Value, h.HexLen)
			}

			tooLong := strings.Repeat("a", h.HexLen+1)
			if err := validate(tooLong); err == nil {
				t.Errorf("validateHash accepted a %d-char digest for %s (expected %d)",
					h.HexLen+1, h.Value, h.HexLen)
			}
		})
	}
}

// TestValidateHashRejectsNonHex confirms the regex still bites for
// every hash type, algorithm choice doesn't bypass hex validation.
func TestValidateHashRejectsNonHex(t *testing.T) {
	t.Parallel()

	ht := "sha256"
	validate := validateHash(&ht)

	for _, bad := range []string{
		"not-hex-at-all-not-hex-at-all-not-hex-at-all-not-hex-at-all-1234",
		strings.Repeat("z", 64),
		strings.Repeat(" ", 64),
	} {
		if err := validate(bad); err == nil {
			t.Errorf("validateHash accepted non-hex input %q", bad)
		}
	}
}

// TestValidateHashEmptyHashTypeIsRejected confirms an empty hashType
// is no longer a soft-fail, even with otherwise-valid hex input,
// the validator demands a chosen algorithm so the table-driven
// length check can run.
func TestValidateHashEmptyHashTypeIsRejected(t *testing.T) {
	t.Parallel()

	emptyHT := ""
	validate := validateHash(&emptyHT)

	// Even-length valid hex still rejected, algorithm must be
	// chosen so the length is checked against an expected size.
	if err := validate(strings.Repeat("a", 64)); err == nil {
		t.Error("validateHash accepted input with empty hashType")
	}

	// Odd-length still also rejected (regex bites first).
	if err := validate(strings.Repeat("a", 63)); err == nil {
		t.Error("validateHash accepted odd-length input with empty hashType")
	}
}

// TestValidateHashUnknownHashTypeIsRejected confirms a hashType not
// in our canonical table is rejected with a clear "unknown hash type"
// error, not silently skipped.
func TestValidateHashUnknownHashTypeIsRejected(t *testing.T) {
	t.Parallel()

	unknownHT := "ripemd160"
	validate := validateHash(&unknownHT)

	if err := validate(strings.Repeat("a", 40)); err == nil {
		t.Errorf("validateHash accepted input with unknown hashType %q", unknownHT)
	}
}

// TestValidateHashRejectsOddLengthForAllInputs confirms the pair-
// regex catches odd-length hex regardless of hashType. The regex
// runs before the table lookup so an empty / unknown hashType
// surfaces as the regex error first when the input is malformed.
func TestValidateHashRejectsOddLengthForAllInputs(t *testing.T) {
	t.Parallel()

	for _, ht := range []string{"sha256", "sha512", "blake2b", "sha1", "md5"} {
		validate := validateHash(&ht)
		cases := []string{
			"a",                     // 1 char
			"abc",                   // 3 chars
			strings.Repeat("a", 63), // 63 chars (sha256 minus 1)
			strings.Repeat("a", 41), // 41 chars (sha1 plus 1)
			"deadbee",               // 7 chars
		}
		for _, in := range cases {
			if err := validate(in); err == nil {
				t.Errorf("hashType=%s: validateHash accepted odd-length %q (len=%d)",
					ht, in, len(in))
			}
		}
	}
}

// TestValidateHashEnforcesAllThreeChecks pins the three composed
// invariants (hex-only, even length, table-driven length) by walking
// the matrix of failure modes for a representative algorithm.
func TestValidateHashEnforcesAllThreeChecks(t *testing.T) {
	t.Parallel()

	ht := "sha256" // 64 hex chars
	validate := validateHash(&ht)

	cases := []struct {
		name string
		in   string
		ok   bool
	}{
		{"empty", "", false},
		{"odd length", strings.Repeat("a", 63), false},
		{"even length but wrong size", strings.Repeat("a", 32), false},
		{"non-hex same length", strings.Repeat("z", 64), false},
		{"correct", strings.Repeat("a", 64), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.in)
			if tc.ok && err != nil {
				t.Errorf("expected accept, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("expected reject, got nil")
			}
		})
	}
}

// --- Claims-as-source-of-truth mode ---

// TestValidateDescription_ExternalMode_AlwaysOK confirms the description
// validator is a no-op in external-hash mode regardless of length.
func TestValidateDescription_ExternalMode_AlwaysOK(t *testing.T) {
	t.Parallel()

	mode := formModeExternal
	validate := validateDescription(&mode)

	for _, in := range []string{"", "short", strings.Repeat("x", 1000)} {
		if err := validate(in); err != nil {
			t.Errorf("external mode should accept any description, rejected %q: %v", in, err)
		}
	}
}

// TestValidateDescription_ClaimsOnly_RejectsShort confirms the local
// meaningful-content rule fires in claims-content mode when the
// description is too short, counted in characters (code points): 31
// two-octet characters are refused, with the count the error names.
// Mutant killed: len counting octets in place of the characters (62
// octets pass).
func TestValidateDescription_ClaimsOnly_RejectsShort(t *testing.T) {
	t.Parallel()

	mode := formModeClaimsOnly
	validate := validateDescription(&mode)

	tooShort := []string{
		"",
		"x",
		strings.Repeat("x", claimsOnlyMinDescription-1),
		strings.Repeat("é", claimsOnlyMinDescription-1),
	}
	for _, in := range tooShort {
		if err := validate(in); err == nil {
			t.Errorf("claims-only mode should reject %q", in)
		}
	}
	err := validate(strings.Repeat("é", claimsOnlyMinDescription-1))
	if want := "description must be at least 32 characters in claims-content mode (got 31)"; err == nil || err.Error() != want {
		t.Errorf("error %v, want %q", err, want)
	}
}

// TestValidateDescription_ClaimsOnly_AcceptsLong confirms a description
// at or above the threshold passes the claims-content validator, counted
// as typed: 32 two-octet characters pass, and so do 31 letters inside
// three spaces each side, 37 characters as sent. Mutant killed: the text
// trimmed before it is counted (the padded description is refused).
func TestValidateDescription_ClaimsOnly_AcceptsLong(t *testing.T) {
	t.Parallel()

	mode := formModeClaimsOnly
	validate := validateDescription(&mode)

	atThreshold := strings.Repeat("x", claimsOnlyMinDescription)
	if err := validate(atThreshold); err != nil {
		t.Errorf("exactly %d chars should pass, got %v", claimsOnlyMinDescription, err)
	}
	above := "On this day I claim the following novel approach as my own original work."
	if err := validate(above); err != nil {
		t.Errorf("above-threshold description should pass, got %v", err)
	}
	for _, in := range []string{
		strings.Repeat("é", claimsOnlyMinDescription),
		"   " + strings.Repeat("x", claimsOnlyMinDescription-1) + "   ",
	} {
		if err := validate(in); err != nil {
			t.Errorf("%q should pass, got %v", in, err)
		}
	}
}

// TestBuildCreateItemPayload_External pins the payload as exactly
// {claims_json, watch}, with the claims text holding each field exactly
// as typed: surrounding spaces, an uppercase hash and a quotation mark
// kept, in the form's order. Mutants killed: the payload carrying the
// claims as top-level members (the server refuses every member but
// claims_json and watch), name or description trimmed, and the hash
// lowercased.
func TestBuildCreateItemPayload_External(t *testing.T) {
	t.Parallel()

	hash := "  ABCDEF" + strings.Repeat("0", 58) + "  "
	payload := buildCreateItemPayload(formModeExternal, "  Doc  ", "  some \"context\"  ", hash, "sha256")

	want := map[string]any{
		"claims_json": `{"name":"  Doc  ","description":"  some \"context\"  ","hash":"` + hash + `","hash_type":"sha256"}`,
		"watch":       true,
	}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload\n got: %#v\nwant: %#v", payload, want)
	}
}

// TestBuildCreateItemPayload_BareHashAddsNoHashType: a hash with no hash
// type chosen is sent without hash_type, for the server to refuse the
// half pair. Mutant killed: the sha256 default for a bare hash restored.
func TestBuildCreateItemPayload_BareHashAddsNoHashType(t *testing.T) {
	t.Parallel()

	payload := buildCreateItemPayload(formModeExternal, "Doc", "", strings.Repeat("ab", 32), "")
	want := map[string]any{
		"claims_json": `{"name":"Doc","hash":"` + strings.Repeat("ab", 32) + `"}`,
		"watch":       true,
	}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload\n got: %#v\nwant: %#v", payload, want)
	}
}

// TestBuildCreateItemPayload_ClaimsOnly_IgnoresHashFields: claims-only
// mode leaves the hash fields out even when they hold stale values, and
// a description is written with its control characters escaped and "<"
// as itself, as items.ClaimsJSONString writes it. Mutants killed: the
// writer leaving a newline unescaped (claims_json is then not JSON), and
// the pane writing its claims with a writer of its own that drifts from
// the shared one (encoding/json's, which writes "<" as \u003c).
func TestBuildCreateItemPayload_ClaimsOnly_IgnoresHashFields(t *testing.T) {
	t.Parallel()

	payload := buildCreateItemPayload(formModeClaimsOnly, "Invention",
		"On this day I claim\nthe following <novel> approach.",
		"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "sha256")
	want := map[string]any{
		"claims_json": `{"name":"Invention","description":"On this day I claim\u000athe following <novel> approach."}`,
		"watch":       true,
	}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload\n got: %#v\nwant: %#v", payload, want)
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(payload["claims_json"].(string)), &decoded); err != nil ||
		decoded["description"] != "On this day I claim\nthe following <novel> approach." {
		t.Errorf("claims_json does not read back as the form's values: %v %v", err, decoded)
	}
}

// TestHashTypePickerHasNoPreselectedType: the picker's first option has
// no value, so a form whose hash type is never picked holds none, both
// when the pane opens and after it is reset. Mutants killed: the picker
// listing sha256 first again (huh's Select sets its value to the first
// option), and the reset setting the hash type to sha256.
func TestHashTypePickerHasNoPreselectedType(t *testing.T) {
	t.Parallel()

	opts := hashTypeSelectOptions()
	if len(opts) != len(hashTypeOptions)+1 || opts[0].Value != "" {
		t.Fatalf("first option %q of %d, want an empty value before the %d types", opts[0].Value, len(opts), len(hashTypeOptions))
	}
	m := newNewItemModel(nil)
	if m.formHashType != "" {
		t.Errorf("a new form holds hash type %q, want none", m.formHashType)
	}
	m.formHashType = "sha512"
	m.Update(resetFormMsg{})
	if m.formHashType != "" {
		t.Errorf("a reset form holds hash type %q, want none", m.formHashType)
	}
}

// TestValidateHashJudgesTheTextAsTyped: the hash is judged as it will be
// sent, so uppercase hex and surrounding spaces are refused rather than
// tidied. Mutant killed: validateHash trimming and lowercasing before it
// judges.
func TestValidateHashJudgesTheTextAsTyped(t *testing.T) {
	t.Parallel()

	ht := "sha256"
	validate := validateHash(&ht)
	for _, in := range []string{strings.Repeat("A", 64), " " + strings.Repeat("a", 64), strings.Repeat("a", 64) + " "} {
		if err := validate(in); err == nil || err.Error() != "hash must be an even-length lowercase hex string" {
			t.Errorf("validateHash(%q) = %v", in, err)
		}
	}
}

// TestParseCreateReply_ClaimsReasons keeps every claims_reasons entry of
// a refusal, and the pane shows each with its pointer, code and message.
// Mutants killed: parseCreateReply dropping claims_reasons, and
// renderForm leaving the reasons out.
func TestParseCreateReply_ClaimsReasons(t *testing.T) {
	t.Parallel()

	reply := parseCreateReply("error", json.RawMessage(`{"code":"invalid","message":"/name begins with a space (U+0020). Remove it. The claims need a description.",`+
		`"claims_reasons":[{"code":"claims_surrounding_whitespace","message":"/name begins with a space (U+0020). Remove it.","pointer":"/name"},`+
		`{"code":"claims_content","message":"The claims need a description.","pointer":""}]}`))
	want := []items.ClaimsReason{
		{Code: "claims_surrounding_whitespace", Message: "/name begins with a space (U+0020). Remove it.", Pointer: "/name"},
		{Code: "claims_content", Message: "The claims need a description.", Pointer: ""},
	}
	if reply.err == nil || reply.err.Error() != "invalid: the server refused the claims" || !reflect.DeepEqual(reply.reasons, want) {
		t.Fatalf("reply %+v", reply)
	}

	m := newNewItemModel(nil)
	m.Update(reply)
	view := m.View(120, 60)
	for _, line := range []string{
		`pointer "/name", claims_surrounding_whitespace: /name begins with a space (U+0020). Remove it.`,
		`pointer "", claims_content: The claims need a description.`,
	} {
		if !strings.Contains(view, line) {
			t.Errorf("view missing %q:\n%s", line, view)
		}
	}
}

// TestParseCreateReply_ItemCodeUnavailable keeps the refusal's own code
// and message. Mutant killed: a refusal without claims_reasons told as
// "the server refused the claims".
func TestParseCreateReply_ItemCodeUnavailable(t *testing.T) {
	t.Parallel()

	reply := parseCreateReply("error", json.RawMessage(`{"code":"item_code_unavailable","message":"Could not issue an item code. Try again."}`))
	if reply.err == nil || reply.err.Error() != "item_code_unavailable: Could not issue an item code. Try again." || reply.reasons != nil {
		t.Errorf("reply %+v", reply)
	}
}

// TestParseCreateReply_ClaimsJCSAndHash: a success carries the reply's
// claims_jcs and claims_hash, and the watching pane shows both whole.
// Mutants killed: claims_jcs not read from the reply, and the Claims JCS
// and Claims hash rows left out of the pane.
func TestParseCreateReply_ClaimsJCSAndHash(t *testing.T) {
	t.Parallel()

	const jcs = `{"hash":"abab","hash_type":"sha256","name":"Doc"}`
	const hash = "0f1e2d3c4b5a69788796a5b4c3d2e1f00f1e2d3c4b5a69788796a5b4c3d2e1f0"
	reply := parseCreateReply("ok", json.RawMessage(`{"item":{"id":"019a0000-0000-7000-8000-000000000001","state":"created",`+
		`"claims":{"name":"Doc"},"claims_hash":"`+hash+`","item_hash":"ih"},"watched":true,`+
		`"claims_jcs":"{\"hash\":\"abab\",\"hash_type\":\"sha256\",\"name\":\"Doc\"}","claims_hash":"`+hash+`"}`))
	if reply.err != nil || reply.item == nil || reply.item.ClaimsJCS != jcs || reply.item.ClaimsHash != hash {
		t.Fatalf("reply %+v", reply)
	}

	m := newNewItemModel(nil)
	m.Update(reply)
	view := m.View(200, 60)
	for _, want := range []string{"Claims JCS", jcs, "Claims hash", hash} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}

// TestParseCreateReply_ItemCodeBesideItemHash: a success carries the
// reply item's item_code, and the watching pane gives it whole, as its own
// "Item code" row on the line after the whole "Item hash" row, never in
// its place. Mutants killed: item_code not read from the reply (no Item
// code row); the Item code row left out of the pane; the code put in the
// Item hash row's place (the Item hash row holds the code).
func TestParseCreateReply_ItemCodeBesideItemHash(t *testing.T) {
	t.Parallel()

	const itemHash = "e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c"
	const code = "WX495M26THGQBJ68"
	reply := parseCreateReply("ok", json.RawMessage(`{"item":{"id":"019a0000-0000-7000-8000-000000000001","state":"created",`+
		`"claims":{"name":"Doc"},"claims_hash":"ch","item_hash":"`+itemHash+`","item_code":"`+code+`"},"watched":true,`+
		`"claims_jcs":"{\"name\":\"Doc\"}","claims_hash":"ch"}`))
	if reply.err != nil || reply.item == nil || reply.item.ItemCode != code {
		t.Fatalf("reply %+v", reply)
	}

	m := newNewItemModel(nil)
	m.Update(reply)
	var rows []string
	for _, line := range strings.Split(m.View(200, 60), "\n") {
		if fields := strings.Fields(strings.ReplaceAll(line, "│", " ")); len(fields) >= 3 && fields[0] == "Item" && (fields[1] == "hash" || fields[1] == "code") {
			rows = append(rows, strings.Join(fields[:3], " "))
		}
	}
	if want := []string{"Item hash " + itemHash, "Item code " + code}; !reflect.DeepEqual(rows, want) {
		t.Errorf("rows %q, want %q", rows, want)
	}
}
