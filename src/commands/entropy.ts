import { Command, decodeBase64, encodeBase64 } from "../deps.ts"

import { verify as verifyEd25519 } from "@stablelib/ed25519"
import { encode as encodeHex } from "@stablelib/hex"
import { hash } from "@stablelib/sha256"
import { canonify } from "@truestamp/canonify"

import { digestMessage, get, logSelectedOutputFormat } from "../utils.ts"

import { EntropyResponse, SignedKey } from "../types.ts"

import { environmentType, outputType } from "../cli.ts"

const ENTROPY_BASE_URL = "https://entropy-v2.truestamp.com"
const PUBLIC_KEYS_BASE_URL = "https://keys.truestamp.com"

function generatePublicKeyHandle(publickey: Uint8Array): string {
  return encodeHex(hash(publickey)).slice(0, 8).toLowerCase()
}

async function verifyPublicKey(publickey: Uint8Array): Promise<boolean> {
  try {
    const handle = generatePublicKeyHandle(publickey)
    const keyObj = await get<SignedKey>(`${PUBLIC_KEYS_BASE_URL}/${handle}`)

    if (!keyObj) {
      return false
    }

    SignedKey.parse(keyObj)

    const { publicKey: foundPublicKey } = keyObj
    if (foundPublicKey !== encodeBase64(publickey)) {
      return false
    }

    // OK
    return true
  } catch (error) {
    console.error(error.message)
    return false
  }
}

async function verifyEntropy(entropy: EntropyResponse): Promise<boolean> {
  const { data, hash, publicKey, signature } = entropy

  const publicKeyBytes = decodeBase64(publicKey)

  const isValidRemotePublicKey = await verifyPublicKey(publicKeyBytes)

  if (!isValidRemotePublicKey) {
    throw new Error("signature : public key fetch from remote keyserver failed")
  }

  const canonicalData = await canonify(data)

  if (!canonicalData) {
    throw new Error("signature : invalid canonical data")
  }

  const canonicalDataHash = await digestMessage(canonicalData)

  if (hash !== encodeHex(canonicalDataHash, true)) {
    throw new Error("signature : entropy hash mismatch")
  }

  const signatureBytes = decodeBase64(signature)

  const isValid = verifyEd25519(
    publicKeyBytes,
    canonicalDataHash,
    signatureBytes
  )

  if (!isValid) {
    throw new Error("signature : invalid signature")
  }

  return true
}

export const entropyCommand = new Command<{
  env: typeof environmentType
  apiKey?: string
  output: typeof outputType
}>()
  .description(
    `Fetch and verify the latest, or historical, observable entropy.

    Learn more about the observable entropy project at:
    https://github.com/truestamp/observable-entropy-worker-v2

    `
  )
  .example(
    "Fetch latest",
    `Call the 'entropy' command with no arguments to fetch the latest entropy:

By default this will log to the console the latest entropy hash, along with the timestamp
of when it was captured in UTC. The entropy hash is a SHA256 hash of the entropy data, and
can be used to later retrieve and verify the entropy data.

The entropy signature is always verified.

$ truestamp entropy
ab84d4760579bef4ecebe2d2e50e7a42d57e2650894bfbd5749e5e2551892e2e [2022-11-19T19:25:22.519Z]

You can call this with the '--output json' option to get the full entropy object:

$ truestamp entropy --output json
`
  )
  .example(
    "Fetch by hash",
    `Call 'entropy' with a single entropy hash argument representing the entropy to fetch:

This is useful to fetch, and verify, entropy from a previous date.

The entropy signature is always verified.

You can call this with the '--output json' option to get the full entropy object:

$ truestamp entropy ab84d4760579bef4ecebe2d2e50e7a42d57e2650894bfbd5749e5e2551892e2e --output json
`
  )
  .arguments("[entropy-hash] [hash:string]")
  .action(async (options, hash?: string) => {
    const entropy: EntropyResponse = await get<EntropyResponse>(
      `${ENTROPY_BASE_URL}/${hash ?? "latest"}`
    )

    // throws if invalid
    await verifyEntropy(EntropyResponse.parse(entropy))

    if (entropy) {
      logSelectedOutputFormat(
        {
          text: `${entropy.hash} [${entropy.data.timestamp.capturedAt}]`,
          json: entropy,
        },
        options.output
      )
    }
  })
