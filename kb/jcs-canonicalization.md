<!--
Copyright (c) 2019-2026 Truestamp, Inc.
SPDX-License-Identifier: Apache-2.0
-->

# JCS Handling

Canonicalization goes through **`internal/jcs`**, never through `github.com/gowebpki/jcs` directly. `internal/jcs` is the only package permitted to import the library; a grep gate keeps it that way:

```bash
grep -rn 'gowebpki/jcs' --include='*.go' . | grep -v internal/jcs   # only test files may appear
```

`internal/jcs.Encode(value)` writes JCS of a value as `internal/strictjson` reads it under the parsing rules of SPEC.md section 10.1, with the library's `NumberToJSON` for numbers, and `internal/jcs.Transform(raw)` is that reader and `Encode` over bytes. `truestamp jcs` and `truestamp hash --jcs` canonicalize through `Transform`, so they read their input as a verifier reads a proof: a repeated member name, invalid UTF-8, a lone surrogate, nesting deeper than MAX_DEPTH or anything after the value is refused, and a number is read as its nearest binary64 value (rule J5), so `1` followed by 800 zeros and `e-800` is written `1`.

## One bound

SPEC.md section 9's MAX_INTEGER, 2^53 - 1, is the one safe-integer bound in every path. Rule J5 refuses an integer of magnitude above it and below 10^21 however it is written (`9007199254740992`, `9007199254740992.0`, `1e16`), and `jcs` and `hash --jcs` refuse it as J5 does: `malformed_json (number)`, exit 1, nothing written. `jcs.MaxSafeInteger` is `strictjson.MaxInteger`, the same bound, which the producer-side guard below holds claims to.

## Producer-side guard in `truestamp items create`

`jcs.UnsafeIntegers(root, decoded)` walks a decoded claims map and returns **every** offending integer with its dotted key path (`claims.metadata.rows[0].id`), numbers compared via `math/big` because a 64-bit id or a 400-digit nonce overflows `int64`. Object keys are walked in sorted order so the list is deterministic despite Go's randomized map iteration. Floats are never reported: a large-magnitude value *spelled* as a float (`1e21`) is not an integer literal.

`cmd/items_create.go`'s `checkClaimsPortability` runs it **after `overlayFlags`** (so a value injected by `--metadata` is caught alongside one read from `--claims`) and **before the network call**, across both submission modes and every input path. `jcs.UnsafeIntegerMessage` is a byte-for-byte mirror of `Truestamp.SafeIntegers.message/2` in truestamp-v2 (path, value, allowed range, and the "send the value as a string" remedy) so a user who trips the local guard and one who trips the server's read the same sentence. Under `--json` the rejection is `{"error":"unsafe_integer","message":…,"violations":[{path,value,min,max}]}` with the numbers as **strings**, so a consumer parsing with doubles cannot re-round the very value being complained about.

