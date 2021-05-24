import { GetBucketMetricsConfigurationOutput, GetBucketMetricsConfigurationRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlGetBucketMetricsConfigurationCommand, serializeAws_restXmlGetBucketMetricsConfigurationCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class GetBucketMetricsConfigurationCommand extends $Command {
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
        const commandName = "GetBucketMetricsConfigurationCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: GetBucketMetricsConfigurationRequest.filterSensitiveLog,
            outputFilterSensitiveLog: GetBucketMetricsConfigurationOutput.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlGetBucketMetricsConfigurationCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlGetBucketMetricsConfigurationCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiR2V0QnVja2V0TWV0cmljc0NvbmZpZ3VyYXRpb25Db21tYW5kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiR2V0QnVja2V0TWV0cmljc0NvbmZpZ3VyYXRpb25Db21tYW5kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSxtQ0FBbUMsRUFBRSxvQ0FBb0MsRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBQ2xILE9BQU8sRUFDTCwwREFBMEQsRUFDMUQsd0RBQXdELEdBQ3pELE1BQU0sNkJBQTZCLENBQUM7QUFDckMsT0FBTyxFQUFFLHVCQUF1QixFQUFFLE1BQU0seUNBQXlDLENBQUM7QUFDbEYsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLCtCQUErQixDQUFDO0FBRS9ELE9BQU8sRUFBRSxPQUFPLElBQUksUUFBUSxFQUFFLE1BQU0sNEJBQTRCLENBQUM7QUFxRWpFLE1BQU0sT0FBTyxvQ0FBcUMsU0FBUSxRQUl6RDtJQUlzQjtJQUFyQixZQUFxQixLQUFnRDtRQUVuRSxLQUFLLEVBQUUsQ0FBQztRQUZXLFVBQUssR0FBTCxLQUFLLENBQTJDO0lBSXJFLENBQUM7SUFLRCxpQkFBaUIsQ0FDZixXQUFtRSxFQUNuRSxhQUFxQyxFQUNyQyxPQUE4QjtRQUU5QixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyxjQUFjLENBQUMsYUFBYSxFQUFFLElBQUksQ0FBQyxTQUFTLEVBQUUsSUFBSSxDQUFDLFdBQVcsQ0FBQyxDQUFDLENBQUM7UUFDMUYsSUFBSSxDQUFDLGVBQWUsQ0FBQyxHQUFHLENBQUMsdUJBQXVCLENBQUMsYUFBYSxDQUFDLENBQUMsQ0FBQztRQUVqRSxNQUFNLEtBQUssR0FBRyxXQUFXLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxlQUFlLENBQUMsQ0FBQztRQUV2RCxNQUFNLEVBQUUsTUFBTSxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ2pDLE1BQU0sVUFBVSxHQUFHLFVBQVUsQ0FBQztRQUM5QixNQUFNLFdBQVcsR0FBRyxzQ0FBc0MsQ0FBQztRQUMzRCxNQUFNLHVCQUF1QixHQUE0QjtZQUN2RCxNQUFNO1lBQ04sVUFBVTtZQUNWLFdBQVc7WUFDWCx1QkFBdUIsRUFBRSxvQ0FBb0MsQ0FBQyxrQkFBa0I7WUFDaEYsd0JBQXdCLEVBQUUsbUNBQW1DLENBQUMsa0JBQWtCO1NBQ2pGLENBQUM7UUFDRixNQUFNLEVBQUUsY0FBYyxFQUFFLEdBQUcsYUFBYSxDQUFDO1FBQ3pDLE9BQU8sS0FBSyxDQUFDLE9BQU8sQ0FDbEIsQ0FBQyxPQUFzQyxFQUFFLEVBQUUsQ0FDekMsY0FBYyxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsT0FBd0IsRUFBRSxPQUFPLElBQUksRUFBRSxDQUFDLEVBQ3hFLHVCQUF1QixDQUN4QixDQUFDO0lBQ0osQ0FBQztJQUVPLFNBQVMsQ0FBQyxLQUFnRCxFQUFFLE9BQXVCO1FBQ3pGLE9BQU8sd0RBQXdELENBQUMsS0FBSyxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ2xGLENBQUM7SUFFTyxXQUFXLENBQ2pCLE1BQXNCLEVBQ3RCLE9BQXVCO1FBRXZCLE9BQU8sMERBQTBELENBQUMsTUFBTSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQ3JGLENBQUM7Q0FJRiJ9