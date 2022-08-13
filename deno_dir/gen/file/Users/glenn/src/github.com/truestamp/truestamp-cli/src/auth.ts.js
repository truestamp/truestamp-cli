// Copyright © 2020-2022 Truestamp Inc. All rights reserved.
// See: https://github.com/truestamp/deviceflow
// See: https://github.com/jatinvaidya/cli-authz-device-flow/blob/master/device/device.js
// On MacOS the config files can be found in a location like:
// cat ~/Library/Preferences/com.truestamp.cli.development/config.json
import { colors, decode, sleep, validate } from "./deps.ts";
import { deleteConfigKeyForEnv, getConfigKeyForEnv, setConfigKeyForEnv } from "./config.ts";
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
function getAuth0DomainForEnv(env) {
    switch(env){
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
    switch(env){
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
    switch(env){
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
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            client_id: getAuth0ClientIdForEnv(env),
            audience: getAuth0AudienceForEnv(env),
            scope: AUTH0_SCOPES
        })
    });
    return resp.json();
}
async function callTokenEndpoint(env, deviceCode) {
    const resp = await fetch(`https://${getAuth0DomainForEnv(env)}/oauth/token`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            client_id: getAuth0ClientIdForEnv(env),
            device_code: deviceCode,
            grant_type: "urn:ietf:params:oauth:grant-type:device_code"
        })
    });
    return resp;
}
// Get the whole token response object by polling until the
// user authenticates or fails at doing so.
async function getTokens(env, deviceCode, interval) {
    let adjustedInterval = interval;
    while(true){
        await sleep(adjustedInterval);
        const resp = await callTokenEndpoint(env, deviceCode);
        if (resp.ok) {
            return await resp.json();
        }
        if (!resp.ok) {
            const respJson = await resp.json();
            switch(respJson.error){
                case "authorization_pending":
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
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                grant_type: "refresh_token",
                client_id: getAuth0ClientIdForEnv(env),
                refresh_token: refreshToken
            })
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
    let idToken;
    try {
        idToken = getConfigKeyForEnv(env, "auth0_id_token");
    } catch (error) {
        throw new Error(`no id token found in config : ${error.message}`);
    }
    if (!idToken) throw new Error(`missing id token`);
    try {
        const { payload  } = validate(decode(idToken));
        return payload;
    } catch (error1) {
        throw new Error(`invalid id token : ${error1.message}`);
    }
}
function setTokensInConfig(env, tokens) {
    try {
        if (tokens.refresh_token) {
            setConfigKeyForEnv(env, "auth0_refresh_token", tokens.refresh_token);
        }
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
export function deleteTokensInConfig(env) {
    deleteConfigKeyForEnv(env, "auth0_refresh_token");
    deleteConfigKeyForEnv(env, "auth0_access_token");
    deleteConfigKeyForEnv(env, "auth0_expires_in");
    deleteConfigKeyForEnv(env, "auth0_scope");
    deleteConfigKeyForEnv(env, "auth0_token_type");
    deleteConfigKeyForEnv(env, "auth0_id_token");
}
export async function getAccessTokenWithPrompts(env) {
    let deviceCodeResp;
    try {
        const accessToken = getConfigAccessToken(env);
        if (accessToken) {
            try {
                // validate (but not signature check!) the saved JWT
                // this is primarily to avoid sending API req with
                // expired token.
                const { header , payload , signature  } = validate(decode(accessToken));
                // console.log(header)
                // console.log(payload)
                // console.log(signature)
                if (header && payload && signature) {
                    // structurally valid and unexpired JWT
                    return new Promise((resolve)=>{
                        resolve(accessToken);
                    });
                }
            } catch  {
                const tokens = await getNewTokensWithRefreshToken(env);
                if (tokens) {
                    setTokensInConfig(env, tokens);
                    if (tokens.access_token) {
                        return new Promise((resolve)=>{
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
    } catch (error1) {
        console.error(colors.bold.red(`${error1.message} error : exiting`));
        Deno.exit(1);
    }
    if (deviceCodeResp && deviceCodeResp.verification_uri_complete) {
        console.log(colors.bold.yellow.underline(`\nAUTHENTICATION\n`));
        console.log(colors.bold.yellow(`Please authenticate yourself by visiting\nthe following URL in a browser:\n`));
        console.log(colors.bold.underline.blue(deviceCodeResp.verification_uri_complete));
        console.log("");
    } else {
        console.error(colors.bold.red(`no verification URI error : exiting`));
        Deno.exit(1);
    }
    try {
        const tokens1 = await getTokens(env, deviceCodeResp.device_code, deviceCodeResp.interval);
        if (!tokens1 || !tokens1.access_token) {
            throw new Error("retrieval of access tokens failed");
        }
        setTokensInConfig(env, tokens1);
        return new Promise((resolve)=>{
            resolve(tokens1.access_token);
        });
    } catch (error2) {
        console.error(colors.bold.red(`${error2.message} error : exiting`));
        Deno.exit(1);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImZpbGU6Ly8vVXNlcnMvZ2xlbm4vc3JjL2dpdGh1Yi5jb20vdHJ1ZXN0YW1wL3RydWVzdGFtcC1jbGkvc3JjL2F1dGgudHMiXSwic291cmNlc0NvbnRlbnQiOlsiLy8gQ29weXJpZ2h0IMKpIDIwMjAtMjAyMiBUcnVlc3RhbXAgSW5jLiBBbGwgcmlnaHRzIHJlc2VydmVkLlxuXG4vLyBTZWU6IGh0dHBzOi8vZ2l0aHViLmNvbS90cnVlc3RhbXAvZGV2aWNlZmxvd1xuLy8gU2VlOiBodHRwczovL2dpdGh1Yi5jb20vamF0aW52YWlkeWEvY2xpLWF1dGh6LWRldmljZS1mbG93L2Jsb2IvbWFzdGVyL2RldmljZS9kZXZpY2UuanNcblxuLy8gT24gTWFjT1MgdGhlIGNvbmZpZyBmaWxlcyBjYW4gYmUgZm91bmQgaW4gYSBsb2NhdGlvbiBsaWtlOlxuLy8gY2F0IH4vTGlicmFyeS9QcmVmZXJlbmNlcy9jb20udHJ1ZXN0YW1wLmNsaS5kZXZlbG9wbWVudC9jb25maWcuanNvblxuXG5pbXBvcnQgeyBjb2xvcnMsIGRlY29kZSwgUGF5bG9hZCwgc2xlZXAsIHZhbGlkYXRlIH0gZnJvbSBcIi4vZGVwcy50c1wiO1xuXG5pbXBvcnQge1xuICBkZWxldGVDb25maWdLZXlGb3JFbnYsXG4gIGdldENvbmZpZ0tleUZvckVudixcbiAgc2V0Q29uZmlnS2V5Rm9yRW52LFxufSBmcm9tIFwiLi9jb25maWcudHNcIjtcblxuY29uc3QgQVVUSDBfU0NPUEVTID0gXCJvcGVuaWQgcHJvZmlsZSBlbWFpbCBvZmZsaW5lX2FjY2Vzc1wiO1xuXG5jb25zdCBBVVRIMF9ET01BSU5fREVWRUxPUE1FTlQgPSBcInRydWVzdGFtcC1kZXYuYXV0aDAuY29tXCI7XG5jb25zdCBBVVRIMF9BVURJRU5DRV9ERVZFTE9QTUVOVCA9IFwiaHR0cHM6Ly9kYi5mYXVuYS5jb20vZGIveXRpamhmcmVneWRmeVwiO1xuY29uc3QgQVVUSDBfQ0xJRU5UX0lEX0RFVkVMT1BNRU5UID0gXCI4ZGpiVDFZczA3OE9aSW1SMXVScjRqaHUyV2I2ZDA1QlwiO1xuXG5jb25zdCBBVVRIMF9ET01BSU5fU1RBR0lORyA9IFwidHJ1ZXN0YW1wLXN0YWdpbmcuYXV0aDAuY29tXCI7XG5jb25zdCBBVVRIMF9BVURJRU5DRV9TVEFHSU5HID0gXCJodHRwczovL2RiLmZhdW5hLmNvbS9kYi95dGlqaGRyY2V5YmZ5XCI7XG5jb25zdCBBVVRIMF9DTElFTlRfSURfU1RBR0lORyA9IFwiVDBkenhHbm5JajNUVTBIcHpDUVJUWjVmeDlONUhiNW1cIjtcblxuY29uc3QgQVVUSDBfRE9NQUlOX1BST0RVQ1RJT04gPSBcImxvZ2luLnRydWVzdGFtcC5jb21cIjtcbmNvbnN0IEFVVEgwX0FVRElFTkNFX1BST0RVQ1RJT04gPSBcImh0dHBzOi8vZGIuZmF1bmEuY29tL2RiL3l0aWo1OTViNnlmZnlcIjtcbmNvbnN0IEFVVEgwX0NMSUVOVF9JRF9QUk9EVUNUSU9OID0gXCJwUzVrUnZxZXV6NFhMb3hOUGQ2VlgyTGxVeU55VTdYalwiO1xuXG5mdW5jdGlvbiBnZXRBdXRoMERvbWFpbkZvckVudihlbnY6IHN0cmluZyk6IHN0cmluZyB7XG4gIHN3aXRjaCAoZW52KSB7XG4gICAgY2FzZSBcImRldmVsb3BtZW50XCI6XG4gICAgICByZXR1cm4gQVVUSDBfRE9NQUlOX0RFVkVMT1BNRU5UO1xuXG4gICAgY2FzZSBcInN0YWdpbmdcIjpcbiAgICAgIHJldHVybiBBVVRIMF9ET01BSU5fU1RBR0lORztcblxuICAgIGNhc2UgXCJwcm9kdWN0aW9uXCI6XG4gICAgICByZXR1cm4gQVVUSDBfRE9NQUlOX1BST0RVQ1RJT047XG5cbiAgICBkZWZhdWx0OlxuICAgICAgdGhyb3cgbmV3IEVycm9yKGBpbnZhbGlkIGVudmlyb25tZW50IDogJyR7ZW52fSdgKTtcbiAgfVxufVxuXG5mdW5jdGlvbiBnZXRBdXRoMEF1ZGllbmNlRm9yRW52KGVudjogc3RyaW5nKTogc3RyaW5nIHtcbiAgc3dpdGNoIChlbnYpIHtcbiAgICBjYXNlIFwiZGV2ZWxvcG1lbnRcIjpcbiAgICAgIHJldHVybiBBVVRIMF9BVURJRU5DRV9ERVZFTE9QTUVOVDtcblxuICAgIGNhc2UgXCJzdGFnaW5nXCI6XG4gICAgICByZXR1cm4gQVVUSDBfQVVESUVOQ0VfU1RBR0lORztcblxuICAgIGNhc2UgXCJwcm9kdWN0aW9uXCI6XG4gICAgICByZXR1cm4gQVVUSDBfQVVESUVOQ0VfUFJPRFVDVElPTjtcblxuICAgIGRlZmF1bHQ6XG4gICAgICB0aHJvdyBuZXcgRXJyb3IoYGludmFsaWQgZW52aXJvbm1lbnQgOiAnJHtlbnZ9J2ApO1xuICB9XG59XG5cbmZ1bmN0aW9uIGdldEF1dGgwQ2xpZW50SWRGb3JFbnYoZW52OiBzdHJpbmcpOiBzdHJpbmcge1xuICBzd2l0Y2ggKGVudikge1xuICAgIGNhc2UgXCJkZXZlbG9wbWVudFwiOlxuICAgICAgcmV0dXJuIEFVVEgwX0NMSUVOVF9JRF9ERVZFTE9QTUVOVDtcblxuICAgIGNhc2UgXCJzdGFnaW5nXCI6XG4gICAgICByZXR1cm4gQVVUSDBfQ0xJRU5UX0lEX1NUQUdJTkc7XG5cbiAgICBjYXNlIFwicHJvZHVjdGlvblwiOlxuICAgICAgcmV0dXJuIEFVVEgwX0NMSUVOVF9JRF9QUk9EVUNUSU9OO1xuXG4gICAgZGVmYXVsdDpcbiAgICAgIHRocm93IG5ldyBFcnJvcihgaW52YWxpZCBlbnZpcm9ubWVudCA6ICcke2Vudn0nYCk7XG4gIH1cbn1cblxuYXN5bmMgZnVuY3Rpb24gZ2V0RGV2aWNlQ29kZShlbnY6IHN0cmluZykge1xuICBjb25zdCByZXNwID0gYXdhaXQgZmV0Y2goXG4gICAgYGh0dHBzOi8vJHtnZXRBdXRoMERvbWFpbkZvckVudihlbnYpfS9vYXV0aC9kZXZpY2UvY29kZWAsXG4gICAge1xuICAgICAgbWV0aG9kOiBcIlBPU1RcIixcbiAgICAgIGhlYWRlcnM6IHtcbiAgICAgICAgXCJDb250ZW50LVR5cGVcIjogXCJhcHBsaWNhdGlvbi9qc29uXCIsXG4gICAgICB9LFxuICAgICAgYm9keTogSlNPTi5zdHJpbmdpZnkoe1xuICAgICAgICBjbGllbnRfaWQ6IGdldEF1dGgwQ2xpZW50SWRGb3JFbnYoZW52KSxcbiAgICAgICAgYXVkaWVuY2U6IGdldEF1dGgwQXVkaWVuY2VGb3JFbnYoZW52KSxcbiAgICAgICAgc2NvcGU6IEFVVEgwX1NDT1BFUyxcbiAgICAgIH0pLFxuICAgIH0sXG4gICk7XG4gIHJldHVybiByZXNwLmpzb24oKTtcbn1cblxuYXN5bmMgZnVuY3Rpb24gY2FsbFRva2VuRW5kcG9pbnQoXG4gIGVudjogc3RyaW5nLFxuICBkZXZpY2VDb2RlOiBzdHJpbmcsXG4pOiBQcm9taXNlPFJlc3BvbnNlPiB7XG4gIGNvbnN0IHJlc3AgPSBhd2FpdCBmZXRjaChgaHR0cHM6Ly8ke2dldEF1dGgwRG9tYWluRm9yRW52KGVudil9L29hdXRoL3Rva2VuYCwge1xuICAgIG1ldGhvZDogXCJQT1NUXCIsXG4gICAgaGVhZGVyczoge1xuICAgICAgXCJDb250ZW50LVR5cGVcIjogXCJhcHBsaWNhdGlvbi9qc29uXCIsXG4gICAgfSxcbiAgICBib2R5OiBKU09OLnN0cmluZ2lmeSh7XG4gICAgICBjbGllbnRfaWQ6IGdldEF1dGgwQ2xpZW50SWRGb3JFbnYoZW52KSxcbiAgICAgIGRldmljZV9jb2RlOiBkZXZpY2VDb2RlLFxuICAgICAgZ3JhbnRfdHlwZTogXCJ1cm46aWV0ZjpwYXJhbXM6b2F1dGg6Z3JhbnQtdHlwZTpkZXZpY2VfY29kZVwiLFxuICAgIH0pLFxuICB9KTtcbiAgcmV0dXJuIHJlc3A7XG59XG5cbi8vIEdldCB0aGUgd2hvbGUgdG9rZW4gcmVzcG9uc2Ugb2JqZWN0IGJ5IHBvbGxpbmcgdW50aWwgdGhlXG4vLyB1c2VyIGF1dGhlbnRpY2F0ZXMgb3IgZmFpbHMgYXQgZG9pbmcgc28uXG5hc3luYyBmdW5jdGlvbiBnZXRUb2tlbnMoZW52OiBzdHJpbmcsIGRldmljZUNvZGU6IHN0cmluZywgaW50ZXJ2YWw6IG51bWJlcikge1xuICBsZXQgYWRqdXN0ZWRJbnRlcnZhbCA9IGludGVydmFsO1xuXG4gIHdoaWxlICh0cnVlKSB7XG4gICAgYXdhaXQgc2xlZXAoYWRqdXN0ZWRJbnRlcnZhbCk7XG4gICAgY29uc3QgcmVzcCA9IGF3YWl0IGNhbGxUb2tlbkVuZHBvaW50KGVudiwgZGV2aWNlQ29kZSk7XG5cbiAgICBpZiAocmVzcC5vaykge1xuICAgICAgcmV0dXJuIGF3YWl0IHJlc3AuanNvbigpO1xuICAgIH1cblxuICAgIGlmICghcmVzcC5vaykge1xuICAgICAgY29uc3QgcmVzcEpzb24gPSBhd2FpdCByZXNwLmpzb24oKTtcblxuICAgICAgc3dpdGNoIChyZXNwSnNvbi5lcnJvcikge1xuICAgICAgICBjYXNlIFwiYXV0aG9yaXphdGlvbl9wZW5kaW5nXCI6XG4gICAgICAgICAgLy8gY29uc29sZS5sb2coY29sb3JzLmJvbGQuZ3JheShcImF1dGhvcml6YXRpb24gcGVuZGluZy4uLlwiKSk7XG4gICAgICAgICAgYnJlYWs7XG5cbiAgICAgICAgY2FzZSBcInNsb3dfZG93blwiOlxuICAgICAgICAgIC8vIGFkZCBhIHNlY29uZCB0byB0aGUgcG9sbGluZyBpbnRlcnZhbCBlYWNoIHRpbWUgcmVjZWl2ZWRcbiAgICAgICAgICBhZGp1c3RlZEludGVydmFsICs9IDE7XG4gICAgICAgICAgYnJlYWs7XG5cbiAgICAgICAgY2FzZSBcImV4cGlyZWRfdG9rZW5cIjpcbiAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoYGV4cGlyZWQgdG9rZW5gKTtcblxuICAgICAgICBjYXNlIFwiYWNjZXNzX2RlbmllZFwiOlxuICAgICAgICAgIHRocm93IG5ldyBFcnJvcihgYWNjZXNzIGRlbmllZGApO1xuXG4gICAgICAgIGRlZmF1bHQ6XG4gICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFxuICAgICAgICAgICAgYHVua25vd24gZXJyb3IgcmVzcG9uc2UgOiAke0pTT04uc3RyaW5naWZ5KHJlc3BKc29uKX1gLFxuICAgICAgICAgICk7XG4gICAgICB9XG4gICAgfVxuICB9XG59XG5cbmFzeW5jIGZ1bmN0aW9uIGdldE5ld1Rva2Vuc1dpdGhSZWZyZXNoVG9rZW4oZW52OiBzdHJpbmcpIHtcbiAgY29uc3QgcmVmcmVzaFRva2VuID0gZ2V0Q29uZmlnUmVmcmVzaFRva2VuKGVudik7XG4gIGlmIChyZWZyZXNoVG9rZW4pIHtcbiAgICBjb25zdCByZXNwID0gYXdhaXQgZmV0Y2goXG4gICAgICBgaHR0cHM6Ly8ke2dldEF1dGgwRG9tYWluRm9yRW52KGVudil9L29hdXRoL3Rva2VuYCxcbiAgICAgIHtcbiAgICAgICAgbWV0aG9kOiBcIlBPU1RcIixcbiAgICAgICAgaGVhZGVyczoge1xuICAgICAgICAgIFwiQ29udGVudC1UeXBlXCI6IFwiYXBwbGljYXRpb24vanNvblwiLFxuICAgICAgICB9LFxuICAgICAgICBib2R5OiBKU09OLnN0cmluZ2lmeSh7XG4gICAgICAgICAgZ3JhbnRfdHlwZTogXCJyZWZyZXNoX3Rva2VuXCIsXG4gICAgICAgICAgY2xpZW50X2lkOiBnZXRBdXRoMENsaWVudElkRm9yRW52KGVudiksXG4gICAgICAgICAgcmVmcmVzaF90b2tlbjogcmVmcmVzaFRva2VuLFxuICAgICAgICB9KSxcbiAgICAgIH0sXG4gICAgKTtcbiAgICByZXR1cm4gYXdhaXQgcmVzcC5qc29uKCk7XG4gIH1cbn1cblxuZXhwb3J0IGZ1bmN0aW9uIGdldENvbmZpZ0FjY2Vzc1Rva2VuKGVudjogc3RyaW5nKTogc3RyaW5nIHwgdW5kZWZpbmVkIHtcbiAgY29uc3QgdCA9IGdldENvbmZpZ0tleUZvckVudihlbnYsIFwiYXV0aDBfYWNjZXNzX3Rva2VuXCIpIGFzIHN0cmluZztcbiAgcmV0dXJuIHQgPyB0IDogdW5kZWZpbmVkO1xufVxuXG5leHBvcnQgZnVuY3Rpb24gZ2V0Q29uZmlnUmVmcmVzaFRva2VuKGVudjogc3RyaW5nKTogc3RyaW5nIHwgdW5kZWZpbmVkIHtcbiAgY29uc3QgdCA9IGdldENvbmZpZ0tleUZvckVudihlbnYsIFwiYXV0aDBfcmVmcmVzaF90b2tlblwiKSBhcyBzdHJpbmc7XG4gIHJldHVybiB0ID8gdCA6IHVuZGVmaW5lZDtcbn1cblxuZXhwb3J0IGZ1bmN0aW9uIGdldENvbmZpZ0lkVG9rZW5QYXlsb2FkKGVudjogc3RyaW5nKTogUGF5bG9hZCB7XG4gIGxldCBpZFRva2VuO1xuICB0cnkge1xuICAgIGlkVG9rZW4gPSBnZXRDb25maWdLZXlGb3JFbnYoZW52LCBcImF1dGgwX2lkX3Rva2VuXCIpIGFzIHN0cmluZztcbiAgfSBjYXRjaCAoZXJyb3IpIHtcbiAgICB0aHJvdyBuZXcgRXJyb3IoYG5vIGlkIHRva2VuIGZvdW5kIGluIGNvbmZpZyA6ICR7ZXJyb3IubWVzc2FnZX1gKTtcbiAgfVxuXG4gIGlmICghaWRUb2tlbikgdGhyb3cgbmV3IEVycm9yKGBtaXNzaW5nIGlkIHRva2VuYCk7XG5cbiAgdHJ5IHtcbiAgICBjb25zdCB7IHBheWxvYWQgfSA9IHZhbGlkYXRlKGRlY29kZShpZFRva2VuKSk7XG4gICAgcmV0dXJuIHBheWxvYWQ7XG4gIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgdGhyb3cgbmV3IEVycm9yKGBpbnZhbGlkIGlkIHRva2VuIDogJHtlcnJvci5tZXNzYWdlfWApO1xuICB9XG59XG5cbmZ1bmN0aW9uIHNldFRva2Vuc0luQ29uZmlnKFxuICBlbnY6IHN0cmluZyxcbiAgdG9rZW5zOiB7XG4gICAgYWNjZXNzX3Rva2VuOiBzdHJpbmc7XG4gICAgaWRfdG9rZW4/OiBzdHJpbmc7XG4gICAgcmVmcmVzaF90b2tlbj86IHN0cmluZztcbiAgICBzY29wZTogc3RyaW5nO1xuICAgIGV4cGlyZXNfaW46IG51bWJlcjtcbiAgICB0b2tlbl90eXBlOiBzdHJpbmc7XG4gIH0sXG4pOiB2b2lkIHtcbiAgdHJ5IHtcbiAgICBpZiAodG9rZW5zLnJlZnJlc2hfdG9rZW4pIHtcbiAgICAgIHNldENvbmZpZ0tleUZvckVudihlbnYsIFwiYXV0aDBfcmVmcmVzaF90b2tlblwiLCB0b2tlbnMucmVmcmVzaF90b2tlbik7XG4gICAgfVxuICAgIHNldENvbmZpZ0tleUZvckVudihlbnYsIFwiYXV0aDBfYWNjZXNzX3Rva2VuXCIsIHRva2Vucy5hY2Nlc3NfdG9rZW4pO1xuICAgIHNldENvbmZpZ0tleUZvckVudihlbnYsIFwiYXV0aDBfZXhwaXJlc19pblwiLCB0b2tlbnMuZXhwaXJlc19pbik7XG4gICAgc2V0Q29uZmlnS2V5Rm9yRW52KGVudiwgXCJhdXRoMF9zY29wZVwiLCB0b2tlbnMuc2NvcGUpO1xuICAgIHNldENvbmZpZ0tleUZvckVudihlbnYsIFwiYXV0aDBfdG9rZW5fdHlwZVwiLCB0b2tlbnMudG9rZW5fdHlwZSk7XG5cbiAgICBpZiAodG9rZW5zLmlkX3Rva2VuKSB7XG4gICAgICBzZXRDb25maWdLZXlGb3JFbnYoZW52LCBcImF1dGgwX2lkX3Rva2VuXCIsIHRva2Vucy5pZF90b2tlbik7XG4gICAgfVxuICB9IGNhdGNoIChlcnJvcikge1xuICAgIHRocm93IG5ldyBFcnJvcihgdW5hYmxlIHRvIHdyaXRlIHRva2VucyB0byBjb25maWcgOiAke2Vycm9yLm1lc3NhZ2V9YCk7XG4gIH1cbn1cblxuLy8gdGhpcyBpcyBob3cgd2UgXCJsb2dvdXRcIlxuZXhwb3J0IGZ1bmN0aW9uIGRlbGV0ZVRva2Vuc0luQ29uZmlnKGVudjogc3RyaW5nKTogdm9pZCB7XG4gIGRlbGV0ZUNvbmZpZ0tleUZvckVudihlbnYsIFwiYXV0aDBfcmVmcmVzaF90b2tlblwiKTtcbiAgZGVsZXRlQ29uZmlnS2V5Rm9yRW52KGVudiwgXCJhdXRoMF9hY2Nlc3NfdG9rZW5cIik7XG4gIGRlbGV0ZUNvbmZpZ0tleUZvckVudihlbnYsIFwiYXV0aDBfZXhwaXJlc19pblwiKTtcbiAgZGVsZXRlQ29uZmlnS2V5Rm9yRW52KGVudiwgXCJhdXRoMF9zY29wZVwiKTtcbiAgZGVsZXRlQ29uZmlnS2V5Rm9yRW52KGVudiwgXCJhdXRoMF90b2tlbl90eXBlXCIpO1xuICBkZWxldGVDb25maWdLZXlGb3JFbnYoZW52LCBcImF1dGgwX2lkX3Rva2VuXCIpO1xufVxuXG5leHBvcnQgYXN5bmMgZnVuY3Rpb24gZ2V0QWNjZXNzVG9rZW5XaXRoUHJvbXB0cyhlbnY6IHN0cmluZyk6IFByb21pc2U8c3RyaW5nPiB7XG4gIGxldCBkZXZpY2VDb2RlUmVzcDtcblxuICB0cnkge1xuICAgIGNvbnN0IGFjY2Vzc1Rva2VuID0gZ2V0Q29uZmlnQWNjZXNzVG9rZW4oZW52KTtcbiAgICBpZiAoYWNjZXNzVG9rZW4pIHtcbiAgICAgIHRyeSB7XG4gICAgICAgIC8vIHZhbGlkYXRlIChidXQgbm90IHNpZ25hdHVyZSBjaGVjayEpIHRoZSBzYXZlZCBKV1RcbiAgICAgICAgLy8gdGhpcyBpcyBwcmltYXJpbHkgdG8gYXZvaWQgc2VuZGluZyBBUEkgcmVxIHdpdGhcbiAgICAgICAgLy8gZXhwaXJlZCB0b2tlbi5cbiAgICAgICAgY29uc3QgeyBoZWFkZXIsIHBheWxvYWQsIHNpZ25hdHVyZSB9ID0gdmFsaWRhdGUoXG4gICAgICAgICAgZGVjb2RlKGFjY2Vzc1Rva2VuKSxcbiAgICAgICAgKTtcblxuICAgICAgICAvLyBjb25zb2xlLmxvZyhoZWFkZXIpXG4gICAgICAgIC8vIGNvbnNvbGUubG9nKHBheWxvYWQpXG4gICAgICAgIC8vIGNvbnNvbGUubG9nKHNpZ25hdHVyZSlcbiAgICAgICAgaWYgKGhlYWRlciAmJiBwYXlsb2FkICYmIHNpZ25hdHVyZSkge1xuICAgICAgICAgIC8vIHN0cnVjdHVyYWxseSB2YWxpZCBhbmQgdW5leHBpcmVkIEpXVFxuICAgICAgICAgIHJldHVybiBuZXcgUHJvbWlzZSgocmVzb2x2ZSkgPT4ge1xuICAgICAgICAgICAgcmVzb2x2ZShhY2Nlc3NUb2tlbik7XG4gICAgICAgICAgfSk7XG4gICAgICAgIH1cbiAgICAgIH0gY2F0Y2gge1xuICAgICAgICBjb25zdCB0b2tlbnMgPSBhd2FpdCBnZXROZXdUb2tlbnNXaXRoUmVmcmVzaFRva2VuKGVudik7XG4gICAgICAgIGlmICh0b2tlbnMpIHtcbiAgICAgICAgICBzZXRUb2tlbnNJbkNvbmZpZyhlbnYsIHRva2Vucyk7XG4gICAgICAgICAgaWYgKHRva2Vucy5hY2Nlc3NfdG9rZW4pIHtcbiAgICAgICAgICAgIHJldHVybiBuZXcgUHJvbWlzZSgocmVzb2x2ZSkgPT4ge1xuICAgICAgICAgICAgICByZXNvbHZlKHRva2Vucy5hY2Nlc3NfdG9rZW4pO1xuICAgICAgICAgICAgfSk7XG4gICAgICAgICAgfVxuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIC8vIHVuYWJsZSB0byByZXRyaWV2ZSBuZXcgYWNjZXNzIHRva2VucyB1c2luZyByZWZyZXNoIHRva2VuLCBjbGVhbnVwIHNhdmVkIHRva2Vuc1xuICAgICAgICAgIGRlbGV0ZVRva2Vuc0luQ29uZmlnKGVudik7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG4gIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgY29uc29sZS5lcnJvcihjb2xvcnMuYm9sZC5yZWQoYCR7ZXJyb3IubWVzc2FnZX0gZXJyb3IgOiBleGl0aW5nYCkpO1xuICAgIERlbm8uZXhpdCgxKTtcbiAgfVxuXG4gIC8vIE5vIHNhdmVkIHRva2VucyBmb3VuZC4gUHJvbXB0IHRoZSB1c2VyIHRvIGF1dGguXG4gIHRyeSB7XG4gICAgZGV2aWNlQ29kZVJlc3AgPSBhd2FpdCBnZXREZXZpY2VDb2RlKGVudik7XG4gICAgLy8gY29uc29sZS5sb2coSlNPTi5zdHJpbmdpZnkoZGV2aWNlQ29kZVJlc3ApKTtcbiAgfSBjYXRjaCAoZXJyb3IpIHtcbiAgICBjb25zb2xlLmVycm9yKGNvbG9ycy5ib2xkLnJlZChgJHtlcnJvci5tZXNzYWdlfSBlcnJvciA6IGV4aXRpbmdgKSk7XG4gICAgRGVuby5leGl0KDEpO1xuICB9XG5cbiAgaWYgKGRldmljZUNvZGVSZXNwICYmIGRldmljZUNvZGVSZXNwLnZlcmlmaWNhdGlvbl91cmlfY29tcGxldGUpIHtcbiAgICBjb25zb2xlLmxvZyhjb2xvcnMuYm9sZC55ZWxsb3cudW5kZXJsaW5lKGBcXG5BVVRIRU5USUNBVElPTlxcbmApKTtcbiAgICBjb25zb2xlLmxvZyhcbiAgICAgIGNvbG9ycy5ib2xkLnllbGxvdyhcbiAgICAgICAgYFBsZWFzZSBhdXRoZW50aWNhdGUgeW91cnNlbGYgYnkgdmlzaXRpbmdcXG50aGUgZm9sbG93aW5nIFVSTCBpbiBhIGJyb3dzZXI6XFxuYCxcbiAgICAgICksXG4gICAgKTtcbiAgICBjb25zb2xlLmxvZyhcbiAgICAgIGNvbG9ycy5ib2xkLnVuZGVybGluZS5ibHVlKGRldmljZUNvZGVSZXNwLnZlcmlmaWNhdGlvbl91cmlfY29tcGxldGUpLFxuICAgICk7XG4gICAgY29uc29sZS5sb2coXCJcIik7XG4gIH0gZWxzZSB7XG4gICAgY29uc29sZS5lcnJvcihjb2xvcnMuYm9sZC5yZWQoYG5vIHZlcmlmaWNhdGlvbiBVUkkgZXJyb3IgOiBleGl0aW5nYCkpO1xuICAgIERlbm8uZXhpdCgxKTtcbiAgfVxuXG4gIHRyeSB7XG4gICAgY29uc3QgdG9rZW5zID0gYXdhaXQgZ2V0VG9rZW5zKFxuICAgICAgZW52LFxuICAgICAgZGV2aWNlQ29kZVJlc3AuZGV2aWNlX2NvZGUsXG4gICAgICBkZXZpY2VDb2RlUmVzcC5pbnRlcnZhbCxcbiAgICApO1xuXG4gICAgaWYgKCF0b2tlbnMgfHwgIXRva2Vucy5hY2Nlc3NfdG9rZW4pIHtcbiAgICAgIHRocm93IG5ldyBFcnJvcihcInJldHJpZXZhbCBvZiBhY2Nlc3MgdG9rZW5zIGZhaWxlZFwiKTtcbiAgICB9XG5cbiAgICBzZXRUb2tlbnNJbkNvbmZpZyhlbnYsIHRva2Vucyk7XG5cbiAgICByZXR1cm4gbmV3IFByb21pc2UoKHJlc29sdmUpID0+IHtcbiAgICAgIHJlc29sdmUodG9rZW5zLmFjY2Vzc190b2tlbik7XG4gICAgfSk7XG4gIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgY29uc29sZS5lcnJvcihjb2xvcnMuYm9sZC5yZWQoYCR7ZXJyb3IubWVzc2FnZX0gZXJyb3IgOiBleGl0aW5nYCkpO1xuICAgIERlbm8uZXhpdCgxKTtcbiAgfVxufVxuIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLDREQUE0RDtBQUU1RCwrQ0FBK0M7QUFDL0MseUZBQXlG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