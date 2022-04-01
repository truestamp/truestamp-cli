// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { Command, createTruestampClient } from "../deps.ts";

// Limit the available hash types for now to those that are supported by the browser
// and crypto.subtle.digest
// https://developer.mozilla.org/en-US/docs/Web/API/SubtleCrypto/digest#syntax
const HASH_TYPES = ['sha-1', 'sha-256', 'sha-384', 'sha-512'];

const itemsCreate = new Command()
  .description(
    `Create a new Item.

Provide a '--hash' in Hex (Base16) encoding.

Hashes must be accompanied by the hash '--type' which must be one of:

  ${HASH_TYPES.join("\n  ")}

This command will return a JSON result by default that contains an
'Envelope' structure that wraps the Item you submitted. The 'Envelope'
structure contains the Item ID, and other metadata.

You *must* store the Item 'id' in a secure location to be able to use
it to verify your data later.

`,
  )
  .option(
    "-H, --hash [hash:string]",
    "An Item hash encoded as a Hex (Base16) string.",
    {
      required: true,
    },
  )
  .option(
    "-t, --type [type:string]",
    "A hash function type from the SHA, or SHA2 families. The byte length of the hash will be validated against the type.",
    {
      required: true,
    },
  )
  .example(
    "SHA-256 : openssl",
    `Generate the hash of the text 'Hello World' using openssl

Most systems have access to the openssl command line tool. You can
get the hash of a string of text or file (note single quotes) using:

  # create a test file
  $ echo -n 'Hello World' > hello.txt

  # get the hash of the file
  $ openssl dgst -sha256 hello.txt
  SHA256(hello.txt)= a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e

  # get the hash of the text directly
  $ echo -n 'Hello World' | openssl dgst -sha256 -hex
  a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e

`,
  )
  .example(
    "SHA-256 : sha256sum",
    `Generate the hash of the text 'Hello World' using sha256sum

Many systems have access to the sha256sum command line tool. You can
get the hash of a string of text or file (note single quotes) using:

  # create a test file
  $ echo -n 'Hello World' > hello.txt

  # get the hash of the file
  $ sha256sum -b hello.txt
  a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e *hello.txt

  # get the hash of the text directly
  $ echo -n 'Hello World' | sha256sum
  a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e  -

You can install the sha256sum command on macOS with Homebrew:

  # Install : https://brew.sh/
  $ brew install coreutils

`,
  )
  .example(
    "SHA-256 : browser",
    `Generate the hash of a file in web browser

There are many online tools that will generate a hash of a file. Here
is a pretty simple to use example:

  # Type or paste some text:
  https://emn178.github.io/online-tools/sha256.html

  # Drag and drop a file:
  https://emn178.github.io/online-tools/sha256_checksum.html

`,
  )
  .example(
    "Submit an Item",
    `A new Item requires at least a 'hash' and 'type' to be provided.

Using the SHA-256 hash from the examples above:

  $ truestamp items create --hash a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e --type sha-256

  `,
  )
  .action(async (options) => {
    const ts = await createTruestampClient(options.env, options.apiKey);

    try {
      const item = await ts.createItem({
        hash: options.hash,
        hashType: options.type,
      });

      console.log(JSON.stringify(item));
    } catch (error) {
      const { key, value, type, response } = error

      if (key || value || type) {
        // is a StructError
        if (value === undefined) {
          throw new Error(`attribute ${key} is required`)
        } else if (type === 'never') {
          throw new Error(`attribute ${key} is unknown`)
        } else {
          throw new Error(`${key} ${value} is invalid`)
        }
      } else if (response) {
        // is a HTTPResponseError
        // This is a custom error type thrown by truestamp-js and
        // has a 'response' property which can be awaited to get the full
        // HTTP response, including body with error info.
        // https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#body
        const { status, code, description } = await response.json()
        throw new Error(`${status} : ${code} : ${description}`)
      } else {
        // is a generic Error
        throw error
      }
    }
  });

const itemsRead = new Command()
  .description("Read an existing Item.")
  .option("-i, --id [id:string]", "An Item Id.", {
    required: true,
  })
  .example(
    "Read an Item",
    `Using a previously generated test ID:

  $ truestamp items read --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF

`,
  )
  .action(async (options) => {
    const ts = await createTruestampClient(options.env, options.apiKey);

    try {
      const item = await ts.getItem(options.id);
      console.log(JSON.stringify(item));
    } catch (error) {
      // throw new Error(`Item not found : ${error.message}`);
      const { key, value, type, response } = error

      if (key || value || type) {
        // is a StructError
        if (value === undefined) {
          throw new Error(`attribute ${key} is required`)
        } else if (type === 'never') {
          throw new Error(`attribute ${key} is unknown`)
        } else {
          throw new Error(`${key} ${value} is invalid`)
        }
      } else if (response) {
        // is a HTTPResponseError
        // This is a custom error type thrown by truestamp-js and
        // has a 'response' property which can be awaited to get the full
        // HTTP response, including body with error info.
        // https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#body
        const { status, code, description } = await response.json()
        throw new Error(`${status} : ${code} : ${description}`)
      } else {
        // is a generic Error
        throw error
      }
    }

  });

const itemsUpdate = new Command()
  .description(
    `Update an existing Item by replacing it with a new one.

Updating an Item returns an updated Id to reflect the update time.

Updates are always non-destructive. Older versions are preserved and can be retrieved with their Id.

Any Item attributes that existed on an older version which are not provided again will be removed from the new version.

Provide an Item '--id' to identify the specific Item you are authorized to update.

Provide an Item '--hash' in Hex (Base16) encoding.

Hash digests must be accompanied by the hash '--type' which must be one of:

    ${HASH_TYPES.join("\n  ")}

This command will return a JSON result by default that contains an
'Envelope' structure that wraps the Item you submitted. The 'Envelope'
structure contains the Item ID, and other metadata.

You *must* store the Item ID in a secure location to be able to use
it to verify your data later.

`
  )
  .option("-i, --id [id:string]", "An Item Id to update.", {
    required: true,
  })
  .option(
    "-H, --hash [hash:string]",
    "An Item hash encoded as a Hex (Base16) string.",
    {
      required: true,
    },
  )
  .option(
    "-t, --type [type:string]",
    "The hash function type used for 'hash'. Must be from the SHA, or SHA2 families. The hash byte length will be validated for type.",
    {
      required: true,
    },
  )
  .example(
    "Update an Item",
    `Using a previously generated test ID:

  $ echo -n 'Hello World Again' | sha256sum
  63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6  -

  $ truestamp items update --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF --hash a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e --type sha-256

  `,
  )
  .action(async (options) => {
    const ts = await createTruestampClient(options.env, options.apiKey);

    try {
      const item = await ts.updateItem(options.id, {
        hash: options.hash,
        hashType: options.type,
      });
      console.log(JSON.stringify(item));
    } catch (error) {
      // throw new Error(`Item update error : ${error.message}`);
      const { key, value, type, response } = error

      if (key || value || type) {
        // is a StructError
        if (value === undefined) {
          throw new Error(`attribute ${key} is required`)
        } else if (type === 'never') {
          throw new Error(`attribute ${key} is unknown`)
        } else {
          throw new Error(`${key} ${value} is invalid`)
        }
      } else if (response) {
        // is a HTTPResponseError
        // This is a custom error type thrown by truestamp-js and
        // has a 'response' property which can be awaited to get the full
        // HTTP response, including body with error info.
        // https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#body
        const { status, code, description } = await response.json()
        throw new Error(`${status} : ${code} : ${description}`)
      } else {
        // is a generic Error
        throw error
      }
    }
  });

// const itemsDelete = new Command()
//   .description("Delete an existing document.")
//   .option("-i, --id [id:string]", "A document ID.", {
//     required: true,
//   })
//   .action(async (options) => {
//     const ts = await createTruestampClient(options.env, options.apiKey);
//     let d;
//     try {
//       d = await ts.deleteDocument(options.id);
//     } catch (error) {
//       throw new Error(`document not found : ${error.message}`);
//     }

//     console.log(JSON.stringify(d));
//   });

// const itemsList = new Command()
//   .description("List all existing documents.")
//   .action(async (options) => {
//     const ts = await createTruestampClient(options.env, options.apiKey);
//     let d;
//     try {
//       d = await ts.getAllDocuments();
//     } catch (error) {
//       throw new Error(`documents not found : ${error.message}`);
//     }

//     console.log(JSON.stringify(d));
//   });

export const items = new Command()
  .description("Create, read, or update Items.")
  .action(() => {
    items.showHelp();
  })
  .command("create", itemsCreate)
  .command("read", itemsRead)
  .command("update", itemsUpdate)
  // .command("delete", itemsDelete)
  // .command("list", itemsList);
