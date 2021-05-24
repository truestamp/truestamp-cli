import { ClientDefaultValues as __ClientDefaultValues } from "./runtimeConfig.ts";
import { resolveEndpointsConfig, resolveRegionConfig, } from "../config-resolver/mod.ts";
import { resolveEventStreamSerdeConfig, } from "../eventstream-serde-config-resolver/mod.ts";
import { resolveBucketEndpointConfig, } from "../middleware-bucket-endpoint/mod.ts";
import { getContentLengthPlugin } from "../middleware-content-length/mod.ts";
import { getAddExpectContinuePlugin } from "../middleware-expect-continue/mod.ts";
import { getHostHeaderPlugin, resolveHostHeaderConfig, } from "../middleware-host-header/mod.ts";
import { getLoggerPlugin } from "../middleware-logger/mod.ts";
import { getRetryPlugin, resolveRetryConfig } from "../middleware-retry/mod.ts";
import { getUseRegionalEndpointPlugin, getValidateBucketNamePlugin } from "../middleware-sdk-s3/mod.ts";
import { getAwsAuthPlugin, resolveAwsAuthConfig, } from "../middleware-signing/mod.ts";
import { getUserAgentPlugin, resolveUserAgentConfig, } from "../middleware-user-agent/mod.ts";
import { Client as __Client, } from "../smithy-client/mod.ts";
export class S3Client extends __Client {
    config;
    constructor(configuration) {
        let _config_0 = {
            ...__ClientDefaultValues,
            ...configuration,
        };
        let _config_1 = resolveRegionConfig(_config_0);
        let _config_2 = resolveEndpointsConfig(_config_1);
        let _config_3 = resolveRetryConfig(_config_2);
        let _config_4 = resolveHostHeaderConfig(_config_3);
        let _config_5 = resolveAwsAuthConfig(_config_4);
        let _config_6 = resolveBucketEndpointConfig(_config_5);
        let _config_7 = resolveUserAgentConfig(_config_6);
        let _config_8 = resolveEventStreamSerdeConfig(_config_7);
        super(_config_8);
        this.config = _config_8;
        this.middlewareStack.use(getRetryPlugin(this.config));
        this.middlewareStack.use(getContentLengthPlugin(this.config));
        this.middlewareStack.use(getHostHeaderPlugin(this.config));
        this.middlewareStack.use(getLoggerPlugin(this.config));
        this.middlewareStack.use(getAwsAuthPlugin(this.config));
        this.middlewareStack.use(getValidateBucketNamePlugin(this.config));
        this.middlewareStack.use(getUseRegionalEndpointPlugin(this.config));
        this.middlewareStack.use(getAddExpectContinuePlugin(this.config));
        this.middlewareStack.use(getUserAgentPlugin(this.config));
    }
    destroy() {
        super.destroy();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiUzNDbGllbnQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJTM0NsaWVudC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFxUEEsT0FBTyxFQUFFLG1CQUFtQixJQUFJLHFCQUFxQixFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFDbEYsT0FBTyxFQUtMLHNCQUFzQixFQUN0QixtQkFBbUIsR0FDcEIsTUFBTSwyQkFBMkIsQ0FBQztBQUNuQyxPQUFPLEVBR0wsNkJBQTZCLEdBQzlCLE1BQU0sNkNBQTZDLENBQUM7QUFDckQsT0FBTyxFQUdMLDJCQUEyQixHQUM1QixNQUFNLHNDQUFzQyxDQUFDO0FBQzlDLE9BQU8sRUFBRSxzQkFBc0IsRUFBRSxNQUFNLHFDQUFxQyxDQUFDO0FBQzdFLE9BQU8sRUFBRSwwQkFBMEIsRUFBRSxNQUFNLHNDQUFzQyxDQUFDO0FBQ2xGLE9BQU8sRUFHTCxtQkFBbUIsRUFDbkIsdUJBQXVCLEdBQ3hCLE1BQU0sa0NBQWtDLENBQUM7QUFDMUMsT0FBTyxFQUFFLGVBQWUsRUFBRSxNQUFNLDZCQUE2QixDQUFDO0FBQzlELE9BQU8sRUFBeUMsY0FBYyxFQUFFLGtCQUFrQixFQUFFLE1BQU0sNEJBQTRCLENBQUM7QUFDdkgsT0FBTyxFQUFFLDRCQUE0QixFQUFFLDJCQUEyQixFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFDeEcsT0FBTyxFQUdMLGdCQUFnQixFQUNoQixvQkFBb0IsR0FDckIsTUFBTSw4QkFBOEIsQ0FBQztBQUN0QyxPQUFPLEVBR0wsa0JBQWtCLEVBQ2xCLHNCQUFzQixHQUN2QixNQUFNLGlDQUFpQyxDQUFDO0FBRXpDLE9BQU8sRUFDTCxNQUFNLElBQUksUUFBUSxHQUduQixNQUFNLHlCQUF5QixDQUFDO0FBMFdqQyxNQUFNLE9BQU8sUUFBUyxTQUFRLFFBSzdCO0lBSVUsTUFBTSxDQUF5QjtJQUV4QyxZQUFZLGFBQTZCO1FBQ3ZDLElBQUksU0FBUyxHQUFHO1lBQ2QsR0FBRyxxQkFBcUI7WUFDeEIsR0FBRyxhQUFhO1NBQ2pCLENBQUM7UUFDRixJQUFJLFNBQVMsR0FBRyxtQkFBbUIsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUMvQyxJQUFJLFNBQVMsR0FBRyxzQkFBc0IsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNsRCxJQUFJLFNBQVMsR0FBRyxrQkFBa0IsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUM5QyxJQUFJLFNBQVMsR0FBRyx1QkFBdUIsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNuRCxJQUFJLFNBQVMsR0FBRyxvQkFBb0IsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNoRCxJQUFJLFNBQVMsR0FBRywyQkFBMkIsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUN2RCxJQUFJLFNBQVMsR0FBRyxzQkFBc0IsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNsRCxJQUFJLFNBQVMsR0FBRyw2QkFBNkIsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUN6RCxLQUFLLENBQUMsU0FBUyxDQUFDLENBQUM7UUFDakIsSUFBSSxDQUFDLE1BQU0sR0FBRyxTQUFTLENBQUM7UUFDeEIsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO1FBQ3RELElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHNCQUFzQixDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO1FBQzlELElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLG1CQUFtQixDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO1FBQzNELElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLGVBQWUsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztRQUN2RCxJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxnQkFBZ0IsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztRQUN4RCxJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQywyQkFBMkIsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztRQUNuRSxJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyw0QkFBNEIsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztRQUNwRSxJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQywwQkFBMEIsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztRQUNsRSxJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxrQkFBa0IsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztJQUM1RCxDQUFDO0lBT0QsT0FBTztRQUNMLEtBQUssQ0FBQyxPQUFPLEVBQUUsQ0FBQztJQUNsQixDQUFDO0NBQ0YifQ==