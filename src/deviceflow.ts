// See: https://github.com/truestamp/deviceflow
// See: https://github.com/jatinvaidya/cli-authz-device-flow/blob/master/device/device.js

import { colors, sleep } from "./deps.ts";

const AUTH0_DOMAIN = "truestamp-dev.auth0.com";
const AUTH0_AUDIENCE = "https://dev-api.truestamp.com/";
const AUTH0_CLIENT_ID = "8djbT1Ys078OZImR1uRr4jhu2Wb6d05B";
const AUTH0_SCOPES = "openid profile offline_access";

async function getDeviceCode() {
  const resp = await fetch(`https://${AUTH0_DOMAIN}/oauth/device/code`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      client_id: AUTH0_CLIENT_ID,
      audience: AUTH0_AUDIENCE,
      scope: AUTH0_SCOPES,
    }),
  });
  return resp.json();
}

async function callTokenEndpoint(deviceCode: string): Promise<Response> {
  const resp = await fetch(`https://${AUTH0_DOMAIN}/oauth/token`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      client_id: AUTH0_CLIENT_ID,
      device_code: deviceCode,
      grant_type: "urn:ietf:params:oauth:grant-type:device_code",
    }),
  });
  return resp;
}

async function getAccessToken(deviceCode: string, interval: number) {
  let adjustedInterval = interval;
  let continuePolling = true;

  while (continuePolling) {
    await sleep(adjustedInterval);
    const resp = await callTokenEndpoint(deviceCode);

    if (resp.ok) {
      return await resp.json();
    }

    if (!resp.ok) {
      const respJson = await resp.json();

      switch (respJson.error) {
        case "authorization_pending":
          console.log(colors.bold.gray("authorization pending..."));
          break;

        case "slow_down":
          // add a second to the polling interval each time we receive this
          adjustedInterval += 1;
          console.log(
            colors.bold.yellow(
              `slow down requested : interval now every '${adjustedInterval}' seconds`,
            ),
          );
          break;

        case "expired_token":
          console.error(colors.bold.red("expired token : exiting"));
          continuePolling = false;
          break;

        case "access_denied":
          console.error(colors.bold.red("access denied : exiting"));
          continuePolling = false;
          break;

        default:
          console.error(`unknown error : ${JSON.stringify(respJson)}`);
          break;
      }
    }
  }
}

export async function getAccessTokenWithPrompts(): Promise<string> {
  const dc = await getDeviceCode();
  // console.log(JSON.stringify(dc));

  console.log(colors.bold.yellow.underline(`\nAUTHENTICATION\n`));
  console.log(
    colors.bold.yellow(
      `Please authenticate yourself by visiting\nthe following URL in a browser:\n`,
    ),
  );
  console.log(colors.bold.underline.blue(dc.verification_uri_complete));
  console.log("");

  const accessTokenResp = await getAccessToken(dc.device_code, dc.interval);

  if (accessTokenResp && accessTokenResp.access_token) {
    // console.log(JSON.stringify(accessTokenResp));
    return new Promise((resolve) => {
      resolve(accessTokenResp.access_token);
    });
  } else {
    throw new Error("missing or invalid access token");
  }
}
