import { ClientDefaultValues as __ClientDefaultValues } from "./runtimeConfig.ts";
import { resolveEndpointsConfig, resolveRegionConfig, } from "../config-resolver/mod.ts";
import { getContentLengthPlugin } from "../middleware-content-length/mod.ts";
import { getHostHeaderPlugin, resolveHostHeaderConfig, } from "../middleware-host-header/mod.ts";
import { getLoggerPlugin } from "../middleware-logger/mod.ts";
import { getRetryPlugin, resolveRetryConfig } from "../middleware-retry/mod.ts";
import { getUserAgentPlugin, resolveUserAgentConfig, } from "../middleware-user-agent/mod.ts";
import { Client as __Client, } from "../smithy-client/mod.ts";
export class SSOClient extends __Client {
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
        let _config_5 = resolveUserAgentConfig(_config_4);
        super(_config_5);
        this.config = _config_5;
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiU1NPQ2xpZW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiU1NPQ2xpZW50LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUlBLE9BQU8sRUFBRSxtQkFBbUIsSUFBSSxxQkFBcUIsRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQ2xGLE9BQU8sRUFLTCxzQkFBc0IsRUFDdEIsbUJBQW1CLEdBQ3BCLE1BQU0sMkJBQTJCLENBQUM7QUFDbkMsT0FBTyxFQUFFLHNCQUFzQixFQUFFLE1BQU0scUNBQXFDLENBQUM7QUFDN0UsT0FBTyxFQUdMLG1CQUFtQixFQUNuQix1QkFBdUIsR0FDeEIsTUFBTSxrQ0FBa0MsQ0FBQztBQUMxQyxPQUFPLEVBQUUsZUFBZSxFQUFFLE1BQU0sNkJBQTZCLENBQUM7QUFDOUQsT0FBTyxFQUF5QyxjQUFjLEVBQUUsa0JBQWtCLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQUN2SCxPQUFPLEVBR0wsa0JBQWtCLEVBQ2xCLHNCQUFzQixHQUN2QixNQUFNLGlDQUFpQyxDQUFDO0FBRXpDLE9BQU8sRUFDTCxNQUFNLElBQUksUUFBUSxHQUduQixNQUFNLHlCQUF5QixDQUFDO0FBMEtqQyxNQUFNLE9BQU8sU0FBVSxTQUFRLFFBSzlCO0lBSVUsTUFBTSxDQUEwQjtJQUV6QyxZQUFZLGFBQThCO1FBQ3hDLElBQUksU0FBUyxHQUFHO1lBQ2QsR0FBRyxxQkFBcUI7WUFDeEIsR0FBRyxhQUFhO1NBQ2pCLENBQUM7UUFDRixJQUFJLFNBQVMsR0FBRyxtQkFBbUIsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUMvQyxJQUFJLFNBQVMsR0FBRyxzQkFBc0IsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNsRCxJQUFJLFNBQVMsR0FBRyxrQkFBa0IsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUM5QyxJQUFJLFNBQVMsR0FBRyx1QkFBdUIsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNuRCxJQUFJLFNBQVMsR0FBRyxzQkFBc0IsQ0FBQyxTQUFTLENBQUMsQ0FBQztRQUNsRCxLQUFLLENBQUMsU0FBUyxDQUFDLENBQUM7UUFDakIsSUFBSSxDQUFDLE1BQU0sR0FBRyxTQUFTLENBQUM7UUFDeEIsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO1FBQ3RELElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHNCQUFzQixDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO1FBQzlELElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLG1CQUFtQixDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDO1FBQzNELElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLGVBQWUsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztRQUN2RCxJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxrQkFBa0IsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztJQUM1RCxDQUFDO0lBT0QsT0FBTztRQUNMLEtBQUssQ0FBQyxPQUFPLEVBQUUsQ0FBQztJQUNsQixDQUFDO0NBQ0YifQ==