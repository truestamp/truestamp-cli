const packageInfo = { version: "3.22.0" };
import { NODE_REGION_CONFIG_FILE_OPTIONS, NODE_REGION_CONFIG_OPTIONS } from "../config-resolver/mod.ts";
import { Hash } from "https://jspm.dev/@aws-sdk/hash-node";
import { NODE_MAX_ATTEMPT_CONFIG_OPTIONS, NODE_RETRY_MODE_CONFIG_OPTIONS } from "../middleware-retry/mod.ts";
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
    defaultUserAgentProvider: defaultUserAgent({
        serviceId: ClientSharedValues.serviceId,
        clientVersion: packageInfo.version,
    }),
    maxAttempts: loadNodeConfig(NODE_MAX_ATTEMPT_CONFIG_OPTIONS),
    region: loadNodeConfig(NODE_REGION_CONFIG_OPTIONS, NODE_REGION_CONFIG_FILE_OPTIONS),
    requestHandler: new FetchHttpHandler(),
    retryModeProvider: loadNodeConfig(NODE_RETRY_MODE_CONFIG_OPTIONS),
    sha256: Hash.bind(null, "sha256"),
    streamCollector,
    utf8Decoder: fromUtf8,
    utf8Encoder: toUtf8,
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicnVudGltZUNvbmZpZy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbInJ1bnRpbWVDb25maWcudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsTUFBTSxXQUFXLEdBQUcsRUFBRSxPQUFPLEVBQUUsUUFBUSxFQUFFLENBQUM7QUFFMUMsT0FBTyxFQUFFLCtCQUErQixFQUFFLDBCQUEwQixFQUFFLE1BQU0sMkJBQTJCLENBQUM7QUFDeEcsT0FBTyxFQUFFLElBQUksRUFBRSxNQUFNLHFDQUFxQyxDQUFDO0FBQzNELE9BQU8sRUFBRSwrQkFBK0IsRUFBRSw4QkFBOEIsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBQzdHLE9BQU8sRUFBRSxVQUFVLElBQUksY0FBYyxFQUFFLE1BQU0sZ0NBQWdDLENBQUM7QUFDOUUsT0FBTyxFQUFFLGdCQUFnQixFQUFFLGVBQWUsRUFBRSxNQUFNLDhCQUE4QixDQUFDO0FBQ2pGLE9BQU8sRUFBRSxVQUFVLEVBQUUsUUFBUSxFQUFFLE1BQU0sNEJBQTRCLENBQUM7QUFDbEUsT0FBTyxFQUFFLG1CQUFtQixFQUFFLE1BQU0saUNBQWlDLENBQUM7QUFDdEUsT0FBTyxFQUFFLGdCQUFnQixFQUFFLE1BQU0sZ0NBQWdDLENBQUM7QUFDbEUsT0FBTyxFQUFFLFFBQVEsRUFBRSxNQUFNLEVBQUUsTUFBTSwwQkFBMEIsQ0FBQztBQUU1RCxPQUFPLEVBQUUsa0JBQWtCLEVBQUUsTUFBTSwyQkFBMkIsQ0FBQztBQUsvRCxNQUFNLENBQUMsTUFBTSxtQkFBbUIsR0FBNkI7SUFDM0QsR0FBRyxrQkFBa0I7SUFDckIsT0FBTyxFQUFFLE1BQU07SUFDZixhQUFhLEVBQUUsVUFBVTtJQUN6QixhQUFhLEVBQUUsUUFBUTtJQUN2QixpQkFBaUIsRUFBRSxtQkFBbUI7SUFDdEMsd0JBQXdCLEVBQUUsZ0JBQWdCLENBQUM7UUFDekMsU0FBUyxFQUFFLGtCQUFrQixDQUFDLFNBQVM7UUFDdkMsYUFBYSxFQUFFLFdBQVcsQ0FBQyxPQUFPO0tBQ25DLENBQUM7SUFDRixXQUFXLEVBQUUsY0FBYyxDQUFDLCtCQUErQixDQUFDO0lBQzVELE1BQU0sRUFBRSxjQUFjLENBQUMsMEJBQTBCLEVBQUUsK0JBQStCLENBQUM7SUFDbkYsY0FBYyxFQUFFLElBQUksZ0JBQWdCLEVBQUU7SUFDdEMsaUJBQWlCLEVBQUUsY0FBYyxDQUFDLDhCQUE4QixDQUFDO0lBQ2pFLE1BQU0sRUFBRSxJQUFJLENBQUMsSUFBSSxDQUFDLElBQUksRUFBRSxRQUFRLENBQUM7SUFDakMsZUFBZTtJQUNmLFdBQVcsRUFBRSxRQUFRO0lBQ3JCLFdBQVcsRUFBRSxNQUFNO0NBQ3BCLENBQUMifQ==