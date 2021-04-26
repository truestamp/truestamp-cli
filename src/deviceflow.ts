// See: https://github.com/truestamp/deviceflow
// See: https://github.com/jatinvaidya/cli-authz-device-flow/blob/master/device/device.js

import {
  colors,
  configDir,
  decode,
  loadJsonFile,
  sleep,
  validate,
} from "./deps.ts"

// FIXME : These are environment specific and should return the right values.
const AUTH0_DOMAIN = "truestamp-dev.auth0.com"
const AUTH0_AUDIENCE = "https://dev-api.truestamp.com/"
const AUTH0_CLIENT_ID = "8djbT1Ys078OZImR1uRr4jhu2Wb6d05B"
const AUTH0_SCOPES = "openid profile offline_access"

const ACCESS_TOKEN_FILE = `${configDir()}/com.truestamp.cli.access.json`
const REFRESH_TOKEN_FILE = `${configDir()}/com.truestamp.cli.refresh.json`

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
  })
  return resp.json()
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
  })
  return resp
}

// Get the whole token response object by polling until the
// user authenticates or fails at doing so.
async function getTokens(deviceCode: string, interval: number) {
  let adjustedInterval = interval

  while (true) {
    await sleep(adjustedInterval)
    const resp = await callTokenEndpoint(deviceCode)

    if (resp.ok) {
      return await resp.json()
    }

    if (!resp.ok) {
      const respJson = await resp.json()

      switch (respJson.error) {
        case "authorization_pending":
          // console.log(colors.bold.gray("authorization pending..."));
          break

        case "slow_down":
          // add a second to the polling interval each time received
          adjustedInterval += 1
          break

        case "expired_token":
          throw new Error(`expired token`)

        case "access_denied":
          throw new Error(`access denied`)

        default:
          throw new Error(
            `unknown error response : ${JSON.stringify(respJson)}`
          )
      }
    }
  }
}

async function getNewTokenWithRefreshToken(refreshToken: string) {
  const resp = await fetch(`https://${AUTH0_DOMAIN}/oauth/token`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      grant_type: "refresh_token",
      client_id: AUTH0_CLIENT_ID,
      refresh_token: refreshToken,
    }),
  })
  return await resp.json()
}

function getSavedAccessToken(): string | undefined {
  try {
    const t = loadJsonFile.sync<{
      access_token: string
      id_token?: string
      refresh_token?: string
      scope: string
      expires_in: number
      token_type: string
    }>(ACCESS_TOKEN_FILE)

    if (t && t.access_token) {
      return t.access_token
    } else {
      return undefined
    }
  } catch {
    // no-op
  }
}

function getSavedRefreshToken(): string | undefined {
  try {
    const t = loadJsonFile.sync<{
      refresh_token: string
    }>(REFRESH_TOKEN_FILE)

    if (t && t.refresh_token) {
      return t.refresh_token
    } else {
      return undefined
    }
  } catch {
    // no-op
  }
}

function writeAccessTokenToFile(tokens: {
  access_token: string
  id_token?: string
  refresh_token?: string
  scope: string
  expires_in: number
  token_type: string
}): void {
  try {
    Deno.writeTextFileSync(ACCESS_TOKEN_FILE, JSON.stringify(tokens))
  } catch (error) {
    throw new Error(`unable to write token file : ${error.message}`)
  }
}

function writeRefreshTokenToFile(refreshToken: string): void {
  try {
    Deno.writeTextFileSync(
      REFRESH_TOKEN_FILE,
      JSON.stringify({ refresh_token: refreshToken })
    )
  } catch (error) {
    throw new Error(`unable to write refresh token file : ${error.message}`)
  }
}

// this is how we "logout"
export function deleteSavedTokenFiles() {
  Deno.removeSync(REFRESH_TOKEN_FILE)
  Deno.removeSync(ACCESS_TOKEN_FILE)
}

export async function getAccessTokenWithPrompts(): Promise<string> {
  var deviceCodeResp

  try {
    const savedAccessToken = getSavedAccessToken()
    if (savedAccessToken) {
      try {
        // validate (but not signature check!) the saved JWT
        // this is primarily to avoid sending API req with
        // expired token.
        const { header, payload, signature } = validate(
          decode(savedAccessToken)
        )

        // console.log(header)
        // console.log(payload)
        // console.log(signature)
        if (header && payload && signature) {
          // structurally valid and unexpired JWT
          return new Promise((resolve) => {
            resolve(savedAccessToken)
          })
        }
      } catch (error) {
        // handle bad JWT by trying to refresh
        // if refresh fails, delete saved tokens file
        console.error(
          `BAD JWT : getting refresh token from file : ${error.message}`
        )

        const refreshToken = getSavedRefreshToken()
        if (refreshToken) {
          console.error("BAD JWT : refresh token found")
          const tokens = await getNewTokenWithRefreshToken(refreshToken)
          if (tokens) {
            console.error("BAD JWT : new tokens received using refresh token")
            writeAccessTokenToFile(tokens)
            console.error("BAD JWT : new access tokens written")
            if (tokens.access_token) {
              return new Promise((resolve) => {
                resolve(tokens.access_token)
              })
            }
          } else {
            // unable to retrieve new access tokens using refresh token, cleanup saved tokens
            deleteSavedTokenFiles()
          }
        } else {
          // no refresh token was found, cleanup saved tokens
          deleteSavedTokenFiles()
        }
      }
    }
  } catch (error) {
    console.error(colors.bold.red(`${error.message} error : exiting`))
    Deno.exit(1)
  }

  try {
    deviceCodeResp = await getDeviceCode()
    // console.log(JSON.stringify(deviceCodeResp));
  } catch (error) {
    console.error(colors.bold.red(`${error.message} error : exiting`))
    Deno.exit(1)
  }

  if (deviceCodeResp && deviceCodeResp.verification_uri_complete) {
    console.log(colors.bold.yellow.underline(`\nAUTHENTICATION\n`))
    console.log(
      colors.bold.yellow(
        `Please authenticate yourself by visiting\nthe following URL in a browser:\n`
      )
    )
    console.log(
      colors.bold.underline.blue(deviceCodeResp.verification_uri_complete)
    )
    console.log("")
  } else {
    console.error(colors.bold.red(`no verification URI error : exiting`))
    Deno.exit(1)
  }

  try {
    const tokens = await getTokens(
      deviceCodeResp.device_code,
      deviceCodeResp.interval
    )

    writeAccessTokenToFile(tokens)

    // keep the refresh token separately so the auth token
    // file can be completely overwritten. The auth token
    // structure doesn't echo back the refresh token when
    // its used to refresh.
    if (tokens && tokens.refresh_token) {
      writeRefreshTokenToFile(tokens.refresh_token)
    }

    return new Promise((resolve) => {
      resolve(tokens.access_token)
    })
  } catch (error) {
    console.error(colors.bold.red(`${error.message} error : exiting`))
    Deno.exit(1)
  }
}
