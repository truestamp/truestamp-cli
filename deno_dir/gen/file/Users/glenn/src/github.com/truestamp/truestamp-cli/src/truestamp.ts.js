import { getAccessTokenWithPrompts, Truestamp } from "./deps.ts";
export async function createTruestampClient(apiEnv) {
    const client = new Truestamp({
        accessToken: await getAccessTokenWithPrompts(apiEnv),
        apiEnv: apiEnv,
    });
    return client;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidHJ1ZXN0YW1wLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsidHJ1ZXN0YW1wLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSx5QkFBeUIsRUFBRSxTQUFTLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFFakUsTUFBTSxDQUFDLEtBQUssVUFBVSxxQkFBcUIsQ0FBQyxNQUFjO0lBRXhELE1BQU0sTUFBTSxHQUFHLElBQUssU0FBaUIsQ0FBQztRQUNwQyxXQUFXLEVBQUUsTUFBTSx5QkFBeUIsQ0FBQyxNQUFNLENBQUM7UUFDcEQsTUFBTSxFQUFFLE1BQU07S0FDZixDQUFDLENBQUM7SUFFSCxPQUFPLE1BQU0sQ0FBQztBQUNoQixDQUFDIn0=