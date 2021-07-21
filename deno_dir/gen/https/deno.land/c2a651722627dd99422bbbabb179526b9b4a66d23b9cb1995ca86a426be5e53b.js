import { HttpRequest } from "../protocol-http/mod.ts";
export function addExpectContinueMiddleware(options) {
    return (next) => async (args) => {
        const { request } = args;
        if (HttpRequest.isInstance(request) && request.body && options.runtime === "node") {
            request.headers = {
                ...request.headers,
                Expect: "100-continue",
            };
        }
        return next({
            ...args,
            request,
        });
    };
}
export const addExpectContinueMiddlewareOptions = {
    step: "build",
    tags: ["SET_EXPECT_HEADER", "EXPECT_HEADER"],
    name: "addExpectContinueMiddleware",
    override: true,
};
export const getAddExpectContinuePlugin = (options) => ({
    applyToStack: (clientStack) => {
        clientStack.add(addExpectContinueMiddleware(options), addExpectContinueMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSx5QkFBeUIsQ0FBQztBQWV0RCxNQUFNLFVBQVUsMkJBQTJCLENBQUMsT0FBMkI7SUFDckUsT0FBTyxDQUFnQyxJQUErQixFQUE2QixFQUFFLENBQ25HLEtBQUssRUFBRSxJQUFnQyxFQUF1QyxFQUFFO1FBQzlFLE1BQU0sRUFBRSxPQUFPLEVBQUUsR0FBRyxJQUFJLENBQUM7UUFDekIsSUFBSSxXQUFXLENBQUMsVUFBVSxDQUFDLE9BQU8sQ0FBQyxJQUFJLE9BQU8sQ0FBQyxJQUFJLElBQUksT0FBTyxDQUFDLE9BQU8sS0FBSyxNQUFNLEVBQUU7WUFDakYsT0FBTyxDQUFDLE9BQU8sR0FBRztnQkFDaEIsR0FBRyxPQUFPLENBQUMsT0FBTztnQkFDbEIsTUFBTSxFQUFFLGNBQWM7YUFDdkIsQ0FBQztTQUNIO1FBQ0QsT0FBTyxJQUFJLENBQUM7WUFDVixHQUFHLElBQUk7WUFDUCxPQUFPO1NBQ1IsQ0FBQyxDQUFDO0lBQ0wsQ0FBQyxDQUFDO0FBQ04sQ0FBQztBQUVELE1BQU0sQ0FBQyxNQUFNLGtDQUFrQyxHQUF3QjtJQUNyRSxJQUFJLEVBQUUsT0FBTztJQUNiLElBQUksRUFBRSxDQUFDLG1CQUFtQixFQUFFLGVBQWUsQ0FBQztJQUM1QyxJQUFJLEVBQUUsNkJBQTZCO0lBQ25DLFFBQVEsRUFBRSxJQUFJO0NBQ2YsQ0FBQztBQUVGLE1BQU0sQ0FBQyxNQUFNLDBCQUEwQixHQUFHLENBQUMsT0FBMkIsRUFBdUIsRUFBRSxDQUFDLENBQUM7SUFDL0YsWUFBWSxFQUFFLENBQUMsV0FBVyxFQUFFLEVBQUU7UUFDNUIsV0FBVyxDQUFDLEdBQUcsQ0FBQywyQkFBMkIsQ0FBQyxPQUFPLENBQUMsRUFBRSxrQ0FBa0MsQ0FBQyxDQUFDO0lBQzVGLENBQUM7Q0FDRixDQUFDLENBQUMifQ==