import { memoize } from "../property-provider/mod.ts";
import { SignatureV4 } from "../signature-v4/mod.ts";
const CREDENTIAL_EXPIRE_WINDOW = 300000;
export const resolveAwsAuthConfig = (input) => {
    const normalizedCreds = input.credentials
        ? normalizeCredentialProvider(input.credentials)
        : input.credentialDefaultProvider(input);
    const { signingEscapePath = true, systemClockOffset = input.systemClockOffset || 0, sha256 } = input;
    let signer;
    if (input.signer) {
        signer = normalizeProvider(input.signer);
    }
    else {
        signer = () => normalizeProvider(input.region)()
            .then(async (region) => [(await input.regionInfoProvider(region)) || {}, region])
            .then(([regionInfo, region]) => {
            const { signingRegion, signingService } = regionInfo;
            input.signingRegion = input.signingRegion || signingRegion || region;
            input.signingName = input.signingName || signingService || input.serviceId;
            return new SignatureV4({
                credentials: normalizedCreds,
                region: input.signingRegion,
                service: input.signingName,
                sha256,
                uriEscapePath: signingEscapePath,
            });
        });
    }
    return {
        ...input,
        systemClockOffset,
        signingEscapePath,
        credentials: normalizedCreds,
        signer,
    };
};
export const resolveSigV4AuthConfig = (input) => {
    const normalizedCreds = input.credentials
        ? normalizeCredentialProvider(input.credentials)
        : input.credentialDefaultProvider(input);
    const { signingEscapePath = true, systemClockOffset = input.systemClockOffset || 0, sha256 } = input;
    let signer;
    if (input.signer) {
        signer = normalizeProvider(input.signer);
    }
    else {
        signer = normalizeProvider(new SignatureV4({
            credentials: normalizedCreds,
            region: input.region,
            service: input.signingName,
            sha256,
            uriEscapePath: signingEscapePath,
        }));
    }
    return {
        ...input,
        systemClockOffset,
        signingEscapePath,
        credentials: normalizedCreds,
        signer,
    };
};
const normalizeProvider = (input) => {
    if (typeof input === "object") {
        const promisified = Promise.resolve(input);
        return () => promisified;
    }
    return input;
};
const normalizeCredentialProvider = (credentials) => {
    if (typeof credentials === "function") {
        return memoize(credentials, (credentials) => credentials.expiration !== undefined &&
            credentials.expiration.getTime() - Date.now() < CREDENTIAL_EXPIRE_WINDOW, (credentials) => credentials.expiration !== undefined);
    }
    return normalizeProvider(credentials);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29uZmlndXJhdGlvbnMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJjb25maWd1cmF0aW9ucy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsT0FBTyxFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFDdEQsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLHdCQUF3QixDQUFDO0FBSXJELE1BQU0sd0JBQXdCLEdBQUcsTUFBTSxDQUFDO0FBNkZ4QyxNQUFNLENBQUMsTUFBTSxvQkFBb0IsR0FBRyxDQUNsQyxLQUFrRCxFQUN2QixFQUFFO0lBQzdCLE1BQU0sZUFBZSxHQUFHLEtBQUssQ0FBQyxXQUFXO1FBQ3ZDLENBQUMsQ0FBQywyQkFBMkIsQ0FBQyxLQUFLLENBQUMsV0FBVyxDQUFDO1FBQ2hELENBQUMsQ0FBQyxLQUFLLENBQUMseUJBQXlCLENBQUMsS0FBWSxDQUFDLENBQUM7SUFDbEQsTUFBTSxFQUFFLGlCQUFpQixHQUFHLElBQUksRUFBRSxpQkFBaUIsR0FBRyxLQUFLLENBQUMsaUJBQWlCLElBQUksQ0FBQyxFQUFFLE1BQU0sRUFBRSxHQUFHLEtBQUssQ0FBQztJQUNyRyxJQUFJLE1BQStCLENBQUM7SUFDcEMsSUFBSSxLQUFLLENBQUMsTUFBTSxFQUFFO1FBRWhCLE1BQU0sR0FBRyxpQkFBaUIsQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDLENBQUM7S0FDMUM7U0FBTTtRQUVMLE1BQU0sR0FBRyxHQUFHLEVBQUUsQ0FDWixpQkFBaUIsQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDLEVBQUU7YUFDOUIsSUFBSSxDQUFDLEtBQUssRUFBRSxNQUFNLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQyxNQUFNLEtBQUssQ0FBQyxrQkFBa0IsQ0FBQyxNQUFNLENBQUMsQ0FBQyxJQUFJLEVBQUUsRUFBRSxNQUFNLENBQXlCLENBQUM7YUFDeEcsSUFBSSxDQUFDLENBQUMsQ0FBQyxVQUFVLEVBQUUsTUFBTSxDQUFDLEVBQUUsRUFBRTtZQUM3QixNQUFNLEVBQUUsYUFBYSxFQUFFLGNBQWMsRUFBRSxHQUFHLFVBQVUsQ0FBQztZQUdyRCxLQUFLLENBQUMsYUFBYSxHQUFHLEtBQUssQ0FBQyxhQUFhLElBQUksYUFBYSxJQUFJLE1BQU0sQ0FBQztZQUdyRSxLQUFLLENBQUMsV0FBVyxHQUFHLEtBQUssQ0FBQyxXQUFXLElBQUksY0FBYyxJQUFJLEtBQUssQ0FBQyxTQUFTLENBQUM7WUFFM0UsT0FBTyxJQUFJLFdBQVcsQ0FBQztnQkFDckIsV0FBVyxFQUFFLGVBQWU7Z0JBQzVCLE1BQU0sRUFBRSxLQUFLLENBQUMsYUFBYTtnQkFDM0IsT0FBTyxFQUFFLEtBQUssQ0FBQyxXQUFXO2dCQUMxQixNQUFNO2dCQUNOLGFBQWEsRUFBRSxpQkFBaUI7YUFDakMsQ0FBQyxDQUFDO1FBQ0wsQ0FBQyxDQUFDLENBQUM7S0FDUjtJQUVELE9BQU87UUFDTCxHQUFHLEtBQUs7UUFDUixpQkFBaUI7UUFDakIsaUJBQWlCO1FBQ2pCLFdBQVcsRUFBRSxlQUFlO1FBQzVCLE1BQU07S0FDUCxDQUFDO0FBQ0osQ0FBQyxDQUFDO0FBR0YsTUFBTSxDQUFDLE1BQU0sc0JBQXNCLEdBQUcsQ0FDcEMsS0FBeUQsRUFDNUIsRUFBRTtJQUMvQixNQUFNLGVBQWUsR0FBRyxLQUFLLENBQUMsV0FBVztRQUN2QyxDQUFDLENBQUMsMkJBQTJCLENBQUMsS0FBSyxDQUFDLFdBQVcsQ0FBQztRQUNoRCxDQUFDLENBQUMsS0FBSyxDQUFDLHlCQUF5QixDQUFDLEtBQVksQ0FBQyxDQUFDO0lBQ2xELE1BQU0sRUFBRSxpQkFBaUIsR0FBRyxJQUFJLEVBQUUsaUJBQWlCLEdBQUcsS0FBSyxDQUFDLGlCQUFpQixJQUFJLENBQUMsRUFBRSxNQUFNLEVBQUUsR0FBRyxLQUFLLENBQUM7SUFDckcsSUFBSSxNQUErQixDQUFDO0lBQ3BDLElBQUksS0FBSyxDQUFDLE1BQU0sRUFBRTtRQUVoQixNQUFNLEdBQUcsaUJBQWlCLENBQUMsS0FBSyxDQUFDLE1BQU0sQ0FBQyxDQUFDO0tBQzFDO1NBQU07UUFDTCxNQUFNLEdBQUcsaUJBQWlCLENBQUMsSUFBSSxXQUFXLENBQUM7WUFDekMsV0FBVyxFQUFFLGVBQWU7WUFDNUIsTUFBTSxFQUFFLEtBQUssQ0FBQyxNQUFNO1lBQ3BCLE9BQU8sRUFBRSxLQUFLLENBQUMsV0FBVztZQUMxQixNQUFNO1lBQ04sYUFBYSxFQUFFLGlCQUFpQjtTQUNqQyxDQUFDLENBQUMsQ0FBQztLQUNMO0lBRUQsT0FBTztRQUNMLEdBQUcsS0FBSztRQUNSLGlCQUFpQjtRQUNqQixpQkFBaUI7UUFDakIsV0FBVyxFQUFFLGVBQWU7UUFDNUIsTUFBTTtLQUNQLENBQUM7QUFDSixDQUFDLENBQUM7QUFFRixNQUFNLGlCQUFpQixHQUFHLENBQUksS0FBc0IsRUFBZSxFQUFFO0lBQ25FLElBQUksT0FBTyxLQUFLLEtBQUssUUFBUSxFQUFFO1FBQzdCLE1BQU0sV0FBVyxHQUFHLE9BQU8sQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLENBQUM7UUFDM0MsT0FBTyxHQUFHLEVBQUUsQ0FBQyxXQUFXLENBQUM7S0FDMUI7SUFDRCxPQUFPLEtBQW9CLENBQUM7QUFDOUIsQ0FBQyxDQUFDO0FBRUYsTUFBTSwyQkFBMkIsR0FBRyxDQUFDLFdBQWdELEVBQXlCLEVBQUU7SUFDOUcsSUFBSSxPQUFPLFdBQVcsS0FBSyxVQUFVLEVBQUU7UUFDckMsT0FBTyxPQUFPLENBQ1osV0FBVyxFQUNYLENBQUMsV0FBVyxFQUFFLEVBQUUsQ0FDZCxXQUFXLENBQUMsVUFBVSxLQUFLLFNBQVM7WUFDcEMsV0FBVyxDQUFDLFVBQVUsQ0FBQyxPQUFPLEVBQUUsR0FBRyxJQUFJLENBQUMsR0FBRyxFQUFFLEdBQUcsd0JBQXdCLEVBQzFFLENBQUMsV0FBVyxFQUFFLEVBQUUsQ0FBQyxXQUFXLENBQUMsVUFBVSxLQUFLLFNBQVMsQ0FDdEQsQ0FBQztLQUNIO0lBQ0QsT0FBTyxpQkFBaUIsQ0FBQyxXQUFXLENBQUMsQ0FBQztBQUN4QyxDQUFDLENBQUMifQ==