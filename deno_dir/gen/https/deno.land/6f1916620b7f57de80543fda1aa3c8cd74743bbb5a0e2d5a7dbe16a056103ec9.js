import process from "https://deno.land/std@0.101.0/node/process.ts";
import { fromEnv } from "../credential-provider-env/mod.ts";
import { ENV_CMDS_FULL_URI, ENV_CMDS_RELATIVE_URI, fromContainerMetadata, fromInstanceMetadata, } from "../credential-provider-imds/mod.ts";
import { fromIni } from "../credential-provider-ini/mod.ts";
import { fromProcess } from "../credential-provider-process/mod.ts";
import { fromSSO } from "../credential-provider-sso/mod.ts";
import { fromTokenFile } from "../credential-provider-web-identity/mod.ts";
import { chain, CredentialsProviderError, memoize } from "../property-provider/mod.ts";
import { loadSharedConfigFiles } from "../shared-ini-file-loader/mod.ts";
import { ENV_PROFILE } from "../util-credentials/mod.ts";
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
        async () => {
            throw new CredentialsProviderError("Could not load credentials from any providers", false);
        },
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
        return () => Promise.reject(new CredentialsProviderError("EC2 Instance Metadata Service access disabled"));
    }
    return fromInstanceMetadata(init);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sT0FBTyxNQUFNLCtDQUErQyxDQUFDO0FBQ3BFLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxtQ0FBbUMsQ0FBQztBQUM1RCxPQUFPLEVBQ0wsaUJBQWlCLEVBQ2pCLHFCQUFxQixFQUNyQixxQkFBcUIsRUFDckIsb0JBQW9CLEdBRXJCLE1BQU0sb0NBQW9DLENBQUM7QUFDNUMsT0FBTyxFQUFFLE9BQU8sRUFBZSxNQUFNLG1DQUFtQyxDQUFDO0FBQ3pFLE9BQU8sRUFBRSxXQUFXLEVBQW1CLE1BQU0sdUNBQXVDLENBQUM7QUFDckYsT0FBTyxFQUFFLE9BQU8sRUFBZSxNQUFNLG1DQUFtQyxDQUFDO0FBQ3pFLE9BQU8sRUFBRSxhQUFhLEVBQXFCLE1BQU0sNENBQTRDLENBQUM7QUFDOUYsT0FBTyxFQUFFLEtBQUssRUFBRSx3QkFBd0IsRUFBRSxPQUFPLEVBQUUsTUFBTSw2QkFBNkIsQ0FBQztBQUN2RixPQUFPLEVBQUUscUJBQXFCLEVBQUUsTUFBTSxrQ0FBa0MsQ0FBQztBQUV6RSxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0sNEJBQTRCLENBQUM7QUFFekQsTUFBTSxDQUFDLE1BQU0saUJBQWlCLEdBQUcsMkJBQTJCLENBQUM7QUFvQzdELE1BQU0sQ0FBQyxNQUFNLGVBQWUsR0FBRyxDQUM3QixPQUE2RixFQUFFLEVBQzNFLEVBQUU7SUFDdEIsTUFBTSxPQUFPLEdBQUcsRUFBRSxPQUFPLEVBQUUsT0FBTyxDQUFDLEdBQUcsQ0FBQyxXQUFXLENBQUMsRUFBRSxHQUFHLElBQUksRUFBRSxDQUFDO0lBQy9ELElBQUksQ0FBQyxPQUFPLENBQUMsWUFBWTtRQUFFLE9BQU8sQ0FBQyxZQUFZLEdBQUcscUJBQXFCLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDOUUsTUFBTSxTQUFTLEdBQUc7UUFDaEIsT0FBTyxDQUFDLE9BQU8sQ0FBQztRQUNoQixPQUFPLENBQUMsT0FBTyxDQUFDO1FBQ2hCLFdBQVcsQ0FBQyxPQUFPLENBQUM7UUFDcEIsYUFBYSxDQUFDLE9BQU8sQ0FBQztRQUN0QixjQUFjLENBQUMsT0FBTyxDQUFDO1FBQ3ZCLEtBQUssSUFBSSxFQUFFO1lBQ1QsTUFBTSxJQUFJLHdCQUF3QixDQUFDLCtDQUErQyxFQUFFLEtBQUssQ0FBQyxDQUFDO1FBQzdGLENBQUM7S0FDRixDQUFDO0lBQ0YsSUFBSSxDQUFDLE9BQU8sQ0FBQyxPQUFPO1FBQUUsU0FBUyxDQUFDLE9BQU8sQ0FBQyxPQUFPLEVBQUUsQ0FBQyxDQUFDO0lBQ25ELE1BQU0sYUFBYSxHQUFHLEtBQUssQ0FBQyxHQUFHLFNBQVMsQ0FBQyxDQUFDO0lBRTFDLE9BQU8sT0FBTyxDQUNaLGFBQWEsRUFDYixDQUFDLFdBQVcsRUFBRSxFQUFFLENBQUMsV0FBVyxDQUFDLFVBQVUsS0FBSyxTQUFTLElBQUksV0FBVyxDQUFDLFVBQVUsQ0FBQyxPQUFPLEVBQUUsR0FBRyxJQUFJLENBQUMsR0FBRyxFQUFFLEdBQUcsTUFBTSxFQUMvRyxDQUFDLFdBQVcsRUFBRSxFQUFFLENBQUMsV0FBVyxDQUFDLFVBQVUsS0FBSyxTQUFTLENBQ3RELENBQUM7QUFDSixDQUFDLENBQUM7QUFFRixNQUFNLGNBQWMsR0FBRyxDQUFDLElBQXdCLEVBQXNCLEVBQUU7SUFDdEUsSUFBSSxPQUFPLENBQUMsR0FBRyxDQUFDLHFCQUFxQixDQUFDLElBQUksT0FBTyxDQUFDLEdBQUcsQ0FBQyxpQkFBaUIsQ0FBQyxFQUFFO1FBQ3hFLE9BQU8scUJBQXFCLENBQUMsSUFBSSxDQUFDLENBQUM7S0FDcEM7SUFFRCxJQUFJLE9BQU8sQ0FBQyxHQUFHLENBQUMsaUJBQWlCLENBQUMsRUFBRTtRQUNsQyxPQUFPLEdBQUcsRUFBRSxDQUFDLE9BQU8sQ0FBQyxNQUFNLENBQUMsSUFBSSx3QkFBd0IsQ0FBQywrQ0FBK0MsQ0FBQyxDQUFDLENBQUM7S0FDNUc7SUFFRCxPQUFPLG9CQUFvQixDQUFDLElBQUksQ0FBQyxDQUFDO0FBQ3BDLENBQUMsQ0FBQyJ9