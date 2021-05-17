import { colors, configDir, decode, loadJsonFile, sleep, validate, } from "./deps.ts";
const AUTH0_SCOPES = "openid profile email offline_access";
const AUTH0_DOMAIN_DEVELOPMENT = "truestamp-dev.auth0.com";
const AUTH0_AUDIENCE_DEVELOPMENT = "https://dev-api.truestamp.com/";
const AUTH0_CLIENT_ID_DEVELOPMENT = "8djbT1Ys078OZImR1uRr4jhu2Wb6d05B";
const AUTH0_TOKEN_FILE_DEVELOPMENT = `${configDir()}/com.truestamp.cli.development.tokens.json`;
const AUTH0_DOMAIN_STAGING = "truestamp-staging.auth0.com";
const AUTH0_AUDIENCE_STAGING = "https://staging-api.truestamp.com/";
const AUTH0_CLIENT_ID_STAGING = "T0dzxGnnIj3TU0HpzCQRTZ5fx9N5Hb5m";
const AUTH0_TOKEN_FILE_STAGING = `${configDir()}/com.truestamp.cli.staging.tokens.json`;
const AUTH0_DOMAIN_PRODUCTION = "login.truestamp.com";
const AUTH0_AUDIENCE_PRODUCTION = "https://api.truestamp.com/";
const AUTH0_CLIENT_ID_PRODUCTION = "pS5kRvqeuz4XLoxNPd6VX2LlUyNyU7Xj";
const AUTH0_TOKEN_FILE_PRODUCTION = `${configDir()}/com.truestamp.cli.production.tokens.json`;
function getAuth0DomainForEnv(env) {
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
function getAuth0AudienceForEnv(env) {
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
function getAuth0ClientIdForEnv(env) {
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
function getAuth0TokenFileForEnv(env) {
    switch (env) {
        case "development":
            return AUTH0_TOKEN_FILE_DEVELOPMENT;
        case "staging":
            return AUTH0_TOKEN_FILE_STAGING;
        case "production":
            return AUTH0_TOKEN_FILE_PRODUCTION;
        default:
            throw new Error(`invalid environment : '${env}'`);
    }
}
async function getDeviceCode(env) {
    const resp = await fetch(`https://${getAuth0DomainForEnv(env)}/oauth/device/code`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            client_id: getAuth0ClientIdForEnv(env),
            audience: getAuth0AudienceForEnv(env),
            scope: AUTH0_SCOPES,
        }),
    });
    return resp.json();
}
async function callTokenEndpoint(env, deviceCode) {
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
async function getTokens(env, deviceCode, interval) {
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
                    break;
                case "slow_down":
                    adjustedInterval += 1;
                    break;
                case "expired_token":
                    throw new Error(`expired token`);
                case "access_denied":
                    throw new Error(`access denied`);
                default:
                    throw new Error(`unknown error response : ${JSON.stringify(respJson)}`);
            }
        }
    }
}
async function getNewTokensWithRefreshToken(env) {
    const refreshToken = getSavedRefreshToken(env);
    if (refreshToken) {
        const resp = await fetch(`https://${getAuth0DomainForEnv(env)}/oauth/token`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                grant_type: "refresh_token",
                client_id: getAuth0ClientIdForEnv(env),
                refresh_token: refreshToken,
            }),
        });
        return await resp.json();
    }
}
export function getSavedAccessToken(env) {
    try {
        const t = loadJsonFile.sync(getAuth0TokenFileForEnv(env));
        if (t && t.access_token) {
            return t.access_token;
        }
        else {
            return undefined;
        }
    }
    catch {
    }
}
export function getSavedRefreshToken(env) {
    try {
        const t = loadJsonFile.sync(getAuth0TokenFileForEnv(env));
        if (t && t.refresh_token) {
            return t.refresh_token;
        }
        else {
            return undefined;
        }
    }
    catch {
    }
}
export function getSavedIdTokenPayload(env) {
    try {
        const t = loadJsonFile.sync(getAuth0TokenFileForEnv(env));
        if (t && t.id_token) {
            const { payload } = validate(decode(t.id_token));
            return payload;
        }
        else {
            return undefined;
        }
    }
    catch {
    }
}
function writeTokensToFile(env, tokens) {
    try {
        Deno.writeTextFileSync(getAuth0TokenFileForEnv(env), JSON.stringify(tokens));
    }
    catch (error) {
        throw new Error(`unable to write token file : ${error.message}`);
    }
}
export function deleteSavedTokens(env) {
    try {
        Deno.removeSync(getAuth0TokenFileForEnv(env));
    }
    catch {
    }
}
export async function getAccessTokenWithPrompts(env) {
    var deviceCodeResp;
    try {
        const savedAccessToken = getSavedAccessToken(env);
        if (savedAccessToken) {
            try {
                const { header, payload, signature } = validate(decode(savedAccessToken));
                if (header && payload && signature) {
                    return new Promise((resolve) => {
                        resolve(savedAccessToken);
                    });
                }
            }
            catch {
                const tokens = await getNewTokensWithRefreshToken(env);
                if (tokens) {
                    writeTokensToFile(env, tokens);
                    if (tokens.access_token) {
                        return new Promise((resolve) => {
                            resolve(tokens.access_token);
                        });
                    }
                }
                else {
                    deleteSavedTokens(env);
                }
            }
        }
    }
    catch (error) {
        console.error(colors.bold.red(`${error.message} error : exiting`));
        Deno.exit(1);
    }
    try {
        deviceCodeResp = await getDeviceCode(env);
    }
    catch (error) {
        console.error(colors.bold.red(`${error.message} error : exiting`));
        Deno.exit(1);
    }
    if (deviceCodeResp && deviceCodeResp.verification_uri_complete) {
        console.log(colors.bold.yellow.underline(`\nAUTHENTICATION\n`));
        console.log(colors.bold.yellow(`Please authenticate yourself by visiting\nthe following URL in a browser:\n`));
        console.log(colors.bold.underline.blue(deviceCodeResp.verification_uri_complete));
        console.log("");
    }
    else {
        console.error(colors.bold.red(`no verification URI error : exiting`));
        Deno.exit(1);
    }
    try {
        const tokens = await getTokens(env, deviceCodeResp.device_code, deviceCodeResp.interval);
        if (!tokens || !tokens.access_token || !tokens.refresh_token) {
            throw new Error("retrieval of access tokens failed");
        }
        writeTokensToFile(env, tokens);
        return new Promise((resolve) => {
            resolve(tokens.access_token);
        });
    }
    catch (error) {
        console.error(colors.bold.red(`${error.message} error : exiting`));
        Deno.exit(1);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYXV0aC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImF1dGgudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBR0EsT0FBTyxFQUNMLE1BQU0sRUFDTixTQUFTLEVBQ1QsTUFBTSxFQUNOLFlBQVksRUFFWixLQUFLLEVBQ0wsUUFBUSxHQUNULE1BQU0sV0FBVyxDQUFBO0FBRWxCLE1BQU0sWUFBWSxHQUFHLHFDQUFxQyxDQUFBO0FBRTFELE1BQU0sd0JBQXdCLEdBQUcseUJBQXlCLENBQUE7QUFDMUQsTUFBTSwwQkFBMEIsR0FBRyxnQ0FBZ0MsQ0FBQTtBQUNuRSxNQUFNLDJCQUEyQixHQUFHLGtDQUFrQyxDQUFBO0FBQ3RFLE1BQU0sNEJBQTRCLEdBQUcsR0FBRyxTQUFTLEVBQUUsNENBQTRDLENBQUE7QUFFL0YsTUFBTSxvQkFBb0IsR0FBRyw2QkFBNkIsQ0FBQTtBQUMxRCxNQUFNLHNCQUFzQixHQUFHLG9DQUFvQyxDQUFBO0FBQ25FLE1BQU0sdUJBQXVCLEdBQUcsa0NBQWtDLENBQUE7QUFDbEUsTUFBTSx3QkFBd0IsR0FBRyxHQUFHLFNBQVMsRUFBRSx3Q0FBd0MsQ0FBQTtBQUV2RixNQUFNLHVCQUF1QixHQUFHLHFCQUFxQixDQUFBO0FBQ3JELE1BQU0seUJBQXlCLEdBQUcsNEJBQTRCLENBQUE7QUFDOUQsTUFBTSwwQkFBMEIsR0FBRyxrQ0FBa0MsQ0FBQTtBQUNyRSxNQUFNLDJCQUEyQixHQUFHLEdBQUcsU0FBUyxFQUFFLDJDQUEyQyxDQUFBO0FBRTdGLFNBQVMsb0JBQW9CLENBQUMsR0FBVztJQUN2QyxRQUFRLEdBQUcsRUFBRTtRQUNYLEtBQUssYUFBYTtZQUNoQixPQUFPLHdCQUF3QixDQUFBO1FBRWpDLEtBQUssU0FBUztZQUNaLE9BQU8sb0JBQW9CLENBQUE7UUFFN0IsS0FBSyxZQUFZO1lBQ2YsT0FBTyx1QkFBdUIsQ0FBQTtRQUVoQztZQUNFLE1BQU0sSUFBSSxLQUFLLENBQUMsMEJBQTBCLEdBQUcsR0FBRyxDQUFDLENBQUE7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