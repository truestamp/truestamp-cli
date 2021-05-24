import process from "https://deno.land/std@0.89.0/node/process.ts";
import { fromEnv } from "../credential-provider-env/mod.ts";
import { ENV_CMDS_FULL_URI, ENV_CMDS_RELATIVE_URI, fromContainerMetadata, fromInstanceMetadata, } from "../credential-provider-imds/mod.ts";
import { ENV_PROFILE, fromIni } from "../credential-provider-ini/mod.ts";
import { fromProcess } from "../credential-provider-process/mod.ts";
import { fromSSO } from "../credential-provider-sso/mod.ts";
import { fromTokenFile } from "../credential-provider-web-identity/mod.ts";
import { chain, memoize, ProviderError } from "../property-provider/mod.ts";
import { loadSharedConfigFiles } from "../shared-ini-file-loader/mod.ts";
export const ENV_IMDS_DISABLED = "AWS_EC2_METADATA_DISABLED";
export const defaultProvider = (init = {}) => {
    const options = { profile: process.env[ENV_PROFILE], ...init };
    if (!options.loadedConfig)
        options.loadedConfig = loadSharedConfigFiles(init);
    const providers = [
        fromSSO(options),
        fromIni(options),
        fromProcess(options),
        fromTokenFile(options),
        remoteProvider(options),
    ];
    if (!options.profile)
        providers.unshift(fromEnv());
    const providerChain = chain(...providers);
    return memoize(providerChain, (credentials) => credentials.expiration !== undefined && credentials.expiration.getTime() - Date.now() < 300000, (credentials) => credentials.expiration !== undefined);
};
const remoteProvider = (init) => {
    if (process.env[ENV_CMDS_RELATIVE_URI] || process.env[ENV_CMDS_FULL_URI]) {
        return fromContainerMetadata(init);
    }
    if (process.env[ENV_IMDS_DISABLED]) {
        return () => Promise.reject(new ProviderError("EC2 Instance Metadata Service access disabled"));
    }
    return fromInstanceMetadata(init);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sT0FBTyxNQUFNLDhDQUE4QyxDQUFDO0FBQ25FLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxtQ0FBbUMsQ0FBQztBQUM1RCxPQUFPLEVBQ0wsaUJBQWlCLEVBQ2pCLHFCQUFxQixFQUNyQixxQkFBcUIsRUFDckIsb0JBQW9CLEdBRXJCLE1BQU0sb0NBQW9DLENBQUM7QUFDNUMsT0FBTyxFQUFFLFdBQVcsRUFBRSxPQUFPLEVBQWUsTUFBTSxtQ0FBbUMsQ0FBQztBQUN0RixPQUFPLEVBQUUsV0FBVyxFQUFtQixNQUFNLHVDQUF1QyxDQUFDO0FBQ3JGLE9BQU8sRUFBRSxPQUFPLEVBQWUsTUFBTSxtQ0FBbUMsQ0FBQztBQUN6RSxPQUFPLEVBQUUsYUFBYSxFQUFxQixNQUFNLDRDQUE0QyxDQUFDO0FBQzlGLE9BQU8sRUFBRSxLQUFLLEVBQUUsT0FBTyxFQUFFLGFBQWEsRUFBRSxNQUFNLDZCQUE2QixDQUFDO0FBQzVFLE9BQU8sRUFBRSxxQkFBcUIsRUFBRSxNQUFNLGtDQUFrQyxDQUFDO0FBR3pFLE1BQU0sQ0FBQyxNQUFNLGlCQUFpQixHQUFHLDJCQUEyQixDQUFDO0FBb0M3RCxNQUFNLENBQUMsTUFBTSxlQUFlLEdBQUcsQ0FDN0IsT0FBNkYsRUFBRSxFQUMzRSxFQUFFO0lBQ3RCLE1BQU0sT0FBTyxHQUFHLEVBQUUsT0FBTyxFQUFFLE9BQU8sQ0FBQyxHQUFHLENBQUMsV0FBVyxDQUFDLEVBQUUsR0FBRyxJQUFJLEVBQUUsQ0FBQztJQUMvRCxJQUFJLENBQUMsT0FBTyxDQUFDLFlBQVk7UUFBRSxPQUFPLENBQUMsWUFBWSxHQUFHLHFCQUFxQixDQUFDLElBQUksQ0FBQyxDQUFDO0lBQzlFLE1BQU0sU0FBUyxHQUFHO1FBQ2hCLE9BQU8sQ0FBQyxPQUFPLENBQUM7UUFDaEIsT0FBTyxDQUFDLE9BQU8sQ0FBQztRQUNoQixXQUFXLENBQUMsT0FBTyxDQUFDO1FBQ3BCLGFBQWEsQ0FBQyxPQUFPLENBQUM7UUFDdEIsY0FBYyxDQUFDLE9BQU8sQ0FBQztLQUN4QixDQUFDO0lBQ0YsSUFBSSxDQUFDLE9BQU8sQ0FBQyxPQUFPO1FBQUUsU0FBUyxDQUFDLE9BQU8sQ0FBQyxPQUFPLEVBQUUsQ0FBQyxDQUFDO0lBQ25ELE1BQU0sYUFBYSxHQUFHLEtBQUssQ0FBQyxHQUFHLFNBQVMsQ0FBQyxDQUFDO0lBRTFDLE9BQU8sT0FBTyxDQUNaLGFBQWEsRUFDYixDQUFDLFdBQVcsRUFBRSxFQUFFLENBQUMsV0FBVyxDQUFDLFVBQVUsS0FBSyxTQUFTLElBQUksV0FBVyxDQUFDLFVBQVUsQ0FBQyxPQUFPLEVBQUUsR0FBRyxJQUFJLENBQUMsR0FBRyxFQUFFLEdBQUcsTUFBTSxFQUMvRyxDQUFDLFdBQVcsRUFBRSxFQUFFLENBQUMsV0FBVyxDQUFDLFVBQVUsS0FBSyxTQUFTLENBQ3RELENBQUM7QUFDSixDQUFDLENBQUM7QUFFRixNQUFNLGNBQWMsR0FBRyxDQUFDLElBQXdCLEVBQXNCLEVBQUU7SUFDdEUsSUFBSSxPQUFPLENBQUMsR0FBRyxDQUFDLHFCQUFxQixDQUFDLElBQUksT0FBTyxDQUFDLEdBQUcsQ0FBQyxpQkFBaUIsQ0FBQyxFQUFFO1FBQ3hFLE9BQU8scUJBQXFCLENBQUMsSUFBSSxDQUFDLENBQUM7S0FDcEM7SUFFRCxJQUFJLE9BQU8sQ0FBQyxHQUFHLENBQUMsaUJBQWlCLENBQUMsRUFBRTtRQUNsQyxPQUFPLEdBQUcsRUFBRSxDQUFDLE9BQU8sQ0FBQyxNQUFNLENBQUMsSUFBSSxhQUFhLENBQUMsK0NBQStDLENBQUMsQ0FBQyxDQUFDO0tBQ2pHO0lBRUQsT0FBTyxvQkFBb0IsQ0FBQyxJQUFJLENBQUMsQ0FBQztBQUNwQyxDQUFDLENBQUMifQ==