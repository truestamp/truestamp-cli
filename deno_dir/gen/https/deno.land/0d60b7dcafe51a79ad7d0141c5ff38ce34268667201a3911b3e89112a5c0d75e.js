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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidXNlLXJlZ2lvbmFsLWVuZHBvaW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsidXNlLXJlZ2lvbmFsLWVuZHBvaW50LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSx5QkFBeUIsQ0FBQztBQW9CdEQsTUFBTSxDQUFDLE1BQU0sNkJBQTZCLEdBQ3hDLENBQUMsTUFBMEIsRUFBNkIsRUFBRSxDQUMxRCxDQUFnQyxJQUErQixFQUE2QixFQUFFLENBQzlGLEtBQUssRUFBRSxJQUFnQyxFQUF1QyxFQUFFO0lBQzlFLE1BQU0sRUFBRSxPQUFPLEVBQUUsR0FBRyxJQUFJLENBQUM7SUFDekIsSUFBSSxDQUFDLFdBQVcsQ0FBQyxVQUFVLENBQUMsT0FBTyxDQUFDLElBQUksTUFBTSxDQUFDLGdCQUFnQjtRQUFFLE9BQU8sSUFBSSxDQUFDLEVBQUUsR0FBRyxJQUFJLEVBQUUsQ0FBQyxDQUFDO0lBQzFGLElBQUksT0FBTyxDQUFDLFFBQVEsS0FBSyxrQkFBa0IsRUFBRTtRQUMzQyxPQUFPLENBQUMsUUFBUSxHQUFHLDRCQUE0QixDQUFDO0tBQ2pEO1NBQU0sSUFBSSxZQUFZLEtBQUssQ0FBQyxNQUFNLE1BQU0sQ0FBQyxNQUFNLEVBQUUsQ0FBQyxFQUFFO1FBQ25ELE9BQU8sQ0FBQyxRQUFRLEdBQUcsa0JBQWtCLENBQUM7S0FDdkM7SUFDRCxPQUFPLElBQUksQ0FBQyxFQUFFLEdBQUcsSUFBSSxFQUFFLENBQUMsQ0FBQztBQUMzQixDQUFDLENBQUM7QUFLSixNQUFNLENBQUMsTUFBTSxvQ0FBb0MsR0FBd0I7SUFDdkUsSUFBSSxFQUFFLE9BQU87SUFDYixJQUFJLEVBQUUsQ0FBQyx1QkFBdUIsRUFBRSxJQUFJLENBQUM7SUFDckMsSUFBSSxFQUFFLCtCQUErQjtJQUNyQyxRQUFRLEVBQUUsSUFBSTtDQUNmLENBQUM7QUFLRixNQUFNLENBQUMsTUFBTSw0QkFBNEIsR0FBRyxDQUFDLE1BQTBCLEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQ2hHLFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxHQUFHLENBQUMsNkJBQTZCLENBQUMsTUFBTSxDQUFDLEVBQUUsb0NBQW9DLENBQUMsQ0FBQztJQUMvRixDQUFDO0NBQ0YsQ0FBQyxDQUFDIn0=