// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

import { Command, createTruestampClient } from "../deps.ts";

const documentsNew = new Command()
  .description(
    `Create a new document.
  
  Provide a hash in Multihash, hex, or Base64 encoding.

  When a hash that is Base64 or Hex encoded it must be accompanied by
  the hash function '--name' parameter which must be one of the names
  found at: https://github.com/multiformats/multicodec/blob/master/table.csv
  `,
  )
  .option(
    "-H, --hash [hash:string]",
    "A document hash encoded as a MultiHash, hex, or Base64 string.",
    {
      required: true,
    },
  )
  .option(
    "-N, --name [name:string]",
    "a MultiHash hash function name. Optional only if '--hash' is a MultiHash.",
    {
      required: false,
    },
  )
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    let d
    try {
      d = await ts.createDocument({
        hash: options.hash,
        name: options.name,
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
    `Update an existing document.

  Provide a hash in Multihash, hex, or Base64 encoding.

  When a hash that is Base64 or Hex encoded it must be accompanied by
  the hash function '--name' parameter which must be one of the names
  found at: https://github.com/multiformats/multicodec/blob/master/table.csv
  `,
  )
  .option("-i, --id [id:string]", "A document ID.", {
    required: true,
  })
  .option(
    "-H, --hash [hash:string]",
    "A document hash encoded as a MultiHash, hex, or Base64 string.",
    {
      required: true,
    },
  )
  .option(
    "-N, --name [name:string]",
    "a MultiHash hash function name. Optional only if '--hash' is a MultiHash.",
    {
      required: false,
    },
  )
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    let d
    try {
      d = await ts.updateDocument(options.id, {
        hash: options.hash,
        name: options.type,
      });
    } catch (error) {
      throw new Error(`document not found : ${error.message}`);
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
