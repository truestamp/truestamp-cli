// Copyright © 2020-2023 Truestamp Inc. All rights reserved.

import {
  Command,
  createTruestampClient,
  getConfigRefreshToken,
  ValidationError,
} from "../deps.ts";

import { logSelectedOutputFormat, throwApiError } from "../utils.ts";

import { environmentType, outputType } from "../cli.ts";

const MAX_DESCRIPTION_LENGTH = 256;

const apiKeyCreate = new Command<{
  env: typeof environmentType;
  apiKey?: string;
  output: typeof outputType;
}>()
  .description("Create a new API key.")
  .option(
    "-d, --description <description:string>",
    `A description of the key. (max length: ${MAX_DESCRIPTION_LENGTH})`,
    {
      required: false,
      default: "",
    },
  )
  .option(
    "-t, --ttl <ttl:integer>",
    "Key Time To Live value (seconds). (min: 60s, 0 for no expiration)",
    {
      required: false,
      default: 0,
    },
  )
  .example(
    "New API Key",
    `Create a new expiring API key for the current environment with optional expiry and description.

  # expires in one hour.
  $ truestamp auth keys create -o text -d "a description" -t 3600
  01FZPNTMAQZZCH170999999999_8djbT1Ys078OZImR1uRr4jh999999999 [production]

  `,
  )
  .action(async (options) => {
    const refreshToken = getConfigRefreshToken(options.env);

    if (!refreshToken) {
      console.error("logged out, you need to 'truestamp auth login' first");
      Deno.exit(1);
    }

    if (options.description.length > MAX_DESCRIPTION_LENGTH) {
      throw new ValidationError(
        `key description max length of ${MAX_DESCRIPTION_LENGTH} characters exceeded`,
      );
    }

    if (options.ttl !== 0 && options.ttl < 60) {
      throw new ValidationError(
        `key TTL must be at least 60 seconds (or '0' for no expiration)`,
      );
    }

    const truestamp = await createTruestampClient(options.env, options.apiKey);

    try {
      const keyResp = await truestamp.createApiKey({
        refreshToken: refreshToken,
        description: options.description,
        ttl: options.ttl,
      });

      logSelectedOutputFormat(
        {
          text: `${keyResp.apiKey} [${options.env}]`,
          json: {
            command: "apikey",
            status: "ok",
            environment: options.env,
            key: keyResp.apiKey,
          },
        },
        options.output,
      );
    } catch (error) {
      throwApiError(
        "key creation error",
        error.message,
      );
    }
  });

export const apiKeys = new Command<{
  env: typeof environmentType;
  apiKey?: string;
  output: typeof outputType;
}>()
  .description(
    `Manage API keys.

    API keys are not needed to use the CLI or web interface. They're provided as a
    convenience for using non-interactive clients like 'cURL' or in machine-to-machine
    contexts.

    Keys can be created with an optional description or a TTL (Time To Live) value. Once the
    TTL has expired the key will be automatically invalidated.`,
  )
  .action(() => {
    apiKeys.showHelp();
  })
  .command("create", apiKeyCreate);
