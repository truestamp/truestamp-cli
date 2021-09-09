// Copyright © 2020-2021 Truestamp Inc. All rights reserved.

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
  .description("Authenticate with a Truestamp host.")
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);

    try {
      await ts.getHealth();
    } catch (error) {
      throw new Error(`auth login health check failed : ${error.message}`);
    }
  });

const authLogout = new Command()
  .description("Log out of a Truestamp host.")
  .action((options) => {
    deleteTokensInConfig(options.env);
    console.log("logout complete");
  });

const authStatus = new Command()
  .description("View authentication status.")
  .action(async (options) => {
    if (
      !getConfigAccessToken(options.env) ||
      !getConfigRefreshToken(options.env)
    ) {
      console.error("logged out");
      Deno.exit(1);
    }

    const ts = await createTruestampClient(options.env);
    try {
      await ts.getHealth();
    } catch (error) {
      throw new Error(`auth status health check failed : ${error.message}`);
    }

    const payload = getConfigIdTokenPayload(options.env);
    if (payload) {
      console.log(
        `logged into '${options.env}' environment as user '${payload.name} (${payload.email})'`,
      );
    } else {
      throw new Error("id token missing or invalid");
    }
  });

export const auth = new Command()
  .description("Login, logout, and show your current authentication status, or manage long-lived API keys.")
  .action(() => {
    auth.showHelp();
  })
  .command("login", authLogin)
  .command("logout", authLogout)
  .command("status", authStatus)
  .command("keys", apiKeys);
