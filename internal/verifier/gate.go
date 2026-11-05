// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds steps 0 to 6 of SPEC.md section 10.2, the gate, one
// function per step in the spec's order.
//
//	| Step            | Function      | On failure                                            |
//	|-----------------|---------------|-------------------------------------------------------|
//	| 0 CONFIGURATION | [Configure]   | bare rejection configuration_error                    |
//	| 1 READ          | [Read]        | bare rejection input_too_large                        |
//	| 2 PARSE         | [Parse]       | bare rejection malformed_json, with its sub_reason    |
//	| 3 ENVELOPE      | [Envelope]    | bare rejection invalid_envelope, with its sub_reason  |
//	| 4 ENVIRONMENT   | [Environment] | row fail environment_mismatch, and the run ends       |
//	| 5 KEY_BINDING   | [KeyBinding]  | row fail key_binding_failed or keyring_unavailable, and the run ends |
//	| 6 SIGNATURE     | [Signature]   | row fail signature_invalid, and the run ends          |
//
// Each of steps 0 to 3 that passes adds one row pass, part null. [Admit]
// makes the four over a document's bytes and [AdmitFile] over a file,
// which it reads only after CONFIGURATION, as step 0 requires. A keyring
// file or a document file that cannot be read at all is an error wrapping
// strictjson.ErrUnreadable: an input that cannot be read (exit code 1,
// SPEC.md section 12.2), not a verification result.
//
// Steps 4 to 6 take the run steps 0 to 3 give and add their one row. A
// failure at any of them sets the attribution none and ends the run
// ([Run.EndRun]): each later step up to EXPECTED_HASH gets one row skip
// prior_failure, part null, and every step function passes an ended run
// through. When SIGNATURE passes, the gate is open and the run carries its
// attribution (SPEC.md section 10.5): truestamp under the compiled pin or
// the keyring read from www.truestamp.com, keyring:<d> under a supplied
// keyring, and unattributed in keyless mode. [Gate] and [GateFile] make
// the seven steps.
//
// An online run under E_v prod with no supplied keyring reads the keyring
// from www.truestamp.com at CONFIGURATION, and an online run given an
// explicit --base-url reads it from that origin (SPEC.md sections 8.3 and
// 10.2, step 0), both through [Config.ReadKeyring], the seam that
// [NewKeyringReader] (keyring_read.go) fills over HTTP and a conformance
// test from a case's replay file; a read that does not
// give a keyring passing the load checks is no rejection, and leaves the
// run with no keyring that can bind, so KEY_BINDING fails
// keyring_unavailable whatever K is (SPEC.md section 8.3). The
// environment_mismatch hint is SPEC.md section 6.4's, which names an
// offline run, and appears only for a marker of dev, staging or test, as
// that section says. SIGNATURE checks V with internal/verifier/ed25519's
// Verify, which makes steps 2 and 3 of SPEC.md section 6.2 and then calls
// Go's crypto/ed25519.

package verifier

import (
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/verifier/ed25519"
	"github.com/truestamp/truestamp-cli/internal/verifier/keyring"
)

// Alg is the literal signature.alg must equal (SPEC.md section 3.1). It is
// compared, and never selects behavior.
const Alg = "truestamp-ed25519-sha256-jcs"

// envelopeChecks are the checks of ENVELOPE, in the order SPEC.md section
// 10.2, step 3, makes them, by their sub_reason.
var envelopeChecks = []specdata.SubReason{
	specdata.SubReasonDocument, specdata.SubReasonProof, specdata.SubReasonSignature,
	specdata.SubReasonAlg, specdata.SubReasonPublicKey, specdata.SubReasonValue,
}

func init() {
	if !slices.Equal(envelopeChecks, specdata.SubReasons(specdata.ReasonInvalidEnvelope)) {
		panic("verifier: the ENVELOPE checks differ from invalid_envelope's sub_reasons")
	}
}

// EnvelopeChecks returns the checks of ENVELOPE, in the order step 3 makes
// them, by their sub_reason.
func EnvelopeChecks() []specdata.SubReason { return slices.Clone(envelopeChecks) }

// Admit makes steps 0 to 3 over the bytes of a document: the run when each
// passes, so it has a report to hold its rows (SPEC.md section 10.5), or a
// bare rejection, or an error for a configuration no run can take or a
// keyring file that cannot be read.
func Admit(cfg Config, data []byte) (*Run, *Rejection, error) {
	run, rejection, err := Configure(cfg)
	if rejection != nil || err != nil {
		return nil, rejection, err
	}
	return afterConfiguration(run, data)
}

// AdmitFile makes steps 0 to 3 over the file at path: CONFIGURATION first,
// then the file is read, at most MAX_READ_OCTETS + 1 octets of it (SPEC.md
// section 10.1, J1).
func AdmitFile(cfg Config, path string) (*Run, *Rejection, error) {
	run, rejection, err := Configure(cfg)
	if rejection != nil || err != nil {
		return nil, rejection, err
	}
	data, err := strictjson.ReadFile(path)
	var tooLarge *strictjson.Rejection
	switch {
	case errors.As(err, &tooLarge):
		return nil, fromReader(tooLarge), nil
	case err != nil:
		return nil, nil, fmt.Errorf("the proof %s cannot be read: %w", path, err)
	}
	return afterConfiguration(run, data)
}

// Gate makes steps 0 to 6 over the bytes of a document: [Admit], then
// ENVIRONMENT, KEY_BINDING and SIGNATURE over the run it gives. A run whose
// gate failed has ended, with the attribution none; a run that passed it
// carries its attribution.
func Gate(cfg Config, data []byte) (*Run, *Rejection, error) {
	return afterAdmission(Admit(cfg, data))
}

// GateFile makes steps 0 to 6 over the file at path, which [AdmitFile]
// reads only after CONFIGURATION.
func GateFile(cfg Config, path string) (*Run, *Rejection, error) {
	return afterAdmission(AdmitFile(cfg, path))
}

func afterAdmission(run *Run, rejection *Rejection, err error) (*Run, *Rejection, error) {
	if run != nil {
		Environment(run)
		KeyBinding(run)
		Signature(run)
	}
	return run, rejection, err
}

func afterConfiguration(run *Run, data []byte) (*Run, *Rejection, error) {
	for _, step := range []func() *Rejection{
		func() *Rejection { return Read(run, data) },
		func() *Rejection { return Parse(run, data) },
		func() *Rejection { return Envelope(run) },
	} {
		if rejection := step(); rejection != nil {
			return nil, rejection, nil
		}
	}
	return run, nil, nil
}

// ---------------------------------------------------------------------------
// 0. CONFIGURATION

// Configure is step 0, CONFIGURATION: it fixes E_v, the pins and the
// keyring that governs, before the file is read, and adds its row. The
// keyring source decides the keyring (SPEC.md sections 6.4 and 8.3):
//
//   - keyless mode: no keyring governs, under any E_v;
//   - none asked for, under E_v prod: offline, the compiled pin, loaded
//     through the load checks under prod; online, the keyring read from
//     www.truestamp.com;
//   - none asked for, under any other E_v: configuration_error, since such
//     a run must not use the compiled pin or the keyring at
//     www.truestamp.com;
//   - a keyring file or a configured keyring: read, and loaded through the
//     load checks under E_v, so under prod it is a production keyring; a
//     file over MAX_READ_OCTETS or a failed check is configuration_error,
//     and a file that cannot be read at all is an error;
//   - the keyring read from the origin an explicit --base-url names: read
//     online, as from www.truestamp.com.
//
// A keyring read online that does not give a keyring passing the load
// checks, for any reason, is no rejection: the run holds no keyring and
// KeyringUnavailable, so no key binds.
//
// Before any read, in every run, a network's source policy at_least n
// whose n is 0 or above the number of its sources is configuration_error
// (sources.go). An online run then confirms each Stellar and Bitcoin
// source's identity (sources.go): a source whose identity contradicts the
// network it serves, whatever the policy and the network's other sources
// give, and a pinned network with no source are each configuration_error,
// and the run holds every other source's outcome, confirmed or
// unreachable, with the pins it serves, and each pinned network's sources
// in their order under its policy. A source configured for no network
// whose answer only pinned networks with a configuration allow serves
// nothing. Of two sources configured for no network that may serve one
// pin, whether both answer and show its network or both give no usable
// answer, the first in the options' order serves it and the other does
// not. An offline run reads no source.
func Configure(cfg Config) (*Run, *Rejection, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	if rejection := checkPolicies(cfg); rejection != nil {
		return nil, rejection, nil
	}
	run := &Run{Config: cfg}
	switch source := cfg.Keyring; {
	case source.Kind == SourceKeyless:
		run.TrustRoot = TrustRoot{Kind: KindNone}
	case source.Kind == SourceDefault && cfg.Environment != "prod":
		return nil, ConfigurationError(fmt.Sprintf(
			"A run whose environment is %s checks proofs of that environment against a supplied keyring, "+
				"and none was given: run again with --keyring FILE, with a keyring configured for %s, "+
				"or with --no-keyring to check the signature only against the key the proof carries.",
			cfg.Environment, cfg.Environment)), nil
	case source.Kind == SourceDefault && !cfg.Online:
		loaded, err := keyring.Load(compiledPin(), "prod", "")
		if err != nil {
			return nil, keyringRefused(err), nil
		}
		run.Keyring, run.TrustRoot = loaded, TrustRoot{Digest: loaded.Digest, Kind: KindCompiledPin}
	case source.Kind == SourceDefault:
		readOnline(run, currentKeyringRead, KindCurrentKeyring)
	case source.Kind == SourceOrigin:
		readOnline(run, source.URL, KindSuppliedKeyring)
	default:
		data, err := strictjson.ReadFile(source.Path)
		var tooLarge *strictjson.Rejection
		switch {
		case errors.As(err, &tooLarge):
			return nil, keyringRefused(&keyring.Failure{Check: keyring.CheckJSON, Key: -1, Rejection: tooLarge}), nil
		case err != nil:
			return nil, nil, fmt.Errorf("the keyring %s cannot be read: %w", source.Path, err)
		}
		loaded, err := keyring.Load(data, cfg.Environment, source.Path)
		if err != nil {
			return nil, keyringRefused(err), nil
		}
		run.Keyring = loaded
		run.TrustRoot = TrustRoot{Digest: loaded.Digest, Kind: KindSuppliedKeyring, Location: source.Path}
	}
	values := map[string]any{"environment": cfg.Environment, "trust_root": string(run.TrustRoot.Kind)}
	message := fmt.Sprintf("The run checks proofs of Truestamp's %s environment, and %s.", cfg.Environment, governs(run))
	if cfg.Online {
		records, sets, rejection := confirmSources(cfg)
		if rejection != nil {
			return nil, rejection, nil
		}
		run.Sources, run.SourceSets = records, sets
		values["sources"] = sourceValues(records)
		message += " " + sourcesSummary(records)
	}
	run.AddRow(PassRow("CONFIGURATION", nil, message, values))
	return run, nil, nil
}

// compiledPin gives the bytes of the compiled pin, the keyring an offline
// run under E_v prod with no keyring option loads (SPEC.md section 8.3).
// It is keyring.CompiledPin; the package's tests replace it, through
// export_test.go, with a keyring holding a key that exists only in the
// test, since the pin holds Truestamp's production key and no test can
// sign with it.
var compiledPin = keyring.CompiledPin

// readOnline reads the keyring at url through the run's reader and loads it
// under E_v, and trust_root takes the digest of the bytes read (SPEC.md
// sections 8.3 and 10.5). A keyring answered from a replay is a value with
// no bytes of its own (section 13.2), so it loads with [keyring.LoadValue]
// and trust_root's digest is null; it answers only the read of
// www.truestamp.com, since a keyring read from another origin is a supplied
// keyring, whose attribution keyring:<d> names the digest of its bytes,
// which such a value lacks. Any failure, of the read, of J1 to J6 or of a
// load check, a replayed value for another origin included, leaves the run
// with no keyring, no digest and KeyringUnavailable: KEY_BINDING then fails
// keyring_unavailable whatever K is, and the compiled pin is not consulted.
func readOnline(run *Run, url string, kind TrustRootKind) {
	run.TrustRoot = TrustRoot{Kind: kind, Location: url}
	if run.Config.ReadKeyring == nil {
		run.KeyringUnavailable = true
		return
	}
	answer, err := run.Config.ReadKeyring(url)
	if err != nil || (answer.Replayed && kind != KindCurrentKeyring) {
		run.KeyringUnavailable = true
		return
	}
	var loaded *keyring.Keyring
	if answer.Replayed {
		loaded, err = keyring.LoadValue(answer.Value, run.Config.Environment, url)
	} else {
		loaded, err = keyring.Load(answer.Bytes, run.Config.Environment, url)
	}
	if err != nil {
		run.KeyringUnavailable = true
		return
	}
	run.Keyring, run.TrustRoot.Digest = loaded, loaded.Digest
}

// governs says which keyring governs key binding, for CONFIGURATION's
// message.
func governs(run *Run) string {
	switch {
	case run.KeyringUnavailable:
		return fmt.Sprintf("the keyring at %s governs key binding, but it could not be read, so no key binds", run.TrustRoot.Location)
	case run.TrustRoot.Kind == KindCompiledPin:
		return "the keys built into this verifier govern key binding"
	case run.TrustRoot.Kind == KindCurrentKeyring:
		return "the keyring read from www.truestamp.com in this run governs key binding"
	case run.TrustRoot.Kind == KindSuppliedKeyring:
		return fmt.Sprintf("the supplied keyring, %s, governs key binding", run.TrustRoot.Location)
	}
	return "the run is keyless: no keyring governs key binding"
}

// keyringRefused is configuration_error for a keyring file, a configured
// keyring or the compiled pin that fails a load check.
func keyringRefused(err error) *Rejection {
	reason := err.Error()
	var failure *keyring.Failure
	if errors.As(err, &failure) {
		reason = failure.Describe()
	}
	return ConfigurationError("The keyring fails a load check, so the run gives no verdict: " + reason + ".")
}

// ---------------------------------------------------------------------------
// 1. READ

// Read is step 1, READ: J1 (SPEC.md section 10.1), else input_too_large.
func Read(run *Run, data []byte) *Rejection {
	if err := strictjson.CheckSize(data); err != nil {
		return InputTooLarge()
	}
	run.AddRow(PassRow("READ", nil, "The file is within the read limit.",
		map[string]any{"octets": strictjson.Number(len(data))}))
	return nil
}

// ---------------------------------------------------------------------------
// 2. PARSE

// Parse is step 2, PARSE: J2 to J6 (SPEC.md section 10.1), else
// malformed_json with its sub_reason.
func Parse(run *Run, data []byte) *Rejection {
	document, err := strictjson.Parse(data)
	if err != nil {
		var rejection *strictjson.Rejection
		if errors.As(err, &rejection) {
			return fromReader(rejection)
		}
		// Parse gives no other error; a fault here comes before steps 0 to
		// 3 have passed, so it gives no report (SPEC.md section 10.5).
		panic(fmt.Sprintf("verifier: the strict reader failed: %v", err))
	}
	run.Document = document
	run.AddRow(PassRow("PARSE", nil, "The file is one JSON text that every verifier reads alike.", nil))
	return nil
}

// ---------------------------------------------------------------------------
// 3. ENVELOPE

// Envelope is step 3, ENVELOPE: the document is exactly the envelope of
// SPEC.md section 3.1. The checks run in the order of [EnvelopeChecks],
// and the first that fails gives invalid_envelope with its name as the
// sub_reason:
//
//  1. document: an object whose members are exactly proof and signature;
//  2. proof: proof is an object;
//  3. signature: an object whose members are exactly alg, public_key and
//     value;
//  4. alg: alg is the string truestamp-ed25519-sha256-jcs;
//  5. public_key: a string that matches B64U32, whose 32 octets K pass the
//     checks of a key's bytes (SPEC.md section 6.2);
//  6. value: a string that matches B64U64.
//
// A B64U32 or B64U64 string is matched against its class before it is
// decoded, so every spelling but the canonical one fails (section 2.4).
// Its message is fixed, and never names a member the document holds. When
// the checks pass, the run holds P, K and V, and the row's values hold K's
// key id.
func Envelope(run *Run) *Rejection {
	document, isObject := run.Document.(map[string]any)
	_, hasProof := document["proof"]
	_, hasSignature := document["signature"]
	if !isObject || len(document) != 2 || !hasProof || !hasSignature {
		return InvalidEnvelope(specdata.SubReasonDocument)
	}
	proof, isObject := document["proof"].(map[string]any)
	if !isObject {
		return InvalidEnvelope(specdata.SubReasonProof)
	}
	signature, isObject := document["signature"].(map[string]any)
	_, hasAlg := signature["alg"]
	_, hasPublicKey := signature["public_key"]
	_, hasValue := signature["value"]
	if !isObject || len(signature) != 3 || !hasAlg || !hasPublicKey || !hasValue {
		return InvalidEnvelope(specdata.SubReasonSignature)
	}
	if alg, isString := signature["alg"].(string); !isString || alg != Alg {
		return InvalidEnvelope(specdata.SubReasonAlg)
	}
	text, _ := signature["public_key"].(string)
	publicKey, err := registry.DecodeB64U32(text)
	if err != nil || ed25519.CheckKeyBytes(publicKey) != nil {
		return InvalidEnvelope(specdata.SubReasonPublicKey)
	}
	text, _ = signature["value"].(string)
	value, err := registry.DecodeB64U64(text)
	if err != nil {
		return InvalidEnvelope(specdata.SubReasonValue)
	}
	run.Proof, run.PublicKey, run.Signature = proof, publicKey, value
	run.AddRow(PassRow("ENVELOPE", nil, "The document is a proof envelope.",
		map[string]any{"key_id": registry.KeyIDOf(publicKey)}))
	return nil
}

// ---------------------------------------------------------------------------
// 4. ENVIRONMENT

// knownMarkers are the markers that name a Truestamp environment, dev,
// staging and test (ENV, SPEC.md section 2.4): every E_v but prod.
var knownMarkers = slices.DeleteFunc(specdata.Environments(), func(e string) bool { return e == "prod" })

// Environment is step 4, ENVIRONMENT: the check of SPEC.md section 6.4,
// one row at /environment. Under E_v prod the member is absent; under any
// other E_v it is present, a string, and equal to E_v. Anything else is
// environment_mismatch, and the run ends. The row's values hold E_v, and on
// a failure the marker the proof declared, whatever JSON value it is, when
// it carries one (section 10.5); the message never quotes it
// ([environmentMessage]).
func Environment(run *Run) {
	if run.Ended {
		return
	}
	e := run.Config.Environment
	marker, present := run.Proof["environment"]
	values := map[string]any{"environment": e}
	if environmentPasses(e, marker, present) {
		run.AddRow(PassRow("ENVIRONMENT", Part("/environment"), environmentPass(e), values))
		return
	}
	if present {
		values["declared"] = marker
	}
	gateFailure(run, NewRow("ENVIRONMENT", specdata.ReasonEnvironmentMismatch, Part("/environment"),
		environmentMessage(e, marker, present), values))
}

func environmentPasses(e string, marker any, present bool) bool {
	if e == "prod" {
		return !present
	}
	text, isString := marker.(string)
	return present && isString && text == e
}

func environmentPass(e string) string {
	if e == "prod" {
		return "The proof carries no environment marker: a production proof."
	}
	return fmt.Sprintf("The proof is marked as issued by Truestamp's %s environment.", e)
}

// environmentMessage is the message of environment_mismatch under E_v e for
// the marker the proof carries, when present says it carries one. It never
// quotes the marker (SPEC.md section 6.4): it names the marked environment
// only when the marker is dev, staging or test, and then by the name of
// [knownMarkers], never by the document's text; it says production when
// there is no marker; and otherwise it says that the proof carries a
// marker no Truestamp environment uses. Only for a marker of dev, staging
// or test does it add SPEC.md section 6.4's hint, an offline run with that
// environment and a keyring file.
func environmentMessage(e string, marker any, present bool) string {
	if !present {
		return fmt.Sprintf("This proof carries no environment marker, so it is a production proof, "+
			"and this run checks proofs of Truestamp's %s environment.", e)
	}
	text, _ := marker.(string)
	i := slices.Index(knownMarkers, text)
	if i < 0 {
		return "This proof carries an environment marker that no Truestamp environment uses."
	}
	named := knownMarkers[i]
	hint := fmt.Sprintf("To check it against the %s keyring, run again offline with --environment %s and --keyring FILE.", named, named)
	if e == "prod" {
		purpose := "testing"
		if named == "dev" {
			purpose = "development"
		}
		return fmt.Sprintf("This proof is marked as issued by Truestamp's %s environment, which is for %s. "+
			"It is not a production proof. %s", named, purpose, hint)
	}
	return fmt.Sprintf("This proof is marked as issued by Truestamp's %s environment, "+
		"and this run checks proofs of its %s environment. %s", named, e, hint)
}

// ---------------------------------------------------------------------------
// 5. KEY_BINDING

// KeyBinding is step 5, KEY_BINDING: the check of SPEC.md section 6.1,
// item 1, one row with part null, against the keyring that governs. The
// run looks for a key of that keyring whose 32 octets equal the 32 octets K
// decodes to, active or not, and reads no key id the proof carries: K's
// key id is only the row's value. Found, the row passes and the run holds
// the key; not found, it fails key_binding_failed. When the keyring that
// would govern could not be read online, the row fails keyring_unavailable
// whatever K is (section 8.3). In keyless mode the row is skip
// key_binding_not_performed. A failure ends the run.
func KeyBinding(run *Run) {
	if run.Ended {
		return
	}
	values := map[string]any{"key_id": registry.KeyIDOf(run.PublicKey)}
	kind := run.TrustRoot.Kind
	switch {
	case kind == KindNone:
		run.AddRow(NewRow("KEY_BINDING", specdata.ReasonKeyBindingNotPerformed, nil,
			"Keyless mode: the signature is checked only against the key the proof carries, "+
				"which no keyring binds, so anyone could have made this proof.", values))
	case run.KeyringUnavailable || run.Keyring == nil:
		gateFailure(run, NewRow("KEY_BINDING", specdata.ReasonKeyringUnavailable, nil, unavailable(run.TrustRoot), values))
	default:
		key, found := run.Keyring.Find(run.PublicKey)
		if !found {
			gateFailure(run, NewRow("KEY_BINDING", specdata.ReasonKeyBindingFailed, nil, notBound(kind), values))
			return
		}
		run.BoundKey = &key
		run.AddRow(PassRow("KEY_BINDING", nil, bound(kind), values))
	}
}

// unavailable is the message of keyring_unavailable: the keyring could not
// be read (SPEC.md section 8.3). For the keyring of www.truestamp.com it
// also names an offline run, and says, as section 8.3 requires of a message
// that mentions one, that such a run checks only against the keys built
// into the verifier and cannot see a key revoked since.
func unavailable(root TrustRoot) string {
	if root.Kind != KindCurrentKeyring {
		return fmt.Sprintf("The keyring could not be read from %s in this run, so no key binds this proof, whatever key it carries.", root.Location)
	}
	return "The keyring could not be read from www.truestamp.com in this run, so no key binds this proof, whatever key it carries. " +
		"A run with --offline checks the proof only against the keys built into this verifier, and cannot see a key Truestamp has revoked since it was built."
}

func bound(kind TrustRootKind) string {
	if kind == KindSuppliedKeyring {
		return "The key this proof carries is in the supplied keyring."
	}
	return "The key this proof carries is one of Truestamp's keys."
}

func notBound(kind TrustRootKind) string {
	switch kind {
	case KindCompiledPin:
		return "The key this proof carries is not among the keys built into this verifier. " +
			"If Truestamp introduced it after this verifier was built, update the verifier or run online."
	case KindCurrentKeyring:
		return "The key this proof carries is not in the keyring read from www.truestamp.com in this run: " +
			"Truestamp never used it, or has revoked it."
	}
	return "The key this proof carries is not in the supplied keyring."
}

// ---------------------------------------------------------------------------
// 6. SIGNATURE

// Signature is step 6, SIGNATURE: the check of SPEC.md section 6.1, item
// 2, one row with part null. The run computes proof_hash = SHA-256(0x50 ||
// JCS(P)) itself, from P as parsed, through internal/registry, and checks V
// under K over its 32 octets by the strict rule of section 6.2
// (internal/verifier/ed25519's Verify), else signature_invalid, and the run
// ends. The row's values hold proof_hash as hex, and on a failure the step
// of section 6.2 that refused. When it passes, the run takes its
// attribution. A proof_hash that cannot be computed is a fault of the
// verifier: a fail row internal_error, and the run ends (section 10.5).
func Signature(run *Run) {
	if run.Ended {
		return
	}
	proofHash, err := registry.RecordHash(registry.ProofHash, run.Proof)
	if err != nil {
		gateFailure(run, NewRow("SIGNATURE", specdata.ReasonInternalError, nil,
			"The verifier could not compute this proof's proof_hash.", map[string]any{"error": err.Error()}))
		return
	}
	run.ProofHash = proofHash
	values := map[string]any{"proof_hash": hex.EncodeToString(proofHash[:])}
	if err := ed25519.Verify(run.PublicKey, proofHash[:], run.Signature); err != nil {
		values["refused"] = err.Error()
		gateFailure(run, NewRow("SIGNATURE", specdata.ReasonSignatureInvalid, nil,
			"The signature does not verify under the key over this proof's proof_hash.", values))
		return
	}
	run.Attribution = attribution(run.TrustRoot)
	run.AddRow(PassRow("SIGNATURE", nil, "The signature verifies under the key.", values))
}

// attribution is the attribution of a run whose KEY_BINDING and SIGNATURE
// passed under root (SPEC.md section 10.5).
func attribution(root TrustRoot) string {
	switch root.Kind {
	case KindCompiledPin, KindCurrentKeyring:
		return AttributionTruestamp
	case KindSuppliedKeyring:
		return "keyring:" + root.Digest
	}
	return AttributionUnattributed
}

// gateFailure adds row, a failure at steps 4 to 6, and ends the run with
// the attribution none (SPEC.md section 10.5).
func gateFailure(run *Run, row Row) {
	run.Attribution = AttributionNone
	run.AddRow(row)
	run.EndRun()
}
