// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { getAccessTokenWithPrompts } from "./deps.ts";

import { TruestampClient } from "@truestamp/client";

function getApiBaseUrlFromEnv(env?: string): string {
  switch (env) {
    case "development":
      return "https://dev-api.truestamp.com";

    case "staging":
      return "https://staging-api.truestamp.com";

    default:
      return "https://api.truestamp.com";
  }
}

export async function createTruestampClient(
  env: string,
  apiKey?: string,
): Promise<TruestampClient> {
  const client = new TruestampClient({
    apiBaseUrl: getApiBaseUrlFromEnv(env),
    apiKey: apiKey ?? (await getAccessTokenWithPrompts(env)),
  });

  return client;
}
