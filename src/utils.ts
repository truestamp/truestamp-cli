// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import {
  extractPublicKeyFromSecretKey,
  generateKeyPair,
  sign,
} from "@stablelib/ed25519";
import { hash as sha256 } from "@stablelib/sha256";
import { z } from "zod";

import { canonify } from "@truestamp/canonify";

import {
  decode as base64Decode,
  encode as base64Encode,
} from "@stablelib/base64";

import { getConfigKeyForEnv, setConfigKeyForEnv } from "./config.ts";

const OutputWrapper = z.object({
  text: z.string(),
  json: z.record(z.string().min(1), z.any()),
});

import type { ItemRequest, Signature } from "./types.ts";

export type OutputWrapper = z.infer<typeof OutputWrapper>;

export function logSelectedOutputFormat(
  data: OutputWrapper,
  outputFormat: "text" | "json" | "silent",
): void {
  switch (outputFormat) {
    case "text":
      console.log(data.text);
      break;
    case "json":
      console.log(JSON.stringify(data.json, null, 2));
      break;
    case "silent":
      // no-op : silent
      break;
  }
}

// export const RFC7807ErrorSchema = z.object({
//   status: z.number(),
//   type: z.optional(z.string().url()),
//   title: z.optional(z.string()),
//   detail: z.optional(z.string()),
//   instance: z.optional(z.string().url()),
// });

// export type RFC7807Error = z.infer<typeof RFC7807ErrorSchema>;

// // RFC7807 Problem Details for HTTP APIs
// // See : https://github.com/PDMLab/http-problem-details
// export function throwApiError(
//   errorMsgPrefix: string,
//   errorData?: RFC7807Error,
// ): void {
//   const result = RFC7807ErrorSchema.safeParse(errorData);

//   // Unknown/missing error format
//   if (!result.success) {
//     throw new Error(
//       `${errorMsgPrefix} : invalid error response :\n${
//         JSON.stringify(
//           errorData,
//           null,
//           2,
//         )
//       }`,
//     );
//   }

//   // RFC7807 error
//   const { title, detail, status, type, instance } = result.data;
//   throw new Error(
//     `${errorMsgPrefix} : ${title} ${detail} [${status}, ${type}, ${instance}]`,
//   );
// }

export function throwApiError(
  errorMsgPrefix: string,
  errorData?: any,
): void {
  throw new Error(
    `${errorMsgPrefix} : invalid error response :\n${
      JSON.stringify(
        errorData,
        null,
        2,
      )
    }`,
  );
}

export async function digestMessage(message: string): Promise<Uint8Array> {
  const msgUint8 = new TextEncoder().encode(message); // encode as (utf-8) Uint8Array
  const hashBuffer = await crypto.subtle.digest("SHA-256", msgUint8); // hash the message
  return new Uint8Array(hashBuffer);
}

// Typed Fetch w/ Timeouts
// See : https://eckertalex.dev/blog/typescript-fetch-wrapper
// See : // https://medium.com/deno-the-complete-reference/fetch-timeout-in-deno-91731bca80a1
async function http<T>(path: string, config: RequestInit): Promise<T> {
  const REQUEST_TIMEOUT = 10000; // 10 seconds

  try {
    const req = new Request(path, config);

    const c = new AbortController();
    const id = setTimeout(() => c.abort(), REQUEST_TIMEOUT);
    const res = await fetch(req, { signal: c.signal });
    clearTimeout(id);

    if (!res.ok) {
      throw new Error(`${res.status} ${res.statusText}`);
    }

    return res.json() as T;
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      throw new Error(`fetch timeout : ${err.message}`);
    } else {
      throw new Error(`fetch error : ${err.message}`);
    }
  }
}

export async function get<T>(path: string, config?: RequestInit): Promise<T> {
  const init = {
    method: "GET",
    headers: { Accept: "application/json" },
    ...config,
  };
  return await http<T>(path, init);
}

export async function post<T, U>(
  path: string,
  body: T,
  config?: RequestInit,
): Promise<U> {
  const init = {
    method: "POST",
    body: JSON.stringify(body),
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    ...config,
  };
  return await http<U>(path, init);
}

export async function put<T, U>(
  path: string,
  body: T,
  config?: RequestInit,
): Promise<U> {
  const init = {
    method: "PUT",
    body: JSON.stringify(body),
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    ...config,
  };
  return await http<U>(path, init);
}

export function signItemData(
  itemRequest: ItemRequest,
  secretKey: Uint8Array | string,
): Signature[] {
  if (typeof secretKey === "string") {
    secretKey = base64Decode(secretKey);
  }

  const publicKey = extractPublicKeyFromSecretKey(secretKey);

  const canonicalData = canonify(itemRequest.itemData);
  const canonicalDataUint8Array = new TextEncoder().encode(canonicalData);
  const canonicalDataHash = sha256(canonicalDataUint8Array);

  const canonicalDataHashSignature = sign(
    secretKey,
    canonicalDataHash,
  );

  return [
    {
      publicKey: base64Encode(publicKey),
      signature: base64Encode(canonicalDataHashSignature),
      signatureType: "ed25519",
    },
  ];
}

export function getSigningSecretKeyForEnv(env: string): Uint8Array {
  const configSecretKeyPropertyName = "signing_secret_key";
  const configPublicKeyPropertyName = "signing_public_key";

  const secretKey = getConfigKeyForEnv(
    env,
    configSecretKeyPropertyName,
  );

  if (secretKey) {
    return base64Decode(secretKey as string);
  } else {
    // Initialize and save a new KeyPair to config
    const keyPair = generateKeyPair();
    setConfigKeyForEnv(
      env,
      configSecretKeyPropertyName,
      base64Encode(keyPair.secretKey),
    );
    setConfigKeyForEnv(
      env,
      configPublicKeyPropertyName,
      base64Encode(keyPair.publicKey),
    );

    return keyPair.secretKey;
  }
}
