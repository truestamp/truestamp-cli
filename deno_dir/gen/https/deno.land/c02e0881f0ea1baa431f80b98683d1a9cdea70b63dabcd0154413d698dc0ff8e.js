import { PutBucketAccelerateConfigurationRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlPutBucketAccelerateConfigurationCommand, serializeAws_restXmlPutBucketAccelerateConfigurationCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class PutBucketAccelerateConfigurationCommand extends $Command {
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
        const commandName = "PutBucketAccelerateConfigurationCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: PutBucketAccelerateConfigurationRequest.filterSensitiveLog,
            outputFilterSensitiveLog: (output) => output,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlPutBucketAccelerateConfigurationCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlPutBucketAccelerateConfigurationCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiUHV0QnVja2V0QWNjZWxlcmF0ZUNvbmZpZ3VyYXRpb25Db21tYW5kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiUHV0QnVja2V0QWNjZWxlcmF0ZUNvbmZpZ3VyYXRpb25Db21tYW5kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSx1Q0FBdUMsRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBQ2hGLE9BQU8sRUFDTCw2REFBNkQsRUFDN0QsMkRBQTJELEdBQzVELE1BQU0sNkJBQTZCLENBQUM7QUFDckMsT0FBTyxFQUFFLHVCQUF1QixFQUFFLE1BQU0seUNBQXlDLENBQUM7QUFDbEYsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLCtCQUErQixDQUFDO0FBRS9ELE9BQU8sRUFBRSxPQUFPLElBQUksUUFBUSxFQUFFLE1BQU0sNEJBQTRCLENBQUM7QUE0RWpFLE1BQU0sT0FBTyx1Q0FBd0MsU0FBUSxRQUk1RDtJQUlzQjtJQUFyQixZQUFxQixLQUFtRDtRQUV0RSxLQUFLLEVBQUUsQ0FBQztRQUZXLFVBQUssR0FBTCxLQUFLLENBQThDO0lBSXhFLENBQUM7SUFLRCxpQkFBaUIsQ0FDZixXQUFtRSxFQUNuRSxhQUFxQyxFQUNyQyxPQUE4QjtRQUU5QixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxjQUFjLENBQUMsYUFBYSxFQUFFLElBQUksQ0FBQyxTQUFTLEVBQUUsSUFBSSxDQUFDLFdBQVcsQ0FBQyxDQUFDLENBQUM7UUFDMUYsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsdUJBQXVCLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQztRQUVqRSxNQUFNLEtBQUssR0FBRyxXQUFXLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxlQUFlLENBQUMsQ0FBQztRQUV2RCxNQUFNLEVBQUUsTUFBTSxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ2pDLE1BQU0sVUFBVSxHQUFHLFVBQVUsQ0FBQztRQUM5QixNQUFNLFdBQVcsR0FBRyx5Q0FBeUMsQ0FBQztRQUM5RCxNQUFNLHVCQUF1QixHQUE0QjtZQUN2RCxNQUFNO1lBQ04sVUFBVTtZQUNWLFdBQVc7WUFDWCx1QkFBdUIsRUFBRSx1Q0FBdUMsQ0FBQyxrQkFBa0I7WUFDbkYsd0JBQXdCLEVBQUUsQ0FBQyxNQUFXLEVBQUUsRUFBRSxDQUFDLE1BQU07U0FDbEQsQ0FBQztRQUNGLE1BQU0sRUFBRSxjQUFjLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDekMsT0FBTyxLQUFLLENBQUMsT0FBTyxDQUNsQixDQUFDLE9BQXNDLEVBQUUsRUFBRSxDQUN6QyxjQUFjLENBQUMsTUFBTSxDQUFDLE9BQU8sQ0FBQyxPQUF3QixFQUFFLE9BQU8sSUFBSSxFQUFFLENBQUMsRUFDeEUsdUJBQXVCLENBQ3hCLENBQUM7SUFDSixDQUFDO0lBRU8sU0FBUyxDQUNmLEtBQW1ELEVBQ25ELE9BQXVCO1FBRXZCLE9BQU8sMkRBQTJELENBQUMsS0FBSyxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3JGLENBQUM7SUFFTyxXQUFXLENBQ2pCLE1BQXNCLEVBQ3RCLE9BQXVCO1FBRXZCLE9BQU8sNkRBQTZELENBQUMsTUFBTSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3hGLENBQUM7Q0FJRiJ9