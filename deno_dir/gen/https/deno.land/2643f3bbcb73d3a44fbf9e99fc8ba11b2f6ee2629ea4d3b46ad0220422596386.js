export const resolveEndpointsConfig = (input) => ({
    ...input,
    tls: input.tls ?? true,
    endpoint: input.endpoint ? normalizeEndpoint(input) : () => getEndPointFromRegion(input),
    isCustomEndpoint: input.endpoint ? true : false,
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
const getEndPointFromRegion = async (input) => {
    const { tls = true } = input;
    const region = await input.region();
    const dnsHostRegex = new RegExp(/^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9])$/);
    if (!dnsHostRegex.test(region)) {
        throw new Error("Invalid region in client config");
    }
    const { hostname } = (await input.regionInfoProvider(region)) ?? {};
    if (!hostname) {
        throw new Error("Cannot resolve hostname from client config");
    }
    return input.urlParser(`${tls ? "https:" : "http:"}//${hostname}`);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiRW5kcG9pbnRzQ29uZmlnLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiRW5kcG9pbnRzQ29uZmlnLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQXlCQSxNQUFNLENBQUMsTUFBTSxzQkFBc0IsR0FBRyxDQUNwQyxLQUFvRCxFQUN2QixFQUFFLENBQUMsQ0FBQztJQUNqQyxHQUFHLEtBQUs7SUFDUixHQUFHLEVBQUUsS0FBSyxDQUFDLEdBQUcsSUFBSSxJQUFJO0lBQ3RCLFFBQVEsRUFBRSxLQUFLLENBQUMsUUFBUSxDQUFDLENBQUMsQ0FBQyxpQkFBaUIsQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLENBQUMsR0FBRyxFQUFFLENBQUMscUJBQXFCLENBQUMsS0FBSyxDQUFDO0lBQ3hGLGdCQUFnQixFQUFFLEtBQUssQ0FBQyxRQUFRLENBQUMsQ0FBQyxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsS0FBSztDQUNoRCxDQUFDLENBQUM7QUFFSCxNQUFNLGlCQUFpQixHQUFHLENBQUMsS0FBZ0QsRUFBc0IsRUFBRTtJQUNqRyxNQUFNLEVBQUUsUUFBUSxFQUFFLFNBQVMsRUFBRSxHQUFHLEtBQUssQ0FBQztJQUN0QyxJQUFJLE9BQU8sUUFBUSxLQUFLLFFBQVEsRUFBRTtRQUNoQyxNQUFNLFdBQVcsR0FBRyxPQUFPLENBQUMsT0FBTyxDQUFDLFNBQVMsQ0FBQyxRQUFRLENBQUMsQ0FBQyxDQUFDO1FBQ3pELE9BQU8sR0FBRyxFQUFFLENBQUMsV0FBVyxDQUFDO0tBQzFCO1NBQU0sSUFBSSxPQUFPLFFBQVEsS0FBSyxRQUFRLEVBQUU7UUFDdkMsTUFBTSxXQUFXLEdBQUcsT0FBTyxDQUFDLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQztRQUM5QyxPQUFPLEdBQUcsRUFBRSxDQUFDLFdBQVcsQ0FBQztLQUMxQjtJQUNELE9BQU8sUUFBUyxDQUFDO0FBQ25CLENBQUMsQ0FBQztBQUVGLE1BQU0scUJBQXFCLEdBQUcsS0FBSyxFQUFFLEtBQWdELEVBQUUsRUFBRTtJQUN2RixNQUFNLEVBQUUsR0FBRyxHQUFHLElBQUksRUFBRSxHQUFHLEtBQUssQ0FBQztJQUM3QixNQUFNLE1BQU0sR0FBRyxNQUFNLEtBQUssQ0FBQyxNQUFNLEVBQUUsQ0FBQztJQUVwQyxNQUFNLFlBQVksR0FBRyxJQUFJLE1BQU0sQ0FBQywwREFBMEQsQ0FBQyxDQUFDO0lBQzVGLElBQUksQ0FBQyxZQUFZLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxFQUFFO1FBQzlCLE1BQU0sSUFBSSxLQUFLLENBQUMsaUNBQWlDLENBQUMsQ0FBQztLQUNwRDtJQUVELE1BQU0sRUFBRSxRQUFRLEVBQUUsR0FBRyxDQUFDLE1BQU0sS0FBSyxDQUFDLGtCQUFrQixDQUFDLE1BQU0sQ0FBQyxDQUFDLElBQUksRUFBRSxDQUFDO0lBQ3BFLElBQUksQ0FBQyxRQUFRLEVBQUU7UUFDYixNQUFNLElBQUksS0FBSyxDQUFDLDRDQUE0QyxDQUFDLENBQUM7S0FDL0Q7SUFFRCxPQUFPLEtBQUssQ0FBQyxTQUFTLENBQUMsR0FBRyxHQUFHLENBQUMsQ0FBQyxDQUFDLFFBQVEsQ0FBQyxDQUFDLENBQUMsT0FBTyxLQUFLLFFBQVEsRUFBRSxDQUFDLENBQUM7QUFDckUsQ0FBQyxDQUFDIn0=