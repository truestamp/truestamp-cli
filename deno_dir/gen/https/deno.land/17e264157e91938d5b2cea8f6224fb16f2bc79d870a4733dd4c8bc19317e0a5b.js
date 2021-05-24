import { getMasterProfileName, parseKnownFiles } from "../credential-provider-ini/mod.ts";
import { ProviderError } from "../property-provider/mod.ts";
export const ENV_PROFILE = "AWS_PROFILE";
export function fromProcess(init = {}) {
    return () => parseKnownFiles(init).then((profiles) => resolveProcessCredentials(getMasterProfileName(init), profiles, init));
}
async function resolveProcessCredentials(profileName, profiles, options) {
    const profile = profiles[profileName];
    if (profiles[profileName]) {
        const credentialProcess = profile["credential_process"];
        if (credentialProcess !== undefined) {
            return await execPromise(credentialProcess)
                .then((processResult) => {
                let data;
                try {
                    data = JSON.parse(processResult);
                }
                catch {
                    throw Error(`Profile ${profileName} credential_process returned invalid JSON.`);
                }
                const { Version: version, AccessKeyId: accessKeyId, SecretAccessKey: secretAccessKey, SessionToken: sessionToken, Expiration: expiration, } = data;
                if (version !== 1) {
                    throw Error(`Profile ${profileName} credential_process did not return Version 1.`);
                }
                if (accessKeyId === undefined || secretAccessKey === undefined) {
                    throw Error(`Profile ${profileName} credential_process returned invalid credentials.`);
                }
                let expirationUnix;
                if (expiration) {
                    const currentTime = new Date();
                    const expireTime = new Date(expiration);
                    if (expireTime < currentTime) {
                        throw Error(`Profile ${profileName} credential_process returned expired credentials.`);
                    }
                    expirationUnix = Math.floor(new Date(expiration).valueOf() / 1000);
                }
                return {
                    accessKeyId,
                    secretAccessKey,
                    sessionToken,
                    expirationUnix,
                };
            })
                .catch((error) => {
                throw new ProviderError(error.message);
            });
        }
        else {
            throw new ProviderError(`Profile ${profileName} did not contain credential_process.`);
        }
    }
    else {
        throw new ProviderError(`Profile ${profileName} could not be found in shared credentials file.`);
    }
}
async function execPromise(command) {
    const process = Deno.run({
        cmd: ["sh", "-c", command],
        stdout: "piped",
    });
    const data = await process.output();
    return new TextDecoder().decode(data);
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxvQkFBb0IsRUFBRSxlQUFlLEVBQUUsTUFBTSxtQ0FBbUMsQ0FBQztBQUMxRixPQUFPLEVBQUUsYUFBYSxFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFJNUQsTUFBTSxDQUFDLE1BQU0sV0FBVyxHQUFHLGFBQWEsQ0FBQztBQW1CekMsTUFBTSxVQUFVLFdBQVcsQ0FBQyxPQUF3QixFQUFFO0lBQ3BELE9BQU8sR0FBRyxFQUFFLENBQ1YsZUFBZSxDQUFDLElBQUksQ0FBQyxDQUFDLElBQUksQ0FBQyxDQUFDLFFBQVEsRUFBRSxFQUFFLENBQUMseUJBQXlCLENBQUMsb0JBQW9CLENBQUMsSUFBSSxDQUFDLEVBQUUsUUFBUSxFQUFFLElBQUksQ0FBQyxDQUFDLENBQUM7QUFDcEgsQ0FBQztBQUVELEtBQUssVUFBVSx5QkFBeUIsQ0FDdEMsV0FBbUIsRUFDbkIsUUFBdUIsRUFFdkIsT0FBd0I7SUFFeEIsTUFBTSxPQUFPLEdBQUcsUUFBUSxDQUFDLFdBQVcsQ0FBQyxDQUFDO0lBRXRDLElBQUksUUFBUSxDQUFDLFdBQVcsQ0FBQyxFQUFFO1FBQ3pCLE1BQU0saUJBQWlCLEdBQUcsT0FBTyxDQUFDLG9CQUFvQixDQUFDLENBQUM7UUFDeEQsSUFBSSxpQkFBaUIsS0FBSyxTQUFTLEVBQUU7WUFDbkMsT0FBTyxNQUFNLFdBQVcsQ0FBQyxpQkFBaUIsQ0FBQztpQkFDeEMsSUFBSSxDQUFDLENBQUMsYUFBa0IsRUFBRSxFQUFFO2dCQUMzQixJQUFJLElBQUksQ0FBQztnQkFDVCxJQUFJO29CQUNGLElBQUksR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDLGFBQWEsQ0FBQyxDQUFDO2lCQUNsQztnQkFBQyxNQUFNO29CQUNOLE1BQU0sS0FBSyxDQUFDLFdBQVcsV0FBVyw0Q0FBNEMsQ0FBQyxDQUFDO2lCQUNqRjtnQkFFRCxNQUFNLEVBQ0osT0FBTyxFQUFFLE9BQU8sRUFDaEIsV0FBVyxFQUFFLFdBQVcsRUFDeEIsZUFBZSxFQUFFLGVBQWUsRUFDaEMsWUFBWSxFQUFFLFlBQVksRUFDMUIsVUFBVSxFQUFFLFVBQVUsR0FDdkIsR0FBRyxJQUFJLENBQUM7Z0JBRVQsSUFBSSxPQUFPLEtBQUssQ0FBQyxFQUFFO29CQUNqQixNQUFNLEtBQUssQ0FBQyxXQUFXLFdBQVcsK0NBQStDLENBQUMsQ0FBQztpQkFDcEY7Z0JBRUQsSUFBSSxXQUFXLEtBQUssU0FBUyxJQUFJLGVBQWUsS0FBSyxTQUFTLEVBQUU7b0JBQzlELE1BQU0sS0FBSyxDQUFDLFdBQVcsV0FBVyxtREFBbUQsQ0FBQyxDQUFDO2lCQUN4RjtnQkFFRCxJQUFJLGNBQWMsQ0FBQztnQkFFbkIsSUFBSSxVQUFVLEVBQUU7b0JBQ2QsTUFBTSxXQUFXLEdBQUcsSUFBSSxJQUFJLEVBQUUsQ0FBQztvQkFDL0IsTUFBTSxVQUFVLEdBQUcsSUFBSSxJQUFJLENBQUMsVUFBVSxDQUFDLENBQUM7b0JBQ3hDLElBQUksVUFBVSxHQUFHLFdBQVcsRUFBRTt3QkFDNUIsTUFBTSxLQUFLLENBQUMsV0FBVyxXQUFXLG1EQUFtRCxDQUFDLENBQUM7cUJBQ3hGO29CQUNELGNBQWMsR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksSUFBSSxDQUFDLFVBQVUsQ0FBQyxDQUFDLE9BQU8sRUFBRSxHQUFHLElBQUksQ0FBQyxDQUFDO2lCQUNwRTtnQkFFRCxPQUFPO29CQUNMLFdBQVc7b0JBQ1gsZUFBZTtvQkFDZixZQUFZO29CQUNaLGNBQWM7aUJBQ2YsQ0FBQztZQUNKLENBQUMsQ0FBQztpQkFDRCxLQUFLLENBQUMsQ0FBQyxLQUFZLEVBQUUsRUFBRTtnQkFDdEIsTUFBTSxJQUFJLGFBQWEsQ0FBQyxLQUFLLENBQUMsT0FBTyxDQUFDLENBQUM7WUFDekMsQ0FBQyxDQUFDLENBQUM7U0FDTjthQUFNO1lBQ0wsTUFBTSxJQUFJLGFBQWEsQ0FBQyxXQUFXLFdBQVcsc0NBQXNDLENBQUMsQ0FBQztTQUN2RjtLQUNGO1NBQU07UUFLTCxNQUFNLElBQUksYUFBYSxDQUFDLFdBQVcsV0FBVyxpREFBaUQsQ0FBQyxDQUFDO0tBQ2xHO0FBQ0gsQ0FBQztBQUVELEtBQUssVUFBVSxXQUFXLENBQUMsT0FBZTtJQUN4QyxNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDO1FBQ3ZCLEdBQUcsRUFBRSxDQUFDLElBQUksRUFBRSxJQUFJLEVBQUUsT0FBTyxDQUFDO1FBQzFCLE1BQU0sRUFBRSxPQUFPO0tBQ2hCLENBQUMsQ0FBQztJQUNILE1BQU0sSUFBSSxHQUFHLE1BQU0sT0FBTyxDQUFDLE1BQU0sRUFBRSxDQUFDO0lBQ3BDLE9BQU8sSUFBSSxXQUFXLEVBQUUsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLENBQUM7QUFDeEMsQ0FBQyJ9