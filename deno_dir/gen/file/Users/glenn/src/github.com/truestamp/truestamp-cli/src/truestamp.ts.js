import { getAccessTokenWithPrompts, Truestamp } from "./deps.ts";
export async function createTruestampClient(apiEnv, apiKey) {
    const client = new Truestamp({
        apiKey: apiKey ?? await getAccessTokenWithPrompts(apiEnv),
        apiEnv: apiEnv ?? "production",
    });
    return client;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidHJ1ZXN0YW1wLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsidHJ1ZXN0YW1wLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSx5QkFBeUIsRUFBRSxTQUFTLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFFakUsTUFBTSxDQUFDLEtBQUssVUFBVSxxQkFBcUIsQ0FBQyxNQUFjLEVBQUUsTUFBZTtJQUN6RSxNQUFNLE1BQU0sR0FBRyxJQUFJLFNBQVMsQ0FBQztRQUMzQixNQUFNLEVBQUUsTUFBTSxJQUFJLE1BQU0seUJBQXlCLENBQUMsTUFBTSxDQUFDO1FBQ3pELE1BQU0sRUFBRSxNQUFNLElBQUksWUFBWTtLQUMvQixDQUFDLENBQUM7SUFFSCxPQUFPLE1BQU0sQ0FBQztBQUNoQixDQUFDIiwic291cmNlc0NvbnRlbnQiOlsiLy8gQ29weXJpZ2h0IMKpIDIwMjAtMjAyMiBUcnVlc3RhbXAgSW5jLiBBbGwgcmlnaHRzIHJlc2VydmVkLlxuXG5pbXBvcnQgeyBnZXRBY2Nlc3NUb2tlbldpdGhQcm9tcHRzLCBUcnVlc3RhbXAgfSBmcm9tIFwiLi9kZXBzLnRzXCI7XG5cbmV4cG9ydCBhc3luYyBmdW5jdGlvbiBjcmVhdGVUcnVlc3RhbXBDbGllbnQoYXBpRW52OiBzdHJpbmcsIGFwaUtleT86IHN0cmluZyk6IFByb21pc2U8VHJ1ZXN0YW1wPiB7XG4gIGNvbnN0IGNsaWVudCA9IG5ldyBUcnVlc3RhbXAoe1xuICAgIGFwaUtleTogYXBpS2V5ID8/IGF3YWl0IGdldEFjY2Vzc1Rva2VuV2l0aFByb21wdHMoYXBpRW52KSxcbiAgICBhcGlFbnY6IGFwaUVudiA/PyBcInByb2R1Y3Rpb25cIixcbiAgfSk7XG5cbiAgcmV0dXJuIGNsaWVudDtcbn1cbiJdfQ==