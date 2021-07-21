import process from "https://deno.land/std@0.101.0/node/process.ts";
import { loadSharedConfigFiles, } from "../shared-ini-file-loader/mod.ts";
export const ENV_PROFILE = "AWS_PROFILE";
export const DEFAULT_PROFILE = "default";
export const parseKnownFiles = async (init) => {
    const { loadedConfig = loadSharedConfigFiles(init) } = init;
    const parsedFiles = await loadedConfig;
    return {
        ...parsedFiles.configFile,
        ...parsedFiles.credentialsFile,
    };
};
export const getMasterProfileName = (init) => init.profile || process.env[ENV_PROFILE] || DEFAULT_PROFILE;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sT0FBTyxNQUFNLCtDQUErQyxDQUFDO0FBQ3BFLE9BQU8sRUFDTCxxQkFBcUIsR0FJdEIsTUFBTSxrQ0FBa0MsQ0FBQztBQUUxQyxNQUFNLENBQUMsTUFBTSxXQUFXLEdBQUcsYUFBYSxDQUFDO0FBQ3pDLE1BQU0sQ0FBQyxNQUFNLGVBQWUsR0FBRyxTQUFTLENBQUM7QUF1QnpDLE1BQU0sQ0FBQyxNQUFNLGVBQWUsR0FBRyxLQUFLLEVBQUUsSUFBdUIsRUFBMEIsRUFBRTtJQUN2RixNQUFNLEVBQUUsWUFBWSxHQUFHLHFCQUFxQixDQUFDLElBQUksQ0FBQyxFQUFFLEdBQUcsSUFBSSxDQUFDO0lBRTVELE1BQU0sV0FBVyxHQUFHLE1BQU0sWUFBWSxDQUFDO0lBQ3ZDLE9BQU87UUFDTCxHQUFHLFdBQVcsQ0FBQyxVQUFVO1FBQ3pCLEdBQUcsV0FBVyxDQUFDLGVBQWU7S0FDL0IsQ0FBQztBQUNKLENBQUMsQ0FBQztBQUtGLE1BQU0sQ0FBQyxNQUFNLG9CQUFvQixHQUFHLENBQUMsSUFBMEIsRUFBVSxFQUFFLENBQ3pFLElBQUksQ0FBQyxPQUFPLElBQUksT0FBTyxDQUFDLEdBQUcsQ0FBQyxXQUFXLENBQUMsSUFBSSxlQUFlLENBQUMifQ==