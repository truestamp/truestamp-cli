// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

// deno-lint-ignore-file no-explicit-any

interface OutputWrapper {
  text: string
  json: Record<string, any>
}

export function getEnv(options: any): string {
  return options?.env ?? "production";
}

export function logSelectedOutputFormat(options: any, data: OutputWrapper): void {
  // --output overrides TRUESTAMP_OUTPUT env var
  // Defaults to 'text'
  const choice = options.output ?? options.outputVar ?? "text";

  // console.log(choice)

  if (choice === "text") {
    console.log(data.text);
  } else if (choice === "json") {
    console.log(JSON.stringify(data.json, null, 2));
  } else {
    // silent
  }
}
