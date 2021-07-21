import { AssumeRoleCommand } from "./commands/AssumeRoleCommand.ts";
import { AssumeRoleWithWebIdentityCommand, } from "./commands/AssumeRoleWithWebIdentityCommand.ts";
const ASSUME_ROLE_DEFAULT_REGION = "us-east-1";
const decorateDefaultRegion = (region) => {
    if (typeof region !== "function") {
        return region === undefined ? ASSUME_ROLE_DEFAULT_REGION : region;
    }
    return async () => {
        try {
            return await region();
        }
        catch (e) {
            return ASSUME_ROLE_DEFAULT_REGION;
        }
    };
};
export const getDefaultRoleAssumer = (stsOptions, stsClientCtor) => {
    let stsClient;
    let closureSourceCreds;
    return async (sourceCreds, params) => {
        closureSourceCreds = sourceCreds;
        if (!stsClient) {
            const { logger, region, requestHandler } = stsOptions;
            stsClient = new stsClientCtor({
                logger,
                credentialDefaultProvider: () => async () => closureSourceCreds,
                region: decorateDefaultRegion(region || stsOptions.region),
                ...(requestHandler ? { requestHandler } : {}),
            });
        }
        const { Credentials } = await stsClient.send(new AssumeRoleCommand(params));
        if (!Credentials || !Credentials.AccessKeyId || !Credentials.SecretAccessKey) {
            throw new Error(`Invalid response from STS.assumeRole call with role ${params.RoleArn}`);
        }
        return {
            accessKeyId: Credentials.AccessKeyId,
            secretAccessKey: Credentials.SecretAccessKey,
            sessionToken: Credentials.SessionToken,
            expiration: Credentials.Expiration,
        };
    };
};
export const getDefaultRoleAssumerWithWebIdentity = (stsOptions, stsClientCtor) => {
    let stsClient;
    return async (params) => {
        if (!stsClient) {
            const { logger, region, requestHandler } = stsOptions;
            stsClient = new stsClientCtor({
                logger,
                region: decorateDefaultRegion(region || stsOptions.region),
                ...(requestHandler ? { requestHandler } : {}),
            });
        }
        const { Credentials } = await stsClient.send(new AssumeRoleWithWebIdentityCommand(params));
        if (!Credentials || !Credentials.AccessKeyId || !Credentials.SecretAccessKey) {
            throw new Error(`Invalid response from STS.assumeRoleWithWebIdentity call with role ${params.RoleArn}`);
        }
        return {
            accessKeyId: Credentials.AccessKeyId,
            secretAccessKey: Credentials.SecretAccessKey,
            sessionToken: Credentials.SessionToken,
            expiration: Credentials.Expiration,
        };
    };
};
export const decorateDefaultCredentialProvider = (provider) => (input) => provider({
    roleAssumer: getDefaultRoleAssumer(input, input.stsClientCtor),
    roleAssumerWithWebIdentity: getDefaultRoleAssumerWithWebIdentity(input, input.stsClientCtor),
    ...input,
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZGVmYXVsdFN0c1JvbGVBc3N1bWVycy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImRlZmF1bHRTdHNSb2xlQXNzdW1lcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBSUEsT0FBTyxFQUFFLGlCQUFpQixFQUEwQixNQUFNLGlDQUFpQyxDQUFDO0FBQzVGLE9BQU8sRUFDTCxnQ0FBZ0MsR0FFakMsTUFBTSxnREFBZ0QsQ0FBQztBQVF4RCxNQUFNLDBCQUEwQixHQUFHLFdBQVcsQ0FBQztBQUsvQyxNQUFNLHFCQUFxQixHQUFHLENBQUMsTUFBNkMsRUFBNkIsRUFBRTtJQUN6RyxJQUFJLE9BQU8sTUFBTSxLQUFLLFVBQVUsRUFBRTtRQUNoQyxPQUFPLE1BQU0sS0FBSyxTQUFTLENBQUMsQ0FBQyxDQUFDLDBCQUEwQixDQUFDLENBQUMsQ0FBQyxNQUFNLENBQUM7S0FDbkU7SUFDRCxPQUFPLEtBQUssSUFBSSxFQUFFO1FBQ2hCLElBQUk7WUFDRixPQUFPLE1BQU0sTUFBTSxFQUFFLENBQUM7U0FDdkI7UUFBQyxPQUFPLENBQUMsRUFBRTtZQUNWLE9BQU8sMEJBQTBCLENBQUM7U0FDbkM7SUFDSCxDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7QUFNRixNQUFNLENBQUMsTUFBTSxxQkFBcUIsR0FBRyxDQUNuQyxVQUF5RSxFQUN6RSxhQUEwRCxFQUM3QyxFQUFFO0lBQ2YsSUFBSSxTQUFvQixDQUFDO0lBQ3pCLElBQUksa0JBQStCLENBQUM7SUFDcEMsT0FBTyxLQUFLLEVBQUUsV0FBVyxFQUFFLE1BQU0sRUFBRSxFQUFFO1FBQ25DLGtCQUFrQixHQUFHLFdBQVcsQ0FBQztRQUNqQyxJQUFJLENBQUMsU0FBUyxFQUFFO1lBQ2QsTUFBTSxFQUFFLE1BQU0sRUFBRSxNQUFNLEVBQUUsY0FBYyxFQUFFLEdBQUcsVUFBVSxDQUFDO1lBQ3RELFNBQVMsR0FBRyxJQUFJLGFBQWEsQ0FBQztnQkFDNUIsTUFBTTtnQkFFTix5QkFBeUIsRUFBRSxHQUFHLEVB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