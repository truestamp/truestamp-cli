export const resolveCustomEndpointsConfig = (input) => ({
    ...input,
    tls: input.tls ?? true,
    endpoint: normalizeEndpoint(input),
    isCustomEndpoint: true,
});
const normalizeEndpoint = (input) => {
    const { endpoint, urlParser } = input;
    if (typeof endpoint === "string") {
        const promisified = Promise.resolve(urlParser(endpoint));
        return () => promisified;
    }
    else if (typeof endpoint === "object") {
        const promisified = Promise.resolve(endpoint);
        return () => promisified;
    }
    return endpoint;
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiQ3VzdG9tRW5kcG9pbnRzQ29uZmlnLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiQ3VzdG9tRW5kcG9pbnRzQ29uZmlnLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQXVCQSxNQUFNLENBQUMsTUFBTSw0QkFBNEIsR0FBRyxDQUMxQyxLQUEwRCxFQUN2QixFQUFFLENBQUMsQ0FBQztJQUN2QyxHQUFHLEtBQUs7SUFDUixHQUFHLEVBQUUsS0FBSyxDQUFDLEdBQUcsSUFBSSxJQUFJO0lBQ3RCLFFBQVEsRUFBRSxpQkFBaUIsQ0FBQyxLQUFLLENBQUM7SUFDbEMsZ0JBQWdCLEVBQUUsSUFBSTtDQUN2QixDQUFDLENBQUM7QUFFSCxNQUFNLGlCQUFpQixHQUFHLENBQUMsS0FBc0QsRUFBc0IsRUFBRTtJQUN2RyxNQUFNLEVBQUUsUUFBUSxFQUFFLFNBQVMsRUFBRSxHQUFHLEtBQUssQ0FBQztJQUN0QyxJQUFJLE9BQU8sUUFBUSxLQUFLLFFBQVEsRUFBRTtRQUNoQyxNQUFNLFdBQVcsR0FBRyxPQUFPLENBQUMsT0FBTyxDQUFDLFNBQVMsQ0FBQyxRQUFRLENBQUMsQ0FBQyxDQUFDO1FBQ3pELE9BQU8sR0FBRyxFQUFFLENBQUMsV0FBVyxDQUFDO0tBQzFCO1NBQU0sSUFBSSxPQUFPLFFBQVEsS0FBSyxRQUFRLEVBQUU7UUFDdkMsTUFBTSxXQUFXLEdBQUcsT0FBTyxDQUFDLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQztRQUM5QyxPQUFPLEdBQUcsRUFBRSxDQUFDLFdBQVcsQ0FBQztLQUMxQjtJQUNELE9BQU8sUUFBUSxDQUFDO0FBQ2xCLENBQUMsQ0FBQyJ9