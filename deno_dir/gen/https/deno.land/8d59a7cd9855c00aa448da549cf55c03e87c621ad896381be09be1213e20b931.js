import { GetBucketAccelerateConfigurationOutput, GetBucketAccelerateConfigurationRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlGetBucketAccelerateConfigurationCommand, serializeAws_restXmlGetBucketAccelerateConfigurationCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class GetBucketAccelerateConfigurationCommand extends $Command {
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
        const commandName = "GetBucketAccelerateConfigurationCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: GetBucketAccelerateConfigurationRequest.filterSensitiveLog,
            outputFilterSensitiveLog: GetBucketAccelerateConfigurationOutput.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlGetBucketAccelerateConfigurationCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlGetBucketAccelerateConfigurationCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiR2V0QnVja2V0QWNjZWxlcmF0ZUNvbmZpZ3VyYXRpb25Db21tYW5kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiR2V0QnVja2V0QWNjZWxlcmF0ZUNvbmZpZ3VyYXRpb25Db21tYW5kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSxzQ0FBc0MsRUFBRSx1Q0FBdUMsRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBQ3hILE9BQU8sRUFDTCw2REFBNkQsRUFDN0QsMkRBQTJELEdBQzVELE1BQU0sNkJBQTZCLENBQUM7QUFDckMsT0FBTyxFQUFFLHVCQUF1QixFQUFFLE1BQU0seUNBQXlDLENBQUM7QUFDbEYsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLCtCQUErQixDQUFDO0FBRS9ELE9BQU8sRUFBRSxPQUFPLElBQUksUUFBUSxFQUFFLE1BQU0sNEJBQTRCLENBQUM7QUEyRGpFLE1BQU0sT0FBTyx1Q0FBd0MsU0FBUSxRQUk1RDtJQUlzQjtJQUFyQixZQUFxQixLQUFtRDtRQUV0RSxLQUFLLEVBQUUsQ0FBQztRQUZXLFVBQUssR0FBTCxLQUFLLENBQThDO0lBSXhFLENBQUM7SUFLRCxpQkFBaUIsQ0FDZixXQUFtRSxFQUNuRSxhQUFxQyxFQUNyQyxPQUE4QjtRQUU5QixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxjQUFjLENBQUMsYUFBYSxFQUFFLElBQUksQ0FBQyxTQUFTLEVBQUUsSUFBSSxDQUFDLFdBQVcsQ0FBQyxDQUFDLENBQUM7UUFDMUYsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsdUJBQXVCLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQztRQUVqRSxNQUFNLEtBQUssR0FBRyxXQUFXLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxlQUFlLENBQUMsQ0FBQztRQUV2RCxNQUFNLEVBQUUsTUFBTSxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ2pDLE1BQU0sVUFBVSxHQUFHLFVBQVUsQ0FBQztRQUM5QixNQUFNLFdBQVcsR0FBRyx5Q0FBeUMsQ0FBQztRQUM5RCxNQUFNLHVCQUF1QixHQUE0QjtZQUN2RCxNQUFNO1lBQ04sVUFBVTtZQUNWLFdBQVc7WUFDWCx1QkFBdUIsRUFBRSx1Q0FBdUMsQ0FBQyxrQkFBa0I7WUFDbkYsd0JBQXdCLEVBQUUsc0NBQXNDLENBQUMsa0JBQWtCO1NBQ3BGLENBQUM7UUFDRixNQUFNLEVBQUUsY0FBYyxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ3pDLE9BQU8sS0FBSyxDQUFDLE9BQU8sQ0FDbEIsQ0FBQyxPQUFzQyxFQUFFLEVBQUUsQ0FDekMsY0FBYyxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsT0FBd0IsRUFBRSxPQUFPLElBQUksRUFBRSxDQUFDLEVBQ3hFLHVCQUF1QixDQUN4QixDQUFDO0lBQ0osQ0FBQztJQUVPLFNBQVMsQ0FDZixLQUFtRCxFQUNuRCxPQUF1QjtRQUV2QixPQUFPLDJEQUEyRCxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUMsQ0FBQztJQUNyRixDQUFDO0lBRU8sV0FBVyxDQUNqQixNQUFzQixFQUN0QixPQUF1QjtRQUV2QixPQUFPLDZEQUE2RCxDQUFDLE1BQU0sRUFBRSxPQUFPLENBQUMsQ0FBQztJQUN4RixDQUFDO0NBSUYifQ==