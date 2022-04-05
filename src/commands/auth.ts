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

import { getEnv, logSelectedOutputFormat } from "../utils.ts";

const authLogin = new Command()
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
      throw new Error(`login is not permitted when an API key is provided as an option`);
    }

    // Do not pass in apiKey, this is the standard JWT access/refresh token login.
    const ts = await createTruestampClient(getEnv(options));

    try {
      await ts.getHealth();
    } catch (error) {
      throw new Error(`health check failed : ${error.message}`);
    }

    logSelectedOutputFormat(options, { text: `logged in [${getEnv(options)}]`, json: { command: 'login', status: 'ok', environment: getEnv(options) } });
  });

const authLogout = new Command()
  .description("Logout.")
  .example(
    "Logout",
    `Logout of the current environment.

  $ truestamp auth logout

  `,
  )
  .action((options) => {
    deleteTokensInConfig(getEnv(options));
    logSelectedOutputFormat(options, { text: `logged out [${getEnv(options)}]`, json: { command: 'logout', status: 'ok', environment: getEnv(options) } });
  });

const authStatus = new Command()
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
        const ts = await createTruestampClient(getEnv(options), options.apiKey);
        await ts.getHealth();
        logSelectedOutputFormat(options, { text: `logged in (with API Key) [${getEnv(options)}]`, json: { command: 'status', status: 'ok', environment: getEnv(options) } });
      } catch (error) {
        throw new Error(`logged out : API key health check failed : ${error.message}`);
      }
    }

    // try simple validation with access/refresh tokens in config
    if (options.apiKey === undefined) {
      if (
        !getConfigAccessToken(getEnv(options)) ||
        !getConfigRefreshToken(getEnv(options))
      ) {
        throw new Error(`logged out : no access/refresh tokens found`);
      }

      try {
        const ts = await createTruestampClient(getEnv(options));
        await ts.getHealth();
      } catch (error) {
        throw new Error(`logged out : access check failed : ${error.message}`);
      }

      try {
        // throws if token in config is invalid
        const payload = getConfigIdTokenPayload(getEnv(options));
        logSelectedOutputFormat(options, { text: `logged in : ${payload.email} [${getEnv(options)}]`, json: { command: 'status', status: 'ok', environment: getEnv(options), idToken: payload } });
      } catch (_error) {
        // ID token in config is invalid or expired
        // handle gracefully since this is the only place ID token is used and
        // we're already proven the access token is valid.
        logSelectedOutputFormat(options, { text: `logged in : [${getEnv(options)}]`, json: { command: 'status', status: 'ok', environment: getEnv(options) } });
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
