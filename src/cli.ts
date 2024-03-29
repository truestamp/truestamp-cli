// Copyright © 2020-2023 Truestamp Inc. All rights reserved.

import {
  colors,
  Command,
  CompletionsCommand,
  EnumType,
  fromZodError,
  HelpCommand,
  ValidationError,
  ZodError,
} from "./deps.ts";

import { auth } from "./commands/auth.ts";
import { commitments } from "./commands/commitments.ts";
import { entropyCommand } from "./commands/entropy.ts";
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
  .version("1.1.0") // RELEASE VERSION : BUMP VERSION HERE
  .description("Truestamp CLI")
  .meta("deno", Deno.version.deno)
  .meta("v8", Deno.version.v8)
  .meta("typescript", Deno.version.typescript)
  .help({
    types: false,
    hints: true,
  })
  .globalType("environment", environmentType)
  .globalEnv("TRUESTAMP_ENV=<env:environment>", "Override API endpoint.", {
    required: false,
    prefix: "TRUESTAMP_", // prefix will be ignored when converting to option name. e.g. TRUESTAMP_ENV becomes 'env'
  })
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
  .globalEnv("TRUESTAMP_OUTPUT=<output:output>", "Preferred output format.", {
    required: false,
    prefix: "TRUESTAMP_",
  })
  .globalOption(
    "-o, --output <output:output>",
    "Output format. Overrides 'TRUESTAMP_OUTPUT' env var.",
    {
      default: "text",
    },
  )
  .globalEnv(
    "TRUESTAMP_SIGNING_SECRET_KEY=<signingSecretKey:string>",
    "A Base64 encoded ed25519 secret key.",
    {
      required: false,
      prefix: "TRUESTAMP_",
    },
  )
  .globalOption(
    "-S, --signing-secret-key <signingSecretKey:string>",
    "A Base64 encoded ed25519 secret key. Overrides 'TRUESTAMP_SIGNING_SECRET_KEY' env var.",
  )
  .action(() => {
    cmd.showHelp();
  })
  .command("auth", auth)
  .command("commitments", commitments)
  .command("items", items)
  .command("entropy", entropyCommand)
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
  } else if (error instanceof ZodError) {
    const validationError = fromZodError(error);
    Deno.stderr.writeSync(
      new TextEncoder().encode(
        colors.red(`  ${colors.bold("Error")}: ${validationError}\n`) + "\n",
      ),
    );
  } else if (error instanceof Error) {
    Deno.stderr.writeSync(
      new TextEncoder().encode(
        colors.red(`  ${colors.bold("Error")}: ${error.message}\n`) + "\n",
      ),
    );
  }
  Deno.exit(error instanceof ValidationError ? error.exitCode : 1);
}
