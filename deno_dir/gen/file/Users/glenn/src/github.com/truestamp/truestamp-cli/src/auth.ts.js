import { colors, decode, sleep, validate } from "./deps.ts";
import { deleteConfigKeyForEnv, getConfigKeyForEnv, setConfigKeyForEnv, } from "./config.ts";
const AUTH0_SCOPES = "openid profile email offline_access";
const AUTH0_DOMAIN_DEVELOPMENT = "truestamp-dev.auth0.com";
const AUTH0_AUDIENCE_DEVELOPMENT = "https://dev-api.truestamp.com/";
const AUTH0_CLIENT_ID_DEVELOPMENT = "8djbT1Ys078OZImR1uRr4jhu2Wb6d05B";
const AUTH0_DOMAIN_STAGING = "truestamp-staging.auth0.com";
const AUTH0_AUDIENCE_STAGING = "https://staging-api.truestamp.com/";
const AUTH0_CLIENT_ID_STAGING = "T0dzxGnnIj3TU0HpzCQRTZ5fx9N5Hb5m";
const AUTH0_DOMAIN_PRODUCTION = "login.truestamp.com";
const AUTH0_AUDIENCE_PRODUCTION = "https://api.truestamp.com/";
const AUTH0_CLIENT_ID_PRODUCTION = "pS5kRvqeuz4XLoxNPd6VX2LlUyNyU7Xj";
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
    const refreshToken = getConfigRefreshToken(env);
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
export function getConfigAccessToken(env) {
    const t = getConfigKeyForEnv(env, "auth0_access_token");
    return t ? t : undefined;
}
export function getConfigRefreshToken(env) {
    const t = getConfigKeyForEnv(env, "auth0_refresh_token");
    return t ? t : undefined;
}
export function getConfigIdTokenPayload(env) {
    const t = getConfigKeyForEnv(env, "auth0_id_token");
    if (t) {
        const { payload } = validate(decode(t));
        return payload;
    }
    else {
        return undefined;
    }
}
function setTokensInConfig(env, tokens) {
    try {
        setConfigKeyForEnv(env, "auth0_refresh_token", tokens.refresh_token);
        setConfigKeyForEnv(env, "auth0_access_token", tokens.access_token);
        setConfigKeyForEnv(env, "auth0_expires_in", tokens.expires_in);
        setConfigKeyForEnv(env, "auth0_scope", tokens.scope);
        setConfigKeyForEnv(env, "auth0_token_type", tokens.token_type);
        if (tokens.id_token) {
            setConfigKeyForEnv(env, "auth0_id_token", tokens.id_token);
        }
    }
    catch (error) {
        throw new Error(`unable to write tokens to config : ${error.message}`);
    }
}
export function deleteTokensInConfig(env) {
    deleteConfigKeyForEnv(env, "auth0_refresh_token");
    deleteConfigKeyForEnv(env, "auth0_access_token");
    deleteConfigKeyForEnv(env, "auth0_expires_in");
    deleteConfigKeyForEnv(env, "auth0_scope");
    deleteConfigKeyForEnv(env, "auth0_token_type");
    deleteConfigKeyForEnv(env, "auth0_id_token");
}
export async function getAccessTokenWithPrompts(env) {
    var deviceCodeResp;
    try {
        const accessToken = getConfigAccessToken(env);
        if (accessToken) {
            try {
                const { header, payload, signature } = validate(decode(accessToken));
                if (header && payload && signature) {
                    return new Promise((resolve) => {
                        resolve(accessToken);
                    });
                }
            }
            catch {
                const tokens = await getNewTokensWithRefreshToken(env);
                if (tokens) {
                    setTokensInConfig(env, tokens);
                    if (tokens.access_token) {
                        return new Promise((resolve) => {
                            resolve(tokens.access_token);
                        });
                    }
                }
                else {
                    deleteTokensInConfig(env);
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
        setTokensInConfig(env, tokens);
        return new Promise((resolve) => {
            resolve(tokens.access_token);
        });
    }
    catch (error) {
        console.error(colors.bold.red(`${error.message} error : exiting`));
        Deno.exit(1);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYXV0aC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImF1dGgudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBS0EsT0FBTyxFQUFFLE1BQU0sRUFBRSxNQUFNLEVBQVcsS0FBSyxFQUFFLFFBQVEsRUFBRSxNQUFNLFdBQVcsQ0FBQztBQUVyRSxPQUFPLEVBQ0wscUJBQXFCLEVBQ3JCLGtCQUFrQixFQUNsQixrQkFBa0IsR0FDbkIsTUFBTSxhQUFhLENBQUM7QUFFckIsTUFBTSxZQUFZLEdBQUcscUNBQXFDLENBQUM7QUFFM0QsTUFBTSx3QkFBd0IsR0FBRyx5QkFBeUIsQ0FBQztBQUMzRCxNQUFNLDBCQUEwQixHQUFHLGdDQUFnQyxDQUFDO0FBQ3BFLE1BQU0sMkJBQTJCLEdBQUcsa0NBQWtDLENBQUM7QUFFdkUsTUFBTSxvQkFBb0IsR0FBRyw2QkFBNkIsQ0FBQztBQUMzRCxNQUFNLHNCQUFzQixHQUFHLG9DQUFvQyxDQUFDO0FBQ3BFLE1BQU0sdUJBQXVCLEdBQUcsa0NBQWtDLENBQUM7QUFFbkUsTUFBTSx1QkFBdUIsR0FBRyxxQkFBcUIsQ0FBQztBQUN0RCxNQUFNLHlCQUF5QixHQUFHLDRCQUE0QixDQUFDO0FBQy9ELE1BQU0sMEJBQTBCLEdBQUcsa0NBQWtDLENBQUM7QUFFdEUsU0FBUyxvQkFBb0IsQ0FBQyxHQUFXO0lBQ3ZDLFFBQVEsR0FBRyxFQUFFO1FBQ1gsS0FBSyxhQUFhO1lBQ2hCLE9BQU8sd0JBQXdCLENBQUM7UUFFbEMsS0FBSyxTQUFTO1lBQ1osT0FBTyxvQkFBb0IsQ0FBQztRQUU5QixLQUFLLFlBQVk7WUFDZixPQUFPLHVCQUF1QixDQUFDO1FBRWpDO1lBQ0UsTUFBTSxJQUFJLEtBQUssQ0FBQywwQkFBMEIsR0FBRyxHQUFHLENBQUMsQ0FBQztLQUNyRDtBQUNILENBQUM7QUFFRCxTQUFTLHNCQUFzQixDQUFDLEdBQVc7SUFDekMsUUFBUSxHQUFHLEVBQUU7UUFDWCxLQUFLLGFBQWE7WUFDaEIsT0FBTywwQkFBMEIsQ0FB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