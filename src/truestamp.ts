// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { getAccessTokenWithPrompts, Truestamp } from "./deps.ts";

export async function createTruestampClient(apiEnv: string, apiKey?: string): Promise<Truestamp> {
  const client = new Truestamp({
    apiKey: apiKey ?? await getAccessTokenWithPrompts(apiEnv),
    apiEnv: apiEnv ?? "production",
  });

  return client;
}
