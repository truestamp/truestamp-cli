import { HttpRequest } from "../protocol-http/mod.ts";
const isClockSkewed = (newServerTime, systemClockOffset) => Math.abs(getSkewCorrectedDate(systemClockOffset).getTime() - newServerTime) >= 300000;
const getSkewCorrectedDate = (systemClockOffset) => new Date(Date.now() + systemClockOffset);
export function awsAuthMiddleware(options) {
    return (next, context) => async function (args) {
        if (!HttpRequest.isInstance(args.request))
            return next(args);
        const signer = typeof options.signer === "function" ? await options.signer() : options.signer;
        const output = await next({
            ...args,
            request: await signer.sign(args.request, {
                signingDate: new Date(Date.now() + options.systemClockOffset),
                signingRegion: context["signing_region"],
                signingService: context["signing_service"],
            }),
        });
        const { headers } = output.response;
        const dateHeader = headers && (headers.date || headers.Date);
        if (dateHeader) {
            const serverTime = Date.parse(dateHeader);
            if (isClockSkewed(serverTime, options.systemClockOffset)) {
                options.systemClockOffset = serverTime - Date.now();
            }
        }
        return output;
    };
}
export const awsAuthMiddlewareOptions = {
    name: "awsAuthMiddleware",
    tags: ["SIGNATURE", "AWSAUTH"],
    relation: "after",
    toMiddleware: "retryMiddleware",
    override: true,
};
export const getAwsAuthPlugin = (options) => ({
    applyToStack: (clientStack) => {
        clientStack.addRelativeTo(awsAuthMiddleware(options), awsAuthMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibWlkZGxld2FyZS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIm1pZGRsZXdhcmUudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLHlCQUF5QixDQUFDO0FBYXRELE1BQU0sYUFBYSxHQUFHLENBQUMsYUFBcUIsRUFBRSxpQkFBeUIsRUFBRSxFQUFFLENBQ3pFLElBQUksQ0FBQyxHQUFHLENBQUMsb0JBQW9CLENBQUMsaUJBQWlCLENBQUMsQ0FBQyxPQUFPLEVBQUUsR0FBRyxhQUFhLENBQUMsSUFBSSxNQUFNLENBQUM7QUFFeEYsTUFBTSxvQkFBb0IsR0FBRyxDQUFDLGlCQUF5QixFQUFFLEVBQUUsQ0FBQyxJQUFJLElBQUksQ0FBQyxJQUFJLENBQUMsR0FBRyxFQUFFLEdBQUcsaUJBQWlCLENBQUMsQ0FBQztBQUVyRyxNQUFNLFVBQVUsaUJBQWlCLENBQy9CLE9BQThCO0lBRTlCLE9BQU8sQ0FBQyxJQUFvQyxFQUFFLE9BQWdDLEVBQWtDLEVBQUUsQ0FDaEgsS0FBSyxXQUFXLElBQXFDO1FBQ25ELElBQUksQ0FBQyxXQUFXLENBQUMsVUFBVSxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUM7WUFBRSxPQUFPLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQztRQUM3RCxNQUFNLE1BQU0sR0FBRyxPQUFPLE9BQU8sQ0FBQyxNQUFNLEtBQUssVUFBVSxDQUFDLENBQUMsQ0FBQyxNQUFNLE9BQU8sQ0FBQyxNQUFNLEVBQUUsQ0FBQyxDQUFDLENBQUMsT0FBTyxDQUFDLE1BQU0sQ0FBQztRQUM5RixNQUFNLE1BQU0sR0FBRyxNQUFNLElBQUksQ0FBQztZQUN4QixHQUFHLElBQUk7WUFDUCxPQUFPLEVBQUUsTUFBTSxNQUFNLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxPQUFPLEVBQUU7Z0JBQ3ZDLFdBQVcsRUFBRSxJQUFJLElBQUksQ0FBQyxJQUFJLENBQUMsR0FBRyxFQUFFLEdBQUcsT0FBTyxDQUFDLGlCQUFpQixDQUFDO2dCQUM3RCxhQUFhLEVBQUUsT0FBTyxDQUFDLGdCQUFnQixDQUFDO2dCQUN4QyxjQUFjLEVBQUUsT0FBTyxDQUFDLGlCQUFpQixDQUFDO2FBQzNDLENBQUM7U0FDSCxDQUFDLENBQUM7UUFFSCxNQUFNLEVBQUUsT0FBTyxFQUFFLEdBQUcsTUFBTSxDQUFDLFFBQWUsQ0FBQztRQUMzQyxNQUFNLFVBQVUsR0FBRyxPQUFPLElBQUksQ0FBQyxPQUFPLENBQUMsSUFBSSxJQUFJLE9BQU8sQ0FBQyxJQUFJLENBQUMsQ0FBQztRQUM3RCxJQUFJLFVBQVUsRUFBRTtZQUNkLE1BQU0sVUFBVSxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUMsVUFBVSxDQUFDLENBQUM7WUFDMUMsSUFBSSxhQUFhLENBQUMsVUFBVSxFQUFFLE9BQU8sQ0FBQyxpQkFBaUIsQ0FBQyxFQUFFO2dCQUN4RCxPQUFPLENBQUMsaUJBQWlCLEdBQUcsVUFBVSxHQUFHLElBQUksQ0FBQyxHQUFHLEVBQUUsQ0FBQzthQUNyRDtTQUNGO1FBRUQsT0FBTyxNQUFNLENBQUM7SUFDaEIsQ0FBQyxDQUFDO0FBQ04sQ0FBQztBQUVELE1BQU0sQ0FBQyxNQUFNLHdCQUF3QixHQUE4QjtJQUNqRSxJQUFJLEVBQUUsbUJBQW1CO0lBQ3pCLElBQUksRUFBRSxDQUFDLFdBQVcsRUFBRSxTQUFTLENBQUM7SUFDOUIsUUFBUSxFQUFFLE9BQU87SUFDakIsWUFBWSxFQUFFLGlCQUFpQjtJQUMvQixRQUFRLEVBQUUsSUFBSTtDQUNmLENBQUM7QUFFRixNQUFNLENBQUMsTUFBTSxnQkFBZ0IsR0FBRyxDQUFDLE9BQThCLEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQ3hGLFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxhQUFhLENBQUMsaUJBQWlCLENBQUMsT0FBTyxDQUFDLEVBQUUsd0JBQXdCLENBQUMsQ0FBQztJQUNsRixDQUFDO0NBQ0YsQ0FBQyxDQUFDIn0=