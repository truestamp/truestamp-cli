// © 2020-2021 Truestamp Inc. All rights reserved.

import { Command, createTruestampClient } from "../deps.ts";

const documentsNew = new Command()
  .description(
    `Create a new document.
  
  When a hash is provided it must be one of the types found at:
  https://github.com/multiformats/multicodec/blob/master/table.csv
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
    "-T, --type [type:string]",
    "a MultiHash hash type. Optional only if '--hash' is a MultiHash.",
    {
      required: false,
    },
  )
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    const d = await ts.createDocument({
      hash: options.hash,
      type: options.type,
    });
    if (d) {
      console.log(JSON.stringify(d));
    } else {
      throw new Error("new document creation failed");
    }
  });

const documentsShow = new Command()
  .description("Show an existing document.")
  .option("-i, --id [type:string]", "A document ID.", {
    required: true,
  })
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    const d = await ts.getDocument(options.id);
    if (d) {
      console.log(JSON.stringify(d));
    } else {
      throw new Error("document not found");
    }
  });

const documentsUpdate = new Command()
  .description(
    `Update an existing document.
  
  When a hash is provided it must be one of the types found at:
  https://github.com/multiformats/multicodec/blob/master/table.csv
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
    "-T, --type [type:string]",
    "a MultiHash hash type. Optional only if '--hash' is a MultiHash.",
    {
      required: false,
    },
  )
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    const d = await ts.updateDocument(options.id, {
      hash: options.hash,
      type: options.type,
    });
    if (d) {
      console.log(JSON.stringify(d));
    } else {
      throw new Error("document not found");
    }
  });

const documentsDelete = new Command()
  .description("Delete an existing document.")
  .option("-i, --id [type:string]", "A document ID.", {
    required: true,
  })
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    const d = await ts.deleteDocument(options.id);
    if (d) {
      console.log(JSON.stringify(d));
    } else {
      throw new Error("document not found");
    }
  });

const documentsList = new Command()
  .description("List all existing documents.")
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    const d = await ts.getAllDocuments();
    if (d) {
      console.log(JSON.stringify(d));
    } else {
      throw new Error("documents not found");
    }
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