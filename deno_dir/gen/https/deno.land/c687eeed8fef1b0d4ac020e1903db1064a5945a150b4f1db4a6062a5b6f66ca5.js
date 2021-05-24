import { PutBucketLifecycleConfigurationRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlPutBucketLifecycleConfigurationCommand, serializeAws_restXmlPutBucketLifecycleConfigurationCommand, } from "../protocols/Aws_restXml.ts";
import { getApplyMd5BodyChecksumPlugin } from "../../middleware-apply-body-checksum/mod.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class PutBucketLifecycleConfigurationCommand extends $Command {
    input;
    constructor(input) {
        super();
        this.input = input;
    }
    resolveMiddleware(clientStack, configuration, options) {
        this.middlewareStack.use(getSerdePlugin(configuration, this.serialize, this.deserialize));
        this.middlewareStack.use(getBucketEndpointPlugin(configuration));
        this.middlewareStack.use(getApplyMd5BodyChecksumPlugin(configuration));
        const stack = clientStack.concat(this.middlewareStack);
        const { logger } = configuration;
        const clientName = "S3Client";
        const commandName = "PutBucketLifecycleConfigurationCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: PutBucketLifecycleConfigurationRequest.filterSensitiveLog,
            outputFilterSensitiveLog: (output) => output,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlPutBucketLifecycleConfigurationCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlPutBucketLifecycleConfigurationCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiUHV0QnVja2V0TGlmZWN5Y2xlQ29uZmlndXJhdGlvbkNvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJQdXRCdWNrZXRMaWZlY3ljbGVDb25maWd1cmF0aW9uQ29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsc0NBQXNDLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUMvRSxPQUFPLEVBQ0wsNERBQTRELEVBQzVELDBEQUEwRCxHQUMzRCxNQUFNLDZCQUE2QixDQUFDO0FBQ3JDLE9BQU8sRUFBRSw2QkFBNkIsRUFBRSxNQUFNLDZDQUE2QyxDQUFDO0FBQzVGLE9BQU8sRUFBRSx1QkFBdUIsRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ2xGLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUUvRCxPQUFPLEVBQUUsT0FBTyxJQUFJLFFBQVEsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBNEhqRSxNQUFNLE9BQU8sc0NBQXVDLFNBQVEsUUFJM0Q7SUFJc0I7SUFBckIsWUFBcUIsS0FBa0Q7UUFFckUsS0FBSyxFQUFFLENBQUM7UUFGVyxVQUFLLEdBQUwsS0FBSyxDQUE2QztJQUl2RSxDQUFDO0lBS0QsaUJBQWlCLENBQ2YsV0FBbUUsRUFDbkUsYUFBcUMsRUFDckMsT0FBOEI7UUFFOUIsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLGFBQWEsRUFBRSxJQUFJLENBQUMsU0FBUyxFQUFFLElBQUksQ0FBQyxXQUFXLENBQUMsQ0FBQyxDQUFDO1FBQzFGLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHVCQUF1QixDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUM7UUFDakUsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsNkJBQTZCLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQztRQUV2RSxNQUFNLEtBQUssR0FBRyxXQUFXLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxlQUFlLENBQUMsQ0FBQztRQUV2RCxNQUFNLEVBQUUsTUFBTSxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ2pDLE1BQU0sVUFBVSxHQUFHLFVBQVUsQ0FBQztRQUM5QixNQUFNLFdBQVcsR0FBRyx3Q0FBd0MsQ0FBQztRQUM3RCxNQUFNLHVCQUF1QixHQUE0QjtZQUN2RCxNQUFNO1lBQ04sVUFBVTtZQUNWLFdBQVc7WUFDWCx1QkFBdUIsRUFBRSxzQ0FBc0MsQ0FBQyxrQkFBa0I7WUFDbEYsd0JBQXdCLEVBQUUsQ0FBQyxNQUFXLEVBQUUsRUFBRSxDQUFDLE1BQU07U0FDbEQsQ0FBQztRQUNGLE1BQU0sRUFBRSxjQUFjLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDekMsT0FBTyxLQUFLLENBQUMsT0FBTyxDQUNsQixDQUFDLE9BQXNDLEVBQUUsRUFBRSxDQUN6QyxjQUFjLENBQUMsTUFBTSxDQUFDLE9BQU8sQ0FBQyxPQUF3QixFQUFFLE9BQU8sSUFBSSxFQUFFLENBQUMsRUFDeEUsdUJBQXVCLENBQ3hCLENBQUM7SUFDSixDQUFDO0lBRU8sU0FBUyxDQUNmLEtBQWtELEVBQ2xELE9BQXVCO1FBRXZCLE9BQU8sMERBQTBELENBQUMsS0FBSyxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3BGLENBQUM7SUFFTyxXQUFXLENBQ2pCLE1BQXNCLEVBQ3RCLE9BQXVCO1FBRXZCLE9BQU8sNERBQTRELENBQUMsTUFBTSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3ZGLENBQUM7Q0FJRiJ9