import { CredentialsProviderError } from "../property-provider/mod.ts";
import { getMasterProfileName, parseKnownFiles } from "../util-credentials/mod.ts";
export const ENV_PROFILE = "AWS_PROFILE";
export const fromProcess = (init = {}) => async () => {
    const profiles = await parseKnownFiles(init);
    return resolveProcessCredentials(getMasterProfileName(init), profiles);
};
const resolveProcessCredentials = async (profileName, profiles) => {
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
                throw new CredentialsProviderError(error.message);
            });
        }
        else {
            throw new CredentialsProviderError(`Profile ${profileName} did not contain credential_process.`);
        }
    }
    else {
        throw new CredentialsProviderError(`Profile ${profileName} could not be found in shared credentials file.`);
    }
};
async function execPromise(command) {
    const process = Deno.run({
        cmd: ["sh", "-c", command],
        stdout: "piped",
    });
    const data = await process.output();
    return new TextDecoder().decode(data);
}
;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSx3QkFBd0IsRUFBRSxNQUFNLDZCQUE2QixDQUFDO0FBR3ZFLE9BQU8sRUFBRSxvQkFBb0IsRUFBRSxlQUFlLEVBQXFCLE1BQU0sNEJBQTRCLENBQUM7QUFLdEcsTUFBTSxDQUFDLE1BQU0sV0FBVyxHQUFHLGFBQWEsQ0FBQztBQVF6QyxNQUFNLENBQUMsTUFBTSxXQUFXLEdBQ3RCLENBQUMsT0FBd0IsRUFBRSxFQUFzQixFQUFFLENBQ25ELEtBQUssSUFBSSxFQUFFO0lBQ1QsTUFBTSxRQUFRLEdBQUcsTUFBTSxlQUFlLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDN0MsT0FBTyx5QkFBeUIsQ0FBQyxvQkFBb0IsQ0FBQyxJQUFJLENBQUMsRUFBRSxRQUFRLENBQUMsQ0FBQztBQUN6RSxDQUFDLENBQUM7QUFFSixNQUFNLHlCQUF5QixHQUFHLEtBQUssRUFBRSxXQUFtQixFQUFFLFFBQXVCLEVBQXdCLEVBQUU7SUFDN0csTUFBTSxPQUFPLEdBQUcsUUFBUSxDQUFDLFdBQVcsQ0FBQyxDQUFDO0lBRXRDLElBQUksUUFBUSxDQUFDLFdBQVcsQ0FBQyxFQUFFO1FBQ3pCLE1BQU0saUJBQWlCLEdBQUcsT0FBTyxDQUFDLG9CQUFvQixDQUFDLENBQUM7UUFDeEQsSUFBSSxpQkFBaUIsS0FBSyxTQUFTLEVBQUU7WUFDbkMsT0FBTyxNQUFNLFdBQVcsQ0FBQyxpQkFBaUIsQ0FBQztpQkFDeEMsSUFBSSxDQUFDLENBQUMsYUFBa0IsRUFBRSxFQUFFO2dCQUMzQixJQUFJLElBQUksQ0FBQztnQkFDVCxJQUFJO29CQUNGLElBQUksR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDLGFBQWEsQ0FBQyxDQUFDO2lCQUNsQztnQkFBQyxNQUFNO29CQUNOLE1BQU0sS0FBSyxDQUFDLFdBQVcsV0FBVyw0Q0FBNEMsQ0FBQyxDQUFDO2lCQUNqRjtnQkFFRCxNQUFNLEVBQ0osT0FBTyxFQUFFLE9BQU8sRUFDaEIsV0FBVyxFQUFFLFdBQVcsRUFDeEIsZUFBZSxFQUFFLGVBQWUsRUFDaEMsWUFBWSxFQUFFLFlBQVksRUFDMUIsVUFBVSxFQUFFLFVBQVUsR0FDdkIsR0FBRyxJQUFJLENBQUM7Z0JBRVQsSUFBSSxPQUFPLEtBQUssQ0FBQyxFQUFFO29CQUNqQixNQUFNLEtBQUssQ0FBQyxXQUFXLFdBQVcsK0NBQStDLENBQUMsQ0FBQztpQkFDcEY7Z0JBRUQsSUFBSSxXQUFXLEtBQUssU0FBUyxJQUFJLGVBQWUsS0FBSyxTQUFTLEVBQUU7b0JBQzlELE1BQU0sS0FBSyxDQUFDLFdBQVcsV0FBVyxtREFBbUQsQ0FBQyxDQUFDO2lCQUN4RjtnQkFFRCxJQUFJLGNBQWMsQ0FBQztnQkFFbkIsSUFBSSxVQUFVLEVBQUU7b0JBQ2QsTUFBTSxXQUFXLEdBQUcsSUFBSSxJQUFJLEVBQUUsQ0FBQztvQkFDL0IsTUFBTSxVQUFVLEdBQUcsSUFBSSxJQUFJLENBQUMsVUFBVSxDQUFDLENBQUM7b0JBQ3hDLElBQUksVUFBVSxHQUFHLFdBQVcsRUFBRTt3QkFDNUIsTUFBTSxLQUFLLENBQUMsV0FBVyxXQUFXLG1EQUFtRCxDQUFDLENBQUM7cUJBQ3hGO29CQUNELGNBQWMsR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksSUFBSSxDQUFDLFVBQVUsQ0FBQyxDQUFDLE9BQU8sRUFBRSxHQUFHLElBQUksQ0FBQyxDQUFDO2lCQUNwRTtnQkFFRCxPQUFPO29CQUNMLFdBQVc7b0JBQ1gsZUFBZTtvQkFDZixZQUFZO29CQUNaLGNBQWM7aUJBQ2YsQ0FBQztZQUNKLENBQUMsQ0FBQztpQkFDRCxLQUFLLENBQUMsQ0FBQyxLQUFZLEVBQUUsRUFBRTtnQkFDdEIsTUFBTSxJQUFJLHdCQUF3QixDQUFDLEtBQUssQ0FBQyxPQUFPLENBQUMsQ0FBQztZQUNwRCxDQUFDLENBQUMsQ0FBQztTQUNOO2FBQU07WUFDTCxNQUFNLElBQUksd0JBQXdCLENBQUMsV0FBVyxXQUFXLHNDQUFzQyxDQUFDLENBQUM7U0FDbEc7S0FDRjtTQUFNO1FBS0wsTUFBTSxJQUFJLHdCQUF3QixDQUFDLFdBQVcsV0FBVyxpREFBaUQsQ0FBQyxDQUFDO0tBQzdHO0FBQ0gsQ0FBQyxDQUFDO0FBRUYsS0FBSyxVQUFVLFdBQVcsQ0FBQyxPQUFlO0lBQ3hDLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUM7UUFDdkIsR0FBRyxFQUFFLENBQUMsSUFBSSxFQUFFLElBQUksRUFBRSxPQUFPLENBQUM7UUFDMUIsTUFBTSxFQUFFLE9BQU87S0FDaEIsQ0FBQyxDQUFDO0lBQ0gsTUFBTSxJQUFJLEdBQUcsTUFBTSxPQUFPLENBQUMsTUFBTSxFQUFFLENBQUM7SUFDcEMsT0FBTyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsQ0FBQztBQUN4QyxDQUFDO0FBQUEsQ0FBQyJ9