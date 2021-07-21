import { HttpRequest } from "../protocol-http/mod.ts";
const CONTENT_LENGTH_HEADER = "content-length";
export function contentLengthMiddleware(bodyLengthChecker) {
    return (next) => async (args) => {
        const request = args.request;
        if (HttpRequest.isInstance(request)) {
            const { body, headers } = request;
            if (body &&
                Object.keys(headers)
                    .map((str) => str.toLowerCase())
                    .indexOf(CONTENT_LENGTH_HEADER) === -1) {
                const length = bodyLengthChecker(body);
                if (length !== undefined) {
                    request.headers = {
                        ...request.headers,
                        [CONTENT_LENGTH_HEADER]: String(length),
                    };
                }
            }
        }
        return next({
            ...args,
            request,
        });
    };
}
export const contentLengthMiddlewareOptions = {
    step: "build",
    tags: ["SET_CONTENT_LENGTH", "CONTENT_LENGTH"],
    name: "contentLengthMiddleware",
    override: true,
};
export const getContentLengthPlugin = (options) => ({
    applyToStack: (clientStack) => {
        clientStack.add(contentLengthMiddleware(options.bodyLengthChecker), contentLengthMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSx5QkFBeUIsQ0FBQztBQVl0RCxNQUFNLHFCQUFxQixHQUFHLGdCQUFnQixDQUFDO0FBRS9DLE1BQU0sVUFBVSx1QkFBdUIsQ0FBQyxpQkFBdUM7SUFDN0UsT0FBTyxDQUFnQyxJQUErQixFQUE2QixFQUFFLENBQ25HLEtBQUssRUFBRSxJQUFnQyxFQUF1QyxFQUFFO1FBQzlFLE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxPQUFPLENBQUM7UUFDN0IsSUFBSSxXQUFXLENBQUMsVUFBVSxDQUFDLE9BQU8sQ0FBQyxFQUFFO1lBQ25DLE1BQU0sRUFBRSxJQUFJLEVBQUUsT0FBTyxFQUFFLEdBQUcsT0FBTyxDQUFDO1lBQ2xDLElBQ0UsSUFBSTtnQkFDSixNQUFNLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQztxQkFDakIsR0FBRyxDQUFDLENBQUMsR0FBRyxFQUFFLEVBQUUsQ0FBQyxHQUFHLENBQUMsV0FBVyxFQUFFLENBQUM7cUJBQy9CLE9BQU8sQ0FBQyxxQkFBcUIsQ0FBQyxLQUFLLENBQUMsQ0FBQyxFQUN4QztnQkFDQSxNQUFNLE1BQU0sR0FBRyxpQkFBaUIsQ0FBQyxJQUFJLENBQUMsQ0FBQztnQkFDdkMsSUFBSSxNQUFNLEtBQUssU0FBUyxFQUFFO29CQUN4QixPQUFPLENBQUMsT0FBTyxHQUFHO3dCQUNoQixHQUFHLE9BQU8sQ0FBQyxPQUFPO3dCQUNsQixDQUFDLHFCQUFxQixDQUFDLEVBQUUsTUFBTSxDQUFDLE1BQU0sQ0FBQztxQkFDeEMsQ0FBQztpQkFDSDthQUNGO1NBQ0Y7UUFFRCxPQUFPLElBQUksQ0FBQztZQUNWLEdBQUcsSUFBSTtZQUNQLE9BQU87U0FDUixDQUFDLENBQUM7SUFDTCxDQUFDLENBQUM7QUFDTixDQUFDO0FBRUQsTUFBTSxDQUFDLE1BQU0sOEJBQThCLEdBQXdCO0lBQ2pFLElBQUksRUFBRSxPQUFPO0lBQ2IsSUFBSSxFQUFFLENBQUMsb0JBQW9CLEVBQUUsZ0JBQWdCLENBQUM7SUFDOUMsSUFBSSxFQUFFLHlCQUF5QjtJQUMvQixRQUFRLEVBQUUsSUFBSTtDQUNmLENBQUM7QUFFRixNQUFNLENBQUMsTUFBTSxzQkFBc0IsR0FBRyxDQUFDLE9BQW9ELEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQ3BILFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxHQUFHLENBQUMsdUJBQXVCLENBQUMsT0FBTyxDQUFDLGlCQUFpQixDQUFDLEVBQUUsOEJBQThCLENBQUMsQ0FBQztJQUN0RyxDQUFDO0NBQ0YsQ0FBQyxDQUFDIn0=