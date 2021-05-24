import { isArrayBuffer } from "../is-array-buffer/mod.ts";
import { Buffer } from "https://deno.land/std@0.93.0/node/buffer.ts";

export const fromArrayBuffer = (input: ArrayBuffer, offset = 0, length: number = input.byteLength - offset): Buffer => {
  if (!isArrayBuffer(input)) {
    throw new TypeError(`The "input" argument must be ArrayBuffer. Received type ${typeof input} (${input})`);
  }

  return Buffer.from(input, offset, length);
};

export type StringEncoding = "ascii" | "utf8" | "utf16le" | "ucs2" | "base64" | "latin1" | "binary" | "hex";

export const fromString = (input: string, encoding?: StringEncoding): Buffer => {
  if (typeof input !== "string") {
    throw new TypeError(`The "input" argument must be of type string. Received type ${typeof input} (${input})`);
  }

  return encoding ? Buffer.from(input, encoding) : Buffer.from(input);
};
