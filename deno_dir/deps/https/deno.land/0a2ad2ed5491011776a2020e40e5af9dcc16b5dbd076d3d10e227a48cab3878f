import { blobReader } from "../chunked-blob-reader/mod.ts";
import { HashConstructor, StreamHasher } from "../types/mod.ts";

export const blobHasher: StreamHasher<Blob> = async function blobHasher(
  hashCtor: HashConstructor,
  blob: Blob
): Promise<Uint8Array> {
  const hash = new hashCtor();

  await blobReader(blob, (chunk) => {
    hash.update(chunk);
  });

  return hash.digest();
};
