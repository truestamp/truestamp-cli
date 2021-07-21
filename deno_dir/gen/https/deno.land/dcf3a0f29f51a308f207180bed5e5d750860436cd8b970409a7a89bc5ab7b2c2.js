import process from "https://deno.land/std@0.101.0/node/process.ts";
import { CredentialsProviderError } from "../property-provider/mod.ts";
export const fromEnv = (envVarSelector) => async () => {
    try {
        const config = envVarSelector(process.env);
        if (config === undefined) {
            throw new Error();
        }
        return config;
    }
    catch (e) {
        throw new CredentialsProviderError(e.message || `Cannot load config from environment variables with getter: ${envVarSelector}`);
    }
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnJvbUVudi5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImZyb21FbnYudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxPQUFPLE1BQU0sK0NBQStDLENBQUM7QUFDcEUsT0FBTyxFQUFFLHdCQUF3QixFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFTdkUsTUFBTSxDQUFDLE1BQU0sT0FBTyxHQUNsQixDQUFhLGNBQWdDLEVBQWUsRUFBRSxDQUM5RCxLQUFLLElBQUksRUFBRTtJQUNULElBQUk7UUFDRixNQUFNLE1BQU0sR0FBRyxjQUFjLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxDQUFDO1FBQzNDLElBQUksTUFBTSxLQUFLLFNBQVMsRUFBRTtZQUN4QixNQUFNLElBQUksS0FBSyxFQUFFLENBQUM7U0FDbkI7UUFDRCxPQUFPLE1BQVcsQ0FBQztLQUNwQjtJQUFDLE9BQU8sQ0FBQyxFQUFFO1FBQ1YsTUFBTSxJQUFJLHdCQUF3QixDQUNoQyxDQUFDLENBQUMsT0FBTyxJQUFJLDhEQUE4RCxjQUFjLEVBQUUsQ0FDNUYsQ0FBQztLQUNIO0FBQ0gsQ0FBQyxDQUFDIn0=