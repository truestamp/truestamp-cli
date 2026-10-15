// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's
// Truestamp.Verifier.Derive.KeyEvents
// (elixir/lib/truestamp/verifier/derive/key_events.ex in
// truestamp-verification), checked against the Truestamp application's
// port, Truestamp.Proof.Verifier.Derive.KeyEvents
// (lib/truestamp/proof/verifier/derive/key_events.ex in truestamp-v2): the
// key-event checks of DERIVE (SPEC.md section 10.3, "Key events"), made
// inside part (b) of step 9 (section 10.2). Wherever a block record carries
// metadata.key_event, at any of the three block positions, the event is
// checked as the table of section 3.3 requires. With B the block node that
// carries it and E the event's pointer B/record/metadata/key_event, every
// failure is key_event_mismatch:
//
//	| Check                                                                   | Events                                | Part                     | Row of 7.3 | Traced |
//	|-------------------------------------------------------------------------|---------------------------------------|--------------------------|------------|--------|
//	| public_key passes the checks of a key's bytes (section 6.2)             | genesis, rotation, emergency_rotation | E/public_key             | key_event  | no     |
//	| key_id = the key id of public_key                                       | the same                              | E/key_id                 | key_event  | yes    |
//	| B/record/signing_key_id = the key id of public_key                      | the same                              | B/record/signing_key_id  | key_event  | yes    |
//	| previous_key_id = the key id of previous_public_key                     | rotation, emergency_rotation          | E/previous_key_id        | handover   | yes    |
//	| height = B/record/height                                                | the same                              | E/height                 | handover   | yes    |
//	| previous_block_hash = B/record/previous_block_hash                      | the same                              | E/previous_block_hash    | handover   | yes    |
//	| handover_signature verifies strictly (section 6.2) under previous_public_key, over the 32 octets of H(0x62, the event without handover_signature) | rotation, and emergency_rotation when it has one | E/handover_signature | handover | yes |
//	| B/record/signing_key_id = active_key_id                                 | prerotation                           | B/record/signing_key_id  | key_event  | yes    |
//
// A revocation introduces no key and is not checked. The key event's type
// chooses only among the alternatives the schema lists for its position,
// and each is checked in full (SPEC.md section 10.6, rule 3). A key id is
// recomputed from a key's 32 octets by internal/registry's KeyIDOf, and the
// ids the event and the record carry are only compared with it: no check
// reads a carried key id in place of its recomputation (rule 4). The
// height and previous_block_hash checks bind a handover to the one place
// in the chain it was signed for (section 3.3).
//
// A public_key is PREIMAGE (section 3.7), so its row gives no trace row.
// For the handover signature, recomputed is the hex of the 32 octets it was
// verified over, and the status is Verify's outcome (section 10.6).
// previous_public_key is only the key of the handover signature, whose
// steps 2 and 3 of section 6.2 internal/verifier/ed25519's Verify makes on
// it. A refused key's or signature's failing row names, under refused, the
// step of section 6.2 that refused it: y_not_below_p (step 2) or
// small_order_key (step 3) for a key, and, for a signature under a key
// that passes them, not_on_curve (step 4), s_not_below_l (step 5) or
// equation (step 7), as the reference names them.
//
// Adapted from the reference: the event without handover_signature is
// written from the members the schema lists for a rotation and an
// emergency rotation, each read by its compiled name, where the reference
// deletes the one member from the event as carried. SHAPE has held the
// event to the schema, whose rotation events are closed objects, so the
// two are the same value; the member set comes from the verifier's
// compiled constants (SPEC.md section 10.6, rule 2), and rule 2's check 7
// holds this file to them. A key or a signature that does not decode, and
// an event of a type the schema does not list, which SHAPE has made
// impossible, are a fault of the verifier, returned as an error, which
// [Derive] reports as internal_error.

package verifier

import (
	"encoding/hex"
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier/ed25519"
)

// keyEventChecks are the checks of the key event the block record record,
// at blockAt/record, carries, in the table's order; none when it carries no
// event, or a revocation.
func keyEventChecks(blockAt Pointer, record map[string]any) ([]deriveCheck, error) {
	metadata, _ := record["metadata"].(map[string]any)
	event, carried := metadata["key_event"].(map[string]any)
	if !carried {
		return nil, nil
	}
	recordAt := under(blockAt, "record")
	eventAt := under(recordAt, "metadata", "key_event")
	switch typ, _ := event["type"].(string); typ {
	case "genesis":
		return introducedChecks(event, eventAt, record, recordAt)
	case "rotation", "emergency_rotation":
		introduced, err := introducedChecks(event, eventAt, record, recordAt)
		if err != nil {
			return nil, err
		}
		handover, err := handoverChecks(event, eventAt, record, recordAt)
		if err != nil {
			return nil, err
		}
		return append(introduced, handover...), nil
	case "prerotation":
		recomputed := event["active_key_id"]
		return []deriveCheck{
			equalCheck(under(recordAt, "signing_key_id"), "key_event", []Pointer{under(eventAt, "active_key_id")},
				record["signing_key_id"], recomputed, "The block record's signing_key_id").failingWith(specdata.ReasonKeyEventMismatch),
		}, nil
	case "revocation":
		return nil, nil
	}
	return nil, fmt.Errorf("the key event at %s has a type the schema does not list", eventAt.Encode())
}

// introducedChecks are the three checks of an event that introduces a key:
// its public_key's bytes, its key_id and its record's signing_key_id.
func introducedChecks(event map[string]any, eventAt Pointer, record map[string]any, recordAt Pointer) ([]deriveCheck, error) {
	keyAt := under(eventAt, "public_key")
	text, _ := event["public_key"].(string)
	key, err := registry.DecodeB64U32(text)
	if err != nil {
		return nil, fmt.Errorf("the key at %s: %w", keyAt.Encode(), err)
	}
	keyID := registry.KeyIDOf(key)
	fromKey := []Pointer{keyAt}
	return []deriveCheck{
		keyBytesCheck(keyAt, text, key),
		equalCheck(under(eventAt, "key_id"), "key_event", fromKey, event["key_id"], keyID,
			"The key event's key_id").failingWith(specdata.ReasonKeyEventMismatch),
		equalCheck(under(recordAt, "signing_key_id"), "key_event", fromKey, record["signing_key_id"], keyID,
			"The block record's signing_key_id").failingWith(specdata.ReasonKeyEventMismatch),
	}, nil
}

// handoverChecks are the four checks of a rotation's handover, the last
// made only when the event carries a handover_signature: SHAPE has held a
// rotation to carry one, and an emergency rotation may carry none.
func handoverChecks(event map[string]any, eventAt Pointer, record map[string]any, recordAt Pointer) ([]deriveCheck, error) {
	previousAt := under(eventAt, "previous_public_key")
	text, _ := event["previous_public_key"].(string)
	previous, err := registry.DecodeB64U32(text)
	if err != nil {
		return nil, fmt.Errorf("the key at %s: %w", previousAt.Encode(), err)
	}
	out := []deriveCheck{
		equalCheck(under(eventAt, "previous_key_id"), "handover", []Pointer{previousAt}, event["previous_key_id"],
			registry.KeyIDOf(previous), "The key event's previous_key_id").failingWith(specdata.ReasonKeyEventMismatch),
		equalCheck(under(eventAt, "height"), "handover", []Pointer{under(recordAt, "height")}, event["height"],
			record["height"], "The key event's height").failingWith(specdata.ReasonKeyEventMismatch),
		equalCheck(under(eventAt, "previous_block_hash"), "handover", []Pointer{under(recordAt, "previous_block_hash")},
			event["previous_block_hash"], record["previous_block_hash"], "The key event's previous_block_hash").failingWith(specdata.ReasonKeyEventMismatch),
	}
	signature, signed := event["handover_signature"]
	if !signed {
		return out, nil
	}
	check, err := handoverSignatureCheck(event, eventAt, signature, previous)
	if err != nil {
		return nil, err
	}
	return append(out, check), nil
}

// handoverSignatureCheck is the check that signature, the event's
// handover_signature, verifies strictly under previous over the 32 octets
// of H(0x62, the event without handover_signature).
func handoverSignatureCheck(event map[string]any, eventAt Pointer, signature any, previous [32]byte) (deriveCheck, error) {
	part := under(eventAt, "handover_signature")
	text, _ := signature.(string)
	value, err := registry.DecodeB64U64(text)
	if err != nil {
		return deriveCheck{}, fmt.Errorf("the signature at %s: %w", part.Encode(), err)
	}
	message, err := registry.RecordHash(registry.Handover, unsignedHandover(event))
	if err != nil {
		return deriveCheck{}, fmt.Errorf("the key event at %s: %w", eventAt.Encode(), err)
	}
	c := newCheck(part, "handover", []Pointer{eventAt, under(eventAt, "previous_public_key")}, text,
		hex.EncodeToString(message[:]), "The handover signature verifies under the outgoing key.")
	c.traced = true
	if err := ed25519.Verify(previous, message[:], value); err != nil {
		c.code, c.refused = specdata.ReasonKeyEventMismatch, err.Error()
		c.message = fmt.Sprintf("The handover signature at %s does not verify under the key event's previous_public_key over the handover it signs.",
			part.Encode())
	}
	return c, nil
}

// unsignedHandover is a rotation or an emergency rotation without its
// handover_signature: each other member the schema lists for it, as the
// event carries it, the optional prerotation_commitment only when it is
// carried (see the file's doc). derive_key_events_test.go holds the list
// to spec/schema.json's rotation events.
func unsignedHandover(event map[string]any) map[string]any {
	unsigned := map[string]any{
		"height":              event["height"],
		"key_id":              event["key_id"],
		"previous_block_hash": event["previous_block_hash"],
		"previous_key_id":     event["previous_key_id"],
		"previous_public_key": event["previous_public_key"],
		"public_key":          event["public_key"],
		"reason":              event["reason"],
		"sequence":            event["sequence"],
		"type":                event["type"],
	}
	if commitment, carried := event["prerotation_commitment"]; carried {
		unsigned["prerotation_commitment"] = commitment
	}
	return unsigned
}

// keyBytesCheck is the check that key, the 32 octets text decodes to at
// part, passes steps 2 and 3 of SPEC.md section 6.2. It compares no values
// and gives no trace row.
func keyBytesCheck(part Pointer, text string, key [32]byte) deriveCheck {
	c := newCheck(part, "key_event", []Pointer{part}, text, nil, "The key event's public_key passes the checks of a key's bytes.")
	c.uncompared = true
	if err := ed25519.CheckKeyBytes(key); err != nil {
		c.code, c.refused = specdata.ReasonKeyEventMismatch, err.Error()
		c.message = fmt.Sprintf("The key event's public_key at %s fails the checks of a key's bytes.", part.Encode())
	}
	return c
}
