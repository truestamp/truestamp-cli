import { Buffer } from "https://deno.land/std@0.101.0/node/buffer.ts";
import { lstatSync } from "https://deno.land/std@0.101.0/node/fs.ts";

export function calculateBodyLength(body: any): number | undefined {
  if (!body) {
    return 0;
  }
  if (typeof body === "string") {
    return Buffer.from(body).length;
  } else if (typeof body.byteLength === "number") {
    // handles Uint8Array, ArrayBuffer, Buffer, and ArrayBufferView
    return body.byteLength;
  } else if (typeof body.size === "number") {
    return body.size;
  } else if (typeof body.path === "string") {
    // handles fs readable streams
    return lstatSync(body.path).size;
  }
}
