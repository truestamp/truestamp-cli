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
            const { logger, region } = stsOptions;
            stsClient = new stsClientCtor({
                logger,
                credentialDefaultProvider: () => async () => closureSourceCreds,
                region: decorateDefaultRegion(region),
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
            const { logger, region } = stsOptions;
            stsClient = new stsClientCtor({
                logger,
                region: decorateDefaultRegion(region),
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZGVmYXVsdFN0c1JvbGVBc3N1bWVycy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImRlZmF1bHRTdHNSb2xlQXNzdW1lcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBSUEsT0FBTyxFQUFFLGlCQUFpQixFQUEwQixNQUFNLGlDQUFpQyxDQUFDO0FBQzVGLE9BQU8sRUFDTCxnQ0FBZ0MsR0FFakMsTUFBTSxnREFBZ0QsQ0FBQztBQVF4RCxNQUFNLDBCQUEwQixHQUFHLFdBQVcsQ0FBQztBQUsvQyxNQUFNLHFCQUFxQixHQUFHLENBQUMsTUFBNkMsRUFBNkIsRUFBRTtJQUN6RyxJQUFJLE9BQU8sTUFBTSxLQUFLLFVBQVUsRUFBRTtRQUNoQyxPQUFPLE1BQU0sS0FBSyxTQUFTLENBQUMsQ0FBQyxDQUFDLDBCQUEwQixDQUFDLENBQUMsQ0FBQyxNQUFNLENBQUM7S0FDbkU7SUFDRCxPQUFPLEtBQUssSUFBSSxFQUFFO1FBQ2hCLElBQUk7WUFDRixPQUFPLE1BQU0sTUFBTSxFQUFFLENBQUM7U0FDdkI7UUFBQyxPQUFPLENBQUMsRUFBRTtZQUNWLE9BQU8sMEJBQTBCLENBQUM7U0FDbkM7SUFDSCxDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7QUFNRixNQUFNLENBQUMsTUFBTSxxQkFBcUIsR0FBRyxDQUNuQyxVQUFzRCxFQUN0RCxhQUEwRCxFQUM3QyxFQUFFO0lBQ2YsSUFBSSxTQUFvQixDQUFDO0lBQ3pCLElBQUksa0JBQStCLENBQUM7SUFDcEMsT0FBTyxLQUFLLEVBQUUsV0FBVyxFQUFFLE1BQU0sRUFBRSxFQUFFO1FBQ25DLGtCQUFrQixHQUFHLFdBQVcsQ0FBQztRQUNqQyxJQUFJLENBQUMsU0FBUyxFQUFFO1lBQ2QsTUFBTSxFQUFFLE1BQU0sRUFBRSxNQUFNLEVBQUUsR0FBRyxVQUFVLENBQUM7WUFDdEMsU0FBUyxHQUFHLElBQUksYUFBYSxDQUFDO2dCQUM1QixNQUFNO2dCQUVOLHlCQUF5QixFQUFFLEdBQUcsRUFBRSxDQUFDLEtBQUssSUFBSSxFQUFFLENBQUMsa0JBQWtCO2dCQUMvRCxNQUFNLEVBQUUscUJBQXFCLENBQUMsTUFBTSxDQUFDO2FBQ3RDLENBQUMsQ0FBQztTQUNKO1FBQ0QsTUFBTSxFQUFFLFdBQVcsRUFBRSxHQUFHLE1BQU0sU0FBUyxDQUFDLElBQUksQ0FBQyxJQUFJLGlCQUFpQixDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUM7UUFDNUUsSUFBSSxDQUFDLFdBQVcsSUFBSSxDQUFDLFdBQVcsQ0FBQyxXQUFXLElBQUksQ0FBQyxXQUFXLENBQUMsZUFBZSxFQUFFO1lBQzVFLE1BQU0sSUFBSSxLQUFLLENBQUMsdURBQXVELE1BQU0sQ0FBQyxPQUFPLEVBQUUsQ0FBQyxDQUFDO1NBQzFGO1FBQ0QsT0FBTztZQUNMLFdBQVcsRUFBRSxXQUFXLENBQUMsV0FBVztZQUNwQyxlQUFlLEVBQUUsV0FBVyxDQUFDLGVBQWU7WUFDNUMsWUFBWSxFQUFFLFdBQVcsQ0FBQyxZQUFZO1lBQ3RDLFVBQVUsRUFBRSxXQUFXLENBQUMsVUFBVTtTQUNuQyxDQUFDO0lBQ0osQ0FBQyxDQUFDO0FBQ0osQ0FBQyxDQUFDO0FBV0YsTUFBTSxDQUFDLE1BQU0sb0NBQW9DLEdBQUcsQ0FDbEQsVUFBc0QsRUFDdEQsYUFBMEQsRUFDOUIsRUFBRTtJQUM5QixJQUFJLFNBQW9CLENBQUM7SUFDekIsT0FBTyxLQUFLLEVBQUUsTUFBTSxFQUFFLEVBQUU7UUFDdEIsSUFBSSxDQUFDLFNBQVMsRUFBRTtZQUNkLE1BQU0sRUFBRSxNQUFNLEVBQUUsTUFBTSxFQUFFLEdBQUcsVUFBVSxDQUFDO1lBQ3RDLFNBQVMsR0FBRyxJQUFJLGFBQWEsQ0FBQztnQkFDNUIsTUFBTTtnQkFDTixNQUFNLEVBQUUscUJBQXFCLENBQUMsTUFBTSxDQUFDO2FBQ3RDLENBQUMsQ0FBQztTQUNKO1FBQ0QsTUFBTSxFQUFFLFdBQVcsRUFBRSxHQUFHLE1BQU0sU0FBUyxDQUFDLElBQUksQ0FBQyxJQUFJLGdDQUFnQyxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUM7UUFDM0YsSUFBSSxDQUFDLFdBQVcsSUFBSSxDQUFDLFdBQVcsQ0FBQyxXQUFXLElBQUksQ0FBQyxXQUFXLENBQUMsZUFBZSxFQUFFO1lBQzVFLE1BQU0sSUFBSSxLQUFLLENBQUMsc0VBQXNFLE1BQU0sQ0FBQyxPQUFPLEVBQUUsQ0FBQyxDQUFDO1NBQ3pHO1FBQ0QsT0FBTztZQUNMLFdBQVcsRUFBRSxXQUFXLENBQUMsV0FBVztZQUNwQyxlQUFlLEVBQUUsV0FBVyxDQUFDLGVBQWU7WUFDNUMsWUFBWSxFQUFFLFdBQVcsQ0FBQyxZQUFZO1lBQ3RDLFVBQVUsRUFBRSxXQUFXLENBQUMsVUFBVTtTQUNuQyxDQUFDO0lBQ0osQ0FBQyxDQUFDO0FBQ0osQ0FBQyxDQUFDO0FBZUYsTUFBTSxDQUFDLE1BQU0saUNBQWlDLEdBQUcsQ0FBQyxRQUFtQyxFQUE2QixFQUFFLENBQUMsQ0FDbkgsS0FBOEIsRUFDOUIsRUFBRSxDQUNGLFFBQVEsQ0FBQztJQUNQLFdBQVcsRUFBRSxxQkFBcUIsQ0FBQyxLQUFLLEVBQUUsS0FBSyxDQUFDLGFBQWEsQ0FBQztJQUM5RCwwQkFBMEIsRUFBRSxvQ0FBb0MsQ0FBQyxLQUFLLEVBQUUsS0FBSyxDQUFDLGFBQWEsQ0FBQztJQUM1RixHQUFHLEtBQUs7Q0FDVCxDQUFDLENBQUMifQ==