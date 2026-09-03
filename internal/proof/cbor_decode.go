// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

// HasCBORTag reports whether data begins with the RFC 8949 self-describing
// tag 55799 (0xd9 0xd9 0xf7).
func HasCBORTag(data []byte) bool {
	return len(data) >= 3 && data[0] == 0xd9 && data[1] == 0xd9 && data[2] == 0xf7
}

// IsCBORProof reports whether data is a CBOR proof bundle. Appendix E.3
// requires a verifier to accept CBOR both wrapped in the self-describing
// tag 55799 and as a bare map, so a bare CBOR map (major type 5, first byte
// 0xa0-0xbf, definite or indefinite length) counts too. Those bytes are
// UTF-8 continuation bytes and can never open a valid JSON document, so
// widening the check cannot steal input from the JSON path.
func IsCBORProof(data []byte) bool {
	if HasCBORTag(data) {
		return true
	}
	return len(data) > 0 && data[0] >= 0xa0 && data[0] <= 0xbf
}

// CBORToJSON converts a CBOR proof bundle to the equivalent JSON document by
// applying Appendix E.3's field-type correspondence:
//
//   - `public_key` and `signature` byte strings become padded base64 text;
//   - every byte string in a hash slot (each block map's previous_block_hash,
//     merkle_root and signing_key_id; subject.signing_key_id; each
//     commitment's epoch_merkle_root, transaction_hash and
//     block_merkle_root) becomes lowercase hex text;
//   - every map that is a JCS preimage (subject.claims, subject.entropy,
//     subject.metadata, every block metadata map, every entropy witness
//     payload) is converted strictly within the JSON value space, and any
//     value with no JSON counterpart inside one (a byte string, a tag,
//     `undefined`, a simple value other than true/false/null, a non-finite
//     float, a non-text key, text that is not UTF-8) is the hard rejection
//     `invalid_subject_data`;
//   - `inclusion_proof`, `epoch_proof`, `txoutproof` and `raw_transaction`
//     are text on the wire; a byte string in one of those slots renders as
//     null, so a required one is refused by the E.6 gate the reference
//     verifier applies and an optional one reads as absent;
//   - everything else keeps its JSON type. A byte string in an unnamed
//     position renders as hex text and a tag is unwrapped, so an unknown
//     optional field is carried through rather than refused (E.24).
//
// The invariant this preserves is E.3's: every hash derived from the CBOR
// bundle equals the hash derived from the equivalent JSON bundle. Integers
// are rendered exactly, floats per their IEEE-754 value, and map members in
// wire order.
func CBORToJSON(data []byte) ([]byte, error) {
	c := &cborConverter{in: data}
	if HasCBORTag(data) {
		c.pos = 3
	}
	mt, _, _, _, err := c.head()
	if err != nil {
		return nil, Rejectf(CodeNotAJSONObject, "CBOR decode: %s", err)
	}
	if mt != 5 {
		return nil, Rejectf(CodeNotAJSONObject, "input is not a CBOR map")
	}
	if err := c.item(roleBundle, ""); err != nil {
		return nil, err
	}
	return c.out.Bytes(), nil
}

// role names the position a CBOR item occupies in the bundle, which decides
// how it maps into the JSON value space.
type role uint8

const (
	roleGeneric        role = iota // JSON value space; bytes render as hex, tags unwrap
	roleHashed                     // a JCS preimage: strictly the JSON value space
	roleBundle                     // the top-level map
	roleSubject                    // subject
	roleWitnesses                  // subject.witnesses
	roleBlockMap                   // any block map
	roleBlockList                  // block_path
	roleCommitmentList             // commitments, signing_key_event.commitments
	roleCommitment                 // one commitment entry
	roleKeyEvent                   // signing_key_event
	roleHex                        // a 32-byte hash or 4-byte key id slot
	roleBase64                     // public_key, signature
	roleText                       // a slot the schema fixes as text: a byte string there is not the field
)

// childRole returns the role of a map member given its parent's role.
func childRole(parent role, key string) role {
	switch parent {
	case roleHashed:
		return roleHashed
	case roleBundle:
		switch key {
		case "public_key", "signature":
			return roleBase64
		case "inclusion_proof":
			return roleText
		case "subject":
			return roleSubject
		case "block":
			return roleBlockMap
		case "block_path":
			return roleBlockList
		case "commitments":
			return roleCommitmentList
		case "signing_key_event":
			return roleKeyEvent
		}
	case roleSubject:
		switch key {
		case "signing_key_id":
			return roleHex
		case "claims", "entropy", "metadata":
			return roleHashed
		case "witnesses":
			return roleWitnesses
		}
	case roleWitnesses:
		if key == WitnessBlock {
			return roleBlockMap
		}
		return roleHashed
	case roleBlockMap:
		switch key {
		case "previous_block_hash", "merkle_root", "signing_key_id":
			return roleHex
		case "metadata":
			return roleHashed
		}
	case roleCommitment:
		switch key {
		case "epoch_merkle_root", "transaction_hash", "block_merkle_root":
			return roleHex
		case "epoch_proof", "txoutproof", "raw_transaction":
			return roleText
		}
	case roleKeyEvent:
		switch key {
		case "block":
			return roleBlockMap
		case "commitments":
			return roleCommitmentList
		}
	}
	return roleGeneric
}

// elementRole returns the role of an array element given the array's role.
func elementRole(parent role) role {
	switch parent {
	case roleHashed:
		return roleHashed
	case roleBlockList:
		return roleBlockMap
	case roleCommitmentList:
		return roleCommitment
	}
	return roleGeneric
}

type cborConverter struct {
	in  []byte
	pos int
	out bytes.Buffer
}

// head decodes the head of the next data item without consuming it.
func (c *cborConverter) head() (mt byte, arg uint64, headLen int, indefinite bool, err error) {
	return cborHead(c.in[c.pos:])
}

// item converts the data item at the cursor and advances past it.
func (c *cborConverter) item(r role, path string) error {
	mt, arg, headLen, indefinite, err := c.head()
	if err != nil {
		return c.malformed(err.Error())
	}
	c.pos += headLen

	switch mt {
	case 0:
		c.out.WriteString(strconv.FormatUint(arg, 10))
		return nil
	case 1:
		if arg <= math.MaxInt64 {
			c.out.WriteString(strconv.FormatInt(-1-int64(arg), 10))
			return nil
		}
		n := new(big.Int).SetUint64(arg)
		n.Add(n, big.NewInt(1))
		n.Neg(n)
		c.out.WriteString(n.String())
		return nil
	case 2:
		content, err := c.stringContent(2, arg, indefinite)
		if err != nil {
			return err
		}
		switch r {
		case roleHashed:
			return rejectSubjectData(path, "CBOR byte string")
		case roleBase64:
			c.writeJSONString(base64.StdEncoding.EncodeToString(content))
		case roleText:
			// The schema fixes this slot as text (`inclusion_proof`,
			// `epoch_proof`, `txoutproof`, `raw_transaction`), and the
			// server never emits a byte string here. A byte string is
			// therefore not the field: it renders as null, so the E.6
			// gates grade it exactly as the reference verifier does
			// (`missing_inclusion_proof`, `invalid_commitment_entry`) and
			// an optional field reads as absent.
			c.out.WriteString("null")
		default:
			c.writeJSONString(hex.EncodeToString(content))
		}
		return nil
	case 3:
		content, err := c.stringContent(3, arg, indefinite)
		if err != nil {
			return err
		}
		if !utf8.Valid(content) {
			if r == roleHashed {
				return rejectSubjectData(path, "CBOR text string that is not valid UTF-8")
			}
			content = bytes.ToValidUTF8(content, []byte("�"))
		}
		c.writeJSONString(string(content))
		return nil
	case 4:
		return c.array(r, path, arg, indefinite)
	case 5:
		return c.object(r, path, arg, indefinite)
	case 6:
		if r == roleHashed {
			return rejectSubjectData(path, fmt.Sprintf("CBOR tag %d", arg))
		}
		// Outside a hashed map a tag carries nothing the format defines;
		// the tagged content is what the field holds.
		return c.item(r, path)
	default:
		return c.simple(r, path, arg, headLen, indefinite)
	}
}

// simple handles major type 7: the three JSON simple values, `undefined`,
// the reserved simple values, and the three float widths.
func (c *cborConverter) simple(r role, path string, arg uint64, headLen int, indefinite bool) error {
	if indefinite {
		return c.malformed("unexpected CBOR break")
	}
	switch headLen {
	case 1, 2:
		switch arg {
		case 20:
			c.out.WriteString("false")
		case 21:
			c.out.WriteString("true")
		case 22:
			c.out.WriteString("null")
		case 23:
			if r == roleHashed {
				return rejectSubjectData(path, "CBOR undefined")
			}
			c.out.WriteString("null")
		default:
			if r == roleHashed {
				return rejectSubjectData(path, fmt.Sprintf("CBOR simple value %d", arg))
			}
			c.out.WriteString("null")
		}
		return nil
	case 3:
		return c.float(r, path, halfToFloat64(uint16(arg)))
	case 5:
		return c.float(r, path, float64(math.Float32frombits(uint32(arg))))
	default:
		return c.float(r, path, math.Float64frombits(arg))
	}
}

func (c *cborConverter) float(r role, path string, f float64) error {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		if r == roleHashed {
			return rejectSubjectData(path, "CBOR non-finite float")
		}
		c.out.WriteString("null")
		return nil
	}
	c.out.WriteString(floatLiteral(f))
	return nil
}

func (c *cborConverter) array(r role, path string, arg uint64, indefinite bool) error {
	c.out.WriteByte('[')
	er := elementRole(r)
	for i := uint64(0); indefinite || i < arg; i++ {
		if indefinite {
			if c.atBreak() {
				c.pos++
				break
			}
		}
		if i > 0 {
			c.out.WriteByte(',')
		}
		if err := c.item(er, fmt.Sprintf("%s[%d]", path, i)); err != nil {
			return err
		}
	}
	c.out.WriteByte(']')
	return nil
}

func (c *cborConverter) object(r role, path string, arg uint64, indefinite bool) error {
	c.out.WriteByte('{')
	seen := map[string]bool{}
	for i := uint64(0); indefinite || i < arg; i++ {
		if indefinite {
			if c.atBreak() {
				c.pos++
				break
			}
		}
		key, err := c.mapKey(r, path)
		if err != nil {
			return err
		}
		// RFC 8949 section 5.6 makes a map with duplicate keys invalid, and
		// accepting one is a split-view forgery: two decoders that resolve
		// the duplicate differently read two bundles under one signature.
		if seen[key] {
			return c.malformed(fmt.Sprintf("duplicate map key %q in %s", key, pathOrTop(path)))
		}
		seen[key] = true
		if i > 0 {
			c.out.WriteByte(',')
		}
		c.writeJSONString(key)
		c.out.WriteByte(':')
		childPath := key
		if path != "" {
			childPath = path + "." + key
		}
		if err := c.item(childRole(r, key), childPath); err != nil {
			return err
		}
	}
	c.out.WriteByte('}')
	return nil
}

// mapKey reads a map key, which Appendix E.3 requires to be a text string.
func (c *cborConverter) mapKey(r role, path string) (string, error) {
	mt, arg, headLen, indefinite, err := c.head()
	if err != nil {
		return "", c.malformed(err.Error())
	}
	if mt != 3 {
		detail := "map key is not a text string"
		if mt == 2 {
			detail = "map key is a CBOR byte string, not a text string"
		}
		if r == roleHashed {
			return "", rejectSubjectData(path, detail)
		}
		return "", c.malformed(detail + " in " + pathOrTop(path))
	}
	c.pos += headLen
	content, err := c.stringContent(3, arg, indefinite)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(content) {
		if r == roleHashed {
			return "", rejectSubjectData(path, "CBOR map key that is not valid UTF-8")
		}
		content = bytes.ToValidUTF8(content, []byte("�"))
	}
	return string(content), nil
}

// stringContent returns the content of a byte or text string whose head has
// already been consumed, following indefinite-length chunks to their break.
func (c *cborConverter) stringContent(want byte, arg uint64, indefinite bool) ([]byte, error) {
	if !indefinite {
		if arg > uint64(len(c.in)-c.pos) {
			return nil, c.malformed("truncated CBOR string")
		}
		content := c.in[c.pos : c.pos+int(arg)]
		c.pos += int(arg)
		return content, nil
	}
	var out []byte
	for {
		if c.pos >= len(c.in) {
			return nil, c.malformed("truncated indefinite-length CBOR string")
		}
		if c.in[c.pos] == 0xff {
			c.pos++
			return out, nil
		}
		mt, chunkLen, headLen, chunkIndef, err := c.head()
		if err != nil || mt != want || chunkIndef {
			return nil, c.malformed("malformed indefinite-length CBOR string")
		}
		c.pos += headLen
		if chunkLen > uint64(len(c.in)-c.pos) {
			return nil, c.malformed("truncated CBOR string")
		}
		out = append(out, c.in[c.pos:c.pos+int(chunkLen)]...)
		c.pos += int(chunkLen)
	}
}

func (c *cborConverter) atBreak() bool {
	return c.pos < len(c.in) && c.in[c.pos] == 0xff
}

func (c *cborConverter) malformed(detail string) error {
	return Rejectf(CodeNotAJSONObject, "CBOR decode: %s", detail)
}

func pathOrTop(path string) string {
	if path == "" {
		return "the top-level map"
	}
	return path
}

// writeJSONString emits s as a JSON string literal per RFC 8259, escaping
// only what JSON requires. Non-ASCII passes through as UTF-8 and the HTML
// characters are not escaped, so the document reads as the wire would.
func (c *cborConverter) writeJSONString(s string) {
	c.out.WriteByte('"')
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch b {
		case '"':
			c.out.WriteString(`\"`)
		case '\\':
			c.out.WriteString(`\\`)
		case '\b':
			c.out.WriteString(`\b`)
		case '\f':
			c.out.WriteString(`\f`)
		case '\n':
			c.out.WriteString(`\n`)
		case '\r':
			c.out.WriteString(`\r`)
		case '\t':
			c.out.WriteString(`\t`)
		default:
			if b < 0x20 {
				fmt.Fprintf(&c.out, `\u%04x`, b)
			} else {
				c.out.WriteByte(b)
			}
		}
	}
	c.out.WriteByte('"')
}

// cborHead decodes the head of a CBOR data item into its major type,
// argument, head length in bytes, and whether it is indefinite-length.
func cborHead(b []byte) (mt byte, arg uint64, headLen int, indefinite bool, err error) {
	if len(b) == 0 {
		return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
	}
	mt = b[0] >> 5
	ai := b[0] & 0x1f
	switch {
	case ai < 24:
		return mt, uint64(ai), 1, false, nil
	case ai == 24:
		if len(b) < 2 {
			return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
		}
		return mt, uint64(b[1]), 2, false, nil
	case ai == 25:
		if len(b) < 3 {
			return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
		}
		return mt, uint64(binary.BigEndian.Uint16(b[1:3])), 3, false, nil
	case ai == 26:
		if len(b) < 5 {
			return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
		}
		return mt, uint64(binary.BigEndian.Uint32(b[1:5])), 5, false, nil
	case ai == 27:
		if len(b) < 9 {
			return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
		}
		return mt, binary.BigEndian.Uint64(b[1:9]), 9, false, nil
	case ai == 31:
		return mt, 0, 1, true, nil
	default:
		return 0, 0, 0, false, fmt.Errorf("reserved CBOR additional information %d", ai)
	}
}

// halfToFloat64 widens an IEEE-754 binary16 value.
func halfToFloat64(bits uint16) float64 {
	sign := 1.0
	if bits&0x8000 != 0 {
		sign = -1.0
	}
	exp := int((bits >> 10) & 0x1f)
	frac := float64(bits & 0x3ff)
	switch exp {
	case 0:
		return sign * math.Ldexp(frac, -24)
	case 31:
		if frac == 0 {
			return math.Inf(int(sign))
		}
		return math.NaN()
	default:
		return sign * math.Ldexp(frac+1024, exp-25)
	}
}

// floatLiteral renders a CBOR float as a JSON number literal that still
// reads as a float. encoding/json and strconv write every |x| in
// [1e-6, 1e21) positionally, so a CBOR double of 1e20 would come back as
// "100000000000000000000", syntactically an integer literal. That
// misclassification was observable twice: the non-portable-integer report
// fired on a value that is a perfectly portable IEEE-754 double, so the
// same logical bundle produced one report as JSON and a different one as
// CBOR; and re-encoding the bundle to CBOR failed, because the literal is
// outside the exactly representable integer range. JCS parses either
// spelling into the same double, so the derived digest is unchanged.
func floatLiteral(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
