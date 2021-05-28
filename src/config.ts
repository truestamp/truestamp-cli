// © 2020-2021 Truestamp Inc. All rights reserved.

import { Conf, ItemType } from "./deps.ts";

function getConfigProjectNameForEnv(env: string): string {
  return `com.truestamp.cli.${env}`;
}

export function getConfigForEnv(
  env: string,
): Conf {
  const projectName = getConfigProjectNameForEnv(env);
  const config = new Conf({
    projectName,
    defaults: { aws_s3_region: "us-east-1" },
  });

  console.log(config);

  return config;
}

export function getConfigKeyForEnv(
  env: string,
  key: string,
): ItemType | null {
  const config = getConfigForEnv(env);
  return config.get(key);
}

export function setConfigKeyForEnv(
  env: string,
  key: string,
  value: ItemType,
): void {
  const config = getConfigForEnv(env);
  config.set(key, value);
}

export function deleteConfigKeyForEnv(
  env: string,
  key: string,
): void {
  const config = getConfigForEnv(env);
  config.delete(key);
}
