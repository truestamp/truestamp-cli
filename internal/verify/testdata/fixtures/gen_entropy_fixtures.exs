# =============================================================================
# Signed entropy proof-bundle fixture generator (t = 30, 31, 32)
# =============================================================================
#
# Emits three standalone, fully signed proof bundles — one per entropy subject
# type — into the directory this script lives in:
#
#   entropy-nist.json      t = 30  entropy_nist
#   entropy-stellar.json   t = 31  entropy_stellar
#   entropy-bitcoin.json   t = 32  entropy_bitcoin
#
# WHY THIS EXISTS
#
# Every other subject type in this repo has a signed artifact, so its tests run
# the Ed25519 proof-signature check (Appendix E.16) for real. Entropy subjects
# had none: they were exercised only by hand-built bundles that must run with
# --skip-signatures, which removes the one check that makes a proof a proof.
# These fixtures close that gap, and internal/verify/entropy_fixture_test.go
# runs the shipped pipeline over them with signatures ENABLED.
#
# THE SIGNING KEY IS ILLUSTRATIVE, NOT TRUESTAMP'S
#
# The key is the same fixed, published seed whitepaper/proof_vectors.exs signs
# the Appendix D vector with: 32 bytes of 0x42, key id f2c39df9. It is NOT in
# Truestamp's published keyring, so verifying a fixture reports Key Binding as
# `skip` — correct and expected. E.16 ("some key signed this bundle") runs and
# passes; E.17 ("Truestamp's key signed it") is honestly unestablished.
#
# THE ENTROPY VALUES ARE ILLUSTRATIVE AND DELIBERATELY UNRESOLVABLE UPSTREAM
#
# The observed values (NIST outputValue, Stellar ledger hash, Bitcoin block
# hash) are deterministic digests of fixed label strings, and the indices name
# pulses/ledgers/blocks that do not exist. A real E.21 re-fetch therefore
# yields no upstream value to compare, which the verifier grades `skip` — never
# `fail`. That is the property the fixtures need: they must not rot when NIST
# retires a chain or Horizon resets, and they must never turn a sound proof
# into a forged-looking one because an upstream is down. Real upstream values
# would couple these committed bytes to live network state instead.
#
# PROPERTIES THE WAITING TESTS DEPEND ON — keep them true on regeneration:
#
#   1. Signatures verify with the bundle's OWN `t`. The three bundles differ
#      only in that byte within the signing payload, so a payload built with a
#      hardcoded `t` verifies for at most one of them. Checked below, both
#      positively and negatively (each bundle's signature must FAIL under the
#      other two type codes).
#   2. s.d matches the shape internal/verify.parseEntropySubject reads, so the
#      report renders real entropy detail rather than blanks.
#   3. s.id is a UUIDv7 (entropy subjects), never a ULID (items), and its
#      embedded ms is at or before b.id's, so the E.20 submission window holds.
#   4. s.mh is the 0x22 hash of the empty metadata map — the constant Appendix
#      C.4 publishes — so the composite preimage is checkable by hand.
#   5. All three bundles share one block and one pair of commitments, so the
#      inclusion proofs are three real leaves of one tree rather than three
#      unrelated one-leaf trees.
#
# Like whitepaper/proof_vectors.exs this is an INDEPENDENT re-implementation:
# only :crypto, Base, Bitwise and the production Jcs canonicalizer. It calls no
# Truestamp module and nothing in the Go CLI, so a fixture it emits and the Go
# verifier accepts cannot merely encode a Go bug.
#
# RUN (from anywhere; jcs beam path is the truestamp-v2 build dir):
#
#   elixir -pa ../truestamp-v2/_build/dev/lib/jcs/ebin \
#     internal/verify/testdata/fixtures/gen_entropy_fixtures.exs
#
# The script writes ONLY into its own directory, and gates every write behind
# the self-check below: a generator bug raises instead of emitting a bad
# fixture.
# =============================================================================

unless Code.ensure_loaded?(Jcs) do
  raise """
  The production Jcs canonicalizer is not on the code path.

  Re-run with the truestamp-v2 jcs beam directory, e.g.

    elixir -pa ../truestamp-v2/_build/dev/lib/jcs/ebin \\
      internal/verify/testdata/fixtures/gen_entropy_fixtures.exs

  (build it there first with `mix deps.compile jcs` if the directory is absent)
  """
end

# ---------------------------------------------------------------------------
# Primitives: loaded from truestamp-v2, never copied
# ---------------------------------------------------------------------------
#
# The derivation helpers (V) and identifier builders (Ids) come from
# whitepaper/proof_vectors.exs in the truestamp-v2 repo, which validates them
# against Appendix C's published known-answer vectors. Copying them here would
# fork the definition of a domain prefix or a len32 width: their copy would
# change and these fixtures would keep silently encoding the old rule.
#
# That script cannot simply be required. It is a script, not a library, and its
# body writes into whitepaper/vectors/ in a repo this generator must not touch.
# So read it and evaluate ONLY its top-level module definitions, which sit
# above every File.write! it performs.
#
# Override the location with TRUESTAMP_V2 if the repos are not siblings.
v2 = System.get_env("TRUESTAMP_V2") || Path.expand("../truestamp-v2", File.cwd!())
vectors_src = Path.join(v2, "whitepaper/proof_vectors.exs")

unless File.exists?(vectors_src) do
  raise """
  Cannot find the shared proof primitives at:

    #{vectors_src}

  Set TRUESTAMP_V2 to the truestamp-v2 checkout, e.g.

    TRUESTAMP_V2=/path/to/truestamp-v2 elixir -pa ... #{:escript.script_name()}
  """
end

# Take each top-level `defmodule X do` ... `end` block and nothing else. A
# missing module raises below rather than silently regenerating fixtures from
# helpers that are no longer the shared ones.
module_src =
  File.read!(vectors_src)
  |> String.split("\n")
  |> Enum.reduce({[], nil}, fn line, {blocks, current} ->
    cond do
      current == nil and String.starts_with?(line, "defmodule ") -> {blocks, [line]}
      current != nil and line == "end" -> {blocks ++ [Enum.join(current ++ [line], "\n")], nil}
      current != nil -> {blocks, current ++ [line]}
      true -> {blocks, current}
    end
  end)
  |> elem(0)
  |> Enum.join("\n\n")

Code.eval_string(module_src, [], file: vectors_src)

for mod <- [V, Ids] do
  Code.ensure_loaded?(mod) ||
    raise "#{inspect(mod)} was not defined by #{vectors_src}: the shared generator has been restructured, so these fixtures cannot be regenerated from it"
end


assert = fn cond, msg -> if cond, do: :ok, else: raise("SELF-VERIFY FAILED: #{msg}") end

kat = fn label, actual, expected ->
  if actual != expected do
    raise "SELF-VERIFY FAILED: #{label}\n  computed: #{inspect(actual)}\n  expected: #{inspect(expected)}"
  end
end

# ===========================================================================
# CHECK 0: the copied primitives still agree with the published vectors
# ===========================================================================
# The helpers above are a copy of whitepaper/proof_vectors.exs. These are the
# Appendix C known-answer values for the primitives this script actually uses,
# transcribed literally. A copy that drifted (a domain prefix, a len32 width,
# an RFC 6962 leaf prefix, the empty-tree short circuit) breaks here rather
# than quietly emitting fixtures under a different rule set.
kat.(
  "C.4 empty_tree_root",
  V.empty_tree_root(),
  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

kat.(
  "C.4 empty_leaf_input",
  V.hex(V.empty_leaf_hash()),
  "96a296d224f285c67bee93c30f8a309157f0daa35dc5b87e410b78630a09cfc7"
)

kat.(
  "C.4 empty_block_metadata (0x33)",
  V.block_meta_hash(%{}),
  "14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1"
)

kat.(
  "C.4 empty_entropy_metadata (0x22)",
  V.entropy_meta_hash(%{}),
  "5dd81d4309de99c9f8e70822e760f0eece8c33370ff5fb5da3af75bab0cbaab8"
)

kat.(
  "C.3 SHA-256(0x11 || \"test data\")",
  V.h(0x11, "test data") |> V.hex(),
  "4b72c11e5d1192e35529c75ce8cff08aacb1e2186735a8e51c6a1ebfc9057d9b"
)

kat.(
  "C.3 SHA-256(0x21 || \"entropy data\")",
  V.h(0x21, "entropy data") |> V.hex(),
  "439c7a900a9e3ba59b94a7e8583b628384ee1084acddd2cda3a4fe256ad716d0"
)

kat.("C.3 JCS({\"name\":\"Alice\",\"age\":30})", V.jcs(%{"age" => 30, "name" => "Alice"}), ~s({"age":30,"name":"Alice"}))

# The Appendix C entropy composite: same id, entropy hash, metadata hash and
# key id the whitepaper publishes, through this file's copy of the 0x23 path.
kat.(
  "C.5 observation_hash (0x23)",
  V.observation_hash(
    "019f93fe-3c04-70a1-8000-0000000000a1",
    V.entropy_hash(%{
      "outputValue" => String.duplicate("ab", 64),
      "pulseIndex" => 2_847_561,
      "chainIndex" => 1,
      "uri" => "nist:1:2847561"
    }),
    V.entropy_meta_hash(%{}),
    "f2c39df9"
  ),
  "e4e5bf625168c678a5b15fc911c4714cade1c4b5e62f2b14e430610cd0808b9b"
)

# ===========================================================================
# KEY, TIME BASE, IDENTIFIERS
# ===========================================================================

# Illustrative, reproducible signing key: the same published 32-byte seed
# whitepaper/proof_vectors.exs uses for the Appendix D vector. NOT a Truestamp
# key, so Key Binding (E.17) reports skip against the real keyring.
seed = :binary.copy(<<0x42>>, 32)
{pub, priv} = :crypto.generate_key(:eddsa, :ed25519, seed)
kid = V.key_id(pub)
pk_b64 = Base.encode64(pub)

kat.("illustrative key_id (0x51)", kid, "f2c39df9")
kat.("illustrative pk (base64)", pk_b64, "IVL40Zt5HSRFMkLhXy6rbLfP+ntqXtMAl5YOBpiB2xI=")

proof_ts = "2026-07-24T12:05:00Z"
{:ok, proof_dt, _} = DateTime.from_iso8601(proof_ts)
proof_ts_ms = DateTime.to_unix(proof_dt, :millisecond)
block_ms = proof_ts_ms

# Observation ids are UUIDv7 (items use ULIDs). Each is minted before the
# block that commits it, so the E.20 submission window holds.
nist_id = Ids.build_uuidv7(block_ms - 300_000, 0x0A1)
stellar_id = Ids.build_uuidv7(block_ms - 240_000, 0x0B2)
bitcoin_id = Ids.build_uuidv7(block_ms - 180_000, 0x0C3)

block_id = Ids.build_uuidv7(block_ms, 0xD40)
prev_block_id = Ids.build_uuidv7(block_ms - 60_000, 0x001)
s_epoch_a = Ids.build_uuidv7(block_ms - 120_000, 0x0E0)
s_epoch_c = Ids.build_uuidv7(block_ms + 60_000, 0x0E5)
b_epoch_d = Ids.build_uuidv7(block_ms + 120_000, 0x0F1)

# ===========================================================================
# THE THREE ENTROPY OBSERVATIONS
# ===========================================================================
#
# Each s.d matches the shape internal/verify.parseEntropySubject reads, so the
# rendered report shows the pulse index / ledger sequence / block height rather
# than blanks. The observed values are deterministic digests of fixed labels
# and the indices name nothing that exists upstream: see the header.

# --- t = 30, NIST Randomness Beacon v2.0 ---
# outputValue is 64 bytes (128 hex characters), the real beacon's width.
nist_output_value =
  :crypto.hash(:sha512, "truestamp-cli entropy fixture: NIST outputValue (illustrative)")
  |> V.hex()

nist_data = %{
  "pulse" => %{
    "chainIndex" => 1,
    "pulseIndex" => 999_999_999,
    "outputValue" => nist_output_value,
    "timeStamp" => "2026-07-24T12:00:00.000Z",
    "version" => "Version 2.0"
  }
}

# --- t = 31, Stellar ledger close ---
stellar_ledger_hash =
  :crypto.hash(:sha256, "truestamp-cli entropy fixture: Stellar ledger hash (illustrative)")
  |> V.hex()

stellar_data = %{
  "hash" => stellar_ledger_hash,
  "sequence" => 4_000_000_000,
  "closed_at" => "2026-07-24T12:01:00Z"
}

# --- t = 32, Bitcoin block header ---
bitcoin_block_hash =
  :crypto.hash(:sha256, "truestamp-cli entropy fixture: Bitcoin block hash (illustrative)")
  |> V.hex()

bitcoin_data = %{
  "hash" => bitcoin_block_hash,
  "height" => 999_999,
  "time" => 1_784_894_520
}

meta_hex = V.entropy_meta_hash(%{})

observations = [
  %{name: "entropy-nist.json", t: 30, id: nist_id, d: nist_data},
  %{name: "entropy-stellar.json", t: 31, id: stellar_id, d: stellar_data},
  %{name: "entropy-bitcoin.json", t: 32, id: bitcoin_id, d: bitcoin_data}
]

observations =
  Enum.map(observations, fn o ->
    e_hex = V.entropy_hash(o.d)
    subject_hex = V.observation_hash(o.id, e_hex, meta_hex, kid)

    # E.10's entropy preimage is 121 bytes: 1 prefix + (4+36) id + (4+32)
    # entropy hash + (4+32) metadata hash + (4+4) key id. The Go verifier
    # refuses to derive a composite whose framing is any other width.
    assert.(
      byte_size(V.observation_ser(o.id, e_hex, meta_hex, kid)) + 1 == 121,
      "#{o.name}: entropy composite preimage is not E.10's 121 bytes"
    )

    Map.merge(o, %{entropy_hex: e_hex, subject_hex: subject_hex})
  end)

# ===========================================================================
# ONE BLOCK OVER ALL THREE OBSERVATIONS
# ===========================================================================
# All three fixtures are leaves of the SAME tree, so each inclusion proof is a
# real sibling path rather than a degenerate single-leaf tree.

block_leaves = Enum.map(observations, fn o -> {o.id, o.subject_hex} end)
btree = V.build(block_leaves)
merkle_root = btree.root

observations =
  Enum.map(observations, fn o ->
    pr = V.proof(btree, o.id)
    assert.(V.walk(pr, o.subject_hex) == merkle_root, "#{o.name}: inclusion proof does not reach the merkle root")
    Map.put(o, :ip, V.b64url(V.encode_proof(pr)))
  end)

block_ph =
  :crypto.hash(:sha256, "truestamp-cli entropy fixture: previous block hash (illustrative)")
  |> V.hex()

block_mh = V.block_meta_hash(%{})
block_hex = V.block_hash(block_id, block_ph, merkle_root, block_mh, kid)

# --- Stellar epoch tree: our block among a few sibling block hashes ---
stellar_epoch_leaves = [
  {s_epoch_a, V.block_hash(s_epoch_a, block_ph, V.empty_tree_root(), block_mh, kid)},
  {block_id, block_hex},
  {s_epoch_c, V.block_hash(s_epoch_c, block_hex, V.empty_tree_root(), block_mh, kid)}
]

stree = V.build(stellar_epoch_leaves)
stellar_root = stree.root
stellar_ep = V.proof(stree, block_id)
stellar_ep_b64 = V.b64url(V.encode_proof(stellar_ep))
assert.(V.walk(stellar_ep, block_hex) == stellar_root, "stellar epoch proof does not reach its root")

# --- Bitcoin epoch tree: an independent tree also containing our block ---
bitcoin_epoch_leaves = [
  {block_id, block_hex},
  {b_epoch_d, V.block_hash(b_epoch_d, block_hex, V.empty_tree_root(), block_mh, kid)}
]

btree2 = V.build(bitcoin_epoch_leaves)
bitcoin_root = btree2.root
bitcoin_ep = V.proof(btree2, block_id)
bitcoin_ep_b64 = V.b64url(V.encode_proof(bitcoin_ep))
assert.(V.walk(bitcoin_ep, block_hex) == bitcoin_root, "bitcoin epoch proof does not reach its root")

stellar_cx = %{
  "t" => 40,
  "net" => "public",
  "l" => 51_234_601,
  "tx" => String.duplicate("5e", 32),
  "memo" => stellar_root,
  "ep" => stellar_ep_b64,
  "ts" => "2026-07-24T12:05:12Z"
}

bitcoin_cx = %{
  "t" => 41,
  "net" => "mainnet",
  "h" => 870_460,
  "tx" => String.duplicate("7a", 32),
  "op" => bitcoin_root,
  "ep" => bitcoin_ep_b64,
  "ts" => "2026-07-24T12:45:00Z"
}

cx = [stellar_cx, bitcoin_cx]
root_hexes = [stellar_root, bitcoin_root]

# ===========================================================================
# SIGN EACH BUNDLE WITH ITS OWN `t`
# ===========================================================================
# The `t` code is a uint16 inside the signed payload, so the three signatures
# differ even though subject-independent inputs (kid, ts, block hash, epoch
# roots) are shared. Building the payload with a hardcoded `t` is the most
# likely bug here and is invisible to everything except a real signature check.

bundles =
  Enum.map(observations, fn o ->
    payload = V.payload(1, o.t, kid, proof_ts_ms, o.subject_hex, block_hex, root_hexes)
    p_hash = V.proof_hash(payload)
    sig = :crypto.sign(:eddsa, :none, V.unhex(p_hash), [priv, :ed25519])

    bundle = %{
      "v" => 1,
      "t" => o.t,
      "ts" => proof_ts,
      "pk" => pk_b64,
      "sig" => Base.encode64(sig),
      "s" => %{"id" => o.id, "d" => o.d, "mh" => meta_hex, "kid" => kid},
      "ip" => o.ip,
      "b" => %{
        "id" => block_id,
        "ph" => block_ph,
        "mr" => merkle_root,
        "mh" => block_mh,
        "kid" => kid
      },
      "cx" => cx
    }

    Map.put(o, :bundle, bundle)
  end)

# ===========================================================================
# SELF-CHECK: RE-DERIVE EVERY BUNDLE FROM THE EMITTED MAP ALONE
# ===========================================================================
# Every input below is read back out of the bundle map — its own ts, pk, s, b
# and cx — never from the locals that built it. This is exactly what a third
# party holding nothing but the .json file can do.

Enum.each(bundles, fn o ->
  b = o.bundle
  label = o.name

  b_pk = Base.decode64!(b["pk"])
  b_kid = V.key_id(b_pk)
  assert.(b_kid == b["s"]["kid"], "#{label}: s.kid does not match the key id derived from pk")
  assert.(b_kid == b["b"]["kid"], "#{label}: b.kid does not match the key id derived from pk")

  # E.10 entropy path: 0x21 over JCS(s.d), then the 0x23 composite.
  b_entropy = V.entropy_hash(b["s"]["d"])
  b_subject = V.observation_hash(b["s"]["id"], b_entropy, b["s"]["mh"], b["s"]["kid"])

  assert.(
    V.walk(V.decode_proof(b["ip"]), b_subject) == b["b"]["mr"],
    "#{label}: inclusion walk from the observation hash does not reach b.mr"
  )

  b_block =
    V.block_hash(b["b"]["id"], b["b"]["ph"], b["b"]["mr"], b["b"]["mh"], b["b"]["kid"])

  # E.15: dispatch on each cx entry's `t`, never on which key is present.
  b_roots =
    Enum.map(b["cx"], fn entry ->
      root = V.epoch_root(entry)
      assert.(is_binary(root), "#{label}: cx entry t=#{inspect(entry["t"])} carries no epoch root")

      assert.(
        V.walk(V.decode_proof(entry["ep"]), b_block) == root,
        "#{label}: epoch walk for cx t=#{entry["t"]} does not reach its committed root"
      )

      root
    end)

  {:ok, b_dt, _} = DateTime.from_iso8601(b["ts"])
  b_ts_ms = DateTime.to_unix(b_dt, :millisecond)
  b_payload = V.payload(b["v"], b["t"], b_kid, b_ts_ms, b_subject, b_block, b_roots)

  assert.(
    byte_size(b_payload) == 81 + 32 * length(b_roots),
    "#{label}: signature payload is not 81 + 32N bytes"
  )

  b_sig = Base.decode64!(b["sig"])

  assert.(
    :crypto.verify(:eddsa, :none, V.unhex(V.proof_hash(b_payload)), b_sig, [b_pk, :ed25519]),
    "#{label}: Ed25519 signature does not verify over the 0x61 proof hash"
  )

  # NEGATIVE: the same bundle under either other entropy type code must FAIL.
  # This is the check that catches a payload built with a hardcoded `t`, which
  # would still verify for whichever bundle happened to match it.
  Enum.each([30, 31, 32] -- [b["t"]], fn wrong_t ->
    wrong = V.payload(b["v"], wrong_t, b_kid, b_ts_ms, b_subject, b_block, b_roots)

    assert.(
      not :crypto.verify(:eddsa, :none, V.unhex(V.proof_hash(wrong)), b_sig, [b_pk, :ed25519]),
      "#{label}: signature verifies under t=#{wrong_t} as well as its own t=#{b["t"]} — the type byte is not bound into the payload"
    )
  end)

  # E.20 submission window: the observation id's ms is at or before the block's.
  assert.(
    Ids.uuidv7_time(b["s"]["id"]) <= Ids.uuidv7_time(b["b"]["id"]),
    "#{label}: submission-window ordering violation, s.id time is after b.id time"
  )

  # The observation id must be a UUIDv7 (36 characters), never a 26-character
  # ULID: items are keyed by ULID, entropy observations by UUIDv7.
  assert.(String.length(b["s"]["id"]) == 36, "#{label}: s.id is not a 36-character UUIDv7")
end)

# The three signatures must all differ: same key, same block, same epoch roots,
# different subject hash AND different type byte.
sigs = Enum.map(bundles, & &1.bundle["sig"])
assert.(length(Enum.uniq(sigs)) == 3, "the three bundles do not carry three distinct signatures")

IO.puts("SELF-VERIFY: all three entropy bundles re-derive and verify from their own emitted bytes")

# ===========================================================================
# EMIT
# ===========================================================================
# Writes go ONLY into the directory this script lives in.
out = __DIR__

assert.(
  String.ends_with?(out, Path.join(["internal", "verify", "testdata", "fixtures"])),
  "refusing to write: this script must live in internal/verify/testdata/fixtures (found #{out})"
)

Enum.each(bundles, fn o ->
  path = Path.join(out, o.name)
  File.write!(path, Jcs.encode(o.bundle))
  IO.puts("WROTE: #{path}  (t=#{o.t}, subject=#{o.subject_hex})")
end)

IO.puts("""

shared block
  id          = #{block_id}
  prev (b.ph) = #{block_ph}
  merkle root = #{merkle_root}
  block hash  = #{block_hex}
  prev block id (illustrative, not carried in the bundle) = #{prev_block_id}
epoch roots
  stellar = #{stellar_root}
  bitcoin = #{bitcoin_root}
key
  pk (base64) = #{pk_b64}
  key id      = #{kid}   (illustrative; not in Truestamp's keyring, so Key Binding reports skip)
""")
