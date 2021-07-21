import process from "https://deno.land/std@0.97.0/node/process.ts";
import { fromEnv } from "../credential-provider-env/mod.ts";
import { fromContainerMetadata, fromInstanceMetadata } from "../credential-provider-imds/mod.ts";
import { fromTokenFile } from "../credential-provider-web-identity/mod.ts";
import { ProviderError } from "../property-provider/mod.ts";
import { loadSharedConfigFiles, } from "../shared-ini-file-loader/mod.ts";
const DEFAULT_PROFILE = "default";
export const ENV_PROFILE = "AWS_PROFILE";
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
export const parseKnownFiles = async (init) => {
    const { loadedConfig = loadSharedConfigFiles(init) } = init;
    const parsedFiles = await loadedConfig;
    return {
        ...parsedFiles.configFile,
        ...parsedFiles.credentialsFile,
    };
};
export const getMasterProfileName = (init) => init.profile || process.env[ENV_PROFILE] || DEFAULT_PROFILE;
const resolveProfileData = async (profileName, profiles, options, visitedProfiles = {}) => {
    const data = profiles[profileName];
    if (Object.keys(visitedProfiles).length > 0 && isStaticCredsProfile(data)) {
        return resolveStaticCredentials(data);
    }
    if (isAssumeRoleWithSourceProfile(data) || isAssumeRoleWithProviderProfile(data)) {
        const { external_id: ExternalId, mfa_serial, role_arn: RoleArn, role_session_name: RoleSessionName = "aws-sdk-js-" + Date.now(), source_profile, credential_source, } = data;
        if (!options.roleAssumer) {
            throw new ProviderError(`Profile ${profileName} requires a role to be assumed, but no` + ` role assumption callback was provided.`, false);
        }
        if (source_profile && source_profile in visitedProfiles) {
            throw new ProviderError(`Detected a cycle attempting to resolve credentials for profile` +
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
                throw new ProviderError(`Profile ${profileName} requires multi-factor authentication,` + ` but no MFA code callback was provided.`, false);
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
    throw new ProviderError(`Profile ${profileName} could not be found or parsed in shared` + ` credentials file.`);
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
        throw new ProviderError(`Unsupported credential source in profile ${profileName}. Got ${credentialSource}, ` +
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sT0FBTyxNQUFNLDhDQUE4QyxDQUFDO0FBQ25FLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxtQ0FBbUMsQ0FBQztBQUM1RCxPQUFPLEVBQUUscUJBQXFCLEVBQUUsb0JBQW9CLEVBQUUsTUFBTSxvQ0FBb0MsQ0FBQztBQUNqRyxPQUFPLEVBQW1DLGFBQWEsRUFBRSxNQUFNLDRDQUE0QyxDQUFDO0FBQzVHLE9BQU8sRUFBRSxhQUFhLEVBQUUsTUFBTSw2QkFBNkIsQ0FBQztBQUM1RCxPQUFPLEVBQ0wscUJBQXFCLEdBS3RCLE1BQU0sa0NBQWtDLENBQUM7QUFHMUMsTUFBTSxlQUFlLEdBQUcsU0FBUyxDQUFDO0FBQ2xDLE1BQU0sQ0FBQyxNQUFNLFdBQVcsR0FBRyxhQUFhLENBQUM7QUFzRnpDLE1BQU0sb0JBQW9CLEdBQUcsQ0FBQyxHQUFRLEVBQTZCLEVBQUUsQ0FDbkUsT0FBTyxDQUFDLEdBQUcsQ0FBQztJQUNaLE9BQU8sR0FBRyxLQUFLLFFBQVE7SUFDdkIsT0FBTyxHQUFHLENBQUMsaUJBQWlCLEtBQUssUUFBUTtJQUN6QyxPQUFPLEdBQUcsQ0FBQyxxQkFBcUIsS0FBSyxRQUFRO0lBQzdDLENBQUMsV0FBVyxFQUFFLFFBQVEsQ0FBQyxDQUFDLE9BQU8sQ0FBQyxPQUFPLEdBQUcsQ0FBQyxpQkFBaUIsQ0FBQyxHQUFHLENBQUMsQ0FBQyxDQUFDO0FBUXJFLE1BQU0sb0JBQW9CLEdBQUcsQ0FBQyxHQUFRLEVBQTZCLEVBQUUsQ0FDbkUsT0FBTyxDQUFDLEdBQUcsQ0FBQztJQUNaLE9BQU8sR0FBRyxLQUFLLFFBQVE7SUFDdkIsT0FBTyxHQUFHLENBQUMsdUJBQXVCLEtBQUssUUFBUTtJQUMvQyxPQUFPLEdBQUcsQ0FBQyxRQUFRLEtBQUssUUFBUTtJQUNoQyxDQUFDLFdBQVcsRUFBRSxRQUFRLENBQUMsQ0FBQyxPQUFP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