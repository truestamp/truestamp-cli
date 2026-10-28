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

SPEC.md section 9's MAX_INTEGER, 2^53 - 1, is the one safe-integer bound in every path. Rule J5 refuses an integer of magnitude above it and below 10^21 however it is written (`9007199254740992`, `9007199254740992.0`, `1e16`), and `jcs` and `hash --jcs` refuse it as J5 does: `malformed_json (number)`, exit 1, nothing written.

## What `truestamp items create` reads before it sends

`items create` sends claims exactly as given, and the server commits their canonical form (`claims_jcs`); the CLI neither canonicalizes nor rewrites them. It reads every claims text it sends with the strict reader, so J5 and the other rules of SPEC.md section 10.1 refuse such a text there, before anything is sent: a claims file (`strictjson.ReadFile`, at most MAX_READ_OCTETS), claims on stdin (`strictjson.Read`), the `--metadata` value and each `--location` number (`strictjson.Decode`) (`cmd/items_create.go`). A refusal names its rule and, for a repeated member name, a number outside the bound or nesting deeper than MAX_DEPTH, what to do. The CLI also refuses a number literal longer than 1,024 octets in any claims (`items.LongNumber`, `items.MaxNumberOctets`): that is a limit of the service's request reader, which reads no request holding one (`TruestampWeb.RequestJSON`, `JSONNumbers.max_octets`), not a rule about the value. It checks `--timestamp` against the grammar the service checks `claims.timestamp` against (`Truestamp.Items.Claims` in truestamp-v2), `YYYY`, `YYYY-MM`, `YYYY-MM-DD`, or `YYYY-MM-DDThh:mm` with optional `:ss`, a fraction of 1 to 9 digits after the seconds, and an optional `Z`, `+hh:mm` or `-hh:mm` offset whose colon may be left out (`+hhmm`), a time following only a whole date, hours (the offset's too) `00` to `23`, minutes and seconds `00` to `59`, `T` and `Z` uppercase; and that a day it names exists; then sends it as typed (`checkTimestamp`, `claimedTimestamp`). Every other claims rule is the server's, reported with its code and its JSON pointer into the claims.
