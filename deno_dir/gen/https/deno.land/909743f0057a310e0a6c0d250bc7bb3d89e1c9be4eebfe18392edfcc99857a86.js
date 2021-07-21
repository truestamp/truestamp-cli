import { ProviderError } from "../property-provider/mod.ts";
export const fromWebToken = (init) => () => {
    const { roleArn, roleSessionName, webIdentityToken, providerId, policyArns, policy, durationSeconds, roleAssumerWithWebIdentity, } = init;
    if (!roleAssumerWithWebIdentity) {
        throw new ProviderError(`Role Arn '${roleArn}' needs to be assumed with web identity,` + ` but no role assumption callback was provided.`, false);
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnJvbVdlYlRva2VuLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZnJvbVdlYlRva2VuLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxhQUFhLEVBQUUsTUFBTSw2QkFBNkIsQ0FBQztBQWlJNUQsTUFBTSxDQUFDLE1BQU0sWUFBWSxHQUFHLENBQUMsSUFBc0IsRUFBc0IsRUFBRSxDQUFDLEdBQUcsRUFBRTtJQUMvRSxNQUFNLEVBQ0osT0FBTyxFQUNQLGVBQWUsRUFDZixnQkFBZ0IsRUFDaEIsVUFBVSxFQUNWLFVBQVUsRUFDVixNQUFNLEVBQ04sZUFBZSxFQUNmLDBCQUEwQixHQUMzQixHQUFHLElBQUksQ0FBQztJQUVULElBQUksQ0FBQywwQkFBMEIsRUFBRTtRQUMvQixNQUFNLElBQUksYUFBYSxDQUNyQixhQUFhLE9BQU8sMENBQTBDLEdBQUcsZ0RBQWdELEVBQ2pILEtBQUssQ0FDTixDQUFDO0tBQ0g7SUFFRCxPQUFPLDBCQUEwQixDQUFDO1FBQ2hDLE9BQU8sRUFBRSxPQUFPO1FBQ2hCLGVBQWUsRUFBRSxlQUFlLElBQUksc0JBQXNCLElBQUksQ0FBQyxHQUFHLEVBQUUsRUFBRTtRQUN0RSxnQkFBZ0IsRUFBRSxnQkFBZ0I7UUFDbEMsVUFBVSxFQUFFLFVBQVU7UUFDdEIsVUFBVSxFQUFFLFVBQVU7UUFDdEIsTUFBTSxFQUFFLE1BQU07UUFDZCxlQUFlLEVBQUUsZUFBZTtLQUNqQyxDQUFDLENBQUM7QUFDTCxDQUFDLENBQUMifQ==