// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports steps 0 to 3 of the reference verifier's
// Truestamp.Verifier.Gate (elixir/lib/truestamp/verifier/gate.ex in
// truestamp-verification), checked against the Truestamp application's
// port, Truestamp.Proof.Verifier.Gate (lib/truestamp/proof/verifier/gate.ex
// in truestamp-v2): the steps of SPEC.md section 10.2 that decide whether a
// run gives a report at all, one function per step in the spec's order.
//
//	| Step            | Function      | On failure                                            |
//	|-----------------|---------------|-------------------------------------------------------|
//	| 0 CONFIGURATION | [Configure]   | bare rejection configuration_error                    |
//	| 1 READ          | [Read]        | bare rejection input_too_large                        |
//	| 2 PARSE         | [Parse]       | bare rejection malformed_json, with its sub_reason    |
//	| 3 ENVELOPE      | [Envelope]    | bare rejection invalid_envelope, with its sub_reason  |
//
// Each step that passes adds one row pass, part null. [Admit] makes the four
// over a document's bytes and [AdmitFile] over a file, which it reads only
// after CONFIGURATION, as step 0 requires. A keyring file or a document file
// that cannot be read at all is an error wrapping strictjson.ErrUnreadable:
// an input that cannot be read (exit code 1, SPEC.md section 12.2), not a
// verification result.
//
// Adapted from the reference, which is offline: an online run under E_v
// prod with no supplied keyring reads the keyring from www.truestamp.com at
// CONFIGURATION, and an online run given an explicit --base-url reads it
// from that origin, both through [Config.ReadKeyring]; a read that does not
// give a keyring passing the load checks is no rejection, and leaves the
// run with no keyring that can bind (SPEC.md section 8.3). Steps 4 to 6,
// the rest of the gate, take the run these steps give.

package verifier

import (
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
func Configure(cfg Config) (*Run, *Rejection, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
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
		loaded, err := keyring.Load(keyring.CompiledPin(), "prod", "")
		if err != nil {
			return nil, keyringRefused(err), nil
		}
		run.Keyring, run.TrustRoot = loaded, TrustRoot{Kind: KindCompiledPin}
	case source.Kind == SourceDefault:
		readOnline(run, CurrentKeyringURL, KindCurrentKeyring)
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
		run.Keyring, run.TrustRoot = loaded, TrustRoot{Kind: KindSuppliedKeyring, Location: source.Path}
	}
	values := map[string]any{"environment": cfg.Environment, "trust_root": string(run.TrustRoot.Kind)}
	message := fmt.Sprintf("The run checks proofs of Truestamp's %s environment, and %s.", cfg.Environment, governs(run))
	run.AddRow(PassRow("CONFIGURATION", nil, message, values))
	return run, nil, nil
}

// readOnline reads the keyring at url through the run's reader and loads it
// under E_v. Any failure, of the read, of J1 to J6 or of a load check,
// leaves the run with no keyring and KeyringUnavailable.
func readOnline(run *Run, url string, kind TrustRootKind) {
	run.TrustRoot = TrustRoot{Kind: kind, Location: url}
	if run.Config.ReadKeyring == nil {
		run.KeyringUnavailable = true
		return
	}
	data, err := run.Config.ReadKeyring(url)
	if err != nil {
		run.KeyringUnavailable = true
		return
	}
	loaded, err := keyring.Load(data, run.Config.Environment, url)
	if err != nil {
		run.KeyringUnavailable = true
		return
	}
	run.Keyring = loaded
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
