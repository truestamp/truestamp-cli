// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

export function getEnv(options: any): string {
  return options?.env ?? "production";
}
