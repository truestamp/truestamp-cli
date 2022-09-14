// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import {
  Command,
  createTruestampClient,
  EnumType,
  parse as pathParse,
  readAllSync,
  ValidationError,
} from "../deps.ts";

import { logSelectedOutputFormat } from "../utils.ts";

import { createDataDir, writeItemToDb } from "../db.ts";

import { environmentType, outputType } from "../cli.ts";

export const inputType = new EnumType(["binary", "json"]);

// Limit the available hash types for now to those that are supported by the browser
// and crypto.subtle.digest
// https://developer.mozilla.org/en-US/docs/Web/API/SubtleCrypto/digest#syntax
const HASH_TYPES = ["sha-1", "sha-256", "sha-384", "sha-512"];

type HashTypes = "sha1" | "sha-256" | "sha-384" | "sha-512";

const itemsCreate = new Command<
  {
    env: typeof environmentType;
    apiKey?: string;
    output: typeof outputType;
  }
>()
  .description(
    `Create a new Item.

A new Item represents the hash of data that is submitted to the Truestamp API. This command allows you to provide a '--hash' and '--hash-type' option to create a new Item. Alternatively, a file can be passed by path or piped to STDIN which will be hashed and submitted for you.

Significantly more complex Items with provenance info, metadata, digital signatures, etc. can be provided as JSON passed into the STDIN or from a file path using the '--input json' option.

An Item Id will be returned in the response that identifies the Item and its temporal version. This Id will be stored in a local database for convenience, but you may also store this Item 'id' elsewhere to enable verification of your data in the future.

See the example sections below for detailed usage examples.

`,
  )
  .option(
    "-H, --hash <hash:string>",
    "An Item hash encoded as a Hex (Base16) string.",
    {
      required: false,
      conflicts: ["stdin", "input"],
      depends: ["hash-type"],
    },
  )
  .option(
    "-t, --hash-type <hashType:string>",
    `A hash function type. Hash byte length is validated against type. Accepts ${
      HASH_TYPES.join(", ")
    }.`,
    {
      required: false,
      conflicts: ["stdin", "input"],
      depends: ["hash"],
    },
  )
  .option(
    "-s, --stdin",
    "Read data from STDIN and submit new Item. Will be hashed unless '--input json' is provided.",
    {
      conflicts: ["hash", "hash-type"],
      depends: ["input"],
    },
  )
  .type("input", inputType)
  .option(
    "--input <input:input>",
    "Input format. If 'binary' is provided the file will be hashed and a new Item constructed and submitted. If 'json' is provided the file not be hashed and will be submitted as-is. JSON content must match the structure expected by the API in all cases. If submitting arbitrary JSON files that are not of the shape the API expects, use 'binary'.",
    {
      conflicts: ["hash", "hash-type"],
    },
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

  $ truestamp items create --hash a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e --hash-type sha-256

  `,
  )
  .example(
    "STDIN File",
    `Pass a file to STDIN of the 'items create' command:

Pipe content to the 'items create' command using '--input binary' plus the '--stdin' option or the '-' path:

  $ echo -n 'Hello World' | truestamp items create --input binary -
  $ echo -n "Hello World" | truestamp items create --input binary --stdin

  $ cat hello.txt | truestamp items create --input binary -
  $ cat hello.txt | truestamp items create --input binary --stdin

  `,
  )
  .example(
    "FILE Path",
    `Pass a file path to the 'items create' command as the first argument:

  $ truestamp items create --input binary hello.txt

  `,
  )
  .example(
    "JSON STDIN",
    `Pass raw JSON to STDIN of the 'items create' command:

Pipe JSON content to the 'items create' command using '--input json' plus the '--stdin' option or the '-' path:

  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' | truestamp items create --input json -
  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' | truestamp items create --input json --stdin

  # Can be full complexity JSON Item data
  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' > hello.json

  $ cat hello.json | truestamp items create --input json -
  $ cat hello.json | truestamp items create --input json --stdin

  `,
  )
  .example(
    "JSON FILE Path",
    `Pass the '--input json' option and a file path argument to the 'items create' command:

  $ truestamp items create --input json hello.json

  `,
  )
  .action(async (options, path?: string) => {
    const ts = await createTruestampClient(options.env, options.apiKey);

    if (!options.input && options.stdin) {
      throw new ValidationError(
        'Option "--stdin" depends on option "--input".',
      );
    }

    if (!options.input && path) {
      throw new ValidationError(
        'Argument "path" depends on option "--input".',
      );
    }

    let jsonItem, altHash, altHashType, stdInData;

    // If the user provided JSON STDIN using the '--input json' and '--stdin' options or the '-' or a file path argument, parse
    // the contents of STDIN and pass it directly to the Truestamp client as a complete Item object.
    if (options.input === "json" && (options.stdin || path === "-")) {
      stdInData = await readAllSync(Deno.stdin);
      const decoder = new TextDecoder();
      jsonItem = JSON.parse(decoder.decode(stdInData));
    } else if (options.input === "json" && path) {
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
    if (options.input === "binary" && (options.stdin || path === "-")) {
      // await copy(Deno.stdin, Deno.stdout);

      stdInData = await readAllSync(Deno.stdin);
      const hash = new Uint8Array(
        await crypto.subtle.digest("SHA-256", stdInData),
      );
      const hashHex = Array.from(
        hash,
        (byte) => byte.toString(16).padStart(2, "0"),
      ).join("");
      // console.log(`SHA-256(stdin) = ${hashHex}`)
      altHash = hashHex;
      altHashType = "sha-256";
    } else if (options.input === "binary" && path) {
      const file = await Deno.open(path, { read: true, write: false });
      // await copy(file, Deno.stdout);

      const data = await readAllSync(file);
      const hash = new Uint8Array(
        await crypto.subtle.digest("SHA-256", data),
      );
      const hashHex = Array.from(
        hash,
        (byte) => byte.toString(16).padStart(2, "0"),
      ).join("");
      // console.log(`SHA-256(stdin) = ${hashHex}`)
      altHash = hashHex;
      altHashType = "sha-256";

      file.close();
    }

    // If the user provided a hash and a hash-type, validate the hash and hash-type
    // and create the Item. If the user did not provide a hash or a hash-type,
    // but instead had provided the '--stdin' flag or the '-' path argument,
    // or an actual path, create the Item with the hash and hash-type from the
    // STDIN or file.
    try {
      let itemResp;

      if (jsonItem) {
        itemResp = await ts.createItem(jsonItem);
      } else if (options.hash && options.hashType) {
        itemResp = await ts.createItem({
          itemData: [{
            hash: options.hash,
            hashType: <HashTypes> options.hashType,
          }],
        });
      } else if (altHash && altHashType) {
        itemResp = await ts.createItem({
          itemData: [{
            hash: altHash,
            hashType: <HashTypes> altHashType,
          }],
        });
      } else {
        throw new Error("No hash or hash-type provided");
      }

      // console.log(JSON.stringify(itemResp, null, 2));

      // store the item in the local database
      writeItemToDb(options.env, itemResp.id, itemResp);

      // console.log(getItemHashById(getEnv(options), itemResp.id));

      // If a path or STDIN was provided it is helpful to archive the contents of the file
      // and tightly associate it with the returned Item Id/envelope.
      // Copy the binary or JSON file to the common data directory.
      // Name it with the Item Id concatenated with its original name or use 'stdin'.
      if (options.input && (options.stdin || path === "-")) {
        // Copy the data passed as STDIN to the common data directory for archival purposes
        const dataDir = createDataDir(options.env);
        const fileDir = `${dataDir}/files`;
        await Deno.mkdir(fileDir, { recursive: true });
        const filePath = `${fileDir}/${itemResp.id}--stdin`;

        if (stdInData) {
          await Deno.writeFile(filePath, stdInData);
          // console.log(filePath)
        }
      } else if (options.input && path) {
        // Copy the data passed as a file path to the common data directory for archival purposes
        const dataDir = createDataDir(options.env);
        const fileDir = `${dataDir}/files`;
        await Deno.mkdir(fileDir, { recursive: true });
        const filePath = `${fileDir}/${itemResp.id}--${pathParse(path).base}`;
        // console.log(filePath)
        await Deno.copyFile(path, filePath);
      }

      logSelectedOutputFormat(
        {
          text: itemResp.id,
          json: { id: itemResp.id },
        },
        options.output,
      );
    } catch (error) {
      const { response } = error;

      if (response) {
        // HTTPResponseError
        // This is a custom error type thrown by truestamp-js and
        // has a 'response' property which can be awaited to get the full
        // HTTP response, including body with error info.
        // https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#body
        const { status, code, description } = await response.json();
        throw new Error(`${status} : ${code} : ${description}`);
      } else {
        // is a generic Error
        throw error;
      }
    }
  });

const itemsUpdate = new Command<
  {
    env: typeof environmentType;
    apiKey?: string;
    output: typeof outputType;
  }
>()
  .description(
    `Update an existing Item by replacing it with a new one.

Updating an Item returns an updated Id to reflect the update time.

Significantly more complex Items with provenance info, metadata, digital signatures, etc. can be provided as JSON passed into the STDIN or from a file path using the '--input json' option.

Updates are always non-destructive. Older versions are preserved and can be retrieved with their Id.

Any Item attributes that existed on an older version which are not provided again will be removed from the new version.

An Item Id will be returned in the response that identifies the Item and its temporal version. This Id will be stored in a local database for convenience, but you may also store this Item 'id' elsewhere to enable verification of your data in the future.

See the example sections below for detailed usage examples.

`,
  )
  .option(
    "-i, --id <id:string>",
    "An Item Id to update.",
    {
      required: true, // BUG : https://github.com/c4spar/deno-cliffy/issues/436
    },
  )
  .option(
    "-H, --hash <hash:string>",
    "An Item hash encoded as a Hex (Base16) string.",
    {
      required: false,
      conflicts: ["stdin", "input"],
      depends: ["hash-type"],
    },
  )
  .option(
    "-t, --hash-type <hashType:string>",
    `A hash function type. Hash byte length is validated against type. Accepts ${
      HASH_TYPES.join(", ")
    }.`,
    {
      required: false,
      conflicts: ["stdin", "input"],
      depends: ["hash"],
    },
  )
  .option(
    "-s, --stdin",
    "Read data from STDIN and submit updated Item. Will be hashed unless '--input json' is provided.",
    {
      conflicts: ["hash", "hash-type"],
      depends: ["input"],
    },
  )
  .type("input", inputType)
  .option(
    "--input <input:input>",
    "Input format. If 'binary' is provided the file will be hashed and an updated Item constructed and submitted. If 'json' is provided the file not be hashed and will be submitted as-is. JSON content must match the structure expected by the API in all cases. If submitting arbitrary JSON files that are not of the shape the API expects, use 'binary'.",
    {
      conflicts: ["hash", "hash-type"],
    },
  )
  .arguments("[path]")
  .example(
    "Update an Item",
    `Using a previously generated test Id, and a new hash:

  $ echo -n 'Hello World Again' | sha256sum
  63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6  -

  $ truestamp items update --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw --hash 63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6 --hash-type sha-256

  `,
  )
  .example(
    "STDIN File",
    `Pass a file from STDIN to the 'items update' command:

Pipe content to the 'items update' command using the '--stdin' option or the '-' path:

  $ echo -n 'Hello World' | truestamp items update --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw --input binary -
  $ echo -n "Hello World" | truestamp items update --stdin --input binary --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw

  $ cat hello.txt | truestamp items update --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw --input binary -
  $ cat hello.txt | truestamp items update --stdin --input binary --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw

  `,
  )
  .example(
    "FILE Path",
    `Pass an '--id' and a file path to the 'items update' command:

  $ truestamp items update --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw --input binary hello.txt

  `,
  )
  .example(
    "JSON STDIN",
    `Pass raw JSON to STDIN of the 'items update' command:

Pipe JSON content to the 'items update' command using '--input json' plus the '--stdin' option or the '-' path:

  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' | truestamp items update --input json --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw -
  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' | truestamp items update --input json --stdin --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw

  # Can be full complexity JSON Item data
  $ echo -n '{"hash": "63df103e8ebcabdf86d8f13e98a02063fef1da8065335ec0dd978378951534d6", "hashType": "sha-256"}' > hello.json

  $ cat hello.json | truestamp items update --input json --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw -
  $ cat hello.json | truestamp items update --input json --stdin --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw

  `,
  )
  .example(
    "JSON FILE Path",
    `Pass the '--input json' option and a file path argument to the 'items update' command:

  $ truestamp items update --input json --id truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw hello.json

  `,
  )
  .action(
    async (options, path) => {
      const ts = await createTruestampClient(
        options.env,
        options.apiKey,
      );

      if (!options.input && options.stdin) {
        throw new ValidationError(
          'Option "--stdin" depends on option "--input".',
        );
      }

      if (!options.input && path) {
        throw new ValidationError(
          'Argument "path" depends on option "--input".',
        );
      }

      let jsonItem, altHash, altHashType;

      // If the user provided JSON STDIN using the '--input json' and '--stdin' options or the '-' or a file path argument, parse
      // the contents of STDIN and pass it directly to the Truestamp client as a complete Item object.
      if (options.input === "json" && (options.stdin || path === "-")) {
        const data = await readAllSync(Deno.stdin);
        const decoder = new TextDecoder();
        jsonItem = JSON.parse(decoder.decode(data));
      } else if (options.input === "json" && path) {
        const file = await Deno.open(path, {
          read: true,
          write: false,
        });
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
      if (options.input === "binary" && (options.stdin || path === "-")) {
        // await copy(Deno.stdin, Deno.stdout);

        const data = await readAllSync(Deno.stdin);
        const hash = new Uint8Array(
          await crypto.subtle.digest("SHA-256", data),
        );
        const hashHex = Array.from(
          hash,
          (byte) => byte.toString(16).padStart(2, "0"),
        ).join("");
        // console.log(`SHA-256(stdin) = ${hashHex}`)
        altHash = hashHex;
        altHashType = "sha-256";
      } else if (options.input === "binary" && path) {
        const file = await Deno.open(path, {
          read: true,
          write: false,
        });
        // await copy(file, Deno.stdout);

        const data = await readAllSync(file);
        const hash = new Uint8Array(
          await crypto.subtle.digest("SHA-256", data),
        );
        const hashHex = Array.from(
          hash,
          (byte) => byte.toString(16).padStart(2, "0"),
        ).join("");
        // console.log(`SHA-256(stdin) = ${hashHex}`)
        altHash = hashHex;
        altHashType = "sha-256";

        file.close();
      }

      try {
        let itemResp;

        if (jsonItem) {
          itemResp = await ts.updateItem(options.id, jsonItem);
        } else if (options.hash && options.hashType) {
          itemResp = await ts.updateItem(options.id, {
            itemData: [{
              hash: options.hash,
              hashType: <HashTypes> options.hashType,
            }],
          });
        } else if (altHash && altHashType) {
          itemResp = await ts.updateItem(options.id, {
            itemData: [{
              hash: altHash,
              hashType: <HashTypes> altHashType,
            }],
          });
        } else {
          throw new Error("No hash or hashType provided");
        }

        // store the item in the local database
        writeItemToDb(
          options.env,
          itemResp.id,
          itemResp,
        );

        logSelectedOutputFormat(
          { text: itemResp.id, json: { id: itemResp.id } },
          options.output,
        );
      } catch (error) {
        const { response } = error;

        if (response) {
          // HTTPResponseError
          // This is a custom error type thrown by truestamp-js and
          // has a 'response' property which can be awaited to get the full
          // HTTP response, including body with error info.
          // https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch#body
          const { status, code, description } = await response.json();
          throw new Error(`${status} : ${code} : ${description}`);
        } else {
          // is a generic Error
          throw error;
        }
      }
    },
  );

export const items = new Command<
  {
    env: typeof environmentType;
    apiKey?: string;
    output: typeof outputType;
  }
>()
  .description("Create or update Items.")
  .action(() => {
    items.showHelp();
  })
  .command("create", itemsCreate)
  // .command("read", itemsRead)
  .command("update", itemsUpdate);
