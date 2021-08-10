// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

import { Command, createTruestampClient } from "../deps.ts";

export const heartbeat = new Command()
  .description("Display results of API server heartbeat call.")
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    const hb = await ts.getHeartbeat();
    console.log(JSON.stringify(hb));
  });
