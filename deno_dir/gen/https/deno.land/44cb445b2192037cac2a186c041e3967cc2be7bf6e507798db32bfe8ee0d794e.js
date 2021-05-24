import { DeleteObjectTaggingOutput, DeleteObjectTaggingRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlDeleteObjectTaggingCommand, serializeAws_restXmlDeleteObjectTaggingCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class DeleteObjectTaggingCommand extends $Command {
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
        const commandName = "DeleteObjectTaggingCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: DeleteObjectTaggingRequest.filterSensitiveLog,
            outputFilterSensitiveLog: DeleteObjectTaggingOutput.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlDeleteObjectTaggingCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlDeleteObjectTaggingCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiRGVsZXRlT2JqZWN0VGFnZ2luZ0NvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJEZWxldGVPYmplY3RUYWdnaW5nQ29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUseUJBQXlCLEVBQUUsMEJBQTBCLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUM5RixPQUFPLEVBQ0wsZ0RBQWdELEVBQ2hELDhDQUE4QyxHQUMvQyxNQUFNLDZCQUE2QixDQUFDO0FBQ3JDLE9BQU8sRUFBRSx1QkFBdUIsRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ2xGLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUUvRCxPQUFPLEVBQUUsT0FBTyxJQUFJLFFBQVEsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBdURqRSxNQUFNLE9BQU8sMEJBQTJCLFNBQVEsUUFJL0M7SUFJc0I7SUFBckIsWUFBcUIsS0FBc0M7UUFFekQsS0FBSyxFQUFFLENBQUM7UUFGVyxVQUFLLEdBQUwsS0FBSyxDQUFpQztJQUkzRCxDQUFDO0lBS0QsaUJBQWlCLENBQ2YsV0FBbUUsRUFDbkUsYUFBcUMsRUFDckMsT0FBOEI7UUFFOUIsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLGFBQWEsRUFBRSxJQUFJLENBQUMsU0FBUyxFQUFFLElBQUksQ0FBQyxXQUFXLENBQUMsQ0FBQyxDQUFDO1FBQzFGLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHVCQUF1QixDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUM7UUFFakUsTUFBTSxLQUFLLEdBQUcsV0FBVyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsZUFBZSxDQUFDLENBQUM7UUFFdkQsTUFBTSxFQUFFLE1BQU0sRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUNqQyxNQUFNLFVBQVUsR0FBRyxVQUFVLENBQUM7UUFDOUIsTUFBTSxXQUFXLEdBQUcsNEJBQTRCLENBQUM7UUFDakQsTUFBTSx1QkFBdUIsR0FBNEI7WUFDdkQsTUFBTTtZQUNOLFVBQVU7WUFDVixXQUFXO1lBQ1gsdUJBQXVCLEVBQUUsMEJBQTBCLENBQUMsa0JBQWtCO1lBQ3RFLHdCQUF3QixFQUFFLHlCQUF5QixDQUFDLGtCQUFrQjtTQUN2RSxDQUFDO1FBQ0YsTUFBTSxFQUFFLGNBQWMsRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUN6QyxPQUFPLEtBQUssQ0FBQyxPQUFPLENBQ2xCLENBQUMsT0FBc0MsRUFBRSxFQUFFLENBQ3pDLGNBQWMsQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLE9BQXdCLEVBQUUsT0FBTyxJQUFJLEVBQUUsQ0FBQyxFQUN4RSx1QkFBdUIsQ0FDeEIsQ0FBQztJQUNKLENBQUM7SUFFTyxTQUFTLENBQUMsS0FBc0MsRUFBRSxPQUF1QjtRQUMvRSxPQUFPLDhDQUE4QyxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUMsQ0FBQztJQUN4RSxDQUFDO0lBRU8sV0FBVyxDQUFDLE1BQXNCLEVBQUUsT0FBdUI7UUFDakUsT0FBTyxnREFBZ0QsQ0FBQyxNQUFNLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDM0UsQ0FBQztDQUlGIn0=