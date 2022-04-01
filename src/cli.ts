// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import {
  colors,
  Command,
  CompletionsCommand,
  EnumType,
  HelpCommand,
  ValidationError,
} from "./deps.ts";

import { auth } from "./commands/auth.ts";
import { items } from "./commands/items.ts";

// FIXME : add --pretty-json and --quiet flags to regulate output of commands?
// FIXME : add --json flag to output json instead of text?

// FIXME : the following bug has been filed related to TRUESTAMP_ENV handling:
// https://github.com/c4spar/deno-cliffy/issues/340

// Top level command
const cmd = new Command()
  .throwErrors()
  .name("truestamp")
  .version("0.0.13") // RELEASE VERSION : BUMP VERSION HERE
  .description("Truestamp CLI")
  .help({
    types: false,
    hints: true,
  })
  .type("environment", new EnumType(["development", "staging", "production"]), {
    global: true,
  })
  .env<{ env: string }>(
    "TRUESTAMP_ENV=<env:environment>",
    "Override API endpoint.",
    {
      global: true,
      required: false,
      prefix: "TRUESTAMP_" // prefix will be ignored when converting to option name. e.g. TRUESTAMP_ENV becomes 'env'
    },
  )
  .option<{ env: string }>(
    "-E, --env [env:environment]",
    "Override API endpoint. Overrides 'TRUESTAMP_ENV' env var.",
    {
      hidden: false,
      global: true,
    },
  )
  .env<{ apiKey: string }>(
    "TRUESTAMP_API_KEY=<apiKey:string>",
    "Force use of API key for authentication.",
    {
      global: true,
      required: false,
      prefix: "TRUESTAMP_" // prefix will be ignored when converting to option name.
    },
  )
  .option<{ apiKey: string }>(
    "-A, --api-key [apiKey:string]",
    "Force use of API key for authentication. Overrides 'TRUESTAMP_API_KEY' env var.",
    {
      hidden: false,
      global: true,
    },
  )
  .action(() => {
    cmd.showHelp();
  })
  .command("auth", auth)
  .command("completions", new CompletionsCommand())
  .command("items", items)
  .command("help", new HelpCommand().global())

try {
  await cmd.parse(Deno.args);
} catch (error) {
  if (error instanceof ValidationError) {
    cmd.showHelp();
    Deno.stderr.writeSync(
      new TextEncoder().encode(
        colors.yellow(
          `  ${colors.bold("Validation Error")}: ${error.message}\n`,
        ) + "\n",
      ),
    );
  } else if (error instanceof Error) {
    Deno.stderr.writeSync(
      new TextEncoder().encode(
        colors.red(
          `  ${colors.bold("Error")}: ${error.message}\n`,
        ) + "\n",
      ),
    );
  }
  Deno.exit(error instanceof ValidationError ? error.exitCode : 1);
}
