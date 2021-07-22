export const retryMiddleware = (options) => (next, context) => async (args) => {
    if (options?.retryStrategy?.mode)
        context.userAgent = [...(context.userAgent || []), ["cfg/retry-mode", options.retryStrategy.mode]];
    return options.retryStrategy.retry(next, args);
};
export const retryMiddlewareOptions = {
    name: "retryMiddleware",
    tags: ["RETRY"],
    step: "finalizeRequest",
    priority: "high",
    override: true,
};
export const getRetryPlugin = (options) => ({
    applyToStack: (clientStack) => {
        clientStack.add(retryMiddleware(options), retryMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicmV0cnlNaWRkbGV3YXJlLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsicmV0cnlNaWRkbGV3YXJlLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQWFBLE1BQU0sQ0FBQyxNQUFNLGVBQWUsR0FBRyxDQUFDLE9BQTRCLEVBQUUsRUFBRSxDQUFDLENBQy9ELElBQWtDLEVBQ2xDLE9BQWdDLEVBQ0YsRUFBRSxDQUFDLEtBQUssRUFDdEMsSUFBbUMsRUFDSyxFQUFFO0lBQzFDLElBQUksT0FBTyxFQUFFLGFBQWEsRUFBRSxJQUFJO1FBQzlCLE9BQU8sQ0FBQyxTQUFTLEdBQUcsQ0FBQyxHQUFHLENBQUMsT0FBTyxDQUFDLFNBQVMsSUFBSSxFQUFFLENBQUMsRUFBRSxDQUFDLGdCQUFnQixFQUFFLE9BQU8sQ0FBQyxhQUFhLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQztJQUNyRyxPQUFPLE9BQU8sQ0FBQyxhQUFhLENBQUMsS0FBSyxDQUFDLElBQUksRUFBRSxJQUFJLENBQUMsQ0FBQztBQUNqRCxDQUFDLENBQUM7QUFFRixNQUFNLENBQUMsTUFBTSxzQkFBc0IsR0FBcUQ7SUFDdEYsSUFBSSxFQUFFLGlCQUFpQjtJQUN2QixJQUFJLEVBQUUsQ0FBQyxPQUFPLENBQUM7SUFDZixJQUFJLEVBQUUsaUJBQWlCO0lBQ3ZCLFFBQVEsRUFBRSxNQUFNO0lBQ2hCLFFBQVEsRUFBRSxJQUFJO0NBQ2YsQ0FBQztBQUVGLE1BQU0sQ0FBQyxNQUFNLGNBQWMsR0FBRyxDQUFDLE9BQTRCLEVBQXVCLEVBQUUsQ0FBQyxDQUFDO0lBQ3BGLFlBQVksRUFBRSxDQUFDLFdBQVcsRUFBRSxFQUFFO1FBQzVCLFdBQVcsQ0FBQyxHQUFHLENBQUMsZUFBZSxDQUFDLE9BQU8sQ0FBQyxFQUFFLHNCQUFzQixDQUFDLENBQUM7SUFDcEUsQ0FBQztDQUNGLENBQUMsQ0FBQyJ9