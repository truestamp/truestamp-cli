export const isImdsCredentials = (arg) => Boolean(arg) &&
    typeof arg === "object" &&
    typeof arg.AccessKeyId === "string" &&
    typeof arg.SecretAccessKey === "string" &&
    typeof arg.Token === "string" &&
    typeof arg.Expiration === "string";
export const fromImdsCredentials = (creds) => ({
    accessKeyId: creds.AccessKeyId,
    secretAccessKey: creds.SecretAccessKey,
    sessionToken: creds.Token,
    expiration: new Date(creds.Expiration),
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiSW1kc0NyZWRlbnRpYWxzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiSW1kc0NyZWRlbnRpYWxzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQVNBLE1BQU0sQ0FBQyxNQUFNLGlCQUFpQixHQUFHLENBQUMsR0FBUSxFQUEwQixFQUFFLENBQ3BFLE9BQU8sQ0FBQyxHQUFHLENBQUM7SUFDWixPQUFPLEdBQUcsS0FBSyxRQUFRO0lBQ3ZCLE9BQU8sR0FBRyxDQUFDLFdBQVcsS0FBSyxRQUFRO0lBQ25DLE9BQU8sR0FBRyxDQUFDLGVBQWUsS0FBSyxRQUFRO0lBQ3ZDLE9BQU8sR0FBRyxDQUFDLEtBQUssS0FBSyxRQUFRO0lBQzdCLE9BQU8sR0FBRyxDQUFDLFVBQVUsS0FBSyxRQUFRLENBQUM7QUFFckMsTUFBTSxDQUFDLE1BQU0sbUJBQW1CLEdBQUcsQ0FBQyxLQUFzQixFQUFlLEVBQUUsQ0FBQyxDQUFDO0lBQzNFLFdBQVcsRUFBRSxLQUFLLENBQUMsV0FBVztJQUM5QixlQUFlLEVBQUUsS0FBSyxDQUFDLGVBQWU7SUFDdEMsWUFBWSxFQUFFLEtBQUssQ0FBQyxLQUFLO0lBQ3pCLFVBQVUsRUFBRSxJQUFJLElBQUksQ0FBQyxLQUFLLENBQUMsVUFBVSxDQUFDO0NBQ3ZDLENBQUMsQ0FBQyJ9