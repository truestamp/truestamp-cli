// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import {
  Command,
  createTruestampClient,
  deleteTokensInConfig,
  getConfigAccessToken,
  getConfigIdTokenPayload,
  getConfigRefreshToken,
} from "../deps.ts";

import { apiKeys } from "./apikeys/apikeys.ts";

const authLogin = new Command()
  .description("Login.")
  .action(async (options) => {
    if (options.apiKey !== undefined) {
      console.log("apiKey:", options.apiKey);
      throw new Error(`login is not permitted when an API key is provided as an option`);
    }

    // Do not pass in apiKey, this is the standard JWT access/refresh token login.
    const ts = await createTruestampClient(options.env);

    try {
      await ts.getHealth();
    } catch (error) {
      throw new Error(`health check failed : ${error.message}`);
    }
  });

const authLogout = new Command()
  .description("Logout.")
  .action((options) => {
    deleteTokensInConfig(options.env);
    console.log("logout complete");
  });

const authStatus = new Command()
  .description("Check login status.")
  .action(async (options) => {

    // try simple validation with provided API key
    if (options.apiKey !== undefined) {
      try {
        const ts = await createTruestampClient(options.env, options.apiKey);
        await ts.getHealth();
        console.log(
          `confirmed access to '${options.env}' environment with API key`,
        );
      } catch (error) {
        throw new Error(`health check failed : ${error.message}`);
      }
    }

    // try simple validation with access/refresh tokens in config
    if (options.apiKey === undefined) {
      if (
        !getConfigAccessToken(options.env) ||
        !getConfigRefreshToken(options.env)
      ) {
        console.error("you appear to be logged out");
        Deno.exit(1);
      }

      try {
        const ts = await createTruestampClient(options.env);
        await ts.getHealth();
      } catch (error) {
        throw new Error(`health check failed : ${error.message}`);
      }

      const payload = getConfigIdTokenPayload(options.env);
      if (payload) {
        console.log(
          `confirmed access to '${options.env}' environment as '${payload.name} (${payload.email})'`,
        );
      } else {
        throw new Error("id token missing or invalid");
      }
    }
  });

export const auth = new Command()
  .description("Login, logout, or check login status. Create API keys.")
  .action(() => {
    auth.showHelp();
  })
  .command("login", authLogin)
  .command("logout", authLogout)
  .command("status", authStatus)
  .command("keys", apiKeys);
