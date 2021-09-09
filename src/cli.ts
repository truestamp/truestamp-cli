// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

import {
  colors,
  Command,
  CompletionsCommand,
  EnumType,
  HelpCommand,
  ValidationError,
} from "./deps.ts";

import { auth } from "./commands/auth.ts";
import { documents } from "./commands/documents.ts";
import { health } from "./commands/health.ts";
import { s3 } from "./commands/s3.ts";

// Top level command
const cmd = new Command()
  .throwErrors()
  .name("truestamp")
  .version("0.0.10") // RELEASE VERSION : BUMP VERSION HERE
  .description("Truestamp CLI")
  .help({
    types: false,
    hints: true,
  })
  .type("environment", new EnumType(["development", "staging", "production"]), {
    global: true,
  })
  .option(
    "-E, --env [env:environment]",
    "API environment to use.",
    {
      hidden: true,
      default: "production",
      global: true,
    },
  )
  .action(() => {
    cmd.showHelp();
  })
  .command("auth", auth)
  .command("completions", new CompletionsCommand())
  .command("documents", documents)
  .command("health", health)
  .command("help", new HelpCommand().global())
  .command("s3", s3);

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
