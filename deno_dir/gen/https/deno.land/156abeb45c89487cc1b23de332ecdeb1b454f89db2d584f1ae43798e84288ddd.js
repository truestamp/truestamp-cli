import { PutBucketAnalyticsConfigurationRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlPutBucketAnalyticsConfigurationCommand, serializeAws_restXmlPutBucketAnalyticsConfigurationCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class PutBucketAnalyticsConfigurationCommand extends $Command {
    input;
    constructor(input) {
        super();
        this.input = input;
    }
    resolveMiddleware(clientStack, configuration, options) {
        this.middlewareStack.use(getSerdePlugin(configuration, this.serialize, this.deserialize));
        this.middlewareStack.use(getBucketEndpointPlugin(configuration));
        const stack = clientStack.concat(this.middlewareStack);
        const { logger } = configuration;
        const clientName = "S3Client";
        const commandName = "PutBucketAnalyticsConfigurationCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: PutBucketAnalyticsConfigurationRequest.filterSensitiveLog,
            outputFilterSensitiveLog: (output) => output,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlPutBucketAnalyticsConfigurationCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlPutBucketAnalyticsConfigurationCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiUHV0QnVja2V0QW5hbHl0aWNzQ29uZmlndXJhdGlvbkNvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJQdXRCdWNrZXRBbmFseXRpY3NDb25maWd1cmF0aW9uQ29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsc0NBQXNDLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUMvRSxPQUFPLEVBQ0wsNERBQTRELEVBQzVELDBEQUEwRCxHQUMzRCxNQUFNLDZCQUE2QixDQUFDO0FBQ3JDLE9BQU8sRUFBRSx1QkFBdUIsRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ2xGLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUUvRCxPQUFPLEVBQUUsT0FBTyxJQUFJLFFBQVEsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBa0pqRSxNQUFNLE9BQU8sc0NBQXVDLFNBQVEsUUFJM0Q7SUFJc0I7SUFBckIsWUFBcUIsS0FBa0Q7UUFFckUsS0FBSyxFQUFFLENBQUM7UUFGVyxVQUFLLEdBQUwsS0FBSyxDQUE2QztJQUl2RSxDQUFDO0lBS0QsaUJBQWlCLENBQ2YsV0FBbUUsRUFDbkUsYUFBcUMsRUFDckMsT0FBOEI7UUFFOUIsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLGFBQWEsRUFBRSxJQUFJLENBQUMsU0FBUyxFQUFFLElBQUksQ0FBQyxXQUFXLENBQUMsQ0FBQyxDQUFDO1FBQzFGLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHVCQUF1QixDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUM7UUFFakUsTUFBTSxLQUFLLEdBQUcsV0FBVyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsZUFBZSxDQUFDLENBQUM7UUFFdkQsTUFBTSxFQUFFLE1BQU0sRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUNqQyxNQUFNLFVBQVUsR0FBRyxVQUFVLENBQUM7UUFDOUIsTUFBTSxXQUFXLEdBQUcsd0NBQXdDLENBQUM7UUFDN0QsTUFBTSx1QkFBdUIsR0FBNEI7WUFDdkQsTUFBTTtZQUNOLFVBQVU7WUFDVixXQUFXO1lBQ1gsdUJBQXVCLEVBQUUsc0NBQXNDLENBQUMsa0JBQWtCO1lBQ2xGLHdCQUF3QixFQUFFLENBQUMsTUFBVyxFQUFFLEVBQUUsQ0FBQyxNQUFNO1NBQ2xELENBQUM7UUFDRixNQUFNLEVBQUUsY0FBYyxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ3pDLE9BQU8sS0FBSyxDQUFDLE9BQU8sQ0FDbEIsQ0FBQyxPQUFzQyxFQUFFLEVBQUUsQ0FDekMsY0FBYyxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsT0FBd0IsRUFBRSxPQUFPLElBQUksRUFBRSxDQUFDLEVBQ3hFLHVCQUF1QixDQUN4QixDQUFDO0lBQ0osQ0FBQztJQUVPLFNBQVMsQ0FDZixLQUFrRCxFQUNsRCxPQUF1QjtRQUV2QixPQUFPLDBEQUEwRCxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUMsQ0FBQztJQUNwRixDQUFDO0lBRU8sV0FBVyxDQUNqQixNQUFzQixFQUN0QixPQUF1QjtRQUV2QixPQUFPLDREQUE0RCxDQUFDLE1BQU0sRUFBRSxPQUFPLENBQUMsQ0FBQztJQUN2RixDQUFDO0NBSUYifQ==