import { CredentialsProviderError } from "../property-provider/mod.ts";
export const fromWebToken = (init) => () => {
    const { roleArn, roleSessionName, webIdentityToken, providerId, policyArns, policy, durationSeconds, roleAssumerWithWebIdentity, } = init;
    if (!roleAssumerWithWebIdentity) {
        throw new CredentialsProviderError(`Role Arn '${roleArn}' needs to be assumed with web identity,` +
            ` but no role assumption callback was provided.`, false);
    }
    return roleAssumerWithWebIdentity({
        RoleArn: roleArn,
        RoleSessionName: roleSessionName ?? `aws-sdk-js-session-${Date.now()}`,
        WebIdentityToken: webIdentityToken,
        ProviderId: providerId,
        PolicyArns: policyArns,
        Policy: policy,
        DurationSeconds: durationSeconds,
    });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnJvbVdlYlRva2VuLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZnJvbVdlYlRva2VuLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSx3QkFBd0IsRUFBRSxNQUFNLDZCQUE2QixDQUFDO0FBaUl2RSxNQUFNLENBQUMsTUFBTSxZQUFZLEdBQ3ZCLENBQUMsSUFBc0IsRUFBc0IsRUFBRSxDQUMvQyxHQUFHLEVBQUU7SUFDSCxNQUFNLEVBQ0osT0FBTyxFQUNQLGVBQWUsRUFDZixnQkFBZ0IsRUFDaEIsVUFBVSxFQUNWLFVBQVUsRUFDVixNQUFNLEVBQ04sZUFBZSxFQUNmLDBCQUEwQixHQUMzQixHQUFHLElBQUksQ0FBQztJQUVULElBQUksQ0FBQywwQkFBMEIsRUFBRTtRQUMvQixNQUFNLElBQUksd0JBQXdCLENBQ2hDLGFBQWEsT0FBTywwQ0FBMEM7WUFDNUQsZ0RBQWdELEVBQ2xELEtBQUssQ0FDTixDQUFDO0tBQ0g7SUFFRCxPQUFPLDBCQUEwQixDQUFDO1FBQ2hDLE9BQU8sRUFBRSxPQUFPO1FBQ2hCLGVBQWUsRUFBRSxlQUFlLElBQUksc0JBQXNCLElBQUksQ0FBQyxHQUFHLEVBQUUsRUFBRTtRQUN0RSxnQkFBZ0IsRUFBRSxnQkFBZ0I7UUFDbEMsVUFBVSxFQUFFLFVBQVU7UUFDdEIsVUFBVSxFQUFFLFVBQVU7UUFDdEIsTUFBTSxFQUFFLE1BQU07UUFDZCxlQUFlLEVBQUUsZUFBZTtLQUNqQyxDQUFDLENBQUM7QUFDTCxDQUFDLENBQUMifQ==