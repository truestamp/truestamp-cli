// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

// See: https://github.com/truestamp/deviceflow
// See: https://github.com/jatinvaidya/cli-authz-device-flow/blob/master/device/device.js

// On MacOS the config files can be found in a location like:
// cat ~/Library/Preferences/com.truestamp.cli.development/config.json

import { colors, decode, Payload, sleep, validate } from "./deps.ts";

import {
  deleteConfigKeyForEnv,
  getConfigKeyForEnv,
  setConfigKeyForEnv,
} from "./config.ts";

const AUTH0_SCOPES = "openid profile email offline_access";

const AUTH0_DOMAIN_DEVELOPMENT = "truestamp-dev.auth0.com";
const AUTH0_AUDIENCE_DEVELOPMENT = "https://db.fauna.com/db/ytijhfregydfy";
const AUTH0_CLIENT_ID_DEVELOPMENT = "8djbT1Ys078OZImR1uRr4jhu2Wb6d05B";

const AUTH0_DOMAIN_STAGING = "truestamp-staging.auth0.com";
const AUTH0_AUDIENCE_STAGING = "https://db.fauna.com/db/ytijhdrceybfy";
const AUTH0_CLIENT_ID_STAGING = "T0dzxGnnIj3TU0HpzCQRTZ5fx9N5Hb5m";

const AUTH0_DOMAIN_PRODUCTION = "login.truestamp.com";
const AUTH0_AUDIENCE_PRODUCTION = "https://db.fauna.com/db/ytij595b6yffy";
const AUTH0_CLIENT_ID_PRODUCTION = "pS5kRvqeuz4XLoxNPd6VX2LlUyNyU7Xj";

function getAuth0DomainForEnv(env: string): string {
  switch (env) {
    case "development":
      return AUTH0_DOMAIN_DEVELOPMENT;

    case "staging":
      return AUTH0_DOMAIN_STAGING;

    case "production":
      return AUTH0_DOMAIN_PRODUCTION;

    default:
      throw new Error(`invalid environment : '${env}'`);
  }
}

function getAuth0AudienceForEnv(env: string): string {
  switch (env) {
    case "development":
      return AUTH0_AUDIENCE_DEVELOPMENT;

    case "staging":
      return AUTH0_AUDIENCE_STAGING;

    case "production":
      return AUTH0_AUDIENCE_PRODUCTION;

    default:
      throw new Error(`invalid environment : '${env}'`);
  }
}

function getAuth0ClientIdForEnv(env: string): string {
  switch (env) {
    case "development":
      return AUTH0_CLIENT_ID_DEVELOPMENT;

    case "staging":
      return AUTH0_CLIENT_ID_STAGING;

    case "production":
      return AUTH0_CLIENT_ID_PRODUCTION;

    default:
      throw new Error(`invalid environment : '${env}'`);
  }
}

async function getDeviceCode(env: string) {
  const resp = await fetch(
    `https://${getAuth0DomainForEnv(env)}/oauth/device/code`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        client_id: getAuth0ClientIdForEnv(env),
        audience: getAuth0AudienceForEnv(env),
        scope: AUTH0_SCOPES,
      }),
    },
  );
  return resp.json();
}

async function callTokenEndpoint(
  env: string,
  deviceCode: string,
): Promise<Response> {
  const resp = await fetch(`https://${getAuth0DomainForEnv(env)}/oauth/token`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      client_id: getAuth0ClientIdForEnv(env),
      device_code: deviceCode,
      grant_type: "urn:ietf:params:oauth:grant-type:device_code",
    }),
  });
  return resp;
}

// Get the whole token response object by polling until the
// user authenticates or fails at doing so.
async function getTokens(env: string, deviceCode: string, interval: number) {
  let adjustedInterval = interval;

  while (true) {
    await sleep(adjustedInterval);
    const resp = await callTokenEndpoint(env, deviceCode);

    if (resp.ok) {
      return await resp.json();
    }

    if (!resp.ok) {
      const respJson = await resp.json();

      switch (respJson.error) {
        case "authorization_pending":
          // console.log(colors.bold.gray("authorization pending..."));
          break;

        case "slow_down":
          // add a second to the polling interval each time received
          adjustedInterval += 1;
          break;

        case "expired_token":
          throw new Error(`expired token`);

        case "access_denied":
          throw new Error(`access denied`);

        default:
          throw new Error(
            `unknown error response : ${JSON.stringify(respJson)}`,
          );
      }
    }
  }
}

async function getNewTokensWithRefreshToken(env: string) {
  const refreshToken = getConfigRefreshToken(env);
  if (refreshToken) {
    const resp = await fetch(
      `https://${getAuth0DomainForEnv(env)}/oauth/token`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          grant_type: "refresh_token",
          client_id: getAuth0ClientIdForEnv(env),
          refresh_token: refreshToken,
        }),
      },
    );
    return await resp.json();
  }
}

export function getConfigAccessToken(env: string): string | undefined {
  const t = getConfigKeyForEnv(env, "auth0_access_token") as string;
  return t ? t : undefined;
}

export function getConfigRefreshToken(env: string): string | undefined {
  const t = getConfigKeyForEnv(env, "auth0_refresh_token") as string;
  return t ? t : undefined;
}

export function getConfigIdTokenPayload(env: string): Payload | undefined {
  const t = getConfigKeyForEnv(env, "auth0_id_token") as string;

  if (t) {
    const { payload } = validate(decode(t));
    return payload;
  } else {
    return undefined;
  }
}

function setTokensInConfig(
  env: string,
  tokens: {
    access_token: string;
    id_token?: string;
    refresh_token?: string;
    scope: string;
    expires_in: number;
    token_type: string;
  },
): void {
  try {
    if (tokens.refresh_token) setConfigKeyForEnv(env, "auth0_refresh_token", tokens.refresh_token);
    setConfigKeyForEnv(env, "auth0_access_token", tokens.access_token);
    setConfigKeyForEnv(env, "auth0_expires_in", tokens.expires_in);
    setConfigKeyForEnv(env, "auth0_scope", tokens.scope);
    setConfigKeyForEnv(env, "auth0_token_type", tokens.token_type);

    if (tokens.id_token) {
      setConfigKeyForEnv(env, "auth0_id_token", tokens.id_token);
    }
  } catch (error) {
    throw new Error(`unable to write tokens to config : ${error.message}`);
  }
}

// this is how we "logout"
export function deleteTokensInConfig(env: string) {
  deleteConfigKeyForEnv(env, "auth0_refresh_token");
  deleteConfigKeyForEnv(env, "auth0_access_token");
  deleteConfigKeyForEnv(env, "auth0_expires_in");
  deleteConfigKeyForEnv(env, "auth0_scope");
  deleteConfigKeyForEnv(env, "auth0_token_type");
  deleteConfigKeyForEnv(env, "auth0_id_token");
}

export async function getAccessTokenWithPrompts(env: string): Promise<string> {
  var deviceCodeResp;

  try {
    const accessToken = getConfigAccessToken(env);
    if (accessToken) {
      try {
        // validate (but not signature check!) the saved JWT
        // this is primarily to avoid sending API req with
        // expired token.
        const { header, payload, signature } = validate(
          decode(accessToken),
        );

        // console.log(header)
        // console.log(payload)
        // console.log(signature)
        if (header && payload && signature) {
          // structurally valid and unexpired JWT
          return new Promise((resolve) => {
            resolve(accessToken);
          });
        }
      } catch {
        const tokens = await getNewTokensWithRefreshToken(env);
        if (tokens) {
          setTokensInConfig(env, tokens);
          if (tokens.access_token) {
            return new Promise((resolve) => {
              resolve(tokens.access_token);
            });
          }
        } else {
          // unable to retrieve new access tokens using refresh token, cleanup saved tokens
          deleteTokensInConfig(env);
        }
      }
    }
  } catch (error) {
    console.error(colors.bold.red(`${error.message} error : exiting`));
    Deno.exit(1);
  }

  // No saved tokens found. Prompt the user to auth.
  try {
    deviceCodeResp = await getDeviceCode(env);
    // console.log(JSON.stringify(deviceCodeResp));
  } catch (error) {
    console.error(colors.bold.red(`${error.message} error : exiting`));
    Deno.exit(1);
  }

  if (deviceCodeResp && deviceCodeResp.verification_uri_complete) {
    console.log(colors.bold.yellow.underline(`\nAUTHENTICATION\n`));
    console.log(
      colors.bold.yellow(
        `Please authenticate yourself by visiting\nthe following URL in a browser:\n`,
      ),
    );
    console.log(
      colors.bold.underline.blue(deviceCodeResp.verification_uri_complete),
    );
    console.log("");
  } else {
    console.error(colors.bold.red(`no verification URI error : exiting`));
    Deno.exit(1);
  }

  try {
    const tokens = await getTokens(
      env,
      deviceCodeResp.device_code,
      deviceCodeResp.interval,
    );

    if (!tokens || !tokens.access_token) {
      throw new Error("retrieval of access tokens failed");
    }

    setTokensInConfig(env, tokens);

    return new Promise((resolve) => {
      resolve(tokens.access_token);
    });
  } catch (error) {
    console.error(colors.bold.red(`${error.message} error : exiting`));
    Deno.exit(1);
  }
}
