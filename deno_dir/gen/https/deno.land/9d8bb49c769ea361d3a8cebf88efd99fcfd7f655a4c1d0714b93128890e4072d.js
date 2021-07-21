import process from "https://deno.land/std@0.101.0/node/process.ts";
import { CredentialsProviderError } from "../property-provider/mod.ts";
import { loadSharedConfigFiles, } from "../shared-ini-file-loader/mod.ts";
const DEFAULT_PROFILE = "default";
export const ENV_PROFILE = "AWS_PROFILE";
export const fromSharedConfigFiles = (configSelector, { preferredFile = "config", ...init } = {}) => async () => {
    const { loadedConfig = loadSharedConfigFiles(init), profile = process.env[ENV_PROFILE] || DEFAULT_PROFILE } = init;
    const { configFile, credentialsFile } = await loadedConfig;
    const profileFromCredentials = credentialsFile[profile] || {};
    const profileFromConfig = configFile[profile] || {};
    const mergedProfile = preferredFile === "config"
        ? { ...profileFromCredentials, ...profileFromConfig }
        : { ...profileFromConfig, ...profileFromCredentials };
    try {
        const configValue = configSelector(mergedProfile);
        if (configValue === undefined) {
            throw new Error();
        }
        return configValue;
    }
    catch (e) {
        throw new CredentialsProviderError(e.message ||
            `Cannot load config for profile ${profile} in SDK configuration files with getter: ${configSelector}`);
    }
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnJvbVNoYXJlZENvbmZpZ0ZpbGVzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZnJvbVNoYXJlZENvbmZpZ0ZpbGVzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sT0FBTyxNQUFNLCtDQUErQyxDQUFDO0FBQ3BFLE9BQU8sRUFBRSx3QkFBd0IsRUFBRSxNQUFNLDZCQUE2QixDQUFDO0FBQ3ZFLE9BQU8sRUFDTCxxQkFBcUIsR0FJdEIsTUFBTSxrQ0FBa0MsQ0FBQztBQUcxQyxNQUFNLGVBQWUsR0FBRyxTQUFTLENBQUM7QUFDbEMsTUFBTSxDQUFDLE1BQU0sV0FBVyxHQUFHLGFBQWEsQ0FBQztBQTZCekMsTUFBTSxDQUFDLE1BQU0scUJBQXFCLEdBQ2hDLENBQ0UsY0FBbUMsRUFDbkMsRUFBRSxhQUFhLEdBQUcsUUFBUSxFQUFFLEdBQUcsSUFBSSxLQUF1QixFQUFFLEVBQy9DLEVBQUUsQ0FDakIsS0FBSyxJQUFJLEVBQUU7SUFDVCxNQUFNLEVBQUUsWUFBWSxHQUFHLHFCQUFxQixDQUFDLElBQUksQ0FBQyxFQUFFLE9BQU8sR0FBRyxPQUFPLENBQUMsR0FBRyxDQUFDLFdBQVcsQ0FBQyxJQUFJLGVBQWUsRUFBRSxHQUFHLElBQUksQ0FBQztJQUVuSCxNQUFNLEVBQUUsVUFBVSxFQUFFLGVBQWUsRUFBRSxHQUFHLE1BQU0sWUFBWSxDQUFDO0lBRTNELE1BQU0sc0JBQXNCLEdBQUcsZUFBZSxDQUFDLE9BQU8sQ0FBQyxJQUFJLEVBQUUsQ0FBQztJQUM5RCxNQUFNLGlCQUFpQixHQUFHLFVBQVUsQ0FBQyxPQUFPLENBQUMsSUFBSSxFQUFFLENBQUM7SUFDcEQsTUFBTSxhQUFhLEdBQ2pCLGFBQWEsS0FBSyxRQUFRO1FBQ3hCLENBQUMsQ0FBQyxFQUFFLEdBQUcsc0JBQXNCLEVBQUUsR0FBRyxpQkFBaUIsRUFBRTtRQUNyRCxDQUFDLENBQUMsRUFBRSxHQUFHLGlCQUFpQixFQUFFLEdBQUcsc0JBQXNCLEVBQUUsQ0FBQztJQUUxRCxJQUFJO1FBQ0YsTUFBTSxXQUFXLEdBQUcsY0FBYyxDQUFDLGFBQWEsQ0FBQyxDQUFDO1FBQ2xELElBQUksV0FBVyxLQUFLLFNBQVMsRUFBRTtZQUM3QixNQUFNLElBQUksS0FBSyxFQUFFLENBQUM7U0FDbkI7UUFDRCxPQUFPLFdBQVcsQ0FBQztLQUNwQjtJQUFDLE9BQU8sQ0FBQyxFQUFFO1FBQ1YsTUFBTSxJQUFJLHdCQUF3QixDQUNoQyxDQUFDLENBQUMsT0FBTztZQUNQLGtDQUFrQyxPQUFPLDRDQUE0QyxjQUFjLEVBQUUsQ0FDeEcsQ0FBQztLQUNIO0FBQ0gsQ0FBQyxDQUFDIn0=