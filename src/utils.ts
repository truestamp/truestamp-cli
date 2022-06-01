// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

interface OutputWrapper {
  text: string
  // deno-lint-ignore no-explicit-any
  json: Record<string, any>
}

export function logSelectedOutputFormat(data: OutputWrapper, outputFormat: string): void {
  switch (outputFormat) {
    case "text":
      console.log(data.text);
      break;
    case "json":
      console.log(JSON.stringify(data.json, null, 2));
      break;
    case "silent":
      // no-op : silent
      break;
  }
}
