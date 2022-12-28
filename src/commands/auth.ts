// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import {
  Command,
  createTruestampClient,
  deleteTokensInConfig,
  getConfigAccessToken,
  getConfigIdTokenPayload,
  getConfigRefreshToken,
} from "../deps.ts";

import { apiKeys } from "./apikeys.ts";

import { logSelectedOutputFormat } from "../utils.ts";

import { environmentType, outputType } from "../cli.ts";

const authLogin = new Command<{
  env: typeof environmentType;
  apiKey?: string;
  output: typeof outputType;
}>()
  .description("Login.")
  .example(
    "Login",
    `Login to the current environment.

  $ truestamp auth login

  `,
  )
  .action(async (options) => {
    if (options.apiKey !== undefined) {
      // console.log("apiKey:", options.apiKey);
      throw new Error(
        `login is not permitted when an API key is provided as an option`,
      );
    }

    // Do not pass in apiKey, this is the standard JWT access/refresh token login.
    const ts = await createTruestampClient(options.env);

    try {
      await ts.getHealth();
    } catch (error) {
      throw new Error(`health check failed : ${error.message}`);
    }

    logSelectedOutputFormat(
      {
        text: `logged in [${options.env}]`,
        json: {
          command: "login",
          status: "ok",
          environment: options.env,
        },
      },
      options.output,
    );
  });

const authLogout = new Command<{
  env: typeof environmentType;
  apiKey?: string;
  output: typeof outputType;
}>()
  .description("Logout.")
  .example(
    "Logout",
    `Logout of the current environment.

  $ truestamp auth logout

  `,
  )
  .action((options) => {
    deleteTokensInConfig(options.env);
    logSelectedOutputFormat(
      {
        text: `logged out [${options.env}]`,
        json: {
          command: "logout",
          status: "ok",
          environment: options.env,
        },
      },
      options.output,
    );
  });

const authStatus = new Command<{
  env: typeof environmentType;
  apiKey?: string;
  output: typeof outputType;
}>()
  .description("Check login status.")
  .example(
    "Status",
    `Check login status for the current environment.

  $ truestamp auth status

  `,
  )
  .action(async (options) => {
    // try simple validation with a provided API key (not JWT token)
    if (options.apiKey !== undefined) {
      try {
        const ts = await createTruestampClient(options.env, options.apiKey);
        await ts.getHealth();
        logSelectedOutputFormat(
          {
            text: `logged in (with API Key) [${options.env}]`,
            json: {
              command: "status",
              status: "ok",
              environment: options.env,
            },
          },
          options.output,
        );
      } catch (error) {
        throw new Error(
          `logged out : API key health check failed : ${error.message}`,
        );
      }
    }

    // try simple validation with access/refresh tokens in config
    if (options.apiKey === undefined) {
      if (
        !getConfigAccessToken(options.env) ||
        !getConfigRefreshToken(options.env)
      ) {
        throw new Error(
          `logged out [${options.env}] : no access/refresh tokens found`,
        );
      }

      try {
        const ts = await createTruestampClient(options.env);
        await ts.getHealth();
      } catch (error) {
        throw new Error(`logged out : access check failed : ${error.message}`);
      }

      try {
        // throws if token in config is invalid
        const payload = getConfigIdTokenPayload(options.env);
        logSelectedOutputFormat(
          {
            text: `logged in : ${payload.email} [${options.env}]`,
            json: {
              command: "status",
              status: "ok",
              environment: options.env,
              idToken: payload,
            },
          },
          options.output,
        );
      } catch (_error) {
        // ID token in config is invalid or expired
        // handle gracefully since this is the only place ID token is used and
        // we're already proven the access token is valid.
        logSelectedOutputFormat(
          {
            text: `logged in : [${options.env}]`,
            json: {
              command: "status",
              status: "ok",
              environment: options.env,
            },
          },
          options.output,
        );
      }
    }
  });

export const auth = new Command<{
  env: typeof environmentType;
  apiKey?: string;
  output: typeof outputType;
}>()
  .description("Login, logout, or check login status. Create API keys.")
  .action(() => {
    auth.showHelp();
  })
  .command("login", authLogin)
  .command("logout", authLogout)
  .command("status", authStatus)
  .command("keys", apiKeys);
