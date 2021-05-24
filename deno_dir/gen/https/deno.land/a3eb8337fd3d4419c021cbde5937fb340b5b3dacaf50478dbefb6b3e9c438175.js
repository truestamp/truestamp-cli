import { HttpRequest } from "../protocol-http/mod.ts";
export const useRegionalEndpointMiddleware = (config) => (next) => async (args) => {
    const { request } = args;
    if (!HttpRequest.isInstance(request) || config.isCustomEndpoint)
        return next({ ...args });
    if (request.hostname === "s3.amazonaws.com") {
        request.hostname = "s3.us-east-1.amazonaws.com";
    }
    else if ("aws-global" === (await config.region())) {
        request.hostname = "s3.amazonaws.com";
    }
    return next({ ...args });
};
export const useRegionalEndpointMiddlewareOptions = {
    step: "build",
    tags: ["USE_REGIONAL_ENDPOINT", "S3"],
    name: "useRegionalEndpointMiddleware",
    override: true,
};
export const getUseRegionalEndpointPlugin = (config) => ({
    applyToStack: (clientStack) => {
        clientStack.add(useRegionalEndpointMiddleware(config), useRegionalEndpointMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidXNlLXJlZ2lvbmFsLWVuZHBvaW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsidXNlLXJlZ2lvbmFsLWVuZHBvaW50LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSx5QkFBeUIsQ0FBQztBQW9CdEQsTUFBTSxDQUFDLE1BQU0sNkJBQTZCLEdBQUcsQ0FBQyxNQUEwQixFQUE2QixFQUFFLENBQUMsQ0FHdEcsSUFBK0IsRUFDSixFQUFFLENBQUMsS0FBSyxFQUFFLElBQWdDLEVBQXVDLEVBQUU7SUFDOUcsTUFBTSxFQUFFLE9BQU8sRUFBRSxHQUFHLElBQUksQ0FBQztJQUN6QixJQUFJLENBQUMsV0FBVyxDQUFDLFVBQVUsQ0FBQyxPQUFPLENBQUMsSUFBSSxNQUFNLENBQUMsZ0JBQWdCO1FBQUUsT0FBTyxJQUFJLENBQUMsRUFBRSxHQUFHLElBQUksRUFBRSxDQUFDLENBQUM7SUFDMUYsSUFBSSxPQUFPLENBQUMsUUFBUSxLQUFLLGtCQUFrQixFQUFFO1FBQzNDLE9BQU8sQ0FBQyxRQUFRLEdBQUcsNEJBQTRCLENBQUM7S0FDakQ7U0FBTSxJQUFJLFlBQVksS0FBSyxDQUFDLE1BQU0sTUFBTSxDQUFDLE1BQU0sRUFBRSxDQUFDLEVBQUU7UUFDbkQsT0FBTyxDQUFDLFFBQVEsR0FBRyxrQkFBa0IsQ0FBQztLQUN2QztJQUNELE9BQU8sSUFBSSxDQUFDLEVBQUUsR0FBRyxJQUFJLEVBQUUsQ0FBQyxDQUFDO0FBQzNCLENBQUMsQ0FBQztBQUtGLE1BQU0sQ0FBQyxNQUFNLG9DQUFvQyxHQUF3QjtJQUN2RSxJQUFJLEVBQUUsT0FBTztJQUNiLElBQUksRUFBRSxDQUFDLHVCQUF1QixFQUFFLElBQUksQ0FBQztJQUNyQyxJQUFJLEVBQUUsK0JBQStCO0lBQ3JDLFFBQVEsRUFBRSxJQUFJO0NBQ2YsQ0FBQztBQUtGLE1BQU0sQ0FBQyxNQUFNLDRCQUE0QixHQUFHLENBQUMsTUFBMEIsRUFBdUIsRUFBRSxDQUFDLENBQUM7SUFDaEcsWUFBWSxFQUFFLENBQUMsV0FBVyxFQUFFLEVBQUU7UUFDNUIsV0FBVyxDQUFDLEdBQUcsQ0FBQyw2QkFBNkIsQ0FBQyxNQUFNLENBQUMsRUFBRSxvQ0FBb0MsQ0FBQyxDQUFDO0lBQy9GLENBQUM7Q0FDRixDQUFDLENBQUMifQ==