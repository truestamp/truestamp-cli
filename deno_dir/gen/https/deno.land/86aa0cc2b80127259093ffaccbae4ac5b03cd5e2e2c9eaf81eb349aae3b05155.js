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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29uZmlndXJhdGlvbnMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJjb25maWd1cmF0aW9ucy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsT0FBTyxFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFDdEQsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLHdCQUF3QixDQUFDO0FBSXJELE1BQU0sd0JBQXdCLEdBQUcsTUFBTSxDQUFDO0FBd0R4QyxNQUFNLENBQUMsTUFBTSxvQkFBb0IsR0FBRyxDQUNsQyxLQUFrRCxFQUN2QixFQUFFO0lBQzdCLE1BQU0sZUFBZSxHQUFHLEtBQUssQ0FBQyxXQUFXO1FBQ3ZDLENBQUMsQ0FBQywyQkFBMkIsQ0FBQyxLQUFLLENBQUMsV0FBVyxDQUFDO1FBQ2hELENBQUMsQ0FBQyxLQUFLLENBQUMseUJBQXlCLENBQUMsS0FBWSxDQUFDLENBQUM7SUFDbEQsTUFBTSxFQUFFLGlCQUFpQixHQUFHLElBQUksRUFBRSxpQkFBaUIsR0FBRyxLQUFLLENBQUMsaUJBQWlCLElBQUksQ0FBQyxFQUFFLE1BQU0sRUFBRSxHQUFHLEtBQUssQ0FBQztJQUNyRyxJQUFJLE1BQStCLENBQUM7SUFDcEMsSUFBSSxLQUFLLENBQUMsTUFBTSxFQUFFO1FBRWhCLE1BQU0sR0FBRyxpQkFBaUIsQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDLENBQUM7S0FDMUM7U0FBTTtRQUVMLE1BQU0sR0FBRyxHQUFHLEVBQUUsQ0FDWixpQkFBaUIsQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDLEVBQUU7YUFDOUIsSUFBSSxDQUFDLEtBQUssRUFBRSxNQUFNLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQyxNQUFNLEtBQUssQ0FBQyxrQkFBa0IsQ0FBQyxNQUFNLENBQUMsQ0FBQyxJQUFJLEVBQUUsRUFBRSxNQUFNLENBQXlCLENBQUM7YUFDeEcsSUFBSSxDQUFDLENBQUMsQ0FBQyxVQUFVLEVBQUUsTUFBTSxDQUFDLEVBQUUsRUFBRTtZQUM3QixNQUFNLEVBQUUsYUFBYSxFQUFFLGNBQWMsRUFBRSxHQUFHLFVBQVUsQ0FBQztZQUdyRCxLQUFLLENBQUMsYUFBYSxHQUFHLEtBQUssQ0FBQyxhQUFhLElBQUksYUFBYSxJQUFJLE1BQU0sQ0FBQztZQUdyRSxLQUFLLENBQUMsV0FBVyxHQUFHLEtBQUssQ0FBQyxXQUFXLElBQUksY0FBYyxJQUFJLEtBQUssQ0FBQyxTQUFTLENBQUM7WUFFM0UsT0FBTyxJQUFJLFdBQVcsQ0FBQztnQkFDckIsV0FBVyxFQUFFLGVBQWU7Z0JBQzVCLE1BQU0sRUFBRSxLQUFLLENBQUMsYUFBYTtnQkFDM0IsT0FBTyxFQUFFLEtBQUssQ0FBQyxXQUFXO2dCQUMxQixNQUFNO2dCQUNOLGFBQWEsRUFBRSxpQkFBaUI7YUFDakMsQ0FBQyxDQUFDO1FBQ0wsQ0FBQyxDQUFDLENBQUM7S0FDUjtJQUVELE9BQU87UUFDTCxHQUFHLEtBQUs7UUFDUixpQkFBaUI7UUFDakIsaUJBQWlCO1FBQ2pCLFdBQVcsRUFBRSxlQUFlO1FBQzVCLE1BQU07S0FDUCxDQUFDO0FBQ0osQ0FBQyxDQUFDO0FBRUYsTUFBTSxpQkFBaUIsR0FBRyxDQUFJLEtBQXNCLEVBQWUsRUFBRTtJQUNuRSxJQUFJLE9BQU8sS0FBSyxLQUFLLFFBQVEsRUFBRTtRQUM3QixNQUFNLFdBQVcsR0FBRyxPQUFPLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxDQUFDO1FBQzNDLE9BQU8sR0FBRyxFQUFFLENBQUMsV0FBVyxDQUFDO0tBQzFCO0lBQ0QsT0FBTyxLQUFvQixDQUFDO0FBQzlCLENBQUMsQ0FBQztBQUVGLE1BQU0sMkJBQTJCLEdBQUcsQ0FBQyxXQUFnRCxFQUF5QixFQUFFO0lBQzlHLElBQUksT0FBTyxXQUFXLEtBQUssVUFBVSxFQUFFO1FBQ3JDLE9BQU8sT0FBTyxDQUNaLFdBQVcsRUFDWCxDQUFDLFdBQVcsRUFBRSxFQUFFLENBQ2QsV0FBVyxDQUFDLFVBQVUsS0FBSyxTQUFTO1lBQ3BDLFdBQVcsQ0FBQyxVQUFVLENBQUMsT0FBTyxFQUFFLEdBQUcsSUFBSSxDQUFDLEdBQUcsRUFBRSxHQUFHLHdCQUF3QixFQUMxRSxDQUFDLFdBQVcsRUFBRSxFQUFFLENBQUMsV0FBVyxDQUFDLFVBQVUsS0FBSyxTQUFTLENBQ3RELENBQUM7S0FDSDtJQUNELE9BQU8saUJBQWlCLENBQUMsV0FBVyxDQUFDLENBQUM7QUFDeEMsQ0FBQyxDQUFDIn0=