// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { z } from "zod";

const OutputWrapper = z.object({
  text: z.string(),
  json: z.record(z.string().min(1), z.any()),
});

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

export const RFC7807ErrorSchema = z.object({
  status: z.number(),
  type: z.optional(z.string().url()),
  title: z.optional(z.string()),
  detail: z.optional(z.string()),
  instance: z.optional(z.string().url()),
});

export type RFC7807Error = z.infer<typeof RFC7807ErrorSchema>;

// RFC7807 Problem Details for HTTP APIs
// See : https://github.com/PDMLab/http-problem-details
export function throwApiError(
  errorMsgPrefix: string,
  errorData?: RFC7807Error,
): void {
  const result = RFC7807ErrorSchema.safeParse(errorData);

  // Unknown/missing error format
  if (!result.success) {
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

  // RFC7807 error
  const { title, detail, status, type, instance } = result.data;
  throw new Error(
    `${errorMsgPrefix} : ${title} ${detail} [${status}, ${type}, ${instance}]`,
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
