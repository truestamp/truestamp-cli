import { getAccessTokenWithPrompts, Truestamp } from "./deps.ts";
export async function createTruestampClient(apiEnv) {
    const client = new Truestamp({
        accessToken: await getAccessTokenWithPrompts(apiEnv),
        apiEnv: apiEnv,
    });
    return client;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidHJ1ZXN0YW1wLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsidHJ1ZXN0YW1wLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSx5QkFBeUIsRUFBRSxTQUFTLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFFakUsTUFBTSxDQUFDLEtBQUssVUFBVSxxQkFBcUIsQ0FBQyxNQUFjO0lBQ3hELE1BQU0sTUFBTSxHQUFHLElBQUksU0FBUyxDQUFDO1FBQzNCLFdBQVcsRUFBRSxNQUFNLHlCQUF5QixDQUFDLE1BQU0sQ0FBQztRQUNwRCxNQUFNLEVBQUUsTUFBTTtLQUNmLENBQUMsQ0FBQztJQUVILE9BQU8sTUFBTSxDQUFDO0FBQ2hCLENBQUMifQ==