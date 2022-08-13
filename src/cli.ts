// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import {
  colors,
  Command,
  CompletionsCommand,
  EnumType,
  HelpCommand,
  ValidationError
} from "./deps.ts";

import { auth } from "./commands/auth.ts";
import { commitments } from "./commands/commitments.ts";
import { items } from "./commands/items.ts";

export const environmentType = new EnumType([
  "development",
  "staging",
  "production",
]);
export const outputType = new EnumType(["silent", "text", "json"]);

// Top level command
const cmd = new Command()
  .throwErrors()
  .name("truestamp")
  .version("0.0.18") // RELEASE VERSION : BUMP VERSION HERE
  .description("Truestamp CLI")
  .help({
    types: false,
    hints: true,
  })
  .globalType("environment", environmentType)
  .globalEnv(
    "TRUESTAMP_ENV=<env:environment>",
    "Override API endpoint.",
    {
      required: false,
      prefix: "TRUESTAMP_", // prefix will be ignored when converting to option name. e.g. TRUESTAMP_ENV becomes 'env'
    },
  )
  .globalOption(
    "-E, --env <env:environment>",
    "Override API endpoint. Overrides 'TRUESTAMP_ENV' env var.",
    {
      required: false,
      default: "production",
    },
  )
  .globalEnv(
    "TRUESTAMP_API_KEY=<apiKey:string>",
    "Force use of API key for authentication.",
    {
      required: false,
      prefix: "TRUESTAMP_",
    },
  )
  .globalOption(
    "-A, --api-key <apiKey:string>",
    "Use API key for authentication. Overrides 'TRUESTAMP_API_KEY' env var.",
  )
  .globalType("output", outputType)
  .globalEnv(
    "TRUESTAMP_OUTPUT=<output:output>",
    "Preferred output format.",
    {
      required: false,
      prefix: "TRUESTAMP_",
    },
  )
  .globalOption(
    "-o, --output <output:output>",
    "Output format. Overrides 'TRUESTAMP_OUTPUT' env var.",
    {
      default: "text",
    },
  )
  .action(() => {
    cmd.showHelp();
  })
  .command("auth", auth.reset())
  .command("commitments", commitments)
  .command("items", items)
  .command("completions", new CompletionsCommand())
  .command("help", new HelpCommand().global());

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
