import { GetRoleCredentialsCommand, SSOClient } from "../client-sso/mod.ts";
import { CredentialsProviderError } from "../property-provider/mod.ts";
import { getHomeDir } from "../shared-ini-file-loader/mod.ts";
import { getMasterProfileName, parseKnownFiles } from "../util-credentials/mod.ts";
import { createHash } from "https://deno.land/std@0.101.0/node/crypto.ts";
import { readFileSync } from "https://deno.land/std@0.101.0/node/fs.ts";
import { join } from "https://deno.land/std@0.101.0/node/path.ts";
export const EXPIRE_WINDOW_MS = 15 * 60 * 1000;
const SHOULD_FAIL_CREDENTIAL_CHAIN = false;
export const fromSSO = (init = {}) => async () => {
    const { ssoStartUrl, ssoAccountId, ssoRegion, ssoRoleName, ssoClient } = init;
    if (!ssoStartUrl && !ssoAccountId && !ssoRegion && !ssoRoleName) {
        const profiles = await parseKnownFiles(init);
        const profileName = getMasterProfileName(init);
        const profile = profiles[profileName];
        if (!isSsoProfile(profile)) {
            throw new CredentialsProviderError(`Profile ${profileName} is not configured with SSO credentials.`);
        }
        const { sso_start_url, sso_account_id, sso_region, sso_role_name } = validateSsoProfile(profile);
        return resolveSSOCredentials({
            ssoStartUrl: sso_start_url,
            ssoAccountId: sso_account_id,
            ssoRegion: sso_region,
            ssoRoleName: sso_role_name,
            ssoClient: ssoClient,
        });
    }
    else if (!ssoStartUrl || !ssoAccountId || !ssoRegion || !ssoRoleName) {
        throw new CredentialsProviderError('Incomplete configuration. The fromSSO() argument hash must include "ssoStartUrl",' +
            ' "ssoAccountId", "ssoRegion", "ssoRoleName"');
    }
    else {
        return resolveSSOCredentials({ ssoStartUrl, ssoAccountId, ssoRegion, ssoRoleName, ssoClient });
    }
};
const resolveSSOCredentials = async ({ ssoStartUrl, ssoAccountId, ssoRegion, ssoRoleName, ssoClient, }) => {
    const hasher = createHash("sha1");
    const cacheName = hasher.update(ssoStartUrl).digest("hex");
    const tokenFile = join(getHomeDir(), ".aws", "sso", "cache", `${cacheName}.json`);
    let token;
    try {
        token = JSON.parse(readFileSync(tokenFile, { encoding: "utf-8" }));
        if (new Date(token.expiresAt).getTime() - Date.now() <= EXPIRE_WINDOW_MS) {
            throw new Error("SSO token is expired.");
        }
    }
    catch (e) {
        throw new CredentialsProviderError(`The SSO session associated with this profile has expired or is otherwise invalid. To refresh this SSO session ` +
            `run aws sso login with the corresponding profile.`, SHOULD_FAIL_CREDENTIAL_CHAIN);
    }
    const { accessToken } = token;
    const sso = ssoClient || new SSOClient({ region: ssoRegion });
    let ssoResp;
    try {
        ssoResp = await sso.send(new GetRoleCredentialsCommand({
            accountId: ssoAccountId,
            roleName: ssoRoleName,
            accessToken,
        }));
    }
    catch (e) {
        throw CredentialsProviderError.from(e, SHOULD_FAIL_CREDENTIAL_CHAIN);
    }
    const { roleCredentials: { accessKeyId, secretAccessKey, sessionToken, expiration } = {} } = ssoResp;
    if (!accessKeyId || !secretAccessKey || !sessionToken || !expiration) {
        throw new CredentialsProviderError("SSO returns an invalid temporary credential.", SHOULD_FAIL_CREDENTIAL_CHAIN);
    }
    return { accessKeyId, secretAccessKey, sessionToken, expiration: new Date(expiration) };
};
export const validateSsoProfile = (profile) => {
    const { sso_start_url, sso_account_id, sso_region, sso_role_name } = profile;
    if (!sso_start_url || !sso_account_id || !sso_region || !sso_role_name) {
        throw new CredentialsProviderError(`Profile is configured with invalid SSO credentials. Required parameters "sso_account_id", "sso_region", ` +
            `"sso_role_name", "sso_start_url". Got ${Object.keys(profile).join(", ")}\nReference: https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html`, SHOULD_FAIL_CREDENTIAL_CHAIN);
    }
    return profile;
};
export const isSsoProfile = (arg) => arg &&
    (typeof arg.sso_start_url === "string" ||
        typeof arg.sso_account_id === "string" ||
        typeof arg.sso_region === "string" ||
        typeof arg.sso_role_name === "string");
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSx5QkFBeUIsRUFBbUMsU0FBUyxFQUFFLE1BQU0sc0JBQXNCLENBQUM7QUFDN0csT0FBTyxFQUFFLHdCQUF3QixFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFDdkUsT0FBTyxFQUFFLFVBQVUsRUFBVyxNQUFNLGtDQUFrQyxDQUFDO0FBRXZFLE9BQU8sRUFBRSxvQkFBb0IsRUFBRSxlQUFlLEVBQXFCLE1BQU0sNEJBQTRCLENBQUM7QUFDdEcsT0FBTyxFQUFFLFVBQVUsRUFBRSxNQUFNLDhDQUE4QyxDQUFDO0FBQzFFLE9BQU8sRUFBRSxZQUFZLEVBQUUsTUFBTSwwQ0FBMEMsQ0FBQztBQUN4RSxPQUFPLEVBQUUsSUFBSSxFQUFFLE1BQU0sNENBQTRDLENBQUM7QUFRbEUsTUFBTSxDQUFDLE1BQU0sZ0JBQWdCLEdBQUcsRUFBRSxHQUFHLEVBQUUsR0FBRyxJQUFJLENBQUM7QUFFL0MsTUFBTSw0QkFBNEIsR0FBRyxLQUFLLENBQUM7QUEyQzNDLE1BQU0sQ0FBQyxNQUFNLE9BQU8sR0FDbEIsQ0FBQyxPQUF3RCxFQUFTLEVBQXNCLEVBQUUsQ0FDMUYsS0FBSyxJQUFJLEVBQUU7SUFDVCxNQUFNLEVBQUUsV0FBVyxFQUFFLFlBQVksRUFBRSxTQUFTLEVBQUUsV0FBVyxFQUFFLFNBQVMsRUFBRSxHQUFHLElBQUksQ0FBQztJQUM5RSxJQUFJLENBQUMsV0FBVyxJQUFJLENBQUMsWUFBWSxJQUFJLENBQUMsU0FBUyxJQUFJLENBQUMsV0FBVyxFQUFFO1FBRS9ELE1BQU0sUUFBUSxHQUFHLE1BQU0sZUFBZSxDQUFDLElBQUksQ0FBQyxDQUFDO1FBQzdDLE1BQU0sV0FBVyxHQUFHLG9CQUFvQixDQUFDLElBQUksQ0FBQyxDQUFDO1FBQy9DLE1BQU0sT0FBTyxHQUFHLFFBQVEsQ0FBQyxXQUFXLENBQUMsQ0FBQztRQUN0QyxJQUFJ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