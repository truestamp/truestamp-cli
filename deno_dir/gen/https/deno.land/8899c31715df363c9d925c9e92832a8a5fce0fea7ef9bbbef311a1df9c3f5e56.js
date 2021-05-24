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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSx5QkFBeUIsQ0FBQztBQVl0RCxNQUFNLHFCQUFxQixHQUFHLGdCQUFnQixDQUFDO0FBRS9DLE1BQU0sVUFBVSx1QkFBdUIsQ0FBQyxpQkFBdUM7SUFDN0UsT0FBTyxDQUFnQyxJQUErQixFQUE2QixFQUFFLENBQUMsS0FBSyxFQUN6RyxJQUFnQyxFQUNLLEVBQUU7UUFDdkMsTUFBTSxPQUFPLEdBQUcsSUFBSSxDQUFDLE9BQU8sQ0FBQztRQUM3QixJQUFJLFdBQVcsQ0FBQyxVQUFVLENBQUMsT0FBTyxDQUFDLEVBQUU7WUFDbkMsTUFBTSxFQUFFLElBQUksRUFBRSxPQUFPLEVBQUUsR0FBRyxPQUFPLENBQUM7WUFDbEMsSUFDRSxJQUFJO2dCQUNKLE1BQU0sQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDO3FCQUNqQixHQUFHLENBQUMsQ0FBQyxHQUFHLEVBQUUsRUFBRSxDQUFDLEdBQUcsQ0FBQyxXQUFXLEVBQUUsQ0FBQztxQkFDL0IsT0FBTyxDQUFDLHFCQUFxQixDQUFDLEtBQUssQ0FBQyxDQUFDLEVBQ3hDO2dCQUNBLE1BQU0sTUFBTSxHQUFHLGlCQUFpQixDQUFDLElBQUksQ0FBQyxDQUFDO2dCQUN2QyxJQUFJLE1BQU0sS0FBSyxTQUFTLEVBQUU7b0JBQ3hCLE9BQU8sQ0FBQyxPQUFPLEdBQUc7d0JBQ2hCLEdBQUcsT0FBTyxDQUFDLE9BQU87d0JBQ2xCLENBQUMscUJBQXFCLENBQUMsRUFBRSxNQUFNLENBQUMsTUFBTSxDQUFDO3FCQUN4QyxDQUFDO2lCQUNIO2FBQ0Y7U0FDRjtRQUVELE9BQU8sSUFBSSxDQUFDO1lBQ1YsR0FBRyxJQUFJO1lBQ1AsT0FBTztTQUNSLENBQUMsQ0FBQztJQUNMLENBQUMsQ0FBQztBQUNKLENBQUM7QUFFRCxNQUFNLENBQUMsTUFBTSw4QkFBOEIsR0FBd0I7SUFDakUsSUFBSSxFQUFFLE9BQU87SUFDYixJQUFJLEVBQUUsQ0FBQyxvQkFBb0IsRUFBRSxnQkFBZ0IsQ0FBQztJQUM5QyxJQUFJLEVBQUUseUJBQXlCO0lBQy9CLFFBQVEsRUFBRSxJQUFJO0NBQ2YsQ0FBQztBQUVGLE1BQU0sQ0FBQyxNQUFNLHNCQUFzQixHQUFHLENBQUMsT0FBb0QsRUFBdUIsRUFBRSxDQUFDLENBQUM7SUFDcEgsWUFBWSxFQUFFLENBQUMsV0FBVyxFQUFFLEVBQUU7UUFDNUIsV0FBVyxDQUFDLEdBQUcsQ0FBQyx1QkFBdUIsQ0FBQyxPQUFPLENBQUMsaUJBQWlCLENBQUMsRUFBRSw4QkFBOEIsQ0FBQyxDQUFDO0lBQ3RHLENBQUM7Q0FDRixDQUFDLENBQUMifQ==