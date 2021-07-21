import { fromEnv } from "../credential-provider-env/mod.ts";
import { fromContainerMetadata, fromInstanceMetadata } from "../credential-provider-imds/mod.ts";
import { fromSSO, isSsoProfile, validateSsoProfile } from "../credential-provider-sso/mod.ts";
import { fromTokenFile } from "../credential-provider-web-identity/mod.ts";
import { CredentialsProviderError } from "../property-provider/mod.ts";
import { getMasterProfileName, parseKnownFiles } from "../util-credentials/mod.ts";
const isStaticCredsProfile = (arg) => Boolean(arg) &&
    typeof arg === "object" &&
    typeof arg.aws_access_key_id === "string" &&
    typeof arg.aws_secret_access_key === "string" &&
    ["undefined", "string"].indexOf(typeof arg.aws_session_token) > -1;
const isWebIdentityProfile = (arg) => Boolean(arg) &&
    typeof arg === "object" &&
    typeof arg.web_identity_token_file === "string" &&
    typeof arg.role_arn === "string" &&
    ["undefined", "string"].indexOf(typeof arg.role_session_name) > -1;
const isAssumeRoleProfile = (arg) => Boolean(arg) &&
    typeof arg === "object" &&
    typeof arg.role_arn === "string" &&
    ["undefined", "string"].indexOf(typeof arg.role_session_name) > -1 &&
    ["undefined", "string"].indexOf(typeof arg.external_id) > -1 &&
    ["undefined", "string"].indexOf(typeof arg.mfa_serial) > -1;
const isAssumeRoleWithSourceProfile = (arg) => isAssumeRoleProfile(arg) && typeof arg.source_profile === "string" && typeof arg.credential_source === "undefined";
const isAssumeRoleWithProviderProfile = (arg) => isAssumeRoleProfile(arg) && typeof arg.credential_source === "string" && typeof arg.source_profile === "undefined";
export const fromIni = (init = {}) => async () => {
    const profiles = await parseKnownFiles(init);
    return resolveProfileData(getMasterProfileName(init), profiles, init);
};
const resolveProfileData = async (profileName, profiles, options, visitedProfiles = {}) => {
    const data = profiles[profileName];
    if (Object.keys(visitedProfiles).length > 0 && isStaticCredsProfile(data)) {
        return resolveStaticCredentials(data);
    }
    if (isAssumeRoleWithSourceProfile(data) || isAssumeRoleWithProviderProfile(data)) {
        const { external_id: ExternalId, mfa_serial, role_arn: RoleArn, role_session_name: RoleSessionName = "aws-sdk-js-" + Date.now(), source_profile, credential_source, } = data;
        if (!options.roleAssumer) {
            throw new CredentialsProviderError(`Profile ${profileName} requires a role to be assumed, but no` + ` role assumption callback was provided.`, false);
        }
        if (source_profile && source_profile in visitedProfiles) {
            throw new CredentialsProviderError(`Detected a cycle attempting to resolve credentials for profile` +
                ` ${getMasterProfileName(options)}. Profiles visited: ` +
                Object.keys(visitedProfiles).join(", "), false);
        }
        const sourceCreds = source_profile
            ? resolveProfileData(source_profile, profiles, options, {
                ...visitedProfiles,
                [source_profile]: true,
            })
            : resolveCredentialSource(credential_source, profileName)();
        const params = { RoleArn, RoleSessionName, ExternalId };
        if (mfa_serial) {
            if (!options.mfaCodeProvider) {
                throw new CredentialsProviderError(`Profile ${profileName} requires multi-factor authentication,` + ` but no MFA code callback was provided.`, false);
            }
            params.SerialNumber = mfa_serial;
            params.TokenCode = await options.mfaCodeProvider(mfa_serial);
        }
        return options.roleAssumer(await sourceCreds, params);
    }
    if (isStaticCredsProfile(data)) {
        return resolveStaticCredentials(data);
    }
    if (isWebIdentityProfile(data)) {
        return resolveWebIdentityCredentials(data, options);
    }
    if (isSsoProfile(data)) {
        const { sso_start_url, sso_account_id, sso_region, sso_role_name } = validateSsoProfile(data);
        return fromSSO({
            ssoStartUrl: sso_start_url,
            ssoAccountId: sso_account_id,
            ssoRegion: sso_region,
            ssoRoleName: sso_role_name,
        })();
    }
    throw new CredentialsProviderError(`Profile ${profileName} could not be found or parsed in shared` + ` credentials file.`);
};
const resolveCredentialSource = (credentialSource, profileName) => {
    const sourceProvidersMap = {
        EcsContainer: fromContainerMetadata,
        Ec2InstanceMetadata: fromInstanceMetadata,
        Environment: fromEnv,
    };
    if (credentialSource in sourceProvidersMap) {
        return sourceProvidersMap[credentialSource]();
    }
    else {
        throw new CredentialsProviderError(`Unsupported credential source in profile ${profileName}. Got ${credentialSource}, ` +
            `expected EcsContainer or Ec2InstanceMetadata or Environment.`);
    }
};
const resolveStaticCredentials = (profile) => Promise.resolve({
    accessKeyId: profile.aws_access_key_id,
    secretAccessKey: profile.aws_secret_access_key,
    sessionToken: profile.aws_session_token,
});
const resolveWebIdentityCredentials = async (profile, options) => fromTokenFile({
    webIdentityTokenFile: profile.web_identity_token_file,
    roleArn: profile.role_arn,
    roleSessionName: profile.role_session_name,
    roleAssumerWithWebIdentity: options.roleAssumerWithWebIdentity,
})();
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxtQ0FBbUMsQ0FBQztBQUM1RCxPQUFPLEVBQUUscUJBQXFCLEVBQUUsb0JBQW9CLEVBQUUsTUFBTSxvQ0FBb0MsQ0FBQztBQUNqRyxPQUFPLEVBQUUsT0FBTyxFQUFFLFlBQVksRUFBRSxrQkFBa0IsRUFBRSxNQUFNLG1DQUFtQyxDQUFDO0FBQzlGLE9BQU8sRUFBbUMsYUFBYSxFQUFFLE1BQU0sNENBQTRDLENBQUM7QUFDNUcsT0FBTyxFQUFFLHdCQUF3QixFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFHdkUsT0FBTyxFQUFFLG9CQUFvQixFQUFFLGVBQWUsRUFBcUIsTUFBTSw0QkFBNEIsQ0FBQztBQXVFdEcsTUFBTSxvQkFBb0IsR0FBRyxDQUFDLEdBQVEsRUFBNkIsRUFBRSxDQUNuRSxPQUFPLENBQUMsR0FBRyxDQUFDO0lBQ1osT0FBTyxHQUFHLEtBQUssUUFBUTtJQUN2QixPQUFPLEdBQUcsQ0FBQyxpQkFBaUIsS0FBSyxRQUFRO0lBQ3pDLE9BQU8sR0FBRyxDQUFDLHFCQUFxQixLQUFLLFFBQVE7SUFDN0MsQ0FBQyxXQUFXLEVBQUUsUUFBUSxDQUFDLENBQUMsT0FBTyxDQUFDLE9BQU8sR0FBRyxDQUFDLGlCQUFpQixDQUFDLEdBQUcsQ0FBQyxDQUFDLENBQUM7QUFRckUsTUFBTSxvQkFBb0IsR0FBRyxDQUFDLEdBQVEsRUFBNkIsRUFBRSxDQUNuRSxPQUFPLENBQUMsR0FBRyxDQUFDO0lBQ1osT0FBTyxHQUFHLEtBQUssUUFBUTtJQUN2QixPQUFPLEdBQUcsQ0FBQyx1QkFBdUIsS0FBSyxRQUFRO0lBQy9DLE9BQU8sR0FBRyxDQUFDLFFBQVEsS0FBSyxRQUFRO0lBQ2hDLENBQUMsV0FBVyxFQUFFLFFBQVEsQ0FBQyxDQUFDLE9BQU8sQ0FBQyxPQUFPLEdBQUcsQ0FBQyxpQkFBaUIs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