import { ClientDefaultValues as __ClientDefaultValues } from "./runtimeConfig.ts";
import { resolveEndpointsConfig, resolveRegionConfig, } from "../config-resolver/mod.ts";
import { getContentLengthPlugin } from "../middleware-content-length/mod.ts";
import { getHostHeaderPlugin, resolveHostHeaderConfig, } from "../middleware-host-header/mod.ts";
import { getLoggerPlugin } from "../middleware-logger/mod.ts";
import { getRetryPlugin, resolveRetryConfig } from "../middleware-retry/mod.ts";
import { resolveStsAuthConfig } from "../middleware-sdk-sts/mod.ts";
import { getUserAgentPlugin, resolveUserAgentConfig, } from "../middleware-user-agent/mod.ts";
import { Client as __Client, } from "../smithy-client/mod.ts";
export class STSClient extends __Client {
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
        let _config_5 = resolveStsAuthConfig(_config_4, STSClient);
        let _config_6 = resolveUserAgentConfig(_config_5);
        super(_config_6);
        this.config = _config_6;
        this.middlewareStack.use(getRetryPlugin(this.config));
        this.middlewareStack.use(getContentLengthPlugin(this.config));
        this.middlewareStack.use(getHostHeaderPlugin(this.config));
        this.middlewareStack.use(getLoggerPlugin(this.config));
        this.middlewareStack.use(getUserAgentPlugin(this.config));
    }
    destroy() {
        super.destroy();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiU1RTQ2xpZW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiU1RTQ2xpZW50LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQWNBLE9BQU8sRUFBRSxtQkFBbUIsSUFBSSxxQkFBcUIsRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQ2xGLE9BQU8sRUFLTCxzQkFBc0IsRUFDdEIsbUJBQW1CLEdBQ3BCLE1BQU0sMkJBQTJCLENBQUM7QUFDbkMsT0FBTyxFQUFFLHNCQUFzQixFQUFFLE1BQU0scUNBQXFDLENBQUM7QUFDN0UsT0FBTyxFQUdMLG1CQUFtQixFQUNuQix1QkFBdUIsR0FDeEIsTUFBTSxrQ0FBa0MsQ0FBQztBQUMxQyxPQUFPLEVBQUUsZUFBZSxFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFDOUQsT0FBTyxFQUF5QyxjQUFjLEVBQUUsa0JBQWtCLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQUN2SCxPQUFPLEVBQTZDLG9CQUFvQixFQUFFLE1BQU0sOEJBQThCLENBQUM7QUFDL0csT0FBTyxFQUdMLGtCQUFrQixFQUNsQixzQkFBc0IsR0FDdkIsTUFBTSxpQ0FBaUMsQ0FBQztBQUV6QyxPQUFPLEVBQ0wsTUFBTSxJQUFJLFFBQVEsR0FHbkIsTUFBTSx5QkFBeUIsQ0FBQztBQXFLakMsTUFBTSxPQUFPLFNBQVUsU0FBUSxRQUs5QjtJQUlVLE1BQU0sQ0FBMEI7SUFFekMsWUFBWSxhQUE4QjtRQUN4QyxJQUFJLFNBQVMsR0FBRztZQUNkLEdBQUcscUJBQXFCO1lBQ3hCLEdBQUcsYUFBYTtTQUNqQixDQUFDO1FBQ0YsSUFBSSxTQUFTLEdBQUcsbUJBQW1CLENBQUMsU0FBUyxDQUFDLENBQUM7UUFDL0MsSUFBSSxTQUFTLEdBQUcsc0JBQXNCLENBQUMsU0FBUyxDQUFDLENBQUM7UUFDbEQsSUFBSSxTQUFTLEdBQUcsa0JBQWtCLENBQUMsU0FBUyxDQUFDLENBQUM7UUFDOUMsSUFBSSxTQUFTLEdBQUcsdUJBQXVCLENBQUMsU0FBUyxDQUFDLENBQUM7UUFDbkQsSUFBSSxTQUFTLEdBQUcsb0JBQW9CLENBQUMsU0FBUyxFQUFFLFNBQVMsQ0FBQyxDQUFDO1FBQzNELElBQUksU0FBUyxHQUFHLHNCQUFzQixDQUFDLFNBQVMsQ0FBQyxDQUFDO1FBQ2xELEtBQUssQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNqQixJQUFJLENBQUMsTUFBTSxHQUFHLFNBQVMsQ0FBQztRQUN4QixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxjQUFjLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUM7UUFDdEQsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsc0JBQXNCLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUM7UUFDOUQsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsbUJBQW1CLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUM7UUFDM0QsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsZUFBZSxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO1FBQ3ZELElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLGtCQUFrQixDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO0lBQzVELENBQUM7SUFPRCxPQUFPO1FBQ0wsS0FBSyxDQUFDLE9BQU8sRUFBRSxDQUFDO0lBQ2xCLENBQUM7Q0FDRiJ9