// Copyright © 2020-2023 Truestamp Inc. All rights reserved.

import { Conf, Json } from "./deps.ts";

// e.g.  cat "/Users/glenn/Library/Preferences/com.truestamp.cli.development/config.json"

function getConfigProjectNameForEnv(env: string): string {
  return `com.truestamp.cli.${env}`;
}

export function getConfigForEnv(env: string): Conf {
  const projectName = getConfigProjectNameForEnv(env);
  const config = new Conf({
    projectName,
  });

  return config;
}

export function getConfigKeyForEnv(env: string, key: string): Json | null {
  const config = getConfigForEnv(env);
  return config.get(key);
}

export function setConfigKeyForEnv(
  env: string,
  key: string,
  value: Json,
): void {
  const config = getConfigForEnv(env);
  config.set(key, value);
}

export function deleteConfigKeyForEnv(env: string, key: string): void {
  const config = getConfigForEnv(env);
  config.delete(key);
}
