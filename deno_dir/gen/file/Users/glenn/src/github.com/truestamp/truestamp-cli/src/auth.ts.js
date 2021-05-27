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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYXV0aC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImF1dGgudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBS0EsT0FBTyxFQUNMLE1BQU0sRUFDTixTQUFTLEVBQ1QsTUFBTSxFQUNOLFlBQVksRUFFWixLQUFLLEVBQ0wsUUFBUSxHQUNULE1BQU0sV0FBVyxDQUFDO0FBRW5CLE1BQU0sWUFBWSxHQUFHLHFDQUFxQyxDQUFDO0FBRTNELE1BQU0sd0JBQXdCLEdBQUcseUJBQXlCLENBQUM7QUFDM0QsTUFBTSwwQkFBMEIsR0FBRyxnQ0FBZ0MsQ0FBQztBQUNwRSxNQUFNLDJCQUEyQixHQUFHLGtDQUFrQyxDQUFDO0FBQ3ZFLE1BQU0sNEJBQTRCLEdBQ2hDLEdBQUcsU0FBUyxFQUFFLDRDQUE0QyxDQUFDO0FBRTdELE1BQU0sb0JBQW9CLEdBQUcsNkJBQTZCLENBQUM7QUFDM0QsTUFBTSxzQkFBc0IsR0FBRyxvQ0FBb0MsQ0FBQztBQUNwRSxNQUFNLHVCQUF1QixHQUFHLGtDQUFrQyxDQUFDO0FBQ25FLE1BQU0sd0JBQXdCLEdBQzVCLEdBQUcsU0FBUyxFQUFFLHdDQUF3QyxDQUFDO0FBRXpELE1BQU0sdUJBQXVCLEdBQUcscUJBQXFCLENBQUM7QUFDdEQsTUFBTSx5QkFBeUIsR0FBRyw0QkFBNEIsQ0FBQztBQUMvRCxNQUFNLDBCQUEwQixHQUFHLGtDQUFrQyxDQUFDO0FBQ3RFLE1BQU0sMkJBQTJCLEdBQy9CLEdBQUcsU0FBUyxFQUFFLDJDQUEyQyxDQUFDO0FBRTVELFNBQVMsb0JBQW9CLENBQUMsR0FBVztJQUN2QyxRQUFRLEdBQUcsRUFBRTtRQUNYLEtBQUssYUFBYTtZQUNoQixPQUFPLHdCQUF3QixDQUFDO1FBRWxDLEtBQUssU0FBUztZQUNaLE9BQU8sb0JBQW9CLENBQUM7UUFFOUIsS0FBSyxZQUFZO1lBQ2YsT0FBTyx1QkFBdUIsQ0FBQztRQUVqQztZQUNFLE1BQU0sSUFBSSxLQUFLLENBQUMsMEJBQTBCLEdBQUcsR0FBRyxDQUFDLENB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