// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

import { getAccessTokenWithPrompts, Truestamp } from "./deps.ts";

export async function createTruestampClient(apiEnv: string) {
  const client = new Truestamp({
    accessToken: await getAccessTokenWithPrompts(apiEnv),
    apiEnv: apiEnv,
  });

  return client;
}
