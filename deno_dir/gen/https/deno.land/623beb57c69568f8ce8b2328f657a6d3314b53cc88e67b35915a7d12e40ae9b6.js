import process from "https://deno.land/std@0.89.0/node/process.ts";
import { ProviderError } from "../property-provider/mod.ts";
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
        throw new ProviderError(e.message || `Cannot load config for profile ${profile} in SDK configuration files with getter: ${configSelector}`);
    }
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnJvbVNoYXJlZENvbmZpZ0ZpbGVzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZnJvbVNoYXJlZENvbmZpZ0ZpbGVzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sT0FBTyxNQUFNLDhDQUE4QyxDQUFDO0FBQ25FLE9BQU8sRUFBRSxhQUFhLEVBQUUsTUFBTSw2QkFBNkIsQ0FBQztBQUM1RCxPQUFPLEVBQ0wscUJBQXFCLEdBSXRCLE1BQU0sa0NBQWtDLENBQUM7QUFHMUMsTUFBTSxlQUFlLEdBQUcsU0FBUyxDQUFDO0FBQ2xDLE1BQU0sQ0FBQyxNQUFNLFdBQVcsR0FBRyxhQUFhLENBQUM7QUE2QnpDLE1BQU0sQ0FBQyxNQUFNLHFCQUFxQixHQUFHLENBQ25DLGNBQW1DLEVBQ25DLEVBQUUsYUFBYSxHQUFHLFFBQVEsRUFBRSxHQUFHLElBQUksS0FBdUIsRUFBRSxFQUMvQyxFQUFFLENBQUMsS0FBSyxJQUFJLEVBQUU7SUFDM0IsTUFBTSxFQUFFLFlBQVksR0FBRyxxQkFBcUIsQ0FBQyxJQUFJLENBQUMsRUFBRSxPQUFPLEdBQUcsT0FBTyxDQUFDLEdBQUcsQ0FBQyxXQUFXLENBQUMsSUFBSSxlQUFlLEVBQUUsR0FBRyxJQUFJLENBQUM7SUFFbkgsTUFBTSxFQUFFLFVBQVUsRUFBRSxlQUFlLEVBQUUsR0FBRyxNQUFNLFlBQVksQ0FBQztJQUUzRCxNQUFNLHNCQUFzQixHQUFHLGVBQWUsQ0FBQyxPQUFPLENBQUMsSUFBSSxFQUFFLENBQUM7SUFDOUQsTUFBTSxpQkFBaUIsR0FBRyxVQUFVLENBQUMsT0FBTyxDQUFDLElBQUksRUFBRSxDQUFDO0lBQ3BELE1BQU0sYUFBYSxHQUNqQixhQUFhLEtBQUssUUFBUTtRQUN4QixDQUFDLENBQUMsRUFBRSxHQUFHLHNCQUFzQixFQUFFLEdBQUcsaUJBQWlCLEVBQUU7UUFDckQsQ0FBQyxDQUFDLEVBQUUsR0FBRyxpQkFBaUIsRUFBRSxHQUFHLHNCQUFzQixFQUFFLENBQUM7SUFFMUQsSUFBSTtRQUNGLE1BQU0sV0FBVyxHQUFHLGNBQWMsQ0FBQyxhQUFhLENBQUMsQ0FBQztRQUNsRCxJQUFJLFdBQVcsS0FBSyxTQUFTLEVBQUU7WUFDN0IsTUFBTSxJQUFJLEtBQUssRUFBRSxDQUFDO1NBQ25CO1FBQ0QsT0FBTyxXQUFXLENBQUM7S0FDcEI7SUFBQyxPQUFPLENBQUMsRUFBRTtRQUNWLE1BQU0sSUFBSSxhQUFhLENBQ3JCLENBQUMsQ0FBQyxPQUFPLElBQUksa0NBQWtDLE9BQU8sNENBQTRDLGNBQWMsRUFBRSxDQUNuSCxDQUFDO0tBQ0g7QUFDSCxDQUFDLENBQUMifQ==