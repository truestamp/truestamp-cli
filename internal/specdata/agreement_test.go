// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package specdata

// The agreement tests of SPEC.md section 13.1, "Beyond the vectors": each
// table this package compiles equals the file under spec/ that publishes it,
// read here from testdata/verification/spec/ with encoding/json, which the
// package itself never uses, and the values SPEC.md states in its text. The
// manifest tests hold the embedded files to that copy byte for byte, so a
// failure here is a table that disagrees with its file: a sync that changed
// a file needs its table changed with it.
//
// No test here assumes what keyring.production.json holds: the keyring
// package's tests hold it to Truestamp's one production key.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

// specMDFiles are the files SPEC.md section 9's table lists under spec/, but
// its one constructions text, specMDConstructionsText.
var specMDFiles = []string{
	"constants.json", "encodings.json", "exit-codes.json", "keyring.production.json",
	"keys.nonproduction.json", "network-pins.json", "networks.json", "prefixes.json",
	"reason-codes.json", "schema.json", "small-order-keys.json", "value-classes.json",
}

// specMDConstructionsText is the one text SPEC.md section 9 names, the text
// of section 7.3.
const specMDConstructionsText = "c0d70826d7501d5089184d1947bad3dcc33c3172269d425f16fc0025ed07df49"

// fixture decodes testdata/verification/spec/<name> with encoding/json,
// numbers as json.Number, refusing anything but whitespace after the value
// and an object that names a member twice, which encoding/json would read as
// its last value.
func fixture(t *testing.T, name string) any {
	t.Helper()
	data := mustRead(t, filepath.Join(fixturesDir, "spec", name))
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("spec/%s: %v", name, err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		t.Fatalf("spec/%s: content after the value", name)
	}
	if err := repeatedName(data); err != nil {
		t.Fatalf("spec/%s: %v", name, err)
	}
	return value
}

// repeatedName walks one JSON value token by token and reports the first
// object that names a member twice.
func repeatedName(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch token {
		case json.Delim('{'):
			seen := map[string]bool{}
			for decoder.More() {
				name, err := decoder.Token()
				if err != nil {
					return err
				}
				if seen[name.(string)] {
					return fmt.Errorf("an object names %q twice", name)
				}
				seen[name.(string)] = true
				if err := walk(); err != nil {
					return err
				}
			}
		case json.Delim('['):
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
		default:
			return nil
		}
		_, err = decoder.Token()
		return err
	}
	return walk()
}

func asInt64(t *testing.T, value any) int64 {
	t.Helper()
	number, ok := value.(json.Number)
	if !ok {
		t.Fatalf("%v (%T) is not a number", value, value)
	}
	n, err := number.Int64()
	if err != nil {
		t.Fatalf("%v is not an integer: %v", number, err)
	}
	return n
}

func asStrings(t *testing.T, value any) []string {
	t.Helper()
	list, ok := value.([]any)
	if !ok {
		t.Fatalf("%v (%T) is not an array", value, value)
	}
	out := make([]string, len(list))
	for i, v := range list {
		if out[i], ok = v.(string); !ok {
			t.Fatalf("%v (%T) is not a string", v, v)
		}
	}
	return out
}

// integers converts every json.Number in value to an int64, so a decoded
// file compares with a table built from Go values.
func integers(t *testing.T, value any) any {
	t.Helper()
	switch v := value.(type) {
	case json.Number:
		return asInt64(t, v)
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, e := range v {
			out[k] = integers(t, e)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = integers(t, e)
		}
		return out
	default:
		return v
	}
}

func TestAgreement_EveryFileUnderSpecIsOneSpecMDLists(t *testing.T) {
	want := append(slices.Clone(specMDFiles), "constructions/"+specMDConstructionsText+".json")
	slices.Sort(want)
	if got := Names(); !slices.Equal(got, want) {
		t.Fatalf("the embedded spec/ holds %q, want the files of SPEC.md section 9: %q", got, want)
	}
}

func TestAgreement_Constants(t *testing.T) {
	file := integers(t, fixture(t, "constants.json"))
	if got := any(Constants()); !reflect.DeepEqual(got, file) {
		t.Fatalf("Constants() = %v, want spec/constants.json, %v", got, file)
	}
}

// SPEC.md section 9's table, written here from SPEC.md's text, so a
// constant given another constant's value is caught even when the file and
// Constants() agree.
func TestAgreement_ConstantsHoldSpecMDSection9Values(t *testing.T) {
	for name, c := range map[string]struct{ got, want int64 }{
		"MAX_READ_OCTETS":       {MaxReadOctets, 262144},
		"MAX_DOCUMENT_OCTETS":   {MaxDocumentOctets, 65536},
		"MAX_DEPTH":             {MaxDepth, 32},
		"MAX_PATH_STEPS":        {MaxPathSteps, 32},
		"MAX_INTEGER":           {MaxInteger, 1<<53 - 1},
		"CLAIMS_MAX_OCTETS":     {ClaimsMaxOctets, 20480},
		"RAW_TX_MAX_OCTETS":     {RawTxMaxOctets, 1000},
		"TXOUTPROOF_MAX_OCTETS": {TxoutproofMaxOctets, 1024},
		"BITCOIN_MAX_TOTAL_TXS": {BitcoinMaxTotalTxs, 4000000 / (60 * 4)},
		"BITCOIN_FINALITY":      {BitcoinFinality, 10},
		"BITCOIN_MEDIAN_SPAN":   {BitcoinMedianSpan, 11},
		"STELLAR_INDEX_MARGIN":  {StellarIndexMargin, 60},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", name, c.got, c.want)
		}
	}
	if MaxReadOctets != 4*MaxDocumentOctets {
		t.Errorf("MAX_READ_OCTETS = %d, want 4 x MAX_DOCUMENT_OCTETS", MaxReadOctets)
	}
	if VectorKey != "2152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12" {
		t.Errorf("VECTOR_KEY = %s, want SPEC.md section 9's", VectorKey)
	}
}

func TestAgreement_SmallOrderKeys(t *testing.T) {
	rows, ok := fixture(t, "small-order-keys.json").([]any)
	if !ok {
		t.Fatal("spec/small-order-keys.json is not an array")
	}
	var want []SmallOrderKey
	for _, r := range rows {
		row := r.(map[string]any)
		if len(row) != 3 {
			t.Fatalf("a row of spec/small-order-keys.json has members %v, want hex, order and sign_bit_variant", row)
		}
		key, err := hex.DecodeString(row["hex"].(string))
		if err != nil || len(key) != 32 || hex.EncodeToString(key) != row["hex"] {
			t.Fatalf("%q is not 32 bytes of lowercase hex", row["hex"])
		}
		want = append(want, SmallOrderKey{
			Key:            [32]byte(key),
			Order:          int(asInt64(t, row["order"])),
			SignBitVariant: row["sign_bit_variant"].(bool),
		})
	}
	if got := SmallOrderKeys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SmallOrderKeys() = %v, want spec/small-order-keys.json, %v", got, want)
	}
	if len(want) != 10 {
		t.Fatalf("SMALL_ORDER_KEYS has %d keys, want the ten encodings of SPEC.md section 6.2", len(want))
	}
}

// members writes a class as the file writes it: each member that is set,
// by the file's name for it.
func members(e Encoding) map[string]any {
	out := map[string]any{}
	if e.Pattern != "" {
		out["pattern"] = e.Pattern
	}
	if e.DecodesToOctets != 0 {
		out["decodes_to_octets"] = int64(e.DecodesToOctets)
	}
	if e.CapsInOctets != nil {
		caps := map[string]any{}
		for k, v := range e.CapsInOctets {
			caps[k] = int64(v)
		}
		out["caps_in_octets"] = caps
	}
	if e.Values != nil {
		out["values"] = anyStrings(e.Values)
	}
	if e.ValuesByChain != nil {
		chains := map[string]any{}
		for k, v := range e.ValuesByChain {
			chains[k] = anyStrings(v)
		}
		out["values_by_chain"] = chains
	}
	if e.Range != nil {
		out["minimum"] = e.Range.Minimum
		out["maximum"] = e.Range.Maximum
	}
	if e.Schema != "" {
		out["schema"] = e.Schema
	}
	if e.RoundTrip {
		out["round_trip"] = true
	}
	return out
}

func anyStrings(list []string) []any {
	out := make([]any, len(list))
	for i, s := range list {
		out[i] = s
	}
	return out
}

func TestAgreement_Encodings(t *testing.T) {
	file := integers(t, fixture(t, "encodings.json"))
	got := map[string]any{}
	for name, class := range Encodings() {
		got[name] = members(class)
	}
	if !reflect.DeepEqual(got, file) {
		t.Fatalf("Encodings() = %v, want spec/encodings.json, %v", got, file)
	}
}

// Every pattern compiles under Go's regexp package, between ^ and $ as the schema
// writes it, and two caps and the range are the constants they name.
func TestAgreement_EncodingsPatternsCompileAndNameTheConstants(t *testing.T) {
	for name, class := range Encodings() {
		if class.Pattern == "" {
			continue
		}
		if _, err := regexp.Compile("^(?:" + class.Pattern + ")$"); err != nil {
			t.Errorf("%s's pattern does not compile: %v", name, err)
		}
	}
	classes := Encodings()
	caps := classes["HEXBYTES"].CapsInOctets
	if caps["raw_transaction"] != RawTxMaxOctets || caps["txoutproof"] != TxoutproofMaxOctets {
		t.Errorf("HEXBYTES caps = %v, want RAW_TX_MAX_OCTETS and TXOUTPROOF_MAX_OCTETS", caps)
	}
	if r := classes["INT"].Range; r == nil || r.Minimum != 0 || r.Maximum != MaxInteger {
		t.Errorf("INT range = %v, want 0 to MAX_INTEGER", r)
	}
	if !reflect.DeepEqual(classes["NETWORK"].ValuesByChain, Networks()) {
		t.Errorf("NETWORK's values = %v, want networks.json's, %v", classes["NETWORK"].ValuesByChain, Networks())
	}
}

func TestAgreement_Networks(t *testing.T) {
	file := fixture(t, "networks.json").(map[string]any)
	want := map[string][]string{}
	for chain, names := range file {
		want[chain] = asStrings(t, names)
	}
	if got := Networks(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Networks() = %v, want spec/networks.json, %v", got, want)
	}
}

func TestAgreement_NetworkPins(t *testing.T) {
	file := fixture(t, "network-pins.json").(map[string]any)
	chains := func(value any) ChainNetworks {
		row := value.(map[string]any)
		if len(row) != 2 {
			t.Fatalf("%v names chains other than bitcoin and stellar", row)
		}
		return ChainNetworks{Bitcoin: row["bitcoin"].(string), Stellar: row["stellar"].(string)}
	}
	want := map[string]NetworkPins{}
	for environment, value := range file {
		row := value.(map[string]any)
		if len(row) != 2 {
			t.Fatalf("the %s row has members %v, want commitment and entropy", environment, row)
		}
		want[environment] = NetworkPins{Commitment: chains(row["commitment"]), Entropy: chains(row["entropy"])}
	}

	got := map[string]NetworkPins{}
	for _, environment := range Environments() {
		pins, ok := DefaultNetworkPins(environment)
		if !ok {
			t.Fatalf("DefaultNetworkPins(%q) has no row", environment)
		}
		got[environment] = pins
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the default network pins = %v, want spec/network-pins.json, %v", got, want)
	}
	if _, ok := DefaultNetworkPins("production"); ok {
		t.Fatal(`DefaultNetworkPins("production") has a row, want none: the E_v is prod`)
	}

	classes := fixture(t, "encodings.json").(map[string]any)
	values := asStrings(t, classes["ENV"].(map[string]any)["values"])
	if got, want := Environments(), append([]string{"prod"}, values...); !slices.Equal(got, want) {
		t.Fatalf("Environments() = %q, want prod and ENV's values, %q", got, want)
	}

	names := Networks()
	for environment, pins := range got {
		for _, pin := range []ChainNetworks{pins.Commitment, pins.Entropy} {
			if !slices.Contains(names["bitcoin"], pin.Bitcoin) || !slices.Contains(names["stellar"], pin.Stellar) {
				t.Errorf("the %s pins %v name a network networks.json does not list", environment, pin)
			}
		}
	}

	// SPEC.md section 9's table of default pins.
	specMD := map[string]NetworkPins{
		"prod":    {Commitment: ChainNetworks{"mainnet", "public"}, Entropy: ChainNetworks{"mainnet", "public"}},
		"dev":     {Commitment: ChainNetworks{"regtest", "testnet"}, Entropy: ChainNetworks{"mainnet", "testnet"}},
		"staging": {Commitment: ChainNetworks{"regtest", "testnet"}, Entropy: ChainNetworks{"mainnet", "testnet"}},
		"test":    {Commitment: ChainNetworks{"regtest", "testnet"}, Entropy: ChainNetworks{"regtest", "testnet"}},
	}
	if !reflect.DeepEqual(got, specMD) {
		t.Fatalf("the default network pins = %v, want SPEC.md section 9's table, %v", got, specMD)
	}
}

func TestAgreement_ReasonCodesWithTheirSubReasons(t *testing.T) {
	var want []ReasonEntry
	for _, r := range fixture(t, "reason-codes.json").([]any) {
		row := r.(map[string]any)
		entry := ReasonEntry{
			Code:    Reason(row["code"].(string)),
			Status:  Status(row["status"].(string)),
			Step:    row["step"].(string),
			Meaning: row["meaning"].(string),
		}
		allowed := 4
		if subs, ok := row["sub_reasons"]; ok {
			allowed++
			list := asStrings(t, subs)
			if len(list) == 0 {
				t.Fatalf("the row of %s has an empty sub_reasons, want a list of names or no sub_reasons member", entry.Code)
			}
			for _, s := range list {
				entry.SubReasons = append(entry.SubReasons, SubReason(s))
			}
		}
		if len(row) != allowed {
			t.Fatalf("the row of %s has members %v, want code, meaning, status, step and, at most, sub_reasons", entry.Code, row)
		}
		want = append(want, entry)
	}
	if got := ReasonCodes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ReasonCodes() = %v, want spec/reason-codes.json, %v", got, want)
	}

	var withSubReasons, bare []Reason
	for _, entry := range want {
		if len(entry.SubReasons) > 0 {
			withSubReasons = append(withSubReasons, entry.Code)
		}
		if entry.Status == StatusBare {
			bare = append(bare, entry.Code)
		}
	}
	if !slices.Equal(withSubReasons, []Reason{ReasonMalformedJSON, ReasonInvalidEnvelope}) {
		t.Errorf("the codes with sub_reasons are %v, want malformed_json and invalid_envelope (SPEC.md section 12.1)", withSubReasons)
	}
	if !slices.Equal(bare, []Reason{ReasonConfigurationError, ReasonInputTooLarge, ReasonMalformedJSON, ReasonInvalidEnvelope}) {
		t.Errorf("the bare codes are %v, want SPEC.md section 12.1's four", bare)
	}
	if len(want) != 36 {
		t.Errorf("the table has %d codes, want SPEC.md section 12.1's 36", len(want))
	}
}

// The sub_reason names SPEC.md sections 10.1 and 10.2, step 3, give, in
// their order, written here from SPEC.md's text.
func TestAgreement_SubReasonsAreSpecMDs(t *testing.T) {
	parse := []SubReason{"syntax", "invalid_utf8", "lone_surrogate", "duplicate_name", "number", "depth"}
	envelope := []SubReason{"document", "proof", "signature", "alg", "public_key", "value"}
	if got := SubReasons(ReasonMalformedJSON); !slices.Equal(got, parse) {
		t.Errorf("SubReasons(malformed_json) = %v, want SPEC.md section 10.1's %v", got, parse)
	}
	if got := SubReasons(ReasonInvalidEnvelope); !slices.Equal(got, envelope) {
		t.Errorf("SubReasons(invalid_envelope) = %v, want SPEC.md section 10.2, step 3's %v", got, envelope)
	}
	for _, entry := range ReasonCodes() {
		if entry.Code != ReasonMalformedJSON && entry.Code != ReasonInvalidEnvelope && SubReasons(entry.Code) != nil {
			t.Errorf("SubReasons(%s) = %v, want none", entry.Code, SubReasons(entry.Code))
		}
	}
}

func TestReasonCodes_LookupsAnswerForEachCodeAndNothingElse(t *testing.T) {
	for _, entry := range ReasonCodes() {
		got, ok := LookupReason(entry.Code)
		if !ok || !reflect.DeepEqual(got, entry) {
			t.Errorf("LookupReason(%s) = %v, %v, want its row", entry.Code, got, ok)
		}
		if !IsReason(string(entry.Code)) {
			t.Errorf("IsReason(%q) = false", entry.Code)
		}
	}
	for _, name := range []string{"not_a_code", "", "MALFORMED_JSON", "malformed_json ", "derivation_mismatches"} {
		if IsReason(name) {
			t.Errorf("IsReason(%q) = true, want false", name)
		}
		if _, ok := LookupReason(Reason(name)); ok {
			t.Errorf("LookupReason(%q) found a row, want none", name)
		}
	}
	if got, _ := LookupReason(ReasonExpectedHashWeakAlgorithm); got.Status != StatusWarn {
		t.Errorf("expected_hash_weak_algorithm's status = %s, want warn", got.Status)
	}

	// A caller that changes what it was given changes nothing here.
	codes := ReasonCodes()
	codes[2].SubReasons[0] = "changed"
	if SubReasons(ReasonMalformedJSON)[0] != SubReasonSyntax {
		t.Error("changing the slice ReasonCodes() returned changed the table")
	}
}

// Every Reason constant of reason_codes.go is a code of the table, and every
// code of the table has one, read from the source so a constant the table
// does not use is caught too.
func TestReasonCodes_ConstantsAreExactlyTheTable(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "reason_codes.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var consts []Reason
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value := spec.(*ast.ValueSpec)
			if ident, ok := value.Type.(*ast.Ident); !ok || ident.Name != "Reason" {
				continue
			}
			for _, v := range value.Values {
				text, err := strconv.Unquote(v.(*ast.BasicLit).Value)
				if err != nil {
					t.Fatal(err)
				}
				consts = append(consts, Reason(text))
			}
		}
	}
	var table []Reason
	for _, entry := range ReasonCodes() {
		table = append(table, entry.Code)
	}
	if !slices.Equal(consts, table) {
		t.Fatalf("the Reason constants are %v, want the table's codes in its order, %v", consts, table)
	}
}

func TestReasonCodes_ACodeListedTwiceStops(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("indexReasons took a table that lists a code twice")
		}
	}()
	indexReasons([]ReasonEntry{{Code: ReasonOffline}, {Code: ReasonPriorFailure}, {Code: ReasonOffline}})
}

func TestAgreement_ExitCodes(t *testing.T) {
	var want []ExitEntry
	for _, r := range fixture(t, "exit-codes.json").([]any) {
		row := r.(map[string]any)
		if len(row) != 2 {
			t.Fatalf("a row of spec/exit-codes.json has members %v, want code and run", row)
		}
		want = append(want, ExitEntry{Code: int(asInt64(t, row["code"])), Run: row["run"].(string)})
	}
	if got := ExitCodes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ExitCodes() = %v, want spec/exit-codes.json, %v", got, want)
	}
}

// SPEC.md section 12.2's table, written here from SPEC.md's text.
func TestAgreement_ExitCodeOfEachEndingIsSpecMDs(t *testing.T) {
	for ending, want := range map[Ending]int{
		EndingVerified:             0,
		EndingRuntimeError:         1,
		EndingFault:                2,
		EndingFailed:               3,
		EndingBareRejection:        4,
		EndingUnattributed:         5,
		EndingSuppliedKeyring:      6,
		EndingInternallyConsistent: 7,
		EndingConfigurationError:   8,
	} {
		if got, ok := ExitCode(ending); !ok || got != want {
			t.Errorf("ExitCode(%q) = %d, %v, want %d", ending, got, ok, want)
		}
	}
	if _, ok := ExitCode("label passed"); ok {
		t.Error(`ExitCode("label passed") has a code, want none`)
	}
}

func TestExitCodes_EndingCodesRefuseATableTheyDoNotMatchOneToOne(t *testing.T) {
	cases := map[string][]ExitEntry{
		"a row no ending begins": append(ExitCodes(), ExitEntry{Code: 9, Run: "label passed"}),
		"an ending that begins no row": slices.DeleteFunc(ExitCodes(), func(e ExitEntry) bool {
			return e.Code == 8
		}),
		"an ending that begins two rows": append(ExitCodes(), ExitEntry{Code: 9, Run: "label verified, again"}),
	}
	for name, table := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := endingCodes(table, endings); err == nil {
				t.Fatal("endingCodes took the table")
			}
		})
	}
	codes, err := endingCodes(ExitCodes(), endings)
	if err != nil || len(codes) != len(endings) {
		t.Fatalf("endingCodes of the table = %v, %v, want a code for each of the %d endings", codes, err, len(endings))
	}
}

func TestAgreement_ConstructionsDigests(t *testing.T) {
	dir := filepath.Join(fixturesDir, "spec", "constructions")
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, f := range files {
		sum := sha256.Sum256(mustRead(t, filepath.Join(dir, f.Name())))
		digest := hex.EncodeToString(sum[:])
		if f.Name() != digest+".json" {
			t.Fatalf("spec/constructions/%s is not named by its SHA-256, %s", f.Name(), digest)
		}
		want = append(want, digest)
	}
	slices.Sort(want)
	if got := Accepted(); !slices.Equal(got, want) {
		t.Fatalf("Accepted() = %q, want the digests of spec/constructions/, %q", got, want)
	}
	for _, digest := range want {
		if !IsAccepted(digest) {
			t.Errorf("IsAccepted(%s) = false", digest)
		}
	}
	if got := Accepted(); !slices.Equal(got, []string{specMDConstructionsText}) {
		t.Errorf("Accepted() = %q, want only %s, the one text SPEC.md section 9 names", got, specMDConstructionsText)
	}
	if !IsAccepted(specMDConstructionsText) {
		t.Errorf("IsAccepted(%s), the text SPEC.md section 9 names, = false", specMDConstructionsText)
	}
	for _, digest := range []string{strings.ToUpper(specMDConstructionsText), specMDConstructionsText[1:], "", strings.Repeat("0", 64)} {
		if IsAccepted(digest) {
			t.Errorf("IsAccepted(%q) = true, want false", digest)
		}
	}
}

func TestConstructions_AFileNotNamedByItsDigestStops(t *testing.T) {
	text := []byte(`{"a":"b"}`)
	sum := sha256.Sum256(text)
	digest := hex.EncodeToString(sum[:])
	other := sha256.Sum256([]byte("other"))
	file := func(data []byte) *fstest.MapFile { return &fstest.MapFile{Data: data} }

	cases := map[string]fstest.MapFS{
		"a name that is another text's digest": {"constructions/" + hex.EncodeToString(other[:]) + ".json": file(text)},
		"its digest in uppercase":              {"constructions/" + strings.ToUpper(digest) + ".json": file(text)},
		"its digest with no .json":             {"constructions/" + digest: file(text)},
		"its digest cut short":                 {"constructions/" + digest[:63] + ".json": file(text)},
		"a final newline added to its bytes":   {"constructions/" + digest + ".json": file(append(slices.Clone(text), '\n'))},
		"a stray .DS_Store beside a good text": {
			"constructions/" + digest + ".json": file(text),
			"constructions/.DS_Store":           file(nil),
		},
		"a directory beside a good text": {
			"constructions/" + digest + ".json":        file(text),
			"constructions/nested/" + digest + ".json": file(text),
		},
		"no text":           {"constructions": &fstest.MapFile{Mode: fs.ModeDir}},
		"no constructions/": {"constants.json": file([]byte("{}"))},
	}
	for name, fsys := range cases {
		t.Run(name, func(t *testing.T) {
			if got, err := acceptedDigests(fsys, "constructions"); err == nil {
				t.Fatalf("acceptedDigests took the directory: %q", got)
			}
		})
	}

	good := fstest.MapFS{"constructions/" + digest + ".json": file(text)}
	got, err := acceptedDigests(good, "constructions")
	if err != nil || !slices.Equal(got, []string{digest}) {
		t.Fatalf("acceptedDigests of one good text = %q, %v, want [%s]", got, err, digest)
	}
}

func TestFile_ServesExactBytesAndNothingOutsideSpec(t *testing.T) {
	for _, name := range Names() {
		first, err := File(name)
		if err != nil {
			t.Fatalf("File(%q): %v", name, err)
		}
		if len(first) > 0 {
			first[0] ^= 1
		}
		second, _ := File(name)
		if !bytes.Equal(second, mustRead(t, filepath.Join(fixturesDir, "spec", filepath.FromSlash(name)))) {
			t.Fatalf("File(%q) after a caller changed its last answer differs from the file", name)
		}
	}
	for _, name := range []string{"", ".", "..", "../SOURCE", "/constants.json", "constructions", "missing.json", "./constants.json", "constants.json/"} {
		if data, err := File(name); err == nil {
			t.Errorf("File(%q) = %d bytes, want an error", name, len(data))
		}
	}
	if _, err := File("missing.json"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("File(missing.json) error = %v, want fs.ErrNotExist", err)
	}
}

func TestPrefixByte_ReadsOnlyTheRegistrySpelling(t *testing.T) {
	for text, want := range map[string]byte{"0x00": 0x00, "0x01": 0x01, "0x50": 0x50, "0x6f": 0x6f, "0xff": 0xff, "0xa9": 0xa9} {
		if got, err := PrefixByte(text); err != nil || got != want {
			t.Errorf("PrefixByte(%q) = %#x, %v, want %#x", text, got, err, want)
		}
	}
	for _, text := range []string{"", "0x", "0x5", "0x500", "0X50", "0x5G", "0xAB", "50", "x050", "0x-1", " 0x50", "0x5 "} {
		if got, err := PrefixByte(text); err == nil {
			t.Errorf("PrefixByte(%q) = %#x, want an error", text, got)
		}
	}
}
