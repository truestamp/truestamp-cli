# JCS Handling

Canonicalization goes through **`internal/jcs`**, never through `github.com/gowebpki/jcs` directly. `internal/jcs` is the only package permitted to import the library; a grep gate keeps it that way:

```bash
grep -rn 'gowebpki/jcs' --include='*.go' . | grep -v internal/jcs   # only test files may appear
```

`internal/jcs.Canonicalize(raw) (canonical []byte, oversized []string, err error)` wraps the library rather than replacing it: for any document free of oversized integers the output is byte-identical to `jcs.Transform`, which is what makes adopting it incapable of moving an existing digest.

The one deviation is deliberate and matches the Truestamp producer. RFC 8785 §3.2.2.3 defines every JSON number by parsing it into an IEEE-754 double and re-serializing, which silently rounds 2^53 + 1 down to 2^53. Truestamp emits integers at arbitrary precision, so reproducing a `claims_hash` means matching the producer. Appendix C.2a pins both halves as normative vectors: 2^53 and 2^53 + 1 canonicalize to distinct strings with distinct 0x11 digests. `Canonicalize` splices such literals out before handing the document to the library and restores them into the canonical output, so sorting, escaping and ES6 float formatting all still run on the library's unmodified code path. Floats are never flagged: they are lossy by construction.

The cost of that choice is that such a bundle is not portably verifiable by a strict RFC 8785 implementation, which Appendix E.4 requires a verifier to **report** rather than hide. `oversized` carries the offending literals, ascending, and every caller surfaces them:

| Surface | How the E.4 signal appears |
| ------- | -------------------------- |
| `truestamp verify` | one `warn` row under Subject Data / cryptographic. Does **not** skip or fail the dependent Inclusion Proof / Proof Signature steps. |
| `truestamp jcs` | one stderr line; `oversized_integers` array under `--json`; nothing under `--silent`. Exit code stays 0. |
| `truestamp hash --jcs` | same stderr line (labelled with the filename when several inputs are hashed); per-input `oversized_integers` under `--json`. |

The parser preserves `json.RawMessage` for the subject data (`s.d`); these are passed straight to `Canonicalize` without re-marshaling through Go structs, which would change field ordering or number formatting. `internal/proof`'s CBOR marshaller decodes `s.d` with `UseNumber()` for the same reason, `cmd/convert_proof.go`'s pretty-printer re-indents with `json.Indent` rather than round-tripping through `any`, and `cmd/create.go` decodes every user-supplied claims document through `decodeUserJSON` (a `json.Decoder` with `UseNumber()`); all three used to round every integer above 2^53 to a float64.

## The two thresholds differ by one, on purpose

`internal/jcs` carries **two** adjacent bounds. They are one apart, and unifying them is a bug in whichever direction it is done:

| Constant | Value | Side | Rule |
| -------- | ----- | ---- | ---- |
| `MaxSafeInteger` | `2^53 - 1` = 9007199254740991 | **producer** (`truestamp create`) | reject `\|n\| > 2^53 - 1` |
| `MaxExactInteger` | `2^53` = 9007199254740992 | **verifier** (`truestamp verify`, `jcs`, `hash --jcs`) | warn at `\|n\| > 2^53` |

The producer follows RFC 8785 Appendix B's SHOULD (JavaScript's `Number.MAX_SAFE_INTEGER`) and matches `Truestamp.SafeIntegers` server-side, so the CLI never emits a claim the server would 422. The verifier stops one value later because 2^53 itself round-trips through a double exactly; warning on it would raise a false alarm about a bundle every conforming implementation *can* check. **Be strict in what you emit, lenient in what you accept.** `internal/jcs.TestThresholdsDifferByOne` fails with an explanatory message if the gap ever closes.

## Producer-side guard in `truestamp create`

`jcs.UnsafeIntegers(root, decoded)` walks a decoded claims map and returns **every** offending integer with its dotted key path (`claims.metadata.rows[0].id`), numbers compared via `math/big` because a 64-bit id or a 400-digit nonce overflows `int64`. Object keys are walked in sorted order so the list is deterministic despite Go's randomized map iteration. Floats are never reported: a large-magnitude value *spelled* as a float (`1e21`) is not an integer literal, exactly as on the verifier side.

`cmd/create.go`'s `checkClaimsPortability` runs it **after `overlayFlags`** (so a value injected by `--metadata` is caught alongside one read from `--claims`) and **before the network call**, across both submission modes and every input path. `jcs.UnsafeIntegerMessage` is a byte-for-byte mirror of `Truestamp.SafeIntegers.message/2` in truestamp-v2 (path, value, allowed range, and the "send the value as a string" remedy) so a user who trips the local guard and one who trips the server's read the same sentence. Under `--json` the rejection is `{"error":"unsafe_integer","message":…,"violations":[{path,value,min,max}]}` with the numbers as **strings**, so a consumer parsing with doubles cannot re-round the very value being complained about.

