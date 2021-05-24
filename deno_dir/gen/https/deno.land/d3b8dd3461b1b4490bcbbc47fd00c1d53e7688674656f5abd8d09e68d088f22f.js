import { GetObjectRetentionOutput, GetObjectRetentionRequest } from "../models/models_0.ts";
import { deserializeAws_restXmlGetObjectRetentionCommand, serializeAws_restXmlGetObjectRetentionCommand, } from "../protocols/Aws_restXml.ts";
import { getBucketEndpointPlugin } from "../../middleware-bucket-endpoint/mod.ts";
import { getSerdePlugin } from "../../middleware-serde/mod.ts";
import { Command as $Command } from "../../smithy-client/mod.ts";
export class GetObjectRetentionCommand extends $Command {
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
        const commandName = "GetObjectRetentionCommand";
        const handlerExecutionContext = {
            logger,
            clientName,
            commandName,
            inputFilterSensitiveLog: GetObjectRetentionRequest.filterSensitiveLog,
            outputFilterSensitiveLog: GetObjectRetentionOutput.filterSensitiveLog,
        };
        const { requestHandler } = configuration;
        return stack.resolve((request) => requestHandler.handle(request.request, options || {}), handlerExecutionContext);
    }
    serialize(input, context) {
        return serializeAws_restXmlGetObjectRetentionCommand(input, context);
    }
    deserialize(output, context) {
        return deserializeAws_restXmlGetObjectRetentionCommand(output, context);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiR2V0T2JqZWN0UmV0ZW50aW9uQ29tbWFuZC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIkdldE9iamVjdFJldGVudGlvbkNvbW1hbmQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLHdCQUF3QixFQUFFLHlCQUF5QixFQUFFLE1BQU0sdUJBQXVCLENBQUM7QUFDNUYsT0FBTyxFQUNMLCtDQUErQyxFQUMvQyw2Q0FBNkMsR0FDOUMsTUFBTSw2QkFBNkIsQ0FBQztBQUNyQyxPQUFPLEVBQUUsdUJBQXVCLEVBQUUsTUFBTSx5Q0FBeUMsQ0FBQztBQUNsRixPQUFPLEVBQUUsY0FBYyxFQUFFLE1BQU0sK0JBQStCLENBQUM7QUFFL0QsT0FBTyxFQUFFLE9BQU8sSUFBSSxRQUFRLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQWdDakUsTUFBTSxPQUFPLHlCQUEwQixTQUFRLFFBSTlDO0lBSXNCO0lBQXJCLFlBQXFCLEtBQXFDO1FBRXhELEtBQUssRUFBRSxDQUFDO1FBRlcsVUFBSyxHQUFMLEtBQUssQ0FBZ0M7SUFJMUQsQ0FBQztJQUtELGlCQUFpQixDQUNmLFdBQW1FLEVBQ25FLGFBQXFDLEVBQ3JDLE9BQThCO1FBRTlCLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxDQUFDLGNBQWMsQ0FBQyxhQUFhLEVBQUUsSUFBSSxDQUFDLFNBQVMsRUFBRSxJQUFJLENBQUMsV0FBVyxDQUFDLENBQUMsQ0FBQztRQUMxRixJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsQ0FBQyx1QkFBdUIsQ0FBQyxhQUFhLENBQUMsQ0FBQyxDQUFDO1FBRWpFLE1BQU0sS0FBSyxHQUFHLFdBQVcsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLGVBQWUsQ0FBQyxDQUFDO1FBRXZELE1BQU0sRUFBRSxNQUFNLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDakMsTUFBTSxVQUFVLEdBQUcsVUFBVSxDQUFDO1FBQzlCLE1BQU0sV0FBVyxHQUFHLDJCQUEyQixDQUFDO1FBQ2hELE1BQU0sdUJBQXVCLEdBQTRCO1lBQ3ZELE1BQU07WUFDTixVQUFVO1lBQ1YsV0FBVztZQUNYLHVCQUF1QixFQUFFLHlCQUF5QixDQUFDLGtCQUFrQjtZQUNyRSx3QkFBd0IsRUFBRSx3QkFBd0IsQ0FBQyxrQkFBa0I7U0FDdEUsQ0FBQztRQUNGLE1BQU0sRUFBRSxjQUFjLEVBQUUsR0FBRyxhQUFhLENBQUM7UUFDekMsT0FBTyxLQUFLLENBQUMsT0FBTyxDQUNsQixDQUFDLE9BQXNDLEVBQUUsRUFBRSxDQUN6QyxjQUFjLENBQUMsTUFBTSxDQUFDLE9BQU8sQ0FBQyxPQUF3QixFQUFFLE9BQU8sSUFBSSxFQUFFLENBQUMsRUFDeEUsdUJBQXVCLENBQ3hCLENBQUM7SUFDSixDQUFDO0lBRU8sU0FBUyxDQUFDLEtBQXFDLEVBQUUsT0FBdUI7UUFDOUUsT0FBTyw2Q0FBNkMsQ0FBQyxLQUFLLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDdkUsQ0FBQztJQUVPLFdBQVcsQ0FBQyxNQUFzQixFQUFFLE9BQXVCO1FBQ2pFLE9BQU8sK0NBQStDLENBQUMsTUFBTSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQzFFLENBQUM7Q0FJRiJ9