import { GetBucketIntelligentTieringConfigurationOutput, GetBucketIntelligentTieringConfigurationRequest, } from "../models/models_0.ts";
import { deserializeAws_restXmlGetBucketIntelligentTieringConfigurationCommand, serializeAws_restXmlGetBucketIntelligentTieringConfigurationCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class GetBucketIntelligentTieringConfigurationCommand extends $Command {
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
        const commandName = "GetBucketIntelligentTieringConfigurationCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: GetBucketIntelligentTieringConfigurationRequest.filterSensitiveLog,
            outputFilterSensitiveLog: GetBucketIntelligentTieringConfigurationOutput.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlGetBucketIntelligentTieringConfigurationCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlGetBucketIntelligentTieringConfigurationCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiR2V0QnVja2V0SW50ZWxsaWdlbnRUaWVyaW5nQ29uZmlndXJhdGlvbkNvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJHZXRCdWNrZXRJbnRlbGxpZ2VudFRpZXJpbmdDb25maWd1cmF0aW9uQ29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQ0wsOENBQThDLEVBQzlDLCtDQUErQyxHQUNoRCxNQUFNLHVCQUF1QixDQUFDO0FBQy9CLE9BQU8sRUFDTCxxRUFBcUUsRUFDckUsbUVBQW1FLEdBQ3BFLE1BQU0sNkJBQTZCLENBQUM7QUFDckMsT0FBTyxFQUFFLHVCQUF1QixFQUFFLE1BQU0seUNBQXlDLENBQUM7QUFDbEYsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLCtCQUErQixDQUFDO0FBRS9ELE9BQU8sRUFBRSxPQUFPLElBQUksUUFBUSxFQUFFLE1BQU0sNEJBQTRCLENBQUM7QUF3RGpFLE1BQU0sT0FBTywrQ0FBZ0QsU0FBUSxRQUlwRTtJQUlzQjtJQUFyQixZQUFxQixLQUEyRDtRQUU5RSxLQUFLLEVBQUUsQ0FBQztRQUZXLFVBQUssR0FBTCxLQUFLLENBQXNEO0lBSWhGLENBQUM7SUFLRCxpQkFBaUIsQ0FDZixXQUFtRSxFQUNuRSxhQUFxQyxFQUNyQyxPQUE4QjtRQUs5QixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxjQUFjLENBQUMsYUFBYSxFQUFFLElBQUksQ0FBQyxTQUFTLEVBQUUsSUFBSSxDQUFDLFdBQVcsQ0FBQyxDQUFDLENBQUM7UUFDMUYsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsdUJBQXVCLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQztRQUVqRSxNQUFNLEtBQUssR0FBRyxXQUFXLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxlQUFlLENBQUMsQ0FBQztRQUV2RCxNQUFNLEVBQUUsTUFBTSxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ2pDLE1BQU0sVUFBVSxHQUFHLFVBQVUsQ0FBQztRQUM5QixNQUFNLFdBQVcsR0FBRyxpREFBaUQsQ0FBQztRQUN0RSxNQUFNLHVCQUF1QixHQUE0QjtZQUN2RCxNQUFNO1lBQ04sVUFBVTtZQUNWLFdBQVc7WUFDWCx1QkFBdUIsRUFBRSwrQ0FBK0MsQ0FBQyxrQkFBa0I7WUFDM0Ysd0JBQXdCLEVBQUUsOENBQThDLENBQUMsa0JBQWtCO1NBQzVGLENBQUM7UUFDRixNQUFNLEVBQUUsY0FBYyxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ3pDLE9BQU8sS0FBSyxDQUFDLE9BQU8sQ0FDbEIsQ0FBQyxPQUFzQyxFQUFFLEVBQUUsQ0FDekMsY0FBYyxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsT0FBd0IsRUFBRSxPQUFPLElBQUksRUFBRSxDQUFDLEVBQ3hFLHVCQUF1QixDQUN4QixDQUFDO0lBQ0osQ0FBQztJQUVPLFNBQVMsQ0FDZixLQUEyRCxFQUMzRCxPQUF1QjtRQUV2QixPQUFPLG1FQUFtRSxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUMsQ0FBQztJQUM3RixDQUFDO0lBRU8sV0FBVyxDQUNqQixNQUFzQixFQUN0QixPQUF1QjtRQUV2QixPQUFPLHFFQUFxRSxDQUFDLE1BQU0sRUFBRSxPQUFPLENBQUMsQ0FBQztJQUNoRyxDQUFDO0NBSUYifQ==