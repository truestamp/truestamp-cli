import { GetRoleCredentialsCommand, SSOClient } from "../client-sso/mod.ts";
import { getMasterProfileName, parseKnownFiles } from "../credential-provider-ini/mod.ts";
import { ProviderError } from "../property-provider/mod.ts";
import { getHomeDir } from "../shared-ini-file-loader/mod.ts";
import { createHash } from "https://deno.land/std@0.93.0/node/crypto.ts";
import { readFileSync } from "https://deno.land/std@0.93.0/node/fs.ts";
import { join } from "https://deno.land/std@0.93.0/node/path.ts";
export const EXPIRE_WINDOW_MS = 15 * 60 * 1000;
const SHOULD_FAIL_CREDENTIAL_CHAIN = false;
export const fromSSO = (init = {}) => async () => {
    const profiles = await parseKnownFiles(init);
    return resolveSSOCredentials(getMasterProfileName(init), profiles, init);
};
const resolveSSOCredentials = async (profileName, profiles, options) => {
    const profile = profiles[profileName];
    if (!profile) {
        throw new ProviderError(`Profile ${profileName} could not be found in shared credentials file.`);
    }
    const { sso_start_url: startUrl, sso_account_id: accountId, sso_region: region, sso_role_name: roleName } = profile;
    if (!startUrl && !accountId && !region && !roleName) {
        throw new ProviderError(`Profile ${profileName} is not configured with SSO credentials.`);
    }
    if (!startUrl || !accountId || !region || !roleName) {
        throw new ProviderError(`Profile ${profileName} does not have valid SSO credentials. Required parameters "sso_account_id", "sso_region", ` +
            `"sso_role_name", "sso_start_url". Reference: https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html`, SHOULD_FAIL_CREDENTIAL_CHAIN);
    }
    const hasher = createHash("sha1");
    const cacheName = hasher.update(startUrl).digest("hex");
    const tokenFile = join(getHomeDir(), ".aws", "sso", "cache", `${cacheName}.json`);
    let token;
    try {
        token = JSON.parse(readFileSync(tokenFile, { encoding: "utf-8" }));
        if (new Date(token.expiresAt).getTime() - Date.now() <= EXPIRE_WINDOW_MS) {
            throw new Error("SSO token is expired.");
        }
    }
    catch (e) {
        throw new ProviderError(`The SSO session associated with this profile has expired or is otherwise invalid. To refresh this SSO session ` +
            `run aws sso login with the corresponding profile.`, SHOULD_FAIL_CREDENTIAL_CHAIN);
    }
    const { accessToken } = token;
    const sso = options.ssoClient || new SSOClient({ region });
    let ssoResp;
    try {
        ssoResp = await sso.send(new GetRoleCredentialsCommand({
            accountId,
            roleName,
            accessToken,
        }));
    }
    catch (e) {
        throw ProviderError.from(e, SHOULD_FAIL_CREDENTIAL_CHAIN);
    }
    const { roleCredentials: { accessKeyId, secretAccessKey, sessionToken, expiration } = {} } = ssoResp;
    if (!accessKeyId || !secretAccessKey || !sessionToken || !expiration) {
        throw new ProviderError("SSO returns an invalid temporary credential.", SHOULD_FAIL_CREDENTIAL_CHAIN);
    }
    return { accessKeyId, secretAccessKey, sessionToken, expiration: new Date(expiration) };
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSx5QkFBeUIsRUFBbUMsU0FBUyxFQUFFLE1BQU0sc0JBQXNCLENBQUM7QUFDN0csT0FBTyxFQUFFLG9CQUFvQixFQUFFLGVBQWUsRUFBcUIsTUFBTSxtQ0FBbUMsQ0FBQztBQUM3RyxPQUFPLEVBQUUsYUFBYSxFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFDNUQsT0FBTyxFQUFFLFVBQVUsRUFBaUIsTUFBTSxrQ0FBa0MsQ0FBQztBQUU3RSxPQUFPLEVBQUUsVUFBVSxFQUFFLE1BQU0sNkNBQTZDLENBQUM7QUFDekUsT0FBTyxFQUFFLFlBQVksRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ3ZFLE9BQU8sRUFBRSxJQUFJLEVBQUUsTUFBTSwyQ0FBMkMsQ0FBQztBQVFqRSxNQUFNLENBQUMsTUFBTSxnQkFBZ0IsR0FBRyxFQUFFLEdBQUcsRUFBRSxHQUFHLElBQUksQ0FBQztBQUUvQyxNQUFNLDRCQUE0QixHQUFHLEtBQUssQ0FBQztBQXNCM0MsTUFBTSxDQUFDLE1BQU0sT0FBTyxHQUFHLENBQUMsT0FBb0IsRUFBRSxFQUFzQixFQUFFLENBQUMsS0FBSyxJQUFJLEVBQUU7SUFDaEYsTUFBTSxRQUFRLEdBQUcsTUFBTSxlQUFlLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDN0MsT0FBTyxxQkFBcUIsQ0FBQyxvQkFBb0IsQ0FBQyxJQUFJLENBQUMsRUFBRSxRQUFRLEVBQUUsSUFBSSxDQUFDLENBQUM7QUFDM0UsQ0FBQyxDQUFDO0FBRUYsTUFBTSxxQkFBcUIsR0FBRyxLQUFLLEVBQ2pDLFdBQW1CLEVBQ25CLFFBQXVCLEVBQ3ZCLE9BQW9CLEVBQ0UsRUFBRTtJQUN4QixNQUFNLE9BQU8sR0FBRyxRQUFRLENBQUMsV0FBVyxDQUFDLENBQUM7SUFDdEMsSUFBSSxDQUFDLE9BQU8sRUFBRTtRQUNaLE1BQU0sSUFBSSxhQUFhLENBQUMsV0FBVyxXQUFXLGlEQUFpRCxDQUFDLENBQUM7S0FDbEc7SUFDRCxNQUFNLEVBQUUsYUFBYSxFQUFFLFFBQVEsRUFBRSxjQUFjLEVBQUUsU0FBUyxFQUFFLFVBQVUsRUFBRSxNQUFNLEVBQUUsYUFBYSxFQUFFLFFBQVEsRUFBRSxHQUFHLE9BQU8sQ0FBQztJQUNwSCxJQUFJLENBQUMsUUFBUSxJQUFJLENBQUMsU0FBUyxJQUFJLENBQUMsTUFBTSxJQUFJLENBQUMsUUFBUSxFQUFFO1FBQ25ELE1BQU0sSUFBSSxhQUFhLENBQUMsV0FBVyxXQUFXLDBDQUEwQyxDQUFDLENBQUM7S0FDM0Y7SUFDRCxJQUFJLENBQUMsUUFBUSxJQUFJLENBQUMsU0FBUyxJQUFJLENBQUMsTUFBTSxJQUFJLENBQUMsUUFBUSxFQUFFO1FBQ25ELE1BQU0sSUFBSSxhQUFhLENBQ3JCLFdBQVcsV0FBVyw0RkFBNEY7WUFDaEgsc0hBQXNILEVBQ3hILDRCQUE0QixDQUM3QixDQUFDO0tBQ0g7SUFDRCxNQUFNLE1BQU0sR0FBRyxVQUFVLENBQUMsTUFBTSxDQUFDLENBQUM7SUFDbEMsTUFBTSxTQUFTLEdBQUcsTUFBTSxDQUFDLE1BQU0sQ0FBQyxRQUFRLENBQUMsQ0FBQyxNQUFNLENBQUMsS0FBSyxDQUFDLENBQUM7SUFDeEQsTUFBTSxTQUFTLEdBQUcsSUFBSSxDQUFDLFVBQVUsRUFBRSxFQUFFLE1BQU0sRUFBRSxLQUFLLEVBQUUsT0FBTyxFQUFFLEdBQUcsU0FBUyxPQUFPLENBQUMsQ0FBQztJQUNsRixJQUFJLEtBQWUsQ0FBQztJQUNwQixJQUFJO1FBQ0YsS0FBSyxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUMsWUFBWSxDQUFDLFNBQVMsRUFBRSxFQUFFLFFBQVEsRUFBRSxPQUFPLEVBQUUsQ0FBQyxDQUFDLENBQUM7UUFDbkUsSUFBSSxJQUFJLElBQUksQ0FBQyxLQUFLLENBQUMsU0FBUyxDQUFDLENBQUMsT0FBTyxFQUFFLEdBQUcsSUFBSSxDQUFDLEdBQUcsRUFBRSxJQUFJLGdCQUFnQixFQUFFO1lBQ3hFLE1BQU0sSUFBSSxLQUFLLENBQUMsdUJBQXVCLENBQUMsQ0FBQztTQUMxQztLQUNGO0lBQUMsT0FBTyxDQUFDLEVBQUU7UUFDVixNQUFNLElBQUksYUFBYSxDQUNyQixnSEFBZ0g7WUFDOUcsbURBQW1ELEVBQ3JELDRCQUE0QixDQUM3QixDQUFDO0tBQ0g7SUFDRCxNQUFNLEVBQUUsV0FBVyxFQUFFLEdBQUcsS0FBSyxDQUFDO0lBQzlCLE1BQU0sR0FBRyxHQUFHLE9BQU8sQ0FBQyxTQUFTLElBQUksSUFBSSxTQUFTLENBQUMsRUFBRSxNQUFNLEVBQUUsQ0FBQyxDQUFDO0lBQzNELElBQUksT0FBd0MsQ0FBQztJQUM3QyxJQUFJO1FBQ0YsT0FBTyxHQUFHLE1BQU0sR0FBRyxDQUFDLElBQUksQ0FDdEIsSUFBSSx5QkFBeUIsQ0FBQztZQUM1QixTQUFTO1lBQ1QsUUFBUTtZQUNSLFdBQVc7U0FDWixDQUFDLENBQ0gsQ0FBQztLQUNIO0lBQUMsT0FBTyxDQUFDLEVBQUU7UUFDVixNQUFNLGFBQWEsQ0FBQyxJQUFJLENBQUMsQ0FBQyxFQUFFLDRCQUE0QixDQUFDLENBQUM7S0FDM0Q7SUFDRCxNQUFNLEVBQUUsZUFBZSxFQUFFLEVBQUUsV0FBVyxFQUFFLGVBQWUsRUFBRSxZQUFZLEVBQUUsVUFBVSxFQUFFLEdBQUcsRUFBRSxFQUFFLEdBQUcsT0FBTyxDQUFDO0lBQ3JHLElBQUksQ0FBQyxXQUFXLElBQUksQ0FBQyxlQUFlLElBQUksQ0FBQyxZQUFZLElBQUksQ0FBQyxVQUFVLEVBQUU7UUFDcEUsTUFBTSxJQUFJLGFBQWEsQ0FBQyw4Q0FBOEMsRUFBRSw0QkFBNEIsQ0FBQyxDQUFDO0tBQ3ZHO0lBQ0QsT0FBTyxFQUFFLFdBQVcsRUFBRSxlQUFlLEVBQUUsWUFBWSxFQUFFLFVBQVUsRUFBRSxJQUFJLElBQUksQ0FBQyxVQUFVLENBQUMsRUFBRSxDQUFDO0FBQzFGLENBQUMsQ0FBQyJ9