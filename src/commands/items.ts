// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { Command, copy, createTruestampClient, readAllSync } from "../deps.ts";

import { getEnv, logSelectedOutputFormat } from "../utils.ts";

// Limit the available hash types for now to those that are supported by the browser
// and crypto.subtle.digest
// https://developer.mozilla.org/en-US/docs/Web/API/SubtleCrypto/digest#syntax
const HASH_TYPES = ['sha-1', 'sha-256', 'sha-384', 'sha-512'];

const itemsCreate = new Command()
  .description(
    `Create a new Item.

A new Item represents the hash of data that is submitted to the Truestamp API. This command allows you to provide a '--hash' and '--type' option to create a new Item. Alternatively, a file can be passed by path or piped to STDIN which will be hashed and submitted for you.

Significantly more complex Items with provenance info, metadata, digital signatures, etc. can be provided as JSON passed into the STDIN or from a file path using the '--json' option.

An Item Id will be returned in the response that identifies the Item and its temporal version. You *must* store this Item 'id' to enable verification of your data in the future.

See the examples section below for usage examples.

`,
  )
  .option(
    "-H, --hash [hash:string]",
    "An Item hash encoded as a Hex (Base16) string.",
    {
      required: false,
      conflicts: ["stdin"],
      depends: ["type"],
    },
  )
  .option(
    "-t, --type [type:string]",
    `A hash function type. Hash byte length is validated against type. Accepts ${HASH_TYPES.join(", ")}.`,
    {
      required: false,
      conflicts: ["stdin"],
      depends: ["hash"],
    },
  )
  .option(
    "-s, --stdin",
    "Read data from STDIN and submit new Item. Will be hashed unless '--json' is provided.",
    {
      conflicts: ["hash", "type"],
    }
  )
  .option(
    "-j, --json",
    "Flag to indicate STDIN or file path argument contains a JSON object with full Item data. Allows submission of arbitrarily complex Items.",
    {
      conflicts: ["hash", "type"],
    }
  )
  .arguments("[path]")
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
    "Submit from hash",
    `A new Item requires at least a 'hash' and 'type' to be provided.

Using the SHA-256 hash from the examples above:

  $ truestamp items create --hash a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e --type sha-256

  `,
  )
  .example(
    "STDIN File",
    `Pass a file to STDIN of the 'items create' command:

Pipe content to the 'items create' command using the '--stdin' option or the '-' path:

  $ echo -n 'Hello World' | truestamp items create -
  $ echo -n "Hello World" | truestamp items create --stdin

  $ cat /tmp/hello.txt | truestamp items create -
  $ cat /tmp/hello.txt | truestamp items create --stdin

  `,
  )
  .example(
    "FILE Path",
    `Pass a file path to the 'items create' command as the first argument:

  $ truestamp items create /tmp/hello.txt

  `,
  )
  .example(
    "JSON STDIN",
    `Pass raw JSON to STDIN of the 'items create' command:

Pipe JSON content to the 'items create' command using '--json' plus the '--stdin' option or the '-' path:

  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' | truestamp items create --json -
  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' | truestamp items create --json --stdin

  # Can be full complexity JSON Item data
  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' > hello.json

  $ cat /tmp/hello.json | truestamp items create --json -
  $ cat /tmp/hello.json | truestamp items create --json --stdin

  `,
  )
  .example(
    "JSON FILE Path",
    `Pass the '--json' option and a file path argument to the 'items create' command:

  $ truestamp items create --json /tmp/hello.json

  `,
  )
  .action(async (options, path: string) => {
    const ts = await createTruestampClient(options.env, options.apiKey);

    let jsonItem, altHash, altType

    // If the user provided JSON STDIN using the '--json' and '--stdin' options or the '-' or a file path argument, parse
    // the contents of STDIN and pass it directly to the Truestamp client as a complete Item object.
    if (options.json && (options.stdin || path === "-")) {
      const data = await readAllSync(Deno.stdin);
      const decoder = new TextDecoder();
      jsonItem = JSON.parse(decoder.decode(data));
    } else if (options.json && path) {
      const file = await Deno.open(path, { read: true, write: false });
      // await copy(file, Deno.stdout);

      const data = await readAllSync(file);
      const decoder = new TextDecoder();
      jsonItem = JSON.parse(decoder.decode(data));

      file.close();
    }

    // If the user provided non-JSON STDIN using the '--stdin' flag or the '-' argument, read the contents
    // of STDIN and hash it with SHA-256. If instead the user provided a path, read the contents
    // of the file and hash it with SHA-256.
    // See : https://github.com/c4spar/deno-cliffy/discussions/180
    if (!options.json && (options.stdin || path === "-")) {
      // await copy(Deno.stdin, Deno.stdout);

      const data = await readAllSync(Deno.stdin);
      const hash = new Uint8Array(await crypto.subtle.digest("SHA-256", data))
      const hashHex = Array.from(hash, (byte) => byte.toString(16).padStart(2, "0")).join("")
      // console.log(`SHA-256(stdin) = ${hashHex}`)
      altHash = hashHex
      altType = "sha-256"
    } else if (!options.json && path) {
      const file = await Deno.open(path, { read: true, write: false });
      // await copy(file, Deno.stdout);

      const data = await readAllSync(file);
      const hash = new Uint8Array(await crypto.subtle.digest("SHA-256", data))
      const hashHex = Array.from(hash, (byte) => byte.toString(16).padStart(2, "0")).join("")
      // console.log(`SHA-256(stdin) = ${hashHex}`)
      altHash = hashHex
      altType = "sha-256"

      file.close();
    }

    // If the user provided a hash and a type, validate the hash and type
    // and create the Item. If the user did not provide a hash or a type,
    // but instead had provided the '--stdin' flag or the '-' path argument,
    // or an actual path, create the Item with the hash and type from the
    // STDIN or file.
    try {
      let itemResp

      if (jsonItem) {
        itemResp = await ts.createItem(jsonItem);
      } else if (options.hash && options.type) {
        itemResp = await ts.createItem({ hash: options.hash, hashType: options.type });
      } else if (altHash && altType) {
        itemResp = await ts.createItem({ hash: altHash, hashType: altType });
      } else {
        throw new Error("No hash or hashType provided");
      }

      logSelectedOutputFormat(options, { text: itemResp.id, json: { id: itemResp.id } });
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
    const ts = await createTruestampClient(getEnv(options), options.apiKey);

    try {
      const item = await ts.getItem(options.id);

      logSelectedOutputFormat(options, { text: JSON.stringify(item, null, 2), json: item });
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

Significantly more complex Items with provenance info, metadata, digital signatures, etc. can be provided as JSON passed into the STDIN or from a file path using the '--json' option.

Updates are always non-destructive. Older versions are preserved and can be retrieved with their Id.

Any Item attributes that existed on an older version which are not provided again will be removed from the new version.

An *new* Item Id will be returned in the response that identifies the Item and its new temporal version. You *must* store this Item 'id' to enable verification of your data in the future. The Id provided as an argument remains valid and it's content unchanged.

See the examples section below for usage examples.

`
  )
  .option("-i, --id [id:string]", "An Item Id to update.", {
    required: true,
  })
  .option(
    "-H, --hash [hash:string]",
    "An Item hash encoded as a Hex (Base16) string.",
    {
      required: false,
      conflicts: ["stdin"],
      depends: ["type"],
    },
  )
  .option(
    "-t, --type [type:string]",
    `A hash function type. Hash byte length is validated against type. Accepts ${HASH_TYPES.join(", ")}.`,
    {
      required: false,
      conflicts: ["stdin"],
      depends: ["hash"],
    },
  )
  .option(
    "-s, --stdin",
    "Read data from STDIN and submit updated Item. Will be hashed unless '--json' is provided.",
    {
      conflicts: ["hash", "type"],
    }
  )
  .option(
    "-j, --json",
    "Flag to indicate STDIN or file path argument contains a JSON object with full Item data. Allows update with arbitrarily complex Items.",
    {
      conflicts: ["hash", "type"],
    }
  )
  .arguments("[path]")
  .example(
    "Update an Item",
    `Using a previously generated test Id, and a new hash:

  $ echo -n 'Hello World Again' | sha256sum
  63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6  -

  $ truestamp items update --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF --hash 63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6 --type sha-256

  `,
  )
  .example(
    "STDIN File",
    `Pass a file from STDIN to the 'items update' command:

Pipe content to the 'items update' command using the '--stdin' option or the '-' path:

  $ echo -n 'Hello World' | truestamp items update --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF -
  $ echo -n "Hello World" | truestamp items update --stdin --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF

  $ cat /tmp/hello.txt | truestamp items update --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF -
  $ cat /tmp/hello.txt | truestamp items update --stdin --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF

  `,
  )
  .example(
    "FILE Path",
    `Pass an '--id' and a file path to the 'items update' command:

  $ truestamp items update --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF /tmp/hello.txt

  `,
  )
  .example(
    "JSON STDIN",
    `Pass raw JSON to STDIN of the 'items update' command:

Pipe JSON content to the 'items update' command using '--json' plus the '--stdin' option or the '-' path:

  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' | truestamp items update --json --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF -
  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' | truestamp items update --json --stdin --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF

  # Can be full complexity JSON Item data
  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' > hello.json

  $ cat /tmp/hello.json | truestamp items update --json --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF -
  $ cat /tmp/hello.json | truestamp items update --json --stdin --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF

  `,
  )
  .example(
    "JSON FILE Path",
    `Pass the '--json' option and a file path argument to the 'items update' command:

  $ truestamp items update --json --id T11_01FZGTTP1JRQ40PD99GGJK07YS_1648758708880000_A523F596217460983B658A78B5E7AACF /tmp/hello.json

  `,
  )
  .action(async (options, path: string) => {
    const ts = await createTruestampClient(options.env, options.apiKey);

    let jsonItem, altHash, altType

    // If the user provided JSON STDIN using the '--json' and '--stdin' options or the '-' or a file path argument, parse
    // the contents of STDIN and pass it directly to the Truestamp client as a complete Item object.
    if (options.json && (options.stdin || path === "-")) {
      const data = await readAllSync(Deno.stdin);
      const decoder = new TextDecoder();
      jsonItem = JSON.parse(decoder.decode(data));
    } else if (options.json && path) {
      const file = await Deno.open(path, { read: true, write: false });
      // await copy(file, Deno.stdout);

      const data = await readAllSync(file);
      const decoder = new TextDecoder();
      jsonItem = JSON.parse(decoder.decode(data));

      file.close();
    }

    // If the user provided non-JSON STDIN using the '--stdin' flag or the '-' argument, read the contents
    // of STDIN and hash it with SHA-256. If instead the user provided a path, read the contents
    // of the file and hash it with SHA-256.
    // See : https://github.com/c4spar/deno-cliffy/discussions/180
    if (!options.json && (options.stdin || path === "-")) {
      // await copy(Deno.stdin, Deno.stdout);

      const data = await readAllSync(Deno.stdin);
      const hash = new Uint8Array(await crypto.subtle.digest("SHA-256", data))
      const hashHex = Array.from(hash, (byte) => byte.toString(16).padStart(2, "0")).join("")
      // console.log(`SHA-256(stdin) = ${hashHex}`)
      altHash = hashHex
      altType = "sha-256"
    } else if (!options.json && path) {
      const file = await Deno.open(path, { read: true, write: false });
      // await copy(file, Deno.stdout);

      const data = await readAllSync(file);
      const hash = new Uint8Array(await crypto.subtle.digest("SHA-256", data))
      const hashHex = Array.from(hash, (byte) => byte.toString(16).padStart(2, "0")).join("")
      // console.log(`SHA-256(stdin) = ${hashHex}`)
      altHash = hashHex
      altType = "sha-256"

      file.close();
    }

    try {
      let itemResp

      if (jsonItem) {
        itemResp = await ts.updateItem(options.id, jsonItem);
      } else if (options.hash && options.type) {
        itemResp = await ts.updateItem(options.id, { hash: options.hash, hashType: options.type });
      } else if (altHash && altType) {
        itemResp = await ts.updateItem(options.id, { hash: altHash, hashType: altType });
      } else {
        throw new Error("No hash or hashType provided");
      }

      logSelectedOutputFormat(options, { text: itemResp.id, json: { id: itemResp.id } });
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
//     const ts = await createTruestampClient(getEnv(options), options.apiKey);
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
//     const ts = await createTruestampClient(getEnv(options), options.apiKey);
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
