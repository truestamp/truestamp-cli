import { DeleteBucketCorsRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlDeleteBucketCorsCommand, serializeAws_restXmlDeleteBucketCorsCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class DeleteBucketCorsCommand extends $Command {
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
        const commandName = "DeleteBucketCorsCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: DeleteBucketCorsRequest.filterSensitiveLog,
            outputFilterSensitiveLog: (output) => output,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlDeleteBucketCorsCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlDeleteBucketCorsCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiRGVsZXRlQnVja2V0Q29yc0NvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJEZWxldGVCdWNrZXRDb3JzQ29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsdUJBQXVCLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUNoRSxPQUFPLEVBQ0wsNkNBQTZDLEVBQzdDLDJDQUEyQyxHQUM1QyxNQUFNLDZCQUE2QixDQUFDO0FBQ3JDLE9BQU8sRUFBRSx1QkFBdUIsRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ2xGLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUUvRCxPQUFPLEVBQUUsT0FBTyxJQUFJLFFBQVEsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBb0RqRSxNQUFNLE9BQU8sdUJBQXdCLFNBQVEsUUFJNUM7SUFJc0I7SUFBckIsWUFBcUIsS0FBbUM7UUFFdEQsS0FBSyxFQUFFLENBQUM7UUFGVyxVQUFLLEdBQUwsS0FBSyxDQUE4QjtJQUl4RCxDQUFDO0lBS0QsaUJBQWlCLENBQ2YsV0FBbUUsRUFDbkUsYUFBcUMsRUFDckMsT0FBOEI7UUFFOUIsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLGFBQWEsRUFBRSxJQUFJLENBQUMsU0FBUyxFQUFFLElBQUksQ0FBQyxXQUFXLENBQUMsQ0FBQyxDQUFDO1FBQzFGLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHVCQUF1QixDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUM7UUFFakUsTUFBTSxLQUFLLEdBQUcsV0FBVyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsZUFBZSxDQUFDLENBQUM7UUFFdkQsTUFBTSxFQUFFLE1BQU0sRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUNqQyxNQUFNLFVBQVUsR0FBRyxVQUFVLENBQUM7UUFDOUIsTUFBTSxXQUFXLEdBQUcseUJBQXlCLENBQUM7UUFDOUMsTUFBTSx1QkFBdUIsR0FBNEI7WUFDdkQsTUFBTTtZQUNOLFVBQVU7WUFDVixXQUFXO1lBQ1gsdUJBQXVCLEVBQUUsdUJBQXVCLENBQUMsa0JBQWtCO1lBQ25FLHdCQUF3QixFQUFFLENBQUMsTUFBVyxFQUFFLEVBQUUsQ0FBQyxNQUFNO1NBQ2xELENBQUM7UUFDRixNQUFNLEVBQUUsY0FBYyxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ3pDLE9BQU8sS0FBSyxDQUFDLE9BQU8sQ0FDbEIsQ0FBQyxPQUFzQyxFQUFFLEVBQUUsQ0FDekMsY0FBYyxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsT0FBd0IsRUFBRSxPQUFPLElBQUksRUFBRSxDQUFDLEVBQ3hFLHVCQUF1QixDQUN4QixDQUFDO0lBQ0osQ0FBQztJQUVPLFNBQVMsQ0FBQyxLQUFtQyxFQUFFLE9BQXVCO1FBQzVFLE9BQU8sMkNBQTJDLENBQUMsS0FBSyxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3JFLENBQUM7SUFFTyxXQUFXLENBQUMsTUFBc0IsRUFBRSxPQUF1QjtRQUNqRSxPQUFPLDZDQUE2QyxDQUFDLE1BQU0sRUFBRSxPQUFPLENBQUMsQ0FBQztJQUN4RSxDQUFDO0NBSUYifQ==