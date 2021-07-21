const packageInfo = { version: "3.17.0" };
import { decorateDefaultCredentialProvider } from "../client-sts/mod.ts";
import { NODE_REGION_CONFIG_FILE_OPTIONS, NODE_REGION_CONFIG_OPTIONS } from "../config-resolver/mod.ts";
import { defaultProvider as credentialDefaultProvider } from "../credential-provider-node/mod.ts";
import { eventStreamSerdeProvider } from "../eventstream-serde-browser/mod.ts";
import { Hash } from "https://jspm.dev/@aws-sdk/hash-node";
import { blobHasher as streamHasher } from "../hash-blob-browser/mod.ts";
import { NODE_USE_ARN_REGION_CONFIG_OPTIONS } from "../middleware-bucket-endpoint/mod.ts";
import { NODE_MAX_ATTEMPT_CONFIG_OPTIONS } from "../middleware-retry/mod.ts";
import { loadConfig as loadNodeConfig } from "../node-config-provider/mod.ts";
import { FetchHttpHandler, streamCollector } from "../fetch-http-handler/mod.ts";
import { fromBase64, toBase64 } from "../util-base64-node/mod.ts";
import { calculateBodyLength } from "../util-body-length-node/mod.ts";
import { defaultUserAgent } from "../util-user-agent-node/mod.ts";
import { fromUtf8, toUtf8 } from "../util-utf8-node/mod.ts";
import { ClientSharedValues } from "./runtimeConfig.shared.ts";
export const ClientDefaultValues = {
    ...ClientSharedValues,
    runtime: "deno",
    base64Decoder: fromBase64,
    base64Encoder: toBase64,
    bodyLengthChecker: calculateBodyLength,
    credentialDefaultProvider: decorateDefaultCredentialProvider(credentialDefaultProvider),
    defaultUserAgentProvider: defaultUserAgent({
        serviceId: ClientSharedValues.serviceId,
        clientVersion: packageInfo.version,
    }),
    eventStreamSerdeProvider,
    maxAttempts: loadNodeConfig(NODE_MAX_ATTEMPT_CONFIG_OPTIONS),
    md5: Hash.bind(null, "md5"),
    region: loadNodeConfig(NODE_REGION_CONFIG_OPTIONS, NODE_REGION_CONFIG_FILE_OPTIONS),
    requestHandler: new FetchHttpHandler(),
    sha256: Hash.bind(null, "sha256"),
    streamCollector,
    streamHasher,
    useArnRegion: loadNodeConfig(NODE_USE_ARN_REGION_CONFIG_OPTIONS),
    utf8Decoder: fromUtf8,
    utf8Encoder: toUtf8,
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicnVudGltZUNvbmZpZy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbInJ1bnRpbWVDb25maWcudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsTUFBTSxXQUFXLEdBQUcsRUFBRSxPQUFPLEVBQUUsUUFBUSxFQUFFLENBQUM7QUFFMUMsT0FBTyxFQUFFLGlDQUFpQyxFQUFFLE1BQU0sc0JBQXNCLENBQUM7QUFDekUsT0FBTyxFQUFFLCtCQUErQixFQUFFLDBCQUEwQixFQUFFLE1BQU0sMkJBQTJCLENBQUM7QUFDeEcsT0FBTyxFQUFFLGVBQWUsSUFBSSx5QkFBeUIsRUFBRSxNQUFNLG9DQUFvQyxDQUFDO0FBQ2xHLE9BQU8sRUFBRSx3QkFBd0IsRUFBRSxNQUFNLHFDQUFxQyxDQUFDO0FBQy9FLE9BQU8sRUFBRSxJQUFJLEVBQUUsTUFBTSxxQ0FBcUMsQ0FBQztBQUMzRCxPQUFPLEVBQUUsVUFBVSxJQUFJLFlBQVksRUFBRSxNQUFNLDZCQUE2QixDQUFDO0FBQ3pFLE9BQU8sRUFBRSxrQ0FBa0MsRUFBRSxNQUFNLHNDQUFzQyxDQUFDO0FBQzFGLE9BQU8sRUFBRSwrQkFBK0IsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBQzdFLE9BQU8sRUFBRSxVQUFVLElBQUksY0FBYyxFQUFFLE1BQU0sZ0NBQWdDLENBQUM7QUFDOUUsT0FBTyxFQUFFLGdCQUFnQixFQUFFLGVBQWUsRUFBRSxNQUFNLDhCQUE4QixDQUFDO0FBRWpGLE9BQU8sRUFBRSxVQUFVLEVBQUUsUUFBUSxFQUFFLE1BQU0sNEJBQTRCLENBQUM7QUFDbEUsT0FBTyxFQUFFLG1CQUFtQixFQUFFLE1BQU0saUNBQWlDLENBQUM7QUFDdEUsT0FBTyxFQUFFLGdCQUFnQixFQUFFLE1BQU0sZ0NBQWdDLENBQUM7QUFDbEUsT0FBTyxFQUFFLFFBQVEsRUFBRSxNQUFNLEVBQUUsTUFBTSwwQkFBMEIsQ0FBQztBQUU1RCxPQUFPLEVBQUUsa0JBQWtCLEVBQUUsTUFBTSwyQkFBMkIsQ0FBQztBQUsvRCxNQUFNLENBQUMsTUFBTSxtQkFBbUIsR0FBNkI7SUFDM0QsR0FBRyxrQkFBa0I7SUFDckIsT0FBTyxFQUFFLE1BQU07SUFDZixhQUFhLEVBQUUsVUFBVTtJQUN6QixhQUFhLEVBQUUsUUFBUTtJQUN2QixpQkFBaUIsRUFBRSxtQkFBbUI7SUFDdEMseUJBQXlCLEVBQUUsaUNBQWlDLENBQUMseUJBQXlCLENBQUM7SUFDdkYsd0JBQXdCLEVBQUUsZ0JBQWdCLENBQUM7UUFDekMsU0FBUyxFQUFFLGtCQUFrQixDQUFDLFNBQVM7UUFDdkMsYUFBYSxFQUFFLFdBQVcsQ0FBQyxPQUFPO0tBQ25DLENBQUM7SUFDRix3QkFBd0I7SUFDeEIsV0FBVyxFQUFFLGNBQWMsQ0FBQywrQkFBK0IsQ0FBQztJQUM1RCxHQUFHLEVBQUUsSUFBSSxDQUFDLElBQUksQ0FBQyxJQUFJLEVBQUUsS0FBSyxDQUFDO0lBQzNCLE1BQU0sRUFBRSxjQUFjLENBQUMsMEJBQTBCLEVBQUUsK0JBQStCLENBQUM7SUFDbkYsY0FBYyxFQUFFLElBQUksZ0JBQWdCLEVBQUU7SUFDdEMsTUFBTSxFQUFFLElBQUksQ0FBQyxJQUFJLENBQUMsSUFBSSxFQUFFLFFBQVEsQ0FBQztJQUNqQyxlQUFlO0lBQ2YsWUFBWTtJQUNaLFlBQVksRUFBRSxjQUFjLENBQUMsa0NBQWtDLENBQUM7SUFDaEUsV0FBVyxFQUFFLFFBQVE7SUFDckIsV0FBVyxFQUFFLE1BQU07Q0FDcEIsQ0FBQyJ9