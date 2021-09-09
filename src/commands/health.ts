// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

import { Command, createTruestampClient } from "../deps.ts";

export const health = new Command()
  .description("Display results of API server health call.")
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    let health
    try {
      health = await ts.getHealth();
    } catch (error) {
      throw new Error(`health : ${error.message}`);
    }

    console.log(JSON.stringify(health));
  });
