import process from "https://deno.land/std@0.97.0/node/process.ts";
import { ProviderError } from "../property-provider/mod.ts";
export const fromEnv = (envVarSelector) => async () => {
    try {
        const config = envVarSelector(process.env);
        if (config === undefined) {
            throw new Error();
        }
        return config;
    }
    catch (e) {
        throw new ProviderError(e.message || `Cannot load config from environment variables with getter: ${envVarSelector}`);
    }
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnJvbUVudi5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImZyb21FbnYudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxPQUFPLE1BQU0sOENBQThDLENBQUM7QUFDbkUsT0FBTyxFQUFFLGFBQWEsRUFBRSxNQUFNLDZCQUE2QixDQUFDO0FBUzVELE1BQU0sQ0FBQyxNQUFNLE9BQU8sR0FBRyxDQUFhLGNBQWdDLEVBQWUsRUFBRSxDQUFDLEtBQUssSUFBSSxFQUFFO0lBQy9GLElBQUk7UUFDRixNQUFNLE1BQU0sR0FBRyxjQUFjLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxDQUFDO1FBQzNDLElBQUksTUFBTSxLQUFLLFNBQVMsRUFBRTtZQUN4QixNQUFNLElBQUksS0FBSyxFQUFFLENBQUM7U0FDbkI7UUFDRCxPQUFPLE1BQVcsQ0FBQztLQUNwQjtJQUFDLE9BQU8sQ0FBQyxFQUFFO1FBQ1YsTUFBTSxJQUFJLGFBQWEsQ0FDckIsQ0FBQyxDQUFDLE9BQU8sSUFBSSw4REFBOEQsY0FBYyxFQUFFLENBQzVGLENBQUM7S0FDSDtBQUNILENBQUMsQ0FBQyJ9