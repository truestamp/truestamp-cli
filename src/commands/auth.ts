// © 2020-2021 Truestamp Inc. All rights reserved.

import {
  Command,
  createTruestampClient,
  deleteTokensInConfig,
  getConfigAccessToken,
  getConfigIdTokenPayload,
  getConfigRefreshToken,
} from "../deps.ts";

const authLogin = new Command()
  .description("Authenticate with a Truestamp host")
  .action(async (options) => {
    const ts = await createTruestampClient(options.env);
    const hb = await ts.getHeartbeat();
    if (hb) {
      console.log("login successful");
    } else {
      throw new Error("auth login heartbeat check failed");
    }
  });

const authLogout = new Command()
  .description("Log out of a Truestamp host")
  .action((options) => {
    deleteTokensInConfig(options.env);
    console.log("logout complete");
  });

const authStatus = new Command()
  .description("View authentication status")
  .action(async (options) => {
    if (
      !getConfigAccessToken(options.env) ||
      !getConfigRefreshToken(options.env)
    ) {
      console.error("logged out");
      Deno.exit(1);
    }

    const ts = await createTruestampClient(options.env);
    const hb = await ts.getHeartbeat();
    if (!hb) {
      throw new Error("auth status heartbeat check failed");
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

const authToken = new Command()
  .description("Get a long-lived API token")
  .action(async (options) => {
    if (
      !getConfigAccessToken(options.env) ||
      !getConfigRefreshToken(options.env)
    ) {
      console.error("logged out");
      Deno.exit(1);
    }

    const ts = await createTruestampClient(options.env);
    const token = await ts.getToken();
    if (!token) {
      throw new Error("auth token retrieval failed");
    }

    console.log(JSON.stringify(token));
  });

export const auth = new Command()
  .description("Login, logout, and show status of your authentication.")
  .action(() => {
    auth.showHelp();
  })
  .command("login", authLogin)
  .command("logout", authLogout)
  .command("status", authStatus)
  .command("token", authToken);
