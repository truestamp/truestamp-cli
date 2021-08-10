// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

import { getAccessTokenWithPrompts, Truestamp } from "./deps.ts";

export async function createTruestampClient(apiEnv: string) {
  // FIXME : https://stackoverflow.com/questions/43623461/new-expression-whose-target-lacks-a-construct-signature-in-typescript
  const client = new (Truestamp as any)({
    accessToken: await getAccessTokenWithPrompts(apiEnv),
    apiEnv: apiEnv,
  });

  return client;
}
