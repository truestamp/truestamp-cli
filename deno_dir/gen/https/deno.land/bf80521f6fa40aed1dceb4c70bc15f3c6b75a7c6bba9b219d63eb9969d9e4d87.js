import { ProviderError } from "../property-provider/mod.ts";
import { httpRequest } from "./remoteProvider/httpRequest.ts";
import { fromImdsCredentials, isImdsCredentials } from "./remoteProvider/ImdsCredentials.ts";
import { providerConfigFromInit } from "./remoteProvider/RemoteProviderInit.ts";
import { retry } from "./remoteProvider/retry.ts";
const IMDS_IP = "169.254.169.254";
const IMDS_PATH = "/latest/meta-data/iam/security-credentials/";
const IMDS_TOKEN_PATH = "/latest/api/token";
export const fromInstanceMetadata = (init = {}) => {
    let disableFetchToken = false;
    const { timeout, maxRetries } = providerConfigFromInit(init);
    const getCredentials = async (maxRetries, options) => {
        const profile = (await retry(async () => {
            let profile;
            try {
                profile = await getProfile(options);
            }
            catch (err) {
                if (err.statusCode === 401) {
                    disableFetchToken = false;
                }
                throw err;
            }
            return profile;
        }, maxRetries)).trim();
        return retry(async () => {
            let creds;
            try {
                creds = await getCredentialsFromProfile(profile, options);
            }
            catch (err) {
                if (err.statusCode === 401) {
                    disableFetchToken = false;
                }
                throw err;
            }
            return creds;
        }, maxRetries);
    };
    return async () => {
        if (disableFetchToken) {
            return getCredentials(maxRetries, { timeout });
        }
        else {
            let token;
            try {
                token = (await getMetadataToken({ timeout })).toString();
            }
            catch (error) {
                if (error?.statusCode === 400) {
                    throw Object.assign(error, {
                        message: "EC2 Metadata token request returned error",
                    });
                }
                else if (error.message === "TimeoutError" || [403, 404, 405].includes(error.statusCode)) {
                    disableFetchToken = true;
                }
                return getCredentials(maxRetries, { timeout });
            }
            return getCredentials(maxRetries, {
                timeout,
                headers: {
                    "x-aws-ec2-metadata-token": token,
                },
            });
        }
    };
};
const getMetadataToken = async (options) => httpRequest({
    ...options,
    host: IMDS_IP,
    path: IMDS_TOKEN_PATH,
    method: "PUT",
    headers: {
        "x-aws-ec2-metadata-token-ttl-seconds": "21600",
    },
});
const getProfile = async (options) => (await httpRequest({ ...options, host: IMDS_IP, path: IMDS_PATH })).toString();
const getCredentialsFromProfile = async (profile, options) => {
    const credsResponse = JSON.parse((await httpRequest({
        ...options,
        host: IMDS_IP,
        path: IMDS_PATH + profile,
    })).toString());
    if (!isImdsCredentials(credsResponse)) {
        throw new ProviderError("Invalid response received from instance metadata service.");
    }
    return fromImdsCredentials(credsResponse);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnJvbUluc3RhbmNlTWV0YWRhdGEuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJmcm9tSW5zdGFuY2VNZXRhZGF0YS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsYUFBYSxFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFJNUQsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLGlDQUFpQyxDQUFDO0FBQzlELE9BQU8sRUFBRSxtQkFBbUIsRUFBRSxpQkFBaUIsRUFBRSxNQUFNLHFDQUFxQyxDQUFDO0FBQzdGLE9BQU8sRUFBRSxzQkFBc0IsRUFBc0IsTUFBTSx3Q0FBd0MsQ0FBQztBQUNwRyxPQUFPLEVBQUUsS0FBSyxFQUFFLE1BQU0sMkJBQTJCLENBQUM7QUFFbEQsTUFBTSxPQUFPLEdBQUcsaUJBQWlCLENBQUM7QUFDbEMsTUFBTSxTQUFTLEdBQUcsNkNBQTZDLENBQUM7QUFDaEUsTUFBTSxlQUFlLEdBQUcsbUJBQW1CLENBQUM7QUFNNUMsTUFBTSxDQUFDLE1BQU0sb0JBQW9CLEdBQUcsQ0FBQyxPQUEyQixFQUFFLEVBQXNCLEVBQUU7SUFFeEYsSUFBSSxpQkFBaUIsR0FBRyxLQUFLLENBQUM7SUFDOUIsTUFBTSxFQUFFLE9BQU8sRUFBRSxVQUFVLEVBQUUsR0FBRyxzQkFBc0IsQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUU3RCxNQUFNLGNBQWMsR0FBRyxLQUFLLEVBQUUsVUFBa0IsRUFBRSxPQUF1QixFQUFFLEVBQUU7UUFDM0UsTUFBTSxPQUFPLEdBQUcsQ0FDZCxNQUFNLEtBQUssQ0FBUyxLQUFLLElBQUksRUFBRTtZQUM3QixJQUFJLE9BQWUsQ0FBQztZQUNwQixJQUFJO2dCQUNGLE9BQU8sR0FBRyxNQUFNLFVBQVUsQ0FBQyxPQUFPLENBQUMsQ0FBQzthQUNyQztZQUFDLE9BQU8sR0FBRyxFQUFFO2dCQUNaLElBQUksR0FBRyxDQUFDLFVBQVUsS0FBSyxHQUFHLEVBQUU7b0JBQzFCLGlCQUFpQixHQUFHLEtBQUssQ0FB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