// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

import { Command, createTruestampClient } from "../deps.ts";

interface HashType {
  minBytes: number
  maxBytes: number
}

interface HashTypes {
  [key: string]: HashType
}

const HASH_TYPES: HashTypes = {
  sha1: { minBytes: 20, maxBytes: 20 },
  'sha2-256': { minBytes: 32, maxBytes: 32 },
  'sha2-384': { minBytes: 48, maxBytes: 48 },
  'sha2-512': { minBytes: 64, maxBytes: 64 },
  'sha3-256': { minBytes: 32, maxBytes: 32 },
  'sha3-384': { minBytes: 48, maxBytes: 48 },
  'sha3-512': { minBytes: 64, maxBytes: 64 },
  'blake-224': { minBytes: 28, maxBytes: 28 },
  'blake-256': { minBytes: 32, maxBytes: 32 },
  'blake-384': { minBytes: 48, maxBytes: 48 },
  'blake-512': { minBytes: 64, maxBytes: 64 },
  blake2b: { minBytes: 1, maxBytes: 64 }, // variable length type
  'blake2b-224': { minBytes: 28, maxBytes: 28 },
  'blake2b-256': { minBytes: 32, maxBytes: 32 },
  'blake2b-384': { minBytes: 48, maxBytes: 48 },
  'blake2b-512': { minBytes: 64, maxBytes: 64 },
  blake2s: { minBytes: 1, maxBytes: 32 }, // variable length type
  'blake2s-224': { minBytes: 28, maxBytes: 28 },
  'blake2s-256': { minBytes: 32, maxBytes: 32 },
  'blake2s-384': { minBytes: 48, maxBytes: 48 },
  'blake2s-512': { minBytes: 64, maxBytes: 64 },
  blake3: { minBytes: 1, maxBytes: 64 }, // variable length type, accept max 64 bytes here
  'blake3-224': { minBytes: 28, maxBytes: 28 },
  'blake3-256': { minBytes: 32, maxBytes: 32 },
  'blake3-384': { minBytes: 48, maxBytes: 48 },
  'blake3-512': { minBytes: 64, maxBytes: 64 },
}

const documentsNew = new Command()
  .description(
    `Create a new document.
  
  Provide a '--hash' digest in Hex (Base16), or Base64 (UrlSafe | UrlUnsafe) encoding.

  Hash digests must be accompanied by the hash '--type' which must be one of:
  
  ${Object.keys(HASH_TYPES).join('\n  ')}

  The 'SHA1' type is assumed to always be 160 bits (20 bytes). The 'BLAKEx' types
  without a bit length are provided as a convenience to allow for variable
  length digests (1 - 64 bytes). These are discouraged as there is then no way to know
  the actual digest length when needing to re-compute it later.

  An optional short text '--description' of the document can also be provided.
  `,
  )
  .option(
    "-H, --hash [hash:string]",
    "A document hash digest encoded as a Hex (Base16) or Base64 (UrlSafe | UrlUnsafe) string.",
    {
      required: true,
    },
  )
  .option(
    "-t, --type [type:string]",
    "A hash function type from the SHA, SHA2, SHA3, BLAKE, BLAKE2, or BLAKE3 families. The byte length of the hash will be validated against the type.",
    {
      required: true,
    },
  )
  .option(
    "-d, --description [description:string]",
    "An optional description of the Document (Max length 256 characters).",
    {
      required: false,
      default: "",
    },
  )
  .action(async (options) => {
    if (options.type && !HASH_TYPES[options.type.toLowerCase()]) {
      console.error("unsupported hash type");
      Deno.exit(1);
    }

    if (options.description && options.description.length > 256) {
      console.error("description is too long (Max length 256)");
      Deno.exit(1);
    }

    const ts = await createTruestampClient(options.env);
    let d
    try {
      d = await ts.createDocument({
        hash: options.hash,
        hashType: options.type,
        description: options.description,
      });
    } catch (error) {
      throw new Error(`new document creation failed : ${error.message}`);
    }

    console.log(JSON.stringify(d));
  });

const documentsShow = new Command()
  .description("Show an existing document.")
  .option("-i, --id [id:string]", "A document ID.", {
    required: true,
  })
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    let d
    try {
      d = await ts.getDocument(options.id);
    } catch (error) {
      throw new Error(`document not found : ${error.message}`);
    }

    console.log(JSON.stringify(d));
  });

const documentsUpdate = new Command()
  .description(
    `Update an existing document. This will replace the existing document with a new one under the same ID.

    Any attributes that are not provided will be removed from the new version.

    Provide an '--id' to identify the specific Document you are authorized to update.

    Provide a '--hash' digest in Hex (Base16), or Base64 (UrlSafe | UrlUnsafe) encoding.

    Hash digests must be accompanied by the hash '--type' which must be one of:
    
    ${Object.keys(HASH_TYPES).join('\n  ')}
  
    The 'SHA1' type is assumed to always be 160 bits (20 bytes). The 'BLAKEx' types
    without a bit length are provided as a convenience to allow for variable
    length digests (1 - 64 bytes). These are discouraged as there is then no way to know
    the actual digest length when needing to re-compute it later.
  
    An optional short text '--description' of the document can also be provided.
    `,
  )
  .option("-i, --id [id:string]", "A document ID.", {
    required: true,
  })
  .option(
    "-H, --hash [hash:string]",
    "A document hash digest encoded as a Hex (Base16) or Base64 (UrlSafe | UrlUnsafe) string.",
    {
      required: true,
    },
  )
  .option(
    "-t, --type [type:string]",
    "A hash function type from the SHA, SHA2, SHA3, BLAKE, BLAKE2, or BLAKE3 families. The byte length of the hash will be validated against the type.",
    {
      required: true,
    },
  )
  .option(
    "-d, --description [description:string]",
    "An optional description of the Document (Max length 256 characters).",
    {
      required: false,
      default: "",
    },
  )
  .action(async (options) => {
    if (options.type && !HASH_TYPES[options.type.toLowerCase()]) {
      console.error("unsupported hash type");
      Deno.exit(1);
    }

    if (options.description && options.description.length > 256) {
      console.error("description is too long (Max length 256)");
      Deno.exit(1);
    }

    const ts = await createTruestampClient(options.env);
    let d
    try {
      d = await ts.updateDocument(options.id, {
        hash: options.hash,
        hashType: options.type,
        description: options.description,
      });
    } catch (error) {
      throw new Error(`document update error : ${error.message}`);
    }

    console.log(JSON.stringify(d));
  });

const documentsDelete = new Command()
  .description("Delete an existing document.")
  .option("-i, --id [id:string]", "A document ID.", {
    required: true,
  })
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    let d
    try {
      d = await ts.deleteDocument(options.id);
    } catch (error) {
      throw new Error(`document not found : ${error.message}`);
    }

    console.log(JSON.stringify(d));
  });

const documentsList = new Command()
  .description("List all existing documents.")
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    let d
    try {
      d = await ts.getAllDocuments();
    } catch (error) {
      throw new Error(`documents not found : ${error.message}`);
    }

    console.log(JSON.stringify(d));
  });

export const documents = new Command()
  .description("Create, read, update, or destroy documents.")
  .action(() => {
    documents.showHelp();
  })
  .command("new", documentsNew)
  .command("show", documentsShow)
  .command("update", documentsUpdate)
  .command("delete", documentsDelete)
  .command("list", documentsList);
