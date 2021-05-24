import { DeleteBucketEncryptionRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlDeleteBucketEncryptionCommand, serializeAws_restXmlDeleteBucketEncryptionCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class DeleteBucketEncryptionCommand extends $Command {
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
        const commandName = "DeleteBucketEncryptionCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: DeleteBucketEncryptionRequest.filterSensitiveLog,
            outputFilterSensitiveLog: (output) => output,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlDeleteBucketEncryptionCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlDeleteBucketEncryptionCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiRGVsZXRlQnVja2V0RW5jcnlwdGlvbkNvbW1hbmQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJEZWxldGVCdWNrZXRFbmNyeXB0aW9uQ29tbWFuZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsNkJBQTZCLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUN0RSxPQUFPLEVBQ0wsbURBQW1ELEVBQ25ELGlEQUFpRCxHQUNsRCxNQUFNLDZCQUE2QixDQUFDO0FBQ3JDLE9BQU8sRUFBRSx1QkFBdUIsRUFBRSxNQUFNLHlDQUF5QyxDQUFDO0FBQ2xGLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSwrQkFBK0IsQ0FBQztBQUUvRCxPQUFPLEVBQUUsT0FBTyxJQUFJLFFBQVEsRUFBRSxNQUFNLDRCQUE0QixDQUFDO0FBc0RqRSxNQUFNLE9BQU8sNkJBQThCLFNBQVEsUUFJbEQ7SUFJc0I7SUFBckIsWUFBcUIsS0FBeUM7UUFFNUQsS0FBSyxFQUFFLENBQUM7UUFGVyxVQUFLLEdBQUwsS0FBSyxDQUFvQztJQUk5RCxDQUFDO0lBS0QsaUJBQWlCLENBQ2YsV0FBbUUsRUFDbkUsYUFBcUMsRUFDckMsT0FBOEI7UUFFOUIsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsY0FBYyxDQUFDLGFBQWEsRUFBRSxJQUFJLENBQUMsU0FBUyxFQUFFLElBQUksQ0FBQyxXQUFXLENBQUMsQ0FBQyxDQUFDO1FBQzFGLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLHVCQUF1QixDQUFDLGFBQWEsQ0FBQyxDQUFDLENBQUM7UUFFakUsTUFBTSxLQUFLLEdBQUcsV0FBVyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsZUFBZSxDQUFDLENBQUM7UUFFdkQsTUFBTSxFQUFFLE1BQU0sRUFBRSxHQUFHLGFBQWEsQ0FBQztRQUNqQyxNQUFNLFVBQVUsR0FBRyxVQUFVLENBQUM7UUFDOUIsTUFBTSxXQUFXLEdBQUcsK0JBQStCLENBQUM7UUFDcEQsTUFBTSx1QkFBdUIsR0FBNEI7WUFDdkQsTUFBTTtZQUNOLFVBQVU7WUFDVixXQUFXO1lBQ1gsdUJBQXVCLEVBQUUsNkJBQTZCLENBQUMsa0JBQWtCO1lBQ3pFLHdCQUF3QixFQUFFLENBQUMsTUFBVyxFQUFFLEVBQUUsQ0FBQyxNQUFNO1NBQ2xELENBQUM7UUFDRixNQUFNLEVBQUUsY0FBYyxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ3pDLE9BQU8sS0FBSyxDQUFDLE9BQU8sQ0FDbEIsQ0FBQyxPQUFzQyxFQUFFLEVBQUUsQ0FDekMsY0FBYyxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsT0FBd0IsRUFBRSxPQUFPLElBQUksRUFBRSxDQUFDLEVBQ3hFLHVCQUF1QixDQUN4QixDQUFDO0lBQ0osQ0FBQztJQUVPLFNBQVMsQ0FBQyxLQUF5QyxFQUFFLE9BQXVCO1FBQ2xGLE9BQU8saURBQWlELENBQUMsS0FBSyxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQzNFLENBQUM7SUFFTyxXQUFXLENBQUMsTUFBc0IsRUFBRSxPQUF1QjtRQUNqRSxPQUFPLG1EQUFtRCxDQUFDLE1BQU0sRUFBRSxPQUFPLENBQUMsQ0FBQztJQUM5RSxDQUFDO0NBSUYifQ==