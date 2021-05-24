const packageInfo = { version: "3.15.0" };
import { decorateDefaultCredentialProvider } from "./defaultStsRoleAssumers.ts";
import { NODE_REGION_CONFIG_FILE_OPTIONS, NODE_REGION_CONFIG_OPTIONS } from "../config-resolver/mod.ts";
import { defaultProvider as credentialDefaultProvider } from "../credential-provider-node/mod.ts";
import { Hash } from "https://jspm.dev/@aws-sdk/hash-node";
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
    maxAttempts: loadNodeConfig(NODE_MAX_ATTEMPT_CONFIG_OPTIONS),
    region: loadNodeConfig(NODE_REGION_CONFIG_OPTIONS, NODE_REGION_CONFIG_FILE_OPTIONS),
    requestHandler: new FetchHttpHandler(),
    sha256: Hash.bind(null, "sha256"),
    streamCollector,
    utf8Decoder: fromUtf8,
    utf8Encoder: toUtf8,
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicnVudGltZUNvbmZpZy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbInJ1bnRpbWVDb25maWcudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsTUFBTSxXQUFXLEdBQUcsRUFBRSxPQUFPLEVBQUUsUUFBUSxFQUFFLENBQUM7QUFFMUMsT0FBTyxFQUFFLGlDQUFpQyxFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFDaEYsT0FBTyxFQUFFLCtCQUErQixFQUFFLDBCQUEwQixFQUFFLE1BQU0sMkJBQTJCLENBQUM7QUFDeEcsT0FBTyxFQUFFLGVBQWUsSUFBSSx5QkFBeUIsRUFBRSxNQUFNLG9DQUFvQyxDQUFDO0FBQ2xHLE9BQU8sRUFBRSxJQUFJLEVBQUUsTUFBTSxxQ0FBcUMsQ0FBQztBQUMzRCxPQUFPLEVBQUUsK0JBQStCLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQUM3RSxPQUFPLEVBQUUsVUFBVSxJQUFJLGNBQWMsRUFBRSxNQUFNLGdDQUFnQyxDQUFDO0FBQzlFLE9BQU8sRUFBRSxnQkFBZ0IsRUFBRSxlQUFlLEVBQUUsTUFBTSw4QkFBOEIsQ0FBQztBQUNqRixPQUFPLEVBQUUsVUFBVSxFQUFFLFFBQVEsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBQ2xFLE9BQU8sRUFBRSxtQkFBbUIsRUFBRSxNQUFNLGlDQUFpQyxDQUFDO0FBQ3RFLE9BQU8sRUFBRSxnQkFBZ0IsRUFBRSxNQUFNLGdDQUFnQyxDQUFDO0FBQ2xFLE9BQU8sRUFBRSxRQUFRLEVBQUUsTUFBTSxFQUFFLE1BQU0sMEJBQTBCLENBQUM7QUFFNUQsT0FBTyxFQUFFLGtCQUFrQixFQUFFLE1BQU0sMkJBQTJCLENBQUM7QUFLL0QsTUFBTSxDQUFDLE1BQU0sbUJBQW1CLEdBQTZCO0lBQzNELEdBQUcsa0JBQWtCO0lBQ3JCLE9BQU8sRUFBRSxNQUFNO0lBQ2YsYUFBYSxFQUFFLFVBQVU7SUFDekIsYUFBYSxFQUFFLFFBQVE7SUFDdkIsaUJBQWlCLEVBQUUsbUJBQW1CO0lBQ3RDLHlCQUF5QixFQUFFLGlDQUFpQyxDQUFDLHlCQUF5QixDQUFDO0lBQ3ZGLHdCQUF3QixFQUFFLGdCQUFnQixDQUFDO1FBQ3pDLFNBQVMsRUFBRSxrQkFBa0IsQ0FBQyxTQUFTO1FBQ3ZDLGFBQWEsRUFBRSxXQUFXLENBQUMsT0FBTztLQUNuQyxDQUFDO0lBQ0YsV0FBVyxFQUFFLGNBQWMsQ0FBQywrQkFBK0IsQ0FBQztJQUM1RCxNQUFNLEVBQUUsY0FBYyxDQUFDLDBCQUEwQixFQUFFLCtCQUErQixDQUFDO0lBQ25GLGNBQWMsRUFBRSxJQUFJLGdCQUFnQixFQUFFO0lBQ3RDLE1BQU0sRUFBRSxJQUFJLENBQUMsSUFBSSxDQUFDLElBQUksRUFBRSxRQUFRLENBQUM7SUFDakMsZUFBZTtJQUNmLFdBQVcsRUFBRSxRQUFRO0lBQ3JCLFdBQVcsRUFBRSxNQUFNO0NBQ3BCLENBQUMifQ==