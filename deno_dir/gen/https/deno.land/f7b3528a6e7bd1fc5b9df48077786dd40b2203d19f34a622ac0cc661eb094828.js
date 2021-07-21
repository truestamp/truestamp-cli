import { HttpRequest } from "../protocol-http/mod.ts";
import { parse as parseArn, validate as validateArn } from "../util-arn-parser/mod.ts";
import { bucketHostname } from "./bucketHostname.ts";
import { getPseudoRegion } from "./bucketHostnameUtils.ts";
export const bucketEndpointMiddleware = (options) => (next, context) => async (args) => {
    const { Bucket: bucketName } = args.input;
    let replaceBucketInPath = options.bucketEndpoint;
    const request = args.request;
    if (HttpRequest.isInstance(request)) {
        if (options.bucketEndpoint) {
            request.hostname = bucketName;
        }
        else if (validateArn(bucketName)) {
            const bucketArn = parseArn(bucketName);
            const clientRegion = getPseudoRegion(await options.region());
            const { partition, signingRegion = clientRegion } = (await options.regionInfoProvider(clientRegion)) || {};
            const useArnRegion = await options.useArnRegion();
            const { hostname, bucketEndpoint, signingRegion: modifiedSigningRegion, signingService, } = bucketHostname({
                bucketName: bucketArn,
                baseHostname: request.hostname,
                accelerateEndpoint: options.useAccelerateEndpoint,
                dualstackEndpoint: options.useDualstackEndpoint,
                pathStyleEndpoint: options.forcePathStyle,
                tlsCompatible: request.protocol === "https:",
                useArnRegion,
                clientPartition: partition,
                clientSigningRegion: signingRegion,
                clientRegion: clientRegion,
                isCustomEndpoint: options.isCustomEndpoint,
            });
            if (modifiedSigningRegion && modifiedSigningRegion !== signingRegion) {
                context["signing_region"] = modifiedSigningRegion;
            }
            if (signingService && signingService !== "s3") {
                context["signing_service"] = signingService;
            }
            request.hostname = hostname;
            replaceBucketInPath = bucketEndpoint;
        }
        else {
            const clientRegion = getPseudoRegion(await options.region());
            const { hostname, bucketEndpoint } = bucketHostname({
                bucketName,
                clientRegion,
                baseHostname: request.hostname,
                accelerateEndpoint: options.useAccelerateEndpoint,
                dualstackEndpoint: options.useDualstackEndpoint,
                pathStyleEndpoint: options.forcePathStyle,
                tlsCompatible: request.protocol === "https:",
                isCustomEndpoint: options.isCustomEndpoint,
            });
            request.hostname = hostname;
            replaceBucketInPath = bucketEndpoint;
        }
        if (replaceBucketInPath) {
            request.path = request.path.replace(/^(\/)?[^\/]+/, "");
            if (request.path === "") {
                request.path = "/";
            }
        }
    }
    return next({ ...args, request });
};
export const bucketEndpointMiddlewareOptions = {
    tags: ["BUCKET_ENDPOINT"],
    name: "bucketEndpointMiddleware",
    relation: "before",
    toMiddleware: "hostHeaderMiddleware",
    override: true,
};
export const getBucketEndpointPlugin = (options) => ({
    applyToStack: (clientStack) => {
        clientStack.addRelativeTo(bucketEndpointMiddleware(options), bucketEndpointMiddlewareOptions);
    },
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYnVja2V0RW5kcG9pbnRNaWRkbGV3YXJlLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiYnVja2V0RW5kcG9pbnRNaWRkbGV3YXJlLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSx5QkFBeUIsQ0FBQztBQVd0RCxPQUFPLEVBQUUsS0FBSyxJQUFJLFFBQVEsRUFBRSxRQUFRLElBQUksV0FBVyxFQUFFLE1BQU0sMkJBQTJCLENBQUM7QUFFdkYsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLHFCQUFxQixDQUFDO0FBQ3JELE9BQU8sRUFBRSxlQUFlLEVBQUUsTUFBTSwwQkFBMEIsQ0FBQztBQUczRCxNQUFNLENBQUMsTUFBTSx3QkFBd0IsR0FDbkMsQ0FBQyxPQUFxQyxFQUE2QixFQUFFLENBQ3JFLENBQ0UsSUFBK0IsRUFDL0IsT0FBZ0MsRUFDTCxFQUFFLENBQy9CLEtBQUssRUFBRSxJQUFnQyxFQUF1QyxFQUFFO0lBQzlFLE1BQU0sRUFBRSxNQUFNLEVBQUUsVUFBVSxFQUFFLEdBQUcsSUFBSSxDQUFDLEtBQTJCLENBQUM7SUFDaEUsSUFBSSxtQkFBbUIsR0FBRyxPQUFPLENBQUMsY0FBYyxDQUFDO0lBQ2pELE1BQU0sT0FBTyxHQUFHLElBQUksQ0FBQyxPQUFPLENBQUM7SUFDN0IsSUFBSSxXQUFXLENBQUMsVUFBVSxDQUFDLE9BQU8sQ0FBQyxFQUFFO1FBQ25DLElBQUksT0FBTyxDQUFDLGNBQWMsRUFBRTtZQUMxQixPQUFPLENBQUMsUUFBUSxHQUFHLFVBQVUsQ0FBQztTQUMvQjthQUFNLElBQUksV0FBVyxDQUFDLFVBQVUsQ0FBQyxFQUFFO1lBQ2xDLE1BQU0sU0FBUyxHQUFHLFFBQVEsQ0FBQyxVQUFVLENBQUMsQ0FBQztZQUN2QyxNQUFNLFlBQVksR0FBRyxlQUFlLENBQUMsTUFBTSxPQUFPLENBQUMsTUFBTSxFQUFFLENBQUMsQ0FBQztZQUM3RCxNQUFNLEVBQUUsU0FBUyxFQUFFLGFB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