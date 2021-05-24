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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSx5QkFBeUIsQ0FBQztBQWV0RCxNQUFNLFVBQVUsMkJBQTJCLENBQUMsT0FBMkI7SUFDckUsT0FBTyxDQUFnQyxJQUErQixFQUE2QixFQUFFLENBQUMsS0FBSyxFQUN6RyxJQUFnQyxFQUNLLEVBQUU7UUFDdkMsTUFBTSxFQUFFLE9BQU8sRUFBRSxHQUFHLElBQUksQ0FBQztRQUN6QixJQUFJLFdBQVcsQ0FBQyxVQUFVLENBQUMsT0FBTyxDQUFDLElBQUksT0FBTyxDQUFDLElBQUksSUFBSSxPQUFPLENBQUMsT0FBTyxLQUFLLE1BQU0sRUFBRTtZQUNqRixPQUFPLENBQUMsT0FBTyxHQUFHO2dCQUNoQixHQUFHLE9BQU8sQ0FBQyxPQUFPO2dCQUNsQixNQUFNLEVBQUUsY0FBYzthQUN2QixDQUFDO1NBQ0g7UUFDRCxPQUFPLElBQUksQ0FBQztZQUNWLEdBQUcsSUFBSTtZQUNQLE9BQU87U0FDUixDQUFDLENBQUM7SUFDTCxDQUFDLENBQUM7QUFDSixDQUFDO0FBRUQsTUFBTSxDQUFDLE1BQU0sa0NBQWtDLEdBQXdCO0lBQ3JFLElBQUksRUFBRSxPQUFPO0lBQ2IsSUFBSSxFQUFFLENBQUMsbUJBQW1CLEVBQUUsZUFBZSxDQUFDO0lBQzVDLElBQUksRUFBRSw2QkFBNkI7SUFDbkMsUUFBUSxFQUFFLElBQUk7Q0FDZixDQUFDO0FBRUYsTUFBTSxDQUFDLE1BQU0sMEJBQTBCLEdBQUcsQ0FBQyxPQUEyQixFQUF1QixFQUFFLENBQUMsQ0FBQztJQUMvRixZQUFZLEVBQUUsQ0FBQyxXQUFXLEVBQUUsRUFBRTtRQUM1QixXQUFXLENBQUMsR0FBRyxDQUFDLDJCQUEyQixDQUFDLE9BQU8sQ0FBQyxFQUFFLGtDQUFrQyxDQUFDLENBQUM7SUFDNUYsQ0FBQztDQUNGLENBQUMsQ0FBQyJ9